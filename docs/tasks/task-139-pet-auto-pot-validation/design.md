# Pet Auto-Pot Validation & Pet Skill Pouches — Design

Task: task-139-pet-auto-pot-validation
Status: Draft for review — revised after rebase onto main (legacy version columns added)
Inputs: `prd.md` (approved), IDA investigation (gms_v48 / gms_v61 / gms_v72 / gms_v79 / gms_v83 / gms_v84 / gms_v87 / gms_v95 / jms_v185), v83 WZ data, repo source.

> **Revision note (rebase onto main).** This design was written when the branch
> was based on a main that supported five client versions. Main now routes
> `PetItemUseHandle` on **eight**: gms_61 (0x8E), gms_72 (0xA5), gms_79 (0xA7),
> gms_83 (0xAB), gms_84 (0xB0), gms_87 (0xB7), gms_95 (0xCB), jms_185 (0xAE).
> Sections 1.1, 1.5, 1.6, 3.2, 3.6 and 7 are revised accordingly; every legacy
> claim below is IDA-verified in this pass, not inferred from the v83 result.
> **gms_48 is a ninth version, not an exclusion.** Its `PetItemUseHandle` is
> simply unrouted: the client has the feature at opcode 0x75 with a
> **petId-less** wire layout, and the matrix ⬜ traces to an audit harvest that
> could not find an unnamed function (§1.7). Wiring it is in scope.
> Out of scope by evidence: the gms_12 / gms_92 templates (partial bring-up
> templates with **no** pet handlers at all).

---

## 1. Investigation results (resolves PRD Open Questions 1–7)

The design phase ran the mandated IDA/WZ investigation. Several PRD premises are **revised** by the findings; each revision cites its evidence.

### 1.1 The client gates auto-pot — but NOT on the 0519 pouch flag in GMS

The auto-pot send path is `CUserLocal::TryConsumePetHP` / `TryConsumePetMP` (called from `CUserLocal::Update`, `CUserLocal::SetDamaged`, `OnNotifyHPDecByField` (HP), `CWvsContext::OnStatChanged` (MP)). Its per-pet gate differs by client family:

| Version | `TryConsumePetHP` | Gate |
|---|---|---|
| gms_v48 | `0x6a840c` (named this pass) | Worn pet-equip ability bit (secured `CPet+264/+272` pair); single pet, no pet array |
| gms_v61 | `0x7ad8e4` | Worn pet-equip ability bit (secured `CPet+340/+348` pair = consumeHP tear) |
| gms_v72 | `0x86805e` | Same mechanism, secured `CPet+348/+356` pair |
| gms_v79 | `0x8b3a06` | Same mechanism, secured `CPet+356/+364` (= `+0x164/+0x16C`, the v83 offsets) |
| gms_v83 | `0x95b9a4` | Worn pet-equip ability bit (secured `CPet+0x164/+0x16C` pair = consumeHP tear) |
| gms_v84 | `0x999c8d` (named this session) | Identical to v83 (same `+0x164/+0x16C` tears, byte-identical size `0x23c`) |
| gms_v87 | `0x9de089` | Identical to v83 (same tears, same size) |
| gms_v95 | `0x90d8a0` | `CPet::m_bConsumeHP` (named field, same mechanism) |
| jms_v185 | `0xa26d8a` | **`CPet::GetUpgradePetSkill(pet) & 0x20`** — the pouch-taught `usPetSkill` |

**Legacy GMS is the same gate, verified not assumed.** `CPet::UpdatePetAbility`
was unnamed in the v61 IDB; it is `0x614b60` (named `CPet__UpdatePetAbility`
this pass). Its body is structurally identical to the v83 function: it walks the
equipped-slot list honoring **24/25 for every pet index**, plus 21–29/46 (pet 0),
31–37/47 (pet 1), 39–45/48 (pet 2); ORs each worn item's `dwPetAbilityFlag`
(`EQUIPITEM+108`); and tears the result bit-by-bit into the secured `CPet`
fields in the same order — `0x01→+288, 0x02→+300, 0x04→+312, 0x08→+324,
0x10→+336, 0x20→+348, 0x40→+360, 0x80→+372`. The `0x20→+348` tear is exactly
the pair `TryConsumePetHP` fuses at `+340/+348`. So the v83 ability-bit map in
the table below holds unchanged for v61, and the slot list the design's equip
gate mirrors is the same list across the whole GMS family.

The GMS ability bits are computed in `CPet::UpdatePetAbility` (v83 `0x705fc3`, v95 `0x6a0a40`) **exclusively by OR-ing the `dwPetAbilityFlag` attribute word of the character's WORN pet-equip items** (equipped-slot list `g_anPetAbilBodyPart`: slots 24/25 for every pet, plus 21–29/46 for pet index 0, 31–37/47 for index 1, 39–45/48 for index 2). The pouch-taught `usPetSkill` is *never* consulted by GMS auto-pot; in GMS clients it only feeds `CPet::AutoSpeakingByEvent` and a slash-command path.

The `dwPetAbilityFlag` bit map was recovered from `CItemInfo::RegisterEquipItemInfo` (v83 `0x5cac3d`, OR sites `0x5cd3cb`–`0x5cd672`, property names resolved via the IDB `StringPoolStrings` enum):

| WZ equip attribute | Bit |
|---|---|
| pickupMeso | 0x01 |
| pickupItem | 0x02 |
| pickupOthers | 0x04 |
| longRange | 0x08 |
| sweepForDrop | 0x10 |
| **consumeHP** | **0x20** |
| **consumeMP** | **0x40** |
| ignorePickup | 0x80 |

WZ (v83, `Character.wz/PetEquip`): only `1812002` carries `consumeHP=1` and only `1812003` carries `consumeMP=1`. Both are cash equips whose designated equipped positions are exactly the slots Atlas already names in `libs/atlas-constants/inventory/slot/constants.go`: `petHP = -24`, `petMP = -25` — and per the client's slot list those two are honored for **all** pet indexes (they are the "24/25 always" arm of `UpdatePetAbility`).

**Consequence (revises FR-3):** on GMS tenants, gating auto-pot on the pet's pouch `flag` would falsely reject every legitimate player (they enable auto-pot by *wearing* 1812002/1812003, never by pouch), violating FR-13. The correct gate is version-family–dependent:
- **GMS (v83/v84/v87/v95): matching pet-ability equip worn** (item whose equip data carries `consumeHP`/`consumeMP`, at the shared `petHP`/`petMP` slots or the pet-index ability range).
- **JMS (v185): matching pet skill flag on the pet** (pouch-taught), exactly as the PRD envisioned.

### 1.2 Packet semantics (`SendStatChangeItemUseRequestByPetQ`) — OQ1, OQ5

Encoder verified at v83 `0xa0955c`, v84 `0xa5393e`, v95 `0x9de400`, jms `0xaee1d4` (layout already byte-fixtured in `libs/atlas-packet/pet/serverbound/item_use_test.go` for those five versions), and — this pass — at **v61 `0x831ab9` (opcode 142 = 0x8E), v72 `0x903f8b` (165 = 0xA5), v79 `0x9552d0` (167 = 0xA7)**. All three legacy encoders emit the identical sequence `EncodeBuffer(petSN, 8) · Encode1(mitigation byte) · Encode4(updateTime) · Encode2(slot) · Encode4(itemId)` — byte-for-byte the layout the existing `pet/serverbound.ItemUse` codec already decodes, and their opcodes match the routed template entries. The v61/v72/v79 matrix cells are therefore 🟡ᶠ purely for want of a fixture, not for a layout question (see plan Task 15).

- `petId` (u64) — the pet's **cash-locker SN** (`CPet::m_liPetLockerSN`). Atlas encodes the pet item's SN as `uint64(pet game id)` (`libs/atlas-packet/model/asset.go:305`), so **the wire petId round-trips as the Atlas pet id**. Existing precedent for resolving it: `pet_chat.go:20-26` (`pet.GetById(uint32(pk.PetId()))`).
- `buffSkill` (byte, misnamed in Atlas) — **damage-mitigation context, log-only**. It is `0` on every trigger path except `SetDamaged`, where (v95 stack-var names, call site `0x9364e2`): `1` = Power Guard reflected damage, `2` = Meso Guard absorbed damage, `4` / `8` (v95+ only for `8`) = other mitigation arms. v83 emits only `{0,1,2,4}` (`0x95965a-0x959675`). It carries no validation-relevant information; the design logs it and nothing else (satisfies FR-6).
- `updateTime` (u32), `source` slot (i16), `itemId` (u32).
- **HP vs MP is NOT on the wire** (OQ5): `TryConsumePetHP` and `TryConsumePetMP` are separate functions but encode identical packets. The server derives intent from the consumed item's spec. For dual HP+MP items, possession of **either** matching skill/equip passes — the wire cannot distinguish the trigger, so the PRD's "either" rule stands.

### 1.3 The 0519 "charge skill" use packet exists only in the JMS client — OQ4

- **jms_v185**: `CWvsContext::SendConsumeCashItemUseRequest` (`0xaef2f5`) jump-table **case 28** (matching Atlas's existing `category 519 → CashSlotItemType 28` mapping at `character_cash_item_use.go:294-299`): opens the pet-selection dialog, then encodes the chosen pet's 8-byte SN into the request (`EncodeBuffer(pet+0x18, 8)` at `0xaf1a42`). Payload = common cash-item-use prefix + `petId u64`. This answers FR-11: the client designates the target pet explicitly.
- **gms_v83 (`0xa0a63f`) and gms_v95 (`0x9eb3e0`)**: the entire function contains **no 8-byte encode and no 519/type-28 arm** — GMS clients cannot use 0519 items at all. (The v83 pet-selection dialog found at `CDraggableItem::OnDoubleClicked 0x4f0aeb` belongs to the *wearable* flow: it leads to `WearEquipItem` + `to_petabil_item_bodypart`, i.e. equipping 1812xxx, not consuming 0519.) gms_v84/v87 senders were not exhaustively swept; they are bracketed by v83/v95 and their auto-pot gates are byte-identical to v83, so they are presumed pouch-inert — treated as "reject as forged if received" (same handling as v83/v95, see §3.2). The same holds a fortiori for gms_v61/v72/v79, which predate v83: they are treated as pouch-inert with no version-specific code, and the pouch arm is wired only where the client can reach it (jms_185). No legacy behaviour depends on this being true — a 0519 use that somehow arrives still runs the owner/spawned validation and the standard consume-error path.

WZ (v83 `Item.wz/Cash/0519.img.xml`) confirms the full item set: `5190000-5190008` (`add=1`) with one skill key each — `pickupItem, consumeHP, longRange, dropSweep, pickupAll, ignorePickup, consumeMP, recall, autoSpeaking` — and removers `5191000-5191004` (`add=0`: `pickupItem, consumeHP, longRange, dropSweep, pickupAll`). **OQ7 confirmed: no consumeMP remover exists in v83 data.** (Note: 0519 uses key `dropSweep` where the equip family calls the same ability `sweepForDrop`.)

### 1.4 Client sync of pouch skills — OQ2/OQ3, FR-12

The pet item wire serialization (`GW_ItemSlotPet::RawDecode`, v83 `0x4e4219`) carries, after name/level/closeness/fullness/deadDate: `petAttribute` (i16), **`usPetSkill` (i16)**, `remainLife` (i32), `attribute` (i16). Atlas currently hardcodes the skill short to `0` (`libs/atlas-packet/model/asset.go:313`) — as does Cosmic (`PacketCreator.java:437`). So FR-12's answer: **the client learns pouch skills from the pet item's `usPetSkill` short**; no dedicated packet exists. Encoding the flag there is required for JMS auto-pot to function and is client-interpreted → DOM-25 config-resolved.

Verified `usPetSkill` wire bits: `0x20 = consumeHP` (jms `0xa26d8a`), `0x100 = autoSpeaking` (v83 `CPet::AutoSpeakingByEvent 0x70761f`). Remaining bits are unverified and are NOT invented — see §3.5.

### 1.5 Repo premise corrections

- **The pet `flag` column already exists and persists** (`services/atlas-pets/.../pet/entity.go:30`, `gorm:"not null;default:0"`, hydrated in `Make`, exposed in REST). The PRD's "no column" claim is wrong. The real gaps: no updater (`administrator.go` has no `updateFlag`), no Kafka command/event for it, no consumer of `Flag()` in atlas-channel, not encoded to the client.
- ~~**`PetItemUseHandle` is dead on gms_95-seeded tenants**~~ — **no longer true; fixed on main.** All eight gms_95 pet handlers now carry `"validator": "LoggedInValidator"` (`template_gms_95_1.json` opCodes 0x52, 0x6E, 0xC7–0xCC; `PetItemUseHandle` at `:636-640`). The `BuildHandlerMap` silent-skip behaviour itself is unchanged (`libs/atlas-opcodes/producer.go:65-69`), so the *rule* still binds every template entry this task adds — but the repair is no longer in scope. Plan Task 13 Step 2 is dropped.
- `character_cash_item_use.go` already validates slot↔item identity for cash items (`GetItemInSlot` + templateId compare, `:25-112`) and already maps `519 → CashSlotItemType 28` but has **no arm** for it — 0519 use is currently a warn-and-stuck no-op (`:110`).
- atlas-data parses neither the 0519 spec skill keys (cash reader keeps only `inc`, `0-9`, `expR`, `drpR`, `time` — `cash/reader.go:99-124`) nor the pet-equip ability attributes (equipment reader has no `consumeHP` etc.).
- The channel has no pet/character cache; both resolve via REST (`CH/pet/processor.go:27-33`, `CH/character/processor.go:65-70`). The canonical unstick is `StatChanged(empty, exclRequestSent=true)` — helper currently local to `character_skill_use.go:131-137`.
- `character_cash_item_use.go` has grown substantially on main; the `category == 519` mapping now sits at `:824`. Plan Task 9's line citations are stale — locate by symbol, not line.

### 1.6 The pet item trailer is NOT uniform across versions (new, blocking for §3.5)

`GW_ItemSlotPet::RawDecode` was decompiled per version this pass. The four
trailer fields Atlas encodes unconditionally are **not** all read by the legacy
clients:

| Version | `petAttribute` i16 | **`usPetSkill` i16** | `remainLife` i32 | `attribute` i16 | Address |
|---|---|---|---|---|---|
| gms_v48 | ✔ | ✔ | ✘ | ✘ | `0x49c77e` (named this pass) |
| gms_v61 | ✔ | ✔ | ✘ | ✘ | `0x4b52f2` |
| gms_v72 | ✔ | ✔ | ✔ | ✘ | `0x4d06dd` (named this pass) |
| gms_v79 | ✔ | ✔ | ✔ | ✔ | `0x4d84c4` (named this pass) |
| gms_v83 | ✔ | ✔ | ✔ | ✔ | `0x4e4219` |

Two consequences:

1. **Good news for the feature:** the `usPetSkill` short exists on every routed
   version at the same relative position, so the §3.5 config-resolved encode
   works unchanged for v61/v72/v79 — the sparse-table rule covers them (no
   verified legacy bit ⇒ encodes 0 ⇒ byte-identical to today).
2. **Pre-existing main defect, in this task's blast radius:**
   `model.Asset.encodePetCashItemInfo` (`libs/atlas-packet/model/asset.go:337-359`)
   has **no version gates** — it always writes `attribute, skill, remainLife(18000),
   attribute`. Against a v61 client that is **6 bytes** of overrun, against v72
   **2 bytes**. This task edits that exact function to insert the skill mask, so
   the gates go in here (plan Task 3) rather than being left as a landmine the
   next reader assumes was considered. The gate idiom is `t.IsRegion("GMS") &&
   t.MajorAtLeast(N)` per the house rule — never a raw `> N` comparison.

### 1.7 gms_48 HAS pet auto-pot — the matrix ⬜ is wrong, and v48 is in scope

The first pass through this section reasoned from a missing symbol and reached
the wrong answer. Re-done positively, v48 has the whole feature:

- **Encoder:** `CWvsContext::SendStatChangeItemUseRequestByPetQ` = `0x70dc8d`
  (unnamed until this pass; named now). Opcode **117 = 0x75**.
- **Senders:** `CUserLocal::TryConsumePetHP` `0x6a840c` and
  `TryConsumePetMP` `0x6a8596` (both previously `sub_…`).
- **Gate:** the same worn-equip ability tear as every other GMS version —
  secured `CPet+264/+272` (consumeHP) and `+276/+284` (consumeMP).
- **Keymap ids:** `CFuncKeyMappedMan+896/+900`, populated by
  `OnPetConsumeItemInit` `0x4e5eb7` — which reads **both** ids from one packet,
  where v61 split them across two (`0x51ab0b` / `0x51ab31`). The class is 904
  bytes, so those two dwords are its last fields.

Why the matrix says n-a: the audit record
`docs/packets/audits/gms_v48/PetItemUse.json` carries
`"IDAComment": "function not found in IDB"` — the harvest could not find an
*unnamed* function. Textbook `unnamed ≠ absent`. Re-harvest after this naming
pass and the cell becomes real.

The mis-named `?TryConsumePetMP@CUserLocal@@` at `0x7204db` is unrelated: it
tests job 132 and skill **1320006 = `DarkKnightBerserkId`**
(`libs/atlas-constants/skill/constants.go:3011`) against an HP ratio and sends a
bodyless opcode 86 — Berserk, not auto-pot. Renamed
`CUserLocal__TryDoBerserk_job132_skill1320006`.

**The v48 wire layout diverges — no `petId`.** `0x70dc8d` encodes only
`Encode1(mitigation) · Encode4(updateTime) · Encode2(slot) · Encode4(itemId)`;
there is no leading 8-byte locker SN, because v48 `CUserLocal` holds a **single**
pet (`this[890]`, a lone pointer — v61+ index a 3-slot array). Two consequences:

1. `pet/serverbound.ItemUse` needs a version gate on the leading `petId u64`:
   present for `(GMS && MajorAtLeast(61)) || JMS`, absent below that. The 48↔61
   boundary is where our IDB set changes; no client between them is available,
   so the gate is pinned at the verified edge.
2. **FR-1 cannot be satisfied from the packet on v48.** There is no pet id to
   validate. With one possible pet the packet is unambiguous, so the handler
   resolves the character's single spawned pet and runs the identical
   ownership/spawn/skill checks against it. The PRD's "some spawned pet is NOT
   sufficient" rule stands where the client *can* have several; on a single-pet
   client it is the same statement. See §3.1.

Opcode 0x75 is unoccupied in `template_gms_48_1.json`, and it lands exactly in
the GMS pet block's usual order — Movement 0x71, Chat 0x72, Command 0x73,
DropPickUp 0x74, **ItemUse 0x75**. Independently corroborated: v48
`CPet::SendDropPickUpRequest` (`0x58ed98`) emits opcode 116 = 0x74, matching that
template entry, so this IDB's raw `COutPacket` ctor values are the template
opCodes directly.

---

## 2. Goals restated against the findings

1. **FR-1/FR-2/FR-4/FR-5** unchanged: specific-pet (exists, owned, spawned), character-alive, unstick + warn on reject, pass-through on success.
2. **FR-3 (revised)**: the "pet skill" gate is version-family aware — GMS = worn pet-ability equip; JMS = pet flag bit. Selection is config-driven, not region-hardcoded (§3.2).
3. **FR-7–FR-12 (scoped)**: pet-skill pouch system implemented end-to-end and data-driven; its client entry point (0519 use) is reachable only from JMS clients, and the flag reaches clients via the pet item's `usPetSkill` short (config-resolved bits).
4. Fix the gms_95 template validator gap so the validated handler is actually live everywhere.

Non-goals unchanged from the PRD (no behavior for the other seven skills beyond storage/sync; no Cosmic multi-pot drain; no keymap work; no reservation-flow changes).

---

## 3. Architecture

### 3.1 Auto-pot validation chain (atlas-channel, `PetItemUseHandle`)

`PetItemUseHandleFunc` (`CH/socket/handler/pet_item_use.go`) becomes a validation pipeline in front of the existing `consumable.RequestItemConsume` call:

```
decode → resolve inputs (parallel) → validate → forward | reject(unstick+warn)
```

1. **Decode** as today (`pet2.ItemUse`). Narrowing guard: `p.PetId() > math.MaxUint32` → reject (Atlas pet ids are uint32; anything else is forged).
   **Pet resolution has two sources** (§1.7): where the wire carries a `petId`
   (GMS ≥61, JMS) the handler resolves that id and the narrowing guard applies;
   on gms_48 the packet has no pet id, so it resolves the character's spawned
   pet — of which a v48 client can have exactly one. Both paths then run the
   *same* ownership/spawn/skill checks; only the lookup differs. A v48 character
   with no spawned pet still rejects (`pet_not_found`), which is the abuse case
   the task exists for. The petId-bearing path never falls back to "any spawned
   pet" — that would re-open the hole FR-1 closes.
2. **Parallel fetch** (mirrors `ConsumeStandard`'s `model.NewGroup` pattern, `atlas-consumables consumable/processor.go:319-344` — one round-trip of latency, not three):
   - `pet.NewProcessor(l, ctx).GetById(uint32(p.PetId()))` — channel pet model already carries `OwnerId()`, `Slot()`, `Flag()`.
   - `character.NewProcessor(l, ctx).GetById(s.CharacterId())` — bare, no decorators; provides `Hp()`.
   - Gate-source data per §3.2 (equipped assets or nothing extra).
   - `data/consumable.GetById(itemId)` — the consumed potion's spec, to classify HP-recovery vs MP-recovery (`SpecTypeHP/HPRecovery/MP/MPRecovery`, same keys `ApplyItemEffects` uses).
3. **Validate**, cheapest-first, all failures identical externally:
   - FR-1: pet fetch error → reject; `OwnerId() != s.CharacterId()` → reject; `Slot() < 0` → reject (spawned-slot semantics per `atlas-pets pet/processor.go:209-211`).
   - FR-2: `c.Hp() == 0` → reject (precedent: `character_skill_use.go:51-53`).
   - FR-3: item recovers HP → require the HP skill source; recovers MP → require MP; recovers both → either suffices; recovers neither (not a potion) → reject.
4. **Reject** = `enableActions` (empty `StatChanged`, `exclRequestSent=true`) + one `l.WithFields(...).Warnf` carrying characterId, petId, itemId, slot, buffSkill byte, and a machine-readable reason (`pet_not_found`, `pet_not_owned`, `pet_not_spawned`, `character_dead`, `missing_pet_skill`, `not_consumable`). The `enableActions` helper moves from `character_skill_use.go:131-137` to a shared location in the handler package so both callers use one implementation. No packet is sent besides the unstick (security posture: no detail leaks).
5. **Forward** unchanged on success (FR-5): `RequestItemConsume(s.Field(), characterId, itemId, source, updateTime)`. The downstream flow is untouched, so no double-unstick can occur: the handler rejects *instead of* forwarding, and the existing consumable ERROR-event unstick (`CH/kafka/consumer/consumable/consumer.go:57-81`) fires only for forwarded requests (FR-13).

`buffSkill` (FR-6): decoded, logged in both the debug line and any warn line, never used for control flow. Documented semantics from §1.2 go in a code comment on the packet struct (constraint the code can't show: wire byte is mitigation context, not a skill id).

### 3.2 The FR-3 gate: two sources, config-selected

Two gate implementations, selected per tenant version via the handler's existing per-handler config options (the `readerOptions`/handler-entry `options` map that socket handlers already receive — same mechanism the templates use for reader options today):

```json
{ "opCode": "0xAB", "validator": "LoggedInValidator", "handler": "PetItemUseHandle",
  "options": { "skillGate": "equipAbility" } }
```

- `skillGate: "equipAbility"` (gms_61, gms_72, gms_79, gms_83, gms_84, gms_87, gms_95 templates — §1.1 verifies the legacy three use the same worn-equip mechanism) — **worn-equip gate**: fetch the character's equipped assets and pass if any equipped item at the pet-ability positions carries the matching ability attribute in its equip data (atlas-data, §3.4). Positions checked mirror `UpdatePetAbility` exactly: shared `petHP`(-24)/`petMP`(-25) for any pet, plus the pet-index ability range (`petRing1..petRing2, petItemIgnore` families at -21..-29/-46, -31..-37/-47, -39..-45/-48 — all already named in `libs/atlas-constants/inventory/slot/constants.go:41-68`) with the pet's `Slot()` selecting the range. The check is attribute-driven (equip data has `consumeHP`), never item-id-hardcoded.
- `skillGate: "petSkillFlag"` (jms_185 template) — **flag gate**: the resolved pet's `Flag()` must contain the matching semantic skill bit (§3.5).
- Missing/unknown option value → fail closed with a startup-visible warn (reject all FR-3 checks, log `skill_gate_unconfigured`) so a template gap is loud, not permissive.

Why config and not `tenant.Region()` branching: the gate difference is client-behavior-derived and belongs with the rest of the per-version handler wiring in tenant templates (same philosophy as DOM-25 and the dispatcher-mode rule: version-dependent behavior resolves from config, code stays uniform). It also lets a future version flip gates without a code change.

Note the deliberate redundancy: on GMS the client already enforces the equip gate, so server rejections there indicate forged packets (or state drift). On JMS the client enforces the flag gate. Server-side we enforce the *same* gate the client family uses — never a stricter foreign one — which is what keeps FR-13 intact.

### 3.3 Pet skill pouch system (FR-7–FR-11)

**Constants** (`libs/atlas-constants`):
- `item.ClassificationPetSkill = Classification(519)` in the existing cash block gap between 517 and 520 (`item/constants.go:69-105`).
- New `pet/skill` (or `pet` package extension) defining the nine semantic skills as typed string keys (`pickupItem, consumeHP, longRange, dropSweep, pickupAll, ignorePickup, consumeMP, recall, autoSpeaking` — canonical spelling = the 0519 WZ spec keys) plus Atlas-canonical `flag` bits `1<<0 … 1<<8` in that order. These bits are **Atlas-internal storage semantics**, deliberately decoupled from client wire bits (which are tenant-config-resolved, §3.5). `dropSweep` is documented as the same ability the equip family spells `sweepForDrop`.

**atlas-channel** (`character_cash_item_use.go`): implement the missing `CashSlotItemType 28` arm. Sub-decode = `petId u64` per §1.3 (jms-verified; byte-fixture required at implementation, including where the trailing/leading updateTime sits for each version's prefix — the existing `cashsb.ItemUse` version gates stay authoritative). Replace the magic `category == 519 → 28` literal with the new classification constant. The arm forwards to `RequestItemConsume` **with the petId** (next paragraph). If a 0519 use arrives on a tenant whose client family cannot send one, downstream validation (owner/spawned checks still apply) plus the standard consume error path handles it; no version-specific rejection code.

**Command plumbing**: `RequestItemConsumeBody` (channel and consumables sides, `kafka/message/consumable/kafka.go`) gains an optional `PetId uint64 \`json:"petId,omitempty"\`` — backward-compatible JSON; only the 0519 arm sets it. (The auto-pot path does NOT set it; validation already happened at the socket and nothing downstream needs the pet.)

**atlas-consumables**: new `ItemConsumer` branch in `RequestItemConsume` (`consumable/processor.go:184-226`): `item2.GetClassification(itemId) == item2.ClassificationPetSkill → ConsumePetSkillPouch(transactionId, characterId, slot, itemId, petId)`. The consumer, on `RESERVED`:
1. `petId == 0` → `ConsumeError` (malformed/forged).
2. Fetch cash item data (`cash.GetById`) — now exposing the skill spec keys + `add` flag (§3.4). No skill key present → `ConsumeError` (bad data).
3. Fetch the pet (consumables' existing `pet` REST client); require `OwnerId == characterId` and `Spawned` (matches the client dialog's reachable set; conservative and consistent with FR-1's posture). Failure → `ConsumeError` with a new `ErrPetCannotLearn` (wire `errorType: "PET_CANNOT_LEARN"`).
4. Emit the new pet command (below) and `cpp.ConsumeItem(characterId, TypeValueCash, transactionId, slot)`. (Use `TypeValueCash` consistently — not repeating `ConsumeCashPetFood`'s Use/Cash mix-up noted at `processor.go:438-460`; that existing inconsistency is out of scope to fix but not to copy.)

**atlas-pets**:
- Command `SET_SKILL` on the existing `COMMAND_TOPIC_PET` (`kafka/message/pet/kafka.go` types block): body `{ Skill string, Enabled bool }` — semantic key on the wire between services, mapped to the canonical bit inside atlas-pets via the constants package. Unknown skill key → warn + drop.
- `administrator.go`: `updateFlag(db, tenantId, petId, flag)` following the existing update fns (`updateSlot` at `:38` etc.).
- Processor `SetSkillAndEmit / SetSkill(mb)` (tx, `GetById`, compute new mask, persist, emit) → new status event `FLAG_CHANGED` on `EVENT_TOPIC_PET_STATUS`, body `{ Slot int8, Flag uint16 }` following `fullnessChangedEventProvider` (`pet/producer.go:113-126`). No-op writes (bit already in desired state) still ack but skip the event.
- Consumer registration for `SET_SKILL` in `kafka/consumer/pet/consumer.go` `InitHandlers` alongside the other command handlers.
- No entity change needed (column exists, §1.5); no backfill (0 default is correct).

**atlas-channel FLAG_CHANGED consumer** (`kafka/consumer/pet/consumer.go`): new handler modeled on `handleClosenessChanged` (`:257`) → `announcePetStatUpdate`-style re-send of the pet's cash-asset entry so the client's `GW_ItemSlotPet.usPetSkill` refreshes (this is the whole FR-12 sync — no new packet family). The channel pet REST model already carries `flag`.

### 3.4 atlas-data exposure

Two readers gain attribute parsing (both follow the established spec/info parsing precedents cited by the exploration):

1. **Cash reader** (`data/cash/reader.go:99-124`): parse the nine 0519 skill keys and `add` from each item's `spec`/`info` nodes (they sit as `string`-typed `0/1` values in `Item.wz/Cash/0519.img.xml` — parse tolerant of string-vs-int, like `GetBool`). Exposed on the cash REST model as a `petSkills []string` + `add bool` (only skills with value `1`). Consumables' `cash` client model mirrors it.
2. **Equipment reader** (`data/equipment/reader.go` info block around `:111`): parse the eight pet-ability booleans (`pickupMeso, pickupItem, pickupOthers, sweepForDrop, longRange, consumeHP, consumeMP, ignorePickup`) into a `petAbilities []string` field on the equipment model/REST. Only `consumeHP`/`consumeMP` gain server behavior (the FR-3 equip gate); parsing all eight keeps the data model complete for the non-goal skills without behavior.

Rollout note: existing tenants' ingested data predates these fields; a re-ingest (or the canonical fallback path) is required before the equip gate can see the attributes — called out in §7.

### 3.5 Client sync encode (FR-12, DOM-25)

`model.Asset.encodePetCashItemInfo` (`libs/atlas-packet/model/asset.go:296-318`) replaces `w.WriteShort(0) // skill` with a wire value resolved from writer options: the pet's semantic flag mask is translated bit-by-bit through a per-tenant `petSkill` code table (same `options` mechanism the dispatcher-family writers use for mode bytes). `SetPetInfo` gains the flag param; the channel's `PetAssetEnrichmentDecorator` path populates it from the pet projection.

Template tables carry **only IDA-verified bits** per version; a semantic bit with no table entry encodes as absent (0) with a debug log — never a guessed value:

| Semantic skill | Verified wire bit | Evidence |
|---|---|---|
| consumeHP | 0x20 (jms_185; same bit family as the fully-mapped v83 equip flags) | jms `0xa26d8a`; v83 `0x5cd4ee` |
| autoSpeaking | 0x100 (v83) | v83 `0x70761f` |
| consumeMP | 0x40 expected (equip-family bit v83 `0x5cd54f`); usPetSkill occurrence unverified | verify at implementation (jms `TryConsumePetMP`) |
| others | unverified for usPetSkill | populate as verified; absent until then |

Only the JMS auto-pot gate and autoSpeaking behavior actually read these bits in supported clients, so the sparse table is behavior-complete; filling the rest is verification work in the plan, not a launch blocker.

### 3.6 Template/config changes

Per-version seed template updates (`services/atlas-configurations/seed-data/templates/`), covering **every template whose client speaks this packet** — nine, not five:
- **Add a new `PetItemUseHandle` entry to `template_gms_48_1.json`**: `{"opCode": "0x75", "validator": "LoggedInValidator", "handler": "PetItemUseHandle", "services": ["channel"], "options": {"skillGate": "equipAbility"}}`, inserted at its sorted position (after `PetDropPickUpHandle` 0x74) per the opcode-order guard. This is the only template gaining a *new* entry rather than an option.
- Add `options.skillGate` to the existing `PetItemUseHandle` entry: `equipAbility` (gms_61_1, gms_72_1, gms_79_1, gms_83_1, gms_84_1, gms_87_1, gms_95_1), `petSkillFlag` (jms_185_1).
  A missing entry is not benign: §3.2 fails closed, so a template left unedited turns **every** auto-pot on that version into a rejection. The three legacy templates are the whole reason this section changed after the rebase.
- ~~Fix gms_95_1 validators~~ — already fixed on main (§1.5).
- Excluded, deliberately: `template_gms_12_1.json` / `template_gms_92_1.json` (partial bring-up templates carrying no pet handlers at all — 24 and 43 handler entries respectively, none of them `Pet*`). This is the DOM "config table → all version templates" rule satisfied by evidence, not by skipping.

  > **Correction (post-merge, main → branch).** The gms_92 half of this
  > exclusion is **no longer true**. Merging main brought a new
  > `PetItemUseHandle` entry into `template_gms_92_1.json` (opCode `0xC8`,
  > `fname: CWvsContext::SendStatChangeItemUseRequestByPetQ`), invalidating the
  > "no pet handlers at all" premise the exclusion rested on. Because §3.2
  > fails closed, the combination would have rejected **every** v92 auto-pot
  > with `skill_gate_unconfigured`. gms_92 is therefore now **in scope** and
  > carries `options.skillGate: "equipAbility"` plus the `petSkill` writer
  > table (see the new §3.6 bullet below). gms_12 remains correctly excluded —
  > it still routes no pet handler.
  >
  > The v92 gate value was IDA-verified rather than inferred from the family:
  > `CUserLocal::TryConsumePetHP` (`0x8f3700`) fuses the secured `CPet+356/+364`
  > pair and `TryConsumePetMP` (`0x8f3a20`) `+368/+376` — the same worn-equip
  > `dwPetAbilityFlag` tears §1.1 records for v79/v83/v84/v87. Neither consults
  > `usPetSkill`. Both functions were named in the v92 IDB during that pass.
- Add the `petSkill` writer code table for the pet-item-encoding writers (sparse per §3.5) in every template that carries those writers.
- Editing any template means `tools/template-opcode-order-guard.sh` must pass (new CLAUDE.md gate item 9) — options are added to existing entries here, so ordering is untouched, but the guard still runs.

Live tenants do not pick up template changes (seed-at-creation only — known pattern `bug_new_opcodes_not_in_live_tenant_config`); §7 covers rollout.

---

## 4. Alternatives considered

1. **Gate location: channel socket handler (chosen) vs atlas-consumables.** Consumables has no session, cannot send the unstick, and the command lacks petId today; the socket layer owns client-protocol semantics and already hosts the identical alive-check precedent (`character_skill_use.go`). Rejected: consumables-side FR-1/2/3.
2. **GMS gate source: worn-equip (chosen) vs pouch flag (PRD original) vs either-of.** Pouch flag on GMS falsely rejects every legitimate auto-pot user (client gates on equips — §1.1) and is unreachable by GMS clients anyway (§1.3); either-of loosens the invariant "server enforces exactly the client's gate" for no user-visible gain. PRD FR-3 is therefore implemented per-family.
3. **Gate selection: handler config option (chosen) vs `tenant.Region()` code branch.** Config keeps version-dependent behavior in tenant templates with the rest of the wiring (house rule since task-102/103), survives new versions without code edits, and fails loud when unconfigured. The code branch is less surface but buries client-family knowledge in Go.
4. **Data source: per-request REST with parallel fetch (chosen) vs new channel pet/character projection.** A Kafka-fed cache would cut latency but adds a whole projection subsystem (staleness, warmup, memory) for a path whose rate is bounded by the client's own cooldown/alert throttles (`TryConsumePet*` caps alerts at 3 and stamps per-pet cooldowns). The PRD's performance NFR says "reuse where available" — nothing is available, and building one is scope explosion. Parallel fetch keeps it at ~1 RTT.
5. **Flag storage: Atlas-canonical semantic bits + config-resolved wire bits (chosen) vs storing client bit values directly.** Direct storage violates DOM-25 (client-interpreted values in domain state) and breaks the moment a version disagrees on a bit.

## 5. Error handling

- Handler rejections: unstick + structured warn (§3.1), no client-visible detail. Fetch *errors* (REST down) also reject-with-unstick but log at warn with the transport error — fail closed, never forward unvalidated.
- Consumables 0519 failures: existing `ConsumeError` machinery (reservation cancel + `ERROR` status event). New `PET_CANNOT_LEARN` errorType mapped in the channel consumable consumer to the standard empty-StatChanged unstick (default arm at `consumer.go:76` already does this — only the enum needs adding if a distinct client response is ever wanted; default handling suffices now).
- atlas-pets `SET_SKILL`: unknown skill key or missing pet → warn + drop (command consumers have no reply channel; the consumables-side validation makes this a should-never state).
- Unconfigured `skillGate`: fail closed + warn (§3.2).

## 6. Data model & API surface

- **No schema changes** (flag column exists). No new REST endpoints; the pets REST `flag` field starts carrying real data.
- Kafka: `SET_SKILL` command + `FLAG_CHANGED` status event (atlas-pets topics); optional `petId` on `REQUEST_ITEM_CONSUME`. All additive/back-compatible.
- While touching the consumables↔pets contract: align the consumables-side `Command` mirror's `PetId` type with atlas-pets' `uint32` (currently `uint64` — exploration finding; JSON-compatible but wrong-typed).

## 7. Rollout notes

1. Template changes apply only to newly-created tenants; existing envs need the equivalent live config PATCH (handler options + writer petSkill tables) and a channel restart — for **all eight** routed versions, legacy included.
2. The GMS equip gate needs atlas-data re-ingest (or fresh env) for the new equip attributes; until then the gate would fail closed on GMS tenants — deploy data change first, or ship the handler defaulting `equipAbility` lookups that find *no attribute data at all* to reject-with-distinct-reason (`equip_data_missing`) so the misordering is diagnosable in logs.
2a. **Legacy WZ availability is unverified and is the one genuine unknown this rebase opens.** Only v83 WZ dumps exist locally (`tmp/<tenant>/GMS/83.1`), so whether a v61/v72/v79 tenant's `Character.wz/PetEquip` carries the `consumeHP`/`consumeMP` attributes at all could not be checked from this workspace. If a legacy version's data lacks them, `equipAbility` would reject every legacy auto-pot — which is why `equip_data_missing` must be a *distinct* logged reason rather than folded into `missing_pet_skill`, and why plan Task 15 Step 3 checks a live legacy tenant's equip data before the branch is called done. The client-side gate is verified present on all three (§1.1); it is only the server's data mirror that is open.
3. No player-visible behavior change for legitimate clients (server now enforces exactly what clients already enforce); the only new observable is warn logs on forged traffic (OQ6 answered: yes, clients gate the send — per family per §1.1).

## 8. Testing strategy

- **Unit (channel)**: validation chain table tests via the project Builder pattern — each rejection reason, the pass-through, dual-recovery either-rule, petId narrowing, both gate modes, unconfigured gate. Mock processors per existing handler-test conventions.
- **Byte fixtures (packet-verifier flow)**: the jms_185 cash-item-use type-28 arm (prefix + petId u64) with `packet-audit:verify` markers; the modified `encodePetCashItemInfo` (flag short) fixtures per version — including the zero-flag case proving byte-identical output to today for flagless pets (regression guard) **and the §1.6 trailer gates (v61 short by 6 bytes, v72 by 2)**; and the three legacy `PetItemUse` serverbound fixtures (v61/v72/v79) that promote those matrix cells off 🟡ᶠ, pinned to the encoder addresses in §1.2.
- **Unit (consumables)**: `ConsumePetSkillPouch` — owner/spawned/data-missing failure paths emit `ConsumeError`; success emits `SET_SKILL` + `ConsumeItem` with `TypeValueCash`.
- **Unit (pets)**: `SetSkill` add/remove idempotency, unknown key, event emission, persistence round-trip (`Make`).
- **Unit (data)**: cash + equipment readers against checked-in fixture XML for 5190001/5191001/1812002.
- **Verification gates**: `go test -race`, `go vet`, `go build` per changed module; `docker buildx bake` for atlas-channel, atlas-consumables, atlas-pets, atlas-data, atlas-configurations; `tools/redis-key-guard.sh`.

## 9. Deferred to plan phase (bounded verification tasks, not open questions)

- Exact jms_185 cash-item-use prefix byte order around the type-28 arm (updateTime placement) — resolved by the byte-fixture task.
- jms `TryConsumePetMP` mask (expected 0x40) and any further usPetSkill bits actually consumed — resolved while writing the writer code tables.
- Confirmation that worn-pet-equip moves land at the named negative positions in atlas-inventory as assumed from `slot/constants.go` (implementation reads real inventory data; if positions differ the gate lookup adjusts, not the design).
