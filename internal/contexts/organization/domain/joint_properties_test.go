package domain

import (
	"context"
	"fmt"
	"testing"
)

// An independent finite transition table checks all timelines of at most four
// records. It does not call Calculate to build fixtures or expected answers.
func TestJointSmallStateSpaceIndependentTable(t *testing.T) {
	transitions := map[string]map[string]string{"active": {"CHANGE": "active", "DEACTIVATE": "inactive"}, "inactive": {"REACTIVATE": "active"}}
	types := []string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"}
	var walk func([]string, int)
	walk = func(kinds []string, remaining int) {
		state, valid := "active", true
		items := []Item{}
		for i, kind := range kinds {
			next := transitions[state][kind]
			if next == "" {
				valid = false
			} else {
				state = next
			}
			it := jadd(fmt.Sprint(i), fmt.Sprintf("%02d", i+2), kind, "", "")
			if kind == "CREATE" {
				it.Reason = "NEW_ORGANIZATION"
			}
			if kind == "CHANGE" || kind == "REACTIVATE" {
				it.Changes = PatchFromFields(map[string]*string{"name": textValue(fmt.Sprintf("段%d", i))})
			}
			items = append(items, it)
		}
		if len(items) > 0 {
			c, e := Calculate(context.Background(), jointFixture("F1"), "f", "2025-01-01", items)
			if (e == nil) != valid {
				t.Fatalf("sequence %v expected accepted=%v got %v", kinds, valid, e)
			}
			if valid && (len(c.Records) != len(kinds)+1 || c.Records[len(c.Records)-1].State != state) {
				t.Fatal(kinds, c)
			}
		}
		if remaining > 0 {
			for _, kind := range types {
				next := append(append([]string{}, kinds...), kind)
				walk(next, remaining-1)
			}
		}
	}
	walk(nil, 3)
}
