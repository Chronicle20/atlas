# Kafka Integration

## Topics Consumed

### EVENT_TOPIC_CHARACTER_STATUS

Character status events from character service.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CREATED | StatusEventCreatedBody | Character was created |
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

### EVENT_TOPIC_SESSION_STATUS

Session status events.

| Type | Body Struct | Description |
|------|-------------|-------------|
| DESTROYED | StatusEvent | Session destroyed for a character |

### COMMAND_TOPIC_MAP

Map commands consumed by the service.

| Type | Body Struct | Description |
|------|-------------|-------------|
| WEATHER_START | WeatherStartCommandBody | Start weather effect in a map |

### COMMAND_TOPIC_CHARACTER

Character warp commands consumed by the service. This topic is also produced by the service (see Topics Produced).

| Type | Body Struct | Description |
|------|-------------|-------------|
| CHANGE_MAP | ChangeMapBody | Move a character to a different map |

### COMMAND_TOPIC_CHARACTER_CHANNEL_CHANGE_REQUEST

Character channel-change requests.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CHANNEL_CHANGE_REQUEST | ChannelChangeRequestCommand | Request to move a character to a different channel |

### COMMAND_TOPIC_MIST

Mist lifecycle commands.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CREATE | CreateCommandBody | Create a mist on a map |
| CANCEL | CancelCommandBody | Cancel an existing mist |

### EVENT_TOPIC_DATA

Data ingestion events, filtered to the MAP worker. Toggled off via DATA_EVENTS_CONSUMER_ENABLED.

| Type | Body Struct | Description |
|------|-------------|-------------|
| DATA_UPDATED | dataUpdatedEventBody | Tenant's map data was re-ingested; triggers a spawn point registry flush for the tenant |

## Topics Produced

### EVENT_TOPIC_MAP_STATUS

Map status events emitted when characters enter or exit maps, when weather effects start or end, and when a map-stay timer is started.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CHARACTER_ENTER | CharacterEnter | Character entered map |
| CHARACTER_EXIT | CharacterExit | Character exited map |
| WEATHER_START | WeatherStart | Weather effect started in map |
| WEATHER_END | WeatherEnd | Weather effect ended in map |
| MAP_TIMER_STARTED | MapTimerStarted | Map-stay timer started for a character |

### COMMAND_TOPIC_CHARACTER

Commands to character service.

| Type | Body Struct | Description |
|------|-------------|-------------|
| CHANGE_MAP | ChangeMapBody | Move a character to a different map |

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

### EVENT_TOPIC_CHARACTER_STATUS

Character status events emitted by the service when it resolves a character's map or channel. This topic is also consumed by the service (see Topics Consumed).

| Type | Body Struct | Description |
|------|-------------|-------------|
| MAP_CHANGED | StatusEventMapChangedBody | Character's map was changed via ChangeMap |
| CHANNEL_CHANGED | ChangeChannelEventLoginBody | Character's channel was changed via a channel-change request |

### EVENT_TOPIC_MIST

Mist lifecycle events.

| Type | Body Struct | Description |
|------|-------------|-------------|
| MIST_CREATED | CreatedBody | Mist was created |
| MIST_DESTROYED | DestroyedBody | Mist was destroyed (expired or cancelled) |

### COMMAND_TOPIC_CHARACTER_BUFF

Disease-application commands sent when a character is inside an active mist's bounding box on a tick.

| Type | Body Struct | Description |
|------|-------------|-------------|
| APPLY | applyDiseaseBody | Apply a disease/stat-change buff to a character |

## Message Types

### Character Status Event (Consumed)

```
StatusEvent[E] {
    transactionId: UUID
    worldId: world.Id
    characterId: uint32
    type: string
    body: E
}
```

#### StatusEventCreatedBody

```
{
    name: string
    mapId: map.Id
    instance: UUID
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
    useTargetPosition: bool
    targetX: int16
    targetY: int16
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

### Session Status Event (Consumed)

```
StatusEvent {
    sessionId: UUID
    accountId: uint32
    characterId: uint32
    worldId: world.Id
    channelId: channel.Id
    issuer: string
    type: string
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

### Character Channel Change Request Command (Consumed)

```
ChannelChangeRequestCommand {
    transactionId: UUID
    characterId: uint32
    worldId: world.Id
    oldChannelId: channel.Id
    targetChannelId: channel.Id
}
```

### Data Event (Consumed)

```
event[E] {
    type: string
    body: E
}
```

#### dataUpdatedEventBody

```
{
    tenantId: string
    worker: string
    completedAt: string
}
```

### Mist Command (Consumed)

```
Command[E] {
    tenant: UUID
    type: string
    body: E
}
```

#### CreateCommandBody

```
{
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
    ownerType: string
    ownerId: uint32
    originX: int16
    originY: int16
    ltX: int16
    ltY: int16
    rbX: int16
    rbY: int16
    disease: string
    diseaseValue: int32
    diseaseDuration: int64
    duration: int64
    tickIntervalMs: int64
    sourceSkillId: uint32
    sourceSkillLevel: uint32
}
```

#### CancelCommandBody

```
{
    mistId: UUID
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

#### MapTimerStarted

```
{
    characterId: uint32
    seconds: uint32
}
```

### Character Command (Produced)

```
Command[E] {
    transactionId: UUID
    worldId: world.Id
    characterId: uint32
    type: string
    body: E
}
```

#### ChangeMapBody

```
{
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
    portalId: uint32
    useTargetPosition: bool
    targetX: int16
    targetY: int16
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

### Mist Event (Produced)

```
Event[E] {
    tenant: UUID
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
    mistId: UUID
    type: string
    body: E
}
```

#### CreatedBody

```
{
    ownerType: string
    ownerId: uint32
    sourceSkillId: uint32
    sourceSkillLevel: uint32
    type: int32
    originX: int16
    originY: int16
    ltX: int16
    ltY: int16
    rbX: int16
    rbY: int16
    duration: int64
}
```

#### DestroyedBody

```
{
    reason: string
}
```

### Character Buff Command (Produced)

```
buffCommand[E] {
    worldId: world.Id
    channelId: channel.Id
    mapId: map.Id
    instance: UUID
    characterId: uint32
    type: string
    body: E
}
```

#### applyDiseaseBody

```
{
    fromId: uint32
    sourceId: int32
    level: byte
    duration: int32
    changes: []statChange
}
```

#### statChange

```
{
    type: string
    amount: int32
}
```

## Transaction Semantics

- Most messages include transactionId (UUID) for tracing; Mist commands/events and the Character Buff command carry tenant/target identifiers instead of transactionId
- Character status consumers generate new transactionId for downstream operations
- Messages are keyed by mapId for partition ordering
- MAP_TIMER_STARTED, CHANGE_MAP, MAP_CHANGED, CHANNEL_CHANGED, and Character Buff APPLY messages are keyed by characterId
- Mist events are keyed by mistId
- Headers include span and tenant information for distributed tracing and multi-tenancy
- Map command consumer starts from last offset (does not replay historical commands)
- Data event (EVENT_TOPIC_DATA) consumer starts from last offset and runs in a dedicated consumer group
- EVENT_TOPIC_CHARACTER_STATUS MAP_CHANGED and CHANNEL_CHANGED are both consumed and produced by this service (self-consumption to update in-memory map registries after a persisted location change)
- COMMAND_TOPIC_CHARACTER CHANGE_MAP is both consumed and produced by this service
