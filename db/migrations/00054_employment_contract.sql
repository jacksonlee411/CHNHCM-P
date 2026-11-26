-- +goose Up
CREATE TABLE staffing.employment_contract (
    instance_id uuid NOT NULL, id uuid NOT NULL, employment_id uuid NOT NULL, number text NOT NULL,
    signed date NOT NULL, period daterange NOT NULL, predecessor uuid,
    PRIMARY KEY(instance_id,id), UNIQUE(instance_id,employment_id,id), UNIQUE(instance_id,employment_id,number),
    UNIQUE(instance_id,predecessor), CHECK(predecessor<>id AND NOT isempty(period) AND NOT lower_inf(period)),
    FOREIGN KEY(instance_id,employment_id) REFERENCES staffing.employment,
    FOREIGN KEY(instance_id,employment_id,predecessor) REFERENCES staffing.employment_contract(instance_id,employment_id,id)
);
CREATE TABLE staffing.contract_process (
    instance_id uuid NOT NULL, contract_id uuid NOT NULL, version uuid NOT NULL, operation text NOT NULL CHECK(operation IN ('sign','renew','end')),
    business_date date NOT NULL, actor uuid NOT NULL, operator_name text NOT NULL, recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    basis text NOT NULL, leave_action uuid, PRIMARY KEY(instance_id,contract_id,version), UNIQUE(instance_id,contract_id,operation),
    FOREIGN KEY(instance_id,contract_id) REFERENCES staffing.employment_contract,
    FOREIGN KEY(instance_id,leave_action) REFERENCES staffing.employment_action(instance_id,id)
);
CREATE UNIQUE INDEX contract_single_origin ON staffing.contract_process(instance_id,contract_id) WHERE operation IN ('sign','renew');
CREATE TABLE staffing.contract_submission (
    instance_id uuid NOT NULL,actor uuid NOT NULL,submission_id uuid NOT NULL,digest text NOT NULL,assignment_id uuid,contract_id uuid, version uuid,
    PRIMARY KEY(instance_id,actor,submission_id),CHECK ((contract_id IS NULL)=(version IS NULL) AND (contract_id IS NULL)=(assignment_id IS NULL)),
    FOREIGN KEY(instance_id,assignment_id) REFERENCES staffing.assignment,
    FOREIGN KEY(instance_id,contract_id,version) REFERENCES staffing.contract_process
);
CREATE TABLE staffing.contract_outbox (
    instance_id uuid NOT NULL,contract_id uuid NOT NULL,version uuid NOT NULL,
    PRIMARY KEY(instance_id,contract_id,version), FOREIGN KEY(instance_id,contract_id,version) REFERENCES staffing.contract_process
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.employment_contract FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.contract_process FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.contract_submission FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.employment_contract,staffing.contract_process,staffing.contract_submission,staffing.contract_outbox TO chnhcm_app;
ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CHECK(action IN ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory','register.read','transfer','record.redate','record.delete','assignment.add','employment.parallel','probation.start','probation.confirm','grade.record','promotion','promotion.withdraw','manager.change','assignment.end','employment.leave','employment.rehire','position.handover','duty.verify','duty.correct','contract.read','contract.manage'));
CREATE TABLE access.contract_fixture(instance_id uuid NOT NULL,principal_id uuid NOT NULL,PRIMARY KEY(instance_id,principal_id),FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture);
-- +goose Down
SELECT 1/0;
