// libs/atlas-redis/set_test.go
package redis

import (
	"context"
	"testing"
)

func TestSet_AddMembersIsMemberSize(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("a3f7")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	s := NewSet(client, "drops:all")

	if err := s.Add(ctx, "x", "y"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Key-format assertion: must be <env>:atlas:<namespace>.
	if !mr.Exists("a3f7:atlas:drops:all") {
		t.Fatalf("expected key a3f7:atlas:drops:all to exist; keys=%v", mr.Keys())
	}
	ok, err := s.IsMember(ctx, "x")
	if err != nil || !ok {
		t.Fatalf("IsMember x = %v,%v want true,nil", ok, err)
	}
	n, err := s.Size(ctx)
	if err != nil || n != 2 {
		t.Fatalf("Size = %d,%v want 2,nil", n, err)
	}
	members, err := s.Members(ctx)
	if err != nil || len(members) != 2 {
		t.Fatalf("Members = %v,%v want len 2", members, err)
	}
	if err := s.Remove(ctx, "x"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if n, _ := s.Size(ctx); n != 1 {
		t.Fatalf("Size after remove = %d want 1", n)
	}
}

func TestTenantSet_PerTenantKeyAndIsolation(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	s := NewTenantSet(client, "transport:channels")
	t1 := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
	t2 := makeTenant("00000000-0000-0000-0000-000000000002", "GMS", 83, 1)

	if err := s.Add(ctx, t1, "0:1"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	wantKey := "atlas:transport:channels:" + TenantKey(t1)
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}
	if m2, _ := s.Members(ctx, t2); len(m2) != 0 {
		t.Fatalf("t2 must not see t1 members: %v", m2)
	}
	if m1, _ := s.Members(ctx, t1); len(m1) != 1 || m1[0] != "0:1" {
		t.Fatalf("t1 members = %v want [0:1]", m1)
	}
}
