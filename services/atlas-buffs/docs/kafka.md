# Kafka Integration

## Topics Consumed

| Topic Environment Variable | Consumer Group | Description |
|---------------------------|----------------|-------------|
| COMMAND_TOPIC_CHARACTER_BUFF | Buff Service (buff_command) | Buff commands |
| EVENT_TOPIC_CHARACTER_STATUS | Buff Service (character_status_event) | Character status events (login, logout, stat changed, map changed, channel changed) |
| EVENT_TOPIC_SKILL_STATUS | Buff Service (skill_status_event) | Skill status events (updated, deleted) |

## Topics Produced

| Topic Environment Variable | Description |
|---------------------------|-------------|
| EVENT_TOPIC_CHARACTER_BUFF_STATUS | Buff status events |
| COMMAND_TOPIC_CHARACTER | Character commands (periodic HP changes) |

## Message Types

### Commands

#### Command

Consumed from COMMAND_TOPIC_CHARACTER_BUFF. Generic envelope with typed body.

| Field | Type |
|-------|------|
| WorldId | world.Id |
| ChannelId | channel.Id |
| MapId | _map.Id |
| Instance | uuid.UUID |
| CharacterId | uint32 |
| Type | string |
| Body | varies |

##### Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| APPLY | ApplyCommandBody | Apply buff to character |
| CANCEL | CancelCommandBody | Cancel buff on character |
| CANCEL_ALL | CancelAllCommandBody | Cancel all buffs on character |
| CANCEL_BY_TYPES | CancelByTypesCommandBody | Cancel buffs whose Changes() intersect a stat-type set |
| CANCEL_BY_CORRELATION | CancelByCorrelationCommandBody | Cancel every buff across the tenant carrying a given correlationId |
| UPDATE_STAT_VALUE | UpdateStatValueCommandBody | Mutate the amount of one stat on an existing buff (or create one) |
| EXPIRE | ExpireCommandBody | Re-evaluate and announce one character's genuinely lapsed buffs |

##### ApplyCommandBody

| Field | Type |
|-------|------|
| FromId | uint32 |
| SourceId | int32 |
| Level | byte |
| Duration | int32 |
| Changes | []StatChange |
| Accumulate | bool |
| NoExpiry | bool |
| CorrelationId | string |

Duration is milliseconds. NoExpiry, when set, requires Duration to be 0; the consumer rejects the command otherwise.

##### CancelCommandBody

| Field | Type |
|-------|------|
| SourceId | int32 |

##### CancelAllCommandBody

Empty body.

##### CancelByTypesCommandBody

| Field | Type |
|-------|------|
| Types | []string |

Each entry is a `TemporaryStatType` string (`"POISON"`, `"DARKNESS"`, `"WEAKEN"`, `"SEAL"`, `"CURSE"`, etc.).

##### CancelByCorrelationCommandBody

| Field | Type |
|-------|------|
| CorrelationId | string |

Sweeps the whole tenant regardless of world/channel. The envelope's CharacterId/WorldId/ChannelId are not consulted; emitters send CharacterId 0.

##### UpdateStatValueCommandBody

| Field | Type |
|-------|------|
| SourceId | int32 |
| StatType | string |
| Operation | string |
| Amount | int32 |
| Cap | int32 |
| CreateIfMissing | bool |
| Level | byte |

Operation is one of `INCREMENT` or `SET`. Cap applies to INCREMENT only. CreateIfMissing and Level apply only to the INCREMENT-against-missing-buff path.

##### ExpireCommandBody

Empty body.

##### StatChange

| Field | Type |
|-------|------|
| Type | string |
| Amount | int32 |

#### CharacterCommand

Produced to COMMAND_TOPIC_CHARACTER. Generic envelope with typed body.

| Field | Type |
|-------|------|
| CharacterId | uint32 |
| WorldId | world.Id |
| Type | string |
| Body | varies |

##### Character Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| CHANGE_HP | ChangeHPCommandBody | Apply a periodic HP effect tick to character |

##### ChangeHPCommandBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| Amount | int16 |

### Events

#### StatusEvent

Produced to EVENT_TOPIC_CHARACTER_BUFF_STATUS. Generic envelope with typed body.

| Field | Type |
|-------|------|
| WorldId | world.Id |
| CharacterId | uint32 |
| Type | string |
| Body | varies |

##### Status Event Types

| Type | Body Type | Description |
|------|-----------|-------------|
| APPLIED | AppliedStatusEventBody | Buff applied (or created via UPDATE_STAT_VALUE's CreateIfMissing) |
| EXPIRED | ExpiredStatusEventBody | Buff expired or cancelled |
| STAT_UPDATED | StatUpdatedStatusEventBody | An existing buff's stat value changed via UPDATE_STAT_VALUE |
| BERSERK | BerserkStatusEventBody | Dark Knight Berserk aura state broadcast |
| PERIODIC_EFFECT | PeriodicEffectStatusEventBody | Visual pulse accompanying a periodic tick's resource change |

##### AppliedStatusEventBody

| Field | Type |
|-------|------|
| FromId | uint32 |
| SourceId | int32 |
| Level | byte |
| Duration | int32 |
| Changes | []StatChange |
| CreatedAt | time.Time |
| ExpiresAt | time.Time |
| NoExpiry | bool |

##### ExpiredStatusEventBody

| Field | Type |
|-------|------|
| SourceId | int32 |
| Level | byte |
| Duration | int32 |
| Changes | []StatChange |
| CreatedAt | time.Time |
| ExpiresAt | time.Time |
| NoExpiry | bool |

##### StatUpdatedStatusEventBody

| Field | Type |
|-------|------|
| SourceId | int32 |
| Level | byte |
| Duration | int32 |
| Changes | []StatChange |
| CreatedAt | time.Time |
| ExpiresAt | time.Time |

CreatedAt/ExpiresAt are the buff's original timestamps; a stat-value update never extends the buff's lifetime.

##### BerserkStatusEventBody

| Field | Type |
|-------|------|
| TransactionId | uuid.UUID |
| ChannelId | channel.Id |
| SkillId | uint32 |
| CharacterLevel | byte |
| SkillLevel | byte |
| Active | bool |

Emitted per tracked Dark Knight every BroadcastPeriod (3s), and inline on a state transition.

##### PeriodicEffectStatusEventBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| SkillId | uint32 |
| StatType | string |

Emitted alongside — never instead of — the tick's CHANGE_HP command, only for periodic-effect table rows flagged specialEffect.

### Consumed Character Status Events

Mirrors atlas-character's produced status events (source of truth: services/atlas-character). Consumed from EVENT_TOPIC_CHARACTER_STATUS.

#### StatusEvent

| Field | Type |
|-------|------|
| TransactionId | uuid.UUID |
| WorldId | world.Id |
| CharacterId | uint32 |
| Type | string |
| Body | varies |

##### Consumed Character Status Event Types

| Type | Body Type | Description |
|------|-----------|-------------|
| LOGIN | StatusEventLoginBody | Character login |
| LOGOUT | StatusEventLogoutBody | Character logout |
| CHANNEL_CHANGED | StatusEventChannelChangedBody | Character channel transfer |
| MAP_CHANGED | StatusEventMapChangedBody | Character map transfer |
| STAT_CHANGED | StatusEventStatChangedBody | Character stat update |

##### StatusEventLoginBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| MapId | _map.Id |
| Instance | uuid.UUID |

##### StatusEventLogoutBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| MapId | _map.Id |
| Instance | uuid.UUID |

##### StatusEventChannelChangedBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| OldChannelId | channel.Id |
| MapId | _map.Id |
| Instance | uuid.UUID |

##### StatusEventMapChangedBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| OldMapId | _map.Id |
| OldInstance | uuid.UUID |
| TargetMapId | _map.Id |
| TargetInstance | uuid.UUID |
| TargetPortalId | uint32 |

##### StatusEventStatChangedBody

| Field | Type |
|-------|------|
| ChannelId | channel.Id |
| ExclRequestSent | bool |
| Updates | []stat.Type |

### Consumed Skill Status Events

Mirrors atlas-skills's produced status events (source of truth: services/atlas-skills). Consumed from EVENT_TOPIC_SKILL_STATUS.

#### StatusEvent

| Field | Type |
|-------|------|
| TransactionId | uuid.UUID |
| WorldId | world.Id |
| CharacterId | uint32 |
| SkillId | uint32 |
| Type | string |
| Body | varies |

##### Consumed Skill Status Event Types

| Type | Body Type | Description |
|------|-----------|-------------|
| UPDATED | StatusEventUpdatedBody | Skill level updated |
| DELETED | StatusEventDeletedBody | Skill deleted |

Handlers filter to SkillId matching the Dark Knight Berserk skill; all other skill ids are ignored.

##### StatusEventUpdatedBody

| Field | Type |
|-------|------|
| Level | byte |
| MasterLevel | byte |
| Expiration | time.Time |

##### StatusEventDeletedBody

Empty body.

## Transaction Semantics

- Commands are processed with persistent configuration
- Headers required: span (tracing), tenant
- Buff status events, berserk status events, periodic-effect status events, and character HP commands are emitted atomically per processing pass via message buffer
