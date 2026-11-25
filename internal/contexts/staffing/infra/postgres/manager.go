package postgres

import (
	"context"
	"slices"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

// Serialize changes to the actual reporting graph. Taking this before any
// person lock also orders exits and concurrent reciprocal appointments.
func (u *unit) lockReporting(ctx context.Context) error {
	return lock(ctx, u.tx, false, "staffing.manager.graph", u.store.Instance)
}

func (u *unit) lifecycleSource(ctx context.Context, assignment, old, action, revision, cutoff string) error {
	return u.cloneAssignmentSource(ctx, assignment, old, action, revision, cutoff, []h.ActionRef{{Action: action, Revision: revision}})
}
func (u *unit) cloneAssignmentSource(ctx context.Context, assignment, old, action, revision, cutoff string, refs []h.ActionRef) error {
	version := f.NewID()
	if err := u.startSourceReplacing(ctx, assignment, old, version, action, revision, refs); err != nil {
		return err
	}
	_, err := u.tx.Exec(ctx, `INSERT INTO staffing.assignment_segment(instance_id,assignment_id,version,period,location_id,location_ref,adoption,department_id,job_id,grade_ref)
		SELECT instance_id,assignment_id,$4,period * daterange(NULL,nullif($5,'')::date,'[)'),location_id,location_ref,adoption,department_id,job_id,grade_ref
		FROM staffing.assignment_segment WHERE instance_id=$1 AND assignment_id=$2 AND version=$3 AND period && daterange(NULL,nullif($5,'')::date,'[)')`,
		u.store.Instance, assignment, old, version, cutoff)
	if err != nil {
		return err
	}
	return u.publishAssignmentSource(ctx, assignment, old, version)
}

func (u *unit) managerFacts(ctx context.Context, a h.Action, day, version string) ([]h.ManagerFact, error) {
	rows, e := u.tx.Query(ctx, `SELECT purpose,coalesce(manager_assignment::text,''),lower(period)::text,coalesce(upper(period)::text,''),name
	FROM (SELECT DISTINCT ON (g.purpose) g.*,coalesce(r.adoption#>>'{manager,name}','') AS name FROM staffing.manager_agreement g
	JOIN staffing.employment_action_revision r USING(instance_id,assignment_id,action_id,revision)
	JOIN staffing.assignment_member m USING(instance_id,assignment_id,action_id,revision)
	WHERE g.instance_id=$1 AND g.assignment_id=$2 AND m.version=$3 AND lower(g.period)<=$4::date
	ORDER BY g.purpose,lower(g.period) DESC) at_day WHERE period @> $4::date AND manager_assignment IS NOT NULL ORDER BY purpose`,
		u.store.Instance, a.Assignment, version, day)
	if e != nil {
		return nil, e
	}
	facts, e := pgx.CollectRows(rows, func(r pgx.CollectableRow) (v h.ManagerFact, e error) {
		e = r.Scan(&v.Purpose, &v.Manager, &v.From, &v.Until, &v.Name)
		return
	})
	if e != nil {
		return nil, e
	}
	visible := []h.ManagerFact{}
	for _, fact := range facts {
		ok, e := u.discloseManager(ctx, &fact, day)
		if e != nil {
			return nil, e
		}
		if ok {
			visible = append(visible, fact)
		}
	}
	return visible, nil
}

func (u *unit) discloseManager(ctx context.Context, fact *h.ManagerFact, day string) (bool, error) {
	// A manager's department may have changed since this source was issued.
	// Re-authorize the target independently even when the subordinate is visible.
	manager, e := u.assignmentIdentity(ctx, fact.Manager)
	if e != nil {
		return false, e
	}
	var dept string
	e = u.tx.QueryRow(ctx, `SELECT coalesce((SELECT department_id FROM staffing.assignment_segment WHERE instance_id=$1 AND assignment_id=$2 AND version=$3 AND period @> $4::date),$5::uuid)`, u.store.Instance, manager.Assignment, manager.SourceVersion, day, manager.Department).Scan(&dept)
	if e != nil {
		return false, e
	}
	ok, e := u.employmentAllowed(ctx, "read", manager.Person, manager.Employer, dept, false)
	if !ok || e != nil {
		return false, e
	}
	people, _, e := u.store.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), u.credentials, []string{manager.Person}, "")
	if e != nil || len(people) != 1 {
		return false, e
	}
	if fact.Name == "" {
		fact.Name = people[0].Name
	}
	fact.RecordNumber = manager.RecordNumber
	return true, nil
}

func (st *Store) SetManager(ctx context.Context, c f.Credentials, id, key string, in h.ManagerCommand) (h.Receipt, error) {
	return st.hireCommand(ctx, c, key, []any{"SetManager", id, in}, func(u *unit) (string, error) {
		term, e := hireDates(in.BusinessDate, in.LastDay)
		if e != nil {
			return "", e
		}
		if !f.ValidID(in.Expected) || !slices.Contains([]string{"direct", "functional"}, in.Purpose) || in.Manager != "" && !f.ValidID(in.Manager) {
			return "", f.Fail(400, "MANAGER_INPUT", "请核对实际经理、用途及期间")
		}
		if e = u.lockReporting(ctx); e != nil {
			return "", e
		}
		a, e := u.assignmentIdentity(ctx, id)
		if e != nil {
			return "", e
		}
		if e = lock(ctx, u.tx, false, "staffing.hire.person", st.Instance, a.Person); e != nil {
			return "", e
		}
		a, e = u.assignmentAt(ctx, id, term.From, "")
		if e != nil {
			return "", e
		}
		if e = u.requireEmployment(ctx, "manager.change", a.Person, a.Employer, a.Department); e != nil {
			return "", e
		}
		if a.Cancelled || a.SourceVersion != in.Expected {
			return "", assignmentConflict()
		}
		fact := h.ManagerFact{Period: term, Manager: in.Manager, Purpose: in.Purpose}
		endpoints := []string{id}
		if in.Manager != "" {
			if in.Manager == id {
				return "", f.Fail(409, "MANAGER_CYCLE", "任职不能以自身为实际经理")
			}
			ok, e := u.discloseManager(ctx, &fact, term.From)
			if e != nil {
				return "", e
			}
			if !ok {
				return "", f.Fail(403, "MANAGER_READ", "所选经理任职不可用")
			}
			endpoints = append(endpoints, in.Manager)
		}
		for _, endpoint := range endpoints {
			var until string
			e := u.tx.QueryRow(ctx, `SELECT coalesce(upper(piece)::text,'') FROM
			(SELECT unnest(range_agg(s.period)) piece FROM staffing.assignment_head head JOIN staffing.assignment_segment s USING(instance_id,assignment_id,version) WHERE head.instance_id=$1 AND head.assignment_id=$2) ranges WHERE piece @> $3::date`, st.Instance, endpoint, term.From).Scan(&until)
			if e == pgx.ErrNoRows {
				return "", f.Fail(409, "MANAGER_PERIOD", "所选日期缺少有效任职")
			}
			if e != nil {
				return "", e
			}
			if until != "" && (fact.Until == "" || fact.Until > until) {
				if in.LastDay != "" {
					return "", f.Fail(409, "MANAGER_PERIOD", "汇报期间超出实际任期")
				}
				fact.Until = until
			}
		}
		rows, e := u.tx.Query(ctx, `SELECT g.assignment_id,coalesce(g.manager_assignment::text,''),lower(g.period)::text,coalesce(upper(g.period)::text,'')
		FROM staffing.manager_agreement g JOIN staffing.assignment_head h USING(instance_id,assignment_id)
		JOIN staffing.assignment_member m ON (m.instance_id,m.assignment_id,m.version,m.action_id,m.revision)=(h.instance_id,h.assignment_id,h.version,g.action_id,g.revision)
		WHERE g.instance_id=$1 AND g.purpose=$2`, st.Instance, in.Purpose)
		if e != nil {
			return "", e
		}
		edges, e := pgx.CollectRows(rows, pgx.RowToStructByPos[h.ManagerEdge])
		if e != nil {
			return "", e
		}
		if e = h.ValidateManagers(append(edges, h.ManagerEdge{Child: id, Parent: in.Manager, From: fact.From, Until: fact.Until})); e != nil {
			return "", e
		}
		if in.Manager == "" {
			var active bool
			e = u.tx.QueryRow(ctx, `SELECT coalesce((SELECT manager_assignment IS NOT NULL AND period @> $4::date FROM staffing.manager_agreement WHERE instance_id=$1 AND assignment_id=$2 AND purpose=$3 AND lower(period)<=$4::date ORDER BY lower(period) DESC LIMIT 1),false)`, st.Instance, id, in.Purpose, term.From).Scan(&active)
			if e != nil {
				return "", e
			}
			if !active {
				return "", f.Fail(409, "MANAGER_ABSENT", "该用途没有需结束的实际汇报")
			}
		}
		return u.storeManagerFact(ctx, a, fact, "", nil)
	})
}

func (u *unit) storeManagerFact(ctx context.Context, a h.Action, fact h.ManagerFact, notice string, basis *h.ActionRef) (string, error) {
	item := employmentEntry{id: f.NewID(), revision: f.NewID(), kind: "SetManager", employment: a.Employment, assignment: a.Assignment, date: fact.From, notice: notice, adoption: map[string]any{"manager": fact, "managerExit": basis}}
	if e := u.newEmploymentAction(ctx, item); e != nil {
		return "", e
	}
	_, e := u.tx.Exec(ctx, `INSERT INTO staffing.manager_agreement(instance_id,assignment_id,action_id,revision,manager_assignment,purpose,period) VALUES($1,$2,$3,$4,nullif($5,'')::uuid,$6,daterange($7::date,nullif($8,'')::date,'[)'))`, u.store.Instance, a.Assignment, item.id, item.revision, fact.Manager, fact.Purpose, fact.From, fact.Until)
	if e != nil {
		return "", e
	}
	return item.id, u.lifecycleSource(ctx, a.Assignment, a.SourceVersion, item.id, item.revision, "")
}
