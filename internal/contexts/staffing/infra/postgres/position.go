package postgres

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"slices"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

func (u *unit) SearchQuotaDepartments(ctx context.Context, c foundation.Credentials, enterprise, day, needle string, offset, limit int) ([]org.DepartmentChoice, error) {
	if u.store.PositionDirectory == nil {
		return nil, errors.New("position directory unavailable")
	}
	return u.store.PositionDirectory.SearchQuotaDepartments(dbtx.With(ctx, u.tx), c, enterprise, day, needle, u.now, offset, limit)
}
func (u *unit) PositionDepartments(ctx context.Context, anchor string, descendants bool, ids []string) ([]string, error) {
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return nil, err
	}
	if u.store.PositionDirectory == nil {
		return nil, errors.New("position directory unavailable")
	}
	return u.store.PositionDirectory.PositionDepartments(dbtx.With(ctx, u.tx), u.credentials, ent.ID, u.now, anchor, descendants, ids)
}

func (u *unit) PositionCandidateIDs(ctx context.Context, field, date, needle string, members []string, offset, limit int) ([]string, error) {
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return nil, err
	}
	if field == "departmentId" || field == "locationId" || field == "buId" {
		if u.store.PositionDirectory == nil {
			return nil, errors.New("position directory unavailable")
		}
		return u.store.PositionDirectory.PositionCandidateIDs(dbtx.With(ctx, u.tx), ent.ID, field, date, needle, offset, limit)
	}
	kind := d.Grade
	if field == "jobId" {
		kind = d.Job
	} else if field != "gradeId" {
		return nil, errors.New("unknown position source")
	}
	query := `SELECT r.object_id::text FROM ` + table(kind) + `_read r JOIN ` + table(kind) + ` h ON (h.instance_id,h.id,h.source_id)=(r.instance_id,r.object_id,r.source_id) WHERE r.instance_id=$1 AND r.enterprise_id=$2 AND daterange(r.start_date,r.end_date,'[)') @> $3::date AND r.active AND ($7::uuid[] IS NULL OR r.object_id=ANY($7)) AND (strpos(lower(r.fact->'data'->>'name'),$4)>0 OR strpos(lower(h.code),$4)>0) ORDER BY (r.fact->'data'->>'name') COLLATE "C",r.object_id OFFSET $5 LIMIT $6`
	rows, err := u.tx.Query(ctx, query, u.store.Instance, ent.ID, date, needle, offset, limit, members)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (u *unit) PositionSources(ctx context.Context, field string, ids []string, locked bool) ([]app.PositionSource, error) {
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return nil, err
	}
	result := []app.PositionSource{}
	if len(ids) == 0 {
		return result, nil
	}
	if field == "departmentId" || field == "locationId" || field == "buId" {
		if u.store.PositionDirectory == nil {
			return nil, errors.New("position directory unavailable")
		}
		query := func(ctx context.Context) error {
			refs, err := u.store.PositionDirectory.PositionReferences(dbtx.With(ctx, u.tx), ent.ID, field, ids, locked, u.now)
			if err != nil {
				return err
			}
			for _, ref := range refs {
				t := d.Timeline{ID: ref.ID, Enterprise: ent.ID, Code: ref.Code, Source: ref.Source}
				for _, record := range ref.Records {
					typ := "CHANGE"
					if !record.Active {
						typ = "DEACTIVATE"
					}
					t.Records = append(t.Records, d.Record{ID: record.ID, Revision: record.Revision, Date: record.Start, End: record.End, Type: typ, Data: map[string]string{"name": record.Name}})
				}
				source := app.PositionSource{Timeline: t}
				if field == "departmentId" {
					scope := ref.DepartmentScope
					if scope == nil {
						return errors.New("missing current department scope")
					}
					source.DepartmentTarget = &access.DepartmentTarget{DepartmentID: ref.ID, EnterpriseID: ent.ID, ScopeFact: access.ScopeFact{EvaluatedAt: u.now, BusinessDate: scope.BusinessDate, Mode: scope.Mode, ManagementBUID: scope.ManagementBUID, CurrentSourceVersion: ref.Source, RecordID: scope.RecordID, Revision: scope.Revision}}
				}
				result = append(result, source)
			}
			return nil
		}
		if locked {
			err = u.withLockBudget(ctx, query)
		} else {
			err = query(ctx)
		}
		return result, err
	}
	kind := d.Job
	if field == "gradeId" {
		kind = d.Grade
	} else if field != "jobId" {
		return nil, errors.New("unknown position source")
	}
	if locked {
		slices.Sort(ids)
		for _, id := range slices.Compact(ids) {
			if err = u.guardLock(ctx, u.tx, true, "staffing."+string(kind), u.store.Instance, id); err != nil {
				return nil, err
			}
		}
	}
	heads, err := u.canonicalHeads(ctx, kind, ent.ID, ids)
	if err != nil {
		return nil, err
	}
	for _, t := range heads {
		result = append(result, app.PositionSource{Timeline: t})
	}
	return result, nil
}

func (u *unit) PositionUses(ctx context.Context, enterprise, field, id string) ([]staff.PositionUse, error) {
	return u.store.PositionUses(dbtx.With(ctx, u.tx), enterprise, field, id)
}

func (s *Store) PositionUses(ctx context.Context, enterprise, field, id string) ([]staff.PositionUse, error) {
	tx, ok := dbtx.From(ctx)
	if !ok || !slices.Contains(d.PositionFields(), field) {
		return nil, errors.New("invalid position dependency query")
	}
	rows, err := tx.Query(ctx, `SELECT head FROM staffing.position WHERE instance_id=$1 AND enterprise_id=$2 AND head @> jsonb_build_object('records',jsonb_build_array(jsonb_build_object('data',jsonb_build_object($3::text,$4::text))))`, s.Instance, enterprise, field, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var uses []staff.PositionUse
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		position, err := decode(raw, d.Position)
		if err != nil {
			return nil, err
		}
		for _, record := range position.Records {
			if record.Active() && record.Data[field] == id {
				uses = append(uses, staff.PositionUse{Start: record.Date, End: record.End, GradeID: record.Data["gradeId"]})
			}
		}
	}
	return uses, rows.Err()
}
