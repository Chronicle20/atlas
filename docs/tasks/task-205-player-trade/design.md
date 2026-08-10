# Player-to-Player Trade — Architecture & Design

Task: task-205-player-trade
Status: Draft for review
Created: 2026-08-09
Input: [`prd.md`](prd.md) (approved)

---

## 0. What this document decides

The PRD left seven open questions and asserted an escrow model. This design:

1. **Resolves PRD open questions 1, 2 and 5 with IDA evidence** (§2, §5) — the
   trade room needs no new enter-result encoder, trade completion is `LEAVE`
   (mode 10) with a status byte, and meso is logically reserved rather than
   escrowed.
2. **Rejects the PRD's escrow-at-staging model** (FR-3.2) and replaces it with
   reserve-at-staging / move-at-settlement (§5). The PRD's NFR claim that
   "escrow lives in the saga/inventory layer" is not true of the current
   primitives: `release_from_character` is a soft-delete with no owner-side
   durable record, so an escrow-at-staging crash orphans assets with nothing to
   reconcile from. This is the single biggest change from the PRD.
3. **Reduces the packet scope**: three new clientbound bodies, not five, and no
   new room encoder (§4).
4. **Defers PRD open questions 3, 4 and the per-version mode bytes** to the
   documented per-cell derivation procedure, with the procedure and decision
   rule made explicit (§10) rather than left as "unknown".
5. **Records findings the PRD did not know about** (§11), including a wrong
   `packet-audit:fname` on an existing codec and a previously unmodelled
   clientbound arm (mode 21).

Everything else in the PRD stands as written.

---

## 1. Reference-version evidence base

All IDA findings below are from **GMS v83** (`MapleStory_dump.exe.i64`), the
project's baseline column and the richest-named handler IDB. Addresses are v83
image addresses. Other versions are derived at implementation time (§10).

### 1.1 Mini-room base frame (already modelled)

`CMiniRoomBaseDlg::OnEnterResultBase` @`0x65ec3d` reads, after the room-type
byte consumed by `OnEnterResultStatic` @`0x65dff3`:

```
Decode1  -> this+51   (capacity)
Decode1  -> this+50   (recipient position)
loop { Decode1 slot; slot < 0 (0xFF) terminates; DecodeAvatar; DecodeStr name }
then virtual vtable+72   (subclass enter-result tail)
```

`CMiniRoomBaseDlg::OnLeaveBase` @`0x65edb5` reads `Decode1 slot`, then calls
virtual `vtable+76` with the slot; the subclass reads the trailing status byte.

Both match the existing `interaction.Room` encoder (`libs/atlas-packet/interaction/room.go:148`)
and `CharacterInteractionLeaveBody` (`interaction_body.go:157`).

### 1.2 `CTradingRoomDlg` clientbound dispatcher

`CTradingRoomDlg`'s mode dispatcher is `sub_7C1F6D` @`0x7c1f6d`:

| Mode | Handler | Body read |
|---|---|---|
| 15 (`0x0F`) | `sub_7C1FB7` @`0x7c1fb7` | `Decode1` side, `Decode1` trade slot, `GW_ItemSlotBase` |
| 16 (`0x10`) | `sub_7C208E` @`0x7c208e` | `Decode1` side, `Decode4` meso (**absolute assignment**, `this[v3+115] = Decode4`) |
| 17 (`0x11`) | `CTradingRoomDlg::OnTrade` @`0x7c20bc` | **nothing** — bodyless |
| 21 (`0x15`) | `sub_7C21BD` @`0x7c21bd` | **nothing** — bodyless |

`CCashTradingRoomDlg::OnPacket` @`0x4833b4` dispatches **15/16/17 only** — the
cash room has no mode-21 arm.

Mode 21 shows `SP_3977` ("Players that are level 15 and below may only trade
1 million mesos per day"), clears the local confirm flag (`this[111] = 0`) and
re-enables both confirm buttons. It is the server's *un-confirm / meso-refused*
arm. It was not in the PRD.

Mode 17 (`OnTrade`) reads no body; it sets `this[112] = 1`, redraws, and
**immediately sends serverbound opcode 123 mode `0x14` (TRANSACTION)** carrying
the client's own `{itemId, itemCRC}` list.

### 1.3 Trade-room enter-result tail: there is none

`CTradingRoomDlg`'s vtable is `off_B37448` (ctor @`0x7c1c76`). Slot +72 (the
subclass enter-result tail) is `0x48314D` = `nullsub_94` — an empty function.

**The trade room's ENTER_RESULT body is exactly the base frame**: room type,
capacity, position, `{slot, avatar, name}` visitors, `0xFF`. Nothing follows.

This resolves PRD open question 1: no third encoder, no new struct. See §4.1.

### 1.4 Trade completion is `LEAVE` with a status byte

`CTradingRoomDlg`'s vtable +76 is `0x7C221D` (`CTradingRoomDlg::OnLeave`). It
reads one byte and branches:

| Status | Client behaviour (StringPool id) |
|---|---|
| 2 | `SP_406` "Trade cancelled by the other character" |
| 7 | Success. Computes the meso delta from local `CharacterData`; if positive shows `SP_408` "Trade successful. Received %d mesos after fees", otherwise `SP_407` "Trade successful, please check the results" |
| 8 | `SP_409` "Trade unsuccessful" |
| 9 | `SP_410` "…there are some items which you cannot carry" |
| 12 | `SP_411` "…the other person's on a different map" |
| 13 | `SP_5566` "The game file has been damaged, so you cannot participate in an item trade" (CRC mismatch) |

The dialog is closed via `(*(*this+52))(this, 8)` before the notice is shown.

This resolves PRD open question 2: **completion is not a distinct mode.** It is
`LEAVE` (10) + slot + status, which the existing `CharacterInteractionLeaveBody`
already encodes. FR-8.5 therefore needs **no new codec** — only new
`leaveReason` tenant-table keys (§4.3).

Note the success message's meso figure is read from the client's own character
data, not from the packet. The server must therefore have already applied the
meso award before sending `LEAVE 7`, or the client prints a stale/zero figure.
This ordering constraint drives §6.4.

### 1.5 Serverbound confirm vs transaction

- `CTradingRoomDlg::Trade` @`0x7c39a0` sends `Encode1(0x11)` + CRC list. Gated
  behind a `YesNo` prompt.
- `CCashTradingRoomDlg::Trade` @`0x485dcd` also sends `Encode1(0x11)` + CRC list.
- The **only** `0x14` sender is `CTradingRoomDlg::OnTrade` @`0x7c20bc`, which
  fires automatically on receipt of clientbound mode 17.

So `TRANSACTION` (0x14) is **not a user action** — it is the client's
CRC-attestation reply to the server's mode-17 prompt. This has two consequences:

1. The existing `packet-audit:fname CCashTradingRoomDlg::Trade` on
   `OperationTransaction`
   (`libs/atlas-packet/interaction/serverbound/operation_transaction.go`) is
   **wrong** on v83 evidence. See §11.1.
2. The settlement handshake has a second round trip. See §6.2.

### 1.6 `TRADE_ADD_MESO` is ExclRequest-gated and absolute

`CTradingRoomDlg::PutMoney` @`0x7c37ca`:

- Gated on `!this[111] && this[52] > 1` — not locally confirmed, and the room
  has two occupants.
- Gated on `CWvsContext::CanSendExclRequest(ctx, 500, 0)`; on send it stamps
  `get_update_time()`. The gate takes a 500 (ms) argument and is stamped with a
  timestamp, so it reads as **time-based and self-clearing**; whether an
  `EnableActions` is additionally required is a per-version check at
  implementation time, per the ExclRequest/EnableActions unlock contract.
- The input dialog's maximum is the character's current meso, 10 digits.
- Client-side guard: send only if `level > 15 || amount <= 1,000,000`; otherwise
  it shows `SP_3977` locally. The daily-meso cap is therefore partly
  client-enforced, and mode 21 is its server-side twin.
- Sends `Encode1(0x10), Encode4(amount)` — **the absolute total from the input
  box**, not a delta. The clientbound echo (mode 16) is likewise an assignment.

---

## 2. Service topology and ownership

### 2.1 Decision: keep atlas-trades as a new service

The PRD's choice stands. Trade is a distinct bounded context from mini-games:
it is DB-backed (the ledger), it owns a settlement saga, and it has an economy
config surface. Folding it into atlas-mini-games would make that service both
in-memory-ephemeral and durable-economic.

**But the PRD's FR-1.2 (one mini-room per character, across kinds) cannot be
enforced inside atlas-trades**, because the other two registries live in
atlas-mini-games (in-memory) and atlas-merchant (DB). There is no shared
occupancy store, and this design does not introduce one.

**Resolution:** atlas-channel performs the cross-family occupancy check before
dispatching a trade `CREATE`/`ENTER`. It is the only component that already
holds all three views — it calls `minigame.NewProcessor(…)` and
`merchant.NewProcessor(…)` today in the `VISIT`, `CHAT` and `EXIT` arms
(`services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go:146-225`).
atlas-trades additionally enforces its own single-room invariant in its
registry (the check that matters for correctness of its own state); the
cross-family check is best-effort and races are resolved by the room the client
actually ends up in. A cross-family collision replies
`OTHER_REQUESTS` (mini-room enter error 3), as FR-1.2 requires.

This is a deliberate accepted limitation, recorded rather than hidden.

### 2.2 Component map

```
                    COMMAND_TOPIC_TRADE
 atlas-channel  ─────────────────────────────►  atlas-trades
   socket/handler/character_interaction.go        trade/registry.go   (in-memory rooms)
   trade/processor.go       (new package)         trade/processor.go  (lifecycle + rules)
   kafka/consumer/trade     (new)                 ledger/             (durable, GORM)
        ▲                                         rest/               (JSON:API)
        │  EVENT_TOPIC_TRADE_STATUS                    │
        └──────────────────────────────────────────────┤
                                                       ├─► COMMAND_TOPIC_INVITE (invite.TypeTrade)
                                                       ├─◄ EVENT_TOPIC_INVITE_STATUS
                                                       ├─► COMMAND_TOPIC_SAGA (settlement)
                                                       ├─◄ EVENT_TOPIC_SAGA_STATUS
                                                       ├─► atlas-inventory (reservations, REST/Kafka)
                                                       └─► atlas-tenants (trade-configs)
```

atlas-channel never mutates inventory or meso for trade. atlas-trades never
writes a packet. All wire encoding stays in atlas-channel, driven by
`EVENT_TOPIC_TRADE_STATUS`.

### 2.3 Room identity: two ids

The client's invite carries a `uint32` serial (`dwSN`), and
`invite.CreateCommandBody.ReferenceId` is `invite.Id` = `uint32`. A `uuid` will
not fit either. Rooms therefore carry:

- **`Id uuid.UUID`** — the REST identity (`GET /trades/rooms/{roomId}`), and the
  registry key.
- **`Handle uint32`** — the wire serial, set to **the owner's character id**.
  This matches the existing mini-room convention in atlas-channel, where the
  `VISIT` arm treats `sp.SerialNumber()` as the owner character id
  (`character_interaction.go:157`). It is unique per tenant because a character
  owns at most one room.

---

## 3. Room lifecycle

### 3.1 States

```
                 CREATE (roomType 3|6)
        ─────────────────────────────────► OPEN_SOLO
                                              │ INVITE
                                              ▼
                                        PENDING_INVITE
                        invite rejected/     │  invite accepted
                        expired ─────────┐   ▼
                                         │  OPEN  ◄── staging (put item / add meso)
                                         │   │ both sides CONFIRM
                                         │   ▼
                                         │  AWAITING_ATTESTATION   (mode 17 sent to both)
                                         │   │ both TRANSACTION replies (or timeout)
                                         │   ▼
                                         │  SETTLING   (settlement saga in flight)
                                         │   │
                                         ├───┴──► DESTROYED
```

`PENDING_INVITE` and `OPEN_SOLO` are distinct: a room whose owner has not yet
invited anyone is legal (the client opens the dialog on `CREATE`), and an
invite can be re-issued after a decline without tearing the room down. The PRD's
FR-2.5 says a decline destroys the room; this design keeps that (the reference
client closes the inviter's dialog), so `PENDING_INVITE → DESTROYED` on decline.

`AWAITING_ATTESTATION` has a **5-second deadline**. On expiry the room settles
anyway using the CRC lists from the two `TRADE_CONFIRM` (0x11) payloads, which
carry the same `{data, crc}` entries. The attestation is defence in depth, not a
liveness dependency — a client that never replies must not be able to wedge a
trade in escrow-equivalent limbo.

### 3.2 Freeze rule (FR-3.6)

From the moment the **first** side confirms, the room rejects `PUT_ITEM`,
`ADD_MESO` and any further `CONFIRM` from either side. The reference client
enforces this locally too (`PutMoney`'s `!this[111]` gate, §1.6), so a
server-side rejection here indicates a modified client — log at WARN and drop,
with no clientbound response.

### 3.3 Teardown triggers (FR-1.4, FR-6.4)

| Trigger | Source | Status sent |
|---|---|---|
| `EXIT` (mode 10) from either side | atlas-channel handler arm | `LEAVE` 2 to the other side; `LEAVE` 0 (silent) to the leaver |
| Character logout / disconnect | existing `EVENT_TOPIC_CHARACTER_STATUS` LOGOUT consumer | `LEAVE` 2 to the survivor |
| Map change | `EVENT_TOPIC_CHARACTER_STATUS` MAP_CHANGED | `LEAVE` 12 (different map) to both |
| Channel change | `EVENT_TOPIC_CHARACTER_STATUS` CHANNEL_CHANGED | `LEAVE` 12 to both |
| Settlement failure | settlement saga compensated | `LEAVE` 8 to both |
| Settlement inventory pre-check failure | atlas-trades | `LEAVE` 9 to both |
| CRC mismatch | atlas-trades attestation check | `LEAVE` 13 to both |
| Settlement success | settlement saga completed | `LEAVE` 7 to both |

atlas-mini-games already consumes character status for exactly this purpose;
atlas-trades mirrors that consumer.

> **Amended by §5A.8.** Every row in this table now unwinds escrow before the
> room is discarded — items back to their owners, escrowed meso refunded — and
> the `LEAVE` is emitted after that unwind reports terminal.

**Cancel loses to settlement (FR-6.5):** once the room enters `SETTLING`, cancel
triggers are recorded and ignored; the saga's terminal status produces the
client's `LEAVE`.

---

## 4. Packet layer

### 4.1 Enter result — reuse `interaction.Room`, no new encoder

Per §1.3 the trade room's blob is the base frame with an empty tail. Add:

```go
// NewTradeRoom builds a trade (roomType 3) or cash-trade (roomType 6)
// enter-result room. CTradingRoomDlg's enter-result tail virtual (vtable+72,
// v83 off_B37448+0x48 -> nullsub_94 @0x48314D) is EMPTY: the body is exactly
// the CMiniRoomBaseDlg::OnEnterResultBase frame — roomType, capacity(2),
// position, visitors, 0xFF. Nothing follows.
func NewTradeRoom(roomType RoomType, position byte, visitors []Visitor) Room
```

`Room.Encode`'s `switch rm.roomType` already falls through for
`TradeRoomType`/`CashTradeRoomType` writing nothing after the `0xFF`, and
`decodeVisitorForRoom`'s `default` arm already handles trade visitors
(`visitor.go:106`). The doc comment on `Room` (`room.go:52`) is updated to say
it now covers the shop **and trade** families, with the game rooms still
excluded.

`CharacterInteractionEnterResultSuccessBody` is reused unchanged.

### 4.2 Three new clientbound bodies

All live in `libs/atlas-packet/interaction/clientbound/interaction_trade.go`
(new file — `interaction_body.go` is already 24.7K and adding a fourth family
to it would be the wrong altitude), with constructors added to
`interaction_body.go` alongside the existing family constructors.

| Key | Mode (v83) | Shape | Authority |
|---|---|---|---|
| `TRADE_PUT_ITEM` | 15 | `side:byte, tradeSlot:byte, Asset` | `sub_7C1FB7` @`0x7c1fb7` |
| `TRADE_ADD_MESO` | 16 | `side:byte, amount:uint32` | `sub_7C208E` @`0x7c208e` |
| `TRADE_CONFIRM` | 17 | *(empty)* | `CTradingRoomDlg::OnTrade` @`0x7c20bc` |

and one discovered arm, new to this task:

| Key | Mode (v83) | Shape | Authority |
|---|---|---|---|
| `TRADE_MESO_LIMIT` | 21 | *(empty)* | `sub_7C21BD` @`0x7c21bd` |

`TRADE_MESO_LIMIT` is in scope because FR-4.8 requires a faithful meso-rejection
and this is the only arm the client has for it. It is bodyless, so the codec is
three lines. On versions where the arm is absent (the cash room has no mode-21
arm at all, §1.2), the rejection degrades to the authoritative re-echo described
below.

**Meso rejection (FR-4.8):** because mode 16 is an *assignment* (§1.2), the
server corrects an out-of-range stage by echoing `TRADE_ADD_MESO` with the last
valid amount — the client's view snaps back. Where `TRADE_MESO_LIMIT` exists,
it is sent as well so the player sees a reason.

Every mode byte is resolved through `atlas_packet.WithResolvedCode("operations", KEY, …)`
(DOM-25). No trade byte is hard-coded in Go.

### 4.3 New `leaveReason` keys

FR-8.5 needs no codec — only keys in the tenant `leaveReason` writer table,
sent through the existing `CharacterInteractionLeaveReasonBody`:

```go
CharacterInteractionLeaveReasonTradeCancelled   = "TRADE_CANCELLED"    // 2
CharacterInteractionLeaveReasonTradeSuccess     = "TRADE_SUCCESS"      // 7
CharacterInteractionLeaveReasonTradeFailed      = "TRADE_FAILED"       // 8
CharacterInteractionLeaveReasonTradeCannotCarry = "TRADE_CANNOT_CARRY" // 9
CharacterInteractionLeaveReasonTradeDifferentMap = "TRADE_DIFFERENT_MAP" // 12
CharacterInteractionLeaveReasonTradeCrcFailed   = "TRADE_CRC_FAILED"   // 13
```

Distinct keys from the shop and mini-game leave reasons, per the precedent in
`interaction_body.go:167-179` — the trade path never depends on another
family's numeric values.

### 4.4 Version gating

Divergent fields use the `MajorAtLeast` idiom in a new
`libs/atlas-packet/interaction/clientbound/version_gate.go`, mirroring the
existing serverbound `tradeCrcPresent`
(`libs/atlas-packet/interaction/serverbound/version_gate.go`). No raw `> N`.

Note the serverbound gate already encodes a verified boundary: the trade CRC
entry list is **absent in GMS ≤ v79, present from v83 and in JMS**. That gate is
reused, not duplicated — and it means on v48/v61/v72/v79 the `TRADE_CONFIRM`
payload is bare, so §6.2's attestation has no CRC to check there. Those versions
skip the CRC comparison and settle on confirm alone.

---

## 5. Escrow model — the core decision

### 5.1 What the PRD assumed, and why it does not hold

FR-3.2 stages items by running `ReleaseFromCharacter` at put-item time, and
NFR "Atomicity" asserts *"escrow lives in the saga/inventory layer, not in the
in-memory registry"*.

Reading the actual primitive: `release_from_character` is expanded by the
orchestrator as a **soft-delete in atlas-inventory**, paired in the same saga
with an `accept_to_*` that recreates the asset from an `AssetData` snapshot at
the destination (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:1266-1334`,
the `expandTransferToStorage` shape; MTS and cash shop follow it identically).
There is no standalone escrow bucket that names an owner. A released asset
belongs to a *saga transaction*, and only that saga's compensation can return it.

So escrow-at-staging means: N independent per-put-item sagas, each completing
with the asset soft-deleted and no destination. If atlas-trades dies, the
in-memory room dies with it (FR-1.3), and nothing knows those transaction ids.
The PRD's own crash-recovery NFR ("on startup, reconcile any assets left in
trade escrow") has no data source. It is unimplementable as specified.

### 5.2 Options considered

**Option A — reserve at staging, move at settlement (recommended).**
Staging is a logical claim in atlas-trades plus an atlas-inventory
*reservation*. The only inventory mutation is the settlement saga, which does
every release and accept as one compensatable unit.

**Option B — escrow at staging + a durable escrow table in atlas-trades.**
Keep the PRD's model but add a `trade_escrow` table recording
`(transactionId, characterId, assetSnapshot)` per staged item, and a startup
reconciler that issues `accept_to_character` for every open row.

**Option C — escrow at staging, ephemeral (the PRD as literally written).**
Rejected: orphans assets on any atlas-trades restart, with no recovery path.

### 5.3 Recommendation: Option A

| | A (reserve) | B (escrow + table) |
|---|---|---|
| Orphan class on crash | none — nothing is deleted | none, but only because of the new table + reconciler |
| Staging latency (NFR 200 ms p99) | one reservation call | a full saga round trip per put-item |
| Cancel cost | drop reservations, no inventory writes | one `accept_to_character` saga per staged item |
| Settlement atomicity | one saga, orchestrator-native compensation | one saga, plus N earlier sagas to unwind on failure |
| New durable state | none | a table + a reconciler + its own failure modes |
| Double-stage / drop / sell prevention | atlas-inventory reservation | genuine absence from inventory |
| Late failure risk | item consumed elsewhere between stage and settle | none |

A wins on every axis except the last, and that risk is bounded:

- Reservations already exist and already block the competing operations —
  `GetReservedQuantity` is consulted by move, merge and drop in
  `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:507,589-590,698,856`.
- The settlement saga revalidates server-side and, on failure, compensates and
  sends `LEAVE 8` ("Trade unsuccessful") — which is a **faithful client
  outcome**, not a broken one. The reference client has that exact string for
  this exact situation.
- FR-3.5 already says a staged item cannot be un-staged, so nothing in the UX
  depends on the item being physically gone.

**One prerequisite change in atlas-inventory:** `RequestReserve` hard-codes a
30-second TTL (`compartment/processor.go:783`), which is the drop-reservation
lifetime and far too short for a trade window. This design adds an explicit
`expiry time.Duration` parameter to `RequestReserve` / the reserve command body,
with the existing 30 s as the caller-supplied default at every current call
site, and atlas-trades supplying a longer window. atlas-trades **refreshes** the
reservations of an open room on a ticker at TTL/3 while the room is alive, and
drops them on teardown. Expiry is therefore a backstop for a dead trade room,
not a normal-path event — and if it does fire mid-trade, settlement fails
cleanly with `LEAVE 8`.

Reservation TTL is `trade.reservationTtlSeconds` in the tenant config (§8),
default 300.

### 5.4 Meso (PRD open question 5)

Meso is **logically reserved only**, never moved before settlement, and is
symmetric with items under Option A. There is no meso-reservation primitive and
this design does not add one: the staged amount is validated against a fresh
character read at stage time and again at settlement pre-check, and the
settlement saga's `AwardMesos` deduction failing is another `LEAVE 8`.

This is what the PRD proposed in FR-6.2; §5.3's item model now matches it
instead of contradicting it.

---

## 5A. Amendment — escrow at staging (supersedes §5.2–§5.4)

> **Status.** Written after the first live test of the implemented branch.
> §5.1's analysis of `release_from_character` stands and is still the reason
> §5.2 Option C was rejected. What changed is that Option A turned out to be
> **wire-incompatible with the reference client**, which §5 never checked. This
> section supersedes §5.2, §5.3 and §5.4; the superseded text is retained
> because its reasoning explains the shape of what replaces it.

### 5A.1 Why reserve-at-staging cannot work

Three facts about the GMS v83 client (`MapleStory_dump.exe.i64`), each read from
the binary, not inferred:

1. **The client does not remove the staged item locally.** `CDraggableItem::PutItem`
   @`0x4f3ef4` runs the client-side trade restrictions (sealed / cash / quest /
   trade-block / account-sharable) and then calls straight into
   `CTradingRoomDlg::PutItem`. It never touches `CharacterData`. Under §5.3
   nothing leaves the compartment either, so a staged item renders in **both**
   the inventory window and the trade window. Same for meso: `PutMoney`
   @`0x7c37ca` only sends, and the meso counter renders from `CharacterData`.

2. **Staging arms the client's exclusive-request lock.** Both
   `CTradingRoomDlg::PutItem` @`0x7c359f` and `PutMoney` @`0x7c37ca` set
   `m_bExclRequestSent = 1` and stamp `get_update_time()` immediately after
   `SendPacket`, and both refuse to run unless `CWvsContext::CanSendExclRequest`
   @`0x485bf7` passes:

   ```c
   return !this[2089]                                  // m_bExclRequestSent
       && (a3 || _ZtlSecureFuse<short>(…) > 0)          // hp > 0
       && get_update_time() - this[2090] >= a2;         // ≥ 500 ms
   ```

3. **Only a server packet clears it.** `CWvsContext::OnInventoryOperation`
   @`0xa1ead9` opens with `if (CInPacket::Decode1(iPacket)) { …[96] = 0; …[100] = get_update_time(); }`;
   `CWvsContext::OnStatChanged` carries the same leading bool; and
   `CWvsContext::OnGameStageChanged` @`0xa0400e` clears it on every `SET_FIELD`.
   There is no client-side timeout that releases it.

Under §5.3 the server mutates neither inventory nor meso at stage time, so it
emits none of those three. The lock therefore latches on the first stage and
never clears: every subsequent `PUT_ITEM` and `PUT_MONEY` is dropped **inside
the client**, before a packet is written. `CTradingRoomDlg::Trade` @`0x7c39a0`
is *not* gated, which is why Confirm keeps working and the failure presents as
"the mesos button stopped responding" rather than as a frozen dialog.

This is not a defect in the implementation of §5.3 — it is §5.3 being
unimplementable against this client without inventing a compensating unlock
packet that the reference server never sends.

### 5A.2 Decision

**Escrow at staging.** Adopt §5.2's Option B, implemented not as an ad-hoc table
but as a first-class custody destination in the accept/release family the
codebase already uses for every other custody transfer:

| Domain | accept | release | custody store |
|---|---|---|---|
| character | `accept_to_character` | `release_from_character` | atlas-inventory compartments |
| storage | `accept_to_storage` | `release_from_storage` | atlas-storage |
| cash shop | `accept_to_cash_shop` | `release_from_cash_shop` | atlas-cashshop |
| MTS | `accept_to_mts_listing` | `release_from_mts_holding` | atlas-mts `holdings` |
| **trade (new)** | **`accept_to_trade`** | **`release_from_trade`** | **atlas-trades `trade_escrow_items`** |

The decisive property is the one §5.1 said Option C lacked: a custody row
**names its owner**, so returning it never requires knowing which saga created
it.

### 5A.3 The escrow store

A new `escrow` package in atlas-trades, modelled on
`services/atlas-mts/atlas.com/mts/holding/entity.go` for the item table and on
atlas-trades' own `settlement/entity.go` for the reconciliation fields:

- **`trade_escrow_items`** — surrogate UUID PK; `(tenant_id, id)` unique;
  `room_id`, `owner_id`, `trade_slot`, `source_inventory_type`,
  `source_slot`, `template_id`, `quantity`, and the item snapshot as **explicit
  name-keyed stat columns, not a JSON blob** (COPY/restore column-order safety,
  the reason `holding` is shaped that way). Index `(tenant_id, room_id)` for the
  unwind and `(tenant_id, owner_id)` for recovery.
- **`trade_escrow_mesos`** — one row per `(room_id, owner_id)` carrying the
  debited `amount`. Separate table rather than a `kind` discriminator: the two
  have nothing in common but the room.

Both tables carry `TenantRegion` / `TenantMajor` / `TenantMinor` alongside
`TenantId`, for the reason `settlement.Entry` already documents: startup
reconciliation runs with no tenant in context and has to rebuild each row's
tenant to scope the commands that follow.

### 5A.4 Staging an item

1. atlas-trades validates §7's restrictions and the target slot — unchanged.
2. It submits a `transfer_to_trade` composite, expanded by the orchestrator into
   `release_from_character` → `accept_to_trade`, following
   `expandTransferToStorage` (`saga/processor.go:1270`) exactly. Compensation is
   orchestrator-native: a failed accept unwinds the release.
3. atlas-inventory deletes the asset and publishes; atlas-channel's asset
   consumer answers with `NewChangeBatch(false, NewRemoveEntry(…))`
   (`kafka/consumer/asset/consumer.go:428,506`). `silent=false` writes the
   leading bool as **true**, so the same packet that removes the item from the
   client's inventory clears `m_bExclRequestSent`.
4. atlas-trades' custody consumer writes the escrow row, marks the room slot
   staged and emits `ITEM_STAGED`, which becomes the mode-15 broadcast to both
   sides — unchanged from §4.2.

**No new clientbound code is required for the unlock.** It is a consequence of
the mutation being real. This is the single strongest argument for the
amendment: §5.3 needed a packet the reference server does not send; §5A needs
none.

### 5A.5 Staging meso

Mode 16 is an **assignment**, not an accumulation (§1.6), so the server works in
deltas against what is already escrowed:

```
delta = requested_total − escrowed_total
delta > 0 → award_mesos(owner, −delta), upsert the escrow meso row
delta < 0 → award_mesos(owner, +(−delta)), decrement or delete the row
delta = 0 → re-echo only
```

Validation against a fresh character read is unchanged (§7, FR-4.8), and so is
the refusal response (authoritative re-echo plus `TRADE_MESO_LIMIT` where the
version has the arm). `atlas-character`'s `statChangedProvider`
(`character/producer.go:238`) already hard-sets `ExclRequestSent: true` and the
meso paths already publish `stat.TypeMeso` (`character/processor.go:851,877,906`),
so — exactly as with items — the real debit both drops the client's meso counter
and clears the lock.

### 5A.6 Every refusal must still unlock

§7 specifies a **silent drop** for FR-4.1–4.4: no clientbound update, the empty
slot is the feedback. With the lock armed on send, a silent drop now wedges the
dialog permanently. The same applies to §3.2's freeze rule, FR-4.8's meso
refusal, and any stage whose saga fails.

Every refusal path therefore emits, to the acting character only, a
`STAT_CHANGED` carrying `exclRequestSent = true` and an **empty** update list —
an unlock with no stat payload. The slot still stays empty, so the player-visible
feedback is unchanged; only the lock is released.

This is written by **atlas-channel**, not atlas-trades: §2.2's rule that
atlas-trades never writes a packet is preserved. atlas-channel's trade status
consumer gains the emission, reusing the shape of
`socket/handler/enable_actions.go`. atlas-trades gains one new status type,
`ITEM_REFUSED`, symmetric with the existing `MESO_REFUSED`; the consumer's
`MESO_REFUSED` arm gains the same unlock.

### 5A.7 Settlement (supersedes §6.3's step list)

The debit already happened at stage time, so settlement no longer releases from
characters and no longer issues negative `award_mesos`:

```
for each escrowed item of A:  release_from_trade(escrowId)
for each escrowed item of B:  release_from_trade(escrowId)
for each escrowed item of A:  accept_to_character(B, snapshot)
for each escrowed item of B:  accept_to_character(A, snapshot)
if A.meso > 0:                award_mesos(B, +A.meso − tax(A.meso))
if B.meso > 0:                award_mesos(A, +B.meso − tax(B.meso))
```

§6.3's ordering rule survives verbatim — all releases precede all accepts, so a
slot freed by an outgoing item is available to an incoming one. §6.5's tax
semantics survive too: the difference is destroyed by crediting the receiver
less than was escrowed, rather than by an asymmetric pair of `award_mesos`.

`trade_settlement` stays a composite and atlas-trades still enumerates no
concrete steps; only the expander's output changes.

§6.1's pre-check 3 ("every reservation is still live and matches the staged
quantity") becomes "every escrow row is still present and matches". The failure
class it guarded against — the staged item consumed elsewhere between stage and
settle, the one axis on which §5.3's table conceded to Option B — cannot occur
once the asset is physically in escrow.

### 5A.8 Teardown, cancel and the return path

Every trigger in §3.3 now unwinds escrow before the room is discarded: for each
item row `release_from_trade` → `accept_to_character(owner_id)`, and for each
meso row `award_mesos(owner_id, +amount)`. The unwind is idempotent and
compare-and-set on the room id, matching the teardown discipline already in
place.

`LEAVE` is emitted after the unwind reports terminal, for the reason §6.4 gives
for `SETTLED`: the client renders its own numbers from its own character data.

**A return can legitimately fail.** If the owner's inventory filled while the
trade was open, `accept_to_character` fails and the saga compensates, leaving
the row in escrow. Nothing is lost — the row is durable and still names its
owner — but the process is alive, so startup recovery will not catch it. A
retry ticker re-attempts every escrow row whose room no longer exists, at the
same cadence the reservation-refresh ticker used to run. This ticker replaces
that one rather than adding to the service's moving parts.

### 5A.9 Crash recovery — now implementable

§5.1's objection was that a released asset belongs to a saga transaction with no
recoverable owner. An escrow row names its owner, so recovery is mechanical:

On startup, after the existing settlement reconciler has drained
`settlement.Entry` (a settlement still in flight may legitimately consume escrow
rows, so this ordering is mandatory), read every remaining escrow row across
every tenant and return it. Rooms are process-local (§9) and die with the
process, so **every surviving row is orphaned by definition** — there is no
room-reconstruction step and no partial-state reasoning. The PRD's crash-recovery
NFR, which §5.1 correctly called unimplementable as specified, is satisfied.

### 5A.10 What this removes

- `services/atlas-trades/atlas.com/trades/compartment/` — the reserve/cancel
  command producer.
- The reservation-refresh ticker (§5.3's TTL/3 refresh), replaced by §5A.8's
  stuck-escrow retry ticker.
- `trade.reservationTtlSeconds` from the tenant config (§8).

Task 7's `expiry` parameter on `RequestReserve` **stays**. It landed repo-wide
with the existing 30 s supplied at every prior call site, it is a correct
generalisation independent of trade, and reverting it would be pure churn.

FR-3.5 (a staged item cannot be un-staged) is unaffected and, if anything,
better justified: the item is genuinely gone.

### 5A.11 The cost, stated plainly

Staging becomes an asynchronous saga round trip instead of one reservation call,
which the PRD's 200 ms p99 staging NFR did not anticipate. The mitigation is
structural rather than a workaround: `m_bExclRequestSent` **is** the client's
own in-flight indicator, and it clears exactly when the resulting
`INVENTORY_OPERATION` or `STAT_CHANGED` lands. A slow stage therefore delays the
unlock instead of corrupting state, which is the client's native semantics for
every other exclusive request (item use, skill use, portal).

The NFR is restated as: **the unlock must reach the client within 1 s p99**, and
a stage that has not resolved within `trade.stageTimeoutSeconds` (new config
key, default 5) emits the bare unlock of §5A.6 and drops the stage. Without that
bound, a saga that never terminates locks the dialog for the rest of the
session.

### 5A.12 Risks introduced

| Risk | Assessment |
|---|---|
| Double unlock (real packet + bare unlock race) | Harmless. Clearing an already-clear flag is a no-op; the 500 ms floor is unaffected. |
| Return fails on a full inventory | Covered by §5A.8's retry ticker. Durable row, named owner, no loss. |
| Escrow row leaks if the retry ticker and startup recovery both miss it | The row is visible in the REST surface (§9) and is a bounded, auditable leak, not a lost asset. |
| Item genuinely gone while the trade is open | Intended, and what the client already renders. Any external process that reads a character's inventory mid-trade sees the true state. |

---

## 6. Settlement

### 6.1 Pre-checks (FR-4.9)

Run in atlas-trades, before any saga is submitted, against fresh reads:

1. Each side has free slots in every compartment its incoming items need
   (counting stackable merges).
2. Each side's meso, after `+incoming_post_tax − outgoing`, is within
   `[0, mesoCap]`.
3. Every reservation is still live and matches the staged quantity.
   *(§5A.7: now "every escrow row is still present and matches".)*
4. Where the version carries CRC entries (§4.4), each attested
   `{itemId, crc}` matches the staged assets.

Failure maps to a `LEAVE` status: (1) → 9, (2) → 9, (3) → 8, (4) → 13.

The PRD's FR-4.9 says a settlement refusal *"reverts nothing (the room is still
live)"*. That contradicts the client: `CTradingRoomDlg::OnLeave` (§1.4) closes
the dialog before showing any of these notices — there is no client state in
which the room survives a status 8/9/13. **This design tears the room down on
settlement refusal**, which is both faithful and simpler. Under Option A nothing
needs reverting anyway.

### 6.2 The attestation round trip

```
A: TRADE_CONFIRM (0x11) [+CRC list on v83+]
B: TRADE_CONFIRM (0x11) [+CRC list on v83+]
      → room enters AWAITING_ATTESTATION
server → A: clientbound TRADE_CONFIRM (mode 17, empty)
server → B: clientbound TRADE_CONFIRM (mode 17, empty)
A: TRANSACTION (0x14) [+CRC list]     (client sends this automatically)
B: TRANSACTION (0x14) [+CRC list]
      → room enters SETTLING
```

Mode 17 is broadcast **only after both sides have confirmed**, never on the
first confirm. Sending it on the first confirm would make the counterparty's
client auto-reply `0x14` without its owner ever pressing Trade (§1.2), which
would let one side drive the other's attestation. The 5-second deadline (§3.1)
covers a client that does not reply.

> **Correction (post-implementation, verified on the GMS v83 client).** §1.5
> describes both messages as carrying "the client's own CRC list". They do not
> carry the *same* list, and the check must not compare them to each other:
>
> - `CTradingRoomDlg::Trade` @`0x7c39a0` (0x11) walks **both** dialog arrays —
>   member 113 (own staged items) and member 114 (the counterparty's) — so the
>   confirm carries **every item in the window**, interleaved per trade slot.
> - `CTradingRoomDlg::OnTrade` @`0x7c20bc` (0x14) walks **member 114 only**, so
>   the attestation carries **just the counterparty's items** — what this
>   character is about to receive.
>
> Comparing the two lists refuses every trade in which both sides stage an item
> (confirm has n+m pairs, attestation m), which the client reports as
> `SP_5566` "the game file has been damaged". A one-sided trade hid it: the
> giver's attestation is empty and the receiver's window holds a single item, so
> the lists coincidentally matched. §6.1 check 4 therefore validates the
> attestation against the **counterparty's staged contribution** — template-id
> multiset — plus containment in this side's own confirm list, which is what
> preserves the CRC-tamper detection. See `attestationMatches` in
> `trade/settlement.go`.

### 6.3 Saga shape

> **Superseded by §5A.7.** The step list below assumes the items are still in
> their owners' compartments at settlement time. Under escrow-at-staging they
> are not: releases come from escrow, and the meso debit has already happened.
> The ordering rule and the tax semantics are unchanged.

One saga per settlement, submitted with a single `transactionId` that is also
the ledger's idempotency key (FR-5.7). Steps, in order:

```
for each staged item of A:  release_from_character(A, assetId, qty)
for each staged item of B:  release_from_character(B, assetId, qty)
for each staged item of A:  accept_to_character(B, snapshot)
for each staged item of B:  accept_to_character(A, snapshot)
if A.meso > 0:              award_mesos(A, −A.meso)
                            award_mesos(B, +A.meso − tax(A.meso))
if B.meso > 0:              award_mesos(B, −B.meso)
                            award_mesos(A, +B.meso − tax(B.meso))
```

All releases precede all accepts so that a slot freed by an outgoing item is
available to an incoming one, and so a failure in either side's release
compensates before anything has been created.

**A new composite action `trade_settlement` is added** to `libs/atlas-saga`,
expanded by the orchestrator into the concrete steps above — the pattern of
`expandTransferToStorage` (`saga/processor.go:1266`). The composite carries the
two participants, their staged item references and meso, and the resolved tax
amounts; expansion performs the inventory lookups and builds the `AssetData`
snapshots exactly as the existing expanders do. atlas-trades never enumerates
concrete saga steps itself.

The tax is computed **in atlas-trades** (it needs the tenant config) and passed
into the composite as resolved integers, so the orchestrator stays
config-free.

### 6.4 Ordering with the completion packet

Per §1.4, the client's "Received %d mesos after fees" line is rendered from its
*own* character data. atlas-trades therefore emits `SETTLED` (which becomes
`LEAVE 7`) only after the saga reports terminal success — by which time
atlas-character has published the meso change and atlas-channel has forwarded
the stat update. If the two race, the player sees `SP_407` (the no-figure
variant) instead of `SP_408`. That is a cosmetic degradation, not a correctness
bug, and is accepted rather than solved with an artificial delay.

### 6.5 Tax (FR-5.4, FR-5.5)

`delivered = m − floor(m × rate(m))`. The difference is destroyed: the negative
`award_mesos` on the giver is the full `m`, the positive on the receiver is
`delivered`. No third party is credited.

Defaults ship as a descending tier table; validation (FR-9.3) enforces strictly
descending thresholds and rates in `[0, 1]`, rejecting an invalid table loudly
and falling back to the shipped defaults.

---

## 7. Restrictions (FR-4)

Evaluated in atlas-trades at stage time, each mapping to a specific client
response:

| Rule | Check | Response |
|---|---|---|
| FR-4.1 untradeable flags | `asset.FlagUntradeable` (0x08), `asset.FlagMergeUntradeable` (0x200) — `libs/atlas-constants/asset/flag.go` | drop the stage, no clientbound update; the client's slot stays empty |
| FR-4.2 WZ `tradeBlock` | atlas-data item lookup | as above |
| FR-4.3 quest items | QUEST compartment | as above |
| FR-4.4 equipped | source compartment is EQUIPPED | as above |
| FR-4.5 min level | tenant config, default 0 | `ENTER_RESULT` error `UNABLE` (6) at create/accept |
| FR-4.6 map disallows trade | atlas-maps field data | `TRADE_NOT_ALLOWED` (7) / `TRADE_NOT_ALLOWED_2` (20), config-resolved per version |
| FR-4.7 dead (HP 0) | atlas-character | `NOT_WHEN_DEAD` (4) |
| FR-4.8 meso out of range | fresh character read | authoritative `TRADE_ADD_MESO` re-echo + `TRADE_MESO_LIMIT` where present (§4.2) |

**FR-4.2 sub-task:** `tradeBlock` is currently surfaced only by the consumable
and setup readers (`services/atlas-data/atlas.com/data/consumable/reader.go:49`,
`setup/reader.go:47`). The equip, etc and cash readers gain the same field. The
PRD is explicit that a missing flag must not be read as "tradeable" — so
atlas-trades treats an atlas-data lookup **failure** (not a `false` value) as a
refusal, and logs it at ERROR.

A silent drop for FR-4.1..4.4 is correct, not a violation of "never a silent
drop": the reference client has no mini-room error for "this item cannot be
traded" at put-item time, and the empty slot is the feedback. The rejection is
logged server-side with the item id and the failing rule.

> **Amended by §5A.6.** "Silent" now means *no visible response*, not *no
> packet*. Because the client arms `m_bExclRequestSent` on send, every refusal
> path must still return the bare unlock (`STAT_CHANGED`, `exclRequestSent =
> true`, empty update list) or the dialog wedges. The empty slot remains the
> only player-visible feedback.

---

## 8. Tenant configuration (FR-9)

A `trade-configs` resource in atlas-tenants, following the `mts-configs`
precedent (`services/atlas-tenants/atlas.com/tenants/configuration/resource.go:817-1026`
— GET / GET-by-id / POST / PATCH / DELETE / seed):

```json
{
  "taxEnabled": true,
  "taxTiers": [
    { "threshold": 100000000, "rate": 0.060 },
    { "threshold":  25000000, "rate": 0.050 },
    { "threshold":  10000000, "rate": 0.040 },
    { "threshold":   5000000, "rate": 0.030 },
    { "threshold":   1000000, "rate": 0.018 },
    { "threshold":    100000, "rate": 0.008 }
  ],
  "maxStagedItems": 9,
  "minTradeLevel": 0,
  "reservationTtlSeconds": 300,
  "attestationTimeoutSeconds": 5
}
```

> **Amended by §5A.10 / §5A.11.** `reservationTtlSeconds` is removed — nothing
> is reserved. `stageTimeoutSeconds` (default 5) is added: a stage whose escrow
> saga has not resolved within it is dropped with the bare unlock of §5A.6.

Absent config → shipped defaults, logged once at INFO (FR-9.2). Never a crash,
never a silent disable.

Two known traps to avoid, from prior tasks:
a seed endpoint that is never invoked (a seed endpoint that exists but is
never called leaves live tenants without the resource — the fallback path in
FR-9.2 must therefore be the *tested* path, not the exceptional one).

---

## 9. Ledger and REST

The PRD's three-table model (`trade_ledger_entries` / `_sides` / `_items`) is
adopted unchanged, with two additions:

- `trade_ledger_entries` gains a unique index on `(tenant_id, transaction_id)`,
  which the PRD names in prose; it is the write-side idempotency guard for
  FR-5.7. A duplicate settle attempts the insert, hits the constraint, and
  returns success without re-emitting.
- `room_type` stays on `trade_ledger_entries` only and is deliberately not
  denormalised onto `trade_ledger_sides`: both sides of a trade always share it.

REST surface is exactly as the PRD specifies (`GET /trades/rooms`,
`GET /trades/rooms/{id}`, `GET /trades/ledger`, `GET /trades/ledger/{id}`), JSON:API
via api2go, tenant-scoped, no write endpoints. Page size capped at 100.

`GET /trades/rooms` reads the in-memory registry, so it is per-pod. atlas-trades
runs single-replica for the same reason atlas-mini-games does — the room
registry is process-local. This is a **scaling constraint that must be stated in
the k8s manifest** (`replicas: 1`, and no HPA), not discovered later.

---

## 10. Per-version work and the remaining open questions

### 10.1 Procedure

Per-version mode bytes, arm presence, and body layouts are **not** assumed from
v83. Each op × version cell is derived and promoted through the project's
existing leaf procedure: `/verify-packet` +
[`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md),
driven for the new codecs by `/implement-packet` +
[`docs/packets/IMPLEMENTING_A_PACKET.md`](../../packets/IMPLEMENTING_A_PACKET.md).

For each of the ten versions the derivation is mechanical:

1. Locate the trade dialog's ctor (search `TradingRoom` / the
   `UI/UIWindow.img/TradingRoom` StringPool id), read its vtable pointer.
2. `vtable+72` → enter-result tail. If it is a nullsub, the version matches
   §1.3 and `NewTradeRoom` needs no version gate.
3. `vtable+76` → `OnLeave`; enumerate the status-byte branches to map the
   `leaveReason` keys for that version.
4. Find the mode dispatcher (the switch reached from
   `CMiniRoomBaseDlg::OnPacketBase`) and enumerate its cases — this yields the
   clientbound mode bytes for `TRADE_PUT_ITEM` / `TRADE_ADD_MESO` /
   `TRADE_CONFIRM` and whether the mode-21 arm exists.
5. Absence of an arm is asserted **only** from a decompiled dispatcher that
   lacks the case — never from an unnamed symbol
   (an unnamed IDB symbol is not evidence of absence).

### 10.2 PRD open question 3 — `TRANSACTION` on legacy versions

`TRANSACTION` is present in the gms_v83/84/87/95 templates and absent from
gms_v48/61/72/79 and jms_v185 (verified against the seed templates).

§1.5 explains *why* it exists — it is the CRC attestation reply — and §4.4 notes
the serverbound gate already places the CRC boundary between v79 and v83. Those
two facts predict that `TRANSACTION` **does not exist below v83**, because there
is no CRC to attest. That is a hypothesis, not a finding: it is confirmed or
refuted by step 4 above, per version. Where confirmed, the cell is marked ⬜ n-a
with the dispatcher evidence, and those versions settle on the pair of
`TRADE_CONFIRM` (0x11) messages alone — `AWAITING_ATTESTATION` is skipped
entirely.

jms_v185 is the exception to the prediction (the gate says JMS *has* CRC) and
must be checked directly.

### 10.3 PRD open question 4 — cash trade room on legacy versions

`CASH_TRADE_OPEN` is present in the gms_v79..v95 templates and absent from
gms_v48/61/72 and jms_v185. Resolved by the same procedure: search each IDB for
a second trading-room class with its own dispatcher (v83's
`CCashTradingRoomDlg::OnPacket` @`0x4833b4` is the shape to match). Absent
dispatcher → ⬜ n-a for every cash-trade cell on that version, and the
`CASH_TRADE_OPEN` handler key is *not* added there.

### 10.4 PRD open question 6 — the gms_92 template gaps

Confirmed: `template_gms_92_1.json` contains none of `TRADE_PUT_ITEM`,
`TRANSACTION`, `CASH_TRADE_OPEN` or `TRADE_NOT_ALLOWED` — the only v48+ template
in that state.

**Decision: fix the trade keys here; record the non-trade gaps as a finding
(§11.3) and do not touch them.** The non-trade keys (`CHAT`,
`PERSONAL_STORE_ITEM_SOLD`, `MEMORY_GAME_PUT_STONE_ERROR`, both
`MERCHANT_VIEW_*`, and the merchant/personal-store handler keys) belong to
families this task does not own and cannot verify; adding them blind would be
exactly the "new opcodes missing from live config" class of bug in reverse.

### 10.5 PRD open question 7 — ledger retention

No retention policy. Accepted as-is at current scale; the table carries a
`settled_at` index so a retention job is a later, additive change.

### 10.6 Template routing (FR-11)

Ten templates gain the writer `operations` keys from §4.2 and the
`leaveReason` keys from §4.3, plus the missing handler keys (`TRANSACTION`
where it exists, `CASH_TRADE_OPEN` where the class exists, the v92 trade
handler keys). Every edit keeps `handlers` and `writers` in strictly ascending
`opCode` order and introduces no duplicate `(implementation, opCode)` binding —
`tools/template-opcode-order-guard.sh` and
`tools/template-duplicate-binding-guard.sh` gate this.

Note there is an **eleventh** template, `template_gms_12_1.json`, which the
PRD's version set does not mention and which has no interaction keys at all. It
is out of scope; recorded in §11.4.

---

## 11. Findings recorded during design

### 11.1 `OperationTransaction` has a wrong `packet-audit:fname`

`libs/atlas-packet/interaction/serverbound/operation_transaction.go` carries
`// packet-audit:fname CCashTradingRoomDlg::Trade`. On v83,
`CCashTradingRoomDlg::Trade` @`0x485dcd` sends `Encode1(0x11)` — that is
`TRADE_CONFIRM`, not `TRANSACTION`. The only `0x14` sender is
`CTradingRoomDlg::OnTrade` @`0x7c20bc`.

Fixing it is in scope (it is this task's family), but note that editing an
`fname` **stales every matrix cell that references it**
(an `fname` edit re-keys the evidence records) — the fix must be made
together with a matrix regeneration and re-pin, not as a drive-by comment edit.

### 11.2 Clientbound mode 21 was unmodelled

Neither the PRD nor `interaction_body.go` knows about the `TRADE_MESO_LIMIT`
arm. Added in §4.2.

### 11.3 gms_92 template is missing non-trade interaction keys

Recorded, not fixed (§10.4). Warrants its own template task covering the
merchant / personal-store / mini-game keys with per-family verification.

### 11.4 `template_gms_12_1.json` is outside the PRD's version set

24.6K versus 77K–190K for the others, and no interaction keys. Not touched.

### 11.5 `RequestReserve`'s TTL is hard-coded

`services/atlas-inventory/atlas.com/inventory/compartment/processor.go:783`
pins 30 s for every caller. Parameterised as part of §5.3. Retained by §5A.10
even though trade no longer reserves.

### 11.6 The trade window's report button is inert in the client

Reported during testing alongside the §5A.1 lock defect, and **not a server
bug**. `CTradingRoomDlg::OnCreate` @`0x7c23d0` creates `UI/Basic.img/BtClaim`
as control id **1005**. The dialog's only button-notify sink is `sub_7C2A71`
@`0x7c2a71` (its sole xref is the vtable slot at `0xb37468`), whose switch
handles 1002 (Trade), 1003 (PutMoney), 1004 (chat send), 1 and 2 — there is no
`case 1005`. Scanning `0x7c1c76`–`0x7c3d60`, the immediate `0x3ED` occurs
exactly once: the `push 3EDh` in `OnCreate`. No server change can affect it.
Out of scope for task-205.

---

## 12. Cross-cutting requirements

**Concurrency.** All room mutation is serialised per room. The registry follows
the atlas-mini-games shape exactly
(`services/atlas-mini-games/atlas.com/mini-games/game/registry.go`): a `sync.Once`
singleton, one `sync.RWMutex` guarding both a `map[tenant.Model]map[uuid.UUID]Room`
and a `map[tenant.Model]map[uint32]uuid.UUID` member index, with the index
maintained only inside Create/Update/Remove under the write lock. Room
transitions are compare-and-set on the room's state field so two simultaneous
confirms cannot both trigger settlement. The known
parallel-`ForEachInMap` hazard applies to
atlas-channel's broadcast callbacks, not to the registry.

**Multi-tenancy.** `tenant.MustFromContext(ctx)` everywhere; every registry map
is tenant-keyed; every ledger query is `tenant_id`-scoped.

**Goroutines.** The reservation-refresh ticker and any timers go through
`routine.Go` (`tools/goroutine-guard.sh`).

**Observability.** Structured log at every state transition carrying tenant,
room id, both character ids, transaction id. Metrics: `trade_rooms_opened`,
`trade_settled`, `trade_cancelled`, `trade_settlement_failed{reason}`,
`trade_meso_taxed_total`, `trade_reservation_expired`.

**Crash recovery.** The registry starts empty. Live reservations expire on their
own TTL — the only residue a crash can leave from a room that had not yet
settled, and it self-heals. A client whose room vanished has its next
interaction rejected with a room-closed error. The PRD's "reconcile escrowed
assets at startup" requirement is **satisfied by construction** under Option A:
there are no escrowed assets to reconcile. The FR-7 acceptance criterion
"restarting atlas-trades mid-trade recovers every escrowed asset to its owner"
is met trivially — nothing left the owner.

A room in `SETTLING` is the **exception**, and it is not covered by any of the
above. Its saga lives in atlas-saga-orchestrator and keeps running across an
atlas-trades restart, so the swap executes whether or not this service is alive
to hear about it. Losing the room would therefore lose a trade that HAPPENED —
no ledger row and no `LEAVE 7`, contradicting FR-7.1's unconditional "every
settled trade writes one durable ledger row".

atlas-trades therefore keeps a **durable settlement record**
(`trade_settlements` and its two child tables), written in the SAME transaction
that enqueues the saga command so the two cannot diverge, and deleted once the
terminal status has been handled so unfinished settlements never accumulate. It
carries everything the terminal path needs without a room: both participants,
their staged items at their re-resolved slots and reservation ids, the frozen
tax split, and the room identity the status event's envelope requires.

Two paths consume it, and both are idempotent:

- the **live** `EVENT_TOPIC_SAGA_STATUS` consumer, which now resolves the trade
  from the record rather than from the in-memory room — so a terminal event
  redelivered after a restart still settles;
- **startup reconciliation**, which asks the orchestrator
  `GET /sagas/{transactionId}` for every unfinished record. That question is
  answerable after a restart because the orchestrator's terminal states are
  durable: completion is a soft delete to `status='completed'`
  (`saga/store.go:203-212`), failure sets `status='failed'` (`:252-258`), both
  are preserved against any later write (`:127-131`), `GetById` applies no
  status filter (`:73-76`), and nothing purges saga rows.

Whoever deletes the record owns the outcome, so exactly one of the two emits.
An outcome the orchestrator cannot answer for — unreachable, or a saga it has
not consumed yet — leaves the record untouched for the next boot or for the
live event; it is **never** read as failure, because a trade that may have
executed must not be reported to the players as unsuccessful.

**Service registration.** atlas-trades follows
[`docs/adding-a-new-service.md`](../../adding-a-new-service.md) in full, with
`tools/service-registration-guard.sh` green. `replicas: 1` per §9.

---

## 13. Testing strategy

- **Codec tests.** One byte-fixture test per new clientbound body per applicable
  version, each carrying a `packet-audit:verify` marker and a pinned evidence
  record. Round-trip-only fixtures do not count as verification
  (a round-trip fixture proves the codec is self-consistent, not client-correct).
- **Registry tests.** Table-driven over the state machine in §3.1, including the
  freeze rule, both cancel-vs-settle orderings, and the attestation timeout.
- **Tax tests.** The PRD's worked example (10,000,000 → 9,600,000 at 4%,
  400,000 destroyed), each tier boundary from both sides, tax-disabled, and an
  invalid tier table falling back to defaults.
- **Settlement tests.** Saga composite expansion, each pre-check failure mapping
  to its `LEAVE` status, and idempotency of a replayed `TRANSACTION`.
- **Builders, not helpers.** Test setup uses the project Builder pattern; no
  `*_testhelpers.go`.

Full gate list per the project CLAUDE.md build-and-verification section runs
before the PR, including `docker buildx bake atlas-trades` and the template
guards.

---

## 14. Phasing

The work decomposes into five independently reviewable slices; the plan phase
should sequence them this way:

1. **Packet layer** — `NewTradeRoom`, three new clientbound bodies + the
   mode-21 arm, version gates, the `fname` correction (§11.1), fixtures, matrix
   promotion. Independent of every service change.
2. **atlas-inventory reservation TTL** (§11.5) and **atlas-data `tradeBlock`**
   (§7). Two small, isolated changes that everything downstream depends on.
3. **atlas-trades skeleton** — service registration, registry, REST, ledger,
   tenant config. No behaviour yet.
4. **atlas-saga `trade_settlement` composite** + orchestrator expansion.
5. **atlas-channel wiring** — replace the eight `l.Debugf` arms, the trade
   processor package, the status consumer, and the templates.

The cash trade room (FR-10) rides slice 5, gated on §10.3's per-version
resolution.

---

## 15. Risks

| Risk | Mitigation |
|---|---|
| A staged item is consumed elsewhere between stage and settle | Reservations block the competing paths; settlement revalidates and fails to `LEAVE 8`, a faithful outcome |
| Reservation TTL parameterisation regresses drop reservations | Every existing call site passes the current 30 s explicitly; no behaviour change outside trade |
| Legacy versions diverge more than v83 predicts | §10.1's procedure derives each version independently; no v83 layout is assumed anywhere |
| Cross-family room occupancy race (§2.1) | Accepted; best-effort check in atlas-channel, authoritative single-room check inside each service |
| Single-replica atlas-trades is an availability point | Same posture as atlas-mini-games. A restart cancels cleanly in every state EXCEPT `SETTLING`, whose saga keeps running in the orchestrator: those are covered by the durable settlement record and startup reconciliation (§12), which complete the trade — ledger row and `LEAVE 7`/`LEAVE 8` — after the room is gone |
| `SP_408` shows no figure if the meso update races `LEAVE 7` | Cosmetic; §6.4 |
