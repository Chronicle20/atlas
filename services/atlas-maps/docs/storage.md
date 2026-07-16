# Storage

## Tables

### character_map_visits

Records the first time a character visits a map.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | Primary key |
| tenant_id | UUID | Not null |
| character_id | uint32 | Not null |
| map_id | uint32 | Not null |
| first_visited_at | timestamp | Not null, default CURRENT_TIMESTAMP |

### character_locations

Persists a character's last-known field (world, channel, map, instance).

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | UUID | Primary key (composite with character_id) |
| character_id | uint32 | Primary key (composite with tenant_id) |
| world_id | world.Id | Not null |
| channel_id | channel.Id | Not null |
| map_id | uint32 | Not null |
| instance | UUID | Not null, default '00000000-0000-0000-0000-000000000000' |
| updated_at | timestamp | Not null |

## Relationships

None.

## Indexes

| Name | Columns | Type |
|------|---------|------|
| idx_visits_tenant_char_map | tenant_id, character_id, map_id | Unique |
| idx_visits_tenant_char | tenant_id, character_id | Non-unique |
| (primary key) | tenant_id, character_id (character_locations) | Unique (composite primary key) |

## Migration Rules

- Table migration via GORM AutoMigrate on service startup
- Schema changes are additive

## In-Memory Registries

### Character Registry

Singleton registry tracking character presence in maps. State is not persisted and is rebuilt from events on service restart.

| Key | Value |
|-----|-------|
| MapKey | []uint32 (character IDs) |

### Spawn Point Registry (Redis)

Redis-backed registry tracking spawn point cooldowns. Lazily initialized from atlas-data on first access per map. Uses Lua scripts for atomic eligibility checks and cooldown updates.

| Key Pattern | Type | Value |
|-------------|------|-------|
| atlas:maps:spawn:{tenant}:{worldId}:{channelId}:{mapId}:{instance} | Hash | field: spawn point ID, value: JSON-encoded spawn point with NextSpawnAt (Unix ms) |

### Weather Registry

Singleton registry tracking active weather effects per map instance. State is not persisted. Expired entries are removed by the weather task.

| Key | Value |
|-----|-------|
| FieldKey | WeatherEntry |

### Map Timer Registry

Singleton registry tracking per-character map-stay timer entries. State is not persisted and is rebuilt as characters change maps after a service restart.

| Key | Value |
|-----|-------|
| (tenant, characterId) | Map Timer Entry |

### Map Info Cache

Process-local cache holding Map Info Models retrieved from atlas-data. State is not persisted.

| Key | Value |
|-----|-------|
| (tenant, mapId) | Data Map Info Model |

### Mist Registry

Tenant-scoped, in-memory registry of active Mist values, keyed by mist id within each tenant's bucket. State is not persisted. Expired entries are removed by the mist tick task.

| Key | Value |
|-----|-------|
| (tenant, mistId) | Mist |
