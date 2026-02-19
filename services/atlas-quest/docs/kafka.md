# Quest Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Direction | Description |
|-------|---------------------|-----------|-------------|
| Quest Command | COMMAND_TOPIC_QUEST | Command | Quest lifecycle commands |
| Monster Status | EVENT_TOPIC_MONSTER_STATUS | Event | Monster kill events for progress tracking |
| Asset Status | EVENT_TOPIC_ASSET_STATUS | Event | Asset events (consumer registered, no handlers) |
| Character Status | EVENT_TOPIC_CHARACTER_STATUS | Event | Character deletion and map change events |

## Topics Produced

| Topic | Environment Variable | Direction | Description |
|-------|---------------------|-----------|-------------|
| Quest Status | EVENT_TOPIC_QUEST_STATUS | Event | Quest state change events |
| Saga Command | COMMAND_TOPIC_SAGA | Command | Saga commands for rewards processing |

## Message Types

### Quest Commands (Consumed)

#### Command

Generic command envelope.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Saga correlation ID |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| characterId | uint32 | Character identifier |
| type | string | Command type |
| body | object | Command-specific payload |

#### Command Types

| Type | Body | Description |
|------|------|-------------|
| START | StartCommandBody | Start a quest |
| COMPLETE | CompleteCommandBody | Complete a quest |
| FORFEIT | ForfeitCommandBody | Forfeit a quest |
| UPDATE_PROGRESS | UpdateProgressCommandBody | Update quest progress |
| RESTORE_ITEM | RestoreItemCommandBody | Restore a lost quest item |

#### StartCommandBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier |
| npcId | uint32 | NPC identifier (optional) |
| force | bool | Skip requirement validation |

#### CompleteCommandBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier |
| npcId | uint32 | NPC identifier (optional) |
| selection | int32 | Reward selection (optional) |
| force | bool | Skip requirement validation |

#### ForfeitCommandBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier |

#### UpdateProgressCommandBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier |
| infoNumber | uint32 | Objective identifier |
| progress | string | Progress value |

#### RestoreItemCommandBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier |
| itemId | uint32 | Item template ID to restore |

### Quest Status Events (Produced)

#### StatusEvent

Generic event envelope.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Saga correlation ID |
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Event type |
| body | object | Event-specific payload |

#### Event Types

| Type | Body | Description |
|------|------|-------------|
| STARTED | QuestStartedEventBody | Quest started |
| COMPLETED | QuestCompletedEventBody | Quest completed |
| FORFEITED | QuestForfeitedEventBody | Quest forfeited |
| PROGRESS_UPDATED | QuestProgressUpdatedEventBody | Progress updated |
| ERROR | ErrorStatusEventBody | Operation failed |

#### QuestStartedEventBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier |
| progress | string | Initial progress string |

#### QuestCompletedEventBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier |
| completedAt | time.Time | Completion timestamp |
| items | []ItemReward | Awarded items (optional) |

#### ItemReward

| Field | Type | Description |
|-------|------|-------------|
| itemId | uint32 | Item template identifier |
| amount | int32 | Item quantity |

#### QuestForfeitedEventBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier |

#### QuestProgressUpdatedEventBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier |
| infoNumber | uint32 | Objective identifier |
| progress | string | Full progress string |

#### ErrorStatusEventBody

| Field | Type | Description |
|-------|------|-------------|
| questId | uint32 | Quest identifier (optional) |
| error | string | Error type |

#### Error Types

| Error | Description |
|-------|-------------|
| QUEST_NOT_FOUND | Quest does not exist |
| QUEST_ALREADY_ACTIVE | Quest already started |
| QUEST_NOT_STARTED | Quest not in started state |
| QUEST_ALREADY_COMPLETED | Quest already completed |
| REQUIREMENTS_NOT_MET | Requirements not satisfied |
| UNKNOWN_ERROR | Unexpected error |

### Monster Status Events (Consumed)

#### StatusEvent

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| uniqueId | uint32 | Monster unique identifier |
| monsterId | uint32 | Monster template identifier |
| type | string | Event type |
| body | object | Event-specific payload |

#### StatusEventKilledBody

Consumed from EVENT_TOPIC_MONSTER_STATUS. Processes KILLED events to update monster kill progress for active quests.

| Field | Type | Description |
|-------|------|-------------|
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| actorId | uint32 | Actor identifier |
| damageEntries | []DamageEntry | Damage entries per character |

#### DamageEntry

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| damage | uint32 | Damage dealt |

### Character Status Events (Consumed)

#### StatusEvent

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| characterId | uint32 | Character identifier |
| worldId | world.Id | World identifier |
| type | string | Event type |
| body | object | Event-specific payload |

#### Event Types Handled

| Type | Body | Description |
|------|------|-------------|
| DELETED | StatusEventDeletedBody | Character deleted, cascade delete quests |
| MAP_CHANGED | StatusEventMapChangedBody | Character changed maps |

#### StatusEventMapChangedBody

Consumed from EVENT_TOPIC_CHARACTER_STATUS. Processes MAP_CHANGED events to:
- Check for auto-start quests on the new map
- Update map visit progress for active quests
- Check for auto-complete after progress updates

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel identifier |
| oldMapId | map.Id | Previous map identifier |
| targetMapId | map.Id | New map identifier |
| targetPortalId | uint32 | Target portal identifier |

### Saga Commands (Produced)

#### Saga

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Unique transaction identifier |
| sagaType | Type | Saga type |
| initiatedBy | string | Originating identifier |
| steps | []Step | Saga steps |

#### Saga Types

| Type | Description |
|------|-------------|
| quest_start | Quest start actions |
| quest_complete | Quest completion rewards |
| quest_restore_item | Quest item restoration |

#### Step

| Field | Type | Description |
|-------|------|-------------|
| stepId | string | Step identifier |
| status | Status | Step status (pending/completed/failed) |
| action | Action | Action type |
| payload | object | Action-specific payload |

#### Actions

| Action | Payload | Description |
|--------|---------|-------------|
| award_asset | AwardItemPayload | Award item to character |
| award_experience | AwardExperiencePayload | Award experience |
| award_mesos | AwardMesosPayload | Award mesos |
| award_fame | AwardFamePayload | Award fame |
| create_skill | CreateSkillPayload | Grant skill |
| destroy_asset | ConsumeItemPayload | Consume item |

#### AwardItemPayload

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| item | ItemDetail | Item details |

#### ItemDetail

| Field | Type | Description |
|-------|------|-------------|
| templateId | uint32 | Item template identifier |
| quantity | uint32 | Item quantity |
| period | uint32 | Item period (optional) |
| expiration | time.Time | Item expiration (optional) |

#### AwardExperiencePayload

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| distributions | []ExperienceDistribution | Experience distributions |

#### ExperienceDistribution

| Field | Type | Description |
|-------|------|-------------|
| experienceType | string | Experience type (WHITE) |
| amount | uint32 | Experience amount |
| attr1 | uint32 | Additional attribute |

#### AwardMesosPayload

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| actorId | uint32 | Quest identifier |
| actorType | string | Actor type (quest) |
| amount | int32 | Meso amount |

#### AwardFamePayload

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| actorId | uint32 | Quest identifier |
| actorType | string | Actor type (quest) |
| amount | int16 | Fame amount |

#### CreateSkillPayload

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| skillId | uint32 | Skill identifier |
| level | byte | Skill level |
| masterLevel | byte | Skill master level |
| expiration | time.Time | Skill expiration (optional) |

#### ConsumeItemPayload

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| templateId | uint32 | Item template identifier |
| quantity | uint32 | Quantity to consume |

## Transaction Semantics

- Quest commands carry a transactionId for saga correlation
- Status events include the originating transactionId
- Saga commands generate a new transactionId for reward processing
- Message ordering is guaranteed per character (partitioned by characterId)
