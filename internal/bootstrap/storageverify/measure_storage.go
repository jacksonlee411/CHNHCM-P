package storageverify

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/bootstrap/offlinestorage"
	"github.com/jackc/pgx/v5"
)

func databaseSize(ctx context.Context, dir string) (any, error) {
	_, pool, e := openDatabase(ctx, dir)
	if e != nil {
		return nil, e
	}
	defer pool.Close()
	var raw []byte
	e = pool.QueryRow(ctx, `SELECT jsonb_build_object('capturedAt',clock_timestamp(),'model',(SELECT contract FROM organization.storage_contract),'databaseBytes',pg_database_size(current_database()),'databaseMB',pg_database_size(current_database())/1000000.0,'databaseMiB',pg_database_size(current_database())/1048576.0,'walBytes',(SELECT coalesce(sum(size),0) FROM pg_ls_waldir()),'walLSN',pg_current_wal_lsn()::text,'sessionRows',(SELECT count(*) FROM access.session),'sessionUpdates',(SELECT coalesce(n_tup_upd,0) FROM pg_stat_user_tables WHERE schemaname='access' AND relname='session'),'counts',jsonb_build_object('BU',(SELECT count(*) FROM organization.business_unit),'department',(SELECT count(*) FROM organization.department),'buSources',(SELECT sum(cardinality(source_refs)) FROM organization.bu_history),'departmentSources',(SELECT sum(cardinality(source_refs)) FROM organization.department_history),'buChunks',(SELECT count(*) FROM organization.bu_history),'departmentChunks',(SELECT count(*) FROM organization.department_history),'securityEvents',(SELECT count(*) FROM access.security_event),'authenticationEvents',(SELECT count(*) FROM access.security_event WHERE event_kind='authentication'),'securityChangeEvents',(SELECT count(*) FROM access.security_event WHERE event_kind='security_change')),'relations',(SELECT jsonb_agg(x ORDER BY total_bytes DESC) FROM (SELECT n.nspname||'.'||c.relname AS relation,pg_relation_size(c.oid) AS main_bytes,pg_table_size(c.oid) AS table_including_toast_bytes,CASE WHEN c.reltoastrelid=0 THEN 0 ELSE pg_total_relation_size(c.reltoastrelid) END AS toast_bytes,pg_indexes_size(c.oid) AS indexes_bytes,pg_total_relation_size(c.oid) AS total_bytes FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access')) x),'settings',(SELECT jsonb_object_agg(name,setting) FROM pg_settings WHERE name IN ('fsync','full_page_writes','synchronous_commit','shared_buffers','work_mem','maintenance_work_mem','max_connections','wal_compression','jit')))`).Scan(&raw)
	if e != nil {
		return nil, e
	}
	var out any
	e = json.Unmarshal(raw, &out)
	return out, e
}
func maintainDatabase(ctx context.Context, dir string, compact, dryRun bool) (any, error) {
	db, pool, e := openDatabase(ctx, dir)
	if e != nil {
		return nil, e
	}
	defer pool.Close()
	started := time.Now()
	maintenance, e := bootstrap.MaintainHistory(ctx, pool, bootstrap.HistoryMaintenanceInput{Instance: db.Instance, OwnedInstances: []string{db.Instance}, DryRun: dryRun})
	if e != nil {
		return nil, e
	}
	if dryRun {
		return maintenance, nil
	}
	if _, e = pool.Exec(ctx, "VACUUM (ANALYZE)"); e != nil {
		return nil, e
	}
	if compact {
		if _, e = offlinestorage.CheckRewriteSpace(ctx, pool, "chnhcm-storage126-product"); e != nil {
			return nil, e
		}
		// Explicitly offline, task-owned layout compaction. No history is removed.
		rows, e := pool.Query(ctx, `SELECT n.nspname,c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access') ORDER BY n.nspname,c.relname`)
		if e != nil {
			return nil, e
		}
		tables := []pgx.Identifier{}
		for rows.Next() {
			var schema, table string
			if e = rows.Scan(&schema, &table); e != nil {
				rows.Close()
				return nil, e
			}
			tables = append(tables, pgx.Identifier{schema, table})
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		for _, table := range tables {
			if _, e = pool.Exec(ctx, "VACUUM (FULL,ANALYZE) "+table.Sanitize()); e != nil {
				return nil, fmt.Errorf("compact %s: %w", table, e)
			}
		}
	}
	size, e := databaseSize(ctx, dir)
	if e != nil {
		return nil, e
	}
	return map[string]any{"unreferencedDictionariesRemoved": maintenance.Removed, "maintenance": maintenance, "offlineTableRewrite": compact, "seconds": time.Since(started).Seconds(), "storage": size}, nil
}
