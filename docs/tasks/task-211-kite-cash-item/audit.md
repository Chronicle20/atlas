# Plan Audit — task-211-kite-cash-item

**Plan Path:** docs/tasks/task-211-kite-cash-item/plan.md
**Audit Date:** 2026-08-10
**Branch:** task-211-kite-cash-item
**Base Branch:** main (merge-base `eca47150f`)
**Diff reviewed:** `eca47150f..e6e8f7a7f` (23 commits, 90 files, +10051/-66)

## Executive Summary

All 17 plan tasks are implemented and evidenced in the diff, with a clean 1:1 (plus a few in-flight fix commits) mapping between plan tasks and the 23 commits on the branch. Every module (`libs/atlas-packet`, `services/atlas-kites`, `services/atlas-channel`, `services/atlas-tenants`) builds, vets, and passes `go test -race ./...` with zero failures across 28+ subtests in atlas-kites alone. All three required `docker buildx bake` targets succeed. All eight repo guards relevant to this branch (service-registration, redis-key, goroutine, template-opcode-order, template-duplicate-binding, template-movement-types, packet-audit matrix/--check, and lint.sh in fix-mode terms) pass; `tools/lint.sh --check` reports a pre-existing, environment-only false-fail (`node v24, need v22`) unrelated to this branch's Go/TS changes, consistent with the documented project gotcha. All seven silent-failure traps enumerated in context.md §6 are closed with direct evidence. The two sanctioned plan deviations (three-value `Registry.Get`, and Task 17 Step 7 code-review deferred to this audit) are present exactly as described and are not gaps. One process-only gap: `plan.md`'s 117 step checkboxes are still all unchecked (`- [ ]`) despite the work being genuinely done — a documentation/bookkeeping omission, not an implementation gap.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Rename `FieldKiteSpawn.kiteType` → `y` | DONE | `libs/atlas-packet/field/clientbound/kite_spawn.go` struct/codec renamed to `y`; `kite_v48_test.go` comment/label updated; four audit JSON `IDAComment` rows updated (`gms_v83/87/95`, `jms_v185`); matrix regen is a byte-for-byte no-op (`git diff --exit-code docs/packets/audits/status.json docs/packets/audits/STATUS.md` exit 0); `go test ./field/clientbound/... -run TestKiteSpawn` passes. Commit `bf70eda24`. |
| 2 | Rename kite destroy animation constants | DONE | `kite_destroy.go` now exports `KiteDestroyAnimated`/`KiteDestroySilent`; `grep -rn "KiteDestroyAnimationType1\|2"` returns no hits repo-wide; `kite_destroy_test.go` uses `KiteDestroySilent`. Commit `0c2153dcb`. |
| 3 | New serverbound codec `ItemUseKite` | DONE | `libs/atlas-packet/cash/serverbound/item_use_kite.go` (66 lines) + `item_use_kite_test.go` (84 lines, no `packet-audit:verify` markers per plan's explicit instruction). `go test ./cash/serverbound/... -run ItemUseKite` passes for all variants + byte-pinned leading/trailing `updateTime` cases. Commit `ee21ccf96`. |
| 4 | Bind three kite writers in all ten tenant templates | DONE | `template_gms_{48,61,72,79,83,84,87,92,95}_1.json` + `template_jms_185_1.json` each gained 24 lines (3 writer bindings: `SpawnKite`, `SpawnKiteError`, `DestroyKite`, each with `fname`), verified at sorted opCode position in `template_gms_95_1.json` (`0x145`/`0x146`/`0x147` slotted before existing `0x148`). `gms_v12` untouched (10 of 11 templates, per design Q5). `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` both pass (22 arrays, no dup bindings). Commit `a1a3a1665`. |
| 5 | `kite-configs` tenant configuration resource | DONE | `services/atlas-tenants/atlas.com/tenants/configuration/{resource,rest,processor,provider}.go` add `GetKiteConfigHandler`/`CreateKiteConfigHandler`/`UpdateKiteConfigHandler`/`DeleteKiteConfigHandler` at `/tenants/{tenantId}/configurations/kite-configs` (resource.go:1395-1398) — GET/POST/PATCH/DELETE only, no `/seed`, no `/{id}`, matching the rankings pattern per context.md §5. `go test ./configuration/...` passes. Commit `6c36d64ea`. |
| 6 | Scaffold `atlas-kites` module | DONE | `services/atlas-kites/atlas.com/kites/go.mod`, `main.go`, package skeleton, kafka message contracts created. Commit `8e97d885f`. |
| 7 | `atlas-kites` domain: model, builder, two registries | DONE | `kite/{model,builder,registry}.go`, `character/{model,registry}.go` created. `kite/registry.go:93-103` implements `Get(ctx, characterId) (Model, bool, error)` — the sanctioned 3-value deviation (see below). `character/registry.go` keys on `MapKey{Tenant, Field}` where `Field` carries the instance (not instance-blind — trap #5 closed). Commits `516d3a014`, `cf81971d0` (fix: propagate real Redis errors). `go test ./kite/... ./character/...` pass (registry_test.go × 2, 254+124 lines). |
| 8 | Placement-policy configuration | DONE | `configuration/{model,registry,requests,rest}.go`. `configuration/model.go:50` hard-codes `maxMessageLength: 182` with the exact CUIHope derivation comment matching design Q4. `go test ./configuration/...` passes. Commit `446ea2f1f`. |
| 9 | `atlas-kites` processor and producer | DONE | `kite/processor.go` (328 lines) implements `Create`/`CreateAndEmit` with ordered checks (map-blocked → message-length → one-per-character, all lock-free; per-map cap under `AcquireFieldLock`), `refuse()` emits `CREATION_FAILED` only (never `CREATED` — FR-3.5), `Destroy`/`DestroyAndEmit` read the kite's field **before** removal (destroy-ordering trap #6 closed, documented at processor.go:222-224). Commits `2e758f908`, `67ccd8320` (fix: propagate per-map filter errors, drop test-only config singleton). `go test ./kite/...` (processor_test.go, 242 lines) passes. |
| 10 | `atlas-kites` REST resource | DONE | `kite/resource.go` (106 lines) + `resource_paginate_test.go` (195 lines) — paginated in-map list per the chalkboards precedent. Commit `33a7d8540`. |
| 11 | `atlas-kites` consumers and service wiring | DONE | `kafka/consumer/{consumer,kite,character}` created; `character/consumer.go` handles login/logout/map-changed/channel-changed, calling `DestroyAndEmit` on logout/map-change/channel-change (FR-6.1/6.2) with no TTL anywhere in the service (`grep -rn "TTL\|Expire"` → no hits, satisfying FR-6.3). Commits `7f482472d`, `1debf7e10` (fix: `producertest.InstallCapturing`). `go test ./kafka/consumer/character/...` (183 lines) passes. |
| 12 | Register `atlas-kites` in every hand-maintained list | DONE | `.github/config/services.json` (+8), `docker-bake.hcl` (+1), `go.work` (+1), `deploy/k8s/base/{atlas-kites.yaml,kustomization.yaml,env-configmap.yaml}`, both overlays' `kustomization.yaml`/patches (image pin + topic literals + `ATLAS_ENV`), `deploy/shared/routes.conf` + `deploy/k8s/base/routes.conf.template.generated`. `tools/service-registration-guard.sh` passes clean. Commit `7132cbaaf`. |
| 13 | `atlas-channel` kite client package | DONE | `channel/kite/{builder,processor,producer,requests,rest}.go` created, `model.go` rewritten (56 lines changed) dropping `ft`/`Type()`, adding `characterId`/`createdAt`; zero pre-existing importers confirmed free rewrite. `processor_drain_test.go` (129 lines) passes. Commit `929da0550`. |
| 14 | `atlas-channel` type-18 handler arm | DONE | `character_cash_item_use.go` +31 lines: `CashSlotItemTypeKite = CashSlotItemType(18)` added beside `CashSlotItemTypeChalkboard`; arm decodes `ItemUseKite`, resolves character via `character2.NewProcessor(...).GetById()`, calls `kite.NewProcessor(...).AttemptUse(...)` with **no** `saga.DestroyAsset` and **no** `EnableActions` call (trap #7 closed, documented inline at the arm). `character_cash_item_use_kite_test.go` (150 lines, real end-to-end wire-format test using the package's actual fixtures rather than the plan's uncompilable sketch — sanctioned per task brief) proves exactly one `CREATE` command with server-derived `x`/`y`/`name`, and that no other topic is touched (FR-4.1 verified). `go test ./socket/handler/... -run Kite -race -v` → PASS on both tests. Commit `a270ed0b7`. |
| 15 | `atlas-channel` kite status consumer | DONE | `kafka/consumer/kite/consumer.go` (118 lines) implements `handleCreatedEvent`/`handleDestroyedEvent`/`handleCreationFailedEvent`, each gated by `sc.Is(...)`; destroy uses `fieldcb.KiteDestroyAnimated` (not `Silent`) per design; `CREATION_FAILED` is targeted via `session.NewProcessor(...).IfPresentByCharacterId`, not a map broadcast, matching plan intent. `main.go` wires `kiteconsumer.InitConsumers`/`InitHandlers`. FR-7.6 (`produceWriters()` already correct) verified, no change made. No dedicated unit test file exists for this consumer — consistent with the sibling `kafka/consumer/chalkboard/` package, which also has none; not a plan deviation. Commit `a5db569bc`. |
| 16 | `atlas-channel` map-entry replay | DONE | `kafka/consumer/map/consumer.go` +24 lines: a `routine.Go` block calling `kite.NewProcessor(...).ForEachInMap(...)`, and `spawnKitesForSession` closing only over `s`/`wp`/`ctx`/`l`, building a fresh `KiteSpawn` per model — confirmed by direct read that no shared mutable state is captured (trap #4 closed). Commit `2e1034b8b`. |
| 17 | Full verification sweep | DONE (independently reproduced) | See Build & Test Results below — every gate this audit re-ran (build/vet/test per module, 3 docker bakes, 8 repo guards, packet-matrix no-op) passed independently. Commit `e6e8f7a7f` (lint-fix-only, per its message). Step 7 (`superpowers:requesting-code-review`) was explicitly not run inside Task 17 per the audit brief — this audit is that step, satisfying it retroactively. |

**Completion Rate:** 17/17 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 17 tasks have direct file/commit evidence of completion, and every module's build/vet/test gate is clean.

## Sanctioned Deviations (confirmed, not gaps)

1. **`kite.Registry.Get` is `(Model, bool, error)`.** Confirmed at `services/atlas-kites/atlas.com/kites/kite/registry.go:93-103`, with an inline comment explaining the Redis-error-vs-cache-miss distinction. All call sites (`kite/processor.go`, others) were written against the three-value form; no call site attempts the two-value plan-text signature. This is exactly the deviation the audit brief pre-authorized (fix commit `cf81971d0`).
2. **Task 17 Step 7 (code review) not run inside Task 17.** Confirmed — no prior `audit.md` existed before this run. This audit constitutes that step.
3. **Task 14's test snippet replaced with a working equivalent.** The plan's Task 14 sketch (plan.md:2746-2756) discards its own `message` parameter and never compiles against `ItemUseKite`'s private fields. The landed test (`character_cash_item_use_kite_test.go`) builds the real wire bytes via `response.Writer` primitives and asserts the identical claims (exactly one `CREATE`, server-derived position/name, message round-trip, no side-topic emission for FR-4.1). Judged DONE on substance, per the audit brief's instruction.

## Spec Coverage Cross-Check (PRD FR → Task)

All PRD requirements in the plan's Spec-coverage table were traced to concrete evidence:

| Requirement | Task | Verified |
|---|---|---|
| FR-1.1–1.3 serverbound decode | 3 | `item_use_kite.go`/test — YES |
| FR-2.1–2.3 rename + audit JSON + matrix no-op | 1 | matrix diff-exit=0 — YES |
| FR-3.1–3.3 domain model/registry/id counter | 7 | registry.go, IDGenerator — YES |
| FR-3.4/3.5 processor, no CREATED on refusal | 9 | `refuse()` only puts `CREATION_FAILED` — YES |
| FR-4.1 not consumed / FR-4.2 ownership | 14 | handler test + `cashItemInSlotFunc` gate — YES |
| FR-5.1–5.4 cap/uniqueness/policy/enforcement | 9 (policy in 8) | ordered checks in `Create` — YES |
| FR-6.1/6.2 owner leaves/logout, FR-6.3 no TTL | 11 | character consumer + no TTL grep — YES |
| FR-6.4 destroy animation | 2, 15 | `KiteDestroyAnimated` used, not `Silent` — YES |
| FR-7.1 handler arm | 14 | arm present — YES |
| FR-7.2/7.3 channel kite package, dead model rewrite, domain doc | 13 | model.go rewritten; `docs/domain.md` +11/-x — YES |
| FR-7.4 three consumers | 15 | Created/Destroyed/CreationFailed handlers — YES |
| FR-7.5 map-entry replay | 16 | replay pass + operator — YES |
| FR-7.6 `produceWriters()` verify-only | 15 | plan explicitly instructs no change; not modified — YES |
| FR-8.1 tenant config producer | 5 | `kite-configs` resource — YES |
| FR-8.1 consumer / FR-8.2 DOM-25 | 8 | `configuration/` package, no hard-coded wire values found in handler (opcodes resolve via templates) — YES |
| FR-9.1–9.4 template writers, v12 excluded, no new handler entry | 4 | 10 templates, `template_gms_12_1.json` untouched, no new `handlers` entries added (only `writers`) — YES |
| §5 REST surface | 10 | resource.go/rest.go — YES |
| §6 data model, no Postgres | 7 | Redis-only (`atlas.NewTenantRegistry`), no gorm import in atlas-kites — YES |
| §7 service registration | 12 | all hand-maintained lists updated, guard passes — YES |
| §8 non-functional gates | 17 | reproduced independently — YES |
| Q4 message bound 182 | 8, 9 | `maxMessageLength: 182` + `len(cmd.Message) > cfg.MaxMessageLength()` check — YES |
| Q6 per-instance cap | 7, 11 | `field.Model` (includes instance) used as the map/lock key throughout — YES |

No FR from the coverage table is unmet.

## Silent-Failure Trap Check (context.md §6)

| # | Trap | Status | Evidence |
|---|---|---|---|
| 1 | Template writer bindings, ten templates, sorted opCode, `fname` present | CLOSED | 10 templates (not v12) each +24 lines; `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` both exit 0; spot-checked `template_gms_95_1.json` shows correctly sorted `0x145/0x146/0x147` insertion with `fname` on each entry. |
| 2 | Kafka topic env keys in base AND both overlays (`behavior: replace`) | CLOSED | `COMMAND_TOPIC_KITE` and `EVENT_TOPIC_KITE_STATUS` present in `deploy/k8s/base/env-configmap.yaml` and re-listed as literals in both `deploy/k8s/overlays/main/kustomization.yaml` and `deploy/k8s/overlays/pr/kustomization.yaml`. |
| 3 | `images:` pin present in both overlays | CLOSED | `ghcr.io/chronicle20/atlas-kites/atlas-kites` present in both `overlays/main/kustomization.yaml:259` and `overlays/pr/kustomization.yaml:417`. |
| 4 | Parallel `ForEachInMap` replay operator captures nothing mutable | CLOSED | `spawnKitesForSession` in `kafka/consumer/map/consumer.go` closes only over `l`/`ctx`/`wp`/`s`, builds a fresh `KiteSpawn` per model — read directly, confirmed. |
| 5 | Instance threaded on every field construction in atlas-kites consumers | CLOSED | `character/consumer.go`'s all four handlers call `field.NewBuilder(...).SetInstance(e.Body.Instance).Build()` (or `OldInstance`/`TargetInstance`); `character/registry.go`'s `MapKey{Tenant, Field}` carries the full field including instance — not instance-blind like the chalkboards bug. |
| 6 | Destroy ordering: old field captured before destroy emit | CLOSED | `kite/processor.go:222-224` — `Destroy` reads the kite's own field off the stored `Model` via `p.r.Get` before `p.r.Remove`; `character/consumer.go`'s map-changed handler comment (lines 76-81) explains why handler-level `of`/`nf` capture order is irrelevant to correctness (Destroy doesn't take a field parameter at all). |
| 7 | No `EnableActions` on the kite handler arm | CLOSED | `character_cash_item_use.go`'s type-18 arm (added at Task 14) contains no `EnableActions` call; inline comment explicitly documents why (modal client dialog self-unlocks). `grep -n EnableActions` in the arm's block: no hits. |

All seven traps closed with direct evidence.

## Build & Test Results

| Service/Module | Build | Vet | Test (-race) | Docker Bake | Notes |
|---|---|---|---|---|---|
| libs/atlas-packet | PASS | PASS | PASS | n/a | `cash/{clientbound,serverbound}`, `field/clientbound` all `ok` |
| services/atlas-kites | PASS | PASS | PASS | PASS | 28 subtests, 0 failures |
| services/atlas-channel | PASS | PASS | PASS | PASS | kite/, socket/handler/, kafka/consumer/{kite,map}/ all PASS |
| services/atlas-tenants | PASS | PASS | PASS | PASS | configuration/, configuration/seed all `ok` |

| Repo Guard | Result |
|---|---|
| `tools/service-registration-guard.sh` | clean (exit 0) |
| `tools/redis-key-guard.sh` | clean (exit 0) |
| `tools/goroutine-guard.sh` | clean (exit 0) |
| `tools/template-opcode-order-guard.sh` | OK, 22 arrays (exit 0) |
| `tools/template-duplicate-binding-guard.sh` | OK, 22 arrays (exit 0) |
| `tools/template-movement-types-guard.sh` | OK, 54 handlers / 11 templates (exit 0) |
| `go run ./tools/packet-audit matrix` + `--check` | no-op, exit 0 both; `git diff --exit-code` on status.json/STATUS.md exit 0 |
| `tools/lint.sh --check` | 0 lint issues on all touched packages; **fails only** on `ui:node-version` (`node v24 found, need v22`) — a pre-existing environment gap (documented project memory: "lint.sh --check false-fails without nvm"), unrelated to this branch's changes. Not a task-211 defect. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Cosmetic, optional) Update `plan.md`'s 117 step checkboxes from `- [ ]` to `- [x]` to reflect actual completion — currently 0/117 are checked despite all 17 tasks being genuinely done. This is a documentation-bookkeeping gap only; it does not block merge.
2. (Environment, not code) Run `tools/lint.sh --check` under `nvm use 22` before merge if a fully clean gate run (rather than the reproduced 0-issues-on-Go-plus-known-node-version-false-fail) is required for the PR record.
