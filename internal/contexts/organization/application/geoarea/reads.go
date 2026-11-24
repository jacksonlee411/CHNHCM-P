package geoarea

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	base "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	cd "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

func (s Service) read(ctx context.Context, c access.Credentials, fn func(Unit) error) error {
	return s.Store.WithinGeoArea(ctx, c, func(u Unit) error {
		e := fn(u)
		var f *foundation.Fault
		if errors.As(e, &f) && f.Status < 500 {
			return submission.RejectRead(e)
		}
		return e
	})
}
func emptyCaps() org.Capabilities {
	return org.Capabilities{ReadFields: []string{}, WriteFields: []string{}, AddWriteFields: []string{}, CorrectWriteFields: []string{}, DeactivateWriteFields: []string{}, ReactivateWriteFields: []string{}}
}
func (s Service) Context(ctx context.Context, c access.Credentials) (out org.GeoAreaContext, err error) {
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		history, e := u.Access().Decide(ctx, intent("history.read", nil))
		if e != nil {
			return e
		}
		caps := emptyCaps()
		caps.ReadFields = fields
		caps.History = history.Allowed
		// Source initialization is intentionally not a public creation capability.
		out = org.GeoAreaContext{CurrentDate: now.In(foundation.Shanghai).Format("2006-01-02"), DefaultEffectiveDate: foundation.DefaultEffectiveDate, Timezone: "Asia/Shanghai", NameMaxLength: base.NameMaxLength, NameGuidance: base.NameGuidance, Capabilities: caps}
		if slices.Contains(fields, "countryRegionId") {
			canName, e := canReadCountryName(ctx, u)
			if e != nil {
				return e
			}
			if canName {
				countries, e := u.AddressCountries(ctx)
				if e != nil {
					return e
				}
				for _, country := range countries {
					if r := cd.Kernel.FactAt(country.Timeline.Records, out.CurrentDate); r != nil {
						out.FilterCountries = append(out.FilterCountries, org.GeoAreaCountryOption{ID: country.Timeline.ID, Name: r.Data.Name})
					}
				}
			}
		}
		return nil
	})
	return
}
func validLevel(level string) bool {
	return slices.Contains([]string{"province", "prefecture", "county", "region", "district", "postal-region", "parish", "county-city", "township"}, level)
}
func canReadCountryName(ctx context.Context, u Unit) (bool, error) {
	decision, err := u.Access().Decide(ctx, access.Intent{Action: "country.read", Kind: "country-catalog"})
	return decision.Allowed && slices.Contains(decision.ReadFields, "name"), err
}
func queryFields(ctx context.Context, u Unit, in org.GeoAreaQuery) ([]string, error) {
	fields, e := readable(ctx, u, false)
	if e != nil || in.Filter == nil || in.Filter.Field != "countryRegionId" {
		return fields, e
	}
	ok, e := canReadCountryName(ctx, u)
	if e != nil {
		return nil, e
	}
	if !ok {
		return nil, denied()
	}
	return fields, nil
}
func normalizeQuery(in *org.GeoAreaQuery) error {
	if in.CountryRegionID != "" && !foundation.ValidID(in.CountryRegionID) || in.ParentAreaID != "" && !foundation.ValidID(in.ParentAreaID) {
		return fault("M02_INVALID_INPUT", "", "国家或父级筛选无效")
	}
	if in.Level != "" && !validLevel(in.Level) {
		return fault("M02_INVALID_INPUT", "level", "层级无效")
	}
	if in.Page == 0 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = 20
	}
	if in.Page < 1 || !slices.Contains([]int{20, 50, 100}, in.PageSize) {
		return fault("M02_INVALID_INPUT", "page", "分页参数无效")
	}
	if in.Status == "all" {
		in.Status = ""
	}
	in.Q = strings.TrimSpace(in.Q)
	if utf8.RuneCountInString(in.Q) > 200 {
		return fault("M02_INVALID_INPUT", "q", "搜索内容过长")
	}
	if in.Sort == "" {
		in.Sort = "code"
	}
	if !slices.Contains([]string{"code", "name", "name_desc"}, in.Sort) || !slices.Contains([]string{"", "active", "inactive", "not_effective"}, in.Status) {
		return fault("M02_INVALID_INPUT", "", "排序或状态无效")
	}
	if f := in.Filter; f != nil {
		switch f.Field {
		case "level", "countryRegionId":
			if f.Operator != "equals" || f.Field == "level" && !validLevel(f.Value) || f.Field == "countryRegionId" && !foundation.ValidID(f.Value) {
				return fault("M02_INVALID_INPUT", "filterValue", "请选择有效的列值")
			}
		default:
			if !slices.Contains([]string{"code", "name", "status"}, f.Field) || !slices.Contains([]string{"contains", "equals", "is", "isAnyOf"}, f.Operator) || f.Value == "" {
				return fault("M02_INVALID_INPUT", "filterField", "筛选字段无效")
			}
		}
	}
	return nil
}
func (s Service) List(ctx context.Context, c access.Credentials, in org.GeoAreaQuery) (out org.GeoAreaList, err error) {
	if err = normalizeQuery(&in); err != nil {
		return
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := queryFields(ctx, u, in)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		day := now.In(foundation.Shanghai).Format("2006-01-02")
		out, e = u.List(ctx, in, fields, day)
		if e != nil {
			return e
		}
		for i := range out.Items {
			row := &out.Items[i]
			if !slices.Contains(fields, "geoAreaPath") && row.CountryRegionID == nil {
				continue
			}
			v, e := u.Load(ctx, row.ID, "")
			if e != nil {
				return e
			}
			r := d.Kernel.FactAt(v.Timeline.Records, day)
			if r == nil {
				r = &v.Timeline.Records[0]
			}
			if slices.Contains(fields, "geoAreaPath") {
				row.GeoAreaPath, e = readPath(ctx, u, v.Timeline, *r, fields, false, day)
				if e != nil {
					return e
				}
			}
			if row.CountryRegionID != nil {
				row.CountryRegionName, e = recordCountryName(ctx, u, *r, false, day, row.GeoAreaPath)
				if e != nil {
					return e
				}
			}
		}
		return nil
	})
	return
}
func project(r d.Record, fields []string, summary *org.OperationSummary) org.GeoAreaRecord {
	out := org.GeoAreaRecord{ID: r.ID, Revision: r.Revision}
	has := func(f string) bool { return slices.Contains(fields, f) }
	if has("effectiveDate") {
		out.EffectiveDate = ptr(r.Date)
		end := ptr(r.End)
		out.EndExclusive = &end
		inclusive := ptr(dated.NextDate(r.End, -1))
		out.EndInclusive = &inclusive
	}
	if has("name") {
		out.Name = &r.Data.Name
	}
	if has("nameEn") {
		v := (d.Policy{}).Clone(r.Data).NameEn
		out.NameEn = &v
	}
	if has("state") {
		out.State = ptr(r.State)
	}
	if has("operationType") {
		out.OperationType = ptr(r.Type)
		out.TypeLabel = ptr(r.TypeLabel)
	}
	if has("operationReason") {
		out.OperationReason = ptr(r.Reason)
		out.ReasonLabel = ptr(r.ReasonLabel)
	}
	st := r.Data.Structure
	if has("countryRegionId") {
		out.CountryRegionID = ptr(st.CountryID)
	}
	if has("parentAreaId") {
		v := ptr(st.ParentID)
		out.ParentAreaID = &v
	}
	if has("scheme") {
		out.Scheme = ptr(st.Scheme)
	}
	if has("nature") {
		out.Nature = ptr(st.Nature)
	}
	if has("level") {
		out.Level = ptr(st.Level)
	}
	out.OperationSummary = summary
	return out
}
func status(t d.Timeline, today string) string {
	r := d.Kernel.FactAt(t.Records, today)
	if r == nil {
		return "not_effective"
	}
	return r.State
}
func (s Service) Detail(ctx context.Context, c access.Credentials, in org.DetailInput) (out org.GeoAreaDetail, err error) {
	if !foundation.ValidID(in.ID) || in.SourceVersion != "" && !foundation.ValidID(in.SourceVersion) || in.SelectedRecordID != "" && !foundation.ValidID(in.SelectedRecordID) || in.BusinessAt != "" {
		return out, fault("M02_INVALID_INPUT", "", "对象、记录或来源标识无效")
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, in.SourceVersion != "")
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		today := now.In(foundation.Shanghai).Format("2006-01-02")
		if e = published(ctx, u, in.ID); e != nil {
			return e
		}
		current, e := u.Load(ctx, in.ID, "")
		if e != nil {
			return e
		}
		if current.Timeline.ID == "" {
			return missing()
		}
		state := current
		if in.SourceVersion != "" && in.SourceVersion != current.Timeline.Source {
			state, e = u.Load(ctx, in.ID, in.SourceVersion)
			if e != nil {
				return e
			}
			if state.Timeline.ID == "" {
				return foundation.Fail(404, "M02_SOURCE_NOT_FOUND", "行政区划与地址分区来源不存在")
			}
		}
		t := state.Timeline
		history, e := u.Access().Decide(ctx, intent("history.read", nil))
		if e != nil {
			return e
		}
		caps := emptyCaps()
		caps.ReadFields = fields
		caps.History = history.Allowed
		out = org.GeoAreaDetail{ID: t.ID, CurrentDate: today, NameGuidance: base.NameGuidance, HeadSourceVersion: current.Timeline.Source, ResolvedSourceVersion: t.Source, ReadOnly: in.SourceVersion != "", Timeline: []org.GeoAreaRecord{}, Capabilities: caps}
		if slices.Contains(fields, "code") {
			out.Code = ptr(t.Origin.Code)
		}
		if slices.Contains(fields, "state") {
			out.Status = ptr(status(t, today))
		}
		selected := in.SelectedRecordID
		if selected == "" {
			r := d.Kernel.FactAt(t.Records, today)
			if r != nil {
				selected = r.ID
			} else {
				selected = t.First
			}
		}
		var original *d.Record
		currentID := t.First
		if fact := d.Kernel.FactAt(t.Records, today); fact != nil {
			currentID = fact.ID
		}
		historyFields := []string{}
		if history.Allowed {
			historyFields, e = readable(ctx, u, true)
			if e != nil {
				return e
			}
		}
		if selected != currentID && !history.Allowed {
			return denied()
		}
		for _, r := range t.Records {
			rowFields := fields
			if r.ID != currentID {
				if !history.Allowed {
					continue
				}
				rowFields = historyFields
			}
			var summary *org.OperationSummary
			if history.Allowed {
				for _, v := range state.Summaries {
					if v.RecordID == r.ID && v.Revision == r.Revision {
						x := v
						summary = &x
						break
					}
				}
			}
			v := project(r, rowFields, summary)
			out.Timeline = append(out.Timeline, v)
			if r.ID == selected {
				out.SelectedRecord = &v
				x := r
				original = &x
				out.SelectedRecordIsFuture = r.Date > today
			}
		}
		if original != nil && slices.Contains(fields, "geoAreaPath") && (selected == currentID || slices.Contains(historyFields, "geoAreaPath")) {
			pathFields := fields
			if selected != currentID {
				pathFields = historyFields
			}
			path, e := readPath(ctx, u, t, *original, pathFields, in.SourceVersion != "", today)
			if e != nil {
				return e
			}
			out.GeoAreaPath = path
			out.PathDate = pathDay(*original, out.ReadOnly, today)
		}
		if original != nil && out.SelectedRecord.CountryRegionID != nil {
			out.CountryRegionName, e = recordCountryName(ctx, u, *original, out.ReadOnly, today, out.GeoAreaPath)
			if e != nil {
				return e
			}
		}
		out.SelectionMissing = out.SelectedRecord == nil
		if out.ReadOnly || original == nil || !history.Allowed {
			return nil
		}
		correct, e := u.Access().Decide(ctx, intent("record.correct", nil))
		if e != nil {
			return e
		}
		out.Capabilities.Correct = correct.Allowed && original.Type != "DEACTIVATE"
		out.Capabilities.CorrectWriteFields = correct.WriteFields
		if out.Capabilities.Correct {
			out.Capabilities.Correction = &org.CorrectionCapabilities{Facets: []string{"CORRECT"}, ClassificationTypes: []string{}}
		}
		if t.Records[len(t.Records)-1].ID != selected {
			return nil
		}
		add, e := u.Access().Decide(ctx, intent("record.add", []string{"effectiveDate", "operationType", "operationReason"}))
		if e != nil {
			return e
		}
		window, e := d.Kernel.PrepareAddition(t.Records, selected, d.MinDate, today)
		if e != nil {
			return e
		}
		if add.Allowed && window.Available && slices.Contains(fields, "effectiveDate") {
			defaults := project(window.Anchor, fields, nil)
			out.Capabilities.Add = true
			out.Capabilities.AddWriteFields = add.WriteFields
			out.AdditionContext = &org.GeoAreaAdditionContext{Available: true, AnchorRecordID: selected, Mode: "APPEND", MinEffectiveDate: window.Min, MaxEffectiveDate: ptr(window.Max), DefaultEffectiveDate: window.Default, Defaults: defaults}
		}
		kind, act := "DEACTIVATE", "deactivate"
		if original.State == "inactive" {
			kind, act = "REACTIVATE", "reactivate"
		}
		allowed, e := u.Access().Decide(ctx, intent(act, []string{"effectiveDate", "operationType", "operationReason"}))
		if e != nil {
			return e
		}
		life, e := d.Kernel.PrepareLifecycleAppend(t.Records, selected, kind, d.MinDate, today)
		if e != nil {
			return e
		}
		if allowed.Allowed && life.Available && slices.Contains(fields, "effectiveDate") {
			out.LifecycleContext = &org.GeoAreaLifecycleContext{AnchorRecordID: selected, OperationType: kind, MinEffectiveDate: life.Min, MaxEffectiveDate: ptr(life.Max), DefaultEffectiveDate: life.Default, Defaults: project(life.Anchor, fields, nil)}
			if kind == "DEACTIVATE" {
				out.Capabilities.Deactivate = true
				out.Capabilities.DeactivateWriteFields = allowed.WriteFields
			} else {
				out.Capabilities.Reactivate = true
				out.Capabilities.ReactivateWriteFields = allowed.WriteFields
			}
		}
		return nil
	})
	return
}
func (s Service) Preview(ctx context.Context, c access.Credentials, id string, in org.GeoAreaPreviewInput) (out org.GeoAreaPreview, err error) {
	cmd := org.GeoAreaCommand{Action: in.Action, GeoAreaID: id, RecordID: in.RecordID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Add: org.GeoAreaAddInput{ExpectedSourceVersion: in.ExpectedSourceVersion, AnchorRecordID: in.AnchorRecordID, EffectiveDate: in.EffectiveDate, OperationType: in.OperationType, OperationReason: in.OperationReason, Changes: in.Changes}, Correct: org.GeoAreaCorrectInput{ExpectedSourceVersion: in.ExpectedSourceVersion, ExpectedRecordRevision: in.ExpectedRecordRevision, Changes: in.Changes}}
	_, declared, e := metadata(cmd)
	if e != nil {
		return out, e
	}
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, true)
		if e != nil {
			return e
		}
		if _, e = require(ctx, u, action(cmd), declared); e != nil {
			return e
		}
		if e = published(ctx, u, id); e != nil {
			return e
		}
		state, e := u.Load(ctx, id, "")
		if e != nil {
			return e
		}
		if state.Timeline.ID == "" {
			return missing()
		}
		if state.Timeline.Source != in.ExpectedSourceVersion {
			return fault("M02_SOURCE_CONFLICT", "", "来源已变化，请重新核对资料")
		}
		candidate, e := candidate(ctx, state.Timeline, cmd, d.Adoption{})
		if e != nil {
			return e
		}
		out.ReadFields = fields
		for _, r := range candidate.Records {
			if r.ID == in.RecordID || !foundation.ValidID(r.ID) {
				v := project(r, fields, nil)
				out.AfterRecord = &v
			}
		}
		return nil
	})
	return
}
