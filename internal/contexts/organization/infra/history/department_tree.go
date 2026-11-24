package history

import (
	"context"
	"encoding/json"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

// The existing Go-published narrow facts select a sibling window. Its path and
// child existence proofs are captured in the SAME statement as its labels.
var departmentTreeSQL = func() string {
	prefix := departmentQueryFactsSQL + `, candidates AS NOT MATERIALIZED (
 SELECT (segment).object_id AS department_id,(segment).q_code_key AS key,(segment).q_parent AS parent,(segment).q_bu AS bu,
 (segment).q_name_key AS name_key,(segment).q_path_date AS q_path_date,true AS path_ok
 FROM checked WHERE (segment).span @> $3::date AND (segment).q_state='active'
 AND organization.q131_set(($4::jsonb)->'allowed',(segment).q_bu,department_id)
 AND CASE ($5::jsonb)->>'mode' WHEN 'nodes' THEN ((segment).q_parent=nullif(($5::jsonb)->>'parent','')::uuid
 OR ((segment).q_parent IS NULL AND coalesce(($5::jsonb)->>'parent','')=''))
 WHEN 'location' THEN department_id=nullif(($5::jsonb)->>'around','')::uuid ELSE true END
 AND (($5::jsonb)->>'q'='' OR strpos((segment).q_code_key,($5::jsonb)->>'q')>0 OR strpos((segment).q_name_key,($5::jsonb)->>'q')>0)
 ), forbidden AS MATERIALIZED (
 SELECT CASE WHEN (($5::jsonb)->>'fieldsComplete')::boolean THEN 0::bigint ELSE
 (SELECT count(*) FILTER (WHERE EXISTS(SELECT 1 FROM unnest(ARRAY['code','name','status','effectiveDate','parentDepartmentId','departmentPath']) field
 WHERE NOT organization.q131_set(($4::jsonb)->'fields'->field,bu,department_id))) FROM candidates) END AS denied
 ), anchor AS (
 SELECT key,department_id FROM candidates WHERE department_id=nullif(($5::jsonb)->>'around','')::uuid
 ), position AS (
 SELECT coalesce((SELECT key FROM anchor),($5::jsonb)->>'afterKey') AS key,
 coalesce((SELECT department_id FROM anchor),nullif(($5::jsonb)->>'afterId','')::uuid) AS id
 ), sibling_window AS MATERIALIZED (
 SELECT c.* FROM candidates c CROSS JOIN position p WHERE p.id IS NULL OR
 CASE WHEN (($5::jsonb)->>'reverse')::boolean THEN (c.key COLLATE "C",c.department_id)<(p.key COLLATE "C",p.id)
 ELSE (c.key COLLATE "C",c.department_id)>(p.key COLLATE "C",p.id) OR c.department_id=(SELECT department_id FROM anchor) END
 ORDER BY CASE WHEN NOT (($5::jsonb)->>'reverse')::boolean THEN c.key END COLLATE "C",
 CASE WHEN (($5::jsonb)->>'reverse')::boolean THEN c.key END COLLATE "C" DESC,
 CASE WHEN NOT (($5::jsonb)->>'reverse')::boolean THEN c.department_id END,
 CASE WHEN (($5::jsonb)->>'reverse')::boolean THEN c.department_id END DESC LIMIT $6::int OFFSET $7::bigint
 ), paged AS MATERIALIZED (
 SELECT department_id,q_path_date,path_ok,row_number() OVER(ORDER BY key COLLATE "C",department_id) AS ordinal FROM sibling_window
 UNION ALL SELECT department_id,NULL::date,true,0 FROM checked WHERE department_id=nullif(($5::jsonb)->>'parent','')::uuid AND (segment).span @> $3::date
 )`
	tail := strings.Replace(departmentQueryTailSQL, "/*DEPARTMENT_ITEM*/", queryDepartmentMaterialSQL, 1)
	tail = strings.Replace(tail, "(SELECT count(*) FROM matching)", "0", 1)
	tail = strings.Replace(tail, "FROM paged),'[]'::jsonb)", "FROM paged WHERE ordinal>0),'[]'::jsonb)", 1)
	// One statement captures every page/path parent. The parent index lets
	// existence stop at the first qualifying child; a wide root must not
	// enumerate its complete child set merely to show an expansion arrow.
	children := `, tree_child_flags AS MATERIALIZED (
 SELECT parent.department_id AS parent,
 EXISTS(SELECT 1 FROM checked child WHERE (child.segment).q_parent=parent.department_id
 AND (child.segment).span @> $3::date AND (child.segment).q_state='active'
 AND organization.q131_set(($4::jsonb)->'allowed',(child.segment).q_bu,child.department_id)) AS has_children,
 (NOT coalesce((($5::jsonb)->>'childFieldsComplete')::boolean,false) AND EXISTS(SELECT 1 FROM checked child WHERE (child.segment).q_parent=parent.department_id
 AND (child.segment).span @> $3::date AND (child.segment).q_state='active'
 AND organization.q131_set(($4::jsonb)->'allowed',(child.segment).q_bu,child.department_id)
 AND (NOT organization.q131_set(($4::jsonb)->'fields'->'code',(child.segment).q_bu,child.department_id)
 OR NOT organization.q131_set(($4::jsonb)->'fields'->'name',(child.segment).q_bu,child.department_id)
 OR NOT organization.q131_set(($4::jsonb)->'fields'->'status',(child.segment).q_bu,child.department_id)
 OR NOT organization.q131_set(($4::jsonb)->'fields'->'effectiveDate',(child.segment).q_bu,child.department_id)
 OR NOT organization.q131_set(($4::jsonb)->'fields'->'parentDepartmentId',(child.segment).q_bu,child.department_id)
 OR NOT organization.q131_set(($4::jsonb)->'fields'->'departmentPath',(child.segment).q_bu,child.department_id)
 OR NOT organization.q131_set(($5::jsonb)->'reference'->'allowed',(child.segment).q_bu,child.department_id)
 OR NOT organization.q131_set(($5::jsonb)->'reference'->'fields'->'code',(child.segment).q_bu,child.department_id)
 OR NOT organization.q131_set(($5::jsonb)->'reference'->'fields'->'name',(child.segment).q_bu,child.department_id)))) AS denied
 FROM facts parent WHERE parent.day=$3::date
 )
`
	tail = strings.Replace(tail, "\nSELECT (SELECT count(DISTINCT department_id)", children+"SELECT (SELECT count(DISTINCT department_id)", 1)
	extra := `
 'treePrevious',EXISTS(SELECT 1 FROM candidates c WHERE (c.key COLLATE "C",c.department_id)<(SELECT w.key COLLATE "C",w.department_id FROM sibling_window w ORDER BY w.key COLLATE "C",w.department_id LIMIT 1)),
 'treeNext',EXISTS(SELECT 1 FROM candidates c WHERE (c.key COLLATE "C",c.department_id)>(SELECT w.key COLLATE "C",w.department_id FROM sibling_window w ORDER BY w.key COLLATE "C" DESC,w.department_id DESC LIMIT 1)),
 'treeChildren',coalesce((SELECT jsonb_object_agg(f.department_id::text, jsonb_build_object(
 'HasChildren',coalesce(c.has_children,false),'Denied',coalesce(c.denied,false)))
 FROM facts f LEFT JOIN tree_child_flags c ON c.parent=f.department_id WHERE f.day=$3::date),'{}'::jsonb),`
	tail = strings.Replace(tail, "'enterpriseProfile',", extra+"'enterpriseProfile',", 1)
	return strings.TrimSuffix(prefix, ")") + tail
}()

// This proves only syntactic set inclusion, never membership through a BU's
// current departments. A false result keeps the per-object SQL checks.
func treeSetCovers(cover, wanted access.ReadSet) bool {
	if cover.All {
		return true
	}
	if wanted.All {
		return false
	}
	contains := func(have, need []string) bool {
		set := make(map[string]struct{}, len(have))
		for _, id := range have {
			set[id] = struct{}{}
		}
		for _, id := range need {
			if _, ok := set[id]; !ok {
				return false
			}
		}
		return true
	}
	return contains(cover.BUs, wanted.BUs) && contains(cover.Departments, wanted.Departments)
}

func captureTreeQuery(ctx context.Context, tx pgx.Tx, guard access.Guard, q app.QueryRequest) (app.QueryCapture, error) {
	p, t := q.Plan, q.Tree
	planner, ok := guard.(access.ReadPlanner)
	if !ok || t.Reference.Origin != planner || t.Reference.Action != access.DepartmentReference || t.Reference.Subject != p.Subject || t.Reference.Session != p.Session || t.Reference.Day != p.Day || t.Reference.Enterprise != p.Enterprise {
		return app.QueryCapture{}, queryUnavailable()
	}
	fields := map[string]access.ReadSet{}
	fieldsComplete := true
	for _, name := range []string{"code", "name", "status", "effectiveDate", "parentDepartmentId", "departmentPath"} {
		fields[name] = p.Fields[name]
		fieldsComplete = fieldsComplete && treeSetCovers(p.Fields[name], p.Allowed)
	}
	reference := queryPlan(access.ReadPlan{Allowed: t.Reference.Allowed, Fields: map[string]access.ReadSet{"code": t.Reference.Fields["code"], "name": t.Reference.Fields["name"]}})
	childFieldsComplete := fieldsComplete && treeSetCovers(t.Reference.Allowed, p.Allowed) && treeSetCovers(t.Reference.Fields["code"], p.Allowed) && treeSetCovers(t.Reference.Fields["name"], p.Allowed)
	input, _ := json.Marshal(map[string]any{"mode": t.Mode, "parent": t.ParentID, "around": t.AroundID, "q": q.Input.Q, "afterKey": t.AfterKey, "afterId": t.AfterID, "reverse": t.Reverse, "reference": reference, "fieldsComplete": fieldsComplete, "childFieldsComplete": childFieldsComplete})
	plan, _ := json.Marshal(queryPlan(access.ReadPlan{Allowed: p.Allowed, Fields: fields}))
	var objects, segments, bad, dictionaries, shapes, denied, count int64
	var raw []byte
	done := foundation.BeginStage(ctx, "department.tree.capture")
	// Root, sibling and location predicates have very different selectivity.
	// Reuse parameter descriptions without retaining a generic server plan,
	// which otherwise scans the enterprise even for one located department.
	err := tx.QueryRow(ctx, departmentTreeSQL, pgx.QueryExecModeCacheDescribe, p.Instance, p.Enterprise, p.Day, plan, input, q.Input.PageSize, 0).Scan(&objects, &segments, &bad, &dictionaries, &shapes, &denied, &count, &raw)
	done()
	if err != nil {
		return app.QueryCapture{}, err
	}
	if bad != 0 || raw == nil {
		return app.QueryCapture{}, queryUnavailable()
	}
	if denied != 0 {
		if t.Mode == "location" {
			return app.QueryCapture{}, foundation.Fail(409, "M02_TREE_SCOPE_UNAVAILABLE", "当前范围不可用")
		}
		return app.QueryCapture{}, foundation.Fail(403, "ACCESS_DENIED", "无部门树必要字段读取权限")
	}
	out, err := decodeDepartmentQuery(ctx, p.Enterprise, p.Day, 0, raw, nil)
	if err != nil {
		return out, err
	}
	out.Stats.Objects, out.Stats.Segments = objects, segments
	out.Stats.Dictionaries, out.Stats.Shapes = dictionaries, shapes
	return out, nil
}
