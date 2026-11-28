package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"

	"github.com/pressly/goose/v3"
)

const (
	gradeRateFile        = "00059_compensation_grade_rate.sql"
	compensationPlanFile = "00060_compensation_plans.sql"
	accessManagementFile = "00064_access_management.sql"
)

// The old security release recorded its management objects as 59. Inspect
// the same objects for admission and again under Goose's migration lock.
const managementStorageSQL = `SELECT
  (root_rel IS NOT NULL OR mandate_rel IS NOT NULL OR publication_rel IS NOT NULL OR receipt_rel IS NOT NULL
   OR owner_column OR to_regprocedure('access.protect_management_mandate()') IS NOT NULL) AS present,
  (owner_column AND EXISTS (SELECT FROM pg_attribute WHERE attrelid=to_regclass('access.principal')
     AND attname='management_owner' AND atttypid='uuid'::regtype AND NOT attisdropped)
   AND (SELECT count(*)=4 FROM (VALUES
     (root_rel, ARRAY['instance_id','principal_id','max_proxy_days','receipt_key']),
     (mandate_rel, ARRAY['instance_id','id','principal_id','owner_id','rights','role_id','scope_id','fields_id','role_kind','scope_kind','fields_kind','max_proxy_days','valid_until','revoked_at']),
     (publication_rel, ARRAY['instance_id','kind','version_id','name','scope_id','fields_id','scope_kind','fields_kind']),
     (receipt_rel, ARRAY['receipt_order','instance_id','request_id','actor_id','operator_name','operation','target_id','subject_id','recorded_at','reason','command_seal','previous_owner','assigned_owner']))
     AS required(rel,cols) JOIN pg_class tbl ON tbl.oid=required.rel AND tbl.relkind='r'
     WHERE required.cols <@ ARRAY(SELECT attname::text FROM pg_attribute WHERE attrelid=tbl.oid AND attnum>0 AND NOT attisdropped)
       AND EXISTS(SELECT FROM pg_index WHERE indrelid=tbl.oid AND indisprimary AND indisvalid))
   AND EXISTS(SELECT FROM pg_attribute WHERE attrelid=receipt_rel AND attname='receipt_order' AND attidentity='a' AND NOT attisdropped)
   AND (SELECT count(*)=3 FROM (VALUES
     (mandate_rel,'immutable_management_mandate','access.protect_management_mandate()'),
     (publication_rel,'immutable_management_publication','access.protect_security_event()'),
     (receipt_rel,'immutable_management_receipt','access.protect_security_event()'))
     AS expected(rel,trig,proc) JOIN pg_trigger t ON t.tgrelid=expected.rel AND t.tgname=expected.trig
       AND t.tgenabled='O' AND t.tgfoid=to_regprocedure(expected.proc))) AS complete
 FROM (SELECT to_regclass('access.management_root') root_rel, to_regclass('access.management_mandate') mandate_rel,
              to_regclass('access.management_publication') publication_rel, to_regclass('access.management_receipt') receipt_rel,
              EXISTS(SELECT FROM pg_attribute WHERE attrelid=to_regclass('access.principal')
                     AND attname='management_owner' AND NOT attisdropped) owner_column) objects`

func managementLineageMigration(files fs.FS, name string, version int64) *goose.Migration {
	up := func(ctx context.Context, tx *sql.Tx) error {
		var present, complete, legacy59 bool
		err := tx.QueryRowContext(ctx, `SELECT present, complete,
			complete AND to_regnamespace('compensation') IS NULL AND to_regnamespace('configuration') IS NULL
			AND (SELECT array_agg(version_id ORDER BY version_id) FROM (
				SELECT DISTINCT ON (version_id) version_id,is_applied FROM public.goose_db_version
				WHERE version_id>0 ORDER BY version_id,id DESC) latest WHERE is_applied)
				=ARRAY(SELECT generate_series(1,59)::bigint)
			FROM (`+managementStorageSQL+`) management`).Scan(&present, &complete, &legacy59)
		if err != nil {
			return err
		}
		if present && !complete {
			return errors.New("incomplete access management storage")
		}
		if name == accessManagementFile && complete {
			return nil
		}
		if name == compensationPlanFile && legacy59 {
			// The salary plan references grade rates. Both immutable SQL files
			// succeed together before Goose appends row 60 to the old ledger.
			if err := runMigrationSQL(ctx, tx, files, gradeRateFile); err != nil {
				return err
			}
		}
		return runMigrationSQL(ctx, tx, files, name)
	}
	return goose.NewGoMigration(version, &goose.GoFunc{RunTx: up}, &goose.GoFunc{RunTx: func(context.Context, *sql.Tx) error {
		return errors.New("published access and compensation history requires forward migration")
	}})
}
