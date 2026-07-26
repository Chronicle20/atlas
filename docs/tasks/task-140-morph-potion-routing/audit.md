# Plan Audit — task-140-morph-potion-routing

**Plan Path:** docs/tasks/task-140-morph-potion-routing/plan.md
**Audit Date:** 2026-07-25
**Branch:** task-140-morph-potion-routing
**Base Branch:** main

<!-- BEGIN plan-adherence-reviewer section -->

## Executive Summary

All 6 planned tasks were faithfully implemented, matching the plan's prescribed code verbatim in every case checked (getter signature, `morph.go` seam, `computeEffectPlan` extraction, FR-7 precedence branch, routing switch case, and the `TestUsesStandardConsumer` table rows). During Task 6's live verification, the executor discovered a genuine production blocker outside the plan's stated scope — atlas-data's WZ reader used the wrong field name (`"prob"` instead of `"prop"`), silently zeroing every `morphRandom` weight table — and the user explicitly authorized a 7th, unplanned commit to fix it on this branch. That fix is exactly the one-word field-name correction plus a tightly-scoped regression test; no other atlas-data code was touched. `go build`/`go vet`/`go test -race` are clean in both `services/atlas-consumables/atlas.com/consumables` and `services/atlas-data/atlas.com/data`; `redis-key-guard.sh` and `goroutine-guard.sh` are clean; `lint.sh --check` scoped to the two touched modules reports 0 issues (the only failing target, `ui:node-missing`, is an unrelated pre-existing environment gap — no atlas-ui files are in this diff). Diff scope is exactly the 8 code files plus task docs — no incidental cross-service edits.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `Morphs()` getter on data-side consumable model (FR-4) | DONE | `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go:190-195` — `func (m Model) Morphs() map[uint32]uint32 { return m.morphs }`, placed directly after `MonsterSummons()` (line 186) exactly as planned. `model_test.go` has `TestMorphsGetter` + `TestMorphsGetter_Empty`, both PASS. Commit `52b14136c`. |
| 2 | Weighted-random morph selection seam `morph.go` (FR-5, FR-6) | DONE | `consumable/morph.go` matches the plan's code block verbatim: `selectMorph(morphs, roll) (uint32, bool)` sorts ids ascending then walks cumulative weight; `rollMorph(morphs) (uint32, error)` sums weights, errors on zero total, draws via `crypto/rand`/`math/big`, delegates to `selectMorph`. All 7 planned tests (`TestSelectMorph_ExhaustiveWeighting`, `_WeightsNotSummingTo100`, `_EmptyTable`, `_AllZeroWeights`, `_ZeroWeightEntrySkipped`, `TestRollMorph_ZeroTotalErrors`, `_ResultAlwaysTableKey`) present and PASS. Commit `1cb082260`. |
| 3 | Extract `computeEffectPlan` from `ApplyItemEffects` (behavior-preserving) | DONE | `processor.go:137-211` — `effectPlan` struct and `computeEffectPlan` match the plan's code verbatim (cures → hp/mp → statups → duration, same field order/comments). `ApplyItemEffects` (`processor.go:216-244`) reduced to `plan := computeEffectPlan(l, c, ci)` followed by the same three-phase execution (cure, HP/MP, single `bp.Apply`) with the task-051 D3 comment intact. `TestComputeEffectPlan_CurePotWithHp`, `_StatPotWithTime`, `_HpRecoveryPercent` (T8 regression set) PASS. Commit `906d5cb82`. |
| 4 | Morph branch in `computeEffectPlan` with FR-7 precedence | DONE | `processor.go:197-205` — `if val, ok := ci.GetSpec(SpecTypeMorph); ok && val > 0 { … } else if len(ci.Morphs()) > 0 { rollMorph(...) ; on error, l.WithError(err).Warnf(...) and skip }`. Structurally double-apply-proof (`else if`). `TestComputeEffectPlan_FixedMorphWithHp`, `_RandomMorphOnly`, `_FixedMorphPrecedence`, `_ZeroWeightMorphTableSkipsMorphOnly` all present and PASS. Commit `ed827c2dc`. |
| 5 | Route classification 221 through `ConsumeStandard` (FR-1, FR-2) | DONE | `processor.go:104-111` — `usesStandardConsumer` switch adds `item2.ClassificationConsumableTransformation` alongside the untouched raw `Classification(200/201/202/205)` literals; doc comment updated to name transformation potions. `TestUsesStandardConsumer` (`processor_test.go:328-358`) has the three new 221 rows (`2210000`, `2211000`, `2212000`, all `want: true`) immediately after the `"all cure potion (205)"` row, exactly as the plan specified; the `2040727`/204-scroll `false` row is untouched. Commit `bcb740c3c`. |
| 6 | Full verification, follow-up backlog filing, acceptance sweep | DONE | Steps 1-3 (test/vet/build + redis-key-guard + diff-scope) re-verified independently (see Build & Test Results below) — all clean. Step 3b legacy sweep recorded in `execution-findings.md` (per-version table: v48/v61 fixed-morph only, v72/v79/v83 have both morphRandom items present but weight-zeroed — this is what surfaced the Task 7 blocker). Step 4 backlog note confirmed filed in the main-repo checkout at `docs/research/missing-features/items-and-consumables.md:163-175` (untracked on this branch, per plan). Step 6 code review partially executed: `backend-guidelines-reviewer` ran and is present in `audit.md` (PASS, no blocking findings); this plan-adherence pass is the second half of that gate. |
| 7 (added, sanctioned) | atlas-data `prob`→`prop` WZ field-name fix + regression test | DONE | `services/atlas-data/atlas.com/data/consumable/reader.go:158-159` — `prop := uint32(mo.GetIntegerWithDefault("prop", 0)); m.Morphs[id] = prop` (was `"prob"`). Confirmed the two other `"prob"` occurrences in the same file (line 103, monster-summon `Probability`; line 175, reward `Prob`) are genuinely different WZ fields, untouched. `reader_test.go:1085-1172` (`TestReaderMorphRandomProp`) pins the real item-2211000-shaped table (`morph→prop` pairs summing to 100, `box.Morphs[23]==15` regression anchor) and PASSes. Commit `4b24dddd2`, findings recorded in `65e29d8fc`. |

**Completion Rate:** 6/6 plan tasks (100%) + 1 sanctioned added task, all DONE.
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task was skipped or left partial. The one scope deviation (atlas-data touched, contra the plan's Global Constraints / PRD acceptance criterion #6) is not an unapproved skip — `execution-findings.md` documents the discovery and the explicit user decision to fix it on this branch, and the resulting diff is exactly the minimal field-name fix plus its test (verified above), not incidental scope creep.

**Minor process note (non-blocking):** every checkbox in `plan.md` remains `- [ ]` (unchecked) despite all six tasks being fully implemented and committed per `execution-findings.md`'s per-task commit table. This is a documentation-hygiene gap, not a functional one — the actual completion evidence (commits, tests, diff) is unambiguous — but the plan file itself does not reflect its own execution state.

## Build & Test Results

| Module | Build | Vet | Tests (`-race -count=1`) | Notes |
|--------|-------|-----|---------------------------|-------|
| `services/atlas-consumables/atlas.com/consumables` | PASS | PASS | PASS | Full suite green, incl. all new `TestSelectMorph_*`, `TestRollMorph_*`, `TestComputeEffectPlan_*`, `TestMorphsGetter*`, and the extended `TestUsesStandardConsumer` table. |
| `services/atlas-data/atlas.com/data` | PASS | PASS | PASS | Full suite green (32 packages), incl. new `TestReaderMorphRandomProp`. |
| `tools/redis-key-guard.sh` (repo root) | — | — | exit 0 | Clean. |
| `tools/goroutine-guard.sh` (repo root) | — | — | exit 0 | Clean. |
| `tools/lint.sh --check` (scoped to the two touched modules) | — | — | 1 unrelated failing target | Only failing target is `ui:node-missing` (Node/nvm not available in this shell — atlas-ui is not in this diff at all, confirmed by the diff-scope check). Both touched Go modules report "0 issues." A full-tree (unscoped) `lint.sh --check` run also fails on `libs/atlas-database`/`libs/atlas-socket` with `Error: parallel golangci-lint is running` plus warnings referencing files under other, now-deleted worktrees (`task-176-gm-hide-controller-relinquish`, `task-123-megaphones-maple-tv`) — these are pre-existing tooling/environment artifacts unconnected to task-140's diff, not new lint violations in the changed files. |
| Diff-scope check | PASS | — | — | `git diff --name-only $(git merge-base HEAD main)..HEAD` returns only files under `services/atlas-consumables/`, `services/atlas-data/atlas.com/data/consumable/{reader.go,reader_test.go}`, and `docs/tasks/task-140-morph-potion-routing/`. |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the standard `frontend-guidelines-reviewer` N/A confirmation — no atlas-ui files touched — and final `superpowers:finishing-a-development-branch` walkthrough, neither of which this audit gates)

## Action Items

1. (Optional, cosmetic) Check off the `- [ ]` boxes in `plan.md` to reflect actual completion state, or note in the plan file that tracking moved to `execution-findings.md`.
2. (Optional, cosmetic) `context.md`/`design.md` §3.1 assert task-131's `reward.go` is "not in this worktree" — already merged to `main` as of this branch's base; already flagged as non-blocking in the backend-guidelines section above, no code impact.
3. No functional or scope-adherence fixes required before merge.

<!-- END plan-adherence-reviewer section -->

<!-- BEGIN backend-guidelines-reviewer section -->

## Backend Guidelines Review (DOM-*/SUB-*/SEC-*)

- **Worktree:** `.worktrees/task-140-morph-potion-routing`
- **Branch:** `task-140-morph-potion-routing` (confirmed via `git branch --show-current`)
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/`
- **Date:** 2026-07-25
- **Scope (per diff `review-b64ee7936..65e29d8fc.diff`):**
  - `services/atlas-consumables/atlas.com/consumables/consumable/morph.go` (new)
  - `services/atlas-consumables/atlas.com/consumables/consumable/morph_test.go` (new)
  - `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (modified — `computeEffectPlan` extraction + morph-random branch + 221 routing)
  - `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` (modified)
  - `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go` (modified — `Morphs()` getter)
  - `services/atlas-consumables/atlas.com/consumables/data/consumable/model_test.go` (new)
  - `services/atlas-data/atlas.com/data/consumable/reader.go` (modified — `"prob"` → `"prop"` one-word fix)
  - `services/atlas-data/atlas.com/data/consumable/reader_test.go` (modified — regression test)

### Objective gate (re-run, not just trusted from execution-findings.md)

```
cd services/atlas-consumables/atlas.com/consumables && go build ./...   → clean, no output
cd services/atlas-consumables/atlas.com/consumables && go vet ./...     → clean, no output
cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/... ./data/consumable/... -count=1 -v
  → PASS: TestComputeEffectPlan_CurePotWithHp, TestComputeEffectPlan_StatPotWithTime,
    TestComputeEffectPlan_HpRecoveryPercent, TestComputeEffectPlan_FixedMorphWithHp,
    TestComputeEffectPlan_RandomMorphOnly, TestComputeEffectPlan_FixedMorphPrecedence,
    TestComputeEffectPlan_ZeroWeightMorphTableSkipsMorphOnly, TestUsesStandardConsumer
    (incl. new 221 rows), TestMorphsGetter, TestMorphsGetter_Empty — all PASS

cd services/atlas-data/atlas.com/data && go build ./...  → clean
cd services/atlas-data/atlas.com/data && go vet ./...    → clean
cd services/atlas-data/atlas.com/data && go test ./consumable/... -count=1 -v
  → PASS: TestReaderMorphRandomProp (new regression test) + all pre-existing reader tests
```

Build/Test gate: **PASS** in both modules. Proceeding to mechanical checklist.

### Domain classification (Phase 2)

Neither touched package is a "domain package" by the strict DOM checklist trigger (no `model.go` added/changed in `consumable`; `data/consumable` has `model.go` but this diff only adds one getter to an already-existing REST-client/read-model package with no `entity.go`/`builder.go`/`resource.go` — it fetches from `atlas-data`, doesn't own persistence). Both are classified **support packages** and run the File Responsibilities Checklist per the mandatory rule that support packages are not blanket-exempt.

### File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `Processor`/`ProcessorImpl` logic stays in `processor.go` | PASS | `consumable/processor.go:74-95` (interface/struct, untouched) and the new `effectPlan`/`computeEffectPlan` (`processor.go:137-211`) all live in `processor.go`. No `ProcessorImpl` method or interface symbol was added to `morph.go`, `model.go`, or any bare-name file. |
| FILE-06 | No package-named catch-all file bundling ≥2 responsibilities | PASS | `consumable/morph.go:1-56` contains only `selectMorph` + `rollMorph` — free functions, not `ProcessorImpl` methods, not `RestModel`, not request funcs, not entity code. Zero of the tracked responsibilities are present, so it qualifies as the file-responsibilities.md "genuine single-purpose utility" exception, not a `<pkg>.go` collapse. Mirrors the pre-existing sibling `consumable/reward.go` (same package, same shape: `rollReward`, `validateRewardTable`, `grantQuantity`, `rewardExpiration`, `substituteWorldMsg` — confirmed present at `consumable/reward.go:1-84`, task-131, already merged to main and NOT relied on here for a prevalence exemption — it independently satisfies the same "no tracked responsibility present" test). |
| FILE-05 | Domain `Model` getters in `model.go` | PASS | `data/consumable/model.go:190-195` — `func (m Model) Morphs() map[uint32]uint32` added directly after the existing `MonsterSummons()` accessor, same file, same convention (returns the internal map reference, read-only by caller contract — matches the documented `MonsterSummons()` pattern one line above it). |

### DOM-* checks applicable to the diff

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No duplication of atlas-constants types/constants | PASS | `processor.go:107` routes classification 221 via the named constant `item2.ClassificationConsumableTransformation`, confirmed defined at `libs/atlas-constants/item/constants.go:39` (`Classification(221)`). Verified by grep that `libs/atlas-constants/item/constants.go` has **no** named constants for classifications 200/201/202/205 (only 203/204/206/207/208/210/212/221/226/228/229/231/233/238 are named in that block) — the pre-existing raw literals `item2.Classification(200)`, `(201)`, `(202)`, `(205)` at `processor.go:106` are untouched by this diff and match the documented "no named constant exists, renaming out of scope" carve-out (design.md §2); they are not a new violation introduced here. |
| DOM-06 | Processor-adjacent pure function accepts `FieldLogger`, not `*logrus.Logger` | PASS | `processor.go:150` — `func computeEffectPlan(l logrus.FieldLogger, c character.Model, ci consumable3.Model) effectPlan`. |
| — | Zero-total morph table: warn + skip, no panic, no fabricated id | PASS | `morph.go:41-43` — `total == 0` returns an explicit `error`, never a fabricated id. `morph.go:50-54` — even the "unreachable" `!ok` branch from `selectMorph` returns an error rather than a zero-value id (comment: "never return a fabricated morph id"). Caller `processor.go:199-205` — on `rollMorph` error, only `l.WithError(err).Warnf(...)` is emitted and the morph statup is omitted; `plan.duration` (line 206-208) and all other already-computed fields (`hpChanges`, `mpChanges`, `cureTypes`) are unaffected — consumption/other-spec application is not short-circuited, matching the "errors logged, consumption not rolled back" semantics carried over from the pre-existing code. |
| — | `crypto/rand` vs `math/rand` file separation | PASS | `morph.go:4` imports `crypto/rand` exclusively for `rollMorph`; `processor.go:31` retains its pre-existing `math/rand` import (used elsewhere in the file, e.g. `ConsumeSummoningSack`, out of this diff's scope). `go build ./...` confirms no import collision — the split file is what makes that coexistence possible, as documented in plan.md Task 2. |
| — | FR-7 precedence structural (fixed `morph` wins, no double-apply) | PASS | `processor.go:197-205` — `if val, ok := ci.GetSpec(SpecTypeMorph); ok && val > 0 { … } else if len(ci.Morphs()) > 0 { … }`. The `else if` makes concurrent application of both branches impossible by construction (not just by convention). |
| DOM-20 | Table-driven tests | WARN (non-blocking) | `processor_test.go` extends the existing table-driven `TestUsesStandardConsumer` correctly (new 221 rows appended to the `cases` slice, `t.Run` per case — confirmed in the running suite as `TestUsesStandardConsumer/morph_potion_(221)` etc.). However, the seven new `TestComputeEffectPlan_*` functions and the six new `TestSelectMorph_*`/`TestRollMorph_*` functions are each standalone `func TestX(t *testing.T)` rather than a `tests := []struct{...}` + `t.Run` table. The guideline (`testing-guide.md:18`) states "Prefer table-driven tests" (soft preference, not a MUST per the checklist's binary wording elsewhere). This mirrors the pre-existing sibling `consumable/reward_test.go`/`consumable/vega_test.go` style in the same package (non-table, per-scenario functions for pure-function pinning) — noted for completeness, not relied on as an exemption; flagging as non-blocking because the guideline's own severity language is "prefer," not a hard rule. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS / N/A | None of the new or modified tests reach an emit path. All seven `TestComputeEffectPlan_*` tests call the pure `computeEffectPlan(...)` directly (never `ApplyItemEffects`, which is the only function in this diff that constructs `buff.NewProcessor(...)` and calls `bp.Apply(...)`). Confirmed by test run: full `./consumable/...` + `./data/consumable/...` suite completes in 0.029s / 0.008s — no ~42s per-emit stalls, consistent with no live Kafka path being exercised by the new tests. |
| — | No test-only constructors (project rule + design §4.4) | PASS | `extractConsumable(t, rm)` (processor_test.go, per diff) wraps the **public** `consumable3.Extract(rm)` — not a private/test-only constructor. `character.NewModelBuilder()` (public Builder) is used for character fixtures. `data/consumable/model_test.go` builds `RestModel{...}` literals and runs them through the public `Extract` (`TestMorphsGetter`, `TestMorphsGetter_Empty`). No `*_testhelpers.go` file was added. |

### atlas-data fix (`reader.go` / `reader_test.go`)

| Check | Status | Evidence |
|-------|--------|----------|
| Field-name fix scoped correctly, no responsibility drift | PASS | `services/atlas-data/atlas.com/data/consumable/reader.go:158-159` — `prop := uint32(mo.GetIntegerWithDefault("prop", 0)); m.Morphs[id] = prop` (was `"prob"`). One-word rename, no structural change; unrelated `"prob"` usages for reward/monster-summon fields at `reader.go:103` and `reader.go:175` are untouched and are genuinely different WZ fields (not the same bug). |
| Regression test follows existing reader-test conventions | PASS | `reader_test.go:1085-1157` (`TestReaderMorphRandomProp`) uses the same `Read(l)(xml.FromByteArrayProvider([]byte(...)))` + inline XML fixture-constant pattern as the pre-existing `TestReader` (`reader_test.go:908`, uses `testXML` const at line 13) and `TestReaderRewardFields` (`reader_test.go:1014`) — no manual/ad-hoc parsing, no new test-only production code path. |
| Scope-conflict disclosure | Out of DOM/SUB/SEC scope, noted for completeness | `execution-findings.md` (committed in this diff) explicitly flags that this fix crosses the plan's "atlas-consumables only" acceptance criterion and states it was escalated to and authorized by the user — consistent with the task prompt's framing. Not re-litigated here since it is a scope/process decision already resolved outside the guidelines checklist. |

### Sub-Domain / External-HTTP-Client / Security / Scaffolding checklists

- **SUB-*:** N/A — no `resource.go` added or changed in this diff.
- **EXT-*:** N/A — no new cross-service `requests.GetRequest[T]`/`PostRequest[T]` call site added; `data/consumable/requests.go` (the package's existing atlas-data client) is untouched by this diff.
- **SEC-*:** N/A — atlas-consumables/atlas-data are not auth/token services.
- **Scaffolding:** N/A — no new service, no new atlas-channel writer/handler.

### Non-checklist observation (informational only, not a DOM/SUB/SEC finding)

`context.md` (committed in this diff) and `design.md` §3.1 assert task-131's `consumable/reward.go` "lives on the unmerged `task-131-random-reward-items` branch — it is NOT in this worktree." This is factually incorrect as of this branch's current base: `reward.go` is present and its tests pass (`git log --oneline -1 -- consumable/reward.go` → `fcfc45c8b feat(task-131): random reward items … (#907)`, already merged to `main`). This did not cause any code defect — `morph.go` independently satisfies the same file-responsibilities exception `reward.go` does, and the crypto/rand-seam design was followed correctly regardless of the doc's stale premise — so it is not scored as a DOM/SUB/SEC finding, only flagged for documentation accuracy.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- DOM-20: `TestComputeEffectPlan_*` / `TestSelectMorph_*` / `TestRollMorph_*` are standalone test functions rather than table-driven; consider consolidating into `tests := []struct{...}` + `t.Run` tables per the guideline's soft preference, if a future touch of this file is warranted.

### Overall: PASS

Build clean, tests clean (re-verified independently), zero blocking DOM/SUB/SEC/FILE findings. One non-blocking style observation (DOM-20) does not prevent PASS per "Rules for Status Assignment" (only FAIL-status checks block PASS; the DOM-20 item above is scored WARN, not FAIL, because the guideline itself uses "prefer" rather than a binary must).

<!-- END backend-guidelines-reviewer section -->
