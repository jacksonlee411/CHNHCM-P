package postgres

import (
	"context"
	"errors"
	"time"

	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

// Current reads use the dated projection. Old versions select the record in
// each sealed position source at the requested business date.
const liveStructurePositions = `
 SELECT h.id,h.code,h.source_id AS position_source,r.record_id,r.fact->'data'->>'name' AS name
 FROM staffing.position h JOIN staffing.position_read r ON (r.instance_id,r.object_id,r.source_id)=(h.instance_id,h.id,h.source_id)
 WHERE h.instance_id=@instance AND h.enterprise_id=@enterprise AND r.active AND r.valid_period @> @day::date`
const sealedStructurePositions = `
 SELECT n.position_id AS id,p.snapshot->>'code' AS code,n.position_source,(f.record->>'id')::uuid AS record_id,f.record->'data'->>'name' AS name
 FROM staffing.position_hierarchy_node n
 JOIN staffing.position_source p ON (p.instance_id,p.object_id,p.source_id)=(n.instance_id,n.position_id,n.position_source)
 CROSS JOIN LATERAL (SELECT record FROM jsonb_array_elements(p.snapshot->'records') record
 WHERE (record->>'date')::date <= @day::date ORDER BY (record->>'date')::date DESC LIMIT 1) f
 WHERE n.instance_id=@instance AND n.hierarchy_id=@root AND n.source_id=@version AND f.record->>'type'<>'DEACTIVATE'`

const structureEdges = `), edges AS (
 SELECT child_id,parent_id FROM staffing.position_hierarchy_edge
 WHERE instance_id=@instance AND hierarchy_id=@root AND source_id=@version AND period @> @day::date AND parent_id IS NOT NULL
), nodes AS NOT MATERIALIZED (
 SELECT p.*,e.parent_id FROM positions p LEFT JOIN edges e ON e.child_id=p.id
)`
const structureColumns = `n.id::text,n.name,CASE WHEN @code THEN n.code ELSE '' END AS code,coalesce(n.parent_id::text,'') AS parent_id,
 EXISTS(SELECT 1 FROM edges e WHERE e.parent_id=n.id) AS has_children,n.position_source::text,n.record_id::text`

func (u *unit) ReadStructure(ctx context.Context, enterprise string, q s.StructureQuery, code, history bool) (out s.StructureView, err error) {
	out = s.StructureView{BusinessAt: q.BusinessAt, CanReadHistory: history, Nodes: []s.StructureNode{}}
	args := pgx.NamedArgs{"instance": u.store.Instance, "enterprise": enterprise, "source": q.SourceVersion, "day": q.BusinessAt,
		"id": q.NodeID, "q": q.Q, "code": code, "offset": (q.Page - 1) * q.PageSize, "limit": q.PageSize + 1}
	var root string
	err = u.tx.QueryRow(ctx, `SELECT h.id::text,v.source_id::text FROM staffing.position_hierarchy h
 JOIN staffing.position_hierarchy_version v ON (v.instance_id,v.hierarchy_id)=(h.instance_id,h.id)
 WHERE h.instance_id=@instance AND h.enterprise_id=@enterprise
 AND (@source::text='' OR v.source_id=nullif(@source,'')::uuid) ORDER BY v.version_no DESC LIMIT 1`, args).Scan(&root, &out.SourceVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		if q.SourceVersion != "" {
			return out, foundation.Fail(404, "M03_NOT_FOUND", "找不到指定的管理汇报版本")
		}
	} else if err != nil {
		return out, err
	}
	// SQL NULL naturally represents a hierarchy with no published versions yet.
	args["root"], args["version"] = nil, nil
	if root != "" {
		args["root"], args["version"] = root, out.SourceVersion
	}
	if q.Mode == "versions" {
		rows, e := u.tx.Query(ctx, `SELECT source_id::text,version_no,recorded_at FROM staffing.position_hierarchy_version
   WHERE instance_id=@instance AND hierarchy_id=@root ORDER BY version_no DESC OFFSET @offset LIMIT @limit`, args)
		if e != nil {
			return out, e
		}
		out.Versions, e = pgx.CollectRows(rows, pgx.RowToStructByPos[s.HierarchyVersion])
		if len(out.Versions) > q.PageSize {
			out.More = true
			out.Versions = out.Versions[:q.PageSize]
		}
		return out, e
	}
	positions := liveStructurePositions
	if q.SourceVersion != "" {
		positions = sealedStructurePositions
	}
	prefix := "WITH RECURSIVE positions AS NOT MATERIALIZED (" + positions + structureEdges
	read := func(tail string) ([]s.StructureNode, error) {
		rows, e := u.tx.Query(ctx, prefix+tail, args)
		if e != nil {
			return nil, e
		}
		return pgx.CollectRows(rows, pgx.RowToStructByPos[s.StructureNode])
	}
	if q.Mode == "path" {
		out.Nodes, err = read(`, ranked AS (SELECT n.*,row_number() OVER(PARTITION BY n.parent_id ORDER BY n.name COLLATE "C",n.id) AS ordinal FROM nodes n), path(id,parent_id,depth,visited) AS (
   SELECT id,parent_id,0,ARRAY[id] FROM nodes WHERE id=@id::uuid
   UNION ALL SELECT n.id,n.parent_id,p.depth+1,p.visited||n.id FROM path p JOIN nodes n ON n.id=p.parent_id WHERE NOT n.id=ANY(p.visited))
   SELECT ((n.ordinal-1)/(@limit::int-1)+1)::int,` + structureColumns + ` FROM path p JOIN ranked n ON n.id=p.id ORDER BY p.depth DESC`)
		if err == nil && (len(out.Nodes) == 0 || out.Nodes[0].ParentID != "") {
			return out, foundation.Fail(404, "M03_NOT_FOUND", "该日期的岗位路径不可用")
		}
		return out, err
	}
	if q.Mode == "relation" || q.Mode == "node" {
		pair, e := read(" SELECT 0," + structureColumns + ` FROM nodes n WHERE n.id=@id::uuid OR n.id=(SELECT parent_id FROM nodes WHERE id=@id::uuid)`)
		if e != nil {
			return out, e
		}
		for i := range pair {
			if pair[i].ID == q.NodeID {
				out.Selected = &pair[i]
			} else {
				out.Parent = &pair[i]
			}
		}
		if out.Selected == nil {
			return out, foundation.Fail(404, "M03_NOT_FOUND", "该日期没有可读的有效岗位")
		}
		if q.Mode == "node" {
			return out, nil
		}
		if out.Selected.ParentID != "" && out.Parent == nil {
			return out, foundation.Fail(403, "M03_FORBIDDEN", "岗位关系不可读")
		}
		var endExclusive string
		e = u.tx.QueryRow(ctx, `SELECT coalesce(upper(current_edge.period)::text,''),coalesce(lower(future.period)::text,'')
   FROM (SELECT 1) one LEFT JOIN LATERAL (SELECT period FROM staffing.position_hierarchy_edge
   WHERE instance_id=@instance AND hierarchy_id=@root AND source_id=@version AND child_id=@id::uuid AND period @> @day::date) current_edge ON true
   LEFT JOIN LATERAL (SELECT period FROM staffing.position_hierarchy_edge
   WHERE instance_id=@instance AND hierarchy_id=@root AND source_id=@version AND child_id=@id::uuid AND lower(period)>@day::date ORDER BY lower(period) LIMIT 1) future ON true`, args).Scan(&endExclusive, &out.NextStart)
		if e != nil {
			return out, e
		}
		if endExclusive == "" {
			endExclusive = out.NextStart
		}
		if endExclusive != "" {
			day, err := time.Parse(time.DateOnly, endExclusive)
			if err != nil {
				return out, err
			}
			out.EndInclusive = day.AddDate(0, 0, -1).Format(time.DateOnly)
		}
	}
	predicate := "n.parent_id IS NOT DISTINCT FROM nullif(@id,'')::uuid"
	if q.Mode == "search" || q.Mode == "candidates" {
		predicate = "(strpos(lower(n.name),lower(@q))>0 OR @code AND strpos(lower(n.code),lower(@q))>0)"
	}
	if q.Mode == "candidates" {
		predicate += " AND n.id IS DISTINCT FROM nullif(@id,'')::uuid"
	}
	out.Nodes, err = read(" SELECT 0," + structureColumns + " FROM nodes n WHERE " + predicate + ` ORDER BY n.name COLLATE "C",n.id OFFSET @offset LIMIT @limit`)
	if len(out.Nodes) > q.PageSize {
		out.More = true
		out.Nodes = out.Nodes[:q.PageSize]
	}
	return out, err
}
