package storage

import (
	"sync"
	"time"
)

// npcContextEntry represents a cached NPC context for a character's storage interaction
type npcContextEntry struct {
	npcId     uint32
	expiresAt time.Time
}

// NpcContextCacheInterface defines the interface for the NPC context cache
type NpcContextCacheInterface interface {
	Get(characterId uint32) (uint32, bool)
	Put(characterId uint32, npcId uint32, ttl time.Duration)
	Remove(characterId uint32)
}

// NpcContextCache is a singleton cache for tracking which NPC a character is interacting with for storage
type NpcContextCache struct {
	mu   sync.RWMutex
	data map[uint32]npcContextEntry
}

var npcContextCache NpcContextCacheInterface
var npcContextCacheOnce sync.Once

// GetNpcContextCache returns the singleton instance of the NPC context cache
func GetNpcContextCache() NpcContextCacheInterface {
	npcContextCacheOnce.Do(func() {
		npcContextCache = &NpcContextCache{
			data: make(map[uint32]npcContextEntry),
		}
	})
	return npcContextCache
}

// Get retrieves the NPC ID for a character if not expired
func (c *NpcContextCache) Get(characterId uint32) (uint32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[characterId]
	if !ok {
		return 0, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return 0, false
	}

	return entry.npcId, true
}

// Put stores the NPC context for a character with expiration
// TTL should be generous (e.g., 30 minutes) since storage sessions can be long
func (c *NpcContextCache) Put(characterId uint32, npcId uint32, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[characterId] = npcContextEntry{
		npcId:     npcId,
		expiresAt: time.Now().Add(ttl),
	}
}

// Remove clears the NPC context for a character (called on storage close or logout)
func (c *NpcContextCache) Remove(characterId uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, characterId)
}
