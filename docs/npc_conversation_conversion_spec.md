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

| State Type         | Description                                                                 |
|--------------------|-----------------------------------------------------------------------------|
| `dialogue`         | Presents a message to the user. Supports `sendOk`, `sendYesNo`, `sendNext`, `sendNextPrev`, `sendPrev`, and `sendAcceptDecline`. |
| `genericAction`    | Executes logic or validation (e.g. meso check, job check, warp).            |
| `craftAction`      | Defines crafting logic with required items and meso cost.                   |
| `transportAction`  | Initiates an instance-based transport via saga-orchestrator. Used when the original script uses `cm.getEventManager(...)` + `em.startInstance(...)`. |
| `listSelection`    | Allows the user to choose from a dynamic list. Sets context values. **Use for menu-style selections (formerly `sendSimple` with `#L` tags).** |

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

### Speaker Override

By default, all dialogue is spoken by the NPC (`NPC_LEFT`). You can optionally override the speaker using the `speaker` field. This is useful when the player character speaks dialogue (common in quest conversations).

**Available speaker values:**
| Speaker | Description |
|---------|-------------|
| `NPC_LEFT` | NPC portrait on the left (default) |
| `NPC_RIGHT` | NPC portrait on the right |
| `CHARACTER_LEFT` | Player character portrait on the left |
| `CHARACTER_RIGHT` | Player character portrait on the right |
| `UNKNOWN` | Unknown speaker type |
| `UNKNOWN2` | Unknown speaker type 2 |

**Example with speaker override:**
```json
{
  "id": "playerResponse",
  "type": "dialogue",
  "dialogue": {
    "dialogueType": "sendNextPrev",
    "text": "Don't be afraid, #b#p1300005##k sent me here.",
    "speaker": "CHARACTER_LEFT",
    "choices": [
      { "text": "Previous", "nextState": "npcDialogue" },
      { "text": "Next", "nextState": "thankYou" },
      { "text": "Exit", "nextState": null }
    ]
  }
}
```

**When to use speaker override:**
- When converting scripts that use `cm.sendNext("...", 2)` or similar with a speaker parameter
- In quest conversations where the player responds to an NPC
- When the original script explicitly sets a non-default speaker type

**Mapping from JavaScript speaker types:**
| JS Type | JSON Speaker |
|---------|--------------|
| 0 | `NPC_LEFT` |
| 1 | `NPC_RIGHT` |
| 2 | `CHARACTER_LEFT` |
| 3 | `CHARACTER_RIGHT` |
| 4 | `UNKNOWN` |
| 5 | `UNKNOWN2` |

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

## 🚂 `transportAction` States

Used for instance-based transports (trains, boats, genies, etc.) that go through saga-orchestrator. Converts the JavaScript pattern of `cm.getEventManager("...")` + `em.startInstance(cm.getPlayer())`.

```json
{
  "id": "startTransport",
  "type": "transportAction",
  "transportAction": {
    "routeName": "kerning-train-to-square",
    "failureState": "transportFailed",
    "capacityFullState": "transportCapacityFull",
    "alreadyInTransitState": "transportAlreadyInTransit",
    "routeNotFoundState": "transportFailed",
    "serviceErrorState": "transportFailed"
  }
}
```

- `routeName` (required): Instance route name, resolved to UUID at runtime by saga-orchestrator
- `failureState` (required): General fallback state for unhandled errors
- `capacityFullState` (optional): State when the transport vehicle is full
- `alreadyInTransitState` (optional): State when the character is already on another transport
- `routeNotFoundState` (optional): State when the route name doesn't exist in config
- `serviceErrorState` (optional): State when the transport service is unavailable

On success, the transport system warps the character to the transit map automatically and the conversation ends. Failure states should be `dialogue` states with appropriate error messages from the original script.

**When to use `transportAction` vs `warp_to_map`:**
- `transportAction`: Script uses `cm.getEventManager(...)` + `em.startInstance(...)` — character boards a vehicle, travels through a transit map with a timer, then arrives
- `warp_to_map`: Script uses `cm.warp(mapId, portal)` — instant teleportation

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
- `excessSp` - Excess skill points beyond expected for job tier (requires `referenceId` with base level: 30 for 2nd job, 70 for 3rd job, 120 for 4th job). Returns `remainingSp - (level - baseLevel) * 3`. Use with `> 0` to check if player has too much unspent SP.
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
- `warp_to_saved_location` - Warp to a saved location and delete it (params: `locationType`) - maps to `cm.getSavedLocation()` + `cm.warp()`
- `save_location` - Save character's current location for later return (params: `locationType`, `mapId` optional, `portalId` optional)
- `award_item` - Give item (params: `itemId`, `quantity`, `expiration` - optional, milliseconds from now)
- `award_mesos` - Give mesos (params: `amount`, `actorId`, `actorType`)
- `award_exp` - Give experience (params: `amount`, `type`, `attr1`)
- `award_level` - Give levels (params: `amount`)
- `destroy_item` - Remove item by template ID (params: `itemId`, `quantity`)
- `destroy_item_from_slot` - Remove item from specific inventory slot (params: `inventoryType`, `slot`, `quantity`) - for equipped items use negative slot values (e.g., -11 for cape)
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
- `send_message` - Send a system message to the character (params: `messageType`, `message`) - maps to `cm.playerMessage()`
  - `messageType` values: `"NOTICE"` (type 0), `"POP_UP"` (type 1), `"PINK_TEXT"` (type 5), `"BLUE_TEXT"` (type 6)

**Local Operations:**
- `local:generate_hair_styles` - Generate hair styles (params: `baseStyles`, `genderFilter`, etc.)
- `local:generate_hair_colors` - Generate hair colors (params: `colors`, etc.)
- `local:generate_face_styles` - Generate face styles (params: `baseStyles`, etc.)
- `local:generate_face_colors_for_onetime_lens` - Generate face colors filtered by owned one-time lens items (params: `validateExists`, `excludeEquipped`, `outputContextKey`)
- `local:select_random_cosmetic` - Random selection (params: `stylesContextKey`, `outputContextKey`)
- `local:select_random_weighted` - Weighted random selection (params: `items`, `weights`, `outputContextKey`)
- `local:fetch_map_player_counts` - Fetch player counts (params: `mapIds`)
- `local:calculate_lens_coupon` - Calculate one-time lens item ID from face (params: `selectedFaceContextKey`, `outputContextKey`)
- `local:get_saved_location` - Fetch saved location into context (params: `locationType`, `defaultMapId`, `mapIdContextKey`, `portalIdContextKey`)
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

#### Detailed: local:calculate_lens_coupon

Calculates the one-time cosmetic lens item ID based on a selected face ID. Used by beauty NPCs that offer one-time lens changes.

**Parameters:**
- `selectedFaceContextKey` (string, required) - Context key containing the selected face ID
- `outputContextKey` (string, required) - Context key to store the calculated lens item ID

**Formula:**
```
lensItemId = 5152100 + (selectedFace / 100) % 10
```

The color is encoded in the hundreds place of face IDs:
- Face 20000 → color 0 → item 5152100
- Face 20100 → color 1 → item 5152101
- Face 20200 → color 2 → item 5152102
- etc.

**Example Usage:**

```json
{
  "id": "applyOneTimeLens",
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "local:calculate_lens_coupon",
        "params": {
          "selectedFaceContextKey": "selectedFace",
          "outputContextKey": "lensItemId"
        }
      }
    ],
    "outcomes": [
      {
        "conditions": [],
        "nextState": "consumeAndApply"
      }
    ]
  }
},
{
  "id": "consumeAndApply",
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "destroy_item",
        "params": {
          "itemId": "{context.lensItemId}",
          "quantity": "1"
        }
      },
      {
        "type": "change_face",
        "params": {
          "styleId": "{context.selectedFace}"
        }
      }
    ],
    "outcomes": [
      {
        "conditions": [],
        "nextState": "success"
      }
    ]
  }
}
```

#### Detailed: local:generate_face_colors_for_onetime_lens

Generates face color variants based on which one-time cosmetic lens items (5152100-5152107) the character owns. This is used by beauty NPCs like Dr.Roberts that allow using one-time lens coupons.

**Parameters:**
- `validateExists` (string, optional) - If "true", validates that generated face IDs exist
- `excludeEquipped` (string, optional) - If "true", excludes the currently equipped face color
- `outputContextKey` (string, optional) - Context key to store the generated face colors (defaults to "onetimeLensColors")

**Behavior:**
- Checks inventory for items 5152100-5152107 (one-time lens coupons)
- For each item the character owns, generates the corresponding face color:
  - Item 5152100 → color offset 0 (base color)
  - Item 5152101 → color offset 100
  - Item 5152102 → color offset 200
  - ... and so on up to 5152107 → color offset 700
- Face IDs are calculated as: genderOffset + baseFace + colorOffset
  - Male genderOffset: 20000
  - Female genderOffset: 21000
- Returns an error if the character has no one-time lens items

**Example Usage:**

```json
{
  "id": "prepareOnetimeLens",
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "local:generate_face_colors_for_onetime_lens",
        "params": {
          "validateExists": "true",
          "excludeEquipped": "true",
          "outputContextKey": "onetimeLensColors"
        }
      }
    ],
    "outcomes": [
      {
        "conditions": [],
        "nextState": "chooseOnetimeLens"
      }
    ]
  }
},
{
  "id": "chooseOnetimeLens",
  "type": "askStyle",
  "askStyle": {
    "text": "What kind of lens would you like to wear? Please choose the style of your liking.",
    "stylesContextKey": "onetimeLensColors",
    "contextKey": "selectedFace",
    "nextState": "calculateOnetimeLensCoupon"
  }
}
```

**Typical Flow for One-Time Lens NPCs:**
1. Check if character has any one-time lens items (5152100-5152107)
2. Use `local:generate_face_colors_for_onetime_lens` to generate available colors
3. Use `askStyle` to let player select a color
4. Use `local:calculate_lens_coupon` to determine which item to consume
5. Consume the item and apply the face change

#### Detailed: local:get_saved_location

Fetches a saved location for a character and stores the map ID and portal ID in context. This is used by "return" NPCs that need to display the destination in dialogue before warping.

**Parameters:**
- `locationType` (string, required) - The saved location type (e.g., "FLORINA", "FREE_MARKET", "WORLDTOUR")
- `defaultMapId` (string, optional) - Fallback map ID if no saved location exists
- `mapIdContextKey` (string, optional) - Context key to store the map ID (defaults to "returnMapId")
- `portalIdContextKey` (string, optional) - Context key to store the portal ID (defaults to "returnPortalId")

**Behavior:**
- Queries atlas-character for the saved location by type
- If found, stores mapId and portalId in the specified context keys
- If not found and `defaultMapId` is provided, uses the default (portalId = 0)
- If not found and no default is provided, returns an error

**JavaScript Mapping:**
- `cm.getPlayer().peekSavedLocation("FLORINA")` → `local:get_saved_location` (peek into context)
- `cm.getPlayer().getSavedLocation("FLORINA")` + `cm.warp(returnmap)` → `warp_to_saved_location` (pop and warp)

**Example Usage:**

```json
{
  "id": "fetchSavedLocation",
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "local:get_saved_location",
        "params": {
          "locationType": "FLORINA",
          "defaultMapId": "104000000",
          "mapIdContextKey": "returnMapId",
          "portalIdContextKey": "returnPortalId"
        }
      }
    ],
    "outcomes": [
      {
        "conditions": [],
        "nextState": "askLeave"
      }
    ]
  }
},
{
  "id": "askLeave",
  "type": "dialogue",
  "dialogue": {
    "dialogueType": "sendNext",
    "text": "Do you want to return to #b#m{context.returnMapId}##k?",
    "choices": [
      {"text": "Next", "nextState": "confirmWarp"},
      {"text": "Exit", "nextState": null}
    ]
  }
},
{
  "id": "confirmWarp",
  "type": "genericAction",
  "genericAction": {
    "operations": [
      {
        "type": "warp_to_saved_location",
        "params": {
          "locationType": "FLORINA"
        }
      }
    ],
    "outcomes": [
      {
        "conditions": [],
        "nextState": null
      }
    ]
  }
}
```

**Common Location Types:**
- `FLORINA` - Return from Florina Beach
- `FREE_MARKET` - Return from Free Market
- `WORLDTOUR` - Return from World Tour destinations
- `MIRROR` - Return from Mirror of Dimension
- `ARIANT` - Return from Ariant Coliseum
- `DOJO` - Return from Mu Lung Dojo
- `BOATS` - Return from boat transit

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

## 🔄 Cross-NPC State Tracking (PartyQuestItem Pattern)

Many original scripts use `gotPartyQuestItem`, `setPartyQuestItemObtained`, and `removePartyQuestItem` as **persistent character-level progress flags** for tracking multi-step quest chains across multiple NPCs. Despite the name, these are NOT actual party quest items — they are simple key-value flags stored on the character.

### The Pattern

```javascript
// NPC A (e.g., Ossyria NPC) sets a flag:
cm.getPlayer().setPartyQuestItemObtained("JB3");

// NPC B (e.g., Grendel) reads the flag:
if (cm.getPlayer().gotPartyQuestItem("JB3")) { ... }

// NPC B clears it and sets a new one:
cm.getPlayer().removePartyQuestItem("JB3");
cm.getPlayer().setPartyQuestItemObtained("JBP");

// NPC B later checks the new flag:
if (cm.getPlayer().gotPartyQuestItem("JBP") && cm.haveItem(4031059)) { ... }
```

### Atlas Mapping: Use Quest Status

Map these flags to **quest status tracking** using `questStatus` conditions and `start_quest`/`complete_quest`/`set_quest_progress` operations.

**General approach:**
1. Each flag transition maps to a quest state change
2. Use the quest IDs referenced in the script comments (e.g., "Custom Quest 100100, 100101")
3. If no quest IDs are mentioned, assign logical quest IDs based on the NPC/job advancement

**Function mapping:**

| Original JS | Atlas JSON Equivalent |
|---|---|
| `gotPartyQuestItem("KEY")` | `questStatus` condition (check if quest is STARTED=2 or COMPLETED=3) |
| `setPartyQuestItemObtained("KEY")` | `start_quest` operation (sets quest to STARTED) |
| `removePartyQuestItem("KEY")` | `complete_quest` operation (sets quest to COMPLETED), or `set_quest_progress` to advance state |

**Example — 3rd Job Advancement (Magician):**

The script uses flags JB3 and JBP with quests 100100 and 100101:

| Flag | Quest Mapping | Meaning |
|---|---|---|
| JB3 set | Quest 100100 STARTED (status=2) | Ossyria NPC sent character to job instructor |
| JB3 removed + JBP set | Quest 100100 COMPLETED (status=3) + Quest 100101 STARTED (status=2) | Job instructor briefed character, go defeat clone |
| JBP removed | Quest 100101 COMPLETED (status=3) | Character proved worthy, proceed to next step |

**Condition examples:**
```json
// Check gotPartyQuestItem("JB3") — quest 100100 is STARTED
{ "type": "questStatus", "operator": "=", "value": "2", "referenceId": "100100" }

// Check !gotPartyQuestItem("JBP") — quest 100101 is NOT started
{ "type": "questStatus", "operator": "<", "value": "2", "referenceId": "100101" }

// Check gotPartyQuestItem("JBP") — quest 100101 is STARTED
{ "type": "questStatus", "operator": "=", "value": "2", "referenceId": "100101" }
```

**Operation examples:**
```json
// setPartyQuestItemObtained("JBP") → start quest 100101
{ "type": "start_quest", "params": { "questId": "100101" } }

// removePartyQuestItem("JBP") → complete quest 100101
{ "type": "complete_quest", "params": { "questId": "100101", "force": "true" } }
```

### Tips

- Look for quest IDs in the script **comments** at the top of the file
- When multiple flags are used in sequence, map them to sequential quest states
- Use `force: "true"` on `complete_quest` when completing quests that have no formal requirements defined
- The same pattern appears in all job advancement NPCs (warriors, bowmen, thieves, pirates) — the flag names change but the mechanism is identical

---

## 🎯 Summary: The Golden Rule

**"Read the implementation first, write the conversion second, ask when uncertain."**

Never write output files based on assumptions. Always validate first.