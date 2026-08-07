package main

import (
	"atlas-transports/instance"
	"atlas-transports/transport"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type fakeScheduledLoader struct {
	routes  []transport.Model
	vessels []transport.SharedVesselModel
	err     error
}

func (f fakeScheduledLoader) LoadConfigurationsForTenant(tenant.Model) ([]transport.Model, []transport.SharedVesselModel, error) {
	return f.routes, f.vessels, f.err
}

type fakeScheduledRegistry struct {
	cleared int
	added   int
}

func (f *fakeScheduledRegistry) ClearTenant() int { f.cleared++; return 0 }
func (f *fakeScheduledRegistry) AddTenant([]transport.Model, []transport.SharedVesselModel) error {
	f.added++
	return nil
}

type fakeInstanceLoader struct {
	routes []instance.RouteModel
	err    error
}

func (f fakeInstanceLoader) LoadConfigurationsForTenant(tenant.Model) ([]instance.RouteModel, error) {
	return f.routes, f.err
}

type fakeInstanceRegistry struct {
	cleared int
	added   int
}

func (f *fakeInstanceRegistry) ClearTenant() int                { f.cleared++; return 0 }
func (f *fakeInstanceRegistry) AddTenant([]instance.RouteModel) { f.added++ }

func quietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func aTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestReconcileScheduled_ClearsThenAddsOnSuccess(t *testing.T) {
	reg := &fakeScheduledRegistry{}
	if err := reconcileScheduled(quietLogger(), aTenant(t), fakeScheduledLoader{}, reg); err != nil {
		t.Fatalf("reconcileScheduled: %v", err)
	}
	if reg.cleared != 1 || reg.added != 1 {
		t.Fatalf("cleared=%d added=%d, want 1 and 1", reg.cleared, reg.added)
	}
}

// The whole point of load-first: a load failure must leave the existing
// registry contents alone rather than wiping them.
func TestReconcileScheduled_LeavesRegistryUntouchedOnLoadError(t *testing.T) {
	reg := &fakeScheduledRegistry{}
	err := reconcileScheduled(quietLogger(), aTenant(t), fakeScheduledLoader{err: errors.New("boom")}, reg)
	if err == nil {
		t.Fatal("reconcileScheduled returned nil error, want the load error")
	}
	if reg.cleared != 0 || reg.added != 0 {
		t.Fatalf("cleared=%d added=%d, want 0 and 0 — a load failure must not clear", reg.cleared, reg.added)
	}
}

func TestReconcileInstance_ClearsThenAddsOnSuccess(t *testing.T) {
	reg := &fakeInstanceRegistry{}
	if err := reconcileInstance(quietLogger(), aTenant(t), fakeInstanceLoader{}, reg); err != nil {
		t.Fatalf("reconcileInstance: %v", err)
	}
	if reg.cleared != 1 || reg.added != 1 {
		t.Fatalf("cleared=%d added=%d, want 1 and 1", reg.cleared, reg.added)
	}
}

func TestReconcileInstance_LeavesRegistryUntouchedOnLoadError(t *testing.T) {
	reg := &fakeInstanceRegistry{}
	err := reconcileInstance(quietLogger(), aTenant(t), fakeInstanceLoader{err: errors.New("boom")}, reg)
	if err == nil {
		t.Fatal("reconcileInstance returned nil error, want the load error")
	}
	if reg.cleared != 0 || reg.added != 0 {
		t.Fatalf("cleared=%d added=%d, want 0 and 0", reg.cleared, reg.added)
	}
}
