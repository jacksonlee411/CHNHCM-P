-- +goose Up
CREATE TABLE staffing.position_hierarchy (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, purpose text NOT NULL CHECK(purpose IN ('reporting','functional')),
 id uuid NOT NULL, PRIMARY KEY(instance_id,id), UNIQUE(instance_id,enterprise_id,purpose)
);
CREATE TABLE staffing.position_hierarchy_version (
 instance_id uuid NOT NULL, hierarchy_id uuid NOT NULL, source_id uuid NOT NULL, version_no bigint NOT NULL CHECK(version_no>0),
 operator jsonb NOT NULL, recorded_at timestamptz NOT NULL, submission_id uuid NOT NULL,
 PRIMARY KEY(instance_id,hierarchy_id,source_id), UNIQUE(instance_id,hierarchy_id,version_no),
 FOREIGN KEY(instance_id,hierarchy_id) REFERENCES staffing.position_hierarchy(instance_id,id)
);
CREATE TABLE staffing.position_hierarchy_node (
 instance_id uuid NOT NULL, hierarchy_id uuid NOT NULL, source_id uuid NOT NULL, position_id uuid NOT NULL, position_source uuid NOT NULL,
 PRIMARY KEY(instance_id,hierarchy_id,source_id,position_id),
 FOREIGN KEY(instance_id,position_id,position_source) REFERENCES staffing.position_source(instance_id,object_id,source_id),
 FOREIGN KEY(instance_id,hierarchy_id,source_id) REFERENCES staffing.position_hierarchy_version(instance_id,hierarchy_id,source_id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE staffing.position_hierarchy_edge (
 instance_id uuid NOT NULL, hierarchy_id uuid NOT NULL, source_id uuid NOT NULL, child_id uuid NOT NULL, parent_id uuid,
 period daterange NOT NULL CHECK(NOT isempty(period) AND NOT lower_inf(period)),
 PRIMARY KEY(instance_id,hierarchy_id,source_id,child_id,period),
 FOREIGN KEY(instance_id,hierarchy_id,source_id) REFERENCES staffing.position_hierarchy_version(instance_id,hierarchy_id,source_id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(instance_id,hierarchy_id,source_id,child_id) REFERENCES staffing.position_hierarchy_node(instance_id,hierarchy_id,source_id,position_id),
 FOREIGN KEY(instance_id,hierarchy_id,source_id,parent_id) REFERENCES staffing.position_hierarchy_node(instance_id,hierarchy_id,source_id,position_id)
);
CREATE TABLE staffing.position_hierarchy_outbox (
 instance_id uuid NOT NULL, hierarchy_id uuid NOT NULL, source_id uuid NOT NULL, event jsonb NOT NULL,
 PRIMARY KEY(instance_id,hierarchy_id,source_id),
 FOREIGN KEY(instance_id,hierarchy_id,source_id) REFERENCES staffing.position_hierarchy_version(instance_id,hierarchy_id,source_id)
);
CREATE TABLE staffing.position_hierarchy_terminal (LIKE staffing.position_terminal INCLUDING ALL);
CREATE TRIGGER immutable_root BEFORE UPDATE OR DELETE ON staffing.position_hierarchy FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_version BEFORE UPDATE OR DELETE ON staffing.position_hierarchy_version FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_node BEFORE UPDATE OR DELETE ON staffing.position_hierarchy_node FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_edge BEFORE UPDATE OR DELETE ON staffing.position_hierarchy_edge FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON staffing.position_hierarchy_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON staffing.position_hierarchy_terminal FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
-- +goose StatementBegin
CREATE FUNCTION staffing.reject_late_hierarchy_member() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM staffing.position_hierarchy_version WHERE (instance_id,hierarchy_id,source_id)=(NEW.instance_id,NEW.hierarchy_id,NEW.source_id)) THEN
  RAISE EXCEPTION 'published hierarchy is immutable';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER sealed_version BEFORE INSERT ON staffing.position_hierarchy_node FOR EACH ROW EXECUTE FUNCTION staffing.reject_late_hierarchy_member();
CREATE TRIGGER sealed_version BEFORE INSERT ON staffing.position_hierarchy_edge FOR EACH ROW EXECUTE FUNCTION staffing.reject_late_hierarchy_member();
GRANT SELECT,INSERT ON staffing.position_hierarchy,staffing.position_hierarchy_version,staffing.position_hierarchy_node,staffing.position_hierarchy_edge,staffing.position_hierarchy_terminal,staffing.position_hierarchy_outbox TO chnhcm_app;
-- +goose StatementBegin
DO $$ DECLARE c record; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  IF position('read_fields' IN c.def)>0 AND position('action' IN c.def)>0 THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR (action LIKE ''positionhierarchy.%%'' AND cardinality(read_fields)=0 AND cardinality(write_fields)=0))',c.conname,substring(c.def from 7));
  ELSIF position('bu.read' IN c.def)>0 AND position('action' IN c.def)>0 THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR action IN (''positionhierarchy.read'',''positionhierarchy.history.read'',''positionhierarchy.publish''))',c.conname,substring(c.def from 7));
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE access.subject_grant ADD CHECK(action NOT LIKE 'positionhierarchy.%' OR (scope_kind='enterprise' AND cardinality(read_fields)=0 AND cardinality(write_fields)=0));
UPDATE staffing.storage_contract SET contract='staffing-hierarchy-1' WHERE singleton AND contract='staffing-position-1';
-- +goose Down
SELECT 1/0;
