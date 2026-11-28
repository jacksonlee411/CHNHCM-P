package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"strings"

	"github.com/pressly/goose/v3"
)

const (
	reportingMigrationFile = "00026_position_reporting.sql"
	positionQuotaFile      = "00027_position_quota.sql"
	quotaConvergenceFile   = "00028_staffing_quota_convergence.sql"
)

// Goose holds the session migration lock and records each version in the same
// transaction as the conversion. Read the predecessor again under that lock.
func quotaLineageMigration(files fs.FS, version int64) *goose.Migration {
	up := func(ctx context.Context, tx *sql.Tx) error {
		var contract string
		if err := tx.QueryRowContext(ctx, "SELECT contract FROM staffing.storage_contract WHERE singleton FOR UPDATE").Scan(&contract); err != nil {
			return err
		}
		run := func(name string) error { return runMigrationSQL(ctx, tx, files, name) }
		reporting := func() error {
			var datedPurposes bool
			if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT FROM pg_attribute WHERE attrelid='staffing.position_hierarchy'::regclass AND attname='purpose' AND NOT attisdropped)").Scan(&datedPurposes); err != nil {
				return err
			}
			if datedPurposes {
				if err := run("lineage315/retire_functional_hierarchy.sql"); err != nil {
					return err
				}
			}
			return run(reportingMigrationFile)
		}
		if version == 26 {
			return reporting()
		}
		// The separately shipped Personnel ledger used 26 and 27 without changing Staffing.
		if contract == "staffing-job-classification-1" {
			var legacy bool
			if err := tx.QueryRowContext(ctx, "SELECT to_regclass('personnel.person') IS NOT NULL").Scan(&legacy); err != nil {
				return err
			}
			if !legacy {
				return errors.New("unknown migration lineage")
			}
			// Preserve historical facts on this path. Reporting migration refuses incompatible roots.
			if err := run(reportingMigrationFile); err != nil {
				return err
			}
			contract = "staffing-classification-reporting-1"
			if version == 28 {
				if err := run(positionQuotaFile); err != nil {
					return err
				}
				contract = "staffing-quota-1"
			}
		}
		if version == 27 {
			switch contract {
			case "staffing-classification-reporting-1":
				return run(positionQuotaFile)
			case "staffing-quota-1":
				return run("lineage315/00027_department_quota.sql")
			default:
				return errors.New("unrecognized staffing migration 26")
			}
		}
		switch contract {
		case "staffing-quota-1":
			if err := run(quotaConvergenceFile); err != nil {
				return err
			}
		case "staffing-allocation-1":
			if err := reporting(); err != nil {
				return err
			}
			// Only newly established positions receive an origin. Preexisting
			// positions on this lineage have no recorded controlled origin.
			if _, err := tx.ExecContext(ctx, `CREATE TABLE staffing.quota_control (
				instance_id uuid NOT NULL, position_id uuid NOT NULL, position_source uuid NOT NULL, controlled_from date NOT NULL,
				PRIMARY KEY(instance_id,position_id), FOREIGN KEY(instance_id,position_id,position_source)
				REFERENCES staffing.position_source(instance_id,object_id,source_id));
				CREATE TRIGGER immutable_control BEFORE UPDATE OR DELETE ON staffing.quota_control
				FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
				GRANT SELECT,INSERT ON staffing.quota_control TO chnhcm_app`); err != nil {
				return err
			}
		default:
			return errors.New("unrecognized staffing migration 27")
		}
		if err := run("lineage315/quota_grants.sql"); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE staffing.storage_contract SET contract='staffing-unified-quota-1' WHERE singleton")
		return err
	}
	return goose.NewGoMigration(version, &goose.GoFunc{RunTx: up}, &goose.GoFunc{RunTx: func(context.Context, *sql.Tx) error {
		return errors.New("published quota histories cannot be downgraded")
	}})
}

func personnelLineageMigration(files fs.FS, name string, version int64) *goose.Migration {
	up := func(ctx context.Context, tx *sql.Tx) error {
		var installed bool
		probe := `select to_regprocedure('access.person_policy_for_use(uuid)') is not null
			and not exists(select from unnest(array['person','number','source','counter','terminal','identity_payload','evidence_payload','contact_payload','material','outbox','query','hold','disposition']) name where to_regclass('personnel.'||name) is null)`
		if version == 30 {
			probe = `select count(*)=2 from pg_index where indrelid=to_regclass('personnel.identity_payload') and indisunique and indisvalid and indexrelid in (to_regclass('personnel.current_document_number'),to_regclass('personnel.current_document_person'))`
		}
		if err := tx.QueryRowContext(ctx, probe).Scan(&installed); err != nil || installed {
			return err
		}
		return runMigrationSQL(ctx, tx, files, name)
	}
	return goose.NewGoMigration(version, &goose.GoFunc{RunTx: up}, &goose.GoFunc{RunTx: func(context.Context, *sql.Tx) error {
		return errors.New("personnel history requires forward migration")
	}})
}

func runMigrationSQL(ctx context.Context, tx *sql.Tx, files fs.FS, name string) error {
	raw, err := fs.ReadFile(files, name)
	if err != nil {
		return err
	}
	up, _, ok := strings.Cut(string(raw), "-- +goose Down")
	if !ok {
		return errors.New("missing migration boundary in " + name)
	}
	_, err = tx.ExecContext(ctx, up)
	return err
}
