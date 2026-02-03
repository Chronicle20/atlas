# Domain

## character

### Responsibility

Orchestrates expiration checks across character inventory, account storage, and cash shop for a given character session.

### Core Models

No local domain models. Consumes REST models from external services:
- `inventory.RestModel`, `inventory.AssetRestModel`
- `storage.RestModel`, `storage.AssetRestModel`
- `cashshop.InventoryRestModel`, `cashshop.ItemRestModel`

### Invariants

None. Service performs read-only checks and emits commands.

### Processors

#### CheckAndExpire

Checks all items for a character across inventory, storage, and cash shop. Emits expire commands for items past their expiration time.

Parameters: `characterId`, `accountId`, `worldId`

---

## expiration

### Responsibility

Provides pure utility functions for expiration time checks.

### Core Models

None.

### Invariants

- Zero time value means no expiration (item never expires)
- Item is expired if current time is after expiration time

### Processors

#### IsExpired

Returns true if expiration time is set and current time is after expiration.

#### HasExpiration

Returns true if expiration time is not zero.

---

## session

### Responsibility

Tracks online character sessions for periodic expiration checks.

### Core Models

#### Session

| Field | Type |
|-------|------|
| CharacterId | uint32 |
| AccountId | uint32 |
| WorldId | byte |
| ChannelId | byte |
| TenantId | uuid.UUID |
| Region | string |
| MajorVersion | uint16 |
| MinorVersion | uint16 |

### Invariants

- Sessions are keyed by CharacterId
- Tracker is a singleton

### Processors

#### Tracker.Add

Adds or updates a session in the tracker.

#### Tracker.Remove

Removes a session by character ID.

#### Tracker.GetAll

Returns a snapshot of all tracked sessions.

---

## data

### Responsibility

Retrieves item replacement information from atlas-data based on template ID ranges.

### Core Models

#### ReplaceInfo

| Field | Type |
|-------|------|
| ReplaceItemId | uint32 |
| ReplaceMessage | string |

### Invariants

- Equipment: template IDs 1000000-1999999
- Consumables: template IDs 2000000-2999999
- Setup: template IDs 3000000-3999999
- Etc: template IDs 4000000-4999999
- Cash items and unknown ranges return empty ReplaceInfo

### Processors

#### GetReplaceInfo

Returns replacement item information for a given template ID.
