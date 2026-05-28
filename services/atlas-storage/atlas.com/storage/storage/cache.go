package storage

import (
	"context"
	"errors"
	"strconv"
	"time"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
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
	reg *atlasredis.Registry[uint32, uint32]
}

var npcContextCache NpcContextCacheInterface

func InitNpcContextCache(client *goredis.Client) {
	npcContextCache = &NpcContextCache{
		reg: atlasredis.NewRegistry[uint32, uint32](
			client,
			"npc-context",
			func(characterId uint32) string {
				return strconv.FormatUint(uint64(characterId), 10)
			},
		),
	}
}

// GetNpcContextCache returns the singleton instance of the NPC context cache
func GetNpcContextCache() NpcContextCacheInterface {
	return npcContextCache
}

// Get retrieves the NPC ID for a character if not expired
func (c *NpcContextCache) Get(characterId uint32) (uint32, bool) {
	npcId, err := c.reg.Get(context.Background(), characterId)
	if err != nil {
		if errors.Is(err, atlasredis.ErrNotFound) {
			return 0, false
		}
		return 0, false
	}
	return npcId, true
}

// Put stores the NPC context for a character with expiration
func (c *NpcContextCache) Put(characterId uint32, npcId uint32, ttl time.Duration) {
	_ = c.reg.PutWithTTL(context.Background(), characterId, npcId, ttl)
}

// Remove clears the NPC context for a character (called on storage close or logout)
func (c *NpcContextCache) Remove(characterId uint32) {
	_ = c.reg.Remove(context.Background(), characterId)
}
