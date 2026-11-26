package postgres

import (
	d "chnhcm.local/chnhcm/internal/contexts/personnel/domain"
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"errors"
	"fmt"
	"maps"
	"slices"
)

func (r *request) fieldWrite(id string, fields ...string) bool {
	return r.allowed("profile.read", id, fields...) && r.allowed("profile.correct", id, fields...)
}
func (r *request) rowWrite(id, item string, fields ...string) bool {
	return r.allowed("profile.read", id, fields...) && r.a.ItemWritable(id, item, fields...)
}
func (r *request) lockedFields(id, item string, fields ...string) []string {
	var result []string
	for _, field := range fields {
		if r.fieldWrite(id, field) && !r.a.ItemWritable(id, item, field) {
			result = append(result, field)
		}
	}
	return result
}
func (v *snapshot) primaryIndex() int {
	return slices.IndexFunc(v.docs, func(x p.DocumentRow) bool { return x.Primary })
}
func (v *snapshot) primaryValue() *p.PrimaryDocument {
	if i := v.primaryIndex(); i >= 0 {
		fields := v.docs[i].Fields
		return &p.PrimaryDocument{Kind: fields["documentKind"], Number: fields["documentNumber"]}
	}
	return nil
}
func newDocument() p.DocumentRow {
	return p.DocumentRow{ItemID: f.NewID(), Revision: f.NewID(), Fields: map[string]string{}}
}

func (r *request) applyProfile(v *snapshot, in p.Correct) error {
	for field, val := range in.Basic {
		if !slices.Contains(d.BasicFields, field) || !r.fieldWrite(v.ID, field) {
			return d.Denied()
		}
		if field == "nationality" {
			v.Nationality = nil
		}
		value, e := d.ProfileText(field, val)
		if e != nil {
			return e
		}
		if value == "" {
			delete(v.Basic, field)
		} else {
			v.Basic[field] = value
		}
	}
	if in.Nationality != nil {
		if !r.fieldWrite(v.ID, "nationality") {
			return d.Denied()
		}
		a, name, err := r.resolveCountry(*in.Nationality)
		if err != nil {
			return err
		}
		v.Nationality = a
		v.Basic["nationality"] = name
	}
	if len(in.Documents) > 0 && (in.PrimaryDocument != nil || in.ClearPrimaryDocument) {
		return d.Invalid("documents", "请选择同一项证件办理")
	}
	if in.PrimaryDocument != nil || in.ClearPrimaryDocument {
		// The original T01 command operates on precisely the current primary row.
		i := v.primaryIndex()
		if i >= 0 && len(r.lockedFields(v.ID, v.docs[i].ItemID, "documentKind", "documentNumber", "documentPrimary")) > 0 {
			return d.Denied()
		}
		if in.ClearPrimaryDocument {
			if i >= 0 {
				if !r.canRemoveDocument(v.ID, v.docs[i], true) {
					return d.Denied()
				}
				v.docs = slices.Delete(v.docs, i, i+1)
			}
		} else {
			if i < 0 {
				v.docs = append(v.docs, newDocument())
				i = len(v.docs) - 1
				v.docs[i].Primary = true
			}
			row := &v.docs[i]
			if row.Fields["documentKind"] != in.PrimaryDocument.Kind || row.Fields["documentNumber"] != in.PrimaryDocument.Number {
				row.Fields["documentKind"] = in.PrimaryDocument.Kind
				row.Fields["documentNumber"] = in.PrimaryDocument.Number
				row.Revision = f.NewID()
			}
		}
	}
	seen := map[string]bool{}
	primary := ""
	for index, edit := range in.Documents {
		i := slices.IndexFunc(v.docs, func(x p.DocumentRow) bool { return x.ItemID == edit.ItemID })
		if edit.ItemID == "" {
			if edit.Remove || edit.ExpectedRevision != "" {
				return d.Invalid("documents", "请选择原证件")
			}
			if !r.fieldWrite(v.ID, "documentKind", "documentNumber") {
				return d.Denied()
			}
			v.docs = append(v.docs, newDocument())
			i = len(v.docs) - 1
		} else {
			if i < 0 || !r.allowed("profile.read", v.ID, "documentKind") {
				return d.Denied()
			}
			if seen[edit.ItemID] || v.docs[i].Revision != edit.ExpectedRevision {
				return d.Conflict()
			}
			seen[edit.ItemID] = true
		}
		row := &v.docs[i]
		if edit.Remove {
			if edit.IssuingCountry != nil || len(edit.Fields) > 0 || edit.LongTerm != nil || edit.Primary != nil {
				return d.Invalid("documents", "移除时只需选择原证件")
			}
			if !r.canRemoveDocument(v.ID, *row, false) {
				return d.Denied()
			}
			v.docs = slices.Delete(v.docs, i, i+1)
			continue
		}
		before := hash(row)
		for field, val := range edit.Fields {
			if !slices.Contains(d.DocumentFields, field) || !r.rowWrite(v.ID, row.ItemID, field) {
				return d.Denied()
			}
			if field == "issuingCountry" {
				row.IssuingCountry = nil
			}
			value, e := d.ProfileText(field, val)
			if e != nil {
				return rowInputError("documents", index, field, e)
			}
			if value == "" {
				delete(row.Fields, field)
			} else {
				row.Fields[field] = value
			}
		}
		if edit.IssuingCountry != nil {
			if !r.rowWrite(v.ID, row.ItemID, "issuingCountry") {
				return d.Denied()
			}
			a, name, err := r.resolveCountry(*edit.IssuingCountry)
			if err != nil {
				return rowInputError("documents", index, "issuingCountry", err)
			}
			row.IssuingCountry = a
			row.Fields["issuingCountry"] = name
		}
		if edit.LongTerm != nil {
			if !r.rowWrite(v.ID, row.ItemID, "longTerm") {
				return d.Denied()
			}
			if *edit.LongTerm && row.Fields["expiryDate"] != "" {
				if !r.rowWrite(v.ID, row.ItemID, "expiryDate") {
					return d.Denied()
				}
				delete(row.Fields, "expiryDate")
			}
			row.LongTerm = *edit.LongTerm
		}
		if edit.Primary != nil {
			if !r.rowWrite(v.ID, row.ItemID, "documentPrimary", "documentNumber") {
				return d.Denied()
			}
			row.Primary = *edit.Primary
			if row.Primary {
				if primary != "" {
					return d.Invalid("documentPrimary", "只可指定一份主要证件")
				}
				primary = row.ItemID
			}
		}
		if err := d.ValidateDocument(row.Fields, row.LongTerm); err != nil {
			return rowInputError("documents", index, "", err)
		}
		if before != hash(row) {
			row.Revision = f.NewID()
		}
	}
	numbers := map[string]bool{}
	for i := range v.docs {
		row := &v.docs[i]
		if numbers[row.Fields["documentNumber"]] {
			return d.Conflict()
		}
		numbers[row.Fields["documentNumber"]] = true
		if primary != "" && row.Primary && row.ItemID != primary {
			if !r.rowWrite(v.ID, row.ItemID, "documentPrimary", "documentNumber") {
				return d.Denied()
			}
			row.Primary = false
			row.Revision = f.NewID()
		}
	}
	v.PrimaryDocument = v.primaryValue()
	return nil
}

func (r *request) canRemoveDocument(id string, row p.DocumentRow, legacy bool) bool {
	fields := []string{}
	for key := range row.Fields {
		if !legacy || key != "documentKind" && key != "documentNumber" {
			fields = append(fields, key)
		}
	}
	if row.LongTerm {
		fields = append(fields, "longTerm")
	}
	if !legacy && row.Primary {
		fields = append(fields, "documentPrimary")
	}
	return len(fields) == 0 || r.rowWrite(id, row.ItemID, fields...)
}
func (r *request) projectProfile(v snapshot, out *p.Detail, view string) {
	out.Basic = map[string]string{}
	for _, key := range d.BasicFields {
		if r.allowed("profile.read", v.ID, key) {
			out.Readable = append(out.Readable, key)
			out.Basic[key] = v.Basic[key]
			if key == "nationality" {
				out.Nationality = v.Nationality
			}
		}
	}
	readDocs := r.allowed("profile.read", v.ID, "documentKind")
	for _, key := range append(slices.Clone(d.DocumentFields), "longTerm", "documentPrimary", "contactPurpose", "contactPrimary", "documentNumber.masked") {
		if r.allowed("profile.read", v.ID, key) {
			out.Readable = append(out.Readable, key)
		}
	}
	for _, kind := range []string{"phone", "email"} {
		if r.allowed("read", v.ID, "name", "number", kind) || r.allowed("read", v.ID, "name", "number", kind+".masked") {
			out.Readable = append(out.Readable, kind)
		}
	}
	if !readDocs {
		return
	}
	for _, row := range v.docs {
		fields := maps.Clone(row.Fields)
		for k := range fields {
			if !slices.Contains(out.Readable, k) {
				delete(fields, k)
			}
		}
		if slices.Contains(out.Readable, "documentNumber") {
			if view != "document-"+row.ItemID && !(view == "edit" && r.rowWrite(v.ID, row.ItemID, "documentNumber")) {
				fields["documentNumber"] = "********"
			}
		} else if slices.Contains(out.Readable, "documentNumber.masked") {
			fields["documentNumber"] = "********"
		}
		if !slices.Contains(out.Readable, "issuingCountry") {
			row.IssuingCountry = nil
		}
		if view == "edit" {
			row.Locked = r.lockedFields(v.ID, row.ItemID, append(slices.Clone(d.DocumentFields), "documentPrimary", "longTerm")...)
		}
		row.Fields = fields
		row.Primary = row.Primary && slices.Contains(out.Readable, "documentPrimary")
		row.LongTerm = row.LongTerm && slices.Contains(out.Readable, "longTerm")
		out.Documents = append(out.Documents, row)
	}
}

func (r *request) editContactFlags(id string, row *item, edit p.EditItem, primary map[string]string) error {
	if edit.Purpose != nil {
		if !r.rowWrite(id, row.ItemID, "contactPurpose") {
			return d.Denied()
		}
		value, err := d.ProfileText("contactPurpose", *edit.Purpose)
		if err != nil {
			return err
		}
		if value != row.Purpose {
			row.Purpose = value
			row.Revision = f.NewID()
		}
	}
	if edit.Primary != nil {
		if !r.rowWrite(id, row.ItemID, "contactPrimary") {
			return d.Denied()
		}
		if *edit.Primary {
			if primary[row.Kind] != "" {
				return d.Invalid("contactPrimary", "同一渠道只能指定一条主要联系方式")
			}
			primary[row.Kind] = row.ItemID
		}
		if row.Primary != *edit.Primary {
			row.Primary = *edit.Primary
			row.Revision = f.NewID()
		}
	}
	return nil
}

func (r *request) resolveCountry(c p.CountryChoice) (*p.CountryAdoption, string, error) {
	if !f.ValidID(c.ID) || !f.ValidID(c.SourceVersion) {
		return nil, "", d.Invalid("country", "请选择有效的国家/地区")
	}
	v, err := r.profileCatalog.AdoptCountry(r.ctx, r.credentials, c.ID, c.SourceVersion, r.a.At.In(f.Shanghai).Format("2006-01-02"))
	if err != nil {
		return nil, "", err
	}
	return &p.CountryAdoption{ID: v.ID, SourceVersion: v.SourceVersion, RecordID: v.RecordID, Revision: v.Revision, Date: v.Date}, v.Name, nil
}

func rowInputError(group string, index int, field string, err error) error {
	var fault *f.Fault
	if !errors.As(err, &fault) || (fault.Status != 400 && fault.Code != "PERSON_PLACE_CHANGED" && fault.Code != "PERSON_COUNTRY_CHANGED" && fault.Code != "PERSON_DOCUMENT_USED") {
		return err
	}
	detail := *fault
	if field == "" {
		field = fault.Field
	}
	if field == "documentType" {
		field = "documentKind"
	}
	detail.Field = fmt.Sprintf("%s.%d.%s", group, index, field)
	return &detail
}
