# Storage

## Tables

### shops

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
| map_id | uint32 | Not null, indexed |
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

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| shop_id | uuid | Not null, indexed |
| item_id | uint32 | Not null |
| item_type | byte | Not null |
| quantity | uint16 | Not null |
| bundle_size | uint16 | Not null |
| bundles_remaining | uint16 | Not null |
| price_per_bundle | uint32 | Not null |
| item_snapshot | jsonb | Nullable |
| transaction_id | uuid | Nullable |
| display_order | uint16 | Not null, default 0 |
| version | uint32 | Not null, default 1 |
| listed_at | timestamp | Not null |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### messages

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

### frederick_items

| Column | Type | Constraints |
|---|---|---|
| id | uuid | Primary key |
| tenant_id | uuid | Not null |
| character_id | uint32 | Not null, indexed |
| item_id | uint32 | Not null |
| item_type | byte | Not null |
| quantity | uint16 | Not null |
| item_snapshot | jsonb | Nullable |
| stored_at | timestamp | Not null |
| last_notified | timestamp | Nullable |
| created_at | timestamp | GORM managed |
| updated_at | timestamp | GORM managed |
| deleted_at | timestamp | GORM managed (soft delete) |

### frederick_mesos

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

## Relationships

- `listings.shop_id` references `shops.id`
- `messages.shop_id` references `shops.id`
- `frederick_items.character_id` references the owning character (external)
- `frederick_mesos.character_id` references the owning character (external)
- `frederick_notifications.character_id` references the owning character (external)

Relationships are application-enforced. No foreign key constraints are defined at the database level (GORM AutoMigrate).

## Indexes

| Table | Column(s) | Type |
|---|---|---|
| shops | tenant_id | B-tree |
| shops | character_id | B-tree |
| shops | map_id | B-tree |
| shops | expires_at | B-tree |
| listings | shop_id | B-tree |
| messages | shop_id | B-tree |
| frederick_items | character_id | B-tree |
| frederick_mesos | character_id | B-tree |
| frederick_notifications | character_id | B-tree |

## Migration Rules

All tables are managed via GORM `AutoMigrate`. Migrations are registered in `main.go`:

```
database.Connect(l, database.SetMigrations(shop.Migration, listing.Migration, message.Migration, frederick.Migration))
```

Each migration function calls `db.AutoMigrate` on its respective entity types. Schema changes are additive only.
