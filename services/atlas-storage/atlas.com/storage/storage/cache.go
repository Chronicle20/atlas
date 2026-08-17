package storage

import (
	"context"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// NpcContextCacheInterface defines the interface for the NPC context cache
type NpcContextCacheInterface interface {
	Get(t tenant.Model, characterId uint32) (uint32, bool)
	Put(t tenant.Model, characterId uint32, npcId uint32, ttl time.Duration)
	Remove(t tenant.Model, characterId uint32)
}

// NpcContextCache is a Redis-backed cache for tracking which NPC a character is interacting with for storage.
// Keys are tenant-scoped: character ids are per-tenant sequences, so two tenants
// can otherwise collide on the same character id.
type NpcContextCache struct {
	reg *atlasredis.TenantRegistry[uint32, uint32]
}

var npcContextCache NpcContextCacheInterface

func InitNpcContextCache(client *goredis.Client) {
	npcContextCache = &NpcContextCache{
		reg: atlasredis.NewTenantRegistry[uint32, uint32](
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
func (c *NpcContextCache) Get(t tenant.Model, characterId uint32) (uint32, bool) {
	npcId, err := c.reg.Get(context.Background(), t, characterId)
	if err != nil {
		return 0, false
	}
	return npcId, true
}

// Put stores the NPC context for a character with expiration
func (c *NpcContextCache) Put(t tenant.Model, characterId uint32, npcId uint32, ttl time.Duration) {
	_ = c.reg.PutWithTTL(context.Background(), t, characterId, npcId, ttl)
}

// Remove clears the NPC context for a character (called on storage close or logout)
func (c *NpcContextCache) Remove(t tenant.Model, characterId uint32) {
	_ = c.reg.Remove(context.Background(), t, characterId)
}
