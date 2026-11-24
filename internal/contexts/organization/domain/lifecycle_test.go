package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func lifecycleBase() []Record { return baseline()[:1] }
func lifeItem(id, date, kind, reason string, changes map[string]*string) Item {
	return Item{ID: id, Action: "ADD", Date: date, Type: kind, Reason: reason, Changes: PatchFromFields(changes)}
}
func TestLifecycleThreeDatesAndIndependentDefaults(t *testing.T) {
	for _, v := range []struct{ label, stop, resume, stopToday, resumeToday string }{
		{"past", "2026-03-01", "2026-04-01", "inactive", "active"},
		{"today", "2026-06-15", "2026-06-16", "inactive", "inactive"},
		{"resume_today", "2026-06-14", "2026-06-15", "inactive", "active"},
		{"future", "2026-07-01", "2026-08-01", "active", "active"},
	} {
		t.Run(v.label, func(t *testing.T) {
			base := lifecycleBase()
			base[0].End = ""
			original := append([]Record{}, base...)
			stopped, e := Calculate(context.Background(), base, "first", "2020-01-01", []Item{lifeItem("stop", v.stop, "DEACTIVATE", "BUSINESS_CLOSURE", nil)})
			if e != nil {
				t.Fatal(e)
			}
			// Closing the first period needs a new revision; bind the exact ID on save.
			s := stopped.Records[1]
			if s.State != "inactive" || s.Data.Name != "甲" || *s.Data.NameEn != "Alpha" || s.InheritedRecord != "first" || s.InheritedRevision != "" || stopped.Records[0].End != v.stop || stopped.NoChange || StatusAt(stopped.Records, "2026-06-15") != v.stopToday {
				t.Fatal(stopped)
			}
			resumed, e := Calculate(context.Background(), stopped.Records, "first", "2020-01-01", []Item{lifeItem("resume", v.resume, "REACTIVATE", "BUSINESS_RESUMPTION", nil)})
			if e != nil {
				t.Fatal(e)
			}
			r := resumed.Records[2]
			if resumed.NoChange || r.Data.Name != "甲" || *r.Data.NameEn != "Alpha" || r.InheritedRecord != "" || r.State != "active" || resumed.Records[1].End != v.resume || StatusAt(resumed.Records, "2026-06-15") != v.resumeToday {
				t.Fatal(resumed)
			}
			fixed, e := Calculate(context.Background(), resumed.Records, "first", "2020-01-01", []Item{{ID: "fix", Action: "CORRECT", RecordID: "first", ExpectedRevision: "r1", Changes: PatchFromFields(map[string]*string{"name": textValue("甲2")})}})
			if e != nil || fixed.Records[1].Data.Name != "甲2" || fixed.Records[2].Data.Name != "甲" || !reflect.DeepEqual(base, original) {
				t.Fatal("independent resume/old source", fixed, e)
			}
		})
	}
}
func TestLifecycleMultiRoundAndExplicitFields(t *testing.T) {
	current := lifecycleBase()
	current[0].End = ""
	for _, it := range []Item{
		lifeItem("s1", "2026-04-01", "DEACTIVATE", "ORG_RESTRUCTURING", nil),
		lifeItem("r1", "2026-05-01", "REACTIVATE", "BUSINESS_RESUMPTION", map[string]*string{"name": textValue("乙"), "nameEn": textValue("Beta")}),
		lifeItem("s2", "2026-08-01", "DEACTIVATE", "BUSINESS_CLOSURE", nil),
		lifeItem("r2", "2026-09-01", "REACTIVATE", "BUSINESS_RESUMPTION", map[string]*string{"name": textValue("丙"), "nameEn": nil}),
		lifeItem("c", "2026-11-01", "CHANGE", "NAME_ADJUSTMENT", map[string]*string{"name": textValue("丁")}),
	} {
		c, e := Calculate(context.Background(), current, "first", "2020-01-01", []Item{it})
		if e != nil {
			t.Fatal(it, e)
		}
		current = c.Records
	}
	want := []struct{ date, end, state, name, en string }{{"2026-01-01", "2026-04-01", "active", "甲", "Alpha"}, {"2026-04-01", "2026-05-01", "inactive", "甲", "Alpha"}, {"2026-05-01", "2026-08-01", "active", "乙", "Beta"}, {"2026-08-01", "2026-09-01", "inactive", "乙", "Beta"}, {"2026-09-01", "2026-11-01", "active", "丙", ""}, {"2026-11-01", "", "active", "丁", ""}}
	for i, w := range want {
		r := current[i]
		en := ""
		if r.Data.NameEn != nil {
			en = *r.Data.NameEn
		}
		if r.Date != w.date || r.End != w.end || r.State != w.state || r.Data.Name != w.name || en != w.en {
			t.Fatal(i, r, w)
		}
		day, _ := time.Parse("2006-01-02", w.date)
		previousState := "not_effective"
		if i > 0 {
			previousState = want[i-1].state
		}
		for _, at := range []struct {
			offset int
			state  string
		}{{-1, previousState}, {0, w.state}, {1, w.state}} {
			businessDay := day.AddDate(0, 0, at.offset).Format("2006-01-02")
			if got := StatusAt(current, businessDay); got != at.state {
				t.Fatal("multi-round date boundary", businessDay, got, at.state)
			}
		}
	}
	c, e := Calculate(context.Background(), current, "first", "2020-01-01", []Item{{ID: "fix", Action: "CORRECT", RecordID: "r1", ExpectedRevision: "", Changes: PatchFromFields(map[string]*string{"name": textValue("乙2")})}})
	if e != nil || c.Records[3].Data.Name != "乙2" || c.Records[4].Data.Name != "丙" {
		t.Fatal(c, e)
	}
}
func TestLifecycleRejectionsAndFullSuccessorScan(t *testing.T) {
	first := lifecycleBase()
	first[0].End = ""
	stopped, _ := Calculate(context.Background(), first, "first", "2020-01-01", []Item{lifeItem("s", "2026-04-01", "DEACTIVATE", "BUSINESS_CLOSURE", nil)})
	cycle, _ := Calculate(context.Background(), stopped.Records, "first", "2020-01-01", []Item{lifeItem("r", "2026-08-01", "REACTIVATE", "BUSINESS_RESUMPTION", nil)})
	for _, v := range []struct {
		base       []Record
		it         Item
		dependency string
	}{
		{first, lifeItem("x", "2026-03-01", "REACTIVATE", "BUSINESS_RESUMPTION", nil), "2026-03-01"},
		{stopped.Records, lifeItem("x", "2026-05-01", "DEACTIVATE", "BUSINESS_CLOSURE", nil), "2026-05-01"},
		{stopped.Records, lifeItem("x", "2026-05-01", "CHANGE", "NAME_ADJUSTMENT", map[string]*string{"name": textValue("新")}), "2026-05-01"},
		{baseline(), lifeItem("x", "2026-03-01", "DEACTIVATE", "BUSINESS_CLOSURE", nil), "2026-06-01"},
		{cycle.Records, lifeItem("x", "2026-05-01", "REACTIVATE", "BUSINESS_RESUMPTION", nil), "2026-08-01"},
		{first, lifeItem("x", "2026-01-01", "DEACTIVATE", "BUSINESS_CLOSURE", nil), ""},
		{first, lifeItem("x", "2025-12-31", "DEACTIVATE", "BUSINESS_CLOSURE", nil), ""},
		{first, lifeItem("x", "2025-12-31", "REACTIVATE", "BUSINESS_RESUMPTION", nil), ""},
		{stopped.Records, lifeItem("x", "2026-04-01", "REACTIVATE", "BUSINESS_RESUMPTION", nil), ""},
		{cycle.Records, lifeItem("x", "2026-08-01", "DEACTIVATE", "BUSINESS_CLOSURE", nil), ""},
		{first, lifeItem("x", "2019-12-31", "DEACTIVATE", "BUSINESS_CLOSURE", nil), ""},
		{first, lifeItem("x", "2026-02-30", "DEACTIVATE", "BUSINESS_CLOSURE", nil), ""},
		{first, lifeItem("x", "2026-04-01", "DEACTIVATE", "BUSINESS_RESUMPTION", nil), ""},
		{first, lifeItem("x", "2026-04-01", "DEACTIVATE", "BUSINESS_CLOSURE", map[string]*string{}), ""},
		{first, lifeItem("x", "2026-04-01", "DEACTIVATE", "BUSINESS_CLOSURE", map[string]*string{"name": textValue("新")}), ""},
		{stopped.Records, lifeItem("x", "2026-05-01", "REACTIVATE", "BUSINESS_RESUMPTION", map[string]*string{}), ""},
		{stopped.Records, lifeItem("x", "2026-05-01", "REACTIVATE", "BUSINESS_RESUMPTION", map[string]*string{"name": nil}), ""},
		{stopped.Records, lifeItem("x", "2026-05-01", "REACTIVATE", "BUSINESS_RESUMPTION", map[string]*string{"name": textValue("😀")}), ""},
	} {
		_, e := Calculate(context.Background(), v.base, "first", "2020-01-01", []Item{v.it})
		var fault *foundation.Fault
		if e == nil || !errors.As(e, &fault) || v.dependency != "" && !strings.Contains(fault.Message, v.dependency) {
			t.Fatal(v.it, e)
		}
	}
}

func TestLifecycleReasonDirectoryAndSnapshots(t *testing.T) {
	active := lifecycleBase()
	active[0].End = ""
	inactive := []Record{active[0], {ID: "stop", Date: "2026-02-01", Type: "DEACTIVATE", State: "inactive", Reason: "BUSINESS_CLOSURE", Data: BUData{Name: "甲", NameEn: textValue("Alpha")}}}
	for _, v := range []struct {
		base                             []Record
		kind, reason, label, reasonLabel string
		allowed                          bool
	}{
		{active, "DEACTIVATE", "ORG_RESTRUCTURING", "停用", "机构撤并", true},
		{active, "DEACTIVATE", "BUSINESS_CLOSURE", "停用", "业务终止", true},
		{inactive, "REACTIVATE", "BUSINESS_RESUMPTION", "重新启用", "业务恢复", true},
		{active, "DEACTIVATE", "BUSINESS_RESUMPTION", "", "", false},
		{inactive, "REACTIVATE", "BUSINESS_CLOSURE", "", "", false},
		{active, "DEACTIVATE", "", "", "", false},
		{inactive, "REACTIVATE", "", "", "", false},
	} {
		candidate, err := Calculate(context.Background(), v.base, "first", "2020-01-01", []Item{lifeItem("new", "2026-03-01", v.kind, v.reason, nil)})
		if (err == nil) != v.allowed {
			t.Fatal(v.kind, v.reason, err)
		}
		if v.allowed {
			r := candidate.Records[len(candidate.Records)-1]
			if r.TypeLabel != v.label || r.ReasonLabel != v.reasonLabel || r.Reason != v.reason {
				t.Fatal("classification snapshot", r)
			}
		}
	}
}

func TestLifecycleStateMatrix(t *testing.T) {
	active := lifecycleBase()
	active[0].End = ""
	stopped, e := Calculate(context.Background(), active, "first", "2020-01-01", []Item{lifeItem("s", "2026-02-01", "DEACTIVATE", "BUSINESS_CLOSURE", nil)})
	if e != nil {
		t.Fatal(e)
	}
	future := lifecycleBase()
	future[0].Date = "2026-09-01"
	future[0].End = ""
	for _, v := range []struct {
		state   string
		base    []Record
		allowed map[string]bool
	}{
		{"no_fact", future, map[string]bool{}},
		{"active", active, map[string]bool{"CHANGE": true, "DEACTIVATE": true}},
		{"inactive", stopped.Records, map[string]bool{"REACTIVATE": true}},
	} {
		for _, kind := range []string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"} {
			reason := map[string]string{"CREATE": "NEW_ORGANIZATION", "CHANGE": "NAME_ADJUSTMENT", "DEACTIVATE": "BUSINESS_CLOSURE", "REACTIVATE": "BUSINESS_RESUMPTION"}[kind]
			var changes map[string]*string
			if kind == "CHANGE" {
				changes = map[string]*string{"name": textValue("新")}
			}
			_, e := Calculate(context.Background(), v.base, "first", "2020-01-01", []Item{lifeItem("new", "2026-03-01", kind, reason, changes)})
			if (e == nil) != v.allowed[kind] {
				t.Fatal(v.state, kind, e)
			}
		}
	}
	// CREATE is permitted for a new identity, never a second first record.
	if _, _, e := CreateTimeline(context.Background(), Create{Code: "NEW", Name: "甲", EffectiveDate: "2026-03-01", OperationType: "CREATE", OperationReason: "NEW_ORGANIZATION"}, "2020-01-01"); e != nil {
		t.Fatal(e)
	}
}
