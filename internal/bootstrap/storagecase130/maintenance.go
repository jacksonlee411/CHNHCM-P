package storagecase130

import (
	"context"
	"fmt"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/bootstrap/offlinestorage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func init() {
	for _, mode := range []string{"maintain-dry-run", "maintain", "compact"} {
		extraModes[mode] = func(ctx context.Context, db database, pool *pgxpool.Pool) (any, error) {
			// Model admission and complete disposable ownership were checked by
			// open before any maintenance, vacuum or table rewrite.
			result, err := bootstrap.MaintainHistory(ctx, pool, bootstrap.HistoryMaintenanceInput{Instance: db.Instance, OwnedInstances: []string{db.Instance}, DryRun: mode == "maintain-dry-run"})
			if err != nil || mode == "maintain-dry-run" {
				return result, err
			}
			if _, err = pool.Exec(ctx, "VACUUM (ANALYZE)"); err != nil {
				return nil, err
			}
			if mode == "compact" {
				if _, err = offlinestorage.CheckRewriteSpace(ctx, pool, db.Container); err != nil {
					return nil, err
				}
				rows, err := pool.Query(ctx, `SELECT n.nspname,c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access','verification') ORDER BY n.nspname,c.relname`)
				if err != nil {
					return nil, err
				}
				tables := []pgx.Identifier{}
				for rows.Next() {
					var schema, name string
					if err = rows.Scan(&schema, &name); err != nil {
						rows.Close()
						return nil, err
					}
					tables = append(tables, pgx.Identifier{schema, name})
				}
				err = rows.Err()
				rows.Close()
				if err != nil {
					return nil, err
				}
				for _, table := range tables {
					if _, err = pool.Exec(ctx, "VACUUM (FULL,ANALYZE) "+table.Sanitize()); err != nil {
						return nil, fmt.Errorf("offline rewrite failed: %w", err)
					}
				}
			}
			storage, err := size(ctx, pool)
			return map[string]any{"maintenance": result, "offlineTableRewrite": mode == "compact", "storage": storage}, err
		}
	}
}
