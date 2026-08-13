# Plan Audit — task-218-player-cast-mists

**Plan Path:** docs/tasks/task-218-player-cast-mists/plan.md
**Audit Date:** 2026-08-12
**Branch:** task-218-player-cast-mists
**Base Branch:** main (range audited: `688740664..6c4faade6`, 20 commits)

## Executive Summary

All 16 plan tasks are implemented and traceable to a matching commit; the plan
uses no per-task checkbox tracking (all 101 step-level `- [ ]` boxes are
unticked, which is this project's normal authoring artifact, not a completion
signal — completion is tracked by the commit list and `context.md` §8.1's
verification sweep instead). Every requirement-coverage row (FR-0 through
FR-8, the mirror-guard discipline, the §8 NFRs, and the §10 acceptance rows)
has direct file:line evidence in the diff. Build, vet, and package-level tests
pass in both changed services (`atlas-maps`, `atlas-channel`) and in
`libs/atlas-constants`; all seven CLAUDE.md guards (redis-key, goroutine,
buff-duration, skill-job-id, trade-contract-mirror, and the new
mist-contract-mirror) exit 0. No gaps found.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Snapshot drain tooling (`mksnapshot`, `wzsnapshot-drain.sh`) | DONE | Commit `51d1d713e`; `libs/atlas-constants/gen/wzsnapshot/cmd/mksnapshot/main.go`, `tools/wzsnapshot-drain.sh` present and executable. |
| 2 | FR-0 re-drain + regenerate binding tables | DONE | Commit `fdf1edfc4`; `22161003:` bound in exactly `version_gms_84_1_gen.go`, `version_gms_87_1_gen.go`, `version_gms_92_1_gen.go`, `version_gms_95_1_gen.go`, `version_jms_185_1_gen.go` (verified live via `grep -l`) — matches prd.md §7's "expected" set, so FR-0.5's correction clause was correctly not triggered. |
| 3 | Mist contract gains PROTECTION/RECOVERY kinds | DONE | Commit `d819a2b25`; `services/atlas-maps/atlas.com/maps/mist/processor.go:60-102` (`ErrUnknownKind`, `knownTargetKind`, `knownEffectKind`, rejection wired into `Create`). |
| 4 | `nType` derivation for Smokescreen | DONE | Commit `b72deb1cd`; `AffectedAreaTypeFor` extended, `AffectedAreaTypeSmoke = int32(2)` present in `mist/model.go`. |
| 5 | Character lookup carries HP | DONE | Commit `5c712e721`; `character/processor.go` `Snapshot`, `tasks.CharacterLookup` rename confirmed in `mist_tick.go`. |
| 6 | `tickRecovery` + effect-kind dispatch | DONE | Commit `695e78cb2`(≈)/`695e78cb2`; `mist_tick.go:418-434` — `switch m.EffectKind()` with `EffectKindRecovery`, `EffectKindProtection` (no-op tick, handled by atlas-channel), `EffectKindDisease/""`, and a `default:` warn-log arm for unknown kinds (defense in depth alongside Task 3's create-time rejection). |
| 7 | Full mist-contract mirror + mirror guard | DONE | Commit `d1b84aa0d`; `tools/mist-contract-mirror-guard.sh` present, executable, and passes (`OK: the mist contract mirror matches its owner.`); CLAUDE.md item 14 added. |
| 8 | `mistcast` shared cast helper | DONE | Commit `0feb5255a`; `services/atlas-channel/atlas.com/channel/skill/handler/mistcast/mistcast.go` present with validation constants (`PlayerMistTickIntervalMs`, `MaxPlayerMistDurationMs`) and doc-commented rationale. |
| 9 | Refactor `poisonmist` onto `mistcast` | DONE | Commit `fd0f569ef`; `git diff --stat` shows `poisonmist.go` shrank from a standalone implementation to a 32-insertion/138-deletion diff, consistent with delegating to the shared helper. |
| 10 | Flame Gear + Poison Bomb handlers | DONE | Commit `6f5f01676`; `skill/handler/flamegear/`, `skill/handler/poisonbomb/` packages present, tests pass (`go test ./skill/handler/flamegear/... ./skill/handler/poisonbomb/...` → ok). |
| 11 | Smokescreen handler | DONE | Commit `23e281724`; `skill/handler/smokescreen/` present, tests pass. |
| 12 | Recovery Aura handler + party snapshot | DONE | Commit `ff4a53f4c`; `skill/handler/recoveryaura/` present, tests pass. |
| 13 | Channel-side protection-mist registry | DONE | Commit `4cf842875`; `services/atlas-channel/atlas.com/channel/mist/protection.go`, `protection_test.go` present. |
| 14 | Populate protection registry from `EVENT_TOPIC_MIST` | DONE | Commit `5fcbe5aed`; consumer wiring confirmed (registry package + consumer tests pass). |
| 15 | Smokescreen damage short-circuit | DONE | Commit `7f67276b5`; `socket/handler/character_damage_smoke.go` (`shieldedBySmoke`) and `character_damage.go:44,97,194` (`inProtectiveMist` dependency wired into `processDamageTaken`'s top-of-function short-circuit). |
| 16 | Full verification sweep | DONE | Commit `6c4faade6`; `context.md` §8.1 records the sweep results (tests/vet/build in all four modules, all guards, lint, `docker buildx bake`), matching what this audit independently re-verified (see Build & Test Results below). |

**Completion Rate:** 16/16 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Three items are recorded as intentionally open in `context.md` §8 /
§8.1 (Recovery Aura's WZ `x` absolute-vs-percentage; USE_SKILL registry
confirmation for smokescreen/recoveryaura by live cast; stale live v95
`dotInterval`/`dotTime`) — these require an in-game cast that was not
performed in this session, exactly as instructed. They are correctly still
labeled OPEN in `context.md` §8.1 ("None of this substitutes for a live
cast... Items 1–3 above remain OPEN — no in-game confirmation was performed
in this session"), not silently marked resolved. This is not a plan-adherence
gap.

## Requirement Coverage Verification (spot-checked beyond the plan's own table)

- **FR-0.5** (availability-table correction if the drain diverges from
  expectation): the observed binding set (`gms_84/87/92/95`, `jms_185`)
  matches prd.md §7's "expected gms 84, 87, 92, 95 and jms 185" exactly, so no
  correction was required and none was made — consistent with the rule as
  written.
- **FR-1.4** (data defects reported, never hard-coded around): `design.md`
  §1.5 ("Data defect to report (FR-1.4)") documents the stale v95
  `dotInterval`/`dotTime` values and explicitly states "No fallback is
  hard-coded anywhere in this design." `mistcast.go`'s doc comment for
  `PlayerMistTickIntervalMs` reiterates that no `dotInterval` WZ node is used.
- **FR-7.4/FR-7.5** (blank imports): `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`
  blank-imports `flamegear`, `poisonbomb`, `recoveryaura`, `smokescreen`, each
  tagged `// task-218`.
- **FR-2.5** (reject at both create and tick): confirmed at both call sites —
  `mist/processor.go:96-102` (`Create` returns `ErrUnknownKind`) and
  `tasks/mist_tick.go:418-434` (tick's `switch` has a `default:` warn-and-skip
  arm for an unrecognised `EffectKind`), matching the plan's stated
  "defense in depth" framing.
- **§10 regression — APPLY_STATUS key set frozen**: `applyStatusBody` in
  `mist_tick.go:199-207` still has exactly the pre-task-218 seven fields
  (`SourceType`, `SourceCharacterId`, `SourceSkillId`, `SourceSkillLevel`,
  `Statuses`, `Duration`, `TickInterval`) — no key added, renamed, or
  retyped.
- **§10 regression — Poison Mist path**: `poisonmist.go` is a refactor (32
  insertions / 138 deletions) onto `mistcast`, not a rewrite; its package
  tests pass unmodified in assertions (`go test ./skill/handler/poisonmist/...`
  — covered by the branch-wide test run recorded in `context.md` §8.1 and
  re-confirmed live in this audit for the sibling mist packages).

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-constants` | PASS | not re-run (per instructions, relying on recorded sweep) | `context.md` §8.1 records clean `go test -race`/`go vet`. |
| `libs/atlas-constants/gen` | PASS | not re-run | `go run . -check` → `OK: … up to date` per §8.1. |
| `services/atlas-maps/atlas.com/maps` | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test ./mist/... ./tasks/... ./character/...` → all `ok`. |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | `go build ./...`, `go vet ./...` clean; `go test` on all five new/refactored mist packages plus `socket/handler` → all `ok`. |

Guards re-run live in this audit, all exit 0: `redis-key-guard.sh`,
`goroutine-guard.sh`, `buff-duration-guard.sh`, `skill-job-id-guard.sh`
("clean (14 divergent const(s) checked)"), `trade-contract-mirror-guard.sh`,
`mist-contract-mirror-guard.sh` ("OK: the mist contract mirror matches its
owner."). `docker buildx bake atlas-maps atlas-channel` was not re-run in this
audit (per the task brief, already verified clean at `6c4faade6` per
`context.md` §8.1) — no reason to doubt it given `go build`/`go vet` are clean
and no `go.mod` changed.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. No fixes required before this plan can be considered complete. The
three explicitly-open items in `context.md` §8/§8.1 are correctly labeled as
requiring a live in-game cast and are not blocking — they are follow-up
confirmations, not implementation gaps.

---

# Backend Guidelines Audit — task-218-player-cast-mists

**Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
**Date:** 2026-08-12
**Scope:** Hand-written Go changed by this branch (see task brief file list). Generated `libs/atlas-constants/**/*_gen.go` and `gen/wzsnapshot/*.json` excluded per instructions.
**Mindset:** FAIL until file:line evidence proves PASS.

## Domain Package Checklist — `atlas-maps/mist` (has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists as a separate file | **FAIL** | `Builder`/`NewBuilder`/`Build()` are defined inside `services/atlas-maps/atlas.com/maps/mist/model.go:338-497`, not in a `builder.go`. Also `Build()` performs no invariant validation (file-responsibilities.md: "`Build()` enforces invariants") — it is a bare struct literal copy. |
| DOM-02 | `ToEntity()` method | **FAIL** | No `entity.go` in the package (`find services/atlas-maps/atlas.com/maps/mist -type f` → `model.go, model_test.go, processor.go, processor_test.go, producer.go, producer_test.go, registry.go, registry_test.go`). No `ToEntity()` anywhere in the package. |
| DOM-03 | `Make(Entity)` function | **FAIL** | Same absence — no `entity.go`, no `Make(`. |
| DOM-04/05 | `Transform`/`TransformSlice` in `rest.go` | N/A | No REST surface for mists (Kafka-only domain); no `rest.go`. Not a finding — nothing to transform to JSON:API. |
| DOM-10 | Test DB has tenant callbacks | N/A | Package has no DB persistence to test. |
| DOM-16 | `administrator.go` for writes | **FAIL** | Mist lifecycle writes (`Registry.Add`/`Remove`, `mist/registry.go:75-104`) go straight to the in-memory `Registry`, with no `administrator.go` — because there is no backing store at all. Mist state is 100% in-process memory (`sync.Once` singleton `Registry`/`ProtectionRegistry`), so a channel or atlas-maps pod restart silently drops every live mist (acknowledged in `mist/protection.go:17-24`'s own doc comment, but that acknowledgment is a design tradeoff note, not a guideline exception). |

These four are mechanical, binary findings against the checklist as written. The package's own doc comments explain the in-memory design is deliberate (avoiding a synchronous REST round-trip on the damage path, bounded restart-gap risk), but per the audit's stated rule only the backend-dev-guidelines resources themselves can exempt a deviation, and none of them documents an "ephemeral in-memory domain" exception to DOM-02/03/16. Reported as findings; whether the team wants to formalize an exception for non-persisted domains is a guideline-authoring question, not something this audit can adjudicate.

## File Responsibilities Checklist

| ID | Package | Check | Status | Evidence |
|----|---------|-------|--------|----------|
| FILE-06 | `atlas-maps/mist` | No package/catch-all file bundling ≥2 responsibilities | **FAIL** | `mist/model.go` bundles the domain **Model** (`Mist` struct + accessors, lines 20-336) *and* the **Builder** (`Builder` struct + fluent setters + `Build()`, lines 339-497) — two of the guideline's distinct file roles (`model.go` vs `builder.go`) collapsed into one file. |
| FILE-06 | `atlas-channel/mist` | No package/catch-all file bundling ≥2 responsibilities | **FAIL** | `mist/protection.go` bundles a domain **Model** (`Protection` struct + accessors, lines 25-60), its **Builder** (`ProtectionBuilder`, lines 62-89), and a **Registry/singleton-cache** (`ProtectionRegistry`, `GetProtectionRegistry`, lines 91-179) — three distinct file-responsibility roles in one file. This is the same shape of collapse the audit brief calls out by name (task-102 `wallet.go`), just with Model+Builder+Registry instead of Processor+RestModel+requests. |
| FILE-01 | `atlas-channel/mist` | `Processor` in `processor.go` | PASS | `mist/processor.go:13,17,22` — interface, impl, and constructor all in `processor.go`. |
| FILE-01 | `atlas-maps/mist` | `Processor` in `processor.go` | PASS | `mist/processor.go:22,30,40` |
| FILE-03 | `atlas-maps/character` | cross-service request funcs in `requests.go` | PASS | `character/requests.go:18-20` (`requestById`) |
| FILE-02 | `atlas-maps/character` | `RestModel`+JSON:API methods in `rest.go` | PASS (partial — see EXT-01) | `character/rest.go:11,19,24,29` |

## External HTTP Client Checklist — `atlas-maps/character` (calls atlas-character via `requests.GetRequest[T]`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | Target REST model implements both `SetToOneReferenceID` and `SetToManyReferenceIDs` | **FAIL** | `character/rest.go` defines `SetToManyReferenceIDs` (line 39) but has **no `SetToOneReferenceID`** anywhere in the package. Per `libs/atlas-rest/CLAUDE.md` (cited by the guideline as the source of the task-037 bug class), api2go errors on any response carrying a `relationships` block if only one of the two methods is implemented. This file is in the branch's explicit scope (`character/{rest,processor}.go`) and was touched by commit `5c712e721`, so it is in-scope for this audit even though the gap predates task-218 (traced back to `92664c915`, task-036). |
| EXT-02 | httptest-backed integration test | PASS | `character/processor_test.go:56-119` — three tests (`TestProcessor_Snapshot_ReturnsCoordinatesFromAtlasCharacter`, `TestProcessor_Snapshot_PropagatesNotFound`, `TestSnapshot_ProjectsPositionAndHp`) stand up `httptest.NewServer`, serve a representative JSON:API fixture (including the new `hp` field added by this branch), and assert the client's `Snapshot` method returns a populated, correctly-typed result. |
| EXT-03 | 404 vs other failures distinguished | PASS | `character/processor_test.go:80-94` asserts `require.ErrorIs(t, err, requests.ErrNotFound)` specifically on a 404 response; the client does not swallow other errors into "not found". |
| EXT-04 | URL via `requests.RootUrl`, not hardcoded | PASS | `character/requests.go:16` — `requests.RootUrl("CHARACTERS")`. |

## Migration / Re-export Hygiene (CLAUDE.md "Code Patterns" / ai-guidance.md "No Type Aliases During Migrations")

| Check | Status | Evidence |
|----|--------|----------|
| No re-export/alias left behind after moving shared logic to `mistcast` | **FAIL** | `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist.go:36-39`: `const ( PlayerMistTickIntervalMs = mistcast.PlayerMistTickIntervalMs; MaxPlayerMistDurationMs = mistcast.MaxPlayerMistDurationMs )`. This branch's own Task 8/9 (per `audit.md`'s plan-adherence section) moved these constants' ownership from `poisonmist` to the new shared `mistcast` package — but `poisonmist` still re-declares both names as thin aliases "because this package's tests... assert against them by these names" (poisonmist.go:32-35). CLAUDE.md's Code Patterns section says "prefer straightforward moves over re-exporting type aliases," and ai-guidance.md's Migration & Refactoring Rules is explicit: "Never leave type aliases..., re-exports, or thin wrappers that just delegate... update ALL service call sites to import from the new library directly." The correct fix is updating `poisonmist_test.go`'s six references (lines 107, 121, 125, 131, 136 and the doc comment) to `mistcast.PlayerMistTickIntervalMs` and deleting the alias block, exactly as `flamegear`/`poisonbomb`/`smokescreen`/`recoveryaura` already do (none of the four new handlers re-declare these constants — only the refactored `poisonmist` kept a copy). |

## Concurrency / Multi-tenancy Review (emphasis area)

| Area | Status | Evidence |
|----|--------|----------|
| `atlas-channel/mist.ProtectionRegistry` — tenant-keyed, RWMutex-guarded, consumer-written / damage-path-read | PASS | `mist/protection.go:93-179` — `sync.RWMutex` guards all access; `Add`/`Remove` take the write lock, `Covering`/`Len` take the read lock; keyed by `t.Id().String()` so tenants cannot cross-contaminate. `Covering` treats expired-but-unpruned entries as absent (line 165: `p.Expired(now)` checked at read time), so a dropped `MIST_DESTROYED` degrades to "no protection" rather than a permanent shield — correctly documented and implemented. |
| `atlas-maps/mist.Registry` — tenant-keyed, RWMutex-guarded, tick-task read/write | PASS | `mist/registry.go:32-165` — same `sync.RWMutex` pattern; `UpdateLastTick`/`Remove` take the write lock. |
| `tasks/mist_tick.go` concurrency (per-tenant + per-mist fan-out via `routine.Go`) | PASS | `runOnce` (line 351) fans out one `routine.Go` per tenant; `processTenant` (line 380) fans out per-mist with a bounded semaphore (`mistTenantConcurrency = 4`, line 93); both use `routine.Go`, not a bare `go` statement — satisfies DOM-26. `producer.Provider` sharing across a tenant's mist workers is explicitly justified in the doc comment (lines 371-379) by citing the underlying writer manager's own mutex. |
| Tenant resolution | PASS | `mist/processor.go:54` — `tenant.MustFromContext(ctx)`; `tasks/mist_tick.go:381` — `tenant.WithContext(ctx, t)` before dispatching REST/Kafka calls per tenant. |
| Goroutine spawning (DOM-26) | PASS | No bare `go` statements found in the scoped diff; `tools/goroutine-guard.sh` reported clean per the task brief's pre-verified sweep, and manual inspection of `tasks/mist_tick.go` confirms `routine.Go(r.l, ctx, func(...) {...})` at lines 357 and 391. |

## Kafka Contract Mirror (emphasis area)

| Check | Status | Evidence |
|----|--------|----------|
| `tools/mist-contract-mirror-guard.sh` byte-for-byte match | PASS | Verified independently: `diff` of the two `kafka.go` files from their `package mist` line onward is empty (identical). |
| Guard coverage sufficiency | **Note, not a finding** | The guard only diffs `kafka/message/mist/kafka.go` in each module. Nothing else duplicates the `CreateCommandBody`/`CreatedBody`/`DestroyedBody` shapes outside that one file pair on either side (`grep -l` across both service trees confirms all other `CreateCommandBody`-adjacent hits are either the two contract files themselves, their tests, or genuinely distinct local envelope structs for *other* contracts, e.g. atlas-maps' `tasks/mist_tick.go` locally mirrors the **buffs/monster/character** contracts, not the mist one — out of this guard's declared scope). Given that, the guard's single-file diff is sufficient for the mist contract specifically; it would not catch drift in those other locally-mirrored contracts, but those pre-date task-218 and are out of scope. |
| DOM-24 (Kafka producer stubbed in emitting tests) | PASS | `mist/processor_test.go` (atlas-maps) injects a `recordingProducer` implementing `producer.Provider` directly into `ProcessorImpl` — never touches the real `producer.ProviderImpl`/`ConfigWriterFactory` path. `mistcast_test.go`, `poisonmist_test.go`, `poisonbomb_test.go`, `flamegear_test.go`, `smokescreen_test.go`, `recoveryaura_test.go` all drive `mistcast.Cast` via injected `Seams{LoadCaster, EmitCreate}` recording funcs — no real Kafka path exercised. `kafka/consumer/mist/consumer_test.go` stubs both `affectedAreaCreatedBroadcaster` and `affectedAreaRemovedBroadcaster` (lines 34, 38) rather than driving the real broadcast/producer path. No unstubbed emit path found in the scoped test files. |

## SEC — Client Wire Value Discipline (DOM-25 emphasis area)

| Check | Status | Evidence |
|----|--------|----------|
| Smokescreen damage short-circuit uses server state only | PASS | `socket/handler/character_damage.go:194` — `deps.inProtectiveMist(f, characterId, c.X(), c.Y())` reads position from the server-side `character.Model` (`c.X()`, `c.Y()`), not from the client-reported packet. `character_damage_smoke.go:61` resolves ownership via the channel-local `ProtectionRegistry` (populated only from `EVENT_TOPIC_MIST`, a server-originated Kafka event) plus a live party-membership REST call — no client-supplied byte feeds the shield decision. |
| `nType` (`AffectedAreaTypeFor`) derived server-side | PASS | `mist/model.go:160-168` — computed from `ownerType`/`effectKind`, both server-known values; not carried on the command from a client packet. |
| Recovery Aura / mist magnitudes are WZ- or target-derived, never client wire bytes | PASS | `recoveryaura.go:109-119` — magnitude from `e.X()` (WZ effect data); `mistcast.go:169-176` — `DiseaseValue: 0`, explicitly documented as target-derived (`monster.ResolvePoisonDamage`), not client-supplied. |

## Summary

### Blocking (must fix)
- DOM-01/FILE-06: `services/atlas-maps/atlas.com/maps/mist/model.go` bundles Model + Builder; split `Builder`/`NewBuilder`/`Build()` into `builder.go` and have `Build()` enforce the invariants `mistcast.Cast` currently enforces ad hoc on the caller side (non-degenerate rect, positive duration, etc.), or provide a guideline-level exception for ephemeral in-memory domains.
- DOM-02/DOM-03/DOM-16: `atlas-maps/mist` has no `entity.go`/`Make`/`ToEntity`/`administrator.go` — either accept the in-memory-only design as a formally documented guideline exception, or note the restart-loses-all-mists behavior as a tracked risk (currently only self-documented in a code comment, not a guideline waiver).
- FILE-06: `services/atlas-channel/atlas.com/channel/mist/protection.go` bundles Model + Builder + Registry/singleton-cache; split into `model.go`/`builder.go`/`cache.go` (or `registry.go`) per file-responsibilities.md, or formally justify the collapse.
- EXT-01: `services/atlas-maps/atlas.com/maps/character/rest.go` is missing `SetToOneReferenceID` (no-op is sufficient) — add it to prevent api2go relationship-block decode failures (the exact task-037 bug class).
- Migration hygiene: `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist.go:36-39` re-declares `PlayerMistTickIntervalMs`/`MaxPlayerMistDurationMs` as aliases of `mistcast`'s constants instead of updating `poisonmist_test.go`'s six call sites to reference `mistcast.` directly and deleting the alias — violates the explicit "no re-exports/thin wrappers after a move" rule in ai-guidance.md and CLAUDE.md.

### Non-Blocking (should fix)
- DOM-20: Several new/touched test files (`mist/model_test.go`, `mist/processor_test.go` in atlas-maps; `mist/protection_test.go`, `smokescreen_test.go` in atlas-channel) use discrete `Test_X` functions rather than the table-driven `tests := []struct{...}{}` + `t.Run` pattern. Coverage itself looks adequate (edge cases are each their own named test), so this is style, not a gap — flagged for completeness against the checklist, not blocking.
