# Instance-Based Transports Implementation Plan

**Last Updated: 2026-02-03**

---

## Executive Summary

This plan describes the implementation of instance-based transports in the Atlas MapleStory server, enabling NPCs to warp characters to isolated map instances for timed transport sequences (e.g., Kerning Square Train). The implementation requires:

1. **Broad adoption of `field.Model`** across all services to standardize location representation
2. **Instance pool management** in atlas-transports for capacity-controlled transport instances
3. **New Kafka message patterns** to support instance-aware character movement
4. **Enhanced atlas-maps service** to track characters by field (including instance UUID)

The design supports:
- Pooled instances with configurable capacity per transport route
- Pool segregation: Tenant → Route → Instance Pool (UUIDs)
- Fixed-duration transports with automatic destination warping
- Early cancellation via map exit with immediate instance release
- Logout handling that returns characters to the transport start point
- Graceful shutdown with character relocation to start maps
- Single-instance deployment (no distributed coordination needed)

---

## Current State Analysis

### Existing Architecture

**field.Model (atlas-constants)**
```go
type Model struct {
    worldId   world.Id      // byte
    channelId channel.Id    // byte
    mapId     _map.Id       // uint32
    instance  uuid.UUID     // UUID for instance isolation
}
```
- Already exists but is **not widely adopted**
- Format: `worldId:channelId:mapId:instance-uuid`
- Supports `uuid.Nil` for non-instanced maps

**atlas-transports Service**
- Handles **scheduled transports only** (boats, trains with timetables)
- Uses state machine: OutOfService → AwaitingReturn → OpenEntry → LockedEntry → InTransit
- Routes configured via JSON with boarding windows, travel durations
- Queries atlas-maps for characters using `world/channel/map` REST pattern
- No instance-based transport support currently

**atlas-maps Service**
- Tracks characters using `MapKey{Tenant, WorldId, ChannelId, MapId}`
- **Does not include instance UUID** in MapKey
- In-memory registry rebuilt from Kafka events
- REST endpoint: `GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/characters`

**Cross-Service Patterns**
- ~616 occurrences of world/channel/map parameters across 103 files
- ~20-25 services with significant integration points
- Inconsistent typing: some use `byte/uint32`, others use typed IDs
- Kafka messages include worldId, channelId, mapId as separate fields

### Key Gaps

1. **No instance UUID in location tracking** - atlas-maps MapKey lacks instance field
2. **No instance transport domain model** - atlas-transports only has scheduled routes
3. **No instance pool management** - no mechanism to allocate/release instances
4. **REST/Kafka patterns lack instance** - all APIs use world/channel/map only
5. **Character warp commands lack instance** - CHANGE_MAP command has no instance field

---

## Proposed Future State

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         NPC Conversation                             │
│  (Character initiates instance transport via NPC)                    │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       atlas-transports                               │
│  ┌─────────────────────┐  ┌──────────────────────────────────────┐  │
│  │ Instance Transport  │  │ Instance Pool Registry               │  │
│  │ Route Configuration │  │ (per route: available/in-use UUIDs) │  │
│  └─────────────────────┘  └──────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ Active Transport Registry                                       ││
│  │ (characterId → {field, routeId, startTime, destinationField})  ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
         │                              │                    ▲
         │ Warp to instance             │ Warp to dest       │ Map exit event
         ▼                              ▼                    │
┌─────────────────────────────────────────────────────────────────────┐
│                          atlas-maps                                  │
│  MapKey: {Tenant, WorldId, ChannelId, MapId, Instance}              │
│  REST: /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/characters   │
└─────────────────────────────────────────────────────────────────────┘
```

### New Domain Models

**Instance Transport Route** (new model in atlas-transports)
```go
type InstanceTransportRoute struct {
    id              uuid.UUID
    name            string
    startMapId      _map.Id       // Where NPC initiates transport
    transitMapId    _map.Id       // Map used during transport (instanced)
    destinationMapId _map.Id      // Final destination after transport
    capacity        uint32        // Max characters per instance
    travelDuration  time.Duration // Fixed duration before auto-warp
    cancelNpcId     uint32        // NPC that can cancel the transport (optional)
}
```

**Instance Pool Registry** (new registry in atlas-transports)
```go
// Pool key: (Tenant, RouteId)
// Each route has its own pool of instance UUIDs

type InstancePoolRegistry struct {
    pools map[PoolKey]*InstancePool  // tenant+route → pool
    mu    sync.RWMutex
}

type PoolKey struct {
    TenantId uuid.UUID
    RouteId  uuid.UUID
}

type InstancePool struct {
    available []uuid.UUID           // UUIDs ready to allocate
    inUse     map[uuid.UUID]uint32  // instance → character count (for future party support)
    capacity  uint32                // Max instances (from config)
    mu        sync.RWMutex
}
```

**Pool Initialization**: When a route is loaded, N instance UUIDs are pre-generated (N = capacity) and added to the `available` slice.

**Active Transport** (new model in atlas-transports)
```go
type ActiveTransport struct {
    characterId     uint32
    routeId         uuid.UUID
    instanceId      uuid.UUID
    originField     field.Model   // Where character came from
    transitField    field.Model   // Current instance field
    destinationField field.Model  // Where they'll arrive
    startedAt       time.Time
    expiresAt       time.Time     // startedAt + travelDuration
}
```

### Enhanced MapKey (atlas-maps)
```go
type MapKey struct {
    Tenant    tenant.Model
    WorldId   world.Id
    ChannelId channel.Id
    MapId     _map.Id
    Instance  uuid.UUID  // NEW: defaults to uuid.Nil for non-instanced
}
```

### New Kafka Message Types

**Instance Transport Commands** (new topic: COMMAND_TOPIC_INSTANCE_TRANSPORT)
```go
// Start instance transport (only command needed - cancellation handled via map exit detection)
type StartInstanceTransportCommand struct {
    CharacterId uint32
    RouteId     uuid.UUID
    WorldId     world.Id
    ChannelId   channel.Id
}
```

**Note**: No cancel command is needed. Cancellation is detected automatically via:
- MAP_STATUS CHARACTER_EXIT events (covers NPC-initiated exits, portal exits, GM warps)
- CHARACTER_STATUS LOGOUT events

**Enhanced Character Commands** (COMMAND_TOPIC_CHARACTER)
```go
type ChangeMapBody struct {
    ChannelId channel.Id
    MapId     _map.Id
    PortalId  uint32
    Instance  uuid.UUID  // NEW: optional, defaults to uuid.Nil
}
```

**Instance Transport Events** (new topic: EVENT_TOPIC_INSTANCE_TRANSPORT)
```go
type InstanceTransportEvent struct {
    Type        string    // "STARTED", "COMPLETED", "CANCELLED"
    CharacterId uint32
    RouteId     uuid.UUID
    InstanceId  uuid.UUID
    Body        interface{}
}
```

### Updated REST Endpoints

**atlas-maps** (new endpoint pattern)
```
GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/characters
```
- Returns characters in specific instance
- Existing endpoint returns only characters with `uuid.Nil` instance (non-instanced)

**atlas-transports** (new endpoints)
```
GET  /transports/instance-routes                    # List all instance routes
GET  /transports/instance-routes/{routeId}          # Get specific route
GET  /transports/instance-routes/{routeId}/pool     # Get pool status (capacity/available)
POST /transports/instance-routes/{routeId}/start    # Start transport for character
POST /transports/instance-routes/{routeId}/cancel   # Cancel transport for character
```

---

## Implementation Phases

### Phase 1: Foundation - field.Model Adoption (Broad Refactor)

**Objective**: Standardize location representation across all services using `field.Model`

This is the largest phase as it touches ~20-25 services with ~616 occurrences of world/channel/map patterns.

#### 1.1 Enhance atlas-constants field.Model

- Add helper methods for common operations
- Add JSON marshaling/unmarshaling
- Add Kafka message serialization helpers
- Ensure backward compatibility with `uuid.Nil` for non-instanced maps

#### 1.2 Update atlas-maps MapKey

- Add `Instance uuid.UUID` field to MapKey
- Update character registry to use instance-aware keys
- Add new REST endpoint for instance-specific queries
- Update spawn point registry for instance awareness
- Maintain backward compatibility: existing queries treat missing instance as `uuid.Nil`

#### 1.3 Update Kafka Message Definitions

Priority order (by dependency):
1. **atlas-constants**: Define shared Kafka message types using field.Model
2. **atlas-channel**: Update character status events with instance field
3. **atlas-maps**: Update map status events with instance field
4. **atlas-monsters**: Update monster events with instance field
5. **atlas-drops**: Update drop events with instance field
6. **atlas-reactors**: Update reactor events with instance field
7. **Remaining services**: Update in dependency order

#### 1.4 Update Service Processors

For each service, update:
- Processor function signatures to accept `field.Model` instead of separate params
- Provider functions to use field-based lookups
- REST request builders to include instance parameter
- Kafka consumers to parse instance field

### Phase 2: Instance Transport Core (atlas-transports)

**Objective**: Implement instance-based transport domain model and pool management

#### 2.1 Instance Transport Route Model

- Create new model for instance transport routes
- Add configuration schema and loader
- Implement route registry (similar to existing scheduled routes)

#### 2.2 Instance Pool Management

- Implement InstancePool with thread-safe allocation/release
- Pre-populate pools based on route capacity configuration
- Add pool status tracking and metrics

#### 2.3 Active Transport Registry

- Track active transports (character → transport mapping)
- Implement expiration tracking for auto-completion
- Support lookup by character, instance, or route

#### 2.4 Transport Lifecycle

- **Start**: Allocate instance, warp character, register active transport
- **Complete**: Warp to destination, release instance, cleanup
- **Cancel**: Return to start map, release instance, cleanup
- **Logout**: Same as cancel, triggered by character status event

### Phase 3: Transport Execution

**Objective**: Implement the transport flow with proper timing and cancellation

#### 3.1 Transport Start Flow

1. Receive start command (from NPC conversation via Kafka)
2. Check pool availability (return error if full → "wagon is full" NPC dialog)
3. Allocate instance from pool
4. Build transit field (world, channel, transitMapId, instanceId)
5. Emit CHANGE_MAP command with instance
6. Register active transport with expiration time
7. Emit STARTED event

#### 3.2 Transport Completion Flow

1. Background ticker checks for expired transports
2. For each expired transport:
   - Build destination field
   - Emit CHANGE_MAP command to destination
   - Release instance to pool
   - Remove from active registry
   - Emit COMPLETED event

#### 3.3 Transport Cancellation Flow

1. Triggered by:
   - Map exit event (character left transit map via NPC, portal, GM warp, etc.)
   - Character logout/disconnect
2. Warp character to start map (transport start point) - only needed for logout, map exit already moved them
3. Release instance immediately to pool
4. Remove from active registry
5. Emit CANCELLED event with reason ("MAP_EXIT" or "LOGOUT")

#### 3.4 Map Exit Detection

- Subscribe to MAP_STATUS events (CHARACTER_EXIT)
- Check if exiting character is in active transport for that instance
- If character exits transit map (not via transport completion), trigger cancel
- No warp needed - they already left; just cleanup registry and release instance

#### 3.5 Double-Transport Prevention

- Before starting a transport, check if character already has an active transport
- Reject with error if already in transport (NPC can handle this gracefully)

#### 3.6 Stuck Transport Timeout

- Add maximum lifetime safety net (e.g., 2x configured duration)
- If transport exceeds max lifetime, force-cancel and warp to start map
- Log warning for investigation

#### 3.7 Graceful Shutdown

On SIGTERM/SIGINT:
1. Stop accepting new transport start commands
2. Iterate through all active transports
3. Warp each character to their route's start map
4. Release all instances
5. Exit cleanly

#### 3.8 Crash Recovery

On service startup:
1. Characters may be stranded in transit maps with invalid instances
2. Subscribe to CHARACTER_STATUS LOGIN events
3. Check if logging-in character's mapId matches any transit map
4. If so, warp them to the corresponding route's start map

### Phase 4: Integration & Configuration

**Objective**: Connect instance transports to NPC conversations and create configuration

#### 4.1 NPC Conversation Integration

- Define Kafka command for starting instance transport
- NPC conversation scripts can check capacity before offering transport
- Handle "full" response gracefully in NPC dialog

#### 4.2 Configuration Files

Create instance transport route configurations:
```json
{
  "name": "kerning-square-train",
  "startMapId": 103000000,
  "transitMapId": 103000100,
  "destinationMapId": 103000200,
  "capacity": 6,
  "travelDurationSeconds": 30,
  "cancelNpcId": 1052007
}
```

#### 4.3 REST API Implementation

- Implement instance route endpoints
- Add pool status endpoint for monitoring
- Document API in service docs

### Phase 5: Testing & Migration

**Objective**: Ensure correctness and provide migration path

#### 5.1 Unit Tests

- Instance pool allocation/release
- Active transport lifecycle
- Cancellation scenarios
- Expiration handling

#### 5.2 Integration Tests

- Full transport flow (start → complete)
- Cancellation via NPC
- Cancellation via map exit
- Logout handling
- Pool exhaustion handling

#### 5.3 Migration Strategy

- Deploy field.Model changes to atlas-constants first
- Update services in dependency order
- Use feature flag for instance transport activation
- Gradual rollout per transport route

---

## Detailed Tasks

### Phase 1 Tasks

| ID | Task | Effort | Dependencies |
|----|------|--------|--------------|
| 1.1.1 | Add JSON marshaling to field.Model | S | - |
| 1.1.2 | Add Kafka serialization helpers to field.Model | S | 1.1.1 |
| 1.1.3 | Add field.Model equality and comparison methods | S | - |
| 1.2.1 | Add Instance field to atlas-maps MapKey | M | 1.1.x |
| 1.2.2 | Update character registry for instance-aware keys | M | 1.2.1 |
| 1.2.3 | Add instance-specific REST endpoint | M | 1.2.2 |
| 1.2.4 | Update spawn point registry for instances | M | 1.2.1 |
| 1.3.1 | Update atlas-channel character status events | L | 1.1.x |
| 1.3.2 | Update atlas-maps map status events | M | 1.2.x |
| 1.3.3 | Update atlas-monsters events | L | 1.1.x |
| 1.3.4 | Update atlas-drops events | L | 1.1.x |
| 1.3.5 | Update atlas-reactors events | L | 1.1.x |
| 1.3.6 | Update remaining services (batch) | XL | 1.3.1-1.3.5 |
| 1.4.1 | Update processor signatures across services | XL | 1.3.x |

### Phase 2 Tasks

| ID | Task | Effort | Dependencies |
|----|------|--------|--------------|
| 2.1.1 | Create InstanceTransportRoute model | S | 1.2.x |
| 2.1.2 | Create instance route configuration schema | S | 2.1.1 |
| 2.1.3 | Implement instance route registry | M | 2.1.2 |
| 2.1.4 | Implement configuration loader | M | 2.1.3 |
| 2.2.1 | Implement InstancePool struct | M | 2.1.x |
| 2.2.2 | Implement allocation/release methods | M | 2.2.1 |
| 2.2.3 | Add pool initialization from config | S | 2.2.2 |
| 2.3.1 | Create ActiveTransport model | S | 1.1.x |
| 2.3.2 | Implement active transport registry | M | 2.3.1 |
| 2.3.3 | Add expiration tracking | M | 2.3.2 |

### Phase 3 Tasks

| ID | Task | Effort | Dependencies |
|----|------|--------|--------------|
| 3.1.1 | Implement transport start command handler | M | 2.x |
| 3.1.2 | Implement pool availability check | S | 2.2.x |
| 3.1.3 | Implement warp with instance | M | 1.3.x |
| 3.2.1 | Implement expiration ticker | M | 2.3.3 |
| 3.2.2 | Implement completion warp | M | 3.1.3 |
| 3.3.1 | Subscribe to MAP_STATUS events | S | 1.3.2 |
| 3.3.2 | Filter for CHARACTER_EXIT in transit maps | S | 3.3.1 |
| 3.3.3 | Implement map exit cancellation | M | 3.3.2 |
| 3.4.1 | Enhance logout handler for instance transports | M | 3.1.x |
| 3.4.2 | Warp to start map on logout | M | 3.4.1 |
| 3.5.1 | Check active transport before starting new | S | 2.3.x |
| 3.6.1 | Add max lifetime config | S | 2.1.x |
| 3.6.2 | Implement stuck transport detection | M | 3.6.1, 3.2.1 |
| 3.7.1 | Register SIGTERM/SIGINT handler | S | - |
| 3.7.2 | Implement graceful shutdown warp | M | 3.7.1, 2.3.x |
| 3.8.1 | Subscribe to CHARACTER_STATUS LOGIN events | S | - |
| 3.8.2 | Detect stranded characters on login | M | 3.8.1, 2.1.x |

### Phase 4 Tasks

| ID | Task | Effort | Dependencies |
|----|------|--------|--------------|
| 4.1.1 | Define start transport Kafka command | S | 3.1.x |
| 4.1.2 | Add capacity check endpoint for NPCs | M | 2.2.x |
| 4.2.1 | Create Kerning Square Train config | S | 2.1.2 |
| 4.2.2 | Document configuration format | S | 4.2.1 |
| 4.3.1 | Implement instance route REST endpoints | M | 2.1.x |
| 4.3.2 | Implement pool status endpoint | S | 2.2.x |
| 4.3.3 | Update service documentation | M | 4.3.x |

### Phase 5 Tasks

| ID | Task | Effort | Dependencies |
|----|------|--------|--------------|
| 5.1.1 | Unit tests for instance pool | M | 2.2.x |
| 5.1.2 | Unit tests for active transport registry | M | 2.3.x |
| 5.1.3 | Unit tests for cancellation logic | M | 3.3.x |
| 5.1.4 | Unit tests for MapKey with instance | S | 1.2.x |
| 5.2.1 | Integration test: full transport flow | L | 3.x, 4.x |
| 5.2.2 | Integration test: map exit cancellation | L | 3.3.x |
| 5.2.3 | Integration test: logout handling | M | 3.4.x |
| 5.2.4 | Integration test: pool exhaustion | M | 2.2.x |
| 5.2.5 | Integration test: graceful shutdown | L | 3.7.x |
| 5.2.6 | Integration test: crash recovery | L | 3.8.x |
| 5.2.7 | Integration test: double-transport prevention | M | 3.5.x |
| 5.2.8 | Integration test: stuck transport timeout | M | 3.6.x |
| 5.3.1 | Create migration documentation | M | All |
| 5.3.2 | Add feature flag support | S | 3.x |

---

## Risk Assessment

### High Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Broad field.Model adoption breaks existing services | Service outages | Phased rollout with feature flags; thorough testing per service |
| Instance leak (instances not released) | Pool exhaustion | Heartbeat/timeout mechanism; monitoring alerts; manual cleanup endpoint |
| Race conditions in pool allocation | Duplicate allocations | RWMutex with proper locking; atomic operations where possible |

### Medium Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Kafka message version mismatch | Message parsing failures | Version headers; backward-compatible message evolution |
| Performance impact of instance tracking | Increased latency | Benchmark before/after; optimize hot paths |
| Character stuck in instance on service restart | Poor UX | Persist active transports; recovery on startup |

### Low Risk

| Risk | Impact | Mitigation |
|------|--------|------------|
| Configuration errors | Transport route misconfiguration | Schema validation; startup health checks |
| NPC dialog inconsistency | Confusing UX | Clear capacity feedback; graceful error handling |

---

## Success Metrics

1. **Functional**: Kerning Square Train transport works end-to-end
2. **Performance**: Transport start latency < 100ms
3. **Reliability**: No instance leaks over 24-hour test period
4. **Correctness**: All cancellation scenarios properly handled
5. **Adoption**: field.Model used consistently across all services

---

## Required Resources and Dependencies

### Services Impacted
- atlas-constants (library)
- atlas-transports (primary)
- atlas-maps (major changes)
- atlas-channel (Kafka messages)
- atlas-monsters, atlas-drops, atlas-reactors (Kafka messages)
- atlas-npc-conversations (integration)
- ~15 additional services (minor Kafka/processor updates)

### External Dependencies
- Kafka topic creation (COMMAND_TOPIC_INSTANCE_TRANSPORT, EVENT_TOPIC_INSTANCE_TRANSPORT)
- Configuration service updates for instance route configs

### Tooling
- Existing Go testing framework
- Kafka testing utilities
- Service documentation templates

---

## Deferred Items

| Item | Notes |
|------|-------|
| Death in transit map | Respawn uses map's `ReturnMapId`. May work correctly if transit map data is configured properly. Test after implementation and revisit if needed. |

---

## Open Questions (Resolved)

| Question | Resolution |
|----------|------------|
| Fresh instance per use vs pooled? | **Pooled instances** - supports capacity-based NPC dialogs |
| Logout behavior - original map or start? | **Transport start point** - consistent behavior |
| Group/party travel support? | **Future consideration** - design for solo, structure for party later |
| Duration model? | **Fixed duration** - configured per route |
| Instance ID type? | **UUID** - consistent with existing field.Model |
| field.Model adoption scope? | **Broad adoption** - update all services |
| Capacity configuration location? | **Transport route config** - per-route capacity |
| Cancel instance release timing? | **Immediate** - no cooldown period |
| Tenant in field.Model? | **Separate** - tenant handled at context/header level |
| Pool coordination for replicas? | **Single instance only** - no distributed coordination needed |
| Graceful shutdown behavior? | **Start map** - warp all mid-transport characters to start |
| NPC cancel command needed? | **No** - map exit detection covers all cancel scenarios |
| Pool segregation? | **Tenant → Route → Pool** - each route has its own pool per tenant |

---

## Appendix: Key File Locations

### atlas-constants
- `libs/atlas-constants/field/model.go` - field.Model definition

### atlas-transports
- `services/atlas-transports/atlas.com/transports/transport/model.go` - Route models
- `services/atlas-transports/atlas.com/transports/transport/route_registry.go` - Route registry
- `services/atlas-transports/atlas.com/transports/transport/processor.go` - Transport processing
- `services/atlas-transports/atlas.com/transports/character/processor.go` - Character warping
- `services/atlas-transports/routes/` - Route configuration files

### atlas-maps
- `services/atlas-maps/atlas.com/maps/map/character/registry.go` - Character registry
- `services/atlas-maps/atlas.com/maps/map/character/model.go` - MapKey definition
- `services/atlas-maps/atlas.com/maps/map/resource.go` - REST endpoints

### Documentation
- `docs/TODO.md` - Instance transport TODOs (lines 197-226)
