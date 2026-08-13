# Item-Expiration Extenders (Magical Sandglass) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-13

---

## 1. Overview

MapleStory's cash shop sells "Magical Sandglass" items (`Item.wz/Cash/0550.img`, classification 550) that a
player drags onto a time-limited piece of equipment to push its expiration date further out. Each sandglass
carries a fixed grant (`addTime`, in seconds) and a fixed ceiling (`maxDays`) beyond which the target's
expiration may not be pushed. The in-client description states the contract plainly:

> "Drag and drop this onto a piece of equipment that has a time limit to extend the time limit by 7days.
> This cannot be used on cash items, and the time limit cannot be extended past 30 days, starting from today."
> — `String.wz/Cash.img/5500001/desc` (GMS v83, local extract)

Atlas has every structural prerequisite in place but no implementation. `CharacterCashItemUseHandleFunc`
(`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`) already dispatches
~14 cash slot-item types; classification 550 is not one of them, is not defined in
`libs/atlas-constants/item/constants.go`, and its `addTime` / `maxDays` WZ fields are parsed nowhere in the
tree. A player who owns a sandglass today can drag it onto an equip and nothing happens beyond a
`not implemented` warn in the channel log.

This task implements the family end-to-end: parse the two WZ fields, define the classification, add the
`CASH_ITEM_USE` handler arm, add an inventory command that extends (rather than replaces) an asset's
expiration, and drive it through the existing saga machinery so the sandglass is consumed atomically with
the extension. The Sealing Lock arm (`character_cash_item_use.go:264-330`) is the structural template
throughout — same sub-body shape, same two-step saga, same asset-mutation event path back to the client.

## 2. Goals

Primary goals:

- A player who drags a Magical Sandglass onto an eligible time-limited equip has that equip's expiration
  extended by the item's WZ `addTime`, clamped to `now + maxDays`, with the sandglass consumed exactly once.
- Ineligible targets (permanent equips, cash equips, non-equip inventories, empty slots) are rejected with
  no item consumed and no state mutation.
- The extension is durable (persisted in `atlas-inventory`) and visible to the client without relog.
- Behavior is identical across every configured tenant template that the item family exists in.

Non-goals:

- Cash-shop *purchase* flow for sandglasses (the commodity/cash-shop path already exists generically).
- Extending expiration on cash items, pets, or any non-equip asset.
- The `protectTime` / Sealing Lock semantics — a separate, already-implemented feature that this task must
  not regress.
- A new packet opcode. `CASH_ITEM_USE` is already registered in 9 of 10 GMS templates; this task adds an
  arm inside the existing dispatcher, not a new socket handler.

## 3. User Stories

- As a player, I want to drag a 7-day Magical Sandglass onto my 5-day-remaining Ilbi Throwing Star equip so
  that its expiration moves out and I keep using it.
- As a player, I want the sandglass *not* to be consumed when I drop it on a permanent equip, so I don't
  lose a cash purchase to a mis-drag.
- As a player, I want the extended expiration to appear in the item tooltip immediately, without relogging.
- As a player holding a 99-day sandglass, I want the extension to stop at the 30-day ceiling rather than
  silently failing, so the item still does something.
- As an operator, I want a rejected use to leave a log line naming the character, item, and reason, so I can
  answer a support ticket without a database query.

## 4. Functional Requirements

### FR-1 — Item data (atlas-data)

- **FR-1.1** `services/atlas-data/atlas.com/data/cash/reader.go` MUST parse `info/addTime` (seconds,
  integer) and `info/maxDays` (days, integer) alongside the existing `protectTime` parse (`reader.go:78`),
  defaulting both to `0` when absent.
- **FR-1.2** Both fields MUST be exposed on the cash REST model
  (`services/atlas-data/atlas.com/data/cash/rest.go`) and mirrored on the channel-side consumer model
  (`services/atlas-channel/atlas.com/channel/data/cash/rest.go`), following the `protectTime` precedent.
- **FR-1.3** Reader unit tests MUST cover the five verified v83 values (see §6) plus the absent-field
  default.

The v83-verified values, extracted from `Item.wz/Cash/0550.img.xml`:

| Item id | `addTime` (s) | = days | `maxDays` | `String.wz` name |
|---|---|---|---|---|
| 5500000 | 86400 | 1 | 30 | 마법의 모래시계 (Korean string only) |
| 5500001 | 604800 | 7 | 30 | `[7days]Magical Sandglass` |
| 5500002 | 1728000 | 20 | 30 | `[20days]Magical Sandglass` |
| 5500005 | 4320000 | 50 | 30 | *(no `String.wz/Cash.img` entry in v83)* |
| 5500006 | 8553600 | 99 | 30 | *(no `String.wz/Cash.img` entry in v83)* |

Note that 5500005 and 5500006 grant 50 and 99 days against a 30-day ceiling — they can never deliver their
nominal grant under the clamp rule. This is treated as upstream WZ data, not a defect to correct: the
implementation reads the data as-is (see FR-3.4).

### FR-2 — Classification and dispatch (libs/atlas-constants, atlas-channel)

- **FR-2.1** `libs/atlas-constants/item/constants.go` MUST define
  `ClassificationExpirationExtender = Classification(550)`, placed in ascending order among the existing
  cash classifications (between `ClassificationRemoteStore` (547) and `ClassificationViciousHammer` (557)).
- **FR-2.2** `GetCashSlotItemType` MUST map classification 550 to the client's slot-item-type value for the
  family. **The numeric value is unverified and MUST be derived from the client during design** — it is not
  present in the handler's current enum block, and guessing it would silently route the arm to the wrong
  sub-body decode. Derivation follows the project's IDA discipline
  (`GetCashSlotItemType` in `CWvsContext`, cross-checked across the in-scope version IDBs).
- **FR-2.3** The dispatch value MUST be version-resolved if the client's enum diverges across the in-scope
  versions, following the `viciousHammerCashSlotItemType` / `CashSlotItemTypeSealTimedV95` precedent
  (`character_cash_item_use.go:658-673, 704-713`) rather than a single hard-coded constant.
- **FR-2.4** No hard-coded literal for the mode/type byte outside the version-resolving helper (DOM-25:
  client wire values are config- or version-resolved).

### FR-3 — Use semantics (atlas-channel handler arm)

- **FR-3.1 Sub-body.** The arm MUST decode a `(int32 inventoryType, int32 slot)` sub-body with the standard
  `updateTimeFirst` header gate. If the client's case-550 arm is byte-identical to `ItemUseSeal`
  (`libs/atlas-packet/cash/serverbound/item_use_seal.go`), the existing codec MUST be reused rather than
  duplicated; a new codec is added only if the derived read order differs. The decision is made from the
  derived client read order, not by inspection of the Go type.

- **FR-3.2 Eligibility gates.** The use MUST be rejected — logged, no consumption, no mutation — when any
  of the following holds:
  1. `inventoryType != inventory.TypeValueEquip`.
  2. The target slot is empty (`GetItemInSlot` errors).
  3. The target's expiration is zero (a permanent item has no time limit to extend).
  4. The target is a cash equip. **Signal:** `asset.Model.CashId() != 0`
     (`services/atlas-channel/atlas.com/channel/asset/model.go:100`; the inventory-side model states the
     same predicate explicitly as `IsCashEquipment()`,
     `services/atlas-inventory/atlas.com/inventory/asset/model.go:106-108`).
  5. The target is lock-expiring rather than genuinely time-limited — i.e. `Locked()` is set and the
     expiration is the Sealing Lock's protect window. Extending a lock window with a sandglass is not the
     feature; the two expiration semantics MUST NOT be conflated.
  6. The clamped result would not advance the target's expiration (already at or past `now + maxDays`).
     Consuming a sandglass for a zero-second extension is a player-visible loss and MUST NOT happen.

- **FR-3.3 Equipped targets.** The equip inventory addresses equipped items with negative slot values. The
  arm MUST handle a negative target slot the same way it handles a positive one (the WZ description's
  Korean text, "장착 중인", explicitly refers to currently-equipped gear), or explicitly reject it — the
  choice MUST be made from the client's actual behavior during design, not left ambiguous.

- **FR-3.4 Extension formula.** Let `E` be the target's current expiration, `A` the item's `addTime`
  seconds, `D` the item's `maxDays`, and `N` the server's current time. The new expiration is:

  ```
  cap      = N + (D * 24h)
  proposed = E + A
  new      = min(proposed, cap)
  ```

  The cap is anchored to **now**, not to the original expiration — matching the description's
  "starting from today". When `proposed > cap`, the result is **clamped and the item is still consumed**
  (explicit product decision); the use is rejected outright only under FR-3.2(6), when the clamp would
  yield no advance at all.

- **FR-3.5 Atomicity.** Consumption and extension MUST run as a single saga with two steps, mirroring
  `SealingLockUse` (`character_cash_item_use.go:296-330`): a `DestroyAsset` step for the sandglass and an
  extension step for the target. A new saga type (e.g. `ExpirationExtenderUse`) MUST be registered in
  `libs/atlas-saga` and wired into the orchestrator's timer and compensator tables alongside
  `ItemTagUse` / `SealingLockUse` / `IncubatorUse`
  (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go:175,205,237`;
  `.../saga/compensator.go:267`). A saga type that is absent from those tables gets no timeout and no
  compensation — a silent class of stuck saga.

- **FR-3.6 No EnableActions.** The cash-item-use arms that mutate inventory without warping send no
  unlock (see the Sealing Lock and kite arms). This arm MUST follow suit unless design derives otherwise
  from the client.

### FR-4 — Persistence (atlas-inventory)

- **FR-4.1** A new compartment command (e.g. `EXTEND_EXPIRATION`, body `{slot, expiration}`) MUST be added
  to `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go`, consumed in
  `kafka/consumer/compartment/consumer.go`, and implemented as a compartment-processor method following the
  `ApplyAssetLock` shape (`compartment/processor.go:1045-1075`).
- **FR-4.2** The asset-level mutation MUST set the expiration **without** touching `FlagLock`. Reusing
  `asset.ApplyLock` is forbidden: it unconditionally sets `FlagLock` and it rejects any asset that is
  unlocked with a non-zero expiration (`asset/processor.go:329-341`) — which is precisely this feature's
  only valid target.
- **FR-4.3** The server MUST re-validate the cap server-side before persisting. The channel-side clamp is a
  UX gate, not a trust boundary; a forged command MUST NOT be able to set an arbitrary expiration.
- **FR-4.4** The mutation MUST emit the existing asset `UPDATED` status event via
  `UpdatedEventStatusProvider`, carrying the new `Expiration` in `AssetData`
  (`kafka/message/asset/kafka.go:32,69-71`).

### FR-5 — Client feedback

- **FR-5.1** No new clientbound packet is required. **Verified:** `atlas-inventory`'s asset `UPDATED` event
  carries `AssetData.Expiration`, and `atlas-channel`'s `handleAssetUpdatedEvent`
  (`kafka/consumer/asset/consumer.go:269-296`) rebuilds the asset and announces an `INVENTORY_OPERATION`
  add-entry for the slot, which rewrites the client's slot record and refreshes the tooltip.
- **FR-5.2** The sandglass's own removal is announced by the existing `DestroyAsset` saga step's event path;
  no additional work.
- **FR-5.3** Every rejection branch under FR-3.2 MUST log at warn with character id, sandglass item id,
  target inventory type, target slot, and the specific reason — matching the Sealing Lock arm's logging
  density.

### FR-6 — Version scope

- **FR-6.1** In scope: **all** configured tenant templates. The user's stated availability is GMS v83+ and
  JMS v176+; the family's per-version item presence and `addTime` / `maxDays` values MUST be verified
  during design against each version's data (local dumps are v83-only — see §9).
- **FR-6.2** For any in-scope version where the client's slot-item-type value for classification 550
  differs, FR-2.3's version-resolving helper covers it. For any version where the item family does not
  exist, no template or code change is needed — the arm is simply never reached.
- **FR-6.3** No socket-config template edits are expected (no new opcode). If design finds that a template
  lacks `CASH_ITEM_USE` registration for an in-scope version, that gap MUST be recorded, and any template
  edit MUST satisfy `tools/template-opcode-order-guard.sh` and
  `tools/template-duplicate-binding-guard.sh`.

## 5. API Surface

No new REST endpoints. Modified surfaces:

**`atlas-data` — `GET /api/data/cash/{itemId}`** (existing route; response gains two fields):

```jsonc
{
  "data": {
    "type": "cash",
    "id": "5500001",
    "attributes": {
      "protectTime": 0,
      "addTime": 604800,   // NEW — seconds granted
      "maxDays": 30        // NEW — ceiling, in days from now
    }
  }
}
```

Both fields are additive and omit-empty-safe; existing consumers are unaffected.

**Kafka — `COMMAND_TOPIC_COMPARTMENT`** (new command type):

```jsonc
{
  "transactionId": "<uuid>",
  "characterId": 12345,
  "type": "EXTEND_EXPIRATION",
  "body": {
    "slot": -8,
    "expiration": "2026-09-12T00:00:00Z"   // absolute, server-computed and re-validated
  }
}
```

**Kafka — `EVENT_TOPIC_ASSET_STATUS`**: no schema change; the existing `UPDATED` event carries the new
expiration.

Error cases: all rejections are server-side, logged, and silent to the client (no error packet). The
sandglass remains in the player's inventory, which is itself the feedback.

## 6. Data Model

No new database tables and no migration. The only persisted field that changes is
`assets.expiration` on the target row, written through the existing
`updateFlagAndExpiration` administrator path (`asset/administrator.go:65`) or a new
expiration-only sibling — design's call, but the write MUST NOT alter `flag` (FR-4.2).

New in-memory / wire types:

| Type | Location | Purpose |
|---|---|---|
| `ClassificationExpirationExtender` | `libs/atlas-constants/item/constants.go` | Classification 550 |
| `CashSlotItemTypeExpirationExtender` (name TBD) | `atlas-channel/.../character_cash_item_use.go` | Client slot-item-type; value derived in design |
| `ExtendExpirationCommandBody` | `atlas-inventory/.../compartment/kafka.go` | `{slot, expiration}` |
| `ExtendAssetExpirationPayload` | `libs/atlas-saga` | Saga step payload |
| `ExpirationExtenderUse` saga type | `libs/atlas-saga` | Saga type constant |
| `AddTime`, `MaxDays` | `atlas-data` + `atlas-channel` cash REST models | WZ fields |

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-constants` | Add `ClassificationExpirationExtender` (550). |
| `libs/atlas-packet` | Reuse `cash/serverbound/item_use_seal.go` if the read order matches; otherwise add a sibling codec with `Encode`/`Decode` + byte-fixture test. |
| `libs/atlas-saga` | Add saga type `ExpirationExtenderUse` and the extend-expiration step action + payload. |
| `atlas-data` | Parse + expose `addTime`, `maxDays` in the cash reader/REST model. |
| `atlas-channel` | New `CASH_ITEM_USE` arm: dispatch, eligibility gates, clamp, saga creation. Mirror the two new cash fields on the channel-side cash REST model. |
| `atlas-inventory` | New `EXTEND_EXPIRATION` compartment command + consumer + processor + asset-level expiration-only mutation emitting `UPDATED`. |
| `atlas-saga-orchestrator` | Register the new saga type in the timer table and compensator branches; handle the new step action. |
| `atlas-configurations` | Expected: none. Only if FR-6.3 uncovers a missing `CASH_ITEM_USE` registration. |

## 8. Non-Functional Requirements

- **Multi-tenancy.** All processors resolve tenant from context (`tenant.MustFromContext(ctx)`); item data
  is fetched per-tenant through `atlas-data`, never from a package-level table. The version-resolving
  helper (FR-2.3) reads `tenant.Model` region/version, never a global.
- **Security / trust boundary.** FR-4.3 — the inventory service re-derives and re-validates the cap from
  item data rather than trusting the channel-supplied absolute timestamp.
- **Idempotency.** The saga is transaction-id keyed; a redelivered `EXTEND_EXPIRATION` command MUST NOT
  stack a second extension. Redelivery is at-least-once in this cluster and non-idempotent handlers have
  duplicated item state before — the extension MUST be a set-to-absolute-value, not an increment, and the
  step must be guarded by the existing saga step-completion machinery.
- **Observability.** Warn-level logs on every rejection branch (FR-5.3); the saga's own step logging covers
  the success path.
- **Testing.** Table-driven unit tests for: the WZ reader values, the clamp formula (under-cap, at-cap,
  over-cap, already-past-cap), each FR-3.2 rejection gate, and the inventory processor's flag-preservation
  invariant. Codec change (if any) carries a byte-fixture test.
- **Performance.** One additional `atlas-data` cash lookup per use — the same call the Sealing Lock arm
  already makes. No hot path affected.

## 9. Open Questions

1. **What `CashSlotItemType` does the client map classification 550 to?** Unverified. Must be derived from
   the client (IDA) per version before any dispatch code is written. This is the single blocking unknown.
2. **Does the client's case-550 sub-body match `ItemUseSeal` byte-for-byte?** Drives FR-3.1's reuse-vs-new
   codec decision. Derive from the same IDA pass as (1).
3. **Are equipped (negative-slot) targets accepted by the client's drag-drop?** Drives FR-3.3. The Korean
   description implies yes; confirm against the client's drop-target validation.
4. **Per-version item presence and values.** The local WZ dumps are GMS 83.1 only
   (`tmp/<tenant>/GMS/83.1/`). `addTime` / `maxDays` for v87 / v92 / v95 / JMS must be read from the live
   `atlas-data` per version before FR-6.1 can be called verified. The user reports availability as
   GMS v83+ and JMS v176+; that is a starting hypothesis, not a verified fact.
5. **Should the lock-expiration gate (FR-3.2(5)) reject, or should a locked item's expiration be
   extendable?** Recommended: reject, since a Sealing Lock window and an item time limit are different
   semantics and Atlas already refuses to conflate them (`asset/processor.go:329-332`). Confirm.
6. **5500005 / 5500006 have no `String.wz/Cash.img` name entry in v83.** They will render nameless in the
   client. Out of scope to fix, but worth confirming they are actually obtainable before treating them as
   in-scope items.

## 10. Acceptance Criteria

- [ ] `ClassificationExpirationExtender = Classification(550)` exists in `libs/atlas-constants` and is used
      by the dispatch path (no bare `550` literal).
- [ ] `atlas-data` parses `addTime` and `maxDays`; unit tests assert all five v83 values from §4's table and
      the absent-field default of 0.
- [ ] Both fields are reachable from `atlas-channel` via the cash REST model.
- [ ] The client slot-item-type for classification 550 is derived from the client, documented in the design
      with its evidence (IDB, address, version), and resolved version-aware where it diverges.
- [ ] Dragging a sandglass onto an equip with a future expiration extends that expiration by `addTime`,
      clamped to `now + maxDays`, and consumes exactly one sandglass.
- [ ] Dragging onto a permanent equip, a cash equip (`CashId() != 0`), a locked equip, an empty slot, or a
      non-equip inventory consumes nothing and mutates nothing; each path emits a distinct warn log.
- [ ] A target already at or past `now + maxDays` rejects the use — the sandglass is not consumed.
- [ ] `FlagLock` is provably unchanged by the extension (asserted in an `atlas-inventory` unit test).
- [ ] The extension is re-validated server-side in `atlas-inventory`; a command carrying an
      out-of-bounds absolute expiration is clamped or rejected there, not trusted.
- [ ] The new saga type appears in the orchestrator's timer table and compensator branches; a failed
      extension step compensates the consumed sandglass back into the inventory.
- [ ] Redelivery of the extension command does not stack a second extension.
- [ ] The client tooltip shows the new expiration without a relog (verified live, not inferred).
- [ ] `go test -race ./...` and `go vet ./...` clean in every changed module.
- [ ] `docker buildx bake atlas-<svc>` clean for every service whose `go.mod` was touched.
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
      `tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh` all clean from the repo root.
- [ ] Code review (`superpowers:requesting-code-review`) run before the PR is opened.
