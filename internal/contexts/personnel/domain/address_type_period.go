package domain

import f "chnhcm.local/chnhcm/internal/foundation"

// AddressTypePeriod describes the single interval supported by the personnel catalog.
// ConfirmedFrom and UndatedStop distinguish a migrated switch from a dated decision.
type AddressTypePeriod struct {
	FirstDay, StopDay, ConfirmedFrom string
	UndatedStop                      bool
}

func (p AddressTypePeriod) State(day string) string {
	switch {
	case day < p.FirstDay:
		return "not_effective"
	case day < p.ConfirmedFrom:
		return "unknown"
	case p.UndatedStop || p.StopDay != "" && day >= p.StopDay:
		return "inactive"
	default:
		return "active"
	}
}

// LegacySelectable projects the original switch, independently of the first-day gate.
func (p AddressTypePeriod) LegacySelectable(day string) bool {
	return !p.UndatedStop && (p.StopDay == "" || (p.StopDay != p.FirstDay && day < p.StopDay))
}

func (p AddressTypePeriod) WithStop(day string) (AddressTypePeriod, error) {
	if day != "" && (!f.ValidDate(day) || day < p.FirstDay || day < p.ConfirmedFrom) {
		return p, Invalid("stopDay", "停用生效日须为有效日期，且不得早于首次生效日和可核实期间的起点")
	}
	p.StopDay, p.UndatedStop = day, false
	return p, nil
}

// Dates are the boundaries whose business meaning can actually be explained.
func (p AddressTypePeriod) Dates() []string {
	days := []string{p.FirstDay}
	if p.ConfirmedFrom > p.FirstDay {
		days = append(days, p.ConfirmedFrom)
	}
	if p.StopDay != "" && p.StopDay != days[len(days)-1] {
		days = append(days, p.StopDay)
	}
	return days
}
