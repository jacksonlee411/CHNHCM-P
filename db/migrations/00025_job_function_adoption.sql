-- +goose Up
-- The upgrader runs after old writers have drained. Capture the same committed
-- cut for all instances, including records visible only in past sources.
LOCK TABLE staffing.job, staffing.job_source, staffing.job_revision IN SHARE MODE;
CREATE TABLE staffing.job_function_legacy (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL,
 record_id uuid NOT NULL, revision uuid NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id,record_id,revision),
 FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.job_source(instance_id,object_id,source_id),
 FOREIGN KEY(instance_id,object_id,record_id,revision) REFERENCES staffing.job_revision(instance_id,object_id,record_id,revision)
);
INSERT INTO staffing.job_function_legacy
 SELECT s.instance_id,s.object_id,s.source_id,(r->>'id')::uuid,(r->>'revision')::uuid
 FROM staffing.job_source s CROSS JOIN LATERAL jsonb_array_elements(s.snapshot->'records') r;
CREATE TRIGGER immutable_legacy BEFORE UPDATE OR DELETE ON staffing.job_function_legacy
 FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT ON staffing.job_function_legacy TO chnhcm_app;
UPDATE staffing.job_read r SET fact=jsonb_set(r.fact,'{functionEvidence}',
 jsonb_build_object('sourceVersion',l.source_id,'recordId',l.record_id,'revision',l.revision))
 FROM staffing.job_function_legacy l
 WHERE (l.instance_id,l.object_id,l.source_id,l.record_id,l.revision)=
 (r.instance_id,r.object_id,r.source_id,r.record_id,(r.fact->>'revision')::uuid);
-- Reverse protection reads decided Job heads even during read-index repair.
CREATE INDEX job_function_uses ON staffing.job USING gin
 (jsonb_path_query_array(head,'$.records[*].data.jobFunctionId') jsonb_path_ops);
-- Admit this field independently of the JobFamily grant. Existing subjects
-- keep their exact permissions until explicitly granted the new dimension.
-- +goose StatementBegin
DO $$ DECLARE c record; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint
 WHERE conrelid='access.subject_grant'::regclass AND contype='c'
 AND position('jobFamilyId' IN pg_get_constraintdef(oid))>0 LOOP
  EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
  EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR
   (action LIKE ''job.%%'' AND read_fields <@ ARRAY[''code'',''name'',''nameEn'',''effectiveDate'',''operationType'',''operationReason'',''jobFamilyId'',''jobFunctionId'',''responsibilities'',''requirements'',''status'',''gradeIds'']::text[] AND
   write_fields <@ ARRAY[''code'',''name'',''nameEn'',''effectiveDate'',''operationType'',''operationReason'',''jobFamilyId'',''jobFunctionId'',''responsibilities'',''requirements'',''gradeIds'']::text[]))', c.conname,substring(c.def from 7));
 END LOOP;
END $$;
-- +goose StatementEnd
UPDATE staffing.storage_contract SET contract='staffing-job-classification-1' WHERE singleton AND contract='staffing-job-function-1';

-- +goose Down
SELECT 1/0;
