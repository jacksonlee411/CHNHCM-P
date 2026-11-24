package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"
)

type StaffingQuota struct {
	PositionSource string        `json:"positionSource,omitempty"`
	ID             string        `json:"quotaId"`
	PositionID     string        `json:"positionId,omitempty"`
	DepartmentID   string        `json:"departmentId,omitempty"`
	Source         string        `json:"sourceVersion"`
	Number         int64         `json:"sourceNumber"`
	Records        []QuotaRecord `json:"records"`
}

type QuotaTarget struct{ Kind, ID string }

func (q StaffingQuota) Target() QuotaTarget {
	if q.DepartmentID != "" {
		return QuotaTarget{"department", q.DepartmentID}
	}
	return QuotaTarget{"position", q.PositionID}
}

type QuotaRecord struct {
	ID       string `json:"recordId"`
	Revision string `json:"revision"`
	Date     string `json:"effectiveDate"`
	Amount   *int   `json:"quota"`
	Reason   string `json:"reason"`
	Audit    Audit  `json:"audit"`
}

type QuotaChange struct {
	ReasonProvided                                   bool
	Action, Source, RecordID, Revision, Date, Reason string
	Amount                                           *int
}
type QuotaPlan struct {
	Before, After StaffingQuota
	Selected      string
	Changed       bool
}

func (q StaffingQuota) At(day string) int {
	i, found := slices.BinarySearchFunc(q.Records, day, func(r QuotaRecord, day string) int { return cmp.Compare(r.Date, day) })
	if found {
		return i
	}
	return i - 1
}
func sameQuota(a, b *int) bool { return a == b || a != nil && b != nil && *a == *b }
func (q StaffingQuota) BoundaryChangesAmount(index int) bool {
	var prior *int
	if index > 0 {
		prior = q.Records[index-1].Amount
	}
	next := q.Records[index].Amount
	if prior == nil || next == nil {
		return prior != next
	}
	return *prior != *next
}
func ChangeQuota(before StaffingQuota, c QuotaChange, pos QuotaPeriods) (QuotaPlan, error) {
	p := QuotaPlan{Before: before, After: before}
	invalid := func(msg string) (QuotaPlan, error) { return p, foundation.Fail(400, "M03_INVALID_INPUT", msg) }
	conflict := func(msg string) (QuotaPlan, error) { return p, foundation.Fail(409, "M03_QUOTA_CONFLICT", msg) }
	dateConflict := func(message string) (QuotaPlan, error) {
		return p, &foundation.Fault{Status: 409, Code: "M03_QUOTA_CONFLICT", Field: "effectiveDate", Message: message}
	}
	if pos.ID == "" || len(pos.Records) == 0 {
		return p, foundation.Fail(404, "M03_NOT_FOUND", "未找到岗位")
	}
	if c.Action == "create" {
		if before.ID != "" {
			if pos.Kind == Kind("department") {
				return conflict("部门已有总额，请刷新后调整")
			}
			return conflict("岗位已有核编，请刷新后调编")
		}
	} else if before.ID == "" || before.Source != c.Source {
		return conflict("额度已变化，请刷新后重新办理")
	}
	if c.Amount != nil && *c.Amount < 0 {
		return p, fieldInvalid("quota", "核定编制须为非负整数")
	}
	reason := strings.TrimSpace(c.Reason)
	if utf8.RuneCountInString(reason) > 500 {
		return p, fieldInvalid("reason", "原因最多 500 字")
	}
	records := slices.Clone(before.Records)
	index := slices.IndexFunc(records, func(r QuotaRecord) bool { return r.ID == c.RecordID })
	record := QuotaRecord{}
	switch c.Action {
	case "create", "adjust", "withdraw":
		if reason == "" || (c.Amount == nil) != (c.Action == "withdraw") {
			return invalid("请填写调整原因及相应的核定编制")
		}
		record = QuotaRecord{ID: foundation.NewID(), Date: c.Date, Amount: c.Amount, Reason: reason}
	case "correct", "allocate", "redate", "delete":
		if index < 0 || records[index].Revision != c.Revision {
			return conflict("所选记录已变化，请刷新")
		}
		record = records[index]
		switch c.Action {
		case "allocate":
			if reason == "" {
				return invalid("请填写调整原因")
			}
			if sameQuota(record.Amount, c.Amount) && record.Reason == reason {
				p.Selected = record.ID
				return p, nil
			}
			record.Amount, record.Reason = c.Amount, reason
		case "correct":
			if c.Amount != nil {
				if record.Amount == nil {
					return invalid("撤去记录不可更正为核定")
				}
				record.Amount = c.Amount
			}
			if c.ReasonProvided || c.Reason != "" {
				if reason == "" {
					return p, fieldInvalid("reason", "请填写原因")
				}
				record.Reason = reason
			}
			if (c.Amount == nil || *c.Amount == *records[index].Amount) && record.Reason == records[index].Reason {
				p.Selected = record.ID
				return p, nil
			}
		case "redate":
			if c.Date == record.Date {
				p.Selected = record.ID
				return p, nil
			}
			if index > 0 && c.Date <= records[index-1].Date || index+1 < len(records) && c.Date >= records[index+1].Date {
				return dateConflict("改日须保持原记录次序")
			}
			record.Date = c.Date
		case "delete":
			if index == 0 {
				return conflict("首条核编不能删除")
			}
		}
	default:
		return invalid("核编动作无效")
	}
	if c.Action != "delete" {
		if !foundation.ValidDate(record.Date) {
			return p, fieldInvalid("effectiveDate", "请输入有效的生效日期")
		}
		if c.Action != "correct" {
			i := pos.At(record.Date)
			if i < 0 {
				return dateConflict("生效日不能早于对象首日 " + pos.Records[0].Date)
			}
			if !pos.Records[i].Active {
				return dateConflict("对象在生效日已停用")
			}
		}
		for _, r := range records {
			if r.Date == record.Date && r.ID != record.ID {
				return dateConflict("该日已有独立记录")
			}
		}
		if c.Action == "withdraw" {
			i := before.At(record.Date)
			if i < 0 || before.Records[i].Amount == nil {
				return dateConflict("该日尚未核编")
			}
		}
		record.Revision = foundation.NewID()
		if index < 0 {
			records = append(records, record)
		} else {
			records[index] = record
		}
		slices.SortFunc(records, func(a, b QuotaRecord) int { return cmp.Compare(a.Date, b.Date) })
	} else {
		records = slices.Delete(records, index, index+1)
	}
	p.Changed = true
	p.Selected = record.ID
	p.After.Records = records
	if before.ID == "" {
		p.After.ID = foundation.NewID()
		if pos.Kind == Kind("department") {
			p.After.DepartmentID = pos.ID
		} else {
			p.After.PositionID = pos.ID
		}
	}
	if pos.Kind == Position {
		p.After.PositionSource = pos.Source
	}
	p.After.Source = foundation.NewID()
	p.After.Number++
	return p, nil
}
