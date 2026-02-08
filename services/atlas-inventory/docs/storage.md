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
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| compartment_id | uuid | NOT NULL |
| slot | int16 | NOT NULL |
| template_id | uint32 | NOT NULL |
| expiration | timestamp | NOT NULL |
| created_at | timestamp | NOT NULL |
| deleted_at | timestamp | INDEX (soft delete) |
| quantity | uint32 | |
| owner_id | uint32 | |
| flag | uint16 | |
| rechargeable | uint64 | |
| strength | uint16 | |
| dexterity | uint16 | |
| intelligence | uint16 | |
| luck | uint16 | |
| hp | uint16 | |
| mp | uint16 | |
| weapon_attack | uint16 | |
| magic_attack | uint16 | |
| weapon_defense | uint16 | |
| magic_defense | uint16 | |
| accuracy | uint16 | |
| avoidability | uint16 | |
| hands | uint16 | |
| speed | uint16 | |
| jump | uint16 | |
| slots | uint16 | |
| locked | bool | |
| spikes | bool | |
| karma_used | bool | |
| cold | bool | |
| can_be_traded | bool | |
| level_type | byte | |
| level | byte | |
| experience | uint32 | |
| hammers_applied | uint32 | |
| equipped_since | timestamp | nullable |
| cash_id | int64 | |
| commodity_id | uint32 | |
| purchase_by | uint32 | |
| pet_id | uint32 | |

---

## Relationships

- `compartments.character_id` references character (external)
- `assets.compartment_id` references `compartments.id`

---

## Indexes

- `assets.deleted_at` - indexed for soft delete queries (GORM DeletedAt)
- Additional indexes managed by GORM AutoMigrate

---

## Migration Rules

- Migrations executed via GORM AutoMigrate on service startup
- Migration order: compartment, asset
- Assets use soft delete via GORM `DeletedAt` field
- UUID generation for compartment IDs handled in `BeforeCreate` hook
