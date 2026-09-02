# Cosmic Map-Action Parity — Plan C: Engine Gaps + Category #2 Conversions

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close capability gaps G1–G14 and land the 39 category-#2 seed documents that
those gaps unblock. Six of the PRD's 45 category-#2 scripts are **not** converted; §0
below records exactly which and why, with the evidence.

**Architecture:** Each gap lands as engine work immediately followed by the seed
documents that exercise it — a gap with no seeded script is unverified code. Work is
ordered cheapest-first: executor arms for saga actions that already exist, then seeds
Plan A already unblocked, then new operations that ride packets the repo already has
verified, then new saga actions with new consumers, then the three-service bulk-mutation
surfaces G5 needs, then the two capabilities that need new server-side persistence.

**Tech Stack:** Go 1.27.0, `libs/atlas-saga` step contracts, `atlas-saga-orchestrator`
handler dispatch, `libs/atlas-packet` clientbound writers (all reused — this plan adds
no new packet), `atlas-query-aggregator` conditions, JSON:API seed documents.

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md)). Per-script literal values and
Cosmic native semantics derived from the Cosmic reference server checkout; the
derivation, its `<cosmic-root>` path, and every `path:line` citation are recorded in
[context.md](context.md).

## Global Constraints

- **[plan.md](plan.md) (Plan A) must be green first**, and [plan-b.md](plan-b.md)
  should land before this plan's seed tasks so `catalog-lint`'s replication check is
  exercised against a full tree.
- **11 version roots, byte-identical**, same list, same authoring-then-`cp` procedure,
  same envelope, same 2-space alphabetically-sorted formatting as Plan B. Re-read
  [plan-b.md](plan-b.md)'s "Global Constraints" before authoring any document here;
  every rule there applies unchanged.
- **Every `spawn_monster` sets `"spawnIfAbsent": "true"`.**
- **Quest-status shift:** Cosmic `NOT_STARTED(0) STARTED(1) COMPLETED(2)` → Atlas
  `NOT_STARTED=1 STARTED=2 COMPLETED=3`. So `isQuestStarted(q)` is
  `questStatus referenceId q = 2`, and `isQuestCompleted(q)` is
  `questStatus referenceId q = 3`.
- **Condition and operation names are whatever `tools/gen-map-action-schema.sh`
  generates.** Every new operation added by this plan must appear in
  `ExecuteOperation`'s switch *before* any document names it, or `catalog-lint` fails —
  which is the intended ordering, not an obstacle.
- **No new packets.** Every wire effect this plan needs already exists in
  `libs/atlas-packet` and is already registered in atlas-channel. Any task that thinks
  it needs a new opcode has hit an unexpected gap: stop and report rather than deriving
  one, because that is packet work under `docs/packets/PROCESS.md` and does not belong
  inside this plan.
- **Verification loop, identical for every seed task:**
  ```bash
  ./tools/gen-map-action-schema.sh --check
  (cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
  git status --short deploy/seed/ | wc -l
  ```
- **Gate:** the flagless `tools/verify.sh` must exit 0 before this plan is done.

---

## §0. The six scripts this plan does not convert, and why

Seven script-facing primitives named in PRD §4.3's G8/G10/G11 rows **do not exist
anywhere in the Cosmic source tree** — an exhaustive grep for each returns zero hits:
`setDirection`, `setDirectionStatus`, `setDirectionMode`, `startDirection`, `lockUI2`,
`sendDirectionInfo` (both the 6-arg and 2-arg overloads), `setStandAloneMode`.

`ms` is a `MapScriptMethods extends AbstractPlayerInteraction`, so each of these is an
unresolved Nashorn method call. `MapScriptManager` catches the resulting
`ScriptException`/`NoSuchMethodException` and aborts `start()` at the first one. The
consequence, per script:

| script | first unresolved call | what Cosmic actually executes | disposition |
|---|---|---|---|
| `cannon_tuto_01` | `setDirection(0)` | nothing | **not converted** |
| `cannon_tuto_direction` | `setDirectionStatus(true)` | nothing — both `showIntro`s die behind it | **not converted** |
| `cannon_tuto_direction1` | `sendDirectionInfo(...)` | `playSound("cannonshooter/flying")` only | converted, Task C8 |
| `cannon_tuto_direction2` | `setDirectionStatus(true)` | `playSound("cannonshooter/bang")` only | converted, Task C8 |
| `Resi_tutor10` | `setStandAloneMode(true)` | nothing | **not converted** |
| `Resi_tutor50` | `setDirectionMode(false)` | nothing — `openNpc(2159006)` dies behind it | **not converted** |
| `Resi_tutor70` | `setDirectionMode(true)` | nothing — `showIntro` dies behind it | **not converted** |
| `Resi_tutor80` | `setDirectionMode(false)` | nothing | **not converted** |

`startDirection("cannon_tuto_02")` additionally references a script that has exactly one
occurrence in the entire Cosmic tree: the call itself. PRD FR-3.1 asks for this to be
"located in WZ or dropped with rationale" — it is dropped, and this table is the
rationale.

Porting the seven primitives means deriving eight unknown clientbound packets from the
client binary with no fname, no opcode and no reference implementation to check against.
That is out of scope for this plan by decision; it stays on issue #1624 alongside
category #3.

**39 of the 45 category-#2 scripts are converted here.** 6 are not.

---

## §0.1 Corrections to the PRD's gap table

Four rows of PRD §4.3 are wrong against the Cosmic source. Each is corrected in the task
that owns it; they are collected here so a reviewer can check them in one place.

- **`count_monster` (G5) is not a capability.** It appears once across all 45 scripts —
  `map.countMonster(9100013) == 0` guarding a spawn in `926000000` — which is exactly
  `spawnIfAbsent`. Plan A already shipped it. No condition, no operation, no task.
- **`resetPQ(n)` (G5) is a field reset, not a party-quest reset.**
  `MapleMap.resetPQ(difficulty)` → `resetMapObjects(difficulty, true)` →
  `clearMapObjects()` + `restoreMapSpawnPoints()` + `instanceMapFirstSpawn(difficulty, true)`.
  The owner is the field, not `atlas-party-quests`. See Task C18.
- **`teachSkill(id, -1, 0, -1)` (G13) removes the skill; it does not teach it.**
  `AbstractPlayerInteraction.teachSkill` falls through to
  `Character.changeSkillLevel(skill, -1, 0, -1)`, whose `newLevel > -1` branch is false,
  so it runs `skills.remove(skill)`, `DELETE FROM skills WHERE skillid = ? AND characterid = ?`
  and sends `updateSkill(id, -1, 0, -1)`. `iceCave` therefore **clears** the five Aran
  tutorial skills on entry. See Task C4.
- **`isCygnus()` needs no `in` operator.** It is `job.getId() / 1000 == 1`, i.e. the
  inclusive band 1000–1999, expressible as two AND-ed conditions. See Task C5.

---

## §0.2 Wire-contract note for `areaInfo` (Task C20)

`containsAreaInfo(area, "k=v;...")` is a **substring** test against a free-form string.
`saga.ValidationConditionInput` has only `Value int` and `Values []int`, so there is no
field that can carry the operand.

Task C20 therefore adds `ValueString string \`json:"valueString,omitempty"\`` to
`libs/atlas-saga/validation.go`'s `ValidationConditionInput`. The addition is purely
additive — `omitempty`, no existing consumer reads it — and `areaInfo` is its first and
only user. This differs from the decision taken for `transportState` (Task C10), which
was expressible as a boolean condition and therefore did **not** widen the wire. It is
called out here because it is the one contract widening in this plan and a reviewer
should agree with it before Task C20 runs.

---

## Task C1: `set_quest_progress` and `start_quest` executor arms (G3, G4)

`SetQuestProgress` (`libs/atlas-saga/model.go:148`) and `StartQuest`
(`libs/atlas-saga/model.go:147`) already exist end to end: payloads at
`libs/atlas-saga/payloads.go:381-387` and `payloads.go:372-378`, orchestrator handler
arms `handleSetQuestProgress` (`saga/handler.go:2202`) and `handleStartQuest`
(`saga/handler.go:2165`), both dispatching to the `quest` domain client. Only the
map-actions executor arms are missing.

### Files

- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — two new switch cases and two new methods
- `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — extend (created in Plan A Task A4)
- `services/atlas-map-actions/docs/map_script_schema.json` — the two new `allOf` param blocks (the operation enum itself is generated)
- `libs/atlas-saga/payloads.go` — read-only; the two payload structs
- Seed documents under all 11 roots at `map-actions/onUserEnter/`: `map-130030000.json`, `map-130030001.json`, `map-914000100.json`, `map-babyPigMap.json` — **new files**, 44 total

Module root: `services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `executeSpawnMonster` (`script/executor.go:120`) for the
read-params → validate → build one-step saga → `e.sagaP.Create(s)` shape, including the
`saga.NewBuilder().SetSagaType(saga.InventoryTransaction).SetInitiatedBy(...).AddStep(...)`
call. Use a distinct `SetInitiatedBy` string per arm, matching the existing convention
(`"map-action-spawn"`, `"ui-lock"`).

**Interfaces:**
- Consumes: `saga.SetQuestProgressPayload{CharacterId uint32, WorldId world.Id, QuestId uint32, InfoNumber uint32, Progress string}`; `saga.StartQuestPayload{CharacterId uint32, WorldId world.Id, QuestId uint32, NpcId uint32, Rewards []QuestRewardItem}`.
- Produces: two document operations. `set_quest_progress` with required params `questId`, `infoNumber`, `progress`. `start_quest` with required param `questId` and optional `npcId`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go`,
reusing the recorder and field fixtures already established there in Plan A Task A4.
Field: `field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(130030000)).Build()`,
characterId `1`.

**`TestExecuteSetQuestProgress`** — params `{"questId":"20010","infoNumber":"20022","progress":"1"}`.
Assert the single captured saga has one step whose action is `saga.SetQuestProgress` and
whose payload, type-asserted to `saga.SetQuestProgressPayload`, equals:

```go
saga.SetQuestProgressPayload{
	CharacterId: 1,
	WorldId:     world.Id(0),
	QuestId:     20010,
	InfoNumber:  20022,
	Progress:    "1",
}
```

**`TestExecuteSetQuestProgressParamValidation`** — table-driven:

| subtest | params | expect error message contains |
|---|---|---|
| `missing questId` | `{"infoNumber":"1","progress":"1"}` | `set_quest_progress operation missing questId parameter` |
| `missing infoNumber` | `{"questId":"1","progress":"1"}` | `set_quest_progress operation missing infoNumber parameter` |
| `missing progress` | `{"questId":"1","infoNumber":"1"}` | `set_quest_progress operation missing progress parameter` |
| `bad questId` | `{"questId":"x","infoNumber":"1","progress":"1"}` | `invalid questId [x]` |
| `bad infoNumber` | `{"questId":"1","infoNumber":"x","progress":"1"}` | `invalid infoNumber [x]` |

`progress` is **not** parsed as an integer — `SetQuestProgressPayload.Progress` is a
`string`, and Cosmic's own progress values are zero-padded strings in some quests. Pass
it through verbatim.

**`TestExecuteStartQuest`** — params `{"questId":"22015","npcId":"9010000"}`. Assert the
payload equals:

```go
saga.StartQuestPayload{
	CharacterId: 1,
	WorldId:     world.Id(0),
	QuestId:     22015,
	NpcId:       9010000,
}
```

with `Rewards` nil.

**`TestExecuteStartQuestDefaultsNpcIdToZero`** — params `{"questId":"22015"}`. Assert
`payload.NpcId == 0`.

**`TestExecuteStartQuestParamValidation`**:

| subtest | params | expect error message contains |
|---|---|---|
| `missing questId` | `{}` | `start_quest operation missing questId parameter` |
| `bad questId` | `{"questId":"x"}` | `invalid questId [x]` |
| `bad npcId` | `{"questId":"1","npcId":"x"}` | `invalid npcId [x]` |

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run 'TestExecuteSetQuestProgress|TestExecuteStartQuest' -v`
Expected: FAIL with `unknown operation type [set_quest_progress]` and
`unknown operation type [start_quest]` — the Plan A Task A4 default arm.

- [ ] **Step 3: Add the two switch cases**

In `ExecuteOperation` (`script/executor.go:36-52`), before the `default:` arm:

```go
	case "set_quest_progress":
		return e.executeSetQuestProgress(f, characterId, op)
	case "start_quest":
		return e.executeStartQuest(f, characterId, op)
```

- [ ] **Step 4: Implement the two methods**

```go
func (e *OperationExecutor) executeSetQuestProgress(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	questIdStr, ok := params["questId"]
	if !ok {
		return fmt.Errorf("set_quest_progress operation missing questId parameter")
	}
	questId, err := strconv.ParseUint(questIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid questId [%s]: %w", questIdStr, err)
	}

	infoNumberStr, ok := params["infoNumber"]
	if !ok {
		return fmt.Errorf("set_quest_progress operation missing infoNumber parameter")
	}
	infoNumber, err := strconv.ParseUint(infoNumberStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid infoNumber [%s]: %w", infoNumberStr, err)
	}

	progress, ok := params["progress"]
	if !ok {
		return fmt.Errorf("set_quest_progress operation missing progress parameter")
	}

	e.l.Debugf("Setting quest [%d] progress [%d]=[%s] for character [%d].", questId, infoNumber, progress, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-quest-progress").
		AddStep(
			fmt.Sprintf("quest-progress-%d-%d", characterId, questId),
			saga.Pending,
			saga.SetQuestProgress,
			saga.SetQuestProgressPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				QuestId:     uint32(questId),
				InfoNumber:  uint32(infoNumber),
				Progress:    progress,
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeStartQuest(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	questIdStr, ok := params["questId"]
	if !ok {
		return fmt.Errorf("start_quest operation missing questId parameter")
	}
	questId, err := strconv.ParseUint(questIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid questId [%s]: %w", questIdStr, err)
	}

	var npcId uint64
	if npcIdStr, has := params["npcId"]; has {
		npcId, err = strconv.ParseUint(npcIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid npcId [%s]: %w", npcIdStr, err)
		}
	}

	e.l.Debugf("Force-starting quest [%d] for character [%d].", questId, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-start-quest").
		AddStep(
			fmt.Sprintf("start-quest-%d-%d", characterId, questId),
			saga.Pending,
			saga.StartQuest,
			saga.StartQuestPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				QuestId:     uint32(questId),
				NpcId:       uint32(npcId),
			},
		).Build()

	return e.sagaP.Create(s)
}
```

- [ ] **Step 5: Add the two schema `allOf` blocks**

In `services/atlas-map-actions/docs/map_script_schema.json`'s
`definitions.operation.allOf` array, append two entries in the same shape as the
existing `field_effect` block (`map_script_schema.json:120-138`):

```json
        {
          "if": {
            "properties": { "type": { "const": "set_quest_progress" } }
          },
          "then": {
            "properties": {
              "params": {
                "type": "object",
                "required": ["questId", "infoNumber", "progress"],
                "properties": {
                  "questId": { "type": "string", "description": "Quest ID whose progress is being set" },
                  "infoNumber": { "type": "string", "description": "Quest info-number sub-key" },
                  "progress": { "type": "string", "description": "Progress value, passed through verbatim" }
                }
              }
            }
          }
        },
        {
          "if": {
            "properties": { "type": { "const": "start_quest" } }
          },
          "then": {
            "properties": {
              "params": {
                "type": "object",
                "required": ["questId"],
                "properties": {
                  "questId": { "type": "string", "description": "Quest ID to force-start" },
                  "npcId": { "type": "string", "description": "NPC credited with starting the quest (default 0)" }
                }
              }
            }
          }
        }
```

Then run `./tools/gen-map-action-schema.sh` so the generated operation enum picks up the
two new switch cases, and confirm `./tools/gen-map-action-schema.sh --check` exits 0.

- [ ] **Step 6: Run the tests**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 7: Author the four seed documents under `gms/83_1`**

`map-130030000.json` — Cosmic body is the single call `ms.setQuestProgress(20010, 20022, 1)`:

```json
{
  "data": {
    "attributes": {
      "description": "Map 130030000 - records quest 20010 progress 20022=1 on entry",
      "rules": [
        {
          "conditions": [],
          "id": "set_progress",
          "operations": [
            {
              "params": {
                "infoNumber": "20022",
                "progress": "1",
                "questId": "20010"
              },
              "type": "set_quest_progress"
            }
          ]
        }
      ],
      "scriptName": "130030000"
    },
    "id": "130030000",
    "type": "map-action"
  }
}
```

`map-130030001.json` — Cosmic's body is byte-identical to `130030000`'s, same
`(20010, 20022, 1)`. Copy the document and change `description`, `scriptName` and `id`
to `130030001`.

`map-914000100.json` — same shape, `(21000, 21002, 1)`:

| param | value |
|---|---|
| `questId` | `21000` |
| `infoNumber` | `21002` |
| `progress` | `1` |

description: `Map 914000100 - records quest 21000 progress 21002=1 on entry`.

`map-babyPigMap.json` — Cosmic body is `ms.unlockUI()` then
`ms.getClient().getQM().forceStartQuest(22015)`. `AbstractPlayerInteraction.forceStartQuest(id)`
credits `NpcId.MAPLE_ADMINISTRATOR`, which is `9010000`
(`<cosmic-root>/src/main/java/constants/id/NpcId.java:6`). Operation order matches
Cosmic: `unlock_ui` first.

```json
{
  "data": {
    "attributes": {
      "description": "Baby pig map - unlocks the UI and force-starts quest 22015",
      "rules": [
        {
          "conditions": [],
          "id": "unlock_and_start",
          "operations": [
            {
              "type": "unlock_ui"
            },
            {
              "params": {
                "npcId": "9010000",
                "questId": "22015"
              },
              "type": "start_quest"
            }
          ]
        }
      ],
      "scriptName": "babyPigMap"
    },
    "id": "babyPigMap",
    "type": "map-action"
  }
}
```

- [ ] **Step 8: Replicate and verify**

```bash
for f in 130030000 130030001 914000100 babyPigMap; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `44`.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-map-actions/ deploy/seed/
git commit -m "feat(map-actions): set_quest_progress and start_quest operations (G3, G4)"
```

---

## Task C2: `open_npc` executor arm (G9)

`StartNpcConversation` (`libs/atlas-saga/model.go:205`) exists end to end; the handler
`handleStartNpcConversation` (`saga/handler.go:1402`) is deliberately **not**
self-completing — it emits to `npc.EnvCommandTopic` and leaves the step Pending until
the conversation-status consumer reports STARTED or START_ERROR. That is the correct
behavior for a map-entry conversation and needs no change.

Two of G9's three scripts convert. `Resi_tutor50` does not — see §0.

### Files

- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — one new switch case and method
- `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — extend
- `services/atlas-map-actions/docs/map_script_schema.json` — one new `allOf` block
- Seed documents under all 11 roots: `map-Resi_tutor40.json`, `map-Resi_tutor60.json` — **new files**, 22 total

Module root: `services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `executeSpawnMonster` (`script/executor.go:120`) for the arm shape, and
Task C1's two methods for the params-validation wording.

**Interfaces:**
- Consumes: `saga.StartNpcConversationPayload{CharacterId uint32, AccountId uint32, NpcTemplateId uint32, WorldId world.Id, ChannelId channel.Id, MapId _map.Id, Instance uuid.UUID}`.
- Produces: the `open_npc` operation with required param `npcId`.

- [ ] **Step 1: Resolve `AccountId`**

`StartNpcConversationPayload` carries an `AccountId` that `executeSpawnMonster` has no
analogue for. Determine where it comes from:

Run:
```bash
grep -rn "characterId" services/atlas-map-actions/atlas.com/map-actions/script/consumer.go
sed -n '1402,1425p' services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go
```

The first command shows what the map-entry command actually carries. A repo-wide search
of `services/atlas-map-actions/` at plan time found `characterId` but **no** account id
of any spelling, so the command almost certainly does not carry one. Confirm that, then read `handleStartNpcConversation` (`saga/handler.go:1402-1420`) to see
whether `AccountId` is read on the way to the NPC command. If the handler ignores it,
pass `0` with a comment recording that; if it does read it, the account id must be
threaded into `ExecuteOperation` the way `characterId` already is, which is a change to
`script/consumer.go`'s handler signature as well. Record which you found in the commit
body; do not guess.

- [ ] **Step 2: Write the failing test**

Append to `script/executor_test.go`. Field:
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(931000400)).SetInstance(inst).Build()`
with `inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")`.

**`TestExecuteOpenNpc`** — params `{"npcId":"2159012"}`. Assert the payload equals:

```go
saga.StartNpcConversationPayload{
	CharacterId:   1,
	AccountId:     0,
	NpcTemplateId: 2159012,
	WorldId:       world.Id(0),
	ChannelId:     channel.Id(1),
	MapId:         _map.Id(931000400),
	Instance:      inst,
}
```

(Substitute the real `AccountId` if Step 1 found one.)

**`TestExecuteOpenNpcParamValidation`**:

| subtest | params | expect error message contains |
|---|---|---|
| `missing npcId` | `{}` | `open_npc operation missing npcId parameter` |
| `bad npcId` | `{"npcId":"x"}` | `invalid npcId [x]` |

- [ ] **Step 3: Run the test and confirm it fails**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run TestExecuteOpenNpc -v`
Expected: FAIL with `unknown operation type [open_npc]`.

- [ ] **Step 4: Implement the arm**

Add `case "open_npc": return e.executeOpenNpc(f, characterId, op)` to the switch, and:

```go
func (e *OperationExecutor) executeOpenNpc(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	npcIdStr, ok := params["npcId"]
	if !ok {
		return fmt.Errorf("open_npc operation missing npcId parameter")
	}
	npcId, err := strconv.ParseUint(npcIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid npcId [%s]: %w", npcIdStr, err)
	}

	e.l.Debugf("Opening NPC [%d] conversation for character [%d].", npcId, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-open-npc").
		AddStep(
			fmt.Sprintf("open-npc-%d-%d", characterId, npcId),
			saga.Pending,
			saga.StartNpcConversation,
			saga.StartNpcConversationPayload{
				CharacterId:   characterId,
				NpcTemplateId: uint32(npcId),
				WorldId:       f.WorldId(),
				ChannelId:     f.ChannelId(),
				MapId:         f.MapId(),
				Instance:      f.Instance(),
			},
		).Build()

	return e.sagaP.Create(s)
}
```

- [ ] **Step 5: Add the schema block and regenerate**

Append to `definitions.operation.allOf`:

```json
        {
          "if": {
            "properties": { "type": { "const": "open_npc" } }
          },
          "then": {
            "properties": {
              "params": {
                "type": "object",
                "required": ["npcId"],
                "properties": {
                  "npcId": { "type": "string", "description": "NPC template ID whose conversation the server starts" }
                }
              }
            }
          }
        }
```

Run `./tools/gen-map-action-schema.sh` then `./tools/gen-map-action-schema.sh --check`.

- [ ] **Step 6: Run the tests**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 7: Author the two seed documents**

`map-Resi_tutor40.json` (Cosmic body: `ms.openNpc(2159012)`) and
`map-Resi_tutor60.json` (`ms.openNpc(2159007)`):

```json
{
  "data": {
    "attributes": {
      "description": "Resistance tutorial - opens NPC <npcId> conversation on entry",
      "rules": [
        {
          "conditions": [],
          "id": "open_npc",
          "operations": [
            {
              "params": {
                "npcId": "<npcId>"
              },
              "type": "open_npc"
            }
          ]
        }
      ],
      "scriptName": "<scriptName>"
    },
    "id": "<scriptName>",
    "type": "map-action"
  }
}
```

| scriptName | npcId |
|---|---|
| `Resi_tutor40` | `2159012` |
| `Resi_tutor60` | `2159007` |

- [ ] **Step 8: Replicate and verify**

```bash
for f in Resi_tutor40 Resi_tutor60; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `22`.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-map-actions/ deploy/seed/
git commit -m "feat(map-actions): open_npc operation (G9); Resi_tutor50 not converted per plan-c §0"
```

---

## Task C3: `show_info` and `update_area_info` executor arms (G12, partial)

`ShowInfo` (`libs/atlas-saga/model.go:163`) and `UpdateAreaInfo`
(`libs/atlas-saga/model.go:162`) exist end to end. Both are synchronous handlers
(`handleShowInfo` at `saga/handler.go:2835`, `handleUpdateAreaInfo` at
`saga/handler.go:2879`) dispatching through `system_message.Processor` to atlas-channel,
which writes them via `CharacterStatusMessageWriter`.

This task lands the two arms and the one G12 script that needs no server-side area-info
*read*: `Resi_tutor30`. `rienArrow` and `rien` both gate on `containsAreaInfo` and wait
for Task C20.

### Files

- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — two new switch cases and methods
- `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — extend
- `services/atlas-map-actions/docs/map_script_schema.json` — two new `allOf` blocks
- Seed document under all 11 roots: `map-Resi_tutor30.json` — **new file**, 11 total

Module root: `services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: Task C1's `executeSetQuestProgress` for the arm shape.

**Interfaces:**
- Consumes: `saga.UpdateAreaInfoPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, Area uint16, Info string}`; `saga.ShowInfoPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, Path string}`.
- Produces: `update_area_info` with required params `area`, `info`; `show_info` with required param `path`.

- [ ] **Step 1: Write the failing tests**

Append to `script/executor_test.go`. Field:
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(931000400)).Build()`.

**`TestExecuteUpdateAreaInfo`** — params
`{"area":"23007","info":"exp1=1;exp2=1;exp3=1;exp4=1"}`. Assert the payload equals:

```go
saga.UpdateAreaInfoPayload{
	CharacterId: 1,
	WorldId:     world.Id(0),
	ChannelId:   channel.Id(1),
	Area:        23007,
	Info:        "exp1=1;exp2=1;exp3=1;exp4=1",
}
```

`Area` is a `uint16`, so parse with `strconv.ParseUint(s, 10, 16)`.

**`TestExecuteShowInfo`** — params
`{"path":"Effect/OnUserEff.img/guideEffect/resistanceTutorial/userTalk"}`. Assert:

```go
saga.ShowInfoPayload{
	CharacterId: 1,
	WorldId:     world.Id(0),
	ChannelId:   channel.Id(1),
	Path:        "Effect/OnUserEff.img/guideEffect/resistanceTutorial/userTalk",
}
```

**`TestExecuteAreaInfoParamValidation`**:

| subtest | operation | params | expect error message contains |
|---|---|---|---|
| `missing area` | `update_area_info` | `{"info":"a=1"}` | `update_area_info operation missing area parameter` |
| `missing info` | `update_area_info` | `{"area":"1"}` | `update_area_info operation missing info parameter` |
| `bad area` | `update_area_info` | `{"area":"x","info":"a=1"}` | `invalid area [x]` |
| `area overflows uint16` | `update_area_info` | `{"area":"70000","info":"a=1"}` | `invalid area [70000]` |
| `missing path` | `show_info` | `{}` | `show_info operation missing path parameter` |

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run 'TestExecuteUpdateAreaInfo|TestExecuteShowInfo|TestExecuteAreaInfoParam' -v`
Expected: FAIL with `unknown operation type [update_area_info]` and
`unknown operation type [show_info]`.

- [ ] **Step 3: Implement the two arms**

Add to the switch:

```go
	case "update_area_info":
		return e.executeUpdateAreaInfo(f, characterId, op)
	case "show_info":
		return e.executeShowInfo(f, characterId, op)
```

```go
func (e *OperationExecutor) executeUpdateAreaInfo(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	areaStr, ok := params["area"]
	if !ok {
		return fmt.Errorf("update_area_info operation missing area parameter")
	}
	area, err := strconv.ParseUint(areaStr, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid area [%s]: %w", areaStr, err)
	}

	info, ok := params["info"]
	if !ok {
		return fmt.Errorf("update_area_info operation missing info parameter")
	}

	e.l.Debugf("Updating area info [%d] for character [%d].", area, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-area-info").
		AddStep(
			fmt.Sprintf("area-info-%d-%d", characterId, area),
			saga.Pending,
			saga.UpdateAreaInfo,
			saga.UpdateAreaInfoPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Area:        uint16(area),
				Info:        info,
			},
		).Build()

	return e.sagaP.Create(s)
}

func (e *OperationExecutor) executeShowInfo(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	path, ok := params["path"]
	if !ok {
		return fmt.Errorf("show_info operation missing path parameter")
	}

	e.l.Debugf("Showing info [%s] for character [%d].", path, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-show-info").
		AddStep(
			fmt.Sprintf("show-info-%d", characterId),
			saga.Pending,
			saga.ShowInfo,
			saga.ShowInfoPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				ChannelId:   f.ChannelId(),
				Path:        path,
			},
		).Build()

	return e.sagaP.Create(s)
}
```

- [ ] **Step 4: Add the schema blocks and regenerate**

Append two `allOf` entries: `update_area_info` requiring `area` and `info`,
`show_info` requiring `path`. Descriptions: `"Area/info number (questId in the protocol)"`,
`"Semicolon-delimited k=v info string"`, `"OnUserEff path (e.g. Effect/OnUserEff.img/guideEffect/...)"`.

Run `./tools/gen-map-action-schema.sh` then `--check`.

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 6: Author `map-Resi_tutor30.json`**

Cosmic body, in order: `ms.updateAreaInfo(23007, "exp1=1;exp2=1;exp3=1;exp4=1")` then
`ms.showInfo("Effect/OnUserEff.img/guideEffect/resistanceTutorial/userTalk")`.

```json
{
  "data": {
    "attributes": {
      "description": "Resistance tutorial - forces area info 23007 and shows the userTalk guide effect",
      "rules": [
        {
          "conditions": [],
          "id": "force_area_info",
          "operations": [
            {
              "params": {
                "area": "23007",
                "info": "exp1=1;exp2=1;exp3=1;exp4=1"
              },
              "type": "update_area_info"
            },
            {
              "params": {
                "path": "Effect/OnUserEff.img/guideEffect/resistanceTutorial/userTalk"
              },
              "type": "show_info"
            }
          ]
        }
      ],
      "scriptName": "Resi_tutor30"
    },
    "id": "Resi_tutor30",
    "type": "map-action"
  }
}
```

- [ ] **Step 7: Replicate and verify**

```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  cp deploy/seed/gms/83_1/map-actions/onUserEnter/map-Resi_tutor30.json \
     "deploy/seed/$r/map-actions/onUserEnter/map-Resi_tutor30.json"
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `11`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-map-actions/ deploy/seed/
git commit -m "feat(map-actions): update_area_info and show_info operations (G12 partial)"
```

---

## Task C4: `clear_skill` executor arm and `iceCave` (G13)

**PRD correction.** PRD §4.5 describes `iceCave` as "teach 20000014–20000018". It does
the opposite. `ms.teachSkill(id, -1, 0, -1)` reaches
`Character.changeSkillLevel(skill, -1, 0, -1)`
(`<cosmic-root>/src/main/java/client/Character.java:1815-1833`), whose `newLevel > -1`
branch is false, so it runs `skills.remove(skill)`, deletes the row, and sends
`updateSkill(id, -1, 0, -1)`. `iceCave` **clears** the five Aran tutorial skills on
entry.

`libs/atlas-saga` has `CreateSkill` and `UpdateSkill` (`model.go:142-143`) but the survey
found no removal action. This task determines whether one exists under another name and,
if not, adds one.

### Files

- `libs/atlas-saga/model.go` — a new `Action` constant, if none exists
- `libs/atlas-saga/payloads.go` — a new payload, if none exists
- `libs/atlas-saga/unmarshal.go` — the matching case
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — type alias + unmarshal case
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — the accepted-action entry
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — interface decl, dispatch case, handler body
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor.go` — the payload case
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/skill/processor.go` — the domain client method
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — one new switch case and method
- `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — extend
- `services/atlas-map-actions/docs/map_script_schema.json` — one new `allOf` block
- Seed document under all 11 roots: `map-iceCave.json` — **new file**, 11 total

Module roots: `libs/atlas-saga`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, and
`services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: the 13-touchpoint `SpawnMonster` checklist recorded in
[context.md](context.md) §"Saga action scaffolding" — it names every file and line a new
action must touch. `UpdateSkill`'s existing chain (`saga/handler.go:1600` →
`skill/processor.go:66`) is the nearest sibling; copy its shape.

**Interfaces:**
- Produces (if none exists): `saga.ClearSkill Action = "clear_skill"` and
  `saga.ClearSkillPayload{CharacterId uint32, SkillId uint32}`; the `clear_skill`
  document operation with required param `skillId`.

- [ ] **Step 1: Check for an existing removal action first**

Run:
```bash
grep -n "Skill" libs/atlas-saga/model.go
grep -rn "Skill" services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/skill/processor.go
grep -rn "func.*Skill" services/atlas-skills/atlas.com/skills/skill/processor.go | head -20
grep -rn "Delete\|Remove\|Clear" services/atlas-skills/atlas.com/skills/ --include='*.go' | grep -i skill | head -20
```

If `libs/atlas-saga` already has a skill-removal action, use it and skip Steps 3-6. If
atlas-skills has no removal path at all, that is a genuine gap: report it and add the
removal to atlas-skills as part of this task, with its own test in that service.
Record which of the two you found in the commit body.

- [ ] **Step 2: Write the failing map-actions test**

Append to `script/executor_test.go`. Field:
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(914000000)).Build()`.

**`TestExecuteClearSkill`** — params `{"skillId":"20000014"}`. Assert the single captured
saga has one step with action `saga.ClearSkill` and payload
`saga.ClearSkillPayload{CharacterId: 1, SkillId: 20000014}`.

**`TestExecuteClearSkillParamValidation`**:

| subtest | params | expect error message contains |
|---|---|---|
| `missing skillId` | `{}` | `clear_skill operation missing skillId parameter` |
| `bad skillId` | `{"skillId":"x"}` | `invalid skillId [x]` |

- [ ] **Step 3: Add the saga action and payload**

In `libs/atlas-saga/model.go`, beside `CreateSkill`/`UpdateSkill` (lines 142-143):

```go
	// ClearSkill removes a skill from a character outright — the Cosmic
	// teachSkill(id, -1, ...) path, which deletes the skill row rather than
	// changing its level (task-290 G13).
	ClearSkill Action = "clear_skill"
```

In `libs/atlas-saga/payloads.go`, beside `UpdateSkillPayload` (lines 297-303):

```go
// ClearSkillPayload represents the payload required to remove a skill entirely.
type ClearSkillPayload struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
}
```

Add the matching `case ClearSkill:` to `libs/atlas-saga/unmarshal.go`, alongside the
existing skill cases.

- [ ] **Step 4: Add a payload round-trip test**

Append to `libs/atlas-saga/unmarshal_test.go`, copying the shape of
`TestUnmarshalEvolvePetStep` (`libs/atlas-saga/unmarshal_test.go:253`):

**`TestUnmarshalClearSkillStep`** — unmarshal a `Step[any]` whose JSON is
`{"stepId":"clear-1-20000014","status":"pending","action":"clear_skill","payload":{"characterId":1,"skillId":20000014}}`
and assert the payload type-asserts to `ClearSkillPayload{CharacterId: 1, SkillId: 20000014}`.

Run: `cd libs/atlas-saga && go test ./... -run TestUnmarshalClearSkillStep -v` — first
FAIL, then PASS after Step 3.

- [ ] **Step 5: Wire the orchestrator's seven touchpoints**

Following the `SpawnMonster` checklist in [context.md](context.md):

1. `saga/model.go` — `ClearSkill = sharedsaga.ClearSkill` alias beside the other skill aliases
2. `saga/model.go` — `ClearSkillPayload = sharedsaga.ClearSkillPayload` alias
3. `saga/model.go` — `case ClearSkill:` in the step-payload unmarshal switch (near line 1251)
4. `saga/event_acceptance.go` — `sharedsaga.ClearSkill: {}` in the accepted set (near line 346)
5. `saga/handler.go` — `handleClearSkill(s Saga, st Step[any]) error` in the interface (near line 138)
6. `saga/handler.go` — `case ClearSkill: return h.handleClearSkill, true` in the dispatch switch (near line 919)
7. `saga/character_extractor.go` — `case ClearSkillPayload:` returning `p.CharacterId` (near line 47)

Handler body, copying `handleUpdateSkill` (`saga/handler.go:1600`):

```go
// handleClearSkill handles the ClearSkill action
func (h *HandlerImpl) handleClearSkill(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ClearSkillPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	if err := h.skillP.RequestClearAndEmit(payload.CharacterId, payload.SkillId); err != nil {
		h.logActionError(s, st, err, fmt.Sprintf("Failed to clear skill %d", payload.SkillId))
		return err
	}
	return nil
}
```

Add the removal method to `skill.Processor`
(`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/skill/processor.go`),
declared as:

```go
// Processor gains one method. Copy RequestUpdateAndEmit (skill/processor.go:66)
// and emit whichever removal command atlas-skills accepts, per Step 1's finding.
type Processor interface {
	RequestClearAndEmit(characterId uint32, skillId uint32) error
}

func (p *ProcessorImpl) RequestClearAndEmit(characterId uint32, skillId uint32) error
```

- [ ] **Step 6: Add the event-acceptance assertion**

`saga/event_acceptance_test.go:30` already asserts the accepted-actions set. Extend that
assertion to include `sharedsaga.ClearSkill` and run:

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... -run TestAccepted -v`
Expected: PASS.

- [ ] **Step 7: Implement the map-actions arm**

Add `case "clear_skill": return e.executeClearSkill(f, characterId, op)` and:

```go
func (e *OperationExecutor) executeClearSkill(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	skillIdStr, ok := params["skillId"]
	if !ok {
		return fmt.Errorf("clear_skill operation missing skillId parameter")
	}
	skillId, err := strconv.ParseUint(skillIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid skillId [%s]: %w", skillIdStr, err)
	}

	e.l.Debugf("Clearing skill [%d] for character [%d].", skillId, characterId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-clear-skill").
		AddStep(
			fmt.Sprintf("clear-skill-%d-%d", characterId, skillId),
			saga.Pending,
			saga.ClearSkill,
			saga.ClearSkillPayload{
				CharacterId: characterId,
				SkillId:     uint32(skillId),
			},
		).Build()

	return e.sagaP.Create(s)
}
```

Add the schema `allOf` block (`clear_skill` requiring `skillId`), regenerate, `--check`.

- [ ] **Step 8: Author `map-iceCave.json`**

Cosmic order: five `teachSkill(id, -1, 0, -1)` calls, then `ms.unlockUI()`, then
`ms.showIntro("Effect/Direction1.img/aranTutorial/ClickLilin")`. Seven operations, one
rule, in that exact order.

```json
{
  "data": {
    "attributes": {
      "description": "Ice cave - clears the five Aran tutorial skills, unlocks the UI and plays the ClickLilin intro",
      "rules": [
        {
          "conditions": [],
          "id": "clear_tutorial_skills",
          "operations": [
            { "params": { "skillId": "20000014" }, "type": "clear_skill" },
            { "params": { "skillId": "20000015" }, "type": "clear_skill" },
            { "params": { "skillId": "20000016" }, "type": "clear_skill" },
            { "params": { "skillId": "20000017" }, "type": "clear_skill" },
            { "params": { "skillId": "20000018" }, "type": "clear_skill" },
            { "type": "unlock_ui" },
            {
              "params": {
                "path": "Effect/Direction1.img/aranTutorial/ClickLilin"
              },
              "type": "show_intro"
            }
          ]
        }
      ],
      "scriptName": "iceCave"
    },
    "id": "iceCave",
    "type": "map-action"
  }
}
```

Re-indent the five `clear_skill` operations to the file's 2-space multi-line style
rather than the single-line form shown above, matching every other seed document.

- [ ] **Step 9: Replicate, verify, commit**

```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  cp deploy/seed/gms/83_1/map-actions/onUserEnter/map-iceCave.json \
     "deploy/seed/$r/map-actions/onUserEnter/map-iceCave.json"
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
cd libs/atlas-saga && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
git add libs/atlas-saga/ services/ deploy/seed/
git commit -m "feat(saga): clear_skill action; iceCave clears Aran tutorial skills (G13)

Cosmic teachSkill(id, -1, 0, -1) reaches changeSkillLevel's newLevel <= -1
branch, which removes the skill and deletes its row. PRD 4.5 described this
as 'teach'; it is a clear."
```

---

*(Tasks C5–C15 continue in [plan-c2.md](plan-c2.md); tasks C16–C23 and the Plan C
completion gate are in [plan-c3.md](plan-c3.md).)*
