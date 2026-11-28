-- +goose Up
-- X03 grants are explicit per real term. Upgrading an existing installation grants no access to personal salary.
CREATE TABLE access.base_pay_grant (
 instance_id uuid NOT NULL, principal_id uuid NOT NULL, assignment_id uuid NOT NULL,
 action text NOT NULL CHECK(action IN ('read','history','first','adjust','correct','withdraw','use')),
 plan_id uuid, CHECK(plan_id IS NULL OR action='use'),
 UNIQUE NULLS NOT DISTINCT(instance_id,principal_id,assignment_id,action,plan_id),
 FOREIGN KEY(instance_id,principal_id) REFERENCES access.principal
);
CREATE TRIGGER guard_base_pay BEFORE INSERT OR UPDATE OR DELETE ON access.base_pay_grant
 FOR EACH ROW EXECUTE FUNCTION access.person_grant_guard();
GRANT SELECT ON access.base_pay_grant TO chnhcm_app;

CREATE TABLE compensation.award_revision (
 instance_id uuid NOT NULL, assignment_id uuid NOT NULL, award_id uuid NOT NULL, revision uuid NOT NULL,
 effective_date date NOT NULL, amount numeric NOT NULL CHECK(amount>=0 AND scale(amount)<=2),
 previous uuid, action text NOT NULL CHECK(action IN ('FIRST','ADJUST','CORRECT','WITHDRAW')),
 reason text NOT NULL, note text NOT NULL, adoption jsonb NOT NULL,
 operator_id uuid NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(instance_id,award_id,revision), UNIQUE(instance_id,assignment_id,award_id,revision),
 FOREIGN KEY(instance_id,award_id,previous) REFERENCES compensation.award_revision
);
CREATE INDEX award_log_by_term ON compensation.award_revision(instance_id,assignment_id,effective_date,recorded_at);
CREATE TABLE compensation.award_timeline (
 instance_id uuid NOT NULL, assignment_id uuid NOT NULL, head uuid NOT NULL, PRIMARY KEY(instance_id,assignment_id)
);
CREATE TABLE compensation.award_period (
 instance_id uuid NOT NULL, assignment_id uuid NOT NULL, award_id uuid NOT NULL, revision uuid NOT NULL,
 start_date date NOT NULL, until_date date, valid_period daterange GENERATED ALWAYS AS (daterange(start_date,until_date,'[)')) STORED,
 CHECK(until_date IS NULL OR until_date>start_date), PRIMARY KEY(instance_id,assignment_id,start_date),
 UNIQUE(instance_id,award_id), FOREIGN KEY(instance_id,assignment_id,award_id,revision) REFERENCES compensation.award_revision(instance_id,assignment_id,award_id,revision),
 EXCLUDE USING gist(instance_id WITH =,assignment_id WITH =,valid_period WITH &&)
);
-- A negative resolution fences a delayed first transmission. Only an explicit retry
-- carrying the original key and confirmation can consume it.
CREATE TABLE compensation.award_recovery (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL,
 confirmation uuid NOT NULL, digest bytea CHECK(octet_length(digest)=32),
 PRIMARY KEY(instance_id,subject_id,submission_id)
);
CREATE TRIGGER immutable_confirmation BEFORE UPDATE OR DELETE ON compensation.award_recovery FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
GRANT SELECT,INSERT ON compensation.award_recovery TO chnhcm_app;
CREATE TABLE compensation.award_receipt (
 instance_id uuid NOT NULL, subject_id uuid NOT NULL, submission_id uuid NOT NULL, digest bytea NOT NULL CHECK(octet_length(digest)=32),
 award_id uuid NOT NULL, revision uuid NOT NULL,
 PRIMARY KEY(instance_id,subject_id,submission_id),
 FOREIGN KEY(instance_id,award_id,revision) REFERENCES compensation.award_revision
);
CREATE TABLE compensation.award_outbox (
 instance_id uuid NOT NULL, assignment_id uuid NOT NULL, award_id uuid NOT NULL, revision uuid NOT NULL,
 PRIMARY KEY(instance_id,award_id,revision),
 FOREIGN KEY(instance_id,assignment_id,award_id,revision) REFERENCES compensation.award_revision(instance_id,assignment_id,award_id,revision)
);
CREATE TRIGGER immutable_revision BEFORE UPDATE OR DELETE ON compensation.award_revision FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
CREATE TRIGGER immutable_receipt BEFORE UPDATE OR DELETE ON compensation.award_receipt FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
CREATE TRIGGER immutable_event BEFORE UPDATE OR DELETE ON compensation.award_outbox FOR EACH ROW EXECUTE FUNCTION compensation.immutable_fact();
GRANT SELECT,INSERT ON compensation.award_revision,compensation.award_timeline,compensation.award_period,compensation.award_receipt,compensation.award_outbox TO chnhcm_app;
GRANT UPDATE(head) ON compensation.award_timeline TO chnhcm_app;
GRANT UPDATE(until_date,revision),DELETE ON compensation.award_period TO chnhcm_app;
-- +goose Down
SELECT 1/0;
