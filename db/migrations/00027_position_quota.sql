-- +goose Up
CREATE TABLE staffing.quota_control (
 instance_id uuid NOT NULL, position_id uuid NOT NULL, position_source uuid NOT NULL, controlled_from date NOT NULL,
 PRIMARY KEY(instance_id,position_id),
 FOREIGN KEY(instance_id,position_id,position_source) REFERENCES staffing.position_source(instance_id,object_id,source_id)
);
CREATE TABLE staffing.quota_source (
 instance_id uuid NOT NULL, position_id uuid NOT NULL, source_id uuid NOT NULL, source_no bigint NOT NULL CHECK (source_no>0),
 previous_source uuid, submission_id uuid NOT NULL, subject_id uuid NOT NULL, position_source uuid NOT NULL,
 action text NOT NULL, reason text NOT NULL, operator_name text NOT NULL, recorded_at timestamptz NOT NULL,
 target_record uuid NOT NULL, previous_date date NOT NULL, effective_date date NOT NULL, previous_quantity integer, new_quantity integer,
 PRIMARY KEY(instance_id,position_id,source_id), UNIQUE(instance_id,position_id,source_no),
 FOREIGN KEY(instance_id,position_id,position_source) REFERENCES staffing.position_source(instance_id,object_id,source_id),
 FOREIGN KEY(instance_id,position_id,previous_source) REFERENCES staffing.quota_source(instance_id,position_id,source_id)
);
CREATE TABLE staffing.quota_record (
 instance_id uuid NOT NULL, position_id uuid NOT NULL, source_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 period daterange NOT NULL CHECK (NOT isempty(period) AND NOT lower_inf(period)), quantity integer CHECK(quantity>=0),
 reason text NOT NULL, operator_name text NOT NULL, recorded_at timestamptz NOT NULL,
 PRIMARY KEY(instance_id,position_id,source_id,record_id),
 FOREIGN KEY(instance_id,position_id,source_id) REFERENCES staffing.quota_source(instance_id,position_id,source_id),
 EXCLUDE USING gist(instance_id WITH =,position_id WITH =,source_id WITH =,period WITH &&)
);
CREATE TABLE staffing.quota_terminal (LIKE staffing.position_terminal INCLUDING ALL);
CREATE TABLE staffing.quota_outbox (
 instance_id uuid NOT NULL, position_id uuid NOT NULL, source_id uuid NOT NULL, event jsonb NOT NULL,
 PRIMARY KEY(instance_id,position_id,source_id),
 FOREIGN KEY(instance_id,position_id,source_id) REFERENCES staffing.quota_source(instance_id,position_id,source_id)
);
CREATE TRIGGER immutable_control BEFORE UPDATE OR DELETE ON staffing.quota_control FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON staffing.quota_source FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_record BEFORE UPDATE OR DELETE ON staffing.quota_record FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON staffing.quota_terminal FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON staffing.quota_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
-- +goose StatementBegin
CREATE FUNCTION staffing.seal_quota_records() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT FROM staffing.quota_outbox WHERE instance_id=NEW.instance_id AND position_id=NEW.position_id AND source_id=NEW.source_id) THEN
  RAISE EXCEPTION 'quota source already sealed';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER sealed_record BEFORE INSERT ON staffing.quota_record FOR EACH ROW EXECUTE FUNCTION staffing.seal_quota_records();
GRANT SELECT,INSERT ON staffing.quota_control,staffing.quota_source,staffing.quota_record,staffing.quota_terminal,staffing.quota_outbox TO chnhcm_app;

-- +goose StatementBegin
DO $$ DECLARE c record; extension text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  extension:=NULL;
  IF position('read_fields' IN c.def)>0 THEN extension:=$expr$(action LIKE 'quota.%' AND read_fields <@ ARRAY['quantity','effectiveDate','reason','operator']::text[] AND write_fields <@ ARRAY['quantity','effectiveDate','reason']::text[])$expr$;
  ELSIF position('bu.read' IN c.def)>0 AND position('action' IN c.def)>0 THEN extension:=$expr$action IN ('quota.read','quota.history.read','quota.change')$expr$;
  ELSIF position('scope_kind = ANY' IN c.def)>0 AND position('action' IN c.def)=0 THEN extension:=$expr$scope_kind='position' AND action LIKE 'quota.%'$expr$;
  END IF;
  IF extension IS NOT NULL THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR (%s))',c.conname,substring(c.def from 7),extension);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE access.subject_grant ADD CHECK(action NOT LIKE 'quota.%' OR (
 scope_kind IN ('enterprise','position')
 AND action IN ('quota.read','quota.history.read','quota.change')
 AND read_fields <@ ARRAY['quantity','effectiveDate','reason','operator']::text[]
 AND write_fields <@ ARRAY['quantity','effectiveDate','reason']::text[]
 AND (action='quota.change' OR cardinality(write_fields)=0)
));
UPDATE staffing.storage_contract SET contract='staffing-quota-1' WHERE singleton AND contract='staffing-classification-reporting-1';
-- +goose Down
SELECT 1/0;
