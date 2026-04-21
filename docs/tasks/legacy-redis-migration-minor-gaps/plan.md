# Redis Migration Minor Gaps — Plan

**Last Updated: 2026-02-19**

## Executive Summary

Two services have residual in-memory patterns that survived the main Redis registry migration:

1. **atlas-messengers** — A process-local `sync.RWMutex` (`createAndJoinLock`) serializes messenger creation and invite flows. This prevents horizontal scaling and contains a latent deadlock bug.
2. **atlas-npc-shops** — A `sync.Once` + `sync.RWMutex` singleton (`ConsumableCache`) caches rechargeable consumable data fetched from the data service. Each instance maintains its own copy independently.

Both are low-risk, bounded changes. No new library features are needed.

---

## Current State Analysis

### atlas-messengers: `createAndJoinLock`

**File:** `services/atlas-messengers/atlas.com/messengers/messenger/processor.go:58`

```go
var createAndJoinLock = sync.RWMutex{}
```

**Usage:** 4 call sites, all using `Lock()`/`Unlock()` (never `RLock`) — it's used as a plain mutex:

| Function | Line | Purpose |
|----------|------|---------|
| `Create` | 63 | Serialize check-then-create for a character's messenger |
| `RequestInvite` | 276 | Serialize invite request + auto-create if actor has no messenger |
| `CreateAndEmit` | 432 | Buffered variant of Create |
| `RequestInviteAndEmit` | 614 | Buffered variant of RequestInvite |

**Race condition prevented:** Without the lock, two concurrent requests could both see `MessengerId() == 0` and both create a messenger for the same character.

**Latent deadlock bug:** `RequestInvite` (line 276) acquires the lock, then calls `Create` (line 309) which also tries to acquire it at line 63. Go's `sync.RWMutex` is not reentrant — this deadlocks. Same pattern in the `*AndEmit` variants. This must be fixed regardless of the Redis migration.

**Existing distributed lock infrastructure:**
- `libs/atlas-redis/lock.go` — `Lock` type with `Acquire(ctx, key)` (single-shot SETNX), `Release(ctx, key)`, `Extend(ctx, key)`. No spin-wait, no ownership tracking.
- `services/atlas-inventory/compartment/lock_registry.go` — `DistributedMutex` with spin-wait (50ms interval, 10s timeout), Lua-based safe release, force-acquire fallback.

### atlas-npc-shops: `ConsumableCache`

**File:** `services/atlas-npc-shops/atlas.com/npc/shops/cache.go`

**Structure:** `sync.Once` singleton containing `map[uuid.UUID][]consumable.Model` protected by `sync.RWMutex`.

**Population:** Lazy-loaded per tenant on first `GetConsumables` call via REST to the atlas-data service (`consumable.NewProcessor(l, ctx).GetRechargeable()`).

**Behavior:** Load-once-and-cache-forever. No TTL, no invalidation, no refresh.

**Serialization challenge:** `consumable.Model` has 50+ unexported fields including nested maps (`map[SpecType]int32`, `map[uint32]uint32`), slices of structs with unexported fields (`[]SummonModel`, `[]RewardModel`). Requires custom `MarshalJSON`/`UnmarshalJSON`.

**Test infrastructure:** `ConsumableCacheInterface` exists. Tests use `mockConsumableCache` via `SetConsumableCacheForTesting`. No direct cache tests exist.

---

## Proposed Future State

### atlas-messengers

Replace the process-local `sync.RWMutex` with a per-character Redis distributed lock. The lock key is scoped to the character being created for (not a global lock), which improves concurrency — two different characters can create messengers simultaneously.

Fix the deadlock bug by extracting internal unlocked variants of `Create`/`CreateAndEmit` that the `RequestInvite`/`RequestInviteAndEmit` functions call while already holding the lock.

### atlas-npc-shops

Replace the in-memory `ConsumableCache` with a Redis-backed `TenantRegistry`. Since the data is static once loaded, use `TenantRegistry` with no TTL. The lazy-load-on-miss pattern remains — `GetConsumables` checks Redis first, falls back to the REST call on cache miss, and populates Redis.

This approach means the first instance to request consumables for a tenant pays the REST cost, and all subsequent instances (or the same instance after restart) read from Redis.

---

## Implementation Phases

### Phase 1: atlas-messengers — Fix Deadlock + Distributed Lock

**Effort:** M

1. **Extract unlocked `create`/`createAndEmit` internal functions** — The `Create` and `CreateAndEmit` functions currently acquire the lock and then do the work. Extract the work into `createInternal`/`createAndEmitInternal` (without lock acquisition). The public `Create`/`CreateAndEmit` call the internal variant under a lock. `RequestInvite`/`RequestInviteAndEmit` call the internal variant while already holding their own lock.

2. **Replace `sync.RWMutex` with Redis distributed lock** — Use the `atlas-redis` `Lock` type. The lock key should be per-character: `messenger:create-lock:{tenantKey}:{characterId}`. Use a short TTL (5-10s). The lock protects the check-then-act pattern per character, not globally.

3. **Spin-wait acquisition** — The `atlas-redis` `Lock.Acquire` is single-shot (returns bool). Add a retry loop in the processor (not the library) with a short timeout (2-3s) and 50ms retry interval. If acquisition fails after timeout, return an error rather than proceeding unlocked.

4. **Update tests** — Processor tests need miniredis setup for the lock. Verify the deadlock is fixed by testing `RequestInvite` when the actor has no messenger (triggers internal create).

### Phase 2: atlas-npc-shops — Redis-Backed Consumable Cache

**Effort:** M

1. **Add `MarshalJSON`/`UnmarshalJSON` to `consumable.Model`** — Create `model_json.go` in the `data/consumable/` package. Handle all unexported fields, nested `SummonModel`, `RewardModel`, and map types. Follow the same pattern used throughout the migration (parallel JSON struct with exported fields).

2. **Replace `ConsumableCache` with Redis-backed implementation** — Replace the `sync.Once` + `sync.RWMutex` + `map` with a `TenantRegistry[uint32, []consumable.Model]` (or a simple Redis GET/SET using the tenant key). Keep the lazy-load-on-miss pattern: check Redis, if miss → REST call → populate Redis.

3. **Update `main.go`** — Initialize the cache with the Redis client (`shops.InitConsumableCache(rc)`). Remove `GetConsumableCache()` singleton.

4. **Add cache tests** — Test the Redis-backed cache with miniredis: cache miss triggers load, cache hit returns stored data, tenant isolation.

5. **Verify existing tests pass** — The `mockConsumableCache` pattern used in processor/REST tests should continue to work since consumers code against `ConsumableCacheInterface`.

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Deadlock fix changes control flow subtly | Low | Medium | Thorough testing of RequestInvite with no existing messenger |
| Redis lock contention under load | Low | Low | Per-character lock key scoping limits contention to same-character races |
| consumable.Model serialization misses a field | Medium | Medium | Round-trip test: marshal → unmarshal → deep-equal |
| ConsumableCache REST fallback on Redis miss adds latency | Low | Low | Same latency as current cold-start; happens once per tenant |

---

## Success Metrics

- All existing tests pass after each change
- `go build` succeeds for both services
- No `sync.Once` or `sync.RWMutex` remains in registry/cache code (infrastructure teardown singletons are exempt)
- atlas-messengers: `RequestInvite` path with no existing messenger works without deadlock
- atlas-npc-shops: consumable data survives service restart (loaded from Redis, not re-fetched)

---

## Dependencies

- `libs/atlas-redis` — Already provides `Lock` type (no changes needed)
- `miniredis` — Already a test dependency in both services
- No new Go module dependencies required
