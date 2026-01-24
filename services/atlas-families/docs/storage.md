# Storage

## Tables

### family_members

Primary table for storing family member data.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uint32 | PRIMARY KEY, AUTO INCREMENT | Internal identifier |
| character_id | uint32 | UNIQUE, NOT NULL | Game character identifier |
| tenant_id | uuid | NOT NULL, INDEX | Multi-tenant identifier |
| senior_id | *uint32 | INDEX | Senior character ID reference |
| junior_ids | []uint32 | JSON serialized | Junior character IDs |
| rep | uint32 | DEFAULT 0 | Total reputation |
| daily_rep | uint32 | DEFAULT 0 | Daily reputation |
| level | uint16 | NOT NULL | Character level |
| world | byte | NOT NULL | World identifier |
| created_at | timestamp | NOT NULL | Creation timestamp |
| updated_at | timestamp | NOT NULL | Last update timestamp |

## Relationships

- senior_id references character_id of another family_members row
- junior_ids contains character_id values of other family_members rows
- Self-referential hierarchy (tree structure)

## Indexes

| Index Name | Columns | Type | Description |
|------------|---------|------|-------------|
| PRIMARY | id | Primary Key | Unique row identifier |
| idx_character_id | character_id | Unique | Character lookup |
| idx_tenant_id | tenant_id | Index | Tenant filtering |
| idx_family_members_tenant_character | tenant_id, character_id | Composite | Tenant-scoped character lookup |
| idx_family_members_world | world | Index | World filtering |
| idx_family_members_updated_at | updated_at | Index | Temporal queries |
| idx_family_members_senior_id | senior_id | Index (WHERE NOT NULL for PostgreSQL) | Senior lookup |

## Migration Rules

### Constraints (PostgreSQL)

| Constraint Name | Rule | Description |
|-----------------|------|-------------|
| check_junior_count | array_length(junior_ids, 1) <= 2 | Maximum 2 juniors |
| check_rep_non_negative | rep >= 0 | Non-negative reputation |
| check_daily_rep_non_negative | daily_rep >= 0 | Non-negative daily rep |
| check_daily_rep_limit | daily_rep <= 5000 | Daily reputation cap |
| check_level_positive | level > 0 | Positive level |
| check_no_self_senior | senior_id != character_id | No self-reference |

### Migration Behavior

- AutoMigrate creates table from Entity struct
- Indexes created via raw SQL after AutoMigrate
- Constraints added conditionally based on database dialect
- SQLite: constraints handled at application level
- PostgreSQL: database-level constraints enforced
