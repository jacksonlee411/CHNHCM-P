-- +goose Up
CREATE TABLE personnel.profile_item_payload (
 instance_id uuid NOT NULL, person_id uuid NOT NULL, version uuid NOT NULL,
 section text NOT NULL CHECK(section IN ('addresses','family')), item_id uuid NOT NULL, revision uuid NOT NULL,
 ordinal int NOT NULL, fields jsonb NOT NULL, adoptions jsonb NOT NULL,
 ordinary_until timestamptz NOT NULL, evidence_until timestamptz NOT NULL,
 collection_policy text NOT NULL, CHECK(evidence_until>=ordinary_until),
 PRIMARY KEY(instance_id,person_id,version,item_id),
 FOREIGN KEY(instance_id,person_id,version) REFERENCES personnel.source
);
CREATE UNIQUE INDEX person_address_primary ON personnel.profile_item_payload(instance_id,person_id,version)
 WHERE section='addresses' AND fields->>'addressPrimary'='yes';
GRANT SELECT,INSERT ON personnel.profile_item_payload TO chnhcm_app;
ALTER TABLE access.person_grant DROP CONSTRAINT person_grant_action_check,
 ADD CHECK(action IN ('read','register','verify','correct','renumber','renumber.decide','history.read','evidence.read','hold.read','profile.read','profile.correct','family.read','family.correct'));
-- The 00031 check restricts item grants to denials. Extend it to both read and write.
ALTER TABLE access.person_grant DROP CONSTRAINT person_grant_check,
 ADD CHECK(item_id IS NULL OR (denied AND source_version IS NULL AND (action IN ('profile.correct','family.read','family.correct')
 OR (action='profile.read' AND cardinality(fields)>0 AND fields <@ ARRAY['addressType','addressCountry','province','area','street','postalCode','addressPrimary']))));
ALTER TABLE personnel.hold DROP CONSTRAINT hold_category_check,
 ADD CHECK(category IN ('identity','evidence','contacts','addresses','family'));
CREATE TABLE access.person_address_family_fixture(instance_id uuid NOT NULL, principal_id uuid NOT NULL,
 PRIMARY KEY(instance_id,principal_id), FOREIGN KEY(instance_id,principal_id) REFERENCES access.person_profile_fixture);
-- +goose Down
SELECT 1/0;
