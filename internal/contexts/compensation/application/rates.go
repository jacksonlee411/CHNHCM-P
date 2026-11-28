package application

import (
	"slices"
	"strings"
	"time"

	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	st "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type RateState struct {
	ID, Enterprise, Version string
	Key                     d.RateKey
	Places                  int
	Records                 d.Timeline[d.RateTerms]
}

func NewRate(input c.WriteRate) (RateState, error) {
	key := d.RateKey{GradeID: input.GradeID, Currency: input.Currency, Frequency: input.Frequency, Unit: input.Unit}
	if err := key.Validate(); err != nil {
		return RateState{}, err
	}
	precision := 2
	if input.Precision != nil {
		precision = *input.Precision
	} else if key.Currency != "CNY" {
		return RateState{}, d.Invalid("precision", "请明确此币种的金额小数位数")
	}
	if precision < 0 || precision > 18 || key.Currency == "CNY" && key.Frequency != "HOUR" && precision != 2 {
		return RateState{}, d.Invalid("precision", "人民币年、月及次额保留两位小数，其他口径须使用有效精度")
	}
	return RateState{ID: foundation.NewID(), Key: key, Places: precision}, nil
}

var RateActions = []string{"ADD", "CORRECT", "DELETE", "DEACTIVATE", "REACTIVATE"}

func Fields(input c.WriteRate) ([]string, error) {
	action := input.Action
	if !slices.Contains(RateActions, action) && action != "CREATE" {
		return nil, d.Invalid("action", "请选择合法办理动作")
	}
	fields := []string{}
	if action == "CREATE" {
		if input.ExpectedVersion != "" || input.ExpectedRevision != "" || input.RecordID != "" {
			return nil, d.Invalid("recordId", "新建不接受已有记录身份")
		}
		fields = append(fields, "gradeId", "currency", "frequency", "unit", "precision")
	} else {
		if input.GradeID != "" || input.Currency != "" || input.Frequency != "" || input.Unit != "" || input.Precision != nil {
			return nil, d.Invalid("gradeId", "已有标准保留原职级和计量口径")
		}
		if !foundation.ValidID(input.ExpectedVersion) || !foundation.ValidID(input.ExpectedRevision) || !foundation.ValidID(input.RecordID) {
			return nil, d.Invalid("recordId", "请选择准确的记录版本")
		}
	}
	if action == "DELETE" && (input.EffectiveDate != "" || input.Reason != "") {
		return nil, d.Invalid("action", "撤去使用原记录的日期和分类")
	}
	if input.EffectiveDate != "" {
		fields = append(fields, "effectiveDate")
	}
	if input.Reason != "" {
		fields = append(fields, "reason")
	}
	for field := range input.Changes {
		if !slices.Contains([]string{"form", "amount", "minimum", "guidance", "maximum"}, field) || action == "DELETE" || action == "DEACTIVATE" {
			return nil, d.Invalid(field, "此动作不接受该字段")
		}
		fields = append(fields, field)
	}
	slices.Sort(fields)
	return fields, nil
}

type ChangedRate struct {
	Records              d.Timeline[d.RateTerms]
	Target               d.Record[d.RateTerms]
	Previous             *d.Record[d.RateTerms]
	Until, PreviousUntil string
	Changed              bool
}

func PrepareRate(state RateState, input c.WriteRate, grade st.CompensationGrade) (ChangedRate, error) {
	var change ChangedRate
	if state.Version != input.ExpectedVersion {
		return change, d.Conflict()
	}
	before := state.Records
	index := slices.IndexFunc(before, func(r d.Record[d.RateTerms]) bool { return r.ID == input.RecordID })
	if index >= 0 {
		change.Previous = &before[index]
		change.PreviousUntil = before.Period(index).Until
		if !before[index].Active && input.Action == "CORRECT" && len(input.Changes) != 0 {
			return change, d.Invalid("changes", "停用记录沿用前段金额")
		}
	}
	if input.Reason != "" {
		operation := input.Action
		if operation == "CORRECT" && change.Previous != nil {
			operation = change.Previous.Operation
		}
		if !slices.Contains(d.RateReasons(operation), strings.TrimSpace(input.Reason)) {
			return change, d.Invalid("reason", "请选择该操作的办理原因")
		}
	}
	edit := d.Edit{Action: input.Action, Date: input.EffectiveDate, Reason: strings.TrimSpace(input.Reason), RecordID: input.RecordID, ExpectedRevision: input.ExpectedRevision}
	records, target, err := d.Amend(before, edit, func(base d.RateTerms) (d.RateTerms, error) {
		amounts, err := base.Amounts.Apply(input.Changes, state.Places)
		base.Amounts = amounts
		return base, err
	})
	if err != nil {
		return change, err
	}
	// An unchanged correction or future copy preserves the prior adoption and receipt target.
	if change.Previous != nil {
		prev := *change.Previous
		if input.Action == "ADD" && target.Data.Amounts == prev.Data.Amounts ||
			input.Action == "CORRECT" && target.Data.Amounts == prev.Data.Amounts && target.Start == prev.Start && target.Reason == prev.Reason {
			return ChangedRate{Records: before, Target: prev}, nil
		}
	}
	segments := make([]d.GradeSegment, 0, len(grade.Segments))
	for _, s := range grade.Segments {
		segments = append(segments, d.GradeSegment{Period: s.Period, Active: s.Active, Evidence: d.GradeTerms{Source: grade.Source, Record: s.RecordID, Revision: s.Revision, Name: s.Name, Code: grade.Code}})
	}
	oldRecords := make(map[string]int, len(before))
	for i, item := range before {
		oldRecords[item.ID] = i
	}
	for pos := range records {
		rec := &records[pos]
		period := records.Period(pos)
		if rec.ID == target.ID {
			change.Until = period.Until
		}
		old, existed := oldRecords[rec.ID]
		if !rec.Active {
			// Deactivation inherits the exact preceding terms, including later
			// corrections. Its former immutable revision remains addressable.
			if pos > 0 && rec.Data != records[pos-1].Data {
				rec.Data = records[pos-1].Data
				rec.Revision = foundation.NewID()
			}
			continue
		}
		if existed && rec.Revision == before[old].Revision && period == before.Period(old) {
			continue
		}
		evidence, err := d.CoverGrade(period, segments)
		if err != nil {
			return change, err
		}
		if rec.ID == target.ID {
			rec.Data.Grade = evidence
		}
	}
	if input.Action != "DELETE" {
		for _, r := range records {
			if r.ID == target.ID {
				target = r
				break
			}
		}
	}
	change.Target, change.Records, change.Changed = target, records, true
	return change, nil
}

// MaintenanceWindow offers the same order and open-ended bounds that Amend checks at save time.
func MaintenanceWindow[T any](records d.Timeline[T], index int, action, today, enterpriseStart string) c.Window {
	shift := func(day string, offset int) string {
		value, _ := time.Parse(time.DateOnly, day)
		shifted := value.AddDate(0, 0, offset).Format(time.DateOnly)
		if !foundation.ValidDate(shifted) {
			return ""
		}
		return shifted
	}
	current := records[index].Start
	w := c.Window{Minimum: shift(current, 1)}
	if index+1 < len(records) {
		w.Maximum = shift(records[index+1].Start, -1)
	}
	if action == "CORRECT" {
		w.Minimum = enterpriseStart
		if index > 0 {
			w.Minimum = shift(records[index-1].Start, 1)
		}
		w.Default = current
	} else {
		w.Default = max(today, w.Minimum)
		if w.Maximum != "" && w.Default > w.Maximum {
			w.Default = w.Minimum
		}
	}
	w.Available = w.Minimum != "" && (w.Maximum == "" || w.Minimum <= w.Maximum)
	if !w.Available {
		w.Default = ""
	}
	return w
}
