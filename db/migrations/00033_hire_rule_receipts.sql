-- +goose Up
-- Accepted submissions already live in the immutable source row.

CREATE TABLE organization.legal_purpose_terminal (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL, owner_id uuid,
 outcome text NOT NULL, version uuid, input_digest bytea,
 failure_status integer, failure_code text, failure_message text,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,subject_id,submission_id),
 FOREIGN KEY(instance_id,owner_id,version) REFERENCES organization.legal_purpose_source,
 CHECK((owner_id IS NULL) = (outcome='not_applied')),
 CHECK((outcome='not_applied' AND version IS NULL AND input_digest IS NULL)
    OR (outcome='no_change' AND version IS NOT NULL AND input_digest IS NOT NULL)
    OR (outcome='rejected' AND version IS NULL AND input_digest IS NOT NULL)),
 CHECK(input_digest IS NULL OR octet_length(input_digest)=32),
 CHECK((outcome='rejected' AND failure_status IS NOT NULL AND failure_status IN (400,409) AND failure_code IS NOT NULL AND failure_message IS NOT NULL)
    OR (outcome<>'rejected' AND failure_status IS NULL AND failure_code IS NULL AND failure_message IS NULL))
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON organization.legal_purpose_terminal FOR EACH ROW EXECUTE FUNCTION organization.immutable_history();
GRANT SELECT,INSERT ON organization.legal_purpose_terminal TO chnhcm_app;

CREATE TABLE staffing.assignment_policy_terminal (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL, owner_id uuid NOT NULL,
 outcome text NOT NULL, version uuid, input_digest bytea,
 failure_status integer, failure_code text, failure_message text,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,subject_id,submission_id),
 FOREIGN KEY(instance_id,owner_id,version) REFERENCES staffing.assignment_policy_source,
 CHECK((outcome='not_applied' AND version IS NULL AND input_digest IS NULL)
    OR (outcome='no_change' AND version IS NOT NULL AND input_digest IS NOT NULL)
    OR (outcome='rejected' AND version IS NULL AND input_digest IS NOT NULL)),
 CHECK(input_digest IS NULL OR octet_length(input_digest)=32),
 CHECK((outcome='rejected' AND failure_status IS NOT NULL AND failure_status IN (400,409) AND failure_code IS NOT NULL AND failure_message IS NOT NULL)
    OR (outcome<>'rejected' AND failure_status IS NULL AND failure_code IS NULL AND failure_message IS NULL))
);
CREATE TRIGGER immutable_terminal BEFORE UPDATE OR DELETE ON staffing.assignment_policy_terminal FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.assignment_policy_terminal TO chnhcm_app;

-- +goose Down
SELECT 1/0;
