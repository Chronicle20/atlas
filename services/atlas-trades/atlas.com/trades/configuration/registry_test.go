package configuration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(discardWriter{})
	return l
}

func tenantContext(t *testing.T) (context.Context, uuid.UUID) {
	t.Helper()
	id := uuid.New()
	tm, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm), id
}

// TestGetFallsBackToDefaultsOnFetchError pins FR-9.2, which design §8 calls out
// as the path that MUST be tested rather than the exceptional one: a tenant
// with no trade-configs resource runs on the shipped defaults instead of
// crashing or silently disabling trading.
func TestGetFallsBackToDefaultsOnFetchError(t *testing.T) {
	ctx, _ := tenantContext(t)

	calls := 0
	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID) (Model, error) {
		calls++
		return Model{}, errors.New("404 no trade-configs resource")
	})

	got := r.Get(testLogger(), ctx)

	d := DefaultConfig()
	if got.TaxEnabled() != d.TaxEnabled() || got.MaxStagedItems() != d.MaxStagedItems() ||
		got.ReservationTtl() != d.ReservationTtl() || got.AttestationTimeout() != d.AttestationTimeout() ||
		len(got.TaxTiers()) != len(d.TaxTiers()) {
		t.Fatalf("expected the shipped defaults on a fetch miss, got %+v", got)
	}
	if calls != 1 {
		t.Fatalf("expected the fetcher to be invoked once, got %d", calls)
	}
}

// TestGetCachesPerTenant pins that a resolved config is fetched once per tenant
// and that the INFO fallback notice therefore cannot spam the log.
func TestGetCachesPerTenant(t *testing.T) {
	ctx, _ := tenantContext(t)

	calls := 0
	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID) (Model, error) {
		calls++
		return DefaultConfig().WithMaxStagedItems(4), nil
	})

	l := testLogger()
	if got := r.Get(l, ctx).MaxStagedItems(); got != 4 {
		t.Fatalf("first call: MaxStagedItems got %d, want 4", got)
	}
	if got := r.Get(l, ctx).MaxStagedItems(); got != 4 {
		t.Fatalf("second call: MaxStagedItems got %d, want 4", got)
	}
	if calls != 1 {
		t.Fatalf("expected exactly one fetch (cache hit on the second call), got %d", calls)
	}
}

// TestGetIsolatesTenants pins that the cache is keyed by tenant id — one
// tenant's configuration must never be served to another.
func TestGetIsolatesTenants(t *testing.T) {
	ctxA, idA := tenantContext(t)
	ctxB, _ := tenantContext(t)

	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, id uuid.UUID) (Model, error) {
		if id == idA {
			return DefaultConfig().WithMaxStagedItems(1), nil
		}
		return DefaultConfig().WithMaxStagedItems(2), nil
	})

	l := testLogger()
	if got := r.Get(l, ctxA).MaxStagedItems(); got != 1 {
		t.Errorf("tenant A: MaxStagedItems got %d, want 1", got)
	}
	if got := r.Get(l, ctxB).MaxStagedItems(); got != 2 {
		t.Errorf("tenant B: MaxStagedItems got %d, want 2", got)
	}
}

// TestGetWithoutATenantReturnsDefaults pins that a context with no tenant
// yields the defaults rather than panicking — configuration lookup must never
// be the thing that takes a request down.
func TestGetWithoutATenantReturnsDefaults(t *testing.T) {
	calls := 0
	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID) (Model, error) {
		calls++
		return DefaultConfig().WithMaxStagedItems(7), nil
	})

	got := r.Get(testLogger(), context.Background())

	if got.MaxStagedItems() != DefaultConfig().MaxStagedItems() {
		t.Errorf("MaxStagedItems: got %d, want the default %d", got.MaxStagedItems(), DefaultConfig().MaxStagedItems())
	}
	if calls != 0 {
		t.Errorf("expected no fetch without a tenant, got %d", calls)
	}
}

// TestGetRegistryIsASingleton pins that every caller shares one cache.
func TestGetRegistryIsASingleton(t *testing.T) {
	if GetRegistry() != GetRegistry() {
		t.Fatal("GetRegistry returned two different instances")
	}
}
