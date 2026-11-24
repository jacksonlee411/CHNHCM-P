package postgres

import (
	staffing "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	departmentapp "chnhcm.local/chnhcm/internal/contexts/organization/application/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	departmentdomain "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/history"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres/generated"
	"chnhcm.local/chnhcm/internal/contexts/organization/infra/postgres/transaction"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	EmploymentUses func(context.Context, string, string) ([]staffing.PurposeUse, error)
	Staffing       staffing.OrganizationDependencies
	historyCatalog history.Catalog
	listCache      *departmentListCache
	Pool           *pgxpool.Pool
	Instance       string
	Clock          func(context.Context, pgx.Tx) (time.Time, error)
	RecordedClock  func(context.Context, pgx.Tx) (time.Time, error)
	Guard          func(context.Context, pgx.Tx, access.Credentials) (access.Guard, error)
}
type unit struct {
	staffing                  staffing.OrganizationDependencies
	catalog                   *history.Catalog
	buHistory                 map[string]*buSource
	departmentHistory         map[string]*departmentSource
	listCache                 *departmentListCache
	departmentFacts           *departmentapp.ReadSnapshot
	stableBUCoverage          *application.Timeline
	stableDepartmentRelations *departmentdomain.Timeline
	tx                        pgx.Tx
	q                         *generated.Queries
	instance                  string
	clock                     func(context.Context, pgx.Tx) (time.Time, error)
	recordedClock             func(context.Context, pgx.Tx) (time.Time, error)
	scope                     *transaction.Scope
	guard                     access.Guard
}

func (u *unit) Access() access.Guard { return u.guard }
func (u *unit) Confirm()             { u.scope.Confirm() }

func (s *Store) Within(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	return s.within(ctx, c, false, fn)
}
func (s *Store) WithinWrite(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	return s.within(ctx, c, true, fn)
}
func (s *Store) within(ctx context.Context, c access.Credentials, write bool, fn func(application.Unit) error) error {
	return s.withUnit(ctx, c, write, func(u *unit) error { return fn(u) })
}
func (s *Store) withUnit(ctx context.Context, c access.Credentials, write bool, fn func(*unit) error) error {
	s.historyCatalog.Enter()
	defer s.historyCatalog.Leave()
	return transaction.Run(ctx, s.Pool, write, func(scope *transaction.Scope) error {
		tx := scope.Tx()
		if foundation.HasStageObserver(ctx) {
			foundation.BeginStage(ctx, fmt.Sprintf("backend.%d", tx.Conn().PgConn().PID()))()
		}
		guarded := foundation.BeginStage(ctx, "access.guard")
		g, err := s.Guard(ctx, tx, c)
		guarded()
		if err != nil {
			return err
		}
		scope.BeforeCommit(g.RecordSessionActivity)
		u := &unit{staffing: s.Staffing, catalog: &s.historyCatalog, listCache: s.departmentReadCache(), tx: tx, q: generated.New(tx), instance: s.Instance, guard: g, clock: s.Clock, recordedClock: s.RecordedClock, scope: scope}
		return fn(u)
	})
}
func (u *unit) Business(ctx context.Context, fn func() error) error {
	return u.scope.Business(ctx, func(tx pgx.Tx) error {
		parent, queries := u.tx, u.q
		u.tx, u.q = tx, generated.New(tx)
		defer func() { u.tx, u.q = parent, queries }()
		return fn()
	})
}
func (u *unit) Now(ctx context.Context) (t time.Time, err error) {
	if u.clock != nil {
		return u.clock(ctx, u.tx)
	}
	err = u.tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&t)
	return
}
func (u *unit) Enterprise(ctx context.Context, lock bool) (org.Enterprise, error) {
	var id string
	query := `SELECT enterprise_id::text FROM organization.enterprise WHERE instance_id=$1`
	if lock {
		query += " FOR SHARE"
	}
	if e := u.tx.QueryRow(ctx, query, u.instance).Scan(&id); e != nil {
		return org.Enterprise{}, dependency(e)
	}
	state, e := (&enterpriseUnit{u}).Load(ctx, id, "")
	if e != nil {
		return org.Enterprise{}, e
	}
	if state.Timeline.ID == "" {
		return org.Enterprise{}, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "企业资料尚未初始化")
	}
	return enterpriseProjection(state.Timeline)
}

func dependency(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "企业尚未初始化")
	}
	return err
}
func (u *unit) LockSubmission(ctx context.Context, id string) error {
	defer foundation.BeginStage(ctx, "guard.submission_wait")()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key, err := (submission.Identity{Kind: submission.BusinessUnit, InstanceID: u.instance, SubjectID: u.guard.Principal().ID, SubmissionID: id}).LockKey()
	if err != nil {
		return err
	}
	_, err = u.tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", key)
	return err
}
func Initialize(ctx context.Context, tx pgx.Tx, instance, code, name, start string) (string, error) {
	if !foundation.ValidID(instance) {
		return "", errors.New("invalid instance or enterprise start")
	}
	// Only this controlled initialization path supplies an omitted initial date.
	// Reopening an existing enterprise must retain its original start and digest.
	if start == "" {
		e := tx.QueryRow(ctx, "SELECT lower(valid_range)::text FROM organization.enterprise WHERE instance_id=$1", instance).Scan(&start)
		if errors.Is(e, pgx.ErrNoRows) {
			start = foundation.DefaultEffectiveDate
		} else if e != nil {
			return "", e
		}
	}
	if !foundation.ValidDate(start) {
		return "", errors.New("invalid instance or enterprise start")
	}
	var now time.Time
	if e := tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); e != nil {
		return "", e
	}
	if start > now.In(foundation.Shanghai).Format("2006-01-02") {
		return "", errors.New("enterprise start is in the future")
	}
	input, e := domain.ValidateEnterpriseInitialization(domain.EnterpriseInitialization{Code: code, Name: name, Start: start})
	if e != nil {
		return "", e
	}
	payload, _ := json.Marshal([]string{instance, input.Code, input.Name, start})
	h := sha256.Sum256(payload)
	var id string
	var old []byte
	e = tx.QueryRow(ctx, "SELECT enterprise_id::text,initialization_digest FROM organization.enterprise WHERE instance_id=$1", instance).Scan(&id, &old)
	if e == nil {
		if fmt.Sprintf("%x", old) != fmt.Sprintf("%x", h[:]) {
			return "", errors.New("enterprise initialization differs; refusing overwrite")
		}
		return id, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return "", e
	}
	id = foundation.NewID()
	_, e = tx.Exec(ctx, `INSERT INTO organization.enterprise VALUES($1,$2,$3,$4,daterange($5::text::date,NULL,'[)'),$6)`, instance, id, input.Code, input.Name, start, h[:])
	return id, e
}

// recordedNow samples the audit timestamp independently of the business day.
func (u *unit) recordedNow(ctx context.Context) (t time.Time, err error) {
	if u.recordedClock != nil {
		return u.recordedClock(ctx, u.tx)
	}
	err = u.tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&t)
	return
}
