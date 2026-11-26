-- +goose Up
CREATE TABLE staffing.position_succession (
    instance_id uuid NOT NULL, id uuid NOT NULL, position_id uuid NOT NULL,
    assignment_id uuid NOT NULL, action_id uuid NOT NULL,
    incumbent uuid NOT NULL, incumbent_source uuid NOT NULL, period daterange NOT NULL,
    PRIMARY KEY(instance_id,id),
    CHECK(assignment_id<>incumbent AND NOT isempty(period) AND NOT lower_inf(period) AND NOT upper_inf(period)),
    FOREIGN KEY(instance_id,assignment_id) REFERENCES staffing.assignment,
    FOREIGN KEY(instance_id,action_id) REFERENCES staffing.employment_action(instance_id,id) DEFERRABLE INITIALLY DEFERRED,
    FOREIGN KEY(instance_id,incumbent,incumbent_source) REFERENCES staffing.assignment_publication(instance_id,assignment_id,version)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.position_succession FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.position_succession TO chnhcm_app;
ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CHECK(action IN ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory','register.read','transfer','record.redate','record.delete','assignment.add','employment.parallel','probation.start','probation.confirm','grade.record','promotion','promotion.withdraw','manager.change','assignment.end','employment.leave','employment.rehire','position.handover'));
CREATE TABLE access.employment_succession_fixture (
    instance_id uuid NOT NULL,principal_id uuid NOT NULL,PRIMARY KEY(instance_id,principal_id),
    FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture
);
-- +goose Down
SELECT 1/0;
