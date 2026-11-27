//go:build integration

package bootstrap_test

import (
	"context"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func TestHistoryMaintenanceCancellationAndLateConnection(t *testing.T) {
	f := fresh(t)
	f.server.Close()
	f.runtime.Close()
	for _, prefix := range []string{"bu", "department"} {
		f.exec(`INSERT INTO organization.`+prefix+`_dictionary(instance_id,enterprise_id,words,binaries,ranges) SELECT instance_id,enterprise_id,ARRAY['controlled cancellation pool'],'{}'::bytea[],'{}'::daterange[] FROM organization.enterprise WHERE instance_id=$1`, f.instance)
	}
	f.exec(`CREATE FUNCTION public.pause_dictionary_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(1);RETURN OLD;END $$;CREATE TRIGGER pause_dictionary_cleanup BEFORE DELETE ON organization.department_dictionary FOR EACH ROW EXECUTE FUNCTION public.pause_dictionary_cleanup()`)
	// Observe from a different database in this disposable cluster. It cannot
	// accidentally satisfy the late target-database connection counterexample.
	u, err := url.Parse(f.admin.Config().ConnConfig.ConnString())
	must(t, err)
	u.Path = "/postgres"
	observer, err := pgx.Connect(f.ctx, u.String())
	must(t, err)
	defer observer.Close(f.ctx)
	input := bootstrap.HistoryMaintenanceInput{Instance: f.instance, OwnedInstances: []string{f.instance}}
	before := f.scalar(`SELECT (SELECT count(*) FROM organization.bu_dictionary)+(SELECT count(*) FROM organization.department_dictionary)`)
	waitForDeletion := func() {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for {
			var paused bool
			must(t, observer.QueryRow(f.ctx, `SELECT EXISTS(SELECT FROM pg_stat_activity WHERE datname=$1 AND wait_event='PgSleep' AND query LIKE 'DELETE FROM organization.department_dictionary%')`, f.admin.Config().ConnConfig.Database).Scan(&paused))
			if paused {
				return
			}
			if time.Now().After(deadline) {
				t.Fatal("maintenance never entered the protected deletion")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	for _, mode := range []string{"cancel", "late_connection"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(f.ctx)
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := bootstrap.MaintainHistory(ctx, f.admin, input); done <- err }()
			waitForDeletion()
			if mode == "cancel" {
				cancel()
			} else {
				late, err := pgx.Connect(f.ctx, f.admin.Config().ConnConfig.ConnString())
				must(t, err)
				defer late.Close(f.ctx)
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("interrupted offline maintenance committed")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("maintenance did not release its transaction")
			}
			if after := f.scalar(`SELECT (SELECT count(*) FROM organization.bu_dictionary)+(SELECT count(*) FROM organization.department_dictionary)`); after != before {
				t.Fatal("interrupted maintenance partially deleted dictionaries", before, after)
			}
		})
	}
	f.exec(`DROP TRIGGER pause_dictionary_cleanup ON organization.department_dictionary;DROP FUNCTION public.pause_dictionary_cleanup()`)
	result, err := bootstrap.MaintainHistory(f.ctx, f.admin, input)
	must(t, err)
	if result.Removed < 2 {
		t.Fatal("clean retry after cancellation did not recover", result)
	}
}

func TestHistoryMaintenanceOfflineScopeRollbackAndReadback(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("MAINTENANCE_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	created := b.call("POST", "/api/v1/departments", departmentCreateBody("MAINTENANCE_DEP", bu), foundation.NewID(), 201)
	id := created["departmentId"].(string)
	corrected := b.call("POST", "/api/v1/departments/"+id+"/records/"+created["targetRecordId"].(string)+"/correct", map[string]any{
		"expectedSourceVersion": created["sourceVersion"], "expectedRecordRevision": created["targetRevision"], "changes": map[string]any{"name": "清理前的新名称"},
	}, foundation.NewID(), 200)
	terminalKey := foundation.NewID()
	noChange := b.call("POST", "/api/v1/departments/"+id+"/records/"+created["targetRecordId"].(string)+"/correct", map[string]any{
		"expectedSourceVersion": corrected["sourceVersion"], "expectedRecordRevision": corrected["targetRevision"], "changes": map[string]any{"name": "清理前的新名称"},
	}, terminalKey, 200)
	if noChange["outcome"] != "no_change" {
		t.Fatal("no-change terminal fixture did not use the real command", noChange)
	}
	terminalDictionary := f.scalar(`SELECT dictionary_id FROM organization.department_terminal WHERE instance_id=$1 AND submission_id=$2`, f.instance, terminalKey)
	loadTerminal := func() any {
		tx, err := f.admin.Begin(f.ctx)
		must(t, err)
		defer tx.Rollback(f.ctx)
		values, data, err := history.LoadTerminal(f.ctx, tx, history.Department, f.instance, b.subject, terminalKey)
		must(t, err)
		operation, err := history.DecodeDepartmentOperation(f.ctx, values, data)
		must(t, err)
		return operation
	}
	terminalBefore := loadTerminal()
	// A valid prefix-compatible pool used only by a read segment must survive,
	// even when no history chunk or standalone terminal refers to that pool.
	var readDictionary, historyDictionary, readSlot int
	var enterprise, source, start, end string
	var ids []string
	var dictionaryHash, shapeHash []byte
	must(t, f.admin.QueryRow(f.ctx, `INSERT INTO organization.department_dictionary(instance_id,enterprise_id,words,binaries,ranges) SELECT instance_id,enterprise_id,words||ARRAY['read-only reference fixture'],binaries,ranges FROM organization.department_dictionary WHERE id=(SELECT dictionary_id FROM organization.department_history WHERE instance_id=$1 AND object_id=$2) RETURNING id,digest`, f.instance, id).Scan(&readDictionary, &dictionaryHash))
	must(t, f.admin.QueryRow(f.ctx, `SELECT d.enterprise_id::text,r.source_id::text,coalesce(lower(r.span)::text,''),coalesce(upper(r.span)::text,''),r.ids::text[],s.digest,r.dictionary_id,r.slot FROM organization.department_read_segment r JOIN organization.department d ON (d.instance_id,d.department_id)=(r.instance_id,r.object_id) JOIN organization.department_shape s ON s.id=r.shape_id WHERE r.instance_id=$1 AND r.object_id=$2 ORDER BY r.slot DESC LIMIT 1`, f.instance, id).Scan(&enterprise, &source, &start, &end, &ids, &shapeHash, &historyDictionary, &readSlot))
	envelope, err := history.ReadSegmentDigest(history.Department, f.instance, enterprise, id, source, start, end, ids, dictionaryHash, shapeHash)
	must(t, err)
	// Only the selected span gets this envelope. Other spans have different
	// bounds, and the query seal must bind the new dictionary and envelope too.
	f.exec(`UPDATE organization.department_read_segment SET dictionary_id=$3,envelope=$4 WHERE instance_id=$1 AND object_id=$2 AND slot=$5`, f.instance, id, readDictionary, envelope, readSlot)
	f.exec(`UPDATE organization.department_read_segment r SET q_dictionary_seal=p.digest,q_seal=organization.q131_seal(r,d.enterprise_id,d.code,d.first_record_id,rel.head_source_id,p.digest,s.digest)
 FROM organization.department d, organization.department_hierarchy_relation rel, organization.department_dictionary p, organization.department_shape s
 WHERE r.instance_id=$1 AND r.object_id=$2 AND r.slot=$3
 AND (d.instance_id,d.department_id)=(r.instance_id,r.object_id)
 AND (rel.instance_id,rel.department_id)=(r.instance_id,r.object_id)
 AND p.id=r.dictionary_id AND s.id=r.shape_id`, f.instance, id, readSlot)
	if f.scalar(`SELECT count(*) FROM organization.department_history WHERE dictionary_id=$1`, readDictionary) != 0 || f.scalar(`SELECT count(*) FROM organization.department_terminal WHERE dictionary_id=$1`, readDictionary) != 0 || f.scalar(`SELECT count(*) FROM organization.department_read_segment WHERE dictionary_id=$1`, readDictionary) != 1 {
		t.Fatal("read-only dictionary fixture has unexpected references")
	}
	if f.scalar(`SELECT count(*) FROM organization.department_history WHERE dictionary_id=$1`, terminalDictionary) != 0 || f.scalar(`SELECT count(*) FROM organization.department_read_segment WHERE dictionary_id=$1`, terminalDictionary) != 0 {
		t.Fatal("terminal-only dictionary fixture has another consumer")
	}
	b.call("GET", "/api/v1/departments?q=MAINTENANCE_DEP", nil, "", 200)
	other := foundation.NewID()
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: other, EnterpriseCode: "OTHER", EnterpriseName: "同任务另一实例", Start: "2020-01-01", Login: "admin", Password: "admin1234567"}))
	insertGarbage := func(prefix, instance, word string) int {
		var n int
		must(t, f.admin.QueryRow(f.ctx, `INSERT INTO organization.`+prefix+`_dictionary(instance_id,enterprise_id,words,binaries,ranges) SELECT instance_id,enterprise_id,ARRAY[$2]::text[],'{}'::bytea[],'{}'::daterange[] FROM organization.enterprise WHERE instance_id=$1 RETURNING id`, instance, word).Scan(&n))
		return n
	}
	buGarbage := insertGarbage("bu", f.instance, "target bu garbage")
	depGarbage := insertGarbage("department", f.instance, "target department garbage")
	otherGarbage := insertGarbage("department", other, "other instance garbage")
	input := bootstrap.HistoryMaintenanceInput{Instance: f.instance, OwnedInstances: []string{f.instance, other}}
	dryInput := input
	dryInput.DryRun = true
	dry, err := bootstrap.MaintainHistory(f.ctx, f.admin, dryInput)
	must(t, err)
	if !dry.DryRun || dry.Removed != 0 || len(dry.Before) != 2 || dry.Before[0].Rows < 1 || dry.Before[1].Rows < 1 {
		t.Fatal("dry-run did not expose the target-only garbage", dry)
	}
	if _, err = bootstrap.MaintainHistory(f.ctx, f.admin, input); err == nil || !strings.Contains(err.Error(), "connections") {
		t.Fatal("maintenance admitted the running business runtime", err)
	}
	f.server.Close()
	f.runtime.Close()
	foreign := input
	foreign.OwnedInstances = []string{f.instance}
	if _, err = bootstrap.MaintainHistory(f.ctx, f.admin, foreign); err == nil {
		t.Fatal("maintenance admitted an undeclared database instance")
	}
	unknown, err := pgx.Connect(f.ctx, f.admin.Config().ConnConfig.ConnString())
	must(t, err)
	if _, err = bootstrap.MaintainHistory(f.ctx, f.admin, input); err == nil {
		t.Fatal("maintenance admitted an unknown administrative connection")
	}
	must(t, unknown.Close(f.ctx))
	before := f.history()
	beforeCount := f.scalar(`SELECT (SELECT count(*) FROM organization.bu_dictionary)+(SELECT count(*) FROM organization.department_dictionary)`)
	f.exec(`CREATE FUNCTION public.fail_dictionary_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled dictionary cleanup interruption' USING ERRCODE='23514'; END $$;CREATE TRIGGER fail_dictionary_cleanup BEFORE DELETE ON organization.department_dictionary FOR EACH ROW EXECUTE FUNCTION public.fail_dictionary_cleanup()`)
	if _, err = bootstrap.MaintainHistory(f.ctx, f.admin, input); err == nil {
		t.Fatal("controlled deletion failure was ignored")
	}
	if n := f.scalar(`SELECT (SELECT count(*) FROM organization.bu_dictionary)+(SELECT count(*) FROM organization.department_dictionary)`); n != beforeCount {
		t.Fatal("partial dictionary deletion escaped rollback", beforeCount, n)
	}
	f.exec(`DROP TRIGGER fail_dictionary_cleanup ON organization.department_dictionary;DROP FUNCTION public.fail_dictionary_cleanup()`)
	result, err := bootstrap.MaintainHistory(f.ctx, f.admin, input)
	must(t, err)
	if result.Removed < 2 || len(result.ProtectedRoots) != 64 {
		t.Fatal("cleanup did not remove garbage and bind protected roots", result)
	}
	if f.scalar(`SELECT count(*) FROM organization.bu_dictionary WHERE id=$1`, buGarbage) != 0 || f.scalar(`SELECT count(*) FROM organization.department_dictionary WHERE id=$1`, depGarbage) != 0 || f.scalar(`SELECT count(*) FROM organization.department_dictionary WHERE id=$1`, otherGarbage) != 1 {
		t.Fatal("cleanup crossed instance scope or left target garbage")
	}
	if !reflect.DeepEqual(before, f.history()) {
		t.Fatal("dictionary cleanup changed complete historical facts")
	}
	for _, keep := range []int{readDictionary, historyDictionary, terminalDictionary} {
		if f.scalar(`SELECT count(*) FROM organization.department_dictionary WHERE id=$1`, keep) != 1 {
			t.Fatal("cleanup removed a singly referenced pool", keep)
		}
	}
	if !reflect.DeepEqual(terminalBefore, loadTerminal()) {
		t.Fatal("cleanup changed the complete original no-change result")
	}
	again, err := bootstrap.MaintainHistory(f.ctx, f.admin, input)
	must(t, err)
	if again.Removed != 0 || again.ProtectedRoots != result.ProtectedRoots {
		t.Fatal("repeat maintenance changed retained data", again)
	}
}

func TestMergedReadIndexAndCoveringPrimaryKeys(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("MERGED_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	body := departmentCreateBody("MERGED_FUTURE", bu)
	body["effectiveDate"] = "2099-01-01"
	created := b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	id := created["departmentId"].(string)
	if n := f.scalar(`SELECT count(*) FROM organization.department_read_segment WHERE instance_id=$1 AND object_id=$2 AND rule=4 AND ((lower_inf(span) AND upper(span)='2099-01-01' AND q_state='not_effective') OR (lower(span)='2099-01-01' AND upper_inf(span) AND q_state='active'))`, f.instance, id); n != 2 {
		t.Fatal("future-first query modes lost their real first-date boundary", n)
	}
	list := b.call("GET", "/api/v1/departments?q=MERGED_FUTURE", nil, "", 200)
	items := list["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["effectiveDate"] != "2099-01-01" || items[0].(map[string]any)["status"] == "active" {
		t.Fatal("merged lookup span changed future-first business state", list)
	}
	b.call("GET", "/api/v1/departments/"+id+"?sourceVersion="+created["sourceVersion"].(string), nil, "", 200)
	if f.scalar(`SELECT count(*) FROM pg_indexes WHERE schemaname='organization' AND indexname IN ('department_list_heads_cover','department_relation_heads_cover')`) != 0 {
		t.Fatal("separate covering indexes remain")
	}
	for _, tc := range []struct {
		index string
		keys  int
		attrs int
	}{{"department_pkey", 2, 4}, {"department_hierarchy_relation_pkey", 3, 4}} {
		var keys, attrs int
		var primary, unique bool
		must(t, f.admin.QueryRow(f.ctx, `SELECT indnkeyatts,indnatts,indisprimary,indisunique FROM pg_index WHERE indexrelid=('organization.'||$1)::regclass`, tc.index).Scan(&keys, &attrs, &primary, &unique))
		if keys != tc.keys || attrs != tc.attrs || !primary || !unique {
			t.Fatal("covering primary key changed uniqueness or lost payload", tc, keys, attrs, primary, unique)
		}
	}
}

func TestHistoryMaintenanceCrossInstanceReferencesAndLockingClient(t *testing.T) {
	f := fresh(t)
	g := &fixture{t: t, ctx: f.ctx, admin: f.admin, instance: foundation.NewID(), password: "admin1234567"}
	must(t, bootstrap.Initialize(g.ctx, g.admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: g.instance, EnterpriseCode: "OTHER", EnterpriseName: "另一归属企业", Start: "2020-01-01", Login: "admin", Password: g.password}))
	var err error
	g.runtime, err = bootstrap.Open(g.ctx, f.runtime.Pool.Config().ConnConfig.ConnString(), g.instance)
	must(t, err)
	t.Cleanup(g.runtime.Close)
	api := &bootstrap.API{Runtime: g.runtime, InsecureLocal: true}
	g.server = httptest.NewUnstartedServer(nil)
	api.Origin = "http://" + g.server.Listener.Addr().String()
	g.server.Config.Handler = api.Handler()
	g.server.Start()
	t.Cleanup(g.server.Close)
	for _, owner := range []*fixture{f, g} {
		b := owner.login("admin", owner.password)
		bu := b.call("POST", "/api/v1/business-units", createBody("OWNER_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
		b.call("POST", "/api/v1/departments", departmentCreateBody("OWNER_DEP", bu), foundation.NewID(), 201)
	}
	protected, targetGarbage, otherGarbage := map[string]int{}, map[string]int{}, map[string]int{}
	for _, prefix := range []string{"bu", "department"} {
		insert := func(instance, word string) int {
			var id int
			must(t, f.admin.QueryRow(f.ctx, `INSERT INTO organization.`+prefix+`_dictionary(instance_id,enterprise_id,words,binaries,ranges) SELECT instance_id,enterprise_id,ARRAY[$2]::text[],'{}'::bytea[],'{}'::daterange[] FROM organization.enterprise WHERE instance_id=$1 RETURNING id`, instance, word).Scan(&id))
			return id
		}
		protected[prefix] = insert(f.instance, "cross-owner read reference")
		targetGarbage[prefix] = insert(f.instance, "shared unreferenced content")
		otherGarbage[prefix] = insert(g.instance, "shared unreferenced content")
		// The derivative FK deliberately references a globally unique dictionary
		// ID. Even an invalid cross-instance/enterprise consumer must prevent GC
		// from deleting that row; normal Go reads separately reject the mismatch.
		f.exec(`UPDATE organization.`+prefix+`_read_segment SET dictionary_id=$2 WHERE instance_id=$1`, g.instance, protected[prefix])
	}
	other := g.login("admin", g.password)
	for _, route := range []string{"business-units", "departments"} {
		status, body, err := other.response("GET", "/api/v1/"+route, nil, "")
		must(t, err)
		if status != 503 || body["items"] != nil {
			t.Fatal("a cross-owner dictionary was exposed as valid facts", route, status)
		}
	}
	before, otherBefore := f.history(), g.history()
	f.server.Close()
	f.runtime.Close()
	g.server.Close()
	g.runtime.Close()
	input := bootstrap.HistoryMaintenanceInput{Instance: f.instance, OwnedInstances: []string{f.instance, g.instance}}
	dry := input
	dry.DryRun = true
	observed, err := bootstrap.MaintainHistory(f.ctx, f.admin, dry)
	must(t, err)
	if len(observed.Before) != 2 || observed.Before[0].Rows != 1 || observed.Before[1].Rows != 1 {
		t.Fatal("dry-run ignored a cross-owner reference or included another instance", observed)
	}
	holder, err := pgx.Connect(f.ctx, f.admin.Config().ConnConfig.ConnString())
	must(t, err)
	defer holder.Close(f.ctx)
	_, err = holder.Exec(f.ctx, `BEGIN; LOCK TABLE organization.department_read_segment IN ROW EXCLUSIVE MODE`)
	must(t, err)
	if _, err = bootstrap.MaintainHistory(f.ctx, f.admin, input); err == nil || !strings.Contains(err.Error(), "connections") {
		t.Fatal("maintenance admitted a client holding a reference-table lock", err)
	}
	for prefix, id := range targetGarbage {
		if f.scalar(`SELECT count(*) FROM organization.`+prefix+`_dictionary WHERE id=$1`, id) != 1 {
			t.Fatal("rejected maintenance removed dictionary data")
		}
	}
	_, err = holder.Exec(f.ctx, `ROLLBACK`)
	must(t, err)
	must(t, holder.Close(f.ctx))
	result, err := bootstrap.MaintainHistory(f.ctx, f.admin, input)
	must(t, err)
	if result.Removed != 2 {
		t.Fatal("cleanup did not remove exactly the target's two unreferenced pools", result)
	}
	for prefix, id := range protected {
		if f.scalar(`SELECT count(*) FROM organization.`+prefix+`_dictionary WHERE id=$1`, id) != 1 || f.scalar(`SELECT count(*) FROM organization.`+prefix+`_dictionary WHERE id=$1`, otherGarbage[prefix]) != 1 {
			t.Fatal("cleanup removed a cross-owner reference or another instance's content")
		}
	}
	if !reflect.DeepEqual(before, f.history()) || !reflect.DeepEqual(otherBefore, g.history()) {
		t.Fatal("cleanup changed either owner's complete immutable history")
	}
	again, err := bootstrap.MaintainHistory(f.ctx, f.admin, input)
	must(t, err)
	if again.Removed != 0 {
		t.Fatal("repeated cleanup removed retained pools", again)
	}
}
