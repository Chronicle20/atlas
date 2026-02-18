package expression

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupRegistryTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func testCtx(t tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), t)
}

func TestRegistry_Add(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	f := field.NewBuilder(0, 1, 100000000).Build()
	m := GetRegistry().add(ctx, 1000, f, 5)

	assert.Equal(t, ten, m.Tenant())
	assert.Equal(t, uint32(1000), m.CharacterId())
	assert.Equal(t, uint32(5), m.Expression())
}

func TestRegistry_Add_SetsExpiration(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	before := time.Now()
	f := field.NewBuilder(0, 1, 100000000).Build()
	m := GetRegistry().add(ctx, 1000, f, 5)
	after := time.Now()

	expectedMin := before.Add(5 * time.Second)
	expectedMax := after.Add(5 * time.Second)

	assert.True(t, !m.Expiration().Before(expectedMin))
	assert.True(t, !m.Expiration().After(expectedMax))
}

func TestRegistry_Add_ReplacesExisting(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, 1000, f, 5)
	m := GetRegistry().add(ctx, 1000, f, 10)

	assert.Equal(t, uint32(10), m.Expression())

	retrieved, found := GetRegistry().get(ctx, 1000)
	assert.True(t, found)
	assert.Equal(t, uint32(10), retrieved.Expression())
}

func TestRegistry_Get(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, 1000, f, 5)

	m, found := GetRegistry().get(ctx, 1000)

	assert.True(t, found)
	assert.Equal(t, uint32(1000), m.CharacterId())
	assert.Equal(t, uint32(5), m.Expression())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	_, found := GetRegistry().get(ctx, 9999)

	assert.False(t, found)
}

func TestRegistry_Clear(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, 1000, f, 5)

	_, found := GetRegistry().get(ctx, 1000)
	assert.True(t, found)

	GetRegistry().clear(ctx, 1000)

	_, found = GetRegistry().get(ctx, 1000)
	assert.False(t, found)
}

func TestRegistry_Clear_NonExistent(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	// Should not panic
	GetRegistry().clear(ctx, 9999)
}

func TestRegistry_PopExpired(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	now := time.Now()
	GetRegistry().SetNowFunc(func() time.Time { return now })

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, 1000, f, 5)

	// Advance clock past TTL
	GetRegistry().SetNowFunc(func() time.Time { return now.Add(6 * time.Second) })

	expired := GetRegistry().popExpired(context.Background())

	assert.Len(t, expired, 1)
	if len(expired) == 0 {
		t.FailNow()
	}
	assert.Equal(t, uint32(1000), expired[0].CharacterId())

	// Verify it was removed
	_, found := GetRegistry().get(ctx, 1000)
	assert.False(t, found)
}

func TestRegistry_PopExpired_LeavesNonExpired(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	now := time.Now()
	GetRegistry().SetNowFunc(func() time.Time { return now })

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, 1000, f, 5)

	// Don't advance clock - nothing should be expired
	expired := GetRegistry().popExpired(context.Background())

	assert.Len(t, expired, 0)

	_, found := GetRegistry().get(ctx, 1000)
	assert.True(t, found)
}

func TestRegistry_PopExpired_MixedExpiration(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	now := time.Now()
	GetRegistry().SetNowFunc(func() time.Time { return now })

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx, 1000, f, 5)

	// Add second expression 3 seconds later
	GetRegistry().SetNowFunc(func() time.Time { return now.Add(3 * time.Second) })
	GetRegistry().add(ctx, 2000, f, 10)

	// Advance to 6 seconds - first expired, second not
	GetRegistry().SetNowFunc(func() time.Time { return now.Add(6 * time.Second) })

	expired := GetRegistry().popExpired(context.Background())

	assert.Len(t, expired, 1)
	assert.Equal(t, uint32(1000), expired[0].CharacterId())

	// Non-expired still exists
	_, found := GetRegistry().get(ctx, 2000)
	assert.True(t, found)

	// Expired was removed
	_, found = GetRegistry().get(ctx, 1000)
	assert.False(t, found)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx1, 1000, f, 5)

	m1, found1 := GetRegistry().get(ctx1, 1000)
	assert.True(t, found1)
	assert.Equal(t, uint32(5), m1.Expression())

	_, found2 := GetRegistry().get(ctx2, 1000)
	assert.False(t, found2)
}

func TestRegistry_TenantIsolation_SameCharacterId(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx1, 1000, f, 5)
	GetRegistry().add(ctx2, 1000, f, 10)

	m1, found1 := GetRegistry().get(ctx1, 1000)
	assert.True(t, found1)
	assert.Equal(t, uint32(5), m1.Expression())

	m2, found2 := GetRegistry().get(ctx2, 1000)
	assert.True(t, found2)
	assert.Equal(t, uint32(10), m2.Expression())
}

func TestRegistry_ConcurrentAdd(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			characterId := uint32(1000 + idx)
			f := field.NewBuilder(0, 1, 100000000).Build()
			GetRegistry().add(ctx, characterId, f, uint32(idx))
		}(i)
	}

	wg.Wait()

	for i := 0; i < numGoroutines; i++ {
		characterId := uint32(1000 + i)
		_, found := GetRegistry().get(ctx, characterId)
		assert.True(t, found, "Character %d should exist", characterId)
	}
}

func TestRegistry_ConcurrentAddAndClear(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			characterId := uint32(1000 + idx)
			f := field.NewBuilder(0, 1, 100000000).Build()
			GetRegistry().add(ctx, characterId, f, uint32(idx))
		}(i)
	}

	wg.Wait()

	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			characterId := uint32(1000 + idx)
			GetRegistry().clear(ctx, characterId)
		}(i)
	}

	wg.Wait()

	for i := 0; i < 25; i++ {
		characterId := uint32(1000 + i)
		_, found := GetRegistry().get(ctx, characterId)
		assert.False(t, found, "Character %d should be cleared", characterId)
	}

	for i := 25; i < 50; i++ {
		characterId := uint32(1000 + i)
		_, found := GetRegistry().get(ctx, characterId)
		assert.True(t, found, "Character %d should still exist", characterId)
	}
}

func TestRegistry_PopExpired_MultipleTenants(t *testing.T) {
	setupRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	now := time.Now()
	GetRegistry().SetNowFunc(func() time.Time { return now })

	f := field.NewBuilder(0, 1, 100000000).Build()
	GetRegistry().add(ctx1, 1000, f, 5)
	GetRegistry().add(ctx2, 2000, f, 10)

	// Advance past TTL
	GetRegistry().SetNowFunc(func() time.Time { return now.Add(6 * time.Second) })

	expired := GetRegistry().popExpired(context.Background())

	assert.Len(t, expired, 2)

	charIds := make(map[uint32]bool)
	for _, e := range expired {
		charIds[e.CharacterId()] = true
	}
	assert.True(t, charIds[1000])
	assert.True(t, charIds[2000])
}
