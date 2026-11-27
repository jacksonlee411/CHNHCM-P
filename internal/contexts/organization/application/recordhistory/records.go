// Package recordhistory assembles accepted Location and LegalEntity record changes.
package recordhistory

import (
	"slices"

	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// Each object keeps its existing strongly typed operation and on-disk revision format.
type revision[T any] interface {
	~struct{ Before, After *dated.Record[T] }
}

type History[T any, R revision[T]] struct {
	Records   []dated.Record[T]
	Revisions []R
	Summaries []org.OperationSummary
	Selected  string
	Target    *dated.Record[T]
}

// Assemble consumes a validated, ordered candidate. Its protected first record always survives.
func Assemble[R revision[T], T, P any](kernel dated.Kernel[T, P], before []dated.Record[T], candidate dated.Candidate[T], requestedID string, previousSummaries []org.OperationSummary, operation org.OperationSummary) History[T, R] {
	h := History[T, R]{Records: kernel.CloneRecords(candidate.Records), Summaries: slices.Clone(previousSummaries)}
	previous := make(map[string]*dated.Record[T], len(before))
	for i := range before {
		previous[before[i].ID] = &before[i]
	}

	selected := &h.Records[0]
	for i := range h.Records {
		r := &h.Records[i]
		changed := len(before) == 0 || slices.Contains(candidate.Impact.AffectedRecordIDs, r.ID)
		if !foundation.ValidID(r.ID) {
			r.ID = foundation.NewID()
		}
		if r.ID == requestedID {
			selected = r
		}
		if changed {
			r.Revision = foundation.NewID()
			r.RevisionNo++
		}
		if i > 0 && r.Type == "DEACTIVATE" {
			// Finalize marked the dependency. Bind the actual predecessor revision allocated above.
			predecessor := &h.Records[i-1]
			r.Data = kernel.Policy.Clone(predecessor.Data)
			r.InheritedRecord, r.InheritedRevision = predecessor.ID, predecessor.Revision
		}
		if !changed {
			continue
		}
		old := previous[r.ID]
		if old == nil {
			selected = r
		}
		h.Revisions = append(h.Revisions, R{Before: old, After: r})
		h.Summaries = slices.DeleteFunc(h.Summaries, func(s org.OperationSummary) bool { return s.RecordID == r.ID })
		summary := operation
		summary.RecordID, summary.Revision = r.ID, r.Revision
		h.Summaries = append(h.Summaries, summary)
	}

	h.Selected, h.Target = selected.ID, selected
	for i := range candidate.Deleted {
		removed := &candidate.Deleted[i]
		h.Revisions = append(h.Revisions, R{Before: removed})
		h.Summaries = slices.DeleteFunc(h.Summaries, func(s org.OperationSummary) bool { return s.RecordID == removed.ID })
		if removed.ID == requestedID {
			// A single deletion identifies the original snapshot in the receipt. Return
			// to the surviving fact covering that snapshot's day in this exact source.
			h.Target = removed
			index := dated.IndexAt(h.Records, removed.Date, func(r *dated.Record[T]) string { return r.Date })
			h.Selected = h.Records[index].ID
		}
	}
	return h
}
