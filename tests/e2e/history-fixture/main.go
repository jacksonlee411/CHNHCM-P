//go:build integration

// Command history-fixture changes one explicitly temporary browser fixture.
package main

import (
	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func run() error {
	var in struct{ Object, Source, Revision string }
	if e := json.NewDecoder(os.Stdin).Decode(&in); e != nil {
		return e
	}
	for _, v := range []string{in.Object, in.Source, in.Revision} {
		if !foundation.ValidID(v) {
			return fmt.Errorf("invalid fixture identity")
		}
	}
	local := func(args ...string) ([]byte, error) {
		return exec.Command("python3", append([]string{"scripts/local-bu.py"}, args...)...).Output()
	}
	raw, e := local("credentials-path")
	if e != nil {
		return fmt.Errorf("fixture profile unavailable")
	}
	path := strings.TrimSpace(string(raw))
	cwd, e := os.Getwd()
	if e != nil {
		return e
	}
	rel, e := filepath.Rel(cwd, path)
	if e != nil || !filepath.IsAbs(path) || !strings.HasPrefix(rel, "../") {
		return fmt.Errorf("private profile must be outside checkout")
	}
	metaRaw, e := os.ReadFile(filepath.Join(filepath.Dir(path), "profile.json"))
	if e != nil {
		return e
	}
	var meta struct {
		Temporary         bool
		Worktree, Project string
	}
	if e = json.Unmarshal(metaRaw, &meta); e != nil {
		return e
	}
	if !meta.Temporary || meta.Worktree != cwd {
		return fmt.Errorf("historical fixture requires this task's temporary browser profile")
	}
	private, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	var cfg struct {
		Instance string
		Password string `json:"database_password"`
	}
	if e = json.Unmarshal(private, &cfg); e != nil {
		return e
	}
	raw, e = local("connection")
	if e != nil {
		return e
	}
	var conn struct {
		Project string
		Ports   struct{ Database int }
	}
	if e = json.Unmarshal(raw, &conn); e != nil {
		return e
	}
	if conn.Project != meta.Project {
		return fmt.Errorf("profile owner mismatch")
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword("postgres", cfg.Password), Host: fmt.Sprintf("127.0.0.1:%d", conn.Ports.Database), Path: "/chnhcm", RawQuery: "sslmode=disable"}
	ctx := context.Background()
	rt, e := bootstrap.Open(ctx, u.String(), cfg.Instance)
	if e != nil {
		return e
	}
	defer rt.Close()
	return bootstrap.MutateIntegrationBU(ctx, rt.Pool, rt, in.Object, in.Source, func(v *bootstrap.BUHistoryFact) {
		for i := range v.State.Timeline.Records {
			r := &v.State.Timeline.Records[i]
			if r.Revision == in.Revision {
				r.Reason = "NAME_CHANGE"
				r.TypeLabel = "资料变更"
				r.ReasonLabel = "名称变更"
			}
		}
		for i := range v.Operation.Revisions {
			r := &v.Operation.Revisions[i].Record
			if r.Revision == in.Revision {
				r.Reason = "NAME_CHANGE"
				r.TypeLabel = "资料变更"
				r.ReasonLabel = "名称变更"
			}
		}
	})
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
