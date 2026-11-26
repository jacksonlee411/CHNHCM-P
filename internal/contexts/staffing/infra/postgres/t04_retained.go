package postgres

import (
	"context"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func (s *Store) inspectT04Retained(ctx context.Context, c f.Credentials) (report h.ProbationRepairReport, err error) {
	err = s.within(ctx, c, false, true, func(u *unit) error {
		rows, err := u.tx.Query(ctx, `SELECT assignment_id,action_id,first_inactive::text,(first_inactive-1)::text
			FROM staffing.assignment_exit WHERE instance_id=$1 ORDER BY action_id,assignment_id`, s.Instance)
		if err != nil {
			return err
		}
		terms, err := pgx.CollectRows(rows, pgx.RowToStructByPos[struct{ Assignment, Action, Boundary, Last string }])
		if err != nil {
			return err
		}
		reported := map[string]bool{}
		for _, term := range terms {
			if reported[term.Action] {
				continue
			}
			a, err := u.assignmentIdentity(ctx, term.Assignment)
			if err != nil {
				return err
			}
			at, err := u.assignmentPlacement(ctx, a, term.Last)
			if err != nil {
				return err
			}
			visible := true
			for _, grant := range []string{"read", "history", "duty.verify"} {
				visible, err = u.employmentAllowed(ctx, grant, a.Person, a.Employer, at.Department, false)
				if err != nil {
					return err
				}
				if !visible {
					break
				}
			}
			if !visible {
				continue
			}
			report.Scanned++
			conflict, err := u.exitConflictsWithDuty(ctx, term.Assignment, term.Boundary)
			if err != nil {
				return err
			}
			if conflict {
				reported[term.Action] = true
				report.Differences++
				report.Unresolved++
				report.Items = append(report.Items, h.ProbationRepairItem{Action: term.Action, Result: "unresolved", Reason: "EXIT_DUTY"})
			}
		}
		return u.retainedIdentities(ctx, &report)
	})
	return report, err
}
