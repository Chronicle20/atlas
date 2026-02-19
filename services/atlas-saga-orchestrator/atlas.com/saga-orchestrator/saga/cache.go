package saga

import (
	"context"
	"sync"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

// Cache is an interface for a saga cache
type Cache interface {
	// GetAll returns all sagas for the tenant in context
	GetAll(ctx context.Context) []Saga

	// GetById returns a saga by its transaction ID for the tenant in context
	GetById(ctx context.Context, transactionId uuid.UUID) (Saga, bool)

	// Put adds or updates a saga in the cache for the tenant in context.
	// Returns an error on version conflict (optimistic locking).
	Put(ctx context.Context, saga Saga) error

	// Remove removes a saga from the cache for the tenant in context
	Remove(ctx context.Context, transactionId uuid.UUID) bool
}

// InMemoryCache is an in-memory implementation of the Cache interface
type InMemoryCache struct {
	// tenantSagas is a map of tenant IDs to maps of transaction IDs to sagas
	tenantSagas map[uuid.UUID]map[uuid.UUID]Saga

	// mutex is used to synchronize access to the cache
	mutex sync.RWMutex
}

// Singleton cache instance
var cacheInstance Cache
var once sync.Once

// GetCache returns the singleton instance of the cache
func GetCache() Cache {
	once.Do(func() {
		cacheInstance = &InMemoryCache{
			tenantSagas: make(map[uuid.UUID]map[uuid.UUID]Saga),
		}
	})
	return cacheInstance
}

// SetCache sets the singleton cache instance (call from main.go before consumers start)
func SetCache(c Cache) {
	cacheInstance = c
	once.Do(func() {}) // ensure once is spent so GetCache doesn't overwrite
}

// ResetCache resets the singleton cache instance for testing
func ResetCache() {
	cacheInstance = &InMemoryCache{
		tenantSagas: make(map[uuid.UUID]map[uuid.UUID]Saga),
	}
}

// GetAll returns all sagas for the tenant in context
func (c *InMemoryCache) GetAll(ctx context.Context) []Saga {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	// Get the tenant's sagas map
	sagas, exists := c.tenantSagas[tenantId]
	if !exists {
		return []Saga{}
	}

	// Convert the map to a slice
	result := make([]Saga, 0, len(sagas))
	for _, saga := range sagas {
		result = append(result, saga)
	}

	return result
}

// GetById returns a saga by its transaction ID for the tenant in context
func (c *InMemoryCache) GetById(ctx context.Context, transactionId uuid.UUID) (Saga, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	// Get the tenant's sagas map
	sagas, exists := c.tenantSagas[tenantId]
	if !exists {
		return Saga{}, false
	}

	// Get the saga by transaction ID
	saga, exists := sagas[transactionId]
	return saga, exists
}

// Put adds or updates a saga in the cache for the tenant in context
func (c *InMemoryCache) Put(ctx context.Context, saga Saga) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	// Ensure the tenant's sagas map exists
	if _, exists := c.tenantSagas[tenantId]; !exists {
		c.tenantSagas[tenantId] = make(map[uuid.UUID]Saga)
	}

	// Add or update the saga
	c.tenantSagas[tenantId][saga.TransactionId()] = saga
	return nil
}

// Remove removes a saga from the cache for the tenant in context
func (c *InMemoryCache) Remove(ctx context.Context, transactionId uuid.UUID) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	// Get the tenant's sagas map
	sagas, exists := c.tenantSagas[tenantId]
	if !exists {
		return false
	}

	// Check if the saga exists
	_, exists = sagas[transactionId]
	if !exists {
		return false
	}

	// Remove the saga
	delete(sagas, transactionId)
	return true
}
