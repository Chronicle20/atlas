# Design — Multi-Pet Evolution Chooser (task-089 extension)

> Extends the Pet Evolution feature (see `design.md`, `plan.md`). The base feature evolves a single eligible summoned pet via the Garnox NPC; when a character has **more than one** eligible pet, the conversation could not present a chooser because the npc-conversation engine had no way to (a) source menu options from a runtime-computed list or (b) branch on it. This design adds a small, reusable engine capability — a context-sourced selection state — that closes that gap and wires it into the Garnox conversation.

## Background: why this was blocked, and what actually exists

Verified against `services/atlas-npc-conversations/atlas.com/npc` on 2026-06-12:

- `genericAction` outcome conditions evaluate through `validation.ValidateCharacterState`, whose `Type` only accepts **character-stat** kinds (`item, meso, jobId, mapId, fame, buddyCapacity, questStatus`). There is no condition type whose operand is a conversation **context** variable, so a flow cannot branch on a count written by a local op (e.g. `evolvableCount`). Only `Conditions()[0]` per outcome is evaluated.
- `listSelection` choices are **static** (`ListSelectionModel{title, choices []ChoiceModel}`); the handler interpolates `{context.x}` into each choice's *label text* but the *set* of choices is fixed at author time.

However, the engine **already** has runtime-sourced selection in the `askStyle` state (`processor.go` `Continue`, `case AskStyleType`): when `StylesContextKey()` is set, it reads a comma-joined list from `ctx.Context()[stylesContextKey]`, validates the selection index against that runtime list, stores `values[selection]` into `ContextKey()`, and advances to `NextState()`. The selection mechanism we need exists; it is merely bound to `askStyle`'s avatar/style-preview presentation, which is semantically wrong for pets.

**Approach chosen (A):** add a new state type `pickFromContext` that reuses `askStyle`'s context-sourcing semantics with `listSelection`'s plain numbered-text presentation, plus an `emptyNextState` so the zero-eligible case routes to a message instead of an empty menu. No new condition type is introduced. (Rejected: a general `context` condition primitive — more surface, not needed once the menu owns empty-routing; and reusing `askStyle` directly — wrong presentation packet.)

## Goal & UX

When a player talks to Garnox (`npcId 1032102`):
- **0 eligible pets** → "come back when your pet is ready" dialogue.
- **≥1 eligible pet** → a numbered menu of `Name (Species)` rows (e.g. `Fluffy (Baby Dragon)`); the player picks one; that pet evolves. The menu is **always shown for ≥1**, including exactly one (a one-item menu) — no auto-select special case (decision: uniform single path).
- Gates: must own the Rock of Time (`5380000 ≥ 1`) and enough mesos.

## Conversation flow (`deploy/seed/gms/12_1/npc-conversations/npc/npc-1032102.json`, rewritten)

1. **`start`** (`genericAction`): operation `local:enumerate_evolvable_pets`; outcomes (first-match): Rock condition `item 5380000 < 1` → `noRock`; else (empty conditions) → `pick`. Rock gate stays before the menu.
2. **`pick`** (`pickFromContext`, NEW): `valuesContextKey=evolvablePets`, `labelsContextKey=evolvablePetLabels`, `contextKey=selectedPetId`, `nextState=confirm`, `emptyNextState=noEligible`, `title="Which pet shall I evolve?"`. Empty list → `noEligible`; otherwise present labels, store chosen id into `selectedPetId`, → `confirm`.
3. **`confirm`** (`genericAction` or `dialogue` sendYesNo): meso gate `meso < <cost>` → `noMeso`; Yes/else → `doEvolve`.
4. **`doEvolve`** (`genericAction`): operations in order `destroy_item {itemId:5380000, quantity:1}`, `award_mesos {amount:-<cost>}`, `evolve_pet {petId:"{context.selectedPetId}"}` → `success`. (Three remote ops → batch auto-tags `PetEvolution` saga; compensation refunds on failure — unchanged from base feature.)
5. **`success` / `noEligible` / `noRock` / `noMeso`** (`dialogue`).

**The critical change:** `evolve_pet` reads `{context.selectedPetId}` (a single id set by the `pick` menu), not `{context.evolvablePets}` (which for >1 was a comma-joined string that failed to parse and safe-failed the batch). This is what makes the multi-pet path actually evolve.

Meso cost: reuse the base feature's placeholder (`600000`) unless changed in the plan.

## New engine capability: `pickFromContext` state type

### Model — `conversation/model.go`
- Add state-type constant `PickFromContextType` (string value `"pickFromContext"`).
- `PickFromContextModel` (immutable, getters + `PickFromContextBuilder`, mirroring `AskStyleModel`):
  - `title string` — header; supports `{context.x}` placeholders.
  - `valuesContextKey string` — context key holding the comma-joined option **values** (the pet ids); the value stored on selection.
  - `labelsContextKey string` — context key holding the parallel comma-joined display **labels**; optional.
  - `contextKey string` — where the selected value is stored.
  - `nextState string` — state after a selection.
  - `emptyNextState string` — state if the values list is absent/empty.
- `StateModel` gains a `PickFromContext() *PickFromContextModel` accessor and `StateBuilder.SetPickFromContext(...)`, mirroring `AskStyle()` / `SetAskStyle(...)`.

### REST (de)serialization — `conversation/rest.go`
- `PickFromContextRestModel` with json tags `title`, `valuesContextKey`, `labelsContextKey`, `contextKey`, `nextState`, `emptyNextState`.
- Wire into `StateRestModel` Transform/Extract the same way `askStyle`/`listSelection` are, so a state with `"type":"pickFromContext"` round-trips from seed JSON to the domain model. Confirm the exact place state types are switched during Transform/Extract and add the case.

### Presentation — `conversation/processor.go`
- `processState` (`ProcessState` switch): add `case PickFromContextType: return p.processPickFromContextState(ctx, state)`.
- `processPickFromContextState(ctx, state) (string, error)` (mirror `processListSelectionState`):
  1. Read `values := ctx.Context()[m.ValuesContextKey()]`. If missing OR empty (after trimming) → `return m.EmptyNextState(), nil` (the engine chains to render that state).
  2. Split values by `,`. Read `labels` from `m.LabelsContextKey()` similarly; if absent or `len(labels) != len(values)`, use `values` as labels (defensive fallback).
  3. Build a numbered menu with `message.NewBuilder()` + `OpenItem(i).BlueText().AddText(processedLabel).CloseItem().NewLine()` (interpolate `{context.x}` in title + labels via `ReplaceContextPlaceholders`), `SendSimple`, then `return state.Id(), nil` (wait for selection).

### Selection — `conversation/processor.go` `Continue`
- Add `case PickFromContextType` (mirror `AskStyleType`):
  - `action == 0` → Exit/Cancel (no store, end/branch per existing convention).
  - else: read+split `values` from `m.ValuesContextKey()`; validate `selection` in `[0, len(values))` (out-of-bounds → error, mirror askStyle); set `choiceContext = {m.ContextKey(): values[selection]}`; `nextStateId = m.NextState()`.

### Invariant
`evolvablePets` (values) and `evolvablePetLabels` (labels) are produced in the same enumerate loop and are therefore index-aligned. Both use comma delimiters; v83 pet names and atlas-data species names contain no commas (restricted client charset), so comma-splitting is safe. Documented here as a contract for `enumerate`.

## `enumerate_evolvable_pets` + client extensions

### `conversation/operation_executor.go`
- Extend the local op to also emit a parallel labels list. New optional param `labelContextKey` (default `evolvablePetLabels`). In the eligibility loop, for each kept pet build `label := fmt.Sprintf("%s (%s)", pt.Name(), species.Name())` where `species` is the already-fetched `petdataP.GetById(pt.TemplateId())` result. Append `pt.Id()` to the ids slice and `label` to the labels slice in lockstep. Write ids to the existing output key (`evolvablePets`) and labels to `labelContextKey`. Keep writing `evolvableCount` (still harmless; title can interpolate it).

### npc `pet` client — `pet/model.go`, `pet/rest.go`
- Add `name string` to the domain `Model` + `Name() string` getter; set it in `Extract` from `RestModel.Name` (the RestModel already carries `name`). Update the `NewModel` constructor signature/callers accordingly (Extract is the only production caller).

### `petdata` client — `petdata/model.go`, `petdata/rest.go`
- Add `name string` + `Name() string`; the atlas-data pet resource already serves `name` (`RestModel.Name`). Populate in `Extract`. This is the species/template display name.

## Files changed (summary)
- `conversation/model.go` — `PickFromContextType` const, `PickFromContextModel` + builder, `StateModel.PickFromContext()` + `StateBuilder.SetPickFromContext`.
- `conversation/rest.go` — `PickFromContextRestModel` + Transform/Extract wiring.
- `conversation/processor.go` — `processPickFromContextState`, `ProcessState` case, `Continue` case.
- `conversation/operation_executor.go` — labels emission in `enumerate_evolvable_pets`.
- `pet/model.go`, `pet/rest.go` — `Name()`.
- `petdata/model.go`, `petdata/rest.go` — `Name()`.
- `deploy/seed/gms/12_1/npc-conversations/npc/npc-1032102.json` — rewrite to the flow above.

No new Go module, no new `libs/`, no `go.mod`/`Dockerfile`/`go.work` changes. Single service touched: `atlas-npc-conversations`.

## Testing (TDD per unit)
- **Model/REST**: round-trip a `pickFromContext` state from JSON → domain model → back; assert all six fields.
- **Presentation** (`processPickFromContextState`): (a) non-empty values → a `SendSimple` menu is sent with the labels and the state waits (returns `state.Id()`); (b) empty/absent values → returns `emptyNextState` (no menu sent); (c) label/value length mismatch → values used as labels.
- **Selection** (`Continue` `PickFromContextType`): a valid index stores `values[selection]` into `contextKey` and advances to `nextState`; out-of-bounds → error; `action==0` cancels without storing.
- **`enumerate`**: stub `petP.GetPets` (two eligible: ids 1,2 sharing template 5000029) + `petdataP.GetById` (evolvable, reqPetLevel 15, species name "Baby Dragon") + pet names; assert `evolvablePets="1,2"` and `evolvablePetLabels="<name1> (Baby Dragon),<name2> (Baby Dragon)"`, index-aligned; ineligible pets excluded from both lists.
- **Clients**: `pet.Extract` populates `Name()`; `petdata.Extract` populates `Name()`.
- **Seed**: `npc-1032102.json` is valid JSON, `data.id=="1032102"`, `startState` present, every `nextState`/`emptyNextState`/outcome target resolves to a defined state, only real state/condition/operation types used.

## Verification gate (CLAUDE.md)
- `services/atlas-npc-conversations/atlas.com/npc`: `go test -race ./... && go vet ./... && go build ./...` clean.
- Repo root: `tools/redis-key-guard.sh` clean (run with the workspace active — `GOWORK=off` yields a known worktree false-positive; no redis code is added regardless).
- Worktree root: `docker buildx bake atlas-npc-conversations` (no `go.mod` touched, but re-bake to keep "verified" honest since service source changed).
- Runtime acceptance: a character with two eligible pets sees a `Name (Species)` menu and evolves the chosen one; a character with one sees a one-item menu; a character with none sees the come-back message.

## Out of scope
- A general context-comparison condition type (Approach B) — not built; `emptyNextState` covers this feature's branching need.
- Auto-select for the single-pet case — explicitly not done (always show the menu).
- Pagination / >N-item menus — pet count is small (few summon slots); the numbered menu suffices.
