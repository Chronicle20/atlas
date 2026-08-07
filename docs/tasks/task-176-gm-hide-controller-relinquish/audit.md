# Plan Audit — task-176-gm-hide-controller-relinquish

**Plan Path:** docs/tasks/task-176-gm-hide-controller-relinquish/plan.md
**Audit Date:** 2026-07-18
**Branch:** task-176-gm-hide-controller-relinquish
**Base Branch:** main (merge-base `be0c94338`)

## Executive Summary

All 13 plan tasks are faithfully implemented with direct file:line evidence; the plan's checkbox tracking in `plan.md` was simply never ticked (0/72 steps checked), but every commit in the 18-commit branch (`be0c94338..7cdab320b`) maps 1:1 to a plan task and its commit message. All three documented intentional deltas from the design are present exactly as declared: the `SetNX`-addition (Task 6-that-doesn't-exist) is correctly absent, the DPS-leader hidden guard is implemented at `monster/processor.go:596`, and the hidden registry stores tenant identity in its payload (`storedHidden` struct). Both critical concurrency bugs found during execution — the `hiddenCache` data race (`e6c75ed42`) and the `npcIds` slice race in GM-reveal enumeration (`12d1539b5`) — are fixed with regression tests and clean under `-race`. Builds, `go vet`, and `-race` test suites are clean in all three affected modules (atlas-monsters, atlas-channel, libs/atlas-packet); `redis-key-guard.sh`, `goroutine-guard.sh`, `go run ./tools/packet-audit matrix --check`, and `docker buildx bake atlas-channel` all pass with exit 0. `tools/lint.sh --check` reports 0 issues on every Go module (its one failure is `ui:node-missing`, an environment/nvm gap unrelated to this Go-only branch — no atlas-ui files are touched in the diff).

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-monsters hidden-character registry | DONE | `character/hidden/registry.go:1-83` (`Registry`, `Add`/`Remove`/`MemberSet`/`GetAll`/`Clear`, `InitRegistry`); `main.go:67` `hidden.InitRegistry(rc)`. Commit `c4972914f`. |
| 2 | atlas-monsters election excludes hidden | DONE | `monster/processor.go:84` `ErrNoControllerCandidate` sentinel; `:299-359` `getControllerCandidate` (hidden-set fetch, puppet-bias skip, pool-leak fix — only seeded ids incremented, lines 327-341); `:364-378` `FindNextController` swallows the sentinel as a no-op; `:596` DPS-leader hidden guard in `Damage`. `zeroValue`/`characterIdKey` helpers confirmed removed (`grep` returns nothing). Tests: `TestGetControllerCandidateExcludesHidden`, `TestControlCountsDoNotResurrectNonPoolControllers`, `TestFindNextControllerNoCandidateIsNoop`, `TestPuppetBiasSkipsHiddenOwner`, `TestDamageDPSLeadSwitchSkippedWhenLeaderHidden` (`processor_test.go`). Commits `a4c8735ef`, `b527b05e0`. |
| 3 | atlas-monsters character-location REST client | DONE | `map/rest.go:39-70` `LocationRestModel`/`ExtractLocation`; `map/requests.go:14,30-31` `characterLocationResource`/`requestCharacterLocation`; `map/processor.go:15,41-46` `GetCharacterField`; `map/mock/processor.go:12,24-26` mock. Commit `a0ba74680`. |
| 4 | atlas-monsters hide/reveal processor + buff consumer | DONE | `monster/processor.go:94,116,430,466` `locationFn` seam, `RelinquishControlOnHide`, `RestoreCandidacyOnReveal`; `kafka/message/buff/kafka.go` event defs; `kafka/consumer/buff/consumer.go:37-70` `InitHandlers`, SourceId-filtered `handleStatusEventApplied`/`handleStatusEventExpired`; `main.go:5,73,81` consumer registration. Commit `772058fa7`. |
| 5 | atlas-monsters reconciliation sweep | DONE | `character/buff/{model,processor,requests,rest}.go` buffs REST client; `character/hidden/task.go:30-63` `NewReconciliationTask`/`Run`/`SleepTime`; `main.go:108` `tasks.Register(l, ctx)(hidden.NewReconciliationTask(...))` inside leader-gated `registerSweepTasks`. `go.mod` diff empty as plan predicted (`git diff` confirmed). Commit `9913a4e6e`. |
| 6 | libs/atlas-packet remove-controller arm | DONE | `libs/atlas-packet/npc/clientbound/remove_controller.go` — `RemoveController`, `NewNpcRemoveController`, `Operation()` returns existing `NpcSpawnRequestControllerWriter`, flag-0 Encode/Decode matching the IDA-derived layout. `coverage-manifest.yaml` present, declares `SPAWN_NPC_REQUEST_CONTROLLER` across 5 versions, `out_of_scope: []`. Independently corroborated by `docs/tasks/task-176-gm-hide-controller-relinquish/completeness-critic.md` — verdict CLEAN, 0 findings. Commit `549c4a8f7`. |
| 7 | atlas-channel NPC-controller registry + Redis bootstrap | DONE | `go.mod:11` `atlas-redis` require (+ `:98` existing replace); `npc/controller/registry.go:61-127` `Claim`/`Release`/`ControllerOf`/`GetAll`/`ControlledBy` on `TenantKeyedHash`; `main.go:62,182-183` `atlas.Connect(l)` + `controllernpc.InitRegistry(rc)`. Commit `c1c9eb855`. |
| 8 | atlas-channel election processor | DONE | `npc/controller/processor.go:98,134,158,236,263` `TryClaim`/`ReleaseFor`/`ElectFor`/`UncontrolledIn`/`IsController`, all matching plan signatures. **Includes post-hoc concurrency fix**: `:37` `hiddenCacheMu sync.Mutex` added by `e6c75ed42` to guard `isHidden`'s check-then-fetch-then-store against `data/npc.ForEachInMap`'s per-NPC goroutine fan-out; regression test `TestTryClaimConcurrentSameCharacterId` added in the same commit. Commit `79622b946` (+ fix `e6c75ed42`). |
| 9 | announce helpers + spawn-path gating | DONE | `npc/controller/announce.go:17,33` `AnnounceGrant`/`AnnounceRevoke`; `kafka/consumer/map/consumer.go:576,591,613,622` `spawnNPCForSession` rewritten to spawn-to-everyone / grant-only-to-elected-controller via `cp.TryClaim`. Commit `38bd324bf`. |
| 10 | NPC reassignment on map exit | DONE | `kafka/consumer/map/consumer.go:571-593` exit handler extended: `cp.ReleaseFor` → `cp.ElectFor(f, released, exiting-char)` → `AnnounceGrant` per assignment, exactly as specified. Commit `e3e43d4f1`. |
| 11 | channel hide/reveal branches (revoke + reassign) | DONE | `kafka/consumer/buff/consumer.go:55-65` registers `handleStatusEventGmHideApplied`/`Expired`; `:147-` and `:196-` implement release+revoke+re-elect (Applied) and uncontrolled-NPC re-election (Expired) via session-field, no atlas-maps call, matching design §3.2. **Includes post-hoc concurrency fix**: `12d1539b5` switched EXPIRED's NPC enumeration from `ForEachInMap`'s parallel-callback append (racy) to `InMapModelProvider(...)()` synchronous fetch + sequential range, same bug class as the Task 9 fix. Commit `ef679fe13` (+ fix `12d1539b5`). |
| 12 | NPC movement/animation guard + relay | DONE | `movement/processor.go:89,101` `ForNPC` gains `controllernpc.IsController` guard + `ForOtherSessionsInMap` relay; `socket/handler/npc_action.go:36,45` animation branch gains the same guard + relay. Commit `7cdab320b`. |
| 13 | full verification sweep | DONE | Re-run independently by this audit (see Build & Test Results) — all gates green: per-module `go build`/`go vet`/`go test -race` clean; `redis-key-guard.sh` exit 0; `goroutine-guard.sh` exit 0; `lint.sh --check` 0 issues on all Go modules (only failure is unrelated `ui:node-missing`); `docker buildx bake atlas-channel` succeeded; `go run ./tools/packet-audit matrix --check` exit 0 with no drift. |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 13 tasks have direct, verifiable file:line evidence and passing tests.

**Process note (not a deficiency):** `plan.md`'s 72 step-level checkboxes (`- [ ]`) were never ticked to `[x]` during execution — `grep` finds 0 checked boxes tree-wide. This appears to be a tracking-hygiene gap in the execute-task workflow rather than incomplete work: every step's deliverable is present, tested, and committed with a message matching the plan's prescribed commit message almost verbatim. Recommend ticking the boxes (or noting in the PR description that tracking was done via commits) before merge, purely for audit-trail cleanliness.

## Build & Test Results

| Service/Module | Build | Vet | Tests (-race) | Notes |
|---|---|---|---|---|
| atlas-monsters | PASS | PASS | PASS | All packages ok, including `character/hidden`, `kafka/consumer/buff`, `monster` (18.3s), `map`. |
| atlas-channel | PASS | PASS | PASS | All packages ok, including `npc/controller` (registry + processor + concurrency regression test), `socket/handler`, `movement`. |
| libs/atlas-packet | PASS | PASS | PASS | `npc/clientbound` and `npc/serverbound` both ok (new remove-controller fixture test included). |

Additional gates (repo root):

| Gate | Result |
|---|---|
| `tools/redis-key-guard.sh` | exit 0 |
| `tools/goroutine-guard.sh` | exit 0 |
| `tools/lint.sh --check` | 0 issues on every Go module; overall script exits non-zero only due to `ui:node-missing` (Node/nvm not sourced in this shell — no atlas-ui files touched on this branch, so this is an environment gap, not a code defect) |
| `docker buildx bake atlas-channel` | success (cached layers, image exported) — mandatory per CLAUDE.md since `go.mod` changed; only `atlas-channel`'s `go.mod` changed (`git diff --stat main...HEAD -- '**/go.mod'`) |
| `go run ./tools/packet-audit matrix --check` | exit 0, no drift |
| `docs/tasks/.../completeness-critic.md` (independent) | verdict CLEAN, 0 findings |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Cosmetic, optional) Tick the 72 step checkboxes in `plan.md` to `[x]`, or add a note to the PR description that task tracking was done via the 18-commit history rather than the checkbox file, so a future reader of `plan.md` alone isn't misled into thinking nothing was done.
2. No code changes required before merge based on this plan-adherence audit. (Backend-guidelines and any other reviewer findings are out of scope for this report — see their own audit sections/files if run separately.)

---

# Backend Guidelines Audit — task-176-gm-hide-controller-relinquish

- **Scope:** Go changes on `be0c94338..7cdab320b` — atlas-monsters (`character/hidden`, `character/buff`, `map` client additions, `monster/processor.go`, `kafka/consumer/buff`, `kafka/message/buff`, `main.go`), atlas-channel (`npc/controller`, `kafka/consumer/buff`, `kafka/consumer/map`, `movement/processor.go`, `socket/handler/npc_action.go`, `main.go`, `go.mod`), libs/atlas-packet (`npc/clientbound/remove_controller.go`).
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-07-18
- **Build:** PASS (`go build ./...` clean in all three modules)
- **Vet:** PASS (`go vet ./...` clean in all three modules)
- **Tests:** PASS, all packages, including `-race` on every touched package (0 failures)
- **Overall:** **NEEDS-WORK** — build/tests/race are clean, but four guideline checks FAIL (one Important structural/wire-protocol finding, three External-HTTP-Client findings on the two new REST-client packages).

## Phase 1 — Build & Test (verified independently)

```
cd services/atlas-monsters/atlas.com/monsters && go build ./...   # clean
cd services/atlas-channel/atlas.com/channel   && go build ./...   # clean
cd libs/atlas-packet                          && go build ./...   # clean

go test ./... -count=1        # all three modules: PASS, 0 failures
go test -race ./... -count=1  # scoped to touched packages: PASS, 0 failures, 0 data races
go vet ./...                  # all three modules: clean
tools/goroutine-guard.sh      # exit 0, tree-wide
tools/redis-key-guard.sh      # exit 0, tree-wide
```

## Phase 2 — Domain Discovery

| Package | Classification | Notes |
|---|---|---|
| `atlas-monsters/character/hidden` | Support (Redis registry + sweep task) | `registry.go` mirrors the pre-existing `monster/puppet_registry.go` singleton pattern; `task.go` is a sweep runner. No `model.go`. |
| `atlas-monsters/character/buff` | Domain-shaped (has `model.go`) but is a stateless REST-client projection, no DB | See DOM-01..04 notes below — entity/builder criteria don't apply to a package with no persistence. |
| `atlas-monsters/map` (client additions) | Support (pre-existing package, extended with a REST client) | `LocationRestModel`/`ExtractLocation`/`GetCharacterField` added to existing `rest.go`/`requests.go`/`processor.go`. |
| `atlas-monsters/monster` (processor.go changes) | Domain (pre-existing, has `model.go`/`entity.go`/etc.) | Only `processor.go` touched; new methods added to the existing interface + impl, in-file. |
| `atlas-monsters/kafka/consumer/buff`, `kafka/message/buff` | Support (Kafka consumer + message contract) | Standard `InitConsumers`/`InitHandlers` curried pattern. |
| `atlas-channel/npc/controller` | Support (Redis registry + election processor) | `registry.go`, `processor.go`, `announce.go` — no `model.go`, single-purpose files. |
| `atlas-channel/kafka/consumer/buff` (buff consumer additions) | Support (pre-existing package, two new handlers added) | |
| `libs/atlas-packet/npc/clientbound/remove_controller.go` | Packet codec | New `RemoveController` Encode/Decode. |

## Phase 3 — Findings

### Blocking (Important)

| ID | Package | Finding |
|---|---|---|
| DOM-25 | `libs/atlas-packet/npc/clientbound/remove_controller.go:43` | `w.WriteByte(0)` hardcodes the client dispatcher flag byte for the remove/revoke arm of `CNpcPool::OnNpcChangeController` (0=remove, 1=grant) as a Go literal instead of resolving it from a tenant writer-options table. This is exactly the anti-pattern documented in `anti-patterns.md` ("Anti-Pattern: Hardcoding client-interpreted wire values") — "the value is version-stable (IDA-verified identical)" is explicitly called out as NOT an exemption (task-102/103 uniformity ruling). The sibling grant arm (`spawn_request_controller.go:40`, `w.WriteByte(1)`) has the same defect but is pre-existing/untouched by this diff — noted as related pre-existing debt, not a new finding, but it means the new code perpetuated rather than fixed the pattern. `context.md` shows the author was aware of this precedent ("Grant packet ... hard-codes flag byte 1 ... reused by the new remove arm") and chose to match it rather than resolve it via config. |
| EXT-01 | `services/atlas-monsters/atlas.com/monsters/map/rest.go` (`LocationRestModel`), `services/atlas-monsters/atlas.com/monsters/character/buff/rest.go` (`RestModel`) | Neither new cross-service REST-client target struct implements `SetToOneReferenceID`/`SetToManyReferenceIDs`. Both are genuinely new HTTP clients introduced by this task (atlas-maps location client, atlas-buffs buffs client), both go through `requests.GetRequest[T]`/`requests.DrainProvider[T,...]`. Per `libs/atlas-rest/CLAUDE.md` and EXT-01, api2go errors on any response carrying a `relationships` block without these (even no-op) methods — this exact failure mode previously surfaced as misleading "not found" errors (task-037). Compare with the correctly-implemented sibling `services/atlas-maps/atlas.com/maps/character/location/rest.go:42-49`, which has both as no-ops. |
| EXT-02 | `services/atlas-monsters/atlas.com/monsters/character/buff/` (no test files at all — `ls character/buff/*_test.go` finds nothing), `services/atlas-monsters/atlas.com/monsters/map/processor_drain_test.go` (only covers `CharacterIdsInFieldProvider`, not `GetCharacterField`) | Neither new client has an httptest-backed integration test exercising the real JSON:API unmarshal path. `hidden/task_test.go` and `monster/processor_test.go` both inject `buffsFn`/`locationFn` seams directly, bypassing `Extract`/`ExtractLocation`/`RestModel` entirely — so a broken unmarshal (e.g. the EXT-01 gap above triggering an api2go error) would not be caught by any test in this diff. |
| EXT-03 | Same two client packages | No `errors.Is(err, requests.ErrNotFound)` anywhere in `map/` or `character/buff/` of atlas-monsters. `RelinquishControlOnHide`/`RestoreCandidacyOnReveal` (`monster/processor.go`) treat a genuine 404 (character has no location row — offline, expected) and a transient atlas-maps outage (5xx/network failure) identically: both hit the same `if err != nil { ...Debugf...; return nil }` branch with no differentiation and no metric. A real atlas-maps outage during a GM-hide event would silently degrade to "skip relinquish, converges later" with no operational signal beyond a Debug-level log line. |

### Non-Blocking (Minor)

| ID | Package | Finding |
|---|---|---|
| DOM-01/02/03/04 | `atlas-monsters/character/buff` | Has `model.go` (private-field + accessor `Model`) but no `builder.go`, no `entity.go` (`ToEntity`/`Make`), and `rest.go` defines `Extract` (inbound) rather than `Transform` (outbound). Judged N/A rather than FAIL: the package is a stateless, read-only projection of a subset of atlas-buffs' JSON:API response with no local persistence — the builder/entity criteria in `file-responsibilities.md` presuppose a GORM-backed domain, which this package genuinely is not. `Extract` is the correct inbound analog per the `requests.go`/EXT pattern. Flagging for visibility since the package technically matches the Phase 2 "has model.go → domain package" trigger. |
| DOM-20 | `hidden/registry_test.go`, `hidden/task_test.go`, `kafka/consumer/buff/consumer_test.go`, `monster/processor_test.go` (new tests), `npc/controller/registry_test.go`, `remove_controller_test.go` | New tests use individual `func TestXxx` per scenario rather than the `tests := []struct{...}; t.Run(...)` table-driven pattern. `testing-guide.md` frames this as "prefer," not mandatory, and each function tests one clear behavior with good naming/comments; scenario coverage is thorough (hidden-exclusion, sentinel, fail-open, pool-leak regression, concurrency) and race-clean. Style gap, not a structural defect. |
| (unlabeled, DOM-28-adjacent) | `monster/processor.go` `RelinquishControlOnHide`/`RestoreCandidacyOnReveal` | Location-lookup failure logs at `Debugf` (not `Warnf`) with no degrade-metric increment, then silently proceeds (hidden-set mutation still applied, monster relinquish skipped). Not a strict DOM-28 violation — no `model.Decorator[...]` is involved, so the check's literal trigger doesn't fire — but the same "swallow failure, continue degraded" shape. Mitigated by being an explicitly documented, deliberate design choice (FR-7.2: "set mutation applied even when location fails... reconciliation repairs") rather than an accidental silent drop, and by the 5-minute reconciliation sweep providing a backstop. Recommend bumping to `Warnf` for the "location fetch errored" sub-case specifically (as opposed to "no location row found," which is the expected offline case) so an atlas-maps outage is operationally visible.

### Passed (representative, file:line evidence)

| ID | Evidence |
|---|---|
| DOM-06 | `character/buff/processor.go:21` `NewProcessor(l logrus.FieldLogger, ctx context.Context)`; `map/processor.go:24`; `npc/controller/processor.go:41` — all take `logrus.FieldLogger`. |
| DOM-21 | `character/buff/model.go:6,31` imports/uses `atlas-constants/skill.SuperGmHideId` rather than a local literal for the production check; `character/hidden/registry.go` and `npc/controller/registry.go` use `tenant.Model`/`field.Model` accessors throughout, no redeclared id types. |
| DOM-23 | `deploy/k8s/base/env-configmap.yaml:94` `EVENT_TOPIC_CHARACTER_BUFF_STATUS: "EVENT_TOPIC_CHARACTER_BUFF_STATUS"`; no literal override found in `deploy/k8s/base/atlas-monsters.yaml` or `atlas-channel.yaml`; both consume via `envFrom`. |
| DOM-22 | N/A — repo-root `Dockerfile` unconditionally `COPY`s and `go.work use`s all 21 shared libs (incl. `atlas-redis`, lines 41/71/94) for every service target (task-074 shared-Dockerfile architecture superseded the per-lib-COPY model `CLAUDE.md` describes for genuinely *new* libs). atlas-channel's new direct require of an already-present lib needs zero Dockerfile edits. Corroborated by the plan-adherence audit's successful `docker buildx bake atlas-channel` run (see above). |
| DOM-24 | `monster/registry_test.go:26-27` package-wide `TestMain` calls `producertest.InstallNoop()`, covering `processor_test.go`'s `TestRelinquishOnHideReassignsControlledMobs`/`TestRestoreCandidacyOnRevealRemovesFromSetAndSweeps` (which use the real `NewProcessor(...)` emit path through `StopControl`/`StartControl`). No `t.Cleanup(producer.ResetInstance)` found anywhere in scope. |
| DOM-26 | `movement/processor.go:60,67,81,109,122,188,210,217,248,257` — every goroutine spawned via `routine.Go`; `main.go:122` (atlas-monsters) leader-election goroutine via `routine.Go`. `tools/goroutine-guard.sh` exit 0. |
| FILE-01..06 | `npc/controller/processor.go` — `Processor` interface + all `ProcessorImpl` methods in `processor.go`; `registry.go` (Redis registry only) and `announce.go` (packet announce only) are single-purpose, not catch-alls. `character/hidden/registry.go` mirrors `monster/puppet_registry.go`'s established singleton-registry convention. No `<pkg>.go` bundling ≥2 responsibilities found in any new/touched package. |
| Concurrency fixes (user-flagged, verified sound) | `npc/controller/processor.go:37-38,73-82` — `hiddenCacheMu sync.Mutex` guards the full check-then-fetch-then-store of `hiddenCache` in `isHidden`; race-clean under `go test -race`. `kafka/consumer/buff/consumer.go:209-226` (atlas-channel) — GM-reveal handler uses `npc2.NewProcessor(l,ctx).InMapModelProvider(f.MapId())()` (synchronous fetch) + a sequential range to build `npcIds`, replacing the prior `ForEachInMap` parallel-callback append that raced on the slice header; matches commit `12d1539b5`'s stated fix and is confirmed present at HEAD. |
| No Cosmic citations | `grep -rniE cosmic` over every touched/new file in scope: 0 matches. |
| Multitenancy | `hidden/registry.go` (`Add`/`Remove`/`MemberSet` all take `tenant.Model`), `npc/controller/processor.go:45` `tenant.MustFromContext(ctx)`, `character/hidden/task.go:35` `tenant.WithContext(ctx, ten)` — all new processors correctly context-scope tenant identity. |

## Backend-Guidelines Summary

### Blocking (must fix)
- DOM-25: `libs/atlas-packet/npc/clientbound/remove_controller.go:43` — resolve the OnNpcChangeController flag byte from a tenant writer-options table instead of a Go literal (and ideally fix the pre-existing sibling `spawn_request_controller.go:40` in the same pass, since both arms of one dispatcher should be resolved together).
- EXT-01: add no-op `SetToOneReferenceID`/`SetToManyReferenceIDs` to `map/rest.go`'s `LocationRestModel` and `character/buff/rest.go`'s `RestModel` in atlas-monsters.
- EXT-02: add an `httptest`-backed test for `map.Processor.GetCharacterField` (fixture matching atlas-maps' actual `character-locations` JSON:API shape) and at least one test file for `character/buff` exercising `Processor.GetByCharacterId` against an `httptest` server.
- EXT-03: distinguish `requests.ErrNotFound` from other failures in `GetCharacterField`/`GetByCharacterId` call sites, or at minimum log the two cases at different levels in `RelinquishControlOnHide`/`RestoreCandidacyOnReveal` so an atlas-maps outage is distinguishable from "character has no location row" in logs/metrics.

### Non-Blocking (should fix)
- Consider table-driven consolidation for the new test files (DOM-20), low priority.
- Bump the "location fetch errored" log branch in `RelinquishControlOnHide`/`RestoreCandidacyOnReveal` from `Debugf` to `Warnf` to distinguish it from the expected "no location row" case.
- `character/buff` package technically triggers the Phase-2 "has model.go → domain package" checklist; consider a package comment noting it's intentionally builder/entity-less (REST-client projection) to preempt future audit re-litigation.
