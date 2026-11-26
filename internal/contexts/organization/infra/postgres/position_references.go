package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"context"
	"errors"
	"strings"
	"time"

	department "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

// Search dated labels under the department's current BU scope, including when the
// requested day precedes a transfer. Page only after applying X03 field predicates.
func (s *Store) SearchQuotaDepartments(ctx context.Context, c foundation.Credentials, enterprise, day, needle string, now time.Time, offset, limit int) ([]org.DepartmentChoice, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("department lookup requires current transaction")
	}
	guard, err := s.Guard(ctx, tx, c)
	if err != nil {
		return nil, err
	}
	planner, ok := guard.(access.ReadPlanner)
	if !ok {
		return nil, errors.New("department read planner unavailable")
	}
	current, err := planner.PlanReads(ctx, enterprise, now, access.DepartmentRead)
	if err != nil {
		return nil, err
	}
	historical := current
	if day != current.Day {
		historical, err = planner.PlanReads(ctx, enterprise, now, access.DepartmentHistory)
		if err != nil {
			return nil, err
		}
	}
	// Enterprise-wide field grants do not depend on the current BU. In this
	// common case the dated label alone suffices for candidate selection.
	unrestricted := current.Fields["name"].All && historical.Fields["name"].All &&
		(current.Fields["code"].All && historical.Fields["code"].All || needle == "")
	rows, err := tx.Query(ctx, `SELECT h.department_id::text
 FROM organization.department h
 JOIN organization.department_read_segment d ON (d.instance_id,d.object_id,d.source_id)=(h.instance_id,h.department_id,h.head_source_id) AND d.span @> $3::date
 LEFT JOIN organization.department_read_segment n ON NOT $12 AND (n.instance_id,n.object_id,n.source_id)=(h.instance_id,h.department_id,h.head_source_id) AND n.span @> $4::date
 WHERE h.instance_id=$1 AND h.enterprise_id=$2 AND d.q_state='active'
 AND ($12 OR (n.object_id IS NOT NULL AND organization.q131_set($6::jsonb,n.q_bu,h.department_id) AND organization.q131_set($7::jsonb,n.q_bu,h.department_id)))
 AND ($5='' OR strpos(lower(d.q_name),$5)>0 OR (strpos(lower(h.code),$5)>0
 AND ($12 OR (organization.q131_set($8::jsonb,n.q_bu,h.department_id) AND organization.q131_set($9::jsonb,n.q_bu,h.department_id)))))
 ORDER BY d.q_name COLLATE "C",h.department_id OFFSET $10 LIMIT $11`,
		s.Instance, enterprise, day, current.Day, needle, current.Fields["name"], historical.Fields["name"], current.Fields["code"], historical.Fields["code"], offset, limit, unrestricted)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	refs, err := s.PositionReferences(ctx, enterprise, "departmentId", ids, false, now)
	if err != nil {
		return nil, err
	}
	byID := map[string]org.DepartmentChoice{}
	for _, r := range refs {
		scope := r.DepartmentScope
		if scope == nil {
			return nil, errors.New("missing department scope")
		}
		bu := scope.ManagementBUID
		if !current.Fields["name"].Contains(bu, r.ID) || !historical.Fields["name"].Contains(bu, r.ID) {
			return nil, errors.New("department lookup scope mismatch")
		}
		choice := org.DepartmentChoice{ID: r.ID}
		if current.Fields["code"].Contains(bu, r.ID) && historical.Fields["code"].Contains(bu, r.ID) {
			choice.Code = r.Code
		}
		for _, record := range r.Records {
			if record.Active && record.Start <= day && (record.End == "" || day < record.End) {
				choice.Name = record.Name
				break
			}
		}
		if choice.Name == "" || (!strings.Contains(strings.ToLower(choice.Name), needle) && !strings.Contains(strings.ToLower(choice.Code), needle)) {
			return nil, errors.New("department lookup fact mismatch")
		}
		byID[r.ID] = choice
	}
	choices := make([]org.DepartmentChoice, 0, len(ids))
	for _, id := range ids {
		choices = append(choices, byID[id])
	}
	return choices, nil
}

func (s *Store) PositionDepartments(ctx context.Context, cred foundation.Credentials, enterprise string, now time.Time, anchor string, descendants bool, ids []string) ([]string, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("department range requires guarded transaction")
	}
	guard, err := s.Guard(ctx, tx, cred)
	if err != nil {
		return nil, err
	}
	planner, ok := guard.(access.ReadPlanner)
	if !ok {
		return nil, errors.New("department read planner unavailable")
	}
	plan, err := planner.PlanReads(ctx, enterprise, now, access.DepartmentRead)
	if err != nil {
		return nil, err
	}
	input := org.DepartmentListInput{ScopeDepartmentID: anchor, IncludeDescendants: &descendants, Status: "all", Sort: "code", Page: 1, PageSize: 20}
	// Validate even an empty set against the anchor. Restrict each capture well
	// below the identity-read ceiling, so a large enterprise cannot lose matches.
	out := []string{}
	for {
		n := min(len(ids), 16000)
		candidates := append([]string{}, ids[:n]...)
		capture, err := history.CaptureDepartmentQuery(ctx, tx, guard, department.QueryRequest{Plan: plan, Input: input, OnlyIDs: candidates, Extent: department.QueryIdentities})
		if err != nil {
			return nil, err
		}
		out = append(out, capture.PageIDs...)
		ids = ids[n:]
		if len(ids) == 0 {
			return out, nil
		}
	}
}

func (s *Store) PositionCandidateIDs(ctx context.Context, enterprise, field, date, needle string, offset, limit int) ([]string, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("position candidates require current transaction")
	}
	if field == "buId" {
		// BU segment names live in the source dictionary. Match their decoded
		// values only after applying the caller's field and object scope.
		rows, err := tx.Query(ctx, `SELECT h.bu_id::text FROM organization.business_unit h
            JOIN organization.bu_read_segment r ON (r.instance_id,r.object_id,r.source_id)=(h.instance_id,h.bu_id,h.head_source_id)
            WHERE h.instance_id=$1 AND h.enterprise_id=$2 AND r.span @> $3::date
            ORDER BY h.bu_id OFFSET $4 LIMIT $5`, s.Instance, enterprise, date, offset, limit)
		if err != nil {
			return nil, err
		}
		return pgx.CollectRows(rows, pgx.RowTo[string])
	}
	var query string
	switch field {
	case "departmentId":
		query = `SELECT r.object_id::text FROM organization.department_read_segment r JOIN organization.department h ON (h.instance_id,h.department_id,h.head_source_id)=(r.instance_id,r.object_id,r.source_id) WHERE h.instance_id=$1 AND h.enterprise_id=$2 AND r.span @> $3::date AND r.q_state='active' AND (strpos(lower(r.q_name),$4)>0 OR strpos(h.normalized_code,$4)>0) ORDER BY r.q_name COLLATE "C",r.object_id OFFSET $5 LIMIT $6`
	case "locationId":
		query = `SELECT r.object_id::text FROM organization.location_read r JOIN organization.location h ON (h.instance_id,h.enterprise_id,h.location_id)=(r.instance_id,r.enterprise_id,r.object_id) WHERE r.instance_id=$1 AND r.enterprise_id=$2 AND r.valid_period @> $3::date AND r.state='active' AND (strpos(lower(r.name),$4)>0 OR strpos(h.normalized_code,$4)>0) ORDER BY r.name COLLATE "C",r.object_id OFFSET $5 LIMIT $6`
	default:
		return nil, errors.New("unknown organization reference")
	}
	rows, err := tx.Query(ctx, query, s.Instance, enterprise, date, needle, offset, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *Store) PositionReferences(ctx context.Context, enterprise, field string, ids []string, lock bool, now time.Time) ([]org.PositionReference, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("position reference requires current transaction")
	}
	table, key := "department", "department_id"
	switch field {
	case "departmentId":
	case "buId":
		table, key = "business_unit", "bu_id"
	case "locationId":
		table, key = "location", "location_id"
	default:
		return nil, errors.New("unknown organization reference")
	}
	query := "SELECT " + key + "::text,head_source_id::text FROM organization." + table + " WHERE instance_id=$1 AND enterprise_id=$2 AND " + key + "=ANY($3::uuid[]) ORDER BY " + key
	if lock {
		query += " FOR SHARE"
	}
	rows, err := tx.Query(ctx, query, s.Instance, enterprise, ids)
	if err != nil {
		return nil, err
	}
	wanted := map[string]string{}
	selected := []string{}
	for rows.Next() {
		var id, source string
		if err = rows.Scan(&id, &source); err != nil {
			rows.Close()
			return nil, err
		}
		wanted[id] = source
		selected = append(selected, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	result := []org.PositionReference{}
	if len(selected) == 0 {
		return result, nil
	}
	s.historyCatalog.Enter()
	defer s.historyCatalog.Leave()
	unit := &unit{instance: s.Instance, tx: tx, catalog: &s.historyCatalog}
	if field == "locationId" {
		locations, err := (&locationUnit{unit: unit, enterprise: org.Enterprise{ID: enterprise}}).locationCurrent(ctx, selected)
		if err != nil {
			return nil, err
		}
		for _, location := range locations {
			t := location.Timeline
			ref := org.PositionReference{ID: t.ID, Source: t.Source, Code: t.Code}
			for _, record := range t.Records {
				ref.Records = append(ref.Records, org.PositionReferenceRecord{ID: record.ID, Revision: record.Revision, Start: record.Date, End: record.End, Name: record.Data.Name, Active: record.State == "active"})
			}
			result = append(result, ref)
		}
	} else if field == "buId" {
		facts, err := unit.currentFacts(ctx, history.BusinessUnit, wanted)
		if err != nil {
			return nil, err
		}
		for _, state := range facts.BUs {
			t := state.Timeline
			ref := org.PositionReference{ID: t.BUID, Source: t.Source, Code: t.Code}
			for _, record := range t.Records {
				ref.Records = append(ref.Records, org.PositionReferenceRecord{ID: record.ID, Revision: record.Revision, Start: record.Date, End: record.End, Name: record.Data.Name, Active: record.State == "active"})
			}
			result = append(result, ref)
		}
	} else {
		facts, err := unit.currentFacts(ctx, history.Department, wanted)
		if err != nil {
			return nil, err
		}
		for _, state := range facts.Departments {
			t := state.Timeline
			if t.Enterprise != enterprise {
				return nil, errors.New("department scope mismatch")
			}
			scope := department.PositionScope(t, now)
			ref := org.PositionReference{ID: t.ID, Source: t.Source, Code: t.Code, DepartmentScope: &scope}
			for _, record := range t.Records {
				ref.Records = append(ref.Records, org.PositionReferenceRecord{ID: record.ID, Revision: record.Revision, Start: record.Date, End: record.End, Name: record.Data.Name, Parent: record.Data.Parent, Active: record.State == "active"})
			}
			result = append(result, ref)
		}
	}
	if len(result) != len(selected) {
		return nil, errors.New("incomplete organization position references")
	}
	return result, nil
}

func checkPositionUses[T any](ctx context.Context, unit *unit, enterprise, field, id string, before, after []dated.Record[T]) error {
	if len(before) == 0 {
		return nil
	}
	if field == "departmentId" && len(after) > 0 && after[0].Date > before[0].Date {
		if unit.staffing == nil {
			return errors.New("department quota guard unavailable")
		}
		floor, e := unit.staffing.DepartmentQuotaStart(dbtx.With(ctx, unit.tx), enterprise, id)
		if e != nil {
			return e
		}
		if floor != "" && after[0].Date > floor {
			return foundation.Fail(409, "M02_REFERENCE_CONFLICT", "部门首日不能晚于已有部门核编，请先维护额度")
		}
	}
	lost, err := dated.RemovedCoverage(before, after)
	if err != nil || len(lost) == 0 {
		return err
	}
	if unit.staffing == nil {
		return errors.New("position reference guard unavailable")
	}
	uses, err := unit.staffing.PositionUses(dbtx.With(ctx, unit.tx), enterprise, field, id)
	if err != nil {
		return err
	}
	first := ""
	for _, removed := range lost {
		for _, use := range uses {
			start := max(removed.Start, use.Start)
			if (removed.End == "" || start < removed.End) && (use.End == "" || start < use.End) && (first == "" || start < first) {
				first = start
			}
		}
	}
	if first != "" {
		return &foundation.Fault{Status: 409, Code: "M02_REFERENCE_CONFLICT", Field: "effectiveDate", Message: "自 " + first + " 起仍有岗位或任职使用此资料，请先核对使用期间"}
	}
	return nil
}
