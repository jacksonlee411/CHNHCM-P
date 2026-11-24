-- +goose Up
ALTER TABLE personnel.identity_payload ADD COLUMN basic jsonb NOT NULL DEFAULT '{}', ADD COLUMN nationality_adoption jsonb;
CREATE TABLE personnel.document_payload (
 instance_id uuid NOT NULL, person_id uuid NOT NULL, version uuid NOT NULL,
 item_id uuid NOT NULL, revision uuid NOT NULL, ordinal int NOT NULL,
 kind text NOT NULL CHECK(kind IN ('resident_id','passport','other')),
 number text COLLATE "C" NOT NULL CHECK(number<>'' AND number=translate(btrim(number),'abcdefghijklmnopqrstuvwxyz','ABCDEFGHIJKLMNOPQRSTUVWXYZ')),
 attributes jsonb NOT NULL, issuing_country_adoption jsonb, long_term boolean NOT NULL, is_primary boolean NOT NULL, occupied boolean NOT NULL CHECK(NOT occupied OR is_primary),
 PRIMARY KEY(instance_id,person_id,version,item_id), UNIQUE(instance_id,person_id,version,number),
 FOREIGN KEY(instance_id,person_id,version) REFERENCES personnel.identity_payload ON DELETE CASCADE
);
CREATE UNIQUE INDEX document_occupied ON personnel.document_payload(instance_id,number) WHERE occupied;
CREATE UNIQUE INDEX document_primary ON personnel.document_payload(instance_id,person_id,version) WHERE is_primary;
INSERT INTO personnel.document_payload
SELECT instance_id,person_id,version,
 first_value(version) OVER(PARTITION BY instance_id,person_id,document_number ORDER BY s.recorded_at,version),
 version,0,document_kind,document_number,'{}',NULL,false,true,current_document
FROM personnel.identity_payload JOIN personnel.source s USING(instance_id,person_id,version)
WHERE document_number IS NOT NULL;
ALTER TABLE personnel.identity_payload DROP COLUMN document_kind,DROP COLUMN document_number,DROP COLUMN current_document;
GRANT SELECT,INSERT,UPDATE(occupied) ON personnel.document_payload TO chnhcm_app;
ALTER TABLE personnel.contact_payload ADD COLUMN purpose text NOT NULL DEFAULT '', ADD COLUMN is_primary boolean NOT NULL DEFAULT false;
-- Only currently usable rows are copied to each new source with an active primary flag.
CREATE UNIQUE INDEX contact_channel_primary ON personnel.contact_payload(instance_id,person_id,version,kind) WHERE is_primary;
ALTER TABLE access.person_grant DROP CONSTRAINT person_grant_action_check;
ALTER TABLE access.person_grant ADD CHECK(action IN ('read','register','verify','correct','renumber','renumber.decide','history.read','evidence.read','hold.read','profile.read','profile.correct'));
-- A scoped restriction on an existing row also applies to indirect changes such as a primary switch.
ALTER TABLE access.person_grant ADD COLUMN item_id uuid,
 ADD CHECK(item_id IS NULL OR (denied AND action='profile.correct' AND source_version IS NULL));
CREATE TABLE access.person_profile_fixture(instance_id uuid NOT NULL,principal_id uuid NOT NULL,PRIMARY KEY(instance_id,principal_id), FOREIGN KEY(instance_id,principal_id) REFERENCES access.person_fixture);
-- +goose Down
SELECT 1/0;
