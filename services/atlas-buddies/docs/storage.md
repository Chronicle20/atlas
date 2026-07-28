# Storage

## Tables

### lists

Stores buddy list metadata for each character.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | uuid | NOT NULL | Tenant identifier for multi-tenancy |
| id | uuid | PRIMARY KEY, DEFAULT uuid_generate_v4() | Unique identifier |
| character_id | uint32 | NOT NULL | Owner character ID |
| capacity | byte | NOT NULL | Maximum buddy capacity |

### buddies

Stores individual buddy entries within buddy lists.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| character_id | uint32 | PRIMARY KEY | Buddy's character ID |
| list_id | uuid | NOT NULL, FOREIGN KEY | Reference to parent list |
| tenant_id | uuid | INDEXED | Tenant identifier for multi-tenancy |
| group | string | NOT NULL | Buddy group name |
| character_name | string | NOT NULL | Buddy's display name |
| channel_id | int8 | NOT NULL, DEFAULT -1 | Current channel (-1 if offline) |
| in_shop | bool | NOT NULL, DEFAULT false | Whether buddy is in cash shop |
| pending | bool | NOT NULL, DEFAULT false | Whether buddy relationship is pending |

### outbox_entries

Provided by the shared `atlas-outbox` library (`outboxlib.Migration`, `main.go`). The transactional outbox table backing the outbox drainer. Its schema is owned by the library, not this service.

---

## Relationships

```
lists (1) ──── (N) buddies
       └── list_id (FK)
```

- One `lists` entry has many `buddies` entries
- `buddies.list_id` references `lists.id`

---

## Indexes

GORM auto-migration creates:
- Primary key index on `lists.id`
- Primary key index on `buddies.character_id`
- Foreign key index on `buddies.list_id`
- Index on `buddies.tenant_id`

---

## Migration Rules

- Migrations are executed via GORM AutoMigrate
- `list.Migration`, `buddy.Migration`, and `outboxlib.Migration` are registered at service startup (`main.go`)
- Schema changes are applied automatically on service start
- `buddy.Migration` additionally backfills `buddies.tenant_id` from the owning `lists.tenant_id` for any row where `tenant_id` is null or the zero UUID
