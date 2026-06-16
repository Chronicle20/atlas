> **SUPERSEDED (2026-06-15)** by `design-quest-driven.md` / `plan-quest-driven.md`. Pet evolution
> is quest-driven; the multi-pet chooser (`pickFromContext`) was backed out. Kept for history.

## Multi-Pet Evolution Chooser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a runtime-sourced selection menu (`pickFromContext` state) to the npc-conversation engine so the Garnox NPC can present a `Name (Species)` chooser when a character has more than one evolution-eligible pet, then evolve the chosen one.

**Architecture:** A new conversation state type `pickFromContext` reuses `askStyle`'s context-sourcing semantics with `listSelection`'s plain numbered-text presentation, plus an `emptyNextState` for the zero-eligible case. The `enumerate_evolvable_pets` local op is extended to emit a parallel display-label list (`Name (Species)`), which requires surfacing the pet's given name (npc `pet` client) and species name (npc `petdata` client). The Garnox seed JSON is rewritten to route `start → pick → confirm → doEvolve`, with `evolve_pet` reading the menu's stored `selectedPetId`. No new condition type, no new Go module — a single service (`atlas-npc-conversations`) is touched.

**Tech Stack:** Go 1.2x, `api2go/jsonapi` REST, immutable Builder models, conversation state-machine engine, JSON seed conversations.

---

## File Structure

**`atlas-npc-conversations` (`services/atlas-npc-conversations/atlas.com/npc/`)**
- Modify `petdata/model.go`, `petdata/rest.go` — surface species `Name()`.
- Modify `pet/model.go`, `pet/rest.go` — surface pet `Name()`.
- Modify `conversation/operation_executor.go` — emit parallel labels in `enumerate_evolvable_pets`.
- Modify `conversation/model.go` — `PickFromContextType` const, `PickFromContextModel` + builder, `StateModel`/`StateBuilder` wiring.
- Modify `conversation/rest.go` — `RestPickFromContextModel` + Transform/Extract + `RestStateModel` field + switch cases.
- Modify `conversation/processor.go` — `processPickFromContextState` presenter + `processState` case + `Continue` case + `splitCSV`/`pickFromContextValues` helpers.
- Modify `deploy/seed/gms/12_1/npc-conversations/npc/npc-1032102.json` — rewrite the conversation.

**Cwd discipline for every task:** run all shell commands from the task worktree root (the `.worktrees/task-089-pet-evolution` checkout); module commands `cd` into `services/atlas-npc-conversations/atlas.com/npc` first. Stage only the explicit files each task names — never `git add -A`/`git add .`. Verify `git branch --show-current` is `task-089-pet-evolution` after each commit. No destructive git ops.

---

# Phase 1 — surface pet name + species name

### Task 1: `petdata` client exposes species `Name()`

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/petdata/rest.go`
- Modify: `services/atlas-npc-conversations/atlas.com/npc/petdata/model.go`
- Test: `services/atlas-npc-conversations/atlas.com/npc/petdata/rest_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `petdata/rest_test.go`:

```go
package petdata

import "testing"

func TestExtractPopulatesName(t *testing.T) {
	rm := RestModel{
		Id:          5000029,
		Name:        "Baby Dragon",
		ReqPetLevel: 15,
		ReqItemId:   5380000,
		Evolutions:  []EvolutionRestModel{{TemplateId: 5000030, Probability: 33}},
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Name() != "Baby Dragon" {
		t.Errorf("Name() = %q, want %q", m.Name(), "Baby Dragon")
	}
	if !m.IsEvolvable() {
		t.Errorf("IsEvolvable() = false, want true")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./petdata/ -run TestExtractPopulatesName -v`
Expected: compile error — `RestModel` has no field `Name`, `Model` has no method `Name`.

- [ ] **Step 3: Add the field + getter + constructor + Extract mapping**

In `petdata/rest.go`, add `Name` to `RestModel` (after `Id`):

```go
type RestModel struct {
	Id          uint32               `json:"-"`
	Name        string               `json:"name"`
	ReqPetLevel uint32               `json:"reqPetLevel"`
	ReqItemId   uint32               `json:"reqItemId"`
	Evolutions  []EvolutionRestModel `json:"evolutions"`
}
```

Update `Extract` to populate `name`:

```go
func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		name:        rm.Name,
		reqPetLevel: rm.ReqPetLevel,
		reqItemId:   rm.ReqItemId,
		evolutions:  len(rm.Evolutions),
	}, nil
}
```

In `petdata/model.go`, add the field, getter, and update `NewModel`:

```go
type Model struct {
	id          uint32
	name        string
	reqPetLevel uint32
	reqItemId   uint32
	evolutions  int
}

func (m Model) Name() string { return m.name }

func NewModel(id uint32, name string, reqPetLevel uint32, reqItemId uint32, evolutions int) Model {
	return Model{
		id:          id,
		name:        name,
		reqPetLevel: reqPetLevel,
		reqItemId:   reqItemId,
		evolutions:  evolutions,
	}
}
```

> `NewModel` gains a `name` parameter. The only callers are this package's `Extract` (uses the struct literal, not `NewModel`) and the test fake in `conversation/operation_executor_petevolution_test.go` — that fake is updated in Task 3. The build will flag any missed caller.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./petdata/ -run TestExtractPopulatesName -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/petdata/model.go services/atlas-npc-conversations/atlas.com/npc/petdata/rest.go services/atlas-npc-conversations/atlas.com/npc/petdata/rest_test.go
git commit -m "feat(npc-conversations): expose pet species name in petdata client"
```

Then verify: `git branch --show-current` → `task-089-pet-evolution`.

### Task 2: npc `pet` client exposes given `Name()`

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/pet/model.go`
- Modify: `services/atlas-npc-conversations/atlas.com/npc/pet/rest.go`
- Test: `services/atlas-npc-conversations/atlas.com/npc/pet/rest_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `pet/rest_test.go`:

```go
package pet

import "testing"

func TestExtractPopulatesName(t *testing.T) {
	rm := RestModel{Id: 7, TemplateId: 5000029, Name: "Fluffy", Level: 20, Slot: 0}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Name() != "Fluffy" {
		t.Errorf("Name() = %q, want %q", m.Name(), "Fluffy")
	}
	if m.TemplateId() != 5000029 || m.Level() != 20 || !m.IsSpawned() {
		t.Errorf("other fields wrong: %+v", m)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./pet/ -run TestExtractPopulatesName -v`
Expected: compile error — `Model` has no method `Name`.

- [ ] **Step 3: Add the field + getter + constructor + Extract mapping**

In `pet/model.go`:

```go
type Model struct {
	id         uint32
	templateId uint32
	name       string
	level      byte
	slot       int8
}

func (m Model) Name() string { return m.name }

func NewModel(id uint32, templateId uint32, name string, level byte, slot int8) Model {
	return Model{
		id:         id,
		templateId: templateId,
		name:       name,
		level:      level,
		slot:       slot,
	}
}
```

In `pet/rest.go`, update `Extract` (the `RestModel` already carries `Name`):

```go
func Extract(rm RestModel) (Model, error) {
	return NewModel(rm.Id, rm.TemplateId, rm.Name, rm.Level, rm.Slot), nil
}
```

> `NewModel` gains a `name` parameter (after `templateId`). Callers: this package's `Extract` and the test fake in `operation_executor_petevolution_test.go` (updated in Task 3).

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./pet/ -run TestExtractPopulatesName -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/pet/model.go services/atlas-npc-conversations/atlas.com/npc/pet/rest.go services/atlas-npc-conversations/atlas.com/npc/pet/rest_test.go
git commit -m "feat(npc-conversations): expose pet given name in pet client"
```

Then verify branch.

---

# Phase 2 — enumerate emits parallel labels

### Task 3: `enumerate_evolvable_pets` writes `Name (Species)` labels

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` (the `case "enumerate_evolvable_pets":` block)
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_petevolution_test.go`

- [ ] **Step 1: Update the test fakes + write the failing assertion**

In `operation_executor_petevolution_test.go`, the existing fakes `fakePetProcessor` and `fakePetDataProcessor` build models via the changed `NewModel` signatures. Update their model construction so each pet has a name and each species has a name. Find where the fakes return pets and petdata and change the `pet.NewModel(...)`/`petdata.NewModel(...)` (or struct construction) calls to include names — e.g. pet ids 1 and 2 named `"Alpha"` and `"Beta"`, both template `5000029`, level 20 and 10; petdata `5000029` named `"Baby Dragon"`, reqPetLevel 15, evolvable.

Then add a focused test:

```go
func TestEnumerateEvolvablePetsEmitsLabels(t *testing.T) {
	// Two summoned pets sharing template 5000029 (Baby Dragon, reqPetLevel 15):
	//   id 1 "Alpha" level 20 (eligible), id 2 "Beta" level 10 (NOT eligible).
	e := newTestExecutorWithPets(t,
		[]pet.Model{
			pet.NewModel(1, 5000029, "Alpha", 20, 0),
			pet.NewModel(2, 5000029, "Beta", 10, 0),
		},
		map[uint32]petdata.Model{
			5000029: petdata.NewModel(5000029, "Baby Dragon", 15, 5380000, 4),
		},
	)
	op := newEnumerateOp(map[string]string{
		"outputContextKey": "evolvablePets",
		"labelContextKey":  "evolvablePetLabels",
		"countContextKey":  "evolvableCount",
	})

	if err := e.executeLocalOperation(testField(t), 100, op); err != nil {
		t.Fatalf("executeLocalOperation: %v", err)
	}

	assertContext(t, e, 100, "evolvablePets", "1")
	assertContext(t, e, 100, "evolvablePetLabels", "Alpha (Baby Dragon)")
	assertContext(t, e, 100, "evolvableCount", "1")
}
```

> Adapt `newTestExecutorWithPets`, `newEnumerateOp`, `testField`, and `assertContext` to the EXACT helpers/fakes the existing `TestEnumerateEvolvablePets` uses in this file — reuse them, do not invent new harness shapes or `*_testhelpers.go`. The point of the test is: only the level-eligible pet (id 1) appears, and its label is `"Alpha (Baby Dragon)"`, index-aligned with the id.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run "TestEnumerateEvolvablePets" -v`
Expected: FAIL — `evolvablePetLabels` is empty/absent (labels not yet emitted).

- [ ] **Step 3: Emit the parallel labels list**

In `operation_executor.go`, replace the `case "enumerate_evolvable_pets":` block body so it also collects and writes labels. The full updated block:

```go
	case "enumerate_evolvable_pets":
		// Params: outputContextKey (default "evolvablePets"),
		//         labelContextKey (default "evolvablePetLabels"),
		//         countContextKey (default "evolvableCount").
		// Lists currently-summoned, evolution-eligible pets into context as
		// index-aligned id + "Name (Species)" label lists, plus a count.
		outputKey := operation.Params()["outputContextKey"]
		if outputKey == "" {
			outputKey = "evolvablePets"
		}
		labelKey := operation.Params()["labelContextKey"]
		if labelKey == "" {
			labelKey = "evolvablePetLabels"
		}
		countKey := operation.Params()["countContextKey"]
		if countKey == "" {
			countKey = "evolvableCount"
		}

		pets, err := e.petP.GetPets(characterId)()
		if err != nil {
			e.l.WithError(err).Errorf("Failed to get pets for character [%d]", characterId)
			return err
		}

		eligible := make([]string, 0)
		labels := make([]string, 0)
		for _, pt := range pets {
			if !pt.IsSpawned() {
				continue
			}
			d, derr := e.petdataP.GetById(pt.TemplateId())
			if derr != nil {
				e.l.WithError(derr).Debugf("Skipping pet [%d] (template [%d]) - no evolution data for character [%d]",
					pt.Id(), pt.TemplateId(), characterId)
				continue
			}
			if d.IsEvolvable() && uint32(pt.Level()) >= d.ReqPetLevel() {
				eligible = append(eligible, strconv.Itoa(int(pt.Id())))
				labels = append(labels, fmt.Sprintf("%s (%s)", pt.Name(), d.Name()))
			}
		}

		if err := e.setContextValue(characterId, outputKey, strings.Join(eligible, ",")); err != nil {
			return err
		}
		if err := e.setContextValue(characterId, labelKey, strings.Join(labels, ",")); err != nil {
			return err
		}
		if err := e.setContextValue(characterId, countKey, strconv.Itoa(len(eligible))); err != nil {
			return err
		}

		e.l.Infof("Enumerated %d evolvable pet(s) for character [%d], stored in context keys [%s, %s, %s]",
			len(eligible), characterId, outputKey, labelKey, countKey)
		return nil
```

`fmt` is already imported in this file.

> **Invariant:** `eligible` (ids) and `labels` are appended in lockstep inside the same loop, so index `i` of the id list corresponds to index `i` of the label list. Both are comma-joined; v83 pet names and atlas-data species names contain no commas (restricted client charset), so comma-splitting downstream is safe.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run "TestEnumerateEvolvablePets" -v`
Expected: PASS (both the original enumerate test and the new labels test).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_petevolution_test.go
git commit -m "feat(npc-conversations): enumerate_evolvable_pets emits Name (Species) labels"
```

Then verify branch.

---

# Phase 3 — the `pickFromContext` state type

### Task 4: `PickFromContextModel` + state wiring (domain model)

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/model.go`
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_model_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `pickfromcontext_model_test.go`:

```go
package conversation

import "testing"

func TestPickFromContextBuilder(t *testing.T) {
	m, err := NewPickFromContextBuilder().
		SetTitle("Which pet?").
		SetValuesContextKey("evolvablePets").
		SetLabelsContextKey("evolvablePetLabels").
		SetContextKey("selectedPetId").
		SetNextState("confirm").
		SetEmptyNextState("noEligible").
		Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.ValuesContextKey() != "evolvablePets" || m.LabelsContextKey() != "evolvablePetLabels" ||
		m.ContextKey() != "selectedPetId" || m.NextState() != "confirm" || m.EmptyNextState() != "noEligible" {
		t.Errorf("fields not set: %+v", m)
	}
}

func TestPickFromContextBuilderRequiresFields(t *testing.T) {
	if _, err := NewPickFromContextBuilder().SetNextState("x").SetEmptyNextState("y").Build(); err == nil {
		t.Error("expected error when valuesContextKey missing")
	}
	if _, err := NewPickFromContextBuilder().SetValuesContextKey("v").SetEmptyNextState("y").Build(); err == nil {
		t.Error("expected error when nextState missing")
	}
	if _, err := NewPickFromContextBuilder().SetValuesContextKey("v").SetNextState("x").Build(); err == nil {
		t.Error("expected error when emptyNextState missing")
	}
}

func TestStateBuilderSetPickFromContext(t *testing.T) {
	pfc, _ := NewPickFromContextBuilder().
		SetValuesContextKey("evolvablePets").SetNextState("confirm").SetEmptyNextState("noEligible").Build()
	s, err := NewStateBuilder().SetId("pick").SetPickFromContext(pfc).Build()
	if err != nil {
		t.Fatalf("state Build: %v", err)
	}
	if s.Type() != PickFromContextType {
		t.Errorf("Type() = %q, want %q", s.Type(), PickFromContextType)
	}
	if s.PickFromContext() == nil || s.PickFromContext().NextState() != "confirm" {
		t.Errorf("PickFromContext() not wired: %+v", s.PickFromContext())
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run "TestPickFromContext|TestStateBuilderSetPickFromContext" -v`
Expected: compile error — types/methods undefined.

- [ ] **Step 3: Add the state-type constant**

In `model.go`, in the `StateType` const block (next to `AskSlideMenuType StateType = "askSlideMenu"`), add:

```go
	PickFromContextType StateType = "pickFromContext"
```

- [ ] **Step 4: Add `PickFromContextModel` + builder**

In `model.go` (place it near `AskStyleModel`), add:

```go
// PickFromContextModel presents a numbered menu whose options are sourced at
// runtime from a context value (a comma-joined list of values), with optional
// parallel display labels. The selected value is stored into ContextKey and the
// conversation advances to NextState. An empty/absent values list routes to
// EmptyNextState instead of presenting a menu.
type PickFromContextModel struct {
	title            string
	valuesContextKey string
	labelsContextKey string
	contextKey       string
	nextState        string
	emptyNextState   string
}

func (m PickFromContextModel) Title() string            { return m.title }
func (m PickFromContextModel) ValuesContextKey() string { return m.valuesContextKey }
func (m PickFromContextModel) LabelsContextKey() string { return m.labelsContextKey }
func (m PickFromContextModel) ContextKey() string       { return m.contextKey }
func (m PickFromContextModel) NextState() string        { return m.nextState }
func (m PickFromContextModel) EmptyNextState() string   { return m.emptyNextState }

type PickFromContextBuilder struct {
	title            string
	valuesContextKey string
	labelsContextKey string
	contextKey       string
	nextState        string
	emptyNextState   string
}

func NewPickFromContextBuilder() *PickFromContextBuilder {
	return &PickFromContextBuilder{contextKey: "selectedValue"}
}

func (b *PickFromContextBuilder) SetTitle(v string) *PickFromContextBuilder            { b.title = v; return b }
func (b *PickFromContextBuilder) SetValuesContextKey(v string) *PickFromContextBuilder { b.valuesContextKey = v; return b }
func (b *PickFromContextBuilder) SetLabelsContextKey(v string) *PickFromContextBuilder { b.labelsContextKey = v; return b }
func (b *PickFromContextBuilder) SetContextKey(v string) *PickFromContextBuilder       { b.contextKey = v; return b }
func (b *PickFromContextBuilder) SetNextState(v string) *PickFromContextBuilder        { b.nextState = v; return b }
func (b *PickFromContextBuilder) SetEmptyNextState(v string) *PickFromContextBuilder   { b.emptyNextState = v; return b }

func (b *PickFromContextBuilder) Build() (*PickFromContextModel, error) {
	if b.valuesContextKey == "" {
		return nil, errors.New("valuesContextKey is required")
	}
	if b.nextState == "" {
		return nil, errors.New("nextState is required")
	}
	if b.emptyNextState == "" {
		return nil, errors.New("emptyNextState is required")
	}
	if b.contextKey == "" {
		b.contextKey = "selectedValue"
	}
	return &PickFromContextModel{
		title:            b.title,
		valuesContextKey: b.valuesContextKey,
		labelsContextKey: b.labelsContextKey,
		contextKey:       b.contextKey,
		nextState:        b.nextState,
		emptyNextState:   b.emptyNextState,
	}, nil
}
```

`errors` is already imported in `model.go`.

- [ ] **Step 5: Wire into `StateModel` and `StateBuilder`**

In `model.go`:
1. Add a field to the `StateModel` struct (after `askSlideMenu *AskSlideMenuModel`): `pickFromContext *PickFromContextModel`.
2. Add the accessor (near `AskSlideMenu()`):

```go
// PickFromContext returns the pick-from-context model (if type is pickFromContext)
func (s StateModel) PickFromContext() *PickFromContextModel {
	return s.pickFromContext
}
```

3. Add the field to the `StateBuilder` struct (mirror `askSlideMenu`).
4. Add the setter (mirror `SetAskStyle`, clearing all other per-type fields including the new one, and setting `b.pickFromContext = pickFromContext`):

```go
func (b *StateBuilder) SetPickFromContext(pickFromContext *PickFromContextModel) *StateBuilder {
	b.stateType = PickFromContextType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	b.pickFromContext = pickFromContext
	return b
}
```

5. In `StateBuilder.Build()`, include `pickFromContext: b.pickFromContext` in the returned `StateModel` literal (mirror how `askSlideMenu` is carried through). Also set the new field to `nil` in every OTHER `SetX` setter that resets per-type fields (mirror exactly how those setters already nil `askSlideMenu`).

> Find each existing `SetDialogue`/`SetGenericAction`/…/`SetAskSlideMenu` setter and add `b.pickFromContext = nil` alongside the other resets, so switching a builder to another type clears a previously-set pickFromContext. Mirror the established pattern precisely.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run "TestPickFromContext|TestStateBuilderSetPickFromContext" -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/model.go services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_model_test.go
git commit -m "feat(npc-conversations): add pickFromContext state model"
```

Then verify branch.

### Task 5: `pickFromContext` REST (de)serialization

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/rest.go`
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_rest_test.go` (create)

- [ ] **Step 1: Write the failing round-trip test**

Create `pickfromcontext_rest_test.go`:

```go
package conversation

import "testing"

func TestPickFromContextRoundTrip(t *testing.T) {
	pfc, _ := NewPickFromContextBuilder().
		SetTitle("Which pet?").
		SetValuesContextKey("evolvablePets").
		SetLabelsContextKey("evolvablePetLabels").
		SetContextKey("selectedPetId").
		SetNextState("confirm").
		SetEmptyNextState("noEligible").
		Build()
	state, err := NewStateBuilder().SetId("pick").SetPickFromContext(pfc).Build()
	if err != nil {
		t.Fatalf("state build: %v", err)
	}

	rest, err := TransformState(state)
	if err != nil {
		t.Fatalf("TransformState: %v", err)
	}
	if rest.StateType != string(PickFromContextType) || rest.PickFromContext == nil {
		t.Fatalf("transform missing pickFromContext: %+v", rest)
	}
	if rest.PickFromContext.ValuesContextKey != "evolvablePets" || rest.PickFromContext.EmptyNextState != "noEligible" {
		t.Errorf("rest fields wrong: %+v", rest.PickFromContext)
	}

	back, err := ExtractState(rest)
	if err != nil {
		t.Fatalf("ExtractState: %v", err)
	}
	got := back.PickFromContext()
	if got == nil || got.ValuesContextKey() != "evolvablePets" || got.LabelsContextKey() != "evolvablePetLabels" ||
		got.ContextKey() != "selectedPetId" || got.NextState() != "confirm" || got.EmptyNextState() != "noEligible" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestPickFromContextRoundTrip -v`
Expected: compile error — `RestStateModel.PickFromContext` / `RestPickFromContextModel` undefined.

- [ ] **Step 3: Add the REST struct + field**

In `rest.go`, add the field to `RestStateModel` (after the `AskSlideMenu` field):

```go
	PickFromContext *RestPickFromContextModel `json:"pickFromContext,omitempty"` // Pick-from-context model (if type is pickFromContext)
```

Add the REST model (near `RestAskStyleModel`):

```go
// RestPickFromContextModel represents the REST model for pickFromContext states
type RestPickFromContextModel struct {
	Title            string `json:"title,omitempty"`
	ValuesContextKey string `json:"valuesContextKey"`
	LabelsContextKey string `json:"labelsContextKey,omitempty"`
	ContextKey       string `json:"contextKey,omitempty"`
	NextState        string `json:"nextState"`
	EmptyNextState   string `json:"emptyNextState"`
}
```

- [ ] **Step 4: Add Transform/Extract functions + switch cases**

In `rest.go`, add (near `TransformAskStyle`/`ExtractAskStyle`):

```go
// TransformPickFromContext converts a PickFromContextModel to its REST form.
func TransformPickFromContext(m PickFromContextModel) RestPickFromContextModel {
	return RestPickFromContextModel{
		Title:            m.Title(),
		ValuesContextKey: m.ValuesContextKey(),
		LabelsContextKey: m.LabelsContextKey(),
		ContextKey:       m.ContextKey(),
		NextState:        m.NextState(),
		EmptyNextState:   m.EmptyNextState(),
	}
}

// ExtractPickFromContext converts a RestPickFromContextModel to the domain model.
func ExtractPickFromContext(r RestPickFromContextModel) (*PickFromContextModel, error) {
	b := NewPickFromContextBuilder().
		SetTitle(r.Title).
		SetValuesContextKey(r.ValuesContextKey).
		SetLabelsContextKey(r.LabelsContextKey).
		SetNextState(r.NextState).
		SetEmptyNextState(r.EmptyNextState)
	if r.ContextKey != "" {
		b.SetContextKey(r.ContextKey)
	}
	return b.Build()
}
```

In `TransformState`'s switch, add (after the `AskSlideMenuType` case):

```go
	case PickFromContextType:
		pfc := m.PickFromContext()
		if pfc != nil {
			restPfc := TransformPickFromContext(*pfc)
			restState.PickFromContext = &restPfc
		}
```

In `ExtractState`'s switch, add (after the `AskSlideMenuType` case, before `default`):

```go
	case PickFromContextType:
		if r.PickFromContext == nil {
			return StateModel{}, fmt.Errorf("pickFromContext is required for pickFromContext state")
		}
		pfc, err := ExtractPickFromContext(*r.PickFromContext)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetPickFromContext(pfc)
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestPickFromContextRoundTrip -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/rest.go services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_rest_test.go
git commit -m "feat(npc-conversations): pickFromContext REST serialization"
```

Then verify branch.

### Task 6: presentation — `processPickFromContextState` + empty routing

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go`
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_processor_test.go` (create)

- [ ] **Step 1: Write the failing test (empty routing)**

Create `pickfromcontext_processor_test.go`. Mirror the harness in `processor_state_transition_test.go` (miniredis + partial `ProcessorImpl{l, ctx, t}` + `testStateContainer` + `NewConversationContextBuilder`). The test asserts that when the values context key is empty/absent, `ProcessState` routes to `emptyNextState` without needing a packet sender:

```go
func TestPickFromContextEmptyRoutesToEmptyNextState(t *testing.T) {
	// Harness: mirror processor_state_transition_test.go EXACTLY for tenant
	// creation, null logger, miniredis, and context.Background usage.
	tctx, l, tm := newProcessorTestEnv(t) // <- replace with the sibling test's inline setup

	pfc, _ := NewPickFromContextBuilder().
		SetValuesContextKey("evolvablePets").
		SetNextState("confirm").
		SetEmptyNextState("noEligible").
		Build()
	pick, _ := NewStateBuilder().SetId("pick").SetPickFromContext(pfc).Build()

	noElig := NewDialogueBuilder().SetDialogueType(SendOk).SetText("none").Build() // match the real dialogue builder API
	noEligible, _ := NewStateBuilder().SetId("noEligible").SetDialogue(noElig).Build()

	container := testStateContainer{start: "pick", states: []StateModel{pick, noEligible}}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(20000)).Build()
	ctx := NewConversationContextBuilder().
		SetField(f).SetCharacterId(7).SetNpcId(1032102).
		SetCurrentState("pick").SetConversation(container).
		AddContextValue("evolvablePets", ""). // empty list → must route to emptyNextState
		Build()
	GetRegistry().SetContext(tctx, ctx.CharacterId(), ctx)

	p := &ProcessorImpl{l: l, ctx: tctx, t: tm}
	if _, err := p.ProcessState(ctx); err != nil {
		t.Fatalf("ProcessState: %v", err)
	}
	got, err := GetRegistry().GetPreviousContext(tctx, ctx.CharacterId())
	if err != nil {
		t.Fatalf("GetPreviousContext: %v", err)
	}
	if got.CurrentState() != "noEligible" {
		t.Errorf("CurrentState = %q, want %q (empty values must route to emptyNextState)", got.CurrentState(), "noEligible")
	}
}
```

> Replace `newProcessorTestEnv` with the EXACT inline setup `processor_state_transition_test.go` uses (it constructs a tenant, a `test.NewNullLogger`, miniredis, and a `tenant.WithContext(context.Background(), tm)` — copy those imports and lines verbatim). Build the `noEligible` dialogue with the package's real dialogue-builder API (inspect an existing dialogue state's construction in the tests or `model.go`). The only assertion that matters: empty values → `CurrentState == "noEligible"`. This path returns BEFORE any packet send, so no producer/sender is required.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestPickFromContextEmptyRoutesToEmptyNextState -v`
Expected: compile error / FAIL — `processState` does not handle `PickFromContextType`.

- [ ] **Step 3: Add the CSV helper, presenter, and switch case**

In `processor.go`, add a helper near the other file-level helpers:

```go
// splitCSV splits a comma-joined context value, returning nil for an empty
// string (so an absent/empty list is len 0, not [""]).
func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
```

Add the presenter (mirror `processListSelectionState`):

```go
func (p *ProcessorImpl) processPickFromContextState(ctx ConversationContext, state StateModel) (string, error) {
	m := state.PickFromContext()
	if m == nil {
		return "", errors.New("pickFromContext is nil")
	}

	values := splitCSV(ctx.Context()[m.ValuesContextKey()])
	if len(values) == 0 {
		// No options to choose from — route to the empty fallback state.
		return m.EmptyNextState(), nil
	}

	// Parallel labels (fall back to the raw values if absent or mismatched).
	labels := values
	if m.LabelsContextKey() != "" {
		ls := splitCSV(ctx.Context()[m.LabelsContextKey()])
		if len(ls) == len(values) {
			labels = ls
		}
	}

	processedTitle, err := ReplaceContextPlaceholders(m.Title(), ctx.Context())
	if err != nil {
		p.l.WithError(err).Warnf("Failed to replace context placeholders in pickFromContext title for state [%s]. Using original title.", state.Id())
		processedTitle = m.Title()
	}

	mb := message.NewBuilder().AddText(processedTitle).NewLine()
	for i, label := range labels {
		processedLabel, lerr := ReplaceContextPlaceholders(label, ctx.Context())
		if lerr != nil {
			processedLabel = label
		}
		mb.OpenItem(i).BlueText().AddText(processedLabel).CloseItem().NewLine()
	}

	npcSender.NewProcessor(p.l, p.ctx).SendSimple(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId())(mb.String())
	return state.Id(), nil
}
```

In `processState`'s switch, add (before `default`):

```go
	case PickFromContextType:
		return p.processPickFromContextState(ctx, state)
```

> The presentation loop emits one `OpenItem(i)` per option with NO skipping, so the client's returned selection index maps directly to `values[selection]` (Task 7). `message`, `npcSender`, `strings`, and `errors` are already imported in `processor.go`.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestPickFromContextEmptyRoutesToEmptyNextState -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_processor_test.go
git commit -m "feat(npc-conversations): present pickFromContext menu, route empty to fallback"
```

Then verify branch.

### Task 7: selection handling — `Continue` case + `pickFromContextValues`

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go`
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_processor_test.go`

- [ ] **Step 1: Write the failing test for the pure selection helper**

Append to `pickfromcontext_processor_test.go`:

```go
func TestPickFromContextValues(t *testing.T) {
	if v, err := pickFromContextValues("10,20,30", 1); err != nil || v != "20" {
		t.Errorf("index 1 -> (%q,%v), want (\"20\",nil)", v, err)
	}
	if v, err := pickFromContextValues("10,20,30", 0); err != nil || v != "10" {
		t.Errorf("index 0 -> (%q,%v), want (\"10\",nil)", v, err)
	}
	if _, err := pickFromContextValues("10,20,30", 3); err == nil {
		t.Error("index 3 (out of bounds) -> want error")
	}
	if _, err := pickFromContextValues("10,20,30", -1); err == nil {
		t.Error("index -1 -> want error")
	}
	if _, err := pickFromContextValues("", 0); err == nil {
		t.Error("empty list -> want error")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestPickFromContextValues -v`
Expected: compile error — `pickFromContextValues` undefined.

- [ ] **Step 3: Add the helper + the `Continue` case**

In `processor.go`, add the helper (near `splitCSV`):

```go
// pickFromContextValues returns the value at the given selection index within a
// comma-joined context list, or an error if the list is empty or the index is
// out of bounds.
func pickFromContextValues(valuesStr string, selection int32) (string, error) {
	values := splitCSV(valuesStr)
	if len(values) == 0 {
		return "", errors.New("no values available in context")
	}
	if selection < 0 || selection >= int32(len(values)) {
		return "", fmt.Errorf("selection [%d] out of bounds [0,%d)", selection, len(values))
	}
	return values[selection], nil
}
```

In `Continue`'s state-type switch (the one with `case ListSelectionType` / `case AskStyleType` that assigns `nextStateId` and `choiceContext`), add a case before `default`:

```go
	case PickFromContextType:
		pfc := state.PickFromContext()
		if pfc == nil {
			return errors.New("pickFromContext is nil")
		}
		// action == 0 means the player cancelled/closed the window: end the
		// conversation (leave nextStateId empty). Otherwise selection is the index.
		if action != 0 {
			selected, serr := pickFromContextValues(ctx.Context()[pfc.ValuesContextKey()], selection)
			if serr != nil {
				p.l.Errorf("Invalid pickFromContext selection [%d] for character [%d] in state [%s]: %v", selection, characterId, state.Id(), serr)
				return serr
			}
			choiceContext = map[string]string{pfc.ContextKey(): selected}
			nextStateId = pfc.NextState()
		}
```

`fmt` and `errors` are already imported.

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestPickFromContextValues -v`
Expected: PASS.

- [ ] **Step 5: Full module test/vet/build**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test -race ./... && go vet ./... && go build ./...`
Expected: all clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_processor_test.go
git commit -m "feat(npc-conversations): handle pickFromContext selection in Continue"
```

Then verify branch.

---

# Phase 4 — wire the Garnox conversation + verify

### Task 8: Rewrite the Garnox conversation to use the menu

**Files:**
- Modify: `deploy/seed/gms/12_1/npc-conversations/npc/npc-1032102.json`

- [ ] **Step 1: Rewrite the seed**

Replace the entire file with the following. Changes vs. the current file: `start`'s else-outcome now → `pick`; a new `pick` (`pickFromContext`) state; the enumerate op gains `labelContextKey`; `doEvolve`'s `evolve_pet` now reads `{context.selectedPetId}`; a new `noEligible` dialogue.

```json
{
  "data": {
    "attributes": {
      "npcId": 1032102,
      "startState": "start",
      "states": [
        {
          "id": "start",
          "type": "genericAction",
          "genericAction": {
            "operations": [
              {
                "type": "local:enumerate_evolvable_pets",
                "params": {
                  "outputContextKey": "evolvablePets",
                  "labelContextKey": "evolvablePetLabels",
                  "countContextKey": "evolvableCount"
                }
              }
            ],
            "outcomes": [
              {
                "conditions": [
                  { "type": "item", "operator": "<", "referenceId": "5380000", "value": "1" }
                ],
                "nextState": "noRock"
              },
              { "conditions": [], "nextState": "pick" }
            ]
          }
        },
        {
          "id": "pick",
          "type": "pickFromContext",
          "pickFromContext": {
            "title": "Which companion shall I guide through evolution?",
            "valuesContextKey": "evolvablePets",
            "labelsContextKey": "evolvablePetLabels",
            "contextKey": "selectedPetId",
            "nextState": "confirm",
            "emptyNextState": "noEligible"
          }
        },
        {
          "id": "confirm",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendYesNo",
            "text": "I am Garnox, keeper of the Rock of Time. For #b600000 mesos#k and a #t5380000#, I can guide your companion through its evolution. Shall we begin?",
            "choices": [
              { "text": "Yes", "nextState": "checkMeso" },
              { "text": "No", "nextState": "decline" },
              { "text": "Exit", "nextState": null }
            ]
          }
        },
        {
          "id": "checkMeso",
          "type": "genericAction",
          "genericAction": {
            "operations": [],
            "outcomes": [
              {
                "conditions": [
                  { "type": "meso", "operator": "<", "value": "600000" }
                ],
                "nextState": "noMeso"
              },
              { "conditions": [], "nextState": "doEvolve" }
            ]
          }
        },
        {
          "id": "doEvolve",
          "type": "genericAction",
          "genericAction": {
            "operations": [
              { "type": "destroy_item", "params": { "itemId": "5380000", "quantity": "1" } },
              { "type": "award_mesos", "params": { "amount": "-600000" } },
              { "type": "evolve_pet", "params": { "petId": "{context.selectedPetId}" } }
            ],
            "outcomes": [
              { "conditions": [], "nextState": "success" }
            ]
          }
        },
        {
          "id": "success",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "The Rock of Time has done its work. Behold your companion's new form! Treat it well, and return to me when it is ready to grow once more.",
            "choices": [
              { "text": "Ok", "nextState": null },
              { "text": "Exit", "nextState": null }
            ]
          }
        },
        {
          "id": "noEligible",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "None of your summoned companions are ready to evolve. Summon a pet that has grown strong enough, then seek me again.",
            "choices": [
              { "text": "Ok", "nextState": null },
              { "text": "Exit", "nextState": null }
            ]
          }
        },
        {
          "id": "noRock",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "Evolution cannot begin without a #t5380000#. Bring me one, and make sure your companion is summoned and strong enough to take the next step.",
            "choices": [
              { "text": "Ok", "nextState": null },
              { "text": "Exit", "nextState": null }
            ]
          }
        },
        {
          "id": "noMeso",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "The ritual requires #b600000 mesos#k. Come back when you can cover the cost.",
            "choices": [
              { "text": "Ok", "nextState": null },
              { "text": "Exit", "nextState": null }
            ]
          }
        },
        {
          "id": "decline",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "Take your time. The Rock of Time waits for those who are ready.",
            "choices": [
              { "text": "Ok", "nextState": null },
              { "text": "Exit", "nextState": null }
            ]
          }
        }
      ]
    },
    "id": "1032102",
    "type": "npc-conversation"
  }
}
```

- [ ] **Step 2: Validate JSON + state-target resolution**

Run:
```bash
cat deploy/seed/gms/12_1/npc-conversations/npc/npc-1032102.json | python3 -m json.tool > /dev/null && echo VALID_JSON
```
Expected: `VALID_JSON`.

Manually confirm every `nextState`/`emptyNextState`/outcome target resolves to a defined state id: targets used are `noRock, pick, confirm, checkMeso, decline, noMeso, doEvolve, success, noEligible` and `null` (end) — all of `pick, confirm, checkMeso, doEvolve, success, noEligible, noRock, noMeso, decline` are defined; `startState` is `start` which exists; `data.id` is `"1032102"` (matches the `npc-1032102.json` filename, required by the seeder).

- [ ] **Step 3: Commit**

```bash
git add deploy/seed/gms/12_1/npc-conversations/npc/npc-1032102.json
git commit -m "feat(npc-conversations): Garnox presents a multi-pet evolution chooser"
```

Then verify branch.

### Task 9: Full verification gate

**Files:** none (verification).

- [ ] **Step 1: Module test/vet/build**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test -race ./... && go vet ./... && go build ./...`
Expected: all clean.

- [ ] **Step 2: redis-key-guard (workspace active — NOT `GOWORK=off`)**

Run from the worktree root: `tools/redis-key-guard.sh`
Expected: exit 0, clean. (Running with `GOWORK=off` produces a known worktree "matched no packages" false-positive; no redis code is added by this work regardless.)

- [ ] **Step 3: docker bake the touched service**

Run from the worktree root: `docker buildx bake atlas-npc-conversations`
Expected: image builds (`naming to docker.io/library/atlas-npc-conversations:local done`). No `go.mod`/`Dockerfile`/`go.work` changed, but re-bake to keep "verified" honest since service source changed.

- [ ] **Step 4: Commit any fixups**

If steps 1-3 required fixes, stage the specific changed files explicitly and:
```bash
git commit -m "chore(task-089): multi-pet chooser verification fixups"
```
Then verify branch.

---

## Acceptance criteria mapping

| Requirement (design) | Task(s) |
|---|---|
| Menu rows show `Name (Species)` | 1, 2, 3 |
| `enumerate` emits index-aligned ids + labels | 3 |
| New `pickFromContext` state: model + REST round-trip | 4, 5 |
| Menu presented for >=1 eligible (incl. one) | 6 |
| 0 eligible → `emptyNextState` (no empty menu) | 6 |
| Selection stores chosen id into `contextKey`, advances to `nextState`; bounds-safe; cancel ends | 7 |
| Garnox flow `start → pick → confirm → doEvolve`; `evolve_pet` uses `{context.selectedPetId}` | 8 |
| Rock + meso gates preserved | 8 |
| Build/test/vet/redis-guard/bake clean | 9 |

## Notes / invariants
- Values (`evolvablePets`) and labels (`evolvablePetLabels`) are produced in lockstep by `enumerate`; comma delimiter is safe because v83 pet/species names contain no commas.
- The presentation emits one menu item per option with no skipping, so the client's returned `selection` index maps directly to `values[selection]` in `Continue`.
- `evolve_pet` reads `{context.selectedPetId}` (a single id from the menu), which is what makes the >1-eligible case work — replacing the base feature's `{context.evolvablePets}` that only parsed for exactly one pet.
- No new condition type, no new Go module, no `go.mod`/`Dockerfile`/`go.work` change. Single service: `atlas-npc-conversations`.
