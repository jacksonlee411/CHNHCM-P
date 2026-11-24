package bootstrap

import (
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

func staffingError(e error) error {
	if e == nil {
		return nil
	}
	var f *foundation.Fault
	if errors.As(e, &f) && strings.HasPrefix(f.Code, "M02_") {
		v := *f
		v.Code = "M03_" + strings.TrimPrefix(f.Code, "M02_")
		return &v
	}
	if errors.As(e, &f) {
		return e
	}
	return foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "服务暂时不可用，请稍后重试")
}
func staffingShape(raw []byte, kind s.Kind, action string, referenceOnly ...bool) error {
	common := []string{"expectedSourceVersion"}
	required := []string{"expectedSourceVersion"}
	allowed := []string{}
	switch action {
	case "CREATE":
		allowed = []string{"code", "name", "nameEn", "effectiveDate", "operationType", "operationReason"}
		required = []string{"code", "name", "operationType", "operationReason"}
		if kind == s.Position {
			allowed = append(allowed, "departmentId", "jobId", "locationId", "gradeId")
			required = append(required, "departmentId", "jobId", "locationId")
		} else if kind == s.JobFunction || kind == s.Family || kind == s.Grade || kind == s.Ladder {
			allowed = append(allowed, "description")
		} else {
			allowed = append(allowed, "jobFunctionId", "jobFamilyId", "responsibilities", "requirements")
			required = append(required, "jobFamilyId")
		}
		if kind == s.Job || kind == s.Ladder {
			allowed = append(allowed, "gradeIds")
		}
		if kind == s.Ladder {
			required = append(required, "gradeIds")
		}
		common = nil
	case "ADD", "DEACTIVATE", "REACTIVATE":
		allowed = []string{"anchorRecordId", "effectiveDate", "operationType", "operationReason"}
		required = append(required, allowed...)
		if action != "DEACTIVATE" {
			allowed = append(allowed, "changes")
		}
	case "CORRECT":
		allowed = []string{"expectedRecordRevision", "changes", "classification", "newEffectiveDate"}
		required = append(required, "expectedRecordRevision")
	case "REDATE":
		allowed = []string{"expectedRecordRevision", "newEffectiveDate"}
		required = append(required, allowed...)
	case "DELETE":
		allowed = []string{"expectedRecordRevision"}
		required = append(required, allowed...)
	case "INSERT_LIFECYCLE_PAIR":
		allowed = []string{"anchorRecordId", "deactivationDate", "reactivationDate", "deactivationReason", "reactivationReason", "changes"}
		required = append(required, allowed[:5]...)
	case "MAINTAIN_LIFECYCLE_PAIR", "DELETE_LIFECYCLE_PAIR":
		allowed = []string{"deactivationRecordId", "deactivationRevision", "reactivationRecordId", "reactivationRevision"}
		required = append(required, allowed...)
		if action == "MAINTAIN_LIFECYCLE_PAIR" {
			allowed = append(allowed, "deactivationDate", "reactivationDate", "deactivationReason", "reactivationReason", "changes")
			required = append(required, allowed[4:8]...)
		}
	default:
		return foundation.Fail(400, "M03_INVALID_INPUT", "操作无效")
	}
	// Transport shape only. An empty optional patch is meaningful for Staffing
	// date/classification-only edits and lifecycle inheritance; domain.Build
	// decides no-change and validates the final timeline.
	var obj map[string]json.RawMessage
	_ = json.Unmarshal(raw, &obj)
	allowed = append(common, allowed...)
	for key, value := range obj {
		if !slices.Contains(allowed, key) || key == "classification" && string(value) == "null" {
			return foundation.Fail(400, "M03_FORBIDDEN_INPUT", "此动作不接受该字段或空分类")
		}
	}
	if len(referenceOnly) > 0 && referenceOnly[0] {
		return nil
	}
	for _, key := range required {
		if _, ok := obj[key]; !ok {
			return foundation.Fail(400, "M03_INVALID_INPUT", "操作内容不完整，请重新打开表单")
		}
	}
	return nil
}
func staffingQuery(r *http.Request, k s.Kind) (s.Query, error) {
	fields := []string{"q", "status", "filterField", "filterOperator", "filterValue", "sort", "page", "pageSize"}
	if k == s.Job || k == s.Family {
		fields = append(fields, "jobFamilyId")
	}
	if k == s.Job {
		fields = append(fields, "jobFunctionId", "jobFunctionMissing")
	}
	if k == s.Position {
		fields = append(fields, "scopeDepartmentId", "includeDescendants")
	}
	q, e := queryValues(r, fields...)
	if e != nil {
		return s.Query{}, staffingError(e)
	}
	page, e := number(q.Get("page"), 1)
	if e != nil {
		return s.Query{}, staffingError(e)
	}
	size, e := number(q.Get("pageSize"), 20)
	var descendants *bool
	if q.Has("scopeDepartmentId") && !foundation.ValidID(q.Get("scopeDepartmentId")) {
		return s.Query{}, foundation.Fail(400, "M03_INVALID_INPUT", "部门范围无效")
	}
	if q.Has("includeDescendants") {
		flag := q.Get("includeDescendants")
		if flag != "true" && flag != "false" || !q.Has("scopeDepartmentId") {
			return s.Query{}, foundation.Fail(400, "M03_INVALID_INPUT", "部门范围条件无效")
		}
		include := flag == "true"
		descendants = &include
	}
	if q.Has("jobFunctionId") && !foundation.ValidID(q.Get("jobFunctionId")) {
		return s.Query{}, foundation.Fail(400, "M03_INVALID_INPUT", "职类筛选无效")
	}
	if q.Has("jobFunctionMissing") && q.Get("jobFunctionMissing") != "true" && q.Get("jobFunctionMissing") != "false" {
		return s.Query{}, foundation.Fail(400, "M03_INVALID_INPUT", "未登记职类筛选无效")
	}
	return s.Query{
		Q: q.Get("q"), Status: q.Get("status"), JobFamilyID: q.Get("jobFamilyId"),
		JobFunctionID: q.Get("jobFunctionId"), JobFunctionMissing: q.Get("jobFunctionMissing") == "true",
		ScopeDepartmentID: q.Get("scopeDepartmentId"), IncludeDescendants: descendants,
		FilterField: q.Get("filterField"), FilterOperator: q.Get("filterOperator"), FilterValue: q.Get("filterValue"),
		Sort: q.Get("sort"), Page: page, PageSize: size,
	}, staffingError(e)
}
func (a *API) staffingRoutes(mux *http.ServeMux) {
	a.hierarchyRoutes(mux)
	a.quotaRoutes(mux)
	for _, entry := range []struct {
		path string
		kind s.Kind
	}{{"/api/v1/job-families", s.Family}, {"/api/v1/job-functions", s.JobFunction}, {"/api/v1/jobs", s.Job}, {"/api/v1/grades", s.Grade}, {"/api/v1/grade-ladders", s.Ladder}, {"/api/v1/positions", s.Position}} {
		base, k := entry.path, entry.kind
		if k == s.Position {
			mux.HandleFunc("POST "+base+"/reference-context", func(w http.ResponseWriter, r *http.Request) {
				query, err := queryValues(r, "field")
				if err != nil {
					writeError(w, staffingError(err))
					return
				}
				var input s.Input
				raw, err := decode(r, &input)
				if err == nil {
					var fields map[string]json.RawMessage
					_ = json.Unmarshal(raw, &fields)
					for _, key := range []string{"objectId", "action", "recordId", "referenceQ", "referencePage", "referencePageSize"} {
						delete(fields, key)
					}
					raw, _ = json.Marshal(fields)
					err = staffingShape(raw, k, input.Action, true)
				}
				if err != nil {
					writeError(w, staffingError(err))
					return
				}
				result, err := a.Runtime.Staffing.PositionReferences(r.Context(), credentials(r), query.Get("field"), s.Command{Kind: k, ObjectID: input.ObjectID, Input: input})
				respond(w, 200, result, staffingError(err))
			})
		}
		if k == s.Job || k == s.Ladder {
			mux.HandleFunc("POST "+base+"/grade-reference-context", func(w http.ResponseWriter, r *http.Request) {
				var in s.Input
				raw, e := decode(r, &in)
				if e == nil {
					var fields map[string]json.RawMessage
					_ = json.Unmarshal(raw, &fields)
					for _, key := range []string{"objectId", "action", "recordId", "referenceQ", "referencePage", "referencePageSize"} {
						delete(fields, key)
					}
					raw, _ = json.Marshal(fields)
					e = staffingShape(raw, k, in.Action, true)
				}
				if e != nil {
					writeError(w, staffingError(e))
					return
				}
				v, e := a.Runtime.Staffing.GradeReferenceContext(r.Context(), credentials(r), s.Command{Kind: k, ObjectID: in.ObjectID, Input: in})
				respond(w, 200, v, staffingError(e))
			})
		}

		mux.HandleFunc("GET "+base+"/context", func(w http.ResponseWriter, r *http.Request) {
			v, e := a.Runtime.Staffing.Context(r.Context(), credentials(r), k)
			respond(w, 200, v, staffingError(e))
		})
		mux.HandleFunc("GET "+base+"/operation-reasons", func(w http.ResponseWriter, r *http.Request) {
			q, e := queryValues(r, "operationType")
			if e != nil {
				writeError(w, staffingError(e))
				return
			}
			v, e := a.Runtime.Staffing.Reasons(r.Context(), credentials(r), k, q.Get("operationType"))
			respond(w, 200, v, staffingError(e))
		})
		mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
			q, e := staffingQuery(r, k)
			if e != nil {
				writeError(w, staffingError(e))
				return
			}
			v, e := a.Runtime.Staffing.List(r.Context(), credentials(r), k, q)
			respond(w, 200, v, staffingError(e))
		})
		mux.HandleFunc("GET "+base+"/{id}", func(w http.ResponseWriter, r *http.Request) {
			q, e := queryValues(r, "sourceVersion", "selectedRecordId")
			if e != nil {
				writeError(w, staffingError(e))
				return
			}
			v, e := a.Runtime.Staffing.Detail(r.Context(), credentials(r), k, r.PathValue("id"), q.Get("sourceVersion"), q.Get("selectedRecordId"))
			respond(w, 200, v, staffingError(e))
		})
		mux.HandleFunc("GET "+base+"/{id}/changes", func(w http.ResponseWriter, r *http.Request) {
			q, e := staffingLogQuery(r)
			if e != nil {
				writeError(w, staffingError(e))
				return
			}
			v, e := a.Runtime.Staffing.Changes(r.Context(), credentials(r), k, r.PathValue("id"), q)
			respond(w, 200, v, staffingError(e))
		})
		mux.HandleFunc("GET "+base+"/{id}/navigation", func(w http.ResponseWriter, r *http.Request) {
			q, e := staffingQuery(r, k)
			if e != nil {
				writeError(w, staffingError(e))
				return
			}
			v, e := a.Runtime.Staffing.Navigation(r.Context(), credentials(r), k, r.PathValue("id"), q)
			respond(w, 200, v, staffingError(e))
		})
		mux.HandleFunc("POST "+base+"/{id}/preview", func(w http.ResponseWriter, r *http.Request) {
			var in s.Input
			raw, e := decode(r, &in)
			if e == nil {
				var fields map[string]json.RawMessage
				_ = json.Unmarshal(raw, &fields)
				delete(fields, "action")
				delete(fields, "recordId")
				raw, _ = json.Marshal(fields)
				e = staffingShape(raw, k, in.Action)
			}
			if e != nil {
				writeError(w, staffingError(e))
				return
			}
			v, e := a.Runtime.Staffing.Preview(r.Context(), credentials(r), s.Command{Kind: k, ObjectID: r.PathValue("id"), Input: in})
			respond(w, 200, v, staffingError(e))
		})
		for _, route := range []struct{ path, action string }{{"", "CREATE"}, {"/{id}/records", "ADD"}, {"/{id}/records/{recordId}/correct", "CORRECT"}, {"/{id}/records/{recordId}/redate", "REDATE"}, {"/{id}/records/{recordId}/delete", "DELETE"}, {"/{id}/deactivate", "DEACTIVATE"}, {"/{id}/reactivate", "REACTIVATE"}, {"/{id}/lifecycle-pairs/insert", "INSERT_LIFECYCLE_PAIR"}, {"/{id}/lifecycle-pairs/maintain", "MAINTAIN_LIFECYCLE_PAIR"}, {"/{id}/lifecycle-pairs/delete", "DELETE_LIFECYCLE_PAIR"}} {
			mux.HandleFunc("POST "+base+route.path, func(w http.ResponseWriter, r *http.Request) {
				var in s.Input
				raw, e := decode(r, &in)
				if e == nil {
					e = staffingShape(raw, k, route.action)
				}
				if e != nil {
					writeError(w, staffingError(e))
					return
				}
				in.Action = route.action
				in.RecordID = r.PathValue("recordId")
				v, e := a.Runtime.Staffing.Execute(r.Context(), credentials(r), s.Command{Kind: k, ObjectID: r.PathValue("id"), SubmissionID: r.Header.Get("Idempotency-Key"), Digest: staffingDigest(r, raw, k), Input: in})
				if e != nil {
					writeError(w, staffingError(e))
					return
				}
				status := 200
				if v.Outcome == "rejected" {
					status = v.FailureStatus
					if status < 400 {
						status = 409
					}
				} else if route.action == "CREATE" && v.Outcome == "accepted" {
					status = 201
				}
				if v.Outcome == "rejected" && v.Failure != nil {
					encoded, _ := json.Marshal(v)
					var body map[string]any
					_ = json.Unmarshal(encoded, &body)
					body["code"] = v.Failure.Code
					body["message"] = v.Failure.Message
					if v.Failure.Field != "" {
						body["field"] = v.Failure.Field
					}
					writeJSON(w, status, body)
					return
				}
				writeJSON(w, status, v)
			})
		}
		mux.HandleFunc("POST "+base+"/submissions/{id}/resolve", func(w http.ResponseWriter, r *http.Request) {
			v, e := a.Runtime.Staffing.Resolve(r.Context(), credentials(r), k, r.PathValue("id"))
			var pg *pgconn.PgError
			if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) || errors.As(e, &pg) && (pg.Code == "55P03" || pg.Code == "57014") {
				writeJSON(w, 202, s.Result{Kind: k, SubmissionID: r.PathValue("id"), Outcome: "pending"})
				return
			}
			respond(w, 200, v, staffingError(e))
		})
		for _, op := range []string{"start", "resolve", "cancel"} {
			mux.HandleFunc("POST "+base+"/selection/"+op, func(w http.ResponseWriter, r *http.Request) {
				var in s.SelectionInput
				if _, e := decode(r, &in); e != nil {
					writeError(w, staffingError(e))
					return
				}
				v, e := a.Runtime.Staffing.Select(r.Context(), credentials(r), k, op, in)
				respond(w, 200, v, staffingError(e))
			})
		}
		mux.HandleFunc("POST "+base+"/export", func(w http.ResponseWriter, r *http.Request) {
			var in s.SelectionInput
			if _, e := decode(r, &in); e != nil {
				writeError(w, staffingError(e))
				return
			}
			v, e := a.Runtime.Staffing.Export(r.Context(), credentials(r), k, in)
			if e != nil {
				writeError(w, staffingError(e))
				return
			}
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="chnhcm-`+string(k)+`-`+v.Date+`.csv"`)
			w.Header().Set("X-Selected-Count", strconv.Itoa(v.Count))
			w.WriteHeader(200)
			_, _ = w.Write(v.Bytes)
		})
	}
	for path, kind := range map[string]s.Kind{
		"/api/v1/jobs/reference-context":          s.Family,
		"/api/v1/jobs/function-reference-context": s.JobFunction,
	} {
		mux.HandleFunc("POST "+path, func(w http.ResponseWriter, r *http.Request) {
			var in s.Input
			if _, e := decode(r, &in); e != nil {
				writeError(w, staffingError(e))
				return
			}
			v, e := a.Runtime.Staffing.ClassificationReferenceContext(r.Context(), credentials(r), kind, s.Command{Kind: s.Job, ObjectID: in.ObjectID, Input: in})
			respond(w, 200, v, staffingError(e))
		})
	}
}

func staffingLogQuery(r *http.Request) (s.LogQuery, error) {
	q, e := queryValues(r, "page", "pageSize", "filterField", "filterOperator", "filterValue", "contextSourceVersion", "throughSourceVersion")
	if e != nil {
		return s.LogQuery{}, staffingError(e)
	}
	for key, values := range q {
		if len(values) != 1 || values[0] == "" {
			return s.LogQuery{}, foundation.Fail(400, "M03_INVALID_INPUT", "日志参数不能为空："+key)
		}
	}
	page, e := number(q.Get("page"), 1)
	if e != nil {
		return s.LogQuery{}, staffingError(e)
	}
	size, e := number(q.Get("pageSize"), 20)
	if page < 1 || size < 1 {
		return s.LogQuery{}, foundation.Fail(400, "M03_INVALID_INPUT", "日志分页参数无效")
	}
	return s.LogQuery{Page: page, PageSize: size, FilterField: q.Get("filterField"), FilterOperator: q.Get("filterOperator"), FilterValue: q.Get("filterValue"), ContextSourceVersion: q.Get("contextSourceVersion"), ThroughSourceVersion: q.Get("throughSourceVersion")}, staffingError(e)
}

// Only Job's unordered collection is canonicalized for submission identity.
// Presence, duplicates and every other request byte retain the existing rules.
func staffingDigest(r *http.Request, raw []byte, k s.Kind) []byte {
	if k != s.Job {
		return digest(r, raw)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return digest(r, raw)
	}
	normalize := func(obj map[string]json.RawMessage) {
		if value, ok := obj["gradeIds"]; ok {
			var ids []string
			if json.Unmarshal(value, &ids) == nil && ids != nil {
				for i := range ids {
					ids[i] = strings.ToLower(ids[i])
				}
				slices.Sort(ids)
				obj["gradeIds"], _ = json.Marshal(ids)
			}
		}
	}
	normalize(object)
	if value, ok := object["changes"]; ok {
		var changes map[string]json.RawMessage
		if json.Unmarshal(value, &changes) == nil {
			normalize(changes)
			object["changes"], _ = json.Marshal(changes)
		}
	}
	raw, _ = json.Marshal(object)
	return digest(r, raw)
}
