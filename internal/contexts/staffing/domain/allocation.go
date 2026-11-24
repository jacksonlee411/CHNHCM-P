package domain

import (
	"math/big"
	"slices"
	"strings"
)

type AllocatedPosition struct {
	Position QuotaPeriods
	Quota    StaffingQuota
}

// AllocationDates returns the boundaries at which this position enters or
// leaves the active roster, or an independent quota record takes effect there.
func (p AllocatedPosition) AllocationDates(department string, includeQuota bool) []string {
	var dates []string
	member := false
	for i, r := range p.Position.Records {
		active := r.Active && r.DepartmentID == department
		if active != member {
			dates = append(dates, r.Date)
		}
		member = active
		if member && r.End != "" && (i+1 == len(p.Position.Records) || p.Position.Records[i+1].Date != r.End) {
			dates = append(dates, r.End)
			member = false
		}
	}
	if includeQuota {
		for _, r := range p.Quota.Records {
			j := p.Position.At(r.Date)
			if j >= 0 && p.Position.Records[j].Active && p.Position.Records[j].DepartmentID == department {
				dates = append(dates, r.Date)
			}
		}
	}
	return dates
}

type AllocationConflict struct{ DepartmentID, Day, Shortfall string }

// A position may change department, become inactive, or acquire an independent
// allocation after the submitted date. Each boundary can change the subtotal.
func DepartmentCapacity(department string, limit StaffingQuota, positions []AllocatedPosition) *AllocationConflict {
	if len(limit.Records) == 0 {
		return nil
	}
	var dates []string
	for _, r := range limit.Records {
		dates = append(dates, r.Date)
	}
	for _, p := range positions {
		for _, r := range p.Position.Records {
			dates = append(dates, r.Date)
		}
		for _, r := range p.Quota.Records {
			dates = append(dates, r.Date)
		}
	}
	slices.Sort(dates)
	for _, day := range slices.Compact(dates) {
		index := limit.At(day)
		if index < 0 || limit.Records[index].Amount == nil {
			continue
		}
		var sum big.Int
		for _, p := range positions {
			i, j := p.Position.At(day), p.Quota.At(day)
			if i < 0 || j < 0 {
				continue
			}
			record := p.Position.Records[i]
			if n := p.Quota.Records[j].Amount; record.Active && record.DepartmentID == department && n != nil {
				sum.Add(&sum, big.NewInt(int64(*n)))
			}
		}
		deficit := sum.Sub(&sum, big.NewInt(int64(*limit.Records[index].Amount)))
		if deficit.Sign() > 0 {
			return &AllocationConflict{department, day, deficit.String()}
		}
	}
	return nil
}

// QuotaPeriods contains only the identity and dated facts needed for allocation.
type QuotaPeriods struct {
	Kind             Kind
	ID, Code, Source string
	Records          []QuotaPeriod
}
type QuotaPeriod struct {
	Date, End, Name, DepartmentID string
	Active                        bool
}

func (p QuotaPeriods) At(day string) int {
	i, found := slices.BinarySearchFunc(p.Records, day, func(r QuotaPeriod, day string) int { return strings.Compare(r.Date, day) })
	if !found {
		i--
	}
	if i >= 0 && p.Records[i].End != "" && day >= p.Records[i].End {
		return -1
	}
	return i
}
func (p QuotaPeriods) Departments() []string {
	var ids []string
	for _, r := range p.Records {
		if r.DepartmentID != "" {
			ids = append(ids, r.DepartmentID)
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}
func QuotaEligibility(t Timeline) QuotaPeriods {
	p := QuotaPeriods{Kind: t.Kind, ID: t.ID, Code: t.Code, Source: t.Source}
	for _, r := range t.Records {
		p.Records = append(p.Records, QuotaPeriod{Date: r.Date, End: r.End, Name: r.Data["name"], DepartmentID: r.Data["departmentId"], Active: r.Active()})
	}
	return p
}
