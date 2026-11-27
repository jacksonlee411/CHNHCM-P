-- name: FindLogin :one
SELECT p.principal_id,p.login,p.state,c.iterations,c.salt,c.password_hash
FROM access.principal p JOIN access.credential c USING(instance_id,principal_id)
WHERE p.instance_id=$1 AND p.normalized_login=$2;

-- name: LockPolicyShared :one
SELECT policy_no FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR SHARE;

-- name: LockPolicyExclusive :one
SELECT policy_no FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE;

-- name: ReadGrants :many
-- The held subject row fixes the dependency set. Classify using the actual
-- version rows returned by FOR SHARE, including updates committed during a wait.
WITH bindings AS MATERIALIZED (
 SELECT a.instance_id,a.grant_id,a.purpose,a.valid_from,a.valid_until,b.role_id,b.scope_id,b.fields_id
 FROM access.grant_assignment a JOIN access.grant_revision b USING(instance_id,grant_id,revision)
 WHERE a.instance_id=$1 AND a.principal_id=$2 AND b.operation<>'revoke'
), locked AS MATERIALIZED (
 SELECT v.kind,v.version_id,v.actions,v.scope_kind,v.scope_id,v.read_fields,v.write_fields,v.disabled_at
 FROM access.policy_version v WHERE v.instance_id=$1 AND (v.kind,v.version_id) IN (
 SELECT 'role',role_id FROM bindings UNION ALL SELECT 'scope',scope_id FROM bindings
 UNION ALL SELECT 'fields',fields_id FROM bindings
 ) ORDER BY v.kind COLLATE "C",v.version_id FOR SHARE OF v
)
SELECT s.scope_kind::text AS scope_kind,s.scope_id::text AS scope_id,op.action::text AS action,
 f.read_fields,f.write_fields,b.purpose,b.valid_from,b.valid_until
FROM bindings b
JOIN locked r ON r.kind='role' AND r.version_id=b.role_id AND r.disabled_at IS NULL
JOIN locked s ON s.kind='scope' AND s.version_id=b.scope_id AND s.disabled_at IS NULL
JOIN locked f ON f.kind='fields' AND f.version_id=b.fields_id AND f.disabled_at IS NULL
CROSS JOIN LATERAL unnest(r.actions) op(action)
ORDER BY b.grant_id,op.action;

-- name: FindSessionSubject :one
SELECT principal_id FROM access.session WHERE instance_id=$1 AND token_digest=$2;

-- name: ReadSession :one
SELECT s.session_id,s.principal_id,p.login,p.state,s.csrf_value,s.idle_expires_at,s.absolute_expires_at,s.revoked_at
FROM access.session s JOIN access.principal p USING(instance_id,principal_id)
WHERE s.instance_id=$1 AND s.token_digest=$2;

-- name: TouchSession :exec
UPDATE access.session SET last_seen_at=GREATEST(last_seen_at,sqlc.arg(accessed_at)::timestamptz),idle_expires_at=GREATEST(idle_expires_at,LEAST(sqlc.arg(accessed_at)::timestamptz+interval '30 minutes',absolute_expires_at))
WHERE instance_id=$1 AND session_id=$2;

-- name: EndSession :exec
UPDATE access.session SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND session_id=$2 AND revoked_at IS NULL;
