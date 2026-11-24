-- +goose Up
-- Validate under the same lock that removes the discriminator. Even a single
-- non-reporting root must never be reinterpreted as management reporting.
LOCK TABLE staffing.position_hierarchy IN ACCESS EXCLUSIVE MODE;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS (SELECT FROM staffing.position_hierarchy WHERE purpose <> 'reporting') THEN
  RAISE EXCEPTION 'cannot convert non-reporting position history into management reporting' USING ERRCODE='23514';
 END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE staffing.position_hierarchy DROP COLUMN purpose;
ALTER TABLE staffing.position_hierarchy ADD CONSTRAINT position_hierarchy_enterprise_unique UNIQUE(instance_id, enterprise_id);
UPDATE staffing.storage_contract SET contract='staffing-reporting-1' WHERE singleton AND contract='staffing-hierarchy-1';

-- +goose Down
SELECT 1/0;
