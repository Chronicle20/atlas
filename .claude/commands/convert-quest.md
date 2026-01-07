---
description: Convert quest conversation JavaScript script to JSON state machine format
argument-hint: Path to quest script file (e.g., "services/atlas-npc-conversations/tmp/quest/quest_2013.js") or paste JavaScript code
---

You are an AI implementer converting MapleStory quest conversation scripts from JavaScript to a structured JSON state machine format.

## Reference Files

1. **Schema**: `services/atlas-npc-conversations/docs/quest_conversation_schema.json` - Your output MUST conform to this (read upfront)
2. **NPC Conversion Spec**: `docs/npc_conversation_conversion_spec.md` - Detailed conversion guidelines (read upfront)
3. **Map Reference**: `docs/Map.txt` - Map IDs to human-readable names (use Grep to look up specific IDs, DO NOT read entire file)
4. **NPC Reference**: `docs/NPC.txt` - NPC IDs to names (use Grep to look up specific IDs, DO NOT read entire file)
5. **Quest Reference**: `docs/Quest.txt` - Quest IDs to names (use Grep to look up specific IDs, DO NOT read entire file)

**IMPORTANT - Token Efficiency:**
Map.txt, NPC.txt, and Quest.txt are very large files (thousands of lines). Reading them in full is extremely wasteful. Instead:
- Only use Grep to search for specific IDs as you encounter them in the script
- Most scripts only reference a handful of maps/NPCs/quests, so targeted Grep searches are much more efficient

## Quest Conversation Structure

Quest scripts differ from NPC scripts in that they typically have TWO phases:
1. **Start Phase** (`start()` function) - Dialogue when quest is NOT_STARTED (accepting the quest)
2. **End Phase** (`end()` function) - Dialogue when quest is STARTED (completing the quest)

These map to the JSON structure:
- `startStateMachine` - For the start phase
- `endStateMachine` - For the end phase (optional if quest has no completion dialogue)

## Conversion Requirements

**CRITICAL - Preserve Original Script Exactly:**
- **DO NOT add any dialogue states that don't exist in the original script**
- **DO NOT invent or paraphrase dialogue text** - use EXACT text from `qm.sendOk()`, `qm.sendNext()`, etc.
- **DO NOT add confirmation messages** after actions unless they exist in the original
- If the original script calls `qm.dispose()` after an action, the JSON should end (nextState: null)
- Every dialogue state you create MUST correspond to an actual `qm.send*()` call in the original script

### 1. Analyze the JavaScript Script

Identify and understand:
- **Quest phases**: Look for `start()` and `end()` functions (or `function start` and `function end`)
- **Quest manager object**: Usually `qm` (similar to `cm` for NPCs)
- **Dialogue types**: `sendOk`, `sendNext`, `sendNextPrev`, `sendPrev`, `sendYesNo`, `sendAcceptDecline`
  - **IMPORTANT**: `sendSimple` with `#L0#`, `#L1#`, etc. tags → use `listSelection` state type
- **Quest operations**:
  - `qm.forceStartQuest()` → `start_quest` operation
  - `qm.forceCompleteQuest()` → `complete_quest` operation
- **Conditional logic**: Level checks, item requirements, job checks
- **Actions**: `gainItem`, `warp`, `gainMeso`, `gainExp`, etc.
- **Branching**: Choice destinations and flow control
- **When conversations end**: Look for `qm.dispose()` calls

### 2. Produce Valid JSON Output

- Extract the quest ID from the script filename or comments
- Extract the NPC ID if mentioned in comments or from `qm.getNpc()`
- Define `startStateMachine` for the `start()` function
- Define `endStateMachine` for the `end()` function (if it exists)
- Each state machine has its own `startState` and `states` array

### 3. State Types (from schema)

- **dialogue**: Present messages with choices (`sendOk`, `sendNext`, `sendNextPrev`, `sendPrev`, `sendYesNo`, `sendAcceptDecline`)
  - **Choice count requirements**:
    - `sendOk`: exactly 2 choices
    - `sendYesNo`: exactly 3 choices
    - `sendAcceptDecline`: exactly 3 choices
    - `sendNext`: exactly 2 choices
    - `sendNextPrev`: exactly 3 choices
    - `sendPrev`: exactly 2 choices
  - **CRITICAL - Required choice text values** (case-sensitive):
    - `sendOk`: `"Ok"` and `"Exit"`
    - `sendYesNo`: `"Yes"`, `"No"`, and `"Exit"`
    - `sendAcceptDecline`: `"Accept"`, `"Decline"`, and `"Exit"`
    - `sendNext`: `"Next"` and `"Exit"`
    - `sendNextPrev`: `"Previous"`, `"Next"`, and `"Exit"`
    - `sendPrev`: `"Previous"` and `"Exit"`
    - `listSelection`: Must include `"Exit"` choice
  - Terminal choices (nextState: null) should use text `"Exit"`
- **genericAction**: Execute logic/validation (item checks, start/complete quest, etc.)
  - All outcomes MUST have `conditions` array (can be empty)
- **listSelection**: Dynamic choice lists

### 4. Quest-Specific Operations

Key operations for quest scripts:
- **start_quest**: Start a quest for the character
  - Params: `questId` (string, optional - defaults from context), `npcId` (string, optional - defaults to conversation NPC)
- **complete_quest**: Complete a quest for the character
  - Params: `questId` (string, optional - defaults from context), `npcId` (string, optional - defaults to conversation NPC)
- **award_item**: Award an item
  - Params: `itemId` (string), `quantity` (string)
- **destroy_item**: Remove items
  - Params: `itemId` (string), `quantity` (string)
- **award_mesos**: Award mesos
  - Params: `amount` (string)
- **award_exp**: Award experience
  - Params: `amount` (string)
- **warp_to_map**: Warp to specific map
  - Params: `mapId` (string), `portalId` (string)

### 5. Condition Types

Common conditions for quest scripts:
- **level**: Check character level (operators: =, >, <, >=, <=)
- **jobId**: Check character's job (operators: =, >, <, >=, <=)
- **item**: Check item possession (operators: =, >, <, >=, <=, requires `referenceId` field)
- **meso**: Check meso amount (operators: =, >, <, >=, <=)
- **questStatus**: Check quest status (operators: =, requires `referenceId` with quest ID)
  - Values: 0 = NOT_STARTED, 1 = STARTED, 2 = COMPLETED
- **questProgress**: Check quest progress (operators: =, >, <, >=, <=, requires `referenceId` and `step` fields)

### 6. Required Validations

- ✅ All dialogue states have correct choice counts
- ✅ Choice text uses exact required values (case-sensitive)
- ✅ `nextState` is `null` (not string) when ending conversation
- ✅ Empty conditions arrays are `[]`, not omitted
- ✅ Operation params use correct names
- ✅ Output conforms to quest_conversation_schema.json (validate before finalizing)
- ✅ Both `startStateMachine` and `endStateMachine` have their own `startState` and `states`

## Task

The script to convert: **$ARGUMENTS**

**Steps:**
1. Read schema file `services/atlas-npc-conversations/docs/quest_conversation_schema.json`
2. Read conversion spec `docs/npc_conversation_conversion_spec.md`
3. **Read validation model AND operation types**:
   - Read `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` to get supported condition types
   - Read `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` to get supported operation types
4. If a file path is provided, read the script file; otherwise use the provided code
5. Analyze the script thoroughly:
   - Identify `start()` and `end()` functions
   - Count all `qm.send*()` calls in each function - each one becomes a dialogue state
   - Identify where `qm.dispose()` is called - those paths end (nextState: null)
   - Note map IDs, NPC IDs, and quest IDs referenced
6. Use Grep to look up specific IDs:
   - Map IDs in `docs/Map.txt` (pattern: `^<mapId> - `)
   - NPC IDs in `docs/NPC.txt` (pattern: `^<npcId> - `)
   - Quest IDs in `docs/Quest.txt` (pattern: `^<questId> - `)
7. Convert to JSON following all requirements above:
   - `start()` function → `startStateMachine`
   - `end()` function → `endStateMachine`
8. **VALIDATE - Script Accuracy**: Verify each dialogue state has a corresponding `qm.send*()` in the original
9. **VALIDATE - Implementation**: Check that all conditions and operations used are actually implemented
10. Determine appropriate output filename based on quest ID (e.g., `quest_2013.json`)
11. Validate against the schema
12. **ONLY if all validations pass**: Write the output file to `services/atlas-npc-conversations/conversations/quests/` directory
13. **VALIDATE - Build Check**: Verify the service still compiles:
    - Run `go build` in `services/atlas-npc-conversations/atlas.com/npc` directory
14. Report completion with summary of states created and build status

**Example Quest Script Structure:**
```javascript
/* Quest: 2013 - Mai's First Training
 * NPC: 1002000 - Mai
 */

var status = -1;

function start(mode, type, selection) {
    if (mode == 1) {
        status++;
    } else {
        status--;
    }
    if (status == 0) {
        qm.sendAcceptDecline("Are you ready for your first training?");
    } else if (status == 1) {
        qm.forceStartQuest();
        qm.sendOk("Great! Go defeat 10 snails.");
        qm.dispose();
    }
}

function end(mode, type, selection) {
    if (mode == 1) {
        status++;
    } else {
        status--;
    }
    if (status == 0) {
        if (!qm.haveItem(4000000, 10)) {
            qm.sendOk("You need 10 Snail Shells.");
            qm.dispose();
        } else {
            qm.sendNext("Well done! Let me take those shells.");
        }
    } else if (status == 1) {
        qm.gainItem(4000000, -10);
        qm.forceCompleteQuest();
        qm.gainExp(100);
        qm.sendOk("Here's your reward!");
        qm.dispose();
    }
}
```

**Converts to:**
```json
{
  "questId": 2013,
  "npcId": 1002000,
  "questName": "Mai's First Training",
  "startStateMachine": {
    "startState": "askAccept",
    "states": [
      {
        "id": "askAccept",
        "type": "dialogue",
        "dialogue": {
          "dialogueType": "sendAcceptDecline",
          "text": "Are you ready for your first training?",
          "choices": [
            {"text": "Accept", "nextState": "startQuest"},
            {"text": "Decline", "nextState": null},
            {"text": "Exit", "nextState": null}
          ]
        }
      },
      {
        "id": "startQuest",
        "type": "genericAction",
        "genericAction": {
          "operations": [
            {"type": "start_quest", "params": {}}
          ],
          "outcomes": [
            {"conditions": [], "nextState": "confirmStart"}
          ]
        }
      },
      {
        "id": "confirmStart",
        "type": "dialogue",
        "dialogue": {
          "dialogueType": "sendOk",
          "text": "Great! Go defeat 10 snails.",
          "choices": [
            {"text": "Ok", "nextState": null},
            {"text": "Exit", "nextState": null}
          ]
        }
      }
    ]
  },
  "endStateMachine": {
    "startState": "checkItems",
    "states": [
      {
        "id": "checkItems",
        "type": "genericAction",
        "genericAction": {
          "operations": [],
          "outcomes": [
            {
              "conditions": [
                {"type": "item", "operator": "<", "value": "10", "referenceId": "4000000"}
              ],
              "nextState": "missingItems"
            },
            {
              "conditions": [],
              "nextState": "hasItems"
            }
          ]
        }
      },
      {
        "id": "missingItems",
        "type": "dialogue",
        "dialogue": {
          "dialogueType": "sendOk",
          "text": "You need 10 Snail Shells.",
          "choices": [
            {"text": "Ok", "nextState": null},
            {"text": "Exit", "nextState": null}
          ]
        }
      },
      {
        "id": "hasItems",
        "type": "dialogue",
        "dialogue": {
          "dialogueType": "sendNext",
          "text": "Well done! Let me take those shells.",
          "choices": [
            {"text": "Next", "nextState": "completeQuest"},
            {"text": "Exit", "nextState": null}
          ]
        }
      },
      {
        "id": "completeQuest",
        "type": "genericAction",
        "genericAction": {
          "operations": [
            {"type": "destroy_item", "params": {"itemId": "4000000", "quantity": "10"}},
            {"type": "complete_quest", "params": {}},
            {"type": "award_exp", "params": {"amount": "100"}}
          ],
          "outcomes": [
            {"conditions": [], "nextState": "reward"}
          ]
        }
      },
      {
        "id": "reward",
        "type": "dialogue",
        "dialogue": {
          "dialogueType": "sendOk",
          "text": "Here's your reward!",
          "choices": [
            {"text": "Ok", "nextState": null},
            {"text": "Exit", "nextState": null}
          ]
        }
      }
    ]
  }
}
```

Begin conversion now.
