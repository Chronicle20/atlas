# Atlas Chairs Service

## Overview

Atlas Chairs is a microservice for the Mushroom game that manages chair entities within the game world. It provides RESTful APIs to track which characters are sitting on chairs and to retrieve chair information by character ID or map location.

## Features

- Track which characters are sitting on chairs
- Retrieve chair information by character ID
- List all chairs in a specific map
- Process chair and character events via Kafka
- Distributed tracing with Jaeger

## API Documentation

### Get Chair by Character ID

Retrieves chair information for a specific character.

```
GET /api/chairs/{characterId}
```

#### Response

```json
{
  "data": {
    "type": "chairs",
    "id": "123",
    "attributes": {
      "type": "wooden_chair",
      "characterId": 456
    }
  }
}
```

### Get Chairs in Map

Retrieves all chairs in a specific map.

```
GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/chairs
```

#### Response

```json
{
  "data": [
    {
      "type": "chairs",
      "id": "123",
      "attributes": {
        "type": "wooden_chair",
        "characterId": 456
      }
    },
    {
      "type": "chairs",
      "id": "124",
      "attributes": {
        "type": "stone_chair",
        "characterId": 789
      }
    }
  ]
}
```

## Environment Variables

- `JAEGER_HOST_PORT` - Jaeger [host]:[port] for distributed tracing
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)
- `REST_PORT` - Port for the REST server
- `COMMAND_TOPIC_CHAIR` - Kafka topic for chair commands
- `EVENT_TOPIC_CHAIR_STATUS` - Kafka topic for chair status events
- `EVENT_TOPIC_CHARACTER_STATUS` - Kafka topic for character status events

## Kafka Commands and Events

### Chair Commands
- `USE` - Command to use a chair
  ```json
  {
    "worldId": 0,
    "channelId": 0,
    "mapId": 100000000,
    "type": "USE",
    "body": {
      "characterId": 123,
      "chairType": "FIXED",
      "chairId": 456
    }
  }
  ```
- `CANCEL` - Command to cancel sitting on a chair
  ```json
  {
    "worldId": 0,
    "channelId": 0,
    "mapId": 100000000,
    "type": "CANCEL",
    "body": {
      "characterId": 123
    }
  }
  ```

### Chair Events
- `USED` - Event indicating a character is now sitting on a chair
  ```json
  {
    "worldId": 0,
    "channelId": 0,
    "mapId": 100000000,
    "chairType": "FIXED",
    "chairId": 456,
    "type": "USED",
    "body": {
      "characterId": 123
    }
  }
  ```
- `CANCELLED` - Event indicating a character has stopped sitting on a chair
  ```json
  {
    "worldId": 0,
    "channelId": 0,
    "mapId": 100000000,
    "chairType": "FIXED",
    "chairId": 456,
    "type": "CANCELLED",
    "body": {
      "characterId": 123
    }
  }
  ```
- `ERROR` - Event indicating an error occurred with a chair operation
  ```json
  {
    "worldId": 0,
    "channelId": 0,
    "mapId": 100000000,
    "chairType": "FIXED",
    "chairId": 456,
    "type": "ERROR",
    "body": {
      "characterId": 123,
      "type": "ALREADY_SITING"
    }
  }
  ```

### Character Events (Consumed)
- `LOGIN` - Event indicating a character has logged in
- `LOGOUT` - Event indicating a character has logged out
- `CHANNEL_CHANGED` - Event indicating a character has changed channels
- `MAP_CHANGED` - Event indicating a character has changed maps
