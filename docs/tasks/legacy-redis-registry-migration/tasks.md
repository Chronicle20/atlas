# Redis Registry Migration — Tasks

**Last Updated: 2026-02-17**

## Phase 1: Foundation — Shared Library + Infrastructure

- [x] **1.1** Create `libs/atlas-redis/` skeleton with `go.mod` (`github.com/Chronicle20/atlas-redis`), add `go-redis/redis/v9` + `alicebob/miniredis/v2` deps, add to `go.work`
- [x] **1.2** Implement `connection.go` — `Connect(logger) *redis.Client` reading `REDIS_URL` from env, with pool config
- [x] **1.3** Implement `keys.go` — `TenantKey(tenant.Model) string`, composite key builders, namespace prefixing
- [x] **1.4** Implement `tenant_registry.go` — Generic `TenantRegistry[K, V]` with Get, GetAllValues, Put, PutWithTTL, Remove, Update, Exists
- [x] **1.5** Implement `ttl.go` — `TTLRegistry[K, V]` with Put, PutWithTTL, PopExpired, PopExpiredWithKeys, SetNowFunc
- [x] **1.6** Deploy Redis to Kubernetes — Deployment + Service + add `REDIS_URL` to `atlas-env.yaml`
- [x] **1.7** Implement `id.go` — Per-tenant auto-increment via Redis INCR
- [x] **1.8** Implement `index.go` — Redis SET-based secondary indexes
- [ ] **1.9** Implement `lock.go` — Distributed locks via `SET NX EX` (deferred to Phase 6)

## Phase 2: Pilot Migration — Low-Risk Services ✅ COMPLETE

- [x] **2.1** atlas-chairs: Migrate `chair/registry.go` (Simple CRUD: charId→Model) — TenantRegistry
- [x] **2.2** atlas-chairs: Migrate `character/registry.go` (Composite key: MapKey→[]charId) — TenantRegistry
- [x] **2.3** atlas-chalkboards: Migrate `chalkboard/registry.go` (Simple CRUD: ChalkboardKey→string) — TenantRegistry
- [x] **2.4** atlas-chalkboards: Migrate `character/registry.go` (Composite key: MapKey→[]charId) — TenantRegistry
- [x] **2.5** atlas-expressions: Migrate `expression/registry.go` (TTL: 5-second expiration) — TTLRegistry + tenant tracking SET
- [x] **2.6** atlas-consumables: Migrate `character/registry.go` (Simple CRUD: charId→MapKey) — TenantRegistry[uint32, field.Model]
- [x] **2.7** atlas-portals: Migrate `blocked/cache.go` (Nested map: tenant→char→portal→bool) — Custom Redis SET-based registry
- [x] **2.8** atlas-portal-actions: Migrate `action/registry.go` (Simple CRUD: sagaId→PendingAction) — TenantRegistry[uuid.UUID, PendingAction]

## Phase 3: Core Game State Migration

- [x] **3.1** atlas-buffs: Migrate `character/registry.go` (TTL + immunity/poison queries)
- [x] **3.2** atlas-skills: Migrate `cooldown_registry.go` (TTL cooldown timestamps)
- [x] **3.3** atlas-effective-stats: Migrate `character/registry.go` (Tenant CRUD + bonus stacking)
- [x] **3.4** atlas-rates: Migrate `character/registry.go` (Tenant CRUD + rate factors)
- [x] **3.5** atlas-rates: Migrate `character/item_tracker.go` (TTL coupon expiration)
- [x] **3.6** atlas-rates: Migrate `character/initializer.go` (Simple bool tracker)
- [x] **3.7** atlas-npc-conversations: Migrate `conversation/registry.go` (CRUD + sagaId cross-reference)
- [x] **3.8** atlas-npc-shops: Migrate `shops/registry.go` (Dual map + shop↔character reverse index)
- [x] **3.9** atlas-storage: Migrate `storage/cache.go` (TTL NPC context)
- [x] **3.10** atlas-storage: Migrate `projection/manager.go` (sync.Map → Redis)
- [x] **3.11** atlas-character: Migrate `session/registry.go` (Tenant CRUD + age tracking)
- [x] **3.12** atlas-pets: Migrate `character/registry.go` (Simple CRUD: charId→MapKey)
- [x] **3.13** atlas-character-factory: Migrate `factory/cache.go` (Two stores + saga completion tracking)
- [ ] ~~**3.14** atlas-cashshop: Migrate `reservation/cache.go` (TTL 5-minute reservations)~~ — SKIPPED: cache is unused
- [x] **3.15** atlas-account: Migrate `account/registry.go` (State machine + transition expiration)

## Phase 4: Social & Coordination Services

- [x] **4.1** atlas-parties: Migrate `party/registry.go` (Auto-inc ID + character→party index)
- [x] **4.2** atlas-parties: Migrate `character/registry.go` (Tenant CRUD)
- [x] **4.3** atlas-invites: Migrate `invite/registry.go` (Auto-inc ID + TTL + triple-nested keys)
- [x] **4.4** atlas-messengers: Migrate `messenger/registry.go` (Auto-inc ID)
- [x] **4.5** atlas-messengers: Migrate `character/registry.go` (Tenant CRUD)
- [x] **4.6** atlas-guilds: Migrate `coordinator/registry.go` (TTL + agreement flow)

## Phase 5: Game World State

- [x] **5.1** atlas-drops: Migrate `drop/registry.go` (Global atomic ID + per-drop locks + reservations + map index)
- [x] **5.2** atlas-reactors: Migrate `reactor/registry.go` (Global running ID + cooldowns + map index + per-map locks)
- [x] ~~**5.3** atlas-party-quests~~ — SKIPPED: Service does not exist
- [x] **5.4** atlas-transports: Migrate `instance/instance_registry.go` (Transport state + boarding)
- [x] **5.5** atlas-transports: Migrate `instance/character_registry.go` (Character→transport mapping)
- [x] **5.6** atlas-transports: Migrate `channel/registry.go` (Channel tracking)
- [x] **5.6b** atlas-transports: Migrate `instance/route_registry.go` (TenantRegistry)
- [x] **5.6c** atlas-transports: Migrate `transport/route_registry.go` (TenantRegistry)
- [x] ~~**5.7** atlas-world: Migrate `channel/registry.go`~~ — Already Redis-backed (TenantRegistry)
- [x] ~~**5.8** atlas-world: Migrate `rate/registry.go`~~ — Already Redis-backed (TenantRegistry)

## Phase 6: Inventory & Locking

- [x] **6.1** atlas-inventory: Migrate `reservation_registry.go` (TTL reservations + swap + bulk cleanup)
- [x] **6.2** atlas-inventory: Migrate `lock_registry.go` (sync.Map of RWMutex → Redis distributed locks)

---

## Per-Service Migration Checklist

_Copy for each service migration:_

- [ ] Add redis/miniredis/testify to service `go.mod` (do NOT add atlas-redis — go.work handles it)
- [ ] Replace `sync.Once` singleton with Redis-backed registry (injected at startup via `main.go`)
- [ ] Add `atlas.Connect(l)` + `xxx.InitRegistry(rc)` to `main.go`
- [ ] Update all callers to pass `context.Context` instead of `t.Id()` or `tenant.Model`
- [ ] Add MarshalJSON/UnmarshalJSON to models with unexported fields (if needed for serialization)
- [ ] Update tests to use `miniredis` — each test gets fresh Redis via `miniredis.RunT(t)`
- [ ] Remove old singleton code (`sync.Once`, `sync.RWMutex`, in-memory maps)
- [ ] Run `go test ./... -count=1` — all pass
- [ ] Run `go build` — succeeds
- [ ] Verify no behavioral changes to REST/Kafka interfaces
- [ ] Update existing tests in OTHER packages that call into the migrated registry (they need InitRegistry setup)

---

## Out of Scope (Tracked Separately)

- [ ] **HIGH-THROUGHPUT**: atlas-monsters registry + cooldowns (see `docs/high-throughput-cache-problem.md`)
- [ ] **HIGH-THROUGHPUT**: atlas-character temporal_data (position updates)
- [ ] **HIGH-THROUGHPUT**: atlas-pets temporal_data (position updates)
- [ ] **HIGH-THROUGHPUT**: atlas-maps character + spawn point registries
- [ ] **POSTGRESQL**: atlas-saga-orchestrator cache → PostgreSQL-backed saga store
- [ ] **EXEMPT**: atlas-login (session + account registries) — no migration
- [ ] **EXEMPT**: atlas-channel (session + account + server registries) — no migration
