# Storage

## Tables

### skills

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | uuid | PRIMARY KEY, NOT NULL | Tenant identifier |
| character_id | uint32 | PRIMARY KEY, NOT NULL | Character identifier |
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

### Redis: Cooldown Registry

Skill cooldowns are stored in Redis using a tenant-scoped registry.

| Key Pattern | Value Type | Description |
|-------------|-----------|-------------|
| `atlas:cooldown:{tenantKey}:{characterId}:{skillId}` | JSON-encoded time.Time | Cooldown expiration timestamp |
| `atlas:cooldown:_tenants` | Redis Set of JSON-encoded tenant models | Tracks tenants with active cooldowns |

### outbox_entries

Provided by the shared `atlas-outbox` library (`outboxlib.Migration`, `main.go`). The transactional outbox table backing the outbox drainer. Its schema is owned by the library, not this service.

## Relationships

- Skills belong to a character within a tenant.
- Macros belong to a character within a tenant.
- The skills table uses a composite primary key of (tenant_id, character_id, id).
- The macros table uses a composite primary key of (character_id, id).

## Indexes

- skills: Composite primary key on (tenant_id, character_id, id); queries filter by character_id.
- macros: Composite primary key on (character_id, id); queries filter by character_id.

## Migration Rules

- Migrations are executed via GORM AutoMigrate on service startup.
- Tables are created if they do not exist.
- `skill.Migration`, `macro.Migration`, and `outboxlib.Migration` are registered at service startup via `database.Connect` (`main.go`).
