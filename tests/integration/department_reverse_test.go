//go:build integration

package integration_test

import (
	"fmt"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestDepartmentV16BUAndParentFinalCandidateReverseGuards(t *testing.T) {
	s := newDepartmentContract(t)
	for _, kind := range []string{"bu", "parent"} {
		for i, action := range []string{"first_redate", "correct_first_date", "stop_redate", "resume_redate", "resume_delete", "pair_insert", "append_stop", "reclassify", "pair_delete"} {
			t.Run(kind+"_"+action, func(t *testing.T) {
				s.f.t = t
				bu := s.b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("REV_%s_%d", kind, i), "2026-01-01"), foundation.NewID(), 201)
				id := bu["buId"].(string)
				base := "/api/v1/business-units/" + id
				first := bu["targetRecordId"].(string)
				var parent any
				if kind == "parent" {
					p := s.b.call("POST", "/api/v1/departments", departmentBody(fmt.Sprintf("P%d", i), "上级", id, "2026-01-01", nil), foundation.NewID(), 201)
					parent = p["departmentId"]
					base = "/api/v1/departments/" + parent.(string)
					first = p["targetRecordId"].(string)
				}
				current := func() map[string]any { return s.b.call("GET", base, nil, "", 200) }
				find := func(record string) map[string]any {
					for _, v := range current()["timeline"].([]any) {
						r := v.(map[string]any)
						if r["recordId"] == record {
							return r
						}
					}
					t.Fatal("target record missing")
					return nil
				}
				addLife := func(route, typ, date, anchor, reason string) map[string]any {
					return s.b.call("POST", base+route, map[string]any{"expectedSourceVersion": current()["headSourceVersion"], "anchorRecordId": anchor, "effectiveDate": date, "operationType": typ, "operationReason": reason}, foundation.NewID(), 201)
				}
				stop, resume := "", ""
				if action != "reclassify" {
					stop = addLife("/deactivate", "DEACTIVATE", "2026-06-01", first, "ORG_RESTRUCTURING")["targetRecordId"].(string)
					resume = addLife("/reactivate", "REACTIVATE", "2026-07-01", stop, "BUSINESS_RESUMPTION")["targetRecordId"].(string)
				}
				// Create child before introducing the protected gap. To obey full-period
				// qualification, first shorten parent gap setup after child is established.
				// Here the independent child is first created on a fully active second BU,
				// then a complete three-record schedule is applied before relation changes.
				// A single create on the gapped parent would correctly be refused.
				childBU := id
				if action != "reclassify" {
					childBU = s.aliases["BU_A"]
				}
				childParent := parent
				if action != "reclassify" {
					childParent = nil
				}
				c := s.b.call("POST", "/api/v1/departments", departmentBody(fmt.Sprintf("C_%s_%d", kind, i), "下级", childBU, "2026-01-01", childParent), foundation.NewID(), 201)
				childID := c["departmentId"].(string)
				childFirst := c["targetRecordId"].(string)
				childPath := "/api/v1/departments/" + childID
				if action != "reclassify" {
					cs := s.b.call("POST", childPath+"/deactivate", map[string]any{"expectedSourceVersion": c["sourceVersion"], "anchorRecordId": childFirst, "effectiveDate": "2026-04-01", "operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}, foundation.NewID(), 201)
					cr := s.b.call("POST", childPath+"/reactivate", map[string]any{"expectedSourceVersion": cs["sourceVersion"], "anchorRecordId": cs["targetRecordId"], "effectiveDate": "2026-08-01", "operationType": "REACTIVATE", "operationReason": "BUSINESS_RESUMPTION", "changes": map[string]any{"managementBuId": id, "parentDepartmentId": parent}}, foundation.NewID(), 201)
					_ = cr
					s.b.departmentCorrect(childID, childFirst, map[string]any{"changes": map[string]any{"managementBuId": id, "parentDepartmentId": parent}}, 200)
				}
				beforeChild := s.b.departmentCurrent(childID)["headSourceVersion"]
				beforeSource := current()["headSourceVersion"]
				target := first
				route := ""
				body := map[string]any{"expectedSourceVersion": beforeSource}
				want := 409
				switch action {
				case "first_redate":
					route = "/records/" + first + "/redate"
					body["newEffectiveDate"] = "2026-02-01"
				case "correct_first_date":
					route = "/records/" + first + "/correct"
					body["newEffectiveDate"] = "2026-02-01"
				case "stop_redate":
					target = stop
					route = "/records/" + target + "/redate"
					body["newEffectiveDate"] = "2026-03-01"
				case "resume_redate":
					target = resume
					route = "/records/" + target + "/redate"
					body["newEffectiveDate"] = "2026-09-01"
				case "resume_delete":
					target = resume
					route = "/records/" + target + "/delete"
				case "pair_insert":
					target = ""
					route = "/lifecycle-pairs"
					body["anchorRecordId"] = first
					body["deactivationDate"] = "2026-02-01"
					body["reactivationDate"] = "2026-03-01"
					body["deactivationReason"] = "ORG_RESTRUCTURING"
					body["reactivationReason"] = "BUSINESS_RESUMPTION"
				case "append_stop":
					target = ""
					route = "/deactivate"
					body["anchorRecordId"] = resume
					body["effectiveDate"] = "2026-09-01"
					body["operationType"] = "DEACTIVATE"
					body["operationReason"] = "ORG_RESTRUCTURING"
				case "reclassify":
					change := s.b.call("POST", base+"/records", map[string]any{"expectedSourceVersion": beforeSource, "anchorRecordId": first, "effectiveDate": "2026-06-01", "operationType": "CHANGE", "operationReason": "NAME_ADJUSTMENT", "changes": map[string]any{"name": "新上级名称"}}, foundation.NewID(), 201)
					beforeSource = change["sourceVersion"]
					body["expectedSourceVersion"] = beforeSource
					target = change["targetRecordId"].(string)
					route = "/records/" + target + "/correct"
					body["classification"] = map[string]any{"operationType": "DEACTIVATE", "operationReason": "ORG_RESTRUCTURING"}
				case "pair_delete":
					target = ""
					route = "/lifecycle-pairs/delete"
					body["deactivationRecordId"] = stop
					body["deactivationRevision"] = find(stop)["revision"]
					body["reactivationRecordId"] = resume
					body["reactivationRevision"] = find(resume)["revision"]
					want = 200
				}
				if target != "" {
					body["expectedRecordRevision"] = find(target)["revision"]
				}
				response := s.b.call("POST", base+route, body, foundation.NewID(), want)
				if want == 409 {
					if current()["headSourceVersion"] != beforeSource || response["code"] != "M02_DEPARTMENT_REFERENCE_CONFLICT" && response["code"] != "M02_DEPARTMENT_PARENT_INVALID" {
						t.Fatal("reverse check missing or premature local rejection", response)
					}
				}
				if s.b.departmentCurrent(childID)["headSourceVersion"] != beforeChild {
					t.Fatal("reverse validation mutated child")
				}
			})
		}
	}
	s.f.t = t
}
