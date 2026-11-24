package foundation

import (
	"context"
	"testing"
	"time"
)

func TestOfflineGuardBudgetKeepsParentCancellationAndDeadline(t *testing.T) {
	ordinary := context.Background()
	if ObjectGuardBudget(ordinary) != 2*time.Second {
		t.Fatal("ordinary guard budget changed")
	}
	parent, cancel := context.WithTimeout(ordinary, time.Second)
	imported := WithOfflineImportGuardBudget(parent)
	if ObjectGuardBudget(imported) != 8*time.Second {
		t.Fatal("offline preparation budget missing")
	}
	guarded, done := context.WithTimeout(imported, ObjectGuardBudget(imported))
	defer done()
	before, _ := parent.Deadline()
	after, _ := guarded.Deadline()
	if !after.Equal(before) {
		t.Fatal("offline budget extended request deadline")
	}
	cancel()
	if guarded.Err() != context.Canceled {
		t.Fatal("offline budget lost cancellation")
	}
	if ObjectGuardBudget(ordinary) != 2*time.Second {
		t.Fatal("offline budget leaked to other requests")
	}
}

func TestDepartmentPreparationKeepsEarlierDeadlineAndOtherObjects(t *testing.T) {
	ordinary := context.Background()
	if DepartmentPreparationBudget(ordinary) != 5*time.Second || ObjectGuardBudget(ordinary) != 2*time.Second {
		t.Fatal("department preparation changed an unrelated object budget")
	}
	parent, cancel := context.WithTimeout(ordinary, time.Second)
	defer cancel()
	for _, input := range []context.Context{parent, WithOfflineImportGuardBudget(parent)} {
		bounded, done := context.WithTimeout(input, DepartmentPreparationBudget(input))
		before, _ := parent.Deadline()
		after, _ := bounded.Deadline()
		if !after.Equal(before) {
			t.Fatal("preparation extended the caller's deadline")
		}
		done()
	}
	cancel()
	bounded, done := context.WithTimeout(parent, DepartmentPreparationBudget(parent))
	defer done()
	if bounded.Err() != context.Canceled || DepartmentPreparationBudget(WithOfflineImportGuardBudget(ordinary)) != 8*time.Second {
		t.Fatal("preparation lost cancellation or changed the offline budget")
	}
}
