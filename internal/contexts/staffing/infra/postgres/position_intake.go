package postgres

import (
	"context"
	"errors"

	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

// AssignmentPositionUsage reads M04's takeover from the original creation and
// the actual segments of the published assignment source. No M05 coverage is implied.
func (st *Store) AssignmentPositionUsage(ctx context.Context, instance, position string) (out d.QuotaUsage, err error) {
	tx, ok := dbtx.From(ctx)
	if !ok || instance != st.Instance {
		return out, errors.New("position usage requires caller transaction")
	}
	var c d.SourceCoverage
	e := tx.QueryRow(ctx, `SELECT lower(period)::text,coalesce(upper(period)::text,''),source_id FROM staffing.position_intake WHERE instance_id=$1 AND position_id=$2`, instance, position).Scan(&c.From, &c.Until, &c.Version)
	if e == pgx.ErrNoRows {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	c.Owner = "M04"
	out.Coverage = []d.SourceCoverage{c}
	// The Employee hire, transfer, secondary and recruiting paths consume these
	// rules. Another category or purpose cannot take over this position's quota.
	rows, e := tx.Query(ctx, `WITH current_policy AS (
    SELECT x.version FROM staffing.position p JOIN staffing.assignment_policy_source x ON (x.instance_id,x.owner_id)=(p.instance_id,p.enterprise_id)
    WHERE p.instance_id=$1 AND p.id=$2 ORDER BY x.sequence DESC LIMIT 1)
 SELECT lower(applicable)::text,coalesce(upper(applicable)::text,'')
 FROM current_policy x JOIN staffing.assignment_policy_member m USING(version)
 CROSS JOIN LATERAL (SELECT m.period * daterange($3::date,nullif($4,'')::date,'[)') AS applicable) p
 WHERE m.instance_id=$1 AND NOT isempty(applicable)
   AND m.fact->>'category'='Employee' AND m.fact->>'purpose' IN ('hire','transfer','secondary','recruit')
   AND m.fact->>'mode' IN ('PositionExempt','PositionCounted')`, instance, position, c.From, c.Until)
	if e != nil {
		return out, e
	}
	out.Managed, e = pgx.CollectRows(rows, pgx.RowToStructByPos[d.UsageSpan])
	if e != nil {
		return out, e
	}
	rows, e = tx.Query(ctx, `SELECT lower(s.period)::text,coalesce(upper(s.period)::text,''),s.version,s.adoption->>'policy',CASE WHEN nullif(s.adoption->>'handoverSource','') IS NOT NULL THEN 'handover' WHEN (s.adoption->>'seats')::int>0 THEN 'occupied' ELSE 'exempt' END,s.department_id,s.job_id,s.assignment_id,coalesce(replacement.incumbent::text,'')
 FROM staffing.assignment_head h JOIN staffing.assignment_segment s USING(instance_id,assignment_id,version)
 LEFT JOIN staffing.position_succession replacement ON replacement.instance_id=s.instance_id AND replacement.id=coalesce(nullif(s.adoption->>'successionSource',''),nullif(s.adoption->>'handoverSource',''))::uuid
 WHERE s.instance_id=$1 AND s.adoption->>'position'=$2`, instance, position)
	if e != nil {
		return out, e
	}
	out.Uses, e = pgx.CollectRows(rows, func(r pgx.CollectableRow) (v d.PositionUse, err error) {
		v.Owner = "M04"
		err = r.Scan(&v.From, &v.Until, &v.Source, &v.Policy, &v.Effect, &v.Department, &v.Job, &v.Assignment, &v.Incumbent)
		return
	})
	for _, v := range out.Uses {
		out.Managed = append(out.Managed, v.UsageSpan)
	}
	return out, e
}
