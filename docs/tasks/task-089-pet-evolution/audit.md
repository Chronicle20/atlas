# Plan Audit — task-089-pet-evolution

## Multi-Pet Chooser — Plan Adherence

**Plan Path:** docs/tasks/task-089-pet-evolution/plan-multi-pet-chooser.md
**Audit Date:** 2026-06-12
**Branch:** task-089-pet-evolution
**Base Range:** f81afe37d..HEAD (8 implementation commits)

### Executive Summary

All 9 plan tasks are faithfully implemented with file:line evidence — none stubbed,
skipped, or deferred. The module's `go test ./...`, `go build ./...`, and `go vet ./...`
all pass clean. The one intentional deviation (Task 3's label coverage consolidated into
the existing `TestEnumerateEvolvablePets` rather than a separate
`TestEnumerateEvolvablePetsEmitsLabels`) is present and asserts the required
`"Alpha (Baby Dragon)"` label. Verdict: PASS / READY_TO_MERGE.

### Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | petdata client `Name()` | PASS | model: `petdata/model.go:6` (field), `:27` (getter), `:13` (NewModel name param); rest: `petdata/rest.go:17` (Name field), `:83` (Extract maps `rm.Name`); test: `petdata/rest_test.go:17` asserts `Name()=="Baby Dragon"`, `:20` `IsEvolvable()` |
| 2 | npc pet client `Name()` | PASS | model: `pet/model.go:7` (field), `:29` (getter), `:13` (NewModel name param); rest: `pet/rest.go:20` (Name field), `:87` (Extract passes `rm.Name`); test: `pet/rest_test.go:11` asserts `Name()=="Fluffy"` |
| 3 | enumerate emits `Name (Species)` labels | PASS | `operation_executor.go:814-816` (labelKey default), `:830` labels slice, `:843` `fmt.Sprintf("%s (%s)", pt.Name(), d.Name())`, `:850` stores labels; test: `operation_executor_petevolution_test.go:95` adds `labelContextKey`, `:130` asserts `evolvablePetLabels == "Alpha (Baby Dragon)"` (consolidated into `TestEnumerateEvolvablePets`) |
| 4 | PickFromContextModel + builder + state wiring + const | PASS | `model.go:47` const `PickFromContextType`, `:464` model, `:473-478` getters, `:481-518` builder w/ required-field validation, `:65`/`:153` StateModel/StateBuilder fields, `:134` accessor, `:365` SetPickFromContext, `:181-360` nil-reset in all sibling setters, `:455` Build carries field; test: `pickfromcontext_model_test.go:5,23,35` |
| 5 | pickFromContext REST Transform/Extract + field + switch | PASS | `rest.go:26` RestStateModel field, `:189` RestPickFromContextModel, `:365` TransformPickFromContext, `:377` ExtractPickFromContext, `:353` TransformState case, `:663` ExtractState case (with nil guard); test: `pickfromcontext_rest_test.go:5` round-trip |
| 6 | processPickFromContextState + processState case + empty routing | PASS | `processor.go:546` presenter, `:552-555` empty→`EmptyNextState()` routing, `:512` processState case, `:520` splitCSV helper; test: `pickfromcontext_processor_test.go:18` asserts empty values → `CurrentState=="noEligible"` |
| 7 | Continue PickFromContextType case + pickFromContextValues | PASS | `processor.go:359` Continue case (action==0 cancel ends; else index→value), `:533` pickFromContextValues helper (bounds + empty checks); test: `pickfromcontext_processor_test.go:68` `TestPickFromContextValues` (in/out of bounds, negative, empty) |
| 8 | Garnox npc-1032102.json rewrite | PASS | Valid JSON; states `[start, pick, confirm, checkMeso, doEvolve, success, noEligible, noRock, noMeso, decline]`; start→pick (else outcome), pick is `pickFromContext` with labelsContextKey/emptyNextState=noEligible, confirm→checkMeso→doEvolve; `evolve_pet petId == "{context.selectedPetId}"`; enumerate op has `labelContextKey`; all 9 targets resolve, no unresolved; `data.id=="1032102"` matches filename |
| 9 | Verification gate | PASS | `go test ./...` all `ok`; `go build ./...` clean; `go vet ./...` clean; no go.mod/go.work/Dockerfile in diff; no raw go-redis added to production code (only test harness in `pickfromcontext_processor_test.go`, matching the existing `processor_state_transition_test.go` pattern) |

**Completion Rate:** 9/9 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Intentional Deviation (validated as acceptable)

Task 3's label assertion was consolidated into the existing `TestEnumerateEvolvablePets`
(`operation_executor_petevolution_test.go:130`, asserting
`evolvablePetLabels == "Alpha (Baby Dragon)"`) rather than a separate
`TestEnumerateEvolvablePetsEmitsLabels` test. The label coverage exists and asserts the
exact required string, so the acceptance intent is fully met. Acceptable, not a gap.

### Build & Test Results

| Service | Build | Tests | Vet | Notes |
|---------|-------|-------|-----|-------|
| atlas-npc-conversations | PASS | PASS | PASS | All packages `ok`; conversation, pet, petdata packages green |

Redis-key-guard / docker-bake: no go.mod/go.work/Dockerfile changed and no raw keyed
go-redis added to production code, so the workspace state is consistent with a clean gate.
The only go-redis references in the diff are miniredis test plumbing in
`pickfromcontext_processor_test.go`, identical to the existing sibling test pattern.

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

### Action Items

None. All tasks implemented with evidence; module verification is green.

---

## Multi-Pet Chooser — Backend Guidelines

- **Service Path:** services/atlas-npc-conversations/atlas.com/npc
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-12
- **Scope:** `git diff f81afe37d..HEAD` restricted to `conversation/`, `pet/`, `petdata/`

### Build & Test Results

- `go build ./...` — PASS (clean, no output).
- `go test ./... -count=1` — PASS. `conversation`, `pet`, `petdata` packages all `ok`.
- `go vet ./conversation/... ./pet/... ./petdata/...` — PASS (clean).

### Applicability note

The change adds a new **conversation-engine state type** (`pickFromContext`) plus two
display-name passthroughs. It does **not** introduce a domain or sub-domain package
(no new `model.go`/`resource.go`/GORM entity). Therefore the structural DOM checks that
target persistence domains (DOM-02 `ToEntity`, DOM-03 `Make`, DOM-11 lazy providers,
DOM-15/16 administrator writes, DOM-22 Dockerfile lib blocks, DOM-23/24 Kafka) are
**N/A** to this diff. The checks that *do* apply are: immutable-model + builder,
functional error handling, DOM-21 (reuse/pattern consistency), and multi-tenancy.

### Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| MODEL-IMMUTABLE | `PickFromContextModel` is immutable (private fields, getters only, no setters on the model) | PASS | conversation/model.go: struct has 6 unexported fields; only value-receiver getters `Title()/ValuesContextKey()/LabelsContextKey()/ContextKey()/NextState()/EmptyNextState()`. Mutation lives only on `PickFromContextBuilder`. |
| BUILDER | Fluent builder with validating `Build()` | PASS | conversation/model.go `NewPickFromContextBuilder()`; `Build()` rejects empty `valuesContextKey`/`nextState`/`emptyNextState`, defaults `contextKey` to `"selectedValue"`, returns `(*Model, error)`. |
| STATE-WIRING | `StateBuilder` setter clears sibling pointers + `Build()` validates presence | PASS | conversation/model.go `SetPickFromContext` nils all 11 sibling fields (matching every existing `SetX`); `Build()` `case PickFromContextType` requires non-nil. Mirrors `SetAskStyle`/`SetAskSlideMenu` exactly. |
| DOM-04/05 (Transform) | REST Transform/Extract present and registered in switch | PASS | conversation/rest.go `TransformPickFromContext` + `ExtractPickFromContext`; `TransformState` `case PickFromContextType` and `ExtractState` `case PickFromContextType` both wired; Extract returns error on nil sub-model. |
| DOM-09 (Transform err) | Transform/Extract errors checked, not discarded | PASS | conversation/rest.go ExtractState path checks `err` from `ExtractPickFromContext`; no `_, _ :=` discards in changed code. |
| NO-PANIC | No panics; bounds-safe selection | PASS | conversation/processor.go `pickFromContextValues` returns error on empty list, `selection < 0`, and `selection >= len`. `Continue` `case PickFromContextType` propagates that error (logged + returned) rather than indexing blindly. Mirrors `AskStyleType` bounds guard. |
| EMPTY-ROUTING | Empty values list routes to `emptyNextState`, not a crash/dead menu | PASS | conversation/processor.go `processPickFromContextState`: `if len(values)==0 { return m.EmptyNextState(), nil }`. Verified by `TestPickFromContextEmptyRoutesToEmptyNextState` (asserts `CurrentState()=="noEligible"`). |
| DOM-12 | No `os.Getenv` in handler/presenter | PASS | grep of changed files: zero matches. |
| MANUAL-JSON | No manual JSON decode in changed code | PASS | The one `json.Unmarshal` in operation_executor.go:211 is pre-existing and outside the diff hunks (confirmed not in `git diff`). |
| DOM-21 (reuse/pattern) | Follows the existing `askStyle`/`listSelection` state-type pattern exactly; no reinvented constants | PASS | Presenter conversation/processor.go:579-582 is structurally identical to `processListSelectionState` (`OpenItem(i).BlueText().AddText(...).CloseItem().NewLine()` + `npcSender...SendSimple`). Selection-bounds logic mirrors `AskStyleType`. New const `PickFromContextType StateType = "pickFromContext"` follows the existing `StateType` string-const block; no atlas-constants type duplicated (pet ids reuse numeric `pet.Model.Id()`; no new id/classification type). |
| MULTITENANCY | Per-character conversation context is tenant-isolated; nothing leaks cross-tenant | PASS | conversation/registry.go: all reads/writes go through `atlas.TenantRegistry` keyed by `tenant.MustFromContext(ctx)` + characterId (lines 37, 46, 54, 63, 77). `setContextValue`/`getContextValue` operate on the same per-(tenant,character) context. The chooser adds only new context *keys*, no new storage path. |
| TEST-QUALITY | Tests verify real behavior, incl. empty-routing and bounds | PASS | `TestPickFromContextValues` covers index 0/middle/out-of-bounds/negative/empty. `TestPickFromContextEmptyRoutesToEmptyNextState` drives a real miniredis-backed registry + `ProcessState`. Round-trip REST test asserts Transform→Extract fidelity. Enumerate test asserts the index-aligned `"Alpha (Baby Dragon)"` label. `pet`/`petdata` Extract tests assert `Name()` populated. |

### Latent-Risk Assessment (requested focus areas)

1. **Comma-delimiter invariant (values/labels lists).** The **values** list is purely
   numeric pet ids (`strconv.Itoa(int(pt.Id()))`, operation_executor.go) — never
   player-controlled, so the index space the client selects against can never be
   corrupted by a comma. The **labels** list is the only one carrying player input
   (pet given name). If a name contained a comma, `splitCSV(labels)` would yield
   `len(labels) != len(values)`; the presenter **detects the mismatch and falls back
   to rendering the raw id values** (conversation/processor.go: `if len(ls)==len(values)`).
   Failure mode is therefore "menu shows ids instead of names," never a
   selection/index mismatch or wrong pet evolving. The invariant is documented as a
   contract in design-multi-pet-chooser.md:65. **Assessment: well-contained, not a
   blocking risk.** (See NB-1 for a hardening option.)

2. **Selection-bounds safety.** Client-supplied `selection int32` is bounds-checked in
   both the standalone helper and the `Continue` path before indexing. No path indexes
   an unvalidated `selection`. The `action == 0` cancel path correctly leaves
   `nextStateId` empty, ending the conversation (matches existing engine convention).

3. **Cross-tenant leakage.** None. The conversation context registry is tenant-keyed at
   every entry point; the chooser introduces no new persistence surface.

### Summary

#### Blocking (must fix)
- None.

#### Non-Blocking (should fix / optional)
- **NB-1 (hardening, optional):** The comma invariant relies on the v83 client charset
  excluding commas from pet names. The presenter degrades gracefully (falls back to
  ids) if violated, so this is defense-in-depth only. If a future client/region permits
  commas in names, the labels column would silently drop to ids. Consider documenting
  the invariant in a code comment at the `enumerate_evolvable_pets` label-build site
  (operation_executor.go) in addition to the design doc, or escape-encoding the label
  list. Not required for merge.

### Overall Verdict: **READY TO MERGE**

Build, tests, and vet are clean. The new state type faithfully reproduces the
established `askStyle`/`listSelection` engine pattern, is immutable + builder-validated,
is bounds-safe and panic-free, routes the empty case correctly, and is fully
tenant-isolated. No Critical or Important issues block the PR.
