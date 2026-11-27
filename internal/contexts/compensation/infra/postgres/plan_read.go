package postgres

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/compensation/application"
	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func (u *unit) planOptions(ctx context.Context, fields []string) (c.PlanOptions, bool, error) {
	out := c.PlanOptions{ScopeDimensions: []string{}, Units: map[string]string{}}
	for _, f := range []string{"MONTH", "YEAR", "HOUR", "ONCE"} {
		out.Units[f], _ = d.FrequencyUnit(f)
	}
	if has(fields, "scope") {
		for _, dim := range []string{"legalEntity", "jobFamily", "grade"} {
			g, err := u.decision(ctx, sourcePermission(dim), nil)
			if err != nil {
				return out, false, err
			}
			if g.Allowed && has(g.ReadFields, "name") {
				out.ScopeDimensions = append(out.ScopeDimensions, dim)
			}
		}
	}
	if !has(fields, "rates") {
		return out, false, nil
	}
	g, err := u.decision(ctx, "graderate.reference", nil)
	return out, g.Allowed && planRateFields(g.ReadFields), err
}

func planWriteFields(read, write []string, canReference bool) []string {
	fields := visibleWrites(read, write)
	return slices.DeleteFunc(fields, func(key string) bool {
		return key == "rates" && !canReference || key == "percentage" && !has(read, "amount")
	})
}

func (s *Store) PlanContext(ctx context.Context, cred foundation.Credentials) (out c.PlanContext, err error) {
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		read, e := u.requireObjectRead(ctx, planCommands)
		if e != nil {
			return e
		}
		grant, e := u.require(ctx, "compplan.create", nil)
		if e != nil {
			return e
		}
		out.Reasons = d.PlanReasons("CREATE")
		var canReference bool
		out.PlanOptions, canReference, e = u.planOptions(ctx, visibleWrites(read.ReadFields, grant.WriteFields))
		if e != nil {
			return e
		}
		out.Writable = planWriteFields(read.ReadFields, grant.WriteFields, canReference)
		out.DefaultDate = foundation.DefaultEffectiveDate
		return nil
	})
	return
}
func (s *Store) ListPlans(ctx context.Context, cred foundation.Credentials, q c.RateQuery) (out c.PlanList, err error) {
	out.Items = []c.Plan{}
	if q.Page < 1 || q.Size < 1 || q.Size > 100 || q.Page > 1000000 {
		return out, d.Invalid("page", "页码或每页条数无效")
	}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		v, e := u.planView(ctx, false)
		if e != nil {
			return e
		}
		create, e := u.decision(ctx, "compplan.create", nil)
		if e != nil {
			return e
		}
		out.Today, out.Create, out.Readable = u.today, create.Allowed, v.fields
		if q.Status != "" && !has(v.fields, "status") {
			return denied()
		}
		const selection = ` FROM compensation.plan h JOIN LATERAL (
		 SELECT p.* FROM compensation.plan_period p WHERE p.instance_id=h.instance_id AND p.plan_id=h.id AND (p.valid_period @> $3::date OR p.start_date>$3::date) ORDER BY p.start_date LIMIT 1
		) p ON true JOIN compensation.plan_revision r ON (r.instance_id,r.plan_id,r.record_id,r.revision)=(p.instance_id,p.plan_id,p.record_id,p.revision)
		WHERE h.instance_id=$1 AND h.enterprise_id=$2 AND ($4='' OR ($5 AND strpos(lower(h.code),lower($4))>0) OR ($6 AND strpos(lower(p.name),lower($4))>0))
		AND ($7='' OR (CASE WHEN p.start_date>$3 THEN 'FUTURE' WHEN p.active THEN 'ACTIVE' ELSE 'INACTIVE' END)=$7)`
		args := []any{s.Instance, u.enterprise.ID, u.today, strings.TrimSpace(q.Q), has(v.fields, "code"), has(v.fields, "name"), q.Status}
		if e = u.tx.QueryRow(ctx, "SELECT count(*)"+selection, args...).Scan(&out.Total); e != nil {
			return e
		}
		rows, e := u.tx.Query(ctx, `SELECT h.id::text,h.head::text,h.code,r.fact,coalesce(p.until_date::text,'')`+selection+" ORDER BY CASE WHEN $5 THEN h.code END,h.id LIMIT $8 OFFSET $9", append(args, q.Size, (q.Page-1)*q.Size)...)
		if e != nil {
			return e
		}
		type entry struct {
			id, head, code, until string
			record                d.Record[d.PlanTerms]
		}
		var found []entry
		for rows.Next() {
			var x entry
			var raw []byte
			if e = rows.Scan(&x.id, &x.head, &x.code, &raw, &x.until); e != nil {
				rows.Close()
				return e
			}
			if e = json.Unmarshal(raw, &x.record); e != nil {
				rows.Close()
				return e
			}
			found = append(found, x)
		}
		rows.Close()
		if e = rows.Err(); e != nil {
			return e
		}
		for _, x := range found {
			record, e := s.projectPlan(ctx, u, x.id, v, x.record, x.until, u.today, false, false)
			if e != nil {
				return e
			}
			if x.record.Start > u.today && has(v.fields, "status") {
				record.Status = "FUTURE"
			}
			p := c.Plan{ID: x.id, Version: x.head, Record: record}
			if has(v.fields, "code") {
				p.Code = x.code
			}
			out.Items = append(out.Items, p)
		}
		return nil
	})
	return
}
func (s *Store) ReadPlan(ctx context.Context, cred foundation.Credentials, id, record, day string) (out c.PlanDetail, err error) {
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		v, e := u.planView(ctx, false)
		if e != nil {
			return e
		}
		state, e := s.loadPlan(ctx, u, id, false)
		if e != nil {
			return e
		}
		if day != "" && !foundation.ValidDate(day) || day != "" && record != "" {
			return d.Invalid("businessAt", "请选择一个适用日期或一条生效记录")
		}
		atDay := day
		if atDay == "" {
			atDay = u.today
		}
		pos := state.Records.At(atDay)
		if record != "" {
			pos = slices.IndexFunc(state.Records, func(r d.Record[d.PlanTerms]) bool { return r.ID == record })
			if pos >= 0 {
				atDay = state.Records[pos].Start
			}
		}
		if day == "" && record == "" && pos < 0 {
			pos = 0
			atDay = state.Records[0].Start
		}
		if pos < 0 {
			return unavailable()
		}
		r := state.Records[pos]
		out.ID, out.Version, out.Today, out.BusinessAt, out.Readable = id, state.Version, u.today, atDay, v.fields
		if has(v.fields, "code") {
			out.Code = state.Code
		}
		out.Record, e = s.projectPlan(ctx, u, id, v, r, state.Records.Period(pos).Until, atDay, false, true)
		if e != nil {
			return e
		}
		out.Dates = []c.DateNode{}
		if has(v.fields, "effectiveDate") {
			for _, rr := range state.Records {
				out.Dates = append(out.Dates, c.DateNode{ID: rr.ID, Start: rr.Start, Active: has(v.fields, "status") && rr.Active})
			}
		}
		h, e := u.decision(ctx, "compplan.history.read", nil)
		if e != nil {
			return e
		}
		out.History = h.Allowed
		var canReference bool
		out.PlanOptions, canReference, e = u.planOptions(ctx, v.fields)
		if e != nil {
			return e
		}
		out.Actions = map[string][]string{}
		out.Reasons = map[string][]string{}
		out.Windows = map[string]c.Window{}
		for _, action := range app.RateActions {
			if action == "DELETE" && pos == 0 || (action == "ADD" || action == "DEACTIVATE") && !r.Active || action == "REACTIVATE" && r.Active {
				continue
			}
			g, e := u.decision(ctx, "compplan."+strings.ToLower(action), nil)
			if e != nil {
				return e
			}
			if !g.Allowed {
				continue
			}
			fields := planWriteFields(v.fields, g.WriteFields, canReference)
			if out.Record.Scope != nil && out.Record.Scope.Restricted {
				fields = slices.DeleteFunc(fields, func(k string) bool { return k == "scope" })
			}
			out.Actions[action] = fields
			if action != "DELETE" {
				op := action
				if op == "CORRECT" {
					op = r.Operation
				}
				if has(fields, "reason") {
					out.Reasons[action] = d.PlanReasons(op)
				}
				if has(v.fields, "effectiveDate") {
					out.Windows[action] = app.MaintenanceWindow(state.Records, pos, action, u.today, u.enterprise.ValidFrom)
				}
			}
		}
		return nil
	})
	return
}
func (s *Store) PlanHistory(ctx context.Context, cred foundation.Credentials, id, version string) (out []c.PlanChange, err error) {
	if !foundation.ValidID(id) || version != "" && !foundation.ValidID(version) {
		return nil, unavailable()
	}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		v, e := u.planView(ctx, true)
		if e != nil {
			return e
		}
		rows, e := u.tx.Query(ctx, `SELECT h.id::text,h.action,h.recorded_at,h.operator,p.code,a.fact,b.fact,coalesce(h.until_date::text,''),coalesce(h.previous_until::text,'')
		FROM compensation.plan p JOIN compensation.plan_version h ON (p.instance_id,p.id)=(h.instance_id,h.plan_id)
		JOIN compensation.plan_revision a ON (a.instance_id,a.plan_id,a.record_id,a.revision)=(h.instance_id,h.plan_id,h.record_id,h.revision)
		LEFT JOIN compensation.plan_revision b ON (b.instance_id,b.plan_id,b.record_id,b.revision)=(h.instance_id,h.plan_id,h.previous_record_id,h.previous_revision)
		WHERE p.instance_id=$1 AND p.enterprise_id=$2 AND p.id=$3 AND ($4::text='' OR h.id::text=$4) ORDER BY h.recorded_at DESC,h.id`, s.Instance, u.enterprise.ID, id, version)
		if e != nil {
			return e
		}
		type entry struct {
			change           c.PlanChange
			after, before    []byte
			until, prevUntil string
		}
		var found []entry
		for rows.Next() {
			var v entry
			var recorded time.Time
			var actor []byte
			if e = rows.Scan(&v.change.Version, &v.change.Action, &recorded, &actor, &v.change.Code, &v.after, &v.before, &v.until, &v.prevUntil); e != nil {
				rows.Close()
				return e
			}
			v.change.RecordedAt = recorded.UTC().Format(time.RFC3339Nano)
			var op access.OperatorSnapshot
			if err := json.Unmarshal(actor, &op); err != nil {
				rows.Close()
				return err
			}
			if op.DisplayName != nil {
				v.change.Operator = *op.DisplayName
			} else if op.Login != nil {
				v.change.Operator = *op.Login
			}
			found = append(found, v)
		}
		rows.Close()
		if e = rows.Err(); e != nil {
			return e
		}
		if len(found) == 0 {
			return unavailable()
		}
		for _, x := range found {
			if !has(v.fields, "code") {
				x.change.Code = ""
			}
			if !has(v.fields, "operator") {
				x.change.Operator = ""
			}
			project := func(raw []byte, end string) (*c.PlanRecord, error) {
				if len(raw) == 0 {
					return nil, nil
				}
				var r d.Record[d.PlanTerms]
				if e := json.Unmarshal(raw, &r); e != nil {
					return nil, e
				}
				p, e := s.projectPlan(ctx, u, id, v, r, end, r.Start, true, true)
				return &p, e
			}
			if x.change.Action != "DELETE" {
				x.change.After, e = project(x.after, x.until)
				if e != nil {
					return e
				}
			}
			x.change.Before, e = project(x.before, x.prevUntil)
			if e != nil {
				return e
			}
			out = append(out, x.change)
		}
		return nil
	})
	return
}

func (s *Store) RateCandidates(ctx context.Context, cred foundation.Credentials, day, query, currency, frequency string, limit int) (items []c.RateCandidate, err error) {
	items = []c.RateCandidate{}
	if !foundation.ValidDate(day) || limit < 1 || limit > 100 {
		return items, d.Invalid("businessAt", "请填写有效的适用日期和候选条数")
	}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		read, e := u.requireObjectRead(ctx, planCommands)
		if e != nil {
			return e
		}
		if !has(read.ReadFields, "components") || !has(read.ReadFields, "rates") {
			return denied()
		}
		allowed := false
		for _, op := range []string{"create", "add", "correct", "reactivate"} {
			g, e := u.decision(ctx, "compplan."+op, []string{"components", "rates"})
			if e != nil {
				return e
			}
			if g.Allowed {
				allowed = true
				break
			}
		}
		if !allowed {
			return denied()
		}
		ref, e := u.require(ctx, "graderate.reference", nil)
		if e != nil {
			return e
		}
		if !planRateFields(ref.ReadFields) {
			return denied()
		}
		rows, e := u.tx.Query(ctx, `SELECT r.id::text,r.grade_id::text,v.fact#>>'{data,grade,name}',v.fact#>>'{data,grade,code}',r.currency,r.frequency,r.unit,r.places,p.start_date::text,coalesce(p.until_date::text,'')
		FROM compensation.grade_rate r JOIN compensation.grade_rate_period p ON r.instance_id=p.instance_id AND r.id=p.rate_id
		JOIN compensation.grade_rate_revision v ON (v.instance_id,v.rate_id,v.record_id,v.revision)=(p.instance_id,p.rate_id,p.record_id,p.revision)
		WHERE r.instance_id=$1 AND r.enterprise_id=$2 AND p.active AND p.valid_period @> $3::date AND ($4='' OR strpos(lower((v.fact#>>'{data,grade,code}')||' '||(v.fact#>>'{data,grade,name}')),lower($4))>0) AND ($5='' OR r.currency=$5) AND ($6='' OR r.frequency=$6)
		ORDER BY v.fact#>>'{data,grade,code}',r.id LIMIT $7`, s.Instance, u.enterprise.ID, day, strings.TrimSpace(query), currency, frequency, limit)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var x c.RateCandidate
			if e = rows.Scan(&x.RateID, &x.GradeID, &x.Name, &x.Code, &x.Currency, &x.Frequency, &x.Unit, &x.Precision, &x.From, &x.Until); e != nil {
				return e
			}
			if !has(ref.ReadFields, "effectiveDate") {
				x.From, x.Until = "", ""
			}
			items = append(items, x)
		}
		return rows.Err()
	})
	return
}
func (s *Store) PlanApplicable(ctx context.Context, cred foundation.Credentials, id, day string, values map[string]string) (match bool, err error) {
	if !foundation.ValidDate(day) {
		return false, d.Invalid("businessAt", "请填写有效的适用日期")
	}
	err = s.within(ctx, cred, false, func(ctx context.Context, u *unit) error {
		read, e := u.requireObjectRead(ctx, planCommands)
		if e != nil {
			return e
		}
		if !has(read.ReadFields, "scope") {
			return denied()
		}
		st, e := s.loadPlan(ctx, u, id, false)
		if e != nil {
			return e
		}
		pos := st.Records.At(day)
		if pos < 0 || !st.Records[pos].Active {
			return nil
		}
		p, e := s.Scopes.ReadPlanScope(ctx, u.enterprise.ID, id, st.Records[pos].Data.ScopeVersion)
		if e != nil {
			return e
		}
		for _, m := range p.Members {
			g, e := u.require(ctx, sourcePermission(m.Dimension), nil)
			if e != nil {
				return e
			}
			if !has(g.ReadFields, "name") {
				return denied()
			}
		}
		match = s.Scopes.Matches(p, values)
		return nil
	})
	return
}
