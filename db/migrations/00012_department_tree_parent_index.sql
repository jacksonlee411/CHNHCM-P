-- +goose Up
-- 00009 is reserved by wt-dev-c for its country catalog. This additive index
-- keeps the existing Go-owned facts and all immutable source layouts intact.
CREATE INDEX department_tree_parent ON organization.department_read_segment
 (instance_id,q_parent,q_code_key COLLATE "C",object_id)
 INCLUDE(span,q_bu,q_path_date) WHERE q_state='active';

-- +goose Down
DROP INDEX organization.department_tree_parent;
