# Pet Auto-Pot Validation & Pet Skill Pouches — Design

Task: task-139-pet-auto-pot-validation
Status: Draft for review
Inputs: `prd.md` (approved), IDA investigation (gms_v83 / gms_v84 / gms_v87 / gms_v95 / jms_v185), v83 WZ data, repo source.

---

## 1. Investigation results (resolves PRD Open Questions 1–7)

The design phase ran the mandated IDA/WZ investigation. Several PRD premises are **revised** by the findings; each revision cites its evidence.

### 1.1 The client gates auto-pot — but NOT on the 0519 pouch flag in GMS

The auto-pot send path is `CUserLocal::TryConsumePetHP` / `TryConsumePetMP` (called from `CUserLocal::Update`, `CUserLocal::SetDamaged`, `OnNotifyHPDecByField` (HP), `CWvsContext::OnStatChanged` (MP)). Its per-pet gate differs by client family:

| Version | `TryConsumePetHP` | Gate |
|---|---|---|
| gms_v83 | `0x95b9a4` | Worn pet-equip ability bit (secured `CPet+0x164/+0x16C` pair = consumeHP tear) |
| gms_v84 | `0x999c8d` (named this session) | Identical to v83 (same `+0x164/+0x16C` tears, byte-identical size `0x23c`) |
| gms_v87 | `0x9de089` | Identical to v83 (same tears, same size) |
| gms_v95 | `0x90d8a0` | `CPet::m_bConsumeHP` (named field, same mechanism) |
| jms_v185 | `0xa26d8a` | **`CPet::GetUpgradePetSkill(pet) & 0x20`** — the pouch-taught `usPetSkill` |

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

Encoder verified at v83 `0xa0955c`, v84 `0xa5393e`, v95 `0x9de400`, jms `0xaee1d4` (layout already byte-fixtured in `libs/atlas-packet/pet/serverbound/item_use_test.go` for all five versions):

- `petId` (u64) — the pet's **cash-locker SN** (`CPet::m_liPetLockerSN`). Atlas encodes the pet item's SN as `uint64(pet game id)` (`libs/atlas-packet/model/asset.go:305`), so **the wire petId round-trips as the Atlas pet id**. Existing precedent for resolving it: `pet_chat.go:20-26` (`pet.GetById(uint32(pk.PetId()))`).
- `buffSkill` (byte, misnamed in Atlas) — **damage-mitigation context, log-only**. It is `0` on every trigger path except `SetDamaged`, where (v95 stack-var names, call site `0x9364e2`): `1` = Power Guard reflected damage, `2` = Meso Guard absorbed damage, `4` / `8` (v95+ only for `8`) = other mitigation arms. v83 emits only `{0,1,2,4}` (`0x95965a-0x959675`). It carries no validation-relevant information; the design logs it and nothing else (satisfies FR-6).
- `updateTime` (u32), `source` slot (i16), `itemId` (u32).
- **HP vs MP is NOT on the wire** (OQ5): `TryConsumePetHP` and `TryConsumePetMP` are separate functions but encode identical packets. The server derives intent from the consumed item's spec. For dual HP+MP items, possession of **either** matching skill/equip passes — the wire cannot distinguish the trigger, so the PRD's "either" rule stands.

### 1.3 The 0519 "charge skill" use packet exists only in the JMS client — OQ4

- **jms_v185**: `CWvsContext::SendConsumeCashItemUseRequest` (`0xaef2f5`) jump-table **case 28** (matching Atlas's existing `category 519 → CashSlotItemType 28` mapping at `character_cash_item_use.go:294-299`): opens the pet-selection dialog, then encodes the chosen pet's 8-byte SN into the request (`EncodeBuffer(pet+0x18, 8)` at `0xaf1a42`). Payload = common cash-item-use prefix + `petId u64`. This answers FR-11: the client designates the target pet explicitly.
- **gms_v83 (`0xa0a63f`) and gms_v95 (`0x9eb3e0`)**: the entire function contains **no 8-byte encode and no 519/type-28 arm** — GMS clients cannot use 0519 items at all. (The v83 pet-selection dialog found at `CDraggableItem::OnDoubleClicked 0x4f0aeb` belongs to the *wearable* flow: it leads to `WearEquipItem` + `to_petabil_item_bodypart`, i.e. equipping 1812xxx, not consuming 0519.) gms_v84/v87 senders were not exhaustively swept; they are bracketed by v83/v95 and their auto-pot gates are byte-identical to v83, so they are presumed pouch-inert — treated as "reject as forged if received" (same handling as v83/v95, see §3.2).

WZ (v83 `Item.wz/Cash/0519.img.xml`) confirms the full item set: `5190000-5190008` (`add=1`) with one skill key each — `pickupItem, consumeHP, longRange, dropSweep, pickupAll, ignorePickup, consumeMP, recall, autoSpeaking` — and removers `5191000-5191004` (`add=0`: `pickupItem, consumeHP, longRange, dropSweep, pickupAll`). **OQ7 confirmed: no consumeMP remover exists in v83 data.** (Note: 0519 uses key `dropSweep` where the equip family calls the same ability `sweepForDrop`.)

### 1.4 Client sync of pouch skills — OQ2/OQ3, FR-12

The pet item wire serialization (`GW_ItemSlotPet::RawDecode`, v83 `0x4e4219`) carries, after name/level/closeness/fullness/deadDate: `petAttribute` (i16), **`usPetSkill` (i16)**, `remainLife` (i32), `attribute` (i16). Atlas currently hardcodes the skill short to `0` (`libs/atlas-packet/model/asset.go:313`) — as does Cosmic (`PacketCreator.java:437`). So FR-12's answer: **the client learns pouch skills from the pet item's `usPetSkill` short**; no dedicated packet exists. Encoding the flag there is required for JMS auto-pot to function and is client-interpreted → DOM-25 config-resolved.

Verified `usPetSkill` wire bits: `0x20 = consumeHP` (jms `0xa26d8a`), `0x100 = autoSpeaking` (v83 `CPet::AutoSpeakingByEvent 0x70761f`). Remaining bits are unverified and are NOT invented — see §3.5.

### 1.5 Repo premise corrections

- **The pet `flag` column already exists and persists** (`services/atlas-pets/.../pet/entity.go:30`, `gorm:"not null;default:0"`, hydrated in `Make`, exposed in REST). The PRD's "no column" claim is wrong. The real gaps: no updater (`administrator.go` has no `updateFlag`), no Kafka command/event for it, no consumer of `Flag()` in atlas-channel, not encoded to the client.
- **`PetItemUseHandle` is dead on gms_95-seeded tenants**: the seed entry (`template_gms_95_1.json:391-393`) has no `validator` key, and `BuildHandlerMap` silently skips validator-less handlers (`libs/atlas-opcodes/producer.go:47-51` — the known task-085 bug pattern). The pet handler block (`:380-398`) is affected. Fixing the template is in-scope (§3.6).
- `character_cash_item_use.go` already validates slot↔item identity for cash items (`GetItemInSlot` + templateId compare, `:25-112`) and already maps `519 → CashSlotItemType 28` but has **no arm** for it — 0519 use is currently a warn-and-stuck no-op (`:110`).
- atlas-data parses neither the 0519 spec skill keys (cash reader keeps only `inc`, `0-9`, `expR`, `drpR`, `time` — `cash/reader.go:99-124`) nor the pet-equip ability attributes (equipment reader has no `consumeHP` etc.).
- The channel has no pet/character cache; both resolve via REST (`CH/pet/processor.go:27-33`, `CH/character/processor.go:65-70`). The canonical unstick is `StatChanged(empty, exclRequestSent=true)` — helper currently local to `character_skill_use.go:131-137`.

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

- `skillGate: "equipAbility"` (gms_83, gms_84, gms_87, gms_95 templates) — **worn-equip gate**: fetch the character's equipped assets and pass if any equipped item at the pet-ability positions carries the matching ability attribute in its equip data (atlas-data, §3.4). Positions checked mirror `UpdatePetAbility` exactly: shared `petHP`(-24)/`petMP`(-25) for any pet, plus the pet-index ability range (`petRing1..petRing2, petItemIgnore` families at -21..-29/-46, -31..-37/-47, -39..-45/-48 — all already named in `libs/atlas-constants/inventory/slot/constants.go:41-68`) with the pet's `Slot()` selecting the range. The check is attribute-driven (equip data has `consumeHP`), never item-id-hardcoded.
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

Per-version seed template updates (`services/atlas-configurations/seed-data/templates/`):
- Add `options.skillGate` to the `PetItemUseHandle` entry: `equipAbility` (gms_83_1, gms_84_1, gms_87_1, gms_95_1), `petSkillFlag` (jms_185_1).
- **Fix gms_95_1**: add `"validator": "LoggedInValidator"` to `PetItemUseHandle` — and, since the same defect makes the four sibling pet handlers dead too (`template_gms_95_1.json:380-398`), add validators to that whole pet block while touching the file (bounded, same root cause; the broader 110-vs-75 validator audit stays out of scope).
- Add the `petSkill` writer code table for the pet-item-encoding writers (sparse per §3.5).

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

1. Template changes apply only to newly-created tenants; existing envs need the equivalent live config PATCH (handler options + v95 validators + writer petSkill tables) and a channel restart.
2. The GMS equip gate needs atlas-data re-ingest (or fresh env) for the new equip attributes; until then the gate would fail closed on GMS tenants — deploy data change first, or ship the handler defaulting `equipAbility` lookups that find *no attribute data at all* to reject-with-distinct-reason (`equip_data_missing`) so the misordering is diagnosable in logs.
3. No player-visible behavior change for legitimate clients (server now enforces exactly what clients already enforce); the only new observable is warn logs on forged traffic (OQ6 answered: yes, clients gate the send — per family per §1.1).

## 8. Testing strategy

- **Unit (channel)**: validation chain table tests via the project Builder pattern — each rejection reason, the pass-through, dual-recovery either-rule, petId narrowing, both gate modes, unconfigured gate. Mock processors per existing handler-test conventions.
- **Byte fixtures (packet-verifier flow)**: the jms_185 cash-item-use type-28 arm (prefix + petId u64) with `packet-audit:verify` markers; the modified `encodePetCashItemInfo` (flag short) fixtures per version — including the zero-flag case proving byte-identical output to today for petless-of-flag tenants (regression guard).
- **Unit (consumables)**: `ConsumePetSkillPouch` — owner/spawned/data-missing failure paths emit `ConsumeError`; success emits `SET_SKILL` + `ConsumeItem` with `TypeValueCash`.
- **Unit (pets)**: `SetSkill` add/remove idempotency, unknown key, event emission, persistence round-trip (`Make`).
- **Unit (data)**: cash + equipment readers against checked-in fixture XML for 5190001/5191001/1812002.
- **Verification gates**: `go test -race`, `go vet`, `go build` per changed module; `docker buildx bake` for atlas-channel, atlas-consumables, atlas-pets, atlas-data, atlas-configurations; `tools/redis-key-guard.sh`.

## 9. Deferred to plan phase (bounded verification tasks, not open questions)

- Exact jms_185 cash-item-use prefix byte order around the type-28 arm (updateTime placement) — resolved by the byte-fixture task.
- jms `TryConsumePetMP` mask (expected 0x40) and any further usPetSkill bits actually consumed — resolved while writing the writer code tables.
- Confirmation that worn-pet-equip moves land at the named negative positions in atlas-inventory as assumed from `slot/constants.go` (implementation reads real inventory data; if positions differ the gate lookup adjusts, not the design).
