-- +goose Up
ALTER TABLE personnel.profile_item_payload DROP CONSTRAINT profile_item_payload_section_check,
 ADD CHECK(section IN ('addresses','family','education','credentials'));
CREATE UNIQUE INDEX person_highest_education ON personnel.profile_item_payload(instance_id,person_id,version)
 WHERE section='education' AND fields->>'highestEducation'='yes';
CREATE UNIQUE INDEX person_highest_degree ON personnel.profile_item_payload(instance_id,person_id,version)
 WHERE section='education' AND fields->>'highestDegree'='yes';
ALTER TABLE access.person_grant DROP CONSTRAINT person_grant_hire_action,
 ADD CONSTRAINT person_grant_hire_action CHECK(action IN ('read','register','verify','correct','renumber','renumber.decide','history.read','evidence.read','hold.read','profile.read','profile.correct','family.read','family.correct','hire.check','qualification.read','qualification.correct'));
ALTER TABLE access.person_grant DROP CONSTRAINT person_grant_check,
 ADD CHECK(item_id IS NULL OR (denied AND source_version IS NULL AND (action IN ('profile.correct','family.read','family.correct','qualification.read','qualification.correct')
 OR (action='profile.read' AND cardinality(fields)>0 AND fields <@ ARRAY['addressType','addressCountry','province','area','street','postalCode','addressPrimary']))));
ALTER TABLE personnel.hold DROP CONSTRAINT hold_category_check,
 ADD CHECK(category IN ('identity','evidence','contacts','addresses','family','education','credentials'));
CREATE TABLE access.person_qualification_fixture(instance_id uuid NOT NULL,principal_id uuid NOT NULL,
 PRIMARY KEY(instance_id,principal_id),FOREIGN KEY(instance_id,principal_id) REFERENCES access.person_profile_fixture);
-- +goose Down
SELECT 1/0;
