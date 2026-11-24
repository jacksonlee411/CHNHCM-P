package location

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	country "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
)

// This read projects the proposed action's adoption date before a terminal area
// has been chosen. It never creates a persistable candidate or invents a reason.
func (s Service) ReferenceContext(ctx context.Context, c access.Credentials, in org.LocationReferenceInput) (out org.LocationReferenceContext, err error) {
	out.Countries = []org.LocationCountryOption{}
	err = s.read(ctx, c, func(u Unit) error {
		fields, e := readable(ctx, u, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		today := now.In(foundation.Shanghai).Format("2006-01-02")
		r := d.Record{ID: "reference", Type: "CREATE", Date: in.EffectiveDate}
		var before []d.Record
		var predecessor *d.Record
		writeAction := "create"
		if in.Action == "CREATE" {
			if in.LocationID != "" || in.ExpectedSourceVersion != "" || in.RecordID != "" || in.AnchorRecordID != "" || in.NewEffectiveDate != "" || in.Classification != nil {
				return fault("M02_FORBIDDEN_INPUT", "", "新建地址上下文不接受维护目标")
			}
			if r.Date == "" {
				r.Date = foundation.DefaultEffectiveDate
			}
		} else {
			if !foundation.ValidID(in.LocationID) || !foundation.ValidID(in.ExpectedSourceVersion) {
				return fault("M02_INVALID_INPUT", "", "地点或来源身份无效")
			}
			state, e := u.Load(ctx, in.LocationID, "")
			if e != nil {
				return e
			}
			t := state.Timeline
			if t.ID == "" {
				return missing()
			}
			if t.Source != in.ExpectedSourceVersion {
				return fault("M02_SOURCE_CONFLICT", "", "资料已更新，请重新核对")
			}
			before = t.Records
			target := in.RecordID
			if in.Action == "ADD" || in.Action == "REACTIVATE" || in.Action == "DEACTIVATE" {
				target = in.AnchorRecordID
			}
			n := slices.IndexFunc(before, func(v d.Record) bool { return v.ID == target })
			if n < 0 {
				return missing()
			}
			r = before[n]
			current := d.Kernel.FactAt(before, today)
			if current == nil || current.ID != target {
				if fields, e = readable(ctx, u, true); e != nil {
					return e
				}
			}
			switch in.Action {
			case "ADD", "REACTIVATE", "DEACTIVATE":
				predecessor = &before[n]
				r.ID = "reference"
				r.Date = in.EffectiveDate
				r.Type = in.Action
				writeAction = "record.add"
				if in.Action == "ADD" {
					r.Type = "CHANGE"
					e = d.Kernel.ValidateAdditionPosition(before, target, u.EnterpriseScope().ValidFrom, r.Date)
				} else {
					writeAction = "reactivate"
					if in.Action == "DEACTIVATE" {
						writeAction = "deactivate"
					}
					e = d.Kernel.ValidateLifecycleAppend(before, target, in.Action, u.EnterpriseScope().ValidFrom, r.Date)
				}
			case "CORRECT", "REDATE":
				writeAction = "record.correct"
				if in.Action == "REDATE" {
					writeAction = "record.redate"
				}
				if r.Revision != in.ExpectedRecordRevision {
					return fault("M02_REVISION_CONFLICT", "", "记录已更新")
				}
				if in.NewEffectiveDate != "" {
					r.Date = in.NewEffectiveDate
					e = d.Kernel.ValidateCorrectionDate(before, target, u.EnterpriseScope().ValidFrom, r.Date)
				}
				if in.Classification != nil {
					typ := in.Classification.OperationType
					if r.ID == t.First && typ != "CREATE" || r.ID != t.First && !slices.Contains([]string{"CHANGE", "DEACTIVATE", "REACTIVATE"}, typ) {
						return fault("M02_INVALID_INPUT", "classification.operationType", "操作类型无效")
					}
					r.Type = typ
				}
				if n > 0 {
					predecessor = &before[n-1]
				}
			default:
				return fault("M02_INVALID_INPUT", "action", "此操作没有地址选择上下文")
			}
			if e != nil {
				return e
			}
		}
		if !foundation.ValidDate(r.Date) || r.Date < u.EnterpriseScope().ValidFrom {
			return fault("M02_INVALID_INPUT", "effectiveDate", "日期须在企业有效范围内")
		}
		writes := []string{}
		for key, v := range in.Changes {
			switch key {
			case "countryRegionId":
				if v == nil {
					return fault("M02_INVALID_INPUT", "changes.countryRegionId", "国家不能为空")
				}
				if *v != "" && !foundation.ValidID(*v) {
					return fault("M02_INVALID_INPUT", "changes.countryRegionId", "国家身份无效")
				}
				r.Data.CountryID = *v
			case "geoAreaId":
				r.Data.AreaID = v
				if v != nil && *v == "" {
					r.Data.AreaID = nil
				}
				if v != nil && *v != "" && !foundation.ValidID(*v) {
					return fault("M02_INVALID_INPUT", "changes.geoAreaId", "区域身份无效")
				}
			default:
				return fault("M02_FORBIDDEN_INPUT", "changes."+key, "地址上下文只接受国家及区域引用")
			}
			writes = append(writes, key)
		}
		if in.NewEffectiveDate != "" {
			if _, e = require(ctx, u, "record.redate", []string{"effectiveDate"}); e != nil {
				return e
			}
		}
		if in.Classification != nil {
			writes = append(writes, "operationType", "operationReason")
		}
		if !(in.Action == "CORRECT" && in.NewEffectiveDate != "" && len(writes) == 0) {
			if _, e = require(ctx, u, writeAction, writes); e != nil {
				return e
			}
		}
		if !slices.Contains(fields, "effectiveDate") {
			return denied()
		}
		out.NewAdoption = d.NeedsAdoption(before, r, predecessor)
		out.AdoptedDate = r.Data.Adoption.Date
		if out.NewAdoption {
			out.AdoptedDate = r.Date
		}
		if !slices.Contains(fields, "countryRegionId") {
			return nil
		}
		read, e := u.Access().Decide(ctx, access.Intent{Action: "country.read", Kind: "country-catalog"})
		if e != nil {
			return e
		}
		if !read.Allowed {
			return nil
		}
		countries, e := u.AddressCountries(ctx)
		if e != nil {
			return e
		}
		for _, v := range countries {
			fact := country.Kernel.FactAt(v.Timeline.Records, out.AdoptedDate)
			if fact == nil || fact.State != "active" {
				continue
			}
			allowed := read.ReadFields
			current := country.Kernel.FactAt(v.Timeline.Records, today)
			if current == nil || current.ID != fact.ID {
				history, e := u.Access().Decide(ctx, access.Intent{Action: "country.history.read", Kind: "country-catalog"})
				if e != nil {
					return e
				}
				if !history.Allowed {
					continue
				}
				allowed = intersection(allowed, history.ReadFields)
			}
			if len(intersection(allowed, []string{"name", "state", "effectiveDate"})) != 3 {
				continue
			}
			option := org.LocationCountryOption{ID: v.Timeline.ID, Name: fact.Data.Name, RequiresArea: v.Timeline.Origin.Code != "MO"}
			if slices.Contains(allowed, "code") {
				option.Code = v.Timeline.Origin.Code
			}
			out.Countries = append(out.Countries, option)
		}
		return nil
	})
	return
}
