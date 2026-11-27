//go:build integration && query131

package bootstrap_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func query131Fixture(t *testing.T) (*fixture, *browser, history.Query131Input) {
	t.Helper()
	f := fresh(t)
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(context.Background())
	must(t, history.InstallQuery131(f.ctx, tx))
	must(t, tx.Commit(f.ctx))
	var enterprise string
	must(t, f.admin.QueryRow(f.ctx, `SELECT enterprise_id::text FROM organization.enterprise WHERE instance_id=$1`, f.instance).Scan(&enterprise))
	b := f.login("admin", f.password)
	return f, b, history.Query131Input{Instance: f.instance, Enterprise: enterprise, Day: "2026-11-09", All: true, Page: 1, PageSize: 20}
}
func query131Read(t *testing.T, f *fixture, q history.Query131Input) (history.Query131Result, history.Query131Stats) {
	t.Helper()
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(context.Background())
	v, s, err := history.CaptureQuery131(f.ctx, tx, q)
	must(t, err)
	must(t, tx.Commit(f.ctx))
	return v, s
}
func query131AssertBad(t *testing.T, f *fixture, b *browser, tx pgx.Tx, q history.Query131Input) {
	t.Helper()
	v, s, err := history.CaptureQuery131(f.ctx, tx, q)
	var fault *foundation.Fault
	if !errors.As(err, &fault) || fault.Status != 503 || fault.Code != "M02_DEPENDENCY_UNAVAILABLE" || v.Count != 0 || len(v.Items) != 0 || s.Bad < 1 {
		t.Fatalf("bad fact escaped integrity gate: count=%d items=%d stats=%+v err=%v", v.Count, len(v.Items), s, err)
	}
	// This harness exercises the separate tagged storage diagnostic. Ordinary
	// list/selection reads intentionally do not repeat this enterprise audit.
}
func query131BU(t *testing.T, b *browser, code string) string {
	t.Helper()
	return b.call("POST", "/api/v1/business-units", createBody(code, "2020-01-01"), foundation.NewID(), 201)["buId"].(string)
}
func query131Create(t *testing.T, b *browser, code, name, bu, date string) map[string]any {
	t.Helper()
	body := departmentCreateBody(code, bu)
	body["name"] = name
	body["effectiveDate"] = date
	return b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
}

func TestDepartmentQuery131FilteringIntegrity(t *testing.T) {
	f, b, q := query131Fixture(t)
	bu := query131BU(t, b, "Q131_A")
	otherBU := query131BU(t, b, "Q131_B")
	for i := 0; i < 21; i++ {
		query131Create(t, b, fmt.Sprintf("A%03d", i), "普通部门", bu, "2026-01-01")
	}
	target := query131Create(t, b, "Z_TARGET", "Needle_%中文𠀀", bu, "2026-01-01")
	id := target["departmentId"].(string)
	v, s := query131Read(t, f, q)
	if v.Count != 22 || len(v.Items) != 20 || s.Objects != 22 || s.Segments != 44 || s.DecodedPageSegments != 20 {
		t.Fatal(v, s)
	}
	search := q
	search.Contains = "NEEDLE_%中文𠀀"
	v, _ = query131Read(t, f, search)
	if v.Count != 1 || v.Items[0]["id"] != id {
		t.Fatal("literal Go-normalized search differs", v)
	}
	empty := q
	empty.Contains = "不存在"
	v, _ = query131Read(t, f, empty)
	if v.Count != 0 || len(v.Items) != 0 {
		t.Fatal(v)
	}
	denied := q
	denied.All = false
	denied.BUs = []string{}
	v, _ = query131Read(t, f, denied)
	if v.Count != 0 {
		t.Fatal(v)
	}
	last := q
	last.Page = 99
	v, _ = query131Read(t, f, last)
	if v.Count != 22 || len(v.Items) != 0 {
		t.Fatal(v)
	}
	f.exec(`CREATE TABLE public.q131_copy AS SELECT * FROM organization.department_read_segment`)
	otherInstance := foundation.NewID()
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{DevelopmentFixture: true, Instance: otherInstance, EnterpriseCode: "Q131_OTHER", EnterpriseName: "另一个实例", Start: "2020-01-01", Login: "admin", Password: foundation.NewID()}))
	var foreignDictionary int32
	must(t, f.admin.QueryRow(f.ctx, `INSERT INTO organization.department_dictionary(instance_id,enterprise_id,words,binaries,ranges)
 SELECT e.instance_id,e.enterprise_id,d.words,d.binaries,d.ranges FROM organization.enterprise e
 CROSS JOIN organization.department_dictionary d
 WHERE e.instance_id=$2 AND d.id=(SELECT dictionary_id FROM public.q131_copy WHERE object_id=$1 LIMIT 1) RETURNING id`, id, otherInstance).Scan(&foreignDictionary))
	cases := []struct {
		name, sql string
		args      []any
	}{
		{"filtered_name", `UPDATE organization.department_read_segment SET q_name='wrong',q_name_key='wrong' WHERE object_id=$1`, nil},
		{"path_day", `UPDATE organization.department_read_segment SET q_path_date='1999-01-01' WHERE object_id=$1`, nil},
		{"code_key", `UPDATE organization.department_read_segment SET q_code_key='wrong' WHERE object_id=$1`, nil},
		{"scope_bu", `UPDATE organization.department_read_segment SET q_bu=$2 WHERE object_id=$1`, []any{otherBU}},
		{"hidden_short_name", `UPDATE organization.department_read_segment SET q_short_name='bad' WHERE object_id=$1`, nil},
		{"null_to_empty_short_name", `UPDATE organization.department_read_segment SET q_short_name='' WHERE object_id=$1`, nil},
		{"short_search_key", `UPDATE organization.department_read_segment SET q_short_key='bad' WHERE object_id=$1`, nil},
		{"name_search_key", `UPDATE organization.department_read_segment SET q_name_key='wrong' WHERE object_id=$1`, nil},
		{"selected_record", `UPDATE organization.department_read_segment SET q_record=gen_random_uuid() WHERE object_id=$1`, nil},
		{"record_revision", `UPDATE organization.department_read_segment SET q_revision=gen_random_uuid() WHERE object_id=$1`, nil},
		{"record_start", `UPDATE organization.department_read_segment SET q_date='2099-01-01' WHERE object_id=$1`, nil},
		{"record_end", `UPDATE organization.department_read_segment SET q_end='2099-01-01' WHERE object_id=$1`, nil},
		{"first_date", `UPDATE organization.department_read_segment SET q_first_date='2099-01-01' WHERE object_id=$1`, nil},
		{"state", `UPDATE organization.department_read_segment SET q_state='inactive' WHERE object_id=$1`, nil},
		{"relation_source", `UPDATE organization.department_read_segment SET q_relation=gen_random_uuid() WHERE object_id=$1`, nil},
		{"parent", `UPDATE organization.department_read_segment SET q_parent=gen_random_uuid() WHERE object_id=$1`, nil},
		{"owner_binding", `UPDATE organization.department_read_segment SET q_owner_seal=decode(repeat('00',32),'hex') WHERE object_id=$1`, nil},
		{"dictionary_binding", `UPDATE organization.department_read_segment SET q_dictionary_seal=decode(repeat('00',32),'hex') WHERE object_id=$1`, nil},
		{"shape_binding", `UPDATE organization.department_read_segment SET q_shape_seal=decode(repeat('00',32),'hex') WHERE object_id=$1`, nil},
		{"triggers_disabled_projection", `UPDATE organization.department_read_segment SET q_name_key='wrong' WHERE object_id=$1`, nil},
		{"query_seal", `UPDATE organization.department_read_segment SET q_seal=decode(repeat('00',32),'hex') WHERE object_id=$1`, nil},
		{"missing_query_seal", `UPDATE organization.department_read_segment SET q_seal=NULL WHERE object_id=$1`, nil},
		{"source", `UPDATE organization.department_read_segment SET source_id=gen_random_uuid() WHERE object_id=$1`, nil},
		{"original_envelope", `UPDATE organization.department_read_segment SET envelope=decode(repeat('00',32),'hex') WHERE object_id=$1`, nil},
		{"uuid_directory", `UPDATE organization.department_read_segment SET ids[1]=gen_random_uuid() WHERE object_id=$1`, nil},
		{"span_moves_out_of_day", `UPDATE organization.department_read_segment SET span=daterange('2099-01-01',NULL,'[)') WHERE object_id=$1 AND slot=1`, nil},
		{"non_current_segment", `UPDATE organization.department_read_segment SET q_name='bad outside D' WHERE object_id=$1 AND slot=0`, nil},
		{"missing_current_segment", `DELETE FROM organization.department_read_segment WHERE object_id=$1 AND slot=1`, nil},
		{"missing_pre_first_segment", `DELETE FROM organization.department_read_segment WHERE object_id=$1 AND slot=0`, nil},
		{"missing_all_segments", `DELETE FROM organization.department_read_segment WHERE object_id=$1`, nil},
		{"swapped_dictionary", `UPDATE organization.department_read_segment SET dictionary_id=(SELECT min(dictionary_id) FROM public.q131_copy WHERE object_id<>$1) WHERE object_id=$1`, nil},
		{"foreign_instance_dictionary", `UPDATE organization.department_read_segment SET dictionary_id=$2 WHERE object_id=$1`, []any{foreignDictionary}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := f.admin.Begin(f.ctx)
			must(t, err)
			defer tx.Rollback(context.Background())
			if tc.name == "triggers_disabled_projection" {
				_, err = tx.Exec(f.ctx, `ALTER TABLE organization.department_read_segment DISABLE TRIGGER ALL`)
				must(t, err)
			}
			if tc.name == "missing_query_seal" {
				// Formal storage rejects NULL natively. Remove that constraint only
				// inside this rolled-back fault transaction to retain read-time coverage.
				_, err = tx.Exec(f.ctx, `ALTER TABLE organization.department_read_segment ALTER COLUMN q_seal DROP NOT NULL`)
				must(t, err)
			}
			_, err = tx.Exec(f.ctx, tc.sql, append([]any{id}, tc.args...)...)
			must(t, err)
			for _, input := range []history.Query131Input{q, search, empty, denied, last} {
				query131AssertBad(t, f, b, tx, input)
			}
			// A different instance has no owners, even if this instance is corrupt.
			outside := q
			outside.Instance = foundation.NewID()
			got, _, err := history.CaptureQuery131(f.ctx, tx, outside)
			must(t, err)
			if got.Count != 0 {
				t.Fatal("cross-instance rows returned", got)
			}
		})
	}
	for _, kind := range []string{"dictionary", "shape"} {
		t.Run("immutable_"+kind, func(t *testing.T) {
			tx, err := f.admin.Begin(f.ctx)
			must(t, err)
			defer tx.Rollback(context.Background())
			change := `words=words||ARRAY['unrelated appended word']`
			if kind == "shape" {
				change = `parts[1]=parts[1]||decode('01','hex')`
			}
			sql := `UPDATE organization.department_` + kind + ` SET ` + change + ` WHERE id=(SELECT ` + kind + `_id FROM organization.department_read_segment WHERE object_id=$1 LIMIT 1)`
			_, err = tx.Exec(f.ctx, sql, id)
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "23514" {
				t.Fatal("immutable content update was not blocked", err)
			}
		})
		t.Run("forced_"+kind+"_damage", func(t *testing.T) {
			tx, err := f.admin.Begin(f.ctx)
			must(t, err)
			defer tx.Rollback(context.Background())
			_, err = tx.Exec(f.ctx, `ALTER TABLE organization.department_`+kind+` DISABLE TRIGGER immutable_`+kind)
			must(t, err)
			change := `words=words||ARRAY['unrelated appended word']`
			if kind == "shape" {
				change = `parts[1]=parts[1]||decode('01','hex')`
			}
			_, err = tx.Exec(f.ctx, `UPDATE organization.department_`+kind+` SET `+change+` WHERE id=(SELECT `+kind+`_id FROM organization.department_read_segment WHERE object_id=$1 LIMIT 1)`, id)
			must(t, err)
			for _, input := range []history.Query131Input{q, search, empty, denied, last} {
				query131AssertBad(t, f, b, tx, input)
			}
		})
	}
	for _, target := range []struct{ table, column string }{
		{"department_read_segment", "q_stored_seal"}, {"department_dictionary", "digest"}, {"department_shape", "digest"},
	} {
		t.Run("generated_"+target.table+"_cannot_be_forged", func(t *testing.T) {
			tx, err := f.admin.Begin(f.ctx)
			must(t, err)
			defer tx.Rollback(context.Background())
			_, err = tx.Exec(f.ctx, `ALTER TABLE organization.`+target.table+` DISABLE TRIGGER ALL`)
			must(t, err)
			_, err = tx.Exec(f.ctx, `UPDATE organization.`+target.table+` SET `+target.column+`=decode(repeat('00',32),'hex')`)
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "428C9" {
				t.Fatal("generated checksum assignment was not rejected", err)
			}
		})
	}
	t.Run("missing_query_seal_native_rejection", func(t *testing.T) {
		tx, err := f.admin.Begin(f.ctx)
		must(t, err)
		defer tx.Rollback(context.Background())
		_, err = tx.Exec(f.ctx, `UPDATE organization.department_read_segment SET q_seal=NULL WHERE object_id=$1`, id)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "23502" {
			t.Fatal("missing query seal not rejected", err)
		}
	})
	t.Run("overlapping_segment_native_rejection", func(t *testing.T) {
		tx, err := f.admin.Begin(f.ctx)
		must(t, err)
		defer tx.Rollback(context.Background())
		var columns string
		must(t, tx.QueryRow(f.ctx, `SELECT string_agg(quote_ident(attname),',' ORDER BY attnum) FROM pg_attribute WHERE attrelid='organization.department_read_segment'::regclass AND attnum>0 AND NOT attisdropped AND attgenerated=''`).Scan(&columns))
		_, err = tx.Exec(f.ctx, `INSERT INTO organization.department_read_segment (`+columns+`) SELECT `+columns+` FROM (SELECT (jsonb_populate_record(NULL::organization.department_read_segment,to_jsonb(s)||jsonb_build_object('slot',999))).* FROM organization.department_read_segment s WHERE object_id=$1 AND slot=1) source`, id)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "23P01" {
			t.Fatal("overlap not rejected", err)
		}
	})
	v, s = query131Read(t, f, q)
	if v.Count != 22 || s.Bad != 0 {
		t.Fatal("fault rollback damaged source", v, s)
	}
	t.Logf("%d projection faults x 5 query shapes rejected; original dictionary/shape immutability and overlap retained", len(cases))
}

func TestDepartmentQuery131UUIDDirectoryIntegrity(t *testing.T) {
	f, b, q := query131Fixture(t)
	bu := query131BU(t, b, "Q131_ARRAY")
	for i := 0; i < 20; i++ {
		query131Create(t, b, fmt.Sprintf("A%03d", i), "普通部门", bu, "2026-01-01")
	}
	target := query131Create(t, b, "Z_ARRAY", "数组故障目标", bu, "2026-01-01")
	id := target["departmentId"].(string)
	search, empty, denied, last := q, q, q, q
	search.Contains, empty.Contains = "数组故障目标", "不存在"
	denied.All, denied.BUs, last.Page = false, []string{}, 99
	inputs := []history.Query131Input{q, search, empty, denied, last}
	for _, input := range inputs {
		query131Read(t, f, input)
	}
	// Array dimensions/lower bounds are lost by unnest, so the UUID values and
	// their original seal can remain unchanged. Neither native rejection nor
	// the pre-filter gate may depend on fetching this target's page.
	cases := []struct{ name, assignment string }{
		{"two_dimensions", `ids=ARRAY[ids]`},
		{"three_dimensions", `ids=ARRAY[ARRAY[ids]]`},
		{"zero_lower_bound", `ids=('[0:'||(cardinality(ids)-1)::text||']='||ids::text)::uuid[]`},
		{"negative_lower_bound", `ids=('[-1:'||(cardinality(ids)-2)::text||']='||ids::text)::uuid[]`},
		{"higher_lower_bound", `ids=('[2:'||(cardinality(ids)+1)::text||']='||ids::text)::uuid[]`},
		{"empty_directory", `ids='{}'::uuid[]`},
		{"null_element", `ids[1]=NULL`},
	}
	for _, tc := range cases {
		t.Run("native_"+tc.name, func(t *testing.T) {
			// Use the application role: malformed storage must be rejected before
			// it can be committed, without requiring a later read.
			tx, err := f.runtime.Pool.Begin(f.ctx)
			must(t, err)
			defer tx.Rollback(context.Background())
			_, err = tx.Exec(f.ctx, `UPDATE organization.department_read_segment SET `+tc.assignment+` WHERE object_id=$1`, id)
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "23514" || pg.ConstraintName != "department_query131_ids_valid" {
				t.Fatalf("invalid UUID directory escaped its storage constraint: %v", err)
			}
		})
		t.Run("capture_"+tc.name, func(t *testing.T) {
			tx, err := f.admin.Begin(f.ctx)
			must(t, err)
			defer tx.Rollback(context.Background())
			// Only this disposable transaction bypasses the new CHECK so the
			// read guard is proved independently; rollback restores everything.
			_, err = tx.Exec(f.ctx, `ALTER TABLE organization.department_read_segment DROP CONSTRAINT IF EXISTS department_query131_ids_valid`)
			must(t, err)
			changed, err := tx.Exec(f.ctx, `UPDATE organization.department_read_segment SET `+tc.assignment+` WHERE object_id=$1`, id)
			must(t, err)
			if changed.RowsAffected() != 2 {
				t.Fatal("array fault did not cover both target segments", changed.RowsAffected())
			}
			for _, input := range inputs {
				query131AssertBad(t, f, b, tx, input)
			}
		})
	}
	v, stats := query131Read(t, f, q)
	if v.Count != 21 || len(v.Items) != 20 || stats.Bad != 0 {
		t.Fatal("array fault rollback changed valid results", v, stats)
	}
	t.Logf("checked %d invalid array forms against application-role storage CHECK and five capture shapes; valid data restored", len(cases))
}

func TestDepartmentQuery131SourceAtomicityAndDates(t *testing.T) {
	f, b, q := query131Fixture(t)
	bu := query131BU(t, b, "Q131_DATE_A")
	other := query131BU(t, b, "Q131_DATE_B")
	created := query131Create(t, b, "DATE_ROOT", "原始名称", bu, "2026-01-01")
	id, oldSource := created["departmentId"].(string), created["sourceVersion"].(string)
	path := "/api/v1/departments/" + id
	f.exec(`CREATE TABLE public.q131_before AS SELECT * FROM organization.department_read_segment`)
	var before int64
	must(t, f.admin.QueryRow(f.ctx, `SELECT sum(cardinality(source_refs)) FROM organization.department_history`).Scan(&before))
	f.exec(`CREATE FUNCTION public.q131_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled query projection failure' USING ERRCODE='23514'; END $$; CREATE TRIGGER q131_fail BEFORE INSERT ON organization.department_read_segment FOR EACH ROW WHEN (NEW.q_seal IS NOT NULL) EXECUTE FUNCTION public.q131_fail()`)
	correct := map[string]any{"expectedSourceVersion": oldSource, "expectedRecordRevision": created["targetRevision"], "changes": map[string]any{"name": "新名称", "shortName": "简称"}}
	b.call("POST", path+"/records/"+created["targetRecordId"].(string)+"/correct", correct, foundation.NewID(), 503)
	var after, difference int64
	must(t, f.admin.QueryRow(f.ctx, `SELECT sum(cardinality(source_refs)) FROM organization.department_history`).Scan(&after))
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM ((SELECT * FROM organization.department_read_segment EXCEPT SELECT * FROM public.q131_before) UNION ALL (SELECT * FROM public.q131_before EXCEPT SELECT * FROM organization.department_read_segment)) d`).Scan(&difference))
	current := b.call("GET", path, nil, "", 200)
	if before != after || difference != 0 || current["headSourceVersion"] != oldSource {
		t.Fatal("failed extension committed source or projection", before, after, difference, current)
	}
	f.exec(`DROP TRIGGER q131_fail ON organization.department_read_segment;DROP FUNCTION public.q131_fail()`)
	changed := b.call("POST", path+"/records/"+created["targetRecordId"].(string)+"/correct", correct, foundation.NewID(), 200)
	v, _ := query131Read(t, f, q)
	if v.Items[0]["name"] != "新名称" || v.Items[0]["shortName"] != "简称" || v.Items[0]["source"] != changed["sourceVersion"] {
		t.Fatal(v)
	}
	old := b.call("GET", path+"?sourceVersion="+oldSource, nil, "", 200)
	if old["selectedRecord"].(map[string]any)["name"] != "原始名称" {
		t.Fatal("old source changed", old)
	}
	added := b.call("POST", path+"/records", map[string]any{"expectedSourceVersion": changed["sourceVersion"], "anchorRecordId": changed["targetRecordId"], "effectiveDate": "2027-01-01", "operationType": "CHANGE", "operationReason": "ORGANIZATION_REASSIGNMENT", "changes": map[string]any{"managementBuId": other}}, foundation.NewID(), 201)
	for _, tc := range []struct {
		day, bu string
		count   int64
	}{{"2026-12-31", bu, 1}, {"2027-01-01", bu, 0}, {"2026-12-31", other, 0}, {"2027-01-01", other, 1}} {
		in := q
		in.Day = tc.day
		in.All = false
		in.BUs = []string{tc.bu}
		v, _ := query131Read(t, f, in)
		if v.Count != tc.count {
			t.Fatal("wrong Go dated membership", tc, v)
		}
	}
	if added["sourceVersion"] == changed["sourceVersion"] {
		t.Fatal("source not advanced")
	}
	future := query131Create(t, b, "FUTURE_ROOT", "未来首条", bu, "2099-01-01")
	for _, tc := range []struct{ day, state string }{{"2098-12-31", "not_effective"}, {"2099-01-01", "active"}} {
		in := q
		in.Day = tc.day
		in.Contains = "未来首条"
		v, _ := query131Read(t, f, in)
		if v.Count != 1 || v.Items[0]["status"] != tc.state || v.Items[0]["date"] != "2099-01-01" || v.Items[0]["firstRecord"] != future["targetRecordId"] {
			t.Fatal("future first manufactured a today record", tc, v)
		}
	}
	stopped := query131Create(t, b, "STOP_ROOT", "停用继承", bu, "2026-01-01")
	b.call("POST", "/api/v1/departments/"+stopped["departmentId"].(string)+"/deactivate", map[string]any{"expectedSourceVersion": stopped["sourceVersion"], "anchorRecordId": stopped["targetRecordId"], "effectiveDate": "2026-06-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 201)
	in := q
	in.Contains = "停用继承"
	in.State = "inactive"
	v, _ = query131Read(t, f, in)
	if v.Count != 1 || v.Items[0]["bu"] != bu {
		t.Fatal("inactive inherited scope lost", v)
	}
	t.Log("actual API source transaction rollback, old source recovery, future first, dated BU transfer and inactive scope passed")
}

// Kept here for subsequent concurrency/cost probes to log machine-readable,
// non-sensitive observations without writing fixture credentials to disk.
func query131Log(t *testing.T, kind string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	must(t, err)
	t.Log(kind + " " + string(raw))
}

func TestDepartmentQuery131CaptureConcurrency(t *testing.T) {
	f, b, q := query131Fixture(t)
	bu := query131BU(t, b, "Q131_CONCURRENT")
	created := query131Create(t, b, "OLD", "needle original", bu, "2026-01-01")
	id, source := created["departmentId"].(string), created["sourceVersion"].(string)
	q.Contains = "needle"
	gate, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer gate.Rollback(context.Background())
	_, err = gate.Exec(f.ctx, `SELECT pg_advisory_xact_lock(1310091)`)
	must(t, err)
	type captured struct {
		count *int64
		raw   []byte
		stats history.Query131Stats
		err   error
	}
	finished := make(chan captured, 1)
	ctx, cancel := context.WithTimeout(f.ctx, 15*time.Second)
	defer cancel()
	go func() {
		var got captured
		tx, e := f.admin.Begin(ctx)
		if e != nil {
			got.err = e
			finished <- got
			return
		}
		defer tx.Rollback(context.Background())
		_, e = tx.Exec(ctx, `SELECT set_config('application_name','q131-capture-barrier',true)`)
		if e != nil {
			got.err = e
			finished <- got
			return
		}
		sql := strings.Replace(history.Query131CaptureSQL, "FROM organization.department d\n", "FROM organization.department d CROSS JOIN (SELECT pg_advisory_xact_lock(1310091)) gate\n", 1)
		got.err = tx.QueryRow(ctx, sql, q.Args()...).Scan(&got.stats.Objects, &got.stats.Segments, &got.stats.Bad, &got.stats.Dictionaries, &got.stats.Shapes, &got.count, &got.raw)
		finished <- got
	}()
	// Observe an actual PostgreSQL lock wait; sleeping is only poll pacing.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var blocked int
		must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM pg_stat_activity WHERE application_name='q131-capture-barrier' AND wait_event_type='Lock' AND wait_event='advisory'`).Scan(&blocked))
		if blocked == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("capture did not reach the server snapshot barrier")
		}
		time.Sleep(5 * time.Millisecond)
	}
	changed := b.call("POST", "/api/v1/departments/"+id+"/records/"+created["targetRecordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": source, "expectedRecordRevision": created["targetRevision"], "changes": map[string]any{"name": "needle changed"}}, foundation.NewID(), 200)
	query131Create(t, b, "NEW", "needle newcomer", bu, "2026-01-01")
	must(t, gate.Commit(f.ctx))
	got := <-finished
	must(t, got.err)
	var items []map[string]any
	must(t, json.Unmarshal(got.raw, &items))
	if got.count == nil || *got.count != 1 || got.stats.Objects != 1 || got.stats.Bad != 0 || len(items) != 1 || items[0]["source"] != source || items[0]["name"] != "needle original" {
		t.Fatal("capture mixed concurrent sources", got.count, got.stats, items)
	}
	// The old dictionary is genuinely no longer referenced after the history
	// chunk moves to its append-only superset. Reclaim it before Go decodes the page.
	oldDictionary := int(items[0]["dictionary"].(float64))
	tag, err := f.admin.Exec(f.ctx, `DELETE FROM organization.department_dictionary WHERE id=$1`, oldDictionary)
	must(t, err)
	if tag.RowsAffected() != 1 {
		t.Fatal("obsolete dictionary was not reclaimed")
	}
	must(t, history.ValidateQuery131Captured(f.ctx, q, items[0]))
	current, stats := query131Read(t, f, q)
	if current.Count != 2 || stats.Objects != 2 {
		t.Fatal("next capture missed committed addition", current, stats)
	}
	found := false
	for _, item := range current.Items {
		if item["id"] == id {
			found = item["source"] == changed["sourceVersion"] && item["name"] == "needle changed"
		}
	}
	if !found {
		t.Fatal("next capture retained old source", current)
	}
	old := b.call("GET", "/api/v1/departments/"+id+"?sourceVersion="+source, nil, "", 200)
	if old["selectedRecord"].(map[string]any)["name"] != "needle original" {
		t.Fatal("history changed after dictionary reclamation", old)
	}
	t.Log("one captured count/page/source survived concurrent correction, addition, segment replacement and actual obsolete-dictionary deletion; next capture saw both commits")
}

func TestDepartmentQuery131Cost(t *testing.T) {
	f, b, q := query131Fixture(t)
	bu := query131BU(t, b, "Q131_COST")
	created := 0
	for _, scale := range []int{20, 137, 1000} {
		writeStart := time.Now()
		for ; created < scale; created++ {
			query131Create(t, b, fmt.Sprintf("C%05d", created), fmt.Sprintf("容量部门 %05d", created), bu, "2026-01-01")
		}
		writesElapsed := time.Since(writeStart)
		f.exec(`ANALYZE organization.department;ANALYZE organization.department_read_segment;ANALYZE organization.department_dictionary;ANALYZE organization.department_shape`)
		var times []float64
		var stats history.Query131Stats
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for i := 0; i < 20; i++ {
			start := time.Now()
			v, s := query131Read(t, f, q)
			times = append(times, float64(time.Since(start).Microseconds())/1000)
			stats = s
			if v.Count != int64(scale) || len(v.Items) != 20 || s.Objects != int64(scale) || s.Segments != int64(scale*2) || s.Bad != 0 || s.DecodedPageSegments != 20 {
				t.Fatal("wrong cost workload", scale, v, s)
			}
		}
		runtime.ReadMemStats(&after)
		sorted := slices.Clone(times)
		slices.Sort(sorted)
		var tableBytes, indexBytes, basePayload, extendedPayload, historySources int64
		must(t, f.admin.QueryRow(f.ctx, `SELECT pg_table_size('organization.department_read_segment'),pg_indexes_size('organization.department_read_segment'),sum(pg_column_size(ROW(instance_id,object_id,slot,source_id,rule,span,dictionary_id,shape_id,ids,envelope))),sum(pg_column_size(s)) FROM organization.department_read_segment s`).Scan(&tableBytes, &indexBytes, &basePayload, &extendedPayload))
		must(t, f.admin.QueryRow(f.ctx, `SELECT sum(cardinality(source_refs)) FROM organization.department_history`).Scan(&historySources))
		query131Log(t, "Q131_COST", map[string]any{"objects": scale, "sources": historySources, "currentSegments": stats.Segments, "checkedDictionaries": stats.Dictionaries, "checkedShapes": stats.Shapes, "pageRowsDecoded": stats.DecodedPageSegments, "wireJSONBytes": stats.ResponseBytes, "samplesMs": times, "p50Ms": sorted[9], "p95Ms": sorted[int(math.Ceil(.95*float64(len(sorted))))-1], "tableBytes": tableBytes, "indexBytes": indexBytes, "baseCompositeBytes": basePayload, "extendedCompositeBytes": extendedPayload, "queryColumnsCompositeDeltaBytes": extendedPayload - basePayload, "processAllocatedBytes20Reads": after.TotalAlloc - before.TotalAlloc, "stageAPIWriteSeconds": writesElapsed.Seconds(), "scope": "single session prototype diagnostics; no HTTP read, path, selection, cold reset or capacity acceptance"})
		if scale == 1000 {
			var plan []byte
			must(t, f.admin.QueryRow(f.ctx, "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) "+history.Query131CaptureSQL, q.Args()...).Scan(&plan))
			query131Log(t, "Q131_EXPLAIN", json.RawMessage(plan))
		}
	}
}

// Adopted from the completion review: independent literal dates across source
// chunks, followed by real redate/delete commands and exact-source recovery.
func TestDepartmentQuery131LongTimelineAndWrites(t *testing.T) {
	f, b, q := query131Fixture(t)
	bu := query131BU(t, b, "REVIEW_LONG_BU")
	parent := query131Create(t, b, "LONG_PARENT", "Parent", bu, "2020-01-01")
	body := departmentCreateBody("LONG_TARGET", bu)
	body["name"] = "Review stage 000"
	body["effectiveDate"] = "2021-01-01"
	body["parentDepartmentId"] = parent["departmentId"]
	first := b.call("POST", "/api/v1/departments", body, foundation.NewID(), 201)
	id := first["departmentId"].(string)
	path := "/api/v1/departments/" + id
	head := first
	start := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	var dates, names, records []string
	dates = append(dates, "2021-01-01")
	names = append(names, "Review stage 000")
	records = append(records, first["targetRecordId"].(string))
	for i := 1; i <= 70; i++ {
		day := start.AddDate(0, 0, 10*i).Format("2006-01-02")
		name := fmt.Sprintf("Review stage %03d", i)
		head = b.call("POST", path+"/records", map[string]any{"expectedSourceVersion": head["sourceVersion"], "anchorRecordId": head["targetRecordId"], "effectiveDate": day, "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": name}}, foundation.NewID(), 201)
		dates = append(dates, day)
		names = append(names, name)
		records = append(records, head["targetRecordId"].(string))
	}
	var chunks, sources, members int
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*),sum(cardinality(source_refs)) FROM organization.department_history WHERE object_id=$1`, id).Scan(&chunks, &sources))
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM organization.department_read_segment WHERE object_id=$1`, id).Scan(&members))
	if chunks < 2 || sources != 71 || members < 71 {
		t.Fatal("fixture does not cross long history/chunk boundary", chunks, sources, members)
	}
	assertDay := func(day, name, record string) {
		t.Helper()
		in := q
		in.Day = day
		in.Contains = "Review stage"
		got, stats := query131Read(t, f, in)
		if got.Count != 1 || len(got.Items) != 1 || got.Items[0]["name"] != name || got.Items[0]["record"] != record || got.Items[0]["parent"] != parent["departmentId"] {
			t.Fatal("independent literal timeline mismatch", day, name, record, got, stats)
		}
	}
	assertDay("2020-12-31", names[0], records[0])
	for _, i := range []int{0, 1, 31, 32, 63, 64, 69, 70} {
		assertDay(dates[i], names[i], records[i])
	}
	// A real middle-record redate must move the query boundary; deleting it
	// must expose the predecessor, while the original source stays readable.
	movedDay := start.AddDate(0, 0, 641).Format("2006-01-02")
	selected := b.call("GET", path+"?selectedRecordId="+records[64], nil, "", 200)
	currentRevision := selected["selectedRecord"].(map[string]any)["revision"]
	moved := b.call("POST", path+"/records/"+records[64]+"/redate", map[string]any{"expectedSourceVersion": head["sourceVersion"], "expectedRecordRevision": currentRevision, "newEffectiveDate": movedDay}, foundation.NewID(), 200)
	assertDay(dates[64], names[63], records[63])
	assertDay(movedDay, names[64], records[64])
	b.call("POST", path+"/records/"+records[64]+"/delete", map[string]any{"expectedSourceVersion": moved["sourceVersion"], "expectedRecordRevision": moved["targetRevision"]}, foundation.NewID(), 200)
	assertDay(movedDay, names[63], records[63])
	assertDay(dates[65], names[65], records[65])
	old := b.call("GET", path+"?sourceVersion="+first["sourceVersion"].(string), nil, "", 200)
	if old["selectedRecord"].(map[string]any)["name"] != names[0] {
		t.Fatal("old source changed", old)
	}
	in := q
	in.Contains = "Review stage"
	got, stats := query131Read(t, f, in)
	query131Log(t, "Q131_REVIEW_LONG", map[string]any{"historyChunksBeforeEdits": chunks, "sourcesBeforeEdits": sources, "segmentsBeforeEdits": members, "capturedObjects": stats.Objects, "capturedSegments": stats.Segments, "pageRows": len(got.Items), "wireBytes": stats.ResponseBytes, "checkedDictionaries": stats.Dictionaries, "checkedShapes": stats.Shapes, "literalChecks": 13, "oldSourceReadable": true})
}
