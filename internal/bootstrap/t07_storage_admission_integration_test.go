//go:build integration

package bootstrap_test

import (
	"fmt"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Only the two legacy upgrade tests capture the real empty public 00057 schema.
func restoreGrantPredecessor(f *fixture) {
	if f.scalar("SELECT count(*) FROM pg_class WHERE oid=to_regclass('access.bound_permission')") == 0 {
		return
	}
	f.exec(`DROP VIEW IF EXISTS access.subject_grant;
 CREATE TABLE access.subject_grant (LIKE fixture.prefix_grant INCLUDING ALL);
 INSERT INTO access.subject_grant SELECT instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields FROM access.bound_permission;
 DROP VIEW access.bound_permission;
 DROP TABLE access.grant_assignment,access.grant_revision,access.policy_version;
 DROP FUNCTION access.check_binding(),access.protect_assignment_terms(),access.protect_policy_version(),access.valid_permission(uuid,text,uuid,text,text[],text[]);
 DELETE FROM public.goose_db_version WHERE version_id=58`)
}

func TestT07ExactCurrentMigrationSetAdmission(t *testing.T) {
	f := freshWithSchema(t, "2020-01-01", true)
	b := f.login("admin", f.password)
	b.call("POST", "/api/v1/business-units", createBody("T07_ADMISSION", "2026-01-01"), foundation.NewID(), 201)
	before := businessSnapshot(t, f)
	dsn := f.runtime.Pool.Config().ConnConfig.ConnString()
	for _, tc := range []struct{ name, setup, restore string }{
		{"mutable_grant_revision", `ALTER TABLE access.grant_revision DISABLE TRIGGER immutable_revision`, `ALTER TABLE access.grant_revision ENABLE TRIGGER immutable_revision`},
		{"missing_f_reserved_versions", `DELETE FROM public.goose_db_version WHERE version_id IN (32,33)`, `INSERT INTO public.goose_db_version(version_id,is_applied) VALUES(32,true),(33,true)`},
		{"missing_person_profile_table", `ALTER TABLE personnel.document_payload RENAME TO t07_hidden_documents`, `ALTER TABLE personnel.t07_hidden_documents RENAME TO document_payload`},
		{"first_hire_rule_period", `ALTER TABLE staffing.hire_rule_source DROP CONSTRAINT hire_rule_period_unique`,
			`ALTER TABLE staffing.hire_rule_source ADD CONSTRAINT hire_rule_period_unique EXCLUDE USING gist(instance_id WITH =, relationship WITH =, period WITH &&)`},
		{"first_hire_rule_seal", `ALTER TABLE staffing.hire_rule_source DISABLE TRIGGER immutable_source`, `ALTER TABLE staffing.hire_rule_source ENABLE TRIGGER immutable_source`},
		{"readiness_receipt_seal", `ALTER TABLE organization.legal_purpose_terminal DISABLE TRIGGER immutable_terminal`, `ALTER TABLE organization.legal_purpose_terminal ENABLE TRIGGER immutable_terminal`},
		{"readiness_binding_seal", `ALTER TABLE organization.legal_purpose_member DISABLE TRIGGER sealed_member`, `ALTER TABLE organization.legal_purpose_member ENABLE TRIGGER sealed_member`},
		{"readiness_policy_event", `ALTER TABLE staffing.assignment_policy_outbox RENAME TO t07_hidden_policy_event`, `ALTER TABLE staffing.t07_hidden_policy_event RENAME TO assignment_policy_outbox`},
		{"staffing_contract", `UPDATE staffing.storage_contract SET contract='task-owned-mismatch'`, `UPDATE staffing.storage_contract SET contract='staffing-unified-quota-1'`},
		{"immutable_reference_check", `ALTER TABLE staffing.grade_source DISABLE TRIGGER immutable_source`, `ALTER TABLE staffing.grade_source ENABLE TRIGGER immutable_source`},
		{"quota_source", `ALTER TABLE staffing.quota_source DISABLE TRIGGER immutable_source`, `ALTER TABLE staffing.quota_source ENABLE TRIGGER immutable_source`},
		{"quota_control", `ALTER TABLE staffing.quota_control DISABLE TRIGGER immutable_control`, `ALTER TABLE staffing.quota_control ENABLE TRIGGER immutable_control`},
		{"quota_outbox", `ALTER TABLE staffing.quota_outbox DISABLE TRIGGER immutable_event`, `ALTER TABLE staffing.quota_outbox ENABLE TRIGGER immutable_event`},
		{"quota_projection", `ALTER TABLE staffing.quota_record RENAME TO t07_missing_quota_record`, `ALTER TABLE staffing.t07_missing_quota_record RENAME TO quota_record`},
		{"quota_terminal", `ALTER TABLE staffing.quota_terminal DISABLE TRIGGER immutable_terminal`, `ALTER TABLE staffing.quota_terminal ENABLE TRIGGER immutable_terminal`},
		{"member_reference_check", `ALTER TABLE staffing.job_grade_member DISABLE TRIGGER member_references`, `ALTER TABLE staffing.job_grade_member ENABLE TRIGGER member_references`},
		{"staffing_table", `ALTER TABLE staffing.job RENAME TO t07_hidden_job`, `ALTER TABLE staffing.t07_hidden_job RENAME TO job`},
		{"hierarchy_seal", `ALTER TABLE staffing.position_hierarchy_edge DISABLE TRIGGER sealed_version`, `ALTER TABLE staffing.position_hierarchy_edge ENABLE TRIGGER sealed_version`},
		{"reporting_enterprise_unique", `ALTER TABLE staffing.position_hierarchy DROP CONSTRAINT position_hierarchy_enterprise_unique;
		 CREATE INDEX position_hierarchy_enterprise_unique ON staffing.position_hierarchy(instance_id,enterprise_id)`,
			`DROP INDEX staffing.position_hierarchy_enterprise_unique;
			 ALTER TABLE staffing.position_hierarchy ADD CONSTRAINT position_hierarchy_enterprise_unique UNIQUE(instance_id,enterprise_id)`},
		{"legacy_job_evidence", `ALTER TABLE staffing.job_function_legacy DISABLE TRIGGER immutable_legacy`, `ALTER TABLE staffing.job_function_legacy ENABLE TRIGGER immutable_legacy`},
		{"function_revision", `ALTER TABLE staffing.job_function_revision DISABLE TRIGGER immutable_revision`, `ALTER TABLE staffing.job_function_revision ENABLE TRIGGER immutable_revision`},
		{"position_table", `ALTER TABLE staffing.position RENAME TO t07_hidden_position`, `ALTER TABLE staffing.t07_hidden_position RENAME TO position`},
		{"position_revision", `ALTER TABLE staffing.position_revision DISABLE TRIGGER immutable_revision`, `ALTER TABLE staffing.position_revision ENABLE TRIGGER immutable_revision`},
		{"missing_first", `DELETE FROM public.goose_db_version WHERE version_id=1`, `INSERT INTO public.goose_db_version(version_id,is_applied) VALUES(1,true)`},
		{"missing_second", `DELETE FROM public.goose_db_version WHERE version_id=2`, `INSERT INTO public.goose_db_version(version_id,is_applied) VALUES(2,true)`},
		{"geoarea_assigned_code", `ALTER INDEX organization.geoarea_assigned_code RENAME TO t07_hidden_code`, `ALTER INDEX organization.t07_hidden_code RENAME TO geoarea_assigned_code`},
		{"legacy_tree_sparse_set", `DELETE FROM public.goose_db_version WHERE version_id IN (9,11,12)`, `INSERT INTO public.goose_db_version(version_id,is_applied) VALUES(9,true),(11,true),(12,true)`},
		{"unknown_applied", `INSERT INTO public.goose_db_version(version_id,is_applied) VALUES(999,true)`, `DELETE FROM public.goose_db_version WHERE version_id=999`},
		{"latest_first_unapplied", `INSERT INTO public.goose_db_version(version_id,is_applied) VALUES(1,false)`, `DELETE FROM public.goose_db_version WHERE version_id=1 AND NOT is_applied`},
		{"collision_enterprise14", `ALTER TABLE organization.enterprise_profile RENAME TO t07_hidden_enterprise`, `ALTER TABLE organization.t07_hidden_enterprise RENAME TO enterprise_profile`},
		{"collision_legalentity15", `ALTER TABLE organization.legal_entity RENAME TO t07_hidden_legalentity`, `ALTER TABLE organization.t07_hidden_legalentity RENAME TO legal_entity`},
		{"missing_ledger", `ALTER TABLE public.goose_db_version RENAME TO t07_hidden_versions`, `ALTER TABLE public.t07_hidden_versions RENAME TO goose_db_version`},
		{"previous_layout", `UPDATE organization.storage_contract SET contract='m02-compact-1'`, `UPDATE organization.storage_contract SET contract='m02-compact-5'`},
		{"previous_no_index_layout", `UPDATE organization.storage_contract SET contract='m02-compact-2'`, `UPDATE organization.storage_contract SET contract='m02-compact-5'`},
		{"previous_read_rule_layout", `UPDATE organization.storage_contract SET contract='m02-compact-3'`, `UPDATE organization.storage_contract SET contract='m02-compact-5'`},
		{"previous_no_query_layout", `UPDATE organization.storage_contract SET contract='m02-compact-4'`, `UPDATE organization.storage_contract SET contract='m02-compact-5'`},
		{"wrong_contract", `UPDATE organization.storage_contract SET contract='task-owned-mismatch'`, `UPDATE organization.storage_contract SET contract='m02-compact-5'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f.exec(tc.setup)
			r, e := bootstrap.Open(f.ctx, dsn, f.instance)
			if e == nil {
				r.Close()
				t.Error("incomplete/current-mismatched storage opened business runtime")
			}
			if e = f.runtime.Ready(f.ctx); e == nil {
				t.Error("readiness accepted incomplete storage")
			}
			if e = bootstrap.CheckCurrentStorage(f.ctx, f.admin); e == nil {
				t.Error("offline tool admission accepted incomplete or previous storage")
			}
			if e = bootstrap.Migrate(f.ctx, f.admin, migrations.Files); e == nil {
				t.Error("migrator accepted incomplete/different current storage")
			}
			f.exec(tc.restore)
			must(t, f.runtime.Ready(f.ctx))
		})
	}
	// The first prefix keeps the 00011 catalog, including its original code constraints.
	removeLaterExtensions := func() {
		restoreGrantPredecessor(f)
		if f.scalar("SELECT (SELECT count(*) FROM organization.geoarea WHERE country_code='MO') + (SELECT count(*) FROM organization.geoarea_package WHERE country_code='MO')") != 0 {
			t.Fatal("prefix fixture contains Macau catalog data")
		}
		f.exec(`ALTER TABLE organization.geoarea DROP CONSTRAINT geoarea_check3, DROP CONSTRAINT geoarea_country_code_check,
 ADD CHECK (country_code IN ('CN','HK','TW')), ADD UNIQUE (instance_id,country_id,scheme,normalized_code);
 ALTER TABLE organization.geoarea_package DROP CONSTRAINT geoarea_package_country_code_check, ADD CHECK (country_code IN ('CN','HK','TW'));
 DROP INDEX organization.geoarea_assigned_code;
 DELETE FROM public.goose_db_version WHERE version_id>=23;`)
		f.exec(`ALTER TABLE access.subject_grant DROP CONSTRAINT location_single_address_fields; DELETE FROM public.goose_db_version WHERE version_id=16;`)
		removeEmptyLegalEntity206(t, f)
		if f.scalar("SELECT count(*) FROM organization.location") != 0 {
			t.Fatal("prefix fixture would remove Location facts")
		}
		f.exec(`DELETE FROM access.subject_grant WHERE action LIKE 'location.%';
 DROP TABLE organization.location_selection_block,organization.location_selection_draft,organization.location_read,organization.location_terminal,organization.location,organization.location_history,organization.location_shape,organization.location_dictionary CASCADE;
 DROP FUNCTION organization.location_stamp_head(),organization.location_append_guard(),organization.location_head_guard(),organization.location_closure_guard(),organization.location_terminal_guard();
 DELETE FROM public.goose_db_version WHERE version_id=13;`)
	}
	removeLaterExtensions()
	// Reconstruct only the task-owned pre-index schema. The business runtime
	// stays closed until the ordinary upgrader installs all reference indexes.
	treeMigration, err := migrations.Files.ReadFile("00012_department_tree_parent_index.sql")
	must(t, err)
	treeParts := strings.Split(string(treeMigration), "-- +goose Down")
	if len(treeParts) != 2 {
		t.Fatal("tree index migration reverse fixture")
	}
	removeEnterprise304Extension(f)
	f.exec(treeParts[1])
	f.exec("DELETE FROM public.goose_db_version WHERE version_id=12")
	if e := bootstrap.CheckCurrentStorage(f.ctx, f.admin); e == nil {
		t.Fatal("business runtime admitted pending tree parent index")
	}
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	var treeIndex bool
	must(t, f.admin.QueryRow(f.ctx, "SELECT to_regclass('organization.department_tree_parent') IS NOT NULL").Scan(&treeIndex))
	if !treeIndex {
		t.Fatal("tree parent index was not installed")
	}
	restoreGrantPredecessor(f)
	removeEnterprise304Extension(f)
	removeLaterExtensions()
	f.exec(treeParts[1])
	f.exec("DELETE FROM public.goose_db_version WHERE version_id=12")
	// Remove the empty country extension only in this disposable fixture, before
	// reconstructing its pre-index baseline. Existing BU facts remain untouched.
	f.exec(`DELETE FROM access.subject_grant WHERE action LIKE 'country.%' OR action LIKE 'geoarea.%';
 DROP TABLE organization.geoarea_package,organization.geoarea_read,organization.geoarea_terminal,organization.geoarea,organization.geoarea_history,organization.geoarea_shape,organization.geoarea_dictionary CASCADE;
 DROP FUNCTION organization.geoarea_stamp_head(),organization.geoarea_append_guard(),organization.geoarea_head_guard(),organization.geoarea_closure_guard(),organization.geoarea_terminal_guard();
 DROP TABLE organization.country_package, organization.country_terminal, organization.country_region, organization.country_history, organization.country_shape, organization.country_dictionary CASCADE;
 DROP FUNCTION organization.country_stamp_head(),organization.country_append_guard(),organization.country_head_guard(),organization.country_closure_guard(),organization.country_terminal_guard();
 DELETE FROM public.goose_db_version WHERE version_id IN (9,10,11);
 ALTER TABLE access.subject_grant DROP CONSTRAINT IF EXISTS country_resource_fields;
 DO $$ DECLARE c record; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
 IF position('country.%' IN c.def)>0 OR c.conname IN ('subject_grant_scope_kind_check','subject_grant_action_check','subject_grant_check2') THEN
 EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
 END IF; END LOOP; END $$;
 ALTER TABLE access.subject_grant ADD CONSTRAINT subject_grant_scope_kind_check CHECK(scope_kind IN ('enterprise','bu','department')), ADD CONSTRAINT subject_grant_action_check CHECK(action NOT LIKE 'country.%'), ADD CONSTRAINT subject_grant_check2 CHECK(action NOT LIKE 'country.%');`)
	draftMigration, err := migrations.Files.ReadFile("00008_department_selection_drafts.sql")
	must(t, err)
	draftParts := strings.Split(string(draftMigration), "-- +goose Down")
	if len(draftParts) != 2 {
		t.Fatal("draft migration reverse fixture")
	}
	f.exec(draftParts[1])
	f.exec("DELETE FROM public.goose_db_version WHERE version_id=8")
	readMigration, err := migrations.Files.ReadFile("00007_source_read_segments.sql")
	must(t, err)
	parts := strings.Split(string(readMigration), "-- +goose Down")
	if len(parts) != 2 {
		t.Fatal("read migration has no single reverse fixture")
	}
	f.exec(parts[1])
	f.exec(`DELETE FROM public.goose_db_version WHERE version_id=7;
DROP INDEX organization.bu_history_dictionary_reference;
DROP INDEX organization.bu_terminal_dictionary_reference;
DROP INDEX organization.department_history_dictionary_reference;
DROP INDEX organization.department_terminal_dictionary_reference;
DELETE FROM public.goose_db_version WHERE version_id=6`)
	if r, e := bootstrap.Open(f.ctx, dsn, f.instance); e == nil {
		r.Close()
		t.Fatal("business runtime admitted pending reference indexes")
	}
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	var indexes int
	must(t, f.admin.QueryRow(f.ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname='organization' AND indexname IN ('bu_history_dictionary_reference','bu_terminal_dictionary_reference','department_history_dictionary_reference','department_terminal_dictionary_reference')`).Scan(&indexes))
	if indexes != 4 {
		t.Fatal("reference indexes were not installed", indexes)
	}
	// Current complete installation remains repeatable, including latest-state
	// interpretation of Goose rows, without changing a source or a terminal.
	must(t, bootstrap.Migrate(f.ctx, f.admin, migrations.Files))
	must(t, bootstrap.Initialize(f.ctx, f.admin, bootstrap.InitInput{Instance: f.instance, EnterpriseCode: "LOCAL", EnterpriseName: "本地验证企业", Start: "2020-01-01", Login: "admin", Password: f.password}))
	r, e := bootstrap.Open(f.ctx, dsn, f.instance)
	must(t, e)
	r.Close()
	for table, n := range businessSnapshot(t, f) {
		if before[table] != n {
			t.Fatal("admission altered business history", table)
		}
	}
}

func TestT07FreshInstallMidMigrationFailureClosesBusiness(t *testing.T) {
	f := fresh(t) // Container belongs solely to this test; create extra EMPTY databases.
	for index, object := range []string{"access.session", "organization.bu_history"} {
		t.Run(object, func(t *testing.T) {
			name := fmt.Sprintf("t07_install_%d", index)
			_, e := f.admin.Exec(f.ctx, "CREATE DATABASE "+name)
			must(t, e)
			config := f.admin.Config().Copy()
			config.ConnConfig.Database = name
			pool, e := pgxpool.NewWithConfig(f.ctx, config)
			must(t, e)
			defer pool.Close()
			// A test-owned event trigger fails an actual middle DDL statement. The
			// embedded migrations remain byte-for-byte the product migrations.
			_, e = pool.Exec(f.ctx, `CREATE FUNCTION public.t07_fail_ddl() RETURNS event_trigger LANGUAGE plpgsql AS $$ DECLARE d record; BEGIN FOR d IN SELECT * FROM pg_event_trigger_ddl_commands() LOOP IF d.object_identity='`+object+`' THEN RAISE EXCEPTION 'T07 controlled install interruption'; END IF; END LOOP; END $$; CREATE EVENT TRIGGER t07_fail_ddl ON ddl_command_end EXECUTE FUNCTION public.t07_fail_ddl()`)
			must(t, e)
			e = bootstrap.Migrate(f.ctx, pool, migrations.Files)
			if e == nil || !strings.Contains(e.Error(), "T07 controlled install interruption") {
				t.Fatal("missing controlled middle DDL failure", e)
			}
			var versions int
			var access, organization bool
			must(t, pool.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM public.goose_db_version WHERE is_applied AND version_id>0),to_regclass('access.principal') IS NOT NULL,to_regclass('organization.business_unit') IS NOT NULL`).Scan(&versions, &access, &organization))
			if versions != index || access != (index == 1) || organization {
				t.Fatal("unexpected per-file rollback boundary", versions, access, organization)
			}
			if r, e := bootstrap.Open(f.ctx, pool.Config().ConnString(), foundation.NewID()); e == nil {
				r.Close()
				t.Fatal("partial install exposed business runtime")
			}
			_, e = pool.Exec(f.ctx, `DROP EVENT TRIGGER t07_fail_ddl; DROP FUNCTION public.t07_fail_ddl()`)
			must(t, e)
			if index == 0 {
				must(t, bootstrap.Migrate(f.ctx, pool, migrations.Files))
				must(t, bootstrap.Migrate(f.ctx, pool, migrations.Files))
				t.Log("00001 failure rolled back its entire schema; zero applied files; removing fault permits repeat current installation")
			} else {
				if e := bootstrap.Migrate(f.ctx, pool, migrations.Files); e == nil {
					t.Fatal("partial development install was silently reconstructed")
				}
				t.Log("00002 failure rolled back organization; 00001 remains applied; runtime and further migration fail closed; use a new isolated database")
			}
		})
	}
}
