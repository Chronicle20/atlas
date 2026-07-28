# Gachapon Domain

## Responsibility

The gachapon domain manages gachapon machines, their per-machine item pools, a shared global item pool, and weighted random reward selection.

## Core Models

### gachapon.Model

Immutable representation of a gachapon machine.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | string | Unique gachapon identifier |
| name | string | Display name |
| npcIds | []uint32 | Associated NPC identifiers |
| commonWeight | uint32 | Weight for common tier selection |
| uncommonWeight | uint32 | Weight for uncommon tier selection |
| rareWeight | uint32 | Weight for rare tier selection |

### item.Model

Immutable representation of a gachapon-specific item.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Auto-incremented identifier |
| gachaponId | string | Parent gachapon identifier |
| itemId | uint32 | Game item identifier |
| quantity | uint32 | Quantity awarded |
| tier | string | Tier classification (common, uncommon, rare) |

### global.Model

Immutable representation of a global gachapon item available to all machines.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Auto-incremented identifier |
| itemId | uint32 | Game item identifier |
| quantity | uint32 | Quantity awarded |
| tier | string | Tier classification (common, uncommon, rare) |

### reward.Model

Immutable representation of a selected reward.

| Field | Type | Description |
|-------|------|-------------|
| itemId | uint32 | Game item identifier |
| quantity | uint32 | Quantity awarded |
| tier | string | Tier classification |
| gachaponId | string | Source gachapon identifier |

### gachapon.GachaponAttributes

Decoded shape of the `attributes` field in a gachapon seed catalog file.

| Field | Type | Description |
|-------|------|-------------|
| name | string | Display name |
| npcIds | []uint32 | Associated NPC identifiers |
| commonWeight | uint32 | Weight for common tier selection |
| uncommonWeight | uint32 | Weight for uncommon tier selection |
| rareWeight | uint32 | Weight for rare tier selection |
| items | []ItemAttrib | Embedded gachapon items (itemId, quantity, tier) consumed by item.Subdomain |

### global.GlobalPoolAttributes

Decoded shape of the `attributes` field in the global gachapon pool seed catalog file.

| Field | Type | Description |
|-------|------|-------------|
| items | []GlobalItemAttrib | Global pool items (itemId, quantity, tier) |

## Invariants

- Gachapon tenantId cannot be nil UUID
- Gachapon id cannot be empty
- Item tenantId cannot be nil UUID
- Item tier must be one of: common, uncommon, rare
- Global item tenantId cannot be nil UUID
- Global item tier must be one of: common, uncommon, rare
- Total tier weight (common + uncommon + rare) must be greater than zero
- Reward selection requires at least one item in the selected tier pool

## Processors

### gachapon.Processor

CRUD operations for gachapon machines.

| Method | Description |
|--------|-------------|
| GetAll | Retrieve all gachapons for tenant |
| GetById | Retrieve gachapon by ID |
| Create | Create a new gachapon |
| Update | Update gachapon name and tier weights |
| Delete | Delete a gachapon |
| Count | Retrieve total gachapon row count for tenant |

### gachapon.Subdomain

Seed catalog integration for gachapon machines.

| Method | Description |
|--------|-------------|
| Decode | Decode a gachapon catalog file's attributes |
| Build | Construct a gachapon Model from decoded attributes and the catalog entity ID |
| BulkCreate | Persist multiple gachapon models in a single transaction |
| DeleteAllForTenant | Delete all gachapons for the tenant |
| Count | Report current gachapon row count for seed status reporting |

### item.Processor

CRUD operations for gachapon-specific items.

| Method | Description |
|--------|-------------|
| GetByGachaponId | Retrieve all items for a gachapon, unpaged and regardless of tier |
| GetByGachaponIdPaged | Retrieve a page of items for a gachapon |
| GetByGachaponIdAndTier | Retrieve all items for a gachapon filtered by tier, unpaged |
| GetByGachaponIdAndTierPaged | Retrieve a page of items for a gachapon filtered by tier |
| Create | Create a new gachapon item |
| Delete | Delete a gachapon item by ID |
| Count | Retrieve total gachapon item row count for tenant |

### item.Subdomain

Seed catalog integration for gachapon-specific items. Reads the same catalog files as gachapon.Subdomain and extracts the embedded items.

| Method | Description |
|--------|-------------|
| Decode | Decode a gachapon catalog file's attributes (delegates to gachapon.Subdomain.Decode) |
| Build | Construct item Models from the decoded attributes' embedded items |
| BulkCreate | Persist multiple item models in a single transaction |
| DeleteAllForTenant | Delete all gachapon items for the tenant |
| Count | Report current gachapon item row count for seed status reporting |

### global.Processor

CRUD operations for global gachapon items.

| Method | Description |
|--------|-------------|
| GetAll | Retrieve a page of global items for tenant |
| GetByTier | Retrieve all global items for a tier, unpaged |
| GetByTierPaged | Retrieve a page of global items for a tier |
| Create | Create a new global item |
| Delete | Delete a global item by ID |
| Count | Retrieve total global item row count for tenant |

### global.Subdomain

Seed catalog integration for global gachapon items.

| Method | Description |
|--------|-------------|
| Decode | Decode the global pool catalog file's attributes |
| Build | Construct global item Models from the decoded attributes |
| BulkCreate | Persist multiple global item models in a single transaction |
| DeleteAllForTenant | Delete all global items for the tenant |
| Count | Report current global item row count for seed status reporting |

### reward.Processor

Reward selection logic.

| Method | Description |
|--------|-------------|
| SelectReward | Select a random reward from a gachapon using weighted tier selection |
| GetPrizePool | Retrieve the merged prize pool for a gachapon, optionally filtered by tier |
