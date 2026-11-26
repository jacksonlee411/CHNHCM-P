-- +goose Up
CREATE TABLE staffing.duty_observation (
 instance_id uuid NOT NULL,id uuid NOT NULL,assignment_id uuid NOT NULL,issuer text NOT NULL,number text NOT NULL,
 PRIMARY KEY(instance_id,id),UNIQUE(instance_id,assignment_id,issuer,number),
 FOREIGN KEY(instance_id,assignment_id) REFERENCES staffing.assignment
);
CREATE TABLE staffing.duty_observation_revision (
 instance_id uuid NOT NULL, observation_id uuid NOT NULL,version uuid NOT NULL, previous uuid,
 actor uuid NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),digest text NOT NULL,
 dispatch uuid NOT NULL,dispatch_version uuid NOT NULL,fact jsonb NOT NULL,
 PRIMARY KEY(instance_id,observation_id,version),UNIQUE(instance_id,version),
 FOREIGN KEY(instance_id,observation_id) REFERENCES staffing.duty_observation,
 FOREIGN KEY(instance_id,observation_id,previous) REFERENCES staffing.duty_observation_revision,
 FOREIGN KEY(instance_id,dispatch,dispatch_version) REFERENCES staffing.appointment_version(instance_id,source,version)
);
CREATE TABLE staffing.duty_observation_head (
 instance_id uuid NOT NULL,observation_id uuid NOT NULL,version uuid NOT NULL,
 PRIMARY KEY(instance_id,observation_id),FOREIGN KEY(instance_id,observation_id,version) REFERENCES staffing.duty_observation_revision
);
CREATE TABLE staffing.duty_application (
 instance_id uuid NOT NULL,observation_id uuid NOT NULL,action_id uuid NOT NULL,
 PRIMARY KEY(instance_id,observation_id),FOREIGN KEY(instance_id,observation_id) REFERENCES staffing.duty_observation,
 FOREIGN KEY(instance_id,action_id) REFERENCES staffing.employment_action(instance_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.duty_observation FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.duty_observation_revision FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.duty_application FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.duty_observation,staffing.duty_observation_revision,staffing.duty_application,staffing.duty_observation_head TO chnhcm_app;
GRANT UPDATE(version) ON staffing.duty_observation_head TO chnhcm_app;
ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CHECK(action IN ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory','register.read','transfer','record.redate','record.delete','assignment.add','employment.parallel','probation.start','probation.confirm','grade.record','promotion','promotion.withdraw','manager.change','assignment.end','employment.leave','employment.rehire','position.handover','duty.verify','duty.correct'));
CREATE TABLE access.duty_fixture(instance_id uuid NOT NULL,principal_id uuid NOT NULL,PRIMARY KEY(instance_id,principal_id),FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture);
-- +goose Down
SELECT 1/0;
