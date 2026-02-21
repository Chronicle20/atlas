package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// writeOp represents a pending write or delete in the coalescing buffer.
type writeOp[V any] struct {
	value   V
	removed bool
}

// CoalescedRegistry provides a write-coalescing, read-caching layer over Redis.
//
// High-frequency writes are buffered locally and flushed to Redis periodically
// via pipeline. Reads are served from a local cache that is refreshed periodically
// from Redis, enabling cross-instance visibility with bounded staleness.
//
// Designed for high-throughput data (character positions, monster movement) where
// individual Redis round-trips per write would be too expensive but bounded
// staleness (50-200ms) is acceptable.
//
// For correctness-critical operations (damage, HP), use the Direct* methods
// which bypass the buffer/cache and operate on Redis immediately.
type CoalescedRegistry[K comparable, V any] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
	marshal   func(V) ([]byte, error)
	unmarshal func([]byte) (V, error)

	writeBuf map[string]writeOp[V]
	writeMu  sync.Mutex

	readCache map[string]V
	cacheMu   sync.RWMutex

	flushInterval   time.Duration
	refreshInterval time.Duration

	stopCh chan struct{}
	done   chan struct{}
}

// NewCoalescedRegistry creates a new write-coalescing registry and starts
// background goroutines for flushing writes and refreshing the read cache.
//
// flushInterval controls how often buffered writes are sent to Redis.
// refreshInterval controls how often the local read cache is updated from Redis.
//
// Call Shutdown() when the registry is no longer needed to stop background
// goroutines and perform a final flush.
func NewCoalescedRegistry[K comparable, V any](
	client *goredis.Client,
	namespace string,
	keyFn func(K) string,
	flushInterval time.Duration,
	refreshInterval time.Duration,
) *CoalescedRegistry[K, V] {
	r := &CoalescedRegistry[K, V]{
		client:          client,
		namespace:       namespace,
		keyFn:           keyFn,
		marshal:         func(v V) ([]byte, error) { return json.Marshal(v) },
		unmarshal:       func(data []byte) (V, error) { var v V; err := json.Unmarshal(data, &v); return v, err },
		writeBuf:        make(map[string]writeOp[V]),
		readCache:       make(map[string]V),
		flushInterval:   flushInterval,
		refreshInterval: refreshInterval,
		stopCh:          make(chan struct{}),
		done:            make(chan struct{}),
	}
	go r.run()
	return r
}

func (r *CoalescedRegistry[K, V]) redisKey(key K) string {
	return namespacedKey(r.namespace, r.keyFn(key))
}

func (r *CoalescedRegistry[K, V]) run() {
	defer close(r.done)

	flushTicker := time.NewTicker(r.flushInterval)
	refreshTicker := time.NewTicker(r.refreshInterval)
	defer flushTicker.Stop()
	defer refreshTicker.Stop()

	for {
		select {
		case <-flushTicker.C:
			r.Flush()
		case <-refreshTicker.C:
			r.refresh()
		case <-r.stopCh:
			r.Flush()
			return
		}
	}
}

// Get retrieves a value, checking the write buffer first (for pending local
// writes), then the read cache (for cross-instance data), then Redis directly
// on cache miss. Returns ErrNotFound if the key does not exist anywhere.
func (r *CoalescedRegistry[K, V]) Get(ctx context.Context, key K) (V, error) {
	rk := r.redisKey(key)

	// Check write buffer first (our own pending writes).
	r.writeMu.Lock()
	if op, ok := r.writeBuf[rk]; ok {
		r.writeMu.Unlock()
		if op.removed {
			var zero V
			return zero, ErrNotFound
		}
		return op.value, nil
	}
	r.writeMu.Unlock()

	// Check read cache.
	r.cacheMu.RLock()
	if v, ok := r.readCache[rk]; ok {
		r.cacheMu.RUnlock()
		return v, nil
	}
	r.cacheMu.RUnlock()

	// Cache miss: fetch from Redis and populate cache.
	return r.DirectGet(ctx, key)
}

// Put buffers a write locally. The value will be flushed to Redis on the next
// flush cycle. Local reads will see the new value immediately.
func (r *CoalescedRegistry[K, V]) Put(_ context.Context, key K, value V) error {
	rk := r.redisKey(key)

	r.writeMu.Lock()
	r.writeBuf[rk] = writeOp[V]{value: value}
	r.writeMu.Unlock()

	r.cacheMu.Lock()
	r.readCache[rk] = value
	r.cacheMu.Unlock()

	return nil
}

// Remove marks a key for deletion. The delete will be flushed to Redis on the
// next flush cycle. Local reads will return ErrNotFound immediately.
func (r *CoalescedRegistry[K, V]) Remove(_ context.Context, key K) error {
	rk := r.redisKey(key)

	r.writeMu.Lock()
	var zero V
	r.writeBuf[rk] = writeOp[V]{value: zero, removed: true}
	r.writeMu.Unlock()

	r.cacheMu.Lock()
	delete(r.readCache, rk)
	r.cacheMu.Unlock()

	return nil
}

// DirectGet bypasses the buffer and cache, reading directly from Redis.
// The result is stored in the read cache for subsequent cached reads.
func (r *CoalescedRegistry[K, V]) DirectGet(ctx context.Context, key K) (V, error) {
	rk := r.redisKey(key)
	data, err := r.client.Get(ctx, rk).Bytes()
	if errors.Is(err, goredis.Nil) {
		var zero V
		return zero, ErrNotFound
	}
	if err != nil {
		var zero V
		return zero, fmt.Errorf("redis get: %w", err)
	}
	v, err := r.unmarshal(data)
	if err != nil {
		var zero V
		return zero, err
	}

	r.cacheMu.Lock()
	r.readCache[rk] = v
	r.cacheMu.Unlock()

	return v, nil
}

// DirectPut writes directly to Redis, bypassing the write buffer.
// Also updates the local read cache.
func (r *CoalescedRegistry[K, V]) DirectPut(ctx context.Context, key K, value V) error {
	rk := r.redisKey(key)
	data, err := r.marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err = r.client.Set(ctx, rk, data, 0).Err(); err != nil {
		return err
	}

	r.cacheMu.Lock()
	r.readCache[rk] = value
	r.cacheMu.Unlock()

	return nil
}

// DirectUpdate performs an atomic read-modify-write on Redis using WATCH/MULTI.
// Bypasses the write buffer entirely. Updates the local read cache on success.
func (r *CoalescedRegistry[K, V]) DirectUpdate(ctx context.Context, key K, fn func(V) V) (V, error) {
	rk := r.redisKey(key)

	var result V
	err := r.client.Watch(ctx, func(tx *goredis.Tx) error {
		data, err := tx.Get(ctx, rk).Bytes()
		if errors.Is(err, goredis.Nil) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		current, err := r.unmarshal(data)
		if err != nil {
			return err
		}
		result = fn(current)
		newData, err := r.marshal(result)
		if err != nil {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.Set(ctx, rk, newData, 0)
			return nil
		})
		return err
	}, rk)

	if err == nil {
		r.cacheMu.Lock()
		r.readCache[rk] = result
		r.cacheMu.Unlock()
	}

	return result, err
}

// Exists checks whether a key exists in the write buffer, read cache, or Redis.
func (r *CoalescedRegistry[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	rk := r.redisKey(key)

	r.writeMu.Lock()
	if op, ok := r.writeBuf[rk]; ok {
		r.writeMu.Unlock()
		return !op.removed, nil
	}
	r.writeMu.Unlock()

	r.cacheMu.RLock()
	_, ok := r.readCache[rk]
	r.cacheMu.RUnlock()
	if ok {
		return true, nil
	}

	n, err := r.client.Exists(ctx, rk).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}

// Flush sends all buffered writes to Redis via pipeline and clears the buffer.
// Called automatically by the background goroutine and on Shutdown.
func (r *CoalescedRegistry[K, V]) Flush() {
	r.writeMu.Lock()
	if len(r.writeBuf) == 0 {
		r.writeMu.Unlock()
		return
	}
	buf := r.writeBuf
	r.writeBuf = make(map[string]writeOp[V], len(buf))
	r.writeMu.Unlock()

	ctx := context.Background()
	pipe := r.client.Pipeline()
	for rk, op := range buf {
		if op.removed {
			pipe.Del(ctx, rk)
		} else {
			data, err := r.marshal(op.value)
			if err != nil {
				continue
			}
			pipe.Set(ctx, rk, data, 0)
		}
	}
	_, _ = pipe.Exec(ctx)
}

// refresh updates the local read cache from Redis for all currently cached keys.
// This picks up writes from other instances.
func (r *CoalescedRegistry[K, V]) refresh() {
	r.cacheMu.RLock()
	keys := make([]string, 0, len(r.readCache))
	for k := range r.readCache {
		keys = append(keys, k)
	}
	r.cacheMu.RUnlock()

	if len(keys) == 0 {
		return
	}

	ctx := context.Background()
	pipe := r.client.Pipeline()
	cmds := make([]*goredis.StringCmd, len(keys))
	for i, k := range keys {
		cmds[i] = pipe.Get(ctx, k)
	}
	_, _ = pipe.Exec(ctx)

	r.cacheMu.Lock()
	r.writeMu.Lock()
	for i, cmd := range cmds {
		// Don't overwrite keys with pending local writes.
		if _, pending := r.writeBuf[keys[i]]; pending {
			continue
		}

		data, err := cmd.Bytes()
		if errors.Is(err, goredis.Nil) {
			delete(r.readCache, keys[i])
			continue
		}
		if err != nil {
			continue
		}
		v, err := r.unmarshal(data)
		if err != nil {
			continue
		}
		r.readCache[keys[i]] = v
	}
	r.writeMu.Unlock()
	r.cacheMu.Unlock()
}

// Shutdown stops background goroutines and performs a final flush.
// Blocks until the background goroutine has exited.
func (r *CoalescedRegistry[K, V]) Shutdown() {
	close(r.stopCh)
	<-r.done
}

// Client returns the underlying Redis client.
func (r *CoalescedRegistry[K, V]) Client() *goredis.Client {
	return r.client
}

// Namespace returns the registry namespace.
func (r *CoalescedRegistry[K, V]) Namespace() string {
	return r.namespace
}
