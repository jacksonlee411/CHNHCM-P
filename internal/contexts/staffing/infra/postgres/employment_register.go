package postgres

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	person "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CanListRegister(ctx context.Context, c f.Credentials) (yes bool, err error) {
	err = s.within(ctx, c, false, true, func(u *unit) error {
		var e error
		yes, e = u.employmentAllowed(ctx, "register.read", "", "", "", true)
		return e
	})
	return
}

func validRegisterQuery(q h.RegisterQuery) bool {
	for _, v := range []string{q.Ref, q.Department, q.FromDepartment, q.ToDepartment, q.Operator} {
		if v != "" && !f.ValidID(v) {
			return false
		}
	}
	for _, v := range []string{q.BusinessFrom, q.BusinessTo, q.RecordedFrom, q.RecordedTo} {
		if v != "" && !f.ValidDate(v) {
			return false
		}
	}
	return (q.BusinessTo == "" || q.BusinessFrom <= q.BusinessTo) && (q.RecordedTo == "" || q.RecordedFrom <= q.RecordedTo) &&
		slices.Contains([]string{"", "Hire", "DataChange", "Transfer", "AdditionalAssignment", "ParallelEmployment", "ProbationStart", "ConfirmEmployment", "Promotion", "GradeRecord", "SetManager", "EndAssignment", "Leave", "Rehire"}, q.Kind) && slices.Contains([]string{"", "effective", "future", "cancelled", "deleted"}, q.Status) &&
		slices.Contains([]string{"", "recordedAt", "-recordedAt", "businessDate", "-businessDate"}, q.Sort) &&
		q.Page >= 1 && q.Page <= 1000000 && slices.Contains([]int{20, 50, 100}, q.PageSize)
}

type registerCandidate struct {
	h.RegisterRow
	actor, employer    string
	recorded           time.Time
	cancelled, removed bool
}

const exitDepartmentJoin = `    LEFT JOIN LATERAL (
      SELECT s.adoption->>'departmentName' AS name FROM staffing.assignment_source src
      JOIN staffing.assignment_segment s ON (s.instance_id,s.assignment_id)=(src.instance_id,src.assignment_id)
        AND s.version IN (src.version,src.previous)
      WHERE (src.instance_id,src.assignment_id,src.version)=(issued.instance_id,issued.assignment_id,issued.version)
        AND s.department_id=(r.adoption->>'department')::uuid AND s.period @> ((r.adoption#>>'{exit,firstInactive}')::date-1)
      ORDER BY (s.version=src.version) DESC LIMIT 1) exit_department ON a.kind='Leave'`

// The current source selects the adopted revision. The source which published that
// revision supplies the department as recorded then, even if the hire was cancelled later.
const registerSQL = `SELECT a.id,r.revision,issued.version,a.kind,r.business_date::text,a.recorded_at,
    e.id,t.id,t.record_number,e.person_id,e.employer_id,
    CASE a.kind WHEN 'Leave' THEN coalesce(r.adoption->>'department','')
      WHEN 'ConfirmEmployment' THEN coalesce(r.adoption#>>'{probation,department}','') WHEN 'ProbationStart' THEN ''
      ELSE coalesce(r.adoption#>>'{promotion,department}',origin.department_id::text,t.department_id::text) END,
    CASE a.kind WHEN 'Leave' THEN coalesce(exit_department.name,'')
      WHEN 'ConfirmEmployment' THEN coalesce(r.adoption#>>'{probation,departmentName}','') WHEN 'ProbationStart' THEN ''
      ELSE coalesce(r.adoption#>>'{promotion,departmentName}',origin.adoption->>'departmentName','') END,a.actor,
    coalesce(nullif(first.operator_snapshot->>'displayName',''),nullif(first.operator_snapshot->>'login',''),a.actor::text),
    EXISTS (SELECT FROM staffing.employment_action cancelled WHERE cancelled.instance_id=a.instance_id AND cancelled.original_id=a.id AND cancelled.kind='CancelHire') OR r.operation='Withdraw',
    r.operation='Delete',coalesce(r.adoption#>>'{transfer,beforeDepartment}',''),coalesce(r.adoption#>>'{transfer,beforeDepartmentName}',''),
    coalesce(r.adoption#>>'{transfer,after,department}',''),coalesce(r.adoption#>>'{transfer,after,segments,0,departmentName}','')
    FROM staffing.employment_action a JOIN staffing.employment e ON (e.instance_id,e.id)=(a.instance_id,a.employment_id)
    JOIN staffing.assignment t ON (t.instance_id,t.id)=(a.instance_id,a.assignment_id)
    JOIN staffing.assignment_head h ON (h.instance_id,h.assignment_id)=(a.instance_id,a.assignment_id) ` + assignmentRevisionJoin + `
    JOIN staffing.employment_action_revision first ON (first.instance_id,first.action_id,first.revision)=(a.instance_id,a.id,a.revision)
` + exitDepartmentJoin + `
    WHERE a.instance_id=$1 AND a.kind<>'CancelHire' AND (a.kind<>'SetManager' OR nullif(r.adoption->'managerExit','null'::jsonb) IS NULL)
    AND (NULLIF($2,'')::date IS NULL OR r.business_date>=NULLIF($2,'')::date) AND (NULLIF($3,'')::date IS NULL OR r.business_date<=NULLIF($3,'')::date)
    AND (NULLIF($4,'')::date IS NULL OR a.recorded_at >= (NULLIF($4,'')::date::timestamp AT TIME ZONE 'Asia/Shanghai'))
    AND (NULLIF($5,'')::date IS NULL OR a.recorded_at < ((NULLIF($5,'')::date+1)::timestamp AT TIME ZONE 'Asia/Shanghai'))
    AND ($6='' OR a.kind=$6)`

func (s *Store) ListRegister(ctx context.Context, c f.Credentials, q h.RegisterQuery) (out h.RegisterResult, err error) {
	if !validRegisterQuery(q) {
		return out, f.Fail(400, "REGISTER_QUERY", "请核对办理类型、日期范围和查询条件")
	}
	out = h.RegisterResult{Rows: []h.RegisterRow{}, Departments: []h.EmployeeDepartment{}, FromDepartments: []h.EmployeeDepartment{}, ToDepartments: []h.EmployeeDepartment{}, Operators: []h.RegisterOperator{}}
	err = s.within(ctx, c, false, true, func(u *unit) error {
		ok, e := u.employmentAllowed(ctx, "register.read", "", "", "", true)
		if e != nil {
			return e
		}
		if !ok {
			return f.Fail(403, "REGISTER_DENIED", "当前权限不允许查询办理记录")
		}
		rows, e := u.tx.Query(ctx, registerSQL, s.Instance, q.BusinessFrom, q.BusinessTo, q.RecordedFrom, q.RecordedTo, q.Kind)
		if e != nil {
			return e
		}
		candidates, e := pgx.CollectRows(rows, func(row pgx.CollectableRow) (r registerCandidate, e error) {
			e = row.Scan(&r.ID, &r.Revision, &r.Source, &r.Kind, &r.BusinessDate, &r.recorded, &r.Employment, &r.Assignment, &r.RecordNumber, &r.Person, &r.employer, &r.Department, &r.DepartmentName, &r.actor, &r.Operator, &r.cancelled, &r.removed, &r.FromDepartment, &r.FromDepartmentName, &r.ToDepartment, &r.ToDepartmentName)
			return
		})
		if e != nil {
			return e
		}
		var allowed []registerCandidate
		var ids []string
		today := u.now.In(f.Shanghai).Format(time.DateOnly)
		for _, r := range candidates {
			// Relation rows have one canonical business ID even when the operator can
			// follow that decision through more than one authorized assignment.
			if h.RelationAction(r.Kind) {
				a, e := u.action(ctx, r.ID)
				if e != nil {
					return e
				}
				scoped, visible, e := u.relationVisibleAssignment(ctx, a, "register.read")
				if e != nil {
					return e
				}
				if !visible {
					continue
				}
				r.Relation, e = u.relationshipSummary(ctx, a, scoped, false)
				if e != nil {
					return e
				}

				r.Assignment, r.RecordNumber, r.Source = scoped.Assignment, scoped.RecordNumber, scoped.SourceVersion
				_, r.CanHistory, e = u.relationVisibleAssignment(ctx, a, "history")
				if e != nil {
					return e
				}
				showOperator, e := u.employmentAllowed(ctx, "register.read", r.Person, r.employer, scoped.Department, false, "operator")
				if e != nil {
					return e
				}
				if !showOperator {
					r.Operator, r.actor = "", ""
				}
				if r.Department != "" {
					read, e := u.employmentAllowed(ctx, "read", r.Person, r.employer, r.Department, false)
					if e != nil {
						return e
					}
					field, e := u.employmentAllowed(ctx, "register.read", r.Person, r.employer, r.Department, false, "department")
					if e != nil {
						return e
					}
					if !read || !field {
						r.Department, r.DepartmentName = "", ""
					}
				}
			} else {

				type side struct {
					id, name  string
					direction string
				}
				sides := []side{{r.Department, r.DepartmentName, ""}}
				if r.Kind == "Transfer" {
					sides = []side{{r.FromDepartment, r.FromDepartmentName, "from"}, {r.ToDepartment, r.ToDepartmentName, "to"}}
				}
				r.Department, r.DepartmentName, r.FromDepartment, r.FromDepartmentName, r.ToDepartment, r.ToDepartmentName = "", "", "", "", "", ""
				granted, showOperator := false, false
				for _, side := range sides {
					if side.id == "" && r.Kind != "Promotion" {
						continue
					}
					visible := func(action string, fields ...string) (bool, error) {
						return u.employmentAllowed(ctx, action, r.Person, r.employer, side.id, false, fields...)
					}
					entry, e := visible("register.read")
					if e != nil {
						return e
					}
					read, e := visible("read")
					if e != nil {
						return e
					}
					if !entry || !read {
						continue
					}
					granted = true
					history, e := visible("history")
					if e != nil {
						return e
					}
					r.CanHistory = r.CanHistory || history
					operator, e := visible("register.read", "operator")
					if e != nil {
						return e
					}
					showOperator = showOperator || operator
					dept, e := visible("register.read", "department")
					if e != nil {
						return e
					}
					if !dept {
						continue
					}
					r.Department, r.DepartmentName = side.id, side.name
					switch side.direction {
					case "from":
						r.FromDepartment, r.FromDepartmentName = side.id, side.name
					case "to":
						r.ToDepartment, r.ToDepartmentName = side.id, side.name
					}
				}
				if !granted {
					continue
				}
				if !showOperator {
					r.Operator, r.actor = "", ""
				}
			}
			r.Status = "effective"
			if r.removed {
				r.Status = "deleted"
			} else if r.cancelled {
				r.Status = "cancelled"
			} else if r.BusinessDate > today {
				r.Status = "future"
			}
			r.RecordedAt = r.recorded.UTC().Format("2006-01-02T15:04:05.000000Z")
			allowed = append(allowed, r)
			ids = append(ids, r.Person)
		}
		people, keyword, e := s.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), c, ids, q.Ref)
		if e != nil {
			return e
		}
		out.Keyword = keyword
		names := map[string]person.Summary{}
		for _, p := range people {
			names[p.ID] = p
		}
		departments := map[string]string{}
		fromDepts, toDepts := map[string]string{}, map[string]string{}
		operators := map[string]string{}
		var matched []h.RegisterRow
		for _, r := range allowed {
			identity, ok := names[r.Person]
			if !ok {
				continue
			}
			r.Number, r.Name = identity.Number, identity.Name
			if r.Department != "" {
				departments[r.Department] = r.DepartmentName
			}
			if r.FromDepartment != "" {
				fromDepts[r.FromDepartment] = r.FromDepartmentName
				departments[r.FromDepartment] = r.FromDepartmentName
			}
			if r.ToDepartment != "" {
				toDepts[r.ToDepartment] = r.ToDepartmentName
				departments[r.ToDepartment] = r.ToDepartmentName
			}
			if r.actor != "" {
				operators[r.actor] = r.Operator
			}
			if (q.Department != "" && r.Department != q.Department && r.FromDepartment != q.Department && r.ToDepartment != q.Department) || (q.Operator != "" && r.actor != q.Operator) || (q.Status != "" && r.Status != q.Status) {
				continue
			}
			if (q.FromDepartment != "" && r.FromDepartment != q.FromDepartment) || (q.ToDepartment != "" && r.ToDepartment != q.ToDepartment) {
				continue
			}
			matched = append(matched, r.RegisterRow)
		}
		choices := func(names map[string]string) []h.EmployeeDepartment {
			result := make([]h.EmployeeDepartment, 0, len(names))
			for id, name := range names {
				result = append(result, h.EmployeeDepartment{ID: id, Name: name})
			}
			slices.SortFunc(result, func(a, b h.EmployeeDepartment) int {
				return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
			})
			return result
		}
		out.Departments, out.FromDepartments, out.ToDepartments = choices(departments), choices(fromDepts), choices(toDepts)
		for id, name := range operators {
			out.Operators = append(out.Operators, h.RegisterOperator{ID: id, Name: name})
		}
		slices.SortFunc(out.Operators, func(a, b h.RegisterOperator) int { return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID)) })
		slices.SortFunc(matched, func(a, b h.RegisterRow) int {
			order := cmp.Compare(a.RecordedAt, b.RecordedAt)
			if strings.TrimPrefix(q.Sort, "-") == "businessDate" {
				order = cmp.Compare(a.BusinessDate, b.BusinessDate)
			}
			if q.Sort == "" || strings.HasPrefix(q.Sort, "-") {
				order = -order
			}
			return cmp.Or(order, cmp.Compare(a.ID, b.ID))
		})
		out.Total = len(matched)
		start := min((q.Page-1)*q.PageSize, out.Total)
		out.Rows = append(out.Rows, matched[start:min(start+q.PageSize, out.Total)]...)
		return nil
	})
	return
}
