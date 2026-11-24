-- +goose Up
CREATE TABLE staffing.hire_rule_source (
 instance_id uuid NOT NULL,
 version text NOT NULL CHECK (version <> ''),
 period daterange NOT NULL CHECK (NOT isempty(period) AND NOT lower_inf(period)),
 relationship text NOT NULL,
 action text NOT NULL,
 initial_state text NOT NULL,
 primary_purpose text NOT NULL,
 reasons text[] NOT NULL CHECK (cardinality(reasons)>0),
 accepted_documents text[] NOT NULL CHECK (cardinality(accepted_documents)>0),
 require_known_expiry boolean NOT NULL,
 require_inspection boolean NOT NULL,
 subject_id uuid NOT NULL,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY (instance_id, version),
 CONSTRAINT hire_rule_period_unique EXCLUDE USING gist (instance_id WITH =, relationship WITH =, period WITH &&)
);
CREATE TRIGGER immutable_source BEFORE UPDATE OR DELETE ON staffing.hire_rule_source
 FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT ON staffing.hire_rule_source TO chnhcm_app;
-- +goose Down
SELECT 1/0;
