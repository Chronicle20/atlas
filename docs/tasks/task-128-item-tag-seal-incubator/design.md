# Item Tag, Sealing Locks, and Incubator (Cash 506 Family) — Design

Version: v1
Status: Draft
Created: 2026-07-02
PRD: `docs/tasks/task-128-item-tag-seal-incubator/prd.md`

---

## 1. Summary

Three cash-item behaviors carved out of the existing `CharacterCashItemUseHandleFunc`
fall-through, each following the established sibling pattern (tasks 123–126): decode a
per-version serverbound sub-body → validate server-side → drive an atomic
`saga.Saga` → let downstream status events refresh the client. Plus one genuinely new
end-to-end field (`owner` on the inventory asset) and one new clientbound packet
(`INCUBATOR_RESULT`).

The three behaviors map to different mutation shapes:

| Behavior | Item ids | Mutation | New machinery |
|---|---|---|---|
| Item Tag | 5060000 | set equip `owner` name | owner field end-to-end + `SetAssetOwner` saga action |
| Sealing Lock | 5060001 (perm), 5061000–5061003 (timed) | set `FlagLock` (+ expiration) | `ApplyAssetLock` saga action + lock-aware expire branch |
| Incubator | 5060002 | destroy sacrifice + award weighted-random reward + result dialog | `incubator-rewards` tenant config + `INCUBATOR_RESULT` writer + `IncubatorResult` saga step |

**IDA-verified during design** (v83 dump port 13342, v95 port 13341):
- `USE_CASH_ITEM` v83 opcode `0x4F` (`COutPacket::COutPacket(&v568, 0x4F)` in
  `CWvsContext::SendConsumeCashItemUseRequest` @ `0xa0a63f`). Per-version opcodes for the
  serverbound handle and the new `INCUBATOR_RESULT` writer are in §7.3 / §10.
- Serverbound sub-body read orders (v83, from the client send-sites): **Item Tag** = `short`
  slot + trailing `int` updateTime (switch case 25 encodes `Encode2` then the common trailing
  `updateTime`); **Sealing Lock** and **Incubator** = `int` inventoryType + `int` slot +
  trailing `int` updateTime (incubator confirm `sub_81AB54` encodes `this[363]`, `this[364]`,
  `updateTime`; the two seal/tag confirm dialogs `sub_82A2A5`/`sub_82AED3` mirror it). These
  match the Cosmic reference. v87/v95/jms re-verify in the plan (open PRD question 1).
- **`INCUBATOR_RESULT` (`CWvsContext::OnIncubatorResult`) is version-dependent.**
  v83 (`0xa28298`) reads exactly `Decode4(itemId)` then `Decode2(count)` — **2 fields**.
  v95 (`0xa00380`) reads `Decode4(itemId)`, `Decode2(count)`, `Decode4(gachaponItemId)`,
  `Decode4(bonusItemId)`, `Decode4(bonusCount)` — **5 fields**. Both branch on
  `itemId <= 0` → client shows the "inventory is full, try again later" dialog
  (`SP_3431`). The writer body therefore differs by version (§7.2).
- WZ (`Cosmic/wz/Item.wz/Cash/0506.img.xml`, GMS v83): exactly 7 items; `protectTime`
  = **7 / 30 / 90 / 365** on 5061000/1/2/3 respectively; 5060000/1/2 carry only `cash=1`.
  No `5062xxx` present in v83. (Resolution of open PRD question 2 in §11.)

---

## 2. Architecture & data flow

```
atlas-channel  CharacterCashItemUseHandleFunc  (character_cash_item_use.go)
   │  decode common prefix (cash/serverbound ItemUse)  → source slot, itemId
   │  it := GetCashSlotItemType(t)(itemId)             → 25 / 26|64|65 / 27
   │  decode per-type sub-body (cash/serverbound/*)     → target slot / invType
   │  server-side validation (read target asset via inventory REST processor)
   ├── invalid ──────────────► validated no-op: warn + (incubator) inline INCUBATOR_RESULT(0);
   │                                              (tag/seal) enable-actions, no saga, nothing consumed
   └── valid ── saga.NewProcessor(l,ctx).Create(Saga{ InitiatedBy:"CASH_ITEM_USE", … })
                          │  → COMMAND_TOPIC_SAGA
                          ▼
        atlas-saga-orchestrator  (per-action handlers → inventory commands)
                          │
                          ▼
        atlas-inventory  (compartment/asset processors → mutate → status events)
                          │  EVENT_TOPIC_ASSET_STATUS (UPDATED / CREATED / DELETED)
                          │  EVENT_TOPIC_SAGA_STATUS   (COMPLETED / FAILED)
                          ▼
        atlas-channel consumers  →  client packets
             asset consumer  →  CharacterInventoryChange (tooltip refresh, item appears)
             incubator event →  INCUBATOR_RESULT (hatch dialog)
```

The channel handler stays the composition root: it decodes, validates, and builds the
saga. All state mutation is owned by atlas-inventory; the saga orchestrator only sequences
commands. This preserves service boundaries (no channel→inventory direct writes) and reuses
the existing asset-status → inventory-change path for tooltip/inventory refresh with no new
consumer for the tag/seal paths.

---

## 3. Owner field, end-to-end

The equip wire body writes an ASCII owner/title name; today it is hardcoded empty at four
sites in `libs/atlas-packet/model/asset.go` — `encodeEquipableInfo:209`,
`encodeCashEquipableInfo:261`, `encodeStackableInfo:287`, `encodeCashItemInfo:332`
(all literal `w.WriteAsciiString("")`). There is **no owner-name string anywhere upstream** —
the inventory/channel `asset.Model` carries only a numeric `ownerId uint32` (the trade/pickup
owner), which is a different concept and cannot render as the tag name.

**Add a real `owner string` field along the whole chain:**

1. **DB** — `services/atlas-inventory/.../asset/entity.go`: add `Owner string` column,
   `gorm:"not null;default:''"`. GORM AutoMigrate handles it (`asset.Migration`,
   `entity.go:18`). Additive, no backfill. Note the baseline publish/restore column-order
   caveat (`bug_baseline_restore_column_order_drift`): if the `assets` table participates in
   baseline publish/restore with explicit name-keyed column lists, add `owner` there and
   re-publish canonical baselines; if `assets` is not a baseline table, no action.
2. **Domain model/builder** — `asset/model.go` (`owner` field + `Owner()` getter),
   `asset/builder.go` (`SetOwner(string)`, `Clone` carries it). Immutable-model pattern.
3. **REST** — `asset/rest.go` `RestModel` gains `Owner string json:"owner"`; Transform/Extract.
4. **Kafka** — `kafka/message/asset/kafka.go` shared `AssetData` gains `Owner string`;
   `asset/producer.go makeAssetData` populates it. `CreateAssetCommandBody` (compartment)
   already has an `OwnerId`; leave that; owner-name flows through `AssetData` and the new
   `SET_OWNER` command (§6).
5. **atlas-channel projection** — `channel/asset/model.go` (+ builder) gains `owner string`;
   the asset consumer’s `buildAssetFrom*Body` helpers set it from `e.Body.Owner`; the bridge
   `channel/socket/model/asset.go NewAsset` calls a new `SetOwner` on the packet `Asset`.
6. **Packet codec** — `libs/atlas-packet/model/asset.go` `Asset` gains `owner string` +
   `SetOwner`; write `m.owner` at 209/261/287/332 (stackables carry empty in practice — tags
   target equips — but the field is written uniformly). Decode mirror at 416/472 already
   discards the name; keep discarding on decode (channel is write-only for this field).

**Tooltip refresh comes for free:** `handleAssetUpdatedEvent`
(`channel/.../kafka/consumer/asset/consumer.go:263`) already re-emits the whole equip as an
`AddEntry` in a `ChangeBatch` (`InventoryChangeWriter`). Once `owner` is threaded through the
model and the UPDATED event carries it, a set-owner mutation refreshes the tooltip with no new
consumer or writer.

**Existing fixtures:** `asset_test.go`/`asset_v84_test.go` assert only length/determinism, not
the empty-owner byte, so empty-owner encodes remain byte-identical. Add a new fixture asserting
a non-empty owner (PRD §4.4.4).

---

## 4. Serverbound sub-body decoders

New codecs live in `libs/atlas-packet/cash/serverbound/`, one file per type, matching the
existing `ItemUseChalkboard`/`ItemUseFieldEffect` shape — a single struct per behavior with an
`updateTimeFirst bool` constructor arg that controls whether the trailing `int updateTime` is
read before the body (GMS ≥95) or after (the channel already computes
`updateTimeFirst := t.Region()=="GMS" && t.MajorVersion()>=95`).

- `item_use_item_tag.go` — `ItemUseItemTag{ slot int16, updateTime uint32, updateTimeFirst bool }`;
  `NewItemUseItemTag(updateTimeFirst)`. Body: `short slot`.
- `item_use_seal.go` — `ItemUseSeal{ inventoryType int32, slot int32, … }`. Body:
  `int inventoryType`, `int slot`.
- `item_use_incubator.go` — `ItemUseIncubator{ inventoryType int32, slot int32, … }`. Body:
  `int inventoryType`, `int slot`.

Each gets encode+decode, `pt.Variants` round-trip tests, and — after per-version IDA
verification — a `// packet-audit:verify packet=cash/serverbound/<Struct> version=<key>
ida=<0xaddr>` marker plus byte fixtures verified against
`CWvsContext::SendConsumeCashItemUseRequest`/`CUIIncubator::OnButtonClicked` per version. This
resolves the standing `// TODO for v83 there is a trailing updateTime.` at
`character_cash_item_use.go:108`.

---

## 5. Item Tag (5060000, type 25)

1. Decode `ItemUseItemTag` → target equipped slot (negative-slot / equipped compartment
   semantics).
2. Validate (channel, before any mutation): resolve the target via the inventory processor
   (`GetItemInSlot` on the equipped compartment); require non-empty **equip**. Zero/invalid
   slot, empty slot, or non-equip → validated no-op (warn with character/item/slot, enable
   actions, no saga). Re-tagging an already-owned equip is allowed (overwrite).
3. Success: `saga.Saga{ SagaType: ItemTagUse, InitiatedBy:"CASH_ITEM_USE", Steps: [
     DestroyAsset{ TemplateId: 5060000, Quantity:1 },        // consume the tag (destroy-first)
     SetAssetOwner{ CharacterId, InventoryType, Slot, Owner: <character current name snapshot> },
   ]}`. The owner is a **snapshot** of the character’s current name at use time (later renames
   do not update the tag — matches reference `setOwner`).
4. Atomicity: destroy-first ordering means a concurrent double-use fails at step 1 (tag already
   gone) and mutates nothing. Compensation for a failed `SetAssetOwner` re-creates the tag
   (`DestroyAsset` re-award inversion, mirroring PetEvolution). The `SetAssetOwner` command
   emits an asset `UPDATED` event → tooltip refresh (§3).

The character’s current name is available in channel session context; the snapshot is captured
when the saga is built so it is fixed even if the mutation lands later.

---

## 6. Sealing Lock (5060001 perm; 5061000–5061003 timed; types 26 / 64 / 65)

### 6.1 Flow

1. Decode `ItemUseSeal` → `inventoryType`, `slot`.
2. Validate (channel): resolve target; require an **equip**. Reject (validated no-op) when the
   target is missing, non-equip, or — per PRD §4.2.6 — already carries a **non-lock**
   expiration (a genuinely time-limited item), which would otherwise launder an expiring item
   into a permanent one. Validating here, before the saga, honors "consume nothing on
   rejection." Stacking a timed lock onto an already-**locked** item is allowed and extends
   from the current expiration.
3. Compute expiration for timed variants: `expiration = now + protectTime days`, where
   `protectTime` is read from the **lock item’s** WZ data via atlas-data (§9). 5060001 (perm)
   sets the flag with no expiration.
4. Success: `saga.Saga{ SagaType: SealingLockUse, InitiatedBy:"CASH_ITEM_USE", Steps: [
     DestroyAsset{ TemplateId: <lock item id>, Quantity:1 },
     ApplyAssetLock{ CharacterId, InventoryType, Slot, Expiration: <zero for perm | now+days> },
   ]}`. Destroy-first; compensation re-creates the lock item on a failed mutation.

`ApplyAssetLock` sets `FlagLock` (`libs/atlas-constants/asset/flag.go` `FlagLock 0x01`) via
`Clone(a).AddFlag(FlagLock).SetExpiration(exp)`; emits `UPDATED` so the client shows the lock
(and, for timed, the expiration timer the equip codec already encodes).

### 6.2 Lock-aware expiration (the destroy-vs-clear branch)

Expiration detection lives in **atlas-asset-expiration** (`character/processor.go:57`
`IsExpired` → emits an `EXPIRE` compartment command), and the destroy happens in
**atlas-inventory** (`compartment/processor.go:920 ExpireAsset` → `asset/processor.go:159
Expire` → `deleteById`). Branch **inventory-side** in `ExpireAsset`, where the asset `Model` is
in hand (`a.Locked()` checkable):

- If `a.Locked()` (lock expiration passed): instead of delete, `Clone(a).RemoveFlag(FlagLock)`
  and reset expiration to no-expiry, persist, emit `UPDATED` (reuse
  `UpdatedEventStatusProvider`). The item survives and becomes tradeable.
- If not locked: today’s destroy/replace behavior unchanged.

Inventory-side is chosen over the expiration-service seam because the inventory already loads
the full asset (flag available); the expiration service’s REST model reads only
expiration/template/slot/id and would need a new flag field to branch there. This keeps the
lock semantics in one service.

---

## 7. Incubator (5060002, type 27)

### 7.1 Flow

1. Decode `ItemUseIncubator` → `inventoryType`, `slot` (the sacrificial target).
2. Roll the reward (channel): fetch `incubator-rewards` for the tenant (§8), perform a
   weighted-random choice over `(itemId, quantity, weight)`. Empty/missing pool → validated
   no-op: warn and send `INCUBATOR_RESULT` with `itemId = 0` (the client renders "inventory is
   full / try again" for `itemId <= 0`; using it as the generic no-op signal keeps the client
   from hanging on the modeless incubator dialog). Nothing consumed.
3. Capacity pre-check (channel): confirm the reward’s target inventory type has a free slot;
   if not, no-op with `INCUBATOR_RESULT(0)`, nothing consumed (PRD §4.3.4 "rejected before
   anything is consumed").
4. Success: `saga.Saga{ SagaType: IncubatorUse, InitiatedBy:"CASH_ITEM_USE", Steps: [
     DestroyAssetFromSlot{ InventoryType, Slot, Quantity:1, TemplateId:<target> },  // sacrifice
     DestroyAsset{ TemplateId: 5060002, Quantity:1 },                                // incubator
     AwardAsset{ Item:{ TemplateId:<rolled>, Quantity:<rolled> } },
     IncubatorResult{ CharacterId, ItemId:<rolled>, Count:<rolled> },                // terminal
   ]}`. Destroy-first keeps double-use safe (a concurrent second use fails at step 1). The
   capacity pre-check (step 3 of §7.1) prevents the common inventory-full case before any
   destroy; if `AwardAsset` still fails (pre-check race), compensation re-creates the sacrifice
   + incubator, and the failed-saga path emits `INCUBATOR_RESULT(0)`.

The reward is rolled **in the channel at saga-build time** so the concrete `itemId`/`quantity`
are baked into both the `AwardAsset` and `IncubatorResult` steps — the roll is fixed, not
re-evaluated per step. `DestroyAssetFromSlot` carries `TemplateId` for the compensator (mirrors
task-125 §5.4).

### 7.2 `INCUBATOR_RESULT` clientbound writer

New packet under `libs/atlas-packet/incubator/clientbound/` with
`const IncubatorResultWriter = "IncubatorResult"` and `Operation()` returning it. **Plain
single-body packet, no mode/operations table** (verified: no leading mode byte in either
`OnIncubatorResult`). Body is **version-dependent**:

- **v83 / v84**: `int itemId`, `short count`.
- **v87 / v95 / JMS**: `int itemId`, `short count`, `int gachaponItemId`, `int bonusItemId`,
  `int bonusCount`.

Model the version delta with a constructor flag (e.g. `NewIncubatorResult(extended bool, …)`)
resolved from tenant version in the channel, the same idiom as `updateTimeFirst`. Atlas rolls a
single reward (no gachapon/bonus concept), so v87+ sends `gachaponItemId=0, bonusItemId=0,
bonusCount=0`; the client tolerates zeros (the bonus branch is skipped). The v83/v84 boundary
is IDA-verified against the v83 dump; JMS v185 has no locally-loaded IDB — default to the
extended (v95-parity) body and IDA-verify at implementation (banner-flag until then, per the
gms_92 precedent).

Registration: add `IncubatorResultWriter` to `produceWriters()` in
`services/atlas-channel/.../main.go` (writer list, ~`:608`); the writer resolves its opcode
from the tenant `socket.writers` table (§10). Emission uses
`session.Announce(l)(ctx)(wp)(incubatorcb.IncubatorResultWriter)(body)`.

### 7.3 Driving `INCUBATOR_RESULT` from saga completion

Use **Precedent A** (dedicated event, cleanest for a one-off packet, mirrors
`handleEmitGachaponWin`): the terminal `IncubatorResult` saga step handler
(`saga/handler.go`) is fire-and-forget — it produces an `incubator_result` event
(`{CharacterId, WorldId, ChannelId, ItemId, Count}`) to a new topic and immediately
`StepCompleted(true)`. A new channel consumer
(`channel/.../kafka/consumer/incubator/consumer.go`) binds that topic at `LastOffset`,
tenant/world/channel-guards, resolves the session by character id, and announces
`INCUBATOR_RESULT`. The no-op / award-fail `INCUBATOR_RESULT(0)` paths (§7.1 steps 2–4) are
written inline in the channel handler / the saga `FAILED` consumer branch
(`channel/.../kafka/consumer/saga/consumer.go handleFailedEvent`, which already switches on
saga type for storage), so both success and failure reach the client.

Precedent B (extend the currently-empty `handleCompletedEvent`) is the rejected alternative:
it would need the completed-event body to carry the rolled itemId/count (the generic saga
status body does not), so the dedicated event is cleaner.

---

## 8. `incubator-rewards` tenant configuration resource

Follows the generic-JSONB configuration pattern in `services/atlas-tenants/.../configuration/`
exactly as `routes`/`vessels`/`instance-routes`. Resource name `"incubator-rewards"`; entry
attributes `itemId uint32`, `quantity uint32 (≥1)`, `weight uint32 (≥1)`.

Files to touch (per the established pattern):
- `configuration/rest.go` — `IncubatorRewardRestModel` (+ `GetID/SetID/GetName`),
  `TransformIncubatorReward`, `ExtractIncubatorReward`, `CreateIncubatorRewardsJsonData`/
  `CreateSingleIncubatorRewardJsonData`.
- `configuration/provider.go` — `GetIncubatorRewardByIdProvider`, `GetAllIncubatorRewardsProvider`.
- `configuration/processor.go` — interface + impl: `GetAllIncubatorRewards`,
  `GetIncubatorRewardById`, Create/Update/Delete (+`…AndEmit`), providers, `SeedIncubatorRewards`.
- `configuration/resource.go` — 6 handlers + `RegisterRoutes` wiring +
  `RegisterInputHandler[IncubatorRewardRestModel]`.
- `rest/handler.go` — `ParseIncubatorRewardId`.
- `configuration/kafka.go` — `EventType…Created/Updated/Deleted` + status-event provider.
- `configuration/seed.go` — `defaultIncubatorRewardsPath`, `getIncubatorRewardsPath()`,
  `LoadIncubatorRewardsFiles()`.
- `configuration/mock/processor.go` — all new interface funcs (compile-time
  `var _ Processor` check enforces completeness).
- `services/atlas-tenants/configurations/incubator-rewards/*.json` — starter pool records,
  and the corresponding `COPY` line for the new dir in the shared root Dockerfile.

**Runtime consumer** (the channel rolls the reward): a small client package in atlas-channel —
`requests.go` (`requests.RootUrl("TENANTS")` + `…/tenants/%s/configurations/incubator-rewards`
→ `GetRequest[[]IncubatorRewardRestModel]`), `rest.go` (REST model + `Extract`), `processor.go`
(`requests.SliceProvider`) — mirroring atlas-transports’ `transport/config` client.

**Starter pool (WZ-verified v83 ids).** All ids below exist in the local v83 dump
(`Cosmic/wz/Item.wz/...`). A plausible weighted seed shared across every version template
(operators tune per tenant later):

| itemId | what | source path (v83) | weight |
|---|---|---|---|
| 2000000 | Red Potion | `Consume/0200.img.xml` | 40 |
| 2000001 | Orange Potion | `Consume/0200.img.xml` | 30 |
| 2000003 | White Potion | `Consume/0200.img.xml` | 15 |
| 2040000 | scroll | `Consume/0204.img.xml` | 10 |
| 1002000 | hat/cap | `Character.wz/Cap/01002000.img.xml` | 4 |
| 1302000 | weapon (sword) | `Character.wz/Weapon/01302000.img.xml` | 1 |

Quantities: potions ×50, scroll/equip ×1. Weighted roll = pick proportional to `weight`.

---

## 9. atlas-data: `protectTime`

The cash reader (`services/atlas-data/atlas.com/data/cash/reader.go`) does not currently parse
`protectTime`. Add it following the `consumeOnPickup` precedent
(`consumable/reader.go:151`): the value sits in the item’s `info` block (not `spec`) in the
0506 WZ, as `<int name="protectTime" value="7"/>` etc. Add a typed
`ProtectTime int json:"protectTime"` field to the cash `RestModel` (`cash/rest.go`) and read it
in `reader.go` via `info.GetIntegerWithDefault("protectTime", 0)`; add a reader test with an
inline `0506`-style fixture (7/30/90/365). The channel resolves the lock item’s `protectTime`
(the lock item id, e.g. 5061002) to compute the seal expiration (§6.1).

**Unit:** the WZ values 7/30/90/365 are **days** (they are the advertised
week/month/quarter/year seal durations). The channel converts `protectTime` days → expiration
`now.AddDate(0,0,protectTime)`.

---

## 10. Seed templates & live-config patching

`INCUBATOR_RESULT` writer opcode (from STATUS.md row 89, all currently ❌):

| version | opcode |
|---|---|
| gms_v83 | 0x045 |
| gms_v84 | 0x047 |
| gms_v87 | 0x047 |
| gms_v95 | 0x048 |
| jms_v185 | 0x03F |

Append `{"opCode":"0x0NN","writer":"IncubatorResult"}` to `socket.writers` in every version
template under `services/atlas-configurations/seed-data/templates/`
(`template_gms_83_1.json`, `_gms_84_1`, `_gms_87_1`, `_gms_92_1`, `_gms_95_1`,
`template_jms_185_1.json`). gms_92 is login-only/parked (no IDB) — include the row for
completeness but treat channel behavior there as unverified.

The serverbound `CharacterCashItemUseHandle` handler must exist per version with
`"validator":"LoggedInValidator"` (a validator-less entry is silently dropped —
`bug_socket_handler_missing_validator_silently_dropped`). It is currently wired only in
gms_83/84; add the handler row (with the per-version `USE_CASH_ITEM` opcode: v87 0x52, v95 0x55,
jms 0x47 per the sibling task notes) where missing, or confirm the sibling tasks already added
it and avoid a duplicate.

**Live tenants** do not get new opcodes from seed templates (applied at creation only —
`bug_new_opcodes_not_in_live_tenant_config`). The plan must include a runbook step: PATCH each
live tenant’s socket config with the new `IncubatorResult` writer row (+ handler row / any new
operations) and **restart atlas-channel** (writers/handlers do not hot-reload).

---

## 11. Type 74 / 5062xxx resolution (open PRD question 2)

The `5062xxx` slot-type-74 arm is **out of scope** and is documented dead routing for this
task, not silently ignored:

- v83 `0506.img` contains no `5062xxx` (verified). Pre-v95 the type-74 branch does not exist in
  `GetCashSlotItemType`, so `5062xxx` cannot be produced there.
- A later dump (v117.2, `ms_1172/wz/Item.wz/Cash/0506.img.xml`) shows `5062xxx` are the
  **Miracle Cube / potential-cube** family (`MiracleCube_*` UI paths) — a distinct feature
  (equipment potential re-roll), **not** item tag / seal / incubator. No local v95 dump exists
  to confirm v95 specifically, but the item family is unrelated regardless.
- Therefore: the type-74 arm is left unimplemented as an explicit validated no-op (warn:
  "cube family not implemented"), and a `// documented: 5062xxx = Miracle Cube, see task-XXX`
  note points at a future cube task. This is deliberate scoping, not a silent gap.

---

## 12. New saga actions & inventory commands

Two new async saga actions (mirror `handleDestroyAsset`), plus the terminal fire-and-forget
`IncubatorResult` action:

1. `libs/atlas-saga/model.go` — `SetAssetOwner`, `ApplyAssetLock`, `IncubatorResult` action
   consts.
2. `libs/atlas-saga/payloads.go` — `SetAssetOwnerPayload{CharacterId, InventoryType, Slot,
   Owner}`, `ApplyAssetLockPayload{CharacterId, InventoryType, Slot, Expiration}`,
   `IncubatorResultPayload{CharacterId, WorldId, ChannelId, ItemId, Count}`.
3. `libs/atlas-saga/unmarshal.go` + `saga/model.go` local `UnmarshalJSON` — add the cases.
4. `saga/handler.go` — `GetHandler` cases + `handleSetAssetOwner`/`handleApplyAssetLock`
   (emit new compartment commands) and `handleIncubatorResult` (emit event, `StepCompleted`).
5. `saga-orchestrator/compartment/processor.go` + producer — `RequestSetOwner`,
   `RequestApplyLock` producing new inventory commands.
6. `saga/event_acceptance.go` — acceptance mapping (`SetAssetOwner`/`ApplyAssetLock` complete on
   asset `UPDATED`; `IncubatorResult` self-completes).
7. `saga/compensator.go` — `compensateSetAssetOwner` (restore prior owner — or accept
   destroy-first re-create of the cash item as the effective rollback) and re-create inversions
   for the destroy-first cash-item consume.

**New atlas-inventory compartment commands** (in
`kafka/message/compartment/kafka.go` + consumer + processor):

- `SET_OWNER` — `SetOwnerCommandBody{Slot, Owner string}` → `Clone(a).SetOwner(owner)` → persist
  → emit asset `UPDATED`.
- `APPLY_LOCK` — `ApplyLockCommandBody{Slot, Expiration time.Time}` → `Clone(a).AddFlag(FlagLock)
  .SetExpiration(exp)` → persist → emit `UPDATED`.

These are **dedicated, targeted mutations** rather than reusing `MODIFY_EQUIPMENT`.
`MODIFY_EQUIPMENT` (`kafka.go:143`) carries the full equip stat block and would clobber stats
unless the orchestrator first read current stats — extra coupling for no benefit. Dedicated
commands touch exactly one field each and keep the mutation minimal and auditable.

### Saga timeout scaling

All three sagas have small, fixed step counts (2–4), so the flat `DefaultSagaTimeout` (30s) or a
`base + perStep` scaling is fine — none has a data-driven step count, so the flat-timeout bug
class (`bug_preset_creation_saga_flat_timeout`) does not apply. Use `base + perStep*N` for
uniformity with the fixed count.

---

## 13. Alternatives considered

- **Owner via existing `ownerId uint32`** — rejected: `ownerId` is a numeric pickup/trade owner,
  not the ASCII display name the equip codec writes; the tag is a name snapshot. A new `owner`
  string is required.
- **Reuse `MODIFY_EQUIPMENT` for set-owner/lock** — rejected: it replaces the full stat block
  (clobber risk, needs a prior read). Dedicated `SET_OWNER`/`APPLY_LOCK` commands are minimal.
- **Award-first incubator ordering** — considered for strict "nothing consumed if full"; rejected
  in favor of destroy-first + capacity pre-check, which keeps the sibling-standard destroy-first
  double-use safety while still preventing the common full case, with compensation as the
  race-safety net.
- **Precedent B (`handleCompletedEvent`) for `INCUBATOR_RESULT`** — rejected: the generic
  completed-event body lacks the rolled itemId/count; a dedicated `incubator_result` event
  (Precedent A) carries exactly what the packet needs.
- **Expiration-service seam for lock-aware expire** — rejected: it would need a flag field on
  the expiration service’s REST model; the inventory-side `ExpireAsset` seam already has the
  full asset.

---

## 14. Verification plan

Per CLAUDE.md, for every changed module: `go test -race ./...`, `go vet ./...`,
`go build ./...`, `docker buildx bake atlas-<svc>` for every service whose `go.mod` changed
(atlas-channel, atlas-inventory, atlas-tenants, atlas-saga-orchestrator, atlas-data,
atlas-asset-expiration if the expire branch lands there, plus `libs/atlas-packet`,
`libs/atlas-saga`, `libs/atlas-constants` consumers), and `tools/redis-key-guard.sh` from repo
root.

Packet side (`docs/packets/audits/VERIFYING_A_PACKET.md`):
- `INCUBATOR_RESULT` byte fixtures per version with `packet-audit:verify` markers + pinned
  evidence; promote STATUS.md row 89 cells (v83/v84 the 2-field body, v87/v95/jms the 5-field
  body). JMS banner-flagged until an IDB confirms.
- New serverbound sub-bodies (`ItemUseItemTag`/`Seal`/`Incubator`) byte-fixtured against the
  send-site fname per version; serverbound needs marker + pinned evidence + generated report
  and the op routed in each version’s seed template (§10), plus a `candidatesFromFName` case in
  `cmd/run.go` if the audit tool needs it.
- New owner fixture (non-empty owner) added; existing empty-owner encodes unchanged.
- Regenerate `packet-audit matrix`; `matrix --check` exit 0.

Acceptance: live v83 tenant — tag stamps name (survives relog, unaffected by rename), seal sets
lock + timer, timed seal expires to unlocked (item survives), incubator hatches with dialog,
full-inventory/empty-pool consumes nothing.

---

## 15. Service / file touch inventory

- **atlas-channel** — handler arms (types 25/26/27, 64/65 route; 74 documented no-op) in
  `character_cash_item_use.go` (un-discard the `writer.Producer` arg for inline result packets);
  `incubator-rewards` config client; new `incubator` event consumer; `IncubatorResultWriter`
  registration; owner threaded through `asset/model.go`, `asset/builder.go`,
  `socket/model/asset.go`, asset consumer `buildAssetFrom*Body`.
- **atlas-inventory** — `owner` column/model/builder/rest/kafka; `SET_OWNER` + `APPLY_LOCK`
  commands (message/consumer/processor); lock-aware `ExpireAsset` branch; owner in `AssetData`
  producer.
- **atlas-tenants** — `incubator-rewards` resource (10-file pattern §8) + seed data dir.
- **atlas-saga-orchestrator** — `SetAssetOwner`/`ApplyAssetLock`/`IncubatorResult` actions,
  handlers, compartment producers, acceptance table, compensator; `incubator_result` event
  producer.
- **atlas-data** — `protectTime` in cash `reader.go`/`rest.go` + test.
- **libs/atlas-packet** — owner in `model/asset.go` (4 sites) + fixture; `cash/serverbound`
  three sub-body codecs + fixtures; `incubator/clientbound` result writer + fixtures.
- **libs/atlas-saga** — action consts, payloads, unmarshal cases.
- **libs/atlas-constants** — optional named id constants for the seven 506 ids (classification
  `ClassificationItemImprints=506` already covers routing; named ids are a nicety, add if the
  handler references them).
- **atlas-configurations templates** — `IncubatorResult` writer row (+ any missing
  `CharacterCashItemUseHandle` handler row) in all version templates.
- **Root Dockerfile** — `COPY` line for the new `configurations/incubator-rewards` seed dir.
- **Docs** — STATUS.md regeneration; live-config patch runbook.

---

## 16. Risks & open items

- **v87/v95/JMS serverbound read orders** — verify in the plan against each version’s send-site
  (v83/v84 confirmed match Cosmic). Open PRD question 1.
- **JMS `INCUBATOR_RESULT` body** — no local JMS IDB; assume 5-field extended, banner-flag,
  verify when a JMS IDB is available (gms_92 precedent).
- **Baseline publish/restore** — confirm whether the `assets` table participates in baseline
  publish/restore; if so, add `owner` to the name-keyed column lists and re-publish
  (`bug_baseline_restore_column_order_drift`).
- **Capacity pre-check race** — the incubator full-inventory pre-check can race a concurrent
  fill; compensation (re-create sacrifice + incubator) is the safety net and the failed-saga
  path still emits `INCUBATOR_RESULT(0)`.
- **`FlagSpikes == FlagKarmaUse == 0x02`** flag-constant collision exists in
  `libs/atlas-constants/asset/flag.go` but `FlagLock 0x01` is unambiguous — the seal path is
  unaffected.
