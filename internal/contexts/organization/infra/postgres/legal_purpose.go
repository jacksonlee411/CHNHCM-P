package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/legalentity"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/legalentity"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"chnhcm.local/chnhcm/internal/platform/rulesource"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"reflect"
	"slices"
)

var purposeSQL = rulesource.Statements{
	Head:         `SELECT version::text FROM organization.legal_purpose_source WHERE instance_id=$1 AND owner_id=$2 ORDER BY sequence DESC LIMIT 1`,
	SourceExists: `SELECT EXISTS(SELECT FROM organization.legal_purpose_source WHERE instance_id=$1 AND owner_id=$2 AND version=$3)`,
	Members:      `SELECT fact FROM organization.legal_purpose_member WHERE instance_id=$1 AND owner_id=$2 AND version=$3 ORDER BY lower(period),record_id`,
	History:      `SELECT version::text,recorded_at FROM organization.legal_purpose_source WHERE instance_id=$1 AND owner_id=$2 ORDER BY sequence DESC`,
	Replay: `SELECT owner_id::text, 'accepted', version::text, input_digest, 0, '', ''
		FROM organization.legal_purpose_source WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3
		UNION ALL SELECT coalesce(owner_id::text,''), outcome, coalesce(version::text,''), input_digest, coalesce(failure_status,0), coalesce(failure_code,''), coalesce(failure_message,'')
		FROM organization.legal_purpose_terminal WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3`,
	Terminal: `INSERT INTO organization.legal_purpose_terminal
		(instance_id,subject_id,submission_id,owner_id,outcome,version,input_digest,failure_status,failure_code,failure_message)
		VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10)`,
	InsertSource: `INSERT INTO organization.legal_purpose_source(instance_id,owner_id,version,subject_id,submission_id,input_digest) VALUES($1,$2,$3,$4,$5,$6)`,
	InsertMember: `INSERT INTO organization.legal_purpose_member(instance_id,owner_id,version,record_id,revision,period,fact) VALUES($1,$2,$3,$4,$5,daterange($6::date,NULLIF($7,'')::date,'[)'),$8)`,
	Publish:      `INSERT INTO organization.legal_purpose_outbox(instance_id,owner_id,version) VALUES($1,$2,$3)`,
}

type purposeUnit struct {
	*legalentityUnit
	rulesource.Store[org.LegalPurpose]
	entity app.State
}

func requirePurposeEntityRead(ctx context.Context, u *unit, enterprise string) error {
	grant, err := u.guard.Decide(ctx, access.Intent{Kind: "legalentity", EnterpriseID: enterprise, Action: "legalentity.read"})
	if err != nil {
		return err
	}
	if !grant.Allowed || !slices.Contains(grant.ReadFields, "name") {
		return f.Fail(403, "ACCESS_DENIED", "法律实体不可读")
	}
	return nil
}

func (s *Store) purpose(ctx context.Context, c f.Credentials, id, action string, write bool, fn func(*purposeUnit) error) error {
	if !f.ValidID(id) {
		return f.Fail(400, "M02_INVALID_INPUT", "法律实体标识无效")
	}
	run := func(u *unit, locked bool) error {
		ent, err := u.Enterprise(ctx, locked)
		if err != nil {
			return err
		}
		if err = access.RequireRule(ctx, u.guard, ent.ID, "legalpurpose."+action); err != nil {
			return err
		}
		if err = requirePurposeEntityRead(ctx, u, ent.ID); err != nil {
			return err
		}
		lu := &legalentityUnit{unit: u, enterprise: ent}
		if locked {
			if err = lu.lockEntity(ctx, id, write); err != nil {
				return err
			}
		}
		state, err := lu.Load(ctx, id, "")
		if err != nil {
			return err
		}
		if state.Timeline.ID == "" {
			return f.Fail(404, "M02_OBJECT_NOT_FOUND", "法律实体不存在或当前不可见")
		}
		err = fn(&purposeUnit{legalentityUnit: lu, Store: rulesource.Store[org.LegalPurpose]{SQL: purposeSQL, Tx: u.tx, Instance: u.instance, Owner: id}, entity: state})
		if err == nil && write && u.scope != nil {
			u.Confirm()
		}
		return err
	}
	return s.withPurposeUnit(ctx, c, write, run)
}
func (s *Store) withPurposeUnit(ctx context.Context, c f.Credentials, write bool, run func(*unit, bool) error) error {
	if tx, ok := dbtx.From(ctx); ok {
		s.historyCatalog.Enter()
		defer s.historyCatalog.Leave()
		g, err := s.Guard(ctx, tx, c)
		if err != nil {
			return err
		}
		return run(&unit{tx: tx, instance: s.Instance, catalog: &s.historyCatalog, guard: g}, true)
	}
	return s.withUnit(ctx, c, write, func(u *unit) error { return run(u, write) })
}
func (s *Store) Purposes(ctx context.Context, c f.Credentials, id, version string) (v f.DatedRules[org.LegalPurpose], err error) {
	err = s.purpose(ctx, c, id, "read", false, func(u *purposeUnit) error {
		hist, e := access.RuleAllowed(ctx, u.guard, u.enterprise.ID, "legalpurpose.history.read")
		if e != nil {
			return e
		}
		if version != "" && !hist {
			return f.Fail(403, "ACCESS_DENIED", "无历史读取权限")
		}
		write, e := access.RuleAllowed(ctx, u.guard, u.enterprise.ID, "legalpurpose.maintain")
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		label := "法律实体"
		if r := d.Kernel.FactAt(u.entity.Timeline.Records, now.In(f.Shanghai).Format("2006-01-02")); r != nil {
			label = r.Data.Name
		}
		v, e = u.View(ctx, label, version, hist, write)
		return e
	})
	return
}
func activeRecords[T any](records []dated.Record[T]) []app.ActiveRecord {
	var out []app.ActiveRecord
	for _, r := range records {
		out = append(out, app.ActiveRecord{Period: f.Period{From: r.Date, Until: r.End}, Active: r.State == "active", Revision: r.Revision})
	}
	return out
}
func (u *purposeUnit) checkTarget(ctx context.Context, kind, id string, period f.Period, locked bool) error {
	intent := access.Intent{Kind: kind, EnterpriseID: u.enterprise.ID, Action: kind + ".read"}
	if kind == "bu" {
		intent.BUID = id
	}
	grant, err := u.guard.Decide(ctx, intent)
	if err != nil {
		return err
	}
	if !grant.Allowed || !slices.Contains(grant.ReadFields, "name") {
		return f.Fail(403, "ACCESS_DENIED", "绑定目标不可读")
	}
	if kind == "bu" {
		if locked {
			var target string
			e := u.tx.QueryRow(ctx, "SELECT bu_id::text FROM organization.business_unit WHERE instance_id=$1 AND enterprise_id=$2 AND bu_id=$3 FOR SHARE", u.instance, u.enterprise.ID, id).Scan(&target)
			if errors.Is(e, pgx.ErrNoRows) {
				return f.Fail(404, "M02_OBJECT_NOT_FOUND", "业务单元不可用")
			}
			if e != nil {
				return e
			}
		}
		t, e := u.LoadTimeline(ctx, id, false)
		if e != nil {
			return e
		}
		return app.CoverActive(period, activeRecords(t.Records), "业务单元")
	}
	if locked {
		var key string
		e := u.unit.tx.QueryRow(ctx, "SELECT location_id::text FROM organization.location WHERE instance_id=$1 AND enterprise_id=$2 AND location_id=$3 FOR SHARE", u.instance, u.enterprise.ID, id).Scan(&key)
		if errors.Is(e, pgx.ErrNoRows) {
			return f.Fail(404, "M02_OBJECT_NOT_FOUND", "地点不存在或当前不可见")
		}
		if e != nil {
			return e
		}
	}
	t, e := (&locationUnit{unit: u.unit, enterprise: u.enterprise}).Load(ctx, id, "")
	if e != nil {
		return e
	}
	return app.CoverActive(period, activeRecords(t.Timeline.Records), "地点")
}
func (s *Store) DeclarePurpose(ctx context.Context, c f.Credentials, id, submission string, in f.RuleChange[org.LegalPurpose]) (v f.RuleReceipt, err error) {
	err = s.purpose(ctx, c, id, "maintain", true, func(u *purposeUnit) error {
		var e error
		v, e = u.Store.Save(ctx, u.guard.Principal().ID, submission, in, func(rows []org.LegalPurpose) ([]org.LegalPurpose, bool, error) {
			next, changed, e := app.ChangePurpose(rows, in)
			if e != nil || !changed {
				return next, changed, e
			}
			if e = app.CoverActive(in.Row.Period, activeRecords(u.entity.Timeline.Records), "法律实体"); e != nil {
				return nil, false, e
			}
			if in.Row.Kind != "declaration" {
				e = u.checkTarget(ctx, in.Row.Kind, in.Row.Target, in.Row.Period, true)
			}
			if e == nil && s.EmploymentUses != nil {
				uses, err := s.EmploymentUses(dbtx.With(ctx, u.tx), u.instance, id)
				if err != nil {
					return nil, false, err
				}
				for _, use := range uses {
					check := org.PurposeCheck{Role: "employer", BU: use.BU, Location: use.Location, Period: f.Period{From: use.From, Until: use.Until}}
					if _, err = app.ResolvePurpose(check, u.entity.Timeline.Source, "", activeRecords(u.entity.Timeline.Records), next); err != nil {
						return nil, false, f.Fail(409, "M02_REFERENCE_CONFLICT", "拟议用途不能覆盖已办理的任职期间")
					}
				}
			}
			return next, changed, e
		}, func(r org.LegalPurpose) (string, string, f.Period) { return r.ID, r.Revision, r.Period })
		return e
	})
	if err == nil && v.Failure != nil {
		err = v.Failure
	}
	return
}

// Recovery is subject-wide. Only three opaque recovery references need to
// survive a reload. Check the original entity's scope before disclosing its result.
func (s *Store) ResolvePurpose(ctx context.Context, c f.Credentials, submission string) (v f.RuleReceipt, err error) {
	err = s.withPurposeUnit(ctx, c, true, func(u *unit, _ bool) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		if e = access.RequireRule(ctx, u.guard, ent.ID, "legalpurpose.read"); e != nil {
			return e
		}
		if e = requirePurposeEntityRead(ctx, u, ent.ID); e != nil {
			return e
		}
		repo := rulesource.Store[org.LegalPurpose]{SQL: purposeSQL, Tx: u.tx, Instance: u.instance}
		v, e = repo.Resolve(ctx, u.guard.Principal().ID, submission, func(owner string) error {
			var visible bool
			if e := u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM organization.legal_entity
				WHERE instance_id=$1 AND enterprise_id=$2 AND legal_entity_id=$3)`, u.instance, ent.ID, owner).Scan(&visible); e != nil {
				return e
			}
			if !visible {
				return f.Fail(404, "M02_OBJECT_NOT_FOUND", "法律实体不存在或当前不可见")
			}
			return nil
		})
		if e == nil && u.scope != nil {
			u.Confirm()
		}
		return e
	})
	return
}

func (s *Store) CheckPurpose(ctx context.Context, c f.Credentials, id string, in org.PurposeCheck) (out []org.LegalAdoption, err error) {
	if in.BU != "" && !f.ValidID(in.BU) || in.Location != "" && !f.ValidID(in.Location) {
		return nil, f.Fail(400, "M02_INVALID_INPUT", "绑定标识无效")
	}
	err = s.purpose(ctx, c, id, "check", false, func(u *purposeUnit) error {
		_, locked := dbtx.From(ctx)
		var e error
		out, e = u.resolvePurpose(ctx, in, locked)
		var fault *f.Fault
		if errors.As(e, &fault) && (fault.Status == 409 || fault.Status == 403 || fault.Status == 404) {
			allowed, authErr := access.RuleAllowed(ctx, u.guard, u.enterprise.ID, "legalpurpose.read")
			if authErr != nil {
				return authErr
			}
			if !allowed {
				return f.Fail(409, "M02_PURPOSE_UNAVAILABLE", "当前安排不满足办理条件")
			}
		}
		return e
	})
	if err != nil {
		return nil, err
	}
	return
}

func (u *purposeUnit) resolvePurpose(ctx context.Context, in org.PurposeCheck, locked bool) ([]org.LegalAdoption, error) {
	version, e := u.Head(ctx)
	if e != nil {
		return nil, e
	}
	rows, e := u.Store.Load(ctx, version)
	if e != nil {
		return nil, e
	}
	out, e := app.ResolvePurpose(in, u.entity.Timeline.Source, version, activeRecords(u.entity.Timeline.Records), rows)
	if e != nil {
		return nil, e
	}

	for index, p := range out {
		if record := d.Kernel.FactAt(u.entity.Timeline.Records, p.From); record != nil {
			out[index].EntityName = record.Data.Name
		}
		for _, r := range rows {
			if slices.Contains(p.Bindings, r.Revision) {
				if e = u.checkTarget(ctx, r.Kind, r.Target, p.Period, locked); e != nil {
					return nil, e
				}
			}
		}
	}
	if in.Adopted != nil && !reflect.DeepEqual(in.Adopted, out) {
		return nil, f.Fail(409, "M02_SOURCE_CHANGED", "主体、用途或绑定来源已变化，请重新核对")
	}
	return out, nil
}

func (s *Store) Candidates(ctx context.Context, c f.Credentials, in org.PurposeCandidates) (out []org.EligibleEntity, err error) {
	if !in.Period.Valid() || !app.ValidRole(in.Role) || in.Page < 1 || in.Page > 1000000 || len(in.Search) > 400 || in.BU != "" && !f.ValidID(in.BU) || in.Location != "" && !f.ValidID(in.Location) || (in.Expected != "" || in.Adopted != nil) {
		return nil, f.Fail(400, "M02_INVALID_INPUT", "用途、拟任期间、上下文或页码无效")
	}
	out = []org.EligibleEntity{}
	err = s.withPurposeUnit(ctx, c, false, func(u *unit, _ bool) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		if e = access.RequireRule(ctx, u.guard, ent.ID, "legalpurpose.check"); e != nil {
			return e
		}
		if e = requirePurposeEntityRead(ctx, u, ent.ID); e != nil {
			return e
		}
		lu := &legalentityUnit{unit: u, enterprise: ent}
		// Prefilter at the start day, then apply the same full-period rule before paging.
		cursor := ""
		skip := (in.Page - 1) * 20
		for {
			rows, e := u.tx.Query(ctx, `SELECT object_id::text,name FROM organization.legal_entity_read r
    WHERE instance_id=$1 AND enterprise_id=$2 AND valid_period @> $3::date AND state='active'
    AND strpos(lower(name),lower($4))>0 AND object_id>COALESCE(NULLIF($5,'')::uuid,'00000000-0000-0000-0000-000000000000')
    AND EXISTS(SELECT FROM organization.legal_purpose_source s WHERE s.instance_id=r.instance_id AND s.owner_id=r.object_id)
    ORDER BY object_id LIMIT 64`, u.instance, ent.ID, in.From, in.Search, cursor)
			if e != nil {
				return e
			}
			candidates, e := pgx.CollectRows(rows, pgx.RowToStructByPos[org.EligibleEntity])
			if e != nil {
				return e
			}
			for _, v := range candidates {
				cursor = v.ID
				state, e := lu.Load(ctx, v.ID, "")
				if e != nil {
					return e
				}
				pu := purposeUnit{legalentityUnit: lu, entity: state, Store: rulesource.Store[org.LegalPurpose]{SQL: purposeSQL, Tx: u.tx, Instance: u.instance, Owner: v.ID}}
				if _, e = pu.resolvePurpose(ctx, in.PurposeCheck, false); e != nil {
					var fault *f.Fault
					if errors.As(e, &fault) && (fault.Status == 409 || fault.Status == 403 || fault.Status == 404) {
						continue
					}
					return e
				}
				if skip > 0 {
					skip--
					continue
				}
				out = append(out, v)
				if len(out) == 20 {
					return nil
				}
			}
			if len(candidates) < 64 {
				return nil
			}
		}
	})
	return
}

// Check current declarations and context bindings under the row guard shared
// with DeclarePurpose. Historical source memberships never block a change.
func protectLegalPurposes[T any](ctx context.Context, u *unit, kind, id string, before, after []dated.Record[T]) error {
	if len(before) == 0 {
		return nil
	}
	lost, e := dated.RemovedCoverage(before, after)
	if e != nil || len(lost) == 0 {
		return e
	}
	for _, span := range lost {
		var first string
		e = u.tx.QueryRow(ctx, `WITH heads AS (
     SELECT DISTINCT ON(owner_id) owner_id,version FROM organization.legal_purpose_source
     WHERE instance_id=$1 AND ($3<>'entity' OR owner_id=$2::uuid) ORDER BY owner_id,sequence DESC
   ) SELECT coalesce(min(greatest(lower(m.period),$4::date))::text,'') FROM heads h
   JOIN organization.legal_purpose_member m ON m.instance_id=$1 AND (m.owner_id,m.version)=(h.owner_id,h.version)
   WHERE ($3='entity' OR (m.fact->>'kind'=$3 AND m.fact->>'target'=$2::text))
   AND m.period && daterange($4::date,NULLIF($5,'')::date,'[)')`, u.instance, id, kind, span.Start, span.End).Scan(&first)
		if e != nil {
			return e
		}
		if first != "" {
			return f.Fail(409, "M02_REFERENCE_CONFLICT", first+" 起仍有用途或绑定，请先调整相应期间")
		}
	}
	return nil
}
