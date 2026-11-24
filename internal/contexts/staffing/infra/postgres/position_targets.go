package postgres

import (
	"context"
	"errors"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func (u *unit) arrangementRules(ctx context.Context) (string, []s.AssignmentPolicy, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return "", nil, e
	}
	if e = access.RequireRule(ctx, u.guard, ent.ID, "assignmentpolicy.check"); e != nil {
		return "", nil, e
	}
	repo := u.policyStore()
	head, e := repo.Head(ctx)
	if e != nil {
		return "", nil, e
	}
	rows, e := repo.Load(ctx, head)
	return head, rows, e
}
func (st *Store) HireArrangement(ctx context.Context, c f.Credentials, in h.ArrangementQuery) (out h.ArrangementMode, err error) {
	term, e := hireDates(in.Arrival, in.LastDay)
	if e != nil {
		return out, e
	}
	if !f.ValidID(in.Person) {
		return out, f.Fail(400, "HIRE_INPUT", "人员无效")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		ok, e := u.employmentAllowed(ctx, hirePermission(in.RehireOf), in.Person, "", "", true)
		if e != nil {
			return e
		}
		if !ok {
			return f.Fail(403, "HIRE_DENIED", "没有本次办理权限")
		}
		if in.RehireOf != nil {
			if e = u.checkRehireOrigin(ctx, in.Person, term.From, in.RehireOf); e != nil {
				return e
			}
		}
		_, policies, e := u.arrangementRules(ctx)
		if e != nil {
			return e
		}
		ent := u.enterprise.ID
		rules := app.QuotaPolicies(ent, policies)
		var days []string
		for _, r := range rules {
			if r.Purpose == "hire" && r.Category == "Employee" {
				days = append(days, r.From, r.Until)
			}
		}
		for _, p := range term.Split(days) {
			rule, e := d.ApplicablePolicy(rules, ent, "Employee", "hire", p.From)
			if e != nil {
				return e
			}
			if rule.Mode == "JobOnly" {
				out.JobRequired = true
			} else {
				out.PositionRequired = true
			}
		}
		if in.Position != "" && !out.JobRequired {
			if !f.ValidID(in.Position) {
				return f.Fail(400, "HIRE_INPUT", "岗位无效")
			}
			ok, e := u.positionEmploymentScope(ctx, in.Position, term, in.Person, "", hirePermission(in.RehireOf))
			if e != nil {
				return e
			}
			if !ok {
				return f.Fail(403, "HIRE_DENIED", "所选安排超出获权范围")
			}
			head, e := u.policyStore().Head(ctx)
			if e != nil {
				return e
			}
			segments, e := u.checkPositionAssignment(ctx, s.AssignmentCheck{Period: term, Enterprise: ent, Category: "Employee", Purpose: "hire", Position: in.Position}, head, policies, false)
			if e != nil {
				return e
			}
			v := segments[0]
			out.Department = &org.EligibleEntity{ID: v.DepartmentID, Name: v.DepartmentName}
			out.Job = &org.EligibleEntity{ID: v.JobID, Name: v.JobName}
		}
		return nil
	})
	return
}
func (u *unit) hirePositionTargets(ctx context.Context, term f.Period, in h.TargetSearch, permission string, out *[]org.EligibleEntity) error {
	head, policies, e := u.arrangementRules(ctx)
	if e != nil {
		return e
	}
	ent := u.enterprise.ID
	allowed, e := u.positionReadable(ctx, ent)
	if e != nil || !allowed {
		return e
	}
	skip := (in.Page - 1) * 20
	for offset := 0; ; offset += 64 {
		rs, e := u.tx.Query(ctx, `SELECT p.id FROM staffing.position p JOIN staffing.position_read r ON (p.instance_id,p.id,p.source_id)=(r.instance_id,r.object_id,r.source_id)
   WHERE p.instance_id=$1 AND p.enterprise_id=$2 AND r.active AND daterange(r.start_date,r.end_date,'[)') && daterange($3::date,nullif($4,'')::date,'[)')
   AND strpos(lower(r.fact->'data'->>'name'),$5)>0 GROUP BY p.id ORDER BY min(r.fact->'data'->>'name') COLLATE "C",p.id LIMIT 64 OFFSET $6`, u.store.Instance, ent, term.From, term.Until, strings.ToLower(strings.TrimSpace(in.Search)), offset)
		if e != nil {
			return e
		}
		ids, e := pgx.CollectRows(rs, pgx.RowTo[string])
		if e != nil {
			return e
		}
		for _, id := range ids {
			visible, e := u.positionEmploymentScope(ctx, id, term, in.Person, in.Employer, permission)
			if e != nil {
				return e
			}
			if !visible {
				continue
			}
			seg, e := u.checkPositionAssignment(ctx, s.AssignmentCheck{Period: term, Enterprise: ent, Category: "Employee", Purpose: "hire", Position: id, Job: in.Job, Department: in.Department}, head, policies, false)
			if e != nil {
				var fault *f.Fault
				if errors.As(e, &fault) && fault.Status < 500 {
					continue
				}
				return e
			}
			if skip > 0 {
				skip--
				continue
			}
			for _, p := range seg {
				if p.Position != "" {
					*out = append(*out, org.EligibleEntity{ID: id, Name: p.PositionName})
					break
				}
			}
			if len(*out) == 20 {
				return nil
			}
		}
		if len(ids) < 64 {
			return nil
		}
	}
}
