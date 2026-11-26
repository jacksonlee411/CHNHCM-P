package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	d "chnhcm.local/chnhcm/internal/contexts/personnel/domain"
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type authorizePersonnel func(context.Context, pgx.Tx, f.Credentials) (access.PersonAccess, access.Guard, error)
type Store struct {
	ProfileCatalog     org.ProfileCatalog
	Pool               *pgxpool.Pool
	Authorize          authorizePersonnel
	AuthorizeDirectory authorizePersonnel
}
type request struct {
	profileCatalog org.ProfileCatalog
	credentials    f.Credentials
	ctx            context.Context
	tx             pgx.Tx
	a              access.PersonAccess
	instance       string
}

func run[T any](ctx context.Context, s *Store, c f.Credentials, fn func(*request) (T, error)) (T, error) {
	return runAuthorized(ctx, s, c, s.Authorize, fn)
}
func runAuthorized[T any](ctx context.Context, s *Store, c f.Credentials, authorize authorizePersonnel, fn func(*request) (T, error)) (T, error) {
	var zero T
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return zero, e
	}
	defer tx.Rollback(ctx)
	a, g, e := authorize(ctx, tx, c)
	if e != nil {
		return zero, e
	}
	r := request{ctx: dbtx.With(ctx, tx), tx: tx, a: a, instance: a.Principal.InstanceID, credentials: c, profileCatalog: s.ProfileCatalog}
	v, e := fn(&r)
	if e != nil {
		return zero, e
	}
	if e = g.RecordSessionActivity(ctx); e != nil {
		return zero, e
	}
	return v, tx.Commit(ctx)
}
func hash(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (r *request) lock(key string) error {
	// A grant or purpose may expire while a command or receipt waits for the lock.
	return r.tx.QueryRow(r.ctx, "select clock_timestamp() from pg_advisory_xact_lock(hashtextextended($1,0))", r.instance+":"+key).Scan(&r.a.At)
}
func (r *request) allowed(action, id string, fields ...string) bool {
	return r.a.Scope(action, fields...).Allows(id)
}
func (r *request) visible(id string) bool {
	return requiredPolicy(r) == nil && r.allowed("read", id, "name", "number")
}
func (r *request) entry() p.Entry {
	scope := r.a.Scope("read", "name", "number")
	v := p.Entry{CanRead: requiredPolicy(r) == nil && scope.Available(), CanRegister: requiredPolicy(r) == nil && r.allowed("register", "", "name"), ContactKinds: []string{}}
	v.CanRegisterDocument = v.CanRegister && r.allowed("register", "", "name", "document")
	v.CanRegisterEvidence = v.CanRegister && r.allowed("register", "", "name", "evidence")
	for _, k := range []string{"phone", "email"} {
		if v.CanRegister && r.allowed("register", "", "name", k) {
			v.ContactKinds = append(v.ContactKinds, k)
		}
	}
	return v
}
func (s *Store) Entry(ctx context.Context, c f.Credentials) (p.Entry, error) {
	return run(ctx, s, c, func(r *request) (p.Entry, error) { return r.entry(), nil })
}
func (r *request) readable(id string) (bool, error) {
	if !r.visible(id) {
		return false, nil
	}
	var ok bool
	e := r.tx.QueryRow(r.ctx, "select exists(select from personnel.person where instance_id=$1 and person_id=$2 and ordinary_until>clock_timestamp())", r.instance, id).Scan(&ok)
	return ok, e
}
func requiredPolicy(r *request) error {
	if r.a.Policy.Version == "" || !r.a.Policy.ValidUntil.IsZero() && !r.a.At.Before(r.a.Policy.ValidUntil) {
		return f.Fail(409, "PERSON_POLICY", "缺少有效的人员用途政策，请联系管理员")
	}
	return nil
}
func noRows(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return d.Missing()
	}
	return e
}

type snapshot struct {
	materialUntil, materialEvidenceUntil time.Time
	profileItems                         []profileItem
	docs                                 []p.DocumentRow
	policy                               string
	p.Detail
	ordinary, evidence time.Time
	items              []item
}
type item struct {
	policy string
	p.Contact
	ordinary, evidence time.Time
}

func (r *request) load(id, version string) (snapshot, error) {
	var v snapshot
	var name, kind, ref *string
	e := r.tx.QueryRow(r.ctx, `select s.person_id,s.number,s.version,s.recorded_at,s.operator_name,s.action,s.decision,n.name,e.kind,e.reference,coalesce(n.basic,'{}'),n.nationality_adoption,b.ordinary_until,b.evidence_until,b.policy_version,b.material_until,b.material_evidence_until
 from personnel.person b join personnel.source s on s.instance_id=b.instance_id and s.person_id=b.person_id and s.version=coalesce(nullif($3,'')::uuid,b.head)
 left join personnel.identity_payload n on (n.instance_id,n.person_id,n.version)=(s.instance_id,s.person_id,s.version) left join personnel.evidence_payload e on (e.instance_id,e.person_id,e.version)=(s.instance_id,s.person_id,s.version)
 where b.instance_id=$1 and b.person_id=$2`, r.instance, id, version).Scan(&v.ID, &v.Number, &v.Version, &v.At, &v.Operator, &v.Action, &v.Decision, &name, &kind, &ref, &v.Basic, &v.Nationality, &v.ordinary, &v.evidence, &v.policy, &v.materialUntil, &v.materialEvidenceUntil)
	if e != nil {
		return v, noRows(e)
	}
	if name != nil {
		v.Name = *name
	}
	if kind != nil {
		v.Evidence = &p.Evidence{Kind: *kind, Reference: *ref}
	}
	rows, e := r.tx.Query(r.ctx, `select item_id,revision,attributes || jsonb_build_object('documentKind',kind,'documentNumber',number),issuing_country_adoption,long_term,is_primary from personnel.document_payload where instance_id=$1 and person_id=$2 and version=$3 order by ordinal`, r.instance, id, v.Version)
	if e != nil {
		return v, e
	}
	v.docs, e = pgx.CollectRows(rows, func(row pgx.CollectableRow) (p.DocumentRow, error) {
		var x p.DocumentRow
		err := row.Scan(&x.ItemID, &x.Revision, &x.Fields, &x.IssuingCountry, &x.LongTerm, &x.Primary)
		return x, err
	})
	if e != nil {
		return v, e
	}
	v.PrimaryDocument = v.primaryValue()
	if version == "" {
		rows, e := r.tx.Query(r.ctx, `select section,item_id,revision,fields,adoptions,ordinary_until,evidence_until,collection_policy from personnel.profile_item_payload where instance_id=$1 and person_id=$2 and version=$3 and ordinary_until>$4 order by section,ordinal`, r.instance, id, v.Version, r.a.At)
		if e != nil {
			return v, e
		}
		v.profileItems, e = pgx.CollectRows(rows, func(row pgx.CollectableRow) (profileItem, error) {
			var x profileItem
			err := row.Scan(&x.Section, &x.ItemID, &x.Revision, &x.Fields, &x.Adoptions, &x.ordinary, &x.evidence, &x.policy)
			return x, err
		})
		if e != nil {
			return v, e
		}
	}
	rows, e = r.tx.Query(r.ctx, `select item_id,revision,kind,value,purpose,is_primary,ordinary_until,evidence_until,collection_policy from personnel.contact_payload where instance_id=$1 and person_id=$2 and version=$3 order by ordinal`, r.instance, id, v.Version)
	if e != nil {
		return v, e
	}
	defer rows.Close()
	for rows.Next() {
		var x item
		if e = rows.Scan(&x.ItemID, &x.Revision, &x.Kind, &x.Value.Value, &x.Purpose, &x.Primary, &x.ordinary, &x.evidence, &x.policy); e != nil {
			return v, e
		}
		v.items = append(v.items, x)
	}
	return v, rows.Err()
}
