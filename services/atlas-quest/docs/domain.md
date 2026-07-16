# Quest Domain

## Responsibility

Manages quest state and progress tracking for characters. Handles quest lifecycle operations including starting, completing, and forfeiting quests. Tracks progress for quest objectives such as monster kills and map visits.

## Core Models

### Model

Represents a character's quest status.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Internal record identifier |
| characterId | uint32 | Character identifier |
| questId | uint32 | Quest definition identifier |
| state | State | Current quest state |
| startedAt | time.Time | When quest was started |
| completedAt | time.Time | When quest was completed |
| expirationTime | time.Time | When quest expires (time-limited quests) |
| completedCount | uint32 | Times completed (repeatable quests) |
| forfeitCount | uint32 | Times forfeited |
| progress | []progress.Model | Progress entries |

### progress.Model

Represents progress for a single quest objective.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Internal record identifier |
| infoNumber | uint32 | Objective identifier (monster ID or map ID) |
| progress | string | Progress value |

### medal.Model

Represents a visited map for a medal quest.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Internal record identifier |
| mapId | _map.Id | Map identifier |

Not referenced by any Processor and its Migration is not registered in main.go's migration list.

### State

| Value | Constant | Description |
|-------|----------|-------------|
| 0 | StateNotStarted | Quest not started or forfeited |
| 1 | StateStarted | Quest in progress |
| 2 | StateCompleted | Quest completed |

## Invariants

- A quest can only be completed if in StateStarted
- A quest can only be forfeited if in StateStarted
- An expired quest cannot be completed
- A repeatable quest cannot be restarted until the interval has elapsed since last completion
- A non-repeatable completed quest cannot be restarted
- Progress can only be updated for quests in StateStarted
- A quest cannot be started unless its start requirements are met, unless validation is explicitly skipped
- A quest cannot be completed unless its end requirements are met, unless validation is explicitly skipped
- A progress write for a given objective (infoNumber) is skipped once that objective's end requirement is already satisfied

## State Transitions

```
StateNotStarted -> StateStarted     (Start)
StateStarted    -> StateCompleted   (Complete)
StateStarted    -> StateNotStarted  (Forfeit)
StateCompleted  -> StateStarted     (Restart, repeatable quests only, after interval)
```

## Processors

### Processor

Manages quest state operations.

| Method | Description |
|--------|-------------|
| WithTransaction | Returns processor bound to a transaction context |
| ByIdProvider | Returns provider for quest status by internal ID |
| ByCharacterIdPagedProvider | Returns provider for one page of a character's quest statuses |
| ByCharacterIdAndQuestIdProvider | Returns provider for specific quest status for a character |
| ByCharacterIdAndStateProvider | Returns provider for all quest statuses by state for a character |
| ByCharacterIdAndStatePagedProvider | Returns provider for one page of a character's quest statuses by state |
| GetById | Retrieves quest status by internal ID |
| GetByCharacterIdAndQuestId | Retrieves specific quest status for a character |
| GetByCharacterIdAndState | Retrieves all quest statuses by state for a character |
| Start | Starts a quest with optional validation and processes start actions |
| StartChained | Starts a quest as part of a chain (skips interval check) |
| Complete | Completes a quest with optional validation and processes rewards via saga |
| Forfeit | Forfeits a quest |
| SetProgress | Updates progress for a specific objective |
| DeleteByCharacterId | Deletes all quest data for a character |
| GetQuestDefinition | Fetches quest definition from atlas-data |
| CheckAutoComplete | Checks if quest can be auto-completed and completes if requirements met |
| CheckAutoStart | Checks for auto-start quests on map entry |

### EventEmitter

Emits quest-related events to Kafka via the transactional outbox.

| Method | Description |
|--------|-------------|
| EmitQuestStarted | Emits quest started event |
| EmitQuestCompleted | Emits quest completed event with awarded items |
| EmitQuestForfeited | Emits quest forfeited event |
| EmitProgressUpdated | Emits quest progress updated event |
| EmitSaga | Emits saga command for rewards processing |

## Quest Definition (data/quest)

### Responsibility

Fetches quest definitions and auto-start quest lists from atlas-data for use by the Quest Domain's Processor.

### Processors

| Method | Description |
|--------|-------------|
| GetQuestDefinition | Fetches the full quest definition for a quest ID from atlas-data |
| GetAutoStartQuests | Fetches all auto-start quest definitions, optionally filtered by map |

## Validation (data/validation)

### Responsibility

Validates a character's start/end quest requirements. Date range, day-of-week, and interval requirements are checked locally; level, job, fame, meso, item, prerequisite-quest, selected-skill, monster-book-count, and pet-tameness requirements are submitted as conditions to query-aggregator.

### Processors

| Method | Description |
|--------|-------------|
| ValidateStartRequirements | Checks whether a character meets a quest's start requirements |
| ValidateEndRequirements | Checks whether a character meets a quest's end requirements |
