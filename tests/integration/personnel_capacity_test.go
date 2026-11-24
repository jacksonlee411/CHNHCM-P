//go:build integration

package integration_test

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func personSample(i int) string    { return fmt.Sprintf("ac000000-0000-4000-8000-%012d", i) }
func personSource(i, j int) string { return fmt.Sprintf("dc000000-0000-4000-%04x-%012d", 32768+j, i) }
func loadPersons(t *testing.T, f *fixture, actor, reader string) {
	tx, e := f.admin.Begin(f.ctx)
	must(t, e)
	defer tx.Rollback(f.ctx)
	_, e = tx.Exec(f.ctx, `create temp table sample on commit drop as select i,('ac000000-0000-4000-8000-'||lpad(i::text,12,'0'))::uuid id, 'P'||lpad(i::text,8,'0') num,
 case when i%50=0 then '同名样本' else '人员'||i end nm from generate_series(1,10000)i;
 create temp table snapshots on commit drop as select s.*,j,('dc000000-0000-4000-'||to_hex(32768+j)||'-'||lpad(i::text,12,'0'))::uuid ver,
 case when j>0 then ('dc000000-0000-4000-'||to_hex(32767+j)||'-'||lpad(i::text,12,'0'))::uuid end prev,
 case when i%100=0 and j=0 then 'LEG-'||lpad(i::text,8,'0') else num end recorded from sample s cross join lateral generate_series(0,case when i%100=0 then 20 else 0 end)j`)
	must(t, e)
	queries := []string{
		`insert into personnel.number select $1,num,id from sample`,
		`insert into personnel.number select $1,recorded,id from snapshots where j=0 and i%100=0`,
		`insert into personnel.person select $1,id,num,ver,'2036-01-01','2037-01-01','local-retention-1','2036-01-01','2037-01-01' from snapshots where j=case when i%100=0 then 20 else 0 end`,
		`insert into personnel.source(instance_id,person_id,version,previous,number,actor,operator_name,recorded_at,policy_version,action) select $1,id,ver,prev,recorded,$2,'合成经办人','2026-01-01'::timestamptz+j*interval '1 second','local-retention-1',case when j=0 then 'register' when j=1 then 'renumber' else 'correct' end from snapshots order by j`,
		`insert into personnel.identity_payload select $1,id,ver,nm from snapshots`,
		`insert into personnel.evidence_payload select $1,id,ver,'form','C-'||i from snapshots`,
		`insert into personnel.contact_payload select $1,id,ver,('bc000000-0000-4000-'||to_hex(32768+k)||'-'||lpad(i::text,12,'0'))::uuid,ver,k,case when k=1 then 'phone' else 'email' end,case when k=1 then '10000'||lpad(i::text,5,'0') else 'demo'||i||'@example.test' end,'2036-01-01','2037-01-01','local-retention-1' from snapshots cross join generate_series(1,2)k where i%5=0`,
		`insert into personnel.counter values($1,10000)`,
		`update access.person_grant set person_ids=(select array_agg(id) from sample where i%4 in (0,1)) where instance_id=$1 and principal_id=$2`,
	}
	for i, sql := range queries {
		args := []any{f.instance}
		if i == 3 {
			args = append(args, actor)
		}
		if i == 8 {
			args = append(args, reader)
		}
		_, e = tx.Exec(f.ctx, sql, args...)
		must(t, e)
	}
	must(t, tx.Commit(f.ctx))
	_, e = f.admin.Exec(f.ctx, "analyze")
	must(t, e)
}
func TestPersonnelCapacity(t *testing.T) {
	if os.Getenv("CHNHCM_PERSONNEL_CAPACITY") != "1" {
		t.Skip("explicit 10,000 person HTTP measurement")
	}
	for _, sessions := range []int{1, 20} {
		t.Run(fmt.Sprint(sessions), func(t *testing.T) {
			start := time.Now()
			f := personnelFixture(t)
			admins, readers := make([]*browser, sessions), make([]*browser, sessions)
			for i := range sessions {
				admins[i] = f.login("admin", f.password)
				readers[i] = f.login("reader", f.readPassword)
			}
			loadPersons(t, f, admins[0].subject, readers[0].subject)
			t.Log("cold startup with fixture", time.Since(start))
			type M = map[string]any
			var httpCount atomic.Int64
			call := func(b *browser, method, path string, body any) (M, error) {
				httpCount.Add(1)
				s, v, e := b.response(method, path, body, "")
				if e == nil && s != 200 {
					e = fmt.Errorf("%s status %d %v", method, s, v["code"])
				}
				return v, e
			}
			list := func(b *browser, term string, page int) (M, error) {
				v, e := call(b, "POST", persons+"/queries", M{"keyword": term})
				if e != nil {
					return nil, e
				}
				return call(b, "GET", fmt.Sprintf("%s?ref=%s&page=%d", persons, v["ref"], page), nil)
			}
			for _, check := range []struct {
				b                            *browser
				term                         string
				page, total, index, expected int
			}{
				{admins[0], "同名样本", 10, 200, 19, 10000}, {readers[0], "", 1, 5000, 0, 1}, {readers[0], "同名样本", 1, 100, 19, 2000}, {readers[0], "同名样本", 2, 100, 0, 2100}, {readers[0], "无此合成人员", 1, 0, 0, 0}} {
				v, e := list(check.b, check.term, check.page)
				must(t, e)
				if v["total"] != float64(check.total) {
					t.Fatal("baseline count", check.total)
				}
				if check.expected > 0 && v["rows"].([]any)[check.index].(M)["id"] != personSample(check.expected) {
					t.Fatal("baseline member", check.expected)
				}
			}
			readers[0].call("GET", persons+"/"+personSample(2), nil, "", 404)
			head, revision := make([]string, 20), make([]string, 20)
			for i := range head {
				head[i] = personSource(100*(i+1), 20)
				revision[i] = head[i]
			}
			// Each write group starts from the same deterministic source graph, including the two original contact values.
			for _, group := range []string{"directory", "restricted", "oldNumber", "detail", "source", "save"} {
				confirmed := make(chan time.Duration, 200)
				work := func(worker, seq int) error {
					b := admins[worker]
					target := (worker + 1) * 100
					switch group {
					case "directory", "restricted", "oldNumber":
						term, page, want := "", 1, 10000
						if group == "restricted" {
							term, page, want, b = "同名样本", 5, 100, readers[worker]
						} else if group == "oldNumber" {
							term, want = "LEG-00000100", 1
						}
						v, e := list(b, term, page)
						if e != nil {
							return e
						}
						if v["total"] != float64(want) {
							return fmt.Errorf("count %v", v["total"])
						}
						rows := v["rows"].([]any)
						first := rows[0].(M)
						if group == "restricted" && (first["id"] != personSample(8100) || rows[19].(M)["id"] != personSample(10000)) {
							return fmt.Errorf("restricted page")
						}
						if group == "oldNumber" && (first["historical"] != true || first["id"] != personSample(100)) {
							return fmt.Errorf("old number")
						}
						return nil
					case "detail", "source":
						path := persons + "/" + personSample(target)
						if group == "source" {
							path += "/sources/" + personSource(target, 0)
						}
						v, e := call(b, "GET", path, nil)
						if e != nil {
							return e
						}
						if len(v["contacts"].([]any)) != 2 {
							return fmt.Errorf("contact count")
						}
						return nil
					default:
						index := worker
						if sessions == 1 {
							index = seq % 20
						}
						target = 100 * (1 + index)
						submission := foundation.NewID()
						path := persons + "/" + personSample(target)
						value := fmt.Sprintf("123%08d", seq+1)
						saveStart := time.Now()
						v, e := call(b, "POST", path+"/corrections", M{"submissionId": submission, "expectedSourceVersion": head[index], "contacts": []M{{"itemId": fmt.Sprintf("bc000000-0000-4000-8001-%012d", target), "expectedRevision": revision[index], "kind": "phone", "value": value}}})
						if e != nil {
							return e
						}
						if v["outcome"] != "accepted" {
							return fmt.Errorf("save %v", v["outcome"])
						}
						receipt, e := call(b, "POST", persons+"/submissions/"+submission+"/resolve", nil)
						if e != nil {
							return e
						}
						if receipt["sourceVersion"] != v["sourceVersion"] {
							return fmt.Errorf("receipt")
						}
						if seq >= 20 {
							confirmed <- time.Since(saveStart)
						}
						// The next command uses the revision actually read from this successful save.
						detail, e := call(b, "GET", path, nil)
						if e != nil {
							return e
						}
						contact := detail["contacts"].([]any)
						if contact[0].(M)["value"] != value || contact[1].(M)["value"] != fmt.Sprintf("demo%d@example.test", target) {
							return fmt.Errorf("changed sibling")
						}
						head[index] = v["sourceVersion"].(string)
						revision[index] = contact[0].(M)["revision"].(string)
						return nil
					}
				}
				for i := 0; i < 20; i++ {
					must(t, work(i%sessions, i))
				}
				httpCount.Store(0)
				times := make(chan time.Duration, 200)
				failures := make(chan error, 200)
				var wg sync.WaitGroup
				ready := make(chan struct{})
				var peak atomic.Int64
				stop, stopped := make(chan struct{}), make(chan struct{})
				go func() {
					defer close(stopped)
					ticker := time.NewTicker(25 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-stop:
							return
						case <-ticker.C:
							var n int64
							if e := f.admin.QueryRow(f.ctx, "select count(*) from pg_stat_activity where application_name='chnhcm' and wait_event_type='Lock'").Scan(&n); e == nil && n > peak.Load() {
								peak.Store(n)
							}
						}
					}
				}()
				begin := time.Now()
				for w := 0; w < sessions; w++ {
					wg.Add(1)
					go func(w int) {
						defer wg.Done()
						<-ready
						for i := w; i < 200; i += sessions {
							at := time.Now()
							e := work(w, i+20)
							if e != nil {
								failures <- e
							} else {
								times <- time.Since(at)
							}
						}
					}(w)
				}
				close(ready)
				wg.Wait()
				elapsed := time.Since(begin)
				close(stop)
				<-stopped
				close(confirmed)
				close(times)
				close(failures)
				samples := []time.Duration{}
				for d := range times {
					samples = append(samples, d)
				}
				sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
				if len(failures) > 0 {
					t.Fatalf("%s failures %d: %v", group, len(failures), <-failures)
				}
				t.Logf("%s sessions=%d N=200 errors=0 HTTP=%d total=%v P50=%v P95=%v P99=%v", group, sessions, httpCount.Load(), elapsed, samples[99], samples[189], samples[197])
				t.Logf("%s sampled peak lock waiters=%d", group, peak.Load())
				if group == "save" {
					latency := []time.Duration{}
					for v := range confirmed {
						latency = append(latency, v)
					}
					sort.Slice(latency, func(i, j int) bool { return latency[i] < latency[j] })
					t.Logf("save through receipt P50=%v P95=%v P99=%v", latency[99], latency[189], latency[197])
				}
				limit := 500 * time.Millisecond
				if sessions == 1 && group != "source" && group != "save" {
					limit = 300 * time.Millisecond
				}
				if samples[189] > limit {
					t.Errorf("%s exceeded %s", group, limit)
				}
			}
			for i := 100; i <= 2000; i += 100 {
				v, e := call(admins[0], "GET", persons+"/"+personSample(i)+"/sources/"+personSource(i, 0), nil)
				must(t, e)
				if v["number"] != fmt.Sprintf("LEG-%08d", i) || v["contacts"].([]any)[0].(M)["value"] != fmt.Sprintf("10000%05d", i) {
					t.Fatal("original changed")
				}
			}
			var count int
			must(t, f.admin.QueryRow(f.ctx, "select count(*) from personnel.number where number like 'LEG-%' and person_id=('ac000000-0000-4000-8000-'||lpad(substring(number from 5),12,'0'))::uuid").Scan(&count))
			if count != 100 {
				t.Fatal("lost old numbers")
			}
		})
	}
}
