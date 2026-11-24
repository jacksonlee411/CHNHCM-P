package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	archive "chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func sealedOperator(id string, name, login, kind *string) org.OperatorSnapshot {
	v := org.OperatorSnapshot{SubjectID: id, DisplayName: name, Login: login, Kind: "unknown", IdentityEvidence: "subject_id_only"}
	if login != nil && kind != nil {
		v.Kind = *kind
		v.IdentityEvidence = "captured"
	}
	return v
}

func (u *unit) operationSummary(ctx context.Context, bu, source string, selected *org.Record) (*org.OperationSummary, error) {
	v, e := u.readBU(ctx, bu, source)
	if e != nil {
		return nil, e
	}
	for _, summary := range v.State.Summaries {
		if summary.RecordID == selected.ID && summary.Revision == selected.Revision {
			return &summary, nil
		}
	}
	for _, r := range v.Operation.Revisions {
		if r.Deleted && r.Record.ID == selected.ID && r.Record.Revision == selected.Revision && r.Before != nil {
			previous, e := u.readBU(ctx, bu, v.Operation.Source.Previous)
			if e != nil {
				return nil, e
			}
			for _, summary := range previous.State.Summaries {
				if summary.RecordID == selected.ID && summary.Revision == r.Previous {
					return &summary, nil
				}
			}
		}
	}
	return nil, resultUnavailable()
}

type logRevision struct {
	ID          string  `json:"record_id"`
	Revision    string  `json:"revision_id"`
	Date        string  `json:"effective_date"`
	End         string  `json:"end_date"`
	Name        string  `json:"name"`
	NameEn      *string `json:"name_en"`
	State       string  `json:"state"`
	Type        string  `json:"operation_type"`
	Reason      string  `json:"operation_reason"`
	TypeLabel   string  `json:"type_label"`
	ReasonLabel string  `json:"reason_label"`
	Deleted     bool    `json:"deleted"`
}

func decodeLogRevision(raw []byte) (*domain.Record, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, false, nil
	}
	var r logRevision
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, false, err
	}
	return &domain.Record{ID: r.ID, Revision: r.Revision, Date: r.Date, End: r.End, State: r.State, Type: r.Type, Reason: r.Reason, TypeLabel: r.TypeLabel, ReasonLabel: r.ReasonLabel, Data: domain.BUData{Name: r.Name, NameEn: r.NameEn}}, r.Deleted, nil
}

func (u *unit) Changes(ctx context.Context, in org.ChangeQuery, ent org.Enterprise) (org.Changes, error) {
	out := org.Changes{Items: []org.Change{}, Page: in.Page, PageSize: in.PageSize}
	if in.ContextSourceVersion != "" {
		out.ContextSourceVersion = &in.ContextSourceVersion
	}
	read, err := u.guard.Decide(ctx, access.Intent{Action: "bu.read", EnterpriseID: ent.ID, BUID: in.ID, Fields: []string{}, Kind: "changes"})
	if err != nil {
		return out, err
	}
	if !read.Allowed {
		return out, application.RejectRead(foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "业务单元不存在或当前不可见"))
	}
	history, err := u.guard.Decide(ctx, access.Intent{Action: "bu.history.read", EnterpriseID: ent.ID, BUID: in.ID, Fields: []string{}, Kind: "changes"})
	if err != nil {
		return out, err
	}
	if !history.Allowed {
		return out, application.RejectRead(foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许读取日志"))
	}
	fields := slices.DeleteFunc(slices.Clone(read.ReadFields), func(f string) bool { return !slices.Contains(history.ReadFields, f) })
	out.ReadFields = fields
	// The cached detail may use ordinary read rights, while the log always uses
	// their intersection with history rights. Keep these scopes explicit.
	out.DetailReadFields = read.ReadFields
	if in.ContextSourceVersion != "" {
		out.DetailReadFields = fields
	}
	if f := in.Filter; f != nil {
		needed := []string{}
		if f.Field == "effectiveDate" {
			needed = []string{"effectiveDate"}
		} else if f.Field == "changedField" {
			needed = strings.Split(f.Value, ",")
		}
		for _, field := range needed {
			if !application.ChangeFieldAllowed(field, fields) {
				return out, application.RejectRead(foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许筛选该字段"))
			}
		}
	}
	var head string
	err = u.tx.QueryRow(ctx, `SELECT head_source_id::text FROM organization.business_unit WHERE instance_id=$1 AND bu_id=$2`, u.instance, in.ID).Scan(&head)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.RejectRead(foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "业务单元不存在或当前不可见"))
	}
	if err != nil {
		return out, err
	}
	sourceNumber := func(id string) (int64, error) {
		v, e := u.readBU(ctx, in.ID, id)
		if errors.Is(e, pgx.ErrNoRows) {
			e = application.RejectRead(foundation.Fail(404, "M02_SOURCE_NOT_FOUND", "无法查看该历史资料"))
		}
		if e != nil {
			return 0, e
		}
		return v.State.Timeline.SourceNo, nil
	}
	contextSource := in.ContextSourceVersion
	if contextSource == "" {
		contextSource = head
	}
	ceiling, err := sourceNumber(contextSource)
	if err != nil {
		return out, err
	}
	upper := in.ThroughSourceVersion
	if upper == "" {
		upper = contextSource
	}
	through, err := sourceNumber(upper)
	if err != nil {
		return out, err
	}
	if in.ContextSourceVersion != "" && through > ceiling {
		return out, application.RejectRead(foundation.Fail(400, "M02_INVALID_LOG_RANGE", "日志查询范围无效，请重置日志条件"))
	}
	out.ThroughSourceVersion = upper
	// Source numbers are contiguous and every source has one operation. For an
	// unfiltered page, verify the original operations but project only its range.
	projectedPage := in.Filter == nil
	page, pageTotal := in.Page, int(through)
	low, high := int64(1), through
	if projectedPage {
		page = min(page, max(1, (pageTotal+in.PageSize-1)/in.PageSize))
		high = through - int64((page-1)*in.PageSize)
		low = max(int64(1), high-int64(in.PageSize)+1)
	}
	matching := []org.Change{}
	var seen int64
	err = u.eachOperation(ctx, archive.BusinessUnit, in.ID, through, func(c archive.Chunk, slot int, raw []byte) error {
		op, e := archive.DecodeBUOperation(ctx, c.Values, raw)
		if e != nil {
			return e
		}
		seen++
		if op.Source.Number != seen || op.Source.Number != c.First+int64(slot) || op.Source.ID != c.ID(c.Sources[slot]) || op.Source.Subject != c.ID(c.Subjects[slot]) || op.Source.Submission != c.ID(c.Submissions[slot]) {
			return resultUnavailable()
		}
		if projectedPage && (op.Source.Number < low || op.Source.Number > high) {
			return nil
		}
		current := org.Change{SourceVersion: op.Source.ID, PreviousSourceVersion: depPtr(op.Source.Previous), RecordedAt: op.Source.RecordedAt.UTC().Format(time.RFC3339Nano), Operator: op.Source.Operator, Action: op.Source.Action, Items: []org.ChangeItem{}}
		revisions := slices.Clone(op.Revisions)
		slices.SortFunc(revisions, func(a, b model.BURevision) int { return strings.Compare(a.Record.ID, b.Record.ID) })
		for _, r := range revisions {
			after := &r.Record
			if r.Deleted {
				after = nil
			}
			role := "derived"
			if r.Previous == "" || r.Record.ID == op.Terminal.RecordID {
				role = "direct"
			}
			for _, it := range op.Terminal.ItemOutcomes {
				if it.Outcome == "accepted" && it.RecordID == r.Record.ID {
					role = "direct"
				}
			}
			current.Items = append(current.Items, application.BuildChangeItem(r.Record.ID, role, op.Source.Previous, op.Source.ID, r.Before, after, fields))
		}
		if application.ChangeMatches(current, in.Filter, op.Source.RecordedAt.In(foundation.Shanghai).Format("2006-01-02")) {
			matching = append(matching, current)
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	if seen != through {
		return out, resultUnavailable()
	}
	slices.Reverse(matching)
	if projectedPage {
		out.Total = pageTotal
		out.Page = page
		out.Items = matching
		return out, nil
	}
	out.Total = len(matching)
	pages := (out.Total + in.PageSize - 1) / in.PageSize
	if pages < 1 {
		pages = 1
	}
	if out.Page > pages {
		out.Page = pages
	}
	start := (out.Page - 1) * in.PageSize
	end := min(start+in.PageSize, out.Total)
	out.Items = matching[start:end]
	return out, nil
}
