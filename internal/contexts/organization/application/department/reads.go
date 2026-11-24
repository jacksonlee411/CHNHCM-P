package department

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/enterprisecheck"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	base "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func fingerprint(v any) string {
	b, _ := json.Marshal(v)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}
func (s Service) Context(ctx context.Context, c access.Credentials) (out org.DepartmentContext, err error) {
	err = s.Store.WithinDepartment(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		visible := false
		read, write := []string{}, []string{}
		create, history := false, false
		for _, g := range u.Access().Grants() {
			if g.ScopeKind == "enterprise" && g.ScopeID != ent.ID {
				continue
			}
			switch g.Action {
			case "department.read":
				visible = true
				read = append(read, g.ReadFields...)
			case "department.history.read":
				history = true
			case "department.create":
				create = true
				write = append(write, g.WriteFields...)
			}
		}
		if !visible {
			return submission.RejectRead(denied())
		}
		// The entry is usable only if at least one real BU scope admits the
		// complete create declaration and its mandatory result read/history rights.
		if create {
			create = false
			bus, e := u.CreationBUs(ctx)
			if e != nil {
				return e
			}
			for _, bu := range bus {
				t := d.Timeline{Enterprise: ent.ID}
				target := fact(t, now, bu)
				dec, e := u.Access().Decide(ctx, requirements(org.DepartmentCommand{Action: "CREATE", Fields: CreateFields}, t, target))
				if e != nil {
					return e
				}
				if dec.Allowed {
					create = true
					break
				}
			}
		}
		slices.Sort(read)
		read = slices.Compact(read)
		slices.Sort(write)
		write = slices.Compact(write)
		out = org.DepartmentContext{Enterprise: enterprisecheck.Display(ent, now), CurrentDate: now.In(foundation.Shanghai).Format("2006-01-02"), DefaultEffectiveDate: foundation.DefaultEffectiveDate, Timezone: "Asia/Shanghai", NameMaxLength: base.NameMaxLength, NameGuidance: base.NameGuidance, CodeMaxLength: 40, FieldCatalog: slices.Clone(ReadFields), Capabilities: org.Capabilities{Create: create, History: history, ReadFields: read, WriteFields: write, AddWriteFields: []string{}, CorrectWriteFields: []string{}, DeactivateWriteFields: []string{}, ReactivateWriteFields: []string{}}}
		return nil
	})
	return
}

type referenceProjection struct {
	value        *org.DepartmentReference
	readableName bool
	err          error
}
type projector struct {
	ctx              context.Context
	u                Unit
	s                d.Snapshot
	enterprise       org.Enterprise
	now              time.Time
	day              string
	refs             map[string]referenceProjection
	paths            *d.PathResolver
	listPrefixes     map[listPathKey]*listPathPrefix
	listVisiting     map[listPathKey]bool
	listCurrent      []listPathPrefix
	listCurrentState []uint8
	digestScratch    digestBytes
	listFacts        *ListFacts
	readSet          *projectedReads
	pathSnapshots    map[string]d.Snapshot
}

func newProjector(ctx context.Context, u Unit, s d.Snapshot, ent org.Enterprise, now time.Time) *projector {
	ent = enterprisecheck.Display(ent, now)
	p := &projector{ctx: ctx, u: u, s: s, enterprise: ent, now: now, day: now.In(foundation.Shanghai).Format("2006-01-02"), refs: map[string]referenceProjection{}, paths: d.NewPathResolver(s, ent.Name)}
	if planned, ok := u.(projectedReadUnit); ok {
		if reads, ok := planned.guard.(projectedReads); ok {
			p.readSet = &reads
		}
	}
	return p
}
func (p *projector) ref(id, kind string) referenceProjection {
	key := kind + ":" + id
	if v, ok := p.refs[key]; ok {
		return v
	}
	out := referenceProjection{value: &org.DepartmentReference{ID: id}}
	var dec access.Decision
	var e error
	if p.readSet != nil {
		dec = p.readSet.referenceDecision(id, kind)
		if !dec.Allowed {
			p.refs[key] = out
			return out
		}
	} else {
		dec, e = reference(p.ctx, p.u, p.s, p.enterprise.ID, id, kind, p.now)
	}
	if e != nil {
		var f *foundation.Fault
		if !errors.As(e, &f) || f.Status >= 500 {
			out.err = e
		}
		p.refs[key] = out
		return out
	}
	code, name := "", ""
	if kind == "bu" {
		b := p.s.BU(id)
		if b != nil {
			code = b.Code
			r := d.BUAt(b.Records, p.day)
			if r == nil && len(b.Records) > 0 {
				r = &b.Records[0]
			}
			if r != nil {
				name = r.Name
			}
		}
	} else {
		t := p.s.Department(id)
		if t != nil {
			code = t.Code
			r := d.ScopeRecord(*t, p.day)
			if r != nil {
				name = r.Data.Name
			}
		}
	}
	if slices.Contains(dec.ReadFields, "code") {
		out.value.Code = ptr(code)
	}
	if slices.Contains(dec.ReadFields, "name") {
		out.value.Name = ptr(name)
		out.readableName = true
	}
	p.refs[key] = out
	return out
}
func (p *projector) record(r d.Record, fields []string, summary *org.OperationSummary) (org.DepartmentRecord, error) {
	out := org.DepartmentRecord{ID: r.ID, Revision: r.Revision, OperationSummary: summary}
	for _, field := range fields {
		switch field {
		case "name":
			out.Name = ptr(r.Data.Name)
		case "shortName":
			name := r.Data.ShortName
			out.ShortName = &name
		case "effectiveDate":
			out.EffectiveDate = ptr(r.Date)
			end := ptr(r.End)
			var inclusive *string
			if r.End != "" {
				inclusive = ptr(dated.NextDate(r.End, -1))
			}
			out.EndExclusive, out.EndInclusive = &end, &inclusive
		case "status":
			out.State = ptr(r.State)
		case "operationType":
			out.OperationType = ptr(r.Type)
			out.TypeLabel = ptr(r.TypeLabel)
		case "operationReason":
			out.OperationReason = ptr(r.Reason)
			out.ReasonLabel = ptr(r.ReasonLabel)
		case "managementBuId":
			out.ManagementBUID = ptr(r.Data.BU)
			ref := p.ref(r.Data.BU, "bu")
			if ref.err != nil {
				return out, ref.err
			}
			out.ManagementBU = ref.value
		case "parentDepartmentId":
			out.ParentDepartmentID = ptr(r.Data.Parent)
			if r.Data.Parent != "" {
				ref := p.ref(r.Data.Parent, "department")
				if ref.err != nil {
					return out, ref.err
				}
				out.ParentDepartment = ref.value
			}
		}
	}
	return out, nil
}
func (p *projector) path(t d.Timeline, r d.Record, fields []string, explicit bool, evidence []d.PathSegment) (org.DepartmentPath, error) {
	restricted := org.DepartmentPath{PathMode: "restricted", PathVersion: fingerprint("restricted")}
	if !slices.Contains(fields, "departmentPath") || !slices.Contains(fields, "effectiveDate") || !slices.Contains(fields, "status") {
		return restricted, nil
	}
	date := p.now.In(foundation.Shanghai).Format("2006-01-02")
	mode := "active"
	pathDate := ""
	if r.State == "inactive" {
		mode = "inactive_before_stop"
		pathDate = dated.NextDate(r.Date, -1)
	} else if len(t.Records) > 0 && date < t.Records[0].Date {
		mode = "future_first"
		pathDate = t.Records[0].Date
	}
	segments := []d.PathSegment{}
	if explicit {
		for _, seg := range evidence {
			if seg.RecordID == r.ID {
				segments = append(segments, seg)
			}
		}
		if e := d.ValidateEvidence(t, r, segments); e != nil {
			return restricted, foundation.Fail(503, "M02_DEPARTMENT_PATH_UNAVAILABLE", "历史路径暂时无法加载")
		}
	} else {
		single := t
		single.Records = []d.Record{r}
		v, e := d.Evidence(p.ctx, p.s, single, p.enterprise.Name)
		if e != nil {
			return restricted, e
		}
		segments = v
	}
	out := org.DepartmentPath{PathMode: mode, PathBusinessAt: ptr(pathDate), PathSegments: []org.DepartmentPathSegment{}}
	for _, seg := range segments {
		names := []string{}
		for _, n := range seg.Nodes {
			// Keep stored evidence intact; expose only actual department names.
			if n.Kind != "department" {
				continue
			}
			ref := p.ref(n.DepartmentID, "department")
			if ref.err != nil {
				return restricted, ref.err
			}
			if !ref.readableName {
				return restricted, nil
			}
			names = append(names, n.Name)
		}
		path := strings.Join(names, " / ")
		var inclusive *string
		if seg.End != "" {
			inclusive = ptr(dated.NextDate(seg.End, -1))
		}
		out.PathSegments = append(out.PathSegments, org.DepartmentPathSegment{EndInclusive: inclusive, Start: seg.Start, EndExclusive: ptr(seg.End), Path: path})
		if len(segments) == 1 {
			out.Path = &path
		}
	}
	out.PathVersion = fingerprint(out)
	return out, nil
}

func (s Service) Detail(ctx context.Context, c access.Credentials, in org.DepartmentDetailInput) (out org.DepartmentDetail, err error) {
	if !foundation.ValidID(in.DepartmentID) || in.SourceVersion != "" && !foundation.ValidID(in.SourceVersion) || in.SelectedRecordID != "" && !foundation.ValidID(in.SelectedRecordID) {
		return out, foundation.Fail(400, "M02_INVALID_INPUT", "查询条件无效，请重新打开资料")
	}
	err = s.Store.WithinDepartment(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		read, e := u.Snapshot(ctx, in.DepartmentID, in.SourceVersion)
		if e != nil {
			return e
		}
		current := read.Snapshot.Department(in.DepartmentID)
		if current == nil {
			return submission.RejectRead(missing())
		}
		fields, e := readable(ctx, u, *current, now, in.SourceVersion != "")
		if e != nil {
			return submission.RejectRead(e)
		}
		t := read.Selected
		if t == nil {
			return submission.RejectRead(missing())
		}
		day := now.In(foundation.Shanghai).Format("2006-01-02")
		out = org.DepartmentDetail{NameGuidance: base.NameGuidance, DepartmentID: t.ID, Enterprise: enterprisecheck.Display(ent, now), CurrentDate: day, ResolvedSourceVersion: t.Source, HeadSourceVersion: current.Source, ReadOnly: in.SourceVersion != "", Timeline: []org.DepartmentRecord{}, Capabilities: org.Capabilities{ReadFields: fields, WriteFields: []string{}, AddWriteFields: []string{}, CorrectWriteFields: []string{}, DeactivateWriteFields: []string{}, ReactivateWriteFields: []string{}}}
		if slices.Contains(fields, "code") {
			out.Code = ptr(t.Code)
		}
		if slices.Contains(fields, "status") {
			out.Status = d.Status(*t, day)
			out.Result = "OK"
			if out.Status == "not_effective" {
				out.Result = "M02_NO_FACT"
			}
		}
		selected := in.SelectedRecordID
		if selected == "" {
			r := d.ScopeRecord(*t, day)
			if r != nil {
				selected = r.ID
			}
		}
		p := newProjector(ctx, u, read.Snapshot, ent, now)
		var selectedDomain *d.Record
		for _, r := range t.Records {
			summary, ok := read.Summaries[r.Revision]
			var summaryPtr *org.OperationSummary
			if ok {
				summaryPtr = &summary
			}
			record, e := p.record(r, fields, summaryPtr)
			if e != nil {
				return e
			}
			out.Timeline = append(out.Timeline, record)
			if r.ID == selected {
				v := r
				selectedDomain = &v
				out.SelectedRecord = &record
				if slices.Contains(fields, "effectiveDate") {
					out.SelectedRecordIsFuture = r.Date > day
				}
			}
		}
		if selectedDomain == nil && in.SourceVersion != "" {
			source, e := u.Source(ctx, in.DepartmentID, t.Source)
			if e != nil {
				return e
			}
			for _, it := range source.Items {
				if it.RecordID != selected || it.After == nil || !it.After.Deleted || it.Before == nil {
					continue
				}
				original, e := u.Snapshot(ctx, in.DepartmentID, it.BeforeSource)
				if e != nil {
					return e
				}
				if original.Selected == nil {
					return foundation.Fail(503, "M02_DEPARTMENT_PATH_UNAVAILABLE", "删除前的资料暂时无法读取")
				}
				r := it.Before.Record
				record, e := p.record(r, fields, &it.Before.Summary)
				if e != nil {
					return e
				}
				record.Deleted = true
				record.Revision = it.After.Record.Revision
				out.SelectedRecord = &record
				out.ReadOnly = true
				out.Path, e = p.path(*original.Selected, r, fields, true, original.Evidence)
				return e
			}
		}
		if selectedDomain == nil {
			if in.SelectedRecordID != "" {
				out.SelectionMissing = true
				out.Path = org.DepartmentPath{PathMode: "restricted", PathVersion: fingerprint("restricted")}
				return nil
			}
			return submission.RejectRead(foundation.Fail(404, "M02_RECORD_NOT_FOUND", "所选历史资料中没有该记录"))
		}
		out.Path, e = p.path(*t, *selectedDomain, fields, in.SourceVersion != "", read.Evidence)
		if e != nil {
			return e
		}
		if in.SourceVersion == "" {
			return populateCapabilities(ctx, u, &out, *current, *selectedDomain, now, p)
		}
		return nil
	})
	return
}

func normalizeDepartmentList(in *org.DepartmentListInput) error {
	if in.ScopeDepartmentID == "" {
		if in.IncludeDescendants != nil {
			return foundation.Fail(400, "M02_INVALID_INPUT", "包含下级必须指定部门范围")
		}
	} else {
		if !foundation.ValidID(in.ScopeDepartmentID) {
			return foundation.Fail(400, "M02_INVALID_INPUT", "部门范围无效，请重新选择")
		}
		if in.IncludeDescendants == nil {
			v := true
			in.IncludeDescendants = &v
		}
	}
	if in.Page == 0 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = 20
	}
	if in.Sort == "" {
		in.Sort = "code"
	}
	if in.Status == "" {
		in.Status = "all"
	}
	if in.Page < 1 || !slices.Contains([]int{20, 50, 100}, in.PageSize) || (in.Page-1) > int(^uint(0)>>1)/in.PageSize || !slices.Contains([]string{"code", "name", "name_desc"}, in.Sort) || !slices.Contains([]string{"all", "active", "inactive", "not_effective"}, in.Status) || utf8.RuneCountInString(in.Q) > 400 {
		return foundation.Fail(400, "M02_INVALID_INPUT", "查询条件无效")
	}
	if in.Cursor != "" && (in.Filter == nil || in.Filter.Field != "departmentPath") {
		return foundation.Fail(400, "M02_INVALID_INPUT", "查询位置无效，请重新查询")
	}
	in.Q = strings.ToLower(strings.TrimSpace(in.Q))
	if in.Filter != nil {
		v := *in.Filter
		switch v.Field {
		case "code", "name", "shortName", "departmentPath":
			v.Value = strings.ToLower(strings.TrimSpace(v.Value))
			if !slices.Contains([]string{"contains", "equals"}, v.Operator) || v.Value == "" || utf8.RuneCountInString(v.Value) > 400 {
				return foundation.Fail(400, "M02_INVALID_INPUT", "列筛选条件无效")
			}
		case "managementBuId":
			v.Value = strings.ToLower(strings.TrimSpace(v.Value))
			if v.Operator != "equals" || !foundation.ValidID(v.Value) {
				return foundation.Fail(400, "M02_INVALID_INPUT", "管理业务单元筛选无效，请重新选择")
			}
		case "status":
			values := strings.Split(v.Value, ",")
			for _, x := range values {
				if !slices.Contains([]string{"active", "inactive", "not_effective"}, x) {
					return foundation.Fail(400, "M02_INVALID_INPUT", "状态筛选无效")
				}
			}
			slices.Sort(values)
			v.Value = strings.Join(slices.Compact(values), ",")
			if !slices.Contains([]string{"is", "isAnyOf"}, v.Operator) {
				return foundation.Fail(400, "M02_INVALID_INPUT", "状态筛选无效")
			}
		case "effectiveDate":
			if !foundation.ValidDate(v.Value) || !slices.Contains([]string{"is", "before", "after", "onOrBefore", "onOrAfter"}, v.Operator) {
				return foundation.Fail(400, "M02_INVALID_INPUT", "日期筛选无效")
			}
		default:
			return foundation.Fail(400, "M02_INVALID_INPUT", "列筛选字段无效")
		}
		in.Filter = &v
	}
	return nil
}

// List/CSV responses borrow immutable facts; candidate mutation still clones.
func borrowedString(v *string) *string {
	if *v == "" {
		return nil
	}
	return v
}
func (s Service) List(ctx context.Context, c access.Credentials, in org.DepartmentListInput) (out org.DepartmentList, err error) {
	if e := normalizeDepartmentList(&in); e != nil {
		return out, e
	}
	err = s.Store.WithinDepartment(ctx, c, func(u Unit) error {
		var rows []listMember
		var e error
		out, rows, _, _, e = queryRows(ctx, u, in, QueryPage)
		if e != nil {
			return submission.RejectRead(e)
		}
		out.Items = memberSummaries(rows)
		return nil
	})
	return
}

func departmentReadEntry(u Unit, enterprise string) bool {
	for _, g := range u.Access().Grants() {
		if g.Action == "department.read" && (g.ScopeKind != "enterprise" || g.ScopeID == enterprise) {
			return true
		}
	}
	return false
}
