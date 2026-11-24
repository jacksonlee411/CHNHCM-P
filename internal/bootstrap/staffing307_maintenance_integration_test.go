//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type staffing307 struct {
	t *testing.T
	f *fixture
	b *browser
}

func staffingSetup307(t *testing.T) *staffing307 {
	f := freshWithEnterpriseStart(t, "1900-01-01")
	return &staffing307{t, f, f.login("admin", f.password)}
}

// Existing Job journeys use a separately established classification.
func (b *browser) functionForJobs(date string) string {
	if b.jobFunctionFixture == "" {
		b.jobFunctionFixture = b.call("POST", "/api/v1/job-functions", map[string]any{"code": "TEST-" + foundation.NewID()[:8], "name": "专业类", "effectiveDate": date, "operationType": "CREATE", "operationReason": "NEW_JOB_FUNCTION"}, foundation.NewID(), 201)["objectId"].(string)
	}
	return b.jobFunctionFixture
}
func base307(k string) string {
	if k == "position" {
		return "/api/v1/positions"
	}
	if k == "jobfunction" {
		return "/api/v1/job-functions"
	}
	if k == "jobfamily" {
		return "/api/v1/job-families"
	}
	if k == "gradeladder" {
		return "/api/v1/grade-ladders"
	}
	if k == "grade" {
		return "/api/v1/grades"
	}
	return "/api/v1/jobs"
}
func (s *staffing307) create(k, code, family string) map[string]any {
	s.t.Helper()
	body := map[string]any{"code": code, "name": "资料 " + code, "nameEn": "Definition " + code, "effectiveDate": "2000-01-01", "operationType": "CREATE", "operationReason": "NEW_JOB_FAMILY"}
	if k == "job" {
		body["jobFunctionId"] = s.b.functionForJobs("1900-01-01")
		body["jobFamilyId"] = family
		body["operationReason"] = "NEW_JOB"
	}
	if k == "jobfunction" {
		body["operationReason"] = "NEW_JOB_FUNCTION"
	}
	if k == "grade" {
		body["operationReason"] = "NEW_GRADE"
	}
	v := s.b.call("POST", base307(k), body, foundation.NewID(), 201)
	return s.detail(k, v["objectId"].(string), "")
}
func (s *staffing307) detail(k, id, record string) map[string]any {
	s.t.Helper()
	q := ""
	if record != "" {
		q = "?selectedRecordId=" + record
	}
	return s.b.call("GET", base307(k)+"/"+id+q, nil, "", 200)
}
func rec307(v map[string]any) map[string]any { return v["selectedRecord"].(map[string]any) }
func (s *staffing307) correct(k string, v map[string]any, changes map[string]any, extra map[string]any, status int) map[string]any {
	s.t.Helper()
	r := rec307(v)
	body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"]}
	if changes != nil {
		body["changes"] = changes
	}
	for k, x := range extra {
		body[k] = x
	}
	return s.b.call("POST", base307(k)+"/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/correct", body, foundation.NewID(), status)
}
func (s *staffing307) add(k string, v map[string]any, action, date string, changes map[string]any, status int) map[string]any {
	s.t.Helper()
	op, reason, suffix := "CHANGE", "NAME_ADJUSTMENT", "records"
	if action == "DEACTIVATE" {
		op, reason, suffix = "DEACTIVATE", "FAMILY_CLOSURE", "deactivate"
		if k == "position" {
			reason = "POSITION_CLOSURE"
		}
		if k == "job" {
			reason = "JOB_CLOSURE"
		}
		if k == "jobfunction" {
			reason = "FUNCTION_CLOSURE"
		}
		if k == "grade" {
			reason = "GRADE_CLOSURE"
		}
		if k == "gradeladder" {
			reason = "GRADE_LADDER_CLOSURE"
		}
	}
	if action == "REACTIVATE" {
		op, reason, suffix = "REACTIVATE", "FAMILY_RESUMPTION", "reactivate"
		if k == "position" {
			reason = "POSITION_RESUMPTION"
		}
		if k == "job" {
			reason = "JOB_RESUMPTION"
		}
		if k == "jobfunction" {
			reason = "FUNCTION_RESUMPTION"
		}
		if k == "grade" {
			reason = "GRADE_RESUMPTION"
		}
		if k == "gradeladder" {
			reason = "GRADE_LADDER_RESUMPTION"
		}
	}
	body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": rec307(v)["recordId"], "effectiveDate": date, "operationType": op, "operationReason": reason}
	if changes != nil {
		body["changes"] = changes
	}
	return s.b.call("POST", base307(k)+"/"+v["id"].(string)+"/"+suffix, body, foundation.NewID(), status)
}
func (s *staffing307) refresh(k string, v map[string]any) map[string]any {
	return s.detail(k, v["id"].(string), "")
}
func (s *staffing307) raw(k, id string) map[string]any {
	s.t.Helper()
	table := "job"
	if k == "grade" {
		table = "grade"
	}
	if k == "jobfamily" {
		table = "job_family"
	}
	var data []byte
	must(s.t, s.f.admin.QueryRow(s.f.ctx, "SELECT head FROM staffing."+table+" WHERE instance_id=$1 AND id=$2", s.f.instance, id).Scan(&data))
	var out map[string]any
	must(s.t, json.Unmarshal(data, &out))
	return out
}
func TestStaffing307DatedMaintenance(t *testing.T) {
	s := staffingSetup307(t)
	f := s.create("jobfamily", "F-DATED", "")
	for _, k := range []string{"jobfamily", "jobfunction", "job"} {
		t.Run(k, func(t *testing.T) {
			v := s.create(k, "DATED", f["id"].(string))
			id := v["id"].(string)
			first, oldsource := rec307(v)["recordId"].(string), v["headSourceVersion"].(string)
			s.add(k, v, "ADD", "2010-01-01", map[string]any{"name": "独立后项"}, 200)
			v = s.detail(k, id, first)
			s.correct(k, v, map[string]any{"name": "更正首项", "nameEn": nil}, nil, 200)
			v = s.detail(k, id, "")
			if rec307(v)["name"] != "独立后项" {
				t.Fatal("successor changed")
			}
			v = s.detail(k, id, first)
			s.add(k, v, "ADD", "2005-01-01", map[string]any{"name": "插入中间"}, 200)
			v = s.detail(k, id, "")
			middle := v["timeline"].([]any)[1].(map[string]any)
			v = s.detail(k, id, middle["recordId"].(string))
			s.correct(k, v, nil, map[string]any{"newEffectiveDate": "2006-01-01"}, 200)
			v = s.detail(k, id, middle["recordId"].(string))
			s.b.call("POST", base307(k)+"/"+id+"/records/"+middle["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": rec307(v)["revision"]}, foundation.NewID(), 200)
			v = s.detail(k, id, "")
			s.add(k, v, "DEACTIVATE", "2015-01-01", nil, 200)
			v = s.detail(k, id, "")
			s.add(k, v, "REACTIVATE", "2018-01-01", map[string]any{"name": "恢复后独立资料"}, 200)
			v = s.detail(k, id, "")
			if len(v["timeline"].([]any)) != 4 || rec307(v)["name"] != "恢复后独立资料" {
				t.Fatal("lifecycle failed")
			}
			old := s.b.call("GET", base307(k)+"/"+id+"?sourceVersion="+oldsource, nil, "", 200)
			if rec307(old)["name"] != "资料 DATED" || len(old["timeline"].([]any)) != 1 {
				t.Fatal("old source rewritten")
			}
			v = s.detail(k, id, first)
			s.b.call("POST", base307(k)+"/"+id+"/records/"+first+"/delete", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": rec307(v)["revision"]}, foundation.NewID(), 400)
		})
	}
}
func TestStaffing307PairAndExactAdoption(t *testing.T) {
	s := staffingSetup307(t)
	f := s.create("jobfamily", "F-ADOPT", "")
	fid := f["id"].(string)
	v := s.create("job", "ADOPT", fid)
	id := v["id"].(string)
	first := rec307(v)["recordId"].(string)
	original := s.raw("job", id)["records"].([]any)[0].(map[string]any)["adoption"]
	source := v["headSourceVersion"].(string)
	s.correct("jobfamily", f, map[string]any{"name": "系列更正名称"}, nil, 200)
	if s.raw("job", id)["source"] != source {
		t.Fatal("rename rewrote job")
	}
	v = s.refresh("job", v)
	if rec307(v)["jobFamilyName"] != "系列更正名称" {
		t.Fatal("current family name not resolved")
	}
	old := s.b.call("GET", "/api/v1/jobs/"+id+"?sourceVersion="+source, nil, "", 200)
	if rec307(old)["jobFamilyName"] != "资料 F-ADOPT" {
		t.Fatal("exact adoption drifted")
	}
	s.correct("job", v, map[string]any{"name": "只改职务名称"}, nil, 200)
	if !reflect.DeepEqual(s.raw("job", id)["records"].([]any)[0].(map[string]any)["adoption"], original) {
		t.Fatal("pure correction re-adopted")
	}
	v = s.refresh("job", v)
	body := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": first, "deactivationDate": "2005-01-01", "reactivationDate": "2007-01-01", "deactivationReason": "JOB_CLOSURE", "reactivationReason": "JOB_RESUMPTION"}
	s.b.call("POST", "/api/v1/jobs/"+id+"/lifecycle-pairs/insert", body, foundation.NewID(), 200)
	v = s.refresh("job", v)
	p := v["pairContext"].(map[string]any)
	stop, start := p["deactivation"].(map[string]any), p["reactivation"].(map[string]any)
	raw := s.raw("job", id)
	adopt := raw["records"].([]any)[2].(map[string]any)["adoption"]
	body = map[string]any{"expectedSourceVersion": v["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": start["recordId"], "reactivationRevision": start["revision"], "deactivationDate": "2006-01-01", "reactivationDate": "2007-01-01", "deactivationReason": "JOB_CLOSURE", "reactivationReason": "JOB_RESUMPTION"}
	s.b.call("POST", "/api/v1/jobs/"+id+"/lifecycle-pairs/maintain", body, foundation.NewID(), 200)
	if !reflect.DeepEqual(s.raw("job", id)["records"].([]any)[2].(map[string]any)["adoption"], adopt) {
		t.Fatal("stop-only maintenance re-adopted")
	}
	v = s.refresh("job", v)
	p = v["pairContext"].(map[string]any)
	stop, start = p["deactivation"].(map[string]any), p["reactivation"].(map[string]any)
	body = map[string]any{"expectedSourceVersion": v["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": start["recordId"], "reactivationRevision": start["revision"]}
	s.b.call("POST", "/api/v1/jobs/"+id+"/lifecycle-pairs/delete", body, foundation.NewID(), 200)
	if len(s.refresh("job", v)["timeline"].([]any)) != 1 {
		t.Fatal("pair delete failed")
	}
}
func TestStaffing307PermissionsAndIntervals(t *testing.T) {
	s := staffingSetup307(t)
	f := s.create("jobfamily", "F-RIGHTS", "")
	id := f["id"].(string)
	v := s.create("job", "RIGHTS", id)
	jid := v["id"].(string)
	s.f.exec(`DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action IN ('job.history.read','jobfamily.history.read')`, s.f.instance, s.b.subject)
	v = s.refresh("job", v)
	if rec307(v)["jobFamilyName"] != "受限" || rec307(v)["jobFamilyId"] != nil {
		t.Fatal("historical family label leaked")
	}
	s.b.call("GET", "/api/v1/jobs/"+jid+"/changes", nil, "", 403)
	s.b.call("GET", "/api/v1/jobs/"+jid+"?sourceVersion="+v["headSourceVersion"].(string), nil, "", 403)
	s.correct("job", v, map[string]any{"name": "无需新采用"}, nil, 200)
	v = s.refresh("job", v)
	s.correct("job", v, nil, map[string]any{"newEffectiveDate": "2001-01-01"}, 403)
	s.f.exec(`UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'jobFamilyId') WHERE instance_id=$1 AND principal_id=$2 AND action='job.record.correct'`, s.f.instance, s.b.subject)
	other := s.create("jobfamily", "OTHER-RIGHTS", "")
	s.correct("job", v, map[string]any{"jobFamilyId": other["id"]}, nil, 403)
	s.b.call("GET", "/api/v1/jobs?sort=jobFamilyName", nil, "", 403)
	// The complete future head, including unreadable Jobs, blocks shortening.
	f = s.refresh("jobfamily", f)
	s.add("jobfamily", f, "DEACTIVATE", "2030-01-01", nil, 409)
	finite := s.create("jobfamily", "FINITE", "")
	s.add("jobfamily", finite, "DEACTIVATE", "2030-01-01", nil, 200)
	body := map[string]any{"code": "FINITE-JOB", "jobFunctionId": s.b.functionForJobs("1900-01-01"), "name": "有限系列职务", "jobFamilyId": finite["id"], "effectiveDate": "2020-01-01", "operationType": "CREATE", "operationReason": "NEW_JOB"}
	out := s.b.call("POST", "/api/v1/jobs", body, foundation.NewID(), 409)
	if !strings.Contains(fmt.Sprint(out["message"]), "完整启用期间") {
		t.Fatal("finite-tail refusal missing")
	}
}

func TestStaffing307InverseCoverageAndArchivedSources(t *testing.T) {
	h := staffingSetup307(t)
	a, b := h.create("jobfamily", "INVERSE-A", ""), h.create("jobfamily", "INVERSE-B", "")
	j := h.create("job", "INVERSE-J", a["id"].(string))
	original := j["headSourceVersion"].(string)
	h.add("job", j, "ADD", "2010-01-01", map[string]any{"jobFamilyId": b["id"]}, 200)
	j = h.refresh("job", j)
	h.add("jobfamily", a, "DEACTIVATE", "2010-01-01", nil, 200)
	// Deletion would implicitly inherit A through its uncovered future tail.
	before := staffingFacts307(t, h.f)
	r := rec307(j)
	h.b.call("POST", "/api/v1/jobs/"+j["id"].(string)+"/records/"+r["recordId"].(string)+"/delete", map[string]any{"expectedSourceVersion": j["headSourceVersion"], "expectedRecordRevision": r["revision"]}, foundation.NewID(), 409)
	after := staffingFacts307(t, h.f)
	for table, hash := range before {
		if !strings.HasSuffix(table, "_terminal") && after[table] != hash {
			t.Fatal("refusal changed business fact", table)
		}
	}
	// Replace current adoption; archived original Job evidence must not prevent A maintenance.
	j = h.detail("job", j["id"].(string), j["timeline"].([]any)[0].(map[string]any)["recordId"].(string))
	h.correct("job", j, map[string]any{"jobFamilyId": b["id"]}, nil, 200)
	a = h.refresh("jobfamily", a)
	first := a["timeline"].([]any)[0].(map[string]any)
	a = h.detail("jobfamily", a["id"].(string), first["recordId"].(string))
	h.correct("jobfamily", a, nil, map[string]any{"newEffectiveDate": "2001-01-01"}, 200)
	old := h.b.call("GET", "/api/v1/jobs/"+j["id"].(string)+"?sourceVersion="+original, nil, "", 200)
	if rec307(old)["jobFamilyName"] != "资料 INVERSE-A" {
		t.Fatal("archived adoption drifted")
	}
	// Mirror a gap in both objects. Removing the Job pair expands its active
	// interval over A's gap and must fail atomically, even with hidden dependents.
	g := h.create("jobfamily", "GAP-F", "")
	v := h.create("job", "GAP-J", g["id"].(string))
	pair := func(k string, x map[string]any) {
		h.b.call("POST", base307(k)+"/"+x["id"].(string)+"/lifecycle-pairs/insert", map[string]any{"expectedSourceVersion": x["headSourceVersion"], "anchorRecordId": rec307(x)["recordId"], "deactivationDate": "2010-01-01", "reactivationDate": "2020-01-01", "deactivationReason": map[string]string{"job": "JOB_CLOSURE", "jobfamily": "FAMILY_CLOSURE"}[k], "reactivationReason": map[string]string{"job": "JOB_RESUMPTION", "jobfamily": "FAMILY_RESUMPTION"}[k], "changes": map[string]any{}}, foundation.NewID(), 200)
	}
	pair("job", v)
	pair("jobfamily", g)
	v = h.refresh("job", v)
	p := v["pairContext"].(map[string]any)
	stop, start := p["deactivation"].(map[string]any), p["reactivation"].(map[string]any)
	h.b.call("POST", "/api/v1/jobs/"+v["id"].(string)+"/lifecycle-pairs/delete", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": start["recordId"], "reactivationRevision": start["revision"]}, foundation.NewID(), 409)
	g = h.refresh("jobfamily", g)
	h.f.exec(`DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='job.read'`, h.f.instance, h.b.subject)
	result := h.correct("jobfamily", g, nil, map[string]any{"newEffectiveDate": "2021-01-01"}, 409)
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), v["id"].(string)) || strings.Contains(string(encoded), "GAP-J") {
		t.Fatal("hidden dependent leaked")
	}
}
