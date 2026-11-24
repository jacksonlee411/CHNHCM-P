//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestT07EveryTerminalReadRevocationAndRestoration(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("T07_READ_TERMINALS", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id + "/records/" + recordOn(t, detail(b, id), "2026-01-01")["recordId"].(string) + "/correct"
	f.exec(`CREATE TABLE access.t07_saved_grant AS SELECT * FROM access.subject_grant WHERE principal_id=$1`, b.subject)
	restore := func() {
		mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1`, b.subject)
		mutateLifeGrants(f, b.subject, `INSERT INTO access.subject_grant SELECT * FROM access.t07_saved_grant`)
	}
	type terminal struct {
		key             string
		input, original map[string]any
		status          int
	}
	terminals := []terminal{}
	for _, outcome := range []string{"accepted", "no_change", "rejected"} {
		d := detail(b, id)
		r := recordOn(t, d, "2026-01-01")
		i := correctBody(d, r, nil)
		en := "ProtectedEnglish"
		status := 200
		if outcome == "rejected" {
			en = "非法😀英文"
			status = 400
		}
		i["changes"] = map[string]any{"nameEn": en}
		in := i
		key := foundation.NewID()
		saved := b.call("POST", path, in, key, status)
		if saved["outcome"] != outcome {
			t.Fatal(outcome, saved)
		}
		original := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
		terminals = append(terminals, terminal{key, in, original, status})
	}
	before := businessSnapshot(t, f)
	for _, mode := range []string{"English", "object", "history"} {
		t.Run(mode, func(t *testing.T) {
			restore()
			// Reading an existing terminal never re-requires its original write grants.
			mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action NOT IN ('bu.read','bu.history.read')`, b.subject)
			switch mode {
			case "English":
				mutateLifeGrants(f, b.subject, `UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'nameEn') WHERE principal_id=$1`, b.subject)
			case "object":
				mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.read'`, b.subject)
			case "history":
				mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='bu.history.read'`, b.subject)
			}
			for _, v := range terminals {
				for _, resolve := range []bool{false, true} {
					route, in, key := path, any(v.input), v.key
					if resolve {
						route = "/api/v1/organization/submissions/" + v.key + "/resolve"
						in = nil
						key = ""
					}
					status, out, e := b.response("POST", route, in, key)
					must(t, e)
					if mode == "English" {
						want := v.status
						if resolve {
							want = 200
						}
						if status != want || out["outcome"] != v.original["outcome"] {
							t.Fatal(mode, status, out)
						}
						raw, _ := json.Marshal(out)
						if strings.Contains(string(raw), "ProtectedEnglish") || strings.Contains(string(raw), "nameEn") || strings.Contains(string(raw), "非法😀英文") {
							t.Fatal("revoked field or location leaked", string(raw))
						}
					} else if status != 403 && status != 404 {
						t.Fatal("lost read right was treated as confirmed result", status, out)
					}
				}
			}
			restore()
			// Restore read grants while keeping writing revoked: the exact original
			// safe result must become readable again, including original diagnosis.
			mutateLifeGrants(f, b.subject, `DELETE FROM access.subject_grant WHERE principal_id=$1 AND action NOT IN ('bu.read','bu.history.read')`, b.subject)
			for _, v := range terminals {
				got := b.call("POST", "/api/v1/organization/submissions/"+v.key+"/resolve", nil, "", 200)
				if !reflect.DeepEqual(got, v.original) {
					t.Fatal("did not recover original terminal", got, v.original)
				}
			}
		})
	}
	if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
		t.Fatal("read authorization mutated business/terminal evidence")
	}
}

func TestT07CorruptDiagnosticsKeepOriginalIdentity(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("T07_BAD_DIAG_BASE", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	for _, action := range []string{"CREATE", "CORRECT"} {
		path := "/api/v1/business-units"
		in := createBody("T07_BAD_DIAG", "2026-01-01")
		in["nameEn"] = "非法😀"
		if action == "CORRECT" {
			d := detail(b, id)
			r := recordOn(t, d, "2026-01-01")
			path += "/" + id + "/records/" + r["recordId"].(string) + "/correct"
			in = correctBody(d, r, map[string]any{"nameEn": "非法😀"})
		}
		key := foundation.NewID()
		b.call("POST", path, in, key, 400)
		original, err := bootstrap.InspectBUResult(f.ctx, f.admin, f.instance, b.subject, key)
		must(t, err)
		encoded, err := json.Marshal(original)
		must(t, err)
		for _, kind := range []string{"missing_format", "unknown_format", "foreign_item"} {
			f.editBUResult(b.subject, key, func(v *bootstrap.IntegrationBUTerminal) {
				switch kind {
				case "missing_format":
					v.Failure.FormatVersion = 0
				case "unknown_format":
					v.Failure.FormatVersion = 999
				case "foreign_item":
					v.Failure.Issues[0].Locations = []org.Location{{ItemID: foundation.NewID(), FieldPath: "changes.nameEn"}}
				}
			})
			for _, resolve := range []bool{true, false} {
				route, body, k := path, any(in), key
				if resolve {
					route = "/api/v1/organization/submissions/" + key + "/resolve"
					body = nil
					k = ""
				}
				out := b.call("POST", route, body, k, 503)
				if out["code"] != "M02_RESULT_UNAVAILABLE" {
					t.Fatal(action, out)
				}
			}
			f.editBUResult(b.subject, key, func(v *bootstrap.IntegrationBUTerminal) {
				*v = bootstrap.IntegrationBUTerminal{}
				must(t, json.Unmarshal(encoded, v))
			})
		}
		b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
		if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 1 {
			t.Fatal("corruption reopened identity")
		}
	}
	if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history WHERE object_id=$1", id) != 1 {
		t.Fatal("corruption changed facts")
	}
}
