# Quest-Aware Drop Systems - Task Checklist

**Last Updated:** 2026-02-03

## Phase 1: Quest Integration Module (atlas-monster-death)

**Effort:** S (Small)

### Tasks

- [ ] **1.1** Create `quest/` package directory
  - Path: `services/atlas-monster-death/atlas.com/monster/quest/`
  - Acceptance: Directory created

- [ ] **1.2** Create quest state model
  - File: `quest/model.go`
  - Content: State type, constants (StateNotStarted=0, StateStarted=1, StateCompleted=2), Model struct
  - Acceptance: Model compiles, has QuestId() getter

- [ ] **1.3** Create quest REST model
  - File: `quest/rest.go`
  - Content: RestModel struct matching atlas-quest response, Extract function
  - Acceptance: Can deserialize quest-status JSON

- [ ] **1.4** Create quest requests
  - File: `quest/requests.go`
  - Content: `requestStartedQuests(characterId)` using QUESTS env var
  - Acceptance: Builds correct URL `/characters/{id}/quests/started`

- [ ] **1.5** Create quest provider
  - File: `quest/provider.go`
  - Content: `GetStartedQuestIds()` returning `map[uint32]bool`
  - Acceptance: Returns set of started quest IDs for character

---

## Phase 2: Monster Drop Quest Filtering

**Effort:** M (Medium)

### Tasks

- [ ] **2.1** Add quest package import to processor
  - File: `services/atlas-monster-death/atlas.com/monster/monster/processor.go`
  - Content: Import `atlas-monster-death/quest`
  - Acceptance: Compiles without errors

- [ ] **2.2** Add QuestId getter to drop model (if missing)
  - File: `services/atlas-monster-death/atlas.com/monster/monster/drop/model.go`
  - Check: Verify `QuestId()` getter exists
  - Note: Already exists per analysis

- [ ] **2.3** Implement `filterByQuestState()` function
  - File: `services/atlas-monster-death/atlas.com/monster/monster/processor.go`
  - Content: Filter function that excludes quest drops when quest not started
  - Acceptance: Returns filtered slice, logs decisions

- [ ] **2.4** Modify `CreateDrops()` to apply quest filter
  - File: `services/atlas-monster-death/atlas.com/monster/monster/processor.go`
  - Content: Call `filterByQuestState()` after fetching drops, before chance evaluation
  - Acceptance: Quest drops only included for started quests

- [ ] **2.5** Add QUESTS environment variable to deployment config
  - Files: Docker/K8s configs for atlas-monster-death
  - Content: `QUESTS=http://atlas-quest:8080`
  - Acceptance: Service can reach quest service

- [ ] **2.6** Write unit tests for quest filtering
  - File: `services/atlas-monster-death/atlas.com/monster/monster/processor_test.go`
  - Content: Test cases for filtering logic
  - Acceptance: All test cases pass

---

## Phase 3: Quest Integration Module (atlas-saga-orchestrator)

**Effort:** S (Small)

### Tasks

- [ ] **3.1** Create `quest/state/` package directory
  - Path: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/quest/state/`
  - Note: Using `state/` subpackage to avoid conflict with existing `quest/` package
  - Acceptance: Directory created

- [ ] **3.2** Create quest state model
  - File: `quest/state/model.go`
  - Content: State type, constants, Model struct (similar to Phase 1.2)
  - Acceptance: Model compiles

- [ ] **3.3** Create quest REST model
  - File: `quest/state/rest.go`
  - Content: RestModel struct, Extract function
  - Acceptance: Can deserialize quest-status JSON

- [ ] **3.4** Create quest requests
  - File: `quest/state/requests.go`
  - Content: `requestStartedQuests(characterId)` using QUESTS env var
  - Acceptance: Builds correct URL

- [ ] **3.5** Create quest provider
  - File: `quest/state/provider.go`
  - Content: `GetStartedQuestIds()` returning `map[uint32]bool`
  - Acceptance: Returns set of started quest IDs

---

## Phase 4: Reactor Drop Quest Filtering

**Effort:** M (Medium)

### Tasks

- [ ] **4.1** Add quest state package import to reactor processor
  - File: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/processor.go`
  - Content: Import `atlas-saga-orchestrator/quest/state`
  - Acceptance: Compiles without errors

- [ ] **4.2** Implement `filterByQuestState()` method on ProcessorImpl
  - File: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/processor.go`
  - Content: Method to filter reactor drops by quest state
  - Acceptance: Returns filtered slice

- [ ] **4.3** Modify `SpawnReactorDrops()` to apply quest filter
  - File: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/processor.go`
  - Content: Call `filterByQuestState()` after fetching drops, before rolling chances
  - Acceptance: Quest drops only included for started quests

- [ ] **4.4** Add QUESTS environment variable to deployment config
  - Files: Docker/K8s configs for atlas-saga-orchestrator
  - Content: `QUESTS=http://atlas-quest:8080`
  - Acceptance: Service can reach quest service

- [ ] **4.5** Write unit tests for reactor quest filtering
  - File: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/processor_test.go`
  - Content: Test cases for filtering logic
  - Acceptance: All test cases pass

---

## Phase 5: Testing and Validation

**Effort:** M (Medium)

### Tasks

- [ ] **5.1** Create integration test for monster quest drops
  - Test: Kill monster with quest drop, verify it only drops when quest started
  - Acceptance: Drop appears when quest started, doesn't appear otherwise

- [ ] **5.2** Create integration test for reactor quest drops
  - Test: Activate reactor with quest drop, verify it only drops when quest started
  - Acceptance: Drop appears when quest started, doesn't appear otherwise

- [ ] **5.3** Performance test with multiple quest drops
  - Test: Kill monster with 10+ quest-linked drops
  - Acceptance: Additional latency < 50ms

- [ ] **5.4** Test error resilience
  - Test: Kill monster when quest service unavailable
  - Acceptance: Non-quest drops still appear, quest drops excluded

- [ ] **5.5** Manual QA validation
  - Test: Play through quest scenario in dev environment
  - Acceptance: Quest items drop correctly throughout quest lifecycle

- [ ] **5.6** Update documentation
  - Files: Service READMEs, architecture docs
  - Content: Document quest-aware drop filtering
  - Acceptance: New behavior documented

---

## Summary

| Phase | Tasks | Effort | Dependencies |
|-------|-------|--------|--------------|
| 1 | 5 | S | None |
| 2 | 6 | M | Phase 1 |
| 3 | 5 | S | None |
| 4 | 5 | M | Phase 3 |
| 5 | 6 | M | Phases 2, 4 |

**Total Tasks:** 27

**Suggested Implementation Order:**
1. Phase 1 + Phase 2 (monster drops - can be deployed independently)
2. Phase 3 + Phase 4 (reactor drops - can be deployed independently)
3. Phase 5 (testing - after both implementations complete)

## Notes

- Phases 1+2 and Phases 3+4 can be developed in parallel by different team members
- Each phase pair (1+2 or 3+4) can be deployed independently
- Phase 5 validates the complete implementation
