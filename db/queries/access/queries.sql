-- name: FindLogin :one
SELECT p.principal_id,p.login,p.state,c.iterations,c.salt,c.password_hash
FROM access.principal p JOIN access.credential c USING(instance_id,principal_id)
WHERE p.instance_id=$1 AND p.normalized_login=$2;

-- name: LockPolicyShared :one
SELECT policy_no FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR SHARE;

-- name: LockPolicyExclusive :one
SELECT policy_no FROM access.policy_guard WHERE instance_id=$1 AND principal_id=$2 FOR UPDATE;

-- name: ReadGrants :many
SELECT scope_kind,scope_id,action,read_fields,write_fields FROM access.subject_grant
WHERE instance_id=$1 AND principal_id=$2 ORDER BY grant_id;

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
