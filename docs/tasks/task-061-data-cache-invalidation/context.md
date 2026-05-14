# task-061 — Implementation Context

Companion to `prd.md` and `design.md`. Captures the concrete code touchpoints and decisions a fresh implementer needs before opening any file.

## Hard Dependency

**Task-060 v2 must be merged to `main` and this branch must be rebased on top of it before `/execute-task` runs.** Task-060 introduces the two `TenantRegistry` instances in `services/atlas-monsters/atlas.com/monsters/monster/information/` (namespaces `monsters:cache:data` and `monsters:cache:data:not_found`), the `MONSTER_DATA_CACHE_ENABLED` kill-switch, and the cache wiring this task hooks into. Without that merge:

- The `posReg`/`negReg` fields referenced in §4.1 of this plan don't exist.
- `monster/information/cache.go` doesn't exist.
- The `FlushTenant` wrapper in this task has nothing to wrap.

Verify before starting: from this worktree, `git log main --oneline | grep -i 'task-060'` should show the merge. If not, **stop and wait for task-060 to merge**, then `git rebase main`.

## Repository Layout — What This Task Touches

| Path | Action | Why |
|---|---|---|
| `libs/atlas-redis/tenant_registry.go` | Modify (add `Clear` method) | The single new library affordance. |
| `libs/atlas-redis/tenant_registry_test.go` | Create or extend | Tests for `Clear`. (Currently `libs/atlas-redis/registry_test.go` exists; tenant-registry tests live next to the type.) |
| `services/atlas-data/atlas.com/data/data/kafka.go` | Modify (add event types, topic constant) | Producer-side envelope. |
| `services/atlas-data/atlas.com/data/data/producer.go` | Modify (add `dataUpdatedEventProvider`) | Producer wiring. |
| `services/atlas-data/atlas.com/data/data/processor.go` | Modify (insert `emitDataUpdated` call in `StartWorker`) | Emit site after each successful worker. |
| `services/atlas-data/atlas.com/data/data/metrics.go` | Create | Two new counters. |
| `services/atlas-data/atlas.com/data/data/processor_test.go` | Modify (existing file) | Producer behavior tests. |
| `services/atlas-monsters/atlas.com/monsters/monster/information/cache.go` | Modify (task-060 file; add `FlushTenant`) | Per-tenant flush wrapper. |
| `services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go` | Modify (task-060 file) | Wrapper tests. |
| `services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/{kafka.go,consumer.go,handler.go,metrics.go,handler_test.go}` | Create | New consumer. |
| `services/atlas-monsters/atlas.com/monsters/main.go` | Modify (register new consumer + group) | Wire-up. |
| `services/atlas-maps/atlas.com/maps/map/monster/registry.go` | Modify (add `FlushTenant` method) | Spawn-registry per-tenant flush. |
| `services/atlas-maps/atlas.com/maps/map/monster/registry_test.go` | Create | `FlushTenant` tests. |
| `services/atlas-maps/atlas.com/maps/kafka/consumer/data/{kafka.go,consumer.go,handler.go,metrics.go,handler_test.go}` | Create | New consumer. |
| `services/atlas-maps/atlas.com/maps/main.go` | Modify (register new consumer + group) | Wire-up. |
| `deploy/k8s/env-configmap.yaml` | Modify (add `EVENT_TOPIC_DATA`) | Topic name available everywhere. |
| `deploy/compose/.env.example` | Modify (add `EVENT_TOPIC_DATA`) | Compose stack. |

No DB migrations. No HTTP API surface changes. No package renames.

## Locked Decisions (from design.md)

1. **Topic name = `EVENT_TOPIC_DATA`**, env-var configurable, broker auto-create-topics handles bootstrap (matches `COMMAND_TOPIC_DATA`).
2. **Single discriminator `Type = "DATA_UPDATED"`**; consumers MUST switch on `Type` and ignore unknown values (forward compat for future event types).
3. **Kafka message key = tenant UUID string**. Tenant headers also flow via `TenantHeaderDecorator(ctx)` (existing producer wrapper).
4. **Shared consumer groups, NOT per-pod.** Group id strings:
   - atlas-monsters: `"Monster Data Cache Invalidator"`
   - atlas-maps: `"Map Spawn Registry Invalidator"`
5. **`auto.offset.reset = latest`** via existing `consumer.SetStartOffset(kafka.LastOffset)` decorator (don't replay history on deploy).
6. **`TenantRegistry.Clear`** uses `SCAN` with `COUNT=100` + pipelined `DEL` in batches of 100. NOT Lua-scripted (would block broker).
7. **atlas-maps filters on `Worker == MAP` only** — does NOT flush on `Worker == MONSTER` (verified: `storedSpawnPoint` stores no monster-derived data; `MobTime` comes from map data).
8. **Per-tenant flush on partial-failure**: `firstErr` accumulator; never abort halfway through a flush; `(deleted_so_far, err)` is the contract.
9. **Producer log-and-continue on Kafka error**: a Kafka outage MUST NOT fail a successful data import.
10. **`DATA_EVENTS_PRODUCER_ENABLED` / `DATA_EVENTS_CONSUMER_ENABLED`** default to `true`; unparseable bool also defaults to `true` (defensive).
11. **`SpawnPointRegistry.FlushTenant`** is a parallel SCAN/DEL implementation, NOT a `TenantRegistry.Clear` call. Hand-rolled key shape `atlas:maps:spawn:{tenant}:*` is incompatible with `tenantEntityKey`. Migration to `TenantRegistry` is out of scope.
12. **Header/body tenant disagreement**: WARN log; prefer body. Do not enforce as a gate. (atlas-monsters needs region/version; atlas-maps does not.)
13. **No `consumer.PerPodGroup` helper.** Sketch archived in design §11.4 for future use.
14. **No new `libs/atlas-cache`.** That library was reverted by task-060. The library affordance is exactly one `TenantRegistry.Clear` method.

## Existing Patterns To Match

- **Producer constructor signature**: see `services/atlas-data/atlas.com/data/data/producer.go:9` (`startWorkerCommandProvider`). Use `producer.SingleMessageProvider(key, value)` from `libs/atlas-kafka/producer`.
- **Producer wrapper**: `producer.ProviderImpl(l)(ctx)(EnvTopic)(provider)` — signature already used at `services/atlas-data/atlas.com/data/data/processor.go:83`.
- **Consumer file layout**: see existing `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/` for the canonical 3-file split (`consumer.go`, `kafka.go`, `handler.go`).
- **Consumer registration**: `consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser)` decorator goes on every config; `consumer.SetStartOffset(kafka.LastOffset)` is the additional decorator new for this task.
- **Kafka miniredis test setup**: see `libs/atlas-redis/registry_test.go` `setupTestRedis(t)` for the canonical helper.
- **Metric declaration**: see `services/atlas-maps/atlas.com/maps/character/location/metrics.go` and existing task-060 metrics in `services/atlas-monsters/atlas.com/monsters/monster/information/metrics.go` (post-merge). Use `promauto.NewCounterVec` with `Name`/`Help`.
- **tenant.Model construction in handler**: prefer the ctx-resolved tenant via `tenant.FromContext(ctx)` (populated by `TenantHeaderParser`); fall back to `tenant.Create(uuid, "", 0, 0)` only on header/body disagreement.

## Test Surfaces

| Module | New tests |
|---|---|
| `libs/atlas-redis` | 6 (`Clear` happy/empty/tenant-iso/namespace-iso/partial-failure/race) |
| `services/atlas-data` | 6 (emit on success / not on failure / kill-switch / producer error tolerance / key=tenantId / RFC3339 UTC) |
| `services/atlas-monsters` (cache wrapper) | 4 (clears both / tenant iso / kill-switch / posReg-error-no-block-negReg) |
| `services/atlas-monsters` (consumer) | 6 (Type filter / Worker filter / parse error / happy path / kill-switch / miniredis end-to-end) |
| `services/atlas-maps` (registry) | 4 (deletes all / tenant iso / Redis error / empty tenant) |
| `services/atlas-maps` (consumer) | 5 (Type filter / Worker=MAP path / Worker=MONSTER skip / Worker=NPC skip / parse error) |

All `go test -race ./...` clean.

## Build Order

The four affected Go modules build independently but `libs/atlas-redis` is a transitive dep of `atlas-monsters`, `atlas-maps`, and (via go.work) any consumer of the registry. After editing the library:

```bash
# from worktree root
go build ./libs/atlas-redis/...
( cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test -race ./... )
( cd services/atlas-maps/atlas.com/maps     && go build ./... && go test -race ./... )
( cd services/atlas-data/atlas.com/data     && go build ./... && go test -race ./... )
```

Docker build verification per CLAUDE.md ("Always verify Docker builds when changing shared libraries"):

```bash
docker build -t atlas-monsters:task-061 services/atlas-monsters/atlas.com/monsters
docker build -t atlas-maps:task-061     services/atlas-maps/atlas.com/maps
docker build -t atlas-data:task-061     services/atlas-data/atlas.com/data
```

## What Already Exists (Do Not Reinvent)

- `libs/atlas-redis/keys.go` already exposes `tenantScanPattern(namespace, t)` — reuse, do NOT redefine.
- `libs/atlas-redis/tenant_registry.go` already has `client *goredis.Client` and `namespace string` fields — `Clear` reads them directly.
- `consumer.SetStartOffset(int64)` already exists in `libs/atlas-kafka/consumer/config.go:37`.
- `consumer.TenantHeaderParser` already exists in `libs/atlas-kafka/consumer/header.go`.
- `producer.SingleMessageProvider(key []byte, value any)` already exists in `libs/atlas-kafka/producer`.
- `producer.ProviderImpl(l)(ctx)(envTopic)(provider) error` already exists.
- `services/atlas-data` already has `producer.ProviderImpl` wrapped in `services/atlas-data/atlas.com/data/kafka/producer/producer.go` with `TenantHeaderDecorator(ctx)` attached automatically.
- `services/atlas-maps/.../map/monster/registry.go:259` already has `func (r *SpawnPointRegistry) Reset(ctx context.Context)` — model the new `FlushTenant` after it (but tenant-scoped).

## Out of Scope (Per PRD §2 / Design §14)

- New caches in any service other than atlas-monsters / atlas-maps.
- `SpawnPointRegistry` migration to `TenantRegistry`.
- Per-id selective invalidation.
- Compacted topic / replay guarantees.
- `consumer.PerPodGroup` helper.
- Admin REST endpoint to manually trigger an event.
- Removing the manual `redis-cli DEL` runbook.
- Future event types (`DATA_IMPORT_STARTED`, etc.).

## Risk Notes

- **Auto-create-topics**: relies on broker config, same as `COMMAND_TOPIC_DATA`. If disabled in prod, producer's `WARN` log fires forever and no events flow. Operator action: create topic manually.
- **Region/version on header/body disagreement**: atlas-monsters' `Clear` uses `tenant.Region()`/`MajorVersion()`/`MinorVersion()` to build the SCAN pattern. If body-only fallback fires, those fields are empty/zero and the SCAN targets the wrong prefix. Mitigation: log loudly on disagreement; in practice headers and body both come from the same `ctx` so they always agree.
- **Concurrent Put during Clear**: by design, `Clear` deletes everything visible at scan time. New puts during the flush survive. Acceptable per PRD §8.2.
