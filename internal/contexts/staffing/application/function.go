package application

import (
	"context"

	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	s "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func prepareFunctions(ctx context.Context, u Unit, p *d.Plan) error {
	t := p.After
	if t.Kind == d.JobFunction && d.CoverageChanged(p.Before, t) {
		jobs, err := u.Dependents(ctx, d.JobFunction, t.ID)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			if err = d.FunctionCoverage(job, t, t.ID); err != nil {
				return foundation.Fail(409, "M03_REFERENCE_CONFLICT", "现有职务启用期间超出职类范围")
			}
		}
	}
	if t.Kind != d.Job {
		return nil
	}
	functions := map[string]d.Timeline{}
	for _, id := range d.FunctionIDs(t) {
		f, err := u.Load(ctx, d.JobFunction, id, "")
		if err != nil {
			return err
		}
		functions[id] = f
	}
	for _, rid := range p.AdoptFunctions {
		r := &t.Records[d.Index(t, rid)]
		for _, purpose := range []string{"read", "history.read"} {
			if purpose == "history.read" && r.Date == today(u) {
				continue
			}
			dec, err := decide(ctx, u, d.JobFunction, t.Enterprise, purpose, nil)
			if err != nil {
				return err
			}
			if !has(dec.ReadFields, "name", "status", "effectiveDate") {
				return foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "无权采用该职类")
			}
		}
		if functions[r.Data["jobFunctionId"]].ID == "" {
			return foundation.Fail(403, "M03_REFERENCE_FORBIDDEN", "无权采用该职类")
		}
	}
	for id, f := range functions {
		if err := d.FunctionCoverage(t, f, id); err != nil {
			return err
		}
	}
	for _, rid := range p.AdoptFunctions {
		r := &t.Records[d.Index(t, rid)]
		f := functions[r.Data["jobFunctionId"]]
		fr := f.Records[d.At(f, r.Date)]
		r.Function = d.FunctionEvidence{FunctionID: f.ID, BusinessAt: r.Date, Source: f.Source, RecordID: fr.ID, Revision: fr.Revision}
	}
	for i := range t.Records {
		if !t.Records[i].Active() && t.Records[i].Data["jobFunctionId"] != "" {
			t.Records[i].Function = t.Records[i-1].Function
		}
	}
	return nil
}

func projectFunction(ctx context.Context, u Unit, t d.Timeline, r d.Record, v s.View, date string, exact bool) error {
	id := r.Data["jobFunctionId"]
	if id == "" {
		v["jobFunctionName"] = "未登记职类"
		return nil
	}
	delete(v, "jobFunctionId")
	delete(v, "jobFunctionDisplayDate")
	delete(v, "jobFunctionAdoption")
	v["jobFunctionName"] = "受限"
	for _, purpose := range []string{"read", "history.read"} {
		if purpose == "history.read" && !exact && date == today(u) {
			continue
		}
		dec, err := allowed(ctx, u, d.JobFunction, t.Enterprise, purpose)
		if err != nil {
			return err
		}
		if !dec.Allowed || !has(dec.ReadFields, "name", "effectiveDate") {
			return nil
		}
	}
	source := ""
	if exact {
		source = r.Function.Source
	}
	f, err := classificationTimeline(ctx, u, d.JobFunction, id, source)
	if err != nil {
		return err
	}
	i := d.At(f, date)
	if exact {
		i = d.Index(f, r.Function.RecordID)
	}
	if i < 0 || f.Enterprise != t.Enterprise || exact && f.Records[i].Revision != r.Function.Revision {
		return foundation.Fail(503, "M03_DEPENDENCY_UNAVAILABLE", "职类采用依据暂不可读")
	}
	v["jobFunctionId"] = id
	v["jobFunctionName"] = f.Records[i].Data["name"]
	v["jobFunctionDisplayDate"] = date
	if exact {
		v["jobFunctionAdoption"] = s.View{"date": r.Function.BusinessAt, "sourceVersion": source, "recordId": r.Function.RecordID, "revision": r.Function.Revision}
	}
	return nil
}
