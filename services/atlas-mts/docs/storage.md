## Tables

### listings (`listing/entity.go`)

Primary key: `id` (uuid). Every row also carries `tenant_id`.

| Column | Type | Notes |
|---|---|---|
| id | uuid | primary key |
| tenant_id | uuid | not null |
| world_id | byte | not null |
| serial | uint32 | not null |
| seller_id | uint32 | not null |
| seller_account_id | uint32 | not null |
| seller_name | string | not null |
| sale_type | string | not null (`fixed`/`auction`/`offer`) |
| state | string | not null (`active`/`settling`/`sold`/`cancelled`/`expired`) |
| template_id | uint32 | not null |
| quantity | uint32 | not null |
| strength, dexterity, intelligence, luck, hp, mp, weapon_attack, magic_attack, weapon_defense, magic_defense, accuracy, avoidability, hands, speed, jump, slots | uint16 | not null; the equip stat-block snapshot |
| level, item_level | byte | not null |
| item_exp, ring_id, vicious_count | uint32 | not null |
| flags | uint16 | not null |
| list_value | uint32 | not null |
| buy_now_price | *uint32 | nullable |
| commission_rate | float64 | not null |
| category, sub_category | string | not null |
| offer_wish_serial, offer_wish_owner_id | uint32 | not null, default 0 |
| ends_at | *time.Time | nullable |
| current_bid, high_bidder_id, min_increment | uint32 | not null |
| bid_count | uint32 | not null, default 0 |
| created_at, updated_at | time.Time | |

### bids (`bid/entity.go`)

Primary key: `id` (uuid).

| Column | Type | Notes |
|---|---|---|
| id | uuid | primary key |
| tenant_id | uuid | not null |
| listing_id | uuid | not null |
| bidder_id | uint32 | not null |
| bidder_account_id | uint32 | not null |
| amount | uint32 | not null (raw base bid) |
| escrow_txn_id | uuid | not null |
| state | string | not null (`held`/`released`/`won`) |
| created_at | time.Time | |

### holdings (`holding/entity.go`)

Primary key: `id` (uuid). GORM soft-delete via `deleted_at`.

| Column | Type | Notes |
|---|---|---|
| id | uuid | primary key |
| tenant_id | uuid | not null |
| world_id | byte | not null |
| serial | uint32 | not null |
| owner_id | uint32 | not null |
| origin | string | not null (`purchased`/`unsold`/`cancelled`/`expired`) |
| template_id | uint32 | not null |
| quantity | uint32 | not null |
| strength, dexterity, intelligence, luck, hp, mp, weapon_attack, magic_attack, weapon_defense, magic_defense, accuracy, avoidability, hands, speed, jump, slots | uint16 | not null; the equip stat-block snapshot |
| level, item_level | byte | not null |
| item_exp, ring_id, vicious_count | uint32 | not null |
| flags | uint16 | not null |
| created_at | time.Time | |
| deleted_at | gorm.DeletedAt | soft-delete column, indexed |

### wish_entries (`wish/entity.go`)

Primary key: `id` (uuid).

| Column | Type | Notes |
|---|---|---|
| id | uuid | primary key |
| tenant_id | uuid | not null |
| world_id | byte | not null |
| serial | uint32 | not null |
| character_id | uint32 | not null |
| item_id | uint32 | not null |
| listing_serial | uint32 | not null, default 0 (set only on `cart` entries) |
| type | string | not null, default `cart` (`cart`/`wanted`) |
| price | uint32 | not null, default 0 |
| count | uint32 | not null, default 1 |
| expires_at | *time.Time | nullable (set only on `wanted` entries) |
| created_at | time.Time | |

### mts_transactions (`transaction/entity.go`)

Primary key: `id` (uuid).

| Column | Type | Notes |
|---|---|---|
| id | uuid | primary key |
| tenant_id | uuid | not null |
| world_id | byte | not null |
| character_id | uint32 | not null |
| counterparty_id | uint32 | not null |
| item_id | uint32 | not null |
| quantity | uint32 | not null |
| total_price | uint32 | not null |
| kind | string | not null (`purchase`/`sale`/`bid_lost`/`cancelled`) |
| created_at | time.Time | auto-create timestamp |

### mts_serials (`serial/entity.go`)

Primary key: composite `(tenant_id, world_id)`.

| Column | Type | Notes |
|---|---|---|
| tenant_id | uuid | primary key (part 1) |
| world_id | byte | primary key (part 2) |
| next_serial | uint32 | not null; the last serial assigned for this `(tenant, world)` |

## Relationships

- `bids.listing_id` addresses a row in `listings.id` (no database foreign
  key constraint is declared; the relationship is enforced in application
  code).
- `listings.serial`, `holdings.serial`, and `wish_entries.serial` are all
  drawn from the same `mts_serials` counter, keyed by `(tenant_id,
  world_id)`; within one world a given serial value maps to at most one
  row across the three tables.
- `mts_transactions.character_id`/`counterparty_id` reference character
  identities owned by other services; atlas-mts stores no character data
  of its own and declares no foreign key for them.

## Indexes

### listings

- `idx_listings_tenant_id` — unique on `(tenant_id, id)`.
- `idx_listings_world_state_category` — on `(tenant_id, world_id, state,
  category)`.
- `idx_listings_seller_state` — on `(tenant_id, seller_id, state)`.
- `idx_listings_world_ends_at` — on `(tenant_id, world_id, ends_at)`.
- `idx_listings_world_serial` — unique on `(tenant_id, world_id, serial)`.

### bids

- `idx_bids_tenant_id` — unique on `(tenant_id, id)`.
- `idx_bids_listing_state` — on `(tenant_id, listing_id, state)`.

### holdings

- `idx_holdings_tenant_id` — unique on `(tenant_id, id)`.
- `idx_holdings_world_owner` — on `(tenant_id, world_id, owner_id)`.
- `idx_holdings_world_serial` — unique on `(tenant_id, world_id, serial)`.
- an index on `deleted_at` (GORM's default soft-delete index).

### wish_entries

- `idx_wish_entries_tenant_id` — unique on `(tenant_id, id)`.
- `idx_wish_entries_character` — on `(tenant_id, character_id)`.
- `idx_wish_entries_world_serial` — unique on `(tenant_id, world_id,
  serial)`.
- `idx_wish_entries_char_item` — unique on `(tenant_id, world_id,
  character_id, item_id, type)`.

### mts_transactions

- `idx_mts_transactions_tenant_id` — unique on `(tenant_id, id)`.
- `idx_mts_transactions_character` — on `(tenant_id, character_id)`.

### mts_serials

- No secondary indexes; the table's primary key is the composite
  `(tenant_id, world_id)`.

## Migration Rules

- Each domain package exposes its own `Migration(db *gorm.DB) error`,
  which calls `db.AutoMigrate(&entity{})`. `listing.Migration`,
  `holding.Migration`, and `wish.Migration` each first call
  `serial.Migration(db)` so the shared `mts_serials` table exists before
  the caller's own table is migrated.
- `main.go` registers every domain migration — `listing.Migration`,
  `holding.Migration`, `bid.Migration`, `wish.Migration`,
  `transaction.Migration` — plus `outboxlib.Migration`, via
  `database.Connect(l, database.SetMigrations(...))`.
- `wish.Migration` additionally drops the pre-existing
  `idx_wish_entries_char_item` index (if present) before calling
  `AutoMigrate`, since `AutoMigrate` does not alter an existing index's
  column set in place; `AutoMigrate` then recreates it with the widened
  `(tenant_id, world_id, character_id, item_id, type)` column set.
- `outboxlib.Migration` creates the transactional-outbox table used by the
  outbox-emission pattern (see `docs/kafka.md`); its schema is owned by
  the shared `atlas-outbox` library and is not restated here.
- All tables shown above are new (no legacy primary-key rewrite), so
  `AutoMigrate` alone produces the correct surrogate-key shape and the
  composite indexes declared on the entity struct tags.
