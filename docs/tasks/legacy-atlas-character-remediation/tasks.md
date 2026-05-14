# Atlas-Character Remediation - Task Checklist

**Last Updated:** 2026-01-13
**Status:** Complete

---

## Phase 1: Blocking Issues (P0)

### 1.1 Update Transform Function
**File:** `services/atlas-character/atlas.com/character/character/rest.go`
**Effort:** S

- [x] Replace `m.id` with `m.Id()`
- [x] Replace `m.accountId` with `m.AccountId()`
- [x] Replace `m.worldId` with `m.WorldId()`
- [x] Replace `m.name` with `m.Name()`
- [x] Replace `m.level` with `m.Level()`
- [x] Replace `m.experience` with `m.Experience()`
- [x] Replace `m.gachaponExperience` with `m.GachaponExperience()`
- [x] Replace `m.strength` with `m.Strength()`
- [x] Replace `m.dexterity` with `m.Dexterity()`
- [x] Replace `m.intelligence` with `m.Intelligence()`
- [x] Replace `m.luck` with `m.Luck()`
- [x] Replace `m.hp` with `m.HP()`
- [x] Replace `m.maxHp` with `m.MaxHP()`
- [x] Replace `m.mp` with `m.MP()`
- [x] Replace `m.maxMp` with `m.MaxMP()`
- [x] Replace `m.meso` with `m.Meso()`
- [x] Replace `m.hpMpUsed` with `m.HPMPUsed()`
- [x] Replace `m.jobId` with `m.JobId()`
- [x] Replace `m.skinColor` with `m.SkinColor()`
- [x] Replace `m.gender` with `m.Gender()`
- [x] Replace `m.fame` with `m.Fame()`
- [x] Replace `m.hair` with `m.Hair()`
- [x] Replace `m.face` with `m.Face()`
- [x] Replace `m.ap` with `m.AP()`
- [x] Replace `m.sp` with `m.SPString()`
- [x] Replace `m.mapId` with `m.MapId()`
- [x] Replace `m.spawnPoint` with `m.SpawnPoint()`
- [x] Replace `m.gm` with `m.GM()`

### 1.2 Update Extract Function
**File:** `services/atlas-character/atlas.com/character/character/rest.go`
**Effort:** S

- [x] Replace direct Model{} instantiation with NewModelBuilder()
- [x] Chain all Set* methods for each field
- [x] End chain with .Build()
- [x] Verify return signature unchanged

### 1.3 Verify Tests Pass
**Effort:** S

- [x] Run `go test ./atlas.com/character/character/...`
- [x] Verify all tests pass (60/60 passed)
- [x] Check for any test failures related to rest.go (none)

### 1.4 Update Tests If Needed
**File:** `services/atlas-character/atlas.com/character/character/rest_test.go`
**Effort:** S

- [x] Review test failures (if any) - No failures
- [x] Update test assertions if needed - Not needed
- [x] Ensure test coverage maintained - Confirmed

---

## Phase 2: Documentation (P2 - Optional)

### 2.1 Document Builder Variants
**Files:** `builder.go`, `model.go`
**Effort:** S

- [ ] Add doc comment to Builder struct in builder.go
- [ ] Add doc comment to modelBuilder struct in model.go
- [ ] Explain when to use each builder

---

## Verification Checklist

### Pre-Implementation
- [x] Read current rest.go implementation
- [x] Verify all accessor methods exist in model.go
- [x] Verify all builder Set* methods exist in model.go

### Post-Implementation
- [x] All tests pass (60/60)
- [x] No new linting errors
- [x] Code compiles successfully
- [x] Transform uses only accessor methods (no private field access)
- [x] Extract uses only builder pattern (no direct struct instantiation)

---

## Commands Reference

```bash
# Navigate to service
cd <repo-root>/services/atlas-character

# Run tests
go test ./atlas.com/character/character/...

# Run tests verbose
go test -v ./atlas.com/character/character/...

# Build check
cd atlas.com/character && GOWORK=off go build ./...

# Lint check (if available)
golangci-lint run ./atlas.com/character/character/...
```

---

## Notes

- Temporal data fields (X, Y, Stance) already use accessor pattern via `td.X()`, `td.Y()`, `td.Stance()` - no changes needed
- The skill subdomain in this service also has an Extract function that accesses private fields, but fixing that is out of scope for this remediation (would require adding a builder to skill/model.go)
- Phase 2 documentation is optional and can be done in a future task
