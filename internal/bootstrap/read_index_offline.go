package bootstrap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RebuildCurrentReadIndexes is not part of server startup or a public command.
// Offline verification callers must first establish disposable ownership and
// stop their API. Existing persistent environments are never admitted here.
func RebuildCurrentReadIndexes(ctx context.Context, pool *pgxpool.Pool, instance string) (any, error) {
	started := time.Now()
	var clients int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND usename IN ('chnhcm_app','chnhcm_api')`).Scan(&clients); err != nil {
		return nil, err
	}
	if clients != 0 {
		return nil, fmt.Errorf("offline read-index rebuild requires the candidate API to be stopped")
	}
	counts := map[string]int{}
	for _, g := range []struct {
		kind             history.Kind
		table, key, name string
	}{{history.BusinessUnit, "business_unit", "bu_id", "BU"}, {history.Department, "department", "department_id", "department"}} {
		type owner struct {
			id, source string
			chunk      int32
		}
		owners := []owner{}
		rows, e := pool.Query(ctx, fmt.Sprintf(`SELECT %s::text,head_source_id::text,head_chunk FROM organization.%s WHERE instance_id=$1 ORDER BY %s`, g.key, g.table, g.key), instance)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			var o owner
			if e = rows.Scan(&o.id, &o.source, &o.chunk); e != nil {
				rows.Close()
				return nil, e
			}
			owners = append(owners, o)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		work, cancel := context.WithCancel(ctx)
		jobs := make(chan owner)
		var wg sync.WaitGroup
		var once sync.Once
		var failure error
		for worker := 0; worker < 8; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for o := range jobs {
					if work.Err() != nil {
						continue
					}
					err := func() error {
						tx, err := pool.Begin(work)
						if err != nil {
							return err
						}
						defer tx.Rollback(context.Background())
						var current string
						err = tx.QueryRow(work, fmt.Sprintf(`SELECT head_source_id::text FROM organization.%s WHERE instance_id=$1 AND %s=$2 FOR UPDATE`, g.table, g.key), instance, o.id).Scan(&current)
						if err != nil {
							return err
						}
						if current != o.source {
							return fmt.Errorf("offline candidate source changed during derivative rebuild")
						}
						if err = history.RebuildCurrentReadIndex(work, tx, g.kind, instance, o.id, o.source, o.chunk); err != nil {
							return err
						}
						return tx.Commit(work)
					}()
					if err != nil {
						once.Do(func() { failure = err; cancel() })
					}
				}
			}()
		}
		for _, o := range owners {
			if work.Err() != nil {
				break
			}
			jobs <- o
		}
		close(jobs)
		wg.Wait()
		cancel()
		if failure != nil {
			return nil, failure
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		counts[g.name] = len(owners)

	}
	return map[string]any{"objects": counts, "readRule": history.ReadRule, "elapsedSeconds": time.Since(started).Seconds(), "businessSourcesRewritten": 0, "scope": "offline task-owned derivative rebuild"}, nil
}
