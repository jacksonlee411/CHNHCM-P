//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

// Run in the original 00055 checkout so every fact, revision and receipt is
// produced by that release's normal HTTP commands.
func TestBuildOriginalProbationFixture(t *testing.T) {
	type M = map[string]any
	dir := os.Getenv("CHNHCM_PROBATION_DUMP_DIR")
	if dir == "" {
		t.Fatal("missing private fixture directory")
	}
	x := setupAssignment(t)
	b := x.h.b
	path := "/api/v1/assignments/" + x.hire["assignment"].(string)
	view := func(day string) M { return b.call("GET", path+"?businessAt="+day, nil, "", 200) }
	post := func(operation, day string, body M) M {
		body["expectedSourceVersion"] = view(day)["sourceVersion"]
		return b.call("POST", path+operation, body, f.NewID(), 200)
	}
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "transfer", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	department := b.call("POST", "/api/v1/departments", departmentCreateBody("RV500", x.bu), f.NewID(), 201)["departmentId"]
	post("/trial-start", "2027-04-01", M{"businessDate": "2027-04-01", "notice": "实际试用开始"})
	confirm := post("/confirm-employment", "2027-07-01", M{"businessDate": "2027-07-01", "notice": "实际转正，通知未载部门"})
	first := view("2027-07-01")["sourceVersion"]
	post("/transfers", "2027-06-01", M{"businessDate": "2027-06-01", "notice": "补录一月前的真实调动", "department": department, "job": x.job, "insert": true})
	original := b.call("GET", "/api/v1/employment-actions/"+confirm["action"].(string), nil, "", 200)
	if view("2027-07-01")["state"].(M)["departmentId"] != department || original["probation"].(M)["department"] != x.dep || original["action"].(M)["revision"] != confirm["revision"] {
		t.Fatal("未形成旧版本的归属偏差")
	}
	file, err := os.OpenFile(filepath.Join(dir, "legacy.dump"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(t, err)
	cmd := exec.Command("docker", "exec", x.h.f.container.GetContainerID(), "pg_dump", "-U", "postgres", "-d", "chnhcm", "-Fc")
	cmd.Stdout = file
	err = cmd.Run()
	must(t, file.Close())
	must(t, err)
	meta, err := json.Marshal(M{"instance": x.h.f.instance, "password": x.h.f.password, "assignment": x.hire["assignment"], "action": confirm["action"], "revision": confirm["revision"], "submission": confirm["submissionId"], "firstSource": first, "department": department, "originalDepartment": x.dep})
	must(t, err)
	must(t, os.WriteFile(filepath.Join(dir, "fixture.json"), meta, 0600))
}
