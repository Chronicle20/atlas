package skill

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestCachePutGet(t *testing.T) {
	t1 := testTenant(t)
	t2 := testTenant(t)

	if _, ok := GetCache().Get(t1, 2001002); ok {
		t.Fatal("unexpected cache hit before Put")
	}
	m, err := Extract(RestModel{Id: 2001002})
	if err != nil {
		t.Fatal(err)
	}
	GetCache().Put(t1, 2001002, m)

	got, ok := GetCache().Get(t1, 2001002)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Id() != 2001002 {
		t.Errorf("Id=%d, want 2001002", got.Id())
	}
	if _, ok := GetCache().Get(t2, 2001002); ok {
		t.Fatal("cache leaked across tenants")
	}
}

// GetById must serve a cached model without issuing any REST call: no
// DATA base URL is configured in tests, so a cache miss would error.
func TestGetByIdReadsThrough(t *testing.T) {
	tt := testTenant(t)
	ctx := tenant.WithContext(context.Background(), tt)
	l, _ := test.NewNullLogger()

	m, err := Extract(RestModel{Id: 4211005})
	if err != nil {
		t.Fatal(err)
	}
	GetCache().Put(tt, 4211005, m)

	got, err := NewProcessor(l, ctx).GetById(4211005)
	if err != nil {
		t.Fatalf("GetById should hit the cache, got err: %v", err)
	}
	if got.Id() != 4211005 {
		t.Errorf("Id=%d, want 4211005", got.Id())
	}
}
