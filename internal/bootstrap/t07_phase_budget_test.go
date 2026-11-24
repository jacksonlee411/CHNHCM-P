package bootstrap_test

import (
	"chnhcm.local/chnhcm/internal/contexts/organization/domain"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

type t07ObservedContext struct {
	context.Context
	cancel                 context.CancelFunc
	line, occurrence, seen int
	at                     time.Time
}

func (c *t07ObservedContext) Err() error {
	e := c.Context.Err()
	_, file, line, ok := runtime.Caller(1)
	if ok && strings.HasSuffix(file, "/domain/dated/candidate.go") && line == c.line {
		c.seen++
		if c.seen == c.occurrence {
			c.at = time.Now()
			c.cancel()
		}
	}
	// Cancellation starts immediately after this real check; the next production
	// check must stop work. In particular standard-library sort must finish first.
	return e
}
func TestT07CancellationAtActualCalculationStages(t *testing.T) {
	raw, e := os.ReadFile("../contexts/organization/domain/dated/candidate.go")
	if e != nil {
		t.Fatal(e)
	}
	text := string(raw)
	base := make([]domain.Record, 120000)
	start, _ := time.Parse("2006-01-02", "2020-01-01")
	for i := range base {
		kind, state := "REACTIVATE", "active"
		if i == 0 {
			kind = "CREATE"
		} else if i%2 == 1 {
			kind, state = "DEACTIVATE", "inactive"
		}
		end := ""
		if i+1 < len(base) {
			end = start.AddDate(0, 0, (i+1)*2).Format("2006-01-02")
		}
		base[i] = domain.Record{ID: fmt.Sprint(i), Revision: "v1", Date: start.AddDate(0, 0, i*2).Format("2006-01-02"), End: end, Type: kind, State: state, Data: domain.BUData{Name: "甲"}}
		if kind == "DEACTIVATE" {
			base[i].InheritedRecord = fmt.Sprint(i - 1)
			base[i].InheritedRevision = "v1"
		}
	}
	for _, tc := range []struct {
		name, marker string
		n            int
	}{
		{"candidate", "for _, it := range items {", 1},
		{"sort_boundary", "currentItem = \"\"\n\tif err := ctx.Err();", 1},
		{"inheritance", "for _, r := range out.Records {", 2},
		{"impact", "func (k Kernel[T, P]) Compare(", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pos := strings.Index(text, tc.marker)
			if pos < 0 {
				t.Fatal("stage anchor missing")
			}
			off := strings.Index(text[pos:], "ctx.Err()")
			if off < 0 {
				t.Fatal("cancel check missing")
			}
			line := strings.Count(text[:pos+off], "\n") + 1
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			observed := &t07ObservedContext{Context: ctx, cancel: cancel, line: line, occurrence: tc.n}
			_, e := domain.Calculate(observed, base, "0", "2020-01-01", []domain.Item{{ID: "change", Action: "UPDATE", RecordID: "0", ExpectedRevision: "v1", Changes: domain.PatchFromFields(map[string]*string{"name": func() *string { v := "乙"; return &v }()})}})
			elapsed := time.Since(observed.at)
			if observed.at.IsZero() || !errors.Is(e, context.Canceled) || elapsed > time.Second {
				t.Fatal("stage not reached or cancellation unbounded", tc.name, observed.seen, e, elapsed)
			}
			t.Logf("actual stage=%s dated/candidate.go:%d occurrence=%d cancel-to-stop=%s (<1s), 120000 records", tc.name, line, tc.n, elapsed)
		})
	}
}
