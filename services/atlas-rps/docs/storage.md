# RPS Storage

This service uses Redis for all state storage. There is no SQL or relational database.

## Tables

Not applicable; there is no relational schema. State is stored as Redis keys via the shared `atlas-redis` TTL registry (`libs/atlas-redis`).

### Session Registry

| Key Pattern | Redis Type | Description |
|-------------|------------|--------------|
| `atlas:rps:{tenantId}.{region}.{major}.{minor}:{characterId}` | String (JSON) | Active RPS session (`game.Model`), keyed by tenant and character id. No native Redis TTL is set on the key itself - expiration is tracked via the sorted set below. |
| `atlas:rps:{tenantId}.{region}.{major}.{minor}:_expiry` | Sorted Set | Per-tenant expiry index; member is the session key above, score is the expiration time (Unix milliseconds). `Put` scores each member `now + defaultTTL` (5 minutes); `PopExpired` scans and removes members with a score `<= now`. |

### Tenant Tracking Set

| Key | Redis Type | Description |
|-----|------------|--------------|
| `atlas:rps:_tenants` | Set | JSON-marshaled `tenant.Model` values for every tenant that has had at least one session `Put`; used by the sweep to fan out `PopExpired` across tenants. |

## Relationships

Each session key and its corresponding entry in the tenant's `_expiry` sorted set are written together (pipelined) on `Put`, and removed together on `Remove` or when popped by `PopExpired`. The tenant-tracking set (`atlas:rps:_tenants`) is written alongside every `Put` so the sweep task can enumerate tenants without a separate index of active sessions.

## Indexes

| Index Key Pattern | Points To |
|--------------------|-----------|
| `atlas:rps:{tenantId}.{region}.{major}.{minor}:_expiry` | `atlas:rps:{tenantId}.{region}.{major}.{minor}:{characterId}` |

## Migration Rules

All state is ephemeral (TTL-bound). State is not preserved across restarts beyond the configured TTL. There are no schema migrations.
