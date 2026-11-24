//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
)

type locationSample204 struct {
	ID, Source, Record, Revision, Country, Area, Code, Name, Address, Date string
	Sources                                                                int
	Original                                                               [32]byte
}
type locationTiming204 struct {
	Group                      string
	Sessions, Requests, Errors int
	P50, P95, P99              float64
	Target                     float64
	Passed                     bool
}

func TestLocation204Capacity(t *testing.T) {
	if os.Getenv("CHNHCM_LOCATION204_CAPACITY") != "1" {
		t.Skip("explicit isolated 10001-location capacity scenario")
	}
	resumed := os.Getenv("CHNHCM_LOCATION204_RESUME_CONTAINER") != ""
	var f *fixture
	if resumed {
		f = resumeLocationCapacity204(t)
	} else {
		f = freshWithEnterpriseStart(t, "1900-01-01")
		must(t, f.runtime.InitializeCountryCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
		must(t, f.runtime.InitializeGeoAreaCatalog(f.ctx, access.LoginInput{Login: "admin", Password: f.password}))
	}
	svc := f.runtime.Location.(app.Service)
	countries, areas := []string{}, []string{}
	for i, c := range []string{"CN", "HK", "MO", "TW"} {
		countries = append(countries, geoCountry203(t, f, c))
		a := ""
		if []string{"110108", "CW", "", "63000010"}[i] != "" {
			a = geoID203(t, f, []string{"110108", "CW", "", "63000010"}[i])
		}
		areas = append(areas, a)
	}
	samples := make([]locationSample204, 10001)
	seedCreds := make([]access.Credentials, 4)
	for i := range seedCreds {
		seedCreds[i] = geoCredentials203(t, f, "admin", f.password)
	}
	failures := make(chan error, 4)
	var wg sync.WaitGroup
	seed := func(ctx context.Context, c access.Credentials, i, region, records, sources int) (locationSample204, error) {
		v := locationSample204{Country: countries[region], Area: areas[region], Code: fmt.Sprintf("CAP204-%05d", i), Name: fmt.Sprintf("容量地点%05d", i), Address: "初始地址", Date: "1900-01-01", Sources: sources}
		var area *string
		if v.Area != "" {
			area = &v.Area
		}
		if resumed {
			if e := f.admin.QueryRow(ctx, `SELECT location_id::text FROM organization.location WHERE instance_id=$1 AND code=$2`, f.instance, v.Code).Scan(&v.ID); e != nil {
				return v, e
			}
			e := svc.Store.WithinLocation(ctx, c, func(u app.Unit) error {
				state, e := u.Load(ctx, v.ID, "")
				if e != nil {
					return e
				}
				timeline := state.Timeline
				if len(timeline.Records) != records || timeline.SourceNo != int64(sources) || timeline.Code != v.Code {
					return fmt.Errorf("resumed input shape %d", i)
				}
				r := timeline.Records[len(timeline.Records)-1]
				v.Source, v.Record, v.Revision, v.Date, v.Address = timeline.Source, r.ID, r.Revision, r.Date, r.Data.AddressLine
				return nil
			})
			if e != nil {
				return v, e
			}
		} else {
			cmd := org.LocationCommand{Action: "CREATE", SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Create: org.LocationCreateInput{Code: v.Code, Name: v.Name, CountryRegionID: v.Country, GeoAreaID: area, AddressLine: v.Address, OperationType: "CREATE", OperationReason: "NEW_LOCATION"}}
			r, e := svc.Maintain(ctx, c, cmd)
			if e != nil {
				return v, e
			}
			v.ID, v.Source, v.Record, v.Revision = *r.LocationID, *r.SourceVersion, *r.SelectedRecordID, *r.TargetRevision
			for n := 2; n <= sources; n++ {
				cmd = org.LocationCommand{SubmissionID: foundation.NewID(), Digest: make([]byte, 32), LocationID: v.ID}
				address := fmt.Sprintf("地址%04d", n)
				if n <= records {
					cmd.Action = "ADD"
					v.Date = fmt.Sprintf("%04d-01-01", 1900+n)
					cmd.Add = org.LocationAddInput{ExpectedSourceVersion: v.Source, AnchorRecordID: v.Record, EffectiveDate: v.Date, OperationType: "CHANGE", OperationReason: "ADDRESS_ADJUSTMENT", Changes: org.LocationChangesInput{"addressLine": &address}}
				} else {
					cmd.Action = "CORRECT"
					cmd.RecordID = v.Record
					cmd.Correct = org.LocationCorrectInput{ExpectedSourceVersion: v.Source, ExpectedRecordRevision: v.Revision, Changes: org.LocationChangesInput{"addressLine": &address}}
				}
				r, e = svc.Maintain(ctx, c, cmd)
				if e != nil {
					return v, e
				}
				v.Source, v.Record, v.Revision, v.Address = *r.SourceVersion, *r.SelectedRecordID, *r.TargetRevision, address
			}
		}
		e := svc.Store.WithinLocation(ctx, c, func(u app.Unit) error {
			ops, e := u.History(ctx, v.ID, v.Source)
			if e != nil {
				return e
			}
			for _, op := range ops {
				for _, rev := range op.Revisions {
					if rev.After != nil {
						// ADD also closes its predecessor; that revision retains its own
						// address. Only the final independent record is corrected later.
						number := 1
						if rev.After.Date != "1900-01-01" {
							year, parseErr := strconv.Atoi(rev.After.Date[:4])
							if parseErr != nil {
								return parseErr
							}
							number = year - 1900
						}
						if number == records && op.Number > int64(number) {
							number = int(op.Number)
						}
						expected := "初始地址"
						if number > 1 {
							expected = fmt.Sprintf("地址%04d", number)
						}
						if rev.After.Data.Name != v.Name || rev.After.Data.CountryID != v.Country || rev.After.Data.AddressLine != expected {
							return fmt.Errorf("independent seed values %d/%d", i, op.Number)
						}
					}
				}
			}
			if len(ops) != sources {
				return fmt.Errorf("seed source count %d", len(ops))
			}
			raw, e := json.Marshal(ops)
			v.Original = sha256.Sum256(raw)
			return e
		})
		return v, e
	}
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := w; i < 10000; i += 4 {
				records := 1 + i%3
				sources := max(records, 1+i%5)
				v, e := seed(f.ctx, seedCreds[w], i, i%4, records, sources)
				if e != nil {
					failures <- fmt.Errorf("seed %d: %w", i, e)
					return
				}
				samples[i] = v
				if i%1000 == w {
					t.Logf("seed worker=%d completed ordinal=%d", w, i)
				}
			}
		}(worker)
	}
	wg.Wait()
	close(failures)
	for e := range failures {
		must(t, e)
	}
	v, e := seed(f.ctx, seedCreds[0], 10000, 0, 1, 1000)
	must(t, e)
	samples[10000] = v
	if f.scalar("SELECT count(*) FROM organization.location") != 10001 {
		t.Fatal("background count")
	}
	t.Log("fixture: 10000 background, four regions 2500 each; records=1+i%3 sources=max(records,1+i%5); extra CN object 1000 sources; all sources use authorized commands")
	// Resolve exact source identities independently from source number index.
	var exact []app.Operation
	must(t, svc.Store.WithinLocation(f.ctx, seedCreds[0], func(u app.Unit) error { var e error; exact, e = u.History(f.ctx, v.ID, v.Source); return e }))
	sourceAt := []string{exact[999].ID, exact[500].ID, exact[0].ID}
	if gate := os.Getenv("CHNHCM_LOCATION204_START"); gate != "" {
		t.Log("fixture ready; waiting for isolated measurement gate")
		for {
			if _, e := os.Stat(gate); e == nil {
				break
			}
			time.Sleep(time.Second)
		}
	}
	clients := make([]*browser, 20)
	writers := make([]*browser, 20)
	// Refresh fixture-only credentials after the optional isolation wait. Each
	// formal group has 20 distinct HTTP sessions; the 10 administrator read
	// clients reuse writer sessions across sequential groups. This prepares only
	// 20 admin logins, respecting the unchanged 20/minute account limit.
	seedCreds[0] = geoCredentials203(t, f, "admin", f.password)
	for i := range clients {
		writers[i] = f.login("admin", f.password)
		clients[i] = writers[i]
		if i%2 == 1 {
			clients[i] = f.login("reader", f.readPassword)
		}
	}
	type receipt struct{ ID, Source, Object string }
	receipts := make([]receipt, 400)
	results := []locationTiming204{}
	for _, sessions := range []int{1, 20} {
		for _, group := range []string{"list", "detail", "save", "exact-source"} {
			duration := make([]float64, 200)
			errs := make([]string, 200)
			start := make(chan struct{})
			var workers sync.WaitGroup
			for worker := 0; worker < sessions; worker++ {
				workers.Add(1)
				go func(w int) {
					defer workers.Done()
					<-start
					for i := w; i < 200; i += sessions {
						b := clients[w]
						method, path, body, key := "GET", "", any(nil), ""
						sample := samples[800+i]
						expectedName := sample.Name
						switch group {
						case "list":
							pages := []int{1, 63, 125}
							if i%4 == 0 {
								pages = []int{1, 64, 126}
							}
							page := pages[(i/4)%3]
							path = fmt.Sprintf("/api/v1/locations?countryRegionId=%s&page=%d&pageSize=20", countries[i%4], page)
						case "detail":
							path = "/api/v1/locations/" + sample.ID
						case "exact-source":
							slot := 0
							if i >= 67 {
								slot = 1
							}
							if i >= 133 {
								slot = 2
							}
							path = "/api/v1/locations/" + v.ID + "?sourceVersion=" + sourceAt[slot]
							expectedName = v.Name
						case "save":
							index := i
							if sessions == 20 {
								index += 200
							}
							sample = samples[index]
							b = writers[w]
							method = "POST"
							path = "/api/v1/locations/" + sample.ID + "/records/" + sample.Record + "/correct"
							key = foundation.NewID()
							body = map[string]any{"expectedSourceVersion": sample.Source, "expectedRecordRevision": sample.Revision, "changes": map[string]any{"name": fmt.Sprintf("保存-%d-%d", sessions, i)}}
						}
						began := time.Now()
						status, out, e := b.response(method, path, body, key)
						duration[i] = float64(time.Since(began)) / float64(time.Millisecond)
						if e != nil || status != 200 {
							errs[i] = fmt.Sprintf("request %d status=%d error=%v", i, status, e)
							continue
						}
						switch group {
						case "list":
							expected := float64(2500)
							if i%4 == 0 {
								expected++
							}
							if out["count"] != expected || len(out["items"].([]any)) != func() int {
								if i%4 == 0 && (i/4)%3 == 2 {
									return 1
								}
								return 20
							}() {
								errs[i] = "list count/page"
							}
						case "detail", "exact-source":
							r, ok := out["selectedRecord"].(map[string]any)
							if !ok || r["name"] != expectedName || r["addressRestricted"] == true {
								errs[i] = "detail semantic mismatch"
							}
							if group == "exact-source" {
								address := "初始地址"
								if i >= 67 {
									address = "地址0500"
								}
								if i >= 133 {
									address = "地址1000"
								}
								if r["addressLine"] != address {
									errs[i] = "exact source address mismatch"
								}
							}
						case "save":
							if out["outcome"] != "accepted" || out["locationId"] != sample.ID {
								errs[i] = "save semantic mismatch"
							} else {
								index := i
								if sessions == 20 {
									index += 200
								}
								receipts[index] = receipt{key, out["sourceVersion"].(string), sample.ID}
							}
						}
					}
				}(worker)
			}
			close(start)
			workers.Wait()
			failures := 0
			for i, e := range errs {
				if e != "" {
					failures++
					t.Logf("%s/%d index=%d %s", group, sessions, i, e)
				}
			}
			sort.Float64s(duration)
			target := 500.
			if sessions == 1 && (group == "list" || group == "detail") {
				target = 300
			}
			r := locationTiming204{Group: group, Sessions: sessions, Requests: 200, Errors: failures, P50: duration[99], P95: duration[189], P99: duration[197], Target: target, Passed: failures == 0 && duration[189] <= target}
			results = append(results, r)
			raw, _ := json.Marshal(r)
			t.Log(string(raw))
			if failures != 0 {
				t.Errorf("%s/%d correctness failures=%d", group, sessions, failures)
			}
		}
	}
	// A captured 10000-member selection does not add the separate long-history object.
	start, e := svc.StartSelection(f.ctx, seedCreds[0], org.LocationSelectionStartInput{Query: org.LocationSelectionQuery{FilterField: "code", FilterOperator: "contains", FilterValue: "CAP204-0"}, Selection: org.Selection{Mode: "allMatching", ExcludedIDs: func() *[]string { v := []string{}; return &v }()}, Page: 1, PageSize: 100})
	must(t, e)
	if start.Summary.SelectedCount != 10000 {
		t.Fatal("10000 selection", start.Summary.SelectedCount)
	}
	ids := []string{}
	for _, r := range receipts {
		if r.ID != "" {
			ids = append(ids, r.ID)
		}
	}
	restartTimelineReceipts(t, f, writers[0], ids, "location")
	svc = f.runtime.Location.(app.Service)

	for i, r := range receipts {
		if r.ID == "" {
			continue
		}
		out := writers[i%20].call("POST", "/api/v1/locations/submissions/"+r.ID+"/resolve", nil, "", 200)
		if out["outcome"] != "accepted" || out["sourceVersion"] != r.Source || out["locationId"] != r.Object {
			t.Fatal("receipt changed", i)
		}
	}
	for i, v := range samples {
		must(t, svc.Store.WithinLocation(f.ctx, seedCreds[0], func(u app.Unit) error {
			ops, e := u.History(f.ctx, v.ID, v.Source)
			if e != nil {
				return e
			}
			raw, e := json.Marshal(ops)
			if sha256.Sum256(raw) != v.Original {
				return fmt.Errorf("original history mismatch %d", i)
			}
			return e
		}))
		if i%1000 == 0 {
			t.Logf("restart complete original history ordinal=%d", i)
		}
	}
	geoAllSources203(t, f)
	t.Logf("all 400 receipts, all original histories for 10001 locations and full geographic sources preserved after PostgreSQL restart; exact sample sources=%v", sourceAt)
	if path := os.Getenv("CHNHCM_LOCATION204_REPORT"); path != "" {
		raw, e := json.MarshalIndent(struct {
			Groups  []locationTiming204
			Samples []locationSample204
		}{results, samples}, "", "  ")
		must(t, e)
		must(t, os.WriteFile(path, raw, 0600))
	}
}
