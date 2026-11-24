package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"errors"
	"slices"
)

func FunctionIDs(t Timeline) []string {
	var ids []string
	for _, r := range t.Records {
		if id := r.Data["jobFunctionId"]; id != "" {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

// Compare only intervals on which both timelines have a Job. Every possible
// change starts at a boundary on one of the two timelines.
func functionChanged(before, after Timeline) bool {
	// Each cursor advances once per record. Empty End is positive infinity.
	for i, j := 0, 0; i < len(before.Records) && j < len(after.Records); {
		old, next := before.Records[i], after.Records[j]
		overlap := (old.End == "" || next.Date < old.End) && (next.End == "" || old.Date < next.End)
		if overlap && old.Data["jobFunctionId"] != next.Data["jobFunctionId"] {
			return true
		}
		switch {
		case old.End == next.End:
			i++
			j++
		case next.End == "" || old.End != "" && old.End < next.End:
			i++
		default:
			j++
		}
	}
	return false
}

func planFunctions(p *Plan) error {
	for i := range p.After.Records {
		r := &p.After.Records[i]
		id := r.Data["jobFunctionId"]
		j := Index(p.Before, r.ID)
		if !r.Active() {
			prev := p.After.Records[i-1]
			// Cutover records each historical stop's own proof. Keep it on an
			// unchanged unregistered segment; a new predecessor supplies its own.
			keepLegacy := id == "" && j > 0 && !p.Before.Records[j].Active() &&
				p.Before.Records[j].Data["jobFunctionId"] == "" && p.Before.Records[j-1].ID == prev.ID &&
				p.Before.Records[j-1].Function == prev.Function
			if !keepLegacy {
				r.Function = prev.Function
			}
			continue
		}
		if j >= 0 {
			old := p.Before.Records[j]
			if id == "" && old.Data["jobFunctionId"] == "" && old.Function == (FunctionEvidence{}) {
				return errors.New("legacy job has no source evidence")
			}
			if old.Type == r.Type && old.Data["jobFunctionId"] == id && (old.Date == r.Date || id == "") {
				continue
			}
		}
		if id == "" {
			return fieldInvalid("jobFunctionId", "请选择职类")
		}
		if j < 0 && r.Type == "CHANGE" && i > 0 && p.After.Records[i-1].Data["jobFunctionId"] == id {
			r.Function = p.After.Records[i-1].Function
			continue
		}
		p.AdoptFunctions = append(p.AdoptFunctions, r.ID)
	}
	return nil
}

func FunctionCoverage(job, function Timeline, id string) error {
	for _, r := range job.Records {
		if r.Active() && r.Data["jobFunctionId"] == id && (function.ID != id || job.Enterprise != function.Enterprise || !Covers(function, r.Date, r.End)) {
			return &foundation.Fault{Status: 409, Code: "M03_REFERENCE_CONFLICT", Field: "jobFunctionId", Message: "职类不能覆盖职务的完整启用期间"}
		}
	}
	return nil
}
