package history

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

//go:embed department_parent_cursor.sql
var DepartmentParentCursorSQL string

func WalkDepartmentParents(ctx context.Context, tx pgx.Tx, guard access.Guard, q app.ParentRequest, visit func(app.ParentFrame) (bool, error)) (err error) {
	planner, ok := guard.(access.ReadPlanner)
	p := q.Plan
	principal := guard.Principal()
	if !ok || p.Origin != planner || p.Instance != principal.InstanceID || p.Subject != principal.ID || p.Session != principal.SessionID || p.Action != access.DepartmentReference || !foundation.ValidID(p.Enterprise) || p.At.IsZero() || p.Day != p.At.In(foundation.Shanghai).Format("2006-01-02") {
		return queryUnavailable()
	}
	parents, bus := []string{}, []string{}
	for _, r := range q.Base.Records {
		if r.Data.Parent != "" {
			parents = append(parents, r.Data.Parent)
		}
		bus = append(bus, r.Data.BU)
	}
	input, _ := json.Marshal(map[string]any{"target": q.Target, "source": q.Source, "bu": q.BU, "selected": q.Selected, "search": strings.ToLower(strings.TrimSpace(q.Search)), "windows": q.Windows, "parents": parents, "bus": bus, "afterCode": q.AfterCode, "afterId": q.AfterID})
	plan, _ := json.Marshal(queryPlan(p))
	if _, err = tx.Exec(ctx, "SET LOCAL plan_cache_mode='force_custom_plan'"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DECLARE parent_facts133 NO SCROLL CURSOR FOR "+DepartmentParentCursorSQL, p.Instance, p.Enterprise, p.Day, plan, input); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			return
		} // Rollback closes an errored/canceled cursor.
		_, closeErr := tx.Exec(ctx, "CLOSE parent_facts133")
		if err == nil {
			err = closeErr
		}
	}()
	common := false
	for {
		rows, e := tx.Query(ctx, "FETCH FORWARD 64 FROM parent_facts133")
		if e != nil {
			return e
		}
		type rawFrame struct {
			kind     int
			code, id string
			material []byte
		}
		batch := make([]rawFrame, 0, 64)
		for rows.Next() {
			var raw rawFrame
			if e = rows.Scan(&raw.kind, &raw.code, &raw.id, &raw.material); e != nil {
				rows.Close()
				return e
			}
			batch = append(batch, raw)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		fetched, transferred := len(batch), 0
		for _, row := range batch {
			transferred += len(row.material)
		}
		for _, raw := range batch {
			if e = ctx.Err(); e != nil {
				return e
			}
			if raw.kind == 0 {
				if common {
					return queryUnavailable()
				}
				common = true
			} else if raw.kind != 1 || !common {
				return queryUnavailable()
			}
			capture, e := decodeParentFrame(ctx, p.Enterprise, raw.material)
			if e != nil {
				return e
			}
			capture.Stats.FetchedFrames, capture.Stats.TransferredMaterialBytes = fetched, transferred
			fetched, transferred = 0, 0
			more, e := visit(app.ParentFrame{Common: raw.kind == 0, ID: raw.id, Code: raw.code, Capture: capture})
			if e != nil {
				return e
			}
			if !more {
				return nil
			}
		}
		if len(batch) < 64 {
			if !common {
				return queryUnavailable()
			}
			return nil
		}
	}
}

func decodeParentFrame(ctx context.Context, enterprise string, raw []byte) (out app.ParentCapture, err error) {
	var payload struct {
		Missing, Conflict, ContextMissing bool
		Departments                       []struct {
			Purpose int
			Fields  []*string
		}
		BUs []struct {
			Fields   []*string
			IDs      []string
			Material capturedMaterial
		}
	}
	if json.Unmarshal(raw, &payload) != nil || payload.Missing {
		return out, queryUnavailable()
	}
	if payload.ContextMissing {
		return out, foundation.Fail(404, "M02_OBJECT_NOT_FOUND", "引用对象不存在或当前不可见")
	}
	if payload.Conflict {
		return out, foundation.Fail(409, "M02_SOURCE_CONFLICT", "资料已更新，请核对最新内容后保存")
	}
	facts, labels := parentMaterialSet{}, parentMaterialSet{}
	for _, item := range payload.Departments {
		fields := make([]pgtype.Text, len(item.Fields))
		for i, v := range item.Fields {
			if v != nil {
				fields[i] = pgtype.Text{String: *v, Valid: true}
			}
		}
		t, e := parentDepartment(enterprise, fields)
		if e != nil {
			return out, e
		}
		s := d.Snapshot{Departments: []d.Timeline{t}}
		switch item.Purpose {
		case 1:
			labels.add(s)
		case 3:
			facts.add(s)
		case 4:
			facts.add(s)
			labels.add(s)
		default:
			return out, queryUnavailable()
		}
	}
	for _, item := range payload.BUs {
		if len(item.Fields) != 5 {
			return out, queryUnavailable()
		}
		for _, v := range item.Fields {
			if v == nil {
				return out, queryUnavailable()
			}
		}
		values, data, e := queryValues(item.IDs, item.Material)
		if e != nil {
			return out, errors.Join(queryUnavailable(), e)
		}
		decoded, e := DecodeBURead(ctx, values, data)
		if e != nil {
			return out, errors.Join(queryUnavailable(), e)
		}
		b := decoded.Timeline
		if b.ID != *item.Fields[0] || b.Source != *item.Fields[2] || b.Source != *item.Fields[4] || b.First != *item.Fields[3] || len(b.Records) == 0 {
			return out, queryUnavailable()
		}
		b.Code = *item.Fields[1]
		s := d.Snapshot{BusinessUnits: []d.BusinessUnit{b}}
		facts.add(s)
		labels.add(s)
	}
	out.Facts, out.Labels = facts.snapshot(), labels.snapshot()
	out.Stats.MaterialBytes = len(raw)
	out.Stats.DecodedBUs = len(payload.BUs)
	out.Stats.Objects = int64(len(out.Facts.Departments))
	out.Stats.Segments = int64(len(payload.Departments))
	return out, nil
}
