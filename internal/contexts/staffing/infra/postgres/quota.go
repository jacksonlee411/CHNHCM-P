package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	g "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres/generated"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

func (u *unit) LockQuotaTarget(ctx context.Context, position string) error {
	return u.guardLock(ctx, u.tx, false, objectGuard(d.Position), u.store.Instance, position)
}
func (u *unit) PositionPeriods(ctx context.Context, position string) (d.QuotaPeriods, error) {
	var pos d.QuotaPeriods
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return pos, err
	}
	rows, err := g.New(u.tx).PositionQuotaPeriods(ctx, g.PositionQuotaPeriodsParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, ID: position})
	if err != nil {
		return pos, err
	}
	if len(rows) > 0 {
		pos = d.QuotaPeriods{ID: position, Code: rows[0].Code, Source: rows[0].SourceID, Kind: d.Position}
	}
	for _, row := range rows {
		pos.Records = append(pos.Records, d.QuotaPeriod{Date: row.StartDay, End: row.EndDay, Name: row.Name, DepartmentID: row.DepartmentID, Active: row.Active})
	}
	return pos, nil
}
func (u *unit) LoadPositionQuota(ctx context.Context, position, source string) (q d.StaffingQuota, err error) {
	return u.LoadTargetQuota(ctx, d.QuotaTarget{Kind: "position", ID: position}, source)
}
func (u *unit) LoadTargetQuota(ctx context.Context, target d.QuotaTarget, source string) (q d.StaffingQuota, err error) {
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return
	}
	raw, err := g.New(u.tx).LoadQuotaSnapshot(ctx, g.LoadQuotaSnapshotParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, TargetID: target.ID, TargetKind: target.Kind, SourceVersion: source})
	if errors.Is(err, pgx.ErrNoRows) {
		return q, nil
	}
	if err != nil {
		return
	}
	err = json.Unmarshal(raw, &q)
	return
}
func (u *unit) QuotaIdentity(ctx context.Context, id string) (d.QuotaTarget, error) {
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return d.QuotaTarget{}, err
	}
	pos, err := g.New(u.tx).QuotaIdentity(ctx, g.QuotaIdentityParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return d.QuotaTarget{}, nil
	}
	return d.QuotaTarget{Kind: pos.TargetKind, ID: pos.TargetID}, err
}
func (u *unit) QuotaCandidates(ctx context.Context, date string) ([]app.QuotaCandidate, error) {
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return nil, err
	}
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	rows, err := g.New(u.tx).QuotaCandidates(ctx, g.QuotaCandidatesParams{InstanceID: u.store.Instance, EnterpriseID: ent.ID, Day: pgtype.Date{Time: day, Valid: true}})
	if err != nil {
		return nil, err
	}
	out := make([]app.QuotaCandidate, 0, len(rows))
	for _, row := range rows {
		period := d.QuotaPeriod{Date: row.StartDay, Name: row.Name, DepartmentID: row.DepartmentID, Active: row.Active}
		out = append(out, app.QuotaCandidate{PositionID: row.ID, Code: row.Code, RecordID: row.RecordID, Amount: row.Amount, Position: period})
	}
	return out, nil
}
func (u *unit) PreviousQuotaSource(ctx context.Context, id, source string) (string, error) {
	previous, e := g.New(u.tx).PreviousQuotaSource(ctx, g.PreviousQuotaSourceParams{InstanceID: u.store.Instance, ObjectID: id, SourceID: source})
	if errors.Is(e, pgx.ErrNoRows) {
		return "", nil
	}
	return previous, e
}
func (u *unit) QuotaHistory(ctx context.Context, id string) ([]app.QuotaHistoryEntry, error) {
	rows, err := g.New(u.tx).QuotaHistory(ctx, g.QuotaHistoryParams{InstanceID: u.store.Instance, ObjectID: id})
	if err != nil {
		return nil, err
	}
	out := make([]app.QuotaHistoryEntry, 0, len(rows))
	for _, r := range rows {
		entry := app.QuotaHistoryEntry{Source: r.SourceID, Action: r.Action, RecordID: r.RecordID, RecordedAt: r.RecordedAt.Time}
		if err := json.Unmarshal(r.Operator, &entry.Operator); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(r.Snapshot, &entry.Snapshot); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}
func (u *unit) SaveQuota(ctx context.Context, p d.QuotaPlan, c s.QuotaCommand, op access.OperatorSnapshot) error {
	q := p.After
	ent, err := u.Enterprise(ctx, false)
	if err != nil {
		return err
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.quota(instance_id,enterprise_id,id,target_kind,target_id,source_id) VALUES($1,$2,$3,$4,$5,$6)
 ON CONFLICT(instance_id,id) DO UPDATE SET source_id=EXCLUDED.source_id`, u.store.Instance, ent.ID, q.ID, q.Target().Kind, q.Target().ID, q.Source)
	if err != nil {
		return err
	}
	_, err = u.tx.Exec(ctx, `INSERT INTO staffing.quota_source(instance_id,object_id,source_id,source_no,submission_id,subject_id,action,record_id,recorded_at,snapshot,operator) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, u.store.Instance, q.ID, q.Source, q.Number, c.SubmissionID, op.SubjectID, c.Action, p.Selected, u.Now(), q, op)
	if err != nil {
		return err
	}
	if _, err = u.tx.Exec(ctx, "DELETE FROM staffing.quota_record WHERE instance_id=$1 AND object_id=$2", u.store.Instance, q.ID); err != nil {
		return err
	}
	batch := &pgx.Batch{}
	for i, r := range q.Records {
		var end any
		if i+1 < len(q.Records) {
			end = q.Records[i+1].Date
		}
		batch.Queue("INSERT INTO staffing.quota_record(instance_id,object_id,record_id,revision,start_date,end_date,amount) VALUES($1,$2,$3,$4,$5,$6,$7)", u.store.Instance, q.ID, r.ID, r.Revision, r.Date, end, r.Amount)
	}
	if err = u.tx.SendBatch(ctx, batch).Close(); err != nil {
		return err
	}
	event := s.View{"type": "staffing.quota.changed", "quotaId": q.ID, "sourceVersion": q.Source, "submissionId": c.SubmissionID, "action": c.Action}
	event[q.Target().Kind+"Id"] = q.Target().ID
	_, err = u.tx.Exec(ctx, "INSERT INTO staffing.quota_outbox VALUES($1,$2,$3,$4)", u.store.Instance, q.ID, q.Source, event)
	return err
}

// PositionUsage reads adopted source evidence in the current physical transaction.
// With no owners connected, all positions remain in the quota-administration stage.
func (u *unit) PositionUsage(ctx context.Context, position string) (d.QuotaUsage, error) {
	if u.store.Usage == nil {
		return d.QuotaUsage{}, nil
	}
	return u.store.Usage(dbtx.With(ctx, u.tx), u.store.Instance, position)
}

// CheckRegularUse participates in the source owner's transaction. The caller
// holds its share guard, persists its own fact and outbox, and
// commits only after saving the returned dated adoptions.
var _ s.RegularQuotaGuard = (*Store)(nil)

func (store *Store) CheckRegularUse(ctx context.Context, request s.RegularUseRequest) ([]s.RegularUseSegment, error) {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return nil, errors.New("staffing quota use requires the owner's transaction")
	}
	u := &unit{store: store, tx: tx, lockRemaining: 2 * time.Second}
	ent, err := u.Enterprise(ctx, true)
	if err != nil {
		return nil, err
	}
	if ent.ID != request.Enterprise {
		return nil, errors.New("staffing quota enterprise mismatch")
	}
	if store.QuotaPolicies == nil {
		return nil, errors.New("staffing quota policy source unavailable")
	}
	intent := d.UsageIntent{UsageSpan: d.UsageSpan{From: request.From, Until: request.Until}, Enterprise: request.Enterprise, Owner: request.Owner, Category: request.Category, Purpose: request.Purpose}
	rules, err := store.QuotaPolicies(ctx, intent)
	if err != nil {
		return nil, err
	}
	var pos d.QuotaPeriods
	var quota d.StaffingQuota
	var evidence d.QuotaUsage
	if request.Position != "" {
		if err = u.LockQuotaTarget(ctx, request.Position); err != nil {
			return nil, err
		}
		if pos, err = u.PositionPeriods(ctx, request.Position); err != nil {
			return nil, err
		}
		for _, rule := range rules {
			overlaps := rule.Contains(intent.From) || intent.Contains(rule.From)
			if rule.Enterprise == intent.Enterprise && rule.Category == intent.Category && rule.Purpose == intent.Purpose && rule.Mode == "PositionManaged" && rule.Consumes && overlaps {
				quota, err = u.LoadPositionQuota(ctx, request.Position, "")
				if err != nil {
					return nil, err
				}
				break
			}
		}
		if evidence, err = u.PositionUsage(ctx, request.Position); err != nil {
			return nil, err
		}
	}
	uses, err := evidence.PlanUse(intent, rules, quota, pos)
	if err != nil {
		return nil, err
	}
	segments := make([]s.RegularUseSegment, 0, len(uses))
	for _, row := range uses {
		segments = append(segments, s.RegularUseSegment{From: row.From, Until: row.Until, Policy: row.Policy, Effect: row.Effect})
	}
	return segments, nil
}

func (s *Store) LockUsagePosition(ctx context.Context, position string) error {
	tx, ok := dbtx.From(ctx)
	if !ok {
		return errors.New("position guard requires owning transaction")
	}
	u := unit{store: s, tx: tx, lockRemaining: 2 * time.Second}
	return u.LockQuotaTarget(ctx, position)
}
