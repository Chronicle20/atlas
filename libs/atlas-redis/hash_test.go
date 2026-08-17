package redis

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestHash_SetGetDelExistsGetAll(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("a3f7")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	h := NewHash(client, "transport:characters")

	if err := h.Set(ctx, "1001", "inst-a"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !mr.Exists("a3f7:atlas:transport:characters") {
		t.Fatalf("expected key a3f7:atlas:transport:characters; keys=%v", mr.Keys())
	}
	v, err := h.Get(ctx, "1001")
	if err != nil || v != "inst-a" {
		t.Fatalf("Get = %q,%v want inst-a,nil", v, err)
	}
	ok, _ := h.Exists(ctx, "1001")
	if !ok {
		t.Fatalf("Exists 1001 want true")
	}
	if _, err := h.Get(ctx, "nope"); err != ErrNotFound {
		t.Fatalf("Get missing = %v want ErrNotFound", err)
	}
	all, _ := h.GetAll(ctx)
	if len(all) != 1 {
		t.Fatalf("GetAll len = %d want 1", len(all))
	}
	if err := h.Del(ctx, "1001"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if ok, _ := h.Exists(ctx, "1001"); ok {
		t.Fatalf("Exists after Del want false")
	}
}

func TestTenantHash_PerTenantKeyAndOps(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	h := NewTenantHash(client, "transport:characters")
	t1 := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)

	if err := h.Set(ctx, t1, "1001", "inst-a"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	wantKey := "atlas:transport:characters:" + TenantKey(t1)
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}
	v, err := h.Get(ctx, t1, "1001")
	if err != nil || v != "inst-a" {
		t.Fatalf("Get = %q,%v want inst-a,nil", v, err)
	}
	ok, _ := h.Exists(ctx, t1, "1001")
	if !ok {
		t.Fatalf("Exists 1001 want true")
	}
	if _, err := h.Get(ctx, t1, "nope"); err != ErrNotFound {
		t.Fatalf("Get missing = %v want ErrNotFound", err)
	}
	all, _ := h.GetAll(ctx, t1)
	if len(all) != 1 {
		t.Fatalf("GetAll len = %d want 1", len(all))
	}
	if err := h.Del(ctx, t1, "1001"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if ok, _ := h.Exists(ctx, t1, "1001"); ok {
		t.Fatalf("Exists after Del want false")
	}
}

// TestTenantHash_TwoTenantsSameFieldDoNotCollide is the bug this task exists
// to fix: character_registry.go used to key an env-global Hash on bare
// characterId, so tenant A's character 12345 and tenant B's character 12345
// wrote the same hash field. Proves the same field name under two different
// tenants resolves to two distinct hashes.
func TestTenantHash_TwoTenantsSameFieldDoNotCollide(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()
	h := NewTenantHash(client, "transport:characters")
	t1 := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
	t2 := makeTenant("00000000-0000-0000-0000-000000000002", "GMS", 83, 1)

	if err := h.Set(ctx, t1, "12345", "inst-a"); err != nil {
		t.Fatalf("Set t1: %v", err)
	}
	if err := h.Set(ctx, t2, "12345", "inst-b"); err != nil {
		t.Fatalf("Set t2: %v", err)
	}

	v1, err := h.Get(ctx, t1, "12345")
	if err != nil || v1 != "inst-a" {
		t.Fatalf("t1 Get(12345) = %q,%v want inst-a,nil", v1, err)
	}
	v2, err := h.Get(ctx, t2, "12345")
	if err != nil || v2 != "inst-b" {
		t.Fatalf("t2 Get(12345) = %q,%v want inst-b,nil", v2, err)
	}

	ok1, _ := h.Exists(ctx, t1, "12345")
	ok2, _ := h.Exists(ctx, t2, "12345")
	if !ok1 || !ok2 {
		t.Fatalf("both tenants must independently see field 12345: t1=%v t2=%v", ok1, ok2)
	}

	// Deleting for t1 must not affect t2's hash.
	if err := h.Del(ctx, t1, "12345"); err != nil {
		t.Fatalf("Del t1: %v", err)
	}
	if ok, _ := h.Exists(ctx, t1, "12345"); ok {
		t.Fatalf("t1 12345 should be gone after Del")
	}
	if ok, _ := h.Exists(ctx, t2, "12345"); !ok {
		t.Fatalf("t2 12345 must survive t1's Del")
	}
}

func TestKeyedHash_PerKeyHashKeyAndOps(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	kh := NewKeyedHash[uuid.UUID](client, "transport:instance:chars", func(id uuid.UUID) string { return id.String() })
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	if got := kh.Key(id); got != "atlas:transport:instance:chars:"+id.String() {
		t.Fatalf("Key = %q", got)
	}
	if err := kh.Set(ctx, id, "1001", "entry"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !mr.Exists("atlas:transport:instance:chars:" + id.String()) {
		t.Fatalf("expected per-key hash; keys=%v", mr.Keys())
	}
	if n, _ := kh.Len(ctx, id); n != 1 {
		t.Fatalf("Len = %d want 1", n)
	}
	all, _ := kh.GetAll(ctx, id)
	if all["1001"] != "entry" {
		t.Fatalf("GetAll = %v", all)
	}
	_ = kh.Del(ctx, id, "1001")
	if n, _ := kh.Len(ctx, id); n != 0 {
		t.Fatalf("Len after Del = %d want 0", n)
	}
}

func TestKeyedHash_ClearByPrefix(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()
	// Maps shape: keyFn embeds the tenant uuid then the field segments.
	kh := NewKeyedHash[string](client, "maps:spawn", func(k string) string { return k })
	_ = kh.Set(ctx, "uuidA:0:1:100:nil", "1", "{}")
	_ = kh.Set(ctx, "uuidA:0:1:200:nil", "1", "{}")
	_ = kh.Set(ctx, "uuidB:0:1:100:nil", "1", "{}")

	// Clear only tenant uuidA's hashes.
	deleted, err := kh.Clear(ctx, "uuidA")
	if err != nil || deleted != 2 {
		t.Fatalf("Clear(uuidA) = %d,%v want 2,nil", deleted, err)
	}
	if n, _ := kh.Len(ctx, "uuidB:0:1:100:nil"); n != 1 {
		t.Fatalf("uuidB hash must survive; Len=%d", n)
	}
	// Clear everything.
	if _, err := kh.Clear(ctx); err != nil {
		t.Fatalf("Clear(all): %v", err)
	}
	if n, _ := kh.Len(ctx, "uuidB:0:1:100:nil"); n != 0 {
		t.Fatalf("Clear(all) should remove uuidB; Len=%d", n)
	}
}
