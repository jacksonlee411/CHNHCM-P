package postgres

import (
	"context"
	"errors"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	"github.com/jackc/pgx/v5"
)

type initialRecordMode struct {
	Action, Revision, Recorded, Effective string
}

// Read the source that first published this revision in its owning term.
// A later correction, deletion or re-order cannot change its initial position.
func (u *unit) initialRecordModes(ctx context.Context, action string) ([]initialRecordMode, error) {
	args := []any{u.store.Instance}
	filter := ""
	if action != "" {
		args = append(args, action)
		filter = " AND a.id=$2"
	}
	rows, err := u.tx.Query(ctx, `WITH first_issue AS (
		SELECT a.instance_id,a.assignment_id,a.id,r.revision,r.operation,r.business_date,src.version,src.previous
		FROM staffing.employment_action a JOIN staffing.employment_action_revision r
			ON (r.instance_id,r.action_id,r.revision)=(a.instance_id,a.id,a.revision)
		LEFT JOIN staffing.employment_revision_publication p
			ON (p.instance_id,p.assignment_id,p.action_id,p.revision)=(a.instance_id,a.assignment_id,a.id,r.revision)
		LEFT JOIN staffing.assignment_source src ON (src.instance_id,src.assignment_id,src.version)=(p.instance_id,p.assignment_id,p.version)
		WHERE a.instance_id=$1 AND r.operation IN ('Add','Insert')
			AND a.kind NOT IN ('Hire','Rehire','AdditionalAssignment','ParallelEmployment','CancelHire')`+filter+`)
	SELECT f.id,f.revision,f.operation,position.effective FROM first_issue f
	CROSS JOIN LATERAL (
		WITH RECURSIVE before_issue AS (
			SELECT a.id,a.kind,r.business_date`+activeRecordMembers+`
			AND (m.instance_id,m.assignment_id,m.version)=(f.instance_id,f.assignment_id,f.previous)
		), same_day AS (
			SELECT a.id,r.revision,
				coalesce(r.adoption#>>'{previous,action}',r.adoption#>>'{probation,afterAction}') AS prior_id,
				coalesce(r.adoption#>>'{previous,revision}',r.adoption#>>'{probation,afterRevision}') AS prior_revision`+activeRecordMembers+`
			AND (m.instance_id,m.assignment_id,m.version)=(f.instance_id,f.assignment_id,f.version)
			AND r.business_date=f.business_date
		), successors(id,revision) AS (
			SELECT f.id,f.revision UNION
			SELECT s.id,s.revision FROM successors p JOIN same_day s
				ON (s.prior_id,s.prior_revision)=(p.id::text,p.revision::text)
		)
		SELECT CASE
			WHEN NOT EXISTS(SELECT FROM before_issue WHERE kind IN ('Hire','Rehire','AdditionalAssignment','ParallelEmployment')) THEN ''
			WHEN EXISTS(SELECT FROM before_issue WHERE business_date>f.business_date) THEN 'Insert'
			WHEN EXISTS(SELECT FROM before_issue b JOIN successors s ON s.id=b.id
				WHERE b.business_date=f.business_date AND b.id<>f.id) THEN 'Insert'
			WHEN f.operation='Insert' AND EXISTS(SELECT FROM before_issue WHERE business_date=f.business_date
				AND kind NOT IN ('Hire','Rehire','AdditionalAssignment','ParallelEmployment')) THEN ''
			ELSE 'Add' END AS effective
	) position ORDER BY f.id`, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[initialRecordMode])
}

func (u *unit) initialPosition(ctx context.Context, a h.Action) (*h.InitialRecordPosition, error) {
	if h.StartsAssignment(a.Kind) || a.Kind == "CancelHire" {
		return nil, nil
	}
	modes, err := u.initialRecordModes(ctx, a.ID)
	if err != nil || len(modes) == 0 || modes[0].Recorded == modes[0].Effective {
		return nil, err
	}
	mode := modes[0]
	visible, err := u.initialModeVisible(ctx, a, mode.Revision)
	if err != nil || !visible {
		return nil, err
	}
	return &h.InitialRecordPosition{Recorded: mode.Recorded, Effective: mode.Effective}, nil
}

func (u *unit) initialModeVisible(ctx context.Context, a h.Action, revision string) (bool, error) {
	visible, err := u.revisionVisible(ctx, a, revision)
	if !errors.Is(err, pgx.ErrNoRows) {
		return visible, err
	}
	// Without the initial publication we cannot establish a department scope.
	for _, right := range []string{"read", "history"} {
		ok, err := u.employmentAllowed(ctx, right, a.Person, a.Employer, "", false)
		if err != nil || !ok {
			return false, err
		}
	}
	return true, nil
}

func (u *unit) retainedRecordModes(ctx context.Context, report *h.ProbationRepairReport) error {
	modes, err := u.initialRecordModes(ctx, "")
	if err != nil {
		return err
	}
	for _, mode := range modes {
		a, err := u.action(ctx, mode.Action)
		if err != nil {
			return err
		}
		current, err := u.actionDisclosure(ctx, a, true)
		if err != nil {
			return err
		}
		if !current.from && !current.to {
			continue
		}
		visible, err := u.initialModeVisible(ctx, a, mode.Revision)
		if err != nil {
			return err
		}
		if !visible {
			continue
		}
		report.Scanned++
		if mode.Recorded == mode.Effective {
			continue
		}
		item := h.ProbationRepairItem{Action: mode.Action, Result: "explained"}
		switch mode.Effective {
		case "Add":
			item.Reason = "原登记为插入，首次发布前的履历位置为末端新增"
		case "Insert":
			item.Reason = "原登记为新增，首次发布前的履历位置为历史插入"
		default:
			item.Result = "unresolved"
			item.Reason = "首次登记的前版来源或同日准确后项不足，请核对当时的办理原件"
			report.Unresolved++
		}
		report.Differences++
		report.Items = append(report.Items, item)
	}
	return nil
}
