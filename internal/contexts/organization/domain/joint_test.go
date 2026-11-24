package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// These fixtures and expected strings are handwritten from 023; Calculate is
// never used to prepare a baseline or to produce an expected answer.
func jointFixture(which string) []Record {
	rows := map[string][]string{
		"F1":  {"f|01|甲|Alpha|CREATE"},
		"F2":  {"f|01|甲|Alpha|CREATE", "c1|06|乙|Beta|CHANGE", "c2|10|丙|Gamma|CHANGE"},
		"F3":  {"f|01|甲|Alpha|CREATE", "s1|04|甲|Alpha|DEACTIVATE", "r1|08|乙|Beta|REACTIVATE", "c1|10|丙|Gamma|CHANGE", "s2|12|丙|Gamma|DEACTIVATE"},
		"F4":  {"f|01|甲|Alpha|CREATE", "s1|04|甲|Alpha|DEACTIVATE"},
		"F5":  {"f|01|甲|Alpha|CREATE", "s1|04|甲|Alpha|DEACTIVATE", "r1|05|乙|Beta|REACTIVATE"},
		"F7":  {"f|06|乙|Beta|CREATE", "s1|07|乙|Beta|DEACTIVATE", "r1|08|丙|Gamma|REACTIVATE", "c1|10|终|Zulu|CHANGE"},
		"F8":  {"f|01|乙|Beta|CREATE", "c1|03|乙|Beta|CHANGE", "s1|05|乙|Beta|DEACTIVATE", "r1|07|乙|Beta|REACTIVATE"},
		"F9":  {"f|01|甲|Alpha|CREATE", "c1|03|乙|Beta|CHANGE", "c2|05|丙|Gamma|CHANGE", "c3|07|终|Zulu|CHANGE"},
		"F10": {"f|01|甲|Alpha|CREATE", "s1|04|甲|Alpha|DEACTIVATE", "r1|05|乙|Beta|REACTIVATE", "s2|08|乙|Beta|DEACTIVATE", "r2|09|丙|Gamma|REACTIVATE", "c1|11|终|Zulu|CHANGE"},
	}[which]
	out := []Record{}
	for _, row := range rows {
		v := strings.Split(row, "|")
		reason := map[string]string{"CREATE": "NEW_ORGANIZATION", "CHANGE": "NAME_ADJUSTMENT", "DEACTIVATE": "BUSINESS_CLOSURE", "REACTIVATE": "BUSINESS_RESUMPTION"}[v[4]]
		out = append(out, Record{ID: v[0], Revision: "rev-" + v[0], RevisionNo: 1, Date: "2026-" + v[1] + "-01", Type: v[4], Reason: reason, TypeLabel: OperationLabel(v[4]), ReasonLabel: OperationReasons(v[4])[reason], State: "active", Data: BUData{Name: v[2], NameEn: textValue(v[3])}})
	}
	for i := range out {
		if i+1 < len(out) {
			out[i].End = out[i+1].Date
		}
		if out[i].Type == "DEACTIVATE" {
			out[i].State = "inactive"
			out[i].InheritedRecord = out[i-1].ID
			out[i].InheritedRevision = out[i-1].Revision
		}
	}
	return out
}
func jadd(id, month, kind, name, en string) Item {
	it := Item{ID: id, Action: "ADD", Date: "2026-" + month + "-01", Type: kind, Reason: map[string]string{"CHANGE": "NAME_ADJUSTMENT", "DEACTIVATE": "BUSINESS_CLOSURE", "REACTIVATE": "BUSINESS_RESUMPTION"}[kind]}
	if name != "" {
		it.Changes = PatchFromFields(map[string]*string{"name": textValue(name)})
	}
	if en != "" {
		if it.Changes == nil {
			it.Changes = PatchFromFields(map[string]*string{})
		}
		it.Changes.NameEn = BUField{Present: true, Value: textValue(en)}
	}
	return it
}
func jupdate(id, target, month, kind, name, en string) Item {
	it := Item{ID: id, Action: "UPDATE", RecordID: target, ExpectedRevision: "rev-" + target, Type: kind}
	if month != "" {
		it.Date = "2026-" + month + "-01"
	}
	if kind != "" {
		it.Reason = map[string]string{"CREATE": "NEW_ORGANIZATION", "CHANGE": "NAME_ADJUSTMENT", "DEACTIVATE": "BUSINESS_CLOSURE", "REACTIVATE": "BUSINESS_RESUMPTION"}[kind]
	}
	if name != "" {
		it.Changes = PatchFromFields(map[string]*string{"name": textValue(name)})
	}
	if en != "" {
		if it.Changes == nil {
			it.Changes = PatchFromFields(map[string]*string{})
		}
		it.Changes.NameEn = BUField{Present: true, Value: textValue(en)}
	}
	return it
}
func jdel(id, target string) Item {
	return Item{ID: id, Action: "DELETE", RecordID: target, ExpectedRevision: "rev-" + target}
}
func facts(rows []Record) string {
	out := []string{}
	for _, r := range rows {
		en := ""
		if r.Data.NameEn != nil {
			en = *r.Data.NameEn
		}
		out = append(out, r.Date[5:7]+":"+r.Data.Name+"/"+en+":"+r.Type)
	}
	return strings.Join(out, ",")
}
func TestJointIndependentMatrix(t *testing.T) {
	cases := []struct {
		id, fixture, want string
		items             []Item
	}{
		{"TL108", "F2", "01:甲/Alpha:CREATE,03:甲/Alpha:DEACTIVATE,04:甲/Alpha:REACTIVATE,06:乙/Beta:CHANGE,10:丙/Gamma:CHANGE", []Item{jadd("a", "03", "DEACTIVATE", "", ""), jadd("b", "04", "REACTIVATE", "", "")}},
		{"TL109", "F3", "01:甲/Alpha:CREATE,10:丙/Gamma:CHANGE,12:丙/Gamma:DEACTIVATE", []Item{jdel("a", "s1"), jdel("b", "r1")}},
		{"TL110", "F2", "01:甲/Alpha:CREATE,06:丙/Gamma:CHANGE,10:乙/Beta:CHANGE", []Item{jupdate("a", "c1", "10", "", "", ""), jupdate("b", "c2", "06", "", "", "")}},
		{"TL111", "F9", "01:甲/Alpha:CREATE,03:终/Zulu:CHANGE,05:乙/Beta:CHANGE,07:丙/Gamma:CHANGE", []Item{jupdate("a", "c1", "05", "", "", ""), jupdate("b", "c2", "07", "", "", ""), jupdate("c", "c3", "03", "", "", "")}},
		{"TL112", "F9", "01:甲/Alpha:CREATE,03:乙2/Alpha:CHANGE,06:丙2/Gamma:CHANGE,07:终/Zulu:CHANGE", []Item{jdel("a", "c1"), jadd("b", "03", "CHANGE", "乙2", ""), jupdate("c", "c2", "06", "", "丙2", "")}},
		{"TL114", "F2", "01:甲/Alpha:CREATE,06:乙/Beta:CHANGE,11:新/Gamma:CHANGE", []Item{jupdate("a", "c2", "11", "", "新", "")}},
		{"TL122", "F1", "01:新/Alpha:CREATE,03:新/Beta:CHANGE", []Item{jadd("a", "03", "CHANGE", "", "Beta"), jupdate("b", "f", "", "", "新", "")}},
		{"TL123", "F1", "01:乙/Alpha:CREATE,05:丙/Alpha:CHANGE", []Item{jupdate("a", "f", "", "", "乙", ""), jadd("b", "03", "CHANGE", "乙", ""), jadd("c", "05", "CHANGE", "丙", "")}},
		{"TL124", "F2", "01:甲/Alpha:CREATE,03:新/Alpha:CHANGE,06:乙/Beta:CHANGE,10:丙/Gamma:CHANGE", []Item{jupdate("a", "f", "", "", "甲", ""), jadd("b", "03", "CHANGE", "新", "")}},
		{"TL125", "F2", "01:甲/Alpha:CREATE,06:乙/Beta:CHANGE,10:丙/Gamma:CHANGE", []Item{jupdate("a", "c1", "", "", "乙", ""), jupdate("b", "c2", "10", "", "", ""), jadd("c", "03", "CHANGE", "甲", "")}},
		{"TL127", "F10", "01:甲/Alpha:CREATE", []Item{jdel("a", "s1"), jdel("b", "r1"), jdel("c", "s2"), jdel("d", "r2"), jdel("e", "c1")}},
		{"TL129", "F1", "01:早/Alpha:CREATE,06:乙/Beta:CHANGE", []Item{jupdate("a", "f", "01", "", "早", "Alpha"), jadd("b", "06", "CHANGE", "乙", "Beta")}},
		{"TL130", "F7", "01:甲/Alpha:CREATE,06:乙/Beta:CHANGE,07:乙/Beta:DEACTIVATE,08:丙/Gamma:REACTIVATE,10:终/Zulu:CHANGE", []Item{jupdate("a", "f", "01", "", "甲", "Alpha"), jadd("b", "06", "CHANGE", "乙", "Beta")}},
		{"TL131", "F7", "01:甲/Alpha:CREATE,07:甲/Alpha:DEACTIVATE,08:丙/Gamma:REACTIVATE,10:终/Zulu:CHANGE", []Item{jupdate("a", "f", "01", "", "甲", "Alpha")}},
		{"TL136", "F1", "01:甲/Alpha:CREATE", []Item{jupdate("a", "f", "", "CREATE", "", "")}},
		{"TL137", "F2", "01:甲/Alpha:CREATE,06:乙/Beta:CHANGE,10:乙/Beta:DEACTIVATE", []Item{jupdate("a", "c2", "", "DEACTIVATE", "", "")}},
		{"TL138", "F4", "01:甲/Alpha:CREATE,04:甲/Alpha:CHANGE", []Item{jupdate("a", "s1", "", "CHANGE", "", "")}},
		{"TL139", "F4", "01:甲/Alpha:CREATE,03:甲/Alpha:DEACTIVATE,04:甲/Alpha:REACTIVATE", []Item{jupdate("a", "s1", "", "REACTIVATE", "", ""), jadd("b", "03", "DEACTIVATE", "", "")}},
		{"TL140", "F5", "01:甲/Alpha:CREATE,05:乙/Beta:CHANGE", []Item{jupdate("a", "r1", "", "CHANGE", "", ""), jdel("b", "s1")}},
		{"TL141", "F5", "01:甲/Alpha:CREATE,05:甲/Alpha:DEACTIVATE", []Item{jupdate("a", "r1", "", "DEACTIVATE", "", ""), jdel("b", "s1")}},
		{"TL142", "F2", "01:甲/Alpha:CREATE,06:乙/Beta:CHANGE,09:乙/Beta:DEACTIVATE,10:丙/Gamma:REACTIVATE", []Item{jupdate("a", "c2", "", "REACTIVATE", "", ""), jadd("b", "09", "DEACTIVATE", "", "")}},
		{"TL143", "F5", "01:甲/Alpha:CREATE,04:甲/Alpha:DEACTIVATE,05:甲/Alpha:REACTIVATE", []Item{jupdate("a", "s1", "05", "REACTIVATE", "", ""), jupdate("b", "r1", "04", "DEACTIVATE", "", "")}},
		{"TL144", "F5", "01:甲/West:CREATE,03:甲2/Alpha:CHANGE,06:乙2/Beta:CHANGE", []Item{jupdate("a", "f", "", "", "", "West"), jupdate("b", "s1", "03", "CHANGE", "甲2", ""), jupdate("c", "r1", "06", "CHANGE", "乙2", "")}},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			base := jointFixture(tc.fixture)
			if tc.id == "TL129" {
				base = jointFixture("F7")[:1]
				base[0].End = ""
			}
			original := slices.Clone(base)
			var first *Candidate
			permutations(tc.items, func(items []Item) {
				c, e := Calculate(context.Background(), base, "f", "2025-01-01", items)
				if e != nil {
					t.Fatal(e)
				}
				if facts(c.Records) != tc.want {
					t.Fatalf("want %s got %s", tc.want, facts(c.Records))
				}
				if first == nil {
					first = &c
				} else if !reflect.DeepEqual(*first, c) {
					t.Fatal("item order changed result")
				}
				if !reflect.DeepEqual(base, original) {
					t.Fatal("baseline mutated")
				}
			})
		})
	}
}
func permutations(items []Item, fn func([]Item)) {
	copy := slices.Clone(items)
	var walk func(int)
	walk = func(n int) {
		if n == len(copy) {
			fn(copy)
			return
		}
		for i := n; i < len(copy); i++ {
			copy[n], copy[i] = copy[i], copy[n]
			walk(n + 1)
			copy[n], copy[i] = copy[i], copy[n]
		}
	}
	walk(0)
}
func TestJointRejectMatrixAndStableItemDiagnostics(t *testing.T) {
	cases := []struct {
		id, fixture, code string
		items             []Item
	}{
		{"TL113", "F9", "M02_DATE_CONFLICT", []Item{jadd("a", "03", "CHANGE", "乙2", ""), jupdate("b", "c2", "06", "", "丙2", "")}},
		{"TL115", "F5", "M02_TIMELINE_INVALID", []Item{jupdate("a", "s1", "05", "", "", ""), jupdate("b", "r1", "04", "", "", "")}},
		{"TL116-order", "F2", "M02_FIRST_RECORD_PROTECTED", []Item{jupdate("a", "f", "11", "", "", "")}},
		{"TL116-create", "F1", "M02_INVALID_INPUT", []Item{jadd("a", "03", "CREATE", "新", "")}},
		{"TL116-delete", "F2", "M02_FIRST_RECORD_PROTECTED", []Item{jdel("a", "f")}},
		{"TL117-item", "F2", "M02_REVISION_ITEM_CONFLICT", []Item{jupdate("a", "c1", "07", "", "", ""), jdel("a", "c2")}},
		{"TL117-target", "F2", "M02_REVISION_ITEM_CONFLICT", []Item{jupdate("a", "c1", "07", "", "", ""), jdel("b", "c1")}},
		{"TL121", "F1", "M02_DATE_CONFLICT", []Item{jadd("a", "03", "CHANGE", "甲", ""), jadd("b", "03", "CHANGE", "终", "")}},
		{"TL134", "F7", "M02_INVALID_INPUT", []Item{jupdate("a", "f", "01", "", "", "")}},
		{"TL145", "F5", "M02_TIMELINE_INVALID", []Item{jupdate("a", "s1", "", "CHANGE", "", "")}},
		{"TL146-first", "F2", "M02_FIRST_RECORD_PROTECTED", []Item{jupdate("a", "f", "", "CHANGE", "", "")}},
		{"TL146-other", "F2", "M02_FIRST_RECORD_PROTECTED", []Item{jupdate("a", "c1", "", "CREATE", "", "")}},
		{"TL147-value", "F2", "M02_FORBIDDEN_INPUT", []Item{jupdate("a", "c2", "", "DEACTIVATE", "丙", "")}},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			start := "2025-01-01"
			if tc.id == "TL134" {
				start = "2026-02-01"
			}
			permutations(tc.items, func(items []Item) {
				_, e := Calculate(context.Background(), jointFixture(tc.fixture), "f", start, items)
				var f *foundation.Fault
				if !errors.As(e, &f) || f.Code != tc.code {
					t.Fatal(tc.code, e)
				}
			})
		})
	}
}
func TestJointNoChangeSnapshotsImpactAndClassificationEvidence(t *testing.T) {
	base := jointFixture("F2")
	same := jupdate("a", "f", "", "", "甲", "")
	add := jadd("b", "03", "CHANGE", "新", "")
	c, e := Calculate(context.Background(), base, "f", "2025-01-01", []Item{same, add})
	if e != nil {
		t.Fatal(e)
	}
	if len(DirectFacets(same, base, c.Records)) != 0 || c.Records[0].End != "2026-03-01" {
		t.Fatal("no-change input lost derived period")
	}
	base = jointFixture("F4")
	base[1].Reason = "RETIRED_REASON"
	base[1].ReasonLabel = "历史原因"
	it := jupdate("a", "s1", "", "DEACTIVATE", "", "")
	it.Reason = "RETIRED_REASON"
	c, e = Calculate(context.Background(), base, "f", "2025-01-01", []Item{it})
	if e != nil || !c.NoChange || c.Records[1].ReasonLabel != "历史原因" {
		t.Fatal("same classification refreshed or revalidated", c, e)
	}
	// Disjoint field changes must leave the unchanged interval out of the data set.
	base = jointFixture("F9")
	c, e = Calculate(context.Background(), base, "f", "2025-01-01", []Item{jupdate("a", "f", "", "", "甲2", ""), jupdate("b", "c2", "", "", "丙2", "")})
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(c.Impact.DataAffectedRanges, []Range{{Start: "2026-01-01", End: textValue("2026-03-01")}, {Start: "2026-05-01", End: textValue("2026-07-01")}}) {
		t.Fatal(c.Impact)
	}
}
