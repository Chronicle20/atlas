package channel

import (
	"context"
	"testing"

	channelConstant "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupChannelTestRegistry(t *testing.T) context.Context {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)

	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Register: %v", err)
	}
	return tenant.WithContext(context.Background(), tenantModel)
}

func TestRegistry_Add_GetAll(t *testing.T) {
	ctx := setupChannelTestRegistry(t)
	r := getRegistry()

	ch := channelConstant.NewModel(world.Id(0), channelConstant.Id(1))
	r.Add(ctx, ch)

	all := r.GetAll(ctx)
	assert.Len(t, all, 1)
	assert.Equal(t, world.Id(0), all[0].WorldId())
	assert.Equal(t, channelConstant.Id(1), all[0].Id())
}

func TestRegistry_Remove(t *testing.T) {
	ctx := setupChannelTestRegistry(t)
	r := getRegistry()

	ch := channelConstant.NewModel(world.Id(0), channelConstant.Id(2))
	r.Add(ctx, ch)
	r.Remove(ctx, ch)

	all := r.GetAll(ctx)
	assert.Len(t, all, 0)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	ctx1 := setupChannelTestRegistry(t)
	r := getRegistry()

	tenantId2 := uuid.New()
	tenantModel2, err := tenant.Register(tenantId2, "GMS", 83, 1)
	assert.NoError(t, err)
	ctx2 := tenant.WithContext(context.Background(), tenantModel2)

	ch := channelConstant.NewModel(world.Id(0), channelConstant.Id(1))
	r.Add(ctx1, ch)

	all1 := r.GetAll(ctx1)
	all2 := r.GetAll(ctx2)

	assert.Len(t, all1, 1)
	assert.Len(t, all2, 0)
}

func TestRegistry_MultipleChannels(t *testing.T) {
	ctx := setupChannelTestRegistry(t)
	r := getRegistry()

	channels := []channelConstant.Model{
		channelConstant.NewModel(world.Id(0), channelConstant.Id(1)),
		channelConstant.NewModel(world.Id(0), channelConstant.Id(2)),
		channelConstant.NewModel(world.Id(1), channelConstant.Id(1)),
	}
	for _, ch := range channels {
		r.Add(ctx, ch)
	}

	all := r.GetAll(ctx)
	assert.Len(t, all, 3)
}
