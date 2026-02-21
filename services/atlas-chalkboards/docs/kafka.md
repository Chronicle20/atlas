# Kafka Integration

## Topics Consumed

| Topic Environment Variable | Consumer Group | Description |
|---------------------------|----------------|-------------|
| COMMAND_TOPIC_CHALKBOARD | Chalkboard Service | Chalkboard commands |
| EVENT_TOPIC_CHARACTER_STATUS | Chalkboard Service | Character status events |

## Topics Produced

| Topic Environment Variable | Description |
|---------------------------|-------------|
| EVENT_TOPIC_CHALKBOARD_STATUS | Chalkboard status events |

## Message Types

### Commands

#### Command (Chalkboard)

Consumed from COMMAND_TOPIC_CHALKBOARD. Generic envelope with typed body.

| Field | Type |
|-------|------|
| TransactionId | uuid.UUID |
| WorldId | world.Id |
| ChannelId | channel.Id |
| MapId | map.Id |
| Instance | uuid.UUID |
| CharacterId | uint32 |
| Type | string |
| Body | varies |

##### Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| SET | SetCommandBody | Set chalkboard message |
| CLEAR | ClearCommandBody | Clear chalkboard message |

##### SetCommandBody

| Field | Type |
|-------|------|
| Message | string |

##### ClearCommandBody

Empty body.

### Events Consumed

#### StatusEvent (Character)

Consumed from EVENT_TOPIC_CHARACTER_STATUS.

| Field | Type |
|-------|------|
| TransactionId | uuid.UUID |
| WorldId | world.Id |
| CharacterId | uint32 |
| Type | string |
| Body | varies |

##### Character Status Event Types

| Type | Body Type | Description |
|------|-----------|-------------|
| LOGIN | StatusEventLoginBody | Character logged in |
| LOGOUT | StatusEventLogoutBody | Character logged out |
| MAP_CHANGED | StatusEventMapChangedBody | Character changed maps |
| CHANNEL_CHANGED | ChangeChannelEventLoginBody | Character changed channels |

##### StatusEventLoginBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| MapId | map.Id |
| Instance | uuid.UUID |

##### StatusEventLogoutBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| MapId | map.Id |
| Instance | uuid.UUID |

##### StatusEventMapChangedBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| OldMapId | map.Id |
| OldInstance | uuid.UUID |
| TargetMapId | map.Id |
| TargetInstance | uuid.UUID |
| TargetPortalId | uint32 |

##### ChangeChannelEventLoginBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| OldChannelId | channel.Id |
| MapId | map.Id |
| Instance | uuid.UUID |

### Events Produced

#### StatusEvent (Chalkboard)

Produced to EVENT_TOPIC_CHALKBOARD_STATUS.

| Field | Type |
|-------|------|
| TransactionId | uuid.UUID |
| WorldId | world.Id |
| ChannelId | channel.Id |
| MapId | map.Id |
| Instance | uuid.UUID |
| CharacterId | uint32 |
| Type | string |
| Body | varies |

##### Chalkboard Status Event Types

| Type | Body Type | Description |
|------|-----------|-------------|
| SET | SetStatusEventBody | Chalkboard message set |
| CLEAR | ClearStatusEventBody | Chalkboard message cleared |

##### SetStatusEventBody

| Field | Type |
|-------|------|
| Message | string |

##### ClearStatusEventBody

Empty body.

## Transaction Semantics

- Commands are processed with persistent configuration
- Commands include transactionId for correlation
- Headers required: span (tracing), tenant
