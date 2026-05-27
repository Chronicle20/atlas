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
