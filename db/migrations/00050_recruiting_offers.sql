-- +goose Up
CREATE SCHEMA recruiting;
GRANT USAGE ON SCHEMA recruiting TO chnhcm_app;

-- A position admitted at creation can have no older M05 obligations. Older positions require separate verified takeover.
CREATE TABLE recruiting.position_intake (
 instance_id uuid NOT NULL, position_id uuid NOT NULL, creation_source uuid NOT NULL, version uuid NOT NULL,
 since date NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,position_id),UNIQUE(instance_id,version)
);
CREATE TABLE recruiting.offer (
 instance_id uuid NOT NULL,id uuid NOT NULL,number text NOT NULL CHECK(length(trim(number)) BETWEEN 1 AND 160),
 person_id uuid NOT NULL,employer_id uuid NOT NULL,department_id uuid NOT NULL,job_id uuid NOT NULL,position_id uuid,
 arrival date NOT NULL, reserve_from date NOT NULL, until_date date, terms jsonb NOT NULL,
 PRIMARY KEY(instance_id,id), UNIQUE(instance_id,number),
 CHECK(reserve_from<=arrival AND (until_date IS NULL OR arrival<until_date))
);
CREATE TABLE recruiting.offer_revision (
 instance_id uuid NOT NULL,offer_id uuid NOT NULL,version uuid NOT NULL,previous uuid,
 operation text NOT NULL CHECK(operation IN ('issue','accept','release','reserve','convert','cancel_conversion','decline')),
 status text NOT NULL CHECK(status IN ('issued','accepted','declined','converted')),
 business_date date NOT NULL, basis text NOT NULL, actor uuid NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,offer_id,version), UNIQUE(instance_id,version),
 FOREIGN KEY(instance_id,offer_id) REFERENCES recruiting.offer,
 FOREIGN KEY(instance_id,offer_id,previous) REFERENCES recruiting.offer_revision
);
CREATE TABLE recruiting.offer_head (
 instance_id uuid NOT NULL,offer_id uuid NOT NULL,version uuid NOT NULL,
 PRIMARY KEY(instance_id,offer_id), FOREIGN KEY(instance_id,offer_id,version) REFERENCES recruiting.offer_revision
);
CREATE TABLE recruiting.offer_segment (
 instance_id uuid NOT NULL, offer_id uuid NOT NULL, version uuid NOT NULL, period daterange NOT NULL CHECK(NOT isempty(period) AND NOT lower_inf(period)),
 position_id uuid, effect text NOT NULL CHECK(effect IN ('reserved','none')), adoption jsonb NOT NULL,
 FOREIGN KEY(instance_id,offer_id,version) REFERENCES recruiting.offer_revision,
 EXCLUDE USING gist(instance_id WITH =,offer_id WITH =,version WITH =,period WITH &&),
 CHECK(effect<>'reserved' OR position_id IS NOT NULL)
);
CREATE INDEX offer_position_use ON recruiting.offer_segment(instance_id,position_id,version) WHERE position_id IS NOT NULL;
CREATE TABLE recruiting.offer_submission (
 instance_id uuid NOT NULL,actor uuid NOT NULL,submission_id uuid NOT NULL,digest text,offer_id uuid,version uuid,
 PRIMARY KEY(instance_id,actor,submission_id),FOREIGN KEY(instance_id,offer_id,version) REFERENCES recruiting.offer_revision,
 CHECK((digest IS NULL AND offer_id IS NULL AND version IS NULL) OR (digest IS NOT NULL AND offer_id IS NOT NULL AND version IS NOT NULL))
);
CREATE TABLE recruiting.outbox (
 instance_id uuid NOT NULL,event_id uuid NOT NULL, offer_id uuid NOT NULL,version uuid NOT NULL,
 PRIMARY KEY(instance_id,event_id),UNIQUE(instance_id,offer_id,version),
 FOREIGN KEY(instance_id,offer_id,version) REFERENCES recruiting.offer_revision
);
-- +goose StatementBegin
CREATE FUNCTION recruiting.immutable_fact() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 RAISE EXCEPTION '招聘录用和预留的原始事实只可追加';
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ DECLARE tab text; BEGIN
 FOREACH tab IN ARRAY ARRAY['position_intake','offer','offer_revision','offer_segment','offer_submission','outbox'] LOOP
   EXECUTE format('CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON recruiting.%I FOR EACH ROW EXECUTE FUNCTION recruiting.immutable_fact()',tab);
 END LOOP;
END $$;
-- +goose StatementEnd
GRANT SELECT,INSERT ON ALL TABLES IN SCHEMA recruiting TO chnhcm_app;
GRANT UPDATE(version) ON recruiting.offer_head TO chnhcm_app;

CREATE TABLE access.offer_grant (
 instance_id uuid NOT NULL,principal_id uuid NOT NULL,grant_id uuid NOT NULL DEFAULT gen_random_uuid(),
 action text NOT NULL CHECK(action IN ('read','issue','accept','reserve','release')),
 person_id uuid,employer_id uuid,department_id uuid,expires_at timestamptz,
 PRIMARY KEY(instance_id,grant_id),FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal
);
CREATE TRIGGER guard_offer_grant BEFORE INSERT OR UPDATE OR DELETE ON access.offer_grant FOR EACH ROW EXECUTE FUNCTION access.person_grant_guard();
GRANT SELECT ON access.offer_grant TO chnhcm_app;
CREATE TABLE access.offer_fixture(instance_id uuid NOT NULL,principal_id uuid NOT NULL,
 PRIMARY KEY(instance_id,principal_id),FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal);
WITH registered AS (
 INSERT INTO access.offer_fixture SELECT instance_id,principal_id FROM access.employment_fixture RETURNING *
)
INSERT INTO access.offer_grant(instance_id,principal_id,action)
 SELECT instance_id,principal_id,a FROM registered CROSS JOIN unnest(ARRAY['read','issue','accept','reserve','release']) a;
-- +goose Down
SELECT 1/0;
