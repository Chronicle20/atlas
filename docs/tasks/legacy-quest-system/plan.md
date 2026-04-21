# Quest System Implementation Plan

**Last Updated:** 2026-01-05

## Executive Summary

This plan outlines the implementation of a comprehensive quest system for the Atlas MapleStory server. The system will follow established architectural patterns (DDD, immutable models, event-driven communication) and integrate with existing services including atlas-data, atlas-npc-conversations, atlas-saga-orchestrator, and atlas-query-aggregator.

### Key Design Decisions
- **Data Source**: Quest definitions loaded from WZ/XML via atlas-data, stored in document database, cached in memory
- **Progress Tracking**: Event-driven via Kafka (monster deaths, inventory changes, map enters)
- **Dialog Ownership**: atlas-npc-conversations owns all dialog flow; quest service tracks state only
- **Scope**: Solo quests only (party quests deferred to future phase)

---

## Current State Analysis

### Existing Infrastructure
1. **atlas-data**: Handles WZ/XML parsing with document storage pattern (registry + PostgreSQL)
2. **atlas-npc-conversations**: JSON state machine for NPC dialogs with operation/condition support
3. **atlas-saga-orchestrator**: Distributed transaction coordination for multi-step operations
4. **atlas-query-aggregator**: Cross-service validation with existing quest condition stubs
5. **Kafka messaging**: Established command/event patterns for service communication

### Existing Quest-Related Code
- `atlas-query-aggregator/quest/`: Contains `QuestStatus` enum and `Model` with stub processor
- NPC conversations support `questStatus` and `questProgress` conditions (not implemented)
- NPC conversations have `complete_quest` operation stub

### Gap Analysis
| Component | Current State | Required |
|-----------|---------------|----------|
| Quest definitions | Not implemented | WZ/XML parsing in atlas-data |
| Quest state storage | Not implemented | New atlas-quest service |
| Progress tracking | Not implemented | Kafka event consumers |
| NPC operations | Stub only | Full implementation |
| Query conditions | Stub only | Full implementation |

---

## Proposed Architecture

### Service Overview

```
                                    ┌─────────────────┐
                                    │   atlas-data    │
                                    │  (Quest Defs)   │
                                    └────────┬────────┘
                                             │ REST
                                             ▼
┌─────────────────┐    REST/Kafka    ┌─────────────────┐    Kafka Events
│ atlas-npc-conv  │◄────────────────►│  atlas-quest    │◄──────────────────┐
│   (Dialog)      │                  │   (NEW)         │                   │
└────────┬────────┘                  └────────┬────────┘                   │
         │                                    │                            │
         │ Saga                               │ REST                       │
         ▼                                    ▼                            │
┌─────────────────┐                  ┌─────────────────┐    ┌─────────────────┐
│ atlas-saga-orch │                  │ query-aggregator│    │ atlas-monsters  │
│   (Rewards)     │                  │  (Validation)   │    │ atlas-inventory │
└─────────────────┘                  └─────────────────┘    │ atlas-maps      │
                                                           └─────────────────┘
```

### New Service: atlas-quest

A dedicated quest state management service responsible for:
- Quest status tracking (NOT_STARTED, STARTED, COMPLETED)
- Progress tracking (mob kills, items collected, maps visited)
- Quest lifecycle operations (start, complete, forfeit)
- Event emission for UI updates

### Data Flow

**Quest Start Flow:**
1. Player interacts with NPC → atlas-npc-conversations
2. NPC conversation evaluates conditions via atlas-query-aggregator
3. Query-aggregator checks quest prerequisites via atlas-quest REST
4. If valid, NPC conversation sends `start_quest` command via Kafka
5. atlas-quest creates quest status record, initializes progress
6. atlas-quest emits `QuestStarted` event
7. atlas-saga-orchestrator executes start actions (if any)

**Progress Update Flow:**
1. Monster dies → atlas-monster-death emits event
2. atlas-quest consumes event, checks active quests needing that mob
3. Updates progress, emits `QuestProgressUpdated` event
4. If auto-complete conditions met, triggers completion

**Quest Complete Flow:**
1. Player returns to NPC → atlas-npc-conversations
2. NPC conversation sends `complete_quest` command to atlas-quest
3. atlas-quest validates completion requirements (mob kills, items, etc.)
4. If valid, atlas-quest initiates saga for reward distribution
5. atlas-saga-orchestrator executes reward steps (EXP, items, meso)
6. atlas-quest marks quest COMPLETED, emits event
7. If nextQuest defined, auto-starts chain quest

---

## Implementation Phases

### Phase 1: Quest Data Infrastructure (atlas-data)
Add quest definition parsing and storage to atlas-data service.

**Components:**
- Quest definition models (QuestInfo, QuestRequirements, QuestActions)
- WZ/XML readers for Quest.wz files
- Document storage and REST endpoints
- Worker registration for quest data loading

### Phase 2: Quest Service Core (atlas-quest)
Create the new atlas-quest service with core functionality.

**Components:**
- Quest status domain model (immutable)
- Quest progress tracking model
- GORM entities and migrations
- REST API for status/progress queries
- Kafka command consumer (start, complete, forfeit)
- Kafka event producer (status changes, progress updates)

### Phase 3: Service Integration
Connect atlas-quest with existing services.

**Components:**
- Update atlas-query-aggregator quest processor
- Implement NPC conversation operations (start_quest, complete_quest, forfeit_quest)
- Add saga orchestrator actions for quest rewards
- Implement progress event consumers (monster death, item gain, map enter)

### Phase 4: Advanced Features
Implement complex quest mechanics.

**Components:**
- Repeatable quests with interval tracking
- Time-limited quests with expiration
- Quest chains (nextQuest auto-start)
- Medal quests (exploration tracking)
- Auto-start/auto-complete quests

### Phase 5: Admin UI (atlas-ui)
Add quest management views to the administrative interface.

**Components:**
- Quest definitions browser (list all quests in tenant)
- Quest detail view (requirements, rewards, metadata)
- Character quest status view (available, started, completed)
- Quest search and filtering

---

## Detailed Task Breakdown

### Phase 1: Quest Data Infrastructure

#### 1.1 Quest Definition Models (Effort: M)

Create domain models for quest definitions in atlas-data:

**QuestInfo Model:**
```go
type QuestInfo struct {
    id              uint32
    name            string
    parent          string  // Category
    autoStart       bool
    autoPreComplete bool
    autoComplete    bool
    timeLimit       uint32  // seconds
    timeLimit2      uint32  // milliseconds
    viewMedalItem   uint32  // Medal item ID
    area            uint32
    order           uint32
}
```

**QuestRequirement Model:**
```go
type Requirement struct {
    requirementType RequirementType
    // Fields vary by type - use interface or tagged union
}

type RequirementType int
const (
    ReqMinLevel RequirementType = iota
    ReqMaxLevel
    ReqJob
    ReqItem
    ReqMob
    ReqQuest
    ReqNpc
    ReqInterval
    ReqFieldEnter
    ReqMeso
    ReqFame
    // ... etc
)
```

**QuestAction Model:**
```go
type Action struct {
    actionType ActionType
    // Fields vary by type
}

type ActionType int
const (
    ActExp ActionType = iota
    ActMeso
    ActItem
    ActSkill
    ActFame
    ActBuff
    ActNextQuest
    // ... etc
)
```

**Acceptance Criteria:**
- [ ] QuestInfo model with all metadata fields
- [ ] QuestRequirement model supporting all requirement types
- [ ] QuestAction model supporting all action types
- [ ] Proper JSON serialization for document storage

#### 1.2 WZ/XML Readers (Effort: L)

Implement XML parsing for quest data files:

**Files to Parse:**
- `Quest.wz/QuestInfo.img.xml` - Quest metadata
- `Quest.wz/Check.img.xml` - Start/completion requirements
- `Quest.wz/Act.img.xml` - Start/completion actions
- `Quest.wz/Say.img.xml` - Default dialog text (optional)

**Implementation:**
```go
// QuestInfo reader
func ReadQuestInfo(path string) ([]QuestInfo, error)

// Check requirements reader
func ReadQuestCheck(path string) (map[uint32]QuestCheckData, error)

// Actions reader
func ReadQuestAct(path string) (map[uint32]QuestActData, error)
```

**Acceptance Criteria:**
- [ ] Parse QuestInfo.img.xml with all metadata fields
- [ ] Parse Check.img.xml for both start (0) and complete (1) requirements
- [ ] Parse Act.img.xml for both start (0) and complete (1) actions
- [ ] Handle nested structures (job lists, item lists, mob lists)
- [ ] Parse probability-based rewards correctly
- [ ] Unit tests with sample XML data

#### 1.3 Quest Data Storage (Effort: S)

Implement document storage following existing pattern:

**Document Types:**
- `QUEST_INFO` - Quest metadata
- `QUEST_CHECK` - Quest requirements
- `QUEST_ACT` - Quest actions

**Acceptance Criteria:**
- [ ] Registry singleton for in-memory caching
- [ ] Database storage via document entity
- [ ] Two-tier lookup (registry → DB → null tenant)

#### 1.4 Quest Data REST API (Effort: S)

Implement REST endpoints for quest data:

**Endpoints:**
- `GET /api/data/quests` - List all quests (returns QuestInfo array)
- `GET /api/data/quests/{questId}` - Get full quest definition

**Full Quest Definition Response:**
```json
{
  "data": {
    "id": "2000",
    "type": "quests",
    "attributes": {
      "name": "Mai's First Training",
      "parent": "Maple Island",
      "autoStart": false,
      "autoComplete": false,
      "timeLimit": 0,
      "startRequirements": [...],
      "completeRequirements": [...],
      "startActions": [...],
      "completeActions": [...]
    }
  }
}
```

**Acceptance Criteria:**
- [ ] JSON:API compliant responses
- [ ] Single endpoint returns complete quest definition
- [ ] Proper error handling (404, 500)
- [ ] Tenant context support
- [ ] Route registration in main.go

#### 1.5 Worker Registration (Effort: S)

Register quest worker for data loading:

**Changes:**
- Add `QUEST` to Workers array
- Add `RegisterQuest` case in StartWorker
- Process Quest.wz directory files

**Acceptance Criteria:**
- [ ] Worker processes Quest.wz files on tenant init
- [ ] Parallel processing with existing pattern
- [ ] Error handling and logging

---

### Phase 2: Quest Service Core

#### 2.1 Service Scaffolding (Effort: S)

Create atlas-quest service structure:

```
services/atlas-quest/
├── atlas.com/quest/
│   ├── main.go
│   ├── database/
│   │   └── connection.go
│   ├── kafka/
│   │   ├── consumer/
│   │   └── producer/
│   ├── quest/
│   │   ├── model.go
│   │   ├── entity.go
│   │   ├── builder.go
│   │   ├── processor.go
│   │   ├── administrator.go
│   │   ├── provider.go
│   │   ├── producer.go
│   │   ├── resource.go
│   │   └── rest.go
│   └── progress/
│       ├── model.go
│       ├── entity.go
│       └── processor.go
├── Dockerfile
├── go.mod
└── go.sum
```

**Acceptance Criteria:**
- [ ] Service compiles and runs
- [ ] Database connection established
- [ ] Kafka consumer/producer initialized
- [ ] REST server started

#### 2.2 Quest Status Model (Effort: M)

Implement immutable quest status domain model:

```go
type Model struct {
    id             uuid.UUID
    characterId    uint32
    questId        uint32
    status         Status
    progress       map[uint32]string  // mobId/itemId → progress
    medalMaps      []uint32           // Visited map IDs
    npcId          uint32
    completionTime int64
    expirationTime int64
    forfeitCount   uint32
    completedCount uint32
    customData     string
    infoNumber     uint32
}

type Status int
const (
    NotStarted Status = iota
    Started
    Completed
)
```

**Acceptance Criteria:**
- [ ] Immutable model with private fields
- [ ] Public accessor methods
- [ ] Builder pattern for construction
- [ ] CloneModel for modifications

#### 2.3 Quest Entities (Effort: M)

Implement GORM entities:

```go
// queststatus table
type questStatusEntity struct {
    TenantId      uuid.UUID
    Id            uuid.UUID `gorm:"primaryKey"`
    CharacterId   uint32    `gorm:"index;not null"`
    QuestId       uint32    `gorm:"not null"`
    Status        int8      `gorm:"not null;default=0"`
    NpcId         uint32
    CompletionTime int64
    ExpirationTime int64
    ForfeitCount  uint32    `gorm:"not null;default=0"`
    CompletedCount uint32   `gorm:"not null;default=0"`
    CustomData    string
    InfoNumber    uint32
}

// questprogress table
type questProgressEntity struct {
    TenantId        uuid.UUID
    Id              uuid.UUID `gorm:"primaryKey"`
    QuestStatusId   uuid.UUID `gorm:"index;not null"`
    ProgressId      uint32    `gorm:"not null"`  // mobId or itemId
    Progress        string    `gorm:"not null"`  // "000" format
}

// medalmaps table
type medalMapEntity struct {
    TenantId      uuid.UUID
    Id            uuid.UUID `gorm:"primaryKey"`
    QuestStatusId uuid.UUID `gorm:"index;not null"`
    MapId         uint32    `gorm:"not null"`
}
```

**Acceptance Criteria:**
- [ ] All entities with proper GORM tags
- [ ] Auto-migration on service start
- [ ] Foreign key relationships (CASCADE delete)
- [ ] Multi-tenant support with TenantId

#### 2.4 Quest Processor (Effort: L)

Implement quest business logic:

```go
type Processor interface {
    // Queries
    GetByCharacterAndQuest(characterId, questId uint32) model.Provider[Model]
    GetByCharacter(characterId uint32) model.Provider[[]Model]
    GetActiveQuests(characterId uint32) model.Provider[[]Model]

    // Commands
    StartQuest(characterId, questId, npcId uint32) model.Provider[Model]
    CompleteQuest(characterId, questId, npcId uint32) model.Provider[Model]
    ForfeitQuest(characterId, questId uint32) model.Provider[Model]

    // Progress
    UpdateMobProgress(characterId, questId, mobId uint32) model.Provider[Model]
    UpdateMedalProgress(characterId, questId, mapId uint32) model.Provider[Model]
    SetCustomProgress(characterId, questId uint32, key, value string) model.Provider[Model]
}
```

**Acceptance Criteria:**
- [ ] All query methods implemented
- [ ] Start quest creates status with progress initialization
- [ ] Complete quest validates and updates status
- [ ] Forfeit quest resets status
- [ ] Progress updates with proper formatting ("000")
- [ ] Transaction safety for all mutations

#### 2.5 Quest REST API (Effort: M)

Implement REST endpoints:

**Endpoints:**
- `GET /api/characters/{characterId}/quests` - All quests for character
- `GET /api/characters/{characterId}/quests/{questId}` - Specific quest status
- `GET /api/characters/{characterId}/quests/{questId}/progress` - Quest progress
- `POST /api/characters/{characterId}/quests/{questId}/start` - Start quest
- `POST /api/characters/{characterId}/quests/{questId}/complete` - Complete quest
- `DELETE /api/characters/{characterId}/quests/{questId}` - Forfeit quest

**Acceptance Criteria:**
- [ ] JSON:API compliant responses
- [ ] Proper HTTP status codes
- [ ] Tenant context extraction
- [ ] Input validation

#### 2.6 Kafka Integration (Effort: M)

Implement Kafka messaging:

**Command Topic:** `COMMAND_TOPIC_QUEST`
```go
type Command int
const (
    CommandStartQuest Command = iota
    CommandCompleteQuest
    CommandForfeitQuest
    CommandUpdateProgress
)

type CommandBody struct {
    TransactionId uuid.UUID
    CharacterId   uint32
    QuestId       uint32
    NpcId         uint32
    Command       Command
    ProgressData  map[string]interface{}
}
```

**Event Topic:** `EVENT_TOPIC_QUEST_STATUS`
```go
type StatusEventType int
const (
    StatusEventStarted StatusEventType = iota
    StatusEventCompleted
    StatusEventForfeited
    StatusEventProgressUpdated
    StatusEventExpired
)

type StatusEvent struct {
    TransactionId uuid.UUID
    CharacterId   uint32
    QuestId       uint32
    Type          StatusEventType
    Body          json.RawMessage
}
```

**Acceptance Criteria:**
- [ ] Command consumer processes all command types
- [ ] Events emitted for all status changes
- [ ] Transaction ID propagation
- [ ] Error handling with logging

---

### Phase 3: Service Integration

#### 3.1 Query Aggregator Integration (Effort: M)

Update atlas-query-aggregator quest processor:

**Changes:**
- Implement `GetQuestStatus` with REST call to atlas-quest
- Implement `GetQuestProgress` with REST call to atlas-quest
- Add environment variable for atlas-quest base URL

**Acceptance Criteria:**
- [ ] Quest status lookups work via REST
- [ ] Quest progress lookups work via REST
- [ ] Proper error handling for service unavailable
- [ ] Caching considerations

#### 3.2 NPC Conversation Operations (Effort: M)

Implement quest operations in atlas-npc-conversations:

**Operations:**
- `start_quest` - Send start command to atlas-quest
- `complete_quest` - Trigger saga for completion with rewards
- `forfeit_quest` - Send forfeit command to atlas-quest
- `update_quest_progress` - Direct progress update

**Acceptance Criteria:**
- [ ] Operations integrated with saga orchestrator
- [ ] Proper parameter extraction from conversation context
- [ ] Error handling with conversation state recovery
- [ ] Context updates with quest status

#### 3.3 Saga Orchestrator Actions (Effort: L)

Add quest-related saga actions for reward distribution:

**Note:** Validation of quest requirements is handled by atlas-quest, not the saga orchestrator. The saga orchestrator's role is to coordinate the distributed transaction for distributing rewards across services.

**New Actions:**
- `mark_quest_completed` - Update quest status to COMPLETED (final step after rewards)

**Reward Actions (existing, verify working):**
- `award_experience` - EXP rewards
- `award_mesos` - Meso rewards
- `award_asset` - Item rewards
- `destroy_asset` - Remove required items from inventory
- `create_skill` - Skill rewards

**Typical Quest Completion Saga (created by atlas-quest):**
```
1. destroy_asset (remove quest items if required)
2. award_experience
3. award_mesos
4. award_asset (give reward items)
5. create_skill (if skill reward)
6. mark_quest_completed
```

**Acceptance Criteria:**
- [ ] mark_quest_completed action registered
- [ ] Actions emit appropriate events for saga tracking
- [ ] Compensation logic for rollback (e.g., if item award fails, rollback EXP)
- [ ] Integration tests for reward distribution

#### 3.4 Progress Event Consumers (Effort: L)

Implement event consumers for progress tracking:

**Monster Death Consumer:**
```go
// Listen: EVENT_TOPIC_DROP_STATUS (or monster death topic)
// Check if dead monster is tracked by any active quest
// Update progress for matching quests
```

**Item Change Consumer:**
```go
// Listen: EVENT_TOPIC_ASSET_STATUS (CREATED, QUANTITY_CHANGED, DELETED)
// Check if gained/lost item affects quest requirements
// Used for item collection quests
```

**Map Enter Consumer:**
```go
// Listen: EVENT_TOPIC_CHARACTER_STATUS (map changed)
// Check if map is tracked by medal quest
// Update medal progress
```

**Acceptance Criteria:**
- [ ] Monster kills update mob progress
- [ ] Progress capped at requirement count
- [ ] Medal map visits tracked
- [ ] Auto-complete triggered when conditions met

---

### Phase 4: Advanced Features

#### 4.1 Repeatable Quests (Effort: M)

Implement interval-based quest repetition:

**Logic:**
- Check `interval` requirement in quest definition
- On completion, record timestamp
- On start attempt, verify interval elapsed since last completion
- Reset quest status to NOT_STARTED when repeatable

**Acceptance Criteria:**
- [ ] Interval requirement checked on start
- [ ] Completion timestamp recorded
- [ ] Quest can be repeated after interval
- [ ] Progress reset on new start

#### 4.2 Time-Limited Quests (Effort: M)

Implement quest expiration:

**Logic:**
- Read `timeLimit` or `timeLimit2` from quest definition
- Set expiration timestamp on quest start
- Background job or event-driven expiration check
- Quest fails if time expires before completion

**Acceptance Criteria:**
- [ ] Expiration set on timed quest start
- [ ] Expiration timestamp sent to client
- [ ] Quest auto-fails on expiration
- [ ] Expired status event emitted

#### 4.3 Quest Chains (Effort: S)

Implement nextQuest auto-start:

**Logic:**
- Read `nextQuest` action from completion actions
- After quest completion, auto-start next quest
- Bypass normal start requirements for chain quests

**Acceptance Criteria:**
- [ ] NextQuest detected in completion actions
- [ ] Chained quest auto-started
- [ ] Chain continues through multiple quests
- [ ] Proper event emission for each quest

#### 4.4 Medal Quests (Effort: M)

Implement exploration-based quests:

**Logic:**
- Detect medal quests via `viewMedalItem` field
- Track map visits in medalMaps table
- Completion requires visiting all specified maps
- Special UI handling (medal display)

**Acceptance Criteria:**
- [ ] Medal quests identified correctly
- [ ] Map visits tracked on character map change
- [ ] Completion requires all maps visited
- [ ] Medal item ID returned for UI

#### 4.5 Auto Quests (Effort: S)

Implement auto-start and auto-complete:

**Logic:**
- `autoStart`: Quest starts when all start requirements met (no NPC needed)
- `autoPreComplete`: Quest pre-completes automatically
- `autoComplete`: Quest completes when all completion requirements met

**Acceptance Criteria:**
- [ ] Auto-start triggered on condition met (level up, map enter, etc.)
- [ ] Auto-complete triggered when requirements satisfied
- [ ] NPC proximity validation skipped for auto quests
- [ ] Proper event emission

---

### Phase 5: Admin UI

#### 5.1 Quest Service Layer (Effort: S)

Create service layer for quest API calls in atlas-ui:

**Files to Create:**
- `services/api/quests.service.ts` - Quest definition API calls
- `services/api/quest-status.service.ts` - Character quest status API calls
- `types/models/quest.ts` - Quest TypeScript types

**Quest Definition Types:**
```typescript
interface QuestInfo {
  id: string;
  name: string;
  parent: string;  // Category
  autoStart: boolean;
  autoPreComplete: boolean;
  autoComplete: boolean;
  timeLimit: number;
  viewMedalItem: number;
  area: number;
}

interface QuestRequirement {
  type: string;  // 'minLevel', 'job', 'item', 'mob', 'quest', etc.
  // Type-specific fields
  value?: number;
  values?: number[];  // For job lists
  items?: { id: number; count: number }[];
  mobs?: { id: number; count: number }[];
}

interface QuestAction {
  type: string;  // 'exp', 'meso', 'item', 'skill', 'nextQuest', etc.
  // Type-specific fields
  value?: number;
  items?: { id: number; count: number; prop?: number }[];
}

interface QuestDefinition {
  info: QuestInfo;
  startRequirements: QuestRequirement[];
  completeRequirements: QuestRequirement[];
  startActions: QuestAction[];
  completeActions: QuestAction[];
}
```

**Character Quest Status Types:**
```typescript
interface CharacterQuestStatus {
  id: string;
  questId: number;
  status: 'NOT_STARTED' | 'STARTED' | 'COMPLETED';
  progress: Record<string, string>;
  npcId?: number;
  completionTime?: number;
  expirationTime?: number;
  forfeitCount: number;
  completedCount: number;
}
```

**Acceptance Criteria:**
- [ ] QuestsService with getAll, getById methods
- [ ] QuestStatusService with getByCharacter method
- [ ] TypeScript types for all quest models
- [ ] Proper tenant header injection

#### 5.2 Quest List View (Effort: M)

Create quest browser page for viewing all quests in a tenant:

**Route:** `/app/quests/page.tsx`

**Features:**
- DataTableWrapper with quest list
- Columns: ID, Name, Category, Level Range, Auto flags
- Sorting by ID, name, category, level
- Search by quest name
- Filter by category (parent field)
- Click row to navigate to detail view

**Column Definitions:**
```typescript
const columns: ColumnDef<QuestInfo>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'name', header: 'Name' },
  { accessorKey: 'parent', header: 'Category' },
  {
    accessorKey: 'levelRange',
    header: 'Level',
    cell: ({ row }) => `${row.minLevel ?? '?'} - ${row.maxLevel ?? '?'}`
  },
  {
    accessorKey: 'flags',
    header: 'Flags',
    cell: ({ row }) => (
      <div className="flex gap-1">
        {row.autoStart && <Badge>Auto Start</Badge>}
        {row.autoComplete && <Badge>Auto Complete</Badge>}
        {row.timeLimit > 0 && <Badge variant="outline">Timed</Badge>}
      </div>
    )
  },
];
```

**Acceptance Criteria:**
- [ ] Quest list loads from atlas-data API
- [ ] Table displays all quest metadata
- [ ] Search filters by quest name
- [ ] Category filter dropdown
- [ ] Row click navigates to detail
- [ ] Loading and error states

#### 5.3 Quest Detail View (Effort: L)

Create detailed quest information page:

**Route:** `/app/quests/[id]/page.tsx`

**Layout:**
```
┌─────────────────────────────────────────────────────┐
│ Quest #2000: Mai's First Training                   │
│ Category: Maple Island                              │
├─────────────────────────────────────────────────────┤
│ [Metadata Card]                                     │
│ Auto Start: No | Auto Complete: No | Time Limit: - │
│ Medal Item: - | Area: 0 | Order: 1                  │
├─────────────────────────────────────────────────────┤
│ ▼ Start Requirements                                │
│   • Level: 1-10                                     │
│   • NPC: Mai (1002000)                              │
├─────────────────────────────────────────────────────┤
│ ▼ Completion Requirements                           │
│   • Kill 3x Snail (100100)                          │
├─────────────────────────────────────────────────────┤
│ ▼ Start Actions                                     │
│   (none)                                            │
├─────────────────────────────────────────────────────┤
│ ▼ Completion Rewards                                │
│   • 10 EXP                                          │
│   • 5x Red Potion (2000000)                         │
└─────────────────────────────────────────────────────┘
```

**Components:**
- Hero card with quest name and category
- Metadata card with flags and settings
- Collapsible sections for requirements/actions
- Requirement renderer (handles all requirement types)
- Action/reward renderer (handles all action types)
- Link to related entities (NPCs, items, monsters)

**Acceptance Criteria:**
- [ ] Load quest info, requirements, actions from API
- [ ] Display all metadata fields
- [ ] Render all requirement types with readable format
- [ ] Render all action types with readable format
- [ ] Link to related NPCs, items, monsters
- [ ] Breadcrumb navigation
- [ ] Loading skeleton

#### 5.4 Character Quest Status View (Effort: L)

Add quest status section to character detail page:

**Location:** `/app/characters/[id]/page.tsx` (extend existing)

**Layout:**
```
┌─────────────────────────────────────────────────────┐
│ ▼ Quest Status                                      │
├─────────────────────────────────────────────────────┤
│ [Tabs: Started | Completed | Available]             │
├─────────────────────────────────────────────────────┤
│ Started Quests (3)                                  │
│ ┌─────────────────────────────────────────────────┐ │
│ │ Quest #2000: Mai's First Training               │ │
│ │ Progress: 1/3 Snails killed                     │ │
│ │ [View Quest]                                    │ │
│ └─────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────┐ │
│ │ Quest #2001: Collecting Mushroom Caps           │ │
│ │ Progress: 5/10 items collected                  │ │
│ │ [View Quest]                                    │ │
│ └─────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────┤
│ Completed Quests (47)                               │
│ [Paginated list of completed quests]                │
├─────────────────────────────────────────────────────┤
│ Available Quests (12)                               │
│ [Quests where requirements are met but not started] │
└─────────────────────────────────────────────────────┘
```

**Features:**
- Tab interface for Started/Completed/Available
- Started quests show progress
- Completed quests show completion time
- Available quests calculated by checking requirements
- Link to quest detail view
- Quest counts in tab headers

**Acceptance Criteria:**
- [ ] Fetch character quest statuses from atlas-quest
- [ ] Display started quests with progress
- [ ] Display completed quests with timestamps
- [ ] Calculate available quests (requirements met, not started)
- [ ] Tab navigation with counts
- [ ] Pagination for long lists
- [ ] Link to quest detail view

#### 5.5 Navigation Integration (Effort: S)

Add quests to sidebar navigation:

**Changes to `/components/app-sidebar.tsx`:**
```typescript
{
  title: "Operations",
  icon: Cog,
  children: [
    { title: "Accounts", url: "/accounts" },
    { title: "Characters", url: "/characters" },
    { title: "Guilds", url: "/guilds" },
    { title: "NPCs", url: "/npcs" },
    { title: "Quests", url: "/quests" },  // NEW
  ],
}
```

**Breadcrumb Support:**
- `/quests` → "Quests"
- `/quests/[id]` → "Quests > Quest #[id]"

**Acceptance Criteria:**
- [ ] Quests link in sidebar navigation
- [ ] Breadcrumb support for quest routes
- [ ] Active state highlighting

#### 5.6 Requirement/Reward Renderers (Effort: M)

Create reusable components for displaying quest requirements and rewards:

**RequirementRenderer Component:**
```typescript
// Renders any requirement type in human-readable format
<RequirementRenderer requirement={req} />

// Examples:
// { type: 'minLevel', value: 10 } → "Level 10+"
// { type: 'job', values: [100, 110] } → "Job: Warrior, Fighter"
// { type: 'item', items: [{id: 4000001, count: 5}] } → "5x Blue Snail Shell"
// { type: 'mob', mobs: [{id: 100100, count: 10}] } → "Kill 10x Snail"
// { type: 'quest', quests: [{id: 1999, state: 2}] } → "Complete Quest #1999"
```

**RewardRenderer Component:**
```typescript
// Renders any action/reward type
<RewardRenderer action={act} />

// Examples:
// { type: 'exp', value: 100 } → "100 EXP"
// { type: 'meso', value: 500 } → "500 Mesos"
// { type: 'item', items: [{id: 2000000, count: 5}] } → "5x Red Potion"
// { type: 'skill', skills: [{id: 1000, level: 1}] } → "Skill: Three Snails Lv.1"
// { type: 'nextQuest', value: 2001 } → "→ Starts Quest #2001"
```

**Entity Resolution:**
- Fetch item names from atlas-data
- Fetch monster names from atlas-data
- Fetch NPC names from atlas-data
- Fetch job names from constants
- Cache lookups for performance

**Acceptance Criteria:**
- [ ] RequirementRenderer handles all requirement types
- [ ] RewardRenderer handles all action types
- [ ] Entity names resolved and displayed
- [ ] Links to related entities
- [ ] Graceful handling of unknown IDs
- [ ] Loading states for async lookups

---

## Risk Assessment

### Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Quest data parsing complexity | Medium | High | Start with common quest types, iterate |
| Event ordering issues | Medium | Medium | Use transaction IDs, implement idempotency |
| Performance with many active quests | Low | Medium | Index optimization, caching |
| Saga failure handling | Medium | High | Thorough compensation logic testing |

### Integration Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking NPC conversations | Medium | High | Feature flag for quest operations |
| Query aggregator changes | Low | Medium | Backward compatible API |
| Kafka topic conflicts | Low | Low | New dedicated topics |

---

## Success Metrics

### Functional Metrics
- [ ] 100% of quest types from documentation parseable
- [ ] Quest start/complete/forfeit working end-to-end
- [ ] Progress tracking accurate for mob kills
- [ ] Rewards distributed correctly via saga

### Performance Metrics
- [ ] Quest status lookup < 50ms
- [ ] Progress update < 100ms
- [ ] Support 1000+ active quests per character

### Quality Metrics
- [ ] Unit test coverage > 80%
- [ ] Integration tests for all flows
- [ ] No data corruption in stress tests

---

## Dependencies

### External Dependencies
- PostgreSQL database
- Kafka cluster
- atlas-data service running
- atlas-saga-orchestrator service running

### Internal Dependencies
- atlas-model library (Provider pattern)
- atlas-kafka library (messaging)
- atlas-rest library (JSON:API)
- atlas-tenant library (multi-tenancy)

### Data Dependencies
- Quest.wz XML files for quest definitions
- Monster data for mob kill tracking
- Item data for item requirements/rewards
- Map data for medal quests

---

## Resolved Questions

1. **Quest Point System**: ~~Should we implement quest points in initial release?~~ **No** - Dropped from scope. Not a known requirement.

2. **Quest UI Packets**: ~~What client packets are needed for quest UI updates?~~ **Handled by atlas-channel** - Out of scope for this plan.

3. **InfoNumber System**: ~~How common are quests using alternate tracking IDs?~~ **Moderately common** - 132 usages across 67 unique values. Include in Phase 2. Implementation: use `infoNumber` as progress lookup key when set.

4. **Scripted Quests**: ~~Should we support JavaScript quest scripts like the reference implementation?~~ **No** - atlas-npc-conversations JSON state machines should suffice. Revisit if needed during NPC script conversion.

---

## Appendix: Kafka Topic Summary

### New Topics
| Topic | Direction | Purpose |
|-------|-----------|---------|
| `COMMAND_TOPIC_QUEST` | In | Quest commands (start, complete, forfeit) |
| `EVENT_TOPIC_QUEST_STATUS` | Out | Quest status changes |

### Consumed Topics
| Topic | Purpose |
|-------|---------|
| `EVENT_TOPIC_DROP_STATUS` | Monster death for mob kill tracking |
| `EVENT_TOPIC_ASSET_STATUS` | Item gained/lost for item collection tracking |
| `EVENT_TOPIC_CHARACTER_STATUS` | Map changes for medal tracking |
