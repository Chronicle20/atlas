package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// TenantCoalescedRegistry provides tenant-scoped write-coalescing and read-caching
// over Redis. See CoalescedRegistry for the general design.
//
// All public methods accept a tenant.Model to scope keys. This is the tenant-aware
// equivalent of CoalescedRegistry, suitable for registries like monster position or
// character temporal data that are partitioned by tenant.
type TenantCoalescedRegistry[K comparable, V any] struct {
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

// NewTenantCoalescedRegistry creates a tenant-scoped write-coalescing registry.
// See NewCoalescedRegistry for parameter descriptions.
func NewTenantCoalescedRegistry[K comparable, V any](
	client *goredis.Client,
	namespace string,
	keyFn func(K) string,
	flushInterval time.Duration,
	refreshInterval time.Duration,
) *TenantCoalescedRegistry[K, V] {
	r := &TenantCoalescedRegistry[K, V]{
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

func (r *TenantCoalescedRegistry[K, V]) entityKey(t tenant.Model, key K) string {
	return tenantEntityKey(r.namespace, t, r.keyFn(key))
}

func (r *TenantCoalescedRegistry[K, V]) run() {
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

// Get retrieves a value, checking the write buffer first, then the read cache,
// then Redis directly on cache miss.
func (r *TenantCoalescedRegistry[K, V]) Get(ctx context.Context, t tenant.Model, key K) (V, error) {
	rk := r.entityKey(t, key)

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

	r.cacheMu.RLock()
	if v, ok := r.readCache[rk]; ok {
		r.cacheMu.RUnlock()
		return v, nil
	}
	r.cacheMu.RUnlock()

	return r.DirectGet(ctx, t, key)
}

// Put buffers a write locally. The value will be flushed to Redis on the next
// flush cycle. Local reads will see the new value immediately.
func (r *TenantCoalescedRegistry[K, V]) Put(_ context.Context, t tenant.Model, key K, value V) error {
	rk := r.entityKey(t, key)

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
func (r *TenantCoalescedRegistry[K, V]) Remove(_ context.Context, t tenant.Model, key K) error {
	rk := r.entityKey(t, key)

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
func (r *TenantCoalescedRegistry[K, V]) DirectGet(ctx context.Context, t tenant.Model, key K) (V, error) {
	rk := r.entityKey(t, key)
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
func (r *TenantCoalescedRegistry[K, V]) DirectPut(ctx context.Context, t tenant.Model, key K, value V) error {
	rk := r.entityKey(t, key)
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
func (r *TenantCoalescedRegistry[K, V]) DirectUpdate(ctx context.Context, t tenant.Model, key K, fn func(V) V) (V, error) {
	rk := r.entityKey(t, key)

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
func (r *TenantCoalescedRegistry[K, V]) Exists(ctx context.Context, t tenant.Model, key K) (bool, error) {
	rk := r.entityKey(t, key)

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

// GetAllValues returns all values for a tenant by scanning Redis keys. Values
// with pending local writes are served from the write buffer. This is useful for
// iteration tasks like StatusExpirationTask that need to process all entries.
func (r *TenantCoalescedRegistry[K, V]) GetAllValues(ctx context.Context, t tenant.Model) ([]V, error) {
	var result []V
	pattern := tenantScanPattern(r.namespace, t)
	prefix := tenantEntityKey(r.namespace, t, "")
	var cursor uint64

	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan: %w", err)
		}

		if len(keys) > 0 {
			pipe := r.client.Pipeline()
			cmds := make([]*goredis.StringCmd, len(keys))
			for i, k := range keys {
				cmds[i] = pipe.Get(ctx, k)
			}
			_, _ = pipe.Exec(ctx)

			r.writeMu.Lock()
			for i, cmd := range cmds {
				// Skip internal keys.
				entityKeySuffix := strings.TrimPrefix(keys[i], prefix)
				if strings.HasPrefix(entityKeySuffix, "_") {
					continue
				}

				// Serve from write buffer if there's a pending write.
				if op, pending := r.writeBuf[keys[i]]; pending {
					if !op.removed {
						result = append(result, op.value)
					}
					continue
				}

				data, err := cmd.Bytes()
				if errors.Is(err, goredis.Nil) {
					continue
				}
				if err != nil {
					continue
				}
				v, err := r.unmarshal(data)
				if err != nil {
					continue
				}
				result = append(result, v)
			}
			r.writeMu.Unlock()
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

// Flush sends all buffered writes to Redis via pipeline and clears the buffer.
func (r *TenantCoalescedRegistry[K, V]) Flush() {
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
func (r *TenantCoalescedRegistry[K, V]) refresh() {
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
func (r *TenantCoalescedRegistry[K, V]) Shutdown() {
	close(r.stopCh)
	<-r.done
}

// Client returns the underlying Redis client.
func (r *TenantCoalescedRegistry[K, V]) Client() *goredis.Client {
	return r.client
}

// Namespace returns the registry namespace.
func (r *TenantCoalescedRegistry[K, V]) Namespace() string {
	return r.namespace
}
