# Atlas Registry Race Condition Fix Plan

**Affected Services:** `atlas-chairs`, `atlas-maps`, `atlas-chalkboards`
**Reference Pattern:** `atlas-monsters` (correctly implemented)
**Last Updated:** 2026-01-13

---

## 1. Executive Summary

Multiple services in the Atlas codebase have a race condition in their character registry implementations. The issue stems from a flawed per-key locking strategy where:

1. The `getMapLock()` function reads from `mapLocks` without holding the mutex (race on map read)
2. The `characterRegister` map is accessed while only holding a per-key lock, not protecting the shared map structure itself

This is a **thread safety bug** that could cause:
- Data corruption under high concurrency
- Panic due to concurrent map read/write
- Incorrect character tracking state

The fix involves applying the correct locking pattern already used in `atlas-monsters/monster/registry.go`.

**Scope:** 3 services with identical bug pattern
**Effort:** Small (S) - straightforward pattern application
**Risk:** Low - proven pattern exists in codebase

---

## 2. Current State Analysis

### Affected Files

| Service | File | Issue |
|---------|------|-------|
| atlas-chairs | `character/registry.go` | Lines 56-66: unsafe getMapLock, Lines 44-54,68-76,78-83: unsafe characterRegister access |
| atlas-maps | `map/character/registry.go` | Lines 56-66: unsafe getMapLock, Lines 44-54,68-76,78-83,85-97: unsafe characterRegister access |
| atlas-chalkboards | `character/registry.go` | Lines 56-66: unsafe getMapLock (likely identical pattern) |

### Root Cause Analysis

**Race Condition #1: Unsafe mapLocks access in getMapLock()**

```go
// BUGGY CODE (atlas-chairs/character/registry.go:56-66)
func (r *Registry) getMapLock(key MapKey) *sync.RWMutex {
    var ml *sync.RWMutex
    var ok bool
    if ml, ok = r.mapLocks[key]; !ok {  // RACE: unprotected map read
        r.mutex.Lock()
        r.mapLocks[key] = &sync.RWMutex{}
        ml = r.mapLocks[key]
        r.mutex.Unlock()
    }
    return ml
}
```

The `r.mapLocks[key]` read at line 59 is not protected by any mutex. A concurrent goroutine could be writing to `mapLocks` inside the mutex-protected block, causing a data race.

**Race Condition #2: Unsafe characterRegister access**

```go
// BUGGY CODE (atlas-chairs/character/registry.go:44-54)
func (r *Registry) AddCharacter(key MapKey, characterId uint32) {
    var ml = r.getMapLock(key)
    ml.Lock()
    defer ml.Unlock()

    if _, ok := r.characterRegister[key]; ok {  // RACE: only per-key lock held
        r.characterRegister[key] = appendIfMissing(r.characterRegister[key], characterId)
        return
    }
    r.characterRegister[key] = []uint32{characterId}  // RACE: map write with only per-key lock
}
```

The per-key lock `ml` only synchronizes operations on the same key. Two different keys can cause concurrent reads/writes to the underlying `characterRegister` map, which is unsafe.

### Correct Pattern (atlas-monsters)

```go
// CORRECT CODE (atlas-monsters/monster/registry.go:38-49)
func (r *Registry) getMapLock(key MapKey) *sync.RWMutex {
    r.mutex.Lock()         // Always lock first
    defer r.mutex.Unlock()

    if val, ok := r.mapLocks[key]; ok {
        return val
    }
    var cm = &sync.RWMutex{}
    r.mapLocks[key] = cm
    r.mapMonsterReg[key] = make([]MonsterKey, 0)  // Initialize both maps atomically
    return cm
}
```

Key differences:
1. Mutex is acquired **before** any map access
2. Both maps are initialized atomically under the same lock
3. Uses `defer` for clean unlock

---

## 3. Proposed Future State

After the fix, all character registries will:

1. **Safe getMapLock()**: Always acquire mutex before checking/modifying mapLocks
2. **Safe map access**: All shared map operations protected by appropriate locks
3. **Atomic initialization**: Both `mapLocks` and `characterRegister` entries created under single lock
4. **Pass race detector**: `go test -race ./...` will pass without warnings

### Solution Options

**Option A: Global Mutex for All Operations (Recommended)**

Apply the atlas-monsters pattern: acquire global mutex in `getMapLock()`, which ensures safe access to both maps.

Pros:
- Simple, proven pattern
- Already exists in codebase
- Easy to verify correctness

Cons:
- Slightly reduced concurrency (global lock contention)
- Acceptable for in-memory registries with low contention

**Option B: Use sync.Map**

Replace both `mapLocks` and `characterRegister` with `sync.Map`.

Pros:
- No explicit locking needed
- Better concurrent performance

Cons:
- API changes required
- Type assertions needed (less type-safe)
- More complex code

**Recommendation:** Option A - it's proven, simple, and consistent with existing patterns.

---

## 4. Implementation Phases

### Phase 1: Fix atlas-chairs (Primary Target)
Fix the race condition in atlas-chairs and verify with tests.

### Phase 2: Fix atlas-maps
Apply the same fix pattern to atlas-maps.

### Phase 3: Fix atlas-chalkboards
Apply the same fix pattern to atlas-chalkboards.

### Phase 4: Verification
Run race detector across all affected services to confirm fixes.

---

## 5. Detailed Tasks

### Phase 1: Fix atlas-chairs

#### Task 1.1: Fix getMapLock() in character/registry.go
**Effort:** S (Small)
**Priority:** P0
**Dependencies:** None

**Current Code (BUGGY):**
```go
func (r *Registry) getMapLock(key MapKey) *sync.RWMutex {
    var ml *sync.RWMutex
    var ok bool
    if ml, ok = r.mapLocks[key]; !ok {
        r.mutex.Lock()
        r.mapLocks[key] = &sync.RWMutex{}
        ml = r.mapLocks[key]
        r.mutex.Unlock()
    }
    return ml
}
```

**Fixed Code:**
```go
func (r *Registry) getMapLock(key MapKey) *sync.RWMutex {
    r.mutex.Lock()
    defer r.mutex.Unlock()

    if ml, ok := r.mapLocks[key]; ok {
        return ml
    }
    ml := &sync.RWMutex{}
    r.mapLocks[key] = ml
    r.characterRegister[key] = make([]uint32, 0)
    return ml
}
```

**Acceptance Criteria:**
- [ ] getMapLock() acquires mutex before any map access
- [ ] Both mapLocks and characterRegister initialized atomically
- [ ] Uses defer for clean unlock

---

#### Task 1.2: Simplify AddCharacter()
**Effort:** S (Small)
**Priority:** P0
**Dependencies:** Task 1.1

Since getMapLock() now initializes `characterRegister[key]`, AddCharacter can be simplified:

**Current Code:**
```go
func (r *Registry) AddCharacter(key MapKey, characterId uint32) {
    var ml = r.getMapLock(key)
    ml.Lock()
    defer ml.Unlock()

    if _, ok := r.characterRegister[key]; ok {
        r.characterRegister[key] = appendIfMissing(r.characterRegister[key], characterId)
        return
    }
    r.characterRegister[key] = []uint32{characterId}
}
```

**Fixed Code:**
```go
func (r *Registry) AddCharacter(key MapKey, characterId uint32) {
    ml := r.getMapLock(key)
    ml.Lock()
    defer ml.Unlock()

    r.characterRegister[key] = appendIfMissing(r.characterRegister[key], characterId)
}
```

**Acceptance Criteria:**
- [ ] AddCharacter simplified to single append operation
- [ ] No redundant map existence check
- [ ] Per-key lock still used for data operations

---

#### Task 1.3: Restore Concurrent Tests
**Effort:** S (Small)
**Priority:** P1
**Dependencies:** Tasks 1.1, 1.2

Restore the concurrent tests that were removed due to race conditions.

**Acceptance Criteria:**
- [ ] TestRegistry_Concurrent restored in character/registry_test.go
- [ ] TestRegistry_ConcurrentDifferentMaps restored
- [ ] Tests pass with race detector: `go test -race ./character/...`

---

### Phase 2: Fix atlas-maps

#### Task 2.1: Fix getMapLock() in map/character/registry.go
**Effort:** S (Small)
**Priority:** P0
**Dependencies:** Phase 1 complete

Apply identical fix pattern from Task 1.1.

**Acceptance Criteria:**
- [ ] getMapLock() fixed with full mutex protection
- [ ] Atomic initialization of both maps

---

#### Task 2.2: Simplify AddCharacter() in atlas-maps
**Effort:** S (Small)
**Priority:** P0
**Dependencies:** Task 2.1

Apply identical simplification from Task 1.2.

**Acceptance Criteria:**
- [ ] AddCharacter simplified
- [ ] All tests pass

---

#### Task 2.3: Fix GetMapsWithCharacters() race
**Effort:** S (Small)
**Priority:** P0
**Dependencies:** Task 2.1

The `GetMapsWithCharacters()` function at line 85-97 iterates over `mapLocks` without holding the global mutex.

**Current Code (BUGGY):**
```go
func (r *Registry) GetMapsWithCharacters() []MapKey {
    var result = make([]MapKey, 0)
    for mk, ml := range r.mapLocks {  // RACE: unprotected iteration
        ml.RLock()
        if mc, ok := r.characterRegister[mk]; ok {
            if len(mc) > 0 {
                result = append(result, mk)
            }
        }
        ml.RUnlock()
    }
    return result
}
```

**Fixed Code:**
```go
func (r *Registry) GetMapsWithCharacters() []MapKey {
    r.mutex.Lock()
    keys := make([]MapKey, 0, len(r.mapLocks))
    for mk := range r.mapLocks {
        keys = append(keys, mk)
    }
    r.mutex.Unlock()

    result := make([]MapKey, 0)
    for _, mk := range keys {
        ml := r.getMapLock(mk)
        ml.RLock()
        if mc := r.characterRegister[mk]; len(mc) > 0 {
            result = append(result, mk)
        }
        ml.RUnlock()
    }
    return result
}
```

**Acceptance Criteria:**
- [ ] Map iteration done while holding mutex
- [ ] Per-key operations done with per-key lock
- [ ] No race conditions in iteration

---

### Phase 3: Fix atlas-chalkboards

#### Task 3.1: Fix getMapLock() in character/registry.go
**Effort:** S (Small)
**Priority:** P0
**Dependencies:** Phase 2 complete

Apply identical fix pattern.

**Acceptance Criteria:**
- [ ] getMapLock() fixed
- [ ] All tests pass

---

#### Task 3.2: Simplify AddCharacter()
**Effort:** S (Small)
**Priority:** P0
**Dependencies:** Task 3.1

Apply identical simplification.

**Acceptance Criteria:**
- [ ] AddCharacter simplified
- [ ] Tests pass with race detector

---

### Phase 4: Verification

#### Task 4.1: Run Race Detector on All Affected Services
**Effort:** S (Small)
**Priority:** P0
**Dependencies:** Phases 1-3 complete

**Commands:**
```bash
cd services/atlas-chairs/atlas.com/chairs && go test -race ./...
cd services/atlas-maps/atlas.com/maps && go test -race ./...
cd services/atlas-chalkboards/atlas.com/chalkboards && go test -race ./...
```

**Acceptance Criteria:**
- [ ] All three services pass race detector
- [ ] No "WARNING: DATA RACE" messages
- [ ] All tests pass

---

## 6. Risk Assessment and Mitigation

### Risk 1: Behavioral Changes
**Likelihood:** Low
**Impact:** Medium
**Mitigation:**
- The fix maintains identical semantics
- Only changes when locks are acquired
- Existing tests verify behavior is preserved

### Risk 2: Performance Regression
**Likelihood:** Low
**Impact:** Low
**Mitigation:**
- Global mutex is held briefly (just for map lookup/insert)
- Per-key locks still used for data operations
- In-memory registries have low contention
- Can measure with benchmarks if concerned

### Risk 3: Missing Similar Bugs
**Likelihood:** Medium
**Impact:** Medium
**Mitigation:**
- Grep searched for `mapLocks.*map[.*]*sync.RWMutex` pattern
- Three services identified with this pattern
- atlas-monsters already correct (reference implementation)

---

## 7. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Race Conditions | 0 | `go test -race ./...` passes |
| Test Coverage | Maintained | `go test -cover` same or better |
| Concurrent Tests | Restored | Previously removed tests pass |
| Build Status | Passing | All tests pass |

---

## 8. Required Resources and Dependencies

### Technical Dependencies
- Go race detector (`go test -race`)
- Access to all three affected services
- Reference implementation in atlas-monsters

### Files to Modify
| Service | File | Lines |
|---------|------|-------|
| atlas-chairs | `character/registry.go` | 44-83 |
| atlas-chairs | `character/registry_test.go` | Restore concurrent tests |
| atlas-maps | `map/character/registry.go` | 44-97 |
| atlas-chalkboards | `character/registry.go` | ~44-83 |

---

## 9. Notes and Considerations

1. **Why Per-Key Locking?** The original design likely intended to reduce lock contention by using per-map locks. However, the implementation was flawed because the maps storing those locks also need protection.

2. **atlas-monsters as Reference:** The monsters service has the correct implementation. It was likely fixed previously or written by a different developer.

3. **Test Coverage:** The atlas-chairs service already has tests that were partially disabled due to race conditions. These should be restored after the fix.

4. **No Functional Changes:** This fix is purely about thread safety. The observable behavior of the registries should be identical.
