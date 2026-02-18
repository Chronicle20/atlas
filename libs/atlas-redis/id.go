package redis

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

const defaultStartID = 1000000000

// IDGenerator provides per-tenant auto-incrementing IDs using Redis INCR.
type IDGenerator struct {
	client    *goredis.Client
	namespace string
	startID   int64
}

func NewIDGenerator(client *goredis.Client, namespace string) *IDGenerator {
	return &IDGenerator{
		client:    client,
		namespace: namespace,
		startID:   defaultStartID,
	}
}

func NewIDGeneratorWithStart(client *goredis.Client, namespace string, startID int64) *IDGenerator {
	return &IDGenerator{
		client:    client,
		namespace: namespace,
		startID:   startID,
	}
}

func (g *IDGenerator) idKey(t tenant.Model) string {
	return tenantEntityKey(g.namespace, t, "_id")
}

// NextID returns the next unique ID for the given tenant.
// On the first call for a tenant, it initializes the counter to startID.
func (g *IDGenerator) NextID(ctx context.Context, t tenant.Model) (uint32, error) {
	key := g.idKey(t)

	// Use SETNX to initialize only if key doesn't exist.
	_, err := g.client.SetNX(ctx, key, g.startID-1, 0).Result()
	if err != nil {
		return 0, fmt.Errorf("redis setnx: %w", err)
	}

	id, err := g.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr: %w", err)
	}
	return uint32(id), nil
}

// GlobalIDGenerator provides a single global auto-incrementing ID counter (not tenant-scoped).
type GlobalIDGenerator struct {
	client    *goredis.Client
	namespace string
	startID   int64
}

func NewGlobalIDGenerator(client *goredis.Client, namespace string, startID int64) *GlobalIDGenerator {
	return &GlobalIDGenerator{
		client:    client,
		namespace: namespace,
		startID:   startID,
	}
}

func (g *GlobalIDGenerator) idKey() string {
	return namespacedKey(g.namespace, "_id")
}

// NextID returns the next globally unique ID.
func (g *GlobalIDGenerator) NextID(ctx context.Context) (uint32, error) {
	key := g.idKey()

	_, err := g.client.SetNX(ctx, key, g.startID-1, 0).Result()
	if err != nil {
		return 0, fmt.Errorf("redis setnx: %w", err)
	}

	id, err := g.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr: %w", err)
	}
	return uint32(id), nil
}
