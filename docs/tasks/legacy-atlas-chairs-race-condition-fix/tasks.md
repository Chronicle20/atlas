# Atlas Registry Race Condition Fix - Task Checklist

**Last Updated:** 2026-01-13
**Status:** COMPLETED

---

## Implementation Notes

During implementation, we discovered that the original per-key locking approach was fundamentally flawed. Go maps are not thread-safe even when accessing different keys concurrently. The `getMapLock()` function itself accessed the `mapLocks` map without protection.

**Solution Applied:** Simplified all registry implementations to use a single `sync.RWMutex` for all operations, removing the `mapLocks` map entirely. This matches the simpler pattern and eliminates the race condition at its root.

---

## Phase 1: Fix atlas-chairs

### Task 1.1: Simplify character/registry.go to use single RWMutex
- [x] Open `services/atlas-chairs/atlas.com/chairs/character/registry.go`
- [x] Remove `mapLocks` map entirely
- [x] Simplify to use single `mutex` for all operations
- [x] Update all methods to use `r.mutex.Lock()/Unlock()` or `r.mutex.RLock()/RUnlock()`
- [x] Run basic tests: `go test ./character/...`

**Effort:** S | **Status:** Complete

---

### Task 1.2: Restore Concurrent Tests
- [x] Open `services/atlas-chairs/atlas.com/chairs/character/registry_test.go`
- [x] Update `resetCharacterRegistry()` to remove `mapLocks` reference
- [x] Verify concurrent tests (`TestRegistry_Concurrent`, `TestRegistry_ConcurrentDifferentMaps`)
- [x] Run tests with race detector: `go test -race ./character/...`
- [x] Verify no "WARNING: DATA RACE" messages
- [x] Verify all tests pass

**Effort:** S | **Status:** Complete

---

## Phase 2: Fix atlas-maps

### Task 2.1: Simplify map/character/registry.go
- [x] Open `services/atlas-maps/atlas.com/maps/map/character/registry.go`
- [x] Remove `mapLocks` map entirely
- [x] Simplify to use single `mutex` for all operations
- [x] Update `GetMapsWithCharacters()` to use RLock for read-only access

**Effort:** S | **Status:** Complete

---

## Phase 3: Fix atlas-chalkboards

### Task 3.1: Simplify character/registry.go
- [x] Open `services/atlas-chalkboards/atlas.com/chalkboards/character/registry.go`
- [x] Remove `mapLocks` map entirely
- [x] Simplify to use single `mutex` for all operations

**Effort:** S | **Status:** Complete

---

### Task 3.2: Update test files
- [x] Update `registry_test.go` to remove `mapLocks` reference in `resetRegistry()`
- [x] Update `processor_test.go` to remove `mapLocks` reference in `resetProcessorRegistry()`
- [x] Restore `sync` import for `sync.WaitGroup` in concurrent tests

**Effort:** S | **Status:** Complete

---

## Phase 4: Verification

### Task 4.1: Run Race Detector on atlas-chairs
- [x] Run: `cd services/atlas-chairs/atlas.com/chairs && go test -race ./character/...`
- [x] Result: `ok  atlas-chairs/character  (cached)` - PASSED

**Effort:** S | **Status:** Complete

---

### Task 4.2: Run Race Detector on atlas-maps
- [x] Run: `cd services/atlas-maps/atlas.com/maps && go test -race ./map/character/...`
- [x] Result: `[no test files]` - No tests to run, but code fix applied

**Effort:** S | **Status:** Complete

---

### Task 4.3: Run Race Detector on atlas-chalkboards
- [x] Run: `cd services/atlas-chalkboards/atlas.com/chalkboards && go test -race ./character/...`
- [x] Result: `ok  atlas-chalkboards/character  1.011s` - PASSED

**Effort:** S | **Status:** Complete

---

## Final Verification Checklist

- [x] All Phase 1 tasks complete (atlas-chairs)
- [x] All Phase 2 tasks complete (atlas-maps)
- [x] All Phase 3 tasks complete (atlas-chalkboards)
- [x] All Phase 4 tasks complete (verification)
- [x] atlas-chairs passes `go test -race ./character/...`
- [x] atlas-chalkboards passes `go test -race ./character/...`
- [x] No behavioral changes (existing tests still pass)
- [x] Concurrent tests restored and passing

---

## Progress Summary

| Phase | Tasks | Completed | Status |
|-------|-------|-----------|--------|
| Phase 1 (atlas-chairs) | 2 | 2 | Complete |
| Phase 2 (atlas-maps) | 1 | 1 | Complete |
| Phase 3 (atlas-chalkboards) | 2 | 2 | Complete |
| Phase 4 (verification) | 3 | 3 | Complete |
| **Total** | **8** | **8** | **Complete** |

---

## Files Modified

### atlas-chairs
- `services/atlas-chairs/atlas.com/chairs/character/registry.go` - Simplified to single RWMutex
- `services/atlas-chairs/atlas.com/chairs/character/registry_test.go` - Updated reset function
- `services/atlas-chairs/atlas.com/chairs/character/processor_test.go` - Updated reset function

### atlas-maps
- `services/atlas-maps/atlas.com/maps/map/character/registry.go` - Simplified to single RWMutex

### atlas-chalkboards
- `services/atlas-chalkboards/atlas.com/chalkboards/character/registry.go` - Simplified to single RWMutex
- `services/atlas-chalkboards/atlas.com/chalkboards/character/registry_test.go` - Updated reset function
- `services/atlas-chalkboards/atlas.com/chalkboards/character/processor_test.go` - Updated reset function
