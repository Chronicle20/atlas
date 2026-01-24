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
| slot | int8 | NOT NULL, DEFAULT -1 | Spawn slot |
| flag | uint16 | NOT NULL, DEFAULT 0 | Pet flags |
| purchase_by | uint32 | NOT NULL, DEFAULT 0 | Purchaser character |

### excludes

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uint32 | PRIMARY KEY, AUTO_INCREMENT | Exclude identifier |
| pet_id | uint32 | NOT NULL | Pet foreign key |
| item_id | uint32 | NOT NULL | Excluded item identifier |

## Relationships

| Parent | Child | Type | Foreign Key |
|--------|-------|------|-------------|
| pets | excludes | one-to-many | pet_id |

## Indexes

GORM auto-migration manages indexes.

## Migration Rules

- Tables are created via GORM AutoMigrate
- Pet migration creates the pets table
- Exclude migration creates the excludes table
