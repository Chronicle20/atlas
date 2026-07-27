# Backend Guidelines

- **Service Path:** `services/atlas-channel/atlas.com/channel`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-27
- **Scope:** Diff `cdfb71aa3..b92637e0f -- services/atlas-channel` only:
  - `socket/handler/character_attack_common.go` (`sacrificeHpCost`, `sacrificeFirstDamageLine`, gated block in `processAttack`)
  - `socket/handler/character_attack_sacrifice_test.go` (new)
- **Build:** PASS
- **Tests:** all packages PASS (`go test ./... -count=1` in `services/atlas-channel/atlas.com/channel`; new package tests `TestSacrificeHpCost` (11 subtests) and `TestSacrificeFirstDamageLine` (4 subtests) all PASS)
- **Overall:** PASS

## Build & Test Results

```
$ go build ./...          # exit 0, no output
$ go vet ./...             # exit 0, no output
$ go test ./... -count=1   # all packages "ok", including socket/handler 0.511s
$ go test ./socket/handler/... -run TestSacrifice -v -count=1
--- PASS: TestSacrificeHpCost (11/11 subtests)
--- PASS: TestSacrificeFirstDamageLine (4/4 subtests)
$ ./tools/goroutine-guard.sh   # exit 0 (no bare `go` findings for any lib/service, incl. atlas-channel)
```

## Domain / Package Classification

`services/atlas-channel/atlas.com/channel/socket/handler` has no `model.go`, `processor.go`, `rest.go`, `entity.go`, or `builder.go` (verified: `ls model.go processor.go rest.go entity.go builder.go` → all "No such file or directory"). It is a **socket packet-dispatch handler package**, not a DOM or SUB domain package — it is the established pattern in this package for per-feature pure-computation helpers to live alongside the dispatch entry points in feature-named files (e.g. `mpEaterAbsorbAmount`/`mpEaterTryProc` and `drainHealAmount`/`drainTryHeal` already coexist in `character_attack_common.go` before this diff, at lines 373 and 505 respectively pre-diff). The File Responsibilities Checklist (Processor/RestModel/requests placement) does not apply — this diff neither adds nor collapses a Processor/RestModel/requests/entity/builder responsibility; it adds two pure functions and one call-site block to a helper file that already contains structurally identical siblings. No FILE-* finding.

## Mechanical Checks

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No duplication of atlas-constants types | PASS | `character_attack_common.go:712` uses `skill3.Id(ai.SkillId()) == skill3.DragonKnightSacrificeId`, importing `skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"` (`character_attack_common.go:27`). `DragonKnightSacrificeId = Id(1311005)` is defined once in `libs/atlas-constants/skill/constants.go:3002` — no literal `1311005` appears anywhere in the diff. |
| DOM-25 | Client-interpreted byte values are config-resolved, never Go literals | N/A (pass) | `skill3.DragonKnightSacrificeId` is a skill identifier consumed server-side to select business logic, not a client wire/dispatcher byte fed through a client-side lookup switch. No `WithResolvedCode`-class value is introduced. No finding. |
| DOM-26 | Goroutines spawned via `routine.Go` | PASS | `grep -n "^\s*go func\|^\s*go [A-Za-z]" character_attack_common.go` → no matches. `tools/goroutine-guard.sh` from repo root exits 0. |
| DOM-27 | Transient DB errors map to 503 | N/A | `socket/handler` package makes no direct HTTP responses and has no DB access; not applicable to this diff. |
| DOM-28 | No silent degradation in decorators/enrichment | N/A | Not a `model.Decorator`/enrichment path. The Sacrifice block is a fire-and-forget command emit (`cp.ChangeHP`), and its failure path is NOT silent: `character_attack_common.go:718-719` logs `l.WithError(herr).Errorf("Sacrifice: CHANGE_HP emit failed for caster [%d] skill [%d].", ...)` before falling through — matching the sibling precedent at `character_attack_common.go:493-494` (MP Eater `DrainMp` emit failure) and the `drainTryHeal` doc comment "Errors are logged and swallowed — never abort the surrounding attack pipeline" (`character_attack_common.go:503`). |
| — | Integer-overflow safety on `-int16(cost)` narrowing | PASS | `sacrificeHpCost` (`character_attack_common.go:384-395`) clamps `cost` to `currentHp-1` first, then to `math.MaxInt16` (32767) before returning `uint16`. `int16(cost)` for `cost <= 32767` cannot overflow, and `-int16(cost)` stays within `int16` range (`-32767 >= -32768`). Test `"narrowing guard caps at MaxInt16"` (`character_attack_sacrifice_test.go:19`) and `"max uint32 line does not wrap"` (`character_attack_sacrifice_test.go:20`) exercise the boundary; both pass. |
| — | Fire-and-forget error handling consistent with file's MP Eater/Drain precedent | PASS | MP Eater emit failure: `character_attack_common.go:493-494` `l.WithError(err).Errorf("MP Eater: DRAIN_MP emit failed...")`, function continues without abort. Drain heal emit is a bare fire-and-forget call at `character_attack_common.go:642` (delegates error handling internally). Sacrifice: `character_attack_common.go:718-719`, same `l.WithError(...).Errorf(...)` shape, same non-aborting continuation into the following `// TODO` block. Consistent. |
| — | Pure-helper / immutability convention | PASS | `sacrificeHpCost` (`character_attack_common.go:384`) and `sacrificeFirstDamageLine` (`character_attack_common.go:403`) are pure functions — no receiver mutation, no I/O, deterministic on inputs — matching the pre-existing `mpEaterAbsorbAmount` (`character_attack_common.go:369`) and `drainHealAmount` (`character_attack_common.go:417`) idiom in the same file. |
| — | Request/response type usage — `cp.ChangeHP` signature | PASS | `character/processor.go:276` `func (p *ProcessorImpl) ChangeHP(f field.Model, characterId uint32, amount int16) error` matches call site `cp.ChangeHP(s.Field(), s.CharacterId(), -int16(cost))` at `character_attack_common.go:718`; `cp := character.NewProcessor(l, ctx)` already in scope at `character_attack_common.go:542` (pre-existing, reused — not a new construction). |
| — | Table-driven tests | PASS | `character_attack_sacrifice_test.go:12-31` (`TestSacrificeHpCost`, 11-case `[]struct{...}` + `t.Run`) and `character_attack_sacrifice_test.go:41-73` (`TestSacrificeFirstDamageLine`, 4-case table). Both follow `tests := []struct{...}` + `t.Run` shape required by DOM-20/testing-guide.md. |
| — | No `*_testhelpers.go` file | PASS | `find services/atlas-channel -iname "*testhelpers*"` → no results. Test file uses an inline local closure (`entry := func(...)`) inside the test function itself, not a package-level test-only constructor file. |
| — | Kafka producer stubbing (DOM-24) | N/A | Neither `sacrificeHpCost` nor `sacrificeFirstDamageLine` — the only two units under test — call `AndEmit`, `message.Emit`, or `producer.ProviderImpl`; both are pure computations over primitives/`packetmodel.AttackInfo`. No emit path is exercised by the new test file, so no producer stub is required for it. (The package's other pre-existing tests exercising `processAttack` itself are out of this diff's scope.) |
| SEC-01..04 | Security review | N/A | atlas-channel socket handler package is not an auth/token/session-credential service; no JWT/redirect/secret code touched by this diff. |

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.

### Notes
- The `if cost > 0` gate before the log/emit (`character_attack_common.go:715`) correctly no-ops the Debugf/emit when `sacrificeHpCost` returns 0 (miss, non-positive `X`, or `currentHp <= 1`), avoiding a spurious `ChangeHP(amount=0)` command — consistent with the drain/MP-Eater `if heal <= 0 { return }` / `if amount == 0 { return }` guards in the same file.
- `skill3.Id(ai.SkillId())` conversion at the gate (`character_attack_common.go:712`) mirrors the existing `isDrainSkill(skill3.Id(ai.SkillId()))` gate at `character_attack_common.go:641`, keeping the skill-id-to-constants-type conversion idiom uniform across this file's skill-gated blocks.

---

# Plan Adherence — task-148-sacrifice-hp-cost

**Plan Path:** docs/tasks/task-148-sacrifice-hp-cost/plan.md
**Audit Date:** 2026-07-27
**Branch:** task-148-sacrifice-hp-cost
**Base Branch:** main (merge-base cdfb71aa3)
**Head:** b92637e0f

## Executive Summary

All 4 plan tasks (2 TDD helper tasks, 1 orchestration task, 1 verification sweep) were faithfully implemented exactly as specified — function signatures, formulas, clamp semantics, log messages, and even inline comments match the plan's prescribed code verbatim. The change is confined entirely to `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` and its new test file; no `go.mod`/`go.work`/lib/contract changes. `go build`, `go vet`, and `go test -race ./...` are all clean in the atlas-channel module, and the repo-root `redis-key-guard.sh`, `goroutine-guard.sh`, and `lint.sh --check` guards all pass. The only gap found is process hygiene, not implementation: none of the plan.md checkboxes (`- [ ]`) were ever checked off despite the work being done and committed.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `sacrificeHpCost` pure helper (TDD) | DONE | `character_attack_common.go:376-395` implements the exact signature `func sacrificeHpCost(firstLine uint32, x int16, currentHp uint16) uint16` from the plan, with identical miss/x≤0/hp≤1 short-circuit, `uint64` truncating math (`cost := uint64(firstLine) * uint64(x) / 100`), `≥ currentHp` clamp, and `math.MaxInt16` cap. Commit `38b03924d`. Test file `character_attack_sacrifice_test.go:10-37` has all 11 plan-specified subtests, all PASS (verified by direct re-run: `go test ./socket/handler/ -run TestSacrificeHpCost -v`). |
| 2 | `sacrificeFirstDamageLine` extraction helper (TDD) | DONE | `character_attack_common.go:397-407` implements `func sacrificeFirstDamageLine(ai packetmodel.AttackInfo) uint32` verbatim, returning `di[0].Damages()[0]` guarded by length checks. Commit `95c8941db`. Test file `character_attack_sacrifice_test.go:39-83` has all 4 plan-specified subtests including the FR-2 multi-target pin ("multi-target attack ignores second entry"), all PASS. |
| 3 | Orchestration block in `processAttack` + TODO removal | DONE | `character_attack_common.go:706-720`: gated `if skill3.Id(ai.SkillId()) == skill3.DragonKnightSacrificeId` block, placed immediately after the `comboOrbTryUpdate` `if`, computes `firstLine`/`cost`, `Debugf`s only when `cost > 0`, calls `cp.ChangeHP(s.Field(), s.CharacterId(), -int16(cost))` with a fresh `herr` swallowed via `Errorf` — matches the plan's Step 1 code block field-for-field, including the exact comment wording. `// TODO decrease HP from DragonKnight Sacrifice` (formerly line 669) is gone (`grep -n` exit 1, no match); all 17 other TODOs in the block (cooldown, dark sight/wind walk, attack effect, Chief Bandit mesos, Bandit Steal, Fire/Ice Demon weaken, Homing Beacon/Bullseye, Flame Thrower, Snow Charge, Hamstring, Slow, Blind, Paladin/White Knight charges, Combo Drain, Mortal Blow, Three Snails, Heavens Hammer, ComboTempest, BodyPressure) are untouched at lines 694-695, 724-741. Commit `b92637e0f`. |
| 4 | Full verification sweep | DONE | `go build ./...`, `go vet ./...`, `go test -race ./... -count=1` all clean in `services/atlas-channel/atlas.com/channel` (re-run by this audit). `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` all clean from repo root (re-run by this audit). `git diff main --name-only -- '**/go.mod' 'go.work*'` empty (re-run, confirms no `docker buildx bake` requirement). TODO-removal grep confirmed empty match / `git diff main -- services/atlas-channel \| grep -c "^+.*TODO"` = 0 (no new TODOs introduced). |

**Completion Rate:** 4/4 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All four plan tasks were implemented as specified. The plan's own "Manual Validation (post-merge / deploy, human-driven)" section (per-version data check, in-game validation) is explicitly out of scope for code review and was already substantively addressed pre-implementation: PRD §8/§9 and context.md record a 2026-07-27 live-`atlas-main` data check confirming nonzero `x` for skill 1311005 on 9 of 10 supported tenants (v48/61/72/79/83/84/87/92/jms185), with v95 explicitly flagged as an environment WZ-data-ingestion gap unrelated to this task's code (any skill attack on v95 already fails upstream at `GetEffect` regardless of this change). This is a documented, called-out limitation, not a silently dropped requirement.

## Acceptance-Criteria Cross-Check (PRD §10 / task brief)

- **Formula `firstDamageLine × X / 100`, truncating `uint64` math:** CONFIRMED — `character_attack_common.go:388` (`cost := uint64(firstLine) * uint64(x) / 100`).
- **Survival clamp (≥1 HP, no emit at Hp≤1):** CONFIRMED — `sacrificeHpCost` returns 0 when `currentHp <= 1` (`:381`) and clamps `cost >= uint64(currentHp)` down to `currentHp - 1` (`:389-391`); orchestration only logs/calls `ChangeHP` when `cost > 0` (`:708`), so `Hp() <= 1` never emits a command.
- **`// TODO decrease HP from DragonKnight Sacrifice` removed, all other TODOs intact:** CONFIRMED — grep for the removed string returns no match; the surrounding block still contains all 17 pre-existing TODO lines verbatim; diff shows 0 new `+.*TODO` lines added anywhere under `services/atlas-channel`.
- **Confined to atlas-channel, no go.mod/lib/contract changes:** CONFIRMED — `git diff main --stat` (excluding task docs) touches only `character_attack_common.go` and the new `character_attack_sacrifice_test.go` under `services/atlas-channel/atlas.com/channel/socket/handler/`. `git diff main --name-only -- '**/go.mod' 'go.work*'` is empty.
- **FR-9 (generic cast-cost block untouched):** CONFIRMED — the `se.HPConsume()`/`se.MPConsume()` block remains unmodified at `character_attack_common.go:575-579`, structurally separate from the new Sacrifice block at `:706-720`, so Sacrifice continues to pay both the flat cast cost and the new damage-proportional cost.
- **Identifier correctness:** `skill3.DragonKnightSacrificeId = Id(1311005)` exists at `libs/atlas-constants/skill/constants.go:3002` (no literal `1311005` anywhere in the diff); `cp.ChangeHP(f field.Model, characterId uint32, amount int16) error` at `character/processor.go:276`; `c.Hp() uint16` at `character/model.go:132`; `se.X() int16` at `data/skill/effect/model.go:154` — all match the plan's cited signatures exactly.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel | PASS | PASS | `go build ./...` clean, `go vet ./...` clean, `go test -race ./... -count=1` clean across all packages including `socket/handler` (which contains the new tests). Targeted `go test ./socket/handler/ -run 'TestSacrifice' -v` shows all 11 `TestSacrificeHpCost` subtests and all 4 `TestSacrificeFirstDamageLine` subtests PASS. |

### Repo-root guards

| Guard | Result |
|-------|--------|
| `tools/redis-key-guard.sh` | PASS (clean, exit 0) |
| `tools/goroutine-guard.sh` | PASS (clean, exit 0) |
| `tools/lint.sh --check` | PASS (clean, exit 0) |
| `git diff main --name-only -- '**/go.mod' 'go.work*'` | Empty — no `go.mod`/`go.work` touched; `docker buildx bake atlas-channel` correctly not required per CLAUDE.md item 4. |

atlas-ui / other services: not applicable — no files outside `services/atlas-channel` and `docs/tasks/task-148-sacrifice-hp-cost` were changed on this branch.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Cosmetic, non-blocking) Check off the `- [ ]` boxes in `plan.md` (20 step-level checkboxes across Tasks 1-4) to reflect actual completion — the plan document currently under-represents finished work. Does not affect code correctness or the merge decision.
