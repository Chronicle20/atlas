# Transport Domain

## Responsibility

Manages transport route scheduling and state transitions for in-game transportation systems. The domain tracks route state based on time-of-day scheduling and coordinates character warping between maps during transport operations.

## Core Models

### Model (transport/model.go)

Represents a transport route with scheduling configuration.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Route identifier |
| name | string | Route name |
| startMapId | map.Id | Starting map ID |
| stagingMapId | map.Id | Staging map ID (boarding area) |
| enRouteMapIds | []map.Id | Maps traversed during transit |
| destinationMapId | map.Id | Destination map ID |
| observationMapId | map.Id | Map for observing transport status |
| state | RouteState | Current route state |
| schedule | []TripScheduleModel | Precomputed trip schedule |
| boardingWindowDuration | time.Duration | Duration boarding is open |
| preDepartureDuration | time.Duration | Duration between boarding close and departure |
| travelDuration | time.Duration | Duration of transit |
| cycleInterval | time.Duration | Interval between trips |

### SharedVesselModel (transport/model.go)

Represents a shared vessel operating on two routes alternately.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Vessel identifier |
| name | string | Vessel name |
| routeAID | uuid.UUID | First route ID |
| routeBID | uuid.UUID | Second route ID |
| turnaroundDelay | time.Duration | Delay between route arrivals |

### TripScheduleModel (transport/model.go)

Represents a single scheduled trip.

| Field | Type | Description |
|-------|------|-------------|
| tripId | uuid.UUID | Trip identifier |
| routeId | uuid.UUID | Associated route ID |
| boardingOpen | time.Time | Time boarding opens |
| boardingClosed | time.Time | Time boarding closes |
| departure | time.Time | Departure time |
| arrival | time.Time | Arrival time |

### RouteState (transport/state.go)

Enumeration of route states.

| Value | Description |
|-------|-------------|
| out_of_service | No scheduled trips available |
| awaiting_return | Vessel is not yet available |
| open_entry | Players can board |
| locked_entry | Boarding closed, pre-departure phase |
| in_transit | Characters are in the en-route map |

## Invariants

- Route name must not be empty
- At least one en-route map ID is required
- Boarding window duration must be positive
- Pre-departure duration must be positive
- Travel duration must be positive
- Cycle interval must be positive
- Route A ID and Route B ID must not be nil for shared vessels
- Turnaround delay must be positive for shared vessels
- Boarding open must be before boarding closed
- Boarding closed must be before departure
- Departure must be before arrival

## State Transitions

Routes transition through states based on the current time of day relative to the trip schedule:

```
out_of_service -> awaiting_return (when next trip schedule exists)
awaiting_return -> open_entry (when boarding window opens)
open_entry -> locked_entry (when boarding window closes)
locked_entry -> in_transit (when departure time is reached)
in_transit -> awaiting_return (when arrival time is reached)
```

State transitions trigger character warping:
- Transition to `awaiting_return`: Characters in en-route maps are warped to destination map
- Transition to `in_transit`: Characters in staging map are warped to first en-route map

## Processors

### transport.Processor (transport/processor.go)

| Method | Description |
|--------|-------------|
| AddTenant | Adds routes with computed schedules for a tenant |
| ClearTenant | Removes all routes for a tenant |
| ByIdProvider | Returns a provider for a route by ID |
| ByStartMapProvider | Returns a provider for a route by start map ID |
| GetByStartMap | Returns a route by its start map ID |
| AllRoutesProvider | Returns a provider for all routes |
| UpdateRoutes | Updates state for all routes |
| UpdateRouteAndEmit | Updates route state and emits Kafka events |
| WarpToRouteStartMapOnLogout | Warps character to route start map on logout |
| WarpToRouteStartMapOnLogoutAndEmit | Warps character and emits Kafka events |

### seed.Processor (seed/processor.go)

| Method | Description |
|--------|-------------|
| Seed | Loads route configurations from JSON files and seeds into registry |

### config.Processor (transport/config/processor.go)

| Method | Description |
|--------|-------------|
| GetRoutes | Returns all routes for a tenant from external configuration service |
| GetVessels | Returns all vessels for a tenant from external configuration service |
| LoadConfigurationsForTenant | Loads routes and vessels for a tenant |

### channel.Processor (channel/processor.go)

| Method | Description |
|--------|-------------|
| Register | Registers a channel for a tenant |
| Unregister | Unregisters a channel for a tenant |
| GetAll | Returns all registered channels for a tenant |

### character.Processor (character/processor.go)

| Method | Description |
|--------|-------------|
| WarpRandom | Warps character to random spawn point in field |
| WarpRandomAndEmit | Warps character and emits Kafka events |
| WarpToPortal | Warps character to specific portal in field |

### portal.Processor (data/portal/processor.go)

| Method | Description |
|--------|-------------|
| InMapProvider | Returns provider for all portals in a map |
| RandomSpawnPointProvider | Returns provider for random spawn point portal |
| RandomSpawnPointIdProvider | Returns provider for random spawn point portal ID |

### map.Processor (map/processor.go)

| Method | Description |
|--------|-------------|
| CharacterIdsInMapProvider | Returns provider for character IDs in a map |

### tenant.Processor (tenant/processor.go)

| Method | Description |
|--------|-------------|
| AllProvider | Returns provider for all tenants |
| GetAll | Returns all tenants |

## Scheduler (transport/scheduler.go)

Computes trip schedules for routes and shared vessels. Independent routes are scheduled with fixed cycle intervals starting at midnight. Shared vessels alternate between routes with turnaround delays between arrivals.
