package data

import (
	"atlas-data/data/workers"
	"context"
	"errors"
)

// ProgressSink receives per-worker lifecycle transitions from RunWorkers.
//
// Implementations return nothing on purpose: progress reporting is best-effort
// telemetry and must never fail, abort, or slow an ingest run (PRD FR-2.5).
// The Redis-backed implementation lives in runtime/ingest — this package
// cannot import it (runtime/ingest imports data), so the interface is declared
// here and satisfied there.
type ProgressSink interface {
	WorkerStarted(ctx context.Context, name string)
	// WorkerFinished reports a worker's terminal transition. skipped is true
	// for the ErrCategoryAbsent case, where err is nil and the run continues.
	WorkerFinished(ctx context.Context, name string, err error, skipped bool)
}

// noopSink is the default: it makes the compose and unit-test paths — where
// no run record exists — need no special-casing at the call sites.
type noopSink struct{}

func (noopSink) WorkerStarted(context.Context, string)               {}
func (noopSink) WorkerFinished(context.Context, string, error, bool) {}

type runConfig struct {
	sink ProgressSink
}

// RunOption configures RunWorkers. Variadic so every existing call site
// compiles unchanged.
type RunOption func(*runConfig)

// WithProgress routes per-worker transitions to s. A nil sink is ignored,
// leaving the no-op default in place.
func WithProgress(s ProgressSink) RunOption {
	return func(c *runConfig) {
		if s != nil {
			c.sink = s
		}
	}
}

func newRunConfig(opts []RunOption) runConfig {
	c := runConfig{sink: noopSink{}}
	for _, o := range opts {
		if o != nil {
			o(&c)
		}
	}
	return c
}

// runWithProgress invokes fn and reports its lifecycle to sink.
//
// The ErrCategoryAbsent contract lives here rather than inline in RunWorkers so
// it is assertable without MinIO, a database, or Redis: an absent category is
// reported as skipped and swallowed, because a category genuinely missing from
// a monolithic Data.wz (v12 has no Quest) must not fail the whole run.
func runWithProgress(ctx context.Context, sink ProgressSink, name string, fn func(context.Context) error) error {
	sink.WorkerStarted(ctx, name)
	err := fn(ctx)
	if errors.Is(err, workers.ErrCategoryAbsent) {
		sink.WorkerFinished(ctx, name, nil, true)
		return nil
	}
	sink.WorkerFinished(ctx, name, err, false)
	return err
}
