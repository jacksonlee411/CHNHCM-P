//go:build integration

package bootstrap_test

import (
	enterpriseapp "chnhcm.local/chnhcm/internal/contexts/organization/application/enterprise"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	app "chnhcm.local/chnhcm/internal/contexts/staffing/application"
	store "chnhcm.local/chnhcm/internal/contexts/staffing/infra/postgres"
	staff "chnhcm.local/chnhcm/internal/contexts/staffing/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"fmt"
	"strings"
	"testing"
)

func TestStaffing310GradeWriteRaces(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	svc := f.runtime.Staffing.(app.Service)
	credentials := geoCredentials203(t, f, "admin", f.password)
	family := h.create("jobfamily", "RACE310-F", "")
	for _, kind := range []staff.Kind{staff.Job, staff.Ladder} {
		for _, gradeFirst := range []bool{false, true} {
			for _, existing := range []bool{false, true} {
				tag := fmt.Sprintf("%s-%v-%v", kind, gradeFirst, existing)
				t.Run(tag, func(t *testing.T) {
					grade := h.create("grade", tag, "")
					ids := []string{strings.ToUpper(grade["id"].(string))}
					cmd := staff.Command{Kind: kind, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: tag, Name: name307("并发成员"), EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: "NEW_GRADE_LADDER", GradeIDs: &ids}}
					if kind == staff.Job {
						cmd.Input.JobFunctionID = name307(h.b.functionForJobs("1900-01-01"))
						cmd.Input.OperationReason = "NEW_JOB"
						cmd.Input.JobFamilyID = name307(family["id"].(string))
					}
					if existing {
						initial := []string{}
						if kind == staff.Ladder {
							other := h.create("grade", "OTHER-"+tag, "")
							initial = []string{other["id"].(string)}
						}
						v := create310(h, string(kind), tag, initial, family["id"].(string))
						r := rec307(v)
						cmd.ObjectID = v["id"].(string)
						cmd.Input = staff.Input{Action: "CORRECT", ExpectedSourceVersion: v["headSourceVersion"].(string), RecordID: r["recordId"].(string), ExpectedRecordRevision: r["revision"].(string), ChangedGradeIDs: &ids}
					}
					stop := staff.Command{Kind: staff.Grade, ObjectID: grade["id"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "DEACTIVATE", ExpectedSourceVersion: grade["headSourceVersion"].(string), AnchorRecordID: rec307(grade)["recordId"].(string), EffectiveDate: "2030-01-01", OperationType: "DEACTIVATE", OperationReason: "GRADE_CLOSURE"}}
					first, second := cmd, stop
					if gradeFirst {
						first, second = stop, cmd
					}
					gate := &gate307{Store: svc.Store, locked: make(chan struct{}), release: make(chan struct{})}
					type answer struct {
						r staff.Result
						e error
					}
					one, two := make(chan answer, 1), make(chan answer, 1)
					go func() { r, e := (app.Service{Store: gate}).Execute(f.ctx, credentials, first); one <- answer{r, e} }()
					wait307(t, f, gate.locked)
					go func() { r, e := svc.Execute(f.ctx, credentials, second); two <- answer{r, e} }()
					blocked307(t, f, gate.release)
					close(gate.release)
					a, b := <-one, <-two
					must(t, a.e)
					must(t, b.e)
					if a.r.Outcome != "accepted" || b.r.Outcome != "rejected" || b.r.Failure.Code != "M03_REFERENCE_CONFLICT" {
						t.Fatalf("illegal serialization: %+v / %+v", a.r, b.r)
					}
				})
			}
		}
	}
	// A Ladder's prerequisite Grade already protects the enterprise floor. Both
	// orders must still wait on the same physical enterprise guard; the legitimate
	// result in either order is a rejected later floor and an accepted Ladder.
	for _, first := range []string{"ladder", "enterprise"} {
		t.Run("enterprise-"+first, func(t *testing.T) {
			es := f.runtime.Enterprise.(enterpriseapp.Service)
			ent := h.b.call("GET", "/api/v1/enterprise/context", nil, "", 200)
			eid := ent["id"].(string)
			detail := h.b.call("GET", "/api/v1/enterprise/"+eid, nil, "", 200)
			r := rec307(detail)
			parent := org.EnterpriseCommand{Action: "CORRECT", EnterpriseID: eid, RecordID: r["recordId"].(string), SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Correct: org.EnterpriseCorrectInput{ExpectedSourceVersion: detail["headSourceVersion"].(string), ExpectedRecordRevision: r["revision"].(string), NewEffectiveDate: "2001-01-01"}}
			grade := h.create("grade", "ENT310-"+first, "")
			ids := []string{grade["id"].(string)}
			child := staff.Command{Kind: staff.Ladder, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: "ENT310-" + first, Name: name307("企业守卫阶梯"), EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: "NEW_GRADE_LADDER", GradeIDs: &ids}}
			release, locked := make(chan struct{}), make(chan struct{})
			childResult := make(chan staff.Result, 1)
			childErr, parentErr := make(chan error, 1), make(chan error, 1)
			runChild := func(s app.Service) { v, e := s.Execute(f.ctx, credentials, child); childResult <- v; childErr <- e }
			runParent := func(s enterpriseapp.Service) { _, e := s.Maintain(f.ctx, credentials, parent); parentErr <- e }
			if first == "ladder" {
				go runChild(app.Service{Store: &gate307{Store: svc.Store, locked: locked, release: release}})
				wait307(t, f, locked)
				go runParent(es)
			} else {
				go runParent(enterpriseapp.Service{Store: &enterpriseGate307{Store: es.Store, locked: locked, release: release}})
				wait307(t, f, locked)
				go runChild(svc)
			}
			blocked307(t, f, release)
			close(release)
			ce, pe, result := <-childErr, <-parentErr, <-childResult
			must(t, ce)
			if pe == nil || result.Outcome != "accepted" {
				t.Fatalf("enterprise/Ladder serialization: %v / %+v", pe, result)
			}
		})
	}
}

func TestStaffing310MemberWritesAtomic(t *testing.T) {
	h := staffingSetup307(t)
	f := h.f
	svc := f.runtime.Staffing.(app.Service)
	credentials := geoCredentials203(t, f, "admin", f.password)
	grade := h.create("grade", "ATOMIC310-G", "")
	family := h.create("jobfamily", "ATOMIC310-F", "")
	ids := []string{grade["id"].(string)}
	for _, k := range []staff.Kind{staff.Job, staff.Ladder} {
		for _, after := range []bool{false, true} {
			cmd := staff.Command{Kind: k, SubmissionID: foundation.NewID(), Digest: make([]byte, 32), Input: staff.Input{Action: "CREATE", Code: fmt.Sprintf("ATOMIC-%s-%v", k, after), Name: name307("原子集合"), EffectiveDate: "2000-01-01", OperationType: "CREATE", OperationReason: "NEW_GRADE_LADDER", GradeIDs: &ids}}
			if k == staff.Job {
				cmd.Input.JobFunctionID = name307(h.b.functionForJobs("1900-01-01"))
				cmd.Input.OperationReason = "NEW_JOB"
				cmd.Input.JobFamilyID = name307(family["id"].(string))
			}
			_, e := (app.Service{Store: fault307{svc.Store, after}}).Execute(f.ctx, credentials, cmd)
			if e == nil {
				t.Fatal("fault not injected")
			}
			table := "job"
			if k == staff.Ladder {
				table = "grade_ladder"
			}
			for _, suffix := range []string{"", "_source", "_revision", "_read", "_grade_manifest", "_grade_member", "_outbox"} {
				if f.scalar("SELECT count(*) FROM staffing."+table+suffix) != 0 {
					t.Fatal("partial write in " + suffix)
				}
			}
			if f.scalar("SELECT count(*) FROM staffing."+table+"_terminal WHERE submission_id=$1", cmd.SubmissionID) != 0 {
				t.Fatal("partial terminal")
			}
			r, e := svc.Resolve(f.ctx, credentials, k, cmd.SubmissionID)
			must(t, e)
			if r.Outcome != "not_applied" {
				t.Fatal("false terminal")
			}
			// Later independent case has the previous explicit not-applied receipt only.
			// Inspect business tables above; terminal count is scoped to this attempt.
		}
	}
	db := svc.Store.(*store.Store)
	probe := &floorProbe307{EnterpriseFloor: db.Organization}
	db.Organization = probe
	f.exec(`CREATE TABLE public.ladder310_tx_probe (pid bigint,txid text);
 GRANT INSERT ON public.ladder310_tx_probe TO chnhcm_app;
 CREATE FUNCTION public.ladder310_probe() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO public.ladder310_tx_probe VALUES(pg_backend_pid(),pg_current_xact_id()::text); RETURN NEW; END $$;
 CREATE TRIGGER ladder310_probe AFTER INSERT ON staffing.grade_ladder_source FOR EACH ROW EXECUTE FUNCTION public.ladder310_probe();`)
	create310(h, "gradeladder", "TX310", ids, "")
	var xid string
	var pid uint32
	must(t, f.admin.QueryRow(f.ctx, `SELECT pid,txid FROM public.ladder310_tx_probe`).Scan(&pid, &xid))
	if pid == 0 || pid != probe.pid || xid != probe.txid {
		t.Fatal("Ladder and enterprise port did not share the physical transaction")
	}
}
