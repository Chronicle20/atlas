# REST

## Endpoints

### GET /tenants

Retrieves all tenants.

**Parameters**: None

**Request Model**: None

**Response Model**:
```json
{
  "data": [
    {
      "type": "tenants",
      "id": "uuid",
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

**Error Conditions**:
- 500: Internal server error

---

### GET /tenants/{tenantId}

Retrieves a tenant by ID.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "tenants",
    "id": "uuid",
    "attributes": {
      "name": "string",
      "region": "string",
      "majorVersion": 0,
      "minorVersion": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 404: Tenant not found

---

### POST /tenants

Creates a new tenant.

**Parameters**: None

**Request Model**:
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

**Response Model**:
```json
{
  "data": {
    "type": "tenants",
    "id": "uuid",
    "attributes": {
      "name": "string",
      "region": "string",
      "majorVersion": 0,
      "minorVersion": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body
- 500: Internal server error

---

### PATCH /tenants/{tenantId}

Updates an existing tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "tenants",
    "id": "uuid",
    "attributes": {
      "name": "string",
      "region": "string",
      "majorVersion": 0,
      "minorVersion": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "tenants",
    "id": "uuid",
    "attributes": {
      "name": "string",
      "region": "string",
      "majorVersion": 0,
      "minorVersion": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error (includes tenant not found)

---

### DELETE /tenants/{tenantId}

Deletes a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format
- 500: Internal server error (includes tenant not found)

---

### GET /tenants/{tenantId}/configurations/routes

Retrieves all routes for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": [
    {
      "type": "routes",
      "id": "string",
      "attributes": {
        "name": "string",
        "startMapId": 0,
        "stagingMapId": 0,
        "enRouteMapIds": [0],
        "destinationMapId": 0,
        "observationMapId": 0,
        "boardingWindowDuration": 0,
        "preDepartureDuration": 0,
        "travelDuration": 0,
        "cycleInterval": 0
      }
    }
  ]
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 500: Internal server error

---

### GET /tenants/{tenantId}/configurations/routes/{routeId}

Retrieves a route by ID.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `routeId` (path, string): Route identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "routes",
    "id": "string",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "stagingMapId": 0,
      "enRouteMapIds": [0],
      "destinationMapId": 0,
      "observationMapId": 0,
      "boardingWindowDuration": 0,
      "preDepartureDuration": 0,
      "travelDuration": 0,
      "cycleInterval": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or missing route ID
- 404: Route not found

---

### POST /tenants/{tenantId}/configurations/routes

Creates a new route.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "routes",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "stagingMapId": 0,
      "enRouteMapIds": [0],
      "destinationMapId": 0,
      "observationMapId": 0,
      "boardingWindowDuration": 0,
      "preDepartureDuration": 0,
      "travelDuration": 0,
      "cycleInterval": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "routes",
    "id": "string",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "stagingMapId": 0,
      "enRouteMapIds": [0],
      "destinationMapId": 0,
      "observationMapId": 0,
      "boardingWindowDuration": 0,
      "preDepartureDuration": 0,
      "travelDuration": 0,
      "cycleInterval": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error

---

### PATCH /tenants/{tenantId}/configurations/routes/{routeId}

Updates an existing route.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `routeId` (path, string): Route identifier

**Request Model**:
```json
{
  "data": {
    "type": "routes",
    "id": "string",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "stagingMapId": 0,
      "enRouteMapIds": [0],
      "destinationMapId": 0,
      "observationMapId": 0,
      "boardingWindowDuration": 0,
      "preDepartureDuration": 0,
      "travelDuration": 0,
      "cycleInterval": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "routes",
    "id": "string",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "stagingMapId": 0,
      "enRouteMapIds": [0],
      "destinationMapId": 0,
      "observationMapId": 0,
      "boardingWindowDuration": 0,
      "preDepartureDuration": 0,
      "travelDuration": 0,
      "cycleInterval": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body, tenant ID format, or missing route ID
- 500: Internal server error (includes route not found)

---

### DELETE /tenants/{tenantId}/configurations/routes/{routeId}

Deletes a route.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `routeId` (path, string): Route identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format or missing route ID
- 500: Internal server error

---

### POST /tenants/{tenantId}/configurations/routes/seed

Deletes all existing routes for a tenant and loads them from seed files.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "deletedCount": 0,
  "createdCount": 0,
  "failedCount": 0,
  "errors": ["string"]
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 500: Internal server error

---

### GET /tenants/{tenantId}/configurations/vessels

Retrieves all vessels for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": [
    {
      "type": "vessels",
      "id": "string",
      "attributes": {
        "name": "string",
        "routeAID": "string",
        "routeBID": "string",
        "turnaroundDelay": 0
      }
    }
  ]
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 500: Internal server error

---

### GET /tenants/{tenantId}/configurations/vessels/{vesselId}

Retrieves a vessel by ID.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `vesselId` (path, string): Vessel identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "vessels",
    "id": "string",
    "attributes": {
      "name": "string",
      "routeAID": "string",
      "routeBID": "string",
      "turnaroundDelay": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or missing vessel ID
- 404: Vessel not found

---

### POST /tenants/{tenantId}/configurations/vessels

Creates a new vessel.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "vessels",
    "attributes": {
      "name": "string",
      "routeAID": "string",
      "routeBID": "string",
      "turnaroundDelay": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "vessels",
    "id": "string",
    "attributes": {
      "name": "string",
      "routeAID": "string",
      "routeBID": "string",
      "turnaroundDelay": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error

---

### PATCH /tenants/{tenantId}/configurations/vessels/{vesselId}

Updates an existing vessel.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `vesselId` (path, string): Vessel identifier

**Request Model**:
```json
{
  "data": {
    "type": "vessels",
    "id": "string",
    "attributes": {
      "name": "string",
      "routeAID": "string",
      "routeBID": "string",
      "turnaroundDelay": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "vessels",
    "id": "string",
    "attributes": {
      "name": "string",
      "routeAID": "string",
      "routeBID": "string",
      "turnaroundDelay": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body, tenant ID format, or missing vessel ID
- 500: Internal server error (includes vessel not found)

---

### DELETE /tenants/{tenantId}/configurations/vessels/{vesselId}

Deletes a vessel.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `vesselId` (path, string): Vessel identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format or missing vessel ID
- 500: Internal server error

---

### POST /tenants/{tenantId}/configurations/vessels/seed

Deletes all existing vessels for a tenant and loads them from seed files.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "deletedCount": 0,
  "createdCount": 0,
  "failedCount": 0,
  "errors": ["string"]
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 500: Internal server error

---

### GET /tenants/{tenantId}/configurations/instance-routes

Retrieves all instance routes for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": [
    {
      "type": "instance-routes",
      "id": "string",
      "attributes": {
        "name": "string",
        "startMapId": 0,
        "transitMapIds": [0],
        "destinationMapId": 0,
        "capacity": 0,
        "boardingWindowSeconds": 0,
        "travelDurationSeconds": 0,
        "transitMessage": "string"
      }
    }
  ]
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 500: Internal server error

---

### GET /tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}

Retrieves an instance route by ID.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `instanceRouteId` (path, string): Instance route identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "instance-routes",
    "id": "string",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "transitMapIds": [0],
      "destinationMapId": 0,
      "capacity": 0,
      "boardingWindowSeconds": 0,
      "travelDurationSeconds": 0,
      "transitMessage": "string"
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or missing instance route ID
- 404: Instance route not found

---

### POST /tenants/{tenantId}/configurations/instance-routes

Creates a new instance route.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "instance-routes",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "transitMapIds": [0],
      "destinationMapId": 0,
      "capacity": 0,
      "boardingWindowSeconds": 0,
      "travelDurationSeconds": 0,
      "transitMessage": "string"
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "instance-routes",
    "id": "string",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "transitMapIds": [0],
      "destinationMapId": 0,
      "capacity": 0,
      "boardingWindowSeconds": 0,
      "travelDurationSeconds": 0,
      "transitMessage": "string"
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error

---

### PATCH /tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}

Updates an existing instance route.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `instanceRouteId` (path, string): Instance route identifier

**Request Model**:
```json
{
  "data": {
    "type": "instance-routes",
    "id": "string",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "transitMapIds": [0],
      "destinationMapId": 0,
      "capacity": 0,
      "boardingWindowSeconds": 0,
      "travelDurationSeconds": 0,
      "transitMessage": "string"
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "instance-routes",
    "id": "string",
    "attributes": {
      "name": "string",
      "startMapId": 0,
      "transitMapIds": [0],
      "destinationMapId": 0,
      "capacity": 0,
      "boardingWindowSeconds": 0,
      "travelDurationSeconds": 0,
      "transitMessage": "string"
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body, tenant ID format, or missing instance route ID
- 500: Internal server error (includes instance route not found)

---

### DELETE /tenants/{tenantId}/configurations/instance-routes/{instanceRouteId}

Deletes an instance route.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `instanceRouteId` (path, string): Instance route identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format or missing instance route ID
- 500: Internal server error

---

### POST /tenants/{tenantId}/configurations/instance-routes/seed

Deletes all existing instance routes for a tenant and loads them from seed files.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "deletedCount": 0,
  "createdCount": 0,
  "failedCount": 0,
  "errors": ["string"]
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 500: Internal server error
