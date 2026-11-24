package domain

import (
	"context"
	"testing"
)

func TestT07CompletionPreservesClassificationAndValidatesReason(t *testing.T) {
	base := jointFixture("F7")
	before := append([]Record(nil), base...)
	out, e := Calculate(context.Background(), base, "f", "2020-01-01", []Item{jupdate("first", "f", "01", "", "甲", "Alpha"), jadd("complete", "06", "CHANGE", "乙", "Beta")})
	if e != nil {
		t.Fatal(e)
	}
	got := out.Records[0]
	if got.Type != "CREATE" || got.Reason != "NEW_ORGANIZATION" || got.TypeLabel != base[0].TypeLabel || got.ReasonLabel != base[0].ReasonLabel {
		t.Fatal("implicit classification change", got)
	}
	for _, reason := range []string{"ORG_RESTRUCTURING", "BUSINESS_RESUMPTION"} {
		it := jupdate("reason", "s1", "", "DEACTIVATE", "", "")
		it.Reason = reason
		out, e = Calculate(context.Background(), base, "f", "2020-01-01", []Item{it})
		if reason == "ORG_RESTRUCTURING" {
			if e != nil || out.Records[1].Reason != reason || out.Records[1].ReasonLabel != OperationReasons("DEACTIVATE")[reason] {
				t.Fatal("new valid S reason", out, e)
			}
		} else if e == nil {
			t.Fatal("wrong classification reason accepted")
		}
	}
	for i := range base {
		if base[i] != before[i] {
			t.Fatal("input mutated")
		}
	}
}
