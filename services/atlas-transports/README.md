# atlas-transports
MapleStory Transport Route Service

## Overview

A Golang service to manage transportation routes within a MapleStory private server. The system simulates travel via ships or similar transports, allowing players to move between maps on a timed schedule.

## Features

- Manages repeatable transportation schedules across maps
- Supports shared-vessel back-and-forth simulation
- Exposes real-time route state via REST API
- Precomputes and exposes a daily schedule per route
- Uses local server time for all scheduling logic
- All routes share a default schedule alignment, starting at midnight (00:00) of the current day

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- REST_PORT - The port for the REST API server (default: 8080)
- BOOTSTRAP_SERVERS - Comma-separated list of Kafka bootstrap servers (for future Kafka integration)
- ROUTE_STATE_TOPIC - Kafka topic for route state transitions (default: "route.state.transitions")

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

#### `GET /routes/:id`

Returns metadata about a single route.

Example response:
```json
{
  "data": {
    "type": "route",
    "id": "ellinia_to_orbis",
    "attributes": {
      "name": "Ellinia Ferry",
      "startMapId": 101000300,
      "stagingMapId": 200090000,
      "enRouteMapId": 200090100,
      "destinationMapId": 200000100,
      "cycleInterval": "10m"
    }
  }
}
```

#### `GET /routes/:id/state`

Returns current state of a route.

Example response:
```json
{
  "data": {
    "type": "route-state",
    "id": "ellinia_to_orbis",
    "attributes": {
      "status": "locked_entry",
      "nextDeparture": "2025-06-20T12:15:00Z",
      "boardingEnds": "2025-06-20T12:14:00Z"
    }
  }
}
```

#### `GET /routes/:id/schedule`

Returns the full schedule of precomputed trips for the current UTC day.

Example response:
```json
{
  "data": [
    {
      "type": "trip-schedule",
      "id": "ellinia_to_orbis_20250620T100000",
      "attributes": {
        "boardingOpen": "2025-06-20T10:00:00Z",
        "boardingClosed": "2025-06-20T10:01:00Z",
        "departure": "2025-06-20T10:01:10Z",
        "arrival": "2025-06-20T10:02:40Z"
      }
    }
  ]
}
```

## Route State Machine

Each route transitions through the following states (from the perspective of the starting map):

- `awaiting_return` – vessel is not yet available
- `open_entry` – players can board
- `locked_entry` – boarding closed, pre-departure phase
- `in_transit` – characters are in the en-route map

## Sample Routes

The service includes the following sample routes:

1. Ellinia to Orbis Ferry
2. Orbis to Ellinia Ferry
3. Ludibrium to Orbis Train
4. Orbis to Ludibrium Train

And the following shared vessels:

1. Ellinia-Orbis Ferry (shared vessel)
2. Ludibrium-Orbis Train (shared vessel)

## Future Integrations

The following integrations are planned for future development:

- Kafka integration for state transitions
- Game server integration for warping characters and broadcasting messages

## TODOs

- Implement Kafka integration for state transitions
- Implement game server integration for warping characters
- Implement game server integration for broadcasting messages
- Add dynamic route reloading
- Add activation windows
- Add rate limiting and concurrency protection
