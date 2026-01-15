# Storage

## Tables

### compartments

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uuid | PRIMARY KEY |
| character_id | uint32 | NOT NULL |
| inventory_type | int | NOT NULL |
| capacity | uint32 | |

### assets

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT |
| compartment_id | uuid | NOT NULL |
| slot | int16 | NOT NULL |
| template_id | uint32 | NOT NULL |
| expiration | timestamp | NOT NULL |
| reference_id | uint32 | NOT NULL |
| reference_type | string | NOT NULL |

### stackables

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT |
| compartment_id | uuid | NOT NULL |
| quantity | uint32 | NOT NULL |
| owner_id | uint32 | NOT NULL |
| flag | uint16 | NOT NULL |
| rechargeable | uint64 | NOT NULL, DEFAULT 0 |

---

## Relationships

- `compartments.character_id` references character (external)
- `assets.compartment_id` references `compartments.id`
- `assets.reference_id` references type-specific table based on `reference_type`:
  - `equipable` - equipables table (external service)
  - `cash-equipable` - cash items table (external service)
  - `consumable`, `setup`, `etc` - `stackables.id`
  - `cash` - cash items table (external service)
  - `pet` - pets table (external service)
- `stackables.compartment_id` references `compartments.id`

---

## Indexes

Managed by GORM AutoMigrate.

---

## Migration Rules

- Migrations executed via GORM AutoMigrate on service startup
- Migration order: compartment, asset, stackable
