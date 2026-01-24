---
description: Convert portal JavaScript script to JSON rules format
argument-hint: Path to portal script file (e.g., "services/atlas-npc-conversations/tmp/portal/under30gate.js")
---

You are an AI implementer converting MapleStory portal scripts from JavaScript to a structured JSON rules format.

## Reference Files

1. **Schema**: `services/atlas-portal-actions/docs/portal_script_schema.json` - Your output MUST conform to this (read upfront)
2. **Map Reference**: `docs/Map.txt` - Map IDs to human-readable names (use Grep to look up specific IDs, DO NOT read entire file)
3. **Context Document**: `dev/active/atlas-portal-actions/atlas-portal-actions-context.md` - Patterns and examples

**IMPORTANT - Token Efficiency:**
Map.txt is a very large file. Only use Grep to search for specific IDs as you encounter them.

## Portal Script Concepts

Portal scripts are **simpler than NPC conversations**. They:
- Have a single entry point: `function enter(pi) { ... }`
- Return `true` (allow entry) or `false` (deny entry)
- Execute operations (warp, message) based on conditions
- Do NOT have dialogue or multi-turn interaction

## Supported Features (Initial Version)

### Condition Types
| Type | When to Use | Required Fields |
|------|-------------|-----------------|
| `level` | `getLevel()` checks | `operator`, `value` |
| `job` | `getJob()` checks | `operator`, `value` |
| `item` | `haveItem()` checks | `operator`, `value`, `referenceId` |
| `meso` | Meso checks | `operator`, `value` |
| `quest_status` | Quest completion | `operator`, `value`, `referenceId` |

### Operation Types
| Type | When to Use | Params |
|------|-------------|--------|
| `play_portal_sound` | `pi.playPortalSound()` | - |
| `warp` | `pi.warp(mapId, portalId)` | `mapId`, `portalId` or `portalName` |
| `drop_message` | `dropMessage()` | `message`, `messageType` |
| `block_portal` | `pi.blockPortal()` | - |
| `show_hint` | `pi.showInstruction(msg, width, height)` | `hint`, `width`, `height` |
| `show_info` | `pi.showInfo(path)` | `path` (e.g., "UI/tutorial.img/25") |

### NOT YET SUPPORTED (Skip These Scripts)
The following patterns require additional design work:
- Event instance checks (`getEventInstance()`, `eim.getProperty()`, `isEventCleared()`)
- Event manager checks (`getEventManager()`)
- Saved location warps (`getSavedLocation()`)
- Area info updates (`containsAreaInfo()`, `updateAreaInfo()`)

If a script uses these features, **skip the conversion** and report it as "requires unsupported features."

## Conversion Requirements

### 1. Analyze the JavaScript Script

Identify the key patterns:

**Supported Condition Checks:**
- `pi.getPlayer().getLevel()` → `level` condition
- `pi.getPlayer().getJob()` → `job` condition
- `pi.haveItem(itemId)` → `item` condition
- `pi.getPlayer().getMeso()` → `meso` condition

**Supported Operations:**
- `pi.playPortalSound()` → `play_portal_sound` operation
- `pi.warp(mapId, portalId)` → `warp` operation
- `pi.getPlayer().dropMessage(type, msg)` → `drop_message` operation
- `pi.blockPortal()` → `block_portal` operation
- `pi.showInstruction(msg, width, height)` → `show_hint` operation (params: `hint`, `width`, `height`)
- `pi.showInfo(path)` → `show_info` operation (params: `path`)

### 2. Convert to Rules Format

Portal scripts become a list of **rules** evaluated in order. First matching rule determines the outcome.

**Key Principles:**
- Each `if` branch becomes a rule
- Conditions within an `if` become the rule's conditions array (AND logic)
- The `return true/false` determines `allow: true/false`
- Operations before `return` go in the `operations` array
- The final `else` or fallback becomes a rule with empty conditions `[]`

### 3. Handle Simple Patterns

**Pattern A: Level Check**
```javascript
if (pi.getPlayer().getLevel() <= 30) {
    pi.playPortalSound();
    pi.warp(990000640, 1);
    return true;
} else {
    pi.getPlayer().dropMessage(5, "You cannot proceed.");
    return false;
}
```
→ Two rules: one with level condition, one default

**Pattern B: Job Check**
```javascript
if (pi.getPlayer().getJob() == 0) {
    pi.playPortalSound();
    pi.warp(100000000, 0);
    return true;
}
return false;
```
→ Rule with job condition, default deny rule

**Pattern C: Item Check**
```javascript
if (pi.haveItem(4031045)) {
    pi.playPortalSound();
    pi.warp(101000000, 0);
    return true;
} else {
    pi.getPlayer().dropMessage(5, "You need a ticket to enter.");
    return false;
}
```
→ Rule with item condition, default deny rule

**Pattern D: Simple Warp (No Conditions)**
```javascript
function enter(pi) {
    pi.playPortalSound();
    pi.warp(240050400, "sp");
    return true;
}
```
→ Single rule with empty conditions, always allows

### 4. Validation Checklist

- [ ] Every `if` branch maps to a rule
- [ ] Rule order matches script logic (first match wins)
- [ ] Default/fallback case has empty conditions `[]`
- [ ] All map IDs looked up in Map.txt for description
- [ ] `allow` matches `return true/false` in script
- [ ] Operations match script actions in correct order
- [ ] Script does NOT use unsupported features

## Task

The script to convert: **$ARGUMENTS**

**Steps:**
1. Read the portal script schema first
2. If a file path is provided, read the script file
3. **Check for unsupported features** - If the script uses any of:
   - `getEventInstance()`, `eim.getProperty()`, `isEventCleared()`, `gridCheck()`
   - `getEventManager()`
   - `getSavedLocation()`
   - `containsAreaInfo()`, `updateAreaInfo()`

   Then **STOP** and report: "Script uses unsupported features: [list features]. Skipping conversion."

4. Analyze the script:
   - Identify all condition checks
   - Identify all operations
   - Map the control flow to rules
5. Use Grep to look up map IDs in `docs/Map.txt` (pattern: `^<mapId> - `)
6. Convert to JSON following the schema
7. Validate:
   - [ ] Schema conformance
   - [ ] Rule order matches script logic
   - [ ] All conditions properly typed
   - [ ] All operations have required params
8. Determine output filename: `portal_{scriptName}.json`
9. Write to `services/atlas-portal-actions/scripts/portals/` directory
10. Report completion with summary

## Example Conversion

**Input: under30gate.js**
```javascript
function enter(pi) {
    if (pi.getPlayer().getLevel() <= 30) {
        pi.playPortalSound();
        pi.warp(990000640, 1);
        return true;
    } else {
        pi.getPlayer().dropMessage(5, "You cannot proceed past this point.");
        return false;
    }
}
```

**Output: portal_under30gate.json**
```json
{
  "portalId": "under30gate",
  "mapId": 990000000,
  "description": "Guild Quest - Level 30 and under gate to Sharen III's Grave",
  "rules": [
    {
      "id": "level_check_pass",
      "conditions": [
        {
          "type": "level",
          "operator": "<=",
          "value": "30"
        }
      ],
      "onMatch": {
        "allow": true,
        "operations": [
          { "type": "play_portal_sound" },
          { "type": "warp", "params": { "mapId": "990000640", "portalId": "1" } }
        ]
      }
    },
    {
      "id": "default_deny",
      "conditions": [],
      "onMatch": {
        "allow": false,
        "operations": [
          { "type": "drop_message", "params": { "message": "You cannot proceed past this point." } }
        ]
      }
    }
  ]
}
```

## Example: Simple Warp Portal

**Input: hontale_out1.js**
```javascript
function enter(pi) {
    pi.playPortalSound();
    pi.warp(240050400, "sp");
    return true;
}
```

**Output: portal_hontale_out1.json**
```json
{
  "portalId": "hontale_out1",
  "mapId": 240060000,
  "description": "Horntail Cave exit portal",
  "rules": [
    {
      "id": "always_allow",
      "conditions": [],
      "onMatch": {
        "allow": true,
        "operations": [
          { "type": "play_portal_sound" },
          { "type": "warp", "params": { "mapId": "240050400", "portalName": "sp" } }
        ]
      }
    }
  ]
}
```

## Example: Unsupported Script (Skip)

**Input: kpq0.js**
```javascript
function enter(pi) {
    var eim = pi.getPlayer().getEventInstance();
    if (eim.getProperty("1stageclear") != null) {
        // ...
    }
}
```

**Output:**
```
Script uses unsupported features: getEventInstance(), eim.getProperty()
Skipping conversion. This script requires event instance support which is not yet implemented.
```

Begin conversion now.
