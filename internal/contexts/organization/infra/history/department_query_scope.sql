, scoped AS NOT MATERIALIZED (
 SELECT department_id,code,(segment).q_name,(segment).q_name_key,(segment).q_short_key,
  (segment).q_bu,(segment).q_date,(segment).q_state,(segment).q_code_key,(segment).q_path_date,
   organization.q131_set(($4::jsonb)->'allowed',(segment).q_bu,department_id) AS allowed,
   organization.q131_set(($4::jsonb)->'fields'->'code',(segment).q_bu,department_id) AS code_ok,
   organization.q131_set(($4::jsonb)->'fields'->'name',(segment).q_bu,department_id) AS name_ok,
   organization.q131_set(($4::jsonb)->'fields'->'departmentPath',(segment).q_bu,department_id)
    AND organization.q131_set(($4::jsonb)->'fields'->'effectiveDate',(segment).q_bu,department_id)
    AND organization.q131_set(($4::jsonb)->'fields'->'status',(segment).q_bu,department_id) AS path_ok
 FROM checked c
 WHERE (segment).span @> $3::date
 AND (NOT coalesce((($5::jsonb)->>'scopeSparse')::boolean,false) OR c.department_id=ANY(ARRAY(SELECT jsonb_array_elements_text(($5::jsonb)->'scopeCandidates')::uuid)))
 AND (coalesce(($5::jsonb)->>'scope','')='' OR c.department_id IN (SELECT id FROM tree_scope_branch))
 AND (NOT ($5::jsonb ? 'ids') OR c.department_id IN (SELECT jsonb_array_elements_text(($5::jsonb)->'ids')::uuid))
), filterable AS NOT MATERIALIZED (
 SELECT v.*,CASE WHEN code_ok AND ($5::jsonb)->>'sort'='code' THEN code ELSE '' END AS visible_code,
  CASE WHEN name_ok AND ($5::jsonb)->>'sort'<>'code' THEN q_name ELSE '' END AS visible_name,
  CASE ($5::jsonb)->>'field'
   WHEN 'code' THEN q_code_key WHEN 'name' THEN q_name_key
   WHEN 'shortName' THEN coalesce(q_short_key,'')
   WHEN 'managementBuId' THEN q_bu::text
   WHEN 'effectiveDate' THEN q_date::text WHEN 'status' THEN q_state
   ELSE '' END AS filter_value
 FROM scoped v WHERE allowed
), visible AS NOT MATERIALIZED (
 -- Keep the selected sort key and evaluated match flag. The independent
 -- permission predicate below still considers every visible object before
 -- matching filters apply; no second materialized copy is required.
 SELECT department_id,q_bu,q_path_date,path_ok,visible_code,visible_name,
 ((($5::jsonb)->>'q'='' OR code_ok AND strpos(q_code_key,($5::jsonb)->>'q')>0 OR name_ok AND strpos(q_name_key,($5::jsonb)->>'q')>0)
 AND (($5::jsonb)->>'status'='all' OR q_state=($5::jsonb)->>'status')
 AND (($5::jsonb)->>'field' IN ('','departmentPath') OR CASE ($5::jsonb)->>'operator'
  WHEN 'contains' THEN strpos(filter_value,($5::jsonb)->>'value')>0
  WHEN 'equals' THEN filter_value=($5::jsonb)->>'value' WHEN 'is' THEN filter_value=($5::jsonb)->>'value'
  WHEN 'isAnyOf' THEN filter_value=ANY(string_to_array(($5::jsonb)->>'value',','))
  WHEN 'before' THEN filter_value<($5::jsonb)->>'value' WHEN 'after' THEN filter_value>($5::jsonb)->>'value'
  WHEN 'onOrBefore' THEN filter_value<=($5::jsonb)->>'value' WHEN 'onOrAfter' THEN filter_value>=($5::jsonb)->>'value'
  ELSE false END)) AS matches
 FROM filterable v
), forbidden AS MATERIALIZED (
 -- Only the existence of a denied object is relevant. Constant-false field
 -- checks need not consume the whole visible set merely to count zero.
 SELECT CASE WHEN (SELECT problem FROM tree_scope_problem)<>0 THEN (SELECT problem FROM tree_scope_problem) WHEN EXISTS(SELECT 1 FROM visible WHERE
 (coalesce(($5::jsonb)->>'scope','')<>'' AND NOT coalesce((($5::jsonb)->>'scopeFieldsComplete')::boolean,false) AND EXISTS(SELECT 1 FROM unnest(ARRAY['code','name','status','effectiveDate','parentDepartmentId','departmentPath']) field WHERE NOT organization.q131_set(($4::jsonb)->'fields'->field,q_bu,department_id))) OR
   (($5::jsonb)->>'field'<>'' AND NOT organization.q131_set(($4::jsonb)->'fields'->(($5::jsonb)->>'field'),q_bu,department_id))
   OR (($5::jsonb)->>'status'<>'all' AND NOT organization.q131_set(($4::jsonb)->'fields'->'status',q_bu,department_id))) THEN 1 ELSE 0 END AS denied
), matching AS MATERIALIZED (
 SELECT v.department_id,v.q_path_date,v.path_ok,v.visible_code,v.visible_name
 FROM visible v CROSS JOIN forbidden f
 WHERE f.denied=0 AND v.matches AND NOT coalesce((($5::jsonb)->>'scopeMatchAll')::boolean,false)
 -- This branch is admitted only by the original complete Go read plan and
 -- normalized query. Reuse the same statement's branch facts; do not reread
 -- every current segment merely to prove that an absent scalar filter matches.
 UNION ALL
 SELECT b.id,b.path_day,true,b.code,'' FROM tree_scope_branch b
 CROSS JOIN forbidden f
 WHERE f.denied=0 AND coalesce((($5::jsonb)->>'scopeMatchAll')::boolean,false)
 AND (NOT ($5::jsonb ? 'ids') OR b.id IN (SELECT jsonb_array_elements_text(($5::jsonb)->'ids')::uuid))
), ordered AS MATERIALIZED (
 -- Select the requested window before numbering it. Count and permission
 -- checks still consume the complete matching/visible sets in this statement.
 SELECT v.*
 FROM matching v
 ORDER BY
   CASE ($5::jsonb)->>'sort' WHEN 'code' THEN visible_code WHEN 'name' THEN visible_name ELSE '' END COLLATE "C",
   CASE WHEN ($5::jsonb)->>'sort'='name_desc' THEN visible_name ELSE '' END COLLATE "C" DESC,department_id
 LIMIT CASE WHEN $8::boolean OR (($5::jsonb)->>'field')='departmentPath' THEN NULL ELSE $6::int END
 OFFSET CASE WHEN $8::boolean OR (($5::jsonb)->>'field')='departmentPath' THEN 0 ELSE $7::bigint END
), paged AS MATERIALIZED (
 SELECT department_id,q_path_date,path_ok,row_number() OVER(ORDER BY
   CASE ($5::jsonb)->>'sort' WHEN 'code' THEN visible_code WHEN 'name' THEN visible_name ELSE '' END COLLATE "C",
   CASE WHEN ($5::jsonb)->>'sort'='name_desc' THEN visible_name ELSE '' END COLLATE "C" DESC,department_id) AS ordinal
 FROM ordered
