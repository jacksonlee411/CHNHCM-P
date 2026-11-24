package application

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application/buauth"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"slices"
)

type maintenanceCommand struct {
	org.MaintainCommand
	PairItems          []domain.Item
	PairExpectedSource string
	PairAction         string
}

func pairField(field, message string) error {
	return &foundation.Fault{Status: 400, Code: "M02_INVALID_INPUT", Field: field, Message: message}
}
func fixedCommand(in org.MaintainCommand) (maintenanceCommand, error) {
	c := maintenanceCommand{MaintainCommand: in}
	switch in.Action {
	case "REVISE_TIMELINE":
		return c, foundation.Fail(400, "M02_ACTION_NOT_ALLOWED", "此操作不可用")
	case "INSERT_LIFECYCLE_PAIR":
		v := in.InsertPair
		if !foundation.ValidID(v.AnchorRecordID) {
			return c, pairField("anchorRecordId", "请选择当前来源中的记录")
		}
		if !foundation.ValidDate(v.DeactivationDate) {
			return c, pairField("deactivationDate", "请选择有效的停用日期")
		}
		if !foundation.ValidDate(v.ReactivationDate) {
			return c, pairField("reactivationDate", "请选择有效的重新启用日期")
		}
		if domain.OperationReasons("DEACTIVATE")[v.DeactivationReason] == "" {
			return c, pairField("deactivationReason", "请选择停用原因")
		}
		if domain.OperationReasons("REACTIVATE")[v.ReactivationReason] == "" {
			return c, pairField("reactivationReason", "请选择重新启用原因")
		}
		c.PairExpectedSource = v.ExpectedSourceVersion
		c.PairItems = []domain.Item{{ID: foundation.NewID(), Action: "ADD", Date: v.DeactivationDate, Type: "DEACTIVATE", Reason: v.DeactivationReason}, {ID: foundation.NewID(), Action: "ADD", Date: v.ReactivationDate, Type: "REACTIVATE", Reason: v.ReactivationReason}}
	case "DELETE_LIFECYCLE_PAIR":
		v := in.DeletePair
		for _, id := range []string{v.DeactivationRecordID, v.ReactivationRecordID, v.DeactivationRevision, v.ReactivationRevision} {
			if !foundation.ValidID(id) {
				return c, pairField("", "停用与重新启用的记录依据不完整")
			}
		}
		if v.DeactivationRecordID == v.ReactivationRecordID {
			return c, pairField("", "两条记录不能相同")
		}
		c.PairExpectedSource = v.ExpectedSourceVersion
		c.PairItems = []domain.Item{{ID: foundation.NewID(), Action: "DELETE", RecordID: v.DeactivationRecordID, ExpectedRevision: v.DeactivationRevision}, {ID: foundation.NewID(), Action: "DELETE", RecordID: v.ReactivationRecordID, ExpectedRevision: v.ReactivationRevision}}
	default:
		return c, nil
	}
	c.PairAction = in.Action
	// Canonical multi-record source/audit storage is retained; callers cannot submit it.
	c.Action = "REVISE_TIMELINE"
	return c, nil
}
func pairIntent(tl Timeline, cmd maintenanceCommand, enterprise string) access.Intent {
	checks := []access.Intent{}
	for i, it := range cmd.PairItems {
		kind := []string{"DEACTIVATE", "REACTIVATE"}[i]
		action := buauth.DeleteAction(kind)
		fields := []string{}
		if cmd.PairAction == "INSERT_LIFECYCLE_PAIR" {
			action = map[string]string{"DEACTIVATE": "bu.deactivate", "REACTIVATE": "bu.reactivate"}[kind]
			fields = []string{"effectiveDate", "operationType", "operationReason"}
		}
		checks = append(checks, access.Intent{Action: action, EnterpriseID: enterprise, BUID: tl.BUID, ItemID: it.ID, Fields: fields, Kind: "lifecycle-pair-item"})
	}
	return access.Intent{Action: checks[0].Action, EnterpriseID: enterprise, BUID: tl.BUID, Fields: []string{}, Kind: "lifecycle-pair", Requirements: checks}
}
func preparePair(ctx context.Context, tl Timeline, enterpriseStart string, cmd maintenanceCommand) ([]domain.Item, error) {
	if cmd.PairAction == "INSERT_LIFECYCLE_PAIR" {
		v := cmd.InsertPair
		if e := domain.ValidateLifecyclePair(tl.Records, v.AnchorRecordID, enterpriseStart, v.DeactivationDate, v.ReactivationDate); e != nil {
			return nil, e
		}
	} else {
		v := cmd.DeletePair
		s, r, e := domain.LifecyclePair(tl.Records, v.DeactivationRecordID)
		if e != nil {
			return nil, e
		}
		if s.ID != v.DeactivationRecordID || r.ID != v.ReactivationRecordID {
			return nil, domainFault("M02_PAIR_NOT_FOUND", "", "两条记录不属于同一停用期间")
		}
		if s.Revision != v.DeactivationRevision || r.Revision != v.ReactivationRevision {
			return nil, domainFault("M02_REVISION_CONFLICT", "", "停用期间已更新，请核对后保存")
		}
	}
	return slices.Clone(cmd.PairItems), ctx.Err()
}

func pairMetadata(cmd maintenanceCommand, base []domain.Record) []org.RequestItem {
	out := pairItemMetadata(cmd.PairItems, base)
	if cmd.PairAction == "DELETE_LIFECYCLE_PAIR" {
		for i := range out {
			kind := "REACTIVATE"
			if out[i].RecordID == cmd.DeletePair.DeactivationRecordID {
				kind = "DEACTIVATE"
			}
			out[i].OriginalType = kind
			out[i].NewType = kind
		}
	}
	return out
}
