# Scissors of Karma — Design

Task: task-223-karma-scissors
PRD: [`prd.md`](prd.md) · IDA record: [`ida-findings.md`](ida-findings.md)
Status: Draft
Created: 2026-08-13

---

## 0. What this phase changed

The PRD shipped with five open questions, four of which blocked implementation. All five are
resolved below from decompiles and WZ data, and two of the resolutions **change the shape of the
feature**:

1. **Eligibility is version-divergent in kind, not just in spelling.** The gms_v83 client asks
   "is `tradeAvailable` non-zero?"; the gms_v87 and gms_v95 clients ask "does `tradeAvailable`
   *equal* the scissors' own `karma` type?". The PRD assumed the equality form everywhere
   (FR-3, §9 OQ-1). §3 replaces both with a single data-driven predicate that is correct on
   every version examined and needs no version gate.
2. **The inverted `KarmaUsed`/`SetKarmaUsed` pair lives in seven services, not four.** FR-4.3
   names `atlas-inventory`, `atlas-channel`, `atlas-login`, `atlas-cashshop`. `atlas-consumables`,
   `atlas-storage` and `atlas-query-aggregator` carry the same defect (§4.3).

Everything else in the PRD stands as written.

---

## 1. Resolved open questions

### OQ-1 — the WZ property spellings (was: blocks FR-3)

Both StringPool ids named in the PRD were resolved to literals by decoding `StringPool::ms_aString`
out of the gms_v95 IDB. The pool is XOR-obfuscated: each entry is `seed byte + encoded bytes + NUL`,
the key is the 16-byte `StringPool::ms_aKey` (`@0xB98830`) rotated left by `seed` bits
(`StringPool::Key::Key @0x746470`, `rotatel<unsigned char> @0x746270`), and a byte that equals its
key byte decodes to the key byte rather than to NUL
(`` `anonymous namespace'::Decode<char> @0x746520 ``). The decoder was validated against two
independently-known entries before being trusted: id `981` (`0x3D5`) decodes to **`info`**, which is
exactly what `CItemInfo::RegisterKarmaScissorsItem @0x5A1120` uses it as, and id `0` decodes to
`http://passport.nexon.n…`.

| StringPool id | Literal | Read by | Meaning |
|---|---|---|---|
| `981` (`0x3D5`) | `info` | `RegisterKarmaScissorsItem @0x5A1120` | container node |
| `5595` | **`karma`** | `RegisterKarmaScissorsItem @0x5A1120` | the *scissors'* own karma type |
| `3234` | **`tradeAvailable`** | `CItemInfo::GetBundleItemInfoData @0x5B3D90` | the *target's* applicable karma type |

The `tradeAvailable` binding is pinned structurally, not by name similarity.
`CItemInfo::BUNDLEITEM.nAppliableKarmaType` sits at offset `0x14` (PDB-backed type), and
`GetBundleItemInfoData` writes `*(v6 + 20)` from the property named by id `3234`. Its immediate
neighbours in the same parse run confirm the alignment: id `3231` → `only` → offset `0xC`
(`bOnly`), id `3232` → `tradeBlock` → offset `0x10` (`bTradeBlock`), id `3235` → `notSale` →
offset `0x18` (`bNotSale`). That is `libs/atlas-constants`-free ground truth:
**`nAppliableKarmaType` is `info/tradeAvailable`.**

The gms_v83 client reaches the same property by name directly, with no karma-type indirection —
`CItemInfo::IsAppliableKarmaItem @0x5D4E8F` (named this pass) reads
`SP_3188_TRADEAVAILABLE` off the item's `info` node and returns `value != 0`.

**Both spellings are confirmed present in the shipped v83 corpus.** `tradeAvailable` occurs 159
times across `Character.wz` (equips) and `Item.wz/Consume/0228.img` (mastery books), always as
`<int name="tradeAvailable" value="1"/>` sitting beside `tradeBlock` under `info` — including
`Character.wz/Cap/01002357.img` (**Zakum Helmet**), which is the FR-3.5 acceptance case, verbatim:

```xml
<int name="only" value="1"/>
<int name="tradeBlock" value="1"/>
<int name="tradeAvailable" value="1"/>
```

`karma` does **not** appear anywhere in the v83 corpus, and `Item.wz/Cash/0552.img/05520000`
carries only `<int name="cash" value="1"/>` — no karma type, and no `05520001`. That is consistent:
the v83 client never reads one.

### OQ-2 — the karma bit at the v83 baseline (was: blocks FR-4)

The functions were unnamed in the v83 IDB, not absent. They were located through
`GW_ItemSlotBase::CreateItem @0x4E34E1`, which assigns the three vtables
(`&off_AF310C` equip, `&off_AF318C` bundle, `&off_AF3204` pet), and read at the two slots
`CUIKarmaDlg::PutItem` calls (`vtable+4` and `vtable+20`). All eight symbols have been **named in
the v83 IDB** this pass, per the CLAUDE.md name-while-reversing rule.

| Slot class | `IsProtectedItem` (slot 1) | `IsPossibleTradingItem` (slot 5) |
|---|---|---|
| `GW_ItemSlotEquip` | `@0x4E9506` → `nAttribute & 0x01` | `@0x4E956E` → `(nAttribute & 0x10) >> 4` |
| `GW_ItemSlotBundle` | `@0x4E9B4F` → `nAttribute & 0x01` | `@0x4E9B6A` → `(nAttribute & 0x02) >> 1` |
| `GW_ItemSlotPet` | `@0x4EA012` → **`return 0`** | `@0x4EA01E` → `nAttribute & 0x01` |

**The `0x10` equip / `0x02` bundle split holds at the v83 baseline**, byte-identical to the gms_v95
values the PRD recorded. FR-4 proceeds as written.

The pet row is new information and it is hostile: on a pet the karma bit is `0x01`, which is
`FlagLock` on every other slot class. The client makes that safe by hard-returning `0` from
`GW_ItemSlotPet::IsProtectedItem`, so the two meanings never coexist on one object. Atlas has no
such guarantee — its asset flag field is shared. See OQ-5.

### OQ-3 — does a hired-merchant sale consume the mark? (was: FR-7.4)

**Resolved as spec'd: yes, a completed merchant sale consumes the mark.**

There is no client behaviour to confirm here and the PRD's "confirm against client behavior" is not
answerable as posed: the mark is a server-owned bit in `nAttribute`, and the client only renders it
(`IsPossibleTradingItem` is read by the tooltip and by the trade/karma dialogs, never written). The
question is therefore a server policy decision, and the grant's stated meaning — "1 time of trading
has been enabled", per the v83 success string
`SP_4664_YOU_HAVE_USED_THE_SCISSORS_OF_KARMA_SO_1_TIME_OF_TRADING_HAS_BEEN_ENABLED` — settles it:
the unit consumed is *a transfer of ownership*, and a hired-merchant sale is one. Listing-only
semantics would let a player launder one mark into unlimited transfers by re-listing.

§7 makes both paths consume it through the same mechanism, so they cannot drift.

### OQ-4 — how does `5520001` differ from `5520000`? (was: FR-8.2)

**Dissolved.** Under the §3 predicate no code path ever compares a karma type against a literal:
the scissors' type is read from its own `info/karma` node and the target's from `info/tradeAvailable`,
and the only operation on them is equality. `5520001` therefore works the moment a tenant's WZ
carries it, gates whatever target set its data says it gates, and is unusable on versions whose
corpus does not contain it — which is FR-8.2's requirement, satisfied by construction rather than by
a version gate. No literal karma type is hard-coded anywhere, so there is nothing left to read out
of the v84 WZ.

The FR-8.2 acceptance criterion is restated in §10 as a data-driven assertion.

### OQ-5 — are pets a valid karma target? (was: FR-3.1, FR-4.2)

**Resolved: out of scope, and explicitly refused.** Three independent reasons:

1. **No pet is eligible in the shipped data.** Every one of the 159 v83 items carrying
   `tradeAvailable` is an equip (`010xxxxx`) or a mastery book (`0228xxxx`). No `5000xxx` pet
   carries it, so `GetAppliableKarmaType(pet) == 0` and gate 2 refuses on every version.
2. **The pet karma bit aliases `FlagLock`.** The client can afford `0x01` to mean karma on a pet
   because `GW_ItemSlotPet::IsProtectedItem` hard-returns `0`. Atlas cannot: `asset.FlagLock` is
   written by the Sealing Lock arm against the same `flag` column, so a pet karma mark would read
   back as a lock (and vice versa) in every service that inspects flags.
3. **Atlas does not model pets as flag-carrying inventory assets.** `atlas-pets` and
   `atlas-npc-shops` carry `karmaUsed` as a plain `bool` field on their own models, not as a flag
   bit — there is no bitfield on that path to set.

FR-5's gate list therefore gains an explicit pet refusal (§5, gate 0c) rather than silently
producing a wrong-bit mutation. This is a documented scope boundary, not a deferral: adding pets
later requires a separate flag column or a third slot class, which is a data-model change the PRD
excludes.

---

## 2. Architecture at a glance

```
 client                atlas-channel                 atlas-saga-orchestrator      atlas-inventory
   │                        │                                 │                        │
   │ USE_CASH_ITEM ────────►│ decode ItemUseKarmaScissors      │                        │
   │  (karma sub-body)      │ resolve cash-slot type (§4.1)    │                        │
   │                        │ gates 0a..3  (§5)                │                        │
   │                        │   ├─ refuse ─► warn + unlock ────┤                        │
   │                        │   └─ pass                        │                        │
   │                        │ Saga{DestroyAsset,ApplyAssetKarma} ─────►                 │
   │                        │                                  │ dispatch ─────────────►│ ApplyKarma
   │                        │                                  │                        │  re-assert gates
   │                        │                                  │                        │  set slot-class bit
   │◄─ INVENTORY_OPERATION (UPDATED, flag) ◄──────────────────────────────────────────  │
```

Consumption of the mark happens nowhere on this diagram — it happens in `atlas-trades` and
`atlas-merchant`, at the point each re-materialises the asset for its new owner (§7).

Three properties drive every choice below:

- **The owning service is the authority.** The channel gates are advisory across a service
  boundary; `atlas-inventory` re-asserts them (FR-6.4). The channel gates exist to fail fast and to
  produce the operator-facing refusal log.
- **Nothing hard-codes a wire value.** Cash-slot type, inventory type and karma type are all
  resolved from the tenant's configured version or from item data (DOM-25).
- **Transfer atomicity is inherited, not invented.** Both transfer paths already round-trip the
  asset through a snapshot that carries `flag`; the mark is cleared *in the snapshot*, so it
  cannot be applied out of step with the transfer (§7).

---

## 3. Eligibility: one predicate, no version gate

### 3.1 What each client actually does

| Version | Function | Gate 2 |
|---|---|---|
| gms_v83 | `CUIKarmaDlg::PutItem @0x830EF7` | `if (!IsAppliableKarmaItem(id))` → refuse. `IsAppliableKarmaItem @0x5D4E8F` = `info/tradeAvailable != 0`. **No karma type exists.** |
| gms_v87 | `CUIKarmaDlg::PutItem @0x895261` | `if (GetAppliableKarmaType(id) != this->m_nKarmaType)` → refuse. `GetAppliableKarmaType @0x606C15` reads `BUNDLEITEM+0x14`. |
| gms_v95 | `CUIKarmaDlg::PutItem @0x7D7BA0` | same equality form; `GetAppliableKarmaType @0x5C09F0`. |

(Both v83 and v87 functions were unnamed and have been named this pass.)

The model changed between v83 and v87 — which is also when `5520001` arrives, so the karma-type
system and the second scissors variant are one change. gms_v84 was not decompiled and does not need
to be; see §3.3.

### 3.2 The server predicate

```
scissorsKarma := scissors.info.karma            // int, default 0
targetKarma   := target.info.tradeAvailable     // int, default 0

eligible := targetKarma != 0
         && (scissorsKarma == 0 || targetKarma == scissorsKarma)
```

Read as: *the target must declare itself karma-applicable at all, and — when the scissors declare a
type — the types must match.*

### 3.3 Why this is correct on every version, including the un-decompiled ones

- **On a v83-era corpus** the scissors carry no `karma` node, so `scissorsKarma == 0`, the second
  clause is vacuous, and the predicate reduces to `targetKarma != 0` — *exactly*
  `IsAppliableKarmaItem`.
- **On a v87/v95-era corpus** the scissors carry a `karma` node, so the predicate is
  `targetKarma == scissorsKarma` plus `targetKarma != 0` — the client's equality test, plus one
  extra condition.
- **That extra condition closes a real client hole.** If a v95-era tenant's WZ omits `karma` on the
  scissors, the client computes `0 != tradeAvailable` and thereby accepts *every ordinary item*
  (whose `tradeAvailable` is absent → 0). The server must not. `targetKarma != 0` refuses that
  whole class. FR-5.6's already-tradeable refusal is a second, independent net under the same hole.
- **gms_v84 needs no decision.** Whichever model it uses, its own data selects the matching branch:
  if v84 carries `karma` on the scissors the predicate is the equality form, and if it does not it
  is the non-zero form. The predicate is a strict subset of the client's rule in both cases, so the
  worst case is a server refusal where the client would have allowed — conservative, visible, and
  logged.

The one behavioural asymmetry worth stating: on a version using the non-zero model, a tenant that
*added* a `karma` node to the scissors would get server-side equality where the client uses
non-zero. That is a data-authoring error, and refusing is the safe response.

### 3.4 Where the properties are parsed (FR-3.1–3.3)

`tradeAvailable` follows the `tradeBlock` precedent exactly — same node, same type, same five
readers, same REST exposure:

| Reader | Existing `tradeBlock` line | New |
|---|---|---|
| `services/atlas-data/atlas.com/data/equipment/reader.go:114` | `TradeBlock` | `TradeAvailable` |
| `.../consumable/reader.go:49` | ″ | ″ |
| `.../setup/reader.go:47` | ″ | ″ |
| `.../etc/reader.go:47` | ″ | ″ |
| `.../cash/reader.go:84` | ″ | ″ + `Karma` |

Both are `int32` read as `i.GetIntegerWithDefault("<name>", 0)` — **integers, not bools.** The
existing `TradeBlock` field is a `bool` via `GetBool`; the karma fields must not copy that, because
the whole point of the equality model is that the value is a type id, and `5520001` exists
precisely to use a second one. `Karma` is parsed on the cash reader only (FR-3.2) and left `0` for
every non-scissors cash item; no classification filter is needed, because absence already yields 0.

Both fields are additive `json:"...,omitempty"`-free integers on the REST models (an explicit `0`
is meaningful and must survive the round trip), mirrored on `atlas-channel`'s consumer models.

---

## 4. The karma bit

### 4.1 Naming and documentation (FR-4.1)

Constant *values* are correct and stay. `libs/atlas-constants/asset/flag.go` gains a comment block
recording that the bit is slot-class dependent, citing the four addresses per version:

| Bit | Equip | Bundle | Pet |
|---|---|---|---|
| `0x01` | lock (`IsProtectedItem`) | lock | **karma** (`IsProtectedItem` hard-returns 0) |
| `0x02` | spikes | **karma** | — |
| `0x10` | **karma** | — | — |

Atlas's names read backwards from that table (`FlagKarmaUse = 0x02` is the *bundle* bit;
`FlagKarmaEquip = 0x10` is the *equip* bit) and are load-bearing in seven services, so they are
documented rather than renamed. The comment is the deliverable; a rename is a separate, larger
change with no behavioural benefit.

### 4.2 The resolver (FR-4.2)

```go
// KarmaFlagFor returns the karma-mark bit for an asset's slot class. The class
// is derived from the template id exactly as the client derives it:
// CItemInfo::GetAppliableKarmaType (gms_v95 @0x5C09F0) branches on
// nItemID / 1000000 == 1.
func KarmaFlagFor(templateId uint32) (Flag, bool)
```

Returns `(FlagKarmaEquip, true)` for `templateId/1000000 == 1`, `(FlagKarmaUse, true)` otherwise —
and `(0, false)` for the pet range `5000000..5000999`, so every caller is forced to handle the
refusal rather than silently writing `0x01`. Deriving from the template id (rather than from the
compartment) mirrors the client and works in `atlas-merchant`, whose `IsListableItem` has only
`(itemId, flag)` in hand.

Placed in `libs/atlas-constants/asset` beside the flags, so no call site picks a bit by hand.

### 4.3 The round-trip fix (FR-4.3–4.5)

`SetKarmaUsed(true)` writes `FlagKarmaEquip` (0x10) while `KarmaUsed()` reads `FlagKarmaUse` (0x02):
a set never reads back, for any asset. **The inverted pair exists in seven services, not the four
FR-4.3 names:**

| Service | Getter | Setter |
|---|---|---|
| `atlas-inventory` | `asset/model.go:84` | `asset/builder.go:147` |
| `atlas-channel` | `asset/model.go:92` | `asset/builder.go:180` |
| `atlas-login` | `.../compartment/asset/model.go:82` | `.../builder.go:144` |
| `atlas-cashshop` | `asset/reference_data.go:54, 373` | `reference_data.go:259, 586` |
| `atlas-consumables` | `asset/model.go:82` | `asset/builder.go:144` |
| `atlas-storage` | `asset/model.go:82` | `asset/builder.go:142` |
| `atlas-query-aggregator` | `asset/model.go:82` | `asset/builder.go:144` |

Five of the seven are fixed the same way: both sides route through `KarmaFlagFor(templateId)`. Their
models already carry the template id (`model.go:58-59` in each), so no signature grows.

`atlas-cashshop` is the exception and takes a simpler fix. `EquipableReferenceData` and
`CashEquipableReferenceData` carry no template id — they are the equip-shaped reference block
hanging off an asset that holds the id — but they are *equip-class by type*, so the correct bit is
unconditionally `FlagKarmaEquip`. Their setters (`:261, 588`) are already right; only the getters
(`:54, 373`) are wrong and become `HasFlag(e.flag, af.FlagKarmaEquip)`. No resolver call, no
signature change, and a comment recording why the bit is fixed rather than resolved.

The setter must remain a targeted set/clear of the resolved bit only (`SetFlag`/`ClearFlag`, never
an assignment), so karma-marking an equip leaves `FlagSpikes` (0x02) untouched — FR-4.5, and the
single sharpest failure mode in this task, since a careless "karma is 0x02" renders spikes on every
karma'd equip.

`atlas-pets` and `atlas-npc-shops` also expose `karmaUsed`, but as plain bools on their own models
with no flag arithmetic; they are untouched, consistent with OQ-5.

---

## 5. The handler arm (atlas-channel)

### 5.1 Sub-body codec (FR-1)

`libs/atlas-packet/cash/serverbound/item_use_karma_scissors.go` — a discrete struct, field-identical
to `ItemUseSeal` (`int32 targetInventoryType`, `int32 targetSlot`, trailing `updateTime` gated on
`updateTimeFirst`), per the discrete-struct-per-mode rule in `docs/packets/DISPATCHER_FAMILY.md`.
Not an alias of `ItemUseSeal`, and emphatically not `ItemUseTargetSlot` (bare `int16`), which is the
Item Tag / expiration-extender shape.

The `updateTime` position is already modelled by `cashsb.UpdateTimeFirst(t)`
(`item_use.go:21-23`), whose gate is `GMS >= 87 || JMS` — which matches the two IDA-derived layouts
in the PRD (v83 trailing, v95 leading) and needs no change.

### 5.2 Type resolution (FR-2)

- `item.ClassificationKarmaScissors = Classification(552)` in `libs/atlas-constants`.
- `GetCashSlotItemType`'s bare `if category == 552` (`character_cash_item_use.go:1103-1109`) uses
  the constant; returned values (`64` on GMS ≥ 95, `63` otherwise) unchanged.
- The arm matches via `karmaScissorsCashSlotItemType(t tenant.Model)`, following
  `viciousHammerCashSlotItemType` (`:760-765`). This is not ceremony: pre-95
  `CashSlotItemTypeSealTimed` is also `64`, and the two arms are disjoint today only because the
  seal arm recomputes itself to `65` at GMS ≥ 95 (`:261-265`). Version-scoped resolvers on both
  sides make the disjointness structural. A test asserts the two resolvers differ on every
  configured tenant version (FR-2.4).

### 5.3 Gate order

The client's three gates (`CUIKarmaDlg::PutItem`) are re-validated server-side in the client's own
order, preceded by the structural checks a crafted packet makes necessary. Each produces a distinct
warn naming character id, scissors template, target inventory type and slot, resolved target
template id, and the failing rule (FR-5.7).

| # | Rule | Source |
|---|---|---|
| 0a | scissors really occupy the claimed CASH slot | `cashItemInSlotFunc` (`:55-59,728`), FR-5.1 |
| 0b | target inventory type is one of the five known types | FR-5.2 |
| 0c | target is not a pet-class template | §4.2 / OQ-5 |
| 0d | target slot ≥ 0 (negative = equipped) | FR-5.3 |
| 0e | target slot is occupied | FR-5.4 |
| 1 | target is not `FlagLock`'d | `IsProtectedItem`, FR-5.5 |
| 2 | §3.2 eligibility predicate | `GetAppliableKarmaType` / `IsAppliableKarmaItem`, FR-5.5 |
| 3 | target is not already karma-marked (`KarmaFlagFor` bit clear) | `IsPossibleTradingItem`, FR-5.5 / FR-6.7 |
| 4 | target is currently *untradeable* | FR-5.6 |

Gate 4 has no client counterpart and is deliberate: karma exists to unlock an untradeable item, and
marking a tradeable one would consume the scissors for nothing. "Untradeable" means the same two
conditions `atlas-trades` enforces — `FlagUntradeable`/`FlagMergeUntradeable`, or the WZ
`tradeBlock` prop — so gate 4 and §7's override are two readings of one definition and cannot
disagree.

Gates 0c and 4 are the two places the server is *stricter* than the client. Both are recorded here
so a future reader does not "fix" them back into client parity.

### 5.4 Client unlock (FR-5.9)

gms_v83 `@0x830FB5` gates the send on `CWvsContext::CanSendExclRequest(500, 0)` and sets the
exclusive-request lock after sending. Every outcome must therefore unlock — including every refusal
above, all of which return without mutating state. Per
`reference_exclrequest_unlock_contract`, the unlock must not be an outcome that warps; the success
path's non-silent `INVENTORY_OPERATION` (from the `UPDATED` event, §6) already clears the lock the
same way the morph-coupon arm relies on, so only the refusal paths need an explicit unlock.

### 5.5 Saga shape (FR-5.8)

Two steps, modelled on the `ItemTagUse` / `SealingLockUse` arms (`:226-259`, `:296-330`):

```
Saga{ SagaType: KarmaScissorsUse, InitiatedBy: "CASH_ITEM_USE", Steps: [
  { consume_scissors,  DestroyAsset,    DestroyAssetPayload{characterId, templateId, quantity:1} },
  { apply_asset_karma, ApplyAssetKarma, ApplyAssetKarmaPayload{characterId, inventoryType, slot} },
]}
```

Order matters: the scissors are destroyed first, so a failure to apply the mark compensates by
restoring the scissors rather than by leaving a free trade behind.

---

## 6. Applying the mark

### 6.1 `libs/atlas-saga` (FR-6.1)

`ApplyAssetKarma Action = "apply_asset_karma"` (`model.go:220-221`), `ApplyAssetKarmaPayload
{CharacterId uint32, InventoryType byte, Slot int16}` (`payloads.go:1088-1100`), unmarshal arm
(`unmarshal.go:576-586`) — all mirroring `ApplyAssetLock`, minus the `Expiration` field.

### 6.2 `atlas-saga-orchestrator` (FR-6.2, FR-6.6)

Dispatch (`saga/handler.go:947-950`), the `Action`/`Payload` aliases (`saga/model.go:222,327`),
the payload decode arm (`saga/model.go:1610`), and event acceptance
(`saga/event_acceptance.go:125-126` → `{EventKindAssetUpdated}`), each following the
`ApplyAssetLock` line exactly.

Compensation clears the bit. The orchestrator does not need a second action for this: the
compensator issues `ApplyAssetKarma` with a `Clear: true` discriminator rather than adding a
near-duplicate `ClearAssetKarma`, keeping the acceptance table one entry wide. (`atlas-inventory`
already pairs `ApplyLock`/`ClearLock` internally; the saga surface stays single.)

### 6.3 `atlas-inventory` (FR-6.3–6.7)

Two layers, modelled on `ApplyLock` (`asset/processor.go:329-342`) and `ApplyAssetLock`
(`compartment/processor.go:1045-1077`):

- `asset.ProcessorImpl.ApplyKarma(mb)(transactionId, characterId)(a Model) error` — resolves the
  bit via `KarmaFlagFor(a.TemplateId())`, sets it with `AddFlag`, persists the `flag` column, and
  buffers the existing `UPDATED` status event. `ClearKarma` is its twin.
- `compartment.ProcessorImpl.ApplyAssetKarma(mb)(transactionId, characterId, inventoryType, slot)`
  — takes the per-character/per-inventory `LockRegistry` lock, resolves the compartment and the
  asset by slot, delegates, and logs. Plus the `…AndEmit` transactional wrapper.

The asset-layer method re-asserts gates 0c, 1, 2, 3 and 4 (FR-6.4). Gates 2 and 4 need item data,
which `atlas-inventory` reads from `atlas-data` the same way `atlas-trades` does for `tradeBlock`;
an unreadable lookup is a refusal, never a permissive default.

Gate 3 at this layer *is* the idempotency guarantee (FR-6.7): a redelivered command finds the bit
already set and refuses rather than double-marking. Because setting a bit is idempotent at the
bit level, the refusal is about the *audit* — a second scissors must never be silently consumed
against an already-marked item.

Persistence is the existing `flag uint16` column (`asset/entity.go:34`); the `UPDATED` event
carries it to the client with no relog (FR-6.5). No migration, no new entity.

---

## 7. Honouring and consuming the mark

### 7.1 The override (FR-7.1–7.3)

`checkRestrictions` (`atlas-trades/trade/restriction.go`) refuses on the untradeable flags *and*,
separately, on the WZ `tradeBlock` prop — both fatal, and untradeable items derive their
untradeability mostly from the latter. Karma overrides both, or it overrides nothing useful.

`assetView` gains `TemplateId` (needed for `KarmaFlagFor`) and the two flag/`tradeBlock` refusals
become conditional on the karma bit being clear. The other three rules — unknown compartment,
equipped slot, unreadable item data — are untouched and still fatal (FR-7.2). Note that
`errItemDataUnknown` stays *above* the `tradeBlock` check in the ordering, so an unreadable lookup
is never rescued by a karma mark.

`atlas-merchant`'s `IsListableItem(itemId, flag)` (`shop/validation.go:133-134`) already has both
arguments the resolver needs; its `ErrUntradeableItem` refusal becomes conditional on the same
predicate (FR-7.3). Its `ErrPetItem` and `ErrCashItem` refusals are untouched.

### 7.2 Consumption (FR-7.4–7.6) — inherited atomicity

Both transfer paths already move an asset by **destroying it on one side and re-materialising it on
the other from a snapshot that carries `flag`**:

- `atlas-trades`: `TradeSettlementPayload → TradeSettlementSide.Items[] → TradeEscrowItem.Snapshot`,
  an `AssetSnapshot` whose fields include `Flag uint16` (`libs/atlas-saga/payloads.go`).
- `atlas-merchant`: `listing.ItemSnapshot asset.AssetData`, persisted as `jsonb` on the listing row
  (`listing/entity.go:22`) and used to create the buyer's asset.

**The mark is therefore cleared in the snapshot, at the moment the receiving asset is built** —
not by a follow-up mutation on the delivered item. Three consequences fall out for free:

1. **FR-7.5 atomicity is structural.** The clear and the transfer are the same write. There is no
   window in which a delivered item still carries a free trade.
2. **FR-7.6 falls out unchanged.** A cancelled trade unwinds by replaying the *same* snapshot back
   to the original owner (`TradeUnwindPayload`), and the unwind path does not clear — so a staged-
   then-unstaged item keeps its mark. No extra code, and no risk of the two paths diverging.
3. **One place to be correct.** A "clear it after the trade completes" design would need the clear
   in the settlement path, the merchant-sale path, and a compensator for each; this needs the bit
   masked off at exactly two snapshot-consumption sites.

The clear itself is `ClearFlag(snapshot.Flag, KarmaFlagFor(snapshot.TemplateId))`, skipped when the
resolver reports no bit (pet class), so a pet passing through a trade is never touched.

---

## 8. Versions and coverage

`USE_CASH_ITEM` is already bound in every tenant socket-config template, so **no template change is
expected** (FR-8.4). If implementation finds one lacking the binding, insertion follows
`docs/packets/TEMPLATE_CONVENTIONS.md` (sorted `opCode`) and must pass
`tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh`.

`5520000` works on every version that binds `USE_CASH_ITEM` (FR-8.1). `5520001` works wherever its
data exists (§3.3 / OQ-4). Neither is a version gate in code.

The `USE_CASH_ITEM` row at `docs/packets/audits/STATUS.md:588` has three unverified columns. This
task adds a sub-body arm behind an existing opcode and changes no already-verified column's wire
layout; a `coverage-manifest.yaml` records the columns it does cover, and
`packet-completeness-critic` is run before the PR (FR-8.3).

---

## 9. Testing strategy

| Area | Test |
|---|---|
| FR-1 | Round-trip byte fixtures for `ItemUseKarmaScissors` at both `updateTime` positions |
| FR-2.4 | Karma vs seal cash-slot-type resolvers differ on every configured tenant version |
| §3.2 | Predicate table: `(scissorsKarma, targetKarma)` ∈ {(0,0)✗, (0,1)✓, (1,1)✓, (1,2)✗, (2,1)✗, (1,0)✗} |
| FR-3.5 | Each of the five readers: present value, absent → `0`, and `01002357` (Zakum Helmet) → `1` |
| FR-4.4 | `SetKarmaUsed(true)` → `KarmaUsed() == true` for an equip **and** a bundle in the six services whose models carry a template id; equip-only in `atlas-cashshop`, whose two types are equip-class by construction |
| FR-4.5 | Karma-marking an equip leaves `FlagSpikes` set/clear exactly as it was |
| §4.2 | `KarmaFlagFor`: equip → `0x10`, bundle → `0x02`, pet range → `(0, false)` |
| FR-5 | One case per gate 0a..4: refused, no saga created, distinct log reason, client unlocked |
| FR-6.7 | Second `ApplyAssetKarma` against a marked asset refuses at the inventory layer |
| FR-6.6 | Saga failing after the mark compensates it away |
| FR-7.1/7.2 | Marked asset stages despite `FlagUntradeable` *and* despite `tradeBlock`; unmarked still refuses; unreadable item data still refuses even when marked |
| FR-7.4/7.6 | Settlement snapshot arrives with the bit clear; unwind snapshot arrives with it set |

Test setup uses the project Builder pattern throughout — no `*_testhelpers.go`.

---

## 10. Acceptance criteria deltas

The PRD's §10 list stands, with these amendments arising from §1:

- FR-4.4's "all four services" becomes **all seven** (`atlas-inventory`, `atlas-channel`,
  `atlas-login`, `atlas-cashshop`, `atlas-consumables`, `atlas-storage`,
  `atlas-query-aggregator`).
- "The bit split is confirmed on the v83 IDB with the relevant symbols named, resolving OQ-2" is
  **done in this phase** (§1/OQ-2); implementation inherits it.
- FR-8.2's "`5520001` works from v84 forward and not before" is asserted as a **data-driven**
  property: with `5520001` present and carrying `karma = K`, only targets whose
  `tradeAvailable == K` are eligible; with it absent from the corpus, the item is unusable. No
  version literal appears in the assertion or in the code.
- New: pet-class targets are refused at both the channel and inventory layers, with a distinct log
  reason (OQ-5).
- New: the parsed karma fields are **integers**, not bools, end to end.

Full build & verification gate per CLAUDE.md is unchanged: `go test -race ./...`, `go vet ./...`,
`go build ./...` in every changed module; `docker buildx bake atlas-<svc>` for every service whose
`go.mod` was touched; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/lint.sh --check` clean.

---

## 11. Residual risks

| Risk | Mitigation |
|---|---|
| Equip `0x02` is spikes; a wrong-bit write renders spikes on every karma'd equip | `KarmaFlagFor` is the only bit selector; FR-4.5 test; no call site names a bit |
| Pet `0x01` aliases `FlagLock` | Pets refused at both layers; `KarmaFlagFor` returns `(0, false)` so the case cannot be forgotten |
| gms_v84's eligibility model is unverified | §3.3 — the predicate is a strict subset of both known models, so v84 is correct either way |
| A v95-era tenant ships scissors without `karma` | `targetKarma != 0` (gate 2) and gate 4 both refuse the resulting over-broad match |
| Seal and karma cash-slot types collide pre-95 | Version-scoped resolvers on both sides + FR-2.4 regression test |
| Seven-service flag fix touches services outside the feature | Each change is the same two-line correction with a paired round-trip test; no behaviour depended on the broken pair, since it never returned `true` |
