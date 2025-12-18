# atlas-chalkboards
Mushroom game chalkboards Service

## Overview

A RESTful microservice that provides chalkboard functionality for the Mushroom game. This service manages chalkboard messages associated with characters and maps in the game world. Chalkboards allow players to leave messages that can be viewed by other players.

## Features

- Retrieve chalkboard messages by character ID
- List all chalkboards in a specific map
- Kafka integration for event-driven communication with other services
- Jaeger tracing for distributed system monitoring

## Environment Variables

### Required
- `REST_PORT` - Port for the REST API server
- `BOOTSTRAP_SERVERS` - Comma-separated list of Kafka broker addresses
- `JAEGER_HOST_PORT` - Jaeger host:port for distributed tracing (e.g., "jaeger:6831")

### Kafka Topics
- `COMMAND_TOPIC_CHALKBOARD` - Topic for chalkboard commands
- `EVENT_TOPIC_CHALKBOARD_STATUS` - Topic for chalkboard status events
- `EVENT_TOPIC_CHARACTER_STATUS` - Topic for character status events

### Optional
- `LOG_LEVEL` - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace (default: Info)

## REST API

### Header Requirements

All RESTful requests require the following header information to identify the server instance:

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

#### Get Chalkboard by Character ID

```
GET /api/chalkboards/{characterId}
```

Retrieves the chalkboard message associated with a specific character.

**Response:**
```json
{
  "data": {
    "type": "chalkboard",
    "id": "123",
    "attributes": {
      "message": "Hello world!"
    }
  }
}
```

#### Get Chalkboards in Map

```
GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/chalkboards
```

Retrieves all chalkboard messages in a specific map.

**Response:**
```json
{
  "data": [
    {
      "type": "chalkboard",
      "id": "123",
      "attributes": {
        "message": "Hello world!"
      }
    },
    {
      "type": "chalkboard",
      "id": "456",
      "attributes": {
        "message": "Another message"
      }
    }
  ]
}
```