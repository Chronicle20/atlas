# Storage

## Tables

### equipment

Stores equipment instances.

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| item_id | uint32 | NOT NULL, DEFAULT 0 |
| strength | uint16 | NOT NULL, DEFAULT 0 |
| dexterity | uint16 | NOT NULL, DEFAULT 0 |
| intelligence | uint16 | NOT NULL, DEFAULT 0 |
| luck | uint16 | NOT NULL, DEFAULT 0 |
| hp | uint16 | NOT NULL, DEFAULT 0 |
| mp | uint16 | NOT NULL, DEFAULT 0 |
| weapon_attack | uint16 | NOT NULL, DEFAULT 0 |
| magic_attack | uint16 | NOT NULL, DEFAULT 0 |
| weapon_defense | uint16 | NOT NULL, DEFAULT 0 |
| magic_defense | uint16 | NOT NULL, DEFAULT 0 |
| accuracy | uint16 | NOT NULL, DEFAULT 0 |
| avoidability | uint16 | NOT NULL, DEFAULT 0 |
| hands | uint16 | NOT NULL, DEFAULT 0 |
| speed | uint16 | NOT NULL, DEFAULT 0 |
| jump | uint16 | NOT NULL, DEFAULT 0 |
| slots | uint16 | NOT NULL, DEFAULT 0 |
| owner_name | string | NOT NULL, DEFAULT '' |
| locked | bool | NOT NULL, DEFAULT false |
| spikes | bool | NOT NULL, DEFAULT false |
| karma_used | bool | NOT NULL, DEFAULT false |
| cold | bool | NOT NULL, DEFAULT false |
| can_be_traded | bool | NOT NULL, DEFAULT false |
| level_type | byte | NOT NULL, DEFAULT 0 |
| level | byte | NOT NULL, DEFAULT 0 |
| experience | uint32 | NOT NULL, DEFAULT 0 |
| hammers_applied | uint32 | NOT NULL, DEFAULT 0 |
| expiration | timestamp | NOT NULL, DEFAULT 0 |

## Relationships

None.

## Indexes

Primary key index on (id).

## Migration Rules

Migrations are executed via GORM AutoMigrate on service startup.
