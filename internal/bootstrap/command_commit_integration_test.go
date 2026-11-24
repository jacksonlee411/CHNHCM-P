//go:build integration

package bootstrap_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
)

// Interrupt only after the application callback has finished, including any
// rejection INSERT. The real store still owns COMMIT, rollback and cleanup.
type interruptedCommitStore struct {
	application.Store
	deadline     bool
	beforeCommit func(context.Context, application.Unit) error
}

func TestUncommittedResultsNeverEscapeApplication(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	endpoint, input, _ := prepareCommitCommand(b, "RESULT_BOUNDARY", "CORRECT")
	id := strings.Split(endpoint, "/")[4]
	d := detail(b, id)
	r := d["timeline"].([]any)[0].(map[string]any)
	u, err := url.Parse(f.server.URL)
	must(t, err)
	creds := access.Credentials{Token: b.client.Jar.Cookies(u)[0].Value, CSRF: b.csrf, Write: true}
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	for _, outcome := range []string{"accepted", "no_change", "rejected", "not_applied"} {
		t.Run(outcome, func(t *testing.T) {
			key := foundation.NewID()
			svc.Store = interruptedCommitStore{Store: original, beforeCommit: func(ctx context.Context, u application.Unit) error {
				saved, err := u.FindResult(ctx, key)
				if err != nil {
					return err
				}
				if saved == nil || saved.Result.Outcome != outcome {
					return fmt.Errorf("expected provisional %s before cancellation", outcome)
				}
				return nil
			}}
			before := businessSnapshot(t, f)
			var result org.Result
			var err error
			if outcome == "not_applied" {
				result, err = svc.Resolve(f.ctx, creds, key)
			} else {
				name := "更正"
				if outcome == "no_change" {
					name = r["name"].(string)
				} else if outcome == "rejected" {
					name = ""
				}
				result, err = svc.Maintain(f.ctx, creds, org.MaintainCommand{Action: "CORRECT", BUID: id, RecordID: r["recordId"].(string), SubmissionID: key, Digest: []byte(key), Fields: []string{"name"}, Correct: org.CorrectInput{ExpectedSourceVersion: input["expectedSourceVersion"].(string), ExpectedRecordRevision: r["revision"].(string), Changes: org.ChangesInput{"name": &name}}})
			}
			if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(result, org.Result{}) {
				t.Fatalf("transaction failure exposed provisional %s result; error=%v outcome=%s", outcome, err, result.Outcome)
			}
			if !reflect.DeepEqual(before, businessSnapshot(t, f)) {
				t.Fatal("cancelled outcome changed persistent business state")
			}
		})
	}
}

func (s interruptedCommitStore) WithinWrite(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	var inner context.Context
	var cancel context.CancelFunc
	if s.deadline {
		inner, cancel = context.WithTimeout(ctx, time.Second)
	} else {
		inner, cancel = context.WithCancel(ctx)
	}
	defer cancel()
	return s.Store.WithinWrite(inner, c, func(u application.Unit) error {
		if err := fn(u); err != nil {
			return err
		}
		if s.beforeCommit != nil {
			if err := s.beforeCommit(inner, u); err != nil {
				return err
			}
		}
		if s.deadline {
			<-inner.Done()
		} else {
			cancel()
		}
		return nil
	})
}

func prepareCommitCommand(b *browser, code, action string) (string, map[string]any, int) {
	if strings.HasPrefix(action, "REDATE_") {
		kind := strings.TrimPrefix(action, "REDATE_")
		if kind == "CREATE" {
			id := b.call("POST", "/api/v1/business-units", createBody(code, "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
			d := detail(b, id)
			r := d["timeline"].([]any)[0].(map[string]any)
			return "/api/v1/business-units/" + id + "/records/" + r["recordId"].(string) + "/redate", map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "newEffectiveDate": "2026-07-01"}, 200
		}
		path, input, _ := prepareDeleteTarget(b, code, kind)
		input["newEffectiveDate"] = "2026-07-01"
		return strings.TrimSuffix(path, "/delete") + "/redate", input, 200
	}
	if action == "CREATE" {
		return "/api/v1/business-units", createBody(code, "2026-01-01"), 201
	}
	for _, kind := range []string{"CHANGE", "DEACTIVATE", "REACTIVATE"} {
		if action == "DELETE_"+kind {
			path, input, _ := prepareDeleteTarget(b, code, kind)
			return path, input, 200
		}
	}
	id := b.call("POST", "/api/v1/business-units", createBody(code, "2026-01-01"), foundation.NewID(), 201)["buId"].(string)
	path := "/api/v1/business-units/" + id
	d := detail(b, id)
	switch action {
	case "LIFECYCLE_PAIR":
		return path + "/lifecycle-pairs", map[string]any{"expectedSourceVersion": d["headSourceVersion"], "anchorRecordId": d["timeline"].([]any)[0].(map[string]any)["recordId"], "deactivationDate": "2026-03-01", "reactivationDate": "2026-04-01", "deactivationReason": "BUSINESS_CLOSURE", "reactivationReason": "BUSINESS_RESUMPTION"}, 200

	case "ADD":
		return path + "/records", addBody(d, "2026-03-01", map[string]any{"name": "新增"}), 201
	case "CORRECT":
		r := d["timeline"].([]any)[0].(map[string]any)
		return path + "/records/" + r["recordId"].(string) + "/correct", map[string]any{"expectedSourceVersion": d["headSourceVersion"], "expectedRecordRevision": r["revision"], "changes": map[string]any{"name": "更正"}}, 200
	case "DEACTIVATE":
		return path + "/deactivate", map[string]any{"anchorRecordId": lifecycleAnchor(d), "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": "2026-05-01", "operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}, 201
	case "REACTIVATE":
		b.call("POST", path+"/deactivate", map[string]any{"anchorRecordId": lifecycleAnchor(d), "expectedSourceVersion": d["headSourceVersion"], "effectiveDate": "2026-05-01", "operationType": "DEACTIVATE", "operationReason": "BUSINESS_CLOSURE"}, foundation.NewID(), 201)
		return path + "/reactivate", map[string]any{"anchorRecordId": lifecycleAnchor(detail(b, id)), "expectedSourceVersion": detail(b, id)["headSourceVersion"], "effectiveDate": "2026-06-01", "operationType": "REACTIVATE", "operationReason": "BUSINESS_RESUMPTION"}, 201
	}
	b.f.t.Fatal("unsupported test action", action)
	return "", nil, 0
}

func TestRejectedCommandCommitBoundary(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	cases := []struct{ action, interruption string }{}
	for _, action := range []string{"CREATE", "ADD", "CORRECT", "DEACTIVATE", "REACTIVATE", "DELETE_CHANGE", "DELETE_DEACTIVATE", "DELETE_REACTIVATE", "REDATE_CREATE", "REDATE_CHANGE", "REDATE_DEACTIVATE", "REDATE_REACTIVATE", "LIFECYCLE_PAIR"} {
		cases = append(cases, struct{ action, interruption string }{action, "cancel"})
	}
	for _, action := range []string{"CREATE", "DELETE_CHANGE", "REDATE_CHANGE", "LIFECYCLE_PAIR"} {
		for _, mode := range []string{"deadline", "commit_failure"} {
			cases = append(cases, struct{ action, interruption string }{action, mode})
		}
	}
	for i, tc := range cases {
		t.Run(tc.action+"/"+tc.interruption, func(t *testing.T) {
			endpoint, input, success := prepareCommitCommand(b, fmt.Sprintf("COMMIT_%d", i), tc.action)
			before := businessSnapshot(t, f)
			key := foundation.NewID()
			// Simulate a recoverable business conflict after every business write.
			failing := faultStore{original, foundation.Fail(409, "M02_SOURCE_CONFLICT", "受控写后冲突")}
			rejectionObserved := false
			if tc.interruption == "commit_failure" {
				// A real deferred constraint failure occurs only at outer COMMIT,
				// after StoreTerminal has returned successfully.
				f.exec(`CREATE FUNCTION organization.test_rejection_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'controlled deferred failure'; END $$;
CREATE CONSTRAINT TRIGGER test_rejection_commit AFTER INSERT ON organization.bu_terminal DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.outcome='rejected') EXECUTE FUNCTION organization.test_rejection_commit()`)
				svc.Store = failing
			} else {
				svc.Store = interruptedCommitStore{Store: failing, deadline: tc.interruption == "deadline", beforeCommit: func(ctx context.Context, u application.Unit) error {
					saved, err := u.FindResult(ctx, key)
					if err != nil {
						return err
					}
					rejectionObserved = saved != nil && saved.Result.Outcome == "rejected"
					return nil
				}}
			}
			f.runtime.Organization = svc
			status, output, err := b.response("POST", endpoint, input, key)
			svc.Store = original
			f.runtime.Organization = svc
			if tc.interruption == "commit_failure" {
				f.exec(`DROP TRIGGER test_rejection_commit ON organization.bu_terminal; DROP FUNCTION organization.test_rejection_commit()`)
			}
			must(t, err)
			if tc.interruption != "commit_failure" && !rejectionObserved {
				t.Fatal("interruption did not reach the intended post-rejection boundary")
			}
			if status != 503 || output["outcome"] != nil || output["failureDetails"] != nil {
				t.Errorf("uncommitted rejection must remain pending: HTTP=%d outcome=%v code=%v", status, output["outcome"], output["code"])
			}
			for table, value := range businessSnapshot(t, f) {
				if before[table] != value {
					t.Fatalf("outer rollback changed %s", table)
				}
			}
			if i%2 == 0 {
				result := b.call("POST", endpoint, input, key, success)
				replay := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
				if result["outcome"] != "accepted" || replay["sourceVersion"] != result["sourceVersion"] {
					t.Fatal("original retry did not produce the only accepted result")
				}
			} else {
				result := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
				if result["outcome"] != "not_applied" {
					t.Fatal("resolve did not close the rolled-back identity")
				}
				late := b.call("POST", endpoint, input, key, 409)
				if late["code"] != "M02_REQUEST_CLOSED" {
					t.Fatal("late command reopened the closed identity")
				}
			}
			if f.scalar("SELECT (SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid)+(SELECT count(*) FROM organization.bu_terminal WHERE submission_id=$1)", key) != 1 {
				t.Fatal("recovery did not leave exactly one terminal")
			}
		})
	}
}
