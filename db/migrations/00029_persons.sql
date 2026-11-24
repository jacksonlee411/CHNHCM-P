-- +goose Up
CREATE TABLE access.person_policy (
 instance_id uuid PRIMARY KEY, version text NOT NULL,
 valid_until timestamptz NOT NULL, ordinary_days int NOT NULL CHECK (ordinary_days>0),
 evidence_days int NOT NULL CHECK (evidence_days>=ordinary_days), allow_permanent_decision boolean NOT NULL,
 material_days int NOT NULL CHECK(material_days>0),material_evidence_days int NOT NULL CHECK(material_evidence_days>=material_days),
 contact_days int NOT NULL CHECK(contact_days>0),contact_evidence_days int NOT NULL CHECK(contact_evidence_days>=contact_days)
);
CREATE TABLE access.person_fixture (instance_id uuid NOT NULL,principal_id uuid NOT NULL, profile text NOT NULL, PRIMARY KEY(instance_id,principal_id),FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal);
CREATE TABLE access.person_grant (
 instance_id uuid NOT NULL, principal_id uuid NOT NULL, grant_id uuid NOT NULL DEFAULT gen_random_uuid(),
 action text NOT NULL CHECK(action IN ('read','register','verify','correct','renumber','renumber.decide','history.read','evidence.read','hold.read')),
 purpose text NOT NULL, fields text[] NOT NULL,
 source_version uuid, person_ids uuid[], denied boolean NOT NULL DEFAULT false, expires_at timestamptz,
 PRIMARY KEY(instance_id,principal_id,grant_id), FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal
);
-- Policy changes serialize with the session's shared authorization guard.
-- +goose StatementBegin
CREATE FUNCTION access.person_grant_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='UPDATE' AND (OLD.instance_id,OLD.principal_id)<>(NEW.instance_id,NEW.principal_id) THEN
  RAISE EXCEPTION 'replace grant to change subject';
 END IF;
 IF TG_OP='DELETE' THEN
  UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=OLD.instance_id AND principal_id=OLD.principal_id;
  RETURN OLD;
 END IF;
 UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=NEW.instance_id AND principal_id=NEW.principal_id;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER guard_person_grant BEFORE INSERT OR UPDATE OR DELETE ON access.person_grant FOR EACH ROW EXECUTE FUNCTION access.person_grant_guard();

CREATE SCHEMA personnel;
CREATE TABLE personnel.person (
 instance_id uuid NOT NULL, person_id uuid NOT NULL, number text NOT NULL, head uuid NOT NULL,
 ordinary_until timestamptz NOT NULL, evidence_until timestamptz NOT NULL, policy_version text NOT NULL,
 material_until timestamptz NOT NULL,material_evidence_until timestamptz NOT NULL,
 PRIMARY KEY(instance_id,person_id)
);
CREATE TABLE personnel.number (
 instance_id uuid NOT NULL, number text NOT NULL, person_id uuid NOT NULL,
 PRIMARY KEY(instance_id,number), UNIQUE(instance_id,person_id,number),
 FOREIGN KEY(instance_id,person_id) REFERENCES personnel.person DEFERRABLE INITIALLY DEFERRED
);
ALTER TABLE personnel.person ADD FOREIGN KEY(instance_id,person_id,number) REFERENCES personnel.number(instance_id,person_id,number) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE personnel.counter(instance_id uuid PRIMARY KEY,last_number int NOT NULL CHECK(last_number BETWEEN 1 AND 99999999));
CREATE TABLE personnel.source (
 instance_id uuid NOT NULL, person_id uuid NOT NULL, version uuid NOT NULL, previous uuid,
 number text NOT NULL, actor uuid NOT NULL, operator_name text NOT NULL, recorded_at timestamptz NOT NULL,
 action text NOT NULL, decision jsonb, policy_version text NOT NULL, verification jsonb, PRIMARY KEY(instance_id,person_id,version),
 FOREIGN KEY(instance_id,person_id,number) REFERENCES personnel.number(instance_id,person_id,number),
 FOREIGN KEY(instance_id,person_id,previous) REFERENCES personnel.source(instance_id,person_id,version)
);
ALTER TABLE personnel.person ADD FOREIGN KEY(instance_id,person_id,head) REFERENCES personnel.source(instance_id,person_id,version) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE personnel.identity_payload (
 instance_id uuid NOT NULL, person_id uuid NOT NULL, version uuid NOT NULL, name text NOT NULL,
 PRIMARY KEY(instance_id,person_id,version), FOREIGN KEY(instance_id,person_id,version) REFERENCES personnel.source
);
CREATE INDEX person_candidate_name ON personnel.identity_payload(instance_id,name);
CREATE TABLE personnel.evidence_payload (
 instance_id uuid NOT NULL, person_id uuid NOT NULL, version uuid NOT NULL, kind text NOT NULL, reference text NOT NULL,
 PRIMARY KEY(instance_id,person_id,version), FOREIGN KEY(instance_id,person_id,version) REFERENCES personnel.source
);
CREATE TABLE personnel.contact_payload (
 instance_id uuid NOT NULL,person_id uuid NOT NULL,version uuid NOT NULL,item_id uuid NOT NULL,revision uuid NOT NULL,
 ordinal int NOT NULL,kind text NOT NULL,value text NOT NULL,ordinary_until timestamptz NOT NULL,evidence_until timestamptz NOT NULL,
 collection_policy text NOT NULL, PRIMARY KEY(instance_id,person_id,version,item_id), FOREIGN KEY(instance_id,person_id,version) REFERENCES personnel.source
);
CREATE TABLE personnel.material (
 instance_id uuid NOT NULL, material_hash text NOT NULL, person_id uuid NOT NULL, registration_hash text,
 source_version uuid NOT NULL, PRIMARY KEY(instance_id,material_hash),
 FOREIGN KEY(instance_id,person_id,source_version) REFERENCES personnel.source
);
CREATE TABLE personnel.outbox (
 instance_id uuid NOT NULL,event_id uuid NOT NULL,person_id uuid NOT NULL,source_version uuid NOT NULL,action text NOT NULL,
 PRIMARY KEY(instance_id,event_id),UNIQUE(instance_id,person_id,source_version),
 FOREIGN KEY(instance_id,person_id,source_version) REFERENCES personnel.source
);
CREATE TABLE personnel.terminal (
 instance_id uuid NOT NULL, actor uuid NOT NULL, submission_id uuid NOT NULL, digest text NOT NULL,
 outcome text NOT NULL, person_id uuid, version uuid, fault jsonb,
 PRIMARY KEY(instance_id,actor,submission_id), FOREIGN KEY(instance_id,person_id,version) REFERENCES personnel.source
);
CREATE TABLE personnel.query (
 instance_id uuid NOT NULL,subject uuid NOT NULL,session_id uuid NOT NULL,ref uuid NOT NULL,keyword text NOT NULL,expires_at timestamptz NOT NULL,
 PRIMARY KEY(instance_id,subject,session_id,ref)
);
CREATE INDEX person_query_expiry ON personnel.query(expires_at);
CREATE TABLE personnel.hold (
 instance_id uuid NOT NULL,person_id uuid NOT NULL,version uuid NOT NULL,category text NOT NULL CHECK(category IN ('identity','evidence','contacts')),
 until_at timestamptz NOT NULL,authority text NOT NULL, PRIMARY KEY(instance_id,person_id,version,category),
 FOREIGN KEY(instance_id,person_id,version) REFERENCES personnel.source
);
CREATE TABLE personnel.disposition (
 instance_id uuid NOT NULL,person_id uuid NOT NULL,version uuid NOT NULL,category text NOT NULL, item text NOT NULL,
 disposed_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(instance_id,person_id,version,category,item),
 FOREIGN KEY(instance_id,person_id,version) REFERENCES personnel.source
);
GRANT SELECT ON access.person_grant,access.person_policy TO chnhcm_app;
-- Only this function can take the row lock on the policy owner's behalf.
-- +goose StatementBegin
CREATE FUNCTION access.person_policy_for_use(customer uuid)
RETURNS SETOF access.person_policy LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$
 SELECT * FROM access.person_policy WHERE instance_id=customer AND valid_until>clock_timestamp() FOR SHARE
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION access.person_policy_for_use(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION access.person_policy_for_use(uuid) TO chnhcm_app;
GRANT USAGE ON SCHEMA personnel TO chnhcm_app;
GRANT SELECT ON ALL TABLES IN SCHEMA personnel TO chnhcm_app;
GRANT INSERT ON personnel.person,personnel.counter,personnel.number,personnel.source,personnel.identity_payload,personnel.evidence_payload,personnel.contact_payload,personnel.material,personnel.terminal,personnel.outbox,personnel.query TO chnhcm_app;
GRANT UPDATE(number,head) ON personnel.person TO chnhcm_app;
GRANT UPDATE(last_number) ON personnel.counter TO chnhcm_app;
GRANT UPDATE(expires_at),DELETE ON personnel.query TO chnhcm_app;
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'personnel history requires forward migration'; END $$;
-- +goose StatementEnd
