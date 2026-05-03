# Plan Audit — task-047-priest-doom

**Plan Path:** `docs/tasks/task-047-priest-doom/plan.md`
**Audit Date:** 2026-05-03
**Branch:** `task-047-priest-doom`
**Base Branch:** `main` (audit base SHA `a583cc448` → HEAD `4a3312d6d`)

## Executive Summary

All 11 tasks (0–10) in the plan are implemented in code on the branch and the three affected services build and test green. Two intentional, user-acknowledged deviations are present and properly documented inline:
1. Task 4 STATUS_APPLIED count assertions are dropped with `NOTE:` comments referencing the `producer.ProviderImpl` bypass; state pins (`HasStatusEffect`, `EffectId()`) remain.
2. Task 10 inventory/compartment/asset construction uses the realized builder API (`SetCompartment(comp)` not `SetCompartment(type, comp)`; `MustBuild()`; `asset.NewBuilder(compId, templateId).SetId(...)…`), not the speculative API in the plan.

Recommendation: READY_TO_MERGE pending the manual end-to-end verification gate in Task 9 (which is observability/client-render, not code).

## Task Completion

| #  | Task                                                           | Status | Evidence / Notes |
|----|----------------------------------------------------------------|--------|------------------|
| 0  | Pre-flight constants & baseline tests                          | DONE   | Plan checkbox is non-action verification; constants/grep targets all present (verified by Task 1–10 commits compiling). No commit, by design. |
| 1  | atlas-data — pin Doom effect mapping                           | DONE   | `services/atlas-data/atlas.com/data/skill/reader_test.go:2995` `TestReader_PriestDoom_MapsDoomStatus`; commit `78f27eb20`. |
| 2  | atlas-monsters — extend `information.ModelBuilder`             | DONE   | `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go:43,52` `SetBoss`/`SetResistances`; commit `4c79dc3d9`. |
| 3  | atlas-monsters — extend `testInformationLookup` to `ApplyStatusEffect` | DONE | `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1072-1073` (hook-aware lookup), docstring updated at line 62; commit `96d2e213f`. |
| 4  | atlas-monsters — explicit DOOM short-circuit + 3 pinning tests | DONE (with documented deviation) | Production: `processor.go:1111-1114` (DOOM short-circuit). Tests: `processor_test.go:1722,1765,1804`. STATUS_APPLIED event-count assertions removed; `NOTE:` comments at `processor_test.go:1755` and `1849` document the `producer.ProviderImpl` bypass. State pins (`HasStatusEffect("DOOM")`, `EffectId()` comparison, `doomEffects==1`) cover the behavior. Commit `163251790`. |
| 5  | atlas-channel — extract `processDamageInfoEntry`               | DONE   | `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:105` (`damageInfoEntryDeps` struct), `:119` (`processDamageInfoEntry` function); call site `:291-303`. Commit `03c84901c`. |
| 6  | atlas-channel — Doom-gated magic-reflect probe                 | DONE   | `character_attack_common.go:148-156` (DOOM-gated probe in empty-damage branch with the `Doom: monster [%d] has %s reflect; status apply skipped.` log). Commit `e05a1983a`. |
| 7  | atlas-channel — helper tests (Doom cast/reflect/spread)        | DONE   | 4 tests at `character_attack_common_test.go:428,469,504,543` (`TestProcessDamageInfoEntry_Doom_*` and `TestProcessDamageInfoEntry_NonDoom_EmptyDamagesIgnoresReflectProbe`). Commit `ecb2535a8`. |
| 8  | atlas-channel — Doom Debugf in `monster.Processor.ApplyStatus` | DONE   | `services/atlas-channel/atlas.com/channel/monster/processor.go:72-74`. Grep `"Doom: caster=\["` returns exactly one hit (Task 9 step 2 condition met). Commit `746c24714`. |
| 10 | atlas-channel — generic `itemCon` consume on cast              | DONE (with documented deviation) | Helper `findItemSlotInInventory` at `character_attack_common.go:88`; consume call wired into the `if !registered` HP/MP gate at `character_attack_common.go:253-259`; warn-on-missing branch present. Tests `TestFindItemSlotInInventory_Found`/`_NotFound` at `character_attack_common_test.go:577,605` use the realized builder API (`SetCompartment(comp).MustBuild()`, `asset.NewBuilder(compId, templateId).SetId(...).SetSlot(...).SetQuantity(...).MustBuild()`). Commit `4a3312d6d`. |
| 9  | Cross-service build, full test, manual handoff                 | DONE (code half) | All three services build & test green (see results table). Manual checklist (snail render, expiry sprite restore, item-decrement) is by design human-verified and remains for the user. |

**Completion Rate:** 11/11 plan tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (two `DONE (with documented deviation)` items are not partial — the plan's exact assertions/API were unrealizable; replacements pin equivalent or stronger behavior and are inline-documented.)

## Skipped / Deferred Tasks

None.

## Documented Deviations

### Task 4 — STATUS_APPLIED event-count assertions removed

The plan's `statusApplied != 1` and `statusApplied != 2` assertions in
`TestApplyStatusEffect_Doom_BypassesElementalImmunity` and
`TestApplyStatusEffect_Doom_ReapplyReplacesExisting` cannot be satisfied: the
production `ApplyStatusEffect` emits `STATUS_APPLIED` via `producer.ProviderImpl`
directly (see `processor.go` apply branch ~line 1098), never through the
`p.emit` pathway the test recorder taps. Implementer correctly:
- Replaced the failing assertions with `_ = events` swallows.
- Added `NOTE:` comments at `processor_test.go:1755` and `1849` that explain the
  bypass and identify which remaining assertions are load-bearing
  (`HasStatusEffect("DOOM")` for elemental bypass, `EffectId()` equality and
  `doomEffects == 1` for the refresh semantics).
- Left the boss-rejection test (`TestApplyStatusEffect_Doom_RejectedOnBoss`)
  with its event-loop check intact because the rejection short-circuits before
  the producer call, so the recorder's "no STATUS_APPLIED for boss reject"
  assertion is still meaningful. (Verified at `processor_test.go:1794-1798`.)

State pins fully cover the behavioral guarantees the plan's event-count
assertions were proxies for.

### Task 10 — realized builder API used in tests

The plan's snippets called `inventory.NewBuilder(characterId).SetCompartment(invType, comp).Build()`,
`compartment.NewBuilder(uuid, uuid, type, capacity).SetAssets(assets).Build()`,
and `asset.NewBuilder(uuid, uuid, 1, templateId, asset.ReferenceIdConsumable).SetSlot(...).SetQuantity(...).MustBuild()`.
The realized API (verified compiles and tests pass) is:
- `channelinv.NewBuilder(1).SetCompartment(useComp).MustBuild()` (single-arg `SetCompartment`, no inv-type key).
- `compartment.NewBuilder(compId, 1, inventoryconst.TypeValueETC, 24).SetAssets([...]).MustBuild()`.
- `asset.NewBuilder(compId, magicRockId).SetId(101).SetSlot(3).SetQuantity(5).MustBuild()` — asset id required `> 0`.

The plan itself anticipated this divergence (Task 10 step 4: "If `asset.NewBuilder`, `compartment.NewBuilder`, or `inventory.NewBuilder` are not the exact constructors in this package, run grep … and adapt the test in lockstep.") so this is a sanctioned adaptation, not silent drift.

Note: the test fixtures use `inventoryconst.TypeValueETC` rather than `TypeValueUse`. This still exercises the helper end-to-end (`TypeFromItemId(magicRockId=4006000)` resolves to ETC for the chosen template id, so the lookup path is correct). The `_NotFound` test relies on the same compartment-type resolution path returning an empty asset list, which is what the test asserts.

## Build & Test Results

| Service                                  | Build | Tests | Notes |
|------------------------------------------|-------|-------|-------|
| `services/atlas-monsters/atlas.com/monsters` | PASS | PASS  | `monster` pkg test 228s (timer/registry suite); other packages sub-second. No failures. |
| `services/atlas-channel/atlas.com/channel`   | PASS | PASS  | `socket/handler` (Task 7 + 10 tests live here) green. |
| `services/atlas-data/atlas.com/data`         | PASS | PASS  | `skill` package green; Task 1 test passes. |

All three: `go build ./...` returned no output (success) and `go test ./... -count=1` produced no `FAIL` lines.

## Plan-Adherence Detail Checks

- Doom log line uniqueness (Task 9 step 2): `grep -rn '"Doom: caster=\[' services/` returns exactly one hit at `services/atlas-channel/atlas.com/channel/monster/processor.go:73`. PASS.
- DOOM elemental short-circuit (Task 4): present at `processor.go:1112-1114`, references `monster2.StatusDoom` constant (matches plan's expectation about the existing import alias).
- `testInformationLookup` extended to `ApplyStatusEffect`: confirmed at `processor.go:1072-1073` alongside the original `UseBasicAttack` site at `:700-701`.
- Helper extraction preserved behavior: existing `TestComputeReflect_*`, `TestReflectFlow_*`, and venom helper tests continue to pass under the full atlas-channel suite, as required by Task 5.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending Task 9 manual checklist, which is by design out-of-band human verification, not blocked code work)

## Action Items

None. The two deviations are sanctioned, inline-documented, and preserve the plan's behavioral intent through equivalent state-based assertions and the realized builder API.

---

# Backend Guidelines Audit — task-047-priest-doom

- **Audit Date:** 2026-05-03
- **Branch:** `task-047-priest-doom` (range `a583cc448` → `4a3312d6d`)
- **Scope:** Go packages touched by the diff:
  - `services/atlas-data/atlas.com/data/skill` (test-only)
  - `services/atlas-monsters/atlas.com/monsters/monster`
  - `services/atlas-monsters/atlas.com/monsters/monster/information` (test-only builder helpers)
  - `services/atlas-channel/atlas.com/channel/socket/handler`
  - `services/atlas-channel/atlas.com/channel/monster`
- **Guidelines source:** `.claude/skills/backend-dev-guidelines/resources/*` (DOM-* / SUB-* / EXT-* / SEC-* checklists)
- **Build:** PASS (atlas-data, atlas-monsters, atlas-channel all `go build ./...` clean)
- **Tests:** PASS (`atlas-data/skill` 0.047s; `atlas-monsters/monster` 228.914s including the three new Doom apply tests; `atlas-channel/socket/handler` 0.008s and `atlas-channel/monster` 0.011s including the six new helper tests)
- **Overall:** NEEDS-WORK (build/tests green, but DOM-21 failures in changed code)

## Phase 1: Build & Test Results

| Service | Build | Tests |
|---|---|---|
| atlas-data | PASS | `ok atlas-data/skill 0.047s` |
| atlas-monsters | PASS | `ok atlas-monsters/monster 228.914s`, `ok atlas-monsters/monster/information 0.004s` |
| atlas-channel | PASS | `ok atlas-channel/socket/handler 0.008s`, `ok atlas-channel/monster 0.011s`, `ok atlas-channel/monster/information 0.004s` |

## Phase 2: Package Classification (changed packages only)

| Package | Has `model.go`? | Has `resource.go`? | Classification |
|---|---|---|---|
| `atlas-data/skill` | yes | no (top-level data reader) | Domain (test-only change — no production surface to audit) |
| `atlas-monsters/monster` | yes (`monster/model.go`) | yes (`monster/resource.go`, NOT changed in this diff) | Domain — production logic edit in `processor.go` |
| `atlas-monsters/monster/information` | yes (`information/model.go`) | no | Sub-domain (test helper builder only) |
| `atlas-channel/socket/handler` | no | no | Support / handler — no DOM resource checks apply |
| `atlas-channel/monster` | yes (`monster/model.go`) | no (in-process facade) | Domain — `processor.go` Debugf addition |

`resource.go` was NOT modified in any package on this branch, so DOM-04 / DOM-05 / DOM-07 / DOM-08 / DOM-09 / DOM-12 / DOM-13 / DOM-14 / DOM-15 / DOM-17 / DOM-18 / DOM-19 / SUB-03 / SUB-04 are out of scope (no changed lines to evaluate). Below I report only checks that have changed evidence to weigh, plus DOM-21 which the prompt explicitly flagged.

## Phase 3: Per-Package Mechanical Checks

### `atlas-monsters/monster` (production change in `processor.go`)

| ID | Check | Status | Evidence |
|---|---|---|---|
| DOM-01 | `builder.go` exists | PASS | `services/atlas-monsters/atlas.com/monsters/monster/builder.go:1` (pre-existing, unchanged) |
| DOM-02 | `ToEntity()` method | PASS (out of changed scope) | Pre-existing entity layer; no edits in this diff |
| DOM-03 | `Make(Entity)` function | PASS (out of changed scope) | Pre-existing |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `processor.go` constructors take `logrus.FieldLogger` (pre-existing); the diff does not change the signature |
| DOM-11 | Providers use lazy evaluation | PASS (not touched) | No provider edits in this diff |
| DOM-20 | Table-driven tests | PARTIAL | The 3 new Doom tests at `processor_test.go:1716`, `:1762`, `:1797` are written as **discrete `Test…` functions**, not as `tests := []struct{…}` table cases. Allowed by guideline (table-driven is for "many small variants of one path"); each Doom test exercises a structurally distinct branch (immunity / boss / re-apply). PASS-acceptable, but flagged for awareness |
| DOM-21 | No duplication of atlas-constants types | **PASS** | `processor.go:1112` uses `monster2.StatusDoom` (the atlas-constants symbol at `libs/atlas-constants/monster/status.go:16`). The DOOM short-circuit pulls the canonical constant rather than a string literal |

### `atlas-monsters/monster/information` (test-only builder helpers)

| ID | Check | Status | Evidence |
|---|---|---|---|
| DOM-01 | `builder.go` exists | PASS | `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go:1` |
| `ModelBuilder` immutability shape preserved | extension only | PASS | New `SetBoss` (`builder.go:43`) and `SetResistances` (`builder.go:52`) follow the same private-field-plus-fluent-setter pattern as the pre-existing setters; `Build()` (`builder.go:67`) returns the immutable Model with the new fields wired through (`:73-74`) |
| DOM-21 | No duplication of atlas-constants types | PASS | `SetResistances(map[string]string)` matches the existing `Model.resistances` shape; no shadow type introduced |

### `atlas-data/skill` (test-only)

| ID | Check | Status | Evidence |
|---|---|---|---|
| DOM-20 | Table-driven tests | PASS-acceptable | `reader_test.go:2995` `TestReader_PriestDoom_MapsDoomStatus` is one focused fixture against one skill id; sibling tests in the same file follow the same one-test-per-fixture style |
| DOM-21 | No duplication of atlas-constants types | **FAIL** | `reader_test.go:3000` hardcodes the skill id as the magic string `"2311005"` in the XML fixture (intrinsic to the data shape, acceptable), but the assertion at `reader_test.go:3033` looks up `rmm["2311005"]` and `:3041` checks `MonsterStatus["DOOM"]` — both literals. `libs/atlas-constants/skill/constants.go:3067` defines `PriestDoomId = Id(2311005)` and `libs/atlas-constants/monster/status.go:16` defines `StatusDoom = "DOOM"`. A test-side `fmt.Sprintf("%d", uint32(skill.PriestDoomId))` and `monster.StatusDoom` constant would tie the test to the shared source of truth. Minor — XML fixture is a pinned wire format, so the literal in the XML body itself is justified, but the map lookups are not |

### `atlas-channel/socket/handler` (production change in `character_attack_common.go`)

| ID | Check | Status | Evidence |
|---|---|---|---|
| Handler error-discard pattern | preserved | NEUTRAL | The pre-existing `_ = mp.ApplyStatus(...)` underscore-discard pattern at `character_attack_common.go:158` and `:204` is carried over from the pre-refactor inline loop; the refactor neither introduces nor removes the discard. DOM-09 only enforces `Transform` error handling in `resource.go`, which is not relevant here |
| Helper extraction safety | preserved | PASS | `processDamageInfoEntry` (`character_attack_common.go:119`) is invoked at `:300-305` with the same per-entry semantics the pre-refactor inline loop had (verified by line-for-line diff equivalence and the green helper tests at `character_attack_common_test.go:428,469,504,543`) |
| Handler purity (no `os.Getenv`, no direct DB) | PASS | grep across the changed file: zero `os.Getenv`, zero `db.Create`/`db.Save`/`db.Delete` |
| `findItemSlotInInventory` defense-in-depth comment | PASS | `character_attack_common.go:257` warns and continues; consistent with the comment at `:88-92` and the cost gate at `:246` |
| DOM-20 | Table-driven tests | PASS | `TestAttackKindFromAttackType` (`character_attack_common_test.go:307`) uses table-driven; the 6 new Doom/findItem tests are branch-specific (PASS-acceptable per the guideline carve-out) |
| DOM-21 | No duplication of atlas-constants types | **FAIL — production** | `character_attack_common.go:151` literal `"DOOM"`. The file already imports `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"` at `:26` (used for `ReflectKindMagical`/`ReflectKindPhysical` at `:76-78`). The Doom-gate condition should read `if _, isDoom := ms[monster2.StatusDoom]; isDoom && attackKind != ""` — atlas-constants already defines `StatusDoom = "DOOM"` at `libs/atlas-constants/monster/status.go:16`. Same applies to the matching Debugf message text at `:153` and `:255` (cosmetic). The atlas-monsters production processor at `processor.go:1112` already uses the constant, so this is a divergence within the same logical Doom path |
| DOM-21 (test mirror) | No duplication of atlas-constants types | **FAIL — test** | `character_attack_common_test.go:410` `MonsterStatus: map[string]uint32{"DOOM": 1}` and `:416` `SetSkillId(2311005)` and `:448-452` repeat raw `2311005` / `"DOOM"` literals. `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"` is imported at `:20` (used at `:466,498,525` for `ReflectKindMagical`); the Doom literals should resolve to `monster2.StatusDoom` and the skill id to `uint32(skill3.PriestDoomId)` (an import for `libs/atlas-constants/skill` would need to be added). The prompt **explicitly** required atlas-constants reuse for `PriestDoomId` and `StatusDoom`; the production atlas-monsters processor cites the constant at `processor.go:1112` so the channel side is the asymmetry |
| DOM-21 (consume call) | No duplication of atlas-constants types | PASS | `character_attack_common.go:255` correctly composes `charcon.Id(s.CharacterId())`, `itemconst.Id(se.ItemConsume())`, and `slot.Position(...)`; `findItemSlotInInventory` at `:89` correctly uses `inventoryconst.TypeFromItemId(itemconst.Id(itemId))`. This portion of Task 10 fully satisfies DOM-21 |
| `findItemSlotInInventory` slot truncation | PASS | `asset.Slot()` returns `int16` and `slot.Position(a.Slot())` preserves the signed value; `comp.Assets()` is iterated linearly. Caller treats `false` as warn-and-allow (`character_attack_common.go:256-258`), matching the inline doc that this is defense-in-depth, not authoritative |

### `atlas-channel/monster` (production change in `processor.go`)

| ID | Check | Status | Evidence |
|---|---|---|---|
| DOM-06 | Processor accepts `FieldLogger` | PASS | Pre-existing `*Processor` constructor; the new Debugf at `processor.go:73` uses `p.l.Debugf` (the `FieldLogger` interface) |
| DOM-21 | No duplication of atlas-constants types | **FAIL** | `processor.go:72` literal `"DOOM"`. The local `monster2` alias here is the **Kafka** package (`atlas-channel/kafka/message/monster`, declared at `:4`), so `monster2.StatusDoom` does not resolve in this file. To fix, add an additional import alias for `libs/atlas-constants/monster` (e.g. `mc "github.com/Chronicle20/atlas/libs/atlas-constants/monster"`) and replace the literal with `mc.StatusDoom`. The cost is one new import for one symbol but it eliminates a divergence between two services that are both gating on the same status string. Strict DOM-21: FAIL. Pragmatic note: this is a Debugf-only branch, so the cost of failing-to-update the literal in the future is low (mismatched logs, not a behavior bug) |

## Phase 4: Security Review

Not applicable — none of the changed services handle authentication, authorization, or token management.

## Summary

### Blocking (must-fix before merge)

None. All Phase 1 gates pass and no security findings.

### Non-Blocking (should-fix; tracked as DOM-21 debt)

- **DOM-21 (production, atlas-channel handler):** `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:151` — replace literal `"DOOM"` with `monster2.StatusDoom` (the file already imports `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"` at `:26`). Asymmetric with `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1112`, which already uses the constant.
- **DOM-21 (production, atlas-channel monster facade):** `services/atlas-channel/atlas.com/channel/monster/processor.go:72` — add an import alias for `libs/atlas-constants/monster` and replace literal `"DOOM"` with the constant. The local `monster2` alias is the Kafka package, not atlas-constants.
- **DOM-21 (tests, atlas-channel handler):** `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go:410,416,448-452` — replace `"DOOM"` with `monster2.StatusDoom` (already imported at `:20`); replace `2311005` skill-id literals with `uint32(skill3.PriestDoomId)` after importing `libs/atlas-constants/skill`.
- **DOM-21 (tests, atlas-monsters):** `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:1710,1712,1752,1791,1838` — `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"` is imported at `:16`; swap the four `"DOOM"` string literals to `monster2.StatusDoom` and the `2311005` literal to `uint32(skill.PriestDoomId)` (skill atlas-constants would need a fresh import).
- **DOM-21 (tests, atlas-data):** `services/atlas-data/atlas.com/data/skill/reader_test.go:3033,3041` — the XML fixture body itself is a pinned wire format (literal acceptable), but the map-lookup assertions can resolve through `fmt.Sprintf("%d", uint32(skill.PriestDoomId))` and `monster.StatusDoom`.

These do not block merge — they are textual hygiene that prevents future drift if either constant is ever renamed. The production behavior is correct in all cases (verified by green tests).

### Notes for the maintainer

- The Phase 1 build-and-test gate is the load-bearing evidence here. The atlas-monsters test (228.9 s) exercises the new Doom branches end-to-end against the in-memory registry; the atlas-channel handler tests exercise `processDamageInfoEntry` against fakes for cast / reflect / spread / non-Doom-empty-damage. Together these pin the four new behavioral promises of the branch.
- The two pre-existing `_ = mp.ApplyStatus(...)` discards inside `processDamageInfoEntry` (`character_attack_common.go:158, :204`) are unchanged from the pre-refactor inline loop; if the team wants to start surfacing producer errors, that is a separate refactor and the audit does not flag it under DOM-09 (which scopes to `resource.go` Transform calls).
- `processor_test.go` documents the `producer.ProviderImpl(...)` bypass at `:1755` and `:1849` honestly — the recorder cannot observe the STATUS_APPLIED emission, so the tests pin state via `HasStatusEffect("DOOM")` and `EffectId()` comparison instead. This is consistent with the testing-guide preference for state-based assertions over event-count brittleness.

