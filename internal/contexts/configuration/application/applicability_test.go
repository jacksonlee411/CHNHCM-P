package application

import (
	"testing"

	config "chnhcm.local/chnhcm/internal/contexts/configuration/public"
)

func TestPlanScopeRequiresEveryDimensionAndHonorsExclusions(t *testing.T) {
	a, b, pro, ops := "61056e13-fc8e-482b-82ed-2e2265bc8df7", "fe47e1a7-44df-49fd-af49-b12e62a13c3b", "04bf8ad4-dc7f-4d92-9268-0d66284768a4", "cd3b8995-4b8f-45f7-90ea-14d6d3e75cc1"
	scope := config.Scope{Mode: "SELECTED", Choices: []config.Choice{{Dimension: "legalEntity", ID: a}, {Dimension: "legalEntity", ID: b}, {Dimension: "jobFamily", ID: pro}, {Dimension: "jobFamily", ID: ops}, {Dimension: "jobFamily", ID: ops, Exclude: true}}}
	n, err := Normalize(scope)
	if err != nil {
		t.Fatal(err)
	}
	p := config.Policy{Mode: n.Mode}
	for _, choice := range n.Choices {
		p.Members = append(p.Members, config.Member{Choice: choice})
	}
	for _, tc := range []struct {
		entity, family string
		want           bool
	}{{a, pro, true}, {b, pro, true}, {b, ops, false}, {a, "", false}, {"", pro, false}} {
		got := Matches(p, map[string]string{"legalEntity": tc.entity, "jobFamily": tc.family})
		if got != tc.want {
			t.Fatalf("%v = %v", tc, got)
		}
	}
	if _, err = Normalize(config.Scope{}); err == nil {
		t.Fatal("empty condition accepted")
	}
	if !Matches(config.Policy{Mode: "ENTERPRISE"}, nil) {
		t.Fatal("explicit enterprise scope rejected")
	}
}
