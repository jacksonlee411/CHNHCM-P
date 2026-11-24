-- +goose Up

CREATE TABLE organization.legal_purpose_source (
 instance_id uuid NOT NULL, owner_id uuid NOT NULL, version uuid NOT NULL, sequence bigint GENERATED ALWAYS AS IDENTITY,
 created_xid xid8 NOT NULL DEFAULT pg_current_xact_id(), subject_id uuid NOT NULL, submission_id uuid NOT NULL, input_digest bytea NOT NULL CHECK(octet_length(input_digest)=32),
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(), FOREIGN KEY(instance_id,owner_id) REFERENCES organization.legal_entity(instance_id,legal_entity_id),
 PRIMARY KEY(instance_id,owner_id,version), UNIQUE(instance_id,subject_id,submission_id)
);
CREATE INDEX legal_purpose_head ON organization.legal_purpose_source(instance_id,owner_id,sequence DESC);
CREATE TABLE organization.legal_purpose_member (
 instance_id uuid NOT NULL, owner_id uuid NOT NULL, version uuid NOT NULL, record_id uuid NOT NULL,
 revision uuid NOT NULL, period daterange NOT NULL CHECK(NOT isempty(period) AND lower_inc(period) AND NOT upper_inc(period)),
 fact jsonb NOT NULL, PRIMARY KEY(instance_id,owner_id,version,record_id),
 FOREIGN KEY(instance_id,owner_id,version) REFERENCES organization.legal_purpose_source
);
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON organization.legal_purpose_source FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
CREATE TRIGGER immutable_member BEFORE UPDATE OR DELETE ON organization.legal_purpose_member FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();

-- Close membership at the same commit as the source and its publication.
-- +goose StatementBegin
CREATE FUNCTION organization.legal_purpose_member_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT FROM organization.legal_purpose_source s WHERE (s.instance_id,s.owner_id,s.version)=(NEW.instance_id,NEW.owner_id,NEW.version)
              AND s.created_xid=pg_current_xact_id()) THEN
   RAISE EXCEPTION 'published configuration membership is sealed';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER sealed_member BEFORE INSERT ON organization.legal_purpose_member FOR EACH ROW EXECUTE FUNCTION organization.legal_purpose_member_guard();
CREATE TABLE organization.legal_purpose_outbox (
 instance_id uuid NOT NULL,owner_id uuid NOT NULL,version uuid NOT NULL,
 PRIMARY KEY(instance_id,owner_id,version), FOREIGN KEY(instance_id,owner_id,version) REFERENCES organization.legal_purpose_source
);
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON organization.legal_purpose_outbox FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
GRANT SELECT,INSERT ON organization.legal_purpose_outbox TO chnhcm_app;
GRANT SELECT,INSERT ON organization.legal_purpose_source,organization.legal_purpose_member TO chnhcm_app;
GRANT USAGE ON SEQUENCE organization.legal_purpose_source_sequence_seq TO chnhcm_app;


CREATE TABLE staffing.assignment_policy_source (
 instance_id uuid NOT NULL, owner_id uuid NOT NULL, version uuid NOT NULL, sequence bigint GENERATED ALWAYS AS IDENTITY,
 created_xid xid8 NOT NULL DEFAULT pg_current_xact_id(), subject_id uuid NOT NULL, submission_id uuid NOT NULL, input_digest bytea NOT NULL CHECK(octet_length(input_digest)=32),
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,owner_id,version), UNIQUE(instance_id,subject_id,submission_id)
);
CREATE INDEX assignment_policy_head ON staffing.assignment_policy_source(instance_id,owner_id,sequence DESC);
CREATE TABLE staffing.assignment_policy_member (
 instance_id uuid NOT NULL, owner_id uuid NOT NULL, version uuid NOT NULL, record_id uuid NOT NULL,
 revision uuid NOT NULL, period daterange NOT NULL CHECK(NOT isempty(period) AND lower_inc(period) AND NOT upper_inc(period)),
 fact jsonb NOT NULL, PRIMARY KEY(instance_id,owner_id,version,record_id),
 FOREIGN KEY(instance_id,owner_id,version) REFERENCES staffing.assignment_policy_source
);
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON staffing.assignment_policy_source FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_member BEFORE UPDATE OR DELETE ON staffing.assignment_policy_member FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();

-- Close membership at the same commit as the source and its publication.
-- +goose StatementBegin
CREATE FUNCTION staffing.assignment_policy_member_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT FROM staffing.assignment_policy_source s WHERE (s.instance_id,s.owner_id,s.version)=(NEW.instance_id,NEW.owner_id,NEW.version)
              AND s.created_xid=pg_current_xact_id()) THEN
   RAISE EXCEPTION 'published configuration membership is sealed';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER sealed_member BEFORE INSERT ON staffing.assignment_policy_member FOR EACH ROW EXECUTE FUNCTION staffing.assignment_policy_member_guard();
CREATE TABLE staffing.assignment_policy_outbox (
 instance_id uuid NOT NULL,owner_id uuid NOT NULL,version uuid NOT NULL,
 PRIMARY KEY(instance_id,owner_id,version), FOREIGN KEY(instance_id,owner_id,version) REFERENCES staffing.assignment_policy_source
);
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON staffing.assignment_policy_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.assignment_policy_outbox TO chnhcm_app;
GRANT SELECT,INSERT ON staffing.assignment_policy_source,staffing.assignment_policy_member TO chnhcm_app;
GRANT USAGE ON SEQUENCE staffing.assignment_policy_source_sequence_seq TO chnhcm_app;

-- Extend the action and field allowlists only for these enterprise-wide grants.
-- +goose StatementBegin
DO $$ DECLARE c record; expr text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) def FROM pg_constraint
 WHERE conrelid='access.subject_grant'::regclass AND contype='c' LOOP
  expr:=substring(c.def from 7);
  IF position('jobfamily.read' IN expr)>0 OR (position('jobfamily.' IN expr)>0 AND position('read_fields' IN expr)>0) THEN
   EXECUTE format('ALTER TABLE access.subject_grant DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE access.subject_grant ADD CONSTRAINT %I CHECK ((%s) OR (%s))',c.conname,expr,
    $p$scope_kind='enterprise' AND action IN ('legalpurpose.read','legalpurpose.history.read','legalpurpose.maintain','legalpurpose.check',
    'assignmentpolicy.read','assignmentpolicy.history.read','assignmentpolicy.maintain','assignmentpolicy.check')
    AND read_fields=ARRAY['rule']::text[] AND write_fields=CASE WHEN action LIKE '%.maintain' THEN ARRAY['rule'] ELSE '{}'::text[] END$p$);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
ALTER TABLE access.subject_grant ADD CONSTRAINT readiness_rule_scope CHECK
 ((action NOT LIKE 'legalpurpose.%' AND action NOT LIKE 'assignmentpolicy.%') OR scope_kind='enterprise');
-- Bootstrap only the two controlled local accounts, within existing enterprise grants.
INSERT INTO access.subject_grant(instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields)
SELECT g.instance_id,gen_random_uuid(),g.principal_id,g.scope_kind,g.scope_id,
 CASE WHEN starts_with(g.action,'job.') THEN 'assignmentpolicy.' ELSE 'legalpurpose.' END || a.action,
 ARRAY['rule'],CASE WHEN a.action='maintain' THEN ARRAY['rule'] ELSE '{}'::text[] END
FROM access.subject_grant g JOIN access.principal p USING(instance_id,principal_id)
CROSS JOIN LATERAL unnest(CASE WHEN g.action LIKE '%.record.correct' THEN ARRAY['maintain']
 WHEN g.action LIKE '%.history.read' THEN ARRAY['history.read'] ELSE ARRAY['read','check'] END) a(action)
WHERE g.scope_kind='enterprise' AND p.normalized_login IN ('admin','reader')
AND g.action IN ('job.read','job.history.read','legalentity.read','legalentity.history.read','job.record.correct','legalentity.record.correct')
AND (a.action<>'maintain' OR p.normalized_login='admin')
AND g.read_fields @> ARRAY['name','effectiveDate','status']
AND (a.action<>'maintain' OR g.write_fields @> ARRAY['name','effectiveDate']);
UPDATE access.policy_guard p SET policy_no=policy_no+1 WHERE EXISTS (SELECT FROM access.subject_grant g
 WHERE (g.instance_id,g.principal_id)=(p.instance_id,p.principal_id) AND g.action IN ('legalpurpose.read','assignmentpolicy.read'));
-- +goose Down
SELECT 1/0;
