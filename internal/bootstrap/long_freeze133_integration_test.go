//go:build integration

package bootstrap_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"chnhcm.local/chnhcm/internal/bootstrap"
)

// Acceptance scenario selected by the user; not a product record limit.
const maintenanceTimelineRecords = 20

// Export only the real application-generated input, before any timed save.
// Repeated comparisons restore this exact input into another owned database.
func freeze133LongInput(t *testing.T, f *fixture, object string) bool {
	return freezeTimelineInput(t, f, object, "bu", os.Getenv("CHNHCM_T07_FREEZE_INPUT"))
}

func freezeTimelineInput(t *testing.T, f *fixture, object, kind, dir string) bool {
	if dir == "" {
		return false
	}
	real, err := filepath.EvalSymlinks(dir)
	must(t, err)
	repo, err := filepath.Abs("../..")
	must(t, err)
	relative, err := filepath.Rel(repo, real)
	must(t, err)
	info, err := os.Stat(real)
	must(t, err)
	if !filepath.IsAbs(dir) || !strings.HasPrefix(relative, ".."+string(filepath.Separator)) || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatal("frozen input requires a private external 0700 directory")
	}
	for _, account := range []struct{ login, previous string }{{"admin", f.password}, {"reader", f.readPassword}} {
		must(t, bootstrap.ChangeLocalPassword(f.ctx, f.admin, bootstrap.LocalPasswordChange{Instance: f.instance, Login: account.login, PreviousPassword: account.previous, Password: "admin1234567"}))
	}
	var source string
	recordCount := maintenanceTimelineRecords
	members := []map[string]any{}
	if kind == "bu" {
		fact := f.buHead(object)
		source = fact.Operation.Source.ID
		if len(fact.State.Timeline.Records) != recordCount {
			t.Fatal("fixture record count differs from requested size")
		}
		for _, index := range []int{0, recordCount / 2, recordCount - 1} {
			r := fact.State.Timeline.Records[index]
			members = append(members, map[string]any{"index": index, "recordId": r.ID, "revision": r.Revision})
		}
	} else {
		fact := f.departmentHead(object)
		source = fact.Operation.Source.ID
		if len(fact.State.Timeline.Records) != recordCount {
			t.Fatal("fixture record count differs from requested size")
		}
		for _, index := range []int{0, recordCount / 2, recordCount - 1} {
			r := fact.State.Timeline.Records[index]
			members = append(members, map[string]any{"index": index, "recordId": r.ID, "revision": r.Revision})
		}
	}
	file, err := os.OpenFile(filepath.Join(real, "input.dump"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(t, err)
	h := sha256.New()
	cmd := exec.Command("docker", "exec", f.container.GetContainerID(), "pg_dump", "-U", "postgres", "-d", "chnhcm", "-Fc")
	cmd.Stdout = io.MultiWriter(file, h)
	err = cmd.Run()
	must(t, file.Close())
	must(t, err)
	metadata := map[string]any{"instance": f.instance, "object": object, "source": source, "kind": kind, "recordCount": recordCount, "members": members, "dumpSHA256": hex.EncodeToString(h.Sum(nil)), "generatedThroughRealApplication": true, "timedRequests": 0}
	raw, err := json.MarshalIndent(metadata, "", "  ")
	must(t, err)
	must(t, os.WriteFile(filepath.Join(real, "input.json"), raw, 0600))
	must(t, os.WriteFile(filepath.Join(real, "snapshot.json"), []byte("{\"tables\":[]}\n"), 0600))
	t.Log("frozen real input exported; no timed request or capacity verdict")
	return true
}
