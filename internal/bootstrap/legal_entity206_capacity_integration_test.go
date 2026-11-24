//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/legalentity"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

// Opt-in, disposable real database. Fixture construction and semantic checks
// are outside request timing; latency failure is reported without hiding it.
func TestLegalEntity206Capacity(t *testing.T) {
	if os.Getenv("CHNHCM_LEGALENTITY206_CAPACITY") != "1" {
		t.Skip("explicit isolated 1000-background / 20-record scenario")
	}
	h := legalSetup206(t)
	f := h.f
	origin, e := url.Parse(f.server.URL)
	must(t, e)
	c := access.Credentials{Token: h.b.client.Jar.Cookies(origin)[0].Value, CSRF: h.b.csrf, Write: true}
	svc := f.runtime.LegalEntity.(app.Service)
	type sample struct {
		ID, Source, Record, Revision, OriginalSource, Name, Country, Kind string
		Records                                                           int
		Original                                                          [32]byte
	}
	samples := make([]sample, 1020)
	countries := []string{"CN", "HK", "MO", "TW", "US"}
	kinds := []string{"legal_person", "branch", "unincorporated", "other"}
	for i := range samples {
		v := sample{Name: fmt.Sprintf("合成主体%04d", i), Country: h.countries[countries[i%5]], Kind: kinds[i%4], Records: 1}
		if i >= 1000 {
			v.Records = 20
		}
		number := "001Ab"
		registration := "登记证"
		cmd := org.LegalEntityCommand{Action: "CREATE", SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Create: org.LegalEntityCreateInput{Code: fmt.Sprintf("CAP206-%04d", i), Name: v.Name, CountryRegionID: v.Country, SubjectType: v.Kind, RegistrationType: &registration, RegistrationNumber: &number, OperationType: "CREATE", OperationReason: "NEW_LEGAL_ENTITY"}}
		r, e := svc.Maintain(f.ctx, c, cmd)
		must(t, e)
		v.ID, v.Source, v.Record, v.Revision = *r.LegalEntityID, *r.SourceVersion, *r.SelectedRecordID, *r.TargetRevision
		for n := 1; n < v.Records; n++ {
			number = fmt.Sprintf("%05d", n)
			r, e = svc.Maintain(f.ctx, c, org.LegalEntityCommand{Action: "ADD", LegalEntityID: v.ID, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Add: org.LegalEntityAddInput{ExpectedSourceVersion: v.Source, AnchorRecordID: v.Record, EffectiveDate: fmt.Sprintf("%04d-01-01", 1900+n), OperationType: "CHANGE", OperationReason: "REGISTRATION_ADJUSTMENT", Changes: org.LegalEntityChangesInput{"registrationNumber": &number}}})
			must(t, e)
			v.Source, v.Record, v.Revision = *r.SourceVersion, *r.SelectedRecordID, *r.TargetRevision
		}
		v.OriginalSource = v.Source
		must(t, svc.Store.WithinLegalEntity(f.ctx, c, func(u app.Unit) error {
			ops, e := u.History(f.ctx, v.ID, v.Source)
			if e != nil {
				return e
			}
			if len(ops) != v.Records {
				return fmt.Errorf("seed source count %d", i)
			}
			for _, op := range ops {
				for _, rev := range op.Revisions {
					if rev.After != nil && (rev.After.Data.Name != v.Name || rev.After.Data.CountryID != v.Country || rev.After.Data.SubjectType != v.Kind) {
						return fmt.Errorf("independent seed values %d", i)
					}
				}
			}
			raw, e := json.Marshal(ops)
			v.Original = sha256.Sum256(raw)
			return e
		}))
		samples[i] = v
		if i%200 == 0 {
			t.Logf("seed completed ordinal=%d", i)
		}
	}
	if f.scalar("SELECT count(*) FROM organization.legal_entity") != 1020 {
		t.Fatal("fixture count")
	}
	t.Log("fixture ready: 1000 background objects + 20 objects each with 20 records; 1400 original sources; five countries and four subject categories")
	writers := make([]*browser, 20)
	readers := make([]*browser, 20)
	writers[0] = h.b
	readers[0] = h.b
	for i := 1; i < 20; i++ {
		writers[i] = f.login("admin", f.password)
		readers[i] = writers[i]
		if i%2 == 1 {
			readers[i] = f.login("reader", f.readPassword)
		}
	}
	type receipt struct{ ID, Source, Object string }
	receipts := make([]receipt, 200)
	results := []locationTiming204{}
	for _, sessions := range []int{1, 20} {
		for _, group := range []string{"list", "detail", "save", "resolve"} {
			durations := make([]float64, 100)
			failures := make([]string, 100)
			start := make(chan struct{})
			var wg sync.WaitGroup
			offset := 0
			if sessions == 20 {
				offset = 100
			}
			for worker := 0; worker < sessions; worker++ {
				wg.Add(1)
				go func(w int) {
					defer wg.Done()
					<-start
					for i := w; i < 100; i += sessions {
						b := readers[w]
						method, path, key := "GET", "", ""
						var body any
						v := &samples[1000+w]
						switch group {
						case "list":
							path = fmt.Sprintf("%s?countryRegionId=%s&page=%d&pageSize=20", legal206Path, h.countries[countries[i%5]], []int{1, 6, 11}[(i/5)%3])
						case "detail":
							path = legal206Path + "/" + v.ID
						case "save":
							b = writers[w]
							method = "POST"
							key = foundation.NewID()
							path = legal206Path + "/" + v.ID + "/records/" + v.Record + "/correct"
							body = map[string]any{"expectedSourceVersion": v.Source, "expectedRecordRevision": v.Revision, "changes": map[string]any{"registrationNumber": fmt.Sprintf("S%02d-%03d", sessions, i)}}
						case "resolve":
							b = writers[w]
							method = "POST"
							path = legal206Path + "/submissions/" + receipts[offset+i].ID + "/resolve"
						}
						began := time.Now()
						status, out, e := b.response(method, path, body, key)
						durations[i] = float64(time.Since(began)) / float64(time.Millisecond)
						if e != nil || status != 200 {
							failures[i] = fmt.Sprintf("HTTP=%d error=%v code=%v", status, e, out["code"])
							continue
						}
						switch group {
						case "list":
							expected := 20
							if (i/5)%3 == 2 {
								expected = 4
							}
							items, ok := out["items"].([]any)
							if !ok || out["count"] != float64(204) || len(items) != expected {
								failures[i] = "list count/page"
								continue
							}
							for _, raw := range items {
								r := raw.(map[string]any)
								if r["countryRegionId"] != h.countries[countries[i%5]] || r["status"] != "active" {
									failures[i] = "list country/status"
								}
							}
						case "detail":
							r, ok := out["selectedRecord"].(map[string]any)
							timeline, ok2 := out["timeline"].([]any)
							if !ok || !ok2 || len(timeline) != 20 || r["name"] != v.Name || r["subjectType"] != v.Kind || r["countryRegionId"] != v.Country || r["countryRestricted"] != false {
								failures[i] = "20-record detail facts"
							}
						case "save":
							if out["outcome"] != "accepted" || out["legalEntityId"] != v.ID {
								failures[i] = "save result"
								continue
							}
							v.Source = out["sourceVersion"].(string)
							v.Revision = out["targetRevision"].(string)
							receipts[offset+i] = receipt{key, v.Source, v.ID}
						case "resolve":
							r := receipts[offset+i]
							if out["outcome"] != "accepted" || out["legalEntityId"] != r.Object || out["sourceVersion"] != r.Source {
								failures[i] = "original receipt identity"
							}
						}
					}
				}(worker)
			}
			close(start)
			wg.Wait()
			errors := 0
			for i, failure := range failures {
				if failure != "" {
					errors++
					t.Logf("%s/%d/%d %s", group, sessions, i, failure)
				}
			}
			sort.Float64s(durations)
			target := 500.
			if sessions == 1 && (group == "list" || group == "detail") {
				target = 300
			}
			result := locationTiming204{Group: group, Sessions: sessions, Requests: 100, Errors: errors, P50: durations[49], P95: durations[94], P99: durations[98], Target: target, Passed: errors == 0 && durations[94] <= target}
			results = append(results, result)
			raw, _ := json.Marshal(result)
			t.Log(string(raw))
			if errors > 0 {
				t.Errorf("%s/%d correctness failures=%d", group, sessions, errors)
			}
		}
	}
	ids := []string{}
	for _, r := range receipts {
		if r.ID != "" {
			ids = append(ids, r.ID)
		}
	}
	before := legalSnapshot206(t, f)
	restartTimelineReceipts(t, f, h.b, ids, "legalentity")
	if !reflect.DeepEqual(before, legalSnapshot206(t, f)) {
		t.Fatal("restart changed legal facts")
	}
	svc = f.runtime.LegalEntity.(app.Service)
	for i, v := range samples {
		must(t, svc.Store.WithinLegalEntity(f.ctx, c, func(u app.Unit) error {
			ops, e := u.History(f.ctx, v.ID, v.OriginalSource)
			if e != nil {
				return e
			}
			raw, e := json.Marshal(ops)
			if sha256.Sum256(raw) != v.Original {
				return fmt.Errorf("original source changed %d", i)
			}
			return e
		}))
	}
	t.Logf("CAPACITY206_COMPLETE requests=800 receipts=%d all1400OriginalSourcesUnchanged=true restart=true", len(ids))
	if path := os.Getenv("CHNHCM_LEGALENTITY206_RESULT"); path != "" {
		raw, e := json.MarshalIndent(results, "", "  ")
		must(t, e)
		must(t, os.WriteFile(path, append(raw, '\n'), 0600))
	}
}
