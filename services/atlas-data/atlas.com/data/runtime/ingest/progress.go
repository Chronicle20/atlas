package ingest

import (
	"atlas-data/ingestrun"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

// progressWriteTimeout caps each progress write. Bounds a wedged Redis to
// ~22 × 5s across a multi-minute run: an ingest that would have succeeded must
// never fail because Redis did (PRD FR-2.5).
const progressWriteTimeout = 5 * time.Second

// redisSink is the Redis-backed data.ProgressSink. It also owns the run's
// Init/Finish bookends, so every write to the run record from the ingest pod
// goes through the same run-id guard and the same context discipline.
type redisSink struct {
	l     logrus.FieldLogger
	reg   *redis.Registry[string, ingestrun.Record]
	key   string
	runId string

	mu     sync.Mutex
	starts map[string]time.Time
}

func newRedisSink(l logrus.FieldLogger, reg *redis.Registry[string, ingestrun.Record], suffix, runId string) *redisSink {
	return &redisSink{
		l:      l,
		reg:    reg,
		key:    suffix + ingestrun.RunKeySuffix,
		runId:  runId,
		starts: make(map[string]time.Time),
	}
}

// writeCtx detaches from the caller's cancellation and bounds the write.
//
// When a worker fails, the errgroup cancels its context — and the terminal
// Finish(failed) write is exactly the write that must still land. Inheriting
// that cancellation would defeat FR-2.4 precisely when it matters most.
func (s *redisSink) writeCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), progressWriteTimeout)
}

// guardedUpdate runs fn through UpdateWithTTL under the run-id guard: a record
// whose RunId differs from this pod's is left untouched.
//
// The guard is evaluated inside the mutator, so it runs against the freshly
// read value on every optimistic-lock retry: a superseded pod can never win
// the race. An empty runId on either side means the guard cannot decide (a
// record written before run ids existed, or a pod with no INGEST_RUN_ID), and
// the write is allowed through. Every mutation — Init included — goes through
// this one guard so a stale pod can never revert a newer run's record.
func (s *redisSink) guardedUpdate(wctx context.Context, fn func(ingestrun.Record) ingestrun.Record) (err error, stale bool) {
	_, err = s.reg.UpdateWithTTL(wctx, s.key, ingestrun.RecordTTL, func(rec ingestrun.Record) ingestrun.Record {
		if rec.RunId != "" && s.runId != "" && rec.RunId != s.runId {
			stale = true
			return rec
		}
		stale = false
		return fn(rec)
	})
	return err, stale
}

// apply mutates the record under the run-id guard, logging (never returning)
// any failure — every non-Init write is best-effort telemetry.
func (s *redisSink) apply(ctx context.Context, what string, fn func(ingestrun.Record) ingestrun.Record) {
	if s == nil || s.reg == nil {
		return
	}
	wctx, cancel := s.writeCtx(ctx)
	defer cancel()

	err, stale := s.guardedUpdate(wctx, fn)
	if stale {
		s.l.Debugf("ingest progress write dropped (%s): record belongs to a different run than this pod (%s)", what, s.runId)
		return
	}
	if err != nil {
		s.l.WithError(err).Warnf("ingest progress write failed (%s, key=%s)", what, s.key)
	}
}

// Init adopts the record the REST pod wrote at Job creation — preserving its
// runId, jobName and startedAt (design Q2) — and only seeds the worker roster
// and confirms phase=running. seed is written only when no record exists, i.e.
// Redis lost it between Job creation and pod start.
//
// Routed through the same guardedUpdate as every other write: a pod scheduled
// so late that Redis already holds a newer run's record (possibly already
// terminal) must not revert that record back to running (design §3.1).
func (s *redisSink) Init(ctx context.Context, seed ingestrun.Record, roster []string, now time.Time) {
	if s == nil || s.reg == nil {
		return
	}
	wctx, cancel := s.writeCtx(ctx)
	defer cancel()

	err, stale := s.guardedUpdate(wctx, func(rec ingestrun.Record) ingestrun.Record {
		return rec.WithRoster(roster).WithPhase(ingestrun.PhaseRunning, now, "")
	})
	if stale {
		s.l.Debugf("ingest progress init dropped: record belongs to a different run than this pod (%s)", s.runId)
		return
	}
	if errors.Is(err, redis.ErrNotFound) {
		if perr := s.reg.PutWithTTL(wctx, s.key, seed, ingestrun.RecordTTL); perr != nil {
			s.l.WithError(perr).Warnf("ingest progress seed failed (key=%s)", s.key)
		}
		return
	}
	if err != nil {
		s.l.WithError(err).Warnf("ingest progress init failed (key=%s)", s.key)
	}
}

// Finish writes the terminal run phase. Called even when runErr aborted the
// errgroup, under a context that may already be cancelled — see writeCtx.
func (s *redisSink) Finish(ctx context.Context, runErr error, now time.Time) {
	if s == nil {
		return
	}
	phase := ingestrun.PhaseSucceeded
	reason := ""
	if runErr != nil {
		phase = ingestrun.PhaseFailed
		reason = runErr.Error()
	}
	s.l.Infof("ingest run %s", phase)
	s.apply(ctx, "finish", func(rec ingestrun.Record) ingestrun.Record {
		return rec.WithPhase(phase, now, reason)
	})
}

func (s *redisSink) WorkerStarted(ctx context.Context, name string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.starts[name] = time.Now()
	s.mu.Unlock()

	now := time.Now().UTC()
	s.l.Infof("ingest worker %s: running", name)
	s.apply(ctx, "worker-started:"+name, func(rec ingestrun.Record) ingestrun.Record {
		return rec.WithWorkerRunning(name, now)
	})
}

func (s *redisSink) WorkerFinished(ctx context.Context, name string, err error, skipped bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	start, ok := s.starts[name]
	delete(s.starts, name)
	s.mu.Unlock()

	var dur time.Duration
	if ok {
		dur = time.Since(start)
	}

	state := ingestrun.WorkerSucceeded
	msg := ""
	switch {
	case skipped:
		state = ingestrun.WorkerSkipped
	case err != nil:
		state = ingestrun.WorkerFailed
		msg = err.Error()
	}

	// Logged at info so an operator debugging without the UI gets the same
	// information from pod logs (PRD §8 Observability).
	s.l.Infof("ingest worker %s: %s (duration=%s)", name, state, dur.Truncate(time.Millisecond))

	now := time.Now().UTC()
	s.apply(ctx, "worker-finished:"+name, func(rec ingestrun.Record) ingestrun.Record {
		return rec.WithWorkerTerminal(name, state, now, msg)
	})
}
