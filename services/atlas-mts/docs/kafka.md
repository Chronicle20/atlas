## Topics Consumed

### `COMMAND_TOPIC_MTS` (env: `mts.EnvCommandTopic`) — command

The high-level MTS command topic. Command types dispatched by registered
handlers (`kafka/consumer/mts/consumer.go`):

| Command type | Message struct |
|---|---|
| `CANCEL_LISTING` | `mts.Command[mts.CancelListingCommandBody]` |
| `CREATE_LISTING` | `mts.Command[mts.CreateListingCommandBody]` |
| `BUY` | `mts.Command[mts.BuyCommandBody]` |
| `PLACE_BID` | `mts.Command[mts.PlaceBidCommandBody]` |
| `TAKE_HOME` | `mts.Command[mts.TakeHomeCommandBody]` |
| `REGISTER_WISH` | `mts.Command[mts.RegisterWishCommandBody]` |
| `REMOVE_WISH` | `mts.Command[mts.RemoveWishCommandBody]` |

`EXPIRE_LISTING` (`mts.CommandExpireListing`) is declared in the message
vocabulary but has no registered handler — auction/listing expiry is driven
by the service's internal periodic sweep task (`task/periodic.go`), not by
a consumed Kafka command.

Consumer registration applies `consumer.SetHeaderParsers(SpanHeaderParser,
TenantHeaderParser)`.

### `COMMAND_TOPIC_MTS_CUSTODY` (env: `custody.EnvCommandTopic`) — command

The MTS custody command topic, dispatched by the saga orchestrator.
Command types dispatched by registered handlers
(`kafka/consumer/custody/consumer.go`):

| Command type | Message struct |
|---|---|
| `ACCEPT_TO_MTS_LISTING` | `custody.Command[custody.AcceptToMtsListingCommandBody]` |
| `RELEASE_FROM_MTS_HOLDING` | `custody.Command[custody.ReleaseFromMtsHoldingCommandBody]` |
| `RESTORE_MTS_HOLDING` | `custody.Command[custody.RestoreMtsHoldingCommandBody]` |
| `MTS_MOVE_LISTING_TO_HOLDING` | `custody.Command[custody.MtsMoveListingToHoldingCommandBody]` |
| `REMOVE_MTS_LISTING` | `custody.Command[custody.RemoveMtsListingCommandBody]` |
| `RESTORE_LISTING_FROM_HOLDING` | `custody.Command[custody.RestoreListingFromHoldingCommandBody]` |

Consumer registration applies `consumer.SetHeaderParsers(SpanHeaderParser,
TenantHeaderParser)`.

## Topics Produced

### `EVENT_TOPIC_MTS_STATUS` (env: `mts.EnvStatusEventTopic`) — event

The high-level MTS status/event topic. Message types emitted by producer
functions in `kafka/producer/mts/producer.go`:

| Status event type | Message struct | Emitted from |
|---|---|---|
| `LISTING_CREATED` | `mts.StatusEvent[mts.StatusEventListingCreatedBody]` | custody `handleAcceptToMtsListing` |
| `LISTING_CANCELLED` | `mts.StatusEvent[mts.StatusEventListingCancelledBody]` | mts `handleCancelListing`; custody `handleMtsMoveListingToHolding` (per released sibling offer) |
| `BID_PLACED` | `mts.StatusEvent[mts.StatusEventBidPlacedBody]` | mts `handlePlaceBid` |
| `OUTBID` | `mts.StatusEvent[mts.StatusEventOutbidBody]` | mts `handlePlaceBid` (on an outbid) |
| `LISTING_SOLD` | `mts.StatusEvent[mts.StatusEventListingSoldBody]` | custody `handleMtsMoveListingToHolding` |
| `ITEM_TAKEN_HOME` | `mts.StatusEvent[mts.StatusEventItemTakenHomeBody]` | custody `handleReleaseFromMtsHolding` |
| `WISH_ADDED` | `mts.StatusEvent[mts.StatusEventWishAddedBody]` | mts `handleRegisterWish` |
| `WISH_REMOVED` | `mts.StatusEvent[mts.StatusEventWishRemovedBody]` | mts `handleRemoveWish` |
| `LISTING_CREATE_FAILED` | `mts.StatusEvent[mts.StatusEventListingCreateFailedBody]` | mts `handleCreateListing` |
| `LISTING_CANCEL_FAILED` | `mts.StatusEvent[mts.StatusEventListingCancelFailedBody]` | mts `handleCancelListing` |
| `BUY_FAILED` | `mts.StatusEvent[mts.StatusEventBuyFailedBody]` | mts `handleBuy` |
| `BID_FAILED` | `mts.StatusEvent[mts.StatusEventBidFailedBody]` | mts `handlePlaceBid` |
| `TAKE_HOME_FAILED` | `mts.StatusEvent[mts.StatusEventTakeHomeFailedBody]` | mts `handleTakeHome` |

`LISTING_EXPIRED` (`ListingExpiredStatusEventProvider`,
`mts.StatusEvent[mts.StatusEventListingExpiredBody]`) and
`ITEM_MOVED_TO_HOLDING` (`ItemMovedToHoldingStatusEventProvider`,
`mts.StatusEvent[mts.StatusEventItemMovedToHoldingBody]`) have producer
functions defined but no call site in the current codebase emits them.

### `EVENT_TOPIC_MTS_CUSTODY_STATUS` (env: `custody.EnvStatusEventTopic`) — event

The MTS custody ack/status topic. Message types emitted by producer
functions in `kafka/producer/custody/producer.go`:

| Status event type | Message struct |
|---|---|
| `ACCEPTED` | `custody.StatusEvent[custody.StatusEventAcceptedBody]` |
| `RELEASED` | `custody.StatusEvent[custody.StatusEventReleasedBody]` |
| `MOVED` | `custody.StatusEvent[custody.StatusEventMovedBody]` |
| `RESTORED` | `custody.StatusEvent[custody.StatusEventRestoredBody]` |
| `ERROR` | `custody.StatusEvent[custody.StatusEventErrorBody]` |

### `COMMAND_TOPIC_SAGA` (env: `saga.EnvCommandTopic`) — command

The shared saga-orchestrator command topic. atlas-mts emits `saga.Saga`
messages (aliased from the shared `atlas-saga` library) of type
`MtsOperation`, composed of one or more steps carrying these payload types:
`AwardMesosPayload`, `TransferToMtsPayload`, `WithdrawFromMtsPayload`,
`MtsSettlePurchasePayload`, `MtsBidEscrowPayload`, `AwardCurrencyPayload`,
`MtsMoveListingToHoldingPayload`.

## Message Types

- `mts.Command[E]` / `mts.StatusEvent[E]` — the generic MTS envelopes
  (`TransactionId`, `Type`, `Body`), defined in
  `kafka/message/mts/kafka.go`.
- `custody.Command[E]` / `custody.StatusEvent[E]` — the generic custody
  envelopes, defined in `kafka/message/custody/kafka.go`.
- `saga.Saga` — the shared saga envelope (re-exported from
  `github.com/Chronicle20/atlas/libs/atlas-saga` in `saga/model.go`),
  composed of `saga.Step` entries.

Required headers: both the `mts` and `custody` command consumers register
with `SpanHeaderParser` and `TenantHeaderParser`, i.e. inbound commands
must carry span and tenant headers. Outbound `mts`/`custody` status events
are partition-keyed by the first 4 bytes of the transaction id
(`keyFor`, `producer.CreateKey`); saga commands are keyed by the full
transaction id string (`saga.CreateCommandProvider`), so all messages for
one transaction/saga land on the same partition in order.

## Transaction Semantics

- Local DB writes and their corresponding status-event emission are
  committed atomically via the transactional outbox: command handlers wrap
  the domain-processor call and the event `Put(...)` calls in one
  `database.ExecuteTransaction`, using `outbox.EmitProvider(l, ctx, tx)` so
  the Kafka message is enqueued as an outbox row on the same transaction as
  the domain row mutation. A background drainer
  (`outboxlib.NewDrainer`, started in `main.go`) publishes queued rows to
  Kafka after commit; its leadership is gated by a Postgres advisory lock so
  multiple replicas can run safely.
- `listing.Cancel` and `listing.PlaceBid` additionally route their
  downstream saga emission (escrow hold/release) through
  `saga.OutboxEmitter` when invoked inside an outer transaction (injected
  via `listing.WithSagaEmitter`), so the saga command is enqueued on the
  same transaction as the local write rather than published directly —
  keeping the local write, its status event, and the outgoing saga command
  atomic.
- Direct (non-outbox) `producer.Provider` emission is used for:
  failure-notice events (`LISTING_CREATE_FAILED`, `LISTING_CANCEL_FAILED`,
  `BUY_FAILED`, `BID_FAILED`, `TAKE_HOME_FAILED` — there is no local write
  to bind them to), the custody `ERROR` ack, and best-effort post-commit
  side effects intentionally outside the settling transaction (the
  sibling-offer `LISTING_CANCELLED` notices emitted from
  `handleMtsMoveListingToHolding`, and `ReleaseHighBidEscrow`'s saga
  emission).
- Sagas are of type `MtsOperation` and are built with an explicit,
  step-count-scaled timeout (a base timeout plus a per-step budget) rather
  than a flat timeout, so the orchestrator's serial per-step Kafka
  round-trips do not trigger a premature saga rollback.
