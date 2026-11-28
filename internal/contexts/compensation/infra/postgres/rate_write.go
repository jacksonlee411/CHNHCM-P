package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	app "chnhcm.local/chnhcm/internal/contexts/compensation/application"
	d "chnhcm.local/chnhcm/internal/contexts/compensation/domain"
	c "chnhcm.local/chnhcm/internal/contexts/compensation/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) SaveRate(ctx context.Context, cred foundation.Credentials, id, submission string, input c.WriteRate) (c.Receipt, error) {
	fields, err := app.Fields(input)
	if err != nil {
		return c.Receipt{}, err
	}
	return saveCommand(ctx, s, cred, rateCommands, id, submission, input.Action, fields, input, s.applyRate)
}
func (s *Store) applyRate(ctx context.Context, u *unit, id string, in c.WriteRate, explicit []string) (c.Receipt, error) {
	var out c.Receipt
	var state app.RateState
	var err error
	if id == "" {
		state, err = app.NewRate(in)
		state.Enterprise = u.enterprise.ID
	} else {
		state, err = s.loadRate(ctx, u, id, true)
	}
	if err != nil {
		return out, err
	}
	if state.Version != in.ExpectedVersion {
		return out, d.Conflict()
	}
	if in.EffectiveDate != "" && in.EffectiveDate < u.enterprise.ValidFrom {
		return out, d.Invalid("effectiveDate", "生效日期早于企业起始日期")
	}
	if next, changing := in.Changes["form"]; changing && next != nil && id != "" {
		for _, r := range state.Records {
			if r.ID != in.RecordID || r.Data.Amounts.Form == *next {
				continue
			}
			removed := []string{"amount"}
			if r.Data.Amounts.Form == "RANGE" {
				removed = []string{"minimum", "guidance", "maximum"}
			}
			if _, e := u.rateWriteGrant(ctx, in.Action, append(slices.Clone(explicit), removed...)); e != nil {
				return out, e
			}
			break
		}
	}
	gd, err := u.require(ctx, "grade.read", nil)
	if err != nil {
		return out, err
	}
	if !slices.Contains(gd.ReadFields, "name") {
		return out, denied()
	}
	grade, err := s.Grades.CompensationGrade(ctx, u.enterprise.ID, state.Key.GradeID, true)
	if err != nil {
		return out, err
	}
	change, err := app.PrepareRate(state, in, grade)
	if err != nil {
		return out, err
	}
	out = c.Receipt{ObjectID: state.ID, RecordID: change.Target.ID, Revision: change.Target.Revision, Version: state.Version, Outcome: "no_change"}
	if !change.Changed {
		return out, nil
	}
	if id != "" {
		if err = s.protectPlanUse(ctx, u, state, change.Records); err != nil {
			return out, err
		}
	}
	op, err := u.guard.OperatorSnapshot(ctx)
	if err != nil {
		return out, err
	}
	operator, err := json.Marshal(op)
	if err != nil {
		return out, err
	}
	out.Outcome, out.Version = "accepted", foundation.NewID()
	if id == "" {
		_, err = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate(instance_id,enterprise_id,id,grade_id,currency,frequency,unit,places,head) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, s.Instance, state.Enterprise, state.ID, state.Key.GradeID, state.Key.Currency, state.Key.Frequency, state.Key.Unit, state.Places, out.Version)
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return c.Receipt{}, foundation.Fail(409, "M09_DUPLICATE_RATE", "此职级及计量口径已有标准，请选用已有标准")
		}
		if err != nil {
			return out, err
		}
	}
	indices := make(map[string]int, len(state.Records))
	for i, record := range state.Records {
		indices[record.ID] = i
	}
	for i, r := range change.Records {
		old, exists := indices[r.ID]
		if !exists || state.Records[old].Revision != r.Revision {
			raw, e := json.Marshal(r)
			if e != nil {
				return out, e
			}
			if _, e = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate_revision(instance_id,rate_id,record_id,revision,fact) VALUES($1,$2,$3,$4,$5)`, s.Instance, state.ID, r.ID, r.Revision, raw); e != nil {
				return out, e
			}
		}
		p := change.Records.Period(i)
		if exists && state.Records[old].Revision == r.Revision && state.Records.Period(old) == p {
			continue
		}
		_, err = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate_period(instance_id,rate_id,record_id,revision,start_date,until_date,active) VALUES($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT(instance_id,rate_id,record_id) DO UPDATE SET revision=excluded.revision,start_date=excluded.start_date,until_date=excluded.until_date,active=excluded.active`, s.Instance, state.ID, r.ID, r.Revision, r.Start, optional(p.Until), r.Active)
		if err != nil {
			return out, err
		}
	}
	if in.Action == "DELETE" {
		if _, err = u.tx.Exec(ctx, "DELETE FROM compensation.grade_rate_period WHERE instance_id=$1 AND rate_id=$2 AND record_id=$3", s.Instance, state.ID, change.Target.ID); err != nil {
			return out, err
		}
	}
	var previousRecord, previousRevision any
	if change.Previous != nil {
		previousRecord, previousRevision = change.Previous.ID, change.Previous.Revision
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate_version(instance_id,rate_id,id,predecessor,action,record_id,revision,previous_record_id,previous_revision,until_date,previous_until,operator) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, s.Instance, state.ID, out.Version, optional(state.Version), in.Action, change.Target.ID, change.Target.Revision, previousRecord, previousRevision, optional(change.Until), optional(change.PreviousUntil), operator)
	if err != nil {
		return out, err
	}
	if id != "" {
		if _, err = u.tx.Exec(ctx, "UPDATE compensation.grade_rate SET head=$3 WHERE instance_id=$1 AND id=$2", s.Instance, state.ID, out.Version); err != nil {
			return out, err
		}
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO compensation.grade_rate_outbox VALUES($1,$2,$3,jsonb_build_object('type','GradeRateChanged','sourceVersion',$3::uuid,'recordId',$4::uuid))`, s.Instance, state.ID, out.Version, change.Target.ID)
	return out, err
}
