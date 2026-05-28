package redis

import (
	"context"
	"testing"
)

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
