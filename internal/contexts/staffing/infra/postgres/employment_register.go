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
		slices.Contains([]string{"", "Hire", "DataChange"}, q.Kind) && slices.Contains([]string{"", "effective", "future", "cancelled"}, q.Status) &&
		slices.Contains([]string{"", "recordedAt", "-recordedAt", "businessDate", "-businessDate"}, q.Sort) &&
		q.Page >= 1 && q.Page <= 1000000 && slices.Contains([]int{20, 50, 100}, q.PageSize)
}

type registerCandidate struct {
	h.RegisterRow
	actor, employer string
	recorded        time.Time
	cancelled       bool
}

// The current source selects the adopted revision. The source which published that
// revision supplies the department as recorded then, even if the hire was cancelled later.
const registerSQL = `SELECT a.id,r.revision,src.version,a.kind,r.business_date::text,a.recorded_at,
    e.id,t.id,t.record_number,e.person_id,e.employer_id,t.department_id,
    coalesce(seg.adoption->>'departmentName',''),a.actor,
    coalesce(nullif(first.operator_snapshot->>'displayName',''),nullif(first.operator_snapshot->>'login',''),a.actor::text),
    EXISTS (SELECT FROM staffing.employment_action cancelled WHERE cancelled.instance_id=a.instance_id AND cancelled.original_id=a.id AND cancelled.kind='CancelHire')
    FROM staffing.employment_action a JOIN staffing.employment e ON (e.instance_id,e.id)=(a.instance_id,a.employment_id)
    JOIN staffing.assignment t ON (t.instance_id,t.id)=(a.instance_id,a.assignment_id)
    JOIN staffing.assignment_head h ON (h.instance_id,h.assignment_id)=(a.instance_id,a.assignment_id) ` + assignmentRevisionJoin + `
    JOIN staffing.employment_action_revision first ON (first.instance_id,first.action_id,first.revision)=(a.instance_id,a.id,a.revision)
    JOIN staffing.assignment_source src ON (src.instance_id,src.action_id,src.revision)=(r.instance_id,r.action_id,r.revision)
    LEFT JOIN staffing.assignment_segment seg ON (seg.instance_id,seg.assignment_id,seg.version)=(src.instance_id,src.assignment_id,src.version) AND seg.period @> r.business_date
    WHERE a.instance_id=$1 AND a.kind<>'CancelHire'
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
			e = row.Scan(&r.ID, &r.Revision, &r.Source, &r.Kind, &r.BusinessDate, &r.recorded, &r.Employment, &r.Assignment, &r.RecordNumber, &r.Person, &r.employer, &r.Department, &r.DepartmentName, &r.actor, &r.Operator, &r.cancelled)
			return
		})
		if e != nil {
			return e
		}
		var allowed []registerCandidate
		var ids []string
		today := u.now.In(f.Shanghai).Format(time.DateOnly)
		for _, r := range candidates {
			visible := func(action string, fields ...string) (bool, error) {
				return u.employmentAllowed(ctx, action, r.Person, r.employer, r.Department, false, fields...)
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
			r.CanHistory, e = visible("history")
			if e != nil {
				return e
			}
			showOperator, e := visible("register.read", "operator")
			if e != nil {
				return e
			}
			showDept, e := visible("register.read", "department")
			if e != nil {
				return e
			}
			if !showDept {
				r.Department, r.DepartmentName = "", ""
			}
			if !showOperator {
				r.Operator, r.actor = "", ""
			}
			r.Status = "effective"
			if r.cancelled {
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
			if r.actor != "" {
				operators[r.actor] = r.Operator
			}
			if (q.Department != "" && r.Department != q.Department) || (q.Operator != "" && r.actor != q.Operator) || (q.Status != "" && r.Status != q.Status) {
				continue
			}
			// Hire and location changes have no transfer sides.
			if q.FromDepartment != "" || q.ToDepartment != "" {
				continue
			}
			matched = append(matched, r.RegisterRow)
		}
		for id, name := range departments {
			out.Departments = append(out.Departments, h.EmployeeDepartment{ID: id, Name: name})
		}
		slices.SortFunc(out.Departments, func(a, b h.EmployeeDepartment) int {
			return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
		})
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
