package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/compensation/application"
	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type visibility struct {
	fields                             []string
	gradeName, gradeCode, gradeHistory bool
}

func (u *unit) visibility(ctx context.Context, history bool) (visibility, error) {
	var v visibility
	read, err := u.requireRateRead(ctx)
	if err != nil {
		return v, err
	}
	v.fields = read.ReadFields
	if history {
		h, err := u.require(ctx, "graderate.history.read", nil)
		if err != nil {
			return v, err
		}
		v.fields = slices.DeleteFunc(slices.Clone(v.fields), func(field string) bool { return !has(h.ReadFields, field) })
	}
	if has(v.fields, "gradeId") {
		g, err := u.decision(ctx, "grade.read", nil)
		if err != nil {
			return v, err
		}
		v.gradeName = g.Allowed && has(g.ReadFields, "name")
		v.gradeCode = g.Allowed && has(g.ReadFields, "code")
		if v.gradeName {
			source, e := u.decision(ctx, "grade.history.read", nil)
			if e != nil {
				return v, e
			}
			v.gradeHistory = source.Allowed && has(source.ReadFields, "name")
		}
	}
	return v, nil
}
func (v visibility) value(key, value string) string {
	if has(v.fields, key) {
		return value
	}
	return ""
}
func (v visibility) record(key d.RateKey, r d.Record[d.RateTerms], until string) c.RateRecord {
	a := r.Data.Amounts
	result := c.RateRecord{ID: r.ID, Revision: r.Revision, Start: v.value("effectiveDate", r.Start), Until: v.value("effectiveDate", until), Reason: v.value("reason", r.Reason), Operation: v.value("reason", r.Operation), Form: v.value("form", a.Form), Amount: v.value("amount", a.Amount), Minimum: v.value("minimum", a.Minimum), Guidance: v.value("guidance", a.Guidance), Maximum: v.value("maximum", a.Maximum)}
	if has(v.fields, "status") {
		result.Status = "ACTIVE"
		if !r.Active {
			result.Status = "INACTIVE"
		}
	}
	if v.gradeName {
		g := r.Data.Grade
		result.Grade = &c.GradeLabel{ID: key.GradeID, Name: g.Name}
		if v.gradeCode {
			result.Grade.Code = g.Code
		}
		result.Adopted = &c.Reference{Open: v.gradeHistory, BusinessAt: g.BusinessAt, SourceVersion: g.Source, RecordID: g.Record, Revision: g.Revision}
	}
	return result
}
func (v visibility) rate(s app.RateState, rec d.Record[d.RateTerms], until string) c.Rate {
	r := c.Rate{ID: s.ID, Version: s.Version, Currency: v.value("currency", s.Key.Currency), Frequency: v.value("frequency", s.Key.Frequency), Unit: v.value("unit", s.Key.Unit), Record: v.record(s.Key, rec, until)}
	if has(v.fields, "precision") {
		r.Precision = &s.Places
	}
	return r
}
func (s *Store) loadRate(ctx context.Context, u *unit, id string, lock bool) (app.RateState, error) {
	var state app.RateState
	if !foundation.ValidID(id) {
		return state, unavailable()
	}
	query := `SELECT id::text,enterprise_id::text,head::text,grade_id::text,currency,frequency,unit,places FROM compensation.grade_rate WHERE instance_id=$1 AND enterprise_id=$2 AND id=$3`
	if lock {
		query += " FOR UPDATE"
	}
	err := u.tx.QueryRow(ctx, query, s.Instance, u.enterprise.ID, id).Scan(&state.ID, &state.Enterprise, &state.Version, &state.Key.GradeID, &state.Key.Currency, &state.Key.Frequency, &state.Key.Unit, &state.Places)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, unavailable()
	}
	if err != nil {
		return state, err
	}
	rows, err := u.tx.Query(ctx, `SELECT r.fact FROM compensation.grade_rate_period p JOIN compensation.grade_rate_revision r USING(instance_id,rate_id,record_id,revision) WHERE p.instance_id=$1 AND p.rate_id=$2 ORDER BY p.start_date`, s.Instance, id)
	if err != nil {
		return state, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		var r d.Record[d.RateTerms]
		if err = rows.Scan(&raw); err != nil {
			return state, err
		}
		if err = json.Unmarshal(raw, &r); err != nil {
			return state, err
		}
		state.Records = append(state.Records, r)
	}
	if err = rows.Err(); err != nil {
		return state, err
	}
	if len(state.Records) == 0 {
		return state, errors.New("missing compensation periods")
	}
	return state, nil
}

func (s *Store) RateContext(ctx context.Context, cred foundation.Credentials) (out c.RateContext, err error) {
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		v, e := u.visibility(ctx, false)
		if e != nil {
			return e
		}
		grant, e := u.require(ctx, "graderate.create", nil)
		if e != nil {
			return e
		}
		out.Reasons = d.RateReasons("CREATE")
		out.Units = map[string]string{}
		for _, frequency := range []string{"MONTH", "YEAR", "HOUR", "ONCE"} {
			out.Units[frequency], _ = d.FrequencyUnit(frequency)
		}
		out.DefaultDate, out.Writable = foundation.DefaultEffectiveDate, visibleWrites(v.fields, grant.WriteFields)
		return nil
	})
	return
}

func (s *Store) ListRates(ctx context.Context, cred foundation.Credentials, q c.RateQuery) (out c.RateList, err error) {
	out.Items = []c.Rate{}
	if q.Page < 1 || q.Size < 1 || q.Size > 100 || q.Page > 1000000 {
		return out, d.Invalid("page", "页码或每页条数无效")
	}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		v, e := u.visibility(ctx, false)
		if e != nil {
			return e
		}
		create, e := u.decision(ctx, "graderate.create", nil)
		if e != nil {
			return e
		}
		out.Create, out.Today = create.Allowed, u.today
		out.Readable = slices.Clone(v.fields)
		if !v.gradeName {
			out.Readable = slices.DeleteFunc(out.Readable, func(key string) bool { return key == "gradeId" })
		}
		for _, filter := range []struct{ key, value string }{{"currency", q.Currency}, {"frequency", q.Frequency}, {"form", q.Form}, {"status", q.Status}} {
			if filter.value != "" && !has(v.fields, filter.key) {
				return denied()
			}
		}
		// Select today's period; before the first day show the first future
		// period. All filtering and counting use the same authorized columns.
		const selection = ` FROM compensation.grade_rate r
		JOIN LATERAL (SELECT p.* FROM compensation.grade_rate_period p
		  WHERE p.instance_id=r.instance_id AND p.rate_id=r.id AND (p.valid_period @> $3::date OR p.start_date>$3::date)
		  ORDER BY p.start_date LIMIT 1) p ON true
		JOIN compensation.grade_rate_revision v ON (v.instance_id,v.rate_id,v.record_id,v.revision)=(p.instance_id,p.rate_id,p.record_id,p.revision)
		WHERE r.instance_id=$1 AND r.enterprise_id=$2 AND
		($4='' OR ($5 AND strpos(lower(v.fact#>>'{data,grade,name}'),lower($4))>0) OR ($6 AND strpos(lower(v.fact#>>'{data,grade,code}'),lower($4))>0))
		AND ($7='' OR r.currency=$7) AND ($8='' OR r.frequency=$8)
		AND ($9='' OR v.fact#>>'{data,amounts,form}'=$9)
		AND ($10='' OR (CASE WHEN p.start_date>$3 THEN 'FUTURE' WHEN p.active THEN 'ACTIVE' ELSE 'INACTIVE' END)=$10)`
		args := []any{s.Instance, u.enterprise.ID, u.today, strings.TrimSpace(q.Q), v.gradeName, v.gradeCode, q.Currency, q.Frequency, q.Form, q.Status}
		if e = u.tx.QueryRow(ctx, "SELECT count(*)"+selection, args...).Scan(&out.Total); e != nil {
			return e
		}
		rows, e := u.tx.Query(ctx, `SELECT r.id::text,r.head::text,r.grade_id::text,r.currency,r.frequency,r.unit,r.places,v.fact,coalesce(p.until_date::text,'')`+selection+" ORDER BY r.id LIMIT $11 OFFSET $12", append(args, q.Size, (q.Page-1)*q.Size)...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var state app.RateState
			var rec d.Record[d.RateTerms]
			var raw []byte
			var until string
			if e = rows.Scan(&state.ID, &state.Version, &state.Key.GradeID, &state.Key.Currency, &state.Key.Frequency, &state.Key.Unit, &state.Places, &raw, &until); e != nil {
				return e
			}
			if e = json.Unmarshal(raw, &rec); e != nil {
				return e
			}
			rate := v.rate(state, rec, until)
			if rec.Start > u.today && has(v.fields, "status") {
				rate.Record.Status = "FUTURE"
			}
			out.Items = append(out.Items, rate)
		}
		return rows.Err()
	})
	return
}

func (s *Store) ReadRate(ctx context.Context, cred foundation.Credentials, id, record, day string) (out c.RateDetail, err error) {
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		v, e := u.visibility(ctx, false)
		if e != nil {
			return e
		}
		state, e := s.loadRate(ctx, u, id, false)
		if e != nil {
			return e
		}
		if day != "" && !foundation.ValidDate(day) || record != "" && day != "" {
			return d.Invalid("businessAt", "请选择一项有效日期或记录")
		}
		asOf := day
		if asOf == "" {
			asOf = u.today
		}
		pos := state.Records.At(asOf)
		if day == "" && pos < 0 {
			pos = 0
		}
		if record != "" {
			pos = slices.IndexFunc(state.Records, func(r d.Record[d.RateTerms]) bool { return r.ID == record })
		}
		if pos < 0 {
			return unavailable()
		}
		r := state.Records[pos]
		out.Rate = v.rate(state, r, state.Records.Period(pos).Until)
		out.Today = u.today
		out.Dates = []c.DateNode{}
		if has(v.fields, "effectiveDate") {
			for _, item := range state.Records {
				out.Dates = append(out.Dates, c.DateNode{ID: item.ID, Start: item.Start, Active: has(v.fields, "status") && item.Active})
			}
		}
		h, e := u.decision(ctx, "graderate.history.read", nil)
		if e != nil {
			return e
		}
		out.History = h.Allowed
		out.Actions = map[string][]string{}
		out.Windows = map[string]c.Window{}
		out.Reasons = map[string][]string{}
		for _, action := range app.RateActions {
			if action == "DELETE" && pos == 0 || (action == "ADD" || action == "DEACTIVATE") && !r.Active || action == "REACTIVATE" && r.Active {
				continue
			}
			grant, e := u.decision(ctx, "graderate."+strings.ToLower(action), nil)
			if e != nil {
				return e
			}
			if grant.Allowed {
				out.Actions[action] = visibleWrites(v.fields, grant.WriteFields)
				if has(out.Actions[action], "reason") && action != "DELETE" {
					op := action
					if op == "CORRECT" {
						op = r.Operation
					}
					out.Reasons[action] = d.RateReasons(op)
				}
				if has(v.fields, "effectiveDate") && action != "DELETE" {
					out.Windows[action] = app.MaintenanceWindow(state.Records, pos, action, u.today, u.enterprise.ValidFrom)
				}
			}
		}
		return nil
	})
	return
}

func (s *Store) RateHistory(ctx context.Context, cred foundation.Credentials, id, version string) (out []c.Change, err error) {
	out = []c.Change{}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		v, e := u.visibility(ctx, true)
		if e != nil {
			return e
		}
		if !foundation.ValidID(id) || version != "" && !foundation.ValidID(version) {
			return unavailable()
		}
		rows, e := u.tx.Query(ctx, `SELECT h.id::text,coalesce(h.predecessor::text,''),h.action,h.recorded_at,h.operator,r.grade_id::text,r.currency,r.frequency,r.unit,r.places,a.fact,b.fact,coalesce(h.until_date::text,''),coalesce(h.previous_until::text,'')
		FROM compensation.grade_rate r JOIN compensation.grade_rate_version h ON r.instance_id=h.instance_id AND r.id=h.rate_id
		JOIN compensation.grade_rate_revision a ON (a.instance_id,a.rate_id,a.record_id,a.revision)=(h.instance_id,h.rate_id,h.record_id,h.revision)
		LEFT JOIN compensation.grade_rate_revision b ON (b.instance_id,b.rate_id,b.record_id,b.revision)=(h.instance_id,h.rate_id,h.previous_record_id,h.previous_revision)
		WHERE r.instance_id=$1 AND r.enterprise_id=$2 AND r.id=$3 AND ($4::uuid IS NULL OR h.id=$4)
		ORDER BY h.recorded_at DESC,h.id`, s.Instance, u.enterprise.ID, id, optional(version))
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var change c.Change
			var key d.RateKey
			var recorded time.Time
			var op, after, before []byte
			var end, prevEnd string
			var places int
			if e = rows.Scan(&change.Version, &change.Previous, &change.Action, &recorded, &op, &key.GradeID, &key.Currency, &key.Frequency, &key.Unit, &places, &after, &before, &end, &prevEnd); e != nil {
				return e
			}
			change.Currency, change.Frequency, change.Unit = v.value("currency", key.Currency), v.value("frequency", key.Frequency), v.value("unit", key.Unit)
			if has(v.fields, "precision") {
				change.Precision = &places
			}
			change.RecordedAt = recorded.UTC().Format(time.RFC3339Nano)
			if has(v.fields, "operator") {
				var actor access.OperatorSnapshot
				if e = json.Unmarshal(op, &actor); e != nil {
					return e
				}
				if actor.DisplayName != nil {
					change.Operator = *actor.DisplayName
				} else if actor.Login != nil {
					change.Operator = *actor.Login
				}
			}
			project := func(raw []byte, until string) (*c.RateRecord, error) {
				if raw == nil {
					return nil, nil
				}
				var fact d.Record[d.RateTerms]
				if e := json.Unmarshal(raw, &fact); e != nil {
					return nil, e
				}
				rec := v.record(key, fact, until)
				return &rec, nil
			}
			if change.Action != "DELETE" {
				if change.After, e = project(after, end); e != nil {
					return e
				}
			}
			if change.Before, e = project(before, prevEnd); e != nil {
				return e
			}
			out = append(out, change)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		if len(out) == 0 {
			return unavailable()
		}
		return nil
	})
	return
}

func (s *Store) Menu(ctx context.Context, cred foundation.Credentials) (out c.MenuAccess, err error) {
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		grant, e := u.decision(ctx, "graderate.read", nil)
		out.GradeRates = grant.Allowed
		return e
	})
	return
}

func visibleWrites(readable, writable []string) []string {
	fields := []string{}
	for _, field := range writable {
		if has(readable, field) {
			fields = append(fields, field)
		}
	}
	return fields
}
