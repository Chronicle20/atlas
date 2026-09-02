# Cosmic Map-Action Parity — Plan C, Part 2 (Tasks C5–C15)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Continue Plan C. This file holds the seeds Plan A already unblocked (C5–C6),
the three new operations that ride packets the repo has already verified (C7, C9), the
`transportInTransit` aggregator condition (C10), and the G1/G2 warp and NPC-spawn work
(C12–C15).

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md)). Read
**[plan-c.md](plan-c.md) first** — its Global Constraints, its §0 (the six scripts not
converted), §0.1 (four PRD corrections) and §0.2 (the one wire widening) all apply
unchanged here and are not repeated.

**Preconditions:** [plan.md](plan.md) green; [plan-b.md](plan-b.md) landed; Tasks C1–C4
landed.

---

## Task C5: G6 seeds — `925040100` and `910510000`

No engine work. Plan A Task A2/A3 plumbed `step` through the condition contract, which is
what `questProgress` needs (the aggregator rejects a step-less `questProgress` at
`query-aggregator/validation/rest.go:238-245`), and Plan A Task A3 added the `map_id`
range operators.

**PRD correction (plan-c.md §0.1).** `isCygnus()` is
`Character.isCygnus() { return getJobType() == 1; }` where `getJobType()` is
`job.getId() / 1000` — the inclusive band **1000–1999**. It needs no `in` operator and no
new condition: `jobId >= 1000` AND `jobId <= 1999` expresses it exactly. Its negation
needs two rules, because a rule's conditions are AND-ed only.

### Files

Create under all 11 roots at `map-actions/onUserEnter/`:

- `map-925040100.json` — **new file**
- `map-910510000.json` — **new file**

22 files total. Author under `deploy/seed/gms/83_1/map-actions/onUserEnter/`.

Patterns to copy: `deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json` for
the multi-rule envelope; Plan B Task B6's `map-100000006.json` for a `questStatus`
condition carrying `referenceId`.

- [ ] **Step 1: Author `map-925040100.json`**

Cosmic body:
`if (ms.isQuestStarted(21747) && ms.getQuestProgressInt(21747, 9300351) == 0) { spawn 9300351 @ (897, 51) }`.

`isQuestStarted(21747)` → `questStatus referenceId 21747 = 2` (Cosmic STARTED(1) + 1).
`getQuestProgressInt(21747, 9300351) == 0` → `questProgress referenceId 21747 step 9300351 = 0`.

```json
{
  "data": {
    "attributes": {
      "description": "Map 925040100 - spawns 9300351 at (897, 51) while quest 21747 is started and its 9300351 progress is 0",
      "rules": [
        {
          "conditions": [
            {
              "operator": "=",
              "referenceId": "21747",
              "type": "questStatus",
              "value": "2"
            },
            {
              "operator": "=",
              "referenceId": "21747",
              "step": "9300351",
              "type": "questProgress",
              "value": "0"
            }
          ],
          "id": "spawn_quest_monster",
          "operations": [
            {
              "params": {
                "monsterId": "9300351",
                "spawnIfAbsent": "true",
                "x": "897",
                "y": "51"
              },
              "type": "spawn_monster"
            }
          ]
        }
      ],
      "scriptName": "925040100"
    },
    "id": "925040100",
    "type": "map-action"
  }
}
```

Note the condition key order stays alphabetical: `operator`, `referenceId`, `step`,
`type`, `value`.

- [ ] **Step 2: Author `map-910510000.json`**

Cosmic body branches on `player.isCygnus()`:
- Cygnus → `if (ms.isQuestStarted(20730) && ms.getQuestProgressInt(20730, 9300285) == 0) { spawn 9300285 @ (680, 258) }`
- non-Cygnus → `if (ms.isQuestStarted(21731) && ms.getQuestProgressInt(21731, 9300344) == 0) { spawn 9300344 @ (680, 258) }`

Three rules, because "not in 1000–1999" is a disjunction:

| rule id | conditions | spawn |
|---|---|---|
| `spawn_cygnus` | `jobId >= 1000`, `jobId <= 1999`, `questStatus ref 20730 = 2`, `questProgress ref 20730 step 9300285 = 0` | `9300285` @ (680, 258) |
| `spawn_explorer_below` | `jobId < 1000`, `questStatus ref 21731 = 2`, `questProgress ref 21731 step 9300344 = 0` | `9300344` @ (680, 258) |
| `spawn_explorer_above` | `jobId > 1999`, `questStatus ref 21731 = 2`, `questProgress ref 21731 step 9300344 = 0` | `9300344` @ (680, 258) |

```json
{
  "data": {
    "attributes": {
      "description": "Map 910510000 - spawns the Cygnus (9300285) or Explorer (9300344) quest monster at (680, 258) depending on job family and quest progress",
      "rules": [
        {
          "conditions": [
            {
              "operator": ">=",
              "type": "jobId",
              "value": "1000"
            },
            {
              "operator": "<=",
              "type": "jobId",
              "value": "1999"
            },
            {
              "operator": "=",
              "referenceId": "20730",
              "type": "questStatus",
              "value": "2"
            },
            {
              "operator": "=",
              "referenceId": "20730",
              "step": "9300285",
              "type": "questProgress",
              "value": "0"
            }
          ],
          "id": "spawn_cygnus",
          "operations": [
            {
              "params": {
                "monsterId": "9300285",
                "spawnIfAbsent": "true",
                "x": "680",
                "y": "258"
              },
              "type": "spawn_monster"
            }
          ]
        },
        {
          "conditions": [
            {
              "operator": "<",
              "type": "jobId",
              "value": "1000"
            },
            {
              "operator": "=",
              "referenceId": "21731",
              "type": "questStatus",
              "value": "2"
            },
            {
              "operator": "=",
              "referenceId": "21731",
              "step": "9300344",
              "type": "questProgress",
              "value": "0"
            }
          ],
          "id": "spawn_explorer_below",
          "operations": [
            {
              "params": {
                "monsterId": "9300344",
                "spawnIfAbsent": "true",
                "x": "680",
                "y": "258"
              },
              "type": "spawn_monster"
            }
          ]
        },
        {
          "conditions": [
            {
              "operator": ">",
              "type": "jobId",
              "value": "1999"
            },
            {
              "operator": "=",
              "referenceId": "21731",
              "type": "questStatus",
              "value": "2"
            },
            {
              "operator": "=",
              "referenceId": "21731",
              "step": "9300344",
              "type": "questProgress",
              "value": "0"
            }
          ],
          "id": "spawn_explorer_above",
          "operations": [
            {
              "params": {
                "monsterId": "9300344",
                "spawnIfAbsent": "true",
                "x": "680",
                "y": "258"
              },
              "type": "spawn_monster"
            }
          ]
        }
      ],
      "scriptName": "910510000"
    },
    "id": "910510000",
    "type": "map-action"
  }
}
```

- [ ] **Step 3: Confirm the rule-evaluation semantics match**

`map_script_schema.json:21` describes rules as "first matching rule wins". Cosmic's
`910510000` is an if/else on `isCygnus()`, so at most one branch fires — first-match is
correct. But **read `script/processor.go`'s rule loop and confirm the engine actually
stops at the first match** rather than running every matching rule:

Run: `grep -rn "Rules()" services/atlas-map-actions/atlas.com/map-actions/script/*.go`

If the engine runs *all* matching rules, this document is still correct — the three rules
are mutually exclusive by construction — but record the finding in the commit body,
because Task C23's `explorationPoint` depends on the answer and would be wrong under
first-match-wins.

- [ ] **Step 4: Replicate and verify**

```bash
for f in 925040100 910510000; do
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

- [ ] **Step 5: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): G6 quest-progress gated spawns; isCygnus as a jobId band"
```

---

## Task C6: G7 seed — `pepeking_effect`

No engine work. Plan A Task A5 added the `monsterIds` param and the uniform pick.

Cosmic body: `mobId = 3300000 + (Math.floor(Math.random() * 3) + 5)`, i.e. one of
`3300005`, `3300006`, `3300007`, spawned at `new Point(-28, -67)`.

### Files

- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-pepeking_effect.json` — **new file**, plus the other ten roots (11 total)

Patterns to copy: Plan B Task B5's boss documents for the `spawn_monster` params.

- [ ] **Step 1: Author the document**

```json
{
  "data": {
    "attributes": {
      "description": "Pepe King map - spawns one of 3300005/3300006/3300007 at (-28, -67), chosen uniformly",
      "rules": [
        {
          "conditions": [],
          "id": "spawn_random_pepe",
          "operations": [
            {
              "params": {
                "monsterIds": "3300005,3300006,3300007",
                "spawnIfAbsent": "true",
                "x": "-28",
                "y": "-67"
              },
              "type": "spawn_monster"
            }
          ]
        }
      ],
      "scriptName": "pepeking_effect"
    },
    "id": "pepeking_effect",
    "type": "map-action"
  }
}
```

`monsterId` is absent — Plan A Task A5 made `monsterId` and `monsterIds` mutually
exclusive and the schema's `oneOf` enforces it, so including both fails validation.

`spawnIfAbsent` is `"true"` here even though Cosmic has no guard on this spawn. That is
deliberate: PRD FR-2.2 and the catalog-lint rule require it on every spawn, and
`spawnIfAbsent` is evaluated per template — so a second entry can still spawn a
*different* one of the three. Note this reasoning in the commit body.

- [ ] **Step 2: Replicate and verify**

```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  cp deploy/seed/gms/83_1/map-actions/onUserEnter/map-pepeking_effect.json \
     "deploy/seed/$r/map-actions/onUserEnter/map-pepeking_effect.json"
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `11`.

- [ ] **Step 3: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): pepeking_effect randomized spawn via monsterIds (G7)"
```

---

## Task C7: `play_sound` end to end (G8, live part)

Cosmic's `AbstractPlayerInteraction.playSound(sound)` is
`getPlayer().getMap().broadcastMessage(PacketCreator.environmentChange(sound, 4))` —
`SendOpcode.FIELD_EFFECT` with mode byte 4.

**No new packet.** `libs/atlas-packet/field/clientbound/effect.go:98-100` already has
`NewFieldEffectSound(mode byte, path string) EffectString`, and
`libs/atlas-packet/field/field_effect_body.go:51` already has
`FieldEffectSoundBody(path string)`, which resolves the mode from the tenant's writer
options. It is already in production use at
`services/atlas-channel/atlas.com/channel/kafka/consumer/party_quest/consumer.go:97`.
This task only adds the command, the saga action and the executor arm.

### Files

- `libs/atlas-saga/model.go` — `PlaySound` action constant
- `libs/atlas-saga/payloads.go` — `PlaySoundPayload`
- `libs/atlas-saga/unmarshal.go` — the matching case
- `libs/atlas-saga/unmarshal_test.go` — round-trip test
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/system_message/kafka.go` — `CommandPlaySound` and `PlaySoundBody`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/system_message/processor.go` — a `PlaySound` method
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — two aliases + unmarshal case
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — the accepted-action entry
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — interface decl, dispatch case, handler body
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/character_extractor.go` — the payload case
- `services/atlas-channel/atlas.com/channel/kafka/consumer/system_message/consumer.go` — `handlePlaySound` and its registration
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` + `executor_test.go`
- `services/atlas-map-actions/docs/map_script_schema.json` — one new `allOf` block
- `libs/atlas-packet/field/field_effect_body.go` — read-only; `FieldEffectSoundBody` at line 51
- `services/atlas-channel/atlas.com/channel/kafka/consumer/party_quest/consumer.go` — read-only; line 97 is the worked example

Module roots: `libs/atlas-saga`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`,
`services/atlas-channel/atlas.com/channel`,
`services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `ShowInfo`'s complete chain — `saga/handler.go:2835` →
`system_message/processor.go`'s `ShowInfo` → `CommandShowInfo`/`ShowInfoBody`
(`kafka/message/system_message/kafka.go:22,48-51`) → atlas-channel's `handleShowInfo`
(`system_message/consumer.go:173`). `PlaySoundBody` is structurally identical to
`ShowInfoBody`; the only difference is the writer body helper the channel handler picks.

**Interfaces:**
- Produces: `saga.PlaySound Action = "play_sound"`; `saga.PlaySoundPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, Path string}`; `system_message.CommandPlaySound = "PLAY_SOUND"`; `system_message.PlaySoundBody{Path string \`json:"path"\`}`; the `play_sound` document operation with required param `path`.

- [ ] **Step 1: Confirm the sound mode is already resolvable per tenant**

Run:
```bash
sed -n '45,60p' libs/atlas-packet/field/field_effect_body.go
grep -rn "SOUND" services/atlas-configurations/seed-data/templates/template_gms_83_1.json | head -5
```

`FieldEffectSoundBody` resolves its mode byte from the tenant's `FieldEffect` writer
options. Confirm the option key it reads exists in the seeded templates. If it does not,
STOP — that is a template gap and adding a wire byte by hand is exactly what
`docs/packets/PROCESS.md` forbids. Report it instead.

- [ ] **Step 2: Write the failing tests**

**`libs/atlas-saga/unmarshal_test.go` → `TestUnmarshalPlaySoundStep`** — unmarshal
`{"stepId":"sound-1","status":"pending","action":"play_sound","payload":{"characterId":1,"worldId":0,"channelId":1,"path":"cannonshooter/flying"}}`
and assert the payload type-asserts to
`PlaySoundPayload{CharacterId: 1, WorldId: 0, ChannelId: 1, Path: "cannonshooter/flying"}`.

**`script/executor_test.go` → `TestExecutePlaySound`** — params
`{"path":"cannonshooter/flying"}`, field
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(913070000)).Build()`. Assert the
captured payload equals
`saga.PlaySoundPayload{CharacterId: 1, WorldId: world.Id(0), ChannelId: channel.Id(1), Path: "cannonshooter/flying"}`.

**`script/executor_test.go` → `TestExecutePlaySoundRequiresPath`** — params `{}`; expect
error containing `play_sound operation missing path parameter`.

- [ ] **Step 3: Run the tests and confirm they fail**

Run:
```bash
cd libs/atlas-saga && go test ./... -run TestUnmarshalPlaySoundStep -v; cd -
cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run TestExecutePlaySound -v; cd -
```
Expected: compile failure in the first (`PlaySoundPayload` undefined);
`unknown operation type [play_sound]` in the second.

- [ ] **Step 4: Add the saga action, payload and unmarshal case**

`libs/atlas-saga/model.go`, beside `ShowInfo` (line 163):

```go
	// PlaySound plays a WZ sound path for one character. Cosmic's
	// AbstractPlayerInteraction.playSound is FIELD_EFFECT mode 4; the mode
	// byte is resolved per tenant by FieldEffectSoundBody, never carried here.
	PlaySound Action = "play_sound"
```

`libs/atlas-saga/payloads.go`, beside `ShowInfoPayload` (lines 455-460):

```go
// PlaySoundPayload represents the payload required to play a sound for a character.
type PlaySoundPayload struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	Path        string     `json:"path"`
}
```

Add `case PlaySound:` to `libs/atlas-saga/unmarshal.go`.

- [ ] **Step 5: Add the system-message command**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/system_message/kafka.go`,
add to the command const block (beside `CommandShowInfo`, line 22):

```go
	CommandPlaySound       = "PLAY_SOUND"
```

and beside `ShowInfoBody` (lines 48-51):

```go
// PlaySoundBody is the body for playing a WZ sound for a character
type PlaySoundBody struct {
	Path string `json:"path"` // Path to the sound (e.g., "cannonshooter/flying")
}
```

Add a `PlaySound(worldId world.Id, channelId channel.Id, characterId uint32, path string) error`
method to `system_message.Processor`, copying its `ShowInfo` method verbatim and
swapping the command constant and body type.

- [ ] **Step 6: Wire the orchestrator's seven touchpoints**

Same checklist as Plan C Task C4 Step 5, substituting `PlaySound`/`PlaySoundPayload`.
Handler body, copying `handleShowInfo` (`saga/handler.go:2835`) — synchronous, marks the
step complete immediately, because a sound has no completion event and no meaningful
compensation:

```go
// handlePlaySound handles the PlaySound action. Fire-and-forget: a sound
// cannot be un-played, so this action registers no compensator.
func (h *HandlerImpl) handlePlaySound(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(PlaySoundPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	if err := h.systemMessageP.PlaySound(payload.WorldId, payload.ChannelId, payload.CharacterId, payload.Path); err != nil {
		h.logActionError(s, st, err, fmt.Sprintf("Failed to play sound %s", payload.Path))
		return err
	}
	return nil
}
```

Match `handleShowInfo`'s exact step-completion call — read it and copy, do not invent.

- [ ] **Step 7: Add the atlas-channel handler**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/system_message/consumer.go`,
add `handlePlaySound` immediately after `handleFieldEffect` (which ends around line 338),
copying that function's shape exactly and swapping the body helper:

```go
func handlePlaySound(sc server.Model, wp writer.Producer) message.Handler[system_message2.Command[system_message2.PlaySoundBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, cmd system_message2.Command[system_message2.PlaySoundBody]) {
		if cmd.Type != system_message2.CommandPlaySound {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		if !sc.Is(t, cmd.WorldId, cmd.ChannelId) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectSoundBody(cmd.Body.Path)))
		if err != nil {
			l.WithError(err).Errorf("Unable to play sound for character [%d].", cmd.CharacterId)
		}
	}
}
```

Register it beside the existing `handleFieldEffect` registration — find that registration
in the same file and add the parallel line.

- [ ] **Step 8: Add the map-actions arm**

`case "play_sound": return e.executePlaySound(f, characterId, op)`, with a method copying
Plan C Task C3's `executeShowInfo` verbatim and swapping `saga.ShowInfo`/`ShowInfoPayload`
for `saga.PlaySound`/`PlaySoundPayload` and the step id prefix for `"play-sound-"` and
`SetInitiatedBy("map-action-play-sound")`.

Add the schema `allOf` block (`play_sound` requiring `path`, description
`"WZ sound path (e.g. cannonshooter/flying)"`), regenerate, `--check`.

- [ ] **Step 9: Build and test all four modules**

```bash
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
```
Expected: all four exit 0.

- [ ] **Step 10: Commit**

```bash
git add libs/atlas-saga/ services/
git commit -m "feat(saga): play_sound action riding the existing FieldEffect sound writer"
```

---

## Task C8: the two convertible cannon-tutorial seeds (G8, live part)

Per [plan-c.md](plan-c.md) §0, `cannon_tuto_direction1` and `cannon_tuto_direction2` each
execute exactly one call before Cosmic aborts on an unresolved method: their
`ms.playSound(...)`. That call, and only that call, converts.

### Files

- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-cannon_tuto_direction1.json` — **new file**
- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-cannon_tuto_direction2.json` — **new file**

22 files total across the 11 roots.

Patterns to copy: Plan B Task B2's `map-Resi_tutor20.json` for the single-operation
envelope.

- [ ] **Step 1: Author the two documents**

```json
{
  "data": {
    "attributes": {
      "description": "Cannoneer tutorial - plays the <path> sound. Cosmic's remaining calls in this script (sendDirectionInfo/setDirectionStatus) are unresolved methods that abort the script; see plan-c.md section 0.",
      "rules": [
        {
          "conditions": [],
          "id": "play_sound",
          "operations": [
            {
              "params": {
                "path": "<path>"
              },
              "type": "play_sound"
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

| scriptName | path |
|---|---|
| `cannon_tuto_direction1` | `cannonshooter/flying` |
| `cannon_tuto_direction2` | `cannonshooter/bang` |

The `description` carries the §0 rationale so a future reader of the seed alone does not
"complete" the script from the PRD's gap table.

- [ ] **Step 2: Replicate and verify**

```bash
for f in cannon_tuto_direction1 cannon_tuto_direction2; do
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

- [ ] **Step 3: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): the two cannon-tutorial sounds Cosmic actually executes (G8 partial)"
```

---

## Task C9: `change_music` and `boat_effect` end to end (G1c)

Two more operations that ride packets the repo already has verified.

- `PacketCreator.musicChange(song)` is `environmentChange(song, 6)` — `FIELD_EFFECT`
  mode 6. `libs/atlas-packet/field/field_effect_body.go:63` already has
  `FieldEffectBackgroundMusicBody(name string)`, in production use at
  `services/atlas-channel/atlas.com/channel/kafka/consumer/event/consumer.go:86` and
  `kafka/consumer/map/consumer.go:380,814,1030`.
- `PacketCreator.crogBoatPacket(true)` is `SendOpcode.CONTI_MOVE` with
  `writeByte(10); writeByte(4)` — state 10, subState 4, which the repo's
  `ContiMove` codec documents as `OnMoveField` → `CShip::AppearShip`
  (`libs/atlas-packet/field/clientbound/conti_move.go:23-29`). That is exactly the
  `ContiMoveShow` key already resolved per tenant by
  `services/atlas-channel/atlas.com/channel/socket/writer/conti_move.go:35`.

**No new packet, and no wire byte in any payload.** Both effects are named, never
encoded, by the saga — matching the `ContiMoveBody` doc comment's explicit rule
(`socket/writer/conti_move.go:13-18`): state/subState are resolved from the tenant's
writer-options table, never carried as free-form config from the triggering domain.

### Files

- `libs/atlas-saga/model.go`, `payloads.go`, `unmarshal.go`, `unmarshal_test.go` — two new actions and payloads
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/system_message/kafka.go` — two commands and bodies
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/system_message/processor.go` — two methods
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/{model.go,event_acceptance.go,handler.go,character_extractor.go}` — the seven touchpoints × 2
- `services/atlas-channel/atlas.com/channel/kafka/consumer/system_message/consumer.go` — two handlers and their registrations
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` + `executor_test.go`
- `services/atlas-map-actions/docs/map_script_schema.json` — two new `allOf` blocks
- `services/atlas-channel/atlas.com/channel/socket/writer/conti_move.go` — read-only; `ContiMoveBody`/`ContiMoveShow`

Module roots: as Task C7.

Patterns to copy: Task C7 in full — this task is that task twice more, with
`FieldEffectBackgroundMusicBody` and `writer.ContiMoveBody(writer.ContiMoveShow)`
substituted for `FieldEffectSoundBody`. For the ContiMove announce, copy
`services/atlas-channel/atlas.com/channel/kafka/consumer/event/consumer.go:73-76` but
target one session rather than `ForSessionsInMap` — Cosmic sends both effects to the
entering client only (`ms.getClient().sendPacket(...)`), not the whole map.

**Interfaces:**
- Produces:
  - `saga.ChangeMusic Action = "change_music"`; `saga.ChangeMusicPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, Path string}`
  - `saga.BoatEffect Action = "boat_effect"`; `saga.BoatEffectPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, Show bool}`
  - `system_message.CommandChangeMusic = "CHANGE_MUSIC"` with `ChangeMusicBody{Path string}`; `system_message.CommandBoatEffect = "BOAT_EFFECT"` with `BoatEffectBody{Show bool}`
  - document operations `change_music` (required param `path`) and `boat_effect` (required param `show`, `"true"`/`"false"`)

- [ ] **Step 1: Write the failing tests**

**`libs/atlas-saga/unmarshal_test.go`** — `TestUnmarshalChangeMusicStep` and
`TestUnmarshalBoatEffectStep`, same shape as Task C7's, with payloads
`{"characterId":1,"worldId":0,"channelId":1,"path":"Bgm04/ArabPirate"}` and
`{"characterId":1,"worldId":0,"channelId":1,"show":true}`.

**`script/executor_test.go`**:

| test | params | expected payload |
|---|---|---|
| `TestExecuteChangeMusic` | `{"path":"Bgm04/ArabPirate"}` | `saga.ChangeMusicPayload{CharacterId: 1, WorldId: 0, ChannelId: 1, Path: "Bgm04/ArabPirate"}` |
| `TestExecuteBoatEffectShow` | `{"show":"true"}` | `saga.BoatEffectPayload{CharacterId: 1, WorldId: 0, ChannelId: 1, Show: true}` |
| `TestExecuteBoatEffectHide` | `{"show":"false"}` | same with `Show: false` |

Plus validation cases: `change_music` with `{}` → `change_music operation missing path parameter`;
`boat_effect` with `{}` → `boat_effect operation missing show parameter`;
`boat_effect` with `{"show":"yes"}` → `invalid show [yes]`.

- [ ] **Step 2: Run the tests and confirm they fail**

Run:
```bash
cd libs/atlas-saga && go test ./... -run 'TestUnmarshalChangeMusicStep|TestUnmarshalBoatEffectStep' -v; cd -
cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run 'TestExecuteChangeMusic|TestExecuteBoatEffect' -v; cd -
```
Expected: compile failure, then `unknown operation type [change_music]` /
`unknown operation type [boat_effect]`.

- [ ] **Step 3: Land `change_music`**

Repeat Task C7 Steps 4-8 verbatim with `ChangeMusic`/`ChangeMusicPayload`/
`CommandChangeMusic`/`ChangeMusicBody`, and the channel handler announcing
`fieldpkt.FieldEffectBackgroundMusicBody(cmd.Body.Path)` under
`fieldcb.FieldEffectWriter`.

- [ ] **Step 4: Land `boat_effect`**

Same, with two differences:

- `BoatEffectBody` carries `Show bool`, not a path. The channel handler maps it to a
  `writer.ContiMoveKey`:

```go
		key := writer.ContiMoveHide
		if cmd.Body.Show {
			key = writer.ContiMoveShow
		}
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(cmd.CharacterId,
			session.Announce(l)(ctx)(wp)(fieldcb.ContiMoveWriter)(writer.ContiMoveBody(key)))
```

- The saga payload carries `Show bool` and nothing else about the wire. Do **not** add a
  state or subState field to any payload, body or document param — that is the rule
  `socket/writer/conti_move.go:13-18` states, and violating it would push wire bytes into
  seed data.

- [ ] **Step 5: Confirm the tenant options resolve**

Run:
```bash
grep -rn "SHOW_STATE\|SHOW_SUB_STATE" services/atlas-configurations/seed-data/templates/ | head -5
```
Expected: hits in the template files. `ContiMoveBody` falls back to `ResolveCode`'s loud
`99` sentinel on a miss, which would crash the client — so a miss here is a hard stop, not
a warning. If the keys are absent for any of the 11 templates, report it rather than
adding them speculatively.

- [ ] **Step 6: Build and test all four modules, then commit**

```bash
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
./tools/gen-map-action-schema.sh && ./tools/gen-map-action-schema.sh --check
git add libs/atlas-saga/ services/
git commit -m "feat(saga): change_music and boat_effect actions on existing verified packets (G1c)"
```

---

## Task C10: the `transportInTransit` aggregator condition (G1b)

Design F2 established that `transportAvailable`
(`query-aggregator/validation/model.go:525-534`) collapses the five-state route machine
(`services/atlas-transports/atlas.com/transports/transport/state.go:8-20`:
`out_of_service`, `awaiting_return`, `open_entry`, `locked_entry`, `in_transit`) to a
boolean at `open_entry`, and that `!open_entry` is the wrong predicate for Cosmic's
`docked == "false"` because it is also true for `out_of_service` and `awaiting_return`.

The Cosmic lifecycle confirms which state is meant. `Boats.js` sets `docked="true"` in
`scheduleNew()` and `docked="false"` in `takeoff()`, and `arrived()` re-docks. So
`docked == "false"` is exactly the `takeoff` → `arrived` window: **`in_transit`**.

Per the decision recorded in [context.md](context.md), this lands as a second boolean
condition shaped exactly like `transportAvailable`, not as a general string-valued
`transportState` — `saga.ValidationConditionInput` carries no string operand, and
`transportAvailable` already set the precedent of collapsing this state machine to a
boolean.

### Files

- `libs/atlas-saga/validation.go` — `TransportInTransitCondition` constant
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` — the `ConditionType` alias and the `Evaluate`/`EvaluateWithContext` arm
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/builder.go` — the accepted-type switch (line 210), the `FromInput` validation (line 334-338 area) and the `Validate()` validation (line 417-420 area)
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/rest.go` — the REST-input validation arm (line 271-274 area)
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model_test.go` — extend
- `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/context.go` — read-only; `GetTransportState` at lines 346-359 is reused unchanged
- `services/atlas-transports/atlas.com/transports/transport/state.go` — read-only; the five state constants

Module root: `services/atlas-query-aggregator/atlas.com/query-aggregator`.

Patterns to copy: `TransportAvailableCondition`'s complete nine-touchpoint chain, listed
with exact `path:line` in [context.md](context.md) §"Aggregator condition scaffolding".
Every touchpoint of the new condition mirrors it; **no new `ValidationContext` getter is
needed**, because `GetTransportState(mapId)` already returns the raw state string.

**Interfaces:**
- Produces: `saga.TransportInTransitCondition = "transportInTransit"`; the aggregator `ConditionType` of the same name; semantics `actualValue = 1` when `GetTransportState(referenceId) == "in_transit"`, else `0`, compared against the condition's `value` with the usual operators.

- [ ] **Step 1: Read the exact state constant name**

Run: `cat services/atlas-transports/atlas.com/transports/transport/state.go`

Use the declared constant, not the string literal, if the aggregator can import it;
`transportAvailable`'s existing arm compares against the literal `"open_entry"`
(`validation/model.go:525-534`) — match whichever form that arm uses so the two are
consistent.

- [ ] **Step 2: Write the failing test**

Append to
`services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model_test.go`,
copying the setup of the existing `transportAvailable` test in that file (locate it with
`grep -n 'TransportAvailable' services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model_test.go`).

**`TestTransportInTransitCondition`** — table-driven over all five route states, with
`referenceId` `200090000` and `operator` `=`, `value` `1`:

| subtest | `GetTransportState` returns | expect passed |
|---|---|---|
| `out_of_service` | `out_of_service` | `false` |
| `awaiting_return` | `awaiting_return` | `false` |
| `open_entry` | `open_entry` | `false` |
| `locked_entry` | `locked_entry` | `false` |
| `in_transit` | `in_transit` | `true` |
| `unknown` | `unknown` | `false` |

And a second table with `value` `0` asserting the exact inverse, so the docked-side
predicate (Task C11's `200090000`) is covered too.

**`TestTransportInTransitRequiresReferenceId`** — build the condition with
`referenceId` 0 and assert the builder errors with
`referenceId is required for transportInTransit conditions`.

**`TestTransportInTransitAcceptedBySetType`** — `SetType("transportInTransit")` returns
no error. This is the assertion that catches the `builder.go:210` omission design §3.6
warns about.

- [ ] **Step 3: Run the tests and confirm they fail**

Run: `cd services/atlas-query-aggregator/atlas.com/query-aggregator && go test ./validation/... -run TestTransportInTransit -v`
Expected: FAIL — `SetType` returns `unsupported condition type: transportInTransit`.

- [ ] **Step 4: Add the shared constant**

In `libs/atlas-saga/validation.go`, beside `TransportAvailableCondition` (line 35):

```go
	// TransportInTransitCondition is true only while the route is sailing
	// (state == in_transit) — the takeoff..arrived window. It is deliberately
	// NOT the negation of transportAvailable, which is also false for
	// out_of_service and awaiting_return (task-290 design F2).
	TransportInTransitCondition = "transportInTransit"
```

- [ ] **Step 5: Wire the aggregator's touchpoints**

Following [context.md](context.md)'s nine-point checklist, with `transportAvailable` as
the model:

1. `validation/model.go` — `TransportInTransitCondition ConditionType = ConditionType(sharedsaga.TransportInTransitCondition)`, beside the `transportAvailable` alias at line 50
2. `validation/model.go` — the `Evaluate` arm beside line 525:
   ```go
   	case TransportInTransitCondition:
   		state := ctx.GetTransportState(_map.Id(c.referenceId))
   		if state == "in_transit" {
   			actualValue = 1
   		} else {
   			actualValue = 0
   		}
   ```
   and the same arm in `EvaluateWithContext` if that function carries a parallel switch —
   check both (`validation/model.go:107` and `:392`).
3. `validation/builder.go:210` — add `TransportInTransitCondition` to the accepted-type `case` list
4. `validation/builder.go` (~line 334) — `FromInput` referenceId check
5. `validation/builder.go` (~line 417) — `Validate()` referenceId check
6. `validation/rest.go` (~line 271) — REST-input referenceId check

All four referenceId checks use the identical message:
`referenceId is required for transportInTransit conditions`.

- [ ] **Step 6: Run the tests**

Run: `cd services/atlas-query-aggregator/atlas.com/query-aggregator && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 7: Regenerate the map-action schema**

The condition-type enum is generated from `libs/atlas-saga/validation.go`, so the new
constant must flow into the schema:

Run: `./tools/gen-map-action-schema.sh && ./tools/gen-map-action-schema.sh --check`
Expected: the diff adds `transportInTransit` to the condition enum; `--check` then exits 0.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-saga/validation.go services/atlas-query-aggregator/ \
        services/atlas-map-actions/docs/map_script_schema.json
git commit -m "feat(query-aggregator): transportInTransit condition for Cosmic's docked==false (G1b)"
```

---

## Task C11: the two dock-arrival seeds — `200090000` and `200090010` (G1c)

Cosmic body of both: `if (map.getDocked() == true) { musicChange("Bgm04/ArabPirate"); crogBoatPacket(true); }`
where `map` is `getMapFactory().getMap(<the script's own map id>)`.

`getDocked() == true` is the complement of the in-transit window, so the condition is
`transportInTransit = 0` with `referenceId` set to the map whose route state is being
read — the same map the script names.

### Files

- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-200090000.json` — **new file**
- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-200090010.json` — **new file**

22 files total.

Patterns to copy: Task C5's `map-925040100.json` for a condition carrying `referenceId`.

- [ ] **Step 1: Confirm `referenceId` semantics for the transport conditions**

`transportAvailable` reads `ctx.GetTransportState(_map.Id(c.referenceId))` — the
reference is a **map id**, not a route id. Confirm by reading
`services/atlas-query-aggregator/atlas.com/query-aggregator/validation/context.go:346-359`
and the transports-side lookup it calls, then use the map id the Cosmic script itself
passes to `getMapFactory().getMap(...)`: `200090000` and `200090010` respectively. If the
lookup turns out to be keyed by something else, record the correct key and use it.

- [ ] **Step 2: Author the two documents**

```json
{
  "data": {
    "attributes": {
      "description": "Dock-arrival map <mapId> - on a docked vessel, sets the ArabPirate BGM and shows the boat",
      "rules": [
        {
          "conditions": [
            {
              "operator": "=",
              "referenceId": "<mapId>",
              "type": "transportInTransit",
              "value": "0"
            }
          ],
          "id": "docked_arrival",
          "operations": [
            {
              "params": {
                "path": "Bgm04/ArabPirate"
              },
              "type": "change_music"
            },
            {
              "params": {
                "show": "true"
              },
              "type": "boat_effect"
            }
          ]
        }
      ],
      "scriptName": "<mapId>"
    },
    "id": "<mapId>",
    "type": "map-action"
  }
}
```

`<mapId>` is `200090000` in the first document and `200090010` in the second — it is both
the `scriptName`/`id` and the condition's `referenceId`, because the Cosmic script reads
its own map's docked flag.

- [ ] **Step 3: Replicate and verify**

```bash
for f in 200090000 200090010; do
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

- [ ] **Step 4: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): dock-arrival music and boat effect for 200090000/200090010 (G1c)"
```

---

## Task C12: the `warp_to_map` saga action (G1a)

Cosmic's `warpAhead(mapId)` sets `Character.newWarpMap`, which is consumed mid-transfer at
`Character.java:1785-1790` and dispatched to `changeMap(int map)`
(`Character.java:1365-1376`) — which resolves `warpMap.getRandomPlayerSpawnpoint()`. So
the target is **a random player spawn point of the destination map**, not a portal and
not fixed coordinates.

`libs/atlas-saga` has four warp actions and every one takes a portal:
`WarpToPortalPayload` (`payloads.go:40-48`) and `WarpPartyQuestMembersToMapPayload`
(`payloads.go:1259-1265`) both carry `PortalId`; `WarpToRandomPortalPayload`
(`payloads.go:34-38`) picks a random *portal*, which is not a spawn point.

### Files

- `libs/atlas-saga/model.go`, `payloads.go`, `unmarshal.go`, `unmarshal_test.go` — the new action and payload
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/{model.go,event_acceptance.go,handler.go,character_extractor.go}` — the seven touchpoints
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go` — a `WarpToMapAndEmit` method
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` + `executor_test.go`
- `services/atlas-map-actions/docs/map_script_schema.json` — one new `allOf` block
- `libs/atlas-saga/payloads.go:34-48` — read-only; the two existing warp payloads being distinguished from

Module roots: `libs/atlas-saga`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`,
`services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `handleWarpToRandomPortal` (`saga/handler.go:1103`) and its client
`character/processor.go:90-96` (`WarpToPortalAndEmit`/`WarpToPortal`, Kafka producer, not
REST). `WarpToRandomPortal` is the nearest sibling because it also resolves the
destination server-side rather than carrying it.

**Interfaces:**
- Produces: `saga.WarpToMap Action = "warp_to_map"`; `saga.WarpToMapPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, MapId _map.Id}`; `character.Processor.WarpToMapAndEmit(...)`; the `warp_to_map` document operation with required param `mapId`.

- [ ] **Step 1: Decide where the spawn point is resolved**

Cosmic resolves `getRandomPlayerSpawnpoint()` inside the warp, not at the call site. The
Atlas equivalent must do the same — the saga must name the map, not a coordinate.

Read how `WarpToRandomPortal` resolves its destination:

Run:
```bash
sed -n '1103,1125p' services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go
sed -n '80,110p' services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go
```

If atlas-character already exposes a "warp to map, pick a spawn point" command, use it and
this task reduces to the saga action plus the executor arm. If it only accepts a portal,
add the map-level command to atlas-character with its own test in that service — this is
the cross-service seam, so the test asserting the NEW contract belongs there, not here.
Record which case you found in the commit body.

Either way `character.Processor` gains one method, declared as:

```go
// Copy WarpToPortalAndEmit (character/processor.go:90) and drop the portal
// argument — the destination service picks the spawn point.
type Processor interface {
	WarpToMapAndEmit(characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id) error
}

func (p *ProcessorImpl) WarpToMapAndEmit(characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id) error
```

- [ ] **Step 2: Write the failing tests**

**`libs/atlas-saga/unmarshal_test.go` → `TestUnmarshalWarpToMapStep`** — payload
`{"characterId":1,"worldId":0,"channelId":1,"mapId":200090010}` asserts to
`WarpToMapPayload{CharacterId: 1, WorldId: 0, ChannelId: 1, MapId: 200090010}`.

**`script/executor_test.go` → `TestExecuteWarpToMap`** — params `{"mapId":"200090010"}`,
field `field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(101000301)).Build()`. Assert
the payload equals
`saga.WarpToMapPayload{CharacterId: 1, WorldId: world.Id(0), ChannelId: channel.Id(1), MapId: _map.Id(200090010)}`.

Note the payload's `MapId` is the **destination**, not `f.MapId()` — assert both, so a
future edit cannot silently swap them.

**`TestExecuteWarpToMapParamValidation`**:

| subtest | params | expect error message contains |
|---|---|---|
| `missing mapId` | `{}` | `warp_to_map operation missing mapId parameter` |
| `bad mapId` | `{"mapId":"x"}` | `invalid mapId [x]` |

- [ ] **Step 3: Run the tests and confirm they fail**

Run:
```bash
cd libs/atlas-saga && go test ./... -run TestUnmarshalWarpToMapStep -v; cd -
cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -run TestExecuteWarpToMap -v; cd -
```

- [ ] **Step 4: Add the action and payload**

`libs/atlas-saga/model.go`, beside the other warp actions (lines 117-119):

```go
	// WarpToMap warps a character to a map, letting the destination service
	// pick the spawn point. Distinct from WarpToPortal/WarpToRandomPortal,
	// both of which target a portal — Cosmic's warpAhead resolves
	// getRandomPlayerSpawnpoint(), which is not a portal (task-290 G1a).
	WarpToMap Action = "warp_to_map"
```

`libs/atlas-saga/payloads.go`, beside `WarpToPortalPayload` (lines 40-48):

```go
// WarpToMapPayload represents the payload required to warp a character to a
// map without naming a portal. The destination service picks the spawn point.
type WarpToMapPayload struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
}
```

Add the `case WarpToMap:` to `unmarshal.go`.

- [ ] **Step 5: Wire the orchestrator's seven touchpoints and the character client**

Same checklist as Plan C Task C4 Step 5. Handler body, copying
`handleWarpToRandomPortal` (`saga/handler.go:1103`):

```go
// handleWarpToMap handles the WarpToMap action
func (h *HandlerImpl) handleWarpToMap(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(WarpToMapPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	if err := h.charP.WarpToMapAndEmit(payload.CharacterId, payload.WorldId, payload.ChannelId, payload.MapId); err != nil {
		h.logActionError(s, st, err, fmt.Sprintf("Failed to warp character %d to map %d", payload.CharacterId, payload.MapId))
		return err
	}
	return nil
}
```

Match `handleWarpToRandomPortal`'s exact step-completion semantics — read it first; if it
leaves the step Pending awaiting a map-change event, `handleWarpToMap` must do the same.

- [ ] **Step 6: Add the executor arm and schema block**

`case "warp_to_map": return e.executeWarpToMap(f, characterId, op)`, with a method in the
shape of Task C1's, building a one-step saga with
`SetInitiatedBy("map-action-warp")` and step id
`fmt.Sprintf("warp-%d-%d", characterId, mapId)`.

Schema `allOf` block: `warp_to_map` requiring `mapId`, description
`"Destination map ID; the destination service picks the spawn point"`. Regenerate,
`--check`.

- [ ] **Step 7: Build, test and commit**

```bash
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-character/atlas.com/character && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
git add libs/atlas-saga/ services/
git commit -m "feat(saga): warp_to_map action for Cosmic warpAhead (G1a)"
```

---

## Task C13: the twelve transport catch-up seeds (G1a/G1b)

Twelve scripts share one body:
`if (em.getProperty("docked") == "false") { ms.getClient().getPlayer().warpAhead(<toMap>); }`
— a player who arrived on the departure map after the vessel left is sent on to the
in-transit map.

`docked == "false"` is `transportInTransit = 1` (Task C10). The `referenceId` is the
script's own map, per Task C11 Step 1's finding.

### Files

Create under all 11 roots at `map-actions/onUserEnter/`, one per script name below —
132 files total.

Patterns to copy: Task C11's documents for the `transportInTransit` condition shape.

- [ ] **Step 1: Author the twelve documents under `gms/83_1`**

```json
{
  "data": {
    "attributes": {
      "description": "Departure map <scriptName> (<eventName> route) - warps a late arrival ahead to <toMap> while the vessel is in transit",
      "rules": [
        {
          "conditions": [
            {
              "operator": "=",
              "referenceId": "<scriptName>",
              "type": "transportInTransit",
              "value": "1"
            }
          ],
          "id": "warp_ahead",
          "operations": [
            {
              "params": {
                "mapId": "<toMap>"
              },
              "type": "warp_to_map"
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

| scriptName | toMap | Cosmic eventName |
|---|---|---|
| `101000301` | `200000112` | Boats |
| `200000112` | `101000301` | Boats |
| `200000122` | `200090100` | Trains |
| `200000132` | `200090200` | Cabin |
| `200000152` | `200090400` | Genie |
| `220000111` | `200090110` | Trains |
| `240000111` | `200090210` | Cabin |
| `260000110` | `200090410` | Genie |
| `540010001` | `540010002` | AirPlane |
| `540010100` | `540010101` | AirPlane |
| `600010002` | `600010003` | Subway |
| `600010004` | `600010005` | Subway |

Note `101000301` → `200000112` and `200000112` → `101000301` are a reciprocal pair; both
literal values are from the Cosmic sources and neither is a transcription error.

PRD §4.5's G1 list writes these as `101000301`→200090010 for the first entry. **That is
wrong** — the Cosmic body of `101000301.js` calls `warpAhead(200000112)`. Use the table
above; record the correction in the commit body.

- [ ] **Step 2: Replicate and verify**

```bash
for f in 101000301 200000112 200000122 200000132 200000152 220000111 240000111 260000110 540010001 540010100 600010002 600010004; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `132`.

- [ ] **Step 3: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): twelve transport catch-up warps (G1a/G1b)

PRD 4.5 lists 101000301 -> 200090010; the Cosmic body calls
warpAhead(200000112). Corrected."
```

---

## Task C14: `spawn_npc` — a field NPC registry in atlas-maps (G2)

Cosmic's `AbstractPlayerInteraction.spawnNpc(npcId, pos, map)` builds a live `NPC` via
`LifeFactory.getNPC(npcId)`, sets its position/rx0/rx1/foothold, calls
`map.addMapObject(npc)` and broadcasts `PacketCreator.spawnNPC(npc)`. It is
**session/instance-scoped** — not persisted, cleared by `clearMapObjects()`, and
re-created by the `onUserEnter` guard `!mapobj.containsNPC(npcId)` on each entry.

No Atlas service owns field-scoped NPC placement today: `atlas-maps` has no `npc` package
at all, `atlas-npc-conversations` owns conversation state machines, `atlas-npc-shops` owns
shop dialogs, and `atlas-player-npcs` owns player-deployed shop NPCs (which is what
`DeployPlayerNpc`, `libs/atlas-saga/model.go:175-179`, is for — explicitly not this).

`atlas-maps` is the owner: it already holds map-scoped monster spawn-point registries
(`services/atlas-maps/atlas.com/maps/map/monster/registry.go`) and already consumes
field-effect commands.

This is the largest single task in Plan C. If it exceeds the implementer budget, split it
at the boundary between "atlas-maps NPC registry + REST" and "saga action + executor
arm", and land them as two commits.

### Files

- `services/atlas-maps/atlas.com/maps/map/npc/registry.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/npc/model.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/npc/processor.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/npc/rest.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/npc/resource.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/npc/registry_test.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/npc/processor_test.go` — **new file**
- `libs/atlas-saga/model.go`, `payloads.go`, `unmarshal.go`, `unmarshal_test.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/npc_spawn/{processor.go,requests.go,rest.go}` — **new package**
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/{model.go,event_acceptance.go,handler.go,character_extractor.go}`
- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` + `executor_test.go`
- `services/atlas-map-actions/docs/map_script_schema.json`

Module roots: `services/atlas-maps/atlas.com/maps`, `libs/atlas-saga`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`,
`services/atlas-map-actions/atlas.com/map-actions`.

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/monster/registry.go` for the
in-memory, tenant-keyed, field-scoped registry shape (read it in full first — it defines
`SpawnPointRegistry`, `InitializeForMap`, `Count`, `ReserveEligibleSpawnPoints`,
`Reset`). For the orchestrator's domain client, copy
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/` in its entirety —
`requests.go`'s `worlds/%d/channels/%d/maps/%d/instances/%s/...` path template,
`rest.go`'s input/response/request triple, and `processor.go`'s
`requestX(p.ctx, f, req.ToRestModel())(p.l, p.ctx)` call shape.

**Interfaces:**
- Produces:
  - atlas-maps: `POST /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/npcs` and `GET` on the same collection; `npc.Processor.Create(f field.Model, input RestModel) (Model, error)` and `npc.Processor.GetInField(f field.Model) ([]Model, error)`
  - `npc.RestModel{Id string, NpcId uint32, X int16, Y int16, Fh int16, SpawnIfAbsent bool}`
  - `saga.SpawnNpc Action = "spawn_npc"`; `saga.SpawnNpcPayload{CharacterId uint32, WorldId world.Id, ChannelId channel.Id, MapId _map.Id, Instance uuid.UUID, NpcId uint32, X int16, Y int16, SpawnIfAbsent bool}`
  - the `spawn_npc` document operation with required params `npcId`, `x`, `y` and optional `spawnIfAbsent`

- [ ] **Step 1: Read the monster registry and confirm the pattern transfers**

Run:
```bash
cat services/atlas-maps/atlas.com/maps/map/monster/registry.go
ls services/atlas-maps/atlas.com/maps/map/monster/
grep -rn "InitResource\|RouteInitializer" services/atlas-maps/atlas.com/maps/ | head
```

Record two decisions before writing code, both in the commit body:
1. **Storage.** Cosmic's NPCs are session-scoped, so an in-memory registry keyed by
   tenant + field is faithful. Confirm whether atlas-maps' existing registries are
   in-memory or Redis-backed and match them — atlas-monsters uses Redis
   (`monster/registry.go:322`), atlas-maps' spawn-point registry may not.
2. **Broadcast.** Cosmic broadcasts `PacketCreator.spawnNPC(npc)` to the map. Determine
   whether atlas-channel already has an NPC-spawn writer:
   `grep -rn "Npc" libs/atlas-packet/npc/clientbound/ 2>/dev/null | head`. If it does not,
   **STOP and report** — that is a new packet and out of this plan's scope per the Global
   Constraints. If it does, the create path emits a status event the channel already
   consumes, or you add that consumption following the monster-created precedent.

- [ ] **Step 2: Write the failing atlas-maps tests**

`services/atlas-maps/atlas.com/maps/map/npc/registry_test.go` and `processor_test.go`,
copying the setup shape from the monster package's tests in the same service.

**`TestCreateNpcInField`** — create npc `1104100` at `(2830, 78)` on field
`field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()`. Assert
`GetInField(f)` returns exactly one model with `NpcId() == 1104100`, `X() == 2830`,
`Y() == 78`.

**`TestCreateNpcSpawnIfAbsentSuppressesWhenPresent`** — create `1104100`, then create it
again with `SpawnIfAbsent: true`. Assert no error, the returned model has
`UniqueId() == 0`, and `GetInField(f)` still returns 1.

**`TestCreateNpcSpawnIfAbsentIsFieldScoped`** — create `1104100` on field instance A, then
with `SpawnIfAbsent: true` on field instance B of the same map. Assert the second create
happens.

**`TestCreateNpcWithoutGuardStacks`** — two creates without `SpawnIfAbsent`;
`GetInField(f)` returns 2.

- [ ] **Step 3: Implement the atlas-maps npc package**

Registry, model, processor, REST models and resource routes, mirroring the monster
package's file layout and the `handleCreateMonsterInMap` handler shape
(`services/atlas-monsters/atlas.com/monsters/world/resource.go:207-231`). The
`SpawnIfAbsent` pre-check is the same shape Plan A Task A7 landed in atlas-monsters:

```go
	if input.SpawnIfAbsent {
		existing, err := p.GetInField(f)
		if err != nil {
			return Model{}, err
		}
		for _, n := range existing {
			if n.NpcId() == input.NpcId {
				return Model{}, nil
			}
		}
	}
```

Register the routes in atlas-maps' route initializer alongside the existing ones.

- [ ] **Step 4: Run the atlas-maps tests**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 5: Add the saga action and orchestrator client**

`libs/atlas-saga/model.go`:

```go
	// SpawnNpc places a scripted NPC on a field. Distinct from DeployPlayerNpc,
	// which deploys a character's own player-NPC (task-290 G2).
	SpawnNpc Action = "spawn_npc"
```

`libs/atlas-saga/payloads.go`, modelled on `SpawnMonsterPayload` (lines 511-523):

```go
// SpawnNpcPayload represents the payload required to place a scripted NPC on a field.
type SpawnNpcPayload struct {
	CharacterId   uint32     `json:"characterId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	NpcId         uint32     `json:"npcId"`
	X             int16      `json:"x"`
	Y             int16      `json:"y"`
	SpawnIfAbsent bool       `json:"spawnIfAbsent,omitempty"`
}
```

Add the `unmarshal.go` case and a `TestUnmarshalSpawnNpcStep` round-trip test.

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/npc_spawn/` by
copying the `monster/` package's three files and changing the path constant to
`worlds/%d/channels/%d/maps/%d/instances/%s/npcs`, the root URL token to whatever
atlas-maps registers (`grep -rn "MAPS" services/atlas-saga-orchestrator/ | head` to find
the existing token), and the models to the NPC shape.

Wire the seven orchestrator touchpoints. The handler resolves a foothold the same way
`handleSpawnMonster` does (`h.footholdP.GetFootholdBelow`, `saga/handler.go:2074-2079`),
because Cosmic's `spawnNpc` sets a foothold on the NPC.

- [ ] **Step 6: Add the executor arm and schema block**

`case "spawn_npc": return e.executeSpawnNpc(f, characterId, op)`. The method reads
`npcId` (required), `x`/`y` (required — unlike `spawn_monster`, where they default to 0,
because every G2 script gives an explicit `Point`), and `spawnIfAbsent` (optional, parsed
with `strconv.ParseBool`, same wording as Plan A Task A5's `invalid spawnIfAbsent [%s]`).

Schema `allOf`: `spawn_npc` requiring `npcId`, `x`, `y`, with optional `spawnIfAbsent`
constrained to `["true","false"]`. Regenerate, `--check`.

Executor test **`TestExecuteSpawnNpc`** — params
`{"npcId":"1104100","spawnIfAbsent":"true","x":"2830","y":"78"}`, field with instance
`inst`. Assert the payload equals
`saga.SpawnNpcPayload{CharacterId: 1, WorldId: 0, ChannelId: 1, MapId: 108010600, Instance: inst, NpcId: 1104100, X: 2830, Y: 78, SpawnIfAbsent: true}`.

- [ ] **Step 7: Build, test, commit**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./... && cd -
cd libs/atlas-saga && go build ./... && go test ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./... && cd -
cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... && cd -
git add services/ libs/atlas-saga/
git commit -m "feat(maps): field NPC registry and spawn_npc saga action (G2)"
```

---

## Task C15: the five explorer-route NPC seeds (G2)

Each Cosmic body is
`if (!mapobj.containsNPC(<npcId>)) { ms.spawnNpc(<npcId>, new Point(<x>, <y>), mapobj); }`
— the guard is exactly `spawnIfAbsent`.

### Files

Create under all 11 roots at `map-actions/onUserEnter/`: `map-108010600.json`,
`map-108010610.json`, `map-108010620.json`, `map-108010630.json`, `map-108010640.json` —
55 files total.

Patterns to copy: Plan B Task B5's boss documents for the single-operation spawn shape.

- [ ] **Step 1: Author the five documents**

```json
{
  "data": {
    "attributes": {
      "description": "Explorer route map <scriptName> - places NPC <npcId> at (<x>, <y>) if absent",
      "rules": [
        {
          "conditions": [],
          "id": "spawn_route_npc",
          "operations": [
            {
              "params": {
                "npcId": "<npcId>",
                "spawnIfAbsent": "true",
                "x": "<x>",
                "y": "<y>"
              },
              "type": "spawn_npc"
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

| scriptName | npcId | x | y |
|---|---|---|---|
| `108010600` | `1104100` | `2830` | `78` |
| `108010610` | `1104101` | `3395` | `-322` |
| `108010620` | `1104102` | `500` | `-522` |
| `108010630` | `1104103` | `-2263` | `-582` |
| `108010640` | `1104104` | `372` | `70` |

- [ ] **Step 2: Replicate and verify**

```bash
for f in 108010600 108010610 108010620 108010630 108010640; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `55`.

- [ ] **Step 3: Confirm catalog-lint's spawn rule covers `spawn_npc`**

Plan A Task A10's rule was written for `spawn_monster`. Decide whether it should also
require `spawnIfAbsent` on `spawn_npc` — all five of these documents set it, and Cosmic
guards all five. Extend `tools/catalog-lint/mapactions.go`'s check to cover both operation
types, add the fixture and the test, and re-run the linter.

- [ ] **Step 4: Commit**

```bash
git add deploy/seed/ tools/catalog-lint/
git commit -m "feat(seed): five explorer-route NPC spawns (G2)"
```

---

*(Tasks C16–C23 continue in [plan-c3.md](plan-c3.md).)*
