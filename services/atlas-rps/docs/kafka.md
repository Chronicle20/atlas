# RPS Kafka Integration

The command consumer uses the `RPS Service` consumer group and parses the span, tenant, and env headers.

## Topics Consumed

### COMMAND_TOPIC_RPS

RPS commands acting on a character's in-progress session.

**Consumer Group:** RPS Service

**Message Types:**

#### BEGIN

Opens the first round of an already-open (`StatusOpen`) session.

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "BEGIN",
  "body": {}
}
```

#### SELECT

Submits the player's throw for the current round.

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "SELECT",
  "body": {
    "throw": 0
  }
}
```

#### CONTINUE

Advances a session already at `StatusAwaitingDecision` to the next round.

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "CONTINUE",
  "body": {}
}
```

#### RETRY

Restarts a lost game, re-charging the entry fee.

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "RETRY",
  "body": {}
}
```

#### COLLECT

Ends the session, paying the resolved prize if the session is at `StatusAwaitingDecision`.

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "COLLECT",
  "body": {}
}
```

#### QUIT

Ends the session with no payout.

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "QUIT",
  "body": {}
}
```

## Topics Produced

### EVENT_TOPIC_RPS

RPS session events. Each message key is the character id.

**Message Types:**

#### GAME_OPENED

Emitted when a new session is opened for a character at an NPC.

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "GAME_OPENED",
  "body": {
    "npcId": 0,
    "ante": 0
  }
}
```

#### ROUND_STARTED

Emitted when a round opens for the player's throw (on `BEGIN`, `CONTINUE`, and `RETRY`).

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "ROUND_STARTED",
  "body": {
    "rung": 0
  }
}
```

#### ROUND_RESULT

Emitted after a round is adjudicated, carrying the opponent's throw, the outcome, the resulting rung, and any prize resolved at that rung.

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "ROUND_RESULT",
  "body": {
    "opponentThrow": 0,
    "outcome": 0,
    "rung": 0,
    "prize": {
      "itemId": 0,
      "quantity": 0,
      "meso": 0
    }
  }
}
```

#### GAME_ENDED

Emitted when a session terminates. `grantedPrize` is present only when `reason` is `collected`. `reason` is one of `collected`, `quit`, `disconnected`.

```json
{
  "characterId": 0,
  "worldId": 0,
  "channelId": 0,
  "type": "GAME_ENDED",
  "body": {
    "reason": "collected",
    "grantedPrize": {
      "itemId": 0,
      "quantity": 0,
      "meso": 0
    }
  }
}
```

### COMMAND_TOPIC_SAGA

atlas-saga-orchestrator commands. atlas-rps is a producer only on this topic (it does not consume the saga status event topic). Each message key is the saga's transaction id, and the message value is a `sharedsaga.Saga` (`libs/atlas-saga`).

Sagas submitted:
- A payout saga (`InventoryTransaction`) on `Collect`, with an `award_mesos` step when the resolved prize's meso is positive and/or an `award_asset` step when the resolved prize's item id and quantity are both positive.
- A fee-deduction saga (`InventoryTransaction`, single `deduct_mesos` step, negative meso amount) on `Retry`, when the ladder's `EntryCostMeso` is positive.
- A consolation payout saga (`InventoryTransaction`, single `award_mesos` step) on `Retry` or on `Collect` of a post-loss session, when the ladder's `ConsolationMeso` is non-zero and the session's rung is 0.

## Transaction Semantics

- Each `EVENT_TOPIC_RPS` message is keyed by the character id.
- Each `COMMAND_TOPIC_SAGA` message is keyed by the saga's transaction id (a newly generated UUID per saga).
- The consumed `COMMAND_TOPIC_RPS` messages carry span, tenant, and env headers (`SpanHeaderParser`, `TenantHeaderParser`, `EnvHeaderParser`).
- All command handlers are registered with persistent configuration.
- Session events buffered during a state transition are emitted atomically after the transition succeeds (`message.EmitWithResult`); a saga submission failure on the `Collect`/`Retry` payout or fee-deduction path aborts the transition before any event is buffered/emitted.
