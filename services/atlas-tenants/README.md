# Atlas Tenants Service

A RESTful microservice that provides tenant management for the Mushroom game platform. This service allows creating, retrieving, updating, and deleting tenant configurations.

## Overview

The Atlas Tenants Service is responsible for managing tenant information across the Mushroom game platform. It provides:

- REST API for tenant CRUD operations
- REST API for tenant-specific configuration management (routes, vessels)
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

### Route Configuration Endpoints

#### GET /api/tenants/{tenantId}/configurations/routes

Retrieves all routes for a specific tenant.

**Response**: 200 OK
```json
{
  "data": [
    {
      "type": "routes",
      "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
      "attributes": {
        "name": "Ellinia to Orbis Ferry",
        "startMapId": 101000300,
        "stagingMapId": 101000301,
        "enRouteMapIds": [200090010, 200090011],
        "destinationMapId": 200000100,
        "boardingWindowDuration": 4,
        "preDepartureDuration": 1,
        "travelDuration": 15,
        "cycleInterval": 40
      }
    }
  ]
}
```

#### GET /api/tenants/{tenantId}/configurations/routes/{routeId}

Retrieves a specific route by ID.

**Response**: 200 OK
```json
{
  "data": {
    "type": "routes",
    "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
    "attributes": {
      "name": "Ellinia to Orbis Ferry",
      "startMapId": 101000300,
      "stagingMapId": 101000301,
      "enRouteMapIds": [200090010, 200090011],
      "destinationMapId": 200000100,
      "observationMapId": 200090012,
      "boardingWindowDuration": 4,
      "preDepartureDuration": 1,
      "travelDuration": 15,
      "cycleInterval": 40
    }
  }
}
```

**Response**: 404 Not Found (if route doesn't exist)

#### POST /api/tenants/{tenantId}/configurations/routes

Creates a new route.

**Request Body**:
```json
{
  "data": {
    "type": "routes",
    "attributes": {
      "name": "Ellinia to Orbis Ferry",
      "startMapId": 101000300,
      "stagingMapId": 101000301,
      "enRouteMapIds": [200090010, 200090011],
      "destinationMapId": 200000100,
      "observationMapId": 200090012,
      "boardingWindowDuration": 4,
      "preDepartureDuration": 1,
      "travelDuration": 15,
      "cycleInterval": 40
    }
  }
}
```

**Response**: 201 Created
```json
{
  "data": {
    "type": "routes",
    "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
    "attributes": {
      "name": "Ellinia to Orbis Ferry",
      "startMapId": 101000300,
      "stagingMapId": 101000301,
      "enRouteMapIds": [200090010, 200090011],
      "destinationMapId": 200000100,
      "boardingWindowDuration": 4,
      "preDepartureDuration": 1,
      "travelDuration": 15,
      "cycleInterval": 40
    }
  }
}
```

#### PATCH /api/tenants/{tenantId}/configurations/routes/{routeId}

Updates an existing route.

**Request Body**:
```json
{
  "data": {
    "type": "routes",
    "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
    "attributes": {
      "name": "Ellinia to Orbis Ferry",
      "startMapId": 101000300,
      "stagingMapId": 101000301,
      "enRouteMapIds": [200090010, 200090011],
      "destinationMapId": 200000100,
      "observationMapId": 200090012,
      "boardingWindowDuration": 4,
      "preDepartureDuration": 1,
      "travelDuration": 15,
      "cycleInterval": 40
    }
  }
}
```

**Response**: 200 OK
```json
{
  "data": {
    "type": "routes",
    "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
    "attributes": {
      "name": "Ellinia to Orbis Ferry",
      "startMapId": 101000300,
      "stagingMapId": 101000301,
      "enRouteMapIds": [200090010, 200090011],
      "destinationMapId": 200000100,
      "boardingWindowDuration": 4,
      "preDepartureDuration": 1,
      "travelDuration": 15,
      "cycleInterval": 40
    }
  }
}
```

**Response**: 404 Not Found (if route doesn't exist)

#### DELETE /api/tenants/{tenantId}/configurations/routes/{routeId}

Deletes a route.

**Response**: 204 No Content

**Response**: 404 Not Found (if route doesn't exist)

### Vessel Configuration Endpoints

#### GET /api/tenants/{tenantId}/configurations/vessels

Retrieves all vessels for a specific tenant.

**Response**: 200 OK
```json
{
  "data": [
    {
      "type": "vessels",
      "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
      "attributes": {
        "name": "Ellinia-Orbis Ferry",
        "routeAID": "uuid-for-route-a",
        "routeBID": "uuid-for-route-b",
        "turnaroundDelay": 0
      }
    }
  ]
}
```

#### GET /api/tenants/{tenantId}/configurations/vessels/{vesselId}

Retrieves a specific vessel by ID.

**Response**: 200 OK
```json
{
  "data": {
    "type": "vessels",
    "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
    "attributes": {
      "name": "Ellinia-Orbis Ferry",
      "routeAID": "uuid-for-route-a",
      "routeBID": "uuid-for-route-b",
      "turnaroundDelay": 0
    }
  }
}
```

**Response**: 404 Not Found (if vessel doesn't exist)

#### POST /api/tenants/{tenantId}/configurations/vessels

Creates a new vessel.

**Request Body**:
```json
{
  "data": {
    "type": "vessels",
    "attributes": {
      "name": "Ellinia-Orbis Ferry",
      "routeAID": "uuid-for-route-a",
      "routeBID": "uuid-for-route-b",
      "turnaroundDelay": 0
    }
  }
}
```

**Response**: 201 Created
```json
{
  "data": {
    "type": "vessels",
    "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
    "attributes": {
      "name": "Ellinia-Orbis Ferry",
      "routeAID": "uuid-for-route-a",
      "routeBID": "uuid-for-route-b",
      "turnaroundDelay": 0
    }
  }
}
```

#### PATCH /api/tenants/{tenantId}/configurations/vessels/{vesselId}

Updates an existing vessel.

**Request Body**:
```json
{
  "data": {
    "type": "vessels",
    "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
    "attributes": {
      "name": "Ellinia-Orbis Ferry",
      "routeAID": "uuid-for-route-a",
      "routeBID": "uuid-for-route-b",
      "turnaroundDelay": 0
    }
  }
}
```

**Response**: 200 OK
```json
{
  "data": {
    "type": "vessels",
    "id": "12aba1dd-3799-42a2-991e-f1f1633b9129",
    "attributes": {
      "name": "Ellinia-Orbis Ferry",
      "routeAID": "uuid-for-route-a",
      "routeBID": "uuid-for-route-b",
      "turnaroundDelay": 0
    }
  }
}
```

**Response**: 404 Not Found (if vessel doesn't exist)

#### DELETE /api/tenants/{tenantId}/configurations/vessels/{vesselId}

Deletes a vessel.

**Response**: 204 No Content

**Response**: 404 Not Found (if vessel doesn't exist)
