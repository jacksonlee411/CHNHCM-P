package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"

	app "chnhcm.local/chnhcm/internal/contexts/compensation/application"
	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	cfg "chnhcm.local/chnhcm/internal/contexts/configuration/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) loadPlan(ctx context.Context, u *unit, id string, lock bool) (state app.PlanState, err error) {
	if !foundation.ValidID(id) {
		return state, unavailable()
	}
	query := "SELECT id::text,enterprise_id::text,code,head::text FROM compensation.plan WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3"
	if lock {
		query += " FOR UPDATE"
	}
	err = u.tx.QueryRow(ctx, query, s.Instance, u.enterprise.ID, id).Scan(&state.ID, &state.Enterprise, &state.Code, &state.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, unavailable()
	}
	if err != nil {
		return
	}
	rows, err := u.tx.Query(ctx, `SELECT r.fact FROM compensation.plan_period p JOIN compensation.plan_revision r USING(instance_id,plan_id,record_id,revision) WHERE p.instance_id=$1 AND p.plan_id=$2 ORDER BY p.start_date`, s.Instance, id)
	if err != nil {
		return state, err
	}
	defer rows.Close()
	for rows.Next() {
		var r d.Record[d.PlanTerms]
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return state, err
		}
		if err = json.Unmarshal(raw, &r); err != nil {
			return state, err
		}
		state.Records = append(state.Records, r)
	}
	if err = rows.Err(); err == nil && len(state.Records) == 0 {
		err = errors.New("missing compensation plan periods")
	}
	return
}
func (s *Store) SavePlan(ctx context.Context, cred foundation.Credentials, id, submission string, in c.WritePlan) (c.Receipt, error) {
	fields, err := app.PlanFields(in)
	if err != nil {
		return c.Receipt{}, err
	}
	return saveCommand(ctx, s, cred, planCommands, id, submission, in.Action, fields, in, s.applyPlan)
}

type changedPlan struct {
	Records              d.Timeline[d.PlanTerms]
	Target               d.Record[d.PlanTerms]
	Previous             *d.Record[d.PlanTerms]
	Until, PreviousUntil string
	Changed              bool
}

func (s *Store) applyPlan(ctx context.Context, u *unit, id string, in c.WritePlan, explicit []string) (out c.Receipt, err error) {
	if slices.Contains(explicit, "rates") {
		if err = u.requirePlanRateReference(ctx); err != nil {
			return out, err
		}
	}
	var state app.PlanState
	if id == "" {
		code, ok := d.NormalizeCode(in.Code)
		if !ok {
			return out, d.Invalid("code", "请输入有效的计划编码")
		}
		state = app.PlanState{ID: foundation.NewID(), Enterprise: u.enterprise.ID, Code: code}
	} else {
		state, err = s.loadPlan(ctx, u, id, true)
		if err != nil {
			return
		}
	}
	if state.Version != in.ExpectedVersion {
		return out, d.Conflict()
	}
	if in.EffectiveDate != "" && in.EffectiveDate < u.enterprise.ValidFrom {
		return out, d.Invalid("effectiveDate", "生效日期早于企业起始日期")
	}
	change, err := s.preparePlan(ctx, u, state, in, explicit)
	if err != nil {
		return out, err
	}
	out = c.Receipt{ObjectID: state.ID, Version: state.Version, RecordID: change.Target.ID, Revision: change.Target.Revision, Outcome: "no_change"}
	if !change.Changed {
		return out, nil
	}
	out.Version = foundation.NewID()
	out.Outcome = "accepted"
	err = s.persistPlan(ctx, u, state, change, in.Action, out.Version)
	return
}
func (s *Store) preparePlan(ctx context.Context, u *unit, state app.PlanState, in c.WritePlan, explicit []string) (change changedPlan, err error) {
	prior := state.Records
	at := slices.IndexFunc(prior, func(r d.Record[d.PlanTerms]) bool { return r.ID == in.RecordID })
	if at >= 0 {
		change.Previous = &prior[at]
		change.PreviousUntil = prior.Period(at).Until
	}
	originalOp := in.Action
	if in.Action == "CORRECT" && at >= 0 {
		originalOp = prior[at].Operation
		if !prior[at].Active && len(in.Changes) > 0 {
			return change, d.Invalid("changes", "停用记录沿用前段计划内容")
		}
	}
	if in.Reason != "" && !slices.Contains(d.PlanReasons(originalOp), strings.TrimSpace(in.Reason)) {
		return change, d.Invalid("reason", "请选择该操作的办理原因")
	}
	var incoming app.IncomingPlan
	edit := d.Edit{Action: in.Action, Date: in.EffectiveDate, Reason: strings.TrimSpace(in.Reason), RecordID: in.RecordID, ExpectedRevision: in.ExpectedRevision}
	records, target, err := d.Amend(prior, edit, func(base d.PlanTerms) (d.PlanTerms, error) {
		next, e := app.ReadPlanInput(base, in.Changes)
		if len(next.ExtraFields) > 0 {
			if _, authErr := u.objectWriteGrant(ctx, planCommands, in.Action, append(slices.Clone(explicit), next.ExtraFields...)); authErr != nil {
				return base, authErr
			}
		}
		incoming = next
		return next.Terms, e
	})
	if err != nil {
		return change, err
	}
	var proposed *cfg.Scope
	if incoming.Scope != nil {
		spec, e := s.Scopes.Normalize(*incoming.Scope)
		if e != nil {
			return change, e
		}
		proposed = &spec
		if target.Data.ScopeVersion != "" {
			old, e := s.Scopes.ReadPlanScope(ctx, u.enterprise.ID, state.ID, target.Data.ScopeVersion)
			if e != nil {
				return change, e
			}
			if current := policyChoices(old); spec.Mode == current.Mode && slices.Equal(spec.Choices, current.Choices) {
				proposed = nil
			} else {
				for _, member := range old.Members {
					grant, e := u.require(ctx, sourcePermission(member.Dimension), nil)
					if e != nil {
						return change, e
					}
					if !has(grant.ReadFields, "name") {
						return change, denied()
					}
				}
			}
		}
	}
	if at >= 0 && proposed == nil && reflect.DeepEqual(target.Data, prior[at].Data) && (in.Action == "ADD" || in.Action == "CORRECT" && target.Start == prior[at].Start && target.Reason == prior[at].Reason) {
		return changedPlan{Records: prior, Target: prior[at]}, nil
	}
	oldIndex := map[string]int{}
	for i, r := range prior {
		oldIndex[r.ID] = i
	}
	// Lock the old as well as the new rate references. Removing the last use
	// serializes with a rate deactivation under the same lock as a new use.
	ids := append(planRateIDs(prior), planRateIDs(records)...)
	slices.Sort(ids)
	rates := map[string]app.RateState{}
	for _, rid := range slices.Compact(ids) {
		r, e := s.loadRate(ctx, u, rid, true)
		if e != nil {
			if f, ok := fault(e); ok && f.Status == 404 {
				return change, d.Invalid("components", "所选职级标准不可用")
			}
			return change, e
		}
		rates[rid] = r
	}
	for i := range records {
		r := &records[i]
		p := records.Period(i)
		if r.ID == target.ID {
			change.Until = p.Until
		}
		if !r.Active {
			if i > 0 && !reflect.DeepEqual(r.Data, records[i-1].Data) {
				r.Data = records[i-1].Data
				r.Revision = foundation.NewID()
			}
			continue
		}
		old, existed := oldIndex[r.ID]
		isTarget := r.ID == target.ID
		extended := false
		if existed {
			o := prior.Period(old)
			extended = p.From < o.From || o.Until != "" && (p.Until == "" || p.Until > o.Until)
		}
		adopt := isTarget && (!existed || r.Start != prior[old].Start || proposed != nil || !maps.Equal(componentRates(r.Data.Components), componentRates(prior[old].Data.Components)))
		if !adopt && !extended {
			continue
		}
		var spec cfg.Scope
		if isTarget && proposed != nil {
			spec = *proposed
		} else {
			policy, e := s.Scopes.ReadPlanScope(ctx, u.enterprise.ID, state.ID, r.Data.ScopeVersion)
			if e != nil {
				return change, e
			}
			spec = policyChoices(policy)
		}
		members, e := s.scopeMembers(ctx, u, spec, p)
		if e != nil {
			return change, e
		}
		if isTarget && proposed != nil {
			policy, e := s.Scopes.SavePlanScope(ctx, u.enterprise.ID, state.ID, spec, p, members)
			if e != nil {
				return change, e
			}
			r.Data.ScopeVersion = policy.Version
		}
		r.Data.Components = cloneComponents(r.Data.Components)
		if e := s.verifyComponents(ctx, u, spec, r.Data.Components, p, rates, adopt); e != nil {
			return change, e
		}
	}
	if in.Action != "DELETE" {
		for _, r := range records {
			if r.ID == target.ID {
				target = r
				break
			}
		}
	}
	change.Changed, change.Records, change.Target = true, records, target
	return change, nil
}
func (s *Store) persistPlan(ctx context.Context, u *unit, state app.PlanState, change changedPlan, action, version string) error {
	op, err := u.guard.OperatorSnapshot(ctx)
	if err != nil {
		return err
	}
	actor, err := json.Marshal(op)
	if err != nil {
		return err
	}
	if state.Version == "" {
		_, err = u.tx.Exec(ctx, "INSERT INTO compensation.plan(instance_id,enterprise_id,id,code,head) VALUES($1,$2,$3,$4,$5)", s.Instance, u.enterprise.ID, state.ID, state.Code, version)
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return &foundation.Fault{Status: 409, Code: "M09_DUPLICATE_PLAN", Field: "code", Message: "此计划编码已使用"}
		}
		if err != nil {
			return err
		}
	}
	old := map[string]int{}
	for i, r := range state.Records {
		old[r.ID] = i
	}
	for i, r := range change.Records {
		ix, exists := old[r.ID]
		newFact := !exists || state.Records[ix].Revision != r.Revision
		p := change.Records.Period(i)
		if newFact {
			raw, e := json.Marshal(r)
			if e != nil {
				return e
			}
			if _, e = u.tx.Exec(ctx, `INSERT INTO compensation.plan_revision(instance_id,plan_id,record_id,revision,fact) VALUES($1,$2,$3,$4,$5)`, s.Instance, state.ID, r.ID, r.Revision, raw); e != nil {
				return e
			}
		}
		if !newFact && p == state.Records.Period(ix) {
			continue
		}
		_, err = u.tx.Exec(ctx, `INSERT INTO compensation.plan_period(instance_id,plan_id,record_id,revision,name,start_date,until_date,active) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(instance_id,plan_id,record_id) DO UPDATE SET revision=excluded.revision,name=excluded.name,start_date=excluded.start_date,until_date=excluded.until_date,active=excluded.active`, s.Instance, state.ID, r.ID, r.Revision, r.Data.Name, p.From, optional(p.Until), r.Active)
		if err != nil {
			return err
		}
		if _, err = u.tx.Exec(ctx, "DELETE FROM compensation.plan_rate_use WHERE instance_id=$1 AND plan_id=$2 AND record_id=$3", s.Instance, state.ID, r.ID); err != nil {
			return err
		}
		if r.Active {
			for _, rateID := range planRateIDs(d.Timeline[d.PlanTerms]{r}) {
				if _, err = u.tx.Exec(ctx, `INSERT INTO compensation.plan_rate_use(instance_id,rate_id,plan_id,record_id,valid_period) VALUES($1,$2,$3,$4,daterange($5,$6,'[)'))`, s.Instance, rateID, state.ID, r.ID, p.From, optional(p.Until)); err != nil {
					return err
				}
			}
		}
	}
	if action == "DELETE" {
		if _, err = u.tx.Exec(ctx, "DELETE FROM compensation.plan_period WHERE instance_id=$1 AND plan_id=$2 AND record_id=$3", s.Instance, state.ID, change.Target.ID); err != nil {
			return err
		}
	}
	var prevID, prevRev any
	if change.Previous != nil {
		prevID, prevRev = change.Previous.ID, change.Previous.Revision
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO compensation.plan_version(instance_id,plan_id,id,predecessor,action,record_id,revision,previous_record_id,previous_revision,until_date,previous_until,operator) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, s.Instance, state.ID, version, optional(state.Version), action, change.Target.ID, change.Target.Revision, prevID, prevRev, optional(change.Until), optional(change.PreviousUntil), actor)
	if err != nil {
		return err
	}
	if state.Version != "" {
		if _, err = u.tx.Exec(ctx, "UPDATE compensation.plan SET head=$3 WHERE instance_id=$1 AND id=$2", s.Instance, state.ID, version); err != nil {
			return err
		}
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO compensation.plan_outbox VALUES($1,$2,$3,jsonb_build_object('type','CompensationPlanChanged','sourceVersion',$3::uuid,'recordId',$4::uuid))`, s.Instance, state.ID, version, change.Target.ID)
	return err
}
