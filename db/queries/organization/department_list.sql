-- name: ReadDepartmentCreationBUs :many
SELECT bu_id::text FROM organization.business_unit WHERE instance_id=$1 ORDER BY bu_id;


-- name: ReadDepartmentListHeads :many
SELECT f.kind::smallint AS kind,f.identity::bytea AS identity FROM organization.list_source_heads($1::uuid) f;

-- name: ReadDepartmentParentHeads :many
SELECT 1::smallint AS kind,(uuid_send(d.department_id)||uuid_send(d.head_source_id)||uuid_send(h.head_source_id))::bytea AS identity,d.management_bus::text[] AS bus,d.parents::text[] AS parents
FROM organization.department d LEFT JOIN organization.department_hierarchy_relation h USING(instance_id,enterprise_id,department_id) WHERE d.instance_id=$1
UNION ALL SELECT 2::smallint,(uuid_send(b.bu_id)||uuid_send(b.head_source_id))::bytea,ARRAY[]::text[],ARRAY[]::text[] FROM organization.business_unit b WHERE b.instance_id=$1;
