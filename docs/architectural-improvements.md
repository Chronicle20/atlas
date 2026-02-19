# Atlas Architectural Improvements

## Overview

This document captures architectural issues identified during a principal-engineer-level review of the Atlas microservice ecosystem, focused on horizontal scaling, resilience, and operational concerns.

---

## Critical: In-Memory Singleton State Prevents Horizontal Scaling

### Status: LARGELY RESOLVED

The `redis-registry-migration` branch implements a shared Redis-backed registry library (`libs/atlas-redis`) and migrates 40+ registries across 25 services. See [redis-registry-migration tasks](../dev/active/redis-registry-migration/redis-registry-migration-tasks.md) for the full checklist.

**Completed:**
- Shared library (`libs/atlas-redis`) — `TenantRegistry[K, V]`, `TTLRegistry[K, V]`, per-tenant auto-increment IDs, SET-based secondary indexes, distributed locks
- Redis deployed to Kubernetes with `REDIS_URL` in `atlas-env.yaml`
- All standard-throughput services migrated (25 services, 40+ registries)
- Residual in-memory patterns resolved: atlas-messengers process-local lock → Redis distributed lock; atlas-npc-shops consumable cache → Redis-backed read-through cache

**Remaining work:**

| Category | Services | Status |
|----------|----------|--------|
| High-throughput | atlas-monsters, atlas-maps, atlas-character (position), atlas-pets (position) | Deferred — see [high-throughput-cache-problem.md](high-throughput-cache-problem.md) |
| PostgreSQL | atlas-saga-orchestrator | RESOLVED — PostgreSQL with optimistic locking + retry + reaper; 2 replicas |
| Exempt | atlas-login, atlas-channel | No migration needed |

### Original Problem

44 runtime-mutable in-memory registries across 28+ services used the same singleton pattern:

```go
var reg *Registry
var once sync.Once

func GetRegistry() *Registry {
    once.Do(func() { reg = &Registry{...} })
    return reg
}
```

State lived in Go `map` types protected by `sync.RWMutex` with no external shared state. The in-memory map was the source of truth for most services — no database backing for runtime state.

### Impact (prior to migration)

Running multiple instances of any affected service caused:

1. **Split-brain state** — Kafka consumer group partitioning split events between instances. Each instance held a fraction of the state with no visibility into the other.
2. **ID collisions** — Services with auto-incrementing in-memory counters (reactors, drops, messengers, invites, parties) generated colliding IDs across instances.
3. **Lost operations** — A Kafka command for an entity on instance A arrived at instance B, which returned "not found".
4. **No crash recovery** — Service restart lost all in-flight state with no way to rebuild it.

### Migrated Services (Redis-backed — horizontally scalable)

| Service | Registries Migrated | Notable |
|---------|---------------------|---------|
| atlas-account | Session state machine | State transitions + TTL expiration |
| atlas-reactors | Reactor state + cooldowns + running ID | Global atomic ID via Redis INCR; per-map locks |
| atlas-drops | Ground items + reservations + atomic ID | Global atomic ID; per-drop locks; map index |
| atlas-parties | Party membership + character registry | Auto-increment ID + character-to-party index |
| atlas-npc-conversations | Conversation state machines + saga index | Complex: `StateContainer` serialized via `storedConversation` pattern |
| atlas-inventory | Slot reservations + per-character locks | Distributed locks via Redis Lua scripts |
| atlas-buffs | Active buffs + immunity/poison tracking | TTL-based buff expiration |
| atlas-skills | Per-character skill cooldowns | TTL cooldown timestamps |
| atlas-effective-stats | Computed character stats | Tenant CRUD + bonus stacking |
| atlas-character | Session state + age tracking | Session registry (position tracking deferred — high-throughput) |
| atlas-pets | Active pet tracking | Pet registry (position tracking deferred — high-throughput) |
| atlas-rates | Rate multipliers + item trackers + initializer | TTL coupon expiration; bool tracker |
| atlas-invites | Pending invites | Auto-increment ID + TTL + triple-nested keys |
| atlas-guilds | Guild creation agreements | TTL + agreement flow |
| atlas-messengers | Chat rooms + members + character registry + create lock | Auto-increment ID; distributed lock for create/invite serialization |
| atlas-transports | Transport instances + boarding + channel + routes | 5 registries across 3 packages |
| atlas-npc-shops | Character-to-shop mapping + consumable cache | Dual map + reverse index; Redis-backed read-through cache |
| atlas-chairs | Sit state + character-in-map tracking | 2 registries |
| atlas-chalkboards | Chalkboard text + character-in-map tracking | 2 registries |
| atlas-storage | NPC context cache + storage projections | TTL cache + sync.Map replaced with TenantRegistry |
| atlas-portal-actions | Pending saga-based portal actions | TenantRegistry[uuid.UUID, PendingAction] |
| atlas-character-factory | Follow-up saga templates + completion tracker | Two stores + saga tracking |
| atlas-consumables | Character-to-map tracking | TenantRegistry[uint32, field.Model] |
| atlas-portals | Blocked portals per character | Custom Redis SET-based registry |
| atlas-expressions | Active facial expressions | TTLRegistry + tenant tracking SET |

### Services Not Yet Migrated

| Service | Reason | Tracked |
|---------|--------|---------|
| atlas-monsters | High-throughput: hundreds of position/HP updates per second per map | [high-throughput-cache-problem.md](high-throughput-cache-problem.md) |
| atlas-maps | High-throughput: spawn cooldown iteration across thousands of spawn points | [high-throughput-cache-problem.md](high-throughput-cache-problem.md) |
| atlas-character (position) | High-throughput: dozens of position updates per second per character | [high-throughput-cache-problem.md](high-throughput-cache-problem.md) |
| atlas-pets (position) | High-throughput: same pattern as character position | [high-throughput-cache-problem.md](high-throughput-cache-problem.md) |
| atlas-saga-orchestrator | Migrated to PostgreSQL (not Redis — needs durability guarantees); horizontally scalable with 2 replicas | RESOLVED |
| atlas-cashshop | Reservation cache is unused code — skipped | N/A |
| atlas-party-quests | Service does not exist | N/A |
| atlas-login | Exempt: low-throughput gateway, single instance sufficient | N/A |
| atlas-asset-expiration | In-memory session tracker (`map[uint32]Session`) for expiration checks | N/A |
| atlas-channel | Exempt: naturally sharded per-channel, holds live `net.Conn` | N/A |

### Solution: Shared Redis Library (`libs/atlas-redis`)

The `sync.Once` + `map` + `sync.RWMutex` singleton pattern was replaced with a shared Redis-backed registry library providing:

- **`TenantRegistry[K, V]`** — Generic tenant-scoped CRUD with Get, GetAllValues, Put, PutWithTTL, Remove, Update, Exists
- **`TTLRegistry[K, V]`** — Time-based expiration with PopExpired support
- **Per-tenant auto-increment IDs** — Via Redis INCR, replacing in-memory counters that collided across instances
- **SET-based secondary indexes** — For reverse lookups (e.g., character-to-party, saga-to-conversation)
- **Distributed locks** — Via Redis Lua scripts, replacing `sync.Map` of `sync.RWMutex`

Key advantages of Redis over Kafka key-based partitioning:
- **True redundancy** — Kafka partitioning solves scaling but not durability; instance death still loses state.
- **No recovery mechanism needed** — Partitioning requires event sourcing or snapshot rebuilds on rebalance.
- **No REST routing complexity** — Partitioning only routes Kafka messages; REST requests still need partition-aware routing.
- **Simpler architecture** — Redis is a proven shared state store vs. building a custom distributed state system.
- **Latency acceptable** — ~0.1-0.5ms per Redis operation (same-network) vs ~100ns in-memory. Imperceptible for most game actions.

---

## Critical: Saga Orchestrator Durability

### Status: RESOLVED

Saga state is now persisted to PostgreSQL via a `PostgresStore` implementing the existing `Cache` interface. The `InMemoryCache` singleton was replaced with a database-backed store while preserving the interface contract — zero changes to Kafka consumers, handlers, compensator, or REST API.

**Horizontally scalable** — deployment updated to 2 replicas. All saga state lives in PostgreSQL with no in-memory caching. Multiple instances safely process concurrent step completions via optimistic locking with automatic retry.

**Implemented:**
- PostgreSQL-backed saga store (`saga/store.go`) with JSONB `saga_data` column
- Optimistic locking via `version` column — concurrent step completions detect conflicts via `VersionConflictError`
- Automatic retry on version conflict — `StepCompletedWithResult` retries up to 3 times with backoff on conflict, re-reading fresh state from DB
- Startup recovery — loads all active/compensating sagas from DB and re-drives them through `Step()`
- Stale saga reaper — background goroutine compensates sagas that exceed their configurable timeout (`SAGA_DEFAULT_TIMEOUT`)
- Idempotent step completion — duplicate Kafka events for already-advanced sagas are detected and ignored
- Full tenant context (region, version) persisted alongside saga for recovery

**Configuration (env vars):**
- `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME` — standard database connection
- `SAGA_DEFAULT_TIMEOUT` — per-saga timeout (default: 5m)
- `SAGA_REAPER_INTERVAL` — reaper check frequency (default: 30s)
- `SAGA_RECOVERY_ENABLED` — toggle startup recovery (default: true)

### Original Problem

`atlas-saga-orchestrator` stored all saga state in an `InMemoryCache` (`map[uuid.UUID]map[uuid.UUID]Saga`) with no database persistence, no TTL, and no timeout mechanism.

### Original Impact

- Service restart lost all in-flight distributed transactions with no recovery path.
- Read-modify-write race condition: `GetById` -> modify -> `Put` without saga-level locking allowed concurrent step completions to corrupt state.
- No stale saga detection or reaper — a saga stuck waiting for a response that never came would leak memory indefinitely.

---

## High: No HTTP Client Timeouts

### Status: RESOLVED

All outbound HTTP requests now have bounded lifetimes via a configured `*http.Client` in `libs/atlas-rest/requests/`. Zero service code changes required — defaults apply transparently to all 53 services.

**Implemented:**
- Package-level `*http.Client` replacing `http.DefaultClient` with configured `Transport` (100 max idle conns, 10 per host, 90s idle timeout)
- Per-request `context.WithTimeout` (default 10s) inside the retry loop, so each attempt gets a fresh timeout window
- `http.Client.Timeout` of 30s as an absolute safety net
- `SetTimeout(d time.Duration)` Configurator for per-request overrides
- `HTTP_CLIENT_TIMEOUT` environment variable to override the default at startup
- 7 unit tests covering timeout triggers, overrides, context cancellation propagation, and retry behavior

**Files:**
- `libs/atlas-rest/requests/client.go` — configured client + `DefaultTimeout` + env var override
- `libs/atlas-rest/requests/config.go` — `timeout` field + `SetTimeout` Configurator
- `libs/atlas-rest/requests/get.go`, `post.go`, `delete.go` — use `client` with `context.WithTimeout`
- `libs/atlas-rest/requests/client_test.go` — timeout behavior tests

### Original Problem

All cross-service REST calls used `http.DefaultClient` which has no default timeout. The only timeout mechanism was Go context cancellation, but services generally passed contexts without deadlines.

### Original Impact

A single slow or unresponsive service could cascade failures across the ecosystem. Goroutines blocked indefinitely waiting for responses, eventually exhausting connection pools and memory.

---

## High: At-Most-Once Kafka Delivery

### Status: RESOLVED

All 48 Kafka consumers now use `FetchMessage()` + explicit `CommitMessages()` after successful handler execution, providing at-least-once delivery. The change is entirely in `libs/atlas-kafka` — zero service code modifications required.

**Implemented:**
- `KafkaReader` interface extended with `FetchMessage()` and `CommitMessages()` (replacing `ReadMessage()`)
- Consumer loop fetches without auto-commit, runs all handlers synchronously, commits only after all handlers succeed
- Handler errors prevent commit — the message will be redelivered on next fetch
- Panic recovery via `safeHandle()` — a panicking handler is caught, logged, and treated as an error (no commit, consumer continues)
- All 60 workspace modules build cleanly; 5 new tests validate commit semantics

**Files:**
- `libs/atlas-kafka/consumer/manager.go` — interface change + consumer loop rewrite
- `libs/atlas-kafka/consumer/manager_test.go` — updated mocks + `TestCommitAfterHandlerCompletes`, `TestHandlerErrorPreventsCommit`, `TestHandlerPanicPreventsCommit`, `TestMultipleHandlersAllCompleteBeforeCommit`

**Remaining:** Idempotency audit — review ~531 handler registrations across 48 services for duplicate-message safety under at-least-once delivery. See [kafka-at-least-once-delivery tasks](../dev/active/kafka-at-least-once-delivery/kafka-at-least-once-delivery-tasks.md).

### Original Problem

Kafka consumers used `ReadMessage()` which auto-commits the offset before the message is processed. If the consumer crashed during processing, the message was lost.

### Original Impact

Silent data loss on consumer crashes. State mutations that should have occurred were permanently skipped.

---

## High: No Authentication

### Problem

No JWT, OAuth, API keys, or bearer tokens on any endpoint. The system relies solely on tenant headers for multi-tenancy, with no verification that headers are legitimate.

### Recommendation

Add authentication at the ingress layer. Internal service-to-service calls can use mTLS or a shared service mesh.

---

## Medium: No Connection Pool Configuration

### Status: RESOLVED

All 29 database connection files across 27 services now have explicit connection pool configuration with environment variable overrides. No service uses Go defaults (unlimited open, 2 idle, forever lifetime).

**Implemented:**
- `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`, `ConnMaxIdleTime` configured in every service's `Connect()` function
- Environment variable overrides: `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`, `DB_CONN_MAX_IDLE_TIME`
- Sensible defaults: 10 max open, 5 max idle, 5m lifetime, 3m idle time
- atlas-data retains higher limits (30 max open, 10 max idle) to match its 4-replica, high-read-traffic profile

| Setting | Standard Default | atlas-data Default | Env Var Override |
|---------|------------------|--------------------|------------------|
| MaxOpenConns | 10 | 30 | `DB_MAX_OPEN_CONNS` |
| MaxIdleConns | 5 | 10 | `DB_MAX_IDLE_CONNS` |
| ConnMaxLifetime | 5m | 5m | `DB_CONN_MAX_LIFETIME` |
| ConnMaxIdleTime | 3m | 3m | `DB_CONN_MAX_IDLE_TIME` |

### Original Problem

25+ services use copy-pasted `database/connection.go` with no connection pool settings. GORM defaults apply (unlimited open connections, no max idle, no lifetime).

### Original Recommendation

Add pool configuration to the shared database library: `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`.

---

## Medium: Manual Tenant Filtering

### Problem

Every database query manually adds `.Where("tenant_id = ?", tenantId)`. No GORM global scope ensures tenant isolation.

### Recommendation

Add a GORM global callback that automatically injects the tenant filter from context, eliminating the class of bugs where a query forgets the tenant clause.

---

## Medium: Single Nginx Ingress as SPOF

### Status: IN PROGRESS (Phase 1 complete)

Phase 1 (resilience) is complete — nginx is no longer a single point of failure. Phase 2 (direct service-to-service communication) is planned but not yet started. See [nginx-ingress-spof plan](../dev/active/nginx-ingress-spof/nginx-ingress-spof-plan.md) for the full roadmap.

**Phase 1 — Resilient Nginx (COMPLETE):**
- 2 replicas with preferred pod anti-affinity across nodes
- Liveness and readiness probes (HTTP GET on port 80)
- PodDisruptionBudget (`minAvailable: 1`) protects against voluntary disruptions
- Proxy timeouts reduced from 1800s to 30s (connect: 10s); WebSocket HMR path retains 3600s

**Phase 2 — Direct Service-to-Service Communication (PLANNED):**
- The `_SERVICE_URL` override mechanism already exists in `libs/atlas-rest/requests/url.go` but is unused
- Adding per-domain env vars (e.g., `CHARACTERS_SERVICE_URL`) to `atlas-env.yaml` will bypass nginx for internal calls
- Incremental rollout: one service pair at a time, starting with low-traffic paths
- See [nginx-ingress-spof tasks](../dev/active/nginx-ingress-spof/nginx-ingress-spof-tasks.md) for the full checklist

**Phase 3 — Debug Tooling Update (PLANNED):**
- `tools/debug-start.sh` currently rewrites the nginx ConfigMap — needs adaptation for direct calls

**Phase 4 — Edge-Only Nginx (PLANNED):**
- Once internal traffic is direct, nginx shrinks to external routes only (UI + external API)
- Potential replacement with Traefik IngressRoute CRDs to eliminate the double-proxy

### Original Problem

All inter-service REST traffic routes through a single nginx deployment. No health checks, no redundancy, no rate limiting.

### Original Recommendation

Consider direct service-to-service communication for internal calls, or deploy the ingress with replicas and health-check-based routing.

---

## Low: Duplicated Database/REST Boilerplate

### Problem

`database/connection.go`, `rest/handler.go`, and `rest/request.go` are copy-pasted across 25+ services with minor variations.

### Recommendation

Extract into shared libraries. The `Provider` pattern already abstracts data access, so the refactor surface is bounded.

---

## Low: Kafka Retry Logic

### Status: RESOLVED

All retry logic across the codebase now uses exponential backoff with full jitter via a shared `libs/atlas-retry` library. The ~29 copy-pasted service-local retry packages have been consolidated and deleted.

**Implemented:**
- Shared retry library (`libs/atlas-retry`) with configurable `Config` struct (builder pattern)
- Exponential backoff: `delay = initialDelay * factor^(attempt-1)` with full jitter: `rand(0, delay)`
- Max delay cap prevents unbounded growth
- Context-aware sleep — `select` on `ctx.Done()` and `time.After`
- Error wrapping with `%w` preserves original errors
- 10 comprehensive tests covering backoff timing, jitter range, context cancellation, and error semantics

**Libraries updated:**
- `libs/atlas-kafka` — consumer fetch retry (10 retries, 100ms→10s) and producer write retry use shared library
- `libs/atlas-rest` — GET/POST/DELETE requests use shared library (200ms→5s)
- Both retain thin wrappers re-exporting `atlas-retry` types for import convenience

**Services consolidated (29 total):**
- All service-local `retry/retry.go` packages deleted
- Database connection retry in each service now uses `atlas-retry` directly (10 retries, 500ms→30s)
- atlas-marriages scheduler (proposal expiry + ceremony timeout) rewritten to use shared `retry.Try`
- atlas-maps linear backoff variant replaced with shared exponential backoff

**Files:**
- `libs/atlas-retry/retry.go` — core library with `Config`, `DefaultConfig()`, `Try()`, `jitteredDelay()`
- `libs/atlas-retry/retry_test.go` — 10 tests
- `libs/atlas-kafka/retry/retry.go` — thin wrapper + `LegacyTry` backward compat
- `libs/atlas-rest/retry/retry.go` — thin wrapper

### Original Problem

Retry logic uses fixed 1-second sleep with no exponential backoff. Default is 1 attempt (no retries). No service overrides this. ~29 services had copy-pasted local retry packages with identical implementations.

### Original Impact

Transient failures (database connection, Kafka broker unavailability) either failed immediately or retried with fixed 1s delays, causing unnecessary downtime during brief outages and thundering-herd effects during recovery.
