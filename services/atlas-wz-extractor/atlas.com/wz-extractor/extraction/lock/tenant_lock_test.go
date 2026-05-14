package lock

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	c := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return c, mr
}

func TestTenantLock_AcquireRelease(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestClient(t)
	tl := NewTenantLock(c, time.Minute)

	ok1, err := tl.Acquire(ctx, "key1", "owner-A")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok1 {
		t.Fatalf("expected first Acquire to succeed")
	}

	ok2, err := tl.Acquire(ctx, "key1", "owner-B")
	if err != nil {
		t.Fatalf("Acquire 2: %v", err)
	}
	if ok2 {
		t.Fatalf("expected second Acquire to fail (held)")
	}

	if err := tl.Release(ctx, "key1", "owner-A"); err != nil {
		t.Fatalf("Release: %v", err)
	}

	ok3, err := tl.Acquire(ctx, "key1", "owner-B")
	if err != nil {
		t.Fatalf("Acquire 3: %v", err)
	}
	if !ok3 {
		t.Fatalf("expected re-Acquire after Release to succeed")
	}
}

func TestTenantLock_ReleaseOnlyOwner(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestClient(t)
	tl := NewTenantLock(c, time.Minute)

	if _, err := tl.Acquire(ctx, "key2", "owner-A"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Different owner attempts release; lock must remain held.
	if err := tl.Release(ctx, "key2", "owner-B"); err != nil {
		t.Fatalf("Release wrong owner: %v", err)
	}

	ok, err := tl.Acquire(ctx, "key2", "owner-C")
	if err != nil {
		t.Fatalf("Acquire after wrong-owner release: %v", err)
	}
	if ok {
		t.Fatalf("lock should still be held after wrong-owner Release")
	}
}

func TestTenantLock_RefreshExtendsTTL(t *testing.T) {
	ctx := context.Background()
	c, mr := newTestClient(t)
	tl := NewTenantLock(c, 10*time.Second)

	if _, err := tl.Acquire(ctx, "key3", "owner-A"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Advance miniredis time so the original TTL would have nearly expired.
	mr.FastForward(8 * time.Second)
	if err := tl.Refresh(ctx, "key3", "owner-A"); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	mr.FastForward(8 * time.Second)
	// TTL was reset; lock should still be held.
	ok, err := tl.Acquire(ctx, "key3", "owner-B")
	if err != nil {
		t.Fatalf("Acquire after refresh: %v", err)
	}
	if ok {
		t.Fatalf("expected lock still held after Refresh")
	}
}
