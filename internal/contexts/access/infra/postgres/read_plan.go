package postgres

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func (g *guard) PlanReads(ctx context.Context, enterprise string, at time.Time, action access.ReadAction) (access.ReadPlan, error) {
	if err := ctx.Err(); err != nil {
		return access.ReadPlan{}, err
	}
	if !foundation.ValidID(enterprise) || at.IsZero() || !slices.Contains([]access.ReadAction{access.DepartmentRead, access.DepartmentReference, access.DepartmentHistory, access.DepartmentExport, access.BURead, access.BUReference}, action) {
		return access.ReadPlan{}, errors.New("invalid read plan purpose")
	}
	p := access.ReadPlan{Origin: g, Instance: g.p.InstanceID, Enterprise: enterprise, Subject: g.p.ID, Session: g.p.SessionID, Day: at.In(foundation.Shanghai).Format("2006-01-02"), At: at, Action: action, Fields: map[string]access.ReadSet{}}
	add := func(s access.ReadSet, grant access.Grant) access.ReadSet {
		switch grant.ScopeKind {
		case "enterprise":
			s.All = true
		case "bu":
			s.BUs = append(s.BUs, grant.ScopeID)
		case "department":
			s.Departments = append(s.Departments, grant.ScopeID)
		}
		return s
	}
	for _, grant := range g.grants {
		if grant.Action != string(action) {
			continue
		}
		if grant.ScopeKind == "enterprise" && grant.ScopeID != enterprise {
			continue
		}
		if !slices.Contains([]string{"enterprise", "bu", "department"}, grant.ScopeKind) || grant.ScopeKind == "department" && !strings.HasPrefix(string(action), "department.") {
			continue
		}
		p.Allowed = add(p.Allowed, grant)
		for _, field := range grant.ReadFields {
			p.Fields[field] = add(p.Fields[field], grant)
		}
	}
	canonical := func(s access.ReadSet) access.ReadSet {
		slices.Sort(s.BUs)
		s.BUs = slices.Compact(s.BUs)
		slices.Sort(s.Departments)
		s.Departments = slices.Compact(s.Departments)
		return s
	}
	p.Allowed = canonical(p.Allowed)
	for k, v := range p.Fields {
		p.Fields[k] = canonical(v)
	}
	return p, nil
}

func (g *guard) DecidePlan(ctx context.Context, p access.ReadPlan, r access.ReadIntent) (access.Decision, error) {
	if err := ctx.Err(); err != nil {
		return access.Decision{}, err
	}
	if p.Origin != g || p.Instance != g.p.InstanceID || p.Subject != g.p.ID || p.Session != g.p.SessionID || p.Enterprise != r.EnterpriseID || string(p.Action) != r.Action || !slices.Contains([]string{"object", "reference"}, r.Kind) {
		return access.Decision{}, errors.New("read plan binding differs")
	}
	out := access.Decision{ReadFields: []string{}, WriteFields: []string{}}
	if !validIntentScope(r.Intent(), p.Day) {
		return out, nil
	}
	bu, department := r.BUID, ""
	if r.DepartmentTarget != nil {
		if r.DepartmentTarget.ScopeFact.EvaluatedAt != p.At {
			return out, nil
		}
		bu, department = r.DepartmentTarget.ScopeFact.ManagementBUID, r.DepartmentTarget.DepartmentID
	}
	out.Allowed = p.Allowed.Contains(bu, department)
	if out.Allowed {
		out.ReadFields = make([]string, 0, len(p.Fields))
		for field, set := range p.Fields {
			if set.Contains(bu, department) {
				out.ReadFields = append(out.ReadFields, field)
			}
		}
	}
	slices.Sort(out.ReadFields)
	return out, nil
}
