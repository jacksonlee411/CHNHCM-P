//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	db "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"testing"
)

func staffingFacts307(t *testing.T, f *fixture) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, kind := range []string{"job_family", "job_function", "job", "grade", "grade_ladder"} {
		parts := []string{"", "_source", "_revision", "_terminal", "_outbox"}
		if kind == "job" || kind == "grade_ladder" {
			parts = append(parts, "_grade_manifest", "_grade_member")
		}
		for _, part := range parts {
			table := "staffing." + kind + part
			var hash string
			must(t, f.admin.QueryRow(f.ctx, "SELECT md5(COALESCE(string_agg(row_to_json(v)::text,E'\\n' ORDER BY row_to_json(v)::text),'')) FROM "+table+" v").Scan(&hash))
			out[table] = hash
		}
	}
	return out
}
func TestStaffing307ImmutableRebuildAndRestart(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	function := h.create("jobfunction", "RESTORE-F", "")
	h.add("jobfunction", function, "ADD", "2010-01-01", map[string]any{"description": "职类日期投影"}, 200)
	family := h.create("jobfamily", "RESTORE-F", "")
	job := h.create("job", "RESTORE-J", family["id"].(string))
	h.correct("job", job, map[string]any{"responsibilities": "重启后保持的职责\n第二行"}, nil, 200)
	job = h.refresh("job", job)
	grade := h.create("grade", "RESTORE-G", "")
	h.add("grade", grade, "ADD", "2010-01-01", map[string]any{"description": "职级投影重建原值"}, 200)
	secondGrade := h.create("grade", "RESTORE-G2", "")
	ladder := create310(h, "gradeladder", "RESTORE-L", []string{grade["id"].(string), secondGrade["id"].(string)}, "")
	h.add("gradeladder", ladder, "ADD", "2011-01-01", map[string]any{"name": "重建后保持阶梯顺序", "gradeIds": []string{secondGrade["id"].(string), grade["id"].(string)}}, 200)
	h.correct("job", job, map[string]any{"gradeIds": []string{grade["id"].(string)}}, nil, 200)
	job = h.refresh("job", job)
	before := staffingFacts307(t, f)
	readHash := func() string {
		var v string
		must(t, f.admin.QueryRow(f.ctx, "SELECT md5(string_agg(v::text,'' ORDER BY v::text)) FROM (SELECT row_to_json(r) v FROM staffing.job_read r UNION ALL SELECT row_to_json(r) v FROM staffing.job_family_read r UNION ALL SELECT row_to_json(r) v FROM staffing.job_function_read r UNION ALL SELECT row_to_json(r) v FROM staffing.grade_read r UNION ALL SELECT row_to_json(r) v FROM staffing.grade_ladder_read r) x").Scan(&v))
		return v
	}
	originalRead := readHash()
	m02 := businessSnapshot(t, f)
	for _, table := range []string{"staffing.job_family_source", "staffing.job_source", "staffing.job_revision", "staffing.job_terminal", "staffing.job_outbox"} {
		_, e := f.admin.Exec(f.ctx, "DELETE FROM "+table)
		if e == nil {
			t.Fatalf("immutable table accepted deletion: %s", table)
		}
	}
	store := f.runtime.Staffing.(app.Service).Store.(*db.Store)
	f.exec(`DELETE FROM staffing.job_read; DELETE FROM staffing.job_family_read; DELETE FROM staffing.job_function_read; DELETE FROM staffing.grade_read; DELETE FROM staffing.grade_ladder_read`)
	must(t, store.RebuildReads(f.ctx))
	if f.scalar(`SELECT count(*) FROM staffing.job_read`) == 0 || f.scalar(`SELECT count(*) FROM staffing.job_family_read`) == 0 {
		t.Fatal("read projection rebuild incomplete")
	}
	if f.scalar(`SELECT count(*) FROM staffing.grade_read WHERE valid_period @> DATE '2010-01-01'`) != 2 {
		t.Fatal("Grade half-open range/rebuild failed")
	}
	if f.scalar(`SELECT count(*) FROM staffing.grade_ladder_read WHERE valid_period @> DATE '2011-01-01'`) != 1 {
		t.Fatal("Ladder half-open range/rebuild failed")
	}
	restoredLadder := h.refresh("gradeladder", ladder)
	if rec307(restoredLadder)["gradeIds"].([]any)[0] != secondGrade["id"] {
		t.Fatal("Ladder rebuild changed current membership order")
	}
	oldLadder := h.b.call("GET", base307("gradeladder")+"/"+ladder["id"].(string)+"?sourceVersion="+ladder["headSourceVersion"].(string), nil, "", 200)
	if rec307(oldLadder)["gradeIds"].([]any)[0] != grade["id"] {
		t.Fatal("Ladder rebuild changed historical membership order")
	}
	if readHash() != originalRead {
		t.Fatal("rebuilt dated projection differs")
	}
	after := staffingFacts307(t, f)
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("rebuild changed fact %s", k)
		}
	}
	dsn := f.runtime.Pool.Config().ConnConfig.ConnString()
	c := geoCredentials203(t, f, "admin", f.password)
	key := foundation.NewID()
	cmd := h.command307(family["id"].(string), "RESTORE-RECEIPT")
	cmd.SubmissionID = key
	r, e := f.runtime.Staffing.Execute(f.ctx, c, cmd)
	must(t, e)
	if r.Outcome != "accepted" {
		t.Fatal("receipt setup failed")
	}
	f.runtime.Close()
	next, e := bootstrap.Open(context.Background(), dsn, f.instance)
	must(t, e)
	t.Cleanup(next.Close)
	got, e := next.Staffing.Resolve(f.ctx, c, staff.Job, key)
	must(t, e)
	if got.Outcome != "accepted" || got.SourceVersion != r.SourceVersion {
		t.Fatal("original receipt did not survive restart")
	}
	detail, e := next.Staffing.Detail(f.ctx, c, staff.Job, job["id"].(string), job["headSourceVersion"].(string), "")
	must(t, e)
	if detail["selectedRecord"].(staff.View)["responsibilities"] != "重启后保持的职责\n第二行" {
		t.Fatal("original source did not survive restart")
	}
	current := businessSnapshot(t, f)
	for k, v := range m02 {
		if current[k] != v {
			t.Fatalf("M02 source changed %s", k)
		}
	}
	// The old Runtime close is idempotent, and fixture cleanup owns only this DB.
}
