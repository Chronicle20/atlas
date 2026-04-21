# Quest System - Task Checklist

**Last Updated:** 2026-01-05

Use this checklist to track implementation progress. Check off items as they are completed.

---

## Phase 1: Quest Data Infrastructure (atlas-data)

### 1.1 Quest Definition Models
**Effort:** M | **Dependencies:** None

- [ ] Create `services/atlas-data/atlas.com/data/quest/` package directory
- [ ] Implement `quest_info.go` - QuestInfo RestModel
  - [ ] id, name, parent fields
  - [ ] autoStart, autoPreComplete, autoComplete flags
  - [ ] timeLimit, timeLimit2 fields
  - [ ] viewMedalItem, area, order fields
  - [ ] JSON:API interface methods (GetID, SetID, GetName)
- [ ] Implement `quest_check.go` - QuestRequirements RestModel
  - [ ] RequirementType enum
  - [ ] MinLevel, MaxLevel requirement
  - [ ] Job requirement (list of job IDs)
  - [ ] Item requirement (id, count pairs)
  - [ ] Mob requirement (id, count pairs)
  - [ ] Quest prerequisite requirement (id, state pairs)
  - [ ] NPC requirement
  - [ ] Interval requirement (milliseconds)
  - [ ] FieldEnter, Meso, Fame requirements
  - [ ] Script, InfoNumber, InfoEx requirements
- [ ] Implement `quest_act.go` - QuestActions RestModel
  - [ ] ActionType enum
  - [ ] Exp action (flat and job-specific)
  - [ ] Meso action
  - [ ] Item action (id, count, prop, gender, job)
  - [ ] Skill action (id, level, masterLevel, job)
  - [ ] Fame action
  - [ ] Buff action
  - [ ] NextQuest action
  - [ ] Info action

### 1.2 WZ/XML Readers
**Effort:** L | **Dependencies:** 1.1

- [ ] Implement `quest_info_reader.go`
  - [ ] Read QuestInfo.img.xml file
  - [ ] Parse quest ID from imgdir name attribute
  - [ ] Extract all metadata fields
  - [ ] Return slice of QuestInfo
  - [ ] Unit tests with sample XML
- [ ] Implement `quest_check_reader.go`
  - [ ] Read Check.img.xml file
  - [ ] Parse quest ID from imgdir name
  - [ ] Parse "0" child for start requirements
  - [ ] Parse "1" child for completion requirements
  - [ ] Handle nested structures (job, item, mob, quest imgdirs)
  - [ ] Unit tests with sample XML
- [ ] Implement `quest_act_reader.go`
  - [ ] Read Act.img.xml file
  - [ ] Parse quest ID from imgdir name
  - [ ] Parse "0" child for start actions
  - [ ] Parse "1" child for completion actions
  - [ ] Handle job-specific exp rewards
  - [ ] Handle probability-based item rewards
  - [ ] Handle gender-specific items
  - [ ] Unit tests with sample XML

### 1.3 Quest Data Storage
**Effort:** S | **Dependencies:** 1.1

- [ ] Create `registry.go` - Quest data registry singleton
  - [ ] QuestInfo registry
  - [ ] QuestCheck registry (start/complete separated)
  - [ ] QuestAct registry (start/complete separated)
- [ ] Implement storage functions
  - [ ] RegisterQuestInfo
  - [ ] RegisterQuestCheck
  - [ ] RegisterQuestAct
  - [ ] GetQuestInfo (by questId)
  - [ ] GetQuestCheck (by questId, phase)
  - [ ] GetQuestAct (by questId, phase)
- [ ] Document types: "QUEST_INFO", "QUEST_CHECK", "QUEST_ACT"

### 1.4 Quest Data REST API
**Effort:** S | **Dependencies:** 1.3

- [ ] Implement `resource.go` - REST route registration
  - [ ] GET /api/data/quests - List all quests
  - [ ] GET /api/data/quests/{questId} - Full quest definition
- [ ] Implement `rest.go` - Handler functions and models
  - [ ] QuestListRestModel (for list endpoint)
  - [ ] QuestDefinitionRestModel (includes info, requirements, actions)
  - [ ] HandleGetQuests (list)
  - [ ] HandleGetQuest (single, full definition)
  - [ ] Transform functions
- [ ] Error handling (404 for not found, 500 for errors)
- [ ] Tenant context extraction from headers

### 1.5 Worker Registration
**Effort:** S | **Dependencies:** 1.2, 1.3

- [ ] Update `data/processor.go`
  - [ ] Add "QUEST" to Workers array
  - [ ] Add RegisterQuest case in StartWorker
- [ ] Implement `quest/processor.go`
  - [ ] RegisterQuest function
  - [ ] Process QuestInfo.img.xml
  - [ ] Process Check.img.xml
  - [ ] Process Act.img.xml
  - [ ] Parallel processing with goroutines
- [ ] Update `main.go`
  - [ ] Import quest package
  - [ ] Add quest.InitResource route initializer
- [ ] Test with actual Quest.wz data upload

---

## Phase 2: Quest Service Core (atlas-quest)

### 2.1 Service Scaffolding
**Effort:** S | **Dependencies:** Phase 1

- [ ] Create `services/atlas-quest/` directory structure
  ```
  atlas-quest/
  ├── atlas.com/quest/
  │   ├── main.go
  │   ├── database/
  │   ├── kafka/
  │   ├── quest/
  │   └── progress/
  ├── Dockerfile
  ├── go.mod
  └── go.sum
  ```
- [ ] Implement `main.go`
  - [ ] Database connection initialization
  - [ ] Kafka consumer/producer setup
  - [ ] REST server initialization
  - [ ] Graceful shutdown handling
- [ ] Implement `database/connection.go`
  - [ ] GORM PostgreSQL connection
  - [ ] Connection pooling configuration
  - [ ] Auto-migration on startup
- [ ] Create `go.mod` with dependencies
- [ ] Create `Dockerfile`
- [ ] Verify service starts successfully

### 2.2 Quest Status Model
**Effort:** M | **Dependencies:** 2.1

- [ ] Implement `quest/model.go`
  - [ ] Status enum (NotStarted, Started, Completed)
  - [ ] Private fields for all quest status data
  - [ ] Public accessor methods
- [ ] Implement `quest/builder.go`
  - [ ] ModelBuilder struct
  - [ ] Set methods for all fields
  - [ ] Build() method
  - [ ] NewModelBuilder() constructor
  - [ ] CloneModel() function
- [ ] Unit tests for model and builder

### 2.3 Quest Entities
**Effort:** M | **Dependencies:** 2.2

- [ ] Implement `quest/entity.go`
  - [ ] questStatusEntity struct with GORM tags
  - [ ] TableName() method
  - [ ] Migration() function
- [ ] Implement `progress/entity.go`
  - [ ] questProgressEntity struct
  - [ ] Foreign key to quest_statuses
  - [ ] TableName() and Migration()
- [ ] Implement medal map entity (if needed in phase 4)
- [ ] Entity ↔ Model transformation functions
  - [ ] Make(entity) (Model, error)
  - [ ] model.ToEntity() entity
- [ ] Verify migrations create correct tables

### 2.4 Quest Processor
**Effort:** L | **Dependencies:** 2.3

- [ ] Implement `quest/processor.go`
  - [ ] Processor interface definition
  - [ ] processorImpl struct with dependencies
  - [ ] NewProcessor constructor
- [ ] Query methods
  - [ ] GetByCharacterAndQuest(characterId, questId)
  - [ ] GetByCharacter(characterId)
  - [ ] GetActiveQuests(characterId) - status = Started
  - [ ] GetCompletedQuests(characterId) - status = Completed
- [ ] Command methods
  - [ ] StartQuest(characterId, questId, npcId)
    - [ ] Validate quest not already started
    - [ ] Create quest status record
    - [ ] Initialize progress tracking
  - [ ] CompleteQuest(characterId, questId, npcId)
    - [ ] Validate quest is started
    - [ ] Update status to Completed
    - [ ] Set completion timestamp
    - [ ] Increment completed count
  - [ ] ForfeitQuest(characterId, questId)
    - [ ] Validate quest is started
    - [ ] Reset status to NotStarted
    - [ ] Clear progress
    - [ ] Increment forfeit count
- [ ] Implement `quest/administrator.go`
  - [ ] Transactional wrappers
  - [ ] Create, Update, Delete operations
- [ ] Implement `quest/provider.go`
  - [ ] Database query methods
  - [ ] Error handling (ErrNotFound)

### 2.5 Progress Processor
**Effort:** M | **Dependencies:** 2.4

- [ ] Implement `progress/model.go`
  - [ ] Progress model with questStatusId, progressId, progress
- [ ] Implement `progress/processor.go`
  - [ ] UpdateMobProgress(characterId, questId, mobId)
    - [ ] Format progress as 3-digit string
    - [ ] Cap at requirement count
  - [ ] GetProgress(characterId, questId)
  - [ ] InitializeProgress(questStatusId, mobRequirements)
- [ ] Progress string formatting ("000" to "999")
- [ ] InfoNumber support
  - [ ] When quest has infoNumber, use it as progress lookup key
  - [ ] Multiple quests can share progress via same infoNumber
- [ ] Unit tests for progress formatting

### 2.6 Quest REST API
**Effort:** M | **Dependencies:** 2.4, 2.5

- [ ] Implement `quest/resource.go`
  - [ ] Route registration
  - [ ] GET /api/characters/{characterId}/quests
  - [ ] GET /api/characters/{characterId}/quests/{questId}
  - [ ] GET /api/characters/{characterId}/quests/{questId}/progress
  - [ ] POST /api/characters/{characterId}/quests/{questId}/start
  - [ ] POST /api/characters/{characterId}/quests/{questId}/complete
  - [ ] DELETE /api/characters/{characterId}/quests/{questId}
- [ ] Implement `quest/rest.go`
  - [ ] RestModel for JSON:API
  - [ ] Transform function (Model → RestModel)
  - [ ] Handler functions
- [ ] Input validation
- [ ] Proper HTTP status codes (200, 201, 404, 400, 500)

### 2.7 Kafka Integration
**Effort:** M | **Dependencies:** 2.4, 2.5

- [ ] Add topic constants to `libs/atlas-kafka/kafka/topics.go`
  - [ ] COMMAND_TOPIC_QUEST
  - [ ] EVENT_TOPIC_QUEST_STATUS
- [ ] Implement `kafka/consumer/quest_command.go`
  - [ ] Command types enum
  - [ ] CommandBody struct
  - [ ] Consumer handler
  - [ ] Route to processor methods
- [ ] Implement `quest/producer.go`
  - [ ] StatusEventType enum
  - [ ] StatusEvent struct
  - [ ] QuestStartedEventProvider
  - [ ] QuestCompletedEventProvider
  - [ ] QuestForfeitedEventProvider
  - [ ] QuestProgressUpdatedEventProvider
- [ ] Wire up consumer in main.go
- [ ] Transaction ID propagation

---

## Phase 3: Service Integration

### 3.1 Query Aggregator Integration
**Effort:** M | **Dependencies:** Phase 2

- [ ] Update `atlas-query-aggregator/quest/processor.go`
  - [ ] Add QUEST_SERVICE_URL environment variable
  - [ ] Implement GetQuestStatus REST call
  - [ ] Implement GetQuestProgress REST call
  - [ ] Implement GetQuest REST call
- [ ] Error handling
  - [ ] Service unavailable returns UNDEFINED status
  - [ ] Timeout handling
- [ ] Add integration tests

### 3.2 NPC Conversation Operations
**Effort:** M | **Dependencies:** 3.1

- [ ] Update operation handling in atlas-npc-conversations
- [ ] Implement `start_quest` operation
  - [ ] Extract questId from params
  - [ ] Extract npcId from context
  - [ ] Send Kafka command or call saga
  - [ ] Update conversation context with result
- [ ] Implement `complete_quest` operation
  - [ ] Extract questId from params
  - [ ] Trigger completion saga
  - [ ] Wait for saga completion
  - [ ] Handle failure gracefully
- [ ] Implement `forfeit_quest` operation
  - [ ] Extract questId from params
  - [ ] Send forfeit command
- [ ] Update context with quest status after operations
- [ ] Test with sample NPC conversation JSON

### 3.3 Saga Orchestrator Actions
**Effort:** M | **Dependencies:** 3.2

Note: Quest validation is handled by atlas-quest. The saga orchestrator only coordinates reward distribution.

- [ ] Create `actions/quest/` package in saga-orchestrator
- [ ] Implement `mark_quest_completed` action
  - [ ] Send command to quest service to mark completed
  - [ ] Wait for status event
  - [ ] Mark step complete/failed
- [ ] Verify existing reward actions work correctly
  - [ ] `award_experience` - Test with quest EXP rewards
  - [ ] `award_mesos` - Test with quest meso rewards
  - [ ] `award_asset` - Test with quest item rewards
  - [ ] `destroy_asset` - Test removing quest items on completion
  - [ ] `create_skill` - Test with quest skill rewards
- [ ] Register new actions in action registry
- [ ] Add compensation logic for rollback
  - [ ] If item award fails, rollback EXP/meso
  - [ ] If mark_completed fails, consider rollback strategy
- [ ] Integration tests for quest reward sagas

### 3.4 Progress Event Consumers
**Effort:** L | **Dependencies:** Phase 2

- [ ] Implement monster death consumer
  - [ ] Subscribe to EVENT_TOPIC_DROP_STATUS or monster death topic
  - [ ] Extract monster ID and killer character ID
  - [ ] Query active quests for character
  - [ ] Check if any quest tracks this monster
  - [ ] Update progress for matching quests
  - [ ] Emit progress update events
- [ ] Implement item change consumer (for item collection quests)
  - [ ] Subscribe to EVENT_TOPIC_ASSET_STATUS
  - [ ] Handle CREATED, QUANTITY_CHANGED events
  - [ ] Extract characterId, templateId from event
  - [ ] Check if any active quest tracks this item
  - [ ] Update item collection progress
  - [ ] Emit progress update event
- [ ] Implement map enter consumer (for medal quests)
  - [ ] Subscribe to EVENT_TOPIC_CHARACTER_STATUS (map changed)
  - [ ] Check if character has active medal quests
  - [ ] Check if map is tracked by quest
  - [ ] Update medal progress
  - [ ] Emit progress update event
- [ ] Auto-complete check
  - [ ] After progress update, check if quest completable
  - [ ] If autoComplete flag set, trigger completion
- [ ] Consumer offset management
- [ ] Idempotency handling

---

## Phase 4: Advanced Features

### 4.1 Repeatable Quests
**Effort:** M | **Dependencies:** Phase 3

- [ ] Add interval check to StartQuest
  - [ ] Get interval requirement from quest definition
  - [ ] Check completion timestamp
  - [ ] Calculate if interval has elapsed
  - [ ] Reject if too soon
- [ ] Reset quest on repeat start
  - [ ] Clear previous progress
  - [ ] Keep forfeit/completed counts
- [ ] Unit tests for interval logic

### 4.2 Time-Limited Quests
**Effort:** M | **Dependencies:** Phase 3

- [ ] Set expiration on timed quest start
  - [ ] Read timeLimit from quest definition
  - [ ] Calculate expiration timestamp
  - [ ] Store in quest status
- [ ] Expiration handling
  - [ ] Background goroutine or scheduled check
  - [ ] Mark quest as failed/expired
  - [ ] Emit expiration event
- [ ] Return expiration time in REST response
- [ ] Unit tests for expiration logic

### 4.3 Quest Chains
**Effort:** S | **Dependencies:** Phase 3

- [ ] Detect nextQuest in completion actions
  - [ ] After completing quest, check actions
  - [ ] If nextQuest present, auto-start
- [ ] Chain quest bypass
  - [ ] Skip normal start requirement validation
  - [ ] Immediately create Started status
- [ ] Chain events
  - [ ] Emit both complete and start events
- [ ] Test with multi-quest chain

### 4.4 Medal Quests
**Effort:** M | **Dependencies:** Phase 3

- [ ] Implement medal map entity
  - [ ] quest_status_id, map_id
  - [ ] Foreign key relationship
- [ ] Medal progress tracking
  - [ ] On map enter, check medal quests
  - [ ] Add map to medal_maps if not visited
- [ ] Medal completion check
  - [ ] Compare visited maps to required maps
  - [ ] Complete when all visited
- [ ] Return medal item ID for UI
- [ ] Unit tests for medal logic

### 4.5 Auto Quests
**Effort:** S | **Dependencies:** Phase 3

- [ ] Auto-start detection
  - [ ] On level up, check for autoStart quests
  - [ ] On map enter, check for autoStart quests
  - [ ] Start quest if requirements met
- [ ] Auto-complete detection
  - [ ] After progress update, check autoComplete flag
  - [ ] Complete quest if requirements met
- [ ] Skip NPC validation for auto quests
- [ ] Event triggers for auto-start conditions

---

## Phase 5: Admin UI (atlas-ui)

### 5.1 Quest Service Layer
**Effort:** S | **Dependencies:** Phase 1 (atlas-data API)

- [ ] Create `types/models/quest.ts`
  - [ ] QuestInfo interface
  - [ ] QuestRequirement interface (union type for all requirement types)
  - [ ] QuestAction interface (union type for all action types)
  - [ ] QuestDefinition interface (combines info, requirements, actions)
  - [ ] CharacterQuestStatus interface
- [ ] Create `services/api/quests.service.ts`
  - [ ] getAll(tenant) - List all quests (info only)
  - [ ] getById(tenant, questId) - Get full quest definition (info + requirements + actions)
- [ ] Create `services/api/quest-status.service.ts`
  - [ ] getByCharacter(tenant, characterId) - All quests for character
  - [ ] getByCharacter(tenant, characterId, status) - Filter by status
  - [ ] getByCharacterAndQuest(tenant, characterId, questId) - Single quest
- [ ] Export services from `services/api/index.ts`

### 5.2 Quest List View
**Effort:** M | **Dependencies:** 5.1

- [ ] Create `app/quests/page.tsx`
  - [ ] Fetch quests using questsService
  - [ ] Loading state with TableSkeleton
  - [ ] Error state with ErrorDisplay
  - [ ] Empty state for no quests
- [ ] Create `app/quests/columns.tsx`
  - [ ] ID column (sortable)
  - [ ] Name column (sortable, searchable)
  - [ ] Category column (filterable)
  - [ ] Level range column
  - [ ] Flags column (badges for auto-start, auto-complete, timed)
  - [ ] Row click handler to navigate to detail
- [ ] Add search input for quest name filtering
- [ ] Add category dropdown filter
- [ ] Test with tenant switch

### 5.3 Quest Detail View
**Effort:** L | **Dependencies:** 5.1, 5.6

- [ ] Create `app/quests/[id]/page.tsx`
  - [ ] Fetch quest definition by ID
  - [ ] Loading state with CardSkeleton
  - [ ] Error state (404, 500)
- [ ] Hero section
  - [ ] Quest ID and name
  - [ ] Category badge
- [ ] Metadata card
  - [ ] Auto-start flag
  - [ ] Auto-complete flag
  - [ ] Time limit (if set)
  - [ ] Medal item (if medal quest)
  - [ ] Area and order values
- [ ] Start Requirements section (Collapsible)
  - [ ] Use RequirementRenderer for each requirement
  - [ ] Empty state if no requirements
- [ ] Completion Requirements section (Collapsible)
  - [ ] Use RequirementRenderer for each requirement
  - [ ] Empty state if no requirements
- [ ] Start Actions section (Collapsible)
  - [ ] Use RewardRenderer for each action
  - [ ] Empty state if no actions
- [ ] Completion Rewards section (Collapsible)
  - [ ] Use RewardRenderer for each action
  - [ ] Empty state if no rewards
- [ ] Breadcrumb navigation (Quests > Quest #ID)

### 5.4 Character Quest Status View
**Effort:** L | **Dependencies:** Phase 2 (atlas-quest API), 5.1

- [ ] Create `components/features/quests/QuestStatusTabs.tsx`
  - [ ] Tabs component: Started | Completed | Available
  - [ ] Count badges on each tab
  - [ ] Tab content areas
- [ ] Update `app/characters/[id]/page.tsx`
  - [ ] Add Quest Status collapsible section
  - [ ] Fetch quest statuses from atlas-quest
  - [ ] Integrate QuestStatusTabs component
- [ ] Started Quests tab
  - [ ] List of in-progress quests
  - [ ] Show quest name (fetch from atlas-data)
  - [ ] Show progress (e.g., "2/5 monsters killed")
  - [ ] Link to quest detail view
  - [ ] Show expiration time if timed quest
- [ ] Completed Quests tab
  - [ ] Paginated list of completed quests
  - [ ] Show quest name
  - [ ] Show completion timestamp
  - [ ] Show completion count (for repeatables)
  - [ ] Link to quest detail view
- [ ] Available Quests tab
  - [ ] Calculate available quests (requirements met, not started)
  - [ ] This requires checking character level, job, items, etc.
  - [ ] May need server-side endpoint for this calculation
  - [ ] Show quest name and level requirement
  - [ ] Link to quest detail view

### 5.5 Navigation Integration
**Effort:** S | **Dependencies:** 5.2

- [ ] Update `components/app-sidebar.tsx`
  - [ ] Add "Quests" item to Operations section
  - [ ] Use appropriate icon (ScrollText or similar)
  - [ ] URL: /quests
- [ ] Verify breadcrumb generation
  - [ ] /quests shows "Quests"
  - [ ] /quests/[id] shows "Quests > Quest #[id]"
- [ ] Test active state highlighting in sidebar

### 5.6 Requirement/Reward Renderers
**Effort:** M | **Dependencies:** 5.1

- [ ] Create `components/features/quests/RequirementRenderer.tsx`
  - [ ] minLevel: "Level X+"
  - [ ] maxLevel: "Level X or below"
  - [ ] job: "Job: [Job Names]" with job name lookup
  - [ ] item: "X × [Item Name]" with item name lookup and link
  - [ ] mob: "Kill X × [Monster Name]" with monster name lookup and link
  - [ ] quest: "Complete Quest #X: [Quest Name]" with link
  - [ ] npc: "Talk to [NPC Name]" with NPC name lookup and link
  - [ ] interval: "Repeatable every X hours"
  - [ ] fieldEnter: "Enter map [Map Name]"
  - [ ] meso: "X Mesos"
  - [ ] fame: "X Fame"
  - [ ] Handle unknown requirement types gracefully
- [ ] Create `components/features/quests/RewardRenderer.tsx`
  - [ ] exp: "X EXP"
  - [ ] meso: "X Mesos"
  - [ ] item: "X × [Item Name]" with item details
    - [ ] Show probability if prop is set
    - [ ] Show job restriction if job-specific
    - [ ] Show gender restriction if gender-specific
  - [ ] skill: "Skill: [Skill Name] Lv.X"
  - [ ] fame: "X Fame"
  - [ ] buff: "Buff: [Buff Name]"
  - [ ] nextQuest: "Starts Quest #X: [Quest Name]" with link
  - [ ] Handle unknown action types gracefully
- [ ] Create entity lookup hooks
  - [ ] useItemName(itemId) - Fetch from atlas-data
  - [ ] useMonsterName(monsterId) - Fetch from atlas-data
  - [ ] useNpcName(npcId) - Fetch from atlas-data
  - [ ] useQuestName(questId) - Fetch from atlas-data
  - [ ] Use React Query for caching
- [ ] Create `lib/constants/jobs.ts` for job ID → name mapping

---

## Testing Tasks

### Unit Tests
- [ ] Quest model and builder tests
- [ ] Entity transformation tests
- [ ] Progress formatting tests
- [ ] WZ/XML reader tests
- [ ] Requirement parsing tests
- [ ] Action parsing tests

### Integration Tests
- [ ] Quest service REST API tests
- [ ] Kafka command/event tests
- [ ] Saga orchestrator quest action tests
- [ ] Query aggregator quest lookup tests

### End-to-End Tests
- [ ] Full quest flow: start → progress → complete
- [ ] Quest rewards distributed correctly
- [ ] Quest chain progression
- [ ] Repeatable quest cooldown
- [ ] Time-limited quest expiration

---

## Documentation Tasks

- [ ] Update README with quest service overview
- [ ] API documentation for REST endpoints
- [ ] Kafka message format documentation
- [ ] NPC conversation quest operation examples
- [ ] Quest definition WZ/XML format reference

---

## Deployment Tasks

- [ ] Add atlas-quest to docker-compose.yml
- [ ] Configure environment variables
- [ ] Database migration execution
- [ ] Kafka topic creation
- [ ] Service health checks
- [ ] Monitoring dashboard setup

---

## Progress Summary

| Phase | Total Tasks | Completed | Percentage |
|-------|-------------|-----------|------------|
| Phase 1 | 26 | 0 | 0% |
| Phase 2 | 35 | 0 | 0% |
| Phase 3 | 20 | 0 | 0% |
| Phase 4 | 16 | 0 | 0% |
| Phase 5 (UI) | 52 | 0 | 0% |
| Testing | 10 | 0 | 0% |
| Documentation | 5 | 0 | 0% |
| Deployment | 6 | 0 | 0% |
| **Total** | **170** | **0** | **0%** |
