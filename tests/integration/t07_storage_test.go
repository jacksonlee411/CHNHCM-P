//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"reflect"
	"testing"
)

// Native owner/directory guards and typed logical checks replace the old wide
// member FKs. Rehashed corruption deliberately passes the physical checksum.
func TestT07StorageCompositeRangesAndSourceChains(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	sources := []map[string]any{}
	for _, code := range []string{"T07_SQL_A", "T07_SQL_B"} {
		created := b.call("POST", "/api/v1/business-units", createBody(code, "2026-01-01"), foundation.NewID(), 201)
		id := created["buId"].(string)
		added := b.call("POST", "/api/v1/business-units/"+id+"/records", addBody(detail(b, id), "2026-06-01", map[string]any{"name": "后继"}), foundation.NewID(), 201)
		sources = append(sources, added)
	}
	source := sources[0]["sourceVersion"].(string)
	original := f.buSource(source)
	foreign := f.buSource(sources[1]["sourceVersion"].(string))
	before := businessSnapshot(t, f)
	cases := []struct {
		name string
		edit func(*bootstrap.BUHistoryFact)
	}{
		{"TL229_foreign_BU_revision", func(v *bootstrap.BUHistoryFact) {
			v.State.Timeline.Records[0].Revision = foreign.State.Timeline.Records[0].Revision
		}},
		{"TL229_wrong_record_revision", func(v *bootstrap.BUHistoryFact) {
			v.State.Timeline.Records[1].Revision = v.State.Timeline.Records[0].Revision
		}},
		{"TL230_wrong_copied_date", func(v *bootstrap.BUHistoryFact) { v.State.Timeline.Records[1].Date = "2026-06-02" }},
		{"TL230_wrong_range_only", func(v *bootstrap.BUHistoryFact) { v.State.Timeline.Records[0].End = "2026-05-01" }},
		{"TL230_missing_date", func(v *bootstrap.BUHistoryFact) {
			v.State.Timeline.Records[0].Date = ""
			v.State.Timeline.Records[0].End = ""
		}},
		{"TL230_overlap", func(v *bootstrap.BUHistoryFact) { v.State.Timeline.Records[0].End = "2026-07-01" }},
		{"TL230_duplicate_day", func(v *bootstrap.BUHistoryFact) { v.State.Timeline.Records[1].Date = v.State.Timeline.Records[0].Date }},
		{"TL231_skipped_source_number", func(v *bootstrap.BUHistoryFact) { v.State.Timeline.SourceNo += 2 }},
		{"TL231_self_cycle", func(v *bootstrap.BUHistoryFact) { v.Operation.Source.Previous = v.Operation.Source.ID }},
		{"TL231_cross_record_predecessor", func(v *bootstrap.BUHistoryFact) {
			v.Operation.Revisions[0].Previous = foreign.State.Timeline.Records[0].Revision
		}},
	}
	physicalRejection := map[string]string{"TL230_missing_date": "23P01", "TL230_duplicate_day": "23514"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if state := physicalRejection[tc.name]; state != "" {
				// Missing or duplicate boundaries create overlapping or empty
				// derived ranges. Native constraints reject the entire corruption
				// transaction before it can commit.
				// Keep that physical protection and verify the old source survives.
				e := bootstrap.MutateIntegrationBU(f.ctx, f.admin, f.runtime, original.Object, source, tc.edit)
				t07SQLState(t, e, state)
				b.call("GET", "/api/v1/business-units/"+original.Object+"?sourceVersion="+source, nil, "", 200)
				if !reflect.DeepEqual(original, f.buSource(source)) {
					t.Fatal("rejected corruption changed the original source")
				}
				return
			}
			f.editBU(source, tc.edit)
			got := b.call("GET", "/api/v1/business-units/"+original.Object+"?sourceVersion="+source, nil, "", 503)
			if got["viewFact"] != nil {
				t.Fatal("corrupt source exposed a fact")
			}
			f.editBU(source, func(v *bootstrap.BUHistoryFact) { *v = original })
			b.call("GET", "/api/v1/business-units/"+original.Object+"?sourceVersion="+source, nil, "", 200)
		})
	}
	t.Run("TL231_cross_BU_head", func(t *testing.T) {
		tx, e := f.runtime.Pool.Begin(f.ctx)
		must(t, e)
		defer tx.Rollback(f.ctx)
		_, e = tx.Exec(f.ctx, `UPDATE organization.business_unit SET head_source_id=$2 WHERE bu_id=$1`, original.Object, foreign.Operation.Source.ID)
		must(t, e)
		_, e = tx.Exec(f.ctx, "SET CONSTRAINTS ALL IMMEDIATE")
		t07SQLState(t, e, "23514")
	})
	if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
		t.Fatal("corruption fixture changed restored logical history")
	}
}
