# Quest-Aware Drop Systems Plan

**Last Updated:** 2026-02-03

## Executive Summary

This plan implements quest-aware filtering for monster and reactor drop systems. Currently, both systems have a `questId` field on drop definitions that is stored but not enforced. Items associated with a quest will only drop if the character has that quest in the "started" state (not "not-started" or "completed").

## Current State Analysis

### Monster Drops (atlas-monster-death)

**Location:** `services/atlas-monster-death/atlas.com/monster/monster/processor.go`

**Current Flow:**
1. `CreateDrops()` is called when a monster dies
2. Drops are fetched from `atlas-drop-information` via `drop.GetByMonsterId()`
3. Each drop has a `questId` field (but it's not used for filtering)
4. Drops are filtered only by chance calculation: `rand.Int31n(999999) < chance * itemDropRate`
5. Successful drops are spawned via `drop.Create()`

**Drop Model** (`monster/drop/model.go`):
```go
type Model struct {
    itemId          uint32
    minimumQuantity uint32
    maximumQuantity uint32
    questId         uint32  // EXISTS but not used for filtering
    chance          uint32
}
```

### Reactor Drops (atlas-saga-orchestrator)

**Location:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/processor.go`

**Current Flow:**
1. `SpawnReactorDrops()` is called after reactor activation
2. Drops fetched from `atlas-drop-information` via classification (reactor ID)
3. Each drop has a `questId` field (not used for filtering)
4. Drops filtered by chance: `rand.Float64() < itemDropRate / chance`
5. Items spawned via Kafka commands

**Drop Model** (`reactor/drop/model.go`):
```go
type Model struct {
    reactorId uint32
    itemId    uint32
    questId   uint32  // EXISTS but not used for filtering
    chance    uint32
}
```

### Quest System (atlas-quest)

**Location:** `services/atlas-quest/atlas.com/quest/`

**Quest States:**
```go
const (
    StateNotStarted State = 0
    StateStarted    State = 1
    StateCompleted  State = 2
)
```

**Available REST Endpoints:**
- `GET /characters/{characterId}/quests` - All quests for character
- `GET /characters/{characterId}/quests/started` - Started quests only
- `GET /characters/{characterId}/quests/{questId}` - Specific quest by ID

## Proposed Future State

### High-Level Architecture

```
Monster/Reactor Dies/Activates
         │
         ▼
   Fetch Drop Pool
         │
         ▼
 ┌───────────────────┐
 │ NEW: Filter drops │
 │ by quest state    │
 │ (questId != 0)    │
 └───────────────────┘
         │
         ▼
   Apply Chance Roll
         │
         ▼
   Spawn Successful Drops
```

### Filtering Logic

For each drop with `questId != 0`:
1. Query atlas-quest service for character's quest state
2. Only include drop in pool if quest state == `StateStarted` (1)
3. Drops with `questId == 0` are always included (non-quest items)

### Performance Considerations

**Option A: Individual Quest Lookups**
- Pros: Simple, accurate
- Cons: N+1 query problem, high latency for many quest-linked drops

**Option B: Batch Quest Lookup (Recommended)**
- Fetch all started quests for character once
- Build a set of started quest IDs
- Filter drops locally against this set
- Pros: Single HTTP call, efficient
- Cons: Slightly more complex

**Selected Approach:** Option B - Use the existing `/characters/{characterId}/quests/started` endpoint to get all started quests in one call, then filter locally.

## Implementation Phases

### Phase 1: Quest Integration Module (atlas-monster-death)

Create a reusable module for querying quest states that can be used by the monster death processor.

**Tasks:**
1. Create `quest/` package in atlas-monster-death
2. Define quest REST model and request functions
3. Implement `GetStartedQuestIds()` function

### Phase 2: Monster Drop Quest Filtering

Modify the monster death processor to filter quest-specific drops.

**Tasks:**
1. Add quest module import to processor
2. Modify `CreateDrops()` to fetch started quest IDs
3. Create `filterByQuestState()` function
4. Apply filter before chance evaluation

### Phase 3: Quest Integration Module (atlas-saga-orchestrator)

Create similar quest integration for the saga orchestrator service.

**Tasks:**
1. Create `quest/` package for quest state queries
2. Implement REST model and requests
3. Implement `GetStartedQuestIds()` function

### Phase 4: Reactor Drop Quest Filtering

Modify the reactor drop processor to filter quest-specific drops.

**Tasks:**
1. Add quest module import to reactor drop processor
2. Modify `SpawnReactorDrops()` to accept character-specific filtering
3. Create quest filtering logic
4. Apply filter before chance evaluation

### Phase 5: Testing and Validation

Create integration tests and validate the implementation.

**Tasks:**
1. Unit tests for quest filtering logic
2. Integration tests for end-to-end flow
3. Performance testing with multiple quest-linked drops
4. Manual QA validation

## Detailed Implementation

### Phase 1: Quest Integration Module (atlas-monster-death)

**File: `services/atlas-monster-death/atlas.com/monster/quest/model.go`**
```go
package quest

type State byte

const (
    StateNotStarted State = 0
    StateStarted    State = 1
    StateCompleted  State = 2
)

type Model struct {
    characterId uint32
    questId     uint32
    state       State
}

func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) QuestId() uint32     { return m.questId }
func (m Model) State() State        { return m.state }
```

**File: `services/atlas-monster-death/atlas.com/monster/quest/rest.go`**
```go
package quest

import "strconv"

type RestModel struct {
    Id          uint32 `json:"-"`
    CharacterId uint32 `json:"characterId"`
    QuestId     uint32 `json:"questId"`
    State       State  `json:"state"`
}

func (r RestModel) GetName() string { return "quest-status" }
func (r RestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }
func (r *RestModel) SetID(idStr string) error {
    id, err := strconv.Atoi(idStr)
    if err != nil { return err }
    r.Id = uint32(id)
    return nil
}

func Extract(rm RestModel) (Model, error) {
    return Model{
        characterId: rm.CharacterId,
        questId:     rm.QuestId,
        state:       rm.State,
    }, nil
}
```

**File: `services/atlas-monster-death/atlas.com/monster/quest/requests.go`**
```go
package quest

import (
    "atlas-monster-death/rest"
    "fmt"
    "github.com/Chronicle20/atlas-rest/requests"
)

const (
    StartedQuestsResource = "characters/%d/quests/started"
)

func getBaseRequest() string {
    return requests.RootUrl("QUESTS")
}

func requestStartedQuests(characterId uint32) requests.Request[[]RestModel] {
    return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+StartedQuestsResource, characterId))
}
```

**File: `services/atlas-monster-death/atlas.com/monster/quest/provider.go`**
```go
package quest

import (
    "context"
    "github.com/Chronicle20/atlas-model/model"
    "github.com/Chronicle20/atlas-rest/requests"
    "github.com/sirupsen/logrus"
)

func startedQuestsProvider(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
    return func(ctx context.Context) func(characterId uint32) model.Provider[[]Model] {
        return func(characterId uint32) model.Provider[[]Model] {
            return requests.SliceProvider[RestModel, Model](l, ctx)(requestStartedQuests(characterId), Extract, model.Filters[Model]())
        }
    }
}

func GetStartedQuestIds(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (map[uint32]bool, error) {
    return func(ctx context.Context) func(characterId uint32) (map[uint32]bool, error) {
        return func(characterId uint32) (map[uint32]bool, error) {
            quests, err := startedQuestsProvider(l)(ctx)(characterId)()
            if err != nil {
                return nil, err
            }

            result := make(map[uint32]bool)
            for _, q := range quests {
                result[q.QuestId()] = true
            }
            return result, nil
        }
    }
}
```

### Phase 2: Monster Drop Quest Filtering

**Modified: `services/atlas-monster-death/atlas.com/monster/monster/processor.go`**

Add quest filtering before chance evaluation:

```go
func CreateDrops(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error {
    return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error {
        return func(worldId byte, channelId byte, mapId uint32, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error {
            dropType := byte(0)

            ds, err := drop.GetByMonsterId(l)(ctx)(monsterId)
            if err != nil {
                return err
            }
            l.Debugf("Monster [%d] has [%d] drops to evaluate.", monsterId, len(ds))

            // NEW: Filter quest-specific drops
            ds = filterByQuestState(l)(ctx)(killerId, ds)
            l.Debugf("After quest filtering, [%d] drops remain.", len(ds))

            r := rates.GetForCharacter(l)(ctx)(worldId, channelId, killerId)
            l.Debugf("Character [%d] rates: itemDrop=%.2f, meso=%.2f", killerId, r.ItemDropRate(), r.MesoRate())

            ds = getSuccessfulDrops(ds, r.ItemDropRate())

            for i, d := range ds {
                _ = drop.Create(l)(ctx)(worldId, channelId, mapId, i+1, id, x, y, killerId, dropType, d, r.MesoRate())
            }
            return nil
        }
    }
}

func filterByQuestState(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, drops []drop.Model) []drop.Model {
    return func(ctx context.Context) func(characterId uint32, drops []drop.Model) []drop.Model {
        return func(characterId uint32, drops []drop.Model) []drop.Model {
            // Check if any drops require quest filtering
            hasQuestDrops := false
            for _, d := range drops {
                if d.QuestId() != 0 {
                    hasQuestDrops = true
                    break
                }
            }

            // Skip quest lookup if no quest-specific drops
            if !hasQuestDrops {
                return drops
            }

            // Fetch started quest IDs for character
            startedQuests, err := quest.GetStartedQuestIds(l)(ctx)(characterId)
            if err != nil {
                l.WithError(err).Warnf("Unable to fetch started quests for character [%d], excluding all quest drops.", characterId)
                // On error, exclude all quest-specific drops for safety
                startedQuests = make(map[uint32]bool)
            }

            result := make([]drop.Model, 0, len(drops))
            for _, d := range drops {
                if d.QuestId() == 0 {
                    // Non-quest item, always include
                    result = append(result, d)
                } else if startedQuests[d.QuestId()] {
                    // Quest item with started quest
                    result = append(result, d)
                }
                // Quest item without started quest is excluded
            }
            return result
        }
    }
}
```

### Phase 3 & 4: Reactor Drop Quest Filtering

Similar pattern for reactor drops in atlas-saga-orchestrator:

**New Files:**
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/state/model.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/state/rest.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/state/requests.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/state/provider.go`

**Modified: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/processor.go`**

Add quest filtering to `rollDrops()`:

```go
func (p *ProcessorImpl) SpawnReactorDrops(...) error {
    // ... existing code ...

    // Fetch rates for the character
    r := rates.GetForCharacter(p.l)(p.ctx)(byte(worldId), byte(channelId), characterId)

    // Fetch reactor drops from atlas-drop-information using classification
    drops, err := p.fetchReactorDrops(classification)
    // ... error handling ...

    // NEW: Filter quest-specific drops
    drops = p.filterByQuestState(characterId, drops)

    // Roll chances to determine which items to drop
    itemsToDrop := p.rollDrops(drops, r.ItemDropRate())

    // ... rest of existing code ...
}

func (p *ProcessorImpl) filterByQuestState(characterId uint32, drops []Model) []Model {
    // Check if any drops require quest filtering
    hasQuestDrops := false
    for _, d := range drops {
        if d.QuestId() != 0 {
            hasQuestDrops = true
            break
        }
    }

    if !hasQuestDrops {
        return drops
    }

    // Fetch started quest IDs
    startedQuests, err := queststate.GetStartedQuestIds(p.l)(p.ctx)(characterId)
    if err != nil {
        p.l.WithError(err).Warnf("Unable to fetch started quests for character [%d], excluding all quest drops.", characterId)
        startedQuests = make(map[uint32]bool)
    }

    result := make([]Model, 0, len(drops))
    for _, d := range drops {
        if d.QuestId() == 0 || startedQuests[d.QuestId()] {
            result = append(result, d)
        }
    }
    return result
}
```

## Risk Assessment and Mitigation

| Risk | Severity | Probability | Mitigation |
|------|----------|-------------|------------|
| Quest service unavailable | Medium | Low | Exclude all quest drops on error (fail-safe) |
| Performance degradation | Medium | Medium | Check for quest drops before making HTTP call; skip if none |
| Race condition on quest state | Low | Low | Quest state changes are rare; momentary inconsistency acceptable |
| Incorrect drop filtering | High | Low | Thorough testing; debug logging |
| Environment variable missing | Medium | Low | Add QUESTS root URL to deployment configs |

## Success Metrics

1. **Functional:** Quest items only drop when quest is in "started" state
2. **Performance:** < 50ms additional latency when quest filtering is needed
3. **Reliability:** No dropped items due to quest service errors (fail-safe to exclude)
4. **Observability:** Debug logs show quest filtering decisions

## Required Resources and Dependencies

### Code Changes
- **atlas-monster-death:** New `quest/` package, modified `processor.go`
- **atlas-saga-orchestrator:** New `quest/state/` package, modified `reactor/drop/processor.go`

### Configuration
- Both services need `QUESTS` environment variable for atlas-quest service URL

### External Dependencies
- atlas-quest service must be available
- atlas-drop-information service (existing dependency)

## Timeline

| Phase | Description | Effort |
|-------|-------------|--------|
| 1 | Quest Integration Module (atlas-monster-death) | S |
| 2 | Monster Drop Quest Filtering | M |
| 3 | Quest Integration Module (atlas-saga-orchestrator) | S |
| 4 | Reactor Drop Quest Filtering | M |
| 5 | Testing and Validation | M |

## Implementation Order

1. Phase 1 + Phase 2 (monster drops)
2. Phase 3 + Phase 4 (reactor drops)
3. Phase 5 (testing)

Each phase pair can be developed and deployed independently.
