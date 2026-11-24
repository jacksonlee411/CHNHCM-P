package application

import (
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

func TestRecordLastApplicableDay(t *testing.T) {
	for _, tc := range []struct{ end, last string }{
		{"2026-03-24", "2026-03-23"},
		{"2026-03-01", "2026-02-28"},
		{"2028-03-01", "2028-02-29"},
		{"2000-03-01", "2000-02-29"},
		{"2100-03-01", "2100-02-28"},
		{"2027-01-01", "2026-12-31"},
		{"0001-01-02", "0001-01-01"},
		{"9999-12-31", "9999-12-30"},
		{"", ""},
	} {
		t.Run(tc.end, func(t *testing.T) {
			record := PublicRecord(domain.Record{Date: "0001-01-01", End: tc.end}, []string{"effectiveDate"})
			if tc.end == "" {
				if record.EndExclusive != nil || record.EndInclusive != nil {
					t.Fatal("open period acquired an end date")
				}
				return
			}
			if record.EndExclusive == nil || *record.EndExclusive != tc.end || record.EndInclusive == nil || *record.EndInclusive != tc.last {
				t.Fatalf("incorrect exclusive or inclusive end: %+v", record)
			}
		})
	}
}

func TestRecordPeriodRespectsDateReadPermission(t *testing.T) {
	r := domain.Record{Date: "2026-03-13", End: "2026-03-24"}
	assertHidden := func(r org.Record) {
		t.Helper()
		if r.EffectiveDate != "" || r.EndExclusive != nil || r.EndInclusive != nil {
			t.Fatal("date permission leaked a period")
		}
	}
	assertHidden(PublicRecord(r, []string{"name"}))
	before, after := PublicRecord(r, []string{"effectiveDate"}), PublicRecord(r, []string{"effectiveDate"})
	result := org.Result{Items: []org.ItemResult{{BeforeRecord: &before, AfterRecord: &after}}}
	trimJointResult(&result, []string{"name"})
	assertHidden(*result.Items[0].BeforeRecord)
	assertHidden(*result.Items[0].AfterRecord)
}
