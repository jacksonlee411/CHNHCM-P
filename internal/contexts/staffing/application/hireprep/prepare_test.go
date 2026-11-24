package hireprep

import (
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"testing"
)

type recordedIdentity struct {
	request IdentityRequest
	result  IdentityResult
}

func (r *recordedIdentity) CheckIdentity(_ context.Context, _ f.Credentials, in IdentityRequest) (IdentityResult, error) {
	r.request = in
	return r.result, nil
}

type recordedDecision struct{ expected Appointment }

func (d recordedDecision) CheckAppointment(_ context.Context, _ f.Credentials, ref DecisionRef, actual Appointment) error {
	if actual != d.expected || ref.Source != "appointment" || ref.Version != "decision-v1" {
		return f.Fail(409, "HIRE_BASIS", "采用与任用决定不符")
	}
	return nil
}

type sampleRule struct{ rule Rule }

func (s sampleRule) FirstHireRule(context.Context, f.Credentials, string) (Rule, error) {
	return s.rule, nil
}

func TestFirstEmployeeSample(t *testing.T) {
	person := f.NewID()
	source := IdentitySource{PersonID: person, PersonVersion: "original", DocumentID: "passport-row", DocumentRevision: "passport-rev"}
	p := &recordedIdentity{result: IdentityResult{Source: source}}
	rule := DirectEmployeeSample()
	rel := Choice{Complete: true, Eligible: []string{"original-relationship"}}
	job := Choice{Complete: true, Eligible: []string{"first-assignment"}}
	notice := Basis{Notice: "人事部 3 月 26 日发给甲的入职通知"}
	appointment := Appointment{Person: person, Employer: f.NewID(), Department: f.NewID(), Job: f.NewID(), Period: f.Period{From: "2027-04-01"}}
	var expected *IdentitySource
	var expectedRule string
	run := func(b Basis) (Preparation, error) {
		return PrepareFirst(context.Background(), Sources{People: p, Decisions: recordedDecision{appointment}, Rules: sampleRule{rule}}, f.Credentials{}, FirstRequest{Appointment: appointment, Basis: b, ExpectedIdentity: expected, ExpectedRuleVersion: expectedRule, Relationship: rel, Assignments: map[string]Choice{"original-relationship": job}})
	}
	got, err := run(notice)
	if err != nil || got.Identity.Source != source || got.Basis != notice || got.Reason != "direct_appointment" || got.PrimaryRelationship != rel.Eligible[0] || got.PrimaryAssignment != job.Eligible[0] || got.InitialState != "Active" || got.RuleVersion != "T02-employee-1" {
		t.Fatalf("first preparation: %+v %v", got, err)
	}
	expectedRule = "superseded-rule"
	if _, err := run(notice); err == nil {
		t.Fatal("prepared a previously selected rule after its version changed")
	}
	expectedRule = rule.Version
	if p.request.Purpose != "hire" || p.request.PersonID != person || p.request.Arrival != appointment.From || p.request.Relationship != "Employee" {
		t.Fatal("wrong purpose, person or date supplied to Personnel")
	}
	if p.request.RequireInspection || p.request.RequireKnownExpiry || !slices.Equal(p.request.AcceptedDocuments, []string{"resident_id", "passport"}) {
		t.Fatal(p.request)
	}
	ref := Basis{Decision: &DecisionRef{Source: "appointment", Version: "decision-v1"}}
	if got, err = run(ref); err != nil || got.Basis != ref {
		t.Fatal("did not reuse original decision")
	}
	if _, err = run(Basis{}); err == nil {
		t.Fatal("accepted missing basis")
	}
	wrong := Basis{Decision: &DecisionRef{Source: "unrelated", Version: "decision-v1"}}
	if _, err = run(wrong); err == nil {
		t.Fatal("unverified appointment")
	}
	expected = &source
	if _, err = run(notice); err != nil || p.request.Expected != expected {
		t.Fatal("expected source not rechecked", err)
	}
	p.result.Source.DocumentRevision = "renewed"
	if _, err = run(notice); err == nil {
		t.Fatal("stale document accepted")
	}
	p.result.Source = source
	expected = nil
	rel.Complete = false
	if _, err = run(notice); err == nil {
		t.Fatal("partial visibility used to choose a primary")
	}
	rel.Complete = true
	job.Eligible = append(job.Eligible, "other")
	if _, err = run(ref); err == nil {
		t.Fatal("arbitrary primary")
	}
	p.result = IdentityResult{Missing: []string{"document"}}
	got, err = run(Basis{})
	if err != nil || !slices.Equal(got.Identity.Missing, []string{"document"}) || got.PrimaryRelationship != "" {
		t.Fatal("wrong missing-field projection")
	}
	expected = &source
	if _, err = run(notice); err == nil {
		t.Fatal("submission accepted a previously adopted document now missing")
	}
}
