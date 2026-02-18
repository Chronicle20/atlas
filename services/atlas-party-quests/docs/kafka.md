# Kafka

## Topics Consumed

| Topic | Env Variable | Direction |
|---|---|---|
| Party Quest Commands | `COMMAND_TOPIC_PARTY_QUEST` | Command |
| Character Status Events | `EVENT_TOPIC_CHARACTER_STATUS` | Event |
| Monster Status Events | `EVENT_TOPIC_MONSTER_STATUS` | Event |

## Topics Produced

| Topic | Env Variable | Direction |
|---|---|---|
| Party Quest Status Events | `EVENT_TOPIC_PARTY_QUEST_STATUS` | Event |
| Character Commands | `COMMAND_TOPIC_CHARACTER` | Command |
| Reactor Commands | `COMMAND_TOPIC_REACTOR` | Command |
| System Message Commands | `COMMAND_TOPIC_SYSTEM_MESSAGE` | Command |
| Map Commands | `COMMAND_TOPIC_MAP` | Command |

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

Force-completes the current stage (bypasses condition evaluation) and advances.

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

**Command[LeaveCommandBody]** — `LEAVE`

Removes the sending character from their active instance.

```
WorldId     world.Id
CharacterId uint32
Type        "LEAVE"
Body:       (empty)
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

**Command[UpdateCustomDataCommandBody]** — `UPDATE_CUSTOM_DATA`

Sets and increments custom data keys in the current stage state.

```
WorldId     world.Id
CharacterId uint32
Type        "UPDATE_CUSTOM_DATA"
Body:
  InstanceId uuid.UUID
  Updates    map[string]string (optional)
  Increments []string (optional)
```

**Command[BroadcastMessageCommandBody]** — `BROADCAST_MESSAGE`

Sends a system message to all characters in the instance.

```
WorldId     world.Id
CharacterId uint32
Type        "BROADCAST_MESSAGE"
Body:
  InstanceId  uuid.UUID
  MessageType string
  Message     string
```

**Command[EnterBonusCommandBody]** — `ENTER_BONUS`

Transitions a completed instance into the bonus stage.

```
WorldId     world.Id
CharacterId uint32
Type        "ENTER_BONUS"
Body:
  InstanceId uuid.UUID
```

### Events Consumed

**StatusEvent[StatusEventLogoutBody]** — `LOGOUT` (from Character Status Events topic)

Triggers automatic leave for the character from their active PQ instance.

```
WorldId     world.Id
CharacterId uint32
Type        "LOGOUT"
Body:
  ChannelId channel.Id
  MapId     map.Id
  Instance  uuid.UUID
```

**StatusEvent[DamagedBody]** — `DAMAGED` (from Monster Status Events topic)

Triggers friendly monster damaged handling if the monster matches a PQ instance's friendly monster configuration.

```
WorldId   world.Id
ChannelId channel.Id
MapId     map.Id
Instance  uuid.UUID
UniqueId  uint32
MonsterId uint32
Type      "DAMAGED"
Body:
  X             int16
  Y             int16
  ObserverId    uint32
  ActorId       uint32
  Boss          bool
  DamageEntries []DamageEntry
```

**StatusEvent[KilledBody]** — `KILLED` (from Monster Status Events topic)

Triggers friendly monster killed handling if the monster matches a PQ instance's friendly monster configuration.

```
WorldId   world.Id
ChannelId channel.Id
MapId     map.Id
Instance  uuid.UUID
UniqueId  uint32
MonsterId uint32
Type      "KILLED"
Body:
  X             int16
  Y             int16
  ActorId       uint32
  Boss          bool
  DamageEntries []DamageEntry
```

**StatusEvent[FriendlyDropBody]** — `FRIENDLY_DROP` (from Monster Status Events topic)

Triggers friendly monster drop handling if the monster matches a PQ instance's friendly monster configuration.

```
WorldId   world.Id
ChannelId channel.Id
MapId     map.Id
Instance  uuid.UUID
UniqueId  uint32
MonsterId uint32
Type      "FRIENDLY_DROP"
Body:
  ItemCount uint32
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

**StatusEvent[CharacterLeftEventBody]** — `CHARACTER_LEFT`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "CHARACTER_LEFT"
Body:
  CharacterId uint32
  ChannelId   channel.Id
  Reason      string
```

**StatusEvent[BonusEnteredEventBody]** — `BONUS_ENTERED`

```
WorldId    world.Id
InstanceId uuid.UUID
QuestId    string
Type       "BONUS_ENTERED"
Body:
  MapId uint32
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

**Command[AwardExperienceCommandBody]** — `AWARD_EXPERIENCE` (to Character Commands topic)

Awards experience to a character.

```
WorldId     world.Id
CharacterId uint32
Type        "AWARD_EXPERIENCE"
Body:
  ChannelId     channel.Id
  Distributions []ExperienceDistributions
```

**Command[DestroyInFieldCommandBody]** — `DESTROY_IN_FIELD` (to Reactor Commands topic)

Destroys all reactors in a field instance.

```
WorldId   world.Id
ChannelId channel.Id
MapId     map.Id
Instance  uuid.UUID
Type      "DESTROY_IN_FIELD"
Body:     (empty)
```

**Command[SendMessageBody]** — `SEND_MESSAGE` (to System Message Commands topic)

Sends a system message to a character.

```
TransactionId uuid.UUID
WorldId       world.Id
ChannelId     channel.Id
CharacterId   uint32
Type          "SEND_MESSAGE"
Body:
  MessageType string
  Message     string
```

**Command[WeatherStartCommandBody]** — `WEATHER_START` (to Map Commands topic)

Starts a weather effect in a field instance.

```
WorldId   world.Id
ChannelId channel.Id
MapId     map.Id
Instance  uuid.UUID
Type      "WEATHER_START"
Body:
  ItemId     uint32
  Message    string
  DurationMs uint32
```

## Transaction Semantics

All processor methods that emit messages use `message.Buffer` for batching. Messages are collected during processing and flushed atomically via `message.Emit(producer)`. This ensures that all Kafka messages for a single operation are produced together or not at all.

Required headers: `span` (tracing), `tenant` (multi-tenancy). Set via `SpanHeaderParser` and `TenantHeaderParser`.
