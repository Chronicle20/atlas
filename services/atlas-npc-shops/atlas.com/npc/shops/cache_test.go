package shops

import (
	"atlas-npc/data/consumable"
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func setupTestCache(t *testing.T) *ConsumableCache {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return &ConsumableCache{client: rc}
}

func TestConsumableCache_SetAndGet(t *testing.T) {
	c := setupTestCache(t)
	l, _ := test.NewNullLogger()
	tenantId := uuid.New()
	ctx := context.Background()

	models := []consumable.Model{}
	data, _ := json.Marshal([]struct {
		Id    uint32 `json:"id"`
		Price uint32 `json:"price"`
	}{
		{Id: 2070000, Price: 500},
		{Id: 2070001, Price: 1000},
	})
	_ = json.Unmarshal(data, &models)

	c.SetConsumables(tenantId, models)

	result := c.GetConsumables(l, ctx, tenantId)

	assert.Len(t, result, 2)
	assert.Equal(t, uint32(2070000), result[0].Id())
	assert.Equal(t, uint32(500), result[0].Price())
	assert.Equal(t, uint32(2070001), result[1].Id())
	assert.Equal(t, uint32(1000), result[1].Price())
}

func TestConsumableCache_TenantIsolation(t *testing.T) {
	c := setupTestCache(t)
	l, _ := test.NewNullLogger()
	tenant1 := uuid.New()
	tenant2 := uuid.New()
	ctx := context.Background()

	models1 := []consumable.Model{}
	data, _ := json.Marshal([]struct {
		Id uint32 `json:"id"`
	}{{Id: 1}})
	_ = json.Unmarshal(data, &models1)

	models2 := []consumable.Model{}
	data, _ = json.Marshal([]struct {
		Id uint32 `json:"id"`
	}{{Id: 2}, {Id: 3}})
	_ = json.Unmarshal(data, &models2)

	c.SetConsumables(tenant1, models1)
	c.SetConsumables(tenant2, models2)

	result1 := c.GetConsumables(l, ctx, tenant1)
	result2 := c.GetConsumables(l, ctx, tenant2)

	assert.Len(t, result1, 1)
	assert.Equal(t, uint32(1), result1[0].Id())
	assert.Len(t, result2, 2)
	assert.Equal(t, uint32(2), result2[0].Id())
}

func TestConsumableCache_EmptyOnMiss(t *testing.T) {
	c := setupTestCache(t)
	l, _ := test.NewNullLogger()
	tenantId := uuid.New()
	ctx := context.Background()

	// Cache miss with no data service will return empty slice
	result := c.GetConsumables(l, ctx, tenantId)

	assert.Empty(t, result)
}
