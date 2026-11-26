//go:build integration

package bootstrap_test

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Restore disposable fixtures produced by the released 26 and 27 Staffing APIs.
// Each backup contains its own synthetic operator and original position receipt.
func TestHire400PositionBackups(t *testing.T) {
	root := os.Getenv("CHNHCM_STAFFING_BACKUP_DIR")
	if root == "" {
		t.Skip("requires isolated 00026 and 00027 Staffing API backups")
	}
	for _, version := range []int{26, 27} {
		t.Run(strconv.Itoa(version), func(t *testing.T) {
			dir := filepath.Join(root, "backup"+strconv.Itoa(version))
			raw, err := os.ReadFile(filepath.Join(dir, "fixture.json"))
			must(t, err)
			var seed struct{ Instance, Password, Position string }
			must(t, json.Unmarshal(raw, &seed))
			f := fresh(t)
			_, err = f.admin.Exec(f.ctx, "CREATE DATABASE old_staffing")
			must(t, err)
			dump, err := os.Open(filepath.Join(dir, "before.dump"))
			must(t, err)
			defer dump.Close()
			restore := exec.CommandContext(f.ctx, "docker", "exec", "-i", f.container.GetContainerID(), "pg_restore", "--exit-on-error", "-U", "postgres", "-d", "old_staffing")
			restore.Stdin = dump
			must(t, restore.Run())
			cfg := f.admin.Config()
			cfg.ConnConfig.Database = "old_staffing"
			db, err := pgxpool.NewWithConfig(f.ctx, cfg)
			must(t, err)
			defer db.Close()
			u, err := url.Parse(f.runtime.Pool.Config().ConnString())
			must(t, err)
			u.Path = "/old_staffing"
			var originalVersion int
			must(t, db.QueryRow(f.ctx, "SELECT max(version_id) FROM goose_db_version WHERE is_applied").Scan(&originalVersion))
			if originalVersion != version {
				t.Fatal("unexpected backup ledger", originalVersion)
			}
			if r, err := bootstrap.Open(f.ctx, u.String(), seed.Instance); err == nil {
				r.Close()
				t.Fatal("opened old Staffing database")
			}
			facts := func() map[string]string {
				digests := map[string]string{}
				tables := []string{"position", "position_source", "position_revision", "position_terminal", "position_outbox"}
				if version == 27 {
					tables = append(tables, "quota_control", "quota_terminal")
				}
				for _, table := range tables {
					var digest string
					must(t, db.QueryRow(f.ctx, "SELECT md5(string_agg(to_jsonb(r)::text, '|' ORDER BY to_jsonb(r)::text)) FROM staffing."+table+" r").Scan(&digest))
					digests[table] = digest
				}
				return digests
			}
			before := facts()
			var source, submission string
			type quotaSource struct {
				Original, Submission, Version, Record, Action, RecordSource string
				Quantity                                                    *int
			}
			var quotaSources []quotaSource
			must(t, db.QueryRow(f.ctx, "SELECT source_id, submission_id FROM staffing.position_source WHERE object_id=$1", seed.Position).Scan(&source, &submission))
			if version == 27 {
				rows, err := db.Query(f.ctx, `SELECT to_jsonb(s)::text, s.submission_id, s.source_id, s.target_record, s.action,
					r.source_id, r.quantity FROM staffing.quota_source s LEFT JOIN staffing.quota_record r
					ON (r.instance_id,r.position_id,r.record_id)=(s.instance_id,s.position_id,s.target_record)
					AND r.source_id=CASE WHEN s.action='DELETE' THEN s.previous_source ELSE s.source_id END
					WHERE s.position_id=$1 ORDER BY s.source_no`, seed.Position)
				must(t, err)
				quotaSources, err = pgx.CollectRows(rows, pgx.RowToStructByPos[quotaSource])
				must(t, err)
				if len(quotaSources) == 0 {
					t.Fatal("expected the original quota records")
				}
			}
			for range 2 {
				must(t, bootstrap.Migrate(f.ctx, db, migrations.Files))
			}
			if !reflect.DeepEqual(before, facts()) {
				t.Fatal("position or quota evidence changed")
			}
			r, err := bootstrap.Open(f.ctx, u.String(), seed.Instance)
			must(t, err)
			defer r.Close()
			server := httptest.NewUnstartedServer(nil)
			server.Config.Handler = (&bootstrap.API{Runtime: r, Origin: "http://" + server.Listener.Addr().String(), InsecureLocal: true}).Handler()
			server.Start()
			defer server.Close()
			restored := fixture{t: t, ctx: f.ctx, admin: db, instance: seed.Instance, runtime: r, server: server}
			b := restored.login("admin", seed.Password)
			b.call("GET", "/api/v1/positions/"+seed.Position+"?sourceVersion="+source, nil, "", 200)
			receipt := b.call("POST", "/api/v1/positions/submissions/"+submission+"/resolve", nil, "", 200)
			if receipt["outcome"] != "accepted" || receipt["objectId"] != seed.Position || receipt["sourceVersion"] != source {
				t.Fatal("original position receipt mismatch")
			}
			for _, q := range quotaSources {
				var preserved bool
				must(t, db.QueryRow(f.ctx, "SELECT snapshot->'originalSource'=$2::jsonb FROM staffing.quota_source WHERE source_id=$1", q.Version, q.Original).Scan(&preserved))
				got := b.call("POST", "/api/v1/staffing-quotas/submissions/"+q.Submission+"/resolve", nil, "", 200)
				if !preserved || got["sourceVersion"] != q.Version || got["action"] != q.Action || got["outcome"] != "accepted" ||
					got["positionId"] != seed.Position || got["selectedRecordId"] != q.Record {
					t.Fatalf("%s: original quota or receipt mismatch", q.Action)
				}
				recordSource := got["sourceVersion"].(string)
				if original, ok := got["originalSourceVersion"].(string); ok {
					recordSource = original
				}
				if recordSource != q.RecordSource {
					t.Fatalf("%s: receipt points to %s, want original record at %s", q.Action, recordSource, q.RecordSource)
				}
				detail := b.call("GET", "/api/v1/staffing-quotas/positions/"+seed.Position+"?sourceVersion="+recordSource+"&selectedRecordId="+q.Record, nil, "", 200)
				if detail["recordId"] != q.Record || q.Quantity == nil && detail["quota"] != nil || q.Quantity != nil && detail["quota"] != float64(*q.Quantity) {
					t.Fatalf("%s: original record or quota value mismatch", q.Action)
				}
			}
			if len(quotaSources) != 0 {
				currentInput := map[string]any{"positionId": seed.Position, "effectiveDate": "2027-04-01", "quota": 2, "reason": "首次核编"}
				conflict := b.call("POST", "/api/v1/staffing-quotas", currentInput, quotaSources[0].Submission, 409)
				if conflict["code"] != "M03_SUBMISSION_MISMATCH" {
					t.Fatal("reusing an original submission ID must not open a new write")
				}
				_, err := db.Exec(f.ctx, "DELETE FROM access.subject_grant WHERE principal_id=$1 AND action='staffingquota.history.read'", b.subject)
				must(t, err)
				for _, q := range quotaSources {
					limited := b.call("POST", "/api/v1/staffing-quotas/submissions/"+q.Submission+"/resolve", nil, "", 200)
					if limited["outcome"] != "accepted" || limited["sourceVersion"] != nil || limited["originalSourceVersion"] != nil || limited["selectedRecordId"] != nil {
						t.Fatalf("%s: original receipt ignored current history permissions", q.Action)
					}
				}
			}
			if !reflect.DeepEqual(before, facts()) {
				t.Fatal("recovering original receipts changed stored evidence")
			}
		})
	}
}
