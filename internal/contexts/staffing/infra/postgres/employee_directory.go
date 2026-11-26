package postgres

import (
	"cmp"
	"context"
	"slices"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	person "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
)

func (s *Store) CanListAssignments(ctx context.Context, c f.Credentials) (yes bool, err error) {
	err = s.within(ctx, c, false, true, func(u *unit) error {
		var e error
		yes, e = u.employmentAllowed(ctx, "directory", "", "", "", true)
		return e
	})
	return
}

func (s *Store) ListAssignments(ctx context.Context, c f.Credentials, q h.AssignmentQuery) (h.EmployeeDirectory, error) {
	return s.directory(ctx, c, q, false, "")
}
func (s *Store) PersonnelDirectory(ctx context.Context, c f.Credentials, q h.PersonnelQuery) (h.PersonnelDirectory, error) {
	if !slices.Contains([]string{"", "primary", "secondary"}, q.Primary) {
		return h.PersonnelDirectory{}, f.Fail(400, "EMPLOYEE_QUERY", "主兼岗条件无效")
	}
	v, err := s.directory(ctx, c, h.AssignmentQuery{Anchor: q.Anchor, Department: q.Department, Ref: q.Ref, Page: q.Page, PageSize: q.PageSize, Sort: q.Sort}, true, q.Primary)
	if err != nil {
		return h.PersonnelDirectory{}, err
	}
	out := h.PersonnelDirectory{Rows: []h.PersonnelRow{}, Departments: v.Departments, Navigation: v.Navigation, People: v.People, Total: v.Total, Assignments: v.Assignments, Keyword: v.Keyword}
	for _, r := range v.Rows {
		row := h.PersonnelRow{ID: r.ID, Person: r.Person, Name: r.Name, Number: r.Number, Historical: r.Historical}
		if r.Employment != "" {
			row.Assignment = &r
			row.Future = r.From > v.BusinessAt
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}
func (s *Store) directory(ctx context.Context, c f.Credentials, q h.AssignmentQuery, personnel bool, primary string) (out h.EmployeeDirectory, err error) {
	if q.Anchor != "" && !f.ValidID(q.Anchor) || q.BusinessAt != "" && !f.ValidDate(q.BusinessAt) || q.Department != "" && !f.ValidID(q.Department) || q.Person != "" && !f.ValidID(q.Person) || q.Page < 1 || q.Page > 1000000 || !slices.Contains([]int{20, 50, 100}, q.PageSize) || !slices.Contains([]string{"", "name", "-name", "number", "-number"}, q.Sort) {
		return out, f.Fail(400, "EMPLOYEE_QUERY", "请核对日期、部门及页码")
	}
	if personnel && q.Ref != "" {
		if e := s.HirePeople.RenewDirectoryQuery(ctx, c, q.Ref); e != nil {
			return out, e
		}
	}
	out.Rows = []h.EmployeeRow{}
	out.Departments = []h.EmployeeDepartment{}
	err = s.within(ctx, c, false, true, func(u *unit) error {
		ok, e := u.employmentAllowed(ctx, "directory", "", "", "", true)
		if e != nil {
			return e
		}
		if !ok && !personnel {
			return f.Fail(403, "EMPLOYEE_DIRECTORY", "当前权限不允许查询员工任职")
		}
		out.BusinessAt = q.BusinessAt
		if out.BusinessAt == "" {
			out.BusinessAt = u.now.In(f.Shanghai).Format("2006-01-02")
		}
		rows, e := u.tx.Query(ctx, `SELECT t.id,t.person_id,t.employment_id,t.record_number,s.department_id,e.employer_id,s.adoption->>'departmentName',s.adoption->>'jobName',coalesce(s.adoption->>'positionName',''),lower(t.period)::text,coalesce((SELECT first_inactive::text FROM staffing.assignment_exit WHERE instance_id=t.instance_id AND assignment_id=t.id),upper(t.period)::text,'')
			FROM staffing.assignment_head head JOIN staffing.assignment_segment s USING(instance_id,assignment_id,version)
			JOIN staffing.assignment t ON (t.instance_id,t.id)=(s.instance_id,s.assignment_id)
			JOIN staffing.employment e ON (e.instance_id,e.id)=(t.instance_id,t.employment_id)
			WHERE head.instance_id=$1 AND $4 AND (s.period @> $2::date OR ($5 AND lower(t.period)>$2::date AND s.period @> lower(t.period)))
      AND ($3='' OR t.person_id=NULLIF($3,'')::uuid)
      ORDER BY t.person_id,t.record_number,lower(t.period),t.id`, s.Instance, out.BusinessAt, q.Person, ok, personnel)
		if e != nil {
			return e
		}
		type candidate struct {
			h.EmployeeRow
			employer string
		}
		var candidates []candidate
		for rows.Next() {
			var r candidate
			if e = rows.Scan(&r.ID, &r.Person, &r.Employment, &r.RecordNumber, &r.Department, &r.employer, &r.DepartmentName, &r.JobName, &r.PositionName, &r.From, &r.Until); e != nil {
				rows.Close()
				return e
			}
			r.PrimaryJob = r.RecordNumber == 0
			candidates = append(candidates, r)
		}
		rows.Close()
		if e = rows.Err(); e != nil {
			return e
		}
		allowed := []h.EmployeeRow{}
		var ids []string
		type recordKey struct {
			person string
			number int
		}
		selected := map[recordKey]bool{}
		for _, r := range candidates {
			key := recordKey{r.Person, r.RecordNumber}
			if personnel && selected[key] {
				continue
			}
			read, e := u.employmentAllowed(ctx, "read", r.Person, r.employer, r.Department, false)
			if e != nil {
				return e
			}
			if !read {
				continue
			}
			list, e := u.employmentAllowed(ctx, "directory", r.Person, r.employer, r.Department, false)
			if e != nil {
				return e
			}
			if !list {
				continue
			}
			selected[key] = true
			allowed = append(allowed, r.EmployeeRow)
			ids = append(ids, r.Person)
		}
		// When there are no visible assignment rows to merge, PostgreSQL can count
		// and page the scoped people directly, including old-number matches.
		if personnel && len(allowed) == 0 && primary == "" && q.Department == "" && q.Anchor == "" {
			page, err := s.HirePeople.DirectoryPage(dbtx.With(ctx, u.tx), c, person.Query{Ref: q.Ref, Page: q.Page, Size: q.PageSize, Sort: q.Sort})
			if err != nil {
				return err
			}
			out.People, out.Total, out.Keyword = page.Total, page.Total, page.Keyword
			for _, p := range page.Rows {
				out.Rows = append(out.Rows, h.EmployeeRow{ID: p.ID, Person: p.ID, Name: p.Name, Number: p.Number, Historical: p.Historical})
			}
			return nil
		}
		var people []person.Summary
		var k string
		if personnel {
			people, k, e = s.HirePeople.DirectoryPeople(dbtx.With(ctx, u.tx), c, q.Ref)
		} else {
			people, k, e = s.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), c, ids, q.Ref)
		}
		if e != nil {
			return e
		}
		out.Keyword = k
		identities := map[string]person.Summary{}
		for _, p := range people {
			identities[p.ID] = p
		}
		departments := map[string]h.EmployeeDepartment{}
		var matching []h.EmployeeRow
		represented := map[string]bool{}
		for _, r := range allowed {
			p, ok := identities[r.Person]
			if !ok {
				continue
			}
			r.Name, r.Number, r.Historical = p.Name, p.Number, p.Historical
			represented[r.Person] = true
			matching = append(matching, r)
			departments[r.Department] = h.EmployeeDepartment{ID: r.Department, Name: r.DepartmentName}
		}
		if personnel && primary == "" && q.Department == "" {
			for _, p := range people {
				if !represented[p.ID] {
					matching = append(matching, h.EmployeeRow{ID: p.ID, Person: p.ID, Name: p.Name, Number: p.Number, Historical: p.Historical})
				}
			}
		}
		var deptIDs []string
		for id := range departments {
			deptIDs = append(deptIDs, id)
		}
		sources, e := u.PositionSources(ctx, "departmentId", deptIDs, false)
		if e != nil {
			return e
		}
		for _, src := range sources {
			index := d.At(src.Timeline, out.BusinessAt)
			if index < 0 {
				continue
			}
			parent := src.Records[index].Data["parent"]
			if _, visible := departments[parent]; !visible {
				continue
			}
			show := true
			for _, name := range []string{"department.read", "department.history.read"} {
				decision, e := u.Access().Decide(ctx, access.Intent{EnterpriseID: src.Enterprise, DepartmentTarget: src.DepartmentTarget, Action: name, Kind: "object"})
				if e != nil {
					return e
				}
				if !decision.Allowed || !slices.Contains(decision.ReadFields, "parentDepartmentId") {
					show = false
					break
				}
			}
			if show {
				v := departments[src.ID]
				v.Parent = parent
				departments[src.ID] = v
			}
		}
		for _, dept := range departments {
			out.Departments = append(out.Departments, dept)
		}
		slices.SortFunc(out.Departments, func(a, b h.EmployeeDepartment) int {
			return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
		})
		kept := matching[:0]
		count := map[string]bool{}
		for _, r := range matching {
			if primary == "primary" && !r.PrimaryJob || primary == "secondary" && r.PrimaryJob {
				continue
			}
			if q.Department != "" {
				dep := r.Department
				for dep != "" && dep != q.Department {
					dep = departments[dep].Parent
				}
				if dep == "" {
					continue
				}
			}
			if r.Employment != "" {
				out.Assignments++
			}
			kept = append(kept, r)
			count[r.Person] = true
		}
		slices.SortFunc(kept, func(a, b h.EmployeeRow) int {
			order := cmp.Compare(a.Number, b.Number)
			if strings.TrimPrefix(q.Sort, "-") == "name" {
				order = cmp.Compare(a.Name, b.Name)
			}
			if strings.HasPrefix(q.Sort, "-") {
				order = -order
			}
			return cmp.Or(order, cmp.Compare(a.Person, b.Person), cmp.Compare(a.RecordNumber, b.RecordNumber), cmp.Compare(a.ID, b.ID))
		})
		out.People, out.Total = len(count), len(kept)
		if q.Anchor != "" {
			for index, row := range kept {
				if row.ID != q.Anchor {
					continue
				}
				nav := h.AssignmentNav{Position: index + 1}
				if index > 0 {
					r := kept[index-1]
					nav.Previous = directoryLink(r)
				}
				if index+1 < len(kept) {
					r := kept[index+1]
					nav.Next = directoryLink(r)
				}
				out.Navigation = &nav
				break
			}
		}
		from := min((q.Page-1)*q.PageSize, len(kept))
		out.Rows = append(out.Rows, kept[from:min(from+q.PageSize, len(kept))]...)
		return nil
	})
	return
}

func directoryLink(r h.EmployeeRow) *h.AssignmentLink {
	link := &h.AssignmentLink{Person: r.Person}
	if r.Employment != "" {
		link.Assignment = r.ID
	}
	return link
}
