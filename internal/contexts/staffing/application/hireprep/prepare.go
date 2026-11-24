// Package hireprep defines Staffing's read-only preparation contract. Personnel
// owns the underlying person and document, including purpose-bound rechecks.
package hireprep

import (
	person "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"strings"
)

type IdentityRequest = person.HireIdentityRequest
type IdentitySource = person.HireIdentitySource
type IdentityResult = person.HireIdentityResult

// CheckIdentity runs in the caller's transaction at submission time. A changed
// selected document or revoked purpose returns an error, never its old preview.
type Personnel interface {
	CheckIdentity(context.Context, f.Credentials, IdentityRequest) (IdentityResult, error)
}
type Rule struct {
	Version, Action, Relationship, InitialState, PrimaryPurpose string
	Reasons, AcceptedDocuments                                  []string
	RequireKnownExpiry, RequireInspection                       bool
}

// DirectEmployeeSample is the first synthetic customer's declared rule, not a
// fallback when an enterprise has no published policy.
func DirectEmployeeSample() Rule {
	return Rule{Version: "T02-employee-1", Action: "Hire", Relationship: "Employee", InitialState: "Active", PrimaryPurpose: "personnel_management", Reasons: []string{"direct_appointment"}, AcceptedDocuments: []string{"resident_id", "passport"}}
}

// DecisionRef identifies the existing decision, optionally its original roster line.
type DecisionRef struct{ Source, Version, Line string }

// Decisions checks the original decision under current authority. A different
// person, appointment, obsolete revision or inaccessible line is rejected.
type Appointment struct {
	Person, Employer, Department, Job string
	f.Period
}
type Decisions interface {
	CheckAppointment(context.Context, f.Credentials, DecisionRef, Appointment) error
}
type Rules interface {
	FirstHireRule(context.Context, f.Credentials, string) (Rule, error)
}

type Sources struct {
	Rules     Rules
	People    Personnel
	Decisions Decisions
}
type Basis struct {
	Decision *DecisionRef
	Notice   string
}
type Choice struct {
	Complete           bool // authority has checked the entire layer, including restricted targets
	Eligible           []string
	Current, Requested string
}
type FirstRequest struct {
	Appointment
	Reason, ExpectedRuleVersion string
	Basis                       Basis
	ExpectedIdentity            *IdentitySource
	Relationship                Choice
	Assignments                 map[string]Choice // per relationship
}
type Preparation struct {
	RuleVersion, Action, Reason, InitialState, PrimaryPurpose string
	PrimaryRelationship, PrimaryAssignment                    string
	Basis                                                     Basis
	Identity                                                  IdentityResult
}

func resolveChoice(choice Choice, label string) (string, error) {
	if !choice.Complete {
		return "", f.Fail(409, "HIRE_CHOICE_UNAVAILABLE", "无法核实"+label+"的完整范围")
	}
	options, current, requested := choice.Eligible, choice.Current, choice.Requested
	if requested != "" {
		if slices.Contains(options, requested) {
			return requested, nil
		}
		return "", f.Fail(409, "HIRE_CHOICE_CHANGED", label+"已不适用")
	}
	if current != "" && slices.Contains(options, current) {
		return current, nil
	}
	if len(options) == 1 && options[0] != "" {
		return options[0], nil
	}
	return "", f.Fail(409, "HIRE_CHOICE_REQUIRED", "请核对"+label)
}

// PrepareFirst consumes authoritative candidates for one first Employee
// relationship and its first assignment. It has no command or persistence port.
func PrepareFirst(ctx context.Context, sources Sources, cred f.Credentials, in FirstRequest) (Preparation, error) {
	var v Preparation
	person, arrival, basis := in.Person, in.From, in.Basis
	if !f.ValidID(person) || !in.Period.Valid() || !f.ValidID(in.Employer) || !f.ValidID(in.Department) || !f.ValidID(in.Job) {
		return v, f.Fail(400, "HIRE_INPUT", "入职人员、日期或拟任安排无效")
	}
	if sources.People == nil {
		return v, f.Fail(503, "HIRE_IDENTITY_UNAVAILABLE", "人员档案暂不可用")
	}
	if sources.Rules == nil {
		return v, f.Fail(503, "HIRE_RULE_UNAVAILABLE", "入职规则暂不可用")
	}
	rule, e := sources.Rules.FirstHireRule(ctx, cred, arrival)
	if e != nil {
		return v, e
	}
	if rule.Version == "" || rule.Relationship != "Employee" || rule.Action != "Hire" || rule.InitialState == "" || rule.PrimaryPurpose == "" || len(rule.AcceptedDocuments) == 0 {
		return v, f.Fail(409, "HIRE_RULE_UNAVAILABLE", "适用的直接入职规则不完整")
	}
	if in.ExpectedRuleVersion != "" && in.ExpectedRuleVersion != rule.Version {
		return v, f.Fail(409, "HIRE_RULE_CHANGED", "入职规则已变化，请重新核对")
	}
	identity, e := sources.People.CheckIdentity(ctx, cred, IdentityRequest{PersonID: person, Purpose: "hire", Relationship: rule.Relationship, Arrival: arrival, AcceptedDocuments: slices.Clone(rule.AcceptedDocuments), RequireKnownExpiry: rule.RequireKnownExpiry, RequireInspection: rule.RequireInspection, Expected: in.ExpectedIdentity})
	if e != nil {
		return v, e
	}
	if len(identity.Missing) > 0 {
		if in.ExpectedIdentity != nil {
			return v, f.Fail(409, "HIRE_IDENTITY", "原档已不能满足本次入职，请重新核对")
		}
		return Preparation{Identity: IdentityResult{Missing: identity.Missing}}, nil
	}
	if identity.Source.PersonID != person || identity.Source.PersonVersion == "" || identity.Source.DocumentID == "" || identity.Source.DocumentRevision == "" || rule.RequireInspection && identity.Source.InspectionRevision == "" {
		return v, f.Fail(409, "HIRE_IDENTITY", "原档核对来源不完整")
	}
	if in.ExpectedIdentity != nil && *in.ExpectedIdentity != identity.Source {
		return v, f.Fail(409, "HIRE_IDENTITY", "采用的原档或证件已变化，请重新核对")
	}
	if basis.Decision != nil {
		if basis.Decision.Source == "" || basis.Decision.Version == "" {
			return v, f.Fail(409, "HIRE_BASIS", "任用决定与本人不符或原件不可用")
		}
		if sources.Decisions == nil {
			return v, f.Fail(503, "HIRE_BASIS_UNAVAILABLE", "任用决定来源暂不可用")
		}
		if err := sources.Decisions.CheckAppointment(ctx, cred, *basis.Decision, in.Appointment); err != nil {
			return v, err
		}
		basis.Notice = ""
	} else if strings.TrimSpace(basis.Notice) == "" {
		return v, f.Fail(400, "HIRE_BASIS", "请填写可追溯的任用通知说明")
	}
	reason, e := resolveChoice(Choice{Eligible: rule.Reasons, Requested: in.Reason, Complete: true}, "入职原因")
	if e != nil {
		return v, e
	}
	rel, e := resolveChoice(in.Relationship, "主工作关系")
	if e != nil {
		return v, e
	}
	job, e := resolveChoice(in.Assignments[rel], "该关系内的主任职")
	if e != nil {
		return v, e
	}
	return Preparation{RuleVersion: rule.Version, Action: rule.Action, Reason: reason, InitialState: rule.InitialState, PrimaryPurpose: rule.PrimaryPurpose, PrimaryRelationship: rel, PrimaryAssignment: job, Basis: basis, Identity: identity}, nil
}
