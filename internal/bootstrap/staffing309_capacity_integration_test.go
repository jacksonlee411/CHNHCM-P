//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"
)

type capacity309 struct {
	Warmup   []float64 `json:"warmupMilliseconds"`
	Failures []string  `json:"semanticErrors"`
	P50      float64   `json:"p50"`
	P99      float64   `json:"p99"`
	Name     string    `json:"name"`
	Sessions int       `json:"sessions"`
	Samples  []float64 `json:"milliseconds"`
	Errors   int       `json:"errors"`
	P95      float64   `json:"p95"`
	Target   float64   `json:"target"`
	Pass     bool      `json:"pass"`
}

func TestStaffing309Capacity(t *testing.T) {
	if os.Getenv("CHNHCM_GRADE_CAPACITY") != "1" {
		t.Skip("explicit isolated M03 capacity run")
	}
	h := staffingSetup307(t)
	f := h.f
	grades := []map[string]any{}
	for i := 0; i < 1000; i++ {
		grades = append(grades, h.create("grade", fmt.Sprintf("CAP-G-%04d", i), ""))
	}
	for i := 0; i < 20; i++ {
		v := grades[i]
		for day := 2; day <= 20; day++ {
			h.add("grade", v, "ADD", fmt.Sprintf("2000-01-%02d", day), map[string]any{"name": fmt.Sprintf("容量职级 %d %d", i, day)}, 200)
			v = h.refresh("grade", v)
		}
		grades[i] = v
	}
	admins, readers := []*browser{}, []*browser{}
	for i := 0; i < 20; i++ {
		admins = append(admins, f.login("admin", f.password))
		readers = append(readers, f.login("reader", f.readPassword))
	}
	report := []capacity309{}
	measure := func(name string, sessions int, target float64, operation func(int, int) error) {
		t.Helper()
		var mu sync.Mutex
		v := capacity309{Name: name, Sessions: sessions, Target: target, Samples: []float64{}, Warmup: []float64{}, Failures: []string{}}
		for w := 0; w < sessions; w++ {
			for j := 0; j < 2; j++ {
				start := time.Now()
				e := operation(w, -j-1)
				v.Warmup = append(v.Warmup, float64(time.Since(start).Nanoseconds())/1e6)
				if e != nil {
					t.Fatalf("prewarm %s: %v", name, e)
				}
			}
		}
		var wg sync.WaitGroup
		for w := 0; w < sessions; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for j := w; j < 100; j += sessions {
					start := time.Now()
					e := operation(w, j)
					elapsed := float64(time.Since(start).Nanoseconds()) / 1e6
					mu.Lock()
					v.Samples = append(v.Samples, elapsed)
					if e != nil {
						v.Errors++
						v.Failures = append(v.Failures, e.Error())
						t.Logf("%s sample error: %v", name, e)
					}
					mu.Unlock()
				}
			}(w)
		}
		wg.Wait()
		ordered := slices.Clone(v.Samples)
		slices.Sort(ordered)
		v.P50 = ordered[int(math.Ceil(.50*float64(len(ordered))))-1]
		v.P99 = ordered[int(math.Ceil(.99*float64(len(ordered))))-1]
		v.P95 = ordered[int(math.Ceil(.95*float64(len(ordered))))-1]
		v.Pass = v.Errors == 0 && (target == 0 || v.P95 <= target)
		report = append(report, v)
		t.Logf("GRADE_CAPACITY %s sessions=%d samples=%d errors=%d p95=%.3f target=%.0f pass=%t", name, sessions, len(v.Samples), v.Errors, v.P95, target, v.Pass)
	}
	call := func(b *browser, method, path string, body any, key string, status int) (map[string]any, error) {
		s, out, e := b.response(method, path, body, key)
		if e == nil && s != status {
			e = fmt.Errorf("status %d wanted %d", s, status)
		}
		return out, e
	}
	for _, entry := range []struct {
		k       string
		objects []map[string]any
	}{{"grade", grades}} {
		k, objects := entry.k, entry.objects
		for _, n := range []int{1, 20} {
			target := 500.0
			if n == 1 {
				target = 300
			}
			for _, role := range []struct {
				name    string
				clients []*browser
			}{{"admin", admins}, {"reader", readers}} {
				clients := role.clients
				measure(k+"/list/"+role.name, n, target, func(w, j int) error {
					out, e := call(clients[w], "GET", base307(k)+"?page=1&pageSize=20", nil, "", 200)
					if e == nil && (out["totalCount"] != float64(len(objects)) || len(out["items"].([]any)) != 20) {
						e = fmt.Errorf("list lost members")
					}
					return e
				})
				measure(k+"/detail/"+role.name, n, target, func(w, j int) error {
					out, e := call(clients[w], "GET", base307(k)+"/"+objects[w]["id"].(string), nil, "", 200)
					if e == nil && (out["id"] != objects[w]["id"] || len(out["timeline"].([]any)) != 20) {
						e = fmt.Errorf("detail identity/decisions differ")
					}
					return e
				})
			}
			// Setup reads are recorded separately from formal single-endpoint timings.
			current := make([]map[string]any, n)
			for w := range n {
				current[w], _ = call(admins[w], "GET", base307(k)+"/"+objects[w]["id"].(string), nil, "", 200)
			}
			measure(k+"/ordinary-save", n, 500, func(w, j int) error {
				v := current[w]
				r := rec307(v)
				input := map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": fmt.Sprintf("保存容量 %s %d %d %d", k, n, w, j)}}
				out, e := call(admins[w], "POST", base307(k)+"/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/correct", input, foundation.NewID(), 200)
				if e != nil {
					return e
				}
				if out["outcome"] != "accepted" {
					return fmt.Errorf("save not accepted")
				}
				v["headSourceVersion"] = out["sourceVersion"]
				r["revision"] = out["selectedRecordRevision"]
				return nil
			})
			receipts := make([]string, n)
			for w := range n {
				v, _ := call(admins[w], "GET", base307(k)+"/"+objects[w]["id"].(string), nil, "", 200)
				r := rec307(v)
				key := foundation.NewID()
				_, e := call(admins[w], "POST", base307(k)+"/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": fmt.Sprintf("查回容量 %s %d %d", k, n, w)}}, key, 200)
				must(t, e)
				receipts[w] = key
			}
			measure(k+"/original-resolve", n, 500, func(w, j int) error {
				out, e := call(admins[w], "POST", base307(k)+"/submissions/"+receipts[w]+"/resolve", map[string]any{}, "", 200)
				if e == nil && (out["outcome"] != "accepted" || out["submissionId"] != receipts[w]) {
					e = fmt.Errorf("receipt differs")
				}
				return e
			})
		}
	}
	if path := os.Getenv("CHNHCM_GRADE_CAPACITY_REPORT"); path != "" {
		b, e := json.MarshalIndent(report, "", "  ")
		must(t, e)
		must(t, os.WriteFile(path, append(b, '\n'), 0600))
	}
	// Restart the real owned PostgreSQL container and API, recover every accepted
	// receipt and exact source, and compare all independent immutable facts.
	for _, k := range []string{"grade"} {
		table := "grade"
		rows, e := f.admin.Query(f.ctx, "SELECT submission_id::text FROM staffing."+table+"_terminal WHERE result->>'outcome'='accepted' ORDER BY submission_id")
		must(t, e)
		ids := []string{}
		for rows.Next() {
			var id string
			must(t, rows.Scan(&id))
			ids = append(ids, id)
		}
		must(t, rows.Err())
		rows.Close()
		restartTimelineReceipts(t, f, h.b, ids, k)
	}
	for _, v := range report {
		if !v.Pass {
			t.Errorf("capacity group failed: %s sessions=%d p95=%.3f errors=%d", v.Name, v.Sessions, v.P95, v.Errors)
		}
	}
	if path := os.Getenv("CHNHCM_GRADE_CAPACITY_REPORT"); path != "" {
		var sizes []byte
		must(t, f.admin.QueryRow(f.ctx, `SELECT jsonb_agg(jsonb_build_object('table',tablename,'totalBytes',pg_total_relation_size(format('%I.%I',schemaname,tablename)::regclass))) FROM pg_tables WHERE schemaname='staffing'`).Scan(&sizes))
		var settings []byte
		must(t, f.admin.QueryRow(f.ctx, `SELECT jsonb_object_agg(name,setting) FROM pg_settings WHERE name IN ('shared_buffers','work_mem','max_connections','fsync','synchronous_commit','full_page_writes')`).Scan(&settings))
		b, e := json.MarshalIndent(map[string]any{"goVersion": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "cpuCount": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "postgresSettings": json.RawMessage(settings), "staffingTables": json.RawMessage(sizes), "grades": 1000, "maxCurrentDecisions": 20, "formalHTTP": 1200, "warmupExcluded": true, "detailedDiagnostics": false}, "", "  ")
		must(t, e)
		must(t, os.WriteFile(path+".resources.json", append(b, '\n'), 0600))
	}
}
