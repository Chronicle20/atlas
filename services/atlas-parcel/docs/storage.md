# Storage

## Tables

### parcels

Items and mesos held in Duey's custody in transit between sender and recipient.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | NOT NULL, INDEX (with recipient_id, status / sender_id, status / expires_at) | Tenant identifier |
| id | UUID | PRIMARY KEY | Parcel identifier |
| world_id | BYTE | NOT NULL | World identifier |
| sender_id | UINT32 | NOT NULL, INDEX (with tenant_id, status) | Sender character identifier |
| sender_account_id | UINT32 | NOT NULL | Sender account identifier |
| sender_name | STRING | NOT NULL | Sender display name |
| recipient_id | UINT32 | NOT NULL, INDEX (with tenant_id, status) | Recipient character identifier |
| recipient_account_id | UINT32 | NOT NULL | Recipient account identifier |
| recipient_name | STRING | NOT NULL, DEFAULT '' | Recipient display name, populated end-to-end by the send saga so the expiry sweep's return leg can address a returned parcel back to the original sender |
| message | STRING | | Sender's note |
| meso_amount | UINT32 | | Mesos carried |
| fee_paid | UINT32 | | Delivery fee already collected upstream |
| item_id | UINT32 | NULLABLE | nil for a meso-only parcel |
| item_type | BYTE | | Item type discriminator |
| quantity | UINT16 | | Item quantity |
| item_snapshot | JSONB | | Full item stat snapshot taken at send time (AssetData) |
| status | STRING | NOT NULL, INDEX (with tenant_id, recipient_id / sender_id / expires_at) | One of `pending`, `received`, `discarded`, `expired` |
| quick | BOOL | NOT NULL | True for a quick-delivery parcel with no transit delay |
| returned | BOOL | NOT NULL | True for a parcel created by the expiry sweep's return leg |
| created_at | TIMESTAMP | NOT NULL | Creation time |
| receivable_at | TIMESTAMP | NOT NULL | When the parcel becomes claimable (createdAt + 24h, or createdAt for a return leg) |
| expires_at | TIMESTAMP | NOT NULL, INDEX (with tenant_id, status) | When the parcel is swept for expiry (29 days after creation, see domain.md's ExpiryWindow) |
| resolved_at | TIMESTAMP | NULLABLE | Set when the parcel leaves `pending` |
| last_notified | TIMESTAMP | NULLABLE | Set once the arrival notification has been sent |

**item_snapshot (AssetData, JSONB)**

Full item stat snapshot: `expiration`, `createdAt`, `quantity`, `ownerId`, `owner`, `flag`, `rechargeable`, equipment stats (`strength`, `dexterity`, `intelligence`, `luck`, `hp`, `mp`, `weaponAttack`, `magicAttack`, `weaponDefense`, `magicDefense`, `accuracy`, `avoidability`, `hands`, `speed`, `jump`, `slots`, `levelType`, `level`, `experience`, `hammersApplied`, `ringId`, `viciousCount`), `equippedSince`, cash fields (`cashId`, `commodityId`, `purchaseBy`), `petId`.

---

## Relationships

- `parcels` has no foreign-key relationships to other tables. Sender and recipient identity (`sender_id`, `sender_account_id`, `recipient_id`, `recipient_account_id`) are opaque uint32 values, not foreign keys enforced at the storage layer.

---

## Indexes

### parcels
- `idx_parcels_recipient`: (tenant_id, recipient_id, status) — the mailbox read
- `idx_parcels_sender`: (tenant_id, sender_id, status) — the sender's outbound read
- `idx_parcels_sweep`: (tenant_id, status, expires_at) — the expiry sweep's claim-by-update scan

---

## Migration Rules

- Migrations are executed via GORM AutoMigrate on service startup
- Table: `parcels`
