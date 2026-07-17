# Storage

## Tables

### gachapons

| Column | Type | Constraints |
|--------|------|-------------|
| uid | uuid | PRIMARY KEY, NOT NULL |
| tenant_id | uuid | NOT NULL |
| id | string | NOT NULL |
| name | string | NOT NULL |
| npc_ids | integer[] | NOT NULL |
| common_weight | uint32 | NOT NULL |
| uncommon_weight | uint32 | NOT NULL |
| rare_weight | uint32 | NOT NULL |

### gachapon_items

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| gachapon_id | string | NOT NULL |
| item_id | uint32 | NOT NULL |
| quantity | uint32 | NOT NULL, DEFAULT 1 |
| tier | string | NOT NULL |

### global_gachapon_items

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| item_id | uint32 | NOT NULL |
| quantity | uint32 | NOT NULL, DEFAULT 1 |
| tier | string | NOT NULL |

### seed_state

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | PRIMARY KEY (composite) |
| group_name | text | PRIMARY KEY (composite) |
| catalog_revision | text | NOT NULL |
| seeded_at | timestamp | NOT NULL |
| result_summary | jsonb | NOT NULL |

## Relationships

None.

## Indexes

| Table | Index Name | Columns |
|-------|-----------|---------|
| gachapons | idx_gachapons_tenant_slug | tenant_id, id (unique) |
| gachapon_items | idx_gachapon_items_tier | gachapon_id, tier |
| global_gachapon_items | idx_global_items_tier | tier |
| seed_state | (composite primary key) | tenant_id, group_name |

## Migration Rules

All tables use GORM AutoMigrate. Migrations run at service startup via `database.Connect` with registered migrators for `gachapon.Migration`, `item.Migration`, `global.Migration`, and `seeder.SeedState` (the last defined and migrated via the `github.com/Chronicle20/atlas/libs/atlas-seeder` shared library).

`gachapon.Migration` additionally runs a one-time transformation (`migrateToSurrogatePK`) on an existing `gachapons` table: it adds the surrogate `uid` primary key column, backfills it deterministically from `(tenant_id, id)`, drops the legacy slug-only primary key, repoints the primary key to `uid`, and creates the unique `idx_gachapons_tenant_slug` index on `(tenant_id, id)`. The transformation is idempotent and no-ops on a fresh database or an already-migrated one.
