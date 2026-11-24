package domain

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"testing"
)

// Adjacent records can retain their exact data while the distribution over
// shared days changes. Only the identity on shared days is a field change.
func TestJobFunctionImplicitWriteIntervals(t *testing.T) {
	const jan = "2027-01-01"
	const feb = "2027-02-01"
	const mar = "2027-03-01"
	const apr = "2027-04-01"
	timeline := func(bounds ...string) Timeline {
		var out Timeline
		for i := 0; i < len(bounds); i += 2 {
			r := Record{Date: bounds[i], Data: map[string]string{"jobFunctionId": bounds[i+1]}}
			if i+2 < len(bounds) {
				r.End = bounds[i+2]
			}
			out.Records = append(out.Records, r)
		}
		return out
	}
	for _, tc := range []struct {
		name          string
		before, after Timeline
		want          bool
	}{
		{"纯改日", timeline(jan, ""), timeline(feb, ""), false},
		{"同类多一段", timeline(jan, "A"), timeline(jan, "A", feb, "A"), false},
		{"独立未登记后移", timeline(jan, "A", mar, ""), timeline(jan, "A", apr, ""), true},
		{"删除回退", timeline(jan, "", feb, "A", apr, ""), timeline(jan, "", apr, ""), true},
		{"增加更早首条且共有期相同", timeline(mar, "A"), timeline(jan, "B", mar, "A"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if functionChanged(tc.before, tc.after) != tc.want {
				t.Fatal("期间归类判定错误")
			}
		})
	}
}

func TestJobFunctionOldBasisAndNewAdoption(t *testing.T) {
	old := create(t, Job, "2027-02-01")
	r := &old.Records[0]
	delete(r.Data, "jobFunctionId")
	evidence := FunctionEvidence{Source: old.Source, RecordID: r.ID, Revision: r.Revision}
	r.Function = evidence
	correct := Command{Action: "CORRECT", RecordID: r.ID, ExpectedRevision: r.Revision, ExpectedSource: old.Source, Changes: map[string]*string{"name": str("更正文字")}}
	p, err := Build(Job, old.Enterprise, old, correct)
	if err != nil || p.After.Records[0].Function != evidence {
		t.Fatalf("旧依据丢失 %v", err)
	}
	add := Command{Action: "ADD", AnchorID: r.ID, ExpectedSource: old.Source, Type: "CHANGE", Reason: "OTHER_DATA_ADJUSTMENT", Date: "2027-03-01"}
	if _, err = Build(Job, old.Enterprise, old, add); err == nil {
		t.Fatal("新增启用段沿用缺值")
	}
	add.Changes = map[string]*string{"jobFunctionId": str(foundation.NewID())}
	p, err = Build(Job, old.Enterprise, old, add)
	if err != nil || len(p.AdoptFunctions) != 1 || len(p.Adopt) != 0 {
		t.Fatalf("采用计划混合两维 %v", err)
	}
	r.Function = FunctionEvidence{}
	if _, err = Build(Job, old.Enterprise, old, correct); err == nil {
		t.Fatal("无来源依据的缺值可维护")
	}
}

func TestUnchangedLegacyStopKeepsItsOwnCutoverEvidence(t *testing.T) {
	job := create(t, Job, "2000-01-01")
	job = apply(t, job, Command{Action: "DEACTIVATE", AnchorID: job.Records[0].ID, Date: "2001-01-01", Type: "DEACTIVATE", Reason: "JOB_CLOSURE"}).After
	for i := range job.Records {
		r := &job.Records[i]
		delete(r.Data, "jobFunctionId")
		r.Function = FunctionEvidence{Source: job.Source, RecordID: r.ID, Revision: r.Revision}
	}
	first := job.Records[0]
	p := apply(t, job, Command{Action: "CORRECT", RecordID: first.ID, ExpectedRevision: first.Revision, Changes: map[string]*string{"name": str(first.Data["name"])}})
	if p.Changed || p.After.Records[1].Function != job.Records[1].Function {
		t.Fatal("原样更正改写了停用原件的未登记依据")
	}
}
