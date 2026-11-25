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

// Only the 00056 application issues these commands. The retained operation and
// its original source remain sealed when this dump returns to the current app.
func TestBuildRecordModeFixture(t *testing.T) {
	type M = map[string]any
	x := lifecycleSample{setupAssignment(t)}
	b := x.h.b
	id := x.hire["assignment"].(string)
	staff := staffing307{t, x.h.f, b}
	p5 := staff.create("grade", "CMODE5", "")["id"].(string)
	p6 := staff.create("grade", "CMODE6", "")["id"].(string)
	staff.correct("job", staff.detail("job", x.job, ""), M{"gradeIds": []string{p5, p6}}, nil, 200)
	x.policy("transfer")
	target := b.call("POST", "/api/v1/departments", departmentCreateBody("MODE56", x.bu), f.NewID(), 201)["departmentId"]
	enter := func(path, day string, body M) M {
		body["businessDate"] = day
		return x.post(id, path, day, body, 200)
	}
	enter("grade-records", "2027-04-01", M{"grade": p5, "notice": "首次任用"})
	july := enter("location-changes", "2027-07-01", M{"location": x.locations[1], "insert": true})
	june := enter("location-changes", "2027-06-01", M{"location": x.locations[2]})
	currentJuly := b.call("GET", "/api/v1/employment-actions/"+july["action"].(string), nil, "", 200)["action"].(M)
	x.post(id, "actions/"+july["action"].(string)+"/delete", "2027-07-01", M{"expectedRevision": currentJuly["revision"]}, 200)
	enter("trial-start", "2027-08-01", M{"notice": "旧版试用开始原件"})
	august := enter("location-changes", "2027-08-01", M{"location": x.locations[3], "insert": true})
	move := enter("transfers", "2027-09-01", M{"department": target, "job": x.job, "notice": "同日后调动"})
	promotion := enter("promotions", "2027-09-01", M{"grade": p6, "notice": "同日先晋升",
		"precedes": M{"action": move["action"], "revision": move["revision"]}})
	currentOperations := map[string]any{}
	for _, receipt := range []M{june, july, august} {
		action := receipt["action"].(string)
		currentOperations[action] = b.call("GET", "/api/v1/employment-actions/"+action, nil, "", 200)["action"].(M)["operation"]
	}
	if currentOperations[july["action"].(string)] != "Delete" {
		t.Fatal("旧版未保留七月办理的删除修订")
	}
	dir := os.Getenv("CHNHCM_RECORD_MODES_DUMP_DIR")
	if dir == "" {
		t.Fatal("需要私有旧库样本目录")
	}
	fp, err := os.OpenFile(filepath.Join(dir, "legacy.dump"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	must(t, err)
	cmd := exec.Command("docker", "exec", x.h.f.container.GetContainerID(), "pg_dump", "-U", "postgres", "-d", "chnhcm", "-Fc")
	cmd.Stdout = fp
	err = cmd.Run()
	must(t, fp.Close())
	must(t, err)
	raw, err := json.Marshal(M{"instance": x.h.f.instance, "password": x.h.f.password,
		"june": june, "july": july, "august": august, "promotion": promotion, "currentOperations": currentOperations})
	must(t, err)
	must(t, os.WriteFile(filepath.Join(dir, "fixture.json"), raw, 0600))
}
