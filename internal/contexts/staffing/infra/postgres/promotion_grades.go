package postgres

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	d "chnhcm.local/chnhcm/internal/contexts/staffing/domain"
	st "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
)

func gradeUnavailable() error {
	return f.Fail(409, "ASSIGNMENT_GRADE", "职级在拟定职务或期间内不可用")
}
func oneDay(day string) f.Period {
	date, _ := time.Parse(time.DateOnly, day)
	return f.Period{From: day, Until: date.AddDate(0, 0, 1).Format(time.DateOnly)}
}
func (u *unit) gradeJob(ctx context.Context, job, day string, locked bool) (app.PositionSource, error) {
	sources, e := u.PositionSources(ctx, "jobId", []string{job}, locked)
	if e != nil {
		return app.PositionSource{}, e
	}
	if len(sources) != 1 || d.PositionConflict(sources[0].Timeline, day, oneDay(day).Until, "") != "" {
		return app.PositionSource{}, gradeUnavailable()
	}
	visible, e := app.JobGradeVisible(ctx, u, sources[0], day)
	if e != nil {
		return app.PositionSource{}, e
	}
	if !visible {
		return app.PositionSource{}, f.Fail(403, "ASSIGNMENT_GRADE_READ", "无权读取该职务的可用职级")
	}
	return sources[0], nil
}
func (u *unit) gradeForJob(ctx context.Context, job, grade string, term f.Period) (*h.GradeRef, error) {
	j, e := u.gradeJob(ctx, job, term.From, true)
	if e != nil {
		return nil, e
	}
	if d.PositionConflict(j.Timeline, term.From, term.Until, grade) != "" {
		return nil, gradeUnavailable()
	}
	gs, e := u.PositionSources(ctx, "gradeId", []string{grade}, true)
	if e != nil {
		return nil, e
	}
	if len(gs) != 1 {
		return nil, gradeUnavailable()
	}
	g := gs[0]
	name, e := app.HireTargetLabel(ctx, u, "gradeId", g, term)
	if e != nil {
		return nil, e
	}
	if name == "" {
		return nil, gradeUnavailable()
	}
	return &h.GradeRef{ID: g.ID, Name: name, Source: g.Source, Revision: g.Records[d.At(g.Timeline, term.From)].Revision, Job: job, JobSource: j.Source, JobRevision: j.Records[d.At(j.Timeline, term.From)].Revision}, nil
}

// Read the declared position grade from the exact adopted M03 revision.
func (u *unit) requiredPositionGrade(ctx context.Context, seg st.AssignmentSegment) (string, error) {
	if seg.Position == "" {
		return "", nil
	}
	p, err := u.Load(ctx, d.Position, seg.Position, seg.PositionSource)
	if err != nil {
		return "", err
	}
	for _, r := range p.Records {
		if r.Revision == seg.PositionRevision {
			return r.Data["gradeId"], nil
		}
	}
	return "", gradeUnavailable()
}

func (u *unit) checkPlacementGrade(ctx context.Context, seg st.AssignmentSegment, grade string) error {
	required, err := u.requiredPositionGrade(ctx, seg)
	if err == nil && required != "" && required != grade {
		return f.Fail(409, "ASSIGNMENT_GRADE", seg.From+" 起个人任职级与岗位要求不符")
	}
	return err
}

func (st *Store) GradeOptions(ctx context.Context, c f.Credentials, id string, in h.GradeSearch) (out []h.GradeRef, err error) {
	out = []h.GradeRef{}
	if in.Position != "" && !f.ValidID(in.Position) || !f.ValidDate(in.BusinessDate) || in.Job != "" && !f.ValidID(in.Job) || in.Page < 1 || in.Page > 100000 || utf8.RuneCountInString(in.Search) > 2000 {
		return out, f.Fail(400, "GRADE_OPTIONS", "请核对业务日期与职务条件")
	}
	err = st.within(ctx, c, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, in.BusinessDate, "")
		if e != nil {
			return e
		}
		allowed, e := u.employmentAllowed(ctx, "promotion", a.Person, a.Employer, a.Department, false)
		if e != nil {
			return e
		}
		if !allowed {
			if e = u.requireEmployment(ctx, "grade.record", a.Person, a.Employer, a.Department); e != nil {
				return e
			}
		}
		job := a.Job
		if in.Job != "" {
			job = in.Job
		}
		required := ""
		if in.Position != "" {
			permitted, e := u.positionEmploymentScope(ctx, in.Position, oneDay(in.BusinessDate), a.Person, a.Employer, "transfer", "transfer")
			if e != nil {
				return e
			}
			visible, e := u.positionReadable(ctx, u.enterprise.ID)
			if e != nil {
				return e
			}
			if !permitted || !visible {
				return f.Fail(403, "ASSIGNMENT_GRADE_READ", "当前权限无法采用拟任岗位")
			}
			pos, e := u.Load(ctx, d.Position, in.Position, "")
			if e != nil {
				return e
			}
			idx := d.At(pos, in.BusinessDate)
			if idx < 0 || !pos.Records[idx].Active() {
				return gradeUnavailable()
			}
			job, required = pos.Records[idx].Data["jobId"], pos.Records[idx].Data["gradeId"]
		}
		j, e := u.gradeJob(ctx, job, in.BusinessDate, false)
		if e != nil {
			return e
		}
		rec := j.Records[d.At(j.Timeline, in.BusinessDate)]
		if len(rec.Grades) == 0 {
			return nil
		}
		eligible := rec.Grades.IDs()
		if required != "" {
			eligible = []string{required}
		}
		ids, e := u.PositionCandidateIDs(ctx, "gradeId", in.BusinessDate, strings.ToLower(strings.TrimSpace(in.Search)), eligible, (in.Page-1)*50, 50)
		if e != nil {
			return e
		}
		gs, e := u.PositionSources(ctx, "gradeId", ids, false)
		if e != nil {
			return e
		}
		byID := map[string]app.PositionSource{}
		for _, g := range gs {
			byID[g.ID] = g
		}
		for _, id := range ids {
			g := byID[id]
			name, e := app.HireTargetLabel(ctx, u, "gradeId", g, oneDay(in.BusinessDate))
			if e != nil {
				return e
			}
			if name != "" {
				out = append(out, h.GradeRef{ID: g.ID, Name: name, Source: g.Source, Revision: g.Records[d.At(g.Timeline, in.BusinessDate)].Revision, Job: job, JobSource: j.Source, JobRevision: rec.Revision})
			}
		}
		return nil
	})
	return
}
