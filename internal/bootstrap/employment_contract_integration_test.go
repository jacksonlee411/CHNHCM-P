//go:build integration

package bootstrap_test

import (
	"reflect"
	"testing"

	f "chnhcm.local/chnhcm/internal/foundation"
)

func TestRelationshipContractsRenewAndCloseOnlyTheSelectedRelationship(t *testing.T) {
	type M = map[string]any
	x := setupAssignment(t)
	b, db := x.h.b, x.h.f
	primary := x.hire["assignment"].(string)
	const browse = "2027-05-01"
	path := func(id string) string { return "/api/v1/assignments/" + id }
	head := func(id string) any {
		return b.call("GET", path(id)+"?businessAt="+browse, nil, "", 200)["sourceVersion"]
	}
	contracts := func(id string) []any {
		return b.call("GET", path(id)+"/contracts?businessAt="+browse, nil, "", 200)["items"].([]any)
	}
	policy := b.call("GET", "/api/v1/assignment-policies", nil, "", 200)
	b.call("POST", "/api/v1/assignment-policies", M{"expectedVersion": policy["version"], "row": M{"id": f.NewID(), "category": "Employee", "purpose": "secondary", "mode": "JobOnly", "from": "2027-04-01"}}, f.NewID(), 200)
	employer := x.h.create("LAW54", "CN", "legal_person")["id"].(string)
	b.call("POST", "/api/v1/legal-entities/"+employer+"/purposes", M{"expectedVersion": "", "row": M{"id": f.NewID(), "role": "employer", "kind": "declaration", "basis": "第二关系用工", "from": "2027-04-01"}}, f.NewID(), 200)
	parallel := b.call("POST", path(primary)+"/additional-assignments", M{"expectedSourceVersion": head(primary), "parallel": true, "employer": employer, "job": x.job, "department": x.dep, "from": browse, "notice": "独立工作关系"}, f.NewID(), 200)["assignment"].(string)
	originalSource := head(primary)
	cmd := func(id, num string) M {
		return M{"operation": "sign", "number": num, "signed": "2027-04-12", "from": "2027-05-01", "lastDay": "2027-06-30", "expectedSourceVersion": head(id)}
	}
	primarySign := cmd(primary, "A-1")
	key := f.NewID()
	db.exec("ALTER TABLE staffing.contract_outbox ADD CONSTRAINT fail_contract CHECK(false) NOT VALID")
	b.call("POST", path(primary)+"/contracts", primarySign, key, 503)
	if len(contracts(primary)) != 0 || db.scalar("SELECT count(*) FROM staffing.contract_submission") != 0 {
		t.Fatal("末步错误留下订立或回执")
	}
	db.exec("ALTER TABLE staffing.contract_outbox DROP CONSTRAINT fail_contract")
	a := b.call("POST", path(primary)+"/contracts", primarySign, key, 200)
	if again := b.call("POST", path(primary)+"/contracts", primarySign, key, 200); !reflect.DeepEqual(again, a) {
		t.Fatal("重送改变办理", again, a)
	}
	ref := func(v M) M { return M{"id": v["id"], "version": v["version"]} }
	parallelSign := cmd(parallel, "B-1")
	delete(parallelSign, "lastDay")
	z := b.call("POST", path(parallel)+"/contracts", parallelSign, f.NewID(), 200)
	before := contracts(parallel)
	renew := M{"operation": "renew", "expectedSourceVersion": head(primary), "ref": ref(a), "number": "A-2", "signed": "2027-06-06", "from": "2027-07-01", "lastDay": "2028-06-30"}
	wrong := M{}
	for k, v := range renew {
		wrong[k] = v
	}
	wrong["ref"] = ref(z)
	b.call("POST", path(primary)+"/contracts", wrong, f.NewID(), 409)
	next := b.call("POST", path(primary)+"/contracts", renew, f.NewID(), 200)
	after := contracts(primary)
	if len(after) != 2 || after[0].(M)["next"] != next["id"] || after[1].(M)["previous"] != a["id"] || after[0].(M)["signed"] != "2027-04-12" || after[0].(M)["from"] != "2027-05-01" || head(primary) != originalSource || !reflect.DeepEqual(before, contracts(parallel)) {
		t.Fatal("两关系的期限、前后继或任职来源错误", after)
	}
	leave := M{"expectedSourceVersion": head(parallel), "date": "2027-08-31", "meaning": "last-active", "notice": "只结束第二段工作关系及其合同"}
	b.call("POST", path(parallel)+"/leave", leave, f.NewID(), 409)
	leave["contracts"] = []M{ref(next)}
	b.call("POST", path(parallel)+"/leave", leave, f.NewID(), 409)
	leave["contracts"] = []M{ref(z)}
	db.exec("ALTER TABLE staffing.employment_outbox ADD CONSTRAINT fail_leave_contract CHECK(false) NOT VALID")
	lkey := f.NewID()
	b.call("POST", path(parallel)+"/leave", leave, lkey, 503)
	db.exec("ALTER TABLE staffing.employment_outbox DROP CONSTRAINT fail_leave_contract")
	if !reflect.DeepEqual(before, contracts(parallel)) {
		t.Fatal("失败时结束合同")
	}
	ended := b.call("POST", path(parallel)+"/leave", leave, lkey, 200)
	endedContract := contracts(parallel)[0].(M)
	ending := endedContract["processes"].([]any)[1].(M)
	if endedContract["firstInactive"] != "2027-09-01" || ending["leaveAction"] != ended["action"] || !reflect.DeepEqual(after, contracts(primary)) {
		t.Fatal("退出处置", endedContract)
	}
	// Ending only a contract preserves both the existing assignment and its publication.
	b.call("POST", path(primary)+"/contracts", M{"operation": "end", "ref": ref(next), "firstInactive": "2028-04-01", "basis": "双方已签终止书", "expectedSourceVersion": head(primary)}, f.NewID(), 200)
	if head(primary) != originalSource {
		t.Fatal("仅终止合同改变任职")
	}
	x.h.policy("UPDATE access.employment_grant SET person_id=$2 WHERE principal_id=$1 AND action='contract.read'", b.subject, f.NewID())
	b.call("GET", path(primary)+"/contracts", nil, "", 403)
	recovered := b.call("POST", "/api/v1/employment-contracts/submissions/"+key+"/resolve", nil, "", 200)
	if recovered["outcome"] != "accepted" || recovered["id"] != nil {
		t.Fatal("撤权后合同行为回执泄露", recovered)
	}
}
