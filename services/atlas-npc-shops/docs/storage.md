# Storage

## Tables

### shops

| Column     | Type      | Constraints     | Description                         |
|------------|-----------|-----------------|-------------------------------------|
| id         | uuid      | PRIMARY KEY     | Unique shop identifier              |
| tenant_id  | uuid      | NOT NULL        | Tenant identifier                   |
| npc_id     | uint32    | NOT NULL        | NPC template identifier             |
| recharger  | bool      | NOT NULL        | Whether shop supports recharging    |
| created_at | timestamp |                 | Record creation timestamp           |
| updated_at | timestamp |                 | Record update timestamp             |
| deleted_at | timestamp |                 | Soft delete timestamp               |

### commodities

| Column            | Type      | Constraints         | Description                              |
|-------------------|-----------|---------------------|------------------------------------------|
| id                | uuid      | PRIMARY KEY         | Unique commodity identifier              |
| tenant_id         | uuid      | NOT NULL            | Tenant identifier                        |
| npc_id            | uint32    | NOT NULL            | NPC template identifier                  |
| template_id       | uint32    | NOT NULL            | Item template identifier                 |
| meso_price        | uint32    | NOT NULL            | Price in mesos                           |
| discount_rate     | byte      | NOT NULL, DEFAULT 0 | Discount percentage                      |
| token_template_id | uint32    | NOT NULL, DEFAULT 0 | Alternative currency item identifier     |
| token_price       | uint32    | NOT NULL, DEFAULT 0 | Price in alternative currency            |
| period            | uint32    | NOT NULL, DEFAULT 0 | Time limit on purchase in minutes        |
| level_limit       | uint32    | NOT NULL, DEFAULT 0 | Minimum level required                   |
| created_at        | timestamp |                     | Record creation timestamp                |
| updated_at        | timestamp |                     | Record update timestamp                  |
| deleted_at        | timestamp |                     | Soft delete timestamp                    |

## Relationships

- shops.tenant_id references tenant identifier
- commodities.tenant_id references tenant identifier
- commodities.npc_id corresponds to shops.npc_id (logical relationship, no foreign key constraint)

## Indexes

- shops: primary key on id
- commodities: primary key on id

## Migration Rules

- Tables are auto-migrated using GORM AutoMigrate at service startup
- Soft deletes are enabled via the deleted_at column (GORM gorm.Model convention)
- Migrations run for both shops and commodities tables: `commodities.Migration`, `shops.Migration`
- The seed operation uses hard deletes (Unscoped) to clear existing data before re-seeding
