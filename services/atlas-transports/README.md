# atlas-transports
MapleStory Transport Route Service

## Overview

A Golang service to manage transportation routes within a MapleStory private server. The system simulates travel via ships or similar transports, allowing players to move between maps on a timed schedule.

## Features

- Manages repeatable transportation schedules across maps
- Supports shared-vessel back-and-forth simulation
- Exposes real-time route state via REST API
- Precomputes and exposes a schedule per route that works irrespective of the day
- Uses local server time for all scheduling logic
- All routes share a default schedule alignment, starting at midnight (00:00)
- Emits Kafka events for state transitions

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- REST_PORT - The port for the REST API server (default: 8080)
- BOOTSTRAP_SERVERS - Comma-separated list of Kafka bootstrap servers
- ROUTE_STATE_TOPIC - Kafka topic for route state transitions

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

All endpoints follow [jsonapi.org](https://jsonapi.org) conventions, except error responses which use only HTTP status codes.

#### `GET /transports/routes`

Returns all routes for the tenant.

**Query Parameters:**
- `filter[startMapId]` (optional): Filter routes by starting map ID

Example request:
```
GET /transports/routes
GET /transports/routes?filter[startMapId]=101000300
```

Example response:
```json
{
  "data": [
    {
      "type": "routes",
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "attributes": {
        "name": "Ellinia Ferry",
        "startMapId": 101000300,
        "stagingMapId": 200090000,
        "enRouteMapIds": [200090100],
        "destinationMapId": 200000100,
        "observationMapId": 200090010,
        "state": "open_entry"
      }
    }
  ]
}
```

#### `GET /transports/routes/{routeId}`

Returns metadata about a single route.

Example response:
```json
{
  "data": {
    "type": "routes",
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "attributes": {
      "name": "Ellinia Ferry",
      "startMapId": 101000300,
      "stagingMapId": 200090000,
      "enRouteMapIds": [200090100],
      "destinationMapId": 200000100,
      "observationMapId": 200090010,
      "state": "awaiting_return"
    }
  }
}
```

## Route State Machine

Each route transitions through the following states (from the perspective of the starting map):

- `out_of_service` - no scheduled trips available
- `awaiting_return` - vessel is not yet available
- `open_entry` - players can board
- `locked_entry` - boarding closed, pre-departure phase
- `in_transit` - characters are in the en-route map

## Kafka Events

The service emits events on state transitions:

- **Arrived Event**: Emitted when a route transitions to `open_entry` state
- **Departed Event**: Emitted when a route transitions to `in_transit` state

## Sample Routes

The service supports routes such as:

1. Ellinia to Orbis Ferry
2. Orbis to Ellinia Ferry
3. Ludibrium to Orbis Train
4. Orbis to Ludibrium Train

And shared vessels:

1. Ellinia-Orbis Ferry (shared vessel)
2. Ludibrium-Orbis Train (shared vessel)

## TODOs

- Add dynamic route reloading
- Add activation windows
- Add rate limiting and concurrency protection
