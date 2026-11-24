package history

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
)

//go:embed bu_coverage_reference.sql
var buCoverageReferenceSQL string

// CaptureBUCoverageReference uses one statement snapshot for the current BU
// source, query member, Department head, dictionary and full current state.
// It does not disclose a Department identity or reuse the result for saving.
func CaptureBUCoverageReference(ctx context.Context, tx pgx.Tx, instance, bu, source string, lost []domain.BUCoverageRange) (*model.DepartmentState, error) {
	windows, err := json.Marshal(lost)
	if err != nil {
		return nil, err
	}
	var current string
	var raw []byte
	done := foundation.BeginStage(ctx, "bu.coverage.capture")
	// BU ownership ranges from no departments to the whole enterprise. Plan
	// with this BU's actual value so a generic average cannot force an empty
	// scope through every department; parameter encoding remains cached.
	err = tx.QueryRow(ctx, buCoverageReferenceSQL, pgx.QueryExecModeCacheDescribe, instance, bu, windows).Scan(&current, &raw)
	done()
	if errors.Is(err, pgx.ErrNoRows) || err == nil && current != source {
		return nil, foundation.Fail(409, "M02_SOURCE_CONFLICT", "资料已更新，请核对最新内容后保存")
	}
	if err != nil || raw == nil {
		return nil, err
	}
	defer foundation.BeginStage(ctx, "bu.coverage.decode_validate")()
	var item struct {
		ID, Enterprise, First, Source, Relation string
		Record, Revision, Start, End, BU, State string
		Number                                  int64
		Valid                                   bool
		IDs                                     []string
		Envelope, DictionaryDigest, ShapeDigest string
		Material                                capturedMaterial
	}
	if err = json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	if !item.Valid || !foundation.ValidID(item.ID) || item.BU != bu || item.State != "active" {
		return nil, compact.ErrCorrupt
	}
	values, data, err := queryValues(item.IDs, item.Material)
	if err != nil {
		return nil, err
	}
	parts := make([][]byte, len(item.Material.Parts))
	for i, part := range item.Material.Parts {
		if parts[i], err = hex.DecodeString(part); err != nil {
			return nil, err
		}
	}
	dh, sh := hashParts(dictionaryParts(values)), hashParts(parts)
	if hex.EncodeToString(dh) != item.DictionaryDigest || hex.EncodeToString(sh) != item.ShapeDigest {
		return nil, compact.ErrCorrupt
	}
	seal, err := ProjectionDigest(Department, instance, item.Enterprise, item.ID, item.Source, item.IDs, dh, sh)
	if err != nil {
		return nil, err
	}
	if hex.EncodeToString(seal) != item.Envelope {
		return nil, compact.ErrCorrupt
	}
	state, err := DecodeDepartmentState(ctx, values, data)
	if err != nil {
		return nil, err
	}
	t := state.Timeline
	if t.ID != item.ID || t.Enterprise != item.Enterprise || t.First != item.First || t.Source != item.Source || t.SourceNo != item.Number || t.RelationSource != item.Relation {
		return nil, compact.ErrCorrupt
	}
	// A corrupt query member cannot manufacture a business conflict. The exact
	// selected record and lost-period overlap must exist in the captured state.
	for _, r := range t.Records {
		if r.ID == item.Record && r.Revision == item.Revision && r.Date == item.Start && r.End == item.End && r.State == "active" && r.Data.BU == bu {
			for _, w := range lost {
				if w.Intersects(r.Date, r.End) {
					return &state, nil
				}
			}
		}
	}
	return nil, compact.ErrCorrupt
}
