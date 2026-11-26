-- +goose Up
-- The actual assignment can start later and end earlier than its relationship.
ALTER TABLE staffing.assignment ADD COLUMN period daterange;
ALTER TABLE staffing.assignment DISABLE TRIGGER immutable_fact;
UPDATE staffing.assignment a SET period=e.period FROM staffing.employment e
    WHERE (a.instance_id,a.employment_id)=(e.instance_id,e.id);
ALTER TABLE staffing.assignment ENABLE TRIGGER immutable_fact;
ALTER TABLE staffing.assignment ALTER COLUMN period SET NOT NULL,
    ADD CHECK(NOT isempty(period) AND NOT lower_inf(period));
CREATE UNIQUE INDEX assignment_issued_positive_number ON staffing.assignment(instance_id,person_id,record_number)
    WHERE record_number>0;
ALTER TABLE staffing.employment_action DROP CONSTRAINT employment_action_kind_check,
    ADD CHECK(kind IN ('Hire','CancelHire','DataChange','Transfer','AdditionalAssignment','ParallelEmployment'));
ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CHECK(action IN ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory','register.read','transfer','record.redate','record.delete','assignment.add','employment.parallel'));
CREATE TABLE access.employment_parallel_fixture (
    instance_id uuid NOT NULL, principal_id uuid NOT NULL, PRIMARY KEY(instance_id,principal_id),
    FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture
);

-- +goose Down
SELECT 1/0;
