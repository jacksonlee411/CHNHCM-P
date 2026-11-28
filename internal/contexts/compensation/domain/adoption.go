package domain

import "chnhcm.local/chnhcm/internal/foundation"

// GradeTerms pins the authority and label as seen at initial adoption.
// Subsequent reads of history never resolve that evidence against today's head.
type GradeTerms struct {
	BusinessAt string `json:"businessAt"`
	Source     string `json:"sourceVersion"`
	Record     string `json:"recordId"`
	Revision   string `json:"revision"`
	Name       string `json:"name"`
	Code       string `json:"code"`
}
type RateTerms struct {
	Amounts Amounts    `json:"amounts"`
	Grade   GradeTerms `json:"grade"`
}
type GradeSegment struct {
	foundation.Period
	Active   bool
	Evidence GradeTerms
}

func CoverGrade(period foundation.Period, segments []GradeSegment) (GradeTerms, error) {
	var first GradeTerms
	cursor := period.From
	for _, seg := range segments {
		if !seg.Period.Includes(cursor) {
			continue
		}
		if !seg.Active {
			break
		}
		if first.Record == "" {
			first = seg.Evidence
			first.BusinessAt = period.From
		}
		if seg.Until == "" || period.Until != "" && seg.Until >= period.Until {
			return first, nil
		}
		cursor = seg.Until
	}
	return GradeTerms{}, Invalid("effectiveDate", cursor+" 起所选职级不能覆盖本条标准")
}
