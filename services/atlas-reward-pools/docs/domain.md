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
| kind | string | Pool kind: `gachapon` (default, tiered), `incubator`, or `cash-surprise` (both flat-weighted) — see Kind union below |

#### Kind union

`gachapon.Model.Kind()` is a closed union of three values:

| Kind | Selection | Notes |
|------|-----------|-------|
| `gachapon` (default) | Tiered: `selectTier` rolls common/uncommon/rare by `commonWeight`/`uncommonWeight`/`rareWeight`, then `getMergedPool` merges the machine's tier items with the shared global pool for that tier. | Behaviorally unchanged by the `incubator`/`cash-surprise` kinds. |
| `incubator` | Flat: rolls the whole machine's item set (all tiers) weighted by `item.Weight`, never merging the global pool. `reward.Model.Tier()` is always `""`. | |
| `cash-surprise` | Same flat path as `incubator`. A `cash-surprise` pool's `id` is the cash shop box **template id** (e.g. `5222000`), exactly as an `incubator` pool's `id` is the egg item id. Every item entry must carry a non-zero `commodityId` (enforced at `item.Builder.Build`); `reward.Model.CommodityId()` carries the selected entry's commodity through to the response. | |

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
| weight | uint32 | Optional explicit roll weight, consumed by flat-weighted (`incubator`/`cash-surprise`) pools; ignored by the tiered `gachapon` roll |
| commodityId | uint32 | Cash shop commodity (serial number) this entry grants. Required (non-zero) for `cash-surprise` entries; `0` for `gachapon`/`incubator` entries, which identify the reward by `itemId` instead |

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
| tier | string | Tier classification (empty string for flat-weighted `incubator`/`cash-surprise` rewards) |
| weight | uint32 | Roll weight of the selected entry (populated by `GetPrizePool`; `0` on a `SelectReward` result) |
| gachaponId | string | Source gachapon identifier |
| commodityId | uint32 | Cash shop commodity (serial number) this reward grants. Non-zero only for `cash-surprise` pools |

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
- Gachapon kind must be one of: gachapon, incubator, cash-surprise
- Item tenantId cannot be nil UUID
- Item tier must be one of: common, uncommon, rare
- Global item tenantId cannot be nil UUID
- Global item tier must be one of: common, uncommon, rare
- Total tier weight (common + uncommon + rare) must be greater than zero (only enforced for `gachapon`-kind pools; `incubator`/`cash-surprise` never invoke `selectTier`)
- Reward selection requires at least one eligible entry in the pool, or `SelectReward` returns `reward.ErrEmptyPool`
- A `cash-surprise` item entry must carry a non-zero `commodityId`, or `item.Builder.Build` returns `item.ErrCommodityIdRequired`

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
| Update | Update an item's itemId, quantity, tier, weight, and commodityId; the owning gachapon is never re-parented |
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
| Update | Update a global item's itemId, quantity, and tier |
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
| SelectReward | Select a random reward from a gachapon. `gachapon`-kind pools use tiered selection (`selectTier` + `getMergedPool`, including the shared global pool); `incubator`/`cash-surprise`-kind pools roll the whole pool flat-weighted by `item.Weight`, excluding the global pool. Returns `reward.ErrEmptyPool` when the pool has no eligible entries. |
| GetPrizePool | Retrieve the merged prize pool for a gachapon, optionally filtered by tier. Follows the same kind-based branch as SelectReward. |
