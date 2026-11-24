package application

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/csv"
	"strings"
	"testing"
)

func TestSelectionCSVTextAndCancellation(t *testing.T) {
	for _, text := range []string{"=SUM(A1:A2)", " +1", "-2", "@x", "\tx", "\rx", "\nx"} {
		if !strings.HasPrefix(csvText(text), "'") {
			t.Fatalf("formula not neutralized: %q", text)
		}
	}
	name := "甲,\"乙\"\n丙"
	data, err := selectionCSV(context.Background(), []org.Summary{{Code: "=1", Name: &name, Status: "not_effective", EffectiveDate: "2027-01-01"}, {Status: "active"}})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xef\xbb\xbf"))).ReadAll()
	if err != nil || len(parsed) != 3 || parsed[1][0] != "'=1" || parsed[1][1] != strings.ReplaceAll(name, "\n", "\n") || parsed[1][2] != "未生效" || parsed[2][0] != "" {
		t.Fatalf("CSV roundtrip: %v %v", parsed, err)
	}
	if strings.Join(parsed[0], ",") != "编码,名称,状态,生效日期" || parsed[2][2] != "启用" {
		t.Fatalf("CSV status label or meaning changed: %v", parsed)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if data, err = selectionCSV(ctx, []org.Summary{{Status: "active"}}); err == nil || data != nil {
		t.Fatal("cancel produced content")
	}
	big := strings.Repeat("x", 1<<20)
	rows := make([]org.Summary, 34)
	for i := range rows {
		rows[i] = org.Summary{Name: &big}
	}
	if data, err = selectionCSV(context.Background(), rows); err == nil || data != nil {
		t.Fatal("resource failure produced content")
	}
}
func TestExplicitAndExcludedSelection(t *testing.T) {
	a, b := foundation.NewID(), foundation.NewID()
	list := org.List{Items: []org.Summary{{ID: a}, {ID: b}}}
	ids := []string{b}
	rows, err := selectRows(list, org.Selection{Mode: "allMatching", ExcludedIDs: &ids})
	if err != nil || len(rows) != 1 || rows[0].ID != a {
		t.Fatal(rows, err)
	}
	for _, choice := range []org.Selection{{Mode: "explicit"}, {Mode: "explicit", IDs: &ids, ExcludedIDs: &ids}, {Mode: "invalid", IDs: &ids}} {
		if _, err = selectRows(list, choice); err == nil {
			t.Fatal("invalid union accepted")
		}
	}
	duplicate := []string{a, a}
	if _, err = selectRows(list, org.Selection{Mode: "explicit", IDs: &duplicate}); err == nil {
		t.Fatal("duplicate accepted")
	}
}
