# Task-020 First-Job AP Rebalance — Context

Quick reference for engineers (or subagents) executing this task. Source of truth is `design.md`; this file exists so a reader who opens the repo cold can find everything in one shot.

## 1. The big picture

Five Explorer first-job NPC conversation scripts and five Cygnus first-job quest scripts currently gate advancement with a class-minimum stat check (e.g., `dexterity >= 25`). On a vanilla v83 client these gates are unsatisfiable because beginner AP auto-allocation pours everything into STR. This task replaces the gate with a server-side rebalance: zero the four primary stats, raise the target stat(s) to the class floor, return the reclaimed pool as unallocated AP.

A new operation `rebalance_ap` is introduced into the NPC/quest state machine, dispatched as a saga step to atlas-character, where a new processor method `RebalanceAPAndEmit` performs the arithmetic and emits a single multi-stat `STAT_CHANGED` event.

## 2. Affected services / packages

| Service / package | What changes |
|---|---|
| `libs/atlas-saga` | Add `RebalanceAP` action constant, `RebalanceAPPayload` struct, unmarshal case. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` | Re-export `RebalanceAP` action and `RebalanceAPPayload`; add unmarshal case. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` | Add `case RebalanceAP` in action-to-handler lookup; add `handleRebalanceAP`. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/character/kafka.go` | Add `CommandRebalanceAP` constant and `RebalanceAPCommandBody` type. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/producer.go` | Add `RebalanceAPProvider`. |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/character/processor.go` + `mock/processor.go` | Add `RebalanceAPAndEmit` and `RebalanceAP(mb)` to Processor interface, impl, and mock. |
| `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go` | Add `CommandRebalanceAP` constant and `RebalanceAPCommandBody` type. |
| `services/atlas-character/atlas.com/character/character/processor.go` | Add `computeRebalance` helper, `RebalanceAP(mb)` and `RebalanceAPAndEmit`; extend Processor interface. |
| `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` | Add `handleRebalanceAP` and register it in `InitHandlers`. |
| `services/atlas-npc-conversations/atlas.com/npc/saga/model.go` | Re-export `RebalanceAP` action and `RebalanceAPPayload`; re-export `RebalanceTarget` if defined. |
| `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` | Add `case "rebalance_ap":` to `createStepForOperation`. |
| `services/atlas-npc-conversations/conversations/npc/` | Edit `npc_1012100.json` (Bowman), `npc_1022000.json` (Warrior), `npc_1032001.json` (Magician), `npc_1052001.json` (Thief), `npc_1090000.json` (Pirate). |
| `services/atlas-npc-conversations/conversations/quests/` | Edit `quest_20101.json` … `quest_20105.json`. |
| `docs/npc_conversation_conversion_spec.md` | Document `rebalance_ap` and `reset_stats`; add first-job guidance. |

## 3. Correct NPC → class mapping

The design file table has a swap between `npc_1022000` and `npc_1032001`. Verified against each file's `change_job → jobId` operation:

| File | jobId target | Class | Rebalance target(s) |
|---|---|---|---|
| `npc_1012100.json` | 300 | Bowman | DEX 25 |
| `npc_1022000.json` | **100** | **Warrior** | **STR 35** |
| `npc_1032001.json` | **200** | **Magician** | **INT 20** |
| `npc_1052001.json` | 400 | Thief | DEX 25 |
| `npc_1090000.json` | 500 | Pirate | DEX 20 |

`npc_1032001.json` (Magician) is the only Explorer NPC with no existing stat-check gate — it only checks `level < 8`. For that file, the plan only adds `rebalance_ap` and removes `reset_stats`; no gate to remove.

Quest file → class mapping is self-describing in each file:

| File | jobId | Class | Rebalance target(s) |
|---|---|---|---|
| `quest_20101.json` | 1100 | Dawn Warrior | STR 35 |
| `quest_20102.json` | 1200 | Blaze Wizard | INT 20 |
| `quest_20103.json` | 1300 | Wind Archer | DEX 25 |
| `quest_20104.json` | 1400 | Night Walker | LUK 25 |
| `quest_20105.json` | 1500 | Thunder Breaker | STR 20, DEX 20 |

## 4. Saga step plumbing — how ChangeJob flows today

Read this once if you haven't touched the saga system before. The equivalent plumbing for `RebalanceAP` follows the exact same loop:

1. `operation_executor.go:1227-1248` — NPC conversation batches a `change_job` op into a saga step with `saga.ChangeJobPayload`.
2. Saga is emitted to atlas-saga-orchestrator via Kafka (outside this task's scope — reused as-is).
3. `saga-orchestrator/saga/handler.go:1086` — `handleChangeJob` unpacks `ChangeJobPayload` and calls `h.charP.ChangeJobAndEmit(...)`.
4. `saga-orchestrator/character/processor.go` — `ChangeJobAndEmit` emits a Kafka command (`CHANGE_JOB` on `COMMAND_TOPIC_CHARACTER`) with body `ChangeJobCommandBody`.
5. `atlas-character/kafka/consumer/character/consumer.go:128-137` — `handleChangeJob` receives the command and calls `character.NewProcessor(...).ChangeJobAndEmit(...)`.
6. `atlas-character/character/processor.go:438-466` — `ChangeJobAndEmit` → `ChangeJob(mb)` does the DB write and emits `JOB_CHANGED` + `STAT_CHANGED` events.

`RebalanceAP` adds a new row to every one of those six layers.

## 5. Rebalance algorithm (one screen)

```
reclaimed = max(0, STR-4) + max(0, DEX-4) + max(0, INT-4) + max(0, LUK-4)
new STR = new DEX = new INT = new LUK = 4
for t in targets: new[t.stat] = t.floor
cost = Σ (t.floor - 4)
new unallocated = old unallocated + reclaimed - cost
if new unallocated < 0: error
```

Canonical test row (PRD §10.7 reference, Pirate): input `(53, 9, 4, 4, 0)`, targets `[{DEX, 20}]`, expected `(4, 20, 4, 4, 38)`.

## 6. Key patterns to reuse

- **Pattern for a new processor method:** see `ResetStats`/`ResetStatsAndEmit` in `services/atlas-character/atlas.com/character/character/processor.go:1784-1844`. This is structurally the nearest neighbor to `RebalanceAP` — same transactional shape, same multi-stat `STAT_CHANGED` emission.
- **Pattern for a new saga action constant & payload:** see how `ResetStats` is added in `libs/atlas-saga/model.go:69`, `libs/atlas-saga/payloads.go:189-194`, and `libs/atlas-saga/unmarshal.go:144-149`.
- **Pattern for a new orchestrator handler:** see `handleResetStats` in `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:2193-2209`.
- **Pattern for a new atlas-character consumer handler:** see `handleResetStats` in `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:382-391` and its registration at `:86`.
- **Pattern for a new operation-executor case:** see the `reset_stats` case in `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:2189-2199` and the parameterized `change_job` case at `:1227-1248`.
- **Pattern for a multi-stat STAT_CHANGED event:** see `ResetStats` emission at `processor.go:1841` — pass `[]stat.Type{stat.TypeAvailableAP, stat.TypeStrength, stat.TypeDexterity, stat.TypeIntelligence, stat.TypeLuck}` and a `values map[string]interface{}`.

## 7. Stat type constants

`libs/atlas-constants/stat/constants.go:5-28`

- `stat.TypeStrength = "STRENGTH"`
- `stat.TypeDexterity = "DEXTERITY"`
- `stat.TypeIntelligence = "INTELLIGENCE"`
- `stat.TypeLuck = "LUCK"`
- `stat.TypeAvailableAP = "AVAILABLE_AP"`

Character entity fields (`services/atlas-character/atlas.com/character/character/entity.go:15-49`) are all `uint16`:
- `Strength`, `Dexterity`, `Intelligence`, `Luck`, `AP`

Model getters: `c.Strength()`, `c.Dexterity()`, `c.Intelligence()`, `c.Luck()`, `c.AP()`.
Model setters (via `dynamicUpdate`): `SetStrength`, `SetDexterity`, `SetIntelligence`, `SetLuck`, `SetAP`.

## 8. JSON condition syntax

- **NPC JSONs** use `{"type": "dexterity", "operator": "<", "value": "25"}` (direct stat-name type).
- **Quest JSONs** use `{"type": "stat", "operator": "<", "value": "25", "referenceId": "dex"}` (generic `stat` with a ref).

Both forms exist in the tree and are evaluated by the validation layer — this task does not change condition evaluation, it removes the conditions outright.

## 9. Out-of-scope reminders

- No Aran script — does not exist in the repo.
- No beginner auto-allocation changes.
- No `RequestDistributeAp` changes.
- No second/third/fourth job advancement changes.
- No tenant-configurable strict-gate mode.
- No advancement-grant AP (“+5 on job change”) — deferred indefinitely unless evidence surfaces.
