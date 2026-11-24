package public

import (
	"slices"
	"time"
)

type PersonScope struct {
	All              bool
	Include, Exclude []string
}

func (s PersonScope) Allows(id string) bool {
	return (s.All || slices.Contains(s.Include, id)) && !slices.Contains(s.Exclude, id)
}

func (s PersonScope) Available() bool {
	if s.All {
		return true
	}
	for _, id := range s.Include {
		if !slices.Contains(s.Exclude, id) {
			return true
		}
	}
	return false
}

type PersonPolicy struct {
	MaterialDays, MaterialEvidenceDays, ContactDays, ContactEvidenceDays int
	Version                                                              string
	OrdinaryDays, EvidenceDays                                           int
	ValidUntil                                                           time.Time
	AllowPermanentDecision                                               bool
}
type PersonGrant struct {
	Action, Purpose, Source, ItemID string
	Fields, IDs                     []string
	Denied                          bool
	Expires                         time.Time
}
type PersonAccess struct {
	Principal Principal
	Policy    PersonPolicy
	Grants    []PersonGrant
	At        time.Time
}

func PersonPurpose(action string) string {
	switch action {
	case "hire.check":
		return "hire"
	case "family.read", "family.correct":
		return "family-contact"
	case "profile.read", "profile.correct":
		return "personal-profile"
	case "read":
		return "directory"
	case "verify", "register":
		return "registration"
	case "correct":
		return "maintenance"
	case "renumber", "renumber.decide":
		return "number-decision"
	case "history.read":
		return "history"
	case "evidence.read":
		return "evidence"
	case "hold.read":
		return "preservation"
	}
	return ""
}

// A single allow must cover every requested field. A denial of any requested
// field removes its entire scope from this projection.
func (a PersonAccess) Scope(action string, fields ...string) PersonScope {
	return a.scope("", "", action, fields...)
}
func (a PersonAccess) EvidenceAllowed(id, version, action, field string) bool {
	return version != "" && a.scope(version, "", action, field).Allows(id)
}
func (a PersonAccess) ItemAllowed(person, item, action string, fields ...string) bool {
	return a.scope("", item, action, fields...).Allows(person)
}
func (a PersonAccess) ItemWritable(person, item string, fields ...string) bool {
	return a.scope("", item, "profile.correct", fields...).Allows(person)
}
func (a PersonAccess) scope(version, item, action string, fields ...string) PersonScope {
	s := PersonScope{Include: []string{}, Exclude: []string{}}
	for _, g := range a.Grants {
		if g.ItemID != "" && g.ItemID != item {
			continue
		}
		if g.Source != version && !(g.Denied && g.Source == "") {
			continue
		}
		if g.Action != action || g.Purpose != PersonPurpose(action) || (!g.Expires.IsZero() && !a.At.Before(g.Expires)) {
			continue
		}
		complete, overlap := true, len(g.Fields) == 0
		for _, f := range fields {
			has := slices.Contains(g.Fields, f)
			complete = complete && has
			overlap = overlap || has
		}
		if g.Denied && overlap {
			if g.IDs == nil {
				return PersonScope{}
			}
			s.Exclude = append(s.Exclude, g.IDs...)
		} else if !g.Denied && complete {
			if g.IDs == nil {
				s.All = true
			} else {
				s.Include = append(s.Include, g.IDs...)
			}
		}
	}
	return s
}

// Explicit synthetic profile grants use the same field identifiers as the command and projection.
var PersonalProfileFields = []string{"formerName", "englishName", "birthDate", "gender", "maritalStatus", "nationality", "ethnicity", "nativePlace", "birthPlace", "householdCategory", "householdNature", "politicalStatus", "documentKind", "documentName", "issuingCountry", "issuer", "issueDate", "expiryDate", "longTerm", "documentPrimary", "contactPurpose", "contactPrimary"}
