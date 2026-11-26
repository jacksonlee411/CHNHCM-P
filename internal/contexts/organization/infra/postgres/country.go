package postgres

import (
	"bytes"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/country"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/country"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history/model"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"slices"
	"strings"
	"time"
)

type countryUnit struct{ *unit }

func (s *Store) WithinCountry(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, false, func(u *unit) error { return fn(&countryUnit{u}) })
}
func (s *Store) WithinCountryWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, true, func(u *unit) error { return fn(&countryUnit{u}) })
}
func (u *countryUnit) LockSubmission(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key, e := (submission.Identity{Kind: submission.Country, InstanceID: u.instance, SubjectID: u.guard.Principal().ID, SubmissionID: id}).LockKey()
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", key)
	return e
}
func (u *countryUnit) Lock(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	b, _ := json.Marshal([]string{"organization.country.object", u.instance, id})
	h := sha256.Sum256(b)
	if _, e := u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(binary.BigEndian.Uint64(h[:8]))); e != nil {
		return e
	}
	_, e := u.tx.Exec(ctx, `SELECT country_region_id FROM organization.country_region WHERE instance_id=$1 AND country_region_id=$2 FOR UPDATE`, u.instance, id)
	return e
}
func (u *countryUnit) Load(ctx context.Context, id, source string) (app.State, error) {
	if source == "" {
		states, e := u.countryCurrent(ctx, []string{id})
		if e != nil {
			return app.State{}, e
		}
		if len(states) == 0 {
			return app.State{}, nil
		}
		return states[0], nil
	}
	number, e := history.Locate(ctx, u.tx, history.Country, u.instance, id, source)
	if errors.Is(e, pgx.ErrNoRows) {
		return app.State{}, nil
	}
	if e != nil {
		return app.State{}, e
	}
	chunk, e := history.LoadChunk(ctx, u.tx, history.Country, u.instance, id, number)
	if e != nil {
		return app.State{}, e
	}
	slot := chunk.Index(source)
	if slot < 0 {
		return app.State{}, compact.ErrCorrupt
	}
	b, e := history.LoadShape(ctx, u.tx, history.Country, u.instance, u.instance, chunk.States[slot])
	if e != nil {
		return app.State{}, e
	}
	v, e := history.DecodeCountryState(ctx, chunk.Values, b)
	if e != nil {
		return app.State{}, e
	}
	if v.State.Timeline.ID != id || v.State.Timeline.Source != source || v.State.Timeline.SourceNo != chunk.First+int64(slot) {
		return app.State{}, compact.ErrCorrupt
	}
	return v.State, validateCountryState(v.State)
}
func validateCountryState(v app.State) error {
	t := v.Timeline
	if !foundation.ValidID(t.ID) || !foundation.ValidID(t.First) || !foundation.ValidID(t.Source) || t.SourceNo < 1 || len(t.Records) == 0 || t.Records[0].ID != t.First {
		return compact.ErrCorrupt
	}
	seen := map[string]bool{}
	summaries := map[string]org.OperationSummary{}
	for _, s := range v.Summaries {
		if summaries[s.RecordID].RecordID != "" || !foundation.ValidID(s.OperationSourceVersion) {
			return compact.ErrCorrupt
		}
		summaries[s.RecordID] = s
	}
	for i, r := range t.Records {
		if !foundation.ValidID(r.ID) || !foundation.ValidID(r.Revision) || seen[r.ID] || !foundation.ValidDate(r.Date) || r.RevisionNo < 1 {
			return compact.ErrCorrupt
		}
		seen[r.ID] = true
		end := ""
		if i+1 < len(t.Records) {
			end = t.Records[i+1].Date
			if end <= r.Date {
				return compact.ErrCorrupt
			}
		}
		if r.End != end {
			return compact.ErrCorrupt
		}
		if summaries[r.ID].Revision != r.Revision {
			return compact.ErrCorrupt
		}
	}
	return nil
}
func (u *countryUnit) countryCurrent(ctx context.Context, ids []string) ([]app.State, error) {
	rows, e := u.tx.Query(ctx, `SELECT b.country_region_id::text,b.catalog_id::text,b.code,b.first_record_id::text,b.head_source_id::text,b.head_no,b.head_chunk,h.dictionary_id,b.current_shape,organization.project_ids(h.ids,b.current_refs)::text[],b.current_envelope,b.source_key FROM organization.country_region b JOIN organization.country_history h ON (h.instance_id,h.object_id,h.chunk_no)=(b.instance_id,b.country_region_id,b.head_chunk) WHERE b.instance_id=$1 AND ($2 OR b.country_region_id=ANY($3::text[]::uuid[])) ORDER BY b.country_region_id`, u.instance, ids == nil, ids)
	if e != nil {
		return nil, e
	}
	projections := []projectionRow{}
	sourceKeys := map[string]string{}
	dicts, shapes := []int32{}, []int32{}
	for rows.Next() {
		var r projectionRow
		var sourceKey string
		if e = rows.Scan(&r.head.ID, &r.head.Enterprise, &r.head.Code, &r.head.First, &r.head.Source, &r.head.Number, &r.head.Chunk, &r.dict, &r.shape, &r.ids, &r.envelope, &sourceKey); e != nil {
			rows.Close()
			return nil, e
		}
		sourceKeys[r.head.ID] = sourceKey
		projections = append(projections, r)
		dicts = append(dicts, r.dict)
		shapes = append(shapes, r.shape)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if e = u.catalog.Prepare(ctx, u.tx, history.Country, u.instance, u.instance, dicts, shapes); e != nil {
		return nil, e
	}
	out := make([]app.State, 0, len(projections))
	for _, r := range projections {
		m, e := u.catalog.Get(history.Country, u.instance, u.instance, r.dict, r.shape)
		if e != nil {
			return nil, e
		}
		seal, e := history.ProjectionDigest(history.Country, u.instance, u.instance, r.head.ID, r.head.Source, r.ids, m.DictionaryDigest, m.ShapeDigest)
		if e != nil || !bytes.Equal(seal, r.envelope) {
			return nil, compact.ErrCorrupt
		}
		m.Values.IDs = r.ids
		state, e := history.DecodeCountryState(ctx, m.Values, m.Data)
		if e != nil {
			return nil, e
		}
		t := state.State.Timeline
		if r.head.Enterprise != u.instance || t.ID != r.head.ID || t.First != r.head.First || t.Source != r.head.Source || t.SourceNo != r.head.Number || t.Origin.Code != r.head.Code || t.Origin.Key != sourceKeys[t.ID] {
			return nil, compact.ErrCorrupt
		}
		if e = validateCountryState(state.State); e != nil {
			return nil, e
		}
		out = append(out, state.State)
	}
	return out, nil
}
func (u *countryUnit) Persist(ctx context.Context, w app.Write) (org.CountryResult, error) {
	if !w.Validated() || w.Before.ID != w.After.Timeline.ID || w.After.Timeline.SourceNo != w.Before.SourceNo+1 {
		return org.CountryResult{}, fmt.Errorf("unvalidated country write")
	}
	t, op := w.After.Timeline, w.Operation
	last := int32(0)
	if w.Before.Source == "" {
		_, e := u.tx.Exec(ctx, `INSERT INTO organization.country_region(instance_id,catalog_id,country_region_id,code,normalized_code,source_key,first_record_id,head_source_id,head_no,head_chunk) VALUES($1,$1,$2,$3,lower($3),$4,$5,$6,1,1)`, u.instance, t.ID, t.Origin.Code, t.Origin.Key, t.First, t.Source)
		if e != nil {
			var pg *pgconn.PgError
			if errors.As(e, &pg) && pg.Code == "23505" {
				return org.CountryResult{}, foundation.Fail(409, "M02_CODE_CONFLICT", "来源代码已存在")
			}
			return org.CountryResult{}, e
		}
	} else {
		if e := u.tx.QueryRow(ctx, `UPDATE organization.country_region SET head_source_id=$4,head_no=$5 WHERE instance_id=$1 AND country_region_id=$2 AND head_source_id=$3 RETURNING head_chunk`, u.instance, t.ID, w.Before.Source, t.Source, t.SourceNo).Scan(&last); e != nil {
			return org.CountryResult{}, e
		}
	}
	previous := history.Chunk{Kind: history.Country, Instance: u.instance, Enterprise: u.instance, Object: t.ID, Number: 1, First: 1}
	if last != 0 {
		var e error
		previous, e = history.LoadChunk(ctx, u.tx, history.Country, u.instance, t.ID, last)
		if e != nil {
			return org.CountryResult{}, e
		}
	}
	chunk, e := history.Append(ctx, u.tx, previous, t.SourceNo, t.Source, op.Subject, op.Submission, func(p *compact.Pool) (state, operation, evidence []byte, err error) {
		state, err = history.EncodeCountryState(ctx, p, model.CountryState{State: w.After})
		if err != nil {
			return
		}
		operation, err = history.EncodeCountryOperation(ctx, p, model.CountryOperation{Operation: op})
		if err != nil {
			return
		}
		evidence, err = history.EncodeCountryEvidence(ctx, p, model.CountryEvidence{PackageDigest: t.Origin.PackageDigest})
		return
	})
	if e != nil {
		return org.CountryResult{}, e
	}
	if _, e = u.tx.Exec(ctx, `UPDATE organization.country_region SET head_chunk=$3 WHERE instance_id=$1 AND country_region_id=$2`, u.instance, t.ID, chunk.Number); e != nil {
		return org.CountryResult{}, e
	}
	e = history.SaveProjection(ctx, u.tx, chunk, t.Source, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeCountryState(ctx, p, model.CountryState{State: w.After})
	})
	return op.Terminal.Result, e
}
func (u *countryUnit) StoreTerminal(ctx context.Context, t app.Terminal) error {
	return history.SaveTerminal(ctx, u.tx, history.Country, u.instance, u.instance, u.guard.Principal().ID, t.ID, history.TerminalEnvelope{Object: t.ObjectID, Outcome: t.Outcome}, func(p *compact.Pool) ([]byte, error) {
		return history.EncodeCountryTerminal(ctx, p, model.CountryTerminal{Terminal: t})
	})
}
func (u *countryUnit) operation(ctx context.Context, chunk history.Chunk, slot int) (app.Operation, error) {
	if slot < 0 || slot >= chunk.Count() {
		return app.Operation{}, compact.ErrCorrupt
	}
	b, e := history.LoadShape(ctx, u.tx, history.Country, u.instance, u.instance, chunk.Operations[slot])
	if e != nil {
		return app.Operation{}, e
	}
	v, e := history.DecodeCountryOperation(ctx, chunk.Values, b)
	if e != nil {
		return app.Operation{}, e
	}
	op := v.Operation
	if op.ID != chunk.ID(chunk.Sources[slot]) || op.Subject != chunk.ID(chunk.Subjects[slot]) || op.Submission != chunk.ID(chunk.Submissions[slot]) || op.Number != chunk.First+int64(slot) || op.Terminal.ID != op.Submission || op.Terminal.ObjectID != chunk.Object {
		return app.Operation{}, compact.ErrCorrupt
	}
	return op, nil
}
func (u *countryUnit) FindResult(ctx context.Context, id string) (*app.Saved, error) {
	subject := u.guard.Principal().ID
	meta, e := history.TerminalIdentity(ctx, u.tx, history.Country, u.instance, subject, id)
	if e == nil {
		out := &app.Saved{RequestObjectID: meta.Object, Result: org.CountryResult{SubmissionID: id, Outcome: meta.Outcome}}
		values, b, e := history.LoadTerminal(ctx, u.tx, history.Country, u.instance, subject, id)
		if e != nil {
			out.DecodeError = e
			return out, nil
		}
		v, e := history.DecodeCountryTerminal(ctx, values, b)
		if e != nil {
			out.DecodeError = e
			return out, nil
		}
		t := v.Terminal
		if t.ID != id || t.ObjectID != meta.Object || t.Outcome != meta.Outcome {
			out.DecodeError = compact.ErrCorrupt
			return out, nil
		}
		out.Digest = t.Digest
		out.Result = t.Result
		return out, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	found, e := history.FindSubmission(ctx, u.tx, history.Country, u.instance, subject, id)
	if e != nil || found == nil {
		return nil, e
	}
	out := &app.Saved{RequestObjectID: found.Object, Result: org.CountryResult{SubmissionID: id, Outcome: "accepted", CountryRegionID: &found.Object}}
	chunk, e := history.LoadChunk(ctx, u.tx, history.Country, u.instance, found.Object, found.Chunk)
	if e != nil {
		out.DecodeError = e
		return out, nil
	}
	op, e := u.operation(ctx, chunk, found.Slot)
	if e != nil {
		out.DecodeError = e
		return out, nil
	}
	out.Digest = op.Terminal.Digest
	out.Result = op.Terminal.Result
	return out, nil
}
func (u *countryUnit) History(ctx context.Context, id, through string) ([]app.Operation, error) {
	state, e := u.Load(ctx, id, through)
	if e != nil {
		return nil, e
	}
	if state.Timeline.ID == "" {
		return nil, pgx.ErrNoRows
	}
	rows, e := u.tx.Query(ctx, `SELECT chunk_no FROM organization.country_history WHERE instance_id=$1 AND object_id=$2 AND first_no<=$3 ORDER BY chunk_no`, u.instance, id, state.Timeline.SourceNo)
	if e != nil {
		return nil, e
	}
	numbers := []int32{}
	for rows.Next() {
		var n int32
		if e = rows.Scan(&n); e != nil {
			rows.Close()
			return nil, e
		}
		numbers = append(numbers, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	out := []app.Operation{}
	for _, n := range numbers {
		chunk, e := history.LoadChunk(ctx, u.tx, history.Country, u.instance, id, n)
		if e != nil {
			return nil, e
		}
		for i := 0; i < chunk.Count() && chunk.First+int64(i) <= state.Timeline.SourceNo; i++ {
			op, e := u.operation(ctx, chunk, i)
			if e != nil {
				return nil, e
			}
			out = append(out, op)
		}
	}
	if int64(len(out)) != state.Timeline.SourceNo {
		return nil, compact.ErrCorrupt
	}
	slices.Reverse(out)
	return out, nil
}
func (u *countryUnit) List(ctx context.Context, in org.ListInput, fields []string, today string) (org.CountryList, error) {
	out := org.CountryList{Items: []org.CountrySummary{}, Page: in.Page, PageSize: in.PageSize, CurrentDate: today}
	states, e := u.countryCurrent(ctx, nil)
	if e != nil {
		return out, e
	}
	has := func(f string) bool { return slices.Contains(fields, f) }
	matches := []org.CountrySummary{}
	if in.Sort != "code" && !has("name") || in.Sort == "code" && !has("code") || in.Status != "" && !has("state") {
		return out, foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许所选筛选或排序")
	}
	for _, v := range states {
		t := v.Timeline
		r := d.Kernel.FactAt(t.Records, today)
		status := "not_effective"
		if r == nil {
			x := t.Records[0]
			r = &x
		} else {
			status = r.State
		}
		if in.Status != "" && in.Status != status {
			continue
		}
		if in.Q != "" {
			parts := []string{}
			if has("code") {
				parts = append(parts, t.Origin.Code)
			}
			if has("name") {
				parts = append(parts, r.Data.Name)
			}
			if has("nameEn") && r.Data.NameEn != nil {
				parts = append(parts, *r.Data.NameEn)
			}
			if !strings.Contains(strings.ToLower(strings.Join(parts, "\n")), strings.ToLower(in.Q)) {
				continue
			}
		}
		if in.Filter != nil {
			f := in.Filter
			field := f.Field
			if field == "status" {
				field = "state"
			}
			if !has(field) {
				return out, foundation.Fail(403, "ACCESS_DENIED", "当前权限不允许筛选字段")
			}
			value := map[string]string{"code": t.Origin.Code, "name": r.Data.Name, "status": status}[f.Field]
			ok := false
			switch f.Operator {
			case "contains":
				ok = strings.Contains(strings.ToLower(value), strings.ToLower(f.Value))
			case "isAnyOf":
				ok = slices.Contains(strings.Split(f.Value, ","), value)
			case "equals", "is":
				ok = strings.EqualFold(value, f.Value)
			default:
				return out, foundation.Fail(400, "M02_INVALID_INPUT", "筛选操作无效")
			}
			if !ok {
				continue
			}
		}
		item := org.CountrySummary{ID: t.ID, SourceVersion: t.Source}
		if has("code") {
			x := t.Origin.Code
			item.Code = &x
		}
		if has("name") {
			x := r.Data.Name
			item.Name = &x
		}
		if has("nameEn") {
			item.NameEn = r.Data.NameEn
		}
		if has("phoneCode") {
			item.PhoneCode = r.Data.PhoneCode
		}
		if has("state") {
			item.Status = &status
		}
		if has("effectiveDate") {
			x := r.Date
			item.EffectiveDate = &x
		}
		matches = append(matches, item)
	}
	text := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	slices.SortFunc(matches, func(a, b org.CountrySummary) int {
		var n int
		if in.Sort == "code" {
			n = strings.Compare(text(a.Code), text(b.Code))
		} else {
			n = strings.Compare(text(a.Name), text(b.Name))
			if in.Sort == "name_desc" {
				n = -n
			}
		}
		if n == 0 {
			return strings.Compare(a.ID, b.ID)
		}
		return n
	})
	out.Count = len(matches)
	start := len(matches)
	if in.Page-1 <= len(matches)/in.PageSize {
		start = min((in.Page-1)*in.PageSize, len(matches))
	}
	out.Items = append(out.Items, matches[start:min(start+in.PageSize, len(matches))]...)
	return out, nil
}

func (u *countryUnit) CatalogComplete(ctx context.Context, p app.CatalogPublication) (bool, error) {
	var saved app.CatalogPublication
	e := u.tx.QueryRow(ctx, `SELECT version,package_digest,archive_digest,country_count FROM organization.country_package WHERE instance_id=$1`, u.instance).Scan(&saved.Version, &saved.Digest, &saved.ArchiveDigest, &saved.Count)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if saved != p {
		return false, foundation.Fail(409, "M02_SOURCE_CONFLICT", "目录已采用其他来源包")
	}
	return true, nil
}
func (u *countryUnit) PublishCatalog(ctx context.Context, p app.CatalogPublication) error {
	var count int
	if e := u.tx.QueryRow(ctx, `SELECT count(*) FROM organization.country_region WHERE instance_id=$1`, u.instance).Scan(&count); e != nil {
		return e
	}
	if count != p.Count {
		return foundation.Fail(409, "M02_SOURCE_CONFLICT", "国家目录数量不一致")
	}
	if _, e := u.tx.Exec(ctx, `INSERT INTO organization.country_package(instance_id,version,package_digest,archive_digest,country_count) VALUES($1,$2,$3,$4,$5) ON CONFLICT(instance_id) DO NOTHING`, u.instance, p.Version, p.Digest, p.ArchiveDigest, p.Count); e != nil {
		return e
	}
	_, e := u.CatalogComplete(ctx, p)
	return e
}

// AdoptCountry locks and validates the chosen catalog record in the caller's transaction.
func (s *Store) AdoptCountry(ctx context.Context, c foundation.Credentials, id, source, date string) (org.AdoptedCountry, error) {
	s.historyCatalog.Enter()
	defer s.historyCatalog.Leave()
	u, err := s.profileCatalogUnit(ctx, c)
	if err != nil {
		return org.AdoptedCountry{}, err
	}
	rows, err := app.ResolveCountries(ctx, u, []app.ReferenceRequest{{CountryID: id, Date: date}})
	if err != nil {
		return org.AdoptedCountry{}, err
	}
	a := rows[0]
	if a.SourceVersion != source {
		return org.AdoptedCountry{}, foundation.Fail(409, "PERSON_COUNTRY_CHANGED", "国家/地区资料已变化，请重新选择")
	}
	v, err := u.Country(ctx, id, source)
	if err != nil {
		return org.AdoptedCountry{}, err
	}
	fact := d.Kernel.FactAt(v.Timeline.Records, date)
	if fact == nil || fact.Revision != a.Revision {
		return org.AdoptedCountry{}, foundation.Fail(409, "PERSON_COUNTRY_CHANGED", "国家/地区资料已变化，请重新选择")
	}
	return org.AdoptedCountry{ID: id, SourceVersion: source, RecordID: a.RecordID, Revision: a.Revision, Date: date, Name: fact.Data.Name}, nil
}
