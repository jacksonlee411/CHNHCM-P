//go:build integration

package bootstrap_test

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"chnhcm.local/chnhcm/internal/contexts/organization/application"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"testing"
)

// The database transaction really commits. Only its acknowledgement to the
// application is replaced, representing a client-side lost COMMIT confirmation.
// This is not a claim that a physical network-disconnection matrix was run.
type jointLostCommitAck struct{ application.Store }

func (s jointLostCommitAck) WithinWrite(ctx context.Context, c access.Credentials, fn func(application.Unit) error) error {
	if e := s.Store.WithinWrite(ctx, c, fn); e != nil {
		return e
	}
	return foundation.Fail(503, "M02_RESULT_PENDING", "受控提交确认丢失")
}
func TestJointCommittedButAcknowledgementLostRecoversAccepted(t *testing.T) {
	f := fresh(t)
	b := f.login("admin", f.password)
	endpoint, input, _ := prepareCommitCommand(b, "COMMIT_ACK_LOST", "LIFECYCLE_PAIR")
	svc := f.runtime.Organization.(application.Service)
	original := svc.Store
	svc.Store = jointLostCommitAck{original}
	f.runtime.Organization = svc
	key := foundation.NewID()
	status, out, e := b.response("POST", endpoint, input, key)
	must(t, e)
	svc.Store = original
	f.runtime.Organization = svc
	if status != 503 || out["code"] != "M02_RESULT_PENDING" || out["outcome"] != nil || out["items"] != nil {
		t.Fatal("unknown COMMIT leaked a confirmed response", status, out)
	}
	if f.scalar("SELECT count(*) FROM organization.bu_history h CROSS JOIN LATERAL unnest(h.submission_refs) r WHERE h.ids[r]=$1::uuid", key) != 1 || f.scalar("SELECT coalesce(sum(cardinality(source_refs)),0) FROM organization.bu_history") != 2 {
		t.Fatal("fixture did not reach a real accepted COMMIT")
	}
	before := businessSnapshot(t, f)
	resolved := b.call("POST", "/api/v1/organization/submissions/"+key+"/resolve", nil, "", 200)
	retry := b.call("POST", endpoint, input, key, 200)
	if resolved["outcome"] != "accepted" || retry["sourceVersion"] != resolved["sourceVersion"] || retry["replayed"] != true {
		t.Fatal("recovery did not return the committed original", resolved, retry)
	}
	for table, value := range businessSnapshot(t, f) {
		if before[table] != value {
			t.Fatal("lookup after lost acknowledgement wrote another fact", table)
		}
	}
	t.Log("real PostgreSQL COMMIT accepted; injected client acknowledgement loss produced HTTP503/pending; exact retry and resolve recovered the sole accepted terminal")
}
