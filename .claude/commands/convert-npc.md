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
- **Dialogue types**: `sendOk`, `sendNext`, `sendYesNo`, `sendSimple`
  - **IMPORTANT**: `sendSimple` with `#L0#`, `#L1#`, etc. tags → use `listSelection` state type
  - Pattern: `#L<number>#<choice text>#l` indicates list items
  - Extract title (text before first `#L`) and parse each list item
- **Conditional logic**: Job checks, meso/item requirements
- **Actions**: `gainItem`, `warp`, `gainMeso`, etc.
- **Branching**: Choice destinations and flow control
- **When conversations end**: Look for `cm.dispose()` calls - these indicate the conversation ends

### 2. Produce Valid JSON Output

- Extract the NPC ID from the script filename or comments
- Define a clear `startState`
- Create a `states` array with proper state objects
- Use context propagation for dynamic values (destination, cost, itemId, etc.)

### 3. State Types (from schema)

- **dialogue**: Present messages with choices (`sendOk`, `sendNext`, `sendNextPrev`, `sendYesNo`, `sendSimple`)
  - **Choice count requirements**:
    - `sendOk`: exactly 2 choices
    - `sendYesNo`: exactly 3 choices
    - `sendSimple`: at least 1 choice
    - `sendNext`: exactly 2 choices
    - `sendNextPrev`: exactly 3 choices
  - **CRITICAL - Required choice text values** (case-sensitive, must match exactly):
    - `sendOk`: `"Ok"` and `"Exit"` (note: lowercase 'k')
    - `sendYesNo`: `"Yes"`, `"No"`, and `"Exit"`
    - `sendNext`: `"Next"` and `"Exit"`
    - `sendNextPrev`: `"Previous"`, `"Next"`, and `"Exit"`
    - `sendSimple`: Must include `"Exit"` choice
    - `listSelection`: Must include `"Exit"` choice
  - **When to use sendNext vs sendNextPrev**:
    - Use `sendNext` for first page of multi-page dialogue or single-page dialogue
    - Use `sendNextPrev` for subsequent pages that allow backward navigation
  - Terminal choices (nextState: null) should use text `"Exit"`
- **genericAction**: Execute logic/validation (meso checks, warps, item operations)
  - All outcomes MUST have `conditions` array (can be empty)
- **craftAction**: Define crafting with materials, quantities, meso cost
- **listSelection**: Dynamic choice lists that populate context
  - MUST include Exit choice

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
- **destroy_item**: Remove items
  - Params: `itemId` (string), `quantity` (string)
- **change_job**: Change character job
  - Params: `jobId` (string)

### 5. Condition Types

Common conditions in `outcomes`:
- **jobId**: Check character's job (operators: =, >, <, >=, <=)
- **meso**: Check meso amount (operators: =, >, <, >=, <=)
- **mapId**: Check current map (operators: =)
- **fame**: Check fame level (operators: =, >, <, >=, <=)
- **item**: Check item possession (operators: =, >, <, >=, <=, requires `itemId` field)

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

### 7. Required Validations

- ✅ All dialogue states have correct choice counts (sendOk: 2, sendYesNo: 3, sendNext: 2, sendNextPrev: 3)
- ✅ **Choice text uses exact required values** (case-sensitive):
  - sendOk: "Ok" and "Exit"
  - sendYesNo: "Yes", "No", and "Exit"
  - sendNext: "Next" and "Exit"
  - sendNextPrev: "Previous", "Next", and "Exit"
  - listSelection/sendSimple: includes "Exit"
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
2. If a file path is provided, read the script file; otherwise use the provided code
3. Analyze the script thoroughly:
   - Count all `cm.send*()` calls - each one becomes a dialogue state
   - Identify where `cm.dispose()` is called - those paths end (nextState: null)
   - Note map IDs and NPC IDs referenced
   - Trace the exact flow and branching logic
4. Use Grep to look up specific map IDs in `docs/Map.txt` (pattern: `^<mapId> - `)
5. Use Grep to look up specific NPC IDs in `docs/NPC.txt` (pattern: `^<npcId> - `)
6. Convert to JSON following all requirements above
7. **VALIDATE**: Verify each dialogue state has a corresponding `cm.send*()` in the original - no extra states!
8. **VALIDATE**: Confirm all dialogue text is copied verbatim from the original script
9. Determine appropriate output filename based on NPC ID (e.g., `npc_2003.json`)
10. Validate against the schema
11. Write the output file to `services/atlas-npc-conversations/conversations/` directory
12. Report completion with summary of states created

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

**Example 2 - sendSimple with list tags:**
```javascript
// Original script
cm.sendSimple("Choose a destination:\r\n#L0#Henesys#l\r\n#L1#Ellinia#l\r\n#L2#Perion#l");
```
❌ WRONG - Using dialogue type:
```json
{
  "type": "dialogue",
  "dialogue": {
    "dialogueType": "sendSimple",
    "text": "Choose a destination:\r\n#L0#Henesys#l\r\n#L1#Ellinia#l\r\n#L2#Perion#l",
    "choices": [...]  // ← Listing out choices manually
  }
}
```

✅ CORRECT - Using listSelection type:
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

Begin conversion now.
