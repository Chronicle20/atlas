package redis

import (
	"context"
	"strconv"
	"testing"
	"time"
)

// --- CoalescedRegistry tests ---

func TestCoalesced_PutAndGet(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

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

func TestCoalesced_GetNotFound(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	_, err := r.Get(ctx, "missing")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCoalesced_PutReadsFromBufferBeforeFlush(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, "key1", "buffered")

	// Value should be readable from buffer before flush.
	val, err := r.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "buffered" {
		t.Fatalf("expected buffered, got %s", val)
	}

	// Redis should NOT have it yet.
	_, err = r.client.Get(ctx, r.redisKey("key1")).Bytes()
	if err == nil {
		t.Fatal("expected key to not be in Redis before flush")
	}
}

func TestCoalesced_FlushWritesToRedis(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, "key1", "value1")
	_ = r.Put(ctx, "key2", "value2")
	r.Flush()

	// Both should now be in Redis.
	data1, err := r.client.Get(ctx, r.redisKey("key1")).Result()
	if err != nil {
		t.Fatalf("expected key1 in Redis after flush, got error: %v", err)
	}
	if data1 != `"value1"` {
		t.Fatalf("expected \"value1\" in Redis, got %s", data1)
	}

	data2, err := r.client.Get(ctx, r.redisKey("key2")).Result()
	if err != nil {
		t.Fatalf("expected key2 in Redis after flush, got error: %v", err)
	}
	if data2 != `"value2"` {
		t.Fatalf("expected \"value2\" in Redis, got %s", data2)
	}
}

func TestCoalesced_WriteCoalescing(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	// Multiple writes to same key before flush — only last value should persist.
	_ = r.Put(ctx, "key1", "first")
	_ = r.Put(ctx, "key1", "second")
	_ = r.Put(ctx, "key1", "third")

	r.Flush()

	val, _ := r.DirectGet(ctx, "key1")
	if val != "third" {
		t.Fatalf("expected third (last write wins), got %s", val)
	}
}

func TestCoalesced_Remove(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, "key1", "value1")
	r.Flush()

	err := r.Remove(ctx, "key1")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Should be gone from local reads immediately.
	_, err = r.Get(ctx, "key1")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after remove, got %v", err)
	}

	// Flush the tombstone to Redis.
	r.Flush()

	// Should be gone from Redis too.
	n, _ := r.client.Exists(ctx, r.redisKey("key1")).Result()
	if n != 0 {
		t.Fatal("expected key1 deleted from Redis after flush")
	}
}

func TestCoalesced_RemoveBeforeFlush(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	// Put and remove in same flush cycle — remove wins.
	_ = r.Put(ctx, "key1", "value1")
	_ = r.Remove(ctx, "key1")

	r.Flush()

	n, _ := r.client.Exists(ctx, r.redisKey("key1")).Result()
	if n != 0 {
		t.Fatal("expected key1 to not exist in Redis (remove after put)")
	}
}

func TestCoalesced_DirectGet(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, "key1", "value1")
	r.Flush()

	val, err := r.DirectGet(ctx, "key1")
	if err != nil {
		t.Fatalf("DirectGet failed: %v", err)
	}
	if val != "value1" {
		t.Fatalf("expected value1, got %s", val)
	}
}

func TestCoalesced_DirectPut(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	err := r.DirectPut(ctx, "key1", "direct")
	if err != nil {
		t.Fatalf("DirectPut failed: %v", err)
	}

	// Should be in Redis immediately.
	data, err := r.client.Get(ctx, r.redisKey("key1")).Result()
	if err != nil {
		t.Fatalf("expected key in Redis after DirectPut: %v", err)
	}
	if data != `"direct"` {
		t.Fatalf("expected \"direct\" in Redis, got %s", data)
	}

	// Should also be in read cache.
	val, _ := r.Get(ctx, "key1")
	if val != "direct" {
		t.Fatalf("expected direct from cache, got %s", val)
	}
}

func TestCoalesced_DirectUpdate(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	type Counter struct {
		Value int `json:"value"`
	}

	r := NewCoalescedRegistry[string, Counter](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.DirectPut(ctx, "c1", Counter{Value: 10})

	result, err := r.DirectUpdate(ctx, "c1", func(c Counter) Counter {
		c.Value += 5
		return c
	})
	if err != nil {
		t.Fatalf("DirectUpdate failed: %v", err)
	}
	if result.Value != 15 {
		t.Fatalf("expected 15, got %d", result.Value)
	}

	// Verify persisted in Redis.
	val, _ := r.DirectGet(ctx, "c1")
	if val.Value != 15 {
		t.Fatalf("expected 15 from Redis, got %d", val.Value)
	}
}

func TestCoalesced_Exists(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	exists, _ := r.Exists(ctx, "key1")
	if exists {
		t.Fatal("expected key1 to not exist")
	}

	_ = r.Put(ctx, "key1", "value1")
	exists, _ = r.Exists(ctx, "key1")
	if !exists {
		t.Fatal("expected key1 to exist after put")
	}

	_ = r.Remove(ctx, "key1")
	exists, _ = r.Exists(ctx, "key1")
	if exists {
		t.Fatal("expected key1 to not exist after remove")
	}
}

func TestCoalesced_ExistsFallsBackToRedis(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	// Put directly into Redis (simulating another instance's write).
	rk := r.redisKey("key1")
	_ = r.client.Set(ctx, rk, `"external"`, 0).Err()

	exists, _ := r.Exists(ctx, "key1")
	if !exists {
		t.Fatal("expected key1 to exist via Redis fallback")
	}
}

func TestCoalesced_ShutdownFlushes(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)

	_ = r.Put(ctx, "key1", "value1")
	r.Shutdown()

	// After shutdown, buffered write should be in Redis.
	data, err := client.Get(ctx, namespacedKey("test", "key1")).Result()
	if err != nil {
		t.Fatalf("expected key in Redis after shutdown flush: %v", err)
	}
	if data != `"value1"` {
		t.Fatalf("expected \"value1\", got %s", data)
	}
}

func TestCoalesced_RefreshPicksUpExternalWrites(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	// Seed the cache by reading a key.
	_ = r.DirectPut(ctx, "key1", "original")

	// Simulate an external write (another instance).
	rk := r.redisKey("key1")
	data, _ := r.marshal("updated-externally")
	_ = r.client.Set(ctx, rk, data, 0).Err()

	// Before refresh, cache still has old value.
	val, _ := r.Get(ctx, "key1")
	if val != "original" {
		t.Fatalf("expected original before refresh, got %s", val)
	}

	// Trigger refresh.
	r.refresh()

	// After refresh, cache should have new value.
	val, _ = r.Get(ctx, "key1")
	if val != "updated-externally" {
		t.Fatalf("expected updated-externally after refresh, got %s", val)
	}
}

func TestCoalesced_RefreshDoesNotOverwritePendingWrites(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	// Seed cache + Redis.
	_ = r.DirectPut(ctx, "key1", "old-redis-value")

	// Buffer a local write (not yet flushed).
	_ = r.Put(ctx, "key1", "pending-local")

	// Refresh should NOT overwrite the pending local value.
	r.refresh()

	val, _ := r.Get(ctx, "key1")
	if val != "pending-local" {
		t.Fatalf("expected pending-local (buffer takes priority), got %s", val)
	}
}

func TestCoalesced_RefreshRemovesDeletedKeys(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	// Seed cache.
	_ = r.DirectPut(ctx, "key1", "value1")

	// Delete from Redis externally.
	_ = r.client.Del(ctx, r.redisKey("key1")).Err()

	// Refresh should remove from cache.
	r.refresh()

	_, err := r.Get(ctx, "key1")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after refresh of deleted key, got %v", err)
	}
}

func TestCoalesced_FlushClearsBuffer(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, "key1", "value1")

	r.writeMu.Lock()
	bufLen := len(r.writeBuf)
	r.writeMu.Unlock()
	if bufLen != 1 {
		t.Fatalf("expected 1 entry in buffer, got %d", bufLen)
	}

	r.Flush()

	r.writeMu.Lock()
	bufLen = len(r.writeBuf)
	r.writeMu.Unlock()
	if bufLen != 0 {
		t.Fatalf("expected 0 entries in buffer after flush, got %d", bufLen)
	}

	// Value should still be readable from cache.
	val, _ := r.Get(ctx, "key1")
	if val != "value1" {
		t.Fatalf("expected value1 from cache after flush, got %s", val)
	}
}

func TestCoalesced_EmptyFlushIsNoop(t *testing.T) {
	client, _ := setupTestRedis(t)

	r := NewCoalescedRegistry[string, string](client, "test", func(k string) string { return k }, time.Hour, time.Hour)
	defer r.Shutdown()

	// Should not panic or error.
	r.Flush()
	r.Flush()
}

// --- TenantCoalescedRegistry tests ---

func TestTenantCoalesced_PutAndGet(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	err := r.Put(ctx, tn, 42, "sitting")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	val, err := r.Get(ctx, tn, 42)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "sitting" {
		t.Fatalf("expected sitting, got %s", val)
	}
}

func TestTenantCoalesced_TenantIsolation(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	t1 := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
	t2 := makeTenant("00000000-0000-0000-0000-000000000002", "EMS", 83, 1)

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, t1, 42, "gms-data")
	_ = r.Put(ctx, t2, 42, "ems-data")

	v1, _ := r.Get(ctx, t1, 42)
	v2, _ := r.Get(ctx, t2, 42)

	if v1 != "gms-data" {
		t.Fatalf("expected gms-data, got %s", v1)
	}
	if v2 != "ems-data" {
		t.Fatalf("expected ems-data, got %s", v2)
	}
}

func TestTenantCoalesced_FlushWritesToRedis(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, tn, 1, "a")
	_ = r.Put(ctx, tn, 2, "b")
	r.Flush()

	rk1 := r.entityKey(tn, 1)
	data, err := r.client.Get(ctx, rk1).Result()
	if err != nil {
		t.Fatalf("expected key in Redis after flush: %v", err)
	}
	if data != `"a"` {
		t.Fatalf("expected \"a\", got %s", data)
	}
}

func TestTenantCoalesced_Remove(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, tn, 42, "value")
	r.Flush()

	_ = r.Remove(ctx, tn, 42)

	_, err := r.Get(ctx, tn, 42)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after remove, got %v", err)
	}

	r.Flush()

	rk := r.entityKey(tn, 42)
	n, _ := r.client.Exists(ctx, rk).Result()
	if n != 0 {
		t.Fatal("expected key deleted from Redis after flush")
	}
}

func TestTenantCoalesced_DirectUpdate(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	type HP struct {
		Current int `json:"current"`
		Max     int `json:"max"`
	}

	r := NewTenantCoalescedRegistry[uint32, HP](client, "monster", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.DirectPut(ctx, tn, 1, HP{Current: 100, Max: 100})

	result, err := r.DirectUpdate(ctx, tn, 1, func(hp HP) HP {
		hp.Current -= 30
		return hp
	})
	if err != nil {
		t.Fatalf("DirectUpdate failed: %v", err)
	}
	if result.Current != 70 {
		t.Fatalf("expected 70 HP, got %d", result.Current)
	}

	// Verify persisted.
	val, _ := r.DirectGet(ctx, tn, 1)
	if val.Current != 70 {
		t.Fatalf("expected 70 from Redis, got %d", val.Current)
	}
}

func TestTenantCoalesced_GetAllValues(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, tn, 1, "a")
	_ = r.Put(ctx, tn, 2, "b")
	_ = r.Put(ctx, tn, 3, "c")
	r.Flush()

	vals, err := r.GetAllValues(ctx, tn)
	if err != nil {
		t.Fatalf("GetAllValues failed: %v", err)
	}
	if len(vals) != 3 {
		t.Fatalf("expected 3 values, got %d", len(vals))
	}
}

func TestTenantCoalesced_GetAllValuesIncludesPendingWrites(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	// Flush one entry to Redis.
	_ = r.Put(ctx, tn, 1, "old-value")
	r.Flush()

	// Buffer a new write for the same key.
	_ = r.Put(ctx, tn, 1, "pending-value")

	vals, err := r.GetAllValues(ctx, tn)
	if err != nil {
		t.Fatalf("GetAllValues failed: %v", err)
	}
	if len(vals) != 1 {
		t.Fatalf("expected 1 value, got %d", len(vals))
	}
	if vals[0] != "pending-value" {
		t.Fatalf("expected pending-value, got %s", vals[0])
	}
}

func TestTenantCoalesced_GetAllValuesExcludesPendingRemoves(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, tn, 1, "a")
	_ = r.Put(ctx, tn, 2, "b")
	r.Flush()

	// Remove one entry (pending, not yet flushed).
	_ = r.Remove(ctx, tn, 1)

	vals, err := r.GetAllValues(ctx, tn)
	if err != nil {
		t.Fatalf("GetAllValues failed: %v", err)
	}
	if len(vals) != 1 {
		t.Fatalf("expected 1 value (removed entry excluded), got %d", len(vals))
	}
}

func TestTenantCoalesced_Exists(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	exists, _ := r.Exists(ctx, tn, 42)
	if exists {
		t.Fatal("expected key to not exist")
	}

	_ = r.Put(ctx, tn, 42, "value")
	exists, _ = r.Exists(ctx, tn, 42)
	if !exists {
		t.Fatal("expected key to exist after put")
	}

	_ = r.Remove(ctx, tn, 42)
	exists, _ = r.Exists(ctx, tn, 42)
	if exists {
		t.Fatal("expected key to not exist after remove")
	}
}

func TestTenantCoalesced_ShutdownFlushes(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)

	_ = r.Put(ctx, tn, 42, "shutting-down")
	r.Shutdown()

	rk := tenantEntityKey("temporal", tn, "42")
	data, err := client.Get(ctx, rk).Result()
	if err != nil {
		t.Fatalf("expected key in Redis after shutdown: %v", err)
	}
	if data != `"shutting-down"` {
		t.Fatalf("expected \"shutting-down\", got %s", data)
	}
}

func TestTenantCoalesced_RefreshPicksUpExternalWrites(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.DirectPut(ctx, tn, 42, "original")

	// External write.
	rk := r.entityKey(tn, 42)
	data, _ := r.marshal("updated-externally")
	_ = r.client.Set(ctx, rk, data, 0).Err()

	r.refresh()

	val, _ := r.Get(ctx, tn, 42)
	if val != "updated-externally" {
		t.Fatalf("expected updated-externally after refresh, got %s", val)
	}
}

func TestTenantCoalesced_WriteCoalescing(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()
	tn := testTenant()

	r := NewTenantCoalescedRegistry[uint32, string](client, "temporal", func(k uint32) string {
		return strconv.FormatUint(uint64(k), 10)
	}, time.Hour, time.Hour)
	defer r.Shutdown()

	_ = r.Put(ctx, tn, 42, "first")
	_ = r.Put(ctx, tn, 42, "second")
	_ = r.Put(ctx, tn, 42, "third")

	r.Flush()

	val, _ := r.DirectGet(ctx, tn, 42)
	if val != "third" {
		t.Fatalf("expected third (last write wins), got %s", val)
	}
}
