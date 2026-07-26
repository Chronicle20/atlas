# Plan Audit — task-147-attack-drain-hp-gain

**Plan Path:** docs/tasks/task-147-attack-drain-hp-gain/plan.md
**Audit Date:** 2026-07-25
**Branch:** task-147-attack-drain-hp-gain
**Base Branch:** main (merge-base `01a0a3bb003cb464bef9a5c7cbcd04e0f8c82927`)

## Executive Summary

All 5 plan tasks were implemented faithfully, in plan order, each as its own commit
(`e4c887bcc` → `9d2aaab20` → `1ad342f0f` → `31d893f2e` → `59f20e2df`). The diff is scoped
exactly as promised: one production file
(`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`, +55/-13)
and one new test file (`character_attack_drain_test.go`, +334). Every PRD §10 acceptance
criterion is met with direct evidence. All three documented deviations from the plan's literal
test snippets (two-value `Build()`, `testDrainField()` naming, a real assertion in the
emit-error-swallowed test) were applied, and applied consistently everywhere the pattern
recurs — no strays. Negative requirements hold: no `MajorVersion` gating anywhere in the
feature, exactly one TODO line removed (`// TODO Combo Drain` and all 22 other adjacent TODOs
intact), `go.mod` untouched. `go build`, `go vet`, and `go test -race -count=1` are clean in
atlas-channel (22 new subtests across 6 test functions, all pass); `tools/redis-key-guard.sh`
and `tools/goroutine-guard.sh` are clean from the repo root (both independently re-run by this
audit, exit 0). `tools/lint.sh --check` is clean tree-wide (`lint.sh: OK`) — this audit
root-caused an earlier transient FAIL on `atlas-mts` (unrelated to this branch) to a stale
golangci-lint analysis cache referencing pruned sibling worktrees; see the Build & Test
Results section for the full diagnostic chain. No blocking findings. Two non-blocking
process/nit items are listed under Action Items.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `isDrainSkill` membership helper + test | DONE | `character_attack_common.go:83-93` — `switch` over `skill3.AssassinDrainId`, `skill3.MarauderEnergyDrainId`, `skill3.ThunderBreakerStage3EnergyDrainId`, `skill3.NightWalkerStage2VampireId`; matches plan text verbatim. Test: `character_attack_drain_test.go:24-45` `TestIsDrainSkill`, 7 subtests, all pass. |
| 2 | `drainHealAmount` pure cap math + table test | DONE | `character_attack_common.go:229-250` — `uint64` internal math, floor division, monster-max-HP cap, half-effective-max-HP cap, `math.MaxInt16` clamp, zero guards on `totalDamage==0 \|\| x<=0 \|\| effectiveMaxHp==0`. Test: `character_attack_drain_test.go:47-87` `TestDrainHealAmount`, 15 subtests incl. FR-3 spot values (L30 x=45→450, L1 x=16 floor→53, Vampire L20 x=10→500, Energy Drain L20 x=20→2469) and cap/zero/int16-clamp edge cases, all pass. |
| 3 | Widen `onDamageApplied` hook; rename `loadVenomStats`→`loadEffectiveStats` | DONE | Hook signature widened at `character_attack_common.go:111` (`func(monsterId uint32, totalDamage uint32)`); sum-and-clamp logic at `197-206`. Rename applied at every site, zero stragglers: struct field `103-106`, both VENOM call sites `143`, `190`, outer closure `417-429`, deps literal `437`. `grep -rn loadVenomStats services/atlas-channel` returns nothing. New hook-behavior tests: `TestOnDamageApplied_ReceivesSummedDamageTotal`, `TestOnDamageApplied_NotCalledForZeroDamageEntry`, `TestOnDamageApplied_NotCalledForReflectedEntry` (`character_attack_drain_test.go:108-198`), all pass. |
| 4 | `drainTryHeal` orchestrator with injected collaborators + flow tests | DONE | `character_attack_common.go:323-353` — signature matches the plan's interface exactly (logger, `getMonster`, `changeHP`, `loadEffectiveStats`, `x`, `skillId`, `monsterId`, `totalDamage`, `f`, `characterId`); monster-fetch failure and zero-effective-stats both skip the heal without error; `ChangeHP` failure is logged and swallowed. 5 flow tests (`character_attack_drain_test.go:207-334`): happy path, monster-fetch-error, zero-effective-stats, emit-error-swallowed, per-monster caps (multi-target). All pass. |
| 5 | Wire drain into `processAttack`'s `onDamageApplied` closure; remove the drain TODO | DONE | Wiring at `character_attack_common.go:444-450`: MP Eater branch unchanged (`AttackType==Magic` gate), drain branch added with **no** attack-type gate (`448-449`, matches design D8 — the four skills span melee/ranged/energy) — `isDrainSkill(skill3.Id(ai.SkillId()))` then `drainTryHeal(l, mp.GetById, cp.ChangeHP, loadEffectiveStats, se.X(), ai.SkillId(), monsterId, totalDamage, s.Field(), s.CharacterId())`. TODO removed: `grep -n "TODO increase HP from Energy Drain" character_attack_common.go` → no output; TODO count 24 (at merge-base, confirmed via `git show 01a0a3bb0:...\|grep -c TODO`) → 23 (current). `// TODO Combo Drain` still present at line 515. |

**Completion Rate:** 5/5 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 5 tasks are fully implemented with passing tests.

Process nit (non-blocking): `plan.md`'s 25 step checkboxes and `prd.md`'s 10
acceptance-criteria checkboxes are all still `- [ ]` despite the underlying work being done
and independently verified by this audit. This does not affect the shipped code, but the
plan/PRD files should be updated to `- [x]` before merge so the committed artifacts reflect
reality.

## Deviations From the Plan's Literal Text (confirmed intentional, applied consistently)

1. **`monster.NewModelBuilder(...).Build(), nil` → two-value `Build()` passthrough.** The
   plan's test snippets show `monster.NewModelBuilder(monsterId, f, 100100).Build(), nil`
   inside a `func(uint32) (monster.Model, error)` literal, but `Build()` already returns
   `(Model, error)` (`services/atlas-channel/atlas.com/channel/monster/builder.go:117` —
   `func (b *modelBuilder) Build() (Model, error)`), so the plan's literal text would not
   compile (an extra `, nil` after a two-value call — this is a plan defect, not an
   implementation shortcut). The implementation correctly uses
   `return monster.NewModelBuilder(id, f, 100100).SetMaxHp(n).Build()` everywhere this
   pattern recurs: `character_attack_drain_test.go:180, 214, 261, 283, 309`. All 5 sites
   checked — no stray `.Build(), nil` anywhere in the file. Applied correctly and
   consistently.
2. **`testField()` → `testDrainField()`.** The plan's snippet names the new field-builder
   helper `testField()`, but that name already exists in the package with an incompatible
   signature — `func testField(mapId _map.Id) field.Model` at
   `services/atlas-channel/atlas.com/channel/socket/handler/mystic_door_enter_test.go:30`
   (a zero-arg `testField()` alongside a one-arg `testField(_map.Id)` would be a
   redeclaration error). The implementation renamed it to `testDrainField()`
   (`character_attack_drain_test.go:104-106`) and every one of the 8 call sites in the new
   file uses the renamed helper (`126, 151, 163, 208, 246, 256, 277, 301`).
   `mystic_door_enter_test.go`'s `testField` is untouched. No collision, no stragglers.
3. **`TestDrainTryHeal_EmitErrorSwallowed` — real assertion added.** The plan's snippet
   only comments `// Reaching here without panic is the assertion.` — a vacuous pass. The
   implementation (`character_attack_drain_test.go:272-296`) instead counts `changeHP`
   invocations via a local `changeHPCalls` counter and asserts `changeHPCalls == 1` —
   proving the error branch inside `drainTryHeal` was actually exercised (not skipped by
   an early return) before returning normally. Strictly stronger than the plan's literal
   text; no other vacuous "reaching here" assertions exist elsewhere in the file.

## PRD §10 Acceptance Criteria

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | Four skill ids heal; other skill ids produce no drain heal | PASS | `isDrainSkill` switch (`:83-93`) + gate at `:448`; `TestIsDrainSkill` covers all 4 positive ids plus Aran Combo Drain, id 0, and an adjacent id as negatives. |
| 2 | Heal = `min(monsterMaxHp, floor(totalDamage×X/100), effectiveMaxHp/2)`, X per FR-3 at a spot-checked level | PASS | `drainHealAmount` (`:235-250`); `TestDrainHealAmount` FR-3 rows: Drain L30 x=45 (450), Drain L1 x=16 floor (53), Vampire L20 x=10 (500), Energy Drain L20 x=20 (2469) — matches the PRD FR-3 table values exactly. |
| 3 | Multi-target: per-monster heal, individually capped | PASS | `TestDrainTryHeal_PerMonsterCaps` (`:298-334`) — two monsters, one under both caps (500), one capped by monster max HP (200), in the same test run with independent `getMonster`/`changeHP` closures per iteration. |
| 4 | Killed monster still contributes its heal | PASS (structural, not empirically fixture-tested) | Design D8 / hook ordering: `drainTryHeal` is invoked synchronously inside `onDamageApplied`, which fires immediately after `applyDamage` (`:197-206`, `:444-450`) — before the attack broadcast and before any async kill-driven monster-registry mutation. `mp.GetById` therefore reads the pre-kill snapshot. No dedicated "monster already removed from registry" fixture test exists; the plan's own traceability table cites design rationale + fetch-then-heal ordering (not a unit test) as the evidence for this row, and the implementation matches that intent. |
| 5 | Zero-damage entries, effect-lookup failures, effective-stats failures produce no heal and never abort the attack | PASS | `TestOnDamageApplied_NotCalledForZeroDamageEntry`, `TestDrainTryHeal_MonsterFetchError_SkipsHeal`, `TestDrainTryHeal_ZeroEffectiveStats_SkipsHeal`, `TestDrainTryHeal_EmitErrorSwallowed` — all pass; `drainTryHeal` has a `void` return (cannot propagate an error into `processAttack`), and `onDamageApplied` itself has no return path back to its caller. |
| 6 | Heal defensively clamped to `int16` before `ChangeHP` | PASS | `drainHealAmount:246-248` clamps to `math.MaxInt16`; `TestDrainHealAmount` "int16 clamp on pathological damage" (4,000,000,000 dmg, x=100, max uint32 caps) asserts exactly 32767. |
| 7 | Table-driven unit tests: X-percentage math, monster-max-HP cap, half-effective-max-HP cap, zero damage, floor/truncation | PASS | `TestDrainHealAmount`, 15 rows covering every listed dimension including floor-of-odd-number (`2001/2=1000`) and "tighter of the two caps wins". |
| 8 | Drain TODO removed; adjacent TODOs untouched | PASS | Confirmed via grep: 24→23 TODO lines, `// TODO increase HP from Energy Drain, Vampire, or Drain` gone, `// TODO Combo Drain` present at `:515`, all 22 other TODOs unchanged (diff shows only the one deleted line in that block). |
| 9 | No per-version code branching; feature relies on the skill-ownership gate | PASS | `git diff 01a0a3bb0..59f20e2df -- 'services/atlas-channel/**' \| grep MajorVersion` → no output. The pre-existing ownership-destroy gate at `character_attack_common.go:376-379` is unmodified by this branch and is the sole version-applicability mechanism, as PRD §8.1 specifies. |
| 10 | `go test -race`, `go vet`, `go build` clean in atlas-channel; `redis-key-guard.sh`, `goroutine-guard.sh`, `lint.sh --check` clean from repo root; bake N/A | PASS | See Build & Test Results below. `go.mod` diff-stat is empty (`git diff --stat 01a0a3bb0..59f20e2df -- .../go.mod` → no output), so no `docker buildx bake` was required, consistent with the plan. |

## Build & Test Results

All commands below were run by this audit from the task worktree root
(`.worktrees/task-147-attack-drain-hp-gain/`) on branch `task-147-attack-drain-hp-gain`
(verified via `git branch --show-current` before starting), except where noted.

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel | PASS | PASS | `go build ./...` clean; `go vet ./...` clean; `go test -race -count=1 ./...` — no `FAIL` lines, every package `ok` or `[no test files]`; `socket/handler` specifically `ok atlas-channel/socket/handler 1.740s`. Targeted re-run `go test -race -count=1 -run 'TestIsDrainSkill\|TestDrainHealAmount\|TestOnDamageApplied\|TestDrainTryHeal' -v ./socket/handler/...` — all 22 new subtests pass (`TestIsDrainSkill` ×7, `TestDrainHealAmount` ×15, `TestOnDamageApplied_*` ×3, `TestDrainTryHeal_*` ×5). |

| Guard | Result |
|---|---|
| `tools/redis-key-guard.sh` (repo root) | PASS — exit 0, scans every service incl. atlas-channel, no findings. |
| `tools/goroutine-guard.sh` (repo root) | PASS — exit 0, scans every service + lib, no findings. |
| `git diff --stat -- .../go.mod` | untouched — no output, confirms no `docker buildx bake` required per CLAUDE.md item 4. |
| `tools/lint.sh --check` (repo root, whole monorepo, ~83 Go modules + atlas-ui) | PASS — `lint.sh: OK` (see diagnostic chain below). |

### `tools/lint.sh --check` — diagnostic chain

This audit's **first** independent run of `tools/lint.sh --check` (after sourcing nvm 22)
failed with exactly one target: `lint.sh: FAIL — 1 failing target(s):
lint:services/atlas-mts/atlas.com/mts`. The accompanying golangci-lint output for that run
was saturated with `generated_file_filter` warnings referencing files under
`.worktrees/task-123-megaphones-maple-tv/...` and
`.worktrees/task-176-gm-hide-controller-relinquish/...` — sibling worktrees that **do not
currently exist** (`git worktree list` confirms only `task-122` through `task-175` are
present; `task-123` and `task-176` were pruned at some point after being analysis-cached).
This branch's diff touches only `atlas-channel`; it does not touch `atlas-mts` at all
(`git diff --stat 01a0a3bb0..59f20e2df` shows only the two `atlas-channel` files plus docs).

To determine whether this was a real regression or a stale-cache artifact, this audit:
1. Cleared `~/.cache/golangci-lint` (golangci-lint's own analysis cache — separate from the
   repo-local `.cache/tools/bin` binary cache `lint.sh` manages).
2. Re-ran golangci-lint directly against `services/atlas-mts/atlas.com/mts` with the
   **exact** flags `tools/lint.sh` uses: `fmt --diff -c .golangci.yml ./...` and
   `run -c .golangci.yml --new-from-rev 01a0a3bb003cb464bef9a5c7cbcd04e0f8c82927 ./...`
   (the real merge-base `tools/lint.sh`'s own `resolve_base()` computes — confirmed via
   `git merge-base HEAD origin/main` == `git merge-base HEAD main` ==
   `01a0a3bb003cb464bef9a5c7cbcd04e0f8c82927`).
3. Both invocations returned **exit 0 / "0 issues."** with no stale-worktree warnings.

This isolates the original failure to a poisoned golangci-lint analysis cache holding
paths into worktrees that no longer exist, not a defect in this branch's diff —
`atlas-mts` was never touched by task-147, and the module lints clean in isolation once the
stale cache entries are gone.

This audit then started a full-tree `tools/lint.sh --check` re-run (cold cache, all ~83 Go
modules + atlas-ui) to confirm the fix held tree-wide. That re-run reached 73 of ~166
per-module fmt+lint invocations — every one reporting `0 issues.`, zero
`FAIL`/`ERROR`/`WARNING` lines — before the background shell process exited without
producing a final verdict line (environment/session lifecycle, not a lint failure — no
failing output was ever produced in that partial run). The full-tree run was completed
separately from the worktree root, reporting: **`lint.sh: OK`**, with ESLint surfacing 6
pre-existing warnings (0 errors) in `atlas-ui` files this branch never touched
(`data-table.tsx`, `CreateBanDialog.tsx`, `ApplyPresetDialog.tsx`, `CreateTenantDialog.tsx`,
`AccountsPage.tsx`, `QuestsPage.tsx` — all `react-hooks/incompatible-library` or
`react-hooks/exhaustive-deps`) and nothing in any Go file. Those exact 6 warnings on those
exact files/lines also appear verbatim in this audit's own first full-run capture, which
this audit read directly — so the UI-layer portion of the final verdict is doubly
corroborated. Net: `tools/lint.sh --check` is clean; the one observed failure on this
branch was a pre-existing, unrelated tooling artifact (stale analysis cache from pruned
sibling worktrees) that this audit reproduced, root-caused, and confirmed fixed by a cache
clear.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Optional, non-blocking) Check off `plan.md`'s 25 step checkboxes and `prd.md`'s 10
   acceptance-criteria checkboxes to `- [x]` so the committed artifacts reflect the
   verified state.
2. (Optional, non-blocking) Consider replacing the raw skill-id literals (`4101005`,
   `14101006`) passed as the `skillId` logging parameter in
   `character_attack_drain_test.go:222, 246, 265, 290, 317-318` with the named
   `skill3.*Id` constants for consistency, though this is not a DOM-21 violation since the
   parameter is only ever logged, never classified.
3. (Informational, no action required on this branch) The stale golangci-lint
   analysis-cache issue (references to pruned sibling worktrees
   `task-123-megaphones-maple-tv` and `task-176-gm-hide-controller-relinquish`) is an
   environment/tooling artifact, not something this branch introduced or can fix. Clearing
   `~/.cache/golangci-lint` resolves it locally; if it recurs in CI it would warrant a
   separate look at whether CI runners share an analysis cache across worktree lifecycles.
