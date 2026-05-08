package redis

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func newTestTenant(t *testing.T, region string) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), region, 0, 83)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestTenantRegistry_Clear_EmptyNamespace(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer client.Close()
	reg := NewTenantRegistry[string, string](client, "test:clear", func(k string) string { return k })
	tm := newTestTenant(t, "GMS")

	deleted, err := reg.Clear(context.Background(), tm)
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}
