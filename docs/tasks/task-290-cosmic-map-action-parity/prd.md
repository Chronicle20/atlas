# Cosmic Map-Action Parity — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-02
Tracking issue: https://github.com/Chronicle20/atlas/issues/1624
---

## 1. Overview

Atlas models map-entry behavior (`onUserEnter`, `onFirstUserEnter`) as declarative JSON rule documents seeded per client/version under `deploy/seed/<client>/<version>/map-actions/`, executed by `atlas-map-actions`. Each document is a `scriptName` plus an ordered list of rules; a rule is an AND-list of conditions and a list of operations. Conditions are either evaluated locally (`map_id`) or forwarded to `atlas-query-aggregator`'s `ValidateCharacterState`; operations are emitted as saga steps consumed by the owning services.

Issue #1624 completed a delta analysis of Cosmic's `scripts/map/` (90 files — 88 `onUserEnter`, 2 `onFirstUserEnter`) against Atlas' seeded set. Only 9 have an Atlas equivalent today, replicated byte-identically across all 11 version directories (`gms/{12_1,48_1,61_1,72_1,79_1,83_1,84_1,87_1,92_1,95_1}`, `jms/185_1` — 99 seed files total). The remaining 81 split into 27 that today's engine can already express, 45 blocked behind 14 capability gaps, and 9 that collapse an entire subsystem behind one native call.

This task closes categories #1 and #2: fix the four pre-existing defects that would corrupt any conversion done today, add an idempotency guard to `spawn_monster`, implement gaps G1–G14, and land all 72 resulting seed documents across all 11 version directories. Category #3 (9 scripts) is explicitly out of scope and stays on the issue for separate design review.

A material finding from repository inspection, not present in the issue: **six of the fourteen gaps already have a saga `Action` defined in `libs/atlas-saga/model.go` and need only a `map-actions` executor arm, not a new cross-service step.** This substantially reduces the engine work and should drive the implementation ordering.

## 2. Goals

Primary goals:

- Correct the four pre-existing defects (schema condition names, schema enum breadth, `/convert-map` output path and envelope, quest-status shift instruction — later found to be a false premise; see the FR-1.4 correction note) before any bulk conversion runs.
- Give `spawn_monster` a `spawnIfAbsent` guard so map re-entry does not stack duplicate monsters, and retrofit the already-seeded `108010301`.
- Implement the 14 engine capability gaps G1–G14 identified in issue #1624.
- Land 72 new map-action seed documents (27 category-#1 + 45 category-#2), each replicated identically to all 11 version directories.
- Leave `/convert-map` in a state where a future conversion produces a correct, correctly-placed, fully-replicated seed on the first attempt.

Non-goals:

- The 9 category-#3 scripts (`aranDirection`, `cygnusJobTutorial`, `goAdventure`, `goLith`, `highposition`, `Massacre_result`, `dojang_Eff`, `dojang_Msg`, `dojang_1st`). These require subsystem design decisions and remain on issue #1624.
- The Pyramid PQ subsystem and the dojo instance subsystem.
- `resetEnteredScript()` support. It appears in three category-#3 scripts and in the already-seeded `spaceGaGa_sMap` (where the guard was dropped); `/convert-map` grants it an explicit exception and that exception stands.
- Re-reviewing the Cosmic sources. Issue #1624 is the inventory of record; its per-script tables are authoritative input to design and planning.
- Version-gating map actions. All 72 documents seed uniformly to all 11 version directories, matching the existing 9. Any script that turns out to be version-specific is a finding to surface, not a decision to make silently.

## 3. User Stories

- As a player entering a job-instructor test map, I want the test monster to be present so I can complete the job advancement, and I want re-entering the map not to fill it with duplicate monsters.
- As a player entering one of the six Demon boss maps, I want the boss to spawn and an announcement to appear.
- As a player boarding a ship or train, I want the vessel to depart with the correct music and boat effect and to be warped to the in-transit map only when the vessel is not docked.
- As a Resistance/Cannoneer/Aran tutorial player, I want the scripted cutscene, sound, NPC conversation and area-info updates to fire so the tutorial progresses.
- As a player entering a Kerning PQ or Ludibrium PQ map, I want drops cleared, reactors reset/shuffled, and PQ state reset so the stage is playable.
- As an explorer entering a new region, I want the region's explorer quest credited.
- As a content author, I want `/convert-map` to emit a valid, JSON:API-enveloped seed in the right directory for every version, so I never hand-fix paths or condition names.
- As a service developer, I want the map-action JSON schema to accept exactly the condition types the engine actually forwards, so a schema-valid script cannot fail at runtime on an unknown condition name.

## 4. Functional Requirements

### 4.1 Defect remediation (must land before any bulk conversion)

**FR-1.1 — Condition names.** `services/atlas-map-actions/docs/map_script_schema.json` declares condition types `map_id`, `gender`, `job`, `level`, `quest_status`. The aggregator accepts `jobId` and `questStatus` (`libs/atlas-saga/validation.go:11,25`); `job` and `quest_status` are not recognized and fail at runtime. The schema must declare the saga names. `map_id` remains the sole locally-evaluated type (`script/evaluator.go:33-37`) and stays as-is.

**FR-1.2 — Enum breadth.** The schema enum must enumerate every condition constant in `libs/atlas-saga/validation.go` (~40: `jobId`, `meso`, `mapId`, `fame`, `item`, `gender`, `level`, `reborns`, `dojoPoints`, `vanquisherKills`, `gmLevel`, `guildId`, `guildLeader`, `guildRank`, `questStatus`, `questProgress`, `hasUnclaimedMarriageGifts`, `strength`, `dexterity`, `intelligence`, `luck`, `buddyCapacity`, `petCount`, `mapCapacity`, `inventorySpace`, `transportAvailable`, `skillLevel`, `hp`, `maxHp`, `buff`, `excessSp`, `partyId`, `partyLeader`, `partySize`, `pqCustomData`, `monsterBookCount`, `petTameness`, `canSpawnPlayerNpc`) plus `map_id`, and any new condition added by §4.3. The operator enum must additionally include `in` (`libs/atlas-saga/validation.go`, `In = "in"`), which the schema currently omits.

**FR-1.3 — `/convert-map` output contract.** `.claude/commands/convert-map.md` instructs the agent to write bare-schema JSON to `services/atlas-map-actions/scripts/map/<hook>/<name>.json`. Real seeds are JSON:API-enveloped:

```json
{"data": {"type": "map-action", "id": "<scriptName>", "attributes": {"scriptName": "<scriptName>", "description": "...", "rules": [...]}}}
```

at `deploy/seed/<client>/<version>/map-actions/<hook>/map-<scriptName>.json`, replicated to all 11 version directories. The command doc must be corrected to emit this shape at this path with this replication, and must use the FR-1.1 condition names.

**FR-1.4 — Quest-status value shift.** Cosmic `QuestStatus` is `UNDEFINED(-1), NOT_STARTED(0), STARTED(1), COMPLETED(2)`; Atlas' aggregator is `UNDEFINED=0, NOT_STARTED=1, STARTED=2, COMPLETED=3`. Every ported `getQuestStatus(x) == n` must be emitted as `n + 1`. `/convert-map` must document this shift, and every conversion in this task must apply it.

> **Correction (task A12 fix round 1):** the premise above is factually wrong and is kept here only
> for history, not as a live requirement. Atlas' aggregator actually numbers quest state
> `NotStarted=0, Started=1, Completed=2`
> (`services/atlas-query-aggregator/atlas.com/query-aggregator/quest/model.go:11-14`), and the
> `QuestStatusCondition` comparison applies no offset to that value
> (`services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go:402-413`). There
> is no `UNDEFINED=0`; the aggregator's sentinel is `State(255)`, which does not correspond to
> Cosmic's `UNDEFINED(-1)`. Every ported `getQuestStatus(x) == n` check ports **1:1, with no
> offset**. `/convert-map` was corrected to reflect this in task A12 fix round 1.

**FR-1.5 — Schema/aggregator drift check.** Because FR-1.2 duplicates a Go constant list into a JSON schema, a check must exist that fails when they diverge — a test or a `tools/verify.sh` guard that reads `libs/atlas-saga/validation.go` and the schema enum and asserts set equality (modulo the locally-evaluated `map_id`).

**FR-1.6 — Seed replication check.** A check must assert that every map-action seed document is byte-identical across all 11 version directories, so a partial replication cannot land.

### 4.2 `spawn_monster` idempotency

**FR-2.1.** The `spawn_monster` operation gains an optional boolean param `spawnIfAbsent`. When true, the spawn is suppressed if a monster of that `monsterId` already exists on the field. Absent or false preserves today's unconditional behavior.

**FR-2.2.** Every Cosmic map spawn is guarded (`getMonsterById(id) != null` / `containsNPC` / `countMonster(...) == 0`). Every spawn converted by this task must set `spawnIfAbsent: "true"`.

**FR-2.3.** The already-seeded `108010301` dropped the guard. Its five rules must be updated to set `spawnIfAbsent: "true"`, in all 11 version directories, alongside the four new duplicates in FR-3.1.

### 4.3 Engine capability gaps (G1–G14)

Gap-to-mechanism classification, derived from `libs/atlas-saga/model.go` and `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:37-52`. **Design must confirm the payload shape and consumer coverage of each "action exists" row before assuming it is executor-only work.**

| Gap | Capability | Saga action status | Work shape |
|---|---|---|---|
| G3 | `set_quest_progress` | `SetQuestProgress` exists (`model.go:148`) | map-actions executor arm |
| G4 | `start_quest` (forced) | `StartQuest` exists (`model.go:147`) | map-actions executor arm |
| G9 | `open_npc` | `StartNpcConversation` exists (`model.go:205`) | map-actions executor arm |
| G12a | `update_area_info` | `UpdateAreaInfo` exists (`model.go:162`) | map-actions executor arm |
| G12b | `show_info` | `ShowInfo` exists (`model.go:163`) | map-actions executor arm |
| G13 | `teach_skill` | `CreateSkill` / `UpdateSkill` exist (`model.go:142-143`) | map-actions executor arm |
| G6 | `questProgress` condition | forwarded already (`evaluator.go` default arm) | schema enum only (FR-1.2) |
| G14a | `map_id` range operators | local evaluation | `evaluator.go:44-53` — add `>`, `<`, `>=`, `<=` |
| G7 | randomized monster selection | n/a | map-actions-local: `monsterIds` list param or a `random` rule selector |
| G1a | warp to map | **no map-warp action** (only `WarpToPortal`, `WarpToRandomPortal`, `WarpToSavedLocation`, `WarpPartyQuestMembersToMap`) | new saga action + consumer |
| G1b | docked/transport-state condition | `transportAvailable` exists (`validation.go:36`) but is unverified for this use | verify, else new aggregator condition |
| G1c | `change_music`, boat/crog effect | none | new saga actions + consumers |
| G2 | `spawn_npc` | none (`DeployPlayerNpc` is unrelated) | new saga action + consumer |
| G5 | `clear_drops`, `reset_reactors` (all + state-filtered), `shuffle_reactors`, `reset_pq`, `count_monster` | none (`HitReactor`, `UpdatePqCustomData` exist but do not cover these) | new saga actions + consumers |
| G8/G10 | direction/cutscene system: `set_direction`, `set_direction_status`, `set_direction_mode`, `start_direction`, `lock_ui2`, `play_sound`, `send_direction_info` | none | new subsystem |
| G11 | `set_standalone_mode` | none | new saga action + consumer |
| G12c | `area_info` condition (`containsAreaInfo`) | none | new aggregator condition |
| G14b | `explorer_quest` | none | new saga action + consumer |
| G6b | job-family / `in`-operator condition for `isCygnus()` | `In` operator exists (`validation.go`) | confirm aggregator handles `in` on `jobId`; else new condition |

**FR-3.0 — No silent no-ops.** `executor.go`'s `default:` arm logs an unknown operation and returns `nil`, so a mistyped or unimplemented operation fails silently. Every operation this task introduces must be present in the executor switch, and the schema's operation enum (FR-1.2's sibling) must be widened in lockstep. Design should decide whether the `default:` arm should return an error instead of `nil` — a silently-ignored operation is how a broken seed reaches production undetected.

**FR-3.1 — G8 unresolved reference.** `cannon_tuto_01` calls `startDirection("cannon_tuto_02")`, referencing a script Cosmic does not ship. This must be resolved (located in WZ, or the call dropped with a documented rationale) before `cannon_tuto_01` is converted. It is the one genuine external unknown in category #2.

### 4.4 Category #1 conversions (27 scripts)

Per issue #1624's inventory. All expressible with today's operations once §4.1 and §4.2 land.

- **`field_effect` only (7):** `go1000000`, `go1010000`, `go1010200`, `go1010300`, `go30000`, `go40000`, `go50000` — single `mapEffect("maplemap/enter/<id>")`.
- **`unlock_ui` + `field_effect` (2):** `go10000`, `go20000`.
- **`unlock_ui` only (1):** `evanleaveD`.
- **`field_effect` (1):** `Resi_tutor20` — `resistance/tutorialGuide`.
- **Gendered cutscene, two rules on `gender = 0` / `gender = 1` (4):** `crash_Dragon`, `getDragonEgg`, `meetWithDragon`, `PromiseDragon` — `Effect/Direction4.img/<name>/Scene{0,1}`. Same shape as the seeded `goArcher`.
- **Job + level gate → `unlock_ui` (1):** `startEreb` — conditions `jobId = 1000`, `level >= 10` (FR-1.1).
- **Boss spawn + message (6):** each `spawn_monster` (with `spawnIfAbsent`, FR-2.2) + `drop_message`.

  | Script | Monster | Position | Message |
  |---|---|---|---|
  | `677000001` | 9400612 | (461, 61) | Marbas has appeared! |
  | `677000003` | 9400610 | (467, 0) | Amdusias has appeared! |
  | `677000005` | 9400609 | (201, 80) | Andras has appeared! |
  | `677000007` | 9400611 | (171, 50) | Crocell has appeared! |
  | `677000009` | 9400613 | (251, -841) | Valefor has appeared! |
  | `677000012` | 9400633 | (842, 0) | Astaroth has appeared! |

- **Quest-gated spawn (1):** `100000006` — `questStatus`, `referenceId 2175`, value `"1"` (corrected in
  task A12 fix round 1 — FR-1.4's shift premise is wrong; see the correction note there; no offset
  applies) → spawn 9300156 at (-1027, 216).
- **Job-instructor duplicates (4):** `108010101`, `108010201`, `108010401`, `108010501` — same 5-rule body as the seeded `108010301`, differing only in `scriptName`/`id`. Each map's WZ `onUserEnter` names its own script, so four separate documents are required.

### 4.5 Category #2 conversions (45 scripts)

Converted once the owning gap from §4.3 lands. Grouped per issue #1624:

- **G1 (14):** `101000301`→200090010, `200000112`→200090000, `200000122`→200090100, `200000132`→200090200, `200000152`→200090400, `220000111`→200090110, `240000111`→200090210, `260000110`→200090410, `540010001`→540010002, `540010100`→540010101, `600010002`→600010003, `600010004`→600010005, plus `200090000` and `200090010` (music + boat effect).
- **G2 (5):** `108010600` (npc 1104100 @ 2830,78), `108010610` (1104101 @ 3395,-322), `108010620` (1104102 @ 500,-522), `108010630` (1104103 @ -2263,-582), `108010640` (1104104 @ 372,70).
- **G3 (3):** `130030000`, `130030001` → (20010, 20022, 1); `914000100` → (21000, 21002, 1).
- **G4 (1):** `babyPigMap` — `unlock_ui` + force-start quest 22015.
- **G5 (4):** `922000000`, `926000000`, `926000010`, `926120300`.
- **G6 (2):** `925040100`, `910510000`.
- **G7 (1):** `pepeking_effect` — one of 3300005/3300006/3300007 at (-28,-67).
- **G8 (4):** `cannon_tuto_01`, `cannon_tuto_direction`, `cannon_tuto_direction1`, `cannon_tuto_direction2`.
- **G9 (3):** `Resi_tutor40` (npc 2159012), `Resi_tutor50` (npc 2159006, also G10), `Resi_tutor60` (npc 2159007).
- **G10 (2):** `Resi_tutor70`, `Resi_tutor80`.
- **G11 (1):** `Resi_tutor10`.
- **G12 (3):** `Resi_tutor30`, `rienArrow`, `rien`.
- **G13 (1):** `iceCave` — teach 20000014–20000018, then `unlock_ui` + `show_intro`.
- **G14 (1):** `explorationPoint` — eight map-ID range branches → `explorer_quest` (29005 Beginner, 29014 Sleepywood, 29006 El Nath, 29007 Ludus Lake, 29008 Undersea, 29009 Mu Lung, 29010 Nihal Desert, 29011 Minar Forest) plus `field_effect maplemap/enter/104000000` on map 104000000.

### 4.6 Seeding

**FR-6.1.** Every new document is written to all 11 version directories under the correct hook subdirectory (`onUserEnter` or `onFirstUserEnter`), named `map-<scriptName>.json`, JSON:API-enveloped with `type: "map-action"` and `id` equal to `scriptName`.

**FR-6.2.** Every document carries a human-readable `description` naming the map and the behavior, matching the style of the existing 9 seeds.

## 5. API Surface

No new REST endpoints. The changed surfaces are:

- **Map-action document schema** (`services/atlas-map-actions/docs/map_script_schema.json`) — condition `type` enum widened per FR-1.2, operator enum gains `in`, operation `type` enum gains every operation from §4.3, and the per-operation `allOf` param constraints extended for each new operation (`spawnIfAbsent` on `spawn_monster`, and the params of every G-gap operation).
- **Saga step actions** (`libs/atlas-saga/model.go` + `payloads.go` + `unmarshal.go`) — new `Action` constants and payload types for the G1/G2/G5/G8/G11/G14 operations, each with a consumer in the owning service.
- **`ValidateCharacterState` conditions** (`atlas-query-aggregator`) — a new `areaInfo` condition (G12c), and whichever of the docked-state (G1b) and job-family (G6b) conditions turn out not to be expressible with the existing constants.

## 6. Data Model

No database entities are added. Map actions are seed documents, not persisted domain objects.

The document model changes as described in §5. Note the schema's `params` is `additionalProperties: {"type": "string"}` — every operation parameter is a string, including numeric and boolean ones (`"spawnIfAbsent": "true"`). New operations must not break this by introducing nested or non-string params without a deliberate schema change.

Migration: none. Seeds are re-applied per deploy; the `108010301` retrofit (FR-2.3) is an in-place edit of 11 existing files.

## 7. Service Impact

| Service / module | Change |
|---|---|
| `services/atlas-map-actions` | Executor arms for every new operation; `evaluator.go` range operators on `map_id`; `spawnIfAbsent` plumbing; schema update; drift and replication checks |
| `libs/atlas-saga` | New `Action` constants, payload structs, unmarshal cases, and tests for G1/G2/G5/G8/G11/G14 operations |
| `services/atlas-query-aggregator` | New condition(s): `areaInfo`; possibly docked-state and job-family |
| `atlas-map` (or the field-owning service) | Consumers for warp-to-map, `spawn_npc`, `clear_drops`, `reset_reactors`, `shuffle_reactors`, `change_music`, boat effect, `count_monster`, `spawnIfAbsent` guard on the spawn step |
| Party-quest service | `reset_pq` consumer (G5) |
| Quest service | `explorer_quest` consumer (G14) |
| `atlas-channel` | Direction/cutscene packets (G8/G10/G11): `set_direction*`, `start_direction`, `lock_ui2`, `play_sound`, `send_direction_info`, `set_standalone_mode`, `start_map_effect` |
| `deploy/seed/**` | 72 new documents × 11 directories, plus the `108010301` retrofit × 11 |
| `.claude/commands/convert-map.md` | Corrected output path, envelope, replication, condition names; removed the false quest-status shift instruction (see FR-1.4 correction) |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every saga step already carries tenant context; new steps must not bypass it. Seeds are per client/version, not per tenant.
- **Idempotency.** Map-entry scripts fire on every entry. Any new operation with observable side effects (spawns, NPC placement, quest progress, area info) must be safe to run repeatedly or must expose an explicit guard, as `spawn_monster` now does.
- **Fail-loud.** Per FR-3.0, an unrecognized operation currently returns `nil`. Whatever design decides, the seeds landed by this task must be covered by a check that would catch a typo'd operation type before deploy.
- **Observability.** New executor arms follow the existing `l.Debugf`/`l.Warnf` pattern in `executor.go`.
- **Performance.** Non-`map_id` conditions each cost an aggregator round-trip. Scripts with many rules (`explorationPoint`'s eight range branches, `108010301`'s five) should keep locally-evaluable `map_id` conditions first so the cheap check short-circuits.
- **Verification.** `tools/verify.sh` (flagless, exit 0) gates completion, per project policy.

## 9. Open Questions

1. **G8 `cannon_tuto_02` (FR-3.1).** Referenced by `cannon_tuto_01` but not shipped by Cosmic. Needs to be located in WZ data or the call dropped with rationale. This is a genuine external unknown; it blocks one script, not the gap.
2. **Docked-state condition (G1b).** Does the existing `transportAvailable` condition express Cosmic's `em.getProperty("docked") == "false"`, or is a distinct condition needed? Resolvable by reading the aggregator's `transportAvailable` implementation during design.
3. **`isCygnus()` (G6b).** Whether `jobId in [...]` through the existing `In` operator is sufficient, or a `jobFamily` condition is warranted.
4. **`default:` arm behavior (FR-3.0).** Return an error on unknown operation instead of `nil`? Changing it is a behavior change for any existing tenant seed carrying an operation the engine does not know.
5. **Task splitting.** Categories #1 and #2 are one task by decision, but G1–G14 are independent. Planning should decide whether this executes as one plan with ordered phases or is split at the plan boundary; §4.1/§4.2 must complete before any conversion regardless.
6. **`start_map_effect`.** Listed under category #3 (`dojang_Msg`) but may also be needed by G1c's boat effect. Design should confirm whether the two are the same primitive.

## 10. Acceptance Criteria

**Defects and guard**

- [ ] `map_script_schema.json` condition enum matches `libs/atlas-saga/validation.go`'s constants plus `map_id`; `job` and `quest_status` no longer appear.
- [ ] Schema operator enum includes `in`.
- [ ] Schema operation enum includes every operation the executor switch handles, and vice versa.
- [ ] A check fails when the schema condition enum and `validation.go` diverge (FR-1.5).
- [ ] A check fails when a map-action seed is not byte-identical across all 11 version directories (FR-1.6).
- [ ] `.claude/commands/convert-map.md` documents the JSON:API envelope, the `deploy/seed/<client>/<version>/map-actions/<hook>/map-<scriptName>.json` path, 11-way replication, the saga condition names, and that quest-status values port 1:1 with no shift (see FR-1.4 correction).
- [ ] `spawn_monster` accepts `spawnIfAbsent`; when true and a monster of that id is present on the field, no spawn occurs. Covered by a test.
- [ ] All 11 copies of `map-108010301.json` set `spawnIfAbsent: "true"` on all five rules.

**Engine gaps**

- [ ] Each of G1–G14 has its operations/conditions implemented end to end: schema entry, executor arm (or evaluator support), saga action + payload + unmarshal where new, and a consumer in the owning service.
- [ ] `evaluator.go` evaluates `map_id` with `>`, `<`, `>=`, `<=` in addition to `=` / `!=`.
- [ ] `questProgress` is usable from a map-action document.
- [ ] For every new cross-service saga step, a test asserts the consumer honors the new contract (per the project's cross-service seam rule).
- [ ] The G8 `cannon_tuto_02` reference is resolved or documented as dropped.

**Conversions**

- [ ] All 27 category-#1 scripts are seeded, in all 11 version directories, matching issue #1624's per-script inventory (monster ids, coordinates, messages, effect paths, quest ids).
- [ ] All 45 category-#2 scripts are seeded, in all 11 version directories.
- [ ] Every converted spawn sets `spawnIfAbsent: "true"`.
- [ ] Every converted quest-status comparison ports its value unchanged, with no shift (see FR-1.4 correction).
- [ ] Every seeded document validates against the updated schema.
- [ ] Total map-action seed count is 81 documents × 11 directories.

**Gates**

- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before PR.
- [ ] Issue #1624 updated: categories #1 and #2 closed out, category #3 retained.
