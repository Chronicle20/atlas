package shops

import (
	"atlas-npc/data/consumable"
	"context"
	"encoding/json"
	"fmt"

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
	client *goredis.Client
}

var consumableCache ConsumableCacheInterface

func consumableCacheKey(tenantId uuid.UUID) string {
	return fmt.Sprintf("atlas:npc-shop:consumables:%s", tenantId.String())
}

// InitConsumableCache initializes the Redis-backed consumable cache
func InitConsumableCache(client *goredis.Client) {
	consumableCache = &ConsumableCache{client: client}
}

// GetConsumableCache returns the consumable cache instance
func GetConsumableCache() ConsumableCacheInterface {
	return consumableCache
}

// GetConsumables returns the rechargeable consumables for a tenant.
// Checks Redis first, falls back to the data service on cache miss.
func (c *ConsumableCache) GetConsumables(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []consumable.Model {
	key := consumableCacheKey(tenantId)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == nil {
		var models []consumable.Model
		if err = json.Unmarshal(data, &models); err == nil {
			return models
		}
		l.WithError(err).Warnf("Failed to unmarshal cached consumables for tenant %s, reloading.", tenantId)
	}

	l.Infof("Loading rechargeable consumables for tenant %s", tenantId)
	cp := consumable.NewProcessor(l.WithField("tenant", tenantId), ctx)
	consumables, err := cp.GetRechargeable()
	if err != nil {
		l.WithError(err).Errorf("Failed to get rechargeable consumables for tenant %s", tenantId)
		return []consumable.Model{}
	}

	l.Infof("Found %d rechargeable consumables for tenant %s", len(consumables), tenantId)

	if data, err = json.Marshal(consumables); err == nil {
		c.client.Set(ctx, key, data, 0)
	}

	return consumables
}

// SetConsumables sets the rechargeable consumables for a tenant
func (c *ConsumableCache) SetConsumables(tenantId uuid.UUID, consumables []consumable.Model) {
	key := consumableCacheKey(tenantId)
	if data, err := json.Marshal(consumables); err == nil {
		c.client.Set(context.Background(), key, data, 0)
	}
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
