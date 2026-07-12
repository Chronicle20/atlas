# Processor Gen3 unification — FR-8 inventory

Ground truth for `docs/tasks/task-116-processor-gen3-unification/plan.md`. Per Global Constraint 15, this document — not the counts in the PRD, design, or plan — is authoritative. Task 1 creates it from the plan-time snapshot; every later task updates the affected rows (`status` → `done`) in the same commit as its conversion; each V-phase re-runs the scan and appends an updated count summary.

**Latest scan:** 2026-07-12, branch `task-116-processor-gen3-unification`, from the worktree root. Result: 377 non-mock `processor.go` files scanned; 131 non-conforming rows. This exactly matches the plan-time snapshot (§"Ground-truth inventory (point-in-time)" in `plan.md`, scanned at `fde55e232`): CP-2 20 (18 pure R1 + 2 that are simultaneously Gen2.5-shaped and fold into R2), Gen2 58, Gen2.5 5, Gen1 50 (45 R3 + 2 R4-client + 3 R6-rename). **No drift found** — every file the plan-time scan named is still present at the same path with the same classification, and the live scan surfaced no files outside the plan's per-task lists.

**Scan after Phase A (2026-07-12):** re-ran the classification scan (script below, verbatim) from the worktree root after Task 7 / commit `74589be603` (the last Phase A conversion). Raw counts: CP-2 9, Gen2 58, Gen2.5 5, Gen1 50 — sum 122, minus the 2 files that double-count in both the CP-2 and Gen2.5 queries (`atlas-messengers/character/processor.go`, `atlas-messengers/messenger/processor.go`, still pending — Task 13/R2) = **120 unique non-conforming rows**, down from 131 (**−11**, exactly the 7 Phase A task rows / 11 files: atlas-doors ×4 [Task 2], atlas-summons ×3 [Task 3], atlas-saga-orchestrator ×1 [Task 4], atlas-pets ×1 [Task 5], atlas-npc-conversations ×1 [Task 6], atlas-mounts ×1 [Task 7]). Verified directly: `grep -rn "func NewProcessor(" services/ --include="*.go" | grep -v "_test.go" | grep -v "/mock/" | grep "\*ProcessorImpl" | grep -E "atlas-doors|atlas-summons|atlas-saga-orchestrator|atlas-pets|atlas-npc-conversations|atlas-mounts"` returns no output — none of the 11 Phase A files remain CP-2-shaped. No new files surfaced outside the plan's per-task lists; no other row's classification changed.

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
| `services/atlas-summons/atlas.com/summons/data/skill/processor.go` | CP-2 | R1 | 3 | done |
| `services/atlas-summons/atlas.com/summons/effectivestats/processor.go` | CP-2 | R1 | 3 | done |
| `services/atlas-summons/atlas.com/summons/inventory/processor.go` | CP-2 | R1 | 3 | done |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/validation/processor.go` | CP-2 | R1 | 4 | done |
| `services/atlas-pets/atlas.com/pets/pet/processor.go` | CP-2 | R1 | 5 | done |
| `services/atlas-npc-conversations/atlas.com/npc/validation/processor.go` | CP-2 | R1 | 6 | done |
| `services/atlas-mounts/atlas.com/mounts/mount/processor.go` | CP-2 | R1 | 7 | done |
| `services/atlas-chairs/atlas.com/chairs/validation/processor.go` | Gen2 | R2 | 8 | done |
| `services/atlas-storage/atlas.com/storage/asset/processor.go` | Gen2 | R2 | 9 | done |
| `services/atlas-storage/atlas.com/storage/storage/processor.go` | Gen2 | R2 | 9 | done |
| `services/atlas-map-actions/atlas.com/map-actions/script/processor.go` | Gen2.5 | R2 | 10 | done |
| `services/atlas-map-actions/atlas.com/map-actions/validation/processor.go` | CP-2 | R1 | 10 | done |
| `services/atlas-portal-actions/atlas.com/portal/script/processor.go` | Gen2.5 | R2 | 11 | done |
| `services/atlas-portal-actions/atlas.com/portal/validation/processor.go` | CP-2 | R1 | 11 | done |
| `services/atlas-reactor-actions/atlas.com/reactor/script/processor.go` | Gen2.5 | R2 | 12 | done |
| `services/atlas-messengers/atlas.com/messengers/character/processor.go` | Gen2.5 (also CP-2 shaped) | R2 | 13 | done |
| `services/atlas-messengers/atlas.com/messengers/invite/processor.go` | Gen1 | R3 | 13 | done |
| `services/atlas-messengers/atlas.com/messengers/messenger/processor.go` | Gen2.5 (also CP-2 shaped) | R2 | 13 | done |
| `services/atlas-configurations/atlas.com/configurations/data/processor.go` | Gen1 | R4-client | 14 | done |
| `services/atlas-configurations/atlas.com/configurations/services/processor.go` | Gen2 | R2 | 14 | done |
| `services/atlas-configurations/atlas.com/configurations/templates/processor.go` | Gen2 | R2 | 14 | done |
| `services/atlas-configurations/atlas.com/configurations/tenants/processor.go` | Gen2 | R2 | 14 | done |
| `services/atlas-character-factory/atlas.com/character-factory/data/processor.go` | Gen1 | R4-client | 15 | done |
| `services/atlas-login/atlas.com/login/guild/processor.go` | Gen2 | R2 | 16 | done |
| `services/atlas-login/atlas.com/login/inventory/processor.go` | CP-2 | R1 | 16 | done |
| `services/atlas-consumables/atlas.com/consumables/cash/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/character/buff/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/character/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/compartment/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/data/consumable/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/data/equipable/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/data/map/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/equipable/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/inventory/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/map/character/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/map/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/monster/drop/position/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/monster/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/pet/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-consumables/atlas.com/consumables/portal/processor.go` | Gen2 | R2 | 17 | done |
| `services/atlas-inventory/atlas.com/inventory/asset/processor.go` | Gen2 | R2 | 18 | done |
| `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` | Gen2 | R2 | 18 | done |
| `services/atlas-inventory/atlas.com/inventory/data/consumable/processor.go` | CP-2 | R1 | 18 | done |
| `services/atlas-inventory/atlas.com/inventory/data/equipment/processor.go` | Gen2 | R2 | 18 | done |
| `services/atlas-inventory/atlas.com/inventory/data/equipment/slot/processor.go` | Gen2 | R2 | 18 | done |
| `services/atlas-inventory/atlas.com/inventory/data/equipment/statistics/processor.go` | Gen2 | R2 | 18 | done |
| `services/atlas-inventory/atlas.com/inventory/data/etc/processor.go` | Gen2 | R2 | 18 | done |
| `services/atlas-inventory/atlas.com/inventory/data/setup/processor.go` | Gen2 | R2 | 18 | done |
| `services/atlas-inventory/atlas.com/inventory/drop/processor.go` | Gen2 | R2 | 18 | done |
| `services/atlas-inventory/atlas.com/inventory/pet/processor.go` | Gen2 | R2 | 18 | done |
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

- `services/atlas-configurations/atlas.com/configurations/data/processor.go` (Task 14): long-lived, startup-wired REST client for atlas-data lookups (skills, items). Constructor is `NewProcessor(l logrus.FieldLogger) Processor` — no `ctx` parameter, unlike every other Gen3 processor in the codebase. Each method (`GetSkillsByIds`, `GetItemById`) takes its own `ctx context.Context` as the first parameter, since the client is constructed once (e.g. at `preset.NewValidator(data.NewProcessor(d.Logger()))` call sites in `templates/resource.go` and `tenants/resource.go`) and reused across requests with different contexts. Full Gen3 (capturing `ctx` at construction) would change failure/cancellation timing across unrelated requests, so this is a rename-only conversion (`Client`→`Processor`, `ClientImpl`→`ProcessorImpl`, `NewClient`→`NewProcessor`) per recipe R4 — method bodies are byte-identical. The existing map-based fake mock (`FakeClient`) was renamed to `ProcessorMock` and kept its stateful map design (not rewritten to func-field shape).
- `services/atlas-character-factory/atlas.com/character-factory/data/processor.go` (Task 15): same shape as `atlas-configurations/data` — long-lived, startup-wired REST client for atlas-data lookups (skills, items), constructed once at `factory.NewProcessor` and reused across requests. Constructor is `NewProcessor(l logrus.FieldLogger) Processor` — no `ctx` parameter. Each method (`GetSkillsByIds`, `GetItemById`) takes its own `ctx context.Context` as the first parameter, so this is a rename-only conversion (`Client`→`Processor`, `ClientImpl`→`ProcessorImpl`, `NewClient`→`NewProcessor`) per recipe R4 — method bodies are byte-identical. The existing map-based fake mock (`FakeClient` in `data/mock/processor.go`) was renamed to `ProcessorMock` and kept its stateful map design (not rewritten to func-field shape); call sites in `factory/processor.go` (field type, constructor, `NewProcessorWithClients` param) and `factory/processor_preset_test.go` (test fixtures) were updated to the new names.
- `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` (Task 18): `WithTransaction(db *gorm.DB) *ProcessorImpl` and `WithAssetProcessor(ap asset.Processor) *ProcessorImpl` keep a concrete `*ProcessorImpl` return type in both the interface and impl, instead of `Processor` (unlike `asset.Processor`'s analogous `WithTransaction`/`WithConsumableProcessor`, which do return `Processor` per the R1 option-pattern wrinkle). Reason: package-internal call sites chain off these methods' results into the *unexported* methods `mergeAssets`/`swapAssets` (`processor.go:449,453`) and the unexported `assetProcessor` field (`processor.go:1765,1799` via `ap.assetProcessor...`/`cp.assetProcessor...`). An interface-typed return would make those unexported members unreachable and force either a logic-shape change or a new getter method neither of which is a declaration-only change. Go permits an interface method to declare a concrete return type, so `WithTransaction`/`WithAssetProcessor` are literal signature renames (`*Processor`→`*ProcessorImpl`) only; every other external caller (`inventory/processor.go`, `compartment/processor_test.go`) only invokes further *exported* methods on the chained result, so this compiles unchanged for them.

## Characterization tests

(Populated by Task 31 — atlas-monster-death `monster` and `monster/drop` packages, per recipe R7.)

## Deferred findings

(Pre-existing bugs discovered during conversion are logged here, not fixed, per Global Constraint 1.)

- `services/atlas-pets/atlas.com/pets/pet/processor.go` (`ProcessorImpl.Create`, task 5): pre-existing `// TODO this needs to generate a cashId if cashId == 0` comment left in the method body — not touched (Constraint 1: no logic changes; the R1 recipe only changes declaration types).
- `services/atlas-login/atlas.com/login/socket/init.go:39` (task 16): pre-existing `go vet` finding `WaitGroup.Add called from inside new goroutine` (the `wg.Add(1)` call sits inside the `go func() { ... }()` closure it starts). Present on `main` before this task (confirmed unchanged since PR #738) and unrelated to `inventory`/`guild` — not touched, per Constraint 1.
- `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` (task 18): the pre-existing (unused, dead) `type Provider interface` was stale relative to the actual `Processor`-to-be method set — it was missing `DecorateAsset` entirely, and its `CreateAssetAndEmit`/`CreateAssetAndLock`/`CreateAsset` signatures were missing the `useAverageStats bool` parameter that the real methods (`processor.go:975,981,990`) already had. No caller in the module referenced `compartment.Provider` (`grep -rn "compartment\.Provider\b"` returns nothing), so it was silently drifting. Per R2 Step 2, the new `Processor` interface was generated fresh from the actual exported method set (53 methods, source order) rather than copied from the stale `Provider` block, which is deleted outright — not a behavior change since nothing referenced it.
- `services/atlas-inventory/atlas.com/inventory/data/equipment/slot/processor.go` and `.../data/equipment/statistics/processor.go` (task 18): both packages populated an exported struct field (`GetById func(id uint32) (...)`, `GetById func(id uint32) ([]Model, error)`) at construction time via `p.GetById = model.CollapseProvider(p.ByIdModelProvider)`. Exported struct fields cannot appear in a `Processor` interface (interfaces have methods, not fields), and the field is used as `slotProcessor.GetById(itemId)`/`statProcessor.GetById(...)` from `data/equipment/processor.go`, which becomes an interface-typed collaborator. `GetById` was converted from a field into a real method — `func (p *ProcessorImpl) GetById(id uint32) (...) { return model.CollapseProvider(p.ByIdModelProvider)(id) }` — mirroring the identical pattern already used by `asset.Processor.GetById` (`asset/processor.go`) and `atlas-pets` `pet.ProcessorImpl.GetById`. Verified behavior-preserving: `model.CollapseProvider` (`libs/atlas-model/model/processor.go:81`) is a pure, stateless wrapper (`return func(a A) (T, error) { return f(a)() }`) with no memoization, so calling it per-invocation via a method is identical to calling a pre-built closure stored in a field.

## R6 file renames

(Populated by Task 24 with each rename's justification, per the R6 table in `plan.md`.)
