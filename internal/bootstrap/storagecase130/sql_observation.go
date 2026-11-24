package storagecase130

import (
	"context"
	"os"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/bootstrap/capacitydiag"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type sqlStageKey struct{}
type sqlStages struct{ splitConstraints bool }
type sqlStage struct {
	name    string
	done    func()
	pid     uint32
	started int64
}

// Only fixed labels leave this tracer: never SQL text, arguments or errors.
func sqlStageName(sql string) string {
	sql = strings.ToLower(strings.TrimSpace(sql))
	verb := "other"
	for _, v := range []string{"select", "insert", "update", "delete", "copy", "commit", "rollback", "savepoint", "release", "set", "begin"} {
		if strings.HasPrefix(sql, v) {
			verb = v
			break
		}
	}
	for _, table := range []string{"department_read_segment", "bu_read_segment", "department_history", "bu_history", "department_shape", "bu_shape", "department_dictionary", "bu_dictionary", "department_terminal", "bu_terminal", "department_hierarchy_relation", "department", "business_unit", "enterprise"} {
		if strings.Contains(sql, "organization."+table) {
			return "sql." + verb + "." + table
		}
	}
	if strings.Contains(sql, "access.") {
		return "sql." + verb + ".access"
	}
	return "sql." + verb
}

func (o sqlStages) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	name := sqlStageName(data.SQL)
	// Explicit offline diagnosis only: force pending checks immediately before
	// COMMIT, in the same transaction. This adds a round trip and must never be
	// used for capacity acceptance. On failure PostgreSQL aborts the transaction;
	// the following COMMIT returns ErrTxCommitRollback to the ordinary caller.
	if o.splitConstraints && strings.EqualFold(strings.TrimSpace(data.SQL), "commit") {
		done := foundation.BeginStage(ctx, "transaction.deferred_checks")
		_, _ = conn.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE")
		done()
	}
	return context.WithValue(ctx, sqlStageKey{}, sqlStage{name, foundation.BeginStage(ctx, name), conn.PgConn().PID(), time.Now().UnixNano()})
}
func (sqlStages) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if stage, ok := ctx.Value(sqlStageKey{}).(sqlStage); ok {
		stage.done()
		capacitydiag.RecordSQLWindow(ctx, stage.pid, stage.started, time.Now().UnixNano())
		capacitydiag.RecordSQLRows(ctx, stage.name, data.CommandTag.RowsAffected())
	}
}
func (sqlStages) TraceCopyFromStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	name := sqlStageName("copy " + strings.Join(data.TableName, "."))
	return context.WithValue(ctx, sqlStageKey{}, sqlStage{name, foundation.BeginStage(ctx, name), conn.PgConn().PID(), time.Now().UnixNano()})
}
func (sqlStages) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	if stage, ok := ctx.Value(sqlStageKey{}).(sqlStage); ok {
		stage.done()
		capacitydiag.RecordSQLWindow(ctx, stage.pid, stage.started, time.Now().UnixNano())
		capacitydiag.RecordSQLRows(ctx, stage.name, data.CommandTag.RowsAffected())
	}
}

// Install on the one pool before serving or starting background cleanup. Never
// mutate adapters already shared with a running goroutine. Formal mode follows
// the ordinary constructor with no SQL observer.
func openObserved(ctx context.Context, dsn, instance string) (*bootstrap.Runtime, error) {
	if os.Getenv("CHNHCM_CAPACITY_DIAGNOSTICS") == "" {
		return bootstrap.Open(ctx, dsn, instance)
	}
	return bootstrap.OpenForDiagnostics(ctx, dsn, instance, sqlStages{splitConstraints: os.Getenv("CHNHCM_CAPACITY_SPLIT_CONSTRAINTS") == "1"})
}
