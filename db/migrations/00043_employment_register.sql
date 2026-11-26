-- +goose Up
ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CONSTRAINT employment_grant_action_check CHECK (action IN
      ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory','register.read'));
CREATE TABLE access.employment_register_fixture (
    instance_id uuid NOT NULL, principal_id uuid NOT NULL, PRIMARY KEY(instance_id,principal_id),
    FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture
);
-- +goose Down
SELECT 1/0;
