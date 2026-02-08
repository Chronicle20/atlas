# Storage

## Tables

### accounts

Stores wallet information for accounts.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Unique identifier |
| tenant_id | uuid | NOT NULL | Tenant identifier for multi-tenancy |
| account_id | uint32 | NOT NULL | Associated account |
| credit | uint32 | NOT NULL, DEFAULT 0 | Credit currency balance |
| points | uint32 | NOT NULL, DEFAULT 0 | Points currency balance |
| prepaid | uint32 | NOT NULL, DEFAULT 0 | Prepaid currency balance |

### wishlist_items

Stores wishlist items for characters.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY, DEFAULT uuid_generate_v4() | Unique identifier |
| tenant_id | uuid | NOT NULL | Tenant identifier for multi-tenancy |
| character_id | uint32 | NOT NULL | Owner character |
| serial_number | uint32 | NOT NULL | Serial number of wished commodity |

### cash_compartments

Stores cash shop inventory compartments.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY, DEFAULT uuid_generate_v4() | Unique identifier |
| tenant_id | uuid | NOT NULL | Tenant identifier for multi-tenancy |
| account_id | uint32 | NOT NULL | Associated account |
| type | byte | NOT NULL | Compartment type (1=Explorer, 2=Cygnus, 3=Legend) |
| capacity | uint32 | NOT NULL, DEFAULT 55 | Maximum number of assets |

### cash_assets

Stores cash shop assets with all item data flattened directly into the row.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uint32 | PRIMARY KEY, AUTO INCREMENT | Unique identifier |
| tenant_id | uuid | NOT NULL | Tenant identifier for multi-tenancy |
| compartment_id | uuid | NOT NULL | Parent compartment |
| cash_id | int64 | NOT NULL | Unique cash item identifier |
| template_id | uint32 | NOT NULL | Item template ID |
| commodity_id | uint32 | NOT NULL, DEFAULT 0 | Commodity catalog entry ID |
| quantity | uint32 | NOT NULL | Item quantity |
| flag | uint16 | NOT NULL | Item flags |
| purchased_by | uint32 | NOT NULL | Character that purchased the item |
| expiration | timestamp | NOT NULL | Item expiration time (zero means permanent) |
| created_at | timestamp | NOT NULL | Creation timestamp |
| deleted_at | timestamp | INDEX, NULLABLE | Soft-delete timestamp |

---

## Relationships

```
accounts (wallet)
    |
    +-- cash_compartments (1:N via account_id)
            |
            +-- cash_assets (1:N via compartment_id)

wishlist_items (standalone, linked to character_id)
```

- One `accounts` (wallet) entry has many `cash_compartments`
- One `cash_compartments` entry has many `cash_assets`
- `cash_assets` contains all item data directly (flattened; no separate items table)
- `wishlist_items` are linked to characters (external)

---

## Indexes

GORM auto-migration creates:
- Primary key index on `accounts.id`
- Primary key index on `wishlist_items.id`
- Primary key index on `cash_compartments.id`
- Primary key index on `cash_assets.id`
- Soft-delete index on `cash_assets.deleted_at`

---

## Migration Rules

- Migrations are executed via GORM AutoMigrate
- Registered migrations: wallet, wishlist, compartment, asset
- Schema changes are applied automatically on service start
