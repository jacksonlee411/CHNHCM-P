package application

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"

	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	cfg "chnhcm.local/chnhcm/internal/contexts/configuration/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

type PlanState struct {
	ID, Enterprise, Code, Version string
	Records                       d.Timeline[d.PlanTerms]
}

func strictValue(raw json.RawMessage, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
func PlanFields(in c.WritePlan) ([]string, error) {
	if in.Action != "CREATE" && !slices.Contains(RateActions, in.Action) {
		return nil, d.Invalid("action", "请选择合法办理动作")
	}
	fields := []string{}
	if in.Action == "CREATE" {
		if in.RecordID != "" || in.ExpectedVersion != "" || in.ExpectedRevision != "" {
			return nil, d.Invalid("recordId", "新建计划不使用已有记录")
		}
		fields = append(fields, "code")
	} else if in.Code != "" || !foundation.ValidID(in.ExpectedVersion) || !foundation.ValidID(in.ExpectedRevision) || !foundation.ValidID(in.RecordID) {
		return nil, d.Invalid("recordId", "请选择已有计划的确切记录")
	}
	if in.EffectiveDate != "" {
		fields = append(fields, "effectiveDate")
	}
	if in.Reason != "" {
		fields = append(fields, "reason")
	}
	if in.Action == "DELETE" && len(fields) > 0 {
		return nil, d.Invalid("action", "撤去使用所选记录的原日期与分类")
	}
	for k, v := range in.Changes {
		if !slices.Contains([]string{"name", "description", "scope", "components"}, k) || in.Action == "DELETE" || in.Action == "DEACTIVATE" {
			return nil, d.Invalid(k, "此动作不接受该字段")
		}
		fields = append(fields, k)
		if k != "components" {
			continue
		}
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal(v, &rows); err != nil {
			return nil, d.Invalid(k, "请填写组成项")
		}
		for _, row := range rows {
			for key := range row {
				switch key {
				case "amount":
					fields = append(fields, "amount")
				case "rates":
					fields = append(fields, "rates")
				case "percent", "baseId", "rounding":
					fields = append(fields, "percentage")
				}
			}
		}
	}
	slices.Sort(fields)
	return slices.Compact(fields), nil
}

type IncomingPlan struct {
	Terms       d.PlanTerms
	Scope       *cfg.Scope
	ExtraFields []string
}

func ReadPlanInput(prior d.PlanTerms, changes map[string]json.RawMessage) (IncomingPlan, error) {
	out := IncomingPlan{Terms: prior}
	for _, key := range []string{"name", "description"} {
		if value, ok := changes[key]; ok {
			var text string
			if err := json.Unmarshal(value, &text); err != nil {
				return out, d.Invalid(key, "请输入文本")
			}
			text = strings.TrimSpace(text)
			if key == "name" {
				out.Terms.Name = text
			} else {
				out.Terms.Description = text
			}
		}
	}
	if out.Terms.Name == "" {
		return out, d.Invalid("name", "请填写计划名称")
	}
	if value, ok := changes["scope"]; ok {
		out.Scope = &cfg.Scope{}
		if err := strictValue(value, out.Scope); err != nil {
			return out, d.Invalid("scope", "请明确适用方式和范围选择")
		}
	} else if prior.ScopeVersion == "" {
		return out, d.Invalid("scope", "请明确选择企业通用或指定范围")
	}
	if value, ok := changes["components"]; ok {
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal(value, &rows); err != nil {
			return out, d.Invalid("components", "请填写组成项")
		}
		old := map[string]d.Component{}
		for _, r := range prior.Components {
			old[r.ID] = r
		}
		out.Terms.Components = make([]d.Component, 0, len(rows))
		for i, row := range rows {
			var id string
			if v, exists := row["id"]; exists {
				if err := json.Unmarshal(v, &id); err != nil {
					return out, d.Invalid("components."+strconv.Itoa(i)+".id", "组成项身份无效")
				}
			}
			if id == "" {
				id = foundation.NewID()
			}
			base, found := old[id]
			if !found {
				base = d.Component{ID: id, Precision: 2}
			}
			item, err := readComponent(base, row, i)
			if err != nil {
				return out, err
			}
			out.Terms.Components = append(out.Terms.Components, item)
			delete(old, id)
			if found {
				out.ExtraFields = append(out.ExtraFields, changedTerms(base, item)...)
			}
		}
		for _, removed := range old {
			out.ExtraFields = append(out.ExtraFields, changedTerms(removed, d.Component{})...)
		}
		if err := d.ValidateComponents(out.Terms.Components); err != nil {
			return out, err
		}
	} else if len(prior.Components) == 0 {
		return out, d.Invalid("components", "计划至少需要一个组成项")
	}
	slices.Sort(out.ExtraFields)
	out.ExtraFields = slices.Compact(out.ExtraFields)
	return out, nil
}
func changedTerms(a, b d.Component) []string {
	var out []string
	if a.Amount != b.Amount {
		out = append(out, "amount")
	}
	if !reflect.DeepEqual(a.Rates, b.Rates) {
		out = append(out, "rates")
	}
	if a.BaseID != b.BaseID || a.Percent != b.Percent || a.Rounding != b.Rounding {
		out = append(out, "percentage")
	}
	return out
}

func readComponent(base d.Component, fields map[string]json.RawMessage, ix int) (d.Component, error) {
	next := base
	path := "components." + strconv.Itoa(ix) + "."
	if raw, exists := fields["method"]; exists {
		var method string
		if err := json.Unmarshal(raw, &method); err != nil {
			return next, d.Invalid(path+"method", "取值方式无效")
		}
		if method != base.Method {
			next.Amount = ""
			next.BaseID = ""
			next.Percent = ""
			next.Rounding = ""
			next.Rates = nil
		}
	}
	stringsByKey := map[string]*string{"code": &next.Code, "name": &next.Name, "nature": &next.Nature, "currency": &next.Currency, "frequency": &next.Frequency, "unit": &next.Unit, "method": &next.Method, "amount": &next.Amount, "baseId": &next.BaseID, "percent": &next.Percent, "rounding": &next.Rounding}
	for key, raw := range fields {
		switch key {
		case "id":
			continue
		case "precision":
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &next.Precision) != nil {
				return next, d.Invalid(path+key, "请填写金额小数位数")
			}
		case "rates":
			var refs []struct {
				GradeID string `json:"gradeId"`
				RateID  string `json:"rateId"`
			}
			if err := strictValue(raw, &refs); err != nil {
				return next, d.Invalid(path+key, "请选择职级和准确标准")
			}
			next.Rates = nil
			for _, v := range refs {
				link := d.RateLink{GradeID: v.GradeID, RateID: v.RateID}
				for _, original := range base.Rates {
					if original.GradeID == v.GradeID && original.RateID == v.RateID {
						link = original
						break
					}
				}
				next.Rates = append(next.Rates, link)
			}
		default:
			dst, ok := stringsByKey[key]
			if !ok {
				return next, d.Invalid(path+key, "此组成项不接受该字段")
			}
			var value string
			if err := json.Unmarshal(raw, &value); err != nil {
				return next, d.Invalid(path+key, "请输入有效字段值")
			}
			*dst = value
		}
	}
	return next, nil
}
