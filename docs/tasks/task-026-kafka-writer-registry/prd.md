# Kafka Writer Registry — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-25

---

## 1. Overview

The atlas-kafka producer library currently constructs a brand-new `kafka.Writer` for every message it publishes, writes the message, and immediately closes the Writer. This pattern silently breaks every stateful kafka-go partition balancer (`LeastBytes`, `RoundRobin`) because each Writer starts with all internal counters at zero and tie-breaks to the first partition. The visible symptom is that publishers with `nil` keys cluster all of their traffic onto a single partition, defeating the purpose of multi-partition topics and rendering replica-based horizontal scaling of consumers ineffective.

This task replaces the create-write-close-per-message pattern with a registry of long-lived, per-topic `kafka.Writer` instances. The registry is a singleton inside `libs/atlas-kafka/producer/`, mirrors the existing `consumer.GetManager()` precedent, and integrates with the existing `service.GetTeardownManager()` so Writers flush and close on graceful shutdown. Each service's `main.go` initializes the registry and registers its `Close()` as a teardown function. The public callsite shape (`producer.ProviderImpl(l)(ctx)(token)` returning a `MessageProducer`) is preserved so the 163 existing producer callsites across the codebase do not require changes.

The change is structural, not behavioral: the default `LeastBytes` balancer is retained, no message routing logic changes, no consumer code changes. The only observable difference is that messages now distribute across partitions as the balancer was always intended to do, which restores the throughput benefit of running multiple consumer replicas on multi-partition topics.

## 2. Goals

Primary goals:
- Eliminate the per-publish Writer construct/close cycle in `libs/atlas-kafka/producer/`.
- Introduce a thread-safe singleton Writer registry, lazily populated per topic on first publish.
- Wire the registry into every service's `main.go` so its shutdown participates in the existing teardown manager.
- Preserve the existing `MessageProducer` and `ProviderImpl` callsite signatures so producer callsites remain untouched.
- Restore correct partition distribution for messages with `nil` keys under the default `LeastBytes` balancer.

Non-goals:
- Changing the default balancer (`LeastBytes` stays).
- Fixing service-specific partitioning bugs whose root cause was this pattern (e.g. atlas-data `START_WORKER` skew). Those become trivially fixable after this work, but are owned by separate tasks.
- Consolidating the 63 per-service `kafka/producer/producer.go` wrappers into the shared library. The wrappers stay; only what they delegate to changes.
- Consumer-side changes.
- Topic provisioning, partition counts, or any deploy-manifest changes.
- Adding new Kafka-related observability beyond what naturally surfaces from the registry's lifecycle (init log, teardown log).

## 3. User Stories

- As a platform engineer, I want producers to reuse one `kafka.Writer` per topic so that the `LeastBytes` balancer's internal byte-counters persist across publishes and partitions actually balance.
- As an operator scaling a service horizontally, I want messages from a single producer pod to spread across all partitions of a topic so that adding consumer replicas yields a proportional throughput increase.
- As a service author writing a new producer callsite, I want the existing `producer.ProviderImpl(l)(ctx)(token)` shape to keep working without surprises so that I can copy patterns from neighboring services.
- As a service author bringing up a new service, I want one canonical line of `main.go` boilerplate to initialize the registry and register its teardown so that the wiring is identical across all 63 services.
- As an SRE, I want producer Writers to flush in-flight batches on graceful shutdown so that we don't lose messages on rolling deploys.

## 4. Functional Requirements

### 4.1 Writer Registry

- `libs/atlas-kafka/producer/` MUST expose a registry type — proposed name `Manager` — that owns a map of resolved-topic-name → `*kafka.Writer`.
- The registry MUST be a process-wide singleton, accessed via `producer.GetManager()`, mirroring `consumer.GetManager()`.
- The registry's internal map MUST be guarded by a `sync.RWMutex` (or equivalent `sync.Map` if a benchmark justifies it). All access patterns MUST be safe for concurrent use across goroutines.
- Lookups MUST be lazy: on first request for a topic the registry constructs the `kafka.Writer` (using the same configuration shape currently in `WriterProvider`), stores it under the resolved topic name, and returns it. Subsequent requests for the same topic MUST return the same `*kafka.Writer` instance.
- The Writer configuration MUST remain functionally equivalent to today's `WriterProvider`:
  - `Addr: kafka.TCP(os.Getenv("BOOTSTRAP_SERVERS"))`
  - `Topic` set to the resolved topic name
  - `Balancer: &kafka.LeastBytes{}`
  - `BatchTimeout: 50 * time.Millisecond`
  - `AllowAutoTopicCreation: true`
- The registry MUST expose a `Close()` (or `Shutdown()`) method that iterates all registered Writers and calls `Close()` on each, in any order. `Close()` MUST be idempotent: calling it twice is a no-op on the second call.
- The registry's `Close()` MUST log (at info level) the number of Writers it shut down and any errors from individual `Writer.Close()` calls. Errors MUST NOT short-circuit the loop — every Writer gets a chance to close.

### 4.2 Producer flow changes

- `libs/atlas-kafka/producer/producer.go`'s `Produce` function MUST be updated so that the per-call code path no longer constructs or closes a Writer.
- The existing `WriterProvider` function MAY be deleted, kept as a deprecated alias, or repurposed to delegate to the registry. The decision MUST keep the public `Produce(l)(provider model.Provider[Writer])(decorators ...HeaderDecorator) MessageProducer` signature stable — i.e. callers can still pass any `model.Provider[Writer]`.
- Inside `Produce`, the line that calls `w.Close()` after writing messages MUST be removed.
- `tryMessage`'s retry behavior MUST be preserved unchanged.
- The `MessageProducer` type and the `RawMessage`/`MessageProvider`/`SingleMessageProvider`/`transformer` helpers in `message.go` MUST remain unchanged.
- Span and tenant header decorators MUST continue to be evaluated per-call (not per-Writer-lifetime), so headers reflect the request context that triggered the publish.

### 4.3 Per-service producer wrappers

- Each of the 63 `services/*/atlas.com/*/kafka/producer/producer.go` files MUST be updated so its `ProviderImpl(l)(ctx)(token)` returns a `MessageProducer` that draws its Writer from the registry instead of constructing a fresh one.
- The exported function names, parameter shapes, and return types MUST NOT change.
- The behavior visible to callsites (`producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(messageProvider)`) MUST be functionally identical aside from the Writer-lifetime fix.

### 4.4 Service `main.go` integration

- Each of the 63 services MUST have its `main.go` updated to:
  - Initialize the producer manager (no-arg or logger-arg call, consistent across services).
  - Register `producer.GetManager().Close` (or equivalent) with `tdm.TeardownFunc(...)` so it runs during graceful shutdown.
- The init/registration line MUST be added in a consistent location across all 63 `main.go` files. Proposed convention: immediately after consumer manager setup, before REST server start.
- Teardown ordering relies on the existing slice-based registration order in `service.GetTeardownManager()`. The PRD does not mandate explicit ordering beyond placing the producer registration after the consumer registration, which makes producers tear down after consumers in the natural order.

### 4.5 Backwards compatibility

- All 163 producer callsites that currently call `producer.ProviderImpl(l)(ctx)(EnvSomethingTopic)(someProvider)` MUST continue to compile and behave identically (modulo the partition-distribution improvement).
- No changes to `producer.SingleMessageProvider`, `producer.MessageProvider`, `producer.CreateKey`, `producer.SpanHeaderDecorator`, or `producer.TenantHeaderDecorator`.
- Consumer-side code MUST NOT be touched.

## 5. API Surface

This is a library refactor. There are no HTTP endpoint changes. The library API surface changes are:

### 5.1 New exports in `libs/atlas-kafka/producer/`

- `producer.GetManager() *Manager` — returns the process-wide registry singleton.
- `Manager.Writer(token string) (Writer, error)` — resolves the env-token to a topic and returns (or lazily constructs) the long-lived Writer for that topic. (Exact name and shape may evolve during design phase; the requirement is "one stable accessor that returns a Writer keyed by topic env-token.")
- `Manager.Close() error` — closes all registered Writers. Idempotent.

### 5.2 Modified internals

- `Produce(l)(provider)(decorators...)` retains its signature but its returned `MessageProducer` no longer closes the Writer after publishing.
- `WriterProvider(provider topic.Provider) model.Provider[Writer]` either remains as a passthrough that delegates to the manager, or is deprecated. Either way, it MUST NOT instantiate a fresh `kafka.Writer` per call from a long-running service.

### 5.3 Unchanged

- `MessageProducer` type
- `RawMessage`, `MessageProvider`, `SingleMessageProvider`, `CreateKey`, `transformer`
- `SpanHeaderDecorator`, `TenantHeaderDecorator`, `HeaderDecorator`
- All per-service `ProviderImpl` exported signatures

## 6. Data Model

No data model changes. No database schema changes. No new entities.

## 7. Service Impact

### 7.1 `libs/atlas-kafka` (primary)

- New `Manager` type and singleton accessor in `libs/atlas-kafka/producer/`.
- Modified `producer.go` Writer lifecycle (no per-call close).
- New unit tests covering: concurrent registration, idempotent shutdown, lazy creation, single-instance-per-topic guarantee.
- Existing `producer_test.go` updated for new lifecycle assumptions.

### 7.2 All 63 services with a `kafka/producer/producer.go` wrapper

The 63 wrappers (one per service) all follow the same shape today. Each requires:
- A one-file edit in `services/<svc>/atlas.com/<name>/kafka/producer/producer.go` to delegate to the registry.
- A two-line `main.go` edit (one init, one teardown registration).

Affected services (from current code survey, alphabetical):
atlas-account, atlas-asset-expiration, atlas-ban, atlas-buffs, atlas-cashshop, atlas-chairs, atlas-chalkboards, atlas-channel, atlas-character, atlas-character-factory, atlas-configurations, atlas-consumables, atlas-data, atlas-drop-information, atlas-drops, atlas-effective-stats, atlas-expressions, atlas-fame, atlas-families, atlas-gachapons, atlas-guilds, atlas-inventory, atlas-invites, atlas-keys, atlas-login, atlas-map-actions, atlas-maps, atlas-marriages, atlas-merchant, atlas-messages, atlas-messengers, atlas-monster-death, atlas-monsters, atlas-notes, atlas-npc-conversations, atlas-npc-shops, atlas-parties, atlas-party-quests, atlas-pets, atlas-portal-actions, atlas-portals, atlas-query-aggregator, atlas-quest, atlas-rates, atlas-reactor-actions, atlas-reactors, atlas-saga-orchestrator, atlas-skills, atlas-storage, atlas-tenants, atlas-transports, atlas-world, plus any not-yet-discovered service that imports the producer library.

(The implementation phase MUST grep the repository for `kafka/producer/producer.go` and `producer.ProviderImpl` to confirm the final list before completion. The migration plan covers this.)

### 7.3 No-impact services

- `atlas-ui` (frontend, no Kafka producer)
- Any service that only consumes Kafka and never produces

## 8. Non-Functional Requirements

### 8.1 Performance

- Per-message publish overhead MUST decrease (no Writer construction, no TCP handshake, no Close per message). Improvement is incidental — not a goal in itself.
- The `LeastBytes` balancer MUST distribute messages across partitions of multi-partition topics. Validation: a smoke test on at least one service (recommended: atlas-data, where the symptom was discovered) confirms messages spread across partitions after the change.
- The registry's lookup hot path MUST be a read-locked map access in the steady state (i.e. once a Writer is registered, subsequent lookups MUST NOT block on a write mutex).

### 8.2 Reliability

- Graceful shutdown MUST flush in-flight batches. Validation: a teardown unit test that calls `Manager.Close()` and asserts that pending writes complete before the call returns.
- A failing `Writer.Close()` on one topic MUST NOT prevent other topics' Writers from closing.
- Lazy Writer construction failures (e.g. invalid env-token) MUST be surfaced as errors to the publishing callsite, not as goroutine panics.

### 8.3 Concurrency

- The `kafka.Writer` type from `segmentio/kafka-go` is documented as safe for concurrent use; the registry relies on this contract.
- The registry MUST tolerate concurrent first-time lookups for the same topic from multiple goroutines without creating duplicate Writers (canonical double-checked-locking or `sync.Once`-per-topic pattern).

### 8.4 Observability

- The registry MUST log (at info level) when it first creates a Writer for a topic, including the resolved topic name.
- The registry MUST log a summary on `Close()`: count closed, count errored.
- No new metrics emission is required by this task. (If desired, that's a separate observability follow-up.)

### 8.5 Multi-tenancy

- Tenant header injection currently happens via `producer.TenantHeaderDecorator(ctx)` evaluated per-call inside `Produce`. This MUST continue to work identically. The fix MUST NOT pre-bind tenant context to a Writer.
- Span context injection MUST continue to be per-call.

### 8.6 Security

- No change. Bootstrap server, auth, and TLS configuration (if any, set via env vars) flow through unchanged.

## 9. Open Questions

- Exact name of the registry type and accessor. Candidates: `Manager` / `GetManager()` (mirrors consumer), `Registry` / `GetRegistry()` (more descriptive), `WriterPool` / `GetWriterPool()` (different vocabulary). To be settled in the design phase.
- Whether `WriterProvider` is deleted, deprecated, or quietly delegated to the registry. Influences whether any non-discovered callsite breaks. Default: keep as a deprecated alias that delegates to the registry, remove in a follow-up.
- Whether the registry's `Close()` should be wrapped in a `context.Context` with a deadline so that a hanging Kafka broker can't block shutdown indefinitely. Default: no, match `kafka.Writer.Close()`'s native behavior; revisit if shutdown hangs are observed.
- Whether to expose a debug HTTP handler (analogous to `consumer.GetManager().DebugHandler()`) listing registered topics and their Writer state. Default: no, deferred to a follow-up.

## 10. Acceptance Criteria

A reviewer can mark this task complete only when **all** of the following are true:

- [ ] `libs/atlas-kafka/producer/` exposes a singleton registry that hands out long-lived Writers keyed by resolved topic name, with concurrent-safe lazy construction.
- [ ] `libs/atlas-kafka/producer/producer.go`'s `Produce` no longer calls `w.Close()` per publish.
- [ ] Every `services/*/atlas.com/*/kafka/producer/producer.go` wrapper delegates to the registry; no wrapper constructs a fresh `kafka.Writer`.
- [ ] Every service `main.go` initializes the registry and registers its `Close()` with `service.GetTeardownManager()`'s `TeardownFunc`.
- [ ] All 163 existing producer callsites compile without modification.
- [ ] `libs/atlas-kafka/producer/producer_test.go` is updated and passes; new tests cover concurrent registration, idempotent shutdown, single-instance-per-topic, and graceful flush.
- [ ] Each affected service builds (`go build ./...`) and its existing tests pass.
- [ ] A manual smoke test (recommended: atlas-data `POST /data/process` with `COMMAND_TOPIC_DATA` configured for ≥4 partitions) confirms messages distribute across partitions instead of clustering on one. Evidence: `kafka-consumer-groups.sh --describe --group "Data Service"` shows non-zero current-offset on multiple partitions after a single ingest run.
- [ ] No consumer-side files are modified.
- [ ] No deploy manifests, configmaps, or topic configurations are modified.
- [ ] Graceful shutdown of any one service flushes in-flight batches: a teardown test or a manual `kubectl rollout restart` shows producer logs reporting "shut down N writers" before the pod exits.
