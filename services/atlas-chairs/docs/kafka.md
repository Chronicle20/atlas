# Kafka Integration

## Topics Consumed

| Topic Environment Variable | Consumer Group | Description |
|---------------------------|----------------|-------------|
| COMMAND_TOPIC_CHAIR | Chairs Service | Chair commands |
| EVENT_TOPIC_CHARACTER_STATUS | Chairs Service | Character status events |

## Topics Produced

| Topic Environment Variable | Description |
|---------------------------|-------------|
| EVENT_TOPIC_CHAIR_STATUS | Chair status events |

## Message Types

### Commands

#### Command (Chair)

Consumed from COMMAND_TOPIC_CHAIR. Generic envelope with typed body.

| Field | Type |
|-------|------|
| WorldId | world.Id |
| ChannelId | channel.Id |
| MapId | map.Id |
| Type | string |
| Body | varies |

##### Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| USE | UseChairCommandBody | Sit on chair |
| CANCEL | CancelChairCommandBody | Stop sitting |

##### UseChairCommandBody

| Field | Type |
|-------|------|
| CharacterId | uint32 |
| ChairType | string |
| ChairId | uint32 |

##### CancelChairCommandBody

| Field | Type |
|-------|------|
| CharacterId | uint32 |

### Events Consumed

#### StatusEvent (Character)

Consumed from EVENT_TOPIC_CHARACTER_STATUS.

| Field | Type |
|-------|------|
| CharacterId | uint32 |
| Type | string |
| WorldId | world.Id |
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

##### StatusEventLogoutBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| MapId | map.Id |

##### StatusEventMapChangedBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| OldMapId | map.Id |
| TargetMapId | map.Id |
| TargetPortalId | uint32 |

##### ChangeChannelEventLoginBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| OldChannelId | channel.Id |
| MapId | map.Id |

### Events Produced

#### StatusEvent (Chair)

Produced to EVENT_TOPIC_CHAIR_STATUS.

| Field | Type |
|-------|------|
| WorldId | world.Id |
| ChannelId | channel.Id |
| MapId | map.Id |
| ChairType | string |
| ChairId | uint32 |
| Type | string |
| Body | varies |

##### Chair Status Event Types

| Type | Body Type | Description |
|------|-----------|-------------|
| USED | StatusEventUsedBody | Character sat on chair |
| CANCELLED | StatusEventCancelledBody | Character left chair |
| ERROR | StatusEventErrorBody | Chair operation failed |

##### StatusEventUsedBody

| Field | Type |
|-------|------|
| CharacterId | uint32 |

##### StatusEventCancelledBody

| Field | Type |
|-------|------|
| CharacterId | uint32 |

##### StatusEventErrorBody

| Field | Type |
|-------|------|
| CharacterId | uint32 |
| Type | string |

## Transaction Semantics

- Commands are processed with persistent configuration
- Headers required: span (tracing), tenant
