package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
)

// EmploymentAllowed evaluates one complete scoped grant. A blank organization
// in a capability query only signals that the form may be opened; all commands
// supply the resolved targets and check the full grant again.
func (s *Store) EmploymentAllowed(ctx context.Context, tx pgx.Tx, actor, action, person, employer, department string, candidates bool, fields ...string) (bool, error) {
	var allowed bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM access.employment_grant WHERE instance_id=$1 AND principal_id=$2 AND action=$3
	AND (($7 AND $4='') OR person_id IS NULL OR person_id=NULLIF($4,'')::uuid) AND (($7 AND $5='') OR employer_id IS NULL OR employer_id=NULLIF($5,'')::uuid)
	AND (($7 AND $6='') OR department_id IS NULL OR department_id=NULLIF($6,'')::uuid) AND (expires_at IS NULL OR expires_at>clock_timestamp())
	AND fields @> coalesce($8::text[], '{}'::text[]) AND fields @> ARRAY['arrangement','basis'] AND ($3 NOT IN ('location.change','location.correct') OR 'location'=ANY(fields)) AND ($3 NOT IN ('transfer','assignment.add','employment.parallel') OR fields @> ARRAY['department','job']))`, s.Instance, actor, action, person, employer, department, candidates, fields).Scan(&allowed)
	return allowed, err
}

func InitializeEmployment(ctx context.Context, tx pgx.Tx, instance, actor string) error {
	_, err := tx.Exec(ctx, `WITH first AS (INSERT INTO access.employment_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
	INSERT INTO access.employment_grant(instance_id,principal_id,action,fields)
	SELECT instance_id,principal_id,action,ARRAY['arrangement','basis'] FROM first CROSS JOIN unnest(ARRAY['hire','read','cancel','history']) AS action`, instance, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH granted AS (INSERT INTO access.appointment_issuer_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
    INSERT INTO access.employment_grant(instance_id,principal_id,action,fields)
    SELECT instance_id,principal_id,'appointment.issue',ARRAY['arrangement','basis'] FROM granted`, instance, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH first AS (INSERT INTO access.employment_location_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
        INSERT INTO access.employment_grant(instance_id,principal_id,action,fields)
        SELECT instance_id,principal_id,action,ARRAY['arrangement','basis','location'] FROM first CROSS JOIN unnest(ARRAY['location.change','location.correct','directory']) action`, instance, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH first AS (INSERT INTO access.employment_register_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
        INSERT INTO access.employment_grant(instance_id,principal_id,action,fields)
        SELECT instance_id,principal_id,'register.read',ARRAY['arrangement','basis','department','operator'] FROM first`, instance, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH first AS (INSERT INTO access.employment_history_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
      INSERT INTO access.employment_grant(instance_id,principal_id,action,fields)
      SELECT instance_id,principal_id,action,ARRAY['arrangement','basis','department','job','location'] FROM first
      CROSS JOIN unnest(ARRAY['transfer','record.redate','record.delete']) action`, instance, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH first AS (INSERT INTO access.employment_parallel_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
      INSERT INTO access.employment_grant(instance_id,principal_id,action,fields)
      SELECT instance_id,principal_id,action,ARRAY['arrangement','basis','department','job','employer'] FROM first
      CROSS JOIN unnest(ARRAY['assignment.add','employment.parallel']) action`, instance, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH first AS (INSERT INTO access.employment_probation_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
      INSERT INTO access.employment_grant(instance_id,principal_id,action,fields)
      SELECT instance_id,principal_id,action,ARRAY['arrangement','basis'] FROM first CROSS JOIN unnest(ARRAY['probation.start','probation.confirm']) action`, instance, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH first AS (INSERT INTO access.employment_promotion_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
      INSERT INTO access.employment_grant(instance_id,principal_id,action,fields)
      SELECT instance_id,principal_id,action,ARRAY['arrangement','basis'] FROM first CROSS JOIN unnest(ARRAY['grade.record','promotion','promotion.withdraw']) action`, instance, actor)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH first AS (INSERT INTO access.employment_lifecycle_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
      INSERT INTO access.employment_grant(instance_id,principal_id,action,fields)
      SELECT instance_id,principal_id,action,ARRAY['arrangement','basis'] FROM first CROSS JOIN unnest(ARRAY['manager.change','assignment.end','employment.leave','employment.rehire']) action`, instance, actor)

	if err != nil {
		return err
	}
	var duty bool
	if err = tx.QueryRow(ctx, "SELECT to_regclass('access.duty_fixture') IS NOT NULL").Scan(&duty); err != nil {
		return err
	}
	if duty {
		_, err = tx.Exec(ctx, `WITH first AS (INSERT INTO access.duty_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
        INSERT INTO access.employment_grant(instance_id,principal_id,action,fields) SELECT instance_id,principal_id,right_name,ARRAY['arrangement','basis','department','job'] FROM first CROSS JOIN unnest(ARRAY['duty.verify','duty.correct']) right_name`, instance, actor)
		if err != nil {
			return err
		}
	}
	var succession bool
	if err = tx.QueryRow(ctx, "SELECT to_regclass('access.employment_succession_fixture') IS NOT NULL").Scan(&succession); err != nil {
		return err
	}
	if succession {
		_, err = tx.Exec(ctx, `WITH first AS (INSERT INTO access.employment_succession_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
        INSERT INTO access.employment_grant(instance_id,principal_id,action,fields) SELECT instance_id,principal_id,'position.handover',ARRAY['arrangement','basis'] FROM first`, instance, actor)
		if err != nil {
			return err
		}
	}
	var ready bool
	if err = tx.QueryRow(ctx, "SELECT to_regclass('access.offer_fixture') IS NOT NULL").Scan(&ready); err != nil || !ready {
		return err
	}
	_, err = tx.Exec(ctx, `WITH seeded AS (INSERT INTO access.offer_fixture VALUES($1,$2) ON CONFLICT DO NOTHING RETURNING *)
        INSERT INTO access.offer_grant(instance_id,principal_id,action)
        SELECT instance_id,principal_id,x FROM seeded CROSS JOIN unnest(ARRAY['read','issue','accept','reserve','release']) x`, instance, actor)
	return err
}

func (s *Store) OfferAllowed(ctx context.Context, tx pgx.Tx, actor, action, person, employer, department string, candidate bool) (bool, error) {
	var ok bool
	e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT FROM access.offer_grant WHERE instance_id=$1 AND principal_id=$2 AND action=$3
        AND (($7 AND $4='') OR person_id IS NULL OR person_id=nullif($4,'')::uuid)
        AND (($7 AND $5='') OR employer_id IS NULL OR employer_id=nullif($5,'')::uuid)
        AND (($7 AND $6='') OR department_id IS NULL OR department_id=nullif($6,'')::uuid)
        AND (expires_at IS NULL OR expires_at>clock_timestamp()))`, s.Instance, actor, action, person, employer, department, candidate).Scan(&ok)
	return ok, e
}
