//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestT07SharedStrictWriteParserAndOptionalEnglish(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	id := b.call("POST", "/api/v1/business-units", createBody("T07_STRICT", "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	d := detail(b, id)
	path := "/api/v1/business-units/" + id + "/records"
	base := addBody(d, "2026-06-01", map[string]any{"name": "合法资料"})
	cases := []struct {
		name             string
		change           func(map[string]any)
		raw              func([]byte) []byte
		key, contentType string
	}{
		{name: "bad_source_uuid", change: func(v map[string]any) { v["expectedSourceVersion"] = "bad" }},
		{name: "date_space", change: func(v map[string]any) { v["effectiveDate"] = " 2026-06-01" }},
		{name: "date_empty", change: func(v map[string]any) { v["effectiveDate"] = "" }},
		{name: "date_short", change: func(v map[string]any) { v["effectiveDate"] = "2026-6-1" }},
		{name: "date_datetime", change: func(v map[string]any) { v["effectiveDate"] = "2026-06-01T00:00:00+08:00" }},
		{name: "date_number", change: func(v map[string]any) { v["effectiveDate"] = 20260601 }},
		{name: "date_null", change: func(v map[string]any) { v["effectiveDate"] = nil }},
		{name: "date_missing", change: func(v map[string]any) { delete(v, "effectiveDate") }},
		{name: "date_impossible", change: func(v map[string]any) { v["effectiveDate"] = "2026-04-31" }},
		{name: "date_zero", change: func(v map[string]any) { v["effectiveDate"] = "0000-01-01" }},
		{name: "date_ten_thousand", change: func(v map[string]any) { v["effectiveDate"] = "10000-01-01" }},
		{name: "duplicate_key", raw: func(v []byte) []byte {
			return bytes.Replace(v, []byte(`"effectiveDate":"2026-06-01"`), []byte(`"effectiveDate":"2026-06-01","effectiveDate":"2026-07-01"`), 1)
		}},
		{name: "unknown_field", change: func(v map[string]any) { v["state"] = "active" }},
		{name: "unknown_nested", change: func(v map[string]any) { v["changes"] = map[string]any{"name": "合法", "state": "active"} }},
		{name: "missing_submission", key: "missing"},
		{name: "bad_submission", key: "invalid"},
		{name: "plain_text", contentType: "text/plain"},
		{name: "missing_content_type", contentType: "missing"},
		{name: "invalid_utf8", raw: func(v []byte) []byte { return bytes.Replace(v, []byte("合法资料"), []byte{0xff}, 1) }},
		{name: "trailing_document", raw: func(v []byte) []byte { return append(v, []byte(` {"x":1}`)...) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := map[string]any{}
			for k, v := range base {
				in[k] = v
			}
			if tc.change != nil {
				tc.change(in)
			}
			raw, e := json.Marshal(in)
			must(t, e)
			if tc.raw != nil {
				raw = tc.raw(raw)
			}
			req, e := http.NewRequest("POST", f.server.URL+path, bytes.NewReader(raw))
			must(t, e)
			req.Header.Set("Origin", f.server.URL)
			req.Header.Set("X-CSRF-Token", b.csrf)
			key := foundation.NewID()
			if tc.key == "invalid" {
				key = "bad-id"
			}
			if tc.key != "missing" {
				req.Header.Set("Idempotency-Key", key)
			}
			content := tc.contentType
			if content == "" {
				content = "application/json"
			}
			if content != "missing" {
				req.Header.Set("Content-Type", content)
			}
			res, e := b.client.Do(req)
			must(t, e)
			res.Body.Close()
			if res.StatusCode != 400 {
				t.Fatal("strict shape accepted", res.StatusCode)
			}
			if f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history WHERE object_id=$1", id) != 1 || f.recordCount(id) != 1 {
				t.Fatal("invalid input wrote business facts")
			}
		})
	}
	// TL012: absence and explicit emptiness normalize to actual SQL NULL, not
	// an empty string whose UI happens to look the same.
	for i, tc := range []struct {
		name    string
		present bool
		value   any
	}{{"absent", false, nil}, {"null", true, nil}, {"empty", true, ""}, {"Zs", true, " \u3000\u00a0 "}} {
		t.Run("english_"+tc.name, func(t *testing.T) {
			in := createBody(fmt.Sprintf("T07_EN_%d", i), "2026-01-01")
			delete(in, "nameEn")
			if tc.present {
				in["nameEn"] = tc.value
			}
			saved := b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 201)
			revisions := f.buSource(saved["sourceVersion"].(string)).Operation.Revisions
			if len(revisions) != 1 || revisions[0].Record.Data.NameEn != nil {
				t.Fatal("optional English was not SQL NULL")
			}
			v := detail(b, saved["buId"].(string))
			if v["viewFact"].(map[string]any)["nameEn"] != nil || v["viewFact"].(map[string]any)["name"] != "未来业务单元" {
				t.Fatal("missing English lost Chinese fallback", v)
			}
		})
	}
	for _, value := range []string{"\n", "\t", "\u2028", "\u2029", "甲\n乙", "\u200b"} {
		in := createBody("T07_BAD_EN_"+foundation.NewID()[:8], "2026-01-01")
		in["nameEn"] = value
		out := b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 400)
		if out["outcome"] == "accepted" {
			t.Fatal("illegal optional text accepted")
		}
	}
	// A decomposed valid name is accepted and stored in current NFC form.
	in := createBody("T07_EN_NFC", "2026-01-01")
	in["nameEn"] = strings.Repeat("e\u0301", 200)
	saved := b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 201)
	if detail(b, saved["buId"].(string))["viewFact"].(map[string]any)["nameEn"] != strings.Repeat("é", 200) {
		t.Fatal("English NFC mismatch")
	}
}

func TestT07CreateClassificationAndServerOwnedFields(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	for _, kind := range []string{"missing_reason", "unknown_reason", "wrong_reason", "wrong_type", "buId", "enterpriseId", "recordId", "state", "firstRecordId", "recordedAt", "typeLabel", "reasonLabel", "note"} {
		t.Run(kind, func(t *testing.T) {
			in := createBody("T07_CREATE_INPUT", "2026-01-01")
			switch kind {
			case "missing_reason":
				delete(in, "operationReason")
			case "unknown_reason":
				in["operationReason"] = "UNKNOWN_REASON"
			case "wrong_reason":
				in["operationReason"] = "NAME_ADJUSTMENT"
			case "wrong_type":
				in["operationType"] = "DEACTIVATE"
			default:
				in[kind] = "forged"
			}
			b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 400)
			for table, n := range businessSnapshot(t, f) {
				// Safe rejected terminals may exist; no business fact may be created.
				if table != "terminals" && n != "[]" {
					t.Fatal("invalid create wrote business facts", table, n)
				}
			}
		})
	}
	in := createBody("T07_CREATE_NEW", "2026-01-01")
	in["operationReason"] = "NEW_ORGANIZATION"
	saved := b.call("POST", "/api/v1/business-units", in, foundation.NewID(), 201)
	r := recordOn(t, detail(b, saved["buId"].(string)), "2026-01-01")
	if r["operationReason"] != "NEW_ORGANIZATION" || r["reasonLabel"] != "新设组织" || r["operationType"] != "CREATE" {
		t.Fatal("server classification snapshot", r)
	}
}
