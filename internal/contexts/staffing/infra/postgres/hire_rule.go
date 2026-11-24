package postgres

import (
	"context"
	"errors"
	"reflect"

	"chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"github.com/jackc/pgx/v5"
)

// InitializeFirstHireRule runs only for an explicitly configured customer.
// The controlled initializer and the customer configuration commit together.
func InitializeFirstHireRule(ctx context.Context, tx pgx.Tx, customer, subject, version string, period f.Period) error {
	rule := hireprep.DirectEmployeeSample()
	if version != rule.Version {
		return errors.New("unknown first hire rule")
	}
	if period.From == "" {
		// Reopening an explicitly selected version retains its original dates.
		var existing f.Period
		err := tx.QueryRow(ctx, `SELECT to_char(lower(period),'YYYY-MM-DD'),coalesce(to_char(upper(period),'YYYY-MM-DD'),'')
			FROM staffing.hire_rule_source WHERE instance_id=$1 AND version=$2`, customer, version).Scan(&existing.From, &existing.Until)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			period.From = f.DefaultEffectiveDate
		case err != nil:
			return err
		default:
			period.From = existing.From
			if period.Until == "" {
				period.Until = existing.Until
			}
		}
	}
	if !period.Valid() {
		return errors.New("invalid first hire rule period")
	}
	tag, err := tx.Exec(ctx, `INSERT INTO staffing.hire_rule_source
		(instance_id,version,period,relationship,action,initial_state,primary_purpose,reasons,accepted_documents,
		 require_known_expiry,require_inspection,subject_id)
		VALUES($1,$2,daterange($3::date,NULLIF($4,'')::date,'[)'),$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (instance_id,version) DO NOTHING`, customer, version, period.From, period.Until,
		rule.Relationship, rule.Action, rule.InitialState, rule.PrimaryPurpose, rule.Reasons, rule.AcceptedDocuments,
		rule.RequireKnownExpiry, rule.RequireInspection, subject)
	if err != nil || tag.RowsAffected() == 1 {
		return err
	}
	stored, original, err := readFirstHireRule(ctx, tx, customer, period.From)
	if err != nil {
		return err
	}
	if original != period || !reflect.DeepEqual(rule, stored) {
		return errors.New("first hire rule version already has a different definition or period")
	}
	return nil
}

func readFirstHireRule(ctx context.Context, tx pgx.Tx, customer, arrival string) (hireprep.Rule, f.Period, error) {
	var rule hireprep.Rule
	var period f.Period
	err := tx.QueryRow(ctx, `SELECT version,relationship,action,initial_state,primary_purpose,reasons,accepted_documents,
		require_known_expiry,require_inspection,to_char(lower(period),'YYYY-MM-DD'),coalesce(to_char(upper(period),'YYYY-MM-DD'),'')
		FROM staffing.hire_rule_source WHERE instance_id=$1 AND relationship='Employee' AND period @> $2::date`, customer, arrival).
		Scan(&rule.Version, &rule.Relationship, &rule.Action, &rule.InitialState, &rule.PrimaryPurpose, &rule.Reasons,
			&rule.AcceptedDocuments, &rule.RequireKnownExpiry, &rule.RequireInspection, &period.From, &period.Until)
	if errors.Is(err, pgx.ErrNoRows) {
		err = f.Fail(409, "HIRE_RULE_UNAVAILABLE", "到岗日没有适用的直接入职规则")
	}
	return rule, period, err
}

// FirstHireRule supplies internal command policy. Tenant identity comes from
// the authenticated session. The public command owns person and action checks.
func (s *Store) FirstHireRule(ctx context.Context, cred f.Credentials, arrival string) (rule hireprep.Rule, err error) {
	if !f.ValidDate(arrival) {
		return rule, f.Fail(400, "HIRE_INPUT", "到岗日无效")
	}
	if tx, ok := dbtx.From(ctx); ok {
		guard, e := s.Guard(ctx, tx, cred)
		if e != nil {
			return rule, e
		}
		rule, _, err = readFirstHireRule(ctx, tx, guard.Principal().InstanceID, arrival)
		return
	}
	err = s.within(ctx, cred, false, true, func(u *unit) error {
		var e error
		rule, _, e = readFirstHireRule(ctx, u.tx, u.guard.Principal().InstanceID, arrival)
		return e
	})
	return
}
