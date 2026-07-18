package game_test

import (
	"atlas-rps/game"
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupRegistryTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	game.InitRegistry(client)
}

func testCtx(t tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), t)
}

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return ten
}

func TestRegistry_PutAndGet(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	m := game.NewModelBuilder(ten).
		SetCharacterId(1000).
		SetWorldId(0).
		SetChannelId(1).
		SetNpcId(9000019).
		SetStatus(game.StatusOpen).
		MustBuild()

	game.GetRegistry().Put(ctx, m)

	retrieved, found := game.GetRegistry().Get(ctx, 1000)
	assert.True(t, found)
	assert.Equal(t, uint32(1000), retrieved.CharacterId())
	assert.Equal(t, game.StatusOpen, retrieved.Status())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	_, found := game.GetRegistry().Get(ctx, 9999)
	assert.False(t, found)
}

func TestRegistry_Remove(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	m := game.NewModelBuilder(ten).
		SetCharacterId(1000).
		MustBuild()
	game.GetRegistry().Put(ctx, m)

	_, found := game.GetRegistry().Get(ctx, 1000)
	assert.True(t, found)

	game.GetRegistry().Remove(ctx, 1000)

	_, found = game.GetRegistry().Get(ctx, 1000)
	assert.False(t, found)
}

func TestRegistry_PopExpired(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	now := time.Now()
	game.GetRegistry().SetNowFunc(func() time.Time { return now })

	m := game.NewModelBuilder(ten).
		SetCharacterId(1000).
		MustBuild()
	game.GetRegistry().Put(ctx, m)

	// Not expired yet.
	expired := game.GetRegistry().PopExpired(context.Background())
	assert.Len(t, expired, 0)

	// Advance clock past the default TTL (5 minutes).
	game.GetRegistry().SetNowFunc(func() time.Time { return now.Add(6 * time.Minute) })

	expired = game.GetRegistry().PopExpired(context.Background())
	assert.Len(t, expired, 1)
	if len(expired) == 0 {
		t.FailNow()
	}
	assert.Equal(t, uint32(1000), expired[0].CharacterId())

	// Verify it was removed from the registry.
	_, found := game.GetRegistry().Get(ctx, 1000)
	assert.False(t, found)
}

func TestRegistry_PopExpired_MultipleTenants(t *testing.T) {
	setupRegistryTest(t)

	ten1, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant1: %v", err)
	}
	ten2, err := tenant.Create(uuid.New(), "EMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant2: %v", err)
	}

	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	now := time.Now()
	game.GetRegistry().SetNowFunc(func() time.Time { return now })

	m1 := game.NewModelBuilder(ten1).SetCharacterId(1000).MustBuild()
	m2 := game.NewModelBuilder(ten2).SetCharacterId(2000).MustBuild()
	game.GetRegistry().Put(ctx1, m1)
	game.GetRegistry().Put(ctx2, m2)

	game.GetRegistry().SetNowFunc(func() time.Time { return now.Add(6 * time.Minute) })

	expired := game.GetRegistry().PopExpired(context.Background())
	assert.Len(t, expired, 2)

	charIds := make(map[uint32]bool)
	for _, e := range expired {
		charIds[e.CharacterId()] = true
	}
	assert.True(t, charIds[1000])
	assert.True(t, charIds[2000])
}
