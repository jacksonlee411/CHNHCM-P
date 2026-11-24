package postgres

import (
	"fmt"
	"slices"
	"testing"
	"time"

	d "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

func summary133Fixture(n int) model.BUState {
	var state model.BUState
	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		id := fmt.Sprintf("00000000-0000-4000-8000-%012x", i+1)
		revision := fmt.Sprintf("00000000-0000-4000-9000-%012x", i+1)
		r := d.Record{ID: id, Revision: revision, RevisionNo: 1, Date: start.AddDate(0, 0, i).Format("2006-01-02"), Type: "CHANGE", Reason: "reason", TypeLabel: "变更", ReasonLabel: "原因", State: "active", Data: d.BUData{Name: "名称"}}
		if i == 0 {
			r.Type = "CREATE"
			state.Timeline.First = id
		}
		if i+1 < n {
			r.End = start.AddDate(0, 0, i+1).Format("2006-01-02")
		}
		state.Timeline.Records = append(state.Timeline.Records, r)
		state.Summaries = append(state.Summaries, org.OperationSummary{RecordID: id, Revision: revision, OperationSourceVersion: "00000000-0000-4000-8000-000000100000", RecordedAt: "2026-11-12T01:00:00Z", Operator: org.OperatorSnapshot{SubjectID: "00000000-0000-4000-8000-000000200000"}})
	}
	return state
}

func TestSummary133ExactMembershipAndEvidence(t *testing.T) {
	state := summary133Fixture(137)
	slices.Reverse(state.Summaries)
	if err := validateBUState(state); err != nil {
		t.Fatal("summary order is not membership", err)
	}
	for _, kind := range []string{"duplicate", "missing", "revision", "source", "operator", "time", "duplicate-record"} {
		t.Run(kind, func(t *testing.T) {
			v := summary133Fixture(137)
			switch kind {
			case "duplicate":
				v.Summaries[136] = v.Summaries[0]
			case "missing":
				v.Summaries = v.Summaries[:136]
			case "revision":
				v.Summaries[136].Revision = v.Summaries[0].Revision
			case "source":
				v.Summaries[136].OperationSourceVersion = "invalid"
			case "operator":
				v.Summaries[136].Operator.SubjectID = "invalid"
			case "time":
				v.Summaries[136].RecordedAt = "invalid"
			case "duplicate-record":
				v.Timeline.Records[136].ID = v.Timeline.Records[0].ID
			}
			if validateBUState(v) == nil {
				t.Fatal("invalid original evidence accepted")
			}
		})
	}
}

func BenchmarkSummary133State(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			state := summary133Fixture(n)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if err := validateBUState(state); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
