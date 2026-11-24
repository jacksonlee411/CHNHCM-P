package geoareaseed

import (
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	"chnhcm.local/chnhcm/internal/foundation"
	"reflect"
	"slices"
	"testing"
)

func TestFixedInputAndStableParentFirstMapping(t *testing.T) {
	p, e := Fixed()
	if e != nil {
		t.Fatal(e)
	}
	instance := foundation.NewID()
	countries := map[string]string{"base.cn": foundation.NewID(), "base.hk": foundation.NewID(), "base.tw": foundation.NewID()}
	rows, e := p.Prepare(instance, countries)
	if e != nil {
		t.Fatal(e)
	}
	seen := map[string]bool{}
	nodes := []d.Node{}
	roots := 0
	byCode := map[string]string{}
	for _, r := range rows {
		n := r.Node
		if n.Structure.ParentID != "" && !seen[n.Structure.ParentID] {
			t.Fatal("not parent first", r)
		}
		if n.Structure.ParentID == "" {
			roots++
		}
		seen[n.ID] = true
		nodes = append(nodes, n)
		byCode[n.Structure.Code] = n.ID
	}
	leaf, e := d.TerminalIDs(nodes)
	if e != nil {
		t.Fatal(e)
	}
	if roots != 56 || len(leaf) != 3235 {
		t.Fatal(roots, len(leaf))
	}
	for _, c := range []string{"441900", "442000", "460400", "620200", "110108", "419001"} {
		if !leaf[byCode[c]] {
			t.Fatal("real missing-level leaf", c)
		}
	}
	for _, c := range []string{"440000", "110000", "63000"} {
		if leaf[byCode[c]] {
			t.Fatal("non-leaf", c)
		}
	}
	slices.Reverse(p.Areas)
	again, e := p.Prepare(instance, countries)
	if e != nil || !reflect.DeepEqual(rows, again) {
		t.Fatal("input order changed identities/order", e)
	}
	other, e := p.Prepare(foundation.NewID(), countries)
	if e != nil || other[0].Node.ID == rows[0].Node.ID {
		t.Fatal("instance identity not isolated", e)
	}
	p.Areas[0].ParentKey = "missing"
	if _, e = p.Prepare(instance, countries); e == nil {
		t.Fatal("missing parent accepted")
	}
}
func TestFixedScopeAndDuplicateRejection(t *testing.T) {
	for _, change := range []func(*Package){func(p *Package) { p.Areas = p.Areas[:len(p.Areas)-1] }, func(p *Package) { p.Areas[0].CountryKey = "base.mo" }, func(p *Package) { p.Areas[0].SourceKey = p.Areas[1].SourceKey }, func(p *Package) { p.Areas[0].EffectiveDate = "1900-02-30" }, func(p *Package) { p.Areas[0].Code = p.Areas[1].Code }, func(p *Package) { p.Areas[0].Name = "未准入改名" }, func(p *Package) { p.Digest = "changed" }} {
		p, e := Fixed()
		if e != nil {
			t.Fatal(e)
		}
		change(&p)
		_, e = p.Prepare(foundation.NewID(), map[string]string{"base.cn": foundation.NewID(), "base.hk": foundation.NewID(), "base.tw": foundation.NewID()})
		if e == nil {
			t.Fatal("invalid package accepted")
		}
	}
}

func TestMacauIndependentCatalog(t *testing.T) {
	p, err := Macau()
	if err != nil {
		t.Fatal(err)
	}
	instance := foundation.NewID()
	countries := map[string]string{"base.mo": foundation.NewID()}
	rows, err := p.Prepare(instance, countries)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"postal-region": {"澳門", "氹仔", "路環"},
		"parish":        {"花王堂區", "望德堂區", "風順堂區", "大堂區", "花地瑪堂區", "嘉模堂區", "聖方濟各堂區"},
	}
	names := map[string][]string{}
	for _, row := range rows {
		s := row.Node.Structure
		if s.CountryID != countries["base.mo"] || s.Code != "" || s.ParentID != "" || s.Nature != "address_region" {
			t.Fatalf("unexpected Macau structure for %s: %+v", row.Name, s)
		}
		names[s.Level] = append(names[s.Level], row.Name)
	}
	for level, expected := range want {
		slices.Sort(names[level])
		slices.Sort(expected)
		if !slices.Equal(names[level], expected) {
			t.Fatalf("%s: %v", level, names[level])
		}
	}
	slices.Reverse(p.Areas)
	again, err := p.Prepare(instance, countries)
	if err != nil || !reflect.DeepEqual(rows, again) {
		t.Fatal("unstable Macau identity", err)
	}
	p.Areas[0].Code = "01"
	if _, err = p.Prepare(instance, countries); err == nil {
		t.Fatal("accepted invented official code")
	}
}
