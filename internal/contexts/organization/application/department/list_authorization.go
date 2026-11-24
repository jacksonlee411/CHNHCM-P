package department

import (
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"fmt"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
)

type projectedReads struct {
	access.Guard
	departments []projectedDepartmentReads
	snapshot    d.Snapshot
	bus         map[string]projectedBUReads
	decisions   []access.Decision
	masks       []uint16
}

func (g projectedReads) department(id string) (projectedDepartmentReads, bool) {
	i, ok := g.snapshot.DepartmentPosition(id)
	if !ok || i >= len(g.departments) || g.departments[i].target == nil {
		return projectedDepartmentReads{}, false
	}
	return g.departments[i], true
}
func (g projectedReads) objectAt(i int) (access.Decision, uint16) {
	if i < 0 || i >= len(g.departments) {
		return access.Decision{}, 0
	}
	v := g.departments[i]
	if v.object < 0 || v.object >= len(g.masks) {
		return access.Decision{}, 0
	}
	return g.decisions[v.object], g.masks[v.object]
}
func (g projectedReads) objectMask(id string) uint16 {
	v, ok := g.department(id)
	if !ok || v.object < 0 || v.object >= len(g.masks) {
		return 0
	}
	return g.masks[v.object]
}

type projectedDepartmentReads struct {
	target            *access.DepartmentTarget
	object, reference int
}
type projectedBUReads struct {
	enterprise string
	reference  int
}

func (g projectedReads) decision(index int) access.Decision {
	if index < 0 || index >= len(g.decisions) {
		return access.Decision{}
	}
	return g.decisions[index]
}
func (g projectedReads) objectDecision(id string) access.Decision {
	v, ok := g.department(id)
	if !ok {
		return access.Decision{}
	}
	return g.decision(v.object)
}
func (g projectedReads) referenceDecision(id, kind string) access.Decision {
	if kind == "bu" {
		v, ok := g.bus[id]
		if !ok {
			return access.Decision{}
		}
		return g.decision(v.reference)
	}
	v, ok := g.department(id)
	if !ok {
		return access.Decision{}
	}
	return g.decision(v.reference)
}

func (g projectedReads) Decide(_ context.Context, in access.Intent) (access.Decision, error) {
	if len(in.Fields) != 0 || len(in.Requirements) != 0 || in.AffectedScope != nil || in.ItemID != "" {
		return access.Decision{}, fmt.Errorf("command intent used in list projection")
	}
	// Index once by stable identity, then compare the entire frozen scope fact.
	// A different date/source/BU must never reuse this request's decision.
	if in.DepartmentTarget != nil && in.BUID == "" {
		v, ok := g.department(in.DepartmentTarget.DepartmentID)
		if ok && v.target != nil && *v.target == *in.DepartmentTarget && in.EnterpriseID == v.target.EnterpriseID {
			switch {
			case in.Action == "department.read" && in.Kind == "object":
				return g.decision(v.object), nil
			case in.Action == "department.reference.read" && in.Kind == "reference":
				return g.decision(v.reference), nil
			}
		}
	} else if in.DepartmentTarget == nil {
		v, ok := g.bus[in.BUID]
		if ok && in.EnterpriseID == v.enterprise {
			switch {
			case in.Action == "bu.reference.read" && in.Kind == "reference":
				return g.decision(v.reference), nil
			}
		}
	}
	return access.Decision{}, fmt.Errorf("unplanned list authorization intent")
}

type projectedReadUnit struct {
	Unit
	guard access.Guard
}

func (u projectedReadUnit) Access() access.Guard { return u.guard }

// Plan a request-local read set; X03 owns every match and field union.
// The set is never reused for a write, another date or another request.
func referenceAuthorization(ctx context.Context, u Unit, s d.Snapshot, enterprise string, now time.Time, kind string) (Unit, error) {
	return authorizeReads(ctx, u, s, enterprise, now, kind)
}
func authorizeReads(ctx context.Context, u Unit, s d.Snapshot, enterprise string, now time.Time, purpose string) (Unit, error) {
	defer foundation.BeginStage(ctx, "list.authorization")()
	s = s.ReadIndexed()
	const batch = 1024
	intents := make([]access.ReadIntent, 0, batch*2)
	day := now.In(foundation.Shanghai).Format("2006-01-02")
	g := projectedReads{Guard: u.Access(), departments: make([]projectedDepartmentReads, len(s.Departments)), snapshot: s, bus: make(map[string]projectedBUReads, len(s.BusinessUnits))}
	targets := make([]access.DepartmentTarget, len(s.Departments))
	evaluate := func() ([]int, error) {
		v, e := u.Access().DecideReads(ctx, intents)
		if e != nil {
			return nil, e
		}
		if len(v.Index) != len(intents) {
			return nil, fmt.Errorf("incomplete list authorization result")
		}
		offset := len(g.decisions)
		for i, index := range v.Index {
			if index < 0 || index >= len(v.Values) {
				return nil, fmt.Errorf("invalid list authorization index")
			}
			v.Index[i] = index + offset
		}
		g.decisions = append(g.decisions, v.Values...)
		for _, decision := range v.Values {
			g.masks = append(g.masks, readMask(decision.ReadFields))
		}
		return v.Index, nil
	}
	for start := 0; purpose != "bu" && start < len(s.Departments); start += batch {
		end := min(start+batch, len(s.Departments))
		intents = intents[:0]
		for i := start; i < end; i++ {
			targets[i] = factOnDay(s.Departments[i], now, day, "")
			target := &targets[i]
			intents = append(intents, access.ReadIntent{DepartmentTarget: target, EnterpriseID: target.EnterpriseID, Action: "department.reference.read", Kind: "reference"})
		}
		index, e := evaluate()
		if e != nil {
			return nil, e
		}
		for i := start; i < end; i++ {
			at := (i - start)
			object, reference := -1, index[at]
			g.departments[i] = projectedDepartmentReads{target: &targets[i], object: object, reference: reference}
		}
	}
	for start := 0; purpose != "department" && start < len(s.BusinessUnits); start += batch {
		end := min(start+batch, len(s.BusinessUnits))
		intents = intents[:0]
		for _, b := range s.BusinessUnits[start:end] {
			intents = append(intents, access.ReadIntent{Action: "bu.reference.read", EnterpriseID: enterprise, BUID: b.ID, Kind: "reference"})
		}
		index, e := evaluate()
		if e != nil {
			return nil, e
		}
		for i := start; i < end; i++ {
			g.bus[s.BusinessUnits[i].ID] = projectedBUReads{enterprise: enterprise, reference: index[i-start]}
		}
	}
	return projectedReadUnit{Unit: u, guard: g}, nil
}
