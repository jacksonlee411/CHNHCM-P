//go:build integration

package storagecase130

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Forcing deferred checks must neither commit a rejected row nor turn a failed
// COMMIT into success. Both paths use pgx's real transaction implementation.
func TestDiagnosticConstraintSplitPreservesCommitFailure(t *testing.T) {
	ctx := context.Background()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	raw, err := os.ReadFile("../../../deploy/local/postgres-image.txt")
	if err != nil {
		t.Fatal(err)
	}
	c, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: tc.ContainerRequest{
		Image: strings.TrimSpace(string(raw)), ExposedPorts: []string{"5432/tcp"},
		Env:        map[string]string{"POSTGRES_HOST_AUTH_METHOD": "trust"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(time.Minute),
	}, Started: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Terminate(context.Background()); err != nil {
			t.Error(err)
		}
	})
	host, err := c.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}
	u := url.URL{Scheme: "postgres", User: url.User("postgres"), Host: host + ":" + port.Port(), Path: "postgres", RawQuery: "sslmode=disable"}
	config, err := pgx.ParseConfig(u.String())
	if err != nil {
		t.Fatal(err)
	}
	config.Tracer = sqlStages{splitConstraints: true}
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	if _, err = conn.Exec(ctx, `CREATE TABLE parent(id int PRIMARY KEY); CREATE TABLE child(id int REFERENCES parent(id) DEFERRABLE INITIALLY DEFERRED); INSERT INTO parent VALUES(1)`); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{2, 1} {
		tx, e := conn.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec(ctx, `INSERT INTO child VALUES($1)`, id); e != nil {
			t.Fatal(e)
		}
		e = tx.Commit(ctx)
		if id == 2 && !errors.Is(e, pgx.ErrTxCommitRollback) {
			t.Fatalf("rejected row returned %v", e)
		}
		if id == 1 && e != nil {
			t.Fatal(e)
		}
	}
	var n int
	if err = conn.QueryRow(ctx, `SELECT count(*) FROM child WHERE id=1`).Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if err = conn.QueryRow(ctx, `SELECT count(*) FROM child WHERE id=2`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
}
