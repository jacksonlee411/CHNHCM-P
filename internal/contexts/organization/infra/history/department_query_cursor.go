package history

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"strconv"
	"strings"
)

// The sort contains only identities and visible keys. Each scalar subplan binds
// one root's current facts, path-day ancestors and dictionaries in this statement.
var DepartmentQueryCursorSQL = func() string {
	scope, _, _ := strings.Cut(departmentQueryScopeSQL, "), ordered AS MATERIALIZED (")
	tail := strings.Replace(departmentQueryTailSQL, "/*DEPARTMENT_ITEM*/", queryDepartmentMaterialSQL, 1)
	tail = strings.Replace(tail, "'enterpriseProfile',"+queryEnterpriseMaterialSQL+",", "", 1)
	start := strings.Index(tail, "\nSELECT (SELECT count(DISTINCT")
	payload := strings.Index(tail[start:], "jsonb_build_object(") + start
	tail = tail[:start] + "\nSELECT " + tail[payload:]
	return departmentQueryFactsSQL + scope + `), anchor AS MATERIALIZED (
 SELECT department_id AS id,CASE WHEN ($5::jsonb)->>'sort'='code' THEN visible_code ELSE visible_name END AS key,q_path_date AS path_day,path_ok FROM matching WHERE department_id=nullif(($5::jsonb)->>'anchor','')::uuid
 ), boundary AS (
 SELECT coalesce((SELECT key FROM anchor),($5::jsonb)->>'afterKey') AS key,coalesce((SELECT id FROM anchor),nullif(($5::jsonb)->>'afterId','')::uuid) AS id
 ), roots AS (
 SELECT 0 AS kind,coalesce((SELECT key FROM anchor),'') AS key,(SELECT id FROM anchor) AS id,(SELECT path_day FROM anchor) AS path_day,coalesce((SELECT path_ok FROM anchor),false) AS path_ok
 UNION ALL SELECT 1,CASE WHEN ($5::jsonb)->>'sort'='code' THEN visible_code ELSE visible_name END,department_id,q_path_date,path_ok FROM matching
 ), positioned AS (
 SELECT r.* FROM roots r CROSS JOIN boundary b WHERE kind=0 OR (($5::jsonb)->>'anchor'='' AND b.id IS NULL) OR
 CASE WHEN (($5::jsonb)->>'sort'='name_desc') <> (($5::jsonb)->>'reverse')::boolean THEN r.key COLLATE "C" < b.key COLLATE "C"
 ELSE r.key COLLATE "C" > b.key COLLATE "C" END
 OR r.key COLLATE "C"=b.key COLLATE "C" AND CASE WHEN (($5::jsonb)->>'reverse')::boolean THEN r.id<b.id ELSE r.id>b.id END
 )
 SELECT r.kind,r.key,coalesce(r.id::text,''),(SELECT denied FROM forbidden), CASE WHEN r.id IS NULL THEN NULL ELSE (
 WITH RECURSIVE paged AS MATERIALIZED (
 SELECT r.id AS department_id,r.path_day AS q_path_date,r.path_ok,1 AS ordinal
 ` + tail + `) END AS material, CASE WHEN r.kind=0 THEN ` + queryEnterpriseMaterialSQL + ` ELSE NULL END AS enterprise FROM positioned r ORDER BY r.kind,
 CASE WHEN (($5::jsonb)->>'sort'='name_desc') = (($5::jsonb)->>'reverse')::boolean THEN r.key ELSE '' END COLLATE "C",
 CASE WHEN (($5::jsonb)->>'sort'='name_desc') <> (($5::jsonb)->>'reverse')::boolean THEN r.key ELSE '' END COLLATE "C" DESC,
 CASE WHEN NOT (($5::jsonb)->>'reverse')::boolean THEN r.id END,
 CASE WHEN (($5::jsonb)->>'reverse')::boolean THEN r.id END DESC`

}()

func WalkDepartmentQuery(ctx context.Context, tx pgx.Tx, guard access.Guard, q app.QueryRequest, visit func(app.QueryFrame) (bool, error)) (err error) {
	planner, ok := guard.(access.ReadPlanner)
	p := q.Plan
	principal := guard.Principal()
	if !ok || p.Origin != planner || p.Instance != principal.InstanceID || p.Subject != principal.ID || p.Session != principal.SessionID || p.Action != access.DepartmentRead || !foundation.ValidID(p.Enterprise) || p.At.IsZero() || p.Day != p.At.In(foundation.Shanghai).Format("2006-01-02") {
		return queryUnavailable()
	}
	q.Input, err = app.NormalizeQuery(q.Input)
	if err != nil {
		return
	}
	field, operator, value := "", "", ""
	if f := q.Input.Filter; f != nil {
		field, operator, value = f.Field, f.Operator, f.Value
	}
	parameters := map[string]any{"q": q.Input.Q, "sort": q.Input.Sort, "status": q.Input.Status, "field": field, "operator": operator, "value": value, "afterKey": q.AfterKey, "afterId": q.AfterID, "anchor": q.AnchorID, "reverse": q.Reverse}
	if e := addTreeScopeParameters(ctx, guard, q, parameters); e != nil {
		return e
	}
	if q.OnlyIDs != nil {
		parameters["ids"] = q.OnlyIDs
	}
	input, _ := json.Marshal(parameters)
	plan, _ := json.Marshal(queryPlan(p))
	if _, err = tx.Exec(ctx, "SET LOCAL plan_cache_mode='force_custom_plan'"); err != nil {
		return
	}
	if _, err = tx.Exec(ctx, "DECLARE query_facts133 NO SCROLL CURSOR FOR "+DepartmentQueryCursorSQL, p.Instance, p.Enterprise, p.Day, plan, input); err != nil {
		return
	}
	defer func() {
		if err != nil {
			return
		} // Rollback closes an errored/canceled cursor.
		_, e := tx.Exec(ctx, "CLOSE query_facts133")
		if err == nil {
			err = e
		}
	}()
	header := false
	// The first window needs its header, page and one lookahead. Large or
	// selective walks continue in full batches from the same open snapshot.
	batchSize := min(64, q.Input.PageSize+2)
	var enterpriseName string
	for {
		rows, e := tx.Query(ctx, "FETCH FORWARD "+strconv.Itoa(batchSize)+" FROM query_facts133")
		if e != nil {
			return e
		}
		type frame struct {
			kind            int
			key, id         string
			denied          int64
			raw, enterprise []byte
		}
		batch := make([]frame, 0, batchSize)
		for rows.Next() {
			var f frame
			if e = rows.Scan(&f.kind, &f.key, &f.id, &f.denied, &f.raw, &f.enterprise); e != nil {
				rows.Close()
				return e
			}
			batch = append(batch, f)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		fetched, transferred := len(batch), 0
		for _, row := range batch {
			transferred += len(row.raw) + len(row.enterprise)
		}
		for _, f := range batch {
			if e = ctx.Err(); e != nil {
				return e
			}
			if f.denied != 0 {
				return scopeQueryError(f.denied)
			}
			if f.kind == 0 {
				if header {
					return queryUnavailable()
				}
				header = true
				var captured capturedEnterprise
				if json.Unmarshal(f.enterprise, &captured) != nil {
					return queryUnavailable()
				}
				enterpriseName, e = decodeQueryEnterpriseName(ctx, p.Enterprise, p.Day, captured)
				if e != nil {
					return e
				}
				if q.AnchorID == "" {
					continue
				}
				if f.id == "" {
					return foundation.Fail(409, "M02_NAVIGATION_CHANGED", "原对象已不在当前查询中，请返回列表")
				}

			}
			if !header || f.kind != 0 && f.kind != 1 {
				return queryUnavailable()
			}
			capture, e := decodeDepartmentQuery(ctx, p.Enterprise, p.Day, 1, f.raw, &enterpriseName)
			if e != nil {
				return e
			}
			capture.Stats.FetchedFrames, capture.Stats.TransferredMaterialBytes = fetched, transferred
			fetched, transferred = 0, 0
			more, e := visit(app.QueryFrame{Anchor: f.kind == 0, Key: f.key, ID: f.id, Capture: capture})
			if e != nil {
				return e
			}
			if !more {
				return nil
			}
		}
		if len(batch) < batchSize {
			if !header {
				return queryUnavailable()
			}
			return nil
		}
		batchSize = 64
	}
}
