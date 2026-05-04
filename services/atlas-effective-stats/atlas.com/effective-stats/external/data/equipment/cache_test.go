package equipment

import (
	"context"
	"errors"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func swapFetcher(t *testing.T, fn fetcher) *int {
	t.Helper()
	prev := defaultFetcher
	calls := 0
	defaultFetcher = func(ctx context.Context, l logrus.FieldLogger, id uint32) (EquipmentRequirements, error) {
		calls++
		return fn(ctx, l, id)
	}
	t.Cleanup(func() {
		defaultFetcher = prev
		getCache().reset()
	})
	return &calls
}

func tenantContext(t *testing.T, region string) (context.Context, tenant.Model) {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), region, 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn), tn
}

func TestProvider_CacheHitOnSecondCall(t *testing.T) {
	l, _ := test.NewNullLogger()
	calls := swapFetcher(t, func(_ context.Context, _ logrus.FieldLogger, id uint32) (EquipmentRequirements, error) {
		return EquipmentRequirements{ReqLuk: 40}, nil
	})

	ctx, _ := tenantContext(t, "GMS")
	p := GetProvider(l)
	if r, ok := p(ctx, 1052095); !ok || r.ReqLuk != 40 {
		t.Fatalf("first call: ok=%v r=%+v", ok, r)
	}
	if r, ok := p(ctx, 1052095); !ok || r.ReqLuk != 40 {
		t.Fatalf("second call: ok=%v r=%+v", ok, r)
	}
	if *calls != 1 {
		t.Errorf("fetch count = %d, want 1", *calls)
	}
}

func TestProvider_ColdCacheFetchFailureReturnsFalse(t *testing.T) {
	l, _ := test.NewNullLogger()
	calls := swapFetcher(t, func(_ context.Context, _ logrus.FieldLogger, id uint32) (EquipmentRequirements, error) {
		return EquipmentRequirements{}, errors.New("boom")
	})

	ctx, _ := tenantContext(t, "GMS")
	p := GetProvider(l)
	if _, ok := p(ctx, 1052095); ok {
		t.Errorf("expected (_, false) on cold-cache fetch failure")
	}
	if _, ok := p(ctx, 1052095); ok {
		t.Errorf("expected (_, false) on second cold-cache fetch failure too")
	}
	if *calls != 2 {
		t.Errorf("fetch count = %d, want 2 (cache only stores success)", *calls)
	}
}

func TestProvider_TenantIsolation(t *testing.T) {
	l, _ := test.NewNullLogger()
	calls := swapFetcher(t, func(_ context.Context, _ logrus.FieldLogger, id uint32) (EquipmentRequirements, error) {
		return EquipmentRequirements{ReqLuk: 40}, nil
	})

	ctxA, _ := tenantContext(t, "GMS")
	ctxB, _ := tenantContext(t, "JMS")
	p := GetProvider(l)
	_, _ = p(ctxA, 1052095)
	_, _ = p(ctxB, 1052095)
	if *calls != 2 {
		t.Errorf("fetch count = %d, want 2 (per-tenant)", *calls)
	}
}
