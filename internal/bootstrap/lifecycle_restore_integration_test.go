//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
)

func TestLifecycleRepairFromPreviousRelease(t *testing.T) {
	type M = map[string]any
	dir := staffingReleaseDump(t, "39660e70ded3257fe0c3e483e694dbe12f81bc2b", "CHNHCM_T04_DUMP_DIR", "TestBuildOriginalLifecycleFixture", "lifecycle_fixture_test.go", "testdata/lifecycle_00056_test.go")
	raw, err := os.ReadFile(filepath.Join(dir, "fixture.json"))
	must(t, err)
	var s struct {
		Instance, Password, Assignment, Department, OldDepartment, Child, Successor, OldManagerSource, OldExitSource string
		Leave, Rehire, DutyExit                                                                                      M
		DuplicateActions                                                                                             []string
	}
	must(t, json.Unmarshal(raw, &s))
	db := restoreStaffingRelease(t, dir, "lifecycle_legacy", s.Instance, 56)
	b := db.login("admin", s.Password)
	counts := func() [5]int {
		return [5]int{db.scalar("SELECT count(*) FROM staffing.employment_action_revision"), db.scalar("SELECT count(*) FROM staffing.assignment_source"), db.scalar("SELECT count(*) FROM staffing.employment_outbox"), db.scalar("SELECT count(*) FROM staffing.hire_terminal"), db.scalar("SELECT count(*) FROM staffing.employment_record")}
	}
	original := func(row M, rev string) M {
		return b.call("GET", "/api/v1/employment-actions/"+row["action"].(string)+"?adoptedRef="+rev, nil, "", 200)
	}
	view := func(id, day, src string) M {
		return b.call("GET", "/api/v1/assignments/"+id+"?businessAt="+day+"&sourceVersion="+src, nil, "", 200)
	}
	origin := s.Leave["revision"].(string)
	if original(s.Leave, "")["action"].(M)["department"] != s.OldDepartment || len(view(s.Child, "2027-09-01", "")["managers"].([]any)) != 0 {
		t.Fatal("恢复的旧事实与原样本不同")
	}
	login := access.LoginInput{Login: "admin", Password: s.Password}
	before := counts()
	report, err := db.runtime.ReconcileT04(db.ctx, login, false)
	must(t, err)
	explained := 0
	for _, item := range report.Items {
		if item.Result == "explained" {
			explained++
		}
	}
	if report.Differences != 5+explained || report.Unresolved != 3 || report.Failed != 0 || report.Repaired != 0 || counts() != before {
		t.Fatal("旧库预检", report, counts(), before)
	}
	expectedManual := map[string]bool{s.DutyExit["action"].(string): false}
	for _, id := range s.DuplicateActions {
		expectedManual[id] = false
	}
	for _, item := range report.Items {
		if item.Result == "unresolved" {
			expectedManual[item.Action] = true
		}
	}
	if len(expectedManual) != 3 {
		t.Fatal("待核范围与旧库原件不符", expectedManual)
	}
	for id, found := range expectedManual {
		if !found {
			t.Fatal("缺少应由经办人核对的原件", id)
		}
	}
	applied, err := db.runtime.ReconcileT04(db.ctx, login, true)
	must(t, err)
	if applied.Repaired != 2 || applied.Failed != 0 || applied.Unresolved != report.Unresolved {
		t.Fatal("有权应用", applied)
	}
	t.Logf("00056 预检扫描 %d，差异 %d，待核 %d；应用追加办理修订 %d", report.Scanned, report.Differences, report.Unresolved, applied.Repaired)
	after := counts()
	if after != [5]int{before[0] + 2, before[1] + 4, before[2] + 2, before[3], before[4]} {
		t.Fatal("每个退出成员和经理变动保留一份来源", before, after)
	}
	again, err := db.runtime.ReconcileT04(db.ctx, login, true)
	must(t, err)
	if again.Differences != 3+explained || again.Repaired != 0 || again.Failed != 0 || again.Unresolved != report.Unresolved || counts() != after {
		t.Fatal("旧库重跑", again)
	}
	if original(s.Leave, "")["action"].(M)["department"] != s.Department || original(s.Leave, origin)["action"].(M)["department"] != s.OldDepartment || original(s.Rehire, "")["rehireOf"].(M)["revision"] != origin {
		t.Fatal("当前退出或当时采用改变")
	}
	oldReceipt := b.call("POST", "/api/v1/employment-actions/submissions/"+s.Leave["submissionId"].(string)+"/resolve", nil, "", 200)
	if oldReceipt["revision"] != origin {
		t.Fatal("原回执被改写", oldReceipt)
	}
	if len(view(s.Child, "2027-09-01", s.OldManagerSource)["managers"].([]any)) != 0 || view(s.Child, "2027-09-01", "")["managers"].([]any)[0].(M)["manager"] != s.Successor || view(s.Assignment, "2027-08-31", s.OldExitSource)["state"].(M)["departmentId"] != s.OldDepartment {
		t.Fatal("旧来源或当前经理错误")
	}
}
