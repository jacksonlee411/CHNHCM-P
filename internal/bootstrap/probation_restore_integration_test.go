//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A supplied dump allows this check to run from a shallow checkout too.
func originalProbationDump(t *testing.T) string {
	return staffingReleaseDump(t, "d981fed840d374837f1797001af444fe94c2db6f", "CHNHCM_PROBATION_DUMP_DIR", "TestBuildOriginalProbationFixture", "testdata/probation_00055_test.go")
}

func staffingReleaseDump(t *testing.T, commit, env, test string, files ...string) string {
	t.Helper()
	if dir := os.Getenv(env); dir != "" {
		return dir
	}
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil || exec.Command("git", "cat-file", "-e", commit+"^{tree}").Run() != nil {
		t.Fatalf("旧库升级测试需要原版 Git 提交 %s 或通过 %s 提供其样本", commit, env)
	}
	root := strings.TrimSpace(string(top))
	dir := t.TempDir()
	src := filepath.Join(dir, "source")
	must(t, os.Mkdir(src, 0700))
	archive := filepath.Join(dir, "source.tar")
	must(t, exec.Command("git", "-C", root, "archive", "--output", archive, commit).Run())
	must(t, exec.Command("tar", "-xf", archive, "-C", src).Run())
	// Match the runner's registry while retaining the archived PostgreSQL version and digest.
	lock := filepath.Join("deploy", "local", "postgres-image.txt")
	current, err := os.ReadFile(filepath.Join(root, lock))
	must(t, err)
	original, err := os.ReadFile(filepath.Join(src, lock))
	must(t, err)
	registry, _, currentOK := strings.Cut(strings.TrimSpace(string(current)), "/")
	_, image, originalOK := strings.Cut(strings.TrimSpace(string(original)), "/")
	if !currentOK || !originalOK || !strings.Contains(image, "@sha256:") {
		t.Fatal("缺少 PostgreSQL 镜像仓库或原版摘要")
	}
	must(t, os.WriteFile(filepath.Join(src, lock), []byte(registry+"/"+image+"\n"), 0600))
	for _, file := range files {
		body, err := os.ReadFile(file)
		must(t, err)
		must(t, os.WriteFile(filepath.Join(src, "internal/bootstrap", filepath.Base(file)), body, 0600))
	}
	cmd := exec.Command("go", "test", "-tags=integration", "./internal/bootstrap", "-run", "^"+test+"$", "-count=1", "-timeout=8m")
	cmd.Dir = src
	cmd.Env = append(os.Environ(), env+"="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("准备旧版业务样本: %v\n%s", err, out)
	}
	return dir
}

// Restore an actual 00055 edition with a transfer recorded retroactively by
// the pre-repair binary. The generated operator login stays outside the tree.
func TestProbationRepairFromPreviousRelease(t *testing.T) {
	dir := originalProbationDump(t)
	raw, err := os.ReadFile(filepath.Join(dir, "fixture.json"))
	must(t, err)
	var sample struct{ Instance, Password, Action, Assignment, Revision, Submission, FirstSource, Department, OriginalDepartment string }
	must(t, json.Unmarshal(raw, &sample))
	db := restoreStaffingRelease(t, dir, "probation_legacy", sample.Instance, 55)
	f, rt := db, db.runtime
	b := db.login("admin", sample.Password)
	original := func(rev string) map[string]any {
		return b.call("GET", "/api/v1/employment-actions/"+sample.Action+"?adoptedRef="+rev, nil, "", 200)
	}
	counts := func() [4]int {
		return [4]int{db.scalar("SELECT count(*) FROM staffing.employment_action_revision"), db.scalar("SELECT count(*) FROM staffing.assignment_source"), db.scalar("SELECT count(*) FROM staffing.employment_outbox"), db.scalar("SELECT count(*) FROM staffing.hire_terminal")}
	}
	login := access.LoginInput{Login: "admin", Password: sample.Password}
	before := counts()
	beforeDept := func(rev string) any { return original(rev)["probation"].(map[string]any)["department"] }
	if beforeDept("") != sample.OriginalDepartment {
		t.Fatal("旧版反例未成立")
	}
	preview, err := rt.ReconcileProbation(f.ctx, login, false)
	must(t, err)
	if preview.Scanned != 1 || preview.Differences != 1 || preview.Repaired != 0 || preview.Unresolved+preview.Failed != 0 || counts() != before {
		t.Fatal("预检", preview)
	}
	applied, err := rt.ReconcileProbation(f.ctx, login, true)
	must(t, err)
	if applied.Repaired != 1 || applied.Differences != 1 || applied.Unresolved+applied.Failed != 0 || beforeDept("") != sample.Department || beforeDept(sample.Revision) != sample.OriginalDepartment {
		t.Fatal("旧单修订", applied)
	}
	after := counts()
	if after != [4]int{before[0] + 1, before[1] + 1, before[2] + 1, before[3]} {
		t.Fatal("修复增量", before, after)
	}
	again, err := rt.ReconcileProbation(f.ctx, login, true)
	must(t, err)
	if again.Scanned != 1 || again.Differences+again.Failed+again.Unresolved != 0 || counts() != after {
		t.Fatal("重跑增量", again)
	}
	historical := b.call("GET", "/api/v1/assignments/"+sample.Assignment+"?businessAt=2027-07-01&sourceVersion="+sample.FirstSource, nil, "", 200)
	if historical["state"].(map[string]any)["departmentId"] != sample.OriginalDepartment {
		t.Fatal("旧来源改变")
	}
	receipt := b.call("POST", "/api/v1/employment-actions/submissions/"+sample.Submission+"/resolve", nil, "", 200)
	if receipt["action"] != sample.Action || receipt["revision"] != sample.Revision {
		t.Fatal("原回执改变", receipt)
	}
}

func restoreStaffingRelease(t *testing.T, dir, database, instance string, wantVersion int) *fixture {
	t.Helper()
	f := fresh(t)
	f.exec("CREATE DATABASE " + database)
	in, err := os.Open(filepath.Join(dir, "legacy.dump"))
	must(t, err)
	defer in.Close()
	cmd := exec.Command("docker", "exec", "-i", f.container.GetContainerID(), "pg_restore", "--exit-on-error", "-U", "postgres", "-d", database)
	cmd.Stdin = in
	must(t, cmd.Run())
	cfg := f.admin.Config()
	cfg.ConnConfig.Database = database
	pool, err := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, err)
	t.Cleanup(pool.Close)
	var baseline int
	must(t, pool.QueryRow(f.ctx, "SELECT max(version_id) FROM goose_db_version WHERE is_applied").Scan(&baseline))
	if baseline != wantVersion {
		t.Fatal("旧库版本", baseline)
	}
	must(t, bootstrap.Migrate(f.ctx, pool, migrations.Files))
	conn, err := url.Parse(f.runtime.Pool.Config().ConnString())
	must(t, err)
	conn.Path = "/" + database
	rt, err := bootstrap.Open(f.ctx, conn.String(), instance)
	must(t, err)
	t.Cleanup(rt.Close)
	server := httptest.NewUnstartedServer(nil)
	api := bootstrap.API{Runtime: rt, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}
	server.Config.Handler = api.Handler()
	server.Start()
	t.Cleanup(server.Close)
	return &fixture{t: t, ctx: f.ctx, admin: pool, runtime: rt, server: server, instance: instance}
}
