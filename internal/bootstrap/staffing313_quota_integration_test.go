//go:build integration

package bootstrap_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	staffdb "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

// The pilot source adapters and their tables exist only on the disposable DB.
// They share the real staffing transaction guard with the normal administrative API.
type quotaPilot struct {
	f          *fixture
	store      *staffdb.Store
	enterprise string
	category   string
}

func readPilotUsage(ctx context.Context, instance, position string) (d.QuotaUsage, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return d.QuotaUsage{}, errors.New("缺少来源共同事务")
	}
	rows, err := tx.Query(ctx, `SELECT first_day::text,coalesce(until_day::text,'') FROM pilot.takeover WHERE instance=$1 AND position=$2`, instance, position)
	if err != nil {
		return d.QuotaUsage{}, err
	}
	var u d.QuotaUsage
	u.Managed, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (s d.UsageSpan, err error) { err = row.Scan(&s.From, &s.Until); return })
	if err != nil || len(u.Managed) == 0 {
		return u, err
	}
	rows, err = tx.Query(ctx, `SELECT first_day::text,coalesce(until_day::text,''),owner,version::text FROM pilot.coverage WHERE instance=$1 AND position=$2`, instance, position)
	if err != nil {
		return u, err
	}
	u.Coverage, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (s d.SourceCoverage, err error) {
		err = row.Scan(&s.From, &s.Until, &s.Owner, &s.Version)
		return
	})
	if err != nil {
		return u, err
	}
	rows, err = tx.Query(ctx, `SELECT first_day::text,coalesce(until_day::text,''),owner,source::text,policy::text,effect FROM pilot.seat WHERE instance=$1 AND position=$2`, instance, position)
	if err != nil {
		return u, err
	}
	u.Uses, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (s d.PositionUse, err error) {
		err = row.Scan(&s.From, &s.Until, &s.Owner, &s.Source, &s.Policy, &s.Effect)
		return
	})
	return u, err
}

func (p quotaPilot) submit(position, owner, id string, inject bool, locked, release chan struct{}) (string, error) {
	ctx := p.f.ctx
	tx, err := p.f.admin.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", owner+id)
	if err != nil {
		return "", err
	}
	var prior, priorPosition string
	err = tx.QueryRow(ctx, "SELECT outcome,position::text FROM pilot.receipt WHERE owner=$1 AND id=$2", owner, id).Scan(&prior, &priorPosition)
	if err == nil {
		if priorPosition != position {
			return "", errors.New("提交号已用于另一岗位")
		}
		return prior, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if _, err = tx.Exec(ctx, "SELECT position FROM pilot.takeover WHERE instance=$1 AND position=$2 FOR SHARE", p.f.instance, position); err != nil {
		return "", err
	}
	if owner == "M05" {
		var unreserved bool
		err = tx.QueryRow(ctx, "SELECT reservation IS NULL FROM pilot.share WHERE position=$1 FOR UPDATE", position).Scan(&unreserved)
		if err != nil {
			return "", err
		}
		if !unreserved {
			return "", errors.New("份额已有预留")
		}
	}
	intent := staff.RegularUseRequest{From: "2027-04-10", Until: "2027-04-15", Enterprise: p.enterprise, Position: position, Category: p.category, Purpose: "常规", Owner: owner}
	uses, decision := p.store.CheckRegularUse(dbtx.With(ctx, tx), intent)
	if locked != nil {
		close(locked)
		select {
		case <-release:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	outcome := "accepted"
	if decision != nil {
		var fault *foundation.Fault
		if !errors.As(decision, &fault) || fault.Status != 409 {
			return "", decision
		}
		outcome = "rejected"
	} else {
		source := foundation.NewID()
		if owner == "M05" && slices.ContainsFunc(uses, func(segment staff.RegularUseSegment) bool { return segment.Effect == "reserved" }) {
			_, err = tx.Exec(ctx, "UPDATE pilot.share SET reservation=$2 WHERE position=$1", position, source)
		} else if owner == "M04" {
			_, err = tx.Exec(ctx, "INSERT INTO pilot.assignment VALUES($1,$2)", position, source)
		}
		if err != nil {
			return "", err
		}
		for _, segment := range uses {
			_, err = tx.Exec(ctx, "INSERT INTO pilot.seat VALUES($1,$2,$3,$4,$5,$6,$7,$8)", p.f.instance, position, owner, source, segment.Policy, segment.Effect, segment.From, segment.Until)
			if err != nil {
				return "", err
			}
		}
		_, err = tx.Exec(ctx, "INSERT INTO pilot.outbox VALUES($1,$2,$3)", owner, position, source)
		if err != nil {
			return "", err
		}
		if inject {
			return "", errors.New("试点故障，原子提交前中断")
		}
	}
	_, err = tx.Exec(ctx, "INSERT INTO pilot.receipt VALUES($1,$2,$3,$4)", owner, id, position, outcome)
	if err != nil {
		return "", err
	}
	return outcome, tx.Commit(ctx)
}

func TestStaffing313Quota(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	locations := location204{t, f, b, map[string]string{"MO": geoCountry203(t, f, "MO")}}
	loc := locations.create("QL313", "MO", "", "2026-01-01")["id"]
	bu := b.call("POST", "/api/v1/business-units", createBody("QB313", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	dep := b.call("POST", "/api/v1/departments", departmentCreateBody("QD313", bu), foundation.NewID(), 201)["departmentId"]
	family := h.create("jobfamily", "QF313", "")
	job := h.create("job", "QJ313", family["id"].(string))
	pilot := quotaPilot{category: "员工", f: f, store: f.runtime.Staffing.(app.Service).QuotaStore.(*staffdb.Store)}
	must(t, f.admin.QueryRow(f.ctx, "SELECT enterprise_id::text FROM organization.enterprise WHERE instance_id=$1", f.instance).Scan(&pilot.enterprise))
	pilot.store.Usage = readPilotUsage
	pilot.store.QuotaPolicies = func(ctx context.Context, intent d.UsageIntent) ([]d.QuotaPolicy, error) {
		tx, ok := dbtx.From(ctx)
		if !ok {
			return nil, errors.New("缺少政策共同事务")
		}
		rows, err := tx.Query(ctx, `SELECT enterprise::text,category,purpose,version::text,first_day::text,coalesce(until_day::text,''),mode,consumes
			FROM pilot.policy WHERE enterprise=$1 AND category=$2 AND purpose=$3 FOR SHARE`, intent.Enterprise, intent.Category, intent.Purpose)
		if err != nil {
			return nil, err
		}
		return pgx.CollectRows(rows, func(row pgx.CollectableRow) (rule d.QuotaPolicy, e error) {
			e = row.Scan(&rule.Enterprise, &rule.Category, &rule.Purpose, &rule.Version, &rule.From, &rule.Until, &rule.Mode, &rule.Consumes)
			return
		})
	}
	f.exec(`CREATE SCHEMA pilot;
	CREATE TABLE pilot.takeover(instance uuid,position uuid,origin uuid,first_day date,until_day date,PRIMARY KEY(instance,position,first_day),
	  FOREIGN KEY(instance,position,origin) REFERENCES staffing.position_source(instance_id,object_id,source_id));
	CREATE TABLE pilot.source_handoff(instance uuid,position uuid,owner text,version uuid,origin uuid,first_day date,until_day date,
		source_evidence jsonb NOT NULL,PRIMARY KEY(instance,position,owner,version),
		FOREIGN KEY(instance,position,origin) REFERENCES staffing.position_source(instance_id,object_id,source_id));
	CREATE TABLE pilot.pending(position uuid,owner text,promise uuid,first_day date,PRIMARY KEY(position,owner,promise));
	CREATE TABLE pilot.coverage(instance uuid,position uuid,owner text,version uuid,origin uuid,first_day date,until_day date,
	  PRIMARY KEY(instance,position,owner,first_day),FOREIGN KEY(instance,position,owner,version) REFERENCES pilot.source_handoff(instance,position,owner,version),FOREIGN KEY(instance,position,origin) REFERENCES staffing.position_source(instance_id,object_id,source_id));
	CREATE TABLE pilot.share(position uuid PRIMARY KEY,reservation uuid);
	CREATE TABLE pilot.assignment(position uuid,source uuid PRIMARY KEY);
	CREATE TABLE pilot.seat(instance uuid,position uuid,owner text,source uuid,policy uuid,effect text,first_day date,until_day date,PRIMARY KEY(source,first_day));
	CREATE TABLE pilot.receipt(owner text,id uuid,position uuid,outcome text, PRIMARY KEY(owner,id));
	CREATE TABLE pilot.outbox(owner text,position uuid,source uuid PRIMARY KEY);
	CREATE TABLE pilot.policy(enterprise uuid NOT NULL,version uuid PRIMARY KEY,category text,purpose text,first_day date,until_day date,mode text,consumes boolean);
	GRANT USAGE ON SCHEMA pilot TO chnhcm_app;
	GRANT SELECT ON ALL TABLES IN SCHEMA pilot TO chnhcm_app`)
	f.exec(`INSERT INTO pilot.policy VALUES
		($1,gen_random_uuid(),'员工','常规','2026-01-01',NULL,'PositionManaged',true),
		(gen_random_uuid(),gen_random_uuid(),'员工','常规','2026-01-01',NULL,'PositionManaged',false),
		($1,gen_random_uuid(),'派遣','常规','2026-01-01',NULL,'PositionManaged',false)`, pilot.enterprise)
	const base = "/api/v1/staffing-quotas"
	setup := func(code string) (string, map[string]any) {
		position := b.call("POST", "/api/v1/positions", map[string]any{"code": code, "name": code, "departmentId": dep, "locationId": loc, "jobId": job["id"], "effectiveDate": "2026-01-01", "operationType": "CREATE", "operationReason": "NEW_POSITION"}, foundation.NewID(), 201)
		id := position["objectId"].(string)
		q := b.call("POST", base, map[string]any{"positionId": id, "effectiveDate": "2026-01-01", "quota": 1, "reason": "试点核编"}, foundation.NewID(), 200)
		f.exec("INSERT INTO pilot.share(position) VALUES($1)", id)
		// Attest the explicit adoption against the actual create command's source
		// and controlled date. A blank usage table alone cannot start control.
		f.exec(`INSERT INTO pilot.takeover SELECT instance_id,position_id,position_source,controlled_from,NULL
			FROM staffing.quota_control WHERE instance_id=$1 AND position_id=$2 AND position_source=$3`, f.instance, id, position["sourceVersion"])
		return id, q
	}
	cover := func(id, owner string) bool {
		// The sole source writers in this isolated pilot use this row for the
		// coverage cutover. A future or pending promise disqualifies zero.
		tag, err := f.admin.Exec(f.ctx, `WITH root AS MATERIALIZED (
		  SELECT * FROM pilot.takeover WHERE instance=$1 AND position=$2 FOR UPDATE
		), delivery AS (
		  INSERT INTO pilot.source_handoff
		  SELECT instance,position,$3,gen_random_uuid(),origin,first_day,until_day,
		     jsonb_build_object('controlledOrigin',origin,'owner',$3::text,'includesFuture',true,
			   'shares',CASE WHEN $3='M05' THEN (SELECT count(*) FROM pilot.share WHERE position=$2) ELSE 0 END)
		  FROM root WHERE NOT EXISTS(SELECT FROM pilot.pending WHERE position=$2 AND owner=$3)
		    AND ($3<>'M05' OR NOT EXISTS(SELECT FROM pilot.share WHERE position=$2 AND reservation IS NOT NULL))
		    AND ($3<>'M04' OR NOT EXISTS(SELECT FROM pilot.assignment WHERE position=$2))
		  RETURNING instance,position,owner,version,origin,first_day,until_day)
		INSERT INTO pilot.coverage SELECT * FROM delivery`, f.instance, id, owner)
		must(t, err)
		return tag.RowsAffected() == 1
	}

	for round, first := range []string{"M05", "M04"} {
		id, quota := setup(fmt.Sprintf("P313-%d", round))
		cover(id, "M05")
		adjust := map[string]any{"quota": 0, "reason": "核实后调减", "effectiveDate": "2027-04-10", "expectedSourceVersion": quota["sourceVersion"]}
		qpath := base + "/" + quota["objectId"].(string) + "/adjust"
		b.call("POST", qpath, adjust, foundation.NewID(), 409)
		cover(id, "M04")
		second := "M04"
		if first == "M04" {
			second = "M05"
		}
		keys := map[string]string{"M04": foundation.NewID(), "M05": foundation.NewID()}
		if _, err := pilot.submit(id, first, keys[first], true, nil, nil); err == nil {
			t.Fatal("故障未中止事务")
		}
		if f.scalar("SELECT count(*) FROM pilot.seat WHERE position=$1", id) != 0 || f.scalar("SELECT count(*) FROM pilot.outbox WHERE position=$1", id) != 0 || f.scalar("SELECT count(*) FROM pilot.receipt WHERE position=$1", id) != 0 ||
			f.scalar("SELECT count(*) FROM pilot.assignment WHERE position=$1", id) != 0 || f.scalar("SELECT count(*) FROM pilot.share WHERE position=$1 AND reservation IS NOT NULL", id) != 0 {
			t.Fatal("回滚后仍有业务事实")
		}
		type answer struct {
			out string
			err error
		}
		ready, release := make(chan struct{}), make(chan struct{})
		winner, loser := make(chan answer, 1), make(chan answer, 1)
		go func() { v, e := pilot.submit(id, first, keys[first], false, ready, release); winner <- answer{v, e} }()
		wait307(t, f, ready)
		go func() { v, e := pilot.submit(id, second, keys[second], false, nil, nil); loser <- answer{v, e} }()
		blocked307(t, f, release)
		close(release)
		one, two := <-winner, <-loser
		must(t, one.err)
		must(t, two.err)
		if one.out != "accepted" || two.out != "rejected" {
			t.Fatal(one, two)
		}
		for owner, want := range map[string]string{first: "accepted", second: "rejected"} {
			v, e := pilot.submit(id, owner, keys[owner], false, nil, nil)
			must(t, e)
			if v != want {
				t.Fatal(owner, v)
			}
		}
		if _, err := pilot.submit(foundation.NewID(), first, keys[first], false, nil, nil); err == nil {
			t.Fatal("原号不得换岗位")
		}
		occupied := 0
		if first == "M04" {
			occupied = 1
		}
		if f.scalar("SELECT count(*) FROM pilot.assignment WHERE position=$1", id) != occupied || f.scalar("SELECT count(*) FROM pilot.share WHERE position=$1 AND reservation IS NOT NULL", id) != 1-occupied ||
			f.scalar("SELECT count(*) FROM pilot.seat WHERE position=$1 AND owner=$2", id, first) != 1 || f.scalar("SELECT count(*) FROM pilot.outbox WHERE position=$1", id) != 1 || f.scalar("SELECT count(*) FROM pilot.receipt WHERE position=$1", id) != 2 {
			t.Fatal("争用后事实不一致")
		}
		b.call("POST", qpath, adjust, foundation.NewID(), 409)
		adjust["effectiveDate"] = "2027-04-15"
		b.call("POST", qpath, adjust, foundation.NewID(), 200)
		t.Logf("%s 先取得名额，%s 拒绝，业务、名额和发布事件各一份，拒绝和成功原号均可重放", first, second)
	}
	id, quota := setup("P313-exempt")
	known := func() any {
		return b.call("GET", base+"/statistics/positions/"+id+"?businessAt=2027-04-12", nil, "", 200)["usageState"]
	}
	if known() != "unknown" {
		t.Fatal("受控设岗不能代替双方覆盖")
	}
	f.exec("INSERT INTO pilot.pending VALUES($1,'M04',gen_random_uuid(),'2028-01-01')", id)
	if cover(id, "M04") || known() != "unknown" {
		t.Fatal("缺少未来承诺交接时错误确认为零")
	}
	f.exec("DELETE FROM pilot.pending WHERE position=$1 AND owner='M04'", id)
	cover(id, "M04")
	if known() != "unknown" {
		t.Fatal("仅人事接管不能宣布余额已核齐")
	}
	cover(id, "M05")
	if known() != "known" {
		t.Fatal("双方原始版本已核齐的零起点未生效")
	}
	exemptPilot := pilot
	exemptPilot.category = "派遣"
	v, e := exemptPilot.submit(id, "M05", foundation.NewID(), false, nil, nil)
	must(t, e)
	if v != "accepted" || known() != "known" || f.scalar("SELECT count(*) FROM pilot.share WHERE position=$1 AND reservation IS NOT NULL", id) != 0 ||
		f.scalar("SELECT count(*) FROM pilot.seat WHERE position=$1 AND owner='M05' AND effect='none'", id) != 1 {
		t.Fatal("免占编招聘须保留采用且保持已核齐的零用量")
	}
	v, e = exemptPilot.submit(id, "M04", foundation.NewID(), false, nil, nil)
	must(t, e)
	if v != "accepted" {
		t.Fatal(v)
	}
	b.call("POST", base+"/"+quota["objectId"].(string)+"/adjust", map[string]any{"quota": 0, "reason": "免占编任职", "effectiveDate": "2027-04-12", "expectedSourceVersion": quota["sourceVersion"]}, foundation.NewID(), 200)
	def := b.call("GET", "/api/v1/positions/"+id, nil, "", 200)
	stop := map[string]any{"expectedSourceVersion": def["headSourceVersion"], "anchorRecordId": rec307(def)["recordId"], "effectiveDate": "2027-04-12", "operationType": "DEACTIVATE", "operationReason": "POSITION_CLOSURE"}
	b.call("POST", "/api/v1/positions/"+id+"/deactivate", stop, foundation.NewID(), 409)
	stop["effectiveDate"] = "2027-04-15"
	b.call("POST", "/api/v1/positions/"+id+"/deactivate", stop, foundation.NewID(), 200)
}
