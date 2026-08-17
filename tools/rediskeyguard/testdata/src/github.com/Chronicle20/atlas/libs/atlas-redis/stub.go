// Package redis is a minimal stub of
// github.com/Chronicle20/atlas/libs/atlas-redis for analysistest — just
// enough of the constructor surface (bare and Tenant-scoped) for the
// bareconstructor fixture to exercise the guard's second check.
package redis

import (
	goredis "github.com/redis/go-redis/v9"
)

// Registry is the bare (env-global) registry — banned outside this package.
type Registry[K comparable, V any] struct{}

func NewRegistry[K comparable, V any](client *goredis.Client, namespace string, keyFn func(K) string) *Registry[K, V] {
	return nil
}

// TenantRegistry is the tenant-scoped sibling — allowed anywhere.
type TenantRegistry[K comparable, V any] struct{}

func NewTenantRegistry[K comparable, V any](client *goredis.Client, namespace string, keyFn func(K) string) *TenantRegistry[K, V] {
	return nil
}

// Index, Uint32Index, IDGenerator and TTLRegistry are bare-shaped names with
// no bare/tenant-scoped split to migrate off of — every method already takes
// a tenant.Model (omitted here; the stub only needs to exist for the
// constructor call to type-check). Not in bannedConstructors.
type Index struct{}

func NewIndex(client *goredis.Client, namespace string, indexName string) *Index {
	return nil
}

type Uint32Index struct{}

func NewUint32Index(client *goredis.Client, namespace string, indexName string) *Uint32Index {
	return nil
}

type IDGenerator struct{}

func NewIDGenerator(client *goredis.Client, namespace string) *IDGenerator {
	return nil
}

func NewIDGeneratorWithStart(client *goredis.Client, namespace string, startID int64) *IDGenerator {
	return nil
}

type TTLRegistry[K comparable, V any] struct{}

func NewTTLRegistry[K comparable, V any](client *goredis.Client, namespace string, keyFn func(K) string, defaultTTL int64) *TTLRegistry[K, V] {
	return nil
}
