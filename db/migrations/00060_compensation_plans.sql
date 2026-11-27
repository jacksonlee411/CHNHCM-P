-- +goose Up
CREATE SCHEMA configuration;
GRANT USAGE ON SCHEMA configuration TO chnhcm_app;

-- X02 owns the selected policy and immutable source evidence. M09 adopts its
-- version in the same transaction as the plan record.
CREATE TABLE configuration.plan_scope_version (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, plan_id uuid NOT NULL,
 version uuid NOT NULL, policy jsonb NOT NULL,
 PRIMARY KEY(instance_id,enterprise_id,plan_id,version)
);
-- +goose StatementBegin
CREATE FUNCTION configuration.immutable_policy() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'immutable applicability policy version' USING ERRCODE='55000'; END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_scope_version BEFORE UPDATE OR DELETE ON configuration.plan_scope_version FOR EACH ROW EXECUTE FUNCTION configuration.immutable_policy();
GRANT SELECT,INSERT ON configuration.plan_scope_version TO chnhcm_app;

CREATE TABLE compensation.plan (
 instance_id uuid NOT NULL, enterprise_id uuid NOT NULL, id uuid NOT NULL,
 code text NOT NULL, head uuid NOT NULL, PRIMARY KEY(instance_id,id),
 UNIQUE(instance_id,enterprise_id,code)
);
CREATE TABLE compensation.plan_revision (
 instance_id uuid NOT NULL, plan_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 fact jsonb NOT NULL, PRIMARY KEY(instance_id,plan_id,record_id,revision),
 FOREIGN KEY(instance_id,plan_id) REFERENCES compensation.plan
);
CREATE TABLE compensation.plan_version (
 instance_id uuid NOT NULL, plan_id uuid NOT NULL, id uuid NOT NULL, predecessor uuid,
 action text NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL, previous_record_id uuid, previous_revision uuid,
 until_date date, previous_until date, operator jsonb NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,plan_id,id),
 FOREIGN KEY(instance_id,plan_id,predecessor) REFERENCES compensation.plan_version,
 FOREIGN KEY(instance_id,plan_id,record_id,revision) REFERENCES compensation.plan_revision,
 CHECK((previous_record_id IS NULL)=(previous_revision IS NULL)),
 FOREIGN KEY(instance_id,plan_id,previous_record_id,previous_revision) REFERENCES compensation.plan_revision
);
ALTER TABLE compensation.plan ADD FOREIGN KEY(instance_id,id,head) REFERENCES compensation.plan_version DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE compensation.plan_period (
 instance_id uuid NOT NULL, plan_id uuid NOT NULL, record_id uuid NOT NULL, revision uuid NOT NULL,
 name text NOT NULL, start_date date NOT NULL, until_date date, active boolean NOT NULL,
 valid_period daterange GENERATED ALWAYS AS (daterange(start_date,until_date,'[)')) STORED,
 PRIMARY KEY(instance_id,plan_id,record_id), UNIQUE(instance_id,plan_id,start_date),
 CHECK(until_date IS NULL OR start_date<until_date),
 FOREIGN KEY(instance_id,plan_id,record_id,revision) REFERENCES compensation.plan_revision
);
-- One live use per rate and plan period, independent of how many rows reference
-- it. The immutable adopted versions remain in the corresponding plan revision.
CREATE TABLE compensation.plan_rate_use (
 instance_id uuid NOT NULL, rate_id uuid NOT NULL, plan_id uuid NOT NULL, record_id uuid NOT NULL,
 valid_period daterange NOT NULL, PRIMARY KEY(instance_id,rate_id,plan_id,record_id),
 FOREIGN KEY(instance_id,rate_id) REFERENCES compensation.grade_rate,
 FOREIGN KEY(instance_id,plan_id,record_id) REFERENCES compensation.plan_period ON DELETE CASCADE
);
CREATE TABLE compensation.plan_receipt (LIKE compensation.grade_rate_receipt INCLUDING ALL);
CREATE TABLE compensation.plan_outbox (
 instance_id uuid NOT NULL, plan_id uuid NOT NULL, version uuid NOT NULL, event jsonb NOT NULL,
 PRIMARY KEY(instance_id,plan_id,version),
 FOREIGN KEY(instance_id,plan_id,version) REFERENCES compensation.plan_version
);
CREATE TRIGGER immutable_revision BEFORE UPDATE OR DELETE ON compensation.plan_revision FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
CREATE TRIGGER immutable_version BEFORE UPDATE OR DELETE ON compensation.plan_version FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
CREATE TRIGGER immutable_receipt BEFORE UPDATE OR DELETE ON compensation.plan_receipt FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON compensation.plan_outbox FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
GRANT SELECT,INSERT ON compensation.plan,compensation.plan_revision,compensation.plan_version,compensation.plan_period,compensation.plan_rate_use,compensation.plan_receipt,compensation.plan_outbox TO chnhcm_app;
GRANT UPDATE(head) ON compensation.plan TO chnhcm_app;
GRANT UPDATE,DELETE ON compensation.plan_period TO chnhcm_app;
GRANT DELETE ON compensation.plan_rate_use TO chnhcm_app;
ALTER FUNCTION access.valid_permission(uuid,text,uuid,text,text[],text[]) RENAME TO valid_pre_plan_permission;
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION access.valid_permission(instance_id uuid,scope_kind text,scope_id uuid,action text,read_fields text[],write_fields text[])
RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT CASE WHEN split_part(action,'.',1)='compplan' THEN
 scope_kind='enterprise' AND action IN ('compplan.read','compplan.history.read','compplan.create','compplan.add','compplan.correct','compplan.delete','compplan.deactivate','compplan.reactivate')
 AND read_fields <@ ARRAY['code','name','description','scope','components','amount','rates','percentage','effectiveDate','reason','status','operator']
 AND write_fields <@ ARRAY['code','name','description','scope','components','amount','rates','percentage','effectiveDate','reason']
 AND (action NOT IN ('compplan.read','compplan.history.read') OR cardinality(write_fields)=0)
 ELSE access.valid_pre_plan_permission(instance_id,scope_kind,scope_id,action,read_fields,write_fields) END
$$;
-- +goose StatementEnd
-- +goose Down
SELECT 1/0;
