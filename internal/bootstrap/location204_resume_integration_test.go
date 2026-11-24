//go:build integration

package bootstrap_test

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
	tc "github.com/testcontainers/testcontainers-go"
)

// Resume only an explicitly identified testcontainers fixture whose real seed
// finished before the execution tool stopped its runner. No business row is
// rewritten: the capacity test independently rechecks every seed operation.
func resumeLocationCapacity204(t *testing.T) *fixture {
	t.Helper()
	id := os.Getenv("CHNHCM_LOCATION204_RESUME_CONTAINER")
	if len(id) != 64 {
		t.Fatal("resume requires the complete owned fixture container ID")
	}
	raw, e := exec.Command("docker", "inspect", id).Output()
	must(t, e)
	var inspected []struct {
		ID, Name string
		Config   struct {
			Image  string
			Env    []string
			Labels map[string]string
		}
	}
	must(t, json.Unmarshal(raw, &inspected))
	image, e := os.ReadFile("../../deploy/local/postgres-image.txt")
	must(t, e)
	if len(inspected) != 1 || inspected[0].ID != id || inspected[0].Config.Image != strings.TrimSpace(string(image)) || inspected[0].Config.Labels["org.testcontainers"] != "true" || inspected[0].Config.Labels["com.docker.compose.project"] != "" {
		t.Fatal("resume target is not the explicitly owned isolated fixture")
	}
	password := ""
	for _, v := range inspected[0].Config.Env {
		if strings.HasPrefix(v, "POSTGRES_PASSWORD=") {
			password = strings.TrimPrefix(v, "POSTGRES_PASSWORD=")
		}
	}
	if password == "" {
		t.Fatal("fixture database credential missing")
	}
	ctx := context.Background()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	c, e := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: tc.ContainerRequest{Image: strings.TrimSpace(string(image)), Name: strings.TrimPrefix(inspected[0].Name, "/"), ExposedPorts: []string{"5432/tcp"}}, Started: true, Reuse: true})
	must(t, e)
	if c.GetContainerID() != id {
		t.Fatal("resume changed container identity")
	}
	host, e := c.Host(ctx)
	must(t, e)
	port, e := c.MappedPort(ctx, "5432/tcp")
	must(t, e)
	dsn := func(user, pass string) string {
		return (&url.URL{Scheme: "postgres", User: url.UserPassword(user, pass), Host: net.JoinHostPort(host, port.Port()), Path: "/chnhcm", RawQuery: "sslmode=disable"}).String()
	}
	admin, e := pgxpool.New(ctx, dsn("postgres", password))
	must(t, e)
	f := &fixture{t: t, ctx: ctx, admin: admin, container: c, password: foundation.NewID(), readPassword: foundation.NewID()}
	if f.scalar("SELECT count(*) FROM organization.enterprise") != 1 || f.scalar("SELECT count(*) FROM organization.location") != 10001 || f.scalar("SELECT sum(cardinality(source_refs)) FROM organization.location_history") != 33666 || f.scalar("SELECT count(*) FROM organization.location_selection_draft") != 0 {
		t.Fatal("resume requires the exact unmeasured 10001/33666 seed")
	}
	must(t, admin.QueryRow(ctx, "SELECT instance_id::text FROM organization.enterprise").Scan(&f.instance))
	t.Cleanup(func() {
		if e := c.Terminate(context.Background()); e != nil {
			t.Error("resumed fixture cleanup failed")
		}
	})
	t.Cleanup(admin.Close)
	before := locationBusinessSnapshot204(t, f)
	apiPassword := foundation.NewID()
	must(t, bootstrap.ConfigureLocalRoles(ctx, admin, apiPassword))
	// The interrupted process held random fixture-only passwords in memory.
	// Replace only these two credentials, keeping principal IDs and grants;
	// actual HTTP logins still use the unchanged 600000-round password verifier.
	for _, account := range []struct{ login, password string }{{"admin", f.password}, {"reader", f.readPassword}} {
		salt := make([]byte, 16)
		_, e = rand.Read(salt)
		must(t, e)
		hash, e := pbkdf2.Key(sha256.New, account.password, salt, 600000, 32)
		must(t, e)
		changed, e := admin.Exec(ctx, `UPDATE access.credential c SET salt=$3,password_hash=$4,iterations=600000,updated_at=clock_timestamp() FROM access.principal p WHERE c.instance_id=$1 AND (p.instance_id,p.principal_id)=(c.instance_id,c.principal_id) AND p.login=$2`, f.instance, account.login, salt, hash)
		must(t, e)
		if changed.RowsAffected() != 1 {
			t.Fatal("resume fixture principal missing")
		}
	}
	_, e = admin.Exec(ctx, `UPDATE access.session SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND revoked_at IS NULL`, f.instance)
	must(t, e)
	_, e = admin.Exec(ctx, `UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1`, f.instance)
	must(t, e)
	f.runtime, e = bootstrap.Open(ctx, dsn("chnhcm_api", apiPassword), f.instance)
	must(t, e)
	t.Cleanup(f.runtime.Close)
	api := &bootstrap.API{Runtime: f.runtime, InsecureLocal: true}
	f.server = httptest.NewUnstartedServer(nil)
	api.Origin = "http://" + f.server.Listener.Addr().String()
	f.server.Config.Handler = api.Handler()
	f.server.Start()
	t.Cleanup(f.server.Close)
	must(t, f.runtime.Ready(ctx))
	after := locationBusinessSnapshot204(t, f)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("resuming changed prepared business input")
	}
	if path := os.Getenv("CHNHCM_LOCATION204_RESUME_PROOF"); path != "" {
		raw, e = json.MarshalIndent(map[string]any{"containerID": id, "businessTablesBefore": before, "businessTablesAfter": after, "businessUnchanged": true, "locations": 10001, "originalSources": 33666, "formalRequestsBeforeResume": 0, "fixtureCredentialsRenewed": true}, "", "  ")
		must(t, e)
		must(t, os.WriteFile(path, raw, 0600))
	}
	t.Log("resumed exact unmeasured 10001/33666 fixture; every organization business table unchanged; independently checking all source values before timing")
	return f
}
