//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/bootstrap/capacitydiag"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestT07CurrentTimeline20Records(t *testing.T) {
	if os.Getenv("CHNHCM_T07_PRESSURE") != "1" {
		t.Skip("explicit20-record real DB/API maintenance and recovery lane")
	}
	f := fresh(t)
	observed, finishStages, err := capacitydiag.Open(f.server.Config.Handler)
	must(t, err)
	f.server.Config.Handler = observed
	t.Cleanup(finishStages)
	pressure133NativeSQL(t, f)
	observeSQL := pressure133SQL(t, f)
	b := f.login("admin", f.password)
	b.client.Timeout = 16 * time.Second
	u, e := url.Parse(f.server.URL)
	must(t, e)
	creds := access.Credentials{Token: b.client.Jar.Cookies(u)[0].Value, CSRF: b.csrf, Write: true}
	var savedSubmissions []string
	sizes := []int{maintenanceTimelineRecords}
	for _, size := range sizes {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			created := b.call("POST", "/api/v1/business-units", createBody(fmt.Sprintf("T07_CURRENT_%d", size), "2020-01-01"), foundation.NewID(), 201)
			id := created["buId"].(string)
			items := []domain.Item{}
			startDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			for i := 1; i < size; i++ {
				name := fmt.Sprintf("长线资料%d", i)
				items = append(items, domain.Item{ID: foundation.NewID(), Action: "ADD", Date: startDate.AddDate(0, 0, i*2).Format("2006-01-02"), Type: "CHANGE", Reason: "NAME_ADJUSTMENT", Changes: domain.PatchFromFields(map[string]*string{"name": &name})})
			}
			// Deliberate storage fixture through the real application: bypass ONLY the
			// HTTP body/time admission to provision the long current timeline in one
			// source. This is not evidence that the oversized request is user-admissible.
			key := foundation.NewID()
			digest := sha256.Sum256([]byte(key))
			ctx, cancel := context.WithTimeout(f.ctx, 5*time.Minute)
			fixtureStart := time.Now()
			fixtureService := f.runtime.Organization.(application.Service)
			fixtureStore := *fixtureService.Store.(*orgdb.Store)
			cfg := fixtureStore.Pool.Config()
			cfg.ConnConfig.Tracer = t07FixtureTrace{t}
			fixturePool, e := pgxpool.NewWithConfig(ctx, cfg)
			must(t, e)
			defer fixturePool.Close()
			fixtureStore.Pool = fixturePool
			guard := fixtureStore.Guard
			fixtureStore.Guard = func(ctx context.Context, tx pgx.Tx, c access.Credentials) (access.Guard, error) {
				g, e := guard(ctx, tx, c)
				if e == nil {
					_, e = tx.Exec(ctx, "SET LOCAL statement_timeout='240s'; SET LOCAL idle_in_transaction_session_timeout='240s'")
				}
				return g, e
			}
			fixtureService.Store = &fixtureStore
			fixture, e := seedLongCurrentSource(ctx, f, fixtureService.Store, creds, id, key, b.subject, digest[:], items)
			cancel()
			must(t, e)
			if fixture.Outcome != "accepted" || len(f.buSource(*fixture.SourceVersion).State.Timeline.Records) != size {
				t.Fatal("fixture size mismatch")
			}
			t.Logf("fixture %d current members, one application-created source in %s; HTTP admission deliberately excluded from fixture", size, time.Since(fixtureStart))
			oldDigest := func() string {
				raw, err := json.Marshal(f.buSource(*fixture.SourceVersion))
				must(t, err)
				return fmt.Sprintf("%x", sha256.Sum256(raw))
			}
			originalDigest := oldDigest()
			if freeze133LongInput(t, f, id) {
				return
			}
			// Only the supported ordinary single-record correction is measured.
			// Arbitrary whole-timeline writes retired with CHNHCM-109.
			for _, member := range []int{0, size / 2, size - 1} {
				t.Run(fmt.Sprintf("member_%d", member), func(t *testing.T) {
					var head, record, revision string
					fact := f.buHead(id)
					head, record, revision = fact.Operation.Source.ID, fact.State.Timeline.Records[member].ID, fact.State.Timeline.Records[member].Revision
					name := "单条更正"
					if member > 0 {
						name = fmt.Sprintf("单条更正%d", member)
					}
					body := map[string]any{"expectedSourceVersion": head, "expectedRecordRevision": revision, "changes": map[string]any{"name": name}}
					submission := foundation.NewID()
					var walBefore string
					if os.Getenv("CHNHCM_T07_SQL_OBSERVE") == "1" {
						must(t, f.admin.QueryRow(f.ctx, "SELECT pg_current_wal_insert_lsn()::text").Scan(&walBefore))
					}
					var before, after runtime.MemStats
					runtime.ReadMemStats(&before)
					finishProfile := pressure133CPU(t)
					at := time.Now()
					finishObservation := observeSQL()
					status, out, e := b.response("POST", "/api/v1/business-units/"+id+"/records/"+record+"/correct", body, submission)
					elapsed := time.Since(at)
					finishObservation()
					finishProfile()
					runtime.ReadMemStats(&after)
					if walBefore != "" {
						var walBytes int64
						must(t, f.admin.QueryRow(f.ctx, "SELECT pg_wal_lsn_diff(pg_current_wal_insert_lsn(),$1::pg_lsn)::bigint", walBefore).Scan(&walBytes))
						t.Logf("LONG_SAVE_WAL bytes=%d scope=isolated_database_including_background", walBytes)
					}
					must(t, e)
					t.Logf("ordinary correction current=%d member=%d elapsed=%s status=%d code=%v allocDelta=%d heapAfter=%d", size, member, elapsed, status, out["code"], after.TotalAlloc-before.TotalAlloc, after.HeapAlloc)
					if elapsed > 13*time.Second {
						t.Fatal("write and cleanup exceeded budget")
					}
					if status == 503 {
						t.Error("20-record ordinary save must succeed; recover original identifier for diagnosis")
						out = b.call("POST", "/api/v1/organization/submissions/"+submission+"/resolve", nil, "", 200)
					} else if status != 200 {
						t.Fatal(status, out)
					}
					if out["outcome"] != "accepted" {
						t.Fatal(out)
					}
					current := f.buHead(id)
					if len(current.State.Timeline.Records) != size {
						t.Fatal("ordinary correction changed the member count")
					}
					if out["outcome"] == "accepted" {
						savedSubmissions = append(savedSubmissions, submission)
						if current.Operation.Source.ID != out["sourceVersion"] || current.State.Timeline.Records[member].Data.Name != name {
							t.Fatal("accepted correction was not stored")
						}
					} else if current.Operation.Source.ID != head {
						t.Fatal("not_applied correction changed the head")
					}
					if oldDigest() != originalDigest {
						t.Fatal("original source modified")
					}
					assert133BUReadSegments(t, f, id)
					t.Logf("ordinary correction final current=%d member=%d HTTP=%d outcome=%v headChanged=%t members=%d originalSourceUnchanged=true", size, member, status, out["outcome"], current.Operation.Source.ID != head, len(current.State.Timeline.Records))
					pressure133ExplainBUUpdate(t, f, id)
				})
			}
			// Cancel an application call at a proven real BU-lock boundary. The
			// separate HTTP cancellation probes cover the socket path; this fixture
			// never labels a direct service call as a browser/client cancellation.
			var head, record, revision string
			fact := f.buHead(id)
			head, record, revision = fact.Operation.Source.ID, fact.State.Timeline.Records[0].ID, fact.State.Timeline.Records[0].Revision
			name := strings.Repeat("取", 100)
			cmd := org.MaintainCommand{Action: "CORRECT", BUID: id, RecordID: record, SubmissionID: foundation.NewID(), Digest: digest[:], Fields: []string{"name"}, Correct: org.CorrectInput{ExpectedSourceVersion: head, ExpectedRecordRevision: revision, Changes: org.ChangesInput{"name": &name}}}
			ctx, cancel = context.WithCancel(f.ctx)
			done := make(chan error, 1)
			go func() { _, e := f.runtime.Organization.Maintain(ctx, creds, cmd); done <- e }()
			// Observe the real lock while the production LoadTimeline query is
			// running, rather than waiting for a potentially over-budget load to end.
			locked := false
			until := time.Now().Add(2 * time.Second)
			for time.Now().Before(until) {
				probe, err := f.admin.Begin(f.ctx)
				must(t, err)
				_, err = probe.Exec(f.ctx, "SELECT FROM organization.business_unit WHERE bu_id=$1 FOR UPDATE NOWAIT", id)
				var pg *pgconn.PgError
				locked = errors.As(err, &pg) && pg.Code == "55P03"
				must(t, probe.Rollback(f.ctx))
				if locked {
					break
				}
				time.Sleep(time.Millisecond)
			}
			if !locked {
				cancel()
				<-done
				t.Fatal("BU lock not observed during actual execution")
			}
			at := time.Now()
			cancel()
			e = <-done
			elapsed := time.Since(at)
			if e == nil || time.Since(at) > 3*time.Second {
				t.Fatal("long-line cancellation or cleanup budget", e, time.Since(at))
			}
			// Measure actual release before issuing resolve; recovery traffic must
			// not hide the server's cleanup time.
			cleanupCtx, cleanupCancel := context.WithDeadline(f.ctx, at.Add(3*time.Second))
			for {
				probe, err := f.admin.Begin(cleanupCtx)
				must(t, err)
				_, err = probe.Exec(cleanupCtx, "SELECT FROM organization.business_unit WHERE bu_id=$1 FOR UPDATE NOWAIT", id)
				must(t, probe.Rollback(f.ctx))
				if err == nil {
					break
				}
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "55P03" {
					cleanupCancel()
					t.Fatal("DB lock cleanup observation", err)
				}
				select {
				case <-cleanupCtx.Done():
					cleanupCancel()
					t.Fatal("DB lock cleanup exceeded combined cancellation budget")
				case <-time.After(time.Millisecond):
				}
			}
			dbReleased := time.Since(at)
			cleanupCancel()
			resolved := b.call("POST", "/api/v1/organization/submissions/"+cmd.SubmissionID+"/resolve", nil, "", 200)
			if resolved["outcome"] != "not_applied" {
				t.Fatal("cancelled mutation not closed", resolved)
			}

			if oldDigest() != originalDigest {
				t.Fatal("cancellation modified original source")
			}
			t.Logf("current=%d cancelToServiceReturn=%s cancelToDBLockReacquired=%s; NOWAIT observed before resolve; original identity resolved and exact original source digest preserved", size, elapsed, dbReleased)
			assert133GenericHeadReferenceBounds(t, f, id)
			pressure133ExplainBURange(t, f, id)
		})
	}
	restart133LongReceipts(t, f, b, savedSubmissions)
}

// Fixture diagnostics never log SQL arguments, credentials or business bodies.
type t07FixtureTrace struct{ t *testing.T }

func (tr t07FixtureTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return ctx
}
func (tr t07FixtureTrace) TraceQueryEnd(_ context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if data.Err != nil {
		var pg *pgconn.PgError
		if errors.As(data.Err, &pg) {
			tr.t.Logf("fixture query error SQLSTATE=%s message=%s", pg.Code, pg.Message)
		} else {
			tr.t.Logf("fixture query error type=%T closed or cancelled=%v", data.Err, errors.Is(data.Err, context.Canceled))
		}
	}
}
