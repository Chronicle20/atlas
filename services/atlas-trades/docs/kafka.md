# Trade Kafka Integration

## Topics Consumed

### COMMAND_TOPIC_TRADE

Trade room commands from atlas-channel. Consumer group: `Trade Service`.
Shared by every handler on this topic; each handler discriminates on
`Command.Type` before touching `Body`.

**Message Types:**

| Type | Body | Description |
|------|------|-------------|
| `CREATE_ROOM` | `CreateRoomCommandBody{roomType}` | Opens a solo room. |
| `INVITE` | `InviteCommandBody{targetCharacterId}` | Offers the caller's room to a target character. |
| `DECLINE_INVITE` | `DeclineInviteCommandBody{serialNumber, errorCode}` | Client-side decline of an offered invite; resolved to the originator's room by the wire serial. |
| `ENTER_ROOM` | `EnterRoomCommandBody{handle, roomType}` | Seats the caller in the room the handle names. |
| `PUT_ITEM` | `PutItemCommandBody{inventoryType, slot, quantity, targetSlot}` | Stages one item. |
| `ADD_MESO` | `AddMesoCommandBody{amount}` | Sets the caller's staged meso to an absolute total. |
| `CONFIRM` | `ConfirmCommandBody{entries}` | Records one side pressing Trade, with its CRC list. |
| `TRANSACTION` | `TransactionCommandBody{entries}` | Records the client's automatic attestation reply to the mode-17 prompt. |
| `CANCEL` | `CancelCommandBody{}` | Client closed its trade dialog; tears down the caller's room. |
| `CHAT` | `ChatCommandBody{message}` | Relays one line of trade-room chat. |

### COMMAND_TOPIC_TRADE_CUSTODY

Escrow custody commands from atlas-saga-orchestrator (the trade limb of the
accept/release custody family). Consumer group: `Trade Service`. atlas-trades
owns this contract; atlas-saga-orchestrator carries a mirror. Shared by every
handler on this topic; each handler discriminates on `Command.Type` before
touching `Body`.

**Message Types:**

| Type | Body | Description |
|------|------|-------------|
| `ACCEPT_TO_TRADE` | `AcceptToTradeCommandBody{escrowId, roomId, ownerId, tradeSlot, sourceInventoryType, assetId, snapshot}` | Writes the escrow row for an item that has already left its owner's compartment. |
| `RELEASE_FROM_TRADE` | `ReleaseFromTradeCommandBody{escrowId}` | Soft-deletes the escrow row. |
| `RESTORE_TRADE_ESCROW` | `RestoreTradeEscrowCommandBody{escrowId}` | Un-soft-deletes the row (late-compensation inverse of release). |
| `REMOVE_TRADE_ESCROW` | `RemoveTradeEscrowCommandBody{escrowId}` | Hard-deletes a spurious row (late-compensation inverse of accept). |

### EVENT_TOPIC_INVITE_STATUS

Answer half of the trade invite atlas-trades issues on `COMMAND_TOPIC_INVITE`.
Consumer group: `Trade Service`. Carries every invite family; each handler
filters on both the status type and `InviteType == invite.TypeTrade`.

**Message Types:**

| Type | Body | Description |
|------|------|-------------|
| `ACCEPTED` | `AcceptedEventBody{originatorId, targetId}` | Seats the acceptor via `EnterRoom`. |
| `REJECTED` | `RejectedEventBody{originatorId, targetId}` | Tears the originator's pending room down (covers both an explicit decline and an atlas-invites-side expiry). |

### EVENT_TOPIC_SAGA_STATUS

Terminal outcomes of every saga in the deployment. Consumer group:
`Trade Service`. Carries every saga; routing to one of this service's four id
spaces (item stage, meso stake, unwind, settlement) is by ownership probe, in
ascending cost, and the first claim wins.

**Message Types:**

| Type | Body | Description |
|------|------|-------------|
| `COMPLETED` | `StatusEventCompletedBody{sagaType, results}` | Probed in order: `StageSucceeded` (escrow row id) -> `MesoStageSucceeded` (stake id) -> `UnwindSucceeded` (unwind transaction id) -> `SettlementSucceeded` (settlement transaction id). |
| `FAILED` | `StatusEventFailedBody{reason, failedStep, characterId, accountId, sagaType, errorCode, mtsKind}` | Probed in order: `StageFailed` -> `MesoStageFailed` -> `UnwindFailed` -> `SettlementFailed`. `characterId` names the failed expanded step's character, never a role; both trade participants are resolved from the transaction id. |

### EVENT_TOPIC_CHARACTER_STATUS

Character status events used to tear a character out of its trade room.
Consumer group: `Trade Service`. Carries every character-status family; each
handler discriminates on `type` before its body is used.

**Message Types:**

| Type | Body | Description |
|------|------|-------------|
| `LOGOUT` | `StatusEventLogoutBody{channelId, mapId, instance}` | Tears down with reason `TRADE_CANCELLED`. |
| `MAP_CHANGED` | `StatusEventMapChangedBody{channelId, oldMapId, oldInstance, targetMapId, targetInstance, targetPortalId, useTargetPosition, targetX, targetY}` | Tears down with reason `TRADE_DIFFERENT_MAP`. |
| `CHANNEL_CHANGED` | `ChangeChannelEventLoginBody{channelId, oldChannelId, mapId, instance}` | Tears down with reason `TRADE_DIFFERENT_MAP`. |

### EVENT_TOPIC_SESSION_STATUS

Session lifecycle events. Consumer group: `Trade Service`.

**Message Types:**

| Type | Body | Description |
|------|------|-------------|
| `DESTROYED` | `StatusEvent{sessionId, accountId, characterId, worldId, channelId, issuer, type}` | Tears down the character's room (reason `TRADE_CANCELLED`) when the session died without a clean logout. Skipped when `characterId == 0` (no character was ever selected). |

## Topics Produced

### EVENT_TOPIC_TRADE_STATUS

Trade room status events to atlas-channel, keyed by the room's map id. Every
event carries both participants (`ownerId`, `visitorId`) and the acting
`characterId` so atlas-channel can address the room without a lookup.

**Message Types:**

| Type | Body | Description |
|------|------|-------------|
| `ROOM_CREATED` | `RoomCreatedEventBody{position}` | A solo room opened. |
| `INVITE_SENT` | `InviteSentEventBody{targetCharacterId, inviterName}` | An invite went out. |
| `INVITE_REJECTED` | `InviteRejectedEventBody{code, targetName}` | An invite could not be sent or was refused; `code` is an `inviteResult` KEY (`CANNOT_FIND_CHARACTER`, `BUSY`). |
| `PARTICIPANT_ENTERED` | `ParticipantEnteredEventBody{characterId, name, position}` | The visitor took seat 1. |
| `ITEM_STAGED` | `ItemStagedEventBody{position, tradeSlot, assetId, snapshot}` | An item's staging saga completed; carries the full asset snapshot from the escrow row. |
| `ITEM_REFUSED` | `ItemRefusedEventBody{position, tradeSlot}` | A stage was refused or failed before reaching escrow; unlocks the staging client's dialog. |
| `MESO_STAGED` | `MesoStagedEventBody{position, amount}` | A meso stake resolved (or a no-op re-stage); carries the absolute staged total. |
| `MESO_REFUSED` | `MesoRefusedEventBody{position, lastValidAmount}` | A meso stake was refused or failed; carries the authoritative re-echo that snaps the client's dialog back and clears its lock. |
| `PARTICIPANT_CONFIRMED` | `ParticipantConfirmedEventBody{position}` | One side pressed Trade. |
| `ATTESTATION_REQUESTED` | `AttestationRequestedEventBody{}` | Both sides confirmed; prompts both clients for CRC attestation (mode 17). |
| `SETTLED` | `SettledEventBody{ledgerEntryId}` | The settlement saga succeeded; carries the ledger row's id. |
| `CANCELLED` | `CancelledEventBody{reason}` | The room ended without settling; `reason` is a `leaveReason` KEY. |
| `ERROR` | `ErrorEventBody{code}` | A command was refused before or independent of a room transition; `code` is an `enterError` KEY. |
| `CHAT` | `ChatEventBody{position, message}` | Relayed trade-room chat. |

### EVENT_TOPIC_TRADE_CUSTODY_STATUS

Escrow custody acks to atlas-saga-orchestrator, keyed by the escrow row id.

**Message Types:**

| Type | Body | Description |
|------|------|-------------|
| `ACCEPTED` | `StatusEventAcceptedBody{escrowId}` | Escrow row created. |
| `RELEASED` | `StatusEventReleasedBody{escrowId}` | Escrow row soft-deleted. |
| `RESTORED` | `StatusEventRestoredBody{escrowId}` | Escrow row un-soft-deleted. |
| `REMOVED` | `StatusEventRemovedBody{escrowId}` | Escrow row hard-deleted. Deliberately not routed into step completion by the orchestrator — both restore and remove are late compensating inverses, not step advances. |
| `ERROR` | `StatusEventErrorBody{escrowId, error}` | The custody write failed; carries the reason. |

### COMMAND_TOPIC_INVITE

Trade-invite commands to atlas-invites.

**Message Types:**

| Type | Body | Description |
|------|------|-------------|
| `CREATE` | `CreateCommandBody{originatorId, targetId, referenceId}` | Offers the room; `referenceId` is the room's uint32 wire handle. |
| `REJECT` | `RejectCommandBody{targetId, originatorId}` | Retires an outstanding offer after a client-originated decline. |

### COMMAND_TOPIC_SAGA

Saga commands to atlas-saga-orchestrator. atlas-trades never enumerates
concrete saga steps; it submits single-step composites the orchestrator
expands.

| Saga Type | Step | Payload | Submitted By |
|-----------|------|---------|--------------|
| `TradeTransaction` | `trade_settlement` (`TradeSettlement`) | `sharedsaga.TradeSettlementPayload` | `settle` on both-attested. |
| `TradeStaging` | `transfer_to_trade` (`TransferToTrade`) | `sharedsaga.TransferToTradePayload` | `PutItem`, keyed by the escrow row id. |
| `TradeStaging` | `stage_meso` (`AwardMesos`) | `sharedsaga.AwardMesosPayload` | `AddMeso`, keyed by a fresh stake id. |
| `TradeTransaction` | `trade_unwind` (`TradeUnwind`) | `sharedsaga.TradeUnwindPayload` | Teardown, failed settlement, orphaned-stage return, and startup reconciliation. |

## Transaction Semantics

- Every command handler runs inside one database transaction
  (`ProcessorImpl.emit`): the handler's durable writes, its registry mutation
  (in-memory, not covered by rollback), and every Kafka message it buffers
  (`message.Buffer`) are enqueued into the transactional outbox
  (`libs/atlas-outbox`) in that same transaction. The outbox drainer
  (booted in `main.go`, leadership gated by a Postgres advisory lock) then
  publishes to Kafka.
- `EVENT_TOPIC_TRADE_STATUS` and `EVENT_TOPIC_TRADE_CUSTODY_STATUS` messages
  are keyed as noted above so a room's or a row's own event stream stays
  ordered; `COMMAND_TOPIC_SAGA` commands are keyed by the saga's transaction
  id.
- Every consumer on every topic registers `SpanHeaderParser`,
  `TenantHeaderParser`, and `EnvHeaderParser`, and every handler is wired
  with `message.PersistentConfig`.
- The settlement saga's transaction id is simultaneously the saga's identity,
  the durable settlement record's key, and the ledger entry's idempotency
  key, so a redelivered terminal status (`SettlementSucceeded`/
  `SettlementFailed`) after the settlement record has already been resolved
  is a no-op.
- A staging saga's transaction id is the escrow row id it created; a
  meso-stake saga's transaction id is the stake id; an unwind saga's
  transaction id is its own, unrelated to the room. `handleSagaCompleted`/
  `handleSagaFailed` probe all four id spaces in a fixed order because they
  are disjoint but the envelope carries no discriminator.
