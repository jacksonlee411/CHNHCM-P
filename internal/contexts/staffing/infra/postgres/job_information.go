package postgres

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

func unknownAssignment() error { return f.Fail(404, "ASSIGNMENT_UNAVAILABLE", "所选任职不可用") }

func (u *unit) assignmentIdentity(ctx context.Context, id string) (h.Action, error) {
	if !f.ValidID(id) {
		return h.Action{}, unknownAssignment()
	}
	a, err := scanHireAction(u.tx.QueryRow(ctx, hireActionSQL+" WHERE a.instance_id=$1 AND a.assignment_id=$2 AND a.kind IN ('Hire','AdditionalAssignment','ParallelEmployment','Rehire')", u.store.Instance, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return a, unknownAssignment()
	}
	return a, err
}

func (u *unit) assignmentAt(ctx context.Context, id, day, source string) (h.Action, error) {
	a, e := u.assignmentIdentity(ctx, id)
	if e != nil {
		return a, e
	}
	if source == "" {
		source = a.SourceVersion
	}
	if day == "" {
		day = u.now.In(f.Shanghai).Format(time.DateOnly)
	}
	// Before entry or after exit the original identity remains addressable.
	// Inside a covered period, authorize the arrangement in the requested source.
	e = u.tx.QueryRow(ctx, `SELECT coalesce(s.department_id,$5::uuid),coalesce(s.job_id,$6::uuid) FROM (SELECT 1) anchor
      LEFT JOIN staffing.assignment_segment s ON s.instance_id=$1 AND s.assignment_id=$2 AND s.version=$3 AND s.period @> $4::date`,
		u.store.Instance, id, source, day, a.Department, a.Job).Scan(&a.Department, &a.Job)
	if e == nil {
		e = u.requireEmployment(ctx, "read", a.Person, a.Employer, a.Department)
	}
	return a, e
}

func (u *unit) locationAt(ctx context.Context, id, date string) (*h.LocationRef, error) {
	if id == "" {
		return nil, nil
	}
	targets, err := u.PositionSources(ctx, "locationId", []string{id}, false)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, nil
	}
	source := targets[0]
	day, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return nil, err
	}
	label, err := app.HireTargetLabel(ctx, u, "locationId", source, f.Period{From: date, Until: day.AddDate(0, 0, 1).Format(time.DateOnly)})
	if err != nil || label == "" {
		return nil, err
	}
	return &h.LocationRef{ID: id, Name: label, Source: source.Source, Revision: source.Records[d.At(source.Timeline, date)].Revision}, nil
}

func (st *Store) AssignmentInfo(ctx context.Context, c f.Credentials, id, day, version string) (out h.AssignmentView, err error) {
	if day != "" && !f.ValidDate(day) || version != "" && !f.ValidID(version) {
		return out, f.Fail(400, "ASSIGNMENT_INPUT", "业务日或来源无效")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, day, version)
		if e != nil {
			return e
		}
		if day == "" {
			day = u.now.In(f.Shanghai).Format("2006-01-02")
		}
		history, e := u.employmentAllowed(ctx, "history", a.Person, a.Employer, a.Department, false)
		if e != nil {
			return e
		}
		if version == "" {
			version = a.SourceVersion
		} else if !history {
			return f.Fail(403, "ASSIGNMENT_HISTORY", "当前权限不允许查看该来源")
		}
		var exists bool
		e = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.assignment_publication WHERE instance_id=$1 AND assignment_id=$2 AND version=$3)`, st.Instance, id, version).Scan(&exists)
		if e != nil {
			return e
		}
		if !exists {
			return unknownAssignment()
		}
		out = h.AssignmentView{Person: a.Person, Assignment: id, Employment: a.Employment, RecordNumber: a.RecordNumber, PrimaryJob: a.PrimaryJob, SourceVersion: version, BusinessAt: day, Actions: []h.Action{}}
		recognized, _, e := st.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), c, []string{a.Person}, "")
		if e != nil {
			return e
		}
		if len(recognized) != 1 {
			return unknownAssignment()
		}
		out.Name, out.Number = recognized[0].Name, recognized[0].Number
		out.Period = a.Period
		if version != a.SourceVersion {
			if e = u.tx.QueryRow(ctx, `SELECT coalesce((SELECT x.first_inactive::text FROM staffing.assignment_exit x JOIN staffing.assignment_member m USING(instance_id,assignment_id,action_id,revision) WHERE x.instance_id=$1 AND x.assignment_id=$2 AND m.version=$3),upper(period)::text,'') FROM staffing.assignment WHERE instance_id=$1 AND id=$2`, st.Instance, id, version).Scan(&out.Until); e != nil {
				return e
			}
		}
		if out.Until != "" {
			last, _ := time.Parse(time.DateOnly, out.Until)
			out.LastActive = last.AddDate(0, 0, -1).Format(time.DateOnly)
		}
		e = u.tx.QueryRow(ctx, `SELECT e.relationship,coalesce((SELECT elem->>'entityName' FROM jsonb_array_elements(coalesce(r.adoption#>'{sources,legal}',r.adoption#>'{secondary,legal}')) elem WHERE (elem->>'from')<=$3 AND (coalesce(elem->>'until','')='' OR (elem->>'until')>$3) LIMIT 1),r.adoption#>>'{sources,legal,0,entityName}',r.adoption#>>'{secondary,legal,0,entityName}','') FROM staffing.employment e JOIN staffing.employment_action a ON (a.instance_id,a.employment_id)=(e.instance_id,e.id) JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(a.instance_id,a.id,a.revision) WHERE a.instance_id=$1 AND a.assignment_id=$2 AND a.kind IN ('Hire','AdditionalAssignment','ParallelEmployment','Rehire')`, st.Instance, id, day).Scan(&out.Relationship, &out.EmployerName)
		if e != nil {
			return e
		}
		var state h.State
		var location string
		e = u.tx.QueryRow(ctx, `SELECT lower(period)::text,coalesce(upper(period)::text,''),department_id,job_id,adoption->>'departmentName',adoption->>'jobName',coalesce(adoption->>'position',''),coalesce(adoption->>'positionName',''),coalesce(location_id::text,''),location_ref,grade_ref,(adoption->>'seats')::int,coalesce(adoption->>'handoverSource','')<>''
			FROM staffing.assignment_segment WHERE instance_id=$1 AND assignment_id=$2 AND version=$3 AND period @> $4::date`, st.Instance, id, version, day).Scan(&state.From, &state.Until, &state.DepartmentID, &state.JobID, &state.Department, &state.Job, &state.Position, &state.PositionName, &location, &state.Location, &state.Grade, &state.Seats, &state.Handover)
		if e == nil {
			if state.Location == nil && location != "" {
				// The original 00037 hire stored only the location identity.
				// Read a permitted label without claiming its current revision was adopted at hire.
				state.Location, e = u.locationAt(ctx, location, day)
				if state.Location != nil {
					state.Location.Source = ""
					state.Location.Revision = ""
				}
			}
			if e != nil {
				return e
			}
			if state.Handover && version == a.SourceVersion {
				usage, err := u.PositionUsage(ctx, state.Position)
				if err != nil {
					return err
				}
				state.Handover = false
				for _, v := range usage.Uses {
					if v.Assignment == a.Assignment && v.Effect == "handover" && v.Contains(day) {
						state.Handover = slices.ContainsFunc(usage.Uses, func(orig d.PositionUse) bool {
							return orig.Owner == "M04" && orig.Assignment == v.Incumbent && orig.Effect == "occupied" && orig.Contains(day)
						})
						break
					}
				}
			}
			out.State = &state
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if out.Probation, e = u.probationAt(ctx, a, version, day); e != nil {
			return e
		}
		out.Managers = []h.ManagerFact{}
		if out.State != nil {
			if out.Managers, e = u.managerFacts(ctx, a, day, version); e != nil {
				return e
			}
		}
		if !a.Cancelled && version == a.SourceVersion && out.State != nil {
			if out.CanSetManager, e = u.employmentAllowed(ctx, "manager.change", a.Person, a.Employer, a.Department, false); e != nil {
				return e
			}
			if out.CanRecordGrade, e = u.employmentAllowed(ctx, "grade.record", a.Person, a.Employer, a.Department, false); e != nil {
				return e
			}
			if out.CanPromote, e = u.employmentAllowed(ctx, "promotion", a.Person, a.Employer, a.Department, false); e != nil {
				return e
			}
			if out.Probation.Start == "" {
				out.CanStartProbation, e = u.employmentAllowed(ctx, "probation.start", a.Person, a.Employer, a.Department, false)
				if e != nil {
					return e
				}
			}
			if out.Probation.Start != "" && out.Probation.Confirmed == "" {
				out.CanConfirm, e = u.employmentAllowed(ctx, "probation.confirm", a.Person, a.Employer, "", true)
				if e != nil {
					return e
				}
			}
		}

		e = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.assignment_member m JOIN staffing.employment_action a ON (a.instance_id,a.id)=(m.instance_id,m.action_id)
            WHERE m.instance_id=$1 AND m.assignment_id=$2 AND m.version=$3 AND a.kind IN ('Hire','Rehire','ParallelEmployment'))
            AND EXISTS(SELECT FROM staffing.assignment_segment WHERE instance_id=$1 AND assignment_id=$2 AND version=$3 AND period @> $4::date)`, st.Instance, id, version, day).Scan(&out.RelationshipPrimary)
		if e != nil {
			return e
		}
		if !a.Cancelled && version == a.SourceVersion {
			var alreadyEnded bool
			if e = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.assignment_exit WHERE instance_id=$1 AND assignment_id=$2)`, st.Instance, id).Scan(&alreadyEnded); e != nil {
				return e
			}
			if out.State != nil && !alreadyEnded {
				out.CanLeave, e = u.employmentAllowed(ctx, "employment.leave", a.Person, a.Employer, a.Department, false)
				if e != nil {
					return e
				}
				if !a.PrimaryJob && !out.RelationshipPrimary {
					out.CanEnd, e = u.employmentAllowed(ctx, "assignment.end", a.Person, a.Employer, a.Department, false)
					if e != nil {
						return e
					}
				}
			}
			if out.State != nil {
				if out.CanAddSecondary, e = u.employmentAllowed(ctx, "assignment.add", a.Person, a.Employer, a.Department, false); e != nil {
					return e
				}
				if out.CanAddParallel, e = u.employmentAllowed(ctx, "employment.parallel", a.Person, a.Employer, a.Department, false); e != nil {
					return e
				}
			}
			out.CanRecordDuty, e = u.employmentAllowed(ctx, "duty.correct", a.Person, a.Employer, a.Department, false)
			if e != nil {
				return e
			}
			out.CanVerifyDuty, e = u.employmentAllowed(ctx, "duty.verify", a.Person, a.Employer, a.Department, false)
			if e != nil {
				return e
			}
			out.CanHandover, e = u.employmentAllowed(ctx, "position.handover", a.Person, a.Employer, a.Department, false)
			if e != nil {
				return e
			}
			out.CanTransfer, e = u.employmentAllowed(ctx, "transfer", a.Person, a.Employer, a.Department, false)
			if e != nil {
				return e
			}
			out.CanChangeLocation, e = u.employmentAllowed(ctx, "location.change", a.Person, a.Employer, a.Department, false)
			if e != nil {
				return e
			}
		}
		rows, e := u.tx.Query(ctx, assignmentActionSelect+` JOIN staffing.assignment_publication h ON h.instance_id=a.instance_id AND h.assignment_id=$2 AND h.version=$3 `+assignmentRevisionJoin+" WHERE a.instance_id=$1 ORDER BY r.business_date DESC,a.recorded_at DESC,a.id", st.Instance, id, version)
		if e != nil {
			return e
		}
		items, e := pgx.CollectRows(rows, func(row pgx.CollectableRow) (h.Action, error) { return scanHireAction(row) })
		if e != nil {
			return e
		}
		for _, item := range items {
			if h.RelationAction(item.Kind) {
				item.Assignment, item.RecordNumber, item.PrimaryJob, item.Department, item.Job = id, a.RecordNumber, a.PrimaryJob, a.Department, a.Job
				item.CanHistory = history
				out.Actions = append(out.Actions, item)
				continue
			}

			read, e := u.actionDisclosure(ctx, item, false)
			if e != nil {
				return e
			}
			if !read.from && !read.to {
				continue
			}
			history, e := u.actionDisclosure(ctx, item, true)
			if e != nil {
				return e
			}
			item.CanHistory = history.from || history.to
			mutable := !h.StartsAssignment(item.Kind) && item.Kind != "CancelHire" && item.Operation != "Delete" && item.Operation != "Withdraw" && !a.Cancelled && version == a.SourceVersion && item.CanHistory
			if item.Kind == "Transfer" {
				mutable = mutable && history.from && history.to
				if !read.to {
					item.Department, item.Job = read.before, read.beforeJob
				}
			}
			if mutable && item.Kind == "Promotion" {
				item.CanCorrectPromotion, e = u.employmentAllowed(ctx, "promotion", item.Person, item.Employer, item.Department, false)
				if e != nil {
					return e
				}
				item.CanWithdrawPromotion, e = u.employmentAllowed(ctx, "promotion.withdraw", item.Person, item.Employer, item.Department, false)
				if e != nil {
					return e
				}
			}
			if mutable && (item.Kind == "DataChange" || item.Kind == "Transfer") {
				permitted := func(permission string) (bool, error) {
					ok, e := u.employmentAllowed(ctx, permission, item.Person, item.Employer, item.Department, false)
					if e != nil || !ok || item.Kind != "Transfer" {
						return ok, e
					}
					return u.employmentAllowed(ctx, permission, item.Person, item.Employer, read.before, false)
				}
				var duty bool
				if item.Kind == "Transfer" {
					if e = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM staffing.duty_application WHERE instance_id=$1 AND action_id=$2)`, st.Instance, item.ID).Scan(&duty); e != nil {
						return e
					}
				}
				if duty {
					if item.CanCorrectDuty, e = permitted("duty.correct"); e != nil {
						return e
					}
					out.Actions = append(out.Actions, item)
					continue
				}
				if item.CanMaintain, e = permitted("record.redate"); e != nil {
					return e
				}
				if item.CanDelete, e = permitted("record.delete"); e != nil {
					return e
				}
				if item.Kind == "DataChange" {
					item.CanCorrectLocation, e = permitted("location.correct")
				}
				if item.Kind == "Transfer" {
					item.CanCorrectTransfer, e = permitted("transfer")
				}
				if e != nil {
					return e
				}
			}
			out.Actions = append(out.Actions, item)
		}
		for first := 0; first < len(out.Actions); {
			last := first + 1
			for last < len(out.Actions) && out.Actions[last].BusinessDate == out.Actions[first].BusinessDate {
				last++
			}
			entries := make([]h.HistoryEntry, 0, last-first)
			byID := map[string]h.Action{}
			for _, a := range out.Actions[first:last] {
				entries = append(entries, h.HistoryEntry{ID: a.ID, Previous: a.Previous})
				byID[a.ID] = a
			}
			ordered, e := h.OrderDay(entries)
			if e != nil {
				return e
			}
			for n, r := range ordered {
				out.Actions[first+n] = byID[r.ID]
			}
			first = last
		}
		return nil
	})
	return
}

func (u *unit) maintenancePeriod(ctx context.Context, a h.Action, day, kind string) (f.Period, error) {
	if !f.ValidDate(day) || !a.Includes(day) || a.Cancelled {
		return f.Period{}, f.Fail(409, "RECORD_PERIOD", "请选择实际任职期间的生效日期")
	}
	term := f.Period{From: day, Until: a.Until}
	var next string
	err := u.tx.QueryRow(ctx, `SELECT coalesce(min(r.business_date)::text,'') FROM staffing.assignment_member m
		JOIN staffing.employment_action a ON (a.instance_id,a.id)=(m.instance_id,m.action_id)
		JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(m.instance_id,m.action_id,m.revision)
		WHERE m.instance_id=$1 AND m.assignment_id=$2 AND m.version=$3 AND (a.kind=$5 OR ($5='Transfer' AND r.adoption ? 'transfer')) AND r.operation NOT IN ('Delete','Withdraw') AND r.business_date>$4::date`, u.store.Instance, a.Assignment, a.SourceVersion, day, kind).Scan(&next)
	if next != "" {
		term.Until = next
	}
	return term, err
}

func (st *Store) LocationOptions(ctx context.Context, c f.Credentials, id string, in h.LocationSearch) (out []h.LocationRef, err error) {
	out = []h.LocationRef{}
	if in.Page < 1 || in.Page > 1000000 || utf8.RuneCountInString(in.Search) > 400 {
		return nil, f.Fail(400, "LOCATION_SEARCH", "地点查询条件无效")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, in.BusinessDate, "")
		if e != nil {
			return e
		}
		allowed, e := u.employmentAllowed(ctx, "location.change", a.Person, a.Employer, a.Department, false)
		if e != nil {
			return e
		}
		if !allowed {
			if e = u.requireEmployment(ctx, "location.correct", a.Person, a.Employer, a.Department); e != nil {
				return e
			}
		}
		period, e := u.maintenancePeriod(ctx, a, in.BusinessDate, "DataChange")
		if e != nil {
			return e
		}
		needle := strings.ToLower(strings.TrimSpace(in.Search))
		skip := (in.Page - 1) * 20
		for offset := 0; ; offset += 64 {
			ids, e := u.PositionCandidateIDs(ctx, "locationId", period.From, needle, nil, offset, 64)
			if e != nil {
				return e
			}
			sources, e := u.PositionSources(ctx, "locationId", ids, false)
			if e != nil {
				return e
			}
			byID := map[string]app.PositionSource{}
			for _, src := range sources {
				byID[src.ID] = src
			}
			for _, id := range ids {
				src := byID[id]
				label, e := app.HireTargetLabel(ctx, u, "locationId", src, period)
				if e != nil {
					return e
				}
				if label == "" || !strings.Contains(strings.ToLower(label), needle) {
					continue
				}
				if skip > 0 {
					skip--
					continue
				}
				out = append(out, h.LocationRef{ID: id, Name: label, Source: src.Source, Revision: src.Records[d.At(src.Timeline, period.From)].Revision})
				if len(out) == 20 {
					return nil
				}
			}
			if len(ids) < 64 {
				return nil
			}
		}
	})
	return
}

func (st *Store) TransferOptions(ctx context.Context, c f.Credentials, id string, in h.TransferSearch) (items []org.EligibleEntity, err error) {
	field := map[string]string{"department": "departmentId", "job": "jobId", "position": "positionId"}[in.Kind]
	if field == "" || in.Job != "" && !f.ValidID(in.Job) || !f.ValidDate(in.BusinessDate) || in.Page < 1 || in.Page > 1000000 || utf8.RuneCountInString(in.Search) > 400 || (in.Department != "" && !f.ValidID(in.Department)) {
		return nil, f.Fail(400, "TRANSFER_INPUT", "调入目标查询无效")
	}
	items = []org.EligibleEntity{}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, in.BusinessDate, "")
		if e != nil {
			return e
		}
		if e = u.requireEmployment(ctx, "transfer", a.Person, a.Employer, a.Department); e != nil {
			return e
		}
		if in.Department != "" {
			if e = u.requireEmployment(ctx, "transfer", a.Person, a.Employer, in.Department); e != nil {
				return e
			}
		}
		term, e := u.maintenancePeriod(ctx, a, in.BusinessDate, "Transfer")
		if e != nil {
			return e
		}
		query := h.TargetSearch{Person: a.Person, Employer: a.Employer, Department: in.Department, Job: in.Job, Search: in.Search, Page: in.Page}
		if in.Kind == "position" {
			return u.positionTargets(ctx, term, query, "transfer", "transfer", &a, &items)
		}
		return u.arrangementTargets(ctx, term, query, field, "transfer", &items)
	})
	return
}
