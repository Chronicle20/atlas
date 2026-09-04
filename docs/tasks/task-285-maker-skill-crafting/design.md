# Maker Skill / Item Maker — Design

Version: v1
Status: Draft
Created: 2026-08-28
Consumes: `prd.md` (v1, approved), `evidence-maker-skill-v72-v79.md`

---

## 1. Scope

This document turns the approved PRD into an implementable architecture. It covers five
surfaces: recipe ingestion in `atlas-data`, a new `atlas-maker` domain service, two codecs in
`libs/atlas-packet`, a handler in `atlas-channel`, and a craft saga in
`atlas-saga-orchestrator` (plus the one new saga action that saga requires).

It does not restate the PRD. Where this design contradicts the PRD, §2 says so explicitly and
gives the evidence; those corrections are binding on the plan.

---

## 2. Corrections to the PRD

Each correction below is grounded in repo source or a decompilation performed during this
design phase. The PRD's assumptions were reasonable; the evidence disagrees.

### C-1 — `MAKER_RESULT` is result-code-prefixed, not mode-prefixed

FR-4.3 describes `MAKER_RESULT` as a mode-prefix dispatcher with "four structurally distinct
bodies … and an enable-actions no-op". The client disagrees.

`CUserLocal::OnMakerResult` was decompiled on four IDBs spanning the full version range:

| Version | IDB session | Address |
|---|---|---|
| `gms_v72` | `99e435d8` | `0x86a152` |
| `gms_v83` | `754107bf` | `0x95dad3` |
| `gms_v95` | `ecc757f4` | `0x9102f0` |
| `jms_v185` | `a977912e` | `0xa29527` |

All four open identically:

```c
v53 = CInPacket::Decode4(a2);     // nResult
if ( v53 <= 1 )
{
  v3 = CInPacket::Decode4(v2);    // nMode
  switch ( v3 ) { case 1: case 2: ... case 3: ... case 4: ... }
}
// nResult / target / count / disassembled-id are then handed to
// CUIItemMaker::OnItemMakeResult, which re-enables the UI on every path.
```

So the first field is a **result code**, the mode is the **second** field, and the "no-op"
form the PRD posited is simply `nResult > 1` with no body at all. There are three body shapes
(`1|2` share one), not four.

**Consequence.** The op is still a legitimate dispatcher family — `DISPATCHER_FAMILY.md`
requires a config-resolved mode, not that the mode be the first field — but the family's
arms are: `CREATE` (1), `CREATE_WITH_UPGRADE` (2, body-identical to 1),
`MONSTER_CRYSTAL` (3), `DISASSEMBLE` (4), and a bodyless `FAILED` form. The result code is a
shared header field carried by every arm struct, not an arm selector.

### C-2 — the wire layout is version-invariant; no `MajorAtLeast` gate is needed

FR-4.5 mandates `MajorAtLeast` gating for "version-divergent fields". Across the four IDBs
above, the `MAKER_RESULT` read order is byte-identical — the function-size differences
(`0x660` v72, `0x6df` v83, `0x8a0` v95, `0x633` jms) come entirely from chat-log string
handling and `CUIStatusBar::ChatLogAdd` inlining, not from packet fields.

`CUIItemMaker::RequestItemMake` is likewise identical: `gms_v95` (`ecc757f4` @ `0x7d58d0`)
builds `COutPacket(&oPacket, 125)` and encodes exactly the field order recorded in
`evidence-maker-skill-v72-v79.md` for v72/v79.

**Consequence.** Both codecs are written **without any version gate**. FR-4.5 is satisfied
vacuously. If Phase 3's per-version re-derivation (still required — see §9, R-1) finds a
divergence on one of the four unsampled versions (`gms_v79`, `gms_v84`, `gms_v87`,
`gms_v92`), the gate is added then, using `MajorAtLeast`, never a raw comparison.

### C-3 — the request encodes the mode **once**, not twice

`evidence-maker-skill-v72-v79.md` renders the request as a leading `Encode4 nMode` followed
by a per-arm `Encode4 nMode // echoed`. That is a transcription artefact of showing each arm's
first field. The v95 decompilation shows a single encode, inside the switch:

```c
COutPacket::COutPacket(&oPacket, 125);
switch ( this->m_nRecipeClass ) {
  case 1u: case 2u:
    COutPacket::Encode4(&oPacket, m_nRecipeClass);   // the ONLY mode encode
    COutPacket::Encode4(&oPacket, this->m_nTargetItem);
    ...
```

There is no pre-switch encode. The corrected layout is in §5.1.

### C-4 — `atlas-data` has no per-domain tables; PRD §6 does not apply

PRD §6 specifies three relational tables (`item_makes`, `item_make_materials`,
`item_make_rewards`). `atlas-data` does not work that way. Every WZ domain persists into one
shared table defined at `services/atlas-data/atlas.com/data/document/entity.go:11-26`:

```go
type Entity struct {
    Id         uuid.UUID
    TenantId   uuid.UUID       `gorm:"uniqueIndex:idx_documents_tenant_type_docid"`
    Type       string          `gorm:"uniqueIndex:idx_documents_tenant_type_docid"`
    DocumentId uint32          `gorm:"uniqueIndex:idx_documents_tenant_type_docid"`
    Content    json.RawMessage `gorm:"type:json;not null"`
    UpdatedAt  time.Time
}
func (e Entity) TableName() string { return "documents" }
```

Writes upsert on `(tenant_id, type, document_id)` via `clause.OnConflict`
(`document/db_storage.go:120-157`), fronted by a per-tenant in-memory
`document.Registry` cache. Idempotency (FR-1.6) is therefore free and structural.

**Consequence.** The `itemmake` domain adds **no migration and no new table**. It registers
document type `"ITEM_MAKE"`, keyed by the created-item id, whose `Content` is the whole
recipe including its ordered `recipe` and `randomReward` lists. Repo conventions outrank the
PRD's speculative schema.

### C-5 — the archive carries `reqQuest`, which FR-1.2 omits

Field-name sweep of the reference `Etc.wz/ItemMake.img.xml` yields:
`catalyst, count, item, itemNum, meso, prob, randomReward, recipe, reqEquip, reqItem,
reqLevel, reqQuest, reqSkillLevel, tuc`.

`reqQuest` is an `imgdir` (quest-id → required state), not a scalar:

```xml
<imgdir name="reqQuest"><int name="21614" value="3"/></imgdir>
```

Occurrence counts in the reference archive: `catalyst` 772, `reqItem` 14, `reqEquip` 2,
`reqQuest` 2.

**Consequence.** `reqQuest` is ingested as an ordered `(questId, state)` list and **enforced**
in eligibility. Ingesting a field and then ignoring it is exactly the "documented gap" the
repo forbids when the prerequisite is producible — and it is: `atlas-quests` already answers
per-character quest state. Two recipes are affected; the cost is one more upstream read,
made only when a candidate recipe actually carries `reqQuest`.

### C-6 — the top-level group digit must be persisted

The archive's two-level tree keys its top level by the created item's leading digit
(`0/1/2/4/8/16`). FR-1.1 describes the tree but the PRD's data model discards the digit.
Mode 3 (§5.3) needs it: the leftover → crystal lookup must be scoped to group `0`, otherwise
an arbitrary recipe that happens to list the leftover as a material would match.

**Consequence.** The ingested document carries a `group` field.

---

## 3. Architecture

### 3.1 Component map

```
 client ──MAKER_SKILL──> atlas-channel
                              │  (handler: decode, resolve character)
                              ▼
                         atlas-maker  ──GET /data/item-makes/*──> atlas-data ── Etc.wz/ItemMake.img.xml
                              │        ──GET /characters/{id}──> atlas-character   (level, mesos)
                              │        ──GET .../skills────────> atlas-skills      (Maker level)
                              │        ──GET .../compartments──> atlas-inventory   (materials, slots)
                              │        ──GET /data/equipment/{id}> atlas-data      (reqLevel, disassembly)
                              ▼
                        craft saga (emit)
                              ▼
                   atlas-saga-orchestrator ──> atlas-inventory / atlas-character
                              │
                              ▼ (completion / failure event)
                         atlas-channel ──MAKER_RESULT──> client
```

`atlas-maker` is the only component that knows what a recipe means. `atlas-channel` decodes
and encodes; `atlas-data` stores; the orchestrator mutates. No service reaches past its
neighbour.

### 3.2 Craft lifecycle

1. Client sends `MAKER_SKILL`. The handler decodes the mode and arm body, resolves the
   character from the session, and `POST`s to `atlas-maker`
   `/characters/{id}/maker/crafts` with the untrusted request verbatim.
2. `atlas-maker` performs **all** validation against server-side state (§4.2), computing the
   exact material consumption plan — concrete `(compartment, slot, quantity)` tuples, not
   template ids — and the exact award.
3. On rejection, `atlas-maker` returns a JSON:API error with a stable `code` and mutates
   nothing. The handler maps the code to a `MAKER_RESULT` failure and writes it.
4. On acceptance, `atlas-maker` emits the craft saga and returns the transaction id. The
   handler writes nothing yet.
5. The orchestrator executes the saga. On terminal success or terminal compensation,
   `atlas-channel` writes the corresponding `MAKER_RESULT` arm.

Step 5 is what makes FR-5.2 ("emit a result on every path") true: the failure result comes
from the synchronous rejection *or* from the saga's terminal event, and there is no third
path. See §7 for the timeout case.

### 3.3 Why the result is written after the saga, not on acceptance

The alternative — write the success `MAKER_RESULT` immediately on validation and let the saga
run behind it — is simpler and one round-trip faster, and it is what the reference server
effectively does. It is rejected here because the create-arm body must enumerate the actual
materials and gems consumed. Reporting a consumption that a later compensation reverses puts
the client's chat log permanently out of step with the inventory, and FR-3.7 requires a
rejected craft to leave state byte-identical. Waiting for the terminal event costs latency
the client already tolerates (its UI is locked from `StartItemMake` until `OnMakerResult`).

---

## 4. Component designs

### 4.1 `atlas-data` — the `itemmake` domain

Follows the `commodity` domain shape exactly
(`services/atlas-data/atlas.com/data/commodity/`): `reader.go`, `rest.go`, `processor.go`,
`registry.go`, `resource.go`, `mock/processor.go`, plus tests. No `entity.go`, no
`administrator.go`, no migration — per C-4.

**Worker registration.** A new `WorkerItemMake` const appended to the `Workers` slice in
`services/atlas-data/atlas.com/data/data/processor.go` (const block ~L42-59, slice L61), with
a dispatch branch mirroring `WorkerCharacterCreation` (L194):

```go
} else if name == WorkerItemMake {
    err = p.RegisterFileData(path, filepath.Join("Etc.wz", "ItemMake.img.xml"),
        itemmake.NewProcessor(p.l, p.ctx, p.db).RegisterItemMake)()
}
```

It gets its own worker name rather than chaining onto `WorkerCommodity` (which already
chains CashPackage behind Commodity): chaining makes the second archive's ingestion
conditional on the first's success for no reason, and `ItemMake.img` is unrelated to
`Commodity.img`.

**Storage.** `document.NewStorage(l, db, GetModelRegistry(), "ITEM_MAKE")`, keyed by the
created-item id.

**Reader.** Iterates the six top-level `ChildNodes` (`0/1/2/4/8/16`), recording each node's
`Name` as `group`; for each entry, `strconv.Atoi` the 8-digit key into the document id, read
the scalars with `GetIntegerWithDefault(name, 0)` (satisfying FR-1.5's default-don't-fail
rule), and walk the `recipe` / `randomReward` / `reqQuest` child lists in document order. The
ordered-child-list idiom is the one used by `quest/reader.go:132-229`.

A malformed individual entry is logged and skipped; the archive continues. This is a
per-entry `continue`, not an error return, because `RegisterFileData`
(`data/processor.go:302-307`) discards the `RegisterFunc` error anyway — a returned error
would be silently swallowed and the operator would see nothing.

**RestModel.**

```go
type RestModel struct {
    Id            uint32               `json:"-"`
    Group         uint32               `json:"group"`
    ReqLevel      uint32               `json:"reqLevel"`
    ReqSkillLevel uint32               `json:"reqSkillLevel"`
    ItemNum       uint32               `json:"itemNum"`
    Tuc           uint32               `json:"tuc"`
    Meso          uint32               `json:"meso"`
    Catalyst      uint32               `json:"catalyst"`
    ReqItem       uint32               `json:"reqItem"`
    ReqEquip      uint32               `json:"reqEquip"`
    Recipe        []MaterialRestModel  `json:"recipe"`
    RandomReward  []RewardRestModel    `json:"randomReward"`
    ReqQuest      []QuestReqRestModel  `json:"reqQuest"`
}
func (r RestModel) GetName() string { return "itemMakes" }
```

`MaterialRestModel{ItemId, Count}`, `RewardRestModel{ItemId, ItemNum, Prob}`,
`QuestReqRestModel{QuestId, State}`. Absent scalars are `0`, absent lists are empty — the
PRD's "0 when absent" convention.

**Resource.** `/data/item-makes` and `/data/item-makes/{itemId}`, registered exactly like
`etc/resource.go:17-27` and wired into `main.go`'s `AddRouteInitializer` chain alongside
`commodity.InitResource(db)(GetServer())` (L192).

### 4.2 `atlas-maker` — the new service

Structured after `atlas-reward-pools`, which is the closest existing shape: a domain service
with seeded reference data, a compute-only subdomain, and no ownership of mutable game state.
Registration follows `docs/adding-a-new-service.md` in full — the k8s overlay entries
(`images:` pin, `db-name-suffix`, `ATLAS_ENV`, the generator-owned PR-overlay files) are the
silent-failure traps, and `tools/service-registration-guard.sh` is run as part of the gate.

**Packages.**

| Package | Role |
|---|---|
| `recipe` | Read-through cache of `atlas-data`'s `itemMakes`, plus the derived indexes. Compute-only: no entity, no table. |
| `reagent` | The seeded gem/reagent → stat table (§4.2.3). Full domain shape: `entity.go`, `administrator.go`, `provider.go`, `model.go`, `builder.go`, `processor.go`, `resource.go`, `rest.go`, `subdomain.go`. |
| `craft` | Validation, the consumption plan, and saga emission. Compute-only. |
| `data/…` | Per-upstream REST clients (`character`, `skills`, `compartment`, `equipment`, `quest`, `itemmake`), following the repo's per-service `requests.go` convention. |
| `seed` | Seed-catalog group registration, mirroring `reward-pools/seed/groups.go`. |

A database is still required (the `reagent` table), so the service takes the standard
`database.Connect` + `seeder.SeedState` bootstrap from
`reward-pools/main.go:41-74`.

#### 4.2.1 Recipe indexes

The recipe set is a few thousand immutable rows per tenant. `atlas-maker` builds, lazily per
tenant and invalidated on the seed/ingestion signal:

- `byItemId map[item.Id]Model` — mode 1/2 and the eligibility listing.
- `byLeftover map[item.Id]Model` — **group `0` only**; mode 3's leftover → crystal recipe.
  This is why C-6 matters.

This resolves **OQ-3**: the `ItemMake.img` group `0` directory is sufficient. Its entries
already pair the leftover as the sole `recipe` material with the crystal tiers as
`randomReward` outcomes, so no join against `atlas-drop-information` is needed. (`GET
/items/{itemId}/drops` exists at
`services/atlas-drop-information/atlas.com/dis/monster/drop/resource.go:26-27` if a future
cross-check is ever wanted; it is not a dependency of this design.)

#### 4.2.2 Eligibility evaluation

`atlas-inventory` has **no** batched all-compartments endpoint — `handleGetCompartmentByType`
requires a `type` query param (`compartment/resource.go:67-72`). The NFR's "single batched
inventory read, not one read per recipe" is therefore satisfied by reading each *inventory
type* once (EQUIP, USE, ETC) into a snapshot, then evaluating every candidate recipe against
that snapshot in memory. Three upstream calls, not one per recipe.

Order of checks, cheapest first, so the expensive reads are skipped for most recipes:

1. `reqLevel` vs `atlas-character` level, `reqSkillLevel` vs the Maker variant's level from
   `atlas-skills` (its `GET /characters/{id}/skills` returns every skill in one drained call).
   The Maker variant is any of `BeginnerMaker`/`NoblesseMaker`/`LegendMaker`/`EvanMaker`, and
   FR-3.5 requires level ≥ 1 for *any* craft, checked once up front.
2. `reqItem` / `reqEquip` against the snapshot.
3. `reqQuest` against `atlas-quests` — **only** for the few recipes that carry one (C-5).
4. Every `recipe` material at its `count`, summed across slots in the snapshot.
5. `meso` vs the character's mesos.
6. Free-slot capacity for every award (FR-3.6), computed before any mutation.

#### 4.2.3 The reagent table (resolves OQ-5)

**Decision: a tenant-owned seeded table in `atlas-maker`.**

```
reagents(tenant_id, reagent_item_id, stat, value)
```

`stat` is the affected equip stat, `value` its delta. Exposed read-only; retunable per tenant
through the seed catalog like every other seeded domain.

The **seed content is derived, not invented.** The client owns the authoritative mapping:
`CItemMakerInfo::Load_GemEffect` (`gms_v72` @ `0x5a2cf5`, `gms_v83` @ `0x5e6f4c`) reads it
out of the WZ archive. Phase 3 derives the source node and its field names from that function
and generates the seed from the archive — the same standard applied to every other game-data
value in this repo. The table is tenant-owned so an operator *can* retune it; its default
content matches the client.

#### 4.2.4 Disassembly derivation (resolves OQ-2)

**Decision: derive from the equip's level requirement.**

`GET /data/equipment/{equipmentId}` already exposes `reqLevel`
(`equipment/rest.go:33`, populated at `equipment/reader.go:103`), so no new `atlas-data`
surface is needed. The level-band → crystal-id mapping is again **derived, not invented**:
`CItemMakerInfo::Load_MonsterCrystalLevel` (`gms_v72` @ `0x5a3033`, `gms_v83` @ `0x5e728a`)
is the client's own loader for exactly this table, and Phase 3 reads the band boundaries and
crystal ids from it. The mapping lives as a small derived table alongside `reagent`, seeded
the same way, for the same retunability reason.

#### 4.2.5 API and errors

Exactly the surface in PRD §5. Two additions:

- `GET /characters/{id}/maker/recipes` accepts the standard pagination params; the eligible
  set is small but the full set is not.
- Every error carries the PRD's stable `code`. Two codes are added for the corrections above:
  `missing_prerequisite_quest` (422, from C-5) and `craft_in_progress` (409, from §7).

#### 4.2.6 No persistent craft state

Per the PRD, `atlas-maker` stores no craft rows. **OQ-4 is resolved as "no audit table"**: the
saga record in `atlas-saga-orchestrator` is the durable history, and the NFR's per-craft
info-level log — tenant, character, mode, recipe id, materials, meso delta, produced item,
correlated by saga id — is the operational record. Adding a second write path for data the
orchestrator already holds is duplication, and it can be added later without touching this
design's interfaces.

### 4.3 `libs/atlas-packet`

#### 4.3.1 `MAKER_SKILL` (serverbound)

One struct with `Encode` and `Decode`, no version gate (C-2). Corrected layout (C-3):

```
Decode4  nMode

mode 1|2   Decode4 nTargetItemID
           Decode1 bUseCatalyst
           Decode4 nGemCount
           nGemCount × Decode4 nGemItemID

mode 3     Decode4 nLeftoverItemID

mode 4     Decode4 nItemID
           Decode4 nInventoryType
           Decode4 nSlotPos
```

The mode is **not** enrolled in a dispatcher family. `DISPATCHER_FAMILY.md` scopes families to
clientbound ops and `TestFamilyCapServerboundSkipped` guards it; task-206 demoted 17 verified
cells by ignoring that. The serverbound mode instead becomes the handler's
`options.operations` routing table in each seed template, which is the documented mechanism
for serverbound mode dispatch.

Registry work first, per FR-4.0: add `MAKER_SKILL` to `docs/packets/registry/gms_v72.yaml`
(opcode 112, `ida.address: 0x760cc3`) and `gms_v79.yaml` (opcode 111, `0x795dc3`), both
`provenance: ida-discovered`, then regenerate. The other six versions are already registered
at 113/113/116/124/125/108 (v83/v84/v87/v92/v95/jms), matching FR-4.1's hex values.

#### 4.3.2 `MAKER_RESULT` (clientbound dispatcher family)

Enrolled per `DISPATCHER_FAMILY.md`: one consolidated file (`maker_result.go`), a discrete
struct per arm, a per-arm body function resolving the mode through
`atlas_packet.WithResolvedCode("operations", <fixed key const>, …)`, a
`docs/packets/dispatchers/maker_result.yaml`, and `case "CUserLocal::OnMakerResult#<Arm>":`
entries in `tools/packet-audit/cmd/run.go`'s `candidatesFromFName`.

Arms and their derived bodies. `nResult` is a shared header field on every struct.

```
Encode4  nResult                       // > 1 ⇒ stop; this is the FAILED arm
Encode4  nMode

CREATE / CREATE_WITH_UPGRADE  (mode 1, 2 — identical bodies)
  Encode1  bNoItemAwarded              // when 0, the pair below follows
    Encode4  nTargetItemID
    Encode4  nItemNum
  Encode4  nMaterialCount
    nMaterialCount × { Encode4 nItemID; Encode4 nCount }
  Encode4  nGemCount
    nGemCount × { Encode4 nItemID }
  Encode1  bCatalystUsed
    Encode4  nCatalystItemID           // only when bCatalystUsed
  Encode4  nMesoCost

MONSTER_CRYSTAL  (mode 3)
  Encode4  nCrystalItemID
  Encode4  nLeftoverItemID             // no meso field on the wire

DISASSEMBLE  (mode 4)
  Encode4  nDisassembledItemID
  Encode4  nCrystalCount
    nCrystalCount × { Encode4 nItemID; Encode4 nCount }
  Encode4  nMesoCost

FAILED
  (nResult only)
```

Two derived details worth pinning, because both are easy to get wrong:

- **The meso field is a cost, not a refund.** The client renders it as
  `Format(SP_292_YOU_HAVE_LOST_MESOS_D, -v37)` in both mode 1|2 and mode 4 — a positive wire
  value displayed as a loss. PRD FR-3.4's "meso refund" wording is therefore about a
  *separate* meso award, not this field. Disassembly's wire meso is what the operation
  **charged**.
- **`bNoItemAwarded` is inverted.** The client reads `if (!Decode1()) { id = Decode4; num =
  Decode4; }`. A truthy byte *suppresses* the pair.

`CREATE` and `CREATE_WITH_UPGRADE` get separate structs despite identical bodies, because
INV-1 forbids one struct mapped by more than one `#`-entry and the two modes are genuinely
distinct operations to the client.

### 4.4 `atlas-channel`

- A `MakerSkillHandle` handler registered in `produceHandlers()` (`main.go`, alongside
  `handlerMap[fieldsb.ItcOperationHandle]` at L1020) and added to the eight applicable seed
  templates under `services/atlas-configurations/seed-data/templates/` with its
  `options.operations` mode table. `template_gms_12_1.json`, `template_gms_48_1.json`, and
  `template_gms_61_1.json` are **not** touched — the op is genuinely `n-a` there.
- A `MakerResult` writer entry in the same eight templates with the family's `operations`
  mode table.
- The handler is `LoggedInValidator`-gated, like every other in-field op.

### 4.5 Saga

#### 4.5.1 The new action

`AwardAsset` carries `ItemPayload{TemplateId, Quantity, Period, Expiration}` and forwards to
`RequestCreateItem(...)`; `CreateAndEquipAsset` adds only a `UseAverageStats bool`. Neither
can express "an equip with `tuc` upgrade slots and reagent-adjusted stats", which FR-3.1 and
FR-3.2 require. Explicit per-stat fields exist today only on *snapshot* payloads that move an
already-existing asset between custodies (`AcceptToMtsListingPayload`,
`AcceptToParcelPayload`), never on a creation payload.

So one new action is added — the PRD's own escape hatch ("a new action type is expected only
if equip-with-`tuc`-and-gem-stats cannot be expressed by `AwardAsset`"). Proposed name
`AwardCraftedAsset`, payload shaped on `AcceptToMtsListingPayload`'s explicit stat block plus
`Slots` for `tuc`.

Wiring an action touches, all of which the plan must enumerate:

| File | Change |
|---|---|
| `libs/atlas-saga/model.go` | action constant |
| `libs/atlas-saga/payloads.go` | `AwardCraftedAssetPayload` |
| `libs/atlas-saga/unmarshal.go` | `Step[T].UnmarshalJSON` case |
| `…/saga-orchestrator/saga/model.go` | local re-export |
| `…/saga/handler.go` | `GetHandler` case + `handleAwardCraftedAsset` |
| `…/saga/event_acceptance.go` | accepted completion events |
| `…/saga/rest.go` | REST payload unmarshaller |
| `…/saga/compensator.go` | reverse-walk case, `lateCompensableActions`, `dispatchLateInverse` |
| `atlas-inventory` | creation command accepting explicit stats + slots |

The `atlas-inventory` extension is the cross-service seam this change introduces, and per
CLAUDE.md it is the part a green `verify.sh` cannot see: the plan must include a test in
`atlas-inventory` asserting the **new** contract, not the old silent drop.

#### 4.5.2 Step sequences

All three craft modes are one saga each, built with `saga.NewBuilder()`.

**Mode 1|2 (create).** `AwardMesos` (negative `Amount`, the recipe's `meso`) → one
`DestroyAssetFromSlot` **per resolved slot** for each material, gem, and the catalyst →
`AwardCraftedAsset` for an equip, or plain `AwardAsset` for a non-equip output.

`DestroyAssetFromSlot` — not `DestroyAsset` or `DestroyAllAssets` — because `DestroyAsset`
resolves a template to the *first* matching slot only (so a 5-count material spanning two
stacks silently under-consumes) and `DestroyAllAssets` is explicitly **not compensable**.
`DestroyAssetFromSlotPayload` carries an optional `TemplateId` precisely so the compensator
can re-create the asset. `atlas-maker` supplies the exact slots from its §4.2.2 snapshot,
which is also what makes the "never trust client quantities" NFR true.

**Mode 3 (crystal).** `AwardMesos` (negative, recipe `meso`) → `DestroyAssetFromSlot` for the
leftover → `AwardAsset` for the weighted `randomReward` draw.

**Mode 4 (disassemble).** `DestroyAssetFromSlot` at the client-supplied slot — *after*
`atlas-maker` has verified that slot actually holds the claimed equip — → `AwardAsset` per
derived crystal → `AwardMesos` for the charge.

#### 4.5.3 Weighted draw

Server-side only, following `atlas-reward-pools`' proven pattern
(`reward/processor.go:227-252`): `totalWeight` + `selectWeightedIndex` as a pure,
unit-testable function, with the roll drawn from `crypto/rand` — **not** `math/rand`. The
weight table is never sent to the client.

---

## 5. Open questions resolved here

| ID | Resolution |
|---|---|
| OQ-0 | Already settled in the evidence doc; §4.3.1 carries it into the registry work. |
| OQ-1 | **Unresolvable on this machine.** Only a GMS 83.1 XML dump exists locally; there is no second version to diff a field set against. Handled as risk R-2, not as a blocker. |
| OQ-2 | Derive from equip `reqLevel`; band table derived from `Load_MonsterCrystalLevel`. §4.2.4. |
| OQ-3 | `ItemMake.img` group `0` alone is sufficient; no drop-table join. §4.2.1. |
| OQ-4 | No audit table; the saga record plus the correlated info log is the history. §4.2.6. |
| OQ-5 | Tenant-owned seeded table in `atlas-maker`, seeded from the node read by `Load_GemEffect`. §4.2.3. |
| OQ-6 | Out of scope by the PRD's own non-goals; Maker arrives through whatever skill-award path the tenant already uses. No design surface. |

### New open question

**OQ-7 — mode 3 leftover quantity.** All four decompiled clients render the mode-3
consumption log as `Format(SP_293_YOU_HAVE_LOST_ITEMS_S_D, <name>, 100)` — the literal `100`
is hard-coded in the client. The reference archive's group-`0` recipe lists its leftover
material with `count: 1`. The wire carries only the item id, so the discrepancy is invisible
to the protocol.

**Decision: consume 100.** A client whose chat log says "lost 100" while the server removed 1
is a visible, exploitable inconsistency, and 100 is the quantity every client build agrees
on. The recipe's `count` is ignored for group `0`. Phase 3 should confirm this against the
reference server's crystal path and reverse the decision if it disagrees — this is recorded
so the plan can check it cheaply, not left as an unowned gap.

---

## 6. Testing

| Surface | Approach |
|---|---|
| `itemmake` reader | Synthetic `xml.Node` fixtures built inline in Go, as `commodity/reader_test.go` does. No `ItemMake.img.xml` ships in the repo (`ZIP_DIR` is externally provisioned), so a file-based fixture is not an option. Cover: each of the six groups, `randomReward` present/absent, `catalyst`/`reqItem`/`reqEquip`/`reqQuest` present/absent, and a malformed entry that is skipped without aborting. |
| `itemmake` resource | Round-trip an entry from each top-level group, asserting scalars and **list order** — FR-1.3/1.4 are ordered lists and an order regression is otherwise silent. |
| Idempotency | Re-ingest and assert row identity. Structural under C-4, but pinned so a future storage change cannot regress FR-1.6 unnoticed. |
| `atlas-maker` eligibility | One test per exclusion reason — level, skill level, material, `reqItem`, `reqEquip`, `reqQuest`, meso, inventory-full — and one per error `code` in §4.2.5. Builders, never `*_testhelpers.go`. |
| Weighted draw | `selectWeightedIndex` tested directly across the cumulative ranges; no `crypto/rand` stubbing. |
| Codecs | Byte fixtures per version per arm, with `// packet-audit:verify packet=… version=… ida=0x…` markers, derived from each version's own IDB. |
| Failure atomicity | Assert pre- and post-request character state is identical for every §4.2.5 rejection — the FR-3.7 acceptance criterion, asserted rather than reasoned about. |
| Cross-service seam | A test in `atlas-inventory` asserting the new explicit-stats creation contract (§4.5.1). |

Gates: `packet-audit dispatcher-lint`, `matrix --check`, `fname-doc --check`,
`operations --check`, `tools/service-registration-guard.sh`, and flagless `tools/verify.sh`.

---

## 7. Error handling, atomicity, anti-duplication

- **Rejection is pre-mutation.** All validation completes, and the full consumption plan is
  computed, before a single saga step is emitted. A rejected craft cannot have mutated
  anything because nothing was emitted.
- **Compensation.** Every step in every sequence in §4.5.2 uses a compensable action.
  `DestroyAllAssets` is excluded for exactly this reason.
- **Replay.** `atlas-maker` holds a per-`(tenant, characterId)` in-flight guard, taken when a
  craft saga is emitted and released on its terminal event. A second `MAKER_SKILL` arriving
  while one is in flight returns `craft_in_progress` (409) and the handler writes a
  `FAILED` result. This is in-memory, consistent with §4.2.6's "no persistent craft state" —
  it is a duplicate-suppression window, not durable state, and a restart losing it degrades
  to the ordinary validation path, which is still server-authoritative and cannot
  double-spend a material that is no longer there.
- **Saga timeout.** If the saga neither completes nor compensates within its timeout, the
  orchestrator's terminal timeout event drives a `FAILED` result. The client UI is
  re-enabled on every `OnMakerResult` regardless of `nResult`, so no path leaves it locked
  (FR-5.2).

---

## 8. Alternatives considered

**Fold maker into `atlas-channel` instead of a new service.** Cheapest by far: no
registration, no overlays, no GHCR package. Rejected — recipe eligibility needs reads from
five upstreams and a seeded reagent table, which is a domain, not a handler. `atlas-channel`
would grow a database.

**Put recipes in `atlas-maker` rather than `atlas-data`.** Would remove one hop. Rejected —
`atlas-data` owns WZ ingestion for every other archive, and FR-1.1 is explicit that recipes
come from the tenant's own archive through the existing pipeline. Splitting WZ ownership
across two services to save a hop is the wrong trade.

**Relational recipe tables (the PRD's §6).** Rejected per C-4: it contradicts the
`documents`-table pattern every other WZ domain uses, and would be the only per-domain schema
in `atlas-data`.

**Extend `AwardAsset` with optional stat fields instead of a new action.** Tempting — one
fewer action. Rejected: `AwardAsset` is used by dozens of call sites and its compensation
path is `RequestDestroyItem(templateId, quantity)`. Widening its payload widens the blast
radius of every one of those, for a field only maker sets. A discrete action keeps the
compensation semantics of both narrow.

**Write `MAKER_RESULT` on validation success rather than on saga completion.** Covered in
§3.3; rejected because the create-arm body enumerates real consumption.

---

## 9. Risks

- **R-1 — four versions unsampled.** `gms_v79`, `gms_v84`, `gms_v87`, and `gms_v92` were not
  decompiled during this design. C-2's "no version gate" claim rests on four samples spanning
  the range (v72, v83, v95, jms_v185), not on all eight. Phase 3 re-derives every version per
  `docs/packets/PROCESS.md` before writing the codec; a divergence turns into a
  `MajorAtLeast` gate and costs a per-arm struct field, not a redesign.
- **R-2 — OQ-1 unanswerable locally.** Only the GMS 83.1 dump exists on this machine, so
  "does every target client ship `ItemMake.img`, with the same field set?" cannot be
  answered here. Mitigation is already structural: FR-1.5's default-don't-fail reading means
  an archive missing a field ingests with zeros, and a tenant whose archive lacks the file
  ingests an empty recipe set rather than failing startup. This is a genuine external
  blocker — the other versions' WZ dumps are not on disk — and is surfaced rather than
  guessed at.
- **R-3 — new-service registration is silent-failure-prone.** A missing `images:` entry
  pins the deployment to `:latest` forever with no error; a `configMapGenerator` with
  `behavior: replace` drops unlisted topic keys per environment; a first GHCR push lands
  private and yields a permanent `ImagePullBackOff`. `service-registration-guard.sh` catches
  some of these and not others; the plan carries the by-hand checks explicitly.
- **R-4 — OQ-7.** See §5; a wrong choice is a visible quantity mismatch, and the check is
  cheap.

---

## 10. Suggested implementation sequencing

Ordered so each stage is independently verifiable and nothing is blocked on a later stage.

1. Registry correction (FR-4.0) — `gms_v72` / `gms_v79` `MAKER_SKILL` entries, matrix
   regenerated, `n-a` preserved on `gms_v48` / `gms_v61`. Standalone and mergeable alone.
2. `atlas-data` `itemmake` domain — reader, storage, resource, tests.
3. Per-version wire re-derivation for both ops across all eight versions (R-1), producing the
   evidence records the codecs are written against.
4. `libs/atlas-packet` — `MAKER_SKILL` codec; `MAKER_RESULT` family with its
   `dispatchers/maker_result.yaml` and `run.go` entries; byte fixtures.
5. `libs/atlas-saga` + `atlas-saga-orchestrator` + `atlas-inventory` — the
   `AwardCraftedAsset` action, end to end including compensation and the seam test.
6. `atlas-maker` — service registration, `reagent` seeded table, recipe indexes, eligibility,
   craft validation, saga emission.
7. `atlas-channel` — handler, writer, eight seed templates.
8. Gates: dispatcher-lint / matrix / fname-doc / operations, service-registration-guard,
   flagless `tools/verify.sh`, then code review.

Stage 3 is derivation-heavy and warrants `model: opus` in `plan.md`; every other stage is
Sonnet work.
