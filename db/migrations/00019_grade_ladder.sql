-- +goose Up
-- G2: GradeLadder facts and immutable per-record membership manifests.
CREATE TABLE staffing.grade_ladder (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, id uuid NOT NULL, code text NOT NULL,
 source_id uuid NOT NULL, source_no bigint NOT NULL CHECK(source_no>0), first_date date NOT NULL,
 head jsonb NOT NULL, PRIMARY KEY(instance_id,id), UNIQUE(instance_id,enterprise_id,code),
 CHECK(code ~ '^[A-Z0-9][A-Z0-9_-]{0,29}$')
);
CREATE INDEX grade_ladder_enterprise_dates ON staffing.grade_ladder(instance_id,enterprise_id,first_date);
CREATE TABLE staffing.grade_ladder_source (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, source_no bigint NOT NULL,
 subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea NOT NULL CHECK(octet_length(digest)=32),
 snapshot jsonb NOT NULL, operation jsonb NOT NULL, operator jsonb NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,object_id,source_id), UNIQUE(instance_id,object_id,source_no),
 FOREIGN KEY(instance_id,object_id) REFERENCES staffing.grade_ladder(instance_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON staffing.grade_ladder_source FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
ALTER TABLE staffing.grade_ladder ADD FOREIGN KEY(instance_id,id,source_id) REFERENCES staffing.grade_ladder_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE staffing.grade_ladder_revision (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 source_id uuid NOT NULL, fact jsonb NOT NULL, PRIMARY KEY(instance_id,object_id,record_id,revision),
 FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.grade_ladder_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER immutable_revision BEFORE UPDATE OR DELETE ON staffing.grade_ladder_revision FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.grade_ladder_read (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, object_id uuid NOT NULL, record_id uuid NOT NULL,
 source_id uuid NOT NULL, start_date date NOT NULL, end_date date, active boolean NOT NULL, fact jsonb NOT NULL,
 valid_period daterange GENERATED ALWAYS AS (daterange(start_date,end_date,'[)')) STORED,
 PRIMARY KEY(instance_id,object_id,record_id), FOREIGN KEY(instance_id,object_id) REFERENCES staffing.grade_ladder(instance_id,id),
 CHECK(end_date IS NULL OR end_date>start_date)
);
CREATE INDEX grade_ladder_periods ON staffing.grade_ladder_read USING gist(valid_period);
CREATE TABLE staffing.grade_ladder_terminal (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea,
 result jsonb NOT NULL, closed_at timestamptz NOT NULL DEFAULT clock_timestamp(), PRIMARY KEY(instance_id,subject_id,submission_id)
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON staffing.grade_ladder_terminal FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.grade_ladder_outbox (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, event jsonb NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id), FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.grade_ladder_source(instance_id,object_id,source_id)
);
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON staffing.grade_ladder_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.grade_ladder_selection (
 instance_id uuid NOT NULL, session_id uuid NOT NULL, id uuid NOT NULL, revision uuid NOT NULL,
 business_date date NOT NULL, query jsonb NOT NULL, members uuid[] NOT NULL, excluded uuid[] NOT NULL DEFAULT '{}',
 expires_at timestamptz NOT NULL, cancelled boolean NOT NULL DEFAULT false,
 PRIMARY KEY(instance_id,session_id,id)
);
GRANT SELECT,INSERT ON staffing.grade_ladder,staffing.grade_ladder_source,staffing.grade_ladder_revision,staffing.grade_ladder_terminal,staffing.grade_ladder_outbox TO chnhcm_app;
GRANT UPDATE(source_id,source_no,first_date,head) ON staffing.grade_ladder TO chnhcm_app;
GRANT SELECT,INSERT,UPDATE,DELETE ON staffing.grade_ladder_read,staffing.grade_ladder_selection TO chnhcm_app;


CREATE TABLE staffing.job_grade_manifest (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, record_id uuid NOT NULL,
 member_count integer NOT NULL CHECK(member_count >=0), digest bytea NOT NULL CHECK(octet_length(digest)=32),
 PRIMARY KEY(instance_id,object_id,source_id,record_id),
 FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.job_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE staffing.job_grade_member (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, record_id uuid NOT NULL,
 grade_id uuid NOT NULL, position integer NOT NULL CHECK(position>0), business_at date NOT NULL,
 adopted_source uuid NOT NULL, adopted_record uuid NOT NULL, adopted_revision uuid NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id,record_id,grade_id),
 UNIQUE(instance_id,object_id,source_id,record_id,position),
 FOREIGN KEY(instance_id,object_id,source_id,record_id) REFERENCES staffing.job_grade_manifest(instance_id,object_id,source_id,record_id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(instance_id,grade_id,adopted_source) REFERENCES staffing.grade_source(instance_id,object_id,source_id),
 FOREIGN KEY(instance_id,grade_id,adopted_record,adopted_revision) REFERENCES staffing.grade_revision(instance_id,object_id,record_id,revision)
);
CREATE INDEX job_grade_reverse ON staffing.job_grade_member(instance_id,grade_id,object_id,source_id,record_id);
CREATE TRIGGER immutable_manifest BEFORE UPDATE OR DELETE ON staffing.job_grade_manifest FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_member BEFORE UPDATE OR DELETE ON staffing.job_grade_member FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.job_grade_manifest,staffing.job_grade_member TO chnhcm_app;

CREATE TABLE staffing.grade_ladder_grade_manifest (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, record_id uuid NOT NULL,
 member_count integer NOT NULL CHECK(member_count >0), digest bytea NOT NULL CHECK(octet_length(digest)=32),
 PRIMARY KEY(instance_id,object_id,source_id,record_id),
 FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.grade_ladder_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE staffing.grade_ladder_grade_member (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, record_id uuid NOT NULL,
 grade_id uuid NOT NULL, position integer NOT NULL CHECK(position>0), business_at date NOT NULL,
 adopted_source uuid NOT NULL, adopted_record uuid NOT NULL, adopted_revision uuid NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id,record_id,grade_id),
 UNIQUE(instance_id,object_id,source_id,record_id,position),
 FOREIGN KEY(instance_id,object_id,source_id,record_id) REFERENCES staffing.grade_ladder_grade_manifest(instance_id,object_id,source_id,record_id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(instance_id,grade_id,adopted_source) REFERENCES staffing.grade_source(instance_id,object_id,source_id),
 FOREIGN KEY(instance_id,grade_id,adopted_record,adopted_revision) REFERENCES staffing.grade_revision(instance_id,object_id,record_id,revision)
);
CREATE INDEX grade_ladder_grade_reverse ON staffing.grade_ladder_grade_member(instance_id,grade_id,object_id,source_id,record_id);
CREATE TRIGGER immutable_manifest BEFORE UPDATE OR DELETE ON staffing.grade_ladder_grade_manifest FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_member BEFORE UPDATE OR DELETE ON staffing.grade_ladder_grade_member FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.grade_ladder_grade_manifest,staffing.grade_ladder_grade_member TO chnhcm_app;

-- Every pre-G2 Job source had no Grade relationship. Only add a sealed empty
-- manifest; do not UPDATE the snapshot, revision, adoption, receipt or event.
INSERT INTO staffing.job_grade_manifest(instance_id,object_id,source_id,record_id,member_count,digest)
 SELECT s.instance_id,s.object_id,s.source_id,(r->>'id')::uuid,0,
 decode('4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945','hex')
 FROM staffing.job_source s CROSS JOIN LATERAL jsonb_array_elements(s.snapshot->'records') r;

-- Extend permitted grants without silently expanding existing subjects' rights.
-- +goose StatementBegin
DO $$ DECLARE c record; extension text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  extension:=NULL;
  IF position('read_fields' IN c.def)>0 THEN extension:=$fields$
   (action LIKE 'gradeladder.%' AND read_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','description','status','gradeIds']::text[] AND write_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','description','gradeIds']::text[])
   OR (action LIKE 'job.%' AND read_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','jobFamilyId','responsibilities','requirements','status','gradeIds']::text[] AND write_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason','jobFamilyId','responsibilities','requirements','gradeIds']::text[])$fields$;
  ELSIF position('bu.read' IN c.def)>0 AND position('action' IN c.def)>0 THEN extension:=$actions$action IN ('gradeladder.read','gradeladder.history.read','gradeladder.export','gradeladder.create','gradeladder.record.add','gradeladder.record.correct','gradeladder.record.delete','gradeladder.record.redate','gradeladder.deactivate','gradeladder.reactivate','gradeladder.lifecycle.pair')$actions$;
  END IF;
  IF extension IS NOT NULL THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR (%s))',c.conname,substring(c.def from 7),extension);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE access.subject_grant ADD CHECK(action NOT LIKE 'gradeladder.%' OR scope_kind='enterprise');
UPDATE staffing.storage_contract SET contract='staffing-grade-ladder-1' WHERE singleton AND contract='staffing-grade-1';

-- +goose Down
-- Sources, members and receipts are immutable. Never discard them on downgrade.
SELECT 1/0;
