# REST API

## Endpoints

### GET /transports/routes

Returns all scheduled routes for the tenant.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| filter[startMapId] | query | uint32 | No | Filter routes by starting map ID |
| include | query | string | No | Comma-separated relationship names. `schedule` attaches the day's trip rows; omitted by default (see note) |
| page[number] | query | int | No | Page number (default 1) |
| page[size] | query | int | No | Page size (default 50, max 250) |

**Response Model:**

Resource type: `routes`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Route identifier (tenant-derived, stable across restarts and replicas) |
| name | string | Route name |
| startMapId | map.Id | Starting map ID |
| stagingMapId | map.Id | Staging map ID |
| enRouteMapIds | []map.Id | En-route map IDs |
| destinationMapId | map.Id | Destination map ID |
| observationMapId | map.Id | Observation map ID |
| state | string | Current route state |
| cycleInterval | time.Duration | **Legacy.** Serialises as an integer nanosecond count. Superseded by `cycleIntervalSeconds` |
| boardingWindowSeconds | uint32 | Boarding window, in seconds |
| preDepartureSeconds | uint32 | Pre-departure hold, in seconds |
| travelDurationSeconds | uint32 | Travel duration, in seconds |
| cycleIntervalSeconds | uint32 | Cycle interval, in seconds |
| nextTransitionAt | string | Absolute instant (RFC3339) of the next state change; empty when `out_of_service` |
| nextState | string | The state the route moves to at `nextTransitionAt`; empty when `out_of_service` |
| voyageId | string | Identity of the trip currently under way (`transport.VoyageId`); omitted unless `state` is `in_transit` |

**Schedule is opt-in.** `Transform` attaches a full day of trip rows (~96 per
route on a 15-minute cycle), so a twelve-route list would carry ~1,000 included
resources. The list endpoint therefore uses a summary transform by default and
attaches the `schedule` relationship only when `include=schedule` is passed.
The detail endpoint always attaches it. Sparse fieldsets cannot express this:
api2go's `FilterSparseFields` rewrites each `included` entry's attributes and
never removes an entry, and an empty field list is a 400.

**Time semantics.** The day's schedule is computed once per reconcile and the
1-second ticker only re-derives state from it, comparing *time of day* only.
Trip-schedule timestamps therefore carry the computing day's date, and only
their time-of-day component is meaningful. `nextTransitionAt` exists so clients
never have to reconstruct that: it is the governing boundary projected onto the
first instant after the server's `now`. `state` and `nextState`/`nextTransitionAt`
come from a single evaluation on a single `now` and cannot disagree.

**Relationships:**

| Name | Type | Cardinality |
|------|------|-------------|
| schedule | trip-schedule | to-many |

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number]/page[size] parameter |
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
| id | uuid.UUID | Route identifier (tenant-derived, stable across restarts and replicas) |
| name | string | Route name |
| startMapId | map.Id | Starting map ID |
| stagingMapId | map.Id | Staging map ID |
| enRouteMapIds | []map.Id | En-route map IDs |
| destinationMapId | map.Id | Destination map ID |
| observationMapId | map.Id | Observation map ID |
| state | string | Current route state |
| cycleInterval | time.Duration | **Legacy.** Serialises as an integer nanosecond count. Superseded by `cycleIntervalSeconds` |
| boardingWindowSeconds | uint32 | Boarding window, in seconds |
| preDepartureSeconds | uint32 | Pre-departure hold, in seconds |
| travelDurationSeconds | uint32 | Travel duration, in seconds |
| cycleIntervalSeconds | uint32 | Cycle interval, in seconds |
| nextTransitionAt | string | Absolute instant (RFC3339) of the next state change; empty when `out_of_service` |
| nextState | string | The state the route moves to at `nextTransitionAt`; empty when `out_of_service` |
| voyageId | string | Identity of the trip currently under way (`transport.VoyageId`); omitted unless `state` is `in_transit` |

**Time semantics.** The day's schedule is computed once per reconcile and the
1-second ticker only re-derives state from it, comparing *time of day* only.
Trip-schedule timestamps therefore carry the computing day's date, and only
their time-of-day component is meaningful. `nextTransitionAt` exists so clients
never have to reconstruct that: it is the governing boundary projected onto the
first instant after the server's `now`. `state` and `nextState`/`nextTransitionAt`
come from a single evaluation on a single `now` and cannot disagree.

The schedule relationship is always attached on this endpoint.

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

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| page[number] | query | int | No | Page number (default 1) |
| page[size] | query | int | No | Page size (default 50, max 250) |

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
| boardingWindow | time.Duration | **Legacy.** Integer nanosecond count. Superseded by `boardingWindowSeconds` |
| travelDuration | time.Duration | **Legacy.** Integer nanosecond count. Superseded by `travelDurationSeconds` |
| boardingWindowSeconds | uint32 | Boarding window, in seconds |
| travelDurationSeconds | uint32 | Travel duration, in seconds |

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number]/page[size] parameter |
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
| boardingWindow | time.Duration | **Legacy.** Integer nanosecond count. Superseded by `boardingWindowSeconds` |
| travelDuration | time.Duration | **Legacy.** Integer nanosecond count. Superseded by `travelDurationSeconds` |
| boardingWindowSeconds | uint32 | Boarding window, in seconds |
| travelDurationSeconds | uint32 | Travel duration, in seconds |

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
| page[number] | query | int | No | Page number (default 1) |
| page[size] | query | int | No | Page size (default 50, max 250) |

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
| createdAt | string | Instance creation instant (RFC3339). The stuck-timeout sweep force-warps when `now - createdAt` exceeds the route's `MaxLifetime()` = `2 × (boardingWindow + travelDuration)` |

**Tenant scoping.** Instances are stored in a per-route, tenant-keyed Redis set
and are read back under the tenant the request carries. A tenant only ever sees
its own instances.

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number]/page[size] parameter |
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

**Time semantics.** These four timestamps carry the date of the day the schedule
was computed; only their time-of-day component is meaningful (see the Time
semantics note under `GET /transports/routes` above).

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Trip identifier |
| boardingOpen | time.Time | Boarding open time |
| boardingClosed | time.Time | Boarding closed time |
| departure | time.Time | Departure time |
| arrival | time.Time | Arrival time |
