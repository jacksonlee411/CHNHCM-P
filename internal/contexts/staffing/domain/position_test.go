package domain

import (
	"slices"
	"testing"
)

func TestPositionUseChangesAreDatedAndFieldSpecific(t *testing.T) {
	active := Record{Date: "2027-01-01", Type: "CREATE", Data: map[string]string{"jobId": "J", "gradeId": "G5"}}
	future := Record{Date: "2027-04-01", Type: "CHANGE", Data: map[string]string{"jobId": "J"}}
	initial := Timeline{Records: []Record{active, future}}
	stopped := Timeline{Records: []Record{active, {Date: "2027-03-01", Type: "DEACTIVATE", Data: active.Data}, future}}
	want := []PositionUseBoundary{{"jobId", "J", "2027-03-01", true}, {"gradeId", "G5", "2027-03-01", false}}
	if !slices.Equal(ChangedPositionUses(initial, stopped), want) || !slices.Equal(ChangedPositionUses(stopped, initial), want) {
		t.Fatal("停用和取消停用都应在受影响日期核对原引用；独立后继不受影响")
	}
	changed := Timeline{Records: []Record{active, {Date: future.Date, Type: future.Type, Data: map[string]string{"jobId": "K"}}}}
	want = []PositionUseBoundary{{"jobId", "J", "2027-04-01", false}, {"jobId", "K", "2027-04-01", false}}
	if got := ChangedPositionUses(initial, changed); !slices.Equal(got, want) {
		t.Fatalf("换职务应精确检查新旧值：%v", got)
	}
}

func TestPositionFutureJobGradeBoundary(t *testing.T) {
	job := Timeline{Records: []Record{
		{Date: "2026-01-01", End: "2027-04-01", Type: "CREATE", Grades: GradeMembers{{GradeID: "G5"}}},
		{Date: "2027-04-01", End: "2027-05-01", Type: "CHANGE"},
		{Date: "2027-05-01", Type: "DEACTIVATE"},
	}}
	for _, sample := range []struct{ end, grade, conflict string }{
		{"", "G5", "2027-04-01"}, {"2027-04-01", "G5", ""}, {"2027-04-02", "G5", "2027-04-01"},
		{"2027-05-01", "", ""}, {"", "", "2027-05-01"},
	} {
		if got := PositionConflict(job, "2027-01-01", sample.end, sample.grade); got != sample.conflict {
			t.Errorf("[2027-01-01,%s) grade %q: got %q, want %q", sample.end, sample.grade, got, sample.conflict)
		}
	}
}
