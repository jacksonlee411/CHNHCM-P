-- +goose Up
-- Add one current-model read-consumer permission; no organization history is changed.
ALTER TABLE access.subject_grant DROP CONSTRAINT subject_grant_action_check;
ALTER TABLE access.subject_grant ADD CONSTRAINT subject_grant_action_check CHECK(action IN ('bu.read','bu.history.read','bu.export','bu.create','bu.record.add','bu.record.correct','bu.record.delete','bu.deactivation.delete','bu.reactivation.delete','bu.record.redate','bu.first.redate','bu.deactivation.redate','bu.reactivation.redate','bu.deactivate','bu.reactivate','bu.timeline.revise','bu.first.reclassify','bu.record.reclassify','bu.deactivation.reclassify','bu.reactivation.reclassify'));

-- +goose Down
-- Preserve grants and audit; automatic permission removal is intentionally unsupported.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Export permission migration cannot be automatically reverted'; END $$;
-- +goose StatementEnd
