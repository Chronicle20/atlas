# Design: `start_quest` operation for atlas-portal-actions

## Problem

Portal scripts that call `pi.forceStartQuest(questId)` (e.g., `services/atlas-npc-conversations/tmp/portal/foxLaidy_map.js`) cannot be converted to the JSON rules format because atlas-portal-actions has no quest-mutation operation. The schema's operation enum (`services/atlas-portal-actions/docs/portal_script_schema.json`, lines 157-171) lists 13 operations, none of which start a quest.

The same capability already exists in atlas-npc-conversations as the `start_quest` operation (`services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:1755`), which emits a `saga.StartQuest` step with a `saga.StartQuestPayload`. atlas-portal-actions can adopt the same operation name and saga payload.

## Goal

Add a `start_quest` operation to atlas-portal-actions so portal scripts can dispatch quest starts. Validate the change end-to-end by converting `foxLaidy_map.js` to JSON.

## Non-goals

- Modifying `saga.StartQuestPayload` (already supports the portal use case — `NpcId: 0` is acceptable).
- Adding unit tests for the new executor (none of the 12 existing executors have unit tests; symmetry beats one-off scaffolding).
- Converting portal scripts beyond `foxLaidy_map.js`.
- Surfacing quest-start failures to the player (the JS API was fire-and-forget, atlas-quests handles "already started" idempotently).

## Architecture decision: independent saga per operation

Each portal operation in `executor.go` calls `e.sagaP.Create(s)` with a single-step saga. Operations are not chained into a multi-step saga; they are dispatched in script order but downstream handlers run asynchronously.

For `foxLaidy_map.js`'s else branch, this means `start_quest` and `warp` will fire as two independent sagas. The warp can complete before atlas-quests finishes processing `StartQuest`. We accept this:

- atlas-quests' `StartQuest` handler is non-blocking from the player's perspective and is force-style (no requirement check), so failures are effectively unreachable for this use case.
- Adding multi-step saga sequencing only for `start_quest` would diverge from the established pattern across all 12 existing operations.
- The visible UX impact is sub-second async lag; the post-warp client state catches up on the next quest packet.

Rejected alternatives:

- **Multi-step saga (`start_quest` then `warp`)** — introduces saga-orchestration complexity for one script.
- **Independent saga + `failureMessage` like `start_instance_transport`** — addresses a failure mode that doesn't occur for force-start.

## Schema change

**File**: `services/atlas-portal-actions/docs/portal_script_schema.json`

1. Add `"start_quest"` to the operation `type` enum (line 157-171). Existing enum order is not alphabetical; append at the end of the list.
2. Append to the `allOf` block:

```json
{
  "if": { "properties": { "type": { "const": "start_quest" } } },
  "then": {
    "properties": {
      "params": {
        "type": "object",
        "required": ["questId"],
        "properties": {
          "questId": { "type": "string", "description": "The quest ID to start" },
          "npcId":   { "type": "string", "description": "Originating NPC ID (default: 0 — portals have no NPC)" }
        }
      }
    }
  }
}
```

No `force` flag — `saga.StartQuestPayload` does not expose one, and the NPC `start_quest` case does not either. The legacy JS verb `forceStartQuest` maps to `start_quest`; "force" was a name-only artifact.

## Executor change

**File**: `services/atlas-portal-actions/atlas.com/portal/script/executor.go`

1. Add a case to the dispatch switch in `ExecuteOperation`, before the `default` arm:

   ```go
   case "start_quest":
       return e.executeStartQuest(f, characterId, op)
   ```

2. New function modelled on `executeApplyConsumableEffect`:

   ```go
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

       var npcId uint64 = 0
       if npcIdStr, hasNpcId := params["npcId"]; hasNpcId {
           npcId, err = strconv.ParseUint(npcIdStr, 10, 32)
           if err != nil {
               return fmt.Errorf("invalid npcId [%s]: %w", npcIdStr, err)
           }
       }

       e.l.Debugf("Starting quest [%d] for character [%d] (npcId=%d)", questId, characterId, npcId)

       s := saga.NewBuilder().
           SetSagaType(saga.InventoryTransaction).
           SetInitiatedBy("portal-action-start-quest").
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

No `action.PendingAction` registration — there is no failure message to surface.

## Tests

No new dedicated unit tests for `executeStartQuest`. None of the 12 existing executor cases have unit tests, and adding scaffolding only for `start_quest` would signal it is special when it is not.

Verification done at higher levels:

- `go build ./...` in `services/atlas-portal-actions` — confirms the new code compiles.
- `go test ./...` in `services/atlas-portal-actions` — confirms existing tests (notably the `entity_test.go` rule-parsing tests) still pass with the schema change.
- Convert `foxLaidy_map.js` to `services/atlas-portal-actions/scripts/portals/portal_foxLaidy_map.json`, verify it loads through the existing script loader without schema errors.

## Doc updates

1. **`services/atlas-portal-actions/docs/portal_script_schema.json`** — schema change above.
2. **`.claude/commands/convert-portal.md`** — three updates:
   - Add row to the supported operations table:
     ```markdown
     | `start_quest` | `pi.forceStartQuest(questId)` | `questId`, `npcId` (optional) |
     ```
   - Remove `forceStartQuest` from the "NOT YET SUPPORTED" / unsupported-features list (it is not currently listed there explicitly, but the conversion-time check should treat it as supported).
   - Add a "Pattern E: Quest Start (gated by quest_status)" example showing the foxLaidy-style script — quest condition + `start_quest` operation in the matching rule.

## Acceptance criteria

1. Schema enum + conditional params block updated; `go test ./...` in atlas-portal-actions still green.
2. `executeStartQuest` added; switch dispatches to it; `go build ./...` and `go test ./...` pass.
3. `.claude/commands/convert-portal.md` updated (table row + new pattern example).
4. `foxLaidy_map.js` converted to `portal_foxLaidy_map.json`; JSON parses through the loader without errors.

## Out of scope

- Unit tests for `executeStartQuest`.
- Any change to `saga.StartQuestPayload`.
- Multi-step-saga sequencing of `start_quest` and `warp`.
- Conversion of portal scripts beyond `foxLaidy_map.js`.
- Surfacing quest-start failures to the player.
