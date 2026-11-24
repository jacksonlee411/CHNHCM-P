-- +goose Up
CREATE TABLE staffing.position_intake (
    instance_id uuid NOT NULL, position_id uuid NOT NULL, source_id uuid NOT NULL,
    period daterange NOT NULL CHECK(NOT isempty(period) AND NOT lower_inf(period)),
    PRIMARY KEY(instance_id,position_id),
    FOREIGN KEY(instance_id,position_id,source_id) REFERENCES staffing.position_source(instance_id,object_id,source_id)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.position_intake FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.position_replenishment (
    instance_id uuid NOT NULL,position_id uuid NOT NULL,revision uuid NOT NULL,
    actor uuid NOT NULL, business_date date NOT NULL, paused boolean NOT NULL,
    reason text NOT NULL CHECK(length(trim(reason)) BETWEEN 1 AND 1000),
    digest text NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY(instance_id,position_id,revision),UNIQUE(instance_id,actor,revision),
    UNIQUE(instance_id,position_id,business_date),
    FOREIGN KEY(instance_id,position_id) REFERENCES staffing.position(instance_id,id)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.position_replenishment FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.position_intake,staffing.position_replenishment TO chnhcm_app;
CREATE INDEX assignment_segment_position_use ON staffing.assignment_segment(instance_id,(adoption->>'position')) WHERE coalesce(adoption->>'position','')<>'';

-- +goose StatementBegin
DO $$ DECLARE c record; expr text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint
 WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  expr:=substring(c.def from 7);
  IF position('jobfamily.read' IN expr)>0 OR (position('jobfamily.' IN expr)>0 AND position('read_fields' IN expr)>0) THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR (%s))',c.conname,expr,
    $p$scope_kind='enterprise' AND action='replenishment.maintain' AND read_fields=ARRAY['rule']::text[] AND write_fields=ARRAY['rule']::text[]$p$);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields)
SELECT DISTINCT g.instance_id,gen_random_uuid(),g.principal_id,'enterprise',g.scope_id,'replenishment.maintain',ARRAY['rule'],ARRAY['rule']
FROM access.subject_grant g JOIN access.principal p USING(instance_id,principal_id)
WHERE p.normalized_login='admin' AND g.scope_kind='enterprise' AND g.action='position.record.add' AND g.write_fields @> ARRAY['effectiveDate'];
UPDATE access.policy_guard p SET policy_no=policy_no+1 WHERE EXISTS (SELECT FROM access.subject_grant g
 WHERE (g.instance_id,g.principal_id)=(p.instance_id,p.principal_id) AND g.action='replenishment.maintain');
-- +goose Down
SELECT 1/0;
