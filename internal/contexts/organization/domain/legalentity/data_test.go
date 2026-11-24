package legalentity

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"strings"
	"testing"
)

func value(s string) *string { return &s }
func sample() Data {
	id := foundation.NewID()
	return Data{Name: "合成主体", CountryID: id, SubjectType: "branch", RegistrationType: value("登记证"), RegistrationNumber: value("001Ab"), EstablishingOrganization: value("设立单位原始说明"), Adoption: country.Adoption{Date: "1900-01-01", ObjectID: id, SourceVersion: foundation.NewID(), RecordID: foundation.NewID(), Revision: foundation.NewID()}}
}
func TestRegistrationPatchAndEvidence(t *testing.T) {
	p := Policy{}
	before := sample()
	after := p.Clone(before)
	*after.RegistrationNumber = "changed"
	*after.EstablishingOrganization = "changed"
	if *before.RegistrationNumber != "001Ab" || *before.EstablishingOrganization != "设立单位原始说明" {
		t.Fatal("clone alias")
	}
	for _, patch := range []Patch{{"registrationType": nil, "registrationNumber": nil}, {"registrationType": value("新登记"), "registrationNumber": value(" 000aB ")}, {"subjectType": value("unincorporated")}, {"establishingOrganization": nil}} {
		v, e := p.Apply(before, &patch)
		if e != nil {
			t.Fatal(e)
		}
		if !p.EqualRevision(before, v) {
			t.Fatal("non-reference edit changed adoption")
		}
		if patch["registrationNumber"] != nil && *v.RegistrationNumber != "000aB" {
			t.Fatal("identifier changed")
		}
	}
	for _, patch := range []Patch{{"name": nil}, {"subjectType": value("company")}, {"registrationType": nil}, {"registrationNumber": nil}, {"registrationType": value("\n证")}, {"registrationNumber": value(strings.Repeat("界", 101))}, {"establishingOrganization": value(strings.Repeat("界", 201))}, {"countryRegionId": nil}, {"use": value("employer")}, {"code": value("NEW")}} {
		if _, e := p.Apply(before, &patch); e == nil {
			t.Fatal("invalid patch admitted", patch)
		}
	}
	after = p.Clone(before)
	after.Adoption.Revision = foundation.NewID()
	if !p.EqualData(before, after) || p.EqualRevision(before, after) {
		t.Fatal("values and evidence conflated")
	}
}
func TestCountryAdoptionMatrix(t *testing.T) {
	first := Record{ID: foundation.NewID(), Date: "1900-01-01", Type: "CREATE", Data: sample()}
	next := first
	next.ID = foundation.NewID()
	next.Date = "2000-01-01"
	next.Type = "CHANGE"
	cases := []struct {
		name   string
		before []Record
		r      Record
		prev   *Record
		want   bool
	}{
		{"create", nil, first, nil, true}, {"same correction", []Record{first}, first, nil, false}, {"unchanged insertion", []Record{first}, next, &first, false}, {"retained successor", []Record{first, next}, next, &first, false},
		{"stop", nil, Record{Type: "DEACTIVATE", Data: first.Data}, &first, false}, {"resume", []Record{first}, Record{Type: "REACTIVATE", Data: first.Data}, &first, true},
	}
	moved := first
	moved.Date = "1901-01-01"
	cases = append(cases, struct {
		name   string
		before []Record
		r      Record
		prev   *Record
		want   bool
	}{"redate", []Record{first}, moved, nil, true})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if NeedsAdoption(c.before, c.r, c.prev) != c.want {
				t.Fatal("wrong adoption decision")
			}
		})
	}
}
func TestDatedStopInheritsAllFields(t *testing.T) {
	ctx := context.Background()
	first, e := Kernel.Create(ctx, foundation.NewID(), "1900-01-01", "1900-01-01", "NEW_LEGAL_ENTITY", sample())
	if e != nil {
		t.Fatal(e)
	}
	first.Records[0].Revision = foundation.NewID()
	patch := Patch{"registrationNumber": value("000ZZ")}
	work, e := Kernel.Prepare(ctx, first.Records, first.Records[0].ID, "1900-01-01", []Item{{ID: "change", Action: "ADD", Date: "2000-01-01", Type: "CHANGE", Reason: "REGISTRATION_ADJUSTMENT", Changes: &patch}, {ID: "stop", Action: "ADD", Date: "2001-01-01", Type: "DEACTIVATE", Reason: "ENTITY_CLOSURE"}})
	if e != nil {
		t.Fatal(e)
	}
	final, e := Kernel.Finalize(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	if len(final.Records) != 3 || *final.Records[2].Data.RegistrationNumber != "000ZZ" || !(Policy{}).EqualRevision(final.Records[0].Data, final.Records[2].Data) {
		t.Fatal("stop failed to inherit complete data")
	}
}
