//go:build integration

package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"runtime"
	"slices"
	"sync"
	"syscall"
	"testing"
	"time"
)

type capacity310 struct {
	capacity309
	ProcessCPUSeconds float64 `json:"processCPUSeconds"`
	GoAllocatedBytes  uint64  `json:"goAllocatedBytes"`
	ElapsedSeconds    float64 `json:"elapsedSeconds"`
}

func TestStaffing310Capacity(t *testing.T) {
	if os.Getenv("CHNHCM_GRADE_G2_CAPACITY") != "1" {
		t.Skip("explicit isolated M03 capacity run")
	}
	started := time.Now()
	h := staffingSetup307(t)
	f := h.f
	grades := []map[string]any{}
	for i := 0; i < 1000; i++ {
		grades = append(grades, h.create("grade", fmt.Sprintf("CAP-G-%04d", i), ""))
	}
	family := h.create("jobfamily", "CAP-FAMILY", "")
	ids := func(start, count int) []string {
		out := []string{}
		for i := start; i < start+count; i++ {
			out = append(out, grades[i]["id"].(string))
		}
		return out
	}
	ladders, jobs := []map[string]any{}, []map[string]any{}
	for i := 0; i < 100; i++ {
		count := 10
		if i >= 80 {
			count = 100
		}
		ladders = append(ladders, create310(h, "gradeladder", fmt.Sprintf("CAP-L-%04d", i), ids(0, count), ""))
	}
	for i := 0; i < 1000; i++ {
		start := 0
		if i >= 500 {
			start = 5
		}
		jobs = append(jobs, create310(h, "job", fmt.Sprintf("CAP-J-%04d", i), ids(start, 5), family["id"].(string)))
	}
	for _, entry := range []struct {
		k       string
		objects []map[string]any
	}{{"grade", grades}, {"gradeladder", ladders}, {"job", jobs}} {
		for i := 0; i < 20; i++ {
			v := entry.objects[i]
			for day := 2; day <= 20; day++ {
				h.add(entry.k, v, "ADD", fmt.Sprintf("2000-01-%02d", day), map[string]any{"name": fmt.Sprintf("容量资料 %d %d", i, day)}, 200)
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
	report := []capacity310{}
	measure := func(name string, sessions int, target float64, operation func(int, int) error) {
		t.Helper()
		groupStarted := time.Now()
		var memoryBefore, memoryAfter runtime.MemStats
		var usageBefore, usageAfter syscall.Rusage
		runtime.ReadMemStats(&memoryBefore)
		must(t, syscall.Getrusage(syscall.RUSAGE_SELF, &usageBefore))
		var mu sync.Mutex
		v := capacity310{capacity309: capacity309{Name: name, Sessions: sessions, Target: target, Samples: []float64{}, Warmup: []float64{}, Failures: []string{}}}
		for w := 0; w < sessions; w++ {
			for j := 0; j < 2; j++ {
				start := time.Now()
				e := operation(w, -j-1)
				v.Warmup = append(v.Warmup, float64(time.Since(start).Nanoseconds())/1e6)
				if e != nil {
					v.Errors++
					v.Failures = append(v.Failures, fmt.Sprintf("prewarm session=%d sample=%d: %v", w, j, e))
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
						v.Failures = append(v.Failures, fmt.Sprintf("formal session=%d sample=%d: %v", w, j, e))
						t.Logf("%s sample error: %v", name, e)
					}
					mu.Unlock()
				}
			}(w)
		}
		wg.Wait()
		must(t, syscall.Getrusage(syscall.RUSAGE_SELF, &usageAfter))
		runtime.ReadMemStats(&memoryAfter)
		v.ElapsedSeconds = time.Since(groupStarted).Seconds()
		v.GoAllocatedBytes = memoryAfter.TotalAlloc - memoryBefore.TotalAlloc
		v.ProcessCPUSeconds = float64(usageAfter.Utime.Sec+usageAfter.Stime.Sec-usageBefore.Utime.Sec-usageBefore.Stime.Sec) + float64(usageAfter.Utime.Usec+usageAfter.Stime.Usec-usageBefore.Utime.Usec-usageBefore.Stime.Usec)/1e6
		ordered := slices.Clone(v.Samples)
		slices.Sort(ordered)
		v.P50 = ordered[int(math.Ceil(.50*float64(len(ordered))))-1]
		v.P99 = ordered[int(math.Ceil(.99*float64(len(ordered))))-1]
		v.P95 = ordered[int(math.Ceil(.95*float64(len(ordered))))-1]
		v.Pass = v.Errors == 0 && (target == 0 || v.P95 <= target)
		report = append(report, v)
		if path := os.Getenv("CHNHCM_GRADE_G2_CAPACITY_REPORT"); path != "" {
			encoded, e := json.MarshalIndent(report, "", "  ")
			must(t, e)
			must(t, os.WriteFile(path, append(encoded, '\n'), 0600))
		}
		t.Logf("GRADE_G2_CAPACITY %s sessions=%d samples=%d errors=%d p95=%.3f target=%.0f pass=%t", name, sessions, len(v.Samples), v.Errors, v.P95, target, v.Pass)
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
	}{{"grade", grades}, {"gradeladder", ladders}, {"job", jobs}} {
		k, objects := entry.k, entry.objects
		for _, n := range []int{1, 20} {
			for w := 0; w < 20; w++ {
				objects[w] = h.refresh(k, objects[w])
			}
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
			if k != "grade" {
				measure(k+"/candidate/admin", n, target, func(w, j int) error {
					v := objects[w]
					r := rec307(v)
					out, e := call(admins[w], "POST", base307(k)+"/grade-reference-context", map[string]any{"action": "CORRECT", "objectId": v["id"], "recordId": r["recordId"], "expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"gradeIds": r["gradeIds"]}, "referencePageSize": 20}, "", 200)
					if e == nil && (out["totalCount"] != float64(1000) || len(out["items"].([]any)) != 20 || out["selectedUnavailable"] != false) {
						e = fmt.Errorf("candidate coverage/selection wrong")
					}
					return e
				})
			}
			// Setup reads are recorded separately from formal single-endpoint timings.
			current := make([]map[string]any, n)
			for w := range n {
				var e error
				current[w], e = call(admins[w], "GET", base307(k)+"/"+objects[w]["id"].(string), nil, "", 200)
				must(t, e)
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
			if k != "grade" {
				measure(k+"/member-save", n, 500, func(w, j int) error {
					v := current[w]
					r := rec307(v)
					count := 5
					if k == "gradeladder" {
						count = 10
					}
					members := ids(0, count-1)
					// Each worker owns a different object. Alternate a real member each call.
					next := 10
					if r["capacityToggle"] == true {
						next = 11
					}
					r["capacityToggle"] = next == 10
					members = append(members, grades[next]["id"].(string))
					out, e := call(admins[w], "POST", base307(k)+"/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"gradeIds": members}}, foundation.NewID(), 200)
					if e != nil {
						return e
					}
					if out["outcome"] != "accepted" {
						return fmt.Errorf("member update not accepted")
					}
					v["headSourceVersion"] = out["sourceVersion"]
					r["revision"] = out["selectedRecordRevision"]
					return nil
				})
			}
			receipts := make([]string, n)
			for w := range n {
				v, e := call(admins[w], "GET", base307(k)+"/"+objects[w]["id"].(string), nil, "", 200)
				must(t, e)
				r := rec307(v)
				key := foundation.NewID()
				_, e = call(admins[w], "POST", base307(k)+"/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": fmt.Sprintf("查回容量 %s %d %d", k, n, w)}}, key, 200)
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
			measure("gradeladder/100-member-detail/"+role.name, n, target, func(w, j int) error {
				out, e := call(role.clients[w], "GET", base307("gradeladder")+"/"+ladders[80+w]["id"].(string), nil, "", 200)
				if e == nil && len(rec307(out)["gradeMembers"].([]any)) != 100 {
					e = fmt.Errorf("100-member detail truncated")
				}
				return e
			})
		}
		current := make([]map[string]any, n)
		for w := range n {
			current[w] = h.refresh("gradeladder", ladders[80+w])
		}
		measure("gradeladder/100-member-candidate/admin", n, target, func(w, j int) error {
			v := current[w]
			r := rec307(v)
			out, e := call(admins[w], "POST", base307("gradeladder")+"/grade-reference-context", map[string]any{"action": "CORRECT", "objectId": v["id"], "recordId": r["recordId"], "expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"gradeIds": r["gradeIds"]}, "referencePageSize": 20}, "", 200)
			if e == nil && (len(out["selected"].([]any)) != 100 || out["selectedUnavailable"] != false || out["totalCount"] != float64(1000)) {
				e = fmt.Errorf("100-member candidate lost selection")
			}
			return e
		})
		measure("gradeladder/100-member-save/admin", n, 500, func(w, j int) error {
			v := current[w]
			r := rec307(v)
			members := ids(0, 100)
			if r["reversed"] != true {
				members[0], members[1] = members[1], members[0]
				r["reversed"] = true
			} else {
				r["reversed"] = false
			}
			out, e := call(admins[w], "POST", base307("gradeladder")+"/"+v["id"].(string)+"/records/"+r["recordId"].(string)+"/correct", map[string]any{"expectedSourceVersion": v["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"gradeIds": members}}, foundation.NewID(), 200)
			if e != nil {
				return e
			}
			if out["outcome"] != "accepted" {
				return fmt.Errorf("100-member reorder not accepted")
			}
			v["headSourceVersion"] = out["sourceVersion"]
			r["revision"] = out["selectedRecordRevision"]
			return nil
		})
	}
	// All 500 Jobs and 100 Ladders retain grade zero. Failed shortening has no
	// documented latency target, but every full HTTP attempt must safely reject.
	g := h.refresh("grade", grades[0])
	r := rec307(g)
	rejected := map[string]map[string]any{}
	measure("grade/500-job-100-ladder-reverse-rejection", 1, 0, func(w, j int) error {
		id := foundation.NewID()
		out, e := call(admins[0], "POST", base307("grade")+"/"+g["id"].(string)+"/deactivate", map[string]any{"expectedSourceVersion": g["headSourceVersion"], "anchorRecordId": r["recordId"], "effectiveDate": "2030-01-01", "operationType": "DEACTIVATE", "operationReason": "GRADE_CLOSURE"}, id, 409)
		if e == nil && (out["code"] != "M03_REFERENCE_CONFLICT" || out["submissionId"] != id || out["outcome"] != "rejected") {
			e = fmt.Errorf("reverse rejection was not a coverage conflict: %v", out)
		}
		if e == nil {
			// The HTTP error envelope repeats these fields from the terminal failure.
			delete(out, "code")
			delete(out, "message")
			delete(out, "field")
			rejected[id] = out
		}
		return e
	})
	if path := os.Getenv("CHNHCM_GRADE_G2_CAPACITY_REPORT"); path != "" {
		b, e := json.MarshalIndent(report, "", "  ")
		must(t, e)
		must(t, os.WriteFile(path, append(b, '\n'), 0600))
	}
	if path := os.Getenv("CHNHCM_GRADE_G2_CAPACITY_REPORT"); path != "" {
		var sizes []byte
		must(t, f.admin.QueryRow(f.ctx, `SELECT jsonb_agg(jsonb_build_object('table',tablename,'totalBytes',pg_total_relation_size(format('%I.%I',schemaname,tablename)::regclass))) FROM pg_tables WHERE schemaname='staffing'`).Scan(&sizes))
		var settings []byte
		must(t, f.admin.QueryRow(f.ctx, `SELECT jsonb_object_agg(name,setting) FROM pg_settings WHERE name IN ('shared_buffers','work_mem','max_connections','fsync','synchronous_commit','full_page_writes')`).Scan(&settings))
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		var usage syscall.Rusage
		must(t, syscall.Getrusage(syscall.RUSAGE_SELF, &usage))
		formal, warmup := 0, 0
		for _, v := range report {
			formal += len(v.Samples)
			warmup += len(v.Warmup)
		}
		b, e := json.MarshalIndent(map[string]any{"harnessElapsedSeconds": time.Since(started).Seconds(), "processUserCPUSeconds": float64(usage.Utime.Sec) + float64(usage.Utime.Usec)/1e6, "processSystemCPUSeconds": float64(usage.Stime.Sec) + float64(usage.Stime.Usec)/1e6, "processMaxRSSKiB": usage.Maxrss, "goTotalAllocatedBytes": memory.TotalAlloc, "apiPoolLimit": f.runtime.Pool.Config().MaxConns, "warmupHTTP": warmup, "totalMeasuredAttempts": formal + warmup, "goVersion": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "cpuCount": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "postgresSettings": json.RawMessage(settings), "staffingTables": json.RawMessage(sizes), "grades": 1000, "ladders": 100, "jobs": 1000, "maxCurrentDecisions": 20, "formalHTTP": len(report) * 100, "groups": len(report), "largestMembership": 100, "reverseJobs": 500, "reverseLadders": 100, "warmupExcluded": true, "detailedDiagnostics": false}, "", "  ")
		must(t, e)
		must(t, os.WriteFile(path+".resources.json", append(b, '\n'), 0600))
	}
	// Match every original rejection to its stored terminal, then resolve all
	// terminals before and after an actual database and API restart.
	for _, k := range []string{"grade", "gradeladder", "job", "jobfamily"} {
		table := map[string]string{"grade": "grade", "gradeladder": "grade_ladder", "job": "job", "jobfamily": "job_family"}[k]
		rows, e := f.admin.Query(f.ctx, "SELECT submission_id::text,result FROM staffing."+table+"_terminal ORDER BY submission_id")
		must(t, e)
		ids := []string{}
		for rows.Next() {
			var id string
			var raw []byte
			must(t, rows.Scan(&id, &raw))
			if original, ok := rejected[id]; k == "grade" && ok {
				var stored map[string]any
				must(t, json.Unmarshal(raw, &stored))
				if !reflect.DeepEqual(stored, original) {
					t.Fatalf("stored rejection differs from original HTTP receipt: %s", id)
				}
				delete(rejected, id)
			}
			ids = append(ids, id)
		}
		must(t, rows.Err())
		rows.Close()
		if k == "grade" && len(rejected) != 0 {
			t.Fatalf("%d original rejections have no terminal", len(rejected))
		}
		restartTimelineReceipts(t, f, h.b, ids, k)
	}
	for _, v := range report {
		if !v.Pass {
			t.Errorf("capacity group failed: %s sessions=%d p95=%.3f errors=%d", v.Name, v.Sessions, v.P95, v.Errors)
		}
	}

}
