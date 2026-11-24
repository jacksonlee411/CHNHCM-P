package postgres

import (
	"slices"

	d "chnhcm.local/chnhcm/internal/contexts/personnel/domain"
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
)

func (r *request) healthWritable(id string) bool {
	return r.allowed("health.read", id, "healthSummary") && r.allowed("health.correct", id, "healthSummary")
}

func healthIndex(v snapshot) int {
	return slices.IndexFunc(v.profileItems, func(x profileItem) bool { return x.Section == "health" })
}

func (r *request) projectHealth(v snapshot, out *p.Detail) {
	if !r.allowed("health.read", v.ID, "healthSummary") {
		return
	}
	out.Readable = append(out.Readable, "healthSummary")
	value := ""
	if i := healthIndex(v); i >= 0 {
		value = v.profileItems[i].Fields["healthSummary"]
	}
	out.HealthSummary = &value
}

func (r *request) applyHealth(v *snapshot, input *string) error {
	if input == nil {
		return nil
	}
	if !r.healthWritable(v.ID) {
		return d.Denied()
	}
	value, err := d.ProfileText("healthSummary", *input)
	if err != nil {
		return err
	}
	i := healthIndex(*v)
	if i < 0 {
		if value == "" {
			return nil
		}
		terms := r.a.Purposes["health-summary"]
		v.profileItems = append(v.profileItems, profileItem{
			ProfileItem: p.ProfileItem{ItemID: f.NewID(), Fields: map[string]string{}, Adoptions: map[string]p.CountryAdoption{}},
			Section:     "health", ordinary: d.Cutoff(r.a.At, terms.OrdinaryDays), evidence: d.Cutoff(r.a.At, terms.EvidenceDays), policy: terms.Version,
		})
		i = len(v.profileItems) - 1
	}
	row := &v.profileItems[i]
	if row.Fields["healthSummary"] != value {
		row.Fields["healthSummary"] = value
		row.Revision = f.NewID()
	}
	return nil
}
