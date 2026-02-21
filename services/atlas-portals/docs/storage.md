# Storage

## Redis

### Keys

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `atlas:blocked-portal:{tenantKey}:{characterId}` | Set | Blocked portals for a character within a tenant |

### Members

Set members use the format `{mapId}:{portalId}`.

### Lifecycle

- Members are added on `BLOCK` commands.
- Members are removed on `UNBLOCK` commands.
- The entire key is deleted when a character logs out.
