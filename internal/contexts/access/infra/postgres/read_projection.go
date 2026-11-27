package postgres

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// DecideReads rechecks each target under the held policy guard. Only field
// unions for identical matches are shared within this call; no proof is built.
func (g *guard) DecideReads(ctx context.Context, inputs []access.ReadIntent) (access.ReadDecisions, error) {
	defer foundation.BeginStage(ctx, "access.read_projection")()
	g.refresh()
	if g.readActions == nil {
		g.readActions = map[string][]access.Grant{}
		for _, grant := range g.grants {
			g.readActions[grant.Action] = append(g.readActions[grant.Action], grant)
		}
	}
	type signature struct {
		action     string
		mask       uint64
		additional string
	}
	unions := map[signature]int{}
	var previous *access.DepartmentTarget
	var frozen access.DepartmentTarget
	var enterprise, bu, day string
	var at time.Time
	var departmentAction, previousValid bool
	results := access.ReadDecisions{Values: []access.Decision{}, Index: make([]int, 0, len(inputs))}
	for i, read := range inputs {
		if i%256 == 0 {
			if err := ctx.Err(); err != nil {
				return access.ReadDecisions{}, err
			}
		}
		if !slices.Contains([]string{"bu.read", "bu.reference.read", "department.read", "department.reference.read"}, read.Action) || !slices.Contains([]string{"object", "reference"}, read.Kind) {
			return access.ReadDecisions{}, errors.New("read projection does not accept command requirements")
		}
		in := read.Intent()
		isDepartment := strings.HasPrefix(in.Action, "department.")
		valid := false
		if in.DepartmentTarget != nil && in.DepartmentTarget == previous && *in.DepartmentTarget == frozen && in.EnterpriseID == enterprise && in.BUID == bu && isDepartment == departmentAction {
			valid = previousValid
		} else {
			if in.DepartmentTarget != nil {
				if value := in.DepartmentTarget.ScopeFact.EvaluatedAt; value != at || day == "" {
					at = value
					day = value.In(foundation.Shanghai).Format("2006-01-02")
				}
			}
			valid = validIntentScope(in, day)
			previous = in.DepartmentTarget
			if previous != nil {
				frozen = *previous
			}
			enterprise, bu, departmentAction, previousValid = in.EnterpriseID, in.BUID, isDepartment, valid
		}
		key := signature{action: in.Action}
		if valid {
			for j, grant := range g.readActions[in.Action] {
				if matchesValidatedIntent(grant, in) {
					if j < 64 {
						key.mask |= uint64(1) << j
					} else {
						key.additional += strconv.Itoa(j) + ","
					}
				}
			}
		}
		index, found := unions[key]
		if !found {
			index = len(results.Values)
			results.Values = append(results.Values, g.evaluate(in))
			unions[key] = index
		}
		if !g.boundary.IsZero() && results.Values[index].Allowed {
			decision := results.Values[index]
			if in.DepartmentTarget != nil {
				target := *in.DepartmentTarget
				in.DepartmentTarget = &target
			}
			g.checks = append(g.checks, func(current *guard) bool { return coversDecision(current.evaluate(in), decision) })
		}
		results.Index = append(results.Index, index)
	}
	return results, ctx.Err()
}
