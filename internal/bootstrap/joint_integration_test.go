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

func TestJointOriginalBusinessDayAcrossRecordedMidnight(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	saved := b.call("POST", "/api/v1/business-units", createBody("JOINT_MIDNIGHT", "2026-01-01"), foundation.NewID(), 201)
	id := saved["buId"].(string)
	path := "/api/v1/business-units/" + id
	svc := f.runtime.Organization.(application.Service)
	store := svc.Store.(*orgdb.Store)
	before, _ := time.Parse(time.RFC3339, "2026-06-14T15:59:59Z")
	store.Clock = func(context.Context, pgx.Tx) (time.Time, error) { return before, nil }
	restoreRecorded := f.recordedAt("2026-06-14T16:00:01Z")
	key := foundation.NewID()
	d := detail(b, id)
	input := map[string]any{"expectedSourceVersion": d["headSourceVersion"], "anchorRecordId": d["timeline"].([]any)[0].(map[string]any)["recordId"], "deactivationDate": "2026-06-15", "reactivationDate": "2026-06-16", "deactivationReason": "BUSINESS_CLOSURE", "reactivationReason": "BUSINESS_RESUMPTION"}
	result := b.call("POST", path+"/lifecycle-pairs", input, key, 200)

	restoreRecorded()
	store.Clock = nil
	resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	for _, r := range []map[string]any{result, resolved} {
		if r["evaluatedBusinessDate"] != "2026-06-14" || r["recordedAt"] != "2026-06-14T16:00:01Z" || len(r["items"].([]any)) != 2 {
			t.Fatal(r)
		}
	}
}
