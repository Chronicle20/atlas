---
description: Convert reactor JavaScript script to JSON rules format
argument-hint: Path to reactor script file (e.g., "tmp/reactor/2000.js")
---

You are an AI implementer converting MapleStory reactor scripts from JavaScript to a structured JSON rules format.

## Reference Files

1. **Schema**: `services/atlas-reactor-actions/docs/reactor_script_schema.json` - Your output MUST conform to this (read upfront)
2. **Map Reference**: `docs/Map.txt` - Map IDs to human-readable names (use Grep to look up specific IDs, DO NOT read entire file)

**IMPORTANT - Token Efficiency:**
Map.txt is a very large file. Only use Grep to search for specific IDs as you encounter them.

## Reactor Script Concepts

Reactor scripts define behavior for two entry points:
- `hit()` - Called when a player hits/attacks the reactor
- `act()` - Called when the reactor triggers (reaches final state)

Both functions use `rm` (reactor manager) to access reactor state and execute operations.

## Supported Features

### Condition Types
| Type | When to Use | Required Fields |
|------|-------------|-----------------|
| `reactor_state` | `rm.getReactor().getState()` checks | `operator`, `value` |

### Operation Types
| Type | When to Use | Params |
|------|-------------|--------|
| `drop_items` | `rm.dropItems()` or `rm.dropItems(meso, min, max, range, item)` | `meso`, `minMeso`, `maxMeso`, `mesoRange`, `item` (all optional) |
| `spawn_monster` | `rm.spawnMonster(id)` | `monsterId`, `count` (optional, default "1") |
| `spray_items` | `rm.sprayItems()` | (none) |
| `weaken_area_boss` | `rm.weakenAreaBoss(id, msg)` | `monsterId`, `message` |
| `move_environment` | `rm.getMap().moveEnvironment(name, val)` | `name`, `value` |
| `kill_all_monsters` | `rm.getMap().killAllMonsters()` | (none) |
| `drop_message` | `rm.dropMessage(type, msg)` | `type`, `message` |

### NOT YET SUPPORTED (Skip These Scripts)
The following patterns require event instance support:
- `rm.getEventInstance()` / `eim` operations
- `eim.getProperty()`, `eim.setProperty()`, `eim.getIntProperty()`, `eim.setIntProperty()`
- `eim.dropMessage()`, `eim.showClearEffect()`, `eim.giveEventPlayersStageReward()`
- `rm.getMap().getReactorByName().hitReactor()` - Programmatic reactor hits
- `rm.getMap().getSummonState()` - Map summon state checks
- `rm.getClient()` - Client-specific operations beyond standard params
- `rm.getEventInstance().getEm().getIv().invokeFunction()` - Event scripting

If a script uses these features, **skip the conversion** and report it as "requires unsupported features."

## Conversion Requirements

### 1. Analyze the JavaScript Script

Identify:
- Does it have a `hit()` function? What does it do?
- Does it have an `act()` function? What does it do?
- What conditions are checked (state checks)?
- What operations are performed?

### 2. Convert to Rules Format

**Key Principles:**
- `hit()` function body becomes `hitRules` array
- `act()` function body becomes `actRules` array
- Each `if` branch becomes a rule
- Conditions within `if` become the rule's conditions array (AND logic)
- Operations become the rule's operations array
- Empty/fallback rules have empty conditions `[]`

### 3. Handle Common Patterns

**Pattern A: Simple Drop (most common)**
```javascript
function act() {
    rm.dropItems();
}
```
→ Single actRule with no conditions, `drop_items` operation with no params

**Pattern B: Drop with Parameters**
```javascript
function act() {
    rm.dropItems(true, 2, 8, 15, 1);
}
```
→ Single actRule with `drop_items` operation:
```json
{
  "type": "drop_items",
  "params": {
    "meso": "true",
    "minMeso": "2",
    "maxMeso": "8",
    "mesoRange": "15",
    "item": "1"
  }
}
```

**Pattern C: State Check in hit()**
```javascript
function hit() {
    if (rm.getReactor().getState() !== 0) {
        return;
    }
    rm.weakenAreaBoss(6090000, "Message here");
}
```
→ hitRule with `reactor_state` condition (operator: `!=`, value: `0`), then operation

**Pattern D: Monster Spawn**
```javascript
function act() {
    rm.spawnMonster(9300048);
}
```
→ actRule with `spawn_monster` operation, `monsterId` param

**Pattern E: Random Monster**
```javascript
rm.spawnMonster(Math.random() >= .6 ? 9300049 : 9300048);
```
→ For random spawns, create two rules with 50% probability each OR document as TODO
→ Initially: Pick one monster, document the random behavior in description

**Pattern F: Environment Move**
```javascript
rm.getMap().moveEnvironment("trap" + rm.getReactor().getName()[5], 1);
```
→ Operation with dynamic name needs context handling - document as TODO or skip

### 4. Validation Checklist

- [ ] Every `hit()` body maps to hitRules
- [ ] Every `act()` body maps to actRules
- [ ] Empty functions result in empty rules array `[]`
- [ ] Conditions properly extracted from `if` statements
- [ ] Operations have all required params
- [ ] Script does NOT use unsupported features

## Task

The script to convert: **$ARGUMENTS**

**Steps:**
1. Read the reactor script schema first (if it exists; if not, validate structure manually)
2. If a file path is provided, read the script file
3. **Check for unsupported features** - If the script uses any of:
   - `getEventInstance()`, `eim.*` methods
   - `getReactorByName().hitReactor()`
   - `getSummonState()`
   - `getEm().getIv().invokeFunction()`

   Then **STOP** and report: "Script uses unsupported features: [list features]. Skipping conversion."

4. Analyze the script:
   - Identify `hit()` function contents
   - Identify `act()` function contents
   - Note all conditions and operations
5. Extract reactor ID from filename (e.g., `2000.js` → `"2000"`)
6. Extract description from comments (if present)
7. Convert to JSON following the schema
8. Validate:
   - [ ] Schema conformance
   - [ ] Rule structure correct
   - [ ] All conditions properly typed
   - [ ] All operations have required params
9. Determine output filename: `reactor_{reactorId}.json`
10. Write to `services/atlas-reactor-actions/scripts/reactors/` directory
11. Report completion with summary

## Example Conversion

**Input: 2000.js**
```javascript
/* @Author Lerk
 *
 * 2000.js: Maple Island Box - drops various items, notably quest items
 */

function act() {
    rm.dropItems(true, 2, 8, 15, 1);
}
```

**Output: reactor_2000.json**
```json
{
  "reactorId": "2000",
  "description": "Maple Island Box - drops various items, notably quest items",
  "hitRules": [],
  "actRules": [
    {
      "id": "drop_items",
      "conditions": [],
      "operations": [
        {
          "type": "drop_items",
          "params": {
            "meso": "true",
            "minMeso": "2",
            "maxMeso": "8",
            "mesoRange": "15",
            "item": "1"
          }
        }
      ]
    }
  ]
}
```

## Example: Simple Drop

**Input: 200.js**
```javascript
function act() {
    rm.dropItems();
}
```

**Output: reactor_200.json**
```json
{
  "reactorId": "200",
  "description": "Basic reactor - drops items",
  "hitRules": [],
  "actRules": [
    {
      "id": "drop_items",
      "conditions": [],
      "operations": [
        {
          "type": "drop_items"
        }
      ]
    }
  ]
}
```

## Example: Hit and Act

**Input: 2119000.js**
```javascript
/**
    Tombstone in Forest of Dead Trees I
*/
function hit() {
    if (rm.getReactor().getState() !== 0) {
        return
    }
    rm.weakenAreaBoss(6090000, "As the tombstone lit up and vanished, Lich lost all his magic abilities.")
}

function act() {
    // If the chest is destroyed before Riche, killing him should yield no exp
}
```

**Output: reactor_2119000.json**
```json
{
  "reactorId": "2119000",
  "description": "Tombstone in Forest of Dead Trees I - weakens Lich boss",
  "hitRules": [
    {
      "id": "weaken_lich_state_zero",
      "conditions": [
        {
          "type": "reactor_state",
          "operator": "=",
          "value": "0"
        }
      ],
      "operations": [
        {
          "type": "weaken_area_boss",
          "params": {
            "monsterId": "6090000",
            "message": "As the tombstone lit up and vanished, Lich lost all his magic abilities."
          }
        }
      ]
    }
  ],
  "actRules": []
}
```

**Note:** The original script uses `!== 0` to return early (do nothing). We invert this to `= 0` meaning "if state IS 0, execute the operation". Empty `act()` results in empty `actRules`.

## Example: Unsupported Script (Skip)

**Input: 6109000.js**
```javascript
function act() {
    var eim = rm.getEventInstance();
    if (eim != null) {
        var mapId = rm.getMap().getId();
        eim.dropMessage(6, "The Warrior Sigil has been activated!");
        eim.setIntProperty("glpq2", eim.getIntProperty("glpq2") + 1);
        // ...
    }
}
```

**Output:**
```
Script uses unsupported features: getEventInstance(), eim.dropMessage(), eim.setIntProperty(), eim.getIntProperty()
Skipping conversion. This script requires event instance support which is not yet implemented.
```

Begin conversion now.
