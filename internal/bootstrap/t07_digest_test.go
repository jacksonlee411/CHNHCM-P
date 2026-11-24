package bootstrap

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCanonicalFixedPairDigest(t *testing.T) {
	base := `{"expectedSourceVersion":"source","anchorRecordId":"anchor","deactivationDate":"2026-03-01","reactivationDate":"2026-04-01","deactivationReason":"BUSINESS_CLOSURE","reactivationReason":"BUSINESS_RESUMPTION"}`
	reordered := `{"reactivationReason":"BUSINESS_RESUMPTION","deactivationReason":"BUSINESS_CLOSURE","reactivationDate":"2026-04-01","deactivationDate":"2026-03-01","anchorRecordId":"anchor","expectedSourceVersion":"source"}`
	req := httptest.NewRequest("POST", "/api/v1/business-units/bu/lifecycle-pairs", nil)
	original := digest(req, []byte(base))
	if !bytes.Equal(original, digest(req, []byte(reordered))) {
		t.Fatal("object order changed identity")
	}
	for _, change := range [][2]string{{"source", "source2"}, {"anchor", "other"}, {"2026-03-01", "2026-03-02"}, {"2026-04-01", "2026-04-02"}, {"BUSINESS_CLOSURE", "different"}, {"BUSINESS_RESUMPTION", "different"}} {
		if bytes.Equal(original, digest(req, []byte(strings.Replace(base, change[0], change[1], 1)))) {
			t.Fatal("semantic input excluded", change[0])
		}
	}
	for _, path := range []string{"/api/v1/business-units/other/lifecycle-pairs", "/api/v1/business-units/bu/lifecycle-pairs/delete"} {
		if bytes.Equal(original, digest(httptest.NewRequest("POST", path, nil), []byte(base))) {
			t.Fatal("path excluded")
		}
	}
}
