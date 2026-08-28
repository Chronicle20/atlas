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
| COMMAND_TOPIC_CHARACTER | Character HP/MP change commands |

## Message Types

### Commands

#### Command (Chair)

Consumed from COMMAND_TOPIC_CHAIR. Generic envelope with typed body.

| Field | Type |
|-------|------|
| WorldId | world.Id |
| ChannelId | channel.Id |
| MapId | map.Id |
| Instance | uuid.UUID |
| Type | string |
| Body | varies |

##### Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| USE | UseChairCommandBody | Sit on chair |
| CANCEL | CancelChairCommandBody | Stop sitting |
| RECOVERY | RecoveryCommandBody | HP/MP recovery tick |

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

##### RecoveryCommandBody

| Field | Type |
|-------|------|
| CharacterId | uint32 |
| Hp | int16 |
| Mp | int16 |

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

#### StatusEvent (Chair)

Produced to EVENT_TOPIC_CHAIR_STATUS.

| Field | Type |
|-------|------|
| WorldId | world.Id |
| ChannelId | channel.Id |
| MapId | map.Id |
| Instance | uuid.UUID |
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

### Commands Produced

#### Command (Character)

Produced to COMMAND_TOPIC_CHARACTER.

| Field | Type |
|-------|------|
| WorldId | world.Id |
| CharacterId | uint32 |
| Type | string |
| Body | varies |

##### Character Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| CHANGE_HP | ChangeHPCommandBody | Change character HP |
| CHANGE_MP | ChangeMPCommandBody | Change character MP |

##### ChangeHPCommandBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| Amount | int16 |

##### ChangeMPCommandBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| Amount | int16 |

## Transaction Semantics

- Commands are processed with persistent configuration
- Headers required: span (tracing), tenant
