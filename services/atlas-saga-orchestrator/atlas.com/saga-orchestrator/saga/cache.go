package saga

import (
	"context"
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	// New entries are inserted with lifecycle SagaLifecyclePending.
	Put(ctx context.Context, saga Saga) error

	// Remove removes a saga from the cache for the tenant in context
	Remove(ctx context.Context, transactionId uuid.UUID) bool

	// TryTransition atomically moves the saga's lifecycle state from `from` to `to`.
	// Returns true if the transition succeeded, false if the current state is not
	// `from` (or the saga does not exist). Only transitions allowed by
	// IsValidTransition are honored.
	TryTransition(ctx context.Context, transactionId uuid.UUID, from, to SagaLifecycleState) bool

	// GetLifecycle returns the current lifecycle state of a saga.
	// Second return is false if the saga does not exist in the cache.
	GetLifecycle(ctx context.Context, transactionId uuid.UUID) (SagaLifecycleState, bool)
}

// inMemoryEntry bundles the Saga with its lifecycle state so both are protected
// by the cache's top-level mutex.
type inMemoryEntry struct {
	saga      Saga
	lifecycle SagaLifecycleState
}

// InMemoryCache is an in-memory implementation of the Cache interface
type InMemoryCache struct {
	// tenantSagas is a map of tenant IDs to maps of transaction IDs to entries
	tenantSagas map[uuid.UUID]map[uuid.UUID]*inMemoryEntry

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
			tenantSagas: make(map[uuid.UUID]map[uuid.UUID]*inMemoryEntry),
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
		tenantSagas: make(map[uuid.UUID]map[uuid.UUID]*inMemoryEntry),
	}
}

// GetAll returns all sagas for the tenant in context
func (c *InMemoryCache) GetAll(ctx context.Context) []Saga {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	// Get the tenant's sagas map
	entries, exists := c.tenantSagas[tenantId]
	if !exists {
		return []Saga{}
	}

	// Convert the map to a slice
	result := make([]Saga, 0, len(entries))
	for _, e := range entries {
		result = append(result, e.saga)
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
	entries, exists := c.tenantSagas[tenantId]
	if !exists {
		return Saga{}, false
	}

	// Get the saga by transaction ID
	e, exists := entries[transactionId]
	if !exists {
		return Saga{}, false
	}
	return e.saga, true
}

// Put adds or updates a saga in the cache for the tenant in context.
// A fresh entry is inserted with lifecycle SagaLifecyclePending; updates
// to an existing entry preserve the current lifecycle state.
func (c *InMemoryCache) Put(ctx context.Context, saga Saga) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	// Ensure the tenant's sagas map exists
	if _, exists := c.tenantSagas[tenantId]; !exists {
		c.tenantSagas[tenantId] = make(map[uuid.UUID]*inMemoryEntry)
	}

	// Add or update the saga, preserving lifecycle on update
	if existing, ok := c.tenantSagas[tenantId][saga.TransactionId()]; ok {
		existing.saga = saga
		return nil
	}
	c.tenantSagas[tenantId][saga.TransactionId()] = &inMemoryEntry{
		saga:      saga,
		lifecycle: SagaLifecyclePending,
	}
	return nil
}

// Remove removes a saga from the cache for the tenant in context
func (c *InMemoryCache) Remove(ctx context.Context, transactionId uuid.UUID) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	// Get the tenant's sagas map
	entries, exists := c.tenantSagas[tenantId]
	if !exists {
		return false
	}

	// Check if the saga exists
	_, exists = entries[transactionId]
	if !exists {
		return false
	}

	// Remove the saga
	delete(entries, transactionId)
	return true
}

// TryTransition atomically moves the saga's lifecycle from `from` to `to`.
// Returns true if the transition succeeded, false otherwise (including when
// the saga is missing or the transition is not permitted by IsValidTransition).
func (c *InMemoryCache) TryTransition(ctx context.Context, transactionId uuid.UUID, from, to SagaLifecycleState) bool {
	if !IsValidTransition(from, to) {
		return false
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	entries, exists := c.tenantSagas[tenantId]
	if !exists {
		return false
	}

	e, exists := entries[transactionId]
	if !exists {
		return false
	}

	if e.lifecycle != from {
		return false
	}
	e.lifecycle = to
	return true
}

// GetLifecycle returns the lifecycle state of a saga, or (zero, false) if missing.
func (c *InMemoryCache) GetLifecycle(ctx context.Context, transactionId uuid.UUID) (SagaLifecycleState, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	entries, exists := c.tenantSagas[tenantId]
	if !exists {
		return "", false
	}
	e, exists := entries[transactionId]
	if !exists {
		return "", false
	}
	return e.lifecycle, true
}
