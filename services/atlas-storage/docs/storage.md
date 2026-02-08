# Storage

## Tables

### storages

Account-level storage containers.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | NOT NULL, UNIQUE INDEX (with world_id, account_id) | Tenant identifier |
| id | UUID | PRIMARY KEY | Storage identifier |
| world_id | BYTE | NOT NULL, UNIQUE INDEX (with tenant_id, account_id) | World identifier |
| account_id | UINT32 | NOT NULL, UNIQUE INDEX (with tenant_id, world_id) | Account identifier |
| capacity | UINT32 | NOT NULL, DEFAULT 4 | Maximum asset count |
| mesos | UINT32 | NOT NULL, DEFAULT 0 | Stored currency |

### storage_assets

Stored items. All item type fields are stored inline in this single table. The relevant fields depend on the item type (determined by template_id): equipment items use the stat fields, stackable items use quantity/owner_id/flag, cash items use cash_id/commodity_id/purchase_by, and pet items use pet_id.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | NOT NULL, INDEX (with storage_id) | Tenant identifier |
| id | UINT32 | PRIMARY KEY, AUTO INCREMENT | Asset identifier |
| storage_id | UUID | NOT NULL, INDEX (with tenant_id) | Parent storage |
| inventory_type | BYTE | NOT NULL, DEFAULT 4 | Inventory category (derived from template_id) |
| slot | INT16 | NOT NULL | Position in storage |
| template_id | UINT32 | NOT NULL | Item template |
| expiration | TIMESTAMP | NOT NULL | Expiration time |
| deleted_at | TIMESTAMP | INDEX, NULLABLE | Soft delete timestamp |
| quantity | UINT32 | | Item count (stackable/cash items) |
| owner_id | UINT32 | | Owner character (stackable items) |
| flag | UINT16 | | Item flags (stackable items) |
| rechargeable | UINT64 | | Rechargeable amount (consumable items) |
| strength | UINT16 | | STR stat (equipment) |
| dexterity | UINT16 | | DEX stat (equipment) |
| intelligence | UINT16 | | INT stat (equipment) |
| luck | UINT16 | | LUK stat (equipment) |
| hp | UINT16 | | HP stat (equipment) |
| mp | UINT16 | | MP stat (equipment) |
| weapon_attack | UINT16 | | WATK stat (equipment) |
| magic_attack | UINT16 | | MATK stat (equipment) |
| weapon_defense | UINT16 | | WDEF stat (equipment) |
| magic_defense | UINT16 | | MDEF stat (equipment) |
| accuracy | UINT16 | | ACC stat (equipment) |
| avoidability | UINT16 | | AVOID stat (equipment) |
| hands | UINT16 | | Hands stat (equipment) |
| speed | UINT16 | | Speed stat (equipment) |
| jump | UINT16 | | Jump stat (equipment) |
| slots | UINT16 | | Upgrade slots remaining (equipment) |
| locked | BOOL | | Lock status (equipment) |
| spikes | BOOL | | Spikes flag (equipment) |
| karma_used | BOOL | | Karma scroll used (equipment) |
| cold | BOOL | | Cold protection (equipment) |
| can_be_traded | BOOL | | Trade eligibility (equipment) |
| level_type | BYTE | | Level type (equipment) |
| level | BYTE | | Item level (equipment) |
| experience | UINT32 | | Item experience (equipment) |
| hammers_applied | UINT32 | | Vicious hammer count (equipment) |
| cash_id | INT64 | | Cash shop serial number (cash items) |
| commodity_id | UINT32 | | Commodity identifier (cash items) |
| purchase_by | UINT32 | | Purchaser character (cash items) |
| pet_id | UINT32 | | Pet identifier (pet items) |

---

## Relationships

- `storages` 1:N `storage_assets` via storage_id

---

## Indexes

### storages
- `idx_tenant_world_account`: UNIQUE (tenant_id, world_id, account_id)

### storage_assets
- `idx_asset_tenant_storage`: (tenant_id, storage_id)
- `idx_storage_assets_deleted_at`: (deleted_at) -- soft delete index

---

## Migration Rules

- Migrations are executed via GORM AutoMigrate
- Schema changes are applied automatically on service startup
- Tables: storages, storage_assets
- Soft deletes are enabled on storage_assets via GORM's DeletedAt field
