//go:build integration

package bootstrap_test

import (
	person "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	foundation "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"
)

// Exercise both row identities with the same published primary and additional
// assignments as the ordinary business test. Expected positions come from the
// synthetic personnel numbers, independently of the HTTP result.
func measureMixedDirectory(t *testing.T, x assignmentFixture, first, next, bare string) {
	type M = map[string]any
	f := x.h.f
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer tx.Rollback(f.ctx)
	_, err = tx.Exec(f.ctx, `CREATE TEMP TABLE people510 ON COMMIT DROP AS SELECT i,'P'||lpad(i::text,8,'0') num,
 ('ac000000-0000-4000-8000-'||lpad(i::text,12,'0'))::uuid id,
 ('dc000000-0000-4000-8000-'||lpad(i::text,12,'0'))::uuid ver FROM generate_series(3,10000)i`)
	must(t, err)
	for _, stmt := range []string{
		`INSERT INTO personnel.number SELECT $1,num,id FROM people510`,
		`INSERT INTO personnel.person SELECT $1,id,num,ver,'2036-01-01','2037-01-01','local-retention-1','2036-01-01','2037-01-01' FROM people510`,
		`INSERT INTO personnel.source(instance_id,person_id,version,number,actor,operator_name,recorded_at,policy_version,action) SELECT $1,id,ver,num,(SELECT principal_id FROM access.principal WHERE instance_id=$1 AND login='admin'),'名录容量试件','2026-01-01','local-retention-1','register' FROM people510`,
		`INSERT INTO personnel.identity_payload SELECT $1,id,ver,'名录容量人员'||i FROM people510`,
		`UPDATE personnel.counter SET last_number=10000 WHERE instance_id=$1`,
	} {
		_, err = tx.Exec(f.ctx, stmt, f.instance)
		must(t, err)
	}
	must(t, tx.Commit(f.ctx))
	_, err = f.admin.Exec(f.ctx, "ANALYZE")
	must(t, err)
	path := "/api/v1/personnel/directory/query"
	clients := []*browser{x.h.b}
	for len(clients) < 20 {
		clients = append(clients, f.login("admin", f.password))
	}
	for _, sessions := range []int{1, 20} {
		for _, group := range []struct {
			label, anchor string
			position      int
			next          string
		}{
			{"混合名录", "", 0, ""}, {"任职锚点", first, 1, next}, {"人员锚点", bare, 4, "ac000000-0000-4000-8000-000000000003"},
		} {
			query := M{"page": 1, "pageSize": 20, "sort": "number"}
			if group.anchor != "" {
				query["anchor"] = group.anchor
			}
			run := func(client *browser) error {
				status, v, e := client.response("POST", path, maps.Clone(query), "")
				if e != nil {
					return e
				}
				if status != 200 || v["total"] != 10002.0 || v["people"] != 10000.0 || v["assignments"] != 3.0 {
					return fmt.Errorf("混合集合不符 HTTP %d: %v/%v/%v", status, v["total"], v["people"], v["assignments"])
				}
				rows := v["rows"].([]any)
				if len(rows) != 20 || rows[0].(M)["id"] != first || rows[3].(M)["id"] != bare {
					return fmt.Errorf("混合行顺序错误")
				}
				if group.anchor != "" {
					nav, ok := v["navigation"].(M)
					if !ok || nav["position"] != float64(group.position) {
						return fmt.Errorf("锚点位置错误")
					}
					dest := nav["next"].(M)
					if dest["person"] != group.next && dest["assignment"] != group.next {
						return fmt.Errorf("后项错误")
					}
					if group.anchor == bare && nav["previous"].(M)["person"] != x.person {
						return fmt.Errorf("人员行前项错误")
					}
				}
				return nil
			}
			for i := range 20 {
				must(t, run(clients[i%sessions]))
			}
			var wg sync.WaitGroup
			durations := make(chan time.Duration, 200)
			issues := make(chan error, 200)
			for i, client := range clients[:sessions] {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := i; j < 200; j += sessions {
						start := time.Now()
						err := run(client)
						durations <- time.Since(start)
						if err != nil {
							issues <- err
						}
					}
				}()
			}
			wg.Wait()
			close(durations)
			close(issues)
			for e := range issues {
				t.Fatal(e)
			}
			samples := make([]time.Duration, 0, 200)
			for d := range durations {
				samples = append(samples, d)
			}
			slices.Sort(samples)
			t.Logf("%s 会话=%d 样本=%d 错误=0 P50=%v P95=%v P99=%v", group.label, sessions, len(samples), samples[99], samples[189], samples[197])
			limit := 500 * time.Millisecond
			if sessions == 1 {
				limit = 300 * time.Millisecond
			}
			if samples[189] > limit {
				t.Errorf("%s P95 超过 %v", group.label, limit)
			}
		}
	}
}

// Expiry between candidate recognition and paging must not split the
// permission snapshot. The very next directory read must observe expiry.
func TestPersonnelDirectoryRetentionBoundary(t *testing.T) {
	x := setupAssignment(t)
	f := x.h.f
	creds := geoCredentials203(t, f, "admin", f.password)
	tx, err := f.admin.BeginTx(f.ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	must(t, err)
	defer tx.Rollback(f.ctx)
	var cutoff time.Time
	err = tx.QueryRow(f.ctx, `update personnel.person set ordinary_until=clock_timestamp()+interval '800 milliseconds' where instance_id=$1 and person_id=$2 returning ordinary_until`, f.instance, x.person).Scan(&cutoff)
	must(t, err)
	service := f.runtime.Personnel.(interface {
		OpenDirectory(context.Context, foundation.Credentials, person.Query) (person.DirectoryLookup, error)
	})
	query := person.Query{Page: 1, Size: 20}
	ctx := dbtx.With(f.ctx, tx)
	lookup, err := service.OpenDirectory(ctx, creds, query)
	must(t, err)
	recognized, _, err := lookup.Candidates([]string{x.person})
	must(t, err)
	if len(recognized) != 1 {
		t.Fatal("到期前无法识别人")
	}
	_, err = tx.Exec(ctx, `select pg_sleep(greatest(0,extract(epoch from ($1::timestamptz-clock_timestamp()))))`, cutoff)
	must(t, err)
	keys := []person.DirectoryKey{{Person: x.person, ID: x.hire["assignment"].(string)}}
	before, err := lookup.Page(keys, keys[0].ID)
	must(t, err)
	if before.Total != 1 || before.Nav.Position != 1 {
		t.Fatal("一次查询的行与导航发生变化", before)
	}
	fresh, err := service.OpenDirectory(ctx, creds, query)
	must(t, err)
	after, err := fresh.Page(keys, keys[0].ID)
	must(t, err)
	if after.Total != 0 || after.Nav.Position != 0 {
		t.Fatal("新查询泄露过期身份", after)
	}
}
