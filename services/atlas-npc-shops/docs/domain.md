# Domain

## Shop

### Responsibility

Represents an NPC shop that sells commodities to characters. Manages character entry/exit tracking, buy/sell/recharge operations, and rechargeable consumable decoration.

### Core Models

#### Shop Model

| Field       | Type               | Description                                     |
|-------------|--------------------|-------------------------------------------------|
| npcId       | uint32             | NPC template identifier                         |
| commodities | []Commodity        | List of commodities sold by this shop           |
| recharger   | bool               | Whether the shop supports recharging throwables |

#### Commodity Model

| Field           | Type      | Description                                      |
|-----------------|-----------|--------------------------------------------------|
| id              | uuid.UUID | Unique commodity identifier                      |
| npcId           | uint32    | NPC template identifier                          |
| templateId      | uint32    | Item template identifier                         |
| mesoPrice       | uint32    | Price in mesos                                   |
| discountRate    | byte      | Discount percentage (0-100)                      |
| tokenTemplateId | uint32    | Alternative currency item identifier             |
| tokenPrice      | uint32    | Price in alternative currency                    |
| period          | uint32    | Time limit on purchase in minutes (0=unlimited)  |
| levelLimit      | uint32    | Minimum level required to purchase (0=no limit)  |
| unitPrice       | float64   | Unit price for rechargeable items                |
| slotMax         | uint32    | Maximum stack size for the item                  |

#### Asset Model (read-only, fetched from atlas-inventory)

A unified model representing any inventory item regardless of type. The inventory type is derived from the templateId at runtime.

| Field           | Type       | Description                                        |
|-----------------|------------|----------------------------------------------------|
| id              | uint32     | Unique asset identifier                            |
| compartmentId   | uuid.UUID  | Parent compartment identifier                      |
| slot            | int16      | Inventory slot position                            |
| templateId      | uint32     | Item template identifier                           |
| expiration      | time.Time  | Expiration timestamp                               |
| createdAt       | time.Time  | Creation timestamp                                 |
| quantity        | uint32     | Stack quantity (stackable/cash items; 1 for equips)|
| ownerId         | uint32     | Owner character identifier                         |
| flag            | uint16     | Item flags                                         |
| rechargeable    | uint64     | Rechargeable data                                  |
| strength        | uint16     | Equipment stat: STR                                |
| dexterity       | uint16     | Equipment stat: DEX                                |
| intelligence    | uint16     | Equipment stat: INT                                |
| luck            | uint16     | Equipment stat: LUK                                |
| hp              | uint16     | Equipment stat: HP                                 |
| mp              | uint16     | Equipment stat: MP                                 |
| weaponAttack    | uint16     | Equipment stat: weapon attack                      |
| magicAttack     | uint16     | Equipment stat: magic attack                       |
| weaponDefense   | uint16     | Equipment stat: weapon defense                     |
| magicDefense    | uint16     | Equipment stat: magic defense                      |
| accuracy        | uint16     | Equipment stat: accuracy                           |
| avoidability    | uint16     | Equipment stat: avoidability                       |
| hands           | uint16     | Equipment stat: hands                              |
| speed           | uint16     | Equipment stat: speed                              |
| jump            | uint16     | Equipment stat: jump                               |
| slots           | uint16     | Equipment upgrade slots remaining                  |
| locked          | bool       | Whether the item is locked                         |
| spikes          | bool       | Whether the item has spikes                        |
| karmaUsed       | bool       | Whether karma has been used on this item           |
| cold            | bool       | Whether the item provides cold protection          |
| canBeTraded     | bool       | Whether the item is tradeable                      |
| levelType       | byte       | Equipment level type                               |
| level           | byte       | Equipment level                                    |
| experience      | uint32     | Equipment experience                               |
| hammersApplied  | uint32     | Number of vicious hammers applied                  |
| equippedSince   | *time.Time | Timestamp when the item was equipped               |
| cashId          | int64      | Cash item serial number                            |
| commodityId     | uint32     | Cash shop commodity identifier                     |
| purchaseBy      | uint32     | Character who purchased the cash item              |
| petId           | uint32     | Pet identifier (for pet cash items)                |

The asset model provides type classification helpers:

- `InventoryType()` derives the inventory type from the templateId
- `IsEquipment()`, `IsConsumable()`, `IsSetup()`, `IsEtc()`, `IsCash()` check against inventory type constants
- `IsPet()` returns true for cash items with a non-zero petId
- `IsStackable()` returns true for consumable, setup, and etc types
- `HasQuantity()` returns true for stackable items and non-pet cash items
- `Quantity()` returns the stored quantity for items with quantity, otherwise returns 1

#### Compartment Model (read-only, fetched from atlas-inventory)

| Field         | Type            | Description                                 |
|---------------|-----------------|---------------------------------------------|
| id            | uuid.UUID       | Unique compartment identifier               |
| characterId   | uint32          | Owner character identifier                  |
| inventoryType | inventory.Type  | Inventory type (equip, use, setup, etc, cash)|
| capacity      | uint32          | Maximum number of slots                     |
| assets        | []Asset         | Assets contained in this compartment        |

The compartment model provides slot management helpers:

- `NextFreeSlot()` finds the lowest available slot, returns error when full
- `FindBySlot(slot)` locates an asset by its slot position
- `FindFirstByItemId(templateId)` locates the first asset matching a template ID

#### Inventory Model (read-only, fetched from atlas-inventory)

| Field        | Type                            | Description                     |
|--------------|---------------------------------|---------------------------------|
| characterId  | uint32                          | Owner character identifier      |
| compartments | map[inventory.Type]Compartment  | Compartments keyed by type      |

Provides typed accessors: `Equipable()`, `Consumable()`, `Setup()`, `ETC()`, `Cash()`, and `CompartmentByType(type)`.

#### Character Model (read-only, fetched from atlas-character)

| Field              | Type            | Description                                     |
|--------------------|-----------------|--------------------------------------------------|
| id                 | uint32          | Character identifier                              |
| accountId          | uint32          | Owning account identifier                          |
| worldId            | world.Id        | World identifier                                   |
| name               | string          | Character name                                     |
| gender             | byte            | Character gender                                   |
| skinColor          | byte            | Character skin color                               |
| face               | uint32          | Face identifier                                    |
| hair               | uint32          | Hair identifier                                    |
| level              | byte            | Character level                                    |
| jobId              | job.Id          | Job identifier                                     |
| strength           | uint16          | STR stat                                           |
| dexterity          | uint16          | DEX stat                                           |
| intelligence       | uint16          | INT stat                                           |
| luck               | uint16          | LUK stat                                           |
| hp                 | uint16          | Current HP                                         |
| maxHp              | uint16          | Maximum HP                                         |
| mp                 | uint16          | Current MP                                         |
| maxMp              | uint16          | Maximum MP                                         |
| hpMpUsed           | int             | HP/MP AP allocation used                           |
| ap                 | uint16          | Unassigned ability points                          |
| sp                 | string          | Comma-separated skill point pools                  |
| experience         | uint32          | Character experience                               |
| fame               | int16           | Fame value                                         |
| gachaponExperience | uint32          | Gachapon experience                                |
| spawnPoint         | uint32          | Spawn point identifier                             |
| gm                 | int             | GM flag (1 = GM)                                   |
| x                  | int16           | X position                                         |
| y                  | int16           | Y position                                         |
| stance             | byte            | Stance                                             |
| meso               | uint32          | Meso balance                                       |
| inventory          | inventory.Model | Character's inventory (populated by InventoryDecorator) |
| skills             | []skill.Model   | Character's skills (populated by callers of the skill processor) |

The character model provides derived helpers: `Gm()` (gm == 1), `HasSPTable()` (true for Evan job stages), `Sp()` (parses the sp string into a slice), `RemainingSp()` (indexes Sp() by job-derived skill book). `Rank()`, `RankMove()`, `JobRank()`, `JobRankMove()`, and `SpawnPoint()` (the byte-returning variant) are unimplemented and always return 0.

#### Skill Model (read-only, fetched from atlas-skill)

| Field             | Type      | Description                          |
|-------------------|-----------|---------------------------------------|
| id                | skill.Id  | Skill identifier                      |
| level              | byte      | Skill level                           |
| masterLevel        | byte      | Skill master level                    |
| expiration         | time.Time | Skill expiration timestamp            |
| cooldownExpiresAt  | time.Time | Timestamp when the skill's cooldown expires |

The skill model provides `IsFourthJob()` (true when the skill's owning job is a fourth job) and `OnCooldown()` (true when the current time is before cooldownExpiresAt). The package-level `GetLevel(skills, id)` helper scans a skill slice for a matching id and returns its level, or 0 if absent.

### Invariants

- Shop npcId must be non-zero
- Commodity id must be non-nil
- Commodity templateId must be non-zero
- A character can only be in one shop at a time (registry enforces this)
- Buy operations require the character to be in a shop, the commodity to exist, sufficient mesos, and a free inventory slot
- Sell operations require the character to be in a shop, the item to exist in the specified slot with matching templateId, and sufficient quantity
- Recharge operations require the shop to be a recharger, the item to be a consumable in the Use inventory, and sufficient mesos

### State Transitions

#### Shop Entry/Exit

- `Enter`: Validates shop exists, registers character in the in-memory registry, emits ENTERED event
- `Exit`: Removes character from the registry, emits EXITED event if the character was in a shop
- Character logout, map change, or channel change triggers automatic exit

### Processors

#### Shop Processor

- Retrieves shops by NPC ID with optional decorators
- Retrieves all shops for a tenant
- Creates shops with commodities in a single operation
- Updates shops with commodities within a database transaction (delete-then-recreate pattern)
- Manages shop entry and exit for characters via an in-memory registry
- Processes buy operations: validates shop membership, commodity existence, meso balance, and inventory capacity; emits meso change and create-asset commands. Rechargeable items purchase a full stack (slotMax quantity) priced by the commodity's mesoPrice if set, otherwise by unitPrice × slotMax. Token-priced commodities (mesoPrice == 0 on a non-rechargeable item) are not implemented and emit a GENERIC_ERROR_WITH_REASON status event.
- Processes sell operations: validates shop membership, item ownership, and quantity; looks up item price from the data service by item type (equipable, consumable, setup, etc); emits meso change and destroy commands
- Processes recharge operations: validates recharger flag, item existence in the consumable compartment, skill-based slot max bonuses (Claw Mastery from NightWalker/Assassin skill lines for throwing stars, Gun Mastery for bullets), and meso balance; emits meso change and recharge commands
- Decorates shops via RechargeableConsumablesDecorator: refreshes the slotMax/unitPrice of any existing commodity that matches a cached rechargeable consumable (all shops); additionally, for shops with recharger=true, auto-adds any rechargeable consumable not already present as a commodity. Rechargeable commodities are always sorted to the end of the commodity list (by templateId) so a rechargeable never occupies the first slot.
- Tracks characters currently in shops via a Redis-backed Registry singleton
- Returns the tenant's shop count and most recent updated_at timestamp (Count)

#### Commodity Processor

- Retrieves commodities by NPC ID
- Retrieves all commodities for a tenant
- Creates, updates, and deletes individual commodities
- Bulk-deletes commodities by NPC ID or all commodities for a tenant
- Retrieves distinct NPC IDs and commodity-ID-to-NPC-ID maps
- Decorates commodities with item data (unitPrice, slotMax) based on inventory type via the DataDecorator
- Supports transactional operations via WithTransaction
- Returns the tenant's commodity count and most recent updated_at timestamp (Count)

#### Shop Subdomain (Seeding)

- Implements the shared seeder library's `Subdomain` interface (`ShopSubdomain`) so shop/commodity data can be loaded from JSON seed files under `npc-shops/shops` on the configured catalog root
- `DeleteAllForTenant` performs a full replace: hard-deletes all existing commodities then all existing shops for the tenant
- `Build` parses one seed file (matching `shop-{npcId}.json`) into a shop model plus its commodity models; `BulkCreate` persists a batch of parsed records in a single transaction
- `Count` reports the tenant's shop row count; `AuxiliaryCounts` additionally reports the commodities row count under a "commodities" key so the seed status response surfaces both

#### Consumable Cache

- Redis-backed per-tenant cache of rechargeable consumable data fetched from the data service
- Lazily loads on first access per tenant, persists to Redis for subsequent reads
