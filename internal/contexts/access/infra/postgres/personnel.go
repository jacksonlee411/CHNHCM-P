package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Store) Personnel(ctx context.Context, tx pgx.Tx, c foundation.Credentials) (access.PersonAccess, access.Guard, error) {
	var a access.PersonAccess
	g, e := s.Guard(ctx, tx, c)
	if e != nil {
		return a, nil, e
	}
	a.Principal = g.Principal()
	if e = tx.QueryRow(ctx, "select clock_timestamp()").Scan(&a.At); e != nil {
		return a, nil, e
	}
	var validUntil *time.Time
	e = tx.QueryRow(ctx, `select version,ordinary_days,evidence_days,allow_permanent_decision,case when isfinite(valid_until) then valid_until else null end,material_days,material_evidence_days,contact_days,contact_evidence_days from access.person_policy_for_use($1)`, s.Instance).Scan(&a.Policy.Version, &a.Policy.OrdinaryDays, &a.Policy.EvidenceDays, &a.Policy.AllowPermanentDecision, &validUntil, &a.Policy.MaterialDays, &a.Policy.MaterialEvidenceDays, &a.Policy.ContactDays, &a.Policy.ContactEvidenceDays)
	if validUntil != nil {
		a.Policy.ValidUntil = *validUntil
	}
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return a, nil, e
	}
	rows, e := tx.Query(ctx, `select action,purpose,fields,person_ids::text[],denied,expires_at,coalesce(source_version::text,''),coalesce(item_id::text,'') from access.person_grant where instance_id=$1 and principal_id=$2 and (expires_at is null or expires_at>$3)`, s.Instance, a.Principal.ID, a.At)
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
