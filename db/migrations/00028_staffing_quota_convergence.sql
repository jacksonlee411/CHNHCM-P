-- +goose Up
-- Canonical predecessor: main's position reporting and position quota.
-- The upgrader handles the independently delivered allocation predecessor in this same goose transaction.
ALTER TABLE staffing.quota_source RENAME TO position_quota_source_27;
ALTER TABLE staffing.quota_record RENAME TO position_quota_record_27;
ALTER TABLE staffing.quota_outbox RENAME TO position_quota_outbox_27;

CREATE TABLE staffing.quota (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, id uuid NOT NULL,
 target_kind text NOT NULL CHECK(target_kind IN ('department','position')), target_id uuid NOT NULL, source_id uuid NOT NULL,
 position_ref uuid GENERATED ALWAYS AS (CASE WHEN target_kind='position' THEN target_id END) STORED,
 PRIMARY KEY(instance_id,id), UNIQUE(instance_id,enterprise_id,target_kind,target_id),
 FOREIGN KEY(instance_id,position_ref) REFERENCES staffing.position(instance_id,id)
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
CREATE TABLE staffing.quota_outbox (
 instance_id uuid NOT NULL, object_id uuid NOT NULL, source_id uuid NOT NULL, event jsonb NOT NULL,
 PRIMARY KEY(instance_id,object_id,source_id), FOREIGN KEY(instance_id,object_id,source_id) REFERENCES staffing.quota_source
);
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON staffing.quota_source FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON staffing.quota_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.quota,staffing.quota_source,staffing.quota_outbox TO chnhcm_app;
GRANT UPDATE(source_id) ON staffing.quota TO chnhcm_app;
GRANT SELECT,INSERT,DELETE ON staffing.quota_record TO chnhcm_app;

-- The original position ID also identifies the migrated root, including existing receipts and deep links.
INSERT INTO staffing.quota(instance_id,enterprise_id,id,target_kind,target_id,source_id)
SELECT DISTINCT ON (s.instance_id,s.position_id) s.instance_id,p.enterprise_id,s.position_id,'position',s.position_id,s.source_id
FROM staffing.position_quota_source_27 s JOIN staffing.position p ON (p.instance_id,p.id)=(s.instance_id,s.position_id)
ORDER BY s.instance_id,s.position_id,s.source_no DESC;

WITH origins AS (
 SELECT DISTINCT ON (r.instance_id,r.position_id,r.revision) r.instance_id,r.position_id,r.revision,s.source_id,s.subject_id,s.action
 FROM staffing.position_quota_record_27 r JOIN staffing.position_quota_source_27 s USING(instance_id,position_id,source_id)
 ORDER BY r.instance_id,r.position_id,r.revision,s.source_no
), records AS (
 SELECT r.instance_id,r.position_id,r.source_id,
 jsonb_agg(jsonb_build_object('recordId',r.record_id,'revision',r.revision,'effectiveDate',lower(r.period),
 'quota',r.quantity,'reason',r.reason,'audit',jsonb_build_object('recordedAt',r.recorded_at,'displayName',r.operator_name,
 'sourceVersion',o.source_id,'subjectId',o.subject_id,'action',o.action)) ORDER BY lower(r.period)) AS records,
 jsonb_agg(to_jsonb(r) ORDER BY lower(r.period)) AS original_records
 FROM staffing.position_quota_record_27 r JOIN origins o USING(instance_id,position_id,revision)
 GROUP BY r.instance_id,r.position_id,r.source_id
)
INSERT INTO staffing.quota_source
SELECT s.instance_id,s.position_id,s.source_id,s.source_no,s.submission_id,s.subject_id,
 CASE s.action WHEN 'SET' THEN CASE WHEN s.previous_source IS NULL THEN 'create' ELSE 'adjust' END ELSE lower(s.action) END,
 s.target_record,s.recorded_at,jsonb_build_object('quotaId',s.position_id,'positionId',s.position_id,
 'positionSource',s.position_source,'sourceVersion',s.source_id,'sourceNumber',s.source_no,'records',COALESCE(r.records,'[]'::jsonb),
 'originalSource',to_jsonb(s),'originalRecords',COALESCE(r.original_records,'[]'::jsonb)),
 jsonb_build_object('subjectId',s.subject_id,'displayName',s.operator_name)
FROM staffing.position_quota_source_27 s LEFT JOIN records r USING(instance_id,position_id,source_id);

INSERT INTO staffing.quota_record(instance_id,object_id,record_id,revision,start_date,end_date,amount)
SELECT r.instance_id,r.position_id,r.record_id,r.revision,lower(r.period),upper(r.period),r.quantity
FROM staffing.position_quota_record_27 r JOIN staffing.quota q
 ON (q.instance_id,q.id,q.source_id)=(r.instance_id,r.position_id,r.source_id);
INSERT INTO staffing.quota_outbox SELECT instance_id,position_id,source_id,event FROM staffing.position_quota_outbox_27;

-- Compare immutable originals to the contents of the new history before dropping the staging tables.
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT to_jsonb(o) FROM staffing.position_quota_source_27 o EXCEPT ALL
           SELECT snapshot->'originalSource' FROM staffing.quota_source)
 OR EXISTS(SELECT to_jsonb(r) FROM staffing.position_quota_record_27 r EXCEPT ALL
           SELECT jsonb_array_elements(snapshot->'originalRecords') FROM staffing.quota_source)
 OR EXISTS(SELECT instance_id,position_id,source_id,event FROM staffing.position_quota_outbox_27 EXCEPT ALL
           SELECT instance_id,object_id,source_id,event FROM staffing.quota_outbox) THEN
  RAISE EXCEPTION 'quota conversion would lose a published fact';
 END IF;
END $$;
-- +goose StatementEnd
DROP TABLE staffing.position_quota_outbox_27,staffing.position_quota_record_27,staffing.position_quota_source_27;
DROP FUNCTION staffing.seal_quota_records();
-- +goose Down
SELECT 1/0;
