# Kafka

## Topics Consumed

| Topic | Env Variable | Direction |
|---|---|---|
| Party Quest Commands | `COMMAND_TOPIC_PARTY_QUEST` | Command |

## Topics Produced

| Topic | Env Variable | Direction |
|---|---|---|
| Party Quest Status Events | `EVENT_TOPIC_PARTY_QUEST_STATUS` | Event |
| Character Commands | `COMMAND_TOPIC_CHARACTER` | Command |

## Message Types

### Commands Consumed

**Command[RegisterCommandBody]** — `REGISTER`

Registers a character/party for a party quest.

```
WorldId     world.Id
CharacterId uint32
Type        "REGISTER"
Body:
  QuestId   string
  PartyId   uint32 (optional)
  ChannelId channel.Id
  MapId     uint32
```

**Command[StartCommandBody]** — `START`

Starts a registered party quest instance.

```
WorldId     world.Id
CharacterId uint32
Type        "START"
Body:
  InstanceId uuid.UUID
```

**Command[StageClearAttemptCommandBody]** — `STAGE_CLEAR_ATTEMPT`

Attempts to clear the current stage by evaluating conditions.

```
WorldId     world.Id
CharacterId uint32
Type        "STAGE_CLEAR_ATTEMPT"
Body:
  InstanceId uuid.UUID
```

**Command[StageAdvanceCommandBody]** — `STAGE_ADVANCE`

Advances the instance to the next stage.

```
WorldId     world.Id
CharacterId uint32
Type        "STAGE_ADVANCE"
Body:
  InstanceId uuid.UUID
```

**Command[ForfeitCommandBody]** — `FORFEIT`

Forfeits and destroys the instance.

```
WorldId     world.Id
CharacterId uint32
Type        "FORFEIT"
Body:
  InstanceId uuid.UUID
```

**Command[UpdateStageStateCommandBody]** — `UPDATE_STAGE_STATE`

Updates item counts and monster kills for the current stage.

```
WorldId     world.Id
CharacterId uint32
Type        "UPDATE_STAGE_STATE"
Body:
  InstanceId   uuid.UUID
  ItemCounts   map[uint32]uint32 (optional)
  MonsterKills map[uint32]uint32 (optional)
```

### Events Produced

**StatusEvent[InstanceCreatedEventBody]** — `INSTANCE_CREATED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "INSTANCE_CREATED"
Body:
  PartyId   uint32
  ChannelId byte
```

**StatusEvent[RegistrationOpenedEventBody]** — `REGISTRATION_OPENED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "REGISTRATION_OPENED"
Body:
  Duration int64
```

**StatusEvent[StartedEventBody]** — `STARTED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "STARTED"
Body:
  StageIndex uint32
  MapIds     []uint32
```

**StatusEvent[StageClearedEventBody]** — `STAGE_CLEARED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "STAGE_CLEARED"
Body:
  StageIndex     uint32
  ChannelId      channel.Id
  MapIds         []uint32
  FieldInstances []uuid.UUID
```

**StatusEvent[StageAdvancedEventBody]** — `STAGE_ADVANCED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "STAGE_ADVANCED"
Body:
  StageIndex uint32
  MapIds     []uint32
```

**StatusEvent[CompletedEventBody]** — `COMPLETED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "COMPLETED"
Body:     (empty)
```

**StatusEvent[FailedEventBody]** — `FAILED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "FAILED"
Body:
  Reason string
```

**StatusEvent[CharacterRegisteredEventBody]** — `CHARACTER_REGISTERED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "CHARACTER_REGISTERED"
Body:
  CharacterId uint32
```

**StatusEvent[InstanceDestroyedEventBody]** — `INSTANCE_DESTROYED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "INSTANCE_DESTROYED"
Body:     (empty)
```

### Commands Produced

**Command[ChangeMapBody]** — `CHANGE_MAP` (to Character Commands topic)

Warps a character to a target map.

```
WorldId     world.Id
CharacterId uint32
Type        "CHANGE_MAP"
Body:
  ChannelId channel.Id
  MapId     map.Id
  Instance  uuid.UUID
  PortalId  uint32
```

## Transaction Semantics

All processor methods that emit messages use `message.Buffer` for batching. Messages are collected during processing and flushed atomically via `message.Emit(producer)`. This ensures that all Kafka messages for a single operation are produced together or not at all.

Required headers: `span` (tracing), `tenant` (multi-tenancy). Set via `SpanHeaderDecorator` and `TenantHeaderDecorator`.
