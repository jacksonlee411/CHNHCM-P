//go:build integration

package bootstrap_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	enterprise "chnhcm.local/chnhcm/internal/contexts/organization/application/enterprise"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	db "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func TestStaffing314Directory(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	base := base307("jobfunction")
	clock := f.runtime.Staffing.(app.Service).Store.(*db.Store)
	setDay := func(day int) {
		clock.Clock = func() time.Time { return time.Date(2027, 4, day, 12, 0, 0, 0, time.UTC) }
	}
	setDay(1)
	body := map[string]any{"code": " fn-01 ", "name": "管理类", "operationType": "CREATE", "operationReason": "NEW_JOB_FUNCTION"}
	h.create("jobfamily", "FN-01", "")
	key := foundation.NewID()
	created := h.b.call("POST", base, body, key, 201)
	h.b.call("POST", base, body, key, 201)
	resolved := h.b.call("POST", base+"/submissions/"+key+"/resolve", map[string]any{}, "", 200)
	if created["sourceVersion"] != resolved["sourceVersion"] || created["objectId"] != resolved["objectId"] || resolved["kind"] != "jobfunction" {
		t.Fatal("original receipt mismatch")
	}
	id := created["objectId"].(string)
	v := h.detail("jobfunction", id, "")
	if rec307(v)["effectiveDate"] != "1900-01-01" || v["code"] != "FN-01" {
		t.Fatalf("unexpected first record %v", v)
	}
	body["code"] = "FN-01"
	dup := h.b.call("POST", base, body, foundation.NewID(), 409)
	if fault := dup["failure"].(map[string]any); fault["code"] != "M03_CODE_CONFLICT" || fault["field"] != "code" {
		t.Fatalf("duplicate: %v", dup)
	}
	h.add("jobfunction", v, "ADD", "2027-04-02", map[string]any{"name": "管理职类"}, 200)
	reader := f.login("reader", f.readPassword)
	for day, want := range []string{"管理类", "管理职类"} {
		setDay(day + 1)
		list := reader.call("GET", base+"?q=FN-01", nil, "", 200)
		if list["totalCount"] != float64(1) || list["items"].([]any)[0].(map[string]any)["name"] != want {
			t.Fatalf("wrong name on day %d", day+1)
		}
	}
	in := map[string]any{"draftId": foundation.NewID(), "query": map[string]any{"q": "FN-01"}}
	draft := reader.call("POST", base+"/selection/start", in, "", 200)
	in["expectedRevision"] = draft["revision"]
	rows := csv307(t, reader, base+"/export", in, 200)
	if !reflect.DeepEqual(rows, [][]string{{"编码", "名称", "英文名称", "说明", "状态", "生效日期"}, {"FN-01", "管理职类", "", "", "启用", "2027-04-02"}}) {
		t.Fatalf("CSV: %v", rows)
	}
	reader.call("POST", base, body, foundation.NewID(), 403)
	f.exec("UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'name') WHERE principal_id=$1 AND action='jobfunction.read'", reader.subject)
	detail := reader.call("GET", base+"/"+id, nil, "", 200)
	if _, shown := rec307(detail)["name"]; shown {
		t.Fatal("restricted name exposed")
	}
	csv307(t, reader, base+"/export", in, 403)
	f.exec("UPDATE access.subject_grant SET write_fields=array_remove(write_fields,'name') WHERE principal_id=$1 AND action='jobfunction.record.correct'", h.b.subject)
	h.correct("jobfunction", h.refresh("jobfunction", v), map[string]any{"name": "越权更名"}, nil, 403)
	if f.scalar("SELECT count(*) FROM staffing.job_function_source") != 2 || f.scalar("SELECT count(*) FROM staffing.job_function_outbox") != 2 {
		t.Fatal("failed or replayed command added facts")
	}
}

func TestStaffing314EnterpriseRace(t *testing.T) {
	for _, mode := range []string{"create", "redate"} {
		t.Run(mode, func(t *testing.T) {
			enterpriseRace307(t, "2005-01-01", func(h *staffing307) staff.Command {
				cmd := staff.Command{Kind: staff.JobFunction, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: "FLOOR", Name: name307("管理类"), EffectiveDate: "2001-01-01", OperationType: "CREATE", OperationReason: "NEW_JOB_FUNCTION"}}
				if mode == "redate" {
					v := h.b.call("POST", base307("jobfunction"), map[string]any{"code": "FLOOR", "name": "管理类", "effectiveDate": "2010-01-01", "operationType": "CREATE", "operationReason": "NEW_JOB_FUNCTION"}, foundation.NewID(), 201)
					cmd.ObjectID = v["objectId"].(string)
					cmd.Input = staff.Input{Action: "REDATE", RecordID: v["selectedRecordId"].(string), ExpectedRecordRevision: v["selectedRecordRevision"].(string), ExpectedSourceVersion: v["sourceVersion"].(string), NewEffectiveDate: "2001-01-01"}
				}
				return cmd
			})
		})
	}
}

type unavailableFloor314 struct{ staff.OrganizationDependencies }

func (unavailableFloor314) EarliestStaffingDate(context.Context, string) (string, error) {
	return "", errors.New("injected reverse port failure")
}

func TestStaffing314EnterpriseFloor(t *testing.T) {
	h := staffingSetup307(t)
	f, b := h.f, h.b
	v := b.call("POST", base307("jobfunction"), map[string]any{"code": "INACTIVE", "name": "管理类", "effectiveDate": "2001-01-01", "operationType": "CREATE", "operationReason": "NEW_JOB_FUNCTION"}, foundation.NewID(), 201)
	h.add("jobfunction", h.detail("jobfunction", v["objectId"].(string), ""), "DEACTIVATE", "2003-01-01", nil, 200)
	ent := b.call("GET", "/api/v1/enterprise/context", nil, "", 200)
	path := "/api/v1/enterprise/" + ent["id"].(string)
	parent := b.call("GET", path, nil, "", 200)
	r := rec307(parent)
	move := func(date string, status int) map[string]any {
		return b.call("POST", path+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": parent["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": date}, foundation.NewID(), status)
	}
	refusal := move("2005-01-01", 409)
	if refusal["code"] != "M02_REFERENCE_CONFLICT" {
		t.Fatalf("unexpected enterprise rejection: %v", refusal)
	}
	store := f.runtime.Enterprise.(enterprise.Service).Store.(*orgdb.Store)
	original := store.Staffing
	defer func() { store.Staffing = original }()
	for _, port := range []staff.OrganizationDependencies{unavailableFloor314{original}, nil} {
		store.Staffing = port
		before := f.scalar("SELECT count(*) FROM organization.enterprise_terminal")
		move("2000-01-01", 503)
		if f.scalar("SELECT count(*) FROM organization.enterprise_terminal") != before {
			t.Fatal("failure persisted a terminal")
		}
	}
	parentAfter := b.call("GET", path, nil, "", 200)
	if parentAfter["headSourceVersion"] != parent["headSourceVersion"] || rec307(parentAfter)["effectiveDate"] != "1900-01-01" {
		t.Fatal("enterprise head changed on rejection")
	}
}

func TestStaffing314FixedPeriod(t *testing.T) {
	h := staffingSetup307(t)
	const kind = "jobfunction"
	v := h.create(kind, "PAIR", "")
	id := v["id"].(string)
	path := base307(kind) + "/" + id
	insert := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "anchorRecordId": rec307(v)["recordId"], "deactivationDate": "2010-01-01", "reactivationDate": "2011-01-01", "deactivationReason": "FUNCTION_CLOSURE", "reactivationReason": "FUNCTION_RESUMPTION", "changes": map[string]any{"description": "恢复时的说明"}}
	saved := h.b.call("POST", path+"/lifecycle-pairs/insert", insert, foundation.NewID(), 200)
	targets := func() map[string]any {
		d := h.detail(kind, id, saved["selectedRecordId"].(string))
		pair := d["pairContext"].(map[string]any)
		stop, start := pair["deactivation"].(map[string]any), pair["reactivation"].(map[string]any)
		return map[string]any{"expectedSourceVersion": d["headSourceVersion"], "deactivationRecordId": stop["recordId"], "deactivationRevision": stop["revision"], "reactivationRecordId": start["recordId"], "reactivationRevision": start["revision"]}
	}
	change := targets()
	change["deactivationDate"], change["reactivationDate"] = "2010-02-01", "2011-01-01"
	change["deactivationReason"], change["reactivationReason"] = "FUNCTION_CLOSURE", "FUNCTION_RESUMPTION"
	h.b.call("POST", path+"/lifecycle-pairs/maintain", change, foundation.NewID(), 200)
	h.b.call("POST", path+"/lifecycle-pairs/delete", targets(), foundation.NewID(), 200)
	if len(h.detail(kind, id, "")["timeline"].([]any)) != 1 {
		t.Fatal("pair delete left active boundaries")
	}
	original := h.b.call("GET", path+"?sourceVersion="+saved["sourceVersion"].(string)+"&selectedRecordId="+saved["selectedRecordId"].(string), nil, "", 200)
	if rec307(original)["description"] != "恢复时的说明" || rec307(original)["revision"] != saved["selectedRecordRevision"] {
		t.Fatal("pair deletion changed archived recovery")
	}
}
