---
description: Convert map entry JavaScript script to JSON rules format
argument-hint: Path to map script file (e.g., "tmp/scripts/map/onUserEnter/go1010100.js")
---

You are an AI implementer converting MapleStory map entry scripts from JavaScript to a structured JSON rules format.

## Reference Files

1. **Schema**: `services/atlas-map-actions/docs/map_script_schema.json` - Your output MUST conform to this (read upfront)
2. **Map Reference**: `docs/Map.txt` - Map IDs to human-readable names (use Grep to look up specific IDs, DO NOT read entire file)

**IMPORTANT - Token Efficiency:**
Map.txt is a very large file. Only use Grep to search for specific IDs as you encounter them.

## Map Script Concepts

Map scripts define behavior for two entry points:
- **onUserEnter**: Runs every time a character enters the map (e.g., map effects, monster spawns)
- **onFirstUserEnter**: Runs only on a character's first visit to the map (e.g., intros, cutscenes)

Both use `ms` (map script manager) to access map state and execute operations.

The script type is determined by the directory the source file is in:
- `tmp/scripts/map/onUserEnter/` → `onUserEnter`
- `tmp/scripts/map/onFirstUserEnter/` → `onFirstUserEnter`

## Supported Features

### Condition Types
| Type | When to Use | Required Fields |
|------|-------------|-----------------|
| `map_id` | `ms.getMapId()` checks | `operator`, `value` |
| `job` | `ms.getPlayer().getJob()` checks | `operator`, `value` |
| `level` | `ms.getPlayer().getLevel()` checks | `operator`, `value` |
| `quest_status` | `ms.getQuestStatus(id)` checks | `operator`, `value`, `referenceId` |

### Operation Types
| Type | When to Use | Params |
|------|-------------|--------|
| `field_effect` | `ms.fieldEffect(path)` | `path` |
| `unlock_ui` | `ms.unlockUI()` | (none) |
| `spawn_monster` | `ms.spawnMonster(id, x, y)` or `spawnMob(x, y, id, map)` | `monsterId`, `x`, `y`, `count` (optional), `mapId` (optional) |
| `show_intro` | `ms.showIntro(path)` | `path` |
| `drop_message` | `ms.dropMessage(msg)` or `ms.getPlayer().dropMessage(type, msg)` | `message`, `messageType` (optional, default: PINK_TEXT) |

### NOT YET SUPPORTED (Skip These Scripts)
The following patterns require additional design work:
- `ms.startExplorerExperience()` - Explorer intro sequence
- `ms.getPlayer().resetEnteredScript()` - Reset first-enter state
- `ms.getClient()` / `ms.getChannelServer()` - Client/server operations
- `ms.getEventManager()` / `eim` operations - Event instance management
- `MapleLifeFactory.getMonster()` with direct Java calls - We handle spawn via saga, not direct Java
- `ms.getPlayer().getMap().getMonsterById()` - Monster existence checks (dedup logic)
- `cm.getEventManager()` / `em.startInstance()` - Event manager transport (use `transportAction` in NPC conversations instead)
- Complex timer/scheduling logic

If a script uses these features, **skip the conversion** and report it as "requires unsupported features."

**EXCEPTION**: If the unsupported feature is `resetEnteredScript()` and the rest of the script only uses supported operations, you may convert the script and note in the description that `resetEnteredScript` is not yet supported.

## Conversion Requirements

### 1. Analyze the JavaScript Script

Identify the key patterns:

**Supported Condition Checks:**
- `ms.getMapId() == 108010301` → `map_id` condition with `=` operator
- `ms.getPlayer().getJob() == 100` → `job` condition
- `ms.getPlayer().getLevel() >= 10` → `level` condition
- `ms.getQuestStatus(2175) == 1` → `quest_status` condition with `referenceId`

**Supported Operations:**
- `ms.fieldEffect("maplemap/enter/1010100")` → `field_effect` operation
- `ms.unlockUI()` → `unlock_ui` operation
- `ms.spawnMonster(9300331, -28, 0)` → `spawn_monster` operation
- `ms.showIntro("Effect/Direction3.img/swordman/Scene0")` → `show_intro` operation
- `ms.dropMessage(msg)` → `drop_message` operation
- `spawnMob(x, y, id, map)` helper → `spawn_monster` operation (extract params)

### 2. Convert to Rules Format

Map scripts become a list of **rules** evaluated in order. First matching rule's operations are executed.

**Key Principles:**
- Each `if` branch becomes a rule
- Conditions within an `if` become the rule's conditions array (AND logic)
- Operations in the branch body go in the `operations` array
- The final `else` or fallback becomes a rule with empty conditions `[]`
- Empty conditions `[]` means the rule always matches

### 3. Handle Common Patterns

**Pattern A: Simple Map Effect (most common)**
```javascript
function start(ms) {
    ms.fieldEffect("maplemap/enter/1010100");
}
```
→ Single rule with empty conditions, `field_effect` operation

**Pattern B: Unlock UI + Effect**
```javascript
function start(ms) {
    ms.unlockUI();
    ms.fieldEffect("maplemap/enter/1020000");
}
```
→ Single rule with two operations: `unlock_ui` then `field_effect`

**Pattern C: Map ID Conditional Spawn**
```javascript
if (ms.getMapId() == 108010301) {
    spawnMob(188, 20, 9001000, ms.getPlayer().getMap());
} else if (ms.getMapId() == 108010201) {
    spawnMob(188, 20, 9001001, ms.getPlayer().getMap());
}
```
→ Multiple rules, each with `map_id` condition and `spawn_monster` operation

**Pattern D: Quest Status Check**
```javascript
if (ms.getQuestStatus(2175) == 1) {
    ms.spawnMonster(9300156, -1027, 216);
}
```
→ Rule with `quest_status` condition (referenceId: "2175", value: "1")

**Pattern E: First User Enter with Monster Spawn**
```javascript
function start(ms) {
    ms.spawnMonster(9300331, -28, 0);
}
```
→ Single rule with empty conditions, `spawn_monster` operation

### 4. Determine Script Name and Type

- **Script name**: Extracted from the filename without extension (e.g., `go1010100.js` → `"go1010100"`, `108010301.js` → `"108010301"`)
- **Script type**: Determined by the parent directory:
  - `onUserEnter/` → the file goes in `scripts/map/onUserEnter/`
  - `onFirstUserEnter/` → the file goes in `scripts/map/onFirstUserEnter/`
- Note: `scriptType` is NOT included in the JSON file itself - it's determined by directory placement

### 5. Validation Checklist

- [ ] Every `if` branch maps to a rule
- [ ] Rule order matches script logic (first match wins)
- [ ] Default/fallback case has empty conditions `[]`
- [ ] All map IDs looked up in Map.txt for description
- [ ] Operations match script actions in correct order
- [ ] Conditions properly typed with correct operators
- [ ] Script does NOT use unsupported features
- [ ] Output conforms to schema

## Task

The script to convert: **$ARGUMENTS**

**Steps:**
1. Read the map script schema first
2. If a file path is provided, read the script file
3. **Check for unsupported features** - If the script uses any of:
   - `startExplorerExperience()`, `resetEnteredScript()`, `getClient()`, `getChannelServer()`
   - `getEventManager()`, `getEventInstance()`, `eim.*` methods
   - Direct Java type calls (`Java.type(...)`)
   - `getMap().getMonsterById()` (dedup checks)

   Then **STOP** and report: "Script uses unsupported features: [list features]. Skipping conversion."

4. Analyze the script:
   - Identify all condition checks
   - Identify all operations
   - Map the control flow to rules
5. Use Grep to look up map IDs in `docs/Map.txt` (pattern: `^<mapId> - `)
6. Determine the script type from the file path
7. Convert to JSON following the schema
8. Validate:
   - [ ] Schema conformance
   - [ ] Rule order matches script logic
   - [ ] All conditions properly typed
   - [ ] All operations have required params
9. Determine output filename: same as input filename but with `.json` extension
10. Determine output directory based on script type:
    - `onUserEnter` → `services/atlas-map-actions/scripts/map/onUserEnter/`
    - `onFirstUserEnter` → `services/atlas-map-actions/scripts/map/onFirstUserEnter/`
11. Write the output file
12. Report completion with summary

## Example Conversion

**Input: go1010100.js** (from `onUserEnter/`)
```javascript
function start(ms) {
    ms.fieldEffect("maplemap/enter/1010100");
}
```

**Output: go1010100.json** (in `scripts/map/onUserEnter/`)
```json
{
  "scriptName": "go1010100",
  "description": "Mushroom Town map entrance effect",
  "rules": [
    {
      "id": "show_field_effect",
      "conditions": [],
      "operations": [
        {
          "type": "field_effect",
          "params": {
            "path": "maplemap/enter/1010100"
          }
        }
      ]
    }
  ]
}
```

## Example: Multi-Operation Script

**Input: go1020000.js** (from `onUserEnter/`)
```javascript
function start(ms) {
    ms.unlockUI();
    ms.fieldEffect("maplemap/enter/1020000");
}
```

**Output: go1020000.json**
```json
{
  "scriptName": "go1020000",
  "description": "Perion entrance - unlock UI and show field effect",
  "rules": [
    {
      "id": "unlock_and_effect",
      "conditions": [],
      "operations": [
        {
          "type": "unlock_ui"
        },
        {
          "type": "field_effect",
          "params": {
            "path": "maplemap/enter/1020000"
          }
        }
      ]
    }
  ]
}
```

## Example: Conditional Monster Spawn

**Input: 108010301.js** (from `onUserEnter/`)
```javascript
function start(ms) {
    if (ms.getMapId() == 108010101) {
        spawnMob(188, 20, 9001002, ms.getPlayer().getMap());
    } else if (ms.getMapId() == 108010301) {
        spawnMob(188, 20, 9001000, ms.getPlayer().getMap());
    }
}
function spawnMob(x, y, id, map) {
    // helper function
}
```

**Output: 108010301.json**
```json
{
  "scriptName": "108010301",
  "description": "Job advancement test maps - spawns job-specific test monster based on map ID",
  "rules": [
    {
      "id": "spawn_archer_test",
      "conditions": [
        {
          "type": "map_id",
          "operator": "=",
          "value": "108010101"
        }
      ],
      "operations": [
        {
          "type": "spawn_monster",
          "params": {
            "monsterId": "9001002",
            "x": "188",
            "y": "20"
          }
        }
      ]
    },
    {
      "id": "spawn_warrior_test",
      "conditions": [
        {
          "type": "map_id",
          "operator": "=",
          "value": "108010301"
        }
      ],
      "operations": [
        {
          "type": "spawn_monster",
          "params": {
            "monsterId": "9001000",
            "x": "188",
            "y": "20"
          }
        }
      ]
    }
  ]
}
```

## Example: Unsupported Script (Skip)

**Input: goSwordman.js**
```javascript
function start(ms) {
    ms.startExplorerExperience();
}
```

**Output:**
```
Script uses unsupported features: startExplorerExperience()
Skipping conversion. This script requires explorer experience support which is not yet implemented.
```

Begin conversion now.
