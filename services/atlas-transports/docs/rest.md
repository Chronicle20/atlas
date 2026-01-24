# REST API

## Endpoints

### GET /transports/routes

Returns all routes for the tenant.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| filter[startMapId] | query | uint32 | No | Filter routes by starting map ID |

**Request Headers:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| TENANT_ID | string | Yes | Tenant identifier |
| REGION | string | Yes | Region code |
| MAJOR_VERSION | uint16 | Yes | Major version |
| MINOR_VERSION | uint16 | Yes | Minor version |

**Response Model:**

Resource type: `routes`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Route identifier |
| name | string | Route name |
| startMapId | map.Id | Starting map ID |
| stagingMapId | map.Id | Staging map ID |
| enRouteMapIds | []map.Id | En-route map IDs |
| destinationMapId | map.Id | Destination map ID |
| observationMapId | map.Id | Observation map ID |
| state | string | Current route state |
| cycleInterval | time.Duration | Cycle interval |

**Relationships:**

| Name | Type | Cardinality |
|------|------|-------------|
| schedule | trip-schedule | to-many |

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid filter[startMapId] parameter |
| 500 | Internal server error retrieving routes |

---

### GET /transports/routes/{routeId}

Returns metadata about a single route.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| routeId | path | uuid.UUID | Yes | Route identifier |

**Request Headers:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| TENANT_ID | string | Yes | Tenant identifier |
| REGION | string | Yes | Region code |
| MAJOR_VERSION | uint16 | Yes | Major version |
| MINOR_VERSION | uint16 | Yes | Minor version |

**Response Model:**

Resource type: `routes`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Route identifier |
| name | string | Route name |
| startMapId | map.Id | Starting map ID |
| stagingMapId | map.Id | Staging map ID |
| enRouteMapIds | []map.Id | En-route map IDs |
| destinationMapId | map.Id | Destination map ID |
| observationMapId | map.Id | Observation map ID |
| state | string | Current route state |
| cycleInterval | time.Duration | Cycle interval |

**Relationships:**

| Name | Type | Cardinality |
|------|------|-------------|
| schedule | trip-schedule | to-many |

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 500 | Route not found or internal server error |

---

### POST /transports/routes/seed

Seeds routes from JSON configuration files.

**Request Headers:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| TENANT_ID | string | Yes | Tenant identifier |
| REGION | string | Yes | Region code |
| MAJOR_VERSION | uint16 | Yes | Major version |
| MINOR_VERSION | uint16 | Yes | Minor version |

**Response Model:**

| Field | Type | Description |
|-------|------|-------------|
| deletedRoutes | int | Number of routes deleted |
| createdRoutes | int | Number of routes created |
| failedCount | int | Number of routes that failed to load |
| errors | []string | Error messages |

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 500 | Internal server error during seeding |

## Related Resource Types

### trip-schedule

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Trip identifier |
| boardingOpen | time.Time | Boarding open time |
| boardingClosed | time.Time | Boarding closed time |
| departure | time.Time | Departure time |
| arrival | time.Time | Arrival time |
