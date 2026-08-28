# Event Storage

This service uses PostgreSQL via GORM for all persistent state. Every table listed below (except `seed_state`, owned by the shared `atlas-seeder` library) carries a `tenant_id` column scoped by the shared tenant-filter GORM callback.

## Tables

### event_definition

| Column | Type | Description |
|--------|------|-------------|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| type | text | Not null; indexed (`ix_def_tenant_type`, priority 2) |
| name | text | Not null |
| enabled | boolean | Not null; default false |
| configuration | jsonb | Not null; opaque, handler-interpreted |
| created_at | timestamp | Not null; default now |
| updated_at | timestamp | Not null; default now |

### event_occurrence

| Column | Type | Description |
|--------|------|-------------|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| event_definition_id | uuid | Not null |
| type | text | Not null |
| state | text | Not null: ACTIVE / COMPLETED / CANCELLED / FAILED |
| stage | text | Handler-defined stage |
| context | jsonb | Not null; default `'{}'`; opaque, handler-interpreted |
| world_id | smallint (nullable) | Promoted from context to a scalar column for index-served gameplay queries |
| channel_id | smallint (nullable) | Promoted from context, same reason as world_id |
| voyage_id | uuid (nullable) | Promoted from context; null when the occurrence has no voyage scope |
| concurrency_key | text | Not null; default `''` |
| started_at | timestamp | Not null |
| next_transition_at | timestamp (nullable) | |
| completed_at | timestamp (nullable) | |
| completion_reason | text | |

### event_occurrence_map

Child table: which maps an occurrence is scoped to, and whether each is visual.

| Column | Type | Description |
|--------|------|-------------|
| tenant_id | uuid | Not null; carried directly (not joined through the parent) so the tenant callback scopes this table on its own |
| occurrence_id | uuid | Primary key (composite with map_id) |
| map_id | uint32 | Primary key (composite with occurrence_id) |
| visual | boolean | Not null; default false — distinguishes the deck (gets the visual) from a cabin/related map (counts toward scope only) |

### event_occurrence_monster

Child table: the SET of monsters an occurrence has spawned or observed, and whether each is currently alive.

| Column | Type | Description |
|--------|------|-------------|
| tenant_id | uuid | Not null; carried directly, same reason as event_occurrence_map |
| occurrence_id | uuid | Primary key (composite with unique_id) |
| unique_id | uint32 | Primary key (composite with occurrence_id) |
| monster_id | uint32 | Not null |
| alive | boolean | Not null; every write path sets this explicitly (no column default, to avoid resurrecting a KILLED-before-CREATED row) |
| observed_at | timestamp | Not null |

### event_occurrence_transition

One row per stage transition of an occurrence — its history.

| Column | Type | Description |
|--------|------|-------------|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| occurrence_id | uuid | Not null; indexed (`ix_trans_occurrence`) |
| from_stage | text | Empty for the creation row |
| to_stage | text | Not null |
| occurred_at | timestamp | Not null |
| trigger_type | text | Not null |
| trigger_reference | text | |

### scheduled_event_work

One row per unit of durable, delayed event work, claimed and executed by the cross-tenant poller.

| Column | Type | Description |
|--------|------|-------------|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| tenant_region | text | Not null; default `''` — denormalized tenant identity |
| tenant_major | smallint | Not null; default 0 — denormalized tenant identity |
| tenant_minor | smallint | Not null; default 0 — denormalized tenant identity |
| event_definition_id | uuid | Not null |
| event_occurrence_id | uuid (nullable) | Null for work that predates an occurrence |
| type | text | Not null: TRIGGER_EVALUATION / OCCURRENCE_TRANSITION |
| context | jsonb | Not null; default `'{}'`; opaque, handler-interpreted |
| execute_at | timestamp | Not null |
| state | text | Not null: PENDING / PROCESSING / COMPLETED / CANCELLED / FAILED |
| claimed_by | text | The claiming replica's instance id |
| claimed_at | timestamp (nullable) | |
| attempts | int | Not null; default 0 |
| last_error | text | |
| dedupe_key | text | Not null; default `''` — empty opts a row out of dedup |

### seed_state

Owned by the shared `atlas-seeder` library, migrated by this service alongside its own tables. Tracks per-tenant, per-seed-group catalog application state (`tenant_id`, `group_name` composite primary key, `catalog_revision`, `seeded_at`, `result_summary`).

## Relationships

- `event_occurrence.event_definition_id` references `event_definition.id`.
- `event_occurrence_map.occurrence_id` and `event_occurrence_monster.occurrence_id` reference `event_occurrence.id`.
- `event_occurrence_transition.occurrence_id` references `event_occurrence.id`.
- `scheduled_event_work.event_definition_id` references `event_definition.id`; `scheduled_event_work.event_occurrence_id` references `event_occurrence.id` when set.

## Indexes

| Index | Table | Columns / Predicate |
|-------|-------|----------------------|
| ix_def_tenant_type | event_definition | (tenant_id priority 1, type priority 2) |
| ix_occ_type_state | event_occurrence | (tenant_id, type, state) |
| ux_occ_concurrency | event_occurrence | UNIQUE (tenant_id, event_definition_id, concurrency_key) WHERE state = 'ACTIVE' AND concurrency_key <> '' |
| ix_occ_map | event_occurrence_map | (map_id, occurrence_id) WHERE visual = true |
| ix_occ_active_scope | event_occurrence | (tenant_id, world_id, channel_id, state) WHERE state = 'ACTIVE' |
| ix_trans_occurrence | event_occurrence_transition | occurrence_id |
| ix_sew_pending_due | scheduled_event_work | execute_at WHERE state = 'PENDING' |
| ix_sew_processing_claimed | scheduled_event_work | claimed_at WHERE state = 'PROCESSING' |
| ux_sew_dedupe | scheduled_event_work | UNIQUE (tenant_id, dedupe_key) WHERE state IN ('PENDING','PROCESSING') AND dedupe_key <> '' |

The legacy `ux_event_occurrence_concurrency_key` index (no state predicate) is dropped during migration in favor of `ux_occ_concurrency`.

## Migration Rules

Tables are created via `gorm.AutoMigrate` for each entity (`definition.MigrateTable`, `occurrence.MigrateTable`, `transition.MigrateTable`, `scheduling.MigrateTable`), plus `seeder.SeedState`, run from `main.go` in that order at startup. `occurrence.MigrateTable` and `scheduling.MigrateTable` additionally create their partial indexes with `CREATE INDEX IF NOT EXISTS`/`CREATE UNIQUE INDEX IF NOT EXISTS` (idempotent) and drop the superseded `ux_event_occurrence_concurrency_key` index with `DROP INDEX IF EXISTS`.
