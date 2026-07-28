# Plan Audit — task-159-maple-warrior-all-stats

**Plan Path:** docs/tasks/task-159-maple-warrior-all-stats/plan.md
**Audit Date:** 2026-07-17
**Branch:** task-159-maple-warrior-all-stats
**Base Branch:** main (commit range 78e6ccde7..HEAD)

## Plan-Adherence Section

### Executive Summary

All 5 implementation tasks (Tasks 1–5) and the verification task (Task 6) were faithfully executed. Every plan step's code was found byte-for-byte matching the plan's prescribed diffs — the `Bonus.basePercent` dimension, `WithSource`, the base-percent accumulator in `ComputeEffectiveStats`, the one-to-many `BonusesForBuffChange` mapping, both consumer migrations, the `MapBuffStatType` deletion, the `docs/domain.md` rewrite, and both new lifecycle/parity tests are all present with correct content. Independently re-run `go build ./...`, `go vet ./...`, and `go test -race -count=1 ./...` in the module all pass clean (36 checkbox items in plan.md, all unchecked in the file itself but all verifiably implemented in code — see note below). `grep -rn "MapBuffStatType"` returns zero matches. `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh` both exit 0. No stubs, TODOs, or deferred work found. `MapStatupType`, HYPER_BODY semantics, and the equipment-snapshot bonus loop were correctly left untouched, matching the plan's explicit out-of-scope constraints.

**Note on checkbox state:** plan.md's checkboxes (`- [ ]`) are all literally unchecked (0/36 checked via `grep -c '^- \[x\]'`), but this reflects the plan template never being edited post-execution, not incomplete work — every step's file/content is verifiably present in the git history and working tree, cross-checked against the plan's exact code blocks.

### Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `Bonus.basePercent` dimension + `WithSource` + dimension-preserving re-sourcing | DONE | `stat/model.go:46-52` (struct field), `:70-72` (`BasePercent()` getter), `:107-113` (`NewBasePercentBonus`), `:117-120` (`WithSource`), `:122-155` (Marshal/Unmarshal incl. `basePercent`) — all match plan verbatim. `character/processor.go:200-206` (`AddBuffBonuses`) and `:220-226` (`AddPassiveBonuses`) both use `b.WithSource(source)`; `StoreEquipmentBonuses`'s equipment loop (`processor.go:180-182`, seen as `AddEquipmentBonuses`) deliberately left on `stat.NewFullBonus(...)`, per plan step 5's explicit "Do NOT change" instruction. Tests `TestNewBasePercentBonus`, `TestBonusWithSource_PreservesDimensions`, `TestBonusJSONRoundTrip_BasePercent`, `TestBonusUnmarshal_LegacyWithoutBasePercent` (stat/model_test.go:190-260) and `TestProcessor_AddBuffBonuses_PreservesBasePercent` (character/processor_test.go:508-529) present verbatim to plan text. Commit 9abdeb480. |
| 2 | Base-percent application in `ComputeEffectiveStats` | DONE | `character/model.go:287-291` (`basePercentFlat` map init), `:296-303` (accumulator loop: `basePercentFlat[b.StatType()] += baseValues[b.StatType()] * b.BasePercent() / 100`), `:317` (`effective := float64(base+flat+basePercentFlat[statType]) * (1.0 + mult)`) — matches plan exactly, including comment about per-bonus truncation and equipment exclusion. Equipment loop (`:305-315`) untouched — still flat-only, gated by `qualified[assetId]`. All 4 required tests present and correct: `TestModelComputeEffectiveStats_BasePercentTruncation` (character/model_test.go:570-589), `_BasePercentExcludesEquipment` (:592-607), `_BasePercentIndependentTruncation` (:610-627), `_BasePercentWithMultiplier` (:630-645) — assertion values (14/4, 140, 17, 121) match plan's worked examples exactly. Commit bb939fbcd. |
| 3 | `BonusesForBuffChange` one-to-many mapping API | DONE | `stat/model.go:435-477` — full switch statement matches plan verbatim: flat types (WEAPON_ATTACK/PAD, MAGIC_ATTACK/MAD, WEAPON_DEFENSE/PDD, MAGIC_DEFENSE/MDD, ACCURACY/ACC, AVOIDABILITY/AVOID/EVA, SPEED, JUMP) each return one `NewBonus`; HYPER_BODY_HP/MP return `NewMultiplierBonus`; MAPLE_WARRIOR returns 4 `NewBasePercentBonus` (STR/DEX/INT/LUK); default returns `[]Bonus{}`. IDA citation comment (v83 @0x77ec9f, v95 @0x732ba0) preserved. `MapBuffStatType` correctly still present at this commit (deleted only in Task 4). Tests `TestBonusesForBuffChange_Flat` (15 sub-cases), `_HyperBody`, `_MapleWarrior`, `_Unknown` all present at stat/model_test.go:312-424, content matches plan. Commit 253a14209. |
| 4 | Migrate both consumers, delete `MapBuffStatType`, update docs | DONE | `kafka/consumer/buff/consumer.go:50-64` (`handleBuffApplied`) uses `stat.BonusesForBuffChange("", change.Type, change.Amount)` with `len(bs)==0` skip-and-debug-log, matching plan exactly (source stays empty, re-stamped via `WithSource` downstream). `character/initializer.go:182-193` (`fetchBuffBonuses`) uses `stat.BonusesForBuffChange(source, change.Type, change.Amount)` with pre-stamped `"buff:%d"` source, matching plan. `MapBuffStatType` fully deleted from `stat/model.go` — confirmed via `grep -rn "MapBuffStatType" services/atlas-effective-stats/` returning zero hits (exit 1). `MapStatupType` (stat/model.go:479-514) untouched, confirmed present and unchanged in content/position. `docs/domain.md:85` rewritten to the exact plan text describing `BonusesForBuffChange` one-to-many semantics. Commit 22793990c. |
| 5 | Lifecycle and path-parity tests | DONE | `TestProcessor_MapleWarriorLifecycle` (character/processor_test.go:533-583) — apply via `BonusesForBuffChange("", "MAPLE_WARRIOR", 10)` + `AddBuffBonuses`, asserts STR 110/DEX 88/LUK 66/INT 44/MaxHp unchanged at 5000, then `RemoveBuffBonuses` and asserts full revert to base + zero bonuses — matches plan verbatim including the base-stat ordering comment (`NewBase(strength, dexterity, luck, intelligence, ...)`). `TestProcessor_MapleWarrior_PathParity` (character/processor_test.go:587-637) — builds live-apply set via empty-source `BonusesForBuffChange` + `AddBuffBonuses`, initializer set via pre-stamped source + `NewModel(...).WithBonuses(...)`, compares both sets by `(Source, StatType)` key with full `!=` struct comparison including `basePercent` — matches plan exactly. Both tests pass under `go test -race -count=1 ./character/...` (independently re-run, confirmed PASS). Commit 89e875258. |
| 6 | Full verification sweep | DONE | Independently re-verified (not merely trusting the plan's embedded claim): `go build ./...` clean, `go vet ./...` clean, `go test -race -count=1 ./stat/... ./character/...` all PASS (including the 7 new task-159 tests), `tools/redis-key-guard.sh` exit 0, `tools/goroutine-guard.sh` exit 0, `grep -rn "MapBuffStatType" services/atlas-effective-stats/` zero matches. `docker buildx bake atlas-effective-stats` was reported already run clean by the controller — not re-run in this audit per instructions (docker daemon build is expensive and non-deterministic to duplicate); no evidence contradicting that claim was found (Dockerfile unmodified by this branch, no new shared-lib dependency introduced that would require a `COPY` line change). |

**Completion Rate:** 5/5 implementation tasks + verification task DONE (6/6 by task; all step-level checkbox items in plan.md have corresponding landed code, 36/36 by step)
**Skipped without approval:** 0
**Partial implementations:** 0

### Skipped / Deferred Tasks

None found. No `PARTIAL`, `SKIPPED`, or `DEFERRED` tasks.

### Scope Discipline Check

Per Global Constraints and context.md decision log, the following were required to be explicitly OUT of scope and were confirmed left untouched:

- **`MapStatupType`** — present unmodified at `stat/model.go:479-514`, correctly excluded from the `BonusesForBuffChange` migration (statup/passive path is a separate mapping, not part of this task).
- **HYPER_BODY semantics** — `HYPER_BODY_HP`/`HYPER_BODY_MP` still map to `NewMultiplierBonus` exactly as before (`stat/model.go:460-463`); no behavior change, only relocated into the new one-to-many function.
- **Equipment-snapshot bonus loop** — `character/model.go:305-315` (equipment loop inside `ComputeEffectiveStats`) and `processor.go`'s `StoreEquipmentBonuses`/`AddEquipmentBonuses` re-sourcing loop (`stat.NewFullBonus(...)`, not `WithSource`) both left byte-identical to pre-branch behavior, per the plan's explicit "Do NOT change" instruction in Task 1 Step 5.
- **atlas-data / atlas-buffs emitters, client display** — no files outside `services/atlas-effective-stats/` were touched (confirmed via `git diff --stat 78e6ccde7..HEAD` — all 9 changed files are within `services/atlas-effective-stats/`).

### Build & Test Results (independently re-run by this audit)

| Service | Build | Vet | Tests (-race -count=1) | Notes |
|---------|-------|-----|-------------------------|-------|
| atlas-effective-stats | PASS | PASS | PASS | All packages `ok`; `stat` and `character` packages re-run without cache, all task-159 tests (7 new: `TestNewBasePercentBonus`, `TestBonusWithSource_PreservesDimensions`, `TestBonusJSONRoundTrip_BasePercent`, `TestBonusUnmarshal_LegacyWithoutBasePercent`, `TestBonusesForBuffChange_Flat/_HyperBody/_MapleWarrior/_Unknown`, `TestModelComputeEffectiveStats_BasePercent*` (4), `TestProcessor_AddBuffBonuses_PreservesBasePercent`, `TestProcessor_MapleWarriorLifecycle`, `TestProcessor_MapleWarrior_PathParity`) confirmed PASS |

Repo-root guards (independently re-run):

| Check | Result |
|---|---|
| `tools/redis-key-guard.sh` | exit 0 (clean) |
| `tools/goroutine-guard.sh` | exit 0 (clean) |
| `grep -rn "MapBuffStatType" services/atlas-effective-stats/` | zero matches (exit 1) |
| `docker buildx bake atlas-effective-stats` | not re-run this audit; controller reported clean prior run; no code evidence contradicts it (no Dockerfile/go.work change on this branch) |

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the parallel backend-guidelines-reviewer pass, per CLAUDE.md's mandatory code-review-before-PR rule; this plan-adherence section does not substitute for the DOM-* checklist)

### Action Items

None. No fixes required for plan adherence. (If a `backend-guidelines-reviewer` section is appended to this same file, its own action items apply independently.)

---

# Backend Guidelines Audit — atlas-effective-stats (task-159)

- **Service Path:** services/atlas-effective-stats/atlas.com/effective-stats
- **Scope:** Changed Go packages only, git range `78e6ccde7..HEAD`: `stat/`, `character/`, `kafka/consumer/buff/`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-17
- **Build:** PASS (per controller; not re-run in this pass)
- **Tests:** PASS (per controller; not re-run in this pass)
- **Overall:** NEEDS-WORK

## Architecture Note (affects applicability of several DOM-* checks)

`atlas-effective-stats` has **no GORM/SQL database**. Verified: `grep -rn "database.Connect\|gorm" services/atlas-effective-stats/atlas.com/effective-stats/*.go character/*.go stat/*.go` (excluding `_test.go`) returns zero matches. State is held entirely in a Redis-backed `TenantRegistry[uint32, Model]` (`character/registry.go:19-38`). Consequently the following checks are **N/A** for the `character`/`stat` packages, not silently passed: DOM-01 (`builder.go`), DOM-02/DOM-03 (`ToEntity()`/`Make(Entity)`), DOM-10 (test-DB tenant callbacks), DOM-11 (GORM provider laziness), DOM-16 (`administrator.go`), DOM-27 (transient-DB-error → 503 mapping). This is a pre-existing, whole-service architectural fact, not something this diff introduced or could remedy — confirmed unaffected by the diff (`registry.go` was not touched, `git diff --stat` in the Plan-Adherence section above shows no `registry.go` line).

## Domain Checklist Results

### stat (domain package — value objects only, no persistence)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform` function | **FAIL** | `stat/rest.go:59-83` `Transform()` exists and is called from `character/resource.go:37`, but the bonus-level transform it delegates to, `TransformBonus` (`stat/rest.go:48-56`), was **not updated** for the new `basePercent` dimension added in this diff (`stat/model.go:47-52,70-72`). `BonusRestModel` (`stat/rest.go:41-46`) has no `BasePercent` field, and `TransformBonus` (`stat/rest.go:49-56`) copies only `Source`/`StatType`/`Amount`/`Multiplier`. Reachability confirmed: `character/resource.go:25-37` `handleGetEffectiveStats` calls `NewProcessor(...).GetEffectiveStats(...)` → `stat.Transform(characterId, computed, bonuses)`, and any Maple Warrior bonus in that slice (`basePercent=10, amount=0, multiplier=0`) serializes to the REST client as `{"source":"buff:2311003","statType":"strength","amount":0,"multiplier":0}` — indistinguishable from a zero bonus. `Computed` (the aggregate numbers) is correct since `ComputeEffectiveStats` folds `basePercent` in before the transform; only the per-bonus breakdown list silently drops the new dimension. |
| DOM-05 | `TransformSlice` function | N/A | `stat/rest.go` has no `TransformSlice`; the package returns a single `RestModel` per character, never a list — consistent with the one `/characters/{id}/stats` GET handler in `character/resource.go:21`. No inline-loop violation introduced. |
| DOM-18 | JSON:API interface on REST models | PASS | `stat/rest.go:26-36` — `RestModel` implements `GetName()`, `GetID()`, `SetID()`. Unchanged by this diff. |
| DOM-19 | Request models flat structure | N/A | `stat` package defines no `CreateRequest`/`UpdateRequest` (read-only endpoint). |
| DOM-20 | Table-driven tests | PASS | `stat/model_test.go` new tests for this diff use table-driven pattern: `TestBonusesForBuffChange_Flat` (`stat/model_test.go:314-345`, `tests := []struct{...}` + `t.Run`), `TestBonusesForBuffChange_HyperBody` (`:359-...`, same pattern). Confirmed via `grep -n "tests := \[\]struct\|t.Run(" stat/model_test.go` → 4 hits. |
| DOM-21 | No duplication of atlas-constants types | PASS | New field is `basePercent int32` (`stat/model.go:51`), a plain percentage-rate integer, not an id/classification/enum type. `grep -rn "percent\|Percent" libs/atlas-constants` → zero matches; no shared-lib equivalent exists to duplicate. |
| FILE-01/02/06 | File Responsibilities | PASS | `stat/model.go` holds only the immutable `Bonus`/`Type`/`Computed`/`Base` value objects plus their pure conversion helpers (`BonusesForBuffChange`, `MapStatupType`) — no `Processor`, `RestModel`, or request funcs leaked in. `stat/rest.go` holds `RestModel`, `BonusRestModel`, `Transform`, `TransformBonus` correctly (see DOM-04 finding above for its *content* gap, not misplacement). No catch-all `<pkg>.go` file exists in this package. |

### character (domain package — Redis-backed, no GORM)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `character/processor.go:44` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. Unchanged by diff. |
| DOM-13/14/15 | No cross-domain logic / provider calls / direct writes in handlers | PASS | `character/resource.go:30` calls `NewProcessor(d.Logger(), d.Context()).GetEffectiveStats(...)` — processor-only; diff touched no handler code. |
| DOM-20 | Table-driven tests | **WARN (non-blocking)** | The 4 new tests in `character/model_test.go:569-648` (`TestModelComputeEffectiveStats_BasePercentTruncation`, `_BasePercentExcludesEquipment`, `_BasePercentIndependentTruncation`, `_BasePercentWithMultiplier`) and the 3 new tests in `character/processor_test.go:507-632` (`TestProcessor_AddBuffBonuses_PreservesBasePercent`, `TestProcessor_MapleWarriorLifecycle`, `TestProcessor_MapleWarrior_PathParity`) are all individual named test functions — none use `tests := []struct{...}` + `t.Run`. `testing-guide.md:18` states "Prefer table-driven tests" (soft guidance, not MUST), and each new test asserts a numerically distinct formula scenario (truncation, equipment-exclusion, independent-truncation, multiplier-ordering) rather than a parameterizable set of like-shaped cases, which is a defensible reason to keep them separate. Flagged as non-blocking per the guideline's own "prefer" wording, not waived for prevalence — the entire `character` package (pre-existing tests included) uses this same non-table-driven style throughout, so this is a package-wide, not diff-introduced, characteristic. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `character/testmain_test.go:11-13` — `TestMain` calls `producertest.InstallNoop()` before `m.Run()`. `character/processor.go:259,265` are the package's only emit call sites (`producer.ProviderImpl(...)`), reached transitively by `checkAndPublishClampCommands`, called from `RecomputeEquipmentBonuses`/`AddBuffBonuses`/etc. — all covered by the package-wide stub. No `t.Cleanup(producer.ResetInstance)` found (`grep -rn "ResetInstance" character/*.go` → zero matches). |
| FILE-01 | `Processor` in `processor.go` | PASS | `grep -n "^func (p \*ProcessorImpl)" character/*.go` — all 18 methods live in `character/processor.go`; none leaked into `model.go`/`initializer.go`/`registry.go`. |

### kafka/consumer/buff (sub-domain — action-event consumer)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic not in handler | PASS | `kafka/consumer/buff/consumer.go:48-67` `handleBuffApplied` delegates entirely to `character.NewProcessor(l, ctx).AddBuffBonuses(...)`; the only local logic is the pure `stat.BonusesForBuffChange` mapping call (data transform, not business/persistence logic). |
| SUB-02 | Writes via processor, not direct DB | PASS | No `db.Create`/`db.Save` calls in `consumer.go`; all state changes go through `character.Processor`. |
| SUB-04 | No manual JSON parsing | PASS | `grep -n "json.NewDecoder\|json.Unmarshal\|io.ReadAll" kafka/consumer/buff/consumer.go` → zero matches; event body is already typed via `message.AdaptHandler`. |

## Non-Blocking Observations (not guideline violations, noted for completeness)

- `character/processor.go:183` (`AddEquipmentBonuses`) still re-sources bonuses via the dimension-dropping `stat.NewFullBonus(source, b.StatType(), b.Amount(), b.Multiplier())`, unlike the dimension-preserving `b.WithSource(source)` now used by `AddBuffBonuses`/`AddPassiveBonuses` (`processor.go:206,225`). This is **intentional and plan-approved** — `plan.md` Task 1 Step 5 explicitly instructed leaving the equipment path untouched, and equipment bonuses are never constructed with a non-zero `basePercent` today (`ComputeEffectiveStats`'s equipment loop, `character/model.go:309-316`, only reads `.Amount()`), so there is no live data loss. Recorded here only as a latent trap if a future equipment-sourced bonus ever needs `basePercent`.

## Summary

### Blocking (must fix)
- **DOM-04** — `stat/rest.go` `TransformBonus`/`BonusRestModel` (lines 41-56) do not serialize the new `Bonus.basePercent` field, so `GET /characters/{id}/stats` silently reports Maple Warrior (and any future base-percent) bonuses as `amount:0, multiplier:0` in the `bonuses` breakdown, even though the aggregate `Computed` numbers are correct. Add a `BasePercent int32 \`json:"basePercent"\`` field to `BonusRestModel` and populate it in `TransformBonus`.

### Non-Blocking (should fix)
- **DOM-20** — New tests in `character/model_test.go:569-648` and `character/processor_test.go:507-632` do not follow the table-driven `t.Run` pattern preferred by `testing-guide.md:18`; consistent with the rest of the `character` package's existing test style, so not diff-introduced, but noted since DOM-20's pass criteria is explicit.
- Equipment-bonus re-sourcing (`processor.go:183`) still uses the dimension-dropping `NewFullBonus` path rather than `WithSource` — harmless today (equipment never carries `basePercent`), intentional per plan.md, flagged only as a latent-drift risk.

---

## Resolution (controller)

- **DOM-04 (blocking) — FIXED** in commit `27597820d`: added `BasePercent int32 `json:"basePercent"`` to `stat/rest.go` `BonusRestModel` and populated it in `TransformBonus`; new `stat/rest_test.go` covers base-percent / flat / multiplier projections (3 tests). Module `go test -race ./...`, `go vet ./...`, `go build ./...` re-run green. Additive/backward-compatible; no `go.mod` change (docker bake unaffected), no Redis/goroutine-guard impact.
- **DOM-20 (non-blocking)** — new tests use named functions matching the pre-existing `character` package style; not diff-introduced. No change.
- **`character/qualification.go:138` (whole-branch Minor, out-of-scope)** — `sumFlatNonEquipBonuses` ignores `basePercent` for equip-requirement checks. NOT a regression (MW was a multiplier bonus before, ignored identically). Left as a documented follow-up; verify intended semantics against IDA before changing.
