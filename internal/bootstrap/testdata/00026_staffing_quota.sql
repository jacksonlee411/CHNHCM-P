-- +goose Up
CREATE TABLE staffing.quota (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, id uuid NOT NULL,
 target_kind text NOT NULL CHECK(target_kind='position'), target_id uuid NOT NULL, source_id uuid NOT NULL,
 PRIMARY KEY(instance_id,id), UNIQUE(instance_id,enterprise_id,target_kind,target_id),
 FOREIGN KEY(instance_id,target_id) REFERENCES staffing.position(instance_id,id)
);
CREATE TABLE staffing.quota_source (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, source_no bigint NOT NULL CHECK(source_no>0),
 submission_id uuid NOT NULL, subject_id uuid NOT NULL, action text NOT NULL, record_id uuid NOT NULL,
 recorded_at timestamptz NOT NULL, snapshot jsonb NOT NULL, operator jsonb NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id), UNIQUE(instance_id,object_id,source_no),
 FOREIGN KEY(instance_id,object_id) REFERENCES staffing.quota
);
ALTER TABLE staffing.quota ADD FOREIGN KEY(instance_id,id,source_id)
 REFERENCES staffing.quota_source(instance_id,object_id,source_id) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE staffing.quota_record (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 start_date date NOT NULL, end_date date, amount bigint CHECK(amount>=0),
 span daterange GENERATED ALWAYS AS (daterange(start_date,end_date,'[)')) STORED,
 PRIMARY KEY(instance_id,object_id,record_id), UNIQUE(instance_id,object_id,start_date),
 FOREIGN KEY(instance_id,object_id) REFERENCES staffing.quota, CHECK(end_date IS NULL OR end_date>start_date),
 EXCLUDE USING gist(instance_id WITH =,object_id WITH =,span WITH &&)
);
CREATE TABLE staffing.quota_terminal (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea,
 result jsonb NOT NULL, closed_at timestamptz NOT NULL DEFAULT clock_timestamp(), PRIMARY KEY(instance_id,subject_id,submission_id)
);
CREATE TABLE staffing.quota_outbox (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, event jsonb NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id), FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.quota_source
);
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON staffing.quota_source FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON staffing.quota_terminal FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON staffing.quota_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.quota,staffing.quota_source,staffing.quota_terminal,staffing.quota_outbox TO chnhcm_app;
GRANT UPDATE(source_id) ON staffing.quota TO chnhcm_app;
GRANT SELECT,INSERT,DELETE ON staffing.quota_record TO chnhcm_app;

ALTER TABLE access.subject_grant DROP CONSTRAINT subject_grant_scope_kind_check;
ALTER TABLE access.subject_grant ADD CHECK(scope_kind IN ('enterprise','bu','department','instance') OR (scope_kind='position' AND action LIKE 'staffingquota.%'));
-- +goose StatementBegin
DO $$ DECLARE c record; extension text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  extension:=NULL;
  IF position('jobfamily.' IN c.def)>0 AND position('read_fields' IN c.def)>0 THEN extension:=$fields$(action LIKE 'staffingquota.%' AND read_fields <@ ARRAY['code','name','departmentId','status','effectiveDate','quota','reason','audit','usageState','sourceVersion']::text[] AND write_fields <@ ARRAY['effectiveDate','quota','reason']::text[])$fields$;
  ELSIF position('jobfamily.read' IN c.def)>0 AND position('action' IN c.def)>0 THEN extension:=$actions$action IN ('staffingquota.read','staffingquota.history.read','staffingquota.create','staffingquota.adjust','staffingquota.correct','staffingquota.redate','staffingquota.delete','staffingquota.withdraw')$actions$;
  END IF;
  IF extension IS NOT NULL THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR (%s))',c.conname,substring(c.def from 7),extension);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE access.subject_grant ADD CHECK(action NOT LIKE 'staffingquota.%' OR scope_kind IN ('enterprise','position'));
UPDATE staffing.storage_contract SET contract='staffing-quota-1' WHERE singleton AND contract='staffing-job-classification-1';
-- +goose Down
SELECT 1/0;
