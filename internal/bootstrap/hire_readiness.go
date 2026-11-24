package bootstrap

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	f "chnhcm.local/chnhcm/internal/foundation"
	"net/http"
)

func ruleBody[T any](run func(*http.Request, T) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		var input T
		if _, e := decode(r, &input); e != nil {
			writeError(w, e)
			return
		}
		result, e := run(r, input)
		respond(w, 200, result, e)
	}
}
func ruleGet(run func(*http.Request, string) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "sourceVersion")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := run(r, q.Get("sourceVersion"))
		respond(w, 200, v, e)
	}
}
func (a *API) readinessRoutes(mux *http.ServeMux) {
	hires := a.Runtime.Hires
	mux.HandleFunc("POST /api/v1/positions/{id}/replenishment/submissions/{sub}/resolve", ruleBody(func(r *http.Request, _ struct{}) (any, error) {
		return a.Runtime.Replenishments.ResolveReplenishment(r.Context(), credentials(r), r.PathValue("id"), r.PathValue("sub"))
	}))
	mux.HandleFunc("GET /api/v1/positions/{id}/replenishment", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "businessAt")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := a.Runtime.Replenishments.Replenishment(r.Context(), credentials(r), r.PathValue("id"), q.Get("businessAt"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/positions/{id}/replenishment", ruleBody(func(r *http.Request, in staff.ReplenishmentChange) (any, error) {
		return a.Runtime.Replenishments.ChangeReplenishment(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("GET /api/v1/assignments/context", func(w http.ResponseWriter, r *http.Request) {
		if _, e := queryValues(r); e != nil {
			writeError(w, e)
			return
		}
		allowed, e := hires.CanListAssignments(r.Context(), credentials(r))
		respond(w, 200, map[string]bool{"canList": allowed}, e)
	})
	mux.HandleFunc("POST /api/v1/assignments/query", ruleBody(func(r *http.Request, in h.AssignmentQuery) (any, error) {
		return hires.ListAssignments(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("GET /api/v1/employment-actions", a.listEmploymentRegister)
	mux.HandleFunc("GET /api/v1/employment-actions/context", a.employmentRegisterContext)
	mux.HandleFunc("GET /api/v1/employment-actions/{id}", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "adoptedRef")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := hires.OriginalAction(r.Context(), credentials(r), r.PathValue("id"), q.Get("adoptedRef"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("GET /api/v1/assignments/{id}", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "businessAt", "sourceVersion")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := hires.AssignmentInfo(r.Context(), credentials(r), r.PathValue("id"), q.Get("businessAt"), q.Get("sourceVersion"))
		respond(w, 200, v, e)
	})
	mux.HandleFunc("POST /api/v1/assignments/{id}/location-options", ruleBody(func(r *http.Request, in h.LocationSearch) (any, error) {
		return hires.LocationOptions(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	mux.HandleFunc("POST /api/v1/assignments/{id}/transfer-arrangement", ruleBody(func(r *http.Request, in h.TransferArrangementQuery) (any, error) {
		return hires.TransferArrangement(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	mux.HandleFunc("POST /api/v1/assignments/{id}/transfer-options", ruleBody(func(r *http.Request, in h.TransferSearch) (any, error) {
		return hires.TransferOptions(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	mux.HandleFunc("POST /api/v1/assignments/{id}/additional-assignments", ruleBody(func(r *http.Request, in h.Secondary) (any, error) {
		return hires.AddSecondary(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("POST /api/v1/assignments/{id}/additional-options", ruleBody(func(r *http.Request, in h.SecondaryOption) (any, error) {
		return hires.SecondaryOptions(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	for _, op := range []struct {
		path  string
		leave bool
	}{{"end", false}, {"leave", true}} {
		mux.HandleFunc("POST /api/v1/assignments/{id}/"+op.path, ruleBody(func(r *http.Request, in h.ExitCommand) (any, error) {
			return hires.ExitAssignment(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), op.leave, in)
		}))
	}
	mux.HandleFunc("POST /api/v1/assignments/{id}/exchange-options", ruleBody(func(r *http.Request, in h.ExchangeSearch) (any, error) {
		return hires.ExchangeOptions(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	mux.HandleFunc("POST /api/v1/assignments/{id}/position-exchanges", ruleBody(func(r *http.Request, in h.ExchangeCommand) (any, error) {
		return hires.ExchangePositions(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("POST /api/v1/assignments/{id}/managers", ruleBody(func(r *http.Request, in h.ManagerCommand) (any, error) {
		return hires.SetManager(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("POST /api/v1/assignments/{id}/grade-options", ruleBody(func(r *http.Request, in h.GradeSearch) (any, error) {
		return hires.GradeOptions(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	for _, op := range []struct{ path, kind string }{{"grade-records", "grade"}, {"promotions", "promotion"}, {"promotions/{action}/correct", "promotion"}, {"promotion-transfers", "pair"}} {
		mux.HandleFunc("POST /api/v1/assignments/{id}/"+op.path, ruleBody(func(r *http.Request, in h.PromotionCommand) (any, error) {
			return hires.RecordPromotion(r.Context(), credentials(r), r.PathValue("id"), r.PathValue("action"), op.kind, r.Header.Get("Idempotency-Key"), in)
		}))
	}
	mux.HandleFunc("POST /api/v1/assignments/{id}/probation-departments", ruleBody(func(r *http.Request, in h.LocationSearch) (any, error) {
		return hires.ProbationDepartments(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	for _, op := range []struct{ path, kind string }{{"trial-start", "ProbationStart"}, {"confirm-employment", "ConfirmEmployment"}} {
		mux.HandleFunc("POST /api/v1/assignments/{id}/"+op.path, ruleBody(func(r *http.Request, in h.ProbationCommand) (any, error) {
			return hires.RecordProbation(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), op.kind, in)
		}))
	}

	changeLocation := ruleBody(func(r *http.Request, in h.LocationChange) (any, error) {
		return hires.ChangeLocation(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), r.PathValue("id"), r.PathValue("action"), in)
	})
	mux.HandleFunc("POST /api/v1/assignments/{id}/location-changes", changeLocation)
	mux.HandleFunc("POST /api/v1/assignments/{id}/location-changes/{action}/correct", changeLocation)
	for _, item := range []struct{ route, command string }{
		{"POST /api/v1/assignments/{id}/transfers", "transfer"},
		{"POST /api/v1/assignments/{id}/transfers/{action}/correct", "transfer"},
		{"POST /api/v1/assignments/{id}/actions/{action}/redate", "redate"},
		{"POST /api/v1/assignments/{id}/actions/{action}/delete", "delete"},
	} {
		mux.HandleFunc(item.route, ruleBody(func(r *http.Request, in h.RecordMaintenance) (any, error) {
			return hires.MaintainRecord(r.Context(), credentials(r), r.PathValue("id"), r.PathValue("action"), item.command, r.Header.Get("Idempotency-Key"), in)
		}))
	}
	mux.HandleFunc("POST /api/v1/appointment-decisions", ruleBody(func(r *http.Request, in h.DecisionPublication) (any, error) {
		return hires.PublishDecision(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("GET /api/v1/persons/{id}/appointment-decisions", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "rehireAction", "rehireRevision")
		if e != nil {
			writeError(w, e)
			return
		}
		var ref *h.ActionRef
		if q.Has("rehireAction") || q.Has("rehireRevision") {
			ref = &h.ActionRef{Action: q.Get("rehireAction"), Revision: q.Get("rehireRevision")}
		}
		result, e := hires.HireDecisions(r.Context(), credentials(r), r.PathValue("id"), ref)
		respond(w, 200, result, e)
	})
	mux.HandleFunc("POST /api/v1/hires/arrangement", ruleBody(func(r *http.Request, in h.ArrangementQuery) (any, error) {
		return hires.HireArrangement(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("POST /api/v1/hires/targets", ruleBody(func(r *http.Request, in h.TargetSearch) (any, error) {
		return hires.HireTargets(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("POST /api/v1/hires/employers", ruleBody(func(r *http.Request, in h.EmployerSearch) (any, error) {
		return hires.HireEmployers(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("POST /api/v1/hires/preview", ruleBody(func(r *http.Request, in h.Hire) (any, error) {
		return hires.PreviewHire(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("POST /api/v1/hires", ruleBody(func(r *http.Request, in h.Hire) (any, error) {
		return hires.Hire(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("POST /api/v1/employment-actions/{id}/cancel", ruleBody(func(r *http.Request, in h.Cancel) (any, error) {
		return hires.CancelHire(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), r.PathValue("id"), in)
	}))
	resolveEmployment := func(w http.ResponseWriter, r *http.Request) {
		if _, err := queryValues(r); err != nil {
			writeError(w, err)
			return
		}
		v, e := hires.ResolveHire(r.Context(), credentials(r), r.PathValue("sub"))
		respond(w, 200, v, e)
	}
	mux.HandleFunc("POST /api/v1/hires/submissions/{sub}/resolve", resolveEmployment)
	mux.HandleFunc("POST /api/v1/employment-actions/submissions/{sub}/resolve", resolveEmployment)
	mux.HandleFunc("GET /api/v1/persons/{id}/employment", func(w http.ResponseWriter, r *http.Request) {
		q, e := queryValues(r, "businessAt")
		if e != nil {
			writeError(w, e)
			return
		}
		v, e := hires.PersonEmployment(r.Context(), credentials(r), r.PathValue("id"), q.Get("businessAt"))
		respond(w, 200, v, e)
	})
	const purposes = "/api/v1/legal-entities/{id}/purposes"
	const policies = "/api/v1/assignment-policies"
	resolve := func(run func(*http.Request) (f.RuleReceipt, error)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if _, e := queryValues(r); e != nil {
				writeError(w, e)
				return
			}
			v, e := run(r)
			respond(w, 200, v, e)
		}
	}
	mux.HandleFunc("POST /api/v1/legal-entities/purpose-submissions/{sub}/resolve", resolve(func(r *http.Request) (f.RuleReceipt, error) {
		return a.Runtime.LegalPurposes.ResolvePurpose(r.Context(), credentials(r), r.PathValue("sub"))
	}))
	mux.HandleFunc("POST "+policies+"/submissions/{sub}/resolve", resolve(func(r *http.Request) (f.RuleReceipt, error) {
		return a.Runtime.AssignmentPolicies.ResolvePolicy(r.Context(), credentials(r), r.PathValue("sub"))
	}))
	mux.HandleFunc("GET "+purposes, ruleGet(func(r *http.Request, v string) (any, error) {
		return a.Runtime.LegalPurposes.Purposes(r.Context(), credentials(r), r.PathValue("id"), v)
	}))
	mux.HandleFunc("POST "+purposes, ruleBody(func(r *http.Request, in f.RuleChange[org.LegalPurpose]) (any, error) {
		return a.Runtime.LegalPurposes.DeclarePurpose(r.Context(), credentials(r), r.PathValue("id"), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("POST "+purposes+"/check", ruleBody(func(r *http.Request, in org.PurposeCheck) (any, error) {
		return a.Runtime.LegalPurposes.CheckPurpose(r.Context(), credentials(r), r.PathValue("id"), in)
	}))
	mux.HandleFunc("POST /api/v1/legal-entities/purpose-candidates", ruleBody(func(r *http.Request, in org.PurposeCandidates) (any, error) {
		return a.Runtime.LegalPurposes.Candidates(r.Context(), credentials(r), in)
	}))
	mux.HandleFunc("GET "+policies, ruleGet(func(r *http.Request, v string) (any, error) {
		return a.Runtime.AssignmentPolicies.Policies(r.Context(), credentials(r), v)
	}))
	mux.HandleFunc("POST "+policies, ruleBody(func(r *http.Request, in f.RuleChange[staff.AssignmentPolicy]) (any, error) {
		return a.Runtime.AssignmentPolicies.PublishPolicy(r.Context(), credentials(r), r.Header.Get("Idempotency-Key"), in)
	}))
	mux.HandleFunc("POST "+policies+"/check", ruleBody(func(r *http.Request, in staff.AssignmentCheck) (any, error) {
		return a.Runtime.AssignmentPolicies.CheckAssignment(r.Context(), credentials(r), in)
	}))
}
