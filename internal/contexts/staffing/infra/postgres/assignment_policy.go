package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"chnhcm.local/chnhcm/internal/platform/rulesource"
	"context"
	"errors"
	"slices"
	"time"
)

var assignmentPolicySQL = rulesource.Statements{
	Head:         `SELECT version::text FROM staffing.assignment_policy_source WHERE instance_id=$1 AND owner_id=$2 ORDER BY sequence DESC LIMIT 1`,
	SourceExists: `SELECT EXISTS(SELECT FROM staffing.assignment_policy_source WHERE instance_id=$1 AND owner_id=$2 AND version=$3)`,
	Members:      `SELECT fact FROM staffing.assignment_policy_member WHERE instance_id=$1 AND owner_id=$2 AND version=$3 ORDER BY lower(period),record_id`,
	History:      `SELECT version::text,recorded_at FROM staffing.assignment_policy_source WHERE instance_id=$1 AND owner_id=$2 ORDER BY sequence DESC`,
	Replay: `SELECT owner_id::text, 'accepted', version::text, input_digest, 0, '', ''
		FROM staffing.assignment_policy_source WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3
		UNION ALL SELECT owner_id::text, outcome, coalesce(version::text,''), input_digest, coalesce(failure_status,0), coalesce(failure_code,''), coalesce(failure_message,'')
		FROM staffing.assignment_policy_terminal WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3`,
	Terminal: `INSERT INTO staffing.assignment_policy_terminal
		(instance_id,subject_id,submission_id,owner_id,outcome,version,input_digest,failure_status,failure_code,failure_message)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10)`,
	InsertSource: `INSERT INTO staffing.assignment_policy_source(instance_id,owner_id,version,subject_id,submission_id,input_digest) VALUES($1,$2,$3,$4,$5,$6)`,
	InsertMember: `INSERT INTO staffing.assignment_policy_member(instance_id,owner_id,version,record_id,revision,period,fact) VALUES($1,$2,$3,$4,$5,daterange($6::date,NULLIF($7,'')::date,'[)'),$8)`,
	Publish:      `INSERT INTO staffing.assignment_policy_outbox(instance_id,owner_id,version) VALUES($1,$2,$3)`,
}

func (u *unit) policyStore() rulesource.Store[s.AssignmentPolicy] {
	return rulesource.Store[s.AssignmentPolicy]{SQL: assignmentPolicySQL, Tx: u.tx, Instance: u.store.Instance, Owner: u.enterprise.ID}
}
func (u *unit) policyLock(ctx context.Context, shared bool) error {
	return lock(ctx, u.tx, shared, "staffing.assignment_policy", u.store.Instance, u.enterprise.ID)
}
func (st *Store) policy(ctx context.Context, c f.Credentials, action string, write bool, fn func(*unit, bool) error) error {
	run := func(u *unit, locked bool) error {
		ent, e := u.Enterprise(ctx, locked)
		if e != nil {
			return e
		}
		if e = access.RequireRule(ctx, u.guard, ent.ID, "assignmentpolicy."+action); e != nil {
			return e
		}
		if locked {
			if e = u.policyLock(ctx, !write); e != nil {
				return e
			}
		}
		return fn(u, locked)
	}
	if tx, ok := dbtx.From(ctx); ok {
		guard, e := st.Guard(ctx, tx, c)
		if e != nil {
			return e
		}
		var now time.Time
		if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
			return e
		}
		return run(&unit{store: st, tx: tx, guard: guard, credentials: c, now: now, lockRemaining: 2 * time.Second}, true)
	}
	return st.within(ctx, c, write, !write, func(u *unit) error { return run(u, write) })
}
func (st *Store) Policies(ctx context.Context, c f.Credentials, version string) (v f.DatedRules[s.AssignmentPolicy], err error) {
	err = st.policy(ctx, c, "read", false, func(u *unit, _ bool) error {
		hist, e := access.RuleAllowed(ctx, u.guard, u.enterprise.ID, "assignmentpolicy.history.read")
		if e != nil {
			return e
		}
		if version != "" && !hist {
			return f.Fail(403, "ACCESS_DENIED", "无历史读取权限")
		}
		write, e := access.RuleAllowed(ctx, u.guard, u.enterprise.ID, "assignmentpolicy.maintain")
		if e != nil {
			return e
		}
		v, e = u.policyStore().View(ctx, u.enterprise.Name, version, hist, write)
		return e
	})
	return
}
func (st *Store) PublishPolicy(ctx context.Context, c f.Credentials, submission string, in f.RuleChange[s.AssignmentPolicy]) (v f.RuleReceipt, err error) {
	err = st.policy(ctx, c, "maintain", true, func(u *unit, _ bool) error {
		var e error
		v, e = u.policyStore().Save(ctx, u.guard.Principal().ID, submission, in, func(rows []s.AssignmentPolicy) ([]s.AssignmentPolicy, bool, error) {
			return app.ChangePolicy(rows, in, u.enterprise.ValidFrom)
		}, func(r s.AssignmentPolicy) (string, string, f.Period) { return r.ID, r.Revision, r.Period })
		return e
	})
	if err == nil && v.Failure != nil {
		err = v.Failure
	}
	return
}
func (st *Store) ResolvePolicy(ctx context.Context, c f.Credentials, submission string) (v f.RuleReceipt, err error) {
	err = st.policy(ctx, c, "read", true, func(u *unit, _ bool) error {
		var e error
		v, e = u.policyStore().Resolve(ctx, u.guard.Principal().ID, submission, nil)
		return e
	})
	return
}

func (st *Store) CheckAssignment(ctx context.Context, c f.Credentials, in s.AssignmentCheck) (out []s.AssignmentSegment, err error) {
	return st.checkAssignment(ctx, c, in, nil)
}
func (st *Store) checkAssignment(ctx context.Context, c f.Credentials, in s.AssignmentCheck, retained []f.Period) (out []s.AssignmentSegment, err error) {
	err = st.policy(ctx, c, "check", false, func(u *unit, locked bool) error {
		if in.Enterprise != u.enterprise.ID {
			return f.Fail(403, "ACCESS_DENIED", "企业不可用")
		}
		repo := u.policyStore()
		head, e := repo.Head(ctx)
		if e != nil {
			return e
		}
		if in.Expected != "" && in.Expected != head {
			return f.Fail(409, "M03_SOURCE_CHANGED", "任职政策已更新")
		}
		rows, e := repo.Load(ctx, head)
		if e != nil {
			return e
		}
		if in.Position != "" {
			out, e = u.checkPositionAssignment(ctx, in, head, rows, locked, retained)
		} else {
			out, e = app.CheckJobAssignment(ctx, u, in, head, rows, locked)
		}
		if e == nil && in.Adopted != nil && !slices.Equal(in.Adopted, out) {
			return f.Fail(409, "M03_SOURCE_CHANGED", "部门、职务或任职政策来源已变化，请重新核对")
		}
		return e
	})
	if err != nil {
		return nil, err
	}
	return
}
func (st *Store) PublishedQuotaPolicies(ctx context.Context, intent d.UsageIntent) ([]d.QuotaPolicy, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("assignment policy requires caller transaction")
	}
	u := &unit{store: st, tx: tx}
	u.enterprise.ID = intent.Enterprise
	if e := u.policyLock(ctx, true); e != nil {
		return nil, e
	}
	repo := u.policyStore()
	head, e := repo.Head(ctx)
	if e != nil {
		return nil, e
	}
	rows, e := repo.Load(ctx, head)
	if e != nil {
		return nil, e
	}
	return app.QuotaPolicies(intent.Enterprise, rows), nil
}
