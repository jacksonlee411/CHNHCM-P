//go:build integration

package bootstrap_test

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This is outside every timed request. Restart the fixture's actual database,
// reconnect to its current published port, and recover the original identities.
func restart133LongReceipts(t *testing.T, f *fixture, b *browser, submissions []string) {
	t.Helper()
	restartTimelineReceipts(t, f, b, submissions, "bu")
}

func restartTimelineReceipts(t *testing.T, f *fixture, b *browser, submissions []string, kind string) {
	t.Helper()
	recovery, objects, key := "organization", "business-units", "buId"
	if kind == "department" {
		recovery, objects, key = "departments", "departments", "departmentId"
	}
	if kind == "enterprise" {
		recovery, objects, key = "enterprise", "enterprise", "enterpriseId"
	}
	if kind == "location" {
		recovery, objects, key = "locations", "locations", "locationId"
	}
	if kind == "legalentity" {
		recovery, objects, key = "legal-entities", "legal-entities", "legalEntityId"
	}
	if kind == "geoarea" {
		recovery, objects, key = "geo-areas", "geo-areas", "geoAreaId"
	}
	if kind == "jobfamily" {
		recovery, objects, key = "job-families", "job-families", "objectId"
	}
	if kind == "job" {
		recovery, objects, key = "jobs", "jobs", "objectId"
	}
	if kind == "grade" {
		recovery, objects, key = "grades", "grades", "objectId"
	}
	if kind == "gradeladder" {
		recovery, objects, key = "grade-ladders", "grade-ladders", "objectId"
	}
	var staffBefore map[string]string
	if kind == "job" || kind == "jobfamily" || kind == "grade" || kind == "gradeladder" {
		staffBefore = staffingFacts307(t, f)
	}
	historyBefore := f.history()
	receipts := make([]map[string]any, len(submissions))
	for i, id := range submissions {
		receipts[i] = b.call("POST", "/api/v1/"+recovery+"/submissions/"+id+"/resolve", nil, "", 200)
	}
	before := logicalBusinessSnapshot(f)
	var locationBusiness map[string]string
	if kind == "location" {
		locationBusiness = locationBusinessSnapshot204(t, f)
	}
	restartFixtureDatabase(t, f)
	exactSources, rejected := 0, 0
	for i, id := range submissions {
		got := b.call("POST", "/api/v1/"+recovery+"/submissions/"+id+"/resolve", nil, "", 200)
		if !reflect.DeepEqual(receipts[i], got) {
			t.Fatal("restarted long-save original receipt changed", id)
		}
		if staffBefore != nil && (got["outcome"] == "rejected" || got["outcome"] == "not_applied") {
			if got["submissionId"] != id || got["sourceVersion"] != nil || got[key] != nil {
				t.Fatal("terminal without a business write returned an object or source", id)
			}
			if got["outcome"] == "rejected" {
				rejected++
				if got["failure"] == nil {
					t.Fatal("restarted rejection lost its failure", id)
				}
			}
			continue
		}
		source := got["sourceVersion"].(string)
		path := "/api/v1/" + objects + "/" + got[key].(string) + "?sourceVersion=" + source
		view := b.call("GET", path, nil, "", 200)
		if view["resolvedSourceVersion"] != source || view["readOnly"] != true {
			t.Fatal("restarted long-save exact source missing", id)
		}
		exactSources++
	}
	if kind == "location" {
		after := locationBusinessSnapshot204(t, f)
		for table, fingerprint := range locationBusiness {
			if after[table] != fingerprint {
				t.Errorf("restart changed business table organization.%s", table)
			}
		}
		if len(after) != len(locationBusiness) {
			t.Error("restart changed business table inventory")
		}
	}
	if !reflect.DeepEqual(before, logicalBusinessSnapshot(f)) || !reflect.DeepEqual(historyBefore, f.history()) {
		t.Fatal("restarted recovery changed original history, heads or terminals")
	}
	if staffBefore != nil && !reflect.DeepEqual(staffBefore, staffingFacts307(t, f)) {
		t.Fatal("restart changed staffing identity/source/revision/receipt/outbox")
	}
	t.Logf("LONG_SAVE_RESTART recovered=%d exactSources=%d rejected=%d allOriginalHistoryUnchanged=true newSources=0 newTerminals=0", len(submissions), exactSources, rejected)
}

// Read-only content fingerprints cover every business table, including all
// Location, country and area histories. Startup intentionally invalidates
// temporary selection headers and their cascading member blocks.
func locationBusinessSnapshot204(t *testing.T, f *fixture) map[string]string {
	t.Helper()
	rows, e := f.admin.Query(f.ctx, `SELECT tablename FROM pg_tables WHERE schemaname='organization' AND tablename NOT IN ('department_selection_draft','department_selection_block','location_selection_draft','location_selection_block','legal_entity_selection_draft','legal_entity_selection_block') ORDER BY tablename`)
	must(t, e)
	tables := []string{}
	for rows.Next() {
		var table string
		must(t, rows.Scan(&table))
		tables = append(tables, table)
	}
	must(t, rows.Err())
	rows.Close()
	out := map[string]string{}
	for _, table := range tables {
		rows, e = f.admin.Query(f.ctx, `SELECT md5(to_jsonb(t)::text) FROM organization.`+table+` t ORDER BY 1`)
		must(t, e)
		hash := sha256.New()
		for rows.Next() {
			var value string
			must(t, rows.Scan(&value))
			hash.Write([]byte(value))
		}
		must(t, rows.Err())
		rows.Close()
		out[table] = fmt.Sprintf("%x", hash.Sum(nil))
	}
	return out
}

// Restart only the disposable fixture database and API, preserving its volume and sessions.
func restartFixtureDatabase(t *testing.T, f *fixture) {
	t.Helper()
	adminConfig, apiConfig := f.admin.Config(), f.runtime.Pool.Config()
	f.server.Close()
	f.runtime.Close()
	f.admin.Close()
	stopTimeout := 30 * time.Second
	must(t, f.container.Stop(f.ctx, &stopTimeout))
	must(t, f.container.Start(f.ctx))
	port, err := f.container.MappedPort(f.ctx, "5432/tcp")
	must(t, err)
	n, err := strconv.ParseUint(port.Port(), 10, 16)
	must(t, err)
	adminConfig.ConnConfig.Port, apiConfig.ConnConfig.Port = uint16(n), uint16(n)
	f.admin, err = pgxpool.NewWithConfig(f.ctx, adminConfig)
	must(t, err)
	t.Cleanup(f.admin.Close)
	deadline := time.Now().Add(20 * time.Second)
	for f.admin.Ping(f.ctx) != nil {
		if time.Now().After(deadline) {
			t.Fatal("restarted fixture database did not become ready")
		}
		time.Sleep(20 * time.Millisecond)
	}
	dsn, err := url.Parse(apiConfig.ConnString())
	must(t, err)
	dsn.Host = net.JoinHostPort(apiConfig.ConnConfig.Host, port.Port())
	f.runtime, err = bootstrap.Open(f.ctx, dsn.String(), f.instance)
	must(t, err)
	t.Cleanup(f.runtime.Close)
	api := &bootstrap.API{Runtime: f.runtime, InsecureLocal: true}
	f.server = httptest.NewUnstartedServer(nil)
	api.Origin = "http://" + f.server.Listener.Addr().String()
	f.server.Config.Handler = api.Handler()
	f.server.Start()
	t.Cleanup(f.server.Close)
}
