# Maker Skill / Item Maker — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28

---

## 1. Overview

The Maker skill is a beginner-tier crafting system. A character who has learned Maker
(`BeginnerMaker`, skill id `1007`, plus the job-variant ids `NoblesseMaker` `10001007`,
`LegendMaker` `20001007`, `EvanMaker` `20011007` — all already declared in
`libs/atlas-constants/skill/identities_gen.go:13,400,520,536`) opens the client's Item Maker
UI and converts monster-drop materials plus mesos into equipment, monster crystals, and
crystal-upgraded gear. It is the only crafting loop in the pre-Big-Bang GMS/JMS content set,
and it is the sink that gives ore, leftover, and crystal drops a purpose.

Atlas has no maker implementation of any kind. `grep -rEiln '\bmaker\b' services/ --include='*.go'`
returns zero files, and both wire operations are unimplemented in the coverage matrix:
`docs/packets/audits/status.json` rows 324 and 551 record `MAKER_SKILL` (serverbound,
`CUIItemMaker::RequestItemMake`) and `MAKER_RESULT` (clientbound, `CUserLocal::OnMakerResult`)
as `incomplete` on every version that carries them — and the `MAKER_SKILL` row additionally
mis-records `gms_v72` and `gms_v79` as `n-a` when both clients in fact send the packet
(FR-4.0). This task therefore spans three layers at
once: packet codecs that do not exist, recipe game-data that is not ingested, and a crafting
domain that has no owning service.

The feature is delivered as a new `atlas-maker` service that owns recipe lookup and craft
validation, an `Etc.wz/ItemMake.img` ingestion path in `atlas-data` that supplies the recipe
table, `MAKER_SKILL`/`MAKER_RESULT` codecs in `libs/atlas-packet`, a channel handler in
`atlas-channel`, and a saga in `atlas-saga-orchestrator` that makes the material burn and the
item award atomic.

## 2. Goals

Primary goals:

- A character meeting a recipe's level, skill-level, material, and meso requirements can craft
  the recipe's output through the in-game Item Maker UI, and the result is visible in their
  inventory without a relog.
- Recipe data is sourced from `Etc.wz/ItemMake.img` through the existing `atlas-data` WZ
  ingestion pipeline, not hand-authored.
- All four Item Maker operations work: craft equipment from materials, craft a monster crystal
  from a leftover, disassemble an equip into crystals, and apply catalyst/gem stat upgrades
  during an equip craft.
- Every material consumption, meso deduction, and item award for a single craft either all
  commits or all rolls back — no partial craft can strand a player's materials.
- `MAKER_SKILL` decodes and `MAKER_RESULT` encodes on every version that carries them, with the
  coverage-matrix cells promoted to verified.

Non-goals:

- The Magatia town content and the NPC that teaches Maker (NPC `2110002` Keol and the
  `2111011`–`2112018` lab objects listed in `docs/research/missing-features/npc-content.md:95`).
  Maker is granted by whatever existing skill-award path the tenant already uses.
- Cash-shop-purchased maker catalysts or premium recipe unlocks.
- An `atlas-ui` screen for browsing or editing recipes.
- Any post-Big-Bang Maker revision (recipe books, profession levels, meister tiers).
- Recipes for item categories absent from the target `ItemMake.img`.

## 3. User Stories

- As a level 45+ character who has learned Maker, I want to open the Item Maker UI and see the
  recipes I qualify for, so that I know what I can craft.
- As a player holding 5× Steel Plate and 5× Fine Gem, I want to craft a Steel Helmet for 50,000
  mesos, so that I get gear I cannot buy from an NPC shop.
- As a player, I want to add a strengthening gem to an equip I am crafting, so that the result
  carries a bonus stat.
- As a player with a stack of monster leftovers, I want to convert them into a monster crystal,
  so that the leftovers become usable maker reagents.
- As a player with an unwanted equip, I want to disassemble it into monster crystals, so that I
  recover value from gear I no longer need.
- As a player whose craft fails validation (missing material, full inventory, insufficient
  mesos), I want a clear failure result and my materials left untouched, so that I do not lose
  items to a rejected craft.
- As an operator, I want maker recipes to load from the tenant's own WZ archive, so that a
  tenant on a different client version gets that version's recipe set.

## 4. Functional Requirements

### 4.1 Recipe ingestion (atlas-data)

`Etc.wz/ItemMake.img.xml` is a two-level tree: top-level directories named by the created item's
leading id digit (`0`, `1`, `2`, `4`, `8`, `16` in the reference archive), each containing
entries keyed by the zero-padded 8-digit created-item id.

- FR-1.1 A new `itemmake` domain in `atlas-data` shall ingest `Etc.wz/ItemMake.img.xml` following
  the established per-archive worker pattern used by `Commodity.img.xml` and `CashPackage.img.xml`
  (`services/atlas-data/atlas.com/data/data/processor.go:177,179`).
- FR-1.2 Each recipe entry shall capture the scalar fields present in the archive: `reqLevel`,
  `reqSkillLevel`, `itemNum`, `tuc`, `meso`, and the optional `catalyst`, `reqItem`, and
  `reqEquip`.
- FR-1.3 Each recipe shall capture its `recipe` child list as ordered `(item, count)` material
  requirements.
- FR-1.4 Each recipe shall capture its optional `randomReward` child list as ordered
  `(item, itemNum, prob)` weighted outcomes. A recipe with a `randomReward` list produces one
  entry sampled by weight; a recipe without one produces `itemNum` of the recipe's own key id.
- FR-1.5 Unknown or absent fields shall default rather than fail ingestion; a malformed
  individual entry shall be logged and skipped without aborting the archive.
- FR-1.6 Ingestion shall be idempotent — re-running it against the same archive yields the same
  rows.

### 4.2 Recipe query (atlas-maker)

- FR-2.1 `atlas-maker` shall expose recipe lookup by created-item id and a listing filtered by
  the requesting character's level and Maker skill level.
- FR-2.2 A recipe shall be reported as craftable for a character only when all of: character
  level ≥ `reqLevel`; Maker skill level ≥ `reqSkillLevel`; the character possesses `reqItem`
  (when set); the character has `reqEquip` equipped (when set); the character holds every
  `recipe` material at the required count; and the character has at least `meso` mesos.
- FR-2.3 Recipe data is read-only and tenant-scoped; `atlas-maker` shall not permit recipe
  mutation through its API.

### 4.3 Craft operations

The four operations are distinguished by the leading `int` mode field of the incoming
`MAKER_SKILL` request. The layout below is **derived from IDA** — `CUIItemMaker::RequestItemMake`
in the v72 and v79 IDBs, which are byte-identical in field order (see
`evidence-maker-skill-v72-v79.md`). It shall be re-derived per version for the remaining
carriers per `docs/packets/PROCESS.md` before the codec is written.

```
Encode4  nMode

nMode 1..2  (create item)
  Encode4  nMode                 // echoed
  Encode4  nTargetItemID
  Encode1  bUseCatalyst
  Encode4  nGemCount             // count of non-null reagent slots
  Encode4  nGemItemID            // repeated, one per non-null slot

nMode 3     (monster crystal from leftover)
  Encode4  3
  Encode4  nLeftoverItemID

nMode 4     (disassemble)
  Encode4  4
  Encode4  nItemID
  Encode4  nInventoryType
  Encode4  nSlotPos
```

- FR-3.1 **Craft item.** Validate per FR-2.2, then atomically deduct `meso`, consume each
  `recipe` material at its `count`, and award the produced item — `itemNum` copies of the
  recipe key id, or one weighted draw from `randomReward` when present. A produced equip shall
  be created with `tuc` upgrade slots.
- FR-3.2 **Catalyst and gems.** When the produced item is an equip, the request may carry a
  catalyst flag and an ordered list of reagent (gem) item ids. Reagents that the character does
  not actually hold shall be dropped from the list rather than failing the craft. The applied
  reagents adjust the produced equip's stats and are consumed. The recipe's `catalyst` field
  names the catalyst item id for that recipe.
- FR-3.3 **Craft monster crystal.** Convert a leftover item into its corresponding monster
  crystal via the `ItemMake.img` top-level `0` directory, whose entries are keyed by crystal id
  with the leftover as the single `recipe` material and the crystal tiers as `randomReward`
  outcomes. Consume the leftover and the recipe's `meso`; award the drawn crystal.
- FR-3.4 **Disassemble equip.** Given an equip slot position and item id, verify the equip is
  present in that EQUIP slot, then destroy it and award the derived crystals plus the derived
  meso refund. See OQ-2 for the derivation source.
- FR-3.5 Every craft operation shall be gated on the character having learned a Maker skill
  variant at level ≥ 1.
- FR-3.6 A craft shall be rejected before any mutation when the character lacks free inventory
  slots for every awarded item.
- FR-3.7 A rejected craft shall leave character state — materials, mesos, equips — byte-identical
  to its pre-request state, and shall return a failure result that re-enables the client's UI.

### 4.4 Wire protocol

- FR-4.0 **Registry correction (prerequisite).** `status.json` row 324 currently records
  `MAKER_SKILL` as `n-a` on `gms_v72` and `gms_v79`. That is wrong: both clients ship the full
  `CUIItemMaker` UI and send the request — `?RequestItemMake@CUIItemMaker@@IAEHXZ` @ `0x760cc3`
  builds `COutPacket(0x70)` on v72, and @ `0x795dc3` builds `COutPacket(111)` on v79. See
  `evidence-maker-skill-v72-v79.md`. Before any codec work, `MAKER_SKILL` shall be added to
  `docs/packets/registry/gms_v72.yaml` (opcode 112) and `gms_v79.yaml` (opcode 111) with
  `provenance: ida-discovered` and the addresses above, and the matrix regenerated.
- FR-4.1 A `MAKER_SKILL` serverbound codec shall decode the request on `gms_v72` (0x70),
  `gms_v79` (0x6F), `gms_v83` (0x71), `gms_v84` (0x71), `gms_v87` (0x74), `gms_v92` (0x7C),
  `gms_v95` (0x7D), and `jms_v185` (0x6C). The op is genuinely `n-a` on `gms_v48` and `gms_v61`
  — neither IDB contains `CItemMakerInfo` or `RequestItemMake` — and shall not be routed there.
- FR-4.2 A `MAKER_RESULT` clientbound codec shall encode the result on `gms_v72` (0x0C7),
  `gms_v79` (0x0CB), `gms_v83` (0x0D9), `gms_v84` (0x0DD), `gms_v87` (0x0E6), `gms_v92` (0x0FA),
  `gms_v95` (0x0F8), and `jms_v185` (0x0E2), per `status.json` row 551. It is `n-a` on `gms_v48`
  and `gms_v61`.
- FR-4.3 `MAKER_RESULT` is a **mode-prefix dispatcher family** — the reference server emits four
  structurally distinct bodies (create-result, crystal-result, disassemble-result, and an
  enable-actions no-op) behind one opcode. It shall be implemented per
  `docs/packets/DISPATCHER_FAMILY.md`: a discrete struct per mode, a config-resolved mode value
  (never hard-coded), and per-mode verification.
- FR-4.4 Opcodes shall be resolved from the per-version registry, never hard-coded, and no wire
  change shall be made to any already-verified version.
- FR-4.5 Version-divergent fields shall be gated with the `MajorAtLeast` idiom, never a raw
  numeric comparison.

### 4.5 Channel handling

- FR-5.1 `atlas-channel` shall register a `MAKER_SKILL` handler on the versions from FR-4.1 that
  resolves the character, calls `atlas-maker` for validation, and emits the craft saga.
- FR-5.2 The handler shall emit a `MAKER_RESULT` on every path, success or failure, so the
  client UI is never left locked.

## 5. API Surface

All endpoints are JSON:API and tenant-scoped by the standard Atlas tenant context headers.

### atlas-data (new)

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/data/item-makes` | Paginated list of ingested maker recipes. |
| `GET` | `/data/item-makes/{itemId}` | One recipe by created-item id. |

Resource `itemMakes` attributes: `reqLevel`, `reqSkillLevel`, `itemNum`, `tuc`, `meso`,
`catalyst`, `reqItem`, `reqEquip`, `recipe` (ordered `{itemId, count}`), `randomReward`
(ordered `{itemId, itemNum, prob}`).

Route registration mirrors `services/atlas-data/atlas.com/data/etc/resource.go:17-27`.

### atlas-maker (new service)

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/characters/{characterId}/maker/recipes` | Recipes the character currently qualifies for. |
| `GET` | `/characters/{characterId}/maker/recipes/{itemId}` | One recipe with per-character eligibility and the computed material/meso cost. |
| `POST` | `/characters/{characterId}/maker/crafts` | Validate a craft request and, on success, emit the craft saga. Returns the saga id. |

Error cases (JSON:API errors with a stable `code`):

| Condition | Status | Code |
|---|---|---|
| Recipe id not found | 404 | `recipe_not_found` |
| Character below `reqLevel` | 422 | `level_too_low` |
| Maker skill absent or below `reqSkillLevel` | 422 | `skill_level_too_low` |
| Missing a `recipe` material | 422 | `insufficient_materials` |
| Missing `reqItem` / `reqEquip` | 422 | `missing_prerequisite_item` |
| Insufficient mesos | 422 | `insufficient_mesos` |
| No free inventory slot for an award | 422 | `inventory_full` |
| Equip not in the named slot (disassemble) | 422 | `equip_not_found` |
| Leftover has no crystal mapping | 422 | `no_crystal_mapping` |

## 6. Data Model

New tables in `atlas-data`, both `tenant_id`-scoped with `tenant_id` in every index and query:

**`item_makes`** — one row per recipe.

| Column | Type | Notes |
|---|---|---|
| `tenant_id` | uuid | Part of the primary key. |
| `item_id` | uint32 | Created-item id; part of the primary key. |
| `req_level` | uint32 | |
| `req_skill_level` | uint32 | |
| `item_num` | uint32 | Quantity produced when no `randomReward`. |
| `tuc` | uint32 | Upgrade slots on a produced equip. |
| `meso` | uint32 | Craft cost. |
| `catalyst` | uint32 | 0 when absent. |
| `req_item` | uint32 | 0 when absent. |
| `req_equip` | uint32 | 0 when absent. |

**`item_make_materials`** — ordered `recipe` entries: `tenant_id`, `item_id` (FK to the recipe),
`ordinal`, `material_item_id`, `count`.

**`item_make_rewards`** — ordered `randomReward` entries: `tenant_id`, `item_id`, `ordinal`,
`reward_item_id`, `item_num`, `prob`.

`atlas-maker` holds **no** persistent craft state. A craft is a saga; its durability lives in
`atlas-saga-orchestrator`. If audit history is wanted, that is OQ-4.

Migration notes: all three tables are populated by re-running the WZ ingestion for the tenant;
no data migration from an existing table is required, since none exists.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-data` | New `itemmake` domain (reader, entity, processor, resource, registry) plus an `Etc.wz` worker registration alongside the existing Commodity/CashPackage/MakeCharInfo registrations. |
| `atlas-maker` | **New service.** Owns recipe query, per-character eligibility, craft validation, and saga emission. Structured after `atlas-reward-pools` (`model`/`entity`/`provider`/`processor`/`resource`/`rest`). Follow `docs/adding-a-new-service.md`. |
| `libs/atlas-packet` | New `MAKER_SKILL` decoder and `MAKER_RESULT` encoder; `MAKER_RESULT` as a dispatcher family with a discrete struct per mode. |
| `atlas-channel` | New `maker_skill` handler; `MAKER_RESULT` writer; route registration across the applicable seed templates. |
| `atlas-saga-orchestrator` | New maker craft saga composed from existing actions in `libs/atlas-saga/model.go`: `AwardAsset`, `DestroyAsset`, `DestroyAssetFromSlot`, `AwardMesos`, with compensation for each. A new action type is expected only if equip-with-`tuc`-and-gem-stats cannot be expressed by `AwardAsset`. |
| `libs/atlas-constants` | Any new maker-specific item-id classification helper (reagent/leftover/crystal predicates) belongs here, not in a service. Check for an existing helper before adding one. |
| `atlas-inventory` | Consumer of the saga only; no interface change expected. Confirm during design. |
| `atlas-skills` | Read-only source of the character's Maker skill level. |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every recipe read, eligibility check, and saga is tenant-scoped through the
  standard tenant context. No cross-tenant recipe read is possible.
- **Atomicity.** A craft mutates materials, mesos, and inventory through a single saga with
  compensation on every step. A crash mid-craft resolves to either fully applied or fully
  compensated, never partial.
- **Anti-duplication.** The craft request shall carry enough identity that a replayed or
  duplicated `MAKER_SKILL` packet cannot craft twice from one material set. Validation must not
  be trusted from client-supplied quantities; material counts come from the server-side recipe
  and the server-side inventory read.
- **Randomness.** `randomReward` sampling is server-side only. The client is never told the
  weight table.
- **Performance.** Recipe eligibility for the full recipe set for one character shall resolve in
  a single batched inventory read, not one read per recipe.
- **Observability.** Each craft logs tenant, character, mode, recipe id, materials consumed,
  meso delta, and produced item at info level, with the saga id as the correlation key.
- **Security.** All slot positions, item ids, and reagent lists from the client are untrusted
  input and shall be re-validated server-side against the character's actual inventory.

## 9. Open Questions

- **OQ-0 — resolved.** Whether `MAKER_SKILL` exists on gms_v72/v79 is settled: it does, at
  opcodes 112 and 111 respectively, confirmed in IDA. See `evidence-maker-skill-v72-v79.md`.
- **OQ-1 — `ItemMake.img` presence per tenant.** The reference archive at
  `Etc.wz/ItemMake.img.xml` (16,595 lines, top-level dirs `0/1/2/4/8/16`) is confirmed present in
  the local reference dumps. **Not yet verified** that the archive shipped by each target tenant's
  client version contains it, or that the field set is identical across GMS v72 → JMS v185.
  Verify per version during design.
- **OQ-2 — Disassembly derivation.** `ItemMake.img` does not encode disassembly. The reference
  server derives the crystal yield and meso refund from the equip's level requirement
  (`getMakerStimulantFromEquip` → `getCrystalForLevel(getEquipLevelReq(equipId))`) and reads
  reagent stat upgrades from a server-owned `makerreagentdata` table. Atlas must decide: derive
  from equip level requirement (matching the reference), or introduce a tenant-owned crystal-tier
  table. This is a design decision, not a blocker.
- **OQ-3 — Leftover → crystal mapping.** The reference server maps a leftover to its crystal via
  `drop_data` (which monster drops the leftover), not via `ItemMake.img`. In Atlas the equivalent
  source is `atlas-drop-information` / `atlas-monsters`. Confirm during design whether the
  `ItemMake.img` top-level `0` directory alone is sufficient (its entries already pair a leftover
  material with crystal `randomReward` outcomes) or whether the drop-table join is required.
- **OQ-4 — Craft audit history.** Should completed crafts be persisted for support/audit, or is
  the saga record sufficient?
- **OQ-5 — Reagent stat-upgrade table.** The gem/reagent → `(stat, value)` mapping has no
  `ItemMake.img` source. Decide whether it is derived from item info, seeded, or tenant-configured.
- **OQ-6 — Maker skill acquisition.** With Magatia NPC content out of scope, confirm how a
  character obtains Maker on the target tenant (starting skill, GM grant, existing quest).

## 10. Acceptance Criteria

Data:

- [ ] `Etc.wz/ItemMake.img.xml` ingests into `item_makes`, `item_make_materials`, and
      `item_make_rewards`, tenant-scoped, and re-ingestion is idempotent.
- [ ] `GET /data/item-makes/{itemId}` returns a recipe whose scalar fields, ordered materials,
      and ordered random rewards match the source XML for at least one entry from each top-level
      directory present in the archive.

Domain:

- [ ] `GET /characters/{id}/maker/recipes` returns only recipes the character qualifies for, and
      excludes ones failing each of level, skill level, material, `reqItem`, `reqEquip`, and meso
      checks — one test per exclusion reason.
- [ ] Every error code in §5 is returned by a test that provokes exactly that condition.

Gameplay:

- [ ] Crafting equipment consumes the exact recipe materials and mesos and awards the produced
      equip with `tuc` upgrade slots.
- [ ] Crafting with a `randomReward` recipe awards exactly one weighted draw.
- [ ] A monster-crystal craft consumes the leftover and awards a crystal.
- [ ] Disassembling an equip destroys the equip in the named slot and awards the derived crystals
      and meso refund.
- [ ] A craft with a catalyst and gems consumes them and applies the stat upgrade to the result.
- [ ] Reagents the character does not hold are dropped from the request without failing the craft.
- [ ] Every failure path (each §5 error condition) leaves materials, mesos, and equips unchanged,
      verified by asserting pre- and post-request state.
- [ ] A craft rejected for `inventory_full` mutates nothing.

Protocol:

- [ ] `MAKER_SKILL` is present in the `gms_v72` and `gms_v79` registries with
      `provenance: ida-discovered`, and both matrix cells have flipped off `n-a`.
- [ ] `MAKER_SKILL` decodes on `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`,
      `gms_v95`, `jms_v185`, with a byte-fixture test per version derived from the client IDB.
- [ ] `MAKER_RESULT` encodes on `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`,
      `gms_v95`, `jms_v185`, with a byte-fixture test per version and per dispatcher mode.
- [ ] `packet-audit dispatcher-lint`, plus `matrix`, `fname-doc`, and `operations --check`, exit 0.
- [ ] The `MAKER_SKILL` and `MAKER_RESULT` rows in `docs/packets/audits/STATUS.md` show verified
      on every non-`n-a` cell, and `n-a` is preserved on `gms_v48`/`gms_v61` for both ops — and
      on no other version for either op.
- [ ] No wire change to any already-verified version.

Gate:

- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.
