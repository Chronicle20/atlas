# Quest Storage

## Tables

### quest_statuses

Stores quest status records for characters.

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| id | uint32 | No | auto | Primary key |
| tenant_id | uuid | No | | Tenant identifier |
| character_id | uint32 | No | | Character identifier |
| quest_id | uint32 | No | | Quest definition identifier |
| state | byte | No | 0 | Quest state |
| started_at | timestamp | No | | When quest was started |
| completed_at | timestamp | Yes | | When quest was completed |
| expiration_time | timestamp | Yes | | When quest expires |
| completed_count | uint32 | No | 0 | Times completed |
| forfeit_count | uint32 | No | 0 | Times forfeited |

### quest_progress

Stores progress entries for quest objectives.

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| id | uint32 | No | auto | Primary key |
| tenant_id | uuid | No | | Tenant identifier |
| quest_status_id | uint32 | No | | Foreign key to quest_statuses |
| info_number | uint32 | No | | Objective identifier |
| progress | string | No | '' | Progress value |

### quest_medal_maps

Stores a visited map for a medal quest.

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| id | uint32 | No | auto | Primary key |
| quest_status_id | uint32 | No | | Quest status identifier |
| map_id | uint32 | No | | Map identifier |

This table's Migration is defined but is not included in main.go's `SetMigrations` list, and the table is not written to or read from by any Processor.

## Relationships

| Parent | Child | Type | Constraint |
|--------|-------|------|------------|
| quest_statuses | quest_progress | One-to-Many | quest_status_id |

## Indexes

### quest_statuses

| Name | Columns | Description |
|------|---------|--------------|
| idx_quest_tenant_char | tenant_id, character_id | Lookup by tenant and character |
| idx_quest_statuses_quest_id | quest_id | Lookup by quest definition |

### quest_progress

| Name | Columns | Description |
|------|---------|--------------|
| idx_quest_progress_tenant_id | tenant_id | Lookup by tenant |
| idx_quest_progress_quest_status_id | quest_status_id | Lookup by parent status |

### quest_medal_maps

| Name | Columns | Description |
|------|---------|--------------|
| idx_medal_quest_map | quest_status_id, map_id | Unique index on quest status and map |

## Migration Rules

- Tables are auto-migrated via GORM AutoMigrate
- Only quest_statuses and quest_progress are registered for migration in main.go; quest_medal_maps is not
- Quest progress entries are deleted when parent quest status is deleted (cascading delete in code)
