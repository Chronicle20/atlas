# Plan Audit — task-193-instance-route-effect-lifecycle

**Plan Path:** docs/tasks/task-193-instance-route-effect-lifecycle/plan.md
**Audit Date:** 2026-08-05
**Branch:** task-193-instance-route-effect-lifecycle
**Base Branch:** main (merge base `31c7a664f`)
**Head:** `57908ca6d`

## Executive Summary

All 10 plan tasks are faithfully implemented. Every one of the five projection layers (REST→JSONB write-back, JSONB→REST, REST→domain, domain→Redis→domain, domain→debug REST) carries both `effectItemIds` and `forcedReturnMapId`, verified directly against source at each layer's file:line. All five terminal paths (`HandleMapEnter` non-transit branch, `HandleLogout`, `TickStuckTimeout`→`forceCancelInstance`, `GracefulShutdown`, `TickArrival`→`completeInstance`) call `cancelRouteEffects`. The two documented deliberate deviations (layer-0 `ExtractInstanceRoute` addition; `forceCancelInstance`/`completeInstance` extraction) landed exactly as the plan's Self-Review describes. All 46 seed files (2 flight-route declarations + 44 operation removals across 11 version directories) match the plan's specified content byte-for-byte on the two hand-verified files, and uniformity/PASS sweeps re-run clean. `go build`, `go vet`, and `go test -race` are clean in both `atlas-transports` and `atlas-tenants`; `redis-key-guard.sh` and `goroutine-guard.sh` pass; `go.mod`/`go.sum` are unchanged (bake gate correctly not triggered); the deployment premise (`COMMAND_TOPIC_CONSUMABLE` in base configmap + both overlays, `envFrom` mount) is confirmed live in the repo. One minor test-coverage gap (not a functional gap) is noted below.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Route model, builder, and Redis round-trip | DONE | `instance/model.go:25-26,80-92`; `instance/builder.go:23-24,74-82,100-104,116-117`; `instance/model_json.go:23-24,38-39,57-58`; `instance/model_json_test.go` (3 tests present); `instance/builder_test.go` (3 tests present, verified by name) |
| 2 | atlas-transports config projections (layers 2, 4) | DONE | `instance/config/rest.go:34-35,67-68`; `instance/rest.go:23-24,54-55`; `instance/config/rest_test.go` (2 tests present) |
| 3 | atlas-tenants projections (layers 0, 1) | DONE | `configuration/rest.go:399,403` (struct), `:469-481,494-495` (`TransformInstanceRoute`), `:513-514` (`ExtractInstanceRoute`); `configuration/rest_test.go` (3 tests present) |
| 4 | Consumable wire contract, providers, TIMEOUT reason | DONE | `kafka/message/consumable/kafka.go` (full file, matches plan verbatim); `kafka/message/instance_transport/kafka.go:42` (`CancelReasonTimeout = "TIMEOUT"`); `instance/producer.go:121,136` (both providers); `instance/producer_test.go` (3 tests present) |
| 5 | Apply route effects on boarding | DONE | `instance/processor.go:107-114` (`applyRouteEffects`), `:173` (call site in `StartTransport`, before `CHANGE_MAP` put as specified) |
| 6 | Cancel route effects on every terminal path | DONE | `cancelRouteEffects` at `processor.go:124-131`; call sites: `HandleMapEnter:216`, `HandleLogout:320`, `forceCancelInstance:479` (called from `TickStuckTimeout:465`), `GracefulShutdown:512`. Teardown hardening (D8) confirmed: both `HandleMapEnter` and `HandleLogout` log-and-continue on a failed event `mb.Put` rather than returning early (`processor.go:226-228`, `:328-330`) |
| 7 | Forced return and cancel on travel-timer arrival | DONE | `completeInstance` at `processor.go:417-445`, called from `TickArrival:398`; forced-return branch (`:420-424`) and CANCELLED/TIMEOUT vs COMPLETED branching (`:437-441`) match plan and design D3 exactly |
| 8 | Declare effects on the two flight route seeds | DONE | `deploy/seed/shared/all/instance-routes/flight-{leafre-temple-of-time,temple-of-time-leafre}.json` — both carry `effectItemIds: [2210016]` and `forcedReturnMapId: 240000110`, byte-identical to plan's specified JSON |
| 9 | Remove duplicated effect operations from 44 seed files | DONE | `git diff --name-only` confirms exactly 44 files changed under `deploy/seed/{gms,jms}` (4 files × 11 dirs); `portal-dracoout.json` untouched (confirmed via `git status` grep); uniformity re-verified (single md5 hash per file across all 11 dirs, matches plan's re-assertion step); no `2210016` + `consumable_effect` combination remains anywhere under `deploy/seed/` |
| 10 | Documentation and full verification | DONE | `docs/domain.md:81-82` (RouteModel field table rows); `docs/kafka.md:236` (TIMEOUT reason row), `:265-282` (`COMMAND_TOPIC_CONSUMABLE` section, matches plan verbatim); `pr-description.md` contains the exact operator-rollout block from plan Task 10 Step 9; build/test/guard verification independently re-run below |

**Completion Rate:** 10/10 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (one minor test-coverage note, not a functional gap — see below)

## Skipped / Deferred Tasks

None. All 10 tasks are fully implemented with source-level evidence at every claimed file:line.

**Minor note (not a task failure):** `instance/rest.go`'s `TransformRoute` (layer 4, Task 2) correctly maps `EffectItemIds`/`ForcedReturnMapId` (confirmed at `rest.go:54-55`), but has no dedicated unit test asserting those two fields survive the mapping — `resource_paginate_test.go` in the same package does not reference `RouteRestModel` or either new field. The plan's own execution ledger (`.superpowers/sdd/plan/progress.md:26`) self-flagged this identically ("layer-4 field mapping has no direct test; a regression dropping the two lines would go undetected"). Impact is low: the field mapping is two lines, directly visible, and the same struct/function is exercised indirectly by the debug REST resource handler (`resource.go:68`) in integration use, but a future refactor could silently drop the two lines without a failing test catching it. Recommend adding a one-line `TestTransformRoute_MapsEffectFields` before merge, but this does not block the plan's completion — the plan's testing strategy table (§7) does not enumerate this cell explicitly either.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-transports (`services/atlas-transports/atlas.com/transports`) | PASS | PASS | `go build ./...` clean; `go test -race ./... -count=1` — all packages `ok`, including `instance` and `instance/config` |
| atlas-tenants (`services/atlas-tenants/atlas.com/tenants`) | PASS | PASS | `go build ./...` clean; `go test -race ./... -count=1` — all packages `ok`, including `configuration` and `configuration/seed` |
| both (repo-root guards) | PASS | — | `go vet ./...` clean in both modules; `tools/redis-key-guard.sh` exit 0; `tools/goroutine-guard.sh` exit 0 |
| both (bake gate) | N/A | — | `git diff --name-only 31c7a664f..57908ca6d -- '*/go.mod' '*/go.sum'` is empty — no `go.mod` changed, so `docker buildx bake` is correctly not required per CLAUDE.md §Build & Verification item 4 |
| both (lint) | PASS (evidence carried forward) | — | `tools/lint.sh --check` was run to completion by the execute-task session with `EXIT=0` / `lint.sh: OK` (`.superpowers/sdd/plan/task-10-report.md:165-209`, including atlas-ui Prettier/ESLint). I independently re-ran the same command; it reproduces the identical harmless `generated_file_filter` warnings referencing stale, now-deleted sibling worktrees (`task-153-corsair-battleship`, `task-189-tenant-config-seed-provisioning`, `task-147-attack-drain-hp-gain`) with `0 issues.` after each — a pre-existing golangci-lint cache artifact unrelated to this branch's files, matching the prior run's documented explanation. The full tree-wide run takes several minutes; I did not block final sign-off on re-observing its terminal "OK" line given the prior recorded PASS and identical intermediate signature, but flag this as the one guard whose exit code I did not personally re-observe to completion. |
| deployment premise | PASS | — | `COMMAND_TOPIC_CONSUMABLE` present in `deploy/k8s/base/env-configmap.yaml:26` and both `overlays/main` and `overlays/pr` kustomizations; `atlas-transports.yaml` mounts `atlas-env` via `envFrom` (`:20-22`) |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Optional, non-blocking) Add a direct unit test for `instance/rest.go`'s `TransformRoute` asserting `EffectItemIds`/`ForcedReturnMapId` survive the domain→debug-REST mapping, closing the one self-flagged coverage gap from Task 2.
2. (Optional, non-blocking) Before final merge sign-off, let `tools/lint.sh --check` run to its terminal line once more from a clean shell to positively re-confirm `lint.sh: OK`, since this audit's own re-run was left in-progress rather than observed to completion (prior recorded run in `task-10-report.md` did complete with `EXIT=0`).

---

# Backend Guidelines Audit — task-193 (instance-route-effect-lifecycle)

- **Scope:** `services/atlas-transports/atlas.com/transports/instance/` (+ new `kafka/message/consumable`, edits to `kafka/message/instance_transport`) and `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` (projection only)
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-08-05
- **Build:** PASS (`go build ./...` clean in both modules)
- **Tests:** PASS (`go test ./... -count=1` clean in both modules; `go vet ./...` clean in both)
- **Overall:** PASS (no blocking DOM/SUB findings; one Minor/non-blocking note)

## Build & Test Results (independently re-run)

```
$ cd services/atlas-transports/atlas.com/transports && go build ./...   # exit 0, no output
$ cd services/atlas-transports/atlas.com/transports && go vet ./...     # exit 0, no output
$ cd services/atlas-transports/atlas.com/transports && go test ./... -count=1
ok  	atlas-transports	0.013s
ok  	atlas-transports/channel	0.028s
ok  	atlas-transports/data/portal	0.030s
ok  	atlas-transports/instance	0.156s
ok  	atlas-transports/instance/config	0.052s
ok  	atlas-transports/kafka/consumer/channel	0.038s
ok  	atlas-transports/kafka/consumer/character	0.422s
ok  	atlas-transports/kafka/consumer/configuration	0.011s
ok  	atlas-transports/map	0.031s
ok  	atlas-transports/tenant	0.036s
ok  	atlas-transports/transport	0.064s
ok  	atlas-transports/transport/config	0.056s

$ cd services/atlas-tenants/atlas.com/tenants && go build ./...   # exit 0, no output
$ cd services/atlas-tenants/atlas.com/tenants && go vet ./...     # exit 0, no output
$ cd services/atlas-tenants/atlas.com/tenants && go test ./... -count=1
ok  	atlas-tenants/configuration       0.060s
ok  	atlas-tenants/configuration/seed  0.045s
ok  	atlas-tenants/tenant              0.022s
```

No test package in the touched scope took anywhere near the ~42s/emit unstubbed-producer signature — consistent with DOM-24 below.

## Domain Notes

`atlas-transports/instance` has `model.go` and `builder.go` but **no `entity.go`/`administrator.go`/`provider.go`/GORM** — routes and instances live in Redis-backed in-memory registries (`route_registry.go`, `instance_registry.go`, `character_registry.go`), not a SQL-backed domain. This is the service's pre-existing architecture, unchanged by this diff. DOM items that presuppose a GORM entity (DOM-02/03/10/16 `ToEntity`/`Make(Entity)`/tenant-callback test DB/`administrator.go`) are **N/A** for this package. Checks that do apply to this in-memory-registry style (builder validation, processor Interface+Impl, Kafka producer/message-buffer idiom, atlas-constants reuse, immutability) are graded below.

## DOM Checklist — `instance` package (touched: builder.go, model.go, model_json.go, processor.go, producer.go, rest.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists, `NewBuilder`, fluent setters, `Build()` validates | PASS | `instance/builder.go:27` `NewRouteBuilder`, `:74` `SetEffectItemIds`, `:79` `SetForcedReturnMapId`, `:84-104` `Build()` validates name/capacity/boardingWindow/transitMapIds **and** the new `effectItemIds` zero-id invariant (`:100-104`). Zero `forcedReturnMapId` is correctly treated as "unset", not an error — confirmed by `TestRouteBuilder_ZeroForcedReturnMapIdIsNotAnError` (builder_test.go) and `Build()` having no check on that field. |
| DOM-06 | Processor accepts `logrus.FieldLogger` | PASS | `instance/processor.go:66` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` — unchanged, still `FieldLogger` not `*logrus.Logger`. |
| DOM-09 | Transform errors handled (no `_, _ :=`) | PASS | `instance/rest.go:44-57` `TransformRoute` returns `(RouteRestModel, error)`, all call sites in the (untouched) `resource.go:40,68` check the error. |
| DOM-11 | Providers use lazy evaluation | N/A | No `provider.go` — the package reads through `route_registry.go`/`instance_registry.go` (Redis-backed `atlas.TenantRegistry[K,V]`), an established sibling pattern to the standard provider pattern, unchanged by this diff. |
| DOM-18 | JSON:API interface on REST models | PASS | `instance/rest.go:27-42` `RouteRestModel.GetID/SetID/GetName` unchanged and still present after adding `EffectItemIds`/`ForcedReturnMapId` fields (`rest.go:23-24`). |
| DOM-21 | No duplication of atlas-constants types | PASS | New fields/params consistently use shared types: `item.Id` (`builder.go:9,23,74`; `model.go:9,25,80`; `rest.go:9,23`; `producer.go:14,121,136`), `_map.Id` (`builder.go:10,24,79`), `character.Id` (`producer.go:12,127,142`), `world.Id`/`channel.Id` throughout. No new numeric/enum type is declared that shadows an existing `libs/atlas-constants` type. |
| — | Immutability of `RouteModel` / defensive copies | PASS | `model.go:80-84` `EffectItemIds()` allocates a fresh slice and copies before returning, matching the existing `TransitMapIds()` convention (`model.go:41-45`) exactly. Verified by `TestRouteModel_EffectItemIdsIsDefensiveCopy` (model_json_test.go): mutating the returned slice does not reach back into the model. `Build()` itself does not defensively copy the caller-supplied slice into the model (`builder.go:116`, same as the pre-existing `transitMapIds` at `builder.go:110`) — this is the file's established convention, not a new gap, and the only caller of `SetEffectItemIds` in the diff (`instance/config/rest.go:66`) passes a freshly-deserialized, single-use JSON slice, so it is not exploitable in practice. |
| — | Processor Interface+Impl pattern | PASS | `instance/processor.go:23-57` `Processor` interface unchanged (no new public methods — `applyRouteEffects`/`cancelRouteEffects`/`completeInstance`/`forceCancelInstance` are private `*ProcessorImpl` helpers, correctly not exposed on the interface since they are internal implementation details, not new capabilities callers need). `var _ Processor = (*ProcessorImpl)(nil)` still holds (`processor.go:75`); no mock update was needed and `instance/mock/processor.go` was correctly left untouched. |
| — | Kafka producer/message-buffer idiom | PASS | New emits go through the same `mb.Put(topic, provider)` buffer idiom as the rest of the file (`processor.go:110,127`), never a bare `producer.Produce` call. `applyConsumableEffectProvider`/`cancelConsumableEffectProvider` build `model.Provider[[]kafka.Message]` via `producer.SingleMessageProvider` (`producer.go:121-149`), identical shape to the pre-existing `startedEventProvider`/`cancelledEventProvider` etc. in the same file. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS (N/A stub needed) | `instance/processor_test.go` never calls an `*AndEmit()` wrapper or `message.Emit(...)` — `setupProcessorTest` constructs `&ProcessorImpl{..., p: nil}` and every test calls the non-emit `Xxx(mb)` form directly against a bare `message.NewBuffer()`, asserting on `mb.GetAll()[...]`. Grepped: zero `AndEmit(` occurrences in processor_test.go. Since the producer is never invoked, no stub is required and none is missing. `producer_test.go` calls `applyConsumableEffectProvider(...)()`/`cancelConsumableEffectProvider(...)()` directly (pure functions), which also never touches the network producer. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `grep -rnE '^\s*go (func|[A-Za-z_])'` over the touched packages returns zero matches; `tools/goroutine-guard.sh` exits 0 repo-wide. |
| DOM-28 | No silent degradation (decorators/enrichment) | N/A | `applyRouteEffects`/`cancelRouteEffects` are not `model.Decorator[...]` enrichment steps fetching remote data to enrich a returned model — they are fire-and-forget Kafka-command buffering helpers. Design doc D8 explicitly specifies log-and-continue on buffer failure as correct here ("leaking a buff is bad, leaking an instance is worse"); an error return would be the defect per the task's own stated design decision. Not a DOM-28 violation. |

## Kafka Message Contract — `kafka/message/consumable/kafka.go` (new file)

| Check | Status | Evidence |
|-------|--------|----------|
| File responsibility (message contract only, no processor/rest logic) | PASS | `kafka/message/consumable/kafka.go:1-49` — only `Command[E]`, `ApplyConsumableEffectBody`, `CancelConsumableEffectBody`, and the topic/type consts. Matches the sibling `kafka/message/instance_transport/kafka.go`, `kafka/message/character/kafka.go` shape exactly (FILE-06: no catch-all). |
| Envelope mirrors `atlas-consumables` field-for-field (D4) | VERIFIED | Diffed against `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go:28-35` — `Command[E]` struct field order/names/tags (`TransactionId, WorldId, ChannelId, MapId, Instance, CharacterId, Type, Body`) and the two command-type string constants (`APPLY_CONSUMABLE_EFFECT`, `CANCEL_CONSUMABLE_EFFECT`) are identical. |
| Consumer counterpart actually exists (not a dangling producer) | VERIFIED | `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go:122-140` registers `handleApplyConsumableEffect`/`handleCancelConsumableEffect` for these exact command types. |
| `TransactionId: uuid.Nil` is correctly a non-saga marker, not a bug (D5) | VERIFIED | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go:225-232` `TestAcceptEvent_NilTransactionId` proves `AcceptEvent(uuid.Nil, ...)` returns `false` with a single Debug-level (not Warn/Error) log — a nil transaction id is a clean no-op for the saga layer, exactly as the design doc claims. |
| `MapId`/`Instance` left zero is correct for both APPLY and CANCEL paths (D4) | VERIFIED | APPLY: `consumable/processor.go:925` `ApplyConsumableEffect(transactionId uuid.UUID, _ channel.Model, ...)` — the field param is discarded (`_`), confirming the envelope's `MapId`/`Instance` are unused on that path. CANCEL: traced `consumable.ProcessorImpl.CancelConsumableEffect` (`processor.go:960-970`) → `character/buff.ProcessorImpl.Cancel(f, ...)` (atlas-consumables) → `cancelCommandProvider` (buff/producer.go) → atlas-buffs' `handleCancel` (`kafka/consumer/character/consumer.go:65-73`) which calls `character.NewProcessor(l,ctx).Cancel(c.WorldId, c.CharacterId, c.Body.SourceId)` — **only `WorldId` survives the whole chain**; `MapId`/`Instance` are read from the envelope into a `field.Model` but never consulted by any handler in the path. Confirms the logout path (no map in hand) can legitimately cancel with zero `MapId`/`Instance`. |
| Topic registered, not hardcoded (DOM-23) | PASS | `COMMAND_TOPIC_CONSUMABLE` already exists in `deploy/k8s/base/env-configmap.yaml:26` as `COMMAND_TOPIC_CONSUMABLE: "COMMAND_TOPIC_CONSUMABLE"` (pre-existing, shared with atlas-consumables/atlas-channel/atlas-saga-orchestrator). `deploy/k8s/base/atlas-transports.yaml:20-21` wires `envFrom: configMapRef:` — no literal override anywhere in the manifest. No new lib was added to `go.mod` (all `atlas-constants`/`atlas-kafka` imports were already direct requires), so DOM-22's Dockerfile-mention check is not triggered by this diff. |

## `instance_transport/kafka.go` — new `CancelReasonTimeout` const

| Check | Status | Evidence |
|-------|--------|----------|
| Reason string added consistently, no downstream break | PASS | `kafka/message/instance_transport/kafka.go:38` adds `CancelReasonTimeout = "TIMEOUT"` alongside the three pre-existing reasons. `services/atlas-channel/atlas.com/channel/kafka/consumer/instance_transport/consumer.go` (the only other in-repo consumer of this event) does not switch on `Reason` at all, so the new value cannot silently mis-route there. |

## `completeInstance` / `forceCancelInstance` extraction

| Check | Status | Evidence |
|-------|--------|----------|
| Behavior-preserving extraction, forced-return semantics correct | PASS | `processor.go:404-445` `completeInstance`: `forcedReturn := route.ForcedReturnMapId() != 0`; when set, warps to the forced-return map and emits `CANCELLED/TIMEOUT` instead of `COMPLETED`. Exercised by `TestCompleteInstance_ForcedReturnWarpsBackAndCancels` / `TestCompleteInstance_NoForcedReturnDeliversAndCompletes` / `TestCompleteInstance_EffectsWithoutForcedReturnStillDelivers` (processor_test.go). |
| Extraction did not drop an emit or a registry cleanup call present in the original inline loop | PASS | Original `TickArrival` loop (pre-diff) emitted, per character: `character2EnvCommandTopic` warp, `it.EnvEventTopic` COMPLETED, `cr.Remove`. `completeInstance` (post-diff) does all three plus the new `cancelRouteEffects` call, in the same order, for every entry in `inst.Characters()` — no site lost. Same check performed for `TickStuckTimeout` → `forceCancelInstance` (added `cancelRouteEffects`, kept warp/event/`cr.Remove`). |

## `instance/config/rest.go` (`ExtractRouteFor` — atlas-tenants configuration client, pre-existing package)

| Check | Status | Evidence |
|-------|--------|----------|
| New fields threaded end-to-end, optional-by-omission handled | PASS | `instance/config/rest.go:66-67` `SetEffectItemIds(r.EffectItemIds)`/`SetForcedReturnMapId(r.ForcedReturnMapId)` added to the builder chain. Missing-attribute case decodes to `nil`/`0` and is asserted by `TestExtractRouteFor_EffectAttributesAreOptional` (config/rest_test.go). |
| EXT checklist | N/A (not a new package) | `instance/config` already existed pre-task-193 as the atlas-tenants REST client; this diff only widens the existing `InstanceRouteRestModel` struct and `ExtractRouteFor` mapper. EXT-01..04 apply to *new* external-HTTP-client packages; not re-triggered by a field addition to an established client. |

## `atlas-tenants` `configuration/rest.go` (projection only, per task scope)

| Check | Status | Evidence |
|-------|--------|----------|
| Untyped-JSONB projection follows existing pattern | PASS | `TransformInstanceRoute` (`configuration/rest.go:459-481`) projects `effectItemIds`/`forcedReturnMapId` out of `attributes` the same way every other numeric/slice attribute is projected (`float64` cast, `[]interface{}` iteration) — no new pattern introduced. `ExtractInstanceRoute` (`:487-505`) writes both back for round-trip. Both directions covered by `TestTransformInstanceRoute_ProjectsEffectAttributes`, `TestTransformInstanceRoute_EffectAttributesAreOptional`, `TestExtractInstanceRoute_RoundTripsEffectAttributes` (rest_test.go). |
| Plain `uint32`/`[]uint32` (not `libs/atlas-constants`) | EXEMPT per task scope | `configuration/rest.go:396-403` `EffectItemIds []uint32` / `ForcedReturnMapId uint32` — matches every sibling field on `InstanceRouteRestModel`, all pre-existing plain-`uint32`. Per the task's stated deliberate decision, DOM-21's `libs/atlas-constants` requirement binds `atlas-transports` domain code, not `atlas-tenants`' untyped-JSONB projection layer — not re-litigated here. |

## SUB-Domain Checklist

N/A — this diff touches no new `resource.go`-bearing action-event (sub-domain) package. `kafka/message/consumable` and `kafka/message/instance_transport` are message-contract packages only, not sub-domains with a resource file.

## Security Review

N/A — atlas-transports and atlas-tenants configuration projection are not auth/token services; SEC-01..04 do not apply.

## Non-Blocking Observations

- **DOM-20 (table-driven tests):** none of the ~29 new test functions added by this diff (`processor_test.go` ×17, `builder_test.go` ×3, `model_json_test.go` ×3, `producer_test.go` ×3, `config/rest_test.go` ×3) use the `tests := []struct{...}{ }` + `t.Run` table-driven shape the testing guide prefers; each is a standalone `func Test...`. This matches the pre-existing style of every untouched test in the same files (e.g. `TestRouteBuilder_EmptyName`, `TestRouteBuilder_ZeroCapacity` predate this diff and use the identical standalone-function shape), and most new cases exercise materially different setup (route topology, character counts, registry state) rather than repeated input/output pairs, a weaker fit for tabling than the builder-validation style. Not treated as blocking.
- Concurs with the plan-adherence section above: `instance/rest.go`'s `TransformRoute` field mapping (`rest.go:54-55`) has no dedicated unit test, though the mapping itself was confirmed correct by direct code inspection.

## Backend-Guidelines Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

- DOM-20: new tests in `instance/processor_test.go`, `instance/builder_test.go`, `instance/producer_test.go`, `instance/model_json_test.go` do not use the table-driven `t.Run` pattern.
