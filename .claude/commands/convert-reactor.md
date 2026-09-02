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
| `reactor_state` | `rm.getReactor().getState()` checks | `type`, `operator`, `value` |
| `pq_custom_data` | `eim.getIntProperty(k)` / `eim.getProperty(k)` comparisons | `type`, `operator`, `value`, `step` (the custom-data key name) |

Operators for both types: `=`, `!=`, `>`, `<`, `>=`, `<=`.

### Operation Types
| Type | When to Use | Params |
|------|-------------|--------|
| `drop_items` | `rm.dropItems()` / `rm.dropItems(meso, mesoChance, minMeso, maxMeso, minItems)` | `meso`, `mesoChance`, `mesoMin`, `mesoMax`, `minItems` (all optional; omit `params` entirely when the call has no arguments) |
| `spray_items` | `rm.sprayItems()` / `rm.sprayItems(meso, mesoChance, minMeso, maxMeso, minItems)` | same five as `drop_items`, read identically — `executeSprayItems` injects `dropType=spray` and delegates |
| `spawn_monster` | `rm.spawnMonster(id)` / `(id, qty)` / `(id, qty, x, y)` | `monsterId` (required), `count` (default `"1"`), `x`, `y` (default: the reactor's own position) |
| `weaken_area_boss` | `rm.weakenAreaBoss(id, msg)` | `monsterId`, `message` |
| `move_environment` | `rm.getMap().moveEnvironment(name, val)` | `name`, `value` |
| `kill_all_monsters` | `rm.getMap().killAllMonsters()` | (none) |
| `drop_message` | `rm.dropMessage(type, msg)` | `type`, `message` |
| `update_pq_state` | `eim.setProperty(k, v)` / `eim.setIntProperty(k, v)` | `updates` (comma-separated `k=v`), `increments` (comma-separated key names incremented by 1) |
| `hit_reactor` | `rm.getMap().getReactorByName(n).hitReactor()` | `reactorName` |
| `broadcast_pq_message` | `eim.dropMessage(type, msg)` | `message`, `type` (optional) |
| `stage_clear_attempt` | `eim.showClearEffect()` + `giveEventPlayersStageReward` | (none) |

**`dropType` is NOT a seed parameter.** It is injected at runtime and must never be written into a file.

### Event-instance (`eim.*`) Mapping

Event-instance scripts ARE supported. Do not skip them. Map as follows:

| Source | Emit |
|---|---|
| `var eim = rm.getEventInstance()` / `rm.getPlayer().getEventInstance()` | nothing — the binding is erased |
| `if (eim != null) { ... }` / `if (rm.getEventInstance() != null) { ... }` | nothing — the null guard is erased; convert the body |
| `eim.getIntProperty("k")` / `eim.getProperty("k")` in a comparison | a `pq_custom_data` condition with `step: "k"` |
| `eim.setProperty("k", "v")` / `eim.setIntProperty("k", <literal>)` | `update_pq_state` with `updates: "k=v"` |
| `var now = eim.getIntProperty("k"); var next = now + 1; eim.setIntProperty("k", next)` | ONE `update_pq_state` with `increments: "k"` — match the three statements as a single idiom |
| `eim.dropMessage(type, msg)` | `broadcast_pq_message` |
| `eim.showClearEffect()` with `giveEventPlayersStageReward` | `stage_clear_attempt` |

Still genuinely unsupported (report rather than guess): `rm.getMap().getSummonState()`, `getEm().getIv().invokeFunction()`, and any `Math.random()` branch.

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
→ Single actRule with `drop_items` operation. Arguments are positional: `(meso, mesoChance, minMeso, maxMeso, minItems)`.
```json
{
  "type": "drop_items",
  "params": {
    "meso": "true",
    "mesoChance": "2",
    "mesoMin": "8",
    "mesoMax": "15",
    "minItems": "1"
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
→ hitRule with a `reactor_state` condition **inverted to the positive form** (`operator: "="`, `value: "0"`), then the operation. The source's `!== 0 → return` means "act only when the state IS 0".

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
3. Identify any genuinely unsupported constructs (`getSummonState()`, `getEm().getIv().invokeFunction()`, `Math.random()` branches) and flag them for report rather than guessing at a conversion.
4. Analyze the script:
   - Identify `hit()` function contents
   - Identify `act()` function contents
   - Note all conditions and operations, including `eim.*` idioms per the mapping table above
5. Extract reactor ID from filename (e.g., `2000.js` → `"2000"`)
6. Extract description from comments (if present)
7. Convert to JSON following the schema
8. Validate:
   - [ ] Schema conformance
   - [ ] Rule structure correct
   - [ ] All conditions properly typed
   - [ ] All operations have required params
9. Determine the output filename: `reactor-<reactorId>.json` (hyphen, not underscore).
10. Write the file, byte-identical, into all eleven tenant seed directories:
    `deploy/seed/gms/{12_1,48_1,61_1,72_1,79_1,83_1,84_1,87_1,92_1,95_1}/reactor-actions/reactors/`
    and `deploy/seed/jms/185_1/reactor-actions/reactors/`.
    The file is a JSON:API envelope:
    `{"data":{"attributes":{...},"id":"<reactorId>","type":"reactor-action"}}`
    2-space indented, alphabetically keyed, LF, one trailing newline.
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

**Output: reactor-2000.json**
```json
{
  "data": {
    "attributes": {
      "actRules": [
        {
          "conditions": [],
          "id": "drop_items",
          "operations": [
            {
              "params": {
                "meso": "true",
                "mesoChance": "2",
                "mesoMin": "8",
                "mesoMax": "15",
                "minItems": "1"
              },
              "type": "drop_items"
            }
          ]
        }
      ],
      "description": "Maple Island Box - drops various items, notably quest items",
      "hitRules": [],
      "reactorId": "2000"
    },
    "id": "2000",
    "type": "reactor-action"
  }
}
```

## Example: Simple Drop

**Input: 200.js**
```javascript
function act() {
    rm.dropItems();
}
```

**Output: reactor-200.json**
```json
{
  "data": {
    "attributes": {
      "actRules": [
        {
          "conditions": [],
          "id": "drop_items",
          "operations": [
            {
              "type": "drop_items"
            }
          ]
        }
      ],
      "description": "Basic reactor - drops items",
      "hitRules": [],
      "reactorId": "200"
    },
    "id": "200",
    "type": "reactor-action"
  }
}
```

The bare `rm.dropItems()` call has no arguments, so the operation object carries only `"type"` — no `params` key at all.

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

**Output: reactor-2119000.json**
```json
{
  "data": {
    "attributes": {
      "actRules": [],
      "description": "Tombstone in Forest of Dead Trees I - weakens Lich when hit in state 0",
      "hitRules": [
        {
          "conditions": [
            {
              "operator": "=",
              "type": "reactor_state",
              "value": "0"
            }
          ],
          "id": "weaken_area_boss",
          "operations": [
            {
              "params": {
                "message": "As the tombstone lit up and vanished, Lich lost all his magic abilities.",
                "monsterId": "6090000"
              },
              "type": "weaken_area_boss"
            }
          ]
        }
      ],
      "reactorId": "2119000"
    },
    "id": "2119000",
    "type": "reactor-action"
  }
}
```

**Note:** The original script uses `!== 0` to return early (do nothing). We invert this to `= 0` — the positive form of the guard — meaning "act only when the state IS 0". Empty `act()` results in empty `actRules`.

## Example: Event-Instance (`eim.*`) Script

**Input: 2512001.js**
```javascript
function act() {
    var eim = rm.getPlayer().getEventInstance();
    if (eim != null) {
        var opened = eim.getIntProperty("openedChests");
        eim.setIntProperty("openedChests", opened + 1);
        rm.sprayItems(true, 1, 50, 100, 15);
    }
}
```

**Output: reactor-2512001.json**
```json
{
  "data": {
    "attributes": {
      "actRules": [
        {
          "conditions": [],
          "id": "update_pq_state_spray_items",
          "operations": [
            {
              "params": {
                "increments": "openedChests"
              },
              "type": "update_pq_state"
            },
            {
              "params": {
                "meso": "true",
                "mesoChance": "1",
                "mesoMax": "100",
                "mesoMin": "50",
                "minItems": "15"
              },
              "type": "spray_items"
            }
          ]
        }
      ],
      "description": "Pirate PQ treasure chest - increments openedChests and sprays items",
      "hitRules": [],
      "reactorId": "2512001"
    },
    "id": "2512001",
    "type": "reactor-action"
  }
}
```

**Note:** The `eim` binding and its null guard are erased entirely — neither becomes a condition. The `getIntProperty`/`setIntProperty` increment idiom collapses into a single `update_pq_state` operation with `increments: "openedChests"`. `rm.sprayItems(meso, mesoChance, minMeso, maxMeso, minItems)` reads its params identically to `drop_items`.

Begin conversion now.
