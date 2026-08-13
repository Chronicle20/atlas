package redis

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func TestTenantKeyedSortedSet_OrderedAddRemoveCountClear(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()

	s := NewTenantKeyedSortedSet[uint32](client, "merchant:shop-visitors", func(id uint32) string {
		return strconv.FormatUint(uint64(id), 10)
	})
	tm := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)

	const shopId uint32 = 42
	shopIdStr := strconv.FormatUint(uint64(shopId), 10)

	// Add three members with scores out of insertion order to prove score-based ordering.
	if err := s.Add(ctx, tm, shopId, "c", 3); err != nil {
		t.Fatalf("Add c: %v", err)
	}
	if err := s.Add(ctx, tm, shopId, "a", 1); err != nil {
		t.Fatalf("Add a: %v", err)
	}
	if err := s.Add(ctx, tm, shopId, "b", 2); err != nil {
		t.Fatalf("Add b: %v", err)
	}

	// Assert Redis key format.
	wantKey := "atlas:merchant:shop-visitors:" + TenantKey(tm) + ":" + shopIdStr
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}

	// Range returns members in score-ascending order (a, b, c — NOT insertion order c, a, b).
	members, err := s.Range(ctx, tm, shopId)
	if err != nil {
		t.Fatalf("Range: %v", err)
	}
	if len(members) != 3 || members[0] != "a" || members[1] != "b" || members[2] != "c" {
		t.Fatalf("Range = %v want [a b c] (score-ascending)", members)
	}

	// Count == 3.
	n, err := s.Count(ctx, tm, shopId)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 3 {
		t.Fatalf("Count = %d want 3", n)
	}

	// Remove "b"; Range should be [a, c]; Count == 2.
	if err := s.Remove(ctx, tm, shopId, "b"); err != nil {
		t.Fatalf("Remove b: %v", err)
	}
	members, err = s.Range(ctx, tm, shopId)
	if err != nil {
		t.Fatalf("Range after Remove: %v", err)
	}
	if len(members) != 2 || members[0] != "a" || members[1] != "c" {
		t.Fatalf("Range after Remove = %v want [a c]", members)
	}
	if n, _ = s.Count(ctx, tm, shopId); n != 2 {
		t.Fatalf("Count after Remove = %d want 2", n)
	}

	// A different shopId key for the same tenant is independent (Count == 0).
	const otherShopId uint32 = 99
	if n, err = s.Count(ctx, tm, otherShopId); err != nil {
		t.Fatalf("Count otherShopId: %v", err)
	}
	if n != 0 {
		t.Fatalf("Count otherShopId = %d want 0 (independent key)", n)
	}

	// Clear; Count == 0, Range empty.
	if err := s.Clear(ctx, tm, shopId); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if n, _ = s.Count(ctx, tm, shopId); n != 0 {
		t.Fatalf("Count after Clear = %d want 0", n)
	}
	members, err = s.Range(ctx, tm, shopId)
	if err != nil {
		t.Fatalf("Range after Clear: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("Range after Clear = %v want empty", members)
	}
}

func TestTenantKeyedSortedSet_AddBounded(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()

	s := NewTenantKeyedSortedSet[uint32](client, "chat:recent", func(id uint32) string {
		return strconv.FormatUint(uint64(id), 10)
	})
	tm := makeTenant("00000000-0000-0000-0000-000000000002", "GMS", 83, 1)
	const characterId uint32 = 7
	ttl := 900 * time.Second

	// Age-based pruning: a member whose score falls below minScore is dropped.
	if err := s.AddBounded(ctx, tm, characterId, "old", 1000, 0, 10, ttl); err != nil {
		t.Fatalf("AddBounded old: %v", err)
	}
	if err := s.AddBounded(ctx, tm, characterId, "new", 2000, 1500, 10, ttl); err != nil {
		t.Fatalf("AddBounded new: %v", err)
	}
	members, err := s.Range(ctx, tm, characterId)
	if err != nil {
		t.Fatalf("Range: %v", err)
	}
	if len(members) != 1 || members[0] != "new" {
		t.Fatalf("age prune: got %v want [new]", members)
	}

	// Count-based trimming: only the newest maxCount members survive.
	for i := 0; i < 5; i++ {
		member := "m" + strconv.Itoa(i)
		if err := s.AddBounded(ctx, tm, characterId, member, float64(3000+i), 0, 3, ttl); err != nil {
			t.Fatalf("AddBounded %s: %v", member, err)
		}
	}
	members, _ = s.Range(ctx, tm, characterId)
	if len(members) != 3 {
		t.Fatalf("count trim: got %d members (%v), want 3", len(members), members)
	}
	if members[0] != "m2" || members[2] != "m4" {
		t.Fatalf("count trim kept wrong members: %v", members)
	}

	// TTL refresh: the key expires after the window.
	wantKey := "atlas:chat:recent:" + TenantKey(tm) + ":7"
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}
	mr.FastForward(ttl + time.Second)
	if mr.Exists(wantKey) {
		t.Fatal("expected key to expire after ttl")
	}
}
