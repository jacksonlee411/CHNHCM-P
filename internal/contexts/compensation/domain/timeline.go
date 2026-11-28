package domain

import (
	"slices"
	"strings"

	"chnhcm.local/chnhcm/internal/foundation"
)

// A record keeps its identity through a correction. The revision identifies
// the precise saved contents independently of changes to adjacent periods.
type Record[T any] struct {
	ID        string `json:"id"`
	Revision  string `json:"revision"`
	Start     string `json:"start"`
	Active    bool   `json:"active"`
	Operation string `json:"operation"`
	Reason    string `json:"reason"`
	Data      T      `json:"data"`
}

type Timeline[T any] []Record[T]

func (t Timeline[T]) At(day string) int {
	n, exact := slices.BinarySearchFunc(t, day, func(r Record[T], d string) int { return strings.Compare(r.Start, d) })
	if exact {
		return n
	}
	return n - 1
}

func (t Timeline[T]) Period(index int) foundation.Period {
	p := foundation.Period{From: t[index].Start}
	if index+1 < len(t) {
		p.Until = t[index+1].Start
	}
	return p
}

type Edit struct {
	Action, Date, Reason, RecordID, ExpectedRevision string
}

func Conflict() error {
	return foundation.Fail(409, "M09_VERSION_CONFLICT", "资料已变更，请重新读取后核对")
}

// Amend applies one action to the selected record or its following interval.
// The caller locks the aggregate and checks its source version before entering.
func Amend[T any](prior Timeline[T], edit Edit, apply func(T) (T, error)) (Timeline[T], Record[T], error) {
	var target Record[T]
	t := slices.Clone(prior)
	if edit.Action == "CREATE" {
		if len(t) > 0 {
			return nil, target, Conflict()
		}
		if !foundation.ValidDate(edit.Date) {
			return nil, target, Invalid("effectiveDate", "请输入有效的生效日期")
		}
		if strings.TrimSpace(edit.Reason) == "" {
			return nil, target, Invalid("reason", "请选择办理原因")
		}
		data, err := apply(target.Data)
		if err != nil {
			return nil, target, err
		}
		target = Record[T]{ID: foundation.NewID(), Revision: foundation.NewID(), Start: edit.Date, Active: true, Operation: "CREATE", Reason: edit.Reason, Data: data}
		return append(t, target), target, nil
	}
	i := slices.IndexFunc(t, func(r Record[T]) bool { return r.ID == edit.RecordID })
	if i < 0 || t[i].Revision != edit.ExpectedRevision {
		return nil, target, Conflict()
	}
	base := t[i]
	switch edit.Action {
	case "ADD", "DEACTIVATE", "REACTIVATE":
		if !foundation.ValidDate(edit.Date) || edit.Date <= base.Start || (i+1 < len(t) && edit.Date >= t[i+1].Start) {
			return nil, target, Invalid("effectiveDate", "日期须在所选记录之后、下一条记录之前")
		}
		if strings.TrimSpace(edit.Reason) == "" {
			return nil, target, Invalid("reason", "请选择办理原因")
		}
		if (edit.Action == "REACTIVATE") == base.Active {
			return nil, target, Invalid("action", "所选记录状态不能办理此操作")
		}
		target = base
		target.ID, target.Revision = foundation.NewID(), foundation.NewID()
		target.Start, target.Reason, target.Operation = edit.Date, edit.Reason, edit.Action
		target.Active = edit.Action != "DEACTIVATE"
		if target.Active {
			data, err := apply(base.Data)
			if err != nil {
				return nil, target, err
			}
			target.Data = data
		}
		t = slices.Insert(t, i+1, target)
	case "CORRECT":
		target = base
		if edit.Date != "" {
			if !foundation.ValidDate(edit.Date) || i > 0 && edit.Date <= t[i-1].Start || i+1 < len(t) && edit.Date >= t[i+1].Start {
				return nil, target, Invalid("effectiveDate", "更正后的日期须留在相邻记录之间")
			}
			target.Start = edit.Date
		}
		if edit.Reason != "" {
			target.Reason = edit.Reason
		}
		if base.Active {
			data, err := apply(base.Data)
			if err != nil {
				return nil, target, err
			}
			target.Data = data
		}
		target.Revision = foundation.NewID()
		t[i] = target
	case "DELETE":
		if i == 0 {
			return nil, target, Invalid("recordId", "保留首条记录，可更正或按日停用")
		}
		target = base
		t = slices.Delete(t, i, i+1)
	default:
		return nil, target, Invalid("action", "所选操作不可用")
	}
	for pos := 1; pos < len(t); pos++ {
		before, after := t[pos-1], t[pos]
		if !before.Active && after.Operation != "REACTIVATE" || before.Active && after.Operation == "REACTIVATE" {
			return nil, target, Invalid("effectiveDate", "本次变更会使后续记录的启停顺序无效")
		}
	}
	return t, target, nil
}
