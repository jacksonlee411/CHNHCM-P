-- +goose Up
ALTER TABLE staffing.quota
 DROP CONSTRAINT quota_target_kind_check,
 DROP CONSTRAINT quota_instance_id_target_id_fkey,
 ADD CONSTRAINT quota_target_kind_check CHECK(target_kind IN ('department','position')),
 ADD COLUMN position_ref uuid GENERATED ALWAYS AS (CASE WHEN target_kind='position' THEN target_id END) STORED,
 ADD FOREIGN KEY(instance_id, position_ref) REFERENCES staffing.position(instance_id,id);
UPDATE staffing.storage_contract SET contract='staffing-allocation-1' WHERE singleton AND contract='staffing-quota-1';
-- +goose Down
SELECT 1/0;
