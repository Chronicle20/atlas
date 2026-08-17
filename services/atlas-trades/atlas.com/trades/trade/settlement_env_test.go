package trade

import (
	"context"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// envMarkerKey is a test-local context key standing in for atlas-env's real
// marker. It pins that detached(), Reconcile and ReconcileEscrow apply the
// installed environment-origination function (SetEnvContext/applyEnvContext
// in settlement.go) to each rebuilt per-tenant context -- without the trade
// package importing atlas-env itself (env-domain-guard forbids that; main.go
// threads the real env.WithContext/env.Self() implementation in as a plain
// function value instead, via trade.SetEnvContext).
type envMarkerKey struct{}

// resetEnvContext restores the package-level environment-origination
// function to the identity default so one test's stub cannot leak into the
// next. Every test that calls SetEnvContext must defer this.
func resetEnvContext() {
	SetEnvContext(func(ctx context.Context) context.Context { return ctx })
}

// TestDetachedAppliesEnvContext pins the settlement.go:157 site: detached()
// rebuilds its context from context.Background() plus the tenant, dropping
// whatever environment the arming command's context carried, so it must
// re-apply the installed envContext rather than leaving the rebuilt context
// bare.
func TestDetachedAppliesEnvContext(t *testing.T) {
	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	ctx := tenant.WithContext(context.Background(), tm)
	p := NewProcessor(reconcileLogger(), ctx, db).(*ProcessorImpl)

	var envContextCalled bool
	var seenTenantId string
	envContext := func(c context.Context) context.Context {
		envContextCalled = true
		seenTenant := tenant.MustFromContext(c)
		seenTenantId = seenTenant.Id().String()
		return context.WithValue(c, envMarkerKey{}, "pod-env")
	}
	SetEnvContext(envContext)
	defer resetEnvContext()

	dp := p.detached()

	if !envContextCalled {
		t.Fatal("detached must apply the installed envContext to its rebuilt context")
	}
	if seenTenantId != tm.Id().String() {
		t.Errorf("envContext must observe the processor's own tenant: got %s, want %s", seenTenantId, tm.Id().String())
	}
	if dp.ctx.Value(envMarkerKey{}) != "pod-env" {
		t.Error("detached's rebuilt context does not carry the value envContext attached")
	}
}

// TestReconcileAppliesEnvContext pins the settlement.go:1255 site: each
// restored tenant's per-row context must go through the installed
// envContext before NewProcessor is built from it, so ReconcileSettlements'
// REST calls and emits carry this pod's environment.
func TestReconcileAppliesEnvContext(t *testing.T) {
	serveSagaOutcome(t, false)

	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	seedSettlementRecord(t, db, tm, uuid.New())

	var envContextCalled bool
	var seenTenantId string
	envContext := func(c context.Context) context.Context {
		envContextCalled = true
		seenTenant := tenant.MustFromContext(c)
		seenTenantId = seenTenant.Id().String()
		return context.WithValue(c, envMarkerKey{}, "pod-env")
	}
	SetEnvContext(envContext)
	defer resetEnvContext()

	if err := Reconcile(reconcileLogger(), context.Background(), db); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	if !envContextCalled {
		t.Fatal("Reconcile must apply the installed envContext to each restored tenant's context")
	}
	if seenTenantId != tm.Id().String() {
		t.Errorf("envContext must observe the restored row's own tenant: got %s, want %s", seenTenantId, tm.Id().String())
	}
}

// TestReconcileEscrowAppliesEnvContext pins the settlement.go:1368 site: the
// stranded-room processor built inside the sweep's loop must be constructed
// from a context that went through the installed envContext.
func TestReconcileEscrowAppliesEnvContext(t *testing.T) {
	db := reconcileDb(t)
	tm := reconcileTenant(t, "only")
	roomId := uuid.New()
	seedEscrowItem(t, db, tm, roomId, 100, 1)

	var envContextCalled bool
	var seenTenantId string
	envContext := func(c context.Context) context.Context {
		envContextCalled = true
		seenTenant := tenant.MustFromContext(c)
		seenTenantId = seenTenant.Id().String()
		return context.WithValue(c, envMarkerKey{}, "pod-env")
	}
	SetEnvContext(envContext)
	defer resetEnvContext()

	if err := ReconcileEscrow(reconcileLogger(), context.Background(), db, nil); err != nil {
		t.Fatalf("reconcile escrow: %v", err)
	}

	if !envContextCalled {
		t.Fatal("ReconcileEscrow must apply the installed envContext to each stranded room's context")
	}
	if seenTenantId != tm.Id().String() {
		t.Errorf("envContext must observe the stranded room's own tenant: got %s, want %s", seenTenantId, tm.Id().String())
	}
}
