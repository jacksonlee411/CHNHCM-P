-- Query only the facts needed by this read. Go's transactional writer and
-- storage constraints own index consistency; this is not an enterprise audit.
WITH RECURSIVE checked AS NOT MATERIALIZED (
 SELECT d.instance_id,d.enterprise_id,d.department_id,d.code,d.first_record_id,
        d.head_source_id,h.head_source_id AS relation_head,s AS segment
 FROM organization.department d
 LEFT JOIN organization.department_hierarchy_relation h
  ON (h.instance_id,h.enterprise_id,h.department_id)=(d.instance_id,d.enterprise_id,d.department_id)
 LEFT JOIN organization.department_read_segment s
  ON (s.instance_id,s.object_id)=(d.instance_id,d.department_id)
 WHERE d.instance_id=$1 AND d.enterprise_id=$2
)
, tree_scope_root AS MATERIALIZED (
 SELECT department_id,(segment).q_parent AS parent,(segment).q_bu AS bu,(segment).q_path_date AS path_day,code FROM checked
 WHERE department_id=nullif(($5::jsonb)->>'scope','')::uuid
 AND (segment).span @> $3::date AND (segment).q_state='active'
), tree_scope_current AS MATERIALIZED (
 -- Group the current adjacency once; each level visits parent groups rather
 -- than rescanning every leaf. Parallel arrays are aggregated from the same
 -- input order and unnested together, retaining each identity/code/path-day.
 SELECT (segment).q_parent AS parent,array_agg(department_id) AS children,array_agg((segment).q_path_date) AS path_days,array_agg(code) AS codes FROM checked
 WHERE coalesce(($5::jsonb)->>'scope','')<>'' AND (($5::jsonb)->>'descendants')::boolean
 AND NOT (($5::jsonb)->>'scopeSparse')::boolean AND (segment).span @> $3::date AND (segment).q_state='active' AND (segment).q_parent IS NOT NULL
 GROUP BY (segment).q_parent
), tree_scope_down(id,path_day,code) AS (
 SELECT department_id,path_day,code FROM tree_scope_root WHERE NOT (($5::jsonb)->>'scopeSparse')::boolean
 UNION
 SELECT unnest(c.children),unnest(c.path_days),unnest(c.codes) FROM tree_scope_current c JOIN tree_scope_down p ON c.parent=p.id
), tree_scope_up(root,id,parent,path_day,code) AS (
 -- Department-only rights need the candidates' ancestor chains, including
 -- unreadable intermediates, rather than all siblings below the scope.
 SELECT department_id,department_id,(segment).q_parent,(segment).q_path_date,code FROM checked
 WHERE (($5::jsonb)->>'scopeSparse')::boolean AND (segment).span @> $3::date AND (segment).q_state='active'
 AND department_id IN (SELECT jsonb_array_elements_text(($5::jsonb)->'scopeCandidates')::uuid)
 UNION
 SELECT w.root,c.department_id,(c.segment).q_parent,w.path_day,w.code
 FROM tree_scope_up w JOIN checked c ON c.department_id=w.parent
 AND (c.segment).span @> $3::date AND (c.segment).q_state='active'
 WHERE w.id<>nullif(($5::jsonb)->>'scope','')::uuid AND (($5::jsonb)->>'descendants')::boolean
), tree_scope_branch(id,path_day,code) AS NOT MATERIALIZED (
 SELECT id,path_day,code FROM tree_scope_down UNION ALL
 SELECT root,path_day,code FROM tree_scope_up WHERE id=nullif(($5::jsonb)->>'scope','')::uuid
), tree_scope_required(id) AS (
 (SELECT department_id FROM tree_scope_root WHERE (($5::jsonb)->>'scopeReferencesComplete')::boolean
 UNION
 SELECT b.id FROM tree_scope_branch b JOIN checked c ON c.department_id=b.id
 AND (c.segment).span @> $3::date AND (c.segment).q_state='active'
 WHERE NOT (($5::jsonb)->>'scopeReferencesComplete')::boolean
 AND organization.q131_set(($4::jsonb)->'allowed',(c.segment).q_bu,b.id))
 UNION
 SELECT (c.segment).q_parent FROM tree_scope_required p JOIN checked c ON c.department_id=p.id
 AND (c.segment).span @> $3::date AND (c.segment).q_state='active'
 WHERE (c.segment).q_parent IS NOT NULL
), tree_scope_problem AS (
 SELECT CASE WHEN coalesce(($5::jsonb)->>'scope','')='' THEN 0
 WHEN NOT EXISTS(SELECT 1 FROM tree_scope_root c
 WHERE organization.q131_set(($4::jsonb)->'allowed',c.bu,c.department_id)
 AND NOT EXISTS(SELECT 1 FROM unnest(ARRAY['code','name','status','effectiveDate','parentDepartmentId','departmentPath']) field
 WHERE NOT organization.q131_set(($4::jsonb)->'fields'->field,c.bu,c.department_id))) THEN -1
 WHEN EXISTS(SELECT 1 FROM tree_scope_required r LEFT JOIN checked c ON c.department_id=r.id
 AND (c.segment).span @> $3::date AND (c.segment).q_state='active' WHERE c.department_id IS NULL) THEN -2
 WHEN EXISTS(SELECT 1 FROM tree_scope_required r JOIN checked c ON c.department_id=r.id
 AND (c.segment).span @> $3::date AND (c.segment).q_state='active' WHERE
 NOT organization.q131_set(($5::jsonb)->'scopeReference'->'allowed',(c.segment).q_bu,c.department_id)
 OR NOT organization.q131_set(($5::jsonb)->'scopeReference'->'fields'->'code',(c.segment).q_bu,c.department_id)
 OR NOT organization.q131_set(($5::jsonb)->'scopeReference'->'fields'->'name',(c.segment).q_bu,c.department_id)) THEN 1
 ELSE 0 END AS problem
)
