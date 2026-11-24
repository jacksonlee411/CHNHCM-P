//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"reflect"
	"testing"
)

func TestT07CreateFactFailureAndReadWithoutSecurityEventDependency(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	f.exec(`CREATE FUNCTION public.t07_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled persistence failure'; END $$`)
	for _, table := range []string{"organization.bu_history"} {
		t.Run(table, func(t *testing.T) {
			in := createBody("T07_CREATE_FAULT", "2026-01-01")
			key := foundation.NewID()
			before := businessSnapshot(t, f)
			f.exec("CREATE TRIGGER t07_fail BEFORE INSERT ON " + table + " FOR EACH ROW EXECUTE FUNCTION public.t07_fail()")
			b.call("POST", "/api/v1/business-units", in, key, 503)
			f.exec("DROP TRIGGER t07_fail ON " + table)
			if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
				t.Fatal("failed create left partial business/terminal")
			}
			if b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)["outcome"] != "not_applied" {
				t.Fatal("failed create identity not closed")
			}
			b.call("POST", "/api/v1/business-units", in, key, 409)
		})
	}
	// A failed optional security-event insert must not be a dependency of reads.
	saved := b.call("POST", "/api/v1/business-units", createBody("T07_READ_NO_AUDIT", "2026-01-01"), foundation.NewID(), 201)
	f.exec(`CREATE TRIGGER t07_fail BEFORE INSERT ON access.security_event FOR EACH ROW EXECUTE FUNCTION public.t07_fail()`)
	out := b.call("GET", "/api/v1/business-units/"+saved["buId"].(string), nil, "", 200)
	f.exec(`DROP TRIGGER t07_fail ON access.security_event`)
	if out["selectedRecord"] == nil || out["timeline"] == nil {
		t.Fatal("authorized read depended on a security event", out)
	}

}

func TestT07LifecycleDedicatedDTOAndCreateEvent(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	saved := b.call("POST", "/api/v1/business-units", createBody("T07_FROZEN_DTO", "2026-01-01"), foundation.NewID(), 201)
	id := saved["buId"].(string)
	path := "/api/v1/business-units/" + id
	d := detail(b, id)
	first := recordOn(t, d, "2026-01-01")
	event := f.buSource(saved["sourceVersion"].(string)).Operation.Event()
	version := event.SchemaVersion
	raw, err := json.Marshal(event.Impact)
	must(t, err)
	var impact map[string]any
	must(t, json.Unmarshal(raw, &impact))
	wantRanges := []any{map[string]any{"start": "2026-01-01", "endExclusive": nil}}
	wantStructure := []any{map[string]any{"recordId": first["recordId"], "beforeRevisionId": nil, "afterRevisionId": first["revision"], "facets": []any{"ADD"}}}
	if version != 2 || !reflect.DeepEqual(impact["dataAffectedRanges"], wantRanges) || !reflect.DeepEqual(impact["structuralChanges"], wantStructure) {
		t.Fatal("CREATE exact event", version, impact)
	}
	stop := lifeBody(d, "2026-04-01", "DEACTIVATE", nil)
	stop["nameEn"] = "forged"
	b.call("POST", path+"/deactivate", stop, foundation.NewID(), 400)
	corr := correctBody(d, first, map[string]any{"name": "更正"})
	delete(corr, "changes")
	b.call("POST", path+"/records/"+first["recordId"].(string)+"/correct", corr, foundation.NewID(), 400)
	lifeSave(b, id, "2026-04-01", "DEACTIVATE", nil)
	d = detail(b, id)
	before := businessSnapshot(t, f)
	b.call("POST", path+"/reactivate", lifeBody(d, "2026-05-01", "REACTIVATE", map[string]any{"name": nil}), foundation.NewID(), 400)
	for table, v := range businessSnapshot(t, f) {
		if table != "terminals" && v != before[table] {
			t.Fatal("bad DTO wrote fact", table)
		}
	}
}

func TestT07ExactReadonlyScopeAndJointFieldPermissions(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	in := createBody("T07_SCOPE_ONE", "2026-01-01")
	in["nameEn"] = "Alpha"
	id := b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-03-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
	d := detail(b, id)
	r := recordOn(t, d, "2026-03-01")
	reader.call("POST", redatePath(id, r), redateBody(d, r, "2026-03-02"), foundation.NewID(), 403)
	lifeSave(b, id, "2026-04-01", "DEACTIVATE", nil)
	d = detail(b, id)
	reader.call("POST", path+"/reactivate", lifeBody(d, "2026-05-01", "REACTIVATE", nil), foundation.NewID(), 403)
	other := b.call("POST", "/api/v1/business-units", createBody("T07_SCOPE_TWO", "2026-01-01"), foundation.NewID(), 201)
	mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET scope_kind='bu',scope_id=$2 WHERE principal_id=$1 AND action LIKE 'bu.%'`, reader.subject, id)
	list := reader.call("GET", "/api/v1/business-units", nil, "", 200)
	if list["count"] != float64(1) || len(list["items"].([]any)) != 1 || list["items"].([]any)[0].(map[string]any)["id"] != id {
		t.Fatal("scope count/items leaked", list)
	}
	hidden := "/api/v1/business-units/" + other["buId"].(string)
	reader.call("GET", hidden, nil, "", 404)
	reader.call("GET", hidden+"?sourceVersion="+other["sourceVersion"].(string), nil, "", 404)
	// Ordinary correction needs only its actual permissions; no generic entry grant.
	mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.timeline.revise'`, b.subject)
	r = recordOn(t, d, "2026-03-01")
	it := correctBody(d, r, nil)
	it["changes"] = map[string]any{"name": "丙"}
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", it, foundation.NewID(), 200)
	mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'nameEn') WHERE principal_id=$1 AND action LIKE 'bu.%'`, b.subject)
	d = detail(b, id)
	r = recordOn(t, d, "2026-03-01")
	before := businessSnapshot(t, f)
	for _, en := range []any{"Alpha", nil} {
		it = correctBody(d, r, nil)
		it["changes"] = map[string]any{"name": "丁", "nameEn": en}
		b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", it, foundation.NewID(), 403)
	}
	for table, v := range businessSnapshot(t, f) {
		if table != "terminals" && v != before[table] {
			t.Fatal("forbidden explicit English wrote fact", table)
		}
	}
	it = correctBody(d, r, nil)
	it["changes"] = map[string]any{"name": "丁"}
	b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", it, foundation.NewID(), 200)
	got := recordOn(t, detail(b, id), "2026-03-01")
	if got["name"] != "丁" || got["nameEn"] != "Alpha" {
		t.Fatal("omitted English changed", got)
	}
}
