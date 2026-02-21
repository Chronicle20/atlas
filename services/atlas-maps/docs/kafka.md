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
| DELETED | StatusEventDeletedBody | Character deleted |

### EVENT_TOPIC_CASH_SHOP_STATUS

Cash shop status events.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CHARACTER_ENTER | CharacterMovementBody | Character entered cash shop |
| CHARACTER_EXIT | CharacterMovementBody | Character exited cash shop |

### EVENT_TOPIC_MONSTER_STATUS

Monster status events from monster service.

| Type | Body Struct | Description |
|------|-------------|-------------|
| KILLED | StatusEventKilledBody | Monster was killed |

### COMMAND_TOPIC_MAP

Map commands consumed by the service.

| Type | Body Struct | Description |
|------|-------------|-------------|
| WEATHER_START | WeatherStartCommandBody | Start weather effect in a map |

## Topics Produced

### EVENT_TOPIC_MAP_STATUS

Map status events emitted when characters enter or exit maps, and when weather effects start or end.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CHARACTER_ENTER | CharacterEnter | Character entered map |
| CHARACTER_EXIT | CharacterExit | Character exited map |
| WEATHER_START | WeatherStart | Weather effect started in map |
| WEATHER_END | WeatherEnd | Weather effect ended in map |

### COMMAND_TOPIC_REACTOR

Commands to reactor service.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CREATE | CreateCommandBody | Create reactor in map |

### COMMAND_TOPIC_MAP_ACTIONS

Commands to map actions service.

| Type | Body Struct | Description |
|------|-------------|-------------|
| ENTER | EnterCommandBody | Trigger map entry script |

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
    instance: UUID
}
```

#### StatusEventLogoutBody

```
{
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
}
```

#### StatusEventMapChangedBody

```
{
    channelId: channel.Id
    oldMapId: map.Id
    oldInstance: UUID
    targetMapId: map.Id
    targetInstance: UUID
    targetPortalId: uint32
}
```

#### ChangeChannelEventLoginBody

```
{
    channelId: channel.Id
    oldChannelId: channel.Id
    mapId: map.Id
    instance: UUID
}
```

#### StatusEventDeletedBody

```
{}
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

### Monster Status Event (Consumed)

```
StatusEvent[E] {
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
    uniqueId: uint32
    monsterId: uint32
    type: string
    body: E
}
```

#### StatusEventKilledBody

```
{
    x: int16
    y: int16
    actorId: uint32
    damageEntries: []DamageEntry
}
```

#### DamageEntry

```
{
    characterId: uint32
    damage: uint32
}
```

### Map Command (Consumed)

```
Command[E] {
    transactionId: UUID
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
    type: string
    body: E
}
```

#### WeatherStartCommandBody

```
{
    itemId: uint32
    message: string
    durationMs: uint32
}
```

### Map Status Event (Produced)

```
StatusEvent[E] {
    transactionId: UUID
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
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

#### WeatherStart

```
{
    itemId: uint32
    message: string
}
```

#### WeatherEnd

```
{
    itemId: uint32
}
```

### Reactor Command (Produced)

```
Command[E] {
    transactionId: UUID
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
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

### Map Actions Command (Produced)

```
Command[E] {
    transactionId: UUID
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
    type: string
    body: E
}
```

#### EnterCommandBody

```
{
    characterId: uint32
    scriptName: string
    scriptType: string
}
```

## Transaction Semantics

- All messages include transactionId (UUID) for tracing
- Character status consumers generate new transactionId for downstream operations
- Messages are keyed by mapId for partition ordering
- Headers include span and tenant information for distributed tracing and multi-tenancy
- Map command consumer starts from last offset (does not replay historical commands)
