# Domain

## Messenger

### Responsibility

Manages ephemeral group chat rooms for up to 3 characters.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Messenger identifier |
| members | []MemberModel | List of members |

#### MemberModel

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| slot | byte | Position in messenger (0-2) |

### Invariants

- Maximum 3 members per messenger
- Member slots are assigned sequentially starting from 0
- Messenger IDs start at 1,000,000,000 and increment per tenant

### State Transitions

| Transition | Description |
|------------|-------------|
| AddMember | Adds a member to the next available slot |
| RemoveMember | Removes a member from the messenger |
| FirstOpenSlot | Finds the lowest unused slot number |
| FindMember | Locates a member by character ID |

### Processors

#### Create

Creates a new messenger with the requesting character as the first member.

- Validates character is not already in a messenger
- Creates messenger with auto-incremented ID
- Adds character as first member in slot 0

#### Join

Adds a character to an existing messenger.

- Validates character is not already in a messenger
- Validates messenger exists
- Validates messenger is not at capacity
- Adds character to next available slot

#### Leave

Removes a character from a messenger.

- Validates character is in the specified messenger
- Removes character from messenger
- Disbands messenger if no members remain

#### RequestInvite

Initiates an invitation to another character.

- Creates a messenger for the actor if not in one
- Validates target character is not already in a messenger
- Validates messenger is not at capacity
- Produces invite command to invite service

---

## Character

### Responsibility

Maintains an in-memory registry of character state for messenger membership tracking.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Character identifier |
| name | string | Character name |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| messengerId | uint32 | Current messenger ID (0 if none) |
| online | bool | Online status |

#### ForeignModel

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| worldId | world.Id | World identifier |
| mapId | map.Id | Map identifier |
| name | string | Character name |
| level | byte | Character level |
| jobId | uint16 | Job identifier |
| gm | int | GM level |

### Invariants

- Character ID is required
- Character name is required

### State Transitions

| Transition | Description |
|------------|-------------|
| JoinMessenger | Sets messenger ID |
| LeaveMessenger | Clears messenger ID to 0 |
| ChangeChannel | Updates channel ID |
| Login | Sets online to true |
| Logout | Sets online to false |

### Processors

#### Login

Processes character login events.

- Creates character in registry if not present (fetches from foreign service)
- Updates online status and channel
- Emits member login event if character is in a messenger

#### Logout

Processes character logout events.

- Updates character to offline in registry
- Emits member logout event if character is in a messenger

#### ChannelChange

Processes channel change events.

- Updates character channel in registry

#### JoinMessenger

Updates character to be in a messenger.

#### LeaveMessenger

Updates character to no longer be in a messenger.

---

## Invite

### Responsibility

Produces invite commands to the invite service for messenger invitations.

### Processors

#### Create

Produces an invite command to invite a character to a messenger.
