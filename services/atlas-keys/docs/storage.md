# Storage

## Tables

### keys

Stores key bindings for characters.

| Column      | Type      | Constraints                          |
|-------------|-----------|--------------------------------------|
| TenantId    | uuid.UUID | NOT NULL                             |
| CharacterId | uint32    | PRIMARY KEY, NOT NULL                |
| Key         | int32     | PRIMARY KEY, NOT NULL                |
| Type        | int8      | NOT NULL                             |
| Action      | int32     | NOT NULL                             |

## Relationships

None.

## Indexes

- Composite primary key on (CharacterId, Key).

## Migration Rules

- Migrations are executed via GORM AutoMigrate on the `entity` struct.
