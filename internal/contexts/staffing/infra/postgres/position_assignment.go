package postgres

import (
	"context"
	"errors"
	"slices"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"github.com/jackc/pgx/v5"
)

func (u *unit) positionReadable(ctx context.Context, enterprise string) (bool, error) {
	for _, action := range []string{"position.read", "position.history.read"} {
		decision, e := u.guard.Decide(ctx, access.Intent{EnterpriseID: enterprise, Kind: "position", Action: action})
		if e != nil {
			return false, e
		}
		if !decision.Allowed {
			return false, nil
		}
		for _, key := range []string{"name", "effectiveDate", "status", "departmentId", "jobId"} {
			if !slices.Contains(decision.ReadFields, key) {
				return false, nil
			}
		}
	}
	return true, nil
}

func (u *unit) checkPositionAssignment(ctx context.Context, in s.AssignmentCheck, head string, rows []s.AssignmentPolicy, locked bool, retained []f.Period, owner, prior string, listFull bool) ([]s.AssignmentSegment, error) {
	if !f.ValidID(in.Position) || !in.Period.Valid() || in.Category == "" || in.Purpose == "" {
		return nil, f.Fail(400, "M03_INVALID_INPUT", "请核对岗位、任期、人员类别与用途")
	}
	rules := app.QuotaPolicies(in.Enterprise, rows)
	var boundaries []string
	for _, r := range rules {
		if r.Purpose == in.Purpose && r.Category == in.Category {
			boundaries = append(boundaries, r.From, r.Until)
		}
	}
	jobRequired, positionRequired, quotaRequired := false, false, false
	for _, span := range in.Split(boundaries) {
		rule, e := d.ApplicablePolicy(rules, in.Enterprise, in.Category, in.Purpose, span.From)
		if e != nil {
			return nil, e
		}
		quotaRequired = quotaRequired || rule.Consumes
		if rule.Mode == "JobOnly" {
			jobRequired = true
		} else if rule.Mode == "PositionManaged" {
			positionRequired = true
		}
	}
	if !positionRequired {
		return nil, f.Fail(409, "M03_POLICY_CONFLICT", "仅按职务不能指定岗位")
	}
	visible, e := u.positionReadable(ctx, in.Enterprise)
	if e != nil {
		return nil, e
	}
	if !visible {
		return nil, f.Fail(403, "ACCESS_DENIED", "当前权限无法采用岗位安排")
	}
	pos, e := u.Load(ctx, d.Position, in.Position, "")
	if e != nil {
		return nil, e
	}
	if len(pos.Records) == 0 {
		return nil, f.Fail(404, "POSITION_UNAVAILABLE", "所选岗位不可用")
	}
	if locked {
		previous := pos.Source
		departments := d.QuotaEligibility(pos).Departments()
		if e = u.LockQuotaDepartments(ctx, departments); e != nil {
			return nil, e
		}
		if e = u.LockQuotaTarget(ctx, in.Position); e != nil {
			return nil, e
		}
		pos, e = u.Load(ctx, d.Position, in.Position, "")
		if e != nil {
			return nil, e
		}
		if pos.Source != previous {
			return nil, f.Fail(409, "M03_SOURCE_CHANGED", "所选岗位已变化，请重新核对")
		}
	}
	evidence, e := u.PositionUsage(ctx, in.Position)
	if e != nil {
		return nil, e
	}
	if prior != "" {
		peer := exchangePeer(ctx, owner, prior, in.Position)
		evidence.Uses = slices.DeleteFunc(evidence.Uses, func(v d.PositionUse) bool {
			return v.Owner == owner && (owner == "M04" && (v.Assignment == prior || peer != "" && v.Assignment == peer) || owner == "M05" && v.Source == prior)
		})
	}
	settings, e := u.replenishmentRows(ctx, in.Position)
	if e != nil {
		return nil, e
	}
	succession, hasSuccession := ctx.Value(successionContext{}).(successionAdmission)
	hasSuccession = hasSuccession && owner == "M04" && succession.recipient == prior && succession.position == in.Position
	if hasSuccession {
		boundaries = append(boundaries, succession.source.From, succession.source.Until)
	}
	proof, hasDuty := ctx.Value(dutyContext{}).(dutyAdmission)
	hasDuty = hasDuty && owner == "M04" && prior == proof.assignment && in.Position == proof.evidence.Position && in.Purpose == "transfer"
	var quota d.StaffingQuota
	if quotaRequired {
		quota, e = u.LoadPositionQuota(ctx, in.Position, "")
		if e != nil {
			return nil, e
		}
	}
	for _, r := range pos.Records {
		boundaries = append(boundaries, r.Date, r.End)
	}
	for _, c := range evidence.Coverage {
		boundaries = append(boundaries, c.From, c.Until)
	}
	for _, q := range quota.Records {
		boundaries = append(boundaries, q.Date)
	}
	for _, use := range evidence.Uses {
		boundaries = append(boundaries, use.From, use.Until)
	}
	for _, r := range retained {
		boundaries = append(boundaries, r.From, r.Until)
	}
	for _, r := range settings {
		boundaries = append(boundaries, r.Date)
	}
	var out []s.AssignmentSegment
	for _, p := range in.Split(boundaries) {
		rule, e := d.ApplicablePolicy(rules, in.Enterprise, in.Category, in.Purpose, p.From)
		if e != nil {
			return nil, e
		}
		targets := in
		targets.Period = p
		var source, revision, name, coverage, intake, recruiting, quotaRevision, handover string
		actual := hasDuty && proof.evidence.Includes(p.From)
		if actual && rule.Mode != "PositionManaged" {
			return nil, f.Fail(409, "DUTY_POLICY", p.From+" 起没有可采用的按岗位任职规则")
		}
		dedicated := hasSuccession && succession.source.Includes(p.From)
		if dedicated && !rule.Consumes {
			return nil, f.Fail(409, "SUCCESSION_POLICY", p.From+" 起不适用交接占编")
		}
		seats := 0
		if rule.Mode == "PositionManaged" {
			i := d.At(pos, p.From)
			if i < 0 || !pos.Records[i].Active() {
				return nil, f.Fail(409, "POSITION_PERIOD", p.From+" 起没有有效岗位")
			}
			record := pos.Records[i]
			targets.Department, targets.Job = record.Data["departmentId"], record.Data["jobId"]
			if !jobRequired && p.From == in.From && (in.Department != "" && targets.Department != in.Department || in.Job != "" && targets.Job != in.Job) {
				return nil, f.Fail(409, "POSITION_TARGET_CHANGED", "岗位归属与拟任安排不符")
			}
			intent := d.UsageIntent{UsageSpan: d.UsageSpan{From: p.From, Until: p.Until}, Enterprise: in.Enterprise, Category: in.Category, Purpose: in.Purpose, Owner: owner}
			admission := evidence
			if dedicated {
				// Only this incumbent's regular seat can be credited in the overlap.
				admission.Uses = slices.DeleteFunc(slices.Clone(evidence.Uses), func(v d.PositionUse) bool {
					return v.Owner == "M04" && v.Assignment == succession.source.Assignment && v.Effect == "occupied"
				})
				handover = succession.source.ID
			}
			if _, e = admission.PlanUse(intent, rules, quota, d.QuotaEligibility(pos)); e != nil {
				var fault *f.Fault
				if !errors.As(e, &fault) || fault.Code != "M03_QUOTA_SHORTAGE" {
					return nil, e
				}
				if !listFull && !actual {
					return nil, f.Fail(409, fault.Code, record.Data["name"]+"："+fault.Message)
				}
			}
			for _, c := range evidence.Coverage {
				if c.Owner == "M05" && c.Contains(p.From) {
					recruiting = c.Version
				}
				if c.Owner == "M04" && c.Contains(p.From) {
					coverage = c.Version
				}
			}
			if rule.Consumes {
				seats = 1
				qi := quota.At(p.From)
				if qi >= 0 {
					quotaRevision = quota.Records[qi].Revision
				}
			}
			intake = coverage
			if r := replenishmentAt(settings, p.From); r != nil {
				intake = r.Revision
				if r.Paused && !actual && !slices.ContainsFunc(retained, func(v f.Period) bool { return v.Includes(p.From) }) {
					return nil, f.Fail(409, "POSITION_PAUSED", p.From+" 起岗位暂停补员")
				}
			}
			source, revision, name = pos.Source, record.Revision, record.Data["name"]
		} else if !f.ValidID(targets.Department) || !f.ValidID(targets.Job) {
			return nil, f.Fail(409, "M03_JOB_ARRANGEMENT", p.From+" 起按职务办理，请选部门与职务")
		}
		pieces, e := app.AssignmentTargets(ctx, u, targets, head, rule.Version, locked)
		if e != nil {
			return nil, e
		}
		for _, piece := range pieces {
			piece.DepartmentID, piece.JobID = targets.Department, targets.Job
			if source != "" {
				piece.Position = in.Position
				piece.PositionSource = source
				piece.PositionRevision = revision
				piece.PositionName = name
				piece.ReplenishmentSource = intake
				piece.CoverageSource = coverage
				if rule.Consumes {
					piece.RecruitingCoverage = recruiting
					piece.Handover = handover
					piece.QuotaSource = quota.Source
					piece.QuotaRevision = quotaRevision
					piece.Seats = seats
				}
			}
			out = append(out, piece)
		}
	}
	return out, nil
}

func (u *unit) replenishmentRows(ctx context.Context, id string) ([]s.ReplenishmentRecord, error) {
	rows, e := u.tx.Query(ctx, `SELECT business_date::text,paused,reason,revision,recorded_at FROM staffing.position_replenishment WHERE instance_id=$1 AND position_id=$2 ORDER BY business_date`, u.store.Instance, id)
	if e != nil {
		return nil, e
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[s.ReplenishmentRecord])
}
func replenishmentAt(rows []s.ReplenishmentRecord, day string) *s.ReplenishmentRecord {
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].Date <= day {
			return &rows[i]
		}
	}
	return nil
}

func (u *unit) positionEmploymentScope(ctx context.Context, id string, term f.Period, person, employer, permission, purpose string) (bool, error) {
	pos, e := u.Load(ctx, d.Position, id, "")
	if e != nil {
		return false, e
	}
	_, policies, e := u.arrangementRules(ctx)
	if e != nil {
		return false, e
	}
	rules := app.QuotaPolicies(u.enterprise.ID, policies)
	var days []string
	for _, r := range rules {
		if r.Category == "Employee" && r.Purpose == purpose {
			days = append(days, r.From, r.Until)
		}
	}
	for _, r := range pos.Records {
		days = append(days, r.Date, r.End)
	}
	for _, span := range term.Split(days) {
		rule, ruleErr := d.ApplicablePolicy(rules, u.enterprise.ID, "Employee", purpose, span.From)
		if ruleErr == nil && rule.Mode == "JobOnly" {
			continue
		}
		i := d.At(pos, span.From)
		if i < 0 || !pos.Records[i].Active() {
			continue
		}
		ok, e := u.employmentAllowed(ctx, permission, person, employer, pos.Records[i].Data["departmentId"], employer == "")
		if e != nil || !ok {
			return false, e
		}
	}
	return pos.ID != "", nil
}
