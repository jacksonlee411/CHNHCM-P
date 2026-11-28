package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Store) Personnel(ctx context.Context, tx pgx.Tx, c foundation.Credentials) (access.PersonAccess, access.PersonnelGuard, error) {
	return s.personnel(ctx, tx, c, false)
}

// PersonnelDirectory restricts the grant projection to ordinary identity reads.
// A large per-person scope on unrelated profile or history rights is not needed
// to authorize a directory search or its session-owned reference.
func (s *Store) PersonnelDirectory(ctx context.Context, tx pgx.Tx, c foundation.Credentials) (access.PersonAccess, access.PersonnelGuard, error) {
	return s.personnel(ctx, tx, c, true)
}
func (s *Store) personnel(ctx context.Context, tx pgx.Tx, c foundation.Credentials, directoryOnly bool) (access.PersonAccess, access.PersonnelGuard, error) {
	var a access.PersonAccess
	g, e := s.authenticatedGuard(ctx, tx, c, false)
	if e != nil {
		return a, nil, e
	}
	a.Principal = g.Principal()
	var validUntil *time.Time
	e = tx.QueryRow(ctx, `select version,ordinary_days,evidence_days,allow_permanent_decision,case when isfinite(valid_until) then valid_until else null end,material_days,material_evidence_days,contact_days,contact_evidence_days from access.person_policy_for_use($1)`, s.Instance).Scan(&a.Policy.Version, &a.Policy.OrdinaryDays, &a.Policy.EvidenceDays, &a.Policy.AllowPermanentDecision, &validUntil, &a.Policy.MaterialDays, &a.Policy.MaterialEvidenceDays, &a.Policy.ContactDays, &a.Policy.ContactEvidenceDays)
	if validUntil != nil {
		a.Policy.ValidUntil = *validUntil
	}
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return a, nil, e
	}
	purposes, e := tx.Query(ctx, `select purpose,version,case when isfinite(valid_until) then valid_until end,ordinary_days,evidence_days from access.person_data_purposes_for_use($1)`, s.Instance)
	if e != nil {
		return a, nil, e
	}
	a.Purposes = map[string]access.PersonDataPurpose{}
	for purposes.Next() {
		var key string
		var p access.PersonDataPurpose
		var expiry *time.Time
		if e = purposes.Scan(&key, &p.Version, &expiry, &p.OrdinaryDays, &p.EvidenceDays); e != nil {
			purposes.Close()
			return a, nil, e
		}
		if expiry != nil {
			p.ValidUntil = *expiry
		}
		a.Purposes[key] = p
	}
	if e = purposes.Err(); e != nil {
		return a, nil, e
	}
	// Both policy reads can wait for writers. Use the time at which their terms are available.
	if e = tx.QueryRow(ctx, "select clock_timestamp()").Scan(&a.At); e != nil {
		return a, nil, e
	}
	rows, e := tx.Query(ctx, `select action,purpose,fields,person_ids::text[],denied,expires_at,coalesce(source_version::text,''),coalesce(item_id::text,'') from access.person_grant where instance_id=$1 and principal_id=$2 and (expires_at is null or expires_at>$3) and (not $4 or (action='read' and purpose='directory' and source_version is null and item_id is null))`, s.Instance, a.Principal.ID, a.At, directoryOnly)
	if e != nil {
		return a, nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var v access.PersonGrant
		var expiry *time.Time
		if e = rows.Scan(&v.Action, &v.Purpose, &v.Fields, &v.IDs, &v.Denied, &expiry, &v.Source, &v.ItemID); e != nil {
			return a, nil, e
		}
		if expiry != nil {
			v.Expires = *expiry
		}
		a.Grants = append(a.Grants, v)
	}
	return a, g, rows.Err()
}

// Only an explicit local fixture declaration provisions the synthetic purpose
// policy and grants. A repeated initializer must not reinstate revoked rights.
func InitializePersonnel(ctx context.Context, tx pgx.Tx, instance, subject, profile string) error {
	if profile == "" {
		return nil
	}
	if profile != "registrar" && profile != "reader" {
		return errors.New("unknown personnel fixture")
	}
	tag, e := tx.Exec(ctx, `insert into access.person_fixture(instance_id,principal_id,profile) values($1,$2,$3) on conflict do nothing`, instance, subject, profile)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, e = tx.Exec(ctx, `insert into access.person_policy values($1,'local-retention-1','infinity',365,730,true,365,730,365,730) on conflict do nothing`, instance)
	if e != nil {
		return e
	}
	actions := []string{"read", "history.read"}
	if profile == "registrar" {
		actions = append(actions, "verify", "register", "correct", "renumber", "renumber.decide")
	}
	fields := []string{"name", "number", "evidence", "phone", "email", "document.masked"}
	if profile == "registrar" {
		fields = append(fields, "document")
	}
	for _, a := range actions {
		_, e = tx.Exec(ctx, `insert into access.person_grant(instance_id,principal_id,action,purpose,fields) values($1,$2,$3,$4,$5)`, instance, subject, a, access.PersonPurpose(a), fields)
		if e != nil {
			return e
		}
	}
	return nil
}

// InitializePersonalProfileFixture is an explicit opt-in, separate from a T01 login.
func InitializePersonalProfileFixture(ctx context.Context, tx pgx.Tx, instance, subject string) error {
	var profile string
	if err := tx.QueryRow(ctx, "select profile from access.person_fixture where instance_id=$1 and principal_id=$2", instance, subject).Scan(&profile); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, "insert into access.person_profile_fixture values($1,$2) on conflict do nothing", instance, subject)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		fields := append([]string{}, access.PersonalProfileFields...)
		actions := []string{"profile.read"}
		if profile == "registrar" {
			if _, err = tx.Exec(ctx, `insert into access.person_grant(instance_id,principal_id,action,purpose,fields)
                select $1,$2,a,'address-types','{}' from unnest(array['address-type.read','address-type.maintain']) a
                where exists (select from pg_constraint where conrelid='access.person_grant'::regclass and conname='person_address_type_grant_scope')`, instance, subject); err != nil {
				return err
			}
			fields = append(fields, "documentNumber")
			actions = append(actions, "profile.correct")
		} else {
			fields = append(fields, "documentNumber.masked")
		}
		for _, action := range actions {
			if _, err = tx.Exec(ctx, "insert into access.person_grant(instance_id,principal_id,action,purpose,fields) values($1,$2,$3,$4,$5)", instance, subject, action, access.PersonPurpose(action), fields); err != nil {
				return err
			}
		}
	}
	if err := initializeQualifications(ctx, tx, instance, subject, profile); err != nil {
		return err
	}
	tag, err = tx.Exec(ctx, "insert into access.person_address_family_fixture values($1,$2) on conflict do nothing", instance, subject)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	for _, group := range []struct {
		action string
		fields []string
	}{
		{"profile", []string{"addressType", "addressCountry", "province", "area", "street", "postalCode", "addressPrimary"}},
		{"family", []string{"familyName", "relationship", "familyBirthDate", "employer", "familyJob", "familyPhone", "familyEmail", "emergency"}},
	} {
		fields := group.fields
		if group.action == "family" && profile == "reader" {
			fields = append([]string{}, group.fields[:5]...)
			fields = append(fields, "familyPhone.masked", "familyEmail.masked", "emergency")
		}
		for _, verb := range []string{"read", "correct"} {
			if verb == "correct" && profile != "registrar" {
				continue
			}
			action := group.action + "." + verb
			if _, err = tx.Exec(ctx, "insert into access.person_grant(instance_id,principal_id,action,purpose,fields) values($1,$2,$3,$4,$5)", instance, subject, action, access.PersonPurpose(action), fields); err != nil {
				return err
			}
		}
	}
	return nil
}

func initializeQualifications(ctx context.Context, tx pgx.Tx, instance, subject, profile string) error {
	fields := []string{"educationType", "school", "major", "educationLevel", "degree", "studyMode", "educationPath", "startMonth", "endMonth", "completion", "awardDate", "highestEducation", "highestDegree", "credentialKind", "credentialName", "credentialLevel", "specialty", "awardingBody", "issuedOn", "validFrom", "validUntil", "indefinite"}
	actions := []string{"qualification.read"}
	if profile == "registrar" {
		fields = append(fields, "certificateNumber")
		actions = append(actions, "qualification.correct")
	} else {
		fields = append(fields, "certificateNumber.masked")
	}
	_, err := tx.Exec(ctx, `WITH enabled AS (
 INSERT INTO access.person_qualification_fixture VALUES ($1,$2) ON CONFLICT DO NOTHING RETURNING instance_id,principal_id
 ) INSERT INTO access.person_grant(instance_id,principal_id,action,purpose,fields)
 SELECT instance_id,principal_id,action,$3,$4 FROM enabled CROSS JOIN unnest($5::text[]) AS action`, instance, subject, access.PersonPurpose("qualification.read"), fields, actions)
	return err
}

// InitializeHireIdentity explicitly enables purpose-bound identity use for the local actor.
func InitializeHireIdentity(ctx context.Context, tx pgx.Tx, instance, subject string) error {
	_, err := tx.Exec(ctx, `WITH first AS (
	 INSERT INTO access.hire_identity_fixture VALUES ($1,$2) ON CONFLICT DO NOTHING RETURNING *
	) INSERT INTO access.person_grant(instance_id,principal_id,action,purpose,fields)
	SELECT instance_id,principal_id,'hire.check','hire',ARRAY['name','document','missing.document','missing.expiryDate','missing.inspection'] FROM first`, instance, subject)
	return err
}

// Only an explicitly provisioned synthetic maintainer receives these purposes.
func InitializeSensitiveProfileFixture(ctx context.Context, tx pgx.Tx, instance, subject string) error {
	tag, err := tx.Exec(ctx, `insert into access.person_sensitive_fixture select instance_id,principal_id from access.person_fixture
 where instance_id=$1 and principal_id=$2 and profile='registrar' on conflict do nothing`, instance, subject)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	for _, group := range []struct {
		action string
		fields []string
	}{
		{"bank", []string{"bankName", "branchName", "accountHolder", "accountNumber", "currency", "bankPrimary"}},
		{"health", []string{"healthSummary"}},
	} {
		purpose := access.PersonPurpose(group.action + ".read")
		if _, err = tx.Exec(ctx, `insert into access.person_data_purpose values($1,$2,'local-sensitive-1','infinity',365,730) on conflict do nothing`, instance, purpose); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `insert into access.person_grant(instance_id,principal_id,action,purpose,fields) select $1,$2,action,$3,$4 from unnest($5::text[]) action`, instance, subject, purpose, group.fields, []string{group.action + ".read", group.action + ".correct"}); err != nil {
			return err
		}
	}
	return nil
}
