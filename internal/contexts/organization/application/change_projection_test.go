package application

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/application/changelog"
	"encoding/json"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

func TestChangeProjectionPermissionAndEmptySemantics(t *testing.T) {
	before := domain.Record{ID: "r", Revision: "r1", Date: "2026-06-01", End: "2026-10-01", State: "active", Type: "CHANGE", Reason: "NAME_ADJUSTMENT", TypeLabel: "保存时类型", ReasonLabel: "保存时原因", Data: domain.BUData{Name: "甲"}}
	after := before
	after.Revision = "r2"
	after.Date = "2026-07-01"
	after.End = ""
	after.Data.Name = "乙"
	full := BuildChangeItem("r", "direct", "s1", "s2", &before, &after, []string{"name", "nameEn", "effectiveDate", "operationType", "operationReason"})
	if len(full.Diffs) != 3 || full.Diffs[0].Field != "name" || full.Diffs[1].Field != "effectiveDate" || full.Diffs[2].Field != "period" {
		t.Fatal("contexts must not become extra differences", full.Diffs)
	}
	if *full.BeforeContext.EndInclusive == nil || **full.BeforeContext.EndInclusive != "2026-09-30" || full.AfterContext.EndExclusive == nil || *full.AfterContext.EndExclusive != nil || full.AfterContext.NameEn == nil || *full.AfterContext.NameEn != nil {
		t.Fatal("authorized null and Go inclusive bound")
	}
	for _, date := range []string{"2026-06-01", "2026-07-01"} {
		if !ChangeMatches(org.Change{Items: []org.ChangeItem{full}}, &org.ColumnFilter{Field: "effectiveDate", Operator: "is", Value: date}, "") {
			t.Fatal("redate must match either side")
		}
	}
	trimmed := BuildChangeItem("r", "direct", "s1", "s2", &before, &after, []string{})
	encoded, _ := json.Marshal(trimmed)
	for _, forbidden := range []string{"甲", "乙", "effectiveDate", "endExclusive", "nameEn", "operationType", "operationReason", "保存时"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatal("unreadable field surfaced", forbidden)
		}
	}
	if len(trimmed.Diffs) != 0 {
		t.Fatal("hidden changes must retain generic operation without differences")
	}
	created := BuildChangeItem("r", "direct", "", "s1", nil, &before, []string{"name", "nameEn"})
	if created.BeforeRef != nil || created.BeforeContext != nil || len(created.Diffs) != 2 || created.Diffs[0].Before.Kind != "missing" {
		t.Fatal("creation absent optional English must not match nameEn", created)
	}
}

func TestChangeFilterDaysAndLiterals(t *testing.T) {
	for _, tc := range []struct {
		op            string
		first, second bool
	}{{"is", false, true}, {"before", true, false}, {"after", false, false}, {"onOrBefore", true, true}, {"onOrAfter", false, true}} {
		f := org.ColumnFilter{Field: "recordedAt", Operator: tc.op, Value: "2026-11-03"}
		if changelog.MatchDate("2026-11-02", f) != tc.first || changelog.MatchDate("2026-11-03", f) != tc.second {
			t.Fatal(tc.op)
		}
	}
	login := "A_%\\B"
	c := org.Change{Operator: org.OperatorSnapshot{SubjectID: "hidden-id", Login: &login, IdentityEvidence: "captured"}}
	for _, tc := range []struct {
		value string
		want  bool
	}{{"_%\\", true}, {"hidden-id", false}, {"a_%\\b", true}, {"a%b", false}} {
		q := org.ChangeQuery{Filter: &org.ColumnFilter{Field: "actor", Operator: "contains", Value: tc.value}}
		if err := NormalizeChangeQuery(&q); err != nil {
			t.Fatal(err)
		}
		if ChangeMatches(c, q.Filter, "") != tc.want {
			t.Fatal("actor must use literal captured display values", tc.value)
		}
	}
}
