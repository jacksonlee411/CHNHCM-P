//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// upgrade.py supplies the connection of its newly restored, stopped database.
func TestRebuildRestoredJobFunctions(t *testing.T) {
	db := os.Getenv("CHNHCM_T03_RESTORED_DB")
	if db == "" {
		t.Skip("由 T03 临时库升级演练调用")
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("CHNHCM_DATABASE_URL"))
	if err != nil || cfg.ConnConfig.Database != db || cfg.ConnConfig.Host != "127.0.0.1" || os.Getenv("CHNHCM_TEMPORARY") != "true" {
		t.Fatal("需要本轮临时库")
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("无法连接临时库")
	}
	defer pool.Close()
	s := Store{Pool: pool, Instance: os.Getenv("CHNHCM_INSTANCE_ID")}
	if err = s.RebuildReads(ctx); err != nil {
		t.Fatal(err)
	}
}
