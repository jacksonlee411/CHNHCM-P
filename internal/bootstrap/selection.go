package bootstrap

import (
	"bytes"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

func decodeIdentityDraft(w http.ResponseWriter, r *http.Request, out any, query bool) bool {
	if _, err := queryValues(r); err != nil {
		writeError(w, err)
		return false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 65537))
	if err != nil {
		writeError(w, foundation.Fail(400, "M02_INVALID_SELECTION", "选择请求不完整"))
		return false
	}
	if len(data) > 65536 {
		writeError(w, foundation.Fail(413, "M02_SELECTION_BATCH_LIMIT", "选择请求过大，请减少本次选择项"))
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	if _, err = decode(r, out); err != nil {
		writeError(w, err)
		return false
	}
	if query {
		var obj map[string]json.RawMessage
		_ = json.Unmarshal(data, &obj)
		if err = selectionPredicateShape(out, obj["query"]); err != nil {
			writeError(w, err)
			return false
		}
	}
	return true
}

// BU sends one complete version-bound set. Its transport envelope must admit
// sparse selections beyond 64 KiB without changing them into all-matching.
const buSelectionBodyLimit = 32 << 20

func decodeSelection(w http.ResponseWriter, r *http.Request, out any) bool {
	if _, e := queryValues(r); e != nil {
		writeError(w, e)
		return false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, buSelectionBodyLimit+1))
	if err != nil {
		writeError(w, foundation.Fail(400, "M02_INVALID_SELECTION", "选择请求不完整"))
		return false
	}
	if len(data) > buSelectionBodyLimit {
		writeError(w, foundation.Fail(413, "M02_SELECTION_BATCH_LIMIT", "选择请求过大，请减少本次选择项"))
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	if _, err = decodeWithLimit(r, out, buSelectionBodyLimit); err != nil {
		writeError(w, err)
		return false
	}
	var obj map[string]json.RawMessage
	_ = json.Unmarshal(data, &obj)
	var choice map[string]json.RawMessage
	_ = json.Unmarshal(obj["selection"], &choice)
	var mode string
	_ = json.Unmarshal(choice["mode"], &mode)
	_, ids := choice["ids"]
	_, excluded := choice["excludedIds"]
	if mode == "explicit" && (!ids || excluded) || mode == "allMatching" && (!excluded || ids) || mode != "explicit" && mode != "allMatching" {
		writeError(w, foundation.Fail(400, "M02_INVALID_SELECTION", "选择内容无效，请重新选择"))
		return false
	}
	if err := selectionPredicateShape(out, obj["query"]); err != nil {
		writeError(w, foundation.Fail(400, "M02_INVALID_SELECTION", "列条件不完整"))
		return false
	}
	return true
}

func queryPredicateShape(raw json.RawMessage) error {
	var query map[string]json.RawMessage
	if json.Unmarshal(raw, &query) != nil || query == nil {
		return foundation.Fail(400, "M02_INVALID_INPUT", "查询条件不完整")
	}
	count := 0
	for _, key := range []string{"filterField", "filterOperator", "filterValue"} {
		if v, ok := query[key]; ok {
			count++
			if string(v) == `""` {
				return foundation.Fail(400, "M02_INVALID_INPUT", "列条件不完整")
			}
		}
	}
	if count != 0 && count != 3 {
		return foundation.Fail(400, "M02_INVALID_INPUT", "列条件不完整")
	}
	return nil
}

func (a *API) resolveSelection(w http.ResponseWriter, r *http.Request) {
	var in org.SelectionRequest
	if !decodeSelection(w, r, &in) {
		return
	}
	v, e := a.Runtime.Organization.ResolveSelection(r.Context(), credentials(r), in)
	if e != nil {
		writeSelectionError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (a *API) exportSelection(w http.ResponseWriter, r *http.Request) {
	var in org.ExportRequest
	if !decodeSelection(w, r, &in) {
		return
	}
	v, e := a.Runtime.Organization.ExportSelection(r.Context(), credentials(r), in)
	if e != nil {
		writeSelectionError(w, e)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="chnhcm-business-units-`+v.BusinessAt+`.csv"`)
	w.Header().Set("X-Selected-Count", strconv.Itoa(v.Count))
	w.WriteHeader(200)
	_, _ = w.Write(v.Data)
}

func writeSelectionError(w http.ResponseWriter, err error) {
	var fault *foundation.Fault
	if !errors.As(err, &fault) || fault.Code == "M02_RESULT_PENDING" {
		err = foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "暂时无法处理所选项，请重试")
	}
	writeError(w, err)
}

func selectionPredicateShape(out any, raw json.RawMessage) error {
	switch out.(type) {
	case *org.DepartmentSelectionStartInput, *org.DepartmentSelectionInput, *org.DepartmentExportInput:
		return departmentPredicateShape(raw)
	}
	return queryPredicateShape(raw)
}
func departmentPredicateShape(raw json.RawMessage) error {
	if err := queryPredicateShape(raw); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	id, hasID := fields["scopeDepartmentId"]
	if hasID {
		var value string
		if json.Unmarshal(id, &value) != nil || !foundation.ValidID(value) {
			return foundation.Fail(400, "M02_INVALID_INPUT", "部门范围无效，请重新选择")
		}
	}
	if value, ok := fields["includeDescendants"]; ok && (!hasID || string(value) != "true" && string(value) != "false") {
		return foundation.Fail(400, "M02_INVALID_INPUT", "包含下级参数无效")
	}
	return nil
}
