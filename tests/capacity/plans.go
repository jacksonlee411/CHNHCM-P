package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// Read the actual current SQL literals so the observation cannot silently drift
// into a simplified query that the running product never executes.
func sqlLiteral(path, contains string) (string, error) {
	f, e := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if e != nil {
		return "", e
	}
	matches := []string{}
	ast.Inspect(f, func(n ast.Node) bool {
		if l, ok := n.(*ast.BasicLit); ok && l.Kind == token.STRING {
			v, e := strconv.Unquote(l.Value)
			if e == nil && strings.Contains(v, contains) {
				matches = append(matches, v)
			}
		}
		return true
	})
	if len(matches) != 1 {
		return "", fmt.Errorf("expected one current SQL literal for %s, got %d", contains, len(matches))
	}
	return matches[0], nil
}
func (r *runner) plans() (any, error) {
	ctx := context.Background()
	var bu, head, old string
	if e := r.db.QueryRow(ctx, `SELECT b.bu_id,b.head_source_id,s.source_id FROM organization.business_unit b JOIN organization.source_version s USING(instance_id,bu_id) WHERE b.instance_id=$1 AND b.code='T07_CAP_0000' AND s.source_no=25`, r.instance).Scan(&bu, &head, &old); e != nil {
		return nil, e
	}
	var grants []byte
	if e := r.db.QueryRow(ctx, `SELECT jsonb_agg(jsonb_build_object('scopeKind',scope_kind,'scopeId',scope_id,'readFields',read_fields)) FROM access.bound_permission WHERE instance_id=$1 AND principal_id=$2 AND action='bu.read' AND purpose='bu' AND (valid_from IS NULL OR valid_from<=statement_timestamp()) AND (valid_until IS NULL OR statement_timestamp()<valid_until)`, r.instance, r.subject).Scan(&grants); e != nil {
		return nil, e
	}
	list, e := sqlLiteral("internal/contexts/organization/infra/postgres/queries.go", "WITH grants AS")
	if e != nil {
		return nil, e
	}
	members, e := sqlLiteral("internal/contexts/organization/infra/postgres/generated/queries.sql.go", "-- name: GetMembers")
	if e != nil {
		return nil, e
	}
	load, e := sqlLiteral("internal/contexts/organization/infra/postgres/timeline.go", "SELECT r.record_id::text,r.revision_id::text,r.revision_no")
	if e != nil {
		return nil, e
	}
	result := object{"role": "chnhcm_api", "source": "exact current Go SQL literals", "environment": r.environment()}
	for _, q := range []struct {
		name, sql string
		args      []any
	}{
		{"list_page_1", list, []any{r.instance, grants, "2026-11-01", "", "all", 1, 20, "code"}},
		{"current_members", members, []any{r.instance, bu, head}},
		{"exact_source_25", members, []any{r.instance, bu, old}},
		{"candidate_load", load, []any{r.instance, bu, head}},
	} {
		tx, e := r.db.Begin(ctx)
		if e != nil {
			return nil, e
		}
		if _, e = tx.Exec(ctx, "SET LOCAL ROLE chnhcm_api; SET LOCAL statement_timeout='60s'"); e != nil {
			_ = tx.Rollback(ctx)
			return nil, e
		}
		var raw []byte
		e = tx.QueryRow(ctx, "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) "+q.sql, q.args...).Scan(&raw)
		_ = tx.Rollback(ctx)
		if e != nil {
			return nil, e
		}
		var plan any
		if e = json.Unmarshal(raw, &plan); e != nil {
			return nil, e
		}
		result[q.name] = object{"sql": q.sql, "plan": plan}
	}
	return result, nil
}
