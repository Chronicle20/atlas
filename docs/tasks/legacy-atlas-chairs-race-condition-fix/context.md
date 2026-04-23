# Atlas Registry Race Condition Fix - Context Document

**Last Updated:** 2026-01-13

---

## 1. Key Files

### Files to Modify
| File | Purpose | Lines to Change |
|------|---------|-----------------|
| `services/atlas-chairs/atlas.com/chairs/character/registry.go` | Character registry with race condition | 44-83 |
| `services/atlas-chairs/atlas.com/chairs/character/registry_test.go` | Restore concurrent tests | ~232-239 |
| `services/atlas-maps/atlas.com/maps/map/character/registry.go` | Character registry with race condition | 44-97 |
| `services/atlas-chalkboards/atlas.com/chalkboards/character/registry.go` | Character registry with race condition | ~44-83 |

### Reference Files (Correct Implementation)
| File | Purpose |
|------|---------|
| `services/atlas-monsters/atlas.com/monsters/monster/registry.go` | Correct getMapLock pattern (lines 38-49) |

### Test Files
| File | Purpose |
|------|---------|
| `services/atlas-chairs/atlas.com/chairs/character/processor_test.go` | Processor tests (has resetProcessorRegistry) |
| `services/atlas-chairs/atlas.com/chairs/character/registry_test.go` | Registry tests (needs concurrent tests restored) |

---

## 2. Key Decisions

### Decision 1: Fix Approach
**Decision:** Apply the atlas-monsters pattern (global mutex in getMapLock)
**Rationale:**
- Proven pattern already exists in codebase
- Simple and verifiable
- Consistent with existing code
- Minimal risk of introducing new bugs

**Alternatives Considered:**
- sync.Map: More complex, requires API changes
- RWMutex for global lock: Overkill for short critical sections
- Channel-based locking: Overcomplicated for this use case

### Decision 2: Atomic Map Initialization
**Decision:** Initialize both `mapLocks` and `characterRegister` in getMapLock()
**Rationale:**
- Ensures both maps always have consistent state
- Simplifies AddCharacter() logic
- Matches atlas-monsters pattern

---

## 3. Race Condition Details

### Race #1: Unprotected Map Read
**Location:** `getMapLock()` line 59
**Problem:** `r.mapLocks[key]` read without mutex
**Concurrent Scenario:**
1. Goroutine A: reads `r.mapLocks[key]` (returns false)
2. Goroutine B: writes `r.mapLocks[key] = &sync.RWMutex{}`
3. Goroutine A: writes `r.mapLocks[key] = &sync.RWMutex{}` (creates duplicate)
4. Result: Memory corruption, possible panic

### Race #2: Unprotected characterRegister Access
**Location:** `AddCharacter()`, `RemoveCharacter()`, `GetInMap()`
**Problem:** `r.characterRegister[key]` accessed with only per-key lock
**Concurrent Scenario:**
1. Goroutine A (key1): holds `ml1.Lock()`, writes `r.characterRegister[key1]`
2. Goroutine B (key2): holds `ml2.Lock()`, writes `r.characterRegister[key2]`
3. Result: Concurrent writes to same underlying map = data race

### Race #3: Unprotected Map Iteration (atlas-maps only)
**Location:** `GetMapsWithCharacters()` line 87
**Problem:** Iterating over `mapLocks` without mutex
**Concurrent Scenario:**
1. Goroutine A: iterating `for mk, ml := range r.mapLocks`
2. Goroutine B: writing `r.mapLocks[newKey] = &sync.RWMutex{}`
3. Result: Undefined behavior, possible panic

---

## 4. Correct Pattern (from atlas-monsters)

```go
// File: services/atlas-monsters/atlas.com/monsters/monster/registry.go
// Lines: 38-49

func (r *Registry) getMapLock(key MapKey) *sync.RWMutex {
    r.mutex.Lock()         // 1. Always lock first
    defer r.mutex.Unlock() // 2. Clean unlock with defer

    if val, ok := r.mapLocks[key]; ok {  // 3. Check if exists
        return val                        // 4. Return existing
    }
    var cm = &sync.RWMutex{}              // 5. Create new
    r.mapLocks[key] = cm                  // 6. Store in map
    r.mapMonsterReg[key] = make([]MonsterKey, 0)  // 7. Initialize data map
    return cm                             // 8. Return new
}
```

**Key Properties:**
- Mutex acquired BEFORE any map access
- Both maps initialized atomically
- Uses defer for clean unlock
- Returns immediately if key exists

---

## 5. Fix Templates

### Template A: Fixed getMapLock()
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

### Template B: Simplified AddCharacter()
```go
func (r *Registry) AddCharacter(key MapKey, characterId uint32) {
    ml := r.getMapLock(key)
    ml.Lock()
    defer ml.Unlock()

    r.characterRegister[key] = appendIfMissing(r.characterRegister[key], characterId)
}
```

### Template C: Fixed GetMapsWithCharacters() (atlas-maps only)
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

---

## 6. Test Commands

### Run Tests Without Race Detector
```bash
cd <home>/source/pers/atlas/services/atlas-chairs/atlas.com/chairs
go test ./...
```

### Run Tests With Race Detector
```bash
cd <home>/source/pers/atlas/services/atlas-chairs/atlas.com/chairs
go test -race ./...
```

### Check Coverage
```bash
cd <home>/source/pers/atlas/services/atlas-chairs/atlas.com/chairs
go test -cover ./character/...
```

---

## 7. Concurrent Test Template

```go
func TestRegistry_Concurrent(t *testing.T) {
    resetCharacterRegistry()

    st := sampleTenant()
    key := sampleMapKey(st, 0, 1, 100000000)

    var wg sync.WaitGroup
    iterations := 100

    // Concurrent adds
    for i := 0; i < iterations; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            getRegistry().AddCharacter(key, uint32(id))
        }(i)
    }

    wg.Wait()

    chars := getRegistry().GetInMap(key)
    if len(chars) != iterations {
        t.Errorf("Expected %d characters, got %d", iterations, len(chars))
    }

    // Concurrent reads and removes
    for i := 0; i < iterations; i++ {
        wg.Add(2)
        go func() {
            defer wg.Done()
            getRegistry().GetInMap(key)
        }()
        go func(id int) {
            defer wg.Done()
            getRegistry().RemoveCharacter(key, uint32(id))
        }(i)
    }

    wg.Wait()

    chars = getRegistry().GetInMap(key)
    if len(chars) != 0 {
        t.Errorf("Expected 0 characters after all removes, got %d", len(chars))
    }
}

func TestRegistry_ConcurrentDifferentMaps(t *testing.T) {
    resetCharacterRegistry()

    st := sampleTenant()
    numMaps := 10
    charsPerMap := 50

    var wg sync.WaitGroup

    // Concurrent adds to different maps
    for m := 0; m < numMaps; m++ {
        key := sampleMapKey(st, 0, 1, _map.Id(100000000+m))
        for c := 0; c < charsPerMap; c++ {
            wg.Add(1)
            go func(k MapKey, charId uint32) {
                defer wg.Done()
                getRegistry().AddCharacter(k, charId)
            }(key, uint32(m*1000+c))
        }
    }

    wg.Wait()

    // Verify each map has correct number of characters
    for m := 0; m < numMaps; m++ {
        key := sampleMapKey(st, 0, 1, _map.Id(100000000+m))
        chars := getRegistry().GetInMap(key)
        if len(chars) != charsPerMap {
            t.Errorf("Map %d: expected %d characters, got %d", m, charsPerMap, len(chars))
        }
    }
}
```

---

## 8. Dependencies

### Go Standard Library
- `sync.Mutex` - Global registry lock
- `sync.RWMutex` - Per-key read/write lock
- `sync.Once` - Singleton initialization
- `sync.WaitGroup` - Test synchronization

### Atlas Libraries
- `github.com/Chronicle20/atlas-tenant` - Tenant model for MapKey
- `github.com/Chronicle20/atlas-constants/*` - World/Channel/Map types

---

## 9. Out of Scope

1. **Performance Optimization:** Not addressing any potential performance issues with the global mutex approach
2. **Other Services:** Only fixing the three identified services with this specific pattern
3. **chair/registry.go:** This file uses a simpler pattern with global RWMutex and is already thread-safe
4. **Architectural Redesign:** Not changing the overall registry architecture, just fixing the locking
