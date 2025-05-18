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

- `JAEGER_HOST` - Jaeger [host]:[port] for distributed tracing
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)
- `BASE_SERVICE_URL` - Base URL for the service [scheme]://[host]:[port]/api/
- `COMMAND_TOPIC_CHAIR` - Kafka topic for chair commands
- `EVENT_TOPIC_CHAIR_STATUS` - Kafka topic for chair status events
- `EVENT_TOPIC_CHARACTER_STATUS` - Kafka topic for character status events
