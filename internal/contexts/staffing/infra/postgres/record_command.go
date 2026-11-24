package postgres

import (
	"context"
	"reflect"
	"slices"
	"strings"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
)

type recordEdit struct {
	h.RecordMaintenance
	kind, operation, location, action string
}

func (st *Store) ChangeLocation(ctx context.Context, c f.Credentials, key, id, action string, in h.LocationChange) (h.Receipt, error) {
	op := "Add"
	if in.Insert {
		op = "Insert"
	}
	if action != "" {
		op = "Correct"
	}
	edit := recordEdit{RecordMaintenance: h.RecordMaintenance{ExpectedSource: in.ExpectedSource, ExpectedRevision: in.ExpectedRevision, BusinessDate: in.BusinessDate, Requires: in.Requires, Insert: in.Insert}, kind: "DataChange", operation: op, location: in.Location, action: action}
	return st.employmentCommand(ctx, c, key, []any{"DataChange", id, action, in}, func(u *unit) (employmentResult, error) { return u.editRecord(ctx, id, edit) })
}

func (st *Store) MaintainRecord(ctx context.Context, c f.Credentials, id, action, command, key string, in h.RecordMaintenance) (h.Receipt, error) {
	edit := recordEdit{RecordMaintenance: in, action: action}
	switch command {
	case "transfer":
		edit.kind, edit.operation = "Transfer", "Add"
		if in.Insert {
			edit.operation = "Insert"
		}
		if action != "" {
			edit.operation = "Correct"
		}
	case "redate":
		edit.operation = "Redate"
	case "delete":
		edit.operation = "Delete"
	default:
		return h.Receipt{}, f.Fail(400, "RECORD_OPERATION", "办理操作无效")
	}
	return st.employmentCommand(ctx, c, key, []any{command, id, action, in}, func(u *unit) (employmentResult, error) { return u.editRecord(ctx, id, edit) })
}

func (e recordEdit) permission(kind string) string {
	switch e.operation {
	case "Delete":
		return "record.delete"
	case "Redate":
		return "record.redate"
	}
	if kind == "Transfer" {
		return "transfer"
	}
	if e.action != "" {
		return "location.correct"
	}
	return "location.change"
}

func (e recordEdit) validate() error {
	if !f.ValidID(e.ExpectedSource) || (e.action == "" && e.ExpectedRevision != "") || (e.action != "" && (!f.ValidID(e.action) || !f.ValidID(e.ExpectedRevision) || e.Insert)) ||
		(e.operation != "Delete" && !f.ValidDate(e.BusinessDate)) || (e.operation == "Delete" && (e.action == "" || e.BusinessDate != "")) ||
		((e.operation == "Redate" || e.operation == "Delete") && (e.Position != "" || e.Department != "" || e.Job != "" || e.Notice != "" || e.Requires != nil)) ||
		(e.kind == "Transfer" && ((e.Position == "" || e.Department != "") && !f.ValidID(e.Department) || (e.Position == "" || e.Job != "") && !f.ValidID(e.Job) || e.Position != "" && !f.ValidID(e.Position))) || (e.kind == "DataChange" && !f.ValidID(e.location)) {
		return f.Fail(400, "RECORD_INPUT", "请核对任职、原办理、生效日期与实际修改")
	}
	if e.Requires != nil && (!f.ValidID(e.Requires.Action) || !f.ValidID(e.Requires.Revision)) {
		return f.Fail(400, "RECORD_DEPENDENCY", "所引用的原办理或修订无效")
	}
	return nil
}

func (u *unit) editRecord(ctx context.Context, id string, edit recordEdit) (out employmentResult, err error) {
	if e := edit.validate(); e != nil {
		return out, e
	}
	a, e := u.assignmentIdentity(ctx, id)
	if e != nil {
		return out, e
	}
	if e = lock(ctx, u.tx, false, "staffing.hire.person", u.store.Instance, a.Person); e != nil {
		return out, e
	}
	a, e = u.assignmentIdentity(ctx, id)
	if e != nil {
		return out, e
	}
	checkDay := edit.BusinessDate
	if checkDay == "" && edit.action != "" {
		original, readErr := u.action(ctx, edit.action)
		if readErr != nil {
			return out, readErr
		}
		if original.Assignment != id {
			return out, unknownAssignment()
		}
		checkDay = original.BusinessDate
	}
	scoped, scopeErr := u.assignmentAt(ctx, id, checkDay, "")
	if scopeErr != nil {
		return out, scopeErr
	}
	if e = u.requireEmployment(ctx, edit.permission(edit.kind), a.Person, a.Employer, scoped.Department); e != nil {
		return out, e
	}
	if a.Cancelled || a.SourceVersion != edit.ExpectedSource {
		return out, assignmentConflict()
	}
	initial, records, e := u.historyRows(ctx, a)
	if e != nil {
		return out, e
	}
	originals := make(map[string]savedRecord, len(records))
	index := -1
	for i, r := range records {
		originals[r.ID] = r
		if r.ID == edit.action {
			index = i
		}
	}
	var target savedRecord
	if edit.action != "" {
		if index < 0 {
			return out, assignmentConflict()
		}
		target = records[index]
		if target.Kind == "Promotion" || target.Kind == "GradeRecord" {
			return out, f.Fail(409, "GRADE_OPERATION", "请从原职级办理使用适用的更正或撤销入口")
		}
		if target.Removed || target.Revision != edit.ExpectedRevision || (edit.kind != "" && target.Kind != edit.kind) {
			return out, assignmentConflict()
		}
		if edit.operation == "Correct" && edit.BusinessDate != target.Day {
			return out, f.Fail(409, "RECORD_REDATE", "请从原办理调整生效日期")
		}

	} else {
		target = savedRecord{HistoryEntry: h.HistoryEntry{ID: f.NewID(), Kind: edit.kind}}
		index = len(records)
		records = append(records, target)
	}
	day := edit.BusinessDate
	if edit.operation == "Delete" {
		day = target.Day
	}
	if !a.Includes(day) {
		return out, f.Fail(409, "RECORD_PERIOD", "请选择实际任期内的业务日期")
	}
	scope, e := u.assignmentPlacement(ctx, a, day)
	if e != nil {
		return out, e
	}
	permit := edit.permission(target.Kind)
	for _, grant := range []string{"read", permit} {
		if e = u.requireEmployment(ctx, grant, a.Person, a.Employer, scope.Department); e != nil {
			return out, e
		}
	}
	if edit.action != "" {
		oldScope, e := u.assignmentPlacement(ctx, a, target.Day)
		if e != nil {
			return out, e
		}
		for _, grant := range []string{"read", permit} {
			if e = u.requireEmployment(ctx, grant, a.Person, a.Employer, oldScope.Department); e != nil {
				return out, e
			}
		}
	}
	if target.Evidence.Transfer != nil {
		for _, dept := range []string{target.Evidence.Transfer.BeforeDepartment, target.Evidence.Transfer.After.Department} {
			for _, grant := range []string{"read", permit} {
				if e = u.requireEmployment(ctx, grant, a.Person, a.Employer, dept); e != nil {
					return out, e
				}
			}
		}
	}
	if edit.operation == "Redate" && day == target.Day {
		return out, f.Fail(409, "RECORD_UNCHANGED", "生效日期未改变")
	}
	if edit.kind == "Transfer" && edit.operation == "Correct" && edit.Requires == nil {
		target.Requires = nil
	}
	if edit.Requires != nil {
		prior, ok := originals[edit.Requires.Action]
		if !ok || prior.Removed || prior.Revision != edit.Requires.Revision || prior.Day >= day {
			return out, assignmentConflict()
		}
		priorAction, e := u.action(ctx, prior.ID)
		if e != nil {
			return out, e
		}
		visible, e := u.actionDisclosure(ctx, priorAction, true)
		if e != nil {
			return out, e
		}
		if !visible.from && !visible.to {
			return out, f.Fail(403, "RECORD_DEPENDENCY", "当前权限不允许采用该原办理")
		}
		if e = u.requireEmployment(ctx, "history", a.Person, a.Employer, scope.Department); e != nil {
			return out, e
		}
		target.Requires = edit.Requires
	}
	target.Revision = f.NewID()
	target.Day = day
	if edit.Previous != nil {
		if e = u.validatePreceding(ctx, a, day, edit.Previous); e != nil {
			return out, e
		}
		target.Previous = edit.Previous
	}
	target.Operation = edit.operation
	target.Removed = edit.operation == "Delete"
	if edit.kind == "Transfer" {
		if edit.Department != "" {
			for _, grant := range []string{"read", "transfer"} {
				if e = u.requireEmployment(ctx, grant, a.Person, a.Employer, edit.Department); e != nil {
					return out, e
				}
			}
		}
		if edit.action == "" && strings.TrimSpace(edit.Notice) == "" {
			return out, f.Fail(400, "TRANSFER_NOTICE", "请填写调动依据")
		}
		if edit.action != "" && target.Transfer != nil && edit.Position == target.Transfer.Position && sameTransferTargets(edit, *target.Transfer) &&
			(strings.TrimSpace(edit.Notice) == "" || strings.TrimSpace(edit.Notice) == target.Notice) && reflect.DeepEqual(target.Requires, originals[edit.action].Requires) {
			return out, f.Fail(409, "TRANSFER_UNCHANGED", "任职安排和调动依据均未改变")
		}

		if edit.Notice != "" {
			target.Notice = strings.TrimSpace(edit.Notice)
		}
		target.Transfer = &h.JobTransfer{Department: edit.Department, Job: edit.Job, Position: edit.Position}
		if edit.action == "" && scope.Adoption.Position == edit.Position && (edit.Department == "" || scope.Department == edit.Department) && (edit.Job == "" || scope.Job == edit.Job) {
			return out, f.Fail(409, "TRANSFER_UNCHANGED", "任职安排未改变")
		}
	}
	records[index] = target
	span := recordPeriod(a.Period, records, target.Kind, target.Day)
	if edit.kind == "DataChange" {
		if scope.Location != nil && scope.Location.ID == edit.location {
			return out, f.Fail(409, "LOCATION_UNCHANGED", "工作地点未改变")
		}
		loc, e := u.adoptHistoryLocation(ctx, edit.location, span)
		if e != nil {
			return out, e
		}
		locEvidence := h.LocationEvidence{After: *loc, Reason: "工作地点变更"}
		if target.Location != nil {
			old := *target.Location
			locEvidence.Replaced = &old
		}
		target.Location = loc
		target.Evidence.LocationChange = &locEvidence
		records[index] = target
	}
	return u.publishRecordEdit(ctx, a, initial, records, originals, index, permit, "")
}

func (u *unit) publishRecordEdit(ctx context.Context, a h.Action, initial []h.Placement, records []savedRecord, originals map[string]savedRecord, index int, permit, pair string) (out employmentResult, err error) {
	target := records[index]
	id := a.Assignment
	if e := u.lockRecordPositions(ctx, initial, records); e != nil {
		return out, e
	}
	if e := u.adoptRecordPeriods(ctx, a, records, originals, target.ID); e != nil {
		return out, e
	}
	projection, e := projectRecords(a, initial, records)
	if e != nil {
		return out, e
	}
	for _, seg := range projection.Segments {
		if seg.Grade != nil {
			if _, e = u.gradeForJob(ctx, seg.Job, seg.Grade.ID, seg.Period); e != nil {
				return out, e
			}
		}
	}
	if projection.Segments, e = u.restoredPositions(ctx, a, projection.Segments, records); e != nil {
		return out, e
	}
	previous, e := u.recordChanges(ctx, a, records, originals, projection, target.ID, permit)
	if e != nil {
		return out, e
	}
	// Save source and all affected revisions only after the whole candidate is legal.
	for i := range records {
		r := records[i]
		old, changed := previous[r.ID]
		if !changed {
			continue
		}
		if old == "" {
			e = u.newEmploymentAction(ctx, employmentEntry{id: r.ID, revision: r.Revision, kind: r.Kind, employment: a.Employment, assignment: id, date: r.Day, notice: r.Notice, operation: r.Operation, adoption: recordAdoption(r)})
		} else {
			e = u.appendRecordRevision(ctx, id, r, old)
		}
		if e != nil {
			return out, e
		}
		if r.ID != target.ID {
			_, e = u.tx.Exec(ctx, "INSERT INTO staffing.employment_outbox(instance_id,event_id,action_id,revision) VALUES($1,$2,$3,$4)", u.store.Instance, f.NewID(), r.ID, r.Revision)
			if e != nil {
				return out, e
			}
		}
	}
	target = records[index]
	if pair != "" {
		for _, other := range records {
			if other.ID == pair {
				_, e = u.tx.Exec(ctx, `INSERT INTO staffing.promotion_transfer(instance_id,assignment_id,promotion,promotion_revision,transfer,transfer_revision) VALUES($1,$2,$3,$4,$5,$6)`, u.store.Instance, a.Assignment, target.ID, target.Revision, other.ID, other.Revision)
				if e != nil {
					return out, e
				}
				break
			}
		}
	}
	e = u.saveReplayedSource(ctx, a, f.NewID(), target, records, projection.Segments)
	return employmentResult{action: target.ID, revision: target.Revision}, e
}

func recordPeriod(term f.Period, records []savedRecord, kind, day string) f.Period {
	period := f.Period{From: day, Until: term.Until}
	for _, r := range records {
		if !r.Removed && (r.Kind == kind || kind == "Transfer" && r.Transfer != nil) && r.Day > day && (period.Until == "" || r.Day < period.Until) {
			period.Until = r.Day
		}
	}
	return period
}

func (u *unit) adoptHistoryLocation(ctx context.Context, id string, period f.Period) (*h.LocationRef, error) {
	sources, e := u.PositionSources(ctx, "locationId", []string{id}, true)
	if e != nil {
		return nil, e
	}
	if len(sources) == 1 {
		s := sources[0]
		label, e := app.HireTargetLabel(ctx, u, "locationId", s, period)
		if e != nil {
			return nil, e
		}
		if label != "" {
			return &h.LocationRef{ID: id, Name: label, Source: s.Source, Revision: s.Records[d.At(s.Timeline, period.From)].Revision}, nil
		}
	}
	return nil, f.Fail(409, "LOCATION_UNAVAILABLE", "工作地点不适用于整个期间")
}

func (u *unit) adoptRecordPeriods(ctx context.Context, a h.Action, records []savedRecord, originals map[string]savedRecord, target string) error {
	ent, e := u.Enterprise(ctx, true)
	if e != nil {
		return e
	}
	var bu string
	if e = u.tx.QueryRow(ctx, "SELECT coalesce(bu_id::text,'') FROM staffing.assignment WHERE instance_id=$1 AND id=$2", u.store.Instance, a.Assignment).Scan(&bu); e != nil {
		return e
	}
	for i := range records {
		r := &records[i]
		if r.Removed {
			continue
		}
		kind := r.Kind
		if r.Transfer != nil {
			kind = "Transfer"
		}
		span := recordPeriod(a.Period, records, kind, r.Day)
		orig, exists := originals[r.ID]
		oldSpan := f.Period{}
		if exists {
			oldSpan = f.Period{From: orig.Day, Until: a.Until}
			for _, after := range originals {
				if !after.Removed && (after.Kind == orig.Kind || kind == "Transfer" && after.Transfer != nil) && after.Day > orig.Day && (oldSpan.Until == "" || after.Day < oldSpan.Until) {
					oldSpan.Until = after.Day
				}
			}
		}
		if r.ID != target && oldSpan.From == span.From && (oldSpan.Until == "" || (span.Until != "" && span.Until <= oldSpan.Until)) {
			continue
		}
		switch kind {
		case "Transfer":
			next := *r.Transfer
			if next.Position != "" {
				allowed, e := u.positionEmploymentScope(ctx, next.Position, span, a.Person, a.Employer, "transfer", "transfer")
				if e != nil {
					return e
				}
				if !allowed {
					return f.Fail(403, "TRANSFER_DENIED", "当前权限不允许采用该岗位")
				}
			}
			department, job := next.Department, next.Job
			if next.Position != "" && len(next.Segments) > 0 {
				department, job = "", ""
				for _, v := range next.Segments {
					if v.Position == "" {
						department, job = v.DepartmentID, v.JobID
						break
					}
				}
			}
			check := staff.AssignmentCheck{Period: span, Enterprise: ent.ID, Category: "Employee", Purpose: "transfer", Department: department, Job: job, Position: next.Position}
			segs, e := u.checkTransferTarget(ctx, a, check)
			if e != nil {
				return e
			}
			if next.Position != "" {
				next.Department, next.Job = segs[0].DepartmentID, segs[0].JobID
			}
			for _, seg := range segs {
				dept := seg.DepartmentID
				if dept == "" {
					dept = next.Department
				}
				for _, right := range []string{"read", "transfer"} {
					if e = u.requireEmployment(ctx, right, a.Person, a.Employer, dept); e != nil {
						return e
					}
				}
			}
			next.Segments = segs
			r.Transfer = &next
		case "DataChange":
			loc, e := u.adoptHistoryLocation(ctx, r.Location.ID, span)
			if e != nil {
				return e
			}
			legal, e := u.store.HirePurposes.CheckPurpose(dbtx.With(ctx, u.tx), u.credentials, a.Employer, org.PurposeCheck{Role: "employer", BU: bu, Location: loc.ID, Period: span})
			if e != nil {
				return e
			}
			next := *r.Evidence.LocationChange
			next.After = *loc
			next.Legal = legal
			r.Evidence.LocationChange = &next
			r.Location = loc
		}
	}
	return nil
}

// Rebase in business-date order so an explicit dependency can name the new
// revision of its earlier fact. Same-day independent edits have no such edge.
func (u *unit) recordChanges(ctx context.Context, a h.Action, records []savedRecord, originals map[string]savedRecord, projection h.HistoryProjection, target, permit string) (map[string]string, error) {
	order := make([]int, len(records))
	for i := range records {
		order[i] = i
	}
	slices.SortStableFunc(order, func(i, j int) int { return strings.Compare(records[i].Day, records[j].Day) })
	for first := 0; first < len(order); {
		last := first + 1
		for last < len(order) && records[order[last]].Day == records[order[first]].Day {
			last++
		}
		rows := make([]h.HistoryEntry, 0, last-first)
		indices := map[string]int{}
		for _, i := range order[first:last] {
			rows = append(rows, records[i].HistoryEntry)
			indices[records[i].ID] = i
		}
		sorted, err := h.OrderDay(rows)
		if err != nil {
			return nil, err
		}
		for offset, row := range sorted {
			order[first+offset] = indices[row.ID]
		}
		first = last
	}
	previous := map[string]string{}
	versions := map[string]string{}
	for _, i := range order {
		r := &records[i]
		old := originals[r.ID]
		if !r.Removed {
			change := projection.Changes[r.ID]
			if r.Previous != nil {
				ref := *r.Previous
				if revision := versions[ref.Action]; revision != "" {
					ref.Revision = revision
				}
				r.Previous = &ref
			}
			if r.Evidence.Promotion != nil {
				ev := *r.Evidence.Promotion
				ev.Before = change.Before.Grade
				ev.Department, ev.DepartmentName = change.Before.Department, change.Before.Adoption.DepartmentName
				if other := change.Alternate; other != nil {
					if other.Job != change.Before.Job {
						for _, v := range []struct{ job, grade string }{{other.Job, ev.Before.ID}, {change.Before.Job, ev.After.ID}} {
							if _, e := u.gradeForJob(ctx, v.job, v.grade, oneDay(r.Day)); e != nil {
								return nil, f.Fail(409, "PROMOTION_ORDER", "请核对晋升与调动的先后，确认当时适用的任职资格")
							}
						}
					}
					if change.Before.Department != other.Department {
						for _, right := range []string{"read", "promotion", permit} {
							ok, e := u.employmentAllowed(ctx, right, a.Person, a.Employer, "", false)
							if e != nil {
								return nil, e
							}
							if !ok {
								return nil, f.Fail(409, "PROMOTION_ORDER", "请核对晋升与调动的先后，以确定本次办理范围")
							}
						}
						ev.Department, ev.DepartmentName = "", "未确定"
					}
				}
				r.Evidence.Promotion = &ev
			}
			if r.Requires != nil {
				req := *r.Requires
				req.Revision = versions[req.Action]
				r.Requires = &req
			}
			if r.Location != nil {
				evidence := *r.Evidence.LocationChange
				evidence.Before = change.Before.Location
				r.Evidence.LocationChange = &evidence
			} else if r.Transfer != nil {
				r.Evidence.Transfer = &h.TransferEvidence{After: *r.Transfer, BeforeGrade: change.Before.Grade, AfterGrade: change.After.Grade, BeforeDepartment: change.Before.Department, BeforeJob: change.Before.Job,
					BeforePosition: change.Before.Adoption.Position, BeforePositionName: change.Before.Adoption.PositionName, BeforeDepartmentName: change.Before.Adoption.DepartmentName, BeforeJobName: change.Before.Adoption.JobName}
			}
			if !reflect.DeepEqual(recordAdoption(*r), recordAdoption(old)) || r.ID == target {
				departments := []string{change.Before.Department}
				if r.Transfer != nil {
					departments = append(departments, r.Transfer.Department)
				}
				if old.Evidence.Transfer != nil {
					departments = append(departments, old.Evidence.Transfer.BeforeDepartment, old.Evidence.Transfer.After.Department)
				}
				for _, dept := range departments {
					for _, grant := range []string{"read", permit} {
						if e := u.requireEmployment(ctx, grant, a.Person, a.Employer, dept); e != nil {
							return nil, e
						}
					}
				}
			}
		}
		if r.ID == target || !reflect.DeepEqual(recordAdoption(*r), recordAdoption(old)) {
			previous[r.ID] = old.Revision
			if r.ID != target && old.ID != "" {
				r.Operation = "Rebase"
				r.Revision = f.NewID()
			}
		}
		versions[r.ID] = r.Revision
	}
	return previous, nil
}

func sameTransferTargets(edit recordEdit, old h.JobTransfer) bool {
	if edit.Position == "" {
		return edit.Department == old.Department && edit.Job == old.Job
	}
	for _, v := range old.Segments {
		if v.Position == "" {
			return edit.Department == v.DepartmentID && edit.Job == v.JobID
		}
	}
	return (edit.Department == "" || edit.Department == old.Department) && (edit.Job == "" || edit.Job == old.Job)
}
