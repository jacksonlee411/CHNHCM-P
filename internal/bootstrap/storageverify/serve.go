package storageverify

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/bootstrap/capacitydiag"
)

func serveDatabase(ctx context.Context, dir string, web bool) (any, error) {
	db, admin, e := openDatabase(ctx, dir)
	if e != nil {
		return nil, e
	}
	admin.Close()
	runtime, e := bootstrap.Open(ctx, db.APIDSN, db.Instance)
	if e != nil {
		return nil, e
	}
	defer runtime.Close()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return nil, e
	}
	origin := "http://" + l.Addr().String()
	api := &bootstrap.API{Runtime: runtime, Origin: origin, InsecureLocal: true}
	handler := api.Handler()
	if web {
		root := "apps/web/dist"
		if _, e := os.Stat(filepath.Join(root, "index.html")); e != nil {
			l.Close()
			return nil, e
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
	handler, closeDiagnostics, e := capacitydiag.Open(handler)
	if e != nil {
		l.Close()
		return nil, e
	}
	defer closeDiagnostics()
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second}
	stopped, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-stopped.Done()
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(c)
	}()
	if e = json.NewEncoder(os.Stdout).Encode(map[string]any{"url": origin, "container": db.Container, "apiConnectionLimit": bootstrap.APIConnectionLimit}); e != nil {
		return nil, e
	}
	e = server.Serve(l)
	if errors.Is(e, http.ErrServerClosed) {
		e = nil
	}
	return map[string]any{"stopped": true}, e
}
