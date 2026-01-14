# Atlas Pets Service

A RESTful microservice for managing pets in the Mushroom game ecosystem. This service handles pet creation, retrieval, and lifecycle management including pet hunger and other attributes.

## Overview

Atlas Pets Service provides a comprehensive API for managing in-game pets, including:
- Pet creation and retrieval
- Pet attribute management (hunger, closeness, etc.)
- Pet-character relationships
- Temporal data tracking (position, stance, etc.)

The service integrates with other game services through Kafka messaging and provides RESTful endpoints for direct interaction.

## Installation

### Prerequisites
- Go 1.24 or higher
- Docker (for containerized deployment)
- Kafka cluster
- PostgreSQL database

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| JAEGER_HOST | Jaeger host and port for tracing | jaeger:14268 |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) | Info |
| REST_PORT | Port for the REST API | 8080 |
| DB_HOST | PostgreSQL database host | localhost |
| DB_PORT | PostgreSQL database port | 5432 |
| DB_USER | PostgreSQL database username | postgres |
| DB_PASS | PostgreSQL database password | postgres |
| DB_NAME | PostgreSQL database name | pets |
| KAFKA_BROKERS | Comma-separated list of Kafka brokers | localhost:9092 |

## API

### Header

All RESTful requests require the following headers to identify the server instance:

```
TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123
REGION: GMS
MAJOR_VERSION: 83
MINOR_VERSION: 1
```

### Endpoints

#### Pet Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/characters/{characterId}/pets | Get all pets for a specific character |
| POST | /api/characters/{characterId}/pets | Create a pet for a specific character |
| POST | /api/pets | Create a pet (general endpoint) |
| GET | /api/pets/{petId} | Get a specific pet by ID |

### Request/Response Examples

#### Get Pet by ID

Request:
```
GET /api/pets/12345 HTTP/1.1
Host: example.com
TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123
REGION: GMS
MAJOR_VERSION: 83
MINOR_VERSION: 1
```

Response:
```json
{
  "data": {
    "type": "pets",
    "id": "12345",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 10,
      "closeness": 100,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 54321,
      "slot": 0,
      "x": 100,
      "y": 200,
      "stance": 0,
      "fh": 5,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

#### Create Pet (General Endpoint)

Request:
```
POST /api/pets HTTP/1.1
Host: example.com
Content-Type: application/json
TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123
REGION: GMS
MAJOR_VERSION: 83
MINOR_VERSION: 1

{
  "data": {
    "type": "pets",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 54321,
      "slot": 0,
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

Response:
```json
{
  "data": {
    "type": "pets",
    "id": "12345",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 54321,
      "slot": 0,
      "x": 0,
      "y": 0,
      "stance": 0,
      "fh": 0,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

#### Create Pet for Character

Request:
```
POST /api/characters/54321/pets HTTP/1.1
Host: example.com
Content-Type: application/json
TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123
REGION: GMS
MAJOR_VERSION: 83
MINOR_VERSION: 1

{
  "data": {
    "type": "pets",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "slot": 0,
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

Response:
```json
{
  "data": {
    "type": "pets",
    "id": "12345",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 54321,
      "slot": 0,
      "x": 0,
      "y": 0,
      "stance": 0,
      "fh": 0,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

## Kafka Integration

### Consumed Topics

| Environment Variable | Description |
|---------------------|-------------|
| EVENT_TOPIC_CHARACTER_STATUS | Character status events (login, logout, delete, map/channel changes) |
| COMMAND_TOPIC_PET | Pet commands (spawn, despawn, attribute updates) |
| COMMAND_TOPIC_PET_MOVEMENT | Pet movement commands |

### Commands

#### Pet Command Topic (COMMAND_TOPIC_PET)

| Command Type | Description |
|-------------|-------------|
| SPAWN | Spawn a pet for a character |
| DESPAWN | Despawn a pet |
| ATTEMPT_COMMAND | Execute a pet command (tricks) |
| AWARD_CLOSENESS | Award closeness points to a pet |
| AWARD_FULLNESS | Award fullness points to a pet |
| AWARD_LEVEL | Award level to a pet |
| EXCLUDE | Set excluded items for pet auto-loot |

##### Sample Command Payload

```json
{
  "transactionId": "550e8400-e29b-41d4-a716-446655440000",
  "actorId": 54321,
  "petId": 12345,
  "type": "SPAWN",
  "body": {
    "lead": true
  }
}
```

#### Pet Movement Command Topic (COMMAND_TOPIC_PET_MOVEMENT)

```json
{
  "worldId": 0,
  "channelId": 1,
  "mapId": 100000000,
  "objectId": 12345,
  "observerId": 54321,
  "x": 100,
  "y": 200,
  "stance": 2
}
```

### Produced Topics

| Environment Variable | Description |
|---------------------|-------------|
| EVENT_TOPIC_PET_STATUS | Pet status change events |

### Events

#### Pet Status Event Topic (EVENT_TOPIC_PET_STATUS)

| Event Type | Description |
|-----------|-------------|
| CREATED | Pet was created |
| DELETED | Pet was deleted |
| SPAWNED | Pet was spawned (made active) |
| DESPAWNED | Pet was despawned (made inactive) |
| COMMAND_RESPONSE | Response to a pet command attempt |
| CLOSENESS_CHANGED | Pet closeness attribute changed |
| FULLNESS_CHANGED | Pet fullness attribute changed |
| LEVEL_CHANGED | Pet level changed |
| SLOT_CHANGED | Pet slot position changed |
| EXCLUDE_CHANGED | Pet excluded items changed |

##### Sample Event Payload

```json
{
  "petId": 12345,
  "ownerId": 54321,
  "type": "SPAWNED",
  "body": {
    "templateId": 5000,
    "name": "Fluffy",
    "slot": 0,
    "level": 10,
    "closeness": 100,
    "fullness": 100,
    "x": 100,
    "y": 200,
    "stance": 2,
    "fh": 5
  }
}
```
