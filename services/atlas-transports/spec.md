# MapleStory Transport Route Service Specification

## Overview

Develop a Golang service to manage transportation routes within a MapleStory private server. The system simulates travel via ships or similar transports, allowing players to move between maps on a timed schedule. It supports predictable, repeated trips, including shared vessels for bidirectional routes, and exposes route states and trip schedules via a JSON:API-compatible REST API.

## Project Goals

- Manage repeatable transportation schedules across maps
- Support shared-vessel back-and-forth simulation
- Expose real-time route state via REST
- Precompute and expose a daily schedule per route
- Use local server time for all scheduling logic
- All routes share a default schedule alignment, starting at midnight (00:00) of the current day

## 1. Data Models

### `TransportRoute`

```go
type TransportRoute struct {
    ID                     string        // UUID format
    Name                   string
    StartMapID             uint32
    StagingMapID           uint32
    EnRouteMapID           uint32
    DestinationMapID       uint32
    BoardingWindowDuration time.Duration
    PreDepartureDuration   time.Duration
    TravelDuration         time.Duration
    CycleInterval          time.Duration
}
```

### `SharedVessel`

```go
type SharedVessel struct {
    ID              string // UUID format
    RouteAID        string
    RouteBID        string
    TurnaroundDelay time.Duration
}
```

## 2. Schedule and Trip Handling

- Schedule is precomputed at service startup for the current **UTC** day
- All trips are aligned to midnight (00:00) **local server time**
- Only trips fully contained within that UTC day are included
- Trip IDs use deterministic format: `{routeID}_{departureTimestamp}`

### `TripSchedule`

```go
type TripSchedule struct {
    TripID         string
    RouteID        string
    BoardingOpen   time.Time
    BoardingClosed time.Time
    Departure      time.Time
    Arrival        time.Time
}
```

## 3. Route State Machine

Each route transitions through the following states (from the perspective of the starting map):

- `awaiting_return` – vessel is not yet available
- `open_entry` – players can board
- `locked_entry` – boarding closed, pre-departure phase
- `in_transit` – characters are in the en-route map

## 4. REST API

All endpoints follow [jsonapi.org](https://jsonapi.org) conventions, except error responses which use only HTTP status codes.

### `GET /routes/:id`

Returns metadata about a single route.

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

### `GET /routes/:id/state`

Returns current state of a route.

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

### `GET /routes/:id/schedule`

Returns the full schedule of precomputed trips for the current UTC day.

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

### Error Responses

- Use HTTP status codes only (`404`, `500`, etc.)
- Response bodies are optional and minimal

## 5. System Behavior

- Uses local server time for all internal calculations
- Schedule and route state are fully recomputed on startup
- No dynamic route reloading
- No activation windows
- No rate limiting or concurrency protection is needed

## 6. Game Server Integration (TODOs)

Integration with the game server is not implemented but should be stubbed:

```go
// TODO: Warp characters from staging map to en-route map  
// TODO: Broadcast messages to players during transitions
```

## 7. Sample Route Loader

No external configuration. All route definitions come from code:

```go
func LoadSampleRoutes() ([]TransportRoute, []SharedVessel)
```

## 8. Kafka Integration for State Transitions

Every time a route changes state, emit a Kafka event.

### Kafka Event Format

```json
{
  "routeId": "ellinia_to_orbis",
  "status": "locked_entry",
  "timestamp": "2025-06-20T12:14:00Z"
}
```

### Kafka Requirements

- Kafka broker configuration should be externalized (e.g., via environment variables)
- Topic name should be configurable (default: `route.state.transitions`)
- Kafka production errors should be logged but **must not crash** the service
- Use a lightweight Kafka client (e.g., `segmentio/kafka-go`)
- Add a `TODO` in code to mock this if Kafka is not connected

## 9. Deliverables

### `transport` Go package:

- Scheduler
- State machine
- REST handlers
- Kafka producer integration

### `main.go`

- Starts the service
- Runs the REST server
- Console logs of state transitions
- REST API endpoints for route data, state, and schedule
- Kafka event emission on state changes

### Other

- `README.md` with setup and usage instructions
- `TODO`s for future integration points with game server