//go:build integration

package bootstrap_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	orgdb "chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pressure133SQLKey struct{}
type pressure133SQLCall struct {
	query string
	start time.Time
}

// Preserve the original API pool and its prepared-plan history when a traced
// replacement pool does not reproduce an unobserved timing. This affects only
// this test's disposable database, and never logs bind parameters.
func pressure133NativeSQL(t *testing.T, f *fixture) {
	dir := os.Getenv("CHNHCM_T07_NATIVE_SQL_DIR")
	if dir == "" {
		return
	}
	if !filepath.IsAbs(dir) {
		t.Fatal("native SQL logs require an external private directory")
	}
	repo, err := filepath.Abs("../..")
	must(t, err)
	real, err := filepath.EvalSymlinks(dir)
	must(t, err)
	relative, err := filepath.Rel(repo, real)
	must(t, err)
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("native SQL logs must remain outside the checkout")
	}
	info, err := os.Stat(dir)
	must(t, err)
	if !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatal("native SQL logs require a 0700 directory")
	}
	for _, sql := range []string{"ALTER SYSTEM SET log_parameter_max_length='0'", "ALTER SYSTEM SET log_parameter_max_length_on_error='0'", "ALTER SYSTEM SET log_min_duration_statement='1ms'"} {
		f.exec(sql)
	}
	f.exec("SELECT pg_reload_conf()")
	t.Cleanup(func() {
		logs, e := f.container.Logs(f.ctx)
		must(t, e)
		defer logs.Close()
		file, e := os.OpenFile(filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "_")+".log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		must(t, e)
		_, e = io.Copy(file, logs)
		must(t, e)
		must(t, file.Close())
	})
}

func pressure133CPU(t *testing.T) func() {
	dir := os.Getenv("CHNHCM_T07_CPU_PROFILE_DIR")
	if dir == "" {
		return func() {}
	}
	file, err := os.Create(filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "_")+".pprof"))
	must(t, err)
	allocs := func(suffix string) {
		f, e := os.Create(filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "_")+suffix+".pprof"))
		must(t, e)
		must(t, pprof.Lookup("allocs").WriteTo(f, 0))
		must(t, f.Close())
	}
	allocs("-alloc-before")
	must(t, pprof.StartCPUProfile(file))
	return func() { pprof.StopCPUProfile(); must(t, file.Close()); allocs("-alloc-after") }
}

// Optional native UPDATE attribution, outside the measured HTTP request. The
// synthetic binding and all its row changes are rolled back in this owned DB.
func pressure133ExplainBUUpdate(t *testing.T, f *fixture, object string) {
	if os.Getenv("CHNHCM_T07_EXPLAIN_UPDATE") != "1" {
		return
	}
	tx, err := f.admin.Begin(f.ctx)
	must(t, err)
	defer func() { must(t, tx.Rollback(f.ctx)) }()
	var dictionary int32
	must(t, tx.QueryRow(f.ctx, `INSERT INTO organization.bu_dictionary(instance_id,enterprise_id,words,binaries,ranges)
 SELECT instance_id,enterprise_id,words||ARRAY['rolled back update diagnostic'],binaries,ranges FROM organization.bu_dictionary
 WHERE id=(SELECT dictionary_id FROM organization.bu_read_segment WHERE object_id=$1 LIMIT 1) RETURNING id`, object).Scan(&dictionary))
	rows, err := tx.Query(f.ctx, `EXPLAIN(ANALYZE,BUFFERS,WAL) UPDATE organization.bu_read_segment SET source_id=$3,dictionary_id=$4,ids=ids,envelope=envelope
 WHERE instance_id=$1 AND object_id=$2`, f.instance, object, foundation.NewID(), dictionary)
	must(t, err)
	defer rows.Close()
	for rows.Next() {
		var line string
		must(t, rows.Scan(&line))
		// Plan constants are fixture UUIDs only. No credential or business data.
		t.Logf("M2_UPDATE_PLAN %s", line)
	}
	must(t, rows.Err())
}

func pressure133ExplainBURange(t *testing.T, f *fixture, object string) {
	if os.Getenv("CHNHCM_T07_EXPLAIN_RANGE") != "1" {
		return
	}
	var slots []int32
	var spans []pgtype.Range[pgtype.Date]
	must(t, f.admin.QueryRow(f.ctx, `SELECT array_agg(slot ORDER BY slot),array_agg(span ORDER BY slot) FROM organization.bu_read_segment WHERE instance_id=$1 AND object_id=$2`, f.instance, object).Scan(&slots, &spans))
	for _, mode := range []string{"force_custom_plan", "force_generic_plan"} {
		tx, err := f.admin.Begin(f.ctx)
		must(t, err)
		_, err = tx.Exec(f.ctx, "SET LOCAL jit=off;SET LOCAL statement_timeout='10s';SET LOCAL plan_cache_mode="+mode)
		must(t, err)
		_, err = tx.Exec(f.ctx, `PREPARE range133(uuid,uuid,integer[],daterange[]) AS DELETE FROM organization.bu_read_segment r WHERE r.instance_id=$1 AND r.object_id=$2
 AND NOT EXISTS(SELECT FROM unnest($3::integer[],$4::daterange[]) v(slot,span) WHERE v.slot=r.slot AND v.span=r.span)`)
		must(t, err)
		var explain string
		must(t, tx.QueryRow(f.ctx, `SELECT format('EXPLAIN(ANALYZE,BUFFERS) EXECUTE range133(%L::uuid,%L::uuid,%L::integer[],%L::daterange[])',$1::uuid,$2::uuid,$3::integer[],$4::daterange[])`, f.instance, object, slots, spans).Scan(&explain))
		rows, err := tx.Query(f.ctx, explain)
		must(t, err)
		for rows.Next() {
			var line string
			must(t, rows.Scan(&line))
			t.Logf("M2_RANGE_PLAN mode=%s %s", mode, line)
		}
		must(t, rows.Err())
		rows.Close()
		_, err = tx.Exec(f.ctx, "DEALLOCATE range133")
		must(t, err)
		must(t, tx.Rollback(f.ctx))
	}
}

type pressure133SQLMetric struct {
	Count        int
	Rows         int64
	Milliseconds float64
}
type pressure133SQLTracer struct {
	mu      sync.Mutex
	active  bool
	metrics map[string]pressure133SQLMetric
}

func (o *pressure133SQLTracer) begin(ctx context.Context, query string) context.Context {
	o.mu.Lock()
	active := o.active
	o.mu.Unlock()
	if !active {
		return ctx
	}
	// Static SQL only, never arguments, DSNs or query results.
	query = strings.Join(strings.Fields(query), " ")
	if len(query) > 180 {
		query = query[:180]
	}
	return context.WithValue(ctx, pressure133SQLKey{}, pressure133SQLCall{query, time.Now()})
}
func (o *pressure133SQLTracer) end(ctx context.Context, rows int64) {
	c, ok := ctx.Value(pressure133SQLKey{}).(pressure133SQLCall)
	if !ok {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	v := o.metrics[c.query]
	v.Count++
	v.Rows += rows
	v.Milliseconds += float64(time.Since(c.start)) / 1e6
	o.metrics[c.query] = v
}
func (o *pressure133SQLTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	return o.begin(ctx, d.SQL)
}
func (o *pressure133SQLTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryEndData) {
	o.end(ctx, d.CommandTag.RowsAffected())
}
func (o *pressure133SQLTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceCopyFromStartData) context.Context {
	return o.begin(ctx, "COPY "+d.TableName.Sanitize())
}
func (o *pressure133SQLTracer) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, d pgx.TraceCopyFromEndData) {
	o.end(ctx, d.CommandTag.RowsAffected())
}

// Optional attribution for the existing ordinary HTTP long-line write. The
// unobserved capacity run retains its original admission, timers and pool size.
func pressure133SQL(t *testing.T, f *fixture) func() func() {
	if os.Getenv("CHNHCM_T07_SQL_OBSERVE") != "1" {
		return func() func() { return func() {} }
	}
	store := f.runtime.Organization.(application.Service).Store.(*orgdb.Store)
	previous := store.Pool
	observer := &pressure133SQLTracer{}
	cfg := previous.Config()
	cfg.ConnConfig.Tracer = observer
	pool, err := pgxpool.NewWithConfig(f.ctx, cfg)
	must(t, err)
	store.Pool = pool
	t.Cleanup(func() { store.Pool = previous; pool.Close() })
	return func() func() {
		observer.mu.Lock()
		observer.metrics = map[string]pressure133SQLMetric{}
		observer.active = true
		observer.mu.Unlock()
		return func() {
			observer.mu.Lock()
			defer observer.mu.Unlock()
			observer.active = false
			for query, v := range observer.metrics {
				t.Logf("M2_SQL count=%d rows=%d totalMS=%.3f query=%s", v.Count, v.Rows, v.Milliseconds, query)
			}
		}
	}
}
