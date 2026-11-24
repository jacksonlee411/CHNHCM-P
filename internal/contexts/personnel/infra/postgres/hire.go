package postgres

import (
	"context"
	"errors"
	"slices"

	d "chnhcm.local/chnhcm/internal/contexts/personnel/domain"
	p "chnhcm.local/chnhcm/internal/contexts/personnel/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CheckIdentity(ctx context.Context, c f.Credentials, in p.HireIdentityRequest) (p.HireIdentityResult, error) {
	var result p.HireIdentityResult
	if !f.ValidID(in.PersonID) || !f.ValidDate(in.Arrival) || in.Purpose != "hire" || in.Relationship != "Employee" || len(in.AcceptedDocuments) == 0 {
		return result, d.Invalid("identity", "入职核对条件无效")
	}
	tx, borrowed := dbtx.From(ctx)
	if !borrowed {
		var err error
		tx, err = s.Pool.Begin(ctx)
		if err != nil {
			return result, err
		}
		defer tx.Rollback(ctx)
	}
	// Take authorization and archive locks in the same order as personnel corrections.
	a, _, err := s.Authorize(ctx, tx, c)
	if err != nil {
		return result, err
	}
	if !a.Scope("hire.check", "name", "document").Allows(in.PersonID) {
		return result, d.Missing()
	}
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock_shared(hashtextextended($1,0))", a.Principal.InstanceID+":personnel-writes"); err != nil {
		return result, err
	}
	// Waiting for the archive can outlast either the session or the purpose grant.
	a, guard, err := s.Authorize(ctx, tx, c)
	if err != nil {
		return result, err
	}
	r := request{ctx: ctx, tx: tx, a: a, instance: a.Principal.InstanceID}
	if requiredPolicy(&r) != nil || !r.allowed("hire.check", in.PersonID, "name", "document") {
		return result, d.Missing()
	}
	result, err = r.hireIdentity(in)
	if err != nil {
		return p.HireIdentityResult{}, err
	}
	if !borrowed {
		if err = guard.RecordSessionActivity(ctx); err == nil {
			err = tx.Commit(ctx)
		}
	}
	return result, err
}

func (r *request) hireIdentity(in p.HireIdentityRequest) (p.HireIdentityResult, error) {
	var result p.HireIdentityResult
	// Missing/disposed identity payload or expired ordinary use must not reveal the archive.
	var version string
	err := r.tx.QueryRow(r.ctx, `SELECT p.head FROM personnel.person p JOIN personnel.identity_payload n
	ON (n.instance_id,n.person_id,n.version)=(p.instance_id,p.person_id,p.head)
	WHERE p.instance_id=$1 AND p.person_id=$2 AND p.ordinary_until>clock_timestamp() AND n.name<>''`, r.instance, in.PersonID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, d.Missing()
	}
	if err != nil {
		return result, err
	}
	if in.Expected != nil && (in.Expected.PersonID != in.PersonID || in.Expected.PersonVersion != version) {
		return result, d.Conflict()
	}
	rows, err := r.tx.Query(r.ctx, `SELECT item_id,revision,kind,coalesce(attributes->>'issueDate',''),coalesce(attributes->>'expiryDate',''),long_term
	FROM personnel.document_payload WHERE instance_id=$1 AND person_id=$2 AND version=$3 AND number<>'' ORDER BY is_primary DESC,ordinal,item_id`, r.instance, in.PersonID, version)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	missing := "document"
	for rows.Next() {
		var id, revision, kind, issue, expiry string
		var longTerm bool
		if err = rows.Scan(&id, &revision, &kind, &issue, &expiry, &longTerm); err != nil {
			return result, err
		}
		if !slices.Contains(in.AcceptedDocuments, kind) || issue > in.Arrival {
			continue
		}
		if in.RequireKnownExpiry && !longTerm && expiry == "" {
			missing = "expiryDate"
			continue
		}
		if in.RequireInspection {
			// The first customer rule has no inspection requirement. A registration
			// reference supplies no independent, document-bound inspection conclusion.
			missing = "inspection"
			continue
		}
		source := p.HireIdentitySource{PersonID: in.PersonID, PersonVersion: version, DocumentID: id, DocumentRevision: revision}
		if in.Expected != nil && *in.Expected != source {
			continue
		}
		return p.HireIdentityResult{Source: source}, nil
	}
	if err = rows.Err(); err != nil {
		return result, err
	}
	if in.Expected != nil {
		return result, d.Conflict()
	}
	if !r.allowed("hire.check", in.PersonID, "name", "document", "missing."+missing) {
		return result, d.Missing()
	}
	return p.HireIdentityResult{Missing: []string{missing}}, nil
}
