-- +goose Up
-- Staffing owns these facts. No M02 history codes, procedures or state machine.
CREATE SCHEMA staffing;
CREATE TABLE staffing.storage_contract (singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton), contract text NOT NULL);
INSERT INTO staffing.storage_contract VALUES(true,'staffing-job-foundation-1');
GRANT SELECT ON staffing.storage_contract TO chnhcm_app;
-- +goose StatementBegin
CREATE FUNCTION staffing.immutable_fact() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'staffing facts are immutable' USING ERRCODE='23514'; END $$;
-- +goose StatementEnd
CREATE TABLE staffing.job_family (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, id uuid NOT NULL, code text NOT NULL,
 source_id uuid NOT NULL, source_no bigint NOT NULL CHECK(source_no>0), first_date date NOT NULL,
 head jsonb NOT NULL, PRIMARY KEY(instance_id,id), UNIQUE(instance_id,enterprise_id,code),
 CHECK(code ~ '^[A-Z0-9][A-Z0-9_-]{0,29}$')
);
CREATE INDEX job_family_enterprise_dates ON staffing.job_family(instance_id,enterprise_id,first_date);
CREATE TABLE staffing.job_family_source (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, source_no bigint NOT NULL,
 subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea NOT NULL CHECK(octet_length(digest)=32),
 snapshot jsonb NOT NULL, operation jsonb NOT NULL, operator jsonb NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,object_id,source_id), UNIQUE(instance_id,object_id,source_no),
 FOREIGN KEY(instance_id,object_id) REFERENCES staffing.job_family(instance_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON staffing.job_family_source FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
ALTER TABLE staffing.job_family ADD FOREIGN KEY(instance_id,id,source_id) REFERENCES staffing.job_family_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE staffing.job_family_revision (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 source_id uuid NOT NULL, fact jsonb NOT NULL, PRIMARY KEY(instance_id,object_id,record_id,revision),
 FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.job_family_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER immutable_revision BEFORE UPDATE OR DELETE ON staffing.job_family_revision FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.job_family_read (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, object_id uuid NOT NULL, record_id uuid NOT NULL,
 source_id uuid NOT NULL, start_date date NOT NULL, end_date date, active boolean NOT NULL, family_id uuid, fact jsonb NOT NULL,
 PRIMARY KEY(instance_id,object_id,record_id), FOREIGN KEY(instance_id,object_id) REFERENCES staffing.job_family(instance_id,id),
 CHECK(end_date IS NULL OR end_date>start_date)
);
CREATE INDEX job_family_references ON staffing.job_family_read(instance_id,enterprise_id,family_id) WHERE active;
CREATE TABLE staffing.job_family_terminal (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea,
 result jsonb NOT NULL, closed_at timestamptz NOT NULL DEFAULT clock_timestamp(), PRIMARY KEY(instance_id,subject_id,submission_id)
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON staffing.job_family_terminal FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.job_family_outbox (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, event jsonb NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id), FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.job_family_source(instance_id,object_id,source_id)
);
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON staffing.job_family_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.job_family_selection (
 instance_id uuid NOT NULL, session_id uuid NOT NULL, id uuid NOT NULL, revision uuid NOT NULL,
 business_date date NOT NULL, query jsonb NOT NULL, members uuid[] NOT NULL, excluded uuid[] NOT NULL DEFAULT '{}',
 expires_at timestamptz NOT NULL, cancelled boolean NOT NULL DEFAULT false,
 PRIMARY KEY(instance_id,session_id,id)
);
CREATE TABLE staffing.job (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, id uuid NOT NULL, code text NOT NULL,
 source_id uuid NOT NULL, source_no bigint NOT NULL CHECK(source_no>0), first_date date NOT NULL,
 head jsonb NOT NULL, PRIMARY KEY(instance_id,id), UNIQUE(instance_id,enterprise_id,code),
 CHECK(code ~ '^[A-Z0-9][A-Z0-9_-]{0,29}$')
);
CREATE INDEX job_enterprise_dates ON staffing.job(instance_id,enterprise_id,first_date);
CREATE TABLE staffing.job_source (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, source_no bigint NOT NULL,
 subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea NOT NULL CHECK(octet_length(digest)=32),
 snapshot jsonb NOT NULL, operation jsonb NOT NULL, operator jsonb NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,object_id,source_id), UNIQUE(instance_id,object_id,source_no),
 FOREIGN KEY(instance_id,object_id) REFERENCES staffing.job(instance_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON staffing.job_source FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
ALTER TABLE staffing.job ADD FOREIGN KEY(instance_id,id,source_id) REFERENCES staffing.job_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE staffing.job_revision (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 source_id uuid NOT NULL, fact jsonb NOT NULL, PRIMARY KEY(instance_id,object_id,record_id,revision),
 FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.job_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER immutable_revision BEFORE UPDATE OR DELETE ON staffing.job_revision FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.job_read (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, object_id uuid NOT NULL, record_id uuid NOT NULL,
 source_id uuid NOT NULL, start_date date NOT NULL, end_date date, active boolean NOT NULL, family_id uuid, fact jsonb NOT NULL,
 PRIMARY KEY(instance_id,object_id,record_id), FOREIGN KEY(instance_id,object_id) REFERENCES staffing.job(instance_id,id),
 CHECK(end_date IS NULL OR end_date>start_date)
);
CREATE INDEX job_references ON staffing.job_read(instance_id,enterprise_id,family_id) WHERE active;
CREATE TABLE staffing.job_terminal (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea,
 result jsonb NOT NULL, closed_at timestamptz NOT NULL DEFAULT clock_timestamp(), PRIMARY KEY(instance_id,subject_id,submission_id)
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON staffing.job_terminal FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.job_outbox (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, event jsonb NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id), FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.job_source(instance_id,object_id,source_id)
);
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON staffing.job_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.job_selection (
 instance_id uuid NOT NULL, session_id uuid NOT NULL, id uuid NOT NULL, revision uuid NOT NULL,
 business_date date NOT NULL, query jsonb NOT NULL, members uuid[] NOT NULL, excluded uuid[] NOT NULL DEFAULT '{}',
 expires_at timestamptz NOT NULL, cancelled boolean NOT NULL DEFAULT false,
 PRIMARY KEY(instance_id,session_id,id)
);
GRANT USAGE ON SCHEMA staffing TO chnhcm_app;
GRANT SELECT,INSERT ON staffing.job_family,staffing.job_family_source,staffing.job_family_revision,staffing.job_family_terminal,staffing.job_family_outbox TO chnhcm_app;
GRANT UPDATE(source_id,source_no,first_date,head) ON staffing.job_family TO chnhcm_app;
GRANT SELECT,INSERT,UPDATE,DELETE ON staffing.job_family_read,staffing.job_family_selection TO chnhcm_app;
GRANT SELECT,INSERT ON staffing.job,staffing.job_source,staffing.job_revision,staffing.job_terminal,staffing.job_outbox TO chnhcm_app;
GRANT UPDATE(source_id,source_no,first_date,head) ON staffing.job TO chnhcm_app;
GRANT SELECT,INSERT,UPDATE,DELETE ON staffing.job_read,staffing.job_selection TO chnhcm_app;

-- Extend exactly the existing action and field predicates; preserve M02 checks.
-- +goose StatementBegin
DO $$ DECLARE c record; extension text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  extension:=NULL;
  IF position('read_fields' IN c.def)>0 THEN extension:=$fields$(action LIKE 'jobfamily.%' AND read_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','description','status']::text[] AND write_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','description']::text[]) OR (action LIKE 'job.%' AND read_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','jobFamilyId','responsibilities','requirements','status']::text[] AND write_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','jobFamilyId','responsibilities','requirements']::text[])$fields$;
  ELSIF position('bu.read' IN c.def)>0 AND position('action' IN c.def)>0 THEN extension:=$actions$action IN ('jobfamily.read','jobfamily.history.read','jobfamily.export','jobfamily.create','jobfamily.record.add','jobfamily.record.correct','jobfamily.record.delete','jobfamily.record.redate','jobfamily.deactivate','jobfamily.reactivate','jobfamily.lifecycle.pair','job.read','job.history.read','job.export','job.create','job.record.add','job.record.correct','job.record.delete','job.record.redate','job.deactivate','job.reactivate','job.lifecycle.pair')$actions$;
  END IF;
  IF extension IS NOT NULL THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR (%s))',c.conname,substring(c.def from 7),extension);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE access.subject_grant ADD CHECK((action NOT LIKE 'jobfamily.%' AND action NOT LIKE 'job.%') OR scope_kind='enterprise');

-- +goose Down
-- Persistent facts must never be removed by an automatic rollback.
SELECT 1/0;
