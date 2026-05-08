# atlas-data Cache Invalidation — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-08
---

## 1. Overview

`atlas-data` is the system of record for static, WZ-derived reference data (maps, monsters, NPCs, items, skills, quests, reactors, etc.). Downstream services consume this data over `GET /api/data/...` HTTP calls. To absorb the resulting load, several services maintain their own caches of `atlas-data` responses:

- `atlas-monsters` — in-process TTL cache over `data/monsters/{id}` (introduced by **task-060**, prerequisite for this task; uses `libs/atlas-cache`).
- `atlas-maps` — Redis-backed spawn-point registry keyed `atlas:maps:spawn:{tenant}:{world}:{channel}:{map}:{...}` that is initialized from `data/maps/{id}` on first map activation and **never refreshes** until the keys are manually `DEL`'d. This is a documented operational pain point: after an `atlas-data` redeploy, operators must run `DEL atlas:maps:spawn:*` and clear `atlas-monsters`' Redis state, or fixes don't visibly take effect.
- A growing list of follow-on caches that future tasks will add as the in-process cache pattern from task-060 spreads to other `atlas-data` consumers (`atlas-channel`, `atlas-monster-death`, `atlas-pets`, `atlas-reactors`, `atlas-portals`, `atlas-consumables`, `atlas-chairs`, `atlas-transports`, `atlas-character`, `atlas-quest`, `atlas-effective-stats`, etc., all of which currently call `atlas-data` HTTP endpoints uncached).

When `atlas-data` (re)imports a tenant's data — triggered today by per-worker `START_WORKER` commands on `COMMAND_TOPIC_DATA` — those caches go stale. Today's only invalidation mechanism is a pod restart for in-process caches and a manual Redis `DEL` for `atlas-maps`. This is brittle, easy to forget, and forces operational toil after every reference-data change.

This task introduces an **event-driven cache-invalidation contract**: `atlas-data` emits a Kafka event on a new shared topic `EVENT_TOPIC_DATA` after each successful per-tenant per-worker import. Cache-owning services subscribe to that topic and flush the caches they own that are sourced from the worker's domain. The contract is designed to scale to many consumers, and v1 wires two concrete consumers as the proof case: `atlas-monsters` (in-process cache from task-060) and `atlas-maps` (existing Redis spawn-point registry).

A second concern this task addresses is **multi-pod fanout**. The repo's existing Kafka consumer pattern uses a single, fixed `consumerGroupId` string per service (e.g. `"Monster Registry Service"`, `"Map Service"`), which gives single-pod delivery — only one replica receives any given message. For cache invalidation that's wrong: every replica owns its own in-process cache and must flush. This task introduces a per-pod consumer-group convention (group id suffixed with `HOSTNAME`) that consumers MUST use when subscribing to `EVENT_TOPIC_DATA`, so the event fans out to every replica.

Out of scope: the actual addition of caches to other services (each is its own follow-up task). This task delivers the contract, the producer, the per-pod-group convention, and the two proof-case consumers.

## 2. Goals

Primary goals:

- Define and ship a stable Kafka event contract on a new topic `EVENT_TOPIC_DATA` that `atlas-data` emits after successful per-tenant per-worker imports, with a discriminator field so the topic can carry additional event types in future tasks without re-versioning.
- Wire `atlas-data`'s producer at the worker-success site so each successful worker emits one event with `{tenantId, worker, completedAt}`. Failed workers emit nothing.
- Establish a per-pod consumer-group convention (`<service>-data-events-<HOSTNAME>` or similar) that fans events out to every replica of every consumer service, with `auto.offset.reset=latest` so a pod restart doesn't replay historical flush events.
- Add a tenant-scoped flush API to `libs/atlas-cache` (`Flush()` on the cache primitive, plus a registry-level helper `FlushTenant(tenantId)` on the per-tenant registry pattern).
- Wire the first two proof-case consumers:
  - `atlas-monsters`: subscribes to `EVENT_TOPIC_DATA`, filters for `worker == MONSTER`, flushes the per-tenant `libs/atlas-cache` registry introduced by task-060.
  - `atlas-maps`: subscribes to `EVENT_TOPIC_DATA`, filters for `worker == MAP` and `worker == MONSTER` (spawn registry is derived from both map data and per-monster spawn metadata), flushes the Redis spawn-point registry for the affected tenant via the existing `Clear` path or by deleting the `atlas:maps:spawn:{tenantId}:*` key range.
- Provide observability so operators can see: how many invalidation events `atlas-data` has emitted, how many each consumer has processed, and how many cache entries each flush evicted.
- Confirm end-to-end: trigger a tenant data import, verify both proof-case consumers flush within seconds, verify caches re-populate from `atlas-data` on next access.

Non-goals:

- Adding caches to other services. Each such task (e.g. wiring `atlas-channel/data/map`, `atlas-pets/data/position`, `atlas-monster-death/monster/information`) is a separate follow-up. This task only ships the contract and the two proof consumers.
- Selective per-id invalidation. Events identify the worker domain (`MONSTER`, `MAP`, ...) and tenant; consumers flush whole-tenant per-domain caches. Per-id invalidation is over-engineering for the use case (whole-data-set re-import is the trigger).
- Atomicity / barrier guarantees that all consumers have flushed before `atlas-data` returns. Eventual consistency is acceptable; the steady-state lag from emit to flush should be < 5 s under normal Kafka conditions.
- A REST/admin endpoint to manually trigger invalidation. Operators trigger by re-importing the tenant's data through the existing `START_WORKER` flow.
- Cross-cluster propagation. Single Kafka cluster deployment is assumed (matches the rest of the project).
- Deduplication / dedup tokens. Idempotent flushes mean a duplicate event is a no-op.
- Replay guarantees. Per-pod groups with `latest` offset reset by design drop history; in-flight events during a deploy may be missed by a starting pod and that is acceptable (next import will re-emit; cache TTL provides backstop).
- Removing the existing Redis-`DEL`-based manual invalidation runbook. It remains as a fallback; this task adds an automatic path.
- Changes to `atlas-data`'s import logic, file format, or REST API.

## 3. User Stories

- As an operator who has just deployed a new `atlas-data` build with corrected WZ values, I want every replica of every dependent service to refresh its in-memory and Redis caches automatically within seconds, so that I no longer need to remember to run `DEL atlas:maps:spawn:*` or restart pods.
- As an SRE diagnosing "fixes don't take effect," I want a metric I can graph showing `data_events_emitted_total` from `atlas-data` and `data_events_processed_total` per consumer service, so I can see at a glance whether the invalidation pipeline is healthy.
- As a developer adding caching to a new `atlas-data` consumer (`atlas-channel`, `atlas-monster-death`, etc.), I want a documented consumer pattern (per-pod group + flush-on-event) and a `libs/atlas-cache` flush API I can call directly, so I can opt into invalidation in <50 lines.
- As a multi-tenant operator, I need confidence that an event emitted by tenant A's import never causes tenant B's caches to flush.
- As an `atlas-data` maintainer, I want emission tied to per-worker success (not per-tenant aggregate), so a partial import re-runs only flush the domains that actually changed.

## 4. Functional Requirements

### 4.1 New Kafka topic and event envelope

A new topic constant is introduced in `atlas-data` and exposed for consumers via `libs/atlas-kafka` topic constants if that's the existing convention, otherwise duplicated as a known string in each consumer:

```
EVENT_TOPIC_DATA = "EVENT_TOPIC_DATA"
```

(The env-var name follows the same `EnvCommandTopic` / `EVENT_TOPIC_*` style as existing topics in `atlas-data/data/kafka.go`. The actual topic name is resolved at runtime via the env var, matching existing producer/consumer code.)

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

The Kafka message **key** is the `tenantId` string (so partition ordering preserves per-tenant emission order, matching the existing pattern of tenant-keyed topics elsewhere in the repo). Partition count and replication factor follow the project's existing topic-creation conventions — confirm at design time which topic-creation mechanism the project uses (currently topics appear to be auto-created by the broker on first produce; if not, this task adds the topic to whatever bootstrap manifest already exists for `COMMAND_TOPIC_DATA`).

The `Type` discriminator is mandatory; consumers MUST switch on `Type` and ignore unknown values. This leaves room for future event types on the same topic (e.g. `DATA_IMPORT_FAILED`, `DATA_IMPORT_STARTED`) without breaking existing consumers.

### 4.2 Producer wiring inside `atlas-data`

In `services/atlas-data/atlas.com/data/data/processor.go`, the `StartWorker` function returns success at the bottom of each per-worker branch with `l.Infof("Worker [%s] completed.", name)`. After that log line and only on `err == nil`, emit one `DATA_UPDATED` event per (tenant, worker) success.

The producer is wired in `services/atlas-data/atlas.com/data/data/producer.go` (existing file), exposing a function with the project's standard producer signature:

```go
func dataUpdatedEventProvider(tenantId string, worker string, completedAt time.Time) model.Provider[[]kafka.Message]
```

`producer.ProviderImpl(l)(ctx)(EnvEventTopic)(dataUpdatedEventProvider(...))` is the call site at the bottom of `StartWorker`. Failures of the producer are logged at WARN and do NOT fail the worker (we don't want a Kafka outage to roll back a successful data import; metric-tracked emission count makes it visible).

If a worker fails (`err != nil`), no event is emitted. The existing error-return path is preserved.

### 4.3 Per-pod consumer-group convention

The repo's existing pattern uses a single, fixed `consumerGroupId` string per service (e.g. `"Monster Registry Service"`). For `EVENT_TOPIC_DATA`, this is wrong because Kafka delivers a partition to exactly one consumer per group, meaning only one replica per service would get any given event.

This task establishes the convention: **consumers of `EVENT_TOPIC_DATA` MUST use a unique-per-pod consumer group**. The naming convention is:

```
"<service-short-name> data-events <HOSTNAME>"
```

`HOSTNAME` is read from the `HOSTNAME` environment variable. In Kubernetes, pod names are populated into `HOSTNAME` automatically and are unique within the cluster, with stable names for `StatefulSet`s and unique-but-changing names for `Deployment`s. Either is acceptable; restarts produce a fresh group, which is intentional.

The consumer config MUST set `auto.offset.reset=latest` (or the project's equivalent) so a starting pod begins consumption from the tail of the topic, not from the beginning. Without this, every pod restart would replay every historical flush event — wasted work and potentially noisy logs.

This convention is documented in the `libs/atlas-cache` README (assuming task-060 has landed it; otherwise added there) AND duplicated as a comment block in the `atlas-monsters` and `atlas-maps` consumer-init files so the reasoning is visible at the use site.

The convention is NOT enforced by `libs/atlas-kafka`; it's a documented requirement on `EVENT_TOPIC_DATA` consumers. If `libs/atlas-kafka` already exposes a convenient helper for "per-pod group," this task uses it; otherwise the consumer init in each service does the suffixing inline (single-line compose: `groupId := fmt.Sprintf("%s data-events %s", serviceName, os.Getenv("HOSTNAME"))`).

### 4.4 Flush API on `libs/atlas-cache` (depends on task-060)

`libs/atlas-cache` is introduced by task-060 as a generic in-process TTL cache. This task extends its public API:

- `Cache[K comparable, V any].Flush()` — clears all entries (positive AND negative). After `Flush()`, `Len() == 0`. Concurrency-safe; behaves as one atomic operation from external observers.
- The per-tenant registry pattern recommended by task-060 (one `Cache` instance per `tenant.Id` in a `sync.Once`-initialized registry) gains a sibling helper that the consumer can call: `FlushTenant(tenantId tenant.Id)`. If the tenant has no live cache entry yet, this is a no-op (no harm, no entry created). The exact placement of the helper (on the registry struct vs. a free function) is a design-time decision; either is acceptable.
- The kill-switch `MONSTER_DATA_CACHE_ENABLED=false` (from task-060) MUST short-circuit `FlushTenant`: if the cache is disabled, flush is a no-op (there's nothing to flush, and we don't want to allocate a registry just to clear it).

Tests for `libs/atlas-cache`:

- `Flush` empties a populated cache; subsequent reads miss.
- `Flush` is safe under concurrent `Get` / `Put` (race-detector clean).
- `Flush` resets size gauges and emits eviction counter increments labeled by `reason="invalidation"`, distinct from the existing `reason="expired_positive"` / `reason="expired_negative"` labels in task-060.

### 4.5 Consumer wiring: `atlas-monsters` (in-process cache)

Add a new Kafka consumer at `services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/`:

- `consumer.go`: registers the consumer on `EVENT_TOPIC_DATA` using the per-pod group convention.
- `kafka.go`: defines the event envelope shape and `dataUpdatedEventBody`.
- `handler.go`: switches on `event.Type`. For `DATA_UPDATED` with `Worker == data.WorkerMonster`, parses `TenantId`, resolves the tenant (via existing `tenant` library), and calls `monster/information.FlushTenant(tenantId)` (new function added by this task that delegates to the per-tenant registry from task-060). For any other `Type` or `Worker`, the handler ignores the message at debug level.
- `monster/information/cache.go` (or wherever task-060 placed the registry): exposes `FlushTenant(tenantId)`.

Wire the new consumer in `services/atlas-monsters/atlas.com/monsters/main.go` alongside the existing `monster2.InitConsumers` and `_map.InitConsumers` calls, but using the per-pod group rather than the existing fixed `"Monster Registry Service"` group.

### 4.6 Consumer wiring: `atlas-maps` (Redis spawn-point registry)

Add a new Kafka consumer at `services/atlas-maps/atlas.com/maps/kafka/consumer/data/` mirroring the `atlas-monsters` consumer file layout.

The handler switches on `Type == "DATA_UPDATED"` and on the `Worker` field:

- `Worker == data.WorkerMap` → flush the spawn-point registry for the tenant. The existing Redis key pattern is `atlas:maps:spawn:{tenantId}:{world}:{channel}:{map}:{...}` (see `services/atlas-maps/atlas.com/maps/map/monster/registry.go:60`), so flushing a tenant means `SCAN`/`DEL` on `atlas:maps:spawn:{tenantId}:*`. The existing registry already has a sweep iterator at `services/atlas-maps/atlas.com/maps/map/monster/registry.go:260` that scans `atlas:maps:spawn:*`; this task adds a per-tenant variant.
- `Worker == data.WorkerMonster` → ALSO flush spawn-point registry for the tenant. Spawn metadata embeds monster ids, levels, and respawn timing taken from monster reference data; a monster reference change should refresh the spawn registry too. (If at design time we determine the monster data isn't actually used by the spawn registry — only the map's spawn-point geometry is — drop this branch. PRD assumes the conservative "flush on either" until verified.)
- All other `Type` / `Worker` combinations are ignored at debug level.

Wire the new consumer in `services/atlas-maps/atlas.com/maps/main.go` using the per-pod group, alongside the existing fixed-group consumers.

### 4.7 Configuration

| Env var | Purpose | Default | Min | Max |
|---|---|---|---|---|
| `EVENT_TOPIC_DATA` | Kafka topic name for data lifecycle events | (no default — must be set in deploy manifests) | n/a | n/a |
| `DATA_EVENTS_PRODUCER_ENABLED` (in `atlas-data`) | Master kill-switch for the producer | `true` | n/a | n/a |
| `DATA_EVENTS_CONSUMER_ENABLED` (in each consumer service) | Master kill-switch per consumer | `true` | n/a | n/a |
| `HOSTNAME` | Unique pod identifier used in consumer group suffix | (always populated in k8s; falls back to a startup-generated UUID if unset, with WARN log) | n/a | n/a |

When `DATA_EVENTS_PRODUCER_ENABLED=false`, `atlas-data` does NOT emit events on worker success. Workers behave exactly as today. This is the rollback for the producer.

When `DATA_EVENTS_CONSUMER_ENABLED=false` in a consumer service, the consumer is not registered at startup. Cache state is unchanged from today (TTL expiration or manual flush remain the only invalidation paths). This is the per-consumer rollback.

`HOSTNAME` fallback: if unset (e.g. in a local Docker compose without `hostname:` set), the consumer init generates a UUID at process start and uses that, logging a WARN with the resolved id. This preserves uniqueness.

The deploy manifests (`docker/`, `helm/`, etc. — confirm at design time which the project uses for k8s deployments; observability says k8s/Grafana is in use) must set `EVENT_TOPIC_DATA` for `atlas-data`, `atlas-monsters`, and `atlas-maps`. Adding the env var to manifests is in scope.

### 4.8 Multi-tenancy

- Events are tenant-scoped via the `TenantId` field. Consumers MUST flush only the per-tenant scope identified by the event.
- Kafka message key is `TenantId` so partition ordering preserves emission order per tenant. (Cross-tenant ordering is irrelevant to invalidation.)
- A consumer that fails to parse `TenantId` MUST log an ERROR with the raw payload and skip the message. It MUST NOT flush all tenants as a panic-fallback — that would amplify a poison message into a service-wide cache miss.

### 4.9 Error handling

- Producer emit failure: WARN log + `data_events_emit_failures_total` counter increment; worker still reports success to the existing flow.
- Consumer parse failure: ERROR log + `data_events_consumer_errors_total{kind="parse"}` counter; message offset is committed (don't loop forever on a poison message).
- Consumer flush failure (e.g. Redis network error in `atlas-maps`): ERROR log + `data_events_consumer_errors_total{kind="flush"}` counter; message offset is committed (cache state will recover via TTL or next event; a stuck consumer is worse than a stale cache).
- Unknown `Type` or `Worker` value: DEBUG log + `data_events_consumer_skipped_total{reason}` counter where `reason ∈ {unknown_type, unrelated_worker}`. Offset committed.

### 4.10 Observability

In `atlas-data`:

- `atlas_data_events_emitted_total{worker, type}` — counter, incremented on every successful Kafka emit.
- `atlas_data_events_emit_failures_total{worker, type}` — counter, incremented when the producer returns an error.

In `atlas-monsters` and `atlas-maps` (and any future consumer):

- `atlas_<service>_data_events_processed_total{worker, type, action}` — counter where `action ∈ {flushed, skipped}`.
- `atlas_<service>_data_events_consumer_errors_total{kind}` — counter where `kind ∈ {parse, flush}`.
- `atlas_<service>_data_events_consumer_skipped_total{reason}` — counter where `reason ∈ {unknown_type, unrelated_worker}`.
- For `atlas-monsters`, the existing `atlas_monsters_data_cache_evictions_total{tenant, reason}` from task-060 gains a new label value `reason="invalidation"` distinct from `expired_positive`/`expired_negative`. The flush call increments this counter by the number of entries evicted.
- For `atlas-maps`, equivalent eviction counter on the spawn-point registry: `atlas_maps_spawn_registry_evictions_total{tenant, reason}` with `reason ∈ {ttl, invalidation}` (TTL may already exist; invalidation is the new value introduced here).

### 4.11 Testing

`atlas-data` producer:

- Unit test: on worker success, the producer is called once with the correct `tenantId`, `worker`, and a `completedAt` within the last second.
- Unit test: on worker failure, the producer is NOT called.
- Unit test: with `DATA_EVENTS_PRODUCER_ENABLED=false`, the producer is NOT called regardless of worker outcome.
- Unit test: producer error is logged but does not fail the worker (StartWorker returns nil).

`libs/atlas-cache` (task-060 module):

- Unit test: `Flush` empties a populated cache.
- Unit test: `Flush` is concurrent-safe; race-detector clean under parallel `Get`/`Put`/`Flush`.
- Unit test: `FlushTenant` on the per-tenant registry only clears the targeted tenant's cache; other tenants' entries remain.
- Unit test: when cache is disabled, `FlushTenant` is a no-op and does not allocate a per-tenant cache.

`atlas-monsters` consumer:

- Unit test: a `DATA_UPDATED` event with `Worker=MONSTER` triggers `FlushTenant` for the right tenant.
- Unit test: a `DATA_UPDATED` event with `Worker=MAP` is ignored (no flush).
- Unit test: a malformed payload increments the parse-error counter and does not crash the consumer.
- Unit test: `DATA_EVENTS_CONSUMER_ENABLED=false` skips consumer registration.

`atlas-maps` consumer:

- Unit test: a `DATA_UPDATED` event with `Worker=MAP` for tenant T deletes only `atlas:maps:spawn:T:*` keys. Other tenants' keys remain.
- Unit test: a `DATA_UPDATED` event with `Worker=MONSTER` triggers the same flush (per §4.6 conservative default; revisit at design time).
- Unit test: a `DATA_UPDATED` event with `Worker=NPC` (or any non-map/non-monster worker) is ignored.
- Unit test: a Redis network error during flush increments the flush-error counter, commits the offset, and does not crash the consumer.

End-to-end (integration / manual):

- Trigger a tenant data import via the existing `START_WORKER` mechanism for `WorkerMonster` against tenant T.
- Observe within 5 s of worker completion: `atlas_data_events_emitted_total{worker="MONSTER"}` increments by 1; on every replica of `atlas-monsters` and `atlas-maps`, `data_events_processed_total{worker="MONSTER", action="flushed"}` increments by 1.
- Observe in `atlas-monsters` cache metrics: positive entry count for tenant T drops to 0 immediately after the event, then rebuilds on next request.
- Observe in Redis: `KEYS atlas:maps:spawn:{T}:*` returns empty after the event, then rebuilds as maps are re-activated.

## 5. API Surface

No HTTP / JSON:API surface changes. No new REST endpoints. No new request/response shapes for any consumer of `atlas-data`'s HTTP API.

New Kafka surface:

- New topic `EVENT_TOPIC_DATA` (env-var configurable name).
- New event type discriminator `DATA_UPDATED` with body shape defined in §4.1.

New internal Go API:

- `libs/atlas-cache.Cache.Flush()` — clears all entries.
- `libs/atlas-cache` per-tenant registry helper `FlushTenant(tenant.Id)`.
- `atlas-monsters/monster/information.FlushTenant(tenant.Id)` — exposed by the cache wrapper for the consumer to call.
- `atlas-maps/map/monster/registry.FlushTenant(tenant.Id)` — sibling to existing registry methods.

All other internal Go APIs are unchanged.

## 6. Data Model

No persistent data changes. No database migrations.

In-memory / Redis state changes:

- `libs/atlas-cache` gains an in-memory `Flush()` operation. Internally implemented by replacing the underlying map with a fresh empty one under the existing `sync.RWMutex` write-lock, or by deleting all keys under the lock — implementation detail.
- `atlas-maps` Redis spawn-point registry gains a per-tenant flush operation that internally `SCAN`s for `atlas:maps:spawn:{tenantId}:*` and pipelines `DEL`s. The existing global-scan iterator at `registry.go:260` is the model.

## 7. Service Impact

### 7.1 `services/atlas-data` (producer)

- New Kafka producer wiring at the bottom of `StartWorker` per worker success.
- New event types in `data/kafka.go` (envelope + body) plus `EnvEventTopic` constant.
- New `producer.go` provider function `dataUpdatedEventProvider`.
- New env var `DATA_EVENTS_PRODUCER_ENABLED`.
- New metrics §4.10.

### 7.2 `libs/atlas-cache` (depends on task-060)

- New public method `Cache.Flush()`.
- New per-tenant registry helper `FlushTenant(tenant.Id)`.
- README updated with the new API and the per-pod-group consumer convention.
- Tests for new code paths.

### 7.3 `services/atlas-monsters` (proof consumer #1)

- New consumer subtree `kafka/consumer/data/`.
- New `monster/information.FlushTenant(tenant.Id)` exposing the per-tenant flush from task-060's registry.
- New env var `DATA_EVENTS_CONSUMER_ENABLED`.
- `main.go` registers the new consumer using the per-pod group convention.
- New metrics §4.10.

### 7.4 `services/atlas-maps` (proof consumer #2)

- New consumer subtree `kafka/consumer/data/`.
- New `map/monster/registry.FlushTenant(tenant.Id)` that scans `atlas:maps:spawn:{tenantId}:*` and pipelines `DEL`s.
- New env var `DATA_EVENTS_CONSUMER_ENABLED`.
- `main.go` registers the new consumer using the per-pod group convention.
- New metrics §4.10.

### 7.5 Other services

- No code changes in this task.
- Future follow-ups (out of scope here):
  - `atlas-channel` (consumes `data/maps`, `data/portals`, `data/skills`, `data/npcs`, `data/quests`, monster info).
  - `atlas-monster-death` (consumes monster info, drop position).
  - `atlas-pets` (consumes data/position).
  - `atlas-reactors` (consumes reactor data).
  - `atlas-portals`, `atlas-consumables`, `atlas-chairs`, `atlas-transports`, `atlas-character`, `atlas-quest`, `atlas-effective-stats`, `atlas-merchant`, `atlas-saga-orchestrator`, `atlas-character-factory`, `atlas-configurations`, `atlas-messages` — all currently call `atlas-data` HTTP endpoints uncached and will need invalidation wiring once they add caching.

### 7.6 Deploy / infra

- `EVENT_TOPIC_DATA` env var added to the deploy manifests for `atlas-data`, `atlas-monsters`, `atlas-maps`.
- Topic creation: confirm at design time whether the broker auto-creates topics or whether a bootstrap manifest (Helm chart, Strimzi custom resource, etc.) needs an entry. If the latter, add the topic with replication factor and partition count matching the existing `COMMAND_TOPIC_DATA` topic.

## 8. Non-Functional Requirements

### 8.1 Performance

- Producer adds at most one Kafka send per successful worker run. Kafka send is asynchronous; it MUST NOT block worker completion by more than a single network round-trip even on the slow path.
- Consumer flush is O(N) in cache size (in-process) or O(K) in Redis key count under the tenant prefix (atlas-maps). Both are bounded by tenant-local entity counts (a few thousand at most).
- End-to-end emit-to-flush latency target: < 5 s under normal Kafka load. No SLO; budget is dominated by Kafka rebalance/poll cycles.

### 8.2 Correctness

- No event leak across tenants under any code path.
- Idempotency: a duplicate event causes a redundant flush (no-op on already-empty cache), never incorrect state.
- A failed consumer flush MUST NOT leave the cache in a half-cleared state. Implementation either does the flush atomically (in-process, single mutex section) or accepts that Redis `DEL` pipelining can partially succeed but converges on the next event / TTL.
- Race-detector clean: `go test -race ./...` in all changed modules.

### 8.3 Observability

- Metrics §4.10 are present and labeled.
- Operator-facing dashboard panels can answer:
  - "Are events being emitted?" — `rate(atlas_data_events_emitted_total[5m])`.
  - "Are consumers keeping up?" — emit rate vs. processed rate per consumer.
  - "Are flushes failing?" — `atlas_<service>_data_events_consumer_errors_total`.
- Dashboard authoring is OUT of scope; metric availability is in scope.

### 8.4 Security & multi-tenancy

- Tenant scoping enforced at the event payload level. Consumers MUST flush only the tenant in `event.Body.TenantId`.
- No new external network egress or new secrets.
- No PII or auth data on the topic (just tenant UUID, worker name, timestamp).

### 8.5 Operability

- `DATA_EVENTS_PRODUCER_ENABLED=false` rolls the producer back without code change.
- `DATA_EVENTS_CONSUMER_ENABLED=false` rolls a single consumer back without code change.
- The existing manual invalidation runbook (`DEL atlas:maps:spawn:*` + monster pod restart) remains documented as a fallback in the user's `reference_atlas_maps_spawn_cache.md` memory note. This task does not retire it.
- No CLI / admin endpoint to manually trigger an event. Operators trigger by re-running the import for the relevant tenant/worker.

## 9. Open Questions

- **Topic name finalization** — `EVENT_TOPIC_DATA` (proposed, mirrors `COMMAND_TOPIC_DATA`) vs. a more specific `EVENT_TOPIC_DATA_LIFECYCLE` or `EVENT_TOPIC_DATA_UPDATES`. User preference is `EVENT_TOPIC_DATA` to leave room for additional event types beyond invalidation. Confirmed.
- **Whether `atlas-maps` should flush on `WorkerMonster`** — §4.6 conservatively says yes, but this assumes monster reference data feeds into the spawn registry. If verification at design time shows the spawn registry is built purely from map-data (`Mob` entries inside `Map.wz`), drop the `WorkerMonster` branch.
- **`HOSTNAME` fallback** — UUID at startup is proposed (§4.7). Alternative: fail-fast with a startup error. Default is the UUID fallback to keep local dev painless; revisit if production deploys ever hit the fallback path.
- **Topic bootstrap** — does the project rely on broker auto-creation, or is there a bootstrap manifest that needs updating? Affects the deploy checklist.
- **`libs/atlas-kafka` per-pod-group helper** — does the existing library expose a convenient "unique-per-pod group" helper, or do consumer-init files have to hand-roll the suffix? Affects the cleanliness of the consumer wiring.
- **Consumer commit semantics on flush failure** — §4.9 says commit-and-continue. If the project's Kafka library doesn't allow per-message commit control, behavior may default to whatever the library does. Confirm at design time.
- **Future event types on the same topic** — `DATA_IMPORT_STARTED`, `DATA_IMPORT_FAILED` are plausible additions. Out of scope to design here, but the discriminator `Type` field is sized for it.

## 10. Acceptance Criteria

A reviewer can mark this task done when ALL of the following are true:

- [ ] Task-060 (`libs/atlas-cache` + `atlas-monsters` in-process cache) is merged on `main` and this branch is rebased on top of it.
- [ ] `atlas-data` emits one `DATA_UPDATED` event per successful (tenant, worker) run on `EVENT_TOPIC_DATA` with the §4.1 envelope and body. No event is emitted on worker failure.
- [ ] `EVENT_TOPIC_DATA` is set in the deploy manifests for `atlas-data`, `atlas-monsters`, `atlas-maps`.
- [ ] `libs/atlas-cache` exposes `Cache.Flush()` and a per-tenant registry `FlushTenant` helper, both race-clean.
- [ ] `atlas-monsters` registers a Kafka consumer for `EVENT_TOPIC_DATA` using the per-pod group convention `atlas-monsters data-events <HOSTNAME>` with `auto.offset.reset=latest`. On `DATA_UPDATED` with `Worker=MONSTER`, the per-tenant cache for `TenantId` is flushed.
- [ ] `atlas-maps` registers a Kafka consumer for `EVENT_TOPIC_DATA` using the per-pod group convention. On `DATA_UPDATED` with `Worker=MAP` (and `Worker=MONSTER` per §4.6 default, unless dropped at design time), the spawn-point registry for `TenantId` is flushed via `SCAN`/`DEL` of `atlas:maps:spawn:{TenantId}:*`.
- [ ] All §4.10 metrics are emitted and labeled correctly.
- [ ] All §4.11 unit tests pass; `go test -race ./...` is clean in `atlas-data`, `libs/atlas-cache`, `atlas-monsters`, `atlas-maps`.
- [ ] Manual end-to-end verification (per §4.11): trigger a `WorkerMonster` import for a test tenant, observe within 5 s that `atlas-monsters`' cache size for that tenant drops to 0 on every replica and rebuilds on next access; observe that `atlas-maps` Redis spawn keys for that tenant are deleted and rebuild on next map activation.
- [ ] Setting `DATA_EVENTS_PRODUCER_ENABLED=false` (in `atlas-data`) suppresses event emission without code change. Setting `DATA_EVENTS_CONSUMER_ENABLED=false` (in either consumer) suppresses consumer registration without code change. Both verified.
- [ ] User memory note `reference_atlas_maps_spawn_cache.md` is updated (or annotated in the PR description) noting that automatic invalidation now exists for `MAP`/`MONSTER` workers and the manual `DEL` runbook is now a fallback rather than the primary path.
- [ ] PRD updated to status `Implemented` and `Date implemented` on merge.
