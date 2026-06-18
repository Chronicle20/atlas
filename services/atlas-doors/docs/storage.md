# Door Storage

This service uses Redis for all state storage. There is no SQL or relational database.

## Keys

### Door Instances

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:door:{tenantId}:{areaDoorId}` | String (JSON) | Door instance data, keyed by area door id |

The door JSON (`storedDoor`) contains the tenant fields (tenantId, region, major, minor), the area field (worldId, channelId, mapId, instance), and the door fields (areaDoorId, townDoorId, ownerCharacterId, partyId, skillId, skillLevel, townMapId, slot, townPortalId, areaX, areaY, townX, townY). `deployTime` and `expiresAt` are serialized as Unix milliseconds (`deployMs`, `expiresMs`), with 0 representing the zero time.

### Index Sets

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:door-field:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Area door ids of doors in a field |
| `atlas:door-owner:{tenantId}:{characterId}` | Set | Area door ids of doors owned by a character |
| `atlas:door-town:{tenantId}:{worldId}:{channelId}:{townMapId}:{partyScope}` | Set | Area door ids in a town-party bucket |

Each Set stores area door ids as decimal strings that correspond to `atlas:door` keys. `partyScope` is the party id when the owner is in a party, or `solo-{characterId}` for solo casters (preventing two solo casters at the same town from sharing a town-party bucket).

### ID Allocation

Door object ids are NOT minted by this service. They come from the shared `atlas-object-id` allocator (`libs/atlas-object-id`). Each door allocates two object ids (area and town). On a town-allocation or persistence failure, already-allocated ids are released back to the tenant's free pool. There is intentionally no MinId fallback, which would cause silent object-id collisions.

### Leader-Election Lock

| Key | Description |
|-----|-------------|
| `doors-sweep` | Distributed lock (`atlas-lock`) gating the expiry sweep so only the leader pod runs it; TTL, refresh, and backoff are configurable. |

## Relationships

Door instances are indexed three ways: by area field (`atlas:door-field`), by owner character (`atlas:door-owner`), and by town-party bucket (`atlas:door-town`). Each index Set contains the area door ids that correspond to `atlas:door` instance keys. On `Put`, the instance is written and added to all three indices; on `Remove`, the stored door is read to reconstruct and clear all three index entries before deleting the instance.

## Indexes

| Index Key Pattern | Points To |
|-------------------|-----------|
| `atlas:door-field:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | `atlas:door:{tenantId}:{areaDoorId}` |
| `atlas:door-owner:{tenantId}:{characterId}` | `atlas:door:{tenantId}:{areaDoorId}` |
| `atlas:door-town:{tenantId}:{worldId}:{channelId}:{townMapId}:{partyScope}` | `atlas:door:{tenantId}:{areaDoorId}` |

## Migration Rules

All state is ephemeral. State is not preserved across restarts. There are no schema migrations.
