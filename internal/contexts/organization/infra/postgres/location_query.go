package postgres

import (
	geo "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/location"

	cd "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"slices"
	"strings"
)

func (u *locationUnit) saveRead(ctx context.Context, t d.Timeline) error {
	if _, e := u.tx.Exec(ctx, `DELETE FROM organization.location_read WHERE instance_id=$1 AND object_id=$2`, u.instance, t.ID); e != nil {
		return e
	}
	for _, r := range t.Records {
		_, e := u.tx.Exec(ctx, `INSERT INTO organization.location_read(instance_id,enterprise_id,object_id,record_id,valid_period,name,name_en,state,effective_date,country_id,area_id,address_line,postal_code) VALUES($1,$2,$3,$4,daterange($5::date,NULLIF($6,'')::date,'[)'),$7,$8,$9,$5::date,$10,$11,$12,$13)`, u.instance, t.Enterprise, t.ID, r.ID, r.Date, r.End, r.Data.Name, r.Data.NameEn, r.State, r.Data.CountryID, r.Data.AreaID, r.Data.AddressLine, r.Data.PostalCode)
		if e != nil {
			return e
		}
	}
	return nil
}
func locationSort(in string, fields []string) (string, bool) {
	if in == "" {
		if slices.Contains(fields, "code") {
			return "code", false
		}
		return "id", false
	}
	key := strings.TrimSuffix(in, "_desc")
	return map[string]string{"code": "code", "name": "name", "nameEn": "name_en", "addressLine": "address_line", "postalCode": "postal_code", "status": "status", "effectiveDate": "effective_date"}[key], strings.HasSuffix(in, "_desc")
}
func locationQuery(instance, enterprise string, in org.LocationQuery, fields []string, today string) (string, []any) {
	args := []any{instance, enterprise, today}
	where := []string{"a.instance_id=$1", "a.enterprise_id=$2"}
	add := func(sql string, value any) {
		args = append(args, value)
		where = append(where, strings.ReplaceAll(sql, "?", fmt.Sprintf("$%d", len(args))))
	}
	status := `CASE WHEN r.effective_date>$3::date THEN 'not_effective' ELSE r.state END`
	if in.Status != "" {
		add(status+"=?", in.Status)
	}
	if in.CountryRegionID != "" {
		add("r.country_id=?::uuid", in.CountryRegionID)
	}
	if in.Q != "" {
		parts := []string{}
		if slices.Contains(fields, "code") {
			parts = append(parts, "a.code")
		}
		if slices.Contains(fields, "name") {
			parts = append(parts, "r.name")
		}
		if len(parts) == 0 {
			where = append(where, "false")
		} else {
			add("strpos(lower("+strings.Join(parts, "||chr(10)||")+"),?)>0", in.Q)
		}
	}
	if f := in.Filter; f != nil {
		col := map[string]string{"code": "a.code", "name": "r.name", "nameEn": "coalesce(r.name_en,'')", "addressLine": "r.address_line", "postalCode": "coalesce(r.postal_code,'')", "status": status, "effectiveDate": "r.effective_date"}[f.Field]
		switch {
		case f.Field == "effectiveDate":
			op := map[string]string{"is": "=", "before": "<", "after": ">", "onOrBefore": "<=", "onOrAfter": ">="}[f.Operator]
			add(col+op+"?::date", f.Value)
		case f.Operator == "contains":
			add("strpos(lower("+col+"),?)>0", f.Value)
		case f.Operator == "isAnyOf":
			add(col+"=ANY(?::text[])", strings.Split(f.Value, ","))
		default:
			add("lower("+col+")=lower(?)", f.Value)
		}
	}
	query := `SELECT a.location_id::text AS id,a.code,r.name,coalesce(r.name_en,'') AS name_en,r.address_line,coalesce(r.postal_code,'') AS postal_code,r.effective_date,` + status + ` AS status FROM organization.location a
 JOIN LATERAL(SELECT * FROM organization.location_read x WHERE (x.instance_id,x.enterprise_id,x.object_id)=(a.instance_id,a.enterprise_id,a.location_id) AND (x.valid_period @> $3::date OR x.record_id=a.first_record_id) ORDER BY x.effective_date DESC LIMIT 1) r ON true WHERE ` + strings.Join(where, " AND ")
	return query, args
}
func locationOrder(sort string, fields []string) string {
	col, desc := locationSort(sort, fields)
	if desc {
		return col + " DESC,id"
	}
	return col + ",id"
}
func (u *locationUnit) List(ctx context.Context, in org.LocationQuery, fields []string, today string) (org.LocationList, error) {
	out := org.LocationList{Items: []org.LocationSummary{}, Page: in.Page, PageSize: in.PageSize, CurrentDate: today}
	query, args := locationQuery(u.instance, u.enterprise.ID, in, fields, today)
	args = append(args, in.PageSize, (in.Page-1)*in.PageSize)
	sql := `WITH matched AS MATERIALIZED (` + query + `), page AS (SELECT id,row_number() OVER(ORDER BY ` + locationOrder(in.Sort, fields) + `) AS ordinal FROM matched ORDER BY ` + locationOrder(in.Sort, fields) + fmt.Sprintf(` LIMIT $%d OFFSET $%d) `, len(args)-1, len(args)) + locationCaptureSQL(`(SELECT count(*) FROM matched)`)
	count, projections, e := u.captureLocationProjections(ctx, sql, args)
	if e != nil {
		return out, e
	}
	out.Count = count
	ids := []string{}
	for _, r := range projections {
		ids = append(ids, r.head.ID)
	}
	states, e := u.decodeLocationProjections(ctx, projections)
	if e != nil {
		return out, e
	}
	byID := map[string]d.Timeline{}
	for _, v := range states {
		byID[v.Timeline.ID] = v.Timeline
	}
	has := func(f string) bool { return slices.Contains(fields, f) }
	// The current page can share exact country adoptions. Resolve each once,
	// using the same saved-source and current permission rules as detail views.
	countryNames := map[cd.Adoption]*string{}
	for _, id := range ids {
		t, ok := byID[id]
		if !ok {
			return out, fmt.Errorf("missing captured location")
		}
		r := d.Kernel.FactAt(t.Records, today)
		state := "not_effective"
		if r == nil {
			r = &t.Records[0]
		} else {
			state = r.State
		}
		v := org.LocationSummary{ID: id}
		if has("code") {
			v.Code = &t.Code
		}
		if has("name") {
			v.Name = &r.Data.Name
		}
		if has("nameEn") {
			v.NameEn = r.Data.NameEn
		}
		if has("countryRegionId") {
			v.CountryRegionID = &r.Data.CountryID
			a := r.Data.Adoption
			ref := a.Country
			key := cd.Adoption{Date: a.Date, ObjectID: ref.ObjectID, SourceVersion: ref.SourceVersion, RecordID: ref.RecordID, Revision: ref.Revision}
			name, resolved := countryNames[key]
			if !resolved {
				a.Parents = nil
				path, err := geo.ProjectAdoption(ctx, u.References(), a, today)
				if err != nil {
					return out, err
				}
				if len(path) > 0 {
					name = &path[0].Name
				}
				countryNames[key] = name
			}
			v.CountryName = name
		}
		if has("geoAreaId") {
			v.GeoAreaID = r.Data.AreaID
		}
		if has("addressLine") {
			v.AddressLine = &r.Data.AddressLine
		}
		if has("postalCode") {
			v.PostalCode = r.Data.PostalCode
		}
		if has("status") {
			v.Status = &state
		}
		if has("effectiveDate") {
			v.EffectiveDate = &r.Date
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}
func (u *locationUnit) Navigate(ctx context.Context, in org.LocationNavigationInput, fields []string, today string) (out org.NavigationResult, err error) {
	q := org.LocationQuery{ListInput: org.ListInput{Q: strings.ToLower(strings.TrimSpace(in.Query.Q)), Status: in.Query.Status, Sort: in.Sort}, CountryRegionID: in.Query.CountryRegionID}
	if q.Status == "all" {
		q.Status = ""
	}
	if in.Query.FilterField != "" {
		q.Filter = &org.ColumnFilter{Field: in.Query.FilterField, Operator: in.Query.FilterOperator, Value: strings.ToLower(strings.TrimSpace(in.Query.FilterValue))}
	}
	query, args := locationQuery(u.instance, u.enterprise.ID, q, fields, today)
	args = append(args, in.AnchorID)
	offset := 0
	if in.Direction == "previous" {
		offset = -1
	} else if in.Direction == "next" {
		offset = 1
	}
	// The shared detail navigator needs the position and both neighbors from
	// the same authorized query. Rank narrow current facts, without loading
	// Location history or decoding adopted country/area labels.
	sql := `WITH m AS NOT MATERIALIZED (` + query + fmt.Sprintf(`), ranked AS MATERIALIZED (
 SELECT id,row_number() OVER w AS position,count(*) OVER () AS total,
 lag(id) OVER w AS previous_id,lead(id) OVER w AS next_id
 FROM m WINDOW w AS (ORDER BY %s)
 ) SELECT id,position,total,previous_id,next_id FROM ranked
 WHERE position=(SELECT position FROM ranked WHERE id=$%d)+%d`, locationOrder(in.Sort, fields), len(args), offset)
	e := u.tx.QueryRow(ctx, sql, args...).Scan(&out.TargetID, &out.Position, &out.Count, &out.PreviousID, &out.NextID)
	if errors.Is(e, pgx.ErrNoRows) {
		return org.NavigationResult{}, nil
	}
	if e != nil {
		return out, e
	}
	return out, nil
}

func (u *locationUnit) LoadMany(ctx context.Context, ids []string) ([]app.State, error) {
	if u.exportHeads == nil {
		return u.locationCurrent(ctx, ids)
	}
	rows := make([]projectionRow, 0, len(ids))
	for _, id := range ids {
		r, ok := u.exportHeads[id]
		if !ok {
			return nil, fmt.Errorf("uncaptured export member")
		}
		rows = append(rows, r)
	}
	return u.decodeLocationProjections(ctx, rows)
}
func (u *locationUnit) QueryIDs(ctx context.Context, in org.LocationQuery, fields []string, today string, only []string) ([]string, error) {
	query, args := locationQuery(u.instance, u.enterprise.ID, in, fields, today)
	if only != nil {
		args = append(args, only)
		query = `SELECT * FROM (` + query + fmt.Sprintf(`) selected WHERE id=ANY($%d::text[])`, len(args))
	}
	rows, e := u.tx.Query(ctx, `SELECT id FROM (`+query+`) m ORDER BY `+locationOrder(in.Sort, fields)+fmt.Sprintf(" LIMIT %d", app.DraftMaxMembers+1), args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		out = append(out, id)
	}
	if len(out) > app.DraftMaxMembers {
		return nil, app.DraftMemberLimit()
	}
	return out, rows.Err()
}

// A single statement captures both membership and sealed current projections.
// Catalog.Enter pins their immutable dictionaries/shapes for this request only.
func locationCaptureSQL(count string) string {
	return `SELECT ` + count + `,coalesce(b.location_id::text,''),coalesce(b.enterprise_id::text,''),coalesce(b.code,''),coalesce(b.first_record_id::text,''),coalesce(b.head_source_id::text,''),coalesce(b.head_no,0),coalesce(b.head_chunk,0),coalesce(h.dictionary_id,0),coalesce(b.current_shape,0),coalesce(organization.project_ids(h.ids,b.current_refs)::text[],ARRAY[]::text[]),coalesce(b.current_envelope,''::bytea) FROM (SELECT 1) singleton LEFT JOIN page p ON true LEFT JOIN organization.location b ON b.instance_id=$1 AND b.location_id=p.id::uuid LEFT JOIN organization.location_history h ON (h.instance_id,h.object_id,h.chunk_no)=(b.instance_id,b.location_id,b.head_chunk) ORDER BY p.ordinal`
}
func (u *locationUnit) captureLocationProjections(ctx context.Context, sql string, args []any) (int, []projectionRow, error) {
	rows, e := u.tx.Query(ctx, sql, args...)
	if e != nil {
		return 0, nil, e
	}
	defer rows.Close()
	count := 0
	out := []projectionRow{}
	for rows.Next() {
		var r projectionRow
		if e = rows.Scan(&count, &r.head.ID, &r.head.Enterprise, &r.head.Code, &r.head.First, &r.head.Source, &r.head.Number, &r.head.Chunk, &r.dict, &r.shape, &r.ids, &r.envelope); e != nil {
			return 0, nil, e
		}
		if r.head.ID != "" {
			out = append(out, r)
		}
	}
	return count, out, rows.Err()
}
func (u *locationUnit) CaptureExport(ctx context.Context, in org.LocationQuery, fields []string, today string, ids []string) ([]string, error) {
	query, args := locationQuery(u.instance, u.enterprise.ID, in, fields, today)
	args = append(args, ids)
	sql := `WITH page AS MATERIALIZED (SELECT *,row_number() OVER(ORDER BY ` + locationOrder(in.Sort, fields) + `) AS ordinal FROM (` + query + fmt.Sprintf(`) selected WHERE id=ANY($%d::text[]) ORDER BY `, len(args)) + locationOrder(in.Sort, fields) + `) ` + locationCaptureSQL(`0`)
	_, rows, e := u.captureLocationProjections(ctx, sql, args)
	if e != nil {
		return nil, e
	}
	u.exportHeads = map[string]projectionRow{}
	ordered := []string{}
	for _, r := range rows {
		u.exportHeads[r.head.ID] = r
		ordered = append(ordered, r.head.ID)
	}
	return ordered, nil
}
