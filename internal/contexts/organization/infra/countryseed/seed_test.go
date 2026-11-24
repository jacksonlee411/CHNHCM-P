package countryseed

import (
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	"chnhcm.local/chnhcm/internal/foundation"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestFixedCountriesMatchAdmittedInput(t *testing.T) {
	p, e := Fixed()
	if e != nil {
		t.Fatal(e)
	}
	f, e := os.Open("../../../../../tests/fixtures/geography/v1/package.json.gz")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	z, e := gzip.NewReader(f)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	var original struct {
		Countries []Row `json:"countries"`
	}
	if e = json.NewDecoder(z).Decode(&original); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(p.Countries, original.Countries) {
		t.Fatal("country fields differ from fixed G0 input")
	}
	instance, subject := foundation.NewID(), foundation.NewID()
	seen := map[string]bool{}
	for _, row := range p.Countries {
		a, e := p.Command(instance, subject, row)
		if e != nil {
			t.Fatal(e)
		}
		if _, e := d.Kernel.Create(context.Background(), foundation.NewID(), a.Import.EffectiveDate, d.MinDate, "SOURCE_INITIALIZATION", d.Data{Name: a.Import.Name, NameEn: a.Import.NameEn, PhoneCode: a.Import.PhoneCode}); e != nil {
			t.Fatal(row.Code, e)
		}
		o := a.Import.Origin
		if e := d.ValidateOrigin(d.Origin{Scheme: o.Scheme, Key: o.Key, Code: o.Code, Name: o.Name, NameEn: o.NameEn, PackageVersion: o.PackageVersion, PackageDigest: o.PackageDigest, ArchiveDigest: o.ArchiveDigest}); e != nil {
			t.Fatal(row.Code, e)
		}
		b, _ := p.Command(instance, subject, row)
		if !foundation.ValidID(a.Import.ID) || !foundation.ValidID(a.SubmissionID) || !reflect.DeepEqual(a, b) || seen[a.Import.ID] {
			t.Fatal("unstable or duplicate identity", row.Code)
		}
		seen[a.Import.ID] = true
		other, _ := p.Command(foundation.NewID(), subject, row)
		if other.Import.ID == a.Import.ID {
			t.Fatal("instance identity leaked")
		}
	}
}
