package postgres

import (
	"context"
	"slices"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

// Position changes and both sides of a transfer share the department and target locks.
// Sorting across the whole change also handles two people moving in opposite directions.
func (u *unit) lockRecordPositions(ctx context.Context, initial []h.Placement, records []savedRecord) error {
	var positions []string
	for _, r := range initial {
		if r.Adoption.Position != "" {
			positions = append(positions, r.Adoption.Position)
		}
	}
	for _, r := range records {
		if r.Transfer != nil && r.Transfer.Position != "" {
			positions = append(positions, r.Transfer.Position)
		}
		if r.Evidence.Transfer != nil && r.Evidence.Transfer.After.Position != "" {
			positions = append(positions, r.Evidence.Transfer.After.Position)
		}
	}
	return u.lockPositions(ctx, positions)
}

func (u *unit) lockExitingPositions(ctx context.Context, assignments []string, firstInactive string) error {
	rows, err := u.tx.Query(ctx, `SELECT DISTINCT s.adoption->>'position' FROM staffing.assignment_head h JOIN staffing.assignment_segment s USING(instance_id,assignment_id,version)
       WHERE h.instance_id=$1 AND h.assignment_id=ANY($2::uuid[]) AND coalesce(s.adoption->>'position','')<>'' AND s.period && daterange($3::date,NULL,'[)')`, u.store.Instance, assignments, firstInactive)
	if err != nil {
		return err
	}
	positions, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	return u.lockPositions(ctx, positions)
}

func (u *unit) lockPositions(ctx context.Context, positions []string) error {
	var departments []string
	if len(positions) == 0 {
		return nil
	}
	if _, e := u.Enterprise(ctx, true); e != nil {
		return e
	}
	if e := u.policyLock(ctx, true); e != nil {
		return e
	}
	slices.Sort(positions)
	positions = slices.Compact(positions)
	sources := make(map[string]string, len(positions))
	for _, id := range positions {
		p, e := u.Load(ctx, d.Position, id, "")
		if e != nil {
			return e
		}
		sources[id] = p.Source
		departments = append(departments, d.QuotaEligibility(p).Departments()...)
	}
	if e := u.LockQuotaDepartments(ctx, departments); e != nil {
		return e
	}
	for _, id := range positions {
		if e := u.LockQuotaTarget(ctx, id); e != nil {
			return e
		}
		p, e := u.Load(ctx, d.Position, id, "")
		if e != nil {
			return e
		}
		if p.Source != sources[id] {
			return f.Fail(409, "M03_SOURCE_CHANGED", "岗位归属已变化，请重新核对")
		}
	}
	return nil
}

// Only the current published assignment can carry its existing duty through a pause.
func (u *unit) retainedPosition(ctx context.Context, a h.Action, position string) ([]f.Period, error) {
	rows, e := u.tx.Query(ctx, `SELECT lower(period)::text,coalesce(upper(period)::text,'') FROM staffing.assignment_segment
        WHERE instance_id=$1 AND assignment_id=$2 AND version=$3 AND adoption->>'position'=$4 ORDER BY lower(period)`, u.store.Instance, a.Assignment, a.SourceVersion, position)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[f.Period])
}
func (u *unit) checkTransferTarget(ctx context.Context, a h.Action, in s.AssignmentCheck) ([]s.AssignmentSegment, error) {
	var retained []f.Period
	if in.Position != "" {
		var e error
		retained, e = u.retainedPosition(ctx, a, in.Position)
		if e != nil {
			return nil, e
		}
	}
	return u.store.checkOwnerAssignment(dbtx.With(ctx, u.tx), u.credentials, in, retained, "M04", a.Assignment, nil)
}

// Moving or deleting the first transfer can restore part of the starting appointment.
// Recheck the additional period on its own admission policy and preserve untouched sources.
func (u *unit) restoredPositions(ctx context.Context, a h.Action, segments []h.Placement, records []savedRecord) ([]h.Placement, error) {
	first := ""
	for _, r := range records {
		if !r.Removed && r.Transfer != nil && r.Transfer.Duty == nil && (first == "" || r.Day < first) {
			first = r.Day
		}
	}
	var out []h.Placement
	for _, seg := range segments {
		actual := slices.ContainsFunc(records, func(r savedRecord) bool {
			return !r.Removed && r.Transfer != nil && r.Transfer.Duty != nil && r.Transfer.Duty.Includes(seg.From)
		})
		if actual || seg.Adoption.Position == "" || first != "" && seg.From >= first {
			out = append(out, seg)
			continue
		}
		held, e := u.retainedPosition(ctx, a, seg.Adoption.Position)
		if e != nil {
			return nil, e
		}
		var days []string
		for _, p := range held {
			days = append(days, p.From, p.Until)
		}
		for _, span := range seg.Split(days) {
			part := seg
			part.Period = span
			if slices.ContainsFunc(held, func(p f.Period) bool { return p.Includes(span.From) }) {
				out = append(out, part)
				continue
			}
			if e := u.requireEmployment(ctx, "read", a.Person, a.Employer, seg.Department); e != nil {
				return nil, e
			}
			if e := u.requireEmployment(ctx, "transfer", a.Person, a.Employer, seg.Department); e != nil {
				return nil, e
			}
			purpose := "hire"
			if a.Kind == "AdditionalAssignment" || a.Kind == "ParallelEmployment" {
				purpose = "secondary"
			}
			adopted, e := u.store.checkOwnerAssignment(dbtx.With(ctx, u.tx), u.credentials, s.AssignmentCheck{Period: span, Enterprise: u.enterprise.ID, Category: "Employee", Purpose: purpose, Department: seg.Department, Job: seg.Job, Position: seg.Adoption.Position}, nil, "M04", a.Assignment, nil)
			if e != nil {
				return nil, e
			}
			for _, v := range adopted {
				if v.Position != seg.Adoption.Position || v.DepartmentID != seg.Department || v.JobID != seg.Job {
					return nil, f.Fail(409, "POSITION_TARGET_CHANGED", "原任职无法覆盖恢复期间，请核对岗位归属")
				}
				part.Period, part.Adoption = v.Period, v
				out = append(out, part)
			}
		}
	}
	return out, nil
}
