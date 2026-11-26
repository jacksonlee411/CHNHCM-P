-- +goose Up
ALTER TABLE staffing.employment_action DROP CONSTRAINT employment_action_kind_check,
    ADD CHECK(kind IN ('Hire','CancelHire','DataChange','Transfer','AdditionalAssignment','ParallelEmployment','ProbationStart','ConfirmEmployment','GradeRecord','Promotion','SetManager','EndAssignment','Leave','Rehire'));
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION staffing.check_action_assignment_scope() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT FROM staffing.employment_action a JOIN staffing.assignment t
        ON (t.instance_id,t.employment_id)=(a.instance_id,a.employment_id)
        WHERE a.instance_id=NEW.instance_id AND a.id=NEW.action_id AND t.id=NEW.assignment_id
          AND (a.assignment_id=t.id OR a.kind IN ('ProbationStart','ConfirmEmployment','Leave'))) THEN
        RAISE EXCEPTION 'action outside assignment scope' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TABLE staffing.manager_agreement (
    instance_id uuid NOT NULL,assignment_id uuid NOT NULL,action_id uuid NOT NULL,revision uuid NOT NULL,
    manager_assignment uuid,purpose text NOT NULL CHECK(purpose IN ('direct','functional')),
    period daterange NOT NULL CHECK(NOT isempty(period) AND NOT lower_inf(period)),
    PRIMARY KEY(instance_id,action_id,revision),CHECK(assignment_id<>manager_assignment),
    FOREIGN KEY(instance_id,assignment_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,assignment_id,action_id,revision),
    FOREIGN KEY(instance_id,manager_assignment) REFERENCES staffing.assignment(instance_id,id)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.manager_agreement FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE UNIQUE INDEX manager_start_per_purpose ON staffing.manager_agreement(instance_id,assignment_id,purpose,lower(period));
CREATE INDEX manager_agreement_endpoints ON staffing.manager_agreement(instance_id,purpose,assignment_id,manager_assignment);
CREATE TABLE staffing.assignment_exit (
    instance_id uuid NOT NULL,assignment_id uuid NOT NULL,action_id uuid NOT NULL,revision uuid NOT NULL,
    first_inactive date NOT NULL, PRIMARY KEY(instance_id,assignment_id),
    FOREIGN KEY(instance_id,assignment_id) REFERENCES staffing.assignment(instance_id,id),
    FOREIGN KEY(instance_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,action_id,revision)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.assignment_exit FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TABLE staffing.employment_exit (
    instance_id uuid NOT NULL,employment_id uuid NOT NULL,action_id uuid NOT NULL,revision uuid NOT NULL,
    first_inactive date NOT NULL,PRIMARY KEY(instance_id,employment_id),
    FOREIGN KEY(instance_id,employment_id) REFERENCES staffing.employment(instance_id,id),
    FOREIGN KEY(instance_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,action_id,revision)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.employment_exit FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.manager_agreement,staffing.assignment_exit,staffing.employment_exit TO chnhcm_app;
GRANT UPDATE(period) ON staffing.employment_primary TO chnhcm_app;
ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CHECK(action IN ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory','register.read','transfer','record.redate','record.delete','assignment.add','employment.parallel','probation.start','probation.confirm','grade.record','promotion','promotion.withdraw','manager.change','assignment.end','employment.leave','employment.rehire'));
CREATE TABLE access.employment_lifecycle_fixture (
    instance_id uuid NOT NULL,principal_id uuid NOT NULL,PRIMARY KEY(instance_id,principal_id),
    FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture
);

-- +goose Down
SELECT 1/0;
