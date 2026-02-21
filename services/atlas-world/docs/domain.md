# Domain

## Channel

### Responsibility

Represents an active channel server instance within a world.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Unique identifier |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier within world |
| ipAddress | string | Server IP address |
| port | int | Server port |
| currentCapacity | uint32 | Current player count |
| maxCapacity | uint32 | Maximum player capacity |
| createdAt | time.Time | Registration timestamp |
| expRate | float64 | Experience rate multiplier (defaults to 1.0) |
| mesoRate | float64 | Meso rate multiplier (defaults to 1.0) |
| itemDropRate | float64 | Item drop rate multiplier (defaults to 1.0) |
| questExpRate | float64 | Quest experience rate multiplier (defaults to 1.0) |

### Invariants

- id must not be nil
- ipAddress must not be empty
- port must be between 1 and 65535
- maxCapacity must be greater than 0

### Processors

#### Processor

| Method | Description |
|--------|-------------|
| AllProvider | Returns all channel servers for the tenant |
| GetByWorld | Returns channels for a specific world |
| ByWorldProvider | Provider for channels filtered by world |
| GetById | Returns a specific channel by world and channel ID |
| ByIdProvider | Provider for a specific channel |
| Register | Registers a new channel server in the registry |
| Unregister | Removes a channel server from the registry |
| RequestStatus | Buffers a status request command |
| RequestStatusAndEmit | Emits a status request command |
| EmitStarted | Returns function to buffer a started event |
| EmitStartedAndEmit | Emits a channel started event |

---

## World

### Responsibility

Represents a game world that contains multiple channel servers.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| id | byte | World identifier |
| name | string | World name |
| state | State | World state flag |
| message | string | Server message |
| eventMessage | string | Event message |
| recommendedMessage | string | Recommendation message |
| capacityStatus | Status | Capacity status indicator |
| channels | []channel.Model | Associated channel servers |
| expRate | float64 | Experience rate multiplier (defaults to 1.0) |
| mesoRate | float64 | Meso rate multiplier (defaults to 1.0) |
| itemDropRate | float64 | Item drop rate multiplier (defaults to 1.0) |
| questExpRate | float64 | Quest experience rate multiplier (defaults to 1.0) |

#### State

| Value | Name | Description |
|-------|------|-------------|
| 0 | StateNormal | Normal state |
| 1 | StateEvent | Event state |
| 2 | StateNew | New world state |
| 3 | StateHot | Hot world state |

#### Status

| Value | Name | Description |
|-------|------|-------------|
| 0 | StatusNormal | Normal capacity |
| 1 | StatusHighlyPopulated | Highly populated |
| 2 | StatusFull | Full capacity |

### Invariants

- name must not be empty

### Processors

#### Processor

| Method | Description |
|--------|-------------|
| ChannelDecorator | Decorates a world with its channels |
| GetWorlds | Returns all worlds for the tenant |
| AllWorldProvider | Provider for all worlds |
| GetWorld | Returns a specific world by ID |
| ByWorldIdProvider | Provider for a specific world |

---

## Rate

### Responsibility

Represents per-world rate multipliers for experience, meso, item drop, and quest experience.

### Core Models

#### Model

| Field | Type | Description |
|-------|------|-------------|
| expRate | float64 | Experience rate multiplier (defaults to 1.0) |
| mesoRate | float64 | Meso rate multiplier (defaults to 1.0) |
| itemDropRate | float64 | Item drop rate multiplier (defaults to 1.0) |
| questExpRate | float64 | Quest experience rate multiplier (defaults to 1.0) |

#### Type

| Value | Description |
|-------|-------------|
| exp | Experience rate |
| meso | Meso rate |
| item_drop | Item drop rate |
| quest_exp | Quest experience rate |

### Processors

#### Processor

| Method | Description |
|--------|-------------|
| GetWorldRates | Returns current rate multipliers for a world |
| UpdateWorldRate | Updates a rate multiplier for a world and emits a rate changed event |
