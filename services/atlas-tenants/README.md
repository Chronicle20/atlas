# Atlas Tenants Service

A RESTful microservice that provides tenant management for the Mushroom game platform. This service allows creating, retrieving, updating, and deleting tenant configurations.

## Overview

The Atlas Tenants Service is responsible for managing tenant information across the Mushroom game platform. It provides:

- REST API for tenant CRUD operations
- Kafka event emission for tenant state changes

## Environment Variables

### Required Environment Variables

- `REST_PORT` - Port for the REST API server
- `BOOTSTRAP_SERVERS` - Kafka bootstrap servers (comma-separated list)
- `DB_HOST` - PostgreSQL database host
- `DB_PORT` - PostgreSQL database port
- `DB_USER` - PostgreSQL database username
- `DB_PASSWORD` - PostgreSQL database password
- `DB_NAME` - PostgreSQL database name

### Optional Environment Variables

- `JAEGER_HOST_PORT` - Jaeger agent host and port for distributed tracing
- `LOG_LEVEL` - Logging level (Panic / Fatal / Error / Warn / Info / Debug / Trace)

## Kafka Events

The service emits events to the following Kafka topics:

### tenant.status

This topic contains events related to tenant lifecycle changes.

Event types:
- `CREATED` - Emitted when a new tenant is created
- `UPDATED` - Emitted when a tenant is updated
- `DELETED` - Emitted when a tenant is deleted

Event structure:
```json
{
  "tenantId": "uuid-string",
  "type": "EVENT_TYPE",
  "body": {
    "name": "string",
    "region": "string",
    "majorVersion": 0,
    "minorVersion": 0
  }
}
```

## API

### Endpoints

#### GET /api/tenants

Retrieves all tenants.

**Response**: 200 OK
```json
{
  "data": [
    {
      "type": "tenants",
      "id": "083839c6-c47c-42a6-9585-76492795d123",
      "attributes": {
        "name": "string",
        "region": "string",
        "majorVersion": 0,
        "minorVersion": 0
      }
    }
  ]
}
```

#### GET /api/tenants/{tenantId}

Retrieves a specific tenant by ID.

**Response**: 200 OK
```json
{
  "data": {
    "type": "tenants",
    "id": "083839c6-c47c-42a6-9585-76492795d123",
    "attributes": {
      "name": "string",
      "region": "string",
      "majorVersion": 0,
      "minorVersion": 0
    }
  }
}
```

**Response**: 404 Not Found (if tenant doesn't exist)

#### POST /api/tenants

Creates a new tenant.

**Request Body**:
```json
{
  "data": {
    "type": "tenants",
    "attributes": {
      "name": "string",
      "region": "string",
      "majorVersion": 0,
      "minorVersion": 0
    }
  }
}
```

**Response**: 201 Created
```json
{
  "data": {
    "type": "tenants",
    "id": "083839c6-c47c-42a6-9585-76492795d123",
    "attributes": {
      "name": "string",
      "region": "string",
      "majorVersion": 0,
      "minorVersion": 0
    }
  }
}
```

#### PATCH /api/tenants/{tenantId}

Updates an existing tenant.

**Request Body**:
```json
{
  "data": {
    "type": "tenants",
    "id": "083839c6-c47c-42a6-9585-76492795d123",
    "attributes": {
      "name": "string",
      "region": "string",
      "majorVersion": 0,
      "minorVersion": 0
    }
  }
}
```

**Response**: 200 OK
```json
{
  "data": {
    "type": "tenants",
    "id": "083839c6-c47c-42a6-9585-76492795d123",
    "attributes": {
      "name": "string",
      "region": "string",
      "majorVersion": 0,
      "minorVersion": 0
    }
  }
}
```

**Response**: 404 Not Found (if tenant doesn't exist)

#### DELETE /api/tenants/{tenantId}

Deletes a tenant.

**Response**: 204 No Content

**Response**: 404 Not Found (if tenant doesn't exist)