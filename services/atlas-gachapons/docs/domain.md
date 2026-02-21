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

### item.Processor

CRUD operations for gachapon-specific items.

| Method | Description |
|--------|-------------|
| GetByGachaponId | Retrieve all items for a gachapon |
| GetByGachaponIdAndTier | Retrieve items for a gachapon filtered by tier |
| Create | Create a new gachapon item |
| Delete | Delete a gachapon item by ID |

### global.Processor

CRUD operations for global gachapon items.

| Method | Description |
|--------|-------------|
| GetAll | Retrieve all global items for tenant |
| GetByTier | Retrieve global items filtered by tier |
| Create | Create a new global item |
| Delete | Delete a global item by ID |

### reward.Processor

Reward selection logic.

| Method | Description |
|--------|-------------|
| SelectReward | Select a random reward from a gachapon using weighted tier selection |
| GetPrizePool | Retrieve the merged prize pool for a gachapon, optionally filtered by tier |

### seed.Processor

Seed data loading from JSON files.

| Method | Description |
|--------|-------------|
| Seed | Delete existing data and load gachapons, items, and global items from JSON files |
