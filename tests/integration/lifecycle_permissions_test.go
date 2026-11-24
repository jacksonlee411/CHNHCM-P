//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func mutateLifeGrants(f *fixture, subject, query string, args ...any) {
	tx, e := f.admin.Begin(f.ctx)
	must(f.t, e)
	defer tx.Rollback(f.ctx)
	_, e = tx.Exec(f.ctx, "SELECT principal_id FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE", f.instance, subject)
	must(f.t, e)
	_, e = tx.Exec(f.ctx, query, args...)
	must(f.t, e)
	_, e = tx.Exec(f.ctx, "UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2", f.instance, subject)
	must(f.t, e)
	must(f.t, tx.Commit(f.ctx))
}

func TestLifecycleDependencyDiagnosticsRespectDateReadPermission(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("HIDDEN_DEPENDENCY", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	b.call("POST", path+"/records", addBody(detail(b, id), "2026-10-01", map[string]any{"name": "后继"}), foundation.NewID(), 201)
	mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'effectiveDate') WHERE instance_id=$1 AND principal_id=$2 AND action IN ('bu.read','bu.history.read')`, f.instance, b.subject)
	for _, preview := range []bool{true, false} {
		request := lifeBody(detail(b, id), "2026-03-01", "DEACTIVATE", nil)
		endpoint := path + "/deactivate"
		if preview {
			request["action"] = "DEACTIVATE"
			endpoint = path + "/preview"
		}
		response := b.call("POST", endpoint, request, foundation.NewID(), 409)
		raw, _ := json.Marshal(response)
		if strings.Contains(string(raw), "2026-10-01") || response["field"] != nil && response["field"] != "" {
			t.Fatal("restricted dependency leaked", response)
		}
	}
}
func TestLifecycleDedicatedPermissionsFieldsAndReadableOriginalResult(t *testing.T) {
	f := fresh(t)
	admin := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	body := createBody("LIFE_FIELDS", "2026-01-01")
	body["name"] = "甲"
	body["nameEn"] = "Alpha"
	id := admin.call("POST", "/api/v1/business-units", body, foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	// Read/create/ordinary correction are never lifecycle permission.
	reader.call("POST", path+"/deactivate", lifeBody(detail(reader, id), "2026-04-01", "DEACTIVATE", nil), foundation.NewID(), 403)
	for _, action := range []string{"bu.deactivate", "bu.reactivate"} {
		mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'bu',$4,$5,ARRAY[]::text[],ARRAY['effectiveDate','operationType','operationReason'])`, f.instance, foundation.NewID(), reader.subject, id, action)
	}
	lifeSave(reader, id, "2026-04-01", "DEACTIVATE", nil)
	d := detail(reader, id)
	if d["capabilities"].(map[string]any)["correct"] != false || d["readOnly"] != false {
		t.Fatal("lifecycle actor gained correction")
	}
	// Same-value explicit English still needs write permission.
	reader.call("POST", path+"/reactivate", lifeBody(d, "2026-05-01", "REACTIVATE", map[string]any{"nameEn": "Alpha"}), foundation.NewID(), 403)
	same := lifeSave(reader, id, "2026-05-01", "REACTIVATE", nil)
	if same["outcome"] != "accepted" {
		t.Fatal("same-name resume is a business fact", same)
	}
	lifeSave(reader, id, "2026-06-01", "DEACTIVATE", nil)
	mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET write_fields=array_append(write_fields,'name') WHERE instance_id=$1 AND principal_id=$2 AND action='bu.reactivate'`, f.instance, reader.subject)
	mutateLifeGrants(f, reader.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE instance_id=$1 AND principal_id=$2 AND action IN ('bu.read','bu.history.read')`, f.instance, reader.subject)
	d = detail(reader, id)
	in := lifeBody(d, "2026-07-01", "REACTIVATE", map[string]any{"name": "新"})
	preview := lifeBody(d, "2026-07-01", "REACTIVATE", nil)
	preview["action"] = "REACTIVATE"
	defaults := reader.call("POST", path+"/preview", preview, "", 200)
	if defaults["defaults"] != nil || defaults["timeline"] != nil || defaults["beforeRecord"] != nil || defaults["targetPreviousState"] != nil || defaults["successorSnapshotsPreserved"] != nil || defaults["writeFields"] != nil || defaults["referenceCoverage"] != nil {
		t.Fatal("hidden default exposed", defaults)
	}
	key := foundation.NewID()
	saved := reader.call("POST", path+"/reactivate", in, key, 201)
	fact := detail(admin, id)["selectedRecord"].(map[string]any)
	if fact["name"] != "新" || fact["nameEn"] != "Alpha" {
		t.Fatal("hidden inherited field damaged", fact)
	}
	// Snapshot the business terminal, revoke write, and recover under current read rights.
	terminal, e := bootstrap.InspectBUResult(f.ctx, f.admin, f.instance, reader.subject, key)
	must(t, e)

	mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='bu.reactivate'`, f.instance, reader.subject)
	replay := reader.call("POST", path+"/reactivate", in, key, 201)
	recovered := reader.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	if replay["sourceVersion"] != saved["sourceVersion"] || recovered["sourceVersion"] != saved["sourceVersion"] {
		t.Fatal("revoked writer cannot read original")
	}
	reader.call("POST", path+"/reactivate", lifeBody(detail(reader, id), "2026-08-01", "REACTIVATE", nil), foundation.NewID(), 403)
	after, e := bootstrap.InspectBUResult(f.ctx, f.admin, f.instance, reader.subject, key)
	must(t, e)
	if !reflect.DeepEqual(terminal, after) {
		t.Fatal("original business terminal changed")
	}

	mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='bu.history.read'`, f.instance, reader.subject)
	reader.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 403)
	// A second BU is not covered by the first BU's lifecycle grants.
	other := admin.call("POST", "/api/v1/business-units", createBody("LIFE_OTHER", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	reader.call("POST", "/api/v1/business-units/"+other+"/deactivate", lifeBody(detail(reader, other), "2026-04-01", "DEACTIVATE", nil), foundation.NewID(), 403)
	// Retaining create/add/correct rights cannot substitute for either dedicated action.
	mutateLifeGrants(f, admin.subject, `DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action IN ('bu.deactivate','bu.reactivate')`, f.instance, admin.subject)
	d = detail(admin, id)
	capabilities := d["capabilities"].(map[string]any)
	if capabilities["add"] != true || capabilities["correct"] != true || capabilities["deactivate"] != false || capabilities["reactivate"] != false {
		t.Fatal("ordinary rights expanded to lifecycle", capabilities)
	}
	for _, action := range []string{"DEACTIVATE", "REACTIVATE"} {
		admin.call("POST", path+"/"+strings.ToLower(action), lifeBody(d, "2026-08-01", action, nil), foundation.NewID(), 403)
		forged := lifeBody(d, "2026-08-01", action, map[string]any{"name": "伪造动作"})
		admin.call("POST", path+"/records", forged, foundation.NewID(), 400)
		forged["action"] = "ADD"
		admin.call("POST", path+"/preview", forged, "", 400)
	}
}
