//go:build integration

package bootstrap_test

import (
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentReadIndexMissingCorruptAndAtomicWrite(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("READ_SEG_BU", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	created := b.call("POST", "/api/v1/departments", departmentCreateBody("READ_SEG_DEPT", bu), foundation.NewID(), 201)
	id := created["departmentId"].(string)
	child := departmentCreateBody("READ_SEG_CHILD", bu)
	child["parentDepartmentId"] = id
	b.call("POST", "/api/v1/departments", child, foundation.NewID(), 201)
	source := created["sourceVersion"].(string)
	path := "/api/v1/departments/" + id
	f.exec(`CREATE TABLE public.saved_read_index AS SELECT * FROM organization.department_read_segment`)
	var columns string
	must(t, f.admin.QueryRow(f.ctx, `SELECT string_agg(quote_ident(attname),',' ORDER BY attnum) FROM pg_attribute WHERE attrelid='organization.department_read_segment'::regclass AND attnum>0 AND NOT attisdropped AND attgenerated=''`).Scan(&columns))
	freshCache := func() {
		old := f.runtime.Department.(department.Service).Store.(*orgdb.Store)
		f.runtime.Department = department.Service{Store: &orgdb.Store{Pool: old.Pool, Instance: old.Instance, Clock: old.Clock, RecordedClock: old.RecordedClock, Guard: old.Guard}}
	}
	// The query must fail for its actual child/ancestor dependency. Arbitrary
	// envelope tampering is covered by the separate tagged storage diagnostic.
	for _, damage := range []string{
		`DELETE FROM organization.department_read_segment WHERE object_id=$1`,
		`UPDATE organization.department_read_segment SET ids[1]=gen_random_uuid() WHERE object_id=$1`,
		`UPDATE organization.department_read_segment SET source_id=gen_random_uuid() WHERE object_id=$1`,
	} {
		f.exec(damage, id)
		freshCache()
		status, value, err := b.response("GET", "/api/v1/departments?q=read_seg_child", nil, "")
		must(t, err)
		if status != 503 || value["items"] != nil {
			t.Fatal("missing or corrupt required ancestor returned a collection", status, value)
		}
		// The exact immutable source has its own history path and current read
		// authority; a missing derivative must not erase or replace that source.
		b.call("GET", path+"?sourceVersion="+source, nil, "", 200)
		f.exec(`DELETE FROM organization.department_read_segment;INSERT INTO organization.department_read_segment (` + columns + `) SELECT ` + columns + ` FROM public.saved_read_index`)
	}
	for _, invalid := range []struct{ query, code string }{
		{`UPDATE organization.department_read_segment SET rule=2`, "23514"},
		{`INSERT INTO organization.department_read_segment (` + columns + `) SELECT ` + columns + ` FROM (SELECT (jsonb_populate_record(NULL::organization.department_read_segment,to_jsonb(s)||jsonb_build_object('slot',2147483647))).* FROM organization.department_read_segment s WHERE slot=1) duplicate`, "23P01"},
	} {
		_, err := f.admin.Exec(f.ctx, invalid.query)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != invalid.code {
			t.Fatal("wrong rule or overlapping span did not reach its native constraint", err)
		}
	}
	freshCache()
	b.call("GET", "/api/v1/departments", nil, "", 200)
	var before int64
	must(t, f.admin.QueryRow(f.ctx, `SELECT sum(cardinality(source_refs)) FROM organization.department_history`).Scan(&before))
	f.exec(`CREATE FUNCTION public.fail_read_index() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled read-index insertion failure' USING ERRCODE='23514'; END $$; CREATE TRIGGER fail_read_index BEFORE INSERT ON organization.department_read_segment FOR EACH ROW EXECUTE FUNCTION public.fail_read_index()`)
	body := map[string]any{"expectedSourceVersion": source, "expectedRecordRevision": created["targetRevision"], "changes": map[string]any{"name": "索引与来源共同提交"}}
	b.call("POST", path+"/records/"+created["targetRecordId"].(string)+"/correct", body, foundation.NewID(), 503)
	current := b.call("GET", path, nil, "", 200)
	var after, difference int64
	must(t, f.admin.QueryRow(f.ctx, `SELECT sum(cardinality(source_refs)) FROM organization.department_history`).Scan(&after))
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM ((SELECT * FROM organization.department_read_segment EXCEPT SELECT * FROM public.saved_read_index) UNION ALL (SELECT * FROM public.saved_read_index EXCEPT SELECT * FROM organization.department_read_segment)) delta`).Scan(&difference))
	if current["headSourceVersion"] != source || before != after || difference != 0 {
		t.Fatal("failed index insertion committed business facts", before, after, difference)
	}
	f.exec(`DROP TRIGGER fail_read_index ON organization.department_read_segment;DROP FUNCTION public.fail_read_index()`)
	accepted := b.call("POST", path+"/records/"+created["targetRecordId"].(string)+"/correct", body, foundation.NewID(), 200)
	var mismatches int
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM organization.department_read_segment s JOIN organization.department d ON d.instance_id=s.instance_id AND d.department_id=s.object_id WHERE s.source_id<>d.head_source_id`).Scan(&mismatches))
	if mismatches != 0 || accepted["sourceVersion"] == source {
		t.Fatal("successful source has a stale index")
	}
	freshCache()
	b.call("GET", "/api/v1/departments", nil, "", 200)
}
