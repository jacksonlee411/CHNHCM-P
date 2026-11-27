// Package vankeverify is an offline, task-owned verification composition root.
// It is never registered in the product server.
package vankeverify

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/bootstrap/capacitydiag"
	"chnhcm.local/chnhcm/internal/bootstrap/offlinestorage"
	"chnhcm.local/chnhcm/internal/bootstrap/storagecase130"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type config struct{ AdminDSN, APIDSN, Instance, Profile string }

func Run(mode, dir string) (any, error) {
	ctx := context.Background()
	absolute, e := filepath.Abs(dir)
	if e != nil {
		return nil, e
	}
	cwd, e := os.Getwd()
	if e != nil {
		return nil, e
	}
	rel, e := filepath.Rel(cwd, absolute)
	if e != nil || !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, errors.New("private directory must be outside checkout")
	}
	info, e := os.Stat(absolute)
	if e != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return nil, errors.New("private directory must have mode 0700")
	}
	path := filepath.Join(absolute, "database-private.json")
	info, e = os.Stat(path)
	if e != nil || info.Mode().Perm() != 0600 {
		return nil, errors.New("private config requires mode 0600")
	}
	raw, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var c config
	if e = json.Unmarshal(raw, &c); e != nil {
		return nil, e
	}
	container, database := "chnhcm-vanke127", "vanke127"
	labelFormat := "{{index .Config.Labels \"io.chnhcm.task\"}}/{{index .Config.Labels \"io.chnhcm.temporary\"}}"
	expectedLabel := "vanke127/true"
	if c.Profile != "" {
		if !persistentImportProfile(c.Profile) {
			return nil, errors.New("unapproved persistent verification profile")
		}
		hash := sha256.Sum256([]byte(cwd + ":" + c.Profile))
		project := fmt.Sprintf("chnhcm-bu-%x", hash[:6])
		container, database = project+"-postgres-1", "chnhcm"
		labelFormat = "{{index .Config.Labels \"com.docker.compose.project\"}}/{{index .Config.Labels \"io.chnhcm.profile\"}}/{{index .Config.Labels \"io.chnhcm.temporary\"}}"
		expectedLabel = project + "/" + c.Profile + "/false"
	}
	label, e := exec.Command("docker", "inspect", "--format", labelFormat, container).Output()
	if e != nil || strings.TrimSpace(string(label)) != expectedLabel {
		return nil, errors.New("verification container ownership mismatch")
	}
	port, e := exec.Command("docker", "port", container, "5432/tcp").Output()
	if e != nil {
		return nil, errors.New("task database port unavailable")
	}
	for _, dsn := range []string{c.AdminDSN, c.APIDSN} {
		u, err := url.Parse(dsn)
		if err != nil || u.Host != strings.TrimSpace(string(port)) || u.Hostname() != "127.0.0.1" || u.Path != "/"+database {
			return nil, errors.New("connection does not name the owned task database")
		}
	}
	pool, e := pgxpool.New(ctx, c.AdminDSN)
	if e != nil {
		return nil, errors.New("invalid database configuration")
	}
	defer pool.Close()
	if e = pool.Ping(ctx); e != nil {
		return nil, errors.New("task database unavailable")
	}
	if mode != "initialize" && mode != "migrate" && mode != "read-index" {
		if e = bootstrap.CheckCurrentStorage(ctx, pool); e != nil {
			return nil, e
		}
	}
	switch mode {
	case "read-index":
		if c.Profile != "" {
			return nil, errors.New("read-index rebuild requires a disposable task container")
		}
		if e = bootstrap.Migrate(ctx, pool, migrations.Files); e != nil {
			return nil, e
		}
		return bootstrap.RebuildCurrentReadIndexes(ctx, pool, c.Instance)
	case "migrate":
		err := bootstrap.Migrate(ctx, pool, migrations.Files)
		return map[string]bool{"migrated": err == nil}, err
	case "initialize":
		if e = bootstrap.Migrate(ctx, pool, migrations.Files); e != nil {
			return nil, e
		}
		u, e := url.Parse(c.APIDSN)
		if e != nil {
			return nil, errors.New("invalid API configuration")
		}
		password, _ := u.User.Password()
		if e = bootstrap.ConfigureLocalRoles(ctx, pool, password); e != nil {
			return nil, e
		}
		for _, login := range []string{"admin", "reader"} {
			if e = bootstrap.Initialize(ctx, pool, bootstrap.InitInput{DevelopmentFixture: true, Instance: c.Instance, EnterpriseCode: "VANKE127", EnterpriseName: "组织历史验证企业", Start: "1984-01-01", Login: login, Password: "admin1234567", ReadOnly: login == "reader"}); e != nil {
				return nil, e
			}
		}
		_, e = pool.Exec(ctx, `CREATE SCHEMA verification; CREATE TABLE verification.source_row (phase text NOT NULL, row_no integer NOT NULL, payload jsonb NOT NULL, result jsonb, PRIMARY KEY(phase,row_no)); REVOKE ALL ON SCHEMA verification FROM PUBLIC`)
		return map[string]any{"initialized": e == nil, "enterpriseStart": "1984-01-01"}, e
	case "serve", "serve-import":
		if mode == "serve-import" {
			proof, err := auditSource(ctx, pool, absolute)
			if err != nil {
				return nil, err
			}
			if proof.(map[string]any)["canonicalSHA256"] != "ed917ce6b6d776420a98bc8681d6500c23586c71b474cc554b64fe00322f4026" {
				return nil, errors.New("source archive differs from the approved V127 dataset")
			}
		}
		return serve(ctx, c, pool, mode == "serve-import")
	case "source-audit":
		return auditSource(ctx, pool, absolute)
	case "archive":
		f, e := os.Open(filepath.Join(absolute, "source-rows.jsonl"))
		if e != nil {
			return nil, e
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		n := 0
		tx, e := pool.Begin(ctx)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback(ctx)
		for scanner.Scan() {
			var row struct {
				Phase           string
				Row             int
				Payload, Result json.RawMessage
			}
			if e = json.Unmarshal(scanner.Bytes(), &row); e != nil {
				return nil, e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO verification.source_row VALUES($1,$2,$3,$4) ON CONFLICT(phase,row_no) DO UPDATE SET result=excluded.result WHERE verification.source_row.result IS DISTINCT FROM excluded.result`, row.Phase, row.Row, row.Payload, row.Result); e != nil {
				return nil, e
			}
			n++
		}
		if e = scanner.Err(); e != nil {
			return nil, e
		}
		if e = tx.Commit(ctx); e != nil {
			return nil, e
		}
		return map[string]any{"sourceRows": n}, nil
	case "dump":
		f, e := os.OpenFile(filepath.Join(absolute, "product-history.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if e != nil {
			return nil, e
		}
		defer f.Close()
		w := bufio.NewWriterSize(f, 1<<20)
		encoder := json.NewEncoder(w)
		counts := map[string]int{"BU": 0, "department": 0}
		e = bootstrap.VisitHistory(ctx, pool, c.Instance, func(b *bootstrap.BUHistoryFact, d *bootstrap.DepartmentHistoryFact) error {
			if b != nil {
				counts["BU"]++
				return encoder.Encode(map[string]any{"Kind": "BU", "Fact": b})
			}
			counts["department"]++
			return encoder.Encode(map[string]any{"Kind": "department", "Fact": d})
		})
		if e != nil {
			return nil, e
		}
		if e = w.Flush(); e != nil {
			return nil, e
		}
		return counts, nil
	case "maintain", "compact", "maintain-dry-run":
		if c.Profile != "" {
			return nil, errors.New("maintenance requires a disposable task container")
		}
		started := time.Now()
		maintenance, err := bootstrap.MaintainHistory(ctx, pool, bootstrap.HistoryMaintenanceInput{Instance: c.Instance, OwnedInstances: []string{c.Instance}, DryRun: mode == "maintain-dry-run"})
		if err != nil {
			return nil, err
		}
		if mode == "maintain-dry-run" {
			return maintenance, nil
		}
		if _, err = pool.Exec(ctx, "VACUUM (ANALYZE)"); err != nil {
			return nil, err
		}
		if mode == "compact" {
			if _, err = offlinestorage.CheckRewriteSpace(ctx, pool, container); err != nil {
				return nil, err
			}
			// Offline measurement only, on the ownership-checked verification database.
			// History, audits and all original source cells remain retained.
			rows, err := pool.Query(ctx, `SELECT n.nspname,c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access','verification') ORDER BY n.nspname,c.relname`)
			if err != nil {
				return nil, err
			}
			tables := []pgx.Identifier{}
			for rows.Next() {
				var schema, table string
				if err = rows.Scan(&schema, &table); err != nil {
					rows.Close()
					return nil, err
				}
				tables = append(tables, pgx.Identifier{schema, table})
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return nil, err
			}
			for _, table := range tables {
				if _, err = pool.Exec(ctx, "VACUUM (FULL,ANALYZE) "+table.Sanitize()); err != nil {
					return nil, err
				}
			}
		}
		return map[string]any{"maintenance": maintenance, "vacuumAnalyze": true, "offlineTableRewrite": mode == "compact", "seconds": time.Since(started).Seconds()}, nil
	case "size":
		var data []byte
		e = pool.QueryRow(ctx, `SELECT jsonb_build_object('capturedAt',clock_timestamp(),'databaseBytes',pg_database_size(current_database()),'walBytes',(SELECT coalesce(sum(size),0) FROM pg_ls_waldir()),'walLSN',pg_current_wal_lsn()::text,'sessionRows',(SELECT count(*) FROM access.session),'sessionUpdates',(SELECT coalesce(n_tup_upd,0) FROM pg_stat_user_tables WHERE schemaname='access' AND relname='session'),'counts',jsonb_build_object('BU',(SELECT count(*) FROM organization.business_unit),'department',(SELECT count(*) FROM organization.department),'buSources',(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history),'departmentSources',(SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.department_history),'securityEvents',(SELECT count(*) FROM access.security_event),'authenticationEvents',(SELECT count(*) FROM access.security_event WHERE event_kind='authentication'),'securityChangeEvents',(SELECT count(*) FROM access.security_event WHERE event_kind='security_change'),'standaloneBuTerminals',(SELECT count(*) FROM organization.bu_terminal),'standaloneDepartmentTerminals',(SELECT count(*) FROM organization.department_terminal),'sourceRows',(SELECT count(*) FROM verification.source_row)),'relations',(SELECT jsonb_agg(x ORDER BY total_bytes DESC) FROM (SELECT n.nspname||'.'||c.relname AS relation,pg_relation_size(c.oid) AS main_bytes,pg_table_size(c.oid) AS table_including_toast_bytes,CASE WHEN c.reltoastrelid=0 THEN 0 ELSE pg_total_relation_size(c.reltoastrelid) END AS toast_bytes,pg_indexes_size(c.oid) AS indexes_bytes,pg_total_relation_size(c.oid) AS total_bytes FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname IN ('organization','access','verification')) x),'settings',(SELECT jsonb_object_agg(name,setting) FROM pg_settings WHERE name IN ('fsync','full_page_writes','synchronous_commit','shared_buffers','work_mem','maintenance_work_mem','max_connections','wal_compression','jit')))`).Scan(&data)
		if e != nil {
			return nil, e
		}
		var out any
		e = json.Unmarshal(data, &out)
		return out, e
	}
	return nil, fmt.Errorf("unknown verification mode %q", mode)
}

// Only explicitly retained Vanke environments may use this offline importer.
// Run still verifies checkout, Compose labels, loopback port and database name.
func persistentImportProfile(profile string) bool {
	return profile == "m02-vanke-validation-b" || profile == "m02-current-vanke-c"
}

func serve(ctx context.Context, c config, pool *pgxpool.Pool, importing bool) (any, error) {
	runtime, e := bootstrap.Open(ctx, c.APIDSN, c.Instance)
	if e != nil {
		return nil, e
	}
	if importing {
		runtime, e = importPool(ctx, runtime, c.Profile)
		if e != nil {
			return nil, e
		}
	}
	defer runtime.Close()
	if importing {
		if e = installImportNames(ctx, runtime, pool); e != nil {
			return nil, e
		}
	}
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return nil, e
	}
	origin := "http://" + l.Addr().String()
	api := (&bootstrap.API{Runtime: runtime, Origin: origin, InsecureLocal: true}).Handler()
	api, closeQueryStats, e := storagecase130.ObserveDepartmentQueries(runtime, api)
	if e != nil {
		l.Close()
		return nil, e
	}
	defer closeQueryStats()
	files := http.FileServer(http.Dir("apps/web/dist"))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("VANKE127_STAGES") == "1" {
			var mu sync.Mutex
			stages := map[string]float64{}
			r = r.WithContext(foundation.WithStageObserver(r.Context(), func(name string, elapsed time.Duration) {
				mu.Lock()
				defer mu.Unlock()
				stages[name] += float64(elapsed) / float64(time.Millisecond)
			}))
			defer func() {
				mu.Lock()
				defer mu.Unlock()
				_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"method": r.Method, "stagesMS": stages})
			}()
		}
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/readyz" || r.URL.Path == "/healthz" {
			api.ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/assets/") {
			files.ServeHTTP(w, r)
		} else {
			http.ServeFile(w, r, "apps/web/dist/index.html")
		}
	})
	handler, closeDiagnostics, e := capacitydiag.Open(h)
	if e != nil {
		l.Close()
		return nil, e
	}
	defer closeDiagnostics()
	s := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second}
	stopped, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-stopped.Done()
		x, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(x)
	}()
	if e = json.NewEncoder(os.Stdout).Encode(map[string]any{"url": origin, "apiConnectionLimit": runtime.Pool.Config().MaxConns}); e != nil {
		return nil, e
	}
	e = s.Serve(l)
	if errors.Is(e, http.ErrServerClosed) {
		e = nil
	}
	return map[string]bool{"stopped": true}, e
}
