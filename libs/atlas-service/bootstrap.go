package service

import (
	"context"
	"sync"
	"sync/atomic"

	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
	"github.com/sirupsen/logrus"
)

// projectionConfig and Projection are defined in projection.go (Task 5).

type bootstrapConfig struct {
	tracer     bool
	gates      []func() bool
	projection *projectionConfig
}

// Option configures Bootstrap.
type Option func(*bootstrapConfig)

// WithoutTracer skips otel tracer initialization (atlas-renders, tests).
func WithoutTracer() Option {
	return func(c *bootstrapConfig) { c.tracer = false }
}

// WithReadinessGate ANDs fn into Runtime.Ready(). Services with richer
// readiness (e.g. projection catch-up state) pass their gate here.
func WithReadinessGate(fn func() bool) Option {
	return func(c *bootstrapConfig) { c.gates = append(c.gates, fn) }
}

// Runtime is the handle Bootstrap returns; main.go composes the rest of
// startup (DB/Redis, consumers, REST server, tasks) around it.
type Runtime struct {
	logger       *logrus.Logger
	tdm          *Manager
	shuttingDown atomic.Bool
	gates        []func() bool
	projection   Projection
}

// Bootstrap owns the fleet-canonical startup sequence: logger, teardown
// manager, tracer (with teardown registered), the readiness controller,
// and — when the option is present — configuration-projection wiring.
// Fatal semantics match the per-service code it replaces (FR-4.5).
func Bootstrap(serviceName string, opts ...Option) *Runtime {
	cfg := &bootstrapConfig{tracer: true}
	for _, o := range opts {
		o(cfg)
	}

	l := CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := GetTeardownManager()
	rt := &Runtime{logger: l, tdm: tdm, gates: cfg.gates}

	if cfg.tracer {
		tc, err := tracing.InitTracer(serviceName)
		if err != nil {
			l.WithError(err).Fatal("Unable to initialize tracer.")
		}
		tdm.TeardownFunc(tracing.Teardown(l)(tc))
	}

	// Readiness controller: SIGTERM teardown flips /readyz to 503 before
	// downstream teardowns destroy state in-flight handlers might touch.
	// Teardown funcs fire concurrently on doneChan close, so registration
	// order here is not semantically meaningful.
	tdm.TeardownFunc(func() {
		rt.shuttingDown.Store(true)
		l.Info("Flipped /readyz to not-ready for graceful shutdown.")
	})

	if cfg.projection != nil {
		rt.startProjection(cfg.projection)
	}

	return rt
}

func (r *Runtime) Logger() *logrus.Logger     { return r.logger }
func (r *Runtime) Context() context.Context   { return r.tdm.Context() }
func (r *Runtime) WaitGroup() *sync.WaitGroup { return r.tdm.WaitGroup() }
func (r *Runtime) TeardownFunc(f func())      { r.tdm.TeardownFunc(f) }

// TeardownManager exposes the underlying *Manager for callees typed on it
// (e.g. atlas-login's buildListener).
func (r *Runtime) TeardownManager() *Manager { return r.tdm }

// Ready reports readiness for /readyz: not shutting down AND every
// WithReadinessGate fn true.
func (r *Runtime) Ready() bool {
	if r.shuttingDown.Load() {
		return false
	}
	for _, g := range r.gates {
		if !g() {
			return false
		}
	}
	return true
}

// Wait blocks until teardown completes, then logs the canonical
// shutdown line.
func (r *Runtime) Wait() {
	r.tdm.Wait()
	r.logger.Infoln("Service shutdown.")
}
