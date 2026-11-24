//go:build integration

package bootstrap_test

import (
	"bytes"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	db "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func csv307(t *testing.T, b *browser, path string, body any, status int) [][]string {
	t.Helper()
	raw, e := json.Marshal(body)
	must(t, e)
	req, e := http.NewRequest("POST", b.f.server.URL+path, bytes.NewReader(raw))
	must(t, e)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", b.f.server.URL)
	req.Header.Set("X-CSRF-Token", b.csrf)
	res, e := b.client.Do(req)
	must(t, e)
	defer res.Body.Close()
	data, e := io.ReadAll(res.Body)
	must(t, e)
	if res.StatusCode != status {
		t.Fatalf("CSV status %d want %d", res.StatusCode, status)
	}
	if status != 200 {
		return nil
	}
	rows, e := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\xef\xbb\xbf"))).ReadAll()
	must(t, e)
	return rows
}
func TestStaffing307SelectionCSVAndNavigation(t *testing.T) {
	h := staffingSetup307(t)
	f := h.create("jobfamily", "CSV-FAMILY", "")
	for _, k := range []string{"jobfamily", "jobfunction", "job"} {
		t.Run(k, func(t *testing.T) {
			var original []map[string]any
			for i, code := range []string{"CSV307-A", "CSV307-B", "CSV307-C"} {
				v := h.create(k, code, f["id"].(string))
				field := "description"
				if k == "job" {
					field = "responsibilities"
				}
				h.correct(k, v, map[string]any{field: []string{"=1+1", "原文,含逗号\n第二行\"引号\"", "排除成员"}[i]}, nil, 200)
				original = append(original, h.refresh(k, v))
			}
			query := map[string]any{"q": "CSV307-", "page": 1, "pageSize": 20}
			draft := foundation.NewID()
			in := map[string]any{"draftId": draft, "query": query}
			selected := h.b.call("POST", base307(k)+"/selection/start", in, "", 200)
			if selected["count"] != float64(3) {
				t.Fatal("full query not captured")
			}
			h.create(k, "CSV307-NEW", f["id"].(string))
			h.correct(k, original[0], map[string]any{"name": "已经改名仍然在固定集合"}, nil, 200)
			in["expectedRevision"] = selected["revision"]
			in["excludedIds"] = []string{original[2]["id"].(string)}
			selected = h.b.call("POST", base307(k)+"/selection/resolve", in, "", 200)
			if selected["count"] != float64(2) {
				t.Fatal("exclusion wrong")
			}
			in["expectedRevision"] = selected["revision"]
			delete(in, "excludedIds")
			in["sort"] = "-code"
			cells := csv307(t, h.b, base307(k)+"/export", in, 200)
			cols := 6
			if k == "job" {
				cols = 9
			}
			if len(cells) != 3 || len(cells[0]) != cols || cells[1][0] != "CSV307-B" || cells[2][0] != "CSV307-A" {
				t.Fatal("CSV membership/order/columns wrong")
			}
			textcol := 3
			if k == "job" {
				textcol = 4
			}
			if cells[1][textcol] != "原文,含逗号\n第二行\"引号\"" || cells[2][textcol] != "'=1+1" {
				t.Fatal("CSV literal content/formula guard wrong")
			}
			nav := h.b.call("GET", base307(k)+"/"+original[1]["id"].(string)+"/navigation?q=CSV307-&sort=-code", nil, "", 200)
			if nav["totalCount"] != float64(4) || nav["position"] != float64(3) {
				t.Fatal("complete filtered navigation wrong")
			}
			unknown := foundation.NewID()
			closed := h.b.call("POST", base307(k)+"/selection/resolve", map[string]any{"draftId": unknown, "query": query}, "", 200)
			if closed["count"] != float64(0) {
				t.Fatal("unknown resolve captured new members")
			}
			h.b.call("POST", base307(k)+"/selection/start", map[string]any{"draftId": unknown, "query": query}, "", 409)
			h.f.exec(`DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action=$3`, h.f.instance, h.b.subject, k+".export")
			csv307(t, h.b, base307(k)+"/export", in, 403)
		})
	}
}
func TestStaffing307ReferencePermissionAndDayRollover(t *testing.T) {
	h := staffingSetup307(t)
	f := h.create("jobfamily", "DATE-FAMILY", "")
	v := h.create("job", "DATE-JOB", f["id"].(string))
	svc := h.f.runtime.Staffing.(app.Service)
	store := svc.Store.(*db.Store)
	today := time.Now()
	store.Clock = func() time.Time { return today }
	query := map[string]any{"q": "DATE-JOB"}
	in := map[string]any{"draftId": foundation.NewID(), "query": query}
	selection := h.b.call("POST", "/api/v1/jobs/selection/start", in, "", 200)
	in["expectedRevision"] = selection["revision"]
	store.Clock = func() time.Time { return today.Add(24 * time.Hour) }
	h.b.call("POST", "/api/v1/jobs/selection/resolve", in, "", 409)
	store.Clock = nil
	h.f.exec(`DELETE FROM access.subject_grant WHERE instance_id=$1 AND principal_id=$2 AND action='jobfamily.read'`, h.f.instance, h.b.subject)
	h.b.call("GET", "/api/v1/jobs?jobFamilyId="+f["id"].(string), nil, "", 403)
	v = h.refresh("job", v)
	if rec307(v)["jobFamilyId"] != nil || rec307(v)["jobFamilyName"] != "受限" {
		t.Fatal("revoked family projection leaked")
	}
	h.correct("job", v, map[string]any{"requirements": "保持原采用"}, nil, 200)
}

func TestStaffing307IndexedListPagesAndPermissions(t *testing.T) {
	h := staffingSetup307(t)
	families := []string{}
	for i := 0; i < 55; i++ {
		v := h.create("jobfamily", fmt.Sprintf("PAGE-%03d", i), "")
		families = append(families, v["id"].(string))
	}
	for i := 0; i < 55; i++ {
		h.create("job", fmt.Sprintf("PAGE-%03d", i), families[i])
		grade := h.create("grade", fmt.Sprintf("PAGE-%03d", i), "")
		create310(h, "gradeladder", fmt.Sprintf("PAGE-%03d", i), []string{grade["id"].(string)}, "")
	}
	// A joined page cannot rely on an inner subquery retaining row order.
	// Force a different join strategy in this owned DB so planner luck cannot
	// make the ascending and descending parity assertions pass accidentally.
	h.f.exec("ALTER DATABASE " + (pgx.Identifier{h.f.runtime.Pool.Config().ConnConfig.Database}).Sanitize() + " SET enable_nestloop=off")
	h.f.runtime.Pool.Reset()
	for _, k := range []string{"jobfamily", "job", "grade", "gradeladder"} {
		orders := []string{"code", "-code"}
		if k == "grade" || k == "gradeladder" {
			// 310 D02 permits code ascending only for these two objects.
			h.b.call("GET", base307(k)+"?sort=-code", nil, "", 400)
			orders = []string{"code"}
		}
		for _, order := range orders {
			ids := map[string]bool{}
			for page := 1; page <= 3; page++ {
				q := fmt.Sprintf("?page=%d&pageSize=20&sort=%s", page, order)
				a := h.b.call("GET", base307(k)+q, nil, "", 200)
				b := h.b.call("GET", base307(k)+q+"&filterField=code&filterOperator=startsWith&filterValue=PAGE-", nil, "", 200)
				if a["totalCount"] != float64(55) || !reflect.DeepEqual(a["items"], b["items"]) {
					t.Fatalf("indexed page differs from full authorized query: kind=%s order=%s page=%d indexed=%v filtered=%v", k, order, page, a["items"], b["items"])
				}
				for _, v := range a["items"].([]any) {
					id := v.(map[string]any)["id"].(string)
					if ids[id] {
						t.Fatal("page duplicated identity")
					}
					ids[id] = true
				}
			}
			if len(ids) != 55 {
				t.Fatal("page lost identities")
			}
		}
	}
	h.f.exec(`UPDATE access.subject_grant SET read_fields=array_remove(read_fields,'code') WHERE action IN ('jobfamily.read','job.read','grade.read','gradeladder.read')`)
	for _, k := range []string{"jobfamily", "job", "grade", "gradeladder"} {
		ids := []string{}
		for page := 1; page <= 3; page++ {
			v := h.b.call("GET", fmt.Sprintf("%s?page=%d&pageSize=20", base307(k), page), nil, "", 200)
			for _, row := range v["items"].([]any) {
				item := row.(map[string]any)
				if _, ok := item["code"]; ok {
					t.Fatal("hidden code exposed")
				}
				ids = append(ids, item["id"].(string))
			}
		}
		if !slices.IsSorted(ids) || len(ids) != 55 {
			t.Fatal("hidden-code fallback lost stable UUID ordering")
		}
		h.b.call("GET", base307(k)+"?sort=code", nil, "", 403)
	}
}

func TestStaffing307CandidatePeriodsAndExactLabels(t *testing.T) {
	h := staffingSetup307(t)
	a := h.create("jobfamily", "CAND-A", "")
	for n := 1; n < 20; n++ {
		h.add("jobfamily", a, "ADD", fmt.Sprintf("2000-02-%02d", n), map[string]any{"name": "候选末条", "description": strings.Repeat("长", 900)}, 200)
		a = h.refresh("jobfamily", a)
	}
	finite := h.create("jobfamily", "CAND-B", "")
	h.add("jobfamily", finite, "DEACTIVATE", "2030-01-01", nil, 200)
	gap := h.create("jobfamily", "CAND-C", "")
	h.add("jobfamily", gap, "DEACTIVATE", "2021-01-01", nil, 200)
	gap = h.refresh("jobfamily", gap)
	h.add("jobfamily", gap, "REACTIVATE", "2023-01-01", map[string]any{"name": "空档后恢复"}, 200)
	gap = h.refresh("jobfamily", gap)
	future := h.create("jobfamily", "CAND-D", "")
	r := rec307(future)
	h.b.call("POST", base307("jobfamily")+"/"+future["id"].(string)+"/records/"+r["recordId"].(string)+"/redate", map[string]any{"expectedSourceVersion": future["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2030-01-01"}, foundation.NewID(), 200)
	future = h.refresh("jobfamily", future)
	for _, b := range []*browser{h.b, h.f.login("reader", h.f.readPassword)} {
		for _, sample := range []struct {
			date, q string
			ids     []any
		}{
			{"2020-01-01", "", []any{a["id"]}},
			{"2024-01-01", "", []any{a["id"], gap["id"]}},
			{"2035-01-01", "", []any{a["id"], gap["id"], future["id"]}},
			{"2035-01-01", "候选末条", []any{a["id"]}},
		} {
			out := b.call("POST", "/api/v1/jobs/reference-context", map[string]any{"action": "CREATE", "effectiveDate": sample.date, "referenceQ": sample.q}, "", 200)
			got := []any{}
			for _, item := range out["items"].([]any) {
				v := item.(map[string]any)
				got = append(got, v["id"])
				if v["date"] != sample.date {
					t.Fatal("candidate adoption date changed")
				}
				if v["id"] == a["id"] && (v["name"] != "候选末条" || v["sourceVersion"] != a["headSourceVersion"]) {
					t.Fatal("twenty-record candidate label/source differs")
				}
			}
			if !reflect.DeepEqual(got, sample.ids) || out["totalCount"] != float64(len(sample.ids)) {
				t.Fatalf("candidate coverage on %s: got %v want %v", sample.date, got, sample.ids)
			}
		}
	}
}
