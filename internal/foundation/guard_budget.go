package foundation

import (
	"context"
	"time"
)

type offlineImportGuardKey struct{}

const DepartmentLockAcquisitionBudget = 2 * time.Second

// WithOfflineImportGuardBudget is for an ownership- and source-checked offline
// importer. HTTP input cannot select it. SQL lock waits and request cancellation
// remain bounded independently; this only allows more dependency preparation.
func WithOfflineImportGuardBudget(ctx context.Context) context.Context {
	return context.WithValue(ctx, offlineImportGuardKey{}, true)
}

func ObjectGuardBudget(ctx context.Context) time.Duration {
	if enabled, _ := ctx.Value(offlineImportGuardKey{}).(bool); enabled {
		return 8 * time.Second
	}
	return 2 * time.Second
}

// DepartmentPreparationBudget includes dependency reads and replanning. It is
// separate from time spent acquiring locks, and never extends a caller's
// deadline. The offline importer keeps its existing, explicitly owned budget.
func DepartmentPreparationBudget(ctx context.Context) time.Duration {
	if enabled, _ := ctx.Value(offlineImportGuardKey{}).(bool); enabled {
		return 8 * time.Second
	}
	return 5 * time.Second
}
