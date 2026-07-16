# Storage

## Tables

### monster_book_cards

Per-character monster card ownership and level.

| Column | Type | Constraints | Description |
|--------|------|-------------|--------------|
| tenant_id | uuid | PRIMARY KEY, NOT NULL | Tenant identifier |
| character_id | uint32 | PRIMARY KEY, NOT NULL | Owning character |
| card_id | uint32 | PRIMARY KEY, NOT NULL | Card item identifier |
| level | uint8 | NOT NULL | Card level (1-5) |
| is_special | bool | NOT NULL, DEFAULT false, indexed | Whether the card is in the special-card range |
| last_event_id | uuid | nullable | Event id of the last applied CARD_PICKED_UP command |
| first_acquired_at | timestamp | auto-set on create | Timestamp the card was first acquired |
| updated_at | timestamp | auto-set on update | Timestamp of the last update |

### monster_book_collections

Per-character aggregate monster-book stats and cover selection.

| Column | Type | Constraints | Description |
|--------|------|-------------|--------------|
| tenant_id | uuid | PRIMARY KEY, NOT NULL | Tenant identifier |
| character_id | uint32 | PRIMARY KEY, NOT NULL | Owning character |
| cover_card_id | uint32 | NOT NULL, DEFAULT 0 | Selected cover card item id |
| cover_mob_id | uint32 | NOT NULL, DEFAULT 0 | Monster id represented by the cover card |
| book_level | uint16 | NOT NULL, DEFAULT 1 | Monster book level |
| normal_count | uint16 | NOT NULL, DEFAULT 0 | Count of owned normal cards |
| special_count | uint16 | NOT NULL, DEFAULT 0 | Count of owned special cards |
| exp_bonus_percent | uint16 | NOT NULL, DEFAULT 0 | Party EXP bonus percentage |
| last_cover_event_id | uuid | nullable | Event id of the last applied SET_COVER command |
| created_at | timestamp | auto-set on create | Timestamp the row was created |
| updated_at | timestamp | auto-set on update | Timestamp of the last update |

### outbox_entries

Provided by the shared `atlas-outbox` library (`outboxlib.Migration`, `main.go`). The transactional outbox table backing the outbox drainer. Its schema is owned by the library, not this service.

## Relationships

No foreign key constraints are defined. `monster_book_cards` and `monster_book_collections` share the (tenant_id, character_id) key space but are not linked via a database relationship.

## Indexes

- `monster_book_cards`: composite primary key (tenant_id, character_id, card_id); index on is_special
- `monster_book_collections`: composite primary key (tenant_id, character_id)

## Migration Rules

- Migrations are executed via GORM AutoMigrate
- `card.Migration`, `collection.Migration`, and `outboxlib.Migration` are registered at service startup via `database.Connect` (`main.go`)
