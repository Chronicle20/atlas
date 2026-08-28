# Rankings Storage

This service uses Postgres for ranking and cycle state. Redis is used only for the leader-election lock (see below); it holds no ranking data.

## Tables

### character_rankings

One row per ranked character per tenant. Backed by `Entity`.

| Column | Type | Description |
|--------|------|-------------|
| tenant_id | uuid | Tenant id; not null |
| id | uuid | Primary key; generated on create if not set |
| character_id | uint32 | Character id; not null |
| name | string | Character name; not null, default '' |
| world_id | world.Id | World id; not null |
| job_category | uint16 | Job category (`jobId / 100`); not null |
| level | byte | Character level; not null, default 0 |
| job_id | job.Id | Job id; not null, default 0 |
| overall_rank | uint32 | 1-based rank within the world; not null |
| overall_rank_move | int32 | Change from the previous cycle's overall rank; not null |
| job_rank | uint32 | 1-based rank within the world and job category; not null |
| job_rank_move | int32 | Change from the previous cycle's job rank; not null |
| computed_at | time.Time | Timestamp of the cycle that produced this row; not null |

0 is never stored as a rank; an unranked character has no row.

### ranking_cycles

One row per tenant, tracking recompute cadence and observability. Backed by `CycleEntity`.

| Column | Type | Description |
|--------|------|-------------|
| tenant_id | uuid | Tenant id; not null |
| id | uuid | Primary key; generated on create if not set |
| last_started_at | time.Time | When the most recent cycle began; not null |
| last_completed_at | *time.Time | When the most recent cycle finished (nil if never completed) |
| characters_ranked | uint32 | Number of characters ranked in the most recent completed cycle |
| duration_ms | uint32 | Wall-clock duration of the most recent completed cycle, in milliseconds |

## Relationships

Neither table has a foreign-key relationship to the other or to any other table in this schema. `character_rankings` rows reference characters and worlds owned by other services (atlas-character) by id only.

## Indexes

| Index | Table | Columns | Type |
|-------|-------|---------|------|
| idx_rankings_tenant_character | character_rankings | tenant_id, character_id | unique |
| idx_rankings_tenant_world | character_rankings | tenant_id, world_id | non-unique |
| (unnamed) | ranking_cycles | tenant_id | unique |

The leaderboard read path additionally orders by `world_id`, `overall_rank` (overall view) or `job_category`, `job_rank` (job-category view); no separate index is declared for these orderings beyond `idx_rankings_tenant_world`.

## Migration Rules

Schema is managed via `gorm.AutoMigrate` against `Entity` and `CycleEntity` (`Migration` in `ranking/entity.go`), run at service startup. Upserts on `character_rankings` conflict on `(tenant_id, character_id)`; upserts on `ranking_cycles` conflict on `tenant_id`.

## Leader-Election Lock

| Key | Store | Description |
|-----|-------|--------------|
| `rankings-recompute` | Redis (`atlas-lock`) | Distributed lock gating the recompute task so only the leader pod runs it per tick; TTL, refresh, and backoff are configurable. |
