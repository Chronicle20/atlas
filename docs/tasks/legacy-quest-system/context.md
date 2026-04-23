# Quest System - Context and Key Decisions

**Last Updated:** 2026-01-05

## Key Concepts

### InfoNumber (Shared Progress Key)

The WZ field `infoNumber` allows multiple quests to share the same progress tracking. Despite the opaque name, it serves a specific purpose:

**What it does**: When set, progress is stored/retrieved using `infoNumber` as the key instead of the quest's own ID.

**Why it exists**: Enables progress continuity across quest "versions" - e.g., daily event quests where Monday's quest (10431) and Tuesday's quest (10432) should share the same "collect 100 tokens" counter.

**Implementation**:
```go
func progressKey(questId, infoNumber uint32) uint32 {
    if infoNumber != 0 {
        return infoNumber
    }
    return questId
}
```

**Naming**: Use `infoNumber` at WZ/REST layer for compatibility, but consider clearer names like `progressKey` or `sharedProgressId` in domain models.

---

## Design Decisions

### 1. Data Source: WZ/XML via atlas-data
**Decision:** Quest definitions are loaded from WZ/XML files, stored in the atlas-data document database, and cached in memory.

**Rationale:**
- Follows existing pattern for equipment, monsters, consumables, etc.
- Supports multi-tenant architecture with tenant-specific data
- Two-tier caching (registry + DB) provides fast lookups
- Data can be refreshed by re-uploading WZ files

**Alternative Considered:** Database-only storage was rejected as it would deviate from established patterns and complicate data management.

### 2. Progress Tracking: Event-Driven via Kafka
**Decision:** Quest progress updates via Kafka event consumption (monster deaths, inventory changes, map enters).

**Rationale:**
- Real-time progress updates for better UX
- Decoupled architecture - quest service doesn't need to know about monster/inventory internals
- Consistent with existing event-driven patterns

**Alternative Considered:** Polling/validation on NPC interaction was rejected due to poor responsiveness and increased NPC conversation complexity.

### 3. Dialog Ownership: NPC Conversations
**Decision:** atlas-npc-conversations owns all quest dialog flow; atlas-quest only tracks state/progress.

**Rationale:**
- Clear separation of concerns
- Quest dialogs can use existing JSON state machine infrastructure
- Complex dialog logic stays in one place
- Quest service remains focused on data management

**Alternative Considered:** Quest-owned dialogs would duplicate NPC conversation infrastructure.

### 4. Party Quests: Deferred
**Decision:** Solo quests only in initial implementation. Party quest support added in future phase.

**Rationale:**
- Reduces initial complexity
- Core quest mechanics can be validated before adding party sharing
- Party quest patterns can be designed with learnings from solo quests

---

## Key Files Reference

### atlas-data (Quest Definitions)

| File | Purpose |
|------|---------|
| `services/atlas-data/atlas.com/data/data/processor.go` | Worker registration - add QUEST worker |
| `services/atlas-data/atlas.com/data/data/resource.go` | Data upload endpoint |
| `services/atlas-data/atlas.com/data/document/entity.go` | Document storage entity |
| `services/atlas-data/atlas.com/data/document/storage.go` | Two-tier storage lookup |
| `services/atlas-data/atlas.com/data/document/registry.go` | In-memory cache pattern |
| `services/atlas-data/atlas.com/data/equipment/reader.go` | Example XML reader pattern |
| `services/atlas-data/atlas.com/data/equipment/resource.go` | Example REST resource |
| `services/atlas-data/atlas.com/data/main.go` | Route initializer registration |

### atlas-npc-conversations (Dialog Integration)

| File | Purpose |
|------|---------|
| `services/atlas-npc-conversations/atlas.com/npc-conversations/conversation/saga_executor.go` | Saga operation execution |
| `services/atlas-npc-conversations/atlas.com/npc-conversations/conversation/processor.go` | Conversation state machine |
| `services/atlas-npc-conversations/atlas.com/npc-conversations/state/generic_action.go` | Generic action handling |

### atlas-saga-orchestrator (Reward Distribution)

| File | Purpose |
|------|---------|
| `services/atlas-saga-orchestrator/atlas.com/saga/saga/processor.go` | Saga step execution |
| `services/atlas-saga-orchestrator/atlas.com/saga/saga/model.go` | Saga and step models |
| `services/atlas-saga-orchestrator/atlas.com/saga/saga/actions/` | Action implementations |

### atlas-query-aggregator (Validation)

| File | Purpose |
|------|---------|
| `services/atlas-query-aggregator/atlas.com/query-aggregator/quest/model.go` | Quest model stub |
| `services/atlas-query-aggregator/atlas.com/query-aggregator/quest/processor.go` | Quest processor stub |
| `services/atlas-query-aggregator/atlas.com/query-aggregator/conditions/` | Condition evaluators |

### Kafka Topics (libs/atlas-kafka)

| File | Purpose |
|------|---------|
| `libs/atlas-kafka/kafka/topics.go` | Topic constants |
| `libs/atlas-kafka/kafka/producer/` | Message production patterns |
| `libs/atlas-kafka/kafka/consumer/` | Consumer patterns |

### Domain Model Patterns

| File | Purpose |
|------|---------|
| `services/atlas-character/atlas.com/character/character/model.go` | Immutable model example |
| `services/atlas-character/atlas.com/character/character/builder.go` | Builder pattern example |
| `services/atlas-character/atlas.com/character/character/entity.go` | GORM entity example |
| `services/atlas-character/atlas.com/character/character/processor.go` | Processor pattern example |
| `services/atlas-character/atlas.com/character/character/producer.go` | Event producer example |

---

## WZ/XML File Reference

### Quest.wz Structure

```
Quest.wz/
├── QuestInfo.img.xml    # Quest metadata (name, autoStart, timeLimit, etc.)
├── Check.img.xml        # Start/completion requirements
├── Act.img.xml          # Start/completion actions (rewards)
├── Say.img.xml          # Default dialog text (optional)
├── PQuest.img.xml       # Party quest config (future phase)
└── Exclusive.img.xml    # Mutually exclusive quests
```

### QuestInfo.img.xml Example
```xml
<imgdir name="2000">
  <string name="name" value="Mai's First Training"/>
  <string name="parent" value="Maple Island"/>
  <int name="autoStart" value="0"/>
  <int name="autoPreComplete" value="0"/>
  <int name="autoComplete" value="0"/>
  <int name="timeLimit" value="0"/>
  <int name="area" value="0"/>
  <int name="order" value="1"/>
</imgdir>
```

### Check.img.xml Example
```xml
<imgdir name="2000">
  <imgdir name="0">  <!-- Start requirements -->
    <int name="lvmin" value="1"/>
    <int name="npc" value="1002000"/>
  </imgdir>
  <imgdir name="1">  <!-- Completion requirements -->
    <imgdir name="mob">
      <imgdir name="0">
        <int name="id" value="100100"/>
        <int name="count" value="3"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>
```

### Act.img.xml Example
```xml
<imgdir name="2000">
  <imgdir name="0">  <!-- Start actions -->
    <!-- Usually empty or gives starter items -->
  </imgdir>
  <imgdir name="1">  <!-- Completion actions -->
    <int name="exp" value="10"/>
    <imgdir name="item">
      <imgdir name="0">
        <int name="id" value="2000000"/>
        <int name="count" value="5"/>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>
```

---

## Database Schema

### queststatus Table
```sql
CREATE TABLE quest_statuses (
    tenant_id UUID NOT NULL,
    id UUID PRIMARY KEY,
    character_id INT NOT NULL,
    quest_id INT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 0,
    npc_id INT,
    completion_time BIGINT,
    expiration_time BIGINT,
    forfeit_count INT NOT NULL DEFAULT 0,
    completed_count INT NOT NULL DEFAULT 0,
    custom_data TEXT,
    info_number INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_quest_statuses_character ON quest_statuses(tenant_id, character_id);
CREATE UNIQUE INDEX idx_quest_statuses_unique ON quest_statuses(tenant_id, character_id, quest_id);
```

### questprogress Table
```sql
CREATE TABLE quest_progress (
    tenant_id UUID NOT NULL,
    id UUID PRIMARY KEY,
    quest_status_id UUID NOT NULL REFERENCES quest_statuses(id) ON DELETE CASCADE,
    progress_id INT NOT NULL,  -- mob_id or item_id
    progress VARCHAR(15) NOT NULL,  -- "000" to "999"
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_quest_progress_status ON quest_progress(quest_status_id);
```

### medalmaps Table
```sql
CREATE TABLE medal_maps (
    tenant_id UUID NOT NULL,
    id UUID PRIMARY KEY,
    quest_status_id UUID NOT NULL REFERENCES quest_statuses(id) ON DELETE CASCADE,
    map_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_medal_maps_status ON medal_maps(quest_status_id);
```

---

## Kafka Message Formats

### Quest Command (COMMAND_TOPIC_QUEST)
```json
{
  "transactionId": "uuid",
  "characterId": 12345,
  "questId": 2000,
  "npcId": 1002000,
  "command": "START_QUEST",
  "payload": {}
}
```

### Quest Status Event (EVENT_TOPIC_QUEST_STATUS)
```json
{
  "transactionId": "uuid",
  "characterId": 12345,
  "questId": 2000,
  "worldId": 0,
  "type": "STARTED",
  "body": {
    "npcId": 1002000,
    "progress": {
      "100100": "000"
    }
  }
}
```

### Quest Progress Event
```json
{
  "transactionId": "uuid",
  "characterId": 12345,
  "questId": 2000,
  "worldId": 0,
  "type": "PROGRESS_UPDATED",
  "body": {
    "mobId": 100100,
    "progress": "001",
    "required": "003"
  }
}
```

---

## Integration Points

### NPC Conversation Operations

Add to operation executor:
```go
case "start_quest":
    questId := params["questId"]
    npcId := params["npcId"]
    // Send Kafka command or call saga

case "complete_quest":
    questId := params["questId"]
    // Trigger saga for validation + rewards + completion

case "forfeit_quest":
    questId := params["questId"]
    // Send Kafka command
```

### Query Aggregator Conditions

Update quest processor to call atlas-quest REST API:
```go
func (p *processorImpl) GetQuestStatus(characterId, questId uint32) model.Provider[QuestStatus] {
    return func() (QuestStatus, error) {
        // GET /api/characters/{characterId}/quests/{questId}
        // Parse response and return status
    }
}
```

### Saga Orchestrator Actions

New actions to register:
```go
"start_quest":    quest.StartQuestAction,
"complete_quest": quest.CompleteQuestAction,
"forfeit_quest":  quest.ForfeitQuestAction,
```

---

## Environment Variables

### atlas-quest Service
```
DB_USER=atlas
DB_PASSWORD=secret
DB_HOST=localhost
DB_PORT=5432
DB_NAME=atlas_quest

KAFKA_BOOTSTRAP_SERVERS=localhost:9092
KAFKA_CONSUMER_GROUP=atlas-quest

DATA_SERVICE_URL=http://atlas-data:8080
```

### Service URLs
```
QUEST_SERVICE_URL=http://atlas-quest:8080  # For query-aggregator
```

---

---

## Admin UI (atlas-ui) Integration

### Technology Stack
- **Framework**: Next.js 16 (App Router)
- **UI Library**: shadcn/ui (Radix UI + Tailwind CSS)
- **State**: TanStack React Query v5
- **Tables**: TanStack Table v8
- **Forms**: React Hook Form + Zod

### Key UI Files Reference

| File | Purpose |
|------|---------|
| `services/atlas-ui/app/` | Next.js app router pages |
| `services/atlas-ui/components/ui/` | shadcn/ui base components |
| `services/atlas-ui/components/common/DataTableWrapper.tsx` | Reusable table with states |
| `services/atlas-ui/components/app-sidebar.tsx` | Navigation sidebar |
| `services/atlas-ui/services/api/` | API service layer |
| `services/atlas-ui/types/models/` | TypeScript type definitions |
| `services/atlas-ui/context/tenant-context.tsx` | Multi-tenant state |
| `services/atlas-ui/lib/api/client.ts` | API client with caching |

### UI File Structure for Quests

```
services/atlas-ui/
├── app/
│   └── quests/
│       ├── page.tsx              # Quest list view
│       ├── columns.tsx           # Table column definitions
│       └── [id]/
│           └── page.tsx          # Quest detail view
├── components/
│   └── features/
│       └── quests/
│           ├── QuestCard.tsx             # Quest summary card
│           ├── RequirementRenderer.tsx   # Render requirements
│           ├── RewardRenderer.tsx        # Render rewards
│           └── QuestStatusTabs.tsx       # Character quest tabs
├── services/
│   └── api/
│       ├── quests.service.ts     # Quest definition API
│       └── quest-status.service.ts # Character quest status API
└── types/
    └── models/
        └── quest.ts              # Quest TypeScript types
```

### Existing Patterns to Follow

**Service Pattern:**
```typescript
// services/api/quests.service.ts
class QuestsService {
  async getAll(tenant: Tenant): Promise<QuestInfo[]> {
    api.setTenant(tenant);
    return api.getList<QuestInfo>('/api/data/quests');
  }

  async getById(tenant: Tenant, questId: string): Promise<QuestDefinition> {
    api.setTenant(tenant);
    return api.getOne<QuestDefinition>(`/api/data/quests/${questId}`);
  }
}

export const questsService = new QuestsService();
```

**Page Pattern:**
```typescript
// app/quests/page.tsx
'use client';

export default function QuestsPage() {
  const { activeTenant } = useTenant();
  const [quests, setQuests] = useState<QuestInfo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    questsService.getAll(activeTenant)
      .then(setQuests)
      .finally(() => setLoading(false));
  }, [activeTenant]);

  return (
    <DataTableWrapper
      columns={columns}
      data={quests}
      loading={loading}
      emptyState={{
        title: "No quests found",
        description: "Upload quest data to view quests."
      }}
    />
  );
}
```

**Detail Page Pattern:**
```typescript
// app/quests/[id]/page.tsx
'use client';

export default function QuestDetailPage({ params }: { params: { id: string } }) {
  const { activeTenant } = useTenant();
  const [quest, setQuest] = useState<QuestDefinition | null>(null);

  useEffect(() => {
    questsService.getById(activeTenant, params.id)
      .then(setQuest);
  }, [activeTenant, params.id]);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Quest #{quest?.info.id}: {quest?.info.name}</CardTitle>
        </CardHeader>
        <CardContent>
          {/* Quest details */}
        </CardContent>
      </Card>

      <Collapsible>
        <CollapsibleTrigger>Start Requirements</CollapsibleTrigger>
        <CollapsibleContent>
          {quest?.startRequirements.map(req => (
            <RequirementRenderer key={req.type} requirement={req} />
          ))}
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
```

### API Endpoints for UI

**Quest Definitions (atlas-data):**
- `GET /api/data/quests` - List all quests (returns QuestInfo[])
- `GET /api/data/quests/{questId}` - Full quest definition (info, requirements, actions)

**Character Quest Status (atlas-quest):**
- `GET /api/characters/{characterId}/quests` - All quest statuses
- `GET /api/characters/{characterId}/quests?status=started` - Filter by status
- `GET /api/characters/{characterId}/quests/{questId}` - Single quest status

---

## Testing Strategy

### Unit Tests
- Quest model builder and accessors
- Entity ↔ Model transformations
- Progress string formatting ("000" padding)
- Requirement/action parsing

### Integration Tests
- Quest start → progress update → complete flow
- Reward distribution via saga
- Event emission verification
- Multi-tenant isolation

### E2E Tests
- NPC conversation → quest start
- Monster kill → progress update → UI notification
- Quest complete → rewards received
- Quest chain progression

---

## Monitoring Considerations

### Metrics to Track
- Quest start/complete/forfeit counts
- Average quest completion time
- Progress update latency
- Failed saga compensation count

### Logging
- Quest state transitions (INFO)
- Progress updates (DEBUG)
- Validation failures (WARN)
- Saga failures (ERROR)

### Alerts
- High quest failure rate
- Saga compensation triggered
- Event consumer lag
