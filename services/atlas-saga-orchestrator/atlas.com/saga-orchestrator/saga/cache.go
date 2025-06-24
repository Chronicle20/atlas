package saga

import (
	"github.com/google/uuid"
	"sync"
)

// Cache is an interface for a saga cache
type Cache interface {
	// GetAll returns all sagas for a tenant
	GetAll(tenantId uuid.UUID) []Saga[any]

	// GetById returns a saga by its transaction ID for a tenant
	GetById(tenantId uuid.UUID, transactionId uuid.UUID) (Saga[any], bool)

	// Put adds or updates a saga in the cache for a tenant
	Put(tenantId uuid.UUID, saga Saga[any])

	// Remove removes a saga from the cache for a tenant
	Remove(tenantId uuid.UUID, transactionId uuid.UUID) bool
}

// InMemoryCache is an in-memory implementation of the Cache interface
type InMemoryCache struct {
	// tenantSagas is a map of tenant IDs to maps of transaction IDs to sagas
	tenantSagas map[uuid.UUID]map[uuid.UUID]Saga[any]

	// mutex is used to synchronize access to the cache
	mutex sync.RWMutex
}

// Singleton instance of the cache
var instance *InMemoryCache
var once sync.Once

// GetCache returns the singleton instance of the cache
func GetCache() Cache {
	once.Do(func() {
		instance = &InMemoryCache{
			tenantSagas: make(map[uuid.UUID]map[uuid.UUID]Saga[any]),
		}
	})
	return instance
}

// GetAll returns all sagas for a tenant
func (c *InMemoryCache) GetAll(tenantId uuid.UUID) []Saga[any] {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Get the tenant's sagas map
	sagas, exists := c.tenantSagas[tenantId]
	if !exists {
		return []Saga[any]{}
	}

	// Convert the map to a slice
	result := make([]Saga[any], 0, len(sagas))
	for _, saga := range sagas {
		result = append(result, saga)
	}

	return result
}

// GetByID returns a saga by its transaction ID for a tenant
func (c *InMemoryCache) GetById(tenantId uuid.UUID, transactionId uuid.UUID) (Saga[any], bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Get the tenant's sagas map
	sagas, exists := c.tenantSagas[tenantId]
	if !exists {
		return Saga[any]{}, false
	}

	// Get the saga by transaction ID
	saga, exists := sagas[transactionId]
	return saga, exists
}

// Put adds or updates a saga in the cache for a tenant
func (c *InMemoryCache) Put(tenantId uuid.UUID, saga Saga[any]) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Ensure the tenant's sagas map exists
	if _, exists := c.tenantSagas[tenantId]; !exists {
		c.tenantSagas[tenantId] = make(map[uuid.UUID]Saga[any])
	}

	// Add or update the saga
	c.tenantSagas[tenantId][saga.TransactionID] = saga
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
