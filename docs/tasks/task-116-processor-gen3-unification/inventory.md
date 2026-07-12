# Processor Gen3 unification — FR-8 inventory

Ground truth for `docs/tasks/task-116-processor-gen3-unification/plan.md`. Per Global Constraint 15, this document — not the counts in the PRD, design, or plan — is authoritative. Task 1 creates it from the plan-time snapshot; every later task updates the affected rows (`status` → `done`) in the same commit as its conversion; each V-phase re-runs the scan and appends an updated count summary.

**Latest scan:** 2026-07-12, branch `task-116-processor-gen3-unification`, from the worktree root. Result: 377 non-mock `processor.go` files scanned; 131 non-conforming rows. This exactly matches the plan-time snapshot (§"Ground-truth inventory (point-in-time)" in `plan.md`, scanned at `fde55e232`): CP-2 20 (18 pure R1 + 2 that are simultaneously Gen2.5-shaped and fold into R2), Gen2 58, Gen2.5 5, Gen1 50 (45 R3 + 2 R4-client + 3 R6-rename). **No drift found** — every file the plan-time scan named is still present at the same path with the same classification, and the live scan surfaced no files outside the plan's per-task lists.

## Classification scan script (re-runnable)

Run from the worktree root. Use a scratch directory instead of `/tmp` when re-running interactively; the paths below (`/tmp/...`) are preserved verbatim from the plan/brief so the script matches what Tasks 36 and each V-phase re-run literally.

```bash
# all non-mock processor.go files
find services -name "processor.go" -not -path "*/mock/*" | sort > /tmp/all.txt
# CP-2: NewProcessor returning *ProcessorImpl
grep -rn "func NewProcessor(" services/ --include="*.go" | grep -v "_test.go" | grep -v "/mock/" | grep "\*ProcessorImpl" | sed 's/:.*//' | sort
# Gen2: concrete Processor struct
grep -rln "type Processor struct" services/ --include="*.go" | grep -v mock | sort
# Gen2.5: ProcessorImpl without an interface anywhere in the package
for f in $(grep -rln "type ProcessorImpl struct" services/ --include="*.go" | grep -v mock); do d=$(dirname $f); grep -q "type Processor interface" $d/*.go 2>/dev/null || echo "$f"; done
# Gen1: processor.go with no Processor type at all in the file, and none in the package
for f in $(cat /tmp/all.txt); do d=$(dirname $f); grep -q "type Processor interface\|type Processor struct\|type ProcessorImpl struct" $d/*.go 2>/dev/null || echo "$f"; done
```

Notes on interpreting the output:

- A file can appear in **both** the CP-2 query and the Gen2.5 query (2 atlas-messengers files: `character/processor.go`, `messenger/processor.go` — `NewProcessor` returns `*ProcessorImpl` *and* the package has no `Processor interface`). These are classified `Gen2.5 (also CP-2 shaped)` below and convert once, via **R2** (R2 Step 3 fixes the CP-2-shaped return as part of the extraction — see plan.md Task 13).
- Acceptance (Task 36) re-runs this script and expects the CP-2 and Gen2 queries to return no output, and the Gen1/Gen2.5 queries to return only the two R4-client and three R6-rename files (which by then are no longer named `processor.go`, so they drop out of `all.txt` entirely).

## Non-Gen3-conforming files (131 rows)

`path | classification (Gen1/Gen2/Gen2.5/CP-2/R4-client/R6-rename) | recipe | task # | status`

| Path | Classification | Recipe | Task # | Status |
|---|---|---|---|---|
| `services/atlas-doors/atlas.com/doors/data/map/processor.go` | CP-2 | R1 | 2 | done |
| `services/atlas-doors/atlas.com/doors/data/skill/processor.go` | CP-2 | R1 | 2 | done |
| `services/atlas-doors/atlas.com/doors/door/processor.go` | CP-2 | R1 | 2 | done |
| `services/atlas-doors/atlas.com/doors/party/processor.go` | CP-2 | R1 | 2 | done |
| `services/atlas-summons/atlas.com/summons/data/skill/processor.go` | CP-2 | R1 | 3 | pending |
| `services/atlas-summons/atlas.com/summons/effectivestats/processor.go` | CP-2 | R1 | 3 | pending |
| `services/atlas-summons/atlas.com/summons/inventory/processor.go` | CP-2 | R1 | 3 | pending |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/validation/processor.go` | CP-2 | R1 | 4 | pending |
| `services/atlas-pets/atlas.com/pets/pet/processor.go` | CP-2 | R1 | 5 | pending |
| `services/atlas-npc-conversations/atlas.com/npc/validation/processor.go` | CP-2 | R1 | 6 | pending |
| `services/atlas-mounts/atlas.com/mounts/mount/processor.go` | CP-2 | R1 | 7 | pending |
| `services/atlas-chairs/atlas.com/chairs/validation/processor.go` | Gen2 | R2 | 8 | pending |
| `services/atlas-storage/atlas.com/storage/asset/processor.go` | Gen2 | R2 | 9 | pending |
| `services/atlas-storage/atlas.com/storage/storage/processor.go` | Gen2 | R2 | 9 | pending |
| `services/atlas-map-actions/atlas.com/map-actions/script/processor.go` | Gen2.5 | R2 | 10 | pending |
| `services/atlas-map-actions/atlas.com/map-actions/validation/processor.go` | CP-2 | R1 | 10 | pending |
| `services/atlas-portal-actions/atlas.com/portal/script/processor.go` | Gen2.5 | R2 | 11 | pending |
| `services/atlas-portal-actions/atlas.com/portal/validation/processor.go` | CP-2 | R1 | 11 | pending |
| `services/atlas-reactor-actions/atlas.com/reactor/script/processor.go` | Gen2.5 | R2 | 12 | pending |
| `services/atlas-messengers/atlas.com/messengers/character/processor.go` | Gen2.5 (also CP-2 shaped) | R2 | 13 | pending |
| `services/atlas-messengers/atlas.com/messengers/invite/processor.go` | Gen1 | R3 | 13 | pending |
| `services/atlas-messengers/atlas.com/messengers/messenger/processor.go` | Gen2.5 (also CP-2 shaped) | R2 | 13 | pending |
| `services/atlas-configurations/atlas.com/configurations/data/processor.go` | Gen1 | R4-client | 14 | pending |
| `services/atlas-configurations/atlas.com/configurations/services/processor.go` | Gen2 | R2 | 14 | pending |
| `services/atlas-configurations/atlas.com/configurations/templates/processor.go` | Gen2 | R2 | 14 | pending |
| `services/atlas-configurations/atlas.com/configurations/tenants/processor.go` | Gen2 | R2 | 14 | pending |
| `services/atlas-character-factory/atlas.com/character-factory/data/processor.go` | Gen1 | R4-client | 15 | pending |
| `services/atlas-login/atlas.com/login/guild/processor.go` | Gen2 | R2 | 16 | pending |
| `services/atlas-login/atlas.com/login/inventory/processor.go` | CP-2 | R1 | 16 | pending |
| `services/atlas-consumables/atlas.com/consumables/cash/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/character/buff/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/character/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/compartment/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/data/consumable/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/data/equipable/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/data/map/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/equipable/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/inventory/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/map/character/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/map/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/monster/drop/position/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/monster/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/pet/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-consumables/atlas.com/consumables/portal/processor.go` | Gen2 | R2 | 17 | pending |
| `services/atlas-inventory/atlas.com/inventory/asset/processor.go` | Gen2 | R2 | 18 | pending |
| `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` | Gen2 | R2 | 18 | pending |
| `services/atlas-inventory/atlas.com/inventory/data/consumable/processor.go` | CP-2 | R1 | 18 | pending |
| `services/atlas-inventory/atlas.com/inventory/data/equipment/processor.go` | Gen2 | R2 | 18 | pending |
| `services/atlas-inventory/atlas.com/inventory/data/equipment/slot/processor.go` | Gen2 | R2 | 18 | pending |
| `services/atlas-inventory/atlas.com/inventory/data/equipment/statistics/processor.go` | Gen2 | R2 | 18 | pending |
| `services/atlas-inventory/atlas.com/inventory/data/etc/processor.go` | Gen2 | R2 | 18 | pending |
| `services/atlas-inventory/atlas.com/inventory/data/setup/processor.go` | Gen2 | R2 | 18 | pending |
| `services/atlas-inventory/atlas.com/inventory/drop/processor.go` | Gen2 | R2 | 18 | pending |
| `services/atlas-inventory/atlas.com/inventory/pet/processor.go` | Gen2 | R2 | 18 | pending |
| `services/atlas-channel/atlas.com/channel/data/cash/processor.go` | Gen2 | R2 | 19 | pending |
| `services/atlas-channel/atlas.com/channel/data/npc/processor.go` | Gen2 | R2 | 19 | pending |
| `services/atlas-channel/atlas.com/channel/data/portal/processor.go` | CP-2 | R1 | 19 | pending |
| `services/atlas-channel/atlas.com/channel/data/skill/processor.go` | CP-2 | R1 | 19 | pending |
| `services/atlas-channel/atlas.com/channel/drop/processor.go` | Gen2 | R2 | 20 | pending |
| `services/atlas-channel/atlas.com/channel/map/processor.go` | Gen2 | R2 | 20 | pending |
| `services/atlas-channel/atlas.com/channel/monster/information/processor.go` | Gen2 | R2 | 20 | pending |
| `services/atlas-channel/atlas.com/channel/monster/processor.go` | Gen2 | R2 | 20 | pending |
| `services/atlas-channel/atlas.com/channel/movement/processor.go` | Gen2 | R2 | 20 | pending |
| `services/atlas-channel/atlas.com/channel/portal/processor.go` | CP-2 | R1 | 20 | pending |
| `services/atlas-channel/atlas.com/channel/reactor/processor.go` | Gen2 | R2 | 20 | pending |
| `services/atlas-channel/atlas.com/channel/weather/processor.go` | Gen2 | R2 | 20 | pending |
| `services/atlas-channel/atlas.com/channel/fame/processor.go` | Gen2 | R2 | 21 | pending |
| `services/atlas-channel/atlas.com/channel/guild/processor.go` | Gen2 | R2 | 21 | pending |
| `services/atlas-channel/atlas.com/channel/guild/thread/processor.go` | Gen2 | R2 | 21 | pending |
| `services/atlas-channel/atlas.com/channel/invite/processor.go` | Gen2 | R2 | 21 | pending |
| `services/atlas-channel/atlas.com/channel/messenger/processor.go` | Gen2 | R2 | 21 | pending |
| `services/atlas-channel/atlas.com/channel/party/processor.go` | Gen2 | R2 | 21 | pending |
| `services/atlas-channel/atlas.com/channel/consumable/processor.go` | Gen2 | R2 | 22 | pending |
| `services/atlas-channel/atlas.com/channel/food/processor.go` | Gen2 | R2 | 22 | pending |
| `services/atlas-channel/atlas.com/channel/macro/processor.go` | Gen2 | R2 | 22 | pending |
| `services/atlas-channel/atlas.com/channel/mount/processor.go` | Gen2 | R2 | 22 | pending |
| `services/atlas-channel/atlas.com/channel/pet/processor.go` | Gen2 | R2 | 22 | pending |
| `services/atlas-channel/atlas.com/channel/session/processor.go` | Gen2 | R2 | 22 | pending |
| `services/atlas-channel/atlas.com/channel/summon/processor.go` | Gen2 | R2 | 22 | pending |
| `services/atlas-channel/atlas.com/channel/door/processor.go` | Gen2 | R2 | 23 | pending |
| `services/atlas-channel/atlas.com/channel/merchant/processor.go` | Gen2 | R2 | 23 | pending |
| `services/atlas-channel/atlas.com/channel/party_quest/processor.go` | Gen2 | R2 | 23 | pending |
| `services/atlas-channel/atlas.com/channel/server/processor.go` | Gen1 | R3 | 23 | pending |
| `services/atlas-gachapons/atlas.com/gachapons/test/processor.go` | Gen1 | R6-rename | 24 | pending |
| `services/atlas-messages/atlas.com/messages/command/processor.go` | Gen1 | R6-rename | 24 | pending |
| `services/atlas-npc-shops/atlas.com/npc/test/processor.go` | Gen1 | R6-rename | 24 | pending |
| `services/atlas-account/atlas.com/account/ban/processor.go` | Gen1 | R3 | 25 | pending |
| `services/atlas-portals/atlas.com/portals/character/processor.go` | Gen1 | R3 | 26 | pending |
| `services/atlas-portals/atlas.com/portals/portal/processor.go` | Gen1 | R3 | 26 | pending |
| `services/atlas-reactors/atlas.com/reactors/reactor/data/processor.go` | Gen1 | R3 | 27 | pending |
| `services/atlas-reactors/atlas.com/reactors/reactor/processor.go` | Gen1 | R3 | 27 | pending |
| `services/atlas-asset-expiration/atlas.com/asset-expiration/cashshop/processor.go` | Gen1 | R3 | 28 | pending |
| `services/atlas-asset-expiration/atlas.com/asset-expiration/character/processor.go` | Gen1 | R3 | 28 | pending |
| `services/atlas-asset-expiration/atlas.com/asset-expiration/data/processor.go` | Gen1 | R3 | 28 | pending |
| `services/atlas-asset-expiration/atlas.com/asset-expiration/inventory/processor.go` | Gen1 | R3 | 28 | pending |
| `services/atlas-asset-expiration/atlas.com/asset-expiration/storage/processor.go` | Gen1 | R3 | 28 | pending |
| `services/atlas-monsters/atlas.com/monsters/map/processor.go` | Gen1 | R3 | 29 | pending |
| `services/atlas-monsters/atlas.com/monsters/monster/drop/processor.go` | Gen1 | R3 | 29 | pending |
| `services/atlas-monsters/atlas.com/monsters/monster/information/processor.go` | Gen1 | R3 | 29 | pending |
| `services/atlas-monsters/atlas.com/monsters/monster/mobskill/processor.go` | Gen1 | R3 | 29 | pending |
| `services/atlas-rates/atlas.com/rates/buffs/processor.go` | Gen1 | R3 | 30 | pending |
| `services/atlas-rates/atlas.com/rates/data/cash/processor.go` | Gen1 | R3 | 30 | pending |
| `services/atlas-rates/atlas.com/rates/data/equipment/processor.go` | Gen1 | R3 | 30 | pending |
| `services/atlas-rates/atlas.com/rates/inventory/processor.go` | Gen1 | R3 | 30 | pending |
| `services/atlas-rates/atlas.com/rates/session/processor.go` | Gen1 | R3 | 30 | pending |
| `services/atlas-monster-death/atlas.com/monster/character/processor.go` | Gen1 | R3 | 31 | pending |
| `services/atlas-monster-death/atlas.com/monster/data/equipment/statistics/processor.go` | Gen2 | R2 | 31 | pending |
| `services/atlas-monster-death/atlas.com/monster/monster/drop/position/processor.go` | Gen1 | R3 | 31 | pending |
| `services/atlas-monster-death/atlas.com/monster/monster/drop/processor.go` | Gen1 | R3 | 31 | pending |
| `services/atlas-monster-death/atlas.com/monster/monster/processor.go` | Gen1 | R3 | 31 | pending |
| `services/atlas-monster-death/atlas.com/monster/party/processor.go` | Gen1 | R3 | 31 | pending |
| `services/atlas-data/atlas.com/data/cash/processor.go` | Gen1 | R3 | 32 | pending |
| `services/atlas-data/atlas.com/data/commodity/processor.go` | Gen1 | R3 | 32 | pending |
| `services/atlas-data/atlas.com/data/consumable/processor.go` | Gen1 | R3 | 32 | pending |
| `services/atlas-data/atlas.com/data/etc/processor.go` | Gen1 | R3 | 32 | pending |
| `services/atlas-data/atlas.com/data/pet/processor.go` | Gen1 | R3 | 32 | pending |
| `services/atlas-data/atlas.com/data/setup/processor.go` | Gen1 | R3 | 32 | pending |
| `services/atlas-data/atlas.com/data/job/processor.go` | Gen1 | R3 | 33 | pending |
| `services/atlas-data/atlas.com/data/map/processor.go` | Gen1 | R3 | 33 | pending |
| `services/atlas-data/atlas.com/data/mobskill/processor.go` | Gen1 | R3 | 33 | pending |
| `services/atlas-data/atlas.com/data/monster/processor.go` | Gen1 | R3 | 33 | pending |
| `services/atlas-data/atlas.com/data/npc/processor.go` | Gen1 | R3 | 33 | pending |
| `services/atlas-data/atlas.com/data/quest/processor.go` | Gen1 | R3 | 33 | pending |
| `services/atlas-data/atlas.com/data/reactor/processor.go` | Gen1 | R3 | 33 | pending |
| `services/atlas-data/atlas.com/data/skill/processor.go` | Gen1 | R3 | 33 | pending |
| `services/atlas-data/atlas.com/data/characters/templates/processor.go` | Gen1 | R3 | 34 | pending |
| `services/atlas-data/atlas.com/data/cosmetic/face/processor.go` | Gen1 | R3 | 34 | pending |
| `services/atlas-data/atlas.com/data/cosmetic/hair/processor.go` | Gen1 | R3 | 34 | pending |
| `services/atlas-data/atlas.com/data/equipment/processor.go` | Gen1 | R3 | 34 | pending |
| `services/atlas-data/atlas.com/data/data/processor.go` | Gen1 | R3 | 35 | pending |

**Classification counts (this scan):** CP-2 20 (18 pure R1 + 2 Gen2.5-shaped), Gen2 58, Gen2.5 5 (3 pure + 2 also-CP-2), Gen1 50 (45 R3 + 2 R4-client + 3 R6-rename). Total 131 rows — matches plan-time exactly; zero drift.

## Sanctioned shape deviations

(Populated by Tasks 14 and 15 when the R4 conversions land — the two ctx-per-call REST clients keep `NewProcessor(l) Processor` without a `ctx` parameter, per design §4.2.)

## Characterization tests

(Populated by Task 31 — atlas-monster-death `monster` and `monster/drop` packages, per recipe R7.)

## Deferred findings

(Pre-existing bugs discovered during conversion are logged here, not fixed, per Global Constraint 1.)

## R6 file renames

(Populated by Task 24 with each rename's justification, per the R6 table in `plan.md`.)
