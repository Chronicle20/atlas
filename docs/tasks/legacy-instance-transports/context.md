# Instance-Based Transports - Context Document

**Last Updated: 2026-02-03**

---

## Key Design Decisions

### 1. Instance Model
- **Decision**: Use UUID-based instances consistent with existing `field.Model`
- **Rationale**: Maintains consistency with atlas-constants library; UUIDs provide guaranteed uniqueness

### 2. Pool Management
- **Decision**: Pooled instances with configurable capacity per route
- **Rationale**: Supports "wagon is full" NPC dialog pattern; capacity-based gameplay mechanics

### 2a. Pool Segregation
- **Decision**: Pools are keyed by (Tenant, RouteId)
- **Structure**: `Tenant → Route → Instance Pool (list of UUIDs)`
- **Rationale**: Each route needs its own pool; multi-tenancy requires tenant isolation
- **Initialization**: N UUIDs pre-generated when route loads (N = capacity config)

### 2b. Pool Coordination
- **Decision**: Single service instance only (no distributed coordination)
- **Rationale**: Simplifies implementation; in-memory pool management is sufficient

### 3. Location Representation
- **Decision**: Broad adoption of `field.Model` across all services
- **Rationale**: Standardized location representation; enables instance-aware operations everywhere

### 4. Tenant Handling
- **Decision**: Keep tenant separate from field.Model
- **Rationale**: Tenant is a cross-cutting concern handled via context/headers; field.Model focuses on in-world location

### 5. Logout Behavior
- **Decision**: Return to transport start point (not original departure location)
- **Rationale**: Consistent behavior; simpler implementation; no need to track origin

### 6. Cancel Instance Release
- **Decision**: Immediate release back to pool
- **Rationale**: Maximizes pool availability; no need for cooldown in this use case

### 7. Duration Model
- **Decision**: Fixed duration configured per route
- **Rationale**: Simpler than event-triggered; matches original game behavior for Kerning Square Train

### 8. Group Travel
- **Decision**: Solo only for now, but structure for future party support
- **Rationale**: Reduces initial complexity; model supports extension

### 9. Cancellation Detection
- **Decision**: No explicit cancel command; use map exit and logout detection
- **Rationale**: NPC cancel actions result in map exit anyway; simpler design with fewer commands

### 10. Graceful Shutdown
- **Decision**: Warp all mid-transport characters to start map on shutdown
- **Rationale**: Consistent with logout behavior; safe fallback location

### 11. Crash Recovery
- **Decision**: Detect stranded characters on login by checking if their mapId is a transit map
- **Rationale**: Handles ungraceful shutdown; ensures characters aren't stuck in invalid instances

### 12. Death in Transit (Deferred)
- **Decision**: Defer - test after implementation to see if transit map's `ReturnMapId` handles respawn correctly
- **Rationale**: May not need code changes if map data is configured properly
- **Fallback options if needed**: Query transports from respawn processor, or add respawn override mechanism

---

## Key Files Reference

### Core Libraries

| File | Purpose |
|------|---------|
| `libs/atlas-constants/field/model.go` | Location composite (world, channel, map, instance) |
| `libs/atlas-constants/field/constants.go` | Field ID format string |
| `libs/atlas-constants/world/constants.go` | World ID type |
| `libs/atlas-constants/channel/model.go` | Channel model |
| `libs/atlas-constants/map/model.go` | Map ID type |

### atlas-transports (Primary Service)

| File | Purpose |
|------|---------|
| `services/atlas-transports/atlas.com/transports/transport/model.go` | Scheduled route model |
| `services/atlas-transports/atlas.com/transports/transport/state.go` | Route state machine |
| `services/atlas-transports/atlas.com/transports/transport/route_registry.go` | Route storage |
| `services/atlas-transports/atlas.com/transports/transport/processor.go` | Route processing logic |
| `services/atlas-transports/atlas.com/transports/transport/scheduler.go` | Trip scheduling |
| `services/atlas-transports/atlas.com/transports/character/processor.go` | Character warping |
| `services/atlas-transports/atlas.com/transports/character/producer.go` | Warp command Kafka producer |
| `services/atlas-transports/atlas.com/transports/channel/registry.go` | Active channel tracking |
| `services/atlas-transports/atlas.com/transports/map/processor.go` | Map character queries |
| `services/atlas-transports/routes/schema.json` | Route configuration schema |
| `services/atlas-transports/routes/*.json` | Route configuration files |

### atlas-maps (Major Changes)

| File | Purpose |
|------|---------|
| `services/atlas-maps/atlas.com/maps/map/character/registry.go` | In-memory character tracking |
| `services/atlas-maps/atlas.com/maps/map/character/model.go` | MapKey definition (needs instance) |
| `services/atlas-maps/atlas.com/maps/map/processor.go` | Character enter/exit logic |
| `services/atlas-maps/atlas.com/maps/map/resource.go` | REST endpoint handler |
| `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go` | Character event consumer |
| `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go` | Map status event producer |

### atlas-channel (Kafka Messages)

| File | Purpose |
|------|---------|
| `services/atlas-channel/socket/handler/map_change.go` | Map change handling |
| `services/atlas-channel/kafka/message/character/kafka.go` | Character status events |

### Related Services (Kafka Updates)

| Service | Key Files |
|---------|-----------|
| atlas-monsters | `kafka/message/monster/kafka.go`, `monster/processor.go` |
| atlas-drops | `kafka/message/drop/kafka.go`, `drop/processor.go` |
| atlas-reactors | `kafka/message/reactor/kafka.go`, `reactor/processor.go` |
| atlas-expressions | `kafka/message/expression/kafka.go` |
| atlas-messages | `kafka/message/chat/kafka.go` |
| atlas-consumables | `map/character/processor.go` |
| atlas-npc-conversations | `map/processor.go` |

---

## Dependencies Graph

```
field.Model (atlas-constants)
    │
    ├─► atlas-maps (MapKey update)
    │       │
    │       └─► REST endpoint update
    │       └─► Character registry update
    │       └─► Map status events update
    │
    ├─► atlas-transports (instance transport core)
    │       │
    │       ├─► InstanceTransportRoute model
    │       ├─► InstancePool registry
    │       ├─► ActiveTransport registry
    │       └─► Transport processors
    │
    ├─► atlas-channel (character status events)
    │       │
    │       └─► CHANGE_MAP command with instance
    │
    └─► Other services (Kafka message updates)
            │
            ├─► atlas-monsters
            ├─► atlas-drops
            ├─► atlas-reactors
            └─► (15+ other services)
```

---

## Kafka Topics

### Existing Topics (to be updated)

| Topic | Changes |
|-------|---------|
| COMMAND_TOPIC_CHARACTER | Add `Instance` field to CHANGE_MAP body |
| EVENT_TOPIC_CHARACTER_STATUS | Add `Instance` field to location-related events |
| EVENT_TOPIC_MAP_STATUS | Add `Instance` field to CHARACTER_ENTER/EXIT events |
| EVENT_TOPIC_MONSTER_STATUS | Add `Instance` field |
| EVENT_TOPIC_DROP_STATUS | Add `Instance` field |
| EVENT_TOPIC_REACTOR_STATUS | Add `Instance` field |

### New Topics (to be created)

| Topic | Purpose |
|-------|---------|
| COMMAND_TOPIC_INSTANCE_TRANSPORT | Command to start instance transport (cancel handled via map exit detection) |
| EVENT_TOPIC_INSTANCE_TRANSPORT | Events for transport lifecycle (STARTED, COMPLETED, CANCELLED) |

---

## Configuration Schema

### Instance Transport Route Config (new)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["name", "startMapId", "transitMapId", "destinationMapId", "capacity", "travelDurationSeconds"],
  "properties": {
    "name": {
      "type": "string",
      "description": "Unique route name"
    },
    "startMapId": {
      "type": "integer",
      "description": "Map where NPC initiates transport"
    },
    "transitMapId": {
      "type": "integer",
      "description": "Map used during transport (will be instanced)"
    },
    "destinationMapId": {
      "type": "integer",
      "description": "Final destination after transport"
    },
    "capacity": {
      "type": "integer",
      "minimum": 1,
      "description": "Maximum characters per instance"
    },
    "travelDurationSeconds": {
      "type": "integer",
      "minimum": 1,
      "description": "Fixed duration before auto-warp to destination"
    },
    "cancelNpcId": {
      "type": "integer",
      "description": "NPC ID that can cancel the transport (optional)"
    }
  }
}
```

### Example: Kerning Square Train

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

---

## REST API Changes

### atlas-maps

**New Endpoint**
```
GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/characters
```

**Existing Endpoint Behavior**
```
GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/characters
```
- Returns characters with `instance = uuid.Nil` (non-instanced)
- Backward compatible

### atlas-transports

**New Endpoints**
```
GET  /transports/instance-routes
GET  /transports/instance-routes/{routeId}
GET  /transports/instance-routes/{routeId}/pool
POST /transports/instance-routes/{routeId}/start
POST /transports/instance-routes/{routeId}/cancel
```

---

## Testing Strategy

### Unit Test Focus Areas
1. Instance pool allocation/release thread safety
2. Active transport expiration logic
3. Cancellation scenarios (NPC, map exit, logout)
4. MapKey with instance equality

### Integration Test Scenarios
1. **Happy path**: Start → Wait → Auto-complete → Arrive at destination
2. **Map exit (NPC)**: Start → Use cancel NPC → Map exit detected → Instance released
3. **Map exit (other)**: Start → Exit via portal/GM → Map exit detected → Instance released
4. **Logout**: Start → Logout → Instance released → Re-login → Warp to start
5. **Pool full**: Request when capacity reached → Rejection
6. **Graceful shutdown**: Characters mid-transport → SIGTERM → All warped to start → Clean exit
7. **Crash recovery**: Service crashes → Characters stranded → Service restarts → Login detects → Warp to start
8. **Double-transport**: Character in transport → Try to start another → Rejection
9. **Stuck transport**: Transport exceeds max lifetime → Force-cancel → Warp to start
10. **Death in transit** (deferred): Test if map's ReturnMapId handles respawn correctly

---

## Migration Notes

### Phased Rollout Order
1. Deploy atlas-constants with field.Model enhancements
2. Deploy atlas-maps with instance-aware MapKey
3. Deploy atlas-channel with instance in Kafka messages
4. Deploy other services with Kafka message updates
5. Deploy atlas-transports with instance transport feature
6. Enable instance transport routes via configuration

### Backward Compatibility
- `uuid.Nil` used for all non-instanced maps
- Existing REST endpoints continue to work
- Kafka messages with missing instance field default to `uuid.Nil`
- Feature flag controls instance transport activation

---

## Related Documentation

- `docs/TODO.md` - Instance transport TODOs (lines 197-226)
- `services/atlas-transports/docs/` - Transport service documentation
- `services/atlas-maps/docs/` - Maps service documentation
