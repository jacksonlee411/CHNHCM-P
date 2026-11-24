package submission_test

import (
	"encoding/hex"
	"testing"

	"chnhcm.local/chnhcm/internal/contexts/organization/application/submission"
)

func TestOriginalBUEncodingAndFixedObjectNamespace(t *testing.T) {
	i := submission.Identity{Kind: submission.BusinessUnit, InstanceID: "instance", SubjectID: "subject", SubmissionID: "submission"}
	key, err := i.LockKey()
	// Constants were calculated independently from the frozen JSON/SHA-256 contract.
	if err != nil || key != 8694418162788534342 {
		t.Fatalf("original BU key changed: %d %v", key, err)
	}
	i.Kind = submission.Department
	key, err = i.LockKey()
	if err != nil || key != 6429945203660494181 {
		t.Fatalf("department namespace changed: %d %v", key, err)
	}
	i.Kind = "untrusted-type"
	if _, err = i.LockKey(); err == nil {
		t.Fatal("unknown kind fell back to BU")
	}
	if got := hex.EncodeToString(submission.BindDigest("instance", "subject", []byte("request"))); got != "eb4c3f1b0a634c10034a9b826a4381b44d41f194be03fcef13c9496b38584fa8" {
		t.Fatal("original request binding changed", got)
	}
	seen := map[int64]bool{}
	for _, kind := range []submission.Kind{submission.BusinessUnit, submission.Department} {
		for _, instance := range []string{"instance", "other-instance"} {
			for _, subject := range []string{"subject", "other-subject"} {
				k, e := (submission.Identity{Kind: kind, InstanceID: instance, SubjectID: subject, SubmissionID: "same-id"}).LockKey()
				if e != nil || seen[k] {
					t.Fatal("identity namespaces collided", e)
				}
				seen[k] = true
			}
		}
	}
}
