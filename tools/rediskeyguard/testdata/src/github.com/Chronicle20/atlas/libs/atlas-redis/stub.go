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
