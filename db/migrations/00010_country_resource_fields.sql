-- +goose Up
-- Provider attribution belongs to the versioned offline input, not field grants.
WITH changed AS (
 UPDATE access.subject_grant
 SET read_fields=array_remove(read_fields,'source'), write_fields=array_remove(write_fields,'source')
 WHERE action LIKE 'country.%' AND ('source'=ANY(read_fields) OR 'source'=ANY(write_fields))
 RETURNING instance_id,principal_id
)
UPDATE access.policy_guard p SET policy_no=policy_no+1
WHERE EXISTS(SELECT FROM changed c WHERE (c.instance_id,c.principal_id)=(p.instance_id,p.principal_id));

ALTER TABLE access.subject_grant ADD CONSTRAINT country_resource_fields
 CHECK(action NOT LIKE 'country.%' OR NOT ('source'=ANY(read_fields) OR 'source'=ANY(write_fields)));

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'removed country source fields cannot be automatically restored'; END $$;
-- +goose StatementEnd
