package foundation

import (
	"context"
	"time"
)

// StageObserver is a request-local diagnostic protocol. It carries no business
// data and never influences a decision, deadline or transaction outcome.
type StageObserver func(string, time.Duration)
type stageObserverKey struct{}

func WithStageObserver(ctx context.Context, observe StageObserver) context.Context {
	return context.WithValue(ctx, stageObserverKey{}, observe)
}

func HasStageObserver(ctx context.Context) bool {
	observe, _ := ctx.Value(stageObserverKey{}).(StageObserver)
	return observe != nil
}

func BeginStage(ctx context.Context, name string) func() {
	observe, _ := ctx.Value(stageObserverKey{}).(StageObserver)
	if observe == nil {
		return noStage
	}
	start := time.Now()
	return func() { observe(name, time.Since(start)) }
}

func noStage() {}
