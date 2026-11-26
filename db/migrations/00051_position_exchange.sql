-- +goose Up
CREATE TABLE staffing.transfer_exchange (
    instance_id uuid NOT NULL, id uuid NOT NULL,
    first_assignment uuid NOT NULL, second_assignment uuid NOT NULL,
    first_action uuid NOT NULL, second_action uuid NOT NULL,
    first_position uuid NOT NULL, second_position uuid NOT NULL,
    PRIMARY KEY(instance_id,id), UNIQUE(instance_id,first_action), UNIQUE(instance_id,second_action),
    CHECK(first_assignment<>second_assignment AND first_position<>second_position AND first_action<>second_action),
    FOREIGN KEY(instance_id,first_action) REFERENCES staffing.employment_action(instance_id,id),
    FOREIGN KEY(instance_id,second_action) REFERENCES staffing.employment_action(instance_id,id)
);
CREATE TABLE staffing.transfer_exchange_revision (
    instance_id uuid NOT NULL, exchange_id uuid NOT NULL, version uuid NOT NULL,
    first_assignment uuid NOT NULL, first_action uuid NOT NULL, first_revision uuid NOT NULL,
    second_assignment uuid NOT NULL, second_action uuid NOT NULL, second_revision uuid NOT NULL,
    PRIMARY KEY(instance_id,exchange_id,version),
    UNIQUE(instance_id,first_action,first_revision),UNIQUE(instance_id,second_action,second_revision),
    FOREIGN KEY(instance_id,exchange_id) REFERENCES staffing.transfer_exchange,
    FOREIGN KEY(instance_id,first_assignment,first_action,first_revision) REFERENCES staffing.employment_action_revision(instance_id,assignment_id,action_id,revision),
    FOREIGN KEY(instance_id,second_assignment,second_action,second_revision) REFERENCES staffing.employment_action_revision(instance_id,assignment_id,action_id,revision)
);
CREATE TRIGGER immutable_exchange BEFORE UPDATE OR DELETE ON staffing.transfer_exchange FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_exchange_revision BEFORE UPDATE OR DELETE ON staffing.transfer_exchange_revision FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT, INSERT ON staffing.transfer_exchange,staffing.transfer_exchange_revision TO chnhcm_app;
-- +goose Down
SELECT 1/0;
