package application

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"reflect"
	"slices"
)

func changeContext(r *domain.Record, fields []string) *org.ChangeRecordContext {
	if r == nil {
		return nil
	}
	v := PublicRecord(*r, fields)
	c := &org.ChangeRecordContext{EffectiveDate: v.EffectiveDate, Name: v.Name, State: v.State, OperationType: v.OperationType, TypeLabel: v.TypeLabel, OperationReason: v.OperationReason, ReasonLabel: v.ReasonLabel}
	if slices.Contains(fields, "nameEn") {
		c.NameEn = &v.NameEn
	}
	if slices.Contains(fields, "effectiveDate") {
		c.EndExclusive = &v.EndExclusive
		c.EndInclusive = &v.EndInclusive
	}
	return c
}
func changeValue(c *org.ChangeRecordContext, field string) org.ChangeValue {
	if c == nil {
		return org.ChangeValue{Kind: "missing"}
	}
	v := org.ChangeValue{Kind: "text"}
	switch field {
	case "name":
		v.Text = c.Name
	case "nameEn":
		if c.NameEn != nil {
			v.Text = *c.NameEn
		}
		if v.Text == nil {
			v.Kind = "empty"
		}
	case "effectiveDate":
		v.Kind = "date"
		v.Text = &c.EffectiveDate
	case "period":
		v.Kind = "period"
		v.Start = c.EffectiveDate
		v.EndExclusive = c.EndExclusive
		v.EndInclusive = c.EndInclusive
	case "state":
		v.Kind = "state"
		v.Code = c.State
	case "operationType":
		v.Kind = "classification"
		v.Code = c.OperationType
		v.Label = c.TypeLabel
	case "operationReason":
		v.Kind = "classification"
		v.Code = c.OperationReason
		v.Label = c.ReasonLabel
	}
	return v
}

// BuildChangeItem consumes exact immutable revisions, never the current object.
func BuildChangeItem(recordID, kind, previousSource, source string, before, after *domain.Record, fields []string) org.ChangeItem {
	item := org.ChangeItem{RecordID: recordID, ChangeKind: kind, BeforeContext: changeContext(before, fields), AfterContext: changeContext(after, fields), Diffs: []org.ChangeDiff{}}
	if before != nil {
		item.BeforeRef = &org.ChangeRef{SourceVersion: previousSource, RecordID: recordID, Revision: before.Revision}
	}
	if after != nil {
		item.AfterRef = &org.ChangeRef{SourceVersion: source, RecordID: recordID, Revision: after.Revision}
	}
	for _, field := range changeFields {
		if !ChangeFieldAllowed(field, fields) {
			continue
		}
		a, b := changeValue(item.BeforeContext, field), changeValue(item.AfterContext, field)
		// An absent optional value is not a material field change on creation/deletion.
		if field == "nameEn" && (a.Kind == "missing" && b.Kind == "empty" || a.Kind == "empty" && b.Kind == "missing") {
			continue
		}
		changed := !reflect.DeepEqual(a, b)
		if field == "operationReason" && before != nil && after != nil && before.Type != after.Type {
			changed = true
		}
		if changed {
			item.Diffs = append(item.Diffs, org.ChangeDiff{Field: field, Before: a, After: b})
		}
	}
	return item
}
