# Domain Documentation

## Asset

### Responsibility
Represents a single inventory item as a unified model. All item types (equipment, consumable, setup, etc, cash, pet) are flattened into a single `Model` struct. The asset's type is determined at runtime from its `templateId` using the `inventory.TypeFromItemId` function.

### Core Models
- `Model` - Unified asset containing:
  - Identity: id (uint32), compartmentId (uuid.UUID), slot (int16), templateId (uint32)
  - Timing: expiration (time.Time), createdAt (time.Time)
  - Stackable fields: quantity (uint32), ownerId (uint32), flag (uint16), rechargeable (uint64)
  - Equipment fields: strength, dexterity, intelligence, luck, hp, mp, weaponAttack, magicAttack, weaponDefense, magicDefense, accuracy, avoidability, hands, speed, jump, slots (all uint16), locked, spikes, karmaUsed, cold, canBeTraded (all bool), levelType (byte), level (byte), experience (uint32), hammersApplied (uint32), equippedSince (*time.Time)
  - Cash fields: cashId (int64), commodityId (uint32), purchaseBy (uint32)
  - Pet fields: petId (uint32), petName (string), closeness (uint16), fullness (byte), petSlot (int8)
- `ModelBuilder` - Fluent builder with `SetX` methods for all fields. Validates that `id > 0` on Build.
- `RestModel` - JSON:API representation with Transform/Extract functions for conversion. `BaseRestModel` is a type alias for `RestModel`.
- `InventoryType` - Type alias for `inventory.Type`. Package-level variables provide convenience aliases: InventoryTypeEquip, InventoryTypeUse, InventoryTypeSetup, InventoryTypeEtc, InventoryTypeCash.

### Invariants
- Asset id must be greater than 0 (`ErrInvalidId`)
- Inventory type is derived from the templateId at runtime; there is no stored type field
- Equipment items always have implicit quantity of 1
- Stackable items (consumable, setup, etc) and non-pet cash items have explicit quantity via `HasQuantity()`
- A pet asset is a cash-type asset (`IsCash()`) with `petId > 0`
- A cash equipment asset has `IsEquipment()` and `cashId != 0`

### State Transitions
None. Assets are read-only projections in the channel service, built from Kafka events or REST responses.

### Processors
None. The channel service does not have an asset processor; assets are constructed by consumers and builders.

---

## Compartment

### Responsibility
Represents a typed subdivision of a character's inventory. Each compartment holds assets of a single inventory type and has a capacity limit.

### Core Models
- `Model` - Contains id (uuid.UUID), characterId (uint32), inventoryType (inventory.Type), capacity (uint32), assets ([]asset.Model). Provides lookup methods: FindBySlot, FindById, FindFirstByItemId, FindByPetId.
- `modelBuilder` - Fluent builder with SetCapacity, AddAsset, SetAssets methods. Validates that id is not uuid.Nil on Build.
- `RestModel` - JSON:API representation with compartment-to-asset relationships

### Invariants
- Compartment id must not be uuid.Nil (`ErrMissingId`)
- A compartment is keyed by its inventory type within an inventory

### Processors
- `Processor` - Retrieves compartments by character ID and type via REST (INVENTORY service). Issues Kafka commands for equip, unequip, move, drop, merge, and sort operations.

---

## Inventory

### Responsibility
Aggregates a character's five compartments (equip, use, setup, etc, cash) into a single model. Provides typed accessors and lookup by compartment type or ID.

### Core Models
- `Model` - Contains characterId (uint32), compartments (map[inventory.Type]compartment.Model). Provides typed accessors: Equipable(), Consumable(), Setup(), ETC(), Cash(), CompartmentByType(), CompartmentById(), Compartments().
- `modelBuilder` - Fluent builder with SetEquipable, SetConsumable, SetSetup, SetEtc, SetCash, and generic SetCompartment methods. Validates characterId > 0.

### Invariants
- Character id must be greater than 0 (`ErrInvalidCharacterId`)
- One compartment per inventory type

### Processors
- `Processor` - Retrieves the full inventory model by character ID via REST (INVENTORY service). Provides ByCharacterIdProvider and GetByCharacterId methods.

---

## Equipment

### Responsibility
Represents the set of equipment slots on a character. Maps slot types to slot models, where each slot holds an optional regular equipable and an optional cash equipable.

### Core Models
- `Model` - Contains slots (map[slot.Type]slot.Model). Initialized from the canonical slot list in `slot.Slots`. Provides Get, Set, and Slots methods.
- `slot.Model` - Contains Position (slot.Position), Equipable (*asset.Model), CashEquipable (*asset.Model)

### Invariants
- Equipment is initialized with all known slot positions
- Regular and cash equipables occupy the same slot type but are stored separately

### Processors
None. Equipment is built by `character.Model.SetInventory()` from the equip compartment's assets based on slot position (negative slots are equipped; further offset by -100 for cash equipment).

---

## Character

### Responsibility
Represents a player character with stats, equipment, inventory, skills, pets, and quests. Provides character data retrieval, stat modification commands, and inventory decoration.

### Core Models
- `Model` - Contains id, accountId, worldId, name, gender, skinColor, face, hair, level, jobId, stats (strength/dexterity/intelligence/luck/hp/mp/maxHp/maxMp), hpMpUsed, ap, sp, experience, fame, gachaponExperience, mapId, spawnPoint, gm, position (x, y, stance), meso, pets ([]pet.Model), equipment (equipment.Model), inventory (inventory.Model), skills ([]skill.Model), quests ([]quest.Model)
- `DistributePacket` - Contains Flag (uint32) and Value (uint32) for AP distribution

### Invariants
- Character must belong to an account
- SP table used for Evan job class (job IDs 2209-2218)
- SetInventory splits equip compartment assets into equipped items (slot <= 0) and inventory items (slot > 0), separating regular and cash equipment by the -100 offset rule

### State Transitions
None within channel. The character model is read-only; mutations are requested via Kafka commands.

### Processors
- `Processor` - Retrieves character by ID or name via REST (CHARACTERS service). Supports decorators: InventoryDecorator, PetModelDecorator, SkillModelDecorator, QuestModelDecorator. Provides GetEquipableInSlot (looks up equip compartment by slot) and GetItemInSlot (looks up any compartment by type and slot). Issues commands for AP distribution, SP distribution, meso drop, HP/MP changes.
- `MockProcessor` - Test double with in-memory character lookup maps.

---

## Character Buff

### Responsibility
Represents active temporary buffs applied to a character.

### Core Models
- `Model` - Contains buffId, characterId, sourceId (skill/item), duration, stats

### Processors
- `Processor` - Retrieves buffs by character ID

---

## Character Key

### Responsibility
Represents key bindings for a character's keyboard configuration.

### Core Models
- `Model` - Contains key, keyType, action

### Processors
- `Processor` - Retrieves key bindings by character ID

---

## Character Skill

### Responsibility
Represents skills learned by a character.

### Core Models
- `Model` - Contains skillId, level, masterLevel, expiration

### Processors
- `Processor` - Retrieves skills by character ID

---

## Channel

### Responsibility
Represents a game channel server instance within a world. Manages channel registration, capacity tracking, and identification.

### Core Models
- `Model` - Immutable channel representation containing id, worldId, channelId, ipAddress, port, currentCapacity, maxCapacity, createdAt

### Invariants
- Channel must have a valid world and channel identifier
- Channel must have a valid IP address and port

### Processors
- `Processor` - Registers channels, retrieves channel by world and channel ID

---

## Server

### Responsibility
Represents the runtime server context for a tenant's world and channel combination. Provides tenant-aware filtering for Kafka message handling.

### Core Models
- `Model` - Contains tenant, worldId, channelId, ipAddress, port

### Invariants
- Server must be bound to a valid tenant
- Server must have valid world and channel identifiers

### Processors
- `Register` - Creates a server model for a tenant/world/channel

---

## Session

### Responsibility
Manages active client socket connections. Tracks session state including account, character, world, channel, and map associations. Handles encrypted packet transmission.

### Core Models
- `Model` - Contains sessionId, accountId, characterId, worldId, channelId, mapId, gm flag, storageNpcId, connection, encryption state, lastPacket timestamp, locale

### Invariants
- Session must have a valid connection
- Session must be associated with a tenant
- Character can only have one active session per world/channel

### Processors
- `Processor` - Creates sessions, manages session lifecycle, handles encryption/decryption, broadcasts announcements, tracks session state changes

---

## Account

### Responsibility
Tracks account login state within the channel service. Maintains an in-memory registry of logged-in accounts per tenant.

### Core Models
- `Key` - Composite key containing Tenant and account Id
- Registry (internal) - Thread-safe map of logged-in accounts

### Invariants
- Account can only be logged in once per tenant

### Processors
- `Processor` - Initializes registry from external service on startup

---

## Buddy List

### Responsibility
Represents a character's friend list.

### Core Models
- `Model` - Contains characterId, capacity, buddies list
- `Buddy Model` - Contains id, name, channelId, visible flag, group name

### Processors
- `Processor` - Retrieves buddy list by character ID

---

## Guild

### Responsibility
Represents player guilds including membership, ranks, and emblem.

### Core Models
- `Model` - Contains id, name, notice, emblem, capacity, members, titles, threads

### Processors
- `Processor` - Retrieves guild by member ID

---

## Party

### Responsibility
Represents player parties for cooperative gameplay.

### Core Models
- `Model` - Contains id, leaderId, members list

### Processors
- `Processor` - Retrieves party by member ID, provides member filtering

---

## Messenger

### Responsibility
Represents in-game messenger rooms for private multi-character chat.

### Core Models
- `Model` - Contains id, members list

### Processors
- `Processor` - Retrieves messenger by character ID

---

## Map

### Responsibility
Provides queries for characters present in a map. Coordinates session lookups for map-based broadcasts.

### Core Models
- Uses `_map.Model` from atlas-constants (worldId, channelId, mapId)

### Processors
- `Processor` - Retrieves character IDs in map, iterates sessions in map, filters for other characters

---

## Monster

### Responsibility
Represents spawned monsters in a map.

### Core Models
- `Model` - Contains uniqueId, monsterId, worldId, channelId, mapId, hp, maxHp, mp, position (x, y, fh), stance, team, controlCharacterId

### Invariants
- Monster must exist in a valid map

### Processors
- `Processor` - Iterates monsters in map

---

## Drop

### Responsibility
Represents items or meso dropped on the ground in a map.

### Core Models
- `Model` - Contains id, itemId, equipmentId, quantity, meso, dropType, position (x, y), ownerId, ownerPartyId, dropTime, dropperId, dropperPosition, playerDrop flag

### Processors
- `Processor` - Iterates drops in map

---

## Reactor

### Responsibility
Represents interactive objects in maps that respond to player actions.

### Core Models
- `Model` - Contains id, worldId, channelId, mapId, classification, name, state, eventState, delay, direction, x, y, updateTime

### Processors
- `Processor` - Iterates reactors in map, issues hit commands

---

## Chair

### Responsibility
Tracks characters sitting in portable chairs.

### Core Models
- `Model` - Contains characterId, chair id

### Processors
- `Processor` - Iterates chairs in map

---

## Chalkboard

### Responsibility
Tracks character chalkboard messages displayed above characters.

### Core Models
- `Model` - Contains character id, message text

### Processors
- `Processor` - Iterates chalkboards in map

---

## Pet

### Responsibility
Represents character pets.

### Core Models
- `Model` - Contains pet id, slot, name, template data

### Processors
- `Processor` - Retrieves pets by owner character ID

---

## Note

### Responsibility
Represents in-game mail notes between characters.

### Core Models
- `Model` - Contains id, senderId, message, timestamp, flag

### Processors
- `Processor` - Retrieves notes by character ID

---

## Macro

### Responsibility
Represents skill macros configured by characters.

### Core Models
- `Model` - Contains id, name, shout flag, skillId1, skillId2, skillId3

### Processors
- `Processor` - Retrieves macros by character ID

---

## World

### Responsibility
Represents a game world with configuration and message of the day.

### Core Models
- `Model` - Contains worldId, message

### Processors
- `Processor` - Retrieves world by ID

---

## Storage

### Responsibility
Handles storage (warehouse) operations for depositing and withdrawing items. Retrieves storage data and projection data from the atlas-storage service. Produces commands for arrange, meso deposit/withdraw, and close operations.

### Core Models
- `StorageData` - Contains Capacity (byte), Mesos (uint32), Assets ([]asset.Model)
- `ProjectionData` - Contains CharacterId (uint32), AccountId (uint32), WorldId (world.Id), Capacity (byte), Mesos (uint32), NpcId (uint32), Compartments (map[string][]asset.Model). Provides GetAllAssetsFromProjection() to retrieve all assets from the equip compartment.
- `StorageRestModel` - JSON:API representation of storage with included assets via relationship
- `AssetRestModel` - Storage asset with id (uint32), slot (int16), templateId (uint32), expiration (time.Time), referenceId (uint32), referenceType (string), referenceData (interface{}). Uses custom UnmarshalJSON with referenceType discriminator.
- `ProjectionRestModel` - Storage projection with compartments as map[string]json.RawMessage, parsed via ParseCompartmentAssets()
- Reference data types: EquipableRestData, ConsumableRestData, SetupRestData, EtcRestData, CashRestData, PetRestData (each embedding BaseData and type-specific fields)

### Invariants
- Default storage capacity is 4 slots (`DefaultStorageCapacity`)
- If storage does not exist, an empty StorageData with default capacity is returned (fail-open)
- Asset transformation from storage REST models uses a `referenceType` discriminator to map typed reference data (equipable, cash_equipable, consumable, setup, etc, cash, pet) into the unified asset.Model

### Processors
- `Processor` - GetStorageData (fetches storage metadata and assets), GetProjectionData (fetches projection with compartment assets), Arrange (sends ARRANGE command), DepositMesos (sends UPDATE_MESOS with ADD), WithdrawMesos (sends UPDATE_MESOS with SUBTRACT), CloseStorage (sends CLOSE_STORAGE command)

---

## Cash Shop

### Responsibility
Handles cash shop operations including inventory, wallet, and wishlist management.

### Core Models
- `Inventory Model` - Contains compartments with cash items
- `Wallet Model` - Contains NX credit balances
- `Wishlist Model` - Contains wish list item entries

### Processors
- `Processor` - Manages cash shop state and operations

---

## NPC

### Responsibility
Provides NPC data and conversation handling.

### Processors
- `Processor` - Retrieves NPCs in map, handles shop and conversation operations

---

## Transport Route

### Responsibility
Manages transport (boat/train) routes and schedules between maps.

### Core Models
- `Model` - Contains route state and map associations

### Processors
- `Processor` - Checks if transport is in map

---

## Quest

### Responsibility
Represents character quest progress and state.

### Core Models
- `Model` - Contains id, characterId, questId, state, startedAt, completedAt, expirationTime, completedCount, forfeitCount, progress
- `Progress` - Contains infoNumber, progress string

### State Transitions
- StateNotStarted -> StateStarted -> StateCompleted
- StateStarted -> StateNotStarted (forfeit)

### Processors
- `Processor` - Retrieves quests by character ID, issues start/complete/forfeit/restore item commands

---

## Saga

### Responsibility
Handles distributed transaction orchestration for multi-step operations.

### Processors
- `Processor` - Manages saga state and compensation
