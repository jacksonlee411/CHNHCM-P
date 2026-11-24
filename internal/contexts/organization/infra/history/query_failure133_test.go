package history

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"testing"
)

func TestQuery133DecodeCancellationIsNotCorruption(t *testing.T) {
	id := "00000000-0000-4000-8000-000000000001"
	enterprise := "00000000-0000-4000-8000-000000000002"
	record := "00000000-0000-4000-8000-000000000003"
	source := "00000000-0000-4000-8000-000000000004"
	relation := "00000000-0000-4000-8000-000000000005"
	timeline := d.Timeline{ID: id, Enterprise: enterprise, First: record, Source: source, RelationSource: relation, Records: []d.Record{{ID: record, Revision: record, Date: "2020-01-01", State: "active", Data: d.Data{Name: "测试", BU: enterprise}}}}
	p := compact.NewPool(compact.Values{})
	data, e := EncodeDepartmentRead(context.Background(), p, model.ReadDepartment(timeline))
	if e != nil {
		t.Fatal(e)
	}
	values := p.Values
	values.IDs = p.IDs
	m := selectionMaterial{kind: 3, values: values, parts: [][]byte{data}, dictionaryDigest: make([]byte, 32), shapeDigest: make([]byte, 32)}
	for _, text := range []string{"2026-11-12", id, "TEST", source, record, relation, source} {
		m.meta = append(m.meta, pgtype.Text{String: text, Valid: true})
	}
	decoder := selectionDecoder{}
	plan := access.ReadPlan{Enterprise: enterprise}
	out := app.QueryCapture{ByDay: map[string]d.Snapshot{}}
	if e = decoder.boundDepartment(context.Background(), plan, m, &out); e != nil {
		t.Fatal("valid material rejected", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e = decoder.boundDepartment(ctx, plan, m, &out)
	var fault *foundation.Fault
	if !errors.Is(e, context.Canceled) || !errors.As(e, &fault) || fault.Status != 503 {
		t.Fatalf("decode cancellation lost: %T %v", e, e)
	}
}
