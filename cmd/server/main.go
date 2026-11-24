package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	addr := os.Getenv("CHNHCM_HTTP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8380"
	}
	origin := os.Getenv("CHNHCM_ORIGIN")
	insecure := os.Getenv("CHNHCM_INSECURE_LOCAL") == "1"
	if !bootstrap.ValidOrigin(origin, insecure) {
		log.Fatal("a valid CHNHCM_ORIGIN is required")
	}
	if insecure {
		host, _, err := net.SplitHostPort(addr)
		if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
			log.Fatal("insecure local mode requires an explicit loopback listener")
		}
	}
	endProfiles := localProfiles(insecure)
	defer endProfiles()
	runtime, err := bootstrap.Open(ctx, os.Getenv("CHNHCM_DATABASE_URL"), os.Getenv("CHNHCM_INSTANCE_ID"))
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Close()
	api := &bootstrap.API{Runtime: runtime, Origin: origin, InsecureLocal: insecure}
	server := &http.Server{
		Addr: addr, Handler: api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()
	log.Printf("CHNHCM BU service listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
