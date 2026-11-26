package postgres

import (
	"context"
	"errors"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type lifecycleRepairCandidate struct {
	action, assignment, person, purpose, kind string
}

func (s *Store) ReconcileT04(ctx context.Context, c f.Credentials, apply bool) (report h.ProbationRepairReport, err error) {
	report.Items = []h.ProbationRepairItem{}
	var candidates []lifecycleRepairCandidate
	err = s.within(ctx, c, false, true, func(u *unit) error {
		rows, err := u.tx.Query(ctx, `SELECT a.id,t.id,t.person_id,coalesce(r.adoption#>>'{manager,purpose}',''),a.kind
			FROM staffing.employment_action a JOIN staffing.assignment t ON (t.instance_id,t.id)=(a.instance_id,a.assignment_id)
			JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(a.instance_id,a.id,a.revision)
			WHERE a.instance_id=$1 AND (a.kind IN ('Leave','EndAssignment') OR
				(a.kind='SetManager' AND r.adoption#>>'{managerExit,action}' IS NOT NULL)) ORDER BY t.person_id,t.id,a.id`, s.Instance)
		if err != nil {
			return err
		}
		candidates, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (v lifecycleRepairCandidate, e error) {
			e = r.Scan(&v.action, &v.assignment, &v.person, &v.purpose, &v.kind)
			return
		})
		return err
	})
	if err != nil {
		return report, err
	}
	for _, candidate := range candidates {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		visible, changed := false, false
		err = s.within(ctx, c, apply, false, func(u *unit) error {
			if candidate.kind == "SetManager" {
				if err := u.lockReporting(ctx); err != nil {
					return err
				}
			}
			if err := lock(ctx, u.tx, false, "staffing.hire.person", s.Instance, candidate.person); err != nil {
				return err
			}
			a, err := u.action(ctx, candidate.action)
			if err != nil {
				return err
			}
			read, err := u.actionDisclosure(ctx, a, true)
			if err != nil || !(read.from || read.to) {
				return err
			}
			permit := map[string]string{"Leave": "employment.leave", "EndAssignment": "assignment.end", "SetManager": "manager.change"}[candidate.kind]
			visible, err = u.employmentAllowed(ctx, permit, a.Person, a.Employer, a.Department, false)
			if err != nil || !visible {
				return err
			}
			if candidate.kind != "SetManager" {
				ref, err := u.reconcileExitDepartment(ctx, candidate.assignment, nil, apply)
				changed = ref != nil
				return err
			}
			decisions, err := u.managerDecisions(ctx, candidate.purpose)
			if err != nil {
				return err
			}
			_, obsolete := reportingChanges(decisions, candidate.assignment, nil)
			for _, d := range obsolete {
				if d.action != candidate.action {
					continue
				}
				var retained []h.ManagerEdge
				for _, other := range decisions {
					if other.action != d.action {
						retained = append(retained, other.edge)
					}
				}
				if err := h.ValidateManagers(retained); err != nil {
					return err
				}
				changed = true
				if !apply {
					return nil
				}
				refs, err := u.withdrawManagerEnds(ctx, []managerDecision{d})
				if err != nil {
					return err
				}
				return u.lifecycleSource(ctx, a.Assignment, a.SourceVersion, refs[0].Action, refs[0].Revision, "")
			}
			return nil
		})
		item := h.ProbationRepairItem{}
		if visible {
			report.Scanned++
			item.Action = candidate.action
		}
		switch {
		case err != nil:
			var fault *f.Fault
			if errors.As(err, &fault) && fault.Status < 500 && visible {
				report.Unresolved++
				item.Result, item.Reason = "unresolved", fault.Code
			} else {
				report.Failed++
				item.Result, item.Reason = "failed", "TRANSACTION_FAILED"
			}
		case !changed:
			continue
		default:
			report.Differences++
			item.Result = "would_repair"
			if apply {
				report.Repaired++
				item.Result = "repaired"
			}
		}
		report.Items = append(report.Items, item)
	}
	remaining, err := s.inspectT04Retained(ctx, c)
	report.Scanned += remaining.Scanned
	report.Differences += remaining.Differences
	report.Unresolved += remaining.Unresolved
	report.Failed += remaining.Failed
	report.Items = append(report.Items, remaining.Items...)
	return report, err
}
