# Dynamic Service Configuration — Design Document

Version: v1
Status: Draft
Created: 2026-05-17
Companion to: `prd.md`

---

## 1. Scope of this document

The PRD locks down WHAT the system must do. This design locks down HOW, with the goal that the subsequent plan can be written without re-litigating architecture. It:

1. Picks one option for each PRD open question.
2. Decomposes the work into named packages with explicit dependencies.
3. Specifies the lifecycle of every shared singleton touched by the change.
4. Names the failure modes the implementation must tolerate, and the test surface that proves it.

It deliberately does not specify task ordering, files-per-task, or test-name granularity — those belong in the plan.

## 2. Architectural overview

```
┌──────────────────────┐                ┌──────────────────────┐
│ atlas-configurations │                │       Kafka          │
│                      │                │                      │
│ REST Create/Update/  │   tx commit    │ CONFIG_SERVICE_STATUS│
│ Delete               ├───────────────►│  (compacted, 1 part) │
│                      │                │                      │
│ outbox.Enqueue       │                │ CONFIG_TENANT_STATUS │
│       │              │                │  (compacted, 1 part) │
│       ▼              │                │                      │
│ outbox_entries table │  pg_notify     └──────┬────────┬──────┘
│       ▲              │                       │        │
│       │              │  publish              │        │
│ outbox.Drainer ──────┼───────────────────────┘        │
│ (pg_advisory_lock)   │                                │
└──────────────────────┘                                │
                                                        ▼
                                              ┌───────────────────┐
                                              │ atlas-channel /   │
                                              │ atlas-login       │
                                              │                   │
                                              │ configuration/    │
                                              │ projection        │
                                              │   ├ end-offset    │
                                              │   │   snapshot    │
                                              │   ├ caught-up gate│
                                              │   ├ serviceConfig │
                                              │   └ tenantConfigs │
                                              │       │           │
                                              │       ▼           │
                                              │ listener.Registry │
                                              │   ├ Add(key, cfg) │
                                              │   ├ Drain(key)    │
                                              │   └ snapshot()    │
                                              │       │           │
                                              │       ▼           │
                                              │ per-(t,w,c):      │
                                              │   ├ server.Model  │
                                              │   ├ socket goro   │
                                              │   ├ kafka handler │
                                              │   │   IDs []      │
                                              │   └ session WG    │
                                              └───────────────────┘
```

Three concerns are isolated:

- **Outbox** (atlas-configurations side, plus a reusable lib): atomic DB+Kafka write.
- **Projection** (channel/login side): build live config state from the stream and gate readiness.
- **Lifecycle** (channel/login side): bind that state to socket listeners, drain on REMOVE.

These concerns are wired together but each is independently testable.

## 3. Open question resolutions

The PRD's §9 lists four open questions. Each is decided here.

### 3.1 atlas-world REST surface for Unregister (PRD Q1)

**Decision: add a new DELETE route to atlas-world.** Read of `services/atlas-world/atlas.com/world/channel/resource.go:22-24` confirms only POST (register) and GET (list/by-id) exist; the existing `processor.Unregister` is reached only by `task.go:41` (expiration sweep). atlas-channel cannot currently call Unregister via REST.

Route shape:

```
DELETE /api/world-server/channel-server/{worldId}/{channelId}
```

Tenant context propagates via the existing tenant header parser. Handler maps to `channel.Processor.Unregister(channel.NewModel(world.Id(wId), channel.Id(cId)))`. Body none; 204 on success, 404 if the channel was not registered for this tenant (idempotent semantically — atlas-channel treats 404 as success on drain).

This is a tightly-scoped addition: resource registration, handler function, and the existing processor method untouched. No business-logic change.

### 3.2 Drain gating mechanism: handler-deregister vs `Is(...)` extension (PRD Q2)

**Decision: handler-deregister is the load-bearing mechanism.** `server.Model.Is(t,w,c)` is not extended to consult registry state.

Reasoning:
- The `Is(...)` extension would couple every consumer (40+ packages) to a new lifecycle dependency just to skip work it shouldn't have received. Whereas `RemoveHandler` already exists (`libs/atlas-kafka/consumer/manager.go:173`) and operates at the only correct chokepoint — the message-dispatch loop.
- An `Is(...)` extension would introduce a window where the consumer goroutine is mid-iteration and `state == Draining` but `Is(...)` still returns true (state was set after the read). Handler-deregister has no analogous window because the dispatch loop's `mu.Lock()` + `handlers` copy guarantees that once the entry is removed, no further dispatches occur.
- Single source of truth is cheaper to reason about. `server.Model.Is(...)` keeps its current pure-data semantics ("does this scope match these coordinates"); drain state lives on `listener.Handle`.

`server.Model.Is(...)` stays unchanged.

### 3.3 atlas-login subscribes to both topics (PRD Q3)

**Decision: atlas-login subscribes to both `CONFIG_SERVICE_STATUS` and `CONFIG_TENANT_STATUS`.** atlas-login uses `tenant.Socket` for protocol-table handler/writer wiring per tenant (analogous to atlas-channel), so it needs the full tenant projection. It uses service-config for `IPAddress` and per-`(t,w,c)` listener port lists.

This keeps the projection-package shape symmetric with atlas-channel: same code, same tests, fewer special cases.

### 3.4 In-flight handler dispatch during drain (PRD Q4)

**Decision: drain does not wait for in-flight dispatches.** The existing `Consumer.processMessage` path (`libs/atlas-kafka/consumer/manager.go`, around `mu.Lock()` + handlers copy) already provides the right semantics: once `RemoveHandler` deletes the entry from `consumer.handlers`, no fresh dispatch starts; in-flight dispatches finish on goroutines that have already captured their handler. Those goroutines may write to a now-closed socket — that's already-tolerated behavior in the existing producer write path (`session.Model.Disconnect` makes writes non-fatal).

The plan should add one test that confirms: an in-flight handler that completes after `Drain` returns produces no panic and no further side-effects on the drained scope.

## 4. Package decomposition

### 4.1 `libs/atlas-outbox` (new library)

```
libs/atlas-outbox/
  go.mod, go.sum
  README.md
  outbox.go         -- Message, Enqueue, Migration helpers, public surface
  entity.go         -- gorm Entity struct + table name (config table)
  drainer.go        -- NewDrainer, options, Run/Stop, lock + publish loop, sweeper
  backfill.go       -- Backfill(...)
  notify.go         -- pg LISTEN connection management + dispatch
  outbox_test.go    -- Enqueue inside tx, tombstone, header default
  drainer_test.go   -- testcontainers-postgres + bufconn kafka mock: NOTIFY wakeup,
                       advisory-lock failover, SKIP LOCKED multi-pod, sweeper
  backfill_test.go  -- idempotent backfill against populated + empty topic
```

**Public API** is exactly as PRD §5.1, with two clarifications:

- `Drainer.Run(ctx)` blocks until ctx is canceled. Designed to be invoked under a teardown manager `WaitGroup`.
- `Backfill(...)` returns the count of rows enqueued. Caller (seeder) logs this.

**Tenant-scope opt-out.** Library queries `outbox_entries` using `database.WithoutTenantFilter(ctx)` (already exists at `libs/atlas-database/tenant_scope.go:19`). No need to disable callbacks per-entity; the public wrapper around every internal DB call sets the bypass context. Adopters that build their own DB session for outbox queries must use the same wrapper.

**Lock strategy.** `pg_try_advisory_lock(hashtext('atlas_outbox_drainer'))`. Non-blocking try. If true: this replica is the leader and runs the publish loop. If false: this replica idles, retrying every `pollInterval`. On loss (connection drop, replica restart), `pg_advisory_unlock` is implicit on session end; another replica takes over within one poll tick.

**NOTIFY semantics.** `Enqueue` runs `NOTIFY atlas_outbox_new, '<topic>'` inside the caller's transaction (the NOTIFY only fires on commit, which is what we want). Drainer's listener goroutine uses a dedicated `pq.Listener` connection (or equivalent under the pgx driver), wakes the publish loop on any notification. If LISTEN connection drops, the poll interval is the floor — correctness is preserved, latency degrades.

**Retention.** The sweeper deletes `WHERE sent_at < NOW() - retention`. Default 7d. This is operational state; the audit trail lives in Kafka.

**Crash semantics.** At-least-once. The PRD is explicit; the library README enumerates the only durable invariants: (a) every committed row is eventually published; (b) consumers must be idempotent.

**Schema-version field in envelope.** Drainer does not inspect; it ships bytes. The library has no opinion on payload structure. Adopters serialize before calling `Enqueue`.

### 4.2 `libs/atlas-kafka` additions

One file, one function:

```
libs/atlas-kafka/consumer/offsets.go
  func ReadEndOffsets(ctx context.Context, brokers []string, topic string) (map[int]int64, error)
```

Uses `kafka.Conn.ReadPartitions(topic)` then `Conn.ReadOffsets()` per partition for the last offset. Returns a map keyed by partition id. Test uses the same testcontainers/bufconn pattern as existing `consumer/manager_test.go`.

### 4.3 atlas-configurations

```
services/atlas-configurations/atlas.com/configurations/
  main.go                            -- + outbox.Migration in DB init
                                     -- + outbox.NewDrainer + go drainer.Run(ctx)
                                     -- + teardown registration
  outbox/
    envelopes.go                     -- service envelope, tenant envelope (json)
  services/processor.go              -- Enqueue inside ExecuteTransaction callback
  tenants/processor.go               -- same
  seeder/seeder.go                   -- + outbox.Backfill after seed-from-json
```

New `outbox` package inside the service is the marshalling boundary — keeps envelope shape out of the `services` and `tenants` packages.

**Envelope marshalling.** `outbox.NewServiceEnvelope(id, rm RestModel) ([]byte, error)` builds:

```json
{"schema_version": 1, "id": "<uuid>", "config": {...}, "emitted_at": "<rfc3339>"}
```

Tombstone path skips marshalling: caller passes `nil` value to `outbox.Enqueue`.

**Topic env vars.** Added to k8s manifest and Dockerfile. The values match the env var names — no aliasing.

**Transaction shape.** Each of the three Create/Update/Delete callbacks already runs inside `database.ExecuteTransaction` (read at `services/atlas-configurations/atlas.com/configurations/services/processor.go:130,154,158`). The Enqueue call is appended to that callback after the existing entity operation. If Enqueue errors, the callback returns the error and the transaction aborts — preserving "DB+outbox commit together or neither."

**Backfill on startup.** The seeder runs after migrations. Backfill is the LAST step (after seed-from-json). It is idempotent in the operational sense: it queries `outbox_entries` for existing rows per topic+key; if any exist for a key, it skips. This is correct for steady-state but degrades after the sweeper deletes old rows. To survive that, backfill additionally checks the topic's high-water-mark via `consumer.ReadEndOffsets`: if any partition has offset > 0, backfill is a no-op (the topic is the source of truth; if rows aren't there it's an operator-recovery scenario, not a startup concern). This handles the "fresh cluster on day 1" case while staying safe on every restart.

### 4.4 atlas-channel

```
services/atlas-channel/atlas.com/channel/
  configuration/
    projection/
      subscriber.go     -- Kafka consumers, per-topic apply
      state.go          -- Singleton: tenantConfigs map, serviceConfig, mu, listeners channels
      caughtup.go       -- End-offset snapshot + watermark tracking + ready gate
      apply.go          -- Diff: previous state → new state → desired listener set;
                          -- emits Add(key, cfg) / Drain(key) / no-op ops on a chan
    envelope.go         -- Decode envelopes from kafka, tombstone detection
    registry.go         -- DELETE (replaced)
  server/
    registry.go         -- []Model → map[Key]Model, add Deregister(key), Get(key)
    model.go            -- unchanged (decision §3.2)
  listener/
    handle.go           -- Handle struct, State enum, HandlerHandle type
    registry.go         -- Registry: Add, Drain (four-phase), Snapshot, evictor hooks
    evict.go            -- RegisterEvictor(func(t tenant.Model)), tenant ref-count
  session/processor.go  -- reorder Destroy (FR-CHN-14)
  channel/
    processor.go        -- + Unregister(ch channel.Model) error
    requests.go         -- + DELETE request builder
  kafka/consumer/**/consumer.go -- InitHandlers signature change (40+ files)
  main.go               -- replace block, see §4.4.5
```

#### 4.4.1 `configuration/projection`

Single goroutine per topic owns the projection mutations for that topic. State has one `sync.RWMutex`. Readers (listener-add lookups, debug snapshot) take RLock. Each apply takes the write lock for a short window.

Service-config filter: applied at decode time. The service topic carries every channel-service's config; the projection drops every message whose envelope `id != SERVICE_ID env`. (Login does the same with its own `SERVICE_ID`.)

Apply step yields a snapshot of `(serviceConfig, tenantConfigs)` and computes the desired listener set:

```
desired = {}
for each tenant t in serviceConfig.Tenants:
  if t.id ∉ tenantConfigs: skip   # tenant referenced but its config not yet seen
  for each world w in t.Worlds (channel-service variant):
    for each channel c in w.Channels:
      desired[(t.id, w.id, c.id)] = listenerConfig{
        ipAddress: t.IPAddress,
        port:      c.Port,
        region:    tenantConfigs[t.id].Region,
        version:   tenantConfigs[t.id].MajorVersion + Minor,
        socket:    tenantConfigs[t.id].Socket,
      }
```

Diff is `current` (listener.Registry.Snapshot keys) vs `desired`:
- key in desired but not current → `Add(key, cfg)`
- key in current but not desired → `Drain(key)`
- key in both, cfg unchanged → no-op
- key in both, cfg changed (port, region/version, socket tables) → `Drain(key)` then `Add(key, newCfg)`

Apply enqueues ops on a buffered ops channel (bounded by listener-count cardinality). A single consumer goroutine drains this channel, ensuring Add/Drain run serially per key. This avoids needing per-key locks in the listener registry.

#### 4.4.2 `configuration/projection/caughtup`

At boot:
1. Snapshot `endOffsets = ReadEndOffsets(...)` for each topic, before subscribing.
2. Subscribe at earliest offset (per-message offsets are visible on `kafka.Message.Offset` returned by the existing `atlas-kafka` consumer plumbing).
3. Maintain `currentOffsets map[partition]int64` per topic.
4. `caughtUp()` returns true iff `currentOffsets[p] >= endOffsets[p]` for every partition on every topic.
5. Once caughtUp returns true, set an `atomic.Bool` to true. Subsequent reads check the bool; no further computation. (One-way per PRD FR-CHN-5.)

Edge case: empty topic. `ReadEndOffsets` returns 0 for the partition. caughtUp evaluates true on first poll. Listener set is empty until apply runs — this is correct: the pod is "ready" but holds no listeners. /readyz returns 200; k8s routes nothing because no Service backend selects this pod for the (nonexistent) tenant. As soon as the first tenant CRUD happens, apply brings up listeners.

Edge case: end-offset query fails. Block startup with a retry loop, NOT a fatal. Exponential backoff to a configurable ceiling (default 30s). The PRD's reliability goal (boot survives atlas-configurations outage) extends naturally: boot also survives transient Kafka metadata errors.

#### 4.4.3 `server.Registry` shape change

Today:
```go
type Registry struct {
    lock sync.RWMutex
    registry []Model
}
```

New:
```go
type Key struct {
    TenantId  uuid.UUID
    WorldId   world.Id
    ChannelId channel.Id
}
type Registry struct {
    lock sync.RWMutex
    entries map[Key]Model
}

func (r *Registry) Register(m Model)              // map upsert
func (r *Registry) Deregister(key Key)            // map delete
func (r *Registry) Get(key Key) (Model, bool)
func (r *Registry) GetAll() []Model               // iteration order undefined (acceptable per existing semantics)
```

Heartbeat (`channel/task.go:36`) iterates `GetAll()`. After `Deregister(key)`, the drained scope simply stops appearing — no further change needed there.

#### 4.4.4 `listener.Registry` and the four-phase drain

`listener.Handle`:

```go
type State int
const (Active State = iota; Draining; Removed)

type HandlerHandle struct { Topic, Id string }

type Handle struct {
    Key            server.Key
    State          State
    Ctx            context.Context
    Cancel         context.CancelFunc
    Wg             *sync.WaitGroup           // counts active sessions for this listener
    ServerModel    server.Model
    KafkaHandlers  []HandlerHandle
}
```

Per-listener `wg` is incremented when a session attaches to this listener (in the socket Accept loop) and decremented in the session destroy path. The plan must wire this into the socket package; it's the only way drain can wait correctly.

`Drain(key)` phases:

1. **Quiesce** — under registry lock: load handle; if absent or `Removed`, return nil (idempotent). Set `state = Draining`. Release lock. Outside lock:
   - `server.Registry.Deregister(key)` — heartbeat stops re-registering.
   - `channel.NewProcessor(l, tctx).Unregister(handle.ServerModel.Channel())` — atlas-world drops the entry. 404 from atlas-world is treated as success (already drained). Other errors are WARN-logged but do not abort drain.

2. **Save-and-kick** — `session.Registry.Snapshot(filter=key)`. For each session: send a server-shutdown status packet, then `session.Processor.Destroy(s)` (whose body is reordered per FR-CHN-14). Destroy decrements the per-listener wg.

3. **Drain deadline**:
   ```go
   done := make(chan struct{})
   go func() { handle.Wg.Wait(); close(done) }()
   select {
   case <-done:           // clean
   case <-time.After(deadline):  // bounded
       l.WithField("key", key).Warn("drain deadline exceeded; <N> sessions still open")
   }
   ```
   Deadline defaults to 5s. Ceiling 10s. Configurable via `DRAIN_DEADLINE_MS` (clamped to ceiling). On timeout, proceed to phase 4 anyway — leaked sessions will write to a closed socket and exit on their next IO. They will not produce their destroy events (best effort).

4. **Tear down**:
   - `handle.Cancel()` — stops `socket.Run` in the listener goroutine.
   - For each `HandlerHandle` in `handle.KafkaHandlers`: `consumer.GetManager().RemoveHandler(topic, id)`. Errors logged but not fatal (idempotent).
   - Set `state = Removed`.
   - Release the per-tenant ref count (see §4.4.6).

Drain is idempotent and safe to call concurrently from multiple goroutines — only the first transitions out of `Active`; subsequent callers see `Draining`/`Removed` and return.

#### 4.4.5 main.go restructuring

The current 209-380 block becomes:

```go
projection := projection.New(l, tdm.Context(), tdm.WaitGroup(), serviceId,
    /* lookup callbacks: writerList, validatorMap, handlerMap, consumerGroupId, ... */)

if err := projection.Start(); err != nil {
    l.WithError(err).Fatal("Unable to start configuration projection.")
}

// blocks until caught up or context canceled
if err := projection.WaitCaughtUp(); err != nil {
    l.WithError(err).Fatal("Configuration projection did not catch up.")
}

go projection.ApplyLoop()  // drives listener.Registry from this point

// REST server with /readyz wired to projection.ReadyChecker()
restserver.New(l).
    ...
    WithReadyChecker(projection.ReadyChecker()).
    Run()
```

The per-`(t,w,c)` block (account-registry init, `server.Register`, all `InitHandlers` calls, `CreateSocketService`) moves into a `listener.Registry.Add(key, cfg)` implementation. The implementation captures returned `[]HandlerHandle` from each `InitHandlers` call into the new `Handle.KafkaHandlers`.

The 40+ `InitHandlers` invocations remain explicit calls in `listener.Registry.Add` — there's no clean reflection or registration-table cleanup in scope, and centralizing makes the per-(t,w,c) startup pathway visible in one place.

#### 4.4.6 Per-tenant local-state eviction

Three singletons today key state by tenant: `monster.StatusMirror`, `monster.NextSkillInbox`, `account.Registry`. Each gains an `Evict(t tenant.Model)` method that deletes that tenant's entries.

`listener.Registry` maintains an internal `map[tenant.Id]int` ref-count. `Add(key, cfg)` increments; `Drain(key)` final phase decrements. When count drops to 0:

1. For each registered evictor function (registered at startup), call `evictor(t)`.
2. Call `tenant.Unregister(t.Id())` (new method on the global registry).

Evictor registration is process-global and one-shot, set in `main.go` after the singletons are constructed. Example wiring:

```go
listener.RegisterEvictor(func(t tenant.Model) {
    monster.GetStatusMirror().Evict(t.Id())
    monster.GetNextSkillInbox().Evict(t.Id())
    account.GetRegistry().Evict(t.Id())
})
```

`account.NewProcessor(...).InitializeRegistry()` (currently `main.go:223`) moves into `listener.Registry.Add` so the registry is created on first listener for that tenant. The Evict hook then has well-defined teardown counterpart.

#### 4.4.7 Session destroy reordering (FR-CHN-14)

Current order at `session/processor.go:330-336`:
```
Remove from registry → Disconnect (close socket) → emit logout cmd → emit destroy event
```

New order:
```
Remove from registry → emit logout cmd → emit destroy event → Disconnect (close socket)
```

Registry-remove stays first to prevent double-destroy from a concurrent path. The two emit calls move before Disconnect: this way a crash after socket close (mid-batch in the producer) still publishes the events on the next process's batch flush — except the next process doesn't exist for this session, so the events are lost. By emitting first, we shift the failure window earlier in the function, before the producer's `BatchTimeout` could clear: the event has been written to the producer's local buffer before we close the socket, and the producer's existing flush-on-shutdown logic catches it.

This change carries a behavioral risk: any downstream code that assumed the socket was already closed when the destroy event was observed must be checked. The PRD calls this out as acceptable; the plan should include a focused integration test that asserts ordering.

### 4.5 atlas-login

Same packages as atlas-channel for `configuration/projection`, simpler `listener.Registry`:

```
Drain(key):
  1. server.Registry.Deregister(key)   -- (login may not have an analogous registry; design phase notes none exists today)
  2. send status packet on existing sessions
  3. handle.Cancel() to stop accept loop
  4. wait for handle.Wg (short — login sessions are stateless after handshake)
  5. RemoveHandler for each captured handle
  6. state = Removed
```

No save-and-kick; no per-tenant evict (login does not hold the same per-tenant singletons). The drain deadline is shorter (recommend 2s default, 5s ceiling) given login sessions don't persist state.

### 4.6 atlas-world

One new file's worth of change:

```
services/atlas-world/atlas.com/world/channel/resource.go
  + r.HandleFunc("/{channelId}", handleUnregisterChannelServer).Methods(http.MethodDelete)
  + handleUnregisterChannelServer(...) (calls existing channel.Processor.Unregister)
```

No business logic changes. Existing `Unregister` and tests cover semantics.

## 5. Lifecycle scenarios

### 5.1 Boot (cold cluster, no Kafka rows yet)

1. atlas-configurations boots → migration creates `outbox_entries` → seeder runs → seed-from-json + `outbox.Backfill` (topic empty, backfill enqueues all rows) → drainer publishes → both topics populated.
2. atlas-channel boots → projection queries `ReadEndOffsets` (returns offsets from step 1) → subscribes earliest → consumes to those offsets → caughtUp = true → /readyz=200 → applyLoop starts → listener.Add for each desired key → sockets bind → heartbeat starts → atlas-world receives Register → players connect.

### 5.2 Boot (atlas-configurations unreachable)

atlas-channel does not call atlas-configurations REST at any point — there is no degradation path to handle. Kafka being unreachable is the analogous failure: projection blocks at `WaitCaughtUp`, /readyz=503, pod is not ready, k8s holds traffic. When Kafka returns, projection catches up and /readyz=200.

### 5.3 Tenant add (running pod)

1. Operator POSTs tenant + service config → atlas-configurations transaction commits → outbox row inserted → NOTIFY fires → drainer publishes message → topic receives event.
2. atlas-channel projection consumes → applies to state → diff finds new key → enqueues Add op → applyLoop calls `listener.Registry.Add(key, cfg)` → server.Register, InitHandlers per package (40+ calls), CreateSocketService → socket listens → heartbeat picks it up → atlas-world Registers → atlas-login sees it in its world query → players can connect.

### 5.4 Tenant remove (running pod)

1. Operator DELETEs tenant → atlas-configurations transaction commits tombstone outbox row → drainer publishes Kafka message with null value.
2. atlas-channel projection consumes → tombstone removes key from `tenantConfigs` → diff finds key in current but not desired → enqueues Drain op → applyLoop calls `Drain(key)`.
3. Phase 1: server.Deregister, atlas-world DELETE → atlas-login stops advertising channel within seconds (its world query loop).
4. Phase 2: every session for this (t,w,c) gets status packet + Destroy (save-and-kick).
5. Phase 3: wait up to 5s for wg to drain.
6. Phase 4: cancel ctx → socket.Run exits → RemoveHandler per captured handle → state=Removed → ref-count decremented → if last for this tenant, evictors fire and `tenant.Unregister`.

### 5.5 Pod SIGTERM

1. SIGTERM → /readyz immediately flips to 503 (independent of the projection state).
2. Pre-drain delay (default 5s) lets the LB stop sending new traffic.
3. teardown manager invokes `listener.Registry.DrainAll()`, parallelizing across active listeners.
4. Each listener follows the four-phase drain. Bounded by `terminationGracePeriodSeconds` (recommend 20s).
5. atlas-outbox drainer in atlas-configurations also shuts down via its own teardown registration: stop accepting new NOTIFYs, complete the current batch, exit.

### 5.6 atlas-configurations replica failover

Two replicas, both run a Drainer. Only one holds `pg_try_advisory_lock` at a time. If the leader crashes, its Postgres session ends, the lock releases, the follower acquires on the next poll tick (≤1s), and resumes publishing from the same SKIP-LOCKED queue. No coordination needed beyond the lock.

## 6. Alternatives considered and rejected

### 6.1 Use Debezium / kafka-connect for outbox publishing

Considered: standard transactional-outbox pattern in the JVM/big-data world. Rejected for this codebase: introduces an operational dependency (kafka-connect deployment) we don't run today; the project has explicit appetite for an in-process Go drainer (PRD FR-OUT-3 specifies advisory-lock-coordinated drainer); the at-least-once semantics work identically. Future replatform is not blocked — the outbox table shape matches what Debezium expects.

### 6.2 Skip the outbox; emit Kafka inside the transaction

Considered: just call producer.Send during the transaction callback. Rejected: even setting aside the "DB committed but Kafka failed" failure mode, the producer batches asynchronously — a "successful" Send returns before the broker ack. Outbox makes this atomic in a way the existing producer manager cannot.

### 6.3 Use ZooKeeper / etcd for live config

Considered: well-trodden path. Rejected: redundant with Kafka (the project already runs Kafka heavily; ZK is a strict dependency add); compaction gives us the audit-trail + replay properties of etcd watch with no extra dep; the existing `atlas-kafka` library already handles reconnect/retry.

### 6.4 Polling-only drainer (no LISTEN/NOTIFY)

Considered: simpler. Rejected for the latency target (PRD §8.1 ≤100ms p50). Polling-only would force 100ms-resolution sleeps with 100ms-tick wakeup overhead; NOTIFY adds ~30 lines of code and a connection. The poll fallback stays as the correctness floor.

### 6.5 Reflection-based or generated handler-ID threading

Considered: codegen the 40+ `InitHandlers` signature changes. Rejected: the change is mechanical but small (return slice instead of nothing). The cost of a generator is higher than the one-time refactor; the generator would need to live in the repo forever. Manual change is auditable in one PR.

### 6.6 Make Drain wait for in-flight handler dispatches

Considered: extend `Consumer` to expose a per-handler-id wg; Drain phase 4 waits on it. Rejected: cost is high (new mutex contention on the dispatch hot path) for a benefit that's already covered by the existing copy-on-dispatch pattern. The plan can add a test to confirm; if a regression surfaces post-rollout, this is the obvious follow-up.

### 6.7 Extend `server.Model.Is(...)` to gate on drain state

Already covered in §3.2.

## 7. Failure modes and observability

| Failure | Detection | Action |
|---|---|---|
| Outbox drainer crash on leader | follower acquires lock within ≤1s | structured log: "outbox.lock_lost" on leader, "outbox.lock_acquired" on follower |
| Kafka unreachable from drainer | publish errors increment `attempts` on row | WARN log; row retried next tick; `outbox_unsent_count` metric increases |
| Kafka unreachable from atlas-channel projection | existing consumer reconnect loop handles | /readyz remains 200 once caughtUp (per FR-CHN-5); apply paused; new events queued |
| Drain deadline exceeded | wg.Wait timed out | WARN log with remaining session count + key; phase 4 proceeds anyway |
| atlas-world DELETE returns non-2xx/non-404 | error logged | drain continues (atlas-world's expiration task will reconcile within its task interval) |
| Tenant referenced in serviceConfig but missing from tenantConfigs | apply step skips key | DEBUG log; key picked up on next tenant event (caught-up gate held listener offline until tenant arrives if both arrive in same boot batch) |
| Listener.Add fails partway through (e.g., port bind error) | error returned | log + don't transition state to Active; cleanup partially-registered handlers; do not block applyLoop |
| Crash between session emit-destroy and disconnect | producer batch flushes on shutdown via existing producer manager teardown | logs already exist; new test asserts ordering |

Observability surface, by service:

- **atlas-configurations**: structured log `outbox.enqueued` (DEBUG, per row), `outbox.batch_published` (INFO, per batch with count), `outbox.lock_*` (INFO), `outbox.publish_failed` (WARN, attempts + last_error), `outbox.sweeper_run` (INFO, deleted count).
- **atlas-channel/login**: `projection.caughtup` (INFO, elapsed ms), `projection.applied` (DEBUG, key + op), `listener.added` (INFO, key), `listener.drain_phase` (INFO, key + phase 1..4), `listener.drain_timeout` (WARN, key + remaining), `tenant.evicted` (INFO, tenant id). `/debug/listeners` exposes `listener.Registry.Snapshot()` (parallel to existing `/debug/consumers`).
- Metrics gauges/histograms only if `atlas-metrics` exists today; otherwise log-only. Names match PRD §8.3.

## 8. Test surface

The plan should provide:

**libs/atlas-outbox**:
- Unit: Enqueue inside a transaction commits the row and triggers NOTIFY.
- Unit: tombstone (nil value) round-trips correctly.
- Integration (testcontainers postgres): drainer-led replica publishes, idle replica polls; on leader-stop, idle takes over within one tick.
- Integration: SKIP LOCKED guarantees two drainers never publish the same row.
- Integration: backfill is idempotent (run twice, second run enqueues 0 rows).
- Integration: sweeper deletes rows older than retention.
- Property: NOTIFY wake-up reduces publish latency vs poll-only.

**libs/atlas-kafka**:
- Unit: `ReadEndOffsets` returns correct offsets on a stub broker.

**atlas-configurations**:
- Integration: Create/Update/Delete each enqueue exactly one outbox row; running the drainer publishes that row; integration consumer sees the envelope.
- Integration: Seeder backfill on fresh broker enqueues N rows where N is the seeded row count; on a re-run, enqueues 0.

**atlas-channel**:
- Integration: with Kafka stub serving only the tenant topic populated, /readyz returns 503 before catch-up and 200 after.
- Unit: projection apply diff produces expected (Add, Drain, no-op) ops for representative state transitions.
- Integration: ADD event brings up a listener and atlas-world Register is called.
- Integration: REMOVE event triggers four-phase drain; atlas-world Unregister called; all KafkaHandlers removed; session count drops to 0; state transitions to Removed.
- Integration: drain deadline exceeded path doesn't deadlock.
- Unit: session.Destroy reordering — emit calls precede Disconnect.
- Unit: last-listener-for-tenant transitions trigger every registered evictor exactly once; tenant.Unregister called.
- Concurrency: two Drain calls for the same key are race-free.

**atlas-login**:
- Integration: same caught-up gate as channel.
- Integration: REMOVE triggers stop-accept + close; no save-and-kick.

**atlas-world**:
- Unit: new DELETE handler invokes existing Unregister; 404 on missing channel.

## 9. Risks and mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Refactoring 40+ InitHandlers signatures introduces a regression in one package | medium | medium | Plan as a single mechanical-change task; all packages share a search-replace; CI build + go vet + tests for atlas-channel catch any drift. |
| Drain phase 4 races with in-flight handler that holds a captured `sc server.Model` | low | low | RemoveHandler prevents new dispatch; in-flight goroutines complete with stale data; sockets are closed by the time they finish (writes drop quietly). Add explicit test. |
| Outbox advisory-lock contention under high CRUD volume | very low | low | atlas-configurations CRUD volume is operator-initiated (rare). The lock is held by one drainer over multiple rows. No contention surface. |
| Compacted topic loses an old key's history → projection on a *new* subscriber doesn't see deletion | n/a | n/a | Compaction preserves the latest value (or tombstone) per key forever. By definition, the projection always sees the latest state. |
| atlas-channel pod sees an UPDATE before the ADD (out-of-order on a single partition is impossible by Kafka semantics; only relevant if topic is repartitioned later) | low | low | Single partition per topic per PRD §5.3 — ordering is guaranteed. If partitions change later, projection must add ordering by `emitted_at` — not in scope. |
| Long drain holds k8s rolling deploy | medium | medium | terminationGracePeriodSeconds set to 20s (PRD §8.1). Drain deadline 5s ensures completion well inside. |
| Outbox table grows unboundedly if drainer is stalled | low | medium | sweeper only deletes published rows. Stalled drainer is alerted via `outbox_unsent_count` (or log scraping). Operator follows up. |
| Backfill on a partially-populated cluster (some rows seeded earlier, sweeper ran, now backfill re-publishes) | low | low | Backfill checks topic end-offsets; if topic has data, backfill is no-op (§4.3). Operators wanting forced re-publish do it explicitly. |
| Session destroy reorder reveals a downstream-consumer assumption about socket-closed-before-destroy-event | low | medium | Code-review the destroy event consumers in saga, character, account; integration test that asserts ordering. |

## 10. Out of scope (carry-forward to future tasks)

- saga-orchestrator adoption of `libs/atlas-outbox`.
- Multi-partition compacted topics (would require additive ordering field; not now).
- Schema-registry / Avro / protobuf for envelopes.
- atlas-login per-tenant graceful drain coordination (login sessions are stateless).
- Saga cancellation on session destroy.
- Multi-owner channels / HA channel migration.
- templates/* on event bus (stays REST).
