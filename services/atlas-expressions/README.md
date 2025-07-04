# atlas-expressions
Mushroom game expressions Service

## Overview

A service that manages character expressions in the game. It provides functionality to change and clear expressions for characters, and emits events when expressions are changed.

## Environment Variables

### Required
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- EVENT_TOPIC_EXPRESSION - Kafka topic for expression events
- COMMAND_TOPIC_EXPRESSION - Kafka topic for expression commands
- EVENT_TOPIC_MAP_STATUS - Kafka topic for map status events

### Optional
- JAEGER_HOST - Jaeger [host]:[port] for distributed tracing
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace (default: Info)
- EXPRESSION_REVERT_INTERVAL - Interval in milliseconds to check for expired expressions (default: 1000)

## Kafka Message API

### Commands
- **Expression Command**: Used to change a character's expression
  - Topic: Defined by COMMAND_TOPIC_EXPRESSION
  - Structure:
    ```json
    {
      "transactionId": "uuid",
      "characterId": 123,
      "worldId": 0,
      "channelId": 0,
      "mapId": 100000000,
      "expression": 1
    }
    ```

### Events
- **Expression Event**: Emitted when a character's expression is changed
  - Topic: Defined by EVENT_TOPIC_EXPRESSION
  - Structure:
    ```json
    {
      "transactionId": "uuid",
      "characterId": 123,
      "worldId": 0,
      "channelId": 0,
      "mapId": 100000000,
      "expression": 1
    }
    ```

- **Map Status Event**: Consumed to clear expressions when a character exits a map
  - Topic: Defined by EVENT_TOPIC_MAP_STATUS
  - Structure:
    ```json
    {
      "transactionId": "uuid",
      "worldId": 0,
      "channelId": 0,
      "mapId": 100000000,
      "type": "CHARACTER_EXIT",
      "body": {
        "characterId": 123
      }
    }
    ```
