-- name: OccupyNumber :execrows
INSERT INTO personnel.number(instance_id,number,person_id)
VALUES(sqlc.arg(instance_id),sqlc.arg(number),sqlc.arg(person_id))
ON CONFLICT DO NOTHING;

-- name: DropSessionQueries :exec
DELETE FROM personnel.query WHERE instance_id=$1 AND session_id=$2;

-- name: ExpireQueries :exec
DELETE FROM personnel.query WHERE expires_at<=clock_timestamp();
