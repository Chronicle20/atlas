# Storage

This service has no persistent database storage — no GORM models, no
migrations. It does hold short-retention working state in Redis: a bounded,
tenant-keyed buffer of recent player-authored chat lines, used to answer
`GET /api/chat/history` (see [REST](rest.md)) for report corroboration.

## Redis

### `chat:recent`

A tenant-keyed sorted set (per-tenant, per-character), one entry per
captured chat line, scored by capture timestamp (unix-milli). Implemented
via `libs/atlas-redis`'s `TenantKeyedSortedSet`; all keyed Redis access goes
through that library, never a raw `go-redis` client in this service.

Bounded on write by both age and count via `AddBounded`:

| Variable | Description | Default |
|----------|-------------|---------|
| `CHAT_CAPTURE_RETENTION_SECONDS` | Max age of a retained line, in seconds | 900 |
| `CHAT_CAPTURE_MAX_LINES` | Max lines retained per character | 200 |

Members are JSON-encoded `chat.Line` values keyed by sender character ID.
This is working state, not an archive: it exists only to let atlas-ban
snapshot a short transcript around the time a report is filed, and entries
age out on their own — there is no migration or backfill concern.

## Tables

None.

## Relationships

None.

## Indexes

None.

## Migration Rules

None.
