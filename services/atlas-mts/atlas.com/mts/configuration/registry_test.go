package configuration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(testWriter{})
	return l
}

type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) { return len(p), nil }

// On a fetch miss/error, GetTenantConfig must return the defaults rather than
// propagating the error.
func TestGetTenantConfigDefaultsOnFetchError(t *testing.T) {
	calls := 0
	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID) (Model, error) {
		calls++
		return Model{}, errors.New("fetch failed")
	})

	got := r.GetTenantConfig(testLogger(), context.Background(), uuid.New())

	want := DefaultConfig()
	if got != want {
		t.Fatalf("expected defaults on fetch error, got %+v want %+v", got, want)
	}
	if calls != 1 {
		t.Fatalf("expected fetcher to be invoked once, got %d", calls)
	}
}

// A successful fetch is cached: the second call for the same tenant must not
// re-invoke the fetcher.
func TestGetTenantConfigCachesFetchedConfig(t *testing.T) {
	tenantId := uuid.New()
	fetched := DefaultConfig()
	fetched.pageSize = 99 // distinguish from defaults

	calls := 0
	r := newRegistryWithFetcher(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID) (Model, error) {
		calls++
		return fetched, nil
	})

	l := testLogger()
	ctx := context.Background()

	first := r.GetTenantConfig(l, ctx, tenantId)
	if first != fetched {
		t.Fatalf("first call: expected fetched config, got %+v", first)
	}

	second := r.GetTenantConfig(l, ctx, tenantId)
	if second != fetched {
		t.Fatalf("second call: expected cached config, got %+v", second)
	}

	if calls != 1 {
		t.Fatalf("expected fetcher to be invoked exactly once (cache hit on second call), got %d", calls)
	}
}
