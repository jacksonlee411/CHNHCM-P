-- +goose Up
-- This predecessor published two different purposes. Only management reporting
-- belongs to the current model. Remove positively identified functional data
-- in FK order while all six tables are locked in the same migration transaction.
LOCK TABLE staffing.position_hierarchy, staffing.position_hierarchy_version,
 staffing.position_hierarchy_node, staffing.position_hierarchy_edge,
 staffing.position_hierarchy_terminal, staffing.position_hierarchy_outbox IN ACCESS EXCLUSIVE MODE;
SET CONSTRAINTS ALL IMMEDIATE;
CREATE TEMP TABLE retired_functional_roots ON COMMIT DROP AS
 SELECT instance_id,id FROM staffing.position_hierarchy WHERE purpose='functional';
-- +goose StatementBegin
DO $$
DECLARE part record;
BEGIN
 IF NOT EXISTS (SELECT FROM retired_functional_roots) THEN RETURN; END IF;
 FOR part IN SELECT * FROM (VALUES
   ('position_hierarchy_terminal','immutable_terminal','t.result->>''objectId''=r.id::text'),
   ('position_hierarchy_outbox','immutable_event','t.hierarchy_id=r.id'),
   ('position_hierarchy_edge','immutable_edge','t.hierarchy_id=r.id'),
   ('position_hierarchy_node','immutable_node','t.hierarchy_id=r.id'),
   ('position_hierarchy_version','immutable_version','t.hierarchy_id=r.id'),
   ('position_hierarchy','immutable_root','t.id=r.id')) AS parts(relation,guard,belongs)
 LOOP
  EXECUTE format('ALTER TABLE staffing.%I DISABLE TRIGGER %I',part.relation,part.guard);
  EXECUTE format('DELETE FROM staffing.%I t USING retired_functional_roots r WHERE t.instance_id=r.instance_id AND %s',part.relation,part.belongs);
  EXECUTE format('ALTER TABLE staffing.%I ENABLE TRIGGER %I',part.relation,part.guard);
 END LOOP;
END $$;
-- +goose StatementEnd
-- +goose Down
SELECT 1/0;
