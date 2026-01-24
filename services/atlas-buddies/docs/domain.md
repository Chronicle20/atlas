# Domain

## List

### Responsibility
Represents a character's buddy list containing buddies and a configurable capacity.

### Core Models

#### Model
- `tenantId` (uuid.UUID): Tenant identifier for multi-tenancy
- `id` (uuid.UUID): Unique identifier for the buddy list
- `characterId` (uint32): Character that owns this buddy list
- `capacity` (byte): Maximum number of buddies allowed
- `buddies` ([]buddy.Model): Collection of buddies in the list

### Invariants
- A character has exactly one buddy list
- Buddy count cannot exceed capacity
- Capacity can only be increased, not decreased

### Processors

#### Processor
- `ByCharacterIdProvider`: Provides a buddy list by character ID
- `GetByCharacterId`: Retrieves a buddy list for a character
- `Create`: Creates a new buddy list with specified capacity
- `Delete`: Deletes a character's buddy list and removes them from all buddies' lists
- `RequestAddBuddy`: Initiates adding a buddy (handles mutual buddy logic and invites)
- `RequestDeleteBuddy`: Initiates removing a buddy or rejecting an invite
- `AcceptInvite`: Accepts a buddy invite and creates mutual buddy relationship
- `DeleteBuddy`: Removes a buddy from a character's list
- `UpdateBuddyChannel`: Updates channel information for a character across all buddy lists
- `UpdateBuddyShopStatus`: Updates shop status for a character across all buddy lists
- `IncreaseCapacity`: Increases buddy list capacity (validates new capacity > current)
- `IncreaseCapacityWithTransaction`: Increases capacity with transaction ID for saga coordination

---

## Buddy

### Responsibility
Represents a single buddy entry within a buddy list.

### Core Models

#### Model
- `listId` (uuid.UUID): Reference to the parent buddy list
- `characterId` (uint32): Character ID of the buddy
- `group` (string): Group categorization for the buddy
- `characterName` (string): Display name of the buddy
- `channelId` (int8): Current channel of the buddy (-1 if offline)
- `inShop` (bool): Whether the buddy is in the cash shop
- `pending` (bool): Whether the buddy relationship is pending acceptance

### Invariants
- A buddy entry belongs to exactly one list
- Channel ID of -1 indicates offline status

---

## Character

### Responsibility
Represents external character information retrieved from the character service.

### Core Models

#### Model
- `id` (uint32): Character identifier
- `name` (string): Character name
- `gm` (int): Game master level (0 for regular players)

### Invariants
- GM level > 0 indicates the character is a game master

### Processors

#### Processor
- `GetById`: Retrieves character information by ID from the external character service

---

## Invite

### Responsibility
Manages buddy invite operations through Kafka commands to an external invite service.

### Processors

#### Processor
- `Create`: Creates a buddy invite for a target character
- `Reject`: Rejects a pending buddy invite
