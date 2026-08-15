# Scissors of Karma — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-13

---

## 1. Overview

Scissors of Karma (`5520000`, `Item.wz/Cash/0552.img`, classification 552) is a cash-slot item that grants
one trade to an item that is otherwise untradeable. The player opens the Scissors UI, drags the target item
into it, and the server marks that item as "tradeable one more time". The canonical practical use is a boss
drop such as the Zakum Helmet: untradeable by data, made transferable once by the scissors.

The mechanic is a *pre-trade mutation of an item already in the character's inventory* — the scissors do not
participate in the trade itself. The client says so in as many words: after a successful use it writes
`SP_4664_YOU_HAVE_USED_THE_SCISSORS_OF_KARMA_SO_1_TIME_OF_TRADING_HAS_BEEN_ENABLED` to the chat log
(`CUIKarmaDlg::_SendConsumeCashItemUseRequest`, gms_v83 `@0x830FB5`).

Atlas has every structural prerequisite and none of the behavior. Classification 552 is already mapped in
`GetCashSlotItemType` (`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1103-1109`)
to cash-slot type `64` on GMS ≥ 95 and `63` otherwise, but no arm consumes it, so a use falls through to the
terminal `not implemented` warn at `:679`. The asset flag bitfield exists end-to-end — constants
(`libs/atlas-constants/asset/flag.go`), GORM column (`services/atlas-inventory/.../asset/entity.go:34`), REST
(`asset/rest.go:17`), and the wire short (`libs/atlas-packet/model/asset.go:247,318`) — and `atlas-trades`
already refuses to stage an untradeable asset (`services/atlas-trades/atlas.com/trades/trade/restriction.go`).
What is missing is anything that *writes* a karma mark and anything that *honors* one.

This task implements the family end-to-end: the `USE_CASH_ITEM` sub-body codec, the handler arm with
server-side re-validation of all three client gates, the WZ prop that decides eligibility, an
`atlas-inventory` in-place flag mutator driven through the existing saga machinery, and the trade-gate
override plus its consumption on trade completion. Task-128 (Item Tag / Sealing Lock) supplies the arm and
saga shape; task-222 (item-expiration extenders) supplies the "new WZ prop + new asset-mutation saga action"
file-touch shape.

## 2. Goals

Primary goals:

- A player who drags an eligible untradeable item into the Scissors of Karma UI has that item marked
  one-trade-enabled, with the scissors consumed exactly once and the mark visible without relog.
- The karma mark permits exactly one transfer — through a direct player trade or a hired-merchant listing —
  after which the item is untradeable again for its new owner.
- Ineligible targets are refused server-side with no scissors consumed and no state mutation, independently
  of what the client allowed.
- The karma mark is written to the flag bit the client actually reads, which differs by slot class.

Non-goals:

- Cash-shop *purchase* of scissors (the generic commodity path already exists).
- A new packet opcode. `USE_CASH_ITEM` already carries this; the task adds a sub-body arm.
- Fixing the general absence of untradeable gating on item *drop* and *NPC sale* — neither consults
  `FlagUntradeable` or `tradeBlock` today. That is a pre-existing gap, not karma's to close.
- Any atlas-ui surface.
- Storage / Duey / mailing transfer paths.

## 3. User Stories

- As a player who just cleared Zakum, I want to drag my untradeable Zakum Helmet into the Scissors of Karma
  so that I can pass it to my other character's account-mate one time.
- As a player, I want the scissors refused — and not consumed — when I drop them on an item the scissors
  don't cover, so a mis-drag doesn't cost me a cash purchase.
- As a player, I want a karma-marked item to become untradeable again once it has been traded, so the mark
  means "once", not "forever".
- As a player, I want the item tooltip to show the one-trade state immediately after use, without relogging.
- As a player, I want the client to unlock after a use — successful or refused — so my next action isn't
  swallowed.
- As an operator, I want every refused use to leave a log line naming the character, the scissors, the
  target, and which rule failed, so I can answer a support ticket without a database query.

## 4. Functional Requirements

### FR-1 — Wire contract (libs/atlas-packet)

The sub-body was derived from `CUIKarmaDlg::_SendConsumeCashItemUseRequest` at both ends of the supported
range. The two versions differ only in `updateTime` position, which the existing `ItemUse` common header
(`libs/atlas-packet/cash/serverbound/item_use.go:21-23`, `UpdateTimeFirst`) already models.

| gms_v83 `@0x830FB5` — opcode `0x4F` | gms_v95 `@0x7D7EF0` — opcode `0x55` |
|---|---|
| `Encode2(m_nPOS)` | `Encode4(get_update_time())` |
| `Encode4(m_nItemID)` | `Encode2(m_nPOS)` |
| `Encode4(m_nTargetTI)` | `Encode4(m_nItemID)` |
| `Encode4(m_nTargetPOS)` | `Encode4(m_nTargetTI)` |
| `Encode4(get_update_time())` | `Encode4(m_nTargetPOS)` |

Both opcodes agree with the `USE_CASH_ITEM` row at `docs/packets/audits/STATUS.md:588`, which already names
`CUIKarmaDlg::_SendConsumeCashItemUseRequest` among that opcode's senders.

- **FR-1.1** A discrete sub-body struct `ItemUseKarmaScissors` MUST be added under
  `libs/atlas-packet/cash/serverbound/`, carrying `int32 targetInventoryType`, `int32 targetSlot`, and the
  `updateTime` / `updateTimeFirst` pair. Both `Encode` and `Decode` MUST be implemented.
- **FR-1.2** The struct's field order and gating MUST be byte-identical to `ItemUseSeal`
  (`item_use_seal.go`). It MUST NOT reuse `ItemUseTargetSlot` (bare `int16`), which is the Item Tag /
  expiration-extender shape and is wrong here — karma carries a target inventory type.
- **FR-1.3** A discrete struct is required rather than an alias of `ItemUseSeal`, per
  `docs/packets/DISPATCHER_FAMILY.md`'s discrete-struct-per-mode rule.
- **FR-1.4** Round-trip byte-fixture tests MUST cover both `updateTime` positions.

### FR-2 — Classification and cash-slot type (libs/atlas-constants, atlas-channel)

- **FR-2.1** `libs/atlas-constants/item/constants.go` MUST define
  `ClassificationKarmaScissors = Classification(552)`.
- **FR-2.2** `GetCashSlotItemType`'s existing bare `if category == 552` branch
  (`character_cash_item_use.go:1103-1109`) MUST be rewritten to use the named constant. Its returned values
  (`64` on GMS ≥ 95, `63` otherwise, including JMS) MUST NOT change.
- **FR-2.3** The arm MUST match via a version-scoped resolver function
  `karmaScissorsCashSlotItemType(t tenant.Model) CashSlotItemType`, following the
  `viciousHammerCashSlotItemType` precedent (`character_cash_item_use.go:760-765`). A bare constant compare
  is forbidden here: pre-95 `CashSlotItemTypeSealTimed` is also `64`. The two are disjoint at runtime only
  because the seal arm recomputes itself to `65` on GMS ≥ 95 (`:261-265`) — a fragile coincidence that a
  version-scoped resolver on both sides makes safe.
- **FR-2.4** A test MUST assert that for every configured tenant version, the karma resolver and the seal
  resolver return different values. This is the regression guard for FR-2.3.

### FR-3 — Item data: the eligibility prop (atlas-data)

Eligibility is decided by a per-item WZ property, read by `CItemInfo::GetAppliableKarmaType`
(gms_v95 `@0x5C09F0`): for ids where `nItemID / 1000000 == 1` it reads `nAppliableKarmaType` off `EQUIPITEM`,
otherwise off `BUNDLEITEM`. The scissors carry their own karma type, loaded into a `KARMASCISSORSITEM`
record by `CItemInfo::RegisterKarmaScissorsItem` (`@0x5A1120`) from a property on the scissors' `info` node.

The client's test is **equality**, not non-zero: `GetAppliableKarmaType(target) != m_nKarmaType` refuses
(`CUIKarmaDlg::PutItem`, `@0x7D7BA0`). This matters directly for the second scissors variant (FR-8.2): a
distinct karma type gates a distinct item set, and a "non-zero" implementation would let either scissors cut
either set.

- **FR-3.1** The applicable-karma-type property MUST be parsed for **all five** item categories in
  `services/atlas-data/atlas.com/data/`: `equipment`, `consumable`, `setup`, `etc`, and `cash` — following
  the `tradeBlock` precedent, which is already read in all five (`equipment/reader.go:114`,
  `consumable/reader.go:49`, `setup/reader.go:47`, `etc/reader.go:47`, `cash/reader.go:84`). It MUST default
  to `0` when absent.
- **FR-3.2** The scissors' own karma type MUST be parsed off the cash reader for classification-552 items.
- **FR-3.3** Both MUST be exposed on the corresponding atlas-data REST models and mirrored on the
  channel-side consumer models, following the `tradeBlock` precedent (`cash/rest.go:53` et al.).
- **FR-3.4** The exact WZ property spelling MUST be resolved before implementation — see §9 OQ-1. The IDB
  reaches both names through `StringPool` ids (`0x3D5` for the container node, `5595` for the scissors' own
  type), which this pass did not resolve to literals. Implementation MUST NOT guess the spelling; it MUST be
  read out of a WZ extract or the ingested atlas-data corpus and recorded in the design doc.
- **FR-3.5** Reader unit tests MUST cover a present value, an absent-field default of `0`, and at least one
  real v83 item id verified to carry a non-zero value (Zakum Helmet is the expected case).

### FR-4 — The karma mark: which bit (libs/atlas-constants and four services)

The client reads the mark from `nAttribute`, and **the bit differs by slot class**:

- `GW_ItemSlotEquip::IsPossibleTradingItem` (gms_v95 `@0x4F6130`) returns `nAttribute & 0x10`.
- `GW_ItemSlotBundle::IsPossibleTradingItem` (gms_v95 `@0x4F67A0`) returns `nAttribute & 0x02`.

This is the same context-dependence that already makes `0x02` mean *spikes* on an equip and *karma* on a
bundle. Atlas's constant names invert the intuition: `FlagKarmaUse = 0x02`
(`libs/atlas-constants/asset/flag.go:8`) is the **bundle** bit, and `FlagKarmaEquip = 0x10` (`:11`) is the
**equip** bit.

Consequently the existing accessors are half-right in a way that never round-trips:
`SetKarmaUsed(true)` writes `FlagKarmaEquip` (0x10) — correct for equips only
(`services/atlas-inventory/.../asset/builder.go:147-153`) — while `KarmaUsed()` reads `FlagKarmaUse` (0x02) —
correct for bundles only (`asset/model.go:84`). A set followed by a get returns `false` for every asset. The
same inverted pair is duplicated in `atlas-channel`, `atlas-login`, and `atlas-cashshop`.

- **FR-4.1** `libs/atlas-constants/asset/flag.go` MUST document each bit's slot-class context in a comment
  citing the two `IsPossibleTradingItem` addresses. The constant *values* MUST NOT change — they match the
  client; only the naming and documentation mislead.
- **FR-4.2** A helper MUST be added to `libs/atlas-constants/asset` that resolves the correct karma bit from
  the asset's slot class (equip vs bundle), so no call site picks a bit by hand.
- **FR-4.3** `KarmaUsed()` and `SetKarmaUsed()` MUST be corrected to branch on slot class via FR-4.2, in all
  four services that carry the pair: `atlas-inventory`, `atlas-channel`, `atlas-login`, `atlas-cashshop`
  (`asset/reference_data.go:54,373` and its two builders).
- **FR-4.4** A test MUST assert `SetKarmaUsed(true)` → `KarmaUsed() == true` for both an equip asset and a
  bundle asset, in each of the four services. This is the regression guard for the round-trip defect.
- **FR-4.5** Setting the karma mark MUST NOT disturb any other bit in `flag`, in particular `FlagSpikes`
  (0x02) on an equip, which shares a value with the bundle karma bit.

### FR-5 — The handler arm (atlas-channel)

`CUIKarmaDlg::PutItem` (gms_v95 `@0x7D7BA0`) refuses in a fixed order. Every one of these is a client-side
check and MUST be re-validated server-side, because a crafted packet reaches the handler regardless.

| # | Client gate | Bit / rule | Server rule |
|---|---|---|---|
| 1 | `IsProtectedItem()` | `nAttribute & 0x01` = `FlagLock` | Refuse a Sealing-Lock'd target |
| 2 | `GetAppliableKarmaType(target) != m_nKarmaType` | WZ prop, **equality** | Refuse a target whose type ≠ the scissors' type |
| 3 | `IsPossibleTradingItem()` | 0x10 equip / 0x02 bundle | Refuse a target already karma-marked |

- **FR-5.1** The arm MUST verify the claimed CASH slot really holds the claimed scissors template, via the
  existing `cashItemInSlotFunc` ownership pre-check (`character_cash_item_use.go:55-59,728`).
- **FR-5.2** The target inventory type off the wire MUST be validated as one of the five known inventory
  types before conversion; an unknown value is a refusal, not a panic.
- **FR-5.3** The target slot MUST be non-negative. A negative slot is an equipped item and MUST be refused.
- **FR-5.4** An empty target slot MUST be refused.
- **FR-5.5** The three gates in the table above MUST be applied in that order, each producing a distinct
  refusal reason.
- **FR-5.6** A target that is *already tradeable* MUST be refused — karma exists to unlock an untradeable
  item, and marking a tradeable one is a no-op that would still consume the scissors. "Tradeable" here means
  neither `FlagUntradeable`/`FlagMergeUntradeable` nor the WZ `tradeBlock` prop applies, matching the two
  conditions `atlas-trades` enforces (FR-7.1).
- **FR-5.7** Every refusal MUST log at warn level naming the character id, the scissors template id, the
  target inventory type and slot, the target template id where resolvable, and the failing rule. No scissors
  are consumed and no state is mutated on any refusal.
- **FR-5.8** On success the arm MUST create a saga with two steps: `DestroyAsset` for the scissors and a new
  `ApplyAssetKarma` action for the target, following the `ItemTagUse` two-step shape
  (`character_cash_item_use.go:226-259`).
- **FR-5.9** The client takes an exclusive-request lock before sending — gms_v83 gates on
  `CanSendExclRequest(500, 0)` and then sets the lock (`@0x830FB5`). The server MUST therefore unlock the
  client on **every** outcome, success and refusal alike. Per the ExclRequest contract, a refusal that
  returns silently leaves the client wedged until the next unlocking packet.

### FR-6 — Applying the mark (libs/atlas-saga, atlas-saga-orchestrator, atlas-inventory)

- **FR-6.1** A new saga action `ApplyAssetKarma` MUST be added with a payload carrying character id, target
  inventory type, and target slot — modeled on `ApplyAssetLock` (`libs/atlas-saga/model.go:220-221`,
  `payloads.go:1088-1100`, `unmarshal.go:576-586`).
- **FR-6.2** The orchestrator MUST dispatch it and accept its completion event, following the `ApplyAssetLock`
  wiring (`saga/handler.go:947-950`, `saga/event_acceptance.go:125-126`).
- **FR-6.3** `atlas-inventory` MUST gain an `ApplyKarma` asset processor that sets the slot-class-correct
  karma bit in place and emits the existing `UPDATED` status event — modeled directly on `ApplyLock`
  (`asset/processor.go:329-342`) — plus a slot-addressed compartment wrapper following `ApplyAssetLock`
  (`compartment/processor.go:1045-1077`).
- **FR-6.4** `ApplyKarma` MUST re-assert the FR-5 gates at the inventory layer. The channel handler's checks
  are advisory across a service boundary; the owning service is the authority.
- **FR-6.5** The mutation MUST be persisted to the `flag` column and MUST reach the client through the
  existing `UPDATED` event path, with no relog required.
- **FR-6.6** A compensator MUST clear the karma bit, so a saga that fails after the mark is applied does not
  leave a free trade behind.
- **FR-6.7** The operation MUST be idempotent: re-applying karma to an already-marked asset MUST be a
  refusal (FR-5.5 gate 3), not a second mark.

### FR-7 — Honoring and consuming the mark (atlas-trades, atlas-merchant)

`checkRestrictions` (`services/atlas-trades/atlas.com/trades/trade/restriction.go`) currently refuses a stage
on the asset flag **and, separately, on the WZ `tradeBlock` prop** — both fatal. Untradeable items derive
their untradeability mostly from `tradeBlock`, so a karma mark that only defeats the flag check would still
be refused. Per the scope decision, karma overrides both.

- **FR-7.1** A karma-marked asset MUST pass `checkRestrictions` despite `FlagUntradeable`,
  `FlagMergeUntradeable`, **and** the WZ `TradeBlock` prop.
- **FR-7.2** The karma override MUST NOT weaken the other restrictions: unknown compartment, equipped
  (negative) slot, and unreadable item data still refuse.
- **FR-7.3** `atlas-merchant`'s hired-merchant listing gate (`shop/validation.go:133-134`,
  `ErrUntradeableItem`) MUST accept a karma-marked asset on the same terms.
- **FR-7.4** On successful completion of a trade or a merchant sale, the karma mark MUST be cleared from the
  transferred asset, so it arrives untradeable for its new owner. This is what makes the grant "one time".
- **FR-7.5** FR-7.4 MUST be atomic with the transfer. A transfer that completes while the clear fails would
  hand over an item with a free trade still on it.
- **FR-7.6** A karma-marked asset that is *staged and then un-staged* (trade cancelled) MUST retain its mark.
  Only a completed transfer consumes it.

### FR-8 — Versions and item ids

- **FR-8.1** `5520000` is the primary scissors and MUST work on every configured tenant version that binds
  `USE_CASH_ITEM`.
- **FR-8.2** `5520001` is introduced at GMS v84. It MUST be supported from v84 forward and MUST NOT be
  usable on versions that predate it. Because eligibility is an equality test against the scissors' own karma
  type (FR-3), the two variants gate different target sets and MUST NOT be collapsed.
- **FR-8.3** The `USE_CASH_ITEM` row at `docs/packets/audits/STATUS.md:588` shows three unverified columns.
  This task MUST NOT regress any already-verified column, and MUST record which columns its own change
  covers in a coverage manifest.
- **FR-8.4** No tenant socket-config template change is expected — `USE_CASH_ITEM` is already bound. If the
  implementation finds a template that lacks the binding, adding it MUST follow
  `docs/packets/TEMPLATE_CONVENTIONS.md` (sorted `opCode` insertion) and pass the template guards.

## 5. API Surface

No new REST endpoints.

Modified atlas-data REST responses — the applicable-karma-type field is added to the item payload for all
five categories, and the scissors' own karma type to the cash payload (FR-3.3). Both are additive integer
fields defaulting to `0`; existing consumers are unaffected.

New Kafka contracts:

- A `COMMAND_TOPIC_COMPARTMENT` command for the karma application, following the existing lock command
  (`services/atlas-inventory/.../kafka/consumer/compartment/consumer.go:371,393`).
- No new event type — the existing asset `UPDATED` status event carries the changed `flag` (FR-6.5).

## 6. Data Model

No new entities and no migration.

The karma mark is a bit in the existing `flag uint16` column on the asset entity
(`services/atlas-inventory/atlas.com/inventory/asset/entity.go:34`), which is already persisted, already
tenant-scoped through its owning compartment, already exposed over REST (`asset/rest.go:17`), and already
written to the wire as a short (`libs/atlas-packet/model/asset.go:247,318`).

Bit assignment, per FR-4:

| Bit | Equip meaning | Bundle meaning |
|---|---|---|
| `0x01` | lock (Sealing Lock) | lock |
| `0x02` | spikes | **karma mark** |
| `0x08` | untradeable | untradeable |
| `0x10` | **karma mark** | (unused by karma) |

The one data-model risk is FR-4.5: on an equip, the bundle karma bit `0x02` is the spikes bit. A careless
"set 0x02 for karma" would render spikes on every karma'd equip.

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-packet` | New `ItemUseKarmaScissors` sub-body codec + round-trip fixtures (FR-1) |
| `libs/atlas-constants` | `ClassificationKarmaScissors`; slot-class karma-bit helper + flag docs (FR-2.1, FR-4.1, FR-4.2) |
| `libs/atlas-saga` | `ApplyAssetKarma` action, payload, unmarshal (FR-6.1) |
| `atlas-channel` | Handler arm, version-scoped type resolver, three server-side gates, saga creation, ExclRequest unlock; consumer-model karma-type fields; `KarmaUsed`/`SetKarmaUsed` fix (FR-2, FR-4.3, FR-5) |
| `atlas-data` | Applicable-karma-type prop across five readers + scissors' own type; REST exposure (FR-3) |
| `atlas-saga-orchestrator` | Dispatch, event acceptance, compensation (FR-6.2, FR-6.6) |
| `atlas-inventory` | `ApplyKarma` asset + compartment processors, Kafka command consumer, re-asserted gates; `KarmaUsed`/`SetKarmaUsed` fix (FR-4.3, FR-6.3–6.7) |
| `atlas-trades` | Karma override of flag + `tradeBlock` in `checkRestrictions`; mark consumption on completion (FR-7.1, FR-7.2, FR-7.4–7.6) |
| `atlas-merchant` | Karma override in listing validation; mark consumption on sale (FR-7.3, FR-7.4) |
| `atlas-login`, `atlas-cashshop` | `KarmaUsed`/`SetKarmaUsed` correctness fix only (FR-4.3, FR-4.4) |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every processor resolves its tenant from context (`tenant.MustFromContext`). Version
  behavior (FR-2.3, FR-8.2) is resolved from the tenant model, never hard-coded.
- **Client wire values are config-resolved.** Per DOM-25, the inventory-type values and the cash-slot type
  are derived from the tenant's configured version, not from literals in the arm.
- **Security.** All three client gates plus the ownership pre-check are re-validated server-side (FR-5), and
  again at the owning service (FR-6.4). A crafted `USE_CASH_ITEM` MUST NOT be able to karma-mark an item the
  scissors do not cover, mark an already-marked item twice, or mark an item the character does not own.
- **Atomicity.** Scissors consumption and mark application are one saga (FR-5.8) with a compensator
  (FR-6.6). Transfer and mark-clear are atomic (FR-7.5).
- **Observability.** Every refusal logs the character, scissors, target, and failing rule (FR-5.7).
- **No regression.** No wire change to any already-verified matrix column (FR-8.3). No behavioral change to
  the Sealing Lock, Item Tag, or expiration-extender arms.

## 9. Open Questions

- **OQ-1 (blocks FR-3).** The exact WZ property spelling for the applicable-karma-type field and for the
  scissors' own karma type. The IDB reaches both through `StringPool` ids (`0x3D5`, `5595`) that this pass
  did not resolve to literals. Must be read from a WZ extract or the ingested atlas-data corpus during
  design — not guessed.
- **OQ-2 (blocks FR-4).** The karma bit values are confirmed on gms_v95 only. `IsPossibleTradingItem` and
  `IsProtectedItem` are unnamed in the v83, v84, and v92 IDBs. Design MUST locate and name them in at least
  the v83 IDB and confirm the `0x10` / `0x02` split holds at the baseline before FR-4 is implemented. Unnamed
  is not absent.
- **OQ-3 (FR-7.4).** Whether a hired-merchant *sale* consumes the mark on the same terms as a direct trade,
  or whether the merchant path should be listing-only. Spec'd as consuming; confirm against client behavior.
- **OQ-4 (FR-8.2).** Whether `5520001` at v84 differs from `5520000` only in karma type, or also in target
  breadth. Resolve from the v84 WZ data.
- **OQ-5 (FR-3.1).** Whether any classification-552 item covers pet-slot targets. `GW_ItemSlotPet` has its
  own `IsPossibleTradingItem` (gms_v95 `@0x4F6A70`), which this pass did not decompile. If pets are in
  range, FR-3.1's five categories and FR-4.2's two slot classes both need a third arm.

## 10. Acceptance Criteria

- [ ] `ItemUseKarmaScissors` decodes and encodes byte-identically to the derived layout, with round-trip
      fixtures covering both `updateTime` positions (FR-1).
- [ ] `ClassificationKarmaScissors` is defined and `GetCashSlotItemType`'s 552 branch uses it, returning
      unchanged values (FR-2.1, FR-2.2).
- [ ] A test asserts the karma and seal cash-slot-type resolvers differ on every configured version (FR-2.4).
- [ ] The applicable-karma-type prop parses for all five item categories and is exposed over REST, with the
      spelling verified against WZ data rather than guessed (FR-3).
- [ ] Reader tests cover present, absent-default, and one verified real v83 item id (FR-3.5).
- [ ] `SetKarmaUsed(true)` → `KarmaUsed() == true` for both an equip and a bundle asset, in all four services
      carrying the pair (FR-4.4).
- [ ] Karma-marking an equip leaves `FlagSpikes` unchanged (FR-4.5).
- [ ] The bit split is confirmed on the v83 IDB with the relevant symbols named, resolving OQ-2 (FR-4).
- [ ] Dragging an eligible untradeable item into the scissors marks it, consumes exactly one scissors, and
      the mark is visible in the tooltip without relog (FR-5.8, FR-6.5).
- [ ] Each of the three client gates, plus locked / empty-slot / equipped-slot / unknown-inventory-type /
      already-tradeable / not-owned, is refused server-side with no scissors consumed, no state mutation, a
      distinct log reason, and the client unlocked (FR-5).
- [ ] A crafted packet naming a target the scissors' karma type does not cover is refused (FR-5.5, FR-6.4).
- [ ] A karma-marked asset stages in a trade despite `FlagUntradeable` and despite WZ `tradeBlock`, while an
      unmarked untradeable asset still refuses (FR-7.1).
- [ ] A karma-marked asset lists in a hired merchant shop (FR-7.3).
- [ ] After a completed trade the item is untradeable for the receiver; a second trade attempt refuses
      (FR-7.4).
- [ ] A cancelled trade leaves the mark intact (FR-7.6).
- [ ] A saga failing after the mark is applied compensates the mark away (FR-6.6).
- [ ] `5520000` works on every version binding `USE_CASH_ITEM`; `5520001` works from v84 forward and not
      before (FR-8.1, FR-8.2).
- [ ] No already-verified `USE_CASH_ITEM` matrix column regresses; a coverage manifest records the columns
      this task covers (FR-8.3).
- [ ] Sealing Lock, Item Tag, and expiration-extender arms are behaviorally unchanged.
- [ ] Full build & verification gate per CLAUDE.md: `go test -race ./...`, `go vet ./...`, `go build ./...`
      clean in every changed module; `docker buildx bake atlas-<svc>` for every service whose `go.mod` was
      touched; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` clean.
