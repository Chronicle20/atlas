# Pet Evolution — Quest-Driven Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the synthetic Garnox free-conversation with the canonical quest-driven path: enforce the pet/tameness quest gate, author the Garnox quest conversations whose end-state-machine performs the evolution, and back out the synthetic conversation + `pickFromContext`.

**Architecture:** The startscript/endscript mechanism already exists (quest conversations keyed by `questId`, start/end state machines, `QuestActionScriptStart/End` trigger). Workstreams: (A) **back out** the synthetic layer; (B) add a contained **`petTameness`** validation condition (query-aggregator already fetches per-pet closeness — it's just discarded) plus a GM `@award <target> tameness <amount>` test command (atlas-messages, reusing atlas-pets `AWARD_CLOSENESS`); (C) **author** quest conversations `8185/8189/4659` from the Cosmic `.js` via `/convert-quest`, end machine running `destroy_item` → `evolve_pet` → `complete_quest`. The evolution engine (atlas-pets EVOLVE, saga PetEvolution, inventory CHANGE_TEMPLATE, atlas-data evol parsing, egg fix) is retained unchanged.

**Tech Stack:** Go (atlas-quest, atlas-query-aggregator, atlas-npc-conversations, atlas-messages), `libs/atlas-saga` (shared condition constants), TypeScript/React (atlas-ui backout), JSON seed data, Kafka, GORM/JSON:API.

**Spec:** `docs/tasks/task-089-pet-evolution/design-quest-driven.md`. Supersedes `design.md`/`plan.md` (free-conversation) for the trigger/UX layer; the engine plan there still stands.

**External dependency:** The Cosmic `q8185s/e.js`, `q8189e.js`, `q4659s/e.js` scripts are **not in the repo** (`find` confirms). Phase C is blocked until they are obtained (user provides, or fetch from the Cosmic source repo). Phases A and B have no such dependency and can land first.

**Path convention:** All commands run from the worktree root unless a `cd` says otherwise. `cd "$(git rev-parse --show-toplevel)"` returns to the worktree root. Go module dirs (run checks there with `GOWORK=off` — the worktree `go.work` is unreliable for `./...`, confirmed this session):
- atlas-quest: `services/atlas-quest/atlas.com/quest`
- atlas-query-aggregator: `services/atlas-query-aggregator/atlas.com/query-aggregator`
- atlas-npc-conversations: `services/atlas-npc-conversations/atlas.com/npc`
- atlas-pets: `services/atlas-pets/atlas.com/pets`
- libs/atlas-saga: `libs/atlas-saga`

Per-module checks: `GOWORK=off go test -race ./...`, `GOWORK=off go vet ./...`, `GOWORK=off go build ./...`. Plus `docker buildx bake atlas-<svc>` from the worktree root for any service whose go.mod is touched, and `tools/redis-key-guard.sh` from repo root.

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `libs/atlas-saga/validation.go` | Shared condition-type constants | add `PetTamenessCondition` |
| `services/atlas-query-aggregator/.../validation/model.go` | Condition eval + input→Condition builder | add `petTameness` eval + input validation |
| `services/atlas-query-aggregator/.../validation/context.go` | ValidationContext (fetched character state) | retain spawned-pet detail (templateId+closeness) |
| `services/atlas-query-aggregator/.../validation/processor.go` | Builds the context (fetches pets) | populate pet detail; gate on `reqs.Pets` |
| `services/atlas-quest/.../data/validation/model.go` | atlas-quest wire condition constants | add `PetTamenessCondition` |
| `services/atlas-quest/.../data/validation/processor.go` | `buildStartConditions`/`buildEndConditions` | emit `petTameness` from `req.Pet`+`req.PetTamenessMin` |
| `services/atlas-messages/.../command/pet/commands.go` + `messages/pet/` lookup + `kafka/message/pet/` producer + `messages/main.go` | GM `@award <target> tameness <amount>` command (reuses atlas-pets `AWARD_CLOSENESS`) | **create** |
| `services/atlas-npc-conversations/.../conversation/{model,rest,processor}.go` + 3 `pickfromcontext_*_test.go` | pickFromContext state type | **remove** |
| `services/atlas-ui/src/types/models/conversation.ts`, `.../conversation/stateMeta.ts`, `transitions.ts` | UI pickFromContext support | **revert** |
| `deploy/seed/gms/*/npc-conversations/npc/npc-9102001.json` (6) | synthetic free conversation | **delete** |
| `deploy/seed/gms/*/npc-conversations/quests/quest-{8185,8189,4659}.json` | authored quest conversations | **create** |

---

## Phase A — Back out the synthetic conversation

> No external dependency. Land first; it removes the wrong approach and shrinks the surface.
> **Keep** `evolve_pet` and `enumerate_evolvable_pets` operations, the evolution engine, and the egg-reader fix — only `pickFromContext` and the free conversation are removed.

### Task A1: Delete the synthetic Garnox free-conversation seeds

**Files:**
- Delete: `deploy/seed/gms/{12_1,83_1,84_1,87_1,92_1,95_1}/npc-conversations/npc/npc-9102001.json` (6 files)

- [ ] **Step 1: Remove the files**

```bash
cd "$(git rev-parse --show-toplevel)"
git rm deploy/seed/gms/*/npc-conversations/npc/npc-9102001.json
```

- [ ] **Step 2: Verify none remain**

Run: `find deploy/seed -name 'npc-9102001.json' | wc -l`
Expected: `0`

- [ ] **Step 3: Commit**

```bash
git commit -m "revert(task-089): remove synthetic Garnox free-conversation seed"
```

### Task A2: Revert the atlas-ui pickFromContext support

The UI additions (`conversation.ts` type, `stateMeta.ts` entry + describe case, `transitions.ts` case) only existed to render the synthetic state type. Remove them so the type union no longer carries `pickFromContext`.

**Files:**
- Modify: `services/atlas-ui/src/types/models/conversation.ts`
- Modify: `services/atlas-ui/src/components/features/npc/conversation/stateMeta.ts`
- Modify: `services/atlas-ui/src/components/features/npc/conversation/transitions.ts`

- [ ] **Step 1: Revert the three files to their pre-fix state**

The exact additions were made earlier on this branch in commit `81f57e4e8`. Revert just those hunks:

```bash
cd "$(git rev-parse --show-toplevel)"
git revert --no-commit 81f57e4e8
```

If the revert conflicts (later edits touched the files), instead manually remove: the `PickFromContextState` interface and the `| "pickFromContext"` union member and `pickFromContext?:` field in `conversation.ts`; the `pickFromContext:` entry in `STATE_TYPE_META` and the `case "pickFromContext":` in `describeState` in `stateMeta.ts`; the `case "pickFromContext":` block in `transitions.ts`.

- [ ] **Step 2: Build the UI (tsc -b type-checks tests too)**

```bash
cd services/atlas-ui
export NVM_DIR="$HOME/.nvm"; . "$NVM_DIR/nvm.sh"; nvm use 22
npm run build
```
Expected: build succeeds; `grep -rn pickFromContext src/` → empty.

- [ ] **Step 3: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-ui/src
git commit -m "revert(task-089): drop atlas-ui pickFromContext support"
```

### Task A3: Remove the pickFromContext backend state type

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/model.go` (const `PickFromContextType`, `PickFromContextModel` + builder, `StateModel`/`StateBuilder` field + accessor + `SetPickFromContext`, nil-reset in sibling setters, `Build` carry)
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/rest.go` (`RestPickFromContextModel`, `RestStateModel` field, `TransformPickFromContext`/`ExtractPickFromContext`, the `TransformState`/`ExtractState` switch cases)
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go` (`processPickFromContextState`, the `processState` case, the `Continue` `PickFromContextType` case, `pickFromContextValues`/`splitCSV` helpers if unused elsewhere)
- Delete: `services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_model_test.go`, `pickfromcontext_rest_test.go`, `pickfromcontext_processor_test.go`

- [ ] **Step 1: Delete the three test files**

```bash
cd "$(git rev-parse --show-toplevel)"
git rm services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_model_test.go \
       services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_rest_test.go \
       services/atlas-npc-conversations/atlas.com/npc/conversation/pickfromcontext_processor_test.go
```

- [ ] **Step 2: Remove pickFromContext from model.go, rest.go, processor.go**

Read each file and delete every symbol enumerated in **Files** above. Find the exact hunks via the introducing commits:

```bash
git log --oneline --all | grep -iE 'pickFromContext|pick-from-context'
```
(Commits `1d2cd249a` model, `6cfc34924` rest, `d9a176f80`+`b7626eeca` processor introduced them.)

Removal guidance: `splitCSV` may be shared — only remove it if `grep -rn 'splitCSV' services/atlas-npc-conversations` shows no other caller. Keep `enumerate_evolvable_pets`/`evolve_pet` (in `operation_executor.go`) untouched.

- [ ] **Step 3: Verify it compiles and tests pass**

```bash
cd services/atlas-npc-conversations/atlas.com/npc
GOWORK=off go build ./... && GOWORK=off go vet ./... && GOWORK=off go test -race ./conversation/...
```
Expected: all pass; `grep -rni pickfromcontext . | grep -v _test` → empty.

- [ ] **Step 4: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-npc-conversations
git commit -m "revert(task-089): remove pickFromContext conversation state type"
```

### Task A4: Mark the superseded design/plan docs

- [ ] **Step 1: Add a supersession banner to the two chooser docs**

Prepend to both `docs/tasks/task-089-pet-evolution/design-multi-pet-chooser.md` and `plan-multi-pet-chooser.md`:

```markdown
> **SUPERSEDED (2026-06-15)** by `design-quest-driven.md` / `plan-quest-driven.md`. Pet evolution
> is quest-driven; the multi-pet chooser (`pickFromContext`) was backed out. Kept for history.
```

- [ ] **Step 2: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add docs/tasks/task-089-pet-evolution/design-multi-pet-chooser.md docs/tasks/task-089-pet-evolution/plan-multi-pet-chooser.md
git commit -m "docs(task-089): mark multi-pet-chooser design/plan superseded"
```

---

## Phase B — `petTameness` validation condition + GM test command

> No external dependency. Independently shippable and testable. Enforces "a **summoned** pet whose
> templateId ∈ {ids} has closeness ≥ N".
>
> **Verified facts this rests on:** query-aggregator's `Condition` already has `values []int` and
> `value int` (`validation/model.go:90-94`); it already fetches per-pet `Closeness`+`TemplateId`
> (`pet/rest.go:12-19`, `pet/processor.go:32`) but the ValidationContext keeps only `petCount`
> (`validation/context.go:37`). Eval reduces a condition to `actualValue` compared via operator to
> `c.value` (`validation/model.go:385`).

### Task B1: Add the shared condition constant

**Files:** Modify `libs/atlas-saga/validation.go` (same const block as `PetCountCondition`, line 32)

- [ ] **Step 1: Add the constant**

```go
	PetTamenessCondition            = "petTameness"
```

- [ ] **Step 2: Build** — `cd libs/atlas-saga && GOWORK=off go build ./...` → success.
- [ ] **Step 3: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add libs/atlas-saga/validation.go
git commit -m "feat(atlas-saga): add petTameness validation condition constant"
```

### Task B2: Retain spawned-pet detail in the ValidationContext

**Files:**
- Modify: `services/atlas-query-aggregator/.../validation/context.go`
- Modify: `services/atlas-query-aggregator/.../validation/processor.go`
- Test: `services/atlas-query-aggregator/.../validation/context_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

```go
func TestValidationContext_MaxPetClosenessForTemplates(t *testing.T) {
	ctx := NewValidationContextBuilder().
		SetSpawnedPets([]SpawnedPet{
			{TemplateId: 5000029, Closeness: 1700},
			{TemplateId: 5000048, Closeness: 50},
		}).
		Build()

	if got := ctx.MaxPetClosenessForTemplates([]uint32{5000029}); got != 1700 {
		t.Fatalf("MaxPetClosenessForTemplates([5000029]) = %d, want 1700", got)
	}
	if got := ctx.MaxPetClosenessForTemplates([]uint32{5000048}); got != 50 {
		t.Fatalf("MaxPetClosenessForTemplates([5000048]) = %d, want 50", got)
	}
	if got := ctx.MaxPetClosenessForTemplates([]uint32{5000030}); got != 0 {
		t.Fatalf("MaxPetClosenessForTemplates([5000030]) = %d, want 0", got)
	}
	if got := ctx.MaxPetClosenessForTemplates([]uint32{5000029, 5000048}); got != 1700 {
		t.Fatalf("MaxPetClosenessForTemplates([...]) = %d, want 1700", got)
	}
}
```

- [ ] **Step 2: Run it — expect compile failure**

Run: `cd services/atlas-query-aggregator/atlas.com/query-aggregator && GOWORK=off go test ./validation/ -run TestValidationContext_MaxPetClosenessForTemplates`
Expected: FAIL — `SpawnedPet`/`SetSpawnedPets`/`MaxPetClosenessForTemplates` undefined.

- [ ] **Step 3: Implement in context.go**

```go
// SpawnedPet is the minimal per-pet detail the validation context retains for
// pet-based conditions (e.g. petTameness).
type SpawnedPet struct {
	TemplateId uint32
	Closeness  uint16
}
```
Add `spawnedPets []SpawnedPet` to both `ValidationContext` and `ValidationContextBuilder`, and to every copy site (search `petCount:` — add a parallel `spawnedPets:` line at each). Then:
```go
func (ctx ValidationContext) SpawnedPets() []SpawnedPet { return ctx.spawnedPets }

// MaxPetClosenessForTemplates returns the highest closeness among spawned pets
// whose template id is in templateIds, or 0 if none match.
func (ctx ValidationContext) MaxPetClosenessForTemplates(templateIds []uint32) int {
	max := 0
	for _, p := range ctx.spawnedPets {
		for _, id := range templateIds {
			if p.TemplateId == id && int(p.Closeness) > max {
				max = int(p.Closeness)
			}
		}
	}
	return max
}

func (b *ValidationContextBuilder) SetSpawnedPets(pets []SpawnedPet) *ValidationContextBuilder {
	b.spawnedPets = pets
	return b
}
```
Ensure `Build()` copies `spawnedPets: b.spawnedPets`.

- [ ] **Step 4: Run the test — expect pass**

Run: `cd services/atlas-query-aggregator/atlas.com/query-aggregator && GOWORK=off go test ./validation/ -run TestValidationContext_MaxPetClosenessForTemplates`
Expected: PASS.

- [ ] **Step 5: Extend the pet `Model` to carry templateId + closeness**

**VERIFIED GAP (`pet/model.go:4-7`, `pet/rest.go:46-47`):** the `Model` holds only `id` + `slot`; `Extract` **drops** templateId/closeness (present on `RestModel` at `rest.go:12,15`). So `GetPets()` Models cannot supply closeness today — extend the Model. In `pet/model.go` add fields `templateId uint32`, `closeness uint16`, accessors `TemplateId()`/`Closeness()`, and widen:
```go
func NewModel(id uint32, slot int8, templateId uint32, closeness uint16) Model {
	return Model{id: id, slot: slot, templateId: templateId, closeness: closeness}
}
```
Update the 4 `NewModel` callers: `pet/rest.go:47` → `NewModel(rm.Id, rm.Slot, rm.TemplateId, rm.Closeness)`; tests `pet/processor_test.go:17,18,19,159` → add `, 0, 0` (or representative values). Use the existing `IsSpawned()` (`model.go:25`).

- [ ] **Step 6: Populate spawnedPets at the fetch site (processor.go)**

**VERIFIED (`processor.go:692`, `:237-239`, `:35,49`):** the block `if reqs.Pets && p.petCountProvider != nil` calls `petCountProvider` (wired from `p.petProcessor.GetSpawnedPetCount`). Add a `petsProvider func(uint32) model.Provider[[]pet.Model]` wired from `p.petProcessor.GetPets`, then replace the block body:
```go
		if reqs.Pets && p.petsProvider != nil {
			pets, err := p.petsProvider(characterId)()
			if err != nil {
				p.l.WithError(err).Debugf("Failed to get pet data for character %d, treating as no pets", characterId)
			} else {
				detail := make([]SpawnedPet, 0, len(pets))
				count := 0
				for _, pt := range pets {
					if pt.IsSpawned() {
						count++
						detail = append(detail, SpawnedPet{TemplateId: pt.TemplateId(), Closeness: pt.Closeness()})
					}
				}
				builder.SetPetCount(count).SetSpawnedPets(detail)
			}
		}
```

- [ ] **Step 7: Set `reqs.Pets` for petTameness**

**VERIFIED (`processor.go:192-193`, unioned at `:84`):** `requirementsFor(conditionType)` returns `ContextRequirements{Pets: true}` for `PetCountCondition`. Extend that case:
```go
	case PetCountCondition, PetTamenessCondition:
		return ContextRequirements{Pets: true}
```

- [ ] **Step 8: Build + tests** — `cd services/atlas-query-aggregator/atlas.com/query-aggregator && GOWORK=off go build ./... && GOWORK=off go test -race ./...` → PASS (includes updated `pet/processor_test.go`).
- [ ] **Step 9: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-query-aggregator
git commit -m "feat(query-aggregator): retain spawned-pet closeness in validation context"
```

### Task B3: Evaluate the `petTameness` condition

**Files:**
- Modify: `services/atlas-query-aggregator/.../validation/model.go` (const alias, input-validation switch ~233, eval switch ~392)
- Test: `services/atlas-query-aggregator/.../validation/model_test.go` (append; reuse the existing Condition/ctx test scaffolding — read it first)

- [ ] **Step 1: Write the failing test**

```go
func TestEvaluate_PetTameness(t *testing.T) {
	ctx := NewValidationContextBuilder().
		SetSpawnedPets([]SpawnedPet{{TemplateId: 5000029, Closeness: 1700}}).
		Build()

	pass := newConditionForTest(PetTamenessCondition, ">=", 1642, []int{5000029})
	if !pass.Evaluate(testLogger(), testCtx(), ctx).Passed {
		t.Fatal("expected petTameness >=1642 to pass for closeness 1700")
	}
	fail := newConditionForTest(PetTamenessCondition, ">=", 1642, []int{5000030})
	if fail.Evaluate(testLogger(), testCtx(), ctx).Passed {
		t.Fatal("expected petTameness to fail when no spawned pet matches the id set")
	}
}
```
(Replace `newConditionForTest`/`testLogger`/`testCtx` with the package's existing test helpers.)

- [ ] **Step 2: Run it — expect failure**

Run: `cd services/atlas-query-aggregator/atlas.com/query-aggregator && GOWORK=off go test ./validation/ -run TestEvaluate_PetTameness`
Expected: FAIL — `PetTamenessCondition` undefined / no eval case.

- [ ] **Step 3: Add const alias + input validation + eval case**

In `validation/model.go` const block:
```go
	PetTamenessCondition            ConditionType = ConditionType(sharedsaga.PetTamenessCondition)
```
Add `PetTamenessCondition` to the "known types" list (the big `case ...:` at ~line 128).
Input-validation switch (~233):
```go
	case PetTamenessCondition:
		if len(input.Values) == 0 {
			b.err = fmt.Errorf("values (pet template ids) required for petTameness conditions")
		}
```
Eval switch (`Evaluate`, ~392), before the switch closes:
```go
	case PetTamenessCondition:
		actualValue = ctx.MaxPetClosenessForTemplates(uint32Slice(c.values))
		description = fmt.Sprintf("Pet Tameness (templates %v) %s %d", c.values, c.operator, c.value)
```
Helper near the bottom (if no equivalent exists):
```go
func uint32Slice(in []int) []uint32 {
	out := make([]uint32, 0, len(in))
	for _, v := range in {
		out = append(out, uint32(v))
	}
	return out
}
```
The existing post-switch operator compare yields pass/fail — no extra logic.

- [ ] **Step 4: Run — expect pass** — same command as Step 1 → PASS.
- [ ] **Step 5: Full module checks** — `GOWORK=off go build ./... && GOWORK=off go vet ./... && GOWORK=off go test -race ./...` → PASS.
- [ ] **Step 6: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-query-aggregator
git commit -m "feat(query-aggregator): evaluate petTameness condition"
```

### Task B4: Emit `petTameness` from atlas-quest validation

**Files:**
- Modify: `services/atlas-quest/.../data/validation/model.go` (const)
- Modify: `services/atlas-quest/.../data/validation/processor.go` (`buildStartConditions` + `buildEndConditions`)
- Test: `services/atlas-quest/.../data/validation/processor_test.go` (append)

- [ ] **Step 1: Write the failing test**

**VERIFIED (`processor.go:55-56`):** `buildStartConditions(questDef dataquest.RestModel)` with `req := questDef.StartRequirements`. The package has **no fixture helper** — tests build `dataquest.RestModel` inline (`processor_test.go:18-24`). Match that:
```go
func TestBuildStartConditions_PetTameness(t *testing.T) {
	qd := dataquest.RestModel{
		StartRequirements: dataquest.RequirementsRestModel{
			Pet:            []uint32{5000029},
			PetTamenessMin: 1642,
		},
	}
	conds := buildStartConditions(qd)

	var found *ConditionInput
	for i := range conds {
		if conds[i].Type == PetTamenessCondition {
			found = &conds[i]
		}
	}
	if found == nil {
		t.Fatal("expected a petTameness condition")
	}
	if found.Operator != ">=" || found.Value != 1642 {
		t.Fatalf("petTameness op/value = %s/%d, want >=/1642", found.Operator, found.Value)
	}
	if len(found.Values) != 1 || found.Values[0] != 5000029 {
		t.Fatalf("petTameness Values = %v, want [5000029]", found.Values)
	}
}
```
(Field names `Pet []uint32` / `PetTamenessMin int16` VERIFIED at `data/quest/rest.go:62-63`; the wrapping `dataquest.RestModel{StartRequirements: ...}` matches the inline pattern at `processor_test.go:18-24`.)

- [ ] **Step 2: Run it — expect failure**

Run: `cd services/atlas-quest/atlas.com/quest && GOWORK=off go test ./data/validation/ -run TestBuildStartConditions_PetTameness`
Expected: FAIL.

- [ ] **Step 3: Add the constant** — in `data/validation/model.go` (next to `MonsterBookCountCondition`):
```go
	PetTamenessCondition      = "petTameness"
```

- [ ] **Step 4: Emit in both builders**

In `buildStartConditions` AND `buildEndConditions`, after existing emissions:
```go
	// Pet + tameness gate: a summoned pet of one of the listed templates must
	// have closeness >= PetTamenessMin. Single composite condition so tameness
	// binds to the same pet and the id set is an OR.
	if len(req.Pet) > 0 && req.PetTamenessMin > 0 {
		values := make([]int, 0, len(req.Pet))
		for _, id := range req.Pet {
			values = append(values, int(id))
		}
		conditions = append(conditions, ConditionInput{
			Type:     PetTamenessCondition,
			Operator: ">=",
			Value:    int(req.PetTamenessMin),
			Values:   values,
		})
	}
```
(Match each builder's local variable names — read the functions.)

- [ ] **Step 5: Run — expect pass**; add + pass the symmetric `TestBuildEndConditions_PetTameness`.
- [ ] **Step 6: Full module checks** — `GOWORK=off go build ./... && GOWORK=off go vet ./... && GOWORK=off go test -race ./...` → PASS.
- [ ] **Step 7: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-quest
git commit -m "feat(atlas-quest): enforce pet + pettamenessmin quest checks via petTameness"
```

### Task B5: GM command `@award <target> tameness <amount>` (test support)

GM commands live in **atlas-messages**, not atlas-channel: each is an `@`-phrase regexp matcher in
`messages/command/<domain>/commands.go`, GM-gated via `c.Gm()`, targeting `me`/`map`/`<name>`,
returning a `command.Executor`, emitting directly via `producer.ProviderImpl(l)(ctx)(<topic>.EnvCommandTopic)(...)`,
and registered in `messages/main.go`. This command follows the exact `@award <target> <thing> <amount>`
family (`@award me meso 5000`) and **reuses atlas-pets' existing `AWARD_CLOSENESS`** command (additive)
— no atlas-pets change. Reference implementation: `command/character/commands.go` `AwardMesoCommandProducer`;
direct-emit pattern: `command/monster/commands.go` `MobSpawnCommandProducer`.

**Files:**
- Create: `services/atlas-messages/atlas.com/messages/command/pet/commands.go`
- Create: `services/atlas-messages/atlas.com/messages/pet/{processor.go,rest.go,requests.go}` (resolve a character's spawned pet id)
- Create: `services/atlas-messages/atlas.com/messages/kafka/message/pet/{kafka.go,producer.go}` (AWARD_CLOSENESS command to the pet command topic)
- Modify: `services/atlas-messages/atlas.com/messages/main.go` (register the producer)
- Test: `services/atlas-messages/atlas.com/messages/command/pet/commands_test.go`

- [ ] **Step 1: Pet lookup in atlas-messages**

**VERIFIED:** atlas-messages has pet *chat* handling (`message/processor.go` `HandlePet`) but **no pet
data lookup** — add one. Mirror query-aggregator's `pet` REST client: the route is
`GET {PETS}/characters/{characterId}/pets` (`query-aggregator/pet/requests.go:9-20`,
`ByCharacterId = "/characters/%d/" + "pets"`), returning a list with `Id`, `Slot` (spawned = slot ≥ 0).
Expose:
```go
// GetSpawnedPetIds returns the ids of the character's spawned pets (slot >= 0).
func (p *ProcessorImpl) GetSpawnedPetIds(characterId uint32) ([]uint32, error)
```

- [ ] **Step 2: Pet command producer in atlas-messages**

`kafka/message/pet/kafka.go`: the pet command **env topic** const + the command envelope (`PetId`,
`Type`, `Body`) + `AwardClosenessCommandBody{ Amount uint16 }` (match atlas-pets' wire shape:
`CommandAwardCloseness = "AWARD_CLOSENESS"`). `kafka/message/pet/producer.go`:
```go
func AwardClosenessCommandProvider(petId uint32, amount uint16) model.Provider[[]kafka.Message]
```
Mirror `kafka/message/monster` producers in atlas-messages.

- [ ] **Step 3: Write the failing command test**

Mirror `command/character/commands_test.go`:
```go
func TestAwardTamenessCommand_MatchesAndGmGated(t *testing.T) {
	// GM character + message "@award me tameness 2000" → Executor returned, found == true
	// non-GM character → found == false
	// non-matching message "@award me meso 5" → found == false
}
```

- [ ] **Step 4: Run — expect failure** — `cd services/atlas-messages/atlas.com/messages && GOWORK=off go test ./command/pet/...` → FAIL (`AwardTamenessCommandProducer` undefined).

- [ ] **Step 5: Implement `AwardTamenessCommandProducer`** in `command/pet/commands.go`:
  - regexp ``^@award\s+(\w+)\s+tameness\s+(\d+)$``;
  - `if !c.Gm() { return nil, false }`;
  - resolve target character ids (`me`/`map`/`<name>`) exactly as `AwardExperienceCommandProducer` does;
  - Executor: for each target character, `GetSpawnedPetIds`, and for each pet id emit
    `AwardClosenessCommandProvider(petId, uint16(amount))` to the pet command topic.

- [ ] **Step 6: Register** in `messages/main.go`: `command.Registry().Add(pet.AwardTamenessCommandProducer)` (alongside the other `command.Registry().Add(...)` calls ~line 42-51).

- [ ] **Step 7: Module checks** — `cd services/atlas-messages/atlas.com/messages && GOWORK=off go build ./... && GOWORK=off go vet ./... && GOWORK=off go test -race ./...` → PASS.

- [ ] **Step 8: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-messages
git commit -m "feat(atlas-messages): @award <target> tameness <amount> GM command (test support)"
```

### Task B6: Bake the touched services

- [ ] **Step 1: Bake + redis guard**

```bash
cd "$(git rev-parse --show-toplevel)"
docker buildx bake atlas-query-aggregator atlas-quest atlas-messages
tools/redis-key-guard.sh
```
Expected: all targets build (catches missing `COPY libs/...` for atlas-saga); guard clean.

---

## Phase C — Author the Garnox quest conversations

> **BLOCKED** until the Cosmic `q8185s/e.js`, `q8189e.js`, `q4659s/e.js` scripts are obtained.
> Dialogue + flow come from those scripts (Say.img is incomplete — design §2/§4.2). Use
> `/convert-quest` to convert each script to the JSON state-machine format, then augment the end
> machine with the evolution operation chain. Do **not** copy from the inherited atlas conversions.

### Task C1: Obtain the Cosmic scripts

- [ ] **Step 1: Acquire the script files** — `q8185s.js`/`q8185e.js`, `q8189s.js`(if present)/`q8189e.js`, `q4659s.js`/`q4659e.js` from the Cosmic source. Place under `docs/tasks/task-089-pet-evolution/cosmic/` (reference only). If a start script is absent, note it — that quest's start machine is a minimal accept dialogue.
- [ ] **Step 2: Record provenance** — `docs/tasks/task-089-pet-evolution/cosmic/SOURCE.md` noting source repo + commit/URL.

### Task C2: Convert + author `quest-8185.json` (Pet's Evolution2 — baby dragon)

**Files:** Create `deploy/seed/gms/{83_1,84_1,87_1,92_1,95_1,12_1}/npc-conversations/quests/quest-8185.json`

- [ ] **Step 1: Convert** — run `/convert-quest` on `q8185s.js` + `q8185e.js` → quest-conversation JSON (`{data:{attributes:{questId,questName,startStateMachine,endStateMachine},id,type:"quest-conversation"}}`). Use `deploy/seed/gms/83_1/npc-conversations/quests/quest-21200.json` as structural reference only.
- [ ] **Step 2: Identity + end operation chain** — `questId: 8185`, `id: "8185"`. The `endStateMachine`, after dialogue, runs (all saga-backed, order matters):
  1. `local:enumerate_evolvable_pets` → put the summoned baby-dragon (`5000029`) pet id into context (e.g. `selectedPetId`);
  2. `destroy_item` `{ "itemId": "5380000", "quantity": "1" }`;
  3. `evolve_pet` `{ "petId": "{context.selectedPetId}" }`;
  4. `complete_quest` for 8185.
  The `PetEvolution` saga compensation refunds the rock if evolve fails.
- [ ] **Step 3: Validate** — `python3 -m json.tool <file> >/dev/null && echo VALID`; manually verify every `nextState`/outcome target resolves; `startState` exists; `questId`/`id` == 8185; only real state/operation types.
- [ ] **Step 4: Replicate across versions** — copy to each version dir where the quest exists: confirm `grep -l '"8185"' tmp/<uuid>/GMS/<ver>/Quest.wz/Check.img.xml` before seeding there.
- [ ] **Step 5: Commit** — `feat(npc-conversations): Garnox quest 8185 conversation (baby dragon evolution)`

### Task C3: Author `quest-4659.json` (Robo Upgrade — baby robo)

**Files:** Create `deploy/seed/gms/*/npc-conversations/quests/quest-4659.json`

- [ ] **Steps 1–5: Same as C2** except: `questId/id` = 4659; required pet `5000048`; end chain consumes **two** items — `destroy_item 5380000 x1` **and** `destroy_item 4000111 x50` — then `evolve_pet`, then `complete_quest`; convert from `q4659s.js`/`q4659e.js`.
- [ ] **Commit:** `feat(npc-conversations): Garnox quest 4659 conversation (robo upgrade)`

### Task C4: Author `quest-8189.json` (Pet's Re-Evolution — adult re-roll)

**Files:** Create `deploy/seed/gms/*/npc-conversations/quests/quest-8189.json`

- [ ] **Steps 1–5: Same as C2** except: `questId/id` = 8189; required pet ∈ adult set {5000030,5000031,5000032,5000033,5000049,5000050,5000051,5000052}; enumerate resolves whichever adult is summoned; end chain `destroy_item 5380000 x1` → `evolve_pet` → `complete_quest`; convert from `q8189e.js` (start script may be absent → minimal accept dialogue).
- [ ] **Re-evolution repeatability check:** quest 8189 must be retakeable. Confirm atlas-quest treats it as repeatable (WZ `interval`/`autoComplete`; atlas-quest repeat handling) and that `GetStateMachineForCharacter` (errors on COMPLETED) doesn't block a second run. If it does, capture the fix as a follow-up sub-task here (do not silently expand scope).
- [ ] **Commit:** `feat(npc-conversations): Garnox quest 8189 conversation (re-evolution)`

### Task C5: Confirm 8184 needs no conversation

- [ ] **Step 1: Verify** — 8184 is a plain item turn-in (Check: pet 5000029 + tameness 1642 + 50×`4000029` + 50×`4000023`; Act: consume items, award 10×`2120000`). It needs only the Phase B gate. Confirm atlas-quest's standard accept/complete + Act-reward path handles it (item give/take already enforced/processed). No file changes expected; if a gap is found, log a follow-up — do not expand scope.

---

## Phase D — Integration verification on the ephemeral

### Task D1: Full local verification

- [ ] **Step 1: Per-module Go checks** — atlas-quest, atlas-query-aggregator, atlas-npc-conversations, atlas-messages, libs/atlas-saga: `GOWORK=off go test -race ./...`, `go vet ./...`, `go build ./...` — all clean.
- [ ] **Step 2: Bake + redis guard** — `docker buildx bake atlas-quest atlas-query-aggregator atlas-npc-conversations atlas-messages` and `tools/redis-key-guard.sh` → all build; guard clean.
- [ ] **Step 3: UI build** — `cd services/atlas-ui && npm run build` → green, no `pickFromContext` references.

### Task D2: Live ephemeral validation

> The PR ephemeral auto-redeploys on push (deploy-env label). After Phase B/C land:

- [ ] **Step 1: Re-ingest + refresh atlas-data** — trigger the tenant ingest; when complete, `kubectl -n <ns> rollout restart deploy/atlas-data` (clears the REST in-memory cache — confirmed-needed this session).
- [ ] **Step 2: Confirm seeding** — `GET /api/quests/8185/conversation` returns the authored machine; `GET /api/npcs/9102001/conversations` is now **empty** (free conversation gone).
- [ ] **Step 3: In-game smoke test** — as a GM, summon a baby dragon, run `@award me tameness 2000` (Task B5) to push closeness over 1642, hold a Rock, talk to Garnox (9102001) → quest 8185 offers/completes → pet evolves in place (random adult), Rock consumed. With low tameness or no Rock → quest refuses; nothing consumed. Repeat for 4659 (robo) and 8189 (re-evolution).
- [ ] **Step 4: Watch services** (Loki/k8s) — saga `PetEvolution` completes/compensates; no "unhandled message" or validation errors.

---

## Self-Review

- **Spec coverage:** §4.1 → Phase B (B1–B4); §4.2 → Phase C (C2–C4); §4.3 (drop chooser) → Phase A (A2–A3) + C2 step 2 (enumerate-based target); §4.4 backout → Phase A; §4.5 retained → untouched; §2/§5 sourcing → C1. 8184 → C5. GM test command (user request) → B5. Open risks (summoned-pet semantics, 8189 repeatability) → confirm-steps in C4 + B-tests.
- **Placeholder scan:** the Cosmic-dependent dialogue in Phase C is genuinely external (not a pre-fillable placeholder) — flagged as a hard dependency (C1) and gated. All Phase A/B code steps carry real code.
- **Type consistency:** `PetTamenessCondition` is identical across `libs/atlas-saga` (value `"petTameness"`), query-aggregator (`ConditionType` alias of `sharedsaga`), atlas-quest (local wire string). `SpawnedPet{TemplateId, Closeness}`, `SetSpawnedPets`, `MaxPetClosenessForTemplates` consistent B2↔B3. atlas-quest `ConditionInput.Values` (local struct, `model.go:6-13`) ↔ query-aggregator `Condition.values` (`model.go:94`) carry the pet-id set; `Value` carries min closeness. GM command (B5) reuses atlas-pets `AWARD_CLOSENESS` (additive) via a new atlas-messages producer + pet lookup; no atlas-pets change.

## Verification Status (file:line pass, 2026-06-15)

Three Explore agents verified every concrete claim against source. Result:

**Confirmed (grounded):** backout symbols (`model.go:47/65/364/441/464/480`, `rest.go:188/353/364/376/663`, `processor.go:359/512/520/530/544`); `splitCSV` is pickFromContext-only (safe to delete); the 3 pickfromcontext test files exist; conversation ops `evolve_pet`(2528, `petId`, ctx-subst), `enumerate_evolvable_pets`(802, the 3 context keys), `destroy_item`(1512, `itemId`/`quantity`), `complete_quest`(1801, `questId`), `start_quest`(1864, `questId`); **PetEvolution saga refund** (`saga-orchestrator/saga/compensator.go:1044-1140` — DestroyAsset→CreateItem, AwardMesos→negative refund); query-aggregator `petCount`(context.go:37), builder copies (483/505/577 + With* 335/342/371/394/417/440), `Condition.values`(model.go:94), `Evaluate`(385), known-types list(128), `sharedsaga` import(16), `requirementsFor`→Pets(192), fetch site(692), `GetPets`(processor.go:32); atlas-quest const block(model.go:16-25), `ConditionInput.Values`(6-13), `buildStart/EndConditions`(55/221) with `req := questDef.StartRequirements`(56), `Pet`/`PetTamenessMin`(rest.go:62-63); atlas-messages `AwardMesoCommandProducer`(character/commands.go:258), `c.Gm()`(63), me/map/name(69-76), registry(main.go:42-62), monster producer pattern(monster/commands.go:254, kafka/message/monster/kafka.go:14/121), pet route(query-aggregator/pet/requests.go:9-20).

**Corrected after verification (were invented/wrong):**
1. query-aggregator pet `Model` stores only `id`+`slot` and `Extract` drops templateId/closeness (`pet/model.go:4-7`, `rest.go:46-47`) — B2 Step 5 now **extends the Model** (the old code used non-existent `TemplateId()`/`Closeness()` accessors and would not compile).
2. B4 test used an invented `questDefWithRequirements` helper — replaced with the real inline `dataquest.RestModel{StartRequirements: ...}` pattern (`processor_test.go:18-24`).
3. B5 claimed atlas-messages had no pet awareness — it has pet *chat* handling but no pet *data* lookup; corrected to add the data lookup with the verified route.

**Remaining assumptions (verify in execution, not pre-confirmable):** the Cosmic `.js` dialogue/flow (C1 — external, not in repo); 8189 repeatability semantics (C4 confirm-step); whether the `petTameness` check should target summoned-only vs owned (design open question — plan assumes summoned, slot ≥ 0).
