# atlas-portals

Mushroom game portals service - handles portal entry and character map transitions.

## Overview

A stateless Kafka processor service that handles portal entry commands. This service does not maintain its own database state - it fetches portal data from an external DATA service and emits Kafka events to trigger character map changes.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      atlas-portals                              │
│                  (Stateless Processor)                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────────────┐    │
│  │   Kafka     │    │   Business   │    │   Kafka         │    │
│  │  Consumer   │───▶│    Logic     │───▶│  Producer       │    │
│  └─────────────┘    └──────────────┘    └─────────────────┘    │
│         │                  │                     │              │
│         │                  ▼                     │              │
│         │          ┌──────────────┐              │              │
│         │          │ REST Client  │              │              │
│         │          │ (DATA Svc)   │              │              │
│         │          └──────────────┘              │              │
│         │                  │                     │              │
└─────────┼──────────────────┼─────────────────────┼──────────────┘
          │                  │                     │
          ▼                  ▼                     ▼
   COMMAND_TOPIC        DATA Service      EVENT_TOPIC_CHARACTER_STATUS
      _PORTAL                              COMMAND_TOPIC_CHARACTER
```

### Key Characteristics

- **Stateless**: No database - all state is derived from external services
- **Event-Driven**: Kafka-only interface (no REST endpoints exposed)
- **Single Responsibility**: Portal entry logic and map transition coordination

## Kafka Topics

### Consumed Topics

| Topic | Environment Variable | Message Type | Description |
|-------|---------------------|--------------|-------------|
| Portal Commands | `COMMAND_TOPIC_PORTAL` | `commandEvent[enterBody]` | Portal entry requests from game clients |

**Command Message Format:**
```json
{
  "worldId": 1,
  "channelId": 1,
  "mapId": 100000000,
  "portalId": 5,
  "type": "ENTER",
  "body": {
    "characterId": 12345
  }
}
```

### Produced Topics

| Topic | Environment Variable | Message Type | Description |
|-------|---------------------|--------------|-------------|
| Character Status | `EVENT_TOPIC_CHARACTER_STATUS` | `statusEvent[statusEventStatChangedBody]` | Enable actions after portal interaction |
| Character Commands | `COMMAND_TOPIC_CHARACTER` | `commandEvent[changeMapBody]` | Map change commands |

**Status Event Format (Enable Actions):**
```json
{
  "characterId": 12345,
  "type": "STAT_CHANGED",
  "worldId": 1,
  "body": {
    "channelId": 1,
    "exclRequestSent": true
  }
}
```

**Command Event Format (Change Map):**
```json
{
  "worldId": 1,
  "characterId": 12345,
  "type": "CHANGE_MAP",
  "body": {
    "channelId": 1,
    "mapId": 200000000,
    "portalId": 0
  }
}
```

## External Dependencies

### DATA Service

The service fetches portal information from an external DATA service via REST.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/data/maps/{mapId}/portals/{portalId}` | GET | Fetch portal by ID |
| `/data/maps/{mapId}/portals?name={name}` | GET | Fetch portals by name |

**Response Format (JSON:API):**
```json
{
  "data": {
    "type": "portals",
    "id": "5",
    "attributes": {
      "name": "portal_name",
      "target": "target_portal_name",
      "type": 0,
      "x": 100,
      "y": 200,
      "targetMapId": 200000000,
      "scriptName": ""
    }
  }
}
```

## Portal Entry Flow

1. **Receive Command**: Kafka consumer receives portal entry command
2. **Fetch Portal**: REST call to DATA service for portal information
3. **Determine Action**:
   - **Portal has script**: Enable actions (script execution handled elsewhere)
   - **Portal has target map**: Warp character to target map/portal
   - **Neither**: Enable actions (dead-end portal)
4. **Emit Events**: Produce Kafka events for character status/commands

## Environment

| Variable | Description | Example |
|----------|-------------|---------|
| `JAEGER_HOST` | Jaeger tracing endpoint | `jaeger:6831` |
| `LOG_LEVEL` | Logging level | `Debug` |
| `BOOTSTRAP_SERVERS` | Kafka bootstrap servers | `kafka:9092` |
| `DATA_SERVICE_URL` | DATA service base URL | `http://data:8080/api/` |
| `COMMAND_TOPIC_PORTAL` | Portal command topic | `COMMAND_TOPIC_PORTAL` |
| `EVENT_TOPIC_CHARACTER_STATUS` | Character status event topic | `EVENT_TOPIC_CHARACTER_STATUS` |
| `COMMAND_TOPIC_CHARACTER` | Character command topic | `COMMAND_TOPIC_CHARACTER` |

## Development

### Running Tests

```bash
cd atlas.com/portals
go test ./...
```

### Building

```bash
cd atlas.com/portals
go build -o atlas-portals .
```
