# Plan Audit — task-212-monster-catch-items

**Plan Path:** docs/tasks/task-212-monster-catch-items/plan.md
**Audit Date:** 2026-08-10
**Branch:** task-212-monster-catch-items
**Base Branch:** main

## Executive Summary

All 13 plan tasks have corresponding commits on the branch (`b531cb3db`..`48cb55054`) with matching file-level changes, tests, and documentation. The plan.md checkboxes (`- [ ]`) were never checked off during execution, but this is cosmetic — every task's file structure, interfaces, and acceptance behavior described in the plan is present in the diff, verified against actual commit contents rather than the checkbox state. All six affected Go modules build, vet, and test clean; all seven mandatory guard scripts pass. No TODOs, stubs, or silent-drop paths were found; three legitimate mid-flight plan amendments (documented in commit messages, not silent deviations) tightened the catch resolution ladder's failure-handling semantics beyond what the original brief specified.

**Update (post-audit, same session):** `docker buildx bake atlas-monsters atlas-consumables atlas-channel` was run from the worktree root after this audit's initial pass (which had flagged it as not-run) and all three targets built and exported cleanly (exit 0). `go run ./tools/packet-audit matrix --check`, `tools/lint.sh --check` (with `nvm use 22` sourced — the atlas-ui half needs Node 22, not a false-fail), `tools/skill-job-id-guard.sh`, and `tools/buff-duration-guard.sh` were also re-run against current HEAD (post-Task-13 commit `48cb55054`) and all passed clean. This resolves the audit's blocking recommendation below — see the Task 13 report at `.superpowers/sdd/plan/task-13-report.md` for the full gate table with exit codes.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `UseCatchItem` serverbound codec | DONE | `6f9b52da4`; `libs/atlas-packet/monster/serverbound/use_catch_item.go` (73 lines) + test (55 lines), no version gate, matches plan's field layout (`updateTime`, `slot`, `itemId`, `monsterUniqueId`) |
| 2 | Registry entries + packet-audit fname linkage | DONE | `fa21e9873`; `docs/packets/registry/gms_v{48,61,72,79}.yaml` + `tools/packet-audit/cmd/run.go` `candidatesFromFName` case added; v61 opcode correction documented and justified in commit body (a plan-compliant, sourced fix, not scope creep) |
| 3 | Route `USE_CATCH_ITEM` in all ten seed templates | DONE | `5c6c61b1e`; all ten `template_*_1.json` files gained the handler entry at the plan's specified opCode; `socket/corpus_test.go` count bumped 3075→3085 |
| 4 | `CMobPool::OnMobPacket` uniqueId prefix unconditional | DONE | `389c6b3ea`; `catch_monster.go`, `inc_mob_charge_count.go`, `monster_special_effect_by_skill.go` — `legacyMobPoolPrefix` deleted, fixtures updated with v83/v95/v92/v48 pins |
| 5 | `CatchMonsterWithItem` gains `uniqueId`; v48 drops result byte | DONE | `c7ea88432`; codec + writer (`socket/writer/catch_monster_with_item.go`) updated with `MajorAtLeast`/`v48CatchByItemNoResult` gate; plan's own byte-fixture arithmetic typo caught and corrected (documented in commit) |
| 6 | Writer routes for v48/v92 clientbound catch cells | DONE | `d73ce0862`; `template_gms_48_1.json` (CatchMonster/CatchMonsterWithItem), `template_gms_92_1.json` (BridleMobCatchFail/CatchMonster/CatchMonsterWithItem) — matches design §8's v92 gap correction; registry entries added for v48 172/173 |
| 7 | Promote coverage-matrix cells | DONE | `bfa98a0ac`; all USE_CATCH_ITEM (10 versions) + CATCH_MONSTER/CATCH_MONSTER_WITH_ITEM/BRIDLE_MOB_CATCH_FAIL cells promoted with evidence + audit reports; no "not promoted, and why" note needed in context.md since all cells were successfully promoted (grep confirms none present) |
| 8 | Atomic delete-and-claim in `libs/atlas-redis`, `ClaimMonster` | DONE | `b0c7fd533`; `libs/atlas-redis/registry.go:73` `RemoveExisting`, `services/atlas-monsters/.../monster/registry.go:577` `ClaimMonster`, concurrency test proving exactly-one-winner |
| 9 | atlas-monsters catch-item data client | DONE | `efbc6a8fc`; `monster/consumable/{requests.go,rest.go,model.go,processor.go,mock/processor.go}` created, uncached per design §7 |
| 10 | atlas-monsters `CATCH` command + resolution ladder | DONE (with documented amendments) | `00d469c1e` base implementation, then `fa1f342cf`, `9e643af49`, `9eff5ea6a` fix commits close every silent-drop path the original brief left open (bare 0-return overload, catch-item lookup failure, ClaimMonster's Redis-error branch). Each fix is described in its commit body as a "plan amendment approved by user" — I could not independently verify a user approval event occurred; treating this as a disclosed, well-reasoned deviation rather than a silent one. `monster/catch.go` final state (read directly) shows no remaining silent-drop exit — every return path emits either the success triple or a failure pair. |
| 11a | atlas-consumables data getters + catch classification | DONE | `248406616`; `data/consumable/model.go` gains 8 getters; `libs/atlas-constants/item/constants.go:41` `ClassificationConsumableCatchItem = Classification(227)` |
| 11b | atlas-consumables request path | DONE | `79f90e7e4`; `catchdelay/registry.go` (Redis useDelay gate), `consumable/catch.go` (`RequestCatchMonster`), producer/consumer wiring; commit body documents 11b+11c as one compile unit — resolved by 11c landing immediately after |
| 11c | atlas-consumables resolution — commit/grant/cancel | DONE | `d00f35d2d` + `4b9f2ff59` (fixes dangling reservation on CATCH command produce failure); `catchResolvedValidator`, `catchResolutionHandler` implemented |
| 12a | atlas-channel handler + command emitter | DONE | `df210a4ff`; `socket/handler/monster_catch_item_use.go` + test, forwards `REQUEST_CATCH_MONSTER`, no unlock sent here (correctly deferred to 12b) |
| 12b | atlas-channel rendering — effects, failure, unlock | DONE | `59f49b2e8`; `kafka/consumer/monster/consumer.go` (99 lines) renders CAUGHT/CATCH_FAILED, maps cause to wire reason, unlocks on every path including UNRESOLVED |
| 13 | Topic configuration, documentation, verification sweep | DONE | `48cb55054`; `deploy/k8s/base/env-configmap.yaml:136` `EVENT_TOPIC_MONSTER_CATCH`, both overlays carry suffixed forms (`-main` / `-PLACEHOLDER_ATLAS_ENV`) — no unsuffixed-fallback trap; `services/atlas-monsters/docs/kafka.md` (+88 lines) and `services/atlas-consumables/docs/kafka.md` (+73 lines) document CATCH/CATCH_RESOLVED/CAUGHT/CATCH_FAILED; both `README.md` topic tables updated |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 13 tasks and their sub-steps have corresponding code, tests, and documentation on the branch.

One item to flag rather than a gap: Task 10's resolution-ladder fix commits (`fa1f342cf`, `9e643af49`, `9eff5ea6a`) describe themselves as "plan amendments approved by user," diverging from the brief's original Kill-style silent-drop text for data-lookup failures. I read the final `monster/catch.go` directly and confirmed the resulting behavior is strictly more correct than the plan's literal text (no dangling reservations, no wedged clients) and is fully covered by `catch_test.go`. I cannot independently verify the claimed user approval occurred, but the change is transparently documented in the commit history rather than silently made, and it improves on FR compliance (every terminal path unlocks — PRD acceptance criterion). Recommend confirming with the task owner that this amendment was in fact approved before merge.

## Build & Test Results

| Service/Module | Build | Vet | Tests | Notes |
|---|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | PASS | all packages `ok` |
| `libs/atlas-redis` | PASS | PASS | PASS | includes `RemoveExisting` concurrency test |
| `libs/atlas-constants` | PASS | PASS | PASS | |
| `services/atlas-monsters/atlas.com/monsters` | PASS | PASS | PASS | `monster` pkg 14.1s, `monster/information` 15.4s |
| `services/atlas-consumables/atlas.com/consumables` | PASS | PASS | PASS | `consumable` pkg 19.3s |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS | full package list `ok`, no failures |

| Guard | Result |
|---|---|
| `tools/redis-key-guard.sh` | PASS |
| `tools/goroutine-guard.sh` | PASS |
| `tools/template-opcode-order-guard.sh` | PASS — 22 arrays ascending |
| `tools/template-duplicate-binding-guard.sh` | PASS — no duplicate bindings |
| `tools/template-movement-types-guard.sh` | PASS — 54 handlers, 11 templates valid |
| `tools/service-registration-guard.sh` | PASS |

**Now run (post-audit):** `docker buildx bake atlas-channel atlas-consumables atlas-monsters` — all three targets PASS (exit 0). `go run ./tools/packet-audit matrix --check` — PASS. `tools/lint.sh --check` (with Node 22 via nvm) — PASS, no drift. `tools/skill-job-id-guard.sh` — PASS. `tools/buff-duration-guard.sh` — PASS. All re-verified against current HEAD (`48cb55054`), not just Task 7's historical commit assertion.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY (the previously-blocking item — mandatory `docker buildx bake` for the three touched services — has since been run and confirmed passing; see "Update" note above)

## Action Items

1. ~~Run `docker buildx bake atlas-channel atlas-consumables atlas-monsters`~~ — DONE, all three targets PASS.
2. ~~Run `go run ./tools/packet-audit matrix --check`, `tools/lint.sh --check`, `tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh` against current HEAD~~ — DONE, all PASS.
3. Confirm with the task owner that the Task 10 resolution-ladder amendments (rounds 2–3, commits `fa1f342cf`/`9e643af49`/`9eff5ea6a`) were in fact approved, since this audit could not independently verify the approval event referenced in those commit messages — though the resulting code is demonstrably correct and well-tested. (Still open — not a code/build gate, needs a human answer.)
4. Check off the plan.md checkboxes (cosmetic only) so the document accurately reflects execution state for future readers. (Still open — cosmetic, non-blocking.)

---

# Backend Guidelines Audit — task-212-monster-catch-items

- **Scope:** Full branch diff (`main...HEAD`) restricted to Go files under `services/atlas-monsters`, `services/atlas-consumables`, `services/atlas-channel`, plus `libs/atlas-redis`, `libs/atlas-constants/item`, and the `deploy/k8s` topic-wiring changes for `EVENT_TOPIC_MONSTER_CATCH`.
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-10
- **Build:** PASS (atlas-monsters, atlas-consumables, atlas-channel — `go build ./...` clean in all three)
- **Tests:** PASS — 0 failed (`go test ./... -count=1` clean in all three modules; no `FAIL` lines in any package)
- **Overall:** NEEDS-WORK at time of audit — see "Resolution (post-audit, same session)" at the end of this section: all 6 blocking findings fixed, re-verified PASS. Overall updated to **READY**.

## Build & Test Results

| Module | Build | Test |
|---|---|---|
| `services/atlas-monsters/atlas.com/monsters` | PASS | PASS (`monster` 14.2s, `monster/information` 15.3s, `monster/consumable` 0.007s, `monster/consumable/mock` no tests) |
| `services/atlas-consumables/atlas.com/consumables` | PASS | PASS (`consumable` 9.99s, `catchdelay` 0.024s) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS (full suite, no failures) |

## DOM-23 — Kafka Topic Naming (EVENT_TOPIC_MONSTER_CATCH)

| Check | Status | Evidence |
|---|---|---|
| Configmap entry `KEY: "KEY"` | PASS | `deploy/k8s/base/env-configmap.yaml:136` — `EVENT_TOPIC_MONSTER_CATCH: "EVENT_TOPIC_MONSTER_CATCH"`, alphabetically ordered between `EVENT_TOPIC_MONSTER_MOVEMENT`/`EVENT_TOPIC_MONSTER_STATUS` |
| Overlay suffix wiring | PASS | `deploy/k8s/overlays/main/kustomization.yaml` (`-main` suffix) and `deploy/k8s/overlays/pr/kustomization.yaml` (`-PLACEHOLDER_ATLAS_ENV` suffix) both add the key, matching sibling topics' pattern |
| No literal env override in service manifest | PASS | `grep -rn "EVENT_TOPIC_MONSTER_CATCH" deploy/k8s/*.yaml` (excluding base/overlays) returns nothing |
| Producer references env var as constant | PASS | `services/atlas-monsters/atlas.com/monsters/monster/kafka.go:16` — `EnvEventTopicMonsterCatch = "EVENT_TOPIC_MONSTER_CATCH"`; emitted via `p.emit(EnvEventTopicMonsterCatch, ...)` in `catch.go:168,174` |
| Consumer references env var as constant | PASS | `services/atlas-consumables/atlas.com/consumables/kafka/message/monster/kafka.go:37` — `EnvEventTopicCatch = "EVENT_TOPIC_MONSTER_CATCH"`; resolved via `topic.EnvProvider` in `kafka/consumer/monster/consumer.go:19` and `consumable/catch.go:84` |
| docs/kafka.md documents the topic | PASS | `services/atlas-monsters/docs/kafka.md` and `services/atlas-consumables/docs/kafka.md` both gained a `EVENT_TOPIC_MONSTER_CATCH` / `CATCH_RESOLVED` section |

**DOM-23 verdict: PASS.** atlas-channel does not consume this topic (it listens on the existing `EVENT_TOPIC_MONSTER_STATUS` for `CAUGHT`/`CATCH_FAILED` presentation events instead), so no manifest changes were needed there.

## DOM-24 — Kafka Producer Stubbed in Tests

| Package | Status | Evidence |
|---|---|---|
| `services/atlas-monsters/atlas.com/monsters/monster` (`catch_test.go`) | PASS | Every `TestCatch_*` constructs `&ProcessorImpl{...}` directly with a custom `emit` field override (e.g. `catch_test.go:44-63`, `newRecordingProcessorWithBodies`) that never touches `producer.ProviderImpl`/the real Kafka manager — equivalent in effect to Pattern B per-test injection. |
| `services/atlas-monsters/atlas.com/monsters/monster` (registry/general) | PASS | `monster/registry_test.go:26-28` `TestMain` calls `producertest.InstallNoop()`. |
| `services/atlas-channel/.../kafka/consumer/monster/consumer_test.go` | PASS (N/A) | New `TestBridleFailReason` calls the pure function `bridleFailReason` directly; no emit path exercised. |
| `services/atlas-channel/.../socket/handler/monster_catch_item_use_test.go` | PASS (N/A) | Only exercises packet decode, never calls `MonsterCatchItemUseHandleFunc` (which would emit). |
| `services/atlas-consumables/atlas.com/consumables/consumable` (`catch_test.go`) | **FAIL** | See finding below. |
| `services/atlas-consumables/atlas.com/consumables/catchdelay/registry_test.go` | PASS (N/A) | Pure Redis registry test, no Kafka emit path. |

### FAIL — DOM-24: service-local stub writer + mid-package reinstall

`services/atlas-consumables/atlas.com/consumables/consumable/catch_test.go:81-100` defines a service-local `stubWriter` type (`Topic()`/`WriteMessages()`/`Close()`) implementing `producer.Writer` with a per-topic `fail` flag, and `catch_test.go:116-118` installs it directly via `producer.ResetInstance()` + `producer.GetManager(producer.ConfigWriterFactory(...))` instead of the shared `producertest` package. `catch_test.go:120-121` then reinstalls the package's stub in `t.Cleanup(func() { emitted = producertest.InstallCapturing() })` to restore state for the rest of the package's tests.

This is the exact pattern `testing-guide.md` bans: *"Do NOT roll your own no-op writer in a service-local `testkafka` package. Use `producertest.InstallNoop()` so the stub stays consistent across services."* and the reviewer's DOM-24(d)/(e): a service-local writer plus a mid-package reinstall-via-`t.Cleanup` that "resets the singleton back to ... default for the next test in the package and partially defeats the TestMain stub" (here reinstalling `InstallCapturing()` rather than `producer.ResetInstance()` directly, but the same singleton-swap-in-Cleanup fragility). The package's own `testmain_test.go:10-13` documents the intended convention as "individual tests call `emitted.Reset()` rather than reinstalling" — this test does the opposite.

The underlying need (a writer that fails for one specific topic to prove `ConsumeCatch` cancels the reservation on produce failure) is legitimate and not served by `producertest.Capture`, which has no failure-injection capability — but the guideline draws no exception for that; the fix belongs upstream in `libs/atlas-kafka/producer/producertest` (e.g. an `InstallFailing(topic string)` helper), not as a per-test local writer.

## File Responsibilities Checklist

| Package | Check | Status | Evidence |
|---|---|---|---|
| `services/atlas-monsters/atlas.com/monsters/monster` | FILE-01 | **FAIL** | `monster/catch.go:108` `func (p *ProcessorImpl) Catch(...)`, `:173` `emitCatchFailure`, `:181` `emitCatchUnresolved` — `ProcessorImpl` methods live in a bare topic-named file (`catch.go`), not `processor.go` or a `processor_catch.go` split. The `Processor` interface itself declares `Catch(...)` in `monster/processor.go:67`, so the interface and its implementation are split across a processor-named file and a non-processor-named file — exactly the anti-pattern the checklist calls out ("a bare topic name like `custody.go`/`register.go`"). |
| `services/atlas-consumables/atlas.com/consumables/consumable` | FILE-01 | **FAIL** | `consumable/catch.go:53` `func (p *ProcessorImpl) RequestCatchMonster(...)`, `:131` `catchError(...)` — same violation. `Processor` interface declares `RequestCatchMonster` in `consumable/processor.go:83`. |
| `services/atlas-monsters/atlas.com/monsters/monster/consumable` | FILE-05 | **FAIL** | `monster/consumable/model.go:21-33` defines `ModelBuilder` (fluent setters + `Build()`) inside `model.go` instead of a separate `builder.go`. Per file-responsibilities.md, "Builder" is its own file with its own responsibility, distinct from "Model". |
| `services/atlas-monsters/atlas.com/monsters/monster/consumable` | FILE-02/03/04 | PASS | `RestModel`/`Transform`-equivalent (`Extract`)/JSON:API methods all in `rest.go`; request funcs (`getBaseRequest`, `requestById`) all in `requests.go`; no `entity.go` needed (pure REST client, no DB entity). |
| `services/atlas-consumables/atlas.com/consumables/monster` | FILE-01 | PASS | `RequestCatch` implemented in `monster/processor.go:42` alongside the interface (`:15`); `producer.go` holds `catchCommandProvider`. |
| `services/atlas-consumables/atlas.com/consumables/catchdelay` | FILE-06 | PASS | Single-purpose Redis-backed registry file; not a multi-responsibility catch-all. |
| `services/atlas-channel/atlas.com/channel/consumable` | FILE-01/FILE-05 | PASS | `RequestCatchMonster` method in `processor.go`, provider in `producer.go`. |
| `services/atlas-channel/atlas.com/channel/socket/handler`, `socket/writer` | — | PASS | New handler/writer files are each single-purpose, correctly named, and delegate to processors only (no provider/DB calls). |

## External HTTP Client Checklist — `services/atlas-monsters/atlas.com/monsters/monster/consumable` (new atlas-data client)

| ID | Check | Status | Evidence |
|---|---|---|---|
| EXT-01 | Relationship interface methods | **FAIL** | `monster/consumable/rest.go` — `RestModel` has no `SetToOneReferenceID`/`SetToManyReferenceIDs`, even as no-ops. Absent methods make api2go error on any atlas-data response carrying a `relationships` block. |
| EXT-02 | httptest-backed integration test | **FAIL** | `monster/consumable/rest_test.go` only unit-tests `Extract()` and `NewModelBuilder()`; no `httptest.NewServer` exercising `requestById`/`GetById` against a fixture JSON:API response. `mock/processor.go` mocks the interface but that bypasses unmarshal per the guideline's explicit carve-out ("FakeClient mocks alone do NOT satisfy this"). |
| EXT-03 | 404 vs. other-failure distinction | PASS (N/A) | `consumable/processor.go:31-33` `GetById` forwards the raw error from `requests.Provider` untranslated — no incorrect collapsing of transport/decode/5xx into a domain "not found" occurs, so there is nothing to flag; the caller (`monster/catch.go`) treats any error uniformly as `UNRESOLVED`, which is a deliberate domain decision, not an EXT-03 violation. |
| EXT-04 | `RootUrl` used, not hardcoded | PASS | `monster/consumable/requests.go:15` — `requests.RootUrl("DATA")`. |

## DOM-21 — atlas-constants Reuse

PASS. `services/atlas-consumables/atlas.com/consumables/consumable/catch.go:34` uses `item2.GetClassification(item2.Id(itemId)) != item2.ClassificationConsumableCatchItem` rather than an ad-hoc `itemId/10000` check; the new classification constant itself was added to the shared library at `libs/atlas-constants/item/constants.go:41` (`ClassificationConsumableCatchItem = Classification(227)`), not redeclared locally.

## DOM-25 — Client Wire Values Config-Resolved

PASS. `services/atlas-monsters` and `services/atlas-consumables` emit only semantic cause strings (`CatchCauseSpeciesMismatch`, `CatchCauseHpTooHigh`, `CatchCauseRollFailed`, `CatchCauseUnresolved`, `CatchCauseUseDelay`, `CatchCauseInventoryFull`, `CatchCauseInvalidItem`). The wire-byte mapping is resolved exclusively in atlas-channel's `bridleFailReason` (`kafka/consumer/monster/consumer.go:730-742`), which is a Go `switch` on the semantic string, not a client wire code hardcoded upstream. The opcode/handler itself is template-routed (`socket/handler/monster_catch_item_use.go:21-22` comment confirms "opcode arrives from tenant configuration via the template route — never a constant here").

## DOM-26 — Goroutines via routine.Go

PASS. `tools/goroutine-guard.sh` exits 0 against the full tree (no diff-scoped bare `go` statements found in any touched package).

## DOM-22 — Dockerfile / go.mod Sync

N/A. No `go.mod` changes in any of the three touched services' modules (`git diff main...HEAD --stat` for all three `go.mod` files returns empty) — the new `libs/atlas-redis` method and `libs/atlas-constants` constant are additions to already-declared dependencies, not new deps.

## Mock Synchronization (testing-guide.md)

PASS. All four interface changes (`consumable.Processor.RequestCatchMonster` in atlas-channel and atlas-consumables, `monster.Processor.RequestCatch` in atlas-consumables, `consumable.Processor.GetById` — new package — in atlas-monsters) have matching mock updates: `services/atlas-channel/.../consumable/mock/processor.go`, `services/atlas-consumables/.../consumable/mock/processor.go`, `services/atlas-consumables/.../monster/mock/processor.go`, `services/atlas-monsters/.../monster/consumable/mock/processor.go` (new file). All follow the function-field + nil-check pattern.

## Summary

### Blocking (must fix)

- **FILE-01** — `services/atlas-monsters/atlas.com/monsters/monster/catch.go:108,173,181` — `ProcessorImpl` methods (`Catch`, `emitCatchFailure`, `emitCatchUnresolved`) live in a bare topic-named file instead of `processor.go`/`processor_catch.go`.
- **FILE-01** — `services/atlas-consumables/atlas.com/consumables/consumable/catch.go:53,131` — `ProcessorImpl` methods (`RequestCatchMonster`, `catchError`) live in a bare topic-named file instead of `processor.go`/`processor_catch.go`.
- **FILE-05** — `services/atlas-monsters/atlas.com/monsters/monster/consumable/model.go:21-33` — `ModelBuilder` lives in `model.go` instead of a separate `builder.go`.
- **DOM-24** — `services/atlas-consumables/atlas.com/consumables/consumable/catch_test.go:81-121` — service-local `stubWriter` installed via raw `producer.ConfigWriterFactory` instead of the shared `producertest` package, plus a mid-package reinstall in `t.Cleanup`.
- **EXT-01** — `services/atlas-monsters/atlas.com/monsters/monster/consumable/rest.go` — `RestModel` missing `SetToOneReferenceID`/`SetToManyReferenceIDs`.
- **EXT-02** — `services/atlas-monsters/atlas.com/monsters/monster/consumable/rest_test.go` — no httptest-backed integration test for the atlas-data client.

### Non-Blocking (should fix)

- None identified beyond the items above.

### Resolution (post-audit, same session)

All six blocking findings were fixed on this branch before PR:

- **FILE-01 (atlas-monsters):** `monster/catch.go` renamed to `monster/processor_catch.go` (and `catch_test.go` to `processor_catch_test.go`); no content change beyond the rename.
- **FILE-01 (atlas-consumables):** `consumable/catch.go` renamed to `consumable/processor_catch.go` (and `catch_test.go` to `processor_catch_test.go`).
- **FILE-05:** `ModelBuilder` moved out of `monster/consumable/model.go` into a new `monster/consumable/builder.go`.
- **EXT-01:** `monster/consumable/rest.go`'s `RestModel` gained no-op `SetToOneReferenceID`/`SetToManyReferenceIDs` methods, matching the convention in `character/buff/rest.go`.
- **EXT-02:** added `monster/consumable/processor_http_test.go` with an `httptest.NewServer`-backed round trip (`TestGetById_HTTPRoundTrip`, `TestGetById_HTTPRoundTrip_NotFound`), mirroring `map/processor_location_http_test.go`'s pattern — exercises the real `requests.Provider` → JSON:API decode → `Extract` path, not a mock.
- **DOM-24:** removed the service-local `stubWriter` and mid-package `t.Cleanup` reinstall from `consumable/processor_catch_test.go`. Instead extended the shared `libs/atlas-kafka/producer/producertest` package itself with `Capture.FailTopic(topicName, fail bool)` (marks one topic's `CapturingWriter` to return `ErrSimulatedProduceFailure`) and changed `Capture.Reset()` to clear each writer's message/fail state **in place** rather than replacing the writer objects. The in-place change fixes a latent correctness gap the removed stub had been silently working around: `producer.Manager` caches one `Writer` per topic for the singleton's lifetime, so a `Reset()` that discards and recreates writer objects orphans any topic already touched by an earlier test — later writes land on the Manager's stale cached reference while the Capture's map points at a new, empty one, so `Messages()` silently reads back nothing. This was verified by reproducing the exact failure (`TestConsumeCatchCancelsReservationWhenCommandProduceFails` failed with "expected 1 compartment command ... got 0" under the naive replace-the-map Reset) before landing the in-place fix. `libs/atlas-kafka`, `services/atlas-monsters/atlas.com/monsters`, `services/atlas-consumables/atlas.com/consumables`, and `services/atlas-channel/atlas.com/channel` were all rebuilt/re-tested/re-baked after this change — all PASS.

All three affected modules (`services/atlas-monsters/atlas.com/monsters`, `services/atlas-consumables/atlas.com/consumables`, `libs/atlas-kafka`) plus their downstream consumers (`services/atlas-channel/atlas.com/channel`, `services/atlas-configurations/atlas.com/configurations`) were rebuilt, vetted, tested (`-race`), guard-checked, linted, and re-baked (`docker buildx bake atlas-monsters atlas-consumables atlas-channel`) after these fixes — all gates PASS. Updated **Overall: READY** (was NEEDS-WORK).

