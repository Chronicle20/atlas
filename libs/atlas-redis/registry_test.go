package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return client, mr
}

func makeTenant(id string, region string, major uint16, minor uint16) tenant.Model {
	data := fmt.Sprintf(`{"id":"%s","region":"%s","majorVersion":%d,"minorVersion":%d}`, id, region, major, minor)
	var t tenant.Model
	_ = json.Unmarshal([]byte(data), &t)
	return t
}

func testTenant() tenant.Model {
	return makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
}

// --- Registry tests ---

func TestRegistry_PutAndGet(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })

	err := r.Put(ctx, "key1", "value1")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	val, err := r.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "value1" {
		t.Fatalf("expected value1, got %s", val)
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })

	_, err := r.Get(ctx, "missing")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRegistry_Remove(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })

	_ = r.Put(ctx, "key1", "value1")
	err := r.Remove(ctx, "key1")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, err = r.Get(ctx, "key1")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after remove, got %v", err)
	}
}

func TestRegistry_Update(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })

	_ = r.Put(ctx, "key1", "hello")
	result, err := r.Update(ctx, "key1", func(v string) string {
		return v + " world"
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %s", result)
	}

	val, _ := r.Get(ctx, "key1")
	if val != "hello world" {
		t.Fatalf("expected 'hello world' after update, got %s", val)
	}
}

func TestRegistry_Exists(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })

	exists, _ := r.Exists(ctx, "key1")
	if exists {
		t.Fatal("expected key1 to not exist")
	}

	_ = r.Put(ctx, "key1", "value1")
	exists, _ = r.Exists(ctx, "key1")
	if !exists {
		t.Fatal("expected key1 to exist")
	}
}

func TestRegistry_PutWithTTL(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })

	if err := r.PutWithTTL(ctx, "ttlkey", "ttlval", 5*time.Second); err != nil {
		t.Fatalf("PutWithTTL failed: %v", err)
	}

	val, err := r.Get(ctx, "ttlkey")
	if err != nil {
		t.Fatalf("Get after PutWithTTL failed: %v", err)
	}
	if val != "ttlval" {
		t.Fatalf("expected ttlval, got %s", val)
	}

	// Verify a TTL was set.
	rk := namespacedKey("test", "ttlkey")
	ttlDuration := mr.TTL(rk)
	if ttlDuration <= 0 {
		t.Fatalf("expected positive TTL, got %v", ttlDuration)
	}

	mr.FastForward(6 * time.Second)

	_, err = r.Get(ctx, "ttlkey")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after TTL expiry, got %v", err)
	}
}

func TestRegistry_GetAll(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })

	if err := r.Put(ctx, "a", "val-a"); err != nil {
		t.Fatalf("Put a: %v", err)
	}
	if err := r.Put(ctx, "b", "val-b"); err != nil {
		t.Fatalf("Put b: %v", err)
	}
	if err := r.Put(ctx, "c", "val-c"); err != nil {
		t.Fatalf("Put c: %v", err)
	}

	vals, err := r.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(vals) != 3 {
		t.Fatalf("expected 3 values, got %d: %v", len(vals), vals)
	}

	// Verify expected key format: atlas:<ns>:<k>
	wantKey := "atlas:test:a"
	exists, err := r.Exists(ctx, "a")
	if err != nil || !exists {
		t.Fatalf("expected key %q to exist", wantKey)
	}
	actualKey := namespacedKey("test", "a")
	if actualKey != wantKey {
		t.Fatalf("expected key format %q, got %q", wantKey, actualKey)
	}
}

func TestRegistry_GetAll_SkipsInternalKeys(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()

	// Use a keyFn that can produce _-prefixed suffixes for internal keys.
	r := NewRegistry[string, string](client, "test:internal", func(k string) string { return k })

	_ = r.Put(ctx, "normal", "val")
	// Directly insert an internal key using the raw client to simulate internal state.
	internalKey := namespacedKey("test:internal", "_expiry")
	_ = r.client.Set(ctx, internalKey, `"internal"`, 0).Err()

	vals, err := r.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	// Only "normal" should be returned; "_expiry" is internal.
	if len(vals) != 1 {
		t.Fatalf("expected 1 value (internal key skipped), got %d: %v", len(vals), vals)
	}
}

func TestRegistry_Clear(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test:clear", func(k string) string { return k })

	for i := 0; i < 5; i++ {
		if err := r.Put(ctx, fmt.Sprintf("k%d", i), "v"); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	deleted, err := r.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if deleted != 5 {
		t.Fatalf("expected 5 deleted, got %d", deleted)
	}

	vals, err := r.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll after Clear failed: %v", err)
	}
	if len(vals) != 0 {
		t.Fatalf("expected empty after Clear, got %d values", len(vals))
	}
}

func TestRegistry_ClearByPrefix(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "monster", func(k string) string { return k })

	// Put keys with three different prefixes.
	for _, k := range []string{"100:1", "100:2", "200:1", "1000:9"} {
		if err := r.Put(ctx, k, "v"); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}

	// ClearByPrefix("100:") should delete "100:1" and "100:2" but NOT "200:1" or "1000:9".
	deleted, err := r.ClearByPrefix(ctx, "100:")
	if err != nil {
		t.Fatalf("ClearByPrefix: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deleted, got %d", deleted)
	}

	// "100:1" and "100:2" must be gone.
	for _, k := range []string{"100:1", "100:2"} {
		if exists, _ := r.Exists(ctx, k); exists {
			t.Fatalf("key %q should have been deleted by ClearByPrefix", k)
		}
	}

	// "200:1" must still exist.
	if exists, _ := r.Exists(ctx, "200:1"); !exists {
		t.Fatal("key \"200:1\" should NOT have been deleted by ClearByPrefix(\"100:\")")
	}

	// "1000:9" must still exist — trailing-delimiter avoids matching a longer prefix.
	if exists, _ := r.Exists(ctx, "1000:9"); !exists {
		t.Fatal("key \"1000:9\" should NOT have been deleted by ClearByPrefix(\"100:\") — trailing delimiter prevents cross-match")
	}
}

// --- TenantRegistry tests ---

func TestTenantRegistry_PutAndGet(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantRegistry[uint32, string](client, "chair", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	})

	err := r.Put(ctx, tn, 42, "sitting")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	val, err := r.Get(ctx, tn, 42)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "sitting" {
		t.Fatalf("expected 'sitting', got %s", val)
	}
}

func TestTenantRegistry_TenantIsolation(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	t1 := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
	t2 := makeTenant("00000000-0000-0000-0000-000000000002", "EMS", 83, 1)

	r := NewTenantRegistry[uint32, string](client, "chair", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	})

	_ = r.Put(ctx, t1, 42, "gms-chair")
	_ = r.Put(ctx, t2, 42, "ems-chair")

	v1, _ := r.Get(ctx, t1, 42)
	v2, _ := r.Get(ctx, t2, 42)

	if v1 != "gms-chair" {
		t.Fatalf("expected gms-chair, got %s", v1)
	}
	if v2 != "ems-chair" {
		t.Fatalf("expected ems-chair, got %s", v2)
	}
}

func TestTenantRegistry_GetAllValues(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantRegistry[uint32, string](client, "test", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	})

	_ = r.Put(ctx, tn, 1, "a")
	_ = r.Put(ctx, tn, 2, "b")
	_ = r.Put(ctx, tn, 3, "c")

	vals, err := r.GetAllValues(ctx, tn)
	if err != nil {
		t.Fatalf("GetAllValues failed: %v", err)
	}
	if len(vals) != 3 {
		t.Fatalf("expected 3 values, got %d", len(vals))
	}
}

func TestTenantRegistry_Remove(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantRegistry[uint32, string](client, "test", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	})

	_ = r.Put(ctx, tn, 42, "value")
	_ = r.Remove(ctx, tn, 42)

	_, err := r.Get(ctx, tn, 42)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTenantRegistry_Update(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	type Counter struct {
		Value int `json:"value"`
	}

	r := NewTenantRegistry[uint32, Counter](client, "test", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	})

	_ = r.Put(ctx, tn, 1, Counter{Value: 10})
	result, err := r.Update(ctx, tn, 1, func(c Counter) Counter {
		c.Value += 5
		return c
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if result.Value != 15 {
		t.Fatalf("expected 15, got %d", result.Value)
	}
}

func TestTenantRegistry_PutWithTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantRegistry[uint32, string](client, "test", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	})

	_ = r.PutWithTTL(ctx, tn, 42, "temporary", 5*time.Second)

	val, err := r.Get(ctx, tn, 42)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "temporary" {
		t.Fatalf("expected temporary, got %s", val)
	}

	mr.FastForward(6 * time.Second)

	_, err = r.Get(ctx, tn, 42)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after TTL expiry, got %v", err)
	}
}

// --- IDGenerator tests ---

func TestIDGenerator_NextID(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	gen := NewIDGenerator(client, "party")

	id1, err := gen.NextID(ctx, tn)
	if err != nil {
		t.Fatalf("NextID failed: %v", err)
	}
	if id1 != 1000000000 {
		t.Fatalf("expected 1000000000, got %d", id1)
	}

	id2, _ := gen.NextID(ctx, tn)
	if id2 != 1000000001 {
		t.Fatalf("expected 1000000001, got %d", id2)
	}

	id3, _ := gen.NextID(ctx, tn)
	if id3 != 1000000002 {
		t.Fatalf("expected 1000000002, got %d", id3)
	}
}

func TestIDGenerator_TenantIsolation(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	t1 := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
	t2 := makeTenant("00000000-0000-0000-0000-000000000002", "EMS", 83, 1)

	gen := NewIDGenerator(client, "party")

	id1, _ := gen.NextID(ctx, t1)
	id2, _ := gen.NextID(ctx, t2)

	if id1 != id2 {
		t.Fatalf("expected same starting ID for different tenants, got %d and %d", id1, id2)
	}
}

func TestGlobalIDGenerator_NextID(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	gen := NewGlobalIDGenerator(client, "drop", 1)

	id1, _ := gen.NextID(ctx)
	id2, _ := gen.NextID(ctx)
	id3, _ := gen.NextID(ctx)

	if id1 != 1 || id2 != 2 || id3 != 3 {
		t.Fatalf("expected 1,2,3 got %d,%d,%d", id1, id2, id3)
	}
}

// --- TTLRegistry tests ---

func TestTTLRegistry_PutAndPopExpired(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	now := time.Now()
	clock := now

	r := NewTTLRegistry[uint32, string](client, "expression", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, 5*time.Second)
	r.SetNowFunc(func() time.Time { return clock })

	_ = r.Put(ctx, tn, 1, "smile")
	_ = r.Put(ctx, tn, 2, "cry")

	// Not yet expired.
	expired, _ := r.PopExpired(ctx, tn)
	if len(expired) != 0 {
		t.Fatalf("expected 0 expired, got %d", len(expired))
	}

	// Fast-forward past TTL.
	clock = now.Add(6 * time.Second)

	expired, err := r.PopExpired(ctx, tn)
	if err != nil {
		t.Fatalf("PopExpired failed: %v", err)
	}
	if len(expired) != 2 {
		t.Fatalf("expected 2 expired, got %d", len(expired))
	}

	// Should be empty now.
	expired, _ = r.PopExpired(ctx, tn)
	if len(expired) != 0 {
		t.Fatalf("expected 0 expired after pop, got %d", len(expired))
	}
}

func TestTTLRegistry_Remove(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTTLRegistry[uint32, string](client, "expression", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, 5*time.Second)

	_ = r.Put(ctx, tn, 1, "smile")
	err := r.Remove(ctx, tn, 1)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, err = r.TenantRegistry.Get(ctx, tn, 1)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- Index tests ---

func TestIndex_AddAndLookup(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	idx := NewUint32Index(client, "party", "char")

	_ = idx.Add(ctx, tn, 42, 1000000001)
	_ = idx.Add(ctx, tn, 43, 1000000001)

	members, err := idx.Lookup(ctx, tn, 42)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if len(members) != 1 || members[0] != 1000000001 {
		t.Fatalf("expected [1000000001], got %v", members)
	}
}

func TestIndex_LookupOne(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	idx := NewUint32Index(client, "party", "char")

	_ = idx.Add(ctx, tn, 42, 1000000001)

	partyId, err := idx.LookupOne(ctx, tn, 42)
	if err != nil {
		t.Fatalf("LookupOne failed: %v", err)
	}
	if partyId != 1000000001 {
		t.Fatalf("expected 1000000001, got %d", partyId)
	}
}

func TestIndex_Remove(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	idx := NewUint32Index(client, "party", "char")

	_ = idx.Add(ctx, tn, 42, 1000000001)
	_ = idx.Remove(ctx, tn, 42, 1000000001)

	_, err := idx.LookupOne(ctx, tn, 42)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- Lock tests ---

func TestLock_AcquireAndRelease(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	lock := NewLock(client, "inventory")

	acquired, err := lock.Acquire(ctx, "char:42:equip")
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if !acquired {
		t.Fatal("expected lock to be acquired")
	}

	// Second acquire should fail.
	acquired2, _ := lock.Acquire(ctx, "char:42:equip")
	if acquired2 {
		t.Fatal("expected second acquire to fail")
	}

	// Release and re-acquire.
	_ = lock.Release(ctx, "char:42:equip")
	acquired3, _ := lock.Acquire(ctx, "char:42:equip")
	if !acquired3 {
		t.Fatal("expected re-acquire to succeed after release")
	}
}

func TestLock_AutoExpiry(t *testing.T) {
	client, mr := setupTestRedis(t)
	ctx := context.Background()

	lock := NewLockWithTTL(client, "inventory", 5*time.Second)

	_, _ = lock.Acquire(ctx, "char:42:equip")

	mr.FastForward(6 * time.Second)

	acquired, _ := lock.Acquire(ctx, "char:42:equip")
	if !acquired {
		t.Fatal("expected acquire to succeed after auto-expiry")
	}
}

func TestLock_AcquireWithToken_AndReleaseToken(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("a3f7")

	client, mr := setupTestRedis(t)
	ctx := context.Background()

	lock := NewLockWithTTL(client, "inventory", 30*time.Second)
	ttl := 30 * time.Second

	// tokA acquires successfully.
	ok, err := lock.AcquireWithToken(ctx, "char:99:equip", "tokA", ttl)
	if err != nil {
		t.Fatalf("AcquireWithToken tokA: %v", err)
	}
	if !ok {
		t.Fatal("expected tokA to acquire the lock")
	}

	// tokB cannot acquire while tokA holds it.
	ok2, err := lock.AcquireWithToken(ctx, "char:99:equip", "tokB", ttl)
	if err != nil {
		t.Fatalf("AcquireWithToken tokB: %v", err)
	}
	if ok2 {
		t.Fatal("expected tokB to fail acquiring the lock held by tokA")
	}

	// Verify the key exists with the correct namespaced format.
	wantKey := "a3f7:atlas:inventory:_lock:char:99:equip"
	if !mr.Exists(wantKey) {
		t.Fatalf("expected lock key %q; keys=%v", wantKey, mr.Keys())
	}

	// tokB cannot release a lock it doesn't hold.
	released, err := lock.ReleaseToken(ctx, "char:99:equip", "tokB")
	if err != nil {
		t.Fatalf("ReleaseToken tokB: %v", err)
	}
	if released {
		t.Fatal("expected tokB ReleaseToken to return false (not the holder)")
	}

	// tokA releases successfully.
	released2, err := lock.ReleaseToken(ctx, "char:99:equip", "tokA")
	if err != nil {
		t.Fatalf("ReleaseToken tokA: %v", err)
	}
	if !released2 {
		t.Fatal("expected tokA ReleaseToken to return true")
	}

	// Now tokB can acquire the free lock.
	ok3, err := lock.AcquireWithToken(ctx, "char:99:equip", "tokB", ttl)
	if err != nil {
		t.Fatalf("AcquireWithToken tokB after release: %v", err)
	}
	if !ok3 {
		t.Fatal("expected tokB to acquire the now-free lock")
	}
}

func TestLock_ForceAcquire_OverridesHolder(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	lock := NewLockWithTTL(client, "inventory", 30*time.Second)
	ttl := 30 * time.Second

	// tokA acquires normally.
	ok, err := lock.AcquireWithToken(ctx, "char:77:use", "tokA", ttl)
	if err != nil || !ok {
		t.Fatalf("AcquireWithToken tokA: ok=%v err=%v", ok, err)
	}

	// tokB force-acquires, overwriting tokA.
	if err := lock.ForceAcquire(ctx, "char:77:use", "tokB", ttl); err != nil {
		t.Fatalf("ForceAcquire tokB: %v", err)
	}

	// tokA can no longer release (it's no longer the holder).
	released, err := lock.ReleaseToken(ctx, "char:77:use", "tokA")
	if err != nil {
		t.Fatalf("ReleaseToken tokA after ForceAcquire: %v", err)
	}
	if released {
		t.Fatal("expected tokA ReleaseToken to return false after force-acquire by tokB")
	}

	// tokB can release.
	released2, err := lock.ReleaseToken(ctx, "char:77:use", "tokB")
	if err != nil {
		t.Fatalf("ReleaseToken tokB: %v", err)
	}
	if !released2 {
		t.Fatal("expected tokB ReleaseToken to return true")
	}
}

func TestLock_ReleaseToken_AbsentKey(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	lock := NewLockWithTTL(client, "inventory", 30*time.Second)

	// ReleaseToken on a key that was never acquired returns (false, nil).
	released, err := lock.ReleaseToken(ctx, "char:nonexistent:equip", "anyToken")
	if err != nil {
		t.Fatalf("ReleaseToken absent key: unexpected error %v", err)
	}
	if released {
		t.Fatal("expected ReleaseToken on absent key to return false")
	}
}

// --- Key tests ---

func TestTenantKey(t *testing.T) {
	tn := testTenant()
	key := TenantKey(tn)
	expected := "00000000-0000-0000-0000-000000000001:GMS:83.1"
	if key != expected {
		t.Fatalf("expected %s, got %s", expected, key)
	}
}

func TestCompositeKey(t *testing.T) {
	key := CompositeKey("world1", "channel2", fmt.Sprintf("%d", 100000000))
	expected := "world1:channel2:100000000"
	if key != expected {
		t.Fatalf("expected %s, got %s", expected, key)
	}
}
