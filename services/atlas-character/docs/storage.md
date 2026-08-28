# Storage

## Tables

### characters

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| TenantId | uuid | no | - | Tenant identifier |
| ID | uint32 | no | autoIncrement | Primary key |
| AccountId | uint32 | no | - | Associated account |
| World | world.Id | no | - | World assignment |
| Name | string | no | - | Character name |
| Level | byte | no | 1 | Current level |
| Experience | uint32 | no | 0 | Current experience |
| GachaponExperience | uint32 | no | 0 | Gachapon experience |
| Strength | uint16 | no | 12 | STR stat |
| Dexterity | uint16 | no | 5 | DEX stat |
| Intelligence | uint16 | no | 4 | INT stat |
| Luck | uint16 | no | 4 | LUK stat |
| HP | uint16 | no | 50 | Current HP |
| MP | uint16 | no | 5 | Current MP |
| MaxHP | uint16 | no | 50 | Maximum HP |
| MaxMP | uint16 | no | 5 | Maximum MP |
| Meso | uint32 | no | 0 | Currency |
| HPMPUsed | int | no | 0 | AP spent on HP/MP |
| JobId | job.Id | no | 0 | Current job |
| SkinColor | byte | no | 0 | Skin color |
| Gender | byte | no | 0 | Gender |
| Fame | int16 | no | 0 | Fame points |
| Hair | uint32 | no | 0 | Hair style ID |
| Face | uint32 | no | 0 | Face ID |
| AP | uint16 | no | 0 | Available AP |
| SP | string | no | 0,0,0,0,0,0,0,0,0,0 | Available SP |
| SpawnPoint | uint32 | no | 0 | Spawn point ID |
| GM | int | no | 0 | GM level |
| X | int16 | no | 0 | X position |
| Y | int16 | no | 0 | Y position |
| Stance | byte | no | 0 | Current stance |

### session_history

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| ID | uint64 | no | autoIncrement | Primary key |
| TenantId | uuid | no | - | Tenant identifier |
| CharacterId | uint32 | no | - | Character |
| WorldId | world.Id | no | - | World |
| ChannelId | channel.Id | no | - | Channel |
| LoginTime | time.Time | no | - | Login timestamp |
| LogoutTime | *time.Time | yes | null | Logout timestamp |

### saved_locations

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| ID | uuid.UUID | no | - | Primary key |
| TenantId | uuid | no | - | Tenant identifier |
| CharacterId | uint32 | no | - | Character |
| LocationType | string | no | - | Type of saved location |
| MapId | map.Id | no | - | Map |
| PortalId | uint32 | no | - | Portal |

### character_equip_slot_extensions

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| Id | uuid.UUID | no | - | Primary key |
| TenantId | uuid | no | - | Tenant identifier |
| CharacterId | uint32 | no | - | Character |
| SlotIndex | int16 | no | - | Atlas canonical equipped-inventory position |
| ExpiresAt | time.Time | no | - | Expiration timestamp |
| TransactionId | uuid.UUID | no | 00000000-0000-0000-0000-000000000000 | Idempotency key of the last-applied Extend call |
| CreatedAt | time.Time | no | - | Row creation timestamp |
| UpdatedAt | time.Time | no | - | Row last-update timestamp |

### teleport_rock_maps

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| ID | uuid.UUID | no | - | Primary key |
| TenantId | uuid | no | - | Tenant identifier |
| CharacterId | uint32 | no | - | Character |
| ListType | string | no | - | "regular" or "vip" |
| Slot | int | no | - | 0-based position within the list |
| MapId | map.Id | no | - | Saved map |

### character_pending_changes

| Column | Type | Nullable | Default | Description |
|--------|------|----------|---------|-------------|
| Id | uuid.UUID | no | - | Primary key |
| TenantId | uuid | no | - | Tenant identifier |
| CharacterId | uint32 | no | - | Character |
| Type | string | no | - | NAME_CHANGE or WORLD_TRANSFER |
| Status | string | no | - | PENDING, APPLIED, CANCELLED, REJECTED, or EXPIRED |
| RequestedName | *string | yes | null | Requested name (NAME_CHANGE) |
| RequestedNameLower | *string | yes | null | Lower-cased RequestedName, backs the name-reservation index |
| DestinationWorldId | *world.Id | yes | null | Destination world (WORLD_TRANSFER) |
| SourceWorldId | world.Id | no | - | World the character occupied at request time |
| AssetId | *uint32 | yes | null | Coupon item template ID consumed at acceptance (item purchase path only) |
| Reason | string | no | '' | Rejection/expiry/cancel reason |
| TransactionId | uuid.UUID | no | - | Correlation ID for the originating request |
| CreatedAt | time.Time | no | - | Creation timestamp |
| ExpiresAt | time.Time | no | - | Expiry sweep deadline |
| ResolvedAt | *time.Time | yes | null | Terminal-transition timestamp |
| NotifiedAt | *time.Time | yes | null | Timestamp the resolution notification was last emitted |

### outbox_entries

Provided by the shared `atlas-outbox` library (`outboxlib.Migration`, `main.go`). The transactional outbox table backing the outbox drainer. Its schema is owned by the library, not this service.

## Relationships

None defined in entities.

## Indexes

### characters
- Primary key on ID column (auto-generated by GORM)

### session_history
- Primary key on ID column (auto-generated by GORM)
- Composite index `idx_session_history_lookup` on (TenantId, CharacterId, LoginTime)

### saved_locations
- Primary key on ID column
- Unique composite index `idx_saved_location_lookup` on (TenantId, CharacterId, LocationType)

### character_equip_slot_extensions
- Primary key on Id column
- Unique composite index `idx_equipslot_unique` on (TenantId, CharacterId, SlotIndex)

### teleport_rock_maps
- Primary key on ID column
- Unique composite index `idx_trock_lookup` on (TenantId, CharacterId, ListType, Slot)

### character_pending_changes
- Primary key on Id column
- Composite index `idx_pc_tenant_char` on (TenantId, CharacterId)
- Index `idx_pc_status` on Status
- Partial unique index `idx_pc_one_pending_per_type` on (TenantId, CharacterId, Type) WHERE Status = 'PENDING'
- Partial unique index `idx_pc_name_reservation` on (TenantId, RequestedNameLower) WHERE Status = 'PENDING' AND Type = 'NAME_CHANGE'

## Migration Rules

- Migration performed via GORM AutoMigrate
- Table names: characters, session_history, saved_locations, character_equip_slot_extensions, teleport_rock_maps, character_pending_changes, outbox_entries
- `character.Migration` runs AutoMigrate for `characters`, then explicitly drops the legacy `MapId`/`Instance` columns if present (idempotent); atlas-maps owns character location state
- `pending_change.Migration` runs AutoMigrate for `character_pending_changes`, then creates `idx_pc_one_pending_per_type` and `idx_pc_name_reservation` via raw DDL (`CREATE UNIQUE INDEX IF NOT EXISTS`, idempotent)
- `character.Migration`, `history.Migration`, `saved_location.Migration`, `teleport_rock.Migration`, `pending_change.Migration`, `equipslot.Migration`, and `outboxlib.Migration` are registered together at service startup via `database.Connect` (`main.go`)
- Schema changes applied on service startup
