package bootstrap

import (
	"bytes"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/exportfile"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type brokenSelectionBody struct{}

func (brokenSelectionBody) Read([]byte) (int, error) { return 0, errors.New("transport interrupted") }

func TestFeedback301SelectionBodyFailureIsNotSizeLimit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   io.Reader
		status int
		code   string
	}{
		{"interrupted", brokenSelectionBody{}, 400, "M02_INVALID_SELECTION"},
		{"oversized", strings.NewReader(strings.Repeat("x", 65537)), 413, "M02_SELECTION_BATCH_LIMIT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/api/v1/departments/selection/start", tc.body)
			response := httptest.NewRecorder()
			var out map[string]any
			if decodeIdentityDraft(response, request, &out, true) {
				t.Fatal("invalid body accepted")
			}
			var result struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if response.Code != tc.status || result.Code != tc.code {
				t.Fatalf("status/code = %d/%s", response.Code, result.Code)
			}
		})
	}
}

func TestFeedback301BUSelectionTransportPreservesLargeExplicitSet(t *testing.T) {
	ids := make([]string, 2021)
	for i := range ids {
		ids[i] = foundation.NewID()
	}
	raw, err := json.Marshal(map[string]any{"query": map[string]any{"q": "QS", "status": "all"}, "expectedSelectionVersion": strings.Repeat("a", 64), "selection": map[string]any{"mode": "explicit", "ids": ids}})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) <= 65536 {
		t.Fatal("fixture must cross old transport limit")
	}
	req := httptest.NewRequest("POST", "/api/v1/business-units/selection/resolve", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	var result org.SelectionRequest
	if !decodeSelection(res, req, &result) || result.Selection.IDs == nil || len(*result.Selection.IDs) != len(ids) {
		t.Fatalf("large explicit set rejected: %d", res.Code)
	}
	for i, id := range *result.Selection.IDs {
		if id != ids[i] {
			t.Fatal("selection altered")
		}
	}
	for _, tc := range []struct {
		body   io.Reader
		status int
	}{
		{brokenSelectionBody{}, 400}, {strings.NewReader(strings.Repeat("x", buSelectionBodyLimit+1)), 413},
	} {
		req := httptest.NewRequest("POST", "/api/v1/business-units/selection/resolve", tc.body)
		res := httptest.NewRecorder()
		if decodeSelection(res, req, &result) || res.Code != tc.status {
			t.Fatal("BU transport reason", res.Code, tc.status)
		}
	}
}

func TestFeedback301ExportFileBoundaryHasDistinctHTTPReasons(t *testing.T) {
	pool := &exportfile.Pool{Directory: t.TempDir()}
	file, err := pool.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	block := bytes.Repeat([]byte("x"), 1<<20)
	for n := int64(0); n < exportfile.MaxBytes; n += int64(len(block)) {
		if _, err := file.Write(block); err != nil {
			t.Fatal(err)
		}
	}
	n, sizeErr := file.Write([]byte("x"))
	if n != 0 {
		t.Fatal("over-limit byte accepted")
	}
	unavailable := &exportfile.Pool{Directory: filepath.Join(t.TempDir(), "absent")}
	_, resourceErr := unavailable.Create(context.Background())
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"file_size", sizeErr, 413, "M02_EXPORT_SIZE_LIMIT"},
		{"temporary_storage", resourceErr, 503, "M02_EXPORT_RESOURCE_LIMIT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			writeSelectionError(response, tc.err)
			var body struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal("non-JSON error response", err)
			}
			if response.Code != tc.status || body.Code != tc.code || response.Header().Get("Content-Disposition") != "" || strings.Contains(response.Header().Get("Content-Type"), "text/csv") {
				t.Fatalf("unexpected export error: %d/%s", response.Code, body.Code)
			}
			if tc.status == 413 && strings.Contains(body.Message, "重试") {
				t.Fatal("hard limit must not suggest waiting")
			}
		})
	}
}
