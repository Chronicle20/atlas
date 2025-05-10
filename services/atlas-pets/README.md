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
