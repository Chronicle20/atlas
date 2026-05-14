# atlas-data Cache Invalidation — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-05-08
Revised: 2026-05-08 — v2 pivot to align with task-060 v2 (Redis-backed cache, no `libs/atlas-cache` module).
---

## 1. Overview

`atlas-data` is the system of record for static, WZ-derived reference data (maps, monsters, NPCs, items, skills, quests, reactors, etc.). Downstream services consume this data over `GET /api/data/...` HTTP calls. To absorb the resulting load, several services maintain Redis-backed caches of `atlas-data` responses:

- `atlas-monsters` — Redis-backed read-through cache for `data/monsters/{id}` (introduced by **task-060**, prerequisite for this task; uses two `libs/atlas-redis.TenantRegistry` instances under the namespaces `monsters:cache:data` and `monsters:cache:data:not_found`).
- `atlas-maps` — Redis-backed spawn-point registry keyed `atlas:maps:spawn:{tenant}:{world}:{channel}:{map}:{...}` that is initialized from `data/maps/{id}` on first map activation and **never refreshes** until the keys are manually `DEL`'d. This is a documented operational pain point: after an `atlas-data` redeploy, operators must run `DEL atlas:maps:spawn:*` and clear `atlas-monsters`' Redis cache namespaces, or fixes don't visibly take effect.
- A growing list of follow-on caches that future tasks will add as the Redis-backed cache pattern from task-060 spreads to other `atlas-data` consumers (`atlas-channel`, `atlas-monster-death`, `atlas-pets`, `atlas-reactors`, `atlas-portals`, `atlas-consumables`, `atlas-chairs`, `atlas-transports`, `atlas-character`, `atlas-quest`, `atlas-effective-stats`, etc., all of which currently call `atlas-data` HTTP endpoints uncached).

When `atlas-data` (re)imports a tenant's data — triggered today by per-worker `START_WORKER` commands on `COMMAND_TOPIC_DATA` — those caches go stale. Today's only invalidation mechanism is a manual `redis-cli` `DEL` against each affected namespace. This is brittle, easy to forget, and forces operational toil after every reference-data change.

This task introduces an **event-driven cache-invalidation contract**: `atlas-data` emits a Kafka event on a new shared topic `EVENT_TOPIC_DATA` after each successful per-tenant per-worker import. Cache-owning services subscribe to that topic and clear the Redis namespaces they own that are sourced from the worker's domain. The contract is designed to scale to many consumers, and v1 wires two concrete consumers as the proof case: `atlas-monsters` (the v2 Redis-backed data cache) and `atlas-maps` (existing Redis spawn-point registry).

Because both proof-case caches live in shared Redis (not per-pod memory), **a single replica per service is sufficient to perform the flush** — every replica sees the result immediately. Standard shared consumer groups (single delivery per service) are the right transport here. (The v1 plan called for per-pod fan-out groups; that requirement disappears with task-060's v2 pivot to Redis. See §4.3.)

Out of scope: the actual addition of caches to other services (each is its own follow-up task). This task delivers the contract, the producer, the shared library affordance for tenant-scoped flushes (`TenantRegistry.Clear`), and the two proof-case consumers.

## 2. Goals

Primary goals:

- Define and ship a stable Kafka event contract on a new topic `EVENT_TOPIC_DATA` that `atlas-data` emits after successful per-tenant per-worker imports, with a discriminator field so the topic can carry additional event types in future tasks without re-versioning.
- Wire `atlas-data`'s producer at the worker-success site so each successful worker emits one event with `{tenantId, worker, completedAt}`. Failed workers emit nothing.
- Add a tenant-scoped clear operation to `libs/atlas-redis`: `TenantRegistry.Clear(ctx, t) (deleted int, err error)` that `SCAN`s and pipelined-`DEL`s every key under `atlas:<namespace>:<tenantKey>:*`. This is the minimal library affordance every Redis-backed `atlas-data` cache will reuse.
- Wire the first two proof-case consumers:
  - `atlas-monsters`: subscribes to `EVENT_TOPIC_DATA`, filters for `worker == MONSTER`, calls `Clear` on both the positive and negative `TenantRegistry` instances introduced by task-060 (via a thin service-side `monster/information.FlushTenant` wrapper).
  - `atlas-maps`: subscribes to `EVENT_TOPIC_DATA`, filters for `worker == MAP` only (see §4.6 — the spawn registry depends only on map data, not monster reference data), calls a new `SpawnPointRegistry.FlushTenant` that scans `atlas:maps:spawn:{tenantId}:*` and pipelines `DEL`s.
- Use **standard shared consumer groups** (not per-pod) for both new consumers. Both proof caches are Redis-backed and shared across replicas; a single delivery per service is sufficient and avoids unbounded consumer-group proliferation in Kafka.
- Provide observability so operators can see: how many invalidation events `atlas-data` has emitted, how many each consumer has processed, and how many Redis keys each flush deleted.
- Confirm end-to-end: trigger a tenant data import, verify both proof-case consumers flush within seconds, verify caches re-populate from `atlas-data` on next access.

Non-goals:

- Adding caches to other services. Each such task (e.g. wiring `atlas-channel/data/map`, `atlas-pets/data/position`, `atlas-monster-death/monster/information`) is a separate follow-up. This task only ships the contract, the `Clear` library affordance, and the two proof consumers.
- Selective per-id invalidation. Events identify the worker domain (`MONSTER`, `MAP`, ...) and tenant; consumers flush whole-tenant per-domain cache namespaces. Per-id invalidation is over-engineering for the use case (whole-data-set re-import is the trigger).
- Per-pod consumer-group fanout. Cache state is shared via Redis; one consumer per service suffices. (A future in-process cache that bypasses Redis would need to revisit this; for now, no in-process cache is planned.)
- Atomicity / barrier guarantees that all consumers have flushed before `atlas-data` returns. Eventual consistency is acceptable; the steady-state lag from emit to flush should be < 5 s under normal Kafka conditions.
- A REST/admin endpoint to manually trigger invalidation. Operators trigger by re-importing the tenant's data through the existing `START_WORKER` flow.
- Cross-cluster propagation. Single Kafka cluster deployment is assumed (matches the rest of the project).
- Deduplication / dedup tokens. Idempotent flushes mean a duplicate event is a no-op.
- Replay guarantees. Consumers use `auto.offset.reset=latest`; in-flight events during a deploy may be missed by a starting service and that is acceptable (next import will re-emit; cache TTL provides backstop).
- Removing the existing Redis-`DEL`-based manual invalidation runbook. It remains as a fallback; this task adds an automatic path.
- Changes to `atlas-data`'s import logic, file format, or REST API.

## 3. User Stories

- As an operator who has just deployed a new `atlas-data` build with corrected WZ values, I want every dependent service to refresh its Redis caches automatically within seconds, so that I no longer need to remember to run `redis-cli --scan --pattern 'atlas:monsters:cache:data:*' | xargs DEL` or `DEL atlas:maps:spawn:*`.
- As an SRE diagnosing "fixes don't take effect," I want a metric I can graph showing `data_events_emitted_total` from `atlas-data` and `data_events_processed_total` per consumer service, so I can see at a glance whether the invalidation pipeline is healthy.
- As a developer adding caching to a new `atlas-data` consumer (`atlas-channel`, `atlas-monster-death`, etc.), I want a documented consumer pattern (subscribe to `EVENT_TOPIC_DATA`, filter on worker, call `TenantRegistry.Clear`) so I can opt into invalidation in <50 lines.
- As a multi-tenant operator, I need confidence that an event emitted by tenant A's import never causes tenant B's caches to flush.
- As an `atlas-data` maintainer, I want emission tied to per-worker success (not per-tenant aggregate), so partial import re-runs only flush the domains that actually changed.

## 4. Functional Requirements

### 4.1 New Kafka topic and event envelope

A new topic constant is introduced in `atlas-data` and duplicated in each consumer service (matches the existing per-service convention; see `services/atlas-cashshop/atlas.com/cashshop/kafka/message/wallet/kafka.go:6` and dozens of other examples):

```
EVENT_TOPIC_DATA = "EVENT_TOPIC_DATA"
```

(The env-var name follows the `EnvEventTopic*` style used across the codebase. The actual topic name is resolved at runtime via `topic.EnvProvider`.)

The on-the-wire event envelope follows the existing `command[E]` shape used by `atlas-data` for `COMMAND_TOPIC_DATA`:

```go
type event[E any] struct {
    Type string `json:"type"`
    Body E      `json:"body"`
}
```

The first event type is:

```
Type: "DATA_UPDATED"
```

Body shape:

```go
type dataUpdatedEventBody struct {
    TenantId    string `json:"tenantId"`     // canonical tenant UUID
    Worker      string `json:"worker"`       // one of WorkerMap, WorkerMonster, ... from data.Workers
    CompletedAt string `json:"completedAt"`  // RFC3339 UTC
}
```

The Kafka message **key** is the `tenantId` string (so partition ordering preserves per-tenant emission order, matching the existing pattern of tenant-keyed topics). The producer wrapper (`services/atlas-data/atlas.com/data/kafka/producer/producer.go`) also attaches the standard tenant headers via `TenantHeaderDecorator(ctx)` — so consumers can recover tenant via headers (idiomatic) or via the body's `TenantId` field (canonical). Partition count and replication factor follow the broker's defaults via auto-create-topics, matching `COMMAND_TOPIC_DATA` and every other dynamic topic in the repo.

The `Type` discriminator is mandatory; consumers MUST switch on `Type` and ignore unknown values. This leaves room for future event types on the same topic (e.g. `DATA_IMPORT_FAILED`, `DATA_IMPORT_STARTED`) without breaking existing consumers.

### 4.2 Producer wiring inside `atlas-data`

In `services/atlas-data/atlas.com/data/data/processor.go`, the `StartWorker` function returns success at the bottom of each per-worker branch with `l.Infof("Worker [%s] completed.", name)`. After that log line and only on `err == nil`, emit one `DATA_UPDATED` event per (tenant, worker) success.

The producer is wired in `services/atlas-data/atlas.com/data/data/producer.go` (existing file), exposing a function with the project's standard producer signature:

```go
func dataUpdatedEventProvider(tenantId string, worker string, completedAt time.Time) model.Provider[[]kafka.Message]
```

`producer.ProviderImpl(l)(ctx)(EnvEventTopic)(dataUpdatedEventProvider(...))` is the call site at the bottom of `StartWorker`. Failures of the producer are logged at WARN and do NOT fail the worker (we don't want a Kafka outage to roll back a successful data import; metric-tracked emission count makes it visible).

If a worker fails (`err != nil`), no event is emitted. The existing error-return path is preserved.

### 4.3 Consumer-group convention (shared, not per-pod)

The repo's existing pattern uses a single, fixed `consumerGroupId` string per service (e.g. `"Monster Registry Service"`, `"Map Service"`). This task adds a **second** fixed group per consumer service for the new `EVENT_TOPIC_DATA` subscription:

- atlas-monsters: `"Monster Data Cache Invalidator"`
- atlas-maps: `"Map Spawn Registry Invalidator"`

A single consumer per service receives each `DATA_UPDATED` event. Because both proof-case caches are stored in **shared Redis**, that one consumer's `Clear` call is immediately visible to every replica of every service. There is no need to fan out the event to every pod.

The consumer config sets `auto.offset.reset=latest` (`consumer.SetStartOffset(kafka.LastOffset)`, an existing decorator) so a starting consumer begins from the tail of the topic, not from the beginning. Without this, a fresh deploy would replay every historical flush event — wasted work and noisy logs.

**Forward compatibility:** if a future task adds an in-process (per-pod) cache that bypasses Redis, that consumer will need per-pod fan-out. We document the consumer-group rationale in the new consumer code so the next maintainer sees why it's shared. We do **not** ship a `consumer.PerPodGroup` helper now (YAGNI — no in-process cache is planned).

### 4.4 New library affordance: `TenantRegistry.Clear`

`libs/atlas-redis.TenantRegistry` (already used by task-060 v2) exposes `Get`, `GetAllValues`, `Put`, `PutWithTTL`, `Remove`, `Update`, `Exists`, `Client`, and `Namespace`, but no whole-tenant flush. This task adds:

```go
// Clear deletes every entry for tenant t in this registry's namespace.
// Uses SCAN with COUNT to avoid blocking the broker; pipelines DEL per
// batch. Returns the count of keys deleted. A partial failure (e.g. a
// network blip mid-scan) is logged and surfaced as the returned error,
// but the partial deletion is not rolled back — Redis's TTL backstop
// covers eventual convergence.
func (r *TenantRegistry[K, V]) Clear(ctx context.Context, t tenant.Model) (deleted int, err error)
```

`Clear` is the single library addition this task makes. All other tenant-scoped flushes in this project — atlas-monsters' two namespaces, future atlas-channel cache namespaces, etc. — reuse it directly. Tests in `libs/atlas-redis/tenant_registry_test.go` verify Clear under miniredis for: empty namespace (returns `(0, nil)`), populated namespace (returns correct count), tenant isolation (other tenant's keys remain), partial-failure resilience (a forced DEL error mid-scan returns `(partial_count, err)` and does not abort).

The kill-switch `MONSTER_DATA_CACHE_ENABLED=false` (from task-060) is honored by the service-side wrapper, not by `Clear` itself. If the cache is disabled, the wrapper's `FlushTenant` returns early and `Clear` is never invoked.

### 4.5 Consumer wiring: `atlas-monsters`

Add a new Kafka consumer at `services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/`:

- `consumer.go`: registers the consumer on `EVENT_TOPIC_DATA` using the new fixed group `"Monster Data Cache Invalidator"`.
- `kafka.go`: defines the event envelope shape and `dataUpdatedEventBody`.
- `handler.go`: switches on `event.Type`. For `DATA_UPDATED` with `Worker == data.WorkerMonster`, parses `TenantId`, resolves the tenant (from Kafka headers via the existing `TenantHeaderParser`), and calls `monster/information.FlushTenant(ctx, tenantModel)`. For any other `Type` or `Worker`, the handler ignores the message at debug level.
- `monster/information/cache.go` (extended from task-060): exposes `FlushTenant(ctx context.Context, t tenant.Model) (deleted int, err error)` that calls `cache.posReg.Clear(ctx, t)` and `cache.negReg.Clear(ctx, t)` and returns the sum of deleted-key counts.

Wire the new consumer in `services/atlas-monsters/atlas.com/monsters/main.go` alongside the existing `monster2.InitConsumers` and `_map.InitConsumers` calls, but using the new fixed group rather than the existing `"Monster Registry Service"` group (so the data-events consumer's offset state is independent of the command-topic consumer's).

### 4.6 Consumer wiring: `atlas-maps`

Add a new Kafka consumer at `services/atlas-maps/atlas.com/maps/kafka/consumer/data/` mirroring the `atlas-monsters` consumer file layout.

The handler switches on `Type == "DATA_UPDATED"` and on the `Worker` field:

- `Worker == data.WorkerMap` → flush the spawn-point registry for the tenant. The existing Redis key pattern is `atlas:maps:spawn:{tenantId}:{world}:{channel}:{map}:{...}` (see `services/atlas-maps/atlas.com/maps/map/monster/registry.go:60`), so flushing a tenant means `SCAN`/`DEL` on `atlas:maps:spawn:{tenantId}:*`. This task adds `SpawnPointRegistry.FlushTenant(ctx, l, tenantId)` modeled on the existing global `Reset(ctx)` method but scoped to a tenant prefix.
- All other `Type` / `Worker` combinations (including `Worker == MONSTER`) are ignored at debug level. **Rationale (verified at design time):** the spawn registry stores only spawn-point geometry and timing (`Id`, `Template`, `MobTime`, `Cy`, `F`, `Fh`, `Rx0`, `Rx1`, `X`, `Y`, `NextSpawnAt`). The `Template` is a monster-id reference but the registry stores nothing **about** the monster — those attributes (HP, MP, drops, etc.) are looked up via the regular HTTP path at spawn time. `MobTime` (respawn cooldown) is part of the **map's** spawn-point definition, populated by `WorkerMap`. So the spawn registry depends only on map data, not monster reference data.

Note: `SpawnPointRegistry` is not currently a `TenantRegistry` (it uses a hand-rolled key shape `atlas:maps:spawn:{tenant}:{world}:{channel}:{map}:{instance}` rather than the `TenantRegistry`'s `atlas:<namespace>:<tenantKey>:<id>` shape). Migrating it to `TenantRegistry` is out of scope; we add a tenant-scoped flush method to it directly, parallel to the new `TenantRegistry.Clear`.

Wire the new consumer in `services/atlas-maps/atlas.com/maps/main.go` using the new fixed group `"Map Spawn Registry Invalidator"`, alongside the existing fixed-group consumers.

### 4.7 Configuration

| Env var | Purpose | Default | Min | Max |
|---|---|---|---|---|
| `EVENT_TOPIC_DATA` | Kafka topic name for data lifecycle events | (no default — must be set in deploy manifests) | n/a | n/a |
| `DATA_EVENTS_PRODUCER_ENABLED` (in `atlas-data`) | Master kill-switch for the producer | `true` | n/a | n/a |
| `DATA_EVENTS_CONSUMER_ENABLED` (in each consumer service) | Master kill-switch per consumer | `true` | n/a | n/a |

When `DATA_EVENTS_PRODUCER_ENABLED=false`, `atlas-data` does NOT emit events on worker success. Workers behave exactly as today. This is the rollback for the producer.

When `DATA_EVENTS_CONSUMER_ENABLED=false` in a consumer service, the consumer's handler is not registered at startup. Cache state is unchanged from today (TTL expiration or manual flush remain the only invalidation paths). This is the per-consumer rollback.

The shared `deploy/k8s/env-configmap.yaml` ConfigMap (referenced by every service's deployment via `envFrom: configMapRef.name: atlas-env`) gains a single new line:

```yaml
EVENT_TOPIC_DATA: "EVENT_TOPIC_DATA"
```

Adding the env var to that ConfigMap makes it available simultaneously to `atlas-data`, `atlas-monsters`, `atlas-maps`, and any future consumer with no per-service deploy edit. The compose stack gets the same line in `deploy/compose/.env.example`.

### 4.8 Multi-tenancy

- Events are tenant-scoped via the `TenantId` field. Consumers MUST flush only the per-tenant scope identified by the event.
- Kafka message key is `TenantId` so partition ordering preserves emission order per tenant. (Cross-tenant ordering is irrelevant to invalidation.)
- A consumer that fails to parse `TenantId` MUST log an ERROR with the raw payload and skip the message. It MUST NOT flush all tenants as a panic-fallback — that would amplify a poison message into a service-wide cache miss.
- A consumer that observes a disagreement between the Kafka tenant header and `event.Body.TenantId` MUST log a WARN, prefer the body, and continue. (This is a defensive cross-check; in practice the producer attaches both from the same `ctx`, so they always agree.)

### 4.9 Error handling

- Producer emit failure: WARN log + `data_events_emit_failures_total` counter increment; worker still reports success to the existing flow.
- Consumer parse failure (malformed envelope or unparseable `TenantId`): ERROR log + `data_events_consumer_errors_total{kind="parse"}` counter; message offset is committed (don't loop forever on a poison message).
- Consumer flush failure (e.g. Redis network error during `SCAN`/`DEL`): ERROR log + `data_events_consumer_errors_total{kind="flush"}` counter; message offset is committed (cache state will recover via TTL or next event; a stuck consumer is worse than a stale cache). Partial deletions are **not** rolled back — Redis converges on the next event.
- Unknown `Type` or `Worker` value: DEBUG log + `data_events_consumer_skipped_total{reason}` counter where `reason ∈ {unknown_type, unrelated_worker}`. Offset committed.

### 4.10 Observability

In `atlas-data`:

- `atlas_data_events_emitted_total{worker, type}` — counter, incremented on every successful Kafka emit.
- `atlas_data_events_emit_failures_total{worker, type}` — counter, incremented when the producer returns an error.

In `atlas-monsters` and `atlas-maps` (and any future consumer):

- `atlas_<service>_data_events_processed_total{worker, type, action}` — counter where `action ∈ {flushed, skipped}`.
- `atlas_<service>_data_events_consumer_errors_total{kind}` — counter where `kind ∈ {parse, flush}`.
- `atlas_<service>_data_events_consumer_skipped_total{reason}` — counter where `reason ∈ {unknown_type, unrelated_worker}`.
- `atlas_<service>_data_events_keys_deleted_total{tenant}` — counter, incremented by the count of Redis keys actually deleted by each successful flush. (Replaces the v1 plan's `evictions_total{reason="invalidation"}` label, which has no home in v2 because task-060 dropped its eviction counter.)

Note: task-060 v2 dropped the cache-size gauge (`atlas_monsters_data_cache_size`) because counting Redis keys requires a `SCAN` per sample. Operator visibility into Redis cache size is a Redis-side concern (`redis-cli DBSIZE`, `INFO keyspace`).

### 4.11 Testing

`atlas-data` producer:

- Unit test: on worker success, the producer is called once with the correct `tenantId`, `worker`, and a `completedAt` within the last second.
- Unit test: on worker failure, the producer is NOT called.
- Unit test: with `DATA_EVENTS_PRODUCER_ENABLED=false`, the producer is NOT called regardless of worker outcome.
- Unit test: producer error is logged but does not fail the worker (StartWorker returns nil).

`libs/atlas-redis.TenantRegistry.Clear`:

- Unit test (miniredis): `Clear` on an empty namespace returns `(0, nil)`.
- Unit test: `Clear` on a populated namespace returns the correct count and leaves the namespace empty.
- Unit test: `Clear` for tenant A leaves tenant B's keys in the same namespace untouched.
- Unit test: `Clear` does not delete keys belonging to a different namespace, even if they share the tenant prefix.
- Unit test: a forced DEL failure mid-scan (test hook) returns `(partial_count, err)` and does not panic.

`atlas-monsters` consumer:

- Unit test: a `DATA_UPDATED` event with `Worker=MONSTER` triggers `FlushTenant` for the right tenant.
- Unit test: a `DATA_UPDATED` event with `Worker=MAP` is ignored (no flush).
- Unit test: a malformed payload increments the parse-error counter and does not crash the consumer.
- Unit test: `DATA_EVENTS_CONSUMER_ENABLED=false` skips handler registration.
- Unit test (miniredis end-to-end): pre-populate positive and negative cache entries for two tenants; emit `DATA_UPDATED` for tenant A; verify all of tenant A's keys (in both namespaces) are deleted and all of tenant B's keys remain.

`atlas-maps` consumer:

- Unit test: a `DATA_UPDATED` event with `Worker=MAP` for tenant T deletes only `atlas:maps:spawn:T:*` keys. Other tenants' keys remain.
- Unit test: a `DATA_UPDATED` event with `Worker=MONSTER` is ignored (no flush).
- Unit test: a `DATA_UPDATED` event with `Worker=NPC` (or any non-map worker) is ignored.
- Unit test: a Redis network error during flush increments the flush-error counter, commits the offset, and does not crash the consumer.

End-to-end (integration / manual):

- Trigger a tenant data import via the existing `START_WORKER` mechanism for `WorkerMonster` against tenant T.
- Observe within 5 s of worker completion: `atlas_data_events_emitted_total{worker="MONSTER"}` increments by 1; in `atlas-monsters`, `data_events_processed_total{worker="MONSTER", action="flushed"}` increments by 1.
- Observe in Redis: `redis-cli --scan --pattern 'atlas:monsters:cache:data:<tenantKey>:*'` returns empty after the event, then rebuilds on next request.
- Observe in `atlas-maps`: trigger `WorkerMap`; `KEYS atlas:maps:spawn:{T}:*` returns empty after the event, then rebuilds as maps are re-activated.

## 5. API Surface

No HTTP / JSON:API surface changes. No new REST endpoints. No new request/response shapes for any consumer of `atlas-data`'s HTTP API.

New Kafka surface:

- New topic `EVENT_TOPIC_DATA` (env-var configurable name).
- New event type discriminator `DATA_UPDATED` with body shape defined in §4.1.

New internal Go API:

- `libs/atlas-redis.TenantRegistry.Clear(ctx, t) (deleted int, err error)` — tenant-scoped namespace flush.
- `services/atlas-monsters/.../monster/information.FlushTenant(ctx, t) (deleted int, err error)` — wraps the two `TenantRegistry.Clear` calls (positive + negative namespaces) plus the kill-switch check.
- `services/atlas-maps/.../map/monster.SpawnPointRegistry.FlushTenant(ctx, l, tenantId) (deleted int, err error)` — tenant-scoped variant of the existing global `Reset(ctx)` method.

All other internal Go APIs are unchanged.

## 6. Data Model

No persistent data changes. No database migrations.

In-memory / Redis state changes:

- `libs/atlas-redis.TenantRegistry` gains a `Clear(ctx, t)` method that performs a `SCAN` with a tenant-namespaced match pattern and pipelines `DEL`s. No new data shapes.
- `atlas-maps` Redis spawn-point registry gains a per-tenant flush operation that internally `SCAN`s for `atlas:maps:spawn:{tenantId}:*` and pipelines `DEL`s. The existing global-scan iterator at `registry.go:260` is the model.

## 7. Service Impact

### 7.1 `services/atlas-data` (producer)

- New Kafka producer wiring at the bottom of `StartWorker` per worker success.
- New event types in `data/kafka.go` (envelope + body) plus `EnvEventTopic` constant.
- New `producer.go` provider function `dataUpdatedEventProvider`.
- New env var `DATA_EVENTS_PRODUCER_ENABLED`.
- New metrics §4.10.

### 7.2 `libs/atlas-redis` (depends on task-060 v2)

- New public method `TenantRegistry.Clear(ctx, t) (deleted int, err error)`.
- README / package doc updated noting the new method and its intended use for cache-namespace invalidation.
- Tests for new code paths under miniredis.

### 7.3 `services/atlas-monsters` (proof consumer #1)

- New consumer subtree `kafka/consumer/data/`.
- New `monster/information.FlushTenant(ctx, t) (deleted int, err error)` exposing the per-tenant flush across both v2 namespaces.
- New env var `DATA_EVENTS_CONSUMER_ENABLED`.
- `main.go` registers the new consumer using the new fixed group `"Monster Data Cache Invalidator"`.
- New metrics §4.10.

### 7.4 `services/atlas-maps` (proof consumer #2)

- New consumer subtree `kafka/consumer/data/`.
- New `map/monster.SpawnPointRegistry.FlushTenant(ctx, l, tenantId) (deleted int, err error)` that scans `atlas:maps:spawn:{tenantId}:*` and pipelines `DEL`s.
- New env var `DATA_EVENTS_CONSUMER_ENABLED`.
- `main.go` registers the new consumer using the new fixed group `"Map Spawn Registry Invalidator"`.
- New metrics §4.10.

### 7.5 Other services

- No code changes in this task.
- Future follow-ups (out of scope here):
  - `atlas-channel` (consumes `data/maps`, `data/portals`, `data/skills`, `data/npcs`, `data/quests`, monster info).
  - `atlas-monster-death` (consumes monster info, drop position).
  - `atlas-pets` (consumes data/position).
  - `atlas-reactors` (consumes reactor data).
  - `atlas-portals`, `atlas-consumables`, `atlas-chairs`, `atlas-transports`, `atlas-character`, `atlas-quest`, `atlas-effective-stats`, `atlas-merchant`, `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-configurations`, `atlas-messages` — all currently call `atlas-data` HTTP endpoints uncached and will need invalidation wiring once they add caching. Each will reuse `TenantRegistry.Clear` directly.

### 7.6 Deploy / infra

- `EVENT_TOPIC_DATA` env var added to the shared `deploy/k8s/env-configmap.yaml` ConfigMap (one line) and `deploy/compose/.env.example` (one line). All services pick it up via existing `envFrom`.
- Topic creation: relies on broker auto-create-topics, matching `COMMAND_TOPIC_DATA` and every other dynamic topic in the repo.

## 8. Non-Functional Requirements

### 8.1 Performance

- Producer adds at most one Kafka send per successful worker run. Kafka send is asynchronous; it MUST NOT block worker completion by more than a single network round-trip even on the slow path.
- Consumer flush is O(K) in Redis key count under the tenant prefix (typical: a few thousand keys per tenant per namespace; bounded by tenant-local entity counts).
- End-to-end emit-to-flush latency target: < 5 s under normal Kafka load. No SLO; budget is dominated by Kafka rebalance/poll cycles and Redis SCAN throughput.

### 8.2 Correctness

- No event leak across tenants under any code path.
- Idempotency: a duplicate event causes a redundant flush (no-op on already-empty namespace), never incorrect state.
- A failed consumer flush MUST NOT leave the cache in a half-cleared state for correctness purposes. The implementation accepts that pipelined `DEL`s can partially succeed but converges on the next event / TTL — a stale subset is no worse than the full pre-event state.
- Race-detector clean: `go test -race ./...` in all changed modules.

### 8.3 Observability

- Metrics §4.10 are present and labeled.
- Operator-facing dashboard panels can answer:
  - "Are events being emitted?" — `rate(atlas_data_events_emitted_total[5m])`.
  - "Are consumers keeping up?" — emit rate vs. processed rate per consumer.
  - "Are flushes failing?" — `atlas_<service>_data_events_consumer_errors_total`.
  - "How many keys does each flush actually delete?" — `rate(atlas_<service>_data_events_keys_deleted_total[5m])`.
- Dashboard authoring is OUT of scope; metric availability is in scope.

### 8.4 Security & multi-tenancy

- Tenant scoping enforced at the event payload level. Consumers MUST flush only the tenant in `event.Body.TenantId`.
- No new external network egress or new secrets.
- No PII or auth data on the topic (just tenant UUID, worker name, timestamp).

### 8.5 Operability

- `DATA_EVENTS_PRODUCER_ENABLED=false` rolls the producer back without code change.
- `DATA_EVENTS_CONSUMER_ENABLED=false` rolls a single consumer back without code change.
- The existing manual invalidation runbook (`redis-cli --scan --pattern 'atlas:monsters:cache:data:*' | xargs DEL` and `DEL atlas:maps:spawn:*`) remains documented as a fallback in the user's `reference_atlas_maps_spawn_cache.md` memory note. This task does not retire it.
- No CLI / admin endpoint to manually trigger an event. Operators trigger by re-running the import for the relevant tenant/worker.

## 9. Open Questions

- **Topic name finalization** — `EVENT_TOPIC_DATA` (proposed, mirrors `COMMAND_TOPIC_DATA`) vs. a more specific `EVENT_TOPIC_DATA_LIFECYCLE` or `EVENT_TOPIC_DATA_UPDATES`. User preference is `EVENT_TOPIC_DATA` to leave room for additional event types beyond invalidation. Confirmed.
- **Should `TenantRegistry.Clear` use Lua-scripted SCAN+DEL atomicity vs. pipelined DEL?** Lua is atomic per-shard but blocks the broker; pipelined DEL is non-atomic but cooperative. Default: pipelined DEL with SCAN COUNT=100. Revisit if a tenant's keyspace ever exceeds ~10 k keys per namespace.
- **`atlas-maps` `SpawnPointRegistry` migration to `TenantRegistry`** — the existing hand-rolled key shape is compatible with `tenantEntityKey` semantically but different formatting. Migrating is out of scope; the hand-rolled FlushTenant is parallel to the library's Clear. A future task may unify.
- **Future event types on the same topic** — `DATA_IMPORT_STARTED`, `DATA_IMPORT_FAILED` are plausible additions. Out of scope to design here, but the discriminator `Type` field is sized for it.
- **In-process cache fanout** — if a future cache *cannot* live in Redis (e.g. for sub-ms hot-path latency), the consumer pattern will need per-pod groups. Out of scope; document the rationale in this task's consumer code so the next maintainer sees why we chose shared groups.

## 10. Acceptance Criteria

A reviewer can mark this task done when ALL of the following are true:

- [ ] Task-060 v2 (`libs/atlas-redis`-backed monster data cache) is merged on `main` and this branch is rebased on top of it.
- [ ] `atlas-data` emits one `DATA_UPDATED` event per successful (tenant, worker) run on `EVENT_TOPIC_DATA` with the §4.1 envelope and body. No event is emitted on worker failure.
- [ ] `EVENT_TOPIC_DATA` is set in `deploy/k8s/env-configmap.yaml` and `deploy/compose/.env.example`.
- [ ] `libs/atlas-redis.TenantRegistry` exposes `Clear(ctx, t) (deleted int, err error)`, miniredis-tested for happy-path, tenant isolation, namespace isolation, and partial-failure resilience; `go test -race ./...` clean.
- [ ] `atlas-monsters` registers a Kafka consumer for `EVENT_TOPIC_DATA` using the fixed group `"Monster Data Cache Invalidator"` with `auto.offset.reset=latest`. On `DATA_UPDATED` with `Worker=MONSTER`, both the positive and negative `TenantRegistry` namespaces for `TenantId` are cleared via `monster/information.FlushTenant`.
- [ ] `atlas-maps` registers a Kafka consumer for `EVENT_TOPIC_DATA` using the fixed group `"Map Spawn Registry Invalidator"` with `auto.offset.reset=latest`. On `DATA_UPDATED` with `Worker=MAP`, the spawn-point registry for `TenantId` is flushed via `SCAN`/`DEL` of `atlas:maps:spawn:{TenantId}:*`. Other workers (including `MONSTER`) are ignored.
- [ ] All §4.10 metrics are emitted and labeled correctly.
- [ ] All §4.11 unit tests pass; `go test -race ./...` is clean in `atlas-data`, `libs/atlas-redis`, `atlas-monsters`, `atlas-maps`.
- [ ] Manual end-to-end verification (per §4.11): trigger a `WorkerMonster` import for a test tenant, observe within 5 s that `atlas_monsters_data_events_processed_total{worker="MONSTER", action="flushed"}` increments by 1 and Redis `--scan --pattern 'atlas:monsters:cache:data:<tenantKey>:*'` returns empty; observe that subsequent `GET /api/data/monsters/{id}` re-populates the cache. Trigger a `WorkerMap` import; verify atlas-maps' Redis spawn keys for that tenant are deleted and rebuild on next map activation.
- [ ] Setting `DATA_EVENTS_PRODUCER_ENABLED=false` (in `atlas-data`) suppresses event emission without code change. Setting `DATA_EVENTS_CONSUMER_ENABLED=false` (in either consumer) suppresses handler registration without code change. Both verified.
- [ ] User memory note `reference_atlas_maps_spawn_cache.md` is updated (or annotated in the PR description) noting that automatic invalidation now exists for `MAP` worker re-imports and the manual `DEL` runbook is now a fallback rather than the primary path.
- [ ] PRD updated to status `Implemented` and `Date implemented` on merge.
