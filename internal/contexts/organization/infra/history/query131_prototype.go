//go:build integration && query131

package history

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"github.com/jackc/pgx/v5"
)

// The fault-query harness consumes the formal writer and migration. It no longer
// installs a parallel schema or adds a writer with a source overlay.
//
//go:embed query131_capture.sql
var query131CaptureTailSQL string

// Storage-diagnostic proof; excluded from ordinary product builds.
//
//go:embed query_integrity.sql
var queryIntegritySQL string

//go:embed query131_department_material.sql
var query131DepartmentMaterialSQL string

var Query131CaptureSQL = queryIntegritySQL + strings.Replace(query131CaptureTailSQL, "/*DEPARTMENT_ITEM*/", query131DepartmentMaterialSQL, 1)

func InstallQuery131(ctx context.Context, tx pgx.Tx) error {
	var model string
	var count int
	if err := tx.QueryRow(ctx, `SELECT contract FROM organization.storage_contract`).Scan(&model); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM organization.department`).Scan(&count); err != nil {
		return err
	}
	if model != "m02-compact-5" || count != 0 {
		return errors.New("query131 requires its own empty current-model fixture")
	}
	return nil
}

// Scope values stand in for a future X03-generated plan. This prototype does
// not compile grants and must never be wired to an HTTP request as authorization.
type Query131Input struct {
	Instance, Enterprise, Day, Contains, State string
	All                                        bool
	BUs                                        []string
	Page, PageSize                             int
}
type Query131Stats struct {
	Objects, Segments, Bad, Dictionaries, Shapes int64
	ResponseBytes, DecodedPageSegments           int
}
type Query131Result struct {
	Count int64
	Items []map[string]any
}

func (q Query131Input) Args() []any {
	bus := q.BUs
	if bus == nil {
		bus = []string{}
	}
	return []any{q.Instance, q.Enterprise, q.Day, q.All, bus, strings.ToLower(strings.TrimSpace(q.Contains)), q.State, q.PageSize, (q.Page - 1) * q.PageSize}
}

func CaptureQuery131(ctx context.Context, tx pgx.Tx, q Query131Input) (out Query131Result, stats Query131Stats, err error) {
	if !foundation.ValidDate(q.Day) || q.Page < 1 || q.Page > 1000000 || q.PageSize < 1 || q.PageSize > 100 {
		return out, stats, errors.New("invalid prototype query")
	}
	var count *int64
	var raw []byte
	err = tx.QueryRow(ctx, Query131CaptureSQL, q.Args()...).Scan(&stats.Objects, &stats.Segments, &stats.Bad, &stats.Dictionaries, &stats.Shapes, &count, &raw)
	if err != nil {
		return out, stats, err
	}
	// Never return count or rows from a partially valid projection. This check is
	// unconditional, including empty scope/search/page; no client sees stats.
	if stats.Bad != 0 || count == nil || raw == nil {
		return out, stats, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门查询事实不完整")
	}
	stats.ResponseBytes = len(raw)
	if err = json.Unmarshal(raw, &out.Items); err != nil {
		return Query131Result{}, stats, err
	}
	for _, item := range out.Items {
		if err = ValidateQuery131Captured(ctx, q, item); err != nil {
			return Query131Result{}, stats, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "部门查询事实不完整")
		}
		stats.DecodedPageSegments++
		delete(item, "_material")
	}
	out.Count = *count
	return out, stats, nil
}

// Decode only captured page material, never a second current/catalog lookup.
// Capturing bytes also survives concurrent reclamation of an obsolete dictionary.
func ValidateQuery131Captured(ctx context.Context, q Query131Input, item map[string]any) error {
	get := func(k string) string { v, _ := item[k].(string); return v }
	v, data, err := query131Material(Department, q.Instance, q.Enterprise, get("id"), get("source"), get("start"), get("finish"), get("envelope"), item["ids"], item["_material"])
	if err != nil {
		return err
	}
	decoded, err := DecodeDepartmentRead(ctx, v, data)
	if err != nil {
		return err
	}
	t := decoded.Domain()
	member := d.ScopeRecord(t, q.Day)
	if member == nil || t.ID != get("id") || t.Source != get("source") || t.First != get("firstRecord") || t.Enterprise != q.Enterprise || t.RelationSource != get("relation") || len(t.Records) == 0 {
		return compact.ErrCorrupt
	}
	for k, want := range map[string]string{"name": member.Data.Name, "bu": member.Data.BU, "parent": member.Data.Parent, "record": member.ID, "revision": member.Revision, "date": member.Date, "end": member.End, "firstDate": t.Records[0].Date, "status": d.Status(t, q.Day)} {
		if get(k) != want {
			return compact.ErrCorrupt
		}
	}
	pathDay := ""
	switch d.Status(t, q.Day) {
	case "inactive":
		pathDay = dated.NextDate(member.Date, -1)
	case "not_effective":
		pathDay = t.Records[0].Date
	}
	if get("pathDay") != pathDay || get("codeKey") != strings.ToLower(get("code")) {
		return compact.ErrCorrupt
	}
	var short *string
	if s, ok := item["shortName"].(string); ok {
		short = &s
	}
	if !reflect.DeepEqual(member.Data.ShortName, short) {
		return compact.ErrCorrupt
	}
	return nil
}
