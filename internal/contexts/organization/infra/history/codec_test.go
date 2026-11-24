package history

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	app "chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
)

func TestHistoryKeepsIndependentIdentityAndFrozenValues(t *testing.T) {
	ctx := context.Background()
	p := compact.NewPool(compact.Values{})
	id, first, source, revision := foundation.NewID(), foundation.NewID(), foundation.NewID(), foundation.NewID()
	name := "Old English name"
	v := model.BUState{Timeline: app.Timeline{BUID: id, Code: "KEEP", First: first, Source: source, SourceNo: 50, Records: []domain.Record{{ID: first, Revision: revision, RevisionNo: 49, Date: "2020-01-01", End: "2026-01-01", Type: "CREATE", Reason: "INITIAL", State: "active", TypeLabel: "创建", ReasonLabel: "首次登记", Data: domain.BUData{Name: "独立名称", NameEn: &name}}}}, Summaries: []org.OperationSummary{{RecordID: first, Revision: revision, OperationSourceVersion: source, RecordedAt: "2026-11-07T01:02:03.123456Z", Operator: org.OperatorSnapshot{SubjectID: foundation.NewID(), IdentityEvidence: "subject_id_only", Kind: "unknown"}}}}
	b, e := EncodeBUState(ctx, p, v)
	if e != nil {
		t.Fatal(e)
	}
	oldValues := compact.NewPool(p.Values).Values
	oldHash, _ := compact.Digest([]string{"instance", "enterprise", id, source}, oldValues, [][]byte{b})
	// A later source may reuse the old revision but must not change its name,
	// classification or original operation identity.
	next := v
	next.Timeline.Records = append([]domain.Record{}, v.Timeline.Records...)
	next.Timeline.Source = foundation.NewID()
	next.Timeline.SourceNo++
	next.Timeline.Records[0].Data.Name = "后来的名称"
	next.Timeline.Records[0].Revision = foundation.NewID()
	if _, e = EncodeBUState(ctx, p, next); e != nil {
		t.Fatal(e)
	}
	got, e := DecodeBUState(ctx, p.Values, b)
	if e != nil || !reflect.DeepEqual(got, v) {
		t.Fatalf("old source changed: %v\n%#v", e, got)
	}
	h, _ := compact.Digest([]string{"instance", "enterprise", id, source}, oldValues, [][]byte{b})
	if !bytes.Equal(h, oldHash) {
		t.Fatal("old envelope changed")
	}
	for _, owner := range [][]string{{"other", "enterprise", id, source}, {"instance", "other", id, source}, {"instance", "enterprise", foundation.NewID(), source}, {"instance", "enterprise", id, foundation.NewID()}} {
		h, _ := compact.Digest(owner, oldValues, [][]byte{b})
		if bytes.Equal(h, oldHash) {
			t.Fatal("owner not bound")
		}
	}
	wrong := compact.NewPool(oldValues).Values
	wrong.IDs[0] = foundation.NewID()
	h, _ = compact.Digest([]string{"instance", "enterprise", id, source}, wrong, [][]byte{b})
	if bytes.Equal(h, oldHash) {
		t.Fatal("pool not bound")
	}
}

func TestPathAndAcceptedTerminalRoundTrip(t *testing.T) {
	ctx := context.Background()
	p := compact.NewPool(compact.Values{})
	id, source, record, revision := foundation.NewID(), foundation.NewID(), foundation.NewID(), foundation.NewID()
	edge := d.MemberRef{DepartmentID: id, Source: foundation.NewID(), MemberID: foundation.NewID(), BU: foundation.NewID(), Start: "2020-01-01", Parent: foundation.NewID(), EdgeID: foundation.NewID(), Revision: foundation.NewID()}
	evidence := model.DepartmentEvidence{Segments: []d.PathSegment{{RecordID: record, Revision: revision, Start: "2021-01-01", End: "2022-01-01", Mode: "before_inactive", BusinessDate: "2020-12-31", Nodes: []d.PathNode{{Kind: "enterprise", Name: "企业"}, {Kind: "department", DepartmentID: id, Source: source, RecordID: record, Revision: revision, Name: "原路径名称", Edge: edge}}}}}
	b, e := EncodeDepartmentEvidence(ctx, p, evidence)
	if e != nil {
		t.Fatal(e)
	}
	got, e := DecodeDepartmentEvidence(ctx, p.Values, b)
	if e != nil || !reflect.DeepEqual(got, evidence) {
		t.Fatalf("path facts changed: %v", e)
	}
	now := time.Date(2026, 11, 7, 1, 2, 3, 123456000, time.UTC)
	sid := foundation.NewID()
	op := model.BUOperation{Source: model.Source{ID: source, Previous: foundation.NewID(), Number: 50, Subject: foundation.NewID(), Submission: sid, Action: "CORRECT", RecordedAt: now}, Terminal: app.Terminal{ID: sid, Outcome: "accepted", Action: "CORRECT", Digest: bytes.Repeat([]byte{0xa5}, 32), Fields: []string{"name"}, EvaluatedAt: &now, Result: org.Result{SubmissionID: sid, Outcome: "accepted", BUID: &id, SourceVersion: &source, TargetRecordID: &record, TargetRevision: &revision}}, EventID: foundation.NewID()}
	b, e = EncodeBUOperation(ctx, p, op)
	if e != nil {
		t.Fatal(e)
	}
	back, e := DecodeBUOperation(ctx, p.Values, b)
	if e != nil || !reflect.DeepEqual(back, op) {
		t.Fatalf("terminal facts changed: %v", e)
	}
	for i := 0; i < len(b); i++ {
		if _, e = DecodeBUOperation(ctx, p.Values, b[:i]); e == nil {
			t.Fatalf("accepted truncated source at %d", i)
		}
	}
	if _, e = DecodeBUOperation(ctx, p.Values, append(append([]byte{}, b...), 0)); e == nil {
		t.Fatal("accepted trailing bytes")
	}
	if _, e = DecodeBUState(ctx, p.Values, b); e == nil {
		t.Fatal("accepted wrong typed format")
	}
	if _, e = DecodeBUOperation(ctx, compact.Values{}, b); e == nil {
		t.Fatal("accepted missing dictionary")
	}
}

func FuzzTypedHistoryRejectsMalformedContent(f *testing.F) {
	p := compact.NewPool(compact.Values{})
	b, _ := EncodeBUOperation(context.Background(), p, model.BUOperation{})
	f.Add(b)
	f.Add([]byte{2, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > compact.MaxPart {
			t.Skip()
		}
		v, e := DecodeBUOperation(context.Background(), p.Values, b)
		if e == nil {
			if _, e = json.Marshal(v); e != nil {
				t.Fatal(e)
			}
		}
	})
}
