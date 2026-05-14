package redis

import (
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func TestComputeKeyPrefix_envUnset(t *testing.T) {
	got := computeKeyPrefix("")
	if got != "atlas" {
		t.Fatalf("computeKeyPrefix(\"\") = %q, want %q", got, "atlas")
	}
}

func TestComputeKeyPrefix_envSet(t *testing.T) {
	got := computeKeyPrefix("a3f7")
	if got != "a3f7:atlas" {
		t.Fatalf("computeKeyPrefix(\"a3f7\") = %q, want %q", got, "a3f7:atlas")
	}
}

func TestKeyPrefix_returnsBaseWhenEnvUnset(t *testing.T) {
	if got := KeyPrefix(); got == "" {
		t.Fatalf("KeyPrefix() returned empty string")
	}
}

func TestNamespacedKey_useEnvAwarePrefix(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("a3f7")

	got := namespacedKey("buffs", "_tenants")
	want := "a3f7:atlas:buffs:_tenants"
	if got != want {
		t.Fatalf("namespacedKey = %q, want %q", got, want)
	}
}

func TestTenantEntityKey_useEnvAwarePrefix(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("a3f7")

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	got := tenantEntityKey("buffs", tm, "42")
	if want := "a3f7:atlas:buffs:"; len(got) < len(want) || got[:len(want)] != want {
		t.Fatalf("tenantEntityKey = %q, want prefix %q", got, want)
	}
}
