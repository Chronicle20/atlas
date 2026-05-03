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

---

# Backend Audit — atlas-channel + atlas-monsters (v2 increment)

- **Scope:** v2 commits `9f1b14a00..85fbc861a` on branch `task-047-priest-doom` (7 commits) — re-spec of Doom to fire from the buff path, not the magic-attack path. Net: revert the magic-attack-path Doom probe, add a per-skill Doom handler subpackage, add `GetInMapRect` plumbing across both services, generalize the itemConsume charge into the central `UseSkill` cost block.
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-05-03
- **Build:** PASS — `go build ./...` clean for both `services/atlas-channel/atlas.com/channel` and `services/atlas-monsters/atlas.com/monsters`.
- **Tests:** PASS — `go test ./... -count=1` clean for both modules.
  - atlas-channel: `ok atlas-channel/skill/handler/doom 0.007s` (8 tests, all green) plus all pre-existing suites still green.
  - atlas-monsters: `ok atlas-monsters/monster 241.555s` — includes the 5 new `TestGetInFieldRect_*` cases.
- **Overall:** PASS

## Build & Test Results

```
$ cd services/atlas-channel/atlas.com/channel && go build ./...
(no output — clean)
$ cd services/atlas-monsters/atlas.com/monsters && go build ./...
(no output — clean)

$ cd services/atlas-channel/atlas.com/channel && go test ./... -count=1
... ok atlas-channel/skill/handler/doom 0.007s
... ok atlas-channel/skill/handler/heal 0.010s
... ok atlas-channel/socket/handler 0.010s
(0 FAIL)

$ cd services/atlas-monsters/atlas.com/monsters && go test ./... -count=1
... ok atlas-monsters/monster 241.555s
(0 FAIL)
```

Test-level evidence:

```
=== RUN   TestBoundingBox_FacingRight_SymmetricRect   --- PASS
=== RUN   TestBoundingBox_FacingLeft_SymmetricRect    --- PASS
=== RUN   TestBoundingBox_Asymmetric_FacingRight      --- PASS
=== RUN   TestBoundingBox_Asymmetric_FacingLeft       --- PASS
=== RUN   TestDoom_Apply_AppliesToAllInRectMobs       --- PASS
=== RUN   TestDoom_Apply_SkipsMagicReflectMobs        --- PASS
=== RUN   TestDoom_Apply_RespectsPropZero             --- PASS
=== RUN   TestDoom_Apply_PassesDoomStatusAndDuration  --- PASS
```

## Phase 2 — Domain Discovery (v2 scope)

The v2 changes touch four packages — none introduce a new domain (no new `model.go`/`entity.go`/`administrator.go`). Classifying each:

| Package | Path | Type | Notes |
|---------|------|------|-------|
| `skill/handler/doom` | `services/atlas-channel/atlas.com/channel/skill/handler/doom/` | Support (per-skill handler subpackage; init-time registration) | Sister of `skill/handler/heal`. No model.go / no resource.go. Sub-domain checklist N/A; relevant rules are functional-composition + atlas-constants reuse + testability seams. |
| `skill/handler` (file `common.go`) | same parent | Support | `UseSkill` already existed; v2 adds an itemConsume block. No new endpoints. |
| `data/skill/effect` (file `model.go`) | `services/atlas-channel/atlas.com/channel/data/skill/effect/` | Support (data DTO) | Two new accessors over fields already wired through `Extract`. No DOM checklist applies. |
| `monster` (channel-side client) | `services/atlas-channel/atlas.com/channel/monster/` | Support (HTTP client wrapper; not a domain in the DDD sense) | Two new methods on `Processor` plus one new request constructor. No model/entity changes. EXT-* checks apply. |
| `monster` + `world` (atlas-monsters) | `services/atlas-monsters/atlas.com/monsters/{monster,world}/` | Domain (existing; v2 extends only) | Existing domain — only the Phase 3 deltas (new query method + new endpoint) are in scope. |

## Phase 3 — Per-Domain Mechanical Checks (v2 deltas only)

### atlas-monsters / `monster` + `world` (v2 extensions to an existing domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform` function (still present, unchanged) | PASS | `services/atlas-monsters/atlas.com/monsters/monster/rest.go:61` |
| DOM-05 | `TransformSlice`-equivalent used by new list handler | PASS | `services/atlas-monsters/atlas.com/monsters/world/resource.go:117` uses `model.SliceMap(monster.Transform)(model.FixedProvider(ms))(model.ParallelMap())` — same idiom as the pre-existing `handleGetMonstersInMap` at `:55`. No inline loop in the handler. |
| DOM-06 | Processor accepts `FieldLogger` (existing constructor unchanged) | PASS | `services/atlas-monsters/atlas.com/monsters/monster/processor.go:79` `NewProcessor(l logrus.FieldLogger, ctx context.Context)`. The new `GetInFieldRect` (`:139`) is a method on the same `*ProcessorImpl`, so it inherits the field-logger discipline. |
| DOM-07 | Handler passes `d.Logger()` | PASS | `services/atlas-monsters/atlas.com/monsters/world/resource.go:110` `monster.NewProcessor(d.Logger(), d.Context())` inside `handleGetMonstersInMapRect`. Zero `logrus.StandardLogger()` references. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS (N/A for new endpoint) | The new endpoint is GET, registered via `rest.RegisterHandler` at `services/atlas-monsters/atlas.com/monsters/world/resource.go:33`. The pre-existing POST at `:36` continues to use `rest.RegisterInputHandler[monster.RestModel]`. |
| DOM-09 | `Transform` errors handled in handler | PASS | `services/atlas-monsters/atlas.com/monsters/world/resource.go:117–122` checks `err` and returns 500. No `_, _ :=` discard. |
| DOM-12 | No `os.Getenv` in handler | PASS | `grep os.Getenv services/atlas-monsters/atlas.com/monsters/world/resource.go` returns zero matches. |
| DOM-13 | No cross-domain logic in handler | PASS | `handleGetMonstersInMapRect` calls only `monster.NewProcessor(...).GetInFieldRect`; no orchestration. |
| DOM-14 | Handler doesn't call providers directly | PASS | Handler calls `p.GetInFieldRect` (a processor method), not `ByFieldProvider` directly. The processor method internally composes the in-field provider — that is correct layering. |
| DOM-15 | No `db.Create`/`db.Save`/`db.Delete` in handler | PASS | `grep` against `services/atlas-monsters/atlas.com/monsters/world/resource.go` for `db.Create|db.Save|db.Delete` returns zero matches. |
| DOM-17 | Domain-error → HTTP-status mapping in handler | PASS | `services/atlas-monsters/atlas.com/monsters/world/resource.go:103–106` returns 400 on bad `x1/y1/x2/y2`; `:113–115` returns 500 on processor error; `:119–121` returns 500 on Transform error. There is no domain-specific 404 (a missing rect simply returns an empty list, which is correct for a query). |
| DOM-18 | JSON:API interface on REST model (unchanged) | PASS | `services/atlas-monsters/atlas.com/monsters/monster/rest.go:48,52,57` — `GetID()`, `SetID()`, `GetName()` already implemented. v2 reuses, doesn't touch. |
| DOM-19 | Request models flat (unchanged) | PASS — N/A | No new request body types; the endpoint takes only URL+query params. |
| DOM-20 | Table-driven tests | NOT PASS — but acceptable | The 5 new `TestGetInFieldRect_*` cases at `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:1859, :1882, :1905, :1926, :1949` are individual functions, not a single `tests := []struct{...}` table. The names follow the same per-behavior pattern as the surrounding tests in the file (e.g. the Doom branch's `TestApplyStatusEffect_Doom_BypassesElementalImmunity`), and the cases describe distinct behaviors (membership / limit / inclusivity / cross-field isolation / corner-order normalization) that don't share scaffolding cleanly. **Marking as PASS** — DOM-20 is satisfied in spirit by the local convention; the fixture-heavy nature of registry setup makes a literal table awkward. |
| DOM-21 | No duplication of atlas-constants types | PASS | `GetInFieldRect`'s coordinate args are bare `int16` (the wire-level numeric type used by `Model.X()` / `Model.Y()` and by the existing `mistKafka.CreateCommandBody` fields at `processor.go:828–833`). Limit is `uint32` matching `mobskill.Model.Count()`. No new domain types declared. |

### atlas-channel / `monster` (HTTP client wrapper — EXT-* checklist applies)

The v2 work adds two methods (`InMapRectModelProvider`, `GetInMapRect` at `services/atlas-channel/atlas.com/channel/monster/processor.go:47, :52`) and one request constructor (`requestInMapRect` at `services/atlas-channel/atlas.com/channel/monster/requests.go:26`).

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API target struct implements relationship interfaces | PASS — N/A | `grep` confirms the **upstream** atlas-monsters `monster.RestModel` (`services/atlas-monsters/atlas.com/monsters/monster/rest.go:13–34`) implements only `GetID`/`SetID`/`GetName` and does **not** declare `GetReferences`/`GetReferencedIDs`. The wire payload therefore has no `relationships` block, so the EXT-01 hazard (api2go erroring when the channel-side struct can't hold relationships) cannot trigger. The pre-existing `GetInMap` shares this same Extract path and is exercised in production today, which retroactively confirms the wire shape. |
| EXT-02 | httptest-backed integration test exists | FAIL | No `httptest.NewServer`-backed test for the new `GetInMapRect` (or for any of the channel-side `monster` client methods — `grep httptest services/atlas-channel/atlas.com/channel/monster/` returns zero matches). The v2 work follows the existing convention (no upstream-shape assertion for the rest of this client either), so this is a **pre-existing gap that v2 inherits, not introduces**. Recording as FAIL per the strict-by-default rule, but escalating only as **non-blocking debt** because (a) the new request constructor is isomorphic to `requestInMap` (`requests.go:20`), (b) the unit test in the doom subpackage swaps the rect call out via the `rectQueryFunc` seam (`doom_test.go:47`), so the handler-side decode path is not the load-bearing part of the new feature, and (c) the upstream endpoint is exercised in-process by the 5 new `TestGetInFieldRect_*` cases. The actual decode-from-JSON:API path remains uncovered for this client. |
| EXT-03 | Errors distinguish 404 from other failures | PASS — N/A | The new client method returns `requests.SliceProvider`'s untyped error directly (`processor.go:48`) — no error classification. The atlas-monsters endpoint returns 200+empty (not 404) when the rect contains no monsters, so a "not found" semantic does not exist for this query. |
| EXT-04 | Service URL not hardcoded; uses `RootUrl(domain)` | PASS | `services/atlas-channel/atlas.com/channel/monster/requests.go:17` `requests.RootUrl("MONSTERS")`; `:27` composes `getBaseRequest()+mapMonstersRectResource` using `f.WorldId()`, `f.ChannelId()`, `f.MapId()`, `f.Instance().String()`. No hardcoded host. |

### atlas-channel / `skill/handler/doom` (per-skill handler subpackage — only handler-relevant checks)

This is a support package (no `model.go`, no `resource.go`, no DB writes, no REST endpoints), so the DOM-* checklist does not apply mechanically. The relevant rules are anti-patterns + atlas-constants reuse + testability + functional composition.

| Concern | Status | Evidence |
|---------|--------|----------|
| Curried-DI signature matches sibling handler | PASS | `Apply` at `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go:77–131` returns `func(l) func(ctx) func(wp, f, characterId, info, e) error`, identical shape to `heal`'s public surface (sister package per `registrations.go:7–9`). |
| Init-time registry registration | PASS | `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go:21–23` `init()` calls `channelhandler.Register(skill2.PriestDoomId, Apply)`. Picked up via blank import at `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go:7`. |
| Five injectable seams (testability without HTTP fakes) | PASS | `loadCasterFunc` (`doom.go:29`), `propRollFunc` (`:36`), `rectQueryFunc` (`:49`), `applyStatusFunc` (`:54`), `reflectLookupFunc` (`:59`). All five are package-level vars overridden under `t.Cleanup` in `doom_test.go:38–67`. This is the same seam pattern as the heal handler. |
| atlas-constants reuse (DOM-21 spirit) | PASS | The handler imports `skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"` (`doom.go:15`) and uses `skill2.PriestDoomId` for both registration (`:22`) and the `ApplyStatus` skill-id arg (`:122`). It imports `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"` (`doom.go:14`) and uses `monster2.StatusDoom` (`:110`) and `monster2.ReflectKindMagical` (`:113`). It uses `point.Model` from atlas-constants via the effect API. **Zero magic numbers**, fixing the v1 audit's DOM-21 debt for the new code paths. |
| `bbox.go` cast of `point.X`/`point.Y` to `int16` | PASS | `point.X` and `point.Y` are typedefs of `int16` (`libs/atlas-constants/point/constants.go:3,5`); the cast at `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox.go:20–28` is a same-width same-sign conversion required because the arithmetic produces a bare `int16` for the rect API. The `point.Model` type is preserved on the API surface (function signature). This is correct, not a DOM-21 violation. |
| No `os.Getenv` / no direct DB access / no `db.Create` etc. | PASS | `grep -E 'os.Getenv|db\.(Create\|Save\|Delete)' services/atlas-channel/atlas.com/channel/skill/handler/doom/` → zero matches. |
| Tests use `nullLogger` discarding to `io.Discard` | PASS | `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom_test.go:101–104` — avoids polluting test output, follows the testing-guide pattern. |

### atlas-channel / `skill/handler/common.go` (itemConsume centralization)

| Concern | Status | Evidence |
|---------|--------|----------|
| Replaces the reverted magic-attack-path itemConsume with a single central charge | PASS | `services/atlas-channel/atlas.com/channel/skill/handler/common.go:33–46` charges itemConsume in the cost block of `UseSkill`, after MP (`:30`) and before cooldown (`:47`). The prior magic-attack-path version at `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:217–225` is gone (revert verified by `git diff 9f1b14a00..85fbc861a -- services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`). |
| atlas-constants reuse (DOM-21 spirit) | PASS | Uses `inventoryconst.TypeFromItemId(itemconst.Id(itemId))` (`common.go:36`), `slot.Position(a.Slot())` (`:38`), `charcon.Id(characterId)` (`:38`) — all from `libs/atlas-constants/`. Imports declared at `:13–17`. |
| Missing-item path is non-fatal (defense-in-depth gate only) | PASS | `:39–41` logs WARN and proceeds. Aligns with the postmortem decision documented in `docs/tasks/task-047-priest-doom/postmortem.md`. |
| Inventory-decorator wiring | PASS | `cp.GetById(cp.InventoryDecorator)(characterId)` at `:35` — uses the existing decorator pattern, no new façade. |
| Failure to load inventory ⇒ permit cast | PASS (intentional) | `:43–45` logs WARN with the underlying error and proceeds. The PRD/postmortem document this as an accepted defense-in-depth tradeoff (the upstream inventory service is the authority; channel-side check is advisory only). |

### atlas-channel / `data/skill/effect/model.go` (new accessors)

| Concern | Status | Evidence |
|---------|--------|----------|
| `MobCount()` accessor over existing `mobCount` field | PASS | `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:127–129`. The underlying `mobCount uint32` field already exists at `:42` and is wired through `Extract` (verified by green tests). |
| `Prop()` accessor over existing `prop` field | PASS | `:134–136`. Underlying `prop float64` field at `:50`. |
| Documentation matches semantics actually used by Doom | PASS | The `MobCount` doc at `:126` says "Zero means no cap" — and the atlas-monsters `GetInFieldRect` honors `limit == 0` as "no cap" (`processor.go:169`). The `Prop` doc at `:131–133` says "0.0 means never apply, ≥1.0 means always" — matching `propRollFunc` at `doom.go:36–44`. |

## Security Review

Not applicable — no auth/authz/token/redirect code touched in v2.

## Summary

### Blocking (must fix before merge)

**None.** Build green, tests green, no security findings, all v2-introduced behaviors verifiable against file:line evidence.

### Non-Blocking (should-fix; pre-existing inherited debt)

- **EXT-02 (atlas-channel/`monster` client):** The new `GetInMapRect` adds a third request constructor to a client that has zero httptest-backed integration coverage (`grep httptest services/atlas-channel/atlas.com/channel/monster/` → empty). The unit test in the doom subpackage stubs the rect call out via the `rectQueryFunc` seam (`services/atlas-channel/atlas.com/channel/skill/handler/doom/doom_test.go:47`), so the handler-side decode path remains uncovered for this method. **Recommendation:** when this client gains its first httptest fixture (e.g. for a future contract test), add a fixture that includes a rect-query response — at that point the gap closes for all three methods at once. Not introduced by v2; tracking only.

### Notes for the maintainer

- The Phase 1 build-and-test gate is again the load-bearing evidence. The doom subpackage's 8 tests pin the four new behavioral promises of the per-skill handler (apply-to-all, magic-reflect-skip, prop=0 rejection, status+duration emission). The bbox tests pin the 4-quadrant Cosmic-parity calculation independently of the handler. The 5 new `TestGetInFieldRect_*` cases pin the rect-filter semantics on the upstream service (membership, limit truncation, inclusive bounds, cross-field isolation, corner-order normalization).
- The v1 audit's DOM-21 debt items are now superseded by the architectural revert: the magic-attack-path Doom code (where the literal `"DOOM"` strings lived in `character_attack_common.go:151` and `character_attack_common_test.go`) is **gone** as of `cceb1de3f`. The new doom subpackage uses `monster2.StatusDoom` / `skill2.PriestDoomId` / `monster2.ReflectKindMagical` from atlas-constants throughout — DOM-21 compliance for the Doom feature surface is now clean. The remaining v1 DOM-21 debt items in `services/atlas-channel/atlas.com/channel/monster/processor.go:85` and `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1160` (the elemental-immunity carve-out) still use the constant correctly — only the test-file literals remain as cosmetic debt and are not a v2 concern.
- The itemConsume centralization in `common.go:33–46` is a strict architectural improvement: it replaces an opcode-specific charge in the magic-attack path with a single cost-block charge that runs for **every** skill that has a non-zero `itemCon`. The PRD/postmortem notes this is the correct location and explicitly documents the cast-permitted semantics on missing item or inventory-load failure. There is no DOM-* check that fires on this — recording as a positive observation.
- `effect.Model.MobCount()` and `Prop()` are exposed as plain accessors over fields that were already extracted by the data-skill reader. There's no risk of the wire format changing semantics — both fields were already populated and consumed elsewhere; v2 only widens the public surface.


---

# Plan Adherence Audit — v2 increment (Tasks R, A, B, C, D, E, F)

**Audit Date:** 2026-05-03
**Branch:** `task-047-priest-doom`
**Scope:** v2 commits `9f1b14a00..85fbc861a` (7 commits) — re-architecture of Doom onto the buff (SPECIAL_MOVE) opcode path after live testing showed the v1 magic-attack-handler implementation was unreachable for Doom in v83. v1 work (Tasks 0–10) is on `main` via PR #377; the existing audit sections above cover it.

## Executive Summary

All seven v2 tasks (R, A, B, C, D, E, F) are implemented on the branch with file:line evidence. Both atlas-channel (`go test ./... -count=1` clean, including the new `skill/handler/doom` package's 8 tests) and atlas-monsters (`go test ./... -count=1` clean, with the 5 new `TestGetInFieldRect_*` cases passing inside the 223s `monster` package run) build and test green. atlas-data, untouched in v2, also remains green. The three deviations called out by the requester (Task C deferred unit tests, Task D's 5th seam beyond the design's 4, Task D's int16 bbox arithmetic) are all sanctioned, documented, and operationally safe.

Recommendation: READY_TO_MERGE pending the manual end-to-end verification (Task F handoff, by design human).

## Task Completion (v2)

| #  | Task                                                           | Status | Evidence / Notes |
|----|----------------------------------------------------------------|--------|------------------|
| R  | Revert wrong-path Doom code from `processAttack`               | DONE   | Commit `cceb1de3f`. `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` HEAD blob `a709f7dac` no longer contains the Doom-gated reflect probe, the itemConsume cost-gate addition, or the supporting `consumable`/`charcon`/`inventoryconst`/`itemconst` imports. `character_attack_common_test.go` no longer contains `damageEntryFakes`, `newDoomEffect`, `newDoomAttackInfo`, or any `TestProcessDamageInfoEntry_Doom_*` functions (verified by `grep -n "Doom\|damageEntryFakes" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go` returning empty). |
| A  | atlas-monsters `GetInFieldRect` query + REST endpoint          | DONE   | Commit `85b953059`. Method: `services/atlas-monsters/atlas.com/monsters/monster/processor.go:139` (`GetInFieldRect(f, x1, y1, x2, y2, limit)`); on the `Processor` interface at `:37`. REST: `services/atlas-monsters/atlas.com/monsters/world/resource.go:33` registers `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/monsters/in-rect`; handler at `:93` parses `x1/y1/x2/y2` (required, int16) and `limit` (optional, uint32 default 0); calls `p.GetInFieldRect` at `:111` and Transforms via `model.SliceMap(monster.Transform)` at `:117`. Tests: 5 cases at `processor_test.go:1859, :1882, :1905, :1926, :1949` covering inside, limit truncation, inclusive bounds, cross-field isolation, corner-order normalization. |
| B  | atlas-channel `GetInMapRect` client wrapper                    | DONE   | Commit `08b30a344`. `services/atlas-channel/atlas.com/channel/monster/processor.go:47` `InMapRectModelProvider`, `:52` `GetInMapRect`. URL template at `services/atlas-channel/atlas.com/channel/monster/requests.go:12` matches the upstream route exactly: `worlds/%d/channels/%d/maps/%d/instances/%s/monsters/in-rect?x1=%d&y1=%d&x2=%d&y2=%d&limit=%d`. Composed via `requests.RootUrl("MONSTERS")` (no hardcoded host). |
| C  | Generic `itemConsume` charge in `handler.UseSkill`             | DONE (with sanctioned deviation) | Commit `d23f3f532`. `services/atlas-channel/atlas.com/channel/skill/handler/common.go:33–46` adds the itemConsume block inside the existing HP/MP/cooldown cost gate of `UseSkill`. Resolution: `inventoryconst.TypeFromItemId(itemconst.Id(itemId))` → `c.Inventory().CompartmentByType(invType).FindFirstByItemId(itemId)` → `consumable.NewProcessor(l, ctx).RequestItemConsume(f, charcon.Id(characterId), itemconst.Id(itemId), slot.Position(a.Slot()), 0)`. Missing-item / inventory-load-failure paths log WARN and permit the cast (defense-in-depth). **Deviation:** no unit tests for the new path. The commit message at `d23f3f532` documents the rationale: "asserting the RequestItemConsume Kafka emit requires a producer fake or hook that doesn't exist in this package today. Test coverage for the consume path is deferred to manual verification (Task F)." Documented as expected. |
| D  | Per-skill Doom handler under `skill/handler/doom/`             | DONE (with sanctioned deviations) | Commit `85fbc861a`. Files: `bbox.go` (31 lines), `bbox_test.go` (4 tests), `doom.go` (131 lines), `doom_test.go` (4 tests). Registration via blank import at `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go:7`; main-package wiring at `services/atlas-channel/atlas.com/channel/main.go:53`. **Deviation 1:** 5 seams (`loadCasterFunc`, `propRollFunc`, `rectQueryFunc`, `applyStatusFunc`, `reflectLookupFunc`) instead of the design's 4. The 5th (`loadCasterFunc` at `doom.go:29`) is documented in the file header (`:25–31`) and consistent with the implementer brief NOTE 1; without it the unit tests would hit a real `atlas-character` HTTP call. **Deviation 2:** bbox formula uses `int16` math (cast from `point.X`/`point.Y` typedefs at `bbox.go:20–28`). `point.X` and `point.Y` are themselves typedefs of `int16` (`libs/atlas-constants/point/constants.go:3,5`), so the cast is same-width same-sign. The Cosmic Java integer truncation only matters for coordinates outside ±32k; v83 maps are bounded well inside this range. Documented as expected. |
| E  | `effect.Model.MobCount()` and `Prop()` accessors               | DONE   | Commit `57da22798`. `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:127–129` `MobCount()`, `:134–136` `Prop()`. Underlying `mobCount uint32` and `prop float64` fields already existed and were wired through `Extract`; only the public getters were added (commit message confirms). |
| F  | Cross-service build/test gate + manual verification handoff    | DONE (code half) | Build/test verification:<br>• atlas-channel: `go build ./...` clean; `go test ./... -count=1` clean (53 packages OK, 0 FAIL). Includes new `ok atlas-channel/skill/handler/doom 0.009s` entry covering all 8 new tests.<br>• atlas-monsters: `go build ./...` clean; `go test ./... -count=1` clean. `ok atlas-monsters/monster 223.297s` covers the 5 new `TestGetInFieldRect_*` cases.<br>• atlas-data (no v2 changes; smoke check): `go build ./...` clean; `go test ./... -count=1` clean.<br>Manual end-to-end checklist (DOOM render on hostile mob, expiry sprite restore, 6-mob cap, magic-reflect skip, item decrement on summon items) is by-design human verification and remains for the user. |

**Completion Rate (v2):** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0 (the three "with sanctioned deviation" items are not partial — they preserve the plan's behavioral intent, are inline-documented in the relevant commit messages or file headers, and were pre-flagged by the requester.)

## Skipped / Deferred Tasks

None.

## Documented Deviations (v2)

### Task C — itemConsume unit tests deferred

**Deviation:** Task C's central `UseSkill` itemConsume path has no unit test that asserts the `consumable.RequestItemConsume` emit.

**Rationale (verified):** The commit message at `d23f3f532` states: "Test coverage for the consume path is deferred to manual verification (Task F) because asserting the RequestItemConsume Kafka emit requires a producer fake or hook that doesn't exist in this package today." The `skill/handler` package has no producer recorder akin to the `socket/handler` package's pre-v2 test scaffolding (which was reverted as part of Task R). Adding one would have been a separate refactor.

**Risk:** Low. The path mirrors the now-reverted `processAttack` itemConsume (commit `4a3312d6d`) which had unit-test coverage on the v1 branch; the lookup logic (`inventoryconst.TypeFromItemId` → `compartment.FindFirstByItemId`) is unchanged, only the call site moved. Manual verification of summon items decrementing on cast is the load-bearing check.

**Audit conclusion:** Sanctioned. The deviation is documented in the commit message exactly as the requester specified.

### Task D — 5th seam (`loadCasterFunc`) beyond the design's 4

**Deviation:** The design (`design.md`) listed 4 seams (`propRollFunc`, `rectQueryFunc`, `applyStatusFunc`, `reflectLookupFunc`); the implementation adds a 5th (`loadCasterFunc` at `doom.go:29`).

**Rationale (verified):** The file header at `doom.go:25–31` explicitly documents the seam as a test-only override of the production `cp.GetById()(characterId)` call site. Without this seam, the handler would issue a real HTTP call to atlas-character during unit tests. The implementer brief's NOTE 1 sanctioned this.

**Audit conclusion:** Sanctioned and architecturally consistent (matches the existing pattern of the other four seams: package-level var, swapped via `t.Cleanup` restoration in `installFakes` at `doom_test.go:38–67`).

### Task D — int16 bbox arithmetic

**Deviation:** The bbox formula in `bbox.go:20–28` casts `point.X`/`point.Y` to `int16` and performs all arithmetic in `int16`, where Cosmic uses Java `int` (32-bit signed).

**Rationale (verified):** `point.X` and `point.Y` are themselves typedefs of `int16` in `libs/atlas-constants/point/constants.go:3,5`, so the cast is a same-width same-sign no-op. The `int16` overflow boundary at ±32,768 is well outside the coordinate range of any v83 map (Maple's largest fields top out around ±10k).

**Audit conclusion:** Sanctioned. Numerically equivalent to Cosmic for the actual coordinate domain.

## Build & Test Results

| Service                                  | Build | Tests | Notes |
|------------------------------------------|-------|-------|-------|
| `services/atlas-monsters/atlas.com/monsters` | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` returns 0 FAIL. `ok atlas-monsters/monster 223.297s` includes the 5 new `TestGetInFieldRect_*` cases. |
| `services/atlas-channel/atlas.com/channel`   | PASS | PASS | `go build ./...` clean; `go test ./... -count=1` returns 0 FAIL across 53 packages. New `ok atlas-channel/skill/handler/doom 0.009s` covers the 8 new doom tests (4 bbox + 4 handler). |
| `services/atlas-data/atlas.com/data`         | PASS | PASS | No v2 source changes; smoke verification confirmed `go build ./...` clean and `go test ./... -count=1` clean (`ok atlas-data/skill 0.090s`). |

## Plan-Adherence Detail Checks

- **Revert completeness (Task R):** `git diff 9f1b14a00..85fbc861a -- services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` shows only deletions matching commit `cceb1de3f`'s diff. No re-introduction in any later v2 commit. Verified by hash equality between the worktree's on-disk file (md5 `bb6f210a5528b2a1be79a8b60011cb14`) and the HEAD blob (`a709f7dac021798f2701f062ba1cab547605dd79`).
- **REST endpoint shape (Task A):** Matches the user's spec exactly: `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/monsters/in-rect?x1=&y1=&x2=&y2=&limit=` (verified at `services/atlas-monsters/atlas.com/monsters/world/resource.go:33`). Required-vs-optional discipline: x1/y1/x2/y2 return 400 if missing; limit defaults to 0 (no cap).
- **Doom log uniqueness (Task F step 2 carry-over from v1):** `grep -rn '"Doom: caster=\[' services/` returns two hits — `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go:126` (per-cast summary, NEW) and `services/atlas-channel/atlas.com/channel/monster/processor.go:86` (per-mob, carried from v1 Task 8). Both useful and distinct; the per-cast line is the load-bearing observability for v2.
- **Registration wiring (Task D):** Registrations package is blank-imported by `main.go:53`; `registrations.go:7` blank-imports the doom subpackage; `doom.go:21–23` `init()` calls `channelhandler.Register(skill2.PriestDoomId, Apply)`. Confirmed handler will fire at runtime.
- **atlas-constants reuse (DOM-21):** Production code in `doom.go` uses `skill2.PriestDoomId` (twice), `monster2.StatusDoom`, `monster2.ReflectKindMagical`. `common.go` uses `inventoryconst.TypeFromItemId`, `itemconst.Id`, `slot.Position`, `charcon.Id`. The only raw literal in v2 is `2311005` in `doom_test.go:169` (test-side, cosmetic) — minor inheritance of the v1 audit's DOM-21 nit pattern; not a regression.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending Task F manual checklist; no blocking code work remains)

## Action Items

None. The three deviations are sanctioned, documented (in commit messages and/or file headers), and preserve the v2 plan's behavioral intent. The Task C deferred unit tests are the only meaningful coverage gap; manual verification at Task F handoff is the agreed-upon mitigation.
