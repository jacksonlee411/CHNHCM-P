-- name: LoadQuotaSnapshot :one
SELECT s.snapshot FROM staffing.quota q JOIN staffing.quota_source s
 ON (s.instance_id,s.object_id)=(q.instance_id,q.id)
WHERE q.instance_id=$1 AND q.enterprise_id=$2 AND q.target_kind=sqlc.arg(target_kind) AND q.target_id=$3
 AND s.source_id=COALESCE(NULLIF(sqlc.arg(source_version)::text,'')::uuid,q.source_id);

-- name: QuotaIdentity :one
SELECT target_kind,target_id FROM staffing.quota WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3;

-- name: QuotaHistory :many
SELECT source_id,action,record_id,recorded_at,operator,snapshot FROM staffing.quota_source
WHERE instance_id=$1 AND object_id=$2 ORDER BY source_no DESC;

-- name: PositionQuotaPeriods :many
SELECT p.id,p.code,p.source_id,r.start_date::text AS start_day,COALESCE(r.end_date::text,'')::text AS end_day,r.active,
 COALESCE(r.fact->'data'->>'departmentId','')::text AS department_id, COALESCE(r.fact->'data'->>'name','')::text AS name
FROM staffing.position p JOIN staffing.position_read r ON (p.instance_id,p.id,p.source_id)=(r.instance_id,r.object_id,r.source_id)
WHERE p.instance_id=$1 AND p.enterprise_id=$2 AND p.id=$3 ORDER BY r.start_date;

-- name: PreviousQuotaSource :one
SELECT prev.source_id FROM staffing.quota_source src JOIN staffing.quota_source prev
 ON (prev.instance_id,prev.object_id,prev.source_no)=(src.instance_id,src.object_id,src.source_no-1)
WHERE src.instance_id=$1 AND src.object_id=$2 AND src.source_id=$3;

-- name: QuotaCandidates :many
SELECT p.id,p.code,r.active,r.start_date::text AS start_day,
 COALESCE(r.fact->'data'->>'departmentId','')::text AS department_id,COALESCE(r.fact->'data'->>'name','')::text AS name,
 COALESCE(n.record_id::text,'')::text AS record_id,COALESCE(n.amount,-1)::bigint AS amount
FROM staffing.position p
JOIN staffing.position_read r ON (p.instance_id,p.id,p.source_id)=(r.instance_id,r.object_id,r.source_id)
 AND r.valid_period @> sqlc.arg(day)::date
LEFT JOIN staffing.quota q ON (q.instance_id,q.enterprise_id,q.target_id)=(p.instance_id,p.enterprise_id,p.id) AND q.target_kind='position'
LEFT JOIN staffing.quota_record n ON (n.instance_id,n.object_id)=(q.instance_id,q.id) AND n.span @> sqlc.arg(day)::date
WHERE p.instance_id=$1 AND p.enterprise_id=$2
ORDER BY p.code,p.id;
