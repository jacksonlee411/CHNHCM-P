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
	addressTypesFile   = "00061_person_address_types.sql"
	addressPeriodsFile = "00062_person_address_type_periods.sql"
	basePayFile        = "00063_assignment_base_pay.sql"
)

// The salary branch shipped these tables with revision, grant and range guards at
// version 61. Recognize the complete published layout before adopting its facts.
const basePayLayout = `(NOT EXISTS (
 SELECT FROM unnest(ARRAY['access.base_pay_grant','compensation.award_revision','compensation.award_period',
 'compensation.award_timeline','compensation.award_receipt','compensation.award_recovery','compensation.award_outbox']) rel
 WHERE to_regclass(rel) IS NULL)
 AND NOT EXISTS (
 SELECT FROM (VALUES ('access.base_pay_grant','guard_base_pay','access.person_grant_guard()'),
 ('compensation.award_revision','immutable_revision','compensation.immutable_fact()'),
 ('compensation.award_receipt','immutable_receipt','compensation.immutable_fact()'),
 ('compensation.award_recovery','immutable_confirmation','compensation.immutable_fact()'),
 ('compensation.award_outbox','immutable_event','compensation.immutable_fact()')) expected(rel,name,fn)
 WHERE NOT EXISTS (SELECT FROM pg_trigger WHERE tgrelid=to_regclass(expected.rel) AND tgname=expected.name
 AND tgenabled='O' AND tgfoid=to_regprocedure(expected.fn)))
 AND EXISTS (SELECT FROM pg_constraint WHERE conrelid=to_regclass('compensation.award_period') AND contype='x' AND convalidated))`

const legacyBasePay61 = `(ARRAY(SELECT version_id FROM (SELECT DISTINCT ON (version_id) version_id,is_applied
 FROM public.goose_db_version WHERE version_id>0 ORDER BY version_id,id DESC) latest WHERE is_applied ORDER BY version_id)
 = ARRAY(SELECT generate_series(1,61)::bigint)
 AND to_regclass('personnel.address_type') IS NULL AND to_regclass('personnel.address_type_revision') IS NULL
 AND to_regclass('personnel.address_type_not_applied') IS NULL
 AND NOT EXISTS (SELECT FROM pg_attribute WHERE attrelid=to_regclass('personnel.profile_item_payload')
 AND attname IN ('address_type_id','address_type_revision','address_type_day') AND NOT attisdropped)
 AND ` + basePayLayout + `)`

func addressPeriodsAfterBasePay(files fs.FS) *goose.Migration {
	return goose.NewGoMigration(62, &goose.GoFunc{RunTx: func(ctx context.Context, tx *sql.Tx) error {
		var legacy bool
		if err := tx.QueryRowContext(ctx, "SELECT "+legacyBasePay61).Scan(&legacy); err != nil {
			return err
		}
		if legacy {
			if err := runMigrationUp(ctx, tx, files, addressTypesFile); err != nil {
				return err
			}
			// The first release seeds an address and its deferred head reference for
			// every existing customer. Validate those references before the period DDL.
			if _, err := tx.ExecContext(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
				return err
			}
		}
		return runMigrationUp(ctx, tx, files, addressPeriodsFile)
	}}, &goose.GoFunc{RunTx: rejectPayDowngrade})
}

func adoptBasePay(files fs.FS) *goose.Migration {
	return goose.NewGoMigration(63, &goose.GoFunc{RunTx: func(ctx context.Context, tx *sql.Tx) error {
		var installed bool
		if err := tx.QueryRowContext(ctx, "SELECT "+basePayLayout).Scan(&installed); err != nil || installed {
			return err
		}
		return runMigrationUp(ctx, tx, files, basePayFile)
	}}, &goose.GoFunc{RunTx: rejectPayDowngrade})
}

func rejectPayDowngrade(context.Context, *sql.Tx) error {
	return errors.New("address and salary history require forward migration")
}

func runMigrationUp(ctx context.Context, tx *sql.Tx, files fs.FS, name string) error {
	body, err := fs.ReadFile(files, name)
	if err != nil {
		return err
	}
	up, _, ok := strings.Cut(string(body), "-- +goose Down")
	if !ok {
		return errors.New("missing migration boundary in " + name)
	}
	_, err = tx.ExecContext(ctx, up)
	return err
}
