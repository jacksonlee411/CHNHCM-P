//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestRetainedRecordModesUseFirstPublishedSource(t *testing.T) {
	type M = map[string]any
	dir := staffingReleaseDump(t, "39660e70ded3257fe0c3e483e694dbe12f81bc2b", "CHNHCM_RECORD_MODES_DUMP_DIR", "TestBuildRecordModeFixture", "lifecycle_fixture_test.go", "testdata/record_modes_00056_test.go")
	raw, err := os.ReadFile(filepath.Join(dir, "fixture.json"))
	must(t, err)
	var fixture struct {
		Instance, Password            string
		June, July, August, Promotion M
		CurrentOperations             map[string]string
	}
	must(t, json.Unmarshal(raw, &fixture))
	db := restoreStaffingRelease(t, dir, "initial_positions", fixture.Instance, 56)
	b := db.login("admin", fixture.Password)
	login := access.LoginInput{Login: "admin", Password: fixture.Password}
	counts := func() [4]int {
		return [4]int{db.scalar("SELECT count(*) FROM staffing.employment_action_revision"), db.scalar("SELECT count(*) FROM staffing.assignment_source"),
			db.scalar("SELECT count(*) FROM staffing.hire_terminal"), db.scalar("SELECT count(*) FROM staffing.employment_outbox")}
	}
	want := map[any]string{fixture.June["action"]: "explained", fixture.July["action"]: "explained", fixture.August["action"]: "unresolved"}
	before := counts()
	for _, apply := range []bool{false, true, true} {
		report, err := db.runtime.ReconcileT04(db.ctx, login, apply)
		must(t, err)
		if report.Differences != 3 || report.Unresolved != 1 || report.Failed+report.Repaired != 0 || len(report.Items) != len(want) || counts() != before {
			t.Fatal("旧操作的还原与重跑应保留全部修订、来源、回执与出站", apply, report)
		}
		for _, item := range report.Items {
			if want[item.Action] != item.Result {
				t.Fatal("将可还原的末端/历史或证据不足的同日操作误分", item)
			}
		}
	}
	for _, example := range []struct {
		receipt             M
		recorded, effective string
	}{
		{fixture.June, "Add", "Insert"}, {fixture.July, "Insert", "Add"}, {fixture.August, "Insert", ""},
	} {
		id := example.receipt["action"].(string)
		for _, rev := range []string{"", example.receipt["revision"].(string)} {
			original := b.call("GET", "/api/v1/employment-actions/"+id+"?adoptedRef="+rev, nil, "", 200)
			position, ok := original["initialRecordPosition"].(M)
			actual, _ := position["effective"].(string)
			operation := fixture.CurrentOperations[id]
			if rev != "" {
				operation = example.recorded
			}
			if !ok || position["recorded"] != example.recorded || actual != example.effective || original["action"].(M)["operation"] != operation {
				t.Fatal("首次位置或原登记操作不正确", id, rev, original)
			}
		}
		resolved := b.call("POST", "/api/v1/employment-actions/submissions/"+example.receipt["submissionId"].(string)+"/resolve", nil, "", 200)
		if resolved["action"] != id || resolved["revision"] != example.receipt["revision"] {
			t.Fatal("原始提交号没有恢复当时的修订", resolved)
		}
	}
	p := b.call("GET", "/api/v1/employment-actions/"+fixture.Promotion["action"].(string), nil, "", 200)
	if p["initialRecordPosition"] != nil || p["action"].(M)["operation"] != "Insert" {
		t.Fatal("同日明确先于已有调动的晋升应保留为插入", p)
	}
	_, err = db.admin.Exec(db.ctx, "UPDATE access.employment_grant SET department_id=$2 WHERE principal_id=$1 AND action='history'", b.subject, f.NewID())
	must(t, err)
	report, err := db.runtime.ReconcileT04(db.ctx, login, false)
	must(t, err)
	if report.Differences+report.Unresolved+report.Failed != 0 || len(report.Items) != 0 || counts() != before {
		t.Fatal("权限收回后仍披露原登记差异", report)
	}
	b.call("GET", "/api/v1/employment-actions/"+fixture.June["action"].(string), nil, "", 403)
}
