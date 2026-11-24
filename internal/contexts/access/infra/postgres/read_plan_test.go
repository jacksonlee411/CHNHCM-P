package postgres

import (
	"context"
	"fmt"
	"math/rand/v2"
	"reflect"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
)

func TestQuery131ReadPlanMatchesExistingDecisions(t *testing.T) {
	id := func(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }
	ctx := context.Background()
	at := time.Date(2026, 11, 9, 8, 0, 0, 0, time.UTC)
	actions := []access.ReadAction{access.DepartmentRead, access.DepartmentReference, access.DepartmentHistory, access.DepartmentExport, access.BURead, access.BUReference}
	rng := rand.New(rand.NewPCG(131, 9))
	checks := 0
	for trial := 0; trial < 80; trial++ {
		g := &guard{p: access.Principal{ID: id(2), SessionID: id(3), InstanceID: id(4)}}
		for i := 0; i < trial+1; i++ {
			kind := []string{"enterprise", "bu", "department", "invalid"}[rng.IntN(4)]
			scope := id(1 + rng.IntN(2))
			if kind == "bu" {
				scope = id(10 + rng.IntN(3))
			}
			if kind == "department" {
				scope = id(20 + rng.IntN(4))
			}
			fields := []string{}
			for _, f := range []string{"code", "name", "shortName", "effectiveDate", "status", "managementBuId", "departmentPath"} {
				if rng.IntN(2) == 1 {
					fields = append(fields, f)
				}
			}
			g.grants = append(g.grants, access.Grant{ScopeKind: kind, ScopeID: scope, Action: string(actions[rng.IntN(len(actions))]), ReadFields: fields})
		}
		for _, action := range actions {
			p, err := g.PlanReads(ctx, id(1), at, action)
			if err != nil {
				t.Fatal(err)
			}
			for bu := 10; bu < 13; bu++ {
				for dep := 20; dep < 24; dep++ {
					for _, mode := range []string{"current_record", "future_first", "invalid"} {
						r := access.ReadIntent{Action: string(action), Kind: "object", EnterpriseID: id(1), BUID: id(bu)}
						if action != access.BURead && action != access.BUReference {
							r.BUID = ""
							r.DepartmentTarget = &access.DepartmentTarget{EnterpriseID: id(1), DepartmentID: id(dep), ScopeFact: access.ScopeFact{EvaluatedAt: at, BusinessDate: p.Day, Mode: mode, ManagementBUID: id(bu), CurrentSourceVersion: id(30), RecordID: id(31), Revision: id(32)}}
						}
						got, err := g.DecidePlan(ctx, p, r)
						if err != nil {
							t.Fatal(err)
						}
						want, err := g.Decide(ctx, r.Intent())
						if err != nil {
							t.Fatal(err)
						}
						if got.Allowed != want.Allowed || !reflect.DeepEqual(got.ReadFields, want.ReadFields) {
							t.Fatalf("trial=%d action=%s bu=%d dep=%d mode=%s got=%+v want=%+v", trial, action, bu, dep, mode, got, want)
						}
						if action != access.DepartmentHistory && action != access.DepartmentExport {
							batch, e := g.DecideReads(ctx, []access.ReadIntent{r})
							if e != nil || !reflect.DeepEqual(batch.Values[batch.Index[0]], want) {
								t.Fatal("existing batch differs", e)
							}
						}
						checks++
					}
				}
			}
		}
	}
	t.Logf("%d guarded read-plan decisions equal existing Decide; four read actions also equal DecideReads", checks)
}

func TestQuery131ReadPlanCurrentScopeAndRequestBinding(t *testing.T) {
	id := func(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }
	at := time.Date(2026, 11, 9, 8, 0, 0, 0, time.UTC)
	ctx := context.Background()
	g := &guard{p: access.Principal{ID: id(1), SessionID: id(2), InstanceID: id(3)}, grants: []access.Grant{
		{ScopeKind: "bu", ScopeID: id(10), Action: "department.read", ReadFields: []string{"code"}},
		{ScopeKind: "department", ScopeID: id(20), Action: "department.read", ReadFields: []string{"name"}},
		{ScopeKind: "bu", ScopeID: id(11), Action: "department.history.read", ReadFields: []string{"name"}},
	}}
	r := access.ReadIntent{Action: "department.read", Kind: "object", EnterpriseID: id(4), DepartmentTarget: &access.DepartmentTarget{EnterpriseID: id(4), DepartmentID: id(20), ScopeFact: access.ScopeFact{EvaluatedAt: at, BusinessDate: "2026-11-09", Mode: "current_record", ManagementBUID: id(10), CurrentSourceVersion: id(30), RecordID: id(31), Revision: id(32)}}}
	p, e := g.PlanReads(ctx, id(4), at, access.DepartmentRead)
	if e != nil {
		t.Fatal(e)
	}
	check := func(want []string) {
		t.Helper()
		d, e := g.DecidePlan(ctx, p, r)
		if e != nil || !reflect.DeepEqual(d.ReadFields, want) || d.Allowed != (len(want) > 0) {
			t.Fatal(d, e, want)
		}
	}
	check([]string{"code", "name"})
	r.DepartmentTarget.ScopeFact.ManagementBUID = id(11)
	check([]string{"name"}) // Explicit department grant survives a BU transfer.
	r.DepartmentTarget.DepartmentID = id(21)
	check([]string{})
	r.Action = "department.history.read"
	hp, e := g.PlanReads(ctx, id(4), at, access.DepartmentHistory)
	if e != nil {
		t.Fatal(e)
	}
	d, e := g.DecidePlan(ctx, hp, r)
	if e != nil || !d.Allowed {
		t.Fatal("current B scope must permit old A history independently", d, e)
	}
	r.DepartmentTarget.ScopeFact.BusinessDate = "2026-11-08"
	d, e = g.DecidePlan(ctx, hp, r)
	if e != nil || d.Allowed {
		t.Fatal("historical day became current authorization", d, e)
	}
	other := &guard{p: g.p}
	if _, e = other.DecidePlan(ctx, hp, r); e == nil {
		t.Fatal("old request plan survived a new guard/revocation")
	}
	if _, e = g.PlanReads(ctx, id(4), at, access.ReadAction("department.record.correct")); e == nil {
		t.Fatal("write accepted")
	}
}
