package location

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"strings"
	"testing"
)

func TestSingleAddressQuery(t *testing.T) {
	for _, tc := range []struct {
		field, value, sort string
		valid              bool
	}{
		{"addressLine", strings.Repeat("址", 1001), "addressLine", true},
		{"addressLine", strings.Repeat("址", 1002), "", false},
		{"addressExtra", "楼层", "", false},
		{"addressLine", "一号", "addressExtra_desc", false},
	} {
		q := org.LocationQuery{ListInput: org.ListInput{Sort: tc.sort, Filter: &org.ColumnFilter{Field: tc.field, Operator: "equals", Value: tc.value}}}
		if e := normalizeList(&q); (e == nil) != tc.valid {
			t.Fatalf("field=%s length=%d sort=%s: %v", tc.field, len([]rune(tc.value)), tc.sort, e)
		}
	}
}
