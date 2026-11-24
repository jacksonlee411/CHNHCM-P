package postgres

import (
	"slices"
	"time"

	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/compact"
)

// validateTerminalHeader replaces the former per-row format/shape checks. The
// application remains the authority for authorization and candidate semantics.
func validateTerminalHeader(id, outcome, action string, digest []byte, fields []string, at *time.Time, failure *org.FailureDetails, errorCode string, department bool) error {
	if !foundation.ValidID(id) {
		return compact.ErrCorrupt
	}
	if outcome == "not_applied" {
		if action != "" || len(digest) != 0 || fields != nil || at != nil || failure != nil || errorCode != "M02_REQUEST_CLOSED" {
			return compact.ErrCorrupt
		}
		return nil
	}
	if !slices.Contains([]string{"accepted", "no_change", "rejected"}, outcome) || !slices.Contains([]string{"CREATE", "ADD", "CORRECT", "DELETE", "REDATE", "DEACTIVATE", "REACTIVATE", "REVISE_TIMELINE"}, action) || len(digest) != 32 || fields == nil || !slices.IsSorted(fields) {
		return compact.ErrCorrupt
	}
	allowed := []string{"code", "name", "nameEn", "effectiveDate", "operationType", "operationReason"}
	if department {
		allowed = []string{"code", "name", "shortName", "managementBuId", "parentDepartmentId", "effectiveDate", "operationType", "operationReason"}
	}
	for i, f := range fields {
		if !slices.Contains(allowed, f) || (i > 0 && fields[i-1] == f) {
			return compact.ErrCorrupt
		}
	}
	if outcome == "accepted" || outcome == "no_change" {
		if at == nil || at.IsZero() || failure != nil || errorCode != "" {
			return compact.ErrCorrupt
		}
	}
	if outcome == "rejected" {
		if errorCode == "" || failure == nil || failure.FormatVersion != 1 || len(failure.Issues) == 0 || failure.Issues[0].Code != errorCode {
			return compact.ErrCorrupt
		}
		for _, issue := range failure.Issues {
			if issue.Code == "" {
				return compact.ErrCorrupt
			}
		}
	}
	return nil
}

func validSpan(start, end string) bool {
	return foundation.ValidDate(start) && (end == "" || (foundation.ValidDate(end) && start < end))
}

// Index each original summary once. Membership remains the exact record and
// revision pair; duplicates and malformed evidence are never overwritten.
func summaryIndex(summaries []org.OperationSummary) map[[2]string]bool {
	out := make(map[[2]string]bool, len(summaries))
	for _, s := range summaries {
		key := [2]string{s.RecordID, s.Revision}
		if out[key] || !foundation.ValidID(s.OperationSourceVersion) || !foundation.ValidID(s.Operator.SubjectID) {
			return nil
		}
		if _, err := time.Parse(time.RFC3339Nano, s.RecordedAt); err != nil {
			return nil
		}
		out[key] = true
	}
	return out
}
