---
description: Convert map entry JavaScript script to JSON rules format
argument-hint: Path to map script file (e.g., "tmp/scripts/map/onUserEnter/go1010100.js")
---

You are an AI implementer converting MapleStory map entry scripts from JavaScript to a structured JSON rules format.

## Reference Files

1. **Schema**: `services/atlas-map-actions/docs/map_script_schema.json` - Your output MUST conform to this (read upfront). This file is GENERATED from `libs/atlas-saga/validation.go` and the `ExecuteOperation` switch — it is the authority for every condition `type` and operation `type` name. Do not use a condition or operation name from memory; quote the schema's `enum`.
2. **Formatting exemplar**: `deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json` - matches the envelope, key ordering and indentation your output must reproduce exactly.
3. **Map Reference**: `docs/Map.txt` - Map IDs to human-readable names (use Grep to look up specific IDs, DO NOT read entire file)

**IMPORTANT - Token Efficiency:**
Map.txt is a very large file. Only use Grep to search for specific IDs as you encounter them.

## Map Script Concepts

Map scripts define behavior for two entry points (hooks):
- **onUserEnter**: Runs every time a character enters the map (e.g., map effects, monster spawns)
- **onFirstUserEnter**: Runs only on a character's first visit to the map (e.g., intros, cutscenes)

Both use `ms` (map script manager) to access map state and execute operations.

The hook is determined by the directory the source file is in:
- `tmp/scripts/map/onUserEnter/` → `onUserEnter`
- `tmp/scripts/map/onFirstUserEnter/` → `onFirstUserEnter`

## Output Contract

### Output path

Every converted script is written to **all 11 version roots**, byte-identically:

```
deploy/seed/<client>/<version>/map-actions/<hook>/map-<scriptName>.json
```

where `<hook>` is `onUserEnter` or `onFirstUserEnter`, and `<client>/<version>` ranges over
exactly these 11 roots:

- `gms/12_1`, `gms/48_1`, `gms/61_1`, `gms/72_1`, `gms/79_1`, `gms/83_1`, `gms/84_1`, `gms/87_1`, `gms/92_1`, `gms/95_1`
- `jms/185_1`

`services/atlas-map-actions/scripts/map/<hook>/<name>.json` does **not** exist and is never the
output path. `tools/catalog-lint` fails the build if the file is not replicated byte-identically
across all 11 roots.

### Envelope

The output is a JSON:API document, not a bare schema object:

```json
{
  "data": {
    "type": "map-action",
    "id": "<scriptName>",
    "attributes": {
      "scriptName": "<scriptName>",
      "description": "<description>",
      "rules": [ ... ]
    }
  }
}
```

There is **no `scriptType` attribute anywhere in the file**. The hook is derived from the
directory the file lives in (`script/subdomain_on_user_enter.go`,
`script/subdomain_on_first_user_enter.go`), never from the JSON payload.

### Formatting

2-space indent, object keys sorted alphabetically at every level (including inside `attributes`,
`rules`, `conditions` and `operations`), trailing newline. Match
`deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json` exactly.

## Supported Features

### Condition Types

The valid `type` values are the `condition.properties.type.enum` list in
`services/atlas-map-actions/docs/map_script_schema.json`. The ones you will see most often when
converting map scripts:

| Type | When to Use | Required Fields |
|------|-------------|-----------------|
| `mapId` or `map_id` | `ms.getMapId()` checks | `operator`, `value` |
| `jobId` | `ms.getPlayer().getJob()` checks | `operator`, `value` |
| `level` | `ms.getPlayer().getLevel()` checks | `operator`, `value` |
| `gender` | `ms.getPlayer().getGender()` checks | `operator`, `value` |
| `questStatus` | `ms.getQuestStatus(id)` checks | `operator`, `value`, `referenceId` |

`job` and `quest_status` are **not** valid condition types — the aggregator rejects them. Use
`jobId` and `questStatus`.

### Operation Types

| Type | When to Use | Params |
|------|-------------|--------|
| `field_effect` | `ms.fieldEffect(path)` | `path` |
| `lock_ui` | `ms.lockUI()` | (none) |
| `unlock_ui` | `ms.unlockUI()` | (none) |
| `spawn_monster` | `ms.spawnMonster(id, x, y)` or `spawnMob(x, y, id, map)` | `monsterId` (or `monsterIds`), `x`, `y`, `count` (optional), `mapId` (optional), `spawnIfAbsent` (always `"true"`) |
| `show_intro` | `ms.showIntro(path)` | `path` |
| `drop_message` | `ms.dropMessage(msg)` or `ms.getPlayer().dropMessage(type, msg)` | `message`, `messageType` (optional, default: PINK_TEXT) |

The valid `type` values are the `operation.properties.type.enum` list in the schema, derived from
the `ExecuteOperation` switch in `libs/atlas-saga`. An operation type outside that enum does not
exist in Atlas — the executor now **errors** on an unrecognized operation type rather than
silently ignoring it. Never invent an operation name; if the script needs an operation the enum
doesn't have, treat it as an unsupported feature (see below).

Every operation `params` value is a **string**, even when the source is a number or boolean
(`"x": "188"`, `"spawnIfAbsent": "true"`).

### `spawnIfAbsent` is mandatory on every `spawn_monster` (FR-2.2)

Cosmic guards every map spawn with a check before calling `spawnMonster`
(`map.getMonsterById(id) != null`, `containsNPC`, `countMonster(...) == 0`, etc.) — the JS
equivalent of "don't double-spawn this monster." Atlas has no separate guard condition for this;
instead, **every emitted `spawn_monster` operation must set `"spawnIfAbsent": "true"`**, whether
or not the source script had an explicit guard. `tools/catalog-lint` fails the build if a
`spawn_monster` operation is missing `spawnIfAbsent`.

### Quest-status value shift (FR-1.4)

Cosmic's `QuestStatus` enum is `UNDEFINED(-1), NOT_STARTED(0), STARTED(1), COMPLETED(2)`. Atlas'
aggregator uses a different enum: `UNDEFINED=0, NOT_STARTED=1, STARTED=2, COMPLETED=3`. **Every
ported `getQuestStatus(x) == n` check is emitted with `value` set to `n + 1`, not `n`.**

Worked example — Cosmic:

```javascript
if (ms.getQuestStatus(2175) == 1) {
    ms.spawnMonster(9300156, -1027, 216);
}
```

becomes the condition:

```json
{
  "operator": "=",
  "referenceId": "2175",
  "type": "questStatus",
  "value": "2"
}
```

(Cosmic's `1` (`STARTED`) becomes Atlas' `2` (`STARTED`), not `1`.)

### NOT YET SUPPORTED (Skip These Scripts)
The following patterns require additional design work:
- `ms.startExplorerExperience()` - Explorer intro sequence
- `ms.getClient()` / `ms.getChannelServer()` - Client/server operations
- `ms.getEventManager()` / `eim` operations - Event instance management
- `MapleLifeFactory.getMonster()` with direct Java calls - We handle spawn via saga, not direct Java
- `cm.getEventManager()` / `em.startInstance()` - Event manager transport (use `transportAction` in NPC conversations instead)
- Complex timer/scheduling logic

If a script uses these features, **skip the conversion** and report it as "requires unsupported features."

**EXCEPTION**: If the unsupported feature is `ms.getPlayer().resetEnteredScript()`, Atlas has no
equivalent operation. Convert the rest of the script normally:
- If the script spawns a monster, the emitted `spawn_monster` operation's `spawnIfAbsent: "true"`
  already covers the re-entry-safety that `resetEnteredScript()` provided in Cosmic — no other
  change is needed.
- Otherwise, convert the script as usual and note the omission explicitly in the output's
  `description` (e.g., "resetEnteredScript() not ported — no Atlas equivalent").
`ms.getPlayer().getMap().getMonsterById()` dedup checks are handled the same way: they exist only
to guard a spawn, so they disappear once the corresponding `spawn_monster` sets
`spawnIfAbsent: "true"`.

## Conversion Requirements

### 1. Analyze the JavaScript Script

Identify the key patterns:

**Supported Condition Checks:**
- `ms.getMapId() == 108010301` → `mapId` condition with `=` operator
- `ms.getPlayer().getJob() == 100` → `jobId` condition
- `ms.getPlayer().getLevel() >= 10` → `level` condition
- `ms.getPlayer().getGender() == 0` → `gender` condition
- `ms.getQuestStatus(2175) == 1` → `questStatus` condition with `referenceId: "2175"` and `value` shifted by +1 (see Quest-status value shift above)

**Supported Operations:**
- `ms.fieldEffect("maplemap/enter/1010100")` → `field_effect` operation
- `ms.lockUI()` → `lock_ui` operation
- `ms.unlockUI()` → `unlock_ui` operation
- `ms.spawnMonster(9300331, -28, 0)` → `spawn_monster` operation with `spawnIfAbsent: "true"`
- `ms.showIntro("Effect/Direction3.img/swordman/Scene0")` → `show_intro` operation
- `ms.dropMessage(msg)` → `drop_message` operation
- `spawnMob(x, y, id, map)` helper → `spawn_monster` operation (extract params), always with `spawnIfAbsent: "true"`

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
→ Multiple rules, each with a `mapId` condition and a `spawn_monster` operation (`spawnIfAbsent: "true"`)

**Pattern D: Quest Status Check**
```javascript
if (ms.getQuestStatus(2175) == 1) {
    ms.spawnMonster(9300156, -1027, 216);
}
```
→ Rule with `questStatus` condition (`referenceId: "2175"`, `value: "2"` — shifted from Cosmic's `1`)

**Pattern E: First User Enter with Monster Spawn**
```javascript
function start(ms) {
    ms.spawnMonster(9300331, -28, 0);
}
```
→ Single rule with empty conditions, `spawn_monster` operation with `spawnIfAbsent: "true"`

### 4. Determine Script Name and Hook

- **Script name**: Extracted from the filename without extension (e.g., `go1010100.js` → `"go1010100"`, `108010301.js` → `"108010301"`)
- **Hook**: Determined by the parent directory of the source script:
  - `onUserEnter/` → the file is replicated to `deploy/seed/<client>/<version>/map-actions/onUserEnter/map-<scriptName>.json` in all 11 version roots
  - `onFirstUserEnter/` → the file is replicated to `deploy/seed/<client>/<version>/map-actions/onFirstUserEnter/map-<scriptName>.json` in all 11 version roots
- Note: the hook name is NOT included anywhere in the JSON file itself (no `scriptType` attribute) - it's determined entirely by directory placement

### 5. Validation Checklist

- [ ] Every `if` branch maps to a rule
- [ ] Rule order matches script logic (first match wins)
- [ ] Default/fallback case has empty conditions `[]`
- [ ] All map IDs looked up in Map.txt for description
- [ ] Operations match script actions in correct order
- [ ] Conditions properly typed with correct operators, using the schema's condition names (`jobId`, `questStatus`, never `job`/`quest_status`)
- [ ] Every `getQuestStatus(x) == n` check emitted as `value: n + 1`
- [ ] Every `spawn_monster` operation carries `"spawnIfAbsent": "true"`
- [ ] Script does NOT use unsupported features
- [ ] Output wrapped in the JSON:API envelope (`data.type`, `data.id`, `data.attributes`), no `scriptType`
- [ ] Output written identically to all 11 version roots
- [ ] Output conforms to schema, 2-space indent, alphabetically sorted keys, trailing newline

## Task

The script to convert: **$ARGUMENTS**

**Steps:**
1. Read the map script schema first
2. If a file path is provided, read the script file
3. **Check for unsupported features** - If the script uses any of:
   - `startExplorerExperience()`, `getClient()`, `getChannelServer()`
   - `getEventManager()`, `getEventInstance()`, `eim.*` methods
   - Direct Java type calls (`Java.type(...)`)

   Then **STOP** and report: "Script uses unsupported features: [list features]. Skipping conversion."

   (`resetEnteredScript()` and `getMap().getMonsterById()` dedup checks are NOT unsupported —
   convert per the exception above.)
4. Analyze the script:
   - Identify all condition checks (map to the schema's condition names)
   - Identify all operations (map to the schema's operation names; add `spawnIfAbsent: "true"` to every `spawn_monster`)
   - Apply the quest-status +1 shift to every `questStatus` condition value
   - Map the control flow to rules
5. Use Grep to look up map IDs in `docs/Map.txt` (pattern: `^<mapId> - `)
6. Determine the hook (`onUserEnter` or `onFirstUserEnter`) from the file path
7. Convert to JSON following the schema, wrapped in the JSON:API envelope
8. Validate:
   - [ ] Schema conformance
   - [ ] Rule order matches script logic
   - [ ] All conditions properly typed with the schema's names
   - [ ] All operations have required params, and every `spawn_monster` has `spawnIfAbsent: "true"`
   - [ ] Envelope present, no `scriptType`
9. Determine output filename: `map-<scriptName>.json`
10. Write the output file to `deploy/seed/<client>/<version>/map-actions/<hook>/map-<scriptName>.json` for **all 11 version roots**, byte-identically:
    - `gms/12_1`, `gms/48_1`, `gms/61_1`, `gms/72_1`, `gms/79_1`, `gms/83_1`, `gms/84_1`, `gms/87_1`, `gms/92_1`, `gms/95_1`, `jms/185_1`
11. Report completion with summary

## Example Conversion

**Input: go1010100.js** (from `onUserEnter/`)
```javascript
function start(ms) {
    ms.fieldEffect("maplemap/enter/1010100");
}
```

**Output: map-go1010100.json** (written to `deploy/seed/<client>/<version>/map-actions/onUserEnter/` in all 11 version roots)
```json
{
  "data": {
    "attributes": {
      "description": "Mushroom Town map entrance effect",
      "rules": [
        {
          "conditions": [],
          "id": "show_field_effect",
          "operations": [
            {
              "params": {
                "path": "maplemap/enter/1010100"
              },
              "type": "field_effect"
            }
          ]
        }
      ],
      "scriptName": "go1010100"
    },
    "id": "go1010100",
    "type": "map-action"
  }
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

**Output: map-go1020000.json**
```json
{
  "data": {
    "attributes": {
      "description": "Perion entrance - unlock UI and show field effect",
      "rules": [
        {
          "conditions": [],
          "id": "unlock_and_effect",
          "operations": [
            {
              "type": "unlock_ui"
            },
            {
              "params": {
                "path": "maplemap/enter/1020000"
              },
              "type": "field_effect"
            }
          ]
        }
      ],
      "scriptName": "go1020000"
    },
    "id": "go1020000",
    "type": "map-action"
  }
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

**Output: map-108010301.json**
```json
{
  "data": {
    "attributes": {
      "description": "Job advancement test maps - spawns job-specific test monster based on map ID",
      "rules": [
        {
          "conditions": [
            {
              "operator": "=",
              "type": "mapId",
              "value": "108010101"
            }
          ],
          "id": "spawn_archer_test",
          "operations": [
            {
              "params": {
                "monsterId": "9001002",
                "spawnIfAbsent": "true",
                "x": "188",
                "y": "20"
              },
              "type": "spawn_monster"
            }
          ]
        },
        {
          "conditions": [
            {
              "operator": "=",
              "type": "mapId",
              "value": "108010301"
            }
          ],
          "id": "spawn_warrior_test",
          "operations": [
            {
              "params": {
                "monsterId": "9001000",
                "spawnIfAbsent": "true",
                "x": "188",
                "y": "20"
              },
              "type": "spawn_monster"
            }
          ]
        }
      ],
      "scriptName": "108010301"
    },
    "id": "108010301",
    "type": "map-action"
  }
}
```

## Example: Quest Status Check with Value Shift

**Input: goQuestSpawn.js** (from `onUserEnter/`)
```javascript
function start(ms) {
    if (ms.getQuestStatus(2175) == 1) {
        ms.spawnMonster(9300156, -1027, 216);
    }
}
```

**Output: map-goQuestSpawn.json**
```json
{
  "data": {
    "attributes": {
      "description": "Spawns a monster once the referenced quest has been started",
      "rules": [
        {
          "conditions": [
            {
              "operator": "=",
              "referenceId": "2175",
              "type": "questStatus",
              "value": "2"
            }
          ],
          "id": "spawn_on_quest_started",
          "operations": [
            {
              "params": {
                "monsterId": "9300156",
                "spawnIfAbsent": "true",
                "x": "-1027",
                "y": "216"
              },
              "type": "spawn_monster"
            }
          ]
        }
      ],
      "scriptName": "goQuestSpawn"
    },
    "id": "goQuestSpawn",
    "type": "map-action"
  }
}
```

Cosmic's `getQuestStatus(2175) == 1` (`STARTED` under Cosmic's `-1/0/1/2` enum) is emitted as
`value: "2"` (`STARTED` under Atlas' `0/1/2/3` enum) — the `n + 1` shift.

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
