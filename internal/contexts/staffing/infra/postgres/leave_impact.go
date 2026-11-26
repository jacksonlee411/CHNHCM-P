package postgres

import (
	"context"
	"time"

	h "chnhcm.local/chnhcm/internal/contexts/staffing/application/hireprep"
	f "chnhcm.local/chnhcm/internal/foundation"
	"chnhcm.local/chnhcm/internal/platform/dbtx"
)

func (st *Store) LeaveImpact(ctx context.Context, credentials f.Credentials, id, date, meaning string) (out h.LeaveImpact, err error) {
	exit, err := (h.ExitCommand{Date: date, Meaning: meaning}).Boundary()
	if err != nil {
		return out, err
	}
	boundary, _ := time.Parse(time.DateOnly, exit.FirstInactive)
	last := boundary.AddDate(0, 0, -1).Format(time.DateOnly)
	err = st.within(ctx, credentials, false, true, func(u *unit) error {
		a, e := u.assignmentAt(ctx, id, last, "")
		if e != nil {
			return e
		}
		if a.Cancelled || !a.Includes(last) {
			return unknownAssignment()
		}
		if e = u.requireEmployment(ctx, "employment.leave", a.Person, a.Employer, a.Department); e != nil {
			return e
		}
		people, _, e := st.HirePeople.RecognizedPeople(dbtx.With(ctx, u.tx), credentials, []string{a.Person}, "")
		if e != nil {
			return e
		}
		if len(people) != 1 {
			return unknownAssignment()
		}
		a.BusinessDate = last
		relation, e := u.relationshipSummary(ctx, a, a, false)
		if e != nil {
			return e
		}
		out.RelationSummary = *relation
		out.Terms, e = u.relationshipTerms(ctx, a, true)
		if e != nil {
			return e
		}
		read, e := u.employmentAllowed(ctx, "contract.read", a.Person, a.Employer, a.Department, false)
		if e != nil || !read {
			return e
		}
		contracts, e := u.contractRows(ctx, a.Employment)
		if e != nil {
			return e
		}
		for _, contract := range contracts {
			if contract.RemainsAt(exit.FirstInactive) {
				out.Contracts = append(out.Contracts, h.LeaveContract{ContractRef: contract.ContractRef, Number: contract.Number})
			}
		}
		if len(out.Contracts) > 0 {
			out.CanManageContracts, e = u.employmentAllowed(ctx, "contract.manage", a.Person, a.Employer, a.Department, false)
		}
		return e
	})
	return out, err
}
