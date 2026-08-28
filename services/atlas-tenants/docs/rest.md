# REST

## Endpoints

### GET /tenants

Retrieves all tenants (paginated). Results are scoped to the caller's environment; a caller with no environment sees every tenant.

**Parameters**:
- `page[number]` (query, int, optional): Page number, 1-based. Default 1.
- `page[size]` (query, int, optional): Page size. Default 50, maximum 250.

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
        "minorVersion": 0,
        "environment": "string"
      }
    }
  ],
  "meta": {
    "total": 0,
    "page": {
      "number": 0,
      "size": 0,
      "last": 0
    }
  },
  "links": {
    "self": "string",
    "first": "string",
    "last": "string",
    "prev": "string",
    "next": "string"
  }
}
```

**Error Conditions**:
- 400: Invalid page[number] or page[size]
- 500: Internal server error

---

### GET /tenants/{tenantId}

Retrieves a tenant by ID. Scoped to the caller's environment; a caller with no environment sees every tenant.

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
      "minorVersion": 0,
      "environment": "string"
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 404: Tenant not found

---

### POST /tenants

Creates a new tenant. The `environment` attribute is not accepted from the request; the created tenant's environment is derived from the caller's context.

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
      "minorVersion": 0,
      "environment": "string"
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
      "minorVersion": 0,
      "environment": "string"
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 403: Caller's environment does not match the target tenant's environment
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
- 403: Caller's environment does not match the target tenant's environment
- 500: Internal server error (includes tenant not found)

---

### GET /tenants/{tenantId}/configurations/routes

Retrieves all routes for a tenant (paginated).

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `page[number]` (query, int, optional): Page number, 1-based. Default 1.
- `page[size]` (query, int, optional): Page size. Default 50, maximum 250.

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
  ],
  "meta": {
    "total": 0,
    "page": {
      "number": 0,
      "size": 0,
      "last": 0
    }
  },
  "links": {
    "self": "string",
    "first": "string",
    "last": "string",
    "prev": "string",
    "next": "string"
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or invalid page[number]/page[size]
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

### POST /tenants/configurations/routes/seed

Triggers asynchronous seed data loading for the tenant's routes from the shared seed catalog (`deploy/seed/shared/all/routes`, merged with any version-specific override root). Deletes all existing routes for the tenant and reloads them from catalog files. Unlike the CRUD endpoints above, the tenant is identified by request headers, not a `{tenantId}` path segment — this endpoint is mounted at `/tenants/configurations/routes/seed` (`configurations` is the *second* path segment here, not the third), which does not collide with the `{tenantId}`-scoped CRUD routes.

**Headers**:
- `TENANT_ID` (uuid, required): Tenant identifier
- `REGION` (string, required): Tenant region
- `MAJOR_VERSION` (uint16, required): Tenant major version
- `MINOR_VERSION` (uint16, required): Tenant minor version

**Parameters**: None

**Request Model**: None

**Response Model**: None. Seeding runs asynchronously in the background; poll `GET /tenants/configurations/routes/seed/status` for progress.

**Error Conditions**:
- 202: Seed operation started
- 400: Missing or invalid tenant headers

---

### GET /tenants/configurations/routes/seed/status

Retrieves seed catalog status for the routes seed group. Response is a plain JSON document, not a JSON:API resource.

**Headers**: Same as `POST /tenants/configurations/routes/seed`.

**Parameters**: None

**Request Model**: None

**Response Model**:
```json
{
  "groupName": "routes",
  "subdomains": {
    "routes": {
      "count": 0,
      "updatedAt": null
    }
  },
  "updatedAt": null,
  "catalogRevision": "string",
  "tenantSeededRevision": null,
  "tenantSeededAt": null
}
```

**Error Conditions**:
- 200: Status retrieved
- 400: Missing or invalid tenant headers
- 500: Internal server error (failure reading seed status)

---

### GET /tenants/{tenantId}/configurations/vessels

Retrieves all vessels for a tenant (paginated).

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `page[number]` (query, int, optional): Page number, 1-based. Default 1.
- `page[size]` (query, int, optional): Page size. Default 50, maximum 250.

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
  ],
  "meta": {
    "total": 0,
    "page": {
      "number": 0,
      "size": 0,
      "last": 0
    }
  },
  "links": {
    "self": "string",
    "first": "string",
    "last": "string",
    "prev": "string",
    "next": "string"
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or invalid page[number]/page[size]
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

### POST /tenants/configurations/vessels/seed

Triggers asynchronous seed data loading for the tenant's vessels from the shared seed catalog (`deploy/seed/shared/all/vessels`, merged with any version-specific override root). Deletes all existing vessels for the tenant and reloads them from catalog files. Unlike the CRUD endpoints above, the tenant is identified by request headers, not a `{tenantId}` path segment — this endpoint is mounted at `/tenants/configurations/vessels/seed` (`configurations` is the *second* path segment here, not the third), which does not collide with the `{tenantId}`-scoped CRUD routes.

**Headers**:
- `TENANT_ID` (uuid, required): Tenant identifier
- `REGION` (string, required): Tenant region
- `MAJOR_VERSION` (uint16, required): Tenant major version
- `MINOR_VERSION` (uint16, required): Tenant minor version

**Parameters**: None

**Request Model**: None

**Response Model**: None. Seeding runs asynchronously in the background; poll `GET /tenants/configurations/vessels/seed/status` for progress.

**Error Conditions**:
- 202: Seed operation started
- 400: Missing or invalid tenant headers

---

### GET /tenants/configurations/vessels/seed/status

Retrieves seed catalog status for the vessels seed group. Response is a plain JSON document, not a JSON:API resource.

**Headers**: Same as `POST /tenants/configurations/vessels/seed`.

**Parameters**: None

**Request Model**: None

**Response Model**:
```json
{
  "groupName": "vessels",
  "subdomains": {
    "vessels": {
      "count": 0,
      "updatedAt": null
    }
  },
  "updatedAt": null,
  "catalogRevision": "string",
  "tenantSeededRevision": null,
  "tenantSeededAt": null
}
```

**Error Conditions**:
- 200: Status retrieved
- 400: Missing or invalid tenant headers
- 500: Internal server error (failure reading seed status)

---

### GET /tenants/{tenantId}/configurations/instance-routes

Retrieves all instance routes for a tenant (paginated).

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `page[number]` (query, int, optional): Page number, 1-based. Default 1.
- `page[size]` (query, int, optional): Page size. Default 50, maximum 250.

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
  ],
  "meta": {
    "total": 0,
    "page": {
      "number": 0,
      "size": 0,
      "last": 0
    }
  },
  "links": {
    "self": "string",
    "first": "string",
    "last": "string",
    "prev": "string",
    "next": "string"
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or invalid page[number]/page[size]
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

### POST /tenants/configurations/instance-routes/seed

Triggers asynchronous seed data loading for the tenant's instance routes from the shared seed catalog (`deploy/seed/shared/all/instance-routes`, merged with any version-specific override root). Deletes all existing instance routes for the tenant and reloads them from catalog files. Unlike the CRUD endpoints above, the tenant is identified by request headers, not a `{tenantId}` path segment — this endpoint is mounted at `/tenants/configurations/instance-routes/seed` (`configurations` is the *second* path segment here, not the third), which does not collide with the `{tenantId}`-scoped CRUD routes.

**Headers**:
- `TENANT_ID` (uuid, required): Tenant identifier
- `REGION` (string, required): Tenant region
- `MAJOR_VERSION` (uint16, required): Tenant major version
- `MINOR_VERSION` (uint16, required): Tenant minor version

**Parameters**: None

**Request Model**: None

**Response Model**: None. Seeding runs asynchronously in the background; poll `GET /tenants/configurations/instance-routes/seed/status` for progress.

**Error Conditions**:
- 202: Seed operation started
- 400: Missing or invalid tenant headers

---

### GET /tenants/configurations/instance-routes/seed/status

Retrieves seed catalog status for the instance-routes seed group. Response is a plain JSON document, not a JSON:API resource.

**Headers**: Same as `POST /tenants/configurations/instance-routes/seed`.

**Parameters**: None

**Request Model**: None

**Response Model**:
```json
{
  "groupName": "instance-routes",
  "subdomains": {
    "instance-routes": {
      "count": 0,
      "updatedAt": null
    }
  },
  "updatedAt": null,
  "catalogRevision": "string",
  "tenantSeededRevision": null,
  "tenantSeededAt": null
}
```

**Error Conditions**:
- 200: Status retrieved
- 400: Missing or invalid tenant headers
- 500: Internal server error (failure reading seed status)

---

### GET /tenants/{tenantId}/configurations/rps-rewards

Retrieves all rps-rewards for a tenant (paginated).

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `page[number]` (query, int, optional): Page number, 1-based. Default 1.
- `page[size]` (query, int, optional): Page size. Default 50, maximum 250.

**Request Model**: None

**Response Model**:
```json
{
  "data": [
    {
      "type": "rps-rewards",
      "id": "string",
      "attributes": {
        "entryCostMeso": 0,
        "consolationMeso": 0,
        "ladder": [
          {
            "rung": 0,
            "itemId": 0,
            "quantity": 0,
            "meso": 0
          }
        ]
      }
    }
  ],
  "meta": {
    "total": 0,
    "page": {
      "number": 0,
      "size": 0,
      "last": 0
    }
  },
  "links": {
    "self": "string",
    "first": "string",
    "last": "string",
    "prev": "string",
    "next": "string"
  }
}
```

If no rps-rewards exist for the tenant, an empty array is returned instead of an error.

**Error Conditions**:
- 400: Invalid tenant ID format or invalid page[number]/page[size]
- 500: Internal server error

---

### GET /tenants/{tenantId}/configurations/rps-rewards/{rpsRewardId}

Retrieves an rps-reward by ID.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `rpsRewardId` (path, string): RPS reward identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "rps-rewards",
    "id": "string",
    "attributes": {
      "entryCostMeso": 0,
      "consolationMeso": 0,
      "ladder": [
        {
          "rung": 0,
          "itemId": 0,
          "quantity": 0,
          "meso": 0
        }
      ]
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or missing rps-reward ID
- 404: RPS reward not found

---

### POST /tenants/{tenantId}/configurations/rps-rewards

Creates a new rps-reward.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "rps-rewards",
    "attributes": {
      "entryCostMeso": 0,
      "consolationMeso": 0,
      "ladder": [
        {
          "rung": 0,
          "itemId": 0,
          "quantity": 0,
          "meso": 0
        }
      ]
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "rps-rewards",
    "id": "string",
    "attributes": {
      "entryCostMeso": 0,
      "consolationMeso": 0,
      "ladder": [
        {
          "rung": 0,
          "itemId": 0,
          "quantity": 0,
          "meso": 0
        }
      ]
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error

---

### PATCH /tenants/{tenantId}/configurations/rps-rewards/{rpsRewardId}

Updates an existing rps-reward.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `rpsRewardId` (path, string): RPS reward identifier

**Request Model**:
```json
{
  "data": {
    "type": "rps-rewards",
    "id": "string",
    "attributes": {
      "entryCostMeso": 0,
      "consolationMeso": 0,
      "ladder": [
        {
          "rung": 0,
          "itemId": 0,
          "quantity": 0,
          "meso": 0
        }
      ]
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "rps-rewards",
    "id": "string",
    "attributes": {
      "entryCostMeso": 0,
      "consolationMeso": 0,
      "ladder": [
        {
          "rung": 0,
          "itemId": 0,
          "quantity": 0,
          "meso": 0
        }
      ]
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body, tenant ID format, or missing rps-reward ID
- 500: Internal server error (includes rps-reward not found)

---

### DELETE /tenants/{tenantId}/configurations/rps-rewards/{rpsRewardId}

Deletes an rps-reward.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `rpsRewardId` (path, string): RPS reward identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format or missing rps-reward ID
- 500: Internal server error

---

### POST /tenants/{tenantId}/configurations/rps-rewards/seed

Deletes all existing rps-rewards for a tenant and loads them from seed files.

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

### GET /tenants/{tenantId}/configurations/mts-configs

Retrieves the MTS configuration for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "mts-configs",
    "id": "string",
    "attributes": {
      "listingFee": 0,
      "commissionRate": 0,
      "maxActiveListings": 0,
      "minLevel": 0,
      "auctionMinHours": 0,
      "auctionMaxHours": 0,
      "priceFloor": 0,
      "pageSize": 0,
      "minBidIncrement": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 404: No MTS configuration found for tenant

---

### GET /tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}

Retrieves an MTS configuration by ID.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `mtsConfigId` (path, string): MTS configuration identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "mts-configs",
    "id": "string",
    "attributes": {
      "listingFee": 0,
      "commissionRate": 0,
      "maxActiveListings": 0,
      "minLevel": 0,
      "auctionMinHours": 0,
      "auctionMaxHours": 0,
      "priceFloor": 0,
      "pageSize": 0,
      "minBidIncrement": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or missing MTS configuration ID
- 404: MTS configuration not found

---

### POST /tenants/{tenantId}/configurations/mts-configs

Creates a new MTS configuration.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "mts-configs",
    "attributes": {
      "listingFee": 0,
      "commissionRate": 0,
      "maxActiveListings": 0,
      "minLevel": 0,
      "auctionMinHours": 0,
      "auctionMaxHours": 0,
      "priceFloor": 0,
      "pageSize": 0,
      "minBidIncrement": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "mts-configs",
    "id": "string",
    "attributes": {
      "listingFee": 0,
      "commissionRate": 0,
      "maxActiveListings": 0,
      "minLevel": 0,
      "auctionMinHours": 0,
      "auctionMaxHours": 0,
      "priceFloor": 0,
      "pageSize": 0,
      "minBidIncrement": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error

---

### PATCH /tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}

Updates an existing MTS configuration.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `mtsConfigId` (path, string): MTS configuration identifier

**Request Model**:
```json
{
  "data": {
    "type": "mts-configs",
    "id": "string",
    "attributes": {
      "listingFee": 0,
      "commissionRate": 0,
      "maxActiveListings": 0,
      "minLevel": 0,
      "auctionMinHours": 0,
      "auctionMaxHours": 0,
      "priceFloor": 0,
      "pageSize": 0,
      "minBidIncrement": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "mts-configs",
    "id": "string",
    "attributes": {
      "listingFee": 0,
      "commissionRate": 0,
      "maxActiveListings": 0,
      "minLevel": 0,
      "auctionMinHours": 0,
      "auctionMaxHours": 0,
      "priceFloor": 0,
      "pageSize": 0,
      "minBidIncrement": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body, tenant ID format, or missing MTS configuration ID
- 500: Internal server error (includes MTS configuration not found)

---

### DELETE /tenants/{tenantId}/configurations/mts-configs/{mtsConfigId}

Deletes an MTS configuration.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `mtsConfigId` (path, string): MTS configuration identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format or missing MTS configuration ID
- 500: Internal server error

---

### POST /tenants/{tenantId}/configurations/mts-configs/seed

Deletes all existing MTS configurations for a tenant and loads them from seed files.

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

### GET /tenants/{tenantId}/configurations/trade-configs

Retrieves the player-to-player trade configuration for a tenant. One configuration per tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "trade-configs",
    "id": "string",
    "attributes": {
      "taxEnabled": true,
      "taxTiers": [
        {
          "threshold": 0,
          "rate": 0.0
        }
      ],
      "maxStagedItems": 0,
      "minTradeLevel": 0,
      "attestationTimeoutSeconds": 0
    }
  }
}
```

Every attribute is optional. On PATCH, an attribute omitted from the request body is left unchanged.

**Error Conditions**:
- 400: Invalid tenant ID format
- 404: No trade configuration found for tenant

---

### GET /tenants/{tenantId}/configurations/trade-configs/{tradeConfigId}

Retrieves a trade configuration by ID.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `tradeConfigId` (path, string): Trade configuration identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "trade-configs",
    "id": "string",
    "attributes": {
      "taxEnabled": true,
      "taxTiers": [
        {
          "threshold": 0,
          "rate": 0.0
        }
      ],
      "maxStagedItems": 0,
      "minTradeLevel": 0,
      "attestationTimeoutSeconds": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or missing trade configuration ID
- 404: Trade configuration not found

---

### POST /tenants/{tenantId}/configurations/trade-configs

Creates a new trade configuration.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "trade-configs",
    "attributes": {
      "taxEnabled": true,
      "taxTiers": [
        {
          "threshold": 0,
          "rate": 0.0
        }
      ],
      "maxStagedItems": 0,
      "minTradeLevel": 0,
      "attestationTimeoutSeconds": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "trade-configs",
    "id": "string",
    "attributes": {
      "taxEnabled": true,
      "taxTiers": [
        {
          "threshold": 0,
          "rate": 0.0
        }
      ],
      "maxStagedItems": 0,
      "minTradeLevel": 0,
      "attestationTimeoutSeconds": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error

---

### PATCH /tenants/{tenantId}/configurations/trade-configs/{tradeConfigId}

Updates an existing trade configuration.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `tradeConfigId` (path, string): Trade configuration identifier

**Request Model**:
```json
{
  "data": {
    "type": "trade-configs",
    "id": "string",
    "attributes": {
      "taxEnabled": true,
      "taxTiers": [
        {
          "threshold": 0,
          "rate": 0.0
        }
      ],
      "maxStagedItems": 0,
      "minTradeLevel": 0,
      "attestationTimeoutSeconds": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "trade-configs",
    "id": "string",
    "attributes": {
      "taxEnabled": true,
      "taxTiers": [
        {
          "threshold": 0,
          "rate": 0.0
        }
      ],
      "maxStagedItems": 0,
      "minTradeLevel": 0,
      "attestationTimeoutSeconds": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body, tenant ID format, or missing trade configuration ID
- 500: Internal server error (includes trade configuration not found)

---

### DELETE /tenants/{tenantId}/configurations/trade-configs/{tradeConfigId}

Deletes a trade configuration.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `tradeConfigId` (path, string): Trade configuration identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format or missing trade configuration ID
- 500: Internal server error

---

### POST /tenants/{tenantId}/configurations/trade-configs/seed

Deletes all existing trade configurations for a tenant and loads them from seed files.

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

### GET /tenants/{tenantId}/configurations/rankings

Retrieves the rankings configuration for a tenant. One configuration per tenant — there is no `/seed` endpoint and no `{rankingsId}` sub-resource.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "rankings",
    "id": "string",
    "attributes": {
      "recomputeIntervalMinutes": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 404: No rankings configuration found for tenant

---

### POST /tenants/{tenantId}/configurations/rankings

Creates the rankings configuration for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "rankings",
    "attributes": {
      "recomputeIntervalMinutes": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "rankings",
    "id": "string",
    "attributes": {
      "recomputeIntervalMinutes": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error

---

### PATCH /tenants/{tenantId}/configurations/rankings

Updates the existing rankings configuration for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "rankings",
    "attributes": {
      "recomputeIntervalMinutes": 0
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "rankings",
    "id": "string",
    "attributes": {
      "recomputeIntervalMinutes": 0
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 404: No rankings configuration found for tenant
- 500: Internal server error

---

### DELETE /tenants/{tenantId}/configurations/rankings

Deletes the rankings configuration for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format
- 500: Internal server error

---

### GET /tenants/{tenantId}/configurations/kite-configs

Retrieves the kite (cash-shop item category 508 message-box) placement
configuration for a tenant. One configuration per tenant — there is no
`/seed` endpoint and no `{kiteConfigId}` sub-resource, matching the
`rankings` resource shape.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "kite-configs",
    "id": "string",
    "attributes": {
      "maxPerMap": 10,
      "maxMessageLength": 182,
      "blockedMapPrefixes": [91]
    }
  }
}
```

- `maxPerMap` (int): Maximum number of kites simultaneously visible on a
  single map. Default `10`.
- `maxMessageLength` (int): Maximum character length of a kite'''s message
  text. Default `182`.
- `blockedMapPrefixes` (array of uint32): Map-id prefixes (the leading digit
  group of a map id, e.g. `91` for the `9100000`-`9199999` cash-shop/event
  range) where kites may not be placed. Default `[91]`.

**Error Conditions**:
- 400: Invalid tenant ID format
- 404: No kite-configs configuration found for tenant

---

### POST /tenants/{tenantId}/configurations/kite-configs

Creates the kite-configs configuration for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "kite-configs",
    "attributes": {
      "maxPerMap": 10,
      "maxMessageLength": 182,
      "blockedMapPrefixes": [91]
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "kite-configs",
    "id": "string",
    "attributes": {
      "maxPerMap": 10,
      "maxMessageLength": 182,
      "blockedMapPrefixes": [91]
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error

---

### PATCH /tenants/{tenantId}/configurations/kite-configs

Updates the existing kite-configs configuration for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "kite-configs",
    "attributes": {
      "maxPerMap": 10,
      "maxMessageLength": 182,
      "blockedMapPrefixes": [91]
    }
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "kite-configs",
    "id": "string",
    "attributes": {
      "maxPerMap": 10,
      "maxMessageLength": 182,
      "blockedMapPrefixes": [91]
    }
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 404: No kite-configs configuration found for tenant
- 500: Internal server error

---

### DELETE /tenants/{tenantId}/configurations/kite-configs

Deletes the kite-configs configuration for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format
- 500: Internal server error

---

### GET /tenants/{tenantId}/configurations/imprint-configs

Retrieves the imprint configuration for a tenant.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "imprint-configs",
    "id": "string",
    "pendingExpiryHours": 0
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format
- 404: No imprint configuration found for tenant

---

### GET /tenants/{tenantId}/configurations/imprint-configs/{imprintConfigId}

Retrieves an imprint configuration by ID.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `imprintConfigId` (path, string): Imprint configuration identifier

**Request Model**: None

**Response Model**:
```json
{
  "data": {
    "type": "imprint-configs",
    "id": "string",
    "pendingExpiryHours": 0
  }
}
```

**Error Conditions**:
- 400: Invalid tenant ID format or missing imprint configuration ID
- 404: Imprint configuration not found

---

### POST /tenants/{tenantId}/configurations/imprint-configs

Creates a new imprint configuration.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier

**Request Model**:
```json
{
  "data": {
    "type": "imprint-configs",
    "pendingExpiryHours": 0
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "imprint-configs",
    "id": "string",
    "pendingExpiryHours": 0
  }
}
```

**Error Conditions**:
- 400: Invalid request body or tenant ID format
- 500: Internal server error

---

### PATCH /tenants/{tenantId}/configurations/imprint-configs/{imprintConfigId}

Updates an existing imprint configuration.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `imprintConfigId` (path, string): Imprint configuration identifier

**Request Model**:
```json
{
  "data": {
    "type": "imprint-configs",
    "id": "string",
    "pendingExpiryHours": 0
  }
}
```

**Response Model**:
```json
{
  "data": {
    "type": "imprint-configs",
    "id": "string",
    "pendingExpiryHours": 0
  }
}
```

**Error Conditions**:
- 400: Invalid request body, tenant ID format, or missing imprint configuration ID
- 500: Internal server error (includes imprint configuration not found)

---

### DELETE /tenants/{tenantId}/configurations/imprint-configs/{imprintConfigId}

Deletes an imprint configuration.

**Parameters**:
- `tenantId` (path, uuid): Tenant identifier
- `imprintConfigId` (path, string): Imprint configuration identifier

**Request Model**: None

**Response Model**: None (204 No Content)

**Error Conditions**:
- 400: Invalid tenant ID format or missing imprint configuration ID
- 500: Internal server error

---

### POST /tenants/{tenantId}/configurations/imprint-configs/seed

Deletes all existing imprint configurations for a tenant and loads them from seed files.

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
