package department

import (
	"context"
	"strings"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/department"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

func (s Service) Parents(ctx context.Context, c access.Credentials, in org.DepartmentParentQuery) (out org.DepartmentParents, err error) {
	out = org.DepartmentParents{Applicable: true, Items: []org.DepartmentReferenceCandidate{}}
	if err = validateReferenceInput(org.DepartmentReferenceQuery{Context: in.Context, Q: in.Q, SelectedID: in.SelectedID, Page: 1, PageSize: in.PageSize}, "department"); err != nil {
		return out, err
	}
	q := in.Context
	err = s.Store.WithinDepartment(ctx, c, func(u Unit) error {
		ent, e := u.Enterprise(ctx, false)
		if e != nil {
			return e
		}
		if !departmentReadEntry(u, ent.ID) {
			return denied()
		}
		now, e := u.Now(ctx)
		if e != nil {
			return e
		}
		bu, target := q.ManagementBUID, ""
		t := d.Timeline{ID: foundation.NewID(), First: foundation.NewID(), Enterprise: ent.ID}
		if q.Mode == "maintain" {
			one, e := u.Snapshot(ctx, q.DepartmentID, "")
			if e != nil {
				return e
			}
			current := one.Snapshot.Department(q.DepartmentID)
			if current == nil {
				return missing()
			}
			if e = referenceMaintenance(ctx, u, *current, now, q); e != nil {
				return e
			}
			t = *current
			bu, _, _, _, _ = explicitTargets(referenceCommand(q), t)
			target = t.ID
		}
		if !foundation.ValidID(bu) {
			return foundation.Fail(400, "M02_INVALID_INPUT", "请先选择管理业务单元")
		}
		scope, e := u.Access().Decide(ctx, access.Intent{Action: "bu.reference.read", EnterpriseID: ent.ID, BUID: bu, Kind: "reference", Fields: []string{}})
		if e != nil {
			return e
		}
		if !scope.Allowed {
			return missingReference()
		}
		base, variable, applicable, e := prepareParent(ctx, t, q, ent.ValidFrom)
		if e != nil {
			return e
		}
		out.Applicable = applicable
		plan, e := parentPlan(ctx, u, ent.ID, now)
		if e != nil {
			return e
		}
		allowed := true
		windows := d.ParentWindows(base, variable)
		if q.Mode == "create" {
			decision, e := u.Access().Decide(ctx, requirements(org.DepartmentCommand{Action: "CREATE", Fields: CreateFields}, t, fact(t, now, bu)))
			if e != nil {
				return e
			}
			allowed = decision.Allowed
			if !allowed {
				windows = []d.ParentWindow{}
			}
		}
		binding := queryCursor{Kind: "department-parent", Instance: plan.Instance, Enterprise: plan.Enterprise, Subject: plan.Subject, Session: plan.Session, Day: plan.Day, Source: t.Source, Query: fingerprint([]any{strings.ToLower(strings.TrimSpace(in.Q)), q, in.PageSize})}
		request := ParentRequest{Plan: plan, Target: target, Source: t.Source, BU: bu, Selected: in.SelectedID, Search: in.Q, Base: base, Windows: windows}
		if in.Cursor != "" {
			request.AfterCode, request.AfterID, e = binding.decode(in.Cursor)
			if e != nil {
				return e
			}
		}
		var choices *d.ParentChoices
		lastCode, lastID := "", ""
		e = u.WalkParents(ctx, request, func(frame ParentFrame) (bool, error) {
			if frame.Common {
				if _, e := reference(ctx, u, frame.Capture.Labels, ent.ID, bu, "bu", now); e != nil {
					return false, e
				}
				projector := newProjector(ctx, u, frame.Capture.Labels, ent, now)
				if in.SelectedID != "" {
					selected := projector.ref(in.SelectedID, "department")
					if selected.err != nil {
						return false, selected.err
					}
					out.SelectedReference = selected.value
				}
				if q.Mode == "maintain" {
					current := frame.Capture.Facts.Department(target)
					if current == nil {
						return false, missing()
					}
					if e := referenceMaintenance(ctx, u, *current, now, q); e != nil {
						return false, e
					}
				}
				if !applicable || !allowed {
					return false, nil
				}
				var e error
				choices, e = d.NewCapturedParentChoices(ctx, frame.Capture.Facts, base, variable)
				if e != nil {
					return false, safeQualification(ctx, u, frame.Capture.Labels, ent.ID, t.ID, now, e)
				}
				return true, nil
			}
			if choices == nil {
				return false, foundation.Fail(503, "M02_DEPENDENCY_UNAVAILABLE", "选项暂时无法加载")
			}
			if e := choices.AddCaptured(frame.Capture.Facts); e != nil {
				return false, e
			}
			projector := newProjector(ctx, u, frame.Capture.Labels, ent, now)
			ref := projector.ref(frame.ID, "department")
			if ref.err != nil {
				return false, ref.err
			}
			if ref.value.Code == nil && ref.value.Name == nil {
				return true, nil
			}
			done := foundation.BeginStage(ctx, "references.qualify")
			eligible, e := choices.Eligible(ctx, frame.ID)
			done()
			if e != nil {
				return false, e
			}
			if !eligible {
				return true, nil
			}
			if len(out.Items) == in.PageSize {
				out.HasMore = true
				return false, nil
			}
			out.Items = append(out.Items, org.DepartmentReferenceCandidate{ID: frame.ID, Code: ref.value.Code, Name: ref.value.Name, Eligibility: "eligible"})
			lastCode, lastID = frame.Code, frame.ID
			return true, nil
		})
		if e != nil {
			return e
		}
		if out.HasMore {
			out.NextCursor = binding.encode(lastCode, lastID)
		}
		return nil
	})
	return
}
