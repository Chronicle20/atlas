# atlas-reactors
Mushroom game reactors Service

## Overview

A RESTful resource which provides reactors services. Reactors are interactive game objects that respond to player actions. This service manages reactor instances in-memory per world/channel/map.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- COMMAND_TOPIC_REACTOR - Kafka topic for reactor commands
- EVENT_TOPIC_REACTOR_STATUS - Kafka topic for reactor status events

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Requests

#### Get Reactor by ID

Retrieves a single reactor by its unique ID.

```
GET /api/reactors/{reactorId}
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| reactorId | uint32 | The unique reactor ID |

**Response Codes:**
| Code | Description |
|------|-------------|
| 200 | Success - returns reactor |
| 404 | Reactor not found |

**Example Response:**
```json
{
  "data": {
    "id": "1000000001",
    "type": "reactors",
    "attributes": {
      "worldId": 1,
      "channelId": 1,
      "mapId": 100000,
      "classification": 2000000,
      "name": "reactor-001",
      "state": 0,
      "eventState": 0,
      "x": 150,
      "y": 250,
      "delay": 0,
      "direction": 0
    }
  }
}
```

#### Get Reactors in Map

Retrieves all reactors in a specific world/channel/map.

```
GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/reactors
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| worldId | byte | The world ID |
| channelId | byte | The channel ID |
| mapId | uint32 | The map ID |

**Response Codes:**
| Code | Description |
|------|-------------|
| 200 | Success - returns array of reactors |
| 500 | Internal server error |

**Example Response:**
```json
{
  "data": [
    {
      "id": "1000000001",
      "type": "reactors",
      "attributes": {
        "worldId": 1,
        "channelId": 1,
        "mapId": 100000,
        "classification": 2000000,
        "name": "reactor-001",
        "state": 0,
        "eventState": 0,
        "x": 150,
        "y": 250,
        "delay": 0,
        "direction": 0
      }
    }
  ]
}
```

#### Get Reactor by ID in Map

Retrieves a specific reactor within a map context. The reactor must exist in the specified world/channel/map.

```
GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/reactors/{reactorId}
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| worldId | byte | The world ID |
| channelId | byte | The channel ID |
| mapId | uint32 | The map ID |
| reactorId | uint32 | The unique reactor ID |

**Response Codes:**
| Code | Description |
|------|-------------|
| 200 | Success - returns reactor |
| 404 | Reactor not found or not in specified map |

#### Create Reactor in Map

Creates a new reactor in the specified map. The request is processed asynchronously via Kafka.

```
POST /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/reactors
```

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| worldId | byte | The world ID |
| channelId | byte | The channel ID |
| mapId | uint32 | The map ID |

**Request Body:**
```json
{
  "data": {
    "type": "reactors",
    "attributes": {
      "classification": 2000000,
      "name": "reactor-001",
      "state": 0,
      "x": 150,
      "y": 250,
      "delay": 0,
      "direction": 0
    }
  }
}
```

**Request Body Fields:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| classification | uint32 | Yes | The reactor type/classification ID |
| name | string | Yes | The reactor name |
| state | int8 | Yes | Initial reactor state |
| x | int16 | Yes | X coordinate position |
| y | int16 | Yes | Y coordinate position |
| delay | uint32 | No | Reactor delay value |
| direction | byte | No | Reactor facing direction |

**Response Codes:**
| Code | Description |
|------|-------------|
| 202 | Accepted - request queued for processing |
| 500 | Internal server error |

## Kafka Topics

### Commands (Consumer)

**Topic:** `COMMAND_TOPIC_REACTOR`

Processes reactor creation commands.

### Events (Producer)

**Topic:** `EVENT_TOPIC_REACTOR_STATUS`

Emits reactor status events:
- `CREATED` - When a reactor is created
- `DESTROYED` - When a reactor is destroyed

## Architecture Notes

This service uses an **in-memory registry pattern** instead of database persistence. This is intentional for managing volatile game state - reactors only exist during active game sessions and do not need to persist across service restarts.
