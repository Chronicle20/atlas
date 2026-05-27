# Dynamic Service Configuration — Implementation Context

Companion to `plan.md`. This is the cheat sheet a fresh subagent should read before picking up a task.

---

## Key files the implementation will touch

### New (created)

```
libs/atlas-outbox/
  go.mod, README.md
  entity.go          — gorm Entity, table outbox_entries
  migration.go       — AutoMigrate helper
  outbox.go          — Message struct + Enqueue (NOTIFY-aware)
  drainer.go         — NewDrainer, Run, publish loop, leader election
  lock.go            — pg_try_advisory_lock + per-leader sql.Conn
  notify.go          — pq.Listener wakeup channel
  backfill.go        — Loader + ToBytes signature; idempotent topic re-enqueue
  *_test.go          — sqlite for unit; testcontainers postgres for integration (build tag)

libs/atlas-kafka/consumer/offsets.go     — ReadEndOffsets(ctx, brokers, topic) → map[partition]int64
libs/atlas-kafka/consumer/offsets_test.go

services/atlas-configurations/atlas.com/configurations/outbox/envelopes.go
  — NewServiceEnvelope, NewTenantEnvelope; schema_version=1 JSON

services/atlas-channel/atlas.com/channel/listener/
  handle.go           — Handle, State (Active|Draining|Removed), HandlerHandle
  registry.go         — Registry: Add (with body callback), Drain, Snapshot, DrainAll, ref-count
  evict.go            — RegisterEvictor + per-tenant ref count

services/atlas-channel/atlas.com/channel/configuration/projection/
  state.go            — RW-lock-protected serviceConfig + tenantConfigs
  caughtup.go         — End-offset snapshot, atomic.Bool, ReadyChecker
  apply.go            — ComputeOps(prev,next) → []Op (Add|Drain)
  subscriber.go       — registers two consumers (service+tenant topics)
  loop.go             — single goroutine consuming ops chan into listener.Registry
  envelope.go         — decode + tombstone detection

services/atlas-login/atlas.com/login/listener/             — mirrors atlas-channel (simpler drain)
services/atlas-login/atlas.com/login/configuration/projection/  — mirrors atlas-channel
```

### Modified

```
go.work
  — append ./libs/atlas-outbox

libs/atlas-database/
  — no schema change; library reuse via WithoutTenantFilter

services/atlas-configurations/atlas.com/configurations/
  main.go                   — register outbox.Migration; init drainer; teardown
  services/processor.go     — Enqueue inside ExecuteTransaction on Create/Update/DeleteById
  tenants/processor.go      — same shape
  seeder/seeder.go          — outbox.Backfill(services topic) + outbox.Backfill(tenants topic) after seed
  atlas-configurations.yml  — two new topic env vars
  Dockerfile                — add atlas-outbox in four hand-edited spots (CLAUDE.md §Build & Verification)

services/atlas-channel/atlas.com/channel/
  main.go                          — replace configuration.Init block with projection;
                                     lift per-(t,w,c) startup into listener.Registry.Add;
                                     register evictors; SIGTERM DrainAll
  server/registry.go               — slice → map, add Deregister/Get; introduce server.Key
  configuration/registry.go        — shim to projection.State (delete REST callers)
  channel/processor.go             — add Unregister(ch channel.Model) error
  channel/requests.go              — add DELETE request builder against atlas-world
  session/processor.go             — Destroy reorder: emit logout + destroy BEFORE Disconnect
  kafka/consumer/*/consumer.go     — 44 files: InitHandlers returns ([]listener.HandlerHandle, error)
  monster/status_mirror.go (or wherever GetStatusMirror lives) — add Evict(tenantId)
  monster/next_skill_inbox.go — add Evict
  account/registry.go         — add Evict (and remove main-loop InitializeRegistry caller)
  atlas-channel.yml                — topic env vars + DRAIN_DEADLINE_MS + bump terminationGracePeriodSeconds

services/atlas-login/atlas.com/login/
  main.go                          — projection + listener wiring (simpler)
  configuration/registry.go        — shim
  kafka/consumer/*/consumer.go     — 4 files: same InitHandlers signature change
  atlas-login.yml                  — topic env vars

services/atlas-world/atlas.com/world/channel/
  resource.go                      — DELETE /api/world-server/channel-server/{worldId}/{channelId}
  resource_test.go                 — handler tests

libs/atlas-tenant/registry.go (or equivalent)
  — add Unregister(id uuid.UUID)
```

---

## Cross-cutting decisions locked in design.md

- **atlas-world REST surface**: ADD a new DELETE route. Atlas-world's existing `processor.Unregister` is reached only by an internal sweep today; no REST exposure exists.
- **Drain gating**: handler deregister is load-bearing. `server.Model.Is(...)` is NOT extended — keeps it pure-data.
- **Login subscribes to both topics**: tenant topic for protocol-table wiring (matches channel), service topic for IPAddress + per-(t,w,c) listener ports.
- **In-flight handler dispatch during drain**: NOT waited on. `consumer.Manager.RemoveHandler` prevents new dispatch; in-flight goroutines may write to a closed socket and that's already tolerated by the producer write path.
- **Outbox table is not tenant-scoped**: queries use `database.WithoutTenantFilter(ctx)`.
- **Single partition per topic** (PRD §5.3): ordering guaranteed by Kafka. Compaction preserves latest per key.
- **At-least-once delivery**: consumers MUST be idempotent. Documented in `libs/atlas-outbox/README.md`.

---

## Dependencies between phases

```
A (atlas-outbox)        ─┐
B (atlas-kafka offsets) ─┤
C (atlas-world DELETE)  ─┴─► D (atlas-configurations adoption)
                                   │
                                   ▼
                              E (server.Registry shape)
                                   │
                                   ▼
                              F (listener pkg + drain)
                                   │
                                   ▼
                              G (projection)
                                   │
                                   ▼
                              H (InitHandlers sweep) ◄─ depends on F (HandlerHandle type)
                                   │
                                   ▼
                              I (Destroy reorder)
                                   │
                                   ▼
                              J (atlas-channel main rewire)
                                   │
                                   ▼
                              K (atlas-login mirror)
                                   │
                                   ▼
                              L (e2e integration tests)
                                   │
                                   ▼
                              M (verification + docker sweep)
```

Phases A, B, C are independent and can be parallelized.

---

## Rollback considerations

- **Outbox library**: pure-additive. Reverting Phase A is safe — no other module imports it until Phase D. Reverting Phase D returns atlas-configurations to direct producer.Send calls, but the migration leaves the `outbox_entries` table behind (a no-op orphan).
- **server.Registry shape change**: slice → map. Reverting requires re-introducing slice iteration ordering, which existing code does NOT depend on (verified by reading `channel/task.go:36-39`). Safe to revert.
- **InitHandlers signature**: rewriting 44 files is mechanical but the rollback requires re-rewriting them. Plan tasks group it into a single commit so revert is one commit.
- **Session.Destroy reorder (Phase I)**: **carries downstream risk**. Any consumer of the Destroy event that assumed the socket was already closed when it observed the event will break. The audit in Task I3 must run before claiming Phase I complete. If a downstream consumer cannot be updated, the reorder must be reverted AND the downstream consumer's assumption must be re-introduced in the new ordering — they are entangled. **Treat I2 as the highest-blast-radius change in this task.**
- **Projection replacing configuration.Init**: removes synchronous REST dependency. Revert restores REST coupling. Safe per se, but means a partial revert (keeping Phase D but undoing Phase G/J) leaves atlas-channel running against atlas-configurations REST while atlas-configurations also publishes events — a benign double-source state.

---

## External assumptions / pre-conditions

- **Postgres ≥ 9.4** (for `pg_advisory_lock` and LISTEN/NOTIFY). atlas-configurations already runs Postgres.
- **Kafka brokers** support log-compacted topics with `cleanup.policy=compact`. Topic provisioning is operational; the service does not auto-create.
- **`terminationGracePeriodSeconds=20s`** in k8s manifests for atlas-channel; current value should be checked.
- **`gorm.io/driver/postgres`** is already in `services/atlas-configurations` go.mod; verify before Phase D.
- **`github.com/lib/pq`** required for `pq.Listener` — may need a `go get` in Phase A6.
- **testcontainers-go** modules for postgres and kafka should already be present in libs go.mods (used elsewhere); confirm before Phase A integration tests.

---

## Test infrastructure cheat sheet

- **Unit tests with sqlite in-memory**: use `gorm.io/driver/sqlite` + `sqlite.Open(":memory:")`. Migration auto-creates tables. Fast, no Docker required.
- **Integration tests with `//go:build integration`**: gated behind a build tag so default `go test ./...` doesn't require Docker. CI invokes with `-tags=integration` when Docker is available.
- **Postgres testcontainer**: `github.com/testcontainers/testcontainers-go/modules/postgres`, image `postgres:16-alpine`.
- **Kafka testcontainer**: `github.com/testcontainers/testcontainers-go/modules/kafka`, image `confluentinc/cp-kafka:7.6.0` (or whatever the project's other tests use — verify by `grep -r testcontainers/kafka`).
- **No `*_testhelpers.go`**: project policy is Builder pattern (per CLAUDE.md §Test Helper Pattern).

---

## Env vars added by this task

| Service | Env Var | Default | Purpose |
|---|---|---|---|
| atlas-configurations | `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` | (operator sets) | service config events |
| atlas-configurations | `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` | (operator sets) | tenant config events |
| atlas-channel | `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` | (operator sets) | subscribes |
| atlas-channel | `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` | (operator sets) | subscribes |
| atlas-channel | `DRAIN_DEADLINE_MS` | `5000` | per-listener drain deadline (capped at 10000) |
| atlas-login | `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` | (operator sets) | subscribes |
| atlas-login | `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` | (operator sets) | subscribes |
| atlas-login | `DRAIN_DEADLINE_MS` | `2000` | shorter drain (stateless sessions) |

---

## Observability events to look for during execution

Logs (structured `logrus` keys):

- `outbox.enqueued` (DEBUG, per row)
- `outbox.batch_published` (INFO, with count)
- `outbox.lock_acquired` / `outbox.lock_lost` (INFO)
- `outbox.publish_failed` (WARN, with attempts + last_error)
- `outbox.sweeper_run` (INFO, with deleted count)
- `projection.caughtup` (INFO, with elapsed ms)
- `projection.applied` (DEBUG, key + op)
- `listener.added` (INFO, key)
- `listener.drain_phase` (INFO, key + phase 1..4)
- `listener.drain_timeout` (WARN, key + remaining sessions)
- `tenant.evicted` (INFO, tenant id)

If `atlas-metrics` exists in this repo, add gauges/histograms per PRD §8.3. If not, log-only is fine — the design explicitly accepts this.

---

## Slash commands relevant during execution

- `/execute-task task-032` — Phase 4 of the workflow; dispatches subagents per plan task.
- `/service-doc <service>` — refresh docs after touching a service. Run for atlas-configurations, atlas-channel, atlas-login, atlas-world.
- `/backend-audit` — adversarial DOM-* audit on touched Go services.
- `/audit-plan` — verifies plan adherence after execution.
- `superpowers:requesting-code-review` — orchestrates the three reviewer agents in parallel; required before opening the PR.

---

## Avoid these foot-guns

- **Do NOT skip the Docker build sweep.** Per CLAUDE.md, only `docker build` exercises the four-place hand-edited lib list in each Dockerfile. `go build`/`go test` against `go.work` will green even when the Dockerfile is missing the new `atlas-outbox` lib reference.
- **Do NOT alias library types**. Plan favors direct `listener.HandlerHandle` over a local alias to keep boundaries clean (CLAUDE.md §Code Patterns).
- **Do NOT add a new domain type for `(t,w,c)` keys.** `world.Id` and `channel.Id` already exist in `libs/atlas-constants` (CLAUDE.md §Code Patterns / DOM-21). `server.Key` composes those plus a `uuid.UUID` for tenant.
- **Do NOT batch the Destroy reorder into the projection rewire commit.** Phase I is the highest-blast-radius commit; keep it alone for clean revert.
- **Do NOT auto-create the compacted topics.** They are operationally provisioned. Subscribers tolerate empty topics at boot (caught-up immediately if end offsets are 0).
- **Do NOT set `DRAIN_DEADLINE_MS` higher than 10000.** Plan code clamps it.

---

## Where to read more

- PRD: `prd.md` (this folder)
- Design: `design.md` (this folder)
- CLAUDE.md: `<repo-root>/CLAUDE.md` — workflow, build verification, code patterns
- Backend guidelines (DOM-* / SUB-* / SEC-*): invoked via `backend-dev-guidelines` skill or `/backend-audit`
- atlas-kafka consumer manager source: `libs/atlas-kafka/consumer/manager.go` — `RegisterHandler` / `RemoveHandler` semantics
- atlas-database transaction helper: `libs/atlas-database/transaction.go`
- atlas-database tenant scope opt-out: `libs/atlas-database/tenant_scope.go:19` (`WithoutTenantFilter`)
