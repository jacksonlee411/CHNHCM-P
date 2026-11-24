package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"slices"
)

func (u *unit) LockHierarchy(ctx context.Context, enterprise string) error {
	return u.guardLock(ctx, u.tx, false, "staffing.hierarchy", u.store.Instance, enterprise)
}
func (u *unit) LockHierarchyNodes(ctx context.Context, ids []string) error {
	sorted := slices.Clone(ids)
	slices.Sort(sorted)
	for _, id := range slices.Compact(sorted) {
		if err := u.guardLock(ctx, u.tx, true, "staffing.position", u.store.Instance, id); err != nil {
			return err
		}
	}
	return nil
}
func (u *unit) HierarchyVersions(ctx context.Context, enterprise string) (string, []s.HierarchyVersion, error) {
	var root string
	err := u.tx.QueryRow(ctx, "SELECT id::text FROM staffing.position_hierarchy WHERE instance_id=$1 AND enterprise_id=$2", u.store.Instance, enterprise).Scan(&root)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", []s.HierarchyVersion{}, nil
	}
	if err != nil {
		return "", nil, err
	}
	rows, err := u.tx.Query(ctx, "SELECT source_id::text,version_no,recorded_at FROM staffing.position_hierarchy_version WHERE instance_id=$1 AND hierarchy_id=$2 ORDER BY version_no DESC", u.store.Instance, root)
	if err != nil {
		return "", nil, err
	}
	versions, err := pgx.CollectRows(rows, pgx.RowToStructByPos[s.HierarchyVersion])
	if err == nil && len(versions) == 0 {
		return "", nil, errors.New("missing published hierarchy version")
	}
	return root, versions, err
}

const hierarchyColumns = "e.child_id::text,coalesce(e.parent_id::text,''),lower(e.period)::text,coalesce(upper(e.period)::text,'')"

func (u *unit) HierarchyEdges(ctx context.Context, root, version string) (d.HierarchyEdges, error) {
	if root == "" {
		return nil, nil
	}
	rows, err := u.tx.Query(ctx, "SELECT "+hierarchyColumns+" FROM staffing.position_hierarchy_edge e WHERE instance_id=$1 AND hierarchy_id=$2 AND source_id=$3 ORDER BY child_id,lower(period)", u.store.Instance, root, version)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[d.HierarchyEdge])
}
func (u *unit) CurrentHierarchyUses(ctx context.Context, enterprise, id string) (d.HierarchyEdges, error) {
	rows, err := u.tx.Query(ctx, "SELECT "+hierarchyColumns+` FROM staffing.position_hierarchy h CROSS JOIN LATERAL (SELECT source_id FROM staffing.position_hierarchy_version v WHERE v.instance_id=h.instance_id AND v.hierarchy_id=h.id ORDER BY version_no DESC LIMIT 1) v JOIN staffing.position_hierarchy_edge e ON e.source_id=v.source_id WHERE h.instance_id=$1 AND h.enterprise_id=$2 AND (e.instance_id,e.hierarchy_id)=(h.instance_id,h.id) AND e.parent_id IS NOT NULL AND (e.parent_id=$3 OR e.child_id=$3)`, u.store.Instance, enterprise, id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[d.HierarchyEdge])
}
func (u *unit) HierarchyPositions(ctx context.Context, enterprise, date, root, version string) ([]d.Timeline, error) {
	if version != "" {
		rows, err := u.tx.Query(ctx, `SELECT p.snapshot FROM staffing.position_hierarchy_node n JOIN staffing.position_source p ON (p.instance_id,p.object_id,p.source_id)=(n.instance_id,n.position_id,n.position_source) WHERE n.instance_id=$1 AND n.hierarchy_id=$2 AND n.source_id=$3`, u.store.Instance, root, version)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var result []d.Timeline
		for rows.Next() {
			var snapshot []byte
			if err = rows.Scan(&snapshot); err != nil {
				return nil, err
			}
			position, err := decode(snapshot, d.Position)
			if err != nil {
				return nil, err
			}
			if position.Enterprise != enterprise {
				return nil, errors.New("hierarchy position enterprise mismatch")
			}
			result = append(result, position)
		}
		return result, rows.Err()
	}
	rows, err := u.tx.Query(ctx, "SELECT object_id::text FROM staffing.position_read WHERE instance_id=$1 AND enterprise_id=$2 AND active AND valid_period @> $3::date", u.store.Instance, enterprise, date)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	return u.canonicalHeads(ctx, d.Position, enterprise, ids)
}
func (u *unit) SaveHierarchy(ctx context.Context, enterprise string, result s.Result, number int, edges d.HierarchyEdges, operator access.OperatorSnapshot) error {
	instance, root, source := u.store.Instance, result.ObjectID, result.SourceVersion
	if number == 1 {
		if _, err := u.tx.Exec(ctx, "INSERT INTO staffing.position_hierarchy(instance_id,enterprise_id,id) VALUES($1,$2,$3)", instance, enterprise, root); err != nil {
			return err
		}
	}
	batch := &pgx.Batch{}
	batch.Queue("INSERT INTO staffing.position_hierarchy_node SELECT instance_id,$2,$3,id,source_id FROM staffing.position WHERE instance_id=$1 AND enterprise_id=$4", instance, root, source, enterprise)
	for _, edge := range edges {
		batch.Queue("INSERT INTO staffing.position_hierarchy_edge(instance_id,hierarchy_id,source_id,child_id,parent_id,period) VALUES($1,$2,$3,$4,nullif($5,'')::uuid,daterange($6,nullif($7,'')::date,'[)'))", instance, root, source, edge.Child, edge.Parent, edge.Start, edge.End)
	}
	batch.Queue("INSERT INTO staffing.position_hierarchy_version VALUES($1,$2,$3,$4,$5,$6,$7)", instance, root, source, number, operator, u.now, result.SubmissionID)
	batch.Queue("INSERT INTO staffing.position_hierarchy_outbox VALUES($1,$2,$3,$4)", instance, root, source, s.View{"type": "staffing.positionHierarchy.published", "hierarchyId": root, "sourceVersion": source, "versionNo": number})
	return u.tx.SendBatch(ctx, batch).Close()
}
