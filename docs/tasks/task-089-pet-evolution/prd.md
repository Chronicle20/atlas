# Pet Evolution Support — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-12
---

## 1. Overview

MapleStory's "evolving" pets — the Dragon family (Dragon Egg → Baby Dragon → adult dragon)
and the Robo family (Robo Egg → Baby Robo → adult robo) — change their item identity as the
player progresses. There are two distinct transformations:

1. **Egg hatching** — a Dragon Egg (`5000028`) or Robo Egg (`5000047`) is consumed and replaced
   by its baby form the first time the player tries to summon it. Automatic, no requirements.
2. **Evolution / re-evolution** — a Baby Dragon (`5000029`) / Baby Robo (`5000048`) that has
   reached the required level is transformed by an NPC into one of several randomly-selected
   adult forms, preserving the pet's name, level, closeness, fullness, and remaining lifespan.
   Adults can be re-evolved (re-rolled) into a different adult of the same family.

Atlas already has a mature pet subsystem (`atlas-pets`), a pet WZ data reader (`atlas-data`),
an inventory/asset service (`atlas-inventory`), an NPC conversation engine
(`atlas-npc-conversations`) with a saga-backed operation framework, and per-version pet packets
(`atlas-channel`). None of this currently implements evolution: `atlas-data` does not parse the
WZ `evol*` nodes, there is no evolution command/event in `atlas-pets`, inventory cannot change an
asset's template in place, and the NPC engine has no evolution operation. This task closes those
gaps so dragon/robo pets can hatch and evolve on every supported tenant version.

This work is faithful to the upstream reference (Cosmic / canonical GMS): in that codebase
evolution is driven **exclusively** through an NPC conversation (NPC Garnox, `1032102.js`,
quests `8185`/`8189`/`4659`); there is no item-use evolution path. Egg hatching is the only
automatic transformation, triggered at summon time. Atlas mirrors this.

## 2. Goals

Primary goals:
- Hatch Dragon/Robo eggs into their baby form automatically when the player summons the egg.
- Allow an NPC conversation to evolve an eligible baby pet into a randomly-selected adult form,
  consuming a Rock of Evolution (`5380000`) and mesos, while preserving the pet's stats.
- Allow re-evolution (re-roll) of an adult pet into a different adult of the same family.
- Source all evolution data (target template ids, outcome probabilities, level/item requirements)
  from WZ via `atlas-data`; let `atlas-pets` perform the random outcome roll.
- Mutate the pet in place — same pet record (`id`, `cashId`) and same inventory asset — so no pet
  history, closeness, or cash linkage is lost.
- Work across every supported tenant version/region without per-version evolution logic.

Non-goals:
- Item-use ("double-click the Rock") evolution. This does not exist in the supported client era;
  evolution is NPC-driven. (The Rock of Evolution's `CashSlotItemType` categorization in
  `atlas-channel` is unrelated UI tab logic and is out of scope.)
- Player-NPC spawning, breeding, or any pet mechanic beyond hatch + evolve.
- Authoring the full production set of Garnox conversation scripts for every world; this PRD
  delivers the engine capability plus one reference conversation definition.
- Changing pet closeness/level/hunger mechanics, which already exist and are correct.

## 3. User Stories

- As a player, when I summon a Dragon Egg or Robo Egg, it hatches into a baby pet so I can raise it.
- As a player, I cannot hatch an egg if I already own the baby form, so I don't create duplicates.
- As a player, when my baby pet reaches the required level, I can visit the evolution NPC and pay a
  Rock of Evolution + mesos to evolve it into a random adult form, keeping its name, level,
  closeness, fullness, and remaining lifespan.
- As a player with more than one evolvable pet summoned, the NPC lets me choose which pet to evolve.
- As a player, I can return to the NPC later to re-evolve an adult pet into a different adult form.
- As a player, if I lack the level, the item, or the mesos, the NPC refuses and nothing is consumed.
- As a server operator, evolution outcomes and requirements come from WZ data, so adding/adjusting
  evolving pets is a data change, not a code change.

## 4. Functional Requirements

### 4.1 Evolution data (atlas-data)

- FR-1.1 The pet WZ reader MUST parse the following nodes from each pet's `info/` block when present:
  `evol` (flag), `evol1`..`evolN` (candidate target template ids), `evolProb1`..`evolProbN`
  (per-candidate weights), `evolNo` (number of candidates), `evolReqPetLvl` (minimum pet level),
  `evolReqItemID` (required consumable, e.g. Rock of Evolution `5380000`).
- FR-1.2 Pets without evolution data MUST continue to read successfully (all evol fields absent →
  pet is non-evolvable). Existing fields (`hungry`, `cash`, `life`, `interact` skills) are unchanged.
- FR-1.3 The pet REST resource MUST expose the evolution data so consumers can read it for a given
  template id (e.g. an `evolutions` collection of `{templateId, probability}` plus `reqPetLevel`
  and `reqItemId`). Field naming follows existing JSON:API conventions in the pet resource.

### 4.2 Egg hatching (atlas-pets)

- FR-2.1 When a pet **spawn** is requested for a template whose WZ `info/evol1` defines a hatch
  target (Dragon Egg `5000028`, Robo Egg `5000047`, and any other egg flagged by data), the egg
  MUST NOT spawn as a pet. Instead the egg is hatched into its `evol1` target.
- FR-2.2 Hatching MUST be refused (with a player-facing message) if the player already owns the
  hatch-target item, matching upstream behavior.
- FR-2.3 Hatching MUST replace the egg inventory asset with the baby asset **in place** (see 4.4),
  preserving the asset's remaining expiration. A hatched baby starts with default pet stats
  (level 1, closeness 0, fullness default) — no stats carry from an egg.
- FR-2.4 Hatch target resolution MUST come from `atlas-data` (`info/evol1`), not hard-coded ids.

### 4.3 Evolution / re-evolution (atlas-pets)

- FR-3.1 `atlas-pets` MUST accept an `EVOLVE` command identifying the target pet (by pet id or by
  owner + slot) and the evolution context (the requesting conversation/saga).
- FR-3.2 On `EVOLVE`, `atlas-pets` MUST:
  1. Load the pet and its current template's evolution data from `atlas-data`.
  2. Reject if the template is not evolvable, or the pet level `< evolReqPetLvl`.
  3. Roll one outcome template id from the candidates using the WZ `evolProb*` weights
     (atlas-pets owns the roll; data supplies the weights — per decision #1).
  4. Mutate the pet record **in place**: change `templateId` to the rolled outcome, **preserve**
     `id`, `cashId`, `name`, `level`, `closeness`, `fullness`, `slot`, `flag`, `excludes`, and
     reset `expiration` to the standard pet lifespan (90 days / 2160h) — matching upstream.
  5. Request the in-place inventory asset template swap (see 4.4).
  6. Emit an `EVOLVED` status event (`oldTemplateId`, `newTemplateId`, `petId`, `slot`).
- FR-3.3 Re-evolution uses the same `EVOLVE` path: an adult template whose WZ data lists evolution
  candidates is rolled again. (Upstream allows adult→adult; data drives whether a template is
  re-evolvable.)
- FR-3.4 If the pet is summoned at evolution time, the channel MUST reflect the new appearance
  (the pet must visually become the new form). This is achieved via the existing spawn/despawn or
  pet-update packet path reacting to `EVOLVED` + the asset template change — no new opcode.
- FR-3.5 Evolution MUST be idempotent/safe against the asset consumer: because the asset is mutated
  in place (not removed + re-added), `atlas-pets`' existing "delete pet on asset DELETED"
  (`consumer/asset/consumer.go`) MUST NOT fire and destroy the pet. (This is the core reason for
  Path A over remove+re-add.)

### 4.4 In-place asset template swap (atlas-inventory)

- FR-4.1 `atlas-inventory` MUST provide a command to change an existing asset's `templateId` in
  place, preserving the asset's slot, compartment, cash reference (`cashId`/referenceId), and
  expiration. This is the enabling capability for both hatching and evolution ("Path A").
- FR-4.2 The swap MUST emit a status event describing the template change (an `UPDATED`-style event
  carrying old and new template ids) so downstream services (`atlas-pets`, `atlas-channel`) react
  without interpreting it as a delete+create.
- FR-4.3 The swap MUST validate the asset exists, is owned by the character, and (for evolution)
  is a pet asset. Failure MUST surface as a saga-compensatable error, not a silent no-op.
- FR-4.4 The swap MUST NOT change the asset's identity in a way that orphans the linked pet record;
  `cashId`/reference continuity is mandatory.

### 4.5 NPC conversation operation (atlas-npc-conversations)

- FR-5.1 Add a remote conversation operation `evolve_pet` that emits a saga driving the evolution.
  It targets a pet chosen earlier in the conversation (by slot/pet id placed in conversation
  context).
- FR-5.2 The NPC engine MUST be able to enumerate the player's evolvable summoned pets so a
  conversation can present a selection menu when more than one is eligible, and auto-select when
  exactly one is eligible. (Reuse the existing pet operation processor `GetPets`/`GetPetIdBySlot`
  and the existing conversation menu/list-selection state — no new client packet.)
- FR-5.3 The conversation MUST consume the Rock of Evolution (`evolReqItemID`) and the meso cost via
  existing operations (`destroy_item` / `destroy_item_from_slot`, `award_mesos` negative), gated by
  existing conversation conditions (has item, has mesos, pet level).
- FR-5.4 The evolution saga MUST coordinate: consume Rock + mesos AND evolve the pet, with
  compensation so that if `evolve_pet` fails (e.g. pet became ineligible), the Rock and mesos are
  not lost. Ordering and compensation follow existing saga-orchestrator patterns.
- FR-5.5 Deliver one reference Garnox-style conversation definition exercising: eligibility gate →
  optional pet-selection menu → confirm → consume + evolve → success message.

### 4.6 Constants & classification

- FR-6.1 Reuse `ClassificationPetEvolution` (538) and Rock of Evolution `5380000` from
  `libs/atlas-constants`; add any new pet template-id constants only if a service needs them
  directly (prefer data-driven over hard-coded ids — egg ids may warrant constants for the
  hatch-on-spawn branch).

## 5. API Surface

### 5.1 atlas-data (REST, JSON:API)

- Modified: `GET /api/data/pets/{id}` and `GET /api/data/pets` — pet resource gains evolution
  attributes, e.g.:
  - `reqPetLevel` (int, omitted/0 when non-evolvable)
  - `reqItemId` (int, omitted/0 when none)
  - `evolutions`: array of `{ templateId: int, probability: int }`
  Exact field names/shape to follow the service's existing resource conventions; finalized in design.

### 5.2 Kafka — atlas-pets

- New command on `COMMAND_TOPIC_PET`: `EVOLVE` — body `{ petId | (ownerId, slot), transactionId }`.
- New status event on `EVENT_TOPIC_PET_STATUS`: `EVOLVED` —
  body `{ petId, slot, oldTemplateId, newTemplateId, transactionId? }`.

### 5.3 Kafka — atlas-inventory

- New command: in-place template swap — body `{ characterId, assetId | (compartment, slot),
  newTemplateId, transactionId }`.
- New/extended status event: template-changed (carries `oldTemplateId`, `newTemplateId`, slot,
  reference id) — distinct enough that consumers do not treat it as delete+create.

### 5.4 atlas-npc-conversations

- New remote operation type `evolve_pet` (params: pet selector — slot or context var). Emits a saga
  step routed to `atlas-pets`/`atlas-inventory` via the saga orchestrator.

### 5.5 atlas-channel

- No new opcode. The existing pet spawn/despawn/update writers MUST reflect the post-evolution
  template when the pet is summoned (react to `EVOLVED` and/or the inventory template-changed event).

## 6. Data Model

- **No new tables.** Evolution mutates existing rows:
  - `atlas-pets` pet row: `template_id` changes; `id`, `cash_id`, stats preserved; `expiration` reset.
  - `atlas-inventory` asset row: `template_id` (and equivalent reference field) changes in place;
    slot, compartment, cash reference, expiration preserved.
- All mutations remain `tenant_id`-scoped (existing GORM tenancy).
- Evolution **data** is read-only from WZ (no persistence) and surfaced via `atlas-data` REST.
- Migration notes: none (schema unchanged). If `atlas-inventory` lacks a clean in-place update path,
  that is a code change, not a migration.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-data` | Parse `evol*` WZ nodes; expose evolution data on the pet REST resource. |
| `atlas-pets` | New `EVOLVE` command + processor (validate, roll outcome via WZ weights, mutate template in place, reset expiration, emit `EVOLVED`); egg hatch-on-spawn branch; ensure asset-DELETED consumer is not triggered by in-place swap. |
| `atlas-inventory` | New in-place template-swap command + status event (Path A); preserve slot/cash reference/expiration. |
| `atlas-npc-conversations` | New `evolve_pet` remote operation; enumerate evolvable summoned pets for selection menu; reference Garnox conversation definition. |
| `atlas-saga-orchestrator` | Wire the evolution saga (consume Rock + mesos + evolve) with compensation. |
| `atlas-channel` | Ensure summoned pet appearance reflects post-evolution template (existing packets, no new opcode). |
| `libs/atlas-constants` | Reuse 538 / `5380000`; add egg/pet template constants only if needed by the hatch branch. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** all commands, events, REST reads, and DB writes are tenant-scoped via existing
  context/header propagation; evolution data is resolved per-tenant via `atlas-data` (version/region).
- **Version coverage:** the mechanic is inventory mutation + (optional) re-spawn — version-agnostic.
  No `MajorVersion`/`Region` branching is introduced for evolution itself. Must function on every
  supported tenant version. Verify pet spawn/appearance refresh on at least one v83 and one v95+ tenant.
- **Atomicity / correctness:** the consume-and-evolve flow is a saga with compensation; a failed
  evolution must not consume the Rock or mesos, and a successful evolution must not lose the pet
  record or its cash linkage (FR-3.5 / FR-4.4 are correctness-critical).
- **Observability:** log evolution attempts/outcomes (old→new template, roll result) at info;
  failures with reason at warn/error. Emit the `EVOLVED` event for downstream/audit.
- **Data-driven:** outcome probabilities and requirements come from WZ; no hard-coded probability
  tables in service code (decision #1). Egg/pet ids may be constants only where a spawn-time branch
  needs them before data is loaded.
- **Code patterns:** follow immutable Builder pattern for pet model changes; processor
  Interface+Impl with `Method(mb)` / `MethodAndEmit()`; `message.Buffer`/`message.Emit`; reuse
  `libs/atlas-constants` types (DOM-21). Verify with `go test -race`, `go vet`,
  `docker buildx bake` per changed service, and `tools/redis-key-guard.sh`.

## 9. Open Questions

- OQ-1 **Outcome roll location vs. `select_random_weighted`:** decision #1 puts the roll in
  `atlas-pets` using WZ weights. Confirm we do *not* also use the conversation's
  `select_random_weighted` (avoid double-rolling); the conversation should delegate the outcome to
  `EVOLVE`. (Leaning: conversation triggers, atlas-pets rolls.)
- OQ-2 **Eligibility gating split:** which checks live in the conversation (has item, has mesos) vs.
  in `atlas-pets` (pet level ≥ `evolReqPetLvl`, template evolvable)? Proposal: conversation gates
  resources, `atlas-pets` is the authority on pet eligibility and re-validates.
- OQ-3 **In-place asset swap feasibility:** does `atlas-inventory` already have a safe path to mutate
  `template_id` in place (preserving cash reference), or must a new administrator/processor method
  be added? Confirm during design by reading the asset model/administrator. (Assumed: new method.)
- OQ-4 **Pet-selection menu vs. lead-pet auto-select:** present a menu only when >1 evolvable pet is
  summoned; otherwise auto-select. Confirm the conversation engine's menu state can be driven from a
  server-computed list of pets (vs. only static config choices).
- OQ-5 **Egg-already-owned guard semantics** (FR-2.2): match upstream "have `petid+1`" exactly, or
  generalize to "owns the hatch-target id"? Proposal: check the data-resolved hatch target id.
- OQ-6 **Hatch asset swap vs. create/delete:** confirm hatching also uses the in-place swap
  (preferred, consistent with FR-3.5) rather than remove+add, even though no stats carry.

## 10. Acceptance Criteria

- [ ] `atlas-data` parses `evol`/`evol1..N`/`evolProb1..N`/`evolNo`/`evolReqPetLvl`/`evolReqItemID`
      and exposes them on the pet resource; non-evolvable pets still read successfully.
- [ ] Summoning a Dragon Egg (`5000028`) or Robo Egg (`5000047`) hatches it into the `evol1` baby,
      consuming the egg in place and preserving expiration; refused if the baby is already owned.
- [ ] An NPC conversation can evolve an eligible baby pet: pet level ≥ `evolReqPetLvl` enforced,
      Rock of Evolution + mesos consumed, outcome rolled from WZ weights by `atlas-pets`.
- [ ] After evolution the pet keeps the same `id`/`cashId`, name, level, closeness, fullness, slot,
      and excludes; `templateId` is the rolled outcome and expiration is reset to 90 days.
- [ ] The linked inventory asset's `templateId` is swapped in place (same slot/cash reference); the
      pet record is NOT deleted by the asset consumer.
- [ ] A summoned pet visually becomes its new form (verified on a v83 and a v95+ tenant).
- [ ] Re-evolving an adult rolls a new adult form via the same path.
- [ ] Failed evolution (ineligible pet) consumes neither the Rock nor mesos (saga compensation).
- [ ] When >1 evolvable pet is summoned, the NPC presents a selection menu; with exactly one it
      auto-selects.
- [ ] `EVOLVE` command and `EVOLVED` event exist on the pet topics; inventory template-swap command
      and its status event exist.
- [ ] No hard-coded evolution probability tables in service code; outcomes are data-driven.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...`, `docker buildx bake atlas-<svc>` for
      every changed service, and `tools/redis-key-guard.sh` are clean.
