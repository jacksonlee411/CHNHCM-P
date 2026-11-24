package main

import (
	"context"
	"encoding/json"
)

// Large append-only fixture loading can leave visibility/statistics maintenance
// pending. Finish ordinary maintenance once before steady-state capacity runs;
// cold measurements subsequently evict these pages again. Never VACUUM FULL,
// rewrite facts, delete history, disable durability or warm timed requests here.
func (r *runner) maintain() (any, error) {
	ctx := context.Background()
	observe := func() (object, error) {
		var raw []byte
		e := r.db.QueryRow(ctx, `SELECT jsonb_build_object('heapPages',relpages,'allVisiblePages',relallvisible,'estimatedRows',reltuples,'liveMembers',(SELECT count(*) FROM organization.source_member)) FROM pg_class WHERE oid='organization.source_member'::regclass`).Scan(&raw)
		if e != nil {
			return nil, e
		}
		v := object{}
		e = json.Unmarshal(raw, &v)
		return v, e
	}
	before, e := observe()
	if e != nil {
		return nil, e
	}
	_, e = r.db.Exec(ctx, `VACUUM (ANALYZE) organization.source_member`)
	if e != nil {
		return nil, e
	}
	after, e := observe()
	return object{"statement": "VACUUM (ANALYZE) organization.source_member", "before": before, "after": after, "boundary": "ordinary task database maintenance after fixture preparation; cold reset still required before cold timing"}, e
}
