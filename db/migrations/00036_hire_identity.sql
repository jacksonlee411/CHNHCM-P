-- +goose Up
ALTER TABLE access.person_grant DROP CONSTRAINT person_grant_action_check,
 ADD CONSTRAINT person_grant_hire_action CHECK(action IN ('read','register','verify','correct','renumber','renumber.decide','history.read','evidence.read','hold.read','profile.read','profile.correct','family.read','family.correct','hire.check'));
CREATE TABLE access.hire_identity_fixture (
 instance_id uuid NOT NULL, principal_id uuid NOT NULL, PRIMARY KEY(instance_id,principal_id),
 FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal
);
-- +goose Down
SELECT 1/0;
