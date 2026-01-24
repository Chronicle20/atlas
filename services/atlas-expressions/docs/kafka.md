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
| expression | uint32 |

### EVENT_TOPIC_MAP_STATUS

Map status events. Only CHARACTER_EXIT type is handled.

| Field | Type |
|-------|------|
| transactionId | uuid.UUID |
| worldId | world.Id |
| channelId | channel.Id |
| mapId | map.Id |
| type | string |
| body | CharacterExit |

#### CharacterExit Body

| Field | Type |
|-------|------|
| characterId | uint32 |

## Topics Produced

### EVENT_TOPIC_EXPRESSION

Expression status events.

| Field | Type |
|-------|------|
| transactionId | uuid.UUID |
| characterId | uint32 |
| worldId | world.Id |
| channelId | channel.Id |
| mapId | map.Id |
| expression | uint32 |

## Message Types

| Type | Direction | Struct |
|------|-----------|--------|
| Command | Consumed | expression.Command |
| StatusEvent | Produced | expression.StatusEvent |
| StatusEvent[CharacterExit] | Consumed | map.StatusEvent[CharacterExit] |

## Transaction Semantics

All messages include a transactionId header for correlation. Tenant context is propagated via header parsers.
