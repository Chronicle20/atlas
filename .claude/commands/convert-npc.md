---
description: Convert NPC conversation JavaScript script to JSON state machine format
argument-hint: Path to NPC script file (e.g., "services/atlas-npc-conversations/tmp/2003.js") or paste JavaScript code
---

You are an AI implementer converting MapleStory NPC conversation scripts from JavaScript to a structured JSON state machine format.

## Reference Files

1. **Schema**: `services/atlas-npc-conversations/docs/npc_conversation_schema.json` - Your output MUST conform to this (read upfront)
2. **Conversion Spec**: `docs/npc_conversation_conversion_spec.md` - Detailed conversion guidelines (read upfront)
3. **Map Reference**: `docs/Map.txt` - Map IDs to human-readable names (use Grep to look up specific IDs, DO NOT read entire file)
4. **NPC Reference**: `docs/NPC.txt` - NPC IDs to names (use Grep to look up specific IDs, DO NOT read entire file)

**IMPORTANT - Token Efficiency:**
Map.txt and NPC.txt are very large files (thousands of lines). Reading them in full is extremely wasteful. Instead:
- Only use Grep to search for specific IDs as you encounter them in the script
- Most scripts only reference a handful of maps/NPCs, so targeted Grep searches are much more efficient

## Conversion Requirements

**🚨 CRITICAL - Preserve Original Script Exactly:**
- **DO NOT add any dialogue states that don't exist in the original script**
- **DO NOT invent or paraphrase dialogue text** - use EXACT text from `cm.sendOk()`, `cm.sendNext()`, etc.
- **DO NOT add confirmation messages** after actions like warps, item grants, etc. unless they exist in the original
- If the original script calls `cm.dispose()` after an action, the JSON should end (nextState: null)
- If the original script has NO dialogue after an action, don't add one
- Every dialogue state you create MUST correspond to an actual `cm.send*()` call in the original script

### 1. Analyze the JavaScript Script

Identify and understand:
- **Dialogue types**: `sendOk`, `sendNext`, `sendNextPrev`, `sendPrev`, `sendYesNo`, `sendAcceptDecline`
  - **IMPORTANT**: `sendSimple` in scripts with `#L0#`, `#L1#`, etc. tags → use `listSelection` state type (NOT dialogue)
  - Pattern: `#L<number>#<choice text>#l` indicates list items
  - Extract title (text before first `#L`) and parse each list item
- **Conditional logic**: Job checks, meso/item requirements
- **Actions**: `gainItem`, `warp`, `gainMeso`, `useItem`, etc.
  - **Note**: `cm.useItem(itemId)` applies item effects without consuming → use `apply_consumable_effect` operation
- **Branching**: Choice destinations and flow control
- **When conversations end**: Look for `cm.dispose()` calls - these indicate the conversation ends

### 2. Produce Valid JSON Output

- Extract the NPC ID from the script filename or comments
- Define a clear `startState`
- Create a `states` array with proper state objects
- Use context propagation for dynamic values (destination, cost, itemId, etc.)

### 3. State Types (from schema)

- **dialogue**: Present messages with choices (`sendOk`, `sendNext`, `sendNextPrev`, `sendPrev`, `sendYesNo`, `sendAcceptDecline`)
  - **Choice count requirements**:
    - `sendOk`: exactly 2 choices
    - `sendYesNo`: exactly 3 choices
    - `sendAcceptDecline`: exactly 3 choices
    - `sendNext`: exactly 2 choices
    - `sendNextPrev`: exactly 3 choices
    - `sendPrev`: exactly 2 choices
  - **CRITICAL - Required choice text values** (case-sensitive, must match exactly):
    - `sendOk`: `"Ok"` and `"Exit"` (note: lowercase 'k')
    - `sendYesNo`: `"Yes"`, `"No"`, and `"Exit"`
    - `sendAcceptDecline`: `"Accept"`, `"Decline"`, and `"Exit"`
    - `sendNext`: `"Next"` and `"Exit"`
    - `sendNextPrev`: `"Previous"`, `"Next"`, and `"Exit"`
    - `sendPrev`: `"Previous"` and `"Exit"`
    - `listSelection`: Must include `"Exit"` choice
  - **When to use sendNext vs sendNextPrev vs sendPrev**:
    - Use `sendNext` for first page of multi-page dialogue or single-page dialogue
    - Use `sendNextPrev` for middle pages that allow both forward and backward navigation
    - Use `sendPrev` for final page that only allows backward navigation (or ends conversation)
  - Terminal choices (nextState: null) should use text `"Exit"`
- **genericAction**: Execute logic/validation (meso checks, warps, item operations)
  - All outcomes MUST have `conditions` array (can be empty)
- **craftAction**: Define crafting with materials, quantities, meso cost
- **listSelection**: Dynamic choice lists that populate context
  - MUST include Exit choice
- **askStyle**: Present cosmetic style selection interface (hair/face/skin)
  - Used for hair salons, face shops, skin color NPCs
  - Requires pre-populated styles in context (via local:generate_* operations)
  - Fields: `text` (prompt), `stylesContextKey` (context key with styles array), `contextKey` (where to store selection), `nextState`

### 4. Operation Types and Parameters

Common operations in `genericAction` states:
- **warp_to_map**: Warp to specific map and portal
  - Params: `mapId` (string), `portalId` (string)
- **award_item**: Award an item
  - Params: `itemId` (string), `quantity` (string)
- **award_mesos**: Award mesos
  - Params: `amount` (string), `actorId` (optional), `actorType` (optional, default "NPC")
- **award_exp**: Award experience
  - Params: `amount` (string), `type` (optional, default "WHITE"), `attr1` (optional, default 0)
- **award_level**: Award character levels
  - Params: `amount` (string)
- **destroy_item**: Remove items
  - Params: `itemId` (string), `quantity` (string)
- **change_job**: Change character job
  - Params: `jobId` (string)
- **change_hair**: Change character hair style (via saga)
  - Params: `styleId` (string, can use context like `{context.selectedHair}`)
- **change_face**: Change character face style (via saga)
  - Params: `styleId` (string, can use context like `{context.selectedFace}`)
- **change_skin**: Change character skin color (via saga)
  - Params: `styleId` (string, can use context like `{context.selectedSkin}`)
- **increase_buddy_capacity**: Increase buddy list capacity
  - Params: `amount` (string, byte value)
- **gain_closeness**: Increase pet closeness/intimacy
  - Params: `petId` (string, uint32) or `petIndex` (string, int8), `amount` (string, uint16)
- **create_skill**: Create a new skill for character
  - Params: `skillId` (string), `level` (string, optional), `masterLevel` (string, optional)
- **update_skill**: Update an existing skill
  - Params: `skillId` (string), `level` (string, optional), `masterLevel` (string, optional)
- **warp_to_random_portal**: Warp to random portal in map
  - Params: `mapId` (string)
- **spawn_monster**: Spawn monsters at a location (foothold resolved automatically by saga-orchestrator)
  - Params: `monsterId` (string), `x` (string), `y` (string), `count` (string, optional, default "1"), `team` (string, optional, default "0")
- **complete_quest**: Complete a quest for the character (stub - no quest service yet)
  - Params: `questId` (string), `npcId` (string, optional - defaults to conversation NPC)
- **start_quest**: Start a quest for the character (stub - no quest service yet)
  - Params: `questId` (string), `npcId` (string, optional - defaults to conversation NPC)
- **apply_consumable_effect**: Apply consumable item effects without consuming from inventory
  - Params: `itemId` (string)
  - Used for NPC-initiated buffs (e.g., Shinsoo's blessing)
  - Applies all item effects (HP/MP recovery, stat buffs) via atlas-consumables
  - **JavaScript mapping**: `cm.useItem(itemId)` → `apply_consumable_effect`

**Local Operations** (executed within npc-conversations service):
- **local:generate_hair_styles**: Generate available hair styles for character
  - Params: `baseStyles`, `genderFilter`, `preserveColor`, `validateExists`, `excludeEquipped`, `outputContextKey`
- **local:generate_hair_colors**: Generate available hair colors for character
  - Params: `colors`, `validateExists`, `excludeEquipped`, `outputContextKey`
- **local:generate_face_styles**: Generate available face styles for character
  - Params: `baseStyles`, `genderFilter`, `validateExists`, `excludeEquipped`, `outputContextKey`
- **local:select_random_cosmetic**: Randomly select from a styles array
  - Params: `stylesContextKey`, `outputContextKey`
- **local:fetch_map_player_counts**: Fetch player counts for multiple maps
  - Params: `mapIds` (comma-separated map IDs, supports context references)
  - Stores results in context with keys: `playerCount_{mapId}`
- **local:log**: Log an informational message
  - Params: `message` (string, supports context references)
- **local:debug**: Log a debug message
  - Params: `message` (string, supports context references)

### 5. Condition Types

**IMPORTANT:** Condition types must match what's implemented in query-aggregator. Read `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` to verify supported types.

Common conditions in `outcomes`:
- **jobId**: Check character's job (operators: =, >, <, >=, <=)
- **meso**: Check meso amount (operators: =, >, <, >=, <=)
- **mapId**: Check current map (operators: =)
- **fame**: Check fame level (operators: =, >, <, >=, <=)
- **gender**: Check character's gender (operators: =) - 0 = male, 1 = female
- **level**: Check character level (operators: =, >, <, >=, <=)
- **reborns**: Check rebirth count (operators: =, >, <, >=, <=)
- **dojoPoints**: Check Mu Lung Dojo points (operators: =, >, <, >=, <=)
- **vanquisherKills**: Check vanquisher kill count (operators: =, >, <, >=, <=)
- **gmLevel**: Check GM level (operators: =, >, <, >=, <=)
- **guildId**: Check guild ID (operators: =, >, <, >=, <=) - 0 = not in guild
- **guildLeader**: Check if guild leader (operators: =) - 0 = not leader, 1 = is leader
- **guildRank**: Check guild rank (operators: =, >, <, >=, <=)
- **questStatus**: Check quest status (operators: =, >, <, >=, <=, requires `referenceId` field)
  - For quest completion: use value "3" (COMPLETED enum value from quest/model.go)
- **questProgress**: Check quest progress (operators: =, >, <, >=, <=, requires `referenceId` and `step` fields)
- **hasUnclaimedMarriageGifts**: Check for unclaimed marriage gifts (operators: =) - 0 = false, 1 = true
- **strength**: Check strength stat (operators: =, >, <, >=, <=)
- **dexterity**: Check dexterity stat (operators: =, >, <, >=, <=)
- **intelligence**: Check intelligence stat (operators: =, >, <, >=, <=)
- **luck**: Check luck stat (operators: =, >, <, >=, <=)
- **buddyCapacity**: Check buddy list capacity (operators: =, >, <, >=, <=)
- **petCount**: Check number of pets (operators: =, >, <, >=, <=)
- **mapCapacity**: Check player count in map (operators: =, >, <, >=, <=, requires `referenceId` with map ID)
- **item**: Check item possession (operators: =, >, <, >=, <=, requires `referenceId` field)
- **transportAvailable**: Check if a transport (train, boat, genie, etc.) is available for boarding (operators: =, requires `referenceId` with destination map ID)
  - Value "0" = transport unavailable (already departed/travelling)
  - Value "1" = transport available for boarding
  - **Use this when script checks**: `cm.getEventManager("Trains")`, `cm.getEventManager("Boats")`, or similar event manager availability checks
  - Example: `{ "type": "transportAvailable", "operator": "=", "value": "0", "referenceId": "200000122" }` checks if train to Ludibrium is unavailable

### 6. Context Propagation Rules

When a player selects from a list (destinations, items, etc.):
- Store dynamic values in `context` object on the choice
- Reference later as `context.destination`, `context.cost`, `context.itemId`, etc.
- Example:
  ```json
  {
    "text": "Ellinia (800 mesos)",
    "nextState": "confirmWarp",
    "context": {
      "destination": "101000000",
      "cost": "800"
    }
  }
  ```

### 6.1. Arithmetic Expressions

Both operation parameters and condition values support arithmetic expressions for dynamic calculations:

**Supported Operators**: `*`, `/`, `+`, `-`

**Common Use Case - Bulk Crafting**:
When converting crafting NPCs that ask for quantity (using `askNumber`), use arithmetic expressions to scale material requirements:

```json
{
  "id": "askQuantity",
  "type": "askNumber",
  "askNumber": {
    "text": "How many would you like to craft?",
    "default": 1,
    "min": 1,
    "max": 100,
    "contextKey": "quantity",
    "nextState": "validateMaterials"
  }
},
{
  "id": "validateMaterials",
  "type": "genericAction",
  "genericAction": {
    "operations": [],
    "outcomes": [
      {
        "conditions": [
          {
            "type": "item",
            "operator": ">=",
            "value": "10 * {context.quantity}",  // If quantity=5, checks for 50 items
            "referenceId": 4000003
          }
        ],
        "nextState": "performCraft"
      }
    ]
  }
},
{
  "id": "performCraft",
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "destroy_item",
        "params": {
          "itemId": "4000003",
          "quantity": "10 * {context.quantity}"  // Destroys scaled amount
        }
      },
      {
        "type": "award_item",
        "params": {
          "itemId": "4003001",
          "quantity": "{context.quantity}"  // Awards requested quantity
        }
      }
    ],
    "outcomes": [{"conditions": [], "nextState": "craftSuccess"}]
  }
}
```

**How It Works**:
1. Context substitution happens first: `{context.quantity}` → `"5"`
2. Expression evaluation happens second: `"10 * 5"` → `50`
3. Result is used in the operation/condition

**Evaluation Order**: Left-to-right without operator precedence.

### 7. Required Validations

- ✅ All dialogue states have correct choice counts (sendOk: 2, sendYesNo: 3, sendNext: 2, sendNextPrev: 3)
- ✅ **Choice text uses exact required values** (case-sensitive):
  - sendOk: "Ok" and "Exit"
  - sendYesNo: "Yes", "No", and "Exit"
  - sendAcceptDecline: "Accept", "Decline", and "Exit"
  - sendNext: "Next" and "Exit"
  - sendNextPrev: "Previous", "Next", and "Exit"
  - listSelection: includes "Exit"
- ✅ `nextState` is `null` (not string) when ending conversation
- ✅ Map IDs are mapped to human-readable names from Map.txt
- ✅ NPC IDs are mapped to names from NPC.txt (when referenced)
- ✅ Empty conditions arrays are `[]`, not omitted
- ✅ Operation params use correct names (e.g., `portalId` not `portal`)
- ✅ Output conforms to schema (validate before finalizing)

### 8. Coding Style

- Prefer **clarity** over cleverness
- **No assumptions** - use explicit structure
- Use descriptive state IDs (e.g., `askDestination`, `checkMeso`, `performWarp`)
- Use `null` for terminal `nextState`, not `"end"` or other strings

## Task

The script to convert: **$ARGUMENTS**

**Steps:**
1. Read schema and conversion spec files (DO NOT read Map.txt or NPC.txt in full)
2. **Read validation model AND operation types**:
   - Read `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` to get supported condition types (check ConditionType constants) and operators
   - Read `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` to get supported operation types (search for `case "operation_name":` statements in the switch)
3. **Read quest model** (if needed): If script has quest checks, read `services/atlas-query-aggregator/atlas.com/query-aggregator/quest/model.go` for QuestStatus enum values
4. If a file path is provided, read the script file; otherwise use the provided code
5. Analyze the script thoroughly:
   - Count all `cm.send*()` calls - each one becomes a dialogue state
   - Identify where `cm.dispose()` is called - those paths end (nextState: null)
   - Note map IDs and NPC IDs referenced
   - Trace the exact flow and branching logic
6. Use Grep to look up specific map IDs in `docs/Map.txt` (pattern: `^<mapId> - `)
7. Use Grep to look up specific NPC IDs in `docs/NPC.txt` (pattern: `^<npcId> - `)
8. Convert to JSON following all requirements above
9. **VALIDATE - Script Accuracy**: Verify each dialogue state has a corresponding `cm.send*()` in the original - no extra states!
10. **VALIDATE - Text Accuracy**: Confirm all dialogue text is copied verbatim from the original script
11. **VALIDATE - Implementation**: Check that all conditions and operations used are actually implemented:
    - Verify all condition types exist in validation/model.go (check the ConditionType constants you read in step 2)
    - Verify all operators are supported (=, >, <, >=, <=)
    - For `questStatus` conditions, verify the value matches the QuestStatus enum (e.g., 3 for COMPLETED)
    - **Verify all operation types exist in operation_executor.go** (check against the switch/case statements you read in step 2)
    - **If ANY validation fails**: STOP immediately, report to user with examples of what's supported, and present options:
      - Option 1: Modify the conversion to use only supported types
      - Option 2: Implement the missing feature first (requires user approval)
      - Option 3: Skip this NPC conversion (mark as TODO)
12. Determine appropriate output filename based on NPC ID (e.g., `npc_2003.json`)
13. Validate against the schema
14. **ONLY if all validations pass**: Write the output file to `services/atlas-npc-conversations/conversations/npc/` directory
15. **VALIDATE - Build Check**: Verify the service still compiles:
    - Run `go build` in `services/atlas-npc-conversations/atlas.com/npc` directory
    - If build fails, report the error and ask user how to proceed
    - If conversion uses newly implemented operations/conditions, inform user that related services may need changes
16. Report completion with summary of states created and build status

**Example Grep Usage:**
- To find map ID 100000000: `Grep` with pattern `^100000000 - ` in `docs/Map.txt`
- To find NPC ID 1012100: `Grep` with pattern `^1012100 - ` in `docs/NPC.txt`

**Example 1 - Choice text requirements:**
```javascript
// Original script with sendYesNo
cm.sendYesNo("Do you want to continue?");
```
❌ WRONG - Incorrect choice text:
```json
{
  "type": "dialogue",
  "dialogue": {
    "dialogueType": "sendYesNo",
    "text": "Do you want to continue?",
    "choices": [
      {"text": "Yes", "nextState": "continue"},
      {"text": "No", "nextState": "decline"},
      {"text": "Cancel", "nextState": null}  // ← WRONG! Must be "Exit"
    ]
  }
}
```

✅ CORRECT - Exact required text:
```json
{
  "type": "dialogue",
  "dialogue": {
    "dialogueType": "sendYesNo",
    "text": "Do you want to continue?",
    "choices": [
      {"text": "Yes", "nextState": "continue"},
      {"text": "No", "nextState": "decline"},
      {"text": "Exit", "nextState": null}  // ← CORRECT!
    ]
  }
}
```

**Example 2 - Converting cm.sendSimple() with list tags:**
```javascript
// Original script uses cm.sendSimple() with #L tags for menu options
cm.sendSimple("Choose a destination:\r\n#L0#Henesys#l\r\n#L1#Ellinia#l\r\n#L2#Perion#l");
```
❌ WRONG - Do NOT use dialogue type (sendSimple is not a valid dialogueType):
```json
{
  "type": "dialogue",
  "dialogue": {
    "dialogueType": "sendSimple",  // ← sendSimple is NOT a valid dialogueType!
    "text": "Choose a destination:\r\n#L0#Henesys#l\r\n#L1#Ellinia#l\r\n#L2#Perion#l",
    "choices": [...]
  }
}
```

✅ CORRECT - Use listSelection state type:
```json
{
  "type": "listSelection",
  "listSelection": {
    "title": "Choose a destination:",
    "choices": [
      {"text": "Henesys", "nextState": "..."},
      {"text": "Ellinia", "nextState": "..."},
      {"text": "Perion", "nextState": "..."},
      {"text": "Exit", "nextState": null}  // ← Required "Exit" choice
    ]
  }
}
```

**Example 3 - Don't add extra dialogues:**
```javascript
// Original script
if (status == 1) {
    cm.warp(104000000, 0);
    cm.dispose();
}
```
❌ WRONG - Adding extra dialogue:
```json
{
  "id": "warp",
  "operations": [{"type": "warp_to_map", "params": {"mapId": "104000000"}}],
  "outcomes": [{"conditions": [], "nextState": "confirmWarp"}]
},
{
  "id": "confirmWarp",
  "dialogue": {"text": "Welcome to Lith Harbor!", ...}  // ← This doesn't exist in original!
}
```

✅ CORRECT - End immediately after warp:
```json
{
  "id": "warp",
  "operations": [{"type": "warp_to_map", "params": {"mapId": "104000000"}}],
  "outcomes": [{"conditions": [], "nextState": null}]  // ← Ends here, matching cm.dispose()
}
```

**Example 4 - Validation failure (unsupported condition):**
```javascript
// Original script uses quest completion check
if (cm.isQuestCompleted(2013))
```

❌ WRONG - Using unsupported condition type:
```json
{
  "conditions": [
    {
      "type": "quest",           // ← Not in validation/model.go!
      "operator": "completed",   // ← Not a supported operator!
      "value": "2013"
    }
  ]
}
```

✅ CORRECT - Validation catches the error:
```
❌ VALIDATION FAILED - Unsupported conditions found:

Condition type "quest" not found in validation/model.go
Supported types: jobId, meso, mapId, fame, item, questStatus, level, ...

Did you mean "questStatus"?

For quest completion check, use:
{
  "type": "questStatus",
  "operator": "=",
  "value": "3",        // COMPLETED from quest/model.go
  "referenceId": "2013"
}

How would you like to proceed?
```

**Example 5 - Validation failure (unsupported operation):**
```javascript
// Original script gains pet closeness
cm.gainCloseness(2, 0);
```

❌ WRONG - Using unsupported operation type without checking:
```json
{
  "operations": [
    {
      "type": "gain_closeness",  // ← Not verified in operation_executor.go!
      "params": {
        "petIndex": "0",
        "amount": "2"
      }
    }
  ]
}
```

✅ CORRECT - Validation catches the error and asks user:
```
❌ VALIDATION FAILED - Unsupported operation found:

Operation type "gain_closeness" not found in operation_executor.go

Supported operation types found in the switch statement:
- warp_to_map
- award_item
- award_mesos
- award_exp
- destroy_item
- change_job
- create_skill
- update_skill
- increase_buddy_capacity

The script requires "gain_closeness" operation to increase pet intimacy.

How would you like to proceed?
1. Skip this NPC conversion (requires gain_closeness operation)
2. Implement the "gain_closeness" operation first (requires changes to npc-conversations, saga-orchestrator, and query-aggregator)
3. Convert but mark gain_closeness as TODO

Please choose an option or provide guidance.
```

Begin conversion now.
