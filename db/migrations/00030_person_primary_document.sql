-- +goose Up
-- Pre-existing identity records have no document. A material reference is never an identity number.
ALTER TABLE personnel.identity_payload
    ADD COLUMN document_kind text,
    ADD COLUMN document_number text COLLATE "C",
    ADD COLUMN current_document boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT primary_document_complete CHECK (
        (document_kind IS NULL AND document_number IS NULL AND NOT current_document) OR
        (document_kind IS NOT NULL AND document_kind IN ('resident_id','passport','other')
         AND document_number IS NOT NULL AND document_number <> ''
         AND document_number = translate(btrim(document_number), 'abcdefghijklmnopqrstuvwxyz', 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'))
    );
CREATE UNIQUE INDEX current_document_number ON personnel.identity_payload(instance_id, document_number) WHERE current_document;
CREATE UNIQUE INDEX current_document_person ON personnel.identity_payload(instance_id, person_id) WHERE current_document;
GRANT UPDATE(current_document) ON personnel.identity_payload TO chnhcm_app;
DROP INDEX personnel.person_candidate_name;

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'personnel identity history requires forward migration'; END $$;
-- +goose StatementEnd
