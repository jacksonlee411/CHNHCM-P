//go:build integration

package integration_test

import (
	"fmt"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestT07DeleteCurrentClassificationAndCrossCategoryGrants(t *testing.T) {
	f := fresh(t)
	admin := f.login("admin", f.password)
	reader := f.login("reader", f.readPassword)
	actions := []string{"bu.record.delete", "bu.deactivation.delete", "bu.reactivation.delete"}
	for i, kind := range []string{"CHANGE", "DEACTIVATE", "REACTIVATE"} {
		id := admin.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("T07_DELETE_%d", i), "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
		path := "/api/v1/business-units/" + id
		if kind == "CHANGE" {
			admin.call("POST", path+"/records", addBody(detail(admin, id), "2026-06-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
		} else {
			lifeSave(admin, id, "2026-04-01", "DEACTIVATE", nil)
			if kind == "REACTIVATE" {
				lifeSave(admin, id, "2026-06-01", "REACTIVATE", nil)
			}
		}
		d := detail(reader, id)
		rs := d["timeline"].([]any)
		target := rs[len(rs)-1].(map[string]any)
		for j, action := range actions {
			if j == i {
				continue
			}
			mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action NOT IN ('bu.read','bu.history.read')`, reader.subject)
			mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),$1,'bu',$2,action,read_fields,write_fields FROM access.subject_grant WHERE principal_id=$3 AND action=$4`, reader.subject, id, admin.subject, action)
			before := businessSnapshot(t, f)
			reader.call("POST", path+"/records/"+target["recordId"].(string)+"/delete", deleteBody(d, target), foundation.NewID(), 403)
			for table, n := range businessSnapshot(t, f) {
				if n != before[table] {
					t.Fatal("cross-category denial changed facts", table)
				}
			}
		}
	}
	// TL-170: category is taken from the currently accepted revision, not the
	// initial record. A CHANGE corrected to DEACTIVATE needs stop deletion rights.
	id := admin.call("POST", "/api/v1/business-units", createBody("T07_CURRENT_TYPE", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	admin.call("POST", path+"/records", addBody(detail(admin, id), "2026-06-01", map[string]any{"name": "乙"}), foundation.NewID(), 201)
	d := detail(admin, id)
	target := recordOn(t, d, "2026-06-01")
	u := correctBody(d, target, nil)
	u["classification"] = map[string]any{"operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}
	admin.call("POST", path+"/records/"+target["recordId"].(string)+"/correct", u, foundation.NewID(), 200)
	d = detail(reader, id)
	target = recordOn(t, d, "2026-06-01")
	for _, action := range []string{"bu.record.delete", "bu.deactivation.delete"} {
		mutateLifeGrants(f, reader.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action NOT IN ('bu.read','bu.history.read')`, reader.subject)
		mutateLifeGrants(f, reader.subject, `INSERT INTO access.subject_grant SELECT instance_id,gen_random_uuid(),$1,'bu',$2,action,read_fields,write_fields FROM access.subject_grant WHERE principal_id=$3 AND action=$4`, reader.subject, id, admin.subject, action)
		status := 403
		if action == "bu.deactivation.delete" {
			status = 200
		}
		reader.call("POST", path+"/records/"+target["recordId"].(string)+"/delete", deleteBody(d, target), foundation.NewID(), status)
	}
}

func TestT07OriginalStopRedateAfterClassificationOrReplacement(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(fmt.Sprintf("replacement_%t", replacement), func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			id := b.call("POST", "/api/v1/business-units", createBody("T07_ORIGINAL_TYPE", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
			path := "/api/v1/business-units/" + id
			lifeSave(b, id, "2026-04-01", "DEACTIVATE", nil)
			d := detail(b, id)
			s := recordOn(t, d, "2026-04-01")
			input := redateBody(d, s, "2026-05-01")
			key := foundation.NewID()
			endpoint := redatePath(id, s)
			saved := b.call("POST", endpoint, input, key, 200)
			d = detail(b, id)
			s = recordOn(t, d, "2026-05-01")
			if replacement {
				deleteRecord(b, id, s)
				b.call("POST", path+"/records", addBody(detail(b, id), "2026-05-01", map[string]any{"name": "替换资料"}), foundation.NewID(), 201)
			} else {
				u := correctBody(d, s, nil)
				u["classification"] = map[string]any{"operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT"}
				u["changes"] = map[string]any{"name": "分类纠错资料"}
				b.call("POST", path+"/records/"+s["recordId"].(string)+"/correct", u, foundation.NewID(), 200)
			}
			mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action IN ('bu.deactivation.redate','bu.deactivation.delete','bu.deactivation.reclassify','bu.deactivate')`, b.subject)
			for _, v := range []map[string]any{b.call("POST", endpoint, input, key, 200), b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
				if v["targetRevision"] != saved["targetRevision"] || v["sourceVersion"] != saved["sourceVersion"] || v["previousEffectiveDate"] != "2026-04-01" || v["effectiveDate"] != "2026-05-01" {
					t.Fatal("old stop result drifted", v)
				}
			}
			old := b.call("GET", path+"?sourceVersion="+saved["sourceVersion"].(string), nil, "", 200)
			if recordOn(t, old, "2026-05-01")["operationType"] != "DEACTIVATE" {
				t.Fatal("old category rewritten")
			}
			d = detail(b, id)
			current := recordOn(t, d, "2026-05-01")
			if current["operationType"] != "CHANGE" {
				t.Fatal(current)
			}
			b.call("POST", redatePath(id, current), redateBody(d, current, "2026-06-01"), foundation.NewID(), 200)
		})
	}
}
