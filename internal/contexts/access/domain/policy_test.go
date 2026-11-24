package domain

import "testing"

// TL-169: the three delete grants are independent capabilities, with expected
// decisions written out independently of M02's operation-to-action mapping.
func TestDeleteCategoryGrantMatrix(t *testing.T) {
	for _, tc := range []struct {
		grant, request string
		allowed        bool
	}{
		{"bu.record.delete", "bu.record.delete", true},
		{"bu.record.delete", "bu.deactivation.delete", false},
		{"bu.record.delete", "bu.reactivation.delete", false},
		{"bu.deactivation.delete", "bu.record.delete", false},
		{"bu.deactivation.delete", "bu.deactivation.delete", true},
		{"bu.deactivation.delete", "bu.reactivation.delete", false},
		{"bu.reactivation.delete", "bu.record.delete", false},
		{"bu.reactivation.delete", "bu.deactivation.delete", false},
		{"bu.reactivation.delete", "bu.reactivation.delete", true},
	} {
		t.Run(tc.grant+"/"+tc.request, func(t *testing.T) {
			if got := Matches("bu", "target", tc.grant, "enterprise", "target", tc.request); got != tc.allowed {
				t.Fatalf("allowed=%v want=%v", got, tc.allowed)
			}
		})
	}
}
