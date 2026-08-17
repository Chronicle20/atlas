package storage

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupTestCacheWithMr(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitNpcContextCache(client)
	return mr
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestPut_And_Get(t *testing.T) {
	setupTestCacheWithMr(t)
	tm := newTestTenant(t)

	GetNpcContextCache().Put(tm, 12345, 9001, 30*time.Minute)

	npcId, ok := GetNpcContextCache().Get(tm, 12345)
	assert.True(t, ok)
	assert.Equal(t, uint32(9001), npcId)
}

func TestGet_NotFound(t *testing.T) {
	setupTestCacheWithMr(t)
	tm := newTestTenant(t)

	_, ok := GetNpcContextCache().Get(tm, 99999)
	assert.False(t, ok)
}

func TestRemove(t *testing.T) {
	setupTestCacheWithMr(t)
	tm := newTestTenant(t)

	GetNpcContextCache().Put(tm, 12345, 9001, 30*time.Minute)
	GetNpcContextCache().Remove(tm, 12345)

	_, ok := GetNpcContextCache().Get(tm, 12345)
	assert.False(t, ok)
}

func TestRemove_NonExistent(t *testing.T) {
	setupTestCacheWithMr(t)
	tm := newTestTenant(t)

	// Should not panic
	GetNpcContextCache().Remove(tm, 99999)
}

func TestPut_Overwrite(t *testing.T) {
	setupTestCacheWithMr(t)
	tm := newTestTenant(t)

	GetNpcContextCache().Put(tm, 12345, 9001, 30*time.Minute)
	GetNpcContextCache().Put(tm, 12345, 9002, 30*time.Minute)

	npcId, ok := GetNpcContextCache().Get(tm, 12345)
	assert.True(t, ok)
	assert.Equal(t, uint32(9002), npcId)
}

func TestPut_TTL_Expiry(t *testing.T) {
	mr := setupTestCacheWithMr(t)
	tm := newTestTenant(t)

	GetNpcContextCache().Put(tm, 12345, 9001, 5*time.Second)

	// Confirm TTL is set
	key := "atlas:npc-context:" + tm.Id().String() + ":" + tm.Region() + ":83.1:12345"
	ttl := mr.TTL(key)
	assert.Greater(t, ttl, time.Duration(0))

	// Fast-forward past TTL
	mr.FastForward(10 * time.Second)

	_, ok := GetNpcContextCache().Get(tm, 12345)
	assert.False(t, ok, "key should be expired after TTL elapses")
}

func TestNpcContextCacheIsTenantScoped(t *testing.T) {
	setupTestCacheWithMr(t)

	t1, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	t2, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	GetNpcContextCache().Put(t1, 12345, 9001, 30*time.Minute)
	GetNpcContextCache().Put(t2, 12345, 9002, 30*time.Minute)

	got1, ok := GetNpcContextCache().Get(t1, 12345)
	if !ok || got1 != 9001 {
		t.Fatalf("tenant 1: got (%d, %v), want (9001, true)", got1, ok)
	}
	got2, ok := GetNpcContextCache().Get(t2, 12345)
	if !ok || got2 != 9002 {
		t.Fatalf("tenant 2: got (%d, %v), want (9002, true)", got2, ok)
	}

	GetNpcContextCache().Remove(t1, 12345)
	if _, ok := GetNpcContextCache().Get(t2, 12345); !ok {
		t.Fatal("removing tenant 1's entry removed tenant 2's")
	}
}
