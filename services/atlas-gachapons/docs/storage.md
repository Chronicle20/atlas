# Storage

## Tables

### gachapons

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | string | PRIMARY KEY, NOT NULL |
| name | string | NOT NULL |
| npc_ids | integer[] | NOT NULL |
| common_weight | uint32 | NOT NULL, DEFAULT 70 |
| uncommon_weight | uint32 | NOT NULL, DEFAULT 25 |
| rare_weight | uint32 | NOT NULL, DEFAULT 5 |

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

## Relationships

None.

## Indexes

| Table | Index Name | Columns |
|-------|-----------|---------|
| gachapon_items | idx_gachapon_items_tier | gachapon_id, tier |
| global_gachapon_items | idx_global_items_tier | tier |

## Migration Rules

All tables use GORM AutoMigrate. Migrations run at service startup via `database.Connect` with registered migrators for `gachapon.Migration`, `item.Migration`, and `global.Migration`.
