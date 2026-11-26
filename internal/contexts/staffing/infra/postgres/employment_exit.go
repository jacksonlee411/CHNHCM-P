package postgres

import (
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"context"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
	"unicode/utf8"
)

func (st *Store) ExitAssignment(ctx context.Context, c f.Credentials, id, key string, leave bool, in h.ExitCommand) (h.Receipt, error) {
	return st.hireCommand(ctx, c, key, []any{"ExitAssignment", id, leave, in}, func(u *unit) (string, error) {
		exit, e := in.Boundary()
		if e != nil {
			return "", e
		}
		if !f.ValidID(in.Expected) || strings.TrimSpace(in.Notice) == "" || utf8.RuneCountInString(in.Notice) > 1000 {
			return "", f.Fail(400, "EXIT_INPUT", "请核对任职和退出依据")
		}
		boundary, _ := time.Parse(time.DateOnly, exit.FirstInactive)
		last := boundary.AddDate(0, 0, -1).Format(time.DateOnly)
		if e = u.lockReporting(ctx); e != nil {
			return "", e
		}
		anchor, e := u.assignmentIdentity(ctx, id)
		if e != nil {
			return "", e
		}
		if e = lock(ctx, u.tx, false, "staffing.hire.person", st.Instance, anchor.Person); e != nil {
			return "", e
		}
		anchor, e = u.assignmentAt(ctx, id, last, "")
		if e != nil {
			return "", e
		}
		if anchor.Cancelled || !anchor.Includes(last) || anchor.SourceVersion != in.Expected {
			return "", assignmentConflict()
		}
		kind, permit := "EndAssignment", "assignment.end"
		if leave {
			kind, permit = "Leave", "employment.leave"
		}
		if e = u.requireEmployment(ctx, permit, anchor.Person, anchor.Employer, anchor.Department); e != nil {
			return "", e
		}
		var primary, unresolved bool
		e = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.employment_primary WHERE instance_id=$1 AND layer='assignment' AND target_id=$2 AND period @> $3::date),
  EXISTS(SELECT FROM staffing.assignment t JOIN staffing.assignment_head h ON (h.instance_id,h.assignment_id)=(t.instance_id,t.id) JOIN staffing.assignment_segment s ON (s.instance_id,s.assignment_id,s.version)=(h.instance_id,h.assignment_id,h.version)
    WHERE t.instance_id=$1 AND t.person_id=$4 AND t.employment_id<>$5 AND s.period && daterange($6::date,NULL,'[)'))`,
			st.Instance, id, last, anchor.Person, anchor.Employment, exit.FirstInactive).Scan(&primary, &unresolved)
		if e != nil {
			return "", e
		}
		if !leave && (anchor.PrimaryJob || primary) {
			return "", f.Fail(409, "EXIT_PRIMARY", "该任职承担主选择，请从本工作关系办理离职")
		}
		if leave && unresolved {
			var personPrimary bool
			if e = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.employment_primary WHERE instance_id=$1 AND layer='employment' AND owner_id=$2 AND target_id=$3 AND period @> $4::date)`, st.Instance, anchor.Person, anchor.Employment, last).Scan(&personPrimary); e != nil {
				return "", e
			}
			if personPrimary {
				return "", f.Fail(409, "EXIT_SUCCESSOR", "仍有独立关系需要人事管理主选择，请先核对接续安排")
			}
		}
		rows, e := u.tx.Query(ctx, `SELECT t.id,h.version,lower(t.period)::text,coalesce(x.first_inactive::text,''),coalesce(upper(t.period)::text,'') FROM staffing.assignment t
   JOIN staffing.assignment_head h ON (h.instance_id,h.assignment_id)=(t.instance_id,t.id)
   LEFT JOIN staffing.assignment_exit x ON (x.instance_id,x.assignment_id)=(t.instance_id,t.id)
   WHERE t.instance_id=$1 AND t.employment_id=$2 AND ($3 OR t.id=$4) ORDER BY t.record_number`, st.Instance, anchor.Employment, leave, id)
		if e != nil {
			return "", e
		}
		type term struct{ ID, Version, Start, Exit, Until string }
		entries, e := pgx.CollectRows(rows, pgx.RowToStructByPos[term])
		if e != nil {
			return "", e
		}
		closing := []string{}
		for _, term := range entries {
			if term.Exit != "" {
				if term.Exit > exit.FirstInactive || term.ID == id {
					return "", f.Fail(409, "EXIT_EXISTS", "所选任职已有退出办理，请核对原件")
				}
				continue
			}
			if term.Until != "" && term.Until < exit.FirstInactive {
				continue
			}
			if term.Start >= exit.FirstInactive {
				return "", f.Fail(409, "EXIT_SUCCESSOR", "退出后仍有已确认的任职安排，请先处置")
			}
			state, e := u.assignmentAt(ctx, term.ID, last, "")
			if e != nil {
				return "", e
			}
			if e = u.requireEmployment(ctx, permit, state.Person, state.Employer, state.Department); e != nil {
				return "", e
			}
			if clipped, err := u.exitConflictsWithDuty(ctx, term.ID, exit.FirstInactive); err != nil {
				return "", err
			} else if clipped {
				return "", f.Fail(409, "EXIT_DUTY", "退出日期与已核实的履职日冲突，请核对退出日期或更正出勤原件")
			}
			var successor bool
			if e = u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM staffing.assignment_member m JOIN staffing.employment_action_revision r USING(instance_id,action_id,revision)
      JOIN staffing.employment_action a ON (a.instance_id,a.id)=(m.instance_id,m.action_id)
      WHERE m.instance_id=$1 AND m.assignment_id=$2 AND m.version=$3 AND r.business_date >= $4::date AND r.operation NOT IN ('Delete','Withdraw')
      AND ($5 OR a.kind NOT IN ('ProbationStart','ConfirmEmployment')))`, st.Instance, term.ID, term.Version, exit.FirstInactive, leave).Scan(&successor); e != nil {
				return "", e
			}
			if successor {
				return "", f.Fail(409, "EXIT_SUCCESSOR", "退出日期之后仍有有效办理，请先处置后继")
			}
			closing = append(closing, term.ID)
		}
		if len(closing) == 0 {
			return "", assignmentConflict()
		}
		if e = u.lockExitingPositions(ctx, closing, exit.FirstInactive); e != nil {
			return "", e
		}
		item := employmentEntry{id: f.NewID(), revision: f.NewID(), kind: kind, employment: anchor.Employment, assignment: id, date: in.Date, notice: strings.TrimSpace(in.Notice), adoption: map[string]any{"exit": exit, "department": anchor.Department}}
		if e = u.newEmploymentAction(ctx, item); e != nil {
			return "", e
		}
		if leave {
			if e = u.closeContracts(ctx, anchor, exit.FirstInactive, item.notice, item.id, in.Contracts); e != nil {
				return "", e
			}
		} else if len(in.Contracts) > 0 {
			return "", f.Fail(400, "CONTRACT_RELATION", "结束单份任职不处置关系合同")
		}
		if e = u.closeReporting(ctx, closing, exit.FirstInactive, item); e != nil {
			return "", e
		}
		if leave {
			if _, e = u.tx.Exec(ctx, `INSERT INTO staffing.employment_exit VALUES($1,$2,$3,$4,$5::date)`, st.Instance, anchor.Employment, item.id, item.revision, exit.FirstInactive); e != nil {
				return "", e
			}
		}
		for _, term := range closing {
			var version string
			if e = u.tx.QueryRow(ctx, `SELECT version FROM staffing.assignment_head WHERE instance_id=$1 AND assignment_id=$2`, st.Instance, term).Scan(&version); e != nil {
				return "", e
			}
			if _, e = u.tx.Exec(ctx, `INSERT INTO staffing.assignment_exit VALUES($1,$2,$3,$4,$5::date)`, st.Instance, term, item.id, item.revision, exit.FirstInactive); e != nil {
				return "", e
			}
			if e = u.lifecycleSource(ctx, term, version, item.id, item.revision, exit.FirstInactive); e != nil {
				return "", e
			}
		}
		if leave {
			if _, e = u.tx.Exec(ctx, `UPDATE staffing.employment_primary SET period=period * daterange(NULL,$3::date,'[)')
    WHERE instance_id=$1 AND ((layer='assignment' AND owner_id=$2) OR (layer='employment' AND target_id=$2)) AND lower(period)<$3::date AND period && daterange($3::date,NULL,'[)')`, st.Instance, anchor.Employment, exit.FirstInactive); e != nil {
				return "", e
			}
		}
		return item.id, nil
	})
}

func (u *unit) closeReporting(ctx context.Context, closing []string, day string, cause employmentEntry) error {
	var future bool
	if err := u.tx.QueryRow(ctx, currentReporting+`SELECT EXISTS(SELECT FROM reporting g JOIN staffing.assignment_head hd USING(instance_id,assignment_id)
        JOIN staffing.assignment_segment s USING(instance_id,assignment_id,version)
        WHERE g.manager_assignment=ANY($2::uuid[]) AND lower(g.period)>=$3::date AND g.effective && s.period)`, u.store.Instance, closing, day).Scan(&future); err != nil {
		return err
	}
	if future {
		return f.Fail(409, "MANAGER_SUCCESSOR", "经理退出后仍有已确定的汇报，请先核对接替安排")
	}
	rows, e := u.tx.Query(ctx, currentReporting+`SELECT g.assignment_id,g.purpose FROM reporting g
 JOIN staffing.assignment_head h USING(instance_id,assignment_id)
 WHERE g.manager_assignment=ANY($2::uuid[]) AND NOT(g.assignment_id=ANY($2::uuid[])) AND g.effective @> $3::date
 AND EXISTS(SELECT FROM staffing.assignment_segment s WHERE (s.instance_id,s.assignment_id,s.version)=(h.instance_id,h.assignment_id,h.version) AND s.period @> $3::date)
 ORDER BY g.assignment_id,g.purpose`, u.store.Instance, closing, day)
	if e != nil {
		return e
	}
	changes, e := pgx.CollectRows(rows, pgx.RowToStructByPos[struct{ Assignment, Purpose string }])
	if e != nil {
		return e
	}
	for _, edge := range changes {
		a, e := u.assignmentIdentity(ctx, edge.Assignment)
		if e != nil {
			return e
		}
		if e = lock(ctx, u.tx, false, "staffing.hire.person", u.store.Instance, a.Person); e != nil {
			return e
		}
		a, e = u.assignmentIdentity(ctx, edge.Assignment)
		if e != nil {
			return e
		}
		_, e = u.storeManagerFact(ctx, a, h.ManagerFact{Period: f.Period{From: day}, Purpose: edge.Purpose}, "经理任职结束", &h.ActionRef{Action: cause.id, Revision: cause.revision}, nil)
		if e != nil {
			return e
		}
	}
	return nil
}
