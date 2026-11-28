//go:build integration

package bootstrap_test

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

type capacityObject210 struct {
	base, receipts, path string
	number               int
	yearly               bool
	components           []planBody
	saves                []planBody
}

func capacityTerms210(yearly bool, value int) planBody {
	if yearly {
		return planBody{"amount": fmt.Sprint(180000 + value)}
	}
	return planBody{"minimum": "10000", "guidance": "12500", "maximum": fmt.Sprint(15000 + value)}
}

func seedCapacity210(t *testing.T, h *staffing307) (rates, plans []capacityObject210) {
	t.Helper()
	var grades []string
	for i := range 20 {
		grade := h.create("grade", fmt.Sprintf("CAP-%02d", i), "")["id"].(string)
		grades = append(grades, grade)
		for _, freq := range []string{"MONTH", "YEAR"} {
			rate := capacityObject210{base: rates210, receipts: "/api/v1/compensation/grade-rate-receipts/", number: i, yearly: freq == "YEAR"}
			for year := range 6 {
				day := fmt.Sprintf("%d-01-01", 2022+year)
				terms := capacityTerms210(rate.yearly, i*10+year)
				var cmd planBody
				path := rate.path
				if year == 0 {
					form := "RANGE"
					if rate.yearly {
						form = "FIXED"
					}
					terms["form"] = form
					cmd = planBody{"action": "CREATE", "gradeId": grade, "currency": "CNY", "frequency": freq, "unit": "PER_" + freq, "reason": "建立标准", "effectiveDate": day, "changes": terms}
					path = rate.base
				} else {
					cmd = command210("ADD", rate.saves[year-1], day, terms)
				}
				saved := h.b.call("POST", path, cmd, foundation.NewID(), 200)
				rate.path = resultPath210(rate.base, saved)
				rate.saves = append(rate.saves, saved)
			}
			rates = append(rates, rate)
		}
	}
	for i := range 12 {
		plan := capacityObject210{base: plans210, receipts: "/api/v1/compensation/plan-receipts/", number: i, yearly: i%2 == 1}
		freq := "MONTH"
		if plan.yearly {
			freq = "YEAR"
		}
		var links, choices []planBody
		for offset := range 5 {
			grade := (i*2 + offset) % len(grades)
			rate := rates[grade*2+i%2]
			links = append(links, planBody{"gradeId": grades[grade], "rateId": rate.saves[0]["objectId"]})
			choices = append(choices, planBody{"dimension": "grade", "id": grades[grade]})
		}
		firstGrade := i * 2 % len(grades)
		basic := sourceRow210(grades[firstGrade], rates[firstGrade*2+i%2].saves[0], freq)
		basic["rates"] = links
		allow := fixed210("ALLOW", "计划津贴", "0", freq)
		bonus := target210(basic["id"].(string), "12.50", freq)
		plan.components = []planBody{basic, allow, bonus}
		for year := range 6 {
			day := fmt.Sprintf("%d-01-01", 2022+year)
			allow["amount"] = fmt.Sprint(300 + i*10 + year)
			path := plan.path
			var cmd planBody
			if year == 0 {
				cmd = newPlan210(fmt.Sprintf("CAP-%02d-%s", i, freq), planBody{"mode": "SELECTED", "choices": choices}, plan.components...)
				cmd["effectiveDate"] = day
				path = plan.base
			} else {
				cmd = command210("ADD", plan.saves[year-1], day, planBody{"components": plan.components})
			}
			saved := h.b.call("POST", path, cmd, foundation.NewID(), 200)
			plan.path = resultPath210(plan.base, saved)
			plan.saves = append(plan.saves, saved)
		}
		plans = append(plans, plan)
	}
	return
}

func (o capacityObject210) checkRecord(t *testing.T, record planBody, year int) {
	t.Helper()
	if record["effectiveDate"] != fmt.Sprintf("%d-01-01", 2022+year) {
		t.Fatal("wrong effective segment", record["effectiveDate"])
	}
	offset := o.number*10 + year
	if o.base == rates210 {
		field, amount := "maximum", 15000+offset
		if o.yearly {
			field, amount = "amount", 180000+offset
		}
		if record[field] != fmt.Sprintf("%d.00", amount) {
			t.Fatalf("%s: got %v, want %d.00", field, record[field], amount)
		}
		return
	}
	links := component210(record, "BASIC")["rates"].([]any)
	allow, bonus := component210(record, "ALLOW"), component210(record, "BONUS")
	if len(record["components"].([]any)) != 3 || len(links) != 5 || allow["amount"] != fmt.Sprintf("%d.00", 300+offset) {
		t.Fatal("plan composition differs")
	}
	for index, item := range links {
		link := item.(planBody)
		grade := (o.number*2 + index) % 20
		data := link["record"].(planBody)
		if data["effectiveDate"] != record["effectiveDate"] {
			t.Fatal("wrong linked rate period")
		}
		if o.yearly {
			// Values in whole yuan yield exact half-fen cases in eighths of a yuan.
			cents := ((180000+grade*10+year)*100 + 4) / 8
			want := fmt.Sprintf("%d.%02d", cents/100, cents%100)
			target := bonus["targets"].([]any)[index].(planBody)
			if target["gradeId"] != link["gradeId"] || target["amount"] != want {
				t.Fatalf("annual target %v, want %s", target, want)
			}
		} else if data["maximum"] != fmt.Sprintf("%d.00", 15000+grade*10+year) || bonus["targets"] != nil {
			t.Fatal("monthly band or percentage formula differs")
		}
	}
}

// The timed scope covers complete HTTP responses and JSON decoding. Each worker
// owns its session and write targets. Historical and as-of reads use untouched years.
func TestCompensationConfigurationCapacity(t *testing.T) {
	if os.Getenv("CHNHCM_COMPENSATION_CAPACITY") != "1" {
		t.Skip("set CHNHCM_COMPENSATION_CAPACITY=1 to exercise the I3 workload")
	}
	h := staffingSetup307(t)
	rates, plans := seedCapacity210(t, h)
	f := h.f
	if f.scalar("SELECT count(*) FROM compensation.grade_rate_period") != 240 || f.scalar("SELECT count(*) FROM compensation.plan_period") != 72 {
		t.Fatal("baseline segment counts differ")
	}
	const rounds = 12
	type workerResult struct {
		times  map[string][]time.Duration
		writes []planBody
	}
	var sessions [8]*browser
	var results [8]workerResult
	for i := range sessions {
		if i < 4 {
			sessions[i] = f.login("admin", f.password)
		} else {
			sessions[i] = f.login("reader", f.readPassword)
		}
		results[i].times = map[string][]time.Duration{}
	}
	t.Run("sessions", func(t *testing.T) {
		for worker, b := range sessions {
			t.Run(fmt.Sprintf("session-%d", worker), func(t *testing.T) {
				t.Parallel()
				metrics := &results[worker]
				call := func(label, method, path string, body any, key string) planBody {
					t.Helper()
					start := time.Now()
					status, v, err := b.response(method, path, body, key)
					metrics.times[label] = append(metrics.times[label], time.Since(start))
					if err != nil || status != 200 {
						t.Fatalf("%s HTTP %d %v %v", label, status, err, v["message"])
					}
					return v
				}
				for round := range rounds {
					for j, objects := range [][]capacityObject210{rates, plans} {
						kind := []string{"rate", "plan"}[j]
						o := objects[(worker*7+round)%len(objects)]
						set := call(kind+" list", "GET", o.base+"?pageSize=10", nil, "")
						if set["total"] != float64(len(objects)) || len(set["items"].([]any)) != 10 || set["create"] != (worker < 4) {
							t.Fatal("unexpected list size or grant", kind, set["total"], set["create"])
						}
						year := (round + worker) % 5
						detail := call(kind+" date", "GET", o.path+fmt.Sprintf("?businessAt=%d-06-30", 2022+year), nil, "")
						if len(detail["dates"].([]any)) != 6 {
							t.Fatal("lost effective segment")
						}
						o.checkRecord(t, detail["record"].(planBody), year)
						original := o.saves[year]
						items := call(kind+" original", "GET", o.path+"/history?sourceVersion="+original["sourceVersion"].(string), nil, "")["items"].([]any)
						if len(items) != 1 {
							t.Fatal("expected one original")
						}
						entry := items[0].(planBody)
						if entry["sourceVersion"] != original["sourceVersion"] || entry["after"].(planBody)["revision"] != original["revision"] {
							t.Fatal("wrong original identity")
						}
						o.checkRecord(t, entry["after"].(planBody), year)
						if worker >= 4 {
							continue
						}
						target := objects[worker]
						prior := target.saves[5]
						if round > 0 {
							prior = metrics.writes[(round-1)*2+j]
						}
						update := capacityTerms210(target.yearly, 500+round)
						if j == 1 {
							rows := target.components
							update = planBody{"components": []planBody{{"id": rows[0]["id"]}, {"id": rows[1]["id"], "amount": fmt.Sprint(600 + round)}, {"id": rows[2]["id"]}}}
						}
						key := foundation.NewID()
						saved := call(kind+" save", "POST", target.path, command210("CORRECT", prior, "", update), key)
						if saved["outcome"] != "accepted" || saved["recordId"] != prior["recordId"] || saved["revision"] == prior["revision"] || saved["submissionId"] != key {
							t.Fatal("incorrect correction receipt")
						}
						recovered := call(kind+" resolve", "POST", target.receipts+key+"/resolve", nil, "")
						if !reflect.DeepEqual(saved, recovered) {
							t.Fatal("original receipt changed")
						}
						metrics.writes = append(metrics.writes, saved)
					}
				}
			})
		}
	})
	if t.Failed() {
		return
	}
	for _, kind := range []string{"rate", "plan"} {
		for _, op := range []string{"list", "date", "original", "save", "resolve"} {
			name := kind + " " + op
			var times []time.Duration
			for _, w := range results {
				times = append(times, w.times[name]...)
			}
			want := rounds * 8
			if op == "save" || op == "resolve" {
				want = rounds * 4
			}
			if len(times) != want {
				t.Fatalf("%s only %d samples", name, len(times))
			}
			slices.Sort(times)
			ms := func(p int) float64 { return float64(times[(p*len(times)+99)/100-1]) / float64(time.Millisecond) }
			t.Logf("%s requests=%d errors=0 p50ms=%.1f p95ms=%.1f p99ms=%.1f", name, len(times), ms(50), ms(95), ms(99))
		}
	}
	objects := append(slices.Clone(rates), plans...)
	receiptCount := 0
	for worker := range 4 {
		for n, r := range results[worker].writes {
			i := worker
			if n%2 == 1 {
				i += len(rates)
			}
			objects[i].saves = append(objects[i].saves, r)
		}
	}
	type snapshot struct{ history, detail planBody }
	before := make([]snapshot, len(objects))
	for i, o := range objects {
		before[i] = snapshot{h.b.call("GET", o.path+"/history", nil, "", 200), h.b.call("GET", o.path+"?businessAt=2027-06-30", nil, "", 200)}
		record := before[i].detail["record"].(planBody)
		if len(before[i].history["items"].([]any)) != len(o.saves) {
			t.Fatal("wrong number of versions before restart")
		}
		if len(o.saves) > 6 {
			if o.base == plans210 && component210(record, "ALLOW")["amount"] != "611.00" || o.base == rates210 && o.yearly && record["amount"] != "180511.00" || o.base == rates210 && !o.yearly && record["maximum"] != "15511.00" {
				t.Fatal("latest correction missing")
			}
		}
	}
	restartFixtureDatabase(t, f)
	for i, o := range objects {
		current := snapshot{h.b.call("GET", o.path+"/history", nil, "", 200), h.b.call("GET", o.path+"?businessAt=2027-06-30", nil, "", 200)}
		if !reflect.DeepEqual(current, before[i]) {
			t.Fatal("history or effective detail changed across restart", o.path)
		}
		for _, saved := range o.saves {
			got := h.b.call("POST", o.receipts+saved["submissionId"].(string)+"/resolve", nil, "", 200)
			if !reflect.DeepEqual(got, saved) {
				t.Fatal("restart changed original receipt")
			}
			receiptCount++
		}
	}
	t.Logf("restarted database and API: %d objects, %d immutable revisions and original receipts", len(objects), receiptCount)
}
