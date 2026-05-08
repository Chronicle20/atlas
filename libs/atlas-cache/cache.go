package cache

import (
	"sync"
	"time"
)

// Cache is a generic in-process TTL cache supporting distinct positive
// and negative entry lifetimes. All methods are safe for concurrent use.
type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Put(key K, value V)
	PutNegative(key K)
	IsNegative(key K) bool
	Delete(key K)
	Len() (positive int, negative int)
}

type entry[V any] struct {
	value     V
	expiresAt time.Time
	negative  bool
}

type cache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]entry[V]
	cfg     Config
	now     func() time.Time
}

// New constructs a Cache. Panics if cfg.TTL <= 0.
func New[K comparable, V any](cfg Config) Cache[K, V] {
	if cfg.TTL <= 0 {
		panic("atlas-cache: Config.TTL must be > 0")
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &cache[K, V]{
		entries: make(map[K]entry[V]),
		cfg:     cfg,
		now:     now,
	}
}

func (c *cache[K, V]) Get(key K) (V, bool)   { var z V; _ = key; return z, false }
func (c *cache[K, V]) Put(key K, value V)    { _, _ = key, value }
func (c *cache[K, V]) PutNegative(key K)     { _ = key }
func (c *cache[K, V]) IsNegative(key K) bool { _ = key; return false }
func (c *cache[K, V]) Delete(key K)          { _ = key }
func (c *cache[K, V]) Len() (int, int)       { return 0, 0 }
