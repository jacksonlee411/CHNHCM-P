//go:build integration

package integration_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/foundation"
)

func TestT07ExactBodyLimitAndRealStalledSocket(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	for _, size := range []int{65536, 65537} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			input, e := json.Marshal(createBody(fmt.Sprintf("T07_BODY_%d", size), "2026-01-01"))
			must(t, e)
			input = append(input, bytes.Repeat([]byte(" "), size-len(input))...)
			key := foundation.NewID()
			req, e := http.NewRequest("POST", f.server.URL+"/api/v1/business-units", bytes.NewReader(input))
			must(t, e)
			req.Header.Set("Origin", f.server.URL)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", b.csrf)
			req.Header.Set("Idempotency-Key", key)
			res, e := b.client.Do(req)
			must(t, e)
			_, e = io.Copy(io.Discard, res.Body)
			res.Body.Close()
			must(t, e)
			want := 201
			if size > 65536 {
				want = 400
			}
			if res.StatusCode != want {
				t.Fatal(res.StatusCode, want)
			}
			n := f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key)
			if (size == 65536 && n != 1) || (size == 65537 && n != 0) {
				t.Fatal("oversize body reached business execution")
			}
		})
	}
	// Send a real TCP request with an incomplete body and do not complete it.
	// A context flag alone cannot unblock this socket Read; the actual handler's
	// ResponseController deadline must return a response within its five seconds.
	u, e := url.Parse(f.server.URL)
	must(t, e)
	conn, e := net.Dial("tcp", u.Host)
	must(t, e)
	defer conn.Close()
	must(t, conn.SetDeadline(time.Now().Add(8*time.Second)))
	key := foundation.NewID()
	cookie := ""
	for _, c := range b.client.Jar.Cookies(u) {
		if cookie != "" {
			cookie += "; "
		}
		cookie += c.Name + "=" + c.Value
	}
	header := fmt.Sprintf("POST /api/v1/business-units HTTP/1.1\r\nHost: %s\r\nOrigin: %s\r\nContent-Type: application/json\r\nContent-Length: 100\r\nCookie: %s\r\nX-CSRF-Token: %s\r\nIdempotency-Key: %s\r\nConnection: close\r\n\r\n{", u.Host, f.server.URL, cookie, b.csrf, key)
	start := time.Now()
	_, e = io.WriteString(conn, header)
	must(t, e)
	res, e := http.ReadResponse(bufio.NewReader(conn), nil)
	must(t, e)
	defer res.Body.Close()
	elapsed := time.Since(start)
	if res.StatusCode != 400 || elapsed < 4500*time.Millisecond || elapsed > 6*time.Second {
		t.Fatal("real body deadline", res.StatusCode, elapsed)
	}
	if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 0 {
		t.Fatal("incomplete body produced terminal")
	}
	t.Logf("actual stalled TCP body stopped in %s (<5s deadline plus scheduling tolerance); no command entered", elapsed)
}
