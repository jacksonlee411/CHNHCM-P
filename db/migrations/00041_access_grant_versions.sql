-- +goose Up
-- Freeze the public 00039 grants and their constraints until parity is proven.
LOCK TABLE access.subject_grant IN ACCESS EXCLUSIVE MODE;
-- The old model has no grant timestamps or grantor. Keep NULL for both.
CREATE TABLE access.policy_version (
 instance_id uuid NOT NULL, kind text NOT NULL CHECK(kind IN ('role','scope','fields')),
 version_id uuid NOT NULL, definition_id uuid NOT NULL, version_no bigint NOT NULL CHECK(version_no>0),
 actions text[], scope_kind text, scope_id uuid, read_fields text[], write_fields text[],
 disabled_at timestamptz, lock_no integer NOT NULL DEFAULT 0 CHECK(lock_no=0),
 PRIMARY KEY(instance_id,kind,version_id), UNIQUE(instance_id,kind,definition_id,version_no),
 CHECK ((kind='role')=(actions IS NOT NULL) AND (kind='scope')=(scope_kind IS NOT NULL)
    AND (kind='scope')=(scope_id IS NOT NULL) AND (kind='fields')=(read_fields IS NOT NULL)
    AND (kind='fields')=(write_fields IS NOT NULL)),
 CHECK (cardinality(actions)>0 AND array_position(actions,NULL) IS NULL),
 CHECK (scope_kind IN ('instance','enterprise','bu','department','position','defined_position','position_catalog')),
 CHECK (array_position(read_fields,NULL) IS NULL AND array_position(write_fields,NULL) IS NULL)
);
CREATE TABLE access.grant_assignment (
 instance_id uuid NOT NULL, grant_id uuid NOT NULL, principal_id uuid NOT NULL,
 source text NOT NULL CHECK(source IN ('migration','development','duty','proxy')),
 purpose text NOT NULL CHECK(length(purpose)>0),
 valid_from timestamptz, valid_until timestamptz, revision bigint NOT NULL CHECK(revision>0),
 PRIMARY KEY(instance_id,grant_id), FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal,
 CHECK(valid_from IS NULL OR valid_until IS NULL OR valid_from<valid_until)
);
CREATE INDEX grant_subject ON access.grant_assignment(instance_id,principal_id);
CREATE TABLE access.grant_revision (
 instance_id uuid NOT NULL, grant_id uuid NOT NULL, revision bigint NOT NULL CHECK(revision>0),
 role_id uuid NOT NULL, scope_id uuid NOT NULL, fields_id uuid NOT NULL,
 role_kind text GENERATED ALWAYS AS ('role'::text) STORED,
 scope_kind text GENERATED ALWAYS AS ('scope'::text) STORED,
 fields_kind text GENERATED ALWAYS AS ('fields'::text) STORED,
 operation text NOT NULL CHECK(operation IN ('issue','adopt','revoke')),
 actor_id uuid, recorded_at timestamptz, request_id uuid, command jsonb,
 PRIMARY KEY(instance_id,grant_id,revision), UNIQUE(instance_id,request_id),
 FOREIGN KEY(instance_id,grant_id) REFERENCES access.grant_assignment,
 FOREIGN KEY(instance_id,actor_id) REFERENCES access.principal,
 FOREIGN KEY(instance_id,role_kind,role_id) REFERENCES access.policy_version,
 FOREIGN KEY(instance_id,scope_kind,scope_id) REFERENCES access.policy_version,
 FOREIGN KEY(instance_id,fields_kind,fields_id) REFERENCES access.policy_version
);
ALTER TABLE access.grant_assignment ADD FOREIGN KEY(instance_id,grant_id,revision)
 REFERENCES access.grant_revision DEFERRABLE INITIALLY DEFERRED;
CREATE TRIGGER immutable_revision BEFORE UPDATE OR DELETE ON access.grant_revision
 FOR EACH ROW EXECUTE FUNCTION access.protect_security_event();
-- +goose StatementBegin
CREATE FUNCTION access.protect_assignment_terms() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR NEW.revision<>OLD.revision+1
    OR to_jsonb(NEW)-'revision' <> to_jsonb(OLD)-'revision' THEN
  RAISE EXCEPTION 'grant terms are fixed; append a revision' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_assignment BEFORE UPDATE OR DELETE ON access.grant_assignment
 FOR EACH ROW EXECUTE FUNCTION access.protect_assignment_terms();

-- +goose StatementBegin
CREATE FUNCTION access.protect_policy_version() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR OLD.disabled_at IS NOT NULL OR NEW.disabled_at IS NULL
    OR to_jsonb(NEW)-'disabled_at' <> to_jsonb(OLD)-'disabled_at' THEN
  RAISE EXCEPTION 'published policy versions are immutable' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_definition BEFORE UPDATE OR DELETE ON access.policy_version
 FOR EACH ROW EXECUTE FUNCTION access.protect_policy_version();

-- Preserve every deployed action/scope/field constraint on a concrete binding.
-- PostgreSQL owns structural integrity, Go owns the decision for the caller.
-- +goose StatementBegin
DO $$ DECLARE predicate text; BEGIN
 SELECT string_agg('('||pg_get_expr(conbin,conrelid)||')',' AND ' ORDER BY conname) INTO STRICT predicate
 FROM pg_constraint WHERE conrelid='access.subject_grant'::regclass AND contype='c';
 EXECUTE format('CREATE FUNCTION access.valid_permission(instance_id uuid,scope_kind text,scope_id uuid,action text,read_fields text[],write_fields text[]) RETURNS boolean LANGUAGE sql IMMUTABLE AS %L','SELECT '||predicate);
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION access.check_binding() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.operation<>'revoke' AND EXISTS (
  SELECT FROM access.policy_version r
  JOIN access.policy_version s ON (s.instance_id,s.kind,s.version_id)=(NEW.instance_id,'scope',NEW.scope_id)
  JOIN access.policy_version f ON (f.instance_id,f.kind,f.version_id)=(NEW.instance_id,'fields',NEW.fields_id)
  CROSS JOIN LATERAL unnest(r.actions) op(action)
  WHERE (r.instance_id,r.kind,r.version_id)=(NEW.instance_id,'role',NEW.role_id)
    AND NOT access.valid_permission(NEW.instance_id,s.scope_kind,s.scope_id,op.action,f.read_fields,f.write_fields)
 ) THEN RAISE EXCEPTION 'invalid action, scope or fields in binding' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER validate_binding BEFORE INSERT ON access.grant_revision
 FOR EACH ROW EXECUTE FUNCTION access.check_binding();

-- Grants keep their original identity. Share only byte-equivalent definitions.
CREATE TEMP TABLE migration_bindings ON COMMIT DROP AS
 SELECT g.*,first_value(grant_id) OVER (PARTITION BY instance_id,action ORDER BY grant_id) role_version,
 first_value(grant_id) OVER (PARTITION BY instance_id,scope_kind,scope_id ORDER BY grant_id) scope_version,
 first_value(grant_id) OVER (PARTITION BY instance_id,read_fields,write_fields ORDER BY grant_id) fields_version
 FROM access.subject_grant g;
INSERT INTO access.policy_version(instance_id,kind,version_id,definition_id,version_no,actions)
 SELECT DISTINCT instance_id,'role',role_version,role_version,1,ARRAY[action] FROM migration_bindings;
INSERT INTO access.policy_version(instance_id,kind,version_id,definition_id,version_no,scope_kind,scope_id)
 SELECT DISTINCT instance_id,'scope',scope_version,scope_version,1,scope_kind,scope_id FROM migration_bindings;
INSERT INTO access.policy_version(instance_id,kind,version_id,definition_id,version_no,read_fields,write_fields)
 SELECT DISTINCT instance_id,'fields',fields_version,fields_version,1,read_fields,write_fields FROM migration_bindings;
INSERT INTO access.grant_assignment(instance_id,grant_id,principal_id,source,purpose,revision)
 SELECT instance_id,grant_id,principal_id,'migration',split_part(action,'.',1),1 FROM access.subject_grant;
INSERT INTO access.grant_revision(instance_id,grant_id,revision,role_id,scope_id,fields_id,operation)
 SELECT instance_id,grant_id,1,role_version,scope_version,fields_version,'issue' FROM migration_bindings;

CREATE VIEW access.bound_permission AS
 SELECT a.instance_id,a.grant_id,a.principal_id,s.scope_kind,s.scope_id,op.action,f.read_fields,f.write_fields,
 a.purpose,a.valid_from,a.valid_until
 FROM access.grant_assignment a JOIN access.grant_revision b USING(instance_id,grant_id,revision)
 JOIN access.policy_version r ON (r.instance_id,r.kind,r.version_id)=(b.instance_id,'role',b.role_id)
 JOIN access.policy_version s ON (s.instance_id,s.kind,s.version_id)=(b.instance_id,'scope',b.scope_id)
 JOIN access.policy_version f ON (f.instance_id,f.kind,f.version_id)=(b.instance_id,'fields',b.fields_id)
 CROSS JOIN LATERAL unnest(r.actions) op(action)
 WHERE b.operation<>'revoke' AND r.disabled_at IS NULL AND s.disabled_at IS NULL AND f.disabled_at IS NULL;
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS (SELECT * FROM access.subject_grant EXCEPT ALL
  SELECT instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields FROM access.bound_permission)
 OR EXISTS (SELECT instance_id,grant_id,principal_id,scope_kind,scope_id,action,read_fields,write_fields FROM access.bound_permission
  EXCEPT ALL SELECT * FROM access.subject_grant) THEN
  RAISE EXCEPTION 'access grant conversion changed actual permissions';
 END IF;
END $$;
-- +goose StatementEnd
DROP TABLE access.subject_grant;
ANALYZE access.policy_version;
ANALYZE access.grant_assignment;
ANALYZE access.grant_revision;
GRANT SELECT ON access.policy_version,access.grant_assignment,access.grant_revision,access.bound_permission TO chnhcm_app;
-- PostgreSQL requires column UPDATE privilege for SELECT FOR SHARE.
GRANT UPDATE(lock_no) ON access.policy_version TO chnhcm_app;
-- +goose Down
SELECT 1/0;
