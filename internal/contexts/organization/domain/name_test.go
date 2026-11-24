package domain

import (
	"context"
	"strings"
	"testing"
	"unicode"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestOrganizationNameProfile(t *testing.T) {
	if unicode.Version != nameUnicodeVersion || foundation.UnicodeNormalizationVersion != nameUnicodeVersion {
		t.Fatal("Unicode upgrade requires deliberate name-profile review", unicode.Version, foundation.UnicodeNormalizationVersion)
	}
	valid := map[string]string{
		"  华东事业部（一期）　": "华东事业部（一期）",
		"AI研发中心":       "AI研发中心", "第1事业部": "第1事业部",
		"R&D / C++ Center #2": "R&D / C++ Center #2",
		"O’Connor Operations": "O’Connor Operations", "西安·研发一部": "西安·研发一部",
		"𠮷祥研发中心": "𠮷祥研发中心", "Société": "Société",
		"Socie\u0301te\u0301": "Société", "ＡＩⅢ研发": "ＡＩⅢ研发",
		"q\u0301研究": "q\u0301研究", "שָׁלוֹם": "שָׁלוֹם",
		"نامه\u200cای": "نامه\u200cای", "ب\u200c\u064Eب": "ب\u200c\u064Eب", "क्\u200dष": "क्\u200dष", "क्\u200cष": "क्\u200cष",
		"葛\U000E0100事业部": "葛\U000E0100事业部", "ᠠ\u180b": "ᠠ\u180b",
		strings.Repeat("中", 199) + "𠮷": strings.Repeat("中", 199) + "𠮷",
		strings.Repeat("e\u0301", 200): strings.Repeat("é", 200),
	}
	for input, want := range valid {
		t.Run(input, func(t *testing.T) {
			got, err := NormalizeName(input, true)
			if err != nil || got != want {
				t.Fatalf("got %q / %v, want %q", got, err, want)
			}
		})
	}
	invalid := []string{
		"", "　 ", "-（ ）", "\u0301研发", "甲\n乙", "甲\n", "\t甲", "甲\u0085", "甲\u2028", "甲\u2029",
		"华东😀事业部", "中国🇨🇳", "第1\ufe0f\u20e3部", "第1\u20e3部", "开发👩\u200d💻", "研发👍🏽", "❤️研发", "ℹ研发",
		"甲\u200b乙", "甲\u202e乙", "甲\u2066乙", "甲\u034f乙", "甲\u3164乙", "甲\ue000乙", "甲\u0378乙", "甲\ufffd乙",
		"甲\u200c乙", "A\u200dB", "ن\u200dم", "ا\u200cب", "क्\u200dA", "क्\u200d", "甲\ufe0f", "A\U000E0100", "葛\U000E0100\U000E0101",
		strings.Repeat("中", 200) + "𠮷", strings.Repeat("e\u0301", 201), string([]byte{0xff}),
	}
	for _, input := range invalid {
		t.Run(input, func(t *testing.T) {
			if _, err := NormalizeName(input, true); err == nil {
				t.Fatalf("accepted invalid name %q", input)
			}
		})
	}
	if got, err := NormalizeName("　 ", false); err != nil || got != "" {
		t.Fatal("optional empty", got, err)
	}
}

func TestNameChangesDoNotRewriteInheritedOrHistoricalText(t *testing.T) {
	base := baseline()
	base[0].Data.Name = "旧😀名称"
	base[0].Data.NameEn = textValue("e\u0301")
	got, err := Calculate(context.Background(), base, "first", "2020-01-01", []Item{{ID: "fix", Action: "CORRECT", RecordID: "first", ExpectedRevision: "r1", Changes: PatchFromFields(map[string]*string{"name": textValue("𠮷祥事业部")})}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Records[0].Data.Name != "𠮷祥事业部" || *got.Records[0].Data.NameEn != "e\u0301" || base[0].Data.Name != "旧😀名称" || got.Records[1].Data.Name != "乙" {
		t.Fatal("undeclared fields, old source or successor changed", got)
	}
	base[0].Data.Name = "e\u0301事业部"
	got, err = Calculate(context.Background(), base, "first", "2020-01-01", []Item{{ID: "same", Action: "CORRECT", RecordID: "first", ExpectedRevision: "r1", Changes: PatchFromFields(map[string]*string{"name": textValue("é事业部"), "nameEn": textValue("é")})}})
	if err != nil || !got.NoChange || got.Records[0].Data.Name != "e\u0301事业部" || *got.Records[0].Data.NameEn != "e\u0301" {
		t.Fatal("canonical equality must not rewrite old text", got, err)
	}
}
