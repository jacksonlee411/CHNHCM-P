-- name: GetUnit :one
SELECT bu_id,enterprise_id,code,first_record_id,head_source_id FROM organization.business_unit WHERE instance_id=$1 AND bu_id=$2;
