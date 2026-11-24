package department

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
)

// ReferencePolicy handles only real relationship inputs. It does not invent names
// or reasons; its read projection cannot be finalized into a write candidate.
type ReferencePolicy struct{ Policy }

func (ReferencePolicy) Normalize(v Data) (Data, error) {
	if v.BU != "" && !foundation.ValidID(v.BU) || v.Parent != "" && !foundation.ValidID(v.Parent) {
		return v, Fault("M02_INVALID_INPUT", "", "引用标识无效")
	}
	return v, nil
}
func (p ReferencePolicy) Apply(v Data, patch *Patch) (Data, error) {
	for key, x := range *patch {
		switch key {
		case "managementBuId":
			if x == nil {
				return v, Fault("M02_INVALID_INPUT", key, "管理业务单元必填")
			}
			v.BU = *x
		case "parentDepartmentId":
			v.Parent = ""
			if x != nil {
				v.Parent = *x
			}
		default:
			return v, Fault("M02_FORBIDDEN_INPUT", key, "引用查询仅接受关系字段")
		}
	}
	return p.Normalize(v)
}

var ReferenceKernel = dated.Kernel[Data, Patch]{Policy: ReferencePolicy{}}

func ValidateBUCoverage(ctx context.Context, s Snapshot, t Timeline) error {
	t.Records = Kernel.CloneRecords(t.Records)
	for i := range t.Records {
		t.Records[i].Data.Parent = ""
	}
	// Qualification depends only on BUs used by this proposed timeline. Keep
	// their complete histories; unrelated candidate BUs cannot affect coverage.
	bus := []BusinessUnit{}
	seen := map[string]bool{}
	for _, r := range t.Records {
		if !seen[r.Data.BU] {
			seen[r.Data.BU] = true
			if b := s.BU(r.Data.BU); b != nil {
				bus = append(bus, *b)
			}
		}
	}
	s.BusinessUnits = bus
	s.Departments = []Timeline{t}
	s = s.Indexed()
	return Validate(ctx, s, Discover(s, t.ID, nil, nil))
}
