# task-290 — Planning Context

Companion to [plan.md](plan.md) (Plan A), [plan-b.md](plan-b.md) (Plan B) and
[plan-c.md](plan-c.md) / [plan-c2.md](plan-c2.md) / [plan-c3.md](plan-c3.md) (Plan C).
Records the decisions taken during planning, the evidence behind the plans' expected
values, and the things a reviewer should check before execution starts.

---

## 1. Decisions taken during planning

### D1 — Seed layout: **rejected**, 11× replication retained

Design §2 D1 recommended migrating map-action seeds to
`deploy/seed/shared/all/map-actions/` (99 files → 9) and flipping `script/groups.go:17`
to `seeder.NewFilesystemCatalogSourceWithShared`. **The reviewer rejected it.** Every plan
therefore keeps the PRD's layout: 11 version roots, byte-identical, authored under
`gms/83_1` and `cp`-replicated.

Consequences already folded into the plans:
- FR-1.6 stands as written — a byte-identity replication check across the 11 roots,
  implemented in Plan A Task A10's `checkMapActions`.
- `libs/atlas-seeder`'s `NewFilesystemCatalogSourceWithShared` is untouched; the one-time
  spurious "seed catalog drift detected" warning it would have caused does not arise.
- Every conversion task writes 11 copies. Plan B lands 297 files, Plan C 429.

### D2 — Plan split: three plans, one branch

Design §2 D7's split, at the user's request all three written up front. Plan A is a
correctness gate; Plan B is throughput; Plan C is where the risk is. Plan C exceeded a
readable single file and is split across three, which is a presentation split only —
`plan-c.md` §0/§0.1/§0.2 apply to all three parts.

### D3 — `transportState`: a **boolean** condition, not a widened wire

Design §2 D8 asked for a general `transportState` condition comparing a **named** route
state. `saga.ValidationConditionInput` (`libs/atlas-saga/validation.go:65-75`) has only
`Value int` and `Values []int` — no string operand — so a named state cannot travel.

Decision: add `transportInTransit`, a boolean condition shaped exactly like the existing
`transportAvailable` (`query-aggregator/validation/model.go:525-534`), which already
collapses the same five-state machine to a boolean. No wire change. Plan C Task C10.

Cosmic evidence that `in_transit` is the right state: `scripts/event/Boats.js` sets
`docked="true"` in `scheduleNew()` and `docked="false"` in `takeoff()`; `arrived()`
re-docks. `docked == "false"` is exactly the takeoff→arrived window.

### D4 — `areaInfo`: **one** additive wire widening, accepted

The exception to D3. `containsAreaInfo(area, "k=v;...")` is a substring test against a
free-form string (`client/Character.java:9782-9788`), and there is no boolean
reformulation. Plan C Task C20 adds
`ValueString string \`json:"valueString,omitempty"\`` to `ValidationConditionInput`.

Additive, `omitempty`, no existing consumer reads it, and `areaInfo` is its only user.
**A reviewer should confirm this before Task C20 runs** — it is the only cross-service
contract change in the whole task, and it touches a struct shared by
`atlas-npc-conversations`, `atlas-portal-actions`, `atlas-party-quests` and
`atlas-query-aggregator`. Task C20 Step 1 includes the build sweep that proves it is
additive.

### D5 — G8/G10/G11: convert the live parts only

Seven direction primitives have no implementation anywhere in Cosmic; the eight scripts
calling them abort at the first unresolved call. Full evidence and the per-script table
are in [plan-c.md](plan-c.md) §0. Six scripts are not converted; the `playSound` calls in
two of them are.

Deriving the seven primitives means eight unknown clientbound packets from the client
binary with no fname, no opcode and no reference behavior to check against. That is
`docs/packets/PROCESS.md` work and stays on issue #1624 with category #3.

### D6 — G5: build all three missing surfaces

`atlas-drops` has no DELETE, `atlas-reactors` has no reset or shuffle, and no service
offers a field reset. Plan C Tasks C16–C18 add all three, each with its acceptance test in
the owning service (the cross-service seam rule).

---

## 2. Corrections to the PRD and issue #1624

Each is applied in the task that owns it; collected here for review.

| # | Claim | Reality | Evidence | Owner |
|---|---|---|---|---|
| 1 | PRD §4.3: G6 `questProgress` is "schema enum only" | `step` is dropped at two layers and the aggregator rejects a step-less `questProgress` | design F1; `script/rest.go:34-39`, `script/evaluator.go:75-80`, `query-aggregator/validation/rest.go:238-245` | Plan A A2, A3 |
| 2 | PRD defect list has four items | There is a fifth: `spawn_monster` hard-codes `Instance: uuid.Nil` | design F3; `script/executor.go:183` vs `libs/atlas-constants/field/model.go:41` | Plan A A5 |
| 3 | PRD §4.3: G5 needs a `count_monster` capability | One use site, `map.countMonster(9100013) == 0`, which is `spawnIfAbsent` | `926000000.js`; `MapleMap.java:1145-1158` | Plan A A5/A7 (already shipped) |
| 4 | PRD §4.3: `reset_pq` belongs to a party-quest service | `resetPQ(n)` is `clearMapObjects + restoreMapSpawnPoints + instanceMapFirstSpawn` — a field reset | `MapleMap.java:3962-3975` | Plan C C18 |
| 5 | PRD §4.3: G5 needs a state-filtered `resetReactors` overload | No such Java overload; `926120300.js` computes the `state >= 7` filter in script and calls `resetReactors(List)` | `MapleMap.java:1545-1578`; `Reactor.java:100` | Plan C C17 |
| 6 | PRD §4.5: `iceCave` teaches skills 20000014–20000018 | `teachSkill(id, -1, 0, -1)` reaches `changeSkillLevel`'s `newLevel <= -1` branch, which **removes** the skill and deletes its row | `AbstractPlayerInteraction.java:927-944`; `Character.java:1815-1833` | Plan C C4 |
| 7 | PRD §4.5 / issue G1: `101000301`→200090010 | `101000301.js` calls `warpAhead(200000112)` | `scripts/map/onUserEnter/101000301.js` | Plan C C13 |
| 8 | PRD §4.3 G6b: `isCygnus()` needs an `in`-operator or job-family condition | `isCygnus()` is `job.getId() / 1000 == 1`, i.e. the band 1000–1999 — two AND-ed `jobId` conditions | `Character.java:5222-5224, 6144-6146` | Plan C C5 |
| 9 | PRD §4.4: the four dragon cutscenes are "the same shape as goArcher" | All four also call `lockUI()` before `showIntro`; the seeded `goArcher` has no `lock_ui` | `crash_Dragon.js` et al. | Plan B B3 |
| 10 | design §9.2 / PRD Q6: is `start_map_effect` the boat/crog primitive? | No. `startMapEffect` → `BLOW_WEATHER`; `crogBoatPacket` → `CONTI_MOVE`; `mapEffect` → `FIELD_EFFECT` mode 3. Three distinct opcodes | `PacketCreator.java:4263, 4639`; `MapEffect.java:43` | resolved, no task |
| 11 | PRD FR-3.1 / design §9.1: locate `cannon_tuto_02` | Exactly one occurrence in the whole Cosmic tree: the call in `cannon_tuto_01.js:5`. No script file, no Java, no resource | exhaustive grep | plan-c.md §0 |
| 12 | design D10: does `explorer_quest` reduce to `SetQuestProgress`/`StartQuest`? | No. It force-starts via NPC 9000066, appends to a deduplicated per-quest visited-map set, and writes the resulting **count** as progress | `MapScriptMethods.java:104-139` | Plan C C22 |

Also: FR-2.3 names only `108010301` for the `spawnIfAbsent` retrofit, but
`spaceGaGa_sMap` also carries an unguarded `spawn_monster` and would fail Plan A Task
A10's lint rule. Plan A Task A9 retrofits both.

---

## 3. What already exists that the plans deliberately reuse

Found during the survey; each removed work the design assumed would be needed.

- **`libs/atlas-script-core/condition`** already carries `step`, `worldId`, `channelId`
  and `includeEquipped` (`condition/model.go:8-17`, `builder.go:8-16`). Only `values` is
  missing. Design §3.2 implied more.
- **`saga.ValidationConditionInput` already has `Values []int`**
  (`libs/atlas-saga/validation.go:69`). Only map-actions' local `ConditionInput`
  (`validation/model.go:9-18`) lacked it.
- **`map-actions`' `validation.ConditionInput` already has `Step`/`WorldId`/`ChannelId`/
  `IncludeEquipped`** — they were simply never populated (`evaluator.go:75-80`).
- **The map-action schema's operation enum already matches the executor switch.** Nothing
  enforced it; Plan A Task A8's generator does.
- **`tools/catalog-lint` already declares both map-action subdomains**
  (`subdomains.go:18-19`) and already parses every seed file (`main.go:52-101`).
- **`libs/atlas-packet/field/clientbound/effect.go` already has
  `NewFieldEffectSound` (line 98) and `NewFieldEffectBackgroundMusic` (line 102)**, and
  `libs/atlas-packet/field/field_effect_body.go` already has `FieldEffectSoundBody`
  (line 51) and `FieldEffectBackgroundMusicBody` (line 63), both in production use
  (`party_quest/consumer.go:97`; `event/consumer.go:86`, `map/consumer.go:380,814,1030`).
  So `play_sound` and `change_music` need **no new packet**.
- **`libs/atlas-packet/field/clientbound/conti_move.go` is a verified `ContiMove` codec**
  documenting state 10 / subState 4 as `OnMoveField → CShip::AppearShip`
  (`conti_move.go:23-29`) — exactly Cosmic's `crogBoatPacket(true)`. The channel already
  resolves the pair per tenant through `writer.ContiMoveBody(writer.ContiMoveShow)`
  (`socket/writer/conti_move.go:35`). So the boat effect needs **no new packet** either,
  and no payload may carry a wire byte.
- **atlas-monsters already exposes `GET`/`DELETE`/`POST .../monsters`**
  (`world/resource.go:34-38`) and a Redis-backed registry with `GetInField`
  (`monster/processor.go:208-217`, `registry.go:411-436`) — everything Plan A Task A7's
  `spawnIfAbsent` guard needs.

Consequence: **Plan C adds no new packet.** Both Plan C files say so as a Global
Constraint, with the instruction to stop and report rather than derive one.

---

## 4. Scaffolding checklists the plans reference

### 4.1 Adding one saga action end to end (worked example: `SpawnMonster`)

| # | File | Anchor |
|---|---|---|
| 1 | `libs/atlas-saga/model.go` | `SpawnMonster Action = "spawn_monster"` — line 172 |
| 2 | `libs/atlas-saga/payloads.go` | `SpawnMonsterPayload` — lines 511-523 |
| 3 | `libs/atlas-saga/unmarshal.go` | `case SpawnMonster:` — lines 318-319 |
| 4 | `saga-orchestrator/saga/model.go` | action alias — line 158 |
| 5 | `saga-orchestrator/saga/model.go` | payload alias — line 341 |
| 6 | `saga-orchestrator/saga/model.go` | step-payload unmarshal case — lines 1251-1252 |
| 7 | `saga-orchestrator/saga/event_acceptance.go` | accepted-action entry — line 346 |
| 8 | `saga-orchestrator/saga/handler.go` | interface method decl — line 138 |
| 9 | `saga-orchestrator/saga/handler.go` | dispatch switch case — lines 919-920 |
| 10 | `saga-orchestrator/saga/handler.go` | handler body — lines 2066-2100 |
| 11 | `saga-orchestrator/saga/character_extractor.go` | `case SpawnMonsterPayload:` — line 47 |
| 12 | `saga-orchestrator/monster/processor.go` | domain client method — line 31 |
| 13 | `saga-orchestrator/monster/requests.go` | REST client — lines 11-27 |

Test shapes to copy: `libs/atlas-saga/unmarshal_test.go:253`
(`TestUnmarshalEvolvePetStep`) for a payload round-trip;
`saga-orchestrator/saga/event_acceptance_test.go:30` for the accepted-set assertion.
Note `SpawnMonster` itself has **no** dedicated unmarshal or handler test — not every
action does, which is why each Plan C task writes one for its new action rather than
assuming coverage exists.

### 4.2 Adding one aggregator condition (worked example: `transportAvailable`)

| # | File | Anchor |
|---|---|---|
| 1 | `libs/atlas-saga/validation.go` | `TransportAvailableCondition = "transportAvailable"` — line 35 |
| 2 | `query-aggregator/validation/model.go` | `ConditionType` alias — line 50 |
| 3 | `query-aggregator/validation/model.go` | `Evaluate` arm — lines 525-539 (check `EvaluateWithContext` at line 392 for a parallel switch) |
| 4 | `query-aggregator/validation/builder.go` | accepted-type switch in `SetType` — **line 210**; omitting this is the failure design §3.6 warns about, and it surfaces only at runtime |
| 5 | `query-aggregator/validation/builder.go` | `FromInput` validation — lines 334-338 |
| 6 | `query-aggregator/validation/builder.go` | `Validate()` validation — lines 417-420 |
| 7 | `query-aggregator/validation/rest.go` | REST-input validation arm — lines 271-274 |
| 8 | `query-aggregator/validation/context.go` | the getter, e.g. `GetTransportState` — lines 346-359; degrades to a sentinel rather than erroring |
| 9 | `query-aggregator/validation/{context.go,builder.go}` | the processor field threaded through — `context.go:51,74,98,417`; `builder.go:38,61,85,172`, auto-wired at `builder.go:85` |

`transportInTransit` (Plan C C10) reuses `GetTransportState` and needs no new getter.
`areaInfo` (Plan C C20) needs a new getter and a new REST client package.

---

## 5. Cosmic derivation — where the plans' literal values come from

The Cosmic reference server checkout is at the path recorded in the session that produced
these plans; re-derive with `<cosmic-root>/scripts/map/onUserEnter/` and
`<cosmic-root>/src/main/java/`. Every literal in Plan B and Plan C was read from those
files, not from the PRD or issue #1624.

Key native semantics, with citations:

| primitive | semantics | citation |
|---|---|---|
| `mapEffect(path)` | `FIELD_EFFECT` mode 3 | `PacketCreator.mapEffect` |
| `playSound(path)` | `FIELD_EFFECT` mode 4, broadcast to map | `AbstractPlayerInteraction.java:1062-1064` |
| `musicChange(song)` | `FIELD_EFFECT` mode 6 | `PacketCreator.java:4221` |
| `crogBoatPacket(true)` | `CONTI_MOVE`, `writeByte(10); writeByte(4)` | `PacketCreator.java:4639` |
| `startMapEffect(...)` | `BLOW_WEATHER` — a third, distinct opcode | `PacketCreator.java:4263` |
| `warpAhead(mapId)` | deferred warp consumed mid-transfer; resolves `getRandomPlayerSpawnpoint()` | `Character.java:1337-1339, 1365-1376, 1785-1790` |
| `docked` property | set true in `scheduleNew()`, false in `takeoff()`, true again in `arrived()` | `scripts/event/Boats.js:28,39` (and Trains/Cabin/Genie/Subway analogues) |
| `resetPQ(n)` | `clearMapObjects + restoreMapSpawnPoints + instanceMapFirstSpawn(n, true)` | `MapleMap.java:3962-3975, 3426` |
| `resetReactors()` / `(List)` | reset to state 0, `setAlive(true)`, broadcast `triggerReactor(r, 0)`; skips `forceDelayedRespawn()` reactors | `MapleMap.java:1545-1578` |
| `shuffleReactors()` | permutes reactor **positions** only | `MapleMap.java:1580-1598` |
| `clearDrops()` (no-arg) | whole map, owner id 0 in the removal packet | `MapleMap.java:3750-3764` |
| `countMonster(id)` | returns an `int`; the one use site compares `== 0` | `MapleMap.java:1145-1158` |
| `spawnNpc(id, pos, map)` | session/instance-scoped live map object; not persisted | `AbstractPlayerInteraction.java:962-973` |
| `containsNPC(id)` | live map object scan | `MapleMap.java:1682-1696` |
| `updateAreaInfo(area, info)` | **replaces** the stored string; persisted in the `area_info` table | `Character.java:326, 9790-9793, 7087-7092, 8540-8544` |
| `containsAreaInfo(area, info)` | **substring** test, not equality | `Character.java:9782-9788` |
| `teachSkill(id, -1, 0, -1)` | removes the skill and deletes its row | `AbstractPlayerInteraction.java:927-944`; `Character.java:1815-1833` |
| `isCygnus()` | `job.getId() / 1000 == 1`, i.e. 1000–1999 | `Character.java:5222-5224, 6144-6146` |
| `explorerQuest(q, name)` | force-start via NPC 9000066 → `addMedalMap(mapId)` dedup → write count as progress under `infoNumber` → compare to `infoEx(0)` | `MapScriptMethods.java:104-139` |
| `forceStartQuest(id)` | credits `NpcId.MAPLE_ADMINISTRATOR` = 9010000 | `AbstractPlayerInteraction.java:465-466`; `constants/id/NpcId.java:6` |

---

## 6. Task sizing notes

Two tasks are deliberately larger than Plan-Task §5a's ~6-file / one-service guideline.
Both are flagged in place with a split point:

- **Plan C Task C14 (`spawn_npc`)** touches four modules and creates two new packages
  (`atlas-maps/map/npc/` and `saga-orchestrator/npc_spawn/`). The two halves —
  "atlas-maps registry + REST" and "saga action + executor arm" — are independently
  testable and the task says to split there if the implementer hits the budget. Kept
  together because the seam test in atlas-maps is the acceptance criterion for the whole
  capability, and splitting hides that.
- **Plan C Task C20 (`areaInfo`)** touches five modules: new persistence in
  atlas-character, the orchestrator's write path, the aggregator condition, the wire
  widening, and the map-actions plumbing. It is the only task carrying a cross-service
  contract change, so it is kept as one review surface on purpose.

Every seed task exceeds six files by a wide margin (44–132), but each is the same
mechanical `cp` repeated across ten roots, which Plan-Task §5a explicitly exempts.

---

## 7. Open items a reviewer should settle before execution

1. **D4's wire widening** (§1) — confirm `ValueString` on
   `saga.ValidationConditionInput` before Plan C Task C20 runs.
2. **Rule-evaluation semantics.** `map_script_schema.json:21` says "first matching rule
   wins", but Plan C Task C23's `explorationPoint` needs *all* matching rules to run
   (Cosmic's `104000000` branch is a separate top-level `if` that fires alongside the
   range chain). Plan C Task C5 Step 3 establishes which the engine actually does, and
   Tasks C21 and C23 both gate on that finding. If the engine is first-match-wins, those
   two documents need restructuring — the plans say so and describe the alternative.
3. **Plan C Task C19 Step 2's saga-ordering question.** `926000000` emits `reset_field`
   then `spawn_monster`. If each executor operation becomes an independent saga, the two
   can race and the spawn may be cleared by its own reset. The task says to check and, if
   so, emit both steps in one saga.
4. **Plan A Task A7 Step 1.** `Create` returning `(Model{}, nil)` on a suppressed spawn —
   confirm what `handleCreateMonsterInMap` (`world/resource.go:207-231`) does with a zero
   `Model`, and whether it should become `204`.
5. **Plan C Task C22 Step 1.** Whether Atlas exposes a quest's `infoNumber` and
   `infoEx[0]`. If not, `explorer_quest` records the count but cannot send the completion
   comparison faithfully; the task says to state that rather than fabricate a threshold.

---

## 8. Execution order

1. **[plan.md](plan.md) — Plan A**, Tasks A1–A12, then the flagless `tools/verify.sh`.
   Nothing may be converted before this is green.
2. **[plan-b.md](plan-b.md) — Plan B**, Tasks B1–B7. 27 documents, no engine work.
3. **[plan-c.md](plan-c.md) → [plan-c2.md](plan-c2.md) → [plan-c3.md](plan-c3.md) —
   Plan C**, Tasks C1–C23 in order. The order is cheapest-first by design: executor arms
   for existing actions, then Plan-A-unblocked seeds, then operations on existing packets,
   then new actions with new consumers, then the three-service bulk-mutation surfaces,
   then the two capabilities needing new persistence.

Totals: 9 documents already seeded, 27 from Plan B, 39 from Plan C = **75 per root**
(74 `onUserEnter` + 1 `onFirstUserEnter`), × 11 roots. 6 of the PRD's 90 Cosmic scripts
are not converted and stay on issue #1624 alongside category #3's 9.
