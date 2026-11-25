package postgres

import (
	"context"
	"errors"
	"reflect"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

// The relation owns one decision. Its immutable revisions belong to the
// original assignment even when a different assignment triggers a replay.
func (u *unit) currentConfirmation(ctx context.Context, a h.Action) (r savedRecord, err error) {
	err = u.tx.QueryRow(ctx, `SELECT act.id,act.assignment_id,rv.revision,rv.business_date::text,rv.notice,rv.operation,rv.adoption
		FROM staffing.employment_action act JOIN staffing.assignment_head hd ON (hd.instance_id,hd.assignment_id)=(act.instance_id,act.assignment_id)
		JOIN staffing.assignment_member m ON (m.instance_id,m.assignment_id,m.version,m.action_id)=(hd.instance_id,hd.assignment_id,hd.version,act.id)
		JOIN staffing.employment_action_revision rv ON (rv.instance_id,rv.action_id,rv.revision)=(m.instance_id,m.action_id,m.revision)
		WHERE act.instance_id=$1 AND act.employment_id=$2 AND act.kind='ConfirmEmployment'`, u.store.Instance, a.Employment).Scan(&r.ID, &r.Owner, &r.Revision, &r.Day, &r.Notice, &r.Operation, &r.Evidence)
	if errors.Is(err, pgx.ErrNoRows) {
		return savedRecord{}, nil
	}
	if err != nil {
		return r, err
	}
	r.Kind = "ConfirmEmployment"
	r.Previous = r.Evidence.Previous
	if ev := r.Evidence.Probation; ev != nil && ev.AfterAction != "" {
		r.Previous = &h.ActionRef{Action: ev.AfterAction, Revision: ev.AfterRevision}
	}
	return r, nil
}

func (u *unit) confirmScope(ctx context.Context, a h.Action, department string) error {
	// The ingress assignment and any explicitly named evidence are checked by
	// their readers. An inferred department only scopes the relationship grant.
	if department == "" {
		global, err := u.employmentAllowed(ctx, "probation.confirm", a.Person, a.Employer, "", false)
		if err != nil {
			return err
		}
		if !global {
			return f.Fail(409, "PROBATION_SCOPE", "办理授权需要核定发生部门，请补足通知或实际先后")
		}
	}
	return u.requireEmployment(ctx, "probation.confirm", a.Person, a.Employer, department)
}

// Calculate the occurrence from the proposed placement, including both lawful
// before-images when a transfer and a confirmation have no documented order.
func (u *unit) replayAttribution(ctx context.Context, a h.Action, r *savedRecord, old savedRecord, change h.HistoryChange, records []savedRecord) error {
	ev := *r.Evidence.Probation
	departments, err := u.relationDepartments(ctx, a.Employment, r.Day)
	if err != nil {
		return err
	}
	main := false
	eligible := map[string]string{}
	for _, d := range departments {
		if d.Assignment == a.Assignment {
			main = main || d.Primary
		} else {
			eligible[d.ID] = d.Name
		}
	}
	eligible[change.Before.Department] = change.Before.Adoption.DepartmentName
	for _, row := range records {
		if row.Transfer == nil || row.Removed || row.Day != r.Day {
			continue
		}
		for _, seg := range row.Transfer.Segments {
			if seg.Includes(r.Day) {
				eligible[seg.DepartmentID] = seg.DepartmentName
			}
		}
	}
	if r.Previous == nil {
		ev.AfterAction, ev.AfterRevision = "", ""
	} else if ev.AfterAction != "" {
		ev.AfterAction, ev.AfterRevision = r.Previous.Action, r.Previous.Revision
	}
	if ev.Attribution == "notice" {
		name, ok := eligible[ev.Department]
		if !ok {
			return f.Fail(409, "PROBATION_DEPARTMENT", "拟保存的安排与转正通知记载的部门不符")
		}
		ev.DepartmentName = name
	} else if main {
		ev.Department, ev.DepartmentName = change.Before.Department, change.Before.Adoption.DepartmentName
		ev.Attribution = "relationship-primary"
		if change.Alternate != nil && change.Alternate.Department != ev.Department {
			ev.Department, ev.DepartmentName, ev.Attribution = "", "", "same-day-order-undetermined"
		} else if ev.AfterAction != "" {
			ev.Attribution = "preceding-transfer"
		}
		if err := u.adoptConfirmationDepartment(ctx, a, r, old, &ev, change.Before); err != nil {
			return err
		}
	}
	r.Evidence.Probation = &ev
	if old.Evidence.Probation == nil || !reflect.DeepEqual(old.Evidence.Probation, &ev) || !reflect.DeepEqual(old.Previous, r.Previous) {
		if prior := old.Evidence.Probation; prior != nil {
			if err := u.confirmScope(ctx, a, prior.Department); err != nil {
				return err
			}
			if old.Previous != nil && r.Previous == nil {
				r.Evidence.Reason = "原前序已删除或改期，同日先后已解除"
			} else {
				r.Evidence.Reason = "按当前任职安排复核转正通知与发生部门"
			}
		}
		return u.confirmScope(ctx, a, ev.Department)
	}
	return nil
}

// Older notices can identify the adopted organization through the primary
// assignment edition that first published their precise relation revision.
func (u *unit) adoptConfirmationDepartment(ctx context.Context, a h.Action, row *savedRecord, old savedRecord, ev *h.ProbationEvidence, before h.Placement) error {
	if ev.Department == "" {
		ev.DepartmentSource, ev.DepartmentRevision = "", ""
		return nil
	}
	source, revision := before.Adoption.DepartmentSource, before.Adoption.DepartmentRevision
	prior := old.Evidence.Probation
	if prior != nil && prior.Department == ev.Department && prior.DepartmentSource == "" && prior.DepartmentRevision == "" {
		var same *bool
		err := u.tx.QueryRow(ctx, `SELECT (SELECT (s.adoption->>'departmentSource',s.adoption->>'departmentRevision')=($7::text,$8::text)
			FROM staffing.assignment_source src JOIN staffing.assignment_member m USING(instance_id,assignment_id,version)
			JOIN staffing.assignment_segment s USING(instance_id,assignment_id,version)
			WHERE m.instance_id=$1 AND m.assignment_id=$2 AND m.action_id=$3 AND m.revision=$4
			AND NOT EXISTS(SELECT FROM staffing.assignment_member prev WHERE (prev.instance_id,prev.assignment_id,prev.version,prev.action_id,prev.revision)=(m.instance_id,m.assignment_id,src.previous,m.action_id,m.revision))
			AND s.period @> $5::date AND s.department_id=$6)`, u.store.Instance, a.Assignment, row.ID, old.Revision, row.Day, ev.Department, source, revision).Scan(&same)
		if err != nil {
			return err
		}
		if same == nil || *same {
			return nil
		}
	}
	ev.DepartmentSource, ev.DepartmentRevision = source, revision
	return nil
}

func (u *unit) publishRelationMembers(ctx context.Context, a h.Action, action, revision, skip string) error {
	rows, err := u.tx.Query(ctx, `SELECT t.id,hd.version,EXISTS(SELECT FROM staffing.assignment_member m JOIN staffing.employment_action_revision r USING(instance_id,action_id,revision)
		WHERE (m.instance_id,m.assignment_id,m.version)=(hd.instance_id,hd.assignment_id,hd.version)
		AND r.operation NOT IN ('Delete','Withdraw') AND r.adoption#>>'{previous,action}'=$3)
		FROM staffing.assignment t JOIN staffing.assignment_head hd ON (hd.instance_id,hd.assignment_id)=(t.instance_id,t.id)
		WHERE t.instance_id=$1 AND t.employment_id=$2 AND t.id IS DISTINCT FROM nullif($4,'')::uuid ORDER BY t.id`, u.store.Instance, a.Employment, action, skip)
	if err != nil {
		return err
	}
	terms, err := pgx.CollectRows(rows, pgx.RowToStructByPos[struct {
		ID, Version string
		Linked      bool
	}])
	if err != nil {
		return err
	}
	for _, term := range terms {
		if !term.Linked {
			if err = u.lifecycleSource(ctx, term.ID, term.Version, action, revision, ""); err != nil {
				return err
			}
			continue
		}
		peer, err := u.assignmentIdentity(ctx, term.ID)
		if err != nil {
			return err
		}
		_, records, err := u.historyRows(ctx, peer)
		if err != nil {
			return err
		}
		entries := make([]h.HistoryEntry, 0, len(records))
		byID := map[string]savedRecord{}
		for _, row := range records {
			entries = append(entries, row.HistoryEntry)
			byID[row.ID] = row
		}
		ordered, err := h.OrderDay(entries)
		if err != nil {
			return err
		}
		changed := map[string]string{action: revision}
		refs := []h.ActionRef{{Action: action, Revision: revision}}
		for _, entry := range ordered {
			row := byID[entry.ID]
			if row.Removed || row.Previous == nil {
				continue
			}
			next := changed[row.Previous.Action]
			if next == "" || next == row.Previous.Revision {
				continue
			}
			if err = u.authorizeSuccessor(ctx, peer, row); err != nil {
				return err
			}
			prior := row.Revision
			row.Operation, row.Revision = "Rebase", f.NewID()
			row.Previous = &h.ActionRef{Action: row.Previous.Action, Revision: next}
			if err = u.appendRecordRevision(ctx, term.ID, row, prior); err != nil {
				return err
			}
			if _, err = u.tx.Exec(ctx, `INSERT INTO staffing.employment_outbox(instance_id,event_id,action_id,revision) VALUES($1,$2,$3,$4)`, u.store.Instance, f.NewID(), row.ID, row.Revision); err != nil {
				return err
			}
			changed[row.ID] = row.Revision
			refs = append(refs, h.ActionRef{Action: row.ID, Revision: row.Revision})
		}
		if err = u.cloneAssignmentSource(ctx, term.ID, term.Version, action, revision, "", refs); err != nil {
			return err
		}
	}
	return nil
}

func (u *unit) authorizeSuccessor(ctx context.Context, a h.Action, row savedRecord) error {
	var departments []string
	permit := "location.correct"
	if row.Evidence.Promotion != nil {
		permit = "promotion"
		if row.Kind == "GradeRecord" {
			permit = "grade.record"
		}
		departments = append(departments, row.Evidence.Promotion.Department)
	}
	if row.Evidence.Transfer != nil {
		if row.Kind == "Transfer" {
			permit = "transfer"
		}
		departments = append(departments, row.Evidence.Transfer.BeforeDepartment, row.Evidence.Transfer.After.Department)
	}
	if len(departments) == 0 {
		scope, err := u.assignmentAt(ctx, a.Assignment, row.Day, "")
		if err != nil {
			return err
		}
		departments = append(departments, scope.Department)
	}
	for _, dept := range departments {
		for _, right := range []string{"read", permit} {
			if err := u.requireEmployment(ctx, right, a.Person, a.Employer, dept); err != nil {
				return err
			}
		}
	}
	return nil
}
