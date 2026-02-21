# Kafka Integration

## Topics Consumed

### COMMAND_TOPIC_EXPRESSION

Expression change commands.

| Field | Type |
|-------|------|
| transactionId | uuid.UUID |
| characterId | uint32 |
| worldId | world.Id |
| channelId | channel.Id |
| mapId | map.Id |
| instance | uuid.UUID |
| expression | uint32 |

Consumer group: `Expression Service`. Header parsers: span, tenant.

### EVENT_TOPIC_MAP_STATUS

Map status events. Only `CHARACTER_EXIT` type is handled.

| Field | Type |
|-------|------|
| transactionId | uuid.UUID |
| worldId | world.Id |
| channelId | channel.Id |
| mapId | map.Id |
| instance | uuid.UUID |
| type | string |
| body | CharacterExit |

#### CharacterExit Body

| Field | Type |
|-------|------|
| characterId | uint32 |

Consumer group: `Expression Service`. Header parsers: span, tenant.

On CHARACTER_EXIT, the expression for the exiting character is cleared.

## Topics Produced

### EVENT_TOPIC_EXPRESSION

Expression status events. Emitted on expression change and on expiration revert.

| Field | Type |
|-------|------|
| transactionId | uuid.UUID |
| characterId | uint32 |
| worldId | world.Id |
| channelId | channel.Id |
| mapId | map.Id |
| instance | uuid.UUID |
| expression | uint32 |

Partition key: characterId.

## Message Types

| Type | Direction | Struct |
|------|-----------|--------|
| Command | Consumed | expression.Command |
| StatusEvent | Produced | expression.StatusEvent |
| StatusEvent[CharacterExit] | Consumed | map.StatusEvent[CharacterExit] |

## Transaction Semantics

All messages include a transactionId for correlation. Tenant context is propagated via header parsers. The RevertTask generates a new transactionId for each expired expression revert.
