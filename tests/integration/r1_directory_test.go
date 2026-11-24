//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"crypto/sha256"
	"encoding/json"
	"reflect"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestR1DirectoryPreservesOldSnapshotsAndAllStoredOutcomes(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	path := "/api/v1/business-units"
	body := createBody("R1_OLD_CREATE", "2026-01-01")
	key := foundation.NewID()
	accepted := b.call("POST", path, body, key, 201)
	id := accepted["buId"].(string)
	// Privileged isolated fixture models an already stored old-directory result.
	// This is not a public old-write path, upgrade test, or production history edit.
	setOldDigest := func(path, key string, body map[string]any) {
		raw, e := json.Marshal([]any{2, "POST", path, body})
		must(t, e)
		sum := sha256.Sum256(raw)
		bound := sha256.New()
		bound.Write([]byte(f.instance + ":" + b.subject + ":"))
		bound.Write(sum[:])
		f.editBUResult(b.subject, key, func(v *bootstrap.IntegrationBUTerminal) { v.Digest = bound.Sum(nil) })
	}
	body["operationReason"] = "INITIAL_SETUP"
	setOldDigest(path, key, body)
	d := detail(b, id)
	r := selected(d)
	f.editBURevision(r["revision"].(string), func(v *bootstrap.IntegrationBURevision) {
		v.Reason = "INITIAL_SETUP"
		v.TypeLabel = "原新建"
		v.ReasonLabel = "初始建档"
	})
	old := detail(b, id)
	for _, reply := range []map[string]any{b.call("POST", path, body, key, 201), b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)} {
		if reply["sourceVersion"] != accepted["sourceVersion"] || reply["outcome"] != "accepted" {
			t.Fatal("accepted result reinterpreted", reply)
		}
	}
	r = selected(old)
	b.call("POST", path+"/"+id+"/records/"+r["recordId"].(string)+"/correct", correctBody(old, r, map[string]any{"name": "新名称"}), foundation.NewID(), 200)
	current := selected(detail(b, id))
	if current["operationReason"] != "INITIAL_SETUP" || current["reasonLabel"] != "初始建档" || current["typeLabel"] != "原新建" {
		t.Fatal("snapshot rewritten", current)
	}
	if !reflect.DeepEqual(detailAtSource(b, id, old["headSourceVersion"].(string))["timeline"], old["timeline"]) {
		t.Fatal("old source changed")
	}
	// Explicit authorized correction of an old first reason uses the sole current
	// CREATE reason, producing a structural revision without changing business data.
	d = detail(b, id)
	classification := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": current["revision"], "classification": map[string]any{"operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}}
	reclassified := b.call("POST", path+"/"+id+"/records/"+current["recordId"].(string)+"/correct", classification, foundation.NewID(), 200)
	if reclassified["outcome"] != "accepted" || selected(detail(b, id))["operationReason"] != "NEW_ORGANIZATION" {
		t.Fatal("old first cannot be explicitly reclassified", reclassified)
	}
	// Old CREATE reasons are no longer accepted for new writes.
	for _, reason := range []string{"INITIAL_SETUP", "NEW_UNIT"} {
		bad := createBody("R1_REJECT_"+reason, "2026-01-01")
		bad["operationReason"] = reason
		b.call("POST", path, bad, foundation.NewID(), 400)
	}
	// A stored rejection retains the original code even when its reason is now retired.
	rejectedBody := createBody("R1_OLD_CREATE", "2026-01-01")
	rejectedKey := foundation.NewID()
	rejected := b.call("POST", path, rejectedBody, rejectedKey, 409)
	rejectedBody["operationReason"] = "NEW_UNIT"
	setOldDigest(path, rejectedKey, rejectedBody)
	replay := b.call("POST", path, rejectedBody, rejectedKey, 409)
	resolved := b.call("POST", "/api/v1/organization/submissions/"+rejectedKey+"/resolve", nil, "", 200)
	if replay["code"] != rejected["code"] || resolved["outcome"] != "rejected" || resolved["errorCode"] != rejected["code"] {
		t.Fatal("rejection changed", rejected, replay, resolved)
	}
	// A stored no-change ADD must not be revalidated against the current directory.
	d = detail(b, id)
	noBody := addBody(d, "2026-06-01", map[string]any{"name": "新名称"})
	noPath := path + "/" + id + "/records"
	noKey := foundation.NewID()
	noChange := b.call("POST", noPath, noBody, noKey, 200)
	noBody["operationReason"] = "NAME_CHANGE"
	setOldDigest(noPath, noKey, noBody)
	before := businessSnapshot(t, f)
	for _, reply := range []map[string]any{b.call("POST", noPath, noBody, noKey, 200), b.call("POST", "/api/v1/organization/submissions/"+noKey+"/resolve", nil, "", 200)} {
		if reply["outcome"] != "no_change" || reply["observedSourceVersion"] != noChange["observedSourceVersion"] {
			t.Fatal("no-change result changed", reply)
		}
	}
	if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
		t.Fatal("replay wrote business facts")
	}
	for _, reason := range []string{"NAME_CHANGE", "ORGANIZATION_REASSIGNMENT"} {
		bad := addBody(detail(b, id), "2026-06-01", map[string]any{"name": "不应保存"})
		bad["operationReason"] = reason
		b.call("POST", noPath, bad, foundation.NewID(), 400)
	}
}

func detailAtSource(b *browser, id, source string) map[string]any {
	return b.call("GET", "/api/v1/business-units/"+id+"?sourceVersion="+source, nil, "", 200)
}
