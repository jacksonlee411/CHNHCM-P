-- +goose Up
ALTER TABLE staffing.assignment_segment ADD COLUMN grade_ref jsonb CHECK(grade_ref IS NULL OR
    (jsonb_typeof(grade_ref)='object' AND grade_ref ?& ARRAY['id','name','sourceVersion','revision','job','jobSource','jobRevision']));
ALTER TABLE staffing.employment_action DROP CONSTRAINT employment_action_kind_check,
    ADD CHECK(kind IN ('Hire','CancelHire','DataChange','Transfer','AdditionalAssignment','ParallelEmployment','ProbationStart','ConfirmEmployment','GradeRecord','Promotion'));
ALTER TABLE staffing.employment_action_revision DROP CONSTRAINT employment_action_revision_operation_check,
    ADD CHECK(operation IN ('Add','Insert','Correct','Redate','Delete','Rebase','Withdraw'));
-- Each receipt recovers the two exact originals issued together, even after a correction.
CREATE TABLE staffing.promotion_transfer (
    instance_id uuid NOT NULL,assignment_id uuid NOT NULL,promotion uuid NOT NULL,promotion_revision uuid NOT NULL,
    transfer uuid NOT NULL,transfer_revision uuid NOT NULL,PRIMARY KEY(instance_id,promotion),UNIQUE(instance_id,transfer),
    FOREIGN KEY(instance_id,assignment_id,promotion,promotion_revision) REFERENCES staffing.employment_action_revision(instance_id,assignment_id,action_id,revision),
    FOREIGN KEY(instance_id,assignment_id,transfer,transfer_revision) REFERENCES staffing.employment_action_revision(instance_id,assignment_id,action_id,revision),
    CHECK(promotion<>transfer)
);
CREATE TRIGGER immutable_fact BEFORE UPDATE OR DELETE ON staffing.promotion_transfer FOR EACH ROW EXECUTE FUNCTION staffing.immutable_fact();
GRANT SELECT,INSERT ON staffing.promotion_transfer TO chnhcm_app;
ALTER TABLE access.employment_grant DROP CONSTRAINT employment_grant_action_check,
    ADD CHECK(action IN ('hire','read','cancel','history','appointment.issue','location.change','location.correct','directory','register.read','transfer','record.redate','record.delete','assignment.add','employment.parallel','probation.start','probation.confirm','grade.record','promotion','promotion.withdraw'));
CREATE TABLE access.employment_promotion_fixture (
    instance_id uuid NOT NULL,principal_id uuid NOT NULL,PRIMARY KEY(instance_id,principal_id),
    FOREIGN KEY(instance_id,principal_id) REFERENCES access.employment_fixture
);

-- +goose Down
SELECT 1/0;
