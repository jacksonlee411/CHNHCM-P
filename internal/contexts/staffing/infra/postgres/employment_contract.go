package postgres

import (
	"context"
	"errors"
	"strings"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

func (u *unit) contractScope(ctx context.Context, id, day string, manage bool) (h.Action, error) {
	a, e := u.assignmentAt(ctx, id, day, "")
	if e != nil {
		return a, e
	}
	people, _, e := u.store.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), u.credentials, []string{a.Person}, "")
	if e != nil {
		return a, e
	}
	if len(people) != 1 {
		return a, unknownAssignment()
	}
	if e = u.requireEmployment(ctx, "contract.read", a.Person, a.Employer, a.Department); e != nil {
		return a, e
	}
	if manage {
		e = u.requireEmployment(ctx, "contract.manage", a.Person, a.Employer, a.Department)
	}
	return a, e
}

func (u *unit) contractRows(ctx context.Context, employment string) ([]h.Contract, error) {
	rows, e := u.tx.Query(ctx, `SELECT c.id,coalesce(ended.version,origin.version),c.employment_id,c.number,c.signed::text,
      lower(c.period)::text,coalesce(upper(c.period)::text,''),coalesce(c.predecessor::text,''),coalesce(next.id::text,''),coalesce(ended.business_date::text,''),
      (SELECT jsonb_agg(jsonb_strip_nulls(jsonb_build_object('version',p.version,'operation',p.operation,'businessDate',p.business_date,'operator',p.operator_name,'recordedAt',p.recorded_at,'basis',p.basis,'leaveAction',p.leave_action)) ORDER BY (p.operation='end')) FROM staffing.contract_process p WHERE (p.instance_id,p.contract_id)=(c.instance_id,c.id))
      FROM staffing.employment_contract c JOIN staffing.contract_process origin ON (c.instance_id,c.id)=(origin.instance_id,origin.contract_id) AND origin.operation IN ('sign','renew')
      LEFT JOIN staffing.contract_process ended ON (c.instance_id,c.id)=(ended.instance_id,ended.contract_id) AND ended.operation='end'
      LEFT JOIN staffing.employment_contract next ON (next.instance_id,next.predecessor)=(c.instance_id,c.id)
      WHERE c.instance_id=$1 AND c.employment_id=$2 ORDER BY lower(c.period),c.number`, u.store.Instance, employment)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (v h.Contract, e error) {
		e = r.Scan(&v.ID, &v.Version, &v.Employment, &v.Number, &v.Signed, &v.From, &v.Until, &v.Previous, &v.Next, &v.End, &v.Processes)
		return
	})
}
func (st *Store) Contracts(ctx context.Context, c f.Credentials, id, day, exitDate, meaning string) (out h.ContractList, err error) {
	err = st.within(ctx, c, false, true, func(u *unit) error {
		if day != "" && !f.ValidDate(day) {
			return f.Fail(400, "CONTRACT_DATE", "查看日期无效")
		}
		a, e := u.contractScope(ctx, id, day, false)
		if e != nil {
			return e
		}
		out.CanManage, e = u.employmentAllowed(ctx, "contract.manage", a.Person, a.Employer, a.Department, false)
		if e != nil {
			return e
		}
		out.Items, e = u.contractRows(ctx, a.Employment)
		if e != nil || exitDate == "" {
			return e
		}
		exit, e := (h.ExitCommand{Date: exitDate, Meaning: meaning}).Boundary()
		if e != nil {
			return e
		}
		for _, contract := range out.Items {
			if contract.RemainsAt(exit.FirstInactive) {
				out.Required = append(out.Required, contract.ContractRef)
			}
		}
		return nil
	})
	return
}
func (u *unit) contractProcess(ctx context.Context, id, op, day, basis, leave string) (string, error) {
	version := f.NewID()
	owner := u.guard.Principal()
	if _, e := u.tx.Exec(ctx, `INSERT INTO staffing.contract_process(instance_id,contract_id,version,operation,business_date,actor,operator_name,basis,leave_action) VALUES($1,$2,$3,$4,$5::date,$6,$7,$8,nullif($9,'')::uuid)`, u.store.Instance, id, version, op, day, owner.ID, owner.Login, basis, leave); e != nil {
		return "", e
	}
	_, e := u.tx.Exec(ctx, `INSERT INTO staffing.contract_outbox VALUES($1,$2,$3)`, u.store.Instance, id, version)
	return version, e
}
func (u *unit) contractReceipt(ctx context.Context, key string) (v h.ContractReceipt, digest string, found bool, err error) {
	v.SubmissionID = key
	var anchor string
	err = u.tx.QueryRow(ctx, `SELECT digest,coalesce(assignment_id::text,''),coalesce(contract_id::text,''),coalesce(version::text,'') FROM staffing.contract_submission WHERE instance_id=$1 AND actor=$2 AND submission_id=$3`, u.store.Instance, u.guard.Principal().ID, key).Scan(&digest, &anchor, &v.ID, &v.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, "", false, nil
	}
	if err != nil {
		return
	}
	found = true
	v.Outcome = "not_applied"
	if v.ID == "" {
		return
	}
	v.Outcome = "accepted"
	_, err = u.contractScope(ctx, anchor, "", false)
	if err != nil {
		var fault *f.Fault
		if errors.As(err, &fault) && (fault.Status == 403 || fault.Status == 404) {
			v.ID, v.Version, err = "", "", nil
		}
	}
	return
}
func (st *Store) ResolveContract(ctx context.Context, c f.Credentials, key string) (out h.ContractReceipt, err error) {
	err = st.within(ctx, c, true, false, func(u *unit) error {
		if !f.ValidID(key) {
			return f.Fail(400, "SUBMISSION_INPUT", "提交号无效")
		}
		if e := lock(ctx, u.tx, false, "staffing.contract.submission", st.Instance, u.guard.Principal().ID, key); e != nil {
			return e
		}
		var found bool
		var e error
		out, _, found, e = u.contractReceipt(ctx, key)
		if found || e != nil {
			return e
		}
		_, e = u.tx.Exec(ctx, `INSERT INTO staffing.contract_submission(instance_id,actor,submission_id,digest) VALUES($1,$2,$3,'')`, st.Instance, u.guard.Principal().ID, key)
		out.Outcome = "not_applied"
		return e
	})
	return
}
func (st *Store) RecordContract(ctx context.Context, c f.Credentials, id, key string, in h.ContractCommand) (out h.ContractReceipt, err error) {
	err = st.within(ctx, c, true, false, func(u *unit) error {
		if !f.ValidID(key) || !f.ValidID(id) {
			return f.Fail(400, "SUBMISSION_INPUT", "提交号或任职无效")
		}
		if e := in.Validate(); e != nil {
			return e
		}
		if e := lock(ctx, u.tx, false, "staffing.contract.submission", st.Instance, u.guard.Principal().ID, key); e != nil {
			return e
		}
		digest, e := hireDigest([]any{id, in})
		if e != nil {
			return e
		}
		var saved string
		var found bool
		out, saved, found, e = u.contractReceipt(ctx, key)
		if e != nil {
			return e
		}
		if found {
			if saved != "" && digest != saved {
				return assignmentConflict()
			}
			return nil
		}
		a, e := u.assignmentIdentity(ctx, id)
		if e != nil {
			return e
		}
		if e = lock(ctx, u.tx, false, "staffing.hire.person", st.Instance, a.Person); e != nil {
			return e
		}
		day := in.Signed
		if in.Operation == "end" {
			day = in.End
		}
		a, e = u.contractScope(ctx, id, max(a.From, day), true)
		if e != nil {
			return e
		}
		if a.SourceVersion != in.Expected || a.Cancelled {
			return assignmentConflict()
		}
		contracts, e := u.contractRows(ctx, a.Employment)
		if e != nil {
			return e
		}
		var prior h.Contract
		for _, row := range contracts {
			if in.Ref != nil && row.ContractRef == *in.Ref {
				prior = row
			}
			if in.Operation != "end" && row.Number == strings.TrimSpace(in.Number) {
				return f.Fail(409, "CONTRACT_NUMBER", "本关系已有同编号合同")
			}
		}
		if in.Ref != nil && prior.ID == "" {
			return f.Fail(409, "CONTRACT_SOURCE", "原合同已变化或不属于本关系")
		}
		var boundary string
		if e = u.tx.QueryRow(ctx, `SELECT coalesce(x.first_inactive::text,upper(r.period)::text,'') FROM staffing.employment r LEFT JOIN staffing.employment_exit x ON (x.instance_id,x.employment_id)=(r.instance_id,r.id) WHERE r.instance_id=$1 AND r.id=$2`, st.Instance, a.Employment).Scan(&boundary); e != nil {
			return e
		}
		contract := prior.ID
		basis := strings.TrimSpace(in.Basis)
		if in.Operation == "end" {
			if prior.End != "" || prior.Until != "" && in.End >= prior.Until {
				return f.Fail(409, "CONTRACT_ENDED", "原合同在所选日期已经终结")
			}
		} else {
			term, e := hireDates(in.From, in.LastDay)
			if e != nil {
				return e
			}
			if boundary != "" && (term.Until == "" || term.Until > boundary) {
				return f.Fail(409, "CONTRACT_EXIT", "原工作关系已确定结束，请核对合同期限")
			}
			if in.Operation == "renew" {
				until := prior.Until
				if prior.End != "" {
					until = prior.End
				}
				if prior.Next != "" || until == "" || in.From < until || in.From <= prior.From {
					return f.Fail(409, "CONTRACT_RENEW", "请核对原合同结束边界及已签订的后继")
				}
			}
			contract = f.NewID()
			basis = strings.TrimSpace(in.Number)
			_, e = u.tx.Exec(ctx, `INSERT INTO staffing.employment_contract VALUES($1,$2,$3,$4,$5::date,daterange($6::date,nullif($7,'')::date,'[)'),nullif($8,'')::uuid)`, st.Instance, contract, a.Employment, basis, in.Signed, term.From, term.Until, prior.ID)
			if e != nil {
				return e
			}
		}
		version, e := u.contractProcess(ctx, contract, in.Operation, day, basis, "")
		if e != nil {
			return e
		}
		_, e = u.tx.Exec(ctx, `INSERT INTO staffing.contract_submission(instance_id,actor,submission_id,digest,assignment_id,contract_id,version) VALUES($1,$2,$3,$4,$5,$6,$7)`, st.Instance, u.guard.Principal().ID, key, digest, id, contract, version)
		out = h.ContractReceipt{SubmissionID: key, Outcome: "accepted", ID: contract, Version: version}
		return e
	})
	return
}

func (u *unit) closeContracts(ctx context.Context, a h.Action, at, basis, leave string, selected []h.ContractRef) error {
	rows, e := u.contractRows(ctx, a.Employment)
	if e != nil {
		return e
	}
	remaining := map[h.ContractRef]bool{}
	for _, r := range rows {
		if r.RemainsAt(at) {
			if r.End != "" {
				return f.Fail(409, "CONTRACT_EXIT", "工作关系结束后仍有合同终结安排，请先核对原办理")
			}
			remaining[r.ContractRef] = true
		}
	}
	if len(remaining) != len(selected) {
		return f.Fail(409, "CONTRACT_EXIT", "请明确处置这段工作关系的待结束合同")
	}
	if len(selected) > 0 {
		for _, grant := range []string{"contract.read", "contract.manage"} {
			if e = u.requireEmployment(ctx, grant, a.Person, a.Employer, a.Department); e != nil {
				return e
			}
		}
	}
	for _, ref := range selected {
		if !remaining[ref] {
			return f.Fail(409, "CONTRACT_SOURCE", "所选合同已变化或不属于本关系")
		}
		delete(remaining, ref)
		if _, e = u.contractProcess(ctx, ref.ID, "end", at, basis, leave); e != nil {
			return e
		}
	}
	return nil
}
