package location

import (
	geo "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"strings"
	"testing"
)

func value(s string) *string { return &s }
func sample() Data {
	country, area := foundation.NewID(), foundation.NewID()
	ref := func(id string) geo.Reference {
		return geo.Reference{ObjectID: id, SourceVersion: foundation.NewID(), RecordID: foundation.NewID(), Revision: foundation.NewID()}
	}
	return Data{Name: "办公室", CountryID: country, AreaID: &area, AddressLine: "门牌一号", PostalCode: value("00100"), Adoption: geo.Adoption{Date: "2020-01-01", Country: ref(country), Parents: []geo.Reference{ref(area)}}}
}
func TestFieldsAndImmutableEvidence(t *testing.T) {
	p := Policy{}
	old := sample()
	v := p.Clone(old)
	v.Adoption.Parents[0].Revision = foundation.NewID()
	*v.PostalCode = "2"
	if old.Adoption.Parents[0].Revision == v.Adoption.Parents[0].Revision || *old.PostalCode != "00100" {
		t.Fatal("clone aliases evidence")
	}
	patch := Patch{"addressLine": value(" 二号 "), "postalCode": value("00100")}
	got, e := p.Apply(old, &patch)
	if e != nil || got.AddressLine != "二号" || *got.PostalCode != "00100" || !p.EqualRevision(old, got) {
		t.Fatal(got, e)
	}
	for _, patch := range []Patch{{"addressLine": nil}, {"countryRegionId": nil}, {"addressLine": value("\n一号")}, {"postalCode": value(strings.Repeat("界", 21))}, {"addressExtra": value("已删除")}, {"addressLine": value(strings.Repeat("界", 1002))}, {"code": value("NEW")}, {"adoptionDate": value("2021-01-01")}} {
		if _, e := p.Apply(old, &patch); e == nil {
			t.Fatal("invalid patch admitted", patch)
		}
	}
	same := p.Clone(old)
	same.Adoption.Country.SourceVersion = foundation.NewID()
	if !p.EqualData(old, same) || p.EqualRevision(old, same) {
		t.Fatal("business values and exact evidence conflated")
	}
}
func TestAdoptionDecisionMatrix(t *testing.T) {
	first := Record{ID: foundation.NewID(), Date: "2020-01-01", Type: "CREATE", Data: sample()}
	next := first
	next.ID = foundation.NewID()
	next.Date = "2021-01-01"
	next.Type = "CHANGE"
	for _, tc := range []struct {
		name     string
		rows     []Record
		r        Record
		previous *Record
		want     bool
	}{
		{"create", nil, first, nil, true}, {"unchanged correction", []Record{first}, first, nil, false},
		{"insertion copies predecessor", []Record{first}, next, &first, false},
		{"deactivation inherits", nil, Record{Type: "DEACTIVATE", Data: first.Data}, &first, false},
		{"reactivation reselects", []Record{first}, Record{Type: "REACTIVATE", Data: first.Data}, &first, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NeedsAdoption(tc.rows, tc.r, tc.previous); got != tc.want {
				t.Fatal(got)
			}
		})
	}
	moved := first
	moved.Date = "2020-01-02"
	if !NeedsAdoption([]Record{first}, moved, nil) {
		t.Fatal("redate inherited old eligibility")
	}
	reclassified := next
	reclassified.Type = "REACTIVATE"
	if !NeedsAdoption([]Record{first, next}, reclassified, &first) {
		t.Fatal("classification did not reselect")
	}
	changed := next
	changed.Data = sample()
	if !NeedsAdoption([]Record{first}, changed, &first) {
		t.Fatal("new address did not reselect")
	}
	// Independent successors retain adoption after an earlier record is removed.
	if NeedsAdoption([]Record{first, next}, next, &first) {
		t.Fatal("retained successor reselected")
	}
}
func TestDatedInheritanceKeepsExactAddress(t *testing.T) {
	ctx := context.Background()
	first, e := Kernel.Create(ctx, foundation.NewID(), "2020-01-01", "2020-01-01", "NEW_LOCATION", sample())
	if e != nil {
		t.Fatal(e)
	}
	first.Records[0].Revision = foundation.NewID()
	patch := Patch{"addressLine": value("二号")}
	work, e := Kernel.Prepare(ctx, first.Records, first.Records[0].ID, "2020-01-01", []Item{{ID: "change", Action: "ADD", Date: "2021-01-01", Type: "CHANGE", Reason: "ADDRESS_ADJUSTMENT", Changes: &patch}, {ID: "stop", Action: "ADD", Date: "2022-01-01", Type: "DEACTIVATE", Reason: "LOCATION_CLOSURE"}})
	if e != nil {
		t.Fatal(e)
	}
	final, e := Kernel.Finalize(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	if len(final.Records) != 3 || final.Records[2].Data.AddressLine != "二号" || !(Policy{}).EqualRevision(final.Records[0].Data, final.Records[2].Data) {
		t.Fatal("stop did not inherit exact evidence", final)
	}
}
