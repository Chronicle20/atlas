# Cosmic Map-Action Parity — Plan B: Category #1 Conversions

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the 27 map-action seed documents that today's engine can already
express, replicated byte-identically to all 11 version roots — 297 files.

**Architecture:** Pure seed authoring. No Go changes, no new saga actions, no new
conditions. Every document uses only the six operations the executor already
implements (`field_effect`, `show_intro`, `spawn_monster`, `drop_message`, `lock_ui`,
`unlock_ui`) and the conditions the generated schema already accepts. Each task
covers one document *shape*: author the document(s) under `deploy/seed/gms/83_1/`,
replicate to the other ten roots with `cp`, then prove correctness with
`tools/catalog-lint` rather than by inspection.

**Tech Stack:** JSON:API seed documents under `deploy/seed/`, validated by
`tools/catalog-lint` (Plan A Task A10) against the generated
`services/atlas-map-actions/docs/map_script_schema.json` (Plan A Task A8).

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md)). Per-script literal values
derived from the Cosmic reference server's `scripts/map/onUserEnter/` directory
(`<cosmic-root>` in [context.md](context.md), which records the derivation and the
checkout path).

## Global Constraints

- **[plan.md](plan.md) (Plan A) must be green first.** Its `tools/verify.sh` gate is
  a precondition: this plan's documents use `spawnIfAbsent`, which the schema does
  not accept and `catalog-lint` does not require until Plan A lands.
- **11 version roots, byte-identical.** `deploy/seed/gms/{12_1,48_1,61_1,72_1,79_1,83_1,84_1,87_1,92_1,95_1}`
  and `deploy/seed/jms/185_1`. Author under `gms/83_1`, then `cp` to the other ten.
  `catalog-lint` fails on any drift or omission.
- **Envelope:** `{"data": {"type": "map-action", "id": "<scriptName>", "attributes": {"scriptName": "<scriptName>", "description": "...", "rules": [...]}}}`.
  **No `scriptType`** — the hook is the directory.
- **Path:** `deploy/seed/<root>/map-actions/onUserEnter/map-<scriptName>.json`. All 27
  scripts in this plan are `onUserEnter`; none exists under Cosmic's
  `onFirstUserEnter/` (which holds only `dojang_1st.js` and `spaceGaGa_sMap.js`).
- **Formatting:** 2-space indent, object keys sorted **alphabetically at every
  level**, trailing newline. Exemplar:
  `deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json`. Within a rule the
  key order is therefore `conditions`, `id`, `operations`; within a condition
  `operator`, `referenceId`, `type`, `value`; within an operation `params`, `type`;
  within `attributes` `description`, `rules`, `scriptName`.
- **Every `spawn_monster` sets `"spawnIfAbsent": "true"`** (PRD FR-2.2). Cosmic guards
  every one of these spawns with `map.getMonsterById(id) != null`. `catalog-lint`
  fails otherwise.
- **Quest-status shift (PRD FR-1.4):** Cosmic `UNDEFINED(-1) NOT_STARTED(0) STARTED(1) COMPLETED(2)`
  → Atlas `UNDEFINED=0 NOT_STARTED=1 STARTED=2 COMPLETED=3`. Every ported
  `getQuestStatus(x) == n` is emitted as `n + 1`.
- **Condition names are the saga names**: `jobId`, `level`, `gender`, `questStatus` —
  never `job` or `quest_status`.
- **Never invent a map name.** Descriptions state the script name, the map id when it
  is derivable from the script name or effect path, and the observable behavior.
  Monster names in the boss documents come from Cosmic's own message strings.

## Verification loop (identical for every task)

Each task ends with this same three-command proof. It is written out per task rather
than referenced, because an implementer sees only their own task.

```bash
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```

---

## Task B1: the seven plain map-entry effects

Seven scripts whose entire Cosmic body is one `ms.mapEffect("maplemap/enter/<id>")`
call with no guard and no condition. The numeric suffix of the effect path equals the
script name's suffix in all seven — verified per script, not assumed.

### Files

Create, under each of the 11 roots at `map-actions/onUserEnter/`:

- `map-go1000000.json` — **new file**
- `map-go1010000.json` — **new file**
- `map-go1010200.json` — **new file**
- `map-go1010300.json` — **new file**
- `map-go30000.json` — **new file**
- `map-go40000.json` — **new file**
- `map-go50000.json` — **new file**

77 files total (7 × 11). Author under `deploy/seed/gms/83_1/map-actions/onUserEnter/`.

Patterns to copy: `deploy/seed/gms/83_1/map-actions/onUserEnter/map-go1010100.json` —
an already-seeded script of exactly this shape. Read it first; it is the template.

- [ ] **Step 1: Read the existing single-effect exemplar**

Run: `cat deploy/seed/gms/83_1/map-actions/onUserEnter/map-go1010100.json`

Match its rule id convention, its `description` phrasing and its key ordering.

- [ ] **Step 2: Author the seven documents under `gms/83_1`**

Each document is this exact shape, with `<name>` and `<mapId>` substituted:

```json
{
  "data": {
    "attributes": {
      "description": "Map entry effect for map <mapId> - plays maplemap/enter/<mapId>",
      "rules": [
        {
          "conditions": [],
          "id": "enter_effect",
          "operations": [
            {
              "params": {
                "path": "maplemap/enter/<mapId>"
              },
              "type": "field_effect"
            }
          ]
        }
      ],
      "scriptName": "<name>"
    },
    "id": "<name>",
    "type": "map-action"
  }
}
```

The seven `(name, mapId)` pairs — the effect path is the literal string from Cosmic,
which for all seven is `maplemap/enter/` plus the script name minus its `go` prefix:

| scriptName | mapId / effect suffix | effect path |
|---|---|---|
| `go1000000` | `1000000` | `maplemap/enter/1000000` |
| `go1010000` | `1010000` | `maplemap/enter/1010000` |
| `go1010200` | `1010200` | `maplemap/enter/1010200` |
| `go1010300` | `1010300` | `maplemap/enter/1010300` |
| `go30000` | `30000` | `maplemap/enter/30000` |
| `go40000` | `40000` | `maplemap/enter/40000` |
| `go50000` | `50000` | `maplemap/enter/50000` |

`conditions` is the empty array: Cosmic's body is unconditional.

- [ ] **Step 3: Replicate to the other ten roots**

```bash
for f in go1000000 go1010000 go1010200 go1010300 go30000 go40000 go50000; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
```

- [ ] **Step 4: Verify**

```bash
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: `--check` exits 0; catalog-lint exits 0; the file count is `77`.

- [ ] **Step 5: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): seven plain map-entry effect map-actions"
```

---

## Task B2: the UI-only and mixed UI/effect scripts

Four scripts. `go10000` and `go20000` call `unlockUI()` **first**, then `mapEffect` —
operation order matters and is verified from the Cosmic source, not assumed.
`evanleaveD`'s entire body is one `ms.unlockUI()`. `Resi_tutor20`'s entire body is one
`ms.mapEffect("resistance/tutorialGuide")` — note the path has no `maplemap/enter/`
prefix.

### Files

Create under each of the 11 roots at `map-actions/onUserEnter/`:

- `map-go10000.json` — **new file**
- `map-go20000.json` — **new file**
- `map-evanleaveD.json` — **new file**
- `map-Resi_tutor20.json` — **new file**

44 files total. Author under `deploy/seed/gms/83_1/map-actions/onUserEnter/`.

Patterns to copy: `deploy/seed/gms/83_1/map-actions/onUserEnter/map-go1010100.json`
for the single-rule envelope. Note that filenames preserve the script name's exact
case — `map-Resi_tutor20.json` and `map-evanleaveD.json`, not lowercased. The
`catalog-lint` filename pattern is `^map-(.+)\.json$`
(`tools/catalog-lint/subdomains.go:18`), so the captured id must equal `data.id`
character for character.

- [ ] **Step 1: Decide the `params` shape for a no-param operation, once**

`lock_ui` and `unlock_ui` take no parameters. Read the generated schema and determine
whether `params` is required for them:

Run: `sed -n '92,120p' services/atlas-map-actions/docs/map_script_schema.json`

`params` is listed under `properties` but `required` is `["type"]` only, so the key
may be omitted entirely. Omit it — an empty `{}` carries no information. Apply this
choice consistently in every task of this plan that emits `lock_ui`/`unlock_ui`
(B2, B3).

- [ ] **Step 2: Author `map-go10000.json`**

```json
{
  "data": {
    "attributes": {
      "description": "Map entry for map 10000 - unlocks the UI, then plays maplemap/enter/10000",
      "rules": [
        {
          "conditions": [],
          "id": "enter_effect",
          "operations": [
            {
              "type": "unlock_ui"
            },
            {
              "params": {
                "path": "maplemap/enter/10000"
              },
              "type": "field_effect"
            }
          ]
        }
      ],
      "scriptName": "go10000"
    },
    "id": "go10000",
    "type": "map-action"
  }
}
```

- [ ] **Step 3: Author `map-go20000.json`**

Identical to Step 2 with `10000` → `20000` in the description, the effect path, the
`scriptName` and the `id`. Operation order is the same: `unlock_ui` then
`field_effect`.

- [ ] **Step 4: Author `map-evanleaveD.json`**

```json
{
  "data": {
    "attributes": {
      "description": "Evan dragon-tutorial exit map - unlocks the UI",
      "rules": [
        {
          "conditions": [],
          "id": "unlock",
          "operations": [
            {
              "type": "unlock_ui"
            }
          ]
        }
      ],
      "scriptName": "evanleaveD"
    },
    "id": "evanleaveD",
    "type": "map-action"
  }
}
```

- [ ] **Step 5: Author `map-Resi_tutor20.json`**

```json
{
  "data": {
    "attributes": {
      "description": "Resistance tutorial - plays the resistance/tutorialGuide effect",
      "rules": [
        {
          "conditions": [],
          "id": "guide_effect",
          "operations": [
            {
              "params": {
                "path": "resistance/tutorialGuide"
              },
              "type": "field_effect"
            }
          ]
        }
      ],
      "scriptName": "Resi_tutor20"
    },
    "id": "Resi_tutor20",
    "type": "map-action"
  }
}
```

The path is `resistance/tutorialGuide` exactly — no `maplemap/enter/` prefix, no
leading slash.

- [ ] **Step 6: Replicate to the other ten roots**

```bash
for f in go10000 go20000 evanleaveD Resi_tutor20; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
```

- [ ] **Step 7: Verify**

```bash
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `44`.

- [ ] **Step 8: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): UI-lock and tutorial-effect map-actions"
```

---

## Task B3: the four Evan dragon cutscenes

Each Cosmic body is exactly two calls: `ms.lockUI()` then
`ms.showIntro("Effect/Direction4.img/<segment>/Scene" + ms.getPlayer().getGender())`.
The gender is **string-concatenated onto the path**, not compared against a literal —
so the Atlas model is two rules, one per gender value, each with its own literal
`Scene0`/`Scene1` path. That is the same shape as the already-seeded `goArcher`.

Two facts the PRD's §4.4 table omits and which must not be dropped:

1. **`lockUI()` precedes `showIntro` in all four**, and the seeded `goArcher` has no
   `lock_ui` at all — so these four are *not* a straight copy of `goArcher`'s body.
   The `lock_ui` operation goes in both rules, before the `show_intro`.
2. **None of the four ever calls `unlockUI()`.** That is Cosmic's actual behavior; the
   UI is unlocked later by whatever follows the cutscene. Do not add an `unlock_ui`
   the source does not have.

### Files

Create under each of the 11 roots at `map-actions/onUserEnter/`:

- `map-crash_Dragon.json` — **new file**
- `map-getDragonEgg.json` — **new file**
- `map-meetWithDragon.json` — **new file**
- `map-PromiseDragon.json` — **new file**

44 files total. Author under `deploy/seed/gms/83_1/map-actions/onUserEnter/`.

Patterns to copy: `deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json` —
the two-rule gendered `show_intro` shape, verbatim, including the `intro_male` /
`intro_female` rule ids. The only structural difference is the added `lock_ui`
operation.

- [ ] **Step 1: Author the four documents under `gms/83_1`**

Each is this shape with `<name>` and `<segment>` substituted:

```json
{
  "data": {
    "attributes": {
      "description": "Evan dragon cutscene <segment> - locks the UI and plays the gendered Direction4 scene",
      "rules": [
        {
          "conditions": [
            {
              "operator": "=",
              "type": "gender",
              "value": "0"
            }
          ],
          "id": "intro_male",
          "operations": [
            {
              "type": "lock_ui"
            },
            {
              "params": {
                "path": "Effect/Direction4.img/<segment>/Scene0"
              },
              "type": "show_intro"
            }
          ]
        },
        {
          "conditions": [
            {
              "operator": "=",
              "type": "gender",
              "value": "1"
            }
          ],
          "id": "intro_female",
          "operations": [
            {
              "type": "lock_ui"
            },
            {
              "params": {
                "path": "Effect/Direction4.img/<segment>/Scene1"
              },
              "type": "show_intro"
            }
          ]
        }
      ],
      "scriptName": "<name>"
    },
    "id": "<name>",
    "type": "map-action"
  }
}
```

The four `(scriptName, segment)` pairs. Note the segment is **not** always the script
name — `crash_Dragon`'s path segment is `crash`:

| scriptName | effect path segment | Scene0 path | Scene1 path |
|---|---|---|---|
| `crash_Dragon` | `crash` | `Effect/Direction4.img/crash/Scene0` | `Effect/Direction4.img/crash/Scene1` |
| `getDragonEgg` | `getDragonEgg` | `Effect/Direction4.img/getDragonEgg/Scene0` | `Effect/Direction4.img/getDragonEgg/Scene1` |
| `meetWithDragon` | `meetWithDragon` | `Effect/Direction4.img/meetWithDragon/Scene0` | `Effect/Direction4.img/meetWithDragon/Scene1` |
| `PromiseDragon` | `PromiseDragon` | `Effect/Direction4.img/PromiseDragon/Scene0` | `Effect/Direction4.img/PromiseDragon/Scene1` |

- [ ] **Step 2: Replicate to the other ten roots**

```bash
for f in crash_Dragon getDragonEgg meetWithDragon PromiseDragon; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
```

- [ ] **Step 3: Verify**

```bash
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `44`.

- [ ] **Step 4: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): four Evan dragon cutscene map-actions"
```

---

## Task B4: `startEreb`

Cosmic body: `if (ms.getJobId() == 1000 && ms.getLevel() >= 10) { ms.unlockUI(); }`.
One rule, two AND-ed conditions. This is the document that exercises PRD FR-1.1: the
condition type is `jobId`, not `job` — `job` is a name the aggregator rejects and the
generated schema no longer contains.

### Files

- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-startEreb.json` — **new file**, plus the same file under the other ten roots (11 total)

Patterns to copy: `deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json` for
the envelope; its `gender` condition shows the condition object's key order.

- [ ] **Step 1: Author the document**

```json
{
  "data": {
    "attributes": {
      "description": "Ereve arrival - unlocks the UI for a Noblesse (jobId 1000) at level 10 or above",
      "rules": [
        {
          "conditions": [
            {
              "operator": "=",
              "type": "jobId",
              "value": "1000"
            },
            {
              "operator": ">=",
              "type": "level",
              "value": "10"
            }
          ],
          "id": "unlock_for_noblesse",
          "operations": [
            {
              "type": "unlock_ui"
            }
          ]
        }
      ],
      "scriptName": "startEreb"
    },
    "id": "startEreb",
    "type": "map-action"
  }
}
```

Conditions in a rule are AND-ed (`map_script_schema.json:43`: "Conditions that must
ALL be true"), which is exactly Cosmic's `&&`. Both are forwarded to the aggregator —
neither is `map_id` — so this document also proves Plan A Task A3's `worldId`/
`channelId` population reaches a real seed.

- [ ] **Step 2: Replicate to the other ten roots**

```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  cp deploy/seed/gms/83_1/map-actions/onUserEnter/map-startEreb.json \
     "deploy/seed/$r/map-actions/onUserEnter/map-startEreb.json"
done
```

- [ ] **Step 3: Verify**

```bash
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
python3 - <<'EOF'
import glob
ps = sorted(glob.glob('deploy/seed/*/*/map-actions/onUserEnter/map-startEreb.json'))
print('files:', len(ps))
print('counts:', sorted({open(p).read().count('"jobId"') for p in ps}))
EOF
```
Expected: both exit 0; the file count is `11`; the script reports `files: 11` and
`counts: [1]`.

- [ ] **Step 4: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): startEreb map-action with the jobId condition name"
```

---

## Task B5: the six Demon boss spawns

Each Cosmic body is: guard on `map.getMonsterById(mobId) != null` → return; else
`map.spawnMonsterOnGroundBelow(LifeFactory.getMonster(mobId), new Point(x, y))` and
`player.message(mobName + " has appeared!")`.

The guard is exactly what `spawnIfAbsent` expresses (PRD FR-2.2). The message strings
are quoted literally from the Cosmic sources, including capitalization and the single
trailing `!`.

### Files

Create under each of the 11 roots at `map-actions/onUserEnter/`:

- `map-677000001.json` — **new file**
- `map-677000003.json` — **new file**
- `map-677000005.json` — **new file**
- `map-677000007.json` — **new file**
- `map-677000009.json` — **new file**
- `map-677000012.json` — **new file**

66 files total. Author under `deploy/seed/gms/83_1/map-actions/onUserEnter/`.

Patterns to copy: `deploy/seed/gms/83_1/map-actions/onUserEnter/map-108010301.json`
for the `spawn_monster` operation's param shape (`monsterId`, `spawnIfAbsent`, `x`,
`y`). Read it after Plan A Task A9 lands, so the `spawnIfAbsent` key is present in the
exemplar.

- [ ] **Step 1: Author the six documents under `gms/83_1`**

Each is this shape with the five values from the table substituted:

```json
{
  "data": {
    "attributes": {
      "description": "Boss map <scriptName> - spawns <bossName> (<monsterId>) at (<x>, <y>) and announces it",
      "rules": [
        {
          "conditions": [],
          "id": "spawn_boss",
          "operations": [
            {
              "params": {
                "monsterId": "<monsterId>",
                "spawnIfAbsent": "true",
                "x": "<x>",
                "y": "<y>"
              },
              "type": "spawn_monster"
            },
            {
              "params": {
                "message": "<bossName> has appeared!"
              },
              "type": "drop_message"
            }
          ]
        }
      ],
      "scriptName": "<scriptName>"
    },
    "id": "<scriptName>",
    "type": "map-action"
  }
}
```

| scriptName | monsterId | x | y | bossName | exact message |
|---|---|---|---|---|---|
| `677000001` | `9400612` | `461` | `61` | Marbas | `Marbas has appeared!` |
| `677000003` | `9400610` | `467` | `0` | Amdusias | `Amdusias has appeared!` |
| `677000005` | `9400609` | `201` | `80` | Andras | `Andras has appeared!` |
| `677000007` | `9400611` | `171` | `50` | Crocell | `Crocell has appeared!` |
| `677000009` | `9400613` | `251` | `-841` | Valefor | `Valefor has appeared!` |
| `677000012` | `9400633` | `842` | `0` | Astaroth | `Astaroth has appeared!` |

`677000009`'s `y` is negative (`-841`) and `677000012`'s monster id (`9400633`) breaks
the otherwise-sequential `9400609`–`9400613` band. Both are the literal Cosmic values;
do not "correct" either.

`drop_message` omits `messageType`; the schema documents the default as `PINK_TEXT`
(`map_script_schema.json:203-206`). Cosmic uses `player.message(...)`, which is that
same pink notice, so the default is correct and the param stays absent.

- [ ] **Step 2: Replicate to the other ten roots**

```bash
for f in 677000001 677000003 677000005 677000007 677000009 677000012; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
```

- [ ] **Step 3: Verify**

```bash
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
python3 - <<'EOF'
import glob
bad = [p for p in glob.glob('deploy/seed/*/*/map-actions/onUserEnter/map-6770000*.json')
       if 'spawnIfAbsent' not in open(p).read()]
print('files:', len(glob.glob('deploy/seed/*/*/map-actions/onUserEnter/map-6770000*.json')))
print('missing spawnIfAbsent:', bad)
EOF
```
Expected: both exit 0; the file count is `66`; the script reports `files: 66` and
`missing spawnIfAbsent: []`.

- [ ] **Step 4: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): six Demon boss spawn map-actions with spawn guards"
```

---

## Task B6: `100000006` — the quest-gated spawn

Cosmic body: `if (ms.getQuestStatus(2175) == 1)` → guard on
`map.getMonsterById(9300156) != null` → else spawn `9300156` at
`new Point(-1027, 216)`. There is **no** `player.message(...)` call, unlike the six
`677*` scripts.

This is the document that exercises PRD FR-1.4. Cosmic's `== 1` is `STARTED`; Atlas'
`STARTED` is `2`. The emitted value is therefore `"2"`, not `"1"`.

### Files

- `deploy/seed/gms/83_1/map-actions/onUserEnter/map-100000006.json` — **new file**, plus the same file under the other ten roots (11 total)

Patterns to copy: the boss documents from Task B5 for the `spawn_monster` params, and
`services/atlas-map-actions/docs/map_script_schema.json`'s condition definition
(`referenceId` at lines 86-89) for where the quest id goes.

- [ ] **Step 1: Author the document**

```json
{
  "data": {
    "attributes": {
      "description": "Map 100000006 - spawns monster 9300156 at (-1027, 216) while quest 2175 is started",
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
          "id": "spawn_quest_monster",
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
      "scriptName": "100000006"
    },
    "id": "100000006",
    "type": "map-action"
  }
}
```

The `value` is `"2"`. Write the FR-1.4 derivation into the commit body so a reviewer
can check the arithmetic without re-reading Cosmic: Cosmic
`getQuestStatus(2175) == 1` is `STARTED`; Atlas' `STARTED` is `2`; `1 + 1 = 2`.

- [ ] **Step 2: Replicate to the other ten roots**

```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  cp deploy/seed/gms/83_1/map-actions/onUserEnter/map-100000006.json \
     "deploy/seed/$r/map-actions/onUserEnter/map-100000006.json"
done
```

- [ ] **Step 3: Verify**

```bash
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
```
Expected: both exit 0; the file count is `11`.

- [ ] **Step 4: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): 100000006 quest-gated spawn with the +1 quest-status shift

Cosmic getQuestStatus(2175) == 1 is STARTED; Atlas STARTED is 2 (FR-1.4)."
```

---

## Task B7: the four job-instructor duplicates

`108010101.js`, `108010201.js`, `108010401.js` and `108010501.js` are **byte-identical**
to `108010301.js` — confirmed by `diff`, all four produce zero output against it. Each
runs the same five-branch `ms.getMapId()` chain and spawns the matching job's test
monster at `(188, 20)` through a local `spawnMob` helper that guards with
`map.getMonsterById(id) != null`.

Each map's WZ `onUserEnter` names its own script, so four separate seed documents are
required even though the bodies are identical. The body is exactly the already-seeded
`map-108010301.json` (as retrofitted by Plan A Task A9); only `scriptName`, `id` and
`description` differ.

### Files

Create under each of the 11 roots at `map-actions/onUserEnter/`:

- `map-108010101.json` — **new file**
- `map-108010201.json` — **new file**
- `map-108010401.json` — **new file**
- `map-108010501.json` — **new file**

44 files total.

Patterns to copy: `deploy/seed/gms/83_1/map-actions/onUserEnter/map-108010301.json` —
copy it verbatim and change three fields. Read it **after** Plan A Task A9 has added
`spawnIfAbsent` to all five of its operations.

- [ ] **Step 1: Confirm the source document carries the spawn guards**

Run:
```bash
python3 -c "print(open('deploy/seed/gms/83_1/map-actions/onUserEnter/map-108010301.json').read().count('spawnIfAbsent'))"
```
Expected: `5`. If it reports `0`, Plan A Task A9 has not landed — stop and run it
first, otherwise these four documents will fail catalog-lint.

- [ ] **Step 2: Create the four documents from the exemplar**

```bash
for n in 108010101 108010201 108010401 108010501; do
  cp deploy/seed/gms/83_1/map-actions/onUserEnter/map-108010301.json \
     "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$n.json"
done
```

Then in each new file change exactly three values, leaving the five rules untouched:

| file | `attributes.scriptName` | `data.id` | `attributes.description` |
|---|---|---|---|
| `map-108010101.json` | `108010101` | `108010101` | `Job advancement test map 108010101 (Archer) - spawns the job-specific test monster based on map ID` |
| `map-108010201.json` | `108010201` | `108010201` | `Job advancement test map 108010201 (Mage) - spawns the job-specific test monster based on map ID` |
| `map-108010401.json` | `108010401` | `108010401` | `Job advancement test map 108010401 (Thief) - spawns the job-specific test monster based on map ID` |
| `map-108010501.json` | `108010501` | `108010501` | `Job advancement test map 108010501 (Pirate) - spawns the job-specific test monster based on map ID` |

The job label in each description is Cosmic's own inline comment word from the
five-branch chain — `// Archer`, `// Mage`, `// Thief`, `// Pirate`, `// Warrior`.
Use Cosmic's word verbatim (`Mage`, not `Magician`) so each document is traceable to
its source.

Do **not** narrow any document's rules to its own map id. All five documents keep all
five `map_id` branches, because Cosmic's body does and because the seeded
`108010301` already does — the rules are selected at runtime by the field's actual
map id.

- [ ] **Step 3: Replicate to the other ten roots**

```bash
for f in 108010101 108010201 108010401 108010501; do
  for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
    cp "deploy/seed/gms/83_1/map-actions/onUserEnter/map-$f.json" \
       "deploy/seed/$r/map-actions/onUserEnter/map-$f.json"
  done
done
```

- [ ] **Step 4: Verify**

```bash
./tools/gen-map-action-schema.sh --check
(cd tools/catalog-lint && GOWORK=off go run . ../../deploy/seed)
git status --short deploy/seed/ | wc -l
for n in 108010101 108010201 108010401 108010501; do
  diff <(python3 -c "import json;print(json.dumps(json.load(open('deploy/seed/gms/83_1/map-actions/onUserEnter/map-$n.json'))['data']['attributes']['rules'],sort_keys=True,indent=2))") \
       <(python3 -c "import json;print(json.dumps(json.load(open('deploy/seed/gms/83_1/map-actions/onUserEnter/map-108010301.json'))['data']['attributes']['rules'],sort_keys=True,indent=2))") \
    && echo "$n rules match"
done
```
Expected: both exit 0; the file count is `44`; all four `rules match` lines print —
proving the bodies are identical to the exemplar and only the three fields differ.

- [ ] **Step 5: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): four job-instructor test-map duplicates of 108010301"
```

---

## Plan B completion gate

- [ ] **Confirm all 27 documents landed in all 11 roots**

Run:
```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/83_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  printf '%s %s\n' "$r" "$(ls deploy/seed/$r/map-actions/onUserEnter/ | wc -l)"
done
```
Expected: every root reports `35` — the 8 previously-seeded `onUserEnter` documents
plus this plan's 27.

- [ ] **Confirm byte-identity across the roots**

Run:
```bash
for r in gms/12_1 gms/48_1 gms/61_1 gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  diff -r deploy/seed/gms/83_1/map-actions deploy/seed/$r/map-actions || echo "DRIFT in $r"
done
```
Expected: no output at all.

- [ ] **Run the full verification gate**

Run: `./tools/verify.sh`
Expected: exit 0. `--quick` and `--no-docker` do **not** satisfy this.

- [ ] **Requirement coverage**

| PRD requirement | Task |
|---|---|
| §4.4 `field_effect` only (7) | B1 |
| §4.4 `unlock_ui` + `field_effect` (2) | B2 |
| §4.4 `unlock_ui` only (`evanleaveD`) | B2 |
| §4.4 `Resi_tutor20` | B2 |
| §4.4 gendered cutscenes (4) | B3 |
| §4.4 `startEreb` (FR-1.1 `jobId`) | B4 |
| §4.4 boss spawn + message (6) | B5 |
| §4.4 `100000006` (FR-1.4 shift) | B6 |
| §4.4 job-instructor duplicates (4) | B7 |
| FR-2.2 every spawn guarded | B5, B6, B7 (enforced by catalog-lint) |
| FR-6.1 all 11 roots, correct hook and filename | every task's replication step |
| FR-6.2 human-readable descriptions | every task |
