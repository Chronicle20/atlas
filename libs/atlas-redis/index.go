package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// Index provides Redis SET-based secondary indexes for reverse lookups.
// For example, mapping characterId -> partyId alongside the main party registry.
type Index struct {
	client    *goredis.Client
	namespace string
	indexName string
}

func NewIndex(client *goredis.Client, namespace string, indexName string) *Index {
	return &Index{
		client:    client,
		namespace: namespace,
		indexName: indexName,
	}
}

func (i *Index) indexKey(t tenant.Model, key string) string {
	return tenantEntityKey(i.namespace, t, fmt.Sprintf("_idx:%s:%s", i.indexName, key))
}

// Add associates a value with the given index key.
func (i *Index) Add(ctx context.Context, t tenant.Model, key string, value string) error {
	rk := i.indexKey(t, key)
	return i.client.SAdd(ctx, rk, value).Err()
}

// Remove disassociates a value from the given index key.
func (i *Index) Remove(ctx context.Context, t tenant.Model, key string, value string) error {
	rk := i.indexKey(t, key)
	return i.client.SRem(ctx, rk, value).Err()
}

// Lookup returns all values associated with the given index key.
func (i *Index) Lookup(ctx context.Context, t tenant.Model, key string) ([]string, error) {
	rk := i.indexKey(t, key)
	return i.client.SMembers(ctx, rk).Result()
}

// LookupOne returns a single value if exactly one exists, useful for 1:1 mappings.
func (i *Index) LookupOne(ctx context.Context, t tenant.Model, key string) (string, error) {
	members, err := i.Lookup(ctx, t, key)
	if err != nil {
		return "", err
	}
	if len(members) == 0 {
		return "", ErrNotFound
	}
	return members[0], nil
}

// RemoveAll removes all values associated with the given index key.
func (i *Index) RemoveAll(ctx context.Context, t tenant.Model, key string) error {
	rk := i.indexKey(t, key)
	return i.client.Del(ctx, rk).Err()
}

// Uint32Index is a typed convenience wrapper around Index for uint32 keys and values.
type Uint32Index struct {
	index *Index
}

func NewUint32Index(client *goredis.Client, namespace string, indexName string) *Uint32Index {
	return &Uint32Index{
		index: NewIndex(client, namespace, indexName),
	}
}

func (i *Uint32Index) Add(ctx context.Context, t tenant.Model, key uint32, value uint32) error {
	return i.index.Add(ctx, t, strconv.FormatUint(uint64(key), 10), strconv.FormatUint(uint64(value), 10))
}

func (i *Uint32Index) Remove(ctx context.Context, t tenant.Model, key uint32, value uint32) error {
	return i.index.Remove(ctx, t, strconv.FormatUint(uint64(key), 10), strconv.FormatUint(uint64(value), 10))
}

func (i *Uint32Index) Lookup(ctx context.Context, t tenant.Model, key uint32) ([]uint32, error) {
	members, err := i.index.Lookup(ctx, t, strconv.FormatUint(uint64(key), 10))
	if err != nil {
		return nil, err
	}
	result := make([]uint32, 0, len(members))
	for _, m := range members {
		v, err := strconv.ParseUint(m, 10, 32)
		if err != nil {
			continue
		}
		result = append(result, uint32(v))
	}
	return result, nil
}

func (i *Uint32Index) LookupOne(ctx context.Context, t tenant.Model, key uint32) (uint32, error) {
	s, err := i.index.LookupOne(ctx, t, strconv.FormatUint(uint64(key), 10))
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse uint32: %w", err)
	}
	return uint32(v), nil
}

func (i *Uint32Index) RemoveAll(ctx context.Context, t tenant.Model, key uint32) error {
	return i.index.RemoveAll(ctx, t, strconv.FormatUint(uint64(key), 10))
}
