//go:build integration

package integration_test

import (
	"fmt"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestFirstRecordIndependentCorrectionAndAddition(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	for _, later := range []bool{false, true} {
		t.Run(fmt.Sprintf("successors_%t", later), func(t *testing.T) {
			input := createBody(fmt.Sprintf("FIRST_STEPS_%t", later), "2026-06-01")
			input["name"], input["nameEn"] = "乙", "Beta"
			id := b.call("POST", "/api/v1/business-units", input, foundation.NewID(), 201)["buId"].(string)
			path := "/api/v1/business-units/" + id
			if later {
				lifeSave(b, id, "2026-07-01", "DEACTIVATE", nil)
				lifeSave(b, id, "2026-08-01", "REACTIVATE", map[string]any{"name": "丙", "nameEn": "Gamma"})
				b.call("POST", path+"/records", addBody(detail(b, id), "2026-10-01", map[string]any{"name": "丁"}), foundation.NewID(), 201)
			}
			// Both ordinary operations must work without a joint-entry grant.
			mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.timeline.revise'`, b.subject)
			original := detail(b, id)
			first := recordOn(t, original, "2026-06-01")
			sources, events := f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history"), f.eventCount()
			correction := correctBody(original, first, map[string]any{"name": "甲", "nameEn": "Alpha"})
			correction["newEffectiveDate"] = "2026-01-01"
			key1 := foundation.NewID()
			corrected := b.call("POST", path+"/records/"+first["recordId"].(string)+"/correct", correction, key1, 200)
			intermediate := detail(b, id)
			if len(intermediate["timeline"].([]any)) != len(original["timeline"].([]any)) || recordOn(t, intermediate, "2026-01-01")["recordId"] != first["recordId"] || recordOn(t, intermediate, "2026-01-01")["operationType"] != "CREATE" {
				t.Fatal("correction created an extra original-date node", intermediate)
			}
			for _, row := range intermediate["timeline"].([]any) {
				if row.(map[string]any)["effectiveDate"] == "2026-06-01" {
					t.Fatal("old date copied implicitly")
				}
			}
			if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != sources+1 || f.eventCount() != events+1 {
				t.Fatal("first step not one independent source/event")
			}
			if got := b.getOnDay("2026-06-01", path)["viewFact"].(map[string]any)["name"]; got != "甲" {
				t.Fatal("first saved result was hidden", got)
			}
			current := b.call("GET", path+"?selectedRecordId="+first["recordId"].(string), nil, "", 200)
			mode := "APPEND"
			if later {
				mode = "INSERT"
				if recordOn(t, intermediate, "2026-07-01")["name"] != "甲" {
					t.Fatal("intermediate stop not inherited")
				}
			}
			if current["additionContext"].(map[string]any)["mode"] != mode {
				t.Fatal(current)
			}
			// Rejection and no-change in the optional second step cannot undo step one.
			b.call("POST", path+"/records", addBody(intermediate, "2026-06-01", map[string]any{"name": "😀"}), foundation.NewID(), 400)
			b.call("POST", path+"/records", addBody(intermediate, "2026-06-01", map[string]any{}), foundation.NewID(), 200)
			var grant string
			var writes []string
			must(t, f.admin.QueryRow(f.ctx, `SELECT grant_id,write_fields FROM access.subject_grant WHERE principal_id=$1 AND action='bu.record.add' LIMIT 1`, b.subject).Scan(&grant, &writes))
			mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET write_fields='{}' WHERE grant_id=$1`, grant)
			b.call("POST", path+"/records", addBody(intermediate, "2026-06-01", map[string]any{"name": "乙", "nameEn": "Beta"}), foundation.NewID(), 403)
			if !reflect.DeepEqual(detail(b, id)["timeline"], intermediate["timeline"]) || detail(b, id)["headSourceVersion"] != corrected["sourceVersion"] {
				t.Fatal("failed second step changed first result")
			}
			mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET write_fields=$2 WHERE grant_id=$1`, grant, writes)
			addition := addBody(intermediate, "2026-06-01", map[string]any{"name": "乙", "nameEn": "Beta"})
			key2 := foundation.NewID()
			added := b.call("POST", path+"/records", addition, key2, 201)
			final := detail(b, id)
			if added["sourceVersion"] == corrected["sourceVersion"] || added["targetRecordId"] == first["recordId"] || recordOn(t, final, "2026-06-01")["name"] != "乙" || f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != sources+2 || f.eventCount() != events+2 {
				t.Fatal("second step did not commit separately", added)
			}
			if later && (recordOn(t, final, "2026-07-01")["name"] != "乙" || recordOn(t, final, "2026-08-01")["name"] != "丙" || recordOn(t, final, "2026-10-01")["name"] != "丁") {
				t.Fatal("successor snapshots changed")
			}
			for _, snapshot := range []map[string]any{original, intermediate} {
				if !reflect.DeepEqual(b.call("GET", path+"?sourceVersion="+snapshot["headSourceVersion"].(string), nil, "", 200)["timeline"], snapshot["timeline"]) {
					t.Fatal("saved source changed")
				}
			}
			for _, step := range []struct {
				endpoint string
				body     map[string]any
				key      string
				status   int
				source   any
			}{{path + "/records/" + first["recordId"].(string) + "/correct", correction, key1, 200, corrected["sourceVersion"]}, {path + "/records", addition, key2, 201, added["sourceVersion"]}} {
				if replay := b.call("POST", step.endpoint, step.body, step.key, step.status); replay["sourceVersion"] != step.source {
					t.Fatal("replay rebound original result", replay)
				}
				if resolved := b.call("POST", "/api/v1/organization/submissions/"+step.key+"/resolve", nil, "", 200); resolved["sourceVersion"] != step.source {
					t.Fatal("resolve rebound original result", resolved)
				}
			}
		})
	}
}
