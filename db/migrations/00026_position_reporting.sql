-- +goose Up
LOCK TABLE staffing.position_hierarchy IN ACCESS EXCLUSIVE MODE;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS (SELECT FROM pg_attribute WHERE attrelid='staffing.position_hierarchy'::regclass AND attname='purpose' AND NOT attisdropped) THEN
  IF EXISTS (SELECT FROM staffing.position_hierarchy WHERE purpose <> 'reporting') THEN
   RAISE EXCEPTION 'cannot convert non-reporting position history into management reporting' USING ERRCODE='23514';
  END IF;
  ALTER TABLE staffing.position_hierarchy ADD CONSTRAINT position_hierarchy_enterprise_unique UNIQUE(instance_id, enterprise_id);
 ELSIF NOT EXISTS (SELECT FROM pg_constraint c JOIN pg_index i ON i.indexrelid=c.conindid
   WHERE c.conrelid='staffing.position_hierarchy'::regclass AND c.contype='u' AND c.convalidated
   AND i.indisvalid AND i.indisready AND pg_get_constraintdef(c.oid)='UNIQUE (instance_id, enterprise_id)') THEN
  RAISE EXCEPTION 'unrecognized position reporting layout';
 END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE staffing.position_hierarchy DROP COLUMN IF EXISTS purpose;
UPDATE staffing.storage_contract SET contract='staffing-classification-reporting-1' WHERE singleton AND contract='staffing-job-classification-1';

-- +goose Down
SELECT 1/0;
