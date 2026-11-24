package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	domain "chnhcm.local/chnhcm/internal/contexts/recruiting/domain"
	r "chnhcm.local/chnhcm/internal/contexts/recruiting/public"
	st "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool         *pgxpool.Pool
	Instance     string
	Guard        func(context.Context, pgx.Tx, access.Credentials) (access.Guard, error)
	Allowed      func(context.Context, pgx.Tx, string, string, string, string, string, bool) (bool, error)
	Enterprise   org.EnterpriseFloor
	Employers    org.LegalPurposes
	Person       func(context.Context, f.Credentials, string) (string, error)
	Arrangement  func(context.Context, f.Credentials, st.AssignmentCheck, string, string, string, string) ([]st.AssignmentSegment, error)
	LockPosition func(context.Context, string) error
}
type unit struct {
	*Store
	tx    pgx.Tx
	actor string
	cred  f.Credentials
}

func (s *Store) within(ctx context.Context, c f.Credentials, write bool, run func(context.Context, *unit) error) error {
	c.Write = write
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	g, e := s.Guard(ctx, tx, c)
	if e != nil {
		return e
	}
	if e = run(dbtx.With(ctx, tx), &unit{s, tx, g.Principal().ID, c}); e != nil {
		return e
	}
	if e = g.RecordSessionActivity(ctx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (u *unit) allowed(ctx context.Context, action string, t r.Terms, candidate bool) (bool, error) {
	return u.Allowed(ctx, u.tx, u.actor, action, t.Person, t.Employer, t.Department, candidate)
}
func (u *unit) require(ctx context.Context, action string, t r.Terms, candidate bool) error {
	ok, e := u.allowed(ctx, action, t, candidate)
	if e != nil {
		return e
	}
	if !ok {
		return f.Fail(403, "OFFER_DENIED", "当前权限不允许办理此录用")
	}
	return nil
}
func (s *Store) PositionOpened(ctx context.Context, instance, position, source, day string) error {
	tx, ok := dbtx.From(ctx)
	if !ok || s.Instance != instance {
		return errors.New("recruiting intake requires the position creation transaction")
	}
	_, e := tx.Exec(ctx, "INSERT INTO recruiting.position_intake(instance_id,position_id,creation_source,version,since) VALUES($1,$2,$3,$4,$5::date)", instance, position, source, f.NewID(), day)
	return e
}
func (s *Store) Usage(ctx context.Context, instance, position string) (out r.Usage, err error) {
	tx, ok := dbtx.From(ctx)
	if !ok || instance != s.Instance {
		return out, errors.New("recruiting usage requires the owning transaction")
	}
	err = tx.QueryRow(ctx, "SELECT since::text,version FROM recruiting.position_intake WHERE instance_id=$1 AND position_id=$2", instance, position).Scan(&out.From, &out.Coverage)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	rows, e := tx.Query(ctx, `SELECT lower(period)::text,coalesce(upper(period)::text,''),s.version,adoption->>'policy',effect,adoption->>'departmentId',adoption->>'jobId'
    FROM recruiting.offer_head h JOIN recruiting.offer_segment s USING(instance_id,offer_id,version) WHERE h.instance_id=$1 AND s.position_id=$2`, instance, position)
	if e != nil {
		return out, e
	}
	out.Segments, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (v r.Reservation, e error) {
		e = row.Scan(&v.From, &v.Until, &v.Source, &v.Policy, &v.Effect, &v.Department, &v.Job)
		return
	})
	return
}

const offerHead = `SELECT o.terms,v.version,v.operation,v.status,v.business_date::text,v.basis,to_char(v.recorded_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
 FROM recruiting.offer o JOIN recruiting.offer_head h ON (h.instance_id,h.offer_id)=(o.instance_id,o.id)
 JOIN recruiting.offer_revision v ON (v.instance_id,v.offer_id,v.version)=(h.instance_id,h.offer_id,coalesce(nullif($3,'')::uuid,h.version)) WHERE o.instance_id=$1 AND o.id=$2`

func (u *unit) load(ctx context.Context, id, version string) (v r.Offer, err error) {
	v.ID = id
	err = u.tx.QueryRow(ctx, offerHead, u.Instance, id, version).Scan(&v.PublishedTerms, &v.Version, &v.Operation, &v.Status, &v.BusinessDate, &v.Basis, &v.RecordedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, f.Fail(404, "OFFER_UNAVAILABLE", "所选录用不可用")
	}
	if err != nil {
		return
	}
	rows, e := u.tx.Query(ctx, `SELECT adoption FROM recruiting.offer_segment WHERE instance_id=$1 AND offer_id=$2 AND version=$3 ORDER BY lower(period)`, u.Instance, id, v.Version)
	if e != nil {
		return v, e
	}
	v.Segments, err = pgx.CollectRows(rows, pgx.RowTo[r.AssignmentSegment])
	return
}
func (u *unit) visible(ctx context.Context, v r.Offer, action string) (bool, error) {
	ok, e := u.allowed(ctx, action, v.Terms, false)
	if e != nil || !ok {
		return ok, e
	}
	for _, seg := range v.Segments {
		if seg.DepartmentID != "" && seg.DepartmentID != v.Department {
			t := v.Terms
			t.Department = seg.DepartmentID
			if ok, e = u.allowed(ctx, action, t, false); e != nil || !ok {
				return ok, e
			}
		}
	}
	return true, nil
}
func held(v r.Offer, day string) bool {
	return slices.ContainsFunc(v.Segments, func(s r.AssignmentSegment) bool { return s.Seats > 0 && (s.Until == "" || s.Until > day) })
}
func (s *Store) Entry(ctx context.Context, c f.Credentials, person string) (out r.Entry, err error) {
	if person != "" && !f.ValidID(person) {
		return out, f.Fail(400, "OFFER_INPUT", "人员无效")
	}
	err = s.within(ctx, c, false, func(ctx context.Context, u *unit) error {
		var e error
		out.CanRead, e = u.allowed(ctx, "read", r.Terms{Person: person}, true)
		if e != nil {
			return e
		}
		out.CanIssue, e = u.allowed(ctx, "issue", r.Terms{Person: person}, true)
		return e
	})
	return
}
func (s *Store) List(ctx context.Context, c f.Credentials, person string) (out r.Directory, err error) {
	if person != "" && !f.ValidID(person) {
		return out, f.Fail(400, "OFFER_INPUT", "人员无效")
	}
	out.Offers = []r.Offer{}
	err = s.within(ctx, c, false, func(ctx context.Context, u *unit) error {
		var e error
		out.CanIssue, e = u.allowed(ctx, "issue", r.Terms{Person: person}, true)
		if e != nil {
			return e
		}
		if e = u.require(ctx, "read", r.Terms{Person: person}, true); e != nil {
			return e
		}
		rows, e := u.tx.Query(ctx, `SELECT id FROM recruiting.offer WHERE instance_id=$1 AND ($2='' OR person_id=nullif($2,'')::uuid) ORDER BY number,id`, s.Instance, person)
		if e != nil {
			return e
		}
		ids, e := pgx.CollectRows(rows, pgx.RowTo[string])
		if e != nil {
			return e
		}
		for _, id := range ids {
			v, e := u.load(ctx, id, "")
			if e != nil {
				return e
			}
			ok, e := u.visible(ctx, v, "read")
			if e != nil {
				return e
			}
			if ok {
				out.Offers = append(out.Offers, v)
			}
		}
		return nil
	})
	return
}
func (s *Store) Get(ctx context.Context, c f.Credentials, id, version string) (out r.Offer, err error) {
	if !f.ValidID(id) || version != "" && !f.ValidID(version) {
		return out, f.Fail(400, "OFFER_INPUT", "录用或版本无效")
	}
	err = s.within(ctx, c, false, func(ctx context.Context, u *unit) error {
		var e error
		out, e = u.load(ctx, id, version)
		if e != nil {
			return e
		}
		ok, e := u.visible(ctx, out, "read")
		if e != nil {
			return e
		}
		if !ok {
			return f.Fail(403, "OFFER_DENIED", "当前权限无法读取该录用")
		}
		rows, e := u.tx.Query(ctx, `SELECT version,operation,status,business_date::text,basis,to_char(recorded_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') FROM recruiting.offer_revision WHERE instance_id=$1 AND offer_id=$2 ORDER BY recorded_at DESC`, s.Instance, id)
		if e != nil {
			return e
		}
		revisions, e := pgx.CollectRows(rows, pgx.RowToStructByPos[r.Revision])
		if e != nil {
			return e
		}
		for _, v := range revisions {
			item, e := u.load(ctx, id, v.Version)
			if e != nil {
				return e
			}
			ok, e := u.visible(ctx, item, "read")
			if e != nil {
				return e
			}
			if ok {
				out.Revisions = append(out.Revisions, v)
			}
		}
		if version != "" {
			return nil
		}
		for _, op := range []string{"accept", "decline", "release", "reserve"} {
			right := op
			if op == "decline" {
				right = "accept"
			}
			ok, e := u.visible(ctx, out, right)
			if e != nil {
				return e
			}
			if op == "reserve" && !out.ReservationRequired {
				continue
			}
			if _, e = domain.Next(out.Status, op, held(out, out.Arrival)); ok && e == nil {
				out.Actions = append(out.Actions, op)
			}
		}
		return nil
	})
	return
}
func (u *unit) submission(ctx context.Context, key string) (r.Receipt, string, error) {
	var v r.Receipt
	var digest string
	_, e := u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "recruiting.submission:"+u.Instance+u.actor+key)
	if e != nil {
		return v, "", e
	}
	e = u.tx.QueryRow(ctx, "SELECT coalesce(digest,''),coalesce(offer_id::text,''),coalesce(version::text,'') FROM recruiting.offer_submission WHERE instance_id=$1 AND actor=$2 AND submission_id=$3", u.Instance, u.actor, key).Scan(&digest, &v.ID, &v.Version)
	if e == nil {
		v.Outcome = "accepted"
		if v.ID == "" {
			v.Outcome = "not_applied"
		}
	}
	if errors.Is(e, pgx.ErrNoRows) {
		e = nil
	}
	return v, digest, e
}
func (u *unit) readReceipt(ctx context.Context, v r.Receipt) error {
	if v.ID == "" {
		return nil
	}
	saved, err := u.load(ctx, v.ID, v.Version)
	if err != nil {
		return err
	}
	ok, err := u.visible(ctx, saved, "read")
	if err != nil {
		return err
	}
	if !ok {
		return f.Fail(403, "OFFER_DENIED", "当前无法读取提交结果")
	}
	return nil
}

func (s *Store) Resolve(ctx context.Context, c f.Credentials, key string) (out r.Receipt, err error) {
	if !f.ValidID(key) {
		return out, f.Fail(400, "OFFER_KEY", "提交号无效")
	}
	err = s.within(ctx, c, true, func(ctx context.Context, u *unit) error {
		v, _, e := u.submission(ctx, key)
		if e != nil {
			return e
		}
		if v.Outcome != "" {
			if e := u.readReceipt(ctx, v); e != nil {
				return e
			}
			out = v
			return nil
		}
		_, e = u.tx.Exec(ctx, "INSERT INTO recruiting.offer_submission(instance_id,actor,submission_id) VALUES($1,$2,$3)", s.Instance, u.actor, key)
		out.Outcome = "not_applied"
		return e
	})
	out.SubmissionID = key
	return
}
func (s *Store) Submit(ctx context.Context, c f.Credentials, id, key string, in r.Change) (out r.Receipt, err error) {
	if !f.ValidID(key) || id != "" && !f.ValidID(id) || !f.ValidDate(in.BusinessDate) || in.Basis == "" || utf8.RuneCountInString(in.Basis) > 1000 {
		return out, f.Fail(400, "OFFER_INPUT", "请核对录用、业务日期及办理依据")
	}
	in.Basis = strings.TrimSpace(in.Basis)
	if in.Basis == "" {
		return out, f.Fail(400, "OFFER_INPUT", "请填写办理依据")
	}
	raw, e := json.Marshal([]any{id, in})
	if e != nil {
		return out, e
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	err = s.within(ctx, c, true, func(ctx context.Context, u *unit) error {
		v, prior, e := u.submission(ctx, key)
		if e != nil {
			return e
		}
		if v.Outcome != "" {
			if v.Outcome != "accepted" || prior != digest {
				return f.Fail(409, "OFFER_KEY", "该提交已结束，请重新办理")
			}
			if e = u.readReceipt(ctx, v); e != nil {
				return e
			}
			out = v
			return nil
		}
		v, e = u.change(ctx, id, in)
		if e != nil {
			return e
		}
		_, e = u.tx.Exec(ctx, "INSERT INTO recruiting.offer_submission(instance_id,actor,submission_id,digest,offer_id,version) VALUES($1,$2,$3,$4,$5,$6)", s.Instance, u.actor, key, digest, v.ID, v.Version)
		out = v
		return e
	})
	out.SubmissionID = key
	return
}
func (u *unit) change(ctx context.Context, id string, in r.Change) (out r.Receipt, err error) {
	var v r.Offer
	right := in.Operation
	if right == "decline" {
		right = "accept"
	}
	if id == "" {
		if in.Operation != "issue" || in.Expected != "" || in.Terms == nil {
			return out, f.Fail(400, "OFFER_INPUT", "请核对新录用安排")
		}
		v.Terms = *in.Terms
		v.Number = strings.TrimSpace(v.Number)
		v.ID = f.NewID()
	} else {
		if in.Terms != nil || in.Operation == "issue" || !f.ValidID(in.Expected) {
			return out, f.Fail(400, "OFFER_INPUT", "请引用正在办理的录用版本")
		}
		_, e := u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", "recruiting.offer:"+u.Instance+id)
		if e != nil {
			return out, e
		}
		v, e = u.load(ctx, id, "")
		if e != nil {
			return out, e
		}
		ok, e := u.visible(ctx, v, right)
		if e != nil {
			return out, e
		}
		if !ok {
			return out, f.Fail(403, "OFFER_DENIED", "当前权限不允许办理此录用")
		}
		if v.Version != in.Expected {
			return out, f.Fail(409, "OFFER_CHANGED", "录用版本已变化，请重读")
		}
	}
	span, e := domain.ValidateTerms(domain.Terms(v.Terms))
	if e != nil {
		return out, e
	}
	if e = u.require(ctx, right, v.Terms, id == ""); e != nil {
		return out, e
	}
	if in.BusinessDate > v.Arrival && in.Operation != "release" && in.Operation != "decline" {
		return out, f.Fail(409, "OFFER_DATE", "办理日期超过约定到岗日")
	}
	status, e := domain.Next(v.Status, in.Operation, held(v, in.BusinessDate))
	if e != nil {
		return out, e
	}
	if in.Operation == "issue" && in.BusinessDate > span.From {
		return out, f.Fail(409, "OFFER_DATE", "预留安排不能早于本次发出录用的日期")
	}
	if id != "" && in.BusinessDate < v.BusinessDate {
		return out, f.Fail(409, "OFFER_DATE", "办理日期早于上一笔录用决定")
	}
	if in.Operation == "issue" || in.Operation == "reserve" || in.Operation == "accept" {
		if in.Operation != "issue" && in.BusinessDate > span.From {
			span.From = in.BusinessDate
		}
		name, e := u.Person(ctx, u.cred, v.Person)
		if e != nil {
			return out, e
		}
		v.PersonName = name
		ent, e := u.Enterprise.StaffingEnterprise(ctx, true)
		if e != nil {
			return out, e
		}
		legal, e := u.Employers.CheckPurpose(ctx, u.cred, v.Employer, org.PurposeCheck{Role: "employer", Period: f.Period{From: v.Arrival, Until: span.Until}})
		if e != nil {
			return out, e
		}
		if len(legal) == 0 {
			return out, f.Fail(409, "OFFER_EMPLOYER", "录用期间缺少用人单位依据")
		}
		if in.Operation == "issue" {
			v.EmployerName = legal[0].EntityName
			for _, source := range legal {
				v.EmployerAdoption = append(v.EmployerAdoption, r.LegalAdoption(source))
			}
		}
		check := st.AssignmentCheck{Enterprise: ent.ID, Category: "Employee", Purpose: "recruit", Period: span, Position: v.Position, Department: v.Department, Job: v.Job}
		sources, e := u.Arrangement(ctx, u.cred, check, v.Version, v.Person, v.Employer, right)
		if e != nil {
			return out, e
		}
		var segs []r.AssignmentSegment
		for _, source := range sources {
			seg := r.AssignmentSegment(source)
			if seg.DepartmentID == "" {
				seg.DepartmentID = check.Department
			}
			if seg.JobID == "" {
				seg.JobID = check.Job
			}
			segs = append(segs, seg)
			dept := seg.DepartmentID
			if dept == "" {
				dept = v.Department
			}
			scope := v.Terms
			scope.Department = dept
			if in.Operation == "issue" && seg.Seats > 0 {
				if e = u.require(ctx, "reserve", scope, false); e != nil {
					return out, e
				}
			}
			if e = u.require(ctx, right, scope, false); e != nil {
				return out, e
			}
		}
		if v.Position != "" {
			v.Department, v.Job = segs[0].DepartmentID, segs[0].JobID
		}
		switch in.Operation {
		case "accept":
			if !domain.PromisedArrangement(bindings(v.Segments), bindings(segs)) {
				return out, f.Fail(409, "OFFER_RESERVATION", "录用缺少与约定岗位及期间一致的安排，请先核对预留")
			}
		case "reserve":
			if !slices.ContainsFunc(segs, func(s r.AssignmentSegment) bool { return s.Seats > 0 }) {
				return out, f.Fail(409, "OFFER_RESERVATION", "此录用期间无需预留")
			}
			v.Segments = append(beforeDate(v.Segments, span.From), segs...)
		default:
			v.ReservationRequired = slices.ContainsFunc(segs, func(s r.AssignmentSegment) bool { return s.Seats > 0 })
			v.Segments = segs
		}
	} else if in.Operation == "release" || in.Operation == "decline" {
		if v.Position != "" {
			if e = u.LockPosition(ctx, v.Position); e != nil {
				return out, e
			}
		}
		v.Segments = beforeDate(v.Segments, in.BusinessDate)
	}
	if in.Operation == "issue" {
		_, e = u.tx.Exec(ctx, `INSERT INTO recruiting.offer(instance_id,id,number,person_id,employer_id,department_id,job_id,position_id,arrival,reserve_from,until_date,terms) VALUES($1,$2,$3,$4,$5,$6,$7,nullif($8,'')::uuid,$9::date,$10::date,nullif($11,'')::date,$12)`, u.Instance, v.ID, v.Number, v.Person, v.Employer, v.Department, v.Job, v.Position, v.Arrival, v.ReserveFrom, span.Until, v.PublishedTerms)
		var fault *pgconn.PgError
		if errors.As(e, &fault) && fault.Code == "23505" {
			return out, f.Fail(409, "OFFER_NUMBER", "录用编号已使用")
		}
		if e != nil {
			return out, e
		}
	}
	return u.append(ctx, v, in.Operation, status, in.BusinessDate, in.Basis)
}
func bindings(segments []r.AssignmentSegment) []domain.Binding {
	out := make([]domain.Binding, 0, len(segments))
	for _, segment := range segments {
		out = append(out, domain.Binding{Period: segment.Period, Position: segment.Position, DepartmentID: segment.DepartmentID, JobID: segment.JobID, Seats: segment.Seats})
	}
	return out
}
func beforeDate(segs []r.AssignmentSegment, day string) []r.AssignmentSegment {
	var result []r.AssignmentSegment
	for _, s := range segs {
		if s.From >= day {
			continue
		}
		if s.Until == "" || s.Until > day {
			s.Until = day
		}
		result = append(result, s)
	}
	return result
}
func (u *unit) append(ctx context.Context, v r.Offer, op, status, day, basis string) (r.Receipt, error) {
	version := f.NewID()
	_, e := u.tx.Exec(ctx, `INSERT INTO recruiting.offer_revision(instance_id,offer_id,version,previous,operation,status,business_date,basis,actor) VALUES($1,$2,$3,nullif($4,'')::uuid,$5,$6,$7::date,$8,$9)`, u.Instance, v.ID, version, v.Version, op, status, day, basis, u.actor)
	if e != nil {
		return r.Receipt{}, e
	}
	for _, seg := range v.Segments {
		effect := "none"
		if seg.Seats > 0 {
			effect = "reserved"
		}
		_, e = u.tx.Exec(ctx, `INSERT INTO recruiting.offer_segment(instance_id,offer_id,version,period,position_id,effect,adoption) VALUES($1,$2,$3,daterange($4::date,nullif($5,'')::date,'[)'),nullif($6,'')::uuid,$7,$8)`, u.Instance, v.ID, version, seg.From, seg.Until, seg.Position, effect, seg)
		if e != nil {
			return r.Receipt{}, e
		}
	}
	_, e = u.tx.Exec(ctx, `INSERT INTO recruiting.offer_head(instance_id,offer_id,version) VALUES($1,$2,$3) ON CONFLICT(instance_id,offer_id) DO UPDATE SET version=excluded.version`, u.Instance, v.ID, version)
	if e != nil {
		return r.Receipt{}, e
	}
	_, e = u.tx.Exec(ctx, "INSERT INTO recruiting.outbox VALUES($1,$2,$3,$4)", u.Instance, f.NewID(), v.ID, version)
	return r.Receipt{Outcome: "accepted", ID: v.ID, Version: version}, e
}
