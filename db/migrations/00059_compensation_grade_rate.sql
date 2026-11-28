-- +goose Up
CREATE SCHEMA compensation;
GRANT USAGE ON SCHEMA compensation TO chnhcm_app;

CREATE TABLE compensation.grade_rate (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, id uuid NOT NULL,
 grade_id uuid NOT NULL, currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 frequency text NOT NULL, unit text NOT NULL, places smallint NOT NULL CHECK(places BETWEEN 0 AND 18),
 head uuid NOT NULL, PRIMARY KEY(instance_id,id),
 UNIQUE(instance_id,enterprise_id,grade_id,currency,frequency,unit),
 CHECK ((frequency,unit) IN (('MONTH','PER_MONTH'),('YEAR','PER_YEAR'),('HOUR','PER_HOUR'),('ONCE','PER_OCCURRENCE')))
);
CREATE TABLE compensation.grade_rate_revision (
 instance_id uuid NOT NULL, rate_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 fact jsonb NOT NULL, PRIMARY KEY(instance_id,rate_id,record_id,revision),
 FOREIGN KEY(instance_id,rate_id) REFERENCES compensation.grade_rate
);
CREATE TABLE compensation.grade_rate_version (
 instance_id uuid NOT NULL, rate_id uuid NOT NULL, id uuid NOT NULL, predecessor uuid,
 action text NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL, previous_record_id uuid, previous_revision uuid, until_date date, previous_until date,
 operator jsonb NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,rate_id,id),
 FOREIGN KEY(instance_id,rate_id,predecessor) REFERENCES compensation.grade_rate_version,
 FOREIGN KEY(instance_id,rate_id,record_id,revision) REFERENCES compensation.grade_rate_revision,
 CHECK ((previous_record_id IS NULL)=(previous_revision IS NULL)),
 FOREIGN KEY(instance_id,rate_id,previous_record_id,previous_revision) REFERENCES compensation.grade_rate_revision
);
ALTER TABLE compensation.grade_rate ADD FOREIGN KEY(instance_id,id,head)
 REFERENCES compensation.grade_rate_version DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE compensation.grade_rate_period (
 instance_id uuid NOT NULL, rate_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 start_date date NOT NULL, until_date date, active boolean NOT NULL,
 valid_period daterange GENERATED ALWAYS AS (daterange(start_date,until_date,'[)')) STORED,
 PRIMARY KEY(instance_id,rate_id,record_id), UNIQUE(instance_id,rate_id,start_date),
 CHECK(until_date IS NULL OR start_date<until_date),
 FOREIGN KEY(instance_id,rate_id,record_id,revision) REFERENCES compensation.grade_rate_revision
);
CREATE INDEX grade_rate_period_lookup ON compensation.grade_rate_period USING gist(valid_period);

CREATE TABLE compensation.grade_rate_receipt (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL,
 digest bytea CHECK(octet_length(digest)=32), result jsonb NOT NULL,
 CHECK(result ? 'outcome' AND result->>'outcome' IN ('accepted','no_change','not_applied','rejected')),
 CHECK((digest IS NULL) = (result->>'outcome'='not_applied')),
 PRIMARY KEY(instance_id,subject_id,submission_id)
);
CREATE TABLE compensation.grade_rate_outbox (
 instance_id uuid NOT NULL, rate_id uuid NOT NULL, version uuid NOT NULL,
 event jsonb NOT NULL, PRIMARY KEY(instance_id,rate_id,version),
 FOREIGN KEY(instance_id,rate_id,version) REFERENCES compensation.grade_rate_version
);

-- +goose StatementBegin
CREATE FUNCTION compensation.immutable_fact() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'immutable compensation fact' USING ERRCODE='55000'; END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_revision BEFORE UPDATE OR DELETE ON compensation.grade_rate_revision FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
CREATE TRIGGER immutable_version BEFORE UPDATE OR DELETE ON compensation.grade_rate_version FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
CREATE TRIGGER immutable_receipt BEFORE UPDATE OR DELETE ON compensation.grade_rate_receipt FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON compensation.grade_rate_outbox FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
GRANT SELECT,INSERT ON ALL TABLES IN SCHEMA compensation TO chnhcm_app;
GRANT UPDATE(head) ON compensation.grade_rate TO chnhcm_app;
GRANT UPDATE,DELETE ON compensation.grade_rate_period TO chnhcm_app;

-- Preserve the deployed predicates and versioned grant identities. New payroll
-- permissions are explicit enterprise grants and never inferred from M03 reads.
ALTER FUNCTION access.valid_permission(uuid,text,uuid,text,text[],text[]) RENAME TO valid_pre_compensation_permission;
-- +goose StatementBegin
CREATE FUNCTION access.valid_permission(instance_id uuid,scope_kind text,scope_id uuid,action text,read_fields text[],write_fields text[])
RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT CASE WHEN split_part(action,'.',1)='graderate' THEN
    scope_kind='enterprise' AND action IN ('graderate.read','graderate.reference','graderate.history.read','graderate.create','graderate.add','graderate.correct','graderate.delete','graderate.deactivate','graderate.reactivate')
    AND read_fields <@ ARRAY['gradeId','currency','frequency','unit','precision','form','amount','minimum','guidance','maximum','effectiveDate','reason','status','operator']
    AND write_fields <@ ARRAY['gradeId','currency','frequency','unit','precision','form','amount','minimum','guidance','maximum','effectiveDate','reason']
    AND (action NOT IN ('graderate.read','graderate.reference','graderate.history.read') OR cardinality(write_fields)=0)
 ELSE access.valid_pre_compensation_permission(instance_id,scope_kind,scope_id,action,read_fields,write_fields) END
$$;
-- +goose StatementEnd

-- +goose Down
-- Destructive rollback would discard accepted terms and original adoptions.
SELECT 1/0;
