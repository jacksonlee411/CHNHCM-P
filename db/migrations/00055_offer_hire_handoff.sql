-- +goose Up
ALTER TABLE recruiting.offer_revision DROP CONSTRAINT offer_revision_operation_check,
    ADD CHECK(operation IN ('issue','accept','release','reserve','convert','cancel_conversion','decline','reschedule'));
CREATE TABLE recruiting.offer_revised_terms (
    instance_id uuid NOT NULL,offer_id uuid NOT NULL,version uuid NOT NULL,terms jsonb NOT NULL,
    PRIMARY KEY(instance_id,offer_id,version), FOREIGN KEY(instance_id,offer_id,version) REFERENCES recruiting.offer_revision
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON recruiting.offer_revised_terms FOR EACH ROW EXECUTE FUNCTION recruiting.immutable_fact();
GRANT SELECT,INSERT ON recruiting.offer_revised_terms TO chnhcm_app;
CREATE TABLE recruiting.hire_handoff (
    instance_id uuid NOT NULL,offer_id uuid NOT NULL,accepted_version uuid NOT NULL,converted_version uuid NOT NULL,
    hire_action uuid NOT NULL,hire_revision uuid NOT NULL,assignment_id uuid NOT NULL,
    PRIMARY KEY(instance_id,offer_id,accepted_version),UNIQUE(instance_id,hire_action),
    FOREIGN KEY(instance_id,offer_id,accepted_version) REFERENCES recruiting.offer_revision,
    FOREIGN KEY(instance_id,offer_id,converted_version) REFERENCES recruiting.offer_revision
);
CREATE TABLE recruiting.handoff_cancellation (
    instance_id uuid NOT NULL,offer_id uuid NOT NULL,accepted_version uuid NOT NULL,cancelled_version uuid NOT NULL,
    cancel_action uuid NOT NULL,cancel_revision uuid NOT NULL,
    PRIMARY KEY(instance_id,offer_id,accepted_version),UNIQUE(instance_id,cancel_action),
    FOREIGN KEY(instance_id,offer_id,accepted_version) REFERENCES recruiting.hire_handoff,
    FOREIGN KEY(instance_id,offer_id,cancelled_version) REFERENCES recruiting.offer_revision
);
CREATE TABLE staffing.hired_offer (
    instance_id uuid NOT NULL,offer_id uuid NOT NULL,accepted_version uuid NOT NULL,conversion_version uuid NOT NULL,
    action_id uuid NOT NULL,revision uuid NOT NULL,
    PRIMARY KEY(instance_id,offer_id,accepted_version),UNIQUE(instance_id,action_id),
    FOREIGN KEY(instance_id,action_id,revision) REFERENCES staffing.employment_action_revision(instance_id,action_id,revision)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON recruiting.hire_handoff FOR EACH ROW EXECUTE FUNCTION recruiting.immutable_fact();
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON recruiting.handoff_cancellation FOR EACH ROW EXECUTE FUNCTION recruiting.immutable_fact();
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.hired_offer FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON recruiting.hire_handoff,recruiting.handoff_cancellation,staffing.hired_offer TO chnhcm_app;
-- +goose Down
SELECT 1/0;
