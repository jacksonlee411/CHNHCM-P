-- +goose Up
CREATE TABLE access.employment_grant (
 instance_id uuid NOT NULL, principal_id uuid NOT NULL, grant_id uuid NOT NULL DEFAULT gen_random_uuid(),
 action text NOT NULL CHECK(action IN ('hire','read','cancel','history')), fields text[] NOT NULL,
 person_id uuid, employer_id uuid, department_id uuid, expires_at timestamptz,
 PRIMARY KEY(instance_id,grant_id), FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal
);
CREATE TRIGGER guard_employment_grant BEFORE INSERT OR UPDATE OR DELETE ON access.employment_grant
 FOR EACH ROW EXECUTE FUNCTION access.person_grant_guard();
CREATE TABLE access.employment_fixture(instance_id uuid NOT NULL,principal_id uuid NOT NULL,PRIMARY KEY(instance_id,principal_id),FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal);
GRANT SELECT ON access.employment_grant TO chnhcm_app;
CREATE TABLE staffing.employment (
 instance_id uuid NOT NULL, id uuid NOT NULL, person_id uuid NOT NULL, employer_id uuid NOT NULL,
 relationship text NOT NULL, period daterange NOT NULL CHECK(NOT isempty(period) AND NOT lower_inf(period)),
 PRIMARY KEY(instance_id,id)
);
CREATE INDEX employment_person ON staffing.employment(instance_id,person_id);
CREATE INDEX employment_employer ON staffing.employment(instance_id,employer_id);
CREATE TABLE staffing.assignment (
 instance_id uuid NOT NULL, id uuid NOT NULL, employment_id uuid NOT NULL, department_id uuid NOT NULL, job_id uuid NOT NULL,
 bu_id uuid, location_id uuid, PRIMARY KEY(instance_id,id), FOREIGN KEY(instance_id,employment_id) REFERENCES staffing.employment
);
CREATE TABLE staffing.assignment_segment (
 instance_id uuid NOT NULL, assignment_id uuid NOT NULL, period daterange NOT NULL CHECK(NOT isempty(period) AND NOT lower_inf(period)), adoption jsonb NOT NULL,
 EXCLUDE USING gist(instance_id WITH =,assignment_id WITH =,period WITH &&), FOREIGN KEY(instance_id,assignment_id) REFERENCES staffing.assignment
);
CREATE TABLE staffing.employment_action (
 instance_id uuid NOT NULL, id uuid NOT NULL, revision uuid NOT NULL, kind text NOT NULL CHECK(kind IN ('Hire','CancelHire')),
 employment_id uuid NOT NULL, assignment_id uuid NOT NULL, original_id uuid, business_date date NOT NULL,
 actor uuid NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(), notice text NOT NULL, adoption jsonb NOT NULL,
 PRIMARY KEY(instance_id,id), UNIQUE(instance_id,original_id), UNIQUE(instance_id,employment_id,kind),
 FOREIGN KEY(instance_id,employment_id) REFERENCES staffing.employment, FOREIGN KEY(instance_id,assignment_id) REFERENCES staffing.assignment,
 FOREIGN KEY(instance_id,original_id) REFERENCES staffing.employment_action,
 CHECK((kind='Hire')=(original_id IS NULL))
);
-- Current primary choices can be withdrawn. Their original targets and policy remain in the immutable action.
CREATE TABLE staffing.employment_primary (
 instance_id uuid NOT NULL, action_id uuid NOT NULL, layer text NOT NULL CHECK(layer IN ('employment','assignment')),
 owner_id uuid NOT NULL, target_id uuid NOT NULL, purpose text NOT NULL, period daterange NOT NULL CHECK(NOT isempty(period) AND NOT lower_inf(period)),
 PRIMARY KEY(instance_id,action_id,layer), FOREIGN KEY(instance_id,action_id) REFERENCES staffing.employment_action,
 EXCLUDE USING gist(instance_id WITH =,layer WITH =,owner_id WITH =,purpose WITH =,period WITH &&)
);
CREATE TABLE staffing.hire_terminal (
 instance_id uuid NOT NULL, actor uuid NOT NULL, submission_id uuid NOT NULL, digest text NOT NULL,
 outcome text NOT NULL CHECK(outcome IN ('accepted','rejected','not_applied')), action_id uuid,
 PRIMARY KEY(instance_id,actor,submission_id), FOREIGN KEY(instance_id,action_id) REFERENCES staffing.employment_action
);
CREATE TABLE staffing.employment_outbox (
 instance_id uuid NOT NULL, event_id uuid NOT NULL, action_id uuid NOT NULL,
 PRIMARY KEY(instance_id,event_id), UNIQUE(instance_id,action_id), FOREIGN KEY(instance_id,action_id) REFERENCES staffing.employment_action
);
-- +goose StatementBegin
DO $$ DECLARE rel text; BEGIN
 FOREACH rel IN ARRAY ARRAY['employment','assignment','assignment_segment','employment_action','hire_terminal','employment_outbox'] LOOP
 EXECUTE format('CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.%I FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact()',rel);
 END LOOP;
END $$;
-- +goose StatementEnd
GRANT SELECT,INSERT ON staffing.employment,staffing.assignment,staffing.assignment_segment,staffing.employment_action,staffing.hire_terminal,staffing.employment_outbox,staffing.employment_primary TO chnhcm_app;
GRANT DELETE ON staffing.employment_primary TO chnhcm_app;
-- +goose Down
SELECT 1/0;
