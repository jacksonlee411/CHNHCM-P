package domain

import (
	"errors"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestExemptRecruitingAdoptionKeepsKnownCapacity(t *testing.T) {
	span := UsageSpan{From: "2027-04-01"}
	q := QuotaUsage{Managed: []UsageSpan{span}, Coverage: []SourceCoverage{{span, "M04", "assignments-v1"}, {span, "M05", "recruiting-v1"}}}
	one := 1
	quota := StaffingQuota{Records: []QuotaRecord{{Date: span.From, Amount: &one}}}
	position := QuotaPeriods{ID: "P", Records: []QuotaPeriod{{Date: span.From, Active: true}}}
	intent := UsageIntent{UsageSpan: span, Enterprise: "E", Category: "实习生", Purpose: "招聘", Owner: "M05"}
	rule := QuotaPolicy{UsageSpan: span, Enterprise: "E", Category: "实习生", Purpose: "招聘", Version: "R1", Mode: "PositionManaged"}
	uses, err := q.PlanUse(intent, []QuotaPolicy{rule}, quota, position)
	if err != nil || len(uses) != 1 || uses[0].Effect != "none" {
		t.Fatalf("免占编招聘的采用 %v, %v", uses, err)
	}
	uses[0].Source = "录用1"
	q.Uses = uses
	if count := q.At(span.From); count != (UsageCount{Managed: true, Known: true}) {
		t.Fatalf("已核齐的岗位用量 %v", count)
	}
	intent.Owner, intent.Purpose = "M04", "任职"
	rule.Purpose, rule.Version, rule.Consumes = "任职", "R2", true
	if segments, err := q.PlanUse(intent, []QuotaPolicy{rule}, quota, position); err != nil || len(segments) != 1 || segments[0].Effect != "occupied" {
		t.Fatalf("剩余常规名额 %v, %v", segments, err)
	}
	q.Coverage = q.Coverage[:1]
	if q.At(span.From).Known {
		t.Fatal("一份免占编录用不能补足招聘的覆盖交接")
	}
}

func TestQuotaTakeoverBoundaries(t *testing.T) {
	from, apr, jun := "2027-01-01", "2027-04-01", "2027-06-01"
	all := UsageSpan{from, ""}
	pos := QuotaPeriods{ID: "P", Records: []QuotaPeriod{{Date: from, Active: true}}}
	stopped := QuotaPeriods{ID: "P", Records: []QuotaPeriod{{Date: from, End: apr, Active: true}, {Date: apr}}}
	one, two, zero := 1, 2, 0
	before := StaffingQuota{Records: []QuotaRecord{{Date: from, Amount: &one}, {Date: apr, Amount: &two}, {Date: jun, Amount: &two}}}
	redated := StaffingQuota{Records: []QuotaRecord{{Date: from, Amount: &one}, {Date: "2027-04-20", Amount: &two}, {Date: jun, Amount: &two}}}
	evidence := QuotaUsage{Managed: []UsageSpan{all}, Coverage: []SourceCoverage{{all, "M04", "assignments-v1"}, {all, "M05", "recruiting-v1"}}}
	if c := evidence.At(apr); !c.Known || c.Occupied+c.Reserved+c.Exempt != 0 {
		t.Fatal(c)
	}
	if e := (QuotaUsage{}).CheckTightening(before, redated, pos, pos); e != nil {
		t.Fatal(e)
	}
	gap := evidence
	gap.Coverage = []SourceCoverage{{all, "M05", "v1"}, {UsageSpan{from, apr}, "M04", "v2"}, {UsageSpan{"2027-04-20", ""}, "M04", "v3"}}
	for _, op := range []struct {
		q StaffingQuota
		p QuotaPeriods
	}{{redated, pos}, {StaffingQuota{}, pos}, {before, stopped}} {
		if e := gap.CheckTightening(before, op.q, pos, op.p); e == nil || !strings.Contains(e.(*foundation.Fault).Message, apr) {
			t.Fatalf("缺口首日 %v", e)
		}
	}
	use := PositionUse{UsageSpan: UsageSpan{"2027-04-10", "2027-04-15"}, Owner: "M04", Source: "A", Policy: "v1", Effect: "occupied"}
	other := use
	other.Source = "B"
	evidence.Uses = []PositionUse{use, other}
	if e := evidence.CheckTightening(before, redated, pos, pos); e == nil || !strings.Contains(e.(*foundation.Fault).Message, use.From) {
		t.Fatalf("缺额首日 %v", e)
	}
	use.Effect = "exempt"
	evidence.Uses = []PositionUse{use}
	if e := evidence.CheckTightening(before, StaffingQuota{Records: []QuotaRecord{{Date: from, Amount: &zero}}}, pos, pos); e != nil {
		t.Fatal(e)
	}
	if e := evidence.CheckTightening(before, before, pos, stopped); e == nil {
		t.Fatal("免占编任职应阻止停岗")
	}
	intent := UsageIntent{UsageSpan: UsageSpan{apr, jun}, Enterprise: "E", Category: "派遣", Purpose: "任职", Owner: "M04"}
	rule := QuotaPolicy{UsageSpan: all, Enterprise: "E", Category: "派遣", Purpose: "任职", Version: "R1", Mode: "PositionManaged", Consumes: true}
	plan := func(rules []QuotaPolicy, p QuotaPeriods) ([]PositionUse, error) {
		return evidence.PlanUse(intent, rules, before, p)
	}
	for _, rules := range [][]QuotaPolicy{nil, {rule, rule}} {
		if _, err := plan(rules, pos); err == nil {
			t.Fatal("缺失或歧义规则被接受")
		}
	}
	foreign := rule
	foreign.Enterprise = "another"
	if _, err := plan([]QuotaPolicy{foreign}, pos); err == nil {
		t.Fatal("采用了另一企业规则")
	}
	exempt := rule
	exempt.From = "2027-04-12"
	exempt.Consumes = false
	exempt.Version = "R2"
	rule.Until = exempt.From
	segments, err := plan([]QuotaPolicy{rule, exempt}, pos)
	if err != nil || len(segments) != 2 || segments[0].Effect != "occupied" || segments[0].Policy != "R1" || segments[0].Until != exempt.From || segments[1].Effect != "exempt" || segments[1].Policy != "R2" {
		t.Fatal(segments, err)
	}
	rule.Mode = "JobOnly"
	rule.Consumes = false
	rule.Until = ""
	segments, err = plan([]QuotaPolicy{rule}, QuotaPeriods{})
	if err != nil || len(segments) != 1 || segments[0].Effect != "none" {
		t.Fatal(segments, err)
	}
	if _, err = plan([]QuotaPolicy{rule}, pos); err == nil {
		t.Fatal("仅按职务任职绑定了岗位")
	}
}

func TestExchangeValidatesEachDestinationAndFutureCommitment(t *testing.T) {
	span := UsageSpan{From: "2027-04-01"}
	before := QuotaUsage{Managed: []UsageSpan{span}, Coverage: []SourceCoverage{{span, "M04", "assignment-source"}, {span, "M05", "offer-source"}},
		Uses: []PositionUse{{UsageSpan: span, Owner: "M04", Source: "real-current", Policy: "ordinary", Effect: "occupied", Assignment: "third"},
			{UsageSpan: span, Owner: "M04", Source: "outgoing", Policy: "exempt", Effect: "exempt", Assignment: "first"}}}
	one := 1
	quota := StaffingQuota{Records: []QuotaRecord{{Date: span.From, Amount: &one}}}
	for _, third := range []string{"occupied", "reserved"} {
		after := before
		after.Uses = append([]PositionUse{}, before.Uses[0])
		if third == "reserved" {
			after.Uses[0].Effect = third
			after.Uses[0].Owner = "M05"
		}
		after.Uses = append(after.Uses, PositionUse{UsageSpan: UsageSpan{From: "2027-05-01", Until: "2027-05-15"}, Owner: "M04", Source: "incoming", Policy: "exempt", Effect: "exempt", Assignment: "second"},
			PositionUse{UsageSpan: UsageSpan{From: "2027-05-15"}, Owner: "M04", Source: "incoming", Policy: "counted", Effect: "occupied", Assignment: "second"})
		e := after.CheckExchange(before, quota, "first", "second")
		var fault *foundation.Fault
		if !errors.As(e, &fault) || fault.Code != "M03_QUOTA_SHORTAGE" || !strings.Contains(fault.Message, "2027-05-15 起缺 1 编") {
			t.Fatal(third, e)
		}
	}
}
