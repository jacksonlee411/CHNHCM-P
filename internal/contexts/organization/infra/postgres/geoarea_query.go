package postgres

import (
	ca "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/geoarea"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"slices"
	"strings"
	"unicode"
)

func (u *unit) AddressCountries(ctx context.Context) ([]ca.State, error) {
	rows, e := u.tx.Query(ctx, `SELECT country_region_id::text FROM organization.country_region WHERE instance_id=$1 AND code IN ('CN','HK','MO','TW') AND EXISTS(SELECT FROM organization.country_package WHERE instance_id=$1) ORDER BY code`, u.instance)
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	return (&countryUnit{u}).countryCurrent(ctx, ids)
}

func (u *geoareaUnit) Country(ctx context.Context, id, source string) (ca.State, error) {
	var ok bool
	if e := u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM organization.country_package WHERE instance_id=$1)`, u.instance).Scan(&ok); e != nil {
		return ca.State{}, e
	}
	if !ok {
		return ca.State{}, foundation.Fail(409, "M02_CATALOG_NOT_READY", "国家目录尚未就绪")
	}
	return (&countryUnit{u.unit}).Load(ctx, id, source)
}
func (u *geoareaUnit) LockReference(ctx context.Context, kind, id string) error {
	table, key := "geoarea", "geoarea_id"
	if kind == "country" {
		table, key = "country_region", "country_region_id"
	} else if kind != "geoarea" {
		return fmt.Errorf("invalid reference owner")
	}
	_, e := u.tx.Exec(ctx, fmt.Sprintf(`SELECT %s FROM organization.%s WHERE instance_id=$1 AND %s=$2 FOR SHARE`, key, table, key), u.instance, id)
	return e
}
func (u *geoareaUnit) Published(ctx context.Context, id string) (bool, error) {
	var ok bool
	e := u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM organization.geoarea a JOIN organization.geoarea_package p ON (p.instance_id,p.country_id)=(a.instance_id,a.country_id) WHERE a.instance_id=$1 AND a.geoarea_id=$2)`, u.instance, id).Scan(&ok)
	return ok, e
}
func (u *geoareaUnit) RegionPublished(ctx context.Context, id string) (bool, error) {
	var ok bool
	e := u.tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM organization.geoarea_package WHERE instance_id=$1 AND country_id=$2)`, u.instance, id).Scan(&ok)
	return ok, e
}
func (u *geoareaUnit) IsTerminal(ctx context.Context, id string) (bool, error) {
	var ok bool
	e := u.tx.QueryRow(ctx, `SELECT NOT EXISTS(SELECT FROM organization.geoarea WHERE instance_id=$1 AND parent_id=$2)`, u.instance, id).Scan(&ok)
	return ok, e
}
func (u *geoareaUnit) Mappings(ctx context.Context, country string) ([]app.Mapping, error) {
	rows, e := u.tx.Query(ctx, `SELECT geoarea_id::text,initial_date::text,country_id::text,country_code,scheme,code,nature,level,coalesce(parent_id::text,''),source_key,source_version,package_version,package_digest,archive_digest FROM organization.geoarea WHERE instance_id=$1 AND country_id=$2 ORDER BY geoarea_id`, u.instance, country)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []app.Mapping{}
	for rows.Next() {
		var m app.Mapping
		o := &m.Origin
		if e = rows.Scan(&m.ID, &m.Date, &o.CountryID, &o.CountryCode, &o.Scheme, &o.Code, &o.Nature, &o.Level, &o.ParentID, &o.Key, &o.SourceVersion, &o.PackageVersion, &o.PackageDigest, &o.ArchiveDigest); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (u *geoareaUnit) CatalogComplete(ctx context.Context, p app.CatalogPublication) (bool, error) {
	var v app.CatalogPublication
	v.CountryID = p.CountryID
	e := u.tx.QueryRow(ctx, `SELECT country_code,version,package_digest,archive_digest,area_count FROM organization.geoarea_package WHERE instance_id=$1 AND country_id=$2`, u.instance, p.CountryID).Scan(&v.CountryCode, &v.Version, &v.Digest, &v.ArchiveDigest, &v.Count)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if v != p {
		return false, foundation.Fail(409, "M02_SOURCE_CONFLICT", "区域目录已采用其他固定包")
	}
	return true, nil
}
func (u *geoareaUnit) PublishCatalog(ctx context.Context, p app.CatalogPublication, expected []app.Mapping) error {
	mappings, e := u.Mappings(ctx, p.CountryID)
	if e != nil {
		return e
	}
	want := map[string]app.Mapping{}
	for _, m := range expected {
		want[m.ID] = m
	}
	if len(mappings) != p.Count || len(want) != p.Count {
		return foundation.Fail(409, "M02_SOURCE_CONFLICT", "区域目录数量不一致")
	}
	for _, m := range mappings {
		if want[m.ID] != m {
			return foundation.Fail(409, "M02_SOURCE_CONFLICT", "区域初始化映射或结构不一致")
		}
	}
	_, e = u.tx.Exec(ctx, `INSERT INTO organization.geoarea_package(instance_id,country_id,country_code,version,package_digest,archive_digest,area_count) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(instance_id,country_id) DO NOTHING`, u.instance, p.CountryID, p.CountryCode, p.Version, p.Digest, p.ArchiveDigest, p.Count)
	if e != nil {
		return e
	}
	_, e = u.CatalogComplete(ctx, p)
	return e
}
func (u *geoareaUnit) CandidateRows(ctx context.Context, in org.GeoAreaCandidateInput, key, id string, limit int) ([]app.CandidateRow, error) {
	// One transaction-local cursor owns the entire request snapshot, including
	// every candidate's finite chain. It streams batches and stores no query job.
	// The scalar child probe stops at one structural child; it cannot become a
	// hashed anti-join that scans all areas just to mark this page's terminal nodes.
	if !u.candidateCursor {
		terms := strings.FieldsFunc(strings.ToLower(in.Q), func(r rune) bool {
			return unicode.IsSpace(r) || strings.ContainsRune("/／>＞,，·", r)
		})
		_, e := u.tx.Exec(ctx, `DECLARE geoarea_candidates NO SCROLL CURSOR FOR SELECT a.geoarea_id::text,a.code,
  (SELECT true FROM organization.geoarea child WHERE child.instance_id=a.instance_id AND child.country_id=a.country_id AND child.parent_id=a.geoarea_id LIMIT 1) IS NULL,
  a.head_source_id::text,a.country_id::text,c.head_source_id::text,a.ancestors::text[],
  ARRAY(SELECT parent.head_source_id::text FROM organization.geoarea parent WHERE parent.instance_id=a.instance_id AND parent.geoarea_id=ANY(a.ancestors) ORDER BY array_position(a.ancestors,parent.geoarea_id))
  FROM organization.geoarea a JOIN organization.country_region c ON (c.instance_id,c.country_region_id)=(a.instance_id,a.country_id)
  JOIN organization.geoarea_read r ON (r.instance_id,r.object_id)=(a.instance_id,a.geoarea_id) AND r.valid_period @> $3::date
  LEFT JOIN LATERAL (
    SELECT string_agg(part.name,'' ORDER BY chain.ordinality) AS full_name
    FROM unnest(a.ancestors||a.geoarea_id) WITH ORDINALITY chain(id,ordinality)
    JOIN organization.geoarea_read part ON part.instance_id=a.instance_id AND part.object_id=chain.id AND part.valid_period @> $3::date
    WHERE $5<>''
  ) address_path ON true
  WHERE a.instance_id=$1 AND a.country_id=$2 AND r.state='active'
  AND (($5='' AND (($4='' AND a.parent_id IS NULL) OR a.parent_id=NULLIF($4,'')::uuid)) OR ($5<>'' AND ($4='' OR NULLIF($4,'')::uuid=ANY(a.ancestors)) AND
    (strpos(lower(a.code),lower($5))>0 OR strpos(lower(r.name),lower($5))>0 OR
      (cardinality($8::text[])>0 AND NOT EXISTS(SELECT FROM unnest($8::text[]) term WHERE strpos(lower(address_path.full_name),term)=0)))))
  AND ($7='' OR (a.code,a.geoarea_id)>($6,NULLIF($7,'')::uuid)) ORDER BY a.code,a.geoarea_id`, u.instance, in.CountryRegionID, in.AdoptedDate, in.ParentAreaID, in.Q, key, id, terms)
		if e != nil {
			return nil, e
		}
		u.candidateCursor = true
	}
	rows, e := u.tx.Query(ctx, fmt.Sprintf("FETCH FORWARD %d FROM geoarea_candidates", limit))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []app.CandidateRow{}
	for rows.Next() {
		var r app.CandidateRow
		if e = rows.Scan(&r.ID, &r.Code, &r.Terminal, &r.Source, &r.CountryID, &r.CountrySource, &r.ParentIDs, &r.ParentSources); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func geoQuery(instance string, in org.GeoAreaQuery, fields []string, today string) (string, []any, error) {
	has := func(f string) bool { return slices.Contains(fields, f) }
	deny := func() (string, []any, error) {
		return "", nil, foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许筛选或排序字段")
	}
	if in.Sort == "code" && !has("code") || in.Sort != "code" && !has("name") || in.Status != "" && !has("state") || in.CountryRegionID != "" && !has("countryRegionId") || in.Level != "" && !has("level") || in.ParentAreaID != "" && !has("parentAreaId") {
		return deny()
	}
	args := []any{instance, today}
	clauses := []string{"a.instance_id=$1"}
	add := func(sql string, value any) {
		args = append(args, value)
		clauses = append(clauses, strings.ReplaceAll(sql, "?", fmt.Sprintf("$%d", len(args))))
	}
	if in.CountryRegionID != "" {
		add("a.country_id=?::uuid", in.CountryRegionID)
	}
	if in.ParentAreaID != "" {
		add("a.parent_id=?::uuid", in.ParentAreaID)
	}
	if in.Level != "" {
		add("a.level=?", in.Level)
	}
	status := `CASE WHEN r.effective_date>$2::date THEN 'not_effective' ELSE r.state END`
	if in.Status != "" {
		add(status+"=?", in.Status)
	}
	if in.Q != "" {
		parts := []string{}
		for f, c := range map[string]string{"code": "a.code", "name": "r.name", "nameEn": "coalesce(r.name_en,'')"} {
			if has(f) {
				parts = append(parts, c)
			}
		}
		if len(parts) == 0 {
			return deny()
		}
		slices.Sort(parts)
		add("strpos(lower("+strings.Join(parts, "||chr(10)||")+"),lower(?))>0", in.Q)
	}
	if in.Filter != nil {
		f := in.Filter
		field := f.Field
		if field == "status" {
			field = "state"
		}
		if !has(field) {
			return deny()
		}
		col := map[string]string{"code": "a.code", "name": "r.name", "status": status, "countryRegionId": "a.country_id::text", "level": "a.level"}[f.Field]
		switch f.Operator {
		case "contains":
			add("strpos(lower("+col+"),lower(?))>0", f.Value)
		case "isAnyOf":
			add(col+"=ANY(?::text[])", strings.Split(f.Value, ","))
		default:
			add("lower("+col+")=lower(?)", f.Value)
		}
	}
	sql := `SELECT a.geoarea_id::text AS id,a.code,r.name,` + status + ` AS status FROM organization.geoarea a JOIN organization.geoarea_package p ON (p.instance_id,p.country_id)=(a.instance_id,a.country_id)
 JOIN LATERAL(SELECT * FROM organization.geoarea_read x WHERE (x.instance_id,x.object_id)=(a.instance_id,a.geoarea_id) AND (x.valid_period @> $2::date OR x.effective_date=a.initial_date) ORDER BY x.effective_date DESC LIMIT 1) r ON true WHERE ` + strings.Join(clauses, " AND ")
	return sql, args, nil
}
func geoSort(sort string) string {
	if sort == "name_desc" {
		return "name DESC,id"
	}
	if sort == "name" {
		return "name,id"
	}
	return "code,id"
}
func (u *geoareaUnit) List(ctx context.Context, in org.GeoAreaQuery, fields []string, today string) (org.GeoAreaList, error) {
	out := org.GeoAreaList{Items: []org.GeoAreaSummary{}, Page: in.Page, PageSize: in.PageSize, CurrentDate: today}
	query, args, e := geoQuery(u.instance, in, fields, today)
	if e != nil {
		return out, e
	}
	if e = u.tx.QueryRow(ctx, `SELECT count(*) FROM (`+query+`) m`, args...).Scan(&out.Count); e != nil {
		return out, e
	}
	if in.Page-1 > out.Count/in.PageSize {
		return out, nil
	}
	args = append(args, in.PageSize, (in.Page-1)*in.PageSize)
	rows, e := u.tx.Query(ctx, `SELECT id FROM (`+query+`) m ORDER BY `+geoSort(in.Sort)+fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if e != nil {
		return out, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return out, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	states, e := u.geoareaCurrent(ctx, ids)
	if e != nil {
		return out, e
	}
	byID := map[string]app.State{}
	for _, v := range states {
		byID[v.Timeline.ID] = v
	}
	has := func(f string) bool { return slices.Contains(fields, f) }
	ptr := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}
	for _, id := range ids {
		t := byID[id].Timeline
		r := d.Kernel.FactAt(t.Records, today)
		status := "not_effective"
		if r == nil {
			r = &t.Records[0]
		} else {
			status = r.State
		}
		st := t.Origin.Structure
		v := org.GeoAreaSummary{ID: id}
		if has("code") {
			v.Code = ptr(st.Code)
		}
		if has("name") {
			v.Name = ptr(r.Data.Name)
		}
		if has("nameEn") {
			v.NameEn = r.Data.NameEn
		}
		if has("state") {
			v.Status = ptr(status)
		}
		if has("effectiveDate") {
			v.EffectiveDate = ptr(r.Date)
		}
		if has("countryRegionId") {
			v.CountryRegionID = ptr(st.CountryID)
		}
		if has("parentAreaId") {
			p := ptr(st.ParentID)
			v.ParentAreaID = &p
		}
		if has("scheme") {
			v.Scheme = ptr(st.Scheme)
		}
		if has("nature") {
			v.Nature = ptr(st.Nature)
		}
		if has("level") {
			v.Level = ptr(st.Level)
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}
func (u *geoareaUnit) Navigate(ctx context.Context, in org.GeoAreaNavigationInput, fields []string, today string) (out org.NavigationResult, err error) {
	query, args, e := geoQuery(u.instance, in.Query, fields, today)
	if e != nil {
		return out, e
	}
	args = append(args, in.AnchorID)
	col, desc := "code", false
	if in.Query.Sort != "code" {
		col = "name"
		desc = in.Query.Sort == "name_desc"
	}
	order := "ASC"
	if desc {
		order = "DESC"
	}
	offset := 0
	if in.Direction == "previous" {
		offset = -1
	} else if in.Direction == "next" {
		offset = 1
	}
	// Rank exactly the same authorized, filtered facts as the list. The UUID
	// tie order remains ascending even when the visible name is descending.
	sql := `WITH m AS NOT MATERIALIZED (` + query + fmt.Sprintf(`), ranked AS MATERIALIZED (
 SELECT id,row_number() OVER w AS position,count(*) OVER () AS total,
 lag(id) OVER w AS previous_id,lead(id) OVER w AS next_id
 FROM m WINDOW w AS (ORDER BY %s %s,id ASC)
 ) SELECT id,position,total,previous_id,next_id FROM ranked
 WHERE position=(SELECT position FROM ranked WHERE id=$%d)+%d`, col, order, len(args), offset)
	e = u.tx.QueryRow(ctx, sql, args...).Scan(&out.TargetID, &out.Position, &out.Count, &out.PreviousID, &out.NextID)
	if errors.Is(e, pgx.ErrNoRows) {
		return org.NavigationResult{}, nil
	}
	if e != nil {
		return out, e
	}
	return out, nil
}

func (u *geoareaUnit) PathHeads(ctx context.Context, s d.Structure) (app.CandidateRow, error) {
	var out app.CandidateRow
	e := u.tx.QueryRow(ctx, `SELECT c.country_region_id::text,c.head_source_id::text,
 CASE WHEN p.geoarea_id IS NULL THEN '{}'::text[] ELSE (p.ancestors||p.geoarea_id)::text[] END,
 ARRAY(SELECT x.head_source_id::text FROM organization.geoarea x WHERE x.instance_id=c.instance_id AND x.geoarea_id=ANY(p.ancestors||p.geoarea_id) ORDER BY array_position(p.ancestors||p.geoarea_id,x.geoarea_id))
 FROM organization.country_region c LEFT JOIN organization.geoarea p ON p.instance_id=c.instance_id AND p.country_id=c.country_region_id AND p.geoarea_id=NULLIF($3,'')::uuid
 WHERE c.instance_id=$1 AND c.country_region_id=$2`, u.instance, s.CountryID, s.ParentID).Scan(&out.CountryID, &out.CountrySource, &out.ParentIDs, &out.ParentSources)
	return out, e
}
