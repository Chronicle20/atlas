# Transport Domain

## Responsibility

Manages transport route scheduling and state transitions for in-game transportation systems. The domain supports two transport models: scheduled routes with time-of-day based state transitions, and instance-based routes with on-demand ephemeral transport instances.

## Core Models

### Model (transport/model.go)

Represents a scheduled transport route with scheduling configuration.

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
| id | string | Vessel identifier |
| name | string | Vessel name |
| routeAID | string | Route A name |
| routeBID | string | Route B name |
| turnaroundDelay | time.Duration | Delay between arrival and next departure |

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

Enumeration of scheduled route states.

| Value | Description |
|-------|-------------|
| out_of_service | No scheduled trips available |
| awaiting_return | Vessel is not yet available |
| open_entry | Players can board |
| locked_entry | Boarding closed, pre-departure phase |
| in_transit | Characters are in the en-route map |

### RouteModel (instance/model.go)

Represents an instance-based transport route configuration.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Route identifier |
| name | string | Route name |
| startMapId | map.Id | Starting map ID |
| transitMapIds | []map.Id | Transit map IDs |
| destinationMapId | map.Id | Destination map ID |
| capacity | uint32 | Maximum characters per instance |
| boardingWindow | time.Duration | Duration boarding is open for each instance |
| travelDuration | time.Duration | Duration of transit |
| transitMessage | string | Message displayed during transit |

### TransportInstance (instance/model.go)

Represents an active ephemeral instance of an instance-based route.

| Field | Type | Description |
|-------|------|-------------|
| instanceId | uuid.UUID | Instance identifier (on-demand UUID) |
| routeId | uuid.UUID | Associated route ID |
| tenantId | uuid.UUID | Tenant identifier |
| characters | []CharacterEntry | Characters in this instance |
| state | InstanceState | Current instance state |
| boardingUntil | time.Time | Boarding window expiry time |
| arrivalAt | time.Time | Arrival time |
| createdAt | time.Time | Creation time |

`MaxLifetime()` returns `2 * (boardingWindow + travelDuration)`.

### CharacterEntry (instance/model.go)

Tracks a character and their field context within an instance.

| Field | Type | Description |
|-------|------|-------------|
| CharacterId | uint32 | Character identifier |
| WorldId | world.Id | World identifier |
| ChannelId | channel.Id | Channel identifier |

### InstanceState (instance/state.go)

Enumeration of instance transport states.

| Value | Description |
|-------|-------------|
| Boarding | Characters can board the instance |
| InTransit | Instance is in transit |

## Invariants

### Scheduled Routes
- Route name must not be empty
- At least one en-route map ID is required
- Boarding window duration must be positive
- Pre-departure duration must not be negative
- Travel duration must be positive
- Cycle interval must be positive
- Route A ID and Route B ID must not be empty for shared vessels
- Turnaround delay must be positive for shared vessels
- Route ID must not be nil for trip schedules
- Boarding open must be before boarding closed
- Departure must not be before boarding closed
- Departure must be before arrival

### Instance Routes
- Route name must not be empty
- Capacity must be greater than zero
- Boarding window must be positive
- Travel duration must not be negative
- Transit map IDs must not be empty

## State Transitions

### Scheduled Routes

Routes transition through states based on the current time of day relative to the trip schedule:

```
out_of_service -> awaiting_return (when next trip schedule exists)
awaiting_return -> open_entry (when boarding window opens)
open_entry -> locked_entry (when boarding window closes)
locked_entry -> in_transit (when departure time is reached)
in_transit -> awaiting_return (when arrival time is reached)
```

State transitions trigger character warping:
- Transition to `awaiting_return` from `in_transit`: Characters in en-route maps are warped to destination map
- Transition to `open_entry`: Emits ARRIVED event on the observation map
- Transition to `in_transit`: Characters in staging map are warped to first en-route map; emits DEPARTED event on the observation map

### Instance Transports

Instances transition through states based on per-instance timers:

```
Boarding -> InTransit (when boarding window expires)
InTransit -> (released) (when arrival time is reached)
```

State transitions trigger character warping:
- Arrival: Characters in instance are warped to destination map with `uuid.Nil` instance
- Map exit during transport: Character is removed and transport is cancelled for that character
- Logout during transport: Character is removed and transport is cancelled for that character
- Login at transit map (crash recovery): Character is warped to route start map
- Stuck timeout (exceeds MaxLifetime): Characters are force-warped to route start map and instance is released
- Graceful shutdown: All characters are warped to route start map

## Processors

### transport.Processor (transport/processor.go)

| Method | Description |
|--------|-------------|
| AddTenant | Adds routes with computed schedules for a tenant |
| ClearTenant | Removes all routes for a tenant, returns count |
| ByIdProvider | Returns a provider for a route by ID |
| ByStartMapProvider | Returns a provider for a route by start map ID |
| GetByStartMap | Returns a route by its start map ID |
| AllRoutesProvider | Returns a provider for all routes |
| UpdateRoutes | Updates state for all routes |
| UpdateRouteAndEmit | Updates route state and emits Kafka events |
| WarpToRouteStartMapOnLogout | Warps character to route start map if in staging or en-route map |
| WarpToRouteStartMapOnLogoutAndEmit | Warps character and emits Kafka events |

### instance.Processor (instance/processor.go)

| Method | Description |
|--------|-------------|
| AddTenant | Adds instance routes for a tenant |
| ClearTenant | Removes all instance routes for a tenant, returns count |
| GetRoutes | Returns all instance routes for a tenant |
| GetRoute | Returns an instance route by ID |
| IsTransitMap | Checks if a map ID is an instance transit map |
| GetRouteByTransitMap | Returns an instance route by transit map ID |
| StartTransport | Starts an instance transport for a character |
| StartTransportAndEmit | Starts transport and emits Kafka events |
| HandleMapEnter | Handles character entering transit map (emits TRANSIT_ENTERED) |
| HandleMapEnterAndEmit | Handles map enter and emits Kafka events |
| HandleMapExit | Handles character exiting transit map (cancels transport) |
| HandleMapExitAndEmit | Handles map exit and emits Kafka events |
| HandleLogout | Handles character logout during transport |
| HandleLogoutAndEmit | Handles logout and emits Kafka events |
| HandleLogin | Handles character login at transit map (crash recovery) |
| HandleLoginAndEmit | Handles login and emits Kafka events |
| TickBoardingExpiration | Transitions expired boarding instances to InTransit |
| TickBoardingExpirationAndEmit | Ticks boarding expiration and emits Kafka events |
| TickArrival | Warps characters to destination on arrival |
| TickArrivalAndEmit | Ticks arrival and emits Kafka events |
| TickStuckTimeout | Force-cancels instances exceeding max lifetime |
| TickStuckTimeoutAndEmit | Ticks stuck timeout and emits Kafka events |
| GracefulShutdown | Warps all mid-transport characters to start maps |
| GracefulShutdownAndEmit | Graceful shutdown and emits Kafka events |

### config.Processor (transport/config/processor.go)

| Method | Description |
|--------|-------------|
| GetRoutes | Returns all routes for a tenant from configuration service |
| GetVessels | Returns all vessels for a tenant from configuration service |
| LoadConfigurationsForTenant | Loads routes and vessels for a tenant |

### instance config.Processor (instance/config/processor.go)

| Method | Description |
|--------|-------------|
| GetInstanceRoutes | Returns all instance routes for a tenant from configuration service |
| LoadConfigurationsForTenant | Loads instance routes for a tenant |

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
