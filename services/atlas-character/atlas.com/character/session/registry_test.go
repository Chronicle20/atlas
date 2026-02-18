package session

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return ten
}

func testCtx(ten tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), ten)
}

func TestAdd_And_Get(t *testing.T) {
	setupTestRegistry(t)
	ten := testTenant(t)
	ctx := testCtx(ten)

	ch := channel.NewModel(world.Id(0), channel.Id(1))
	err := GetRegistry().Add(ctx, 12345, ch, StateLoggedIn)
	assert.NoError(t, err)

	m, err := GetRegistry().Get(ctx, 12345)
	assert.NoError(t, err)
	assert.Equal(t, uint32(12345), m.CharacterId())
	assert.Equal(t, world.Id(0), m.WorldId())
	assert.Equal(t, channel.Id(1), m.ChannelId())
	assert.Equal(t, StateLoggedIn, m.State())
}

func TestAdd_AlreadyLoggedIn(t *testing.T) {
	setupTestRegistry(t)
	ten := testTenant(t)
	ctx := testCtx(ten)

	ch := channel.NewModel(world.Id(0), channel.Id(1))
	err := GetRegistry().Add(ctx, 12345, ch, StateLoggedIn)
	assert.NoError(t, err)

	err = GetRegistry().Add(ctx, 12345, ch, StateLoggedIn)
	assert.Error(t, err)
}

func TestAdd_TransitionState_Allowed(t *testing.T) {
	setupTestRegistry(t)
	ten := testTenant(t)
	ctx := testCtx(ten)

	ch := channel.NewModel(world.Id(0), channel.Id(1))
	// First set to transition state
	err := GetRegistry().Set(ctx, 12345, ch, StateTransition)
	assert.NoError(t, err)

	// Add should succeed since state is Transition, not LoggedIn
	err = GetRegistry().Add(ctx, 12345, ch, StateLoggedIn)
	assert.NoError(t, err)
}

func TestSet_Unconditional(t *testing.T) {
	setupTestRegistry(t)
	ten := testTenant(t)
	ctx := testCtx(ten)

	ch := channel.NewModel(world.Id(0), channel.Id(1))
	err := GetRegistry().Set(ctx, 12345, ch, StateLoggedIn)
	assert.NoError(t, err)

	// Set again should succeed even when already logged in
	ch2 := channel.NewModel(world.Id(0), channel.Id(2))
	err = GetRegistry().Set(ctx, 12345, ch2, StateTransition)
	assert.NoError(t, err)

	m, err := GetRegistry().Get(ctx, 12345)
	assert.NoError(t, err)
	assert.Equal(t, channel.Id(2), m.ChannelId())
	assert.Equal(t, StateTransition, m.State())
}

func TestGet_NotFound(t *testing.T) {
	setupTestRegistry(t)
	ten := testTenant(t)
	ctx := testCtx(ten)

	_, err := GetRegistry().Get(ctx, 99999)
	assert.Error(t, err)
}

func TestRemove(t *testing.T) {
	setupTestRegistry(t)
	ten := testTenant(t)
	ctx := testCtx(ten)

	ch := channel.NewModel(world.Id(0), channel.Id(1))
	_ = GetRegistry().Add(ctx, 12345, ch, StateLoggedIn)

	GetRegistry().Remove(ctx, 12345)

	_, err := GetRegistry().Get(ctx, 12345)
	assert.Error(t, err)
}

func TestGetAll_Empty(t *testing.T) {
	setupTestRegistry(t)

	results := GetRegistry().GetAll(context.Background())
	assert.Len(t, results, 0)
}

func TestGetAll_SingleTenant(t *testing.T) {
	setupTestRegistry(t)
	ten := testTenant(t)
	ctx := testCtx(ten)

	ch := channel.NewModel(world.Id(0), channel.Id(1))
	_ = GetRegistry().Add(ctx, 12345, ch, StateLoggedIn)
	_ = GetRegistry().Add(ctx, 67890, ch, StateTransition)

	results := GetRegistry().GetAll(context.Background())
	assert.Len(t, results, 2)
}

func TestGetAll_CrossTenant(t *testing.T) {
	setupTestRegistry(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	ch := channel.NewModel(world.Id(0), channel.Id(1))
	_ = GetRegistry().Add(ctx1, 12345, ch, StateLoggedIn)
	_ = GetRegistry().Add(ctx2, 67890, ch, StateLoggedIn)

	results := GetRegistry().GetAll(context.Background())
	assert.Len(t, results, 2)

	charIds := make(map[uint32]bool)
	for _, m := range results {
		charIds[m.CharacterId()] = true
	}
	assert.True(t, charIds[12345])
	assert.True(t, charIds[67890])
}

func TestTenantIsolation(t *testing.T) {
	setupTestRegistry(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	ctx1 := testCtx(ten1)
	ctx2 := testCtx(ten2)

	ch := channel.NewModel(world.Id(0), channel.Id(1))
	_ = GetRegistry().Add(ctx1, 12345, ch, StateLoggedIn)

	// Tenant 1 can see it
	_, err := GetRegistry().Get(ctx1, 12345)
	assert.NoError(t, err)

	// Tenant 2 cannot see it
	_, err = GetRegistry().Get(ctx2, 12345)
	assert.Error(t, err)
}

func TestRemove_NonExistent(t *testing.T) {
	setupTestRegistry(t)
	ten := testTenant(t)
	ctx := testCtx(ten)

	// Should not panic
	GetRegistry().Remove(ctx, 99999)
}

func TestModel_JsonRoundTrip(t *testing.T) {
	setupTestRegistry(t)
	ten := testTenant(t)
	ctx := testCtx(ten)

	ch := channel.NewModel(world.Id(0), channel.Id(3))
	_ = GetRegistry().Add(ctx, 12345, ch, StateLoggedIn)

	m, err := GetRegistry().Get(ctx, 12345)
	assert.NoError(t, err)

	mt := m.Tenant()
	assert.Equal(t, ten.Id(), mt.Id())
	assert.Equal(t, uint32(12345), m.CharacterId())
	assert.Equal(t, world.Id(0), m.WorldId())
	assert.Equal(t, channel.Id(3), m.ChannelId())
	assert.Equal(t, StateLoggedIn, m.State())
}
