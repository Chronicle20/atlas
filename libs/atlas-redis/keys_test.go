package redis

import (
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func TestKeyEnv_fallsBackToAtlasEnv(t *testing.T) {
	t.Setenv("ATLAS_ENV", "a435")
	t.Setenv("ATLAS_REDIS_ENV", "")
	if got := keyEnv(); got != "a435" {
		t.Fatalf("keyEnv() = %q, want %q", got, "a435")
	}
}

// A sparse overlay sets ATLAS_REDIS_ENV to the baseline's environment id so
// its override pods address the same keyspace as the baseline pods, while
// ATLAS_ENV stays per-deployment for libs/atlas-lock's lease scoping.
func TestKeyEnv_redisEnvOverridesAtlasEnv(t *testing.T) {
	t.Setenv("ATLAS_ENV", "a435")
	t.Setenv("ATLAS_REDIS_ENV", "main")
	if got := keyEnv(); got != "main" {
		t.Fatalf("keyEnv() = %q, want %q", got, "main")
	}
}

func TestKeyEnv_bothUnset(t *testing.T) {
	t.Setenv("ATLAS_ENV", "")
	t.Setenv("ATLAS_REDIS_ENV", "")
	if got := keyEnv(); got != "" {
		t.Fatalf("keyEnv() = %q, want %q", got, "")
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
