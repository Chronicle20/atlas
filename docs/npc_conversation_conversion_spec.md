# NPC Conversation Conversion Specification

This specification outlines how to convert MapleStory NPC JavaScript scripts into a **state-machine driven JSON** format based on the provided `npc_conversation_schema.json`. The result must strictly conform to that schema and support integration with your existing saga-orchestration and query services.

---

## 🔁 Overview of Structure

Each converted conversation must include:

- `npcId`: The numeric ID of the NPC.
- `startState`: The ID of the first state.
- `states`: A flat array of distinct, named states.
- Optional: `optionSets` for grouped crafting options.

---

## 📦 State Types (from schema)

| State Type       | Description                                                                 |
|------------------|-----------------------------------------------------------------------------|
| `dialogue`       | Presents a message to the user. Supports `sendOk`, `sendYesNo`, `sendNext`, `sendNextPrev`, `sendPrev`, and `sendSimple`. |
| `genericAction`  | Executes logic or validation (e.g. meso check, job check, warp).            |
| `craftAction`    | Defines crafting logic with required items and meso cost.                   |
| `listSelection`  | Allows the user to choose from a dynamic list. Sets context values.         |

---

## 🧠 Dialogue State Format

```json
{
  "id": "exampleDialogue",
  "type": "dialogue",
  "dialogue": {
    "dialogueType": "sendYesNo",
    "text": "Do you want to go to Ellinia?",
    "choices": [
      { "text": "Yes", "nextState": "checkMeso" },
      { "text": "No", "nextState": "decline" },
      { "text": "Exit", "nextState": null }
    ]
  }
}
```

- All `dialogue` states must include an explicit **exit**.
- Use appropriate `dialogueType` values: `sendOk`, `sendNext`, `sendNextPrev`, `sendPrev`, `sendYesNo`, `sendSimple`.

---

## ⚙️ `genericAction` States

Used for validations, logic branching, item or meso manipulation, and warp actions.

```json
{
  "id": "checkMeso",
  "type": "genericAction",
  "genericAction": {
    "operations": [],
    "outcomes": [
      {
        "conditions": [{ "type": "meso", "operator": "<", "value": "context.cost" }],
        "nextState": "insufficientMeso"
      },
      {
        "conditions": [],
        "nextState": "warp"
      }
    ]
  }
}
```

- All outcomes must have `conditions`, even if empty.
- `operations` include types like `award_mesos`, `removeItem`, `warp_to_map`.

---

## 🧾 `listSelection` States

Used for dynamic user options (e.g. selecting a travel destination or crafting item).

```json
{
  "id": "selectDestination",
  "type": "listSelection",
  "listSelection": {
    "title": "Choose your destination.",
    "choices": [
      {
        "text": "Ellinia",
        "nextState": "confirmTrip",
        "context": {
          "destination": "101000000",
          "cost": "-800"
        }
      },
      {
        "text": "Exit",
        "nextState": null
      }
    ]
  }
}
```

- Each `choice` can populate context for use in later states.
- Values like `context.destination` and `context.cost` are reused downstream.

---

## 🧪 `craftAction` States

Used for defining item crafting.

```json
{
  "id": "craftItem",
  "type": "craftAction",
  "craftAction": {
    "itemId": 1092022,
    "materials": [4000021, 4003000],
    "quantities": [30, 5],
    "mesoCost": 10000,
    "successState": "successMsg",
    "failureState": "failMsg",
    "missingMaterialsState": "missingMsg"
  }
}
```

- Use only when crafting logic is explicitly required.
- Prefer `optionSets` for dynamic crafting menus.

---

## 💬 Action Outcome Handling

Always represent outcomes clearly:

```json
"outcomes": [
  {
    "conditions": [{ "type": "jobId", "operator": "=", "value": "0" }],
    "nextState": "beginnerOptions"
  },
  {
    "conditions": [],
    "nextState": "regularOptions"
  }
]
```

- First matching condition applies.
- Use `context` values for dynamic transitions.

---

## 🌍 Destination Names

- Always replace raw map IDs with correct names using `Map.txt`.
- Example: `100000000 → Victoria Road - Henesys`

---

## 🧩 Context Propagation

- When values (like cost, destination, item ID) are selected from a menu, use `context` to store them.
- Access context later via: `context.cost`, `context.destination`, etc.

---

## 📁 File Output

- The result must be saved as `output.json`
- The result **must** conform to `npc_conversation_schema.json`.

---

## ✅ Validation Against Implementation

**CRITICAL:** Before finalizing the conversion, validate that all conditions and operations are actually implemented in the query-aggregator service.

### Supported Condition Types

Read `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` to get the authoritative list of supported condition types.

Currently supported (verify by reading the file):
- `jobId` - Character's job ID
- `meso` - Character's meso amount
- `mapId` - Character's current map
- `fame` - Character's fame level
- `item` - Item possession check (requires `referenceId`)
- `questStatus` - Quest status check (requires `referenceId`, value is quest.QuestStatus enum)
- `level` - Character level
- And others (check model.go for complete list)

### Supported Operators

From `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go`:
- `=` - Equals
- `>` - Greater than
- `<` - Less than
- `>=` - Greater than or equal
- `<=` - Less than or equal

**DO NOT** use custom operators like "completed", "started", etc.

### Quest Status Values

If using `questStatus` conditions, read `services/atlas-query-aggregator/atlas.com/query-aggregator/quest/model.go` for the QuestStatus enum:

```go
const (
    UNDEFINED = 0
    NOT_STARTED = 1
    STARTED = 2
    COMPLETED = 3
)
```

For checking quest completion: `{"type": "questStatus", "operator": "=", "value": "3", "referenceId": "questId"}`

### Supported Operation Types

Check the saga-orchestrator service for supported operation types. Common operations:
- `warp_to_map` - Teleport character
- `award_item` - Give item to character
- `award_mesos` - Give mesos to character
- `award_exp` - Give experience
- `destroy_item` - Remove item from character
- `change_job` - Change character's job

### Validation Steps

Before writing the output file:

1. **Read validation model**: Read `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` to get the current list of supported condition types
2. **Check all condition types**: Verify every condition type used in the conversion is in the supported list
3. **Check all operators**: Verify all operators are in the supported list (=, >, <, >=, <=)
4. **Validate enum values**: For conditions using enums (like questStatus), read the enum definition and use correct values
5. **Check operations**: Verify all operation types are implemented
6. **If validation fails**: Report which conditions/operations are not supported and ask the user how to proceed before writing the file

### Example Validation Failure Response

If you encounter unsupported conditions or operations:

```
❌ VALIDATION FAILED - Unsupported conditions/operations found:

Conditions not implemented:
- "quest" - Not found in validation/model.go
  → Did you mean "questStatus"?

Operations not implemented:
- "teleport_player" - Not found in supported operations
  → Did you mean "warp_to_map"?

How would you like to proceed?
1. Skip this NPC and document the missing features
2. Use alternative conditions/operations (provide suggestions)
3. Implement the missing features first
```