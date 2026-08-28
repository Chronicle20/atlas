# Trade Domain

## Responsibility

Manages the trade room lifecycle: opening a solo room, inviting and seating a
visitor, staging items and meso, the two-phase confirm/attest handshake, and
settlement or teardown. Enforces every staging restriction and settlement
pre-check, resolves the tenant's meso tax, and drives every asset or meso
movement through a saga submitted to atlas-saga-orchestrator. Live rooms are
held in a process-local, tenant-partitioned in-memory registry, not the
database.

## Core Models

### State

The room lifecycle (`trade` package):

| State | Meaning |
|-------|---------|
| `OPEN_SOLO` | Room created, no visitor invited or seated. |
| `PENDING_INVITE` | An invite is outstanding, naming one admitted character. |
| `OPEN` | Both sides seated; staging is permitted. |
| `AWAITING_ATTESTATION` | Both sides confirmed; both clients have been prompted (mode 17) for their CRC attestation. |
| `SETTLING` | Both sides attested; a settlement saga is in flight. |

### StagedItem

One item claimed for trade, held in a `Participant`.

| Field | Type | Description |
|-------|------|-------------|
| tradeSlot | byte | The 1..9 client dialog slot. |
| assetId | asset.Id | The source asset's identity. |
| templateId | item.Id | The item template. |
| quantity | asset.Quantity | Amount staged (may be a partial stack). |
| inventoryType | inventory.Type | Source compartment, kept for provenance only. |
| sourceSlot | slot.Position | Source slot, kept for provenance only. |
| escrowId | uuid.UUID | The custody row this item lives in; also the transaction id of the staging saga that created it. |
| pending | bool | True between submitting the stage and the escrow row being confirmed. A pending item holds its dialog slot but is not yet announced to either dialog. |

### Participant

One side of the trade room.

| Field | Type | Description |
|-------|------|-------------|
| characterId | character.Id | The participant. |
| name | string | |
| position | byte | 0 = room owner, 1 = invited/seated visitor. |
| confirmed | bool | Pressed Trade. |
| attested | bool | Replied to the mode-17 CRC prompt. |
| mesoStaged | uint32 | Committed staged meso (advances only when a stake resolves). |
| mesoTax | uint32 | Resolved at settlement: tax destroyed from this side's staged meso. |
| mesoDelivered | uint32 | Resolved at settlement: what the counterparty receives out of this side's staged meso. |
| confirmEntries | []CrcEntry | The CRC list sent with TRADE_CONFIRM. |
| attestEntries | []CrcEntry | The CRC list sent with the TRANSACTION reply. |
| items | []StagedItem | Items staged by this side. |
| pendingMesoTxId | uuid.UUID | Transaction id of an in-flight meso stake, or uuid.Nil. |
| pendingMesoAmount | uint32 | Absolute target the in-flight stake is moving toward. |

### Room

One live trade room. Immutable; constructed via `Builder` (`NewBuilder`) and
mutated only through `With*` copy transforms applied inside `Registry.Update`.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | REST identity and registry key. |
| handle | uint32 | Wire serial the client's invite carries; defaults to the owner's character id. |
| roomType | byte | miniroom type byte: `miniroom.Trade` (3) or `miniroom.CashTrade` (6). |
| f | field.Model | World, channel, map, instance the room was opened in. |
| state | State | |
| settlementId | uuid.UUID | Transaction id of the settlement saga this room submitted; also the ledger's idempotency key. |
| invitedId | character.Id | The character an outstanding invite names; the only character `EnterRoom` will seat. Zero when no invite is outstanding. |
| participants | []Participant | 1 (solo) or 2 (paired) participants. |
| createdAt | time.Time | |

## Invariants

- A character occupies at most one room per tenant, and a wire handle
  identifies at most one room per tenant (`Registry.Create`/`Update`).
- `Room.Admits(characterId)` is the only predicate that decides whether a
  character may be seated by `EnterRoom`: the wire handle defaults to the
  owner's character id and is therefore public, so possession of it proves
  nothing on its own.
- `Room.Frozen()` is true once the room has left `OPEN`, or once either
  participant has confirmed. A frozen room rejects `PutItem`, `AddMeso`, and
  any further `Confirm`.
- `Room.BothConfirmed()`/`BothAttested()` require exactly two participants; a
  solo room can never satisfy either.
- Every registry mutation (`Create`, `Update`, `RemoveIf`) is a
  compare-and-set under one write lock, so two racing commands against the
  same room cannot both win a state transition.
- A staged item is born `pending`; it holds its dialog slot but is not
  announced to either dialog until its `transfer_to_trade` saga completes
  (`StageSucceeded`).
- `checkRestrictions` (staging) refuses: an unstageable source compartment, an
  equipped item (negative source slot), an asset carrying the untradeable or
  merge-untradeable flag (unless masked by a karma mark), an unreadable
  atlas-data lookup, or an item whose WZ data sets `tradeBlock`. A refusal is
  silent to the client; the empty dialog slot is the only feedback.
- A karma mark defeats both the untradeable-flag and `tradeBlock` refusals for
  exactly one transfer, and is consumed (masked off) by a settled transfer; an
  unwound (cancelled) trade replays the item unmasked.
- `maxStageableMeso` bounds a single side's staged meso at `math.MaxInt32`.
- `settle` runs four pre-checks against fresh reads before submitting a
  settlement saga: (1) each side has room to receive what it is about to be
  handed, simulating atlas-inventory's merge-then-spill accept; (2) each
  side's meso stays inside `[0, maxStageableMeso]` after paying out and being
  paid, using the counterparty's *delivered* (post-tax) amount; (3) each
  side's meso escrow custody agrees exactly with what the room is about to
  deliver, with nothing still in flight; (4) each side's TRANSACTION
  attestation matches the counterparty's staged contribution, as a multiset
  of `{data, crc}` pairs cross-checked against that side's own CONFIRM list.
  An empty attestation always matches (GMS <= v79 carries none; the timeout
  path settles with none).
- A settlement refusal or a teardown trigger always tears the room down
  (removes it from the registry and returns every escrowed asset/meso to its
  owner); nothing is left half-open.
- A room already `SETTLING` is immune to teardown (`FR-6.5`): cancel loses to
  settlement, and the saga's terminal status produces that room's client-visible
  outcome.
- Meso staging (`AddMeso`) assigns an absolute target, not a delta (mode 16 is
  an assignment on the client). The movement submitted to the saga is the
  delta against the sum of committed-plus-in-flight escrow, computed against
  the escrow store, never against the room's own `mesoStaged`, which only
  advances once a stake resolves.
- A meso reduction may be netted against in-flight raises; a meso raise may
  never be netted against an in-flight reduction that has not yet landed —
  doing so would mint meso against a debit that could still fail.
- The attestation deadline (`attestationTimers`, default 5s, tenant
  configurable) is armed the instant a room moves to `AWAITING_ATTESTATION`
  and settles the room on whatever attestation arrived if it fires
  (`ExpireAttestation`); it is defence in depth, not a liveness dependency.
- A `SETTLING` transition that is not durably followed by a submitted saga
  (its command failed after the in-memory swap) is recovered by closing the
  trade (`recoverAbandonedSettlement`/`abandonSettlement`), never by
  reverting the state — the room cannot be driven forward a second time.

## State Transitions

### Room Lifecycle

1. **CreateRoom**: Validates the caller is alive, on a map that allows
   trading, and at or above the tenant's minimum trade level, and that they
   occupy no other room. Opens a solo room (`OPEN_SOLO`).
2. **Invite**: Only the owner, only from `OPEN_SOLO` or `PENDING_INVITE`.
   Validates the target is alive, reachable, in the same field, not already
   in a room, and not the inviter. Moves to `PENDING_INVITE`, records the
   target as the admitted character, and issues a trade invite via
   atlas-invites.
3. **EnterRoom**: Admits only the character the outstanding invite named,
   into a room of the requested `roomType`, re-running the dead/map/level
   ladder against the enterer's *current* location. Seats the visitor at
   position 1 and moves to `OPEN`.
4. **DeclineInvite / InviteRejected**: Tears the originator's `PENDING_INVITE`
   room down. `DeclineInvite` (client-originated) also retires the offer in
   atlas-invites; `InviteRejected` (atlas-invites already retired it, via an
   explicit reject or an expiry) does not re-notify atlas-invites.
5. **PutItem / AddMeso**: Legal only against an unfrozen, seated room.
   `PutItem` validates the compartment, restrictions, and available quantity,
   then stages a pending item and submits a `transfer_to_trade` saga keyed by
   the escrow row id. `AddMeso` computes the delta against committed-plus-
   in-flight escrow and submits an `award_mesos` saga keyed by a fresh stake
   id.
6. **StageSucceeded / StageFailed**: `StageSucceeded` clears the item's
   pending flag and announces it to both dialogs (`ITEM_STAGED`).
   `StageFailed` frees the dialog slot and unlocks the staging client
   (`ITEM_REFUSED`) if the item was still pending; otherwise the item is
   already escrowed and its return is left to the saga's own compensation.
7. **MesoStageSucceeded / MesoStageFailed**: Resolve one in-flight meso stake
   against the durable escrow row first, then the room if one still exists.
   A room-less resolution (torn down mid-stake) refunds or discards the
   settled delta directly.
8. **Confirm**: Legal only from `OPEN`. Records one side's confirm and CRC
   list. Once both sides have confirmed, moves to `AWAITING_ATTESTATION`,
   prompts both clients, and arms the attestation deadline.
9. **Attest / ExpireAttestation**: `Attest` records one side's TRANSACTION
   reply. Once both sides have attested (or the deadline lapses), `settle`
   runs the four pre-checks; a pass submits one `trade_settlement` saga and
   moves the room to `SETTLING`; a failure tears the room down with the
   corresponding leave reason.
10. **SettlementSucceeded / SettlementFailed**: Driven by the durable
    settlement record, not the live room — the record survives a restart the
    room cannot. On success: writes the ledger row, discharges the escrow
    meso, deletes the record, removes the room, and announces `SETTLED`. On
    failure: deletes the record, unwinds every escrowed asset/meso from the
    record, removes the room, and announces `CANCELLED`.
11. **UnwindFailed / UnwindSucceeded**: Recover or discard the bookkeeping of
    a `trade_unwind` saga's own terminal status — an id space none of the
    other three (stage, stake, settlement) claims.
12. **TeardownCharacter**: Removes the character's room (unless it is
    `SETTLING`) and returns every escrowed asset/meso to its owner via a
    `trade_unwind` saga.

### Leave Reasons

| Reason | Trigger |
|--------|---------|
| `TRADE_CANCELLED` | Client CANCEL, invite decline/rejection, logout, session destroy, empty settlement path. |
| `TRADE_FAILED` | Settlement submission itself failed, or the settlement saga reported terminal failure, or a meso/reservation check failed at settlement. |
| `TRADE_CANNOT_CARRY` | A settlement pre-check found insufficient inventory space or meso headroom. |
| `TRADE_DIFFERENT_MAP` | The character changed map or channel away from the room's field. |
| `TRADE_CRC_FAILED` | A settlement attestation did not match the counterparty's staged items. |

### Startup Reconciliation

`ReconcileAtBoot` runs, in order: (1) `Reconcile` — for every settlement
record still unresolved, asks atlas-saga-orchestrator for the saga's outcome
and drives `SettlementSucceeded`/`SettlementFailed` accordingly, leaving an
unknown outcome untouched; (2) `ReconcileEscrow` — returns every escrow item
and meso row whose room id is not named by a still-unresolved settlement
record, grouped per stranded room into one `trade_unwind` saga. The exclusion
set is captured before the settlement pass runs, so a settlement resolved by
this same boot is not double-swept.

## Processors

### Processor (`trade` package)

Owns every trade-room operation; REST handlers and Kafka consumers go through
it rather than the registry directly. Constructed per request via
`NewProcessor(l, ctx, db)`, which resolves the tenant from `ctx` once. Every
command runs inside `emit`, which opens one DB transaction, hands the closure
a tx-scoped processor, and enqueues every buffered Kafka message into the
transactional outbox in that same transaction — a command's durable writes
and all its status events commit atomically and publish in buffer order.
Registry mutations are in-memory and are **not** rolled back by a failed
transaction.

Selected operations (see the `Processor` interface for the full contract):
`RoomsForTenant`, `RoomById`, `RoomForCharacter`, `RoomByHandle`,
`CreateRoom`, `Invite`, `DeclineInvite`, `InviteRejected`, `EnterRoom`,
`PutItem`, `Chat`, `AddMeso`, `Confirm`, `Attest`, `ExpireAttestation`,
`SettlementSucceeded`, `SettlementFailed`, `UnwindFailed`, `UnwindSucceeded`,
`ReconcileSettlements`, `StageSucceeded`, `StageFailed`,
`MesoStageSucceeded`, `MesoStageFailed`, `TeardownCharacter`.

### Registry (`trade` package)

The tenant-partitioned in-memory store of live rooms. One `RWMutex` guards
three maps (rooms, member index, handle index), maintained only inside
`Create`/`Update`/`Remove`/`RemoveIf` under the write lock. `Update` and
`RemoveIf` are compare-and-set: the caller's transform/claim function runs
under the write lock and must be pure (no I/O, no re-entry).

### restriction (`trade` package)

`checkRestrictions` evaluates the staging rules against an `assetView` (flags,
source slot, template id) and an `itemDataView` (WZ `tradeBlock`, lookup
readability). `stageableInventoryType` decodes the raw wire compartment byte
into one of the five stageable `inventory.Type`s.

### attestationTimers (`trade` package, settlement.go)

Process-wide registry of armed attestation deadlines, one per room in
`AWAITING_ATTESTATION`. `Arm` replaces any existing deadline for a room;
`Cancel` disarms one; `StopAll` disarms every deadline at shutdown. Armed
against a detached context (tenant preserved, no cancellation), because the
deadline fires long after the command that armed it has returned.

# Escrow Domain

## Responsibility

Durable custody store for staged items and staged meso (`escrow` package).
The reference client arms an exclusive-request lock on `PUT_ITEM`/`PUT_MONEY`
that only a real inventory or stat mutation clears, so a staged item
genuinely leaves its owner's compartment at stage time rather than being
merely reserved. This package is the record of who staged what, so a crash
cannot strand it.

## Core Models

### ItemModel

One staged asset in trade custody.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Escrow row id; also the staging saga's transaction id. |
| roomId | uuid.UUID | Owning trade room. |
| ownerId | character.Id | The staging character. |
| tradeSlot | byte | Client dialog slot. |
| sourceInventoryType | inventory.Type | Provenance only; not replayed on return. |
| assetId | asset.Id | Source asset identity. |
| snapshot | sharedsaga.AssetSnapshot | Full re-materialisation state: template, quantity, stats, expiration, cash id, pet block, etc. |

### MesoModel

One participant's escrowed meso for one room. `Amount` is the CONFIRMED
escrowed total — the sum of the stake deltas that have actually landed — and
is signed, transiently negative while resolution order permits an
intermediate below zero.

### MesoStakeModel

One in-flight `award_mesos` debit/credit against a participant's escrow row.
`Amount` is the absolute total the player typed for this stake; `Delta` is
the signed movement this stake submitted, and is the only safe basis for
refunding an orphaned stake (a teardown zeroes `MesoModel.Amount` while
leaving stakes armed).

### MesoRefundModel

Records what one `trade_unwind` took from a participant's escrowed meso, so a
failed unwind can put it back. Removed once the unwind reaches a terminal
state.

## Invariants

- `MesoModel.Amount == the sum of the deltas award_mesos actually moved` —
  never assigned from a stake's absolute target.
- More than one meso stake can be outstanding for one participant at once
  (the reference client permits retyping the box faster than a saga round
  trip); each stake resolves independently against its own row.
- An escrow item row is released (soft-deleted) on a normal settlement
  release, and hard-deleted only for a spurious accept; a released row can be
  restored (late compensation), a removed row cannot.
- `ItemEntity.ReturningAt`/`ReturningTxId` is a single-claimant latch: exactly
  one caller may win the right to submit a `trade_unwind` for a given row,
  because two independent code paths (a teardown's sweep and an orphaned
  stage's own terminal status) can both observe the same unclaimed row.
- `ClaimMesoForReturn` is destructive: it zeroes the row as the act of taking
  the amount, so the amount taken must be durably recorded
  (`MesoRefundModel`) in the same transaction before the unwind is
  submitted, or a failed unwind destroys the meso outright.
- Every write to an escrow row is paired with its Kafka ack (`ACCEPTED`,
  `RELEASED`, `RESTORED`, `REMOVED`, or `ERROR`) inside one transactional
  buffer, so the row and its ack cannot diverge.

## Processors

### Processor (`escrow` package)

Applies custody commands and acks them. `Accept`/`Release`/`Restore`/`Remove`
are the four saga-step operations (mirrors `CommandAcceptToTrade` /
`CommandReleaseFromTrade` / `CommandRestoreTradeEscrow` /
`CommandRemoveTradeEscrow`). `ClaimItemForReturn`, `UpsertMeso`,
`DeleteMeso`, `DischargeMeso`, `ClaimMesoForReturn`,
`ReleaseItemReturnClaims`, `RecordMesoRefund`, `RestoreMesoRefunds`,
`DiscardMesoRefunds`, `DeleteResolvedMeso`, `ArmMesoStake`,
`CommitMesoStake`, `AbandonMesoStake`, `MesoStakeById`,
`EffectiveMesoByOwner`, and `InFlightMesoDelta` are plain database operations
that are **not** saga steps and emit no Kafka ack — they maintain the durable
record the trade package's settlement, staging, and teardown paths read and
write.

# Ledger Domain

## Responsibility

Durable, immutable record of every settled trade (`ledger` package). Written
once, on terminal settlement success, and never updated or deleted.

## Core Models

### Item

One asset a side gave, as recorded at settlement time. `AssetId` and
`ReferenceId` are present only for identity-bearing assets (equips, pets,
cash); a plain stackable carries neither.

### SideModel

One participant's recorded contribution: `mesoStaged`/`mesoTax` describe what
this side gave, `mesoDelivered` what it received after the counterparty's
tax; `items` is the list of assets this side gave.

### Model

One settled trade: `transactionId` (the settlement saga's id, unique per
tenant — the write-side idempotency key), `field`, `roomType`, `settledAt`,
and two `SideModel`s.

## Invariants

- A ledger entry is written at most once per `(tenantId, transactionId)`; a
  duplicate settle for the same settlement saga produces no second row
  (unique index enforced at the database).
- Rows are never updated or deleted after creation.
- `Model.Sides()` ordering (by character id on read) is a determinism
  guarantee only, never a role guarantee: a caller must match a side by
  `CharacterId`, never by position.

## Processors

### Processor (`ledger` package)

`Record` writes one ledger entry (idempotent per transaction id).
`GetById`/`GetByTransactionId` read a single entry. `GetByCharacter`/
`GetPageByCharacterId` read every entry naming a character as either side
within an optional time window, paged in SQL.

# Settlement Domain

## Responsibility

Durable, transient record of a settlement saga that has been submitted but
whose terminal outcome has not yet been observed (`settlement` package). It
exists because the live trade room is process-local in-memory state while the
settlement saga survives an atlas-trades restart; without this record a
restart between submission and the terminal status would lose the room and
leave a trade that fully executed with no ledger row and no client-visible
outcome.

## Core Models

### Item

One escrowed asset carried into the settlement payload: `escrowId`
(the custody row released), `inventoryType`/`sourceSlot` (provenance),
`assetId`, `templateId`, `quantity`.

### SideModel

One participant's contribution at submission time, with the tax split
already resolved (frozen against a later tenant configuration change):
`position`, `characterId`, `characterName`, `mesoStaged`, `mesoTax`,
`mesoDelivered`, `items`.

### Model

One in-flight settlement: `id`, `tenantId`, `roomId`, `handle`, `roomType`,
`field`, `ownerId`, `visitorId`, `submittedAt`, two `SideModel`s. Denormalises
the room identity because the room itself is gone by the time a restart
requires this record to be read.

## Invariants

- Written in the same transaction that publishes the settlement saga command,
  so the two cannot diverge.
- Deleted (`Resolve`) exactly once its terminal status has been handled; the
  delete's rows-affected count is the arbiter between two concurrent
  deliveries of the same terminal status, so exactly one proceeds to emit the
  client-visible outcome.
- Not an audit record — the ledger is. This record exists only to bridge a
  restart between saga submission and the saga's terminal status.

## Processors

### Processor (`settlement` package)

`Submit` writes the record. `GetByTransactionId` reads it.
`Resolve` deletes it and reports whether this call won the delete race.
`Unresolved` (instance method, tenant-scoped) and the package-level
`Unresolved`/`allUnresolved` (no tenant in context, used at startup) list
every record still awaiting a terminal status.

# Configuration Domain

## Responsibility

Per-tenant trade configuration, fetched from atlas-tenants and cached
per-tenant (`configuration` package): the meso-tax tier table, the
staged-item cap, the minimum trading level, and the attestation timeout.

## Core Models

### Tier

One meso-tax band: `Threshold` (uint32) and `Rate` (float64, `[0, 1]`).
Tiers apply to amounts `>= Threshold`, with the first (highest) matching
tier winning; the table must be strictly descending by threshold.

### Model

Immutable per-tenant configuration: `taxEnabled`, `taxTiers`,
`maxStagedItems`, `minTradeLevel`, `attestationTimeout`.

## Invariants

- `ValidateTiers` enforces strictly descending thresholds and rates in
  `[0, 1]`; an invalid table is rejected loudly and never partially accepted.
- A `Model`'s tax-tier table is always valid: the only two ways to install one
  (`WithTaxTiers`, and `Extract` through it) fall back to the shipped default
  table on rejection or an empty table.
- The shipped default (`DefaultConfig`) is used whenever a tenant has no
  `trade-configs` resource, or its table fails validation: the service never
  hard-fails on a missing configuration and never silently disables trading.
- `Tax(m, amount)` computes `delivered = amount - floor(amount * rate)`; the
  difference (tax) is destroyed, credited to no one.

## Processors

### Registry (`configuration` package)

Lazy, per-tenant, process-wide cache (`GetRegistry`). `Get` resolves the
request's tenant from context and returns its cached configuration, fetching
and caching it from atlas-tenants on first access; any fetch miss or error
is cached as `DefaultConfig` so a resolution failure is never what fails a
trade.
