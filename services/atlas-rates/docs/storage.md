# Storage

## Tables

None. This service does not use a relational database.

## Redis

All state is stored in Redis via `atlas.TenantRegistry`.

| Registry | Namespace | Key | Value | Description |
|----------|-----------|-----|-------|-------------|
| Registry | `rates` | `{tenant}:{characterId}` | `character.Model` (JSON) | Rate factors per character |
| ItemTracker | `rates-items` | `{tenant}:{characterId}:{templateId}` | `character.TrackedItem` (JSON) | Time-based rate items |
| initializedRegistry | `rates-init` | `{tenant}:{characterId}` | `bool` | Tracks which characters have been lazily initialized |

State persists across service restarts via Redis. Lazy initialization re-queries external services only for characters not yet marked as initialized.

## Relationships

Not applicable.

## Indexes

Not applicable.

## Migration Rules

Not applicable.
