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

Stored items.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | NOT NULL, INDEX (with storage_id) | Tenant identifier |
| id | UINT32 | PRIMARY KEY, AUTO INCREMENT | Asset identifier |
| storage_id | UUID | NOT NULL, INDEX (with tenant_id) | Parent storage |
| inventory_type | BYTE | NOT NULL, DEFAULT 4 | Inventory category |
| slot | INT16 | NOT NULL | Position in storage |
| template_id | UINT32 | NOT NULL | Item template |
| expiration | TIMESTAMP | NOT NULL | Expiration time |
| reference_id | UINT32 | NOT NULL | Type-specific reference |
| reference_type | VARCHAR(20) | NOT NULL | Reference data type |

### storage_stackables

Stackable item data.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| asset_id | UINT32 | PRIMARY KEY | Parent asset |
| quantity | UINT32 | NOT NULL, DEFAULT 1 | Item count |
| owner_id | UINT32 | NOT NULL, DEFAULT 0 | Owner character |
| flag | UINT16 | NOT NULL, DEFAULT 0 | Item flags |

---

## Relationships

- `storages` 1:N `storage_assets` via storage_id
- `storage_assets` 1:1 `storage_stackables` via asset_id (for stackable types only)

---

## Indexes

### storages
- `idx_tenant_world_account`: UNIQUE (tenant_id, world_id, account_id)

### storage_assets
- `idx_asset_tenant_storage`: (tenant_id, storage_id)

---

## Migration Rules

- Migrations are executed via GORM AutoMigrate
- Schema changes are applied automatically on service startup
- Tables: storages, storage_assets, storage_stackables
