//go:build integration

package bootstrap

import (
	migrationfiles "chnhcm.local/chnhcm/db/migrations"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	app "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	geo "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestLocationAddressMigration(t *testing.T) {
	ctx := context.Background()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	image, e := os.ReadFile("../../deploy/local/postgres-image.txt")
	must(e)
	password := foundation.NewID() + foundation.NewID()
	container, e := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: tc.ContainerRequest{Image: strings.TrimSpace(string(image)), ExposedPorts: []string{"5432/tcp"}, Env: map[string]string{"POSTGRES_PASSWORD": password, "POSTGRES_DB": "chnhcm", "POSTGRES_INITDB_ARGS": "--encoding=UTF8 --locale=C"}, WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(time.Minute)}, Started: true})
	must(e)
	t.Cleanup(func() { must(container.Terminate(context.Background())) })
	host, e := container.Host(ctx)
	must(e)
	port, e := container.MappedPort(ctx, "5432/tcp")
	must(e)
	u := url.URL{Scheme: "postgres", User: url.UserPassword("postgres", password), Host: host + ":" + port.Port(), Path: "/chnhcm", RawQuery: "sslmode=disable"}
	pool, e := pgxpool.New(ctx, u.String())
	must(e)
	defer pool.Close()
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	old, e := goose.NewProvider(goose.DialectPostgres, db, migrationfiles.Files, goose.WithExcludeNames([]string{"00016_location_address.sql", "00017_staffing.sql", "00018_grade.sql", "00019_grade_ladder.sql", "00020_staffing_member_reference_checks.sql", "00021_position.sql", "00022_position_hierarchy.sql", "00023_geoarea_macau.sql", "00024_job_function.sql", "00025_job_function_adoption.sql", "00026_position_reporting.sql", "00027_position_quota.sql", "00028_staffing_quota_convergence.sql", "00029_persons.sql", "00030_person_primary_document.sql", "00031_person_profile.sql", "00032_hire_readiness.sql", "00033_hire_rule_receipts.sql", "00034_person_address_family.sql", "00035_first_hire_rule.sql", "00036_hire_identity.sql", "00037_employee_hire.sql", "00038_person_education_credentials.sql", "00039_appointment_decision.sql", "00040_person_bank_health.sql", "00041_employment_record.sql", "00042_job_information_sources.sql", "00043_employment_register.sql", "00044_assignment_history.sql"}))
	must(e)
	_, e = old.Up(ctx)
	must(e)
	instance, enterprise, object, subject, record := foundation.NewID(), foundation.NewID(), foundation.NewID(), foundation.NewID(), foundation.NewID()
	_, e = pool.Exec(ctx, `INSERT INTO organization.enterprise VALUES($1,$2,'MERGE','地址合并验证',daterange('1900-01-01',NULL,'[)'),decode(repeat('00',32),'hex'));`, instance, enterprise)
	must(e)
	_, e = pool.Exec(ctx, `INSERT INTO access.principal(initialization_digest,instance_id,principal_id,login,normalized_login,state) VALUES(decode(repeat('00',32),'hex'),$1,$2,'merge','merge','active');`, instance, subject)
	must(e)
	_, e = pool.Exec(ctx, `INSERT INTO access.policy_guard VALUES($1,$2,1)`, instance, subject)
	must(e)
	for i, fields := range [][]string{{"addressLine", "addressExtra"}, {"addressLine"}, {"addressExtra"}, {"name"}} {
		action := []string{"location.read", "location.history.read", "location.export", "location.create"}[i]
		_, e = pool.Exec(ctx, `INSERT INTO access.subject_grant VALUES($1,$2,$3,'enterprise',$4,$5,$6,$6)`, instance, foundation.NewID(), subject, enterprise, action, fields)
		must(e)
	}
	priorExtras = map[string]string{}
	defer func() { priorExtras = nil }()
	var expectedStates []model.LocationState
	var expectedOps []model.LocationOperation
	var chunk history.Chunk
	var previous *d.Record
	previousSource := ""
	for n := 1; n <= 70; n++ {
		line := fmt.Sprintf("某路%d号", n)
		if n == 70 {
			line = strings.Repeat("址", 500)
		}
		if n%3 != 0 {
			priorExtras[line] = fmt.Sprintf("第%d层", n)
		}
		if n == 70 {
			priorExtras[line] = strings.Repeat("室", 500)
		}
		source, sid := foundation.NewID(), foundation.NewID()
		r := d.Record{ID: record, Revision: foundation.NewID(), RevisionNo: int64(n), Date: "2020-01-01", Type: "CREATE", Reason: "NEW_LOCATION", TypeLabel: "新建", ReasonLabel: "新设地点", State: "active", Data: d.Data{Name: "地点", CountryID: foundation.NewID(), AddressLine: line, Adoption: geo.Adoption{Date: "2020-01-01"}}}
		r.Data.Adoption.Country = geo.Reference{ObjectID: r.Data.CountryID, SourceVersion: foundation.NewID(), RecordID: foundation.NewID(), Revision: foundation.NewID()}
		state := model.LocationState{State: app.State{Timeline: d.Timeline{ID: object, Enterprise: enterprise, Code: "MERGE", First: record, Source: source, SourceNo: int64(n), Records: []d.Record{r}}, Summaries: []org.OperationSummary{{RecordID: record, Revision: r.Revision, OperationSourceVersion: source}}}}
		op := model.LocationOperation{Operation: app.Operation{ID: source, Previous: previousSource, Subject: subject, Submission: sid, Action: "CORRECT", EventID: foundation.NewID(), Number: int64(n), RecordedAt: time.Date(2026, 11, 15, 0, n, 0, 0, time.UTC), Revisions: []app.Revision{{Before: previous, After: &r}}, Terminal: app.Terminal{ID: sid, ObjectID: object, Action: "CORRECT", Outcome: "accepted", Digest: []byte{1, 2, 3}, Result: org.LocationResult{SubmissionID: sid, Outcome: "accepted", LocationID: &object, SourceVersion: &source}}}}
		tx, e := pool.Begin(ctx)
		must(e)
		_, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(organization.submission_key('location',$1,$2,$3))`, instance, subject, sid)
		must(e)
		if n == 1 {
			_, e = tx.Exec(ctx, `INSERT INTO organization.location(instance_id,enterprise_id,location_id,code,normalized_code,first_record_id,head_source_id,head_no,head_chunk) VALUES($1,$2,$3,'MERGE','merge',$4,$5,1,1)`, instance, enterprise, object, record, source)
			must(e)
			chunk = history.Chunk{Kind: history.Location, Instance: instance, Enterprise: enterprise, Object: object, Number: 1, First: 1}
		} else {
			_, e = tx.Exec(ctx, `UPDATE organization.location SET head_source_id=$3,head_no=$4 WHERE instance_id=$1 AND location_id=$2`, instance, object, source, n)
			must(e)
		}
		chunk, e = history.Append(ctx, tx, chunk, int64(n), source, subject, sid, func(p *compact.Pool) (a, b, c []byte, err error) {
			a, err = encodePriorLocationState(ctx, p, state)
			if err != nil {
				return
			}
			b, err = encodePriorLocationOperation(ctx, p, op)
			if err != nil {
				return
			}
			c, err = history.EncodeLocationEvidence(ctx, p, model.LocationEvidence{ObjectID: object})
			return
		})
		must(e)
		_, e = tx.Exec(ctx, `UPDATE organization.location SET head_chunk=$3 WHERE instance_id=$1 AND location_id=$2`, instance, object, chunk.Number)
		must(e)
		must(history.SaveProjection(ctx, tx, chunk, source, func(p *compact.Pool) ([]byte, error) { return encodePriorLocationState(ctx, p, state) }))
		must(tx.Commit(ctx))
		// Expected current-model facts differ only in the combined address text.
		state.State.Timeline.Records = append([]d.Record(nil), state.State.Timeline.Records...)
		state.State.Timeline.Records[0].Data.AddressLine = mergeAddress(line, priorExtras[line])
		copyAfter := r
		copyAfter.Data.AddressLine = mergeAddress(line, priorExtras[line])
		op.Operation.Revisions[0].After = &copyAfter
		if previous != nil {
			before := *previous
			before.Data.AddressLine = mergeAddress(before.Data.AddressLine, priorExtras[before.Data.AddressLine])
			op.Operation.Revisions[0].Before = &before
		}
		expectedStates = append(expectedStates, state)
		expectedOps = append(expectedOps, op)
		previous = &r
		previousSource = source
	}
	_, e = pool.Exec(ctx, `INSERT INTO organization.location_read VALUES($1,$2,$3,$4,daterange('2020-01-01',NULL,'[)'),'地点',NULL,'active','2020-01-01',$5,NULL,$6,$7,'00100')`, instance, enterprise, object, record, previous.Data.CountryID, previous.Data.AddressLine, priorExtras[previous.Data.AddressLine])
	must(e)
	// A conversion failure must restore every input, trigger and ledger entry.
	fingerprint := func() string {
		var v string
		must(pool.QueryRow(ctx, `SELECT md5(string_agg(to_jsonb(h)::text,'' ORDER BY chunk_no)) FROM organization.location_history h`).Scan(&v))
		return v
	}
	before := fingerprint()
	_, e = pool.Exec(ctx, `CREATE FUNCTION public.reject_address_merge() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected migration failure'; END $$; CREATE TRIGGER reject_merge BEFORE UPDATE ON organization.location_read FOR EACH ROW EXECUTE FUNCTION public.reject_address_merge()`)
	must(e)
	current, e := goose.NewProvider(goose.DialectPostgres, db, migrationfiles.Files, goose.WithExcludeNames([]string{"00016_location_address.sql"}), goose.WithGoMigrations(locationAddressMigration()))
	must(e)
	if _, e = current.Up(ctx); e == nil || !strings.Contains(e.Error(), "injected migration failure") {
		t.Fatalf("expected injected conversion failure, got %v", e)
	}
	if fingerprint() != before {
		t.Fatal("failed conversion mutated original history")
	}
	var count int
	must(pool.QueryRow(ctx, `SELECT count(*) FROM goose_db_version WHERE version_id=16 AND is_applied`).Scan(&count))
	if count != 0 {
		t.Fatal("failure recorded migration")
	}
	must(pool.QueryRow(ctx, `SELECT count(*) FROM pg_trigger WHERE tgrelid='organization.location_history'::regclass AND NOT tgisinternal AND tgenabled='D'`).Scan(&count))
	if count != 0 {
		t.Fatal("failure disabled guards")
	}
	_, e = pool.Exec(ctx, `DROP TRIGGER reject_merge ON organization.location_read;DROP FUNCTION public.reject_address_merge()`)
	must(e)
	_, e = current.Up(ctx)
	must(e)
	verify := func() {
		tx, e := pool.Begin(ctx)
		must(e)
		defer tx.Rollback(ctx)
		for i, want := range expectedStates {
			no, e := history.Locate(ctx, tx, history.Location, instance, object, want.State.Timeline.Source)
			must(e)
			c, e := history.LoadChunk(ctx, tx, history.Location, instance, object, no)
			must(e)
			slot := c.Index(want.State.Timeline.Source)
			b, e := history.LoadShape(ctx, tx, history.Location, instance, enterprise, c.States[slot])
			must(e)
			got, e := history.DecodeLocationState(ctx, c.Values, b)
			must(e)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("source %d changed", i+1)
			}
			b, e = history.LoadShape(ctx, tx, history.Location, instance, enterprise, c.Operations[slot])
			must(e)
			op, e := history.DecodeLocationOperation(ctx, c.Values, b)
			must(e)
			if !reflect.DeepEqual(op, expectedOps[i]) {
				t.Fatalf("operation/receipt %d changed", i+1)
			}
		}
	}
	verify()
	var line string
	must(pool.QueryRow(ctx, `SELECT address_line FROM organization.location_read`).Scan(&line))
	if line != expectedStates[69].State.Timeline.Records[0].Data.AddressLine || len([]rune(line)) != 1001 {
		t.Fatal("merged read address truncated")
	}
	for i, action := range []string{"location.read", "location.history.read", "location.export", "location.create"} {
		var b []byte
		must(pool.QueryRow(ctx, `SELECT to_jsonb(read_fields) FROM access.subject_grant WHERE action=$1`, action).Scan(&b))
		var fields []string
		must(json.Unmarshal(b, &fields))
		want := [][]string{{"addressLine"}, {}, {}, {"name"}}[i]
		if !reflect.DeepEqual(fields, want) {
			t.Fatal("grant widened", action, fields)
		}
	}
	must(pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema='organization' AND table_name='location_read' AND column_name='address_extra'`).Scan(&count))
	if count != 0 {
		t.Fatal("redundant storage column remains")
	}
	after := fingerprint()
	_, e = current.Up(ctx)
	must(e)
	if fingerprint() != after {
		t.Fatal("rerun rewrote history")
	}
	// Reconnect and recover every original source/receipt from committed storage.
	db.Close()
	pool.Reset()
	verify()
	t.Log("70 original sources and operation receipts, 1001-rune addresses, field-right intersection, atomic rollback and repeated migration verified")
}
