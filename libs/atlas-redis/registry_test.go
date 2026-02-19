package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
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
