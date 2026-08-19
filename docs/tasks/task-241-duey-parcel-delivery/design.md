# Duey Parcel Delivery — Design

Task: `task-241-duey-parcel-delivery`
Status: Draft for review
Created: 2026-08-19
Input: `docs/tasks/task-241-duey-parcel-delivery/prd.md` (approved)

---

## 0. What changed since the PRD

The PRD was written from Cosmic as the behavioral reference and explicitly deferred
every wire fact to IDB derivation. That derivation was done during this design phase
against the checked-in IDBs (GMS v83 `MapleStory_dump.exe.i64`, GMS v72
`GMS_v72.1_U_DEVM.exe.i64`). Eight PRD statements are now either confirmed by the
client or corrected by it, and three PRD open questions are closed outright. The
findings are load-bearing for the data model and the saga shape, so they lead.

| # | PRD said | Client says | Where |
|---|---|---|---|
| A | Fee tiers from Cosmic `Trade.getFee` | **Confirmed exactly**, and the client computes the same table itself for the confirm dialog | v72 `sub_6590A1` @0x6590A1; v83 `sub_6EEDFE` @0x6EEDFE |
| B | Quick send replaces the 5,000 surcharge | **Confirmed directly**: NPC arm formats `fee + 5000`, quick arm formats `fee` | v83 `sub_6F36A8` @0x6F36A8 vs `sub_6F1DF5` @0x6F1DF5 |
| C | A send carries an optional message | **Corrected**: the message exists ONLY on the quick-delivery arm. The NPC send arm encodes no message at all | v83 @0x6F36A8 (no `EncodeStr` after the quick byte) vs @0x6F1DF5 (`EncodeStr` after it) |
| D | Cosmic's 21-value action enum, with gaps, as a hypothesis | **Replaced** with the derived v83 arm set (§5.2). Cosmic misidentifies the shape of 0x17/0x18/0x19/0x1B and omits 0x1A/0x1C | v83 `CParcelDlg::OnPacket` @0x6F56EA, `CParcelDlg::NoticeResult` @0x6F5BE2 |
| E | `DUEY_ACTION` may genuinely not exist on v72/v79 (OQ-2) | **It exists on v72 at opcode 0x040.** The registry gap is a CSV-import gap, not a client gap | v72 `CTabReceive::ReceiveParcel` @0x65AF41 builds `COutPacket(64)` |
| F | Meso limit is an unknown cap (OQ-5) | **It is a level gate**: characters at level ≤ 15 may send at most 1,000,000 meso. Separately the input dialog hard-caps a single parcel at 100,000,000 | v83 `sub_6F3875` @0x6F3875 (`_ZtlSecureFuse<unsigned char>(lvl) > 0xF \|\| meso <= 1000000`), SP_3977 |
| G | Mailbox size unconstrained | **The client's receive tab holds at most 10 parcels** (`idx <= 9` bound on both the receive and discard paths). Mode 0x0E "the receiver's package storage is full" is what enforces it | v83 `CTabReceive::ReceiveParcel` @0x6F0CA3, `DiscardParcel` @0x6F0DC3 |
| H | 30-day expiry is a server policy (Cosmic) | The **client independently enforces a 30-day window** on receive and refuses outside it with SP_3864. Server expiry must therefore be ≤ 30 days or parcels become unclaimable before the server retires them | v72 @0x65AF41 (`(parcelTime - now) / 864000000000 < 30`) |

Nothing here changes the PRD's goals. C, F, G and the custody correction in §4.2 change
what gets built.

**Open questions closed by this design:** OQ-1 (§6.1), OQ-2 (§5.4), OQ-3 (§5.2),
OQ-4 (§8.3), OQ-5 (§6.4), OQ-6 (§9.3), OQ-7 (§7.4), OQ-8 (§7.1).

---

## 1. Architecture at a glance

```
  NPC 9010009 script            Quick Delivery Ticket 5330000
  (atlas-npc-conversations)     (atlas-channel USE_CASH_ITEM)
          |                                  |
          |  open_duey op                    |  handleDueyCouponUse
          v                                  v
  saga: ShowParcel  --->  atlas-saga-orchestrator  ---> Kafka SHOW_PARCEL
          |                                                    |
          |                                                    v
          |                                        atlas-channel consumer
          |                                        announces PARCEL[OPEN]
          |                                                    ^
          |                        GET /parcels (read-only) ---+
          v
  +------------------------------------------------------------------+
  | atlas-parcel  (new service)                                       |
  |   parcels table  |  expiry+return task  |  notification task      |
  |   custody consumer: accept_to_parcel / release_from_parcel        |
  |   REST: /parcels, /characters/{id}/parcel-status                  |
  +------------------------------------------------------------------+
          ^                              ^                    ^
          | saga custody steps           | gate 12 lookup     | status events
          |                              |                    |
  atlas-saga-orchestrator        atlas-character        atlas-channel
   ParcelSend / ParcelReceive     pending_change         PARCEL alarms
```

Three seams, each with an existing precedent in the repo:

1. **Item custody** rides the established four-action custody protocol
   (`release_from_character` / `accept_to_X` / `release_from_X` / `accept_to_character`)
   that storage, trade, cash shop and MTS already use — not raw destroy/award. §4.2.
2. **The NPC entry point** rides the `open_storage` → `ShowStorage` → `SHOW_STORAGE`
   command chain that Fredrick already uses
   (`services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:2123`).
3. **Notification** rides the frederick notification-task → Kafka status event →
   `atlas-channel` consumer → `session.Announce` chain
   (`services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go`,
   `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go:454`).

---

## 2. Why a new service

Three options were considered.

**A. New `atlas-parcel` service (chosen).** Duey owns durable custody of assets and
meso belonging to two characters at once, a 30-day lifecycle task, a notification
task, and a synchronous eligibility endpoint that `atlas-character` polls on the
world-transfer path. That is the same weight of thing as `atlas-mts`, which is its
own service for the same reasons. It also keeps the world-transfer gate's dependency
graph honest: gate 12 needs a service that can be down and fail closed, and a narrow
service is a narrow blast radius.

**B. Fold into `atlas-merchant` beside `frederick`.** Tempting because the entity
shape is nearly identical. Rejected: `atlas-merchant` is already the hired-merchant
service, its Kafka status-event topic is merchant-scoped, and gate 10
(`merchant_open`) already routes there — a `merchant_open`/`parcel_pending`
distinction inside one service's REST surface invites exactly the kind of confusion
the gate table's per-reason precision exists to avoid.

**C. Fold into `atlas-storage`.** Rejected: storage is account-scoped, single-owner,
and has no delivery semantics. Duey is character-to-character with an expiry.

Consequence: the full `docs/adding-a-new-service.md` checklist applies — build/CI
target, k8s base, both overlays, ingress (it is a REST service), database, GHCR
package visibility. This is a real cost and it is the largest single block of
non-feature work in the task; §11 sequences it first so it cannot be discovered late.

---

## 3. Data model

`parcels`, one table, GORM `AutoMigrate` in `parcel.Migration`, modelled on
`services/atlas-merchant/atlas.com/merchant/frederick/entity.go`.

```go
type Entity struct {
    gorm.Model
    Id       uuid.UUID `gorm:"type:uuid;primaryKey"`
    TenantId uuid.UUID `gorm:"type:uuid;not null"`
    WorldId  byte      `gorm:"not null"`

    SenderId        uint32 `gorm:"not null;index"`
    SenderAccountId uint32 `gorm:"not null"`
    SenderName      string `gorm:"not null"`

    RecipientId        uint32 `gorm:"not null;index"`
    RecipientAccountId uint32 `gorm:"not null"`

    Message   string  // quick sends only (§0 finding C); server-authored on a return leg
    MesoAmount uint32
    FeePaid    uint32

    // Item custody. Nil ItemId == a meso-only parcel.
    ItemId       *uint32
    ItemType     byte
    Quantity     uint16
    ItemSnapshot asset.AssetData `gorm:"type:jsonb"`

    Status   string `gorm:"not null"` // pending | received | discarded | expired
    Quick    bool   `gorm:"not null"`
    Returned bool   `gorm:"not null"`

    CreatedAt    time.Time `gorm:"not null"`
    ReceivableAt time.Time `gorm:"not null"`
    ExpiresAt    time.Time `gorm:"not null"`
    ResolvedAt   *time.Time
    LastNotified *time.Time
}
```

Deltas from PRD §6, each with a reason:

- `ItemId` is a pointer. A meso-only parcel has no item, and `0` is a legal item id in
  no context but reads as one; the PRD called the column nullable but typed it
  `uint32`, which GORM would store as `0`.
- `WorldId` is `world.Id` at the model layer and a `byte` column, matching how
  `atlas-mts` persists world scope. Recipient resolution is world-scoped (§6.1) and
  the mailbox query is world-scoped, so this is not decoration.
- `Message` stays, but its only *client* source is the quick-send arm. On a return leg
  the server authors it (§7.4). It is not part of the standard NPC send.
- No `NotifiedTiers`/`NextDay` escalation. Frederick escalates because a hired merchant's
  goods rot for 100 days; a parcel has one arrival event and one 30-day death. A single
  nullable `LastNotified` is enough (FR-24).

Indexes, unchanged from the PRD and all three justified by a real query:

| Index | Query |
|---|---|
| `(tenant_id, recipient_id, status)` | the mailbox list on the interactive open path (NFR-4) |
| `(tenant_id, sender_id, status)` | the outbound half of gate 12 |
| `(tenant_id, status, expires_at)` | the expiry sweep |

### 3.1 State machine

```
                     receive
   pending  ------------------------>  received
      |  \
      |   \  discard
      |    ------------------------->  discarded
      |
      |  expires_at <= now
      +------------------------------>  expired
                                          |
                        Returned == false |  creates
                                          v
                                    pending (Returned = true, addressed to original sender)
```

An `expired` parcel that already carried `Returned = true` creates nothing (FR-20). The
transition and the return-leg insert happen in **one database transaction inside
atlas-parcel** — see §4.4 for why this deliberately is not a saga.

---

## 4. Asset custody — the central design decision

### 4.1 The problem with the PRD's saga sketch

PRD §7 proposes `DestroyAssetFromSlot` on send and `AwardAsset` on receive. Those two
actions do not compose into a custody transfer: `DestroyAssetFromSlot` annihilates the
asset row, and `AwardAsset` mints a fresh one from a template id and a quantity. An
equipment item's scrolled stats, item level and EXP, owner tag, lock/karma flags,
expiration, cash serial and ring id would all be lost across the send. Duey's whole
purpose is moving *that* item, not an item like it.

### 4.2 Chosen: the existing four-action custody protocol

`libs/atlas-saga/model.go:163-232` already defines this protocol four times over —
storage, trade, cash shop, MTS. The shape is fixed:

- a **composite** action (`transfer_to_mts`, `transfer_to_trade`) that the orchestrator
  expands into `release_from_character` + `accept_to_<custodian>`, looking the item
  snapshot up from inventory during expansion;
- a **composite** withdraw (`withdraw_from_mts`) expanded into
  `release_from_<custodian>` + `accept_to_character`;
- **atomic** accept/release steps dispatched to the custodian service over Kafka, which
  is where the custodian creates or destroys its custody row.

Duey adds the same four:

```go
// Parcel custody (task-241). transfer_to_parcel is a COMPOSITE expanded into
// release_from_character + accept_to_parcel, the same shape as transfer_to_mts.
TransferToParcel   Action = "transfer_to_parcel"
AcceptToParcel     Action = "accept_to_parcel"
ReleaseFromParcel  Action = "release_from_parcel"
WithdrawFromParcel Action = "withdraw_from_parcel"
```

`AcceptToParcelPayload` carries the full snapshot plus the delivery parameters
(recipient identity, meso, fee, quick flag, message, the computed `ReceivableAt` /
`ExpiresAt`) so that atlas-parcel creates the row from the payload alone — exactly
how `AcceptToMtsListingPayload` carries the snapshot plus the sale parameters
(`libs/atlas-saga/payloads.go:879`). `ReleaseFromParcelPayload` carries only the
transaction id and the parcel id, mirroring `ReleaseFromMtsHoldingPayload`.

Cost: two new saga types, four new actions, one new Kafka custody consumer in
atlas-parcel, and orchestrator expansion + compensation entries. That is real work,
and it is the same work `atlas-mts` and `atlas-trades` each paid. The alternative
loses player property.

### 4.3 The two sagas

**`ParcelSend`** (`saga.Type = "parcel_send"`), initiated by atlas-channel after
pre-flight (§6.2):

| # | Action | Notes |
|---|---|---|
| 1 | `award_mesos` (negative `mesoAmount + fee`) | first, so an unaffordable send costs nothing downstream; compensation re-credits |
| 2 | `destroy_asset` (the Quick Delivery Ticket, qty 1) | **only when `Quick`**; FR-26 — consumed on send, not on open |
| 3 | `transfer_to_parcel` | composite → `release_from_character` + `accept_to_parcel` |

Meso first is deliberate and matches `note_send`'s destroy-first reasoning
(`services/atlas-channel/atlas.com/channel/socket/handler/note_send.go:17-19`): the
irreversible-looking step goes first so that the compensating action is a credit
rather than an item re-mint. A meso-only parcel omits step 3's item half; the composite
still runs to create the row (the release/accept pair degenerates to a row create with
a nil snapshot — see §12 RISK-2).

On completion the channel announces `PARCEL[SUCCESSFULLY_SENT]` (mode 0x12), which is
also the arm that makes the client reset its send tab or close the dialog
(`CParcelDlg::OnPacket` default arm, `a1 == 18`).

**`ParcelReceive`** (`saga.Type = "parcel_receive"`), initiated by atlas-channel:

| # | Action | Notes |
|---|---|---|
| 1 | `withdraw_from_parcel` | composite → `release_from_parcel` + `accept_to_character` |
| 2 | `award_mesos` (positive `mesoAmount`) | |

The parcel row transitions to `received` as part of `release_from_parcel`, inside
atlas-parcel's own transaction — the status change and the custody release are the same
fact and must not be two steps that can disagree.

### 4.4 Discard, expiry and return are NOT sagas

Three flows deliberately stay inside atlas-parcel:

- **Discard** destroys the contents (FR-17). Nothing leaves custody; the row goes to
  `discarded` and its snapshot is dropped. A saga would buy nothing.
- **Expiry** (FR-18) transitions `pending` → `expired`.
- **Return** (FR-19/FR-20) inserts a new `pending` row addressed to the original
  sender, carrying the same snapshot and meso, with `Returned = true`.

Expiry and return are one `UPDATE` plus one `INSERT` against rows the service already
owns exclusively. The asset never re-enters an inventory, so there is no cross-service
atomicity to protect and no compensation to define. Routing it through the orchestrator
would add a distributed failure mode to a purely local one. This is a simplification of
PRD FR-19/FR-20, not a reduction in behavior.

---

## 5. Packet work

### 5.1 Families and version span

| Family | Direction | Fname | Span |
|---|---|---|---|
| `PARCEL` | clientbound | `CParcelDlg::OnPacket` | v72 0x120, v79 0x12C, v83 0x142, v84 0x149, v87 0x153, v92 0x175, v95 0x17D, JMS185 0x160 |
| `DUEY_ACTION` | serverbound | `CTabSend::SendParcel` et al. | **v72 0x040 (derived, §5.4)**, v79 TBD, v83/v84 0x041, v87 0x044, v92 0x047, v95 0x046, JMS185 0x039 |

v48 and v61 are `⬜` in both rows and stay out of span.

`PARCEL` is a mode-prefix dispatcher and is implemented per
`docs/packets/DISPATCHER_FAMILY.md`: one discrete struct per arm in a single
consolidated `libs/atlas-packet/parcel/clientbound/parcel.go`, a per-arm body function
in `parcel_body.go` resolving the mode through
`atlas_packet.WithResolvedCode("operations", <fixed key>, ...)`, a
`docs/packets/dispatchers/parcel.yaml` per-version mode table, one
`candidatesFromFName` case per arm, and per-arm verification. It is **not** added to
`dispatcher-lint-baseline.yaml`.

### 5.2 The `PARCEL` arm set (OQ-3 — closed)

Derived from `CParcelDlg::OnPacket` @0x6F56EA and `CParcelDlg::NoticeResult` @0x6F5BE2
on GMS v83. Cosmic's enum is superseded; where the two disagree, this table is right.

| Mode | Key | Body | Client effect |
|---|---|---|---|
| 0x08 | `OPEN` | `bool quickEnabled`, then the parcel list (§5.3) | constructs `CParcelDlg(quickEnabled ? 2 : 0)` |
| 0x09 | `SEND_ENABLE_ACTIONS` | — | re-enables the dialog; `NoticeResult` shows nothing |
| 0x0A | `NOT_ENOUGH_MESOS` | — | SP_5599 |
| 0x0B | `INCORRECT_REQUEST` | — | SP_3903 |
| 0x0C | `NAME_DOES_NOT_EXIST` | — | SP_3904 "double-check the receiver's name" |
| 0x0D | `SAME_ACCOUNT` | — | SP_3905 |
| 0x0E | `RECEIVER_STORAGE_FULL` | — | SP_3906 (the 10-parcel cap, §0 finding G) |
| 0x0F | `RECEIVER_UNABLE_TO_RECEIVE` | — | SP_3907 |
| 0x10 | `SENDER_UNIQUE_CONFLICT` | — | SP_3908 (recipient already owns the one-of-a-kind) |
| 0x11 | `MESO_LIMIT` | — | SP_3977 (level ≤ 15, 1M cap — §6.4) |
| 0x12 | `SUCCESSFULLY_SENT` | — | SP_3901; also resets the send tab or closes the dialog |
| 0x13 | `UNKNOWN_ERROR` | — | SP_3909 |
| 0x14 | `RECV_ENABLE_ACTIONS` | — | re-enables only |
| 0x15 | `RECV_NO_FREE_SLOTS` | — | SP_3910 |
| 0x16 | `RECV_UNIQUE_CONFLICT` | — | SP_3911 |
| 0x17 | `PARCEL_REMOVED` | `uint32 parcelId`, `byte kind` | removes the row from the list; `kind == 3` → SP_3899 "deleted", else SP_3900 "claimed" |
| 0x18 | `PARCEL_ARRIVED` | one `PARCEL` struct | appends to the open dialog + SP_3902 "a new package has been sent" |
| 0x19 | `ALARM_NAMED` | `string senderName`, `bool` | `CUIFadeYesNo::CreateParcelAlarm` — the arrival toast |
| 0x1A | `OPEN_QUICK` | — | constructs `CParcelDlg(1)`, the quick-send-only dialog, no list |
| 0x1B | `ALARM_GENERIC` | `bool` | `CreateParcelAlarm` with the window name |
| 0x1C | `UNKNOWN_ERROR_2` | — | SP_3909 |

Note the shape corrections against PRD §5.3: 0x17 has a body (Cosmic called it a bare
"recv successful message"), 0x18/0x19/0x1B are the notification arms rather than
generic results, and 0x1A/0x1C exist at all. Arms 0x00–0x07 do not exist —
`OnPacket`'s `default` runs `NoticeResult`, which returns without a notice for anything
below 0x0A, so the observable arm set starts at 0x08.

Every mode above is **v83-derived**. Each other version column re-derives its own arm
set before that column is claimed; the storage family's yaml
(`docs/packets/dispatchers/storage_operation.yaml`) is the precedent for how a shifted
column gets recorded, and JMS is expected to shift.

### 5.3 The `PARCEL` struct and the open body

`PARCEL::Decode` @0x4E4345 reads a **fixed 234-byte block** followed by
`bool hasItem` and, if set, one `GW_ItemSlotBase`. Field offsets confirmed by their
consumers:

| Offset | Field | Evidence |
|---|---|---|
| +0 | `uint32 parcelId` | receive/discard encode `*(parcel+0)` as the id (v83 @0x6F0CA3, @0x6F0DC3) |
| +4 | `char[13] senderName` | `CTabReceive::SetParcel` formats SP_3878 with `parcel+4` |
| +17 | `uint32 mesos` | SP_3879 formatted with `*(parcel+17)` |
| +21 | `FILETIME sentAt` | v72 `ReceiveParcel` computes its 30-day window from `*(uint64*)(parcel+21)` |
| +29..233 | message + padding | the remainder of the fixed block |
| +234 | optional item | `bool` + `GW_ItemSlotBase` |

The exact byte layout of +29..233 is derived during implementation; the design
requires only that the block is fixed-width and that the message lives inside it, which
is why a quick-send message is visible to the recipient at all.

`OPEN` (0x08) body, from `CTabReceive::SetParcel` @0x6EF69C:

```
bool  quickEnabled
byte  count            ; the mailbox
PARCEL x count
byte  newCount         ; parcels arrived since the last open
PARCEL x newCount      ; each raised as its own CUtilDlg::Notice (SP_3878/3879/3880)
```

The second list is the client's own "what showed up while you were away" mechanism.
Atlas populates it from parcels whose `LastNotified` is null, and stamps
`LastNotified` when the open packet is built — this is the cheapest correct
implementation of FR-24 and it needs no extra packet.

### 5.4 `DUEY_ACTION` arms, and OQ-2 — closed

Derived arms, opcode 0x041 on v83:

| Mode | Key | Body | Source |
|---|---|---|---|
| 0x02 | `SEND` | `byte invType`, `uint16 slot`, `uint16 quantity`, `uint32 mesos`, `string recipientName`, `bool quick`; **and when `quick`**: `string message`, `uint32 ticketRef` | @0x6F36A8 (quick=0, stops after the flag) and @0x6F1DF5 (quick=1, two extra fields) |
| 0x04 | `RECEIVE` | `uint32 parcelId` | v72 @0x65AF41 |
| 0x05 | `DISCARD` | `uint32 parcelId` | v83 @0x6F0DC3 |
| 0x07 | `CLOSE` | — | v83 `CParcelDlg::CloseParcelDlg` @0x6F5691 |

Modes 0x00, 0x03 and 0x06 are unattested in either build; the registry's fname list
names six send sites and four are accounted for above, with
`CUIFadeYesNo::OnButtonClicked` being the confirm path into 0x04/0x05. Implementation
re-derives per version and records any additional arm.

**OQ-2 is closed: the serverbound op exists on v72 at opcode 0x040.** `CTabReceive::ReceiveParcel`
@0x65AF41 constructs `COutPacket(64)` and encodes mode 4 plus the parcel id, and
`CTabSend::SendParcel` @0x65D940 constructs the same opcode with mode 2. The `⬜` on
v72/v79 in `docs/packets/audits/STATUS.md:596` is a CSV-import coverage gap
(`provenance: csv-import` on every `DUEY_ACTION` registry entry, versus
`provenance: ida-discovered` on the v72 `PARCEL` entry) — not a client that lacks the
op. The span is **not** reduced. v72's registry entry is added with
`provenance: ida-discovered` and the address above; v79 is derived the same way during
implementation.

Note the wire is a superset of the PRD's FR-1: the quick arm carries a message and a
ticket reference the PRD did not anticipate, and neither arm carries the message on the
NPC path.

### 5.5 The 234-byte fixed block and the version span

The GW_ItemSlotBase encoder already exists and is version-aware; the fixed block is
new. It is written once with version gates via the `MajorAtLeast` idiom rather than raw
comparisons, per the repository's packet conventions, and no wire change is made to an
already-verified version (there are none in these two rows today).

---

## 6. Send flow

### 6.1 Recipient resolution (OQ-1 — closed)

Resolved **in atlas-channel**, before any saga, exactly as `note_send` resolves its
receiver (`note_send.go:66`):

```go
cs, err := character.NewProcessor(l, ctx).ByNameProvider(toName)()
```

`ByNameProvider` returns `[]Model` from atlas-character's `get_characters_by_name`
(`services/atlas-character/atlas.com/character/character/resource.go:35`), which is
tenant-scoped and name-filtered but **not** world-filtered. The channel filters the
result to `s.WorldId()` itself; the model already exposes `WorldId()` and
`AccountId()` (`services/atlas-channel/atlas.com/channel/character/model.go:241,269`),
which is what makes the same-account check (FR-8) possible without a second lookup.

No new endpoint on atlas-character is needed. This is a change from the PRD's OQ-1
expectation that a world-scoped name lookup might have to be built.

Ambiguity handling: a name that resolves to several characters across worlds narrows to
at most one after the world filter, because character names are unique within a world
under both `NameScopeWorld` and `NameScopeTenant`. Zero matches after filtering →
`PARCEL[NAME_DOES_NOT_EXIST]` (0x0C).

### 6.2 Pre-flight, in atlas-channel

All of these reject inline with a `PARCEL` result arm and start no saga, mirroring
`note_send`'s pre-flight posture:

| Check | Result arm |
|---|---|
| no item and no meso | `INCORRECT_REQUEST` (0x0B) |
| `mesoAmount + fee` overflows, or exceeds the 100,000,000 per-parcel cap | `INCORRECT_REQUEST` |
| sender level ≤ 15 and `mesoAmount > 1,000,000` | `MESO_LIMIT` (0x11) |
| sender holds < `mesoAmount + fee` | `NOT_ENOUGH_MESOS` (0x0A) |
| name resolves to nothing in the sender's world | `NAME_DOES_NOT_EXIST` (0x0C) |
| recipient's account == sender's account | `SAME_ACCOUNT` (0x0D) |
| recipient already holds 10 pending parcels | `RECEIVER_STORAGE_FULL` (0x0E) |
| quick flag set but sender holds no 5330000 | `INCORRECT_REQUEST` + warn log |
| message > 100 chars (quick arm only) | `INCORRECT_REQUEST` |

**Never a disconnect** (NFR-5). Cosmic disconnects and autobans on the packet-edit
cases; Atlas rejects and logs at warn. This is a deliberate divergence and it is
asserted by test.

The mailbox-capacity check needs a count from atlas-parcel and is one call to
`GET /parcels?filter[recipientId]=&filter[status]=pending`; it is the only pre-flight
that leaves the channel besides name resolution.

### 6.3 Fee

```
fee(m) = 0                    m <  100_000
       = trunc(m * 0.008)     m >= 100_000
       = trunc(m * 0.018)     m >= 1_000_000
       = trunc(m * 0.03)      m >= 5_000_000
       = trunc(m * 0.04)      m >= 10_000_000
       = trunc(m * 0.05)      m >= 25_000_000
       = trunc(m * 0.06)      m >= 100_000_000
total  = fee(m) + (quick ? 0 : 5_000)
```

**This contradicts NFR-8, deliberately.** NFR-8 asks for integer arithmetic
(`m * 18 / 1000`). The client computes the fee itself in IEEE-754 double and shows the
result to the player in the confirm dialog before the packet is ever sent
(v72 `sub_6590A1` @0x6590A1, v83 `sub_6EEDFE` @0x6EEDFE, formatted into SP_3897 /
SP_3896). If the server charges an integer-derived figure and the client quoted a
double-derived one, the player is charged a number they were not shown. Matching the
client exactly is the only way to keep the quote and the charge identical, so the
implementation uses `uint32(float64(m) * rate)` with the same rate constants and the
same descending comparison order.

Overflow is checked in `uint64` before any deduction, and the 100,000,000 per-parcel
meso ceiling the client's own input dialog enforces is re-enforced server-side.

### 6.4 The meso limit (OQ-5 — closed)

Mode 0x11 is not a per-parcel cap and not the recipient's meso ceiling. From
`sub_6F3875` @0x6F3875: the client refuses and shows SP_3977 when the character's level
is **≤ 15** and the meso amount **exceeds 1,000,000**. The string names it a per-day
limit; the client enforces it per-transaction only. Atlas enforces the
per-transaction form the client enforces — a per-day accumulator has no wire support,
no client display, and no goal in the PRD requiring it. Server-side enforcement is
mandatory regardless, because the client check is trivially bypassable.

Separately, the input dialog hard-caps a single parcel at 100,000,000 meso; that is
enforced as an `INCORRECT_REQUEST`.

---

## 7. Receive, notification and return

### 7.1 Notification transport (OQ-8 — closed)

Reuses the frederick chain verbatim:

```
atlas-parcel notification task
   -> Kafka parcel status event (PARCEL_ARRIVED)
       -> atlas-channel consumer, tenant-matched against sc.Tenant()
           -> session.IfPresentByCharacterId(channel)
               -> Announce PARCEL[ALARM_NAMED] (0x19) { senderName, hasItem }
```

The consumer shape is
`services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go:454`,
including its tenant guard and its `IfPresentByCharacterId` no-op when the recipient is
not on this channel.

Two mechanisms, one for each requirement:

- **FR-23 (online arrival)**: the notification task sweeps parcels whose
  `ReceivableAt` has passed and whose `LastNotified` is null, emits the event, and
  stamps `LastNotified`. If the recipient is offline the announce is a no-op and
  `LastNotified` is still stamped — which is correct, because FR-24 is served by a
  different path.
- **FR-24 (login / open)**: the `OPEN` packet's second parcel list (§5.3) is exactly a
  "these arrived since you last looked" list, rendered by the client as one notice per
  parcel. Populated from `LastNotified IS NULL` at open time, and stamped there too.

`LastNotified` therefore has a single meaning — "the player has been told about this
parcel once" — and satisfies FR-24's no-renotify requirement without a tier ladder.

A parcel that arrives while the dialog is already open uses `PARCEL_ARRIVED` (0x18)
instead of the alarm, since 0x18 both appends the row and raises SP_3902. The channel
picks between 0x18 and 0x19 by whether the session has an open parcel dialog; if the
channel does not track that state, it always sends 0x19 and accepts that a player with
the dialog open sees a toast rather than a live row (documented, low-severity).

### 7.2 Receive

Pre-flight in atlas-channel before the `ParcelReceive` saga:

| Check | Result arm |
|---|---|
| no free slot of the item's inventory type | `RECV_NO_FREE_SLOTS` (0x15) |
| recipient already holds a one-of-a-kind copy | `RECV_UNIQUE_CONFLICT` (0x16) |
| parcel not `pending`, not addressed to this character, or `ReceivableAt` in the future | `INCORRECT_REQUEST` (0x0B) |

Both inventory checks are existing atlas-inventory queries; no new endpoint. On saga
completion the channel announces `PARCEL_REMOVED` (0x17) with `kind != 3`, which
removes the row and shows SP_3900 "successfully claimed".

The parcel stays `pending` on every rejection (FR-15, FR-16).

### 7.3 Discard

`DISCARD` (0x05) → atlas-parcel marks the row `discarded`, contents dropped. The
channel announces `PARCEL_REMOVED` (0x17) with `kind == 3` → SP_3899 "successfully
deleted". The client already confirms via `CUIFadeYesNo` before sending (SP_3889), so
no server-side confirmation exists or is needed.

### 7.4 Expiry, return, and how a return leg presents (OQ-7 — closed)

The expiry task (§8.2) transitions `pending` → `expired` at `ExpiresAt` and, when
`Returned == false`, inserts a return leg: same snapshot, same meso, `SenderId` and
`RecipientId` swapped, `Returned = true`, `FeePaid = 0` (the fee is not refunded,
FR-21), fresh `CreatedAt` / `ReceivableAt` / `ExpiresAt`.

**There is no wire field for "this is a return."** The `PARCEL` struct carries a sender
name and a message and nothing else that could encode it. So the distinction is
server-side, surfaced through the two fields that do exist:

- `SenderName` is set to the original **recipient's** name, so the parcel reads as
  coming back from the person who never claimed it;
- `Message` is server-authored: `"Unclaimed parcel returned."`.

`ReceivableAt` on a return leg is `CreatedAt` (no 24-hour delay) — the delay models
shipping time to a third party, and the goods are coming home.

A return leg that itself expires is destroyed and creates nothing, which is what the
`Returned` flag exists to guarantee.

**The 30-day client guard.** v72's `ReceiveParcel` refuses to send a receive request
when the parcel's timestamp falls outside a 30-day window (§0 finding H). Since server
expiry is also 30 days, a parcel can in principle become client-unclaimable in the same
hour it becomes server-expired. The implementation pins the exact polarity of that
comparison against the IDB and, if the client's window is the binding one, expiry moves
to 29 days so the server always retires a parcel before the client refuses it. This is
a small, evidence-driven adjustment to FR-18 and is recorded when made.

---

## 8. Background tasks

Both follow `services/atlas-merchant/atlas.com/merchant/frederick/task.go` and
`notification_task.go`: a `Run()` + `SleepTime()` pair, driven off
`database.WithoutTenantFilter(ctx)` for the sweep and re-entering a per-tenant context
before any Kafka emit.

### 8.1 Concurrency (NFR-7 — resolved)

Frederick's tasks are not leader-elected; every replica runs them. That is safe there
because the operations are idempotent deletes. Duey's expiry is not idempotent — it
inserts a return leg — so it needs a guard, and the cheapest correct one is the update
itself:

```sql
UPDATE parcels SET status = 'expired', resolved_at = NOW()
 WHERE status = 'pending' AND expires_at <= NOW()
 LIMIT :batch
 RETURNING id, ...;
```

Only one replica's `UPDATE` claims a given row; the returned rows are that replica's
work list, and the return-leg inserts are built from them. No leader election, no long
transaction, and the batch bound satisfies NFR-7's "must not hold a long transaction
over the whole table."

The notification sweep uses the same claim-by-update shape on `LastNotified`.

### 8.2 Intervals

`DefaultExpiryInterval = 1 * time.Hour` and `DefaultNotificationInterval = 5 * time.Minute`,
both overridable by env like frederick's. A 30-day deadline does not need a tighter
expiry sweep; an arrival notification does.

### 8.3 Testing the 24-hour delay (OQ-4 — closed)

Chosen: **an injectable clock on the processor**, `now func() time.Time`, defaulted to
`time.Now` in `NewProcessor` and overridden in tests through the same unexported
with-style seam `pending_change` uses for `gateDeps`
(`processor_eligibility.go:73`). Production behavior is fixed at 24 hours and no
configuration surface is added, which is what the PRD preferred.

The delay itself is not injectable — `ReceivableAt` is a stored column, so a test that
wants a receivable parcel builds one with the Builder and a past `ReceivableAt`. The
clock seam exists for the *sweeps*, where "now" is the thing under test.

---

## 9. Cross-service changes

### 9.1 atlas-character — gate 12 `parcel_pending`

Follows the file's contract exactly (`processor_eligibility.go`):

- a `parcelPending func(l, ctx, characterId) (bool, error)` field on `gateDeps`;
- wired in `productionGateDeps()` to a `parcelPending` REST client in `requests.go`
  hitting `GET /characters/{characterId}/parcel-status`, one narrow call, matching how
  `mtsHoldingOpen` is a narrow lookup (`requests.go:184`);
- a `checkParcelPending` method returning `("parcel_pending", true, nil)`;
- **appended as gate 12 in BOTH** `evaluateTransferEligibility` and
  `evaluateTransferEligibilityIndependent`, in the same relative order — the file's
  documented invariant, and the reason the CHECK-time and BUY-time handlers agree;
- destination-INDEPENDENT;
- a dependency error propagates and `runGates` converts it to `check_unavailable`
  (`processor_eligibility.go:134-142`), failing closed without asserting a reason the
  server does not know.

The endpoint returns one boolean answering "any parcel in flight, either direction" —
outbound `pending`, or inbound `pending` with `ReceivableAt <= now`. One endpoint rather
than two list calls, so the gate is one round trip.

### 9.2 Surfacing the reason

Rejection reasons are mapped to client error codes in the CashShop error-writer options
of each seed template, not in Go. `parcel_pending` is added alongside `merchant_open`
and `mts_listings_open` — the `CANNOT_TRANSFER_OUT` bucket — in **all nine**
`services/atlas-configurations/seed-data/templates/template_*.json`
(e.g. `template_gms_83_1.json:5041`, value `222`; the bucket's numeric value differs per
template and each is read from its own file, never copied).

Missing this in even one template leaves that version showing "please try again"
instead of "cannot transfer out"; §11 makes it one task so it cannot be partially done.

### 9.3 GM restriction (OQ-6 — closed)

**Not implemented.** Cosmic gates Duey behind `MINIMUM_GM_LEVEL_TO_USE_DUEY`; no PRD
goal requires it, no client arm expresses it, and Atlas's existing GM handling on this
surface is gate 2 (`is_gm`) blocking world transfer, which is a different concern.
Adding an unrequested restriction is scope the PRD did not ask for.

### 9.4 atlas-npc-conversations

NPC 9010009 gets a conversation whose terminal operation is a new `open_duey` op in
`operation_executor.go`, modelled on `open_storage` at line 2105: read the NPC id from
the conversation context, build a `ShowParcelPayload{CharacterId, NpcId, WorldId,
ChannelId}`, and return `saga.ShowParcel` as a `Pending` step. The orchestrator turns it
into a `SHOW_PARCEL` Kafka command; atlas-channel's consumer fetches the mailbox over
REST and announces `PARCEL[OPEN]`.

### 9.5 atlas-channel — the Quick Delivery Ticket

`handleDueyCouponUse` is added to the USE_CASH_ITEM classification dispatch beside
`handleRemoteMerchantUse` (`character_cash_item_use.go:787`), classification-first for
the same reason stated there (cash-slot type bytes collide). It announces
`PARCEL[OPEN_QUICK]` (0x1A) and **consumes nothing** — the ticket is destroyed by the
`ParcelSend` saga (FR-26), and the client itself pre-checks
`CWvsContext::IsExist(5330000)` before letting the player send.

`GetCashSlotItemType` already maps `ClassificationDueyCoupon` → 31
(`character_cash_item_use.go:1410`); no change there.

### 9.6 Constants

`libs/atlas-constants` has no `npc` package and no id constant for either value today
(grep for `9010009` and `5330000` returns nothing repo-wide). Rather than create an
`npc` package for one id, the NPC id lives in the conversation script data where every
other NPC id lives, and the ticket id becomes a constant in
`libs/atlas-constants/item` beside the existing `ClassificationDueyCoupon`
(`item/constants.go:109`). Checked against the existing constants first, per the
repository convention.

---

## 10. Testing

| Requirement | Test |
|---|---|
| FR-2 fee table | table test at all twelve boundaries, asserting the double-truncated value |
| FR-3 surcharge direction | two cases, quick and non-quick, asserting `+5000` only on non-quick |
| FR-5/FR-6 validation | rejection cases assert the result arm AND that the session is not closed (NFR-5) |
| FR-7/FR-8 resolution | name resolving in another world → `NAME_DOES_NOT_EXIST`; same account → `SAME_ACCOUNT` |
| §0 finding G | an 11th parcel to a full mailbox → `RECEIVER_STORAGE_FULL` |
| §6.4 meso limit | level 15 + 1,000,001 → `MESO_LIMIT`; level 16 + same → allowed |
| FR-12 delay | a parcel with `ReceivableAt` in the future is absent from the mailbox query and rejected on receive |
| FR-18/19/20 | expiry produces a return leg; a return leg's expiry produces nothing |
| NFR-1 tenancy | `provider_tenant_test.go`, modelled on frederick's and storage's |
| NFR-2 atomicity | saga rollback test: a failed `accept_to_parcel` re-credits meso and restores the item with its snapshot intact |
| NFR-3 idempotency | replayed receive awards once, following `services/atlas-storage/atlas.com/storage/storage/processor_idempotency_test.go` |
| gate 12 | rejection from BOTH evaluate functions; atlas-parcel outage → `check_unavailable`; no parcels → unaffected |
| packets | per-arm byte fixtures with `// packet-audit:verify` markers; `dispatcher-lint`, `matrix --check`, `fname-doc --check`, `operations --check` all exit 0 |

All setup uses the project's Builder pattern; no `*_testhelpers.go`.

The NFR-2 rollback test is the one that earns its keep — it is the test that would have
caught the destroy/award custody bug §4.1 describes, because a re-awarded item loses
its stats and a released/re-accepted one does not.

---

## 11. Build order

Ordered so that each block is independently verifiable and nothing blocks on a
half-finished predecessor.

1. **Service scaffolding.** `atlas-parcel` per `docs/adding-a-new-service.md` — build
   target, k8s base, both overlays, ingress, database, package visibility. Entity,
   migration, builder, provider, administrator, processor, REST, tenant test. No
   feature behavior. Verifiable on its own and the largest infrastructure risk, so it
   goes first.
2. **Packet derivation.** `docs/packets/dispatchers/parcel.yaml` per-version mode
   tables and the v72/v79 `DUEY_ACTION` registry entries (§5.4). Documentation and
   registry only — this is the block that fixes OQ-2's gap in the record.
3. **Codecs.** `libs/atlas-packet/parcel/{clientbound,serverbound}` — the discrete
   arm structs, the body functions, the `PARCEL` 234-byte struct, `run.go` cases.
   Per-arm fixtures and verification.
4. **Saga custody.** The four actions and two saga types in `libs/atlas-saga`,
   orchestrator expansion and compensation, atlas-parcel's custody consumer.
5. **Send flow.** atlas-channel `DUEY_ACTION` handler, pre-flight, fee, `ParcelSend`.
6. **Receive and discard flow.** Handler arms, `ParcelReceive`, discard.
7. **Entry points.** `open_duey` NPC op + `ShowParcel` chain; `handleDueyCouponUse`.
8. **Tasks.** Expiry/return sweep, notification sweep, the channel-side alarm consumer.
9. **World transfer.** Gate 12, the REST client, the `parcel-status` endpoint, and the
   `parcel_pending` reason in all nine templates — as one task.

---

## 12. Risks

**RISK-1 — the packet block is nine columns wide.** Two families across eight version
columns each, one of them a 21-arm dispatcher, is the bulk of the work and the part
most likely to expand. v83 is fully derived; every other column is not. Mitigation: the
span is fixed in §5.1 and each column re-derives rather than assuming v83; a column that
turns out to diverge is recorded in `parcel.yaml` the way storage's divergence is,
not papered over.

**RISK-2 — a meso-only parcel through a custody protocol.** `transfer_to_parcel`
expands into a release/accept pair whose whole purpose is moving an asset. A meso-only
parcel has no asset. The expansion must produce a parcel row with a nil snapshot without
dispatching a meaningless release, and the compensation must not try to re-award an item
that never existed. This is the sharpest edge in §4 and it needs an explicit test.

**RISK-3 — the double-precision fee.** §6.3 chooses to match the client's floating
point rather than NFR-8's integer arithmetic. If any version's client uses a different
rate constant or a different rounding mode, the quote and the charge diverge on that
version. Mitigation: the fee function is re-read per version during the packet pass,
and the boundary table test runs against whatever the client's constants actually are.

**RISK-4 — the 30-day client guard.** §7.4 flags that the client may refuse a receive
before the server expires the parcel. The polarity of that comparison is not yet pinned
and it decides whether expiry stays at 30 days or moves to 29. It is a one-line
decision but it must be made from the IDB, not assumed.

**RISK-5 — nine template edits for one reason string.** §9.2 touches every seed
template. A missed file is silent: that version shows the wrong error text and nothing
fails. Mitigation: it is one task, and the acceptance check greps all nine.
