-- +goose Up
ALTER TABLE access.principal ADD COLUMN management_owner uuid;
-- Pre-existing accounts without an explicit owner remain under the instance operator's control.
CREATE TABLE access.management_root (
 instance_id uuid PRIMARY KEY, principal_id uuid NOT NULL, max_proxy_days integer NOT NULL CHECK(max_proxy_days BETWEEN 0 AND 3650),
 receipt_key bytea NOT NULL CHECK(octet_length(receipt_key)=32),
 FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal
);
CREATE TABLE access.management_publication (
 instance_id uuid NOT NULL, kind text NOT NULL, version_id uuid NOT NULL, name text NOT NULL CHECK(length(name)>0),
 scope_id uuid, fields_id uuid, scope_kind text GENERATED ALWAYS AS ('scope'::text) STORED, fields_kind text GENERATED ALWAYS AS ('fields'::text) STORED,
 PRIMARY KEY(instance_id,kind,version_id), FOREIGN KEY(instance_id,kind,version_id) REFERENCES access.policy_version,
 FOREIGN KEY(instance_id,scope_kind,scope_id) REFERENCES access.policy_version,
 FOREIGN KEY(instance_id,fields_kind,fields_id) REFERENCES access.policy_version,
 CHECK((kind='role')=(scope_id IS NOT NULL) AND (kind='role')=(fields_id IS NOT NULL))
);
CREATE TABLE access.management_mandate (
 instance_id uuid NOT NULL, id uuid NOT NULL, principal_id uuid NOT NULL, owner_id uuid NOT NULL,
 rights text[] NOT NULL CHECK(cardinality(rights)>0 AND array_position(rights,NULL) IS NULL AND rights <@ ARRAY['account','credential','role','policy','assign','delegate']),
 role_id uuid NOT NULL,scope_id uuid NOT NULL,fields_id uuid NOT NULL,
 role_kind text GENERATED ALWAYS AS ('role'::text) STORED, scope_kind text GENERATED ALWAYS AS ('scope'::text) STORED, fields_kind text GENERATED ALWAYS AS ('fields'::text) STORED,
 max_proxy_days integer NOT NULL CHECK(max_proxy_days BETWEEN 0 AND 3650),valid_until timestamptz,revoked_at timestamptz,
 PRIMARY KEY(instance_id,id),FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal,
 FOREIGN KEY(instance_id,role_kind,role_id) REFERENCES access.policy_version,
 FOREIGN KEY(instance_id,scope_kind,scope_id) REFERENCES access.policy_version,
 FOREIGN KEY(instance_id,fields_kind,fields_id) REFERENCES access.policy_version
);
-- +goose StatementBegin
CREATE FUNCTION access.protect_management_mandate() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR OLD.revoked_at IS NOT NULL OR NEW.revoked_at IS NULL OR
    (NEW.instance_id,NEW.id,NEW.principal_id,NEW.owner_id,NEW.rights,NEW.role_id,NEW.scope_id,NEW.fields_id,NEW.max_proxy_days,NEW.valid_until)
    IS DISTINCT FROM (OLD.instance_id,OLD.id,OLD.principal_id,OLD.owner_id,OLD.rights,OLD.role_id,OLD.scope_id,OLD.fields_id,OLD.max_proxy_days,OLD.valid_until) THEN
   RAISE EXCEPTION 'management mandate terms are immutable' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_management_mandate BEFORE UPDATE OR DELETE ON access.management_mandate
 FOR EACH ROW EXECUTE FUNCTION access.protect_management_mandate();
CREATE INDEX management_mandate_subject ON access.management_mandate(instance_id,principal_id) WHERE revoked_at IS NULL;
CREATE TABLE access.management_receipt (
 receipt_order bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
 instance_id uuid NOT NULL, request_id uuid NOT NULL, actor_id uuid, operator_name text, operation text NOT NULL, target_id uuid NOT NULL,subject_id uuid,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(), reason text NOT NULL, command_seal bytea NOT NULL CHECK(octet_length(command_seal)=32),
 previous_owner uuid, assigned_owner uuid,
 CHECK(previous_owner IS NULL OR operation='account.move'), CHECK(assigned_owner IS NULL OR operation IN ('account.create','account.move')),
 PRIMARY KEY(instance_id,request_id), CHECK((actor_id IS NULL)=(operator_name IS NOT NULL)),
 FOREIGN KEY(instance_id,actor_id) REFERENCES access.principal, FOREIGN KEY(instance_id,subject_id) REFERENCES access.principal
);
CREATE INDEX management_history ON access.management_receipt(instance_id,target_id,recorded_at);
CREATE TRIGGER immutable_management_receipt BEFORE UPDATE OR DELETE ON access.management_receipt FOR EACH ROW EXECUTE FUNCTION access.protect_security_event();
CREATE TRIGGER immutable_management_publication BEFORE UPDATE OR DELETE ON access.management_publication FOR EACH ROW EXECUTE FUNCTION access.protect_security_event();
GRANT SELECT ON access.management_root,access.management_mandate,access.management_publication,access.management_receipt TO chnhcm_app;
GRANT INSERT ON access.principal,access.credential,access.policy_guard,access.management_mandate,access.management_publication,access.management_receipt,access.grant_assignment,access.grant_revision,access.policy_version TO chnhcm_app;
GRANT UPDATE(revoked_at) ON access.management_mandate TO chnhcm_app;
GRANT UPDATE(state,management_owner) ON access.principal TO chnhcm_app;
GRANT UPDATE(salt,password_hash,iterations,updated_at) ON access.credential TO chnhcm_app;
GRANT UPDATE(disabled_at) ON access.policy_version TO chnhcm_app;
GRANT UPDATE(revision) ON access.grant_assignment TO chnhcm_app;
-- +goose Down
SELECT 1/0;
