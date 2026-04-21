# Redis Registry Migration — Context

**Last Updated: 2026-02-17**

## Current Status

**Phase 1:** COMPLETE (1.1-1.6 done; 1.7-1.9 deferred to later phases)
**Phase 2:** COMPLETE (all 8 registries migrated)
**Phase 3:** COMPLETE (all registries migrated; 3.14 skipped — unused cache)
**Phase 4:** COMPLETE (all 6 registries migrated)
**Phase 5:** COMPLETE (all registries migrated; 5.3 skipped — service doesn't exist; 5.7/5.8 already Redis-backed)
**Phase 6:** COMPLETE (both registries migrated)

### This Session's Completed Work
- **5.2 atlas-reactors** — Full migration completed
  - Created JSON serialization for nested type hierarchy: item, point, state, data, reactor model_json.go (5 new files)
  - Rewritten registry.go with Redis: reactor:{id} JSON, map SET index, global SET, INCR+Lua wraparound ID
  - Cooldowns use Redis TTL keys (no more background cleanup task needed)
  - CleanupExpiredCooldowns() is now a no-op (Redis TTL handles expiration)
  - Updated processor_test.go: miniredis-based setup, removed cleanupRegistry()
  - Updated main.go: added atlas.Connect(l) + reactor.InitRegistry(rc)
  - All tests pass, build clean
- **6.1 atlas-inventory reservation_registry.go** — Redis-backed reservations
  - Per-slot reservation keys: `reservation:{tenantId}:{characterId}:{inventoryType}:{slot}` → JSON array
  - Expiry filtering done on read in GetReservedQuantity (no background cleanup needed)
  - RemoveAllReservationsForCharacter uses SCAN pattern
  - SwapReservation reads/writes two keys
- **6.2 atlas-inventory lock_registry.go** — Redis distributed locks
  - `DistributedMutex` with `Lock()/Unlock()` using `SET NX EX` + Lua unlock script
  - Lock keys: `invlock:{characterId}:{inventoryType}` with 30s TTL
  - Spin-wait with 50ms retry, 10s timeout, force-acquire fallback
  - All 19 call sites in processor.go work without modification (they use `.Lock()`/`.Unlock()`)
  - TestMain setup in processor_test.go initializes both registries with miniredis

### All Phases Complete
The main migration plan (Phases 1-6) is now complete. Remaining work tracked in "Out of Scope":
- HIGH-THROUGHPUT services (atlas-monsters, atlas-character temporal_data, atlas-pets temporal_data, atlas-maps)
- POSTGRESQL migration for atlas-saga-orchestrator
- EXEMPT services (atlas-login, atlas-channel)

---

## Key Architectural Decisions

### 1. Redis over Kafka Partitioning
Kafka key-based partitioning was considered but rejected because:
- It solves scaling but not redundancy (instance death loses state)
- Requires event sourcing or snapshot recovery on rebalance
- REST routing still needs partition-aware load balancing
- Redis is a proven shared state store vs. building a custom distributed system

### 2. Single Redis Instance (Initial)
Start with a single Redis instance. The current architecture runs one instance per service already, so Redis is an improvement even without HA. Plan for Redis Sentinel when needed.

### 3. atlas-login and atlas-channel Exempt
These services hold `net.Conn` (live TCP sockets) that are physically process-local. They cannot be externalized. atlas-login has light load; atlas-channel scales by running one instance per game channel.

### 4. Saga Orchestrator → PostgreSQL (Separate Work)
Sagas need transactional guarantees (durability, crash recovery) better served by PostgreSQL than Redis. This is tracked as separate work.

### 5. High-Throughput Services Deferred
Monster position, character position, pet position, and spawn point tracking have update frequencies where naive per-operation Redis calls could create throughput pressure. These are documented in `docs/high-throughput-cache-problem.md` and will use optimized strategies (pipelining, write coalescing, Lua scripts).

### 6. TenantRegistry API Pattern (KEY — learned during migration)
The `TenantRegistry[K, V]` methods require **explicit tenant.Model parameter**:
```go
func (r *TenantRegistry[K, V]) Get(ctx context.Context, t tenant.Model, key K) (V, error)
func (r *TenantRegistry[K, V]) Put(ctx context.Context, t tenant.Model, key K, value V) error
```
Registry wrappers extract tenant via `tenant.MustFromContext(ctx)` in each method and pass it:
```go
func (r *Registry) Get(ctx context.Context, characterId uint32) (Model, error) {
    t := tenant.MustFromContext(ctx)
    return r.characters.Get(ctx, t, characterId)
}
```

### 7. channel.Model Serialization
`channel.Model` from atlas-constants has unexported fields and NO JSON methods. Serialize worldId/channelId separately and reconstruct:
```go
m.ch = channel.NewModel(aux.WorldId, aux.ChannelId)
```

### 8. Composite Key Pattern for Redis
For multi-field keys (like characterId:templateId in item_tracker), use a composite key struct:
```go
type itemTrackerKey struct { CharacterId uint32; TemplateId uint32 }
```
With string formatter: `"charId:templateId"`

### 9. SCAN-Based Filtering
When TenantRegistry.GetAllValues returns ALL items for a tenant but you need to filter by a key prefix (e.g., items for a specific character), use Redis SCAN with pattern:
```go
scanPattern := "atlas:" + t.items.Namespace() + ":" + atlas.TenantKey(ten) + ":" + charIdStr + ":*"
```
Use `t.items.Client()` and `t.items.Namespace()` (exported from TenantRegistry).

### 10. go.mod Rules
- **NEVER add `github.com/Chronicle20/atlas-redis` to service go.mod files** — `go.work` handles local resolution
- DO add `github.com/redis/go-redis/v9` and `github.com/alicebob/miniredis/v2` as direct deps
- Import atlas-redis as: `atlas "github.com/Chronicle20/atlas-redis"`

---

## Key Files — Shared Library References

| File | Purpose | Why Relevant |
|------|---------|--------------|
| `libs/atlas-redis/tenant_registry.go` | TenantRegistry[K,V] generic type | Core abstraction; Get/Put/Remove/GetAllValues/Update/Exists + Client()/Namespace() |
| `libs/atlas-redis/keys.go` | TenantKey(tenant.Model) string | Key format: `atlas:{namespace}:{tenantKey}:{entityKey}` |
| `libs/atlas-redis/connection.go` | Connect(logger) *redis.Client | Reads REDIS_URL from env |
| `libs/atlas-redis/errors.go` | ErrNotFound sentinel error | Used for not-found mapping in registry wrappers |
| `libs/atlas-tenant/processor.go` | tenant.MustFromContext(ctx) | Extract tenant from context in registry methods |
| `go.work` | Go workspace | Contains `./libs/atlas-redis` entry |

---

## Migration Catalog Summary

### MIGRATE (36 registries, 22 services)

| Service | Registries | Key Patterns |
|---------|------------|--------------|
| atlas-account | 1 (account/registry.go) | State machine + expiration |
| atlas-buffs | 1 (character/registry.go) | TTL + complex queries |
| atlas-cashshop | 1 (reservation/cache.go) | TTL (5min) |
| atlas-chairs | 2 (chair + character registries) | Simple CRUD + composite key |
| atlas-chalkboards | 2 (chalkboard + character registries) | Simple CRUD + composite key |
| atlas-character | 1 (session/registry.go) | Tenant CRUD |
| atlas-character-factory | 1 (factory/cache.go) | Two stores + saga tracking |
| atlas-consumables | 1 (character/registry.go) | Simple CRUD |
| atlas-drops | 1 (drop/registry.go) | Global ID + locks + reservations |
| atlas-effective-stats | 1 (character/registry.go) | Tenant CRUD + bonus stacking |
| atlas-expressions | 1 (expression/registry.go) | TTL (5s) |
| atlas-guilds | 1 (coordinator/registry.go) | TTL + agreement flow |
| atlas-invites | 1 (invite/registry.go) | Auto-inc ID + TTL + nested keys |
| atlas-messengers | 2 (messenger + character registries) | Auto-inc ID + CRUD |
| atlas-npc-conversations | 1 (conversation/registry.go) | CRUD + saga cross-ref |
| atlas-npc-shops | 1 (shops/registry.go) | Dual map + reverse index |
| atlas-parties | 2 (party + character registries) | Auto-inc ID + reverse index |
| atlas-party-quests | 1 (instance/registry.go) | Char index + complex state |
| atlas-portal-actions | 1 (action/registry.go) | Simple CRUD |
| atlas-portals | 1 (blocked/cache.go) | Nested map |
| atlas-rates | 3 (registry + item_tracker + initializer) | CRUD + TTL + bool tracker |
| atlas-reactors | 1 (reactor/registry.go) | Global ID + cooldowns + map index |
| atlas-skills | 1 (cooldown_registry.go) | TTL |
| atlas-storage | 2 (storage/cache + projection/manager) | TTL + sync.Map |
| atlas-transports | 3 (instance + character + channel registries) | Complex state + CRUD |
| atlas-world | 2 (channel + rate registries) | Channel tracking + rates |

### DEFER (6 registries, 4 services)
- atlas-monsters: monster/registry.go, monster/cooldown.go
- atlas-character: character/temporal_data.go
- atlas-pets: pet/temporal_data.go
- atlas-maps: map/character/registry.go, map/monster/registry.go

### EXEMPT (5 registries, 2 services)
- atlas-login: account/registry.go, session/registry.go
- atlas-channel: account/registry.go, session/registry.go, server/registry.go

### SEPARATE (1 registry)
- atlas-saga-orchestrator: saga/cache.go → PostgreSQL

### SKIP (26 registries)
- atlas-data: 17 document registries (read-only)
- 6 configuration registries (read-only after startup)
- atlas-npc-shops/cache.go (read-only data cache)
- atlas-messages/command/registry.go (static function registry)
- atlas-reactors/tasks/cooldown_cleanup.go (not a registry)

---

## Dependencies Between Tasks

```
Phase 1 (Foundation)
  └──> Phase 2 (Pilot: chairs, chalkboards, expressions, consumables, portals, portal-actions)
         └──> Phase 3 (Core: buffs, skills, stats, rates, conversations, shops, storage, account)
         └──> Phase 4 (Social: parties, invites, messengers, guilds)
         └──> Phase 5 (World: drops, reactors, party-quests, transports, world)
                └──> Phase 6 (Inventory: reservation + lock registries)
```

Phase 2 must complete before 3-5 to validate the migration pattern.
Phase 6 should be last (highest correctness risk — distributed locking).
Phases 3, 4, 5 can be parallelized.

---

## Uncommitted Changes

All changes from Phase 2, Phase 3 (up to 3.6), and prior sessions are uncommitted on branch `updates`. Total: 119 files changed, ~3585 insertions, ~3252 deletions across 16+ services. Consider committing before continuing.
