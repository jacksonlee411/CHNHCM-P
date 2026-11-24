// Command capacity runs the explicit T07 workload against a task-owned local profile.
// All business writes use the real HTTP API; the administrator connection only
// observes counters/plans and runs ordinary VACUUM/ANALYZE in this isolated database.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgxpool"
)

type object = map[string]any
type connection struct {
	API     string `json:"apiOrigin"`
	Web     string `json:"webOrigin"`
	Project string `json:"project"`
	Ports   struct {
		Database int `json:"database"`
	} `json:"ports"`
}
type secrets struct {
	Instance, Login, Password string
	DatabasePassword          string `json:"database_password"`
}
type runner struct {
	conn     connection
	db       *pgxpool.Pool
	instance string
	subject  string
	clients  []*client
}
type client struct {
	run  *runner
	http *http.Client
	csrf string
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
func local(args ...string) []byte {
	b, err := exec.Command("python3", append([]string{"scripts/local-bu.py"}, args...)...).Output()
	if err != nil {
		panic("local profile command failed; inspect its non-sensitive runtime log")
	}
	return bytes.TrimSpace(b)
}
func openRunner(sessions int) *runner {
	if !strings.HasPrefix(os.Getenv("CHNHCM_LOCAL_PROFILE"), "m02-t07-") {
		panic("requires an explicit m02-t07-* task profile")
	}
	r := &runner{}
	check(json.Unmarshal(local("connection"), &r.conn))
	p := string(local("credentials-path"))
	abs, err := filepath.Abs(p)
	check(err)
	cwd, err := os.Getwd()
	check(err)
	if strings.HasPrefix(abs, cwd+string(os.PathSeparator)) {
		panic("credentials must be outside the repository")
	}
	info, err := os.Stat(abs)
	check(err)
	if info.Mode().Perm() != 0600 {
		panic("credentials must have mode 0600")
	}
	b, err := os.ReadFile(abs)
	check(err)
	var s secrets
	check(json.Unmarshal(b, &s))
	r.instance = s.Instance
	u := url.URL{Scheme: "postgres", User: url.UserPassword("postgres", s.DatabasePassword), Host: fmt.Sprintf("127.0.0.1:%d", r.conn.Ports.Database), Path: "/chnhcm", RawQuery: "sslmode=disable"}
	r.db, err = pgxpool.New(context.Background(), u.String())
	if err != nil {
		panic("capacity observer configuration invalid")
	}
	var foreign int
	check(r.db.QueryRow(context.Background(), `SELECT count(*) FROM organization.business_unit WHERE instance_id<>$1 OR code NOT LIKE 'T07\_%' ESCAPE '\'`, r.instance).Scan(&foreign))
	if foreign != 0 {
		panic("capacity profile contains non-task business data")
	}
	for i := 0; i < sessions; i++ {
		// Respect the real login limiter, including across independently run phases.
		time.Sleep(3100 * time.Millisecond)
		jar, e := cookiejar.New(nil)
		check(e)
		c := &client{run: r, http: &http.Client{Jar: jar, Timeout: 15 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 4}}}
		v, e := c.call("POST", "/api/v1/session/login", object{"login": s.Login, "password": s.Password}, "", 200)
		check(e)
		c.csrf = v["csrf"].(string)
		r.subject = v["subjectId"].(string)
		r.clients = append(r.clients, c)
	}
	return r
}
func (c *client) call(method, path string, body any, key string, want int) (object, error) {
	var raw []byte
	var err error
	if body != nil {
		raw, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, c.run.conn.API+path, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", c.run.conn.Web)
	req.Header.Set("X-CSRF-Token", c.csrf)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP transport failed on %s %s", method, path)
	}
	defer res.Body.Close()
	raw, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("HTTP body incomplete")
	}
	v := object{}
	if err = json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON response, status %d", res.StatusCode)
	}
	if res.StatusCode != want {
		return v, fmt.Errorf("%s %s status=%d want=%d code=%v", method, path, res.StatusCode, want, v["code"])
	}
	return v, nil
}
func (c *client) detail(id string) (object, error) {
	return c.call("GET", "/api/v1/business-units/"+id, nil, "", 200)
}
func rows(d object) []object {
	a := d["timeline"].([]any)
	r := make([]object, len(a))
	for i, v := range a {
		r[i] = v.(map[string]any)
	}
	return r
}
func (c *client) correct(id string, d, r, changes object) (object, error) {
	return c.call("POST", "/api/v1/business-units/"+id+"/records/"+r["recordId"].(string)+"/correct", object{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": changes}, foundation.NewID(), 200)
}
func parallel(n int, clients []*client, fn func(int, *client) error) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(clients))
	for w, c := range clients {
		wg.Add(1)
		go func(w int, c *client) {
			defer wg.Done()
			for i := w; i < n; i += len(clients) {
				if e := fn(i, c); e != nil {
					errs <- e
					return
				}
			}
		}(w, c)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		return e
	}
	return nil
}
func writeJSON(path string, value any) {
	b, e := json.MarshalIndent(value, "", "  ")
	check(e)
	check(os.MkdirAll(filepath.Dir(path), 0755))
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	check(e)
	_, e = f.Write(append(b, '\n'))
	check(e)
	check(f.Close())
}
func main() {
	phase := flag.String("phase", "inspect", "seed, measure, history, maintain, plans or inspect")
	out := flag.String("out", "", "new evidence JSON path (never overwritten)")
	cache := flag.String("cache", "hot", "cache condition supplied by the controlled runner")
	flag.Parse()
	if *out == "" {
		panic("-out is required")
	}
	if _, e := os.Stat(*out); e == nil {
		panic("evidence already exists")
	}
	sessions := 20
	if *phase == "inspect" || *phase == "maintain" {
		sessions = 0
	}
	if *phase == "history" || *phase == "plans" {
		sessions = 1
	}
	if *cache != "hot" && *cache != "cold" {
		panic("cache must be hot or cold")
	}
	r := openRunner(sessions)
	defer r.db.Close()
	start := time.Now()
	before := r.snapshot()
	var result any
	var err error
	switch *phase {
	case "seed":
		result, err = r.seed()
	case "measure":
		result, err = r.measure(*cache)
	case "history":
		result, err = r.history()
	case "maintain":
		result, err = r.maintain()
	case "plans":
		result, err = r.plans()
	case "inspect":
		result = r.environment()
	default:
		panic("unknown phase")
	}
	report := object{"phase": *phase, "profile": os.Getenv("CHNHCM_LOCAL_PROFILE"), "connection": r.conn, "startedAt": start.UTC(), "durationSeconds": time.Since(start).Seconds(), "before": before, "after": r.snapshot(), "result": result, "passed": err == nil}
	if err != nil {
		report["error"] = err.Error()
	}
	writeJSON(*out, report)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s complete: %s\n", *phase, *out)
}
