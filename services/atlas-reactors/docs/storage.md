# Reactor Storage

## Tables

This service uses Redis for volatile storage. No relational database is used.

### Key Patterns

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `reactor:{tenantId}:{id}` | String (JSON) | JSON-serialized reactor `Model` for one reactor |
| `reactors:all` | Set | Cross-tenant index of `{tenantId}:{id}` member strings — every live reactor in every tenant appears here exactly once |
| `reactors:map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Per-field index; members are reactor `id` values |
| `reactor:cd:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}:{classification}:{x}:{y}` | String (TTL) | Cooldown marker; TTL is the reactor's `delay` in milliseconds |
| `reactor:spot:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}:{classification}:{x}:{y}` | String | Spatial-slot guard; reserved by `TryClaimSpot` to prevent two concurrent CREATE commands from producing duplicate reactors at the same position |

ID allocation does NOT live in this service — see [ID Generation](#id-generation) below.

### `reactor:{tenantId}:{id}` Value Structure

JSON-serialized `Model` containing:

| Field          | JSON Type | Description                    |
|----------------|-----------|--------------------------------|
| tenant         | object    | Tenant model                   |
| id             | number    | Reactor instance ID            |
| worldId        | number    | World identifier               |
| channelId      | number    | Channel identifier             |
| mapId          | number    | Map identifier                 |
| instance       | string    | UUID instance identifier       |
| classification | number    | Reactor classification ID      |
| name           | string    | Reactor name                   |
| data           | object    | Reactor game data              |
| state          | number    | Current reactor state          |
| eventState     | number    | Event state                    |
| delay          | number    | Respawn delay in milliseconds  |
| direction      | number    | Facing direction               |
| x              | number    | X coordinate                   |
| y              | number    | Y coordinate                   |
| updateTime     | string    | Last update timestamp          |

## Relationships

- Each `reactor:{tenantId}:{id}` is referenced from exactly one `reactors:map:...` set (the field it lives in) AND from `reactors:all` (as a `{tenantId}:{id}` tuple).
- `reactor:cd:...` and `reactor:spot:...` are coordinate-keyed and live independently of any specific reactor instance — a destroyed reactor's slot/cooldown persists past the reactor's removal so that respawn timing and dedup work correctly across the next CREATE.

## Indexes

- `reactors:all` is the cross-tenant teardown sweep index.
- `reactors:map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` is the field-scoped query index used by hit/destroy/list paths.
- Spot and cooldown keys are content-addressed by coordinates and need no separate index.

## Migration Rules

N/A. Redis state is volatile and not preserved across restarts.

### ID Generation

Reactor IDs are NOT minted by this service. They come from the shared `atlas-object-id` allocator (`libs/atlas-object-id/allocator.go`), which is also used by atlas-monsters and atlas-drops.

Allocator-managed keys (per tenant, NOT per service):

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:oid:{tenantId}:next` | String (counter) | Sequential ID counter; range `1000000` (`MinId`) to `2147483647` (`MaxId`, the v83 wire-format positive int32 ceiling) |
| `atlas:oid:{tenantId}:free` | List | LIFO recycle pool; only consulted once the counter passes `RecycleThreshold = MaxId - 100M` |

A single tenant-scoped namespace is shared across reactors, monsters, and drops because the v83 client keys map objects by oid alone — colliding IDs across entity types crash the client. Per-tenant rather than per-field: each service stores its entities under `<entity>:{tenantId}:{id}` with no field component in the key, so per-field allocation would collide in storage when the same id was minted in two different fields. See the package-level comment in `libs/atlas-object-id/allocator.go` for the full rationale.

### Cooldown Expiration

Cooldown keys (`reactor:cd:...`) carry a Redis TTL equal to the reactor's `delay` in milliseconds. Expiration is automatic; this service never explicitly deletes a cooldown key on the timer's account.

### Spatial Slot Guard

`TryClaimSpot` writes a no-TTL value at `reactor:spot:...`. `ReleaseSpot` deletes it (called from `Destroy`, `DestroyInField`, and on Create-failure rollback). `ClearAllSpotsForMap` SCANs and DELs the per-map prefix during teardown.

The guard exists to dedupe concurrent CREATE commands — two racing map-Enter spawns can both observe the same "missing reactor" and both issue CREATE; the first `TryClaimSpot` wins via Redis `SETNX`, the loser logs and returns without creating a duplicate.
