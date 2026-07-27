# Shared Service Bootstrap — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Every Atlas Go service hand-assembles its own startup: a copy-pasted logger bootstrap (56 copies of `logger/init.go`, 3 structurally drifted), a copy-pasted Kafka producer wrapper (52 copies of `kafka/producer/producer.go`, atlas-quest already diverged), and a per-service `main.go` that wires tracer, teardown, consumers, REST server, and — in exactly 4 services — a readiness endpoint and a ~40-line configuration-projection catch-up block including a byte-identical `parseProjectionCatchupTimeout()` helper. This duplication is the mechanism behind every "straggler" bug class in `docs/architectural-improvements.md`: when login/channel were fixed for the config-projection crash (task-090), the services whose `main.go` wasn't touched were silently left behind (RR-3), and any change to the producer decorator chain or log schema requires 50+ synchronized edits that in practice never all happen.

This task consolidates the three duplication findings DUP-1, DUP-2, and DUP-3 — plus the two items the owner explicitly pulled into scope: CP-9 (structured-log field naming standardization, "natural to bundle with DUP-2") and the endpoint half of OPS-2 (baseline `/readyz` in the shared bootstrap). The deliverable is: `Provider`/`ProviderImpl` moved into `libs/atlas-kafka/producer`, `CreateLogger` moved into `libs/atlas-service`, a new `service.Bootstrap(serviceName, opts)` entry point in `libs/atlas-service` owning logger/tracer/teardown/readiness/projection wiring, **full fleet migration** of all 58 service `main.go`s onto it, and deletion of every local copy.

The owner chose full migration over incremental ("full migration", "alias is only tech debt"): partial migration would leave the drift risk this task exists to remove. The task is sequenced **after** task-114 (outbox adoption) and task-116 (processor gen3 unification) land, since both rewrite overlapping surface (producer wiring, `main.go`s) fleet-wide.

## 2. Goals

Primary goals:
- One canonical `ProviderImpl` + `Provider` in `libs/atlas-kafka/producer`; zero service-local copies; all import sites rewritten (no re-export aliases).
- One canonical `CreateLogger` in `libs/atlas-service`; zero service-local `logger/init.go` copies.
- One `service.Bootstrap(serviceName, opts)` in `libs/atlas-service` owning: logger creation, tracer init + teardown, teardown manager, Redis-independent startup ordering, baseline `/readyz` mounting, graceful-shutdown readiness flip, and opt-in configuration-projection wiring (subscriber start, catch-up gate, `parseProjectionCatchupTimeout`).
- All 58 Go services migrated onto `Bootstrap` in this task.
- Every service exposes `/readyz` (today: 4 of 58).
- Structured-log field keys standardized to `snake_case` at emit time (CP-9), without requiring a 1,500-call-site rename.

Non-goals:
- Kubernetes manifest changes: adding `readinessProbe`/`livenessProbe`/resources to the ~55 unprobed workloads is OPS-1's shared-patch work and stays out of scope. This task only guarantees the endpoint exists so OPS-1 has something to probe.
- RR-3 behavioral fixes (atlas-transports / atlas-party-quests captured-slice tickers, atlas-drops `sync.Once`+`Fatalf` registry): the bootstrap makes those migrations cheap, but the migrations themselves are a follow-up.
- OPS-3 readiness-path restructuring (`/readyz` vs `/api/readyz` mounting semantics) — the bootstrap must not *worsen* the trap, but relocating the mount to the root router is a separate decision (flagged in Open Questions).
- DUP-4 (876 copy-pasted `replace` directives) and any go.mod manifest generation.
- Changing what is logged (levels, messages, ECS formatter) — only field-key casing is standardized.
- atlas-ui (TypeScript) and non-service artifacts (atlas-wz-extractor, atlas-pr-bootstrap).

## 3. User Stories

- As a platform maintainer, I want the producer decorator chain (span/tenant headers) defined once so that a change to message headers is a one-file edit, not a 51-file sweep with drift risk.
- As a platform maintainer, I want log-schema changes (formatter, hooks, field naming) defined once so that observability improvements reach all services simultaneously.
- As a service author, I want `service.Bootstrap` to own startup boilerplate so that a new service starts with correct teardown, tracing, readiness, and (if needed) projection wiring by construction, instead of by copy-pasting the nearest `main.go`.
- As an operator, I want every service to expose `/readyz` so that rollout gating (OPS-1) can be applied fleet-wide without per-service code work.
- As an on-call engineer querying Loki, I want one field-name spelling (`character_id`, not `characterId`-or-`character_id`) so that queries don't need dual-spelling disjunctions.
- As a future fixer of a "straggler-class" bug, I want the fix to live in `Bootstrap` so that no service can be left behind by an incomplete sweep.

## 4. Functional Requirements

### 4.1 DUP-1 — Shared Kafka producer wrapper

- FR-1.1: `libs/atlas-kafka/producer` exports `type Provider func(token string) producer.MessageProducer` and `ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) Provider`, byte-equivalent in behavior to the 51 identical service copies (decorator order: `SpanHeaderDecorator(ctx)`, `TenantHeaderDecorator(ctx)`; writer: `ManagerWriterProvider(l)(token)`).
- FR-1.2: All 52 `services/*/atlas.com/*/kafka/producer/producer.go` wrapper files are deleted. Sibling files under each service's `kafka/producer/` tree (per-domain event builders, e.g. atlas-cashshop's `kafka/producer/cashshop/producer.go`, and the two `producer_test.go` files in atlas-reactors/atlas-marriages) are **kept** but rewritten to reference the lib `Provider` type.
- FR-1.3: Every import site of a service-local `kafka/producer` package's `Provider`/`ProviderImpl` (hundreds of files) is rewritten to the lib import path. No local type alias or re-export shim may remain (owner: "alias is only tech debt"; CLAUDE.md: straightforward moves over re-exports).
- FR-1.4: atlas-quest's divergent copy (Provider type only, no `ProviderImpl` — it doesn't wire the emit path in `main.go`) migrates to the lib type the same way; the migration must not force it to re-adopt `ProviderImpl`.
- FR-1.5: Behavior is bit-identical: same headers, same decorator order, same manager usage. This is a move, not a redesign. Any post-task-114 outbox-era changes to the wrapper shape are adopted as found at rebase time.

### 4.2 DUP-2 — Shared logger

- FR-2.1: `libs/atlas-service` exports `CreateLogger(serviceName string) *logrus.Logger` with the canonical behavior: stdout output, `service.name` field hook, `ecslogrus.Formatter`, `LOG_LEVEL` env parsing (invalid values silently keep the default, as today).
- FR-2.2: All 56 `services/*/atlas.com/*/logger/init.go` files (and their now-empty `logger` packages) are deleted; all `main.go`s use the shared logger via `Bootstrap` (FR-4). The 3 drifted variants (atlas-map-actions, atlas-reactor-actions, atlas-portal-actions) are semantically identical (renamed hook struct, reordered declarations) — verified by diff — and unify without behavior change.

### 4.3 CP-9 — Log field-key standardization

- FR-3.1: The shared logger guarantees `snake_case` field keys in emitted log records. Chosen mechanism (subject to design-phase confirmation): an emit-time logrus hook in `CreateLogger` that converts camelCase field keys to snake_case (e.g. `characterId` → `character_id`, `transactionId` → `transaction_id`), so the ~1,500 existing `WithField` call sites (measured: `characterId` 619, `worldId` 462, `transactionId` 271 vs. snake_case 19–218 each) do not need a fleet rename and future call sites cannot reintroduce drift.
- FR-3.2: Dotted ECS/namespaced keys (`service.name`, and any `x.y` form) pass through unchanged; already-snake_case keys pass through unchanged. Conversion must be deterministic and covered by unit tests including collision behavior (a record carrying both `characterId` and `character_id` must not lose data silently — last-writer-wins is acceptable if tested and documented).
- FR-3.3: `docs/observability.md` (or the nearest logging doc) is updated to state snake_case as the canonical spelling and document the normalization hook.

### 4.4 DUP-3 — `service.Bootstrap`

- FR-4.1: `libs/atlas-service` exports `Bootstrap(serviceName string, opts ...Option)` (exact API shape decided in design) that owns, in order: logger creation (FR-2.1), teardown manager acquisition, tracer init with teardown registration (replacing today's per-service `tracing.InitTracer` + `Fatal` + close wiring), and a readiness controller (FR-5). It returns a handle exposing at minimum the logger, the teardown manager (context + waitgroup), and the readiness hook needed for REST server wiring.
- FR-4.2: Configuration-projection wiring is an opt-in `Option` that reproduces the 4-service block: subscriber construction from `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` (warn if unset), unique projection consumer-group id, subscriber start bound to teardown context/waitgroup, catch-up gating with `parseProjectionCatchupTimeout()` (helper moves into the lib; the 4 byte-identical copies in atlas-world/atlas-character-factory/atlas-login/atlas-channel `main.go`s are deleted).
- FR-4.3: Bootstrap does **not** absorb the REST server builder, route registration, Redis connection, consumer registration, or service-specific tasks — those remain in `main.go`, composed with the Bootstrap handle. (Rationale: routes/consumers differ per service; forcing them into options would create a config-object monolith.)
- FR-4.4: All 58 `services/*/atlas.com/*/main.go` files are migrated to `Bootstrap` in this task. A migrated `main.go` must contain no direct calls to `logger.CreateLogger` (local), `tracing.InitTracer`, or hand-rolled shutting-down/readiness flag wiring.
- FR-4.5: Startup failure semantics are preserved: conditions that today `Fatal` at boot (tracer init failure, handler registration failure) still terminate the process with a logged error.

### 4.5 OPS-2 (endpoint half) — Baseline `/readyz`

- FR-5.1: Every service that runs the shared REST server mounts a `/readyz` endpoint. For the ~54 services with no readiness signal today, the baseline semantic is: ready once `Bootstrap` completes and the REST server is serving; not-ready once teardown begins (the shutting-down flip that login/channel/world/character-factory hand-roll today moves into the shared readiness controller).
- FR-5.2: Services with richer gates compose them: the projection option (FR-4.2) automatically ANDs catch-up state into readiness, matching today's 4-service behavior (`SnapshotReady() && !shuttingDown`).
- FR-5.3: Mount path parity: the endpoint mounts under the existing basePath semantics (`/api/readyz` effective path) exactly as the 4 current services do — this task must not silently change the effective URL of existing probed services (`bug_readiness_probe_path_under_api_basepath`). Whether to also expose a root-mounted `/readyz` (OPS-3's fix) is an Open Question for design, not a requirement here.
- FR-5.4: Any service that does not run a REST server (if the migration sweep finds one) is documented in the task's notes with its readiness story rather than silently skipped.

### 4.6 Sequencing & conflict management

- FR-6.1: This task's implementation starts only after task-114 (outbox adoption) and task-116 (processor gen3 unification) merge to main; the worktree rebases onto their landed state before any fleet edits. Both tasks rewrite producer wiring and/or `main.go`/processor surface across many services — the extraction must consume their final shapes (e.g. outbox-era emit paths like atlas-quest's `ProviderImpl`-less wiring) rather than the pre-114 snapshot described here.
- FR-6.2: If, at rebase time, the canonical `producer.go`/`main.go` shapes have changed from what this PRD measured (counts, decorator chain, projection block), the design/plan documents re-measure before locking file lists. The counts in this PRD are a 2026-07-02 snapshot.

## 5. API Surface

No HTTP API changes except:

- `GET /readyz` (effective `/api/readyz` under the standard basePath) on all ~58 services — `200` when ready, `503` when not, matching the existing `server.MountReadiness` contract.

New/changed Go API (shared libs — final shapes in design.md):

- `libs/atlas-kafka/producer`: `type Provider`, `func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) Provider`.
- `libs/atlas-service`: `func CreateLogger(serviceName string) *logrus.Logger`; `func Bootstrap(serviceName string, opts ...Option) *Handle` (name illustrative) exposing logger, teardown context/waitgroup, readiness hook, and projection catch-up gate when enabled; `Option` constructors incl. projection wiring; field-normalization hook (unexported, applied by `CreateLogger`).

## 6. Data Model

None. No database entities, migrations, or Kafka message schema changes. Kafka **transport headers** (span/tenant) must remain byte-identical after the producer move.

## 7. Service Impact

- `libs/atlas-service` — gains `CreateLogger`, snake_case normalization hook, `Bootstrap` + options, projection wiring helper, readiness controller. Currently teardown-only; already in root `Dockerfile` COPY lines and `go.work` — **no new module, no Dockerfile edits**.
- `libs/atlas-kafka` — `producer` package gains `Provider` + `ProviderImpl`.
- All 58 Go services — `main.go` rewritten onto `Bootstrap`; local `logger/` package deleted (56 services); local `kafka/producer/producer.go` wrapper deleted with import-site rewrites throughout the service (52 services); `/readyz` gained (~54 services).
- The 4 projection services (atlas-world, atlas-character-factory, atlas-login, atlas-channel) additionally lose the duplicated projection block + `parseProjectionCatchupTimeout` in favor of the Bootstrap option, with identical runtime behavior.
- `docs/architectural-improvements.md` — DUP-1, DUP-2, DUP-3, CP-9 marked resolved (✓), OPS-2 annotated (endpoint half done, probes pending OPS-1).

## 8. Non-Functional Requirements

- **Behavior preservation:** startup ordering, teardown semantics, Kafka headers, and readiness semantics of the 4 already-migrated services are preserved exactly; the only intentional observable change fleet-wide is snake_case log keys and the new `/readyz` route.
- **Multi-tenancy:** untouched — tenant header decoration moves location, not behavior.
- **Observability:** log volume/levels unchanged; field-key normalization is CPU-trivial (per-entry map rewrite) and must not allocate pathologically on hot paths (benchmark or bound it in design if the hook copies maps).
- **Verification (per CLAUDE.md, all mandatory):** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake all-go-services` clean (every service's `go.mod`-adjacent source is touched); `tools/redis-key-guard.sh` clean.
- **Migration safety:** because all 58 services change, the PR should be reviewable service-by-service (mechanical commits separated from the shared-lib commits) and land as one branch — no partial-fleet intermediate state on main.

## 9. Open Questions

1. **Bootstrap API shape** — functional options vs. small config struct; what exactly the returned handle exposes (design-phase decision via `superpowers:brainstorming`).
2. **Root-mounted `/readyz` (OPS-3)** — mount readiness on the root router (making `/readyz` mean `/readyz`) now, or preserve `/api/readyz` and leave OPS-3 alone? Preserving is the FR-5.3 default; design may propose dual-mount if cheap and safe.
3. **snake_case mechanism confirmation** — emit-time hook (FR-3.1 default) vs. fleet call-site rename. Hook is strongly preferred (zero churn, drift-proof); design must validate collision/ECS-key edge cases.
4. **Non-REST services** — the sweep may surface services with no REST server (FR-5.4); decide per-case whether to add the server for readiness or document an exception.
5. **Post-114/116 producer shape** — if task-114 introduces a different canonical emit wrapper (outbox-aware), does `ProviderImpl` still deserve to exist as-is, or does the lib export the outbox-era shape? Re-measure at rebase (FR-6.2).

## 10. Acceptance Criteria

- [ ] `libs/atlas-kafka/producer` exports `Provider` + `ProviderImpl`; `find services -path '*/kafka/producer/producer.go'` returns 0 files; no service-local `Provider` type or alias remains; all call sites import the lib.
- [ ] `libs/atlas-service` exports `CreateLogger`; `find services -path '*/logger/init.go'` returns 0 files.
- [ ] `service.Bootstrap` exists with projection option; `grep -rl parseProjectionCatchupTimeout services` returns 0 files (helper lives once, in the lib); all 4 projection services use the option with unchanged runtime behavior.
- [ ] All 58 service `main.go`s call `Bootstrap`; none call `tracing.InitTracer` or define local shutting-down/readiness flags directly.
- [ ] Every REST-serving service mounts `/readyz`; hitting it on a booted service returns 200, and 503 after SIGTERM begins (verified on at least one previously-readiness-less service); the 4 existing services' `/api/readyz` behavior is unchanged.
- [ ] Emitted log records use snake_case keys — unit tests cover camelCase conversion, dotted-key passthrough, and collision behavior; a live log line from one migrated service shows `character_id`-style keys.
- [ ] Task branch is rebased on main **after** task-114 and task-116 merge; producer/main.go shapes re-measured post-rebase per FR-6.2.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in all changed modules; `docker buildx bake all-go-services` clean; `tools/redis-key-guard.sh` clean.
- [ ] `docs/architectural-improvements.md` updated (DUP-1/2/3 ✓, CP-9 ✓, OPS-2 endpoint-half noted); observability doc documents snake_case convention.
