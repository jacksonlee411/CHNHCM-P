-- +goose Up
CREATE TABLE personnel.address_type (
 instance_id uuid NOT NULL, item_id uuid NOT NULL DEFAULT gen_random_uuid(), code text NOT NULL,
 name text NOT NULL, sort_order integer NOT NULL, first_day date NOT NULL, selectable boolean NOT NULL,
 head uuid NOT NULL DEFAULT gen_random_uuid(), legacy_label text,
 PRIMARY KEY(instance_id,item_id), UNIQUE(instance_id,code), UNIQUE(instance_id,legacy_label)
);
CREATE INDEX address_type_choices ON personnel.address_type(instance_id,sort_order,item_id) WHERE selectable;
CREATE TABLE personnel.address_type_revision (
 instance_id uuid NOT NULL, item_id uuid NOT NULL, revision uuid NOT NULL,
 name text NOT NULL, sort_order integer NOT NULL, selectable boolean NOT NULL,
 actor_id uuid, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 submission_id uuid, digest text, PRIMARY KEY(instance_id,item_id,revision),
 UNIQUE(instance_id,actor_id,submission_id),
 CHECK ((actor_id IS NULL)=(submission_id IS NULL) AND (actor_id IS NULL)=(digest IS NULL)),
 FOREIGN KEY(instance_id,item_id) REFERENCES personnel.address_type
);
-- +goose StatementBegin
CREATE FUNCTION personnel.reject_address_type_revision_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 RAISE EXCEPTION 'address type revisions are immutable';
END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_revision BEFORE UPDATE OR DELETE ON personnel.address_type_revision
 FOR EACH ROW EXECUTE FUNCTION personnel.reject_address_type_revision_change();
ALTER TABLE personnel.address_type ADD FOREIGN KEY(instance_id,item_id,head)
 REFERENCES personnel.address_type_revision DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE personnel.profile_item_payload
 ADD COLUMN address_type_id uuid, ADD COLUMN address_type_revision uuid, ADD COLUMN address_type_day date,
 ADD CHECK ((address_type_id IS NULL AND address_type_revision IS NULL AND address_type_day IS NULL)
 OR (section='addresses' AND address_type_id IS NOT NULL AND address_type_revision IS NOT NULL AND address_type_day IS NOT NULL)),
 ADD FOREIGN KEY(instance_id,address_type_id,address_type_revision) REFERENCES personnel.address_type_revision;
GRANT SELECT,INSERT ON personnel.address_type,personnel.address_type_revision TO chnhcm_app;
GRANT UPDATE(name,sort_order,selectable,head) ON personnel.address_type TO chnhcm_app;

-- A system-provided initial revision has no human operator or submission.
-- +goose StatementBegin
CREATE FUNCTION personnel.initialize_address_types(customer uuid) RETURNS void LANGUAGE sql AS $$
 WITH seeded AS (
  INSERT INTO personnel.address_type(instance_id,code,name,sort_order,first_day,selectable,legacy_label)
  SELECT customer,code,label,position,DATE '1900-01-01',true,label FROM (VALUES
   ('RESIDENTIAL','居住地址',10),('REGISTERED','户籍地址',20),('MAILING','通讯地址',30),('OTHER','其他地址',40)
  ) AS initial(code,label,position) ON CONFLICT(instance_id,code) DO NOTHING RETURNING *
 )
 INSERT INTO personnel.address_type_revision(instance_id,item_id,revision,name,sort_order,selectable)
 SELECT instance_id,item_id,head,name,sort_order,selectable FROM seeded
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION personnel.initialize_address_types(uuid) FROM PUBLIC;
SELECT personnel.initialize_address_types(instance_id) FROM (SELECT DISTINCT instance_id FROM access.principal) customers;

ALTER TABLE access.person_grant DROP CONSTRAINT person_grant_hire_action,
 ADD CONSTRAINT person_grant_hire_action CHECK(action IN ('read','register','verify','correct','renumber','renumber.decide','history.read','evidence.read','hold.read','profile.read','profile.correct','family.read','family.correct','hire.check','qualification.read','qualification.correct','bank.read','bank.correct','health.read','health.correct','address-type.read','address-type.maintain')),
 ADD CONSTRAINT person_address_type_grant_scope CHECK(action NOT IN ('address-type.read','address-type.maintain') OR
 (purpose='address-types' AND person_ids IS NULL AND source_version IS NULL AND item_id IS NULL AND cardinality(fields)=0));
-- Upgrade only expressly provisioned local profile maintenance fixtures.
INSERT INTO access.person_grant(instance_id,principal_id,action,purpose,fields)
 SELECT f.instance_id,f.principal_id,a,'address-types','{}' FROM access.person_profile_fixture p
 JOIN access.person_fixture f USING(instance_id,principal_id) CROSS JOIN unnest(ARRAY['address-type.read','address-type.maintain']) a
 WHERE f.profile='registrar';

-- Resolve-before-send closes a missing submission while holding the same transaction lock as save.
CREATE TABLE personnel.address_type_not_applied (
 instance_id uuid NOT NULL, actor_id uuid NOT NULL, submission_id uuid NOT NULL,
 PRIMARY KEY(instance_id,actor_id,submission_id)
);
GRANT SELECT,INSERT ON personnel.address_type_not_applied TO chnhcm_app;

-- +goose Down
SELECT 1/0;
