# Kafka Consumer Self-Healing + Visibility — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-20
---

## 1. Overview

Every atlas-ms service runs Kafka consumers through the shared `libs/atlas-kafka/consumer` manager. Today those consumers can **stop fetching messages mid-run without surfacing any error**, leaving the process alive but effectively deaf on one or more topics. The concrete incident that motivated this task: atlas-channel was subscribed to `EVENT_TOPIC_ASSET_STATUS` at startup (startup log line present), quest-reward events were successfully produced by atlas-inventory and consumed by atlas-quest in the cluster, but the locally-run atlas-channel delivered zero asset-status messages to its handlers. The user-visible symptom was an inventory desync: quest-started items and dropped items never produced `InventoryChange` packets. The process was healthy; only the asset consumer was dead. Restart resolved it. No error log.

The root cause is a silent exit in the fetcher goroutine in `libs/atlas-kafka/consumer/manager.go` `Consumer.start`. There are three paths out of the loop that look like clean shutdown but aren't: (a) `FetchMessage` returning `io.EOF` on a transient broker disconnect is indistinguishable from the reader being intentionally closed (manager.go:179, 190–192); (b) after ~10 retries of a `FetchMessage` error the loop returns with only an `Errorf` log (manager.go:188, 193–196); (c) the outer `start` goroutine blocks on `<-ctx.Done()` with no mechanism to observe that the inner fetcher already exited (manager.go:205, 208), so the Manager continues reporting the consumer as registered long after it's dead. Handler-side panic recovery is fine (manager.go:261–269); the failure mode is strictly at the reader lifecycle.

This task makes the consumer loop self-healing by wrapping the reader lifecycle (not just the fetch call) in an outer recreate loop, re-derives shutdown from only the parent context's cancellation, and introduces a capped exponential backoff between recreations to avoid rebalance storms. It also adds per-consumer observable state (`lastFetchAt`, `lastErrorAt`, `lastError`, `recreateCount`, `aliveSince`) and a JSON:API debug route (`GET /api/debug/consumers`) mounted on each service's existing REST server so dead or churning consumers can be diagnosed by inspection. No Prometheus, no alerts, no staleness heuristics, no process crashes — explicit constraints from the design conversation.

## 2. Goals

Primary goals:
- A Kafka consumer whose underlying `kafka-go` reader dies (EOF, retry exhaustion, rebalance error, any non-ctx-cancel error) rebuilds the reader and rejoins the consumer group automatically, without crashing the process, without disconnecting in-flight game sessions, and without requiring operator action.
- Recreation is logged at `Info` level every time it occurs, so the log signal that was absent during the motivating incident is present in future incidents.
- Each `Consumer` tracks enough state for an operator to tell at a glance whether it is alive and healthy: `lastFetchAt` (time of last successful `FetchMessage`), `lastErrorAt` / `lastError` (most recent fetch failure), `recreateCount` (how many times the reader has been rebuilt), `aliveSince` (when the current reader was created).
- That state is exposed via a JSON:API route (`GET /api/debug/consumers`) on each service's existing REST server — one HTTP listener per service, not two.
- Every service in the monorepo that owns Kafka consumers surfaces the debug route. Services that do not currently have a REST server (8 of them, atlas-channel among them) gain a minimal REST server as part of this task.
- The change is source-compatible with the 49 services that call `consumer.GetManager()` today — no existing `main.go` loses behavior; services with existing REST servers gain one `AddRouteInitializer` call; HTTP-less services gain a minimal server scaffold.

Non-goals:
- No Prometheus metrics, Grafana dashboards, or OpenTelemetry traces for consumer health in this task.
- No alert rules, no staleness-based "silent consumer" detectors (explicitly rejected — there are legitimately idle topics in a test system with no user activity).
- No k8s liveness/readiness probes (explicitly rejected — a crash-loop restart would disconnect all connected game sessions and is disproportionate for a single-topic failure).
- No changes to `handler.Handler`, `safeHandle`, or the `cont=false` handler-deregistration path (already correct).
- No producer-side changes.
- No ingress exposure of the debug endpoint in this task. Debug access is via `kubectl port-forward` / `kubectl exec` / cluster-internal IP. Per-service ingress rules can be added later as a follow-up and are orthogonal.
- No dedicated debug HTTP port. The debug route lives on each service's existing REST server (existing `REST_PORT`).
- No feature flags or rollout toggles. The change is deployed to the shared library; every service rebuilds against it and picks up the new behavior.

## 3. User Stories

- As an Atlas developer reproducing a mid-session desync, I want my locally-run service's Kafka consumers to recover from a transient broker hiccup on their own, so that a 30-second WSL sleep or kafka-go reconnect doesn't silently break the client until I notice and bounce the process.
- As an operator debugging a service that "seems healthy but isn't processing messages on topic X," I want to `kubectl port-forward <pod> 8080:8080 && curl localhost:8080/api/debug/consumers` and immediately see which topic has a stale `lastFetchAt` or a non-zero `recreateCount`, so that I don't have to SSH into Kafka and run `kafka-consumer-groups --describe` to confirm the suspicion.
- As a developer reading logs after an incident, I want to see `Fetcher exited with <err>; recreating reader (attempt N)` log lines every time a consumer rebuilds, so that "the consumer silently died" is never the diagnosis again.
- As a developer shipping a new service, I want the self-healing behavior to be the default — I should not have to remember to opt in, and I should not be able to accidentally disable it.
- As a developer running the existing test suite, I want `libs/atlas-kafka/consumer/manager_test.go` to cover the new recreation path with a fake reader that returns EOF mid-run and assert that the fetcher recreates and continues.

## 4. Functional Requirements

### 4.1 Outer reader-lifecycle loop in `Consumer.start`

Rewrite `libs/atlas-kafka/consumer/manager.go` `Consumer.start` (line 162) so the reader is created and destroyed **inside** the goroutine, not before it. Shape:

```go
func (c *Consumer) start(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
    wg.Add(1)
    defer wg.Done()
    l.Infof("Creating topic consumer.")

    backoff := newBackoff()   // 500ms initial, 10s cap, reset on success
    for attempt := 0; ; attempt++ {
        if ctx.Err() != nil {
            l.Infof("Parent context canceled; shutting down topic consumer.")
            return
        }

        reader := c.newReader()              // calls the ReaderProducer with stored config
        c.setReader(reader, attempt)         // updates aliveSince, zeroes lastError, exposes reader for handler commits
        if attempt == 0 {
            l.Infof("Start consuming topic.")
        } else {
            l.Infof("Recreated reader for topic (attempt %d).", attempt)
        }

        err := c.runFetchLoop(l, ctx, reader)
        _ = reader.Close()

        if ctx.Err() != nil || errors.Is(err, context.Canceled) {
            l.Infof("Topic consumer stopped.")
            return
        }

        c.recordError(err)
        l.WithError(err).Errorf("Fetcher exited; recreating reader after backoff.")
        select {
        case <-ctx.Done():
            return
        case <-time.After(backoff.Next()):
        }
    }
}
```

`newReader()` takes the same `kafka.ReaderConfig` that's currently built inline in `AddConsumer` (manager.go:93–99). Move that config build into a stored field on `Consumer` so the goroutine can rebuild without re-entering `AddConsumer`.

`setReader` acquires `c.mu`, assigns `c.reader = reader`, updates `c.aliveSince = time.Now()`, increments `c.recreateCount` when `attempt > 0`, and clears `c.lastError`. `CommitMessages` (from `processMessage`) must read `c.reader` under the same mutex so a mid-commit reader swap is safe. Alternative acceptable: commit via the exact reader instance the fetcher used for that message, passed down through `runFetchLoop` — decide during implementation.

### 4.2 Inner fetch loop — short bounded retry, then fall through

Extract the current retry logic (manager.go:175–196) into `runFetchLoop(l, ctx, reader) error`. Keep the inner `retry.Try` but **shorten** it: 3 attempts, initial delay 100ms, max delay 500ms — roughly 1 second before giving up and returning to the outer recreate loop. A transient Kafka hiccup that recovers within a second stays inside the current reader; anything slower forces a reader rebuild, which in turn forces a fresh consumer-group rejoin.

Return conditions:
- `ctx.Err() != nil` → return `ctx.Err()` (outer loop treats as shutdown).
- Inner retry exhausts → return the last error (outer loop treats as recreate-eligible).
- `FetchMessage` returns `io.EOF` → return `io.EOF` (outer loop treats as recreate-eligible — see §4.3).
- Successful fetch → process via `c.processMessage`, commit, update `c.lastFetchAt`, continue the loop.

### 4.3 `io.EOF` is never "shutdown"

Remove the `err == io.EOF || errors.Is(err, context.Canceled) → log "Reader closed, shutdown." and return` special case from the current code (manager.go:179, 190–192). The new rule is simple and uniform: **only a canceled parent `ctx` means shutdown.** Any other error (including EOF) falls through to the outer recreate loop. This is the single most important behavior change — the EOF-as-shutdown assumption is what caused the motivating incident.

### 4.4 Backoff

Capped exponential backoff between reader recreations: initial 500ms, doubling (500ms → 1s → 2s → 4s → 8s → 10s), capped at **10s**, reset to initial on the next successful fetch. The backoff state lives on the `Consumer` (so `recreateCount` and the current backoff window stay consistent across recreations). A bounded max keeps a broker outage from producing minute-long silent windows; the 500ms floor prevents a hot loop against a broker that rejects connections instantly.

No jitter is required for a dev cluster with single-digit consumer replicas per group; revisit if we deploy at scale.

### 4.5 Observable state on `Consumer`

Add to the `Consumer` struct (manager.go:154):
- `aliveSince time.Time` — when the *current* reader was created.
- `lastFetchAt time.Time` — most recent successful `FetchMessage` return.
- `lastErrorAt time.Time` — most recent fetch failure (including ones that triggered a recreate).
- `lastError string` — message of the most recent fetch failure; cleared on successful fetch.
- `recreateCount int` — number of times the reader has been rebuilt for this consumer since process start (monotonic).
- `brokers []string`, `groupId string`, `topic string`, `name string` — already present as local to `AddConsumer`; store on the struct so the debug route can report them without re-deriving.

All writes happen under `c.mu` (same mutex that protects `handlers`); all reads from the debug route happen under the same mutex and copy out.

### 4.6 Debug route

Add a new exported function in `libs/atlas-kafka/consumer` (new file, e.g. `debug.go`):

```go
// DebugRouteInitializer returns a server.RouteInitializer that registers
// GET /debug/consumers on a libs/atlas-rest/server router. The route is
// tenant-agnostic, read-only, and safe to mount on any service's main REST
// server. Pair with the service's existing base path (typically "/api/")
// to yield the canonical URL GET /api/debug/consumers.
func DebugRouteInitializer(m *Manager) server.RouteInitializer
```

Registered path (under the service's base path `/api/`): `GET /api/debug/consumers`

**Response content-type:** `application/vnd.api+json`

**Response shape (JSON:API):**

```json
{
  "data": [
    {
      "type": "consumers",
      "id": "<topic>",
      "attributes": {
        "name": "asset_status_event",
        "topic": "EVENT_TOPIC_ASSET_STATUS",
        "groupId": "Channel Service - e7fb1d7e-47b8-46bd-97dc-867d93530001",
        "brokers": ["kafka-broker-0.kafka:9092"],
        "aliveSince": "2026-04-21T00:52:15.451Z",
        "lastFetchAt": "2026-04-21T01:02:14.712Z",
        "lastErrorAt": "2026-04-21T00:58:03.115Z",
        "lastError": "EOF",
        "recreateCount": 2,
        "handlerCount": 8
      }
    }
  ]
}
```

**Resource id** is the topic name — unique per `Manager` (`AddConsumer` already rejects duplicates at manager.go:88).

**Tenancy:** none. Consumer groups are per-service-process, not per-tenant; the route reports service-level state. The route is not exposed via the public ingress (see §4.8), so no tenant-header gating is required. This is an explicit decision — see §8 Observability.

**Serialization:** use `github.com/manyminds/api2go/jsonapi` (the library already in use across Atlas services per the `GetName()` resource-type pattern). The `DebugRouteInitializer` constructs a response model that implements `GetName() string { return "consumers" }` and `GetID() string { return topic }`.

### 4.7 Service REST scaffolding

Every service that calls `consumer.GetManager().AddConsumer(...)` must:

1. Own a `libs/atlas-rest/server.Builder` running on `REST_PORT` (existing convention).
2. Register the debug route: `.AddRouteInitializer(consumer.DebugRouteInitializer(consumer.GetManager()))`.

**Services that already own a REST server (41 services)** gain exactly one new line: the `AddRouteInitializer` call. Example (atlas-quest, main.go:83–89 today):

```go
server.New(l).
    WithContext(tdm.Context()).
    WithWaitGroup(tdm.WaitGroup()).
    SetBasePath(GetServer().GetPrefix()).
    SetPort(os.Getenv("REST_PORT")).
    AddRouteInitializer(quest.InitResource(GetServer())(db)).
    AddRouteInitializer(consumer.DebugRouteInitializer(consumer.GetManager())).  // NEW
    Run()
```

**Services without a REST server today (8 services)** gain a minimal REST server scaffold:

```go
server.New(l).
    WithContext(tdm.Context()).
    WithWaitGroup(tdm.WaitGroup()).
    SetBasePath("/api/").
    SetPort(os.Getenv("REST_PORT")).
    AddRouteInitializer(consumer.DebugRouteInitializer(consumer.GetManager())).
    Run()
```

These 8 are: atlas-channel, atlas-asset-expiration, atlas-consumables, atlas-expressions, atlas-fame, atlas-login, atlas-messages, atlas-monster-death.

For each, the scaffolding is identical: import `github.com/Chronicle20/atlas/libs/atlas-rest/server`, set `REST_PORT` in the deployment manifests, and run the builder. No other routes are added. The 8 services stay as tight as they are today in every other respect.

**Shutdown ordering:** `server.Builder.Run()` already participates in the `WaitGroup` + `ctx`-cancel pattern used elsewhere in the codebase; teardown sequencing is unchanged.

### 4.8 Ingress and access

The debug route is **not** added to `deploy/k8s/ingress.yaml` in this task. Access is cluster-internal:
- `kubectl port-forward pod/<pod-name> 8080:8080` then `curl localhost:8080/api/debug/consumers`.
- `kubectl exec pod/<pod-name> -- curl localhost:8080/api/debug/consumers`.

This is intentional. The debug route is a dev/ops surface, not a production API. Opening it to the public ingress would require (a) a per-service ingress rule to disambiguate the path (which collides across 49 services today), and (b) a security review of unauthenticated consumer-state disclosure. Both are follow-up work if the need arises. The design captures enough information that adding a per-service nginx rule later (e.g., `location ~ ^/api/debug/(?<svc>[^/]+)/consumers$ { proxy_pass http://atlas-$svc:8080/api/debug/consumers; }`) is straightforward.

### 4.9 Deployment manifests

Every service that gains a new REST server scaffold (the 8 HTTP-less services) needs a `REST_PORT` env var and a `containerPort: 8080` entry in its k8s deployment manifest under `deploy/k8s/`. If the service already exposes port 8080 for another reason, no change is needed beyond adding `REST_PORT`.

For the 41 services that already own a REST server, no manifest changes.

### 4.10 `ResetInstance` compatibility

`manager.go:56` `ResetInstance` is used by tests to null the singleton. It must continue to work. New Manager state must be held on the Manager instance, not in package-level globals, so `ResetInstance` actually resets it.

## 5. API Surface

### 5.1 New Go APIs

In `libs/atlas-kafka/consumer`:

- `func (m *Manager) DebugHandler() http.Handler` — returns the raw `http.Handler` for `GET /debug/consumers`. Available for callers who want to mount it on a non-Atlas router.
- `func DebugRouteInitializer(m *Manager) server.RouteInitializer` — the canonical wiring for Atlas services; registers `GET /debug/consumers` on a `libs/atlas-rest/server` router.

No existing exported signatures change. `AddConsumer`, `RegisterHandler`, `AddConsumerAndRegister`, `RemoveHandler`, `GetManager`, `ResetInstance`, `NewConfig`, `SetStartOffset`, `SetMaxWait`, `SetHeaderParsers`, `ConfigReaderProducer`, `KafkaReader`, `MessageReader`, `MessageCommitter`, `ReaderProducer`, `ManagerConfig` are all source-compatible.

### 5.2 New HTTP surface

- `GET /api/debug/consumers` on every consumer-owning service's existing (or newly-added minimal) REST server. JSON:API response per §4.6. Not routed through the public ingress.

No other HTTP surface changes.

### 5.3 New Kafka surface

None. This task does not touch any topic, producer, or event body.

### 5.4 Config surface

- No new env vars. `REST_PORT` is the existing convention; the 8 HTTP-less services adopt it.

## 6. Data Model

No database changes. No migrations. No persisted state. All new state (`aliveSince`, `lastFetchAt`, `lastErrorAt`, `lastError`, `recreateCount`) is in-process only and resets on process restart — which is the correct behavior for "is this consumer alive right now?"

## 7. Service Impact

| Service / Library | Change |
|---|---|
| `libs/atlas-kafka/consumer/manager.go` | Rewrite `Consumer.start` into an outer reader-lifecycle loop + inner `runFetchLoop` (§4.1–§4.3). Add capped exponential backoff (§4.4). Add observable state fields on `Consumer` and thread-safe accessors (§4.5). Add `Manager.DebugHandler()`. Move `kafka.ReaderConfig` build into a stored field so the goroutine can recreate. |
| `libs/atlas-kafka/consumer/debug.go` (new) | `DebugRouteInitializer(m *Manager) server.RouteInitializer`. JSON:API serialization of the consumer list. Imports `libs/atlas-rest/server` and `api2go/jsonapi`. |
| `libs/atlas-kafka/consumer/manager_test.go` | Add cases: EOF from `FetchMessage` → outer loop recreates reader, `recreateCount` increments. Retry-exhaustion error → outer loop recreates, backoff observed. `context.Canceled` → clean exit, no recreate. Two handlers registered, one recreate event, both handlers survive. |
| `libs/atlas-kafka/consumer/debug_test.go` (new) | `DebugRouteInitializer` / `DebugHandler` returns expected JSON:API shape for zero, one, and many consumers. Content-type is `application/vnd.api+json`. |
| 41 service `main.go` files | Add `.AddRouteInitializer(consumer.DebugRouteInitializer(consumer.GetManager()))` to the existing `server.New(l)...Run()` chain. |
| 8 service `main.go` files (atlas-channel, atlas-asset-expiration, atlas-consumables, atlas-expressions, atlas-fame, atlas-login, atlas-messages, atlas-monster-death) | Add a minimal `libs/atlas-rest/server.Builder` scaffold exposing only `GET /api/debug/consumers`. Import `atlas-rest/server`. Read `REST_PORT`. |
| `deploy/k8s/*.yaml` for the 8 services above | Add `REST_PORT: "8080"` env var and `containerPort: 8080` if not already present. No ingress rules in this task. |
| `libs/atlas-kafka/consumer/config.go`, `header.go` | No change. |

No changes to: producers, handlers, topic definitions, event payloads, database schemas, other REST resources, UI, saga definitions, conversation scripts, or any game-facing behavior.

## 8. Non-Functional Requirements

**Performance.**
- Steady-state cost: one additional `time.Now()` on the successful-fetch path per message; one mutex-guarded struct update per recreate. Negligible.
- Failure-state cost: exponential backoff capped at 10s bounds how often a broken broker is hit. A single pod reconnecting against a broker that refuses for ~1 minute will attempt roughly `log2(60s / 0.5s) ≈ 7` recreates, each incurring a consumer-group rejoin and rebalance. Across 49 pods on a shared broker this is still cluster-tolerable.
- Rebalance impact: each reader recreation triggers a fresh consumer-group join. Groups in atlas today typically have a single member (one pod per service per consumer group), so rebalances are cheap. The 10s cap keeps even a storm's worst case below one rebalance every 10 seconds per consumer.
- HTTP surface: adding one route to an existing server is free. The 8 HTTP-less services gain a single `http.Server` goroutine — negligible resource overhead per pod.

**Reliability & availability.**
- Self-healing is the core goal; there should be no failure mode in which a transient broker disconnect silently removes a topic from the service.
- Process restart is explicitly avoided — atlas-channel maintains live game sessions and a pod crash disconnects all connected players. Recovery without restart is a hard design constraint.

**Backwards compatibility.**
- Public `libs/atlas-kafka/consumer` API is source-compatible. Services that have not yet added the `DebugRouteInitializer` line continue to work; they just lack the debug endpoint.
- On-wire Kafka traffic is unchanged — no producer changes, no topic changes, no payload changes.
- Consumer-group behavior is unchanged in steady state. The only new behavior is "when a reader dies, create a new one with the same group id," which from the broker's perspective looks like a consumer rejoining.

**Observability.**
- Log at `Info`: each recreate attempt with the preceding error and the attempt number. This is the diagnostic that was missing during the incident.
- Log at `Info`: shutdown-on-ctx-cancel (unchanged behavior, preserved).
- Log at `Warn`: handler errors and commit failures (preserved from existing code).
- Debug route JSON:API payload includes timestamps in RFC 3339 with `Z` timezone.
- No stdout/log pollution in steady-state success — handler panic-recovery and retry-inner-attempt logs remain at their existing levels.

**Security.**
- Debug route is unauthenticated. Justification: it runs on a port that is **not** exposed via ingress; access requires cluster-internal network or `kubectl port-forward`; data is not tenant-scoped and contains no credentials, PII, or game state. If this surface is ever promoted to the public ingress, an auth layer must be added at that time — flag in `risks.md`.
- No new credentials, no new secrets, no new inbound network paths through the ingress.

**Multi-tenancy.**
- Debug route is tenant-agnostic. Consumer groups are per-service-process, not per-tenant; the data reports service-level state (one fetcher per topic, shared across tenants). Ingress exposure — if added later — should **not** require a tenant header, nor should it strip one into the response; the response shape is tenant-free by design.

**Testing.**
- Tests use the existing `ReaderProducer`-injection seam (`ConfigReaderProducer`, manager.go:41) to swap in a fake `KafkaReader` whose `FetchMessage` can be scripted per invocation (EOF, context.Canceled, timeout errors, successful reads).
- Fake reader counts `Close` calls to assert the outer loop closes the dead reader before rebuilding.
- Time is injected where possible (a clock interface on `Consumer`, or the test patches `time.Now` via a package-level var). Prefer the first.
- Debug route tests exercise the JSON:API response shape with fixture `Consumer` structs constructed directly (bypassing `AddConsumer` → `start`) to keep the test deterministic.

## 9. Open Questions

- Whether `api2go/jsonapi` (used by other atlas-ms REST resources) is the right serializer for this read-only list endpoint or whether a hand-rolled JSON:API response is cleaner for a one-shot debug route. Lean toward `api2go/jsonapi` for consistency; decide at implementation time based on fit.
- Whether to pass the reader-per-message down through `processMessage` for commit or to protect `c.reader` under the mutex for mid-commit safety. Either works; the decision is ergonomic — pick the cleaner one during implementation.
- Whether `recreateCount` should be in `atomic.Int64` rather than mutex-guarded. Mutex is adequate given low churn; `atomic` would slightly simplify reads from the debug handler. Decide at implementation time.
- Whether the 8 HTTP-less services need any other routes in their new minimal REST server (e.g., a root `/` or health-check). Default: no. Add only `GET /api/debug/consumers`.

## 10. Acceptance Criteria

### Behavioral

- [ ] A consumer whose `FetchMessage` returns `io.EOF` once (then succeeds on the rebuilt reader) logs `Fetcher exited with EOF; recreating reader.` at Info, rebuilds the reader, resumes consuming, and increments `recreateCount` to 1 on the debug route.
- [ ] A consumer whose `FetchMessage` returns repeated transient errors (three retries inside a single `runFetchLoop` call) rebuilds the reader and resumes consuming on the next outer-loop iteration.
- [ ] A consumer that hits a broker that refuses connections for 30 seconds rebuilds the reader multiple times with exponential backoff capped at 10 seconds; `recreateCount` on the debug route reflects the number of rebuilds; once the broker is reachable again, the consumer resumes consuming without operator action.
- [ ] A consumer whose parent `ctx` is canceled returns cleanly within one `MaxWait` tick, logs `Parent context canceled; shutting down topic consumer.`, and **does not** attempt to rebuild.
- [ ] `io.EOF` is never interpreted as shutdown when `ctx` is still alive.
- [ ] A handler panic is still recovered by `safeHandle` and does not affect the reader-lifecycle loop.

### Observability

- [ ] `GET /api/debug/consumers` on every consumer-owning service returns `application/vnd.api+json` with one entry per registered consumer containing all fields listed in §4.6.
- [ ] `lastFetchAt` updates on every successful `FetchMessage` return; `lastErrorAt` and `lastError` update on every fetch failure; `recreateCount` is monotonic; `aliveSince` resets on each reader rebuild.
- [ ] The 8 previously-HTTP-less services (atlas-channel, atlas-asset-expiration, atlas-consumables, atlas-expressions, atlas-fame, atlas-login, atlas-messages, atlas-monster-death) respond to `GET /api/debug/consumers` on their `REST_PORT`.
- [ ] The debug route is reachable via `kubectl port-forward` for any consumer-owning service, verified manually for at least atlas-channel and atlas-quest during implementation.

### Non-regression

- [ ] All services build cleanly on the new `libs/atlas-kafka/consumer` API.
- [ ] All existing `libs/atlas-kafka/consumer` tests pass unchanged.
- [ ] All existing service-level tests pass unchanged.
- [ ] Docker builds for the primary affected services (at minimum atlas-channel, atlas-quest, atlas-inventory, atlas-saga-orchestrator, atlas-npc-conversations, plus all 8 HTTP-less services) succeed against the new library version, per project CLAUDE.md ("Always verify Docker builds when changing shared libraries").
- [ ] Kafka topic list, consumer group list, and lag numbers in `kafka-consumer-groups --describe` are equivalent to pre-change behavior for a service running under steady state with no broker disruption.
- [ ] No in-flight game sessions are terminated as a result of this change during rolling deploy (implicit — no process crashes are introduced).

### Tests

- [ ] `manager_test.go` covers the EOF-recreate path with a scripted `KafkaReader` fake.
- [ ] `manager_test.go` covers the retry-exhaustion-recreate path.
- [ ] `manager_test.go` covers `ctx`-cancel → clean exit, no recreate.
- [ ] `manager_test.go` covers backoff bounds (first recreate ~500ms, later recreates capped at 10s; values verified via an injectable clock).
- [ ] `debug_test.go` covers the debug route response shape for zero, one, and multi-consumer `Manager` instances, and asserts content-type `application/vnd.api+json`.
- [ ] A new integration-style test (may live in `manager_test.go` or a sibling file) confirms a real handler registered via `RegisterHandler` still receives messages after the reader is force-rebuilt.

### Build

- [ ] `libs/atlas-kafka/consumer` builds.
- [ ] All 49 services that own Kafka consumers build with the updated wiring.
- [ ] The 8 previously-HTTP-less services build with their new minimal REST scaffold and successfully serve `GET /api/debug/consumers` in a manual smoke test.
- [ ] `go vet ./...` clean across the affected tree.
- [ ] Docker builds green for the services exercised by task-013/014/015 recent work (sanity overlap) plus all 8 HTTP-less services.
- [ ] k8s manifests under `deploy/k8s/` for the 8 HTTP-less services include `REST_PORT` and `containerPort: 8080`.
