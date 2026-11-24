package storagecase130

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func size(ctx context.Context, pool *pgxpool.Pool) (any, error) {
	var raw []byte
	err := pool.QueryRow(ctx, `SELECT jsonb_build_object('capturedAt',clock_timestamp(),'model',(SELECT contract FROM organization.storage_contract),'databaseBytes',pg_database_size(current_database()),'walBytes',(SELECT coalesce(sum(size),0) FROM pg_ls_waldir()),'walLSN',pg_current_wal_lsn()::text,
 'counts',jsonb_build_object('BU',(SELECT count(*) FROM organization.business_unit),'department',(SELECT count(*) FROM organization.department),'buSources',(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history),'departmentSources',(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history),'departmentSegments',(SELECT count(*) FROM organization.department_read_segment),'buSegments',(SELECT count(*) FROM organization.bu_read_segment),'departmentDictionaries',(SELECT count(*) FROM organization.department_dictionary),'departmentShapes',(SELECT count(*) FROM organization.department_shape),'securityEvents',(SELECT count(*) FROM access.security_event),'authenticationEvents',(SELECT count(*) FROM access.security_event WHERE event_kind='authentication'),'securityChangeEvents',(SELECT count(*) FROM access.security_event WHERE event_kind='security_change')),
 'historyIds',jsonb_build_object('count',(SELECT sum(cardinality(ids)) FROM organization.department_history),'columnBytes',(SELECT sum(pg_column_size(ids)) FROM organization.department_history)),
 'relations',(SELECT jsonb_agg(x ORDER BY total_bytes DESC) FROM (SELECT n.nspname||'.'||c.relname relation,pg_table_size(c.oid) table_including_toast_bytes,pg_indexes_size(c.oid) indexes_bytes,CASE WHEN c.reltoastrelid=0 THEN 0 ELSE pg_total_relation_size(c.reltoastrelid) END toast_bytes,pg_total_relation_size(c.oid) total_bytes FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access','verification')) x),
 'settings',(SELECT jsonb_object_agg(name,setting) FROM pg_settings WHERE name IN ('fsync','full_page_writes','synchronous_commit','shared_buffers','work_mem','maintenance_work_mem','max_connections','wal_compression','jit')))`).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var value any
	err = json.Unmarshal(raw, &value)
	return value, err
}

func verifySnapshot(ctx context.Context, dir string, db database, pool *pgxpool.Pool, strict bool) (any, error) {
	var manifest snapshot
	if strict {
		if err := readPrivate(filepath.Join(dir, "snapshot.json"), &manifest); err != nil {
			return nil, err
		}
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	results := []map[string]any{}
	for _, t := range manifest.Tables {
		if !strict {
			break
		}
		if strings.HasSuffix(t.Name, "_read_segment") {
			continue
		}
		rows, err := tx.Query(ctx, `SELECT row_to_json(v)::jsonb FROM (SELECT `+columns(t.Columns)+" FROM "+tableName(t)+" ORDER BY "+columns(t.Keys)+") v")
		if err != nil {
			tx.Rollback(context.Background())
			return nil, err
		}
		h := sha256.New()
		var count int64
		for rows.Next() {
			var raw []byte
			if err = rows.Scan(&raw); err != nil {
				break
			}
			_, _ = h.Write(raw)
			_, _ = h.Write([]byte{'\n'})
			count++
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			tx.Rollback(context.Background())
			return nil, err
		}
		if count != t.Rows || hex.EncodeToString(h.Sum(nil)) != t.SHA256 {
			tx.Rollback(context.Background())
			return nil, fmt.Errorf("snapshot values differ in %s", t.Name)
		}
		results = append(results, map[string]any{"table": t.Schema + "." + t.Name, "rows": count, "sha256": t.SHA256, "passed": true})
	}
	if err = tx.Rollback(context.Background()); err != nil {
		return nil, err
	}
	// The output contains only full-fact digests. A/B comparison checks every
	// decoded source, including all history/path/operation values, not heads.
	name := fmt.Sprintf("decoded-source-digests-%d.jsonl.gz", time.Now().UnixNano())
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	z := gzip.NewWriter(f)
	encoder := json.NewEncoder(z)
	count, nodes, segments := 0, 0, 0
	err = bootstrap.VisitHistory(ctx, pool, db.Instance, func(b *bootstrap.BUHistoryFact, d *bootstrap.DepartmentHistoryFact) error {
		var fact any
		var kind, object, source string
		var number int64
		if b != nil {
			fact, kind, object, source, number = b, "BU", b.Object, b.Operation.Source.ID, b.Operation.Source.Number
		} else if d != nil {
			fact, kind, object, source, number = d, "department", d.Object, d.Operation.Source.ID, d.Operation.Source.Number
			segments += len(d.Evidence.Segments)
			for _, s := range d.Evidence.Segments {
				nodes += len(s.Nodes)
			}
		} else {
			return errors.New("decoder did not return a source")
		}
		h := sha256.New()
		if err := json.NewEncoder(h).Encode(fact); err != nil {
			return err
		}
		count++
		return encoder.Encode([]any{kind, object, source, number, hex.EncodeToString(h.Sum(nil))})
	})
	closeZ, closeF := z.Close(), f.Close()
	if err != nil {
		return nil, err
	}
	if closeZ != nil {
		return nil, closeZ
	}
	if closeF != nil {
		return nil, closeF
	}
	return map[string]any{"strictOriginalTableEquality": strict, "tables": results, "decodedSources": count, "frozenPathSegments": segments, "frozenPathNodes": nodes, "sourceDigestFile": name, "scope": "all source values decoded and checksummed; compare every A/B digest row separately"}, nil
}
