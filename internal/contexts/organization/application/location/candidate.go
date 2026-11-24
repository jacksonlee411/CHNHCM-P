package location

import (
	geo "chnhcm.local/chnhcm/internal/contexts/organization/application/geoarea"
	"chnhcm.local/chnhcm/internal/contexts/organization/domain/dated"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/location"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"reflect"
	"slices"
)

func fault(code, field, message string) error {
	status := 409
	if code == "M02_INVALID_INPUT" || code == "M02_FORBIDDEN_INPUT" {
		status = 400
	}
	return &foundation.Fault{Status: status, Code: code, Field: field, Message: message}
}
func localCandidate(ctx context.Context, cmd org.LocationCommand, t d.Timeline, enterpriseStart string, kernel dated.Kernel[d.Data, d.Patch]) (dated.WorkingCandidate[d.Data, d.Patch], []d.Item, error) {
	defer foundation.BeginStage(ctx, "location.candidate")()
	var work dated.WorkingCandidate[d.Data, d.Patch]
	if cmd.Action == "CREATE" {
		w, e := kernel.PrepareCreate(ctx, t.First, cmd.Create.EffectiveDate, enterpriseStart, cmd.Create.OperationReason, d.Data{Name: cmd.Create.Name, NameEn: cmd.Create.NameEn, CountryID: cmd.Create.CountryRegionID, AreaID: cmd.Create.GeoAreaID, AddressLine: cmd.Create.AddressLine, PostalCode: cmd.Create.PostalCode})
		return w, nil, e
	}
	item := d.Item{ID: "candidate", RecordID: cmd.RecordID}
	patch := func(v org.LocationChangesInput) *d.Patch {
		if v == nil {
			return nil
		}
		p := d.Patch(v)
		return &p
	}
	var err error
	switch cmd.Action {
	case "ADD", "DEACTIVATE", "REACTIVATE":
		item.Action = "ADD"
		item.RecordID = foundation.NewID()
		item.Date = cmd.Add.EffectiveDate
		item.Type = cmd.Add.OperationType
		item.Reason = cmd.Add.OperationReason
		item.Changes = patch(cmd.Add.Changes)
		if cmd.Action == "ADD" {
			// Missing CHANGE fields mean copied defaults; the kernel still validates
			// the position and lifecycle before eliminating an unchanged addition.
			if item.Changes == nil {
				empty := d.Patch{}
				item.Changes = &empty
			}
			err = kernel.ValidateAdditionPosition(t.Records, cmd.Add.AnchorRecordID, enterpriseStart, item.Date)
		} else {
			err = kernel.ValidateLifecycleAppend(t.Records, cmd.Add.AnchorRecordID, cmd.Action, enterpriseStart, item.Date)
		}
	case "CORRECT":
		item.Action = "UPDATE"
		item.ExpectedRevision = cmd.Correct.ExpectedRecordRevision
		item.Date = cmd.Correct.NewEffectiveDate
		item.Changes = patch(cmd.Correct.Changes)
		if cmd.Correct.Classification != nil {
			item.Type = cmd.Correct.Classification.OperationType
			item.Reason = cmd.Correct.Classification.OperationReason
		}
		if item.Date != "" {
			err = kernel.ValidateCorrectionDate(t.Records, item.RecordID, enterpriseStart, item.Date)
		}
	case "REDATE":
		item.Action = "REDATE"
		item.ExpectedRevision = cmd.Redate.ExpectedRecordRevision
		item.Date = cmd.Redate.NewEffectiveDate
		err = kernel.ValidateCorrectionDate(t.Records, item.RecordID, enterpriseStart, item.Date)
	case "DELETE":
		item.Action = "DELETE"
		item.ExpectedRevision = cmd.Delete.ExpectedRecordRevision
	case "INSERT_LIFECYCLE_PAIR":
		p := cmd.InsertPair
		err = kernel.ValidateLifecyclePair(t.Records, p.AnchorRecordID, enterpriseStart, p.DeactivationDate, p.ReactivationDate)
		if err != nil {
			return work, nil, err
		}
		items := []d.Item{{ID: "deactivation", RecordID: foundation.NewID(), Action: "ADD", Date: p.DeactivationDate, Type: "DEACTIVATE", Reason: p.DeactivationReason}, {ID: "reactivation", RecordID: foundation.NewID(), Action: "ADD", Date: p.ReactivationDate, Type: "REACTIVATE", Reason: p.ReactivationReason}}
		w, e := kernel.Prepare(ctx, t.Records, t.First, enterpriseStart, items)
		return w, items, e
	case "MAINTAIN_LIFECYCLE_PAIR":
		p := cmd.MaintainPair
		stop, resume, e := kernel.LifecyclePair(t.Records, p.DeactivationRecordID)
		if e != nil {
			return work, nil, e
		}
		if stop.ID != p.DeactivationRecordID || resume.ID != p.ReactivationRecordID || stop.Revision != p.DeactivationRevision || resume.Revision != p.ReactivationRevision {
			return work, nil, fault("M02_REVISION_CONFLICT", "", "启停记录已更新")
		}
		for _, v := range []struct{ id, date string }{{stop.ID, p.DeactivationDate}, {resume.ID, p.ReactivationDate}} {
			if e = kernel.ValidateCorrectionDate(t.Records, v.id, enterpriseStart, v.date); e != nil {
				return work, nil, e
			}
		}
		items := []d.Item{{ID: "deactivation", RecordID: stop.ID, ExpectedRevision: stop.Revision, Action: "UPDATE", Date: p.DeactivationDate, Type: "DEACTIVATE", Reason: p.DeactivationReason}, {ID: "reactivation", RecordID: resume.ID, ExpectedRevision: resume.Revision, Action: "UPDATE", Date: p.ReactivationDate, Type: "REACTIVATE", Reason: p.ReactivationReason, Changes: patch(p.Changes)}}
		w, e := kernel.Prepare(ctx, t.Records, t.First, enterpriseStart, items)
		return w, items, e
	case "DELETE_LIFECYCLE_PAIR":
		p := cmd.DeletePair
		stop, resume, e := kernel.LifecyclePair(t.Records, p.DeactivationRecordID)
		if e != nil {
			return work, nil, e
		}
		if stop.ID != p.DeactivationRecordID || resume.ID != p.ReactivationRecordID || stop.Revision != p.DeactivationRevision || resume.Revision != p.ReactivationRevision {
			return work, nil, fault("M02_REVISION_CONFLICT", "", "停用和重新启用记录已更新，请核对后再操作")
		}
		items := []d.Item{{ID: "deactivation", RecordID: stop.ID, ExpectedRevision: stop.Revision, Action: "DELETE"}, {ID: "reactivation", RecordID: resume.ID, ExpectedRevision: resume.Revision, Action: "DELETE"}}
		w, e := kernel.Prepare(ctx, t.Records, t.First, enterpriseStart, items)
		return w, items, e
	}
	if err != nil {
		return work, nil, err
	}
	items := []d.Item{item}
	work, err = kernel.Prepare(ctx, t.Records, t.First, enterpriseStart, items)
	return work, items, err
}

func candidate(ctx context.Context, u Unit, t d.Timeline, cmd org.LocationCommand) (d.Candidate, error) {
	work, _, e := localCandidate(ctx, cmd, t, u.EnterpriseScope().ValidFrom, d.Kernel)
	if e != nil {
		return d.Candidate{}, e
	}
	if e = authorizeDataChanges(ctx, u, t.Records, work.Records, cmd); e != nil {
		return d.Candidate{}, e
	}
	requests := []geo.ReferenceRequest{}
	indexes := []int{}
	for i := range work.Records {
		r := &work.Records[i]
		var prev *d.Record
		if i > 0 {
			prev = &work.Records[i-1]
		}
		if d.NeedsAdoption(t.Records, *r, prev) {
			area := ""
			if r.Data.AreaID != nil {
				area = *r.Data.AreaID
			}
			requests = append(requests, geo.ReferenceRequest{CountryID: r.Data.CountryID, AreaID: area, Date: r.Date})
			indexes = append(indexes, i)
		}
	}
	adopted, e := geo.ResolveAddresses(ctx, u.References(), requests)
	if e != nil {
		return d.Candidate{}, e
	}
	for j, i := range indexes {
		work.Records[i].Data.Adoption = adopted[j]
	}
	// Copy the final predecessor evidence for a new unchanged CHANGE after all
	// earlier new references have been captured. Existing successors stay intact.
	for i := range work.Records {
		r := &work.Records[i]
		exists := slices.ContainsFunc(t.Records, func(old d.Record) bool { return old.ID == r.ID })
		if i > 0 && (r.Type == "DEACTIVATE" || !exists && r.Type == "CHANGE" && !slices.Contains(indexes, i)) {
			r.Data.Adoption = (d.Policy{}).Clone(work.Records[i-1].Data).Adoption
		}
	}
	c, e := d.Kernel.Finalize(ctx, work)
	if e != nil {
		return c, e
	}
	for _, r := range c.Records {
		if e = d.ValidateAdoption(r.Data); e != nil {
			return c, e
		}
	}
	if e = authorizeDataChanges(ctx, u, t.Records, c.Records, cmd); e != nil {
		return c, e
	}

	return c, nil
}

func authorizeDataChanges(ctx context.Context, u Unit, before, after []d.Record, cmd org.LocationCommand) error {
	implicit := []string{}
	for _, r := range after {
		for _, old := range before {
			if old.ID != r.ID {
				continue
			}
			oldValues := []any{old.Data.Name, old.Data.NameEn, old.Data.CountryID, old.Data.AreaID, old.Data.AddressLine, old.Data.PostalCode}
			newValues := []any{r.Data.Name, r.Data.NameEn, r.Data.CountryID, r.Data.AreaID, r.Data.AddressLine, r.Data.PostalCode}
			for i, f := range dataFields {
				if !reflect.DeepEqual(oldValues[i], newValues[i]) {
					implicit = append(implicit, f)
				}
			}
		}
	}
	if len(implicit) > 0 {
		slices.Sort(implicit)
		_, e := require(ctx, u, action(cmd), slices.Compact(implicit))
		return e
	}
	return nil
}
