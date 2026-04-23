# Storage

## Tables

This service uses Redis for storage. No relational database tables exist.

## Redis Key Structures

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `drop:{tenantId}:{id}` | String (JSON) | Serialized `dropEntry` for a single drop |
| `drops:all` | Set | Cross-tenant index of `{tenantId}:{id}` member strings — every live drop in every tenant appears here exactly once |
| `drops:map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}` | Set | Per-field index; members are drop `id` values |

ID allocation does NOT live in this service — see [ID Generation](#id-generation) below.

### Drop Entry Format

Each `drop:{tenantId}:{id}` key stores a JSON-serialized `dropEntry`:

| Field | Type | Description |
|-------|------|-------------|
| drop | Model (JSON) | Full drop model including tenant, field, item/meso data, equipment stats |
| reservedBy | uint32 | Character ID that reserved the drop (0 if not reserved) |

## Relationships

- Each drop belongs to exactly one field (identified by tenant + world + channel + map + instance) and is referenced from one `drops:map:...` set AND from `drops:all` (as a `{tenantId}:{id}` tuple).
- Equipment drops carry all stats inline on the drop model; there is no separate equipment storage.
- Drop reservations are tracked within the drop entry itself, pairing the drop model with the reserving character ID.

## Indexes

- Drop by ID: direct key lookup via `drop:{tenantId}:{id}`.
- Drops by map: set membership via `drops:map:{tenantId}:{worldId}:{channelId}:{mapId}:{instance}`.
- Cross-tenant teardown sweep: `drops:all`.

## ID Generation

Drop IDs are NOT minted by this service. They come from the shared `atlas-object-id` allocator (`libs/atlas-object-id/allocator.go`), which is also used by atlas-reactors and atlas-monsters.

Allocator-managed keys (per tenant, NOT per service):

| Key Pattern | Redis Type | Description |
|-------------|------------|-------------|
| `atlas:oid:{tenantId}:next` | String (counter) | Sequential ID counter; range `1000000` (`MinId`) to `2147483647` (`MaxId`, the v83 wire-format positive int32 ceiling) |
| `atlas:oid:{tenantId}:free` | List | LIFO recycle pool; only consulted once the counter passes `RecycleThreshold = MaxId - 100M` |

A single tenant-scoped namespace is shared across reactors, monsters, and drops because the v83 client keys map objects by oid alone — colliding IDs across entity types crash the client. Per-tenant rather than per-field: each service stores its entities under `<entity>:{tenantId}:{id}` with no field component in the key, so per-field allocation would collide in storage when the same id was minted in two different fields. See the package-level comment in `libs/atlas-object-id/allocator.go` for the full rationale.

## Migration Rules

Not applicable. Drops are ephemeral and do not require schema migrations. On service shutdown, all drops are expired and corresponding EXPIRED events are emitted before the process exits.
