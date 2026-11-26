-- +goose Up
ALTER TABLE staffing.employment_action DROP CONSTRAINT employment_action_kind_check,
    ADD CHECK(kind IN ('Hire','CancelHire','DataChange','Transfer','AdditionalAssignment','ParallelEmployment','ProbationStart','ConfirmEmployment'));
CREATE UNIQUE INDEX relationship_probation_fact ON staffing.employment_action(instance_id,employment_id,kind)
    WHERE kind IN ('ProbationStart','ConfirmEmployment');

-- A relationship fact can change several assignment views. Each source still
-- identifies the exact revision of the one business action that caused it.
ALTER TABLE staffing.assignment_source DROP CONSTRAINT assignment_source_instance_id_action_id_revision_key,
    DROP CONSTRAINT assignment_source_instance_id_assignment_id_action_id_revi_fkey,
    ADD UNIQUE(instance_id,assignment_id,action_id,revision),
    ADD FOREIGN KEY(instance_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,action_id,revision);
ALTER TABLE staffing.assignment_member DROP CONSTRAINT assignment_member_instance_id_assignment_id_action_id_revi_fkey,
    ADD FOREIGN KEY(instance_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,action_id,revision);
-- +goose StatementBegin
CREATE FUNCTION staffing.check_action_assignment_scope() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT FROM staffing.employment_action a JOIN staffing.assignment t
        ON (t.instance_id,t.employment_id)=(a.instance_id,a.employment_id)
        WHERE a.instance_id=NEW.instance_id AND a.id=NEW.action_id AND t.id=NEW.assignment_id
          AND (a.assignment_id=t.id OR a.kind IN ('ProbationStart','ConfirmEmployment'))) THEN
        RAISE EXCEPTION 'action outside assignment scope' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER check_action_scope BEFORE INSERT ON staffing.assignment_source FOR EACH ROW EXECUTE FUNCTION staffing.check_action_assignment_scope();
CREATE TRIGGER check_action_scope BEFORE INSERT ON staffing.assignment_member FOR EACH ROW EXECUTE FUNCTION staffing.check_action_assignment_scope();
-- A revision's publication denotes exactly the source in its original assignment.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION staffing.record_revision_publication() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO staffing.employment_revision_publication(instance_id,assignment_id,action_id,revision,version)
    SELECT m.instance_id,m.assignment_id,m.action_id,m.revision,m.version FROM staffing.assignment_member m
    JOIN staffing.employment_action a ON (a.instance_id,a.id,a.assignment_id)=(m.instance_id,m.action_id,m.assignment_id)
    WHERE (m.instance_id,m.assignment_id,m.version)=(NEW.instance_id,NEW.assignment_id,NEW.version)
    ON CONFLICT(instance_id,action_id,revision) DO NOTHING;
    RETURN NEW;
END $$;
-- +goose StatementEnd
ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CHECK(action IN ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory','register.read','transfer','record.redate','record.delete','assignment.add','employment.parallel','probation.start','probation.confirm'));
CREATE TABLE access.employment_probation_fixture (
    instance_id uuid NOT NULL,principal_id uuid NOT NULL,PRIMARY KEY(instance_id,principal_id),
    FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture
);

-- +goose Down
SELECT 1/0;
