package application

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/application/buauth"
	"slices"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
)

// Each declared facet retains its existing permission; no joint-entry grant is needed.
func correctionIntent(tl Timeline, record, enterprise string, in org.CorrectInput) access.Intent {
	original := "CHANGE"
	for _, r := range tl.Records {
		if r.ID == record {
			original = r.Type
		}
	}
	final := original
	if in.Classification != nil {
		final = in.Classification.OperationType
	}
	checks := []access.Intent{}
	add := func(action string, fs []string) {
		if action != "" {
			checks = append(checks, access.Intent{Action: action, EnterpriseID: enterprise, BUID: tl.BUID, Fields: fs, Kind: "record-correction-facet"})
		}
	}
	if in.Changes != nil {
		fs := []string{}
		for f := range in.Changes {
			fs = append(fs, f)
		}
		slices.Sort(fs)
		add("bu.record.correct", fs)
	}
	if in.NewEffectiveDate != "" {
		for _, kind := range slices.Compact([]string{original, final}) {
			add(buauth.RedateAction(kind), []string{"effectiveDate"})
		}
	}
	if in.Classification != nil {
		for _, kind := range slices.Compact([]string{original, final}) {
			add(buauth.ReclassifyAction(kind), []string{"operationType", "operationReason"})
		}
	}
	action := "bu.record.correct"
	if len(checks) > 0 {
		action = checks[0].Action
	}
	return access.Intent{Action: action, EnterpriseID: enterprise, BUID: tl.BUID, Fields: []string{}, Requirements: checks, Kind: "record-correction"}
}
