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
| currency | uint32 | NOT NULL, DEFAULT 0 | Wallet bucket this asset was purchased with (1=credit/NX, 2=Maple Points, other=prepaid; 0 means legacy row predating this column, or an asset never bought with currency) |
| quantity | uint32 | NOT NULL | Item quantity |
| flag | uint16 | NOT NULL | Item flags |
| pet_id | uint32 | NOT NULL, DEFAULT 0 | Associated pet ID (0 if the asset is not a pet) |
| purchased_by | uint32 | NOT NULL | Character that purchased the item |
| expiration | timestamp | NOT NULL | Item expiration time (zero means permanent) |
| created_at | timestamp | NOT NULL | Creation timestamp |
| gift_from | varchar(13) | NOT NULL, DEFAULT '' | Sender's character name for a GIFT purchase; empty for every other asset |
| gift_message | varchar(73) | NOT NULL, DEFAULT '' | Sender's message for a GIFT purchase; empty for every other asset |
| gift_acknowledged | boolean | NOT NULL, DEFAULT false | Whether the gift list carrying this asset has been presented to the recipient (LOAD_GIFT_SUCCESS announced) |
| gift_note_sent | boolean | NOT NULL, DEFAULT false | Whether the gift-forward note for this asset has already been sent |
| deleted_at | timestamp | INDEX, NULLABLE | Soft-delete timestamp |

### cash_rings

Stores one row per half of a ring pair (couple/friendship rings).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Unique identifier |
| tenant_id | uuid | NOT NULL, INDEX | Tenant identifier |
| pair_id | uuid | NOT NULL, INDEX | Shared identifier linking both halves of a pair |
| character_id | uint32 | NOT NULL, INDEX | Owner of this half |
| partner_character_id | uint32 | NOT NULL | Owner of the sibling half |
| asset_id | uint32 | NOT NULL | Locker asset backing this half |
| cash_id | int64 | NOT NULL, DEFAULT 0 | This half's asset's cash id, captured at purchase time (0 for rows predating this column) |
| item_template_id | uint32 | NOT NULL | Item template ID |
| ring_type | string | NOT NULL | `COUPLE` or `FRIENDSHIP` |
| state | string | NOT NULL | `ACTIVE`, `BROKEN`, or `EXPIRED` |
| created_at | timestamp | NOT NULL | Creation timestamp |

### cash_purchase_records

Stores the durable, non-soft-deletable answer to "has this account ever bought serial X".

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Unique identifier |
| tenant_id | uuid | NOT NULL, UNIQUE (with account_id, serial_number) | Tenant identifier |
| account_id | uint32 | NOT NULL, UNIQUE (with tenant_id, serial_number) | Associated account |
| serial_number | uint32 | NOT NULL, UNIQUE (with tenant_id, account_id) | Commodity serial number purchased |
| count | uint32 | NOT NULL | Number of times purchased |
| first_at | timestamp | NOT NULL | Timestamp of the first purchase |
| last_at | timestamp | NOT NULL | Timestamp of the most recent purchase |

### coupons

Stores coupon definitions.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Unique identifier |
| tenant_id | uuid | NOT NULL, INDEX, UNIQUE (with code), INDEX (with batch_id) | Tenant identifier |
| batch_id | uuid | NULLABLE, INDEX (with tenant_id) | Owning generation batch, if bulk-generated |
| code | varchar(32) | NOT NULL, UNIQUE (with tenant_id) | Redemption code, stored normalized (trimmed, uppercased) |
| description | text | | Admin description |
| active | boolean | NOT NULL | Whether the coupon is currently redeemable |
| starts_at | timestamp | NULLABLE | Earliest redemption time |
| expires_at | timestamp | NULLABLE | Latest redemption time |
| max_uses | uint32 | NULLABLE | Maximum total redemptions (unlimited if null) |
| redemption_count | uint32 | NOT NULL, DEFAULT 0 | Current redemption count |
| rewards | jsonb | NOT NULL | Reward bundle (currency and/or cash item rewards) |
| created_at | timestamp | | Creation timestamp |
| updated_at | timestamp | | Last update timestamp |

### coupon_batches

Stores one row per bulk coupon-generation request.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Unique identifier |
| tenant_id | uuid | NOT NULL, INDEX | Tenant identifier |
| description | text | | Admin description |
| requested_count | uint32 | NOT NULL | Number of coupons requested |
| generated_count | uint32 | NOT NULL | Number of coupons actually generated (always equals requested_count on a successful batch) |
| created_at | timestamp | | Creation timestamp |

### coupon_redemptions

Stores one row per successful coupon redemption (audit trail).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Unique identifier |
| tenant_id | uuid | NOT NULL, UNIQUE (with coupon_id, account_id), INDEX (with account_id) | Tenant identifier |
| coupon_id | uuid | NOT NULL, INDEX, UNIQUE (with tenant_id, account_id) | Redeemed coupon |
| account_id | uint32 | NOT NULL, UNIQUE (with tenant_id, coupon_id), INDEX (with tenant_id) | Redeeming account |
| character_id | uint32 | NOT NULL | Redeeming character |
| transaction_id | uuid | NOT NULL | Correlation id of the redemption command |
| rewards_granted | jsonb | NOT NULL | Snapshot of the reward bundle granted at redemption time |
| redeemed_at | timestamp | NOT NULL | Redemption timestamp |

### cash_surprise_openings

Stores one row per successfully committed Cash Shop Surprise box open (idempotency ledger; see docs/domain.md's Surprise section).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | uuid | PRIMARY KEY (with transaction_id) | Tenant identifier |
| transaction_id | uuid | PRIMARY KEY (with tenant_id) | Idempotency key minted by atlas-channel per click |
| account_id | uint32 | NOT NULL | Opening account |
| asset_id | uint32 | NOT NULL | Granted reward asset |
| created_at | timestamp | NOT NULL | Open timestamp |

### idempotency_keys

Shared `atlas-database` library table used by `ledger.Claim` to claim per-transaction idempotency for gift, package, ring, equip-slot, and locker-rebate purchase commands.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | uuid | PRIMARY KEY (with key) | Tenant identifier |
| key | string | PRIMARY KEY (with tenant_id) | Claimed idempotency key (the command's transaction id, as a string) |
| operation | string | NOT NULL | Command type the key was claimed for |
| created_at | timestamp | NOT NULL, INDEX | Claim timestamp |

### outbox_entries

Transactional outbox table (shared `atlas-outbox` library schema) used to atomically enqueue Kafka messages with the database write that produced them, for asynchronous draining to Kafka.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uint64 | PRIMARY KEY | Unique identifier |
| topic | string | NOT NULL, INDEX (partial, where sent_at IS NULL) | Destination Kafka topic |
| message_key | []byte | NOT NULL | Kafka message key |
| message_value | []byte | | Kafka message value |
| headers | JSON | NOT NULL, DEFAULT '{}' | Kafka message headers |
| enqueued_at | timestamp | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Time the entry was enqueued |
| sent_at | timestamp | NULLABLE, INDEX (partial, where sent_at IS NOT NULL) | Time the entry was published to Kafka |
| attempts | int | NOT NULL, DEFAULT 0 | Publish attempt count |
| last_error | string | NULLABLE | Last publish error, if any |

---

## Relationships

```
accounts (wallet)
    |
    +-- cash_compartments (1:N via account_id)
            |
            +-- cash_assets (1:N via compartment_id)

wishlist_items (standalone, linked to character_id)

coupon_batches
    |
    +-- coupons (1:N via batch_id, nullable)
            |
            +-- coupon_redemptions (1:N via coupon_id)

cash_rings (standalone; two rows share a pair_id, no foreign key between them)

cash_purchase_records (standalone, linked to account_id / serial_number)

cash_surprise_openings (standalone, linked to account_id)

idempotency_keys (standalone, no foreign key to other tables)

outbox_entries (standalone, no foreign key to other tables)
```

- One `accounts` (wallet) entry has many `cash_compartments`
- One `cash_compartments` entry has many `cash_assets`
- `cash_assets` contains all item data directly (flattened; no separate items table)
- `wishlist_items` are linked to characters (external)
- One `coupon_batches` entry has many `coupons` (via nullable `batch_id`); a coupon created outside a batch has a null `batch_id`
- One `coupons` entry has many `coupon_redemptions`
- `cash_rings` holds two rows per pair (`pair_id`), inserted together in one statement; no database-level foreign key links the two halves
- `cash_purchase_records` and `cash_surprise_openings` hold no foreign key to other tables in this schema
- `idempotency_keys` and `outbox_entries` hold no foreign key to any other table in this schema

---

## Indexes

GORM auto-migration creates:
- Primary key index on `accounts.id`
- Primary key index on `wishlist_items.id`
- Primary key index on `cash_compartments.id`
- Primary key index on `cash_assets.id`
- Soft-delete index on `cash_assets.deleted_at`
- Primary key index on `cash_rings.id`; index on `cash_rings.tenant_id`; index on `cash_rings.pair_id`; index on `cash_rings.character_id`
- Primary key index on `cash_purchase_records.id`; unique index on `(cash_purchase_records.tenant_id, account_id, serial_number)`
- Primary key index on `coupons.id`; index on `coupons.tenant_id`; unique index on `(coupons.tenant_id, code)`; index on `(coupons.tenant_id, batch_id)`
- Primary key index on `coupon_batches.id`; index on `coupon_batches.tenant_id`
- Primary key index on `coupon_redemptions.id`; unique index on `(coupon_redemptions.tenant_id, coupon_id, account_id)`; index on `coupon_redemptions.coupon_id`; index on `(coupon_redemptions.tenant_id, account_id)`
- Composite primary key on `(cash_surprise_openings.tenant_id, transaction_id)`
- Composite primary key on `(idempotency_keys.tenant_id, key)`; index on `idempotency_keys.created_at`
- Primary key index on `outbox_entries.id`
- Partial index on `outbox_entries.topic` where `sent_at IS NULL`
- Partial index on `outbox_entries.sent_at` where `sent_at IS NOT NULL`

---

## Migration Rules

- Migrations are executed via GORM AutoMigrate
- Registered migrations: wallet, wishlist, compartment, asset, opening (Surprise), purchaserecord, ring, coupon, batch, redemption, outbox (`atlas-outbox` library), idempotency (`atlas-database` library)
- Schema changes are applied automatically on service start
- `purchaserecord.Backfill` runs once on every boot (idempotent no-op once `cash_purchase_records` has any row) to seed purchase history from existing `cash_assets` rows, recovering history for accounts that bought before `cash_purchase_records` existed
