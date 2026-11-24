package main

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

func (r *runner) snapshot() object {
	ctx := context.Background()
	var raw []byte
	check(r.db.QueryRow(ctx, `SELECT jsonb_build_object('databaseBytes',pg_database_size(current_database()),'walLSN',pg_current_wal_lsn()::text,'walBytes',pg_wal_lsn_diff(pg_current_wal_lsn(),'0/0'),'businessUnits',(SELECT count(*) FROM organization.business_unit),'sources',(SELECT count(*) FROM organization.source_version),'members',(SELECT count(*) FROM organization.source_member),'revisions',(SELECT count(*) FROM organization.record_revision),'results',(SELECT count(*) FROM organization.command_result),'outbox',(SELECT count(*) FROM organization.outbox),'stats',(SELECT to_jsonb(s) FROM pg_stat_database s WHERE datname=current_database()),'tables',(SELECT jsonb_agg(jsonb_build_object('name',relname,'bytes',pg_total_relation_size(relid),'heapBytes',pg_relation_size(relid))) FROM pg_stat_user_tables WHERE schemaname IN ('organization','access')))`).Scan(&raw))
	v := object{}
	check(json.Unmarshal(raw, &v))
	return v
}
func (r *runner) environment() object {
	var version string
	check(r.db.QueryRow(context.Background(), "SELECT version()").Scan(&version))
	settings := object{}
	rs, e := r.db.Query(context.Background(), `SELECT name,setting,unit FROM pg_settings WHERE name IN ('shared_buffers','work_mem','max_connections','effective_cache_size','synchronous_commit','fsync','full_page_writes','wal_level','max_wal_size','checkpoint_timeout','timezone','block_size','random_page_cost','effective_io_concurrency')`)
	check(e)
	for rs.Next() {
		var name, value string
		var unit *string
		check(rs.Scan(&name, &value, &unit))
		settings[name] = object{"value": value, "unit": unit}
	}
	check(rs.Err())
	rs.Close()
	cpu, _ := os.ReadFile("/proc/cpuinfo")
	memory, _ := os.ReadFile("/proc/meminfo")
	model := ""
	for _, line := range strings.Split(string(cpu), "\n") {
		if strings.HasPrefix(line, "model name") {
			model = line
			break
		}
	}
	disk, _ := exec.Command("df", "-B1", ".").Output()
	kernel, _ := exec.Command("uname", "-sr").Output()
	return object{"disk": strings.TrimSpace(string(disk)), "kernel": strings.TrimSpace(string(kernel)), "go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "logicalCPU": runtime.NumCPU(), "cpu": model, "memory": string(memory), "postgres": version, "settings": settings, "apiPoolMaxConnections": bootstrap.APIConnectionLimit, "clientSessions": 20, "transport": "loopback HTTP real running API"}
}

type sample struct {
	Millis float64 `json:"ms"`
	OK     bool    `json:"ok"`
	Error  string  `json:"error,omitempty"`
}

func summarize(values []sample, target float64) object {
	latency := make([]float64, len(values))
	errors := 0
	for i, v := range values {
		latency[i] = v.Millis
		if !v.OK {
			errors++
		}
	}
	sort.Float64s(latency)
	p := func(q float64) float64 { return latency[int(math.Ceil(q*float64(len(latency))))-1] }
	return object{"count": len(values), "errors": errors, "errorRate": float64(errors) / float64(len(values)), "p50Ms": p(.50), "p95Ms": p(.95), "p99Ms": p(.99), "maxMs": latency[len(latency)-1], "targetP95Ms": target, "targetMet": p(.95) <= target && errors == 0, "samples": values}
}
func (r *runner) measure(cache string) (any, error) {
	type unit struct{ ID, Source string }
	units := []unit{}
	rs, e := r.db.Query(context.Background(), `SELECT b.bu_id,s.source_id FROM organization.business_unit b JOIN organization.source_version s USING(instance_id,bu_id) WHERE b.code LIKE 'T07_CAP_%' AND s.source_no=25 ORDER BY b.code`)
	if e != nil {
		return nil, e
	}
	for rs.Next() {
		var u unit
		if e = rs.Scan(&u.ID, &u.Source); e != nil {
			return nil, e
		}
		units = append(units, u)
	}
	e = rs.Err()
	rs.Close()
	if e != nil {
		return nil, e
	}
	if len(units) != 2000 {
		return nil, fmt.Errorf("full 2000 BU fixture required")
	}
	result := object{"cache": cache, "environment": r.environment(), "concurrency": 20, "timing": "HTTP request through complete response JSON; save preparation GET excluded"}
	// A sampler records actual server lock waits; zero observations are not claimed
	// as proof that sub-sampling-duration waits never occurred.
	stop := make(chan struct{})
	done := make(chan struct{})
	lockSamples := []object{}
	go func() {
		defer close(done)
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				var active, waiting, blocked int
				err := r.db.QueryRow(context.Background(), `SELECT count(*) FILTER(WHERE state='active'),count(*) FILTER(WHERE wait_event_type='Lock'),count(*) FILTER(WHERE cardinality(pg_blocking_pids(pid))>0) FROM pg_stat_activity WHERE application_name='chnhcm'`).Scan(&active, &waiting, &blocked)
				if err == nil {
					lockSamples = append(lockSamples, object{"at": time.Now().UTC(), "active": active, "lockWait": waiting, "blocked": blocked})
				}
			}
		}
	}()
	var problem error
	coldResets := []object{}
	for _, kind := range []string{"list", "detail", "source", "save1", "noChange"} {
		rounds := 10
		if cache == "cold" {
			rounds = 5
		}
		values := make([]sample, rounds*20)
		target := 500.
		if kind == "list" || kind == "detail" {
			target = 300
		}
		// Worker owns a distinct BU for the complete write phase: no intentional
		// object contention. Reads span all 2000 objects, including old sources.
		for round := 0; round < rounds; round++ {
			jobs := make([]func() (object, error), 20)
			expectedSources := make([]string, 20)
			e = parallel(20, r.clients, func(worker int, c *client) error {
				i := round*20 + worker
				u := units[(i*997)%2000]
				expectedSources[worker] = u.Source
				path := "/api/v1/business-units/" + u.ID
				var d object
				var change object
				if kind == "list" {
					path = fmt.Sprintf("/api/v1/business-units?page=%d&pageSize=20&businessAt=2026-11-01", 1+(i%100))
				}
				if kind == "source" {
					path += "?sourceVersion=" + u.Source + "&businessAt=2026-02-01"
				}
				if strings.HasPrefix(kind, "save") || kind == "noChange" {
					u = units[i%20]
					path = "/api/v1/business-units/" + u.ID
					var err error
					d, err = c.detail(u.ID)
					if err != nil {
						return err
					}
					change = object{"name": fmt.Sprintf("容量测量单条轮%d", time.Now().UnixNano())}
					if kind == "noChange" {
						change["name"] = rows(d)[0]["name"]
					}
				}
				if change != nil {
					jobs[worker] = func() (object, error) { return c.correct(u.ID, d, rows(d)[0], change) }
				} else {
					jobs[worker] = func() (object, error) { return c.call("GET", path, nil, "", 200) }
				}
				return nil
			})
			if e != nil {
				break
			}
			if cache == "hot" && (kind == "list" || kind == "detail" || kind == "source") {
				// Warm the exact same twenty read requests before timing them.
				if e = parallel(20, r.clients, func(worker int, _ *client) error { _, err := jobs[worker](); return err }); e != nil {
					break
				}
			}
			if cache == "cold" {
				raw, err := exec.Command("python3", "scripts/t07-cold-cache.py").Output()
				if err != nil {
					e = fmt.Errorf("isolated cold-cache reset failed")
					break
				}
				reset := object{}
				if err = json.Unmarshal(raw, &reset); err != nil {
					e = err
					break
				}
				reset["kind"], reset["round"] = kind, round
				coldResets = append(coldResets, reset)
				r.db.Reset()
			}
			e = parallel(20, r.clients, func(worker int, _ *client) error {
				i := round*20 + worker
				start := time.Now()
				v, err := jobs[worker]()
				values[i] = sample{Millis: float64(time.Since(start)) / float64(time.Millisecond), OK: err == nil}
				if err == nil && kind == "source" && v["resolvedSourceVersion"] != expectedSources[worker] {
					err = fmt.Errorf("exact source mismatch")
					values[i].OK = false
				}
				if err != nil {
					values[i].Error = err.Error()
				}
				return nil
			})
			if e != nil {
				break
			}
		}
		if e != nil {
			problem = e
			break
		}
		summary := summarize(values, target)
		result[kind] = summary
		fmt.Printf("%s %s p95=%.2fms errors=%v targetMet=%v\n", cache, kind, summary["p95Ms"], summary["errors"], summary["targetMet"])
		if summary["targetMet"] != true {
			problem = fmt.Errorf("one or more p95/error targets missed")
		}
	}
	close(stop)
	<-done
	result["lockSamples"] = lockSamples
	result["coldResets"] = coldResets
	return result, problem
}
