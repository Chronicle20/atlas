package configuration

import (
	"context"
	"errors"
	"testing"
	"time"

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

// TestGetFallsBackToDefaultsOnFetchError pins FR-2.6's must-test path: an
// unseeded tenant (no imprint-configs resource) resolves to the shipped 168h
// default rather than failing or expiring every pending change instantly.
func TestGetFallsBackToDefaultsOnFetchError(t *testing.T) {
	ctx, _ := tenantContext(t)

	calls := 0
	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID) (Model, error) {
		calls++
		return Model{}, errors.New("404 no imprint-configs resource")
	})

	got := r.Get(testLogger(), ctx)

	if got.PendingExpiry() != DefaultPendingExpiry {
		t.Fatalf("PendingExpiry = %v, want the default %v", got.PendingExpiry(), DefaultPendingExpiry)
	}
	if calls != 1 {
		t.Fatalf("expected the fetcher to be invoked once, got %d", calls)
	}
}

// TestGetResolvesASeededTenantsConfiguredValue pins that a seeded tenant's
// operator-set expiry is what the registry serves — not merely that a fetch
// error falls back to the default.
func TestGetResolvesASeededTenantsConfiguredValue(t *testing.T) {
	ctx, _ := tenantContext(t)

	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID) (Model, error) {
		return Extract(RestModel{Id: "imprint-configs", PendingExpiryHours: 72}), nil
	})

	got := r.Get(testLogger(), ctx).PendingExpiry()
	if got != 72*time.Hour {
		t.Fatalf("PendingExpiry = %v, want 72h", got)
	}
}

// TestGetCachesPerTenant pins that a resolved config is fetched once per
// tenant, so the fallback/seeded notice cannot spam the log.
func TestGetCachesPerTenant(t *testing.T) {
	ctx, _ := tenantContext(t)

	calls := 0
	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID) (Model, error) {
		calls++
		return Extract(RestModel{PendingExpiryHours: 48}), nil
	})

	l := testLogger()
	if got := r.Get(l, ctx).PendingExpiry(); got != 48*time.Hour {
		t.Fatalf("first call: PendingExpiry got %v, want 48h", got)
	}
	if got := r.Get(l, ctx).PendingExpiry(); got != 48*time.Hour {
		t.Fatalf("second call: PendingExpiry got %v, want 48h", got)
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
			return Extract(RestModel{PendingExpiryHours: 12}), nil
		}
		return Extract(RestModel{PendingExpiryHours: 96}), nil
	})

	l := testLogger()
	if got := r.Get(l, ctxA).PendingExpiry(); got != 12*time.Hour {
		t.Errorf("tenant A: PendingExpiry got %v, want 12h", got)
	}
	if got := r.Get(l, ctxB).PendingExpiry(); got != 96*time.Hour {
		t.Errorf("tenant B: PendingExpiry got %v, want 96h", got)
	}
}

// TestGetWithoutATenantReturnsDefaults pins that a context with no tenant
// yields the defaults rather than panicking.
func TestGetWithoutATenantReturnsDefaults(t *testing.T) {
	calls := 0
	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID) (Model, error) {
		calls++
		return Extract(RestModel{PendingExpiryHours: 12}), nil
	})

	got := r.Get(testLogger(), context.Background())

	if got.PendingExpiry() != DefaultPendingExpiry {
		t.Errorf("PendingExpiry: got %v, want the default %v", got.PendingExpiry(), DefaultPendingExpiry)
	}
	if calls != 0 {
		t.Errorf("expected no fetch without a tenant, got %d", calls)
	}
}

// TestGetRegistryIsASingleton pins that every caller shares one cache.
func TestGetRegistryIsASingleton(t *testing.T) {
	first := GetRegistry()
	second := GetRegistry()
	if first != second {
		t.Fatal("GetRegistry returned two different instances")
	}
}
