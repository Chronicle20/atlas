# Plan Audit (Plan Adherence) — task-229-summon-bag-town-scroll-opcodes

**Plan Path:** docs/tasks/task-229-summon-bag-town-scroll-opcodes/plan.md
**Audit Date:** 2026-08-14
**Branch:** task-229-summon-bag-town-scroll-opcodes
**Base Branch:** main (merge-base 314ff8ad0)
**HEAD:** 481f27e8b

## Executive Summary

All 16 tasks were implemented and are substantively verifiable in the tree: both `USE_SUMMON_BAG` and `USE_RETURN_SCROLL` read `verified` on all ten matrix columns (`docs/packets/audits/status.json`), `USE_ITEM`/`USE_UPGRADE_SCROLL`/`PET_FOOD` read `verified` on `gms_v92`, all template bindings match the plan's tables exactly, and every registry gains the expected `packet:` link. `go build`/`go vet`/`go test -race` are clean in both `libs/atlas-packet` and `tools/packet-audit`, and every CI gate (`matrix --check`, `fname-doc --check`, `operations --check`, `dispatcher-lint`, `doc-freshness --check`, `gate-check --check`) plus every repo guard (opcode-order, duplicate-binding, movement-types, redis-key, goroutine) exits 0. The four pre-disclosed deviations (no generated audit reports, the gms_72 `fname` correction, the gms_v48 template rebind/expansion, and the `TestFoodRoundTrip` verifies-name substitution) are all confirmed present in the diff and are consistently reflected downstream (registry notes, `candidatesFromFName` test cases, matrix state) — none left an inconsistency. The only hygiene gap is that plan.md's 54 step-level checkboxes were never ticked to `[x]` despite the work being done; `tools/lint.sh --check` also fails on an unrelated environment issue (node v24 vs required v22), not a code defect.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Audit-only wrapper codecs (`SummonBagItemUse`, `ReturnScrollItemUse`) | DONE | `libs/atlas-packet/inventory/serverbound/summon_bag_item_use.go`, `return_scroll_item_use.go` created, each embeds `ItemUse`, carries `packet-audit:fname` marker and matches plan Step 3 exactly. Test files exist with round-trip + shared-body tests. `go test -race -count=1 ./inventory/serverbound/...` PASS. Commit `3efeafcd7`. |
| 2 | Link registry fnames to wrapper codecs (`candidatesFromFName`) | DONE | `tools/packet-audit/cmd/run.go:2211-2242` (approx) adds cases for `SendMobSummonItemUseRequest`, `sub_955499`, `SendPortalScrollUseRequest`, `SendReturnScrollUseRequest`, `sub_841AA5` — plus an unplanned but consistent extra case `sub_904154` (see Deviation 2). `TestCandidatesItemUseFamilyWrappers` in `disambiguation_test.go:188` covers all cases including the regression guard for the potion sender. `go test ./cmd/...` PASS. Commit `38582aeeb`. |
| 3 | Declare `packet:` link in the op registry (9 files) | DONE | `grep -c` confirms exactly 2 hits (`InventorySummonBagItemUse`/`InventoryReturnScrollItemUse`) in each of `gms_v61/72/79/83/84/87/92/95.yaml` and `jms_v185.yaml`. `gms_v48.yaml` also carries both (added by Task 15). Commit `88f91a56b`. |
| 4 | Bind missing item-use handlers in seed templates | DONE | Diffs for `template_gms_87_1.json`, `_92_`, `_95_`, `template_jms_185_1.json` match the plan's opcode/handler/fname tables exactly (verified via `git diff main...HEAD` grep of opCode/handler/fname triples). `gms_92` gained all 5 entries (`0x4F/0x52/0x53/0x5C/0x5D`) including `PetFoodHandle` (D3). `template_gms_48_1.json` deliberately untouched in this task per Step 5 (later modified only by Task 15). `tools/template-opcode-order-guard.sh`/`-duplicate-binding-guard.sh`/`-movement-types-guard.sh` all exit 0. Commit `b19448c9b`. |
| 5 | Promote the 3 v92 item-use rows needing only a fixture | DONE | `status.json`: `USE_ITEM`, `USE_UPGRADE_SCROLL`, `PET_FOOD` all read `verified` on `gms_v92`. Evidence file `docs/packets/evidence/gms_v92/pet.serverbound.PetFood.yaml` present, `verifies: food_test.go#TestFoodRoundTrip` (see Deviation 4 — plan named a nonexistent `TestPetFoodRoundTrip`, correctly substituted). Commit `6da8c2802`. |
| 6 | gms_v83 (reference column) per-version verify | DONE | `status.json` both ops `verified` @ `gms_v83`, opcode 75/85 matching plan. `summon_bag_item_use_test.go` marker `ida=0xa0977b`, `return_scroll_item_use_test.go` marker `ida=0xa1e1ce`. Registry `gms_v83.yaml` carries both `packet:` links. Commit `29f73aa9a`. |
| 7 | gms_v61 per-version verify | DONE | `status.json` verified @ `gms_v61` opcode 70/78. Markers `ida=0x831c83` (SB) / `ida=0x841aa5` (RS). Commit `227763bae`. |
| 8 | gms_v84 per-version verify | DONE | `status.json` verified @ `gms_v84` opcode 75/85. Markers `ida=0xa53b5d` / `ida=0xa6946d`. Commit `bd0761a6d`. |
| 9 | gms_v87 per-version verify | DONE | `status.json` verified @ `gms_v87` opcode 78/88. Markers `ida=0xa9f027` / `ida=0xab5507`. Commit `5436f7c6a`. |
| 10 | gms_v92 per-version verify | DONE | `status.json` verified @ `gms_v92` opcode 82/92. Markers `ida=0x9b3b80` / `ida=0x9d15c0`. Commit `1c480ba1b`. |
| 11 | gms_v95 per-version verify | DONE | `status.json` verified @ `gms_v95` opcode 81/92. Markers `ida=0x9de580` / `ida=0x9fca70`. Commit `9221ea617`. |
| 12 | jms_v185 per-version verify | DONE | `status.json` verified @ `jms_v185` opcode 67/77. Markers `ida=0xaee405` / `ida=0xb04d24`. Export spliced into `docs/packets/ida-exports/gms_jms_185.json` (correct path per Global Constraints). Commit `1421cd31a`. |
| 13 | gms_v72 per-version verify | DONE | `status.json` verified @ `gms_v72` opcode 74/84. Markers `ida=0x904154` (SB, matches the corrected fname `sub_904154`, see Deviation 2) / `ida=0x917221` (RS). Commit `6e34fcfdf`. |
| 14 | gms_v79 per-version verify | DONE | `status.json` verified @ `gms_v79` opcode 73/83. Markers `ida=0x955499` / `ida=0x968c52`. Commit `6b6d65eb6`. |
| 15 | Resolve gms_v48 | DONE | `status.json`: both ops `verified` @ `gms_v48` (not `n-a` — the sender was found, Step 3a branch taken, not 3b). Registry `gms_v48.yaml` gained `USE_SUMMON_BAG` (opcode 59, `sub_70DDAA`-derived name `CWvsContext::SendMobSummonItemUseRequest`) and `USE_RETURN_SCROLL` (opcode 65, `CWvsContext::SendPortalScrollUseRequest`), plus a correction of the pre-existing mislabeled `USE_ITEM` entry (opcode 65→56). Template gained `0x38` (`CharacterItemUseHandle`), `fname` backfill on `0x3B`, and `0x41` rebind from `CharacterItemUseHandle`/`sub_719DD9` to `CharacterItemUseTownScrollHandle`/`SendPortalScrollUseRequest` (Deviation 3, judged justified — see below). Commit `99d446c8b`. |
| 16 | Full verification and PR preparation | DONE | `matrix --check` exits 0 at HEAD (verified independently, see Build & Test Results). All CI gates and guards independently re-run and pass (see below). `docs/packets/audits/STATUS.md`/`status.json` regenerated in the final commit `481f27e8b` (toolSha refresh after the gms_v48 commit, matching the documented toolSha-reads-git-HEAD behavior). Templates/`item_use.go`/`atlas-channel`/`atlas-consumables` no-regression assertions independently re-verified (see below). This audit itself is part of Step 5's parallel review-agent dispatch. |

**Completion Rate:** 16/16 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All 16 tasks show concrete evidence of completion in the git history, the matrix, the registries, and the templates.

## Deliberate Deviations (assessed, not penalized)

1. **No generated audit reports.** Confirmed: `docs/packets/audits/gms_v83/` and `docs/packets/audits/gms_v48/` contain no `InventorySummonBagItemUse.{json,md}` / `InventoryReturnScrollItemUse.{json,md}` files, though evidence records and verify markers exist for all 10 versions × 2 ops (checked exhaustively). This is exactly the condition the plan's own "Design deviations recorded during planning" §1 anticipates (`grade.go:206-215` promotes via registry `packet:` + fresh evidence with no report). Both rows independently confirm `verified` on all 10 columns via `matrix`, so the deviation did not leave a downstream inconsistency — it is a report-generation shortfall, not a promotion shortfall. Judged **justified**.
2. **`template_gms_72_1.json` fname correction.** Confirmed via `git diff main...HEAD`: the summon-bag entry's `fname` changed from `sub_955499` to `sub_904154`. This is a byte change to a template the Global Constraints table says must stay byte-identical. However: (a) the old value was a genuine defect — `sub_955499` is v79's dummy fname, not v72's; (b) the correction is consistently propagated — `run.go`'s switch gained a `sub_904154` case, `disambiguation_test.go` gained a matching test case, the registry note documents the correction, and the verify marker in `summon_bag_item_use_test.go` uses `ida=0x904154` matching the corrected fname's address. No dangling reference to the old value remains anywhere in the tree (`grep -r sub_955499` still matches gms_v79 legitimately, not gms_v72). Judged **justified and internally consistent**.
3. **Task 15 expanded beyond its stated brief on gms_v48.** Confirmed via `git diff main...HEAD -- template_gms_48_1.json`: added a new `0x38`/`CharacterItemUseHandle`/`SendStatChangeItemUseRequest` entry, backfilled the `0x3B` summon-bag `fname`, and rebound `0x41` from `CharacterItemUseHandle`/`sub_719DD9` to `CharacterItemUseTownScrollHandle`/`SendPortalScrollUseRequest`. This directly contradicts the plan's own Global Constraints opcode table (line 36: `USE_ITEM 65 0x41`), which the IDB decompile proved wrong — the registry's `USE_ITEM` entry was independently corrected to opcode 56 (`0x38`), and its lengthy `note:` documents the full re-decompile (categories, gates, structural cross-check against v61/v83 `USE_ITEM`) that motivated the correction. The template, registry, and test-marker addresses are all mutually consistent post-correction (opcode 56↔`0x38`↔`CharacterItemUseHandle`; opcode 65↔`0x41`↔`CharacterItemUseTownScrollHandle`). Judged **justified** — the plan's own no-guessing rule (Task 15 Step 1: "Do NOT reason from opcode position … only the IDB decides") anticipates exactly this kind of correction, and CLAUDE.md's grounding rule requires acting on it rather than shipping a known-wrong binding.
4. **Task 5's `--verifies` used `TestFoodRoundTrip`.** Confirmed: `libs/atlas-packet/pet/serverbound/food_test.go:15` defines `func TestFoodRoundTrip`, not `TestPetFoodRoundTrip`. The evidence record `docs/packets/evidence/gms_v92/pet.serverbound.PetFood.yaml` correctly references `food_test.go#TestFoodRoundTrip`. Judged **justified** — matches plan Step 2's own escape hatch ("If a `--verifies` test name does not exist, read the file and use the real one").

## Build & Test Results

| Module | Build | Vet | Tests (`-race -count=1`) | Notes |
|--------|-------|-----|---------------------------|-------|
| `libs/atlas-packet` | PASS | PASS | PASS (all packages; `inventory/serverbound` and `pet/serverbound` re-run uncached) | |
| `tools/packet-audit` | PASS | PASS | PASS (all packages, `-count=1`) | |

| Gate / Guard | Result | Notes |
|---|---|---|
| `go run ./tools/packet-audit matrix --check` | EXIT=0 | Only pre-existing `n-a` evidence notes printed (`CASHSHOP_CASH_ITEM_GACHAPON_RESULT`, `USE_TELEPORT_ROCK` — unrelated to this task) |
| `go run ./tools/packet-audit fname-doc --check` | EXIT=0 | |
| `go run ./tools/packet-audit operations --check` | EXIT=0 | |
| `go run ./tools/packet-audit dispatcher-lint` | EXIT=0 | |
| `go run ./tools/packet-audit doc-freshness --check` | EXIT=0 | |
| `go run ./tools/packet-audit gate-check --check` | EXIT=0 | |
| `tools/template-opcode-order-guard.sh` | EXIT=0 | "22 template arrays in ascending opcode order" |
| `tools/template-duplicate-binding-guard.sh` | EXIT=0 | "22 template arrays carry no duplicate (name, opCode) binding" |
| `tools/template-movement-types-guard.sh` | EXIT=0 | |
| `tools/redis-key-guard.sh` | EXIT=0 | |
| `tools/goroutine-guard.sh` | EXIT=0 | |
| `tools/lint.sh --check` | reports FAIL on `ui:node-version` | Environment-only failure (node v24 present, v22 required) — a documented pre-existing false-fail unrelated to this branch's Go/JSON changes (see project memory: lint.sh --check false-fails without nvm). All Go-lint targets reported "0 issues". |

No service `go.mod` was touched (`git diff --name-only main...HEAD \| grep go.mod` → empty), so `docker buildx bake` is correctly not required.

## No-Regression Assertions (re-verified independently)

- `libs/atlas-packet/inventory/serverbound/item_use.go`: **no diff** vs `main`.
- `services/atlas-channel/**`, `services/atlas-consumables/**`: **no diff** vs `main`.
- Templates touched: exactly `template_gms_48_1.json`, `_72_1.json` (1-line fname fix), `_87_1.json`, `_92_1.json`, `_95_1.json`, `template_jms_185_1.json`. `_61_`, `_79_`, `_83_`, `_84_` are byte-identical to `main`.
- All 10 ida-export files show **only additive** diffs (2 spliced function entries each, ~42-44 lines, zero deletions) — consistent with the "surgical splice only" Global Constraint.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the parallel backend-guidelines review's own sign-off, and resolution of the minor hygiene item below)

## Action Items

1. (Minor, non-blocking) `docs/tasks/task-229-summon-bag-town-scroll-opcodes/plan.md` still shows all 54 step-level checkboxes as `- [ ]` despite every task being demonstrably complete. Consider a final commit ticking them to `- [x]` for future readers, though this has no functional effect on the branch.
2. (Informational) `tools/lint.sh --check`'s `ui:node-version` failure is an environment issue (node v24 vs required v22), not introduced by this branch. Re-run under nvm 22 before treating lint as blocking, per the project's known false-fail pattern.
