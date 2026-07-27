# Plan Audit — task-116-processor-gen3-unification

**Plan Path:** docs/tasks/task-116-processor-gen3-unification/plan.md
**Audit Date:** 2026-07-12
**Branch:** task-116-processor-gen3-unification
**Base / Merge-base:** 38d4d0ba22 (main)

## Plan-Adherence Review

### Executive Summary

The plan was faithfully executed. All 37 tasks are complete. Both Task-36 primary acceptance greps return **no output**, the universal conformance-assertion sweep reports **0 missing** across all 374 non-mock `processor.go` packages, the diff is confined to `services/` + the task folder, and the 5-service build/vet/`test -race` sample is fully green. The known deviations flagged in the brief are all present, correctly implemented, and documented in `inventory.md`. **No Critical or Important gaps found.**

### Acceptance-Grep Results (Task 36)

| Gate | Result |
|---|---|
| `grep NewProcessor(...) *ProcessorImpl` (non-test, non-mock) | **empty** (PASS) |
| `grep "type Processor struct"` (non-mock) | **empty** (PASS) |
| Conformance-assertion sweep over all 374 non-mock `processor.go` pkgs | **0 MISSING** (PASS) |
| Diff-scope `git diff --stat <base>..HEAD -- ':!services' ':!docs/tasks/...'` | **empty** (PASS) |

### Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | FR-8 inventory baseline | DONE | inventory.md (131 rows, plan-time snapshot matched, zero drift) |
| 2–7 | Phase A R1 CP-2 fixes (doors, summons, saga-orch, pets, npc-conv, mounts) | DONE | inventory rows 37–47 `done`; Phase-A rescan −11 files |
| 8–23 | Phase B R2/R1/R4 (chairs…channel×5) | DONE | inventory rows 48–120 `done`; Phase-B rescan CP-2=0, Gen2.5=0 |
| 24 | R6 non-processor renames ×3 | DONE | `command/types.go`, `gachapons/test/fixtures.go`, `npc/test/fixtures.go` present; old `processor.go` gone |
| 25–35 | Phase C R3/R2 (account…atlas-data) | DONE | inventory rows 124–167 `done`; Phase-C rescan all buckets empty |
| 31 | atlas-monster-death characterization + conversion | DONE | test commit cb71f830bc precedes conversion 32c4345333 |
| 36 | Acceptance sweep | DONE | greps empty; assertion sweep 0 missing; 244 pre-existing pkgs got assertions |
| 37 | Code review | IN PROGRESS | this audit |

**Completion Rate:** 37/37 (100%). Skipped without approval: 0. Partial: 0.

### Recipe Spot-Checks (file:line)

- **R1 CP-2** — `atlas-pets/pet/processor.go` `NewProcessor(...) Processor`; option method retyped. Portal deviation confirmed: `atlas-channel/channel/portal/processor.go` interface declares `Enter`/`Warp`/`WarpToPosition` only; `WarpToPortal` stays on impl (processor.go:59) but omitted from interface (no caller) — matches R1 minimal-addition rule.
- **R2 Gen2→Gen3** — `atlas-channel/channel/macro/processor.go` (worked example) conforms; 16 atlas-consumables + 10 atlas-inventory packages converted.
- **R3 Gen1→Gen3** — `atlas-rates/buffs/processor.go:9,13,18,25,28` full Gen3 shape. Capture semantics preserved: no `tenant.MustFromContext`/`producer.ProviderImpl` acquisition moved into any `NewProcessor` constructor (verified atlas-account/ban, atlas-portals/portal, atlas-rates/session — constructors are pure struct literals).
- **R4 rename-only clients** — both `atlas-configurations/data` and `atlas-character-factory/data` keep `NewProcessor(l) Processor` (no ctx), ctx-per-call methods, `var _ Processor = (*ProcessorImpl)(nil)`, mock renamed to `ProcessorMock` with assertion. Documented under "Sanctioned shape deviations".
- **R6** — three files renamed content-untouched; no longer named `processor.go`.

### Known Deviations — Validated (not gaps)

1. portal (atlas-channel) 3-of-4 interface methods — CONFIRMED correct.
2. compartment `WithTransaction`/`WithAssetProcessor` return `*ProcessorImpl` in both interface (processor.go:34-35) and impl (116,129) for unexported chaining — CONFIRMED, documented.
3. atlas-data `calcDropPos` stays a package function (map/processor.go:360) — CONFIRMED, documented.
4. atlas-data adapter closures / workers plumbing (Tasks 32–35) — behavior-preserving, documented.

### Characterization Tests (Task 31 / R7)

- `atlas-monster-death/monster/monster/characterization_test.go` and `.../monster/drop/characterization_test.go` exist and pass. Target pure functions (`calculateExperienceStandardDeviationThreshold`, `isWhiteExperienceGain`, `getSuccessfulDrops`, `getRandomStat`, `isEquipment`) remain **package functions** (no receiver) per R3 Step 3, so tests were untouched by conversion. Test commit (cb71f830bc) lands **before** the conversion commit (32c4345333).

### Build & Test Results (representative sample)

| Service (module) | Build | Vet | test -race |
|---|---|---|---|
| atlas-monster-death | PASS | PASS | PASS (monster, monster/drop) |
| atlas-consumables | PASS | PASS | PASS |
| atlas-inventory | PASS | PASS | PASS (incl. compartment) |
| atlas-channel | PASS | PASS | PASS (51 packages) |
| atlas-data | PASS | PASS | PASS (31 packages) |

Docker bake not re-run (per brief; already run at branch-end).

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the parallel backend-guidelines review)

### Action Items

None. No behavior-changing gaps, no skipped tasks, no missing assertions or mocks.

## Backend-Guidelines Review

**Reviewer:** backend-guidelines-reviewer (adversarial, DOM-*/SUB-*/SEC-*)
**Date:** 2026-07-12
**Scope:** whole-branch Gen3 processor unification (569 changed `.go` files, ~130 converted packages + 244 assertion-only additions across ~47 services). Behavior-preserving refactor: declaration/type changes + conformance assertions + hand-written mocks only.

### Verdict: PASS — no Critical or Important findings

This is a strictly behavior-preserving, declaration-level refactor and it holds up to adversarial scrutiny. Every capture-semantics, mock-shape, and sanctioned-deviation claim in `inventory.md`/`design.md` was independently re-verified against source.

### Objective gate — build/vet/test (representative sample)

| Module | build | vet | test |
|---|---|---|---|
| atlas-monster-death | PASS | PASS | PASS (incl. `monster`/`monster/drop` characterization tests, `-race`) |
| atlas-consumables | PASS | PASS | PASS |
| atlas-inventory | PASS | PASS | PASS |
| atlas-data | PASS | PASS | PASS |
| atlas-channel | PASS | PASS | PASS |

All five exited 0; no FAIL/panic/compile errors. Docker bake not re-run (branch-end sweep already done; no `go.mod` touched — confirmed).

### Invariants verified

- **Acceptance greps both empty:** `NewProcessor(` returning `*ProcessorImpl` → 0 hits; `type Processor struct` outside `mock/` → 0 hits.
- **Diff scope:** `git diff --stat 38d4d0ba22..HEAD -- ':!services' ':!docs/tasks/...'` → empty. No changes outside `services/` + task folder.
- **No `go.mod`/`go.sum`/`go.work` changes** in the diff (matches the "no dependency changes" constraint).
- **No new stubs/TODOs:** the only added `// TODO` lines (`determine type of drop`, `parties`, `account for healing`) each appear as a matched removal+addition — relocated verbatim during the Gen1→Gen3 collapse, not introduced. Pre-existing TODOs (pets `Create` cashId, rates orphaned doc comment) left untouched per Constraint 1; already logged as deferred findings.
- **Impl conformance assertion** present in every non-mock `processor.go` except the 2 `atlas-messages` packages (`message`, `saga`), whose assertions correctly live in `processor_test.go:10`/`:29` — the one documented location exception (design §2.3).
- **Mock conformance assertion** (`var _ <pkg>.Processor = (*ProcessorMock)(nil)`) present in all 122 new + 4 modified `mock/processor.go` files — zero missing (a first grep pass mis-reported misses due to a BRE-escaping bug in the pattern; a fixed-string re-check confirmed 0 genuinely missing).

### Behavior-preservation spot-checks (DOM capture-semantics)

- **Capture-point sweep:** scanned every added `NewProcessor` body across the diff for `tenant.MustFromContext` / `producer.ProviderImpl`. Zero constructors acquire a tenant or producer at construction time — all such acquisitions remain in method bodies (e.g. `data/data/processor.go` `StartWorker`, `reactors/reactor/processor.go`, `messengers/invite`, `monster-death/character`, `portals/character`). Failure/emission timing preserved.
- **atlas-monster-death `monster/processor.go` (R3):** `CreateDrops`/`DistributeExperience` bodies byte-identical apart from `l→p.l`, `ctx→p.ctx`, currying collapse, and a `p→pt` local rename forced by the new receiver name `p` (avoids shadowing). Collaborator acquisition (`drop.NewProcessor(p.l,p.ctx)`, `party.NewProcessor(...)`) stays per-call, matching the original per-call curried acquisition.
- **atlas-consumables `consumable/processor.go` (R2) — cdp closures:** package-level `Consume*` closures previously read the unexported field `p.cdp`; because `NewProcessor` now returns the interface type `Processor` (no exported `cdp` field reachable), each closure constructs a local `cdp := consumable3.NewProcessor(l, ctx)` with the same enclosing `l, ctx`. Verified behavior-preserving: `data/consumable.NewProcessor` is a pure struct literal (no side effects), so the local instance is equivalent to the field it replaced. The redundant `p.cdp` still constructed inside `NewProcessor` is unused but harmless (same purity).
- **atlas-data `npc` + `data/data` orchestrator + `workers` (R3):** worker call sites use bare method values `npc.NewProcessor(l, ctx, db).RegisterNpc` (`workers/npc.go:39`, `data/processor.go:192`); `RegisterFunc` flattened from curried `func(l)func(ctx)func(path)error` → `func(path string) error` (matched removal+addition). Leaf constructors confirmed pure struct literals, so constructing per-walk vs per-file changes only construction count, not resolved `l/ctx/db` — behavior-identical (matches inventory.md task-35 note).

### Mock shape (spot-checked)

Hand-written func-field mocks with per-method nil-guard returning zero values, plus the `var _` assertion — e.g. `monster-death/monster/mock/processor.go` (2 methods), `data/npc/mock/processor.go` (`RegisterNpc func(path string) error`). Signatures mirror the interface exactly.

### Sanctioned deviations — validated, not false-flagged

- **R4 clients** `configurations/data` + `character-factory/data`: `NewProcessor(l logrus.FieldLogger) Processor` (no `ctx`), methods take `ctx` per-call (`GetSkillsByIds(ctx, ...)`, `GetItemById(ctx, ...)`). Matches design §4.2.
- **`inventory/compartment`**: `WithTransaction`/`WithAssetProcessor` return `*ProcessorImpl` in both interface (`:34-35`) and impl (`:116,:129`) for unexported-member chaining (`p.assetProcessor.WithTransaction(tx)...`). Go permits a concrete return type in an interface method; signature-rename-only. Validated.
- **`atlas-channel/portal`** (R1): interface exposes 3 of 4 exported methods (`Enter`/`Warp`/`WarpToPosition`); `WarpToPortal` omitted. Confirmed genuinely uncalled — the only `saga.WarpToPortal` references are a distinct saga-action constant, not the method.

### DOM-21 (shared-constants)

No reinvented `libs/atlas-constants` types introduced. The only new `type` decls outside the Processor/Mock triple are the two `RegisterFunc` function-type simplifications (plumbing, replacements of pre-existing curried decls) — not domain-value types.

### Findings

None at Critical or Important severity. No Minor findings warranting a fix — the pre-existing bugs surfaced during conversion (compartment stale `Provider` interface deletion, rates orphaned doc comment, data `RegisterAllData` swallowed errors, pets cashId TODO) are correctly logged as deferred (`inventory.md` "Deferred findings") and are out of scope for a behavior-preserving refactor per Global Constraint 1.
