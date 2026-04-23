# Quest-Aware Drop Systems - Context

**Last Updated:** 2026-02-03

## Overview

This document contains key context, files, decisions, and dependencies for implementing quest-aware filtering in the monster and reactor drop systems.

## Key Files

### Monster Death Service (atlas-monster-death)

| File | Purpose |
|------|---------|
| `services/atlas-monster-death/atlas.com/monster/monster/processor.go` | Main drop creation logic - needs quest filter |
| `services/atlas-monster-death/atlas.com/monster/monster/drop/model.go` | Drop model with `QuestId()` getter |
| `services/atlas-monster-death/atlas.com/monster/monster/drop/provider.go` | Fetches drops from atlas-drop-information |
| `services/atlas-monster-death/atlas.com/monster/monster/drop/requests.go` | REST request to drop-information |
| `services/atlas-monster-death/atlas.com/monster/monster/drop/rest.go` | Drop REST model with QuestId field |
| `services/atlas-monster-death/atlas.com/monster/rest/request.go` | Base REST request helpers |
| `services/atlas-monster-death/atlas.com/monster/character/requests.go` | Example of REST client pattern |

### Saga Orchestrator Service (atlas-saga-orchestrator)

| File | Purpose |
|------|---------|
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/processor.go` | Reactor drop spawning - needs quest filter |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/model.go` | Reactor drop model with `QuestId()` getter |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/processor.go` | Existing quest processor (Kafka-based, not for queries) |

### Quest Service (atlas-quest)

| File | Purpose |
|------|---------|
| `services/atlas-quest/atlas.com/quest/quest/state.go` | Quest state constants (0=NotStarted, 1=Started, 2=Completed) |
| `services/atlas-quest/atlas.com/quest/quest/model.go` | Quest model definition |
| `services/atlas-quest/atlas.com/quest/quest/rest.go` | Quest REST model - use as template |
| `services/atlas-quest/atlas.com/quest/quest/resource.go` | REST endpoints including `/started` |

### Drop Information Service (atlas-drop-information)

| File | Purpose |
|------|---------|
| `services/atlas-drop-information/atlas.com/dis/monster/drop/model.go` | Source of truth for monster drop definitions |
| `services/atlas-drop-information/atlas.com/dis/reactor/drop/model.go` | Source of truth for reactor drop definitions |

## Key Code Patterns

### REST Client Pattern (from atlas-monster-death)

```go
// requests.go
const (
    Resource = "characters/%d/quests/started"
)

func getBaseRequest() string {
    return requests.RootUrl("QUESTS")  // Uses QUESTS env var
}

func requestStartedQuests(characterId uint32) requests.Request[[]RestModel] {
    return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

// provider.go
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

### Drop Model Pattern

Both monster and reactor drop models have a `QuestId()` getter:

```go
// Monster drop (atlas-monster-death)
type Model struct {
    itemId          uint32
    minimumQuantity uint32
    maximumQuantity uint32
    questId         uint32
    chance          uint32
}
func (m Model) QuestId() uint32 { return m.questId }

// Reactor drop (atlas-saga-orchestrator)
type Model struct {
    reactorId uint32
    itemId    uint32
    questId   uint32
    chance    uint32
}
func (m Model) QuestId() uint32 { return m.questId }
```

### Quest State Constants

```go
// From atlas-quest/quest/state.go
type State byte

const (
    StateNotStarted State = 0
    StateStarted    State = 1
    StateCompleted  State = 2
)
```

## Key Decisions

### Decision 1: Batch Query vs Individual Queries

**Decision:** Use batch query to fetch all started quests once per drop event.

**Rationale:**
- Avoids N+1 query problem
- Single HTTP roundtrip regardless of number of quest drops
- Uses existing `/characters/{characterId}/quests/started` endpoint

### Decision 2: Error Handling Strategy

**Decision:** On quest service error, exclude all quest-specific drops (fail-safe).

**Rationale:**
- Prevents accidental quest item drops when service is unavailable
- Better player experience to miss a quest drop than get one they shouldn't
- Non-quest items still drop normally

### Decision 3: Quest ID Zero Handling

**Decision:** Drops with `questId == 0` are always included in the pool.

**Rationale:**
- Zero indicates non-quest item
- Maintains backward compatibility
- Matches existing data conventions

### Decision 4: Early Exit Optimization

**Decision:** Check if any drops have `questId != 0` before making HTTP call.

**Rationale:**
- Most monsters don't have quest drops
- Avoids unnecessary network calls
- Significant performance improvement for common case

## Dependencies

### Service Dependencies

```
atlas-monster-death ──────► atlas-drop-information (existing)
                    ──────► atlas-quest (NEW)

atlas-saga-orchestrator ──► atlas-drop-information (existing)
                        ──► atlas-quest (NEW)
```

### Environment Variables

| Service | Variable | Value |
|---------|----------|-------|
| atlas-monster-death | `QUESTS` | atlas-quest service URL |
| atlas-saga-orchestrator | `QUESTS` | atlas-quest service URL |

### Atlas Quest API

**Endpoint:** `GET /characters/{characterId}/quests/started`

**Response:**
```json
{
  "data": [
    {
      "type": "quest-status",
      "id": "123",
      "attributes": {
        "characterId": 12345,
        "questId": 100,
        "state": 1
      }
    }
  ]
}
```

## Testing Considerations

### Unit Test Cases

1. Drop with `questId=0` always included
2. Drop with `questId=X` included when quest X is started
3. Drop with `questId=X` excluded when quest X is not started
4. Drop with `questId=X` excluded when quest X is completed
5. All quest drops excluded when quest service returns error
6. No HTTP call when no drops have questId != 0

### Integration Test Scenarios

1. Kill monster with quest drop when quest started
2. Kill monster with quest drop when quest not started
3. Kill monster with quest drop when quest completed
4. Kill monster with mix of quest and non-quest drops
5. Reactor activation with quest items
6. Service resilience when quest service unavailable

## Related Documentation

- Quest Service REST API: `services/atlas-quest/docs/rest.md`
- Drop Information Service: `services/atlas-drop-information/README.md`
