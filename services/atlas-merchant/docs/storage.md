# Storage

All entities are GORM structs that embed `gorm.Model` and additionally declare an explicit `Id uuid.UUID` tagged `primaryKey`. Both `gorm.Model.ID` and `Id` resolve to column `id`; the explicit uuid `Id` is the effective primary key.

## Tables

### shops

`shop/entity.go`. Migration: `shop.Migration`.

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null, indexed |
| tenant_region | varchar(10) | Not null, default '' |
| tenant_major | uint16 | Not null, default 0 |
| tenant_minor | uint16 | Not null, default 0 |
| character_id | uint32 | Not null, indexed |
| shop_type | byte | Not null |
| state | byte | Not null |
| title | varchar(255) | Not null, default '' |
| world_id | byte | Not null, default 0 |
| channel_id | byte | Not null, default 0 |
| map_id | uint32 | Not null, indexed |
| instance_id | uuid | Not null, default all-zero uuid |
| x | int16 | Not null |
| y | int16 | Not null |
| permit_item_id | uint32 | Not null |
| expires_at | timestamp | Nullable, indexed |
| closed_at | timestamp | Nullable |
| close_reason | byte | Not null, default 0 |
| meso_balance | uint32 | Not null, default 0 |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### listings

`listing/entity.go`. Migration: `listing.Migration`.

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| shop_id | uuid | Not null, indexed |
| item_id | uint32 | Not null, indexed |
| item_type | byte | Not null |
| quantity | uint16 | Not null |
| bundle_size | uint16 | Not null |
| bundles_remaining | uint16 | Not null |
| price_per_bundle | uint32 | Not null |
| item_snapshot | jsonb | asset.AssetData |
| display_order | uint16 | Not null, default 0 |
| version | uint32 | Not null, default 1 |
| listed_at | timestamp | Not null |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### messages

`message/entity.go`. Migration: `message.Migration`.

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| shop_id | uuid | Not null, indexed |
| character_id | uint32 | Not null |
| content | text | Not null |
| sent_at | timestamp | Not null |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### merchant_blacklists

`blacklist/entity.go`. Migration: `blacklist.Migration`.

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null, unique index `idx_merchant_blacklists_tenant_shop_name` |
| shop_id | uuid | Not null, unique index `idx_merchant_blacklists_tenant_shop_name` |
| name | string | Not null, unique index `idx_merchant_blacklists_tenant_shop_name` |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### merchant_visits

`visit/entity.go`. Migration: `visit.Migration`.

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null, unique index `idx_merchant_visits_tenant_shop_name` |
| shop_id | uuid | Not null, unique index `idx_merchant_visits_tenant_shop_name` |
| name | string | Not null, unique index `idx_merchant_visits_tenant_shop_name` |
| count | uint32 | Not null, default 0 |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### listing_search_counts

`searchcount/entity.go`. Migration: `searchcount.Migration`.

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null, unique index `idx_listing_search_counts_tenant_world_item` |
| world_id | byte | Not null, unique index `idx_listing_search_counts_tenant_world_item` |
| item_id | uint32 | Not null, unique index `idx_listing_search_counts_tenant_world_item` |
| count | uint64 | Not null, default 0 |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### frederick_items

`frederick/entity.go`. Migration: `frederick.Migration`.

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| character_id | uint32 | Not null, indexed |
| item_id | uint32 | Not null |
| item_type | byte | Not null |
| quantity | uint16 | Not null |
| item_snapshot | jsonb | asset.AssetData |
| stored_at | timestamp | Not null |
| last_notified | timestamp | Nullable |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### frederick_mesos

`frederick/entity.go`. Migration: `frederick.Migration`.

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| character_id | uint32 | Not null, indexed |
| amount | uint32 | Not null |
| stored_at | timestamp | Not null |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### frederick_notifications

`frederick/notification_entity.go`. Migration: `frederick.Migration`.

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| tenant_region | string | Not null |
| tenant_major | uint16 | Not null |
| tenant_minor | uint16 | Not null |
| character_id | uint32 | Not null, indexed |
| stored_at | timestamp | Not null |
| next_day | uint16 | Not null |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### outbox

Provided by the shared `atlas-outbox` library (`outboxlib.Migration`, `main.go:71`). The transactional outbox table backing the outbox drainer. Its schema is owned by the library, not this service.

## Relationships

- `listings.shop_id` references `shops.id`
- `messages.shop_id` references `shops.id`
- `merchant_blacklists.shop_id` references `shops.id`
- `merchant_visits.shop_id` references `shops.id`
- `frederick_items.character_id`, `frederick_mesos.character_id`, `frederick_notifications.character_id` reference the owning character (external)

Relationships are application-enforced. No foreign key constraints are defined at the database level (GORM `AutoMigrate`).

## Indexes

| Table | Column(s) | Type |
|---|---|---|
| shops | tenant_id | B-tree |
| shops | character_id | B-tree |
| shops | map_id | B-tree |
| shops | expires_at | B-tree |
| listings | shop_id | B-tree |
| listings | item_id | B-tree |
| messages | shop_id | B-tree |
| merchant_blacklists | (tenant_id, shop_id, name) | Unique (`idx_merchant_blacklists_tenant_shop_name`) |
| merchant_visits | (tenant_id, shop_id, name) | Unique (`idx_merchant_visits_tenant_shop_name`) |
| listing_search_counts | (tenant_id, world_id, item_id) | Unique (`idx_listing_search_counts_tenant_world_item`) |
| frederick_items | character_id | B-tree |
| frederick_mesos | character_id | B-tree |
| frederick_notifications | character_id | B-tree |

## Migration Rules

All in-service tables are managed via GORM `AutoMigrate`. Migrations are registered in `main.go:62`:

```
database.SetMigrations(shop.Migration, listing.Migration, message.Migration, frederick.Migration, searchcount.Migration, blacklist.Migration, visit.Migration, outboxlib.Migration)
```

`frederick.Migration` migrates three entities (item, meso, notification). Each migration function calls `db.AutoMigrate` on its entity types. Schema changes are additive only. `merchant_blacklists`, `merchant_visits`, and `listing_search_counts` use a surrogate uuid primary key plus a tenant-scoped composite unique index (tenant-safe multi-tenant key pattern); the other in-service tables use a surrogate uuid primary key with plain secondary indexes only.
</content>
