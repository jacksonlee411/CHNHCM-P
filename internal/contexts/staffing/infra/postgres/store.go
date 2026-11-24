// Package postgres persists Staffing facts. Organization values are supplied
// exclusively through its public port in the same physical transaction.
package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	"chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	"chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres/generated"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"slices"
	"strings"
	"time"
)

type Store struct {
	HirePeople interface {
		hireprep.Personnel
		CheckHirePerson(context.Context, foundation.Credentials, string) error
	}
	HirePurposes      org.LegalPurposes
	EmploymentAllowed func(context.Context, pgx.Tx, string, string, string, string, string, bool) (bool, error)
	QuotaPolicies     func(context.Context, d.UsageIntent) ([]d.QuotaPolicy, error)
	Usage             func(context.Context, string, string) (d.QuotaUsage, error)
	Pool              *pgxpool.Pool
	Instance          string
	PositionDirectory org.PositionDirectory
	Organization      org.EnterpriseFloor
	Guard             func(context.Context, pgx.Tx, access.Credentials) (access.Guard, error)
	Clock             func() time.Time
}
type unit struct {
	credentials   foundation.Credentials
	lockRemaining time.Duration
	store         *Store
	tx            pgx.Tx
	guard         access.Guard
	now           time.Time
	enterprise    org.Enterprise
}

func (u *unit) Access() access.Guard { return u.guard }
func (u *unit) Now() time.Time       { return u.now }
func (s *Store) Within(ctx context.Context, c foundation.Credentials, write bool, fn func(app.Unit) error) error {
	return s.within(ctx, c, write, !write, func(u *unit) error { return fn(u) })
}

// Capturing a position set spans M02 membership and M03 timelines. All reads
// and the persisted draft must observe the same snapshot, including export.
func (s *Store) WithinSelection(ctx context.Context, c foundation.Credentials, fn func(app.Unit) error) error {
	return s.within(ctx, c, true, true, func(u *unit) error { return fn(u) })
}
func (s *Store) WithinQuota(ctx context.Context, c foundation.Credentials, write bool, fn func(app.QuotaTx) error) error {
	return s.within(ctx, c, write, !write, func(u *unit) error { return fn(u) })
}
func (s *Store) within(ctx context.Context, c foundation.Credentials, write, snapshot bool, fn func(*unit) error) error {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	// Session activity is infrastructure, updated before a pure read snapshot.
	// Updating it inside REPEATABLE READ causes concurrent reads of one session
	// to abort with 40001. Re-authorize inside the snapshot under its policy guard.
	if snapshot {
		if e := s.readActivity(ctx, c); e != nil {
			return e
		}
	}
	opts := pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
	if snapshot {
		opts.IsoLevel = pgx.RepeatableRead
	}
	tx, e := s.Pool.BeginTx(ctx, opts)
	if e != nil {
		return e
	}
	defer rollback(tx)
	if _, e = tx.Exec(ctx, "SET LOCAL jit=off; SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='5s'; SET LOCAL idle_in_transaction_session_timeout='12s'"); e != nil {
		return e
	}
	g, e := s.Guard(ctx, tx, c)
	if e != nil {
		return e
	}
	var now time.Time
	if s.Clock != nil {
		now = s.Clock()
	} else if e = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
		return e
	}
	u := &unit{credentials: c, store: s, tx: tx, guard: g, now: now, lockRemaining: 2 * time.Second}
	if e = fn(u); e != nil {
		return e
	}
	if write && !snapshot {
		if e = g.RecordSessionActivity(ctx); e != nil {
			return e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		code, msg := "M03_DEPENDENCY_UNAVAILABLE", "查询暂不可用，请重试"
		if write {
			code, msg = "M03_RESULT_PENDING", "提交结果尚未确认，请查看保存结果"
		}
		return foundation.Fail(503, code, msg)
	}
	return nil
}
func (s *Store) readActivity(ctx context.Context, c foundation.Credentials) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer rollback(tx)
	if _, e = tx.Exec(ctx, "SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='5s'"); e != nil {
		return e
	}
	g, e := s.Guard(ctx, tx, c)
	if e != nil {
		return e
	}
	if e = g.RecordSessionActivity(ctx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func rollback(tx pgx.Tx) {
	ctx, c := context.WithTimeout(context.Background(), 2*time.Second)
	defer c()
	_ = tx.Rollback(ctx)
}
func (u *unit) Enterprise(ctx context.Context, lock bool) (org.Enterprise, error) {
	if u.enterprise.ID != "" && !lock {
		return u.enterprise, nil
	}
	var v org.Enterprise
	var e error
	if lock {
		e = u.withLockBudget(ctx, func(bounded context.Context) error {
			var err error
			v, err = u.store.Organization.StaffingEnterprise(dbtx.With(bounded, u.tx), true)
			return err
		})
	} else {
		v, e = u.store.Organization.StaffingEnterprise(dbtx.With(ctx, u.tx), false)
	}
	if e == nil {
		u.enterprise = v
	}
	return v, e
}
func table(k d.Kind) string {
	switch k {
	case d.Kind(s.Quota):
		return "staffing.quota"
	case d.Kind(s.Hierarchy):
		return "staffing.position_hierarchy"
	case d.Position:
		return "staffing.position"
	case d.JobFunction:
		return "staffing.job_function"
	case d.Family:
		return "staffing.job_family"
	case d.Job:
		return "staffing.job"
	case d.Grade:
		return "staffing.grade"
	case d.Ladder:
		return "staffing.grade_ladder"
	default:
		panic("invalid internal staffing kind")
	}
}
func objectGuard(k d.Kind) string {
	if k == d.Family {
		return "staffing.family"
	}
	return "staffing." + string(k)
}
func lockKey(parts ...string) int64 {
	b, _ := json.Marshal(parts)
	sum := sha256.Sum256(b)
	return int64(binary.BigEndian.Uint64(sum[:8]))
}
func lock(ctx context.Context, tx pgx.Tx, shared bool, parts ...string) error {
	sql := "SELECT pg_advisory_xact_lock($1)"
	if shared {
		sql = "SELECT pg_advisory_xact_lock_shared($1)"
	}
	_, e := tx.Exec(ctx, sql, lockKey(parts...))
	return e
}
func (u *unit) LockSubmission(ctx context.Context, k d.Kind, id string) error {
	return u.guardLock(ctx, u.tx, false, "staffing.submission", string(k), u.store.Instance, u.guard.Principal().ID, id)
}
func (u *unit) Terminal(ctx context.Context, k d.Kind, id string) (out app.StoredResult, e error) {
	var b []byte
	e = u.tx.QueryRow(ctx, "SELECT digest,result FROM "+table(k)+"_terminal WHERE instance_id=$1 AND subject_id=$2 AND submission_id=$3", u.store.Instance, u.guard.Principal().ID, id).Scan(&out.Digest, &b)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, nil
	}
	if e != nil {
		return
	}
	e = json.Unmarshal(b, &out.Result)
	out.Exists = e == nil
	return
}
func (u *unit) Close(ctx context.Context, k d.Kind, id string, digest []byte, result s.Result) error {
	b, e := json.Marshal(result)
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, "INSERT INTO "+table(k)+"_terminal(instance_id,subject_id,submission_id,digest,result) VALUES($1,$2,$3,$4,$5)", u.store.Instance, u.guard.Principal().ID, id, digest, b)
	return e
}
func decode(b []byte, k d.Kind) (d.Timeline, error) {
	var t d.Timeline
	if e := json.Unmarshal(b, &t); e != nil {
		return t, e
	}
	if t.Kind != k || !foundation.ValidID(t.ID) || len(t.Records) == 0 || !foundation.ValidID(t.Source) {
		return t, errors.New("invalid staffing source")
	}
	for i, r := range t.Records {
		end := ""
		if i+1 < len(t.Records) {
			end = t.Records[i+1].Date
		}
		if r.End != end {
			return t, errors.New("staffing period index differs")
		}
		if !slices.Contains([]string{"CREATE", "CHANGE", "DEACTIVATE", "REACTIVATE"}, r.Type) || i == 0 && r.Type != "CREATE" {
			return t, errors.New("invalid staffing operation")
		}
		if !foundation.ValidID(r.ID) || !foundation.ValidID(r.Revision) || !foundation.ValidDate(r.Date) || i > 0 && r.Date <= t.Records[i-1].Date {
			return t, errors.New("invalid staffing decision")
		}
		if k == d.Position {
			fields := 0
			for _, field := range d.PositionFields() {
				id := r.Data[field]
				if id == "" && field == "gradeId" {
					continue
				}
				ref := r.PositionRefs[field]
				if !foundation.ValidID(id) || !foundation.ValidID(ref.SourceVersion) || !foundation.ValidID(ref.RecordID) || !foundation.ValidID(ref.Revision) || !foundation.ValidDate(ref.BusinessAt) || ref.BusinessAt > r.Date || ref.Name == "" {
					return t, errors.New("invalid position reference evidence")
				}
				fields++
			}
			if fields != len(r.PositionRefs) {
				return t, errors.New("unexpected position reference evidence")
			}
		}
	}
	return t, nil
}
func (u *unit) Load(ctx context.Context, k d.Kind, id, source string) (d.Timeline, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return d.Timeline{}, e
	}
	q := generated.New(u.tx)
	var b []byte
	if source != "" {
		if k == d.Position {
			b, e = q.LoadPositionSource(ctx, generated.LoadPositionSourceParams{InstanceID: u.store.Instance, ObjectID: id, SourceID: source})
		} else if k == d.JobFunction {
			b, e = q.LoadJobFunctionSource(ctx, generated.LoadJobFunctionSourceParams{InstanceID: u.store.Instance, ObjectID: id, SourceID: source})
		} else if k == d.Family {
			b, e = q.LoadFamilySource(ctx, generated.LoadFamilySourceParams{InstanceID: u.store.Instance, ObjectID: id, SourceID: source})
		} else if k == d.Ladder {
			b, e = q.LoadGradeLadderSource(ctx, generated.LoadGradeLadderSourceParams{InstanceID: u.store.Instance, ObjectID: id, SourceID: source})
		} else if k == d.Grade {
			b, e = q.LoadGradeSource(ctx, generated.LoadGradeSourceParams{InstanceID: u.store.Instance, ObjectID: id, SourceID: source})
		} else {
			b, e = q.LoadJobSource(ctx, generated.LoadJobSourceParams{InstanceID: u.store.Instance, ObjectID: id, SourceID: source})
		}
	} else if k == d.Position {
		b, e = q.LoadPositionHead(ctx, generated.LoadPositionHeadParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, ID: id})
	} else if k == d.JobFunction {
		b, e = q.LoadJobFunctionHead(ctx, generated.LoadJobFunctionHeadParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, ID: id})
	} else if k == d.Family {
		b, e = q.LoadFamilyHead(ctx, generated.LoadFamilyHeadParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, ID: id})
	} else if k == d.Ladder {
		b, e = q.LoadGradeLadderHead(ctx, generated.LoadGradeLadderHeadParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, ID: id})
	} else if k == d.Grade {
		b, e = q.LoadGradeHead(ctx, generated.LoadGradeHeadParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, ID: id})
	} else {
		b, e = q.LoadJobHead(ctx, generated.LoadJobHeadParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, ID: id})
	}
	if errors.Is(e, pgx.ErrNoRows) {
		return d.Timeline{}, nil
	}
	if e != nil {
		return d.Timeline{}, e
	}
	t, e := decode(b, k)
	if e == nil && t.Enterprise != ent.ID {
		return d.Timeline{}, nil
	}
	if e == nil {
		e = u.hydrateMembers(ctx, &t)
	}
	return t, e
}

// ReferenceClassifications reads every current-head interval needed for Go coverage
// validation, without decoding descriptions, revisions or operation summaries.
// The immutable full source remains the authority for adopted historical views.
func (u *unit) ReferenceClassifications(ctx context.Context, kind d.Kind) ([]d.Timeline, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return nil, e
	}
	rows, e := generated.New(u.tx).ListClassificationReferenceFacts(ctx, generated.ListClassificationReferenceFactsParams{Kind: string(kind), InstanceID: u.store.Instance, EnterpriseID: ent.ID})
	if e != nil {
		return nil, e
	}
	out := []d.Timeline{}
	for _, row := range rows {
		if len(out) == 0 || out[len(out)-1].ID != row.ObjectID {
			out = append(out, d.Timeline{Kind: kind, Enterprise: ent.ID, ID: row.ObjectID, Code: row.Code, Source: row.SourceID})
		}
		t := &out[len(out)-1]
		t.Records = append(t.Records, d.Record{ID: row.RecordID, Date: row.StartDate, End: row.EndDate, Type: row.OperationType, Data: map[string]string{"name": row.Name}})
	}
	return out, nil
}

func (u *unit) ReferenceGrades(ctx context.Context) ([]d.Timeline, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return nil, e
	}
	rows, e := generated.New(u.tx).ListGradeReferenceFacts(ctx, generated.ListGradeReferenceFactsParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID})
	if e != nil {
		return nil, e
	}
	// One record slab avoids thousands of small, pointer-rich allocations.
	// Every identity receives a bounded slice; no append can overwrite a peer.
	out := make([]d.Timeline, 0, len(rows))
	records := make([]d.Record, len(rows))
	start := 0
	for i, row := range rows {
		if len(out) == 0 || out[len(out)-1].ID != row.GradeID {
			start = i
			out = append(out, d.Timeline{Kind: d.Grade, Enterprise: ent.ID, ID: row.GradeID, Code: row.Code})
		}
		records[i] = d.Record{Date: row.StartDate, End: row.EndDate, Type: row.OperationType, Data: map[string]string{"name": row.Name}}
		out[len(out)-1].Records = records[start : i+1 : i+1]
	}
	return out, nil
}

func (u *unit) Heads(ctx context.Context, k d.Kind, ids []string) ([]d.Timeline, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return nil, e
	}
	if k == d.Grade {
		return u.gradeHeads(ctx, ent.ID, ids)
	}
	return u.canonicalHeads(ctx, k, ent.ID, ids)
}

func (u *unit) canonicalHeads(ctx context.Context, k d.Kind, enterprise string, ids []string) ([]d.Timeline, error) {
	rows, e := u.tx.Query(ctx, "SELECT head FROM "+table(k)+" WHERE instance_id=$1 AND enterprise_id=$2 AND id=ANY($3::uuid[])", u.store.Instance, enterprise, ids)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []d.Timeline{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		t, e := decode(b, k)
		if e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	rows.Close()
	for i := range out {
		if e = u.hydrateMembers(ctx, &out[i]); e != nil {
			return nil, e
		}
	}
	return out, nil
}

// Current reads one Go-produced dated projection per identity. SQL only locates
// an indexed half-open interval (or the not-yet-effective first decision).
// It does not resolve lifecycle, permissions or relationship qualification.
func (u *unit) Current(ctx context.Context, k d.Kind) ([]d.Timeline, error) {
	return u.current(ctx, k, 0, 0, true, false)
}

// Ordinary unfiltered lists locate page identities before decoding their facts.
// The count and current intervals share the same read snapshot and enterprise.
func (u *unit) CurrentPage(ctx context.Context, k d.Kind, offset, limit int, byCode, descending bool) ([]d.Timeline, int, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return nil, 0, e
	}
	var count int
	if e = u.tx.QueryRow(ctx, "SELECT count(*) FROM "+table(k)+" WHERE instance_id=$1 AND enterprise_id=$2", u.store.Instance, ent.ID).Scan(&count); e != nil {
		return nil, 0, e
	}
	rows, e := u.current(ctx, k, offset, limit, byCode, descending)
	return rows, count, e
}

func (u *unit) current(ctx context.Context, k d.Kind, offset, limit int, byCode, descending bool) ([]d.Timeline, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return nil, e
	}
	head := table(k)
	outerOrder := ""
	if limit > 0 {
		order := "id"
		if byCode {
			order = "code"
		}
		if descending {
			order += " DESC"
		}
		if byCode {
			order += ",id"
		}
		head = fmt.Sprintf("(SELECT * FROM %s WHERE instance_id=$1 AND enterprise_id=$2 ORDER BY %s LIMIT %d OFFSET %d)", head, order, limit, offset)
		// The join may use a hash/merge plan and reorder the selected page.
		// Preserve its public order at the outermost result boundary as well.
		outerOrder = " ORDER BY h." + strings.ReplaceAll(order, ",id", ",h.id")
	}
	rows, e := u.tx.Query(ctx, `SELECT h.id::text,h.code,h.source_id::text,
 r.record_id::text,r.fact->>'revision',r.start_date::text,COALESCE(r.end_date::text,''),r.fact->>'type',r.fact->'data', r.fact->'functionEvidence'
 FROM `+head+` h JOIN `+table(k)+`_read r ON r.instance_id=h.instance_id AND r.object_id=h.id AND r.source_id=h.source_id
 WHERE h.instance_id=$1 AND h.enterprise_id=$2 AND ((r.start_date<=$3::date AND (r.end_date IS NULL OR r.end_date>$3::date)) OR (h.first_date>$3::date AND r.start_date=h.first_date))`+outerOrder, u.store.Instance, ent.ID, u.now.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02"))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []d.Timeline{}
	for rows.Next() {
		v := d.Timeline{Kind: k, Enterprise: ent.ID}
		var b []byte
		var r d.Record
		var evidence []byte
		if e = rows.Scan(&v.ID, &v.Code, &v.Source, &r.ID, &r.Revision, &r.Date, &r.End, &r.Type, &b, &evidence); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(b, &r.Data); e != nil {
			return nil, e
		}
		if len(evidence) > 0 {
			if e = json.Unmarshal(evidence, &r.Function); e != nil {
				return nil, e
			}
		}
		v.Records = []d.Record{r}
		out = append(out, v)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	if k == d.Job {
		timelines := make([]*d.Timeline, len(out))
		for i := range out {
			timelines[i] = &out[i]
		}
		if e = u.hydrateFunctions(ctx, timelines); e != nil {
			return nil, e
		}
	}
	return out, nil
}

// The savepoint is also the retry boundary: rollback releases *all* family/job
// locks from that attempt. Never add an earlier lock after locking a Job.
func (u *unit) LockObject(ctx context.Context, k d.Kind, id string, explicit, explicitFunctions, explicitGrades []string) (d.Timeline, error) {
	if k == d.JobFunction || k == d.Family || k == d.Grade || k == d.Position {
		if e := u.guardLock(ctx, u.tx, false, objectGuard(k), u.store.Instance, id); e != nil {
			return d.Timeline{}, e
		}
		return u.Load(ctx, k, id, "")
	}
	// Retain every original/proposed dependency while replanning, but release
	// the prior attempt before acquiring the new sorted union.
	ids, gradeIDs := slices.Clone(explicit), slices.Clone(explicitGrades)
	functions := slices.Clone(explicitFunctions)
	for attempt := 0; attempt < 4; attempt++ {
		families, oldFunctions, grades, e := u.dependencyIDs(ctx, k, id)
		if e != nil {
			return d.Timeline{}, e
		}
		functions = append(functions, oldFunctions...)
		slices.Sort(functions)
		functions = slices.Compact(functions)
		ids = append(ids, families...)
		slices.Sort(ids)
		ids = slices.Compact(ids)
		gradeIDs = append(gradeIDs, grades...)
		slices.Sort(gradeIDs)
		gradeIDs = slices.Compact(gradeIDs)
		parent := u.tx
		sp, e := parent.Begin(ctx)
		if e != nil {
			return d.Timeline{}, e
		}
		u.tx = sp
		for _, fid := range ids {
			e = u.guardLock(ctx, sp, true, "staffing.family", u.store.Instance, fid)
			if e != nil {
				break
			}
		}
		for _, fid := range functions {
			if e != nil {
				break
			}
			e = u.guardLock(ctx, sp, true, objectGuard(d.JobFunction), u.store.Instance, fid)
		}
		if e == nil && len(gradeIDs) > 0 {
			keys := make([]int64, len(gradeIDs))
			for i, gid := range gradeIDs {
				keys[i] = lockKey("staffing.grade", u.store.Instance, gid)
			}
			// The subquery is an ordering boundary: acquire shared Grade guards in
			// caller-sorted ID order with one round trip, under the same total budget.
			e = u.withLockBudget(ctx, func(bounded context.Context) error {
				_, err := sp.Exec(bounded, `SELECT pg_advisory_xact_lock_shared(key) FROM (SELECT key FROM unnest($1::bigint[]) WITH ORDINALITY AS locks(key,position) ORDER BY position OFFSET 0) ordered`, keys)
				return err
			})
		}
		if e == nil {
			e = u.guardLock(ctx, sp, false, "staffing."+string(k), u.store.Instance, id)
		}
		var fresh d.Timeline
		if e == nil {
			fresh, e = u.Load(ctx, k, id, "")
		}
		complete := e == nil
		for _, fid := range d.FamilyIDs(fresh) {
			complete = complete && slices.Contains(ids, fid)
		}
		// A committed reclassification between the tentative scan and the
		// Job guard requires a fresh plan, even when all its keys were explicit.
		slices.Sort(oldFunctions)
		complete = complete && slices.Equal(oldFunctions, d.FunctionIDs(fresh))
		for _, gid := range d.GradeIDs(fresh) {
			complete = complete && slices.Contains(gradeIDs, gid)
		}
		u.tx = parent
		if complete {
			if e = sp.Commit(ctx); e != nil {
				return fresh, e
			}
			return fresh, nil
		}
		clean, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		re := sp.Rollback(clean)
		cancel()
		if re != nil {
			return d.Timeline{}, foundation.Fail(503, "M03_RESULT_PENDING", "暂时无法确认提交结果")
		}
		if e != nil {
			return d.Timeline{}, e
		}
	}
	return d.Timeline{}, foundation.Fail(409, "M03_CONFLICT", "引用范围正在变化，请重新查询后重试")
}
func (u *unit) Dependents(ctx context.Context, kind d.Kind, id string) ([]d.Timeline, error) {
	ent, e := u.Enterprise(ctx, false)
	if e != nil {
		return nil, e
	}
	predicate := "EXISTS(SELECT 1 FROM staffing.job_read r WHERE r.instance_id=j.instance_id AND r.object_id=j.id AND r.family_id=$3 AND r.active)"
	if kind == d.JobFunction {
		predicate = `jsonb_path_query_array(j.head,'$.records[*].data.jobFunctionId') @> jsonb_build_array($3::text)`
	}
	rows, e := u.tx.Query(ctx, `SELECT j.head FROM staffing.job j WHERE j.instance_id=$1 AND j.enterprise_id=$2 AND `+predicate+` ORDER BY j.id`, u.store.Instance, ent.ID, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []d.Timeline{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		t, e := decode(b, d.Job)
		if e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (u *unit) Business(ctx context.Context, fn func() error) (err error) {
	parent := u.tx
	sp, e := parent.Begin(ctx)
	if e != nil {
		return e
	}
	u.tx = sp
	defer func() { u.tx = parent }()
	if err = fn(); err != nil {
		clean, c := context.WithTimeout(context.Background(), 2*time.Second)
		defer c()
		if e = sp.Rollback(clean); e != nil {
			_ = parent.Conn().Close(clean)
			return foundation.Fail(503, "M03_RESULT_PENDING", "暂时无法确认提交结果")
		}
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" && strings.HasSuffix(pg.ConstraintName, "_instance_id_enterprise_id_code_key") {
			return &foundation.Fault{Status: 409, Code: "M03_CODE_CONFLICT", Field: "code", Message: "同类资料的编码已存在"}
		}
		return err
	}
	return sp.Commit(ctx)
}
func (u *unit) Save(ctx context.Context, p d.Plan, c s.Command, operator access.OperatorSnapshot) error {
	t := p.After
	name := table(t.Kind)
	b, e := json.Marshal(t)
	if e != nil {
		return e
	}
	op, e := json.Marshal(c)
	if e != nil {
		return e
	}
	actor, e := json.Marshal(operator)
	if e != nil {
		return e
	}
	batch := &pgx.Batch{}
	batch.Queue("INSERT INTO "+name+"(instance_id,enterprise_id,id,code,source_id,source_no,first_date,head) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(instance_id,id) DO UPDATE SET source_id=excluded.source_id,source_no=excluded.source_no,first_date=excluded.first_date,head=excluded.head", u.store.Instance, t.Enterprise, t.ID, t.Code, t.Source, t.Number, t.Records[0].Date, b)
	batch.Queue("INSERT INTO "+name+"_source(instance_id,object_id,source_id,source_no,subject_id,submission_id,digest,snapshot,operation,operator,recorded_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)", u.store.Instance, t.ID, t.Source, t.Number, u.guard.Principal().ID, c.SubmissionID, c.Digest, b, op, actor, u.Now())
	// A revision already contains its complete decided dates and facts. Reuse
	// unchanged consumer projections while moving them to the new head source;
	// only replaced/deleted decisions require index deletion and reinsertion.
	unchanged := []string{}
	if d.HasGrades(t.Kind) {
		for _, r := range t.Records {
			if i := d.Index(p.Before, r.ID); i >= 0 && p.Before.Records[i].Revision == r.Revision {
				unchanged = append(unchanged, r.ID)
			}
		}
	}
	if len(unchanged) > 0 {
		batch.Queue("UPDATE "+name+"_read SET source_id=$4 WHERE instance_id=$1 AND object_id=$2 AND source_id=$3 AND record_id=ANY($5::uuid[])", u.store.Instance, t.ID, p.Before.Source, t.Source, unchanged)
	}
	batch.Queue("DELETE FROM "+name+"_read WHERE instance_id=$1 AND object_id=$2 AND NOT (record_id=ANY($3::uuid[]))", u.store.Instance, t.ID, unchanged)
	for _, r := range t.Records {
		if slices.Contains(unchanged, r.ID) {
			continue
		}
		rb, e := json.Marshal(r)
		if e != nil {
			return e
		}
		batch.Queue("INSERT INTO "+name+"_revision VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING", u.store.Instance, t.ID, r.ID, r.Revision, t.Source, rb)
		var end, family any
		if r.End != "" {
			end = r.End
		}
		if r.Data["jobFamilyId"] != "" {
			family = r.Data["jobFamilyId"]
		}
		queueRead(batch, t.Kind, u.store.Instance, t, r, end, family, rb)
	}
	memberCopies := u.queueMemberCopies(batch, t, p.Before)
	event, _ := json.Marshal(map[string]any{"type": "staffing." + string(t.Kind) + ".changed", "objectId": t.ID, "sourceVersion": t.Source, "sourceNumber": t.Number, "action": c.Input.Action, "submissionId": c.SubmissionID})
	batch.Queue("INSERT INTO "+name+"_outbox VALUES($1,$2,$3,$4)", u.store.Instance, t.ID, t.Source, event)
	if t.Kind == d.Position && p.Before.ID == "" {
		batch.Queue("INSERT INTO staffing.quota_control VALUES($1,$2,$3,$4)", u.store.Instance, t.ID, t.Source, t.Records[0].Date)
	}
	// Pipeline independent decided writes; consume the projection update count
	// before accepting the batch. Any failure rolls back this business savepoint,
	// including Outbox rows already queued before changed-member COPY transport.
	results := u.tx.SendBatch(ctx, batch)
	defer results.Close()
	for i := 0; i < 2; i++ {
		if _, e = results.Exec(); e != nil {
			return e
		}
	}
	if len(unchanged) > 0 {
		result, err := results.Exec()
		if err != nil {
			return err
		}
		if result.RowsAffected() != int64(len(unchanged)) {
			return errors.New("current staffing projection incomplete")
		}
	}
	if e = results.Close(); e != nil {
		return e
	}
	return u.saveMembers(ctx, t, memberCopies)
}
func (u *unit) Sources(ctx context.Context, k d.Kind, id string) ([]app.Source, error) {
	rows, e := u.tx.Query(ctx, "SELECT snapshot,operation,operator,recorded_at FROM "+table(k)+"_source WHERE instance_id=$1 AND object_id=$2 ORDER BY source_no DESC", u.store.Instance, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []app.Source{}
	for rows.Next() {
		var v app.Source
		var b, c, a []byte
		if e = rows.Scan(&b, &c, &a, &v.RecordedAt); e != nil {
			return nil, e
		}
		if v.Timeline, e = decode(b, k); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(c, &v.Command); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(a, &v.Operator); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	rows.Close()
	for i := range out {
		if e = u.hydrateMembers(ctx, &out[i].Timeline); e != nil {
			return nil, e
		}
	}
	return out, nil
}
func (u *unit) Draft(ctx context.Context, k d.Kind, id string, locking bool) (v app.Draft, found bool, e error) {
	sql := "SELECT revision::text,business_date::text,query,members::text[],excluded::text[],expires_at,cancelled FROM " + table(k) + "_selection WHERE instance_id=$1 AND session_id=$2 AND id=$3"
	if locking {
		sql += " FOR UPDATE"
	}
	var b []byte
	e = u.tx.QueryRow(ctx, sql, u.store.Instance, u.guard.Principal().SessionID, id).Scan(&v.Revision, &v.BusinessAt, &b, &v.Members, &v.ExcludedIDs, &v.Expiry, &v.Cancelled)
	if errors.Is(e, pgx.ErrNoRows) {
		return v, false, nil
	}
	if e != nil {
		return
	}
	v.DraftID = id
	v.State = "active"
	if v.Cancelled {
		v.State = "cancelled"
	}
	e = json.Unmarshal(b, &v.Query)
	v.Count = len(v.Members) - len(v.ExcludedIDs)
	return v, e == nil, e
}
func (u *unit) SaveDraft(ctx context.Context, k d.Kind, v app.Draft) error {
	b, e := json.Marshal(v.Query)
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, "INSERT INTO "+table(k)+"_selection(instance_id,session_id,id,revision,business_date,query,members,excluded,expires_at,cancelled) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(instance_id,session_id,id) DO UPDATE SET revision=excluded.revision,excluded=excluded.excluded,cancelled=excluded.cancelled", u.store.Instance, u.guard.Principal().SessionID, v.DraftID, v.Revision, v.BusinessAt, b, v.Members, v.ExcludedIDs, v.Expiry, v.Cancelled)
	return e
}
func (s *Store) EarliestStaffingDate(ctx context.Context, enterprise string) (string, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return "", errors.New("staffing reverse query requires existing transaction")
	}
	var first *string
	e := tx.QueryRow(ctx, `SELECT min(d)::text FROM (
 SELECT min(lower(period)) AS d FROM staffing.assignment_policy_member m WHERE instance_id=$1 AND owner_id=$2 AND version=(SELECT version FROM staffing.assignment_policy_source WHERE instance_id=$1 AND owner_id=$2 ORDER BY sequence DESC LIMIT 1)
 UNION ALL SELECT min(first_date) FROM staffing.job_family WHERE instance_id=$1 AND enterprise_id=$2
 UNION ALL SELECT min(first_date) FROM staffing.job_function WHERE instance_id=$1 AND enterprise_id=$2
 UNION ALL SELECT min(first_date) FROM staffing.job WHERE instance_id=$1 AND enterprise_id=$2
 UNION ALL SELECT min(first_date) FROM staffing.grade WHERE instance_id=$1 AND enterprise_id=$2
 UNION ALL SELECT min(first_date) FROM staffing.grade_ladder WHERE instance_id=$1 AND enterprise_id=$2
 UNION ALL SELECT min(first_date) FROM staffing.position WHERE instance_id=$1 AND enterprise_id=$2) dates`, s.Instance, enterprise).Scan(&first)
	if first == nil {
		return "", e
	}
	return *first, e
}
func (s *Store) DropSession(ctx context.Context, id string) error {
	for _, k := range []d.Kind{d.Family, d.JobFunction, d.Job, d.Grade, d.Ladder, d.Position} {
		if _, e := s.Pool.Exec(ctx, "DELETE FROM "+table(k)+"_selection WHERE instance_id=$1 AND session_id=$2", s.Instance, id); e != nil {
			return e
		}
	}
	return nil
}
func (s *Store) Sweep(ctx context.Context) error {
	for _, k := range []d.Kind{d.Family, d.JobFunction, d.Job, d.Grade, d.Ladder, d.Position} {
		if _, e := s.Pool.Exec(ctx, "DELETE FROM "+table(k)+"_selection WHERE instance_id=$1 AND expires_at<clock_timestamp()", s.Instance); e != nil {
			return e
		}
	}
	return nil
}

var _ app.Store = (*Store)(nil)
var _ s.EarliestDate = (*Store)(nil)

// All explicit locks, including replans, spend one cumulative acquisition budget.
func (u *unit) withLockBudget(ctx context.Context, fn func(context.Context) error) error {
	if u.lockRemaining <= 0 {
		return context.DeadlineExceeded
	}
	bounded, cancel := context.WithTimeout(ctx, u.lockRemaining)
	defer cancel()
	start := time.Now()
	defer func() { u.lockRemaining -= time.Since(start) }()
	return fn(bounded)
}
func (u *unit) guardLock(ctx context.Context, tx pgx.Tx, shared bool, parts ...string) error {
	return u.withLockBudget(ctx, func(c context.Context) error { return lock(c, tx, shared, parts...) })
}
