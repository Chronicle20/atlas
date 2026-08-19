# Duey Parcel Delivery — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-19

---

## 1. Overview

Duey is MapleStory's asynchronous player-to-player delivery service. A player talks to the
Duey NPC (9010009), addresses a parcel to another character by name, attaches up to one item
stack and/or a quantity of meso, pays a meso fee, and walks away. The parcel enters durable
custody. Some time later the recipient — who need not have been online at any point during
the send — opens Duey, sees the parcel in a mailbox tab, and either receives it into their
inventory or discards it. A Quick Delivery Ticket (item 5330000) lets a player open the same
send interface from anywhere on the map, without walking to the NPC.

Atlas has none of this. A grep for `duey|parcel` across `services/` and `libs/` returns only
two incidental hits: the item classification constant (`libs/atlas-constants/item/constants.go:109`
`ClassificationDueyCoupon = Classification(533)`, surfaced as `"duey-coupon"` in
`services/atlas-data/atlas.com/data/item/classify.go:203` and
`services/atlas-ui/src/lib/items/taxonomy.ts:123`), and an unimplemented branch in
`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1410`
that matches `ClassificationDueyCoupon` and then does nothing. There is no service, no
handler, no packet codec, and no NPC script. Both relevant packet families are ❌ in every
version column of the coverage matrix.

This feature matters because Duey is the only sanctioned mechanism in the game for
transferring items to an offline player. Without it, every item hand-off requires both
parties online and co-located in a trade window. Duey is also the natural companion to the
already-implemented Fredrick custody flow (`services/atlas-merchant/atlas.com/merchant/frederick`),
which solves the adjacent problem of returning a hired merchant's goods to its owner — Duey
generalises the same custody primitive to a sender-chosen recipient.

## 2. Goals

Primary goals:

- Players can send a parcel (one item stack and/or meso) to any character in the same world
  by name, paying a fee, whether or not the recipient is online.
- Parcels persist durably and tenant-scoped, surviving service restarts and both parties
  logging out.
- Recipients can list, receive, and discard parcels through the in-game Duey interface, and
  are notified when a parcel is waiting.
- The Quick Delivery Ticket (5330000) opens the send interface from anywhere, consuming the
  ticket in place of the standard NPC surcharge.
- Parcels that go unclaimed expire and return to their sender rather than vanishing.
- A character with parcels in flight — in either direction — cannot world-transfer into a
  state where those parcels are stranded.
- The `PARCEL` and `DUEY_ACTION` packet families are implemented and verified across every
  applicable version column, promoting their coverage-matrix cells.

Non-goals:

- Cross-world delivery. Recipient lookup is scoped to the sender's world (see §4.2 and OQ-1).
- Multiple item stacks per parcel. The client's send interface accepts one item slot; this
  PRD does not extend it.
- An atlas-ui administrative surface for inspecting or moderating parcels.
- Any MTS or auction-house adjacency.
- Migrating the existing Fredrick custody into a shared abstraction. The two services stay
  separate; §7 notes only the pattern reuse.

## 3. User Stories

- As a player, I want to send an item to a friend who is offline, so that I do not have to
  coordinate a simultaneous login for a trade.
- As a player, I want to send meso to another character, so that I can pay someone without
  meeting them.
- As a player, I want to attach a short note to my parcel, so that the recipient knows what
  it is for.
- As a player, I want to see what is waiting for me and choose to receive it, so that I am
  not forced to accept items I have no inventory room for.
- As a player, I want to discard a parcel I do not want, so that my mailbox does not fill up.
- As a player who bought a Quick Delivery Ticket, I want to send from where I stand, so that
  I do not have to travel to Duey.
- As a player whose parcel was never claimed, I want my items and meso back, so that a
  mistyped recipient name does not cost me the goods permanently.
- As a player, I want to be told when a parcel arrives for me, so that I do not have to check
  Duey speculatively.
- As a player with an unclaimed parcel, I want the world-transfer purchase to tell me why it
  is blocked, so that I can resolve it rather than guess.

## 4. Functional Requirements

### 4.1 Reference behavior and its authority

Cosmic's implementation is the behavioral reference for this feature:
`src/main/java/client/processor/npc/DueyProcessor.java`,
`src/main/java/net/server/channel/handlers/DueyHandler.java`,
`src/main/java/server/DueyPackage.java`, and
`src/main/java/net/server/task/DueyFredrickTask.java`. Values quoted in this PRD were read
from that source at spec time and are cited inline.

**Cosmic is a behavioral reference, not a wire-format authority.** Every opcode, mode byte,
and field order in §5 MUST be re-derived from the client binary per
`docs/packets/PROCESS.md` and `docs/packets/DISPATCHER_FAMILY.md`. The Cosmic action codes
reproduced in §5.3 are a starting hypothesis to check against the IDB, never a source to
copy from.

### 4.2 Send flow

FR-1. A send request carries: inventory type, item slot position, item quantity, meso
amount, an optional message, the recipient's character name, and a quick-delivery flag.

FR-2. **Fee.** The fee is computed from the meso amount on a tiered percentage scale. Read
from Cosmic `Trade.getFee` (`src/main/java/server/Trade.java:89-105`):

| Meso amount | Fee |
|---|---|
| ≥ 100,000,000 | 6% |
| ≥ 25,000,000 | 5% |
| ≥ 10,000,000 | 4% |
| ≥ 5,000,000 | 3% |
| ≥ 1,000,000 | 1.8% |
| ≥ 100,000 | 0.8% |
| < 100,000 | 0 |

FR-3. **Surcharge.** A non-quick send (i.e. at the NPC) adds a flat 5,000 meso surcharge on
top of the tiered fee. A quick send does NOT add the surcharge; it consumes a Quick Delivery
Ticket instead. This is the direction stated in Cosmic `DueyProcessor.java:308-313` — the
ticket replaces the surcharge, it does not add to it.

FR-4. The sender must hold at least `mesoAmount + fee` meso. If not, the request is rejected
with the not-enough-meso result and no state changes.

FR-5. A request with neither an item (quantity < 1) nor meso (amount 0) is invalid and
rejected. A request whose `mesoAmount + fee` overflows or is negative is invalid and
rejected. Cosmic treats both as packet-edit evidence and disconnects; Atlas MUST reject and
log at warn level rather than disconnect (see NFR-5).

FR-6. An attached message longer than 100 characters is invalid and rejected.

FR-7. **Recipient resolution.** The recipient name must resolve to exactly one character in
the **sender's world**. If it resolves to none, the request is rejected with the
name-does-not-exist result. Cross-world sends are out of scope; see OQ-1 for the
tenant-with-one-world case.

FR-8. **Same-account block.** If the resolved recipient's account equals the sender's
account, the request is rejected with the same-account result. This is why recipient
resolution must return the account id, not only the character id.

FR-9. **Quick delivery validation.** A request with the quick flag set MUST be rejected
unless the sender actually holds a Quick Delivery Ticket (5330000). On success the ticket is
consumed. A quick-flagged request from a player without the ticket is rejected and logged at
warn.

FR-10. On success, atomically: deduct `mesoAmount + fee` from the sender, remove the item
stack from the sender's inventory, consume the ticket if quick, and create the parcel record.
Any failure rolls the whole set back (see §7, saga).

FR-11. The sender receives the successfully-sent result and the interface re-enables.

### 4.3 Delivery timing

FR-12. **A parcel becomes receivable 24 hours after it is sent.** Before that instant the
recipient does not see it in their parcel list and cannot receive it.

This is a deliberate divergence from Cosmic, which delivers instantly (Cosmic has no
delay field; `DueyProcessor.createPackage` timestamps at insert and the receive path does not
gate on it). It matches official GMS behavior. Consequences:

- The parcel record needs a distinct `receivable_at` alongside `created_at` (§6).
- The recipient-facing list query filters on `receivable_at <= now`.
- End-to-end testing cannot observe a receive without either time control or a test-only
  override; the design phase must choose a mechanism (OQ-4).

FR-13. Expiry (FR-18) is measured from send time, not from `receivable_at`.

### 4.4 Receive and discard flows

FR-14. The recipient's parcel list shows every parcel addressed to them, in their world,
whose `receivable_at` has passed and which is not already received, discarded, expired, or
returned.

FR-15. **Receive.** The recipient must have a free inventory slot of the item's type; if not,
the request is rejected with the no-free-slots result and the parcel stays pending. On
success, atomically: award the item to the recipient's inventory, award the meso, and mark
the parcel received.

FR-16. **Unique-item guard.** If the parcel holds an item the recipient may only hold one of
and they already hold one, the receive is rejected with the receiver-with-unique result and
the parcel stays pending.

FR-17. **Discard.** The recipient may discard a pending parcel. The contents are destroyed —
they do NOT return to the sender. Discard is irreversible and the client already confirms it
via `CUIFadeYesNo`.

### 4.5 Expiry and return

FR-18. A parcel unclaimed 30 days after send expires. The 30-day window is Cosmic's
(`DueyProcessor.runDueyExpireSchedule`, `DueyProcessor.java:485` — `c.add(Calendar.DATE, -30)`).

FR-19. **On expiry the parcel returns to its sender**, restoring the item and the meso
amount. This diverges from Cosmic, which deletes the row and destroys the contents
(`removePackageFromDB` then `DELETE FROM dueypackages`). Return-to-sender was chosen so a
mistyped recipient name is recoverable.

FR-20. The returned parcel is itself delivered as a parcel addressed to the original sender,
so the same receive flow applies and no items are pushed into an offline player's inventory.
A returned parcel is flagged as such so it can be presented differently and so it cannot
itself expire into an infinite return loop — a returned parcel that expires again is
destroyed, not re-returned.

FR-21. The fee is NOT refunded on expiry or return.

FR-22. Expiry runs as a periodic background task, following the existing pattern in
`services/atlas-merchant/atlas.com/merchant/frederick/task.go`.

### 4.6 Notification

FR-23. When a parcel becomes receivable and its recipient is online, the recipient is
notified in-game.

FR-24. When a recipient logs in with at least one receivable, unnotified parcel, they are
notified. Notification state is tracked per parcel so a player is not re-notified on every
login for the same parcel — mirroring the `LastNotified` column and `NotificationEntity` in
`services/atlas-merchant/atlas.com/merchant/frederick/entity.go`.

### 4.7 Quick Delivery Ticket

FR-25. Using item 5330000 opens the Duey send interface at the player's current position,
without an NPC. This implements the currently-empty `ClassificationDueyCoupon` branch at
`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1410`.

FR-26. The ticket is consumed on a successful send, not on opening the interface. A player
who opens the interface and closes it without sending keeps the ticket.

### 4.8 World transfer interaction

Because delivery is same-world only (FR-7), a character who transfers worlds would strand
every parcel they have in flight: an outbound parcel whose recipient is now in a different
world from the sender, and an inbound parcel the character can no longer receive.

FR-27. **A new destination-independent eligibility gate blocks world transfer for a character
with any parcel in flight in either direction** — outbound sent-but-not-yet-resolved, or
inbound receivable-but-not-yet-received.

This extends the existing gate table in
`services/atlas-character/atlas.com/character/pending_change/processor_eligibility.go`. That
file already implements eleven gates with an exact precedent: gate 11 `checkMtsHolding`
blocks transfer on a live MTS holding, with the documented rationale that auto-cancelling is
not reversible by compensation. A pending parcel is the same shape of problem.

FR-28. The new gate is **gate 12, `parcel_pending`**, destination-INDEPENDENT. It must be
added to BOTH `evaluateTransferEligibility` and `evaluateTransferEligibilityIndependent`, in
the same relative order, per that file's documented contract — otherwise the CHECK-time
handler (`cash_shop_check_transfer_world_possible.go`) and the BUY-time handler disagree.

FR-29. The gate follows the file's established conventions exactly: a `parcelPending` field
on `gateDeps`, wired in `productionGateDeps()`, overridable via `withTransferEligibilityGates`
for tests; a dependency error fails CLOSED and reports the distinct `check_unavailable`
reason rather than the affirmative `parcel_pending` reason.

FR-30. The rejection reason string is `parcel_pending`, and it must be surfaced to the player
through whatever mapping the existing reasons use.

FR-31. Blocking is chosen over auto-cancelling: the player can resolve it themselves by
receiving or discarding their parcels, or by waiting out expiry, and a silent auto-return
during a transfer would be a cross-world asset movement — exactly what FR-7 forbids.

## 5. API Surface

### 5.1 Packet families

Both families are ❌ in every version column today.

**`PARCEL` — clientbound, `CParcelDlg::OnPacket`** (`docs/packets/audits/STATUS.md:455`). A
mode-prefix dispatcher; it MUST be implemented per `docs/packets/DISPATCHER_FAMILY.md`
(discrete struct per mode, config-resolved mode byte, per-mode body functions, per-mode
verification). Applicable columns and opcodes as recorded in the matrix:

| v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | JMS185 |
|---|---|---|---|---|---|---|---|---|---|
| ⬜ | ⬜ | 0x120 | 0x12C | 0x142 | 0x149 | 0x153 | 0x175 | 0x17D | 0x160 |

**`DUEY_ACTION` — serverbound** (`docs/packets/audits/STATUS.md:596`). Registered fnames:
`CTabSend::SendParcel`, `CTabReceive::ReceiveParcel`, `CTabReceive::DiscardParcel`,
`CTabQuickSend::SendQuickDelivery`, `CParcelDlg::CloseParcelDlg`, `CUIFadeYesNo::OnButtonClicked`.

| v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | JMS185 |
|---|---|---|---|---|---|---|---|---|---|
| ⬜ | ⬜ | ⬜ | ⬜ | 0x041 | 0x041 | 0x044 | 0x047 | 0x046 | 0x039 |

### 5.2 Version span asymmetry

`PARCEL` has an opcode on v72 and v79; `DUEY_ACTION` does not. A clientbound parcel dialog
with no serverbound counterpart on those two versions is not a coherent protocol. This must
be resolved before the packet work is planned — see OQ-2.

### 5.3 Mode bytes (hypothesis, requires IDB derivation)

Cosmic's `DueyProcessor.Actions` enum (`DueyProcessor.java:65-90`) gives the following
hypothesis. **These are unverified against the client binary and MUST NOT be hard-coded from
this table.** Per `docs/packets/DISPATCHER_FAMILY.md`, mode bytes are config-resolved.

Serverbound (`DUEY_ACTION`): `0x00` recv-item, `0x02` send-item, `0x04` claim-package,
`0x05` remove-package, `0x07` close.

Clientbound (`PARCEL`): `0x08` open, `0x09` send-enable-actions, `0x0A` not-enough-meso,
`0x0B` incorrect-request, `0x0C` name-does-not-exist, `0x0D` same-account-error,
`0x0E` receiver-storage-full, `0x0F` receiver-unable-to-recv,
`0x10` receiver-storage-with-unique, `0x11` meso-limit, `0x12` successfully-sent,
`0x13` recv-unknown-error, `0x14` recv-enable-actions, `0x15` recv-no-free-slots,
`0x16` recv-receiver-with-unique, `0x17` recv-successful-msg, `0x1B` recv-package-msg.

Note the gaps (`0x01`, `0x03`, `0x06`, `0x18`–`0x1A`). Cosmic's enum is not necessarily
exhaustive of the client's arms; the IDB derivation is authoritative for which arms exist.

### 5.4 REST — atlas-parcel

JSON:API conventions, tenant-scoped via context per the standard Atlas pattern.

- `GET /parcels?filter[recipientId]=&filter[worldId]=` — list parcels addressed to a
  character. Supports the receivable filter backing FR-14.
- `GET /parcels?filter[senderId]=` — list parcels a character has sent that are still in
  flight. Backs the world-transfer gate's outbound half.
- `GET /parcels/{parcelId}` — fetch one.
- `GET /characters/{characterId}/parcel-status` — a narrow boolean endpoint answering "does
  this character have any parcel in flight, either direction?", consumed by the
  world-transfer gate (FR-27). A dedicated endpoint rather than two list calls, matching how
  `mtsHolding` is a single narrow lookup in `processor_eligibility.go`.
- `POST /parcels` — create (send). Invoked by the saga, not directly by the channel.
- `PATCH /parcels/{parcelId}` — state transitions (received, discarded, expired, returned).

Error cases map to the §5.3 clientbound result modes: insufficient meso, unknown recipient,
same account, recipient inventory full, recipient unique-item conflict, meso limit, generic
incorrect request.

## 6. Data Model

New service `atlas-parcel`. Entity shape follows
`services/atlas-merchant/atlas.com/merchant/frederick/entity.go`, which is the closest
existing precedent for tenant-scoped item custody with a jsonb asset snapshot.

`parcels`:

| Field | Type | Notes |
|---|---|---|
| `Id` | uuid | primary key |
| `TenantId` | uuid | not null; every query tenant-scoped |
| `WorldId` | world.Id | not null; scopes recipient resolution (FR-7) |
| `SenderId` | uint32 | not null, indexed |
| `SenderAccountId` | uint32 | not null; supports the same-account check (FR-8) |
| `SenderName` | string | denormalised for display without a lookup |
| `RecipientId` | uint32 | not null, indexed |
| `RecipientAccountId` | uint32 | not null |
| `Message` | string | max 100 chars (FR-6) |
| `MesoAmount` | uint32 | meso delivered to the recipient, excluding the fee |
| `FeePaid` | uint32 | recorded for audit; not refunded (FR-21) |
| `ItemId` | uint32 | nullable — a meso-only parcel has no item |
| `ItemType` | byte | inventory type |
| `Quantity` | uint16 | |
| `ItemSnapshot` | jsonb | full asset snapshot, as `frederick.ItemEntity.ItemSnapshot` |
| `Status` | string | see state machine below |
| `Quick` | bool | whether sent via Quick Delivery Ticket |
| `Returned` | bool | true if this parcel is the return leg of an expired one (FR-20) |
| `CreatedAt` | time | send time; expiry measured from here (FR-13) |
| `ReceivableAt` | time | `CreatedAt + 24h` (FR-12) |
| `ExpiresAt` | time | `CreatedAt + 30d` (FR-18) |
| `ResolvedAt` | time | nullable; when it left `pending` |
| `LastNotified` | time | nullable (FR-24) |

Status state machine: `pending` → `received` \| `discarded` \| `expired`. `expired` triggers
creation of a new `pending` parcel with `Returned = true` addressed to the original sender
(FR-19, FR-20); an `expired` parcel that already had `Returned = true` creates nothing.

Indexes: `(TenantId, RecipientId, Status)` for the mailbox query;
`(TenantId, SenderId, Status)` for the outbound half of the world-transfer gate;
`(TenantId, Status, ExpiresAt)` for the expiry sweep.

Migration: a single new table via GORM `AutoMigrate`, following
`frederick.Migration`. No changes to existing tables.

## 7. Service Impact

**`atlas-parcel` (new).** Owns the parcel table, the state machine, the expiry/return task,
and the notification task. Onboarded per `docs/adding-a-new-service.md`. Structure mirrors
`atlas-merchant/frederick`: `entity.go`, `model.go`, `builder.go`, `administrator.go`,
`provider.go`, `processor.go`, `rest.go`, `resource.go`, `producer.go`, `task.go`.

**`atlas-channel`.** New handler for the `DUEY_ACTION` serverbound family and the writer side
of the `PARCEL` clientbound dispatcher, modelled on the existing
`socket/handler/storage_operation.go` (the closest existing custody-UI handler). Implements
the `ClassificationDueyCoupon` branch at `character_cash_item_use.go:1410` (FR-25).

**`atlas-saga-orchestrator`.** Two new sagas. Send: `DestroyAssetFromSlot` (the item),
`AwardMesos` (negative, the cost), consume ticket if quick, create parcel. Receive:
`AwardAsset`, `AwardMesos`, mark received. Both compose from actions that already exist in
`libs/atlas-saga` (`AwardAsset`, `DestroyAssetFromSlot`, `AwardMesos` — confirmed present in
`libs/atlas-saga/payloads.go` and the action list in `world_transfer_test.go:97-98`), so no
new saga action type should be needed; the design phase confirms this.

**`atlas-character`.** Gate 12 `parcel_pending` in
`pending_change/processor_eligibility.go` — a `parcelPending` field on `gateDeps`, a
`checkParcelPending` method, entries in both evaluate functions, and the REST client in
`pending_change/requests.go` (FR-27 to FR-30).

**`atlas-inventory`.** No direct changes; asset movement flows through the sagas. Capacity
and unique-item checks (FR-15, FR-16) are queries against existing endpoints.

**`atlas-npc-conversations`.** NPC 9010009 entry point opening the Duey interface.

**`libs/atlas-packet`.** Two new families across their applicable version spans (§5.1).

**`libs/atlas-constants`.** NPC id for Duey (9010009) and the Quick Delivery Ticket id
(5330000), checked against existing constants before defining new ones per the repository
convention.

## 8. Non-Functional Requirements

NFR-1. **Multi-tenancy.** Every parcel query is tenant-scoped via context. Cross-tenant
recipient resolution is impossible by construction; a provider-level tenant test is required,
matching `frederick`'s and `storage`'s `provider_tenant_test.go`.

NFR-2. **Atomicity.** Neither send nor receive may partially apply. A crash between meso
deduction and parcel creation must not destroy the sender's meso; a crash between item award
and status update must not duplicate the item. Both flows are sagas with compensation.

NFR-3. **Idempotency.** A replayed receive must not award twice. Follow the idempotency
pattern already tested in
`services/atlas-storage/atlas.com/storage/storage/processor_idempotency_test.go`.

NFR-4. **Mailbox query performance.** The recipient list query is on the interactive path and
must be served by the `(TenantId, RecipientId, Status)` index without a table scan.

NFR-5. **Validation posture.** Cosmic responds to malformed requests by disconnecting the
client and raising an autoban. Atlas MUST reject the request and log at warn level instead.
Disconnection is not an Atlas validation mechanism and an autoban integration is out of scope.

NFR-6. **Observability.** Every state transition logs at info with tenant, parcel id, sender,
recipient, and reason. Every rejection logs its reason code.

NFR-7. **Expiry sweep bounds.** The sweep must be batched and must not hold a long
transaction over the whole table. It must be safe to run concurrently on multiple instances
(or explicitly leader-elected) — the design phase decides which, following whatever
`frederick/task.go` does today.

NFR-8. **Fee arithmetic.** The tiered fee is computed in integer arithmetic with the exact
truncation the table in FR-2 implies (`meso * 18 / 1000`, not `meso * 0.018`). Overflow is
checked before deduction (FR-5).

## 9. Open Questions

**OQ-1 (recipient scope).** Recipient resolution is same-world (FR-7), decided at spec time.
Confirm during design how world id is obtained for the recipient — Cosmic's
`getAccountCharacterIdFromCNAME` queries by name alone with no world predicate, so Atlas needs
a world-scoped name lookup that may not exist yet on `atlas-character`.

**OQ-2 (version span asymmetry).** `PARCEL` has opcodes on v72/v79 but `DUEY_ACTION` does not
(§5.2). Determine from the IDB whether the serverbound op exists on those versions under a
different fname, whether the export is missing it, or whether the client genuinely has a
read-only parcel dialog there. This changes the scope of the packet work and must be answered
before planning.

**OQ-3 (dispatcher arm completeness).** Cosmic's action enum has gaps at `0x01`, `0x03`,
`0x06`, and `0x18`–`0x1A` (§5.3). Derive the true arm set from `CParcelDlg::OnPacket`; the
Cosmic enum is not authoritative.

**OQ-4 (testing the 24-hour delay).** FR-12 makes the happy path unobservable in a normal
test run. Decide the mechanism: an injectable clock on the processor, a configurable delay
whose test value is zero, or a direct `ReceivableAt` override in test setup. Prefer the
injectable clock — it keeps production behavior fixed and needs no configuration surface.

**OQ-5 (meso limit).** The clientbound modes include a meso-limit result (`0x11` in the §5.3
hypothesis). Determine the actual cap — whether it is a per-parcel cap, the recipient's meso
ceiling, or both — and where it is enforced.

**OQ-6 (GM restriction).** Cosmic gates Duey behind a minimum GM level for GM accounts
(`DueyProcessor.java:293`, `MINIMUM_GM_LEVEL_TO_USE_DUEY`). Decide whether Atlas wants an
equivalent restriction; it is not required by any goal above.

**OQ-7 (returned-parcel presentation).** FR-20 flags a return leg. Decide whether the client
can visually distinguish it, or whether the distinction is server-side only and conveyed
through the message field.

**OQ-8 (notification transport).** FR-23/FR-24 need a delivery mechanism. Determine whether
this reuses `atlas-messages`, the `frederick` notification pattern, or a direct channel
broadcast.

## 10. Acceptance Criteria

Functional:

- [ ] A player can send an item-only, meso-only, or item-and-meso parcel to a character in
      their world via NPC 9010009.
- [ ] The fee matches the FR-2 tier table exactly at each boundary (99,999 / 100,000 /
      999,999 / 1,000,000 / 4,999,999 / 5,000,000 / 9,999,999 / 10,000,000 / 24,999,999 /
      25,000,000 / 99,999,999 / 100,000,000), asserted by unit test.
- [ ] A non-quick send adds the 5,000 surcharge; a quick send does not and consumes the
      ticket instead (FR-3, FR-9, FR-26).
- [ ] A send to a character on the sender's own account is rejected with the same-account
      result.
- [ ] A send to a nonexistent name is rejected with the name-does-not-exist result.
- [ ] A send by a player who cannot afford `meso + fee` is rejected and changes no state.
- [ ] A send with an over-100-character message is rejected.
- [ ] A quick-flagged send without a ticket is rejected and logged at warn — the session is
      NOT disconnected (NFR-5).
- [ ] A parcel is not visible to its recipient before `ReceivableAt` and is visible after.
- [ ] A recipient with no free slot of the item's type is rejected with no-free-slots and the
      parcel remains pending.
- [ ] A recipient who already holds a unique item is rejected with receiver-with-unique and
      the parcel remains pending.
- [ ] A discarded parcel destroys its contents and does not return to the sender.
- [ ] A parcel unclaimed at 30 days becomes `expired` and produces a return-leg parcel to the
      original sender with `Returned = true`.
- [ ] A return-leg parcel that itself expires is destroyed and produces no further parcel.
- [ ] Item 5330000 opens the send interface with no NPC nearby.
- [ ] An online recipient is notified when a parcel becomes receivable; a recipient logging in
      with a waiting parcel is notified once, not on every subsequent login.

World transfer:

- [ ] A character with an outbound in-flight parcel is refused world transfer with reason
      `parcel_pending`.
- [ ] A character with an inbound receivable parcel is refused world transfer with reason
      `parcel_pending`.
- [ ] The refusal is produced by BOTH `CheckTransferEligibility` and
      `CheckTransferEligibilityIndependent`, asserted by test in
      `services/atlas-character/atlas.com/character/pending_change`.
- [ ] An `atlas-parcel` outage during the gate check yields `check_unavailable`, not
      `parcel_pending`, and fails closed.
- [ ] A character with no parcels is unaffected.

Packets:

- [ ] `PARCEL` implemented as a dispatcher family per `docs/packets/DISPATCHER_FAMILY.md`,
      with a discrete struct per mode and no hard-coded mode byte.
- [ ] `packet-audit dispatcher-lint` exits 0, and `PARCEL` is not added to
      `dispatcher-lint-baseline.yaml`.
- [ ] Every applicable `PARCEL` and `DUEY_ACTION` cell is promoted to verified in
      `docs/packets/audits/STATUS.md`, or OQ-2 has explicitly and in writing reduced the
      span, with the reduction recorded in the task folder.
- [ ] `packet-audit matrix --check`, `fname-doc --check`, and `operations --check` all exit 0.

Non-functional:

- [ ] Tenant isolation asserted by a provider-level tenant test.
- [ ] Send and receive are saga-driven with compensation; a mid-flow failure leaves no meso
      or item duplicated or destroyed, asserted by a rollback test.
- [ ] A replayed receive awards once, asserted by an idempotency test.
- [ ] Test setup uses the project's Builder pattern; no `*_testhelpers.go` files.

Gate:

- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `backend-guidelines-reviewer` and `plan-adherence-reviewer` both clear before PR.
