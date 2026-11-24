// Offline server for the single explicitly disposable synthetic profile.
// Ordinary product handlers and connection limits remain unchanged.
package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
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
	"chnhcm.local/chnhcm/internal/bootstrap/storagecase130"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

func run() error {
	mode := flag.String("mode", "serve", "serve, dump or migrate the owned temporary profile")
	output := flag.String("output", "", "new private dump file")
	flag.Parse()
	if *mode != "serve" && *mode != "dump" && *mode != "migrate" {
		return errors.New("unsupported verification mode")
	}
	const profile = "tree300-synthetic-d"
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(root + ":" + profile))
	key := fmt.Sprintf("%x", digest[:6])
	project := "chnhcm-bu-" + key
	container := project + "-postgres-1"
	labels, err := exec.Command("docker", "inspect", "--format", "{{index .Config.Labels \"io.chnhcm.profile\"}}/{{index .Config.Labels \"io.chnhcm.temporary\"}}/{{index .Config.Labels \"com.docker.compose.project\"}}", container).Output()
	if err != nil || strings.TrimSpace(string(labels)) != profile+"/true/"+project {
		return errors.New("synthetic profile ownership mismatch")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".local", "state", "chnhcm", key, "runtime.json")
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		return errors.New("private runtime configuration required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var config struct {
		Instance      string `json:"instance"`
		Password      string `json:"api_password"`
		AdminPassword string `json:"database_password"`
	}
	if json.Unmarshal(raw, &config) != nil {
		return errors.New("invalid private configuration")
	}
	port, err := exec.Command("docker", "port", container, "5432/tcp").Output()
	address := strings.TrimSpace(string(port))
	host, _, splitErr := net.SplitHostPort(address)
	if err != nil || splitErr != nil || host != "127.0.0.1" {
		return errors.New("owned database must bind loopback")
	}
	dsn := url.URL{Scheme: "postgres", User: url.UserPassword("chnhcm_api", config.Password), Host: address, Path: "/chnhcm", RawQuery: "sslmode=disable"}
	ctx := context.Background()
	if *mode == "migrate" {
		dsn.User = url.UserPassword("postgres", config.AdminPassword)
		pool, err := pgxpool.New(ctx, dsn.String())
		if err != nil {
			return errors.New("cannot open owned migration database")
		}
		defer pool.Close()
		return bootstrap.Migrate(ctx, pool, migrations.Files)
	}
	if *mode == "dump" {
		target, err := filepath.Abs(*output)
		if err != nil || *output == "" {
			return errors.New("private output required")
		}
		relative, err := filepath.Rel(root, target)
		info, statErr := os.Stat(filepath.Dir(target))
		if err != nil || !strings.HasPrefix(relative, ".."+string(filepath.Separator)) || statErr != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
			return errors.New("dump requires an external 0700 directory")
		}
		dsn.User = url.UserPassword("postgres", config.AdminPassword)
		pool, err := pgxpool.New(ctx, dsn.String())
		if err != nil {
			return errors.New("cannot open owned history reader")
		}
		defer pool.Close()
		if err = bootstrap.CheckCurrentStorage(ctx, pool); err != nil {
			return err
		}
		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		defer file.Close()
		writer := bufio.NewWriterSize(file, 1<<20)
		encoder := json.NewEncoder(writer)
		counts := map[string]int{"BU": 0, "department": 0}
		err = bootstrap.VisitHistory(ctx, pool, config.Instance, func(b *bootstrap.BUHistoryFact, d *bootstrap.DepartmentHistoryFact) error {
			if b != nil {
				counts["BU"]++
				return encoder.Encode(map[string]any{"Kind": "BU", "Fact": b})
			}
			counts["department"]++
			return encoder.Encode(map[string]any{"Kind": "department", "Fact": d})
		})
		if err != nil {
			return err
		}
		if err = writer.Flush(); err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(counts)
	}
	runtime, err := bootstrap.Open(ctx, dsn.String(), config.Instance)
	if err != nil {
		return errors.New("cannot open owned current-model runtime")
	}
	defer runtime.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	origin := "http://" + listener.Addr().String()
	handler := (&bootstrap.API{Runtime: runtime, Origin: origin, InsecureLocal: true}).Handler()
	handler, closeQueries, err := storagecase130.ObserveDepartmentQueries(runtime, handler)
	if err != nil {
		return err
	}
	defer closeQueries()
	api := handler
	files := http.FileServer(http.Dir("apps/web/dist"))
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			api.ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/assets/") {
			files.ServeHTTP(w, r)
		} else {
			http.ServeFile(w, r, "apps/web/dist/index.html")
		}
	})
	handler, closeDiagnostics, err := capacitydiag.Open(handler)
	if err != nil {
		return err
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
	if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"url": origin, "container": container, "apiConnectionLimit": runtime.Pool.Config().MaxConns,
		"departmentPreparationBudgetMS": foundation.DepartmentPreparationBudget(context.Background()).Milliseconds(), "departmentLockAcquisitionBudgetMS": foundation.DepartmentLockAcquisitionBudget.Milliseconds()}); err != nil {
		return err
	}
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
