package hireprep

import (
	"slices"
	"time"

	f "chnhcm.local/chnhcm/internal/foundation"
)

type ManagerCommand struct {
	BusinessDate string `json:"businessDate"`
	Expected     string `json:"expectedSourceVersion"`
	Purpose      string `json:"purpose"`
	Manager      string `json:"manager,omitempty"`
	LastDay      string `json:"lastDay,omitempty"`
}
type ManagerFact struct {
	RecordNumber int `json:"recordNumber"`
	f.Period
	Manager string `json:"manager,omitempty"`
	Name    string `json:"name,omitempty"`
	Purpose string `json:"purpose"`
}
type ExitCommand struct {
	Date     string `json:"date"`
	Meaning  string `json:"meaning"`
	Expected string `json:"expectedSourceVersion"`
	Notice   string `json:"notice"`
}
type ExitEvidence struct {
	Date          string `json:"date"`
	Meaning       string `json:"meaning"`
	FirstInactive string `json:"firstInactive"`
}

func (in ExitCommand) Boundary() (ExitEvidence, error) {
	ev := ExitEvidence{Date: in.Date, Meaning: in.Meaning, FirstInactive: in.Date}
	if !f.ValidDate(in.Date) || !slices.Contains([]string{"last-active", "first-inactive"}, in.Meaning) {
		return ev, f.Fail(400, "EXIT_DATE", "请核对原通知日期及其含义")
	}
	if in.Meaning == "last-active" {
		d, _ := time.Parse(time.DateOnly, in.Date)
		ev.FirstInactive = d.AddDate(0, 0, 1).Format(time.DateOnly)
	}
	if !f.ValidDate(ev.FirstInactive) {
		return ev, f.Fail(400, "EXIT_DATE", "任期边界无效")
	}
	return ev, nil
}

type ManagerEdge struct{ Child, Parent, From, Until string }

// Effective agreement starts are actual dated decisions. A later appointment
// replaces the earlier one for its purpose, even when the replacement expires.
func ValidateManagers(edges []ManagerEdge) error {
	dates := map[string]bool{}
	for _, edge := range edges {
		dates[edge.From] = true
		if edge.Until != "" {
			dates[edge.Until] = true
		}
	}
	for date := range dates {
		latest := map[string]ManagerEdge{}
		for _, edge := range edges {
			if edge.From > date {
				continue
			}
			old, ok := latest[edge.Child]
			if ok && old.From == edge.From {
				return f.Fail(409, "MANAGER_CONFLICT", "同用途、同日已有汇报安排")
			}
			if !ok || old.From < edge.From {
				latest[edge.Child] = edge
			}
		}
		done := map[string]bool{}
		for start := range latest {
			path := map[string]bool{}
			for node := start; node != "" && !done[node]; {
				if path[node] {
					return f.Fail(409, "MANAGER_CYCLE", "该用途的实际汇报会形成循环")
				}
				path[node] = true
				edge := latest[node]
				if edge.Until != "" && edge.Until <= date {
					break
				}
				node = edge.Parent
			}
			for node := range path {
				done[node] = true
			}
		}
	}
	return nil
}
