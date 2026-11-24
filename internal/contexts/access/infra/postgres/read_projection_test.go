package postgres

import (
	access "chnhcm.local/chnhcm/internal/contexts/access/public"
	"context"
	"testing"
)

func TestReadProjectionRejectsCommandPurposeBeforeEvaluation(t *testing.T) {
	g := &guard{}
	valid := access.ReadIntent{Action: "department.read", Kind: "object"}
	for name, in := range map[string]access.ReadIntent{
		"write":   {Action: "department.record.correct", Kind: "object"},
		"closure": {Action: "department.read", Kind: "recovery"},
		"history": {Action: "department.history.read", Kind: "history"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, e := g.DecideReads(context.Background(), []access.ReadIntent{valid, in}); e == nil {
				t.Fatal("write-shaped batch was accepted")
			}
		})
	}
}
