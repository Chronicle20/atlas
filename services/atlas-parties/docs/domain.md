# Domain

## Party

### Responsibility

Represents a group of characters with a designated leader. Manages membership operations and leader assignment.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Party identifier |
| leaderId | uint32 | Character ID of party leader |
| members | []uint32 | List of character IDs in party |

#### Builder

Fluent builder for constructing party models with validation.

### Invariants

- Party ID must be greater than 0
- Leader ID must be greater than 0
- Leader must be a member of the party when members are provided
- No duplicate member IDs
- Member IDs must be greater than 0
- Maximum party size is 6 members
- Beginners cannot create parties
- GMs cannot create parties
- Character must not already be in a party to create or join one

### State Transitions

- AddMember: Appends character to members list
- RemoveMember: Removes character from members list
- SetLeader: Assigns specific character as leader
- ElectLeader: Randomly selects leader from current members

### Processors

#### PartyProcessor

| Method | Description |
|--------|-------------|
| AllProvider | Returns all parties for tenant |
| ByIdProvider | Returns party by ID |
| GetSlice | Returns parties matching filters |
| GetById | Returns single party by ID |
| GetByCharacter | Returns party containing character |
| ByCharacterProvider | Provider for character-to-party lookup |
| Create | Creates new party with leader |
| CreateAndEmit | Creates party and emits status event |
| Join | Adds character to party |
| JoinAndEmit | Joins party and emits status event |
| Expel | Removes character from party (forced) |
| ExpelAndEmit | Expels and emits status event |
| Leave | Character leaves party voluntarily |
| LeaveAndEmit | Leaves and emits status event |
| ChangeLeader | Transfers leadership to another member |
| ChangeLeaderAndEmit | Changes leader and emits status event |
| RequestInvite | Creates party invitation for target character |
| RequestInviteAndEmit | Requests invite and emits event |

---

## Character

### Responsibility

Maintains local cache of character state relevant to party operations. Tracks party membership, location, and status.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Character identifier |
| name | string | Character name |
| level | byte | Character level |
| jobId | job.Id | Character job identifier |
| field | field.Model | Character location (worldId, channelId, mapId, instance) |
| partyId | uint32 | Current party identifier (0 if none) |
| online | bool | Online status |
| gm | int | GM level |

#### ForeignModel

Subset of character data retrieved from external character service.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Character identifier |
| worldId | world.Id | World identifier |
| mapId | _map.Id | Map identifier |
| name | string | Character name |
| level | byte | Character level |
| jobId | job.Id | Job identifier |
| gm | int | GM level |

### Invariants

- Character can belong to at most one party at a time

### State Transitions

- Login: Sets online to true
- Logout: Sets online to false
- JoinParty: Sets partyId to party identifier
- LeaveParty: Sets partyId to 0
- ChangeMap: Updates mapId
- ChangeChannel: Updates channelId
- ChangeLevel: Updates level
- ChangeJob: Updates jobId

### Processors

#### CharacterProcessor

| Method | Description |
|--------|-------------|
| LoginAndEmit | Processes login and emits member status event |
| Login | Updates character to online state |
| LogoutAndEmit | Processes logout and emits member status event |
| Logout | Updates character to offline state |
| ChannelChange | Updates character channel |
| LevelChangeAndEmit | Updates level and emits event if in party |
| LevelChange | Updates character level |
| JobChangeAndEmit | Updates job and emits event if in party |
| JobChange | Updates character job |
| MapChange | Updates character map |
| JoinParty | Associates character with party |
| LeaveParty | Disassociates character from party |
| Delete | Removes character from registry |
| ByIdProvider | Returns character by ID |
| GetById | Returns single character by ID |
| GetForeignCharacterInfo | Retrieves character data from external service |

---

## Invite

### Responsibility

Coordinates party invitation flow by producing invite commands.

### Processors

#### InviteProcessor

| Method | Description |
|--------|-------------|
| Create | Produces invite command for target character |
