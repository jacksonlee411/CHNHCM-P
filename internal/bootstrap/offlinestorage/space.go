// Package offlinestorage contains preconditions shared by disposable, offline
// verification tools. The caller must first verify database/container ownership.
package offlinestorage

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RewriteSpace struct {
	AvailableBytes int64 `json:"availableBytes"`
	RequiredBytes  int64 `json:"requiredBytes"`
}

// CheckRewriteSpace reserves temporary table/index space and WAL headroom.
// It runs before the first FULL; a successful check is not a disk reservation.
func CheckRewriteSpace(ctx context.Context, pool *pgxpool.Pool, container string) (RewriteSpace, error) {
	var total, largest int64
	result := RewriteSpace{}
	err := pool.QueryRow(ctx, `SELECT coalesce(sum(pg_total_relation_size(c.oid)),0),coalesce(max(pg_total_relation_size(c.oid)),0) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access','verification')`).Scan(&total, &largest)
	if err != nil {
		return result, err
	}
	out, err := exec.CommandContext(ctx, "docker", "exec", container, "df", "-B1", "/var/lib/postgresql").Output()
	if err != nil {
		return result, errors.New("offline rewrite free space could not be verified")
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	fields := strings.Fields(lines[len(lines)-1])
	if len(lines) < 2 || len(fields) < 4 {
		return result, errors.New("offline rewrite free space could not be parsed")
	}
	result.AvailableBytes, err = strconv.ParseInt(fields[3], 10, 64)
	result.RequiredBytes = 2*total + largest + 64*1024*1024
	if err != nil || result.AvailableBytes < result.RequiredBytes {
		return result, errors.New("insufficient verified space for offline rewrite")
	}
	return result, nil
}
