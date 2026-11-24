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

type capacity307 struct {
	Name     string    `json:"name"`
	Sessions int       `json:"sessions"`
	Samples  []float64 `json:"milliseconds"`
	Errors   int       `json:"errors"`
	P95      float64   `json:"p95"`
	Target   float64   `json:"target"`
	Pass     bool      `json:"pass"`
}

func TestStaffing307Capacity(t *testing.T) {
	if os.Getenv("CHNHCM_M03_CAPACITY") != "1" {
		t.Skip("explicit isolated M03 capacity run")
	}
	h := staffingSetup307(t)
	f := h.f
	families := []map[string]any{}
	for i := 0; i < 100; i++ {
		families = append(families, h.create("jobfamily", fmt.Sprintf("CAP-F-%03d", i), ""))
	}
	jobs := []map[string]any{}
	for i := 0; i < 1000; i++ {
		parent := 0
		if i >= 500 {
			parent = 1 + (i-500)%99
		}
		jobs = append(jobs, h.create("job", fmt.Sprintf("CAP-J-%04d", i), families[parent]["id"].(string)))
	}
	// Twenty independently editable objects of each kind, twenty current decisions each.
	for _, entry := range []struct {
		k       string
		objects []map[string]any
	}{{"jobfamily", families}, {"job", jobs}} {
		for i := 0; i < 20; i++ {
			v := entry.objects[i]
			for day := 1; day < 20; day++ {
				date, act, changes := fmt.Sprintf("2000-01-%02d", day+1), "ADD", map[string]any{"name": fmt.Sprintf("容量资料 %d %d", i, day)}
				if entry.k == "job" && day == 5 {
					act = "DEACTIVATE"
					changes = nil
				}
				if entry.k == "job" && day == 6 {
					act = "REACTIVATE"
				}
				if entry.k == "job" && day == 19 {
					date = "2035-01-01"
					changes["jobFamilyId"] = families[1]["id"]
				}
				h.add(entry.k, v, act, date, changes, 200)
				v = h.refresh(entry.k, v)
			}
			entry.objects[i] = v
		}
	}
	admins, readers := []*browser{}, []*browser{}
	for i := 0; i < 20; i++ {
		admins = append(admins, f.login("admin", f.password))
		readers = append(readers, f.login("reader", f.readPassword))
	}
	report := []capacity307{}
	measure := func(name string, sessions int, target float64, operation func(int, int) error) {
		t.Helper()
		for w := 0; w < sessions; w++ {
			for j := 0; j < 2; j++ {
				if e := operation(w, -j-1); e != nil {
					t.Fatalf("prewarm %s: %v", name, e)
				}
			}
		}
		var mu sync.Mutex
		v := capacity307{Name: name, Sessions: sessions, Target: target, Samples: []float64{}}
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
						t.Logf("%s sample error: %v", name, e)
					}
					mu.Unlock()
				}
			}(w)
		}
		wg.Wait()
		ordered := slices.Clone(v.Samples)
		slices.Sort(ordered)
		v.P95 = ordered[int(math.Ceil(.95*float64(len(ordered))))-1]
		v.Pass = v.Errors == 0 && (target == 0 || v.P95 <= target)
		report = append(report, v)
		t.Logf("M03_CAPACITY %s sessions=%d samples=%d errors=%d p95=%.3f target=%.0f pass=%t", name, sessions, len(v.Samples), v.Errors, v.P95, target, v.Pass)
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
	}{{"jobfamily", families}, {"job", jobs}} {
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
	for _, n := range []int{1, 20} {
		target := 500.0
		if n == 1 {
			target = 300
		}
		for _, role := range []struct {
			name    string
			clients []*browser
		}{{"admin", admins}, {"reader", readers}} {
			measure("job/candidate/"+role.name, n, target, func(w, j int) error {
				out, e := call(role.clients[w], "POST", "/api/v1/jobs/reference-context", map[string]any{"action": "CREATE", "effectiveDate": "2000-01-01"}, "", 200)
				if e == nil && (out["totalCount"] != float64(100) || len(out["items"].([]any)) != 20) {
					e = fmt.Errorf("candidate population differs")
				}
				return e
			})
		}
	}
	// High-fanout qualification rejection has no latency acceptance threshold.
	f0 := h.refresh("jobfamily", families[0])
	firstID := f0["timeline"].([]any)[0].(map[string]any)["recordId"].(string)
	legal := h.detail("jobfamily", f0["id"].(string), firstID)
	legalNumber := 0
	measure("jobfamily/500-reference-legal-period-observation", 1, 0, func(w, j int) error {
		date := "1999-01-01"
		if legalNumber%2 == 1 {
			date = "2000-01-01"
		}
		legalNumber++
		r := rec307(legal)
		out, e := call(admins[w], "POST", "/api/v1/job-families/"+legal["id"].(string)+"/records/"+firstID+"/redate", map[string]any{"expectedSourceVersion": legal["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": date}, foundation.NewID(), 200)
		if e != nil {
			return e
		}
		if out["outcome"] != "accepted" {
			return fmt.Errorf("legal period change not accepted")
		}
		legal["headSourceVersion"] = out["sourceVersion"]
		r["revision"] = out["selectedRecordRevision"]
		return nil
	})
	f0 = h.refresh("jobfamily", families[0])
	for _, n := range []int{1, 20} {
		measure("jobfamily/500-reference-rejection-observation", n, 0, func(w, j int) error {
			out, e := call(admins[w], "POST", "/api/v1/job-families/"+f0["id"].(string)+"/deactivate", map[string]any{"expectedSourceVersion": f0["headSourceVersion"], "anchorRecordId": rec307(f0)["recordId"], "effectiveDate": "2030-01-01", "operationType": "DEACTIVATE", "operationReason": "FAMILY_CLOSURE"}, foundation.NewID(), 409)
			if e == nil && out["code"] != "M03_REFERENCE_CONFLICT" {
				e = fmt.Errorf("wrong qualification refusal")
			}
			return e
		})
	}
	if path := os.Getenv("CHNHCM_M03_CAPACITY_REPORT"); path != "" {
		b, e := json.MarshalIndent(report, "", "  ")
		must(t, e)
		must(t, os.WriteFile(path, append(b, '\n'), 0600))
	}
	// Restart the real owned PostgreSQL container and API, recover every accepted
	// receipt and exact source, and compare all independent immutable facts.
	for _, k := range []string{"jobfamily", "job"} {
		table := "job"
		if k == "jobfamily" {
			table = "job_family"
		}
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
	if path := os.Getenv("CHNHCM_M03_CAPACITY_REPORT"); path != "" {
		var sizes []byte
		must(t, f.admin.QueryRow(f.ctx, `SELECT jsonb_agg(jsonb_build_object('table',tablename,'totalBytes',pg_total_relation_size(format('%I.%I',schemaname,tablename)::regclass))) FROM pg_tables WHERE schemaname='staffing'`).Scan(&sizes))
		var settings []byte
		must(t, f.admin.QueryRow(f.ctx, `SELECT jsonb_object_agg(name,setting) FROM pg_settings WHERE name IN ('shared_buffers','work_mem','max_connections','fsync','synchronous_commit','full_page_writes')`).Scan(&settings))
		b, e := json.MarshalIndent(map[string]any{"goVersion": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "cpuCount": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "postgresSettings": json.RawMessage(settings), "staffingTables": json.RawMessage(sizes), "families": 100, "jobs": 1000, "maxCurrentDecisions": 20, "formalHTTP": 3100, "warmupExcluded": true, "detailedDiagnostics": false}, "", "  ")
		must(t, e)
		must(t, os.WriteFile(path+".resources.json", append(b, '\n'), 0600))
	}
}
