package storagecase130

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func init() {
	extraModes["identity-cost"] = identityCost
}

// This is a disposable SQL representation experiment, never a product format.
// TEMP tables preserve the complete ownership and UUID mapping. Their WAL is
// not comparable to logged product tables and is deliberately not called a gain.
func identityCost(ctx context.Context, db database, pool *pgxpool.Pool) (any, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(ctx, `CREATE TEMP TABLE identity130_full(instance uuid,enterprise uuid,object uuid,chunk integer,ids uuid[] NOT NULL,PRIMARY KEY(instance,object,chunk)) ON COMMIT DROP;
CREATE TEMP TABLE identity130_map(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,instance uuid NOT NULL,enterprise uuid NOT NULL,original uuid NOT NULL,UNIQUE(instance,enterprise,original)) ON COMMIT DROP;
CREATE TEMP TABLE identity130_short(instance uuid,enterprise uuid,object uuid,chunk integer,ids bigint[] NOT NULL,PRIMARY KEY(instance,object,chunk)) ON COMMIT DROP;`)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	_, err = tx.Exec(ctx, `INSERT INTO identity130_full SELECT instance_id,enterprise_id,object_id,chunk_no,ids FROM organization.department_history WHERE instance_id=$1 ORDER BY instance_id,object_id,chunk_no`, db.Instance)
	if err != nil {
		return nil, err
	}
	fullWrite := time.Since(started).Seconds()
	started = time.Now()
	_, err = tx.Exec(ctx, `INSERT INTO identity130_map(instance,enterprise,original) SELECT DISTINCT instance,enterprise,u FROM identity130_full CROSS JOIN LATERAL unnest(ids) u ORDER BY instance,enterprise,u`)
	if err != nil {
		return nil, err
	}
	mapWrite := time.Since(started).Seconds()
	started = time.Now()
	_, err = tx.Exec(ctx, `INSERT INTO identity130_short SELECT f.instance,f.enterprise,f.object,f.chunk,array_agg(m.id ORDER BY r.n) FROM identity130_full f CROSS JOIN LATERAL unnest(f.ids) WITH ORDINALITY r(original,n) JOIN identity130_map m ON (m.instance,m.enterprise,m.original)=(f.instance,f.enterprise,r.original) GROUP BY f.instance,f.enterprise,f.object,f.chunk ORDER BY f.instance,f.object,f.chunk`)
	if err != nil {
		return nil, err
	}
	shortWrite := time.Since(started).Seconds()
	_, err = tx.Exec(ctx, `ANALYZE identity130_full;ANALYZE identity130_map;ANALYZE identity130_short`)
	if err != nil {
		return nil, err
	}
	started = time.Now()
	var fullBytes int64
	if err = tx.QueryRow(ctx, `SELECT sum(octet_length(array_send(ids))) FROM identity130_full`).Scan(&fullBytes); err != nil {
		return nil, err
	}
	fullDirectRead := time.Since(started).Seconds()
	started = time.Now()
	var mismatch, rows, entries int64
	err = tx.QueryRow(ctx, `SELECT count(*) FILTER(WHERE f.ids IS DISTINCT FROM v.ids),count(*),sum(cardinality(f.ids)) FROM identity130_full f FULL JOIN (SELECT s.instance,s.object,s.chunk,array_agg(m.original ORDER BY r.n) ids FROM identity130_short s CROSS JOIN LATERAL unnest(s.ids) WITH ORDINALITY r(id,n) LEFT JOIN identity130_map m ON m.id=r.id AND m.instance=s.instance AND m.enterprise=s.enterprise GROUP BY s.instance,s.object,s.chunk) v USING(instance,object,chunk)`).Scan(&mismatch, &rows, &entries)
	if err != nil || mismatch != 0 {
		return nil, errors.New("identity cost experiment failed complete UUID ownership roundtrip")
	}
	fullRead := time.Since(started).Seconds()
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT jsonb_build_object('mappingRows',(SELECT count(*) FROM identity130_map),'fullArrayBytes',(SELECT sum(pg_column_size(ids)) FROM identity130_full),'shortArrayBytes',(SELECT sum(pg_column_size(ids)) FROM identity130_short),'relations',(SELECT jsonb_agg(jsonb_build_object('name',relname,'tableBytes',pg_table_size(oid),'indexBytes',pg_indexes_size(oid),'totalBytes',pg_total_relation_size(oid)) ORDER BY relname) FROM pg_class WHERE oid IN ('identity130_full'::regclass,'identity130_map'::regclass,'identity130_short'::regclass)))`).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var sizes any
	if err = json.Unmarshal(raw, &sizes); err != nil {
		return nil, err
	}
	return map[string]any{"rows": rows, "identityEntries": entries, "allUUIDsAndOwnersRoundtripped": true, "mismatches": mismatch, "fullUUIDWriteSeconds": fullWrite, "mappingWriteSeconds": mapWrite, "shortReferenceWriteSeconds": shortWrite, "fullDirectReadSeconds": fullDirectRead, "completeMappedReadAndCompareSeconds": fullRead, "sizes": sizes, "temporaryTablesRolledBack": true, "scope": "equal identity-array component with complete owner-scoped mapping and indexes; direct read materializes all arrays, mapped read also validates all rows; natural warm caches, no same-work read speed ratio claimed; TEMP WAL is not logged-product WAL; no product identity-format adoption"}, nil
}
