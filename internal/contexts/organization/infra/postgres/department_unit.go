package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	baseapp "chnhcm.local/chnhcm/internal/contexts/organization/application"
	app "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	base "chnhcm.local/chnhcm/internal/contexts/organization/domain"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres/generated"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

type departmentUnit struct {
	*unit
	forwardPreview *departmentForwardPreview
	publication    *departmentPublication
}

type departmentPublication struct {
	id, source string
	read       app.ReadSnapshot
	history    app.HistorySource
}
type departmentForwardPreview struct {
	id           string
	bus, parents []string
	read         app.ReadSnapshot
}

func (s *Store) WithinDepartment(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, false, func(u *unit) error { return fn(&departmentUnit{unit: u}) })
}
func (s *Store) WithinDepartmentWrite(ctx context.Context, c access.Credentials, fn func(app.Unit) error) error {
	return s.withUnit(ctx, c, true, func(u *unit) error { return fn(&departmentUnit{unit: u}) })
}

func (u *departmentUnit) Business(ctx context.Context, fn func() error) error {
	previous := u.publication
	err := u.unit.Business(ctx, fn)
	if err != nil {
		u.publication = previous
	}
	return err
}
func (u *departmentUnit) LockSubmission(ctx context.Context, id string) error {
	defer foundation.BeginStage(ctx, "guard.submission_wait")()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key, e := (submission.Identity{Kind: submission.Department, InstanceID: u.instance, SubjectID: u.guard.Principal().ID, SubmissionID: id}).LockKey()
	if e != nil {
		return e
	}
	_, e = u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", key)
	return e
}
func (u *unit) departmentSnapshot(ctx context.Context, id, source string) (out app.ReadSnapshot, err error) {
	return u.departmentSnapshotScope(ctx, id, source, id == "", nil, nil, false, false)
}
func (u *unit) departmentSnapshotScope(ctx context.Context, id, source string, all bool, bus, parents []string, writeDependencies, allBUs bool) (out app.ReadSnapshot, err error) {
	defer foundation.BeginStage(ctx, "department.snapshot")()
	out.Snapshot = d.Snapshot{Departments: []d.Timeline{}, BusinessUnits: []d.BusinessUnit{}}
	out.Evidence = []d.PathSegment{}
	out.Summaries = map[string]org.OperationSummary{}
	clean := func(ids []string) []string {
		out := []string{}
		for _, id := range ids {
			if id != "" {
				out = append(out, id)
			}
		}
		slices.Sort(out)
		return slices.Compact(out)
	}
	var selected *departmentSource
	if source != "" && id != "" {
		selected, err = u.readDepartment(ctx, id, source, true)
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		if err != nil {
			return out, err
		}
		for _, r := range selected.State.Timeline.Records {
			bus = append(bus, r.Data.BU)
		}
		for _, p := range selected.Evidence.Segments {
			for _, n := range p.Nodes {
				if n.DepartmentID != "" {
					parents = append(parents, n.DepartmentID)
				}
			}
		}
	}
	bus, parents = clean(bus), clean(parents)
	// Discover storage identities in bounded indexed frontiers. Business
	// qualification and lock closure remain in the Department domain.
	headsDone := foundation.BeginStage(ctx, "department.snapshot.heads")
	ds, bs, relations, e := u.departmentHeads(ctx, id, bus, parents, all, writeDependencies, allBUs)
	headsDone()
	if e != nil {
		return out, e
	}
	factsDone := foundation.BeginStage(ctx, "department.snapshot.facts")
	deps, e := u.currentFacts(ctx, history.Department, ds)
	factsDone()
	if e != nil {
		return out, e
	}
	busFacts, e := u.currentFacts(ctx, history.BusinessUnit, bs)
	if e != nil {
		return out, e
	}
	for id, v := range deps.Departments {
		if v.Timeline.RelationSource != relations[id] {
			return out, resultUnavailable()
		}
		out.Snapshot.Departments = append(out.Snapshot.Departments, v.Timeline)
	}
	for _, v := range busFacts.BUs {
		out.Snapshot.BusinessUnits = append(out.Snapshot.BusinessUnits, departmentBU(v.Timeline))
	}
	slices.SortFunc(out.Snapshot.Departments, func(a, b d.Timeline) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(out.Snapshot.BusinessUnits, func(a, b d.BusinessUnit) int { return strings.Compare(a.ID, b.ID) })
	out.Snapshot = out.Snapshot.Indexed()
	if !writeDependencies && id != "" {
		if selected == nil {
			if current, ok := deps.Departments[id]; ok {
				t := current.Timeline
				out.Selected = &t
				for _, s := range current.Summaries {
					out.Summaries[s.Revision] = s
				}
			}
		} else {
			t := selected.State.Timeline
			if current := out.Snapshot.Department(id); current != nil {
				t.Edges = current.Edges
				t.RelationSource = current.RelationSource
				t.RelationNo = current.RelationNo
			}
			out.Selected = &t
			out.Evidence = selected.Evidence.Segments
			for _, s := range selected.State.Summaries {
				out.Summaries[s.Revision] = s
			}
		}
	}
	return out, nil
}
func departmentBU(t baseapp.Timeline) d.BusinessUnit {
	out := d.BusinessUnit{ID: t.BUID, Code: t.Code, Source: t.Source, First: t.First, Records: []d.BURecord{}}
	for _, r := range t.Records {
		out.Records = append(out.Records, d.BURecord{ID: r.ID, Revision: r.Revision, Date: r.Date, End: r.End, State: r.State, Name: r.Data.Name})
	}
	return out
}

func (u *departmentUnit) Snapshot(ctx context.Context, id, source string) (app.ReadSnapshot, error) {
	if p := u.publication; p != nil && p.id == id && p.source == source {
		return p.read, ctx.Err()
	}
	return u.departmentSnapshot(ctx, id, source)
}

// The application requests reverse facts only when the complete domain plan
// changes dated relation coverage. Saving independently locks and revalidates.
func (u *departmentUnit) PreviewSnapshot(ctx context.Context, id string, bus, parents []string, reverse bool) (app.ReadSnapshot, error) {
	u.forwardPreview = nil
	read, err := u.departmentSnapshotScope(ctx, id, "", false, bus, parents, reverse, false)
	if err == nil && !reverse {
		u.forwardPreview = &departmentForwardPreview{id: id, bus: slices.Clone(bus), parents: slices.Clone(parents), read: read}
	}
	return read, err
}

// BU candidates need complete BU histories and the maintained target, not unrelated departments.
func (u *departmentUnit) BUReferenceSnapshot(ctx context.Context, id string) (app.ReadSnapshot, error) {
	return u.departmentSnapshotScope(ctx, id, "", false, nil, nil, false, true)
}

func (u *departmentUnit) Lock(ctx context.Context, in app.LockRequest) (app.ReadSnapshot, error) {
	return u.lockDepartmentDependencies(ctx, in, "", false)
}

func (u *departmentUnit) LockUnchangedRelations(ctx context.Context, in app.LockRequest, before d.Timeline, rows []d.Record) (app.ReadSnapshot, error) {
	if in.Creating || in.DepartmentID != before.ID || before.Source == "" || d.Plan(before, rows).Changed {
		return app.ReadSnapshot{}, fmt.Errorf("unchanged department relations require a complete original candidate")
	}
	// Reuse this request's exact preliminary snapshot only to plan the initial
	// locks. It never replaces the complete reread after acquiring those locks.
	var initial *app.ReadSnapshot
	if p := u.forwardPreview; p != nil && p.id == in.DepartmentID && slices.Equal(p.bus, in.BUs) && slices.Equal(p.parents, in.Parents) {
		if target := p.read.Snapshot.Department(in.DepartmentID); target != nil && target.Source == before.Source {
			initial = &p.read
		}
	}
	u.forwardPreview = nil
	read, err := u.lockDepartmentDependenciesFrom(ctx, in, "", true, initial)
	if err == nil {
		u.stableDepartmentRelations = &before
	}
	return read, err
}

func administrativeGuardKey(instance, enterprise, bu string) int64 {
	b, _ := json.Marshal([]string{"organization.department.admin-relations", instance, enterprise, bu, "DEPARTMENT_ADMIN"})
	h := sha256.Sum256(b)
	return int64(binary.BigEndian.Uint64(h[:8]))
}
func departmentCodeKey(instance, enterprise, code string) int64 {
	b, _ := json.Marshal([]string{"organization.department.code", instance, enterprise, strings.ToLower(strings.TrimSpace(code))})
	h := sha256.Sum256(b)
	return int64(binary.BigEndian.Uint64(h[:8]))
}

var retryDepartmentDependencies = errors.New("department dependencies changed")

// Object locks live in a savepoint so a changed dependency set can be released
// and acquired again in the same global order. X03 and submission guards remain.
func (u *unit) lockDepartmentDependencies(ctx context.Context, in app.LockRequest, writeBU string, forwardOnly bool) (app.ReadSnapshot, error) {
	return u.lockDepartmentDependenciesFrom(ctx, in, writeBU, forwardOnly, nil)
}
func (u *unit) lockDepartmentDependenciesFrom(ctx context.Context, in app.LockRequest, writeBU string, forwardOnly bool, initialRead *app.ReadSnapshot) (out app.ReadSnapshot, err error) {
	defer foundation.BeginStage(ctx, "department.guard")()
	u.stableDepartmentRelations = nil
	// Dependency preparation and lock acquisition need different budgets. Keep
	// one monotonic preparation deadline across discovery and replanning, plus
	// a cumulative two-second allowance for the actual lock SQL. Neither budget
	// is renewed after a savepoint retry; caller cancellation always wins.
	ctx, cancel := context.WithTimeout(ctx, foundation.DepartmentPreparationBudget(ctx))
	defer cancel()
	remainingLocks := foundation.DepartmentLockAcquisitionBudget
	acquire := func(fn func(context.Context) error) error {
		if remainingLocks <= 0 {
			return context.DeadlineExceeded
		}
		started := time.Now()
		lockCtx, release := context.WithTimeout(ctx, remainingLocks)
		e := fn(lockCtx)
		remainingLocks -= time.Since(started)
		if e == nil {
			e = lockCtx.Err()
		}
		release()
		if e == nil && remainingLocks <= 0 {
			return context.DeadlineExceeded
		}
		return e
	}
	var ent org.Enterprise
	err = acquire(func(lockCtx context.Context) error {
		var e error
		ent, e = u.Enterprise(lockCtx, true)
		return e
	})
	if err != nil {
		return out, err
	}
	for {
		if err = ctx.Err(); err != nil {
			return out, err
		}
		seed := slices.Clone(in.BUs)
		if writeBU != "" {
			seed = append(seed, writeBU)
		}
		var initial app.ReadSnapshot
		var e error
		if initialRead != nil {
			initial = *initialRead
			initialRead = nil // Replanning always discovers fresh facts.
		} else {
			initial, e = u.departmentSnapshotScope(ctx, in.DepartmentID, "", false, seed, in.Parents, !forwardOnly, false)
			if e != nil {
				return out, e
			}
		}
		deps := d.Discover(initial.Snapshot, in.DepartmentID, seed, in.Parents)
		e = u.Business(ctx, func() error {
			for _, id := range deps.BUs {
				lockDone := foundation.BeginStage(ctx, "guard.bu_wait")
				e := acquire(func(lockCtx context.Context) error {
					var e error
					if id == writeBU {
						_, e = u.q.LockDepartmentBUWrite(lockCtx, generated.LockDepartmentBUWriteParams{InstanceID: u.instance, Column2: []string{id}})
					} else {
						_, e = u.q.LockDepartmentBUShared(lockCtx, generated.LockDepartmentBUSharedParams{InstanceID: u.instance, Column2: []string{id}})
					}
					return e
				})
				lockDone()
				if e != nil {
					return e
				}
			}
			// Fresh identities and existing leaves may share the range guard.
			// A leaf writer exclusively locks its own fact/relation heads; every
			// incoming-edge writer must share-lock those same heads. An edge committed
			// before that lock is detected below and causes a complete replan, which
			// then takes the exclusive range guard for the newly non-leaf target.
			// A proven unchanged relation envelope also shares the range guard:
			// incoming-edge writers still share-lock its exclusively locked head.
			// BU/other non-leaf writers keep the exclusive range guard.
			rangeLock := "SELECT pg_advisory_xact_lock($1)"
			leaf := in.DepartmentID != "" && initial.Snapshot.Department(in.DepartmentID) != nil && !d.HasIncomingReferences(initial.Snapshot, in.DepartmentID)
			if writeBU == "" && (forwardOnly || in.Creating || leaf) {
				rangeLock = "SELECT pg_advisory_xact_lock_shared($1)"
			}
			for _, id := range deps.BUs {
				lockDone := foundation.BeginStage(ctx, "guard.relation_wait")
				e := acquire(func(lockCtx context.Context) error {
					_, e := u.tx.Exec(lockCtx, rangeLock, administrativeGuardKey(u.instance, ent.ID, id))
					return e
				})
				lockDone()
				if e != nil {
					return e
				}
			}
			// Shared dependencies are acquired in sorted runs around the writable
			// target. Each row still locks its fact and relation head; one SQL round
			// trip per run avoids a fanout-sized sequence of network waits.
			shared := []string{}
			flush := func() error {
				if len(shared) == 0 {
					return nil
				}
				defer foundation.BeginStage(ctx, "guard.department_wait")()
				e := acquire(func(lockCtx context.Context) error {
					_, e := u.q.LockDepartmentDependenciesShared(lockCtx, generated.LockDepartmentDependenciesSharedParams{InstanceID: u.instance, Column2: shared})
					return e
				})
				shared = nil
				return e
			}
			for _, id := range deps.Departments {
				if id != in.DepartmentID {
					shared = append(shared, id)
					continue
				}
				if e := flush(); e != nil {
					return e
				}
				if e := acquire(func(lockCtx context.Context) error {
					defer foundation.BeginStage(ctx, "guard.department_wait")()
					if _, e := u.q.LockDepartmentHeadWrite(lockCtx, generated.LockDepartmentHeadWriteParams{InstanceID: u.instance, DepartmentID: id}); e != nil && !errors.Is(e, pgx.ErrNoRows) {
						return e
					}
					if _, e := u.q.LockDepartmentRelationWrite(lockCtx, generated.LockDepartmentRelationWriteParams{InstanceID: u.instance, DepartmentID: id}); e != nil && !errors.Is(e, pgx.ErrNoRows) {
						return e
					}
					return nil
				}); e != nil {
					return e
				}
			}
			if e := flush(); e != nil {
				return e
			}
			if in.Code != "" {
				if e := acquire(func(lockCtx context.Context) error {
					_, e := u.tx.Exec(lockCtx, "SELECT pg_advisory_xact_lock($1)", departmentCodeKey(u.instance, ent.ID, in.Code))
					return e
				}); e != nil {
					return e
				}
			}
			current, e := u.departmentSnapshotScope(ctx, in.DepartmentID, "", false, seed, in.Parents, !forwardOnly, false)
			if e != nil {
				return e
			}
			next := d.Discover(current.Snapshot, in.DepartmentID, seed, in.Parents)
			if !reflect.DeepEqual(deps, next) || dependencyHeads(initial.Snapshot, deps) != dependencyHeads(current.Snapshot, next) {
				return retryDepartmentDependencies
			}
			out = current
			return nil
		})
		if errors.Is(e, retryDepartmentDependencies) {
			continue
		}
		if e != nil {
			return out, e
		}
		u.departmentFacts = &out
		return out, nil
	}
}
func dependencyHeads(s d.Snapshot, deps d.Dependencies) string {
	values := []string{}
	for _, id := range deps.BUs {
		if b := s.BU(id); b != nil {
			values = append(values, id, b.Source)
		}
	}
	for _, id := range deps.Departments {
		if t := s.Department(id); t != nil {
			values = append(values, id, t.Source, t.RelationSource)
		}
	}
	b, _ := json.Marshal(values)
	return string(b)
}
func (u *unit) validateBUReferences(ctx context.Context, id string, rows []d.BURecord) error {
	if u.departmentFacts == nil {
		return fmt.Errorf("BU relation validation requires locked facts")
	}
	snapshot := u.departmentFacts.Snapshot
	snapshot.BusinessUnits = slices.Clone(snapshot.BusinessUnits)
	for i := range snapshot.BusinessUnits {
		if snapshot.BusinessUnits[i].ID == id {
			snapshot.BusinessUnits[i].Records = rows
		}
	}
	if e := d.Validate(ctx, snapshot, d.Discover(snapshot, "", []string{id}, nil)); e != nil {
		var invalid *d.QualificationError
		if errors.As(e, &invalid) {
			return foundation.Fail(409, "M02_DEPARTMENT_REFERENCE_CONFLICT", "当前关系不满足保存条件")
		}
		return e
	}
	return nil
}

// Source returns one immutable operation, including deleted revisions. Callers
// must first apply the same current authorization as their public read entry.
func (u *departmentUnit) Source(ctx context.Context, id, source string) (app.HistorySource, error) {
	if p := u.publication; p != nil && p.id == id && p.source == source {
		return p.history, ctx.Err()
	}
	v, err := u.readDepartment(ctx, id, source, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.HistorySource{}, foundation.Fail(404, "M02_SOURCE_NOT_FOUND", "无法查看该部门的历史资料")
	}
	if err != nil {
		return app.HistorySource{}, err
	}
	return v.Operation.History, nil
}

func (u *departmentUnit) History(ctx context.Context, id, through string) (out []app.HistorySource, err error) {
	v, e := u.readDepartment(ctx, id, through, false)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, foundation.Fail(404, "M02_SOURCE_NOT_FOUND", "无法查看该部门的历史资料")
	}
	if e != nil {
		return nil, e
	}
	out = []app.HistorySource{}
	err = u.eachOperation(ctx, history.Department, id, v.State.Timeline.SourceNo, func(c history.Chunk, slot int, b []byte) error {
		op, e := history.DecodeDepartmentOperation(ctx, c.Values, b)
		if e != nil {
			return e
		}
		if op.Source.ID != c.ID(c.Sources[slot]) || op.Source.Subject != c.ID(c.Subjects[slot]) || op.Source.Submission != c.ID(c.Submissions[slot]) || op.History.ID != op.Source.ID {
			return resultUnavailable()
		}
		out = append(out, op.History)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Reverse(out)
	return out, nil
}

func (u *unit) ValidateReferences(ctx context.Context, t baseapp.Timeline, c base.Candidate) error {
	if base.SameBUCoverage(t.Records, c.Records) {
		return ctx.Err()
	}
	if u.departmentFacts == nil {
		lost, e := base.RemovedBUCoverage(t.Records, c.Records)
		if e != nil {
			return e
		}
		if len(lost) == 0 {
			return ctx.Err()
		}
		witness, e := history.CaptureBUCoverageReference(ctx, u.tx, u.instance, t.BUID, t.Source, lost)
		if e != nil || witness == nil {
			return e
		}
		if e = validateDepartmentState(*witness); e != nil {
			return e
		}
		return foundation.Fail(409, "M02_DEPARTMENT_REFERENCE_CONFLICT", "当前关系不满足保存条件")
	}
	rows := []d.BURecord{}
	for _, r := range c.Records {
		rows = append(rows, d.BURecord{ID: r.ID, Revision: r.Revision, Date: r.Date, End: r.End, State: r.State, Name: r.Data.Name})
	}
	return u.validateBUReferences(ctx, t.BUID, rows)
}
