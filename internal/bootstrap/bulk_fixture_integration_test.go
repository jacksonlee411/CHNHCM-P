//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"strings"
)

// Controlled synthetic historical source preparation, not a public command or an
// alternative runtime path. It keeps the long-current-line capacity fixture
// independent of HTTP admission and of the retired arbitrary write operation.
func seedLongCurrentSource(ctx context.Context, f *fixture, store application.Store, creds access.Credentials, id, key, subject string, digest []byte, items []domain.Item) (result org.Result, err error) {
	// The three explicit sizes reuse this synthetic principal and enterprise.
	// Reuse its test-only authority rather than inserting a duplicate scope.
	f.exec(`INSERT INTO access.subject_grant SELECT $1,$2,$3,'enterprise',enterprise_id,'bu.timeline.revise',ARRAY[]::text[],ARRAY[]::text[] FROM organization.enterprise WHERE instance_id=$1 ON CONFLICT(instance_id,principal_id,scope_kind,scope_id,action) DO NOTHING`, f.instance, foundation.NewID(), subject)
	err = store.WithinWrite(ctx, creds, func(u application.Unit) error {
		if e := u.LockSubmission(ctx, key); e != nil {
			return e
		}
		ent, e := u.Enterprise(ctx, true)
		if e != nil {
			return e
		}
		tl, e := u.LoadTimeline(ctx, id, true)
		if e != nil {
			return e
		}
		slices.SortFunc(items, func(a, b domain.Item) int { return strings.Compare(a.ID, b.ID) })
		meta := []org.RequestItem{}
		results := []org.ItemResult{}
		checks := []access.Intent{}
		for _, it := range items {
			fields := []string{"effectiveDate", "name", "operationReason", "operationType"}
			meta = append(meta, org.RequestItem{ItemID: it.ID, Action: "ADD", EffectiveDate: it.Date, NewType: it.Type, DeclaredFields: fields, ProcessingFacets: []string{"ADD"}})
			results = append(results, org.ItemResult{ItemID: it.ID, Outcome: "accepted", RecordID: it.ID, EffectiveDate: it.Date, ProcessingFacets: []string{"ADD"}})
			checks = append(checks, access.Intent{Action: "bu.record.add", EnterpriseID: ent.ID, BUID: id, Fields: fields, ItemID: it.ID, Kind: "historical-fixture-item"})
		}
		decision, e := u.Access().Decide(ctx, access.Intent{Action: "bu.timeline.revise", EnterpriseID: ent.ID, BUID: id, Kind: "historical-fixture", Fields: []string{}, Requirements: checks})
		if e != nil {
			return e
		}
		if !decision.Allowed {
			return foundation.Fail(403, "FIXTURE_DENIED", "historical fixture authority denied")
		}
		candidate, e := domain.Calculate(ctx, tl.Records, tl.First, ent.ValidFrom, items)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		w := application.Write{Timeline: tl, Candidate: candidate, Items: items, Terminal: application.Terminal{ID: key, Action: "REVISE_TIMELINE", BUID: id, ExpectedSource: tl.Source, Digest: digest, Fields: []string{}, Items: meta, ItemOutcomes: results, EvaluatedAt: &now}}
		if e := u.Business(ctx, func() error { var e error; result, e = u.Persist(ctx, w); return e }); e != nil {
			return e
		}
		u.Confirm()
		return nil
	})
	return
}
