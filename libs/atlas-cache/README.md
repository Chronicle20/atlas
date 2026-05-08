# atlas-cache

Generic in-process TTL cache with distinct positive and negative entry
lifetimes. Concurrency-safe; lazy expiration on read; no background
goroutine; no I/O dependencies.

Intended for read-heavy reference data fronting an upstream HTTP service
(e.g. `data/monsters/{id}` lookups). Cache state is **per-process** and is
**lost on restart** — that is the supported invalidation mechanism for
upstream redeploys in v1. There is no admin endpoint, no event-driven
flush, and no cross-pod coherence.

## Usage

```go
import "github.com/Chronicle20/atlas/libs/atlas-cache"

c := cache.New[uint32, MyModel](cache.Config{
    TTL:         5 * time.Minute,
    NegativeTTL: 30 * time.Second,
})

if v, ok := c.Get(id); ok {
    return v // hit
}
if c.IsNegative(id) {
    return zero, ErrNotFound // negative hit
}
v, err := upstream(id)
if err == nil { c.Put(id, v) } else if isNotFound(err) { c.PutNegative(id) }
```

## Operability

- `Config.OnEviction(kind)` is called under the write lock when a lazy
  expiration purges an entry. Use it to bump a Prometheus counter.
- `Config.Now` is the clock function (defaults to `time.Now`). Inject a
  fake for deterministic tests.
- `NegativeTTL == 0` disables negative caching (`PutNegative` is a no-op,
  `IsNegative` always returns false).
- `Len() (positive, negative int)` is O(n); call it only for sampled
  metrics, not on hot paths.
