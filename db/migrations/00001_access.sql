-- +goose Up
CREATE SCHEMA access;
-- +goose StatementBegin
DO $$ BEGIN
 IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='chnhcm_app') THEN CREATE ROLE chnhcm_app NOLOGIN; END IF;
END $$;
-- +goose StatementEnd
CREATE TABLE access.principal (
 initialization_digest bytea NOT NULL CHECK(octet_length(initialization_digest)=32),
 instance_id uuid NOT NULL, principal_id uuid NOT NULL, login text NOT NULL, normalized_login text NOT NULL,
 state text NOT NULL CHECK(state IN ('active','disabled')), created_at timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,principal_id), UNIQUE(instance_id,normalized_login)
);
CREATE TABLE access.credential (
 instance_id uuid NOT NULL, principal_id uuid NOT NULL, algorithm text NOT NULL CHECK(algorithm='pbkdf2-sha256'),
 parameter_version integer NOT NULL CHECK(parameter_version=1), iterations integer NOT NULL CHECK(iterations>=600000),
 salt bytea NOT NULL CHECK(octet_length(salt)=16), password_hash bytea NOT NULL CHECK(octet_length(password_hash)=32),
 updated_at timestamptz(6) NOT NULL DEFAULT clock_timestamp(), PRIMARY KEY(instance_id,principal_id),
 FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal
);
CREATE TABLE access.policy_guard (
 instance_id uuid NOT NULL, principal_id uuid NOT NULL, policy_no bigint NOT NULL CHECK(policy_no>0),
 PRIMARY KEY(instance_id,principal_id), FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal
);
CREATE TABLE access.session (
 instance_id uuid NOT NULL, session_id uuid NOT NULL, principal_id uuid NOT NULL,
 token_digest bytea NOT NULL CHECK(octet_length(token_digest)=32), csrf_value bytea NOT NULL CHECK(octet_length(csrf_value)=32),
 created_at timestamptz(6) NOT NULL, last_seen_at timestamptz(6) NOT NULL, idle_expires_at timestamptz(6) NOT NULL,
 absolute_expires_at timestamptz(6) NOT NULL, revoked_at timestamptz(6),
 PRIMARY KEY(instance_id,session_id), UNIQUE(instance_id,token_digest),
 FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal,
 CHECK(created_at<=last_seen_at AND last_seen_at<=idle_expires_at AND idle_expires_at<=absolute_expires_at)
);
CREATE INDEX session_subject ON access.session(instance_id,principal_id,revoked_at);
CREATE TABLE access.subject_grant (
 instance_id uuid NOT NULL, grant_id uuid NOT NULL, principal_id uuid NOT NULL,
 scope_kind text NOT NULL CHECK(scope_kind IN ('enterprise','bu')), scope_id uuid NOT NULL,
 action text NOT NULL CHECK(action IN ('bu.read','bu.history.read','bu.create','bu.record.add','bu.record.correct','bu.record.delete','bu.deactivation.delete','bu.reactivation.delete','bu.record.redate','bu.first.redate','bu.deactivation.redate','bu.reactivation.redate','bu.deactivate','bu.reactivate','bu.timeline.revise','bu.first.reclassify','bu.record.reclassify','bu.deactivation.reclassify','bu.reactivation.reclassify')),
 read_fields text[] NOT NULL, write_fields text[] NOT NULL,
 PRIMARY KEY(instance_id,grant_id), FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal,
 UNIQUE(instance_id,principal_id,scope_kind,scope_id,action),
 CHECK(array_position(read_fields,NULL) IS NULL AND array_position(write_fields,NULL) IS NULL),
 CHECK(read_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason']::text[]),
 CHECK(write_fields <@ ARRAY['code','name','nameEn','effectiveDate','operationType','operationReason']::text[])
);
-- +goose StatementBegin
CREATE FUNCTION access.protect_security_event() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'security events are immutable' USING ERRCODE='23514'; END $$;
-- +goose StatementEnd
CREATE TABLE access.security_event (
 instance_id uuid NOT NULL, event_id uuid NOT NULL, event_kind text NOT NULL CHECK(event_kind IN ('authentication','security_change')),
 occurred_at timestamptz(6) NOT NULL, action text NOT NULL, decision text NOT NULL CHECK(decision IN ('allow','deny')),
 reason_code text NOT NULL, actor_subject_id uuid, target_subject_id uuid, request_id uuid,
 PRIMARY KEY(instance_id,event_id),
 FOREIGN KEY(instance_id,actor_subject_id) REFERENCES access.principal,
 FOREIGN KEY(instance_id,target_subject_id) REFERENCES access.principal
);
CREATE TRIGGER immutable_security_event BEFORE UPDATE OR DELETE ON access.security_event FOR EACH ROW EXECUTE FUNCTION access.protect_security_event();
GRANT USAGE ON SCHEMA access TO chnhcm_app;
GRANT SELECT ON ALL TABLES IN SCHEMA access TO chnhcm_app;
GRANT INSERT ON access.session,access.security_event TO chnhcm_app;
GRANT UPDATE(last_seen_at,idle_expires_at,revoked_at) ON access.session TO chnhcm_app;
GRANT UPDATE(policy_no) ON access.policy_guard TO chnhcm_app;

-- +goose Down
DROP SCHEMA access CASCADE;
