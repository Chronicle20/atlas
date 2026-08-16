package service

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TenantLister is the per-environment tenant source. Services pass their own
// tenant processor's GetAll, bound to the environment's context.
type TenantLister func(ctx context.Context) ([]tenant.Model, error)

// ForEachOwnedEnvironment runs body once per (environment, tenant) pair this
// deployment owns. BOTH dimensions are resolved fresh on every call
// (FR-6.4): a tenant provisioned after this pod started must be picked up
// without a restart, because a baseline pod cannot be redeployed to serve an
// ephemeral environment (G7, NG6). The pre-existing pattern this replaces
// loaded the tenant list once before the ticker and closed over the slice
// (design C4) — do not reintroduce that shape.
//
// Iteration is SERIAL, preserving the shape of the loops it replaces — every
// class-1 loop today is `for _, t := range tenants { work }`. Adding
// concurrency here would be a behavioural change unrelated to environment
// isolation, so it is opt-in via ForEachOwnedEnvironmentConcurrently.
//
// Each (environment, tenant) body runs under its own recover so one
// environment's fault does not stop another's (FR-6.5). Recovery does NOT
// require a goroutine; routine.Go merely bundles the two.
//
// If service is empty, env.CurrentRegistry().EnvironmentsOwnedBy("") is
// called as-is: no fallback is invented here. Against the legacy registry
// (no records projected, including the never-configured SERVICE_NAME case —
// Task 29A) that still yields exactly today's single legacy iteration
// (FR-1.8, FR-6.6). Against a populated registry, an empty service name
// never matches a real override, so ownership reduces to "am I the baseline
// of this environment" for every active environment.
func ForEachOwnedEnvironment(l logrus.FieldLogger, ctx context.Context,
	service string, tenants TenantLister, body func(context.Context),
) {
	eachOwned(l, ctx, service, tenants, func(el logrus.FieldLogger, c context.Context) {
		safely(el, c, body)
	})
}

// ForEachOwnedEnvironmentConcurrently runs each (environment, tenant) body in
// its own goroutine and blocks until all complete. Use it ONLY where the loop
// being converted already ran its tenants concurrently — otherwise a
// one-second ticker becomes a burst of goroutines across every tenant of
// every environment.
func ForEachOwnedEnvironmentConcurrently(l logrus.FieldLogger, ctx context.Context,
	service string, tenants TenantLister, body func(context.Context),
) {
	var wg sync.WaitGroup
	eachOwned(l, ctx, service, tenants, func(el logrus.FieldLogger, c context.Context) {
		wg.Add(1)
		routine.Go(el, c, func(gc context.Context) {
			defer wg.Done()
			body(gc)
		})
	})
	wg.Wait()
}

// eachOwned resolves the (environment, tenant) pairs this deployment owns and
// hands each to visit. BOTH dimensions are resolved fresh on every call
// (FR-6.4): a tenant provisioned after this pod started must be picked up
// without a restart, because a baseline pod cannot be redeployed to serve an
// ephemeral environment (G7, NG6). The pre-existing pattern this replaces
// loaded the tenant list once before the ticker and closed over the slice
// (design C4) — do not reintroduce that shape.
func eachOwned(l logrus.FieldLogger, ctx context.Context, service string,
	tenants TenantLister, visit func(logrus.FieldLogger, context.Context),
) {
	reg := env.CurrentRegistry()
	// tr is nil against a Registry that does not project tenants (including
	// legacyRegistry): the filter below then admits every tenant, matching
	// today's single-environment behaviour exactly.
	tr, _ := reg.(env.TenantResolver)
	for _, e := range reg.EnvironmentsOwnedBy(service) {
		ectx := env.WithContext(ctx, e)
		el := l.WithField("environment", string(e))
		ts, err := tenants(ectx)
		if err != nil {
			el.WithError(err).Error("Unable to list tenants; skipping this environment's iteration.")
			continue
		}
		for _, t := range ts {
			// A TenantLister is documented to return only its own
			// environment's tenants, but not every caller honors that
			// (e.g. a local DB/session-backed lister that ignores context).
			// Filter here so the contract holds regardless of the lister:
			// keep t when it is projected to e, or when it is not projected
			// at all (unknown tenants pass through, mirroring the
			// unknown-tenant rule in Reconcile / tenants.go).
			if tr != nil {
				if te, ok := tr.EnvironmentOfTenant(t.Id().String()); ok && te != e {
					continue
				}
			}
			visit(el, tenant.WithContext(ectx, t))
		}
	}
}

// safely runs body with panic recovery, on the CALLING goroutine. It is
// deliberately not routine.Go: fault isolation and concurrency are separate
// concerns, and this helper needs only the first.
func safely(l logrus.FieldLogger, ctx context.Context, body func(context.Context)) {
	defer func() {
		if r := recover(); r != nil {
			l.WithField("panic", fmt.Sprintf("%v", r)).
				WithField("stack", string(debug.Stack())).
				Error("Recovered panic in a per-environment iteration.")
		}
	}()
	body(ctx)
}
