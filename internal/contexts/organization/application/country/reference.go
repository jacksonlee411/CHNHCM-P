package country

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
	"time"
)

// ReferenceUnit uses the consumer's transaction and the country's existing lock.
// It exposes no geography chain, publication, or write capability.
type ReferenceUnit interface {
	Access() access.Guard
	Now(context.Context) (time.Time, error)
	Country(context.Context, string, string) (State, error)
	LockReference(context.Context, string, string) error
}
type ReferenceRequest struct{ CountryID, Date string }

func countryReferenceFields(ctx context.Context, u ReferenceUnit, historical bool) ([]string, error) {
	r, e := u.Access().Decide(ctx, access.Intent{Action: "country.read", Kind: "country-catalog"})
	if e != nil {
		return nil, e
	}
	if !r.Allowed {
		return nil, nil
	}
	fields := slices.Clone(r.ReadFields)
	if historical {
		h, e := u.Access().Decide(ctx, access.Intent{Action: "country.history.read", Kind: "country-catalog"})
		if e != nil {
			return nil, e
		}
		if !h.Allowed {
			return nil, nil
		}
		fields = slices.DeleteFunc(fields, func(f string) bool { return !slices.Contains(h.ReadFields, f) })
	}
	return fields, nil
}
func ResolveCountries(ctx context.Context, u ReferenceUnit, requests []ReferenceRequest) ([]d.Adoption, error) {
	out := make([]d.Adoption, len(requests))
	if len(requests) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, foundation.ObjectGuardBudget(ctx))
	defer cancel()
	ids := make([]string, 0, len(requests))
	for _, r := range requests {
		if !foundation.ValidID(r.CountryID) || !foundation.ValidDate(r.Date) {
			return nil, foundation.Fail(400, "M02_INVALID_INPUT", "国家或采用日期无效")
		}
		ids = append(ids, r.CountryID)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	// Authorize before loading a target; locks and reads then use one bounded scope.
	fields, e := countryReferenceFields(ctx, u, false)
	if e != nil {
		return nil, e
	}
	for _, f := range []string{"name", "state", "effectiveDate"} {
		if !slices.Contains(fields, f) {
			return nil, foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许选用国家")
		}
	}
	for _, id := range ids {
		if e = u.LockReference(ctx, "country", id); e != nil {
			return nil, e
		}
	}
	now, e := u.Now(ctx)
	if e != nil {
		return nil, e
	}
	today := now.In(foundation.Shanghai).Format("2006-01-02")
	states := map[string]State{}
	for _, id := range ids {
		v, e := u.Country(ctx, id, "")
		if e != nil {
			return nil, e
		}
		states[id] = v
	}
	for i, r := range requests {
		t := states[r.CountryID].Timeline
		fact := d.Kernel.FactAt(t.Records, r.Date)
		current := d.Kernel.FactAt(t.Records, today)
		if t.ID == "" || fact == nil || fact.State != "active" {
			return nil, &foundation.Fault{Status: 409, Code: "M02_REFERENCE_UNAVAILABLE", Field: "countryRegionId", Message: "国家/地区在采用日不可选用"}
		}
		allowed, e := countryReferenceFields(ctx, u, current == nil || current.ID != fact.ID)
		if e != nil {
			return nil, e
		}
		for _, f := range []string{"name", "state", "effectiveDate"} {
			if !slices.Contains(allowed, f) {
				return nil, foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许选用国家")
			}
		}
		out[i] = d.Adoption{Date: r.Date, ObjectID: t.ID, SourceVersion: t.Source, RecordID: fact.ID, Revision: fact.Revision}
	}
	return out, nil
}

// ProjectCountry never substitutes today's label for an adopted source.
func ProjectCountry(ctx context.Context, u ReferenceUnit, a d.Adoption) (*string, error) {
	fields, e := countryReferenceFields(ctx, u, true)
	if e != nil {
		return nil, e
	}
	if !slices.Contains(fields, "name") {
		return nil, nil
	}
	if e = d.ValidateAdoption(a.ObjectID, a); e != nil {
		return nil, e
	}
	v, e := u.Country(ctx, a.ObjectID, a.SourceVersion)
	if e != nil {
		return nil, e
	}
	fact := d.Kernel.FactAt(v.Timeline.Records, a.Date)
	if v.Timeline.ID != a.ObjectID || v.Timeline.Source != a.SourceVersion || fact == nil || fact.ID != a.RecordID || fact.Revision != a.Revision || fact.State != "active" {
		return nil, foundation.Fail(500, "M02_REFERENCE_CORRUPT", "国家采用依据读取失败")
	}
	name := fact.Data.Name
	return &name, nil
}
