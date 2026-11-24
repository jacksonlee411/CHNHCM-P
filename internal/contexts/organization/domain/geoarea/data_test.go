package geoarea

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"testing"
)

func ref(id string) Reference {
	return Reference{ObjectID: id, SourceVersion: foundation.NewID(), RecordID: foundation.NewID(), Revision: foundation.NewID()}
}
func sample() Data {
	country := foundation.NewID()
	english := "Province"
	return Data{Name: "广东省", NameEn: &english, Structure: Structure{CountryID: country, CountryCode: "CN", Scheme: "CN-MCA", Code: "440000", Nature: "administrative", Level: "province"}, Adoption: Adoption{Date: "1900-01-01", Country: ref(country)}}
}
func TestDisplayMaintenanceProtectsStructureAndAdoption(t *testing.T) {
	p := Policy{}
	before := sample()
	name := "广东显示名称"
	after, e := p.Apply(before, &Patch{"name": &name, "nameEn": nil})
	if e != nil {
		t.Fatal(e)
	}
	if after.Name != name || after.NameEn != nil || before.NameEn == nil || !p.EqualRevision(before, after) || before.Structure != after.Structure {
		t.Fatal("display patch lost fixed evidence")
	}
	for _, field := range []string{"code", "countryRegionId", "parentAreaId", "nature", "level", "scheme", "effectiveDate", "adoption", "source"} {
		if _, e := p.Apply(before, &Patch{field: &name}); e == nil {
			t.Fatal("protected field accepted", field)
		}
	}
	clone := p.Clone(before)
	*clone.NameEn = "changed"
	if *before.NameEn == "changed" {
		t.Fatal("alias")
	}
	bad := sample()
	bad.Adoption.Country.Revision = ""
	if _, e = p.Normalize(bad); e == nil {
		t.Fatal("incomplete revision accepted")
	}
	child := before
	child.Structure.Level = "county"
	child.Structure.Code = "440106"
	child.Structure.ParentID = foundation.NewID()
	child.Adoption.Parents = []Reference{ref(child.Structure.ParentID)}
	clone = p.Clone(child)
	clone.Adoption.Parents[0].Revision = foundation.NewID()
	if p.EqualRevision(child, clone) {
		t.Fatal("adoption alias or comparison")
	}
}
func TestSharedTimelineKeepsAdoptionOnStatusChange(t *testing.T) {
	data := sample()
	id := foundation.NewID()
	initial, e := Kernel.Create(context.Background(), id, "1900-01-01", MinDate, "SOURCE_INITIALIZATION", data)
	if e != nil {
		t.Fatal(e)
	}
	initial.Records[0].Revision = foundation.NewID()
	stop, e := Kernel.Calculate(context.Background(), initial.Records, id, MinDate, []Item{{ID: "stop", Action: "ADD", Date: "2026-01-01", Type: "DEACTIVATE", Reason: "STOP_SELECTION"}})
	if e != nil {
		t.Fatal(e)
	}
	if len(stop.Records) != 2 || stop.Records[1].State != "inactive" || !(Policy{}).EqualRevision(data, stop.Records[1].Data) {
		t.Fatal("state change rewrote adoption")
	}
}
func TestStructureRejectsBadParentAndAcceptsRealMissingLevel(t *testing.T) {
	s := sample().Structure
	root := Node{ID: foundation.NewID(), EffectiveDate: "1900-01-01", Structure: s}
	child := Node{ID: foundation.NewID(), EffectiveDate: "1900-01-01", Structure: s}
	child.Structure.Code = "440106"
	child.Structure.Level = "county"
	child.Structure.ParentID = root.ID
	if _, e := ValidateTree([]Node{child, root}); e != nil {
		t.Fatal("legal missing level rejected", e)
	}
	for _, mutate := range []func(*Node){func(n *Node) { n.Structure.ParentID = n.ID }, func(n *Node) { n.Structure.ParentID = foundation.NewID() }, func(n *Node) { n.Structure.CountryID = foundation.NewID() }, func(n *Node) { n.EffectiveDate = "1800-01-01" }, func(n *Node) { n.Structure.Level = "province" }, func(n *Node) { n.Structure.Code = root.Structure.Code }} {
		bad := child
		mutate(&bad)
		if _, e := ValidateTree([]Node{root, bad}); e == nil {
			t.Fatal("illegal relation accepted", bad)
		}
	}
}
