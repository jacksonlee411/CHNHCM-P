package history

import "strings"

// Rank narrow, currently authorized matching facts once. Capture only the
// anchor, target and its two neighbors using the ordinary material/Go checks.
// Unused ordinary-page parameters remain typed for the shared query protocol.
var DepartmentNavigationSQL = func() string {
	// Ordinary pages limit before numbering. Navigation needs the full matching
	// order for its global position and neighbors, so stop before that window.
	scope, _, _ := strings.Cut(departmentQueryScopeSQL, "), ordered AS MATERIALIZED (")
	// Avoid storing three wide copies of the matching facts. Object/field
	// visibility still comes from the same guarded plan. With no explicit
	// field/status/tree filter there is no filter-field permission to check;
	// do not scan the entire visible set a second time just to count zero.
	scope = strings.Replace(scope, "scoped AS MATERIALIZED", "scoped AS NOT MATERIALIZED", 1)
	scope = strings.Replace(scope, "visible AS MATERIALIZED", "visible AS NOT MATERIALIZED", 1)
	scope = strings.Replace(scope, " SELECT v.* FROM visible v CROSS JOIN forbidden f", " SELECT v.department_id,v.q_path_date,v.path_ok,v.visible_code,v.visible_name FROM visible v CROSS JOIN forbidden f", 1)
	scope = strings.Replace(scope, "forbidden AS MATERIALIZED (", "filter_permissions AS MATERIALIZED (", 1)
	scope = strings.Replace(scope, "), matching AS MATERIALIZED (", `), forbidden AS MATERIALIZED (
 SELECT CASE WHEN coalesce(($5::jsonb)->>'scope','')='' AND ($5::jsonb)->>'field'='' AND ($5::jsonb)->>'status'='all'
 THEN 0::bigint ELSE (SELECT denied FROM filter_permissions) END AS denied
), matching AS MATERIALIZED (`, 1)
	scope += `), ordered AS MATERIALIZED (
 SELECT v.*,row_number() OVER(ORDER BY
   CASE ($5::jsonb)->>'sort' WHEN 'code' THEN visible_code WHEN 'name' THEN visible_name ELSE '' END COLLATE "C",
   CASE WHEN ($5::jsonb)->>'sort'='name_desc' THEN visible_name ELSE '' END COLLATE "C" DESC,department_id) AS ordinal
 FROM matching v
), navigation AS MATERIALIZED (
 SELECT ordinal + CASE ($5::jsonb)->>'direction' WHEN 'previous' THEN -1 WHEN 'next' THEN 1 ELSE 0 END AS target
 FROM ordered WHERE department_id=(($5::jsonb)->>'anchor')::uuid
), paged AS MATERIALIZED (
 SELECT o.* FROM ordered o WHERE (o.ordinal BETWEEN (SELECT target-1 FROM navigation) AND (SELECT target+1 FROM navigation)
 OR o.department_id=(($5::jsonb)->>'anchor')::uuid) AND $6::int>0 AND $7::bigint>=0 AND NOT $8::boolean
`
	tail := strings.Replace(departmentQueryTailSQL, "/*DEPARTMENT_ITEM*/", queryDepartmentMaterialSQL, 1)
	tail = strings.Replace(tail, "'enterpriseProfile',", `'navigation',(SELECT jsonb_build_object('targetId',t.department_id,
 'position',t.ordinal,'count',(SELECT count(*) FROM matching),
 'previousId',(SELECT department_id FROM ordered WHERE ordinal=t.ordinal-1),
 'nextId',(SELECT department_id FROM ordered WHERE ordinal=t.ordinal+1))
 FROM ordered t WHERE t.ordinal=(SELECT target FROM navigation)),
 'enterpriseProfile',`, 1)
	return departmentQueryFactsSQL + scope + tail
}()
