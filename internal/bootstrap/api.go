package bootstrap

import (
	"bytes"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const cookieName = "chnhcm_session"

type API struct {
	Runtime       *Runtime
	Origin        string
	InsecureLocal bool
	mu            sync.Mutex
	attempts      map[string][]time.Time
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	a.accessRoutes(mux)
	a.personnelRoutes(mux)
	a.departmentRoutes(mux)
	a.enterpriseRoutes(mux)
	a.countryRoutes(mux)
	a.geoareaRoutes(mux)
	a.locationRoutes(mux)
	a.legalentityRoutes(mux)
	a.readinessRoutes(mux)
	a.recruitingRoutes(mux)
	a.staffingRoutes(mux)
	mux.Handle("GET /healthz", Handler())
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if a.Runtime.Ready(r.Context()) != nil {
			writeError(w, foundation.Fail(503, "NOT_READY", "初始化尚未完成"))
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("POST /api/v1/session/login", a.login)
	mux.HandleFunc("GET /api/v1/session", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Access.Session(r.Context(), credentials(r))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("DELETE /api/v1/session", func(w http.ResponseWriter, r *http.Request) {
		if e := a.Runtime.Access.Logout(r.Context(), credentials(r)); e != nil {
			writeError(w, e)
			return
		}
		// The server has revoked this session. A delayed response must not erase
		// a newer login cookie; the next login replaces the now-inert old value.
		w.WriteHeader(204)
	})
	mux.HandleFunc("GET /api/v1/organization/context", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Organization.Context(r.Context(), credentials(r))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/organization/operation-reasons", func(w http.ResponseWriter, r *http.Request) {
		_, e := a.Runtime.Organization.Context(r.Context(), credentials(r))
		if e != nil {
			writeError(w, e)
			return
		}
		kind := r.URL.Query().Get("operationType")
		choices := domain.OperationReasons(kind)
		if choices == nil {
			writeError(w, foundation.Fail(501, "M02_CAPABILITY_UNSUPPORTED", "未开放该操作"))
			return
		}
		codes := make([]string, 0, len(choices))
		for code := range choices {
			codes = append(codes, code)
		}
		slices.Sort(codes)
		reasons := []org.Reason{}
		for _, code := range codes {
			reasons = append(reasons, org.Reason{Code: code, Name: choices[code], Default: len(choices) == 1})
		}
		writeJSON(w, 200, reasons)

	})
	mux.HandleFunc("GET /api/v1/business-units", a.list)
	mux.HandleFunc("POST /api/v1/business-units", a.create)
	mux.HandleFunc("POST /api/v1/business-units/selection/resolve", a.resolveSelection)
	mux.HandleFunc("POST /api/v1/business-units/export", a.exportSelection)
	mux.HandleFunc("POST /api/v1/business-units/navigation", func(w http.ResponseWriter, r *http.Request) {
		var in org.NavigationInput
		if !decodeIdentityDraft(w, r, &in, true) {
			return
		}
		v, e := a.Runtime.Organization.Navigate(r.Context(), credentials(r), in)
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/business-units/{id}/records", a.add)
	mux.HandleFunc("POST /api/v1/business-units/{id}/records/{recordId}/correct", a.correct)
	mux.HandleFunc("POST /api/v1/business-units/{id}/records/{recordId}/delete", a.deleteRecord)
	mux.HandleFunc("POST /api/v1/business-units/{id}/records/{recordId}/redate", a.redateRecord)
	mux.HandleFunc("POST /api/v1/business-units/{id}/deactivate", a.deactivate)
	mux.HandleFunc("POST /api/v1/business-units/{id}/reactivate", a.reactivate)
	mux.HandleFunc("POST /api/v1/business-units/{id}/preview", a.preview)
	mux.HandleFunc("POST /api/v1/business-units/{id}/lifecycle-pairs", a.insertLifecyclePair)
	mux.HandleFunc("POST /api/v1/business-units/{id}/lifecycle-pairs/delete", a.deleteLifecyclePair)
	mux.HandleFunc("GET /api/v1/business-units/{id}", a.detail)
	mux.HandleFunc("GET /api/v1/business-units/{id}/changes", a.changes)
	mux.HandleFunc("POST /api/v1/organization/submissions/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
		v, e := a.Runtime.Organization.Resolve(r.Context(), credentials(r), r.PathValue("id"))
		if e != nil {
			var pg *pgconn.PgError
			if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) || (errors.As(e, &pg) && (pg.Code == "55P03" || pg.Code == "57014")) {
				writeJSON(w, 202, org.Result{SubmissionID: r.PathValue("id"), Outcome: "pending"})
				return
			}
		}
		respond(w, 200, v, e)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method != "GET" && r.Method != "HEAD" {
			if !ValidOrigin(a.Origin, a.InsecureLocal) || r.Header.Get("Origin") != a.Origin {
				writeError(w, foundation.Fail(403, "ACCESS_DENIED", "请求来源不受信任"))
				return
			}
		}
		budget := 10 * time.Second
		// Department selection resolve performs the same complete guarded
		// capture as start/export. Submission-result and BU resolve retain
		// their existing shorter budget.
		shortResolve := strings.HasSuffix(r.URL.Path, "/resolve") && r.URL.Path != "/api/v1/departments/selection/resolve"
		if strings.HasSuffix(r.URL.Path, "/preview") || shortResolve || strings.HasPrefix(r.URL.Path, "/api/v1/departments/") && strings.HasSuffix(r.URL.Path, "/changes") {
			budget = 5 * time.Second
		}
		ctx, cancel := context.WithTimeout(r.Context(), budget)
		defer cancel()
		r = r.WithContext(ctx)
		// An empty body has already started the connection background reader;
		// setting its read deadline would cancel a valid long-running request.
		if r.Body != nil && r.Body != http.NoBody {
			deadline := time.Now().Add(5 * time.Second)
			if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
				deadline = d
			}
			_ = http.NewResponseController(w).SetReadDeadline(deadline)
		}
		r.Header.Set("X-CHNHCM-Request-ID", foundation.NewID())
		if os.Getenv("CHNHCM_MEASURE_STAGES") == "1" {
			stages := []map[string]any{}
			var stageMu sync.Mutex
			ctx = foundation.WithStageObserver(ctx, func(name string, elapsed time.Duration) {
				stageMu.Lock()
				defer stageMu.Unlock()
				stages = append(stages, map[string]any{"stage": name, "ms": float64(elapsed) / float64(time.Millisecond)})
			})
			r = r.WithContext(ctx)
			defer func() {
				stageMu.Lock()
				defer stageMu.Unlock()
				sample := r.Header.Get("X-CHNHCM-Measurement-ID")
				if !foundation.ValidID(sample) {
					sample = ""
				}
				b, _ := json.Marshal(map[string]any{"request": r.Header.Get("X-CHNHCM-Request-ID"), "sample": sample, "method": r.Method, "stages": stages})
				log.Printf("M02_STAGES %s", b)
			}()
		}
		mux.ServeHTTP(w, r)
	})
}

func credentials(r *http.Request) access.Credentials {
	c := access.Credentials{CSRF: r.Header.Get("X-CSRF-Token"), Write: r.Method != "GET" && r.Method != "HEAD"}
	if v, e := r.Cookie(cookieName); e == nil {
		c.Token = v.Value
	}
	return c
}
func (a *API) setCookie(w http.ResponseWriter, v string, max int) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: v, Path: "/", HttpOnly: true, Secure: !a.InsecureLocal, SameSite: http.SameSiteLaxMode, MaxAge: max})
}
func (a *API) limit(key string, max int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.attempts == nil {
		a.attempts = map[string][]time.Time{}
	}
	now := time.Now()
	cut := now.Add(-time.Minute)
	for k, v := range a.attempts {
		if len(v) == 0 || v[len(v)-1].Before(cut) {
			delete(a.attempts, k)
		}
	}
	old := a.attempts[key]
	current := []time.Time{}
	for _, t := range old {
		if !t.Before(cut) {
			current = append(current, t)
		}
	}
	if len(current) >= max || len(a.attempts) >= 5000 {
		return false
	}
	a.attempts[key] = append(current, now)
	return true
}
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in access.LoginInput
	if _, e := decode(r, &in); e != nil {
		writeError(w, e)
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if !a.limit("ip:"+host, 120) || !a.limit("user:"+strings.ToLower(strings.TrimSpace(in.Login)), 20) {
		writeError(w, foundation.Fail(429, "AUTH_RATE_LIMITED", "登录尝试过于频繁，请稍后重试"))
		return
	}
	in.PreviousToken = credentials(r).Token
	v, e := a.Runtime.Access.Login(r.Context(), in, r.Header.Get("X-CHNHCM-Request-ID"))
	if e != nil {
		writeError(w, e)
		return
	}
	a.setCookie(w, v.Token, 8*60*60)
	writeJSON(w, 200, v.Principal)
}
func (a *API) create(w http.ResponseWriter, r *http.Request) {
	var in org.CreateInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	v, e := a.Runtime.Organization.Create(r.Context(), credentials(r), org.CreateCommand{Input: in, Digest: digest(r, raw), SubmissionID: r.Header.Get("Idempotency-Key"), Fields: fields(raw)})
	commandResponse(w, 201, v, e)
}

func (a *API) list(w http.ResponseWriter, r *http.Request) {
	q, e := queryValues(r, "q", "status", "sort", "page", "pageSize", "filterField", "filterOperator", "filterValue")
	if e != nil {
		writeError(w, e)
		return
	}
	page, e := number(q.Get("page"), 1)
	if e != nil {
		writeError(w, e)
		return
	}
	size, e := number(q.Get("pageSize"), 20)
	if e != nil {
		writeError(w, e)
		return
	}
	var filter *org.ColumnFilter
	if q.Has("filterField") || q.Has("filterOperator") || q.Has("filterValue") {
		filter = &org.ColumnFilter{Field: q.Get("filterField"), Operator: q.Get("filterOperator"), Value: q.Get("filterValue")}
	}
	v, e := a.Runtime.Organization.List(r.Context(), credentials(r), org.ListInput{Q: q.Get("q"), Status: q.Get("status"), Sort: q.Get("sort"), Page: page, PageSize: size, Filter: filter})
	respond(w, 200, v, e)
}
func (a *API) detail(w http.ResponseWriter, r *http.Request) {
	q, e := queryValues(r, "sourceVersion", "selectedRecordId")
	if e != nil {
		writeError(w, e)
		return
	}
	for _, name := range []string{"sourceVersion", "selectedRecordId"} {
		if q.Has(name) && !foundation.ValidID(q.Get(name)) {
			writeError(w, foundation.Fail(400, "M02_INVALID_INPUT", "所选历史资料或记录无效"))
			return
		}
	}
	v, e := a.Runtime.Organization.Detail(r.Context(), credentials(r), org.DetailInput{ID: r.PathValue("id"), SourceVersion: q.Get("sourceVersion"), SelectedRecordID: q.Get("selectedRecordId")})
	respond(w, 200, v, e)
}
func queryValues(r *http.Request, names ...string) (url.Values, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, foundation.Fail(400, "M02_INVALID_INPUT", "查询条件编码无效")
	}
	for k, v := range q {
		found := false
		for _, n := range names {
			if n == k {
				found = true
			}
		}
		if !found || len(v) != 1 {
			return nil, foundation.Fail(400, "M02_INVALID_INPUT", "未知或重复的查询条件")
		}
	}
	return q, nil
}
func number(s string, fallback int) (int, error) {
	if s == "" {
		return fallback, nil
	}
	n, e := strconv.Atoi(s)
	if e != nil || n < 1 || n > 1000000 {
		return 0, foundation.Fail(400, "M02_INVALID_INPUT", "分页参数无效")
	}
	return n, nil
}
func respond(w http.ResponseWriter, status int, v any, e error) {
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, status, v)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error) {
	var f *foundation.Fault
	if !errors.As(err, &f) {
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			log.Printf("request database failure: SQLSTATE=%s constraint=%s", pg.Code, pg.ConstraintName)
		}
		f = foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "服务暂时不可用，请稍后重试")
	}
	writeJSON(w, f.Status, f)
}

func decode(r *http.Request, out any) ([]byte, error) {
	return decodeWithLimit(r, out, 65536)
}

func decodeWithLimit(r *http.Request, out any, limit int64) ([]byte, error) {
	bad := foundation.Fail(400, "M02_INVALID_INPUT", "请求内容无效")
	media, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil || media != "application/json" {
		return nil, bad
	}
	data, e := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if e != nil || int64(len(data)) > limit || !utf8.Valid(data) {
		return nil, bad
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) < 2 || trimmed[0] != '{' {
		return nil, bad
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if e = uniqueJSON(d); e != nil {
		return nil, bad
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, bad
	}
	shapeCode := "M02_FORBIDDEN_INPUT"
	if strings.HasPrefix(reflect.TypeOf(out).Elem().Name(), "Department") {
		shapeCode = "M02_INVALID_INPUT"
	}
	if !strictShape(data, reflect.TypeOf(out).Elem()) {
		return nil, foundation.Fail(400, shapeCode, "请求内容不完整或格式有误")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if e = d.Decode(out); e != nil {
		return nil, foundation.Fail(400, shapeCode, "请求包含未知、只读或类型不正确的字段")
	}
	return data, nil
}
func uniqueJSON(d *json.Decoder) error {
	t, e := d.Token()
	if e != nil {
		return e
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return e
			}
			s, ok := k.(string)
			if !ok || seen[s] {
				return errors.New("duplicate key")
			}
			seen[s] = true
			if e = uniqueJSON(d); e != nil {
				return e
			}
		}
		_, e = d.Token()
		return e
	case '[':
		for d.More() {
			if e = uniqueJSON(d); e != nil {
				return e
			}
		}
		_, e = d.Token()
		return e
	default:
		return errors.New("unexpected delimiter")
	}
}

// Only explicitly inlined request structs contribute their fields to the wire object.
func protocolFields(t reflect.Type) []reflect.StructField {
	var fields []reflect.StructField
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct && field.Tag.Get("json") == ",inline" {
			fields = append(fields, protocolFields(field.Type)...)
		} else {
			fields = append(fields, field)
		}
	}
	return fields
}

// Match exact protocol names and presence; encoding/json alone accepts case-folded names and null strings.
func strictShape(raw []byte, t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		if string(raw) == "null" {
			return true
		}
		return strictShape(raw, t.Elem())
	}
	if string(raw) == "null" {
		return false
	}
	if t == reflect.TypeFor[time.Time]() {
		var moment time.Time
		return json.Unmarshal(raw, &moment) == nil
	}
	if t.Kind() == reflect.Struct {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return false
		}
		expected := map[string]bool{}
		for _, f := range protocolFields(t) {
			tag := f.Tag.Get("json")
			parts := strings.Split(tag, ",")
			key := parts[0]
			if key == "-" || key == "" {
				continue
			}
			expected[key] = true
			value, ok := obj[key]
			optional := slicesContains(parts, "omitempty") || f.Type.Kind() == reflect.Pointer
			if !ok {
				if !optional {
					return false
				}
				continue
			}
			if !strictShape(value, f.Type) {
				return false
			}
		}
		for key := range obj {
			if !expected[key] {
				return false
			}
		}
		return true
	}
	if t.Kind() == reflect.Slice {
		var values []json.RawMessage
		if json.Unmarshal(raw, &values) != nil {
			return false
		}
		for _, v := range values {
			if !strictShape(v, t.Elem()) {
				return false
			}
		}
		return true
	}
	if t.Kind() == reflect.Map {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return false
		}
		for key, v := range obj {
			if (t.Name() == "ChangesInput" || t.Name() == "EnterpriseChangesInput") && (key != "name" && key != "nameEn") {
				return false
			}
			if t.Name() == "LegalEntityChangesInput" && (!slices.Contains([]string{"name", "nameEn", "countryRegionId", "subjectType", "registrationType", "registrationNumber", "establishingOrganization"}, key) || slices.Contains([]string{"name", "countryRegionId", "subjectType"}, key) && string(v) == "null") {
				return false
			}
			if t.Name() == "LocationChangesInput" && (!slices.Contains([]string{"name", "nameEn", "countryRegionId", "geoAreaId", "addressLine", "postalCode"}, key) || slices.Contains([]string{"name", "countryRegionId", "addressLine"}, key) && string(v) == "null") {
				return false
			}
			if t.Name() == "CountryChangesInput" && !slices.Contains([]string{"name", "nameEn", "phoneCode"}, key) {
				return false
			}
			if t.Name() == "DepartmentChangesInput" && !slices.Contains([]string{"name", "shortName", "managementBuId", "parentDepartmentId"}, key) {
				return false
			}
			if (key == "name" || t.Name() == "DepartmentChangesInput" && key == "managementBuId") && string(v) == "null" {
				return false
			}
			if !strictShape(v, t.Elem()) {
				return false
			}
		}
		return true
	}
	return true
}
func slicesContains(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
