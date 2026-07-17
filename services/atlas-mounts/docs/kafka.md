# Kafka Integration

## Topics Consumed

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| EVENT_TOPIC_CHARACTER_BUFF_STATUS | Event | Character buff status events |
| EVENT_TOPIC_TAMING_MOB_FOOD | Event | Taming-mob food (feed) events |
| EVENT_TOPIC_CHARACTER_STATUS | Event | Character status events |

## Topics Produced

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| EVENT_TOPIC_MOUNT_STATUS | Event | Mount status events |

## Message Types

### Character Buff Status Event (Consumed)

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "APPLIED|EXPIRED",
  "body": {}
}
```

The APPLIED body carries `fromId`, `sourceId`, `level`, `duration`, `changes`, `createdAt`, `expiresAt`. The EXPIRED body carries `sourceId`, `level`, `duration`, `changes`, `createdAt`, `expiresAt`. Each entry in `changes` is a `StatChange` with `type` and `amount`.

| Type | Behavior |
|------|----------|
| APPLIED | Processed only when `changes` contains a `MONSTER_RIDING` stat change. `sourceId` is the skill id; the `MONSTER_RIDING` change `amount` is the vehicle item id. If the skill is a tamed-mount skill, the character is added to the active-mount registry (world/skill/vehicle); otherwise (skill-only mount) no registry entry is created. In both cases a SET event is emitted. |
| EXPIRED | Processed only when `changes` contains a `MONSTER_RIDING` stat change. The character's active-mount registry entry is removed (a no-op when none exists). |

### Taming-Mob Food Event (Consumed)

```json
{
  "worldId": 0,
  "characterId": 12345,
  "itemId": 2260000,
  "tirednessHeal": 30
}
```

Applies the feed math for the character's mount via `ApplyFeedAndEmit`, using `tirednessHeal` as the heal maximum, and emits a FEED event.

### Character Status Event (Consumed)

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 12345,
  "type": "LOGIN|LOGOUT",
  "body": {}
}
```

| Type | Behavior |
|------|----------|
| LOGOUT | Removes the character's active-mount registry entry (a no-op when none exists) |
| LOGIN | No-op |

### Mount Status Event (Produced)

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "SET|TICK|FEED",
  "body": {
    "level": 1,
    "exp": 0,
    "tiredness": 0,
    "levelUp": false,
    "tooTired": false
  }
}
```

| Type | Body Fields | Description |
|------|-------------|-------------|
| SET | level, exp, tiredness, levelUp (false), tooTired (false) | Current progression on mount activation |
| TICK | level, exp, tiredness, levelUp (false), tooTired | Progression after a tiredness tick |
| FEED | level, exp, tiredness, levelUp, tooTired (false) | Progression after a feed |

The message key is the character identifier.

## Transaction Semantics

- TICK and FEED state mutations are wrapped in a database transaction; the Kafka message is buffered and emitted only after the transaction commits.
- Messages are buffered via a `message.Buffer` and emitted atomically after the buffered work completes.
- Emission is via the transactional outbox (`atlas-outbox` library, `outbox.EmitProvider`): buffered messages are written to the outbox table within the same database transaction as the state change, then published to Kafka asynchronously by a background drainer. The drainer runs leader-elected via a Postgres advisory lock (`main.go`).
- SET events (buff-applied handling) are emitted outside the TICK/FEED transaction, via a separate outbox-backed transaction that only writes the event (no state mutation).

## Required Headers

- Span header (tracing)
- Tenant header (multi-tenancy)
