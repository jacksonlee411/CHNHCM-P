package postgres

import (
	"context"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type probationPredecessor struct {
	h.PrecedingOption
	assignment string
	evidence   h.Original
	visible    bool
}

// The personnel-management primary belongs to the relationship, even when the
// operator opens the confirmation from a secondary assignment.
func (u *unit) probationPredecessors(ctx context.Context, a h.Action, day string) ([]probationPredecessor, error) {
	rows, err := u.tx.Query(ctx, `SELECT act.id,r.revision,act.kind,r.notice,r.business_date::text,act.assignment_id,r.adoption
		FROM staffing.employment_primary p JOIN staffing.assignment_head hd ON (hd.instance_id,hd.assignment_id)=(p.instance_id,p.target_id)
		JOIN staffing.assignment_member m ON (m.instance_id,m.assignment_id,m.version)=(hd.instance_id,hd.assignment_id,hd.version)
		JOIN staffing.employment_action act ON (act.instance_id,act.id,act.assignment_id)=(m.instance_id,m.action_id,m.assignment_id)
		JOIN staffing.employment_action_revision r ON (r.instance_id,r.action_id,r.revision)=(m.instance_id,m.action_id,m.revision)
		WHERE p.instance_id=$1 AND p.layer='assignment' AND p.owner_id=$2 AND p.purpose='personnel_management'
		AND p.period @> $3::date AND act.kind IN ('Transfer','Promotion') AND r.business_date=$3::date
		AND r.operation NOT IN ('Delete','Withdraw') ORDER BY act.id`, u.store.Instance, a.Employment, day)
	if err != nil {
		return nil, err
	}
	facts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (p probationPredecessor, err error) {
		err = row.Scan(&p.ID, &p.Revision, &p.Kind, &p.Notice, &p.BusinessDate, &p.assignment, &p.evidence)
		return
	})
	if err != nil {
		return nil, err
	}
	for i := range facts {
		p := &facts[i]
		p.visible, err = u.canReadProbationPredecessor(ctx, a, p.evidence)
		if err != nil {
			return nil, err
		}
	}
	return facts, nil
}

func (u *unit) canReadProbationPredecessor(ctx context.Context, a h.Action, evidence h.Original) (bool, error) {
	var departments []string
	if promotion := evidence.Promotion; promotion != nil {
		departments = append(departments, promotion.Department)
	}
	if move := evidence.Transfer; move != nil {
		departments = append(departments, move.After.Department)
	}
	for _, dept := range departments {
		for _, right := range []string{"read", "history"} {
			ok, err := u.employmentAllowed(ctx, right, a.Person, a.Employer, dept, false)
			if err != nil || !ok {
				return false, err
			}
		}
	}
	return len(departments) > 0, nil
}

func (st *Store) ProbationPredecessors(ctx context.Context, c f.Credentials, id, day string) (out []h.PrecedingOption, err error) {
	if !f.ValidDate(day) {
		return nil, f.Fail(400, "PROBATION_INPUT", "请核对转正日期")
	}
	out = []h.PrecedingOption{}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, err := u.assignmentAt(ctx, id, day, "")
		if err != nil {
			return err
		}
		ok, err := u.employmentAllowed(ctx, "probation.confirm", a.Person, a.Employer, "", true)
		if err != nil || !ok {
			return err
		}
		facts, err := u.probationPredecessors(ctx, a, day)
		if err != nil {
			return err
		}
		for _, p := range facts {
			if p.visible {
				out = append(out, p.PrecedingOption)
			}
		}
		return nil
	})
	return
}

func (u *unit) applyProbationPredecessor(ctx context.Context, a h.Action, in h.ProbationCommand, ev *h.ProbationEvidence) error {
	facts, err := u.probationPredecessors(ctx, a, in.BusinessDate)
	if err != nil {
		return err
	}
	for _, p := range facts {
		if p.ID != in.AfterAction || p.Revision != in.AfterRevision {
			continue
		}
		if !p.visible {
			return f.Fail(403, "HIRE_DENIED", "当前权限不可采用所引原件")
		}
		ev.AfterAction, ev.AfterRevision = p.ID, p.Revision
		if ev.Attribution == "notice" {
			return nil
		}
		// A promotion can carry its own transfer. Its occurrence department is
		// the before-image, while the adopted segment supplies the after-image.
		if move := p.evidence.Transfer; move != nil {
			for _, seg := range move.After.Segments {
				if seg.Includes(in.BusinessDate) {
					ev.Department, ev.DepartmentName = seg.DepartmentID, seg.DepartmentName
					if ev.Department == "" {
						ev.Department = move.After.Department
					}
					ev.DepartmentSource, ev.DepartmentRevision = seg.DepartmentSource, seg.DepartmentRevision
					ev.Attribution = "preceding-transfer"
					return nil
				}
			}
			return assignmentConflict()
		}
		// A grade-only decision documents order but does not disambiguate an
		// independently timed move. Use the same daily projection as a replay.
		primary, err := u.assignmentIdentity(ctx, p.assignment)
		if err != nil {
			return err
		}
		initial, records, err := u.historyRows(ctx, primary)
		if err != nil {
			return err
		}
		pending := savedRecord{HistoryEntry: h.HistoryEntry{ID: f.NewID(), Kind: "ConfirmEmployment", Day: in.BusinessDate, Previous: &h.ActionRef{Action: p.ID, Revision: p.Revision}}}
		projected, err := projectRecords(primary, initial, append(records, pending))
		if err != nil {
			return err
		}
		change := projected.Changes[pending.ID]
		if change.Alternate != nil && change.Alternate.Department != change.Before.Department {
			ev.Attribution = "same-day-order-undetermined"
			return nil
		}
		seg := change.Before.Adoption
		ev.Department, ev.DepartmentName = change.Before.Department, seg.DepartmentName
		ev.DepartmentSource, ev.DepartmentRevision = seg.DepartmentSource, seg.DepartmentRevision
		ev.Attribution = "preceding-action"
		return nil
	}
	return f.Fail(409, "PROBATION_ORDER", "所引办理与关系主任职、日期或修订不符")
}
