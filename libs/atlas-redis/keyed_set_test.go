package redis

import (
	"context"
	"sort"
	"testing"
)

func TestKeyedSet_KeyFormatAndMembers(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	s := NewKeyedSet[string](client, "monster-map", func(k string) string { return k })

	if err := s.Add(ctx, "kA", "m1", "m2"); err != nil {
		t.Fatalf("Add kA: %v", err)
	}

	// Verify key format: atlas:<namespace>:<k>
	wantKey := "atlas:monster-map:kA"
	if !mr.Exists(wantKey) {
		t.Fatalf("expected Redis key %q; got keys=%v", wantKey, mr.Keys())
	}

	members, err := s.Members(ctx, "kA")
	if err != nil {
		t.Fatalf("Members: %v", err)
	}
	sort.Strings(members)
	if len(members) != 2 || members[0] != "m1" || members[1] != "m2" {
		t.Fatalf("Members = %v, want [m1 m2]", members)
	}
}

func TestKeyedSet_IsMember(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()
	s := NewKeyedSet[string](client, "monster-map", func(k string) string { return k })

	// Not a member before Add.
	ok, err := s.IsMember(ctx, "kA", "m1")
	if err != nil {
		t.Fatalf("IsMember (absent): %v", err)
	}
	if ok {
		t.Fatal("expected IsMember false before Add")
	}

	if err := s.Add(ctx, "kA", "m1"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	ok, err = s.IsMember(ctx, "kA", "m1")
	if err != nil {
		t.Fatalf("IsMember (present): %v", err)
	}
	if !ok {
		t.Fatal("expected IsMember true after Add")
	}

	ok, err = s.IsMember(ctx, "kA", "m99")
	if err != nil {
		t.Fatalf("IsMember (absent member): %v", err)
	}
	if ok {
		t.Fatal("expected IsMember false for absent member")
	}
}

func TestKeyedSet_KeyIsolationAndClear(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()
	s := NewKeyedSet[string](client, "monster-map", func(k string) string { return k })

	_ = s.Add(ctx, "kA", "m1", "m2")
	_ = s.Add(ctx, "kB", "m3")

	// kA and kB are independent.
	membersB, _ := s.Members(ctx, "kB")
	if len(membersB) != 1 || membersB[0] != "m3" {
		t.Fatalf("Members kB = %v, want [m3]", membersB)
	}

	// Clear kA, kB should be unaffected.
	if err := s.Clear(ctx, "kA"); err != nil {
		t.Fatalf("Clear kA: %v", err)
	}
	membersA, _ := s.Members(ctx, "kA")
	if len(membersA) != 0 {
		t.Fatalf("Members kA after Clear = %v, want empty", membersA)
	}
	membersB, _ = s.Members(ctx, "kB")
	if len(membersB) != 1 {
		t.Fatalf("Members kB after kA Clear = %v, want [m3]", membersB)
	}
}

func TestTenantKeyedSet_IsMember(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()
	s := NewTenantKeyedSet[string](client, "drops:map", func(k string) string { return k })
	tm := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)

	// Member not in set before Add.
	ok, err := s.IsMember(ctx, tm, "0:1:100:nil", "42")
	if err != nil {
		t.Fatalf("IsMember (absent key) error: %v", err)
	}
	if ok {
		t.Fatal("expected IsMember false for absent key")
	}

	if err := s.Add(ctx, tm, "0:1:100:nil", "42", "43"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// "42" is a member.
	ok, err = s.IsMember(ctx, tm, "0:1:100:nil", "42")
	if err != nil {
		t.Fatalf("IsMember (present) error: %v", err)
	}
	if !ok {
		t.Fatal("expected IsMember true for added member")
	}

	// "99" is not a member.
	ok, err = s.IsMember(ctx, tm, "0:1:100:nil", "99")
	if err != nil {
		t.Fatalf("IsMember (absent member) error: %v", err)
	}
	if ok {
		t.Fatal("expected IsMember false for absent member")
	}
}

func TestTenantKeyedSet_PerTenantPerKey(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	s := NewTenantKeyedSet[string](client, "drops:map", func(k string) string { return k })
	tm := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)

	if err := s.Add(ctx, tm, "0:1:100:nil", "42", "43"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	wantKey := "atlas:drops:map:" + TenantKey(tm) + ":0:1:100:nil"
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}
	members, _ := s.Members(ctx, tm, "0:1:100:nil")
	if len(members) != 2 {
		t.Fatalf("Members = %v want len 2", members)
	}
	_ = s.Remove(ctx, tm, "0:1:100:nil", "42")
	members, _ = s.Members(ctx, tm, "0:1:100:nil")
	if len(members) != 1 {
		t.Fatalf("Members after remove = %v want len 1", members)
	}
	if err := s.Clear(ctx, tm, "0:1:100:nil"); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	members, _ = s.Members(ctx, tm, "0:1:100:nil")
	if len(members) != 0 {
		t.Fatalf("Members after Clear = %v want empty", members)
	}
}
