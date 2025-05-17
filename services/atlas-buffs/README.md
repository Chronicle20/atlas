# atlas-buffs

A microservice for managing character buffs in the Mushroom game. This service provides a RESTful API for retrieving character buffs and uses Kafka for receiving commands to apply or cancel buffs.

## Overview

The atlas-buffs service is responsible for:
- Managing temporary stat modifications (buffs) for game characters
- Tracking buff durations and handling expirations
- Providing a REST API to query current character buffs
- Processing buff application and cancellation commands via Kafka

## API Endpoints

### GET /api/characters/{characterId}/buffs

Retrieves all active buffs for a specific character.

**Response:**
```json
[
  {
    "sourceId": 123,
    "duration": 30000,
    "changes": [
      {
        "type": "str",
        "amount": 5
      }
    ],
    "createdAt": "2023-01-01T12:00:00Z",
    "expiresAt": "2023-01-01T12:00:30Z"
  }
]
```

## Kafka Integration

The service consumes messages from a Kafka topic to process buff commands:

### Apply Buff Command
Applies a new buff to a character with specified stat changes, source ID, and duration.

### Cancel Buff Command
Cancels an existing buff for a character based on the source ID.

## Installation

### Prerequisites
- Go 1.24 or higher
- Kafka cluster
- Jaeger (for tracing)

## Environment Variables

- `JAEGER_HOST` - Jaeger [host]:[port] for distributed tracing
- `JAEGER_HOST_PORT` - Alternative to JAEGER_HOST for specifying Jaeger endpoint
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)
- `BOOTSTRAP_SERVERS` - Comma-separated list of Kafka bootstrap servers
- `EVENT_TOPIC_CHARACTER_BUFF_STATUS` - Kafka topic for publishing buff status events
- `COMMAND_TOPIC_CHARACTER_BUFF` - Kafka topic for receiving buff commands

## Tasks

The service includes a task system that handles buff expirations automatically.
