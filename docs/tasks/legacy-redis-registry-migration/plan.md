# Redis Registry Migration Plan

**Last Updated: 2026-02-17**

## Executive Summary

Replace the `sync.Once` + `map` + `sync.RWMutex` singleton registry pattern used across 36 registries in 22 services with a shared Redis-backed registry library (`atlas-redis`). This enables horizontal scaling by externalizing runtime-mutable state to a shared data store.

**Scope:**
- Build a new shared library: `libs/atlas-redis` (module: `github.com/Chronicle20/atlas-redis`)
- Migrate 36 registries across 22 services
- 6 high-throughput registries across 4 services are **deferred** (separate optimization work)
- 5 registries in atlas-login/atlas-channel are **exempt**
- 1 registry (atlas-saga-orchestrator) migrates to **PostgreSQL** (separate work)
- 26 read-only registries are **skipped** (no migration needed)

**Estimated Total Effort:** XL (multi-sprint initiative)

---

## Current State Analysis

### The Singleton Pattern

Every affected service follows the identical pattern:

```go
var registry *Registry
var once sync.Once

func GetRegistry() *Registry {
    once.Do(func() {
        registry = &Registry{}
        registry.someMap = make(map[tenant.Model]map[uint32]Model)
    })
    return registry
}
```

State lives in Go `map` types with `sync.RWMutex` or `sync.Mutex` protection. Tenant isolation uses `tenant.Model` as the top-level map key. Per-tenant sub-locks are common.

### Pattern Taxonomy

From analysis of all 36 registries, there are 8 distinct patterns the library must support:

| Pattern | Count | Example Services |
|---------|-------|------------------|
| Simple tenant-keyed CRUD | 14 | buffs, chairs, chalkboards, rates, effective-stats |
| Auto-increment + tenant-keyed CRUD | 4 | parties, invites, messengers, drops, reactors |
| TTL-based entries | 11 | expressions (5s), cashshop (5min), inventory reservations, cooldowns |
| Secondary/reverse indexes | 4 | parties (char→party), npc-shops (shop→chars), party-quests (char→instance) |
| Composite-key registries | 4 | maps/character, chairs/character, chalkboards/character |
| Global ID space (non-tenant) | 2 | drops, reactors |
| Distributed locks | 1 | inventory lock_registry |
| Saga cross-reference | 2 | npc-conversations (sagaId lookup), portal-actions (sagaId→pending) |

### Existing Shared Library Conventions

- Module path: `github.com/Chronicle20/atlas-<name>`
- Located in: `libs/atlas-<name>/`
- `go.mod` at library root
- Services import via module path, resolved locally via `go.work`
- Versioned with semver tags (e.g., `v1.2.16`)

### Tenant Key Design

`tenant.Model` is a value type used as map keys:
```go
type Model struct {
    id           uuid.UUID
    region       string
    majorVersion uint16
    minorVersion uint16
}
```

For Redis keys, serialize as: `{uuid}:{region}:{major}.{minor}`

---

## Proposed Future State

### Library: `libs/atlas-redis`

A shared Go library providing Redis-backed implementations of common registry operations. The library should:

1. **Be a drop-in replacement** for the existing `GetRegistry()` singleton pattern
2. **Handle tenant-scoped key namespacing** automatically
3. **Support all 8 identified patterns** via composable building blocks
4. **Provide JSON serialization** for model storage (consistent with existing JSON:API patterns)
5. **Use the `go-redis/redis` client** (the Go standard for Redis)
6. **Support connection configuration** via environment variables (consistent with existing `DATABASE_URL`, `BOOTSTRAP_SERVERS` patterns)

### Package Structure

```
libs/atlas-redis/
├── go.mod                    # github.com/Chronicle20/atlas-redis
├── registry.go               # Core Registry[K, V] generic type
├── options.go                # Configuration and functional options
├── keys.go                   # Key serialization (tenant, composite keys)
├── id.go                     # Auto-increment ID generation (INCR)
├── ttl.go                    # TTL support (EXPIRE, expiration scanning)
├── lock.go                   # Distributed locking (SET NX EX)
├── index.go                  # Secondary index maintenance
├── connection.go             # Redis connection management
└── registry_test.go          # Tests with miniredis
```

### Core API Design

```go
// Registry is the core generic type replacing map-based registries.
type Registry[K comparable, V any] struct {
    client    *redis.Client
    namespace string           // service-scoped prefix
    keyFn     func(K) string   // serialize K to Redis key suffix
    marshal   func(V) ([]byte, error)
    unmarshal func([]byte) (V, error)
}

// Standard CRUD operations
func (r *Registry[K, V]) Get(ctx context.Context, key K) (V, error)
func (r *Registry[K, V]) GetAll(ctx context.Context) (map[K]V, error)
func (r *Registry[K, V]) Put(ctx context.Context, key K, value V) error
func (r *Registry[K, V]) PutWithTTL(ctx context.Context, key K, value V, ttl time.Duration) error
func (r *Registry[K, V]) Remove(ctx context.Context, key K) error
func (r *Registry[K, V]) Update(ctx context.Context, key K, fn func(V) V) (V, error)
func (r *Registry[K, V]) Exists(ctx context.Context, key K) (bool, error)

// Tenant-scoped convenience
func NewTenantRegistry[V any](client *redis.Client, namespace string, opts ...Option) *TenantRegistry[V]

// Auto-increment ID generation
func (r *TenantRegistry[V]) NextID(ctx context.Context, t tenant.Model) (uint32, error)

// Secondary index
func NewIndex[K comparable, V comparable](client *redis.Client, namespace string) *Index[K, V]
func (i *Index[K, V]) Add(ctx context.Context, key K, value V) error
func (i *Index[K, V]) Remove(ctx context.Context, key K, value V) error
func (i *Index[K, V]) Lookup(ctx context.Context, key K) ([]V, error)

// Distributed lock
func NewLock(client *redis.Client, namespace string) *Lock
func (l *Lock) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)
func (l *Lock) Release(ctx context.Context, key string) error
```

### Redis Key Schema

```
atlas:{namespace}:{tenant_key}:{entity_key}      # Main data
atlas:{namespace}:{tenant_key}:_id                # Auto-increment counter
atlas:{namespace}:{tenant_key}:_idx:{index_name}:{index_key}  # Secondary index
atlas:{namespace}:_lock:{lock_key}                # Distributed lock
```

Example for atlas-parties:
```
atlas:party:abc-def:GMS:83.1:1000000001          # Party model (JSON)
atlas:party:abc-def:GMS:83.1:_id                  # Party ID counter
atlas:party:abc-def:GMS:83.1:_idx:char:42         # Character 42's party ID
```

### Infrastructure

- Add Redis to Kubernetes manifests (Deployment + Service)
- Add `REDIS_URL` to `atlas-env.yaml` shared ConfigMap
- Use single Redis instance initially (no clustering needed for current scale)

---

## Implementation Phases

### Phase 1: Foundation — Shared Library + Infrastructure (Effort: L)

Build `libs/atlas-redis` and deploy Redis infrastructure.

**1.1 Create atlas-redis library skeleton**
- Effort: M
- Create `libs/atlas-redis/` with `go.mod` (module `github.com/Chronicle20/atlas-redis`)
- Add `go-redis/redis/v9` dependency
- Add `alicebob/miniredis/v2` for testing
- Add to `go.work`
- **Acceptance:** `go build ./libs/atlas-redis/...` passes

**1.2 Implement connection management**
- Effort: S
- `connection.go`: Read `REDIS_URL` from env (default `localhost:6379`)
- Support password, DB selection, pool size configuration
- Provide `Connect(logger) *redis.Client` matching existing database connection pattern
- **Acceptance:** Can connect to Redis, logs connection status

**1.3 Implement key serialization**
- Effort: S
- `keys.go`: `TenantKey(t tenant.Model) string` → `"{uuid}:{region}:{major}.{minor}"`
- Composite key builders for multi-field keys
- Namespace prefixing
- **Acceptance:** Deterministic, reversible key serialization with tests

**1.4 Implement core Registry[K, V]**
- Effort: L
- `registry.go`: Generic `Registry[K, V]` with Get, GetAll, Put, Remove, Update, Exists
- JSON marshal/unmarshal via `encoding/json` (models already have JSON tags)
- Pipeline support for batch operations
- **Acceptance:** Full CRUD test suite passing against miniredis

**1.5 Implement TenantRegistry convenience layer**
- Effort: M
- `registry.go`: `TenantRegistry[V]` wrapping `Registry` with automatic tenant key scoping
- Matches the dominant `map[tenant.Model]map[entityId]Model` pattern
- **Acceptance:** Tenant-scoped CRUD tests passing

**1.6 Implement auto-increment ID generation**
- Effort: S
- `id.go`: Per-tenant counters using Redis `INCR` on `{namespace}:{tenant}:_id`
- Configurable start value (default 1000000000 to match existing pattern)
- **Acceptance:** Concurrent ID generation produces unique, monotonically increasing IDs

**1.7 Implement TTL support**
- Effort: S
- `ttl.go`: `PutWithTTL` sets Redis `EXPIRE`
- `GetExpired` pattern using sorted sets (score = expiration timestamp)
- **Acceptance:** Entries auto-expire; GetExpired returns expired entries before removal

**1.8 Implement secondary index support**
- Effort: M
- `index.go`: Redis SET-based indexes for reverse lookups
- Atomic add/remove with main registry operations
- **Acceptance:** Index stays consistent with registry through Put/Remove operations

**1.9 Implement distributed locking**
- Effort: S
- `lock.go`: `SET key value NX EX ttl` pattern
- Auto-release on context cancellation
- Configurable TTL with sensible defaults
- **Acceptance:** Concurrent lock acquisition is mutually exclusive; locks auto-expire

**1.10 Deploy Redis infrastructure**
- Effort: S
- Kubernetes Deployment + Service for Redis
- Add `REDIS_URL` to `atlas-env.yaml`
- **Acceptance:** Redis accessible from all services in the cluster

### Phase 2: Pilot Migration — Low-Risk Services (Effort: L)

Migrate simple, low-traffic services first to validate the library and establish migration patterns.

**Target services (8 registries, 6 services):**

| # | Service | Registry | Pattern | Effort |
|---|---------|----------|---------|--------|
| 2.1 | atlas-chairs | chair/registry.go | Simple CRUD (charId→Model) | S |
| 2.2 | atlas-chairs | character/registry.go | Composite key (MapKey→[]charId) | S |
| 2.3 | atlas-chalkboards | chalkboard/registry.go | Simple CRUD (ChalkboardKey→string) | S |
| 2.4 | atlas-chalkboards | character/registry.go | Composite key (MapKey→[]charId) | S |
| 2.5 | atlas-expressions | expression/registry.go | TTL (5s expiration) | S |
| 2.6 | atlas-consumables | character/registry.go | Simple CRUD (charId→MapKey) | S |
| 2.7 | atlas-portals | blocked/cache.go | Nested map (tenant→char→portal→bool) | S |
| 2.8 | atlas-portal-actions | action/registry.go | Simple CRUD (sagaId→PendingAction) | S |

**Migration pattern per service:**
1. Add `github.com/Chronicle20/atlas-redis` to service's `go.mod`
2. Replace `sync.Once` singleton with Redis-backed `TenantRegistry` injected at startup
3. Update all callers to pass `context.Context` (for Redis operations)
4. Update tests to use miniredis
5. Run `go test ./... -count=1`
6. Run `go build`

**Acceptance per service:** All existing tests pass with Redis backend. No behavioral changes to REST/Kafka interfaces.

### Phase 3: Core Game State Migration (Effort: XL)

Migrate critical game-state services. These are higher risk and require careful testing.

**Target services (16 registries, 10 services):**

| # | Service | Registry | Pattern | Effort |
|---|---------|----------|---------|--------|
| 3.1 | atlas-buffs | character/registry.go | TTL + complex queries (immunity, poison) | M |
| 3.2 | atlas-skills | cooldown_registry.go | TTL (cooldown timestamps) | S |
| 3.3 | atlas-effective-stats | character/registry.go | Tenant CRUD + bonus stacking | M |
| 3.4 | atlas-rates | character/registry.go | Tenant CRUD + factor management | M |
| 3.5 | atlas-rates | character/item_tracker.go | TTL (coupon expiration) | S |
| 3.6 | atlas-rates | character/initializer.go | Simple bool tracker | S |
| 3.7 | atlas-npc-conversations | conversation/registry.go | Tenant CRUD + saga cross-ref | M |
| 3.8 | atlas-npc-shops | shops/registry.go | Dual map + reverse index | M |
| 3.9 | atlas-storage | storage/cache.go | TTL (NPC context) | S |
| 3.10 | atlas-storage | projection/manager.go | sync.Map replacement | S |
| 3.11 | atlas-character | session/registry.go | Tenant CRUD + age tracking | M |
| 3.12 | atlas-pets | character/registry.go | Simple CRUD (charId→MapKey) | S |
| 3.13 | atlas-character-factory | factory/cache.go | Two stores + saga tracking | M |
| 3.14 | atlas-cashshop | reservation/cache.go | TTL (5min) + background cleanup | S |
| 3.15 | atlas-consumables | character/registry.go | Moved to Phase 2 if simple enough | S |
| 3.16 | atlas-account | account/registry.go | State machine + expiration | L |

### Phase 4: Social & Coordination Services (Effort: L)

Migrate services with auto-incrementing IDs and secondary indexes.

| # | Service | Registry | Pattern | Effort |
|---|---------|----------|---------|--------|
| 4.1 | atlas-parties | party/registry.go | Auto-inc ID + char→party index | M |
| 4.2 | atlas-parties | character/registry.go | Tenant CRUD | S |
| 4.3 | atlas-invites | invite/registry.go | Auto-inc ID + TTL + nested keys | M |
| 4.4 | atlas-messengers | messenger/registry.go | Auto-inc ID | M |
| 4.5 | atlas-messengers | character/registry.go | Tenant CRUD | S |
| 4.6 | atlas-guilds | coordinator/registry.go | TTL + agreement tracking | M |

### Phase 5: Game World State (Effort: L)

Migrate services managing game world objects. These have the most complex registries.

| # | Service | Registry | Pattern | Effort |
|---|---------|----------|---------|--------|
| 5.1 | atlas-drops | drop/registry.go | Global atomic ID + per-entity locks + reservations + map index | L |
| 5.2 | atlas-reactors | reactor/registry.go | Global running ID + cooldowns + map index + per-map locks | L |
| 5.3 | atlas-party-quests | instance/registry.go | char→instance index + complex state | L |
| 5.4 | atlas-transports | instance/instance_registry.go | Transport state + character tracking | M |
| 5.5 | atlas-transports | instance/character_registry.go | Character→transport mapping | S |
| 5.6 | atlas-transports | channel/registry.go | Channel tracking | S |
| 5.7 | atlas-world | channel/registry.go | Channel registration | M |
| 5.8 | atlas-world | rate/registry.go | World rates | S |

### Phase 6: Inventory & Locking (Effort: M)

Migrate the distributed locking and reservation patterns. Highest correctness risk.

| # | Service | Registry | Pattern | Effort |
|---|---------|----------|---------|--------|
| 6.1 | atlas-inventory | reservation_registry.go | TTL reservations + swap + bulk cleanup | L |
| 6.2 | atlas-inventory | lock_registry.go | sync.Map → Redis distributed locks | M |

---

## Risk Assessment and Mitigation

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| Redis single point of failure | High | Medium | Deploy Redis with persistence (RDB snapshots). Plan for Redis Sentinel/HA in future. Game state is recoverable (players reconnect). |
| Data loss during migration | High | Low | No migration of existing data needed — registries are ephemeral. Service restart rebuilds state from Kafka/client reconnection. |
| Latency regression for hot paths | Medium | Medium | Benchmark before/after per service. Use Redis pipelining for batch operations. High-throughput services deferred to separate work. |
| Context propagation gaps | Medium | Medium | Some code paths may not have `context.Context` available. Provide fallback `context.Background()` but log warnings. |
| JSON serialization overhead | Low | Medium | Models already have JSON tags. Benchmark serialization cost. Consider msgpack if JSON proves too slow. |
| Test infrastructure | Low | Low | Use `alicebob/miniredis/v2` for unit tests — no external Redis dependency. |
| Concurrent migration across services | Medium | Medium | Phase services in dependency order. Pilot with low-risk services first. Each service can be migrated independently. |

---

## Success Metrics

1. **All 36 registries migrated** — No service uses `sync.Once` + `map` singleton for runtime-mutable state (except exempt and deferred)
2. **All existing tests pass** — Zero behavioral regressions
3. **Horizontal scaling verified** — At least one migrated service runs with 2+ replicas under test load
4. **Latency within bounds** — P99 latency increase < 5ms for any API endpoint
5. **Zero data corruption** — No item duplication, lost invites, or orphaned state during testing

---

## Required Resources and Dependencies

### New Dependencies
- `github.com/redis/go-redis/v9` — Redis client
- `github.com/alicebob/miniredis/v2` — In-memory Redis for tests
- Redis server (Kubernetes deployment)

### Infrastructure
- Redis Kubernetes Deployment + Service in `atlas` namespace
- `REDIS_URL` environment variable in `atlas-env.yaml`

### No Changes Required
- No changes to Kafka topology
- No changes to REST API contracts
- No changes to ingress routing
- No changes to database schemas

---

## Timeline Estimates

| Phase | Description | Effort | Dependencies |
|-------|-------------|--------|--------------|
| Phase 1 | Foundation — Library + Infrastructure | L | None |
| Phase 2 | Pilot — 8 registries, 6 services | L | Phase 1 |
| Phase 3 | Core Game State — 16 registries, 10 services | XL | Phase 2 (patterns validated) |
| Phase 4 | Social & Coordination — 6 registries, 4 services | L | Phase 1 |
| Phase 5 | Game World State — 8 registries, 5 services | L | Phase 1 |
| Phase 6 | Inventory & Locking — 2 registries, 1 service | M | Phase 1 |

Phases 3-6 can be parallelized after Phase 2 validates the migration pattern. Phase 6 should be last due to highest correctness risk (distributed locking).
