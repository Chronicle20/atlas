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
- a registry entry is unregistered if not re-registered within 15 seconds of its createdAt timestamp

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

---

## Broadcast

### Responsibility

Represents a per (tenant, world, family) serialized queue of Maple TV and avatar-megaphone broadcast requests, activating one entry at a time and expiring it after its duration elapses.

### Core Models

#### Family

| Value | Description |
|-------|-------------|
| TV | Maple TV broadcast family |
| AVATAR | Avatar megaphone broadcast family |

#### Payload

| Field | Type | Description |
|-------|------|-------------|
| channelId | byte | Channel identifier |
| senderName | string | Sender display name |
| senderMedal | string | Sender medal name |
| messages | []string | Broadcast message lines |
| whispersOn | bool | Whether whispers are enabled during the broadcast |
| itemId | uint32 | Item identifier associated with the broadcast |
| tvMessageType | string | Semantic message type key (NORMAL, STAR, HEART); resolved to a client wire byte at the packet layer, never carried as a byte in the domain |
| senderLook | sharedsaga.AvatarSnapshot | Sender avatar appearance snapshot |
| receiverName | string | Receiver display name |
| receiverLook | sharedsaga.AvatarSnapshot | Receiver avatar appearance snapshot (nullable) |

#### Entry

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Unique identifier |
| characterId | uint32 | Character identifier of the requester |
| payload | Payload | Render payload |
| durationSeconds | uint32 | Duration, in seconds, the entry stays active once activated |
| activatedAt | time.Time | Timestamp the entry was activated |
| expiresAt | time.Time | Timestamp the entry expires (activatedAt + durationSeconds) |

#### QueueModel

| Field | Type | Description |
|-------|------|-------------|
| active | *Entry | The currently active entry, or nil if the queue is idle |
| pending | []Entry | Entries waiting behind the active entry, in FIFO order |

### Invariants

- family must be one of Family TV ("TV") or Family Avatar ("AVATAR")
- a queue holds at most one active entry at a time
- an active entry is considered expired when the current time is not before its expiresAt (the boundary is inclusive)
- each (tenant, world, family) queue is mutated only through compare-and-swap; a mutation function must be side-effect free since it may be re-applied on contention

### State Transitions

| Transition | Description |
|------------|-------------|
| Append | Adds an entry to the tail of pending, preserving existing order |
| ActivateNext | Pops the head of pending into active, stamping activatedAt=now and expiresAt=now+durationSeconds; no-op if pending is empty |
| ClearActive | Removes the active entry, leaving pending untouched |

### Processors

#### Processor

| Method | Description |
|--------|-------------|
| Enqueue | Appends an entry to the (worldId, family) queue; if the queue was idle the entry activates immediately (STARTED emitted in addition to QUEUED with waitSeconds 0), otherwise only QUEUED is emitted with the wait computed before the append |
| GetQueue | Returns the current QueueModel for a (worldId, family) queue |
| SweepTenant | Expires active entries past their deadline and promotes each queue's next pending entry, for every (worldId, family) queue belonging to the tenant |
