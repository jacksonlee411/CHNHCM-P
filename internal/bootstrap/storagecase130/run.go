// Package storagecase130 owns disposable A/B verification only. It is never
// imported by the business server and provides no production upgrade route.
package storagecase130

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	"syscall"
	"time"

	"chnhcm.local/chnhcm/db/migrations"
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/bootstrap/capacitydiag"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

type database struct {
	Container, Volume, AdminDSN, APIDSN, Instance, Role, Port string
}

func resources(role string) (container, volume, task string, err error) {
	switch role {
	case "A":
		return "chnhcm-storage126-product", "chnhcm-storage126-product-data", "storage126-product", nil
	case "B":
		return "chnhcm-storage130-b", "chnhcm-storage130-b-data", "storage130", nil
	default:
		return "", "", "", errors.New("only explicit A/B roles are admitted")
	}
}

func privateDirectory(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil || abs != dir {
		return errors.New("absolute private directory required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(cwd, abs)
	info, statErr := os.Stat(abs)
	if err != nil || !strings.HasPrefix(rel, ".."+string(filepath.Separator)) || statErr != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return errors.New("private directory must be outside the checkout with mode 0700")
	}
	return nil
}

func readPrivate(path string, value any) error {
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		return errors.New("private configuration requires mode 0600")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}

func writePrivate(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0600)
}

func docker(args ...string) ([]byte, error) {
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("task Docker %s failed", args[0])
	}
	return out, nil
}

func secret() string {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(raw[:])
}

func prepare(ctx context.Context, dir, role string) (any, error) {
	container, volume, task, err := resources(role)
	if err != nil {
		return nil, err
	}
	if exec.Command("docker", "inspect", container).Run() == nil || exec.Command("docker", "volume", "inspect", volume).Run() == nil {
		return nil, errors.New("A/B resource already exists; refusing overwrite")
	}
	image, err := os.ReadFile("deploy/local/postgres-image.txt")
	if err != nil {
		return nil, err
	}
	password, apiPassword := secret(), secret()
	if _, err = docker("volume", "create", "--label", "io.chnhcm.task="+task, "--label", "io.chnhcm.parent-task=storage130", "--label", "io.chnhcm.temporary=true", volume); err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		if !keep {
			_, _ = docker("stop", "--time", "10", container)
			_, _ = docker("rm", container)
			_, _ = docker("volume", "rm", volume)
		}
	}()
	// The complete Vanke capacity fixture exceeds Docker's 64 MiB default
	// during concurrent PostgreSQL work. This is a disposable fixture setting;
	// it does not change API budgets or retained development environments.
	if _, err = docker("run", "-d", "--name", container, "--shm-size", "512m", "--label", "io.chnhcm.task="+task, "--label", "io.chnhcm.parent-task=storage130", "--label", "io.chnhcm.temporary=true", "-p", "127.0.0.1::5432", "-e", "POSTGRES_DB=storage126", "-e", "POSTGRES_PASSWORD="+password, "-e", "POSTGRES_INITDB_ARGS=--encoding=UTF8 --locale=C", "-v", volume+":/var/lib/postgresql", strings.TrimSpace(string(image)), "postgres", "-c", "synchronous_commit=on", "-c", "timezone=UTC"); err != nil {
		return nil, err
	}
	portRaw, err := docker("port", container, "5432/tcp")
	if err != nil {
		return nil, err
	}
	host, port, err := net.SplitHostPort(strings.TrimSpace(string(portRaw)))
	if err != nil || host != "127.0.0.1" {
		return nil, errors.New("owned database requires one loopback port")
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword("postgres", password), Host: net.JoinHostPort(host, port), Path: "/storage126", RawQuery: "sslmode=disable&application_name=storage130"}
	db := database{Container: container, Volume: volume, AdminDSN: u.String(), Role: role, Port: port}
	u.User = url.UserPassword("chnhcm_api", apiPassword)
	db.APIDSN = u.String()
	pool, err := pgxpool.New(ctx, db.AdminDSN)
	if err != nil {
		return nil, errors.New("candidate connection could not be initialized")
	}
	defer pool.Close()
	for until := time.Now().Add(30 * time.Second); ; {
		if err = pool.Ping(ctx); err == nil {
			break
		}
		if time.Now().After(until) {
			return nil, errors.New("candidate readiness timed out")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err = bootstrap.Migrate(ctx, pool, migrations.Files); err != nil {
		return nil, err
	}
	if err = bootstrap.ConfigureLocalRoles(ctx, pool, apiPassword); err != nil {
		return nil, err
	}
	if err = writePrivate(filepath.Join(dir, "candidate-private.json"), db); err != nil {
		return nil, err
	}
	keep = true
	return map[string]any{"role": role, "container": container, "port": port, "temporary": true}, nil
}

func open(ctx context.Context, dir string) (database, *pgxpool.Pool, error) {
	var db database
	if err := readPrivate(filepath.Join(dir, "candidate-private.json"), &db); err != nil {
		return db, nil, err
	}
	container, volume, task, err := resources(db.Role)
	if err != nil || db.Container != container || db.Volume != volume {
		return db, nil, errors.New("foreign A/B configuration")
	}
	labels, err := docker("inspect", "--format", "{{index .Config.Labels \"io.chnhcm.task\"}}/{{index .Config.Labels \"io.chnhcm.parent-task\"}}/{{index .Config.Labels \"io.chnhcm.temporary\"}}", container)
	if err != nil || strings.TrimSpace(string(labels)) != task+"/storage130/true" {
		return db, nil, errors.New("A/B task ownership mismatch")
	}
	portRaw, err := docker("port", container, "5432/tcp")
	if err != nil {
		return db, nil, err
	}
	for _, dsn := range []*string{&db.AdminDSN, &db.APIDSN} {
		u, e := url.Parse(*dsn)
		if e != nil || u.Path != "/storage126" || u.Hostname() != "127.0.0.1" {
			return db, nil, errors.New("foreign candidate connection")
		}
		u.Host = strings.TrimSpace(string(portRaw))
		if u.Hostname() != "127.0.0.1" {
			return db, nil, errors.New("candidate database is not loopback")
		}
		*dsn = u.String()
	}
	cfg, err := pgxpool.ParseConfig(db.AdminDSN)
	if err != nil {
		return db, nil, errors.New("invalid candidate configuration")
	}
	cfg.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return db, nil, errors.New("candidate connection failed")
	}
	if err = bootstrap.CheckCurrentStorage(ctx, pool); err != nil {
		pool.Close()
		return db, nil, err
	}
	return db, pool, nil
}

func serve(ctx context.Context, db database, web bool) (any, error) {
	runtime, err := openObserved(ctx, db.APIDSN, db.Instance)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()
	var model string
	if err = runtime.Pool.QueryRow(ctx, `SELECT contract FROM organization.storage_contract`).Scan(&model); err != nil {
		return nil, err
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	origin := "http://" + l.Addr().String()
	api := &bootstrap.API{Runtime: runtime, Origin: origin, InsecureLocal: true}
	handler := api.Handler()
	handler, closeQueryStats, err := observeQuery131(runtime, handler)
	if err != nil {
		l.Close()
		return nil, err
	}
	defer closeQueryStats()
	if web {
		root := "apps/web/dist"
		if _, err = os.Stat(filepath.Join(root, "index.html")); err != nil {
			l.Close()
			return nil, err
		}
		files, apiHandler := http.FileServer(http.Dir(root)), handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				apiHandler.ServeHTTP(w, r)
			} else if strings.HasPrefix(r.URL.Path, "/assets/") {
				files.ServeHTTP(w, r)
			} else {
				http.ServeFile(w, r, filepath.Join(root, "index.html"))
			}
		})
	}
	handler, closeDiagnostics, err := capacitydiag.Open(handler)
	if err != nil {
		l.Close()
		return nil, err
	}
	defer closeDiagnostics()
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second}
	stopped, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-stopped.Done()
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(cleanup)
	}()
	if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"url": origin, "role": db.Role, "container": db.Container, "model": model, "apiConnectionLimit": bootstrap.APIConnectionLimit,
		"departmentPreparationBudgetMS": foundation.DepartmentPreparationBudget(context.Background()).Milliseconds(), "departmentLockAcquisitionBudgetMS": foundation.DepartmentLockAcquisitionBudget.Milliseconds()}); err != nil {
		return nil, err
	}
	err = server.Serve(l)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	return map[string]bool{"stopped": true}, err
}

var extraModes = map[string]func(context.Context, database, *pgxpool.Pool) (any, error){}

func Run(mode, dir, role string) (any, error) {
	if err := privateDirectory(dir); err != nil {
		return nil, err
	}
	ctx := context.Background()
	if mode == "prepare" {
		return prepare(ctx, dir, role)
	}
	if mode == "export" {
		return exportSource(ctx, dir)
	}
	db, pool, err := open(ctx, dir)
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	switch mode {
	case "serve", "serve-web":
		pool.Close()
		return serve(ctx, db, mode == "serve-web")
	case "import":
		return importSnapshot(ctx, dir, db, pool)
	case "initialize-current":
		// A restored historical fixture predates enterprise maintenance. Use
		// the controlled initializer on this already ownership-checked database;
		// preserve the immutable enterprise declaration and account identities.
		var code, name, start string
		if err := pool.QueryRow(ctx, `SELECT code,name,lower(valid_range)::text FROM organization.enterprise WHERE instance_id=$1`, db.Instance).Scan(&code, &name, &start); err != nil {
			return nil, err
		}
		for _, login := range []string{"admin", "reader"} {
			if err := bootstrap.Initialize(ctx, pool, bootstrap.InitInput{Instance: db.Instance, EnterpriseCode: code, EnterpriseName: name, Start: start, Login: login, Password: "admin1234567", ReadOnly: login == "reader"}); err != nil {
				return nil, err
			}
		}
		return map[string]any{"initializedCurrentFixture": true, "role": db.Role}, nil
	case "verify":
		return verifySnapshot(ctx, dir, db, pool, true)
	case "verify-history":
		return verifySnapshot(ctx, dir, db, pool, false)
	case "size":
		return size(ctx, pool)
	}
	if fn, ok := extraModes[mode]; ok {
		return fn(ctx, db, pool)
	}
	return nil, errors.New("unknown storage130 mode")
}
