# Kite Storage

This service uses Redis for all state storage. There is no SQL or relational database.

## Keys

### Kite Instances

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:kite:{tenantId}:{characterId}` | String (JSON) | Kite instance data, keyed by owning character id |

The kite JSON (`entry`) contains the wire id, field (worldId, channelId, mapId, instance), characterId, name, templateId, message, x, y, and createdAt.

### Wire ID Counter

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:kite:{tenantId}:_id` | String (integer, INCR) | Per-tenant counter allocating each kite's wire id (`Registry.NextId`), independent of characterId |

### Per-Field Placement Lock

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:kite-cap:_lock:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | String (SETNX, TTL) | Distributed lock serialising the per-map cap check (count -> validate -> allocate -> insert) for one field |

### Character-Presence Index

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:kite-char:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Character ids currently present in the field, as decimal strings |

## Relationships

A kite instance (`atlas:kite`) is looked up directly by owning character id; there is no secondary index on it. The character-presence index (`atlas:kite-char`) is independent of the kite registry: it tracks all characters in a field (from consumed character status events), and the kite domain's `InMapModelProvider` intersects that index with kite ownership (`atlas:kite` existence checks) to answer "kites in this field."

## Indexes

| Index Key Pattern | Points To |
|--------------------|-----------|
| `atlas:kite-char:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Character ids, cross-referenced against `atlas:kite:{tenantId}:{characterId}` |

## Migration Rules

All state is ephemeral. State is not preserved across restarts. There are no schema migrations.
