package postgres

import (
	d "chnhcm.local/chnhcm/internal/contexts/personnel/domain"
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
)

type profileItem struct {
	p.ProfileItem
	Section            string
	ordinary, evidence time.Time
	policy             string
}

func (r *request) itemRead(s d.ItemSection, id, item string, k string) bool {
	return r.a.ItemAllowed(id, item, s.Action+".read", k)
}
func (r *request) itemWrite(s d.ItemSection, id, item string, k string) bool {
	return r.itemRead(s, id, item, k) && r.a.ItemAllowed(id, item, s.Action+".correct", k)
}
func (r *request) removable(s d.ItemSection, id string, row profileItem) bool {
	for k := range row.Fields {
		if !r.itemWrite(s, id, row.ItemID, k) {
			return false
		}
	}
	return true
}
func (r *request) projectItems(v snapshot, out *p.Detail, view string) {
	for _, s := range d.ItemSections {
		if !r.itemRead(s, v.ID, "", s.Required[0]) {
			continue
		}
		for _, k := range s.Fields {
			if r.itemRead(s, v.ID, "", k) || itemMaskable(k) && r.itemRead(s, v.ID, "", k+".masked") {
				out.Readable = append(out.Readable, k)
			}
		}
		for _, row := range v.profileItems {
			if row.Section != s.Name || !r.itemRead(s, v.ID, row.ItemID, s.Required[0]) {
				continue
			}
			x := p.ProfileItem{ItemID: row.ItemID, Revision: row.Revision, Fields: map[string]string{}, Adoptions: map[string]p.CountryAdoption{}}
			for _, k := range s.Fields {
				full := r.itemRead(s, v.ID, row.ItemID, k)
				if full {
					if val, ok := row.Fields[k]; ok {
						x.Fields[k] = val
						if k == "certificateNumber" && view != "credential-"+row.ItemID && !(view == "edit" && r.itemWrite(s, v.ID, row.ItemID, k)) {
							x.Fields[k] = "********"
						}
					}
					if a, ok := row.Adoptions[k]; ok {
						a.Parents = nil
						x.Adoptions[k] = a
					}
				} else if itemMaskable(k) && r.itemRead(s, v.ID, row.ItemID, k+".masked") {
					x.Masked = append(x.Masked, k)
					if val := row.Fields[k]; val != "" {
						if k == "certificateNumber" {
							x.Fields[k] = "********"
						} else {
							x.Fields[k] = d.Mask(familyChannel(k), val)
						}
					}
				} else if slices.Contains(out.Readable, k) {
					x.Hidden = append(x.Hidden, k)
				}
				if view == "edit" && r.itemWrite(s, v.ID, "", k) && !r.itemWrite(s, v.ID, row.ItemID, k) {
					x.Locked = append(x.Locked, k)
				}
			}
			x.CanRemove = view == "edit" && r.removable(s, v.ID, row)
			switch s.Name {
			case "education":
				out.Education = append(out.Education, x)
			case "credentials":
				out.Credentials = append(out.Credentials, x)
			case "addresses":
				out.Addresses = append(out.Addresses, x)
			case "family":
				out.Family = append(out.Family, x)
			}
		}
	}
}
func itemMaskable(key string) bool { return key == "certificateNumber" || familyChannel(key) != "" }

func familyChannel(k string) string {
	switch k {
	case "familyPhone":
		return "phone"
	case "familyEmail":
		return "email"
	}
	return ""
}

// Replaying a rejected submission can expose its original field error only while the
// actor can still see that row and perform the affected edits.
func (r *request) checkItemFault(id string, in p.Correct, field string) error {
	group, rest, rowField := strings.Cut(field, ".")
	var section d.ItemSection
	var edits []p.ProfileItemEdit
	switch group {
	case "addresses":
		section, edits = d.Addresses, in.Addresses
	case "family":
		section, edits = d.Family, in.Family
	case "education":
		section, edits = d.Education, in.Education
	case "credentials":
		section, edits = d.Credentials, in.Credentials
	default:
		return nil
	}
	if !rowField {
		if !r.itemRead(section, id, "", section.Required[0]) {
			return d.Denied()
		}
		return nil
	}
	index, key, located := strings.Cut(rest, ".")
	n, err := strconv.Atoi(index)
	if !located || err != nil || n < 0 || n >= len(edits) {
		return d.Denied()
	}
	edit := edits[n]
	if !r.itemRead(section, id, edit.ItemID, section.Required[0]) || !r.itemRead(section, id, edit.ItemID, key) {
		return d.Denied()
	}
	for k := range edit.Fields {
		if !r.itemWrite(section, id, edit.ItemID, k) {
			return d.Denied()
		}
	}
	for k := range edit.Choices {
		if !r.itemWrite(section, id, edit.ItemID, k) {
			return d.Denied()
		}
	}
	if edit.ItemID == "" {
		for _, required := range section.Required {
			if !r.itemWrite(section, id, "", required) {
				return d.Denied()
			}
		}
	} else {
		current, err := r.load(id, "")
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(current.profileItems, func(x profileItem) bool {
			return x.Section == group && x.ItemID == edit.ItemID
		}) {
			return d.Denied()
		}
	}
	if group == d.Family.Name && familyChannel(key) != "" && !r.itemRead(section, id, edit.ItemID, "emergency") {
		return d.Invalid(field, "请核对联系方式")
	}
	return nil
}

func (r *request) applyItems(v *snapshot, s d.ItemSection, edits []p.ProfileItemEdit) error {
	if len(edits) == 0 {
		return nil
	}
	seen := map[string]bool{}
	selected := map[string]string{}
	for n, edit := range edits {
		idx := slices.IndexFunc(v.profileItems, func(x profileItem) bool { return x.Section == s.Name && x.ItemID == edit.ItemID })
		if edit.ItemID == "" {
			if edit.Remove || edit.ExpectedRevision != "" {
				return d.Invalid(s.Name, "请选择已有记录")
			}
			for _, k := range s.Required {
				if !r.itemWrite(s, v.ID, "", k) {
					return d.Denied()
				}
			}
			ordinary, evidence := r.a.Policy.OrdinaryDays, r.a.Policy.EvidenceDays
			if s.Name == d.Family.Name {
				ordinary, evidence = r.a.Policy.ContactDays, r.a.Policy.ContactEvidenceDays
			}
			row := profileItem{
				ProfileItem: p.ProfileItem{ItemID: f.NewID(), Fields: map[string]string{}, Adoptions: map[string]p.CountryAdoption{}},
				Section:     s.Name, policy: r.a.Policy.Version, ordinary: d.Cutoff(r.a.At, ordinary), evidence: d.Cutoff(r.a.At, evidence),
			}
			v.profileItems = append(v.profileItems, row)
			idx = len(v.profileItems) - 1
		} else {
			if idx < 0 || !r.itemRead(s, v.ID, edit.ItemID, s.Required[0]) {
				return d.Denied()
			}
			if seen[edit.ItemID] || edit.ExpectedRevision != v.profileItems[idx].Revision {
				return d.Conflict()
			}
			seen[edit.ItemID] = true
		}
		row := &v.profileItems[idx]
		if edit.Remove {
			if len(edit.Fields)+len(edit.Choices) != 0 {
				return d.Invalid(s.Name, "移除时只需选择原记录")
			}
			if !r.removable(s, v.ID, *row) {
				return d.Denied()
			}
			v.profileItems = slices.Delete(v.profileItems, idx, idx+1)
			continue
		}
		before := hash(row.ProfileItem)
		if err := r.editItemFields(s, v.ID, n, row, edit); err != nil {
			return err
		}
		for _, flag := range []string{"addressPrimary", "highestEducation", "highestDegree"} {
			if edit.Fields[flag] != "yes" {
				continue
			}
			if selected[flag] != "" {
				return rowInputError(s.Name, n, flag, d.Invalid(flag, "同一标记只能指定一条记录"))
			}
			selected[flag] = row.ItemID
		}
		if before != hash(row.ProfileItem) {
			row.Revision = f.NewID()
		}
	}
	for flag, primary := range selected {
		for i := range v.profileItems {
			x := &v.profileItems[i]
			if x.Section == s.Name && x.ItemID != primary && x.Fields[flag] == "yes" {
				if !r.itemRead(s, v.ID, x.ItemID, s.Required[0]) || !r.itemWrite(s, v.ID, x.ItemID, flag) {
					return d.Denied()
				}
				delete(x.Fields, flag)
				x.Revision = f.NewID()
			}
		}
	}
	if s.Name == d.Education.Name {
		facts := map[string]bool{}
		for _, row := range v.profileItems {
			if row.Section != s.Name {
				continue
			}
			fields := maps.Clone(row.Fields)
			delete(fields, "highestEducation")
			delete(fields, "highestDegree")
			key := hash(fields)
			if facts[key] {
				return d.Conflict()
			}
			facts[key] = true
		}
	}
	return nil
}
func placeChanged() error {
	return f.Fail(409, "PERSON_PLACE_CHANGED", "地址目录或上下级已变化，请重新选择")
}
func placeError(n int, k string, err error) error {
	var fault *f.Fault
	if errors.As(err, &fault) {
		if fault.Status == 403 {
			return &f.Fault{Status: 403, Code: "PERSON_PLACE_DENIED", Field: fmt.Sprintf("addresses.%d.%s", n, k), Message: "所选目录项已不可读取，请重新核对地址"}
		}
		if fault.Status == 404 || fault.Status == 409 {
			err = placeChanged()
		}
	}
	return rowInputError("addresses", n, k, err)
}
func (r *request) resolvePlace(k string, c p.PlaceChoice, existing map[string]p.CountryAdoption) (*p.CountryAdoption, string, error) {
	if k == "addressCountry" {
		if len(c.Ancestors) != 0 {
			return nil, "", placeChanged()
		}
		return r.resolveCountry(p.CountryChoice{ID: c.ID, SourceVersion: c.SourceVersion})
	}
	if !f.ValidID(c.ID) || !f.ValidID(c.SourceVersion) {
		return nil, "", d.Invalid(k, "请选择有效地区")
	}
	path, name, err := r.adoptArea(r.ctx, r.tx, r.guard, c.ID, r.a.At.In(f.Shanghai).Format("2006-01-02"))
	if err != nil {
		return nil, "", err
	}
	// Previously saved parents keep their own adoption. Compare their identities for containment,
	// and compare all current ancestor versions to those of this new candidate below.
	if len(path) < 2 || len(c.Ancestors) != len(path)-1 || path[0].ID != existing["addressCountry"].ID {
		return nil, "", placeChanged()
	}
	leaf := path[len(path)-1]
	if leaf.ID != c.ID || leaf.SourceVersion != c.SourceVersion || k == "province" && len(path) != 2 {
		return nil, "", placeChanged()
	}
	if k == "area" && (len(path) < 3 || path[1].ID != existing["province"].ID) {
		return nil, "", placeChanged()
	}
	for i, parent := range c.Ancestors {
		if path[i].ID != parent.ID || path[i].SourceVersion != parent.SourceVersion {
			return nil, "", placeChanged()
		}
		leaf.Parents = append(leaf.Parents, parent)
	}
	return &leaf, name, nil
}

func (r *request) editItemFields(s d.ItemSection, id string, n int, row *profileItem, edit p.ProfileItemEdit) error {
	for k, value := range edit.Fields {
		if !slices.Contains(s.Fields, k) || !r.itemWrite(s, id, row.ItemID, k) {
			return d.Denied()
		}
		val, err := d.ProfileText(k, value)
		if err != nil {
			return rowInputError(s.Name, n, k, err)
		}
		if slices.Contains([]string{"emergency", "addressPrimary", "highestEducation", "highestDegree", "indefinite"}, k) && val != "" && val != "yes" {
			return rowInputError(s.Name, n, k, d.Invalid(k, "请选择有效标记"))
		}
		delete(row.Adoptions, k)
		if val == "" {
			delete(row.Fields, k)
		} else {
			row.Fields[k] = val
		}
	}
	for k := range edit.Choices {
		if s.Name != "addresses" || !slices.Contains([]string{"addressCountry", "province", "area"}, k) || !r.itemWrite(s, id, row.ItemID, k) {
			return d.Denied()
		}
		if _, raw := edit.Fields[k]; raw {
			return rowInputError(s.Name, n, k, d.Invalid(k, "请选择目录项或输入原文"))
		}
	}
	for _, k := range []string{"addressCountry", "province", "area"} {
		c, ok := edit.Choices[k]
		if !ok {
			continue
		}
		a, name, err := r.resolvePlace(k, c, row.Adoptions)
		if err != nil {
			return placeError(n, k, err)
		}
		row.Adoptions[k] = *a
		row.Fields[k] = name
	}
	if s.Name == "addresses" {
		// Changing a parent cannot silently detach an adopted child or turn it into raw text.
		for _, pair := range [][2]string{{"addressCountry", "province"}, {"addressCountry", "area"}, {"province", "area"}} {
			parent, child := pair[0], pair[1]
			a, ok := row.Adoptions[child]
			if !ok {
				continue
			}
			if selected := row.Adoptions[parent]; selected.ID == "" || !slices.ContainsFunc(a.Parents, func(parent p.CountryChoice) bool { return parent.ID == selected.ID }) {
				return placeError(n, parent, placeChanged())
			}
		}
	}
	// A change of type or duration can clear stored fields. Authorize those
	// changes even when the client did not receive the old values.
	for k := range row.Fields {
		obsolete := s.Name == d.Education.Name && !d.EducationFieldApplies(row.Fields["educationType"], k) ||
			s.Name == d.Credentials.Name && k == "validUntil" && row.Fields["indefinite"] == "yes" && edit.Fields["indefinite"] == "yes"
		if !obsolete {
			continue
		}
		if !r.itemWrite(s, id, row.ItemID, k) {
			return d.Denied()
		}
		if edit.Fields[k] != "" {
			return rowInputError(s.Name, n, k, d.Invalid(k, "该字段不适用于当前记录"))
		}
		delete(row.Fields, k)
	}
	if err := s.Validate(row.Fields); err != nil {
		var fault *f.Fault
		if errors.As(err, &fault) {
			if !r.itemRead(s, id, row.ItemID, fault.Field) {
				return d.Denied()
			}
			if fault.Field == "emergency" && !r.itemWrite(s, id, row.ItemID, "emergency") {
				for _, k := range []string{"familyPhone", "familyEmail"} {
					if _, changed := edit.Fields[k]; changed {
						return rowInputError(s.Name, n, k, err)
					}
				}
			}
		}
		return rowInputError(s.Name, n, "", err)
	}
	return nil
}
