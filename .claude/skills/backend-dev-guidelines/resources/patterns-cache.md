
---
title: Cache Patterns
description: Singleton cache implementation patterns for microservices.
---

# Cache Patterns

## Overview
Caches in this architecture MUST be implemented as **singleton instances** shared across all requests, not as per-instance or per-request objects.

## Core Principle
**Caches are application-scoped, not request-scoped.**

A cache created per-request defeats the purpose of caching. Each HTTP request would get a fresh, empty cache that is discarded after the request completes.

---

## Standard Singleton Cache Pattern

### File: `cache.go`

```go
package mypackage

import (
    "sync"
    "time"
)

// cacheEntry represents a cached data entry with expiration
type cacheEntry struct {
    value     interface{}
    expiresAt time.Time
}

// CacheInterface defines the interface for the cache
type CacheInterface interface {
    Get(key uint32) (interface{}, bool)
    Put(key uint32, value interface{}, ttl time.Duration)
}

// Cache is a singleton cache for package data
type Cache struct {
    mu   sync.RWMutex
    data map[uint32]cacheEntry
}

var cache CacheInterface
var cacheOnce sync.Once

// GetCache returns the singleton instance of the cache
func GetCache() CacheInterface {
    cacheOnce.Do(func() {
        cache = &Cache{
            data: make(map[uint32]cacheEntry),
        }
    })
    return cache
}

// Get retrieves a value from cache if not expired
func (c *Cache) Get(key uint32) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, ok := c.data[key]
    if !ok {
        return nil, false
    }

    // Check if expired
    if time.Now().After(entry.expiresAt) {
        return nil, false
    }

    return entry.value, true
}

// Put stores a value in cache with expiration
func (c *Cache) Put(key uint32, value interface{}, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.data[key] = cacheEntry{
        value:     value,
        expiresAt: time.Now().Add(ttl),
    }
}
```

---

## Using Cache in Processors

### ❌ WRONG: Cache per processor instance
```go
// DON'T DO THIS
type ProcessorImpl struct {
    l     logrus.FieldLogger
    ctx   context.Context
    cache map[uint32]interface{}  // ❌ Per-instance cache
    mu    sync.RWMutex
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
    return &ProcessorImpl{
        l:     l,
        ctx:   ctx,
        cache: make(map[uint32]interface{}),  // ❌ Fresh cache every request
    }
}
```
**Problem:** Every request creates a new processor with a new empty cache. Cache never persists across requests.

---

### ✅ CORRECT: Singleton cache shared across processors
```go
// DO THIS
type ProcessorImpl struct {
    l     logrus.FieldLogger
    ctx   context.Context
    cache CacheInterface  // ✅ Reference to singleton
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
    return &ProcessorImpl{
        l:     l,
        ctx:   ctx,
        cache: GetCache(),  // ✅ Get singleton instance
    }
}

func (p *ProcessorImpl) GetData(id uint32) (Data, error) {
    // Check cache first
    if cached, ok := p.cache.Get(id); ok {
        p.l.Debugf("Cache hit for [%d]", id)
        return cached.(Data), nil
    }

    // Fetch from source
    data, err := p.fetchFromSource(id)
    if err != nil {
        return Data{}, err
    }

    // Cache the result
    p.cache.Put(id, data, time.Hour)
    return data, nil
}
```

---

## Multi-Tenant Cache Pattern

For multi-tenant architectures, cache keys should include tenant ID:

```go
package mypackage

import (
    "github.com/google/uuid"
    "sync"
)

// CacheInterface defines the interface for the multi-tenant cache
type CacheInterface interface {
    Get(tenantId uuid.UUID, key uint32) (interface{}, bool)
    Put(tenantId uuid.UUID, key uint32, value interface{})
}

// Cache is a singleton cache partitioned by tenant
type Cache struct {
    mu   sync.RWMutex
    data map[uuid.UUID]map[uint32]interface{}
}

var cache CacheInterface
var cacheOnce sync.Once

// GetCache returns the singleton instance of the cache
func GetCache() CacheInterface {
    cacheOnce.Do(func() {
        cache = &Cache{
            data: make(map[uuid.UUID]map[uint32]interface{}),
        }
    })
    return cache
}

// Get retrieves a value from cache for a specific tenant
func (c *Cache) Get(tenantId uuid.UUID, key uint32) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    tenantData, ok := c.data[tenantId]
    if !ok {
        return nil, false
    }

    value, ok := tenantData[key]
    return value, ok
}

// Put stores a value in cache for a specific tenant
func (c *Cache) Put(tenantId uuid.UUID, key uint32, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if _, ok := c.data[tenantId]; !ok {
        c.data[tenantId] = make(map[uint32]interface{})
    }

    c.data[tenantId][key] = value
}
```

---

## Testing Pattern

### Provide a test setter for cache replacement

```go
// SetCacheForTesting replaces the singleton instance for testing
// This function is only intended to be used in tests
func SetCacheForTesting(c CacheInterface) {
    cache = c
}

// ResetCache resets the singleton cache for testing
func ResetCache() {
    cache = &Cache{
        data: make(map[uint32]cacheEntry),
    }
}
```

### In tests:
```go
func TestProcessorWithMockCache(t *testing.T) {
    // Setup mock cache
    mockCache := &MockCache{
        data: map[uint32]interface{}{
            123: expectedData,
        },
    }
    SetCacheForTesting(mockCache)
    defer ResetCache()

    // Test processor
    processor := NewProcessor(log, ctx)
    result, err := processor.GetData(123)

    assert.NoError(t, err)
    assert.Equal(t, expectedData, result)
}
```

---

## Examples in Codebase

Reference implementations:
- `services/atlas-npc-shops/atlas.com/npc/shops/cache.go` - Consumable cache with lazy loading
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/cache.go` - Multi-tenant saga cache
- `services/atlas-cashshop/atlas.com/cashshop/cashshop/inventory/asset/reservation/cache.go` - Reservation cache with background cleanup

---

## Key Takeaways

| Aspect | Implementation |
|--------|----------------|
| **Scope** | Application-wide singleton, not per-request |
| **Thread Safety** | Always use `sync.RWMutex` |
| **Initialization** | Use `sync.Once` to initialize singleton |
| **Access Pattern** | `GetCache()` function returns singleton |
| **Processor Usage** | Store reference to singleton, not create cache |
| **Multi-Tenancy** | Partition cache by tenant ID |
| **Testing** | Provide setter/reset functions for test isolation |
| **Expiration** | Include TTL for time-based invalidation |

---

## Decision Flow

```
Need to cache data?
  │
  ├─ Is this data shared across requests?
  │  └─ YES → Singleton cache (this pattern)
  │  └─ NO  → Consider if caching is needed
  │
  ├─ Does this need multi-tenancy?
  │  └─ YES → Use tenant-partitioned cache
  │  └─ NO  → Use simple singleton cache
  │
  └─ Where to initialize?
     └─ NEVER in processor constructor
     └─ ALWAYS via GetCache() singleton accessor
```
