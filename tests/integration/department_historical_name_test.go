//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"crypto/sha256"
	"encoding/json"
	"net/url"
	"testing"
)

func TestDepartmentHistoricalNameRetainsOriginAndOrdinaryBoundary(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	bu := b.call("POST", "/api/v1/business-units", createBody("HISTORY_BU", "2020-01-01"), foundation.NewID(), 201)
	root := b.call("POST", "/api/v1/departments", departmentBody("HISTORY_PARENT", "上级", bu["buId"].(string), "2020-01-01", nil), foundation.NewID(), 201)
	input := org.DepartmentCreateInput{Code: "HISTORY_NAME", Name: " 原始?名称 ", ManagementBUID: bu["buId"].(string), EffectiveDate: "2020-01-01", OperationType: "CREATE", OperationReason: "NEW_ORGANIZATION"}
	b.call("POST", "/api/v1/departments", input, foundation.NewID(), 400)
	raw, e := json.Marshal(input)
	must(t, e)
	digest := sha256.Sum256(raw)
	cmd := org.DepartmentCommand{Action: "CREATE", SubmissionID: foundation.NewID(), Digest: digest[:], Create: input}
	u, e := url.Parse(f.server.URL)
	must(t, e)
	credentials := foundation.Credentials{CSRF: b.csrf, Write: true}
	for _, cookie := range b.client.Jar.Cookies(u) {
		if cookie.Name == "chnhcm_session" {
			credentials.Token = cookie.Value
		}
	}
	if _, e = bootstrap.ImportIntegrationDepartmentName(f.ctx, f.runtime, credentials, cmd, "另一原名"); e == nil {
		t.Fatal("mismatched source accepted")
	}
	imported, e := bootstrap.ImportIntegrationDepartmentName(f.ctx, f.runtime, credentials, cmd, input.Name)
	must(t, e)
	if imported.Outcome != "accepted" {
		t.Fatal(imported.Outcome)
	}
	id, first, old := *imported.DepartmentID, *imported.TargetRecordID, *imported.SourceVersion
	base := "/api/v1/departments/" + id
	detail := b.call("GET", base, nil, "", 200)
	if detail["selectedRecord"].(map[string]any)["name"] != input.Name {
		t.Fatal("historical spelling normalized")
	}
	b.departmentCorrect(id, first, map[string]any{"changes": map[string]any{"parentDepartmentId": root["departmentId"]}}, 200)
	detail = b.call("GET", base, nil, "", 200)
	if detail["selectedRecord"].(map[string]any)["name"] != input.Name {
		t.Fatal("unrelated change rewrote original")
	}
	versionName := "新增?历史名称"
	versionInput := org.DepartmentAddInput{ExpectedSourceVersion: detail["headSourceVersion"].(string), AnchorRecordID: first, EffectiveDate: "2021-01-01", OperationType: "CHANGE", OperationReason: "NAME_ADJUSTMENT", Changes: org.DepartmentChangesInput{"name": &versionName}}
	b.call("POST", base+"/records", versionInput, foundation.NewID(), 400)
	versionRaw, e := json.Marshal(versionInput)
	must(t, e)
	versionDigest := sha256.Sum256(versionRaw)
	versionCommand := org.DepartmentCommand{Action: "ADD", DepartmentID: id, SubmissionID: foundation.NewID(), Digest: versionDigest[:], Add: versionInput}
	version, e := bootstrap.ImportIntegrationDepartmentName(f.ctx, f.runtime, credentials, versionCommand, versionName)
	must(t, e)
	if version.Outcome != "accepted" {
		t.Fatal("historical explicit version rejected", version.Outcome)
	}
	short := "正常后继资料"
	b.call("POST", base+"/records", org.DepartmentAddInput{ExpectedSourceVersion: *version.SourceVersion, AnchorRecordID: *version.TargetRecordID, EffectiveDate: "2022-01-01", OperationType: "CHANGE", OperationReason: "NAME_ADJUSTMENT", Changes: org.DepartmentChangesInput{"shortName": &short}}, foundation.NewID(), 201)
	detail = b.call("GET", base, nil, "", 200)
	if detail["selectedRecord"].(map[string]any)["name"] != versionName {
		t.Fatal("ordinary addition rewrote inherited historical name")
	}
	b.departmentCorrect(id, first, map[string]any{"changes": map[string]any{"name": input.Name}}, 400)
	b.departmentCorrect(id, first, map[string]any{"changes": map[string]any{"name": "合法更正名称"}}, 200)
	exact := b.call("GET", base+"?sourceVersion="+old, nil, "", 200)
	if exact["selectedRecord"].(map[string]any)["name"] != input.Name || exact["readOnly"] != true {
		t.Fatal("old source changed")
	}
	resolved := b.call("POST", "/api/v1/departments/submissions/"+cmd.SubmissionID+"/resolve", nil, "", 200)
	if resolved["sourceVersion"] != old {
		t.Fatal("original import result lost")
	}
	reader := f.login("reader", f.readPassword)
	cmd.SubmissionID = foundation.NewID()
	cmd.Create.Code = "DENIED_HISTORY"
	credentials.CSRF = reader.csrf
	for _, cookie := range reader.client.Jar.Cookies(u) {
		if cookie.Name == "chnhcm_session" {
			credentials.Token = cookie.Value
		}
	}
	if _, e = bootstrap.ImportIntegrationDepartmentName(f.ctx, f.runtime, credentials, cmd, input.Name); e == nil {
		t.Fatal("history entry bypassed authorization")
	}
}
