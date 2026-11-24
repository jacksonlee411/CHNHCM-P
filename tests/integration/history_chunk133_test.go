//go:build integration

package integration_test

import (
	"fmt"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestHistory133LoadedChunkRolloverAndRollback(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("C", "部门", "BU_A", "", "2026-01-01", s.b, 201)
	facts := func() map[string]any {
		out := map[string]any{}
		h := s.f.history()
		for _, v := range h.BUs {
			out[v.Operation.Source.ID] = v
		}
		for _, v := range h.Departments {
			out[v.Operation.Source.ID] = v
		}
		return out
	}
	for _, kind := range []string{"bu", "department"} {
		t.Run(kind, func(t *testing.T) {
			id, route := s.aliases["BU_A"], "/api/v1/business-units/"
			resolve := "/api/v1/organization/submissions/"
			if kind == "department" {
				id, route, resolve = s.aliases["C"], "/api/v1/departments/", "/api/v1/departments/submissions/"
			}
			original := facts()
			var last map[string]any
			write := func(number, status int) string {
				t.Helper()
				current := s.b.call("GET", route+id, nil, "", 200)
				r := current["timeline"].([]any)[0].(map[string]any)
				key := foundation.NewID()
				last = s.b.call("POST", route+id+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": current["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": fmt.Sprintf("块边界%d", number)}}, key, status)
				return key
			}
			for n := 2; n <= 64; n++ {
				write(n, 200)
			}
			var chunks, sources int
			must(t, s.f.admin.QueryRow(s.f.ctx, `SELECT count(*),sum(cardinality(source_refs)) FROM organization.`+kind+`_history WHERE object_id=$1`, id).Scan(&chunks, &sources))
			if chunks != 1 || sources != 64 {
				t.Fatal("fixture must reach the exact bounded directory", chunks, sources)
			}
			before := s.f.physicalOrganizationSnapshot()
			s.f.exec(`CREATE FUNCTION public.chunk133_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled rollover failure'; END $$`)
			s.f.exec(`CREATE TRIGGER chunk133_fail BEFORE INSERT ON organization.` + kind + `_history FOR EACH ROW WHEN (NEW.chunk_no>1) EXECUTE FUNCTION public.chunk133_fail()`)
			failed := write(65, 503)
			s.f.exec(`DROP TRIGGER chunk133_fail ON organization.` + kind + `_history`)
			s.f.exec(`DROP FUNCTION public.chunk133_fail()`)
			if !reflect.DeepEqual(before, s.f.physicalOrganizationSnapshot()) {
				t.Fatal("failed rollover left a sealed prefix or partial source")
			}
			if s.b.call("POST", resolve+failed+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
				t.Fatal("failed rollover lost its original identity")
			}
			for n := 65; n <= 66; n++ {
				key := write(n, 200)
				if s.b.call("POST", resolve+key+"/resolve", nil, "", 200)["sourceVersion"] != last["sourceVersion"] {
					t.Fatal("rollover receipt did not bind the actual new source")
				}
			}
			var valid bool
			must(t, s.f.admin.QueryRow(s.f.ctx, `SELECT count(*)=2 AND sum(cardinality(source_refs))=66 AND bool_and(CASE WHEN chunk_no=1 THEN sealed AND cardinality(source_refs)=64 ELSE first_no=65 AND cardinality(source_refs)=2 END) FROM organization.`+kind+`_history WHERE object_id=$1`, id).Scan(&valid))
			if !valid {
				t.Fatal("new chunk did not preserve the bounded contiguous prefix")
			}
			after := facts()
			for key, value := range original {
				if !reflect.DeepEqual(after[key], value) {
					t.Fatal("original complete history changed", key)
				}
			}
		})
	}
}
