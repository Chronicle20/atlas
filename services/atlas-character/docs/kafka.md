# Kafka

## Topics Consumed

| Topic | Environment Variable | Direction |
|-------|---------------------|-----------|
| Character Command | COMMAND_TOPIC_CHARACTER | Command |
| Character Movement Command | COMMAND_TOPIC_CHARACTER_MOVEMENT | Command |
| Character Status Event | EVENT_TOPIC_CHARACTER_STATUS | Event |
| Session Status Event | EVENT_TOPIC_SESSION_STATUS | Event |
| Drop Status Event | EVENT_TOPIC_DROP_STATUS | Event |
| Account Status Event | EVENT_TOPIC_ACCOUNT_STATUS | Event |

## Topics Produced

| Topic | Environment Variable | Direction |
|-------|---------------------|-----------|
| Character Status Event | EVENT_TOPIC_CHARACTER_STATUS | Event |
| Character Command | COMMAND_TOPIC_CHARACTER | Command |
| Skill Command | COMMAND_TOPIC_SKILL | Command |
| Drop Command | COMMAND_TOPIC_DROP | Command |

## Message Types

### Commands Consumed

#### Character Command Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| CREATE_CHARACTER | Command[CreateCharacterCommandBody] | Create new character |
| CHANGE_JOB | Command[ChangeJobCommandBody] | Change character job |
| CHANGE_HAIR | Command[ChangeHairCommandBody] | Change hair style |
| CHANGE_FACE | Command[ChangeFaceCommandBody] | Change face |
| CHANGE_SKIN | Command[ChangeSkinCommandBody] | Change skin color |
| AWARD_EXPERIENCE | Command[AwardExperienceCommandBody] | Award experience |
| AWARD_LEVEL | Command[AwardLevelCommandBody] | Award levels |
| REQUEST_CHANGE_MESO | Command[RequestChangeMesoBody] | Request meso change |
| REQUEST_DROP_MESO | Command[RequestDropMesoCommandBody] | Request meso drop |
| REQUEST_CHANGE_FAME | Command[RequestChangeFameBody] | Request fame change |
| REQUEST_DISTRIBUTE_AP | Command[RequestDistributeApCommandBody] | Distribute AP |
| REQUEST_DISTRIBUTE_SP | Command[RequestDistributeSpCommandBody] | Distribute SP |
| CHANGE_HP | Command[ChangeHPBody] | Change HP |
| CHANGE_MP | Command[ChangeMPBody] | Change MP |
| SET_HP | Command[SetHPBody] | Set HP to specific value |
| DEDUCT_EXPERIENCE | Command[DeductExperienceCommandBody] | Deduct experience |
| RESET_STATS | Command[ResetStatsCommandBody] | Reset character stats |
| REBALANCE_AP | Command[RebalanceAPCommandBody] | Rebalance primary stat AP |
| TRANSFER_AP | Command[TransferAPCommandBody] | Transfer AP between stats/pools (AP Reset) |
| CLAMP_HP | Command[ClampHPBody] | Clamp HP to max value |
| CLAMP_MP | Command[ClampMPBody] | Clamp MP to max value |
| DELETE_CHARACTER | Command[DeleteCharacterCommandBody] | Saga-correlated delete (idempotent on missing rows) |

`CHANGE_MAP` (Command[ChangeMapBody]) is defined on the wire but has no registered consumer handler.

#### Character Movement Command Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| (none) | MovementCommand | Character movement update |

### Events Consumed

#### Character Status Event Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| LEVEL_CHANGED | StatusEvent[LevelChangedStatusEventBody] | Process level change bonuses |
| JOB_CHANGED | StatusEvent[JobChangedStatusEventBody] | Process job change bonuses |

#### Session Status Event Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| CREATED | StatusEvent | Session created (triggers login/channel change) |
| DESTROYED | StatusEvent | Session destroyed (triggers transition state) |

#### Drop Status Event Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| RESERVED | StatusEvent[ReservedStatusEventBody] | Drop reserved for pickup |

#### Account Status Event Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| DELETED | StatusEvent | Account deleted - triggers character cleanup |

### Events Produced

#### Character Status Event Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| CREATED | StatusEvent[StatusEventCreatedBody] | Character created |
| CREATION_FAILED | StatusEvent[StatusEventCreationFailedBody] | Character creation failed |
| DELETED | StatusEvent[StatusEventDeletedBody] | Character deleted |
| LOGIN | StatusEvent[StatusEventLoginBody] | Character logged in |
| LOGOUT | StatusEvent[StatusEventLogoutBody] | Character logged out |
| JOB_CHANGED | StatusEvent[JobChangedStatusEventBody] | Job changed |
| EXPERIENCE_CHANGED | StatusEvent[ExperienceChangedStatusEventBody] | Experience changed |
| LEVEL_CHANGED | StatusEvent[LevelChangedStatusEventBody] | Level changed |
| MESO_CHANGED | StatusEvent[MesoChangedStatusEventBody] | Meso changed |
| FAME_CHANGED | StatusEvent[FameChangedStatusEventBody] | Fame changed |
| STAT_CHANGED | StatusEvent[StatusEventStatChangedBody] | Stats changed |
| NAME_CHANGED | StatusEvent[StatusEventNameChangedBody] | Name changed |
| HAIR_CHANGED | StatusEvent[StatusEventHairChangedBody] | Hair changed |
| FACE_CHANGED | StatusEvent[StatusEventFaceChangedBody] | Face changed |
| GENDER_CHANGED | StatusEvent[StatusEventGenderChangedBody] | Gender changed |
| SKIN_COLOR_CHANGED | StatusEvent[StatusEventSkinColorChangedBody] | Skin color changed |
| GM_CHANGED | StatusEvent[StatusEventGmChangedBody] | GM status changed |
| DIED | StatusEvent[StatusEventDiedBody] | Character died |
| ERROR | StatusEvent[StatusEventMesoErrorBody] | Not enough meso error |
| ERROR | StatusEvent[StatusEventApTransferErrorBody] | AP transfer (point reset) rejected |

`CHANNEL_CHANGED` (StatusEvent[ChangeChannelEventLoginBody]), `MAP_CHANGED` (StatusEvent[StatusEventMapChangedBody]), and `UPDATED` (StatusEvent[StatusEventUpdatedBody]) are defined on the wire but are never emitted by any processor path.

### Commands Produced

#### Character Command Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| AWARD_LEVEL | Command[AwardLevelCommandBody] | Emitted back onto the Character Command topic when experience award crosses a level threshold |

#### Skill Command Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| REQUEST_CREATE | Command[RequestCreateBody] | Request skill creation |
| REQUEST_UPDATE | Command[RequestUpdateBody] | Request skill update |

#### Drop Command Topic

| Type | Message Struct | Description |
|------|---------------|-------------|
| SPAWN_FROM_CHARACTER | Command[SpawnFromCharacterCommandBody] | Spawn meso drop |
| REQUEST_PICK_UP | Command[RequestPickUpCommandBody] | Request drop pickup |
| CANCEL_RESERVATION | Command[CancelReservationCommandBody] | Cancel drop reservation |

## Account Status Event Messages

**StatusEvent**
```
account_id: uint32
name: string
status: string
```

## Transaction Semantics

- All commands include transactionId for correlation
- Commands are keyed by characterId for ordering
- Drop commands are keyed by mapId for ordering
- Headers include tenant context and trace span
- Account deletion events trigger cascade deletion of all characters for the account
- The Character Command topic is both consumed and produced by this service (AWARD_LEVEL is self-produced when AWARD_EXPERIENCE crosses a level threshold)
- Most database-mutating character operations emit through a transactional outbox (atlas-outbox): the status event is committed in the same database transaction as the row mutation and drained to Kafka asynchronously. LOGIN, LOGOUT, and TRANSFER_AP's resulting events are produced directly (not outbox-backed)
