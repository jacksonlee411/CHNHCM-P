-- name: LockDepartmentBUShared :many
SELECT bu_id FROM organization.business_unit WHERE instance_id=$1 AND bu_id=ANY($2::uuid[]) ORDER BY bu_id FOR SHARE;


-- name: LockDepartmentBUWrite :many
SELECT bu_id FROM organization.business_unit WHERE instance_id=$1 AND bu_id=ANY($2::uuid[]) ORDER BY bu_id FOR UPDATE;


-- name: LockDepartmentHeadWrite :one
SELECT head_source_id FROM organization.department WHERE instance_id=$1 AND department_id=$2 FOR UPDATE;


-- name: LockDepartmentRelationWrite :one
SELECT head_source_id FROM organization.department_hierarchy_relation WHERE instance_id=$1 AND department_id=$2 FOR UPDATE;


-- name: LockDepartmentDependenciesShared :many
SELECT d.department_id FROM organization.department d
JOIN organization.department_hierarchy_relation h USING(instance_id,enterprise_id,department_id)
WHERE d.instance_id=$1 AND d.department_id=ANY($2::uuid[])
ORDER BY d.department_id FOR SHARE OF d,h;
