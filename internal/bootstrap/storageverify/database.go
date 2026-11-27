package storageverify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"github.com/jackc/pgx/v5/pgxpool"
)

const containerName = "chnhcm-storage126-product"
const volumeName = "chnhcm-storage126-product-data"

type privateDatabase struct {
	Container, Volume, AdminDSN, APIDSN, Instance string
	Port                                          string
}

func secret() string {
	v := make([]byte, 32)
	if _, e := rand.Read(v); e != nil {
		panic(e)
	}
	return hex.EncodeToString(v)
}
func docker(args ...string) ([]byte, error) {
	out, e := exec.Command("docker", args...).CombinedOutput()
	if e != nil {
		return nil, fmt.Errorf("task Docker command %s failed: %s", args[0], strings.TrimSpace(string(out)))
	}
	return out, nil
}
func writeJSON(path string, v any, mode os.FileMode) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, append(b, '\n'), mode)
}
func externalDirectory(dir string) error {
	abs, e := filepath.Abs(dir)
	if e != nil {
		return e
	}
	cwd, e := os.Getwd()
	if e != nil {
		return e
	}
	rel, e := filepath.Rel(cwd, abs)
	if e != nil || rel == "." || !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("private verification files require a directory outside checkout")
	}
	info, e := os.Stat(abs)
	if e != nil {
		return e
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("private verification directory must have mode 0700")
	}
	return nil
}
func prepareDatabase(ctx context.Context, dir string) (any, error) {
	if e := externalDirectory(dir); e != nil {
		return nil, e
	}
	if exec.Command("docker", "inspect", containerName).Run() == nil || exec.Command("docker", "volume", "inspect", volumeName).Run() == nil {
		return nil, fmt.Errorf("task resource already exists; refusing overwrite")
	}
	image, e := os.ReadFile("deploy/local/postgres-image.txt")
	if e != nil {
		return nil, e
	}
	db := privateDatabase{Container: containerName, Volume: volumeName}
	password, apiPassword := secret(), secret()
	if _, e = docker("volume", "create", "--label", "io.chnhcm.task=storage126-product", "--label", "io.chnhcm.temporary=true", volumeName); e != nil {
		return nil, e
	}
	keep := false
	defer func() {
		if !keep {
			_, _ = docker("rm", "-f", "-v", containerName)
			_, _ = docker("volume", "rm", volumeName)
		}
	}()
	_, e = docker("run", "-d", "--name", containerName, "--label", "io.chnhcm.task=storage126-product", "--label", "io.chnhcm.temporary=true", "-p", "127.0.0.1::5432", "-e", "POSTGRES_DB=storage126", "-e", "POSTGRES_PASSWORD="+password, "-e", "POSTGRES_INITDB_ARGS=--encoding=UTF8 --locale=C", "-v", volumeName+":/var/lib/postgresql", strings.TrimSpace(string(image)), "postgres", "-c", "synchronous_commit=on", "-c", "timezone=UTC")
	if e != nil {
		return nil, e
	}
	ports, e := docker("port", containerName, "5432/tcp")
	if e != nil {
		return nil, e
	}
	_, db.Port, e = net.SplitHostPort(strings.TrimSpace(string(ports)))
	if e != nil {
		return nil, e
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword("postgres", password), Host: net.JoinHostPort("127.0.0.1", db.Port), Path: "/storage126", RawQuery: "sslmode=disable&application_name=storage126"}
	db.AdminDSN = u.String()
	u.User = url.UserPassword("chnhcm_api", apiPassword)
	db.APIDSN = u.String()
	var pool *pgxpool.Pool
	for until := time.Now().Add(30 * time.Second); time.Now().Before(until); {
		pool, e = pgxpool.New(ctx, db.AdminDSN)
		if e == nil {
			e = pool.Ping(ctx)
			if e == nil {
				break
			}
			pool.Close()
		}
		time.Sleep(250 * time.Millisecond)
	}
	if e != nil {
		return nil, fmt.Errorf("candidate PostgreSQL readiness failed")
	}
	defer pool.Close()
	if e = bootstrap.Migrate(ctx, pool, migrations.Files); e != nil {
		return nil, e
	}
	if e = bootstrap.ConfigureLocalRoles(ctx, pool, apiPassword); e != nil {
		return nil, e
	}
	if e = writeJSON(filepath.Join(dir, "candidate-private.json"), db, 0600); e != nil {
		return nil, e
	}
	keep = true
	var size int64
	if e = pool.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&size); e != nil {
		return nil, e
	}
	return map[string]any{"container": containerName, "volume": volumeName, "temporary": true, "privateConfig": filepath.Join(dir, "candidate-private.json"), "initialDatabaseBytes": size, "port": db.Port}, nil
}
func openDatabase(ctx context.Context, dir string, migrate ...bool) (privateDatabase, *pgxpool.Pool, error) {
	var db privateDatabase
	if e := externalDirectory(dir); e != nil {
		return db, nil, e
	}
	info, e := os.Stat(filepath.Join(dir, "candidate-private.json"))
	if e != nil {
		return db, nil, e
	}
	if info.Mode().Perm() != 0600 {
		return db, nil, fmt.Errorf("private config permissions changed")
	}
	raw, e := os.ReadFile(filepath.Join(dir, "candidate-private.json"))
	if e != nil {
		return db, nil, e
	}
	if e = json.Unmarshal(raw, &db); e != nil {
		return db, nil, e
	}
	if db.Container != containerName || db.Volume != volumeName {
		return db, nil, fmt.Errorf("foreign candidate resource")
	}
	labels, e := docker("inspect", "--format", "{{json .Config.Labels}}", db.Container)
	if e != nil {
		return db, nil, e
	}
	var l map[string]string
	if e = json.Unmarshal(labels, &l); e != nil {
		return db, nil, e
	}
	if l["io.chnhcm.task"] != "storage126-product" || l["io.chnhcm.temporary"] != "true" {
		return db, nil, fmt.Errorf("candidate ownership mismatch")
	}
	// Docker may allocate a new host port when a randomly published port is
	// restarted. Resolve the owned container again; credentials remain private.
	ports, e := docker("port", db.Container, "5432/tcp")
	if e != nil {
		return db, nil, e
	}
	host, port, e := net.SplitHostPort(strings.TrimSpace(string(ports)))
	if e != nil || host != "127.0.0.1" {
		return db, nil, fmt.Errorf("candidate has no single loopback database port")
	}
	for _, dsn := range []*string{&db.AdminDSN, &db.APIDSN} {
		u, err := url.Parse(*dsn)
		if err != nil {
			return db, nil, fmt.Errorf("invalid private candidate connection")
		}
		u.Host = net.JoinHostPort(host, port)
		*dsn = u.String()
	}
	db.Port = port
	pool, e := pgxpool.New(ctx, db.AdminDSN)
	if e != nil {
		return db, nil, e
	}
	if len(migrate) > 0 && migrate[0] {
		e = bootstrap.Migrate(ctx, pool, migrations.Files)
	} else {
		e = bootstrap.CheckCurrentStorage(ctx, pool)
	}
	if e != nil {
		pool.Close()
		return db, nil, e
	}
	return db, pool, nil
}
func importAuxiliary(ctx context.Context, dir string) (any, error) {
	db, pool, e := openDatabase(ctx, dir)
	if e != nil {
		return nil, e
	}
	defer pool.Close()
	raw, e := os.ReadFile(filepath.Join(dir, "auxiliary.json"))
	if e != nil {
		return nil, e
	}
	var tables map[string][]map[string]json.RawMessage
	if e = json.Unmarshal(raw, &tables); e != nil {
		return nil, e
	}
	tx, e := pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	counts := map[string]int{}
	for _, table := range []string{"organization.enterprise", "access.principal", "access.credential", "access.policy_guard", "access.session", "access.subject_grant"} {
		rows := tables[table]
		if table == "access.subject_grant" {
			payload, err := json.Marshal(rows)
			if err != nil {
				return nil, err
			}
			if err = bootstrap.RestoreGrantArchive(ctx, tx, payload); err != nil {
				return nil, err
			}
			counts[table] = len(rows)
			continue
		}
		columns, e := tx.Query(ctx, "SELECT attname FROM pg_attribute WHERE attrelid=$1::regclass AND attnum>0 AND NOT attisdropped", table)
		if e != nil {
			return nil, e
		}
		valid := map[string]bool{}
		for columns.Next() {
			var col string
			if e = columns.Scan(&col); e != nil {
				columns.Close()
				return nil, e
			}
			valid[col] = true
		}
		e = columns.Err()
		columns.Close()
		if e != nil {
			return nil, e
		}
		for _, row := range rows {
			for col := range row {
				if !valid[col] {
					return nil, fmt.Errorf("unmapped auxiliary column %s.%s", table, col)
				}
			}
		}
		payload, e := json.Marshal(rows)
		if e != nil {
			return nil, e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO "+table+" SELECT * FROM jsonb_populate_recordset(NULL::"+table+",$1)", payload); e != nil {
			return nil, e
		}
		counts[table] = len(rows)
	}
	if e = tx.QueryRow(ctx, "SELECT instance_id::text FROM organization.enterprise").Scan(&db.Instance); e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	if e = writeJSON(filepath.Join(dir, "candidate-private.json"), db, 0600); e != nil {
		return nil, e
	}
	return counts, nil
}
