package saga

import (
	"sync"

	"github.com/google/uuid"
)

// Cache is an interface for a saga cache
type Cache interface {
	// GetAll returns all sagas for a tenant
	GetAll(tenantId uuid.UUID) []Saga

	// GetById returns a saga by its transaction ID for a tenant
	GetById(tenantId uuid.UUID, transactionId uuid.UUID) (Saga, bool)

	// Put adds or updates a saga in the cache for a tenant.
	// Returns an error on version conflict (optimistic locking).
	Put(tenantId uuid.UUID, saga Saga) error

	// Remove removes a saga from the cache for a tenant
	Remove(tenantId uuid.UUID, transactionId uuid.UUID) bool
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

// GetAll returns all sagas for a tenant
func (c *InMemoryCache) GetAll(tenantId uuid.UUID) []Saga {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

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

// GetByID returns a saga by its transaction ID for a tenant
func (c *InMemoryCache) GetById(tenantId uuid.UUID, transactionId uuid.UUID) (Saga, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Get the tenant's sagas map
	sagas, exists := c.tenantSagas[tenantId]
	if !exists {
		return Saga{}, false
	}

	// Get the saga by transaction ID
	saga, exists := sagas[transactionId]
	return saga, exists
}

// Put adds or updates a saga in the cache for a tenant
func (c *InMemoryCache) Put(tenantId uuid.UUID, saga Saga) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Ensure the tenant's sagas map exists
	if _, exists := c.tenantSagas[tenantId]; !exists {
		c.tenantSagas[tenantId] = make(map[uuid.UUID]Saga)
	}

	// Add or update the saga
	c.tenantSagas[tenantId][saga.TransactionId()] = saga
	return nil
}

// Remove removes a saga from the cache for a tenant
func (c *InMemoryCache) Remove(tenantId uuid.UUID, transactionId uuid.UUID) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
