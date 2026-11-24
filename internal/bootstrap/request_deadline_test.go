package bootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"sync/atomic"
	"testing"
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// Only HTTP deadline/connection behavior is under test. Domain acceptance is
// independently covered by the real-DB suite; this service deliberately delays
// after HTTP parsing, without consuming a database or timing a business rule.
type delayedDepartment struct {
	org.DepartmentService
	delayRead bool
	reads     atomic.Int32
}

func (s *delayedDepartment) Detail(ctx context.Context, _ foundation.Credentials, in org.DepartmentDetailInput) (org.DepartmentDetail, error) {
	if s.delayRead && s.reads.Add(1) == 1 {
		if e := waitForApplication(ctx); e != nil {
			return org.DepartmentDetail{}, e
		}
	}
	return org.DepartmentDetail{DepartmentID: in.DepartmentID}, nil
}
func (s *delayedDepartment) Execute(ctx context.Context, _ foundation.Credentials, _ org.DepartmentCommand) (org.DepartmentResult, error) {
	if e := waitForApplication(ctx); e != nil {
		return org.DepartmentResult{}, e
	}
	return org.DepartmentResult{Outcome: "accepted"}, nil
}
func waitForApplication(ctx context.Context) error {
	timer := time.NewTimer(5200 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func TestBodyDeadlineEndsBeforeApplicationAndKeepAlive(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			service := &delayedDepartment{delayRead: method == "GET"}
			api := &bootstrap.API{Runtime: &bootstrap.Runtime{Department: service}, InsecureLocal: true}
			server := httptest.NewServer(api.Handler())
			defer server.Close()
			api.Origin = server.URL
			transport := &http.Transport{MaxConnsPerHost: 1}
			defer transport.CloseIdleConnections()
			client := &http.Client{Transport: transport, Timeout: 8 * time.Second}
			var connections []net.Conn
			call := func(method, path string, body any, want int) {
				t.Helper()
				var raw []byte
				if body != nil {
					var e error
					raw, e = json.Marshal(body)
					if e != nil {
						t.Fatal(e)
					}
				}
				req, e := http.NewRequest(method, server.URL+path, bytes.NewReader(raw))
				if e != nil {
					t.Fatal(e)
				}
				req.Header.Set("Origin", server.URL)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Idempotency-Key", foundation.NewID())
				req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{GotConn: func(info httptrace.GotConnInfo) { connections = append(connections, info.Conn) }}))
				response, e := client.Do(req)
				if e != nil {
					t.Fatal(e)
				}
				data, e := io.ReadAll(response.Body)
				response.Body.Close()
				if e != nil || response.StatusCode != want {
					t.Fatalf("%s status=%d body=%s err=%v", method, response.StatusCode, data, e)
				}
			}
			id := foundation.NewID()
			if method == "GET" {
				call("GET", "/api/v1/departments/"+id, nil, 200)
			} else {
				call("POST", "/api/v1/departments", map[string]any{"code": "DEADLINE", "name": "请求体预算", "effectiveDate": "2020-01-01", "managementBuId": foundation.NewID(), "operationType": "CREATE", "operationReason": "NEW_ORGANIZATION"}, 201)
			}
			call("GET", "/api/v1/departments/"+id, nil, 200)
			if len(connections) != 2 || connections[0] != connections[1] {
				t.Fatal(fmt.Sprintf("request lost its reusable connection: %d", len(connections)))
			}
		})
	}
}
