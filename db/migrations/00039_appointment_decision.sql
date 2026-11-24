-- +goose Up
ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
 ADD CHECK(action IN ('hire','read','cancel','history','appointment.issue'));
CREATE TABLE access.appointment_issuer_fixture (
 instance_id uuid NOT NULL, principal_id uuid NOT NULL, PRIMARY KEY(instance_id,principal_id),
 FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture
);
-- Each publication seals the entire original, including its stable roster identities.
CREATE TABLE staffing.appointment_version (
 instance_id uuid NOT NULL, source uuid NOT NULL, version uuid NOT NULL, edition integer NOT NULL CHECK(edition>0),
 number text NOT NULL, roster boolean NOT NULL, withdrawn boolean NOT NULL, members jsonb NOT NULL CHECK(jsonb_typeof(members)='array'),
 digest text NOT NULL, actor uuid NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,source,version), UNIQUE(instance_id,source,edition)
);
CREATE UNIQUE INDEX appointment_original_number ON staffing.appointment_version(instance_id,number) WHERE edition=1;
CREATE TABLE staffing.appointment_outbox (
 instance_id uuid NOT NULL, source uuid NOT NULL, version uuid NOT NULL, PRIMARY KEY(instance_id,source,version),
 FOREIGN KEY(instance_id,source,version) REFERENCES staffing.appointment_version
);
CREATE TABLE staffing.hire_decision_use (
 instance_id uuid NOT NULL, source uuid NOT NULL, version uuid NOT NULL, line_id uuid,
 action_id uuid NOT NULL, PRIMARY KEY(instance_id,action_id),
 FOREIGN KEY(instance_id,action_id) REFERENCES staffing.employment_action,
 FOREIGN KEY(instance_id,source,version) REFERENCES staffing.appointment_version
);
CREATE UNIQUE INDEX hire_single_decision ON staffing.hire_decision_use(instance_id,source) WHERE line_id IS NULL;
CREATE UNIQUE INDEX hire_roster_line ON staffing.hire_decision_use(instance_id,source,line_id) WHERE line_id IS NOT NULL;
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.appointment_version FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.appointment_outbox FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.hire_decision_use FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.appointment_version,staffing.appointment_outbox,staffing.hire_decision_use TO chnhcm_app;
-- +goose Down
SELECT 1/0;
