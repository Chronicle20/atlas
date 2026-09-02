# Tier-1 Reactor Script Conversion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Repair the reactor conversion contract (skill + JSON schema), then generate all 159 Tier-1 reactor seed files across eleven tenant directories from a committed, fail-closed Go generator, gated by a new schema-validating lint.

**Architecture:** Two new single-purpose Go modules under `tools/` — `reactor-seed-gen` (reads `tier1-inventory.md`, matches a whitelisted JS grammar, emits JSON:API seed envelopes to eleven directories, aborts on any unrecognized statement) and `reactor-seed-lint` (validates every seed under `deploy/seed/*/reactor-actions/reactors/` against `reactor_script_schema.json` plus three corpus rules the schema cannot express). A shell wrapper wires the lint into `tools/verify.sh` and runs both modules' Go tests, because `verify.sh`'s Go sweep walks only `services/` and `libs/`.

**Tech Stack:** Go 1.27.0, `github.com/santhosh-tekuri/jsonschema/v6` (module cache already holds v6.0.3), bash + shellcheck for the wrapper, JSON Schema draft-07.

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md))

## Global Constraints

- **Seed file byte conventions, verified against `deploy/seed/gms/83_1/reactor-actions/reactors/reactor-2001.json`:** `json.MarshalIndent(v, "", "  ")` output plus a single trailing `\n`, LF line endings, alphabetically ordered keys at every level (Go map marshalling produces this for free).
- **Envelope shape:** `{"data":{"attributes":{...},"id":"<reactorId>","type":"reactor-action"}}`. `data.id` == `attributes.reactorId` == the `reactor-<id>.json` filename stem.
- **Attribute keys:** `actRules`, `description`, `hitRules`, `reactorId` (alphabetical). `touchRules` is NOT emitted by the generator.
- **Rule keys:** `conditions`, `id`, `operations`. **Condition keys:** `operator`, `step` (omitted when empty), `type`, `value`. **Operation keys:** `params` (omitted entirely when there are none), `type`. All param values are JSON **strings**.
- **The eleven tenant seed directories, in this order:** `deploy/seed/gms/12_1`, `gms/48_1`, `gms/61_1`, `gms/72_1`, `gms/79_1`, `gms/83_1`, `gms/84_1`, `gms/87_1`, `gms/92_1`, `gms/95_1`, `jms/185_1` — each with the suffix `/reactor-actions/reactors/`. All eleven copies of a reactor MUST be byte-identical.
- **Cosmic `dropItems` signature (`ReactorActionManager.java:142`):** `(meso, mesoChance, minMeso, maxMeso, minItems)` → Atlas params `meso`, `mesoChance`, `mesoMin`, `mesoMax`, `minItems`. The names `mesoRange` and `item` do not exist and MUST NOT appear anywhere.
- **`dropType` is never written into a seed file.** `executeSprayItems` injects it at runtime (`services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:247-256`).
- **Never write a literal home/absolute path into a committed file** (CLAUDE.md). The Cosmic reference checkout is referred to as `<Cosmic>/scripts/reactor/<id>.js`.
- **Preserve existing line endings**; never normalize CRLF→LF as a side effect.
- **`tools/` Go modules are excluded from `verify.sh`'s `go build`/`go vet`/`go test` sweep** — `all_modules()` walks only `$ROOT/services` and `$ROOT/libs` (`tools/verify.sh:392-398`). Go tests for the two new tools therefore run from `tools/reactor-seed-lint_test.sh`, which `verify.sh` picks up via `changed_tool_suites()` (`tools/verify.sh:225-243`).
- **Every new `tools/*.sh` must pass `tools/shell-guard.sh --require-shellcheck`** — `bash -n`/`sh -n` parse plus `shellcheck` at severity `error`.

---

## Task 1: Repair the `/convert-reactor` skill

### Files

- `.claude/commands/convert-reactor.md` — rewrite the operation table, condition table, `eim.*` guidance, output contract, and the three worked examples
- `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` — read-only; the authoritative operation list (`ExecuteOperation`, lines 45-84)
- `services/atlas-reactor-actions/atlas.com/reactor/script/evaluator.go` — read-only; the authoritative condition list
- `deploy/seed/gms/83_1/reactor-actions/reactors/reactor-9108000.json` — read-only; a real multi-rule seed envelope to model the examples on

There is no module root; this task edits Markdown only.

**Currently wrong (`.claude/commands/convert-reactor.md`), verified by reading the file:**

| Line(s) | Defect |
|---|---|
| 29 | Condition table lists only `reactor_state`; `pq_custom_data` is missing |
| 34 | `drop_items` documented as `meso, minMeso, maxMeso, mesoRange, item` — shifted by one, and two of the names do not exist |
| 35 | `spawn_monster` missing `x`/`y` |
| 36 | `spray_items` documented as taking no params |
| 39-40 | Only 7 of 11 operations listed — `update_pq_state`, `hit_reactor`, `broadcast_pq_message`, `stage_clear_attempt` missing |
| 42-52 | "NOT YET SUPPORTED (Skip These Scripts)" instructs skipping every `eim.*` script |
| 113 | Says the inverted guard produces `operator: "!="` — it must be `"="` (line 295 of the same file correctly says `= 0`; the two contradict) |
| 152-158 | Step 3 restates the skip instruction |
| 172-173 | Output `reactor_{reactorId}.json` into `services/atlas-reactor-actions/scripts/reactors/` — that directory does not exist and the filename separator is wrong |
| 190-215, 226-243, 265-293 | Three worked examples emit the bare script object, not the JSON:API envelope; the first emits `minMeso`/`mesoRange`/`item` |
| 297-316 | "Unsupported Script (Skip)" example |

### Steps

- [ ] **Step 1: Replace the condition table (line 26-29)**

```markdown
### Condition Types
| Type | When to Use | Required Fields |
|------|-------------|-----------------|
| `reactor_state` | `rm.getReactor().getState()` checks | `type`, `operator`, `value` |
| `pq_custom_data` | `eim.getIntProperty(k)` / `eim.getProperty(k)` comparisons | `type`, `operator`, `value`, `step` (the custom-data key name) |

Operators for both types: `=`, `!=`, `>`, `<`, `>=`, `<=`.
```

- [ ] **Step 2: Replace the operation table (line 31-40) with all 11 operations**

```markdown
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
```

- [ ] **Step 3: Replace the "NOT YET SUPPORTED" section (lines 42-52) with the `eim.*` mapping**

```markdown
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
```

- [ ] **Step 4: Fix the guard-inversion statement at line 113**

Change `→ hitRule with \`reactor_state\` condition (operator: \`!=\`, value: \`0\`), then operation` to:

```markdown
→ hitRule with a `reactor_state` condition **inverted to the positive form** (`operator: "="`, `value: "0"`), then the operation. The source's `!== 0 → return` means "act only when the state IS 0".
```

- [ ] **Step 5: Fix the output contract (Task section, steps 3 and 9-10)**

Delete step 3's skip instruction entirely. Replace steps 9-10 with:

```markdown
9. Determine the output filename: `reactor-<reactorId>.json` (hyphen, not underscore).
10. Write the file, byte-identical, into all eleven tenant seed directories:
    `deploy/seed/gms/{12_1,48_1,61_1,72_1,79_1,83_1,84_1,87_1,92_1,95_1}/reactor-actions/reactors/`
    and `deploy/seed/jms/185_1/reactor-actions/reactors/`.
    The file is a JSON:API envelope:
    `{"data":{"attributes":{...},"id":"<reactorId>","type":"reactor-action"}}`
    2-space indented, alphabetically keyed, LF, one trailing newline.
```

- [ ] **Step 6: Rewrite the three worked examples as envelopes**

The `2119000` example becomes (note `hitRules` carries the guarded rule and `actRules` is empty):

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

The `dropItems(true, 2, 8, 15, 1)` example emits `"meso": "true", "mesoChance": "2", "mesoMin": "8", "mesoMax": "15", "minItems": "1"`. The bare `rm.dropItems()` example emits an operation object with only `"type": "drop_items"` and no `params` key.

- [ ] **Step 7: Delete the "Example: Unsupported Script (Skip)" section (lines 297-316)**

Replace it with a worked `eim.*` example using `2512001`:

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

- [ ] **Step 8: Verify no forbidden strings remain**

Run: `grep -n 'mesoRange\|"item"\|scripts/reactors\|reactor_{' .claude/commands/convert-reactor.md`
Expected: no output.

Run: `grep -c 'update_pq_state\|hit_reactor\|broadcast_pq_message\|stage_clear_attempt\|pq_custom_data' .claude/commands/convert-reactor.md`
Expected: a non-zero count.

- [ ] **Step 9: Commit**

```bash
git add .claude/commands/convert-reactor.md
git commit -m "docs(convert-reactor): correct operation set, condition set, meso mapping, and output contract"
```

---

## Task 2: Repair `reactor_script_schema.json`

### Files

- `services/atlas-reactor-actions/docs/reactor_script_schema.json` — add `touchRules`; replace the `drop_items` param branch; add a `spray_items` branch; add `additionalProperties: false` to both
- `services/atlas-reactor-actions/atlas.com/reactor/script/entity.go` — lines 31-59; read-only; the authoritative `jsonReactorScript`/`jsonRule`/`jsonCondition`/`jsonOperation` shape
- `services/atlas-reactor-actions/atlas.com/reactor/script/processor.go` — lines 185-187; read-only; the touchRules→hitRules fallback

No module root; this task edits JSON only.

### Steps

- [ ] **Step 1: Add `touchRules` to the top-level `properties` (FR-8)**

Insert after the `actRules` property block, keeping `required` as `["reactorId", "hitRules", "actRules"]` — `touchRules` is optional and falls back to `hitRules` when absent:

```json
"touchRules": {
  "type": "array",
  "description": "Rules evaluated when reactor is touched (player collision). Optional — when a script declares none, the engine falls back to hitRules (processor.go:185-187).",
  "items": {
    "$ref": "#/definitions/rule"
  }
}
```

- [ ] **Step 2: Replace the `drop_items` `allOf` branch (FR-9)**

Replace the entire `drop_items` `if`/`then` block. Note `additionalProperties: false`, which is what turns the legacy-key ban into an enforced rule rather than a documented one — `"mesoRange": "15"` becomes a validation failure:

```json
{
  "if": {
    "properties": { "type": { "const": "drop_items" } }
  },
  "then": {
    "properties": {
      "params": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "meso": {
            "type": "string",
            "description": "Whether meso may drop (\"true\"/\"false\")"
          },
          "mesoChance": {
            "type": "string",
            "description": "1-in-N chance a drop slot becomes meso (default \"1\")"
          },
          "mesoMin": {
            "type": "string",
            "description": "Minimum meso multiplier (default \"1\")"
          },
          "mesoMax": {
            "type": "string",
            "description": "Maximum meso multiplier (default \"1\")"
          },
          "minItems": {
            "type": "string",
            "description": "Minimum guaranteed drops, padded with meso (default \"0\")"
          }
        }
      }
    }
  }
}
```

- [ ] **Step 3: Add the `spray_items` `allOf` branch (FR-4, FR-9)**

Insert immediately after the `drop_items` branch. Identical parameter set; `dropType` is deliberately absent and `additionalProperties: false` therefore rejects it:

```json
{
  "if": {
    "properties": { "type": { "const": "spray_items" } }
  },
  "then": {
    "properties": {
      "params": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "meso": {
            "type": "string",
            "description": "Whether meso may drop (\"true\"/\"false\")"
          },
          "mesoChance": {
            "type": "string",
            "description": "1-in-N chance a drop slot becomes meso (default \"1\")"
          },
          "mesoMin": {
            "type": "string",
            "description": "Minimum meso multiplier (default \"1\")"
          },
          "mesoMax": {
            "type": "string",
            "description": "Maximum meso multiplier (default \"1\")"
          },
          "minItems": {
            "type": "string",
            "description": "Minimum guaranteed drops, padded with meso (default \"0\")"
          }
        }
      }
    }
  }
}
```

- [ ] **Step 4: Verify the schema still parses and the forbidden names are gone**

Run: `python3 -c "import json;d=json.load(open('services/atlas-reactor-actions/docs/reactor_script_schema.json'));print(sorted(d['properties']))"`
Expected: `['actRules', 'description', 'hitRules', 'reactorId', 'touchRules']`

Run: `grep -n 'mesoRange\|"item"' services/atlas-reactor-actions/docs/reactor_script_schema.json`
Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reactor-actions/docs/reactor_script_schema.json
git commit -m "docs(reactor-actions): document touchRules and correct drop_items/spray_items params in the schema"
```

---

## Task 3: Seed `touchRules` through the seeder's Build path

### Files

- `services/atlas-reactor-actions/atlas.com/reactor/script/subdomain.go` — lines 43-73; add the missing `TouchRules` loop to `ReactorSubdomain.Build`
- `services/atlas-reactor-actions/atlas.com/reactor/script/subdomain_test.go` — **new file**
- `services/atlas-reactor-actions/atlas.com/reactor/script/entity.go` — lines 86-94; read-only; `Make()` already does exactly this loop, copy its shape
- `services/atlas-reactor-actions/atlas.com/reactor/script/entity_test.go` — lines 282-347; read-only; `TestMakeTouchRules` is the table shape to copy

Module root for `go build`/`go test`: `services/atlas-reactor-actions/atlas.com/reactor`

**Why this is here:** Task 2 makes the schema advertise `touchRules` as a seedable key, but `ReactorSubdomain.Build` iterates only `attrs.HitRules` and `attrs.ActRules` — a seed file's `touchRules` is silently dropped, while the REST and entity paths (`entity.go:88-93`, `rest.go:182-187`) handle it. Documenting a key the seeder ignores would be worse than not documenting it. This is a six-line fix on a seam Task 2 exposes.

Patterns to copy: `services/atlas-reactor-actions/atlas.com/reactor/script/entity.go:86-94` (the identical loop), `services/atlas-reactor-actions/atlas.com/reactor/script/entity_test.go:282-347` (table shape).

### Steps

- [ ] **Step 1: Write the failing test**

`TestReactorSubdomainBuildTouchRules` in a new `subdomain_test.go`, package `script`. Table-driven; each case decodes a JSON:API attributes payload through `ReactorSubdomain{}.Decode` then `Build`, and asserts the touch-rule count and first id. Build a tenant with the project's tenant builder as `entity_test.go` does; `Build` ignores it (`_ = t`), so any valid `tenant.Model` works.

| case | attributes JSON | want len(TouchRules()) | want TouchRules()[0].Id() |
|---|---|---|---|
| touchRules present | `{"reactorId":"6109013","description":"d","hitRules":[],"actRules":[],"touchRules":[{"id":"t1","conditions":[],"operations":[]}]}` | 1 | `t1` |
| touchRules absent | `{"reactorId":"2001","description":"d","hitRules":[],"actRules":[]}` | 0 | (not checked) |
| touchRules empty array | `{"reactorId":"2001","description":"d","hitRules":[],"actRules":[],"touchRules":[]}` | 0 | (not checked) |

Note `Decode` takes the **attributes** bytes, not the full envelope (`seeder.DecodeAttributes` unwraps nothing here — see `subdomain.go:36-41`); pass the attributes object above directly.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-reactor-actions/atlas.com/reactor && go test ./script/ -run TestReactorSubdomainBuildTouchRules -v`
Expected: FAIL — `len(TouchRules) = 0, want 1` for the first case.

- [ ] **Step 3: Add the loop to `Build`**

Insert immediately after the `attrs.ActRules` loop in `subdomain.go`:

```go
	for _, jr := range attrs.TouchRules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return nil, fmt.Errorf("reactor-actions: convert touch rule %q: %w", jr.Id, err)
		}
		builder.AddTouchRule(rule)
	}
```

- [ ] **Step 4: Run the test and confirm it passes**

Run: `cd services/atlas-reactor-actions/atlas.com/reactor && go test ./script/ -run TestReactorSubdomainBuildTouchRules -v`
Expected: PASS, three subtests.

Run: `cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go test ./script/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reactor-actions/atlas.com/reactor/script/subdomain.go services/atlas-reactor-actions/atlas.com/reactor/script/subdomain_test.go
git commit -m "fix(reactor-actions): seed touchRules through ReactorSubdomain.Build"
```

---

## Task 4: `tools/reactor-seed-lint` — module scaffold, schema conformance, envelope well-formedness

### Files

- `tools/reactor-seed-lint/go.mod` — **new file**
- `tools/reactor-seed-lint/go.sum` — **new file** (generated)
- `tools/reactor-seed-lint/main.go` — **new file**; CLI + walk + the four assertions' driver
- `tools/reactor-seed-lint/schema.go` — **new file**; schema compilation + per-file validation
- `tools/reactor-seed-lint/main_test.go` — **new file**
- `tools/reactor-seed-lint/testdata/` — **new file** tree (fixtures below)
- `go.work` — add `./tools/reactor-seed-lint` to the `use` block, alphabetically between `./tools/packet-audit` and `./tools/seed-splitters`
- `tools/catalog-lint/main.go` — read-only; the walk + error-accumulation shape to copy
- `tools/catalog-lint/main_test.go` — read-only; the build-then-exec test shape to copy
- `tools/catalog-lint/go.mod` — read-only; the module + `replace` shape to copy
- `libs/atlas-seeder/jsonapi.go` — lines 9-80; read-only; `Envelope`, `ParseEnvelope`, `ExtractEntityID`

Module root for `go build`/`go test`: `tools/reactor-seed-lint`

Patterns to copy: `tools/catalog-lint/main.go:25-105` (walk + `errs []string` accumulation + single aggregated error), `tools/catalog-lint/main_test.go:10-21` (`buildLint` helper).

**Assertions 1 and 2 (of four; 3 and 4 land in Task 5):**

1. **Schema conformance** — unwrap `.data.attributes` and validate that object against `services/atlas-reactor-actions/docs/reactor_script_schema.json`. The schema describes the bare script object; no envelope wrapper is added to it.
2. **Envelope well-formedness** — `data.type == "reactor-action"`, and `data.id == attributes.reactorId ==` the filename stem matched by `^reactor-(.+)\.json$`.

### Steps

- [ ] **Step 1: Create the module and pin the dependency**

```bash
mkdir -p tools/reactor-seed-lint/testdata
cd tools/reactor-seed-lint
cat > go.mod <<'EOF'
module github.com/Chronicle20/atlas/tools/reactor-seed-lint

go 1.27.0

require (
	github.com/Chronicle20/atlas/libs/atlas-seeder v0.0.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
)

replace github.com/Chronicle20/atlas/libs/atlas-seeder => ../../libs/atlas-seeder

replace github.com/Chronicle20/atlas/libs/atlas-routine => ../../libs/atlas-routine
EOF
```

Then add `./tools/reactor-seed-lint` to `go.work`'s `use` block and run `GOFLAGS=-mod=mod go mod tidy` from `tools/reactor-seed-lint`.

`v6.0.3` is already in the local module cache (`$GOPATH/pkg/mod/cache/download/github.com/santhosh-tekuri/jsonschema/v6/@v/v6.0.3.zip`), so this resolves offline.

- [ ] **Step 2: Build the testdata fixtures**

Create these under `tools/reactor-seed-lint/testdata/`. Each `good`/`bad` root mimics `deploy/seed`'s `<region>/<version>/reactor-actions/reactors/` layout.

| fixture path | content | expected verdict |
|---|---|---|
| `good/gms/83_1/reactor-actions/reactors/reactor-2001.json` | valid envelope, `drop_items` with `meso/mesoChance/mesoMin/mesoMax/minItems` = `true/2/8/15/1`, description `"Maple Island Box"` | exit 0 |
| `bad/legacy-keys/gms/83_1/reactor-actions/reactors/reactor-2001.json` | same but params are `meso/minMeso/maxMeso/mesoRange/item` | exit non-zero, message names `mesoRange` or `additionalProperties` |
| `bad/type-mismatch/gms/83_1/reactor-actions/reactors/reactor-2001.json` | `data.type` is `"reactor-drop"` | exit non-zero, message names `type` |
| `bad/id-mismatch/gms/83_1/reactor-actions/reactors/reactor-2001.json` | `data.id` is `"2001"` but `attributes.reactorId` is `"2002"` | exit non-zero |
| `bad/missing-required/gms/83_1/reactor-actions/reactors/reactor-2001.json` | `attributes` omits `actRules` | exit non-zero |

- [ ] **Step 3: Write the failing tests**

`main_test.go`, package `main`. Copy `buildLint` verbatim from `tools/catalog-lint/main_test.go:10-21`, renaming the binary to `reactor-seed-lint`. The lint takes two arguments: the seed root and the schema path.

```
TestLint_GoodCorpusExitsZero        → exe testdata/good  ../../services/atlas-reactor-actions/docs/reactor_script_schema.json   → exit 0
TestLint_LegacyKeysExitNonZero      → exe testdata/bad/legacy-keys      <schema> → non-zero, stderr contains "mesoRange" or "additionalProperties"
TestLint_TypeMismatchExitsNonZero   → exe testdata/bad/type-mismatch    <schema> → non-zero
TestLint_IDMismatchExitsNonZero     → exe testdata/bad/id-mismatch      <schema> → non-zero
TestLint_MissingRequiredExitsNonZero→ exe testdata/bad/missing-required <schema> → non-zero
```

For the three that assert on message text, capture `cmd.CombinedOutput()` and `strings.Contains` the expected substring; for the rest assert only the exit status, as `catalog-lint`'s tests do.

- [ ] **Step 4: Run the tests and confirm they fail**

Run: `cd tools/reactor-seed-lint && go test ./... -v`
Expected: FAIL — build error, `main.go` does not exist yet.

- [ ] **Step 5: Implement `schema.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// compileSchema loads the reactor script schema from disk. The schema
// describes the bare script object (attributes), not the JSON:API envelope.
func compileSchema(path string) (*jsonschema.Schema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open schema: %w", err)
	}
	defer func() { _ = f.Close() }()

	doc, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		return nil, fmt.Errorf("parse schema: %w", err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("reactor_script_schema.json", doc); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}
	return c.Compile("reactor_script_schema.json")
}

// validateAttributes validates one seed file's data.attributes object.
func validateAttributes(s *jsonschema.Schema, attrs json.RawMessage) error {
	var v any
	if err := json.Unmarshal(attrs, &v); err != nil {
		return fmt.Errorf("parse attributes: %w", err)
	}
	return s.Validate(v)
}
```

These five signatures were read from the cached module source at plan time and are exact for `v6.0.3`:
`func UnmarshalJSON(r io.Reader) (any, error)` (`loader.go:255`), `func NewCompiler() *Compiler` (`compiler.go:21`), `func (c *Compiler) AddResource(url string, doc any) error` (`compiler.go:121`), `func (c *Compiler) Compile(loc string) (*Schema, error)` (`compiler.go:178`), `func (sch *Schema) Validate(v any) error` (`validator.go:15`). The schema declares `"$schema": "http://json-schema.org/draft-07/schema#"`, which v6 compiles. This is the repo's first use of the library, so if `go build` disagrees, trust the compiler and fix the block.

- [ ] **Step 6: Implement `main.go`**

Structure it exactly like `tools/catalog-lint/main.go`: a `main` that checks `len(os.Args) != 3`, prints `usage: reactor-seed-lint <seed-root> <schema-path>` to stderr and exits 2; a `lint(root, schemaPath string) error` that accumulates `errs []string` and returns one aggregated error; `os.Exit(1)` on a non-nil error.

`lint` must:
- compile the schema once, up front
- `filepath.WalkDir(root, ...)`, skipping dirs and files whose base name starts with `_` or `.`
- select only files whose parent-of-parent path relative to the version dir is `reactor-actions/reactors` and whose name matches `^reactor-(.+)\.json$`
- for each: `seeder.ParseEnvelope(b)`, then assert `env.Data.Type == "reactor-action"`; `seeder.ExtractEntityID(base, reactorFilePattern)` and assert it equals `env.Data.ID`; unmarshal `env.Data.Attributes` into a struct with `ReactorId string \`json:"reactorId"\`` and assert `reactorId == env.Data.ID`; then `validateAttributes`
- record every failure as `fmt.Sprintf("%s: %v", path, err)` and keep walking — the point is one run listing every bad file, not the first

Declare at package scope:

```go
var reactorFilePattern = regexp.MustCompile(`^reactor-(.+)\.json$`)
```

Check `libs/atlas-seeder/jsonapi.go:9-19` for whether `EnvelopeData.Attributes` is `json.RawMessage` or `[]byte` and match it; if it is neither, re-marshal `env.Data.Attributes` before validating.

- [ ] **Step 7: Run the tests and confirm they pass**

Run: `cd tools/reactor-seed-lint && go build ./... && go vet ./... && go test ./... -v`
Expected: all five tests PASS.

- [ ] **Step 8: Commit**

```bash
git add go.work tools/reactor-seed-lint
git commit -m "feat(tools): add reactor-seed-lint with schema and envelope assertions"
```

---

## Task 5: `reactor-seed-lint` — non-empty description and cross-tenant byte identity

### Files

- `tools/reactor-seed-lint/main.go` — add assertions 3 and 4 to `lint`
- `tools/reactor-seed-lint/identity.go` — **new file**; the cross-tenant SHA-256 comparison
- `tools/reactor-seed-lint/main_test.go` — new file from Task 4; add three tests
- `tools/reactor-seed-lint/testdata/` — new file tree from Task 4; add three fixture trees

Module root: `tools/reactor-seed-lint`

**Assertions 3 and 4:**

3. **Non-empty `description`** (FR-15) — a corpus rule, not a schema rule: the schema keeps `description` optional because the REST resource legitimately allows it to be absent.
4. **Cross-tenant byte identity** (FR-14) — every reactor id present in ANY version directory must be present in ALL of them with an identical SHA-256.

### Steps

- [ ] **Step 1: Add the fixtures**

| fixture path | content | expected |
|---|---|---|
| `bad/no-description/gms/83_1/reactor-actions/reactors/reactor-2001.json` | valid envelope, `attributes` omits `description` | non-zero, message contains `description` |
| `bad/divergent-copies/gms/83_1/.../reactor-2001.json` and `bad/divergent-copies/gms/95_1/.../reactor-2001.json` | valid, but `95_1`'s description differs by one word | non-zero, message names both directories or the reactor id `2001` |
| `bad/missing-copy/gms/83_1/.../reactor-2001.json` and `bad/missing-copy/gms/95_1/.../reactor-9102002.json` | each dir has a reactor the other lacks | non-zero |

Extend `testdata/good` to a second version directory (`good/gms/95_1/reactor-actions/reactors/reactor-2001.json`) that is a byte-for-byte copy, so assertion 4 has a passing case.

- [ ] **Step 2: Write the failing tests**

```
TestLint_MissingDescriptionExitsNonZero → exe testdata/bad/no-description    <schema> → non-zero, output contains "description"
TestLint_DivergentCopiesExitNonZero     → exe testdata/bad/divergent-copies  <schema> → non-zero, output contains "2001"
TestLint_MissingCopyExitsNonZero        → exe testdata/bad/missing-copy      <schema> → non-zero
```

`TestLint_GoodCorpusExitsZero` from Task 4 must still pass with the added `95_1` copy.

- [ ] **Step 3: Run the tests and confirm they fail**

Run: `cd tools/reactor-seed-lint && go test ./... -run 'MissingDescription|DivergentCopies|MissingCopy' -v`
Expected: FAIL — all three exit 0 today.

- [ ] **Step 4: Implement assertion 3 in `main.go`**

Extend the attributes struct decoded in `lint` to `struct { ReactorId string \`json:"reactorId"\`; Description string \`json:"description"\` }` and record an error when `strings.TrimSpace(Description) == ""`:

```go
if strings.TrimSpace(attrs.Description) == "" {
	errs = append(errs, fmt.Sprintf("%s: empty or missing description", path))
}
```

- [ ] **Step 5: Implement assertion 4 in `identity.go`**

```go
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// checkIdentity asserts every reactor id seen in any version directory is
// present in every version directory with an identical SHA-256. seen maps
// version dir (e.g. "gms/83_1") -> reactor id -> file digest.
func checkIdentity(seen map[string]map[string]string) []string {
	var errs []string

	versions := make([]string, 0, len(seen))
	for v := range seen {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	ids := map[string]struct{}{}
	for _, v := range versions {
		for id := range seen[v] {
			ids[id] = struct{}{}
		}
	}
	allIds := make([]string, 0, len(ids))
	for id := range ids {
		allIds = append(allIds, id)
	}
	sort.Strings(allIds)

	for _, id := range allIds {
		var missing []string
		digests := map[string][]string{}
		for _, v := range versions {
			d, ok := seen[v][id]
			if !ok {
				missing = append(missing, v)
				continue
			}
			digests[d] = append(digests[d], v)
		}
		if len(missing) > 0 {
			errs = append(errs, fmt.Sprintf("reactor %s: missing from %s", id, strings.Join(missing, ", ")))
		}
		if len(digests) > 1 {
			var groups []string
			for d, vs := range digests {
				groups = append(groups, fmt.Sprintf("%s=[%s]", d[:12], strings.Join(vs, " ")))
			}
			sort.Strings(groups)
			errs = append(errs, fmt.Sprintf("reactor %s: copies differ: %s", id, strings.Join(groups, " ")))
		}
	}
	return errs
}

// digest is the file's SHA-256, hex encoded.
func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// versionKey turns "<root>/gms/83_1/reactor-actions/reactors/reactor-2001.json"
// into "gms/83_1".
func versionKey(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 5 {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}
```

Populate `seen` inside the walk in `lint` and append `checkIdentity(seen)...` to `errs` after the walk returns.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `cd tools/reactor-seed-lint && go build ./... && go vet ./... && go test ./... -v`
Expected: all eight tests PASS.

- [ ] **Step 7: Commit**

```bash
git add tools/reactor-seed-lint
git commit -m "feat(tools): assert non-empty description and cross-tenant byte identity in reactor-seed-lint"
```

---

## Task 6: Wrapper script and `verify.sh` wiring

### Files

- `tools/reactor-seed-lint.sh` — **new file**; the wrapper `verify.sh` calls
- `tools/reactor-seed-lint_test.sh` — **new file**; its suite, and the home for both tools modules' `go test`
- `tools/verify.sh` — add the path-gated step
- `tools/shell-guard.sh` — read-only; the parse + shellcheck rules the new scripts must satisfy
- `tools/mode-select_test.sh` — read-only; the `_test.sh` shape to copy

Patterns to copy: `tools/verify.sh:730-734` (the two-line `if touched ... step ... else skip ... fi` block), `tools/mode-select_test.sh:1-9` (`set -eu` + `fail()` preamble).

**Why the Go tests live in the shell suite:** `verify.sh`'s `all_modules()` walks only `$ROOT/services` and `$ROOT/libs` and explicitly excludes `tools/` modules (`tools/verify.sh:392-398`). `changed_tool_suites()` (`tools/verify.sh:225-243`) maps a changed `tools/foo.sh` to `tools/foo_test.sh` and runs it. Putting `go test ./tools/reactor-seed-lint/...` and `go test ./tools/reactor-seed-gen/...` in `reactor-seed-lint_test.sh` is the only wiring that puts them inside the gate. The precedent is `tools/verify.sh:883`, which runs the topic-generator's Go tests as a named `step`.

### Steps

- [ ] **Step 1: Write `tools/reactor-seed-lint.sh`**

```bash
#!/usr/bin/env bash
# tools/reactor-seed-lint.sh — validates the reactor-action seed corpus.
#
# Why this exists: reactor_script_schema.json describes the reactor script
# resource, but nothing ever validated a seed file against it. The corpus
# grew a meso-parameter regression (reactor-2001 seeded minMeso/maxMeso/
# mesoRange/item, none of which executor.go reads) that no gate could see.
#
# Runs tools/reactor-seed-lint over deploy/seed. That binary asserts, per
# file: schema conformance of data.attributes, envelope well-formedness
# (type/id/filename agreement), a non-empty description, and — across the
# eleven tenant directories — byte identity of every reactor's copies.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

SEED_ROOT="${1:-deploy/seed}"
SCHEMA="services/atlas-reactor-actions/docs/reactor_script_schema.json"

if [ ! -d "$SEED_ROOT" ]; then
    echo "reactor-seed-lint: ERROR — seed root not found: $SEED_ROOT" >&2
    exit 1
fi
if [ ! -f "$SCHEMA" ]; then
    echo "reactor-seed-lint: ERROR — schema not found: $SCHEMA" >&2
    exit 1
fi

go run ./tools/reactor-seed-lint "$SEED_ROOT" "$SCHEMA"
```

- [ ] **Step 2: Write `tools/reactor-seed-lint_test.sh`**

```bash
#!/usr/bin/env bash
# tools/reactor-seed-lint_test.sh — suite for reactor-seed-lint.sh, and the
# gate's only entry point for the Go tests of tools/reactor-seed-lint and
# tools/reactor-seed-gen. verify.sh's all_modules() walks services/ and
# libs/ only (verify.sh:392-398), so tools/ modules are not in the go
# build/vet/test sweep; changed_tool_suites() (verify.sh:225-243) runs this
# file whenever reactor-seed-lint.sh changes, which is where they belong.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
fail() { echo "FAIL: $*" >&2; exit 1; }

# 1. The tools' own Go tests.
(cd tools/reactor-seed-lint && go test ./...) || fail "reactor-seed-lint go test"
if [ -d tools/reactor-seed-gen ]; then
    (cd tools/reactor-seed-gen && go test ./...) || fail "reactor-seed-gen go test"
fi

# 2. The real corpus passes.
./tools/reactor-seed-lint.sh >/dev/null || fail "reactor-seed-lint.sh over deploy/seed"

# 3. A missing seed root is an error, not a silent pass.
if ./tools/reactor-seed-lint.sh no/such/dir >/dev/null 2>&1; then
    fail "missing seed root should exit non-zero"
fi

echo "reactor-seed-lint_test.sh: OK"
```

Guard 3 is the important one: a wrapper that exits 0 when it validated nothing is the failure mode this suite exists to prevent.

- [ ] **Step 3: Wire it into `tools/verify.sh`**

Insert after the `npc-shop contract mirror guard` block (`tools/verify.sh:775-779`), matching that block's shape exactly:

```bash
if touched '^deploy/seed/.*/reactor-actions/|^services/atlas-reactor-actions/docs/reactor_script_schema\.json$|^tools/reactor-seed-lint|^tools/reactor-seed-gen/'; then
    step "reactor seed lint" ./tools/reactor-seed-lint.sh
else
    skip "reactor seed lint (no reactor seed or schema changed)"
fi
```

- [ ] **Step 4: Make the scripts executable and run the guards**

```bash
chmod +x tools/reactor-seed-lint.sh tools/reactor-seed-lint_test.sh
./tools/shell-guard.sh --require-shellcheck tools/reactor-seed-lint.sh tools/reactor-seed-lint_test.sh
```

Expected: exit 0. Fix any `shellcheck` finding at severity `error` before proceeding.

- [ ] **Step 5: Run the suite and confirm it FAILS on the real corpus**

Run: `./tools/reactor-seed-lint_test.sh`
Expected: **FAIL** at guard 2 — `deploy/seed/*/*/reactor-actions/reactors/reactor-2001.json` still carries `minMeso`/`maxMeso`/`mesoRange`/`item`, which Task 2's `additionalProperties: false` now rejects. This is the expected state; Task 7 fixes it. Record the failure text; do not weaken the lint to make it pass.

- [ ] **Step 6: Verify the gate selects the step**

Run: `./tools/verify.sh --facts 2>&1 | grep -i 'reactor seed lint'`
Expected: the step appears in the selected list (this branch has touched `deploy/seed`-adjacent paths and `tools/reactor-seed-lint*`).

- [ ] **Step 7: Commit**

```bash
git add tools/reactor-seed-lint.sh tools/reactor-seed-lint_test.sh tools/verify.sh
git commit -m "feat(tools): wire reactor-seed-lint into verify.sh"
```

---

## Task 7: Correct `reactor-2001.json` and audit the other twelve existing seeds

### Files

- `deploy/seed/gms/12_1/reactor-actions/reactors/reactor-2001.json` — correct the meso params
- `deploy/seed/gms/48_1/reactor-actions/reactors/reactor-2001.json` — same
- `deploy/seed/gms/61_1/reactor-actions/reactors/reactor-2001.json` — same
- `deploy/seed/gms/72_1/reactor-actions/reactors/reactor-2001.json` — same
- `deploy/seed/gms/79_1/reactor-actions/reactors/reactor-2001.json` — same
- `deploy/seed/gms/83_1/reactor-actions/reactors/reactor-2001.json` — same
- `deploy/seed/gms/84_1/reactor-actions/reactors/reactor-2001.json` — same
- `deploy/seed/gms/87_1/reactor-actions/reactors/reactor-2001.json` — same
- `deploy/seed/gms/92_1/reactor-actions/reactors/reactor-2001.json` — same
- `deploy/seed/gms/95_1/reactor-actions/reactors/reactor-2001.json` — same
- `deploy/seed/jms/185_1/reactor-actions/reactors/reactor-2001.json` — same
- `docs/tasks/task-291-reactor-tier1-conversion/existing-seed-audit.md` — **new file**; the recorded audit outcome

This is eleven copies of one mechanical edit plus one new document — the repeated-mechanical-change exception to the six-file rule. No Go module is touched.

The twelve PQ scripts' sources are at `<Cosmic>/scripts/reactor/<id>.js` for ids `9102002`–`9102007` and `9108000`–`9108005`. The Cosmic checkout is a sibling reference-server clone outside this repository; ask the user for its path if it is not already known. **Do not write that absolute path into the audit document** — refer to sources as `<Cosmic>/scripts/reactor/<id>.js`.

### Steps

- [ ] **Step 1: Confirm the pre-state**

Run: `grep -rln 'minMeso\|maxMeso\|mesoRange\|"item"' deploy/seed/*/*/reactor-actions/`
Expected: exactly the eleven `reactor-2001.json` paths and nothing else. (Verified at plan time — `reactor-2001.json` is the only affected file.)

- [ ] **Step 2: Correct `reactor-2001.json` in all eleven directories**

The single `drop_items` operation's `params` object becomes exactly (keys alphabetical, values strings):

```json
"params": {
  "meso": "true",
  "mesoChance": "2",
  "mesoMax": "15",
  "mesoMin": "8",
  "minItems": "1"
}
```

Source: `rm.dropItems(true, 2, 8, 15, 1)` against `(meso, mesoChance, minMeso, maxMeso, minItems)`. Every other byte of the file — including the `description`, the rule `id` `drop_items`, the empty `hitRules`, the 2-space indent and the trailing newline — is unchanged.

- [ ] **Step 3: Verify the correction and the byte identity**

Run: `grep -rn 'minMeso\|maxMeso\|mesoRange\|"item"' deploy/seed/*/*/reactor-actions/`
Expected: no output.

Run: `md5sum deploy/seed/*/*/reactor-actions/reactors/reactor-2001.json | awk '{print $1}' | sort -u | wc -l`
Expected: `1`.

- [ ] **Step 4: Audit the other twelve against their Cosmic sources**

Read each of `<Cosmic>/scripts/reactor/{9102002..9102007,9108000..9108005}.js` and its seed file in `deploy/seed/gms/83_1/reactor-actions/reactors/`. For each, decide `correct` or `corrected: <specific change>` — the PRD's acceptance criterion is the **recorded outcome**, so a bare "checked" is a task failure. Apply any correction to all eleven copies.

Two reference points established at plan time:
- `9102002`'s source is `function act() { rm.dropItems(); }` and its seed is one `actRule` with a single `{"type":"drop_items"}` operation carrying no `params` key — `correct`.
- `9108000`'s seed carries two `actRules` (a `pq_custom_data` `stage = 5` guarded rule with `update_pq_state`/`hit_reactor`/`stage_clear_attempt`, then an unguarded `update_pq_state`/`hit_reactor` fallback). Rule order is load-bearing: `processor.go` takes the first matching rule, so the guarded rule must stay first.

- [ ] **Step 5: Write `existing-seed-audit.md`**

```markdown
# Existing Seed Audit — task-291

Thirteen reactor-action seeds existed before this task. Each was read against
its Cosmic source and given a recorded outcome. Sources are referenced as
`<Cosmic>/scripts/reactor/<id>.js`; seeds as
`deploy/seed/gms/83_1/reactor-actions/reactors/reactor-<id>.json` (the eleven
tenant copies are byte-identical).

## Outcomes

| Reactor | Source body | Seeded shape | Verdict |
|---|---|---|---|
| 2001 | `rm.dropItems(true, 2, 8, 15, 1)` | `drop_items` | **corrected**: `minMeso=2, maxMeso=8, mesoRange=15, item=1` → `mesoChance=2, mesoMin=8, mesoMax=15, minItems=1` |
| 9102002 | `rm.dropItems()` | `drop_items`, no params | correct |
| ... one row per remaining reactor ... |
```

Fill in every one of the thirteen rows with the actual source body and the actual verdict. A row reading "checked" or "OK" without naming the shape is not an outcome.

- [ ] **Step 6: Run the lint over the real corpus**

Run: `./tools/reactor-seed-lint_test.sh`
Expected: **PASS** — this is the first point at which the 13-file corpus is clean. If it still fails, the failure names the file and the assertion; fix the file, never the lint.

- [ ] **Step 7: Commit**

```bash
git add deploy/seed docs/tasks/task-291-reactor-tier1-conversion/existing-seed-audit.md
git commit -m "fix(seed): correct reactor-2001 meso mapping and record the existing-seed audit"
```

---

## Task 8: `tools/reactor-seed-gen` — module scaffold and the inventory parser

### Files

- `tools/reactor-seed-gen/go.mod` — **new file**
- `tools/reactor-seed-gen/parse.go` — **new file**
- `tools/reactor-seed-gen/parse_test.go` — **new file**
- `tools/reactor-seed-gen/testdata/inventory-fixture.md` — **new file**
- `go.work` — add `./tools/reactor-seed-gen` to the `use` block, alphabetically after `./tools/packet-audit`
- `docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md` — read-only; the real input, 159 `### \`<id>\`` sections

Module root for `go build`/`go test`: `tools/reactor-seed-gen`

Patterns to copy: `tools/catalog-lint/go.mod` (module + `replace` shape — this module needs no `replace` because it has no repo-internal dependency), `tools/catalog-lint/main_test.go` (test file shape).

**Inventory format, read from the real file:**

```
### `1002008`
*(no comment in source)*
```javascript
function act() {
    rm.dropItems();
}
```

### `1012000`
**Source comment:** @Author Lerk * * 1012000.js: Ellinia Plant - drops meso ... www.gnu.org/licenses/>.
```javascript
function act() {
    rm.dropItems(true, 2, 20, 40);
}
```
```

Exactly 159 headings. A section is: an `### \`<id>\`` line, then either `*(no comment in source)*` or a `**Source comment:** <text>` line, then a ```` ```javascript ```` fence holding one or two function bodies (`function hit() {...}` and/or `function act() {...}`, in either order, sometimes `function act() {}` on one line).

### Steps

- [ ] **Step 1: Create the module**

```bash
mkdir -p tools/reactor-seed-gen/testdata
cd tools/reactor-seed-gen
cat > go.mod <<'EOF'
module github.com/Chronicle20/atlas/tools/reactor-seed-gen

go 1.27.0
EOF
```

Add `./tools/reactor-seed-gen` to `go.work`'s `use` block.

- [ ] **Step 2: Write the fixture**

`testdata/inventory-fixture.md` — six sections exercising every structural variant, copied verbatim from the real inventory:

| id | comment line | body |
|---|---|---|
| `1002008` | `*(no comment in source)*` | `function act() {\n    rm.dropItems();\n}` |
| `1012000` | `**Source comment:** @Author Lerk * * 1012000.js: Ellinia Plant - drops meso, tree branches, red pots, and Plant Samples (quest item) www.gnu.org/licenses/>.` | `function act() {\n    rm.dropItems(true, 2, 20, 40);\n}` |
| `2119000` | `**Source comment:** * Tombstone in Forest of Dead Trees I ...` | `function hit() {\n    if (rm.getReactor().getState() !== 0) {\n        return\n    }\n    rm.weakenAreaBoss(6090000, "As the tombstone lit up and vanished, Lich lost all his magic abilities.")\n}\nfunction act() {\n}` |
| `2612004` | `**Source comment:** 2612004.js - Zenumist crystal *@author Ronan www.gnu.org/licenses/>.` | `function hit() {\n    rm.sprayItems();\n}\nfunction act() {}` |
| `9018000` | `**Source comment:** * * @author BubblesDev * @purpose Flower 1 www.gnu.org/licenses/>.` | `function act() {\n}` |
| `2511000` | `**Source comment:** 2511000- Reactor for PPQ [Pirate PQ] @author Jvlaple www.gnu.org/licenses/>.` | the four-statement PQ body from the real inventory |

- [ ] **Step 3: Write the failing test**

`parse_test.go`, package `main`. `TestParseInventory` calls `parseInventory(<fixture bytes>)` and asserts, subtest per case:

| subtest | assertion |
|---|---|
| `count` | `len(scripts) == 6` |
| `order preserved` | ids in file order: `1002008`, `1012000`, `2119000`, `2612004`, `9018000`, `2511000` |
| `no comment` | `scripts[0].Comment == ""` |
| `comment captured` | `scripts[1].Comment` starts with `@Author Lerk` and ends with `www.gnu.org/licenses/>.` (the raw comment, uncleaned — cleaning is Task 10) |
| `act only` | `scripts[0].HitBody == ""` and `scripts[0].ActBody` contains `rm.dropItems();` |
| `hit and empty act` | `scripts[2].HitBody` contains both `getState() !== 0` and `weakenAreaBoss`; `scripts[2].ActBody == ""` |
| `single-line empty act` | `scripts[3].HitBody` contains `rm.sprayItems();`; `scripts[3].ActBody == ""` |
| `empty act only` | `scripts[4].HitBody == ""` and `scripts[4].ActBody == ""` |
| `multi-statement act` | `scripts[5].ActBody` has 4 non-empty lines |

Add `TestParseInventory_RealFile`: parse `../../docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md` and assert `len(scripts) == 159` and that every `Id` is non-empty and unique. This is the regression that catches a parser silently dropping sections.

- [ ] **Step 4: Run the test and confirm it fails**

Run: `cd tools/reactor-seed-gen && go test ./... -v`
Expected: FAIL — `parseInventory` undefined.

- [ ] **Step 5: Implement `parse.go`**

```go
package main

// sourceScript is one reactor's verbatim inventory section.
type sourceScript struct {
	Id      string // e.g. "2119000"
	Comment string // raw Source comment text, "" when the section says (no comment in source)
	HitBody string // statements inside function hit() { ... }, "" when absent or empty
	ActBody string // statements inside function act() { ... }, "" when absent or empty
}

func parseInventory(b []byte) ([]sourceScript, error)
```

Implementation notes, all mechanical:
- split on lines matching `^### ` + backtick + `(\S+)` + backtick + `\s*$`
- within a section, a line matching `^\*\*Source comment:\*\* (.*)$` yields `Comment`; `*(no comment in source)*` yields `""`
- capture the text between the ```` ```javascript ```` and ```` ``` ```` fences as the raw body
- from the raw body, extract each `function <name>() {` ... matching `}` by brace counting (bodies are never nested beyond the `if`/`for` blocks, so a simple depth counter is sufficient and must not be replaced by a regex)
- trim the extracted body; an all-whitespace body is `""`
- an unknown function name, an unterminated fence, or a heading with no fence is an **error** naming the reactor id — never a skip

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `cd tools/reactor-seed-gen && go build ./... && go vet ./... && go test ./... -v`
Expected: PASS, including `TestParseInventory_RealFile` reporting 159.

- [ ] **Step 7: Commit**

```bash
git add go.work tools/reactor-seed-gen
git commit -m "feat(tools): add reactor-seed-gen inventory parser"
```

---

## Task 9: `reactor-seed-gen` — the whitelisted grammar

### Files

- `tools/reactor-seed-gen/convert.go` — **new file**; the grammar, the guard hoist, the loop unroll, the PQ idiom
- `tools/reactor-seed-gen/doc.go` — **new file**; the `scriptDoc`/`ruleDoc`/`opDoc` types
- `tools/reactor-seed-gen/convert_test.go` — **new file**
- `tools/reactor-seed-gen/parse.go` — read-only; `sourceScript` is the input
- `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` — lines 100-256; read-only; the parameter names this must produce

Module root: `tools/reactor-seed-gen`

### Steps

- [ ] **Step 1: Define the document types in `doc.go`**

```go
package main

// scriptDoc is the bare script object that becomes data.attributes.
type scriptDoc struct {
	ReactorId   string
	Description string
	HitRules    []ruleDoc
	ActRules    []ruleDoc
}

// ruleDoc is one rule. Conditions are ANDed; an empty slice always matches.
type ruleDoc struct {
	Id         string
	Conditions []condDoc
	Operations []opDoc
}

type condDoc struct {
	Type     string // "reactor_state" | "pq_custom_data"
	Operator string // "=" "!=" ">" "<" ">=" "<="
	Value    string
	Step     string // pq_custom_data only; omitted when empty
}

type opDoc struct {
	Type   string
	Params map[string]string // nil means "emit no params key at all"
}
```

- [ ] **Step 2: Write the failing test**

`convert_test.go`, package `main`. `TestConvertBody` — table-driven, one case per grammar row. Each case feeds a `sourceScript` and asserts the resulting `scriptDoc`'s `HitRules`/`ActRules` with `reflect.DeepEqual`. Every input below is verbatim from `tier1-inventory.md`.

| case (reactor) | body | expected rules |
|---|---|---|
| bare drop (`1002008`) | `rm.dropItems();` | 1 act rule, id `drop_items`, no conditions, ops `[{drop_items, Params: nil}]` |
| drop 4-arg (`1012000`) | `rm.dropItems(true, 2, 20, 40);` | ops `[{drop_items, {meso:"true", mesoChance:"2", mesoMin:"20", mesoMax:"40"}}]` |
| drop 5-arg (`2001` shape) | `rm.dropItems(true, 2, 8, 15, 1);` | ops `[{drop_items, {meso:"true", mesoChance:"2", mesoMin:"8", mesoMax:"15", minItems:"1"}}]` |
| drop 5-arg false (`3102000`) | `rm.dropItems(false, 0, 0, 0, 3);` | ops `[{drop_items, {meso:"false", mesoChance:"0", mesoMin:"0", mesoMax:"0", minItems:"3"}}]` |
| bare spray (`2612004` hit) | `rm.sprayItems();` | 1 **hit** rule, id `spray_items`, ops `[{spray_items, Params: nil}]`, and `ActRules` empty |
| spray 5-arg (`1052001`) | `rm.sprayItems(true, 1, 500, 1000, 15);` | ops `[{spray_items, {meso:"true", mesoChance:"1", mesoMin:"500", mesoMax:"1000", minItems:"15"}}]` |
| spawn 1-arg (`1021000`) | `rm.spawnMonster(9300091);` | ops `[{spawn_monster, {monsterId:"9300091"}}]` |
| spawn 2-arg (`2201000`) | `rm.spawnMonster(9300011, 10);` | ops `[{spawn_monster, {monsterId:"9300011", count:"10"}}]` |
| spawn 4-arg (`8001000`) | `rm.spawnMonster(9400112, 1, 420, 160);` | ops `[{spawn_monster, {monsterId:"9400112", count:"1", x:"420", y:"160"}}]` |
| spawn 4-arg negative x (`9201000`) | `rm.spawnMonster(9300033, 8, -100, 50);` | ops `[{spawn_monster, {monsterId:"9300033", count:"8", x:"-100", y:"50"}}]` |
| weaken, no semicolon (`2119000` hit) | `rm.weakenAreaBoss(6090000, "As the tombstone lit up and vanished, Lich lost all his magic abilities.")` | ops `[{weaken_area_boss, {monsterId:"6090000", message:"As the tombstone lit up and vanished, Lich lost all his magic abilities."}}]` |
| weaken, semicolon (`2119004`) | `rm.weakenAreaBoss(6090001, "The light at the altar appeases the hatred of the Snow Witch. The force of the Witch has weakened.");` | ops `[{weaken_area_boss, {monsterId:"6090001", message:"The light at the altar appeases the hatred of the Snow Witch. The force of the Witch has weakened."}}]` |
| **guard hoist** (`2119000` hit) | `if (rm.getReactor().getState() !== 0) {\n    return\n}\nrm.weakenAreaBoss(6090000, "...")` | 1 hit rule, id `weaken_area_boss`, conditions `[{reactor_state, "=", "0", Step:""}]`, ops as the weaken row above |
| **loop unroll, one op** (`2201001`) | `for (var i = 0; i < 3; i++) {\n    rm.spawnMonster(9300007);\n}` | ops `[{spawn_monster, {monsterId:"9300007", count:"3"}}]` |
| **loop unroll, two ops** (`2511001`) | `for (var i = 0; i < 6; i++) {\n    rm.spawnMonster(9300124);\n    rm.spawnMonster(9300125);\n}` | ops `[{spawn_monster, {monsterId:"9300124", count:"6"}}, {spawn_monster, {monsterId:"9300125", count:"6"}}]`, rule id `spawn_monster` |
| **increment idiom** (`2511000`) | `var eim = rm.getPlayer().getEventInstance();\nvar now = eim.getIntProperty("openedBoxes");\nvar nextNum = now + 1;\neim.setIntProperty("openedBoxes", nextNum);\nrm.spawnMonster(9300109, 3);\nrm.spawnMonster(9300110, 5);` | ops `[{update_pq_state, {increments:"openedBoxes"}}, {spawn_monster, {monsterId:"9300109", count:"3"}}, {spawn_monster, {monsterId:"9300110", count:"5"}}]`, rule id `update_pq_state_spawn_monster` |
| **increment + spray** (`2512001`) | the `openedChests` body plus `rm.sprayItems(true, 1, 50, 100, 15);` | ops `[{update_pq_state, {increments:"openedChests"}}, {spray_items, {meso:"true", mesoChance:"1", mesoMin:"50", mesoMax:"100", minItems:"15"}}]`, rule id `update_pq_state_spray_items` |
| **setProperty after drop** (`2002003`) | `rm.dropItems();\nvar eim = rm.getEventInstance();\neim.setProperty("statusStg7", "1");` | ops `[{drop_items, nil}, {update_pq_state, {updates:"statusStg7=1"}}]`, rule id `drop_items_update_pq_state` |
| **inline setProperty** (`2008006`) | `rm.getEventInstance().setProperty("statusStg3", "0");` | ops `[{update_pq_state, {updates:"statusStg3=0"}}]`, rule id `update_pq_state` |
| **null-guard erased** (`9208009`) | `if (rm.getEventInstance() != null) {\n    rm.getEventInstance().setProperty("canRevive", "1");\n}` | ops `[{update_pq_state, {updates:"canRevive=1"}}]`, rule id `update_pq_state`, **no** condition |
| **empty act** (`9018000`) | `` (empty ActBody) | `ActRules` is an empty, non-nil slice; `HitRules` likewise |

`TestConvertBody_NegativeCases` — each asserts `convertScript` returns an error whose message contains the reactor id **and** the offending source line:

| case | body | why it must abort |
|---|---|---|
| unknown call | `rm.doSomethingElse(1);` | not in the grammar |
| random spawn | `rm.spawnMonster(Math.random() >= .6 ? 9300049 : 9300048);` | not a literal |
| non-unit increment | `var now = eim.getIntProperty("k");\nvar nextNum = now + 2;\neim.setIntProperty("k", nextNum);` | the idiom is `+ 1`; `+ 2` must not be silently converted to `increments` |
| non-spawn in a loop | `for (var i = 0; i < 3; i++) {\n    rm.dropItems();\n}` | design §3.2: only `spawn_monster` unrolls |
| non-literal loop bound | `for (var i = 0; i < n; i++) {\n    rm.spawnMonster(1);\n}` | bound is not a literal |
| guard on a non-zero state | *(no such script in Tier 1 — omit)* | — |

- [ ] **Step 3: Run the tests and confirm they fail**

Run: `cd tools/reactor-seed-gen && go test ./... -run TestConvertBody -v`
Expected: FAIL — `convertScript` undefined.

- [ ] **Step 4: Implement `convert.go`**

```go
package main

// convertScript walks a parsed inventory section against the whitelisted
// grammar. It sets ReactorId, HitRules and ActRules; Description is left
// empty and filled in by main.go from describe() (Task 10), so the grammar
// and the comment cleaner stay independently testable. Any line matching no
// whitelisted form is a hard error naming the reactor id and the line —
// there is no skip-and-report and no partial output.
func convertScript(s sourceScript) (scriptDoc, error)

// convertBody turns one function body into its rules.
func convertBody(reactorId, body string) ([]ruleDoc, error)
```

Rules that are easy to get wrong and are asserted above:

- **One rule per body.** Every Tier-1 body produces at most one rule; the guard, when present, applies to it. Do not emit one rule per operation.
- **Rule id** = the distinct operation types, in first-appearance order, joined with `_`. `[update_pq_state, spawn_monster, spawn_monster]` → `update_pq_state_spawn_monster`.
- **Guard inversion.** `if (rm.getReactor().getState() !== K) { return }` becomes `{Type: "reactor_state", Operator: "=", Value: "K"}` — the positive form. `!==` → `=` is the inversion; do not carry `!=` through.
- **Loop unroll multiplies.** `count = K × (the call's own count, default 1)`. Every Tier-1 loop body uses the 1-arg `spawnMonster`, so K is the count; write the multiplication anyway so a 2-arg call inside a loop cannot silently lose its factor.
- **The increment idiom is matched as a three-statement unit** (`getIntProperty(k)` → `var x = <that> + 1` → `setIntProperty(k, x)`), with the same key `k` in the read and the write. Matching `getIntProperty` alone would lose the fact that the write is an increment of the read. A mismatched key, or any addend other than `1`, is an error.
- **`var eim = ...getEventInstance()` and `if (<anything>.getEventInstance() != null) {` produce nothing** — the binding and the null guard are erased, and the guard's body is converted in place. A null guard is NOT a condition.
- **Bare `rm.dropItems()` / `rm.sprayItems()` set `Params: nil`**, not an empty map — that is what makes the emitter omit the `params` key, matching `reactor-9102002.json`.
- **Trailing semicolons are optional** (four Tier-1 `weakenAreaBoss` calls have none).
- **String arguments keep their content verbatim**, including `.` and `!`; strip only the surrounding double quotes.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd tools/reactor-seed-gen && go build ./... && go vet ./... && go test ./... -v`
Expected: PASS, every positive and negative case.

- [ ] **Step 6: Commit**

```bash
git add tools/reactor-seed-gen
git commit -m "feat(tools): add the whitelisted reactor grammar to reactor-seed-gen"
```

---

## Task 10: `reactor-seed-gen` — descriptions and the override table

### Files

- `tools/reactor-seed-gen/describe.go` — **new file**; the boilerplate cleaner and the override table
- `tools/reactor-seed-gen/describe_test.go` — **new file**
- `docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md` — read-only; the comment corpus

Module root: `tools/reactor-seed-gen`

### Steps

- [ ] **Step 1: Write the failing test**

`describe_test.go`, package `main`. `TestDescribe` — table-driven; each case is a real inventory comment.

| subtest | input comment (verbatim) | want |
|---|---|---|
| `full boilerplate then purpose` | `This file is part of the HeavenMS MapleStory Server Copyleft (L) 2016 - 2019 RonanLana This program is free software: ... see <http://www.gnu.org/licenses/>. @Author Ronan * * 1021000.js: relic room fail www.gnu.org/licenses/>.` | `Relic room fail` |
| `author then purpose` | `@Author Lerk * * 1012000.js: Ellinia Plant - drops meso, tree branches, red pots, and Plant Samples (quest item) www.gnu.org/licenses/>.` | `Ellinia Plant - drops meso, tree branches, red pots, and Plant Samples (quest item)` |
| `purpose tag` | `* * @author BubblesDev * @purpose Flower 1 www.gnu.org/licenses/>.` | `Flower 1` |
| `blogspot debris` | `* Tombstone in Forest of Dead Trees I MSEA reference: http://mymapleland.blogspot.com/2009/09/kill-lich-at-forest-of-dead-trees-i-to.html www.gnu.org/licenses/>. mymapleland.blogspot.com/2009/09/kill-lich-at-forest-of-dead-trees-i-to.html If the chest is destroyed before Riche, killing him should yield no exp` | must contain `Tombstone in Forest of Dead Trees I` and must NOT contain `blogspot` |
| `no comment, override present` | `""` for reactor `1002008` | the override table's entry for `1002008` |
| `reduces to nothing, override present` | `www.gnu.org/licenses/>.` for reactor `8001000` | the override table's entry for `8001000` |

`TestDescribe_MissingOverrideAborts` — a `sourceScript` with an empty comment and an id absent from the override table returns an error naming that id. This is the fail-closed property: the generator never invents a description.

`TestDescribe_NoBoilerplateLeaks` — run `describe` over **every one of the 159** real inventory sections and assert no returned description contains any of `gnu.org`, `Copyleft`, `@author`, `@Author`, `@purpose`, `blogspot`, `WITHOUT ANY WARRANTY`, `HeavenMS`, `OdinMS`, or `.js:`, and that none is empty. This is the corpus-wide guarantee; a per-case test cannot give it.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd tools/reactor-seed-gen && go test ./... -run TestDescribe -v`
Expected: FAIL — `describe` undefined.

- [ ] **Step 3: Implement the cleaner in `describe.go`**

```go
package main

// describe derives a seed description from a reactor's raw source comment.
// The cleaner strips AGPL boilerplate, author and purpose tags, URL debris,
// the "<id>.js:" prefix and "*" decorations. When the result is empty or
// shorter than minDescriptionLen, the override table must supply one;
// there is no fallback that invents text.
func describe(reactorId, comment string) (string, error)

const minDescriptionLen = 6
```

Cleaning order (each step's removals are verified against the real corpus):
1. remove the HeavenMS/OdinMS license paragraph — everything from `This file is part of` through the first `<http://www.gnu.org/licenses/>.`
2. remove `@Author <name>`, `@author <name>`, and the `@purpose ` tag (keeping the text that follows `@purpose`)
3. remove `www.gnu.org/licenses/>.`, `http://...` and `mymapleland.blogspot.com/...` runs
4. remove a leading `<digits>.js:` or `<digits>.js -` prefix and the reactor's own `<id>.js` anywhere
5. strip `*` decorations and collapse runs of whitespace; trim
6. upper-case the first rune

Step 4 must strip the id prefix *after* step 2, because the corpus writes `@Author Ronan * * 1021000.js: relic room fail`.

- [ ] **Step 4: Build the override table**

Also in `describe.go`:

```go
// descriptionOverrides supplies a description for every reactor whose source
// comment cleans to nothing or to something uninformative. An entry is
// REQUIRED whenever the cleaner falls short — describe() returns an error
// rather than emit a guess, so this table is the one place human judgment
// enters the generated corpus. Review this, not the 1,749 files.
var descriptionOverrides = map[string]string{
	// ...
}
```

Populate it by running the cleaner over the real inventory and adding an entry for every id it rejects. At plan time, `grep -c '(no comment in source)' tier1-inventory.md` reports **15** sections with no comment at all, and an approximation of the cleaner leaves at least these ids empty or degenerate: `1022001`, `1032000`, `1202000` (cleans to the wrong id `1102000.js`), `1402000` (cleans to `93`), `200000`–`200009`, `2052001`, `2112015`, `2119004`, `2119005`, `2119006`, `2229009`, `2402006` (cleans to `HTPQ Box` — acceptable, but check), `6741001`, `6741015`, `6742014`, `6802000`, `6802001`, `8001000`. Expect roughly 40 entries.

Each override must state the reactor's **observable behaviour** (FR-15), derived from its converted operations — e.g. `"2119004": "Altar in El Nath - weakens the Snow Witch"`, `"1002008": "Henesys box - drops items"`. Do not write "Reactor 1002008".

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd tools/reactor-seed-gen && go build ./... && go vet ./... && go test ./... -v`
Expected: PASS. `TestDescribe_NoBoilerplateLeaks` covering all 159 is the one that matters — if it fails, it names the reactor and the leaked substring.

- [ ] **Step 6: Commit**

```bash
git add tools/reactor-seed-gen
git commit -m "feat(tools): derive reactor descriptions with a fail-closed override table"
```

---

## Task 11: `reactor-seed-gen` — emission, fan-out, and the CLI

### Files

- `tools/reactor-seed-gen/emit.go` — **new file**; marshalling and the eleven-directory fan-out
- `tools/reactor-seed-gen/emit_test.go` — **new file**
- `tools/reactor-seed-gen/main.go` — **new file**; flag parsing and the pipeline
- `tools/reactor-seed-gen/testdata/golden/reactor-2119000.json` — **new file**; the byte-equality golden
- `deploy/seed/gms/83_1/reactor-actions/reactors/reactor-9108000.json` — read-only; the real key ordering to match

Module root: `tools/reactor-seed-gen`

### Steps

- [ ] **Step 1: Write the golden file**

`testdata/golden/reactor-2119000.json`, byte-for-byte what the generator must produce for `2119000` — 2-space indent, alphabetical keys at every level, LF, one trailing newline:

```json
{
  "data": {
    "attributes": {
      "actRules": [],
      "description": "Tombstone in Forest of Dead Trees I",
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

If Task 10's cleaner yields a different description for `2119000`, update this golden to match the cleaner's actual output rather than forcing the cleaner — but the description must still contain `Tombstone in Forest of Dead Trees I`.

- [ ] **Step 2: Write the failing tests**

`emit_test.go`, package `main`.

- `TestEmit_GoldenBytes` — build the `2119000` `scriptDoc` by hand, call `marshalEnvelope`, and assert `bytes.Equal` with the golden file read from disk. On mismatch, print both as strings — a whitespace-only diff is otherwise invisible.
- `TestEmit_BareDropOmitsParams` — a `scriptDoc` whose single operation has `Params: nil` marshals to an operations entry containing `"type": "drop_items"` and NOT containing the substring `"params"`.
- `TestEmit_EmptyRulesAreArraysNotNull` — a `scriptDoc` with nil `HitRules` and nil `ActRules` marshals to `"actRules": []` and `"hitRules": []`, never `null`. (FR-19: `9018000`–`9018005` depend on this.)
- `TestEmit_ConditionOmitsEmptyStep` — a `reactor_state` condition marshals without a `step` key; a `pq_custom_data` condition with `Step: "stage"` marshals with it.
- `TestEmit_TrailingNewline` — output ends with exactly one `\n` and contains no `\r`.
- `TestFanOut_WritesElevenIdenticalCopies` — call `fanOut(tempDir, "2119000", golden)` and assert eleven files exist at the exact expected relative paths and all eleven have the same SHA-256.

- [ ] **Step 3: Run the tests and confirm they fail**

Run: `cd tools/reactor-seed-gen && go test ./... -run 'TestEmit|TestFanOut' -v`
Expected: FAIL — `marshalEnvelope` and `fanOut` undefined.

- [ ] **Step 4: Implement `emit.go`**

```go
package main

// seedDirs is the literal fan-out list. Keeping it a literal here (rather
// than globbing deploy/seed) is deliberate: adopting the shared catalog root
// later (design §9) is then a change to this one variable plus a git rm.
var seedDirs = []string{
	"gms/12_1", "gms/48_1", "gms/61_1", "gms/72_1", "gms/79_1",
	"gms/83_1", "gms/84_1", "gms/87_1", "gms/92_1", "gms/95_1",
	"jms/185_1",
}

// marshalEnvelope renders one script as its JSON:API seed file. Nested
// map[string]any (not structs) so encoding/json's key sorting reproduces the
// existing corpus's alphabetical ordering byte for byte.
func marshalEnvelope(d scriptDoc) ([]byte, error)

// fanOut writes the same bytes to all eleven tenant directories, so
// byte-identity is structural rather than something checked afterwards.
func fanOut(seedRoot, reactorId string, b []byte) error
```

`marshalEnvelope` builds `map[string]any` all the way down, uses `json.MarshalIndent(v, "", "  ")`, and appends `'\n'`. Rules and operations are `[]any` so an empty slice marshals as `[]`. An `opDoc` with `Params == nil` contributes only the `type` key; a `condDoc` with `Step == ""` contributes only `operator`, `type`, `value`.

`fanOut` writes to `filepath.Join(seedRoot, dir, "reactor-actions/reactors", "reactor-"+reactorId+".json")` for each of `seedDirs`, with `0o644`. It does not create the directories — all eleven already exist, and a missing one is a real error worth surfacing.

- [ ] **Step 5: Implement `main.go`**

```
usage: reactor-seed-gen [-inventory PATH] [-seed-root PATH] [-check]

  -inventory   default docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md
  -seed-root   default deploy/seed
  -check       regenerate in memory and diff against disk; write nothing;
               exit 1 listing every path that differs or is missing
```

Pipeline: read the inventory → `parseInventory` → for each `sourceScript`: `doc, err := convertScript(s)` (Task 9 — sets `ReactorId`, `HitRules`, `ActRules`), then `doc.Description, err = describe(s.Id, s.Comment)` (Task 10), then `marshalEnvelope(doc)` → `fanOut` (or, under `-check`, compare against the file on disk). Any error from any stage aborts the whole run with a non-zero exit and no partial write — collect nothing, write nothing, print the reactor id and the cause. Print a one-line summary on success: `reactor-seed-gen: 159 reactors × 11 directories = 1749 files written`.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `cd tools/reactor-seed-gen && go build ./... && go vet ./... && go test ./... -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add tools/reactor-seed-gen
git commit -m "feat(tools): emit reactor seed envelopes and fan out to eleven tenants"
```

---

## Task 12: Generate the 1,749 files

### Files

- `deploy/seed/gms/12_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/gms/48_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/gms/61_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/gms/72_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/gms/79_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/gms/83_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/gms/84_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/gms/87_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/gms/92_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/gms/95_1/reactor-actions/reactors/` — 159 new files
- `deploy/seed/jms/185_1/reactor-actions/reactors/` — 159 new files
- `tools/reactor-seed-gen/` — new module from Tasks 8-11; read-only here, this task only runs it

This task writes no source. Its whole content is one command and the assertions on its output.

### Steps

- [ ] **Step 1: Run the generator**

```bash
go run ./tools/reactor-seed-gen
```

Expected: `reactor-seed-gen: 159 reactors × 11 directories = 1749 files written`, exit 0.

If it aborts, it names a reactor id and an unmatched line. That is the fail-closed design working: extend the grammar in `convert.go` with a test first (Task 9's table), then re-run. Do not hand-write the file it could not generate.

- [ ] **Step 2: Assert the file counts**

Run: `for d in deploy/seed/gms/{12_1,48_1,61_1,72_1,79_1,83_1,84_1,87_1,92_1,95_1} deploy/seed/jms/185_1; do echo "$d $(ls "$d/reactor-actions/reactors" | wc -l)"; done`
Expected: `172` in every one of the eleven (13 pre-existing + 159 new).

- [ ] **Step 3: Assert `--check` is clean**

Run: `go run ./tools/reactor-seed-gen -check`
Expected: exit 0, no differences. This proves the on-disk corpus is exactly what the inventory produces.

- [ ] **Step 4: Run the lint over the full corpus**

Run: `./tools/reactor-seed-lint.sh`
Expected: exit 0 over 1,892 files (172 × 11). This is the FR-14/FR-15 gate: schema conformance, envelope well-formedness, non-empty description, and cross-tenant byte identity, all four asserted mechanically.

- [ ] **Step 5: Assert the specific acceptance criteria**

All five run from the worktree root. They read files that Step 1 has just created, so run them only after the generator has succeeded.

FR-18, the two unrolled loops:

Run: `grep -A3 '"count"' deploy/seed/gms/83_1/reactor-actions/reactors/reactor-2201001.json`
Expected: one `spawn_monster` carrying `"count": "3"` and `"monsterId": "9300007"`.

Run: `grep -c '"count": "6"' deploy/seed/gms/83_1/reactor-actions/reactors/reactor-2511001.json`
Expected: `2` — one per `spawn_monster` (`9300124` and `9300125`).

FR-19, the six empty-body flowers:

```bash
for i in 0 1 2 3 4 5; do
  python3 -c "import json;a=json.load(open('deploy/seed/gms/83_1/reactor-actions/reactors/reactor-901800$i.json'))['data']['attributes'];print('901800$i',a['hitRules'],a['actRules'],bool(a['description']))"
done
```

Expected: each line reads `901800<i> [] [] True`.

FR-17, the four guarded scripts emit the positive form:

Run: `grep -l '"operator": "!="' deploy/seed/gms/83_1/reactor-actions/reactors/*.json`
Expected: no output — every `reactor_state` condition in the corpus uses `=`.

- [ ] **Step 6: Confirm no legacy keys anywhere**

Run: `grep -rn 'minMeso\|maxMeso\|mesoRange\|"item"\|dropType' deploy/seed/*/*/reactor-actions/`
Expected: no output. This is FR-21's gate for Task 13.

- [ ] **Step 7: Commit**

```bash
git add deploy/seed
git commit -m "feat(seed): generate 159 Tier-1 reactor actions across eleven tenants"
```

---

## Task 13: Remove the legacy `minMeso`/`maxMeso` fallback

### Files

- `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` — lines 99-150; delete both fallback branches and the doc comment claiming backward compatibility
- `services/atlas-reactor-actions/atlas.com/reactor/script/executor_test.go` — **new file**
- `services/atlas-reactor-actions/atlas.com/reactor/saga/processor.go` — lines 14-16; read-only; `Processor` is the one-method interface the test fake implements
- `services/atlas-reactor-actions/atlas.com/reactor/script/evaluator_test.go` — lines 1-11; read-only; the package-`script` test imports and `test.NewNullLogger()` setup to copy
- `libs/atlas-script-core/operation/builder.go` — lines 14-45; read-only; `operation.NewBuilder().SetType(...).SetParams(...).Build()`

Module root for `go build`/`go test`: `services/atlas-reactor-actions/atlas.com/reactor`

**Gate first.** Do not start this task until `grep -rn 'minMeso\|maxMeso\|mesoRange\|"item"' deploy/seed/*/*/reactor-actions/` returns nothing (FR-21). Run it and record the empty result.

Patterns to copy: `services/atlas-reactor-actions/atlas.com/reactor/script/evaluator_test.go:1-11` (imports + null logger).

### Steps

- [ ] **Step 1: Write the failing test**

`executor_test.go`, package `script`. `OperationExecutor.sagaP` is an unexported field of an interface type, and the test is in the same package, so the test substitutes a fake without any new constructor (no `*_testhelpers.go`):

```go
type fakeSagaProcessor struct {
	created []saga.Saga
	err     error
}

func (f *fakeSagaProcessor) Create(s saga.Saga) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, s)
	return nil
}
```

`saga` here is `github.com/Chronicle20/atlas/libs/atlas-saga`; the field's type is `atlas-reactor-actions/saga`.`Processor`, whose sole method is `Create(sharedsaga.Saga) error`.

`TestExecuteDropItems_MesoParams` — table-driven. Each case builds an `operation.Model` via `operation.NewBuilder().SetType("drop_items").SetParams(params).Build()`, runs it through an executor with the fake, and asserts the single created saga step's `SpawnReactorDropsPayload` fields. Build the `ReactorContext` with a `field.Model` matching `evaluator_test.go`'s construction.

| case | params | want Meso | MesoChance | MesoMin | MesoMax | MinItems | DropType |
|---|---|---|---|---|---|---|---|
| all new params | `{meso:"true", mesoChance:"2", mesoMin:"8", mesoMax:"15", minItems:"1"}` | `true` | `2` | `8` | `15` | `1` | `drop` |
| no params | `{}` | `false` | `1` | `1` | `1` | `0` | `drop` |
| **legacy minMeso ignored** | `{meso:"true", minMeso:"2", maxMeso:"8"}` | `true` | `1` | `1` | `1` | `0` | `drop` |
| **legacy mixed with new** | `{meso:"true", mesoMin:"8", minMeso:"999"}` | `true` | `1` | `8` | `1` | `0` | `drop` |
| unparseable value falls back to default | `{meso:"true", mesoChance:"abc"}` | `true` | `1` | `1` | `1` | `0` | `drop` |

The third and fourth cases are the point of this task: after removal, `minMeso`/`maxMeso` must have **no effect at all**, so `mesoMin`/`mesoMax` stay at their defaults of `1`.

`TestExecuteSprayItems_SetsDropType` — a `spray_items` operation with `{meso:"true", mesoChance:"1", mesoMin:"50", mesoMax:"100", minItems:"15"}` produces one saga step with `DropType == "spray"` and `MesoMin == 50`, `MesoMax == 100`, `MinItems == 15`.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-reactor-actions/atlas.com/reactor && go test ./script/ -run 'TestExecuteDropItems_MesoParams|TestExecuteSprayItems' -v`
Expected: FAIL on the `legacy minMeso ignored` case — `MesoMin = 2, want 1` — because the fallback is still in place. The other cases pass.

- [ ] **Step 3: Delete the fallback**

In `executeDropItems`, the two `else if` branches go:

```go
	// New format: mesoMin, fallback to legacy minMeso
	if v, ok := params["mesoMin"]; ok {
		...
	} else if v, ok := params["minMeso"]; ok {   // DELETE this branch
		...
	}
```

becomes

```go
	if v, ok := params["mesoMin"]; ok {
		if parsed, err := strconv.ParseUint(v, 10, 32); err == nil {
			mesoMin = uint32(parsed)
		}
	}
```

and the same for `mesoMax`/`maxMeso`. Also replace the function's doc comment (`executor.go:99-100`):

```go
// executeDropItems handles reactor item drops via saga orchestration.
// Parameters map positionally from Cosmic's
// rm.dropItems(meso, mesoChance, minMeso, maxMeso, minItems) onto
// meso, mesoChance, mesoMin, mesoMax, minItems. dropType is injected by
// executeSprayItems and is never present in seed data.
```

and delete the `// Parse meso configuration with backward compatibility` comment.

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go vet ./... && go test ./script/ -v`
Expected: PASS, all cases, and no other test in the package regresses.

Run: `grep -n 'minMeso\|maxMeso\|backward compat' services/atlas-reactor-actions/atlas.com/reactor/script/executor.go`
Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reactor-actions/atlas.com/reactor/script/executor.go services/atlas-reactor-actions/atlas.com/reactor/script/executor_test.go
git commit -m "refactor(reactor-actions): drop the legacy minMeso/maxMeso fallback"
```

---

## Task 14: Sampled review and the full gate

### Files

- `docs/tasks/task-291-reactor-tier1-conversion/existing-seed-audit.md` — new file from Task 7; append the sampled-review section
- `docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md` — read-only; the source bodies to read against
- `deploy/seed/gms/83_1/reactor-actions/reactors/` — read-only; the generated files to read

### Steps

- [ ] **Step 1: Read one reactor per shape against its source**

PRD §8 gate 2 requires one converted reactor from each of the ten shapes in PRD §6, and the sample must include `reactor-2001`, one `reactor_state`-guarded script, **both** unrolled-loop scripts, and one empty-body script. Use exactly this sample — every id is verified present in `tier1-inventory.md`:

| shape (PRD §6) | reactor | source body |
|---|---|---|
| `drop_items` only | `1012000` | `rm.dropItems(true, 2, 20, 40);` |
| `spray_items` only | `1052001` | `rm.sprayItems(true, 1, 500, 1000, 15);` |
| `spawn_monster` only | `8001000` | `rm.spawnMonster(9400112, 1, 420, 160);` |
| `weaken_area_boss` only | `2619003` | `rm.weakenAreaBoss(6090004, "Rurumo has been poisoned. It may finally be defeatable!");` |
| empty `act()` | `9018000` | `function act() {\n}` |
| `reactor_state` + `weaken_area_boss` | `2119000` | guarded `hit()` |
| `update_pq_state` only | `2008006` | `rm.getEventInstance().setProperty("statusStg3", "0");` |
| `drop_items` + `update_pq_state` | `2002003` | `rm.dropItems();` + `eim.setProperty("statusStg7", "1")` |
| `update_pq_state` + `spawn_monster` | `2511000` | the `openedBoxes` increment + two spawns |
| `update_pq_state` + `spray_items` | `2512001` | the `openedChests` increment + spray |
| loop unroll (mandated) | `2201001` | `for (var i = 0; i < 3; i++) { rm.spawnMonster(9300007); }` |
| loop unroll (mandated) | `2511001` | `for (var i = 0; i < 6; i++) { rm.spawnMonster(9300124); rm.spawnMonster(9300125); }` |
| meso-shift regression (mandated) | `2001` | `rm.dropItems(true, 2, 8, 15, 1);` |

- [ ] **Step 2: Append the review to `existing-seed-audit.md`**

```markdown
## Sampled Conversion Review (PRD §8 gate 2)

One converted reactor per shape, read against its body in `tier1-inventory.md`.

| Shape | Reactor | Source body | Emitted rules | Verdict |
|---|---|---|---|---|
| ... one row per sampled reactor, with the ACTUAL emitted operations and params ... |
```

Each row states the operations and parameter values actually present in the generated file. A row that says "matches" without naming what matched is not a review.

- [ ] **Step 3: Run the full gate**

Run: `./tools/verify.sh 2>&1 | tee /tmp/verify-291.log; ./tools/gate-summary.sh /tmp/verify-291.log`
Expected: exit 0, flagless. Per CLAUDE.md, `--quick`/`--no-docker` do not count. The `reactor seed lint` step must appear as PASSED, not SKIPPED — if it is skipped, the `touched` pattern in Task 6 Step 3 is wrong.

- [ ] **Step 4: Confirm every acceptance criterion**

Walk PRD §10 top to bottom and record each box's evidence. The mechanical ones:

| Run | Expected |
|---|---|
| `grep -c 'update_pq_state\|hit_reactor\|broadcast_pq_message\|stage_clear_attempt' .claude/commands/convert-reactor.md` | non-zero (FR-1) |
| `grep -c 'mesoRange\|"item"\|scripts/reactors' .claude/commands/convert-reactor.md` | `0` (FR-3, FR-7) |
| `grep -c 'touchRules' services/atlas-reactor-actions/docs/reactor_script_schema.json` | non-zero (FR-8) |
| `grep -rn 'minMeso\|maxMeso\|mesoRange\|"item"' deploy/seed/*/*/reactor-actions/` | no output (FR-21) |
| `ls deploy/seed/jms/185_1/reactor-actions/reactors \| wc -l` | `172` (FR-14) |
| `go run ./tools/reactor-seed-gen -check` | exit 0 |
| `./tools/reactor-seed-lint.sh` | exit 0 |

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-291-reactor-tier1-conversion/existing-seed-audit.md
git commit -m "docs(task-291): record the sampled conversion review"
```

---

## Known limitations to state at PR time

These are true of the merged branch and are not defects introduced by it. State them in the PR body rather than letting QA find them:

- **`weaken_area_boss` only logs today** — `executor.go:259-305` writes a debug line and returns nil, and its own comment says the saga action for boss weakening does not exist yet. This branch adds nothing there and removes nothing there. Seven Tier-1 reactors emit the operation and will log rather than weaken anything. The PRD §3 user story "Lich … defeatable as designed" is not satisfied by this task alone — the seed data becomes correct ahead of the engine. `move_environment` and `kill_all_monsters` are stubs on the same footing.
- **`update_pq_state` fails when the character is not in a PQ instance** (`executor.go:351-358`), unlike the `pq_custom_data` *condition*, which downgrades the same lookup failure to "condition not met" (`evaluator.go:101-104`). The asymmetry is pre-existing; the four new Tier-1 call sites make it reachable in normal play. PRD §9.3 flags it for a follow-up decision.
- **`9018000`–`9018005` are seeded as explicitly empty** (FR-19). If Cosmic has a default behaviour for a script defining no operations, it is not visible in the script.
- **`tier1-inventory.md` is the generator's only input** and lives in the task folder. After merge, re-running `reactor-seed-gen` requires that file. The durable gate is `reactor-seed-lint`, which depends only on the corpus and the schema; `-check` is a task-lifetime tool.
- **The eleven-copy fan-out is deferred technical debt.** `libs/atlas-seeder`'s `NewFilesystemCatalogSourceWithShared` (`catalog.go:47`) would reduce 1,749 files to 159. PRD §9.1 defers it; `emit.go`'s `seedDirs` literal is the single place that changes when it is adopted. Worth revisiting before Tier 2 adds another 65 reactors at 11× cost.
