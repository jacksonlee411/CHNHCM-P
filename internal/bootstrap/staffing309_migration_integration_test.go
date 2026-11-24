//go:build integration

package bootstrap_test

import (
	migrationfiles "chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Build the actual 00001–00017 schema, then verify the explicit upgrade. No
// candidate downgrade or mutation of a delivered migration is used as setup.
func TestStaffing309MigrationPreservesPriorFacts(t *testing.T) {
	staffingMigrationPreservesPriorFacts(t, 17)
}
func TestStaffing310MigrationPreservesPriorFacts(t *testing.T) {
	staffingMigrationPreservesPriorFacts(t, 18)
}
func TestStaffing312MigrationPreservesPriorFacts(t *testing.T) {
	staffingMigrationPreservesPriorFacts(t, 20)
}
func TestStaffing312HierarchyMigration(t *testing.T) {
	staffingMigrationPreservesPriorFacts(t, 21)
}
func newStaffingMigrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
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
	t.Cleanup(pool.Close)
	return pool
}

func staffingMigrationPreservesPriorFacts(t *testing.T, prior int) {
	ctx := context.Background()
	pool := newStaffingMigrationPool(t)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	exclude := []string{"00019_grade_ladder.sql", "00020_staffing_member_reference_checks.sql", "00021_position.sql", "00022_position_hierarchy.sql", "00023_geoarea_macau.sql", "00024_job_function.sql", "00025_job_function_adoption.sql", "00026_position_reporting.sql", "00027_position_quota.sql", "00028_staffing_quota_convergence.sql", "00029_persons.sql", "00030_person_primary_document.sql", "00031_person_profile.sql", "00032_hire_readiness.sql", "00033_hire_rule_receipts.sql", "00034_person_address_family.sql", "00035_first_hire_rule.sql", "00036_hire_identity.sql", "00037_employee_hire.sql", "00038_person_education_credentials.sql", "00039_appointment_decision.sql", "00040_person_bank_health.sql", "00041_employment_record.sql", "00042_job_information_sources.sql", "00043_employment_register.sql", "00044_assignment_history.sql","00045_parallel_assignments.sql","00046_relationship_probation.sql","00047_assignment_promotion.sql"}
	if prior == 17 {
		exclude = append(exclude, "00018_grade.sql")
	}
	old, e := goose.NewProvider(goose.DialectPostgres, db, migrationfiles.Files, goose.WithExcludeNames(exclude))
	must(e)
	_, e = old.Up(ctx)
	must(e)

	instance, enterprise, subject := foundation.NewID(), foundation.NewID(), foundation.NewID()
	var family d.Timeline
	for _, table := range []string{"job_family", "job"} {
		kind, reason := d.Family, "NEW_JOB_FAMILY"
		changes := map[string]*string{"name": name307("原名称"), "description": name307("原值\n第二行")}
		if table == "job" {
			kind, reason = d.Job, "NEW_JOB"
			changes = map[string]*string{"name": name307("原职务名称"), "jobFamilyId": &family.ID, "responsibilities": name307("原职责")}
		}
		data := map[string]string{}
		for field, value := range changes {
			data[field] = *value
		}
		plan := d.Plan{After: d.Timeline{Kind: kind, Enterprise: enterprise, ID: foundation.NewID(), Code: "PRIOR309", Number: 1, Source: foundation.NewID(), Records: []d.Record{{ID: foundation.NewID(), Revision: foundation.NewID(), Date: "2000-01-01", Type: "CREATE", Reason: reason, ReasonLabel: d.Reasons(kind, "CREATE")[reason], Data: data}}}}
		if kind == d.Job {
			fr := family.Records[0]
			plan.After.Records[0].Adoption = d.Adoption{FamilyID: family.ID, Date: "2000-01-01", Source: family.Source, RecordID: fr.ID, Revision: fr.Revision}
			d.Finalize(&plan)
		}
		timeline := plan.After
		record := timeline.Records[0]
		if kind == d.Family {
			family = timeline
		}
		id, source, key := timeline.ID, timeline.Source, foundation.NewID()
		snapshot, e := json.Marshal(timeline)
		must(e)
		fact, e := json.Marshal(record)
		must(e)
		result, e := json.Marshal(staff.Result{Kind: staff.Kind(kind), SubmissionID: key, Outcome: "accepted", ObjectID: id, SourceVersion: source, SelectedRecordID: record.ID, SelectedRecordRevision: record.Revision, Action: "CREATE"})
		must(e)
		operation, e := json.Marshal(staff.Command{Kind: staff.Kind(kind), SubmissionID: key, Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: "PRIOR309", Name: changes["name"], Description: changes["description"], JobFamilyID: changes["jobFamilyId"], Responsibilities: changes["responsibilities"], EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: reason}})
		must(e)
		event, e := json.Marshal(map[string]any{"type": "staffing." + string(kind) + ".changed", "objectId": id, "sourceVersion": source, "sourceNumber": 1, "action": "CREATE", "submissionId": key})
		must(e)
		tx, e := pool.Begin(ctx)
		must(e)
		_, e = tx.Exec(ctx, "INSERT INTO staffing."+table+"(instance_id,enterprise_id,id,code,source_id,source_no,first_date,head) VALUES($1,$2,$3,'PRIOR309',$4,1,'2000-01-01',$5)", instance, enterprise, id, source, snapshot)
		must(e)
		_, e = tx.Exec(ctx, "INSERT INTO staffing."+table+`_source(instance_id,object_id,source_id,source_no,subject_id,submission_id,digest,snapshot,operation,operator) VALUES($1,$2,$3,1,$4,$5,decode(repeat('00',32),'hex'),$6,$7,'{"displayName":"原操作者"}')`, instance, id, source, subject, key, snapshot, operation)
		must(e)
		_, e = tx.Exec(ctx, "INSERT INTO staffing."+table+"_revision VALUES($1,$2,$3,$4,$5,$6)", instance, id, record.ID, record.Revision, source, fact)
		must(e)
		_, e = tx.Exec(ctx, "INSERT INTO staffing."+table+"_terminal(instance_id,subject_id,submission_id,digest,result) VALUES($1,$2,$3,decode(repeat('00',32),'hex'),$4)", instance, subject, key, result)
		must(e)
		_, e = tx.Exec(ctx, "INSERT INTO staffing."+table+"_outbox VALUES($1,$2,$3,$4)", instance, id, source, event)
		must(e)
		var parent any
		if kind == d.Job {
			parent = family.ID
		}
		_, e = tx.Exec(ctx, "INSERT INTO staffing."+table+"_read VALUES($1,$2,$3,$4,$5,'2000-01-01',NULL,true,$6,$7)", instance, enterprise, id, record.ID, source, parent, fact)
		must(e)
		must(tx.Commit(ctx))
	}

	if prior >= 20 {
		exclude := []string{"00022_position_hierarchy.sql", "00023_geoarea_macau.sql", "00024_job_function.sql", "00025_job_function_adoption.sql", "00026_position_reporting.sql", "00027_position_quota.sql", "00028_staffing_quota_convergence.sql", "00029_persons.sql", "00030_person_primary_document.sql", "00031_person_profile.sql", "00032_hire_readiness.sql", "00033_hire_rule_receipts.sql", "00034_person_address_family.sql", "00035_first_hire_rule.sql", "00036_hire_identity.sql", "00037_employee_hire.sql", "00038_person_education_credentials.sql", "00039_appointment_decision.sql", "00040_person_bank_health.sql", "00041_employment_record.sql", "00042_job_information_sources.sql", "00043_employment_register.sql", "00044_assignment_history.sql","00045_parallel_assignments.sql","00046_relationship_probation.sql","00047_assignment_promotion.sql"}
		if prior == 20 {
			exclude = append(exclude, "00021_position.sql")
		}
		g2, err := goose.NewProvider(goose.DialectPostgres, db, migrationfiles.Files, goose.WithExcludeNames(exclude))
		must(err)
		_, err = g2.Up(ctx)
		must(err)
	}
	if prior == 21 {
		key := foundation.NewID()
		result, err := json.Marshal(staff.Result{Kind: staff.Position, SubmissionID: key, Outcome: "not_applied"})
		must(err)
		_, err = pool.Exec(ctx, "INSERT INTO staffing.position_terminal(instance_id,subject_id,submission_id,result) VALUES($1,$2,$3,$4)", instance, subject, key, result)
		must(err)
	}
	facts := func() map[string]string {
		out := map[string]string{}
		tables := []string{"job_family", "job"}
		if prior == 21 {
			tables = append(tables, "position")
		}
		for _, table := range tables {
			for _, suffix := range []string{"", "_source", "_revision", "_terminal", "_outbox", "_read", "_selection"} {
				var hash string
				name := "staffing." + table + suffix
				row := "row_to_json(v)::text"
				if name == "staffing.job_read" {
					// 00025 adds only the provenance pointer to the derived read row.
					// Compare every pre-existing field and check the pointer separately.
					row = "jsonb_set(to_jsonb(v),'{fact}',v.fact-'functionEvidence')::text"
				}
				must(pool.QueryRow(ctx, "SELECT md5(COALESCE(string_agg("+row+",E'\\n' ORDER BY "+row+"),'')) FROM "+name+" v").Scan(&hash))
				out[name] = hash
			}
		}
		if prior >= 20 {
			for _, table := range []string{"job_grade_manifest", "job_grade_member", "grade_ladder_grade_manifest", "grade_ladder_grade_member"} {
				var digest string
				must(pool.QueryRow(ctx, "SELECT md5(coalesce(string_agg(to_jsonb(row)::text,'' ORDER BY to_jsonb(row)::text),'')) FROM staffing."+table+" row").Scan(&digest))
				out[table] = digest
			}
		}
		return out
	}
	before := facts()
	if bootstrap.CheckCurrentStorage(ctx, pool) == nil {
		t.Fatal("API admitted unupgraded ledger", prior)
	}
	must(bootstrap.Migrate(ctx, pool, migrationfiles.Files))
	must(bootstrap.CheckCurrentStorage(ctx, pool))
	must(bootstrap.Migrate(ctx, pool, migrationfiles.Files))
	if !reflect.DeepEqual(before, facts()) {
		t.Fatal("upgrade changed prior identities, histories, adoption, receipts or outbox")
	}
	var marked int
	must(pool.QueryRow(ctx, `SELECT count(*) FROM staffing.job_read r JOIN staffing.job_function_legacy l
        USING(instance_id,object_id,source_id,record_id) WHERE l.revision=(r.fact->>'revision')::uuid
        AND r.fact->'functionEvidence'=jsonb_build_object('sourceVersion',l.source_id,'recordId',l.record_id,'revision',l.revision)`).Scan(&marked))
	if marked != 1 {
		t.Fatal("upgraded read missing exact original evidence")
	}
	var seals int
	must(pool.QueryRow(ctx, "SELECT count(*) FROM staffing.job_grade_manifest WHERE member_count=0 AND digest=decode('4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945','hex')").Scan(&seals))
	if seals != 1 {
		t.Fatal("old Job source missing explicit zero-member seal")
	}
	var count int
	must(pool.QueryRow(ctx, "SELECT count(*) FROM staffing.grade").Scan(&count))
	if count != 0 {
		t.Fatal("upgrade fabricated Grade identities")
	}
}

func TestReportingBranch24JobClassificationUpgrade(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	original, err := goose.NewProvider(goose.DialectPostgres, db, migrationfiles.Files)
	must(t, err)
	_, err = original.UpTo(ctx, 23)
	must(t, err)
	branch, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("testdata"))
	must(t, err)
	_, err = branch.ApplyVersion(ctx, 24, true)
	must(t, err)

	// Interrupt after creating the missing catalog, just before 25 commits.
	_, err = pool.Exec(ctx, "ALTER TABLE staffing.storage_contract ADD CONSTRAINT reject_classification_cutover CHECK (contract <> 'staffing-job-classification-1')")
	must(t, err)
	if err = bootstrap.Migrate(ctx, pool, migrationfiles.Files); err == nil {
		t.Fatal("expected classification cutover to roll back")
	}
	var originalLayout bool
	must(t, pool.QueryRow(ctx, `SELECT contract='staffing-reporting-1' AND to_regclass('staffing.job_function') IS NULL
		AND (SELECT max(version_id)=24 FROM public.goose_db_version) FROM staffing.storage_contract`).Scan(&originalLayout))
	if !originalLayout {
		t.Fatal("failed upgrade changed the reporting-only database")
	}
	_, err = pool.Exec(ctx, "ALTER TABLE staffing.storage_contract DROP CONSTRAINT reject_classification_cutover")
	must(t, err)
	must(t, bootstrap.Migrate(ctx, pool, migrationfiles.Files))
	must(t, bootstrap.Migrate(ctx, pool, migrationfiles.Files))
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
}

func TestReportingOnlyMigration(t *testing.T) {
	for _, scenario := range []string{"reporting", "functional", "mixed"} {
		t.Run(scenario, func(t *testing.T) {
			pool := newStaffingMigrationPool(t)
			ctx := t.Context()
			db := stdlib.OpenDBFromPool(pool)
			defer db.Close()
			old, err := goose.NewProvider(goose.DialectPostgres, db, migrationfiles.Files, goose.WithExcludeNames([]string{"00026_position_reporting.sql", "00027_position_quota.sql", "00028_staffing_quota_convergence.sql", "00029_persons.sql", "00030_person_primary_document.sql", "00031_person_profile.sql", "00032_hire_readiness.sql", "00033_hire_rule_receipts.sql", "00034_person_address_family.sql", "00035_first_hire_rule.sql", "00036_hire_identity.sql", "00037_employee_hire.sql", "00038_person_education_credentials.sql", "00039_appointment_decision.sql", "00040_person_bank_health.sql", "00041_employment_record.sql", "00042_job_information_sources.sql", "00043_employment_register.sql", "00044_assignment_history.sql","00045_parallel_assignments.sql","00046_relationship_probation.sql","00047_assignment_promotion.sql"}))
			must(t, err)
			_, err = old.Up(ctx)
			must(t, err)
			instance, enterprise, root, source, key := foundation.NewID(), foundation.NewID(), foundation.NewID(), foundation.NewID(), foundation.NewID()
			purpose := scenario
			if scenario == "mixed" {
				_, err = pool.Exec(ctx, "INSERT INTO staffing.position_hierarchy VALUES($1,$2,'reporting',$3)", instance, enterprise, foundation.NewID())
				must(t, err)
				purpose = "functional"
			}
			_, err = pool.Exec(ctx, "INSERT INTO staffing.position_hierarchy VALUES($1,$2,$3,$4)", instance, enterprise, purpose, root)
			must(t, err)
			_, err = pool.Exec(ctx, "INSERT INTO staffing.position_hierarchy_version VALUES($1,$2,$3,1,'{}',now(),$4)", instance, root, source, key)
			must(t, err)
			_, err = pool.Exec(ctx, "INSERT INTO staffing.position_hierarchy_outbox VALUES($1,$2,$3,jsonb_build_object('purpose',$4::text))", instance, root, source, purpose)
			must(t, err)
			snapshot := func() string {
				var facts string
				must(t, pool.QueryRow(ctx, `SELECT row_to_json(v)::text||e.event::text FROM staffing.position_hierarchy_version v JOIN staffing.position_hierarchy_outbox e USING(instance_id,hierarchy_id,source_id) WHERE v.source_id=$1`, source).Scan(&facts))
				return facts
			}
			before := snapshot()
			err = bootstrap.Migrate(ctx, pool, migrationfiles.Files)
			must(t, err)
			must(t, bootstrap.CheckCurrentStorage(ctx, pool))
			var roots, versions, events int
			must(t, pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM staffing.position_hierarchy),
				(SELECT count(*) FROM staffing.position_hierarchy_version),(SELECT count(*) FROM staffing.position_hierarchy_outbox)`).Scan(&roots, &versions, &events))
			wantRoots, wantVersions := 1, 0
			if scenario == "functional" {
				wantRoots = 0
			}
			if scenario == "reporting" {
				wantVersions = 1
				if before != snapshot() {
					t.Fatal("管理汇报的发布来源或事件变化")
				}
			}
			if roots != wantRoots || versions != wantVersions || events != wantVersions {
				t.Fatalf("剩余根、版本、发布事件 %d %d %d", roots, versions, events)
			}
			if roots == 0 {
				_, err = pool.Exec(ctx, "INSERT INTO staffing.position_hierarchy VALUES($1,$2,$3)", instance, enterprise, foundation.NewID())
				must(t, err)
			}
			_, err = pool.Exec(ctx, "INSERT INTO staffing.position_hierarchy VALUES($1,$2,$3)", instance, enterprise, foundation.NewID())
			if err == nil {
				t.Fatal("企业出现第二棵管理汇报树")
			}

		})
	}
}

func TestStaffing315AllocationPredecessor(t *testing.T) {
	pool := newStaffingMigrationPool(t)
	ctx := t.Context()
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	prefix, err := goose.NewProvider(goose.DialectPostgres, db, migrationfiles.Files, goose.WithExcludeNames([]string{"00026_position_reporting.sql", "00027_position_quota.sql", "00028_staffing_quota_convergence.sql", "00029_persons.sql", "00030_person_primary_document.sql", "00031_person_profile.sql", "00032_hire_readiness.sql", "00033_hire_rule_receipts.sql", "00034_person_address_family.sql", "00035_first_hire_rule.sql", "00036_hire_identity.sql", "00037_employee_hire.sql", "00038_person_education_credentials.sql", "00039_appointment_decision.sql", "00040_person_bank_health.sql", "00041_employment_record.sql", "00042_job_information_sources.sql", "00043_employment_register.sql", "00044_assignment_history.sql","00045_parallel_assignments.sql","00046_relationship_probation.sql","00047_assignment_promotion.sql"}))
	must(t, err)
	_, err = prefix.Up(ctx)
	must(t, err)
	raw, err := os.ReadFile("testdata/00026_staffing_quota.sql")
	must(t, err)
	up, _, found := strings.Cut(string(raw), "-- +goose Down")
	if !found {
		t.Fatal("缺少原 00026 SQL")
	}
	_, err = pool.Exec(ctx, up)
	must(t, err)
	_, err = pool.Exec(ctx, "INSERT INTO public.goose_db_version(version_id,is_applied) VALUES(26,true)")
	must(t, err)
	instance, enterprise, root := foundation.NewID(), foundation.NewID(), foundation.NewID()
	_, err = pool.Exec(ctx, "INSERT INTO staffing.position_hierarchy VALUES($1,$2,'functional',$3)", instance, enterprise, root)
	must(t, err)
	_, err = pool.Exec(ctx, `ALTER TABLE staffing.storage_contract ADD CONSTRAINT pilot_abort_28 CHECK(contract<>'staffing-unified-quota-1')`)
	must(t, err)
	if err = bootstrap.Migrate(ctx, pool, migrationfiles.Files); err == nil {
		t.Fatal("故障注入被忽略")
	}
	var intact bool
	must(t, pool.QueryRow(ctx, `SELECT contract='staffing-allocation-1' AND (SELECT max(version_id)=27 FROM public.goose_db_version)
		AND EXISTS(SELECT FROM staffing.position_hierarchy WHERE id=$1 AND purpose='functional')
		AND to_regclass('staffing.quota_control') IS NULL FROM staffing.storage_contract`, root).Scan(&intact))
	if !intact {
		t.Fatal("失败时未恢复原根和原合同")
	}
	_, err = pool.Exec(ctx, "ALTER TABLE staffing.storage_contract DROP CONSTRAINT pilot_abort_28")
	must(t, err)
	must(t, bootstrap.Migrate(ctx, pool, migrationfiles.Files))
	must(t, bootstrap.Migrate(ctx, pool, migrationfiles.Files))
	must(t, bootstrap.CheckCurrentStorage(ctx, pool))
	var count int
	must(t, pool.QueryRow(ctx, "SELECT count(*) FROM staffing.position_hierarchy").Scan(&count))
	if count != 0 {
		t.Fatal("仍有职能协作根")
	}
}
