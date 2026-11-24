//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"reflect"
	"testing"
)

// A dictionary inserted before a failing shared shape must roll back together
// with the business source. Recovery under the original identity stays exact.
func TestHistory133DictionaryShapeFailureAtomic(t *testing.T) {
	s := newDepartmentContract(t)
	s.create("C", "部门", "BU_A", "", "2026-01-01", s.b, 201)
	for _, kind := range []string{"bu", "department"} {
		t.Run(kind, func(t *testing.T) {
			id, route, resolve := s.aliases["BU_A"], "/api/v1/business-units/", "/api/v1/organization/submissions/"
			if kind == "department" {
				id, route, resolve = s.aliases["C"], "/api/v1/departments/", "/api/v1/departments/submissions/"
			}
			for _, point := range []string{"dictionary", "shape"} {
				current := s.b.call("GET", route+id, nil, "", 200)
				r := current["timeline"].([]any)[0].(map[string]any)
				body := map[string]any{"expectedSourceVersion": current["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": fmt.Sprintf("原子失败恢复%s%s", kind, point)}}
				key := foundation.NewID()
				before := s.f.physicalOrganizationSnapshot()
				s.f.exec(`CREATE FUNCTION public.dictionary133_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled material failure';END $$`)
				s.f.exec(`CREATE TRIGGER dictionary133_fail BEFORE INSERT ON organization.` + kind + `_` + point + ` FOR EACH ROW EXECUTE FUNCTION public.dictionary133_fail()`)
				s.b.call("POST", route+id+"/records/"+r["recordId"].(string)+"/correct", body, key, 503)
				s.f.exec(`DROP TRIGGER dictionary133_fail ON organization.` + kind + `_` + point)
				s.f.exec(`DROP FUNCTION public.dictionary133_fail()`)
				if !reflect.DeepEqual(before, s.f.physicalOrganizationSnapshot()) {
					t.Fatal("failed material write left a dictionary, shape, history or projection")
				}
				if s.b.call("POST", resolve+key+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
					t.Fatal("failed source became visible")
				}
				// Resolving not_applied closes that original request. A user retry
				// has a new identity; it must not reopen the closed terminal.
				s.b.call("POST", route+id+"/records/"+r["recordId"].(string)+"/correct", body, key, 409)
				key = foundation.NewID()
				accepted := s.b.call("POST", route+id+"/records/"+r["recordId"].(string)+"/correct", body, key, 200)
				after := s.f.physicalOrganizationSnapshot()
				for range 2 {
					receipt := s.b.call("POST", resolve+key+"/resolve", nil, "", 200)
					if receipt["sourceVersion"] != accepted["sourceVersion"] {
						t.Fatal("original identity did not recover the exact source")
					}
				}
				if !reflect.DeepEqual(after, s.f.physicalOrganizationSnapshot()) {
					t.Fatal("recovery appended a duplicate business source")
				}
			}
		})
	}
}
