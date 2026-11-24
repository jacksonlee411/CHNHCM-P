package application

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"reflect"
	"testing"
)

func TestFixedPairCommandsAndAuthority(t *testing.T) {
	id := foundation.NewID()
	source := foundation.NewID()
	anchor := foundation.NewID()
	if _, e := fixedCommand(org.MaintainCommand{Action: "REVISE_TIMELINE"}); e == nil {
		t.Fatal("generic command accepted")
	}
	c, e := fixedCommand(org.MaintainCommand{Action: "INSERT_LIFECYCLE_PAIR", BUID: id, InsertPair: org.InsertLifecyclePairInput{ExpectedSourceVersion: source, AnchorRecordID: anchor, DeactivationDate: "2026-03-01", ReactivationDate: "2026-04-01", DeactivationReason: "BUSINESS_CLOSURE", ReactivationReason: "BUSINESS_RESUMPTION"}})
	if e != nil {
		t.Fatal(e)
	}
	if c.Action != "REVISE_TIMELINE" || len(c.PairItems) != 2 || c.PairItems[0].Type != "DEACTIVATE" || c.PairItems[1].Type != "REACTIVATE" {
		t.Fatal(c)
	}
	intent := pairIntent(Timeline{BUID: id, Records: []domain.Record{{ID: anchor, Type: "CREATE"}}}, c, "enterprise")
	if len(intent.Requirements) != 2 || intent.Requirements[0].Action != "bu.deactivate" || intent.Requirements[1].Action != "bu.reactivate" || intent.Action == "bu.timeline.revise" {
		t.Fatal(intent)
	}
	if !reflect.DeepEqual(intent.Requirements[1].Fields, []string{"effectiveDate", "operationType", "operationReason"}) {
		t.Fatal(intent)
	}
	c, e = fixedCommand(org.MaintainCommand{Action: "DELETE_LIFECYCLE_PAIR", DeletePair: org.DeleteLifecyclePairInput{ExpectedSourceVersion: source, DeactivationRecordID: foundation.NewID(), DeactivationRevision: foundation.NewID(), ReactivationRecordID: foundation.NewID(), ReactivationRevision: foundation.NewID()}})
	if e != nil {
		t.Fatal(e)
	}
	intent = pairIntent(Timeline{BUID: id}, c, "enterprise")
	if intent.Requirements[0].Action != "bu.deactivation.delete" || intent.Requirements[1].Action != "bu.reactivation.delete" || len(intent.Requirements[0].Fields) != 0 {
		t.Fatal(intent)
	}
}
