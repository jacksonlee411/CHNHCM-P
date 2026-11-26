//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	pg "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	f "chnhcm.local/chnhcm/internal/foundation"
)

// This fixture runs in the 00056 Git tree. Every discrepancy is created
// through the original HTTP commands before pg_dump captures the database.
func TestBuildOriginalLifecycleFixture(t *testing.T) {
	type M = map[string]any
	dir := os.Getenv("CHNHCM_T04_DUMP_DIR")
	if dir == "" {
		t.Fatal("missing private fixture directory")
	}
	x := lifecycleSample{setupAssignment(t)}
	b := x.h.b
	x.policy("secondary")
	x.policy("transfer")
	exitManager := x.hire["assignment"].(string)
	successor := x.hirePerson("接任经理", "")["assignment"].(string)
	child := x.hirePerson("原汇报任职", "")["assignment"].(string)
	dutyHire := x.hirePerson("已出勤人员", "")
	dutyID := dutyHire["assignment"].(string)
	product := b.call("POST", "/api/v1/departments", departmentCreateBody("D509", x.bu), f.NewID(), 201)["departmentId"].(string)
	add := M{"from": "2027-05-01", "department": x.dep, "job": x.job, "notice": "两次登记同一任命", "parallel": false}
	one := x.post(exitManager, "additional-assignments", "2027-05-01", add, 200)
	two := x.post(exitManager, "additional-assignments", "2027-05-01", add, 200)
	if one["recordNumber"] == two["recordNumber"] {
		t.Fatal("未形成旧库双号")
	}
	x.post(child, "managers", "2027-05-01", M{"manager": exitManager, "purpose": "direct", "businessDate": "2027-05-01"}, 200)
	leave := x.exit(exitManager, "leave", "2027-08-31", "last-active", 200)
	oldManagerSource := x.view(child, "2027-09-01", "")["sourceVersion"]
	oldExitSource := x.view(exitManager, "2027-08-31", "")["sourceVersion"]
	x.post(child, "managers", "2027-07-01", M{"manager": successor, "purpose": "direct", "businessDate": "2027-07-01"}, 200)
	x.post(exitManager, "transfers", "2027-07-01", M{"businessDate": "2027-07-01", "job": x.job, "department": product, "notice": "七月调入产品", "insert": true}, 200)
	rehire := b.call("POST", "/api/v1/hires", M{"arrival": "2027-10-01", "person": x.person, "employer": x.entity, "job": x.job, "department": product, "notice": "正式再次任用", "rehireOf": M{"action": leave["action"], "revision": leave["revision"]}}, f.NewID(), 200)
	if len(x.view(child, "2027-09-01", "")["managers"].([]any)) != 0 || b.call("GET", "/api/v1/employment-actions/"+leave["action"].(string), nil, "", 200)["action"].(M)["department"] != x.dep {
		t.Fatal("原版次未形成退出或经理偏差")
	}

	x.h.f.runtime.Staffing.(app.Service).Store.(*pg.Store).Clock = func() time.Time { return time.Date(2027, 4, 4, 12, 0, 0, 0, f.Shanghai) }
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	for _, v := range policy["rows"].([]any) {
		r := v.(M)
		if r["purpose"] == "transfer" {
			b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": M{"id": r["id"], "revision": r["revision"], "category": "Employee", "purpose": "transfer", "mode": "PositionCounted", "from": "2027-04-01"}}, f.NewID(), 200)
		}
	}
	position := b.call("POST", "/api/v1/positions", M{"code": "DUTY509", "name": "门店岗", "effectiveDate": "2027-04-01", "operationType": "CREATE", "operationReason": "NEW_POSITION", "departmentId": x.dep, "jobId": x.job, "locationId": x.locations[0]}, f.NewID(), 201)["objectId"]
	b.call("POST", "/api/v1/staffing-quotas", M{"positionId": position, "effectiveDate": "2027-04-01", "quota": 1, "reason": "核定一编"}, f.NewID(), 200)
	source := f.NewID()
	dutyPerson := x.view(dutyID, "2027-04-01", "")["person"]
	dispatch := b.call("POST", "/api/v1/appointment-decisions", M{"source": source, "number": "派岗509", "roster": false, "rows": []M{{"person": dutyPerson, "position": M{"id": position, "name": "门店岗"}, "arrival": "2027-04-01"}}}, f.NewID(), 200)
	paper := M{"from": "2027-04-01", "lastDay": "2027-04-03", "observedThrough": "2027-04-04", "number": "出勤509", "issuer": "门店", "dispatch": M{"source": source, "version": dispatch["version"]}}
	x.post(dutyID, "actual-duty", "2027-04-01", M{"paper": paper}, 200)
	dutyExit := x.exit(dutyID, "leave", "2027-04-03", "first-inactive", 200)

	file, err := os.OpenFile(filepath.Join(dir, "legacy.dump"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(t, err)
	cmd := exec.Command("docker", "exec", x.h.f.container.GetContainerID(), "pg_dump", "-U", "postgres", "-d", "chnhcm", "-Fc")
	cmd.Stdout = file
	err = cmd.Run()
	must(t, file.Close())
	must(t, err)
	meta, err := json.Marshal(M{"instance": x.h.f.instance, "password": x.h.f.password, "assignment": exitManager, "department": product, "oldDepartment": x.dep, "leave": leave, "rehire": rehire, "child": child, "successor": successor, "oldManagerSource": oldManagerSource, "oldExitSource": oldExitSource, "dutyExit": dutyExit, "duplicateActions": []any{one["action"], two["action"]}})
	must(t, err)
	must(t, os.WriteFile(filepath.Join(dir, "fixture.json"), meta, 0600))
}
