package storageverify

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"strings"
	"time"
)

// explainReads runs only on the ownership-checked disposable case. Plans have
// no business labels; output is kept separate from timed acceptance samples.
func explainReads(ctx context.Context, dir string) (any, error) {
	db, pool, err := openDatabase(ctx, dir)
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	out := map[string]json.RawMessage{}
	queries := map[string]string{
		"department-heads":       `SELECT 1::smallint AS kind,(uuid_send(d.department_id)||uuid_send(d.head_source_id)||uuid_send(h.head_source_id))::bytea AS identity FROM organization.department d LEFT JOIN organization.department_hierarchy_relation h USING(instance_id,enterprise_id,department_id) WHERE d.instance_id=$1 UNION ALL SELECT 2::smallint,(uuid_send(b.bu_id)||uuid_send(b.head_source_id))::bytea FROM organization.business_unit b WHERE b.instance_id=$1`,
		"department-current-512": `SELECT b.department_id::text,b.enterprise_id::text,b.code,b.first_record_id::text,b.head_source_id::text,b.head_no,b.head_chunk,h.dictionary_id,b.current_shape,organization.project_ids(h.ids,b.current_refs)::text[],b.current_envelope FROM organization.department b JOIN organization.department_history h ON (h.instance_id,h.object_id,h.chunk_no)=(b.instance_id,b.department_id,b.head_chunk) WHERE b.instance_id=$1 AND ($2 OR b.department_id=ANY($3::text[]::uuid[])) ORDER BY b.department_id`,
	}
	queries["department-segments-512"] = `SELECT b.department_id::text,b.enterprise_id::text,b.code,b.first_record_id::text,b.head_source_id::text,coalesce(lower(s.span)::text,''),coalesce(upper(s.span)::text,''),s.rule,s.dictionary_id,s.shape_id,s.ids::text[],s.envelope FROM jsonb_to_recordset($2::jsonb) w(id uuid,date date) JOIN organization.department b ON b.instance_id=$1 AND b.department_id=w.id JOIN LATERAL (SELECT s.* FROM organization.department_read_segment s WHERE s.instance_id=b.instance_id AND s.object_id=b.department_id AND s.source_id=b.head_source_id AND coalesce(lower(s.span),'-infinity'::date)<=w.date AND s.span @> w.date ORDER BY coalesce(lower(s.span),'-infinity'::date) DESC LIMIT 1) s ON true`

	var ids []string
	rows, err := pool.Query(ctx, "SELECT department_id::text FROM organization.department WHERE instance_id=$1 ORDER BY department_id LIMIT 512", db.Instance)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for name, query := range queries {
		args := []any{db.Instance}
		if name == "department-current-512" {
			args = append(args, false, ids)
		}
		if strings.HasPrefix(name, "department-segments") {
			wanted := []map[string]string{}
			for _, id := range ids {
				wanted = append(wanted, map[string]string{"id": id, "date": time.Now().In(foundation.Shanghai).Format("2006-01-02")})
			}
			raw, e := json.Marshal(wanted)
			if e != nil {
				return nil, e
			}
			args = append(args, raw)
		}
		var plan []byte
		if err = pool.QueryRow(ctx, "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) "+query, args...).Scan(&plan); err != nil {
			return nil, err
		}
		out[name] = plan
	}
	return out, nil
}
