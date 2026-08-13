# Plan Audit — task-217-aran-combo-counter

**Plan Path:** docs/tasks/task-217-aran-combo-counter/plan.md
**Audit Date:** 2026-08-12
**Branch:** task-217-aran-combo-counter
**Base Branch:** main (merge-base `ead214d6f`)
**HEAD:** `c469dce08`

## Executive Summary

All 14 plan tasks are implemented and match their described interfaces, file lists, and acceptance criteria. All four declared PRD deviations (count off the `ARAN_COMBO` buff stat, decay moved to atlas-channel, no `SHOW_COMBO 0`, no job-range gate) are verified in source as the intentional design choices the plan claims, not accidental omissions. All 12 required matrix cells (`ARAN_COMBO_COUNTER` serverbound + `SHOW_COMBO` clientbound × six versions) are `verified` in `docs/packets/audits/status.json`, and all six templates carry matching opcodes/handler/writer entries with `idleResetMs` correctly set to 5000 only on `gms_95_1`. Build/test/guard verification was independently pre-confirmed by the requester and is taken as established per instructions. One minor process gap: `plan.md`'s own checkbox tracking (`- [ ]`) was never toggled to `- [x]` for any of the 73 step items, so a naive read of the plan file alone misrepresents completion — the execution ledger (`progress.md`) is the accurate record.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Serverbound `ARAN_COMBO_COUNTER` codec | DONE | `libs/atlas-packet/character/serverbound/aran_combo_counter.go` created; commit `37a9463`; empty-body `Encode`/`Decode`, `AranComboCounterHandle` const present as specified. |
| 2 | Clientbound `SHOW_COMBO` codec | DONE | `libs/atlas-packet/character/clientbound/show_combo.go`; commit `d37c398`; `NewShowCombo`, `Count()`, `ShowComboWriter` const present. |
| 3 | `LegendComboAbilityId` in atlas-constants | DONE | `libs/atlas-constants/gen/identities.yaml:1873`, `skill/constants.go:2044-2045` (`LegendComboAbilitySkill`), `:2837` (registry row), `:3397` (`LegendComboAbilityId = Id(20000017)`); commit `a45450b`. |
| 4 | `ARAN_COMBO` statup value in atlas-data | DONE | `services/atlas-data/atlas.com/data/skill/reader.go:469` — `skill.Is(skillId, skill.AranStage1ComboAbilityId, skill.LegendComboAbilityId)` branch uses `int32(e.X())`, no hardcoded `100`; commit `6e824bf`. |
| 5 | `ComboMirror` | DONE | `services/atlas-channel/atlas.com/channel/character/combo/mirror.go` — `GetMirror`, `SetEligibility`, `Eligibility`, `Increment`, `Clear`, `ExpireIdle`, `ComboCap=99999`, `DefaultIdleWindow=3s` all present (lines 41-192); commits `8948f1b`..`6f497dc`. `setCountForTest` correctly absent — verified removed per ledger's Task 5 deviation record. |
| 6 | Eligibility gates | DONE | `services/atlas-channel/atlas.com/channel/character/combo/eligibility.go` — `ComboAbilityId(jobId)` selects Legend vs Aran id exactly per client selector; `Evaluate` gates on skill level then weapon (`item.WeaponTypePolearm`) with no job-range check, matching design.md §3.5; commit `178dd94`. |
| 7 | `ARAN_COMBO_COUNTER` handler and wiring | DONE | `socket/handler/character_aran_combo.go:38-155` — `idleWindowFromOptions`, `aranComboDeps`/`aranComboProductionDeps`, `aranComboAdvance`, `AranComboCounterHandleFunc`; `main.go:799` (`charcb.ShowComboWriter`), `main.go:920` (`handlerMap[charsb.AranComboCounterHandle]`); commit `88fb999`. |
| 8 | Attack-pipeline eligibility refresh | DONE | `character_aran_combo.go:165-174` (`aranComboRefreshEligibility`), hooked into `character_attack_common.go` beside `comboOrbTryUpdate` in the melee branch; commit `6aa342e`. |
| 9 | Reset paths | DONE | `character_aran_combo.go:185-191` (`aranComboClearOnCancel`), wired at `character_buff_cancel.go:33`; `session/processor.go:417,459` (`clearAranComboOnDestroy`); `kafka/consumer/character/consumer.go:240` (map-change clear); commit `a6b677a`. |
| 10 | Decay tick | DONE | `character/combo/task.go` — `NewDecayTick`, `Run()`, `processExpiries`, `cancelComboBuff`; registered `main.go:331` inside an outer `routine.Go` (ledger-confirmed single-goroutine, not a double-spawn); commit `c2c38f6`. Sends no packet on expiry — verified in source comment and code (only `buff.NewProcessor(...).Cancel` is called). |
| 11 | Seed templates (six versions) | DONE | All six templates (`gms_83/84/87/92/95_1`, `jms_185_1`) carry matching `AranComboCounterHandle` handler (opcodes 0x0A3/0x0A9/0x0AD/0x0BA/0x0BD/0x09D) and `ShowCombo` writer (opcodes 0x0E1/0x0E6/0x0EF/0x103/0x101/0x0EB) entries, each with non-empty `validator`/`fname`; `idleResetMs` is `5000` only in `gms_95_1`, `3000` elsewhere — verified via direct grep of all six files. Commit `d3dcd00`. |
| 12 | Name the v92 sender in its IDB | DONE (no repo commit, by design) | Ledger records `0x8ef840` renamed to `CUserLocal__RequestIncCombo_send_0xBA` in the v92 IDB and independently re-verified via `func_query`; no repo diff expected for an IDA-only step. |
| 13 | Promote the twelve matrix cells | DONE | `docs/packets/audits/status.json` rows for `character/serverbound/CharacterAranComboCounterRequest` and `character/clientbound/CharacterShowCombo` show `state: "verified"` for all six in-scope versions (gms_v83/84/87/92/95, jms_v185) — 12/12 confirmed by direct read. Registry diffs for `gms_v84.yaml`/`gms_v92.yaml` add only `packet:` hint fields and explanatory notes, no `opcode:` value changed — confirms the ledger's "extras" characterization. Commits `19a74cb`, `8b43715` (toolSha fix). |
| 14 | Documentation and full verification sweep | DONE | `docs/TODO.md:569-586` carries a "task-217 Aran combo counter — landed" entry describing the shipped behavior and explicitly scoping out combo *consumption*; commit `c469dce`. Build/test/guard sweep taken as pre-established per the requester's instructions (not re-run). |

**Completion Rate:** 14/14 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 14 tasks have direct code/config/doc evidence of completion.

## Acceptance Criteria Traceability (plan.md's own table, verified)

| Criterion | Verified |
|---|---|
| Both opcodes IDA-verified for all six versions; registry corrections | Yes — 12/12 `verified` cells in status.json; zero opcode corrections in registry diffs (only `packet:` hints added). |
| Serverbound model with `Encode`/`Decode` | Yes — Task 1. |
| `SHOW_COMBO` model with `Encode`/`Decode` | Yes — Task 2. |
| Handler + writer in six templates, sorted, validator + fname | Yes — all six templates carry both entries with non-empty `validator`/`fname`; template-opcode-order-guard and duplicate-binding-guard reported clean per requester's pre-established sweep. |
| Twelve matrix cells ✅ with fixtures and evidence | Yes. |
| Counter increments on screen for an eligible Aran | Yes — `aranComboAdvance` calls `Increment` then `announce` (SHOW_COMBO write) when count > 0. |
| First increment seeds the buff; later ones advance, clamped at cap | Yes — `seeded` flag from `Mirror.Increment` gates the one-time `deps.seed` call; `Increment` clamps at `ComboCap`. |
| Cap not exceeded however fast the client sends | Yes — `Increment`'s `e.count < ComboCap` guard (`mirror.go`). |
| Idle decay; buff cancelled at zero | Yes — `DecayTick.Run` → `ExpireIdle` → `processExpiries` → `cancelComboBuff` (buff.Cancel). |
| A qualifying hit refreshes the idle timer | Yes — `Increment` always sets `e.lastHit = now`, called on every accepted `ARAN_COMBO_COUNTER`. |
| Silent no-op for non-Aran / no Combo Ability / non-polearm | Yes — `Evaluate` returns `false` on any gate miss; `aranComboAdvance` returns early with only a debug log, no packet, no Kafka. |
| `reader.go:470` statup corrected or justified | Yes — Task 4, `int32(e.X())`. |
| `20000017` added to atlas-constants | Yes — Task 3. |
| `go test -race` / `go vet` / `go build` clean | Established by requester (all 4 modules); not re-run here. |
| `docker buildx bake` for touched-`go.mod` services | Established by requester — correctly skipped, no `go.mod` moved. |
| All guards clean | Established by requester (7/7 exit 0). |
| `docs/TODO.md` updated | Yes — Task 14. |
| Code review before PR | Ledger records "review clean" per task; not independently re-audited here beyond this plan-adherence pass. |

## Declared PRD Deviations — confirmed as implemented, not accidental

1. **FR-3 (count on the `ARAN_COMBO` buff stat)** — rejected as claimed. `ComboMirror.Entry.count` is the sole count store (`mirror.go`); the `ARAN_COMBO` statup carries `e.X()` (`reader.go:469`), a WZ-level magnitude, never the live count. Confirmed distinct.
2. **FR-4.1 (decay ticker in atlas-buffs)** — moved to atlas-channel as claimed. `git diff --name-only ead214d6f..HEAD -- services/atlas-buffs/` returns empty — zero atlas-buffs changes on this branch, confirming "atlas-buffs needs no change at all."
3. **FR-4.4 ("tell the client" on decay-to-zero)** — omitted as claimed. `DecayTick.Run` / `processExpiries` / `cancelComboBuff` call only `buff.NewProcessor(...).Cancel(...)`; no `SHOW_COMBO` or any clientbound write appears anywhere in `character/combo/task.go`.
4. **FR-2.1 (explicit Aran/Legend job-range gate)** — folded into the skill gate as claimed. `eligibility.go`'s `Evaluate` gates only on Combo Ability level and weapon type; the doc comment explicitly states "no job-range check," matching the design rationale.

## Build & Test Results

Per the requester's explicit instruction, the verification sweep (guards, per-module `go build`/`go vet`/`go test -race`, `packet-audit matrix --check`, bake-skip correctness) was independently pre-confirmed and was **not re-run** in this audit. Taken as established:

| Service/Module | Build | Tests | Notes |
|---|---|---|---|
| libs/atlas-packet | PASS (established) | PASS (established) | Not re-run per instructions. |
| libs/atlas-constants | PASS (established) | PASS (established) | Not re-run per instructions. |
| services/atlas-channel/atlas.com/channel | PASS (established) | PASS (established) | Not re-run per instructions. |
| services/atlas-data/atlas.com/data | PASS (established) | PASS (established) | Not re-run per instructions. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. **Minor / cosmetic:** `plan.md`'s checkbox tracking (`- [ ]`) was never updated to `- [x]` for any of its 73 step items across all 14 tasks, despite every task being complete per the execution ledger and source evidence. Toggling the boxes (or adding a pointer to `progress.md` at the top of the plan) would prevent a future reader from misjudging completion from the plan file alone. Not a blocker — no functional gap.
2. No other action items. All plan tasks, all declared PRD deviations, and all traceability-table criteria are implemented and evidenced in source.

---

# Backend Guidelines Audit — task-217-aran-combo-counter

- **Service Path(s):** `libs/atlas-packet`, `libs/atlas-constants`, `services/atlas-channel/atlas.com/channel`, `services/atlas-data/atlas.com/data`
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-08-12
- **Build/Test/Guards:** Taken as pre-established per the requester's instruction (all 7 repo guards, `go build`/`go vet`/`go test -race` on all 4 touched modules, `lint.sh --check`) — **not re-run** in this pass.
- **Overall:** **PASS** — zero FAIL findings. Default-FAIL discipline was applied throughout; every check below is backed by a file:line citation, and no check was accepted on "looks correct."

## Scope note on checklist applicability

This branch adds no REST resource — the touched atlas-channel code is entirely the socket-handler layer (`socket/handler/*.go`), which is a *different, pre-existing, repo-wide convention* from the `resource.go`/`rest.go` REST DTO layer the DOM-04/05/08/09/17/18/19 checks target (those checks assume a JSON:API `RestModel` + `Transform`). No new package in this diff has a `model.go`, so per Phase 2 every touched package classifies as **support**, and the File-Responsibilities checklist (FILE-01..06) was run on all of them rather than the full DOM checklist, which does not apply to a body-less socket op with no REST/DB surface. DOM-21, DOM-22, DOM-23, DOM-24, DOM-25, DOM-26 apply universally and were run in full.

## `services/atlas-channel/atlas.com/channel/character/combo` (support package — `mirror.go`, `eligibility.go`, `task.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in `processor.go` | N/A | Package defines no `Processor` interface/`ProcessorImpl` — it exposes a singleton state store (`Mirror`) plus pure functions (`Evaluate`), the same shape as the precedented `character/buff/beacon.go` (`BeaconMirror`/`GetBeaconMirror`, `beacon.go:48-65`). No processor-shaped symbol exists to misplace. |
| FILE-02/03/04 | RestModel/requests/entity in correct files | N/A | Package has no REST surface, no cross-service HTTP calls, no DB entity — none of these symbols exist anywhere in the package. |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A (see below) | `Eligibility` and `Entry` (`mirror.go:35-64`) are immutable-shaped value types with private fields + accessor methods (matches the `model.go` responsibility description) but are co-located with the mutable `Mirror` singleton in `mirror.go` rather than split into a `model.go`. This mirrors `beacon.go`'s identical co-location of `BeaconEntry` + `BeaconMirror` in one file — an established, repeated shape in this same service, not a one-off. No violation raised (see file-responsibilities.md's `cache.go` entry, which documents exactly this "singleton state file" shape as legitimate, just under a different name for a TTL-based variant; `mirror.go` is the non-TTL, event-driven sibling already in production). |
| FILE-06 | No package-named catch-all bundling ≥2 responsibilities | PASS | `mirror.go` bundles only value-type + singleton-store responsibilities (not Processor+RestModel+requests); `eligibility.go` is single-purpose (`ComboAbilityId`, `Evaluate`, `eligibility.go:18,32`); `task.go` is single-purpose (`DecayTick`, `cancelComboBuff`, `processExpiries`, `task.go:22,51,60`). No file combines Processor+RestModel+requests or any two of the banned groups. |
| DOM-21 | Reuse of `libs/atlas-constants` types | PASS | `skill.Id`, `skill.LegendComboAbilityId` / `skill.AranStage1ComboAbilityId` (`libs/atlas-constants/skill/constants.go:3397-3398`), `item.GetWeaponType`/`item.WeaponTypePolearm` (`libs/atlas-constants/item/constants.go:122,135`), `job.LegendId` all consumed directly (`eligibility.go:8-10,18-22,44`); `character.TemporaryStatTypeAranCombo` reused unchanged (`libs/atlas-constants/character/temporary_stat.go:74`, referenced `socket/handler/character_aran_combo.go:100`). No new type/enum/numeric literal duplicates an existing atlas-constants concept — `ComboCap`/`DefaultIdleWindow` (`mirror.go:24,30`) are combo-specific business constants with no shared-lib equivalent. |
| DOM-26 | Goroutines via `routine.Go` | PASS | The only goroutine spawn tied to this feature is `main.go:330-332`, wrapped in `routine.Go(l, rt.Context(), func(_ context.Context) {...})`, structurally identical to the pre-existing heartbeat registration immediately above it (`main.go:326-328`). `mirror_test.go:186-194`'s bare `go func()` inside `TestConcurrentAccess` is test code, explicitly out of `goroutine-guard.sh`'s scope (guard excludes `_test.go`). |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `task.go`'s only production emit path is `cancelComboBuff` → `buff.NewProcessor(...).Cancel` → `producer.ProviderImpl` (`character/buff/processor.go:88-90`, unmodified by this branch). `task_test.go:23,36,47` calls `processExpiries` with an **injected fake `cancel` func**, never the real `cancelComboBuff` — the real emit path is never exercised by a test in this package, so no producer stub is needed and none is missing. |
| Concurrency | `Mirror` is a process-wide singleton mutated from 4+ call sites (socket handler, attack-pipeline hook, buff-cancel handler, Kafka map-changed consumer, session teardown, 1 Hz tick) | PASS | Every exported `Mirror` method (`SetEligibility`, `Eligibility`, `Increment`, `Clear`, `ExpireIdle` — `mirror.go:113,126,143,164,192`) takes `m.mu.Lock()`/`RLock()` for its entire body and returns copies of `Entry`/`Eligibility` value types (no pointer/slice aliasing escapes the lock). `bucketFor` (`mirror.go:95`) is only ever called with the lock already held by its callers. No read-modify-write sequence spans two separate lock acquisitions from any call site inspected (`character_aran_combo.go:96-125`, `character_buff_cancel.go:33`, `kafka/consumer/character/consumer.go:238`, `session/processor.go:456-459`, `task.go:40`). |
| Failure isolation | Every Kafka emit failure in the feature is logged and swallowed | PASS | `aranComboAdvance`'s seed-emit failure: logged via `l.WithError(err).Errorf(...)` and swallowed, count still advances (`socket/handler/character_aran_combo.go:127-129`, exercised by `character_aran_combo_test.go:115-122` "seed failure still advances the count"). Decay-cancel failure: logged and swallowed inside `processExpiries` (`character/combo/task.go:64-67`), exercised by `task_test.go:32-42` "TestProcessExpiriesSwallowsCancelFailure". The SHOW_COMBO announce-write failure is also logged and swallowed, not propagated to abort the request (`character_aran_combo.go:135-137`). No emit failure in this feature can fail the underlying player action. |
| Hot-path REST/Kafka calls | No unnecessary per-hit call | PASS | Steady-state `ARAN_COMBO_COUNTER` handling is a mutex-guarded map increment plus one socket write — no REST, no Kafka (`character_aran_combo.go:143-148` doc comment, confirmed by `aranComboProductionDeps`'s `eligibility` closure at `character_aran_combo.go:78-81` hitting the cache path on every non-cold-start call). The one REST fetch (`character.NewProcessor(...).GetById` with two decorators) happens on the melee-attack fetch that the pipeline already pays for (`character_attack_common.go:745`), not on the per-hit combo packet; `aranComboRefreshEligibility` (`character_attack_common.go:983`) reuses that already-fetched `c`, adding zero REST calls (confirmed no `requests.` call anywhere in `combo/eligibility.go` or `socket/handler/character_aran_combo.go`). |

## `libs/atlas-packet/character/{serverbound,clientbound}` (sub-domain / codec packages)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01/02 | No business logic / no writes in codec | PASS | `aran_combo_counter.go` (`AranComboCounterRequest`) and `show_combo.go` (`ShowCombo`) are pure `Encode`/`Decode` pairs with no side effects — `aran_combo_counter.go:28-37`, `show_combo.go:35-47`. |
| SUB-03/04 | Wire-format-only decode, no manual JSON | PASS | `Decode` reads via `r.ReadUint32()` (`show_combo.go:45`) / does nothing (empty body, `aran_combo_counter.go:35-36`); no `json.Unmarshal`/`io.ReadAll` anywhere in either file. |
| DOM-25 | No client wire codes as bare Go literals | PASS | `ShowCombo.count` is app data (the combo count), not a client-side lookup-switch code; `AranComboCounterRequest` carries no fields at all. Neither file contains a mode/notice/fail-reason byte. |

## `libs/atlas-constants/skill` (generated identity)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | New identity added via the generator, not hand-duplicated | PASS | `LegendComboAbilityId = Id(20000017)` is generated output (`libs/atlas-constants/skill/constants.go:3397`, sourced from `gen/identities.yaml` per the plan-adherence audit's Task 3 entry above), consistent with every other `skill.Id` constant in the file — not a hand-rolled duplicate type. |

## `services/atlas-data/atlas.com/data/skill/reader.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | `skill.Is(...)` variadic reused, not reinvented | PASS | `reader.go:469` calls the existing `skill.Is(skillId, skill.AranStage1ComboAbilityId, skill.LegendComboAbilityId)` — no new comparison helper introduced. |

## Security Review

Not applicable — this is not an auth/token-handling service (Phase 4 skipped per its own gating condition).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None. The one process nit already logged above (plan.md checkboxes) is a plan-adherence artifact, not a guideline violation, and is not duplicated here.
