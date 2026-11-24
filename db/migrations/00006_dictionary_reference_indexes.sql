-- +goose Up
-- Dictionary IDs are globally unique integer identities. Index the referencing
-- side so removing unused encoding material does not scan history per FK check.
CREATE INDEX bu_history_dictionary_reference ON organization.bu_history(dictionary_id);
CREATE INDEX bu_terminal_dictionary_reference ON organization.bu_terminal(dictionary_id);
CREATE INDEX department_history_dictionary_reference ON organization.department_history(dictionary_id);
CREATE INDEX department_terminal_dictionary_reference ON organization.department_terminal(dictionary_id);

-- +goose Down
DROP INDEX organization.department_terminal_dictionary_reference;
DROP INDEX organization.department_history_dictionary_reference;
DROP INDEX organization.bu_terminal_dictionary_reference;
DROP INDEX organization.bu_history_dictionary_reference;
