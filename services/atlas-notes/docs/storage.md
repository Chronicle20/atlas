# Storage

## Tables

### notes

| Column | Type | Description |
|--------|------|-------------|
| id | uint32 | Primary key, auto-increment |
| tenant_id | uuid | Tenant identifier |
| character_id | uint32 | ID of the character who owns the note |
| sender_id | uint32 | ID of the character who sent the note |
| message | string | Note content |
| timestamp | time.Time | When the note was created |
| flag | byte | Note flag |
| created_at | time.Time | Record creation timestamp |
| updated_at | time.Time | Record update timestamp |
| deleted_at | gorm.DeletedAt | Soft delete timestamp |

## Relationships

None.

## Indexes

- Primary key on `id`
- Index on `deleted_at` (for soft delete queries)

## Migration Rules

- Schema is managed via GORM AutoMigrate
- Entity struct defines the table structure
