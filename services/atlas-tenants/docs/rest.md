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
- 400: Invalid request body
- 500: Internal server error (includes tenant not found)

---

### DELETE /tenants/{tenantId}

Deletes a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
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
      "id": "uuid",
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
    "id": "uuid",
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
    "id": "uuid",
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
- 400: Invalid request body
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
    "id": "uuid",
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
    "id": "uuid",
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
- 400: Invalid request body
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
      "id": "uuid",
      "attributes": {
        "name": "string",
        "routeAID": "uuid",
        "routeBID": "uuid",
        "turnaroundDelay": 0
      }
    }
  ]
}
```

**Error Conditions**:
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
    "id": "uuid",
    "attributes": {
      "name": "string",
      "routeAID": "uuid",
      "routeBID": "uuid",
      "turnaroundDelay": 0
    }
  }
}
```

**Error Conditions**:
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
      "routeAID": "uuid",
      "routeBID": "uuid",
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
    "id": "uuid",
    "attributes": {
      "name": "string",
      "routeAID": "uuid",
      "routeBID": "uuid",
      "turnaroundDelay": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body
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
    "id": "uuid",
    "attributes": {
      "name": "string",
      "routeAID": "uuid",
      "routeBID": "uuid",
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
    "id": "uuid",
    "attributes": {
      "name": "string",
      "routeAID": "uuid",
      "routeBID": "uuid",
      "turnaroundDelay": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body
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
- 500: Internal server error
