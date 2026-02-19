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
- `Processor` - Retrieves character by ID or name via REST (CHARACTERS service). Supports decorators: InventoryDecorator, PetAssetEnrichmentDecorator, SkillModelDecorator, QuestModelDecorator. Provides GetEquipableInSlot (looks up equip compartment by slot) and GetItemInSlot (looks up any compartment by type and slot). Issues commands for AP distribution, SP distribution, meso drop, HP/MP changes.
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
Represents spawned monsters in a map. Provides monster data retrieval, damage application, skill usage, and status effect management.

### Core Models
- `Model` - Contains field (field.Model), uniqueId (uint32), monsterId (uint32), maxHp (uint32), hp (uint32), mp (uint32), controlCharacterId (uint32), x (int16), y (int16), fh (int16), stance (byte), team (int8). Delegates WorldId(), ChannelId(), MapId(), Instance() to embedded field.Model. Controlled() returns true when controlCharacterId != 0.
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
- `Processor` (interface) - Enter(f, portalName, characterId) looks up portal by name in map data and emits ENTER command. Warp(f, characterId, targetMapId) emits WARP command with target map ID.

---

## Fame

### Responsibility
Handles fame change requests between characters.

### Processors
- `Processor` - RequestChange(f, characterId, targetId, amount) emits fame change command

---

## Consumable

### Responsibility
Handles item consumption and scroll use requests.

### Processors
- `Processor` - RequestItemConsume(f, characterId, itemId, source, updateTime) emits item consume command. RequestScrollUse(f, characterId, scrollSlot, equipSlot, whiteScroll, legendarySpirit, updateTime) emits scroll use command.

---

## Invite

### Responsibility
Handles invite accept and reject operations for party and guild invitations. Invites are world-scoped, not field-scoped.

### Processors
- `Processor` - Accept(actorId, worldId, inviteType, referenceId) emits accept invite command. Reject(actorId, worldId, inviteType, originatorId) emits reject invite command.

---

## Character Expression

### Responsibility
Handles character expression (emote) changes.

### Processors
- `Processor` (interface) - Change(characterId, f, expression) emits expression command

---

## Message

### Responsibility
Handles chat message production across multiple chat types. Provides type-specific methods that delegate to the appropriate Kafka command structure.

### Processors
- `Processor` (interface) - GeneralChat (field-scoped, with balloonOnly flag), BuddyChat/PartyChat/GuildChat/AllianceChat (delegate to MultiChat with type string), MultiChat (with recipients list), WhisperChat (with recipientName), MessengerChat (with recipients list), PetChat (with ownerId, petSlot, type, action, balloon)

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
- `Model` - Contains id (uint32), name (string), trunkPut (int32), trunkGet (int32), storebank (bool)

### Processors
- `Processor` (interface) - GetById(npcId) fetches NPC template via REST

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
