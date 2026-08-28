# Dragon Storage

This service uses Redis for all state storage. There is no SQL or relational database.

## Keys

### Dragon Instances

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:dragon:{tenantId}:{ownerCharacterId}` | String (JSON) | Dragon instance data, keyed by owner character id |

The dragon JSON (`storedDragon`) contains the tenant fields (tenantId, tenantRegion, tenantMajorVersion, tenantMinorVersion), the field (worldId, channelId, mapId, instance), and the dragon fields (ownerCharacterId, x, y, stance, jobId).

### Field Index

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:dragon-map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Owner character ids of dragons in a field |

The Set stores owner character ids as decimal strings that correspond to `atlas:dragon` keys.

## Relationships

Each dragon instance is indexed by its field in the field index. On `Put`, the instance is written and added to the field index; if the character already had a dragon on a different field, the stale field-index membership is removed first. On `Remove`, the stored dragon is read to reconstruct and remove the field-index entry before deleting the instance.

## Indexes

| Index Key Pattern | Points To |
|--------------------|-----------|
| `atlas:dragon-map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | `atlas:dragon:{tenantId}:{ownerCharacterId}` |

`GetInField` skips stale index entries (a dragon removed without going through `Remove`) rather than surfacing an error.

## Migration Rules

All state is ephemeral. State is not preserved across restarts. There are no schema migrations.
