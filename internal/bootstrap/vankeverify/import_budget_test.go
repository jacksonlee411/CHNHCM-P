package vankeverify

import (
	org "chnhcm.local/chnhcm/internal/contexts/organization/public"
	"chnhcm.local/chnhcm/internal/foundation"
	"context"
	"testing"
	"time"
)

type budgetCapture struct {
	org.DepartmentService
	budget time.Duration
}

func (s *budgetCapture) Execute(ctx context.Context, _ foundation.Credentials, _ org.DepartmentCommand) (org.DepartmentResult, error) {
	s.budget = foundation.DepartmentPreparationBudget(ctx)
	return org.DepartmentResult{}, nil
}
func TestOnlyAdmittedSourceKeysReceiveOfflinePreparationBudget(t *testing.T) {
	underlying := &budgetCapture{}
	approved := importKey("parent-change", 32)
	adapter := &importNames{DepartmentService: underlying, admitted: map[string]bool{approved: true}}
	for _, test := range []struct {
		key    string
		budget time.Duration
	}{{foundation.NewID(), 5 * time.Second}, {approved, 8 * time.Second}, {foundation.NewID(), 5 * time.Second}} {
		_, err := adapter.Execute(context.Background(), foundation.Credentials{}, org.DepartmentCommand{SubmissionID: test.key})
		if err != nil || underlying.budget != test.budget {
			t.Fatalf("unexpected budget %v, error %v", underlying.budget, err)
		}
	}
}
