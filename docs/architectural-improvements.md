# Atlas Architectural Improvements

## Overview

This document captures architectural issues identified during a principal-engineer-level review of the Atlas microservice ecosystem, focused on horizontal scaling, resilience, and operational concerns.

---

## Critical: In-Memory Singleton State Prevents Horizontal Scaling

### Problem

44 runtime-mutable in-memory registries across 28+ services use the same singleton pattern:

```go
var reg *Registry
var once sync.Once

func GetRegistry() *Registry {
    once.Do(func() { reg = &Registry{...} })
    return reg
}
```

State lives in Go `map` types protected by `sync.RWMutex`. No service uses any form of external shared state (Redis, database-backed cache, distributed locks). The in-memory map IS the source of truth for most services — there is no database backing for runtime state.

### Impact

Running multiple instances of any affected service causes:

1. **Split-brain state** — Kafka consumer group partitioning splits events between instances. Each instance holds a fraction of the state with no visibility into the other.
2. **ID collisions** — Services with auto-incrementing in-memory counters (reactors, drops, messengers, invites, parties) generate colliding IDs across instances.
3. **Lost operations** — A Kafka command for an entity on instance A arrives at instance B, which returns "not found".
4. **No crash recovery** — Service restart loses all in-flight state with no way to rebuild it.

### Affected Services (CRITICAL — runtime-mutated, in-memory-only)

| Service | State | Key Risk |
|---------|-------|----------|
| atlas-saga-orchestrator | In-flight saga transactions | Lost distributed transactions on restart |
| atlas-account | Account session state machine | Duplicate logins; login-to-channel transitions fail |
| atlas-monsters | Live monster HP/position/AI/cooldowns | Damage invisible cross-instance; ID collisions |
| atlas-reactors | Reactor state + cooldowns + running ID | ID collisions; HIT on wrong instance = 404 |
| atlas-drops | Ground items + reservations + atomic ID | Duplicate pickups; ID collisions |
| atlas-parties | Party membership + character metadata | Party queries return partial members |
| atlas-party-quests | PQ instances + stage progress + timers | Timer ticks only see local instances |
| atlas-npc-conversations | Conversation state machines + saga refs | Conversation continues on wrong instance = orphaned |
| atlas-inventory | Slot reservations + per-character locks | Two instances reserve same slot = item duplication |
| atlas-buffs | Active buffs + poison ticks | Buffs invisible cross-instance; poison ticks duplicated |
| atlas-maps | Characters-in-map + spawn cooldowns | Partial map population; boss respawns diverge |
| atlas-skills | Per-character skill cooldowns | Cooldown enforcement fails = exploit |
| atlas-effective-stats | Computed character stats | Stale/wrong stat queries |
| atlas-character | Session state + real-time position | Position queries wrong; duplicate login detection fails |
| atlas-pets | Pet position + active pet tracking | Pet visible on one instance only |
| atlas-rates | Per-character rate multipliers + item trackers | EXP/meso/drop rates calculated incorrectly |
| atlas-invites | Pending invites (party/guild/trade) | Accept/decline hits wrong instance = not found |
| atlas-guilds | Guild creation agreements | Multi-step agreement flow breaks |
| atlas-messengers | Chat rooms + members | Chat visible on one instance only |
| atlas-transports | Transport instances + boarding state | Characters stuck/duplicated on boats |
| atlas-npc-shops | Character-to-shop mapping | Shop context lost = purchase operations fail |
| atlas-chairs | Sit state per character | Visual state diverges |
| atlas-chalkboards | Chalkboard text per character | Text visible on one instance only |
| atlas-storage | NPC context + storage projections | Storage operations fail without NPC context |
| atlas-portal-actions | Pending saga-based portal actions | Saga completion can't find the portal action |
| atlas-cashshop | Item reservations (5min TTL) | Double-purchase of cash items |
| atlas-character-factory | Follow-up saga templates + completion tracker | Character creation follow-up sagas never fire |
| atlas-consumables | Character-to-map tracking | Consumable effects use stale map context |
| atlas-portals | Blocked portals per character | Portal anti-spam not enforced cross-instance |
| atlas-expressions | Active facial expressions (5s TTL) | Expression broadcasts miss state |

### Services Exempt from Migration

| Service | Reason |
|---------|--------|
| atlas-login | Low-throughput gateway; single instance is sufficient |
| atlas-channel | Naturally sharded per-channel; session holds `net.Conn` (physically process-local) |

### Recommended Solution: Shared Redis Library

Replace the `sync.Once` + `map` + `sync.RWMutex` singleton pattern with a shared Redis-backed registry library. See [high-throughput-cache-problem.md](high-throughput-cache-problem.md) for services that need special handling due to update frequency.

Key advantages of Redis over Kafka key-based partitioning:
- **True redundancy** — Kafka partitioning solves scaling but not durability; instance death still loses state.
- **No recovery mechanism needed** — Partitioning requires event sourcing or snapshot rebuilds on rebalance.
- **No REST routing complexity** — Partitioning only routes Kafka messages; REST requests still need partition-aware routing.
- **Simpler architecture** — Redis is a proven shared state store vs. building a custom distributed state system.
- **Latency acceptable** — ~0.1-0.5ms per Redis operation (same-network) vs ~100ns in-memory. Imperceptible for most game actions.

---

## Critical: Saga Orchestrator Durability

### Problem

`atlas-saga-orchestrator` stores all saga state in an `InMemoryCache` (`map[uuid.UUID]map[uuid.UUID]Saga`) with no database persistence, no TTL, and no timeout mechanism.

### Impact

- Service restart loses all in-flight distributed transactions with no recovery path.
- Read-modify-write race condition: `GetById` -> modify -> `Put` without saga-level locking allows concurrent step completions to corrupt state.
- No stale saga detection or reaper — a saga stuck waiting for a response that never comes will leak memory indefinitely.

### Recommendation

Persist saga state to PostgreSQL (not Redis — sagas need durability guarantees). Add:
- Database-backed saga store with row-level locking
- Timeout mechanism with configurable per-step deadlines
- Stale saga reaper that compensates abandoned sagas
- Idempotent step completion to handle duplicate Kafka events

---

## High: No HTTP Client Timeouts

### Problem

All cross-service REST calls use `http.DefaultClient` which has no default timeout. The only timeout mechanism is Go context cancellation, but services generally pass contexts without deadlines.

### Impact

A single slow or unresponsive service can cascade failures across the ecosystem. Goroutines block indefinitely waiting for responses, eventually exhausting connection pools and memory.

### Recommendation

Add configurable client-side timeouts in `libs/atlas-rest`. Default to 5-10 seconds for standard calls. The nginx ingress timeouts (1800s) provide no meaningful protection.

---

## High: At-Most-Once Kafka Delivery

### Problem

Kafka consumers use `ReadMessage()` which auto-commits the offset before the message is processed. If the consumer crashes during processing, the message is lost.

### Impact

Silent data loss on consumer crashes. State mutations that should have occurred are permanently skipped.

### Recommendation

Switch to `FetchMessage()` + explicit `CommitMessages()` after successful processing. This gives at-least-once delivery. Combine with idempotent message handlers where needed.

---

## High: No Authentication

### Problem

No JWT, OAuth, API keys, or bearer tokens on any endpoint. The system relies solely on tenant headers for multi-tenancy, with no verification that headers are legitimate.

### Recommendation

Add authentication at the ingress layer. Internal service-to-service calls can use mTLS or a shared service mesh.

---

## Medium: No Connection Pool Configuration

### Problem

25+ services use copy-pasted `database/connection.go` with no connection pool settings. GORM defaults apply (unlimited open connections, no max idle, no lifetime).

### Recommendation

Add pool configuration to the shared database library: `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`.

---

## Medium: Manual Tenant Filtering

### Problem

Every database query manually adds `.Where("tenant_id = ?", tenantId)`. No GORM global scope ensures tenant isolation.

### Recommendation

Add a GORM global callback that automatically injects the tenant filter from context, eliminating the class of bugs where a query forgets the tenant clause.

---

## Medium: Single Nginx Ingress as SPOF

### Problem

All inter-service REST traffic routes through a single nginx deployment. No health checks, no redundancy, no rate limiting.

### Recommendation

Consider direct service-to-service communication for internal calls, or deploy the ingress with replicas and health-check-based routing.

---

## Low: Duplicated Database/REST Boilerplate

### Problem

`database/connection.go`, `rest/handler.go`, and `rest/request.go` are copy-pasted across 25+ services with minor variations.

### Recommendation

Extract into shared libraries. The `Provider` pattern already abstracts data access, so the refactor surface is bounded.

---

## Low: Kafka Retry Logic

### Problem

Retry logic uses fixed 1-second sleep with no exponential backoff. Default is 1 attempt (no retries). No service overrides this.

### Recommendation

Add exponential backoff with jitter. Consider configuring reasonable retry counts for critical paths.
