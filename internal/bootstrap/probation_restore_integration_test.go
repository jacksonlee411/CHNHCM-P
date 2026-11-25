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
	t.Helper()
	if dir := os.Getenv("CHNHCM_PROBATION_DUMP_DIR"); dir != "" {
		return dir
	}
	const commit = "d981fed840d374837f1797001af444fe94c2db6f"
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil || exec.Command("git", "cat-file", "-e", commit+"^{tree}").Run() != nil {
		t.Skip("requires original 00055 Git tree or CHNHCM_PROBATION_DUMP_DIR")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "source")
	must(t, os.Mkdir(src, 0700))
	archive := filepath.Join(dir, "source.tar")
	must(t, exec.Command("git", "-C", strings.TrimSpace(string(top)), "archive", "--output", archive, commit).Run())
	must(t, exec.Command("tar", "-xf", archive, "-C", src).Run())
	generator, err := os.ReadFile("testdata/probation_00055_test.go")
	must(t, err)
	must(t, os.WriteFile(filepath.Join(src, "internal/bootstrap/probation_00055_test.go"), generator, 0600))
	cmd := exec.Command("go", "test", "-tags=integration", "./internal/bootstrap", "-run", "^TestBuildOriginalProbationFixture$", "-count=1", "-timeout=5m")
	cmd.Dir = src
	cmd.Env = append(os.Environ(), "CHNHCM_PROBATION_DUMP_DIR="+dir)
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
	f := fresh(t)
	f.exec("CREATE DATABASE probation_legacy")
	in, err := os.Open(filepath.Join(dir, "legacy.dump"))
	must(t, err)
	defer in.Close()
	cmd := exec.Command("docker", "exec", "-i", f.container.GetContainerID(), "pg_restore", "--exit-on-error", "-U", "postgres", "-d", "probation_legacy")
	cmd.Stdin = in
	must(t, cmd.Run())
	cfg := f.admin.Config()
	cfg.ConnConfig.Database = "probation_legacy"
	pool, err := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, err)
	t.Cleanup(pool.Close)
	var baseline int
	must(t, pool.QueryRow(f.ctx, "SELECT max(version_id) FROM goose_db_version WHERE is_applied").Scan(&baseline))
	if baseline != 55 {
		t.Fatal("旧库版本", baseline)
	}
	must(t, bootstrap.Migrate(f.ctx, pool, migrations.Files))
	conn, err := url.Parse(f.runtime.Pool.Config().ConnString())
	must(t, err)
	conn.Path = "/probation_legacy"
	rt, err := bootstrap.Open(f.ctx, conn.String(), sample.Instance)
	must(t, err)
	t.Cleanup(rt.Close)
	server := httptest.NewUnstartedServer(nil)
	api := bootstrap.API{Runtime: rt, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}
	server.Config.Handler = api.Handler()
	server.Start()
	t.Cleanup(server.Close)
	db := &fixture{t: t, ctx: f.ctx, admin: pool, runtime: rt, server: server, instance: sample.Instance}
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
