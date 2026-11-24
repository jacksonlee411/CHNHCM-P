-- name: LoadFamilyHead :one
SELECT head FROM staffing.job_family WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3;
-- name: ListClassificationReferenceFacts :many
SELECT h.id::text AS object_id,h.code,h.source_id::text AS source_id,
 r.record_id::text AS record_id,r.start_date::text AS start_date,
 COALESCE(r.end_date::text,'')::text AS end_date,
 COALESCE(r.fact->>'type','')::text AS operation_type,
 COALESCE(r.fact->'data'->>'name','')::text AS name
FROM staffing.job_family h
JOIN staffing.job_family_read r ON (r.instance_id,r.object_id,r.source_id)=(h.instance_id,h.id,h.source_id)
WHERE sqlc.arg(kind)::text='jobfamily' AND h.instance_id=sqlc.arg(instance_id) AND h.enterprise_id=sqlc.arg(enterprise_id)
UNION ALL
SELECT h.id::text,h.code,h.source_id::text,r.record_id::text,r.start_date::text,
 COALESCE(r.end_date::text,'')::text,COALESCE(r.fact->>'type','')::text,COALESCE(r.fact->'data'->>'name','')::text
FROM staffing.job_function h
JOIN staffing.job_function_read r ON (r.instance_id,r.object_id,r.source_id)=(h.instance_id,h.id,h.source_id)
WHERE sqlc.arg(kind)::text='jobfunction' AND h.instance_id=sqlc.arg(instance_id) AND h.enterprise_id=sqlc.arg(enterprise_id)
ORDER BY code,object_id,start_date;
-- name: LoadFamilySource :one
SELECT snapshot FROM staffing.job_family_source WHERE instance_id=$1 AND object_id=$2 AND source_id=$3;
-- name: LoadJobHead :one
SELECT head FROM staffing.job WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3;
-- name: LoadJobSource :one
SELECT snapshot FROM staffing.job_source WHERE instance_id=$1 AND object_id=$2 AND source_id=$3;
-- name: LoadGradeHead :one
SELECT head FROM staffing.grade WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3;
-- name: LoadGradeSource :one
SELECT snapshot FROM staffing.grade_source WHERE instance_id=$1 AND object_id=$2 AND source_id=$3;

-- name: LoadGradeLadderHead :one
SELECT head FROM staffing.grade_ladder WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3;
-- name: LoadGradeLadderSource :one
SELECT snapshot FROM staffing.grade_ladder_source WHERE instance_id=$1 AND object_id=$2 AND source_id=$3;

-- name: ListGradeReferenceFacts :many
SELECT h.id::text AS grade_id,h.code,r.start_date::text AS start_date,
 COALESCE(r.end_date::text,'')::text AS end_date,
 COALESCE(r.fact->>'type','')::text AS operation_type,
 COALESCE(r.fact->'data'->>'name','')::text AS name
FROM staffing.grade h
JOIN staffing.grade_read r ON (r.instance_id,r.object_id,r.source_id)=(h.instance_id,h.id,h.source_id)
WHERE h.instance_id=$1 AND h.enterprise_id=$2
ORDER BY h.code,h.id,r.start_date;

-- name: LoadPositionHead :one
SELECT head FROM staffing.position WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3;
-- name: LoadPositionSource :one
SELECT snapshot FROM staffing.position_source WHERE instance_id=$1 AND object_id=$2 AND source_id=$3;

-- name: LoadJobFunctionHead :one
SELECT head FROM staffing.job_function WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3;
-- name: LoadJobFunctionSource :one
SELECT snapshot FROM staffing.job_function_source WHERE instance_id=$1 AND object_id=$2 AND source_id=$3;

-- name: LegacyJobFunctionEvidence :many
SELECT object_id::text AS job,source_id::text AS source,record_id::text AS record,revision::text AS revision
FROM staffing.job_function_legacy
WHERE instance_id=sqlc.arg(instance)::uuid AND object_id=ANY(sqlc.arg(objects)::uuid[])
 AND source_id=ANY(sqlc.arg(sources)::uuid[]);
