package redis

import (
	"context"
	"strconv"
	"testing"
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
