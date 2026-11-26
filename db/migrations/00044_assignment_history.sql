-- +goose Up
-- All sources through version 43 share the original assignment's department and job.
ALTER TABLE staffing.assignment_segment ADD COLUMN department_id uuid, ADD COLUMN job_id uuid;
ALTER TABLE staffing.assignment_segment DISABLE TRIGGER immutable_fact;
UPDATE staffing.assignment_segment s SET department_id=t.department_id,job_id=t.job_id FROM staffing.assignment t
    WHERE (s.instance_id,s.assignment_id)=(t.instance_id,t.id);
ALTER TABLE staffing.assignment_segment ENABLE TRIGGER immutable_fact;
ALTER TABLE staffing.assignment_segment ALTER COLUMN department_id SET NOT NULL, ALTER COLUMN job_id SET NOT NULL;

ALTER TABLE staffing.employment_action DROP CONSTRAINT employment_action_kind_check,
    ADD CONSTRAINT employment_action_kind_check CHECK(kind IN ('Hire','CancelHire','DataChange','Transfer'));
ALTER TABLE staffing.employment_action_revision ADD COLUMN operation text NOT NULL DEFAULT 'Add';
ALTER TABLE staffing.employment_action_revision DISABLE TRIGGER immutable_fact;
UPDATE staffing.employment_action_revision SET operation='Correct' WHERE previous_revision IS NOT NULL;
ALTER TABLE staffing.employment_action_revision ENABLE TRIGGER immutable_fact;
ALTER TABLE staffing.employment_action_revision ADD CHECK (operation IN ('Add','Insert','Correct','Redate','Delete','Rebase'));

-- Several dependent action revisions can first appear in a single new daily source.
CREATE TABLE staffing.employment_revision_publication(
    instance_id uuid NOT NULL, assignment_id uuid NOT NULL, action_id uuid NOT NULL, revision uuid NOT NULL, version uuid NOT NULL,
    PRIMARY KEY(instance_id,action_id,revision),
    FOREIGN KEY(instance_id,assignment_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,assignment_id,action_id,revision),
    FOREIGN KEY(instance_id,assignment_id,version) REFERENCES staffing.assignment_publication
);
INSERT INTO staffing.employment_revision_publication SELECT instance_id,assignment_id,action_id,revision,version FROM staffing.assignment_source;
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.employment_revision_publication FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
-- +goose StatementBegin
CREATE FUNCTION staffing.record_revision_publication() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO staffing.employment_revision_publication(instance_id,assignment_id,action_id,revision,version)
    SELECT m.instance_id,m.assignment_id,m.action_id,m.revision,m.version FROM staffing.assignment_member m
    WHERE (m.instance_id,m.assignment_id,m.version)=(NEW.instance_id,NEW.assignment_id,NEW.version)
    ON CONFLICT(instance_id,action_id,revision) DO NOTHING;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER revision_publication AFTER INSERT ON staffing.assignment_publication FOR EACH ROW EXECUTE FUNCTION staffing.record_revision_publication();
GRANT SELECT,INSERT ON staffing.employment_revision_publication TO chnhcm_app;

ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CONSTRAINT employment_grant_action_check CHECK(action IN ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory','register.read','transfer','record.redate','record.delete'));
CREATE TABLE access.employment_history_fixture (
    instance_id uuid NOT NULL, principal_id uuid NOT NULL, PRIMARY KEY(instance_id,principal_id),
    FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture
);

-- +goose Down
SELECT 1/0;
