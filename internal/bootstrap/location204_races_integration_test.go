//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	geo "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

type locationGate204 struct {
	app.Store
	kind, id        string
	locked, release chan struct{}
}
type locationGateUnit204 struct {
	app.Unit
	gate *locationGate204
}
type locationGateRefs204 struct {
	geo.ReferenceUnit
	gate *locationGate204
}

func (g *locationGate204) WithinLocationWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return g.Store.WithinLocationWrite(ctx, c, func(u app.Unit) error { return fn(locationGateUnit204{u, g}) })
}
func (u locationGateUnit204) References() geo.ReferenceUnit {
	return locationGateRefs204{u.Unit.References(), u.gate}
}
func (u locationGateRefs204) LockReference(ctx context.Context, kind, id string) error {
	if e := u.ReferenceUnit.LockReference(ctx, kind, id); e != nil {
		return e
	}
	if kind == u.gate.kind && id == u.gate.id {
		close(u.gate.locked)
		select {
		case <-u.gate.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
func TestLocation204ReferenceRaces(t *testing.T) {
	f := freshWithEnterpriseStart(t, "1900-01-01")
	must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	pub, cmds := geoPartial203(t, f)
	must(t, f.runtime.GeoArea.(geo.Service).InitializeCatalog(f.ctx, geoCredentials203(t, f, "admin", f.password), pub, cmds))
	b := f.login("admin", f.password)
	svc := f.runtime.Location.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	country, area := geoCountry203(t, f, "HK"), geoID203(t, f, "CW")
	for index, target := range []struct{ kind, id, path string }{{"country", country, "country-regions"}, {"geoarea", geoID203(t, f, "HK"), "geo-areas"}, {"geoarea", area, "geo-areas"}} {
		stopDate := fmt.Sprintf("%04d-01-01", 2000+index*5)
		resumeDate := fmt.Sprintf("%04d-01-02", 2000+index*5)
		gate := &locationGate204{Store: svc.Store, kind: target.kind, id: target.id, locked: make(chan struct{}), release: make(chan struct{})}
		gated := app.Service{Store: gate}
		cmd := org.LocationCommand{Action: "CREATE", SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Create: org.LocationCreateInput{Code: fmt.Sprintf("LOCK-%d", index), Name: "锁内地点", CountryRegionID: country, GeoAreaID: &area, AddressLine: "一号", EffectiveDate: stopDate, OperationType: "CREATE", OperationReason: "NEW_LOCATION"}}
		result := make(chan error, 1)
		go func() { _, e := gated.Maintain(f.ctx, c, cmd); result <- e }()
		select {
		case <-gate.locked:
		case e := <-result:
			t.Fatalf("adoption did not lock: %v", e)
		case <-time.After(3 * time.Second):
			t.Fatal("lock gate timeout")
		}
		detail := b.call("GET", "/api/v1/"+target.path+"/"+target.id, nil, "", 200)
		r := detail["selectedRecord"].(map[string]any)
		body := map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": stopDate, "operationType": "DEACTIVATE", "operationReason": "STOP_SELECTION"}
		stopped := make(chan error, 1)
		go func() {
			status, _, e := b.response("POST", "/api/v1/"+target.path+"/"+target.id+"/deactivate", body, foundation.NewID())
			if e == nil && status != 201 {
				e = fmt.Errorf("stop status %d", status)
			}
			stopped <- e
		}()
		deadline := time.Now().Add(time.Second)
		for f.scalar("SELECT count(*) FROM pg_locks WHERE NOT granted AND locktype='transactionid'") == 0 {
			if time.Now().After(deadline) {
				close(gate.release)
				t.Fatal("stop was not serialized behind reference lock")
			}
			time.Sleep(5 * time.Millisecond)
		}
		close(gate.release)
		must(t, <-result)
		must(t, <-stopped)
		// A subsequent new adoption at the stopped date is refused, and no Location is published.
		cmd.SubmissionID = foundation.NewID()
		cmd.Create.Code = fmt.Sprintf("AFTER-STOP-%d", index)
		_, e := svc.Maintain(f.ctx, c, cmd)
		if e == nil {
			t.Fatal("stop-first adoption accepted")
		}
		if f.scalar("SELECT count(*) FROM organization.location WHERE code=$1", cmd.Create.Code) != 0 {
			t.Fatal("partial address persisted")
		}
		detail = b.call("GET", "/api/v1/"+target.path+"/"+target.id, nil, "", 200)
		r = detail["selectedRecord"].(map[string]any)
		b.call("POST", "/api/v1/"+target.path+"/"+target.id+"/reactivate", map[string]any{"expectedSourceVersion": detail["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": resumeDate, "operationType": "REACTIVATE", "operationReason": "RESUME_SELECTION"}, foundation.NewID(), 201)
		t.Log("real locked Location adoption then catalog stop both committed; reverse order refused", target.kind, index)
	}
	// The consumer's actual transaction acquires the same finite union even
	// when a caller supplies countries in opposite order.
	mo := geoCountry203(t, f, "MO")
	requests := []geo.ReferenceRequest{{CountryID: country, AreaID: area, Date: "2021-01-01"}, {CountryID: mo, Date: "2021-01-01"}}
	expectedCountries := []string{country, mo}
	slices.Sort(expectedCountries)
	expected := []string{"country:" + expectedCountries[0], "country:" + expectedCountries[1], "geoarea:" + geoID203(t, f, "HK"), "geoarea:" + area}
	for round := 0; round < 2; round++ {
		must(t, svc.Store.WithinLocationWrite(f.ctx, c, func(u app.Unit) error {
			var locks []string
			refs := locationOrderRefs204{ReferenceUnit: u.References(), locks: &locks}
			adopted, e := geo.ResolveAddresses(f.ctx, refs, requests)
			if e != nil {
				return e
			}
			if !slices.Equal(locks, expected) || len(adopted) != 2 {
				return fmt.Errorf("unexpected union lock order")
			}
			if adopted[0].Country.ObjectID != requests[0].CountryID {
				return fmt.Errorf("input-order evidence changed")
			}
			u.Confirm() // Explicitly finish this lock-only read transaction.
			return nil
		}))
		slices.Reverse(requests)
	}
	original := b.call("GET", "/api/v1/geo-areas/"+area, nil, "", 200)
	originalName := original["selectedRecord"].(map[string]any)["name"].(string)
	gate := &locationGate204{Store: svc.Store, kind: "geoarea", id: area, locked: make(chan struct{}), release: make(chan struct{})}
	create := org.LocationCommand{Action: "CREATE", SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Create: org.LocationCreateInput{Code: "RENAME-LOCK", Name: "更名竞争", CountryRegionID: country, GeoAreaID: &area, AddressLine: "一号", EffectiveDate: "2021-01-01", OperationType: "CREATE", OperationReason: "NEW_LOCATION"}}
	type saved struct {
		value org.LocationResult
		err   error
	}
	done := make(chan saved, 1)
	go func() { r, e := (app.Service{Store: gate}).Maintain(f.ctx, c, create); done <- saved{r, e} }()
	select {
	case <-gate.locked:
	case <-time.After(3 * time.Second):
		t.Fatal("rename reference lock not reached")
	}
	renamed := make(chan error, 1)
	r := original["selectedRecord"].(map[string]any)
	go func() {
		status, _, e := b.response("POST", "/api/v1/geo-areas/"+area+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": original["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "竞争之后的新名称"}}, foundation.NewID())
		if e == nil && status != 200 {
			e = fmt.Errorf("rename status %d", status)
		}
		renamed <- e
	}()
	deadline := time.Now().Add(time.Second)
	for f.scalar("SELECT count(*) FROM pg_locks WHERE NOT granted AND locktype='transactionid'") == 0 {
		if time.Now().After(deadline) {
			close(gate.release)
			t.Fatal("rename did not wait for adoption")
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(gate.release)
	result := <-done
	must(t, result.err)
	must(t, <-renamed)
	old := b.call("GET", "/api/v1/locations/"+*result.value.LocationID, nil, "", 200)
	if !strings.Contains(fmt.Sprint(old["selectedRecord"].(map[string]any)["geoAreaPath"]), originalName) {
		t.Fatal("concurrent rename replaced original adopted label")
	}
	create.SubmissionID = foundation.NewID()
	create.Create.Code = "RENAME-AFTER"
	newer, e := svc.Maintain(f.ctx, c, create)
	must(t, e)
	now := b.call("GET", "/api/v1/locations/"+*newer.LocationID, nil, "", 200)
	if !strings.Contains(fmt.Sprint(now["selectedRecord"].(map[string]any)["geoAreaPath"]), "竞争之后的新名称") {
		t.Fatal("new adoption did not use committed name")
	}

}

type locationOrderRefs204 struct {
	geo.ReferenceUnit
	locks *[]string
}

func (u locationOrderRefs204) LockReference(ctx context.Context, kind, id string) error {
	*u.locks = append(*u.locks, kind+":"+id)
	return u.ReferenceUnit.LockReference(ctx, kind, id)
}
