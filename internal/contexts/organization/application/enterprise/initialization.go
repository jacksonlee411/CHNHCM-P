package enterprise

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	d "chnhcm.local/chnhcm/internal/contexts/organization/domain/enterprise"
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"time"
)

// Initialize records the preserved initialization declaration as the first source.
// The controlled upgrader owns this port; it is not an HTTP creation command.
func Initialize(ctx context.Context, id, code, name, start, digest string, now time.Time, operator access.OperatorSnapshot) (Write, error) {
	t := d.Timeline{ID: id, Code: code, First: foundation.NewID(), InitializationDigest: digest}
	c, e := d.Kernel.Create(ctx, t.First, start, d.MinDate, "NEW_ORGANIZATION", d.Data{Name: name})
	if e != nil {
		return Write{}, e
	}
	cmd := org.EnterpriseCommand{Action: "CREATE", EnterpriseID: id, SubmissionID: foundation.NewID()}
	term := Terminal{ID: cmd.SubmissionID, ObjectID: id, Action: "CREATE"}
	return assemble(State{Timeline: t}, c, cmd, term, now, operator), nil
}
