package storage

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// NpcContextCacheInterface defines the interface for the NPC context cache
type NpcContextCacheInterface interface {
	Get(characterId uint32) (uint32, bool)
	Put(characterId uint32, npcId uint32, ttl time.Duration)
	Remove(characterId uint32)
}

// NpcContextCache is a Redis-backed cache for tracking which NPC a character is interacting with for storage
type NpcContextCache struct {
	client *goredis.Client
}

var npcContextCache NpcContextCacheInterface

func InitNpcContextCache(client *goredis.Client) {
	npcContextCache = &NpcContextCache{client: client}
}

// GetNpcContextCache returns the singleton instance of the NPC context cache
func GetNpcContextCache() NpcContextCacheInterface {
	return npcContextCache
}

func (c *NpcContextCache) key(characterId uint32) string {
	return fmt.Sprintf("atlas:npc-context:%d", characterId)
}

// Get retrieves the NPC ID for a character if not expired
func (c *NpcContextCache) Get(characterId uint32) (uint32, bool) {
	val, err := c.client.Get(context.Background(), c.key(characterId)).Result()
	if err != nil {
		return 0, false
	}
	npcId, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(npcId), true
}

// Put stores the NPC context for a character with expiration
func (c *NpcContextCache) Put(characterId uint32, npcId uint32, ttl time.Duration) {
	c.client.Set(context.Background(), c.key(characterId), npcId, ttl)
}

// Remove clears the NPC context for a character (called on storage close or logout)
func (c *NpcContextCache) Remove(characterId uint32) {
	c.client.Del(context.Background(), c.key(characterId))
}
