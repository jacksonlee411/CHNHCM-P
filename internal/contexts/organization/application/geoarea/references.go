package geoarea

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	ca "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	cd "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"slices"
)

var areaRequired = []string{"name", "code", "level", "countryRegionId", "parentAreaId", "geoAreaPath", "state", "effectiveDate"}
var countryRequired = []string{"name", "state", "effectiveDate"}

type referenceFieldKey struct {
	kind       string
	historical bool
}

// Catalog permissions have one instance target and are fixed by this request's
// policy guard. Reuse the at-most-four field intersections along its finite paths.
func referenceFields(ctx context.Context, u ReferenceUnit, kind string, historical bool) ([]string, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	cached, ok := u.(*cachedUnit)
	if !ok {
		return calculateReferenceFields(ctx, u, kind, historical)
	}
	key := referenceFieldKey{kind, historical}
	if fields, found := cached.readFields[key]; found {
		return fields, nil
	}
	fields, e := calculateReferenceFields(ctx, u, kind, historical)
	if e == nil {
		if cached.readFields == nil {
			cached.readFields = map[referenceFieldKey][]string{}
		}
		cached.readFields[key] = fields
	}
	return fields, e
}

func calculateReferenceFields(ctx context.Context, u ReferenceUnit, kind string, historical bool) ([]string, error) {
	r, e := u.Access().Decide(ctx, access.Intent{Action: kind + ".read", Kind: kind + "-catalog"})
	if e != nil {
		return nil, e
	}
	if !r.Allowed {
		return nil, denied()
	}
	fields := slices.Clone(r.ReadFields)
	if historical {
		h, e := u.Access().Decide(ctx, access.Intent{Action: kind + ".history.read", Kind: kind + "-catalog"})
		if e != nil {
			return nil, e
		}
		if !h.Allowed {
			return nil, denied()
		}
		fields = slices.DeleteFunc(fields, func(f string) bool { return !slices.Contains(h.ReadFields, f) })
	}
	return fields, nil
}
func required(fields, need []string) error {
	for _, f := range need {
		if !slices.Contains(fields, f) {
			return denied()
		}
	}
	return nil
}
func refUnavailable() error {
	return fault("M02_REFERENCE_UNAVAILABLE", "", "国家或区域在采用日不可选用")
}

// adopt is private to fixed initialization. Ordinary maintenance inherits the
// old evidence. Locks are taken Country -> root -> direct parent.
func adopt(ctx context.Context, u ReferenceUnit, s d.Structure, day string, lock bool) (d.Adoption, error) {
	a := d.Adoption{Date: day, Parents: []d.Reference{}}
	now, e := u.Now(ctx)
	if e != nil {
		return a, e
	}
	today := now.In(foundation.Shanghai).Format("2006-01-02")
	if lock {
		if e = u.LockReference(ctx, "country", s.CountryID); e != nil {
			return a, e
		}
	}
	c, e := u.Country(ctx, s.CountryID, "")
	if e != nil {
		return a, e
	}
	if c.Timeline.ID == "" || c.Timeline.Origin.Code != s.CountryCode {
		return a, refUnavailable()
	}
	cr := cd.Kernel.FactAt(c.Timeline.Records, day)
	current := cd.Kernel.FactAt(c.Timeline.Records, today)
	if cr == nil || cr.State != "active" {
		return a, refUnavailable()
	}
	fields, e := referenceFields(ctx, u, "country", current == nil || current.ID != cr.ID)
	if e != nil {
		return a, e
	}
	if e = required(fields, countryRequired); e != nil {
		return a, e
	}
	a.Country = d.Reference{ObjectID: c.Timeline.ID, SourceVersion: c.Timeline.Source, RecordID: cr.ID, Revision: cr.Revision}
	chain := []d.Timeline{}
	id := s.ParentID
	seen := map[string]bool{}
	for id != "" {
		if seen[id] || len(chain) >= 3 {
			return a, refUnavailable()
		}
		seen[id] = true
		v, e := u.Load(ctx, id, "")
		if e != nil {
			return a, e
		}
		t := v.Timeline
		if t.ID == "" || t.Origin.CountryID != s.CountryID {
			return a, refUnavailable()
		}
		chain = append(chain, t)
		id = t.Origin.ParentID
	}
	slices.Reverse(chain)
	for _, t := range chain {
		if lock {
			if e = u.LockReference(ctx, "geoarea", t.ID); e != nil {
				return a, e
			}
			v, e := u.Load(ctx, t.ID, "")
			if e != nil {
				return a, e
			}
			t = v.Timeline
		}
		r := d.Kernel.FactAt(t.Records, day)
		current := d.Kernel.FactAt(t.Records, today)
		if r == nil || r.State != "active" {
			return a, refUnavailable()
		}
		fields, e := referenceFields(ctx, u, "geoarea", current == nil || current.ID != r.ID)
		if e != nil {
			return a, e
		}
		if e = required(fields, areaRequired); e != nil {
			return a, e
		}
		a.Parents = append(a.Parents, d.Reference{ObjectID: t.ID, SourceVersion: t.Source, RecordID: r.ID, Revision: r.Revision})
	}
	return a, d.ValidateAdoption(s, a)
}

// ResolveReference runs inside the consumer's write unit. Its returned chain
// includes the terminal node and represents the actual locked versions.
func ResolveReference(ctx context.Context, u ReferenceUnit, id, day string) (d.Adoption, error) {
	if !foundation.ValidID(id) || !foundation.ValidDate(day) {
		return d.Adoption{}, fault("M02_INVALID_INPUT", "", "区域身份或采用日期无效")
	}
	if ok, e := u.Published(ctx, id); e != nil {
		return d.Adoption{}, e
	} else if !ok {
		return d.Adoption{}, refUnavailable()
	}
	v, e := u.Load(ctx, id, "")
	if e != nil {
		return d.Adoption{}, e
	}
	if v.Timeline.ID == "" {
		return d.Adoption{}, missing()
	}
	s := v.Timeline.Origin.Structure
	s.ParentID = id
	a, e := adopt(ctx, u, s, day, true)
	if e != nil {
		return a, e
	}
	terminal, e := u.IsTerminal(ctx, id)
	if e != nil {
		return a, e
	}
	if !terminal {
		return a, refUnavailable()
	}
	return a, nil
}

func pathDay(r d.Record, exact bool, today string) string {
	if exact {
		return r.Data.Adoption.Date
	}
	if r.Date > today || r.End != "" && today >= r.End {
		return r.Date
	}
	return today
}

func readPath(ctx context.Context, u Unit, t d.Timeline, r d.Record, fields []string, exact bool, today string) ([]org.GeoAreaPathNode, error) {
	if !slices.Contains(fields, "name") {
		return nil, nil
	}
	a := r.Data.Adoption
	if !exact {
		day := pathDay(r, false, today)
		// Display follows facts even when a parent is inactive; unlike selection it
		// does not infer new eligibility or recapture adoption evidence.

		heads, e := u.PathHeads(ctx, r.Data.Structure)
		if e != nil {
			return nil, e
		}
		cached := &cachedUnit{Unit: u, areas: map[string]State{}, countries: map[string]ca.State{}}
		if e = cached.pin(ctx, heads); e != nil {
			return nil, e
		}
		u = cached
		a = d.Adoption{Date: day, Country: d.Reference{ObjectID: heads.CountryID, SourceVersion: heads.CountrySource}, Parents: []d.Reference{}}
		for i, id := range heads.ParentIDs {
			a.Parents = append(a.Parents, d.Reference{ObjectID: id, SourceVersion: heads.ParentSources[i]})
		}

	}
	path, err := projectPath(ctx, u, a, exact, today)
	if err != nil || len(path) == 0 {
		return path, err
	}
	self := org.GeoAreaPathNode{ID: t.ID, Name: r.Data.Name}
	if slices.Contains(fields, "code") {
		self.Code = t.Origin.Code
	}
	if slices.Contains(fields, "level") {
		self.Level = t.Origin.Level
	}
	if exact {
		self.SourceVersion, self.RecordID, self.Revision = t.Source, r.ID, r.Revision
	}
	return append(path, self), nil
}

func recordCountryName(ctx context.Context, u Unit, r d.Record, exact bool, today string, path []org.GeoAreaPathNode) (*string, error) {
	if len(path) > 0 {
		return ptr(path[0].Name), nil
	}
	adoption := r.Data.Adoption
	if !exact {
		adoption.Country = d.Reference{ObjectID: r.Data.Structure.CountryID}
		adoption.Date = pathDay(r, false, today)
	}
	country, e := projectCountry(ctx, u, adoption, exact, today)
	if e != nil || country == nil {
		return nil, e
	}
	return ptr(country.Name), nil
}

// The country label has its own reference and name permissions. It can remain
// readable when the caller cannot see the whole ancestor chain.
func projectCountry(ctx context.Context, u ReferenceUnit, a d.Adoption, exact bool, today string) (*org.GeoAreaPathNode, error) {
	c, e := u.Country(ctx, a.Country.ObjectID, a.Country.SourceVersion)
	if e != nil {
		return nil, e
	}
	cr := cd.Kernel.FactAt(c.Timeline.Records, a.Date)
	if cr == nil {
		return nil, nil
	}
	current, e := u.Country(ctx, a.Country.ObjectID, "")
	if e != nil {
		return nil, e
	}
	cf := cd.Kernel.FactAt(current.Timeline.Records, today)
	fields, e := referenceFields(ctx, u, "country", exact || cf == nil || cf.ID != cr.ID)
	if e != nil {
		var f *foundation.Fault
		if errors.As(e, &f) && f.Status == 403 {
			return nil, nil
		}
		return nil, e
	}
	if !slices.Contains(fields, "name") {
		return nil, nil
	}
	if exact && (cr.ID != a.Country.RecordID || cr.Revision != a.Country.Revision) {
		return nil, refUnavailable()
	}
	cn := org.GeoAreaPathNode{ID: c.Timeline.ID, Name: cr.Data.Name}
	if slices.Contains(fields, "code") {
		cn.Code = c.Timeline.Origin.Code
	}
	if exact {
		cn.SourceVersion = a.Country.SourceVersion
		cn.RecordID = a.Country.RecordID
		cn.Revision = a.Country.Revision
	}
	return &cn, nil
}

func projectPath(ctx context.Context, u ReferenceUnit, a d.Adoption, exact bool, today string) ([]org.GeoAreaPathNode, error) {
	out := []org.GeoAreaPathNode{}
	country, err := projectCountry(ctx, u, a, exact, today)
	if err != nil || country == nil {
		return out, err
	}
	out = append(out, *country)
	for _, ref := range a.Parents {
		v, e := u.Load(ctx, ref.ObjectID, ref.SourceVersion)
		if e != nil {
			return nil, e
		}
		r := d.Kernel.FactAt(v.Timeline.Records, a.Date)
		if r == nil {
			return []org.GeoAreaPathNode{}, nil
		}
		current, e := u.Load(ctx, ref.ObjectID, "")
		if e != nil {
			return nil, e
		}
		cf := d.Kernel.FactAt(current.Timeline.Records, today)
		fields, e := referenceFields(ctx, u, "geoarea", exact || cf == nil || cf.ID != r.ID)
		if e != nil {
			var f *foundation.Fault
			if errors.As(e, &f) && f.Status == 403 {
				return []org.GeoAreaPathNode{}, nil
			}
			return nil, e
		}
		if !slices.Contains(fields, "name") || !slices.Contains(fields, "geoAreaPath") {
			return []org.GeoAreaPathNode{}, nil
		}
		if exact && (r.ID != ref.RecordID || r.Revision != ref.Revision) {
			return nil, refUnavailable()
		}
		n := org.GeoAreaPathNode{ID: v.Timeline.ID, Name: r.Data.Name}
		if slices.Contains(fields, "code") {
			n.Code = v.Timeline.Origin.Code
		}
		if slices.Contains(fields, "level") {
			n.Level = v.Timeline.Origin.Level
		}
		if exact {
			n.SourceVersion = ref.SourceVersion
			n.RecordID = ref.RecordID
			n.Revision = ref.Revision
		}
		out = append(out, n)
	}
	return out, nil
}
