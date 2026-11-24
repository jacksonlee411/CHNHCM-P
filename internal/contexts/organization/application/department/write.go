package department

import (
	"context"
	"fmt"
	"slices"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func metadata(items []d.Item, cmd org.DepartmentCommand, before d.Timeline) []org.RequestItem {
	if cmd.Action != "INSERT_LIFECYCLE_PAIR" && cmd.Action != "DELETE_LIFECYCLE_PAIR" {
		return nil
	}
	out := []org.RequestItem{}
	for _, it := range items {
		typ := it.Type
		for _, r := range before.Records {
			if r.ID == it.RecordID {
				typ = r.Type
			}
		}
		facets := []string{it.Action}
		out = append(out, org.RequestItem{ItemID: it.ID, Action: it.Action, RecordID: it.RecordID, ExpectedRecordRevision: it.ExpectedRevision, EffectiveDate: it.Date, ProcessingFacets: facets, DeclaredFields: []string{}, OriginalType: typ, NewType: it.Type})
	}
	return out
}
func impact(v dated.Impact) *org.TimelineImpact {
	out := &org.TimelineImpact{DataAffectedRanges: []org.DateRange{}, StructuralChanges: []org.Structure{}, AffectedRecordIDs: v.AffectedRecordIDs, PeriodChanges: []org.Period{}}
	for _, r := range v.DataAffectedRanges {
		out.DataAffectedRanges = append(out.DataAffectedRanges, org.DateRange{Start: r.Start, End: r.End})
	}
	for _, s := range v.StructuralChanges {
		out.StructuralChanges = append(out.StructuralChanges, org.Structure{RecordID: s.RecordID, ItemID: s.ItemID, BeforeRevision: s.BeforeRevision, AfterRevision: s.AfterRevision, Facets: s.Facets})
	}
	convert := func(r *dated.Range) *org.DateRange {
		if r == nil {
			return nil
		}
		return &org.DateRange{Start: r.Start, End: r.End}
	}
	for _, p := range v.PeriodChanges {
		out.PeriodChanges = append(out.PeriodChanges, org.Period{RecordID: p.RecordID, ItemID: p.ItemID, Before: convert(p.Before), After: convert(p.After)})
	}
	return out
}
func edgeChange(before, after *d.Edge) EdgeChange {
	out := EdgeChange{}
	if before != nil {
		out.EdgeID = before.ID
		out.BeforeRevisionID = ptr(before.Revision)
		out.BeforeManagementBUID = ptr(before.BU)
		out.BeforeParentDepartmentID = ptr(before.Parent)
		out.BeforeRange = &org.DateRange{Start: before.Start, End: ptr(before.End)}
	}
	if after != nil {
		out.EdgeID = after.ID
		out.AfterRevisionID = ptr(after.Revision)
		out.AfterManagementBUID = ptr(after.BU)
		out.AfterParentDepartmentID = ptr(after.Parent)
		out.AfterRange = &org.DateRange{Start: after.Start, End: ptr(after.End)}
	}
	return out
}

// assemble runs only after final authorization. It binds real identifiers and
// freezes evidence before a repository can receive a validated Write.
func assemble(ctx context.Context, snapshot d.Snapshot, before d.Timeline, candidate d.Candidate, plan d.RelationPlan, items []d.Item, cmd org.DepartmentCommand, terminal Terminal, operator access.OperatorSnapshot, enterpriseName string) (Write, error) {
	defer foundation.BeginStage(ctx, "department.assemble")()
	w := Write{Before: before, After: before, Candidate: candidate, RelationChanged: plan.Changed, Operator: operator, Terminal: terminal, Changes: []Change{}, EdgeChanges: []EdgeChange{}}
	if operator.Login == nil || *operator.Login == "" || operator.IdentityEvidence != "captured" {
		return w, fmt.Errorf("department operator snapshot incomplete")
	}
	w.After.Source = foundation.NewID()
	w.After.SourceNo++
	w.After.RecordedAt = terminal.EvaluatedAt.UTC().Format(time.RFC3339Nano)
	w.After.Records = d.Kernel.CloneRecords(candidate.Records)
	w.After.Edges = slices.Clone(plan.Edges)
	planned := map[string]d.MemberRef{}
	if plan.Changed {
		w.After.RelationSource = foundation.NewID()
		w.After.RelationNo++
		for i := range w.After.Edges {
			edge := &w.After.Edges[i]
			key := edge.Ref.MemberID
			changed := edge.Ref.Source == "planned"
			if changed {
				var previous *d.Edge
				for _, e := range before.Edges {
					if edge.ID != "" && e.ID == edge.ID {
						v := e
						previous = &v
					}
				}
				if edge.ID == "" {
					edge.ID = foundation.NewID()
				}
				edge.Revision = foundation.NewID()
				edge.RevisionNo++
				edge.Source = w.After.RelationSource
				w.EdgeChanges = append(w.EdgeChanges, edgeChange(previous, edge))
				w.NewEdges = append(w.NewEdges, *edge)
			}
			edge.Ref = d.MemberRef{DepartmentID: before.ID, Source: w.After.RelationSource, MemberID: foundation.NewID(), BU: edge.BU, Start: edge.Start, End: edge.End, Parent: edge.Parent, EdgeID: edge.ID, Revision: edge.Revision}
			if changed {
				planned[key] = edge.Ref
			}
		}
		for _, old := range plan.Deleted {
			v := old
			v.Revision = foundation.NewID()
			v.RevisionNo++
			v.Source = w.After.RelationSource
			w.DeletedEdges = append(w.DeletedEdges, v)
			w.EdgeChanges = append(w.EdgeChanges, edgeChange(&old, nil))
		}
	}
	changed := map[string]bool{}
	for _, id := range candidate.Impact.AffectedRecordIDs {
		changed[id] = true
	}
	for i := range w.After.Records {
		r := &w.After.Records[i]
		if r.Data.Member.Source == "planned" {
			ref, ok := planned[r.Data.Member.MemberID]
			if !ok {
				return w, fmt.Errorf("missing final relation member")
			}
			r.Data.Member = ref
		}
		if changed[r.ID] {
			r.Revision = foundation.NewID()
			r.RevisionNo++
		}
		if r.Type == "DEACTIVATE" {
			if i == 0 {
				return w, fmt.Errorf("inactive first record")
			}
			r.Data = (d.Policy{}).Clone(w.After.Records[i-1].Data)
			r.InheritedRecord = w.After.Records[i-1].ID
			r.InheritedRevision = w.After.Records[i-1].Revision
		}
		if changed[r.ID] {
			w.NewRecords = append(w.NewRecords, *r)
		}
	}
	for _, r := range candidate.Deleted {
		r.Revision = foundation.NewID()
		r.RevisionNo++
		w.DeletedRecords = append(w.DeletedRecords, r)
	}
	w.Candidate.Records = w.After.Records
	w.Candidate.Impact = d.Kernel.Compare(ctx, before.Records, w.After.Records, items)
	for i := range w.Candidate.Impact.StructuralChanges {
		c := &w.Candidate.Impact.StructuralChanges[i]
		for _, r := range append(slices.Clone(w.NewRecords), w.DeletedRecords...) {
			if c.RecordID == r.ID {
				c.AfterRevision = ptr(r.Revision)
			}
		}
	}
	for _, r := range append(slices.Clone(w.NewRecords), w.DeletedRecords...) {
		ch := Change{RecordID: r.ID, AfterRevision: r.Revision, Role: "derived", Action: "UPDATE", Facets: []string{}}
		for _, old := range before.Records {
			if old.ID == r.ID {
				ch.BeforeRevision = old.Revision
			}
		}
		if ch.BeforeRevision == "" {
			ch.Action = "ADD"
			ch.Facets = []string{"ADD"}
		}
		if slices.ContainsFunc(w.DeletedRecords, func(v d.Record) bool { return v.ID == r.ID }) {
			ch.Action = "DELETE"
			ch.Facets = []string{"DELETE"}
		}
		if cmd.Action == "CREATE" {
			ch.Role = "direct"
		}
		for _, it := range items {
			if it.RecordID == r.ID {
				ch.Role = "direct"
			}
		}
		if ch.Action == "UPDATE" {
			for _, sc := range w.Candidate.Impact.StructuralChanges {
				if sc.RecordID == r.ID {
					ch.Facets = append(ch.Facets, sc.Facets...)
				}
			}
			for _, old := range before.Records {
				if old.ID == r.ID && !(d.Policy{}).EqualData(old.Data, r.Data) {
					ch.Facets = append(ch.Facets, "CORRECT")
				}
			}
			if len(ch.Facets) == 0 {
				ch.Facets = []string{"CORRECT"}
			}
			slices.Sort(ch.Facets)
			ch.Facets = slices.Compact(ch.Facets)
		}
		w.Changes = append(w.Changes, ch)
	}
	result := org.DepartmentResult{SubmissionID: terminal.ID, Outcome: "accepted", DepartmentID: ptr(before.ID), SourceVersion: ptr(w.After.Source), Action: terminal.Action, ExpectedSourceVersion: ptr(terminal.ExpectedSource), RecordedAt: ptr(w.After.RecordedAt), Impact: impact(w.Candidate.Impact)}
	setEvaluation(&result, *terminal.EvaluatedAt)
	result.CurrentStatus = d.Status(w.After, *result.EvaluatedBusinessDate)
	targetID := cmd.RecordID
	if cmd.Action == "CREATE" {
		targetID = before.First
	}
	if len(items) == 1 && items[0].Action == "ADD" {
		targetID = items[0].RecordID
	}
	if terminal.Action != "REVISE_TIMELINE" {
		result.TargetRecordID = ptr(targetID)
		result.SelectedRecordID = ptr(targetID)
		for _, r := range append(slices.Clone(w.After.Records), w.DeletedRecords...) {
			if r.ID == targetID {
				result.TargetRevision = ptr(r.Revision)
				result.EffectiveDate = ptr(r.Date)
			}
		}
		for _, r := range before.Records {
			if r.ID == targetID {
				result.PreviousEffectiveDate = ptr(r.Date)
				if cmd.Action == "DELETE" {
					selected := d.At(w.After.Records, r.Date)
					if selected != nil {
						result.SelectedRecordID = ptr(selected.ID)
					}
				}
			}
		}
	} else {
		result.Items = []org.DepartmentItemResult{}
		for _, it := range items {
			item := org.DepartmentItemResult{ItemID: it.ID, Outcome: "accepted", RecordID: it.RecordID, BeforeRevisionID: ptr(it.ExpectedRevision), ProcessingFacets: []string{it.Action}}
			for _, r := range append(slices.Clone(w.After.Records), w.DeletedRecords...) {
				if r.ID == it.RecordID {
					item.AfterRevisionID = ptr(r.Revision)
					item.EffectiveDate = r.Date
				}
			}
			result.Items = append(result.Items, item)
		}
	}
	w.Terminal.Outcome = "accepted"
	w.Terminal.Result = result
	s := snapshot.With(w.After)
	var err error
	w.Evidence, err = d.Evidence(ctx, s, w.After, enterpriseName)
	if err != nil {
		return w, err
	}
	w.authorized = true
	return w, nil
}
