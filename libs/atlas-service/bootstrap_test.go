package service

import (
	"sync/atomic"
	"testing"
)

// All Bootstrap tests use WithoutTracer so unit tests don't install global
// otel state. GetTeardownManager is a process-wide singleton; that is fine —
// each Bootstrap call just registers more teardown funcs on it.

func TestBootstrapRuntimeAccessors(t *testing.T) {
	rt := Bootstrap("atlas-test", WithoutTracer())
	if rt.Logger() == nil {
		t.Fatal("Logger() is nil")
	}
	if rt.Context() == nil {
		t.Fatal("Context() is nil")
	}
	if rt.WaitGroup() == nil {
		t.Fatal("WaitGroup() is nil")
	}
	if rt.TeardownManager() != GetTeardownManager() {
		t.Fatal("TeardownManager() must return the process singleton")
	}
	if !rt.Ready() {
		t.Fatal("fresh Runtime must be Ready")
	}
}

func TestBootstrapReadinessGatesAnd(t *testing.T) {
	var gateA, gateB atomic.Bool
	gateA.Store(true)
	gateB.Store(true)
	rt := Bootstrap("atlas-test", WithoutTracer(),
		WithReadinessGate(gateA.Load),
		WithReadinessGate(gateB.Load),
	)
	if !rt.Ready() {
		t.Fatal("all gates true → Ready")
	}
	gateB.Store(false)
	if rt.Ready() {
		t.Fatal("any gate false → not Ready")
	}
}
