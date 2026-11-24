package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgconn"
)

func digest(r *http.Request, raw []byte) []byte {
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	_ = d.Decode(&v)

	b, _ := json.Marshal([]any{2, r.Method, r.URL.Path, v})
	h := sha256.Sum256(b)
	return h[:]
}
func fields(raw []byte) []string {
	var v map[string]json.RawMessage
	_ = json.Unmarshal(raw, &v)
	out := []string{}
	for f := range v {
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}
func commandResponse(w http.ResponseWriter, status int, v org.Result, e error) {
	if v.Outcome == "rejected" {
		code := "M02_INVALID_INPUT"
		if v.ErrorCode != nil {
			code = *v.ErrorCode
		}
		status = 400
		if code == "ACCESS_DENIED" {
			status = 403
		}
		var f *foundation.Fault
		if errors.As(e, &f) {
			status = f.Status
		} else if slices.Contains([]string{"M02_SOURCE_CONFLICT", "M02_REVISION_CONFLICT", "M02_DATE_CONFLICT", "M02_CODE_CONFLICT", "M02_TIMELINE_INVALID", "M02_FIRST_RECORD_PROTECTED", "M02_ACTION_NOT_ALLOWED", "M02_RECORD_NOT_FOUND", "M02_PAIR_NOT_FOUND"}, code) {
			status = 409
		}
		field := ""
		if v.FailureDetails != nil && len(v.FailureDetails.Issues) > 0 && len(v.FailureDetails.Issues[0].Locations) > 0 {
			field = v.FailureDetails.Issues[0].Locations[0].FieldPath
		}
		message := "原请求已拒绝，请核对输入和当前资料后重新办理"
		if errors.As(e, &f) {
			message = f.Message
		}
		body := map[string]any{"code": code, "message": message, "field": field, "submissionId": v.SubmissionID, "outcome": "rejected", "failureDetails": v.FailureDetails}
		if v.CurrentReadFields != nil {
			body["readFields"] = v.CurrentReadFields
		}
		writeJSON(w, status, body)
		return
	}
	if e != nil {
		writeScopedError(w, e, v.CurrentReadFields)
		return
	}
	if v.Outcome == "no_change" {
		status = 200
	}
	respond(w, status, v, e)
}

// The response carries the current object scope only after an authorized read.
// This sidecar never changes the immutable terminal or exposes dependency errors.
func writeScopedError(w http.ResponseWriter, err error, readFields []string) {
	var f *foundation.Fault
	if readFields != nil && errors.As(err, &f) && f.Status >= 400 && f.Status < 500 {
		writeJSON(w, f.Status, map[string]any{"code": f.Code, "message": f.Message, "field": f.Field, "readFields": readFields})
		return
	}
	writeError(w, err)
}
func (a *API) add(w http.ResponseWriter, r *http.Request) {
	var in org.AddInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	v, e := a.Runtime.Organization.Maintain(r.Context(), credentials(r), org.MaintainCommand{Action: "ADD", BUID: r.PathValue("id"), SubmissionID: r.Header.Get("Idempotency-Key"), Add: in, Digest: digest(r, raw), Fields: append([]string{"effectiveDate", "operationType", "operationReason"}, fieldsMust(in.Changes)...)})
	commandResponse(w, 201, v, e)
}
func fieldsMust(changes org.ChangesInput) []string {
	out := []string{}
	for f := range changes {
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}

func (a *API) deactivate(w http.ResponseWriter, r *http.Request) {
	var in org.DeactivateInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	a.lifecycle(w, r, "DEACTIVATE", org.AddInput{AnchorRecordID: in.AnchorRecordID, ExpectedSourceVersion: in.ExpectedSourceVersion, EffectiveDate: in.EffectiveDate, OperationType: in.OperationType, OperationReason: in.OperationReason}, raw)
}
func (a *API) reactivate(w http.ResponseWriter, r *http.Request) {
	var in org.ReactivateInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	a.lifecycle(w, r, "REACTIVATE", org.AddInput{AnchorRecordID: in.AnchorRecordID, ExpectedSourceVersion: in.ExpectedSourceVersion, EffectiveDate: in.EffectiveDate, OperationType: in.OperationType, OperationReason: in.OperationReason, Changes: in.Changes}, raw)
}
func (a *API) lifecycle(w http.ResponseWriter, r *http.Request, action string, in org.AddInput, raw []byte) {
	v, e := a.Runtime.Organization.Maintain(r.Context(), credentials(r), org.MaintainCommand{Action: action, BUID: r.PathValue("id"), SubmissionID: r.Header.Get("Idempotency-Key"), Add: in, Digest: digest(r, raw), Fields: append([]string{"effectiveDate", "operationType", "operationReason"}, fieldsMust(in.Changes)...)})
	commandResponse(w, 201, v, e)
}
func (a *API) correct(w http.ResponseWriter, r *http.Request) {
	var in org.CorrectInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	v, e := a.Runtime.Organization.Maintain(r.Context(), credentials(r), org.MaintainCommand{Action: "CORRECT", BUID: r.PathValue("id"), RecordID: r.PathValue("recordId"), SubmissionID: r.Header.Get("Idempotency-Key"), Correct: in, Digest: digest(r, raw), Fields: org.CorrectionFields(in)})
	commandResponse(w, 200, v, e)
}
func (a *API) deleteRecord(w http.ResponseWriter, r *http.Request) {
	var in org.DeleteInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	v, e := a.Runtime.Organization.Maintain(r.Context(), credentials(r), org.MaintainCommand{Action: "DELETE", BUID: r.PathValue("id"), RecordID: r.PathValue("recordId"), SubmissionID: r.Header.Get("Idempotency-Key"), Delete: in, Digest: digest(r, raw), Fields: []string{}})
	commandResponse(w, 200, v, e)
}
func (a *API) redateRecord(w http.ResponseWriter, r *http.Request) {
	var in org.RedateInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	v, e := a.Runtime.Organization.Maintain(r.Context(), credentials(r), org.MaintainCommand{Action: "REDATE", BUID: r.PathValue("id"), RecordID: r.PathValue("recordId"), SubmissionID: r.Header.Get("Idempotency-Key"), Redate: in, Digest: digest(r, raw), Fields: []string{"effectiveDate"}})
	commandResponse(w, 200, v, e)
}
func (a *API) preview(w http.ResponseWriter, r *http.Request) {
	var in org.PreviewInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	for _, field := range fields(raw) {
		if field == "anchorRecordId" && !slices.Contains([]string{"ADD", "DEACTIVATE", "REACTIVATE"}, in.Action) {
			writeError(w, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "本动作不接受新增位置"))
			return
		}
		if field == "newEffectiveDate" && in.Action != "REDATE" && in.Action != "CORRECT" {
			writeError(w, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "此动作不接受新日期"))
			return
		}
		if in.Action == "REDATE" && !slices.Contains([]string{"action", "expectedSourceVersion", "recordId", "expectedRecordRevision", "newEffectiveDate"}, field) {
			writeError(w, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "改日内容无效，请重新打开表单"))
			return
		}
	}
	if in.Action == "DELETE" {
		for _, field := range fields(raw) {
			if !slices.Contains([]string{"action", "expectedSourceVersion", "recordId", "expectedRecordRevision"}, field) {
				writeError(w, foundation.Fail(400, "M02_FORBIDDEN_INPUT", "无法读取删除影响，请重新打开表单"))
				return
			}
		}
	}
	v, e := a.Runtime.Organization.Preview(r.Context(), credentials(r), r.PathValue("id"), in)
	e = previewExecutionError(e)
	if e != nil {
		writeScopedError(w, e, v.ReadFields)
		return
	}
	respond(w, 200, v, e)
}

func previewExecutionError(err error) error {
	var pg *pgconn.PgError
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
		(errors.As(err, &pg) && (pg.Code == "57014" || pg.Code == "55P03")) {
		return foundation.Fail(503, "M02_EXECUTION_LIMIT", "操作影响暂时无法加载，请重试")
	}
	return err
}

func (a *API) insertLifecyclePair(w http.ResponseWriter, r *http.Request) {
	var in org.InsertLifecyclePairInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	v, e := a.Runtime.Organization.Maintain(r.Context(), credentials(r), org.MaintainCommand{Action: "INSERT_LIFECYCLE_PAIR", BUID: r.PathValue("id"), SubmissionID: r.Header.Get("Idempotency-Key"), InsertPair: in, Digest: digest(r, raw)})
	commandResponse(w, 200, v, e)
}
func (a *API) deleteLifecyclePair(w http.ResponseWriter, r *http.Request) {
	var in org.DeleteLifecyclePairInput
	raw, e := decode(r, &in)
	if e != nil {
		writeError(w, e)
		return
	}
	v, e := a.Runtime.Organization.Maintain(r.Context(), credentials(r), org.MaintainCommand{Action: "DELETE_LIFECYCLE_PAIR", BUID: r.PathValue("id"), SubmissionID: r.Header.Get("Idempotency-Key"), DeletePair: in, Digest: digest(r, raw)})
	commandResponse(w, 200, v, e)
}
