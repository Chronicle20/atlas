package redis

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTenantCounter_SetAndTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if err := c.Set(ctx, tm, "42", 1000, 35*time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	rk := tenantEntityKey("test-counter", tm, "42")
	if got := mr.TTL(rk); got != 35*time.Minute {
		t.Fatalf("TTL = %v, want 35m", got)
	}
	newV, existed, err := c.DecrByIfExists(ctx, tm, "42", 100, 35*time.Minute)
	if err != nil {
		t.Fatalf("DecrByIfExists: %v", err)
	}
	if !existed || newV != 900 {
		t.Fatalf("DecrByIfExists = (%d, %v), want (900, true)", newV, existed)
	}
}

func TestTenantCounter_DecrRefreshesTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if err := c.Set(ctx, tm, "1", 500, 10*time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	mr.FastForward(9 * time.Minute)
	if _, _, err := c.DecrByIfExists(ctx, tm, "1", 10, 10*time.Minute); err != nil {
		t.Fatalf("DecrByIfExists: %v", err)
	}
	rk := tenantEntityKey("test-counter", tm, "1")
	if got := mr.TTL(rk); got != 10*time.Minute {
		t.Fatalf("TTL after decr = %v, want refreshed 10m", got)
	}
}

func TestTenantCounter_DecrMissingKeyDoesNotCreate(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")

	newV, existed, err := c.DecrByIfExists(context.Background(), tm, "77", 100, time.Minute)
	if err != nil {
		t.Fatalf("DecrByIfExists: %v", err)
	}
	if existed || newV != 0 {
		t.Fatalf("DecrByIfExists = (%d, %v), want (0, false)", newV, existed)
	}
	if mr.Exists(tenantEntityKey("test-counter", tm, "77")) {
		t.Fatal("missing key was created by DecrByIfExists")
	}
}

func TestTenantCounter_RemoveIdempotent(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if err := c.Set(ctx, tm, "9", 5, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Remove(ctx, tm, "9"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := c.Remove(ctx, tm, "9"); err != nil {
		t.Fatalf("Remove (missing): %v", err)
	}
	if _, existed, _ := c.DecrByIfExists(ctx, tm, "9", 1, time.Minute); existed {
		t.Fatal("key still exists after Remove")
	}
}

// Exactly one concurrent caller observes the zero crossing
// (newValue <= 0 && newValue+delta > 0), and no decrement is lost.
func TestTenantCounter_ConcurrentDecrExactlyOneCrossing(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	const workers = 10
	const delta = int64(100)
	if err := c.Set(ctx, tm, "ship", 500, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var wg sync.WaitGroup
	crossings := make(chan int64, workers)
	var finalMu sync.Mutex
	var lastValues []int64
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newV, existed, err := c.DecrByIfExists(ctx, tm, "ship", delta, time.Minute)
			if err != nil || !existed {
				t.Errorf("DecrByIfExists = (%d, %v, %v)", newV, existed, err)
				return
			}
			finalMu.Lock()
			lastValues = append(lastValues, newV)
			finalMu.Unlock()
			if newV <= 0 && newV+delta > 0 {
				crossings <- newV
			}
		}()
	}
	wg.Wait()
	close(crossings)

	var crossed int
	for range crossings {
		crossed++
	}
	if crossed != 1 {
		t.Fatalf("crossings = %d, want exactly 1", crossed)
	}
	var min int64
	for _, v := range lastValues {
		if v < min {
			min = v
		}
	}
	if want := int64(500) - int64(workers)*delta; min != want {
		t.Fatalf("lowest observed value = %d, want %d (no decrement lost)", min, want)
	}
}
