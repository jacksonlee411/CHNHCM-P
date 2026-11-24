package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"strings"
	"testing"
)

func TestCreateIndependentBoundaries(t *testing.T) {
	base := Create{Code: "BU_01", Name: "正常名称", EffectiveDate: "2024-02-29", OperationType: "CREATE", OperationReason: "NEW_ORGANIZATION"}
	cases := []struct {
		name   string
		change func(*Create)
		ok     bool
	}{
		{"leap date", func(*Create) {}, true},
		{"invalid date", func(v *Create) { v.EffectiveDate = "2023-02-29" }, false},
		{"before enterprise", func(v *Create) { v.EffectiveDate = "2019-12-31" }, false},
		{"enterprise first day", func(v *Create) { v.EffectiveDate = "2020-01-01" }, true},
		{"April has thirty days", func(v *Create) { v.EffectiveDate = "2026-04-31" }, false},
		{"year zero", func(v *Create) { v.EffectiveDate = "0000-01-01" }, false},
		{"year ten thousand", func(v *Create) { v.EffectiveDate = "10000-01-01" }, false},
		{"200 supplementary letters", func(v *Create) { v.Name = strings.Repeat("𠮷", 200) }, true},
		{"201 supplementary letters", func(v *Create) { v.Name = strings.Repeat("𠮷", 201) }, false},
		{"emoji not a name", func(v *Create) { v.Name = "甲😀" }, false},
		{"combining marks normalize before counting", func(v *Create) { v.Name = strings.Repeat("e\u0301", 200) }, true},
		{"invalid UTF8", func(v *Create) { v.Name = string([]byte{0xff}) }, false},
		{"control in name", func(v *Create) { v.Name = "甲\n乙" }, false},
		{"spaces", func(v *Create) { v.Code = " BU_01 "; v.Name = "  名称  " }, true},
		{"non ASCII code", func(v *Create) { v.Code = "编码" }, false},
		{"code 41", func(v *Create) { v.Code = strings.Repeat("A", 41) }, false},
		{"code 1", func(v *Create) { v.Code = "A" }, true},
		{"code 40", func(v *Create) { v.Code = strings.Repeat("A", 40) }, true},
		{"code empty", func(v *Create) { v.Code = "" }, false},
		{"code illegal start", func(v *Create) { v.Code = "_A" }, false},
		{"code inner space", func(v *Create) { v.Code = "A B" }, false},
		{"unknown action", func(v *Create) { v.OperationType = "CORRECT" }, false},
		{"reason mandatory", func(v *Create) { v.OperationReason = "" }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := base
			tc.change(&v)
			_, e := ValidateCreate(v, "2020-01-01")
			if (e == nil) != tc.ok {
				t.Fatalf("valid=%v want %v", e == nil, tc.ok)
			}
		})
	}
	blank := " 　"
	base.NameEn = &blank
	got, e := ValidateCreate(base, "2020-01-01")
	if e != nil || got.NameEn != nil {
		t.Fatal("explicit empty English should normalize to null")
	}
	for _, d := range []string{"0001-01-01", "9999-12-31"} {
		if !foundation.ValidDate(d) {
			t.Fatalf("supported boundary date rejected: %s", d)
		}
	}
	base.Code, base.Name = " bu_01 ", "　名称  "
	normalized, e := ValidateCreate(base, "2020-01-01")
	if e != nil || normalized.Code != "bu_01" || normalized.Name != "名称" {
		t.Fatal("trim must preserve display code case and normalize name", normalized, e)
	}
}
