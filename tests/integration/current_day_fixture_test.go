//go:build integration

package integration_test

import (
	"time"

	"chnhcm.local/chnhcm/internal/bootstrap"
	"chnhcm.local/chnhcm/internal/foundation"
)

// getOnDay exercises the ordinary HTTP contract at a controlled server day.
// Only this isolated fixture's Go clock changes; no date is sent in the request,
// and host/database clocks and the original date-boundary assertions are retained.
func (b *browser) getOnDay(day, path string) map[string]any {
	b.f.t.Helper()
	at, err := time.ParseInLocation("2006-01-02", day, foundation.Shanghai)
	must(b.f.t, err)
	restore := bootstrap.SetIntegrationClock(b.f.runtime, func() time.Time { return at.Add(12 * time.Hour) })
	defer restore()
	result := b.call("GET", path, nil, "", 200)
	if result["businessAt"] != day {
		b.f.t.Fatalf("server did not resolve the controlled day: %v", result["businessAt"])
	}
	return result
}
