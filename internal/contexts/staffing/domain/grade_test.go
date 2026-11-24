package domain

import (
	"strings"
	"testing"
)

func TestGradeValueAndDatedSnapshots(t *testing.T) {
	ptr := func(s string) *string { return &s }
	create := Command{Action: "CREATE", Code: " g-01 ", Type: "CREATE", Reason: "NEW_GRADE", Changes: map[string]*string{"name": ptr(" 初级 "), "nameEn": ptr("e\u0301"), "description": ptr("第一行\r\n第二行")}}
	p, err := Build(Grade, "enterprise", Timeline{}, create)
	if err != nil {
		t.Fatal(err)
	}
	if p.After.Code != "G-01" || p.After.Records[0].Date != "1900-01-01" || p.After.Records[0].Data["nameEn"] != "é" || p.After.Records[0].Data["description"] != "第一行\n第二行" {
		t.Fatalf("unexpected Grade: %+v", p.After)
	}
	before := p.After
	add, err := Build(Grade, "enterprise", before, Command{Action: "ADD", ExpectedSource: before.Source, AnchorID: before.Records[0].ID, Date: "2030-01-01", Type: "CHANGE", Reason: "DESCRIPTION_ADJUSTMENT", Changes: map[string]*string{"name": ptr("未来职级")}})
	if err != nil {
		t.Fatal(err)
	}
	first := add.After.Records[0]
	correct, err := Build(Grade, "enterprise", add.After, Command{Action: "CORRECT", ExpectedSource: add.After.Source, RecordID: first.ID, ExpectedRevision: first.Revision, Changes: map[string]*string{"name": ptr("修正首项"), "nameEn": nil}})
	if err != nil {
		t.Fatal(err)
	}
	if correct.After.Records[1].Data["name"] != "未来职级" || correct.After.Records[0].Data["nameEn"] != "" || before.Records[0].Data["name"] != "初级" || len(correct.Adopt) != 0 {
		t.Fatal("Grade snapshots/optional clearing crossed an independent boundary")
	}
	for _, field := range []string{"jobFamilyId", "gradeIds", "salary", "rank", "code"} {
		c := create
		c.Changes = map[string]*string{"name": ptr("有效名称"), field: ptr("not-a-grade-field")}
		if _, err := Build(Grade, "enterprise", Timeline{}, c); err == nil {
			t.Fatalf("Grade accepted %s", field)
		}
	}
	for _, value := range []string{strings.Repeat("长", 1001), "含\t控制符"} {
		if _, err := (GradePatch{Description: ptr(value)}).Apply(GradeData{Name: "职级"}); err == nil {
			t.Fatal("invalid Grade description accepted")
		}
	}
	create.Reason = "NEW_JOB_FAMILY"
	if _, err := Build(Grade, "enterprise", Timeline{}, create); err == nil {
		t.Fatal("Grade accepted Family classification")
	}
}
