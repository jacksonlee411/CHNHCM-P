package history

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

//go:embed department_selection_query.sql
var departmentSelectionTailSQL string

// Full selection shares the ordinary page's fact scope and transfers required
// immutable material as native columns instead of one giant JSON aggregate.
// UNION deduplicates (identity, date) closure and terminates even for a cycle.
// Go's existing path resolver rejects cycles before any page/summary/CSV escapes.
var DepartmentSelectionQuerySQL = departmentQueryFactsSQL + departmentQueryScopeSQL + strings.Replace(departmentSelectionTailSQL, "/*ENTERPRISE_ITEM*/", queryEnterpriseMaterialSQL, 1)

type selectionMaterial struct {
	kind                          int
	meta                          []pgtype.Text
	values                        compact.Values
	ranges                        []pgtype.Range[pgtype.Date]
	parts                         [][]byte
	dictionaryDigest, shapeDigest []byte
}

func captureDepartmentSet(ctx context.Context, tx pgx.Tx, p access.ReadPlan, q app.QueryRequest, plan, query []byte) (app.QueryCapture, error) {
	out := app.QueryCapture{ByDay: map[string]d.Snapshot{}}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer foundation.BeginStage(ctx, "department.query.capture_stream")()
	// Preserve the existing selection workspace while changing only read duties.
	// Use the actual policy/query plan for this transaction; SQL and HTTP
	// budgets remain unchanged.
	if _, err := tx.Exec(ctx, "SET LOCAL work_mem='64MB'; SET LOCAL plan_cache_mode='force_custom_plan'"); err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, "DECLARE selection_facts133 NO SCROLL CURSOR FOR "+DepartmentSelectionQuerySQL, p.Instance, p.Enterprise, p.Day, plan, query, q.Input.PageSize, (q.Input.Page-1)*q.Input.PageSize, true); err != nil {
		return out, err
	}
	// Native material batches all belong to the declaration's one snapshot.
	// Full capture exhausts it; neither pages nor transactions survive a request.
	decoder := selectionDecoder{dates: map[pgtype.Date]string{}}
	headerSeen := false
	var pending []selectionMaterial
	// Nothing is returned until the statement, header and every Go check succeed.
	batchSize := 0
	captureBatch := func(rows pgx.Rows) error {
		defer rows.Close()
		if len(rows.FieldDescriptions()) != 9 {
			return queryUnavailable()
		}
		for rows.Next() {
			batchSize++
			if e := ctx.Err(); e != nil {
				return e
			}
			values := rows.RawValues()
			if len(values) != 9 {
				return queryUnavailable()
			}
			for _, value := range values {
				out.Stats.MaterialBytes += len(value)
			}
			var m selectionMaterial
			if e := rows.Scan(&m.kind, &m.meta, &m.values.Strings, &m.values.Binary, &m.ranges, &m.parts, &m.dictionaryDigest, &m.shapeDigest, &m.values.IDs); e != nil {
				return e
			}
			if m.kind == 0 {
				if headerSeen || len(m.meta) != 1 || !m.meta[0].Valid {
					return queryUnavailable()
				}
				headerSeen = true
				var header struct {
					Objects, Segments, Bad, Dictionaries, Shapes, Denied, Count int64
					Enterprise                                                  capturedEnterprise `json:"enterpriseProfile"`
					Missing                                                     bool
					Page                                                        []string
				}
				if json.Unmarshal([]byte(m.meta[0].String), &header) != nil || header.Bad != 0 || header.Missing || header.Count < 0 || uint64(header.Count) > uint64(^uint(0)>>1) {
					return queryUnavailable()
				}
				if header.Denied != 0 {
					return scopeQueryError(header.Denied)
				}
				out.Count = int(header.Count)
				out.PageIDs = header.Page
				if _, exists := out.ByDay[p.Day]; !exists {
					out.ByDay[p.Day] = d.Snapshot{Departments: make([]d.Timeline, 0, len(header.Page))}
				}
				var err error
				out.EnterpriseName, err = decodeQueryEnterpriseName(ctx, p.Enterprise, p.Day, header.Enterprise)
				if err != nil {
					return err
				}
				out.Stats.Objects = header.Objects
				out.Stats.Segments = header.Segments
				out.Stats.Dictionaries = header.Dictionaries
				out.Stats.Shapes = header.Shapes

			} else if !decoder.ready(m) {
				// UNION ALL may deliver facts before their shared material. Do
				// not sort the entire wide result just to enforce transfer order.
				pending = append(pending, m)
			} else if e := decoder.decode(ctx, p, m, &out); e != nil {
				return e
			}
		}
		return rows.Err()
	}
	captureErr := func() error {
		for {
			rows, e := tx.Query(ctx, "FETCH FORWARD 256 FROM selection_facts133")
			if e != nil {
				return e
			}
			batchSize = 0
			if e = captureBatch(rows); e != nil {
				return e
			}
			if batchSize < 256 {
				break
			}
		}
		if _, e := tx.Exec(ctx, "CLOSE selection_facts133"); e != nil {
			return e
		}
		if !headerSeen {
			return queryUnavailable()
		}
		for _, m := range pending {
			if e := ctx.Err(); e != nil {
				return e
			}
			if e := decoder.decode(ctx, p, m, &out); e != nil {
				return e
			}
		}
		return nil
	}()
	if captureErr != nil {
		cancel()
		return app.QueryCapture{}, captureErr
	}
	for day, s := range out.ByDay {
		out.ByDay[day] = s.ReadIndexed()
	}
	return out, nil
}

type selectionDecoder struct {
	dates        map[pgtype.Date]string
	dictionaries map[string]compact.Values
	shapes       map[string][]byte
}

func (decoder *selectionDecoder) ready(m selectionMaterial) bool {
	if m.kind != 3 {
		return true
	}
	_, dictionary := decoder.dictionaries[string(m.dictionaryDigest)]
	_, shape := decoder.shapes[string(m.shapeDigest)]
	return (dictionary || m.values.Strings != nil || m.values.Binary != nil || m.ranges != nil) && (shape || m.parts != nil)
}

// Formatting and verified shared material live only in this capture. Every
// dictionary/shape is decoded as needed, and each fact retains its own UUID
// directory and source identity. Storage auditing belongs outside this read.
func (decoder *selectionDecoder) date(value pgtype.Date) string {
	if text, ok := decoder.dates[value]; ok {
		return text
	}
	text := value.Time.Format("2006-01-02")
	decoder.dates[value] = text
	return text
}

func (decoder *selectionDecoder) decode(ctx context.Context, p access.ReadPlan, m selectionMaterial, out *app.QueryCapture) error {
	switch m.kind {
	case -2:
		if len(m.dictionaryDigest) != 32 || len(m.meta) != 0 || len(m.values.IDs) != 0 || len(m.parts) != 0 || len(m.shapeDigest) != 0 {
			return queryUnavailable()
		}
		if e := decoder.ranges(&m); e != nil {
			return e
		}
		if decoder.dictionaries == nil {
			decoder.dictionaries = map[string]compact.Values{}
		}
		key := string(m.dictionaryDigest)
		if _, exists := decoder.dictionaries[key]; exists {
			return queryUnavailable()
		}
		decoder.dictionaries[key] = m.values
		return nil
	case -1:
		if len(m.shapeDigest) != 32 || len(m.meta) != 0 || len(m.values.IDs) != 0 || len(m.values.Strings) != 0 || len(m.values.Binary) != 0 || len(m.ranges) != 0 || len(m.dictionaryDigest) != 0 {
			return queryUnavailable()
		}
		if decoder.shapes == nil {
			decoder.shapes = map[string][]byte{}
		}
		key := string(m.shapeDigest)
		if _, exists := decoder.shapes[key]; exists {
			return queryUnavailable()
		}
		decoder.shapes[key] = bytes.Join(m.parts, nil)
		return nil
	case 3:
		return decoder.boundDepartment(ctx, p, m, out)
	case 2:
		// BU facts use the same native columns as department facts. Preserve
		// the original source/day checks without a JSON/hex round trip.
		if len(m.meta) != 6 {
			return queryUnavailable()
		}
		for _, value := range m.meta {
			if !value.Valid {
				return queryUnavailable()
			}
		}
		id, day, enterprise, code := m.meta[0].String, m.meta[1].String, m.meta[2].String, m.meta[3].String
		source := m.meta[4].String
		if enterprise != p.Enterprise || source != m.meta[5].String || !foundation.ValidDate(day) {
			return queryUnavailable()
		}
		for _, r := range m.ranges {
			// Match the former coalesce(lower/upper(range), '') transport,
			// including unused empty/null/unbounded dictionary ranges.
			v := compact.Range{}
			if r.Valid {
				if r.LowerType != pgtype.Unbounded && r.LowerType != pgtype.Empty {
					v.Start = decoder.nativeBUDate(r.Lower)
				}
				if r.UpperType != pgtype.Unbounded && r.UpperType != pgtype.Empty {
					v.End = decoder.nativeBUDate(r.Upper)
				}
			}
			m.values.Ranges = append(m.values.Ranges, v)
		}
		data := bytes.Join(m.parts, nil)
		decoded, e := DecodeBURead(ctx, m.values, data)
		if e != nil {
			return errors.Join(queryUnavailable(), e)
		}
		b := decoded.Timeline
		if b.ID != id || b.Source != source || len(b.Records) == 0 || b.First != b.Records[0].ID {
			return queryUnavailable()
		}
		b.Code = code
		s := out.ByDay[day]
		s.BusinessUnits = append(s.BusinessUnits, b)
		out.ByDay[day] = s
		out.Stats.DecodedBUs++
	default:
		return queryUnavailable()
	}
	return nil
}

func (decoder *selectionDecoder) material(m *selectionMaterial) ([]byte, error) {
	if len(m.dictionaryDigest) != 32 || len(m.shapeDigest) != 32 {
		return nil, queryUnavailable()
	}
	if m.values.Strings == nil && m.values.Binary == nil && m.ranges == nil {
		values, found := decoder.dictionaries[string(m.dictionaryDigest)]
		if !found {
			return nil, queryUnavailable()
		}
		values.IDs = m.values.IDs
		m.values = values
	} else {
		if e := decoder.ranges(m); e != nil {
			return nil, e
		}
	}
	if m.parts == nil {
		data, found := decoder.shapes[string(m.shapeDigest)]
		if !found {
			return nil, queryUnavailable()
		}
		return data, nil
	}
	if len(m.parts) == 1 {
		return m.parts[0], nil
	}
	return bytes.Join(m.parts, nil), nil
}

// A narrow source segment contains the relation identities needed by path
// versions. Decode it once; do not rebuild query columns or storage seals.
func (decoder *selectionDecoder) boundDepartment(ctx context.Context, p access.ReadPlan, m selectionMaterial, out *app.QueryCapture) error {
	if len(m.meta) != 7 {
		return queryUnavailable()
	}
	for _, value := range m.meta {
		if !value.Valid {
			return queryUnavailable()
		}
	}
	day, id, code, source := m.meta[0].String, m.meta[1].String, m.meta[2].String, m.meta[3].String
	if !foundation.ValidDate(day) || source != m.meta[6].String {
		return queryUnavailable()
	}
	data, err := decoder.material(&m)
	if err != nil {
		return err
	}
	decoded, err := DecodeDepartmentRead(ctx, m.values, data)
	if err != nil {
		return errors.Join(queryUnavailable(), err)
	}
	t := decoded.Domain()
	if t.ID != id || t.Enterprise != p.Enterprise || t.Source != source || t.First != m.meta[4].String || t.RelationSource != m.meta[5].String || len(t.Records) == 0 {
		return queryUnavailable()
	}
	t.Code = code
	snapshot := out.ByDay[day]
	snapshot.Departments = append(snapshot.Departments, t)
	out.ByDay[day] = snapshot
	out.Stats.DecodedDepartments++
	return nil
}

func (decoder *selectionDecoder) ranges(m *selectionMaterial) error {
	for _, r := range m.ranges {
		if !r.Valid {
			return queryUnavailable()
		}
		v := compact.Range{}
		if r.LowerType != pgtype.Unbounded && r.LowerType != pgtype.Empty {
			v.Start = decoder.date(r.Lower)
		}
		if r.UpperType != pgtype.Unbounded && r.UpperType != pgtype.Empty {
			v.End = decoder.date(r.Upper)
		}
		m.values.Ranges = append(m.values.Ranges, v)
	}
	return nil
}

// PostgreSQL date text preserves explicit infinities. This keeps the native
// transport equivalent to the former SQL conversion, not a new date rule.
func (decoder *selectionDecoder) nativeBUDate(value pgtype.Date) string {
	switch value.InfinityModifier {
	case pgtype.Infinity:
		return "infinity"
	case pgtype.NegativeInfinity:
		return "-infinity"
	default:
		return decoder.date(value)
	}
}
