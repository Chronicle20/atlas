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
| worldId | byte | Selected world identifier |
| channelId | byte | Selected channel identifier |
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
| worldId | byte | World identifier |
| name | string | Character name |
| gender | byte | Character gender |
| skinColor | byte | Skin color |
| face | uint32 | Face identifier |
| hair | uint32 | Hair identifier |
| level | byte | Character level |
| jobId | uint16 | Job identifier |
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
| mapId | uint32 | Current map identifier |
| spawnPoint | uint32 | Spawn point identifier |
| gm | int | GM status |
| meso | uint32 | Currency amount |
| pets | []pet.Model | Active pets |
| equipment | equipment.Model | Equipped items |
| inventory | inventory.Model | Inventory contents |

### World

Represents world server data.

| Field | Type | Description |
|-------|------|-------------|
| id | byte | World identifier |
| name | string | World name |
| state | State | World state (Normal, Event, New, Hot) |
| message | string | World message |
| eventMessage | string | Event message |
| recommendedMessage | string | Recommendation message (non-empty indicates recommended) |
| capacityStatus | Status | Capacity status (Normal, HighlyPopulated, Full) |
| channels | []channel.Model | Available channels |

### Channel

Represents channel server data.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Channel identifier |
| worldId | byte | World identifier |
| channelId | byte | Channel number |
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

### Equipment

Represents equipped items mapped by slot type.

| Field | Type | Description |
|-------|------|-------------|
| slots | map[slot.Type]slot.Model | Equipment slots |

### Pet

Represents pet data.

| Field | Type | Description |
|-------|------|-------------|
| id | uint64 | Pet identifier |
| itemId | uint32 | Pet item identifier |

## Invariants

- Session model is immutable; state changes produce new instances via clone operations.
- Account model is immutable; modifications produce new instances via builder pattern.
- Session registry is keyed by tenant ID and session ID.
- Account registry tracks login status per tenant and account ID.
- A session timeout task monitors inactive sessions and destroys them.

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

### Account Session Processor

Manages account session commands via Kafka.

- Create: Emits a create session command.
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

### Character Processor

Retrieves character data via REST.

- IsValidName: Validates character name format and availability.
- ByAccountAndWorldProvider: Provides characters by account and world.
- GetForWorld: Retrieves characters for an account in a world.
- ByNameProvider: Provides characters by name.
- GetByName: Retrieves characters by name.
- ByIdProvider: Provides character by ID.
- GetById: Retrieves character by ID.
- InventoryDecorator: Decorates character with inventory data.
- DeleteById: Deletes a character via REST.

### Character Factory Processor

Creates new characters via REST.

- SeedCharacter: Initiates character creation with specified attributes.

### World Processor

Retrieves world data via REST.

- GetAll: Retrieves all worlds.
- AllProvider: Provides all worlds.
- GetById: Retrieves world by ID.
- ByIdModelProvider: Provides world by ID.
- GetCapacityStatus: Retrieves world capacity status.

### Channel Processor

Retrieves channel data via REST.

- ByIdModelProvider: Provides channel by world and channel ID.
- GetById: Retrieves channel by world and channel ID.
- ByWorldModelProvider: Provides channels for a world.
- GetForWorld: Retrieves channels for a world.
- GetRandomInWorld: Retrieves a random channel in a world.

### Inventory Processor

Retrieves inventory data via REST.

- ByCharacterIdProvider: Provides inventory by character ID.
- GetByCharacterId: Retrieves inventory by character ID.
