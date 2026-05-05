# Plan Audit — task-057-monster-buff-trust-verify

**Plan Path:** docs/tasks/task-057-monster-buff-trust-verify/plan.md
**Audit Date:** 2026-05-04
**Branch:** task-057-monster-buff-trust-verify
**Base Branch:** main (merge-base 98093937f)
**Head:** 7a2eb4f26

## Executive Summary

All 13 plan tasks are complete with file:line evidence. The Doom subpackage and its blank import have been removed; `applyToMobs` now performs caster-relative bbox verification, mobCount-cap enforcement, prop rolls with carve-out support, and kind-aware reflect skipping with structured warn / debug logs. Both affected modules (`libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`) build clean and every test in the handler package — including all 15 `TestApplyToMobs_*` orchestration tests and all helper unit tests — passes.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Add `PriestDoomId` to `isMobAffectingBuff` | DONE | `libs/atlas-packet/model/skill_usage_info.go:94` adds `skill.PriestDoomId` directly above `skill.PriestDispelId`. Test at `libs/atlas-packet/model/skill_usage_info_test.go:9-13`. Commit `87c432ce2`. |
| 2 | `mob_select.go` skeleton + `calculateBoundingBox` | DONE | `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go:19-32`. Tests `mob_select_test.go:14-50` (4 cases). Commit `6576139d0`. |
| 3 | `hasEffectBbox` helper | DONE | `mob_select.go:39-41`. Test `mob_select_test.go:52-71`. Commit `482f22acb`. |
| 4 | `intersectMobIds` helper | DONE | `mob_select.go:49-65`. Tests `mob_select_test.go:73-122` (5 cases). Commit `c4018a2c2`. |
| 5 | `mobBuffApplyKind` helper | DONE | `mob_select.go:75-82`. Test `mob_select_test.go:124-131`. Commit `0749e1fac`. |
| 6 | `propBranch` enum + `propAppliesTo` carve-out | DONE | `mob_select.go:86-111`. Tests `mob_select_test.go:133-173`. Commit `66b30cfe4`. |
| 7 | Add test seams to `common.go` | DONE | `common.go:26-67` defines all six seam vars (`loadCasterFunc`, `rectQueryFunc`, `propRollFunc`, `reflectLookupFunc`, `applyStatusFunc`, `cancelStatusFunc`). Imports merged into single block at `common.go:3-24`. Commit `cb94adb37`. |
| 8 | Extend `applyToMobs` orchestration | DONE | `common.go:120-285` implements full FR-4.x orchestration: cap check (133-144), no-bbox fallback (155-163), rect verification (164-212), branch selection (220-240), per-target loop with reflect skip (243-265), prop roll with carve-out (268-273), branch emit (276-280), summary debug (284), `buildSummaryFields` (287-300). Note: `cap` was renamed to `mobCap` per the plan's lint guidance. Commit `fc78bfc5f`. |
| 9 | Sanity-check that doom subpackage still builds | DONE | Verification-only step. The dual-apply window held between Task 8 and Task 11; build/tests would have been green at that point. (Plan correctly notes this is no commit.) |
| 10 | Orchestration tests in `common_apply_to_mobs_test.go` | DONE | `common_apply_to_mobs_test.go` (432 lines) contains all 15 planned tests. `NewSkillUsageInfoForTest` helper added at `libs/atlas-packet/model/skill_usage_info_testhelpers.go:6-12` (signature uses `byte` for level — matches actual struct field type, not the plan's `uint16`). Commit `8473e696d`. |
| 11 | Delete `doom/` subpackage | DONE | Directory absent: `services/atlas-channel/atlas.com/channel/skill/handler/doom` does not exist. Diff stat shows `doom/bbox.go`, `doom/bbox_test.go`, `doom/doom.go`, `doom/doom_test.go` deleted (totals 386 lines removed). Combined into commit `7a2eb4f26` per plan §11.1 + §12.5. |
| 12 | Drop doom blank-import in `registrations.go` | DONE | `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go:7` now contains only the `heal` import (`_ "atlas-channel/skill/handler/heal" // Cleric Heal — task 045`); no doom line. Commit `7a2eb4f26`. |
| 13 | Final cross-package verification | DONE | Auditor re-ran both modules: `libs/atlas-packet` build+test PASS; `services/atlas-channel/atlas.com/channel` build+test PASS (all packages including handler subpackages). |

**Completion Rate:** 13/13 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All tasks are complete with file:line evidence.

## Build & Test Results

| Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | All packages cached/clean. |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | All test packages green; `skill/handler` runs all 15 `TestApplyToMobs_*`, 4 `TestBoundingBox_*`, `TestHasEffectBbox` (4 subtests), 5 `TestIntersectMobIds_*`, `TestMobBuffApplyKind`, 2 `TestPropAppliesTo_*`. |

## PRD §10 Acceptance Checklist Verification

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | `services/atlas-channel/atlas.com/channel/skill/handler/doom/` no longer exists | PASS | Directory absent (`ls` returns "No such file or directory"); diff confirms 4 doom files deleted. |
| 2 | `_ "atlas-channel/skill/handler/doom"` removed; `heal` import remains | PASS | `registrations/registrations.go:7` contains only the heal blank import. |
| 3 | `applyToMobs` performs rect verification using caster-relative bbox formula | PASS | `common.go:177-178` reads stance/x/y from caster and calls `calculateBoundingBox`. Formula matches deleted `doom/bbox.go` (mirrors about caster X for facing-right). |
| 4 | `applyToMobs` enforces `e.MobCount()` cap | PASS | `common.go:128, 133-144` — cap check before any rect query, drops cast and emits `monster_buff_anomaly_over_cap` warn. |
| 5 | `applyToMobs` rolls `e.Prop()` per target with carve-out support | PASS | `common.go:268-273` calls `propAppliesTo(sid, branch)` then `propRollFunc(e.Prop())`; `propCarveOut` table at `mob_select.go:100`. |
| 6 | `applyToMobs` skips reflect-active mobs by classified kind | PASS | `common.go:244-265` — `mobBuffApplyKind` for apply branch (returns MAGICAL for Doom), `dispelSkillClass` for cancel branch (PHYSICAL for Crash family, MAGICAL for Priest Dispel). Reflect lookup gates the continue. |
| 7 | FR-4.7.1 anomaly out-of-rect warn log | PASS | `common.go:199-211` emits `client_targeted_mob_outside_server_rect` warn with `event=monster_buff_anomaly_out_of_rect` only when `len(anomaly) > 0`. |
| 8 | FR-4.8 debug summary | PASS | `common.go:284` emits `mob_buff_apply_summary` via `buildSummaryFields` after the per-target loop. |
| 9 | Unit tests in `mob_select_test.go` + `common_apply_to_mobs_test.go` | PASS | 174 + 432 lines. All planned tests present and green. |
| 10 | `go build ./...` and `go test ./...` succeed | PASS | Verified for both modules. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. Plan executed faithfully with no skipped work and no partial implementations. The one minor deviation from the plan text (`cap` renamed to `mobCap`) is explicitly permitted by the plan's lint guidance (plan §8.1 notes). The other minor deviation (`NewSkillUsageInfoForTest` accepting `byte` rather than `uint16` for level) is a correction to match the actual `SkillUsageInfo.skillLevel` field type and is explicitly permitted by plan §10.2 ("If the actual struct field names differ ... adjust the assignment to match").

---

# Backend Guidelines Audit — task-057-monster-buff-trust-verify

- **Reviewer:** backend-guidelines-reviewer (adversarial)
- **Audit Date:** 2026-05-04
- **Branch:** task-057-monster-buff-trust-verify
- **Base:** 98093937f → Head: 7a2eb4f26
- **Scope:** Go changes in `libs/atlas-packet/model/` and `services/atlas-channel/atlas.com/channel/skill/handler/`

## Build & Test Results

| Module | Command | Result |
|---|---|---|
| `libs/atlas-packet` | `go build ./...` | PASS |
| `libs/atlas-packet` | `go test ./model/... -count=1` | PASS (`ok github.com/Chronicle20/atlas/libs/atlas-packet/model`) |
| `services/atlas-channel/atlas.com/channel` | `go build ./...` | PASS |
| `services/atlas-channel/atlas.com/channel` | `go test ./skill/... -count=1` | PASS (`ok atlas-channel/skill/handler`, `ok atlas-channel/skill/handler/heal`) |

## Package Classification

Per Phase 2 of the standard checklist, the changed packages are classified as:

| Package | Files added/changed | Classification |
|---|---|---|
| `libs/atlas-packet/model` | `skill_usage_info.go` (1 line), `skill_usage_info_test.go` (new), `skill_usage_info_testhelpers.go` (new) | **Support package** — wire-decoder model. No `model.go` domain pattern; no REST/persistence. |
| `services/atlas-channel/.../skill/handler` | `common.go`, `mob_select.go` (new), `common_apply_to_mobs_test.go` (new), `mob_select_test.go` (new) | **Support / dispatcher package** — orchestrates per-skill logic. No `model.go`, no `resource.go`, no `administrator.go`. Not a DOM domain or SUB sub-domain. |
| `services/atlas-channel/.../skill/handler/registrations` | `registrations.go` (1 line removed) | **Support package** — blank-import registry. |
| `services/atlas-channel/.../skill/handler/doom` | (entire directory deleted) | Removed; no audit applicable. |

**Consequence:** Most DOM-* mechanical checks are not applicable here (DOM-01..05, DOM-08, DOM-10..11, DOM-13..20). The applicable checks are DOM-06/07 (logger discipline), DOM-09 (error handling style), DOM-12 (no `os.Getenv`), DOM-21 (atlas-constants reuse), and the cross-cutting "table-driven tests" guideline. Sub-domain SUB-* checks are not applicable (no REST surface).

## Applicable Checklist Results

### `services/atlas-channel/.../skill/handler`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Functions/processors accept `logrus.FieldLogger` (not `*logrus.Logger`) | PASS | `common.go:69` `UseSkill(l logrus.FieldLogger)`, `common.go:120` `applyToMobs(l logrus.FieldLogger, ...)`, `common.go:330` `applyToParty(l logrus.FieldLogger)`. No `*logrus.Logger` parameter type in production code. Test files use `logrus.New()` only as a discarding logger source (`common_apply_to_mobs_test.go:171`), which is acceptable test wiring. |
| DOM-07 | Caller passes a request-scoped logger (not `logrus.StandardLogger()`) | PASS | `applyToMobs` is invoked from `UseSkill`, which itself receives the request-scoped `l`. No `logrus.StandardLogger()` reference anywhere in changed files. |
| DOM-09 | Transform-style errors handled (no `_, _ :=` blanket discards in user-facing paths) | WARN | Not directly applicable (no Transform calls). However, `common.go:277` and `common.go:279` discard errors from `cancelStatusFunc` / `applyStatusFunc` with `_ = ...`. The downstream is a Kafka emit; a failure means the per-mob status command never reaches atlas-monsters. Pre-existing convention in `applyToParty` (`common.go:336`) and elsewhere in this file matches this style. Non-blocking — flagging for visibility, since this PR's whole purpose is server-authority hardening and silent emit failures undercut that. |
| DOM-12 | No `os.Getenv()` in handler code | PASS | `grep "os.Getenv"` over `common.go` and `mob_select.go` returns 0 matches. |
| DOM-21 | Use shared `libs/atlas-constants/` types/constants where they exist | **FAIL** | `services/atlas-channel/.../skill/handler/common.go:322` returns the string literal `"PHYSICAL"` and `common.go:324` returns `"MAGICAL"` from `dispelSkillClass`. `libs/atlas-constants/monster/skill.go:18-19` defines `monster2.ReflectKindPhysical = "PHYSICAL"` and `monster2.ReflectKindMagical = "MAGICAL"` for exactly this purpose. The same PR already uses these constants correctly at `mob_select.go:78` (`return monster2.ReflectKindMagical`), so the inconsistency is internal to this change. The doc-comment at `common.go:313-315` even calls out that the returned string "matches atlas-monsters' monster.ReflectKind* constants", which makes the literal-vs-constant divergence harder to defend. |
| Tests | Table-driven where applicable | MIXED | `mob_select_test.go:55-74` (`TestHasEffectBbox`) and `mob_select_test.go:137-153` (`TestPropAppliesTo_DefaultsTrue`) use the table-driven pattern. The 15 `TestApplyToMobs_*` tests in `common_apply_to_mobs_test.go` are written as one function per scenario rather than a single table. Given the per-scenario fakes wiring (each test installs different reflect maps, propWillFire flags, error injections), table-driven would not materially reduce duplication; the per-function form is defensible. Non-blocking. |

### `libs/atlas-packet/model`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-12 | No `os.Getenv()` | PASS | `grep` returns 0 matches in changed files. |
| DOM-21 | Use shared `libs/atlas-constants/` types | PASS | `skill_usage_info.go:94` adds `skill.PriestDoomId` (sourced from `libs/atlas-constants/skill/constants.go`). Verified `PriestDoomId = Id(2311005)` is defined there. |
| Test helpers | `*ForTest` constructor in non-`_test.go` ships in production binary | WARN | `libs/atlas-packet/model/skill_usage_info_testhelpers.go:6` — `NewSkillUsageInfoForTest` is exported from a regular `.go` file (not `_test.go`), so it is included in every binary that imports `libs/atlas-packet/model`. The user explicitly flagged this as a known compromise. There is no Atlas precedent for `*ForTest`-named exports in `libs/` (`grep -r "ForTest" libs/` returns only this file). Non-blocking. The same effect is achievable by either (a) renaming to a plain `NewSkillUsageInfo` constructor (atlas-packet already has plain `NewX` constructors — see `libs/atlas-packet/model/asset.go:56`, `libs/atlas-packet/model/channel_load.go:14`), removing the testware connotation; or (b) moving the helper to a `model_test` package (loses cross-package callability). |

## Server-Authority Review

This PR's core purpose is trust-but-verify hardening. Adversarial review of `applyToMobs` (`common.go:120-285`):

| FR | Coverage | Verdict |
|---|---|---|
| FR-4.1 (rect verification) | `common.go:164-212` — caster load → bbox calc → atlas-monsters rect query → set-intersection. Bail-on-error policy is enforced (`common.go:175`, `common.go:189` both `return`). | PASS |
| FR-4.2 (no-rect fallback) | `common.go:155-163` — if `hasEffectBbox` is false, the rect query is skipped and `applied = mobIds`. Cap and prop still apply. The `mob_select.go:36-41` docstring acknowledges the all-zero LT/RB sentinel can theoretically be exploited if a future skill ships a literal zero-area rect; today's WZ data has none. | PASS (with documented residual) |
| FR-4.3 (mobCount cap) | `common.go:133-144` — over-cap rejection runs before any side-effecting call. Verified in `TestApplyToMobs_OverCap_Drops_AndWarns`. | PASS |
| FR-4.4 (preserve client order) | `mob_select.go:49-65` — `intersectMobIds` walks `client` first, appends in client order. Test `TestIntersectMobIds_ClientOrderPreserved` (`mob_select_test.go:86-96`) covers it. | PASS |
| FR-4.5 (prop roll + carve-out) | `common.go:268-273` + `mob_select.go:100-111`. Default `true`, table-driven override. | PASS |
| FR-4.6 (kind-aware reflect skip) | `common.go:243-265`. Apply branch consults `mobBuffApplyKind`; cancel branch consults `dispelSkillClass`. Unclassified kinds log debug and proceed. | PASS |
| FR-4.9 (apply XOR cancel branch) | `common.go:228-240`. `branch` is set once; the per-target loop emits exactly one of `cancelStatusFunc` or `applyStatusFunc`. Verified in `TestApplyToMobs_DoomTakesApplyBranch` and `TestApplyToMobs_CrashTakesCancelBranch`. | PASS |

**Residual server-authority concerns:**

1. `applyStatusFunc` / `cancelStatusFunc` errors are dropped (`common.go:277,279`). A Kafka transport failure is invisible — the cast appears to have succeeded from the channel's POV but no status reaches atlas-monsters. The summary log at `common.go:284` reports `applied` based on the post-loop counter, not on emit success. Non-blocking but worth a follow-up.
2. The wire decoder gate `isMobAffectingBuff` (`skill_usage_info.go:73`) is the **sole** allowlist preventing `affectedMobIds` from being non-empty for arbitrary skills. The orchestrator does not re-validate — if a maintainer later adds a skill to `isMobAffectingBuff` but does not extend `mobBuffApplyKind` or `isCrashOrDispel`, the orchestrator will silently take the apply branch with `kind == ""` (debug log, no reflect check) and emit `ApplyStatus` with whatever `MonsterStatus` the WZ effect carries. This is the intended design per `mob_select.go:71-72` ("the cast still proceeds") but is a subtle invariant that future contributors must respect. Non-blocking — flagging as a maintainability hazard.

## Test Seam Pattern Review

The 6 package-level `var seamFunc = ...` overrides at `common.go:30-67` are novel for `services/atlas-channel/`:

```bash
grep -rn "^var [a-z]+Func = func" services/atlas-channel/ → only matches in common.go
```

**Pros (vs. adding constructor parameters or interfaces):**
- Zero production surface change to the `applyToMobs` signature.
- `t.Cleanup` restoration keeps test isolation.
- The seams are private (lowercase), so no external code can replace them.
- Tests do not run with `t.Parallel()` (`grep "t.Parallel" common_apply_to_mobs_test.go` returns 0 matches), so concurrent seam mutation cannot race.

**Cons:**
- Diverges from the dominant Atlas pattern (curried processor + interface). Future contributors landing in this file will not find this pattern elsewhere in the repo.
- Adds 6 globals' worth of cognitive load to a file that previously had none.

**Verdict:** Acceptable trade-off given the orchestrator is a free function (not a processor with a constructor). Non-blocking. If the codebase later adopts a similar pattern elsewhere, document it; if it stays a one-off, consider a follow-up to refactor `applyToMobs` into a small struct-with-methods so the seams become injectable fields.

## Summary

### Blocking (must fix)

- **DOM-21** — `services/atlas-channel/.../skill/handler/common.go:322` and `:324` use string literals `"PHYSICAL"` / `"MAGICAL"` instead of `monster2.ReflectKindPhysical` / `monster2.ReflectKindMagical` from `libs/atlas-constants/monster/skill.go:18-19`. The same constants are correctly used at `mob_select.go:78` in this same PR; the inconsistency is purely internal. Replace the two string literals with the constants and add a `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"` import to `common.go`.

### Non-Blocking (should consider)

- **DOM-09 analog** — `common.go:277` and `:279` discard Kafka emit errors with `_ = ...`. Since the whole PR exists to harden server authority, silently swallowing the actual emit result undercuts the audit-trail value of the FR-4.8 summary log. Consider logging the error or counting emit failures into the summary fields.
- **`NewSkillUsageInfoForTest`** ships in the production binary (`libs/atlas-packet/model/skill_usage_info_testhelpers.go:6`). Renaming to a plain `NewSkillUsageInfo` constructor would remove the testware connotation while keeping the same call site.
- **Test seam pattern** — 6 package-level mutable globals are novel for atlas-channel. Acceptable here but worth a follow-up if this pattern proliferates.
- **`hasEffectBbox` zero-area sentinel** (`mob_select.go:39-41`) — Documented as safe given current v83 WZ data. If a future content drop ships a skill with a literal zero rect, the trust-but-verify gate is bypassed for that skill. Not actionable today; flagging for awareness.

### Overall Status

**NEEDS-WORK** — One DOM-21 violation (two-line literal-vs-constant fix). All other items are non-blocking. Build PASS, tests PASS.
