package department

import (
	"context"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
)

func TestProjectedReadRejectsChangedScopeAndUnplannedIntent(t *testing.T) {
	target := access.DepartmentTarget{EnterpriseID: "enterprise", DepartmentID: "department", ScopeFact: access.ScopeFact{Mode: "current_record", ManagementBUID: "bu", CurrentSourceVersion: "source", RecordID: "record", Revision: "revision", BusinessDate: "2026-11-06", EvaluatedAt: time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)}}
	frozen := target
	g := projectedReads{decisions: []access.Decision{{Allowed: true}, {Allowed: false}}, departments: []projectedDepartmentReads{{target: &frozen, object: 0, reference: 1}}, snapshot: d.Snapshot{Departments: []d.Timeline{{ID: "department"}}}.Indexed(), bus: map[string]projectedBUReads{"bu": {enterprise: "enterprise", reference: 1}}}
	for _, v := range []struct{ action, kind, id string }{{"department.read", "object", "object"}, {"department.reference.read", "reference", "reference"}} {
		got, err := g.Decide(context.Background(), intent(target, v.action, v.kind, nil))
		if err != nil || got.Allowed != (v.id != "reference") {
			t.Fatalf("projection lost distinct decision: %s: %+v %v", v.id, got, err)
		}
	}
	for _, change := range []func(*access.Intent){
		func(v *access.Intent) { v.DepartmentTarget.ScopeFact.CurrentSourceVersion = "new-source" },
		func(v *access.Intent) { v.DepartmentTarget.ScopeFact.ManagementBUID = "other-bu" },
		func(v *access.Intent) { v.DepartmentTarget.ScopeFact.BusinessDate = "2026-11-07" },
		func(v *access.Intent) {
			v.DepartmentTarget.ScopeFact.EvaluatedAt = target.ScopeFact.EvaluatedAt.Add(time.Second)
		},
		func(v *access.Intent) { v.DepartmentTarget.ScopeFact.Revision = "new-revision" },
		func(v *access.Intent) { v.EnterpriseID = "other-enterprise" },
		func(v *access.Intent) { v.BUID = "bu" },
		func(v *access.Intent) { v.Kind = "history" },
		func(v *access.Intent) { v.Action = "department.history.read" },
		func(v *access.Intent) { v.Fields = []string{"name"} },
	} {
		in := intent(target, "department.read", "object", nil)
		change(&in)
		if _, err := g.Decide(context.Background(), in); err == nil {
			t.Fatalf("accepted unplanned scope or intent: %+v", in)
		}
	}
	for _, in := range []access.Intent{{EnterpriseID: "other-enterprise", BUID: "bu", Action: "bu.reference.read", Kind: "reference"}, {EnterpriseID: "enterprise", BUID: "bu", Action: "bu.read", Kind: "object"}} {
		if _, err := g.Decide(context.Background(), in); err == nil {
			t.Fatal("accepted unplanned BU scope or purpose")
		}
	}
}
