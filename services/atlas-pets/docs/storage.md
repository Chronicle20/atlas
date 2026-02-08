# Storage

## Tables

### pets

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uint32 | PRIMARY KEY, AUTO_INCREMENT | Pet identifier |
| tenant_id | uuid | NOT NULL | Tenant identifier |
| owner_id | uint32 | NOT NULL | Owning character identifier |
| cash_id | uint64 | NOT NULL | Cash shop identifier |
| template_id | uint32 | NOT NULL | Pet template reference |
| name | string(13) | NOT NULL | Pet name |
| level | byte | NOT NULL, DEFAULT 1 | Pet level |
| closeness | uint16 | NOT NULL, DEFAULT 0 | Pet closeness |
| fullness | byte | NOT NULL, DEFAULT 100 | Pet fullness |
| expiration | timestamp | NOT NULL | Pet expiration |
| slot | int8 | NOT NULL, DEFAULT -1 | Spawn slot (-1 = despawned, 0-2 = spawned) |
| flag | uint16 | NOT NULL, DEFAULT 0 | Pet flags |
| purchase_by | uint32 | NOT NULL, DEFAULT 0 | Purchaser character identifier |

### excludes

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uint32 | PRIMARY KEY, AUTO_INCREMENT | Exclude identifier |
| pet_id | uint32 | NOT NULL | Pet foreign key |
| item_id | uint32 | NOT NULL | Excluded item identifier |

## Relationships

| Parent | Child | Type | Foreign Key |
|--------|-------|------|-------------|
| pets | excludes | one-to-many | excludes.pet_id |

## Indexes

GORM auto-migration manages indexes. The primary keys on `pets.id` and `excludes.id` are auto-incremented. The foreign key relationship from `excludes.pet_id` to `pets.id` is managed by GORM via the `foreignkey:PetId` struct tag.

## Migration Rules

- Tables are created via GORM AutoMigrate at service startup
- Pet migration (`pet.Migration`) creates the `pets` table
- Exclude migration (`exclude.Migration`) creates the `excludes` table
- Both migrations are registered in `main.go` via `database.SetMigrations(pet.Migration, exclude.Migration)`
- Excludes are replaced atomically: existing excludes for a pet are deleted, then new ones are inserted, within a single transaction
