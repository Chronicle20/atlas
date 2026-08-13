# Pet Name Tag (5170000) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-13
---

## 1. Overview

The Pet Name Tag (item `5170000`, WZ path `Item.wz/Cash/0517`, item classification `517`)
is a cash-slot consumable that lets a player rename a pet they own. It is one of the
"remaining one-off cash types" catalogued in
`docs/research/missing-features/items-and-consumables.md:40` — items whose classification is
already recognized by the channel's cash-slot mapper but which have no behavior wired up.

Today the item does nothing. Two independent defects sit between the player and a rename:

1. **The dispatch arm is missing.** `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
   has no `CashSlotItemType` constant for 17 and no handler arm, so a use falls through to
   the `l.Warnf("… of type [%d]")` at `character_cash_item_use.go:679`.
2. **atlas-pets has no rename path at all.** The pet name is a construction-time-only field
   (`services/atlas-pets/atlas.com/pets/pet/model.go:12`, getter at `:34`) with no `SetName`
   on the model or builder, no `updateName` administrator function
   (contrast the six that do exist at `pet/administrator.go:39,57,75,93,111,129`), no
   `RENAME`/`SET_NAME` Kafka command (`pets/kafka/message/pet/kafka.go:13-21`), no
   `NAME_CHANGED` status event (`:86-97`), and no mutating REST endpoint
   (`pet/resource.go:18-30` is GET + POST-create only).

Unlike task-220 (Meso Sack), this feature also carries a **packet workload**. The clientbound
`PET_NAMECHANGE` op (`CPet::OnNameChanged`) is registered in the opcode tables for nine of the
ten matrix versions but is `❌` in every column (`docs/packets/audits/STATUS.md:233`), and no
codec exists anywhere under `libs/atlas-packet/pet/clientbound/`. The serverbound sub-body
likewise has no model in `libs/atlas-packet/cash/serverbound/`.

### 1.1 Client behavior — derived from the GMS v83 IDB, not assumed

The following is read from the v83 client (`MapleStory_dump.exe.i64`), not inferred:

**Cash-slot type resolution.** `get_cashslot_item_type` @ `0x48645b`:

```c
case 517:
  return a1 % 10000 != 0 ? 0 : 17;
```

So `5170000 → 17`. `get_consume_cash_item_type` @ `0x4863d5` passes 17 through
(it falls in the `result >= 12 && result <= 32` band), which routes the item to
`CWvsContext::SendConsumeCashItemUseRequest` @ `0xa0a63f` rather than to
`SendCashSlotItemUseRequest` or `SendEtcCashItemUseRequest`.

> **Defect this exposes.** The Go mirror of that rule at
> `character_cash_item_use.go:937` reads `if 10000*itemId/10000 != itemId { return 0 }`.
> `item.Id` is `uint32`, so `10000 * 5170000` overflows (`51,700,000,000 mod 2^32 = 160,392,448`;
> `/10000 = 16039 ≠ 5170000`) and the arm returns `0` instead of the pet-name-tag type.
> The client's actual predicate is `itemId % 10000 != 0`. This must be fixed for the
> feature to dispatch at all, and the fix is in scope.

**The case-17 arm** (v83 `0xa0ba15`–`0xa0bd2b`, labelled by IDA as `jumptable 00A0A6E6 case 17`)
executes, in order:

| Step | Evidence |
|---|---|
| Read character data, resolve the target pet's item, fetch its display name | `GetCharacterData` `0xa0ba23`, `sub_46D2D5` `0xa0ba47`, `CItemInfo::GetItemName` `0xa0ba6a` |
| Confirmation prompt (StringPool id `0x319`) | `0xa0ba90` → `CUtilDlg::YesNo` `0xa0baa2` |
| Name input dialog (StringPool id `0x31A`) | `0xbaf2` → `SetUtilDlgEx` `0xa0bb19`, `DoModal` `0xa0bb4d`, `GetInputStr_Result` `0xa0bb68` |
| Profanity/charset filter on the entered name | `CCurseProcess::ProcessString` `0xa0bb9a` |
| Second confirmation showing the new name (StringPool id `0x31C`, `Format`) | `0xa0bc35`, `Format` `0xa0bc5e`, `YesNo` `0xa0bc88` |
| **Encode the name — the only `Encode*` call in the entire arm** | `COutPacket::EncodeStr` `0xa0bcb5` |
| Client-side rejection notice (StringPool id `0x31B`) | `0xa0bcf4` → `CUtilDlg::Notice` `0xa0bd06` |

Two consequences follow directly, and they shape the whole design:

- **The serverbound sub-body is a single `EncodeStr(name)`.** No pet index, no pet locker SN,
  no pet slot is written. The `SetUtilDlgEx_Pet` pet-picker dialog (`0x9acb27`) is *not* called
  from `SendConsumeCashItemUseRequest` at all — its only three callers are
  `CDraggableItem::OnDoubleClicked` (pet equip), `CScriptMan::OnAskPet`, and
  `CScriptMan::OnAskPetAll`. **The server must resolve which pet is being renamed.**
- **The client's only rejection UI in this arm is client-side** (the curse-filter `Notice`).
  There is no server-driven error op in the case-17 path. A server-side rejection therefore
  has to reuse an existing generic message op — nothing purpose-built exists to reach for.
  See §9 OQ-2.

**Clientbound body.** `CPet::OnNameChanged` @ `0x704801` (v83) decompiles to:

```c
v3 = CInPacket::DecodeStr(a2, v18);   // new name → this+38
ZXString<char>::operator=(this + 38, v3);
if ( CInPacket::Decode1(a2) )         // flag: load the name-tag layer
  v5 = *(*(this + 34) + 16);
else
  v5 = 0;
... CLife::MakeNameTag(this, v7, ..., /*1003*/ ...);
```

So the body is `str name` + `byte flag`, where the flag selects whether a decorative layer is
passed to `MakeNameTag`. This matches the note already recorded in
`docs/packets/registry/gms_v84.yaml:1174`. The packet carries no pet id — the receiver is the
`CPet` instance the packet is routed to, which means the wire framing for the pet identity is
whatever the version's `CPet` dispatch uses and must be derived per version (§4, FR-6).

### 1.2 WZ data — nothing to ingest

`Item.wz/Cash/0517.img.xml` in the v83 tree contains exactly one node — `05170000/info` with
`canvas icon`, `canvas iconRaw`, `int slotMax = 1`, `int cash = 1`. There is no value node
analogous to task-220's `info/meso`. Consequently:

- No `services/atlas-data/atlas.com/data/cash/reader.go` change is required.
- No `Cash` `RestModel` field is required in atlas-data or in the atlas-channel view model.
- **There is no WZ re-ingest prerequisite**, which is the one operational burden task-220 carried
  (`docs/tasks/task-220-meso-sack-cash-item/rollout.md:11-19`).

The per-version WZ trees for the other nine versions must be re-checked during design before
this is asserted as universal (§9 OQ-1).

## 2. Goals

Primary goals:

- A player who owns at least one pet can use a Pet Name Tag from their cash slot and set a new
  name on that pet.
- The rename persists in atlas-pets and survives despawn/respawn, channel change, and relog.
- Every observer in the map sees the new name via `PET_NAMECHANGE`, and observers who enter the
  map later see it via the existing `PetActivated` spawn body
  (`services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go:143,147`).
- The name tag is consumed **only** after the rename has been confirmed applied.
- `PET_NAMECHANGE` is implemented and verified across all applicable matrix versions, promoting
  the `docs/packets/audits/STATUS.md:233` row off `❌`.

Non-goals:

- Renaming anything other than a pet (character, guild, and pet-*equipment* renames are separate).
- Any UI for pet naming in atlas-ui.
- The other unimplemented one-off cash types in
  `docs/research/missing-features/items-and-consumables.md:33-49`.
- Changing how a pet's *initial* name is assigned at purchase
  (`services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:166-187`).
- Widening the 13-character name limit (see FR-4.1).

## 3. User Stories

- As a player, I want to use a Pet Name Tag on my pet so that it displays a name I chose
  instead of the default `"Pet"` or the WZ species name.
- As a player, I want the rename to be visible to everyone in the map immediately, so the name
  is a social signal and not a private one.
- As a player, I want my name tag *not* consumed when the rename fails, so a rejected name does
  not cost me a cash item.
- As a player with multiple pets, I want a predictable rule for which pet gets renamed, so the
  outcome is never a surprise.
- As an operator, I want a rejected rename to leave no partial state — no half-applied name, no
  orphaned consumption — so support tickets do not require manual DB repair.

## 4. Functional Requirements

### FR-1 — Cash-slot dispatch

- **FR-1.1** `character_cash_item_use.go` MUST define `CashSlotItemTypePetNameTag = 17`
  alongside the existing constants at `:683-724`.
- **FR-1.2** The `ClassificationPetImprints` arm of `GetCashSlotItemType`
  (`character_cash_item_use.go:936-941`) MUST use the predicate `itemId%10000 != 0 → 0`,
  matching `get_cashslot_item_type` @ `0x48645b` `case 517`. The current
  `10000*itemId/10000 != itemId` form overflows `uint32` and MUST be removed.
- **FR-1.3** A unit test MUST assert `GetCashSlotItemType(t)(5170000) == 17` and that a
  non-multiple-of-10000 id in the 517 range (e.g. `5170001`) maps to `0`. This test MUST fail
  against the pre-fix predicate.
- **FR-1.4** A new handler arm for type 17 MUST be added to the `if it == …` chain, following the
  task-220 shape: the arm decodes the sub-body and delegates to a `handlePetNameTagUse` in a new
  sibling file `character_cash_item_use_pet_name_tag.go`.
- **FR-1.5** The existing ownership guard (`character_cash_item_use.go:56-60`, seam
  `cashItemInSlotFunc` at `:730`) MUST continue to gate the arm — the CASH slot named in the
  request must actually hold template `5170000`.

### FR-2 — Serverbound codec

- **FR-2.1** A new sub-body type MUST be added to `libs/atlas-packet/cash/serverbound/`
  (suggested `item_use_pet_name_tag.go`), modelled on `item_use_pet_skill.go:12-38` including
  its IDA-derivation doc comment.
- **FR-2.2** For GMS v83 the body after the common `ItemUse` header is **exactly one
  length-prefixed string** (the new name) — the sole `Encode*` in the case-17 arm is
  `EncodeStr` @ `0xa0bcb5`. The codec MUST NOT invent a pet identifier field.
- **FR-2.3** The common header handling — including the `cashsb.UpdateTimeFirst(t)` version gate
  at `character_cash_item_use.go:51` / `libs/atlas-packet/cash/serverbound/item_use.go:22` —
  MUST be reused unchanged.
- **FR-2.4** The body MUST be re-derived from each version's own IDB before that version's codec
  is gated. It MUST NOT be assumed identical to v83. Divergence MUST be expressed with the
  `MajorAtLeast` idiom, never a raw `> N` comparison.

### FR-3 — Pet resolution

- **FR-3.1** Because the request carries no pet identifier (FR-2.2), the server MUST resolve the
  target pet itself. The rule MUST be: **the character's lead active pet** — the first entry of
  the character's active-pet set, i.e. the pet occupying the lowest active slot.
- **FR-3.2** If the character has no active pet, the use MUST be rejected per FR-7.3 and the
  name tag MUST NOT be consumed.
- **FR-3.3** The resolved pet MUST be owned by the requesting character. A pet resolved from any
  other character's state is a fail-closed rejection.
- **FR-3.4** The chosen rule MUST be documented in the handler as a comment citing this PRD,
  because it is a server-side policy decision the wire format does not constrain.

### FR-4 — Name validation

- **FR-4.1** The accepted name MUST be **1 to 13 characters** inclusive. 13 is the DB ceiling
  (`services/atlas-pets/atlas.com/pets/pet/entity.go:23`, `gorm:"size:13"`). Names longer than 13
  MUST be rejected, never silently truncated — a truncating write would produce a name the
  player did not choose and, on some drivers, a runtime error rather than a clean failure.
- **FR-4.2** The name MUST be trimmed of leading and trailing whitespace before length
  validation, mirroring the client's `TrimRight`/`TrimLeft` (`0xa0aac6`/`0xa0aacd` in the
  adjacent arm) and its empty-after-trim rejection.
- **FR-4.3** An empty or whitespace-only name MUST be rejected.
- **FR-4.4** Validation MUST be enforced server-side regardless of the client's own
  `CCurseProcess::ProcessString` pass (`0xa0bb9a`) — a modified client can bypass it entirely.
- **FR-4.5** No profanity filter is in scope for this task. There is no existing profanity
  service in the repo to reuse, and inventing one is a separate concern. The charset/length
  rules above are the whole server-side contract. This is a deliberate scope boundary, not an
  oversight — see §9 OQ-3.

### FR-5 — Rename in atlas-pets

- **FR-5.1** A `RENAME` command MUST be added to `pets/kafka/message/pet/kafka.go:13-21`
  carrying the pet id, the requesting character id, and the new name.
- **FR-5.2** A `NAME_CHANGED` status event MUST be added to `kafka.go:86-97` with a producer in
  `pet/producer.go`, carrying the pet id, owner character id, and new name.
- **FR-5.3** An `UpdateName` administrator function MUST be added to `pet/administrator.go`
  following the existing update-function shape (`:39,57,75,93,111,129`).
- **FR-5.4** `pet.Model` MUST gain a `SetName` builder method
  (`pet/builder.go`) so `Clone` and the update path do not have to reconstruct through the
  constructor.
- **FR-5.5** The rename MUST be idempotent: applying the same name twice MUST NOT error and
  MUST emit the status event both times (Kafka is at-least-once; a redelivered `RENAME` must not
  corrupt state — see the known redelivery pattern in project memory).
- **FR-5.6** The command handler MUST re-validate FR-4's rules. atlas-pets MUST NOT trust
  atlas-channel to have validated.

### FR-6 — Clientbound `PET_NAMECHANGE`

- **FR-6.1** A new codec MUST be added at `libs/atlas-packet/pet/clientbound/name_changed.go`
  with **both** `Encode` and `Decode`, immutable struct + builder per project convention.
- **FR-6.2** The v83 body is `str name` + `byte flag`, derived from `CPet::OnNameChanged`
  @ `0x704801`. The `flag` selects whether the name-tag layer is loaded before
  `CLife::MakeNameTag`.
- **FR-6.3** The framing that identifies *which* `CPet` the packet applies to is not visible in
  the `OnNameChanged` body itself — it is applied by the caller's dispatch. The per-version
  framing MUST be derived from each IDB during design and recorded in the codec's doc comment.
  It MUST NOT be guessed.
- **FR-6.4** A `PetNameChangedWriter` MUST be registered in
  `services/atlas-channel/atlas.com/channel/main.go` alongside the existing pet writers
  (`main.go:724-732`).
- **FR-6.5** The writer MUST be driven by an atlas-channel consumer of the FR-5.2
  `NAME_CHANGED` status event, and MUST broadcast to every character in the pet owner's map.
- **FR-6.6** The value written for `flag` MUST be config-resolved or justified in a comment, per
  DOM-25 (`feedback_client_wire_values_config_resolved`) — it MUST NOT be a bare literal with no
  provenance.

### FR-7 — Consumption saga

- **FR-7.1** A new saga type `PetNameTagUse` MUST be added to `libs/atlas-saga/model.go` and
  re-exported in `services/atlas-channel/atlas.com/channel/saga/model.go`, following the
  task-220 `MesoSackUse` precedent (`libs/atlas-saga/model.go:44`).
- **FR-7.2** The saga MUST order its steps **rename first, consume second** — `rename_pet`
  followed by `consume_pet_name_tag` (`saga.DestroyAsset`, by template). This is the inverse of
  task-220's consume-then-award ordering and is a deliberate requirement: a failed rename must
  not cost the player the item.
- **FR-7.3** Every pre-flight rejection (no active pet per FR-3.2, ownership failure per FR-3.3,
  name invalid per FR-4) MUST fail closed **before** the saga starts, consuming nothing, and MUST
  call `enableActions()` so the client is unlocked — the same fail-closed guard shape as
  `character_cash_item_use_meso_sack.go:83`.
- **FR-7.4** If `rename_pet` succeeds but `consume_pet_name_tag` fails, the orchestrator MUST
  compensate by reverting the pet's name to its prior value. The prior name MUST therefore be
  captured before the rename step.
- **FR-7.5** Compensator, producer, model, and timer registrations MUST be added in
  `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/` mirroring the task-220
  additions.
- **FR-7.6** The saga MUST NOT be unlocked by anything that warps the character — see
  `reference_exclrequest_unlock_contract`. This flow does not warp, so a plain
  `enableActions()` unlock is correct.

### FR-8 — Tenant socket templates

- **FR-8.1** The new `PetNameChanged` writer MUST be registered in every applicable tenant
  socket-config template under `services/atlas-configurations/seed-data/templates/`, at each
  version's registered `PET_NAMECHANGE` opcode:

  | Version | Opcode | Source |
  |---|---|---|
  | gms_v48 | — | n/a (`docs/packets/audits/support/gms_v48.md:462`) |
  | gms_v61 | `0x083` | registry |
  | gms_v72 | `0x09D` | registry |
  | gms_v79 | `0x0A1` | registry |
  | gms_v83 | `0x0AC` | `docs/packets/registry/gms_v83.yaml:878` |
  | gms_v84 | `0x0B0` | `docs/packets/registry/gms_v84.yaml:1167` |
  | gms_v87 | `0x0B9` | registry |
  | gms_v92 | `0x0C8` | registry |
  | gms_v95 | `0x0CB` | `docs/packets/registry/gms_v95.yaml:1044` |
  | jms_v185 | `0x0B2` | `docs/packets/registry/jms_v185.yaml:885` |

  These values MUST be re-confirmed against the registry at implementation time rather than
  copied from this table.
- **FR-8.2** Every writer entry MUST carry an `fname` — a seed writer without one is silently
  dropped (`bug_seed_template_writers_require_fname`).
- **FR-8.3** Entries MUST be inserted at their sorted `opCode` position, not appended next to
  semantically-related entries — `tools/template-opcode-order-guard.sh` enforces this.
- **FR-8.4** The serverbound handler binding for `CharacterCashItemUseHandle` already exists in
  every template and MUST NOT be duplicated; adding a second binding for the same
  `(implementation, opCode)` pair is banned by
  `tools/template-duplicate-binding-guard.sh`.
- **FR-8.5** A live tenant whose config predates this change will silently drop the new writer
  (`bug_new_opcodes_not_in_live_tenant_config`). Reconciling live tenant socket configs to the
  updated templates is a rollout step and MUST be documented in `rollout.md` at execution time.

### FR-9 — Coverage matrix

- **FR-9.1** `pet/clientbound/PET_NAMECHANGE` MUST promote from `❌` to `✅` in every applicable
  column of `docs/packets/audits/STATUS.md:233`, via the single-cell procedure in
  `docs/packets/audits/VERIFYING_A_PACKET.md` — byte fixture with a `packet-audit:verify`
  marker, pinned evidence record, regenerated matrix.
- **FR-9.2** `gms_v48` MUST be recorded `n-a` and MUST pass the n-a consistency gate.
- **FR-9.3** The new serverbound sub-body MUST likewise be verified per version.
- **FR-9.4** A round-trip fixture alone is NOT sufficient evidence
  (`bug_matrix_roundtrip_fixture_false_verify`) — each cell's read order MUST be derived from
  that version's client.

## 5. API Surface

### 5.1 atlas-pets REST

`pet/resource.go:18-30` currently registers GET and POST-create only. This task adds:

```
PATCH /pets/{petId}
```

- Request: JSON:API document, resource type `pets`, single writable attribute `name`.
- Response: `200` with the updated `pets` resource.
- `400` — name fails FR-4 validation.
- `404` — pet does not exist for the tenant.
- `403` — pet is not owned by the requesting character (when a character context is supplied).

The PATCH endpoint is a convenience/admin surface. **The gameplay path is Kafka, not REST** —
atlas-channel emits `RENAME` (FR-5.1) through the saga and never calls this endpoint. The
endpoint exists so operators can correct a name without a direct DB write.

### 5.2 Kafka

**Command** — `COMMAND_TOPIC_PET`, type `RENAME`:

```
{ petId, characterId, name }
```

**Status event** — `EVENT_TOPIC_PET_STATUS`, type `NAME_CHANGED`:

```
{ petId, characterId, name }
```

Both MUST carry tenant headers. `TenantHeaderDecorator` silently drops headers if misconfigured
(`bug_config_status_event_missing_tenant_headers`) — the design phase MUST confirm the producer
path decorates correctly.

### 5.3 Packets

- **Serverbound**: existing `CharacterCashItemUseHandle` opcode, new type-17 sub-body
  (`str name`).
- **Clientbound**: new `PET_NAMECHANGE` writer, body `str name` + `byte flag`, per-version
  opcode per FR-8.1.

## 6. Data Model

No schema migration is required.

- `pets.name` already exists: `services/atlas-pets/atlas.com/pets/pet/entity.go:23`,
  `Name string \`gorm:"size:13;not null"\``.
- The 13-character ceiling is the binding constraint behind FR-4.1 and is **not** being widened
  in this task. Widening it would require a migration plus a re-derivation of the client's own
  input-field cap in every version, which is out of scope.
- Tenant scoping is unchanged — atlas-pets rows are already tenant-scoped and the rename path
  adds no new table.
- The saga's compensation (FR-7.4) needs the prior name, which is read from the existing row
  and held in saga state. No new persisted column.

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-channel** | New `CashSlotItemTypePetNameTag` constant; fixed `GetCashSlotItemType` 517 predicate (FR-1.2); new type-17 dispatch arm; new `character_cash_item_use_pet_name_tag.go` handler; new `PetNameChanged` writer registration in `main.go`; new consumer arm for the `NAME_CHANGED` status event; saga type re-export. |
| **atlas-pets** | `SetName` on builder; `UpdateName` administrator; `RENAME` command + consumer; `NAME_CHANGED` event + producer; `PATCH /pets/{petId}` resource. |
| **atlas-saga-orchestrator** | New `PetNameTagUse` saga type; `rename_pet` step; compensator that restores the prior name; producer, model, and timer registrations. |
| **libs/atlas-packet** | New `cash/serverbound/item_use_pet_name_tag.go`; new `pet/clientbound/name_changed.go`. |
| **libs/atlas-saga** | New `PetNameTagUse` saga type constant. |
| **atlas-configurations** | `PetNameChanged` writer added to nine seed templates at each version's opcode. |
| **atlas-data** | **None.** WZ node `Cash/0517` carries no parseable value (§1.2). |
| **atlas-ui** | **None.** |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every new processor, consumer, and producer MUST resolve tenant via
  `tenant.MustFromContext(ctx)`. Per-version opcode values MUST come from tenant socket config,
  never from a hard-coded table in Go.
- **Version awareness.** Any version-divergent wire behavior MUST use the `MajorAtLeast` idiom.
  Raw `> N` comparisons are banned — `bug_majorversion_gt83_is_off_by_one_v87` is the standing
  example of why.
- **Idempotency.** FR-5.5. Kafka is at-least-once; a redelivered `RENAME` MUST be safe. A
  redelivered *consume* step is the more dangerous half — the saga must not destroy two name tags
  for one rename.
- **Observability.** The handler MUST log the resolved pet id, the requesting character, and the
  rejection reason on every fail-closed path. The current silent `Warnf` fallthrough is exactly
  the failure mode this feature is fixing; the replacement must not be equally silent.
- **Concurrency.** `ForEachInMap` on the channel side is parallel
  (`bug_channel_foreachinmap_parallel_shared_state`) — the FR-6.5 broadcast MUST NOT mutate
  shared state inside the iteration callback.
- **No stubs.** Per project policy, no `// TODO`, stubbed handler, or 501 may land. Every version
  in scope is implemented or explicitly recorded `n-a`.

## 9. Open Questions

- **OQ-1 — Per-version WZ parity.** §1.2 establishes that `Cash/0517.img.xml` in the v83 tree has
  no parseable value node. The other nine version trees MUST be checked during design before the
  "no atlas-data change" conclusion is generalized. If any version adds a value node, FR-9 gains
  an atlas-data reader change and a re-ingest prerequisite.
- **OQ-2 — Server-side rejection UX.** The v83 case-17 arm has no server-driven error path; the
  only rejection UI is the client's own curse-filter `Notice` (`0xa0bd06`). So a server rejection
  can either (a) unlock silently via `enableActions()`, matching task-220's guard shape, or
  (b) reuse a generic status/notice op to surface a reason. The user's stated preference is a
  real error message; **design MUST identify a concrete existing message op and confirm it
  renders on every version in scope, or fall back to (a) and say so explicitly.** Do not ship a
  message op that renders on v83 and is a no-op elsewhere.
- **OQ-3 — Profanity filtering.** FR-4.5 scopes it out. Confirm this is acceptable, or open a
  separate task — the client already filters (`CCurseProcess::ProcessString`), so the gap is only
  exploitable by a modified client.
- **OQ-4 — Multi-pet disambiguation.** FR-3.1 picks the lead active pet. The client shows a
  confirmation naming the target before sending (`0xa0baa2`), so the player is told which pet is
  being renamed — but the *client's* choice of pet and the *server's* FR-3.1 rule must agree.
  Design MUST confirm what `sub_46D2D5` @ `0xa0ba47` resolves, and align FR-3.1 to it if they
  differ.
- **OQ-5 — Client input-field cap.** FR-4.1 caps at 13 per the DB column. The client's own input
  dialog cap is passed to `sub_9AC7CB` @ `0xa0bb2f` and was not read. If the client permits more
  than 13, every over-length attempt becomes a server rejection the player cannot predict from
  the UI. Design MUST read that argument and note the mismatch if one exists.

## 10. Acceptance Criteria

**Dispatch**
- [ ] `GetCashSlotItemType(t)(5170000) == 17` with a unit test that fails against the pre-fix
      overflow predicate (FR-1.2, FR-1.3).
- [ ] Using a Pet Name Tag no longer reaches the `Warnf` fallthrough at
      `character_cash_item_use.go:679`.

**Rename**
- [ ] A player with an active pet can rename it; the new name appears immediately for the player
      and every other character in the map.
- [ ] The name persists across pet despawn/respawn, channel change, and relog.
- [ ] A later observer entering the map sees the new name in the `PetActivated` spawn body.
- [ ] `PATCH /pets/{petId}` updates the name and returns the updated JSON:API resource.

**Validation & failure**
- [ ] Names of 1–13 characters are accepted; 14+ is rejected and **not** truncated.
- [ ] Leading/trailing whitespace is trimmed; an empty-after-trim name is rejected.
- [ ] Every rejection leaves the name tag in the player's cash slot.
- [ ] Every rejection unlocks the client (no soft-lock).
- [ ] A player with no active pet gets a clean rejection and keeps the item.
- [ ] Validation holds when the request bypasses the client filter (test at the Kafka/handler
      layer, not through a real client).

**Saga**
- [ ] The name tag is consumed only after the rename is confirmed applied (FR-7.2).
- [ ] A forced failure of the consume step restores the pet's prior name (FR-7.4).
- [ ] A redelivered `RENAME` command does not corrupt state and does not consume a second tag.

**Packets & templates**
- [ ] `libs/atlas-packet/pet/clientbound/name_changed.go` has both `Encode` and `Decode`.
- [ ] `PET_NAMECHANGE` is `✅` in all nine applicable columns of
      `docs/packets/audits/STATUS.md:233`, and `n-a` for `gms_v48`.
- [ ] The serverbound type-17 sub-body is verified per version.
- [ ] Each cell's evidence record is pinned and the matrix regenerated.
- [ ] Nine seed templates carry the `PetNameChanged` writer with an `fname`, at the correct
      per-version opcode, in sorted `opCode` position.

**Build & guards** (per CLAUDE.md §Build & Verification)
- [ ] `go test -race ./...` clean in every changed module.
- [ ] `go vet ./...` clean in every changed module.
- [ ] `go build ./...` clean in every changed service.
- [ ] `docker buildx bake atlas-<svc>` clean for every service whose `go.mod` was touched.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check`,
      `tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh` clean.
- [ ] `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh`,
      `tools/template-movement-types-guard.sh` clean (templates changed).
- [ ] Code review run before PR.

**Documentation**
- [ ] `docs/research/missing-features/items-and-consumables.md:40` updated to reflect that
      5170000 is implemented.
- [ ] `docs/research/missing-features/packet-gap-inference.md:469` updated for `PET_NAMECHANGE`.
- [ ] `rollout.md` documents the live-tenant socket-config reconciliation step (FR-8.5).
