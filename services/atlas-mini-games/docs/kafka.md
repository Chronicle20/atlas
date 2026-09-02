# Mini-Game Kafka Integration

Produced events are published via a transactional outbox (`atlas-outbox`): a command handler enqueues its status events into the outbox table inside the same database transaction as any persisted write (the `record.ApplyResult` upsert on game end), and a background drainer publishes them to Kafka once the transaction commits. All consumers use consumer group `Mini Game Service` and parse the span, tenant, and env headers.

## Topics Consumed

### COMMAND_TOPIC_MINI_GAME

Mini-game lifecycle and gameplay commands.

**Consumer Group:** Mini Game Service

**Message Types:**

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "CREATE",
  "body": {}
}
```

| Type | Body | Notes |
|------|------|-------|
| CREATE | `CreateCommandBody{roomType, title, private, password, pieceType}` | Opens a room |
| VISIT | `VisitCommandBody{roomId, password}` | Seats a visitor |
| LEAVE | `EmptyCommandBody{}` | Member leaves |
| CHAT | `ChatCommandBody{message}` | Rebroadcasts a chat message |
| READY | `EmptyCommandBody{}` | Visitor readies |
| UNREADY | `EmptyCommandBody{}` | Visitor un-readies |
| START | `EmptyCommandBody{}` | Owner starts the game |
| MOVE_STONE | `MoveStoneCommandBody{x, y, stoneType}` | Omok stone placement |
| FLIP_CARD | `FlipCardCommandBody{first, cardIndex}` | Match Cards flip |
| REQUEST_TIE | `EmptyCommandBody{}` | Proposes a tie |
| ANSWER_TIE | `AnswerCommandBody{accept}` | Answers a tie proposal |
| GIVE_UP | `EmptyCommandBody{}` | Forfeits the running game |
| REQUEST_RETREAT | `EmptyCommandBody{}` | Omok undo request |
| ANSWER_RETREAT | `AnswerCommandBody{accept}` | Answers an undo request |
| EXPEL | `EmptyCommandBody{}` | Owner ejects the visitor |
| SKIP | `EmptyCommandBody{}` | Yields the current turn |
| EXIT_AFTER_GAME | `EmptyCommandBody{}` | Sets the exit-after-game flag |
| CANCEL_EXIT_AFTER_GAME | `EmptyCommandBody{}` | Clears the exit-after-game flag |

### EVENT_TOPIC_SESSION_STATUS

Session status events. Only `DESTROYED` is consumed.

**Consumer Group:** Mini Game Service

**Message Types:**

#### DESTROYED

Tears down whatever mini-game room the character occupies on session destroy (disconnect/kick).

```json
{
  "sessionId": "uuid",
  "accountId": 0,
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "issuer": "",
  "type": "DESTROYED"
}
```

### EVENT_TOPIC_CHARACTER_STATUS

Character status events. `LOGOUT`, `MAP_CHANGED`, and `CHANNEL_CHANGED` are consumed.

**Consumer Group:** Mini Game Service

**Message Types:**

#### LOGOUT

Tears down the character's mini-game membership.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "LOGOUT",
  "body": {
    "channelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

#### MAP_CHANGED

Tears down the character's mini-game membership (a room is field-scoped).

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "MAP_CHANGED",
  "body": {
    "channelId": 0,
    "oldMapId": 0,
    "oldInstance": "uuid",
    "targetMapId": 0,
    "targetInstance": "uuid",
    "targetPortalId": 0
  }
}
```

#### CHANNEL_CHANGED

Tears down the character's mini-game membership.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "CHANNEL_CHANGED",
  "body": {
    "channelId": 0,
    "oldChannelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

## Topics Produced

### EVENT_TOPIC_MINI_GAME_STATUS

Mini-game status events. Each message key is the room field's map id. Every event carries `roomId`/`ownerId`/`visitorId` (room identity) and `characterId` (the acting character).

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "roomId": 0,
  "ownerId": 0,
  "visitorId": 0,
  "characterId": 0,
  "type": "CREATED",
  "body": {}
}
```

| Type | Body | Notes |
|------|------|-------|
| CREATED | `CreatedEventBody{roomType, title, pieceType, ownerRecord}` | Room opened |
| CREATE_ERROR | `ErrorEventBody{code}` | `code` is an enterError KEY (`NOT_WHEN_DEAD`, `CANNOT_START_GAME_HERE`, `CANNOT_OPEN_MINI_ROOM_HERE`, `UNABLE`) |
| ENTERED | `EnteredEventBody{slot, roomType, title, pieceType, ownerRecord, visitorRecord, ownerScore, visitorScore}` | Visitor seated |
| ENTER_ERROR | `ErrorEventBody{code}` | `code` is an enterError KEY (`UNABLE`, `ROOM_CLOSED`, `FULL`, `INCORRECT_PASSWORD`, `NOT_WHEN_DEAD`, `CANNOT_OPEN_MINI_ROOM_HERE`) |
| LEFT | `LeftEventBody{slot, status}` | `status` is a leaveReason KEY (`MINIGAME_LEFT`, `MINIGAME_EXPELLED`) |
| ROOM_CLOSED | `RoomClosedEventBody{visitorStatus}` | `visitorStatus` is a leaveReason KEY (`MINIGAME_CLOSED`); owner left, room removed |
| CHAT | `ChatEventBody{slot, message}` | Rebroadcast chat |
| READY | `EmptyEventBody{}` | Visitor readied |
| UNREADY | `EmptyEventBody{}` | Visitor un-readied |
| STARTED | `StartedEventBody{roomType, firstMover, deck}` | `deck` is empty for Omok |
| STONE_PLACED | `StonePlacedEventBody{x, y, stoneType}` | Omok stone placement |
| PUT_STONE_ERROR | `PutStoneErrorEventBody{code}` | `code` is a putStoneError KEY (`DOUBLE_THREE`, `CANNOT_PLACE`); targeted at the acting character only |
| CARD_FLIPPED | `CardFlippedEventBody{secondFlip, slot, firstSlot, resultType}` | Match Cards flip |
| TIE_REQUESTED | `EmptyEventBody{}` | Targeted at the opponent |
| TIE_ANSWERED | `AnswerEventBody{accept}` | Decline only; accept resolves via `GAME_ENDED` |
| RETREAT_REQUESTED | `EmptyEventBody{}` | Omok undo request; targeted at the opponent |
| RETREAT_ANSWERED | `AnswerEventBody{accept}` | Omok undo answer |
| SKIPPED | `SkippedEventBody{who}` | `who` is the next-mover slot |
| GAME_ENDED | `GameEndedEventBody{resultType, winnerSlot, ownerRecord, visitorRecord, ownerScore, visitorScore}` | `resultType` is a resultType KEY (`WIN`, `TIE`, `FORFEIT`); `winnerSlot` is ignored for a tie |
| BALLOON_UPDATED | `BalloonEventBody{remove, roomType, title, hasPassword, pieceType, occupancy, inProgress}` | Field-entry balloon state; `remove` tears the balloon down |

`RecordBody{gameType, wins, ties, losses}` is the shared shape used for `ownerRecord`/`visitorRecord` fields above.

## Transaction Semantics

- Every produced status event is enqueued into the transactional outbox inside the same database transaction as any persisted write the command performs (`record.ApplyResult` on game end), so the write and its events commit and publish atomically.
- Each produced status event is keyed by the room field's map id.
- Consumed commands and character/session status events carry span, tenant, and env headers (`SpanHeaderParser`, `TenantHeaderParser`, `EnvHeaderParser`).
- All command and status-event handlers are registered with persistent configuration (`message.PersistentConfig`).
