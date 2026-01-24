# Domain Documentation

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

## Character

### Responsibility
Represents a player character with stats, equipment, inventory, skills, and pets. Provides character data retrieval and stat modification commands.

### Core Models
- `Model` - Contains id, accountId, worldId, name, gender, skinColor, face, hair, level, jobId, stats (str/dex/int/luk/hp/mp), ap, sp, experience, fame, mapId, position, meso, pets, equipment, inventory, skills

### Invariants
- Character must belong to an account
- Character must have valid stat values

### Processors
- `Processor` - Retrieves character by ID or name, decorates with inventory/skills/pets, requests stat distribution, issues drop meso and HP/MP change commands

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

## Storage

### Responsibility
Handles storage (warehouse) operations for depositing and withdrawing items.

### Processors
- `Processor` - Manages storage interaction with NPCs

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
- StateNotStarted → StateStarted → StateCompleted
- StateStarted → StateNotStarted (forfeit)

### Processors
- `Processor` - Retrieves quests by character ID, issues start/complete/forfeit/restore item commands

---

## Saga

### Responsibility
Handles distributed transaction orchestration for multi-step operations.

### Processors
- `Processor` - Manages saga state and compensation
