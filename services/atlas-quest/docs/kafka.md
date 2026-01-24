# Quest Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Direction | Description |
|-------|---------------------|-----------|-------------|
| Quest Command | COMMAND_TOPIC_QUEST | Command | Quest lifecycle commands |
| Monster Status | EVENT_TOPIC_MONSTER_STATUS | Event | Monster kill events for progress tracking |
| Asset Status | EVENT_TOPIC_ASSET_STATUS | Event | Asset events (consumer registered but no handlers) |
| Character Status | EVENT_TOPIC_CHARACTER_STATUS | Event | Character map change events for progress and auto-start |

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
| award_inventory | AwardItemPayload | Award item to character |
| award_experience | AwardExperiencePayload | Award experience |
| award_mesos | AwardMesosPayload | Award mesos |
| award_fame | AwardFamePayload | Award fame |
| create_skill | CreateSkillPayload | Grant skill |
| destroy_asset | ConsumeItemPayload | Consume item |

### Monster Status Events (Consumed)

Consumed from EVENT_TOPIC_MONSTER_STATUS. Processes KILLED events to update monster kill progress for active quests.

### Character Status Events (Consumed)

Consumed from EVENT_TOPIC_CHARACTER_STATUS. Processes MAP_CHANGED events to:
- Update map visit progress for active quests
- Trigger auto-start quest checks for the new map
- Check for auto-complete after progress updates

## Transaction Semantics

- Quest commands carry a transactionId for saga correlation
- Status events include the originating transactionId
- Saga commands generate a new transactionId for reward processing
- Message ordering is guaranteed per character (partitioned by characterId)
