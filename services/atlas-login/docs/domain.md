# Login

## Responsibility

The login domain manages client authentication and session state for the game login server. It coordinates the login flow, tracks active sessions in-memory, and maintains account login status via an in-memory registry.

## Core Models

### Session

Represents an active client connection to the login server.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Session identifier |
| accountId | uint32 | Associated account identifier |
| ch | channel.Model | Selected channel (carries worldId and channelId) |
| con | net.Conn | TCP connection |
| send | crypto.AESOFB | Send encryption cipher |
| sendLock | *sync.Mutex | Mutex for send operations |
| recv | crypto.AESOFB | Receive encryption cipher |
| encryptFunc | crypto.EncryptFunc | Encryption function |
| lastPacket | time.Time | Timestamp of last packet received |
| locale | byte | Client locale identifier |

### Account

Represents account data retrieved from the account service.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Account identifier |
| name | string | Account name |
| password | string | Account password |
| pin | string | Account PIN |
| pic | string | Account PIC (secondary password) |
| pinAttempts | int | Number of PIN verification attempts |
| picAttempts | int | Number of PIC verification attempts |
| loggedIn | int | Login state |
| lastLogin | uint64 | Last login timestamp |
| gender | byte | Account gender |
| banned | bool | Ban status |
| tos | bool | Terms of service acceptance |
| language | string | Account language |
| country | string | Account country |
| characterSlots | int16 | Number of character slots |

### Character

Represents character data retrieved from the character service.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| accountId | uint32 | Associated account identifier |
| worldId | world.Id | World identifier |
| name | string | Character name |
| gender | byte | Character gender |
| skinColor | byte | Skin color |
| face | uint32 | Face identifier |
| hair | uint32 | Hair identifier |
| level | byte | Character level |
| jobId | job.Id | Job identifier |
| strength | uint16 | Strength stat |
| dexterity | uint16 | Dexterity stat |
| intelligence | uint16 | Intelligence stat |
| luck | uint16 | Luck stat |
| hp | uint16 | Current HP |
| maxHp | uint16 | Maximum HP |
| mp | uint16 | Current MP |
| maxMp | uint16 | Maximum MP |
| hpMpUsed | int | HP/MP used count |
| ap | uint16 | Available ability points |
| sp | string | Skill points (comma-separated) |
| experience | uint32 | Experience points |
| fame | int16 | Fame points |
| gachaponExperience | uint32 | Gachapon experience points |
| mapId | _map.Id | Current map identifier |
| spawnPoint | uint32 | Spawn point identifier |
| gm | int | GM status |
| meso | uint32 | Currency amount |
| pets | []pet.Model | Active pets |
| equipment | equipment.Model | Equipped items (derived from inventory) |
| inventory | inventory.Model | Inventory contents |

Character has a `SetInventory` method that processes the raw inventory response. It separates the equipable compartment into positively-slotted items (kept in the equip compartment) and negatively-slotted items (placed into equipment slots by position). Cash equipment (slot < -100) and normal equipment are distinguished and placed into the appropriate `Equipable`/`CashEquipable` fields of each equipment slot. During this process, assets placed into equipment slots have their compartment ID set to `uuid.Nil` via `asset.Clone(a).SetCompartmentId(uuid.Nil).Build()`.

Character supports SP table logic for Evan job advancement (jobs 2001, 2200-2218), where the skill book index is derived from the job ID.

Character uses a builder pattern (`NewBuilder`/`ToBuilder`/`Builder.Build`) for immutable construction and modification.

### World

Represents world server data.

| Field | Type | Description |
|-------|------|-------------|
| id | world.Id | World identifier |
| name | string | World name |
| state | State | World state (Normal=0, Event=1, New=2, Hot=3) |
| message | string | World message |
| eventMessage | string | Event message |
| recommendedMessage | string | Recommendation message (non-empty indicates recommended) |
| capacityStatus | Status | Capacity status (Normal=0, HighlyPopulated=1, Full=2) |
| channels | []channel.Model | Available channels |

### Channel

Represents channel server data.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Channel identifier |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel number |
| ipAddress | string | Server IP address |
| port | int | Server port |
| currentCapacity | uint32 | Current player count |
| maxCapacity | uint32 | Maximum player capacity |
| createdAt | time.Time | Creation timestamp |

### Inventory

Represents character inventory data.

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| compartments | map[inventory.Type]compartment.Model | Inventory compartments by type |

Compartment types are: Equip, Use, Setup, ETC, Cash.

Inventory supports lookup by type (`CompartmentByType`) and by UUID (`CompartmentById`).

Inventory uses a builder pattern (`NewBuilder`/`Clone`/`ModelBuilder.Build`) for construction and modification. The builder supports setting compartments by type (`SetEquipable`, `SetConsumable`, `SetSetup`, `SetEtc`, `SetCash`) or generically via `SetCompartment`. `BuilderSupplier` provides a model.Provider for a new builder, and `FoldCompartment` folds a compartment into a builder.

### Compartment

Represents a single inventory compartment (tab).

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Compartment identifier |
| characterId | uint32 | Character identifier |
| inventoryType | inventory.Type | Inventory type (Equip, Use, Setup, ETC, Cash) |
| capacity | uint32 | Maximum number of asset slots |
| assets | []asset.Model | Assets contained in this compartment |

Compartment supports lookup by slot (`FindBySlot`) and by template ID (`FindFirstByItemId`).

Compartment uses a builder pattern (`NewBuilder`/`Clone`/`ModelBuilder.Build`). The builder supports `SetCapacity`, `AddAsset`, and `SetAssets`.

### Asset

Represents a unified inventory item. All item types (equipment, stackable, cash, pet) share a single model with type-specific fields.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Asset identifier |
| compartmentId | uuid.UUID | Owning compartment identifier |
| slot | int16 | Slot position within compartment |
| templateId | uint32 | Item template identifier |
| expiration | time.Time | Expiration timestamp |
| createdAt | time.Time | Creation timestamp |
| quantity | uint32 | Stack quantity (stackable/cash items) |
| ownerId | uint32 | Owner identifier |
| flag | uint16 | Item flags |
| rechargeable | uint64 | Rechargeable amount |
| strength | uint16 | STR bonus (equipment) |
| dexterity | uint16 | DEX bonus (equipment) |
| intelligence | uint16 | INT bonus (equipment) |
| luck | uint16 | LUK bonus (equipment) |
| hp | uint16 | HP bonus (equipment) |
| mp | uint16 | MP bonus (equipment) |
| weaponAttack | uint16 | Weapon attack (equipment) |
| magicAttack | uint16 | Magic attack (equipment) |
| weaponDefense | uint16 | Weapon defense (equipment) |
| magicDefense | uint16 | Magic defense (equipment) |
| accuracy | uint16 | Accuracy (equipment) |
| avoidability | uint16 | Avoidability (equipment) |
| hands | uint16 | Hands (equipment) |
| speed | uint16 | Speed (equipment) |
| jump | uint16 | Jump (equipment) |
| slots | uint16 | Upgrade slots remaining (equipment) |
| locked | bool | Lock status (equipment) |
| spikes | bool | Spikes flag (equipment) |
| karmaUsed | bool | Karma used flag (equipment) |
| cold | bool | Cold protection (equipment) |
| canBeTraded | bool | Tradeable flag (equipment) |
| levelType | byte | Level type (equipment) |
| level | byte | Equipment level (equipment) |
| experience | uint32 | Equipment experience (equipment) |
| hammersApplied | uint32 | Hammers applied count (equipment) |
| equippedSince | *time.Time | Timestamp when equipped (equipment) |
| cashId | int64 | Cash shop item identifier (cash) |
| commodityId | uint32 | Commodity identifier (cash) |
| purchaseBy | uint32 | Purchaser identifier (cash) |
| petId | uint32 | Pet identifier (pet reference) |

Asset provides type-classification methods: `IsEquipment`, `IsCashEquipment`, `IsConsumable`, `IsSetup`, `IsEtc`, `IsCash`, `IsPet`, `IsStackable`, `HasQuantity`. The `InventoryType` is derived from the `templateId` at runtime via `inventory.TypeFromItemId`.

The `Quantity` method returns the `quantity` field for stackable and non-pet cash items, and returns 1 for all other item types.

Asset uses a builder pattern (`NewBuilder`/`Clone`/`ModelBuilder.Build`). `NewBuilder` takes `compartmentId` and `templateId` as required parameters. `Clone` creates a builder initialized from an existing model. The builder provides setter methods for all fields.

### Equipment

Represents equipped items mapped by slot type.

| Field | Type | Description |
|-------|------|-------------|
| slots | map[slot.Type]slot.Model | Equipment slots |

Each slot contains a `Position`, an optional `Equipable` (normal equipment), and an optional `CashEquipable` (cash equipment).

### Equipment Slot

Represents a single equipment slot.

| Field | Type | Description |
|-------|------|-------------|
| Position | slot.Position | Slot position |
| Equipable | *asset.Model | Normal equipment in this slot (nil if empty) |
| CashEquipable | *asset.Model | Cash equipment in this slot (nil if empty) |

### Pet

Represents pet data.

| Field | Type | Description |
|-------|------|-------------|
| id | uint64 | Pet identifier |
| itemId | uint32 | Pet item identifier |

### Guild

Represents guild data retrieved from the guild service.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Guild identifier |
| leaderId | uint32 | Leader character identifier |
| members | []member.Model | Guild members |

Guild provides `IsLeader` to check if a character is the guild leader. Returns false if the guild ID or character ID is zero.

### Guild Member

Represents a guild member.

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |

### Channel Load

Represents channel capacity data for display in the server list.

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel number |
| capacity | uint32 | Current player count |

### Channel Select

Represents channel connection information returned after character selection.

| Field | Type | Description |
|-------|------|-------------|
| IPAddress | string | Channel server IP address |
| Port | uint16 | Channel server port |
| CharacterId | uint32 | Selected character identifier |

### Recommendation

Represents a world recommendation entry.

| Field | Type | Description |
|-------|------|-------------|
| worldId | world.Id | World identifier |
| reason | string | Recommendation reason text |

## Invariants

- Session model is immutable; state changes produce new instances via clone operations.
- Account model is immutable; modifications produce new instances via builder pattern.
- Character model is immutable; modifications produce new instances via builder pattern.
- Inventory, Compartment, and Asset models are immutable; modifications produce new instances via builder pattern.
- Session registry is keyed by tenant ID and session ID.
- Account registry tracks login status per tenant and account ID.
- A session timeout task monitors inactive sessions and destroys them.
- Asset model is unified across all item types. The inventory type is derived from the template ID, not stored explicitly.
- Equipment slots are populated by processing equip-compartment assets with negative slot values during `SetInventory`.

## Processors

### Session Processor

Manages session lifecycle within the login server.

- AllInTenantProvider: Retrieves all sessions for a tenant.
- ByIdModelProvider: Retrieves a session by session ID.
- IfPresentById: Executes an operator if session exists.
- ByAccountIdModelProvider: Retrieves a session by account ID.
- IfPresentByAccountId: Executes an operator if session exists for account.
- SetAccountId: Associates an account with a session.
- UpdateLastRequest: Updates the last packet timestamp.
- SetWorldId: Sets the selected world for a session.
- SetChannelId: Sets the selected channel for a session.
- SessionCreated: Emits session created event.
- Create: Creates a new session for an incoming connection.
- DestroyByIdWithSpan: Destroys a session with tracing.
- DestroyById: Destroys a session by ID.
- Destroy: Destroys a session and emits destroyed event.
- Decrypt: Returns a decryption function for incoming packets.
- WithContext: Creates a new processor with an updated context.

### Account Session Processor

Manages account session commands via Kafka.

- Create: Emits a create session command with account name, password, IP address, and HWID.
- Destroy: Emits a logout command.
- UpdateState: Emits a progress state command.

### Account Processor

Retrieves and manages account data via REST.

- ForAccountByName: Executes an operator for an account by name.
- ForAccountById: Executes an operator for an account by ID.
- ByNameModelProvider: Provides account by name.
- ByIdModelProvider: Provides account by ID.
- AllProvider: Provides all accounts.
- GetById: Retrieves account by ID.
- GetByName: Retrieves account by name.
- IsLoggedIn: Checks if account is logged in via registry.
- InitializeRegistry: Initializes account login status registry.
- UpdatePin: Updates account PIN via REST.
- UpdatePic: Updates account PIC via REST.
- UpdateTos: Updates account TOS acceptance via REST.
- UpdateGender: Updates account gender via REST.
- RecordPinAttempt: Records a PIN verification attempt and returns the attempt count and whether the limit was reached.
- RecordPicAttempt: Records a PIC verification attempt and returns the attempt count and whether the limit was reached.

### Character Processor

Retrieves character data via REST.

- IsValidName: Validates character name format (3-12 alphanumeric or CJK characters) and availability.
- ByAccountAndWorldProvider: Provides characters by account and world.
- GetForWorld: Retrieves characters for an account in a world.
- ByNameProvider: Provides characters by name.
- GetByName: Retrieves characters by name.
- ByIdProvider: Provides character by ID.
- GetById: Retrieves character by ID.
- InventoryDecorator: Decorates character with inventory data by fetching from inventory service and calling SetInventory.
- DeleteById: Deletes a character via REST.

### Character Factory Processor

Creates new characters via REST.

- SeedCharacter: Initiates character creation with specified attributes (account, world, name, job, appearance, stats).

### World Processor

Retrieves world data via REST.

- GetAll: Retrieves all worlds.
- AllProvider: Provides all worlds.
- GetById: Retrieves world by ID.
- ByIdModelProvider: Provides world by ID.
- GetCapacityStatus: Retrieves world capacity status. Returns Full if the world cannot be retrieved.

### Channel Processor

Retrieves channel data via REST.

- ByIdModelProvider: Provides channel by world and channel ID.
- GetById: Retrieves channel by world and channel ID.
- ByWorldModelProvider: Provides channels for a world.
- GetForWorld: Retrieves channels for a world.
- GetRandomInWorld: Selects a random channel from all channels in a world.

### Inventory Processor

Retrieves inventory data via REST.

- ByCharacterIdProvider: Provides inventory by character ID.
- GetByCharacterId: Retrieves inventory by character ID.

### Guild Processor

Retrieves guild data via REST.

- GetByMemberId: Retrieves a guild by member character ID.
- ByMemberIdProvider: Provides guilds filtered by member ID.
- IsGuildMaster: Checks if a character is the guild master. Returns false if the character has no guild.
