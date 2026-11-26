-- +goose Up
-- A relationship decision can be reached through several assignments. Keep
-- the return context of new submissions alongside their immutable terminals.
CREATE TABLE staffing.probation_receipt_assignment (
    instance_id uuid NOT NULL,
    actor uuid NOT NULL,
    submission_id uuid NOT NULL,
    assignment_id uuid NOT NULL,
    PRIMARY KEY (instance_id, actor, submission_id),
    FOREIGN KEY (instance_id, actor, submission_id) REFERENCES staffing.hire_terminal,
    FOREIGN KEY (instance_id, assignment_id) REFERENCES staffing.assignment(instance_id, id)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.probation_receipt_assignment
    FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT, INSERT ON staffing.probation_receipt_assignment TO chnhcm_app;

-- +goose Down
SELECT 1/0;
