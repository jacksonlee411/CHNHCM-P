package postgres

import (
	"context"
	"errors"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

// Each relationship is re-read under the online person lock. A retry compares
// against the latest published edition and never mutates a previous edition.
func (s *Store) ReconcileProbation(ctx context.Context, c f.Credentials, apply bool) (report h.ProbationRepairReport, err error) {
	report.Items = []h.ProbationRepairItem{}
	var ids []string
	err = s.within(ctx, c, false, true, func(u *unit) error {
		allowed, err := u.employmentAllowed(ctx, "probation.confirm", "", "", "", true)
		if err != nil {
			return err
		}
		if !allowed {
			return f.Fail(403, "HIRE_DENIED", "当前账号没有转正办理权限")
		}
		rows, err := u.tx.Query(ctx, `SELECT id FROM staffing.employment_action WHERE instance_id=$1 AND kind='ConfirmEmployment' ORDER BY id`, s.Instance)
		if err != nil {
			return err
		}
		ids, err = pgx.CollectRows(rows, pgx.RowTo[string])
		return err
	})
	if err != nil {
		return report, err
	}
	for _, id := range ids {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		visible, changed := false, false
		err = s.within(ctx, c, apply, false, func(u *unit) error {
			act, err := u.action(ctx, id)
			if err != nil {
				return err
			}
			if err = lock(ctx, u.tx, false, "staffing.hire.person", s.Instance, act.Person); err != nil {
				return err
			}
			act, err = u.action(ctx, id)
			if err != nil {
				return err
			}
			_, visible, err = u.relationVisibleAssignment(ctx, act, "history")
			if !visible || err != nil {
				return err
			}
			permitted, err := u.employmentAllowed(ctx, "probation.confirm", act.Person, act.Employer, "", true)
			if err != nil || !permitted {
				visible = false
				return err
			}
			depts, err := u.relationDepartments(ctx, act.Employment, act.BusinessDate)
			if err != nil {
				return err
			}
			term := act.Assignment
			for _, d := range depts {
				if d.Primary {
					term = d.Assignment
					break
				}
			}
			a, err := u.assignmentAt(ctx, term, act.BusinessDate, "")
			if err != nil {
				return err
			}
			confirm, err := u.currentConfirmation(ctx, a)
			if err != nil {
				return err
			}
			if confirm.ID != id || !a.Includes(confirm.Day) {
				return f.Fail(409, "PROBATION_PERIOD", "请核对该日工作关系与主任职")
			}
			initial, records, err := u.historyRows(ctx, a)
			if err != nil {
				return err
			}
			originals := map[string]savedRecord{confirm.ID: confirm}
			for _, row := range records {
				originals[row.ID] = row
			}
			confirm.Operation = "Rebase"
			records = append(records, confirm)
			result, err := u.replayRecordEdit(ctx, a, initial, records, originals, len(records)-1, "probation.confirm", "", !apply)
			if err != nil {
				return err
			}
			changed = !result.reused
			if apply && changed {
				_, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_outbox(instance_id,event_id,action_id,revision) VALUES($1,$2,$3,$4) ON CONFLICT(instance_id,action_id,revision) DO NOTHING`, s.Instance, f.NewID(), result.action, result.revision)
			}
			return err
		})
		item := h.ProbationRepairItem{}
		if visible {
			report.Scanned++
			item.Action = id
		}
		if err != nil {
			var fault *f.Fault
			if errors.As(err, &fault) && fault.Status < 500 && visible {
				report.Unresolved++
				item.Result, item.Reason = "unresolved", fault.Code
			} else {
				report.Failed++
				item.Result, item.Reason = "failed", "TRANSACTION_FAILED"
			}
		} else if !changed {
			continue
		} else {
			report.Differences++
			item.Result = "would_repair"
			if apply {
				report.Repaired++
				item.Result = "repaired"
			}
		}
		report.Items = append(report.Items, item)
	}
	return report, nil
}
