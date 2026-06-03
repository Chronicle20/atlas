package storage

import "testing"

// TestResolveCacheGateNegativeNotCached pins the F3 fix: a negative probe
// (has=false) returns "shared" and signals shouldCache=false. Pre-fix the
// caller pinned "shared" in the scope cache for the pod lifetime, causing
// PR-544 to need a pod restart after ingest published tenant data.
func TestResolveCacheGateNegativeNotCached(t *testing.T) {
	scope, shouldCache := resolveCacheGate("t1", false)
	if scope != "shared" {
		t.Fatalf("scope = %q, want shared", scope)
	}
	if shouldCache {
		t.Fatal("shouldCache = true, want false (negative verdict must not be cached)")
	}
}

// TestResolveCacheGatePositiveCached asserts the positive path still caches.
func TestResolveCacheGatePositiveCached(t *testing.T) {
	scope, shouldCache := resolveCacheGate("t1", true)
	if scope != "tenants/t1" {
		t.Fatalf("scope = %q, want tenants/t1", scope)
	}
	if !shouldCache {
		t.Fatal("shouldCache = false, want true (positive verdict must be cached)")
	}
}
