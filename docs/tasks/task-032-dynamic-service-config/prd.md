# Dynamic Service Configuration — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-17
---

## 1. Overview

Today, atlas-channel and atlas-login bootstrap by making synchronous REST calls to atlas-configurations to fetch their service and tenant configuration. The configuration package fatals the process if the call fails or the data is missing (`services/atlas-channel/atlas.com/channel/configuration/registry.go:19`, `:27`, `:35`, `:47`, `:55`). Because every per-tenant listener, per-`(t,w,c)` Kafka handler registration, per-tenant account registry initialization, and TCP listener binding is computed eagerly inside `main()` from the returned `config.Tenants` (`services/atlas-channel/atlas.com/channel/main.go:209-380`), the boot is a hard linear dependency on atlas-configurations being reachable *and* having the right rows at the moment the pod starts. Adding or removing a tenant requires restarting every consuming pod.

This task replaces the synchronous REST dependency with a Kafka-driven event stream. atlas-configurations gains a transactional outbox so every CRUD on services/tenants durably emits a config event to a log-compacted Kafka topic. atlas-channel and atlas-login subscribe, build a local projection of the current state, and gate readiness on having caught up to the topic's boot-time end offset. Once running, they react live to ADD/UPDATE/REMOVE events: bringing up new listeners, refreshing tenants whose protocol tables changed, and draining tenants that disappear from config — without restarting the pod.

To make REMOVE safe, atlas-channel grows a first-class per-`(tenant, world, channel)` listener-handle lifecycle and a four-phase drain primitive. Phase 1 deregisters the listener locally (so the existing 10s heartbeat in `services/atlas-channel/atlas.com/channel/channel/task.go:36-39` stops re-registering it) and calls `atlas-world.Unregister(ch)` so login immediately stops advertising the channel; phase 2 walks per-listener sessions and runs the existing save-and-kick logout path; phase 3 waits for the per-listener `sync.WaitGroup` to drain (deadline-bounded); phase 4 cancels the listener context (stopping `socket.Run` in `services/atlas-channel/atlas.com/channel/socket/init.go:41`) and deregisters every Kafka handler registered for that scope. This requires threading the handler-IDs returned by `consumer.Manager.RegisterHandler` (`libs/atlas-kafka/consumer/manager.go:149`) back through every `InitHandlers` factory, which currently discards them (e.g. `services/atlas-channel/atlas.com/channel/kafka/consumer/account/consumer.go:34`).

The outbox itself becomes a reusable library — `libs/atlas-outbox` — because the same atomic "DB commit + Kafka emit" need exists elsewhere in the codebase (`docs/TODO.md` flags this as an outstanding architectural issue for multi-topic operations). Only atlas-configurations adopts it in this task; other adopters (notably the saga orchestrator) will follow in separate tasks.

## 2. Goals

Primary goals:
- atlas-channel and atlas-login no longer require atlas-configurations to be reachable at boot. They subscribe to two log-compacted Kafka topics and build their config view from the stream.
- Adding or removing a tenant from atlas-configurations causes affected atlas-channel/atlas-login pods to dynamically add/drain listeners without a restart.
- A REMOVE during active play drains sessions gracefully via the existing save-and-kick pipeline, with a bounded deadline (5s default, 10s ceiling) and crash-safe ordering of events vs. socket close.
- The outbox library guarantees no config event is published unless the DB row was committed (and vice versa), supports multi-replica atlas-configurations deployments, and uses LISTEN/NOTIFY for low-latency wake-up rather than polling-only.
- Kafka-handler deregister becomes a first-class primitive, plumbed through every atlas-channel consumer package.

Non-goals:
- Saga orchestrator adoption of the outbox library — separate task.
- Multi-owner channels / HA channel migration (option (c) from brainstorming).
- atlas-configurations character-creation `templates/*` — they stay on REST in this task.
- Atomic multi-topic emission for non-config domains (e.g. inventory transfers) — separate task even though the library will support it.
- atlas-login per-tenant graceful drain coordination beyond stop-accepting + close. Login sessions are short-lived and have no save pipeline.
- Saga cancellation on session destroy. Existing wart; not made worse by this work, not in scope to fix.
- Schema-registry-backed schemas (Avro/protobuf). JSON envelopes with additive evolution are policy.

## 3. User Stories

- As an **operator**, I want to add a new tenant to atlas-configurations without restarting every atlas-channel and atlas-login pod, so that onboarding a tenant doesn't require coordinated downtime.
- As an **operator**, I want to remove a tenant from atlas-configurations and have channel/login pods drain that tenant's sessions cleanly within seconds, so that decommissioning a tenant has predictable, observable behavior.
- As an **operator**, I want atlas-channel and atlas-login to boot successfully even when atlas-configurations is unreachable, so that single-service outages don't cascade into broader unavailability.
- As an **operator**, I want a Kafka-backed audit trail of every config change, so that "what changed when" is answerable without reading DB binlogs.
- As an **atlas-configurations developer**, I want a single library call to durably publish an event in the same transaction as a DB write, so that I can't accidentally introduce a divergent-state bug.
- As a **future service author** (saga orchestrator, others), I want a reusable outbox library so I don't have to re-implement transactional-outbox semantics per service.
- As a **player**, I want my character to save and disconnect cleanly (not lose progress) if my channel is reconfigured while I'm playing, so that operational changes don't corrupt my state.

## 4. Functional Requirements

### 4.1 `libs/atlas-outbox` — transactional outbox library

- **FR-OUT-1.** Library exposes `outbox.Enqueue(tx *gorm.DB, msg outbox.Message) error` that inserts a row into the caller's `outbox_entries` table using the provided GORM transaction. The caller is responsible for owning the surrounding transaction; the library does not start or commit transactions.
- **FR-OUT-2.** `outbox.Message` contains: `Topic string` (resolved Kafka topic name), `Key []byte` (Kafka message key, required), `Value []byte` (nullable; null = tombstone), `Headers map[string]string` (optional; defaults captured by drainer if absent).
- **FR-OUT-3.** Library provides a `Drainer` constructed via `outbox.NewDrainer(l, db, producer, opts...)` and run as `drainer.Run(ctx)`. The drainer holds a `pg_advisory_lock` named after the table; only the lock-holder publishes. Non-holders idle-poll for the lock so any replica can take over on failover.
- **FR-OUT-4.** The drainer wakes on `pg_notify` and additionally polls at a configurable interval (default 1s). `Enqueue` issues a `NOTIFY` on the same channel inside the caller's transaction so commit triggers an immediate wake-up.
- **FR-OUT-5.** The drainer fetches unsent rows with `SELECT ... WHERE sent_at IS NULL ORDER BY enqueued_at LIMIT batch_size FOR UPDATE SKIP LOCKED`, publishes via the existing `atlas-kafka` producer manager, and `UPDATE`s `sent_at` on success. On Kafka error, increments `attempts`, records `last_error`, releases the row, and retries on the next tick.
- **FR-OUT-6.** Publish semantics are at-least-once: a crash between Kafka publish and `sent_at` UPDATE causes the row to be republished. Library documents that consumers must be idempotent.
- **FR-OUT-7.** Library provides an idempotent migration helper `outbox.Migration(db)` that creates the `outbox_entries` table and required indexes. Each adopting service runs this in its own migration chain.
- **FR-OUT-8.** Library provides `outbox.Backfill(db, topic, keyFn, valueFn, sourceTable)` that re-enqueues rows from a source table where no outbox row currently exists for that key. Used by atlas-configurations seeder to handle fresh-cluster bootstrap. Safe to invoke on every startup; no-op when topics are already populated relative to the source DB.
- **FR-OUT-9.** Drainer cleanup: rows with `sent_at IS NOT NULL` older than 7 days are deleted by a periodic sweeper (interval configurable; default 1h). Default retention is operational state, not the audit trail — the audit trail lives in the Kafka topic.
- **FR-OUT-10.** The `outbox_entries` table is **not** tenant-scoped. The library's GORM session disables the `tenant_scope.go` callbacks for queries against its own entity.
- **FR-OUT-11.** Drainer logs structured events on every transition: lock acquired, lock lost, batch published (n rows), publish failure, sweeper run. Lock acquisition success/failure is visible at INFO; publish-per-row at DEBUG.

### 4.2 `libs/atlas-kafka` — end-offset query helper

- **FR-KAF-1.** Add `consumer.ReadEndOffsets(ctx, brokers []string, topic string) (map[int]int64, error)` that returns the current end offset per partition for the given topic. Used by subscribers to compute the "caught up" threshold at boot. Implementation uses `kafka-go`'s `Conn.ReadPartitions` + `Conn.ReadOffsets`.

### 4.3 atlas-configurations — event publishing

- **FR-CFG-1.** New env vars (and broker-side topics): `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`, `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`. Both topics are configured as log-compacted, single partition, with `delete.retention.ms` ≥ 7 days. Topic provisioning is operational; the service does not auto-create with these settings.
- **FR-CFG-2.** Migration adds `outbox_entries` table via `outbox.Migration` and registers it alongside existing migrations in `services/atlas-configurations/atlas.com/configurations/main.go`.
- **FR-CFG-3.** `services.Processor.Create`, `Update`, `Delete` use `database.ExecuteTransaction` (already the case at `services/processor.go:130, 154, 158`) to wrap entity changes; inside that callback, call `outbox.Enqueue` with the service envelope payload. Key: `service:<service-uuid>`. Value: envelope JSON for Create/Update; nil for Delete (tombstone).
- **FR-CFG-4.** `tenants.Processor` (analogous methods) same shape, topic = tenant topic. Key: `tenant:<tenant-uuid>`.
- **FR-CFG-5.** `main.go` initializes the outbox drainer with `producer.GetManager()` and registers its shutdown in the existing teardown manager.
- **FR-CFG-6.** Seeder (`seeder/seeder.go`) runs `outbox.Backfill` for both topics after its existing seed-from-JSON pass. The backfill is idempotent: rows already represented in `outbox_entries` are skipped. This handles fresh-cluster bootstrap without inventing a separate code path.

### 4.4 Topic envelope schema

- **FR-SCH-1.** Both topics carry JSON envelopes with this shape:
  ```json
  {
    "schema_version": 1,
    "id": "<uuid>",
    "config": { /* existing RestModel shape, verbatim */ },
    "emitted_at": "<RFC3339>"
  }
  ```
  `id` mirrors the Kafka key. `config` for the service topic is the existing `services/service.ChannelRestModel` or `LoginRestModel` shape (per service type). `config` for the tenant topic is `tenant.RestModel`.
- **FR-SCH-2.** Schema evolution policy: additive only — new fields may be added, existing fields may not be removed, renamed, or have their semantics changed. Consumers ignore unknown fields. If an incompatible change is genuinely needed, a new topic is minted (e.g. `..._STATUS_V2`); the schema_version field alone is not used to switch interpretation.
- **FR-SCH-3.** Deletion is a Kafka message with null value. Consumers project this as removal of the key from local state.
- **FR-SCH-4.** Headers carry span context only (via existing `consumer.SpanHeaderParser`). No tenant header on either topic — the tenant key for the tenant topic is the message key; the service topic is cross-tenant.

### 4.5 atlas-channel — subscriber, projection, listener lifecycle, drain

#### 4.5.1 Subscriber and projection

- **FR-CHN-1.** Replace `configuration/registry.go` global `sync.Once`-guarded REST pull with a Kafka subscriber. Two consumer subscriptions: one for the service topic, one for the tenant topic. Both start at earliest offset; both apply messages into local projection state.
- **FR-CHN-2.** Local projection has two key spaces: `serviceConfig` (singleton — the current service's own config row, filtered by `SERVICE_ID`) and `tenantConfigs map[uuid.UUID]tenant.RestModel`. Updated under a single `sync.RWMutex`.
- **FR-CHN-3.** Eager-apply tenant events, lazy-reference from service config. Every tenant event updates `tenantConfigs` unconditionally. Service-config changes (including initial caught-up state) recompute the desired listener set from `serviceConfig.Tenants` × the current `tenantConfigs` map.
- **FR-CHN-4.** At boot, before subscribing, query end offsets via `consumer.ReadEndOffsets` for both topics. Mark the pod "caught up" only once consumed offsets ≥ snapshotted end offsets on every partition of both topics. `/readyz` returns not-ready until then.
- **FR-CHN-5.** "Caught up" is one-way: once true for the lifetime of the process, it stays true. Subsequent transient consumer errors do not flip it back.
- **FR-CHN-6.** On SIGTERM, `/readyz` immediately flips to not-ready (before the drain begins) so k8s stops routing new traffic.

#### 4.5.2 Listener lifecycle

- **FR-CHN-7.** Introduce `listener.Handle` keyed by `(tenantId, worldId, channelId)`. Fields: `state` (`Active | Draining | Removed`), `ctx`/`cancel` (child of pod ctx), `wg` (`*sync.WaitGroup`), `serverModel`, `kafkaHandlerIds []HandlerHandle{Topic, Id}`. Stored in a process-global `listener.Registry` keyed by the same triple.
- **FR-CHN-8.** `Registry.Add(key, ...)` is idempotent — re-applying an unchanged config is a no-op. A config change to a previously-registered scope where any of `port`, region/version (from tenant), or socket-tables changed results in `Drain(key)` followed by `Add(key, newCfg)`.
- **FR-CHN-9.** `server.Registry` (currently `services/atlas-channel/atlas.com/channel/server/registry.go`) gains a `Deregister(key)` method keyed by `(t,w,c)`. Backing store becomes a map.
- **FR-CHN-10.** The heartbeat task (`channel/task.go:36-39`) iterates `server.GetAll()`. After Deregister, the heartbeat naturally skips the drained scope — no separate change needed.

#### 4.5.3 Four-phase drain primitive

- **FR-CHN-11.** `Drain(key)` is sequenced as:
  1. **Quiesce.** Set `state = Draining`. Call `server.Registry.Deregister(key)` (heartbeat stops touching it). Call `channel.NewProcessor(...).Unregister(sc.Channel())` (new REST client, mirrors existing Register) — atlas-world drops the entry; login stops advertising on next query.
  2. **Save-and-kick.** Walk `session.Registry` filtered by `(t,w,c)`. For each session, send a "server is going down" status message packet, then run the existing `session.Processor.Destroy(s)` path (`session/processor.go:330`) which emits the logout command and SessionStatusDestroyed event before closing the socket (see FR-CHN-14 — emit ordering).
  3. **Drain deadline.** `select { case <-allClosed: ; case <-time.After(drainDeadline): }`. Default deadline 5s, ceiling 10s, exposed via env var.
  4. **Tear down.** Cancel the listener's `ctx` (stops `socket.Run`). Call `consumer.GetManager().RemoveHandler(topic, id)` for each captured handler-ID. Wait the listener `wg`. Mark `state = Removed`.
- **FR-CHN-12.** `Drain(key)` is idempotent. Calling it for a non-existent or already-`Removed` key is a no-op.
- **FR-CHN-13.** On pod-shutdown SIGTERM, the pod-level teardown calls `Drain(key)` for every active listener in parallel (bounded by k8s `terminationGracePeriodSeconds`, recommended 15-20s).

#### 4.5.4 Crash-safe session destroy

- **FR-CHN-14.** Reorder `session.Processor.Destroy` (`session/processor.go:330-336`) to emit the logout command and SessionStatusDestroyed event **before** calling `s.Disconnect()`. Today's order is registry-remove → Disconnect → emit-logout → emit-destroy. Reordering guarantees that if the pod dies after socket close but before the producer's `BatchTimeout` flush, downstream services have already seen the destroy event in Kafka. (The producer's batching is the source of the gap; a synchronous flush per session is too expensive at scale.)

#### 4.5.5 Threading handler-IDs through `InitHandlers`

- **FR-CHN-15.** Every package under `services/atlas-channel/atlas.com/channel/kafka/consumer/*/consumer.go` changes its `InitHandlers` factory signature from `... func(rf RegFunc) error` to `... func(rf RegFunc) ([]HandlerHandle, error)`, where `HandlerHandle = struct { Topic string; Id string }`. The factory captures the `(topic, id)` returned by each `rf(...)` call into the slice instead of discarding it.
- **FR-CHN-16.** `main.go` (`services/atlas-channel/atlas.com/channel/main.go:246-374`) collects each `InitHandlers` call's returned handles into the `listener.Handle` for that `(t,w,c)`. Drain phase 4 iterates them and calls `RemoveHandler`.
- **FR-CHN-17.** Same shape applies to atlas-login's consumer packages (smaller set).

#### 4.5.6 Tenant-Evict hooks

- **FR-CHN-18.** Add an `Evict(t tenant.Model)` method to each tenant-scoped local-state singleton: `monster.GetStatusMirror()`, `monster.GetNextSkillInbox()`, `account.GetRegistry()`. When the last `listener.Handle` for a given tenant transitions to `Removed`, the lifecycle code invokes `Evict(t)` on each and calls `tenant.Unregister(t.Id())` (new method on the global `tenant` registry).
- **FR-CHN-19.** `account.NewProcessor(...).InitializeRegistry()` (per-tenant boot step, `main.go:223`) is moved out of `main.go` into the listener-Add path and gains a corresponding teardown invoked from the Evict hook chain.

### 4.6 atlas-channel — channel-side Unregister REST client

- **FR-CHN-20.** Add `Unregister(ch channel.Model) error` to `channel.Processor` (`services/atlas-channel/atlas.com/channel/channel/processor.go:14`) and a mirroring REST request in `requests.go`. Calls atlas-world's existing endpoint corresponding to `Unregister(ctx, ch)` (`services/atlas-world/atlas.com/world/channel/processor.go:96`). Verify the atlas-world REST surface exposes the endpoint; if it does not, atlas-world gets a minimal addition in this task (REST DELETE on the channel resource).

### 4.7 atlas-login — subscriber, projection, simple drain

- **FR-LGN-1.** Same projection shape and caught-up readiness gate as atlas-channel (FR-CHN-1 through FR-CHN-6).
- **FR-LGN-2.** Listener lifecycle is simpler: on `(t, w, c)` REMOVE, stop accepting new connects on the listener, send a "server unavailable, please reconnect" status packet on existing sessions, close the socket, and tear down handlers (FR-CHN-15 — same handler-ID threading applies). No save-and-kick; login sessions are stateless past handshake.
- **FR-LGN-3.** Login does **not** need to consume the tenant-status topic for drain coordination — it queries atlas-world for channel availability and already gets the right answer when atlas-channel calls `Unregister`. It does subscribe to both topics for its own config needs.

## 5. API Surface

### 5.1 `libs/atlas-outbox` (Go)

```go
package outbox

type Message struct {
    Topic   string
    Key     []byte
    Value   []byte            // nil = tombstone
    Headers map[string]string // optional
}

// Enqueue inserts an outbox row inside the caller's transaction.
// MUST be called with tx representing an open transaction; library will not commit.
func Enqueue(tx *gorm.DB, msg Message) error

// Migration is added to the caller's GORM migration chain.
func Migration(db *gorm.DB) error

// Backfill scans sourceTable and re-enqueues any rows whose key is not present
// in outbox_entries. Idempotent and safe on every startup.
func Backfill(db *gorm.DB, topic string, keyFn, valueFn func(any) ([]byte, error), sourceTable string) error

type Drainer struct { /* ... */ }

type DrainerOption func(*drainerConfig)

func WithPollInterval(d time.Duration) DrainerOption   // default 1s
func WithBatchSize(n int) DrainerOption                 // default 100
func WithSweeperInterval(d time.Duration) DrainerOption // default 1h
func WithRetention(d time.Duration) DrainerOption       // default 7d

func NewDrainer(l logrus.FieldLogger, db *gorm.DB, pm *producer.Manager, opts ...DrainerOption) *Drainer

func (d *Drainer) Run(ctx context.Context)
func (d *Drainer) Stop()
```

### 5.2 `libs/atlas-kafka` additions

```go
package consumer

func ReadEndOffsets(ctx context.Context, brokers []string, topic string) (map[int]int64, error)
```

### 5.3 Kafka topics

| Env var | Broker topic name | Partitions | Cleanup |
|---|---|---|---|
| `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` | `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` | 1 | `compact` |
| `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` | `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` | 1 | `compact` |

Envelope shape (both topics):

```json
{
  "schema_version": 1,
  "id": "<uuid>",
  "config": { /* existing RestModel verbatim */ },
  "emitted_at": "<RFC3339>"
}
```

Deletion: Kafka message with `Value: nil` and the corresponding key.

### 5.4 atlas-channel internal APIs

```go
package server

func (r *Registry) Deregister(key Key) error
func (r *Registry) Get(key Key) (Model, bool)

type Key struct {
    TenantId  uuid.UUID
    WorldId   world.Id
    ChannelId channel.Id
}
```

```go
package listener

type State int
const (
    Active State = iota
    Draining
    Removed
)

type HandlerHandle struct {
    Topic string
    Id    string
}

type Handle struct {
    Key             server.Key
    State           State
    Ctx             context.Context
    Cancel          context.CancelFunc
    Wg              *sync.WaitGroup
    ServerModel     server.Model
    KafkaHandlerIds []HandlerHandle
}

func (r *Registry) Add(key server.Key, /* config */) error    // idempotent
func (r *Registry) Drain(key server.Key) error                // 4-phase; idempotent
func (r *Registry) Snapshot() []Handle                        // for /debug + shutdown
```

```go
package channel  // atlas-channel client of atlas-world

func (p *ProcessorImpl) Unregister(ch channel.Model) error  // new
```

### 5.5 atlas-world endpoint (verify or add)

`DELETE /api/world-server/channel-server/{worldId}/{channelId}` or equivalent JSON:API route mapping to `channel.Processor.Unregister`. Verify it exists; add if missing.

## 6. Data Model

### 6.1 `outbox_entries` (added per adopting service; first adopter: atlas-configurations)

```sql
CREATE TABLE outbox_entries (
    id            BIGSERIAL PRIMARY KEY,
    topic         TEXT NOT NULL,
    message_key   BYTEA NOT NULL,
    message_value BYTEA,                    -- NULL = tombstone
    headers       JSONB NOT NULL DEFAULT '{}',
    enqueued_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at       TIMESTAMPTZ,
    attempts      INTEGER NOT NULL DEFAULT 0,
    last_error    TEXT
);

CREATE INDEX outbox_entries_unsent_idx
    ON outbox_entries (enqueued_at)
    WHERE sent_at IS NULL;

CREATE INDEX outbox_entries_sweeper_idx
    ON outbox_entries (sent_at)
    WHERE sent_at IS NOT NULL;
```

Not tenant-scoped — operational table, no `tenant_id` column, GORM tenant-scope callbacks must be disabled for this entity.

### 6.2 No schema changes to existing atlas-configurations tables

Service and tenant entities (`services.Entity`, `tenants.Entity`) remain unchanged. Existing JSON-blob payloads are passed through to outbox envelopes verbatim.

### 6.3 No persistent schema changes in atlas-channel or atlas-login

All projection state is in-memory. No new tables.

## 7. Service Impact

### 7.1 `libs/atlas-outbox` (new)

New library. Files: `outbox.go` (Enqueue, Message), `drainer.go` (lock, NOTIFY, publish loop, sweeper), `migration.go`, `backfill.go`, `entity.go`. Mirrors structure of existing `libs/atlas-kafka` and `libs/atlas-database`.

### 7.2 `libs/atlas-kafka`

One new function: `ReadEndOffsets`. New file `consumer/offsets.go`. No interface changes to existing types.

### 7.3 `atlas-configurations`

- `main.go`: register `outbox.Migration` in DB init; initialize and run `outbox.Drainer`.
- `services/processor.go`: in `Create`, `Update`, `Delete`, call `outbox.Enqueue` inside the existing `database.ExecuteTransaction` callback.
- `tenants/processor.go`: same pattern.
- `seeder/seeder.go`: call `outbox.Backfill` after seed-from-JSON.
- Dockerfile / k8s manifest: add the two new topic env vars.

### 7.4 `atlas-channel`

- `main.go`: replace `configuration.Init(...) + GetServiceConfig()` block. Build listener registry from projection; move per-listener startup work (account registry init, `server.Register`, Kafka handler registration, `CreateSocketService`) into `listener.Registry.Add`. Wire `/readyz` to caught-up state.
- `configuration/registry.go`: replaced by Kafka subscriber + projection (new package: `configuration/projection`).
- `server/registry.go`: slice → map; add `Deregister(key)`.
- `server/model.go`: no shape change, but `Is(t,w,c)` may consult `Registry` state to natively reject Draining/Removed scopes (alternative: drain just removes handlers and that's enough — choose simplest path during design).
- `session/processor.go`: reorder `Destroy` (FR-CHN-14).
- `channel/processor.go` and `channel/requests.go`: add `Unregister`.
- `channel/task.go`: heartbeat already iterates `server.GetAll()`; no change needed once Registry uses `Deregister`.
- `kafka/consumer/**/consumer.go`: change `InitHandlers` signature to return `[]HandlerHandle` across all ~40 packages.
- Per-tenant local-state singletons: add `Evict(t)` to `monster.GetStatusMirror`, `monster.GetNextSkillInbox`, `account` registry, global `tenant` registry.
- New `listener` package implementing `Handle`, `Registry`, `Add`, `Drain`.
- Tracker: env var for drain deadline. Dockerfile/k8s manifest gets the two new topic env vars and (if needed) `terminationGracePeriodSeconds` bump to 15-20s.

### 7.5 `atlas-login`

- `main.go`: same projection swap; smaller listener lifecycle (no save-and-kick).
- `configuration/registry.go`: replaced same as atlas-channel.
- `kafka/consumer/**/consumer.go`: same `InitHandlers` signature change for the smaller consumer set.
- Dockerfile / k8s manifest: add the two new topic env vars.

### 7.6 `atlas-world`

- Verify a REST DELETE (or equivalent) route exists for `channel.Processor.Unregister`. If missing, add it. No business-logic changes; existing `Unregister` is already correct.

## 8. Non-Functional Requirements

### 8.1 Performance

- **Boot time.** atlas-channel/atlas-login caught-up time should be comparable to today's REST-pull-once latency for tenant counts ≤ 20 (tens-to-hundreds of ms). End-offset query + consume + apply N records on a single-partition compacted topic is dominated by N round trips, similar shape to today's serial REST calls.
- **Drain time.** Per-listener drain completes within 5s in the common case (no stuck sessions). Hard ceiling 10s. Pod shutdown drains all listeners concurrently within k8s `terminationGracePeriodSeconds` (15-20s recommended).
- **Outbox drainer latency.** Publish latency from `Enqueue` commit to Kafka publish ≤ 100ms in the common case (NOTIFY-driven wake-up); ≤ 1s worst case (polling fallback).
- **Per-event projection cost.** Applying one config event must not exceed 10ms (single map insert + recompute desired listener set).

### 8.2 Security

- No new authentication surface — Kafka subscriber uses existing broker credentials.
- Outbox writes are gated behind atlas-configurations' existing REST authorization. No new public endpoints in atlas-configurations.

### 8.3 Observability

- atlas-configurations: structured logs on every outbox enqueue (DEBUG), drainer publish batch (INFO with row count), lock state transitions (INFO), publish failures (WARN with attempt count + last_error).
- atlas-channel: structured logs on caught-up transition (INFO with elapsed boot time), each config event applied (DEBUG), listener Add/Drain phase transitions (INFO with `(t,w,c)` and phase), Drain timeout exceeded (WARN with session count remaining), Evict invocations (INFO with tenant).
- atlas-channel `/debug/consumers` already exists (`libs/atlas-kafka/consumer/debug.go`); add a parallel `/debug/listeners` returning `listener.Registry.Snapshot()` for live diagnostics.
- Trace: span context propagates through outbox (captured at Enqueue, restored on publish). Consumer-side, the existing `SpanHeaderParser` decorator wires it back into ctx.
- Metrics (if `atlas-metrics` or equivalent exists; otherwise log-only): outbox_unsent_count (gauge), outbox_publish_latency (histogram), listener_count (gauge), listener_drain_duration (histogram), caught_up_age_seconds (gauge).

### 8.4 Multi-tenancy

- The outbox table is operational, not tenant-scoped. Standard tenant scoping rules in `libs/atlas-database/tenant_scope.go` must be opted out of for the outbox entity.
- Tenant topic events are keyed by tenant UUID. Each event's `config` payload contains a single tenant's data. There is no cross-tenant event.
- atlas-channel/login projection state is process-global, but every consumer of projection state derives a tenant via `tenant.MustFromContext` per the existing pattern. Projection lookups are by tenant UUID.

### 8.5 Reliability

- atlas-configurations multi-replica safe via `pg_advisory_lock` leadership in the drainer. Failover takes ≤ 2× the lock-poll interval (which defaults to the same value as the drainer's poll interval, 1s).
- Kafka unavailability at boot: atlas-channel/atlas-login remain not-ready until catch-up succeeds. No crash-loop; they retry via the standard `atlas-kafka` consumer reconnect path.
- Kafka unavailability mid-run: existing consumer recreate logic in `libs/atlas-kafka/consumer/manager.go:329-369` handles this. Projection state remains valid; readiness stays true.
- Compaction misconfiguration on broker: detected operationally, not in code. PRD calls out the requirement; topic provisioning is out of scope (operator concern).

## 9. Open Questions

1. **atlas-world REST surface for Unregister.** Need to verify (early in design phase) whether the existing endpoint is reachable from atlas-channel. If missing, the small atlas-world add is in scope; if missing *and* the endpoint shape doesn't fit JSON:API conventions, design phase decides the route shape.
2. **Existing `server.Model.Is(t,w,c)` self-gating vs. handler-deregister.** Two valid drain mechanisms exist (extend `Is(...)` to consult registry state for free gating, or rely solely on `RemoveHandler`). Design phase picks one. Current PRD wording assumes deregister is the load-bearing mechanism and `Is(...)` extension is optional polish.
3. **Login projection: one consumer or two?** atlas-login may not need to subscribe to *both* topics (it cares about tenant config for protocol tables; whether the service topic is relevant depends on what login uses from it today). Design phase to confirm by reading login's current `configuration/registry.go` consumers.
4. **In-flight handler dispatch during drain.** When `RemoveHandler` is called, goroutines already mid-dispatch are not interrupted. Probably fine (errors swallowed on write to closed socket), but worth deciding whether the drain primitive needs to wait for `processMessage` goroutines to finish via the existing manager — the `mu.Lock` + copy pattern in `processMessage` should already give us this in practice. Plan-phase verification.

## 10. Acceptance Criteria

- [ ] `libs/atlas-outbox` exists with `Enqueue`, `Drainer`, `Migration`, `Backfill`, and tests covering: transactional enqueue, NOTIFY wake-up, advisory-lock failover, SKIP LOCKED multi-pod safety, idempotent backfill, sweeper retention.
- [ ] `libs/atlas-kafka` exposes `consumer.ReadEndOffsets` with a unit test against a stub or testcontainers Kafka.
- [ ] atlas-configurations migrates the `outbox_entries` table on startup and emits to both topics on every services/tenants CRUD. Verifiable by integration test or local docker-compose run + `kafka-console-consumer` watch.
- [ ] atlas-configurations seeder backfill produces correct events on a fresh broker without duplicates on a populated broker (idempotency test).
- [ ] atlas-channel boots with atlas-configurations unreachable and reaches ready once the config topic is caught up. /readyz returns not-ready before catch-up and ready after.
- [ ] atlas-channel does not require an atlas-configurations REST call at any point. (`requests.go` for configuration package can be removed.)
- [ ] Adding a tenant to atlas-configurations brings up a new listener on the running atlas-channel pod without restart. Sessions can connect to the new channel within seconds of the config commit.
- [ ] Removing a tenant from atlas-configurations triggers the four-phase drain on the running atlas-channel pod. Active sessions complete the save-and-kick path within 5s. After drain, the listener port is no longer accepting connects, `server.GetAll()` no longer contains the entry, atlas-world has unregistered the channel, and all per-`(t,w,c)` Kafka handlers are removed from the consumer manager.
- [ ] On SIGTERM, /readyz flips to not-ready immediately; all listeners drain in parallel; pod exits cleanly within k8s `terminationGracePeriodSeconds`.
- [ ] `session.Destroy` emit order: Kafka events are produced before socket close. Verified by reading the function and by a test that asserts ordering.
- [ ] Every per-tenant local-state singleton has an `Evict(t)` hook called when its tenant's last listener drains. Verified by test.
- [ ] atlas-login boots and reaches ready without atlas-configurations. Drains affected listeners on REMOVE events without save-and-kick (just close).
- [ ] `go test -race ./...` passes in every changed module.
- [ ] `go vet ./...` passes in every changed module.
- [ ] `docker build` succeeds for every changed service from the worktree root.
- [ ] Service READMEs updated for atlas-configurations, atlas-channel, atlas-login describing the new event-driven config flow.
- [ ] `libs/atlas-outbox` has a README documenting Enqueue/Drainer usage, idempotency expectations, and the at-least-once semantics consumers must respect.
