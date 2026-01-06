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
| `dialogue`       | Presents a message to the user. Supports `sendOk`, `sendYesNo`, `sendNext`, `sendNextPrev`, `sendPrev`, and `sendAcceptDecline`. |
| `genericAction`  | Executes logic or validation (e.g. meso check, job check, warp).            |
| `craftAction`    | Defines crafting logic with required items and meso cost.                   |
| `listSelection`  | Allows the user to choose from a dynamic list. Sets context values. **Use for menu-style selections (formerly `sendSimple` with `#L` tags).** |

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
- Use appropriate `dialogueType` values: `sendOk`, `sendNext`, `sendNextPrev`, `sendPrev`, `sendYesNo`, `sendAcceptDecline`.
- **Note**: For menu-style selections (scripts using `sendSimple` with `#L` tags), use the `listSelection` state type instead of `dialogue`.

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
        "conditions": [{ "type": "meso", "operator": "<", "value": "{context.cost}" }],
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
- **Always use curly brace format** `{context.xxx}` to reference context values for consistency.
- This format works in all locations:
  - Dialogue text: `"Do you want #b#m{context.destination}##k for #b{context.cost} mesos#k?"`
  - Condition values: `"value": "{context.cost}"`
  - Operation params: `"amount": "-{context.cost}"`, `"mapId": "{context.destination}"`
- Legacy format `context.xxx` (without braces) is still supported but deprecated.

---

## 📁 File Output

- The result must be saved as `output.json`
- The result **must** conform to `npc_conversation_schema.json`.

---

## ✅ MANDATORY Validation Workflow

**🚨 CRITICAL - DO NOT SKIP THESE STEPS 🚨**

You **MUST** complete validation **BEFORE** writing any output files. Never assume condition types or operation types exist - always verify against the implementation.

---

### Step 1: Load Implementation Reference (DO THIS FIRST)

Before analyzing the script, read these files to build your validation reference:

1. **Read condition types**: `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go`
   - Extract all `ConditionType` constants (lines 15-40)
   - Note which conditions require `referenceId` (check validation logic)

2. **Read quest enums** (if script uses quests): `services/atlas-query-aggregator/atlas.com/query-aggregator/quest/model.go`
   - Extract `QuestStatus` enum values (NOT_STARTED=1, STARTED=2, COMPLETED=3, etc.)

3. **Read operation types** (if needed): Check saga-orchestrator for supported operations

**Build a mental/written list of valid types before proceeding.**

---

### Step 2: Analyze Script and Plan Conversion

After loading the validation reference:

1. Read and analyze the JavaScript NPC script
2. Identify all conditions needed (level checks, quest checks, item checks, etc.)
3. Identify all operations needed (warps, item awards, etc.)
4. **Map each script requirement to a valid condition/operation type**

---

### Step 3: Pre-Conversion Validation Checklist

Before writing the JSON output, **STOP** and complete this checklist:

#### ✅ Condition Type Validation

For each condition you plan to use, verify:

- [ ] Condition type exists in `validation/model.go` ConditionType constants
- [ ] Using exact spelling (case-sensitive): `mapCapacity` not `mapPlayerCount`
- [ ] Operator is valid (=, >, <, >=, <=) - NO custom operators
- [ ] If requires `referenceId`, you're providing it (item, questStatus, mapCapacity, etc.)
- [ ] Enum values are correct (questStatus uses 1/2/3, not "started"/"completed")

#### ✅ Operation Type Validation

For each operation you plan to use, verify:

- [ ] Operation type is implemented in saga-orchestrator
- [ ] Using correct parameter names (`mapId`, `portalId`, `itemId`, `quantity`, etc.)
- [ ] All required parameters are provided

#### ✅ Script Accuracy Validation

- [ ] Every dialogue state corresponds to a `cm.send*()` call in the original script
- [ ] No extra dialogue states added that don't exist in original
- [ ] All dialogue text is copied verbatim (no paraphrasing)
- [ ] Conversation ends (nextState: null) where `cm.dispose()` is called

---

### Step 4: STOP and Report (if uncertain or issues found)

**IF** you encounter ANY of the following, **STOP immediately and ask the user**:

- ❓ Condition type you need might not exist in validation/model.go
- ❓ Unsure which condition type to use for a script requirement
- ❓ Operation type might not be implemented
- ❓ Uncertain about enum values or parameter names
- ❌ Validation checklist has unchecked items

**Present a validation report:**

```markdown
## 🔍 Pre-Conversion Validation Report

### ✅ Valid Conditions (verified in model.go)
- `level` - Character level check
- `questStatus` - Quest status check (requires referenceId)

### ❓ Uncertain/Missing Conditions
- Need to check player count in map
  - Found: `mapCapacity` in model.go (line 39)
  - Requires: referenceId with map ID
  - **Confirming this is correct?**

### ⚠️ Issues Found
- Script uses `cm.getPlayerCount()` but I'm uncertain about implementation
- **Should I proceed with `mapCapacity` condition?**

### 📋 Conversion Plan
- State 1: Level check using `level` condition
- State 2: Quest check using `questStatus` condition
- State 3: Map capacity check using `mapCapacity` condition

**Ready to proceed? Or would you like me to adjust the approach?**
```

**Wait for user approval before proceeding.**

---

### Step 5: Write Output (only after validation passes)

Only write the JSON file after:
- ✅ All checklist items verified
- ✅ User approved (if you reported uncertainties)
- ✅ No assumption of condition/operation types

---

### Quick Reference: Supported Types

#### Condition Types (from validation/model.go)

Always verify by reading the file, but common types include:
- `jobId` - Character's job ID
- `meso` - Character's meso amount
- `mapId` - Character's current map
- `fame` - Character's fame level
- `gender` - Character gender (0 = male, 1 = female)
- `level` - Character level
- `reborns` - Rebirth count
- `dojoPoints` - Mu Lung Dojo points
- `vanquisherKills` - Vanquisher kill count
- `gmLevel` - GM level
- `guildId` - Guild ID (0 = not in guild)
- `guildLeader` - Guild leader status (0 = not leader, 1 = is leader)
- `guildRank` - Guild rank
- `questStatus` - Quest status (requires `referenceId`, value is 1/2/3)
- `questProgress` - Quest progress step (requires `referenceId` and `step`)
- `hasUnclaimedMarriageGifts` - Unclaimed marriage gifts (0 = false, 1 = true)
- `strength`, `dexterity`, `intelligence`, `luck` - Character stats
- `buddyCapacity` - Buddy list capacity
- `petCount` - Number of pets
- `mapCapacity` - Player count in map (requires `referenceId` with map ID)
- `item` - Item possession (requires `referenceId`)
- And others (check model.go for complete list)

#### Operators (from validation/model.go)

- `=` - Equals
- `>` - Greater than
- `<` - Less than
- `>=` - Greater than or equal
- `<=` - Less than or equal

**DO NOT** use custom operators like "completed", "started", "active", etc.

#### Quest Status Enum Values (from quest/model.go)

```go
const (
    UNDEFINED = 0
    NOT_STARTED = 1
    STARTED = 2
    COMPLETED = 3
)
```

Use numeric values: `{"type": "questStatus", "operator": "=", "value": "2", "referenceId": "22515"}`

#### Common Operation Types

Verify in saga-orchestrator, but common operations:
- `warp_to_map` - Teleport character (params: `mapId`, `portalId`)
- `warp_to_random_portal` - Warp to random portal (params: `mapId`)
- `award_item` - Give item (params: `itemId`, `quantity`)
- `award_mesos` - Give mesos (params: `amount`, `actorId`, `actorType`)
- `award_exp` - Give experience (params: `amount`, `type`, `attr1`)
- `award_level` - Give levels (params: `amount`)
- `destroy_item` - Remove item (params: `itemId`, `quantity`)
- `change_job` - Change job (params: `jobId`)
- `change_hair` - Change hair style (params: `styleId`)
- `change_face` - Change face style (params: `styleId`)
- `change_skin` - Change skin color (params: `styleId`)
- `increase_buddy_capacity` - Increase buddy capacity (params: `amount`)
- `gain_closeness` - Increase pet closeness (params: `petId` or `petIndex`, `amount`)
- `create_skill` - Create skill (params: `skillId`, `level`, `masterLevel`)
- `update_skill` - Update skill (params: `skillId`, `level`, `masterLevel`)
- `spawn_monster` - Spawn monsters at location (params: `monsterId`, `x`, `y`, `count`, `team`) - foothold resolved automatically
- `complete_quest` - Complete a quest (params: `questId`, `npcId`) - stub implementation
- `start_quest` - Start a quest (params: `questId`, `npcId`) - stub implementation
- `apply_consumable_effect` - Apply consumable effects without consuming (params: `itemId`) - for NPC buffs (maps to `cm.useItem()`)

**Local Operations:**
- `local:generate_hair_styles` - Generate hair styles (params: `baseStyles`, `genderFilter`, etc.)
- `local:generate_hair_colors` - Generate hair colors (params: `colors`, etc.)
- `local:generate_face_styles` - Generate face styles (params: `baseStyles`, etc.)
- `local:select_random_cosmetic` - Random selection (params: `stylesContextKey`, `outputContextKey`)
- `local:select_random_weighted` - Weighted random selection (params: `items`, `weights`, `outputContextKey`)
- `local:fetch_map_player_counts` - Fetch player counts (params: `mapIds`)
- `local:log` - Log message (params: `message`)
- `local:debug` - Debug log (params: `message`)

#### Detailed: local:select_random_weighted

Performs weighted random selection from a list of items. Useful for reward systems, loot drops, or any scenario requiring probability-based item selection.

**Parameters:**
- `items` (string, required) - Comma-separated list of values to select from
- `weights` (string, required) - Comma-separated list of integer weights (must match length of items)
- `outputContextKey` (string, required) - Context key to store the selected value

**Behavior:**
- Items and weights must have the same length
- Weights must be non-negative integers
- Higher weight = higher probability of selection
- Total weight = sum of all weights
- Each item has probability = (weight / total weight)
- Weight of 0 means item will never be selected
- Selected value is stored as a string in the context

**Example Usage:**

```json
{
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "local:select_random_weighted",
        "params": {
          "items": "1040052,1040054,1040130",
          "weights": "10,20,15",
          "outputContextKey": "selectedItem"
        }
      }
    ],
    "outcomes": [
      {
        "conditions": [],
        "nextState": "awardItem"
      }
    ]
  }
}
```

In this example:
- Item 1040052 has 10/45 (~22%) chance
- Item 1040054 has 20/45 (~44%) chance
- Item 1040130 has 15/45 (~33%) chance
- Selected item ID is stored in `context.selectedItem`

**Using Selected Value:**

```json
{
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "award_item",
        "params": {
          "itemId": "{context.selectedItem}",
          "quantity": "1"
        }
      }
    ]
  }
}
```

**Real-World Example (Gender-Based Rewards):**

```json
{
  "id": "selectMaleReward",
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "local:select_random_weighted",
        "params": {
          "items": "1040052,1040130,1042013",
          "weights": "10,15,5",
          "outputContextKey": "reward"
        }
      }
    ],
    "outcomes": [
      {
        "conditions": [],
        "nextState": "giveReward"
      }
    ]
  }
}
```

---

### Example: Good Validation Flow

```
1. Read validation/model.go → Found mapCapacity, questStatus, level
2. Analyze script → Needs level check, quest check, map player count check
3. Map requirements:
   - cm.getLevel() >= 20 → `level` condition ✅
   - cm.isQuestActive(22515) → `questStatus` with value "2" ✅
   - cm.getPlayerCount(mapId) >= 5 → `mapCapacity` condition ✅
4. Checklist complete → All verified
5. Write output file
```

### Example: Bad Validation Flow (DON'T DO THIS)

```
1. Analyze script
2. Assume mapPlayerCount exists ❌
3. Write output file ❌
4. User finds error ❌
```

---

## 🎯 Summary: The Golden Rule

**"Read the implementation first, write the conversion second, ask when uncertain."**

Never write output files based on assumptions. Always validate first.