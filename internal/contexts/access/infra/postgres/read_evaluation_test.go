package postgres

import (
	"context"
	"reflect"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
)

func TestCompactReadUsesSameCurrentScopeAndFieldUnions(t *testing.T) {
	ent, bu, dept, source, record, rev := "12345678-abcd-4123-8123-123456789ab1", "12345678-abcd-4123-8123-123456789ab2", "12345678-abcd-4123-8123-123456789ab3", "12345678-abcd-4123-8123-123456789ab4", "12345678-abcd-4123-8123-123456789ab5", "12345678-abcd-4123-8123-123456789ab6"
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	g := &guard{grants: []access.Grant{{Action: "department.read", ScopeKind: "bu", ScopeID: bu, ReadFields: []string{"code", "name"}}, {Action: "department.reference.read", ScopeKind: "department", ScopeID: dept, ReadFields: []string{"name"}}, {Action: "department.read", ScopeKind: "department", ScopeID: dept, ReadFields: []string{"departmentPath"}}, {Action: "department.record.correct", ScopeKind: "enterprise", ScopeID: ent, WriteFields: []string{"name"}}}}
	target := access.DepartmentTarget{EnterpriseID: ent, DepartmentID: dept, ScopeFact: access.ScopeFact{ManagementBUID: bu, CurrentSourceVersion: source, RecordID: record, Revision: rev, Mode: "current_record", BusinessDate: "2026-11-06", EvaluatedAt: now}}
	for _, action := range []string{"department.read", "department.reference.read", "bu.read"} {
		for index, change := range []func(*access.DepartmentTarget){func(*access.DepartmentTarget) {}, func(v *access.DepartmentTarget) { v.ScopeFact.ManagementBUID = "12345678-abcd-4123-8123-123456789ab7" }, func(v *access.DepartmentTarget) { v.DepartmentID = "12345678-abcd-4123-8123-123456789ab8" }, func(v *access.DepartmentTarget) { v.ScopeFact.BusinessDate = "2026-11-05" }, func(v *access.DepartmentTarget) { v.ScopeFact.CurrentSourceVersion = "invalid" }} {
			v := target
			change(&v)
			in := access.Intent{Action: action, Kind: "object", EnterpriseID: ent, DepartmentTarget: &v}
			want := []string{}
			if index < 3 {
				if action == "department.reference.read" && index != 2 {
					want = []string{"name"}
				}
				if action == "department.read" {
					if index == 0 {
						want = []string{"code", "departmentPath", "name"}
					}
					if index == 1 {
						want = []string{"departmentPath"}
					}
					if index == 2 {
						want = []string{"code", "name"}
					}
				}
			}
			results, err := g.DecideReads(context.Background(), []access.ReadIntent{{EnterpriseID: in.EnterpriseID, DepartmentTarget: &v, Action: action, Kind: "object"}})
			if err != nil || len(results.Index) != 1 {
				t.Fatal(err, results)
			}
			got := results.Values[results.Index[0]]
			if got.Allowed != (len(want) > 0) || !reflect.DeepEqual(got.ReadFields, want) || len(got.WriteFields) != 0 {
				t.Fatal("incorrect current scope/field decision", action, index, got, want)
			}
		}
	}
}

func TestCommandRequiresOneCompleteGrantPerRequirement(t *testing.T) {
	g := &guard{grants: []access.Grant{
		{Action: "bu.record.correct", ScopeKind: "enterprise", ScopeID: "e", WriteFields: []string{"name"}},
		{Action: "bu.record.correct", ScopeKind: "enterprise", ScopeID: "e", WriteFields: []string{"nameEn"}},
	}}
	in := access.Intent{Action: "bu.record.correct", EnterpriseID: "e", Fields: []string{"name", "nameEn"}}
	if g.evaluate(in).Allowed {
		t.Fatal("partial grants were combined")
	}
	in.Fields = []string{"name"}
	if !g.evaluate(in).Allowed {
		t.Fatal("complete grant was rejected")
	}
	in.Requirements = []access.Intent{{Action: "bu.record.correct", EnterpriseID: "other", ItemID: "second", Fields: []string{"name"}}}
	if d := g.evaluate(in); d.Allowed || d.DeniedItemID != "second" {
		t.Fatal("required scope was bypassed", d)
	}
}

func TestReadDictionaryPreservesLargeGrantSetsAndInvalidScopeDenials(t *testing.T) {
	id := func(last string) string { return "12345678-abcd-4123-8123-123456789a" + last }
	ent, bu, a, b, other := id("b1"), id("b2"), id("b3"), id("b4"), id("b5")
	grants := make([]access.Grant, 64)
	for i := range grants {
		grants[i] = access.Grant{Action: "department.read", ScopeKind: "department", ScopeID: other, ReadFields: []string{"code"}}
	}
	grants = append(grants, access.Grant{Action: "department.read", ScopeKind: "department", ScopeID: a, ReadFields: []string{"name"}}, access.Grant{Action: "department.read", ScopeKind: "department", ScopeID: b, ReadFields: []string{"departmentPath"}})
	g := &guard{grants: grants}
	first := access.DepartmentTarget{EnterpriseID: ent, DepartmentID: a, ScopeFact: access.ScopeFact{Mode: "current_record", ManagementBUID: bu, CurrentSourceVersion: id("b6"), RecordID: id("b7"), Revision: id("b8"), BusinessDate: "2026-11-08", EvaluatedAt: time.Date(2026, 11, 8, 4, 0, 0, 0, time.UTC)}}
	second, bad, day := first, first, first
	second.DepartmentID = b
	bad.ScopeFact.Revision = "invalid"
	day.ScopeFact.BusinessDate = "2026-11-09"
	inputs := []access.ReadIntent{}
	for _, target := range []*access.DepartmentTarget{&first, &first, &second, &bad, &day} {
		inputs = append(inputs, access.ReadIntent{DepartmentTarget: target, EnterpriseID: ent, Action: "department.read", Kind: "object"})
	}
	got, err := g.DecideReads(context.Background(), inputs)
	if err != nil || len(got.Index) != 5 {
		t.Fatal(err, got)
	}
	for i, want := range [][]string{{"name"}, {"name"}, {"departmentPath"}, {}, {}} {
		decision := got.Values[got.Index[i]]
		if decision.Allowed != (i < 3) || !reflect.DeepEqual(decision.ReadFields, want) {
			t.Fatalf("scope %d: %+v", i, decision)
		}
	}
	if got.Index[0] != got.Index[1] || got.Index[0] == got.Index[2] {
		t.Fatal("grant sequence dictionary lost identity")
	}
	// A new request with different current grants cannot reuse the old union.
	g = &guard{}
	denied, err := g.DecideReads(context.Background(), inputs[:1])
	if err != nil || denied.Values[denied.Index[0]].Allowed {
		t.Fatal("read union survived revocation", err)
	}
}

func TestAssignmentWindowUsesServiceTimeForReadAndWrite(t *testing.T) {
	from := time.Date(2026, 11, 23, 1, 0, 0, 0, time.UTC)
	until := from.Add(time.Hour)
	obj := "12345678-abcd-4123-8123-123456789ab1"
	g := &guard{windows: []timedGrant{
		{access.Grant{Action: "bu.read", ScopeKind: "enterprise", ScopeID: obj, ReadFields: []string{"name"}}, from, until},
		{access.Grant{Action: "bu.record.correct", ScopeKind: "enterprise", ScopeID: obj, WriteFields: []string{"name"}}, from, until},
	}}
	for _, tc := range []struct {
		at      time.Time
		allowed bool
	}{{from.Add(-time.Microsecond), false}, {from, true}, {until.Add(-time.Microsecond), true}, {until, false}} {
		g.selectAt(tc.at)
		for _, intent := range []access.Intent{{Action: "bu.read", EnterpriseID: obj}, {Action: "bu.record.correct", EnterpriseID: obj, Fields: []string{"name"}}} {
			if g.evaluate(intent).Allowed != tc.allowed {
				t.Fatal(tc.at, intent.Action)
			}
		}
	}
}

func TestPermissionOnlyRechecksTheRequestedFields(t *testing.T) {
	obj := "12345678-abcd-4123-8123-123456789ab1"
	surviving := access.Grant{Action: "bu.record.correct", ScopeKind: "enterprise", ScopeID: obj, WriteFields: []string{"name"}}
	proxy := surviving
	proxy.WriteFields = []string{"name", "nameEn"}
	g := &guard{grants: []access.Grant{surviving, proxy}, boundary: time.Now().Add(time.Hour)}
	in := access.Intent{Action: surviving.Action, EnterpriseID: obj, Fields: []string{"name"}, PermissionOnly: true}
	d, err := g.Decide(context.Background(), in)
	if err != nil || !d.Allowed || len(d.WriteFields) != 0 {
		t.Fatal(d, err)
	}
	g.grants, g.changedWindow = []access.Grant{surviving}, true
	if err := g.ValidateUse(); err != nil {
		t.Fatal("the remaining grant still covers the command", err)
	}
	g.grants, g.changedWindow = nil, true
	if err := g.ValidateUse(); err == nil {
		t.Fatal("command remained allowed after its last grant")
	}
}
