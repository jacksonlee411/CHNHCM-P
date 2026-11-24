, matching AS MATERIALIZED (
 SELECT c.* FROM checked c CROSS JOIN integrity i
 WHERE i.bad=0 AND (segment).span @> $3::date
   AND ($4::boolean OR (segment).q_bu=ANY($5::uuid[]))
   AND strpos((segment).q_name_key,$6::text)>0
   AND ($7::text='' OR (segment).q_state=$7)
), paged AS MATERIALIZED (
 SELECT * FROM matching ORDER BY code,department_id LIMIT $8 OFFSET $9
)
-- One unconditional result row forces consumption of the integrity aggregate,
-- even for no matches, an empty authorized scope, or an out-of-range page.
SELECT i.objects,i.segments,i.bad,i.dictionaries,i.shapes,
 CASE WHEN i.bad=0 THEN (SELECT count(*) FROM matching) END AS total,
 CASE WHEN i.bad=0 THEN (
  SELECT coalesce(jsonb_agg(item ORDER BY code,department_id),'[]'::jsonb)
  FROM (
   SELECT code,department_id,/*DEPARTMENT_ITEM*/ AS item
   FROM paged
   JOIN organization.department_dictionary d ON (d.instance_id,d.enterprise_id,d.id)=($1::uuid,$2::uuid,(segment).dictionary_id)
   JOIN organization.department_shape h ON (h.instance_id,h.enterprise_id,h.id)=($1::uuid,$2::uuid,(segment).shape_id)
  ) page
 ) END AS items
FROM integrity i
