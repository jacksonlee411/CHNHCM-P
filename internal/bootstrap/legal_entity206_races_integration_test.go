//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	country "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/legalentity"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
)

type legalGate206 struct {
	app.Store
	id              string
	locked, release chan struct{}
}
type legalGateUnit206 struct {
	app.Unit
	gate *legalGate206
}
type legalGateReference206 struct {
	country.ReferenceUnit
	gate *legalGate206
}

func (g *legalGate206) WithinLegalEntityWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return g.Store.WithinLegalEntityWrite(ctx, c, func(u app.Unit) error { return fn(legalGateUnit206{u, g}) })
}
func (u legalGateUnit206) References() country.ReferenceUnit {
	return legalGateReference206{u.Unit.References(), u.gate}
}
func (u legalGateReference206) LockReference(ctx context.Context, kind, id string) error {
	if e := u.ReferenceUnit.LockReference(ctx, kind, id); e != nil {
		return e
	}
	if id == u.gate.id {
		close(u.gate.locked)
		select {
		case <-u.gate.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

type legalLockOrder206 struct {
	country.ReferenceUnit
	order *[]string
}

func (u legalLockOrder206) LockReference(ctx context.Context, kind, id string) error {
	*u.order = append(*u.order, kind+":"+id)
	return u.ReferenceUnit.LockReference(ctx, kind, id)
}
func TestLegalEntity206CountryRaces(t *testing.T) {
	h := legalSetup206(t)
	f, b := h.f, h.b
	svc := f.runtime.LegalEntity.(app.Service)
	c := geoCredentials203(t, f, "admin", f.password)
	id := h.countries["US"]
	gate := &legalGate206{Store: svc.Store, id: id, locked: make(chan struct{}), release: make(chan struct{})}
	cmd := org.LegalEntityCommand{Action: "CREATE", SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Create: org.LegalEntityCreateInput{Code: "RACE206", Name: "并发主体", CountryRegionID: id, SubjectType: "branch", EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: "NEW_LEGAL_ENTITY"}}
	done := make(chan error, 1)
	go func() { _, e := (app.Service{Store: gate}).Maintain(f.ctx, c, cmd); done <- e }()
	select {
	case <-gate.locked:
	case e := <-done:
		t.Fatalf("adoption failed before lock: %v", e)
	case <-time.After(3 * time.Second):
		t.Fatal("lock timeout")
	}
	path := "/api/v1/country-regions/" + id
	directory := b.call("GET", path, nil, "", 200)
	r := directory["selectedRecord"].(map[string]any)
	stopped := make(chan error, 1)
	go func() {
		status, _, e := b.response("POST", path+"/deactivate", map[string]any{"expectedSourceVersion": directory["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "2000-01-01", "operationType": "DEACTIVATE", "operationReason": "STOP_SELECTION"}, foundation.NewID())
		if e == nil && status != 201 {
			e = fmt.Errorf("stop status %d", status)
		}
		stopped <- e
	}()
	deadline := time.Now().Add(time.Second)
	for f.scalar("SELECT count(*) FROM pg_locks WHERE NOT granted AND locktype='transactionid'") == 0 {
		if time.Now().After(deadline) {
			close(gate.release)
			t.Fatal("country write did not wait for actual reference lock")
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(gate.release)
	must(t, <-done)
	must(t, <-stopped)
	cmd.SubmissionID = foundation.NewID()
	cmd.Create.Code = "STOP-FIRST206"
	if _, e := svc.Maintain(f.ctx, c, cmd); e == nil {
		t.Fatal("stop-first adoption admitted")
	}
	if f.scalar("SELECT count(*) FROM organization.legal_entity WHERE code='STOP-FIRST206'") != 0 {
		t.Fatal("partial source")
	}
	requests := []country.ReferenceRequest{{CountryID: h.countries["CN"], Date: "1900-01-01"}, {CountryID: id, Date: "1900-01-01"}, {CountryID: h.countries["CN"], Date: "1901-01-01"}}
	ids := []string{id, h.countries["CN"]}
	slices.Sort(ids)
	want := []string{"country:" + ids[0], "country:" + ids[1]}
	for round := 0; round < 2; round++ {
		must(t, svc.Store.WithinLegalEntityWrite(f.ctx, c, func(u app.Unit) error {
			var order []string
			out, e := country.ResolveCountries(f.ctx, legalLockOrder206{u.References(), &order}, requests)
			if e != nil {
				return e
			}
			if !slices.Equal(order, want) || out[0].ObjectID != requests[0].CountryID {
				return fmt.Errorf("union order or receipt order differs")
			}
			u.Confirm()
			return nil
		}))
		slices.Reverse(requests)
	}
	// Object-lock cancellation must roll back and release so the same identity can be retried.
	tx, e := f.admin.Begin(f.ctx)
	must(t, e)
	must(t, tx.QueryRow(f.ctx, "SELECT country_region_id FROM organization.country_region WHERE country_region_id=$1 FOR UPDATE", h.countries["MO"]).Scan(new(string)))
	cmd.SubmissionID = foundation.NewID()
	cmd.Create.Code = "CANCEL206"
	cmd.Create.CountryRegionID = h.countries["MO"]
	ctx, cancel := context.WithTimeout(f.ctx, 100*time.Millisecond)
	_, e = svc.Maintain(ctx, c, cmd)
	cancel()
	if e == nil {
		t.Fatal("blocked cancelled save accepted")
	}
	must(t, tx.Rollback(f.ctx))
	result, e := svc.Maintain(f.ctx, c, cmd)
	must(t, e)
	if result.Outcome != "accepted" {
		t.Fatal(result)
	}
	// A concurrent rename must also wait for adoption's shared country row lock;
	// the committed entity keeps the pre-rename source, while the next adopts new.
	id = h.countries["CN"]
	path = "/api/v1/country-regions/" + id
	directory = b.call("GET", path, nil, "", 200)
	r = directory["selectedRecord"].(map[string]any)
	originalName := r["name"]
	gate = &legalGate206{Store: svc.Store, id: id, locked: make(chan struct{}), release: make(chan struct{})}
	cmd.SubmissionID, cmd.Create.Code, cmd.Create.CountryRegionID = foundation.NewID(), "RENAME-FIRST206", id
	adopted := make(chan org.LegalEntityResult, 1)
	go func() { v, e := (app.Service{Store: gate}).Maintain(f.ctx, c, cmd); adopted <- v; done <- e }()
	select {
	case <-gate.locked:
	case e := <-done:
		t.Fatalf("rename adoption failed: %v", e)
	case <-time.After(3 * time.Second):
		t.Fatal("rename lock timeout")
	}
	go func() {
		status, _, e := b.response("POST", path+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": directory["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "并发目录新名"}}, foundation.NewID())
		if e == nil && status != 200 {
			e = fmt.Errorf("rename status %d", status)
		}
		stopped <- e
	}()
	deadline = time.Now().Add(time.Second)
	for f.scalar("SELECT count(*) FROM pg_locks WHERE NOT granted AND locktype='transactionid'") == 0 {
		if time.Now().After(deadline) {
			close(gate.release)
			t.Fatal("rename did not wait for actual reference lock")
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(gate.release)
	must(t, <-done)
	must(t, <-stopped)
	adoptedResult := <-adopted
	if legalRecord206(h.detail(*adoptedResult.LegalEntityID, ""))["countryName"] != originalName {
		t.Fatal("concurrent rename changed original adoption")
	}
	if legalRecord206(h.create("RENAME-AFTER206", "CN", "other"))["countryName"] != "并发目录新名" {
		t.Fatal("post-rename adoption stale")
	}
	t.Log("real country write/adoption serialization in both orders, deduplicated lock order, cancellation and retry verified")
}
