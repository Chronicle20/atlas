# REST API

## Endpoints

### GET /transports/routes

Returns all scheduled routes for the tenant.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| filter[startMapId] | query | uint32 | No | Filter routes by starting map ID |

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

Returns a single scheduled route with its trip schedule.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| routeId | path | uuid.UUID | Yes | Route identifier |

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

### GET /transports/instance-routes

Returns all instance routes for the tenant.

**Response Model:**

Resource type: `instance-routes`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Route identifier |
| name | string | Route name |
| startMapId | map.Id | Starting map ID |
| transitMapIds | []map.Id | Transit map IDs |
| destinationMapId | map.Id | Destination map ID |
| capacity | uint32 | Maximum characters per instance |
| boardingWindow | time.Duration | Boarding window duration |
| travelDuration | time.Duration | Travel duration |

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 500 | Internal server error retrieving instance routes |

---

### GET /transports/instance-routes/{routeId}

Returns a single instance route.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| routeId | path | uuid.UUID | Yes | Route identifier |

**Response Model:**

Resource type: `instance-routes`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Route identifier |
| name | string | Route name |
| startMapId | map.Id | Starting map ID |
| transitMapIds | []map.Id | Transit map IDs |
| destinationMapId | map.Id | Destination map ID |
| capacity | uint32 | Maximum characters per instance |
| boardingWindow | time.Duration | Boarding window duration |
| travelDuration | time.Duration | Travel duration |

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 404 | Route not found |
| 500 | Internal server error retrieving instance route |

---

### GET /transports/instance-routes/{routeId}/status

Returns active instance statuses for a route.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| routeId | path | uuid.UUID | Yes | Route identifier |

**Response Model:**

Resource type: `instance-status`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Instance identifier |
| routeId | uuid.UUID | Route identifier |
| state | string | Instance state (boarding, in_transit) |
| characters | int | Number of characters in instance |
| boardingUntil | string | Boarding window expiry (RFC3339) |
| arrivalAt | string | Arrival time (RFC3339) |

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 500 | Internal server error retrieving instance status |

---

### POST /transports/instance-routes/{routeId}/start

Starts an instance transport for a character on the specified route.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| routeId | path | uuid.UUID | Yes | Route identifier |

**Request Model:**

Resource type: `start-transport`

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |

**Response:** 204 No Content on success.

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid routeId, character already in transport, or route not found |

## Related Resource Types

### trip-schedule

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Trip identifier |
| boardingOpen | time.Time | Boarding open time |
| boardingClosed | time.Time | Boarding closed time |
| departure | time.Time | Departure time |
| arrival | time.Time | Arrival time |
