# Storage

## Tables

### skills

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | uuid | NOT NULL | Tenant identifier |
| character_id | uint32 | NOT NULL | Character identifier |
| id | uint32 | PRIMARY KEY, NOT NULL | Skill identifier |
| level | byte | NOT NULL | Current skill level |
| master_level | byte | NOT NULL | Master level |
| expiration | timestamp | NOT NULL | Expiration time |

### macros

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | uuid | NOT NULL | Tenant identifier |
| character_id | uint32 | PRIMARY KEY, NOT NULL | Character identifier |
| id | uint32 | PRIMARY KEY, NOT NULL, AUTO INCREMENT FALSE | Macro identifier |
| name | string | NOT NULL | Macro display name |
| shout | bool | NOT NULL | Shout flag |
| skill_id_1 | uint32 | NOT NULL | First skill ID |
| skill_id_2 | uint32 | NOT NULL | Second skill ID |
| skill_id_3 | uint32 | NOT NULL | Third skill ID |

## Relationships

- Skills belong to a character within a tenant.
- Macros belong to a character within a tenant.
- The macros table uses a composite primary key of (character_id, id).

## Indexes

- skills: Primary key on id; queries filter by tenant_id and character_id.
- macros: Composite primary key on (character_id, id); queries filter by tenant_id and character_id.

## Migration Rules

- Migrations are executed via GORM AutoMigrate on service startup.
- Tables are created if they do not exist.
