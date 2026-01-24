# Kafka Integration

## Topics Consumed

### EVENT_TOPIC_CHARACTER_STATUS

Character status events from character service.

| Type | Body Struct | Description |
|------|-------------|-------------|
| LOGIN | StatusEventLoginBody | Character logged in |
| LOGOUT | StatusEventLogoutBody | Character logged out |
| MAP_CHANGED | StatusEventMapChangedBody | Character changed maps |
| CHANNEL_CHANGED | ChangeChannelEventLoginBody | Character changed channels |

### EVENT_TOPIC_CASH_SHOP_STATUS

Cash shop status events.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CHARACTER_ENTER | CharacterMovementBody | Character entered cash shop |
| CHARACTER_EXIT | CharacterMovementBody | Character exited cash shop |

## Topics Produced

### EVENT_TOPIC_MAP_STATUS

Map status events emitted when characters enter or exit maps.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CHARACTER_ENTER | CharacterEnter | Character entered map |
| CHARACTER_EXIT | CharacterExit | Character exited map |

### COMMAND_TOPIC_REACTOR

Commands to reactor service.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CREATE | CreateCommandBody | Create reactor in map |

## Message Types

### Character Status Event (Consumed)

```
StatusEvent[E] {
    transactionId: UUID
    characterId: uint32
    type: string
    worldId: world.Id
    body: E
}
```

#### StatusEventLoginBody

```
{
    channelId: channel.Id
    mapId: map.Id
}
```

#### StatusEventLogoutBody

```
{
    channelId: channel.Id
    mapId: map.Id
}
```

#### StatusEventMapChangedBody

```
{
    channelId: channel.Id
    oldMapId: map.Id
    targetMapId: map.Id
    targetPortalId: uint32
}
```

#### ChangeChannelEventLoginBody

```
{
    channelId: channel.Id
    oldChannelId: channel.Id
    mapId: map.Id
}
```

### Cash Shop Status Event (Consumed)

```
StatusEvent[E] {
    transactionId: UUID
    worldId: world.Id
    type: string
    body: E
}
```

#### CharacterMovementBody

```
{
    characterId: uint32
    channelId: channel.Id
    mapId: map.Id
}
```

### Map Status Event (Produced)

```
StatusEvent[E] {
    transactionId: UUID
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    type: string
    body: E
}
```

#### CharacterEnter

```
{
    characterId: uint32
}
```

#### CharacterExit

```
{
    characterId: uint32
}
```

### Reactor Command (Produced)

```
Command[E] {
    transactionId: UUID
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    type: string
    body: E
}
```

#### CreateCommandBody

```
{
    classification: uint32
    name: string
    state: int8
    x: int16
    y: int16
    delay: uint32
    direction: byte
}
```

## Transaction Semantics

- All messages include transactionId (UUID) for tracing
- Character status consumers generate new transactionId for downstream operations
- Messages are keyed by mapId for partition ordering
- Headers include span and tenant information for distributed tracing and multi-tenancy
