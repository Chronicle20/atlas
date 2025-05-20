# atlas-chalkboards
Mushroom game chalkboards Service

## Overview

A RESTful microservice that provides chalkboard functionality for the Mushroom game. This service manages chalkboard messages associated with characters and maps in the game world. Chalkboards allow players to leave messages that can be viewed by other players.

## Features

- Retrieve chalkboard messages by character ID
- List all chalkboards in a specific map
- Kafka integration for event-driven communication with other services
- Jaeger tracing for distributed system monitoring

## Setup

### Prerequisites

- Go 1.24.x
- Docker (for containerized deployment)

### Environment Variables

- `JAEGER_HOST` - Jaeger [host]:[port] for distributed tracing
- `LOG_LEVEL` - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

## Building and Running

### Local Development

```bash
cd atlas.com/chalkboards
go mod download
go run main.go
```

### Docker Build

Windows:
```
docker-build.bat
```

Linux/Mac:
```bash
./docker-build.sh
```

Or manually:
```bash
docker build -t atlas-chalkboards .
docker run -p 8080:8080 atlas-chalkboards
```

## API

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

## Architecture

This service is part of a microservices architecture and communicates with other services through Kafka. It consumes events related to characters and chalkboards, and provides REST API endpoints for retrieving chalkboard data.
