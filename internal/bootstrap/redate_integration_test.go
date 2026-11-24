//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
	"testing"
	"time"
)

func TestRedateOriginalDayAcrossMidnight(t *testing.T) {
	for _, kind := range []string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"} {
		t.Run(kind, func(t *testing.T) {
			f := fresh(t)
			b := f.login("admin", f.password)
			endpoint, in, _ := prepareCommitCommand(b, "REDATE_MIDNIGHT", "REDATE_"+kind)
			svc := f.runtime.Organization.(application.Service)
			store := svc.Store.(*orgdb.Store)
			before, _ := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
			store.Clock = func(context.Context, pgx.Tx) (time.Time, error) { return before, nil }
			restoreRecorded := f.recordedAt("2026-06-14T16:00:01Z")
			key := foundation.NewID()
			saved := b.call("POST", endpoint, in, key, 200)
			expected := "active"
			if kind == "CREATE" {
				expected = "not_effective"
			}
			if kind == "REACTIVATE" {
				expected = "inactive"
			}
			if saved["evaluatedBusinessDate"] != "2026-06-14" || saved["currentStatus"] != expected {
				t.Fatal("original redate day/state changed", saved)
			}
			restoreRecorded()
			store.Clock = nil
			r := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
			if r["evaluatedBusinessDate"] != "2026-06-14" || r["evaluatedAt"] != "2026-06-14T15:59:59Z" || r["currentStatus"] != expected {
				t.Fatal("recovery relabeled redate", r)
			}
			recorded := f.buSource(saved["sourceVersion"].(string)).Operation.Source.RecordedAt.UTC().Format(time.RFC3339Nano)
			if recorded != "2026-06-14T16:00:01Z" {
				t.Fatal(recorded)
			}
		})
	}
}
