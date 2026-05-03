# `start_quest` Portal-Actions Operation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `start_quest` operation to atlas-portal-actions so portal scripts can dispatch quest starts as independent sagas, then validate by converting `foxLaidy_map.js` end-to-end.

**Architecture:** Each portal operation creates its own single-step saga via `e.sagaP.Create(s)`. `start_quest` follows that pattern — emits `saga.StartQuest` with `saga.StartQuestPayload` and returns. No multi-step sequencing; no `action.PendingAction` registration. Mirrors `executeApplyConsumableEffect` for parameter parsing and saga emission.

**Tech Stack:** Go 1.x, atlas-saga library (`saga.NewBuilder`, `saga.StartQuest`, `saga.StartQuestPayload`), atlas-script-core (`operation.Model`), JSON Schema (Draft 7) for the script-format contract.

---

## Context

- Design: `docs/tasks/task-048-portal-actions-start-quest/design.md`
- NPC reference impl: `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:1755`
- Saga payload definition: `libs/atlas-saga/payloads.go:281`
- Existing similar executor (template): `services/atlas-portal-actions/atlas.com/portal/script/executor.go` → `executeApplyConsumableEffect` (~line 474)
- Driving portal script: `services/atlas-npc-conversations/tmp/portal/foxLaidy_map.js`
- Working branch: `task-048-portal-actions-start-quest` (already checked out)

## File Manifest

- **Modify** `services/atlas-portal-actions/docs/portal_script_schema.json` — add enum entry + conditional params block.
- **Modify** `services/atlas-portal-actions/atlas.com/portal/script/executor.go` — add switch case + new method.
- **Modify** `.claude/commands/convert-portal.md` — supported-ops table + new pattern example.
- **Create** `services/atlas-portal-actions/scripts/portals/portal_foxLaidy_map.json` — converted portal script (validation artifact).

## Verification commands (used throughout)

- Build: `cd services/atlas-portal-actions && go build ./...`
- Test: `cd services/atlas-portal-actions && go test ./...`
- JSON well-formedness: `python3 -m json.tool <path>` (returns non-zero on parse error)

---

### Task 1: Add `start_quest` to the schema

**Files:**
- Modify: `services/atlas-portal-actions/docs/portal_script_schema.json`

- [ ] **Step 1: Open the file and locate the operation `type` enum**

The enum is in the `operation` definition's `type` property, lines 157-171. It currently lists 13 entries; the last one is `"cancel_consumable_effect"`. The order is **not** alphabetical — append at the end.

- [ ] **Step 2: Add `"start_quest"` to the enum**

Edit the `enum` array so it ends with `"cancel_consumable_effect", "start_quest"`. The full enum becomes:

```json
"enum": [
  "play_portal_sound",
  "warp",
  "drop_message",
  "block_portal",
  "show_info",
  "show_hint",
  "create_skill",
  "update_skill",
  "start_instance_transport",
  "save_location",
  "warp_to_saved_location",
  "apply_consumable_effect",
  "cancel_consumable_effect",
  "start_quest"
]
```

- [ ] **Step 3: Append the conditional params block**

The `operation` definition has an `allOf` array of `if/then` blocks (one per op that has param requirements). Find the last block in `allOf` (the `cancel_consumable_effect` one, ending at the file's closing `]` for `allOf`). Append a new sibling block immediately before that closing `]`:

```json
,
{
  "if": {
    "properties": { "type": { "const": "start_quest" } }
  },
  "then": {
    "properties": {
      "params": {
        "type": "object",
        "required": ["questId"],
        "properties": {
          "questId": {
            "type": "string",
            "description": "The quest ID to start"
          },
          "npcId": {
            "type": "string",
            "description": "Originating NPC ID (default: 0 — portals have no NPC)"
          }
        }
      }
    }
  }
}
```

- [ ] **Step 4: Verify JSON is well-formed**

Run: `python3 -m json.tool services/atlas-portal-actions/docs/portal_script_schema.json > /dev/null`
Expected: exit code 0, no output.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-portal-actions/docs/portal_script_schema.json
git commit -m "feat(atlas-portal-actions): add start_quest to portal script schema

Adds the start_quest operation to the script-format contract so
portal scripts can dispatch quest starts. Required param: questId.
Optional param: npcId (defaults to 0 — portals have no NPC).

Refs: docs/tasks/task-048-portal-actions-start-quest/design.md"
```

---

### Task 2: Implement `executeStartQuest`

**Files:**
- Modify: `services/atlas-portal-actions/atlas.com/portal/script/executor.go`

- [ ] **Step 1: Add the dispatch case**

Open `executor.go`. Find the `switch op.Type()` block in `ExecuteOperation` (around line 41). The last case before `default` is `case "warp_to_saved_location"` (around line 75-76). Add a new case immediately after it, before `default:`:

```go
case "start_quest":
    return e.executeStartQuest(f, characterId, op)
```

- [ ] **Step 2: Append the new method to the file**

Add this method at the bottom of the file (after `executeCancelConsumableEffect`, which is the last method around line 619):

```go
// executeStartQuest dispatches a saga to start a quest for the character.
// questId is required. npcId is optional and defaults to 0 since portals have no NPC context.
func (e *OperationExecutor) executeStartQuest(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	questIdStr, ok := params["questId"]
	if !ok {
		return fmt.Errorf("start_quest operation missing questId parameter")
	}
	questId, err := strconv.ParseUint(questIdStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid questId [%s]: %w", questIdStr, err)
	}

	var npcId uint64 = 0
	if npcIdStr, hasNpcId := params["npcId"]; hasNpcId {
		npcId, err = strconv.ParseUint(npcIdStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid npcId [%s]: %w", npcIdStr, err)
		}
	}

	e.l.Debugf("Starting quest [%d] for character [%d] (npcId=%d)", questId, characterId, npcId)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("portal-action-start-quest").
		AddStep(
			fmt.Sprintf("start-quest-%d-%d", characterId, questId),
			saga.Pending,
			saga.StartQuest,
			saga.StartQuestPayload{
				CharacterId: characterId,
				WorldId:     f.WorldId(),
				QuestId:     uint32(questId),
				NpcId:       uint32(npcId),
			},
		).Build()

	return e.sagaP.Create(s)
}
```

No new imports needed — `saga`, `field`, `strconv`, `fmt`, and `operation` are all already imported (verify by checking the existing import block around lines 3-18).

- [ ] **Step 3: Build the package**

Run: `cd services/atlas-portal-actions && go build ./...`
Expected: exit code 0, no output. If you see "undefined: saga.StartQuest" or "undefined: saga.StartQuestPayload", check that `libs/atlas-saga/model.go` defines `StartQuest` (line 81) and `libs/atlas-saga/payloads.go` defines `StartQuestPayload` (line 282). Both should resolve via the existing `atlas-saga` import in executor.go.

- [ ] **Step 4: Run existing tests**

Run: `cd services/atlas-portal-actions && go test ./...`
Expected: PASS for all packages. The schema change in Task 1 should not break `entity_test.go`, `rest_test.go`, `seed_status_test.go`, `processor_test.go`, or `builder_test.go`. If any test fails, do not proceed — read the failure message and fix before continuing.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-portal-actions/atlas.com/portal/script/executor.go
git commit -m "feat(atlas-portal-actions): implement start_quest operation executor

Adds executeStartQuest, modeled on executeApplyConsumableEffect. Emits
an independent saga.StartQuest step with saga.StartQuestPayload. No
PendingAction registration since there is no failure UX to surface.

Refs: docs/tasks/task-048-portal-actions-start-quest/design.md"
```

---

### Task 3: Update the convert-portal slash command doc

**Files:**
- Modify: `.claude/commands/convert-portal.md`

- [ ] **Step 1: Add `start_quest` to the supported operations table**

Open `.claude/commands/convert-portal.md`. Find the "Operation Types" table (the table that lists `play_portal_sound`, `warp`, etc., with columns "Type | When to Use | Params"). Add this row at the bottom of that table:

```markdown
| `start_quest` | `pi.forceStartQuest(questId)` | `questId` (required), `npcId` (optional, defaults to 0) |
```

- [ ] **Step 2: Add `forceStartQuest` to the supported list in the analysis section**

In the section titled "Supported Operations:" (a bulleted list under "### 1. Analyze the JavaScript Script"), add this bullet at the bottom:

```markdown
- `pi.forceStartQuest(questId)` → `start_quest` operation (params: `questId`, optional `npcId`)
```

- [ ] **Step 3: Confirm `forceStartQuest` is not in the unsupported list**

Search the file for `forceStartQuest`. The only occurrences after this edit should be:
1. The supported-operations table row (Step 1)
2. The analysis-section bullet (Step 2)
3. Any new pattern example (Step 4)

If the file currently lists `forceStartQuest` in the "NOT YET SUPPORTED" section, remove that mention. If it does not, no change needed.

Run: `grep -n forceStartQuest .claude/commands/convert-portal.md`
Expected: 2-3 lines, all in supported sections.

- [ ] **Step 4: Add a new conversion pattern example**

Find the section with conversion patterns (titled "### 3. Handle Simple Patterns" with Patterns A through D). Append a new pattern after Pattern D:

````markdown
**Pattern E: Quest-Gated Quest Start**
```javascript
function enter(pi) {
    if (!(pi.isQuestStarted(3647) && pi.haveItem(4031793, 1))) {
        pi.playPortalSound();
        pi.warp(222010200, "east00");
    } else {
        if (!pi.isQuestStarted(23647)) {
            pi.forceStartQuest(23647);
        }
        pi.playPortalSound();
        pi.warp(922220000, "east00");
    }
    return true;
}
```
→ Two rules. Inner `if (!pi.isQuestStarted(23647))` collapses away because `start_quest` is dispatched as an independent saga and atlas-quests handles "already started" idempotently. The matching rule fires `start_quest` plus the warp; the default rule warps elsewhere.

Resulting rules (first match wins):
- Rule 1: `quest_status` quest 3647 = started AND `item` 4031793 owned → `play_portal_sound`, `start_quest` (questId=23647), `warp` to 922220000 portalName "east00"
- Rule 2: default (empty conditions) → `play_portal_sound`, `warp` to 222010200 portalName "east00"
````

- [ ] **Step 5: Commit**

```bash
git add .claude/commands/convert-portal.md
git commit -m "docs(convert-portal): document start_quest support

Adds start_quest to the supported operations table, adds an analysis
bullet for pi.forceStartQuest, and adds Pattern E (quest-gated quest
start) as a worked example mirroring foxLaidy_map.js.

Refs: docs/tasks/task-048-portal-actions-start-quest/design.md"
```

---

### Task 4: Convert `foxLaidy_map.js` end-to-end

**Files:**
- Create: `services/atlas-portal-actions/scripts/portals/portal_foxLaidy_map.json`

- [ ] **Step 1: Re-read the source JS to confirm semantics**

Read `services/atlas-npc-conversations/tmp/portal/foxLaidy_map.js`. Confirm it matches the Pattern E example from Task 3.

- [ ] **Step 2: Look up destination map context**

Run: `grep -rln "922220000" services/atlas-npc-conversations/tmp/npcs/ | head -3`
Expected: at least `2071012.js` listed. Read it and confirm the file header comment identifies map 922220000 as `"Hidden Street : Gloomy Forest"`. This is the success-branch destination.

Map 222010200 has no other references in the codebase. From naming convention (222 prefix is Korean Folk Town in MapleStory v83), it is the Korean Folk Town hidden-entry map and is also the source map where the `foxLaidy_map` portal is placed (the failure branch loops back to the same map).

- [ ] **Step 3: Write the converted script**

Create `services/atlas-portal-actions/scripts/portals/portal_foxLaidy_map.json` with this content:

```json
{
  "portalId": "foxLaidy_map",
  "mapId": 222010200,
  "description": "Korean Folk Town hidden street entry — gates entry to Gloomy Forest (922220000) on having quest 3647 started + item 4031793 (Soft Silver Fur). On success, force-starts quest 23647 (Familiar Lady follow-up) and warps; otherwise loops back to the same map.",
  "rules": [
    {
      "id": "has_silver_fur_and_quest_started",
      "conditions": [
        {
          "type": "quest_status",
          "operator": "=",
          "value": "1",
          "referenceId": "3647"
        },
        {
          "type": "item",
          "operator": ">=",
          "value": "1",
          "referenceId": "4031793"
        }
      ],
      "onMatch": {
        "allow": true,
        "operations": [
          { "type": "play_portal_sound" },
          { "type": "start_quest", "params": { "questId": "23647" } },
          { "type": "warp", "params": { "mapId": "922220000", "portalName": "east00" } }
        ]
      }
    },
    {
      "id": "default_loop_back",
      "conditions": [],
      "onMatch": {
        "allow": true,
        "operations": [
          { "type": "play_portal_sound" },
          { "type": "warp", "params": { "mapId": "222010200", "portalName": "east00" } }
        ]
      }
    }
  ]
}
```

Notes baked into the structure:
- `quest_status` value `"1"` = STARTED (matches `portal_aranTutorOut1.json` precedent at `services/atlas-portal-actions/scripts/portals/portal_aranTutorOut1.json`).
- `item` condition uses operator `">="` value `"1"` (player must hold ≥1 of the item) per the portal-actions item-check convention. The `referenceId` is the item ID 4031793 (Soft Silver Fur).
- The default rule has `allow: true` (matching JS `return true`) and warps back to 222010200, mirroring the JS failure branch.
- The inner JS check `if (!pi.isQuestStarted(23647))` is intentionally collapsed: `start_quest` is dispatched as an independent saga and atlas-quests is responsible for handling "already started" idempotently (see design §3 architecture decision).

- [ ] **Step 4: Verify JSON is well-formed**

Run: `python3 -m json.tool services/atlas-portal-actions/scripts/portals/portal_foxLaidy_map.json > /dev/null`
Expected: exit code 0, no output.

- [ ] **Step 5: Verify the loader accepts the script**

Write a one-off Go program to drive `script.LoadPortalScriptFiles` against the local scripts directory. Save it as `services/atlas-portal-actions/cmd/load-check/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"atlas-portal-actions/script"
)

func main() {
	os.Setenv("PORTAL_SCRIPTS_DIR", "scripts/portals")
	scripts, errs := script.LoadPortalScriptFiles()
	fmt.Printf("Loaded %d scripts; %d errors\n", len(scripts), len(errs))
	for _, e := range errs {
		fmt.Println("ERROR:", e)
	}
	for _, s := range scripts {
		if s.PortalId() == "foxLaidy_map" {
			fmt.Println("FOUND foxLaidy_map: rules=", len(s.Rules()))
		}
	}
	if len(errs) > 0 {
		os.Exit(1)
	}
}
```

Run from the service root:

```bash
cd services/atlas-portal-actions
go run ./cmd/load-check
```

Expected: `Loaded N scripts; 0 errors` and a line `FOUND foxLaidy_map: rules= 2`.

If you see an error specifically about `portal_foxLaidy_map.json`, fix the JSON and re-run.

If the script defines a getter other than `PortalId()` / `Rules()`, adjust the load-check `main.go` accordingly — read `services/atlas-portal-actions/atlas.com/portal/script/model.go` to confirm getter names. **Do not** modify the model just to make this verification work.

- [ ] **Step 6: Remove the verification harness**

Run:

```bash
rm -rf services/atlas-portal-actions/cmd/load-check
```

The harness is verification scaffolding only; it must not be committed.

- [ ] **Step 7: Run full build and test once more**

Run: `cd services/atlas-portal-actions && go build ./... && go test ./...`
Expected: build clean, all tests pass.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-portal-actions/scripts/portals/portal_foxLaidy_map.json
git commit -m "feat(atlas-portal-actions): convert foxLaidy_map portal script

Converts services/atlas-npc-conversations/tmp/portal/foxLaidy_map.js
to JSON rules format. Validates the new start_quest operation
end-to-end. Two rules: gated path (quest 3647 started + Soft Silver
Fur owned) starts quest 23647 and warps to Gloomy Forest; default
loops back to map 222010200.

Refs: docs/tasks/task-048-portal-actions-start-quest/design.md"
```

---

### Task 5: Final verification

- [ ] **Step 1: Confirm clean working tree and expected branch**

Run: `git status` and `git rev-parse --abbrev-ref HEAD`
Expected: `working tree clean`; current branch `task-048-portal-actions-start-quest`.

- [ ] **Step 2: Confirm commit history**

Run: `git log --oneline main..HEAD`
Expected: 5 commits in this order (oldest at the bottom):
1. `feat(atlas-portal-actions): convert foxLaidy_map portal script`
2. `docs(convert-portal): document start_quest support`
3. `feat(atlas-portal-actions): implement start_quest operation executor`
4. `feat(atlas-portal-actions): add start_quest to portal script schema`
5. `docs(task-048): design for start_quest portal-actions operation`

The design commit (5) was made at the end of brainstorming, before plan execution. Plus a `plan.md` commit will land before Task 1 runs (handled by the executing skill, not this plan).

- [ ] **Step 3: Run the service-wide sanity checks**

```bash
cd services/atlas-portal-actions
go build ./...
go test ./...
```

Expected: both pass.

- [ ] **Step 4: Spot-check the schema changes**

```bash
grep -c '"start_quest"' services/atlas-portal-actions/docs/portal_script_schema.json
```

Expected: `2` (one in the enum, one in the conditional `if` block).

- [ ] **Step 5: Spot-check the executor change**

```bash
grep -n 'executeStartQuest\|"start_quest"' services/atlas-portal-actions/atlas.com/portal/script/executor.go
```

Expected: 3 hits — the case label in the dispatch, the call to the method, and the method definition.

- [ ] **Step 6: Spot-check the converted script**

```bash
grep -c '"start_quest"' services/atlas-portal-actions/scripts/portals/portal_foxLaidy_map.json
```

Expected: `1`.

---

## Acceptance Summary

- `services/atlas-portal-actions/docs/portal_script_schema.json` — `start_quest` in enum + conditional params block.
- `services/atlas-portal-actions/atlas.com/portal/script/executor.go` — `case "start_quest"` dispatch + `executeStartQuest` method.
- `.claude/commands/convert-portal.md` — supported-ops table updated, Pattern E added.
- `services/atlas-portal-actions/scripts/portals/portal_foxLaidy_map.json` — converted script loads through `LoadPortalScriptFiles` without errors.
- `go build ./...` and `go test ./...` pass in `services/atlas-portal-actions`.
- No new unit tests for the executor (per design §3 — symmetry with the 12 existing executors).
