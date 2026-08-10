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

func TestTenantCounter_InitIfMissingAndDecrBy_InitializesWhenAbsent(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	newV, err := c.InitIfMissingAndDecrBy(ctx, tm, "ship", 8800, 300, time.Minute)
	if err != nil {
		t.Fatalf("InitIfMissingAndDecrBy: %v", err)
	}
	if newV != 8500 {
		t.Fatalf("InitIfMissingAndDecrBy = %d, want 8500 (8800 seeded then -300)", newV)
	}
}

func TestTenantCounter_InitIfMissingAndDecrBy_DoesNotReinitWhenPresent(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if err := c.Set(ctx, tm, "ship", 500, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// initial=9999 must be IGNORED — the key already exists at 500.
	newV, err := c.InitIfMissingAndDecrBy(ctx, tm, "ship", 9999, 100, time.Minute)
	if err != nil {
		t.Fatalf("InitIfMissingAndDecrBy: %v", err)
	}
	if newV != 400 {
		t.Fatalf("InitIfMissingAndDecrBy = %d, want 400 (existing 500 -100, not 9999-100)", newV)
	}
}

func TestTenantCounter_InitIfMissingAndDecrBy_RefreshesTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if _, err := c.InitIfMissingAndDecrBy(ctx, tm, "ship", 1000, 100, 35*time.Minute); err != nil {
		t.Fatalf("InitIfMissingAndDecrBy: %v", err)
	}
	rk := tenantEntityKey("test-counter", tm, "ship")
	if got := mr.TTL(rk); got != 35*time.Minute {
		t.Fatalf("TTL = %v, want 35m", got)
	}
}

func TestTenantCounter_IncrWithTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-incr")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	// The FIRST increment must create the key AND set the TTL. A plain INCR
	// leaves a fresh key with no expiry, which makes a rate-limit window
	// permanent and locks the account out forever.
	v, err := c.IncrWithTTL(ctx, tm, "acct-1", time.Minute)
	if err != nil {
		t.Fatalf("first incr: %v", err)
	}
	if v != 1 {
		t.Errorf("first incr = %d, want 1", v)
	}
	if got := mr.TTL(c.entityKey(tm, "acct-1")); got <= 0 {
		t.Fatalf("TTL = %v, want a positive expiry on the first increment", got)
	}

	v, err = c.IncrWithTTL(ctx, tm, "acct-1", time.Minute)
	if err != nil {
		t.Fatalf("second incr: %v", err)
	}
	if v != 2 {
		t.Errorf("second incr = %d, want 2", v)
	}
}

func TestTenantCounter_IncrWithTTLDoesNotSlideTheWindow(t *testing.T) {
	// A sliding window lets an attacker keep the key alive forever by
	// attempting just often enough. The TTL is set ONCE, on creation.
	client, mr := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-incr-slide")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if _, err := c.IncrWithTTL(ctx, tm, "acct-2", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	first := mr.TTL(c.entityKey(tm, "acct-2"))
	if _, err := c.IncrWithTTL(ctx, tm, "acct-2", 10*time.Hour); err != nil {
		t.Fatal(err)
	}
	second := mr.TTL(c.entityKey(tm, "acct-2"))
	if second > first {
		t.Errorf("TTL grew from %v to %v — the window slid", first, second)
	}
}

func TestTenantCounter_IncrWithTTLIsTenantScoped(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-incr-tenant")
	tm := newTestTenant(t, "GMS")
	other := newTestTenant(t, "GMS")
	ctx := context.Background()

	if _, err := c.IncrWithTTL(ctx, tm, "acct-3", time.Minute); err != nil {
		t.Fatal(err)
	}
	v, err := c.IncrWithTTL(ctx, other, "acct-3", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Errorf("other tenant's counter = %d, want 1 — counters must not share a key", v)
	}
}

func TestTenantCounter_GetMissingKeyIsZero(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-get")
	tm := newTestTenant(t, "GMS")

	v, err := c.Get(context.Background(), tm, "absent")
	if err != nil {
		t.Fatalf("Get (missing): %v", err)
	}
	if v != 0 {
		t.Errorf("Get (missing) = %d, want 0", v)
	}
}

func TestTenantCounter_GetReadsWithoutMutating(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-get")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	if _, err := c.IncrWithTTL(ctx, tm, "acct-4", time.Minute); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		v, err := c.Get(ctx, tm, "acct-4")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if v != 1 {
			t.Fatalf("Get #%d = %d, want 1 — Get must not mutate the counter", i, v)
		}
	}
}

// Concurrent InitIfMissingAndDecrBy calls racing an absent key must seed the
// counter exactly once and lose no decrement — the failure mode a plain
// "read-missing then Set(full-damage)" pair exhibits under concurrency.
func TestTenantCounter_InitIfMissingAndDecrBy_ConcurrentNoLostDecrement(t *testing.T) {
	client, _ := setupTestRedis(t)
	defer func() { _ = client.Close() }()
	c := NewTenantCounter(client, "test-counter")
	tm := newTestTenant(t, "GMS")
	ctx := context.Background()

	const workers = 10
	const delta = int64(100)
	const initial = int64(500)

	var wg sync.WaitGroup
	var finalMu sync.Mutex
	var lastValues []int64
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newV, err := c.InitIfMissingAndDecrBy(ctx, tm, "ship", initial, delta, time.Minute)
			if err != nil {
				t.Errorf("InitIfMissingAndDecrBy: %v", err)
				return
			}
			finalMu.Lock()
			lastValues = append(lastValues, newV)
			finalMu.Unlock()
		}()
	}
	wg.Wait()

	var min int64
	for _, v := range lastValues {
		if v < min {
			min = v
		}
	}
	if want := initial - int64(workers)*delta; min != want {
		t.Fatalf("lowest observed value = %d, want %d (initial seeded once, no decrement lost)", min, want)
	}
}
