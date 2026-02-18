package messenger

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
}

func createTestCtx(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), ten)
}

func TestSunnyDayCreate(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	p := GetRegistry().Create(ctx, 1)

	assert.NotZero(t, p.Id())
	assert.Len(t, p.Members(), 1)
	assert.Equal(t, uint32(1), p.Members()[0].Id())
}

func TestMultiMessengerCreate(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	p1 := GetRegistry().Create(ctx, 1)
	p2 := GetRegistry().Create(ctx, 2)

	assert.NotZero(t, p1.Id())
	assert.NotZero(t, p2.Id())
	assert.Greater(t, p2.Id(), p1.Id())
	assert.Len(t, p1.Members(), 1)
	assert.Equal(t, uint32(1), p1.Members()[0].Id())
	assert.Len(t, p2.Members(), 1)
	assert.Equal(t, uint32(2), p2.Members()[0].Id())
}

func TestMultiTenantCreate(t *testing.T) {
	setupTestRegistry(t)
	ctx1 := createTestCtx(t)
	ctx2 := createTestCtx(t)

	p1 := GetRegistry().Create(ctx1, 1)
	p2 := GetRegistry().Create(ctx2, 2)

	assert.NotZero(t, p1.Id())
	assert.NotZero(t, p2.Id())
	assert.Len(t, p1.Members(), 1)
	assert.Equal(t, uint32(1), p1.Members()[0].Id())
	assert.Len(t, p2.Members(), 1)
	assert.Equal(t, uint32(2), p2.Members()[0].Id())
}

func TestRegistry_Get(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	created := GetRegistry().Create(ctx, 100)

	found, err := GetRegistry().Get(ctx, created.Id())

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), found.Id())
	assert.Len(t, found.Members(), 1)
	assert.Equal(t, uint32(100), found.Members()[0].Id())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	_, err := GetRegistry().Get(ctx, 999999)

	assert.Error(t, err)
}

func TestRegistry_GetAll(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	GetRegistry().Create(ctx, 100)
	GetRegistry().Create(ctx, 200)

	all := GetRegistry().GetAll(ctx)

	assert.Len(t, all, 2)
}

func TestRegistry_Update(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	created := GetRegistry().Create(ctx, 100)

	updated, err := GetRegistry().Update(ctx, created.Id(), func(m Model) Model {
		return m.AddMember(200)
	})

	assert.NoError(t, err)
	assert.Len(t, updated.Members(), 2)
}

func TestRegistry_Update_AtCapacity(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	created := GetRegistry().Create(ctx, 100)
	GetRegistry().Update(ctx, created.Id(), func(m Model) Model { return m.AddMember(200) })
	GetRegistry().Update(ctx, created.Id(), func(m Model) Model { return m.AddMember(300) })

	_, err := GetRegistry().Update(ctx, created.Id(), func(m Model) Model { return m.AddMember(400) })

	assert.ErrorIs(t, err, ErrAtCapacity)
}

func TestRegistry_Remove(t *testing.T) {
	setupTestRegistry(t)
	ctx := createTestCtx(t)

	created := GetRegistry().Create(ctx, 100)

	GetRegistry().Remove(ctx, created.Id())

	_, err := GetRegistry().Get(ctx, created.Id())
	assert.Error(t, err)
}
