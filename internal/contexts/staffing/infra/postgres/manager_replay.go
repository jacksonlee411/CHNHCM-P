package postgres

import (
	"context"
	"slices"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

// A later effective decision ends the preceding agreement, even if the later
// decision has an end of its own. Withdrawn derived ends have no agreement row
// for their new revision, so each source selects its exact reporting graph.
const currentReporting = `WITH reporting AS (
	SELECT g.*,g.period * daterange(NULL,lead(lower(g.period)) OVER (
	    PARTITION BY g.instance_id,g.assignment_id,g.purpose ORDER BY lower(g.period)),'[)') AS effective
	FROM staffing.manager_agreement g JOIN staffing.assignment_head hd USING(instance_id,assignment_id)
	JOIN staffing.assignment_member m ON (m.instance_id,m.assignment_id,m.version,m.action_id,m.revision)=(hd.instance_id,hd.assignment_id,hd.version,g.action_id,g.revision)
	WHERE g.instance_id=$1) `

type managerDecision struct {
	edge             h.ManagerEdge
	action, revision string
	derived          bool
	closing          []string
}

func (u *unit) managerDecisions(ctx context.Context, purpose string) ([]managerDecision, error) {
	rows, err := u.tx.Query(ctx, currentReporting+`SELECT g.assignment_id,coalesce(g.manager_assignment::text,''),lower(g.period)::text,coalesce(upper(g.period)::text,''),
		g.action_id,g.revision,r.adoption#>>'{managerExit,action}' IS NOT NULL,
		ARRAY(SELECT x.assignment_id::text FROM staffing.assignment_exit x WHERE x.instance_id=g.instance_id AND x.action_id=(r.adoption#>>'{managerExit,action}')::uuid)
		FROM reporting g JOIN staffing.employment_action_revision r USING(instance_id,action_id,revision)
		WHERE g.purpose=$2 ORDER BY g.assignment_id,lower(g.period)`, u.store.Instance, purpose)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (v managerDecision, err error) {
		err = row.Scan(&v.edge.Child, &v.edge.Parent, &v.edge.From, &v.edge.Until, &v.action, &v.revision, &v.derived, &v.closing)
		return
	})
}

// Re-evaluate only automatic ends for this subordinate and purpose. Explicit
// decisions are authoritative, including the manual decision to have no manager.
func reportingChanges(decisions []managerDecision, child string, proposed *h.ManagerFact) (edges []h.ManagerEdge, obsolete []managerDecision) {
	explicit := []h.ManagerEdge{}
	for _, d := range decisions {
		if d.edge.Child == child && !d.derived {
			explicit = append(explicit, d.edge)
		}
	}
	if proposed != nil {
		explicit = append(explicit, h.ManagerEdge{Child: child, Parent: proposed.Manager, From: proposed.From, Until: proposed.Until})
	}
	for _, d := range decisions {
		if d.derived && d.edge.Child == child {
			var latest h.ManagerEdge
			for _, e := range explicit {
				if e.From <= d.edge.From && e.From >= latest.From {
					latest = e
				}
			}
			if latest.From == d.edge.From || latest.Parent == "" || !slices.Contains(d.closing, latest.Parent) || latest.Until != "" && latest.Until <= d.edge.From {
				obsolete = append(obsolete, d)
				continue
			}
		}
		edges = append(edges, d.edge)
	}
	return
}

func (u *unit) withdrawManagerEnds(ctx context.Context, ends []managerDecision) ([]h.ActionRef, error) {
	if len(ends) == 0 {
		return nil, nil
	}
	operator, err := u.guard.OperatorSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	var refs []h.ActionRef
	for _, end := range ends {
		ref := h.ActionRef{Action: end.action, Revision: f.NewID()}
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_action_revision(instance_id,assignment_id,action_id,revision,previous_revision,business_date,operation,notice,actor,operator_snapshot,adoption)
			SELECT instance_id,assignment_id,action_id,$4,revision,business_date,'Withdraw',notice,$5,$6,adoption || jsonb_build_object('reason','已核对退出日的独立汇报安排')
			FROM staffing.employment_action_revision WHERE instance_id=$1 AND action_id=$2 AND revision=$3`, u.store.Instance, end.action, end.revision, ref.Revision, u.guard.Principal().ID, operator)
		if err != nil {
			return nil, err
		}
		_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_outbox(instance_id,event_id,action_id,revision) VALUES($1,$2,$3,$4)`, u.store.Instance, f.NewID(), ref.Action, ref.Revision)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}
