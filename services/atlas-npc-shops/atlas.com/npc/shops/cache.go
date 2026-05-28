package shops

import (
	"atlas-npc/data/consumable"
	"context"
	"errors"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ConsumableCacheInterface defines the interface for the consumable cache
type ConsumableCacheInterface interface {
	GetConsumables(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []consumable.Model
	SetConsumables(tenantId uuid.UUID, consumables []consumable.Model)
}

// ConsumableCache is a Redis-backed cache of rechargeable consumables per tenant
type ConsumableCache struct {
	reg *atlasredis.Registry[uuid.UUID, []consumable.Model]
}

var consumableCache ConsumableCacheInterface

// InitConsumableCache initializes the Redis-backed consumable cache
func InitConsumableCache(client *goredis.Client) {
	consumableCache = &ConsumableCache{
		reg: atlasredis.NewRegistry[uuid.UUID, []consumable.Model](client, "npc-shop:consumables", func(id uuid.UUID) string {
			return id.String()
		}),
	}
}

// GetConsumableCache returns the consumable cache instance
func GetConsumableCache() ConsumableCacheInterface {
	return consumableCache
}

// GetConsumables returns the rechargeable consumables for a tenant.
// Checks Redis first, falls back to the data service on cache miss.
func (c *ConsumableCache) GetConsumables(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []consumable.Model {
	models, err := c.reg.Get(ctx, tenantId)
	if err == nil {
		return models
	}
	if !errors.Is(err, atlasredis.ErrNotFound) {
		l.WithError(err).Warnf("Failed to read cached consumables for tenant %s, reloading.", tenantId)
	}

	l.Infof("Loading rechargeable consumables for tenant %s", tenantId)
	cp := consumable.NewProcessor(l.WithField("tenant", tenantId), ctx)
	consumables, err := cp.GetRechargeable()
	if err != nil {
		l.WithError(err).Errorf("Failed to get rechargeable consumables for tenant %s", tenantId)
		return []consumable.Model{}
	}

	l.Infof("Found %d rechargeable consumables for tenant %s", len(consumables), tenantId)

	_ = c.reg.Put(ctx, tenantId, consumables)

	return consumables
}

// SetConsumables sets the rechargeable consumables for a tenant
func (c *ConsumableCache) SetConsumables(tenantId uuid.UUID, consumables []consumable.Model) {
	_ = c.reg.Put(context.Background(), tenantId, consumables)
}

// GetDistinctTenants returns a list of distinct tenant IDs from the shop entities
func GetDistinctTenants(db *gorm.DB) ([]uuid.UUID, error) {
	var tenantIds []uuid.UUID
	err := db.Model(&Entity{}).Distinct("tenant_id").Pluck("tenant_id", &tenantIds).Error
	return tenantIds, err
}

// SetConsumableCacheForTesting replaces the consumable cache instance with the provided instance.
// This function is only intended to be used in tests.
func SetConsumableCacheForTesting(cache ConsumableCacheInterface) {
	consumableCache = cache
}
