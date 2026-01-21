package blocked

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// Cache is an interface for a blocked portals cache
type Cache interface {
	// IsBlocked checks if a portal is blocked for a character
	IsBlocked(tenantId uuid.UUID, characterId uint32, mapId uint32, portalId uint32) bool

	// Block adds a portal to the blocked list for a character
	Block(tenantId uuid.UUID, characterId uint32, mapId uint32, portalId uint32)

	// Unblock removes a portal from the blocked list for a character
	Unblock(tenantId uuid.UUID, characterId uint32, mapId uint32, portalId uint32)

	// ClearForCharacter removes all blocked portals for a character
	ClearForCharacter(tenantId uuid.UUID, characterId uint32)

	// GetForCharacter returns all blocked portals for a character
	GetForCharacter(tenantId uuid.UUID, characterId uint32) []Model
}

// InMemoryCache is an in-memory implementation of the Cache interface
type InMemoryCache struct {
	// blocked maps: tenantId -> characterId -> portalKey -> true
	// portalKey is "mapId:portalId"
	blocked map[uuid.UUID]map[uint32]map[string]bool
	mutex   sync.RWMutex
}

// Singleton instance of the cache
var instance *InMemoryCache
var once sync.Once

// GetCache returns the singleton instance of the cache
func GetCache() Cache {
	once.Do(func() {
		instance = &InMemoryCache{
			blocked: make(map[uuid.UUID]map[uint32]map[string]bool),
		}
	})
	return instance
}

// ResetCache resets the singleton cache instance for testing
func ResetCache() {
	instance = &InMemoryCache{
		blocked: make(map[uuid.UUID]map[uint32]map[string]bool),
	}
}

// portalKey creates a unique key for a map/portal combination
func portalKey(mapId uint32, portalId uint32) string {
	return fmt.Sprintf("%d:%d", mapId, portalId)
}

// parsePortalKey parses a portal key back to mapId and portalId
func parsePortalKey(key string) (uint32, uint32) {
	var mapId, portalId uint32
	fmt.Sscanf(key, "%d:%d", &mapId, &portalId)
	return mapId, portalId
}

// IsBlocked checks if a portal is blocked for a character
func (c *InMemoryCache) IsBlocked(tenantId uuid.UUID, characterId uint32, mapId uint32, portalId uint32) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	tenantBlocked, exists := c.blocked[tenantId]
	if !exists {
		return false
	}

	characterBlocked, exists := tenantBlocked[characterId]
	if !exists {
		return false
	}

	return characterBlocked[portalKey(mapId, portalId)]
}

// Block adds a portal to the blocked list for a character
func (c *InMemoryCache) Block(tenantId uuid.UUID, characterId uint32, mapId uint32, portalId uint32) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.blocked[tenantId]; !exists {
		c.blocked[tenantId] = make(map[uint32]map[string]bool)
	}

	if _, exists := c.blocked[tenantId][characterId]; !exists {
		c.blocked[tenantId][characterId] = make(map[string]bool)
	}

	c.blocked[tenantId][characterId][portalKey(mapId, portalId)] = true
}

// Unblock removes a portal from the blocked list for a character
func (c *InMemoryCache) Unblock(tenantId uuid.UUID, characterId uint32, mapId uint32, portalId uint32) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	tenantBlocked, exists := c.blocked[tenantId]
	if !exists {
		return
	}

	characterBlocked, exists := tenantBlocked[characterId]
	if !exists {
		return
	}

	delete(characterBlocked, portalKey(mapId, portalId))

	// Clean up empty maps
	if len(characterBlocked) == 0 {
		delete(tenantBlocked, characterId)
	}
	if len(tenantBlocked) == 0 {
		delete(c.blocked, tenantId)
	}
}

// ClearForCharacter removes all blocked portals for a character
func (c *InMemoryCache) ClearForCharacter(tenantId uuid.UUID, characterId uint32) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	tenantBlocked, exists := c.blocked[tenantId]
	if !exists {
		return
	}

	delete(tenantBlocked, characterId)

	// Clean up empty tenant map
	if len(tenantBlocked) == 0 {
		delete(c.blocked, tenantId)
	}
}

// GetForCharacter returns all blocked portals for a character
func (c *InMemoryCache) GetForCharacter(tenantId uuid.UUID, characterId uint32) []Model {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	tenantBlocked, exists := c.blocked[tenantId]
	if !exists {
		return []Model{}
	}

	characterBlocked, exists := tenantBlocked[characterId]
	if !exists {
		return []Model{}
	}

	result := make([]Model, 0, len(characterBlocked))
	for key := range characterBlocked {
		mapId, portalId := parsePortalKey(key)
		result = append(result, NewModel(characterId, mapId, portalId))
	}

	return result
}
