-- +goose Up
-- G1: Grade facts only. Existing Job/Family sources are untouched.
CREATE TABLE staffing.grade (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, id uuid NOT NULL, code text NOT NULL,
 source_id uuid NOT NULL, source_no bigint NOT NULL CHECK(source_no>0), first_date date NOT NULL,
 head jsonb NOT NULL, PRIMARY KEY(instance_id,id), UNIQUE(instance_id,enterprise_id,code),
 CHECK(code ~ '^[A-Z0-9][A-Z0-9_-]{0,29}$')
);
CREATE INDEX grade_enterprise_dates ON staffing.grade(instance_id,enterprise_id,first_date);
CREATE TABLE staffing.grade_source (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, source_no bigint NOT NULL,
 subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea NOT NULL CHECK(octet_length(digest)=32),
 snapshot jsonb NOT NULL, operation jsonb NOT NULL, operator jsonb NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,object_id,source_id), UNIQUE(instance_id,object_id,source_no),
 FOREIGN KEY(instance_id,object_id) REFERENCES staffing.grade(instance_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON staffing.grade_source FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
ALTER TABLE staffing.grade ADD FOREIGN KEY(instance_id,id,source_id) REFERENCES staffing.grade_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE staffing.grade_revision (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 source_id uuid NOT NULL, fact jsonb NOT NULL, PRIMARY KEY(instance_id,object_id,record_id,revision),
 FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.grade_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER immutable_revision BEFORE UPDATE OR DELETE ON staffing.grade_revision FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.grade_read (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, object_id uuid NOT NULL, record_id uuid NOT NULL,
 source_id uuid NOT NULL, start_date date NOT NULL, end_date date, active boolean NOT NULL, fact jsonb NOT NULL,
 valid_period daterange GENERATED ALWAYS AS (daterange(start_date,end_date,'[)')) STORED,
 PRIMARY KEY(instance_id,object_id,record_id), FOREIGN KEY(instance_id,object_id) REFERENCES staffing.grade(instance_id,id),
 CHECK(end_date IS NULL OR end_date>start_date)
);
CREATE INDEX grade_periods ON staffing.grade_read USING gist(valid_period);
CREATE TABLE staffing.grade_terminal (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea,
 result jsonb NOT NULL, closed_at timestamptz NOT NULL DEFAULT clock_timestamp(), PRIMARY KEY(instance_id,subject_id,submission_id)
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON staffing.grade_terminal FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.grade_outbox (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, event jsonb NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id), FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.grade_source(instance_id,object_id,source_id)
);
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON staffing.grade_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.grade_selection (
 instance_id uuid NOT NULL, session_id uuid NOT NULL, id uuid NOT NULL, revision uuid NOT NULL,
 business_date date NOT NULL, query jsonb NOT NULL, members uuid[] NOT NULL, excluded uuid[] NOT NULL DEFAULT '{}',
 expires_at timestamptz NOT NULL, cancelled boolean NOT NULL DEFAULT false,
 PRIMARY KEY(instance_id,session_id,id)
);
GRANT SELECT,INSERT ON staffing.grade,staffing.grade_source,staffing.grade_revision,staffing.grade_terminal,staffing.grade_outbox TO chnhcm_app;
GRANT UPDATE(source_id,source_no,first_date,head) ON staffing.grade TO chnhcm_app;
GRANT SELECT,INSERT,UPDATE,DELETE ON staffing.grade_read,staffing.grade_selection TO chnhcm_app;

-- Append Grade predicates without altering any existing grants or facts.
-- +goose StatementBegin
DO $$ DECLARE c record; extension text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  extension:=NULL;
  IF position('read_fields' IN c.def)>0 THEN extension:=$fields$(action LIKE 'grade.%' AND read_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','description','status']::text[] AND write_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','description']::text[])$fields$;
  ELSIF position('bu.read' IN c.def)>0 AND position('action' IN c.def)>0 THEN extension:=$actions$action IN ('grade.read','grade.history.read','grade.export','grade.create','grade.record.add','grade.record.correct','grade.record.delete','grade.record.redate','grade.deactivate','grade.reactivate','grade.lifecycle.pair')$actions$;
  END IF;
  IF extension IS NOT NULL THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR (%s))',c.conname,substring(c.def from 7),extension);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE access.subject_grant ADD CHECK(action NOT LIKE 'grade.%' OR scope_kind='enterprise');
UPDATE staffing.storage_contract SET contract='staffing-grade-1' WHERE singleton AND contract='staffing-job-foundation-1';

-- +goose Down
-- Sources and receipts are immutable; downgrades must not discard them.
SELECT 1/0;
