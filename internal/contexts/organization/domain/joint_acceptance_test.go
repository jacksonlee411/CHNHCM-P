package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"testing"
)

func TestJointRemainingNegativeBoundaryParameters(t *testing.T) {
	cases := []struct {
		id, code string
		items    []Item
	}{
		{"TL117-two-updates", "M02_REVISION_ITEM_CONFLICT", []Item{jupdate("a", "c1", "07", "", "", ""), jupdate("b", "c1", "08", "", "", "")}},
		{"TL146-first-stop", "M02_FIRST_RECORD_PROTECTED", []Item{jupdate("a", "f", "", "DEACTIVATE", "", "")}},
		{"TL146-first-resume", "M02_FIRST_RECORD_PROTECTED", []Item{jupdate("a", "f", "", "REACTIVATE", "", "")}},
		{"TL146-deleted-classification", "M02_RECORD_NOT_FOUND", []Item{jupdate("a", "deleted", "", "REACTIVATE", "", "")}},
	}
	stop := jupdate("a", "c2", "", "DEACTIVATE", "", "")
	stop.Changes = PatchFromFields(map[string]*string{"nameEn": nil})
	cases = append(cases, struct {
		id, code string
		items    []Item
	}{"TL147-explicit-null-English", "M02_FORBIDDEN_INPUT", []Item{stop}})
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			permutations(tc.items, func(items []Item) {
				_, e := Calculate(context.Background(), jointFixture("F2"), "f", "2025-01-01", items)
				var f *foundation.Fault
				if !errors.As(e, &f) || f.Code != tc.code {
					t.Fatalf("want %s got %v", tc.code, e)
				}
			})
		})
	}
}

func TestJointMultipleFieldErrorsUseCanonicalItemPriority(t *testing.T) {
	for _, nameFirst := range []bool{true, false} {
		first := jupdate("a", "f", "", "", "", "")
		second := jupdate("b", "c1", "", "", "", "")
		wantField := "changes.name"
		if nameFirst {
			first.Changes = PatchFromFields(map[string]*string{"name": textValue("")})
			second.Date = "invalid"
		} else {
			first.Date = "invalid"
			second.Changes = PatchFromFields(map[string]*string{"name": textValue("")})
			wantField = "newEffectiveDate"
		}
		permutations([]Item{second, first}, func(items []Item) {
			_, e := Calculate(context.Background(), jointFixture("F2"), "f", "2025-01-01", items)
			var fault *foundation.Fault
			var item *ItemFault
			if !errors.As(e, &fault) || !errors.As(e, &item) || item.ItemID != "a" || fault.Code != "M02_INVALID_INPUT" || fault.Field != wantField {
				t.Fatal("TL216 noncanonical diagnosis", e)
			}
		})
	}
}
