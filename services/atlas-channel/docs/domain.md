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
  - Pet fields: petId (uint32), petName (string), petLevel (byte), closeness (uint16), fullness (byte), petSlot (int8)
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
- `Model` - Contains id (uint32), accountId (uint32), worldId (world.Id), name (string), gender (byte), skinColor (byte), face (uint32), hair (uint32), level (byte), jobId (job.Id), strength (uint16), dexterity (uint16), intelligence (uint16), luck (uint16), hp (uint16), maxHp (uint16), mp (uint16), maxMp (uint16), hpMpUsed (int), ap (uint16), sp (string), experience (uint32), fame (uint32), gachaponExperience (uint32), mapId (uint32), spawnPoint (uint32), gm (int), x (int16), y (int16), stance (byte), meso (uint32), pets ([]pet.Model), equipment (equipment.Model), inventory (inventory.Model), skills ([]skill.Model), quests ([]quest.Model)
- `DistributePacket` - Contains Flag (uint32) and Value (uint32) for AP distribution

### Invariants
- Character id must be greater than 0 (`ErrInvalidId`)
- Character must belong to an account
- SP stored as comma-separated string; SP table used for Evan job class (job IDs 2200-2218)
- SetInventory splits equip compartment assets into equipped items (slot <= 0) and inventory items (slot > 0), separating regular and cash equipment by the -100 offset rule

### State Transitions
None within channel. The character model is read-only; mutations are requested via Kafka commands.

### Processors
- `Processor` - Retrieves character by ID or name via REST (CHARACTERS service). Supports decorators: InventoryDecorator, PetAssetEnrichmentDecorator, SkillModelDecorator, QuestModelDecorator. Provides GetEquipableInSlot (looks up equip compartment by slot) and GetItemInSlot (looks up any compartment by type and slot). Issues commands for AP distribution, SP distribution, meso drop, HP/MP changes.
- `MockProcessor` - Test double with in-memory character lookup maps.

---

## Character Buff

### Responsibility
Represents active temporary buffs applied to a character.

### Core Models
- `Model` - Contains sourceId (int32), level (byte), duration (int32), changes ([]stat.Model), createdAt (time.Time), expiresAt (time.Time). Expired() checks if expiresAt is before current time.
- `stat.Model` - Contains statType (string), amount (int32)

### Processors
- `Processor` - Retrieves buffs by character ID via REST (BUFFS service)

---

## Character Key

### Responsibility
Represents key bindings for a character's keyboard configuration.

### Core Models
- `Model` - Contains key (int32), theType (int8), action (int32)

### Processors
- `Processor` - Retrieves key bindings by character ID via REST (KEYS service). Issues PATCH to update individual key bindings.

---

## Character Skill

### Responsibility
Represents skills learned by a character.

### Core Models
- `Model` - Contains id (skill.Id), level (byte), masterLevel (byte), expiration (time.Time), cooldownExpiresAt (time.Time). IsFourthJob() and OnCooldown() derived methods.

### Processors
- `Processor` - Retrieves skills by character ID via REST (SKILLS service)

---

## Channel

### Responsibility
Represents a game channel server instance within a world. Manages channel registration, capacity tracking, and identification.

### Core Models
- `Model` - Immutable channel representation containing id (uuid.UUID), worldId (world.Id), channelId (channel.Id), ipAddress (string), port (int), currentCapacity (uint32), maxCapacity (uint32), createdAt (time.Time)

### Invariants
- Channel id must not be uuid.Nil (`ErrInvalidId`)

### Processors
- `Processor` - Registers channels via POST (CHANNELS service), retrieves channel by world and channel ID via GET

---

## Server

### Responsibility
Represents the runtime server context for a tenant's world and channel combination. Provides tenant-aware filtering for Kafka message handling.

### Core Models
- `Model` - Contains tenant (tenant.Model), ch (channel.Model), ipAddress (string), port (int). Convenience methods: Tenant(), Channel(), WorldId(), ChannelId(), IpAddress(), Port(), Map(), Field(), Is(), IsWorld(), String()

### Invariants
- Server must be bound to a valid tenant
- Server must have valid world and channel identifiers

### Processors
- `Register` - Creates a server model for a tenant/world/channel and adds it to the singleton registry
- `GetAll` - Returns all registered server models

---

## Session

### Responsibility
Manages active client socket connections. Tracks session state including account, character, field (world/channel/map/instance), and storage NPC associations. Handles encrypted packet transmission.

### Core Models
- `Model` - Contains id (uuid.UUID), accountId (uint32), characterId (uint32), field (field.Model), gm (bool), storageNpcId (uint32), con (net.Conn), send (crypto.AESOFB), sendLock (*sync.Mutex), recv (crypto.AESOFB), encryptFunc (crypto.EncryptFunc), lastPacket (time.Time), locale (byte)

### Invariants
- Session must have a valid connection
- AESOFB cipher initialization varies based on tenant region/version: GMS v12 and below use FillIvZeroGenerator; JMS disables MapleEncryption
- Session state is immutable; mutations produce new session instances via CloneSession

### Processors
- `Processor` - Creates sessions, manages session lifecycle via tenant-keyed registry, handles encryption/decryption, broadcasts announcements to sessions, tracks session state changes, provides ForEachByCharacterId for map-based iteration

---

## Account

### Responsibility
Tracks account login state within the channel service. Maintains an in-memory registry of logged-in accounts per tenant.

### Core Models
- `Model` - Contains id (string), name (string), password (string), pin (string), pic (string), loggedIn (int), lastLogin (uint64), gender (byte), banned (bool), tos (bool), language (string), country (string), characterSlots (int16)
- `Key` - Composite key containing Tenant and account Id

### Invariants
- Account can only be logged in once per tenant

### Processors
- `Processor` - Retrieves account by ID via REST (ACCOUNTS service). InitializeRegistry populates account state on startup. IsLoggedIn checks the in-memory registry.

---

## Account Session

### Responsibility
Manages account session lifecycle by producing Kafka commands for session state progression and logout.

### Processors
- `Processor` (interface) - Destroy(sessionId, accountId) emits LOGOUT command. UpdateState(sessionId, accountId, state, params) emits PROGRESS_STATE command.

---

## Buddy List

### Responsibility
Represents a character's friend list.

### Core Models
- `Model` - Contains tenantId (uuid.UUID), id (uuid.UUID), characterId (uint32), capacity (byte), buddies ([]buddy.Model)
- `buddy.Model` - Contains listId (uuid.UUID), characterId (uint32), group (string), characterName (string), channelId (int8), inShop (bool), pending (bool)

### Processors
- `Processor` - Retrieves buddy list by character ID via REST (BUDDIES service). Issues RequestAdd and RequestDelete commands via Kafka.

---

## Guild

### Responsibility
Represents player guilds including membership, ranks, emblem, and BBS threads.

### Core Models
- `Model` - Contains id (uint32), worldId (world.Id), name (string), notice (string), points (uint32), capacity (uint32), logo (uint16), logoColor (byte), logoBackground (uint16), logoBackgroundColor (byte), leaderId (uint32), members ([]member.Model), titles ([]title.Model)
- `member.Model` - Contains characterId (uint32), name (string), jobId (uint16), level (byte), title (byte), online (bool), allianceTitle (byte)
- `title.Model` - Contains name (string), index (byte)
- `thread.Model` - Contains tenantId (uuid.UUID), guildId (uint32), id (uint32), posterId (uint32), emoticonId (uint32), title (string), message (string), notice (bool), createdAt (time.Time), replies ([]reply.Model)
- `reply.Model` - Contains id (uint32), posterId (uint32), message (string), createdAt (time.Time)

### Invariants
- Members() returns members sorted by name
- IsLeader() checks if characterId matches leaderId
- IsLeadership() checks if member title <= 2
- TitlePossible() validates leadership permission

### Processors
- `Processor` - Retrieves guild by ID or by member ID via REST (GUILDS service). Issues guild commands via Kafka for creation, emblem, notice, titles, member title, and leave operations.
- `thread.Processor` - Retrieves threads and thread details via REST (GUILD_THREADS service). Issues guild thread commands via Kafka for create, update, delete, add reply, and delete reply operations.

---

## Party

### Responsibility
Represents player parties for cooperative gameplay.

### Core Models
- `Model` - Contains id (uint32), leaderId (uint32), members ([]MemberModel)
- `MemberModel` - Contains id (uint32), name (string), level (byte), jobId (job.Id), field (field.Model), online (bool). Provides WorldId(), ChannelId(), MapId(), Instance() convenience accessors.

### Processors
- `Processor` - Retrieves party by member ID or by ID via REST (PARTIES service). Issues commands via Kafka for create, leave, expel, change leader, and request invite operations.

---

## Messenger

### Responsibility
Represents in-game messenger rooms for private multi-character chat.

### Core Models
- `Model` - Contains id (uint32), members ([]MemberModel)
- `MemberModel` - Contains id (uint32), name (string), worldId (world.Id), channelId (channel.Id), online (bool), slot (byte)

### Processors
- `Processor` - Retrieves messenger by ID or by character ID via REST (MESSENGERS service). Issues commands via Kafka for create, leave, and request invite operations.

---

## Map

### Responsibility
Provides queries for characters present in a map. Coordinates session lookups for map-based broadcasts.

### Processors
- `Processor` - CharacterIdsInMapModelProvider (retrieves character IDs in a field instance via REST from MAPS service), GetCharacterIdsInMap, ForSessionsInMap (iterates sessions for characters in a field), ForSessionsInSessionsMap (iterates sessions in the caller session's map), ForOtherSessionsInMap (excludes a reference character), CharacterIdsInMapAllInstancesModelProvider (retrieves across all instances), ForSessionsInMapAllInstances

---

## Monster

### Responsibility
Represents spawned monsters in a map. Provides monster data retrieval, damage application, skill usage, and status effect management.

### Core Models
- `Model` - Contains field (field.Model), uniqueId (uint32), monsterId (uint32), maxHp (uint32), hp (uint32), mp (uint32), controlCharacterId (uint32), x (int16), y (int16), fh (int16), stance (byte), team (int8), statusEffects ([]StatusEffectEntry). Delegates WorldId(), ChannelId(), MapId(), Instance() to embedded field.Model. Controlled() returns true when controlCharacterId != 0.
- `StatusEffectEntry` - Contains sourceSkillId (uint32), sourceSkillLevel (uint32), statuses (map[string]int32), expiresAt (time.Time)
- `modelBuilder` - Constructor requires uniqueId, field, monsterId. Validates uniqueId > 0 (`ErrInvalidUniqueId`).

### Invariants
- uniqueId must be greater than 0
- Monster field identity delegated to embedded field.Model

### Processors
- `Processor` - GetById (fetches monster by uniqueId via REST from MONSTERS service), InMapModelProvider/ForEachInMap/GetInMap (retrieves and iterates monsters in a field), Damage (emits DAMAGE command), UseSkill (emits USE_SKILL command), ApplyStatus (emits APPLY_STATUS command), CancelStatus (emits CANCEL_STATUS command)

---

## Drop

### Responsibility
Represents items or meso dropped on the ground in a map.

### Core Models
- `Model` - Contains id (uint32), itemId (uint32), equipmentId (uint32), quantity (uint32), meso (uint32), dropType (byte), x (int16), y (int16), ownerId (uint32), ownerPartyId (uint32), dropTime (time.Time), dropperId (uint32), dropperX (int16), dropperY (int16), playerDrop (bool)

### Invariants
- Drop id must be greater than 0 (`ErrInvalidId`)

### Processors
- `Processor` - InMapModelProvider/ForEachInMap (retrieves and iterates drops in a field via REST from DROPS service). RequestReservation (emits drop pickup reservation command via Kafka).

---

## Reactor

### Responsibility
Represents interactive objects in maps that respond to player actions.

### Core Models
- `Model` - Contains id (uint32), field (field.Model), classification (uint32), name (string), state (int8), eventState (byte), delay (uint32), direction (byte), x (int16), y (int16), updateTime (time.Time)

### Invariants
- Reactor id must be greater than 0 (`ErrInvalidId`)

### Processors
- `Processor` - InMapModelProvider/ForEachInMap (retrieves and iterates reactors in a field via REST from REACTORS service). Hit (emits HIT command via Kafka).

---

## Chair

### Responsibility
Tracks characters sitting in portable chairs.

### Core Models
- `Model` - Contains id (uint32), chairType (string), characterId (uint32)

### Processors
- `Processor` - InMapModelProvider/ForEachInMap (retrieves chairs in a field via REST from CHAIRS service). Use (emits USE command), Cancel (emits CANCEL command) via Kafka.

---

## Chalkboard

### Responsibility
Tracks character chalkboard messages displayed above characters.

### Core Models
- `Model` - Contains id (uint32), message (string)

### Processors
- `Processor` - InMapModelProvider/ForEachInMap (retrieves chalkboards in a field via REST from CHALKBOARDS service). AttemptUse (emits SET command), Close (emits CLEAR command) via Kafka.

---

## Pet

### Responsibility
Represents character pets with position, stats, and exclude lists.

### Core Models
- `Model` - Contains id (uint32), cashId (uint64), templateId (uint32), name (string), level (byte), closeness (uint16), fullness (byte), expiration (time.Time), ownerId (uint32), slot (int8), x (int16), y (int16), stance (byte), fh (int16), excludes ([]exclude.Model), flag (uint16), purchaseBy (uint32)
- `exclude.Model` - Contains id (uint32), itemId (uint32)

### Invariants
- Pet id must be greater than 0 (`ErrInvalidId`)
- Slot 0 indicates lead pet

### Processors
- `Processor` - Retrieves pets by ID or by owner via REST (PETS service). Issues commands via Kafka for spawn, despawn, attempt command, and set exclude list.

---

## Note

### Responsibility
Represents in-game mail notes between characters.

### Core Models
- `Model` - Contains id (uint32), characterId (uint32), senderId (uint32), message (string), timestamp (time.Time), flag (byte)

### Invariants
- Note id must be greater than 0 (`ErrInvalidId`)

### Processors
- `Processor` - Retrieves notes by character ID or by note ID via REST (NOTES service). Issues commands via Kafka for SendNote and DiscardNotes.

---

## Macro

### Responsibility
Represents skill macros configured by characters.

### Core Models
- `Model` - Contains id (uint32), name (string), shout (bool), skillId1 (skill.Id), skillId2 (skill.Id), skillId3 (skill.Id)

### Processors
- `Processor` - Retrieves macros by character ID via REST (SKILLS service). Issues UPDATE command via Kafka.

---

## World

### Responsibility
Represents a game world with configuration, channels, and rates.

### Core Models
- `Model` - Contains id (world.Id), name (string), state (State: Normal/Event/New/Hot), message (string), eventMessage (string), recommendedMessage (string), capacityStatus (Status: Normal/HighlyPopulated/Full), channels ([]channel.Model)

### Processors
- `Processor` - Retrieves world by ID via REST (WORLDS service)

---

## Storage

### Responsibility
Handles storage (warehouse) operations for depositing and withdrawing items. Retrieves storage data and projection data from the atlas-storage service. Produces commands for arrange, meso deposit/withdraw, and close operations.

### Core Models
- `StorageData` - Contains Capacity (byte), Mesos (uint32), Assets ([]asset.Model)
- `ProjectionData` - Contains CharacterId (uint32), AccountId (uint32), WorldId (world.Id), Capacity (byte), Mesos (uint32), NpcId (uint32), Compartments (map[string][]asset.Model). Provides GetAllAssetsFromProjection() to retrieve all assets from the equip compartment.
- `StorageRestModel` - JSON:API representation with included assets via relationship. Resource type: "storages".
- `AssetRestModel` - Storage asset with id (uint32), slot (int16), templateId (uint32), expiration (time.Time), quantity (uint32), ownerId (uint32), flag (uint16), rechargeable (uint64), equipment stats, cashId (int64), commodityId (uint32), purchaseBy (uint32), petId (uint32). Resource type: "storage_assets".
- `ProjectionRestModel` - Storage projection with characterId, accountId, worldId, storageId, capacity, mesos, npcId, and compartments (map[string]json.RawMessage). Resource type: "storage_projections". Parsed via ParseCompartmentAssets().

### Invariants
- Default storage capacity is 4 slots (`DefaultStorageCapacity`)
- If storage does not exist, an empty StorageData with default capacity is returned (fail-open)

### Processors
- `Processor` - GetStorageData (fetches storage metadata and assets via REST), GetProjectionData (fetches projection with compartment assets via REST), Arrange (sends ARRANGE command via Kafka), DepositMesos (sends UPDATE_MESOS with ADD), WithdrawMesos (sends UPDATE_MESOS with SUBTRACT), CloseStorage (sends CLOSE_STORAGE command)

---

## Cash Shop

### Responsibility
Handles cash shop operations including entry/exit, purchases, inventory management, and item transfers.

### Core Models
- `inventory.Model` - Contains accountId (uint32), compartments (map[CompartmentType]compartment.Model). AccountId must be > 0.
- `compartment.Model` - Contains id (uuid.UUID), accountId (uint32), type_ (CompartmentType), capacity (uint32), assets ([]asset.Model). CompartmentType: TypeExplorer (1), TypeCygnus (2), TypeLegend (3).
- `asset.Model` - Contains id (uuid.UUID), compartmentId (uuid.UUID), item (item.Model). Both id and compartmentId must not be uuid.Nil.
- `item.Model` - Contains id (uint32), cashId (int64), templateId (uint32), commodityId (uint32), quantity (uint32), flag (uint16), purchasedBy (uint32), expiration (time.Time). Id must be > 0.
- `wallet.Model` - Contains id (uuid.UUID), accountId (uint32), credit (uint32), points (uint32), prepaid (uint32)
- `wishlist.Model` - Contains id (uuid.UUID), characterId (uint32), serialNumber (uint32)

### Processors
- `Processor` - Enter/Exit (emits cash shop enter/exit commands), RequestPurchase, RequestInventoryIncreasePurchaseByType/ByItem, RequestStorageIncreasePurchase/ByItem, RequestCharacterSlotIncreasePurchaseByItem, MoveFromCashInventory, MoveToCashInventory
- `inventory.asset.Processor` - ByIdProvider/GetById, ByCompartmentIdProvider/GetByCompartmentId, GetByItemId (retrieves cash shop assets via REST from CASHSHOP service)
- `inventory.compartment.Processor` - ByTypeProvider/GetByType (retrieves compartments via REST from CASHSHOP service)
- `wallet.Processor` - Retrieves wallet by account ID via REST (CASHSHOP service)
- `wishlist.Processor` - Retrieves, adds, and clears wishlist via REST (CASHSHOP service)

---

## NPC

### Responsibility
Provides NPC conversation handling and shop operations.

### Processors
- `Processor` - StartConversation (emits START_CONVERSATION command), ContinueConversation (emits CONTINUE_CONVERSATION command), DisposeConversation (emits END_CONVERSATION command) via Kafka
- `shops.Processor` - Retrieves NPC shop by template ID via REST (NPC_SHOP service). EnterShop, ExitShop, BuyItem, SellItem, RechargeItem (emit NPC shop commands via Kafka)

### NPC Shop Models
- `shops.Model` - Contains npcId (uint32), commodities ([]commodities.Model). NpcId must be > 0.
- `commodities.Model` - Contains id (uuid.UUID), templateId (uint32), mesoPrice (uint32), tokenPrice (uint32), period (uint32), levelLimit (uint32), discountRate (byte), tokenTemplateId (uint32), unitPrice (float64), slotMax (uint32). Id must not be uuid.Nil.

---

## Transport Route

### Responsibility
Manages transport (boat/train) routes and schedules between maps.

### Core Models
- `Model` - Contains id (uuid.UUID), name (string), startMapId (_map.Id), stagingMapId (_map.Id), destinationMapId (_map.Id), enRouteMapIds ([]_map.Id), state (RouteState), schedule ([]TripScheduleModel), boardingWindowDuration (time.Duration), preDepartureDuration (time.Duration), travelDuration (time.Duration), cycleInterval (time.Duration)
- `RouteState` constants: OutOfService, Boarding, PreDeparture, InTransit, Arrived
- `TripScheduleModel` - Contains tripId (uuid.UUID), routeId (uuid.UUID), boardingOpen (time.Time), boardingClosed (time.Time), departure (time.Time), arrival (time.Time)

### Invariants
- Route id must not be uuid.Nil (`ErrInvalidId`)

### Processors
- `Processor` - Retrieves routes by ID, by state, by schedule via REST (ROUTES service). InTenantProvider retrieves all routes in tenant. IsBoatInMap checks if any route's stagingMapId or enRouteMapIds match the given map and the route is in Boarding/PreDeparture/InTransit state.

---

## Quest

### Responsibility
Represents character quest progress and state.

### Core Models
- `Model` - Contains id (uint32), characterId (uint32), questId (uint32), state (State), startedAt (time.Time), completedAt (time.Time), expirationTime (time.Time), completedCount (uint32), forfeitCount (uint32), progress ([]Progress)
- `Progress` - Contains infoNumber (uint32), progress (string)
- `State` constants: NotStarted, Started, Completed

### State Transitions
- NotStarted -> Started -> Completed
- Started -> NotStarted (forfeit)

### Processors
- `Processor` - Retrieves quests by character ID via REST (QUESTS service). Issues commands via Kafka: StartQuestConversation, StartQuest, CompleteQuest, ForfeitQuest, RestoreItem.

---

## Saga

### Responsibility
Handles distributed transaction orchestration for multi-step operations. Re-exports types from the atlas-saga shared library.

### Core Models
- `Saga` - Re-exported from shared library with Type, Status, Actions, Steps
- Payload types: AwardMesosPayload, AwardAssetPayload, DestroyAssetPayload, SetHPPayload, and others from the shared library
- Local `TransferToCashShopPayload` - Contains CashId (uint64), overriding the shared library's int64 type
- Status constants: Pending, Completed, Failed
- Action constants: AwardMesos, DestroyAsset, DepositToStorage, WithdrawFromStorage, and others

### Processors
- `Processor` - Create(s Saga) emits saga commands via Kafka

---

## Party Quest

### Responsibility
Provides party quest timer data for characters.

### Core Models
- `TimerModel` - Contains characterId (uint32), duration (time.Duration)

### Processors
- `Processor` - GetTimerByCharacterId (retrieves timer via REST from PARTY_QUESTS service)

---

## Movement

### Responsibility
Handles entity movement processing for characters, NPCs, pets, and monsters. Folds movement elements into final position/stance summaries and broadcasts results to map sessions.

### Core Models
- `summary` - Accumulated movement result containing X (int16), Y (int16), Stance (byte)
- Movement type constants: TypeNormal, TypeTeleport, TypeStartFallDown, TypeFlyingBlock, TypeJump, TypeStatChange

### Invariants
- ForMonster validates that the monster's worldId/channelId/mapId matches the field; rejects movement on mismatch

### Processors
- `Processor` - ForCharacter (broadcasts character movement to map sessions, emits character movement command), ForNPC (sends NPC action to controller session), ForPet (broadcasts pet movement to map sessions, emits pet movement command), ForMonster (validates map consistency, sends movement ACK to controller, broadcasts to map sessions, emits monster movement command; triggers monster UseSkill when skillId > 0)

---

## Respawn

### Responsibility
Handles character death and respawn logic. Orchestrates experience loss calculation, protective item detection, and multi-step saga creation for the respawn sequence.

### Invariants
- Beginners lose no experience on death
- Maps with NoExpLossOnDeath field limit prevent experience loss
- Protective items (Safety Charm in Cash, Easter Basket or ProtectOnDeath in ETC) prevent experience loss
- Experience loss in towns: 1% of current experience
- Experience loss outside towns with luck < 50: 10%
- Experience loss outside towns with luck >= 50: 5%
- Wheel of Fortune keeps character in current map on death; otherwise warps to map's returnMapId

### Processors
- `Processor` (interface) - Respawn(ch, characterId, currentMapId) orchestrates death penalty via saga with conditional steps: consume_wheel_of_fortune (if used), consume_protective_item (if present), set_hp (always, sets HP to 50), deduct_experience (if loss > 0), cancel_all_buffs (always), warp_to_spawn (always, portalId 0)

---

## Portal

### Responsibility
Handles portal entry and warp commands for map transitions.

### Processors
- `Processor` (interface) - Enter(f, portalName, characterId) looks up portal by name in map data and emits ENTER command via Kafka. Warp(f, characterId, targetMapId) emits WARP command with target map ID.

---

## Fame

### Responsibility
Handles fame change requests between characters.

### Processors
- `Processor` - RequestChange(f, characterId, targetId, amount) emits fame change command via Kafka

---

## Consumable

### Responsibility
Handles item consumption and scroll use requests.

### Processors
- `Processor` - RequestItemConsume(f, characterId, itemId, source, updateTime) emits item consume command via Kafka. RequestScrollUse(f, characterId, scrollSlot, equipSlot, whiteScroll, legendarySpirit, updateTime) emits scroll use command via Kafka.

---

## Invite

### Responsibility
Handles invite accept and reject operations for party, guild, buddy, and messenger invitations. Invites are world-scoped, not field-scoped.

### Processors
- `Processor` - Accept(actorId, worldId, inviteType, referenceId) emits accept invite command. Reject(actorId, worldId, inviteType, originatorId) emits reject invite command. Invite types: PARTY, BUDDY, GUILD, MESSENGER.

---

## Character Expression

### Responsibility
Handles character expression (emote) changes.

### Processors
- `Processor` (interface) - Change(characterId, f, expression) emits expression command via Kafka

---

## Message

### Responsibility
Handles chat message production across multiple chat types. Provides type-specific methods that delegate to the appropriate Kafka command structure.

### Processors
- `Processor` (interface) - GeneralChat (field-scoped, with balloonOnly flag), BuddyChat/PartyChat/GuildChat/AllianceChat (delegate to MultiChat with type string), MultiChat (with recipients list), WhisperChat (with recipientName), MessengerChat (with recipients list), PetChat (with ownerId, petSlot, type, action, balloon)

---

## Weather

### Responsibility
Provides active weather state for a field.

### Processors
- `Processor` - GetActive(f) retrieves active weather via REST (WEATHER service)

---

## Kite

### Responsibility
Represents kite/balloon display items in the game world.

### Core Models
- `Model` - Contains id (uint32), templateId (uint32), message (string), name (string), x (int16), y (int16), ft (int16, accessed via Type() getter)

### Processors
None. Model-only domain.

---

## Data/Skill

### Responsibility
Provides static skill data retrieval from the DATA service, including skill metadata and level-indexed effect lookup.

### Core Models
- `Model` - Contains id (uint32), action (bool), element (string), animationTime (uint32), effects ([]effect.Model)

### Invariants
- GetEffect returns empty model when level is 0
- GetEffect indexes effects at level-1; returns error if level exceeds effects array length

### Processors
- `Processor` (interface) - GetById(skillId) fetches skill data via REST. GetEffect(skillId, level) fetches skill then returns effect at level-1 index. SetCooldownCommandProvider emits SET_COOLDOWN command to COMMAND_TOPIC_SKILL.

---

## Data/Skill Effect

### Responsibility
Represents a single skill effect level with stat modifications, resource costs, monster status effects, and cure information.

### Core Models
- `Model` - Contains stat modifiers (weaponAttack, magicAttack, weaponDefense, magicDefense, accuracy, avoidability, speed, jump as int16), resource fields (hp, mp as uint16, hpr, mpr as float64), rate fields (mhprRate, mmprRate as uint16, mhpR, mmpR as byte), mob skill fields (mobSkill, mobSkillLevel as uint16), combat fields (damage, attackCount as uint32, fixDamage as int32, bulletCount, bulletConsume as uint16), cost fields (hpCon, mpCon as uint16, moneyCon as uint32, itemCon as uint32, itemConNo as uint32), timing fields (duration as int32, cooldown as uint32), targeting fields (target as uint32, mobCount as uint32), effect fields (morphId, ghost, fatigue, berserk, booster as uint32, prop as float64, barrier as int32, moveTo as int32, cp, nuffSkill as uint32), flags (overtime, repeatEffect, skill as bool, mapProtection as byte), position (x, y as int16), collections (cureAbnormalStatuses as []string, statups as []statup.Model, monsterStatus as map[string]uint32)
- `statup.Model` - Contains buffType (string), amount (int32). Mask() returns buffType.
- Public getters: StatUps(), HPConsume(), MPConsume(), Duration(), Cooldown(), ItemConsume(), ItemConsumeAmount(), MonsterStatus(), CureAbnormalStatuses()

---

## Data/Map

### Responsibility
Provides static map metadata from the DATA service including return map, field limits, and town status.

### Core Models
- `Model` - Contains clock (bool), returnMapId (_map.Id), fieldLimit (uint32), town (bool). Derived method: NoExpLossOnDeath() delegates to _map.NoExpLossOnDeath(fieldLimit).

### Processors
- `Processor` (interface) - GetById(mapId) fetches map metadata via REST

---

## Data/Portal

### Responsibility
Provides static portal data lookup from the DATA service by map and portal name.

### Core Models
- `Model` - Contains id (uint32), name (string), target (string), portalType (uint8), x (int16), y (int16), targetMapId (_map.Id), scriptName (string). Only Id() has a public getter.

### Processors
- `Processor` (interface) - InMapByNameModelProvider(mapId, name) returns portal slice provider. GetInMapByName(mapId, name) returns first matching portal.

---

## Data/NPC

### Responsibility
Provides static NPC positional data from the DATA service for map-spawned NPC instances.

### Core Models
- `Model` - Contains id (uint32), template (uint32), x (int16), cy (int16), f (uint32), fh (uint16), rx0 (int16), rx1 (int16)

### Processors
- `Processor` - ForEachInMap(mapId, operator) iterates NPCs in parallel. InMapModelProvider(mapId) returns NPC slice provider. InMapByObjectIdModelProvider(mapId, objectId) filters by object ID. GetInMapByObjectId(mapId, objectId) returns first match.

---

## Data/NPC Template

### Responsibility
Provides NPC template metadata from the DATA service including storage-related configuration.

### Core Models
- `Model` - Contains id (uint32), name (string), trunkPut (int32), trunkGet (int32), storebank (bool). IsStorageNpc(), GetDepositFee(), GetWithdrawFee() derived methods.

### Processors
- `Processor` (interface) - ByIdProvider(npcId) returns NPC template provider. GetById(npcId) fetches NPC template via REST.

---

## Data/Quest

### Responsibility
Provides static quest definition data from the DATA service including requirements, actions, and rewards.

### Core Models
- `Model` - Contains id (uint32), name (string), parent (string), area (uint32), order (uint32), autoStart (bool), autoPreComplete (bool), autoComplete (bool), timeLimit (uint32), timeLimit2 (uint32), selectedMob (bool), summary (string), demandSummary (string), rewardSummary (string), startRequirements (RequirementsModel), endRequirements (RequirementsModel), startActions (ActionsModel), endActions (ActionsModel)
- `RequirementsModel` - Contains npcId (uint32), levelMin (uint16), levelMax (uint16), fameMin (int16), mesoMin (uint32), mesoMax (uint32), jobs ([]uint16), quests ([]QuestRequirementModel), items ([]ItemRequirementModel), mobs ([]MobRequirementModel), fieldEnter ([]uint32), pet ([]uint32), petTamenessMin (uint16), dayOfWeek ([]string), start (string), end (string), interval (uint32), startScript (string), endScript (string), infoNumber (uint32), normalAutoStart (bool), completionCount (uint32)
- `ActionsModel` - Contains npcId (uint32), exp (int32), money (int32), fame (int16), items ([]ItemRewardModel), skills ([]SkillRewardModel), nextQuest (uint32), buffItemId (uint32), interval (uint32), levelMin (uint16)
- `QuestRequirementModel` - Contains id (uint32), state (uint8)
- `ItemRequirementModel` - Contains id (uint32), count (int32)
- `MobRequirementModel` - Contains id (uint32), count (uint32)
- `ItemRewardModel` - Contains id (uint32), count (int32), job (int32), gender (int8), prop (int32), period (uint32), dateExpire (string), variable (uint32)
- `SkillRewardModel` - Contains id (uint32), level (int32), masterLevel (int32), jobs ([]uint16)

### Processors
- `Processor` (interface) - GetById(questId) fetches single quest. GetAll() fetches all quests. GetAutoStart() fetches quests with autoStart enabled. ByIdProvider(questId) returns quest model provider.

---

## Data/Cash

### Responsibility
Provides static cash item data from the DATA service.

### Core Models
- `RestModel` - Contains id (uint32), stateChangeItem (uint32), bgmPath (string). Resource type: "cash_items".

### Processors
- `Processor` - GetById(itemId) fetches cash item data via REST
