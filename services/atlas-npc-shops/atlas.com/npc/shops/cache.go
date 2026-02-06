package shops

import (
	"atlas-npc/data/consumable"
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ConsumableCacheInterface defines the interface for the consumable cache
type ConsumableCacheInterface interface {
	GetConsumables(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []consumable.Model
	SetConsumables(tenantId uuid.UUID, consumables []consumable.Model)
}

// ConsumableCache is a cache of rechargeable consumables per tenant
type ConsumableCache struct {
	mutex       sync.RWMutex
	consumables map[uuid.UUID][]consumable.Model
}

var consumableCache ConsumableCacheInterface
var cacheOnce sync.Once

// GetConsumableCache returns the singleton instance of the consumable cache
func GetConsumableCache() ConsumableCacheInterface {
	cacheOnce.Do(func() {
		consumableCache = &ConsumableCache{
			consumables: make(map[uuid.UUID][]consumable.Model),
		}
	})
	return consumableCache
}

// GetConsumables returns the rechargeable consumables for a tenant
// If the consumables are not in the cache, they will be loaded from the data service
func (c *ConsumableCache) GetConsumables(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) []consumable.Model {
	// First check if we have the consumables in the cache
	c.mutex.RLock()
	if consumables, ok := c.consumables[tenantId]; ok {
		c.mutex.RUnlock()
		// Return a copy of the slice to prevent external modifications
		result := make([]consumable.Model, len(consumables))
		copy(result, consumables)
		return result
	}
	c.mutex.RUnlock()

	// If not in cache, load them from the data service
	l.Infof("Loading rechargeable consumables for tenant %s", tenantId)
	cp := consumable.NewProcessor(l.WithField("tenant", tenantId), ctx)
	consumables, err := cp.GetRechargeable()
	if err != nil {
		l.WithError(err).Errorf("Failed to get rechargeable consumables for tenant %s", tenantId)
		return []consumable.Model{}
	}

	l.Infof("Found %d rechargeable consumables for tenant %s", len(consumables), tenantId)

	// Cache the consumables
	c.mutex.Lock()
	c.consumables[tenantId] = consumables
	c.mutex.Unlock()

	// Return a copy of the slice to prevent external modifications
	result := make([]consumable.Model, len(consumables))
	copy(result, consumables)
	return result
}

// SetConsumables sets the rechargeable consumables for a tenant
func (c *ConsumableCache) SetConsumables(tenantId uuid.UUID, consumables []consumable.Model) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.consumables[tenantId] = consumables
}

// GetDistinctTenants returns a list of distinct tenant IDs from the shop entities
func GetDistinctTenants(db *gorm.DB) ([]uuid.UUID, error) {
	var tenantIds []uuid.UUID
	err := db.Model(&Entity{}).Distinct("tenant_id").Pluck("tenant_id", &tenantIds).Error
	return tenantIds, err
}

// SetConsumableCacheForTesting replaces the singleton instance of the consumable cache with the provided instance
// This function is only intended to be used in tests
func SetConsumableCacheForTesting(cache ConsumableCacheInterface) {
	consumableCache = cache
}
