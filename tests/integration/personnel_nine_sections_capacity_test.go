//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

type profileJSON = map[string]any

type capacityArchive struct {
	target int
	last   int
	head   string
	rows   map[string][]profileJSON
}

var capacitySections = []string{"documents", "addresses", "contacts", "family", "education", "credentials", "bankAccounts"}

func (p capacityArchive) path() string         { return persons + "/" + personSample(p.target) }
func (p capacityArchive) label(seq int) string { return fmt.Sprintf("样本%d轮%d", p.target, seq) }

// Both fixture commands and independent readback assertions use the predetermined sample number.
func (p capacityArchive) seed() profileJSON {
	row := func(fields profileJSON) profileJSON { return profileJSON{"fields": fields} }
	v := profileJSON{"submissionId": foundation.NewID(), "expectedSourceVersion": personSource(p.target, 20), "basic": profileJSON{"formerName": p.label(0)}, "healthSummary": p.label(0)}
	for i := 0; i < 2; i++ {
		label := p.label(0)
		if i == 1 {
			label = "保留"
		}
		vals := map[string]profileJSON{
			"documents":    {"documentKind": "other", "documentNumber": fmt.Sprintf("SYN-%d-%d", p.target, i), "documentName": label},
			"addresses":    {"addressType": "通讯地址", "addressCountry": "合成地区", "street": label},
			"family":       {"relationship": "其他", "familyName": label},
			"education":    {"educationType": "学习经历", "school": label},
			"credentials":  {"credentialKind": "职称", "credentialName": label},
			"bankAccounts": {"bankName": "合成银行", "branchName": label, "accountHolder": "合成户名", "accountNumber": fmt.Sprintf("00%d%d", p.target, i), "currency": "CNY"},
		}
		for section, fields := range vals {
			items, _ := v[section].([]profileJSON)
			v[section] = append(items, row(fields))
		}
	}
	return v
}

var capacityField = map[string]string{"documents": "documentName", "addresses": "street", "family": "familyName", "education": "school", "credentials": "credentialName", "bankAccounts": "branchName"}

func (p capacityArchive) correction(seq int) profileJSON {
	label := p.label(seq)
	v := profileJSON{"submissionId": foundation.NewID(), "expectedSourceVersion": p.head, "healthSummary": label, "basic": profileJSON{"formerName": label}}
	for _, section := range capacitySections {
		prior := p.rows[section][0]
		edit := profileJSON{"itemId": prior["itemId"], "expectedRevision": prior["revision"]}
		if section == "contacts" {
			edit["kind"] = "phone"
			edit["value"] = fmt.Sprintf("133%04d%04d", p.target, seq)
		} else {
			edit["fields"] = profileJSON{capacityField[section]: label}
		}
		v[section] = []profileJSON{edit}
	}
	return v
}

func (p *capacityArchive) verify(v profileJSON, seq int, head any) error {
	label := p.label(seq)
	basic, _ := v["basic"].(profileJSON)
	if v["id"] != personSample(p.target) || v["number"] != fmt.Sprintf("P%08d", p.target) || v["sourceVersion"] != head || basic["formerName"] != label || v["healthSummary"] != label {
		return fmt.Errorf("sample %d basic, health or identity", p.target)
	}
	next := map[string][]profileJSON{}
	for _, section := range capacitySections {
		rows, _ := v[section].([]any)
		if len(rows) != 2 {
			return fmt.Errorf("%s: expected two rows", section)
		}
		for j, entry := range rows {
			row := entry.(profileJSON)
			if p.rows != nil && row["itemId"] != p.rows[section][j]["itemId"] {
				return fmt.Errorf("%s: identity changed", section)
			}
			if j == 1 && p.rows != nil && row["revision"] != p.rows[section][j]["revision"] {
				return fmt.Errorf("%s: sibling revision changed", section)
			}
			if section == "contacts" {
				want := fmt.Sprintf("demo%d@example.test", p.target)
				if j == 0 {
					if seq == 0 {
						want = fmt.Sprintf("10000%05d", p.target)
					} else {
						want = fmt.Sprintf("133%04d%04d", p.target, seq)
					}
				}
				if row["value"] != want {
					return fmt.Errorf("contact readback")
				}
			} else {
				fields, _ := row["fields"].(profileJSON)
				want := label
				if j == 1 {
					want = "保留"
				}
				if fields[capacityField[section]] != want {
					return fmt.Errorf("%s: unexpected business value", section)
				}
				if section == "bankAccounts" && (fields["accountNumber"] != "********" || fields["currency"] != "CNY") {
					return fmt.Errorf("bank number projection")
				}
			}
			next[section] = append(next[section], row)
		}
	}
	p.head = head.(string)
	p.rows = next
	p.last = seq
	return nil
}

func TestPersonnelNineSectionsCapacity(t *testing.T) {
	if os.Getenv("CHNHCM_PERSONNEL_CAPACITY") != "1" {
		t.Skip("explicit 10,000 person HTTP measurement")
	}
	for _, sessions := range []int{1, 20} {
		t.Run(fmt.Sprint(sessions), func(t *testing.T) {
			started := time.Now()
			f := personnelFixture(t)
			readers := f.login("reader", f.readPassword)
			agents := make([]*browser, sessions)
			for i := range agents {
				agents[i] = f.login("admin", f.password)
			}
			loadPersons(t, f, agents[0].subject, readers.subject)
			if f.scalar("select count(*) from personnel.person where instance_id=$1", f.instance) != 10000 {
				t.Fatal("unexpected capacity population")
			}
			people := make([]capacityArchive, 20)
			for i := range people {
				p := &people[i]
				p.target = 100 * (i + 1)
				r := agents[0].call("POST", p.path()+"/corrections", p.seed(), "", 200)
				must(t, p.verify(agents[0].call("GET", p.path(), nil, "", 200), 0, r["sourceVersion"]))
				limited := readers.call("GET", p.path(), nil, "", 200)
				if limited["healthSummary"] != nil || limited["bankAccounts"] != nil {
					t.Fatal("ordinary reader saw sensitive fields")
				}
			}
			readers.call("GET", persons+"/"+personSample(2), nil, "", 404)
			t.Logf("setup=%s customers=1 persons=10000 nine-section samples=20", time.Since(started))
			options := agents[0].call("GET", people[0].path()+"/address-type-candidates", nil, "", 200)["items"].([]any)
			if len(options) != 4 {
				t.Fatal("seeded candidate count")
			}
			for _, chain := range []string{"detail", "correct-resolve-read", "address-type-candidates", "address-type-correct-resolve-read"} {
				call := func(b *browser, m, path string, input any) (profileJSON, error) {
					status, v, e := b.response(m, path, input, "")
					if e == nil && status != 200 {
						e = fmt.Errorf("%s HTTP %d %v", m, status, v["code"])
					}
					return v, e
				}
				work := func(worker, i int) error {
					b := agents[worker]
					p := &people[i%20]
					seq := p.last
					head := p.head
					var selected profileJSON
					if chain == "address-type-candidates" {
						v, err := call(b, "GET", p.path()+"/address-type-candidates?itemId="+p.rows["addresses"][0]["itemId"].(string), nil)
						if err != nil {
							return err
						}
						opts, ok := v["items"].([]any)
						if !ok || len(opts) != 4 || opts[0].(profileJSON)["code"] != "RESIDENTIAL" {
							return fmt.Errorf("unexpected candidate scope or order")
						}
						return nil
					}
					if chain == "correct-resolve-read" || chain == "address-type-correct-resolve-read" {
						seq = i + 1
						cmd := p.correction(seq)
						if chain == "address-type-correct-resolve-read" {
							selected = options[(i/20)%2].(profileJSON)
							cmd["addresses"].([]profileJSON)[0]["addressTypeChoice"] = profileJSON{"id": selected["id"], "revision": selected["revision"]}
						}
						result, e := call(b, "POST", p.path()+"/corrections", cmd)
						if e != nil {
							return e
						}
						receipt, e := call(b, "POST", persons+"/submissions/"+cmd["submissionId"].(string)+"/resolve", nil)
						if e != nil {
							return e
						}
						if result["outcome"] != "accepted" || receipt["sourceVersion"] != result["sourceVersion"] || receipt["id"] != personSample(p.target) {
							return fmt.Errorf("unconfirmed correction")
						}
						head = result["sourceVersion"].(string)
					}
					detail, e := call(b, "GET", p.path(), nil)
					if e != nil {
						return e
					}
					if selected != nil {
						address := detail["addresses"].([]any)[0].(profileJSON)
						ref := address["addressTypeAdoption"].(profileJSON)
						if ref["id"] != selected["id"] || ref["revision"] != selected["revision"] || address["fields"].(profileJSON)["addressType"] != selected["name"] {
							return fmt.Errorf("wrong type adoption")
						}
					}
					return p.verify(detail, seq, head)
				}
				for i := range 20 {
					must(t, work(i%sessions, i))
				}
				var wg sync.WaitGroup
				gate := make(chan struct{})
				times := make(chan time.Duration, 200)
				failures := make(chan error, 200)
				for w := range sessions {
					wg.Add(1)
					go func() {
						defer wg.Done()
						<-gate
						for i := w; i < 200; i += sessions {
							at := time.Now()
							if e := work(w, i+20); e != nil {
								failures <- e
							} else {
								times <- time.Since(at)
							}
						}
					}()
				}
				stop := make(chan struct{})
				peak := make(chan int, 1)
				go func() {
					tick := time.NewTicker(25 * time.Millisecond)
					defer tick.Stop()
					maxWaiters := 0
					for {
						select {
						case <-stop:
							peak <- maxWaiters
							return
						case <-tick.C:
							var waiting int
							err := f.admin.QueryRow(f.ctx, "select count(*) from pg_stat_activity where application_name='chnhcm' and wait_event_type='Lock'").Scan(&waiting)
							if err == nil {
								maxWaiters = max(maxWaiters, waiting)
							}
						}
					}
				}()
				start := time.Now()
				close(gate)
				wg.Wait()
				elapsed := time.Since(start)
				close(stop)
				waiters := <-peak
				close(times)
				close(failures)
				if n := len(failures); n > 0 {
					t.Fatalf("%s errors=%d/200 first=%v", chain, n, <-failures)
				}
				var samples []time.Duration
				for x := range times {
					samples = append(samples, x)
				}
				sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
				http := 200
				if chain == "correct-resolve-read" || chain == "address-type-correct-resolve-read" {
					http *= 3
				}
				t.Logf("%s sessions=%d N=200 HTTP=%d errors=0 total=%s P50=%s P95=%s P99=%s peakLockWaiters=%d", chain, sessions, http, elapsed, samples[99], samples[189], samples[197], waiters)
				limit := 500 * time.Millisecond
				if sessions == 20 {
					limit = 1500 * time.Millisecond // Temporary acceptance target for the M01 address-type pilot (317).
				} else if chain == "detail" {
					limit = 300 * time.Millisecond
				}
				if samples[189] > limit {
					t.Errorf("%s P95 exceeded %s", chain, limit)
				}
			}
			if f.scalar("select count(*) from personnel.number where number like 'LEG-%'") != 100 {
				t.Fatal("historical number count")
			}
			for _, p := range people {
				source := agents[0].call("GET", p.path()+"/sources/"+personSource(p.target, 0), nil, "", 200)
				if source["number"] != fmt.Sprintf("LEG-%08d", p.target) || source["contacts"].([]any)[0].(profileJSON)["value"] != fmt.Sprintf("10000%05d", p.target) {
					t.Fatal("original source changed")
				}
			}
		})
	}
}
