# Storage

## Tables

### documents

Stores all game data documents as JSON blobs with tenant isolation.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT uuid_generate_v4() | Unique document identifier |
| tenant_id | UUID | NOT NULL, part of unique index `idx_documents_tenant_type_docid` | Tenant identifier for multi-tenancy (may be a real tenant or the version-scoped canonical tenant id) |
| type | VARCHAR | NOT NULL, part of unique index `idx_documents_tenant_type_docid` | Document type discriminator |
| document_id | INTEGER (uint32) | NOT NULL, part of unique index `idx_documents_tenant_type_docid` | Domain-specific document identifier |
| content | JSON | NOT NULL | JSON representation of the document |
| updated_at | TIMESTAMP | auto-update on write | Last write time; drives `GET /api/data/status` |

### map_search_index

Trigram-searchable projection of MAP documents, written in the same transaction as the owning `documents` row.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY (composite) | Tenant identifier |
| map_id | INTEGER (uint32) | PRIMARY KEY (composite) | Map id |
| name | VARCHAR | NOT NULL | Map name |
| street_name | VARCHAR | NOT NULL | Map street name |
| updated_at | TIMESTAMP | auto-update on write | Last write time |

### npc_search_index

Trigram-searchable projection of NPC documents.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY (composite) | Tenant identifier |
| npc_id | INTEGER (uint32) | PRIMARY KEY (composite) | NPC id |
| name | VARCHAR | NOT NULL | NPC name |
| storebank | BOOLEAN | NOT NULL, default false | Storebank flag |
| updated_at | TIMESTAMP | auto-update on write | Last write time |

### monster_search_index

Trigram-searchable projection of MONSTER documents.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY (composite) | Tenant identifier |
| monster_id | INTEGER (uint32) | PRIMARY KEY (composite) | Monster id |
| name | VARCHAR | NOT NULL | Monster name |
| updated_at | TIMESTAMP | auto-update on write | Last write time |

### reactor_search_index

Trigram-searchable projection of REACTOR documents.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY (composite) | Tenant identifier |
| reactor_id | INTEGER (uint32) | PRIMARY KEY (composite) | Reactor id |
| name | VARCHAR | NOT NULL | Reactor name |
| updated_at | TIMESTAMP | auto-update on write | Last write time |

### item_string_search_index

Trigram-searchable projection of ITEM_STRING documents, with item classification columns used for compartment/subcategory/class filtering.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY (composite) | Tenant identifier |
| item_id | INTEGER (uint32) | PRIMARY KEY (composite) | Item id |
| name | VARCHAR | NOT NULL | Item name |
| compartment | SMALLINT | NOT NULL, default 0 | `inventory.Type` classification |
| subcategory | TEXT | NOT NULL, default '' | Subcategory label within the compartment |
| job_mask | SMALLINT | nullable | Equipment job-class bitmask (null for non-equipment) |
| updated_at | TIMESTAMP | auto-update on write | Last write time |

### monster_spawn_index

Per-map monster spawn counts, written alongside `map_search_index` whenever a MAP document is added (one row per distinct monster template spawning on the map).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY (composite) | Tenant identifier |
| monster_id | INTEGER (uint32) | PRIMARY KEY (composite) | Monster id |
| map_id | INTEGER (uint32) | PRIMARY KEY (composite) | Map id |
| name | VARCHAR | NOT NULL | Map name |
| street_name | VARCHAR | NOT NULL | Map street name |
| spawn_count | INTEGER (uint32) | NOT NULL | Count of this monster's spawns on the map |
| updated_at | TIMESTAMP | auto-update on write | Last write time |

### npc_spawn_index

Per-map NPC spawn counts, written alongside `map_search_index` whenever a MAP document is added.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY (composite) | Tenant identifier |
| npc_id | INTEGER (uint32) | PRIMARY KEY (composite) | NPC id |
| map_id | INTEGER (uint32) | PRIMARY KEY (composite) | Map id |
| name | VARCHAR | NOT NULL | Map name |
| street_name | VARCHAR | NOT NULL | Map street name |
| spawn_count | INTEGER (uint32) | NOT NULL | Count of this NPC's spawns on the map |
| updated_at | TIMESTAMP | auto-update on write | Last write time |

### tenant_baselines

Tracks the last baseline restored into each tenant.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY | Target tenant id |
| region | VARCHAR | NOT NULL | Restored baseline's region |
| major_version | INTEGER | NOT NULL | Restored baseline's major version |
| minor_version | INTEGER | NOT NULL | Restored baseline's minor version |
| baseline_sha256 | VARCHAR | NOT NULL | sha256 of the restored dump |
| restored_at | VARCHAR | NOT NULL, default now() | Restore timestamp |

## Relationships

The `documents` table stores all data types in a single table with the `type` column discriminating between document types.

Document types:
- CASH
- CHARACTER_TEMPLATE
- COMMODITY
- CONSUMABLE
- EQUIPMENT
- ETC
- FACE
- HAIR
- ITEM_STRING
- MAP
- MOB_SKILL
- MONSTER
- NPC
- PET
- QUEST
- REACTOR
- SETUP
- SKILL

`map_search_index`, `npc_search_index`, `monster_search_index`, `reactor_search_index`, and `item_string_search_index` each mirror a subset of columns from the corresponding `documents` type (MAP, NPC, MONSTER, REACTOR, ITEM_STRING respectively), keyed by `(tenant_id, <entity>_id)`. `monster_spawn_index` and `npc_spawn_index` are derived from MAP documents (one row per `(tenant_id, monster_id or npc_id, map_id)` combination present in that map's spawn list). `tenant_baselines` is keyed by `tenant_id` alone and is independent of `documents`.

## Indexes

- `documents`: unique index `idx_documents_tenant_type_docid` on `(tenant_id, type, document_id)`.
- `map_search_index`, `npc_search_index`, `monster_search_index`, `reactor_search_index`, `item_string_search_index`: composite primary key `(tenant_id, <entity>_id)`, plus a GIN trigram index (`gin_trgm_ops`, via the `pg_trgm` extension) on `LOWER(name)` (and, for `map_search_index`, also on `LOWER(street_name)`).
- `npc_search_index` additionally has a partial index on `(tenant_id, storebank) WHERE storebank = true`.
- `item_string_search_index` additionally has indexes on `(tenant_id, compartment)` and `(tenant_id, compartment, subcategory)`.
- `monster_spawn_index`: index `idx_monster_spawn_index_lookup` on `(tenant_id, monster_id, spawn_count DESC)`.
- `npc_spawn_index`: index `idx_npc_spawn_index_lookup` on `(tenant_id, npc_id, spawn_count DESC)`.
- `tenant_baselines`: primary key on `tenant_id`.

## Migration Rules

- Migrations are executed via GORM AutoMigrate, run in this order on service startup: `documents`, `map_search_index`, `npc_search_index`, `monster_search_index`, `monster_spawn_index`, `npc_spawn_index`, `reactor_search_index`, `item_string_search_index`, `tenant_baselines`.
- Search-index migrations additionally enable the `pg_trgm` Postgres extension (`CREATE EXTENSION IF NOT EXISTS pg_trgm`) and create their trigram/lookup indexes with `CREATE INDEX IF NOT EXISTS`.
- All tables are created automatically on service startup; schema changes are applied incrementally by GORM (plus the raw `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` statements the `item_string_search_index` migration issues for its classification columns).
- Migrations only run against the REST-mode process; `MODE=ingest` pods connect to the database without running migrations.
