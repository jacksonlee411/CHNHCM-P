package geoarea

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	ca "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	cd "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type CandidateRow struct {
	Source, CountryID, CountrySource string
	ParentIDs, ParentSources         []string
	ID, Code                         string
	Terminal                         bool
}
type cachedUnit struct {
	clock *time.Time
	Unit
	areas      map[string]State
	countries  map[string]ca.State
	readFields map[referenceFieldKey][]string
}

func (u *cachedUnit) Load(ctx context.Context, id, source string) (State, error) {
	k := id + source
	if v, ok := u.areas[k]; ok {
		return v, nil
	}
	v, e := u.Unit.Load(ctx, id, source)
	if e == nil {
		u.areas[k] = v
	}
	return v, e
}
func (u *cachedUnit) Country(ctx context.Context, id, source string) (ca.State, error) {
	k := id + source
	if v, ok := u.countries[k]; ok {
		return v, nil
	}
	v, e := u.Unit.Country(ctx, id, source)
	if e == nil {
		u.countries[k] = v
	}
	return v, e
}
func (s Service) Candidates(ctx context.Context, c access.Credentials, in org.GeoAreaCandidateInput) (out org.GeoAreaCandidates, err error) {
	in.Q = strings.TrimSpace(in.Q)
	if in.PageSize == 0 {
		in.PageSize = 20
	}
	if !foundation.ValidID(in.CountryRegionID) || in.ParentAreaID != "" && !foundation.ValidID(in.ParentAreaID) || !foundation.ValidDate(in.AdoptedDate) || !slices.Contains([]int{20, 50, 100}, in.PageSize) || utf8.RuneCountInString(in.Q) > 200 {
		return out, fault("M02_INVALID_INPUT", "", "候选条件无效")
	}
	out = org.GeoAreaCandidates{Mode: "areas", Items: []org.GeoAreaCandidate{}}
	err = s.read(ctx, c, func(base Unit) error {
		u := &cachedUnit{Unit: base, areas: map[string]State{}, countries: map[string]ca.State{}}
		fields, e := readable(ctx, u, false)
		if e != nil {
			return e
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		today := now.In(foundation.Shanghai).Format("2006-01-02")
		country, e := u.Country(ctx, in.CountryRegionID, "")
		if e != nil {
			return e
		}
		if country.Timeline.ID == "" {
			return missing()
		}
		cr := cd.Kernel.FactAt(country.Timeline.Records, in.AdoptedDate)
		cc := cd.Kernel.FactAt(country.Timeline.Records, today)
		if cr == nil || cr.State != "active" {
			return refUnavailable()
		}
		cf, e := referenceFields(ctx, u, "country", cc == nil || cr.ID != cc.ID)
		if e != nil {
			return e
		}
		if e = required(cf, countryRequired); e != nil {
			return e
		}
		if country.Timeline.Origin.Code == "MO" {
			if in.ParentAreaID != "" || in.Cursor != "" {
				return fault("M02_INVALID_INPUT", "", "基础地址范围无区域分页")
			}
			out.Mode = "basic_address"
			return nil
		}
		if e = required(fields, areaRequired); e != nil {
			return e
		}
		ok, e := u.RegionPublished(ctx, in.CountryRegionID)
		if e != nil {
			return e
		}
		if !ok {
			return fault("M02_CATALOG_NOT_READY", "", "该地区资料尚未就绪")
		}
		if in.ParentAreaID != "" {
			parent, e := u.Load(ctx, in.ParentAreaID, "")
			if e != nil {
				return e
			}
			if parent.Timeline.ID == "" || parent.Timeline.Origin.CountryID != in.CountryRegionID {
				return fault("M02_INVALID_INPUT", "parentAreaId", "上级不属于所选地区")
			}
		}
		p := u.Access().Principal()
		q := in
		q.Cursor = ""
		bind, _ := json.Marshal(struct {
			Query  org.GeoAreaCandidateInput
			Grants []access.Grant
		}{q, u.Access().Grants()})
		digest := sha256.Sum256(bind)
		binding := queryCursor{Kind: "geoarea", Instance: p.InstanceID, Subject: p.ID, Session: p.SessionID, Day: today, Query: fmt.Sprintf("%x", digest), Source: fmt.Sprint(u.Access().PolicyVersion())}
		key, id := "", ""
		if in.Cursor != "" {
			key, id, e = binding.decode(in.Cursor)
			if e != nil {
				return e
			}
		}
		lastKey, lastID := "", ""
		for {
			rows, e := u.CandidateRows(ctx, in, key, id, in.PageSize+1)
			if e != nil {
				return e
			}
			if e := u.prefetch(ctx, rows); e != nil {
				return e
			}
			for _, row := range rows {
				key, id = row.Code, row.ID
				if e := u.pin(ctx, row); e != nil {
					return e
				}
				v, e := u.Load(ctx, row.ID, "")
				if e != nil {
					return e
				}
				st := v.Timeline.Origin.Structure
				st.ParentID = row.ID
				a, e := adopt(ctx, u, st, in.AdoptedDate, false)
				if e != nil {
					var f *foundation.Fault
					if errors.As(e, &f) && (f.Code == "ACCESS_DENIED" || f.Code == "M02_REFERENCE_UNAVAILABLE") {
						continue
					}
					return e
				}
				if len(out.Items) == in.PageSize {
					out.HasMore = true
					out.Cursor = ptr(binding.encode(lastKey, lastID))
					return nil
				}
				r := d.Kernel.FactAt(v.Timeline.Records, in.AdoptedDate)
				path, e := projectPath(ctx, u, a, false, today)
				if e != nil {
					return e
				}
				current := d.Kernel.FactAt(v.Timeline.Records, today)
				visible, e := referenceFields(ctx, u, "geoarea", current == nil || current.ID != r.ID)
				if e != nil {
					return e
				}
				if len(path) != 1+len(a.Parents) {
					continue
				}
				path[0].SourceVersion = a.Country.SourceVersion
				for i, ref := range a.Parents {
					path[i+1].SourceVersion = ref.SourceVersion
				}
				item := org.GeoAreaCandidate{ID: row.ID, Code: row.Code, Name: r.Data.Name, Level: st.Level, Path: path, Terminal: row.Terminal}
				if slices.Contains(visible, "nature") {
					item.Nature = ptr(st.Nature)
				}
				if slices.Contains(visible, "scheme") {
					item.Scheme = ptr(st.Scheme)
				}
				if slices.Contains(visible, "nameEn") {
					item.NameEn = r.Data.NameEn
				}
				out.Items = append(out.Items, item)
				lastKey, lastID = key, id
			}
			if len(rows) < in.PageSize+1 {
				return nil
			}
		}
	})
	return
}

func (u *cachedUnit) Now(ctx context.Context) (time.Time, error) {
	if u.clock != nil {
		return *u.clock, nil
	}
	now, e := u.Unit.Now(ctx)
	if e == nil {
		u.clock = &now
	}
	return now, e
}

// The PostgreSQL cursor captures all heads in one read snapshot. Later decoding
// addresses immutable sources, never a newer ancestor head.
func (u *cachedUnit) pin(ctx context.Context, row CandidateRow) error {
	ids := slices.Clone(row.ParentIDs)
	sources := slices.Clone(row.ParentSources)
	if row.ID != "" {
		ids = append(ids, row.ID)
		sources = append(sources, row.Source)
	}
	if len(ids) != len(sources) {
		return refUnavailable()
	}
	for i, id := range ids {
		key := id + sources[i]
		v, ok := u.areas[key]
		if !ok {
			var e error
			v, e = u.Unit.Load(ctx, id, sources[i])
			if e != nil {
				return e
			}
			u.areas[key] = v
		}
		u.areas[id] = v
	}
	key := row.CountryID + row.CountrySource
	c, ok := u.countries[key]
	if !ok {
		if current, hit := u.countries[row.CountryID]; hit && current.Timeline.Source == row.CountrySource {
			c = current
			ok = true
		}
	}
	if !ok {
		var e error
		c, e = u.Unit.Country(ctx, row.CountryID, row.CountrySource)
		if e != nil {
			return e
		}
	}
	u.countries[key] = c
	u.countries[row.CountryID] = c
	return nil
}

// Only this streamed batch and its finite ancestors are read together. The
// cursor's captured sources still win if a head changes before decoding.
func (u *cachedUnit) prefetch(ctx context.Context, rows []CandidateRow) error {
	want := map[string]string{}
	for _, row := range rows {
		if len(row.ParentIDs) != len(row.ParentSources) {
			return refUnavailable()
		}
		for i, id := range row.ParentIDs {
			if _, ok := u.areas[id+row.ParentSources[i]]; !ok {
				want[id] = row.ParentSources[i]
			}
		}
		if _, ok := u.areas[row.ID+row.Source]; !ok {
			want[row.ID] = row.Source
		}
	}
	if len(want) == 0 {
		return nil
	}
	values, e := u.Unit.LoadSources(ctx, want)
	if e != nil {
		return e
	}
	for id, source := range want {
		v, ok := values[id]
		if !ok || v.Timeline.Source != source {
			return refUnavailable()
		}
		u.areas[id+source] = v
	}
	return nil
}
