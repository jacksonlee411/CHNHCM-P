-- +goose Up
-- Executed by the paired Go migration after converting immutable location facts.
SET CONSTRAINTS ALL IMMEDIATE;
ALTER TABLE organization.location_history ENABLE TRIGGER USER;
 ALTER TABLE organization.location ENABLE TRIGGER protect_head;
 UPDATE organization.location_read SET address_line=CASE WHEN coalesce(address_extra,'')='' THEN address_line ELSE address_line||' '||address_extra END;
 ALTER TABLE organization.location_read DROP COLUMN address_extra;
 -- Selection signatures include the removed field. Expire these transient sets.
 DELETE FROM organization.location_selection_draft;
 WITH changed AS (
  UPDATE access.subject_grant SET
   read_fields=array_remove(CASE WHEN 'addressExtra'=ANY(read_fields) THEN read_fields ELSE array_remove(read_fields,'addressLine') END,'addressExtra'),
   write_fields=array_remove(CASE WHEN 'addressExtra'=ANY(write_fields) THEN write_fields ELSE array_remove(write_fields,'addressLine') END,'addressExtra')
  WHERE action LIKE 'location.%' AND ('addressExtra'=ANY(read_fields) OR 'addressExtra'=ANY(write_fields) OR 'addressLine'=ANY(read_fields) OR 'addressLine'=ANY(write_fields))
  RETURNING instance_id,principal_id
 ) UPDATE access.policy_guard p SET policy_no=policy_no+1 WHERE EXISTS(SELECT FROM changed c WHERE (c.instance_id,c.principal_id)=(p.instance_id,p.principal_id));
 ALTER TABLE access.subject_grant ADD CONSTRAINT location_single_address_fields CHECK(action NOT LIKE 'location.%' OR NOT ('addressExtra'=ANY(read_fields) OR 'addressExtra'=ANY(write_fields)));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'merged location addresses cannot be split automatically'; END $$;
-- +goose StatementEnd
