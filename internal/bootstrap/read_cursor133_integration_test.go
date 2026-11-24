//go:build integration && remediation133

package bootstrap_test

import (
	"context"
	"testing"
	"time"

	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Protocol prototype only. Synthetic facts isolate snapshot/locking behavior;
// this does not claim that the production capture SQL has been implemented.
func TestRemediation133CursorSnapshotAndActivity(t *testing.T) {
	f := fresh(t)
	ctx := f.ctx
	_, e := f.admin.Exec(ctx, `CREATE SCHEMA probe133;
 CREATE TABLE probe133.fact(id integer PRIMARY KEY,parent integer,bu integer,source integer,dict integer);
 CREATE TABLE probe133.dictionary(id integer PRIMARY KEY,value text);
 CREATE TABLE probe133.policy(id integer PRIMARY KEY,version integer);
 CREATE TABLE probe133.activity(id integer PRIMARY KEY,n integer);
 INSERT INTO probe133.dictionary VALUES(1,'old');
 INSERT INTO probe133.fact SELECT i,1,1,1,1 FROM generate_series(1,45)i;
 INSERT INTO probe133.policy VALUES(1,1); INSERT INTO probe133.activity VALUES(1,0);`)
	must(t, e)
	tx, e := f.admin.Begin(ctx)
	must(t, e)
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, `SELECT version FROM probe133.policy WHERE id=1 FOR SHARE;
 DECLARE facts NO SCROLL CURSOR FOR SELECT f.id,f.parent,f.bu,f.source,d.value FROM probe133.fact f JOIN probe133.dictionary d ON d.id=f.dict ORDER BY f.id`)
	must(t, e)
	read := func(n int, statement string) {
		rows, e := tx.Query(ctx, statement)
		must(t, e)
		defer rows.Close()
		count := 0
		for rows.Next() {
			var id, parent, bu, source int
			var value string
			must(t, rows.Scan(&id, &parent, &bu, &source, &value))
			if parent != 1 || bu != 1 || source != 1 || value != "old" {
				t.Fatal("mixed snapshot", id)
			}
			count++
		}
		must(t, rows.Err())
		if count != n {
			t.Fatalf("rows=%d want=%d", count, n)
		}
	}
	read(20, "FETCH FORWARD 20 FROM facts")
	// A different request atomically reparents, transfers BU, changes the target
	// source and reclaims the old dictionary. The open cursor must remain whole.
	_, e = f.admin.Exec(ctx, `BEGIN;INSERT INTO probe133.dictionary VALUES(2,'new');UPDATE probe133.fact SET parent=2,bu=2,source=2,dict=2;DELETE FROM probe133.dictionary WHERE id=1;UPDATE probe133.activity SET n=n+1 WHERE id=1;COMMIT`)
	must(t, e)
	read(25, "FETCH FORWARD 30 FROM facts")
	revoke, e := f.admin.Begin(ctx)
	must(t, e)
	_, e = revoke.Exec(ctx, "SET LOCAL lock_timeout='20ms';UPDATE probe133.policy SET version=2 WHERE id=1")
	var pg *pgconn.PgError
	if !errors.As(e, &pg) || pg.Code != "55P03" {
		t.Fatalf("revocation bypassed guard: %v", e)
	}
	must(t, revoke.Rollback(ctx))
	_, e = tx.Exec(ctx, "CLOSE facts;UPDATE probe133.activity SET n=n+1 WHERE id=1")
	must(t, e)
	must(t, tx.Commit(ctx))
	var activity int
	must(t, f.admin.QueryRow(ctx, "SELECT n FROM probe133.activity WHERE id=1").Scan(&activity))
	if activity != 2 {
		t.Fatal("same-session activity lost", activity)
	}
	// The next request gets current facts and a fresh guard; there is no cross-page snapshot.
	var current int
	must(t, f.admin.QueryRow(ctx, "SELECT count(*) FROM probe133.fact f JOIN probe133.dictionary d ON d.id=f.dict WHERE source=2 AND parent=2 AND bu=2 AND value='new'").Scan(&current))
	if current != 45 {
		t.Fatal("next request did not refresh")
	}
	_, e = f.admin.Exec(ctx, "UPDATE probe133.policy SET version=2 WHERE id=1")
	must(t, e)
	// Cancellation releases both cursor and guard; a subsequent request remains usable.
	canceled, cancel := context.WithCancel(ctx)
	tx, e = f.admin.Begin(canceled)
	must(t, e)
	_, e = tx.Exec(canceled, "SELECT version FROM probe133.policy WHERE id=1 FOR SHARE;DECLARE canceled_facts CURSOR FOR SELECT * FROM probe133.fact")
	must(t, e)
	cancel()
	clean, done := context.WithTimeout(ctx, time.Second)
	defer done()
	_ = tx.Rollback(clean)
	must(t, f.admin.Ping(ctx))
	if f.admin.Stat().AcquiredConns() != 0 {
		t.Fatal("cursor connection retained")
	}
}

// READ COMMITTED is explicitly selected per transaction; global isolation does
// not change. Its multi-statement reads are not a snapshot substitute.
func TestRemediation133ReadCommittedIsNotMultiQuerySnapshot(t *testing.T) {
	f := fresh(t)
	_, e := f.admin.Exec(f.ctx, "CREATE TABLE probe133_value(n int);INSERT INTO probe133_value VALUES(1)")
	must(t, e)
	tx, e := f.admin.BeginTx(f.ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	must(t, e)
	defer tx.Rollback(f.ctx)
	var a, b int
	must(t, tx.QueryRow(f.ctx, "SELECT n FROM probe133_value").Scan(&a))
	_, e = f.admin.Exec(f.ctx, "UPDATE probe133_value SET n=2")
	must(t, e)
	must(t, tx.QueryRow(f.ctx, "SELECT n FROM probe133_value").Scan(&b))
	if a != 1 || b != 2 {
		t.Fatal("counterexample did not reproduce")
	}
}
