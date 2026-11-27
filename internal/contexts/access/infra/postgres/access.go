package postgres

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"chnhcm.local/chnhcm/internal/contexts/access/domain"
	"chnhcm.local/chnhcm/internal/contexts/access/infra/postgres/generated"
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool        *pgxpool.Pool
	Instance    string
	AfterLogout func(context.Context, access.Principal)
}
type guard struct {
	policyVersion int64
	tx            pgx.Tx
	p             access.Principal
	grants        []access.Grant
	readActions   map[string][]access.Grant
	windows       []timedGrant
	boundary      time.Time
	checks        []func(*guard) bool
	dependents    []*guard
	changedWindow bool
	now           time.Time
}

type guardContextKey struct{}

func (g *guard) Bind(ctx context.Context) context.Context {
	return context.WithValue(ctx, guardContextKey{}, g)
}

func (g *guard) PolicyVersion() int64        { return g.policyVersion }
func (g *guard) Principal() access.Principal { return g.p }

func (g *guard) RecordSessionActivity(ctx context.Context) error {
	err := generated.New(g.tx).TouchSession(ctx, generated.TouchSessionParams{
		InstanceID: g.p.InstanceID, SessionID: g.p.SessionID, AccessedAt: stamp(g.now),
	})
	if err != nil {
		return err
	}
	return g.ValidateUse()
}

func (g *guard) OperatorSnapshot(ctx context.Context) (access.OperatorSnapshot, error) {
	v := access.OperatorSnapshot{SubjectID: g.p.ID, IdentityEvidence: "captured"}
	// Capture X03 identity under the existing policy guard. Copy these values
	// into the source transaction; future account changes never affect evidence.
	err := g.tx.QueryRow(ctx, `SELECT display_name,login,principal_kind FROM access.principal WHERE instance_id=$1 AND principal_id=$2 AND state='active'`, g.p.InstanceID, g.p.ID).Scan(&v.DisplayName, &v.Login, &v.Kind)
	if err != nil {
		return access.OperatorSnapshot{}, err
	}
	if v.Login == nil || strings.TrimSpace(*v.Login) == "" || (v.Kind != "human" && v.Kind != "service") {
		return access.OperatorSnapshot{}, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "操作者信息暂时无法读取，请重试")
	}
	return v, nil
}

func stamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
func token() (string, []byte) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return base64.RawURLEncoding.EncodeToString(b), b
}
func digest(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }
func denied() error          { return foundation.Fail(401, "AUTH_REQUIRED", "请登录后继续") }

func (s *Store) Guard(ctx context.Context, tx pgx.Tx, c access.Credentials) (access.Guard, error) {
	q := generated.New(tx)
	if c.Token == "" {
		return nil, denied()
	}
	id, err := q.FindSessionSubject(ctx, generated.FindSessionSubjectParams{InstanceID: s.Instance, TokenDigest: digest(c.Token)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, denied()
	}
	if err != nil {
		return nil, err
	}
	version, err := q.LockPolicyShared(ctx, generated.LockPolicySharedParams{InstanceID: s.Instance, PrincipalID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, denied()
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.ReadGrants(ctx, generated.ReadGrantsParams{InstanceID: s.Instance, PrincipalID: id})
	if err != nil {
		return nil, err
	}
	session, err := q.ReadSession(ctx, generated.ReadSessionParams{InstanceID: s.Instance, TokenDigest: digest(c.Token)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, denied()
	}
	if err != nil {
		return nil, err
	}
	var now time.Time
	if err = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return nil, err
	}
	if session.State != "active" || session.RevokedAt.Valid || !now.Before(session.IdleExpiresAt.Time) || !now.Before(session.AbsoluteExpiresAt.Time) {
		return nil, denied()
	}
	csrf := base64.RawURLEncoding.EncodeToString(session.CsrfValue)
	if c.Write && subtle.ConstantTimeCompare([]byte(csrf), []byte(c.CSRF)) != 1 {
		return nil, foundation.Fail(403, "AUTH_REQUEST_VALIDATION_FAILED", "登录状态需要刷新")
	}
	g := &guard{policyVersion: version, tx: tx, p: access.Principal{ID: id, SessionID: session.SessionID, Login: session.Login, CSRF: csrf, InstanceID: s.Instance}, now: now}
	for _, r := range rows {
		if r.Purpose != purposeFor(r.Action) {
			continue
		}
		v := access.Grant{ScopeKind: r.ScopeKind, ScopeID: r.ScopeID, Action: r.Action, ReadFields: r.ReadFields, WriteFields: r.WriteFields}
		g.windows = append(g.windows, timedGrant{v, r.ValidFrom.Time, r.ValidUntil.Time})
	}
	g.selectAt(time.Now())
	// A nested call may run inside a pgx savepoint. Authenticate it separately
	// and retain its time-sensitive decisions until the physical transaction commits.
	if parent, ok := ctx.Value(guardContextKey{}).(*guard); ok && !g.boundary.IsZero() && parent.tx.Conn() == tx.Conn() {
		parent.dependents = append(parent.dependents, g)
	}
	return g, nil
}

func matchesIntent(grant access.Grant, in access.Intent) bool {
	// Actions are exact in both domain matchers. Reject an unrelated grant
	// before validating the same department scope fact again for every action.
	if grant.Action != in.Action {
		return false
	}
	return validIntentScope(in, "") && matchesValidatedIntent(grant, in)
}

// The same scope validation is used by commands and batched reads. A batch may
// reuse the formatted day and validation of an identical adjacent scope fact.
func validIntentScope(in access.Intent, day string) bool {
	if in.DepartmentTarget == nil {
		if in.Kind == "department-entry" {
			return (in.Action == "department.read" || in.Action == "department.create" || in.Action == "department.history.read") &&
				foundation.ValidID(in.EnterpriseID) && in.BUID == "" && in.AffectedScope == nil && len(in.Fields) == 0 && len(in.Requirements) == 0
		}
		return !strings.HasPrefix(in.Action, "department.")
	}
	target := in.DepartmentTarget
	fact := target.ScopeFact
	if day == "" {
		day = fact.EvaluatedAt.In(foundation.Shanghai).Format("2006-01-02")
	}
	if in.BUID != "" || !strings.HasPrefix(in.Action, "department.") || target.EnterpriseID != in.EnterpriseID ||
		!foundation.ValidID(target.EnterpriseID) || !foundation.ValidID(fact.ManagementBUID) || fact.EvaluatedAt.IsZero() ||
		fact.BusinessDate != day {
		return false
	}
	switch fact.Mode {
	case "create_candidate":
		if fact.CurrentSourceVersion != "" || fact.RecordID != "" || fact.Revision != "" {
			return false
		}
	case "current_record", "future_first":
		if !foundation.ValidID(target.DepartmentID) || !foundation.ValidID(fact.CurrentSourceVersion) || !foundation.ValidID(fact.RecordID) || !foundation.ValidID(fact.Revision) {
			return false
		}
	default:
		return false
	}
	if in.AffectedScope != nil {
		affected := in.AffectedScope
		if in.Action != "department.reassign" || !foundation.ValidID(affected.ManagementBUID) {
			return false
		}
	}
	return true
}

func matchesValidatedIntent(grant access.Grant, in access.Intent) bool {
	if grant.Action != in.Action {
		return false
	}
	if in.Kind == "department-entry" && in.DepartmentTarget == nil {
		return grant.ScopeKind == "enterprise" && grant.ScopeID == in.EnterpriseID || grant.ScopeKind == "department" || grant.ScopeKind == "bu"
	}
	if strings.HasPrefix(in.Action, "staffingquota.") && grant.ScopeKind == "position" {
		return foundation.ValidID(in.EnterpriseID) && ((in.Kind == "quota-list" && in.PositionID == "") || (in.PositionID != "" && grant.ScopeID == in.PositionID))
	}
	if in.DepartmentTarget == nil {
		return domain.Matches(grant.ScopeKind, grant.ScopeID, grant.Action, in.EnterpriseID, in.BUID, in.Action)
	}
	if in.AffectedScope != nil {
		return domain.Matches(grant.ScopeKind, grant.ScopeID, grant.Action, in.EnterpriseID, in.AffectedScope.ManagementBUID, in.Action)
	}
	target := in.DepartmentTarget
	fact := target.ScopeFact
	return domain.MatchesDepartment(grant.ScopeKind, grant.ScopeID, grant.Action, in.EnterpriseID, target.DepartmentID, fact.ManagementBUID, in.Action)
}

// Catalog scopes authorize only positions visible through the position directory.
func (g *guard) quotaCatalogGrant(grant access.Grant, in access.Intent) (access.Grant, bool) {
	if grant.Action != in.Action || !strings.HasPrefix(in.Action, "staffingquota.") ||
		(in.Kind != "quota-list" && in.Kind != "quota-target") {
		return grant, false
	}
	if grant.ScopeKind == "position_catalog" && grant.ScopeID != in.EnterpriseID ||
		grant.ScopeKind == "defined_position" && in.Kind == "quota-target" && grant.ScopeID != in.PositionID {
		return grant, false
	}
	op := "position.read"
	if in.Action == "staffingquota.history.read" {
		op = "position.history.read"
	}
	def := g.evaluate(access.Intent{Kind: "position", EnterpriseID: in.EnterpriseID, Action: op})
	if !def.Allowed || !domain.ContainsAll(def.ReadFields, []string{"name", "effectiveDate", "status"}) {
		return grant, false
	}
	if in.Action != "staffingquota.read" && in.Action != "staffingquota.history.read" &&
		!domain.ContainsAll(grant.WriteFields, []string{"quota", "effectiveDate", "reason"}) {
		return grant, false
	}
	fields := append([]string{}, grant.ReadFields...)
	for _, f := range []string{"name", "code", "departmentId", "status"} {
		if domain.Contains(def.ReadFields, f) {
			fields = append(fields, f)
		}
	}
	grant.ReadFields = fields
	return grant, true
}

// evaluate checks each complete grant independently. Fields from partial grants
// cannot be combined to authorize a command.
func (g *guard) evaluate(in access.Intent) access.Decision {
	d := access.Decision{ReadFields: []string{}, WriteFields: []string{}}
	read, write := map[string]bool{}, map[string]bool{}
	scopeValid := validIntentScope(in, "")
	for _, grant := range g.grants {
		match := scopeValid && g.matchesGrant(grant, in)
		if scopeValid && (grant.ScopeKind == "position_catalog" || grant.ScopeKind == "defined_position") {
			grant, match = g.quotaCatalogGrant(grant, in)
		}
		if (in.Kind == "list" || in.Kind == "context") && in.Action == "bu.read" && grant.Action == in.Action && grant.ScopeKind == "bu" {
			match = true
		}
		if match && domain.ContainsAll(grant.WriteFields, in.Fields) {
			d.Allowed = true
			for _, f := range grant.ReadFields {
				read[f] = true
			}
			for _, f := range grant.WriteFields {
				write[f] = true
			}
		}
	}
	if in.Kind == "session" || in.Kind == "resolve-own" {
		d.Allowed = true
	}
	for _, required := range in.Requirements {
		complete := false
		requiredScopeValid := validIntentScope(required, "")
		for _, grant := range g.grants {
			if requiredScopeValid && g.matchesGrant(grant, required) && domain.ContainsAll(grant.WriteFields, required.Fields) {
				complete = true
				break
			}
		}
		if !complete && d.DeniedItemID == "" {
			d.DeniedItemID = required.ItemID
		}
		d.Allowed = d.Allowed && complete
	}
	for f := range read {
		d.ReadFields = append(d.ReadFields, f)
	}
	for f := range write {
		d.WriteFields = append(d.WriteFields, f)
	}
	sort.Strings(d.ReadFields)
	sort.Strings(d.WriteFields)
	return d
}

func (g *guard) Decide(ctx context.Context, in access.Intent) (access.Decision, error) {
	if err := ctx.Err(); err != nil {
		return access.Decision{}, err
	}
	g.refresh()
	d := g.evaluate(in)
	if in.PermissionOnly {
		d.ReadFields, d.WriteFields = nil, nil
	}
	if d.Allowed && !g.boundary.IsZero() {
		g.checks = append(g.checks, func(current *guard) bool { return coversDecision(current.evaluate(in), d) })
	}
	return d, nil
}

// Change is an upgrader-only operation serialized against current business requests.
func Change(ctx context.Context, pool *pgxpool.Pool, instance, login, operation, action, grantID string) error {
	if operation != "disable" && operation != "revoke" {
		return errors.New("unsupported security operation")
	}
	if operation == "revoke" && ((action == "") == (grantID == "") || grantID != "" && !foundation.ValidID(grantID)) {
		return errors.New("revocation requires a grant ID or a single legacy action")
	}
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	q := generated.New(tx)
	p, e := q.FindLogin(ctx, generated.FindLoginParams{InstanceID: instance, NormalizedLogin: strings.ToLower(strings.TrimSpace(login))})
	if e != nil {
		return errors.New("account not found")
	}
	if _, e = q.LockPolicyExclusive(ctx, generated.LockPolicyExclusiveParams{InstanceID: instance, PrincipalID: p.PrincipalID}); e != nil {
		return e
	}
	if operation == "disable" {
		if _, e = tx.Exec(ctx, "UPDATE access.principal SET state='disabled' WHERE instance_id=$1 AND principal_id=$2", instance, p.PrincipalID); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "UPDATE access.session SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND principal_id=$2 AND revoked_at IS NULL", instance, p.PrincipalID); e != nil {
			return e
		}
	} else {
		// Old single-action CLI requests can identify exactly one grant. Ambiguous
		// requests require the grant ID and never revoke multiple sources.
		rows, err := tx.Query(ctx, `SELECT a.grant_id::text,a.revision,cardinality(v.actions)=1 FROM access.grant_assignment a
 JOIN access.grant_revision r USING(instance_id,grant_id,revision)
 JOIN access.policy_version v ON (v.instance_id,v.kind,v.version_id)=(a.instance_id,'role',r.role_id)
 WHERE a.instance_id=$1 AND a.principal_id=$2 AND r.operation<>'revoke'
 AND (a.grant_id::text=$3 OR $4=ANY(v.actions))`, instance, p.PrincipalID, grantID, action)
		if err != nil {
			return err
		}
		type match struct {
			id           string
			revision     int64
			singleAction bool
		}
		candidates, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (match, error) {
			var v match
			err := row.Scan(&v.id, &v.revision, &v.singleAction)
			return v, err
		})
		if err != nil {
			return err
		}
		if len(candidates) != 1 || grantID == "" && !candidates[0].singleAction {
			return errors.New("revocation requires one existing grant ID")
		}
		_, err = changeAssignment(ctx, tx, instance, access.AssignmentChange{Operation: "revoke", ID: candidates[0].id, Subject: p.PrincipalID, RequestID: foundation.NewID(), ExpectedRevision: candidates[0].revision})
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if _, e = tx.Exec(ctx, "UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2", instance, p.PrincipalID); e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO access.security_event(instance_id,event_id,event_kind,occurred_at,action,decision,reason_code,target_subject_id,request_id) VALUES($1,$2,'security_change',clock_timestamp(),$3,'allow','CONTROLLED_UPGRADER',$4,$5)`, instance, foundation.NewID(), operation+":"+action, p.PrincipalID, foundation.NewID())
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Store) Login(ctx context.Context, in access.LoginInput, requestID string) (access.LoginResult, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return access.LoginResult{}, err
	}
	defer tx.Rollback(ctx)
	q := generated.New(tx)
	login := strings.ToLower(strings.TrimSpace(in.Login))
	c, err := q.FindLogin(ctx, generated.FindLoginParams{InstanceID: s.Instance, NormalizedLogin: login})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return access.LoginResult{}, err
	}
	var previousSubject string
	if in.PreviousToken != "" {
		previousSubject, err = q.FindSessionSubject(ctx, generated.FindSessionSubjectParams{InstanceID: s.Instance, TokenDigest: digest(in.PreviousToken)})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return access.LoginResult{}, err
		}
	}
	ids := []string{}
	if c.PrincipalID != "" {
		ids = append(ids, c.PrincipalID)
	}
	if previousSubject != "" && previousSubject != c.PrincipalID {
		ids = append(ids, previousSubject)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, err = q.LockPolicyExclusive(ctx, generated.LockPolicyExclusiveParams{InstanceID: s.Instance, PrincipalID: id}); err != nil {
			return access.LoginResult{}, err
		}
	}
	if c.PrincipalID != "" {
		c, err = q.FindLogin(ctx, generated.FindLoginParams{InstanceID: s.Instance, NormalizedLogin: login})
		if err != nil {
			return access.LoginResult{}, err
		}
	}
	iterations, salt, want := 600000, make([]byte, 16), make([]byte, 32)
	if c.PrincipalID != "" {
		iterations = int(c.Iterations)
		salt = c.Salt
		want = c.PasswordHash
	}
	got, err := pbkdf2.Key(sha256.New, in.Password, salt, iterations, 32)
	if err != nil {
		return access.LoginResult{}, err
	}
	valid := subtle.ConstantTimeCompare(got, want) == 1 && c.PrincipalID != "" && c.State == "active"
	if !valid {
		_, e := tx.Exec(ctx, `INSERT INTO access.security_event(instance_id,event_id,event_kind,occurred_at,action,decision,reason_code,actor_subject_id,request_id) VALUES($1,$2,'authentication',clock_timestamp(),'login','deny','INVALID_CREDENTIALS',$3,$4)`, s.Instance, foundation.NewID(), nullable(c.PrincipalID), requestID)
		if e != nil {
			return access.LoginResult{}, e
		}
		if e = tx.Commit(ctx); e != nil {
			return access.LoginResult{}, e
		}
		return access.LoginResult{}, foundation.Fail(401, "AUTH_REQUIRED", "登录名或密码不正确")
	}
	if previousSubject != "" {
		if _, err = tx.Exec(ctx, "UPDATE access.session SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND token_digest=$2 AND revoked_at IS NULL", s.Instance, digest(in.PreviousToken)); err != nil {
			return access.LoginResult{}, err
		}
		if _, err = tx.Exec(ctx, "UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2", s.Instance, previousSubject); err != nil {
			return access.LoginResult{}, err
		}
	}
	value, _ := token()
	csrf, csrfBytes := token()
	sessionID := foundation.NewID()
	_, err = tx.Exec(ctx, `INSERT INTO access.session(instance_id,session_id,principal_id,token_digest,csrf_value,created_at,last_seen_at,idle_expires_at,absolute_expires_at) SELECT $1,$2,$3,$4,$5,t,t,t+interval '30 minutes',t+interval '8 hours' FROM (SELECT clock_timestamp() t) x`, s.Instance, sessionID, c.PrincipalID, digest(value), csrfBytes)
	if err != nil {
		return access.LoginResult{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO access.security_event(instance_id,event_id,event_kind,occurred_at,action,decision,reason_code,actor_subject_id,request_id) VALUES($1,$2,'authentication',clock_timestamp(),'login','allow','VALID_CREDENTIALS',$3,$4)`, s.Instance, foundation.NewID(), c.PrincipalID, requestID)
	if err != nil {
		return access.LoginResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return access.LoginResult{}, err
	}
	return access.LoginResult{Token: value, Principal: access.Principal{ID: c.PrincipalID, Login: c.Login, SessionID: sessionID, CSRF: csrf, InstanceID: s.Instance}}, nil
}
func (s *Store) Session(ctx context.Context, c access.Credentials) (access.Principal, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return access.Principal{}, e
	}
	defer tx.Rollback(ctx)
	g, e := s.Guard(ctx, tx, c)
	if e != nil {
		return access.Principal{}, e
	}
	if _, e = g.Decide(ctx, access.Intent{Action: "session.read", Kind: "session", Fields: []string{}}); e != nil {
		return access.Principal{}, e
	}
	if e = g.RecordSessionActivity(ctx); e != nil {
		return access.Principal{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return access.Principal{}, e
	}
	return g.Principal(), nil
}
func (s *Store) Logout(ctx context.Context, c access.Credentials) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	q := generated.New(tx)
	id, e := q.FindSessionSubject(ctx, generated.FindSessionSubjectParams{InstanceID: s.Instance, TokenDigest: digest(c.Token)})
	if errors.Is(e, pgx.ErrNoRows) {
		return denied()
	}
	if e != nil {
		return e
	}
	if _, e = q.LockPolicyExclusive(ctx, generated.LockPolicyExclusiveParams{InstanceID: s.Instance, PrincipalID: id}); e != nil {
		return e
	}
	g, e := s.Guard(ctx, tx, c)
	if e != nil {
		return e
	}
	if _, e = g.Decide(ctx, access.Intent{Action: "session.logout", Kind: "session", Fields: []string{}}); e != nil {
		return e
	}
	if e = q.EndSession(ctx, generated.EndSessionParams{InstanceID: s.Instance, SessionID: g.Principal().SessionID}); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2", s.Instance, id); e != nil {
		return e
	}
	if e = g.RecordSessionActivity(ctx); e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	if s.AfterLogout != nil {
		s.AfterLogout(ctx, g.Principal())
	}
	return nil
}

// ChangeLocalPassword is called by the local upgrader with the previous credential.
// Repeating a completed change is safe if the caller has not yet saved local state.
func ChangeLocalPassword(ctx context.Context, pool *pgxpool.Pool, instance, login, previousPassword, password string) error {
	login, valid := domain.NormalizeLogin(login)
	if !valid || len(password) < 12 || len(password) > 1024 {
		return errors.New("invalid local account or password length")
	}
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	q := generated.New(tx)
	params := generated.FindLoginParams{InstanceID: instance, NormalizedLogin: strings.ToLower(strings.TrimSpace(login))}
	account, e := q.FindLogin(ctx, params)
	if e != nil {
		return e
	}
	if _, e = q.LockPolicyExclusive(ctx, generated.LockPolicyExclusiveParams{InstanceID: instance, PrincipalID: account.PrincipalID}); e != nil {
		return e
	}
	account, e = q.FindLogin(ctx, params)
	if e != nil {
		return e
	}
	current, _ := pbkdf2.Key(sha256.New, password, account.Salt, int(account.Iterations), 32)
	if subtle.ConstantTimeCompare(current, account.PasswordHash) == 1 {
		return nil
	}
	previous, _ := pbkdf2.Key(sha256.New, previousPassword, account.Salt, int(account.Iterations), 32)
	if subtle.ConstantTimeCompare(previous, account.PasswordHash) != 1 {
		return errors.New("previous local password does not match")
	}
	_, salt := token()
	salt = salt[:16]
	hash, _ := pbkdf2.Key(sha256.New, password, salt, 600000, 32)
	if _, e = tx.Exec(ctx, `UPDATE access.credential SET salt=$3,password_hash=$4,iterations=600000,updated_at=clock_timestamp() WHERE instance_id=$1 AND principal_id=$2`, instance, account.PrincipalID, salt, hash); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `UPDATE access.session SET revoked_at=clock_timestamp() WHERE instance_id=$1 AND principal_id=$2 AND revoked_at IS NULL`, instance, account.PrincipalID); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `UPDATE access.policy_guard SET policy_no=policy_no+1 WHERE instance_id=$1 AND principal_id=$2`, instance, account.PrincipalID); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO access.security_event(instance_id,event_id,event_kind,occurred_at,action,decision,reason_code,target_subject_id) VALUES($1,$2,'security_change',clock_timestamp(),'local.password.change','allow','LOCAL_DEVELOPMENT',$3)`, instance, foundation.NewID(), account.PrincipalID); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

// Initialize is called only by the controlled upgrader, with credentials from outside the repository.
func Initialize(ctx context.Context, tx pgx.Tx, instance, enterprise, login, password string, readOnly, developmentFixture bool) (subject string, created bool, err error) {
	id := foundation.NewID()
	subject, err = InitializePrincipal(ctx, tx, instance, enterprise, id, login, password, readOnly, developmentFixture)
	return subject, err == nil && subject == id, err
}

// InitializePrincipal lets a disposable historical fixture retain its original
// actor ID before any credential, grant or business fact is inserted.
func InitializePrincipal(ctx context.Context, tx pgx.Tx, instance, enterprise, id, login, password string, readOnly, developmentFixture bool) (string, error) {
	login, validLogin := domain.NormalizeLogin(login)
	if !validLogin || len(password) < 12 || len(password) > 1024 {
		return "", errors.New("initial account requires a 1-100 byte login and a 12-1024 byte password")
	}
	initial, _ := json.Marshal([]any{strings.ToLower(strings.TrimSpace(login)), enterprise, readOnly})
	initialDigest := sha256.Sum256(initial)
	q := generated.New(tx)
	old, e := q.FindLogin(ctx, generated.FindLoginParams{InstanceID: instance, NormalizedLogin: strings.ToLower(strings.TrimSpace(login))})
	if e == nil {
		got, _ := pbkdf2.Key(sha256.New, password, old.Salt, int(old.Iterations), 32)
		if subtle.ConstantTimeCompare(got, old.PasswordHash) != 1 {
			return "", errors.New("initial account differs; refusing overwrite")
		}
		var recorded []byte
		if e = tx.QueryRow(ctx, "SELECT initialization_digest FROM access.principal WHERE instance_id=$1 AND principal_id=$2", instance, old.PrincipalID).Scan(&recorded); e != nil {
			return "", e
		}
		if subtle.ConstantTimeCompare(recorded, initialDigest[:]) != 1 {
			return "", errors.New("initial account scope differs; refusing overwrite")
		}
		return old.PrincipalID, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return "", e
	}
	_, e = tx.Exec(ctx, `INSERT INTO access.principal(instance_id,principal_id,login,normalized_login,state,initialization_digest) VALUES($1,$2,$3,$4,'active',$5);`, instance, id, strings.TrimSpace(login), strings.ToLower(strings.TrimSpace(login)), initialDigest[:])
	if e != nil {
		return "", e
	}
	_, salt := token()
	salt = salt[:16]
	hash, _ := pbkdf2.Key(sha256.New, password, salt, 600000, 32)
	_, e = tx.Exec(ctx, `INSERT INTO access.credential(instance_id,principal_id,algorithm,parameter_version,iterations,salt,password_hash) VALUES($1,$2,'pbkdf2-sha256',1,600000,$3,$4)`, instance, id, salt, hash)
	if e != nil {
		return "", e
	}
	_, e = tx.Exec(ctx, "INSERT INTO access.policy_guard VALUES($1,$2,1)", instance, id)
	if e != nil {
		return "", e
	}
	if !developmentFixture {
		return id, nil
	}
	var batch pgx.Batch
	fields := []string{"code", "name", "nameEn", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"bu.read", "bu.history.read"}
	if !readOnly {
		actions = append(actions, "bu.export", "bu.create", "bu.record.add", "bu.record.correct", "bu.deactivate", "bu.reactivate", "bu.first.reclassify", "bu.record.reclassify", "bu.deactivation.reclassify", "bu.reactivation.reclassify", "bu.record.delete", "bu.deactivation.delete", "bu.reactivation.delete", "bu.record.redate", "bu.first.redate", "bu.deactivation.redate", "bu.reactivation.redate")
	}
	for _, a := range actions {
		write := []string{}
		if a != "bu.read" && a != "bu.history.read" && a != "bu.export" && !strings.HasSuffix(a, ".delete") {
			write = fields
		}
		if strings.HasSuffix(a, ".redate") {
			write = []string{"effectiveDate"}
		}
		queueDevelopmentGrant(&batch, instance, id, "enterprise", enterprise, a, fields, write)
	}
	initializeDepartmentGrants(&batch, instance, enterprise, id, readOnly)
	if err := tx.SendBatch(ctx, &batch).Close(); err != nil {
		return "", err
	}
	return id, nil
}

// Development declarations run once when creating an explicit fixture principal.
func initializeDepartmentGrants(batch *pgx.Batch, instance, enterprise, subject string, readOnly bool) {
	fields := []string{"code", "name", "shortName", "managementBuId", "parentDepartmentId", "departmentPath", "status", "effectiveDate", "operationType", "operationReason"}
	writable := []string{"code", "name", "shortName", "managementBuId", "parentDepartmentId", "effectiveDate", "operationType", "operationReason"}
	actions := []string{"department.read", "department.history.read", "department.reference.read", "bu.reference.read"}
	if !readOnly {
		actions = append(actions, "department.create", "department.export", "department.record.add", "department.record.correct", "department.deactivate", "department.reactivate", "department.record.delete", "department.deactivation.delete", "department.reactivation.delete", "department.record.redate", "department.first.redate", "department.deactivation.redate", "department.reactivation.redate", "department.record.reclassify", "department.first.reclassify", "department.deactivation.reclassify", "department.reactivation.reclassify", "department.reassign")
	}
	for _, action := range actions {
		read, write := fields, []string{}
		if strings.HasSuffix(action, "reference.read") {
			read = []string{"code", "name"}
		}
		if !strings.HasSuffix(action, ".read") && action != "department.export" && !strings.HasSuffix(action, ".delete") {
			write = writable
		}
		if strings.HasSuffix(action, ".redate") {
			write = []string{"effectiveDate"}
		}
		if strings.HasSuffix(action, ".reclassify") {
			write = []string{"operationType", "operationReason"}
		}
		if action == "department.reassign" {
			write = []string{"managementBuId", "parentDepartmentId"}
		}
		queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, action, read, write)
	}
	initializeCountryGrants(batch, instance, subject, readOnly)
	initializeEnterpriseGrants(batch, instance, subject, readOnly)
	initializeGeoAreaGrants(batch, instance, subject, readOnly)
	initializeLocationGrants(batch, instance, enterprise, subject, readOnly)
	initializeLegalEntityGrants(batch, instance, enterprise, subject, readOnly)
	initializeStaffingGrants(batch, instance, enterprise, subject, readOnly)
	initializeGradeRateGrants(batch, instance, enterprise, subject, readOnly)
	for _, kind := range []string{"legalpurpose", "assignmentpolicy"} {
		actions := []string{"read", "history.read", "check"}
		if !readOnly {
			actions = append(actions, "maintain")
		}
		for _, action := range actions {
			writes := []string{}
			if action == "maintain" {
				writes = []string{"rule"}
			}
			queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, kind+"."+action, []string{"rule"}, writes)
		}
	}
	if !readOnly {
		queueDevelopmentGrant(batch, instance, subject, "enterprise", enterprise, "replenishment.maintain", []string{"rule"}, []string{"rule"})
	}
}
