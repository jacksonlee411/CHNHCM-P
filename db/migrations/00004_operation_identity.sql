-- +goose Up
ALTER TABLE access.principal
 ADD COLUMN display_name text CHECK(display_name IS NULL OR length(btrim(display_name))>0),
 ADD COLUMN principal_kind text NOT NULL DEFAULT 'human' CHECK(principal_kind IN ('human','service'));
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'operation identities cannot be automatically removed'; END $$;
-- +goose StatementEnd
