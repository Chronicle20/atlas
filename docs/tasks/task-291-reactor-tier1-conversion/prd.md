# Tier-1 Reactor Script Conversion — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-02
Tracking issue: [#1625](https://github.com/Chronicle20/atlas/issues/1625)
---

## 1. Overview

Atlas models reactor behaviour as declarative JSON rules seeded into `atlas-reactor-actions`, replacing the JavaScript `hit()`/`act()` scripts that Cosmic (the OdinMS-derived reference server) executes directly. A delta analysis of the full Cosmic corpus (`Cosmic/scripts/reactor/`, 292 scripts) against Atlas's seeded catalogue (`deploy/seed/*/reactor-actions/reactors/`, 13 scripts) found **279 unconverted reactors**, of which **159 are fully expressible with operations and conditions the engine already dispatches today**. Those 159 are this task.

Before any of them can be converted, a defect in the conversion tooling must be corrected. The `/convert-reactor` skill documents `dropItems` as taking `(meso, minMeso, maxMeso, mesoRange, item)`. Cosmic's actual signature (`ReactorActionManager.java:142`) is `(meso, mesoChance, minMeso, maxMeso, minItems)` — **the arguments are shifted by one position**, and `mesoRange`/`item` do not exist. `executor.go` already models the correct shape (`mesoChance`, `mesoMin`, `mesoMax`, `minItems`) and never reads `mesoRange` or `item`, so the single already-seeded reactor that passes drop parameters (`reactor-2001.json`) is seeding wrong values today. The skill is stale in three further ways: it documents 7 of the engine's 11 operations, 1 of its 2 condition types, instructs the converter to skip any script touching `eim.*` (which would wrongly reject most of the corpus), and writes output to a path that is not where seeds live.

This task therefore has two halves: repair the conversion contract and the documentation that describes it, then execute the 159 conversions against the repaired contract. It makes no behavioural change to `atlas-reactor-actions` beyond removing a now-unused legacy parameter fallback.

## 2. Goals

Primary goals:

- Correct the `/convert-reactor` skill's operation set, condition set, parameter mappings, output path, and seed fan-out instructions so that it describes the shipped engine.
- Bring `services/atlas-reactor-actions/docs/reactor_script_schema.json` into agreement with the shipped engine, including `touchRules`, which the schema does not document.
- Correct the meso parameter mapping in the 13 already-seeded reactor actions.
- Convert all 159 Tier-1 Cosmic reactor scripts to `reactor-action` seed JSON, fanned out across all 11 tenant seed directories.
- Remove the legacy `minMeso`/`maxMeso` fallback from `executor.go` once no seed depends on it.

Non-goals:

- Any new operation or condition type. All 65 Tier-2 scripts, and the engine work in issue #1625 §2 (`map_message`, `warp`, `spawn_npc`, `toggle_environment`, `give_pq_exp`, the `map_id`/`monster_count`/`random_chance` conditions, etc.) are out of scope.
- `untouchRules`. Nine Tier-3 scripts need it; it is a separate task.
- The 55 Tier-3 scripts in issue #1625 §3.
- Migrating `atlas-reactor-actions` to the shared catalog root (`deploy/seed/shared/all/`). Considered and explicitly deferred — see §9.
- Any change to the drop, monster, or saga services that consume the emitted saga steps.

## 3. User Stories

- As a player, I want reactors across Victoria Island, Ossyria, and the boss maps to drop items when destroyed, so that the world behaves like the reference server rather than being inert.
- As a player in a party quest, I want PQ chest and box reactors to advance the instance's counters when opened, so that stage progression works.
- As a player fighting an area boss, I want tombstone and altar reactors to weaken the boss when hit in the correct state, so that Lich, the Snow Witch, Rurumo, and the Scholar Ghost are defeatable as designed.
- As an Atlas developer, I want `/convert-reactor` to describe the operations the engine actually dispatches, so that running it produces a correct conversion rather than one that silently drops or shifts parameters.
- As an Atlas developer, I want the reactor script JSON schema to be authoritative, so that schema validation is a meaningful gate on generated seed data.

## 4. Functional Requirements

### 4.1 Conversion contract repair (must land before 4.3)

- **FR-1.** The `/convert-reactor` skill's operation table MUST list all 11 operations dispatched by `ExecuteOperation` in `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go`: `drop_items`, `spawn_monster`, `spray_items`, `weaken_area_boss`, `move_environment`, `kill_all_monsters`, `drop_message`, `update_pq_state`, `hit_reactor`, `broadcast_pq_message`, `stage_clear_attempt`.
- **FR-2.** The skill's condition table MUST list both condition types evaluated by `evaluator.go`: `reactor_state` and `pq_custom_data` (the latter carrying a `step` key naming the custom-data field), over the operators `=`, `!=`, `>`, `<`, `>=`, `<=`.
- **FR-3.** The skill MUST document `drop_items` parameters as `meso`, `mesoChance`, `mesoMin`, `mesoMax`, `minItems`, mapped positionally from `rm.dropItems(meso, mesoChance, minMeso, maxMeso, minItems)`. The names `mesoRange` and `item` MUST NOT appear.
- **FR-4.** The skill MUST document `spray_items` as taking the same five parameters as `drop_items`, mapped from `rm.sprayItems(meso, mesoChance, minMeso, maxMeso, minItems)`. `executeSprayItems` injects `dropType=spray` and delegates to `executeDropItems`, so the parameters are read identically.
- **FR-5.** The skill MUST document `spawn_monster` as accepting `monsterId`, `count`, `x`, and `y`, mapped from `rm.spawnMonster(id)`, `rm.spawnMonster(id, qty)`, and `rm.spawnMonster(id, qty, x, y)`. `x`/`y` default to the reactor's position when absent.
- **FR-6.** The skill MUST NOT instruct the converter to skip scripts using `eim.*`. It MUST instead document the mapping from event-instance calls to the PQ operations: `eim.getIntProperty`/`getProperty` → `pq_custom_data` condition; `eim.setIntProperty`/`setProperty` → `update_pq_state` (`updates` for assignment, `increments` for `+ 1`); `eim.dropMessage` → `broadcast_pq_message`; `eim.showClearEffect` with `giveEventPlayersStageReward` → `stage_clear_attempt`.
- **FR-7.** The skill MUST state the correct output contract: a JSON:API envelope `{"data":{"id":"<reactorId>","type":"reactor-action","attributes":{...}}}` written as `reactor-<reactorId>.json` into each of the eleven tenant seed directories listed in FR-14. It MUST NOT reference `services/atlas-reactor-actions/scripts/reactors/`.
- **FR-8.** `services/atlas-reactor-actions/docs/reactor_script_schema.json` MUST document `touchRules` alongside `hitRules` and `actRules`. The engine has supported touch rules since task-249 (PR #1459, `b2aaa10da`) (`model.go:38`, `entity.go:36`, `rest.go:24`, `processor.go:185`), falling back to `hitRules` when a script declares none. The schema does not mention them.
- **FR-9.** The schema's `drop_items` and `spray_items` parameter definitions MUST describe `mesoChance`, `mesoMin`, `mesoMax`, and `minItems`, and MUST NOT describe `mesoRange` or `item`.

### 4.2 Existing seed correction

- **FR-10.** The 13 already-seeded reactor actions MUST be re-audited against their Cosmic sources and corrected where the meso mapping is shifted. `reactor-2001.json` is known to be affected: `rm.dropItems(true, 2, 8, 15, 1)` must produce `meso=true, mesoChance=2, mesoMin=8, mesoMax=15, minItems=1`, replacing the current `meso=true, minMeso=2, maxMeso=8, mesoRange=15, item=1`.
- **FR-11.** The remaining twelve (`9102002`–`9102007`, `9108000`–`9108005`) MUST be confirmed against their sources. They are PQ scripts using `update_pq_state`, `hit_reactor`, `pq_custom_data`, and `stage_clear_attempt`; the audit must record for each whether it was correct or corrected, not merely assert that it was checked.
- **FR-12.** Corrections MUST be applied identically across all eleven tenant directories.

### 4.3 Tier-1 conversion

- **FR-13.** All 159 reactors enumerated in `tier1-inventory.md` MUST be converted. That file carries each script's verbatim source body and source comment, so conversion and review require no second pass over `Cosmic/scripts/reactor/`.
- **FR-14.** Each converted reactor MUST be written as `reactor-<reactorId>.json` into all eleven tenant seed directories: `deploy/seed/gms/{12_1,48_1,61_1,72_1,79_1,83_1,84_1,87_1,92_1,95_1}/reactor-actions/reactors/` and `deploy/seed/jms/185_1/reactor-actions/reactors/`. The eleven copies MUST be byte-identical, matching the existing convention (verified by `md5sum` across `gms/12_1`, `gms/95_1`, `jms/185_1` for `reactor-2001.json`).
- **FR-15.** Every generated file MUST carry a `description` attribute. Where the Cosmic source has a comment, it is the basis for the description; where it does not, the description MUST state the reactor's observable behaviour, not be left empty.
- **FR-16.** `hit()` maps to `hitRules`; `act()` maps to `actRules`. An absent or empty function produces `[]`.
- **FR-17.** An early-return guard (`if (rm.getReactor().getState() !== 0) { return }` followed by an operation) MUST be inverted into a positive `reactor_state` condition (`operator: "="`, `value: "0"`) on the rule carrying that operation. Four scripts use this form.
- **FR-18.** The two scripts with literal-bound loops MUST be unrolled to `count` parameters, not emitted as repeated operations: `2201001` (`for i<3: spawnMonster(9300007)`) becomes one `spawn_monster` with `count: "3"`; `2511001` (`for i<6: spawnMonster(9300124); spawnMonster(9300125)`) becomes two `spawn_monster` operations with `count: "6"` each.
- **FR-19.** The six scripts with an empty `act()` body (`9018000`–`9018005`) MUST be seeded with `{"hitRules": [], "actRules": []}` rather than omitted, so that the catalogue's coverage is explicit rather than ambiguous with "not yet converted".

### 4.4 Legacy parameter removal

- **FR-20.** Once FR-10 through FR-14 have landed and no seed file in `deploy/seed/` contains the keys `minMeso`, `maxMeso`, `mesoRange`, or `item`, the legacy fallback branches in `executeDropItems` (`executor.go`) that read `minMeso` and `maxMeso` MUST be removed, along with the comment advertising backward compatibility.
- **FR-21.** Removal MUST be ordered after seed correction, never before. A `grep` over `deploy/seed/` proving zero remaining legacy keys is the gate.

## 5. API Surface

No REST endpoint is added, removed, or altered. The `reactor-action` resource shape is unchanged; `TouchRules` is already present in `rest.go:24`.

The seed catalogue's on-disk contract is unchanged in shape and grows by 159 logical entries (1,749 files across eleven tenants).

## 6. Data Model

No schema migration. No new entity, field, or relationship.

`atlas-reactor-actions` persists reactor scripts through `entity.go`, whose `jsonRule` payload already accommodates every operation and condition used by this task. The 159 new rows per tenant are ordinary catalogue data inserted by the existing idempotent seeder (`libs/atlas-seeder`), tenant-scoped exactly as the current 13 are.

The pattern distribution of the 159, for sizing:

| Count | Shape |
|---|---|
| 106 | `drop_items` only |
| 18 | `spray_items` only |
| 13 | `spawn_monster` only |
| 7 | `weaken_area_boss` only |
| 6 | empty `act()` — no operations |
| 4 | `reactor_state` condition + `weaken_area_boss` |
| 2 | `update_pq_state` only |
| 1 | `drop_items` + `update_pq_state` |
| 1 | `update_pq_state` + `spawn_monster` |
| 1 | `update_pq_state` + `spray_items` |

## 7. Service Impact

| Service / artifact | Change |
|---|---|
| `deploy/seed/{gms/*,jms/185_1}/reactor-actions/reactors/` | 159 new files per tenant directory (1,749 total); 13 existing files audited and corrected where the meso mapping is shifted |
| `services/atlas-reactor-actions/docs/reactor_script_schema.json` | Documents `touchRules`; `drop_items`/`spray_items` params corrected |
| `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` | Legacy `minMeso`/`maxMeso` fallback removed from `executeDropItems` (FR-20) |
| `/convert-reactor` skill | Operation set, condition set, parameter mappings, `eim.*` guidance, and output contract corrected |

No other Atlas service changes. The drop, monster, and party-quest services consume the same saga steps they consume today; only the parameter *values* carried by `SpawnReactorDropsPayload` change, and only for `reactor-2001` among existing seeds.

## 8. Non-Functional Requirements

- **Multi-tenancy.** Seed data is tenant-scoped by the existing seeder; the eleven directories correspond to the eleven client-version tenant templates. Reactor behaviour is keyed to reactor classification and is version-independent, which is why the eleven copies are byte-identical.
- **Idempotency.** Reseeding MUST remain idempotent. Adding 159 entries must not change the seeder's re-run semantics.
- **Verification.** Two gates, per the scoping decision:
  1. **Schema validation on every generated file** — all 1,749 files (plus the 13 corrected) validate against `reactor_script_schema.json`. This must be a runnable command, not a manual pass.
  2. **Sampled human review by pattern** — at least one converted reactor from each of the ten shapes in §6, read against its source body in `tier1-inventory.md`. The sample MUST include `reactor-2001` (the meso-shift regression), one of the four `reactor_state`-guarded scripts, both unrolled-loop scripts, and one empty-body script.
- **`tools/verify.sh` (flagless) must exit 0** before the branch is considered done, per repository policy.
- **Observability.** No new logging or metrics. `executeDropItems` already debug-logs the resolved meso parameters, which is sufficient to confirm the corrected mapping at runtime.
- **Reviewability.** A 1,749-file diff is not human-reviewable file by file. The generation step SHOULD be scripted and the script committed, so review targets the generator and the sampled output rather than the bulk diff.

## 9. Open Questions

1. **Shared catalog root — deferred, not rejected.** `libs/atlas-seeder` already exposes `NewFilesystemCatalogSourceWithShared` (used by `atlas-events` and `atlas-tenants`), which would let reactor actions live once under `deploy/seed/shared/all/reactor-actions/reactors/` instead of eleven times. Adopting it would reduce this task's output from 1,749 files to 159 and make every future reactor edit a one-file change. The decision was to fan out to eleven copies now and keep this task pure data. Worth revisiting before Tier 2 adds another 65 reactors at 11× cost.
2. **`spray_items` params and the schema.** `executeSprayItems` mutates `op.Params()` in place to inject `dropType` before delegating. The schema currently declares no parameters for `spray_items`; FR-9 corrects the documentation, but whether `dropType` should be a first-class declared parameter (rather than an internal injection) is unresolved.
3. **PQ scripts in Tier 1 and instance lookup failure.** Four Tier-1 scripts call `update_pq_state`, which resolves the character's PQ instance via a REST call to `atlas-party-quests` and returns an error when there is none. Whether a reactor triggered outside a PQ instance should log and continue rather than fail the operation is existing behaviour, not introduced here, but the four new call sites make it reachable more often.
4. **`9018000`–`9018005` semantics.** These have an empty `act()` in Cosmic. FR-19 seeds them as explicitly empty. If Cosmic relies on some default behaviour when a script defines no operations, that behaviour is not visible in the script and has not been confirmed against the engine.

## 10. Acceptance Criteria

Conversion contract:

- [ ] `/convert-reactor` documents all 11 operations and both condition types.
- [ ] `/convert-reactor` documents `drop_items` and `spray_items` as `meso`/`mesoChance`/`mesoMin`/`mesoMax`/`minItems`; the strings `mesoRange` and `item` appear nowhere in the skill.
- [ ] `/convert-reactor` documents `spawn_monster`'s `count`, `x`, and `y` parameters.
- [ ] `/convert-reactor` documents the `eim.*` → PQ-operation mappings and no longer instructs the converter to skip such scripts.
- [ ] `/convert-reactor` documents the JSON:API envelope and the eleven-directory fan-out, and no longer references `services/atlas-reactor-actions/scripts/reactors/`.
- [ ] `reactor_script_schema.json` documents `touchRules` and the corrected `drop_items`/`spray_items` parameters.

Existing seeds:

- [ ] `reactor-2001.json` emits `mesoChance=2, mesoMin=8, mesoMax=15, minItems=1` in all eleven directories.
- [ ] Each of the other twelve existing reactors has a recorded audit outcome (correct, or corrected with the specific change named).

Tier-1 conversion:

- [ ] All 159 reactor IDs from `tier1-inventory.md` exist as `reactor-<id>.json` in all eleven tenant directories — 1,749 new files.
- [ ] For each of the 159, the eleven copies are byte-identical (verifiable by `md5sum` across directories).
- [ ] Every generated file has a non-empty `description`.
- [ ] The four `reactor_state`-guarded scripts emit a positive `=` condition, not the source's negated form.
- [ ] `2201001` emits one `spawn_monster` with `count: "3"`; `2511001` emits two with `count: "6"`.
- [ ] `9018000`–`9018005` emit empty `hitRules` and `actRules`.

Gates:

- [ ] Every file under `deploy/seed/*/reactor-actions/reactors/` validates against `reactor_script_schema.json` via a committed, runnable command.
- [ ] `grep -rn 'minMeso\|maxMeso\|mesoRange\|"item"' deploy/seed/*/reactor-actions/` returns no matches.
- [ ] The legacy `minMeso`/`maxMeso` fallback is removed from `executeDropItems`, and `atlas-reactor-actions` tests pass.
- [ ] At least one reactor per shape in §6 has been read against its source and the review recorded.
- [ ] Flagless `tools/verify.sh` exits 0.
